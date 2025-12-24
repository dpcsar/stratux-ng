package decoder

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type LineClientConfig struct {
	Name string
	Addr string

	ReconnectDelay time.Duration
	MaxLineBytes   int

	// DialTimeout is used for the initial TCP connect.
	DialTimeout time.Duration
}

type LineClient struct {
	cfg LineClientConfig

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

type LineSnapshot struct {
	Name        string `json:"name"`
	Addr        string `json:"addr"`
	State       string `json:"state"`
	LastError   string `json:"last_error,omitempty"`
	LastSeenUTC string `json:"last_seen_utc,omitempty"`
	Lines       uint64 `json:"lines"`
}

func NewLineClient(cfg LineClientConfig) (*LineClient, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("line client name is required")
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("line client addr is required")
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

	return &LineClient{cfg: cfg, state: "stopped", done: make(chan struct{})}, nil
}

// Start connects to the configured TCP endpoint and reads newline-delimited
// lines. For each line, onLine is called with a copy of the raw bytes.
//
// onLine should be fast; if it can block, it should offload work.
func (c *LineClient) Start(ctx context.Context, onLine func(line []byte) error) error {
	if c == nil {
		return fmt.Errorf("line client is nil")
	}
	if c.closed.Load() {
		return fmt.Errorf("line client is closed")
	}
	if onLine == nil {
		return fmt.Errorf("line onLine is nil")
	}
	if c.started.Swap(true) {
		return fmt.Errorf("line client already started")
	}

	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.setState("connecting", "")

	go func() {
		defer close(c.done)
		c.runLoop(runCtx, onLine)
	}()
	return nil
}

func (c *LineClient) Close() {
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

func (c *LineClient) Snapshot(nowUTC time.Time) LineSnapshot {
	if c == nil {
		return LineSnapshot{}
	}
	c.mu.RLock()
	state := c.state
	lastErr := c.lastErr
	lastSeen := c.lastSeen
	count := c.count
	c.mu.RUnlock()

	out := LineSnapshot{
		Name:      c.cfg.Name,
		Addr:      c.cfg.Addr,
		State:     state,
		LastError: lastErr,
		Lines:     count,
	}
	if !lastSeen.IsZero() {
		out.LastSeenUTC = lastSeen.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func (c *LineClient) runLoop(ctx context.Context, onLine func(line []byte) error) {
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
				c.setState("error", fmt.Sprintf("line too large (%d bytes)", len(line)))
				continue
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			raw := append([]byte(nil), line...)
			if err := onLine(raw); err != nil {
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

func (c *LineClient) setState(state string, lastErr string) {
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
