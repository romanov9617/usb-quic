BIN_NAME := usb-quic
CMD_PKG := ./cmd/usb-quic
DIST_DIR := dist
BIN_PATH := $(DIST_DIR)/$(BIN_NAME)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION_VAR := usb-quic/internal/adapter/delivery/cli.version
LDFLAGS := -s -w -X $(VERSION_VAR)=$(VERSION)

.PHONY: help build test bench stats clean

help:
	@echo "Available targets:"
	@echo "  make build    Build $(BIN_PATH) with VERSION=$(VERSION)"
	@echo "  make test     Run unit tests"
	@echo "  make bench    Run Go benchmarks with memory stats"
	@echo "  make stats    Print project and test coverage stats"
	@echo "  make clean    Remove build artifacts"

build:
	@mkdir -p $(DIST_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_PATH) $(CMD_PKG)

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./...

stats:
	@echo "Version: $(VERSION)"
	@echo "Go packages: $$(go list ./... | wc -l)"
	@echo "Go files: $$(git ls-files '*.go' | wc -l)"
	@echo "Go lines: $$(git ls-files '*.go' | xargs -r wc -l | tail -n 1 | awk '{print $$1}')"
	@echo "Coverage:"
	@go test -cover ./...
	@if [ -f "$(BIN_PATH)" ]; then \
		echo "Binary: $(BIN_PATH)"; \
		ls -lh "$(BIN_PATH)" | awk '{print "Binary size: " $$5}'; \
	else \
		echo "Binary: not built"; \
	fi

clean:
	rm -rf $(DIST_DIR)
