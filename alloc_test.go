package servo

import (
	"testing"
	"time"
	"unsafe"
)

// silentPWM keeps only the last value, so it does not allocate during a
// benchmark or an allocation contract test.
type silentPWM struct {
	top  uint32
	last uint32
}

func (p *silentPWM) Set(_ uint8, v uint32) { p.last = v }
func (p *silentPWM) Top() uint32           { return p.top }

func TestNoAllocContinuous(t *testing.T) {
	p := &silentPWM{top: top14}
	c, err := NewContinuous(Config{PWM: p, Ramp: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(200, func() {
		c.Clockwise(50)
		c.Update(time.Millisecond)
		c.CounterClockwise(30)
		c.Update(2 * time.Millisecond)
		c.SetPulse(1500 * time.Microsecond)
		c.Stop()
		c.Speed()
		c.Pulse()
		c.Resolution()
		c.Settling(500*time.Microsecond, 2500*time.Microsecond)
		c.Release()
	})
	if allocs != 0 {
		t.Fatalf("continuous path allocates %v per run", allocs)
	}
}

func TestNoAllocPositional(t *testing.T) {
	p := &silentPWM{top: top14}
	pos, err := NewPositional(Config{PWM: p})
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(200, func() {
		pos.SetAngle(90)
		pos.SetPercent(50)
		pos.SetRadians(1.5)
		pos.Angle()
	})
	if allocs != 0 {
		t.Fatalf("positional path allocates %v per run", allocs)
	}
}

func TestNoAllocBank(t *testing.T) {
	p := &silentPWM{top: top14}
	b, err := NewBank(BankConfig{PWM: p})
	if err != nil {
		t.Fatal(err)
	}
	c, err := b.AddContinuous(0)
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(200, func() {
		c.Clockwise(40)
		b.Len()
	})
	if allocs != 0 {
		t.Fatalf("bank path allocates %v per run", allocs)
	}
}

func TestFootprint(t *testing.T) {
	size := unsafe.Sizeof(Servo{})
	bank := unsafe.Sizeof(Bank{})
	t.Logf("Servo=%d B Bank=%d B", size, bank)
	if size > 128 {
		t.Fatalf("Servo is %d B, budget is 128 B", size)
	}
	if bank > 64 {
		t.Fatalf("Bank is %d B, budget is 64 B", bank)
	}
}

// TestNoWriteWithoutChange: a fixed pulse must not be rewritten in a loop,
// otherwise the servo can jitter.
func TestNoWriteWithoutChange(t *testing.T) {
	p := &recordingPWM{top: top14}
	c, err := NewContinuous(Config{PWM: p})
	if err != nil {
		t.Fatal(err)
	}
	c.SetPulse(1200 * time.Microsecond)
	n := len(p.writes)
	for i := 0; i < 50; i++ {
		c.Update(time.Duration(i) * time.Millisecond)
	}
	if len(p.writes) != n {
		t.Fatalf("Update wrote %d extra times without a new command", len(p.writes)-n)
	}
}
