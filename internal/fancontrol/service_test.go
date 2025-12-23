package fancontrol

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakePWMDriver struct {
	setFreqCalls atomic.Int64
	setDutyCalls atomic.Int64
	lastDuty     atomic.Int64
	dutyCh       chan float64
}

func (d *fakePWMDriver) SetFrequencyHz(hz int) error {
	d.setFreqCalls.Add(1)
	return nil
}

func (d *fakePWMDriver) SetDutyPercent(p float64) error {
	d.setDutyCalls.Add(1)
	d.lastDuty.Store(int64(p))
	select {
	case d.dutyCh <- p:
	default:
	}
	return nil
}

func (d *fakePWMDriver) Close() error { return nil }

func TestServiceStart_IsNonBlocking(t *testing.T) {
	// Make the startup test durations huge to catch accidental blocking.
	oldFull := startupFullDutyDuration
	oldMin := startupMinDutyDuration
	startupFullDutyDuration = time.Hour
	startupMinDutyDuration = time.Hour
	t.Cleanup(func() {
		startupFullDutyDuration = oldFull
		startupMinDutyDuration = oldMin
	})

	fake := &fakePWMDriver{dutyCh: make(chan float64, 8)}
	oldOpen := openPWMFn
	openPWMFn = func(pin int) (pwmDriver, error) { return fake, nil }
	t.Cleanup(func() { openPWMFn = oldOpen })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := New(Config{Enable: true, PWMPin: 18, Backend: "pwm"})

	start := time.Now()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("Start took too long (likely blocked): %v", time.Since(start))
	}

	// Prove the async startup goroutine actually began.
	select {
	case duty := <-fake.dutyCh:
		if duty != 100 {
			t.Fatalf("first duty=%v want 100", duty)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected startup duty set quickly")
	}

	cancel()
	svc.Close()
}

func TestServiceClose_TurnsFanOffOnGracefulShutdown(t *testing.T) {
	// Make the startup test durations huge to ensure we exercise the ctx cancel path.
	oldFull := startupFullDutyDuration
	oldMin := startupMinDutyDuration
	startupFullDutyDuration = time.Hour
	startupMinDutyDuration = time.Hour
	t.Cleanup(func() {
		startupFullDutyDuration = oldFull
		startupMinDutyDuration = oldMin
	})

	fake := &fakePWMDriver{dutyCh: make(chan float64, 16)}
	oldOpen := openPWMFn
	openPWMFn = func(pin int) (pwmDriver, error) { return fake, nil }
	t.Cleanup(func() { openPWMFn = oldOpen })

	ctx, cancel := context.WithCancel(context.Background())

	svc := New(Config{Enable: true, PWMPin: 18, Backend: "pwm"})
	if err := svc.Start(ctx); err != nil {
		cancel()
		t.Fatalf("Start: %v", err)
	}

	// Ensure startup goroutine began.
	select {
	case <-fake.dutyCh:
	case <-time.After(200 * time.Millisecond):
		cancel()
		svc.Close()
		t.Fatalf("expected initial duty set")
	}

	// Graceful shutdown should leave fan OFF.
	cancel()
	svc.Close()
	if got := fake.lastDuty.Load(); got != 0 {
		t.Fatalf("last duty=%d want 0", got)
	}
}
