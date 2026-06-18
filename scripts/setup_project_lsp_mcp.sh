#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_BIN="$(go env GOPATH)/bin"

echo "[1/4] 检查运行时"
command -v npm >/dev/null 2>&1 || { echo "缺少 npm"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "缺少 go"; exit 1; }

echo "[2/4] 安装 Go LSP: gopls"
if ! GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}" go install golang.org/x/tools/gopls@latest; then
  echo "gopls 安装失败，重试并跳过 sumdb 校验"
  GOSUMDB=off GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}" go install golang.org/x/tools/gopls@latest
fi

echo "[3/4] 安装前端 LSP 依赖"
cd "$ROOT/frontend"
npm install --save-dev typescript typescript-language-server @vue/language-server

echo "[4/4] 验证配置文件"
test -f "$ROOT/.codex/config.toml"
test -f "$ROOT/.codex/cclsp.json"
test -x "$GO_BIN/gopls" || { echo "gopls 未安装成功"; exit 1; }
test -d "$ROOT/frontend/node_modules/typescript-language-server"
test -d "$ROOT/frontend/node_modules/@vue/language-server"

cat <<EOF

已完成项目级 LSP MCP 准备。

后续使用方式：
1. 在仓库根目录启动 Codex，使其读取项目级 .codex/config.toml
2. MCP server 名称为: project-lsp
3. LSP 能力由 cclsp 提供，覆盖 Go / TS / Vue

如果要给 Claude 也复用这套配置，可在 Claude 的项目级 MCP 配置中同样指向：
  command = npx
  args    = -y cclsp@latest
  env.CCLSP_CONFIG_PATH = $ROOT/.codex/cclsp.json

说明：
- Go LSP 使用 $GO_BIN/gopls
- TS/Vue LSP 使用仓库 frontend/node_modules 内的依赖

EOF
