package servo

import (
	"testing"
	"time"
)

const top14 = 1<<14 - 1

// recordingPWM keeps every write, so the tests can inspect the sequence.
type recordingPWM struct {
	top    uint32
	writes []write
}

type write struct {
	ch uint8
	v  uint32
}

func (p *recordingPWM) Set(ch uint8, v uint32) {
	p.writes = append(p.writes, write{ch, v})
}

func (p *recordingPWM) Top() uint32 { return p.top }

func (p *recordingPWM) last() uint32 {
	if len(p.writes) == 0 {
		return 0
	}
	return p.writes[len(p.writes)-1].v
}

func newTestServo(t *testing.T, cfg Config) (*Servo, *recordingPWM) {
	t.Helper()
	p := &recordingPWM{top: top14}
	cfg.PWM = p
	s, err := newServo(cfg)
	if err != nil {
		t.Fatalf("newServo: %v", err)
	}
	return s, p
}

func newTestCont(t *testing.T, cfg Config) (*Continuous, *recordingPWM) {
	t.Helper()
	p := &recordingPWM{top: top14}
	cfg.PWM = p
	c, err := NewContinuous(cfg)
	if err != nil {
		t.Fatalf("NewContinuous: %v", err)
	}
	return c, p
}

func TestDefaults(t *testing.T) {
	s, _ := newTestServo(t, Config{})
	if s.Period() != DefaultPeriod {
		t.Fatalf("period = %v", s.Period())
	}
	min, max := s.Range()
	if min != DefaultMinPulse || max != DefaultMaxPulse {
		t.Fatalf("range = %v..%v", min, max)
	}
	if s.StopPulse() != 1500*time.Microsecond {
		t.Fatalf("stop = %v", s.StopPulse())
	}
}

// TestInitialPulse locks the parity decision: the constructor writes MinPulse,
// like the official driver.
func TestInitialPulse(t *testing.T) {
	c, p := newTestCont(t, Config{})
	if len(p.writes) != 1 {
		t.Fatalf("writes at construction = %d, want 1", len(p.writes))
	}
	if got := p.last(); got != 410 {
		t.Fatalf("initial counts = %d, want 410 (500µs)", got)
	}
	if c.Pulse() != DefaultMinPulse {
		t.Fatalf("pulse = %v, want 500µs", c.Pulse())
	}
}

// TestConversion checks pulse to counts at the real 14-bit resolution.
func TestConversion(t *testing.T) {
	s, p := newTestServo(t, Config{})
	cases := []struct {
		pulse  time.Duration
		counts uint32
	}{
		{500 * time.Microsecond, 410},
		{1000 * time.Microsecond, 819},
		{1500 * time.Microsecond, 1229},
		{2000 * time.Microsecond, 1638},
		{2500 * time.Microsecond, 2048},
	}
	for _, c := range cases {
		s.SetPulse(c.pulse)
		if got := p.last(); got != c.counts {
			t.Fatalf("pulse %v: counts = %d, want %d", c.pulse, got, c.counts)
		}
	}
}

// TestConversionOtherResolutions makes sure the math is resolution agnostic.
func TestConversionOtherResolutions(t *testing.T) {
	cases := []struct {
		top    uint32
		pulse  time.Duration
		counts uint32
	}{
		{1<<8 - 1, 1500 * time.Microsecond, 19},    // 8 bits
		{1<<16 - 1, 1500 * time.Microsecond, 4915}, // 16 bits
	}
	for _, c := range cases {
		p := &recordingPWM{top: c.top}
		s, err := newServo(Config{PWM: p})
		if err != nil {
			t.Fatal(err)
		}
		s.SetPulse(c.pulse)
		if got := p.last(); got != c.counts {
			t.Fatalf("top %d: counts = %d, want %d", c.top, got, c.counts)
		}
	}
}

func TestClamp(t *testing.T) {
	s, p := newTestServo(t, Config{})
	s.SetPulse(100 * time.Microsecond)
	if got := s.Pulse(); got != DefaultMinPulse {
		t.Fatalf("low clamp = %v", got)
	}
	s.SetPulse(9 * time.Millisecond)
	if got := s.Pulse(); got != DefaultMaxPulse {
		t.Fatalf("high clamp = %v", got)
	}
	if p.last() != 2048 {
		t.Fatalf("counts at max = %d", p.last())
	}
}

func TestResolution(t *testing.T) {
	s, _ := newTestServo(t, Config{})
	if got := s.Resolution(); got != 1220*time.Nanosecond {
		t.Fatalf("resolution = %v, want 1220ns (14 bits @ 50Hz)", got)
	}
}

func TestSettling(t *testing.T) {
	if got := Settling(500*time.Microsecond, 2500*time.Microsecond); got != 300*time.Millisecond {
		t.Fatalf("settling full range = %v, want 300ms", got)
	}
	if got := Settling(1500*time.Microsecond, 1000*time.Microsecond); got != 75*time.Millisecond {
		t.Fatalf("settling 500µs = %v, want 75ms", got)
	}
}

func TestPositional(t *testing.T) {
	p := &recordingPWM{top: top14}
	pos, err := NewPositional(Config{PWM: p})
	if err != nil {
		t.Fatal(err)
	}
	pos.SetAngle(90)
	if got := pos.Pulse(); got != 1500*time.Microsecond {
		t.Fatalf("90° = %v, want 1500µs", got)
	}
	if got := pos.Angle(); got != 90 {
		t.Fatalf("angle = %v", got)
	}

	pos.SetAngle(-5)
	if got := pos.Pulse(); got != DefaultMinPulse {
		t.Fatalf("clamp low = %v", got)
	}
	pos.SetAngle(200)
	if got := pos.Pulse(); got != DefaultMaxPulse {
		t.Fatalf("clamp high = %v", got)
	}

	pos.SetPercent(50)
	if got := pos.Pulse(); got != 1500*time.Microsecond {
		t.Fatalf("50%% = %v", got)
	}
	pos.SetRadians(1.5708)
	if got := pos.Pulse(); got < 1499*time.Microsecond || got > 1501*time.Microsecond {
		t.Fatalf("π/2 = %v, want ~1500µs", got)
	}
}

func TestContinuous(t *testing.T) {
	c, p := newTestCont(t, Config{})

	c.Stop()
	if got := c.Pulse(); got != 1500*time.Microsecond {
		t.Fatalf("stop = %v", got)
	}
	if c.Speed() != 0 {
		t.Fatalf("speed after stop = %d", c.Speed())
	}

	c.Clockwise(100)
	if got := c.Pulse(); got != 500*time.Microsecond {
		t.Fatalf("cw 100 = %v", got)
	}
	c.Clockwise(50)
	if got := c.Pulse(); got != 1000*time.Microsecond {
		t.Fatalf("cw 50 = %v", got)
	}
	c.CounterClockwise(100)
	if got := c.Pulse(); got != 2500*time.Microsecond {
		t.Fatalf("ccw 100 = %v", got)
	}
	c.CounterClockwise(25)
	if got := c.Pulse(); got != 1750*time.Microsecond {
		t.Fatalf("ccw 25 = %v", got)
	}

	// Minimum speed clamp, like the official driver.
	c.Clockwise(5)
	if got := c.Pulse(); got != 1400*time.Microsecond {
		t.Fatalf("cw 5 with minSpeed 10 = %v, want 1400µs", got)
	}
	if c.Speed() != 10 {
		t.Fatalf("speed = %d, want 10", c.Speed())
	}

	c.SetSpeed(-50)
	if got := c.Pulse(); got != 2000*time.Microsecond {
		t.Fatalf("setSpeed -50 = %v", got)
	}
	if c.Speed() != -50 {
		t.Fatalf("speed = %d", c.Speed())
	}
	c.SetSpeed(0)
	if got := c.Pulse(); got != 1500*time.Microsecond {
		t.Fatalf("setSpeed 0 = %v", got)
	}
	_ = p
}

func TestMinSpeedDisabled(t *testing.T) {
	c, _ := newTestCont(t, Config{NoMinSpeed: true})
	c.Clockwise(5)
	if got := c.Pulse(); got != 1450*time.Microsecond {
		t.Fatalf("cw 5 without clamp = %v, want 1450µs", got)
	}
}

func TestSetRange(t *testing.T) {
	c, _ := newTestCont(t, Config{})
	if err := c.SetRange(1000*time.Microsecond, 2000*time.Microsecond); err != nil {
		t.Fatal(err)
	}
	if got := c.StopPulse(); got != 1500*time.Microsecond {
		t.Fatalf("stop = %v", got)
	}
	c.Clockwise(100)
	if got := c.Pulse(); got != 1000*time.Microsecond {
		t.Fatalf("cw 100 on narrow range = %v", got)
	}
	if err := c.SetRange(0, 1000*time.Microsecond); err != ErrBadRange {
		t.Fatalf("bad range error = %v", err)
	}
}

func TestStopPulseTrim(t *testing.T) {
	c, _ := newTestCont(t, Config{})
	if err := c.SetStopPulse(1520 * time.Microsecond); err != nil {
		t.Fatal(err)
	}
	c.Stop()
	if got := c.Pulse(); got != 1520*time.Microsecond {
		t.Fatalf("trimmed stop = %v", got)
	}
	if err := c.SetStopPulse(4 * time.Millisecond); err != ErrBadStop {
		t.Fatalf("bad trim = %v", err)
	}
}

func TestRelease(t *testing.T) {
	c, p := newTestCont(t, Config{})
	c.Clockwise(50)
	c.Release()
	if !c.Released() {
		t.Fatal("Released() = false")
	}
	if p.last() != 0 {
		t.Fatalf("duty after release = %d, want 0", p.last())
	}
	c.Stop()
	if c.Released() {
		t.Fatal("Stop must clear the released state")
	}
	if got := c.Pulse(); got != 1500*time.Microsecond {
		t.Fatalf("pulse after stop = %v", got)
	}
}

func TestRamp(t *testing.T) {
	c, _ := newTestCont(t, Config{Ramp: time.Second})
	c.SetPulse(500 * time.Microsecond)

	// A configured ramp only sets the target; Update moves towards it.
	c.setTarget(DefaultMaxPulse)
	if c.Target() != DefaultMaxPulse {
		t.Fatalf("target = %v", c.Target())
	}
	if c.Pulse() != 500*time.Microsecond {
		t.Fatalf("pulse before update = %v", c.Pulse())
	}

	c.Update(0) // first call only records the time base
	c.Update(500 * time.Millisecond)
	if got := c.Pulse(); got != 1500*time.Microsecond {
		t.Fatalf("half ramp = %v, want 1500µs", got)
	}
	c.Update(time.Second)
	if got := c.Pulse(); got != DefaultMaxPulse {
		t.Fatalf("full ramp = %v, want 2500µs", got)
	}

	// Without a ramp the command applies immediately.
	f, _ := newTestCont(t, Config{})
	f.setTarget(1500 * time.Microsecond)
	if got := f.Pulse(); got != 1500*time.Microsecond {
		t.Fatalf("no ramp = %v", got)
	}
}

func TestSetRamp(t *testing.T) {
	c, _ := newTestCont(t, Config{})
	c.SetPulse(500 * time.Microsecond)
	c.SetRamp(time.Second)
	if c.Ramp() != time.Second {
		t.Fatalf("ramp = %v", c.Ramp())
	}
	c.setTarget(2500 * time.Microsecond)
	if c.Pulse() != 500*time.Microsecond {
		t.Fatal("ramp must delay the write")
	}
	c.Update(0)
	c.Update(time.Second)
	if got := c.Pulse(); got != 2500*time.Microsecond {
		t.Fatalf("ramp end = %v", got)
	}
	c.SetRamp(0)
	if c.Ramp() != 0 {
		t.Fatalf("ramp off = %v", c.Ramp())
	}
}

func TestConstructorErrors(t *testing.T) {
	if _, err := newServo(Config{}); err != ErrNilPWM {
		t.Fatalf("nil pwm = %v", err)
	}
	if _, err := newServo(Config{PWM: &recordingPWM{top: 0}}); err != ErrBadPWM {
		t.Fatalf("zero top = %v", err)
	}
	if _, err := newServo(Config{PWM: &recordingPWM{top: top14}, Period: time.Millisecond / 2}); err != ErrBadPeriod {
		t.Fatalf("short period = %v", err)
	}
	if _, err := newServo(Config{PWM: &recordingPWM{top: top14}, MinPulse: 2 * time.Millisecond, MaxPulse: time.Millisecond}); err != ErrBadRange {
		t.Fatalf("bad range = %v", err)
	}
	if _, err := newServo(Config{PWM: &recordingPWM{top: top14}, StopPulse: 4 * time.Millisecond}); err != ErrBadStop {
		t.Fatalf("bad stop = %v", err)
	}
}

func TestBank(t *testing.T) {
	p := &recordingPWM{top: top14}
	b, err := NewBank(BankConfig{PWM: p})
	if err != nil {
		t.Fatal(err)
	}
	c, err := b.AddContinuous(0)
	if err != nil {
		t.Fatal(err)
	}
	pos, err := b.AddPositional(1)
	if err != nil {
		t.Fatal(err)
	}
	if b.Len() != 2 {
		t.Fatalf("len = %d", b.Len())
	}
	if _, err := b.AddContinuous(0); err != ErrInUse {
		t.Fatalf("reuse channel = %v", err)
	}
	if _, err := b.AddContinuous(maxChannels); err != ErrBadChannel {
		t.Fatalf("bad channel = %v", err)
	}
	c.Clockwise(50)
	pos.SetAngle(45)
	if got := pos.Pulse(); got != 1000*time.Microsecond {
		t.Fatalf("45° = %v", got)
	}
	b.Close()
	if _, err := b.AddContinuous(2); err != ErrClosed {
		t.Fatalf("add after close = %v", err)
	}
	if p.last() != 0 {
		t.Fatalf("last write after close = %d, want 0", p.last())
	}
}
