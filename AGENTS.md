# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## What this is

**Code Switch** is a Wails v3 desktop app (Go backend + Vue 3 frontend) that runs a **local HTTP relay** in front of AI coding CLIs (Codex, Codex, Gemini CLI). It manages multiple API providers, automatically fails over between them on errors, rewrites model names per provider, and records request/token/cost statistics. The app injects its local relay address into each CLI's config so requests transparently route through it.

## Build & run

Uses [Task](https://taskfile.dev) wrapping the `wails3` CLI. **The `wails3` CLI version is pinned** — Linux release builds broke when unpinned (see commit `0c8e5c4`).

```bash
wails3 dev          # Dev mode: Vite HMR + Go hot reload (see build/config.yml dev_mode)
wails3 task build   # Build for current OS (delegates to build/<os>/Taskfile.yml)
wails3 task package # Production package
wails3 task run     # Run built binary
make test-app       # 用仓库内隔离 HOME 启动测试实例（默认 ./.tmp/test-home, relay 18110）
```

- Frontend deps install + bindings generation are wired as Task dependencies; you rarely call them directly. To regenerate frontend bindings after changing Go service signatures: `wails3 task common:generate:bindings`.
- macOS builds require **CGO_ENABLED=1** (sqlite via `modernc.org/sqlite` plus native deps); Linux release builds need GTK4 system deps (commit `a0502e1`).

## Tests

```bash
go test ./services/...                          # all backend tests
go test ./services/... -run TestMatchWildcard -v # single test by name
go test ./services/... -cover                   # coverage
go test ./services/... -bench=. -benchmem        # benchmarks
```

Test focus areas (see `services/TEST_README.md`): model wildcard matching, model-support/mapping logic (`providerservice_test.go`), end-to-end relay request handling (`providerrelay_test.go`), provider rename, concurrent DB writes, and the skills unification flow (`skillservice_test.go`, `skilllinks_test.go`). Frontend has no test runner configured.

## Architecture

### Service registration (the spine)
`main.go` constructs every service, wires their dependencies by hand (constructors take other services as args), and registers them with `application.New({ Services: [...] })`. Each `*Service` struct's exported methods are auto-exposed to the frontend as TypeScript bindings under `frontend/bindings/`. **Adding a backend method callable from the UI = add an exported method to a registered service, then regenerate bindings.** In Wails v3, lifecycle hooks are `ServiceStartup(ctx, options)` / `ServiceShutdown()`.

### The relay (`services/providerrelay.go`, ~80KB — the core)
A Gin HTTP server started in a goroutine from `main.go` (`providerRelay.Start()`). Routes in `registerRoutes`:
- `POST /v1/messages` → Codex (Anthropic protocol)
- `POST /responses` + `GET /v1/models` → Codex
- `POST /gemini/v1beta/*` , `/gemini/v1/*` → Gemini
- `POST /custom/:toolId/v1/messages` → user-defined custom CLIs

`proxyHandler` → `forwardRequest` implements the failover engine:
1. Load providers for the platform (`kind`), filter to enabled ones.
2. Skip blacklisted providers (`blacklistService.IsBlacklisted`).
3. Group by **Level** (1 = highest priority … 10 = fallback); try each level in order.
4. Within a level, optionally **round-robin** across providers.
5. **Model-aware downgrade**: a provider is only tried if it supports the requested model (`supportedModels` whitelist / `modelMapping`); model names are rewritten per provider before forwarding.
6. On upstream failure, blacklist the provider (with retry/recovery config) and advance.

SSE responses stream through hooks (`ReqeustLogHook`, `protocolConvertHook`) that accumulate bytes and parse token usage on the fly. `protocol_adapter.go` converts OpenAI-style SSE ↔ Anthropic when a provider speaks a different protocol than the client.

### Config injection & restoration (`proxystate.go`)
Enabling the proxy edits the real CLI config files (Codex `settings.json`, Codex `config.toml`/`auth.json`, Gemini) to point at the local relay. To avoid clobbering user config, it records a **`ProxyState`** baseline per platform at `~/.code-switch/proxy-state/{platform}.json`: which keys existed, their original values, and what was injected. Disabling performs *surgical* restoration (only reverting values still equal to what was injected) rather than overwriting whole files. CLI-specific read/write logic lives in `claudesettings.go`, `codexsettings.go`, `geminiservice.go`, `cliconfigservice.go`, `customcliservice.go`.

### Persistence
- **Provider configs**: JSON files under `~/.code-switch/` — `Codex.json`, `codex.json`, and `providers/{toolId}.json` for custom CLIs (`providerservice.go: providerFilePath`). Atomic writes via `atomic_write*.go` (platform-split for Windows).
- **SQLite** at `~/.code-switch/app.db` (`database.go: InitDatabase`) for request logs, blacklist, settings. All writes go through `dbqueue.go`'s **two global write queues** to eliminate `SQLITE_BUSY`: `GlobalDBQueue` (single-op, heterogeneous: blacklist/settings) and `GlobalDBQueueLogs` (batched, request_log only — 50/batch, 100ms flush). Use these queues for writes; do not write to the DB directly from handlers.

### Frontend (`frontend/`)
Vue 3 + TypeScript + Vite + Tailwind v4, hash-routed (`router/index.ts`). One page-component directory per feature under `components/` (Main, Logs, Mcp, Skill, Prompts, Availability, SpeedTest, EnvCheck, Console, Setting, Gemini, Tray). `frontend/src/services/*.ts` are thin wrappers over the generated `bindings/`. i18n via `vue-i18n` (`locales/`), charts via `chart.js`/`vue-chartjs`, config editing via CodeMirror. `Tray/` renders the macOS tray popover window (a separate Wails window driven from `main.go`).

### Skills (`services/skillservice.go`, `services/skilllinks.go`, `frontend/src/components/Skill/`)
Skills are now unified under `~/.agents/skills`. Claude and Codex no longer maintain separate physical skill directories; instead their entrypoints should link to the unified directory:
- `~/.claude/skills -> ~/.agents/skills`
- `~/.codex/skills -> ~/.agents/skills`

`SkillService.ServiceStartup()` runs `EnsureSkillLinks()` once at startup to repair links, migrate legacy platform directories, reconcile enabled overrides back into `SKILL.md`, and persist backup/migration diagnostics for the UI. Conflicts are recorded into `Migrations` for the Skills page to surface rather than blocking app startup.

## Conventions

- **Code & comments are in Chinese.** Match the surrounding language and style.
- Version lives in `version_service.go` (`AppVersion = "vX.Y.Z"`) and `build/config.yml`; releases are tagged via `scripts/publish_release.sh`. Update service consumes `AppVersion`.
- The repo root contains many transient artifacts (`test_*.exe`, `temp.txt`, ad-hoc `scripts/*.go` debug programs, version-stamped `*_PLAN/*_NOTES/*.md`). These are not part of the build — don't treat them as canonical.

## UI 迭代约束

- 当用户要求“参考某个现有产品/页面”做界面时，必须优先按参考对象的**信息架构、分组标题、层级、间距、交互位置**对齐，不要先做风格化发挥。
- 不要把参考界面的“列表标题”“详情信息”“资源链接”“操作入口”混成新的自定义布局；优先保持原有职责边界，只替换为当前产品的领域文案。
- Wails 桌面端调用与浏览器调试模式要分开处理。对后端 binding 调用，不要默认套前端超时包装，否则容易把成功结果误判成“加载失败”。
- Skills 页面属于高密度信息界面：默认使用**细线分割、扁平列表、紧凑间距**；除非用户明确要求，不要引入大圆角卡片、大面积强调色按钮、重复信息块。
- 当同一信息已经在列表区展示过，详情区不要再次用另一种结构重复展示；详情区应该补充“选中项的完整信息”，而不是复制列表摘要。
- 推荐列表中的未实现功能，优先做成**菜单壳或占位入口**，不要提前接入错误的主按钮交互，更不要伪造完整可用流程。
- 每次完成前端视觉改动后，必须先自行 CR 一次实际界面效果，再向用户汇报。至少检查：分组标题是否对齐、信息是否重复、图标是否正确、空状态/错误态是否被误触发、参考布局是否被改形。
- 如果用户明确给了截图标注，优先逐条消除截图中的差异，不要在同一轮里擅自删除字段或改写成新的交互模型。

## Notes for this environment

- `~/.Codex/AGENTS.md` (user global) defines Git/commit rules: branches `type/short-description`, Conventional Commits, one logical change per PR, minimal diffs (no unrequested refactors/fallbacks/abstractions), and the **Fix-to-Code Protocol** (any manual fix must be written back into source/script/config immediately).
- Shell commands are auto-rewritten through `rtk` (token-optimizing proxy) via a Codex hook; this folds function bodies in `cat`/`grep` output. Use `rtk proxy <cmd>` to bypass filtering when you need raw output.
