package udp

import (
	"fmt"
	"net"
)

type Broadcaster struct {
	dest     string
	conn     *net.UDPConn
}

func NewBroadcaster(dest string) (*Broadcaster, error) {
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
		conn:     conn,
	}, nil
}

func (b *Broadcaster) Send(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	_, err := b.conn.Write(payload)
	return err
}

func (b *Broadcaster) Close() error {
	if b.conn == nil {
		return nil
	}
	return b.conn.Close()
}
