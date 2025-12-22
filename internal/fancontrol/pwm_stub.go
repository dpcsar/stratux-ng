//go:build !linux || (!arm && !arm64)

package fancontrol

import "fmt"

type pwmDriver interface {
	SetFrequencyHz(hz int) error
	SetDutyPercent(p float64) error
	Close() error
}

type unsupportedPWM struct{}

func openPWM(pin int) (pwmDriver, error) {
	return nil, fmt.Errorf("fancontrol: pwm unsupported on this platform")
}

func (u *unsupportedPWM) SetFrequencyHz(hz int) error {
	return fmt.Errorf("fancontrol: pwm unsupported")
}
func (u *unsupportedPWM) SetDutyPercent(p float64) error {
	return fmt.Errorf("fancontrol: pwm unsupported")
}
func (u *unsupportedPWM) Close() error { return nil }
