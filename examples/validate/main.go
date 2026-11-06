//go:build esp32 || rp2040

// Validation firmware. It runs ONE test, in a loop, so the operator watches a
// single behaviour and answers a single question.
//
// Set test in config.go and flash again for the next test.
//
// The firmware drives every candidate signal pin with the same pulse, so the
// Grove port of the servo does not matter.
package main

import (
	"time"

	"machine"

	servo "github.com/nicolasmmb/M5Servo-Go"
)

func main() {
	time.Sleep(800 * time.Millisecond)

	if err := servo.ConfigurePeriod(machine.PWM0, servo.DefaultPeriod); err != nil {
		println("pwm:", err.Error())
		return
	}
	var all []*servo.Continuous
	for _, pin := range servoPins {
		ch, err := servo.AllocateChannel(machine.PWM0, pin)
		if err != nil {
			println("channel:", err.Error())
			continue
		}
		s, err := servo.NewContinuous(servo.Config{PWM: machine.PWM0, Channel: ch})
		if err != nil {
			println("servo:", err.Error())
			return
		}
		all = append(all, s)
		println("pin", pin, "channel", ch)
	}
	if len(all) == 0 {
		println("no channel")
		return
	}
	println("resolution:", int64(all[0].Resolution()), "ns")

	each := func(fn func(*servo.Continuous)) {
		for _, s := range all {
			fn(s)
		}
	}

	println("TEST", test, "-", testName(test))

	for {
		switch test {
		case 1: // full speed clockwise
			println("full speed clockwise (4s)")
			each(func(s *servo.Continuous) { s.Clockwise(100) })
			time.Sleep(4 * time.Second)
			println("stop (6s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(6 * time.Second)

		case 2: // full speed counterclockwise
			println("full speed counterclockwise (4s)")
			each(func(s *servo.Continuous) { s.CounterClockwise(100) })
			time.Sleep(4 * time.Second)
			println("stop (6s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(6 * time.Second)

		case 3: // half speed clockwise
			println("clockwise 50% (4s)")
			each(func(s *servo.Continuous) { s.Clockwise(50) })
			time.Sleep(4 * time.Second)
			println("stop (6s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(6 * time.Second)

		case 4: // half speed counterclockwise
			println("counterclockwise 50% (4s)")
			each(func(s *servo.Continuous) { s.CounterClockwise(50) })
			time.Sleep(4 * time.Second)
			println("stop (6s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(6 * time.Second)

		case 5: // minimum speed clamp
			println("clockwise 1% -> clamped to 10% (4s)")
			each(func(s *servo.Continuous) { s.Clockwise(1) })
			time.Sleep(4 * time.Second)
			println("stop (6s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(6 * time.Second)

		case 6: // hold
			println("stop, torque on (keep 10s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(10 * time.Second)

		case 7: // release
			println("release, no pulse, no torque (keep 10s)")
			each(func(s *servo.Continuous) { s.Release() })
			time.Sleep(10 * time.Second)

		case 8: // ramp
			println("ramp 0 -> 100% in 1.5s, then hold 4s, stop 4s")
			each(func(s *servo.Continuous) {
				s.SetRamp(1500 * time.Millisecond)
				s.Clockwise(100)
			})
			t0 := time.Now()
			for time.Since(t0) < 4*time.Second {
				now := time.Since(t0)
				each(func(s *servo.Continuous) { s.Update(now) })
				time.Sleep(20 * time.Millisecond)
			}
			each(func(s *servo.Continuous) {
				s.SetRamp(0)
				s.Stop()
			})
			time.Sleep(4 * time.Second)

		case 14: // ramp inside the proportional band
			println("SetRange(1000us, 2000us), stop stays 1500us")
			each(func(s *servo.Continuous) {
				if err := s.SetRange(1000*time.Microsecond, 2000*time.Microsecond); err != nil {
					println("range:", err.Error())
				}
				s.Stop()
				s.SetRamp(3 * time.Second)
				s.Clockwise(100)
			})
			println("ramp stop -> 100% in 3s")
			t0 := time.Now()
			for time.Since(t0) < 3500*time.Millisecond {
				now := time.Since(t0)
				each(func(s *servo.Continuous) { s.Update(now) })
				time.Sleep(20 * time.Millisecond)
			}
			println("hold 100% (3s)")
			each(func(s *servo.Continuous) { s.Update(4 * time.Second) })
			time.Sleep(3 * time.Second)
			println("stop (3s)")
			each(func(s *servo.Continuous) {
				s.SetRamp(0)
				s.Stop()
			})
			time.Sleep(3 * time.Second)

		case 13: // narrow range makes 50% real
			println("SetRange(1000us, 2000us)")
			each(func(s *servo.Continuous) {
				if err := s.SetRange(1000*time.Microsecond, 2000*time.Microsecond); err != nil {
					println("range:", err.Error())
				}
			})
			println("clockwise 100% (4s) - pulse 1000us")
			each(func(s *servo.Continuous) { s.Clockwise(100) })
			time.Sleep(4 * time.Second)
			println("clockwise 50% (4s) - pulse 1250us")
			each(func(s *servo.Continuous) { s.Clockwise(50) })
			time.Sleep(4 * time.Second)
			println("clockwise 10% (4s) - pulse 1350us")
			each(func(s *servo.Continuous) { s.Clockwise(10) })
			time.Sleep(4 * time.Second)
			println("stop (4s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(4 * time.Second)

		case 12: // where saturation starts
			println("fixed 1000us (4s) - fast clockwise")
			each(func(s *servo.Continuous) { s.SetPulse(1000 * time.Microsecond) })
			time.Sleep(4 * time.Second)
			println("fixed 1300us (4s) - slow clockwise")
			each(func(s *servo.Continuous) { s.SetPulse(1300 * time.Microsecond) })
			time.Sleep(4 * time.Second)
			println("stop 1500us (4s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(4 * time.Second)

		case 11: // proportional band around the midpoint
			println("fixed 1300us (4s) - slow clockwise")
			each(func(s *servo.Continuous) { s.SetPulse(1300 * time.Microsecond) })
			time.Sleep(4 * time.Second)
			println("fixed 1700us (4s) - slow counterclockwise")
			each(func(s *servo.Continuous) { s.SetPulse(1700 * time.Microsecond) })
			time.Sleep(4 * time.Second)
			println("stop 1500us (4s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(4 * time.Second)

		case 10: // speed A/B: 100% against 50%, clockwise
			println("clockwise 100% (4s) - phase A")
			each(func(s *servo.Continuous) { s.Clockwise(100) })
			time.Sleep(4 * time.Second)
			println("clockwise 50% (4s) - phase B")
			each(func(s *servo.Continuous) { s.Clockwise(50) })
			time.Sleep(4 * time.Second)
			println("stop (4s)")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(4 * time.Second)

		case 9: // constant pulse, jitter check
			println("fixed 1200us, keep 8s")
			each(func(s *servo.Continuous) { s.SetPulse(1200 * time.Microsecond) })
			time.Sleep(8 * time.Second)

		default:
			println("no test selected, holding stop")
			each(func(s *servo.Continuous) { s.Stop() })
			time.Sleep(5 * time.Second)
		}
	}
}

func testName(t int) string {
	switch t {
	case 1:
		return "full speed clockwise"
	case 2:
		return "full speed counterclockwise"
	case 3:
		return "clockwise 50%"
	case 4:
		return "counterclockwise 50%"
	case 5:
		return "clockwise 1% clamped to 10%"
	case 6:
		return "stop with torque"
	case 7:
		return "release without torque"
	case 8:
		return "ramp"
	case 9:
		return "constant pulse 1200us"
	case 10:
		return "speed A/B: 100% vs 50% clockwise"
	case 11:
		return "proportional band: 1300us vs 1700us"
	case 12:
		return "saturation: 1000us vs 1300us"
	case 13:
		return "narrow range 1000-2000us: 100% / 50% / 10%"
	case 14:
		return "ramp inside 1000-2000us band, 3s"
	}
	return "idle"
}
