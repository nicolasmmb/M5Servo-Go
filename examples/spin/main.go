//go:build esp32 || rp2040

// Example: sweep a 360° servo like the official Arduino sketch does.
//
// The package has no board knowledge. This example only needs two things from
// you: a PWM peripheral (machine.PWM0 here) and the pin wired to the servo
// signal (servoPin, defined per chip family in pins_*.go).
package main

import (
	"time"

	"machine"

	servo "github.com/nicolasmmb/M5Servo-Go"
)

func main() {
	time.Sleep(time.Second)

	s, err := servo.NewNative(machine.PWM0, servoPin)
	if err != nil {
		println("servo:", err.Error())
		return
	}
	println("resolution:", int64(s.Resolution()), "ns")
	println("settling full range:", int64(s.Settling(servo.DefaultMinPulse, servo.DefaultMaxPulse)), "ns")

	for {
		for _, speed := range []uint8{25, 50, 75, 100} {
			s.Clockwise(speed)
			println("cw", speed, "% pulse", int64(s.Pulse()))
			time.Sleep(600 * time.Millisecond)
		}
		s.Stop()
		println("stop")
		time.Sleep(800 * time.Millisecond)

		for _, speed := range []uint8{25, 50, 75, 100} {
			s.CounterClockwise(speed)
			println("ccw", speed, "% pulse", int64(s.Pulse()))
			time.Sleep(600 * time.Millisecond)
		}
		s.Stop()
		println("stop")
		time.Sleep(800 * time.Millisecond)
	}
}
