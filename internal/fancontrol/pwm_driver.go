package fancontrol

// pwmDriver is the minimal interface fancontrol needs from a PWM/GPIO backend.
//
// Implementations are called by the fancontrol service loop.
// Duty is expressed in percent (0..100).
// Frequency semantics follow upstream Stratux conventions (see implementations).
//
// Close should be best-effort and leave the system in a safe state.
//
//nolint:revive // internal interface name matches domain.
type pwmDriver interface {
	SetFrequencyHz(hz int) error
	SetDutyPercent(p float64) error
	Close() error
}
