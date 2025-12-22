package ahrs

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"stratux-ng/internal/i2c"
	"stratux-ng/internal/sensors/bmp280"
	"stratux-ng/internal/sensors/icm20948"
)

type Config struct {
	Enable   bool
	I2CBus   int
	IMUAddr  uint16
	BaroAddr uint16

	OrientationForwardAxis int
	OrientationGravitySet  bool
	OrientationGravity     [3]float64
}

type Snapshot struct {
	Valid            bool
	IMUDetected      bool
	BaroDetected     bool
	IMULastUpdateAt  time.Time
	BaroLastUpdateAt time.Time

	OrientationSet         bool
	OrientationForwardAxis int

	RollDeg  float64
	PitchDeg float64

	PressureAltFeet    float64
	PressureAltValid   bool
	VerticalSpeedFpm   int
	VerticalSpeedValid bool

	LastError string
	UpdatedAt time.Time
}

type Service struct {
	cfg Config

	imuErr  string
	baroErr string

	rollOffsetDeg  float64
	pitchOffsetDeg float64

	gyroBiasXDegPerSec float64
	gyroBiasYDegPerSec float64
	gyroBiasZDegPerSec float64

	zeroDriftCh chan chan error
	orientCh    chan orientReq

	startupOnce sync.Once

	// Orientation: maps sensor-frame vectors into a body frame suitable for roll/pitch.
	// This mirrors Stratux's "set forward" + "done" concept.
	// bodyXInSensor, bodyYInSensor, bodyZInSensor are unit vectors expressed in sensor coordinates.
	orientationSet  bool
	forwardAxis     int // +/-1..+/-3 (sensor axis index & sign), like Stratux
	gravityInSensor [3]float64
	bodyXInSensor   [3]float64
	bodyYInSensor   [3]float64
	bodyZInSensor   [3]float64

	mu   sync.RWMutex
	snap Snapshot

	bus  *i2c.Bus
	imu  *icm20948.Device
	baro *bmp280.Device

	stopOnce sync.Once
	stopCh   chan struct{}
}

func New(cfg Config) *Service {
	if cfg.I2CBus == 0 {
		cfg.I2CBus = 1
	}
	if cfg.IMUAddr == 0 {
		cfg.IMUAddr = icm20948.DefaultAddress()
	}
	if cfg.BaroAddr == 0 {
		cfg.BaroAddr = bmp280.DefaultAddress()
	}
	s := &Service{cfg: cfg, stopCh: make(chan struct{}), zeroDriftCh: make(chan chan error, 1), orientCh: make(chan orientReq, 1)}
	// Default orientation: identity.
	s.bodyXInSensor = [3]float64{1, 0, 0}
	s.bodyYInSensor = [3]float64{0, 1, 0}
	s.bodyZInSensor = [3]float64{0, 0, 1}
	return s
}

func (s *Service) Close() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		if s.bus != nil {
			_ = s.bus.Close()
			s.bus = nil
		}
	})
}

func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *Service) Start(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("ahrs: service is nil")
	}
	if !s.cfg.Enable {
		return nil
	}

	busPath := fmt.Sprintf("/dev/i2c-%d", s.cfg.I2CBus)
	bus, err := i2c.Open(busPath)
	if err != nil {
		s.setIMUErr(fmt.Sprintf("open %s: %v", busPath, err))
		return err
	}
	s.bus = bus

	imu, err := icm20948.New(bus.Dev(s.cfg.IMUAddr))
	if err != nil {
		s.setIMUErr(fmt.Sprintf("imu init: %v", err))
		_ = bus.Close()
		s.bus = nil
		return err
	}
	s.imu = imu
	// Mark IMU present and load persisted forward axis (if any).
	s.mu.Lock()
	s.snap.IMUDetected = true
	if s.cfg.OrientationForwardAxis != 0 {
		s.forwardAxis = s.cfg.OrientationForwardAxis
		s.snap.OrientationForwardAxis = s.forwardAxis
	}
	s.mu.Unlock()
	// Load persisted orientation (if any). Best-effort: do not fail service start on bad persisted values.
	if s.cfg.OrientationForwardAxis != 0 && s.cfg.OrientationGravitySet {
		_ = s.applyOrientationFromGravity([3]float64{s.cfg.OrientationGravity[0], s.cfg.OrientationGravity[1], s.cfg.OrientationGravity[2]})
	}

	baro, err := bmp280.New(bus.Dev(s.cfg.BaroAddr))
	if err != nil {
		s.setBaroErr(fmt.Sprintf("baro init: %v", err))
		_ = bus.Close()
		s.bus = nil
		return err
	}
	s.baro = baro
	s.mu.Lock()
	s.snap.BaroDetected = true
	s.mu.Unlock()

	go s.run(ctx)
	go s.startupCal(ctx)
	return nil
}

// Orientation returns the current persisted-or-pending orientation state.
// gravityOK indicates whether gravity is available for persistence.
func (s *Service) Orientation() (forwardAxis int, gravity [3]float64, gravityOK bool) {
	if s == nil {
		return 0, [3]float64{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	forwardAxis = s.forwardAxis
	if s.orientationSet {
		return forwardAxis, s.gravityInSensor, true
	}
	return forwardAxis, [3]float64{}, false
}

// OrientForward mirrors Stratux's "Set Forward Direction" step.
// User points the sensor so the end that will face the airplane nose points up (toward the sky),
// then we detect which accelerometer axis is most aligned with gravity.
func (s *Service) OrientForward(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("ahrs: service is nil")
	}
	if ctx == nil {
		return fmt.Errorf("ahrs: ctx is nil")
	}
	done := make(chan error, 1)
	select {
	case s.orientCh <- orientReq{action: orientActionForward, done: done}:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("ahrs: orientation already in progress")
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// OrientDone mirrors Stratux's "Done" step.
// User places the sensor in the mounted in-flight orientation (level) and keeps it stationary.
// We capture the gravity vector in that pose and build a stable sensor->body rotation.
func (s *Service) OrientDone(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("ahrs: service is nil")
	}
	if ctx == nil {
		return fmt.Errorf("ahrs: ctx is nil")
	}
	done := make(chan error, 1)
	select {
	case s.orientCh <- orientReq{action: orientActionDone, done: done}:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("ahrs: orientation already in progress")
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetLevel re-zeros roll/pitch so the current attitude becomes (0,0).
// This mirrors Stratux's "cage/level" style control.
// The offset is not persisted; it lives for the process lifetime.
func (s *Service) SetLevel() error {
	if s == nil {
		return fmt.Errorf("ahrs: service is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.snap.Valid {
		return fmt.Errorf("ahrs: not valid (%s)", s.snap.LastError)
	}
	// Adjust offsets so current output becomes zero.
	s.rollOffsetDeg -= s.snap.RollDeg
	s.pitchOffsetDeg -= s.snap.PitchDeg
	return nil
}

// ZeroDrift estimates stationary gyro bias over ~2 seconds and subtracts it.
// This is only meaningful when gyro integration is used.
func (s *Service) ZeroDrift(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("ahrs: service is nil")
	}
	if ctx == nil {
		return fmt.Errorf("ahrs: ctx is nil")
	}
	// Require IMU presence.
	s.mu.RLock()
	imuDetected := s.snap.IMUDetected
	s.mu.RUnlock()
	if !imuDetected {
		return fmt.Errorf("ahrs: imu not detected")
	}

	done := make(chan error, 1)
	select {
	case s.zeroDriftCh <- done:
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("ahrs: zero drift already in progress")
	}

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) run(ctx context.Context) {
	imuTick := time.NewTicker(20 * time.Millisecond)   // 50 Hz
	baroTick := time.NewTicker(200 * time.Millisecond) // 5 Hz
	defer imuTick.Stop()
	defer baroTick.Stop()

	// Complementary filter state (radians).
	var haveEst bool
	var estRollRad, estPitchRad float64
	var lastIMUAt time.Time

	// Zero drift calibration state.
	var calActive bool
	var calDone chan error
	var calStart time.Time
	var calSumX, calSumY, calSumZ float64
	var calN int

	// Orientation state.
	var lastSample icm20948.Sample
	var haveLastSample bool
	var orientActive bool
	var orientAction orientAction
	var orientDone chan error
	var orientStart time.Time
	var orientSum [3]float64
	var orientN int

	var lastBaroAltFeet float64
	var lastBaroAt time.Time
	var vsFpm float64
	var baroConsecutiveFailures int
	var baroLastReinitAt time.Time

	for {
		select {
		case <-ctx.Done():
			s.Close()
			return
		case <-s.stopCh:
			return
		case done := <-s.zeroDriftCh:
			// Start a new calibration window.
			calActive = true
			calDone = done
			calStart = time.Now().UTC()
			calSumX, calSumY, calSumZ = 0, 0, 0
			calN = 0
		case req := <-s.orientCh:
			// Start an orientation request (handled using current/next IMU samples).
			if req.done == nil {
				continue
			}
			if orientActive {
				req.done <- fmt.Errorf("ahrs: orientation already active")
				continue
			}
			if req.action == orientActionForward {
				if !haveLastSample {
					req.done <- fmt.Errorf("ahrs: no imu samples yet")
					continue
				}
				f := dominantAxis(lastSample.Ax, lastSample.Ay, lastSample.Az)
				s.mu.Lock()
				s.forwardAxis = f
				s.orientationSet = false
				s.gravityInSensor = [3]float64{0, 0, 0}
				s.snap.OrientationForwardAxis = f
				s.snap.OrientationSet = false
				s.mu.Unlock()
				req.done <- nil
				continue
			}
			if req.action == orientActionDone {
				s.mu.RLock()
				f := s.forwardAxis
				s.mu.RUnlock()
				if f == 0 {
					req.done <- fmt.Errorf("ahrs: forward direction not set")
					continue
				}
				orientActive = true
				orientAction = req.action
				orientDone = req.done
				orientStart = time.Now().UTC()
				orientSum = [3]float64{0, 0, 0}
				orientN = 0
				continue
			}
			req.done <- fmt.Errorf("ahrs: unknown orientation action")
		case <-imuTick.C:
			sample, err := s.imu.Read()
			if err != nil {
				s.setIMUErr(err.Error())
				continue
			}
			lastSample = sample
			haveLastSample = true

			now := time.Now().UTC()
			dt := 0.0
			if !lastIMUAt.IsZero() {
				dt = now.Sub(lastIMUAt).Seconds()
			}
			lastIMUAt = now
			if dt <= 0 || dt > 0.5 {
				dt = 0
			}

			// Map sensor vectors into body frame if an orientation has been set.
			ax, ay, az := sample.Ax, sample.Ay, sample.Az
			gx, gy, gz := sample.Gx, sample.Gy, sample.Gz
			s.mu.RLock()
			orientSet := s.orientationSet
			xb := s.bodyXInSensor
			yb := s.bodyYInSensor
			zb := s.bodyZInSensor
			s.mu.RUnlock()
			if orientSet {
				ax, ay, az = dot3(ax, ay, az, xb), dot3(ax, ay, az, yb), dot3(ax, ay, az, zb)
				gx, gy, gz = dot3(gx, gy, gz, xb), dot3(gx, gy, gz, yb), dot3(gx, gy, gz, zb)
			}

			// Compute roll/pitch from accel only (gravity vector).
			accRollRad := math.Atan2(ay, az)
			accPitchRad := math.Atan2(-ax, math.Sqrt(ay*ay+az*az))

			// Integrate gyro (deg/s) -> rad.
			gxRad := gx * math.Pi / 180.0
			gyRad := gy * math.Pi / 180.0
			// gz is currently unused for roll/pitch.
			// Apply bias in deg/s (stored) converted to rad/s.
			s.mu.RLock()
			biasX := s.gyroBiasXDegPerSec * math.Pi / 180.0
			biasY := s.gyroBiasYDegPerSec * math.Pi / 180.0
			s.mu.RUnlock()
			gxRad -= biasX
			gyRad -= biasY

			if !haveEst {
				estRollRad = accRollRad
				estPitchRad = accPitchRad
				haveEst = true
			} else if dt > 0 {
				estRollRad += gxRad * dt
				estPitchRad += gyRad * dt
			}

			// Complementary filter blend.
			if haveEst {
				tau := 0.5 // seconds
				alpha := 0.0
				if dt > 0 {
					alpha = tau / (tau + dt)
				}
				// If dt is unknown (startup), just use accel.
				if alpha <= 0 || alpha >= 1 {
					estRollRad = accRollRad
					estPitchRad = accPitchRad
				} else {
					estRollRad = alpha*estRollRad + (1-alpha)*accRollRad
					estPitchRad = alpha*estPitchRad + (1-alpha)*accPitchRad
				}
			}

			roll := estRollRad * 180 / math.Pi
			pitch := estPitchRad * 180 / math.Pi

			// Update zero-drift calibration if requested.
			if calActive {
				calSumX += gx
				calSumY += gy
				calSumZ += gz
				calN++
				if now.Sub(calStart) >= 2*time.Second {
					if calN <= 0 {
						calDone <- fmt.Errorf("ahrs: zero drift failed (no samples)")
					} else {
						bx := calSumX / float64(calN)
						by := calSumY / float64(calN)
						bz := calSumZ / float64(calN)
						s.mu.Lock()
						s.gyroBiasXDegPerSec = bx
						s.gyroBiasYDegPerSec = by
						s.gyroBiasZDegPerSec = bz
						s.mu.Unlock()
						calDone <- nil
					}
					calActive = false
					calDone = nil
				}
			}

			// Update orientation "done" capture if active.
			if orientActive && orientAction == orientActionDone {
				orientSum[0] += ax
				orientSum[1] += ay
				orientSum[2] += az
				orientN++
				if now.Sub(orientStart) >= 1*time.Second {
					avg := [3]float64{orientSum[0] / float64(orientN), orientSum[1] / float64(orientN), orientSum[2] / float64(orientN)}
					err := s.applyOrientationFromGravity(avg)
					orientDone <- err
					orientActive = false
					orientDone = nil
				}
			}

			s.mu.Lock()
			s.snap.Valid = true
			s.snap.RollDeg = roll + s.rollOffsetDeg
			s.snap.PitchDeg = pitch + s.pitchOffsetDeg
			s.snap.UpdatedAt = now
			s.snap.IMULastUpdateAt = now
			s.snap.OrientationForwardAxis = s.forwardAxis
			s.snap.OrientationSet = s.orientationSet
			// Clear IMU error on success, but keep baro errors visible.
			s.imuErr = ""
			if s.baroErr == "" {
				s.snap.LastError = ""
			}
			s.mu.Unlock()

		case <-baroTick.C:
			tc, p, err := s.baro.Read()
			_ = tc
			if err != nil {
				baroConsecutiveFailures++
				s.setBaroErr(err.Error())
				// Best-effort recovery: periodically re-init the baro if we keep failing.
				if baroConsecutiveFailures >= 10 && time.Since(baroLastReinitAt) >= 2*time.Second {
					if s.bus != nil {
						if b, reErr := bmp280.New(s.bus.Dev(s.cfg.BaroAddr)); reErr == nil {
							s.baro = b
							baroConsecutiveFailures = 0
							baroLastReinitAt = time.Now().UTC()
						} else {
							baroLastReinitAt = time.Now().UTC()
							s.setBaroErr(fmt.Sprintf("baro reinit: %v", reErr))
						}
					}
				}
				continue
			}
			if p <= 0 {
				baroConsecutiveFailures++
				s.setBaroErr("baro pressure invalid")
				if baroConsecutiveFailures >= 10 && time.Since(baroLastReinitAt) >= 2*time.Second {
					if s.bus != nil {
						if b, reErr := bmp280.New(s.bus.Dev(s.cfg.BaroAddr)); reErr == nil {
							s.baro = b
							baroConsecutiveFailures = 0
							baroLastReinitAt = time.Now().UTC()
						} else {
							baroLastReinitAt = time.Now().UTC()
							s.setBaroErr(fmt.Sprintf("baro reinit: %v", reErr))
						}
					}
				}
				continue
			}
			baroConsecutiveFailures = 0

			altFeet := pressureToAltitudeFeet(p)
			now := time.Now().UTC()
			if !lastBaroAt.IsZero() {
				dt := now.Sub(lastBaroAt).Seconds()
				if dt > 0 {
					rawVs := (altFeet - lastBaroAltFeet) / dt * 60.0
					// Simple low-pass to reduce noise.
					alpha := 0.2
					vsFpm = (1-alpha)*vsFpm + alpha*rawVs
				}
			}
			lastBaroAt = now
			lastBaroAltFeet = altFeet

			s.mu.Lock()
			s.snap.PressureAltFeet = altFeet
			s.snap.PressureAltValid = true
			s.snap.VerticalSpeedFpm = int(math.Round(vsFpm))
			s.snap.VerticalSpeedValid = true
			s.snap.UpdatedAt = now
			s.snap.BaroLastUpdateAt = now
			// Clear baro error on success, but keep IMU errors visible.
			s.baroErr = ""
			if s.imuErr == "" {
				s.snap.LastError = ""
			}
			s.mu.Unlock()
		}
	}
}

func (s *Service) setIMUErr(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.imuErr = msg
	// Maintain LastError as the most recent/current error across IMU+baro.
	s.snap.LastError = "imu: " + msg
	s.snap.Valid = false
	s.snap.UpdatedAt = now
}

func (s *Service) setBaroErr(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.baroErr = msg
	// Keep IMU attitude valid even if the baro is misbehaving.
	s.snap.PressureAltValid = false
	s.snap.VerticalSpeedValid = false
	// Maintain LastError as the most recent/current error across IMU+baro.
	s.snap.LastError = "baro: " + msg
	s.snap.UpdatedAt = now
}

type orientAction int

const (
	orientActionForward orientAction = iota
	orientActionDone
)

type orientReq struct {
	action orientAction
	done   chan error
}

func dominantAxis(ax, ay, az float64) int {
	// Return +/-1..+/-3 based on the accel component with max absolute value.
	a1 := math.Abs(ax)
	a2 := math.Abs(ay)
	a3 := math.Abs(az)
	if a1 >= a2 && a1 >= a3 {
		if ax >= 0 {
			return 1
		}
		return -1
	}
	if a2 >= a1 && a2 >= a3 {
		if ay >= 0 {
			return 2
		}
		return -2
	}
	if az >= 0 {
		return 3
	}
	return -3
}

func dot3(ax, ay, az float64, b [3]float64) float64 {
	return ax*b[0] + ay*b[1] + az*b[2]
}

func norm3(v [3]float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func unit3(v [3]float64) ([3]float64, error) {
	n := norm3(v)
	if n <= 0 {
		return [3]float64{}, fmt.Errorf("zero vector")
	}
	return [3]float64{v[0] / n, v[1] / n, v[2] / n}, nil
}

func cross3(a, b [3]float64) [3]float64 {
	return [3]float64{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func (s *Service) applyOrientationFromGravity(avgAccel [3]float64) error {
	// Build an orthonormal basis in sensor coordinates.
	s.mu.Lock()
	defer s.mu.Unlock()

	f := s.forwardAxis
	if f == 0 {
		return fmt.Errorf("ahrs: forward direction not set")
	}

	z, err := unit3(avgAccel)
	if err != nil {
		return fmt.Errorf("ahrs: invalid gravity vector: %v", err)
	}

	// Forward axis unit vector in sensor coordinates.
	x := [3]float64{0, 0, 0}
	idx := f
	sign := 1.0
	if idx < 0 {
		idx = -idx
		sign = -1.0
	}
	if idx < 1 || idx > 3 {
		return fmt.Errorf("ahrs: invalid forward axis %d", f)
	}
	x[idx-1] = sign

	// Remove any component along gravity to ensure forward is horizontal.
	dot := x[0]*z[0] + x[1]*z[1] + x[2]*z[2]
	xh := [3]float64{x[0] - dot*z[0], x[1] - dot*z[1], x[2] - dot*z[2]}
	xu, err := unit3(xh)
	if err != nil {
		return fmt.Errorf("ahrs: forward axis nearly vertical; try again")
	}

	yu := cross3(z, xu)
	yu, err = unit3(yu)
	if err != nil {
		return fmt.Errorf("ahrs: invalid basis; try again")
	}

	// Save basis vectors.
	s.gravityInSensor = z
	s.bodyXInSensor = xu
	s.bodyYInSensor = yu
	s.bodyZInSensor = z
	s.orientationSet = true
	s.snap.OrientationSet = true
	s.snap.OrientationForwardAxis = s.forwardAxis
	return nil
}

func (s *Service) startupCal(ctx context.Context) {
	// Best-effort: do this once per process start.
	// We wait a bit to allow the IMU filter to settle, then perform SetLevel and ZeroDrift.
	s.startupOnce.Do(func() {
		if s == nil {
			return
		}
		settle := time.NewTimer(3 * time.Second)
		defer settle.Stop()
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-settle.C:
		}

		// Try SetLevel (requires valid).
		deadline := time.NewTimer(5 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-deadline.C:
				goto ZERO
			default:
			}
			snap := s.Snapshot()
			if snap.Valid {
				_ = s.SetLevel()
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

	ZERO:
		// Try ZeroDrift (non-fatal if it fails).
		zdCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		defer cancel()
		_ = s.ZeroDrift(zdCtx)
	})
}

func pressureToAltitudeFeet(pressurePa float64) float64 {
	// International Standard Atmosphere approximation.
	// h(m) = 44330 * (1 - (p/p0)^(1/5.255))
	p0 := 101325.0
	hMeters := 44330.0 * (1.0 - math.Pow(pressurePa/p0, 1.0/5.255))
	return hMeters * 3.28084
}
