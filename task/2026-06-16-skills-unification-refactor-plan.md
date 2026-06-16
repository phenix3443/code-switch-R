# Skills 统一目录与分类展示重构方案

**文档版本**: v0.3
**创建日期**: 2026-06-16
**作者**: Codex（v0.3 CR 修订：Claude）
**状态**: 已实施

> **v0.3 变更摘要**：修正 §5.5 启用/禁用「不写 SKILL.md」的功能性缺陷（必须投影回 `SKILL.md`）；澄清 §7.2 接口的新增/破坏性变更/已存在分类；新增 §9.3 Codex 路径前置验证；补充内容指纹判定与软链替换原子性（§5.3）；按「与 VS Code 扩展面板 1:1 对齐」重写展示规范（§6.16）。

---

## 一、目标

本次重构包含两个直接目标：

1. 将 Claude 和 Codex 的 skills 统一到同一个 user 级 skills 目录，不再按平台分别维护两份物理目录。
2. 在 Skills 页面按 GitHub 仓库分组展示 skills。

本次方案不追求引入通用插件系统，也不顺手重构其他无关模块。重点是收敛 skills 的物理存储结构、同步机制和展示模型。

---

## 二、现状

当前项目中的 skills 管理位于 `services/skillservice.go`，其核心假设是：

1. skills 按平台分别安装：
   - Claude: `~/.claude/skills` 或 `./.claude/skills`
   - Codex: `~/.codex/skills` 或 `./.codex/skills`
2. 前端通过两个接口拼接数据：
   - `ListSkillsForPlatform(platform)` 返回当前平台已安装技能
   - `ListSkills()` 返回仓库中可安装的技能
3. 技能的启用/禁用通过直接修改 `SKILL.md` front matter 中的 `disable-model-invocation` 字段实现。
4. 仓库配置为全局配置，不区分 Claude/Codex。

这套设计的问题是：

1. 它把“平台”和“技能内容”绑定在一起，但你的新需求明确要求 skills 本身不分平台。
2. 同一个 skill 现在会被重复安装到不同平台目录，维护成本高，且容易漂移。
3. 前端列表是“已安装”和“可安装”两套来源拼接出来的，分类能力弱。
4. 后端仍保留明显偏 Claude 的旧逻辑，例如默认安装目录和 `ListSkills()` 的本地 installed 判定只看 `~/.claude/skills`。

---

## 三、关键决策

### 3.1 固定 user 级 skills 目录

user 级 skills 目录固定为：

- `~/.agent/skills`

约束如下：

1. 应用内部统一只认 `~/.agent/skills`
2. 所有迁移、扫描、安装、卸载、状态存储和 UI 展示都基于该目录

### 3.2 Claude/Codex 与 user 级 skills 目录的关系

Claude 与 Codex 的 user 级 skills 目录统一指向：

- `~/.claude/skills -> ~/.agent/skills`
- `~/.codex/skills -> ~/.agent/skills`

约束如下：

1. Claude 与 Codex 不再各自拥有独立 skills 内容。
2. 所有安装、卸载、编辑、启用/禁用操作，最终都应该落到 `~/.agent/skills`。
3. “同步”本质上不是内容同步，而是“链接关系修复与保持一致”。
4. 平台差异只存在于“入口目录是否正确链接到 `~/.agent/skills`”，不再存在于 skill 内容本身。

---

## 四、重构后的目标模型

### 4.1 统一目录模型

统一的 user 级 skills 目录：

- `~/.agent/skills`

引入两个平台侧入口目录：

- `~/.claude/skills`
- `~/.codex/skills`

目标状态：

1. `~/.agent/skills` 存在
2. `~/.claude/skills` 是指向 `~/.agent/skills` 的软链接
3. `~/.codex/skills` 是指向 `~/.agent/skills` 的软链接

如果平台目录已经存在但不是软链接，则进入迁移流程。

### 4.2 数据视角

统一后，技能应分为两种视角，而不是按平台拆分：

1. **Installed Skills**
   - 来源：`~/.agent/skills`
   - 唯一事实来源
2. **Available Skills**
   - 来源：配置的 GitHub 仓库扫描结果

平台不再是 skills 实体的维度，而只是“链接状态”的维度：

1. Claude 链接是否健康
2. Codex 链接是否健康

### 4.3 展示视角

Skills 页面展示分为两层：

1. 顶层视图切换
   - 已安装
   - 可安装
2. 组内分类方式
   - `按 GitHub 仓库分组`

示例：

- `mattpocock/skills`
  - skill-a
  - skill-b
- `anthropics/skills`
  - skill-c

对于本地已安装但无法映射回仓库来源的 skills，单独归入：

- `Local / Unknown Source`

### 4.4 安装目录唯一性

`~/.agent/skills` 下的安装目录名是 installed skill 的唯一主键。

约束如下：

1. 每个 installed skill 对应一个唯一目录名
2. 同名目录视为同一个 installed skill
3. 若两个不同来源的 skill 使用相同目录名：
   - 不自动重命名
   - 不允许覆盖安装
   - 直接标记为目录名冲突
4. 启用状态覆盖、provenance、迁移记录都以目录名作为关联键

因此，安装流程在写入前必须先检查目标目录名是否已被其他来源占用。

---

## 五、后端重构方案

### 5.1 SkillService 职责调整

当前 `SkillService` 同时负责：

1. 本地目录扫描
2. 远程 repo 下载
3. 安装/卸载
4. 目录打开
5. 启用/禁用
6. repo 配置管理

本次不强制拆多文件，但职责上明确分成三块内部逻辑：

1. **User skills directory management**
   - user 级 skills 目录定位
   - 平台软链接检测
   - 平台软链接修复
   - 首次迁移
2. **Installed skill management**
   - 扫描 `~/.agent/skills`
   - 读取 `SKILL.md`
   - 读取本地状态覆盖
   - 安装/卸载
   - 维护已安装 skill 来源信息
3. **Remote catalog management**
   - repo 配置
   - zip 下载
   - skill 元数据扫描
   - repo 分组信息构建

### 5.2 新路径规则

增加统一路径解析函数，包含以下概念：

- `getUserSkillsPath() -> ~/.agent/skills`
- `getPlatformSkillsLinkPath("claude") -> ~/.claude/skills`
- `getPlatformSkillsLinkPath("codex") -> ~/.codex/skills`

旧的 `getInstallPath(platform, location)` 应逐步退场。

本次重构直接移除 project-level skills 语义。

原因如下：

1. 单一 user 级 skills 目录 + 双平台软链接已经足够构成单一模型
2. 保留 project-level 会重新引入多事实来源
3. 现有前端和后端路径模型会继续复杂化

本方案不保留 project-level skills。

### 5.3 软链接同步机制

增加一个显式同步/修复动作：

- `EnsureSkillLinks()`

该动作负责：

1. 确保 `~/.agent/skills` 存在
2. 确保 `~/.claude`、`~/.codex` 父目录存在
3. 检查 `~/.claude/skills` 是否为指向 `~/.agent/skills` 的软链接
4. 检查 `~/.codex/skills` 是否为指向 `~/.agent/skills` 的软链接
5. 若不是，则按迁移规则修复

迁移规则如下：

1. 若平台目录不存在：
   - 直接创建软链接
2. 若平台目录已是正确软链接：
   - 不处理
3. 若平台目录已存在且是实体目录：
   - 先备份原始平台目录
   - 将目录中的每个 skill 子目录提取为候选迁移项
   - 按 skill 目录名执行并集合并，而不是整目录覆盖
   - 若目标 skill 在 `~/.agent/skills` 中不存在，则迁入
   - 若目标 skill 在 `~/.agent/skills` 中已存在，则标记为同名冲突
   - 仅当该平台目录内所有 skill 都成功迁入或可忽略时，才将平台目录替换为软链接
4. 若平台目录是指向其他路径的软链接：
   - 记录当前目标路径
   - 若该目标与 `~/.agent/skills` 一致，则不处理
   - 否则作为冲突状态处理

本次重点不是“后台自动持续同步”，而是“`~/.agent/skills` 与平台入口目录的一致性修复”。

#### 双平台目录迁移规则

迁移时必须同时考虑：

- `~/.claude/skills`
- `~/.codex/skills`

迁移流程：

1. 为“存在且为实体目录”的平台 skills 目录创建原始目录备份
2. 收集两个平台目录中的全部 skill 目录名
3. 生成候选迁移清单
4. 以 skill 目录名为粒度执行并集合并
5. 若同名 skill 在多个来源中同时存在：
   - 若内容校验一致，可视为重复项，只保留 `~/.agent/skills` 中的一份
   - 若内容不一致，标记为冲突，停止自动迁移并提示用户处理
6. 仅在两个平台目录都完成迁移后，才统一替换为软链接

备份规则：

1. 仅对“存在且为实体目录”的平台 skills 目录执行备份
2. 备份发生在任何迁移、合并、替换操作之前
3. 备份保留原目录完整内容与目录结构
4. 备份目录位于 `~/.code-switch/backups/skills/`
5. 备份目录名称包含平台与时间戳，例如：
   - `~/.code-switch/backups/skills/claude-20260616-223000/`
   - `~/.code-switch/backups/skills/codex-20260616-223000/`
6. 备份完成后才允许进入迁移和软链接替换流程

这里的关键约束是：

1. 不允许按目录整体覆盖
2. 不允许迁移顺序决定结果
3. 不允许静默吞掉同名不同内容的 skill

**「内容一致」的判定（CR 补充）**：对一个 skill 目录递归计算指纹——收集目录内全部文件，按相对路径排序，对「相对路径 + 文件字节内容」做 sha256 累积。两个 skill 目录指纹相同即视为内容一致（可去重）；不同即为同名冲突，停止自动迁移。不使用 mtime/size 这类弱判据。

**软链接替换的原子性与崩溃恢复（CR 补充）**：用软链替换实体目录不是原子操作，必须按以下顺序，保证任一步崩溃都不丢数据：

1. 备份（已存在且为实体目录的平台目录）
2. 将平台目录中的 skill 按目录粒度合并进 `~/.agent/skills`
3. 校验合并结果（指纹比对）通过
4. 将平台目录改名为 `<platform>/skills.bak-<timestamp>`（**不直接删除**）
5. 创建 `<platform>/skills -> ~/.agent/skills` 软链接
6. 校验软链指向正确后，再清理 `.bak`（或保留交由「查看备份」入口手动清理）

崩溃恢复：若发现 `skills.bak-*` 存在但 `skills` 软链缺失/损坏，则回滚到 `.bak`。

### 5.4 安装与卸载行为

安装 skill 时：

1. 仅写入 `~/.agent/skills/<directory>`
2. 安装前先执行 `EnsureSkillLinks()`
3. 安装完成后无需对 Claude/Codex 分别复制或写入

卸载 skill 时：

1. 仅删除 `~/.agent/skills/<directory>`
2. 不再接受 platform/location 作为卸载定位条件

> **实现注意（CR 补充）**：现有 `NewSkillService` 的 `installDir` 硬编码为 `~/.claude/skills`（`skillservice.go:117`），`isInstalled` / `mergeLocalSkills` / `UninstallSkill` 都依赖它。本次必须改为 `getUserSkillsPath() = ~/.agent/skills`，并清理所有对 `getInstallPath(platform, location)` 的调用点（`ListSkillsForPlatform`、`installFromPathEx`、`UninstallSkillEx`、`OpenSkillFolder`、`GetSkillContent`、`SaveSkillContent`、`ToggleSkill`）。

### 5.5 启用/禁用行为

> **关键约束（CR 修正）**：Claude Code / Codex 判断一个 skill 是否启用，**唯一依据是 `SKILL.md` front matter 中的 `disable-model-invocation`**。code-switch 自身不参与 skill 的实际调用，因此**任何只写本地 sidecar、不回写 `SKILL.md` 的方案，对真实 CLI 行为零影响**——开关会沦为纯装饰。故启用/禁用**必须最终落到 `SKILL.md`**。

采用「sidecar 为期望状态(desired state) + 投影回 `SKILL.md`」的双层模型：

1. **sidecar 是期望状态的事实来源**：在 `~/.code-switch/skill.json` 中记录
   - `enabled_overrides[directory] = true|false`
2. **`SKILL.md` 是派生投影,不是用户状态**：code-switch 负责把 override 投影到 `SKILL.md` 的 `disable-model-invocation` 字段（`enabled=true → disable-model-invocation=false`），复用现有的最小文本补丁 `patchSkillFrontMatterBool`，保留原有格式/注释/字段顺序。

投影时机（reconcile）：

1. 用户在 UI 切换启用状态时 → 立即写 sidecar，并投影到 `SKILL.md`
2. 执行 `EnsureSkillLinks()` 时 → 对所有有 override 的 skill 做一次 reconcile，修复用户手工改动造成的漂移

启用状态的读取（展示）计算规则：

1. 若存在本地 override，则以 override 为准
2. 若不存在 override，则读取 `SKILL.md` 当前值

效果与代价：

1. 由于 `~/.claude/skills` 与 `~/.codex/skills` 软链到同一份物理 `SKILL.md`，**Claude 与 Codex 必然共享同一启用状态**。
2. **能力收窄（显式声明）**：旧模型下可对 Claude 启用、对 Codex 禁用；统一目录后不再支持按平台分别启用。这是 unify 的必然代价，符合「skills 不分平台」目标。
3. sidecar 让启用状态在 `SKILL.md` 被用户手工覆盖后仍可恢复；但 `SKILL.md` 仍会被 code-switch 写入（这是让 CLI 生效的前提，无法规避）。

### 5.6 仓库来源字段

要支持按 GitHub 仓库分组展示，后端返回的 `Skill` 模型需要稳定包含：

- `repo_owner`
- `repo_name`
- `repo_branch`

新增前端可直接使用的字段：

- `source_group_key`
- `source_group_label`

例如：

- `source_group_key = "mattpocock/skills"`
- `source_group_label = "mattpocock/skills"`

对本地未知来源 skill：

- `source_group_key = "local"`
- `source_group_label = "Local / Unknown Source"`

前端不再需要自行拼接分组标签。

### 5.7 已安装 skill 的来源持久化

仅靠扫描 `~/.agent/skills` 无法在应用重启后恢复：

- `repo_owner`
- `repo_name`
- `repo_branch`

安装时必须持久化 provenance 信息。

本地状态中为每个 installed skill 记录：

```json
{
  "directory": "example-skill",
  "source": {
    "type": "github",
    "repo_owner": "mattpocock",
    "repo_name": "skills",
    "repo_branch": "main"
  }
}
```

对于手工放入共享目录、无法追溯来源的 skill：

```json
{
  "directory": "local-skill",
  "source": {
    "type": "local"
  }
}
```

扫描 installed skills 时，后端将目录扫描结果与本地 provenance 状态合并，从而保证：

1. 重启后仍能按仓库分组
2. 本地未知来源 skill 自动归到 `Local / Unknown Source`
3. 目录名冲突可以被稳定识别与阻断

---

## 六、前端重构方案

### 6.1 页面模型调整

当前页面有两个平台 tab：Claude / Codex。

新的界面直接参考 VS Code 扩展面板的信息架构，不再保留 Claude/Codex 作为主切换维度。

目标形态：

1. 左侧为导航与 skill 列表
2. 右侧为当前选中 skill 的详情面板
3. 页面整体交互、信息密度和浏览方式尽量贴近 VS Code 扩展界面

界面风格目标：

1. 用户进入页面后先浏览列表，再查看详情
2. 详情区域始终服务于当前选中项，而不是通过弹窗承载主要内容
3. Installed 与 marketplace-like 浏览体验统一

### 6.2 VS Code 风格界面结构

Skills 页面采用左右分栏布局：

1. **左侧边栏**
   - 搜索框
   - 分组导航
   - skills 列表
2. **右侧详情区**
   - skill 标题
   - 作者/来源仓库
   - 操作按钮
   - 详情说明
   - 状态信息
   - 原始内容或扩展信息

左侧边栏导航参考 VS Code 扩展侧栏，包含：

1. `Local - Installed`
2. `Recommended`

当前阶段：

1. `Local - Installed` 完整实现
2. `Recommended` 保留入口和空状态，占位展示，不加载实际推荐数据

`Recommended` 在视觉上保留为正式导航项，避免未来再次改动信息架构。

### 6.3 界面信息架构

Skills 页面重构为三大区域：

1. **左侧边栏**
   - 搜索
   - 导航分组
   - skill 列表
2. **右侧详情区**
   - 当前 skill 详情
   - 操作区
   - README / SKILL 内容
3. **顶部状态条或详情区状态模块**
   - `~/.agent/skills` 状态
   - Claude/Codex 链接状态
   - 迁移/备份状态

页面不再让用户先选平台，再看内容。页面直接回答三个问题：

1. 当前统一目录是否健康
2. Claude/Codex 是否正确接入
3. skills 当前有哪些、来自哪里、能做什么

### 6.4 左侧边栏设计

左侧边栏顶部包含：

1. 搜索框
2. 导航分组标题
3. 分组计数

导航结构：

1. `Local - Installed`
2. `Recommended`

`Local - Installed` 下展示已安装 skill 列表。

`Recommended` 当前只展示空状态，占位文案为：

1. 推荐技能即将支持

左侧列表项参考 VS Code 扩展列表：

1. skill 名称
2. 简短描述
3. 来源仓库或本地标记
4. 启用状态徽标
5. 冲突或异常状态徽标

列表交互：

1. 单击切换当前选中 skill
2. 当前选中项高亮
3. 搜索仅过滤左侧列表，不改变右侧详情布局
4. 支持按仓库分组折叠/展开

### 6.5 右侧详情区设计

右侧详情区完全参考 VS Code 扩展详情页结构，固定展示当前选中 skill 的完整信息。

详情区从上到下包含：

1. **标题区**
   - skill 名称
   - 目录名
   - 来源仓库
   - 本地来源标记
2. **主操作区**
   - 启用 / 禁用
   - 安装 / 卸载
   - 打开目录
   - 打开仓库
3. **状态区**
   - 是否启用
   - 是否存在目录名冲突
   - Claude 链接状态
   - Codex 链接状态
4. **说明区**
   - 描述
   - README 或 `SKILL.md` 摘要
5. **详细内容区**
   - 原始 `SKILL.md`
   - 备份/迁移相关信息

右侧详情区不依赖弹窗承载核心信息，弹窗只用于辅助诊断。

### 6.6 顶部状态条与状态卡

页面顶部或右侧详情区上方新增状态条/状态卡区域：

1. **User Skills 目录卡**
   - 路径：`~/.agent/skills`
   - 目录是否存在
   - skills 数量
2. **Claude 链接卡**
   - 路径：`~/.claude/skills`
   - 状态：已链接 / 缺失 / 冲突 / 修复失败
   - 当前目标路径
3. **Codex 链接卡**
   - 路径：`~/.codex/skills`
   - 状态：已链接 / 缺失 / 冲突 / 修复失败
   - 当前目标路径

状态区主操作：

1. `打开 ~/.agent/skills`
2. `修复链接`
3. `刷新`
4. `查看备份`

状态颜色约定：

1. 绿色：已链接
2. 黄色：需迁移或需确认
3. 红色：冲突或修复失败
4. 灰色：未检测

### 6.7 迁移与备份区

页面中部新增迁移与备份摘要区，用于承接这次目录模型重构带来的用户感知变化。

该区域显示：

1. 是否发现原始 `~/.claude/skills`
2. 是否发现原始 `~/.codex/skills`
3. 最近一次备份时间
4. 最近一次备份目录
5. 最近一次迁移结果
6. 当前是否存在目录名冲突

该区域提供两个入口：

1. `查看备份记录`
2. `查看迁移冲突`

如果存在冲突，则页面在该区域显示显式告警，而不是只在日志或 toast 中提示。

### 6.8 左侧列表的数据组织

左侧列表不再沿用当前三块结构：

1. Project Skills
2. User Skills
3. Available Skills

列表视图改为：

1. `Local - Installed`
2. `Recommended`

`Local - Installed` 内部按仓库分组：

- `mattpocock/skills`
- `anthropics/skills`
- `Local / Unknown Source`

每个仓库分组展示：

1. 仓库名称
2. skill 数量
3. 展开/收起状态
4. 仓库跳转入口

左侧列表项与分组容器统一使用 VS Code 扩展列表风格。

### 6.9 Installed Skills 详情设计

Installed skill 在右侧详情区展示：

1. 名称
2. 目录名
3. 来源仓库
4. 当前启用状态
5. 是否为本地来源
6. 最近来源记录状态

卡片操作：

1. 启用 / 禁用
2. 查看内容
3. 卸载
4. 打开目录
5. 打开来源仓库

如果 skill 属于 `Local / Unknown Source`，详情区弱化来源信息，强调该 skill 为手工放入目录。

### 6.10 Recommended / 可安装详情设计

`Recommended` 导航后续承载推荐与可安装 skills。

当前阶段右侧详情区为占位结构，保留以下信息形态：

1. 名称
2. 目录名
3. 来源仓库
4. 描述
5. 是否与当前已安装目录名冲突

卡片操作：

1. 安装
2. 查看仓库
3. 查看 README

若目标目录名已被占用：

1. 安装按钮禁用
2. 卡片显示“目录名冲突”
3. 冲突信息指向已安装 skill

### 6.11 安装交互调整

当前安装弹窗让用户选择：

- user
- project

该弹窗直接取消。

安装流程变为：

1. 用户点击安装
2. 直接安装到 `~/.agent/skills`

如果安装前发现链接状态异常：

1. 阻断安装
2. 打开“修复链接”流程

### 6.12 目录打开行为

当前页面支持打开：

- 用户级目录
- 项目级目录

新模型下应改为：

1. 打开 user 级 skills 目录：`~/.agent/skills`
2. 打开平台入口目录：
   - `~/.claude/skills`
   - `~/.codex/skills`

目录打开行为：

- 默认“打开 `~/.agent/skills`”
- 在状态区提供“查看 Claude 链接”“查看 Codex 链接”

### 6.13 分组数据结构

前端不应继续使用“平铺数组 + 页面层 merge/filter”方式。

后端直接返回可分组结果，或者最少返回足够稳定的分组字段，由前端只做轻量 `groupBy`。

推荐的数据形态：

```ts
type SkillGroup = {
  group_key: string
  group_label: string
  skills: SkillSummary[]
}
```

页面渲染逻辑只负责：

1. 渲染 installed groups
2. 渲染 available groups

不要再在页面里拼两套来源、补 installed 标志、再手工去重。

### 6.14 异常与空状态设计

页面需要覆盖以下状态：

1. **未检测到任何 skill**
   - 显示空状态
   - 提供“从仓库安装第一个 skill”入口
2. **链接异常**
   - 显示状态告警
   - 提供“修复链接”按钮
3. **迁移冲突**
   - 显示冲突摘要
   - 提供“查看冲突详情”入口
4. **备份成功**
   - 显示最近备份结果
   - 提供“打开备份目录”入口
5. **备份失败**
   - 阻断迁移
   - 显示失败原因

### 6.15 迁移冲突与备份详情界面

新增两个辅助界面：

1. **备份记录弹窗**
   - 平台
   - 备份时间
   - 备份路径
   - 打开目录按钮
2. **迁移冲突弹窗**
   - 冲突 skill 目录名
   - 冲突来源
   - 冲突类型
   - 建议处理方式

这两个弹窗不承担编辑功能，只承担诊断和导航功能。

### 6.16 VS Code 扩展面板 1:1 视觉对齐细则

目标：Skills 页面与 VS Code「Extensions」面板**视觉与信息架构一摸一样**，只是把「扩展」替换为「skill」、把「发布者(publisher)」替换为「来源仓库(`owner/name`)」。逐元素映射如下。

**A. 左侧栏顶部（对应 VS Code 的 EXTENSIONS 标题区）**

| VS Code 元素 | Skills 页面对应 |
| --- | --- |
| `EXTENSIONS` 标题 | `SKILLS` 标题 |
| 右上角刷新图标 | 刷新（重新扫描 + 拉取 catalog） |
| 右上角 `...` 菜单 | 溢出菜单：管理仓库、打开 `~/.agent/skills`、修复链接、查看备份 |
| `Search Extensions in Marketplace` 搜索框 + 清除/筛选图标 | `Search skills` 搜索框 + 清除图标 + 筛选图标（按启用状态/来源仓库/冲突筛选） |

**B. 左侧分组（对应 LOCAL - INSTALLED / RECOMMENDED 折叠组）**

1. 顶层折叠组与 VS Code 完全一致：
   - `LOCAL - INSTALLED`，组标题右侧带**数量徽标**（如 VS Code 的 `137`）
   - `RECOMMENDED`，带数量徽标；当前阶段为占位空状态
2. `LOCAL - INSTALLED` 内部按来源仓库再分**可折叠子组**（这正是 VS Code 折叠组的同一交互范式，满足目标 #2 的「按仓库分组」）：
   - 子组标题 = `owner/name`，右侧数量徽标
   - `Local / Unknown Source` 作为兜底子组
3. VS Code 的 `SSH: NETCUP - INSTALLED` 这类远程组在本场景无对应，**不实现**。

**C. 左侧列表项（对应单个扩展行）**

逐元素对齐 VS Code 列表行：

1. 左侧图标（skill 无图标时用首字母/默认占位图标）
2. 名称（加粗，单行截断）
3. 名称行右侧的耗时徽标位（VS Code 显示 `207ms` 激活耗时）→ 复用为**启用状态徽标 / 冲突徽标**
4. 第二行：一行截断的描述
5. 第三行：来源 `owner/name`（对应 VS Code 的 publisher）
6. 行尾齿轮图标 → 管理菜单（启用/禁用、卸载、打开目录、打开仓库）
7. `RECOMMENDED` 项行尾带 `安装` 按钮（对应 VS Code 的 `Install`）；当前阶段占位不可点

交互：单击选中并在右侧展示详情；当前项高亮；搜索只过滤左侧列表；子组可折叠。

**D. 右侧详情区（对应扩展详情页）**

1. **头部区**（对应大图标 + 标题行）：
   - 大图标、skill 名称（大字号）、来源 `owner/name`（对应 publisher）、目录名
   - 一行描述
2. **操作按钮行**（对应 `Disable ▾` / `Uninstall ▾` / `Auto Update ☑` / 齿轮）：
   - `禁用 ▾` / `启用`（已安装时）
   - `卸载 ▾`（已安装时）/ `安装`（可安装时）
   - `打开目录`、`打开仓库`
   - 若目录名冲突：`安装`/`启用` 禁用并提示冲突指向
3. **Tab 区**（对应 `DETAILS / FEATURES / CHANGELOG`）：
   - `说明`（README / `SKILL.md` body 渲染）— 对应 DETAILS
   - `SKILL.md`（原始 front matter + 内容，可查看/编辑）— 对应 FEATURES
   - `状态 & 迁移`（链接状态、备份/迁移记录）— 对应 CHANGELOG 位
4. **右侧信息侧栏**（对应 Installation / Marketplace / Categories / Resources）：
   - `安装信息`：目录名、来源 `owner/name`、分支、安装时间
   - `链接状态`：Claude 链接 / Codex 链接 健康度
   - `资源`：仓库链接、README、打开 `~/.agent/skills`

**E. 颜色与密度**：沿用 §6.6 状态色约定（绿/黄/红/灰），整体信息密度、留白、字号层级贴近 VS Code 暗色扩展面板。

> 说明：目标 #2「按仓库分组」与「VS Code 一摸一样」并不冲突——VS Code 本身就用带数量徽标的可折叠分组组织列表，本方案把仓库作为 `LOCAL - INSTALLED` 下的折叠子组即可同时满足两者。

---

## 七、接口调整建议

### 7.1 废弃的旧接口

- `ListSkillsForPlatform(platform)`
- `UninstallSkillEx(directory, platform, location)`
- `OpenSkillFolder(platform, location)`

这些接口都建立在“skills 按平台/位置分布”的旧模型之上。

### 7.2 新增的接口

接口清单（区分「新增」「破坏性变更」「已存在」，CR 修正）：

1. `ListInstalledSkills()` — **新增**，返回 `~/.agent/skills` 中的已安装 skills
2. `ListAvailableSkills()` — **新增**，返回远程仓库中可安装 skills
3. `ListGroupedSkills()` — **新增**，返回已按仓库分组的 installed/available 数据
4. `InstallSkill(directory, repoOwner, repoName, repoBranch)` — **破坏性签名变更**：当前签名为 `InstallSkill(req installRequest)`，需重写并去掉 `platform/location`
5. `UninstallSkill(directory)` — **已存在且签名相同**（`skillservice.go:449`），只需把内部实现从 `installDir` 改为 `~/.agent/skills`，无需新增
6. `EnsureSkillLinks()` — **新增**，修复 Claude/Codex 到 `~/.agent/skills` 的软链接（须幂等：状态已 linked 时跳过修复，避免 Windows junction 反复弹权限）
7. `GetSkillLinkStatus()` — **新增**，返回 `~/.agent/skills`、Claude 链接、Codex 链接当前状态
8. `OpenUserSkillsFolder()` — **新增**，打开 `~/.agent/skills`

`ToggleSkill` 当前签名为 `ToggleSkill(directory, platform, location, enabled)` — **破坏性变更**为 `ToggleSkill(directory, enabled)`，内部按 §5.5 写 sidecar 并投影回 `SKILL.md`。

> **落地提醒**：上述任何签名变更后，必须执行 `wails3 task common:generate:bindings` 重新生成 `frontend/bindings/`，并同步修改 `frontend/src/services/skill.ts`（当前直接 `Call.ByName(...)` 旧方法名）。废弃接口 `ListSkillsForPlatform`/`UninstallSkillEx`/`OpenSkillFolder` 删除后，前端对应封装与调用点一并清理。

接口主次关系：

1. Skills 页面主接口使用 `ListGroupedSkills()`
2. `ListInstalledSkills()` 与 `ListAvailableSkills()` 用于服务内部复用、调试接口或非分组场景
3. 前端页面不同时维护 grouped 和 flat 两套渲染逻辑

---

## 八、迁移策略

### 8.1 目标

让现有用户从：

- `~/.claude/skills`
- `~/.codex/skills`

迁移到：

- `~/.agent/skills`

同时尽量避免无提示覆盖。

### 8.2 迁移步骤

应用首次启动新版本时：

1. 检查 `~/.agent/skills` 是否存在
2. 检查 Claude/Codex 平台目录状态
3. 为需要迁移且为实体目录的原始平台目录创建备份
4. 生成迁移诊断结果

迁移状态可分为：

1. `clean`
   - 平台目录不存在或已是正确软链接
2. `migratable`
   - 平台目录中的 skill 可按目录粒度无冲突迁入 `~/.agent/skills`
3. `conflict`
   - 存在同名不同内容 skill、异常软链接或其他无法安全自动合并的情况

迁移执行策略：

1. `clean` 自动修复
2. `migratable` 自动迁移并修复
3. `conflict` 阻断自动迁移，在 UI 中提示用户处理

备份是迁移前置条件，不因 `migratable` 或 `conflict` 被跳过。

### 8.3 数据持久化迁移

当前 `skill.json` 中旧的 `skills` installed 记录价值较低，因为 installed 真相来自目录本身。

本次调整为：

1. 保留 repo 配置
2. 删除旧的按 directory 记录 installed 的状态模型
3. 新增：
   - enabled override
   - provenance 信息
   - 迁移状态记录
   - 备份记录

否则它会继续携带旧平台假设。

---

## 九、风险与边界

### 9.1 软链接跨平台差异

Windows、macOS、Linux 对软链接支持不同，尤其 Windows 可能涉及管理员权限或开发者模式。

因此实现时需要明确：

1. macOS/Linux 直接使用符号链接
2. Windows 使用 directory junction 作为平台入口映射方案
3. 如果 junction 创建失败：
   - 不自动回退到复制模式
   - 明确将状态标记为 `unsupported` 或 `repair_failed`
   - 在 UI 中提示用户手动修复或提升权限

平台入口映射范围：

1. macOS/Linux 使用 symlink
2. Windows 使用 junction
3. 不引入按平台复制 fallback

### 9.2 仓库分组对已安装 skills 的来源依赖

远程 catalog 中的 skill 有 repo 来源，但纯本地安装或手工拷贝的 skill 没有 repo 来源。

因此分组必须接受：

- `Local / Unknown Source`

不能假设所有 installed skill 都能映射回 GitHub 仓库。

> **provenance 来源约束（CR 补充）**：installed skill 的仓库归属**只能来自安装时持久化的 provenance**，不得用「目录名去 catalog 里反查」的模糊匹配——否则手工放入、恰好与某仓库重名的 skill 会被错误归到该仓库。无 provenance 即归 `Local / Unknown Source`。

### 9.3 Codex skills 支持（已验证）

验证结论（2026-06，OpenAI 官方文档 + 社区资料）：

1. **Codex CLI 原生支持 skills，用户级目录确为 `~/.codex/skills`**（官方 scope 优先级表：module `.codex/skills` > repo `.codex/skills` > user `~/.codex/skills` > system `/etc/codex/skills`）。双平台软链接前提成立。
2. **`SKILL.md` 是跨 agent 标准，`disable-model-invocation` 对 Codex 同样可移植、可被识别**。Claude 与 Codex 软链到同一份物理 `SKILL.md`，写入该字段可同时影响两端，符合「统一启用状态」目标。

需要在实现与文案上注意的两个语义差异：

1. **Codex 的「规范」禁用方式是 `~/.codex/config.toml` 的 `[[skills.config]] enabled=false`**（按 `SKILL.md` 路径定位）；`disable-model-invocation` 仅阻止**隐式/自动**调用，用户仍可用 `$name` 手动触发。即本方案中「禁用」语义 = 「模型不再自动调用」,并非「完全移除」。这与现有代码的语义一致（`enabled = !disable-model-invocation`），v1 保持不变。
2. **已知 bug**：`disable-model-invocation: true` 会把该 skill 从模型初始列表中完全隐藏，可能影响显式调用体验。v1 接受此现状,UI 文案用「禁用自动调用」而非「彻底禁用」以免误导；如需「彻底禁用 + 可恢复」,后续再评估写 `~/.codex/config.toml`。

结论：**Codex 软链接作为既定事实推进**；§5.5 的投影字段维持 `disable-model-invocation`,无需按平台分叉。

---

## 十、推荐实施顺序

### 阶段 1：目录模型与状态模型收敛

1. 引入 user 级 skills 目录 `~/.agent/skills`
2. 实现 Claude/Codex 软链接修复
3. 引入 enabled override 与 provenance 状态存储
4. 安装/卸载/扫描全部改为只认 `~/.agent/skills`
5. 去掉 skills 的 platform/location 安装语义

### 阶段 2：接口与页面模型收敛

1. 废弃 `ListSkillsForPlatform`
2. 新增统一 installed/available 接口
3. 移除前端 Claude/Codex skills tab
4. 将页面改为统一 skills 视图

### 阶段 3：仓库分组展示

1. 后端稳定提供 repo 分组字段
2. 前端 installed/available 按仓库分组渲染
3. 为未知来源 skill 提供 local 分组

### 阶段 4：迁移与状态提示

1. 增加链接状态检测
2. 增加迁移冲突提示
3. 增加“修复链接”入口

---

## 十一、最终产品行为

重构完成后的 skills 模型：

1. Skills 只有一份，位于 `~/.agent/skills`
2. Claude 和 Codex 只是共享使用这份 skills
3. Code-Switch 负责确保：
   - `~/.claude/skills`
   - `~/.codex/skills`
   都正确链接到 `~/.agent/skills`
4. Skills 页面展示的是：
   - 已安装 skills
   - 可安装 skills
   - 且都按来源 GitHub 仓库分组

该模型替代当前“按平台切 tab、按位置切 user/project、再拼远程列表”的复杂结构。

---

## 十二、已收敛结论

本方案当前已明确以下结论：

1. user 级 skills 目录固定为 `~/.agent/skills`
2. 不保留 project-level skills
3. 启用状态以 sidecar 为期望状态，但**必须投影回 `SKILL.md` 的 `disable-model-invocation`**，否则 CLI 不生效（CR 修正，见 §5.5）
4. 已安装 skill 必须持久化 provenance 信息；仓库归属只认 provenance，不做目录名反查
5. Windows 第一阶段使用 junction，不使用复制 fallback
6. Skills 页面移除 Claude/Codex tab，改成与 VS Code 扩展面板 1:1 对齐的统一视图（见 §6.16）
7. Codex 软链接以「Codex 确实读取 `~/.codex/skills`」为前置验证条件（见 §9.3）

在这些结论基础上，可以继续细化为正式实施计划。

---

## 十三、可执行实施清单

实施清单已拆为独立文档，便于作为可勾选的执行清单跟踪进度：

- 👉 [2026-06-16-skills-refactor-checklist.md](./2026-06-16-skills-refactor-checklist.md)

该清单按 6 个阶段（阶段 0 状态模型 → 阶段 5 首启迁移与收尾）串行拆分，每阶段含文件级步骤与验收标准，章节号回指本方案。
