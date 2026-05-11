VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?=
BUILD_LDFLAGS := $(strip -X main.version=$(VERSION) $(LDFLAGS))

.PHONY: build install test

build:
	go build -ldflags "$(BUILD_LDFLAGS)" -o ggo ./cmd/ggo

install: build
	mkdir -p "$${GGO_INSTALL_DIR:-$$HOME/.local/bin}"
	cp ggo "$${GGO_INSTALL_DIR:-$$HOME/.local/bin}/ggo"
	chmod +x "$${GGO_INSTALL_DIR:-$$HOME/.local/bin}/ggo"

test:
	go test ./...
