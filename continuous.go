package servo

import "time"

// Continuous controls a 360° continuous rotation servo.
type Continuous struct {
	*Servo

	speed int8
}

// NewContinuous returns a 360° servo and writes MinPulse.
//
// Warning: MinPulse is full speed clockwise, exactly like the official
// M5Stack driver. The servo rotates from power-on until Stop is called.
func NewContinuous(cfg Config) (*Continuous, error) {
	s, err := newServo(cfg)
	if err != nil {
		return nil, err
	}
	return &Continuous{Servo: s}, nil
}

// Clockwise rotates with the given speed (0..100 percent). The pulse goes
// from StopPulse down to MinPulse.
//
// Speed 0 stops the servo. A speed between 1 and MinSpeed is raised to
// MinSpeed to overcome static friction, like the official driver. Set
// MinSpeed to 0 in Config to disable the clamp.
func (c *Continuous) Clockwise(speed uint8) {
	if speed > 100 {
		speed = 100
	}
	if speed == 0 {
		c.Stop()
		return
	}
	if !c.noMinSpd && c.minSpd > 0 && speed < c.minSpd {
		speed = c.minSpd
	}
	c.setTarget(c.stop - time.Duration(int64(c.span()/2)*int64(speed)/100))
	c.speed = int8(speed)
}

// CounterClockwise rotates with the given speed (0..100 percent). The pulse
// goes from StopPulse up to MaxPulse.
func (c *Continuous) CounterClockwise(speed uint8) {
	if speed > 100 {
		speed = 100
	}
	if speed == 0 {
		c.Stop()
		return
	}
	if !c.noMinSpd && c.minSpd > 0 && speed < c.minSpd {
		speed = c.minSpd
	}
	c.setTarget(c.stop + time.Duration(int64(c.span()/2)*int64(speed)/100))
	c.speed = -int8(speed)
}

// Stop writes StopPulse and holds the servo in place.
func (c *Continuous) Stop() {
	c.setTarget(c.stop)
	c.speed = 0
}

// SetSpeed uses the sign as direction: positive is clockwise, negative is
// counterclockwise, zero stops. The value is clamped to -100..100.
func (c *Continuous) SetSpeed(percent int8) {
	switch {
	case percent == 0:
		c.Stop()
	case percent > 0:
		c.Clockwise(uint8(percent))
	default:
		c.CounterClockwise(uint8(-percent))
	}
}

// Speed returns the last commanded speed: positive clockwise, negative
// counterclockwise, zero stopped.
func (c *Continuous) Speed() int8 { return c.speed }
