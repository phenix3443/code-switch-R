# Skills 统一目录重构 · 可执行实施清单

**配套方案**: [2026-06-16-skills-unification-refactor-plan.md](./2026-06-16-skills-unification-refactor-plan.md)（v0.3）
**创建日期**: 2026-06-16
**状态**: 已实施

> 说明：勾选项为可独立提交的最小步骤。每个阶段末尾的「验收」必须通过才进入下一阶段。涉及 Go 服务签名变更后，统一执行 `wails3 task common:generate:bindings`。每阶段对应一个 Conventional Commits 提交，一个 PR 一个逻辑变更。章节号 §x 指向配套方案。

---

## 阶段 0：状态模型与路径基建（纯后端，无行为变更）

文件：`services/skillservice.go`

- [ ] 0.1 扩展 `skillStore`：新增 `EnabledOverrides map[string]bool`、`Provenance map[string]skillProvenance`、`Migrations []migrationRecord`、`Backups []backupRecord`；保留 `Repos`；标记旧 `Skills map[string]skillState` 为 deprecated（读时兼容，写时不再写入）
- [ ] 0.2 定义 `skillProvenance{ Type string; RepoOwner/RepoName/RepoBranch string }`（`type` ∈ `github|local`）
- [ ] 0.3 新增路径 helper：`getUserSkillsPath() = ~/.agent/skills`、`getPlatformSkillsLinkPath("claude"|"codex")`、`getSkillBackupRoot() = ~/.code-switch/backups/skills`
- [ ] 0.4 `skillStore` 升级迁移：加载旧 `skill.json` 时，把 `Skills` 中的 directory 迁为 `Provenance[dir]={type:local}`（无来源信息），保留 `Repos`
- **验收**：`go test ./services/... -run TestSkill` 编译通过；新字段读写有单测

## 阶段 1：软链接 + 迁移 + 备份引擎

文件：新增 `services/skilllinks.go`（跨平台逻辑）、`services/skilllinks_unix.go` / `services/skilllinks_windows.go`（symlink vs junction，参照 `atomic_write*.go` 的平台拆分）

- [ ] 1.1 `computeSkillFingerprint(dir) (string, error)`：按 §5.3 递归 sha256（排序相对路径 + 内容）
- [ ] 1.2 `GetSkillLinkStatus() SkillLinkStatus`：返回 `~/.agent/skills` 是否存在 + count、claude/codex 链接状态（`linked|missing|conflict|repair_failed|unsupported`）与当前指向
- [ ] 1.3 `backupPlatformDir(platform) (backupRecord, error)`：仅对「存在且为实体目录」备份到 `~/.code-switch/backups/skills/<platform>-<timestamp>/`，完整保留结构
- [ ] 1.4 `EnsureSkillLinks() error`：幂等。健康则跳过；按 §5.3 迁移规则（备份→按目录粒度并集合并→指纹去重/冲突检测→平台目录改名 `.bak`→建链→校验→清理）；Windows 用 junction，失败标 `repair_failed`/`unsupported`，**不复制 fallback**
- [ ] 1.5 同名不同内容 → 记 `conflict`，停止该项自动迁移，写入 `Migrations`
- [ ] 1.6 崩溃恢复：启动时若发现孤儿 `skills.bak-*` 且链接缺失则回滚
- **验收**：新增 `services/skilllinks_test.go` 覆盖：clean / migratable / conflict / 已是正确链 / 指向他处 / 双平台合并去重 / 备份生成；`go test ./services/... -run TestSkillLinks -v` 通过

## 阶段 2：安装/卸载/启用 收敛到 `~/.agent/skills`

文件：`services/skillservice.go`

- [ ] 2.1 `NewSkillService().installDir` 改为 `getUserSkillsPath()`
- [ ] 2.2 重写 `InstallSkill(directory, repoOwner, repoName, repoBranch)`：装前 `EnsureSkillLinks()`；**装前检查目录名是否被其他来源占用→冲突则阻断**（§4.4）；仅写 `~/.agent/skills/<dir>`；持久化 `Provenance[dir]`
- [ ] 2.3 `UninstallSkill(directory)`：仅删 `~/.agent/skills/<dir>`，清 `Provenance`/`EnabledOverrides`
- [ ] 2.4 `ToggleSkill(directory, enabled)`：写 `EnabledOverrides`，并用现有 `patchSkillFrontMatterBool` 投影到 `~/.agent/skills/<dir>/SKILL.md`（§5.5）
- [ ] 2.5 `EnsureSkillLinks()` 内追加：对所有 `EnabledOverrides` 做一次 reconcile（修复手工漂移）
- [ ] 2.6 删除 `getInstallPath`、`ListSkillsForPlatform`、`UninstallSkillEx`、`OpenSkillFolder`、`installFromPathEx` 的 platform/location 形参；移除 project-level 语义
- [ ] 2.7 `GetSkillContent(directory)` / `SaveSkillContent(directory, content)` 去掉 platform/location
- **验收**：`go test ./services/...` 全绿；目录名冲突阻断有单测；toggle 后 `SKILL.md` 实际被改写有单测

## 阶段 3：分组接口 + bindings + 前端服务层

- [ ] 3.1 `Skill` 模型：去掉 `Platform`/`InstallLocation`；新增 `SourceGroupKey`/`SourceGroupLabel`（§5.6）；`RepoOwner/RepoName/RepoBranch` 稳定填充
- [ ] 3.2 `ListInstalledSkills()`：扫 `~/.agent/skills` + 合并 `Provenance`；无来源→`local`/`Local / Unknown Source`（**只认 provenance，不做目录名反查**，§9.2）
- [ ] 3.3 `ListAvailableSkills()`：远程 catalog（复用现有 `prepareRepoSnapshot`）
- [ ] 3.4 `ListGroupedSkills() []SkillGroup`（`{group_key, group_label, skills}`，§6.13），installed/available 各自分组
- [ ] 3.5 `OpenUserSkillsFolder()`；保留按平台「打开链接目录」入口
- [ ] 3.6 `wails3 task common:generate:bindings`
- [ ] 3.7 重写 `frontend/src/services/skill.ts`：改调新方法，删除 `ListSkillsForPlatform`/`UninstallSkillEx`/`OpenSkillFolder` 封装
- **验收**：bindings 生成无误；`frontend` 构建（`wails3 dev` 或 `vite build`）类型检查通过

## 阶段 4：前端 VS Code 风格界面（§6.16 为唯一视觉规范）

文件：`frontend/src/components/Skill/Index.vue` 重写；`SkillCard.vue` 改造为「列表行」+「详情面板」组件；`frontend/src/locales/*` 补 i18n

- [ ] 4.1 左右分栏骨架：左侧列表 + 右侧详情
- [ ] 4.2 左栏顶部：`SKILLS` 标题、刷新、`...` 溢出菜单（管理仓库/打开目录/修复链接/查看备份）、搜索框 + 清除/筛选图标
- [ ] 4.3 顶层折叠组 `LOCAL - INSTALLED`（数量徽标）/`RECOMMENDED`（占位空状态）
- [ ] 4.4 `LOCAL - INSTALLED` 内按 `owner/name` 可折叠子组 + `Local / Unknown Source` 兜底
- [ ] 4.5 列表行：图标/名称/启用·冲突徽标/描述/来源/齿轮菜单（§6.16-C）
- [ ] 4.6 详情区：头部 + 操作按钮行（启用▾/卸载▾/打开目录/打开仓库，冲突禁用）+ Tab（`说明`/`SKILL.md`/`状态&迁移`）+ 右侧信息栏（安装信息/链接状态/资源）
- [ ] 4.7 取消安装弹窗的 user/project 选择，点击直接装到 `~/.agent/skills`；链接异常则阻断并跳「修复链接」
- [ ] 4.8 状态卡（§6.6）：User Skills 目录卡 / Claude 链接卡 / Codex 链接卡 + 颜色约定
- [ ] 4.9 迁移与备份摘要区（§6.7）+ 备份记录弹窗 + 迁移冲突弹窗（§6.15）
- [ ] 4.10 空状态/链接异常/迁移冲突/备份成功失败（§6.14）
- [ ] 4.11 移除 Claude/Codex 平台 tab 与三块旧结构（Project/User/Available）
- **验收**：与截图视觉对照（分组徽标、列表行、详情 Tab、右信息栏均到位）；安装/卸载/启用/修复链接/查看备份 全链路手测通过
  - [x] 实机验证 `修复链接`：隔离 HOME `/tmp/codeswitch-stage4-runhome` 下，桌面点击后 `~/.codex/skills` 恢复为指向 `~/.agent/skills` 的链接，`skill.json` 记录 codex 迁移与备份
  - [x] 实机验证 `查看备份`：桌面弹窗展示 `/tmp/codeswitch-stage4-runhome/.code-switch/backups/skills/...` 备份记录，与 `skill.json` 一致
  - [x] 实机验证 `启用/禁用`：桌面点击后 `demo-skill/SKILL.md` 中 `disable-model-invocation` 在 `true/false` 间切换
  - [x] 实机验证 `安装`：桌面详情区点击 `artifacts-builder` 安装后，`/tmp/codeswitch-stage4-runhome/.agent/skills/artifacts-builder` 新增；运行日志记录 `SkillService.InstallSkill args=["artifacts-builder","ComposioHQ","awesome-claude-skills","master"]`
  - [x] 实机验证 `卸载`：桌面弹出页内确认框后点击 `卸载`，`/tmp/codeswitch-stage4-runhome/.agent/skills/artifacts-builder` 被删除；运行日志记录 `SkillService.UninstallSkill args=["artifacts-builder"]`

## 阶段 5：首启迁移触发 + 收尾

- [x] 5.1 在 `SkillService.ServiceStartup()`（或 main 启动序列）触发一次 `EnsureSkillLinks()` + 迁移诊断，结果存 `Migrations`；`conflict` 不自动执行，仅置状态供 UI 提示
- [x] 5.2 更新 `services/TEST_README.md` 增补 skills 链接/迁移/冲突测试说明
- [x] 5.3 `go test ./services/... -cover` + `go vet` + 前端类型检查全绿（完成定义）
- [x] 5.4 文档同步：配套方案标记「已实施」，如行为变化影响 `CLAUDE.md` 描述则同步
- **验收**：lint/类型检查通过、相关测试通过、文档同步（符合全局「完成定义」）

---

## 跨阶段注意

1. 每阶段独立 PR，Conventional Commits（如 `refactor(skill): 统一 user 级 skills 目录到 ~/.agent/skills`）
2. 破坏性签名变更与 bindings 重生成必须同 PR，避免前后端断档
3. Windows junction/symlink 路径在 macOS 上无法完整验证，需在 Windows 实机或 CI 补测阶段 1
4. 改动最小化：不顺手重构 relay/provider 等无关模块
