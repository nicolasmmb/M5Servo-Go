package servo

import (
	"testing"
	"time"
)

func benchServo(b *testing.B, cfg Config) (*Continuous, *silentPWM) {
	b.Helper()
	p := &silentPWM{top: top14}
	cfg.PWM = p
	c, err := NewContinuous(cfg)
	if err != nil {
		b.Fatal(err)
	}
	return c, p
}

// BenchmarkSetPulse measures the cost of one pulse command.
func BenchmarkSetPulse(b *testing.B) {
	c, p := benchServo(b, Config{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.SetPulse(1500 * time.Microsecond)
	}
	b.StopTimer()
	_ = p
}

// BenchmarkClockwise measures the mapping plus the write.
func BenchmarkClockwise(b *testing.B) {
	c, _ := benchServo(b, Config{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Clockwise(uint8(i%100) + 1)
	}
}

// BenchmarkSetAngle measures the positional path.
func BenchmarkSetAngle(b *testing.B) {
	p := &silentPWM{top: top14}
	pos, err := NewPositional(Config{PWM: p})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos.SetAngle(float32(i % 180))
	}
}

// BenchmarkRampUpdate is the worst case: a ramp step on every call.
func BenchmarkRampUpdate(b *testing.B) {
	c, _ := benchServo(b, Config{Ramp: time.Second})
	c.SetPulse(DefaultMinPulse)
	c.setTarget(DefaultMaxPulse)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Update(time.Duration(i) * time.Millisecond)
	}
}

// BenchmarkBankFourServos commands four servos that share one timer.
func BenchmarkBankFourServos(b *testing.B) {
	p := &silentPWM{top: top14}
	bk, err := NewBank(BankConfig{PWM: p})
	if err != nil {
		b.Fatal(err)
	}
	var servos [4]*Continuous
	for i := range servos {
		s, err := bk.AddContinuous(uint8(i))
		if err != nil {
			b.Fatal(err)
		}
		servos[i] = s
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range servos {
			s.Clockwise(50)
		}
	}
}
