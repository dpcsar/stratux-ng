package fancontrol

import (
	"testing"
	"time"
)

func TestPID_UpdateDuration_ZeroDT(t *testing.T) {
	p := newPID(0.2, 0.2, 0.1)
	p.SetOutputLimits(-100, 0)
	p.Set(50)

	out := p.UpdateDuration(60, 0)
	if out != 0 {
		t.Fatalf("out=%v want 0", out)
	}
}

func TestPID_ClampsToLimits(t *testing.T) {
	p := newPID(10, 0, 0)
	p.SetOutputLimits(-5, 0)
	p.Set(50)

	// Large negative error => large negative output, should clamp at -5.
	out := p.UpdateDuration(100, 1*time.Second)
	if out != -5 {
		t.Fatalf("out=%v want -5", out)
	}

	// Large positive error => large positive output, should clamp at 0.
	out = p.UpdateDuration(-100, 1*time.Second)
	if out != 0 {
		t.Fatalf("out=%v want 0", out)
	}
}

func TestPID_SignConvention(t *testing.T) {
	p := newPID(0.2, 0, 0)
	p.SetOutputLimits(-100, 0)
	p.Set(50)

	// measurement above setpoint => error negative => output negative.
	out := p.UpdateDuration(60, 1*time.Second)
	if out >= 0 {
		t.Fatalf("out=%v want negative", out)
	}
}
