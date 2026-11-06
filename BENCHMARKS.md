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
| `Release` (no pulse) | Inconclusive by hand: the gear train is stiff |
| Ramp, official range 500-2500 µs | Subtle: half of the ramp is saturated |
| Ramp, range 1000-2000 µs, 3 s | Clear acceleration |
| Fixed pulse 1200 µs for 8 s | Constant speed, no jitter |

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
