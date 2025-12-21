package udp

import (
	"fmt"
	"net"
)

type udpConn interface {
	Write(p []byte) (int, error)
	Close() error
}

type udpDialer func(network string, laddr, raddr *net.UDPAddr) (udpConn, error)
type udpResolver func(network, address string) (*net.UDPAddr, error)

type Broadcaster struct {
	dest string
	conn udpConn
}

func NewBroadcaster(dest string) (*Broadcaster, error) {
	return newBroadcaster(dest, net.ResolveUDPAddr, func(network string, laddr, raddr *net.UDPAddr) (udpConn, error) {
		return net.DialUDP(network, laddr, raddr)
	})
}

func newBroadcaster(dest string, resolve udpResolver, dial udpDialer) (*Broadcaster, error) {
	addr, err := resolve("udp", dest)
	if err != nil {
		return nil, fmt.Errorf("resolve dest: %w", err)
	}

	// DialUDP selects a suitable local address automatically.
	conn, err := dial("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %w", err)
	}

	return &Broadcaster{
		dest: dest,
		conn: conn,
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
