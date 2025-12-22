package fancontrol

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

var openPWMFn = openPWM
var afterFn = time.After

var startupFullDutyDuration = 5 * time.Second
var startupMinDutyDuration = 10 * time.Second

type Config struct {
	Enable bool

	// PWMPin is BCM GPIO numbering (matches upstream Stratux).
	PWMPin int
	// PWMFrequency is the configured base frequency; upstream uses 64000.
	// Upstream multiplies this by 100 before calling into rpio.
	PWMFrequency int
	// TempTargetC is the CPU temperature target in degrees C.
	TempTargetC float64
	// PWMDutyMin is minimum duty (0-100) to keep the fan spinning.
	PWMDutyMin int
	// UpdateInterval controls how often duty is recomputed.
	UpdateInterval time.Duration
}

type Snapshot struct {
	Enabled bool `json:"enabled"`

	CPUValid bool    `json:"cpu_valid"`
	CPUTempC float64 `json:"cpu_temp_c"`

	PWMAvailable bool `json:"pwm_available"`
	PWMDuty      int  `json:"pwm_duty"`

	LastUpdateAt time.Time `json:"last_update_utc,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
}

type Service struct {
	cfg Config

	mu   sync.RWMutex
	snap Snapshot

	drvMu sync.Mutex
	drv   pwmDriver

	wg sync.WaitGroup

	stopOnce sync.Once
	stopCh   chan struct{}
}

func New(cfg Config) *Service {
	if cfg.PWMPin == 0 {
		cfg.PWMPin = 18
	}
	if cfg.PWMFrequency == 0 {
		cfg.PWMFrequency = 64000
	}
	if cfg.TempTargetC == 0 {
		cfg.TempTargetC = 50.0
	}
	if cfg.UpdateInterval <= 0 {
		cfg.UpdateInterval = 5 * time.Second
	}

	// Note: upstream Stratux fancontrol uses alwaysOn=true and always runs a short
	// startup fan test. We mirror that behavior here.

	return &Service{cfg: cfg, stopCh: make(chan struct{})}
}

func (s *Service) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *Service) Close() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})

	// Ensure the PWM driver is not used concurrently with Close.
	s.wg.Wait()

	s.drvMu.Lock()
	drv := s.drv
	s.drvMu.Unlock()
	if drv != nil {
		_ = drv.Close()
	}
}

func (s *Service) setErr(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.LastError = msg
	s.snap.LastUpdateAt = time.Now().UTC()
}

func (s *Service) setState(update func(*Snapshot)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	update(&s.snap)
	s.snap.LastUpdateAt = time.Now().UTC()
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (s *Service) Start(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("fancontrol: service is nil")
	}
	if !s.cfg.Enable {
		return nil
	}

	s.setState(func(sn *Snapshot) {
		sn.Enabled = true
	})

	drv, err := openPWMFn(s.cfg.PWMPin)
	if err != nil {
		s.setErr(err.Error())
		return err
	}
	s.drvMu.Lock()
	s.drv = drv
	s.drvMu.Unlock()

	// Upstream multiplies by 100 before calling into rpio pin.Freq().
	if err := drv.SetFrequencyHz(s.cfg.PWMFrequency * 100); err != nil {
		s.setErr(fmt.Sprintf("fancontrol: set pwm frequency failed: %v", err))
		_ = drv.Close()
		s.drvMu.Lock()
		if s.drv == drv {
			s.drv = nil
		}
		s.drvMu.Unlock()
		return err
	}

	s.setState(func(sn *Snapshot) {
		sn.PWMAvailable = true
	})

	// Upstream fancontrol runs as a separate component and does not block the
	// rest of Stratux. Mirror that behavior: run startup test + control loop
	// asynchronously.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.startupAndRun(ctx, drv)
	}()

	// Ensure resources are released if the runtime context is canceled.
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	return nil
}

func (s *Service) startupAndRun(ctx context.Context, drv pwmDriver) {
	defer func() {
		// Safe failover: if we ever bail out unexpectedly, try to force fan ON.
		if drv != nil {
			_ = drv.SetDutyPercent(100)
			s.setState(func(sn *Snapshot) { sn.PWMDuty = 100 })
		}
	}()

	// Startup test (mirrors upstream):
	// - 5s full duty
	// - 10s min duty
	if err := drv.SetDutyPercent(100); err != nil {
		s.setErr(fmt.Sprintf("fancontrol: set pwm duty failed: %v", err))
		return
	}
	s.setState(func(sn *Snapshot) { sn.PWMDuty = 100 })
	select {
	case <-afterFn(startupFullDutyDuration):
	case <-ctx.Done():
		return
	case <-s.stopCh:
		return
	}
	minDuty := float64(clamp(float64(s.cfg.PWMDutyMin), 0, 100))
	if err := drv.SetDutyPercent(minDuty); err != nil {
		s.setErr(fmt.Sprintf("fancontrol: set pwm duty failed: %v", err))
		return
	}
	s.setState(func(sn *Snapshot) { sn.PWMDuty = int(math.Round(minDuty)) })
	select {
	case <-afterFn(startupMinDutyDuration):
	case <-ctx.Done():
		return
	case <-s.stopCh:
		return
	}

	s.runLoop(ctx, drv)
}

func (s *Service) runLoop(ctx context.Context, drv pwmDriver) {
	pid := newPID(0.2, 0.2, 0.1)
	pid.SetOutputLimits(-100, 0)
	pid.Set(s.cfg.TempTargetC)

	t := time.NewTicker(s.cfg.UpdateInterval)
	defer t.Stop()

	var lastPWM float64
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-t.C:
			cpuC, err := ReadCPUTempC()
			if err != nil {
				s.setState(func(sn *Snapshot) {
					sn.CPUValid = false
					sn.LastError = err.Error()
				})
				// Fail-safe: keep fan full on if we cannot read temperature.
				if err := drv.SetDutyPercent(100); err != nil {
					s.setState(func(sn *Snapshot) {
						sn.LastError = fmt.Sprintf("fancontrol: set pwm duty failed: %v", err)
					})
					continue
				}
				s.setState(func(sn *Snapshot) { sn.PWMDuty = 100 })
				continue
			}

			pidOut := -pid.UpdateDuration(cpuC, s.cfg.UpdateInterval)
			// Upstream has a small deadband; keep a similar behavior.
			var duty float64
			if pidOut > 5.0 || lastPWM != 0.0 {
				lastPWM = pidOut
				duty = pidOut
			} else {
				lastPWM = 0
				// Upstream behavior: alwaysOn=true.
				duty = 1
			}

			// Map duty into [PWMDutyMin..100].
			mappedMin := clamp(float64(s.cfg.PWMDutyMin), 0, 100)
			duty = clamp(duty, 0, 100)
			if duty > 0 {
				duty = mappedMin + (duty*(100.0-mappedMin))/100.0
			}
			duty = clamp(duty, 0, 100)

			if err := drv.SetDutyPercent(duty); err != nil {
				s.setState(func(sn *Snapshot) {
					sn.LastError = fmt.Sprintf("fancontrol: set pwm duty failed: %v", err)
				})
				continue
			}
			s.setState(func(sn *Snapshot) {
				sn.CPUValid = true
				sn.CPUTempC = cpuC
				sn.PWMDuty = int(math.Round(duty))
				sn.LastError = ""
			})
		}
	}
}
