#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_HOME="${CODE_SWITCH_TEST_HOME:-${TEST_HOME:-$ROOT_DIR/.tmp/test-home}}"
TEST_PORT="${CODE_SWITCH_TEST_PORT:-${TEST_PORT:-18110}}"
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

export PATH="$(dirname "$WAILS3_BIN"):$PATH"

"$ROOT_DIR/scripts/prepare-test-home.sh"

export HOME="$TEST_HOME"
export USERPROFILE="$TEST_HOME"
export XDG_CONFIG_HOME="$TEST_HOME/.config"

echo "Starting test app with HOME=$HOME"
echo "Relay port: $TEST_PORT"

cd "$ROOT_DIR"
exec "$WAILS3_BIN" dev -config ./build/config.yml -port "${WAILS_VITE_PORT:-9245}"
