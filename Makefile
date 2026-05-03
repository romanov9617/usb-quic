BINARIES := usb-quic daemon
DIST_DIR := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
CLI_VERSION_VAR := usb-quic/internal/adapter/delivery/cli.version
DAEMON_VERSION_VAR := usb-quic/internal/adapter/delivery/daemon.version
LDFLAGS := -s -w -X $(CLI_VERSION_VAR)=$(VERSION) -X $(DAEMON_VERSION_VAR)=$(VERSION)

.PHONY: help build test bench stats clean

help:
	@echo "Available targets:"
	@echo "  make build    Build binaries into $(DIST_DIR) with VERSION=$(VERSION)"
	@echo "  make test     Run unit tests"
	@echo "  make bench    Run Go benchmarks with memory stats"
	@echo "  make stats    Print project and test coverage stats"
	@echo "  make clean    Remove build artifacts"

build:
	@mkdir -p $(DIST_DIR)
	@for bin in $(BINARIES); do \
		go build -trimpath -ldflags "$(LDFLAGS)" -o "$(DIST_DIR)/$$bin" "./cmd/$$bin"; \
	done

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
	@for bin in $(BINARIES); do \
		path="$(DIST_DIR)/$$bin"; \
		if [ -f "$$path" ]; then \
			echo "Binary: $$path"; \
			ls -lh "$$path" | awk '{print "Binary size: " $$5}'; \
		else \
			echo "Binary: $$path not built"; \
		fi; \
	done

clean:
	rm -rf $(DIST_DIR)
