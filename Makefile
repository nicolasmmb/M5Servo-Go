TARGET ?= m5stick-c
PORT ?=
TINYGO ?= tinygo
COUNT ?= 8
BENCH ?= .
BENCHTIME ?= 1s

.PHONY: test vet fmt check bench bench-save bench-compare alloc footprint escape profile examples flash spin validate

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

check: vet test alloc examples

bench:
	COUNT=$(COUNT) BENCH=$(BENCH) BENCHTIME=$(BENCHTIME) ./scripts/bench.sh run

bench-save:
	COUNT=$(COUNT) BENCH=$(BENCH) BENCHTIME=$(BENCHTIME) ./scripts/bench.sh save $(NAME)

bench-compare:
	./scripts/bench.sh compare $(FROM) $(TO)

alloc:
	./scripts/bench.sh alloc

footprint:
	go test -run TestFootprint -v

escape:
	./scripts/bench.sh escape

profile:
	./scripts/bench.sh profile

examples:
	$(TINYGO) build -target=$(TARGET) -size short -o /tmp/m5servo-spin.bin ./examples/spin
	$(TINYGO) build -target=$(TARGET) -size short -o /tmp/m5servo-validate.bin ./examples/validate

# make flash EXAMPLE=spin PORT=/dev/cu.usbmodem1234
flash:
	$(TINYGO) flash -target=$(TARGET) -port=$(PORT) -size short ./examples/$(EXAMPLE)

# Timed validation sequence; watch the servo.
validate:
	$(TINYGO) flash -target=$(TARGET) -port=$(PORT) -size short -monitor ./examples/validate
