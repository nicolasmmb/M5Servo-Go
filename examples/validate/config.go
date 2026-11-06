//go:build esp32 || rp2040

package main

// test selects the single validation test to run. Change it and flash again.
//
//	1  full speed clockwise          6  stop with torque
//	2  full speed counterclockwise  7  release without torque
//	3  clockwise 50%                8  ramp 0 -> 100% in 1.5 s
//	4  counterclockwise 50%         9  constant pulse 1200 us
//	5  clockwise 1% (min speed 10%)
const test = 0
