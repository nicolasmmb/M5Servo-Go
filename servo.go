// Package servo drives 50 Hz hobby servos (reference hardware: M5Stack
// Servo Kit 360° and 180°) through a hardware PWM peripheral. It is pure Go
// and runs on TinyGo and on the host.
//
// The package knows nothing about boards. The core depends only on the
// two-method PWM interface (Set and Top), so any MCU, any PWM timer and any
// pin work. Board details stay outside: the caller picks the peripheral and
// the pin.
//
// The package mirrors the official M5Stack UIFlow2 drivers
// (Servo180Unit/Servo360Unit): 20 ms period, 500-2500 µs pulse, stop at the
// midpoint, and a 10 % minimum speed for continuous rotation.
//
// Two hardware differences are deliberate:
//
//   - no blocking wait: Settling reports how long the servo needs to move,
//     the caller decides whether to sleep;
//   - no console output.
//
// Like the official driver, the constructors write MinPulse immediately. On a
// 360° servo that means full speed clockwise until Stop is called.
package servo

import (
	"errors"
	"time"
)

// PWM is the subset of the TinyGo PWM peripheral the driver needs.
// machine.LEDCPWM (machine.PWM0..PWM3 on ESP32) satisfies it.
type PWM interface {
	// Set writes the duty of channel, in counts from 0 to Top().
	Set(channel uint8, value uint32)
	// Top returns (1<<resolution)-1 of the configured timer.
	Top() uint32
}

// Defaults of the M5Stack Servo Kit and the official drivers.
const (
	DefaultPeriod   = 20 * time.Millisecond // 50 Hz
	DefaultMinPulse = 500 * time.Microsecond
	DefaultMaxPulse = 2500 * time.Microsecond
	DefaultMinSpeed = 10 // percent, overcomes static friction

	// settlingPerPulse is the official constant: 0.15 ms per µs of travel.
	settlingPerPulse = 150 * time.Microsecond
)

// Errors returned by the constructors and by Bank. They are sentinels:
// compare with errors.Is.
var (
	ErrNilPWM     = errors.New("servo: nil pwm")
	ErrBadPWM     = errors.New("servo: pwm top is zero, configure the timer first")
	ErrBadPeriod  = errors.New("servo: period must be between 1ms and 1s")
	ErrBadRange   = errors.New("servo: min pulse must be lower than max pulse")
	ErrBadStop    = errors.New("servo: stop pulse is outside the range")
	ErrInUse      = errors.New("servo: channel already in use in this bank")
	ErrClosed     = errors.New("servo: bank is closed")
	ErrBadChannel = errors.New("servo: channel must be below 32")
)

// Config configures one servo. Zero values take the defaults.
type Config struct {
	// PWM is the configured peripheral. Required.
	PWM PWM
	// Channel is the channel of PWM that drives the servo pin.
	Channel uint8
	// Period is the PWM period. Default DefaultPeriod (50 Hz).
	Period time.Duration
	// MinPulse is the pulse for 0° (180°) or full speed (360°).
	MinPulse time.Duration
	// MaxPulse is the pulse for 180° (180°) or full speed (360°).
	MaxPulse time.Duration
	// StopPulse is the pulse that stops a 360° servo. Default: midpoint.
	StopPulse time.Duration
	// MinSpeed is the minimum non-zero speed in percent (360° only).
	// Zero means DefaultMinSpeed.
	MinSpeed uint8
	// NoMinSpeed disables the minimum speed clamp of a 360° servo.
	// The official driver clamps 1..9 % to 10 % to overcome friction.
	NoMinSpeed bool
	// Ramp is the time to travel the full pulse range. Zero disables the
	// ramp and every command applies immediately.
	Ramp time.Duration
}

// Servo is the shared 50 Hz driver. Positional and Continuous embed it.
//
// All setters clamp values and never allocate. Use Update to apply a
// configured ramp; without a ramp Update is not needed.
type Servo struct {
	pwm      PWM
	period   time.Duration
	min      time.Duration
	max      time.Duration
	stop     time.Duration
	ramp     time.Duration
	last     time.Duration
	target   time.Duration
	lastAt   time.Duration
	ch       uint8
	minSpd   uint8
	noMinSpd bool
	released bool
	started  bool
}

// newServo validates cfg, fills the defaults and writes MinPulse, like the
// official driver does in its constructor.
func newServo(cfg Config) (*Servo, error) {
	if cfg.PWM == nil {
		return nil, ErrNilPWM
	}
	top := cfg.PWM.Top()
	if top == 0 {
		return nil, ErrBadPWM
	}
	if cfg.Period == 0 {
		cfg.Period = DefaultPeriod
	}
	if cfg.Period < time.Millisecond || cfg.Period > time.Second {
		return nil, ErrBadPeriod
	}
	if cfg.MinPulse == 0 {
		cfg.MinPulse = DefaultMinPulse
	}
	if cfg.MaxPulse == 0 {
		cfg.MaxPulse = DefaultMaxPulse
	}
	if cfg.MinPulse >= cfg.MaxPulse {
		return nil, ErrBadRange
	}
	if cfg.StopPulse == 0 {
		cfg.StopPulse = (cfg.MinPulse + cfg.MaxPulse) / 2
	}
	if cfg.StopPulse < cfg.MinPulse || cfg.StopPulse > cfg.MaxPulse {
		return nil, ErrBadStop
	}
	s := &Servo{
		pwm:      cfg.PWM,
		ch:       cfg.Channel,
		period:   cfg.Period,
		min:      cfg.MinPulse,
		max:      cfg.MaxPulse,
		stop:     cfg.StopPulse,
		ramp:     cfg.Ramp,
		minSpd:   cfg.MinSpeed,
		noMinSpd: cfg.NoMinSpeed,
	}
	if s.minSpd == 0 && !s.noMinSpd {
		s.minSpd = DefaultMinSpeed
	}
	s.write(s.min)
	s.last = s.min
	s.target = s.min
	return s, nil
}

// Channel returns the PWM channel of the servo.
func (s *Servo) Channel() uint8 { return s.ch }

// Period returns the configured PWM period.
func (s *Servo) Period() time.Duration { return s.period }

// Resolution returns the pulse step of the current PWM configuration,
// Period/(Top+1). At 50 Hz with 14 bits it is about 1.22 µs.
func (s *Servo) Resolution() time.Duration {
	ticks := int64(s.pwm.Top()) + 1
	return time.Duration(int64(s.period) / ticks)
}

// Pulse returns the pulse last written to the PWM.
func (s *Servo) Pulse() time.Duration { return s.last }

// Target returns the pulse the ramp is moving towards.
func (s *Servo) Target() time.Duration { return s.target }

// Released reports whether Release was called.
func (s *Servo) Released() bool { return s.released }

// Range returns the configured MinPulse and MaxPulse.
func (s *Servo) Range() (min, max time.Duration) { return s.min, s.max }

// StopPulse returns the pulse that stops a 360° servo.
func (s *Servo) StopPulse() time.Duration { return s.stop }

// SetPulse writes a pulse immediately and cancels a pending ramp.
// The value is clamped to the configured range.
func (s *Servo) SetPulse(p time.Duration) {
	p = s.clamp(p)
	s.write(p)
	s.last = p
	s.target = p
	s.released = false
}

// SetStopPulse trims the pulse that stops a 360° servo. Use it when the
// midpoint is not the real stop point of the unit.
func (s *Servo) SetStopPulse(p time.Duration) error {
	if p < s.min || p > s.max {
		return ErrBadStop
	}
	s.stop = p
	return nil
}

// SetRange changes the pulse range, for example to 1000-2000 µs for a 360°
// servo with a narrower course. StopPulse becomes the new midpoint and a
// pending ramp is cancelled.
func (s *Servo) SetRange(min, max time.Duration) error {
	if min <= 0 || min >= max {
		return ErrBadRange
	}
	s.min, s.max = min, max
	s.stop = (min + max) / 2
	s.SetPulse(s.zap(s.last))
	return nil
}

// Release removes the pulse. A hobby servo has no torque without a pulse,
// so the shaft turns freely. This is not the same as Stop, which holds the
// servo at the stop pulse.
func (s *Servo) Release() {
	s.pwm.Set(s.ch, 0)
	s.last = 0
	s.target = 0
	s.released = true
	s.started = false
}

// Update applies the ramp towards Target. Call it at the rate of the control
// loop with a monotonic timestamp. Without a ramp it only records the time.
func (s *Servo) Update(now time.Duration) {
	if s.ramp <= 0 || s.released {
		return
	}
	if !s.started {
		s.lastAt = now
		s.started = true
		return
	}
	dt := now - s.lastAt
	if dt <= 0 {
		return
	}
	s.lastAt = now
	if s.last == s.target {
		return
	}
	step := time.Duration(int64(s.span()) * int64(dt) / int64(s.ramp))
	if step <= 0 {
		return
	}
	if s.target > s.last {
		s.last += step
		if s.last > s.target {
			s.last = s.target
		}
	} else {
		s.last -= step
		if s.last < s.target {
			s.last = s.target
		}
	}
	s.write(s.last)
}

// SetRamp sets the time to travel the full pulse range and restarts the ramp
// time base. Zero disables the ramp.
func (s *Servo) SetRamp(d time.Duration) {
	if d < 0 {
		d = 0
	}
	s.ramp = d
	s.started = false
}

// Ramp returns the configured full-range travel time, zero when disabled.
func (s *Servo) Ramp() time.Duration { return s.ramp }

// Settling reports how long the servo needs to travel from one pulse to
// another. It uses the official constant of 0.15 ms per µs.
func (s *Servo) Settling(from, to time.Duration) time.Duration {
	return Settling(from, to)
}

// Settling is the package-level form of Servo.Settling.
func Settling(from, to time.Duration) time.Duration {
	d := to - from
	if d < 0 {
		d = -d
	}
	return time.Duration(int64(d) / 1000 * int64(settlingPerPulse))
}

// span returns MaxPulse-MinPulse.
func (s *Servo) span() time.Duration { return s.max - s.min }

// clamp limits p to the configured range.
func (s *Servo) clamp(p time.Duration) time.Duration {
	if p < s.min {
		return s.min
	}
	if p > s.max {
		return s.max
	}
	return p
}

// zap re-clamps a pulse after a range change, using the new limits.
func (s *Servo) zap(p time.Duration) time.Duration {
	if p == 0 {
		return s.min
	}
	return s.clamp(p)
}

// write converts the pulse to counts and writes the PWM.
func (s *Servo) write(p time.Duration) {
	ticks := int64(s.pwm.Top()) + 1
	counts := (int64(p)*ticks + int64(s.period)/2) / int64(s.period)
	if counts < 0 {
		counts = 0
	}
	if max := ticks - 1; counts > max {
		counts = max
	}
	s.pwm.Set(s.ch, uint32(counts))
}

// setTarget clamps the target and applies it, with or without a ramp.
func (s *Servo) setTarget(p time.Duration) {
	p = s.clamp(p)
	s.target = p
	s.released = false
	if s.ramp <= 0 {
		s.write(p)
		s.last = p
		s.started = false
	}
}
