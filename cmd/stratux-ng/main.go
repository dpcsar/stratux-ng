package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/sim"
	"stratux-ng/internal/udp"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "./dev.yaml", "Path to YAML config")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	broadcaster, err := udp.NewBroadcaster(cfg.GDL90.Dest)
	if err != nil {
		log.Fatalf("udp broadcaster init failed: %v", err)
	}
	defer broadcaster.Close()

	log.Printf("stratux-ng starting")
	log.Printf("udp dest=%s interval=%s", cfg.GDL90.Dest, cfg.GDL90.Interval)
	if cfg.GDL90.Mode != "test" {
		log.Printf("output mode=%s", cfg.GDL90.Mode)
	}
	if cfg.Sim.Ownship.Enable {
		log.Printf("sim ownship enabled center=(%.6f,%.6f)", cfg.Sim.Ownship.CenterLatDeg, cfg.Sim.Ownship.CenterLonDeg)
	}

	go func() {
		ticker := time.NewTicker(cfg.GDL90.Interval)
		defer ticker.Stop()

		var seq uint64
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
					frames := buildGDL90Frames(cfg, time.Now().UTC())
					for _, f := range frames {
						if err := broadcaster.Send(f); err != nil {
							log.Printf("udp send failed: %v", err)
							cancel()
							return
						}
					}
				}
			}
		}
	}()

	<-ctx.Done()
	log.Printf("stratux-ng stopping")
}

func buildGDL90Frames(cfg config.Config, now time.Time) [][]byte {
	// If the ownship simulator is enabled, we advertise GPS as valid so EFBs
	// don't show "No GPS reception".
	gpsValid := cfg.Sim.Ownship.Enable
	frames := make([][]byte, 0, 16)
	frames = append(frames,
		gdl90.HeartbeatFrame(gpsValid, false),
		gdl90.StratuxHeartbeatFrame(gpsValid, false),
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
	lat, lon, trk := s.Position(now)
	frames = append(frames, gdl90.OwnshipReportFrame(gdl90.Ownship{
		ICAO:     icao,
		LatDeg:   lat,
		LonDeg:   lon,
		AltFeet:  cfg.Sim.Ownship.AltFeet,
		GroundKt: cfg.Sim.Ownship.GroundKt,
		TrackDeg: trk,
		Callsign: cfg.Sim.Ownship.Callsign,
		Emitter:  0x01,
	}))
	frames = append(frames, gdl90.OwnshipGeometricAltitudeFrame(cfg.Sim.Ownship.AltFeet))

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
			VvelFpm:         0,
			OnGround:        false,
			Extrapolated:    false,
			EmitterCategory: 0x01,
			Tail:            fmt.Sprintf("TGT%04d", i+1),
			PriorityStatus:  0,
		}))
	}

	return frames
}
