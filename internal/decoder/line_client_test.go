package decoder

import (
	"testing"
	"time"
)

func TestLineClient_setState_ClearsStaleErrorOnConnected(t *testing.T) {
	c, err := NewLineClient(LineClientConfig{Name: "t", Addr: "127.0.0.1:1"})
	if err != nil {
		t.Fatalf("NewLineClient: %v", err)
	}

	c.setState("error", "dial tcp: connection refused")
	c.setState("connected", "")

	snap := c.Snapshot(time.Time{})
	if snap.State != "connected" {
		t.Fatalf("state=%q want %q", snap.State, "connected")
	}
	if snap.LastError != "" {
		t.Fatalf("last_error=%q want empty", snap.LastError)
	}
}
