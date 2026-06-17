#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_HOME="${CODE_SWITCH_TEST_HOME:-${TEST_HOME:-$ROOT_DIR/.tmp/test-home}}"
TEST_PORT="${CODE_SWITCH_TEST_PORT:-${TEST_PORT:-18110}}"

mkdir -p \
  "$TEST_HOME/.code-switch" \
  "$TEST_HOME/.cc-switch" \
  "$TEST_HOME/.config"

cat > "$TEST_HOME/.code-switch/network.json" <<EOF
{
  "listenMode": "localhost",
  "relayPort": ${TEST_PORT},
  "currentAddress": "127.0.0.1:${TEST_PORT}",
  "wslAutoConfig": false,
  "targetCli": {
    "claudeCode": true,
    "codex": true,
    "gemini": true
  }
}
EOF

echo "Prepared isolated test home: $TEST_HOME"
echo "Configured relay port: $TEST_PORT"
