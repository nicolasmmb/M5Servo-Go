package servo

import "time"

// Positional controls a 180° servo.
type Positional struct {
	*Servo

	angle float32
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

// SetAngle sets the shaft to 0..180 degrees. Values outside the range are
// clamped, like in the official driver.
func (p *Positional) SetAngle(degrees float32) {
	if degrees < 0 {
		degrees = 0
	}
	if degrees > 180 {
		degrees = 180
	}
	p.setTarget(p.min + time.Duration(float32(p.span())*(degrees/180)))
	p.angle = degrees
}

// SetPercent sets the position as 0..100 percent. Values are clamped.
func (p *Positional) SetPercent(percent uint8) {
	if percent > 100 {
		percent = 100
	}
	p.SetAngle(float32(percent) * 1.8)
}

// SetRadians sets the position as 0..π radians. Values are clamped.
func (p *Positional) SetRadians(radians float32) {
	if radians < 0 {
		radians = 0
	}
	if radians > 3.1415926535 {
		radians = 3.1415926535
	}
	p.SetAngle(radians * 57.29577951)
}

// Angle returns the last commanded angle in degrees.
func (p *Positional) Angle() float32 { return p.angle }
