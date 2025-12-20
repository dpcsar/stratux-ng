package udp

import (
	"context"
	"fmt"
	"net"
	"time"
)

type Broadcaster struct {
	dest     string
	interval time.Duration
	conn     *net.UDPConn
}

func NewBroadcaster(dest string, interval time.Duration, _ []byte) (*Broadcaster, error) {
	addr, err := net.ResolveUDPAddr("udp", dest)
	if err != nil {
		return nil, fmt.Errorf("resolve dest: %w", err)
	}

	// DialUDP selects a suitable local address automatically.
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %w", err)
	}

	return &Broadcaster{
		dest:     dest,
		interval: interval,
		conn:     conn,
	}, nil
}

func (b *Broadcaster) Run(ctx context.Context, payload func(seq uint64) []byte) error {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	var seq uint64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			seq++
			p := payload(seq)
			if len(p) == 0 {
				continue
			}
			if _, err := b.conn.Write(p); err != nil {
				return err
			}
		}
	}
}

func (b *Broadcaster) Close() error {
	if b.conn == nil {
		return nil
	}
	return b.conn.Close()
}
