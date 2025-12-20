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

	broadcaster, err := udp.NewBroadcaster(cfg.GDL90.Dest, cfg.GDL90.Interval, []byte(cfg.GDL90.TestPayload))
	if err != nil {
		log.Fatalf("udp broadcaster init failed: %v", err)
	}
	defer broadcaster.Close()

	log.Printf("stratux-ng starting")
	log.Printf("udp dest=%s interval=%s", cfg.GDL90.Dest, cfg.GDL90.Interval)

	go func() {
		err := broadcaster.Run(ctx, func(seq uint64) []byte {
			// Placeholder payload until real GDL90 framing/encoding is implemented.
			// This is primarily to validate Wi‑Fi/AP networking end-to-end.
			return []byte(fmt.Sprintf("%s seq=%d ts=%s", cfg.GDL90.TestPayload, seq, time.Now().UTC().Format(time.RFC3339Nano)))
		})
		if err != nil && ctx.Err() == nil {
			log.Printf("udp broadcaster stopped: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Printf("stratux-ng stopping")
}
