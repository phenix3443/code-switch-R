#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_HOME="${CODE_SWITCH_TEST_HOME:-${TEST_HOME:-$ROOT_DIR/.tmp/test-home}}"
TEST_PORT="${CODE_SWITCH_TEST_PORT:-${TEST_PORT:-18110}}"
TEST_VITE_PORT="${WAILS_VITE_PORT:-9255}"
TEST_GOPATH="${GOPATH:-$(go env GOPATH 2>/dev/null)}"
WAILS3_BIN="${WAILS3_BIN:-}"
COMMAND="${1:-start}"

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

is_running() {
  if [[ ! -f "$PID_FILE" ]]; then
    return 1
  fi
  local pid
  pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [[ -z "$pid" ]]; then
    return 1
  fi
  kill -0 "$pid" >/dev/null 2>&1
}

kill_matching_processes() {
  local pattern="$1"
  local pids
  pids="$(pgrep -f "$pattern" 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    while IFS= read -r pid; do
      [[ -n "$pid" ]] || continue
      kill "$pid" >/dev/null 2>&1 || true
    done <<<"$pids"
  fi
}

export HOME="$TEST_HOME"
export USERPROFILE="$TEST_HOME"
export XDG_CONFIG_HOME="$TEST_HOME/.config"
if [[ -n "$TEST_GOPATH" ]]; then
  export GOPATH="$TEST_GOPATH"
fi
STATE_DIR="$HOME/.code-switch"
PID_FILE="$STATE_DIR/test-app.pid"
LOG_FILE="$STATE_DIR/test-app.log"

case "$COMMAND" in
  stop)
    kill_matching_processes "/Users/liushangliang/go/bin/wails3 dev -config ./build/config.yml -port ${TEST_VITE_PORT}"
    if is_running; then
      pid="$(cat "$PID_FILE")"
      kill "$pid"
      rm -f "$PID_FILE"
      echo "Stopped test app: PID $pid"
    else
      rm -f "$PID_FILE"
      echo "Test app is not running"
    fi
    exit 0
    ;;
  logs)
    if [[ ! -f "$LOG_FILE" ]]; then
      echo "Log file not found: $LOG_FILE" >&2
      exit 1
    fi
    tail -n 200 -f "$LOG_FILE"
    ;;
esac

TEST_VITE_PORT="$(find_available_port "$TEST_VITE_PORT")"
export WAILS_VITE_PORT="$TEST_VITE_PORT"

echo "Starting test app with HOME=$HOME"
echo "Relay port: $TEST_PORT"
echo "Vite port: $TEST_VITE_PORT"
if [[ -n "${GOPATH:-}" ]]; then
  echo "Go path: $GOPATH"
fi
echo "Log file: $LOG_FILE"

if is_running; then
  echo "Test app is already running: PID $(cat "$PID_FILE")"
  exit 0
fi

kill_matching_processes "/Users/liushangliang/go/bin/wails3 dev -config ./build/config.yml -port ${TEST_VITE_PORT}"

cd "$ROOT_DIR"
mkdir -p "$STATE_DIR"
nohup "$WAILS3_BIN" dev -config ./build/config.yml -port "$TEST_VITE_PORT" >>"$LOG_FILE" 2>&1 &
app_pid=$!
echo "$app_pid" >"$PID_FILE"
echo "Started test app in background: PID $app_pid"
