//go:build rp2040

package main

import "machine"

// servoPins are the candidate signal pins. Change the list for your board.
var servoPins = []machine.Pin{machine.GPIO2}
