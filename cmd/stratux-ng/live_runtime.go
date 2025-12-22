package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"stratux-ng/internal/ahrs"
	"stratux-ng/internal/config"
	"stratux-ng/internal/fancontrol"
	"stratux-ng/internal/gps"
	"stratux-ng/internal/udp"
	"stratux-ng/internal/web"
)

type liveRuntime struct {
	resolvedConfigPath string
	status             *web.Status
	sender             *safeBroadcaster
	ahrsSvc            *ahrs.Service
	gpsSvc             *gps.Service
	fanSvc             *fancontrol.Service

	cfg      config.Config
	scenario scenarioRuntime
	ticker   *time.Ticker
}

func newLiveRuntime(ctx context.Context, cfg config.Config, resolvedConfigPath string, status *web.Status, sender *safeBroadcaster) (*liveRuntime, error) {
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

	// Optional: real AHRS bring-up.
	var ahrsSvc *ahrs.Service
	if c.AHRS.Enable {
		var gravity [3]float64
		var gravitySet bool
		if g := c.AHRS.Orientation.GravityInSensor; len(g) == 3 {
			gravity = [3]float64{g[0], g[1], g[2]}
			gravitySet = true
		}
		svc := ahrs.New(ahrs.Config{
			Enable:   c.AHRS.Enable,
			I2CBus:   c.AHRS.I2CBus,
			IMUAddr:  c.AHRS.IMUAddr,
			BaroAddr: c.AHRS.BaroAddr,

			OrientationForwardAxis: c.AHRS.Orientation.ForwardAxis,
			OrientationGravitySet:  gravitySet,
			OrientationGravity:     gravity,
		})
		if err := svc.Start(ctx); err != nil {
			// Keep Stratux-NG running even if AHRS fails to init; AHRS will be marked invalid.
			log.Printf("ahrs init failed: %v", err)
		}
		ahrsSvc = svc
	}

	r := &liveRuntime{
		resolvedConfigPath: resolvedConfigPath,
		status:             status,
		sender:             sender,
		cfg:                c,
		scenario:           sc,
		ticker:             t,
		ahrsSvc:            ahrsSvc,
	}

	// Optional: real GPS bring-up (USB serial NMEA).
	if c.GPS.Enable {
		svc := gps.New(gps.Config{
			Enable:    c.GPS.Enable,
			Source:    c.GPS.Source,
			GPSDAddr:  c.GPS.GPSDAddr,
			Device:    c.GPS.Device,
			Baud:      c.GPS.Baud,
		})
		if err := svc.Start(ctx); err != nil {
			// Keep Stratux-NG running even if GPS fails to init.
			log.Printf("gps init failed: %v", err)
		}
		r.gpsSvc = svc
	}

	// Optional: fan control.
	if c.Fan.Enable {
		svc := fancontrol.New(fancontrol.Config{
			Enable:         c.Fan.Enable,
			PWMPin:         c.Fan.PWMPin,
			PWMFrequency:   c.Fan.PWMFrequency,
			TempTargetC:    c.Fan.TempTargetC,
			PWMDutyMin:     c.Fan.PWMDutyMin,
			UpdateInterval: c.Fan.UpdateInterval,
		})
		// Keep a reference even if init fails so status can report errors.
		r.fanSvc = svc
		if err := svc.Start(ctx); err != nil {
			// Keep Stratux-NG running even if fan control fails to init.
			log.Printf("fancontrol init failed: %v", err)
		}
	}

	return r, nil
}

func (r *liveRuntime) Close() {
	if r == nil {
		return
	}
	if r.ahrsSvc != nil {
		r.ahrsSvc.Close()
		r.ahrsSvc = nil
	}
	if r.gpsSvc != nil {
		r.gpsSvc.Close()
		r.gpsSvc = nil
	}
	if r.fanSvc != nil {
		r.fanSvc.Close()
		r.fanSvc = nil
	}
	if r.ticker != nil {
		r.ticker.Stop()
		r.ticker = nil
	}
}

func (r *liveRuntime) FanSnapshot() (fancontrol.Snapshot, bool) {
	if r == nil || r.fanSvc == nil {
		return fancontrol.Snapshot{}, false
	}
	return r.fanSvc.Snapshot(), true
}

func (r *liveRuntime) AHRSSnapshot() (ahrs.Snapshot, bool) {
	if r == nil || r.ahrsSvc == nil {
		return ahrs.Snapshot{}, false
	}
	return r.ahrsSvc.Snapshot(), true
}

func (r *liveRuntime) GPSSnapshot() (gps.Snapshot, bool) {
	if r == nil || r.gpsSvc == nil {
		return gps.Snapshot{}, false
	}
	return r.gpsSvc.Snapshot(), true
}

func (r *liveRuntime) AHRSSetLevel() error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.SetLevel()
}

func (r *liveRuntime) AHRSZeroDrift(ctx context.Context) error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.ZeroDrift(ctx)
}

func (r *liveRuntime) AHRSOrientForward(ctx context.Context) error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.OrientForward(ctx)
}

func (r *liveRuntime) AHRSOrientDone(ctx context.Context) error {
	if r == nil || r.ahrsSvc == nil {
		return fmt.Errorf("ahrs unavailable")
	}
	return r.ahrsSvc.OrientDone(ctx)
}

func (r *liveRuntime) AHRSOrientation() (forwardAxis int, gravity [3]float64, gravityOK bool) {
	if r == nil || r.ahrsSvc == nil {
		return 0, [3]float64{}, false
	}
	return r.ahrsSvc.Orientation()
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
	if c.AHRS.Enable != r.cfg.AHRS.Enable || c.AHRS.I2CBus != r.cfg.AHRS.I2CBus || c.AHRS.IMUAddr != r.cfg.AHRS.IMUAddr || c.AHRS.BaroAddr != r.cfg.AHRS.BaroAddr {
		return fmt.Errorf("ahrs settings require restart")
	}
	if c.GPS.Enable != r.cfg.GPS.Enable || strings.TrimSpace(c.GPS.Device) != strings.TrimSpace(r.cfg.GPS.Device) || c.GPS.Baud != r.cfg.GPS.Baud {
		return fmt.Errorf("gps settings require restart")
	}
	if c.Fan.Enable != r.cfg.Fan.Enable || c.Fan.PWMPin != r.cfg.Fan.PWMPin || c.Fan.PWMFrequency != r.cfg.Fan.PWMFrequency || c.Fan.TempTargetC != r.cfg.Fan.TempTargetC || c.Fan.PWMDutyMin != r.cfg.Fan.PWMDutyMin || c.Fan.UpdateInterval != r.cfg.Fan.UpdateInterval {
		return fmt.Errorf("fan settings require restart")
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
	r.status.SetStatic(r.cfg.GDL90.Dest, r.cfg.GDL90.Interval.String(), simInfoSnapshot(r.resolvedConfigPath, r.cfg))
	return nil
}
