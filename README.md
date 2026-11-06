# M5Servo-Go

[![test](https://github.com/nicolasmmb/M5Servo-Go/actions/workflows/test.yml/badge.svg)](https://github.com/nicolasmmb/M5Servo-Go/actions/workflows/test.yml)

Pure-Go driver for 50 Hz hobby servos, with the **M5Stack Servo Kit 360°** and
**Servo Kit 180°** (SKU A076-C) as reference hardware. No C, C++, CGo or
Arduino.

The package knows nothing about boards:

- the core depends only on `Set(channel, duty)` and `Top()`;
- the caller passes the PWM peripheral and the pin;
- the native adapter serves any chip whose TinyGo PWM has
  `Configure(Period)` + `Channel(pin)` + `Set` + `Top` (ESP32 and RP2040);
- a board with another PWM API only needs a two-method wrapper.

## Features

- 360° continuous rotation: speed, direction, stop, release.
- 180° positional: degrees, percent, radians.
- Direct pulse in microseconds, plus `Pulse()`, `Target()`, `Resolution()`.
- Same defaults as the official M5Stack drivers: 20 ms period, 500-2500 µs,
  stop at the midpoint, 10 % minimum speed.
- Range and stop-point trim (`SetRange`, `SetStopPulse`) for real servos,
  where the midpoint is not always the stop point.
- Optional ramp (`SetRamp` + `Update`) and `Settling` for travel time.
- `Bank` for up to 8 servos on one timer.
- Zero heap allocation per command. `Servo` is 88 bytes.
- Host tests with a fake PWM, allocation and footprint contracts.

## Hardware

| Parameter | Value |
|---|---|
| Products | Servo Kit 360° (continuous), Servo Kit 180° (positional) |
| Servo | SG90, 9 g |
| Interface | PWM, 50 Hz, no chip, no protocol |
| Pulse | 0.5 ms to 2.5 ms |
| Stop pulse | 1.5 ms (midpoint) |
| Supply | 4.8 V to 6.0 V |
| Current | 60 mA idle, 750 mA in stall |
| Signal wire | Grove yellow |

Wiring: the kit includes a Grove-to-servo adapter. Use the yellow wire as the
signal pin. On a StickC-style board the yellow wire is GPIO32 on PORT A and
GPIO26 on PORT B, but any GPIO works.

Power:

- One 9 g servo on a Grove port is fine.
- Two servos in stall reach about 1.5 A. A Grove port does not supply this.
  Use an external 5 V supply with a common ground.

## Install

```bash
go get github.com/nicolasmmb/M5Servo-Go@v0.1.0
```

## Quick start (any board)

```go
// 1. Pick the PWM peripheral and the pin wired to the servo signal.
pw := machine.PWM0
pin := machine.GPIO32

// 2. The adapter configures 50 Hz and allocates a channel.
s, err := servo.NewNative(pw, pin)
if err != nil {
    println("servo:", err.Error())
    return
}

s.Clockwise(50)   // 50 % of the course, clockwise
time.Sleep(time.Second)
s.Stop()
```

Board with another PWM API, or a custom peripheral:

```go
type myPWM struct{ /* ... */ }

func (p *myPWM) Set(ch uint8, v uint32) { /* write duty */ }
func (p *myPWM) Top() uint32           { return 16383 }

s, _ := servo.NewContinuous(servo.Config{PWM: &myPWM{}, Channel: 0})
```

## API

| Function | Description |
|---|---|
| `NewContinuous(cfg) (*Continuous, error)` | 360° servo, writes `MinPulse` |
| `NewPositional(cfg) (*Positional, error)` | 180° servo, writes `MinPulse` |
| `NewNative(pwm, pin)` / `NewNativePositional(pwm, pin)` | Adapter for ESP32 and RP2040 |
| `ConfigurePeriod(pwm, d) error` | Programs the timer, default 20 ms |
| `AllocateChannel(pwm, pin) (uint8, error)` | Reserves a channel |
| `Clockwise(pct)` / `CounterClockwise(pct)` | 0..100 %, clamped to `MinSpeed` |
| `SetSpeed(int8)` / `Speed() int8` | Signed speed, positive clockwise |
| `Stop()` | Stop pulse, torque on |
| `Release()` | No pulse, no torque |
| `SetPulse(d)` / `Pulse()` / `Target()` | Direct pulse |
| `SetAngle(deg)` / `SetPercent(p)` / `SetRadians(r)` | Positional commands |
| `Angle() float32` | Last commanded angle |
| `SetRange(min, max) error` | Course and stop point, midpoint |
| `SetStopPulse(d) error` | Stop trim |
| `SetRamp(d)` / `Ramp()` / `Update(now)` | Ramp control |
| `Settling(from, to)` | Travel time, 0.15 ms per µs |
| `Resolution()` | Pulse step of the current PWM |
| `NewBank(cfg)` / `AddContinuous(ch)` / `AddPositional(ch)` / `Release()` / `Close()` | Group |

## Parity with the official drivers

| Official | This package |
|---|---|
| `count_low=500`, `count_high=2500` | `DefaultMinPulse`, `DefaultMaxPulse` |
| `freq=50` | `DefaultPeriod = 20ms` |
| `pin = port[1]` | caller passes the pin |
| `set_duty(us)` | `SetPulse(time.Duration)` |
| `set_angle`, `set_percent`, `set_radian` | `SetAngle`, `SetPercent`, `SetRadians` |
| `clockwise`, `counterclockwise`, `stop` | same names |
| 10 % minimum speed | `MinSpeed`, `NoMinSpeed` |
| constructor writes `count_low` | same: `MinPulse` |
| `deinit()` | `Release()` and `Bank.Close()` |
| `wait=True` blocks | `Settling(from, to)` returns the duration |
| `print` on every command | no console output |

## Measured behaviour (real unit)

All commands passed on an M5StickC Plus2 with the real servo. The complete
table is in [BENCHMARKS.md](BENCHMARKS.md). Two findings matter:

1. **This SG90 360° saturates near 1000 µs.** 500 µs and 1000 µs give the same
   speed, and 1250 µs and 1350 µs also give the same speed. The official range
   500-2500 µs therefore gives full speed for most of the course.
2. **A narrow range fixes it.** With `SetRange(1000µs, 2000µs)` the ramp and
   the intermediate speeds become visible.
3. **`Release` was proven electrically.** The firmware reads its own PWM pin
   as an input: `Stop()` shows a 1497 µs pulse at 50 Hz, `Release()` shows
   **zero edges**. By hand both feel the same, because this servo has no
   holding torque inside the deadband.

Recommended setup for smooth speed control:

```go
c, _ := servo.NewContinuous(servo.Config{PWM: pw, Channel: ch})
_ = c.SetRange(1000*time.Microsecond, 2000*time.Microsecond) // stop stays 1500µs
_ = c.SetStopPulse(1500 * time.Microsecond)                  // trim if needed
c.SetRamp(500 * time.Millisecond)
```

## Limitations

- Open loop: a hobby servo has no position feedback.
- Pulse resolution is 1.22 µs at 50 Hz with TinyGo PWM (14 bits). The servo
  deadband is about 8 µs, so this is enough. A 16-bit LEDC setup would need
  direct register access.
- The official `wait` is not implemented as a sleep. Use `Settling`.
- `Release()` removes the pulse (proven: 0 edges on the pin). This servo has
  no holding torque at the stop pulse either, so `Stop` and `Release` feel the
  same by hand. Do not rely on the hand to tell them apart.
- The stop point of a 360° servo is not always the midpoint. Measure it and
  use `SetStopPulse`.

## Validation firmware

`examples/validate` runs **one test per flash**, in a loop, so an operator can
watch a single behaviour and answer a single question. Set `test` in
`examples/validate/config.go`:

| `test` | Behaviour |
|---:|---|
| 1 | full speed clockwise |
| 2 | full speed counterclockwise |
| 3 | clockwise 50 % |
| 4 | counterclockwise 50 % |
| 5 | clockwise 1 % (clamped to 10 %) |
| 6 | stop with torque |
| 7 | release without torque |
| 8 | ramp, official range |
| 9 | fixed pulse 1200 µs |
| 10 | speed A/B: 100 % vs 50 % |
| 11 | proportional band: 1300 µs vs 1700 µs |
| 12 | saturation: 1000 µs vs 1300 µs |
| 13 | narrow range 1000-2000 µs |
| 14 | ramp inside 1000-2000 µs, 3 s |
| 17 | release A/B with motion cues |
| 18 | electrical report: edges and pulse width of its own pin |

The firmware drives every candidate signal pin with the same pulse, so the
Grove port does not matter.

```bash
make validate PORT=/dev/cu.usbserial-XXXX       # flash and monitor
make flash EXAMPLE=spin PORT=/dev/cu.usbserial-XXXX
```

## References

- Servo Kit 360° (A076-C): https://docs.m5stack.com/en/accessory/Servo_Kit_360
- Official drivers: `uiflow-micropython/m5stack/libs/unit/servo180.py` and
  `servo360.py`
- Official example:
  `M5-ProductExampleCodes/Accessory/ServoKit180_360/Arduino/ServoKit180_360.ino`
- TinyGo: `machine_esp32_pwm.go`, `machine_esp32xx_pwm.go`, `machine_rp2_pwm.go`

## License

MIT — see [LICENSE](LICENSE).
