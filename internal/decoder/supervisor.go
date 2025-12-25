package decoder

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type SupervisorConfig struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	WorkDir string

	Restart bool

	BackoffInitial time.Duration
	BackoffMax     time.Duration

	StdoutTailLines int
	StderrTailLines int

	// MaxLineBytes limits any single line stored in the tail buffers.
	// If 0, defaults to 16 KiB.
	MaxLineBytes int
}

type Supervisor struct {
	cfg SupervisorConfig

	started atomic.Bool
	closed  atomic.Bool

	mu      sync.RWMutex
	pid     int
	state   string
	lastErr string

	stdout *tailBuffer
	stderr *tailBuffer

	cancel context.CancelFunc
	done   chan struct{}
}

type Snapshot struct {
	Name      string   `json:"name"`
	Running   bool     `json:"running"`
	PID       int      `json:"pid,omitempty"`
	State     string   `json:"state"`
	LastError string   `json:"last_error,omitempty"`
	Stdout    []string `json:"stdout_tail,omitempty"`
	Stderr    []string `json:"stderr_tail,omitempty"`
}

func NewSupervisor(cfg SupervisorConfig) (*Supervisor, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.Command = strings.TrimSpace(cfg.Command)
	if cfg.Name == "" {
		return nil, fmt.Errorf("decoder supervisor name is required")
	}
	if cfg.Command == "" {
		return nil, fmt.Errorf("decoder supervisor command is required")
	}
	if cfg.BackoffInitial <= 0 {
		cfg.BackoffInitial = 250 * time.Millisecond
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = 10 * time.Second
	}
	if cfg.StdoutTailLines <= 0 {
		cfg.StdoutTailLines = 50
	}
	if cfg.StderrTailLines <= 0 {
		cfg.StderrTailLines = 200
	}
	if cfg.MaxLineBytes <= 0 {
		cfg.MaxLineBytes = 16 * 1024
	}

	s := &Supervisor{
		cfg:    cfg,
		pid:    0,
		state:  "stopped",
		stdout: newTailBuffer(cfg.StdoutTailLines, cfg.MaxLineBytes),
		stderr: newTailBuffer(cfg.StderrTailLines, cfg.MaxLineBytes),
		done:   make(chan struct{}),
	}
	return s, nil
}

func (s *Supervisor) Start(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("supervisor is nil")
	}
	if s.closed.Load() {
		return fmt.Errorf("supervisor is closed")
	}
	if s.started.Swap(true) {
		return fmt.Errorf("supervisor already started")
	}

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.setState("starting", "")
	go s.runLoop(runCtx)
	return nil
}

func (s *Supervisor) Close() {
	if s == nil {
		return
	}
	if s.closed.Swap(true) {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
}

func (s *Supervisor) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	s.mu.RLock()
	pid := s.pid
	state := s.state
	lastErr := s.lastErr
	s.mu.RUnlock()

	running := pid != 0 && state == "running"
	return Snapshot{
		Name:      s.cfg.Name,
		Running:   running,
		PID:       pid,
		State:     state,
		LastError: lastErr,
		Stdout:    s.stdout.snapshot(),
		Stderr:    s.stderr.snapshot(),
	}
}

func (s *Supervisor) runLoop(ctx context.Context) {
	defer close(s.done)

	backoff := s.cfg.BackoffInitial
	for {
		select {
		case <-ctx.Done():
			s.setState("stopped", "")
			return
		default:
		}

		exitErr := s.runOnce(ctx)
		if ctx.Err() != nil {
			s.setState("stopped", "")
			return
		}

		if exitErr != nil {
			s.setState("exited", exitErr.Error())
		} else {
			s.setState("exited", "")
		}

		if !s.cfg.Restart {
			return
		}

		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			s.setState("stopped", "")
			return
		case <-t.C:
		}
		backoff *= 2
		if backoff > s.cfg.BackoffMax {
			backoff = s.cfg.BackoffMax
		}
		s.setState("restarting", "")
	}
}

func (s *Supervisor) runOnce(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, s.cfg.Command, s.cfg.Args...)
	if s.cfg.WorkDir != "" {
		cmd.Dir = s.cfg.WorkDir
	}
	if len(s.cfg.Env) > 0 {
		cmd.Env = append(cmd.Environ(), envMapToList(s.cfg.Env)...)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	s.mu.Lock()
	s.pid = pid
	s.state = "running"
	s.lastErr = ""
	s.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		readLinesToTail(stdoutPipe, s.stdout)
	}()
	go func() {
		defer wg.Done()
		readLinesToTail(stderrPipe, s.stderr)
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	s.mu.Lock()
	s.pid = 0
	s.mu.Unlock()

	if waitErr == nil {
		return nil
	}
	if errors.Is(waitErr, context.Canceled) {
		return nil
	}
	return waitErr
}

func (s *Supervisor) setState(state string, lastErr string) {
	s.mu.Lock()
	s.state = state
	if strings.TrimSpace(lastErr) != "" {
		s.lastErr = lastErr
	}
	s.mu.Unlock()
}

func envMapToList(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out = append(out, k+"="+v)
	}
	return out
}

func readLinesToTail(r io.Reader, t *tailBuffer) {
	if r == nil || t == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	// Increase the maximum token size to something sane for log lines.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, t.maxLineBytes)

	for scanner.Scan() {
		t.add(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.add("[tail error] " + err.Error())
	}
}
