#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_HOME="${CODE_SWITCH_TEST_HOME:-${TEST_HOME:-$ROOT_DIR/.tmp/test-home}}"
TEST_PORT="${CODE_SWITCH_TEST_PORT:-${TEST_PORT:-18110}}"
TEST_VITE_PORT="${WAILS_VITE_PORT:-9255}"
TEST_GOPATH="${GOPATH:-$(go env GOPATH 2>/dev/null)}"
WAILS3_BIN="${WAILS3_BIN:-}"

if [[ -z "$WAILS3_BIN" ]]; then
  if command -v wails3 >/dev/null 2>&1; then
    WAILS3_BIN="$(command -v wails3)"
  else
    GOPATH_BIN="$(go env GOPATH 2>/dev/null)/bin/wails3"
    if [[ -x "$GOPATH_BIN" ]]; then
      WAILS3_BIN="$GOPATH_BIN"
    else
      echo "wails3 not found in PATH or GOPATH/bin" >&2
      exit 127
    fi
  fi
fi

find_available_port() {
  local start_port="$1"
  local port="$start_port"
  while [[ "$port" -le 65535 ]]; do
    if ! lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
      echo "$port"
      return 0
    fi
    port=$((port + 1))
  done
  return 1
}

export PATH="$(dirname "$WAILS3_BIN"):$PATH"

"$ROOT_DIR/scripts/prepare-test-home.sh"

export HOME="$TEST_HOME"
export USERPROFILE="$TEST_HOME"
export XDG_CONFIG_HOME="$TEST_HOME/.config"
if [[ -n "$TEST_GOPATH" ]]; then
  export GOPATH="$TEST_GOPATH"
fi
TEST_VITE_PORT="$(find_available_port "$TEST_VITE_PORT")"
export WAILS_VITE_PORT="$TEST_VITE_PORT"

echo "Starting test app with HOME=$HOME"
echo "Relay port: $TEST_PORT"
echo "Vite port: $TEST_VITE_PORT"
if [[ -n "${GOPATH:-}" ]]; then
  echo "Go path: $GOPATH"
fi

cd "$ROOT_DIR"
exec "$WAILS3_BIN" dev -config ./build/config.yml -port "$TEST_VITE_PORT"
