package servo

import "time"

// maxChannels is the number of channels of the ESP32 LEDC. The Bank bitmask
// covers 32 channels, so it also fits other PWM peripherals.
const maxChannels = 32

// BankConfig configures a group of servos that share one PWM timer, and so
// one frequency.
type BankConfig struct {
	// PWM is the configured peripheral. Required.
	PWM PWM
	// Period is the PWM period. Default DefaultPeriod (50 Hz).
	Period time.Duration
}

// Bank holds several servos on the same timer. It allocates no channel by
// itself: the caller resolves the channel (for example with
// machine.PWM0.Channel(pin) on ESP32) and passes the number to Add.
type Bank struct {
	pwm    PWM
	period time.Duration
	used   uint32
	closed bool
}

// NewBank returns an empty bank.
func NewBank(cfg BankConfig) (*Bank, error) {
	if cfg.PWM == nil {
		return nil, ErrNilPWM
	}
	if cfg.PWM.Top() == 0 {
		return nil, ErrBadPWM
	}
	if cfg.Period == 0 {
		cfg.Period = DefaultPeriod
	}
	if cfg.Period < time.Millisecond || cfg.Period > time.Second {
		return nil, ErrBadPeriod
	}
	return &Bank{pwm: cfg.PWM, period: cfg.Period}, nil
}

// Len returns the number of servos in the bank.
func (b *Bank) Len() int {
	n := 0
	for m := b.used; m != 0; m &= m - 1 {
		n++
	}
	return n
}

// AddContinuous adds a 360° servo on channel.
func (b *Bank) AddContinuous(channel uint8) (*Continuous, error) {
	if err := b.reserve(channel); err != nil {
		return nil, err
	}
	return NewContinuous(Config{PWM: b.pwm, Channel: channel, Period: b.period})
}

// AddPositional adds a 180° servo on channel.
func (b *Bank) AddPositional(channel uint8) (*Positional, error) {
	if err := b.reserve(channel); err != nil {
		return nil, err
	}
	return NewPositional(Config{PWM: b.pwm, Channel: channel, Period: b.period})
}

// Release removes the pulse from every servo of the bank. The shafts turn
// freely and hold no torque.
func (b *Bank) Release() {
	for ch := uint8(0); ch < maxChannels; ch++ {
		if b.used&(1<<ch) != 0 {
			b.pwm.Set(ch, 0)
		}
	}
}

// Close releases every servo and rejects new additions.
func (b *Bank) Close() {
	if b.closed {
		return
	}
	b.Release()
	b.closed = true
}

func (b *Bank) reserve(channel uint8) error {
	if b.closed {
		return ErrClosed
	}
	if channel >= maxChannels {
		return ErrBadChannel
	}
	if b.used&(1<<channel) != 0 {
		return ErrInUse
	}
	b.used |= 1 << channel
	return nil
}
