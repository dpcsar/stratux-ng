package decoder

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type NDJSONClientConfig struct {
	Name string
	Addr string

	ReconnectDelay time.Duration
	MaxLineBytes   int

	// DialTimeout is used for the initial TCP connect.
	DialTimeout time.Duration
}

type NDJSONClient struct {
	cfg NDJSONClientConfig

	started atomic.Bool
	closed  atomic.Bool

	mu       sync.RWMutex
	state    string
	lastErr  string
	lastSeen time.Time
	count    uint64

	cancel context.CancelFunc
	done   chan struct{}
}

type NDJSONSnapshot struct {
	Name        string `json:"name"`
	Addr        string `json:"addr"`
	State       string `json:"state"`
	LastError   string `json:"last_error,omitempty"`
	LastSeenUTC string `json:"last_seen_utc,omitempty"`
	Messages    uint64 `json:"messages"`
}

func NewNDJSONClient(cfg NDJSONClientConfig) (*NDJSONClient, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("ndjson client name is required")
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("ndjson client addr is required")
	}
	if cfg.ReconnectDelay <= 0 {
		cfg.ReconnectDelay = 1 * time.Second
	}
	if cfg.MaxLineBytes <= 0 {
		cfg.MaxLineBytes = 256 * 1024
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 2 * time.Second
	}

	return &NDJSONClient{cfg: cfg, state: "stopped", done: make(chan struct{})}, nil
}

// Start connects to the configured TCP endpoint and reads newline-delimited JSON.
// For each JSON object, onObject is called with a copy of the raw bytes.
//
// onObject should be fast; if it can block, it should offload work.
func (c *NDJSONClient) Start(ctx context.Context, onObject func(raw json.RawMessage) error) error {
	if c == nil {
		return fmt.Errorf("ndjson client is nil")
	}
	if c.closed.Load() {
		return fmt.Errorf("ndjson client is closed")
	}
	if onObject == nil {
		return fmt.Errorf("ndjson onObject is nil")
	}
	if c.started.Swap(true) {
		return fmt.Errorf("ndjson client already started")
	}

	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.setState("connecting", "")

	go func() {
		defer close(c.done)
		c.runLoop(runCtx, onObject)
	}()
	return nil
}

func (c *NDJSONClient) Close() {
	if c == nil {
		return
	}
	if c.closed.Swap(true) {
		return
	}
	if c.cancel != nil {
		c.cancel()
	}
	<-c.done
}

func (c *NDJSONClient) Snapshot(nowUTC time.Time) NDJSONSnapshot {
	if c == nil {
		return NDJSONSnapshot{}
	}
	c.mu.RLock()
	state := c.state
	lastErr := c.lastErr
	lastSeen := c.lastSeen
	count := c.count
	c.mu.RUnlock()

	out := NDJSONSnapshot{
		Name:      c.cfg.Name,
		Addr:      c.cfg.Addr,
		State:     state,
		LastError: lastErr,
		Messages:  count,
	}
	if !lastSeen.IsZero() {
		out.LastSeenUTC = lastSeen.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func (c *NDJSONClient) runLoop(ctx context.Context, onObject func(raw json.RawMessage) error) {
	dialer := &net.Dialer{Timeout: c.cfg.DialTimeout}

	for {
		select {
		case <-ctx.Done():
			c.setState("stopped", "")
			return
		default:
		}

		c.setState("connecting", "")
		conn, err := dialer.DialContext(ctx, "tcp", c.cfg.Addr)
		if err != nil {
			c.setState("error", err.Error())
			if !sleepCtx(ctx, c.cfg.ReconnectDelay) {
				c.setState("stopped", "")
				return
			}
			continue
		}

		c.setState("connected", "")
		_ = conn.SetReadDeadline(time.Time{})
		reader := bufio.NewReader(conn)

		for {
			select {
			case <-ctx.Done():
				_ = conn.Close()
				c.setState("stopped", "")
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				_ = conn.Close()
				if errors.Is(err, net.ErrClosed) {
					c.setState("disconnected", "")
				} else {
					c.setState("disconnected", err.Error())
				}
				break
			}

			if len(line) == 0 {
				continue
			}
			if len(line) > c.cfg.MaxLineBytes {
				// Drop oversized lines to avoid memory issues.
				c.setState("error", fmt.Sprintf("ndjson line too large (%d bytes)", len(line)))
				continue
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			// Validate JSON object quickly. We keep raw bytes for downstream.
			var js any
			if err := json.Unmarshal(line, &js); err != nil {
				c.setState("error", "json parse: "+err.Error())
				continue
			}

			raw := append(json.RawMessage(nil), line...)
			if err := onObject(raw); err != nil {
				c.setState("error", "handler: "+err.Error())
				continue
			}

			now := time.Now().UTC()
			c.mu.Lock()
			c.lastSeen = now
			c.count++
			c.mu.Unlock()
		}

		if !sleepCtx(ctx, c.cfg.ReconnectDelay) {
			c.setState("stopped", "")
			return
		}
	}
}

func (c *NDJSONClient) setState(state string, lastErr string) {
	c.mu.Lock()
	c.state = state
	if lastErr != "" {
		c.lastErr = lastErr
	} else {
		// Clear stale errors on healthy/neutral states so status output doesn't
		// look broken after a transient startup failure.
		if state == "connected" || state == "connecting" || state == "stopped" {
			c.lastErr = ""
		}
	}
	c.mu.Unlock()
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
