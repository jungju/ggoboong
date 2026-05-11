.PHONY: build install test

build:
	go build -o ggo ./cmd/ggo

install: build
	mkdir -p "$${GGO_INSTALL_DIR:-$$HOME/.local/bin}"
	cp ggo "$${GGO_INSTALL_DIR:-$$HOME/.local/bin}/ggo"
	chmod +x "$${GGO_INSTALL_DIR:-$$HOME/.local/bin}/ggo"

test:
	go test ./...
