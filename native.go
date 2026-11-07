//go:build esp32 || esp32c3 || esp32s3 || rp2040

// Adapter for chips whose TinyGo PWM API is Configure(Period) + Channel(pin)
// + Set(channel, value) + Top(). The classic ESP32, the ESP32-C3/S3 (same LEDC
// API through machine_esp32xx_pwm.go) and the RP2040 all expose this shape.
//
// A board is never named here: the caller passes the PWM peripheral it wants
// (machine.PWM0..PWM3, machine.PWM0..PWM7, ...) and the pin that the servo
// signal is wired to. Boards with another PWM API can still use the package
// by implementing the two-method PWM interface themselves.

package servo

import (
	"machine"
	"time"
)

// NativePWM is the subset of the TinyGo PWM peripheral the driver uses.
// machine.LEDCPWM (ESP32) and machine's pwmGroup (RP2040) satisfy it.
type NativePWM interface {
	Configure(machine.PWMConfig) error
	Channel(pin machine.Pin) (uint8, error)
	Set(channel uint8, value uint32)
	Top() uint32
}

// ConfigurePeriod programs the PWM timer. A zero period means DefaultPeriod.
func ConfigurePeriod(pwm NativePWM, period time.Duration) error {
	if period == 0 {
		period = DefaultPeriod
	}
	return pwm.Configure(machine.PWMConfig{Period: uint64(period)})
}

// AllocateChannel reserves a PWM channel for pin. Call ConfigurePeriod first.
func AllocateChannel(pwm NativePWM, pin machine.Pin) (uint8, error) {
	return pwm.Channel(pin)
}

// NewNative configures the timer at 50 Hz and returns a 360° servo on pin.
//
// The caller chooses the peripheral and the pin, so the same call works on any
// board: pass machine.PWM0 with the pin wired to the servo signal.
func NewNative(pwm NativePWM, pin machine.Pin) (*Continuous, error) {
	ch, err := prepare(pwm, pin)
	if err != nil {
		return nil, err
	}
	return NewContinuous(Config{PWM: pwm, Channel: ch})
}

// NewNativePositional is the 180° variant of NewNative.
func NewNativePositional(pwm NativePWM, pin machine.Pin) (*Positional, error) {
	ch, err := prepare(pwm, pin)
	if err != nil {
		return nil, err
	}
	return NewPositional(Config{PWM: pwm, Channel: ch})
}

// DetachNative is the closest equivalent of the official deinit(): it stops the
// pulses, returns the pin to input mode (high impedance) and frees the channels
// of the timer.
//
// TinyGo has no PWM deinit, so the channel release works by reconfiguring the
// timer with the same period: Configure disables and frees every channel of
// that timer. Side effect: any other servo on the same timer also stops.
//
// Call it on the Servo you want to release; the pin is the one passed to
// NewNative.
func DetachNative(s *Servo, pwm NativePWM, pin machine.Pin) error {
	s.Release()
	pin.Configure(machine.PinConfig{Mode: machine.PinInput})
	return pwm.Configure(machine.PWMConfig{Period: uint64(s.Period())})
}

// DetachNativeAll is DetachNative for a group: it returns every pin to input
// mode and frees the timer once.
//
// The timer release works by reconfiguring it, which is the only way to free
// LEDC channels in TinyGo. Every servo on that timer stops, and their Servo
// values must not be used afterwards.
func DetachNativeAll(pwm NativePWM, period time.Duration, pins ...machine.Pin) error {
	for _, p := range pins {
		p.Configure(machine.PinConfig{Mode: machine.PinInput})
	}
	if period == 0 {
		period = DefaultPeriod
	}
	return pwm.Configure(machine.PWMConfig{Period: uint64(period)})
}

func prepare(pwm NativePWM, pin machine.Pin) (uint8, error) {
	if err := ConfigurePeriod(pwm, DefaultPeriod); err != nil {
		return 0, err
	}
	return AllocateChannel(pwm, pin)
}
