package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/replay"
	"stratux-ng/internal/sim"
	"stratux-ng/internal/udp"
	"stratux-ng/internal/web"
)

type ahrsProxy struct {
	mu sync.RWMutex
	rt *liveRuntime
}

func (p *ahrsProxy) setRuntime(rt *liveRuntime) {
	p.mu.Lock()
	p.rt = rt
	p.mu.Unlock()
}

func (p *ahrsProxy) clearRuntime(rt *liveRuntime) {
	p.mu.Lock()
	if p.rt == rt {
		p.rt = nil
	}
	p.mu.Unlock()
}

func (p *ahrsProxy) SetLevel() error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSSetLevel()
}

func (p *ahrsProxy) ZeroDrift(ctx context.Context) error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSZeroDrift(ctx)
}

func (p *ahrsProxy) OrientForward(ctx context.Context) error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSOrientForward(ctx)
}

func (p *ahrsProxy) OrientDone(ctx context.Context) error {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return rt.AHRSOrientDone(ctx)
}

func (p *ahrsProxy) Orientation() (forwardAxis int, gravity [3]float64, gravityOK bool) {
	p.mu.RLock()
	rt := p.rt
	p.mu.RUnlock()
	if rt == nil {
		return 0, [3]float64{}, false
	}
	return rt.AHRSOrientation()
}

func decodeAttitudeFromFrames(frames [][]byte) web.AttitudeSnapshot {
	var out web.AttitudeSnapshot

	for _, frame := range frames {
		msg, crcOK, err := gdl90.Unframe(frame)
		if err != nil || !crcOK || len(msg) == 0 {
			continue
		}

		// Stratux heartbeat: bit0 = AHRS valid.
		if msg[0] == 0xCC && len(msg) >= 2 {
			out.Valid = (msg[1] & 0x01) != 0
		}

		// ForeFlight AHRS: 0x65, sub-id 0x01.
		if msg[0] == 0x65 && len(msg) >= 12 && msg[1] == 0x01 {
			roll := int16(msg[2])<<8 | int16(msg[3])
			pitch := int16(msg[4])<<8 | int16(msg[5])
			if roll != int16(0x7FFF) {
				v := float64(roll) / 10.0
				out.RollDeg = &v
			}
			if pitch != int16(0x7FFF) {
				v := float64(pitch) / 10.0
				out.PitchDeg = &v
			}
			continue
		}

		// Stratux LE AHRS report: starts with 0x4C 0x45 0x01 0x01.
		if len(msg) >= 24 && msg[0] == 0x4C && msg[1] == 0x45 && msg[2] == 0x01 && msg[3] == 0x01 {
			roll := int16(msg[4])<<8 | int16(msg[5])
			pitch := int16(msg[6])<<8 | int16(msg[7])
			hdg := int16(msg[8])<<8 | int16(msg[9])
			if roll != int16(0x7FFF) {
				v := float64(roll) / 10.0
				out.RollDeg = &v
			}
			if pitch != int16(0x7FFF) {
				v := float64(pitch) / 10.0
				out.PitchDeg = &v
			}
			if hdg != int16(0x7FFF) {
				v := float64(hdg) / 10.0
				out.HeadingDeg = &v
			}
			continue
		}
	}
	return out
}

type ctxSleeper struct{ ctx context.Context }

func (cs ctxSleeper) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-cs.ctx.Done():
		return
	case <-t.C:
		return
	}
}

type replayOpener func(path string) (io.ReadCloser, error)
type frameSender func(frame []byte) error

type applyRequest struct {
	cfg  config.Config
	resp chan error
}

type safeBroadcaster struct {
	mu sync.Mutex
	b  *udp.Broadcaster
}

func (s *safeBroadcaster) Send(payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.b
	if b == nil {
		return errors.New("udp broadcaster is nil")
	}
	return b.Send(payload)
}

func (s *safeBroadcaster) Swap(next *udp.Broadcaster) {
	s.mu.Lock()
	old := s.b
	s.b = next
	if old != nil {
		_ = old.Close()
	}
	s.mu.Unlock()
}

func (s *safeBroadcaster) Close() {
	s.mu.Lock()
	old := s.b
	s.b = nil
	if old != nil {
		_ = old.Close()
	}
	s.mu.Unlock()
}

type scenarioRuntime struct {
	scenario    *sim.Scenario
	startUTC    time.Time
	ownshipICAO [3]byte
	trafficICAO [][3]byte
	elapsed     time.Duration
}

func simInfoSnapshot(resolvedConfigPath string, cfg config.Config) map[string]any {
	return map[string]any{
		"config_path": resolvedConfigPath,
		"scenario":    cfg.Sim.Scenario.Enable,
		"traffic":     cfg.Sim.Traffic.Enable,
		"record":      cfg.GDL90.Record.Enable,
		"replay":      cfg.GDL90.Replay.Enable,
	}
}

func loadScenarioFromConfig(cfg config.Config) (scenarioRuntime, error) {
	var rt scenarioRuntime
	if !cfg.Sim.Scenario.Enable {
		return rt, nil
	}
	startUTC, err := time.Parse(time.RFC3339, cfg.Sim.Scenario.StartTimeUTC)
	if err != nil {
		return scenarioRuntime{}, fmt.Errorf("scenario start_time_utc parse failed: %w", err)
	}
	script, err := sim.LoadScenarioScript(cfg.Sim.Scenario.Path)
	if err != nil {
		return scenarioRuntime{}, fmt.Errorf("scenario load failed: %w", err)
	}
	sc, err := sim.NewScenario(script)
	if err != nil {
		return scenarioRuntime{}, fmt.Errorf("scenario validate failed: %w", err)
	}

	rt.scenario = sc
	rt.startUTC = startUTC

	ownICAO := strings.TrimSpace(script.Ownship.ICAO)
	if ownICAO == "" {
		ownICAO = "F00000"
	}
	icao, err := gdl90.ParseICAOHex(ownICAO)
	if err != nil {
		return scenarioRuntime{}, fmt.Errorf("scenario invalid ownship icao %q: %w", ownICAO, err)
	}
	rt.ownshipICAO = icao

	rt.trafficICAO = make([][3]byte, len(script.Traffic))
	for i := range script.Traffic {
		icaoStr := strings.TrimSpace(script.Traffic[i].ICAO)
		if icaoStr == "" {
			// Deterministic, non-zero ICAO for each target.
			// Keep it in the "self assigned" range (not necessarily valid ICAO).
			rt.trafficICAO[i] = [3]byte{0xF1, 0x00, byte(i + 1)}
			continue
		}
		p, err := gdl90.ParseICAOHex(icaoStr)
		if err != nil {
			return scenarioRuntime{}, fmt.Errorf("scenario invalid traffic[%d] icao %q: %w", i, icaoStr, err)
		}
		rt.trafficICAO[i] = p
	}

	return rt, nil
}

func runReplay(ctx context.Context, cfg config.Config, open replayOpener, send frameSender) error {
	if open == nil {
		open = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	}
	if send == nil {
		return errors.New("send is nil")
	}

	rc, err := open(cfg.GDL90.Replay.Path)
	if err != nil {
		return err
	}
	defer rc.Close()

	recs, err := replay.NewReader(rc).ReadAll()
	if err != nil {
		return err
	}

	return replay.Play(recs, cfg.GDL90.Replay.Speed, cfg.GDL90.Replay.Loop, ctxSleeper{ctx: ctx}, func(frame []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return send(frame)
	})
}

func main() {
	var configPath string
	var resolvedConfigPath string
	var recordPath string
	var replayPath string
	var replaySpeed float64
	var replayLoop bool
	var logSummaryPath string
	var listenMode bool
	var listenAddr string
	var listenHex bool
	var webListen string

	flag.StringVar(&configPath, "config", "", "Path to YAML config (optional; defaults to /data/stratux-ng/config.yaml; STRATUX_NG_CONFIG overrides)")
	flag.StringVar(&recordPath, "record", "", "Record framed GDL90 packets to PATH (overrides config)")
	flag.StringVar(&replayPath, "replay", "", "Replay framed GDL90 packets from PATH (overrides config)")
	flag.Float64Var(&replaySpeed, "replay-speed", -1, "Replay speed multiplier (e.g., 2.0 = 2x). -1 uses config")
	flag.BoolVar(&replayLoop, "replay-loop", false, "Loop replay forever (overrides config when true)")
	flag.StringVar(&logSummaryPath, "log-summary", "", "Print summary of a record/replay log at PATH and exit")
	flag.BoolVar(&listenMode, "listen", false, "Listen for UDP GDL90 frames and dump decoded messages (no transmit)")
	flag.StringVar(&listenAddr, "listen-addr", ":4000", "UDP address to bind in listen mode (e.g. :4000 or 127.0.0.1:4000)")
	flag.BoolVar(&listenHex, "listen-hex", false, "In listen mode, also print raw frame bytes as hex")
	flag.StringVar(&webListen, "web-listen", "", "Web UI listen address (overrides config when non-empty)")
	flag.Parse()

	if strings.TrimSpace(logSummaryPath) != "" {
		if err := printLogSummary(logSummaryPath); err != nil {
			log.Fatalf("log summary failed: %v", err)
		}
		return
	}
	if listenMode {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		if err := runListen(ctx, listenAddr, listenHex); err != nil && ctx.Err() == nil {
			log.Fatalf("listen mode failed: %v", err)
		}
		return
	}

	cfg, resolved, err := config.LoadAuto(configPath)
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}
	resolvedConfigPath = resolved

	// CLI overrides.
	if strings.TrimSpace(webListen) != "" {
		cfg.Web.Listen = webListen
	}

	// CLI overrides (useful for quick record/replay without editing YAML).
	if recordPath != "" {
		cfg.GDL90.Record.Enable = true
		cfg.GDL90.Record.Path = recordPath
	}
	if replayPath != "" {
		cfg.GDL90.Replay.Enable = true
		cfg.GDL90.Replay.Path = replayPath
	}
	if replaySpeed >= 0 {
		cfg.GDL90.Replay.Speed = replaySpeed
	}
	if replayLoop {
		cfg.GDL90.Replay.Loop = true
	}
	if recordPath != "" || replayPath != "" || replaySpeed >= 0 || replayLoop || strings.TrimSpace(webListen) != "" {
		if err := config.DefaultAndValidate(&cfg); err != nil {
			log.Fatalf("config validation failed after CLI overrides: %v", err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logBuf := web.NewLogBuffer(4000)
	log.SetOutput(io.MultiWriter(os.Stdout, logBuf))

	status := web.NewStatus()
	status.SetStatic(cfg.GDL90.Dest, cfg.GDL90.Interval.String(), simInfoSnapshot(resolvedConfigPath, cfg))

	applyCh := make(chan applyRequest)
	applyFunc := func(nextCfg config.Config) error {
		req := applyRequest{cfg: nextCfg, resp: make(chan error, 1)}
		select {
		case applyCh <- req:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case err := <-req.resp:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	log.Printf("web ui enabled listen=%s", cfg.Web.Listen)
	proxy := &ahrsProxy{}
	go func() {
		err := web.Serve(ctx, cfg.Web.Listen, status, web.SettingsStore{ConfigPath: resolvedConfigPath, Apply: applyFunc}, logBuf, proxy)
		if err != nil && ctx.Err() == nil {
			if errors.Is(err, syscall.EACCES) {
				log.Printf("web ui bind failed (permission denied) listen=%s: %v", cfg.Web.Listen, err)
				log.Printf("to use port 80 without running as root, grant CAP_NET_BIND_SERVICE (examples):")
				log.Printf("  setcap: sudo setcap 'cap_net_bind_service=+ep' $(readlink -f ./stratux-ng)")
				log.Printf("  systemd: set AmbientCapabilities=CAP_NET_BIND_SERVICE in the service unit")
				return
			}
			log.Printf("web ui stopped: %v", err)
			cancel()
		}
	}()

	var rec *replay.Writer
	if cfg.GDL90.Record.Enable {
		w, err := replay.CreateWriter(cfg.GDL90.Record.Path)
		if err != nil {
			log.Fatalf("record init failed: %v", err)
		}
		rec = w
		defer func() {
			if err := rec.Close(); err != nil {
				log.Printf("record close failed: %v", err)
			}
		}()
		log.Printf("recording enabled path=%s", cfg.GDL90.Record.Path)
	}

	initialBroadcaster, err := udp.NewBroadcaster(cfg.GDL90.Dest)
	if err != nil {
		log.Fatalf("udp broadcaster init failed: %v", err)
	}
	sender := &safeBroadcaster{b: initialBroadcaster}
	defer sender.Close()

	log.Printf("stratux-ng starting")
	log.Printf("udp dest=%s interval=%s", cfg.GDL90.Dest, cfg.GDL90.Interval)
	if cfg.Sim.Scenario.Enable {
		log.Printf("sim scenario enabled path=%s start_time_utc=%s loop=%t", cfg.Sim.Scenario.Path, cfg.Sim.Scenario.StartTimeUTC, cfg.Sim.Scenario.Loop)
	} else {
		log.Printf("sim ownship enabled center=(%.6f,%.6f)", cfg.Sim.Ownship.CenterLatDeg, cfg.Sim.Ownship.CenterLonDeg)
	}

	go func() {
		rt, err := newLiveRuntime(ctx, cfg, resolvedConfigPath, status, sender)
		if err != nil {
			log.Printf("runtime init failed: %v", err)
			cancel()
			return
		}
		proxy.setRuntime(rt)
		defer rt.Close()
		defer proxy.clearRuntime(rt)

		// Replay sender runs separately so we can still accept live updates.
		if rt.Config().GDL90.Replay.Enable {
			cur := rt.Config()
			log.Printf("replay enabled path=%s speed=%.3gx loop=%t", cur.GDL90.Replay.Path, cur.GDL90.Replay.Speed, cur.GDL90.Replay.Loop)
			go func() {
				err := runReplay(ctx, cur, nil, sender.Send)
				if err != nil && ctx.Err() == nil {
					log.Printf("replay stopped: %v", err)
					cancel()
				}
			}()
		}

		var seq uint64
		for {
			tickC := rt.TickChan()
			select {
			case <-ctx.Done():
				return
			case req := <-applyCh:
				err := rt.Apply(req.cfg)
				req.resp <- err
			case <-tickC:
				curCfg := rt.Config()
				if curCfg.GDL90.Replay.Enable {
					// Replay mode doesn't use the tick loop.
					continue
				}
				seq++
				sc := rt.Scenario()
				var now time.Time
				var frames [][]byte
				if sc != nil && sc.scenario != nil {
					now = sc.startUTC.Add(sc.elapsed)
					frames = buildGDL90FramesFromScenario(curCfg, now.UTC(), sc.elapsed, sc.scenario, sc.ownshipICAO, sc.trafficICAO)
				} else {
					now = time.Now()
					var snap ahrs.Snapshot
					var haveAHRS bool
					if fanSnap, haveFan := rt.FanSnapshot(); haveFan {
						status.SetFan(now.UTC(), fanSnap)
					}
					if curCfg.AHRS.Enable {
						snap, haveAHRS = rt.AHRSSnapshot()
						// Publish AHRS sensor health for the Status page.
						nowUTC := now.UTC()
						imuWorking := haveAHRS && snap.IMUDetected && !snap.IMULastUpdateAt.IsZero() && nowUTC.Sub(snap.IMULastUpdateAt.UTC()) <= 2*time.Second
						baroWorking := haveAHRS && snap.BaroDetected && !snap.BaroLastUpdateAt.IsZero() && nowUTC.Sub(snap.BaroLastUpdateAt.UTC()) <= 5*time.Second
						ah := web.AHRSSensorsSnapshot{
							Enabled:        true,
							IMUDetected:    haveAHRS && snap.IMUDetected,
							BaroDetected:   haveAHRS && snap.BaroDetected,
							IMUWorking:     imuWorking,
							BaroWorking:    baroWorking,
							OrientationSet: snap.OrientationSet,
							ForwardAxis:    snap.OrientationForwardAxis,
							LastError:      snap.LastError,
						}
						if !snap.IMULastUpdateAt.IsZero() {
							ah.IMULastUpdateUTC = snap.IMULastUpdateAt.UTC().Format(time.RFC3339Nano)
						}
						if !snap.BaroLastUpdateAt.IsZero() {
							ah.BaroLastUpdateUTC = snap.BaroLastUpdateAt.UTC().Format(time.RFC3339Nano)
						}
						status.SetAHRSSensors(nowUTC, ah)
					} else {
						status.SetAHRSSensors(now.UTC(), web.AHRSSensorsSnapshot{Enabled: false})
					}
					frames = buildGDL90Frames(curCfg, now.UTC(), haveAHRS, snap)
				}
				status.SetAttitude(now.UTC(), decodeAttitudeFromFrames(frames))
				status.MarkTick(now.UTC(), len(frames))
				for _, frame := range frames {
					if rec != nil {
						if err := rec.WriteFrame(now, frame); err != nil {
							log.Printf("record write failed: %v", err)
							cancel()
							return
						}
					}
					if err := sender.Send(frame); err != nil {
						log.Printf("udp send failed: %v", err)
						cancel()
						return
					}
				}
				if rec != nil {
					if err := rec.Flush(); err != nil {
						log.Printf("record flush failed: %v", err)
						cancel()
						return
					}
				}
				if sc != nil && sc.scenario != nil {
					sc.elapsed += curCfg.GDL90.Interval
				}
			}
		}
	}()

	<-ctx.Done()
	log.Printf("stratux-ng stopping")
}

func runListen(ctx context.Context, addr string, dumpHex bool) error {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer pc.Close()

	log.Printf("listen mode: udp bind=%s", addr)
	buf := make([]byte, 64*1024)
	for {
		_ = pc.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, src, err := pc.ReadFrom(buf)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}
		frame := append([]byte(nil), buf[:n]...)

		msg, crcOK, uerr := gdl90.Unframe(frame)
		if uerr != nil {
			log.Printf("rx src=%s bytes=%d unframe_err=%v", src.String(), len(frame), uerr)
			if dumpHex {
				log.Printf("rx hex=%s", hex.EncodeToString(frame))
			}
			continue
		}

		id := msg[0]
		info := fmt.Sprintf("id=0x%02X", id)
		if id == 0x65 && len(msg) >= 2 {
			info = fmt.Sprintf("id=0x65 sub=0x%02X", msg[1])
		}

		log.Printf("rx src=%s bytes=%d crc_ok=%t %s msg_len=%d", src.String(), len(frame), crcOK, info, len(msg))
		if dumpHex {
			log.Printf("rx hex=%s", hex.EncodeToString(frame))
		}
	}
}

func buildGDL90Frames(cfg config.Config, now time.Time, haveAHRS bool, ahrsSnap ahrs.Snapshot) [][]byte {
	// Ownship sim is always enabled (unless scenario mode is driving frames).
	// Advertise GPS/AHRS as valid so EFBs don't show "No GPS reception".
	gpsValid := true
	ahrsValid := true
	if cfg.AHRS.Enable {
		ahrsValid = haveAHRS && ahrsSnap.Valid
	}
	frames := make([][]byte, 0, 16)
	frames = append(frames,
		gdl90.HeartbeatFrameAt(now, gpsValid, false),
		gdl90.StratuxHeartbeatFrame(gpsValid, ahrsValid),
	)

	// Identify as a Stratux-like device for apps that key off 0x65.
	frames = append(frames, gdl90.ForeFlightIDFrame("Stratux", "Stratux-NG"))

	icao, err := gdl90.ParseICAOHex(cfg.Sim.Ownship.ICAO)
	if err != nil {
		log.Printf("sim ownship invalid sim.ownship.icao: %v", err)
		return frames
	}

	s := sim.OwnshipSim{
		CenterLatDeg: cfg.Sim.Ownship.CenterLatDeg,
		CenterLonDeg: cfg.Sim.Ownship.CenterLonDeg,
		AltFeet:      cfg.Sim.Ownship.AltFeet,
		GroundKt:     cfg.Sim.Ownship.GroundKt,
		RadiusNm:     cfg.Sim.Ownship.RadiusNm,
		Period:       cfg.Sim.Ownship.Period,
	}
	lat, lon, trk, altFeet, vvelFpm := s.Kinematics(now)
	nacp := gdl90.NACpFromHorizontalAccuracyMeters(cfg.Sim.Ownship.GPSHorizontalAccuracyM)

	// GDL90 Ownship Report (0x0A) altitude is pressure altitude when available.
	// Mirror upstream Stratux behavior by preferring baro-derived pressure altitude.
	ownshipAltFeet := altFeet
	if cfg.AHRS.Enable && haveAHRS && ahrsSnap.PressureAltValid {
		ownshipAltFeet = int(ahrsSnap.PressureAltFeet)
	}
	frames = append(frames, gdl90.OwnshipReportFrame(gdl90.Ownship{
		ICAO:        icao,
		LatDeg:      lat,
		LonDeg:      lon,
		AltFeet:     ownshipAltFeet,
		HaveNICNACp: true,
		NIC:         8,
		NACp:        nacp,
		GroundKt:    cfg.Sim.Ownship.GroundKt,
		TrackDeg:    trk,
		OnGround:    cfg.Sim.Ownship.GroundKt == 0,
		VvelFpm:     vvelFpm,
		VvelValid:   true,
		Callsign:    cfg.Sim.Ownship.Callsign,
		Emitter:     0x01,
	}))
	frames = append(frames, gdl90.OwnshipGeometricAltitudeFrame(altFeet))

	// Stratux-like AHRS messages (sim-driven). Even without a real IMU, some
	// EFBs expect to see these message types.
	roll := 0.0
	pitch := 0.0
	if cfg.AHRS.Enable && haveAHRS {
		roll = ahrsSnap.RollDeg
		pitch = ahrsSnap.PitchDeg
	}

	pressureAltFeet := float64(altFeet)
	pressureAltValid := true
	if cfg.AHRS.Enable {
		pressureAltValid = haveAHRS && ahrsSnap.PressureAltValid
		if pressureAltValid {
			pressureAltFeet = ahrsSnap.PressureAltFeet
		}
	}

	vs := vvelFpm
	vsValid := true
	if cfg.AHRS.Enable {
		vsValid = haveAHRS && ahrsSnap.VerticalSpeedValid
		if vsValid {
			vs = ahrsSnap.VerticalSpeedFpm
		}
	}

	att := gdl90.Attitude{
		Valid:                ahrsValid,
		RollDeg:              roll,
		PitchDeg:             pitch,
		HeadingDeg:           trk,
		SlipSkidDeg:          0,
		YawRateDps:           0,
		GLoad:                1.0,
		IndicatedAirspeedKt:  cfg.Sim.Ownship.GroundKt,
		TrueAirspeedKt:       cfg.Sim.Ownship.GroundKt,
		PressureAltitudeFeet: pressureAltFeet,
		PressureAltValid:     pressureAltValid,
		VerticalSpeedFpm:     vs,
		VerticalSpeedValid:   vsValid,
	}
	frames = append(frames,
		gdl90.ForeFlightAHRSFrame(att),
		gdl90.AHRSGDL90LEFrame(att),
	)

	if !cfg.Sim.Traffic.Enable {
		return frames
	}

	ts := sim.TrafficSim{
		CenterLatDeg: cfg.Sim.Ownship.CenterLatDeg,
		CenterLonDeg: cfg.Sim.Ownship.CenterLonDeg,
		BaseAltFeet:  cfg.Sim.Ownship.AltFeet,
		GroundKt:     cfg.Sim.Traffic.GroundKt,
		RadiusNm:     cfg.Sim.Traffic.RadiusNm,
		Period:       cfg.Sim.Traffic.Period,
	}
	targets := ts.Targets(now, cfg.Sim.Traffic.Count)
	for i, tgt := range targets {
		if !tgt.Visible {
			continue
		}
		// Deterministic, non-zero ICAO for each target.
		// Keep it in the "self assigned" range (not necessarily valid ICAO).
		icaoT := [3]byte{0xF1, 0x00, byte(i + 1)}
		frames = append(frames, gdl90.TrafficReportFrame(gdl90.Traffic{
			AddrType:        0x00,
			ICAO:            icaoT,
			LatDeg:          tgt.LatDeg,
			LonDeg:          tgt.LonDeg,
			AltFeet:         tgt.AltFeet,
			NIC:             8,
			NACp:            8,
			GroundKt:        tgt.GroundKt,
			TrackDeg:        tgt.TrackDeg,
			VvelFpm:         tgt.VvelFpm,
			OnGround:        tgt.GroundKt == 0,
			Extrapolated:    tgt.Extrapolated,
			EmitterCategory: 0x01,
			Tail:            fmt.Sprintf("TGT%04d", i+1),
			PriorityStatus:  0,
		}))
	}

	return frames
}

func buildGDL90FramesFromScenario(cfg config.Config, now time.Time, elapsed time.Duration, scenario *sim.Scenario, ownshipICAO [3]byte, trafficICAO [][3]byte) [][]byte {
	// Scenario is always deterministic and self-contained, so we advertise GPS/AHRS valid.
	gpsValid := true
	ahrsValid := true
	frames := make([][]byte, 0, 16)
	frames = append(frames,
		gdl90.HeartbeatFrameAt(now, gpsValid, false),
		gdl90.StratuxHeartbeatFrame(gpsValid, ahrsValid),
	)
	frames = append(frames, gdl90.ForeFlightIDFrame("Stratux", "Stratux-NG"))

	if scenario == nil {
		return frames
	}
	state := scenario.StateAt(elapsed, cfg.Sim.Scenario.Loop)

	nacp := gdl90.NACpFromHorizontalAccuracyMeters(state.Ownship.GPSHorizontalAccuracyM)
	call := state.Ownship.Callsign
	if strings.TrimSpace(call) == "" {
		call = "STRATUX"
	}
	frames = append(frames, gdl90.OwnshipReportFrame(gdl90.Ownship{
		ICAO:        ownshipICAO,
		LatDeg:      state.Ownship.LatDeg,
		LonDeg:      state.Ownship.LonDeg,
		AltFeet:     state.Ownship.AltFeet,
		HaveNICNACp: true,
		NIC:         8,
		NACp:        nacp,
		GroundKt:    state.Ownship.GroundKt,
		TrackDeg:    state.Ownship.TrackDeg,
		OnGround:    state.Ownship.GroundKt == 0,
		Callsign:    call,
		Emitter:     0x01,
	}))
	frames = append(frames, gdl90.OwnshipGeometricAltitudeFrame(state.Ownship.AltFeet))

	att := gdl90.Attitude{
		Valid:                ahrsValid,
		RollDeg:              0,
		PitchDeg:             0,
		HeadingDeg:           state.Ownship.TrackDeg,
		SlipSkidDeg:          0,
		YawRateDps:           0,
		GLoad:                1.0,
		IndicatedAirspeedKt:  state.Ownship.GroundKt,
		TrueAirspeedKt:       state.Ownship.GroundKt,
		PressureAltitudeFeet: float64(state.Ownship.AltFeet),
		PressureAltValid:     true,
		VerticalSpeedFpm:     0,
		VerticalSpeedValid:   true,
	}
	frames = append(frames,
		gdl90.ForeFlightAHRSFrame(att),
		gdl90.AHRSGDL90LEFrame(att),
	)

	for i, tgt := range state.Traffic {
		tail := strings.TrimSpace(tgt.Callsign)
		if tail == "" {
			tail = fmt.Sprintf("TGT%04d", i+1)
		}
		icao := [3]byte{0xF1, 0x00, byte(i + 1)}
		if i < len(trafficICAO) {
			icao = trafficICAO[i]
		}
		frames = append(frames, gdl90.TrafficReportFrame(gdl90.Traffic{
			AddrType:        0x00,
			ICAO:            icao,
			LatDeg:          tgt.LatDeg,
			LonDeg:          tgt.LonDeg,
			AltFeet:         tgt.AltFeet,
			NIC:             8,
			NACp:            8,
			GroundKt:        tgt.GroundKt,
			TrackDeg:        tgt.TrackDeg,
			VvelFpm:         0,
			OnGround:        tgt.GroundKt == 0,
			Extrapolated:    false,
			EmitterCategory: 0x01,
			Tail:            tail,
			PriorityStatus:  0,
		}))
	}

	return frames
}
