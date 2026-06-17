#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_HOME="${CODE_SWITCH_TEST_HOME:-${TEST_HOME:-$ROOT_DIR/.tmp/test-home}}"
TEST_PORT="${CODE_SWITCH_TEST_PORT:-${TEST_PORT:-18110}}"

mkdir -p \
  "$TEST_HOME/.code-switch" \
  "$TEST_HOME/.cc-switch" \
  "$TEST_HOME/.agents/skills" \
  "$TEST_HOME/.config"

for platform_dir in "$TEST_HOME/.claude/skills" "$TEST_HOME/.codex/skills"; do
  if [[ -L "$platform_dir" ]]; then
    target="$(readlink "$platform_dir" || true)"
    if [[ "$target" == "$TEST_HOME/.agent/skills" || "$target" == ".agent/skills" ]]; then
      rm -f "$platform_dir"
    fi
  fi
done

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
