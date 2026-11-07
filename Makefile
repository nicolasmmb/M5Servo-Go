TARGET ?= m5stick-c
PORT ?=
TINYGO ?= tinygo
COUNT ?= 8
BENCH ?= .
BENCHTIME ?= 1s

.PHONY: test vet fmt check bench bench-save bench-compare alloc footprint escape profile examples flash spin validate agnostic

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

# Board agnosticism + chip tag coverage, no hardware required.
agnostic:
	@rm -rf /tmp/servo-agn && mkdir -p /tmp/servo-agn && cd /tmp/servo-agn && \
	  MOD=$(shell awk '/^module /{print $$2}' go.mod) && \
	  printf 'module agn\n\ngo 1.26\n\nrequire %s v0.0.0\nreplace %s => %s\n' "$$MOD" "$$MOD" "$(CURDIR)" > go.mod && \
	  printf 'package main\n\nimport lib "%s"\n\nvar _ = lib.NewContinuous\n\nfunc main() {}\n' "$$MOD" > main.go && \
	  go mod tidy >/dev/null && \
	  $(TINYGO) build -target=pico -o /tmp/servo-agn/pico.bin . && \
	  $(TINYGO) build -target=arduino-nano33 -o /tmp/servo-agn/nrf.bin . && \
	  printf 'package main\n\nimport lib "%s"\n\nvar _ = lib.NewNative\n\nfunc main() {}\n' "$$MOD" > main.go && \
	  go mod tidy >/dev/null && \
	  for t in m5stick-c m5stamp-c3 m5stamp-s3a pico; do \
	    $(TINYGO) build -target=$$t -o /tmp/servo-agn/$$t.bin . || exit 1; \
	  done && \
	  echo "agnostic: core pico+nrf OK, NewNative esp32/c3/s3/rp2040 OK"
