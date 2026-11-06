//go:build esp32

package main

import "machine"

// servoPin is the pin wired to the servo signal. Change it for your board.
// On M5 boards the signal is the yellow Grove wire.
const servoPin = machine.GPIO32
