# Benchmarks — M5Servo-Go

## Method

- Host: Apple Silicon (darwin/arm64), Go 1.26.3, `-count=3 -benchtime=200ms`.
- PWM fake (`silentPWM`): it keeps only the last duty, so the benchmark
  measures the driver, not the peripheral.
- No bus traffic exists, so `tx/op` does not apply. The driver writes **one
  duty per command**, and a ramp step writes one duty per change.
- Regenerate with `make bench`. Compare with `make bench-save NAME=before` /
  `make bench-compare FROM=before TO=after`.

## Results (v0.1.0)

| Benchmark | Scenario | ns/op | allocs/op | writes/op |
|---|---|---:|---:|---:|
| `SetPulse` | one pulse command | ~3,3 | 0 | 1 |
| `Clockwise` | mapping + write | ~5,3 | 0 | 1 |
| `SetAngle` | positional path | ~5,2 | 0 | 1 |
| `RampUpdate` | ramp step, no change needed | ~1,9 | 0 | 0 |
| `BankFourServos` | four servos, one command each | ~22,6 | 0 | 4 |

Notes:

- The conversions are integer only. There is no division by a float and no
  `fmt` in the command path.
- `RampUpdate` is cheaper than a command because the step often does not
  change the counts; the driver skips the write in that case.
- At a 50 Hz control loop a command costs ~3 ns. The servo dominates: the
  pulse period is 20 ms.

## Footprint

| Type | Size | Budget in the test |
|---|---:|---:|
| `Servo` | 88 B | ≤ 128 B |
| `Bank` | 32 B | ≤ 64 B |

## Flash (TinyGo, `m5stick-c`)

| Target | code | data | bss | flash | ram |
|---|---:|---:|---:|---:|---:|
| `examples/spin` | 12 408 | 88 | 4 644 | 12 496 B | 4 732 B |
| `examples/validate` | 12 804 | 88 | 4 644 | 12 892 B | 4 732 B |

## Contracts checked

```bash
make alloc      # zero allocation and footprint tests
make escape     # go build -gcflags=-m | grep escapes
```

- `SetPulse`, `Clockwise`, `CounterClockwise`, `SetSpeed`, `SetAngle`,
  `SetPercent`, `SetRadians`, `Release`, `Update`, `Settling`: **0 allocs/op**.
- `NewContinuous`, `NewPositional` and `NewBank`: one allocation at setup.
- No heap escape in the command path.

## Hardware measurements (M5StickC Plus2 + Servo Kit 360°)

Measured with the guided firmware (`examples/validate`), one test per flash:

| Measurement | Result |
|---|---|
| Resolution at 50 Hz | 1220 ns per count (14 bits) |
| Full speed clockwise | OK |
| Full speed counterclockwise | OK |
| Minimum speed clamp (1 % → 10 %) | OK, slow rotation, no buzz |
| Stop with torque (`Stop`) | OK, light holding force |
| `Release` (no pulse) | **0 edges** on the pin: no pulses at all (see below) |
| Ramp, official range 500-2500 µs | Subtle: half of the ramp is saturated |
| Ramp, range 1000-2000 µs, 3 s | Clear acceleration |
| Fixed pulse 1200 µs for 8 s | Constant speed, no jitter |
| Ramp profile (measured pulse) | 1500 → 1250 → 1000 µs, linear |

### Electrical measurement without a scope

The driven pin is still readable as an input (`machine.Pin.Get` works for
output pins), so `examples/validate` test 18 counts edges and measures the high
time of its own PWM:

| State | Edges in 120 ms | High time |
|---|---:|---|
| `Stop()` (1500 µs) | 12 (50 Hz) | 7461 counts → **1497 µs** |
| `Clockwise(100)` (1000 µs) | 12 | 2491–4976 counts → **1000 µs** |
| `Release()` | **0** | **0** |

Calibration is in-block: the two known pulses (1500 and 1000 µs) give the
loop rate, then the third value is scaled with it. The loop rate changes with
interrupt load, so absolute counts from different blocks are not comparable;
ratios inside one block are.

Ramp profile, same method, `SetRange(1000-2000 µs)`, `SetRamp(3s)`, target
100 % (1000 µs):

| Time | High counts | Pulse |
|---:|---:|---:|
| 0 ms | 7461 | 1500 µs |
| 751 ms | 6215 | 1250 µs |
| 1502 ms | 4976 | 1000 µs (target) |
| 2253 ms | 4970 | 1000 µs |
| 3004 ms | 4970 | 1000 µs |

The ramp reaches the target in 1.5 s, which is correct: `Ramp` is the time to
travel the **full** span (2000−1000 = 1000 µs), and this move was half of it.

Conclusion for `Release`: the pulse train really stops. The mechanical test is
useless on this unit because the servo has no holding torque inside its
deadband, so both states feel the same by hand.

### Servo curve of this unit

| Pulses compared | Observation |
|---|---|
| 500 µs vs 1000 µs | Same speed (saturated) |
| 1000 µs vs 1300 µs | 1000 µs clearly faster |
| 1250 µs vs 1350 µs | Same speed |
| 1300 µs vs 1700 µs | Slow clockwise / slow counterclockwise |
| 1500 µs | Stop |

Conclusion: this SG90 360° saturates near 1000 µs and has a narrow
proportional band around 1500 µs. The official range 500-2500 µs gives full
speed for most of the course. For proportional control, use
`SetRange(1000µs, 2000µs)` and trim the stop point with `SetStopPulse`.
