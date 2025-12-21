package main

import (
	"fmt"
	"strings"
	"time"

	"stratux-ng/internal/config"
	"stratux-ng/internal/udp"
	"stratux-ng/internal/web"
)

type liveRuntime struct {
	resolvedConfigPath string
	status             *web.Status
	sender             *safeBroadcaster

	cfg      config.Config
	scenario scenarioRuntime
	ticker   *time.Ticker
}

func newLiveRuntime(cfg config.Config, resolvedConfigPath string, status *web.Status, sender *safeBroadcaster) (*liveRuntime, error) {
	c := cfg
	if err := config.DefaultAndValidate(&c); err != nil {
		return nil, err
	}
	if status == nil {
		return nil, fmt.Errorf("status is nil")
	}
	if sender == nil {
		return nil, fmt.Errorf("sender is nil")
	}

	sc, err := loadScenarioFromConfig(c)
	if err != nil {
		return nil, err
	}

	var t *time.Ticker
	if !c.GDL90.Replay.Enable {
		t = time.NewTicker(c.GDL90.Interval)
	}

	r := &liveRuntime{
		resolvedConfigPath: resolvedConfigPath,
		status:             status,
		sender:             sender,
		cfg:                c,
		scenario:           sc,
		ticker:             t,
	}
	return r, nil
}

func (r *liveRuntime) Close() {
	if r == nil {
		return
	}
	if r.ticker != nil {
		r.ticker.Stop()
		r.ticker = nil
	}
}

func (r *liveRuntime) TickChan() <-chan time.Time {
	if r == nil || r.ticker == nil {
		return nil
	}
	return r.ticker.C
}

func (r *liveRuntime) Config() config.Config {
	if r == nil {
		return config.Config{}
	}
	return r.cfg
}

func (r *liveRuntime) Scenario() *scenarioRuntime {
	if r == nil {
		return nil
	}
	return &r.scenario
}

func (r *liveRuntime) Apply(next config.Config) error {
	if r == nil {
		return fmt.Errorf("runtime is nil")
	}

	c := next
	if err := config.DefaultAndValidate(&c); err != nil {
		return err
	}

	// Keep live scope intentionally small/safe.
	if c.GDL90.Record.Enable != r.cfg.GDL90.Record.Enable || c.GDL90.Record.Path != r.cfg.GDL90.Record.Path {
		return fmt.Errorf("gdl90.record settings require restart")
	}
	if c.GDL90.Replay.Enable != r.cfg.GDL90.Replay.Enable || c.GDL90.Replay.Path != r.cfg.GDL90.Replay.Path || c.GDL90.Replay.Speed != r.cfg.GDL90.Replay.Speed || c.GDL90.Replay.Loop != r.cfg.GDL90.Replay.Loop {
		return fmt.Errorf("gdl90.replay settings require restart")
	}
	if c.Web.Listen != r.cfg.Web.Listen {
		return fmt.Errorf("web.listen requires restart")
	}

	// Pre-validate side effects before committing anything.
	var nextBroadcaster *udp.Broadcaster
	if strings.TrimSpace(c.GDL90.Dest) != strings.TrimSpace(r.cfg.GDL90.Dest) {
		b, err := udp.NewBroadcaster(c.GDL90.Dest)
		if err != nil {
			return fmt.Errorf("udp broadcaster init failed: %w", err)
		}
		nextBroadcaster = b
	}

	nextScenario, err := loadScenarioFromConfig(c)
	if err != nil {
		if nextBroadcaster != nil {
			_ = nextBroadcaster.Close()
		}
		return err
	}

	// Commit: swap broadcaster.
	if nextBroadcaster != nil {
		r.sender.Swap(nextBroadcaster)
	}

	// Commit: swap ticker.
	if r.ticker != nil && c.GDL90.Interval != r.cfg.GDL90.Interval {
		r.ticker.Stop()
		r.ticker = time.NewTicker(c.GDL90.Interval)
	}

	// Commit: scenario runtime (reset elapsed on any scenario config change).
	if c.Sim.Scenario.Enable != r.cfg.Sim.Scenario.Enable ||
		c.Sim.Scenario.Path != r.cfg.Sim.Scenario.Path ||
		c.Sim.Scenario.StartTimeUTC != r.cfg.Sim.Scenario.StartTimeUTC ||
		c.Sim.Scenario.Loop != r.cfg.Sim.Scenario.Loop {
		r.scenario = nextScenario
		r.scenario.elapsed = 0
	}

	r.cfg = c
	r.status.SetStatic(r.cfg.GDL90.Mode, r.cfg.GDL90.Dest, r.cfg.GDL90.Interval.String(), simInfoSnapshot(r.resolvedConfigPath, r.cfg))
	return nil
}
