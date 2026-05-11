#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${GGO_INSTALL_DIR:-$HOME/.local/bin}"
BIN_PATH="$BIN_DIR/ggo"

cd "$ROOT_DIR"

echo "building ggo..."
VERSION="${GGO_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
go build -ldflags "-X main.version=$VERSION" -o ggo ./cmd/ggo

mkdir -p "$BIN_DIR"
cp "$ROOT_DIR/ggo" "$BIN_PATH"
chmod +x "$BIN_PATH"

echo "installed: $BIN_PATH"

case ":$PATH:" in
  *":$BIN_DIR:"*)
    ;;
  *)
    echo
    echo "$BIN_DIR is not in PATH."
    echo "Add this to your shell profile:"
    echo "  export PATH=\"$BIN_DIR:\$PATH\""
    ;;
esac
