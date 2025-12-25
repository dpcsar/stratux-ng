package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/decoder"
	"stratux-ng/internal/fancontrol"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/gps"
	"stratux-ng/internal/traffic"
	"stratux-ng/internal/udp"
	"stratux-ng/internal/web"
)

type liveRuntime struct {
	resolvedConfigPath string
	status             *web.Status
	sender             *safeBroadcaster
	ahrsSvc            *ahrs.Service
	gpsSvc             *gps.Service
	fanSvc             *fancontrol.Service

	adsb1090Sup   *decoder.Supervisor
	uat978Sup     *decoder.Supervisor
	adsb1090File  *decoder.JSONFilePoller
	uat978Stream  *decoder.NDJSONClient
	uat978Raw     *decoder.LineClient
	uat978UplinkQ chan []byte

	trafficStore *traffic.Store

	cfg      config.Config
	scenario scenarioRuntime
	ticker   *time.Ticker
}

func decoderBandEqual(a, b config.DecoderBandConfig) bool {
	if a.Enable != b.Enable {
		return false
	}
	if strings.TrimSpace(a.Decoder.Command) != strings.TrimSpace(b.Decoder.Command) {
		return false
	}
	if strings.TrimSpace(a.Decoder.JSONListen) != strings.TrimSpace(b.Decoder.JSONListen) {
		return false
	}
	if strings.TrimSpace(a.Decoder.JSONAddr) != strings.TrimSpace(b.Decoder.JSONAddr) {
		return false
	}
	if strings.TrimSpace(a.Decoder.JSONFile) != strings.TrimSpace(b.Decoder.JSONFile) {
		return false
	}
	if strings.TrimSpace(a.Decoder.RawListen) != strings.TrimSpace(b.Decoder.RawListen) {
		return false
	}
	if strings.TrimSpace(a.Decoder.RawAddr) != strings.TrimSpace(b.Decoder.RawAddr) {
		return false
	}
	if a.Decoder.JSONFileInterval != b.Decoder.JSONFileInterval {
		return false
	}
	if strings.TrimSpace(a.SDR.SerialTag) != strings.TrimSpace(b.SDR.SerialTag) {
		return false
	}
	if strings.TrimSpace(a.SDR.Path) != strings.TrimSpace(b.SDR.Path) {
		return false
	}
	if (a.SDR.Index == nil) != (b.SDR.Index == nil) {
		return false
	}
	if a.SDR.Index != nil && b.SDR.Index != nil && *a.SDR.Index != *b.SDR.Index {
		return false
	}
	if len(a.Decoder.Args) != len(b.Decoder.Args) {
		return false
	}
	for i := range a.Decoder.Args {
		if a.Decoder.Args[i] != b.Decoder.Args[i] {
			return false
		}
	}
	return true
}

func newLiveRuntime(ctx context.Context, cfg config.Config, resolvedConfigPath string, status *web.Status, sender *safeBroadcaster) (*liveRuntime, error) {
	c := cfg
	if err := config.DefaultAndValidate(&c); err != nil {
		return nil, err
	}
	if status == nil {
		return nil, fmt.Errorf("status is nil")
	}
	if sender == nil {
		return nil, fmt.Errorf("sender is nil")
	}

	sc, err := loadScenarioFromConfig(c)
	if err != nil {
		return nil, err
	}

	var t *time.Ticker
	if !c.GDL90.Replay.Enable {
		t = time.NewTicker(c.GDL90.Interval)
	}

	// Optional: real AHRS bring-up.
	var ahrsSvc *ahrs.Service
	if c.AHRS.Enable {
		var gravity [3]float64
		var gravitySet bool
		if g := c.AHRS.Orientation.GravityInSensor; len(g) == 3 {
			gravity = [3]float64{g[0], g[1], g[2]}
			gravitySet = true
		}
		svc := ahrs.New(ahrs.Config{
			Enable:   c.AHRS.Enable,
			I2CBus:   c.AHRS.I2CBus,
			IMUAddr:  c.AHRS.IMUAddr,
			BaroAddr: c.AHRS.BaroAddr,

			OrientationForwardAxis: c.AHRS.Orientation.ForwardAxis,
			OrientationGravitySet:  gravitySet,
			OrientationGravity:     gravity,
		})
		if err := svc.Start(ctx); err != nil {
			// Keep Stratux-NG running even if AHRS fails to init; AHRS will be marked invalid.
			log.Printf("ahrs init failed: %v", err)
		}
		ahrsSvc = svc
	}

	r := &liveRuntime{
		resolvedConfigPath: resolvedConfigPath,
		status:             status,
		sender:             sender,
		cfg:                c,
		scenario:           sc,
		ticker:             t,
		ahrsSvc:            ahrsSvc,
		uat978UplinkQ:      make(chan []byte, 512),
		trafficStore:       traffic.NewStore(traffic.StoreConfig{MaxTargets: 200, TTL: 30 * time.Second}),
	}

	// Optional: external decoders (1090/dump1090-fa, 978/dump978-fa).
	// Start supervised processes (if configured) and attach NDJSON clients.
	if err := r.initDecoders(ctx); err != nil {
		r.Close()
		return nil, err
	}

	// Optional: real GPS bring-up (USB serial NMEA).
	if c.GPS.Enable {
		svc := gps.New(gps.Config{
			Enable:   c.GPS.Enable,
			Source:   c.GPS.Source,
			GPSDAddr: c.GPS.GPSDAddr,
			Device:   c.GPS.Device,
			Baud:     c.GPS.Baud,
		})
		if err := svc.Start(ctx); err != nil {
			// Keep Stratux-NG running even if GPS fails to init.
			log.Printf("gps init failed: %v", err)
		}
		r.gpsSvc = svc
	}

	// Optional: fan control.
	if c.Fan.Enable {
		svc := fancontrol.New(fancontrol.Config{
			Enable:         c.Fan.Enable,
			Backend:        c.Fan.Backend,
			PWMPin:         c.Fan.PWMPin,
			PWMFrequency:   c.Fan.PWMFrequency,
			TempTargetC:    c.Fan.TempTargetC,
			PWMDutyMin:     c.Fan.PWMDutyMin,
			UpdateInterval: c.Fan.UpdateInterval,
		})
		// Keep a reference even if init fails so status can report errors.
		r.fanSvc = svc
		if err := svc.Start(ctx); err != nil {
			// Keep Stratux-NG running even if fan control fails to init.
			log.Printf("fancontrol init failed: %v", err)
		}
	}

	return r, nil
}

func (r *liveRuntime) initDecoders(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("runtime is nil")
	}
	// 1090
	if r.cfg.ADSB1090.Enable {
		band := r.cfg.ADSB1090
		jsonFile := strings.TrimSpace(band.Decoder.JSONFile)
		if jsonFile == "" {
			return fmt.Errorf("adsb1090.decoder.json_file is required")
		}
		log.Printf("adsb1090 enabled json_file=%s interval=%s", jsonFile, band.Decoder.JSONFileInterval)
		if cmd := strings.TrimSpace(band.Decoder.Command); cmd != "" {
			log.Printf("adsb1090 supervising decoder cmd=%s args=%q", cmd, band.Decoder.Args)
			// If we're supervising dump1090-fa and asking it to write under jsonFile,
			// ensure the directory exists (common path: /run/dump1090-fa).
			if dir := filepath.Dir(jsonFile); dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("adsb1090.decoder.json_file directory: %w", err)
				}
			}
			sup, err := decoder.NewSupervisor(decoder.SupervisorConfig{
				Name:    "adsb1090",
				Command: cmd,
				Args:    band.Decoder.Args,
				Restart: true,
			})
			if err != nil {
				return fmt.Errorf("adsb1090 supervisor: %w", err)
			}
			if err := sup.Start(ctx); err != nil {
				return fmt.Errorf("adsb1090 supervisor start: %w", err)
			}
			r.adsb1090Sup = sup
			go func() {
				// Give the child process a moment to start and emit errors.
				time.Sleep(2 * time.Second)
				snap := sup.Snapshot()
				if snap.Running {
					log.Printf("adsb1090 decoder running pid=%d", snap.PID)
					return
				}
				log.Printf("adsb1090 decoder not running state=%s last_error=%s", snap.State, snap.LastError)
				if len(snap.Stderr) > 0 {
					log.Printf("adsb1090 decoder stderr tail:\n%s", strings.Join(snap.Stderr, "\n"))
				}
			}()
		}
		poller, err := decoder.NewJSONFilePoller(decoder.JSONFilePollerConfig{
			Name:     "adsb1090",
			Path:     jsonFile,
			Interval: band.Decoder.JSONFileInterval,
		})
		if err != nil {
			return fmt.Errorf("adsb1090 jsonfile: %w", err)
		}
		if err := poller.Start(ctx, func(raw json.RawMessage) error {
			// Keep the poller healthy: never return errors for parse issues.
			targets := traffic.ParseDump1090FAAircraftJSON(raw)
			if len(targets) > 0 && r.trafficStore != nil {
				r.trafficStore.UpsertMany(time.Now().UTC(), targets)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("adsb1090 jsonfile start: %w", err)
		}
		r.adsb1090File = poller
		go func() {
			time.Sleep(2 * time.Second)
			snap := poller.Snapshot(time.Now().UTC())
			if snap.State == "error" {
				log.Printf("adsb1090 jsonfile poller error path=%s err=%s", snap.Path, snap.LastError)
			}
		}()
	}

	// 978
	if r.cfg.UAT978.Enable {
		band := r.cfg.UAT978
		endpoint := strings.TrimSpace(band.Decoder.JSONAddr)
		if endpoint == "" {
			endpoint = strings.TrimSpace(band.Decoder.JSONListen)
		}
		rawEndpoint := strings.TrimSpace(band.Decoder.RawAddr)
		if rawEndpoint == "" {
			rawEndpoint = strings.TrimSpace(band.Decoder.RawListen)
		}
		log.Printf("uat978 enabled json_endpoint=%s raw_endpoint=%s", endpoint, rawEndpoint)
		if cmd := strings.TrimSpace(band.Decoder.Command); cmd != "" {
			log.Printf("uat978 supervising decoder cmd=%s args=%q", cmd, band.Decoder.Args)
			sup, err := decoder.NewSupervisor(decoder.SupervisorConfig{
				Name:    "uat978",
				Command: cmd,
				Args:    band.Decoder.Args,
				Restart: true,
			})
			if err != nil {
				return fmt.Errorf("uat978 supervisor: %w", err)
			}
			if err := sup.Start(ctx); err != nil {
				return fmt.Errorf("uat978 supervisor start: %w", err)
			}
			r.uat978Sup = sup
			go func() {
				time.Sleep(2 * time.Second)
				snap := sup.Snapshot()
				if snap.Running {
					log.Printf("uat978 decoder running pid=%d", snap.PID)
					return
				}
				log.Printf("uat978 decoder not running state=%s last_error=%s", snap.State, snap.LastError)
				if len(snap.Stderr) > 0 {
					log.Printf("uat978 decoder stderr tail:\n%s", strings.Join(snap.Stderr, "\n"))
				}
			}()
		}
		if endpoint != "" {
			client, err := decoder.NewNDJSONClient(decoder.NDJSONClientConfig{
				Name: "uat978",
				Addr: endpoint,
			})
			if err != nil {
				return fmt.Errorf("uat978 ndjson: %w", err)
			}
			if err := client.Start(ctx, func(raw json.RawMessage) error {
				// Keep the stream healthy: never return errors for parse issues.
				targets := traffic.ParseDump978NDJSON(raw)
				if len(targets) > 0 && r.trafficStore != nil {
					r.trafficStore.UpsertMany(time.Now().UTC(), targets)
				}
				return nil
			}); err != nil {
				return fmt.Errorf("uat978 ndjson start: %w", err)
			}
			r.uat978Stream = client
			go func() {
				time.Sleep(2 * time.Second)
				snap := client.Snapshot(time.Now().UTC())
				if snap.State != "connected" {
					log.Printf("uat978 ndjson state=%s addr=%s last_error=%s", snap.State, snap.Addr, snap.LastError)
				}
			}()
		}
		if rawEndpoint != "" {
			lc, err := decoder.NewLineClient(decoder.LineClientConfig{
				Name: "uat978-raw",
				Addr: rawEndpoint,
			})
			if err != nil {
				return fmt.Errorf("uat978 raw: %w", err)
			}
			if err := lc.Start(ctx, func(line []byte) error {
				payload, ok := traffic.ParseDump978RawUplinkLine(line)
				if !ok {
					return nil
				}
				frame := gdl90.UATUplinkFrame(payload)
				select {
				case r.uat978UplinkQ <- frame:
				default:
				}
				return nil
			}); err != nil {
				return fmt.Errorf("uat978 raw start: %w", err)
			}
			r.uat978Raw = lc
			go func() {
				time.Sleep(2 * time.Second)
				snap := lc.Snapshot(time.Now().UTC())
				if snap.State != "connected" {
					log.Printf("uat978 raw state=%s addr=%s last_error=%s", snap.State, snap.Addr, snap.LastError)
				}
			}()
		}
	}

	return nil
}

func (r *liveRuntime) Close() {
	if r == nil {
		return
	}
	if r.ahrsSvc != nil {
		r.ahrsSvc.Close()
		r.ahrsSvc = nil
	}
	if r.gpsSvc != nil {
		r.gpsSvc.Close()
		r.gpsSvc = nil
	}
	if r.fanSvc != nil {
		r.fanSvc.Close()
		r.fanSvc = nil
	}
	if r.adsb1090File != nil {
		r.adsb1090File.Close()
		r.adsb1090File = nil
	}
	if r.uat978Stream != nil {
		r.uat978Stream.Close()
		r.uat978Stream = nil
	}
	if r.uat978Raw != nil {
		r.uat978Raw.Close()
		r.uat978Raw = nil
	}
	if r.adsb1090Sup != nil {
		r.adsb1090Sup.Close()
		r.adsb1090Sup = nil
	}
	if r.uat978Sup != nil {
		r.uat978Sup.Close()
		r.uat978Sup = nil
	}
	if r.ticker != nil {
		r.ticker.Stop()
		r.ticker = nil
	}
}

func (r *liveRuntime) TrafficTargets(nowUTC time.Time) []gdl90.Traffic {
	if r == nil || r.trafficStore == nil {
		return nil
	}
	return r.trafficStore.Snapshot(nowUTC)
}

func (r *liveRuntime) ADSB1090DecoderSnapshot(nowUTC time.Time) (web.DecoderStatusSnapshot, bool) {
	if r == nil {
		return web.DecoderStatusSnapshot{}, false
	}
	cur := r.cfg.ADSB1090
	if !cur.Enable {
		return web.DecoderStatusSnapshot{Enabled: false}, true
	}
	jsonFile := strings.TrimSpace(cur.Decoder.JSONFile)
	snap := web.DecoderStatusSnapshot{
		Enabled:   true,
		SerialTag: strings.TrimSpace(cur.SDR.SerialTag),
		Command:   strings.TrimSpace(cur.Decoder.Command),
		JSONFile:  jsonFile,
	}
	if r.adsb1090Sup != nil {
		snap.Supervisor = r.adsb1090Sup.Snapshot()
	}
	if r.adsb1090File != nil {
		fs := r.adsb1090File.Snapshot(nowUTC)
		snap.File = &fs
	}
	return snap, true
}

func (r *liveRuntime) UAT978DecoderSnapshot(nowUTC time.Time) (web.DecoderStatusSnapshot, bool) {
	if r == nil {
		return web.DecoderStatusSnapshot{}, false
	}
	cur := r.cfg.UAT978
	if !cur.Enable {
		return web.DecoderStatusSnapshot{Enabled: false}, true
	}
	ep := strings.TrimSpace(cur.Decoder.JSONAddr)
	if ep == "" {
		ep = strings.TrimSpace(cur.Decoder.JSONListen)
	}
	rawEP := strings.TrimSpace(cur.Decoder.RawAddr)
	if rawEP == "" {
		rawEP = strings.TrimSpace(cur.Decoder.RawListen)
	}
	snap := web.DecoderStatusSnapshot{
		Enabled:      true,
		SerialTag:    strings.TrimSpace(cur.SDR.SerialTag),
		Command:      strings.TrimSpace(cur.Decoder.Command),
		JSONEndpoint: ep,
		RawEndpoint:  rawEP,
	}
	if r.uat978Sup != nil {
		snap.Supervisor = r.uat978Sup.Snapshot()
	}
	if r.uat978Stream != nil {
		st := r.uat978Stream.Snapshot(nowUTC)
		snap.Stream = &st
	}
	if r.uat978Raw != nil {
		rs := r.uat978Raw.Snapshot(nowUTC)
		snap.RawStream = &rs
	}
	return snap, true
}

func (r *liveRuntime) DrainUAT978UplinkFrames(max int) [][]byte {
	if r == nil || r.uat978UplinkQ == nil {
		return nil
	}
	if max <= 0 {
		max = 1
	}
	out := make([][]byte, 0, max)
	for i := 0; i < max; i++ {
		select {
		case f := <-r.uat978UplinkQ:
			if len(f) > 0 {
				out = append(out, f)
			}
		default:
			return out
		}
	}
	return out
}

func (r *liveRuntime) FanSnapshot() (fancontrol.Snapshot, bool) {
	if r == nil || r.fanSvc == nil {
		return fancontrol.Snapshot{}, false
	}
	return r.fanSvc.Snapshot(), true
}

func (r *liveRuntime) AHRSSnapshot() (ahrs.Snapshot, bool) {
	if r == nil || r.ahrsSvc == nil {
		return ahrs.Snapshot{}, false
	}
	return r.ahrsSvc.Snapshot(), true
}

func (r *liveRuntime) GPSSnapshot() (gps.Snapshot, bool) {
	if r == nil || r.gpsSvc == nil {
		return gps.Snapshot{}, false
	}
	return r.gpsSvc.Snapshot(), true
}

func (r *liveRuntime) AHRSSetLevel() error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.SetLevel()
}

func (r *liveRuntime) AHRSZeroDrift(ctx context.Context) error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.ZeroDrift(ctx)
}

func (r *liveRuntime) AHRSOrientForward(ctx context.Context) error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.OrientForward(ctx)
}

func (r *liveRuntime) AHRSOrientDone(ctx context.Context) error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.OrientDone(ctx)
}

func (r *liveRuntime) AHRSOrientation() (forwardAxis int, gravity [3]float64, gravityOK bool) {
	if r == nil || r.ahrsSvc == nil {
		return 0, [3]float64{}, false
	}
	return r.ahrsSvc.Orientation()
}

func (r *liveRuntime) TickChan() <-chan time.Time {
	if r == nil || r.ticker == nil {
		return nil
	}
	return r.ticker.C
}

func (r *liveRuntime) Config() config.Config {
	if r == nil {
		return config.Config{}
	}
	return r.cfg
}

func (r *liveRuntime) Scenario() *scenarioRuntime {
	if r == nil {
		return nil
	}
	return &r.scenario
}

func (r *liveRuntime) Apply(next config.Config) error {
	if r == nil {
		return fmt.Errorf("runtime is nil")
	}

	c := next
	if err := config.DefaultAndValidate(&c); err != nil {
		return err
	}

	// Keep live scope intentionally small/safe.
	if c.GDL90.Record.Enable != r.cfg.GDL90.Record.Enable || c.GDL90.Record.Path != r.cfg.GDL90.Record.Path {
		return fmt.Errorf("gdl90.record settings require restart")
	}
	if c.GDL90.Replay.Enable != r.cfg.GDL90.Replay.Enable || c.GDL90.Replay.Path != r.cfg.GDL90.Replay.Path || c.GDL90.Replay.Speed != r.cfg.GDL90.Replay.Speed || c.GDL90.Replay.Loop != r.cfg.GDL90.Replay.Loop {
		return fmt.Errorf("gdl90.replay settings require restart")
	}
	if c.Web.Listen != r.cfg.Web.Listen {
		return fmt.Errorf("web.listen requires restart")
	}
	if c.AHRS.Enable != r.cfg.AHRS.Enable || c.AHRS.I2CBus != r.cfg.AHRS.I2CBus || c.AHRS.IMUAddr != r.cfg.AHRS.IMUAddr || c.AHRS.BaroAddr != r.cfg.AHRS.BaroAddr {
		return fmt.Errorf("ahrs settings require restart")
	}
	if c.GPS.Enable != r.cfg.GPS.Enable || strings.TrimSpace(c.GPS.Device) != strings.TrimSpace(r.cfg.GPS.Device) || c.GPS.Baud != r.cfg.GPS.Baud {
		return fmt.Errorf("gps settings require restart")
	}
	if c.Fan.Enable != r.cfg.Fan.Enable || c.Fan.PWMPin != r.cfg.Fan.PWMPin || c.Fan.PWMFrequency != r.cfg.Fan.PWMFrequency || c.Fan.TempTargetC != r.cfg.Fan.TempTargetC || c.Fan.PWMDutyMin != r.cfg.Fan.PWMDutyMin || c.Fan.UpdateInterval != r.cfg.Fan.UpdateInterval {
		return fmt.Errorf("fan settings require restart")
	}
	if !decoderBandEqual(c.ADSB1090, r.cfg.ADSB1090) {
		return fmt.Errorf("adsb1090 settings require restart")
	}
	if !decoderBandEqual(c.UAT978, r.cfg.UAT978) {
		return fmt.Errorf("uat978 settings require restart")
	}

	// Pre-validate side effects before committing anything.
	var nextBroadcaster *udp.Broadcaster
	if strings.TrimSpace(c.GDL90.Dest) != strings.TrimSpace(r.cfg.GDL90.Dest) {
		b, err := udp.NewBroadcaster(c.GDL90.Dest)
		if err != nil {
			return fmt.Errorf("udp broadcaster init failed: %w", err)
		}
		nextBroadcaster = b
	}

	nextScenario, err := loadScenarioFromConfig(c)
	if err != nil {
		if nextBroadcaster != nil {
			_ = nextBroadcaster.Close()
		}
		return err
	}

	// Commit: swap broadcaster.
	if nextBroadcaster != nil {
		r.sender.Swap(nextBroadcaster)
	}

	// Commit: swap ticker.
	if r.ticker != nil && c.GDL90.Interval != r.cfg.GDL90.Interval {
		r.ticker.Stop()
		r.ticker = time.NewTicker(c.GDL90.Interval)
	}

	// Commit: scenario runtime (reset elapsed on any scenario config change).
	if c.Sim.Scenario.Enable != r.cfg.Sim.Scenario.Enable ||
		c.Sim.Scenario.Path != r.cfg.Sim.Scenario.Path ||
		c.Sim.Scenario.StartTimeUTC != r.cfg.Sim.Scenario.StartTimeUTC ||
		c.Sim.Scenario.Loop != r.cfg.Sim.Scenario.Loop {
		r.scenario = nextScenario
		r.scenario.elapsed = 0
	}

	r.cfg = c
	r.status.SetStatic(r.cfg.GDL90.Dest, r.cfg.GDL90.Interval.String(), simInfoSnapshot(r.resolvedConfigPath, r.cfg))
	return nil
}
