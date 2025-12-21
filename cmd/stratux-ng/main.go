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
	"syscall"
	"time"

	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/replay"
	"stratux-ng/internal/sim"
	"stratux-ng/internal/udp"
	"stratux-ng/internal/web"
)

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
	var recordPath string
	var replayPath string
	var replaySpeed float64
	var replayLoop bool
	var logSummaryPath string
	var listenMode bool
	var listenAddr string
	var listenHex bool
	var webEnable bool
	var webListen string

	flag.StringVar(&configPath, "config", "./dev.yaml", "Path to YAML config")
	flag.StringVar(&recordPath, "record", "", "Record framed GDL90 packets to PATH (overrides config)")
	flag.StringVar(&replayPath, "replay", "", "Replay framed GDL90 packets from PATH (overrides config)")
	flag.Float64Var(&replaySpeed, "replay-speed", -1, "Replay speed multiplier (e.g., 2.0 = 2x). -1 uses config")
	flag.BoolVar(&replayLoop, "replay-loop", false, "Loop replay forever (overrides config when true)")
	flag.StringVar(&logSummaryPath, "log-summary", "", "Print summary of a record/replay log at PATH and exit")
	flag.BoolVar(&listenMode, "listen", false, "Listen for UDP GDL90 frames and dump decoded messages (no transmit)")
	flag.StringVar(&listenAddr, "listen-addr", ":4000", "UDP address to bind in listen mode (e.g. :4000 or 127.0.0.1:4000)")
	flag.BoolVar(&listenHex, "listen-hex", false, "In listen mode, also print raw frame bytes as hex")
	flag.BoolVar(&webEnable, "web", false, "Enable Web UI (overrides config)")
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

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	// CLI overrides.
	if webEnable {
		cfg.Web.Enable = true
	}
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
	if recordPath != "" || replayPath != "" || replaySpeed >= 0 || replayLoop || webEnable || strings.TrimSpace(webListen) != "" {
		if err := config.DefaultAndValidate(&cfg); err != nil {
			log.Fatalf("config validation failed after CLI overrides: %v", err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logBuf := web.NewLogBuffer(4000)
	log.SetOutput(io.MultiWriter(os.Stdout, logBuf))

	status := web.NewStatus()
	status.SetStatic(cfg.GDL90.Mode, cfg.GDL90.Dest, cfg.GDL90.Interval.String(), map[string]any{
		"config_path": configPath,
		"scenario":    cfg.Sim.Scenario.Enable,
		"ownship":     cfg.Sim.Ownship.Enable,
		"traffic":     cfg.Sim.Traffic.Enable,
		"record":      cfg.GDL90.Record.Enable,
		"replay":      cfg.GDL90.Replay.Enable,
	})

	if cfg.Web.Enable {
		log.Printf("web ui enabled listen=%s", cfg.Web.Listen)
		go func() {
			err := web.Serve(ctx, cfg.Web.Listen, status, web.SettingsStore{ConfigPath: configPath}, logBuf)
			if err != nil && ctx.Err() == nil {
				log.Printf("web ui stopped: %v", err)
				cancel()
			}
		}()
	}

	broadcaster, err := udp.NewBroadcaster(cfg.GDL90.Dest)
	if err != nil {
		log.Fatalf("udp broadcaster init failed: %v", err)
	}
	defer broadcaster.Close()

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

	log.Printf("stratux-ng starting")
	log.Printf("udp dest=%s interval=%s", cfg.GDL90.Dest, cfg.GDL90.Interval)
	if cfg.GDL90.Mode != "test" {
		log.Printf("output mode=%s", cfg.GDL90.Mode)
	}
	if cfg.Sim.Scenario.Enable {
		log.Printf("sim scenario enabled path=%s start_time_utc=%s loop=%t", cfg.Sim.Scenario.Path, cfg.Sim.Scenario.StartTimeUTC, cfg.Sim.Scenario.Loop)
	} else if cfg.Sim.Ownship.Enable {
		log.Printf("sim ownship enabled center=(%.6f,%.6f)", cfg.Sim.Ownship.CenterLatDeg, cfg.Sim.Ownship.CenterLonDeg)
	}

	go func() {
		if cfg.GDL90.Replay.Enable {
			log.Printf("replay enabled path=%s speed=%.3gx loop=%t", cfg.GDL90.Replay.Path, cfg.GDL90.Replay.Speed, cfg.GDL90.Replay.Loop)
			err := runReplay(ctx, cfg, nil, broadcaster.Send)
			if err != nil && ctx.Err() == nil {
				log.Printf("replay stopped: %v", err)
				cancel()
			}
			return
		}

		// Scenario scripts are loaded once and then sampled deterministically each tick.
		// This avoids wall-clock jitter affecting EFB behavior.
		var scenario *sim.Scenario
		var scenarioStartUTC time.Time
		var scenarioOwnshipICAO [3]byte
		var scenarioTrafficICAO [][3]byte
		if cfg.Sim.Scenario.Enable {
			startUTC, err := time.Parse(time.RFC3339, cfg.Sim.Scenario.StartTimeUTC)
			if err != nil {
				log.Printf("scenario start_time_utc parse failed: %v", err)
				cancel()
				return
			}
			scenarioStartUTC = startUTC

			script, err := sim.LoadScenarioScript(cfg.Sim.Scenario.Path)
			if err != nil {
				log.Printf("scenario load failed: %v", err)
				cancel()
				return
			}
			sc, err := sim.NewScenario(script)
			if err != nil {
				log.Printf("scenario validate failed: %v", err)
				cancel()
				return
			}
			scenario = sc

			ownICAO := strings.TrimSpace(script.Ownship.ICAO)
			if ownICAO == "" {
				ownICAO = "F00000"
			}
			icao, err := gdl90.ParseICAOHex(ownICAO)
			if err != nil {
				log.Printf("scenario invalid ownship icao %q: %v", ownICAO, err)
				cancel()
				return
			}
			scenarioOwnshipICAO = icao

			scenarioTrafficICAO = make([][3]byte, len(script.Traffic))
			for i := range script.Traffic {
				icaoStr := strings.TrimSpace(script.Traffic[i].ICAO)
				if icaoStr == "" {
					// Deterministic, non-zero ICAO for each target.
					// Keep it in the "self assigned" range (not necessarily valid ICAO).
					scenarioTrafficICAO[i] = [3]byte{0xF1, 0x00, byte(i + 1)}
					continue
				}
				p, err := gdl90.ParseICAOHex(icaoStr)
				if err != nil {
					log.Printf("scenario invalid traffic[%d] icao %q: %v", i, icaoStr, err)
					cancel()
					return
				}
				scenarioTrafficICAO[i] = p
			}
		}

		ticker := time.NewTicker(cfg.GDL90.Interval)
		defer ticker.Stop()

		var seq uint64
		var elapsed time.Duration
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				seq++

				switch cfg.GDL90.Mode {
				case "test":
					p := []byte(fmt.Sprintf("%s seq=%d ts=%s", cfg.GDL90.TestPayload, seq, time.Now().UTC().Format(time.RFC3339Nano)))
					if err := broadcaster.Send(p); err != nil {
						log.Printf("udp send failed: %v", err)
						cancel()
						return
					}
				default:
					var now time.Time
					var frames [][]byte
					if scenario != nil {
						now = scenarioStartUTC.Add(elapsed)
						frames = buildGDL90FramesFromScenario(cfg, now.UTC(), elapsed, scenario, scenarioOwnshipICAO, scenarioTrafficICAO)
					} else {
						now = time.Now()
						frames = buildGDL90Frames(cfg, now.UTC())
					}
					status.MarkTick(now.UTC(), len(frames))
					for _, frame := range frames {
						if rec != nil {
							if err := rec.WriteFrame(now, frame); err != nil {
								log.Printf("record write failed: %v", err)
								cancel()
								return
							}
						}
						if err := broadcaster.Send(frame); err != nil {
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
					if scenario != nil {
						elapsed += cfg.GDL90.Interval
					}
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

func buildGDL90Frames(cfg config.Config, now time.Time) [][]byte {
	// If the ownship simulator is enabled, we advertise GPS as valid so EFBs
	// don't show "No GPS reception".
	gpsValid := cfg.Sim.Ownship.Enable
	ahrsValid := cfg.Sim.Ownship.Enable
	frames := make([][]byte, 0, 16)
	frames = append(frames,
		gdl90.HeartbeatFrameAt(now, gpsValid, false),
		gdl90.StratuxHeartbeatFrame(gpsValid, ahrsValid),
	)

	// Identify as a Stratux-like device for apps that key off 0x65.
	frames = append(frames, gdl90.ForeFlightIDFrame("Stratux", "Stratux-NG"))

	if !cfg.Sim.Ownship.Enable {
		return frames
	}

	icao, err := gdl90.ParseICAOHex(cfg.Sim.Ownship.ICAO)
	if err != nil {
		log.Printf("sim ownship disabled: invalid sim.ownship.icao: %v", err)
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
	frames = append(frames, gdl90.OwnshipReportFrame(gdl90.Ownship{
		ICAO:        icao,
		LatDeg:      lat,
		LonDeg:      lon,
		AltFeet:     altFeet,
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
	att := gdl90.Attitude{
		Valid:                ahrsValid,
		RollDeg:              0,
		PitchDeg:             0,
		HeadingDeg:           trk,
		SlipSkidDeg:          0,
		YawRateDps:           0,
		GLoad:                1.0,
		IndicatedAirspeedKt:  cfg.Sim.Ownship.GroundKt,
		TrueAirspeedKt:       cfg.Sim.Ownship.GroundKt,
		PressureAltitudeFeet: float64(altFeet),
		PressureAltValid:     true,
		VerticalSpeedFpm:     vvelFpm,
		VerticalSpeedValid:   true,
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
