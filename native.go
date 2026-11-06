//go:build esp32 || rp2040

// Adapter for chips whose TinyGo PWM API is Configure(Period) + Channel(pin)
// + Set(channel, value) + Top(). Both the classic ESP32 (LEDC) and the
// RP2040 expose this shape, so the same code serves any board built on them.
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

func prepare(pwm NativePWM, pin machine.Pin) (uint8, error) {
	if err := ConfigurePeriod(pwm, DefaultPeriod); err != nil {
		return 0, err
	}
	return AllocateChannel(pwm, pin)
}
