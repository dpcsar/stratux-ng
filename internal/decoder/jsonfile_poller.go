package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type JSONFilePollerConfig struct {
	Name     string
	Path     string
	Interval time.Duration
}

type JSONFilePoller struct {
	cfg JSONFilePollerConfig

	started atomic.Bool
	closed  atomic.Bool

	mu           sync.RWMutex
	state        string
	lastErr      string
	lastSeen     time.Time
	lastModTime  time.Time
	lastFileSize int64

	reads   uint64
	skips   uint64
	errors  uint64
	updates uint64

	cancel context.CancelFunc
	done   chan struct{}
}

type JSONFileSnapshot struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Interval    string `json:"interval"`
	State       string `json:"state"`
	LastError   string `json:"last_error,omitempty"`
	LastSeenUTC string `json:"last_seen_utc,omitempty"`

	Reads   uint64 `json:"reads"`
	Skips   uint64 `json:"skips"`
	Errors  uint64 `json:"errors"`
	Updates uint64 `json:"updates"`
}

func NewJSONFilePoller(cfg JSONFilePollerConfig) (*JSONFilePoller, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("jsonfile poller name is required")
	}
	if cfg.Path == "" {
		return nil, fmt.Errorf("jsonfile poller path is required")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 1 * time.Second
	}
	return &JSONFilePoller{cfg: cfg, state: "stopped", done: make(chan struct{})}, nil
}

// Start polls the configured file periodically. When the file changes, onObject
// is called with the full JSON document bytes.
//
// onObject should be fast; if it can block, it should offload work.
func (p *JSONFilePoller) Start(ctx context.Context, onObject func(raw json.RawMessage) error) error {
	if p == nil {
		return fmt.Errorf("jsonfile poller is nil")
	}
	if p.closed.Load() {
		return fmt.Errorf("jsonfile poller is closed")
	}
	if onObject == nil {
		return fmt.Errorf("jsonfile onObject is nil")
	}
	if p.started.Swap(true) {
		return fmt.Errorf("jsonfile poller already started")
	}

	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.setState("polling", "")

	go func() {
		defer close(p.done)
		p.runLoop(runCtx, onObject)
	}()
	return nil
}

func (p *JSONFilePoller) Close() {
	if p == nil {
		return
	}
	if p.closed.Swap(true) {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	<-p.done
}

func (p *JSONFilePoller) Snapshot(nowUTC time.Time) JSONFileSnapshot {
	if p == nil {
		return JSONFileSnapshot{}
	}
	p.mu.RLock()
	state := p.state
	lastErr := p.lastErr
	lastSeen := p.lastSeen
	p.mu.RUnlock()

	out := JSONFileSnapshot{
		Name:     p.cfg.Name,
		Path:     p.cfg.Path,
		Interval: p.cfg.Interval.String(),
		State:    state,
		Reads:    atomic.LoadUint64(&p.reads),
		Skips:    atomic.LoadUint64(&p.skips),
		Errors:   atomic.LoadUint64(&p.errors),
		Updates:  atomic.LoadUint64(&p.updates),
	}
	if lastErr != "" {
		out.LastError = lastErr
	}
	if !lastSeen.IsZero() {
		out.LastSeenUTC = lastSeen.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func (p *JSONFilePoller) runLoop(ctx context.Context, onObject func(raw json.RawMessage) error) {
	ticker := time.NewTicker(p.cfg.Interval)
	defer ticker.Stop()

	// Try immediately so startup is responsive.
	p.tick(ctx, onObject)

	for {
		select {
		case <-ctx.Done():
			p.setState("stopped", "")
			return
		case <-ticker.C:
			p.tick(ctx, onObject)
		}
	}
}

func (p *JSONFilePoller) tick(ctx context.Context, onObject func(raw json.RawMessage) error) {
	_ = ctx

	st, err := os.Stat(p.cfg.Path)
	if err != nil {
		atomic.AddUint64(&p.errors, 1)
		p.setState("error", err.Error())
		return
	}

	p.mu.RLock()
	prevMod := p.lastModTime
	prevSize := p.lastFileSize
	p.mu.RUnlock()

	// Fast path: unchanged.
	if !prevMod.IsZero() && st.ModTime().Equal(prevMod) && st.Size() == prevSize {
		atomic.AddUint64(&p.skips, 1)
		p.setState("polling", "")
		return
	}

	b, err := os.ReadFile(p.cfg.Path)
	atomic.AddUint64(&p.reads, 1)
	if err != nil {
		atomic.AddUint64(&p.errors, 1)
		p.setState("error", err.Error())
		return
	}

	// Validate quickly so we don't call downstream with garbage.
	var js any
	if err := json.Unmarshal(b, &js); err != nil {
		atomic.AddUint64(&p.errors, 1)
		p.setState("error", "json parse: "+err.Error())
		return
	}

	raw := append(json.RawMessage(nil), b...)
	if err := onObject(raw); err != nil {
		atomic.AddUint64(&p.errors, 1)
		p.setState("error", "handler: "+err.Error())
		return
	}

	now := time.Now().UTC()
	p.mu.Lock()
	p.lastSeen = now
	p.lastModTime = st.ModTime()
	p.lastFileSize = st.Size()
	p.mu.Unlock()

	atomic.AddUint64(&p.updates, 1)
	p.setState("polling", "")
}

func (p *JSONFilePoller) setState(state string, lastErr string) {
	p.mu.Lock()
	p.state = state
	if lastErr != "" {
		p.lastErr = lastErr
	}
	p.mu.Unlock()
}
