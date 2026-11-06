//go:build esp32

package main

import "machine"

// servoPins are the candidate signal pins. The firmware drives all of them, so
// only the pin wired to the servo matters. Change the list for your board.
// On M5 boards the signal is the yellow Grove wire of the port.
var servoPins = []machine.Pin{machine.GPIO32, machine.GPIO26}
