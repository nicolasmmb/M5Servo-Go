package servo

// Positional controls a 180° servo. It embeds Servo, so it has the same
// commands as the 360° type; the difference is only the meaning of the pulse.
type Positional struct {
	*Servo
}

// NewPositional returns a positional servo and writes MinPulse (0°), like the
// official M5Stack driver.
func NewPositional(cfg Config) (*Positional, error) {
	s, err := newServo(cfg)
	if err != nil {
		return nil, err
	}
	return &Positional{Servo: s}, nil
}
