package fancontrol

import "time"

// pidController is a tiny PID controller modeled after the upstream Stratux fancontrol.
//
// It outputs a signed control value in the configured range.
// We keep this self-contained to avoid extra dependencies.
//
// Not safe for concurrent use.
type pidController struct {
	kp, ki, kd float64
	setpoint   float64
	outMin     float64
	outMax     float64

	integral  float64
	prevError float64
	prevAt    time.Time
	havePrev  bool
}

func newPID(kp, ki, kd float64) *pidController {
	return &pidController{kp: kp, ki: ki, kd: kd, outMin: -100, outMax: 0}
}

func (p *pidController) SetOutputLimits(min, max float64) {
	p.outMin = min
	p.outMax = max
}

func (p *pidController) Set(setpoint float64) {
	p.setpoint = setpoint
	p.integral = 0
	p.prevError = 0
	p.havePrev = false
	p.prevAt = time.Time{}
}

func (p *pidController) UpdateDuration(measurement float64, dt time.Duration) float64 {
	if dt <= 0 {
		// Keep behavior deterministic: no time => no update.
		return 0
	}
	sec := dt.Seconds()
	// error = setpoint - measurement (classic)
	err := p.setpoint - measurement
	p.integral += err * sec

	derivative := 0.0
	if p.havePrev {
		derivative = (err - p.prevError) / sec
	}
	p.prevError = err
	p.havePrev = true

	out := p.kp*err + p.ki*p.integral + p.kd*derivative
	if out < p.outMin {
		out = p.outMin
	}
	if out > p.outMax {
		out = p.outMax
	}
	return out
}
