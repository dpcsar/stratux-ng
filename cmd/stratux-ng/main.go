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
					// Minimal real GDL90 output: standard heartbeat + Stratux heartbeat.
					// GPS/AHRS validity are hardcoded false until sensors are implemented.
					frames := [][]byte{
						gdl90.HeartbeatFrame(false, false),
						gdl90.StratuxHeartbeatFrame(false, false),
					}
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
