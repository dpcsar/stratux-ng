package udp

import (
	"errors"
	"net"
	"testing"
)

type fakeConn struct {
	writes    [][]byte
	writeErr  error
	closed    bool
	closeErr  error
	writeHits int
}

func (c *fakeConn) Write(p []byte) (int, error) {
	c.writeHits++
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	cp := append([]byte(nil), p...)
	c.writes = append(c.writes, cp)
	return len(p), nil
}

func (c *fakeConn) Close() error {
	c.closed = true
	return c.closeErr
}

func TestNewBroadcaster_DialsResolvedAddr(t *testing.T) {
	var gotNetwork string
	var gotRaddr *net.UDPAddr
	fc := &fakeConn{}

	resolve := func(network, address string) (*net.UDPAddr, error) {
		return net.ResolveUDPAddr(network, address)
	}

	dial := func(network string, laddr, raddr *net.UDPAddr) (udpConn, error) {
		gotNetwork = network
		gotRaddr = raddr
		return fc, nil
	}

	b, err := newBroadcaster("127.0.0.1:4000", resolve, dial)
	if err != nil {
		t.Fatalf("newBroadcaster() error: %v", err)
	}
	defer b.Close()

	if gotNetwork != "udp" {
		t.Fatalf("network=%q want %q", gotNetwork, "udp")
	}
	if gotRaddr == nil || gotRaddr.Port != 4000 || !gotRaddr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("raddr=%v want 127.0.0.1:4000", gotRaddr)
	}
}

func TestNewBroadcaster_ResolveFailure(t *testing.T) {
	resolveErr := errors.New("nope")
	resolve := func(network, address string) (*net.UDPAddr, error) {
		return nil, resolveErr
	}
	dial := func(network string, laddr, raddr *net.UDPAddr) (udpConn, error) {
		return &fakeConn{}, nil
	}

	_, err := newBroadcaster("bad:addr", resolve, dial)
	if !errors.Is(err, resolveErr) {
		t.Fatalf("err=%v want %v", err, resolveErr)
	}
}

func TestBroadcaster_Send_EmptyNoWrite(t *testing.T) {
	fc := &fakeConn{}
	b := &Broadcaster{dest: "x", conn: fc}

	if err := b.Send(nil); err != nil {
		t.Fatalf("Send(nil) error: %v", err)
	}
	if err := b.Send([]byte{}); err != nil {
		t.Fatalf("Send(empty) error: %v", err)
	}
	if fc.writeHits != 0 {
		t.Fatalf("expected no writes, got %d", fc.writeHits)
	}
}

func TestBroadcaster_Send_WritesPayload(t *testing.T) {
	fc := &fakeConn{}
	b := &Broadcaster{dest: "x", conn: fc}

	p := []byte{0x01, 0x02, 0x03}
	if err := b.Send(p); err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if fc.writeHits != 1 {
		t.Fatalf("expected 1 write, got %d", fc.writeHits)
	}
	if len(fc.writes) != 1 {
		t.Fatalf("expected 1 captured write, got %d", len(fc.writes))
	}
	if string(fc.writes[0]) != string(p) {
		t.Fatalf("write=%v want %v", fc.writes[0], p)
	}
}

func TestBroadcaster_Send_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	fc := &fakeConn{writeErr: wantErr}
	b := &Broadcaster{dest: "x", conn: fc}

	err := b.Send([]byte{0x01})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err=%v want %v", err, wantErr)
	}
}

func TestBroadcaster_Close_NilConnNoPanic(t *testing.T) {
	b := &Broadcaster{}
	if err := b.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}
