# 模型白名单与映射功能测试指南

## 📦 测试文件清单

```
services/
├── providerservice_test.go    # 核心算法单元测试（~350行）
├── providerrelay_test.go      # 请求处理与端到端测试（~250行）
├── skill_integration_test.go  # skills 服务级集成测试
├── testhelpers/
│   └── env.go                 # HOME 隔离、测试文件辅助
└── testdata/
    └── skills/
        └── demo-skill/
            └── SKILL.md       # skills 集成测试样例
    └── example-claude-config.json  # 测试配置示例
```

## 🧪 运行测试

### 运行所有测试
```bash
cd G:\claude-lit\cc-r
go test ./services/... -v
```

### 运行特定测试文件
```bash
# 测试核心算法
go test ./services/providerservice_test.go ./services/providerservice.go -v

# 测试请求处理
go test ./services/providerrelay_test.go ./services/providerrelay.go -v
```

### 运行特定测试用例
```bash
# 测试通配符匹配
go test ./services/... -run TestMatchWildcard -v

# 测试模型支持检查
go test ./services/... -run TestProvider_IsModelSupported -v

# 测试端到端场景
go test ./services/... -run TestModelMappingEndToEnd -v

# 测试 skills 模块完整集成流
go test ./services/... -run TestSkillModuleIntegrationFlow -v
```

### 运行性能测试
```bash
go test ./services/... -bench=. -benchmem
```

## 📊 测试覆盖率

### 查看测试覆盖率
```bash
go test ./services/... -cover
```

### 生成详细覆盖率报告
```bash
go test ./services/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 🧩 Skills 统一目录重构测试

### 运行 skills 相关测试
```bash
# 运行所有 skills 测试
go test ./services/... -run 'TestSkill|TestSkillLinks' -v

# 仅验证软链接/迁移/备份逻辑
go test ./services/... -run TestSkillLinks -v

# 验证启动时自动执行 links 检查
go test ./services/... -run TestSkillServiceServiceStartup -v
```

### skills 覆盖重点

#### skillservice_test.go
- ✅ 旧 `skill.json` 状态迁移到 `Provenance`
- ✅ 统一目录 `~/.agents/skills` 的安装 / 卸载 / 内容读写
- ✅ 目录来源冲突阻断
- ✅ `ToggleSkill` 将 override 投影回 `SKILL.md`
- ✅ installed / available skills 的来源分组

#### skilllinks_test.go
- ✅ clean state 下自动创建 Claude/Codex 链接
- ✅ 从原始平台目录迁移到 `~/.agents/skills`
- ✅ 同名不同内容冲突记录
- ✅ 双平台重复内容去重
- ✅ 备份记录生成
- ✅ `EnsureSkillLinks()` reconcile `EnabledOverrides`
- ✅ `SkillService.ServiceStartup()` 启动时自动执行 links 检查，冲突不阻断启动

#### skill_integration_test.go
- ✅ 隔离 HOME 下跑完整 startup / repair / migrate / install / toggle / content / uninstall 流
- ✅ 验证 repo snapshot 安装链路与 diagnostics 输出
- ✅ 验证统一目录与 Claude/Codex link 状态

## 🧪 前端测试基建

前端新增 `Vitest` 基础配置，当前先覆盖 `frontend/src/services/skill.ts` 的调用层 smoke tests：

```bash
cd frontend
npm test
```

## 🖥️ 隔离测试实例

如果需要启动一个不污染本地 `~/.code-switch` 的测试版 app，可以直接使用仓库根目录下的命令：

```bash
make test-app
```

默认行为：

- 使用隔离 HOME：`./.tmp/test-home`
- 使用独立 relay 端口：`18110`
- 使用独立前端 dev 端口：`9255`
- 测试实例配置文件写入：`./.tmp/test-home/.code-switch/network.json`
- 默认复用当前机器的 `GOPATH`，避免把 Go module cache 写进测试 HOME 导致清理失败

可选环境变量：

```bash
CODE_SWITCH_TEST_HOME=/tmp/code-switch-test-home CODE_SWITCH_TEST_PORT=18120 WAILS_VITE_PORT=9265 make test-app
```

## 🎯 测试覆盖范围

### providerservice_test.go

#### 1. **通配符匹配测试** (`TestMatchWildcard`)
- ✅ 精确匹配
- ✅ 前缀通配符 (`claude-*`)
- ✅ 后缀通配符 (`*-4`)
- ✅ 中间通配符 (`claude-*-4`)
- ✅ 边界情况（空前缀、空后缀）

#### 2. **通配符映射应用测试** (`TestApplyWildcardMapping`)
- ✅ 前缀通配符映射 (`claude-*` → `anthropic/claude-*`)
- ✅ 中间通配符映射 (`claude-*-4` → `anthropic/claude-*-v4`)
- ✅ 无通配符场景
- ✅ 边界情况

#### 3. **模型支持检查测试** (`TestProvider_IsModelSupported`)
- ✅ 向后兼容（未配置白名单）
- ✅ 原生支持 - 精确匹配
- ✅ 原生支持 - 通配符匹配
- ✅ 映射支持 - 精确匹配
- ✅ 映射支持 - 通配符匹配
- ✅ 混合模式（原生 + 映射）

#### 4. **获取有效模型测试** (`TestProvider_GetEffectiveModel`)
- ✅ 无映射场景
- ✅ 精确映射
- ✅ 通配符映射
- ✅ 精确优先于通配符

#### 5. **配置验证测试** (`TestProvider_ValidateConfiguration`)
- ✅ 有效配置
- ✅ 无效映射（目标不在白名单）
- ✅ 警告：只配置映射未配置白名单
- ✅ 警告：自映射
- ✅ 通配符映射（跳过验证）

### providerrelay_test.go

#### 1. **请求体模型替换测试** (`TestReplaceModelInRequestBody`)
- ✅ 简单替换
- ✅ 复杂嵌套 JSON
- ✅ 特殊字符处理
- ✅ 错误场景（缺少 model 字段、无效 JSON）

#### 2. **端到端场景测试** (`TestModelMappingEndToEnd`)
- ✅ 完整降级流程模拟
- ✅ 通配符映射实际应用
- ✅ 多供应商场景
- ✅ 不支持的模型处理

#### 3. **配置验证集成测试** (`TestProviderConfigValidation`)
- ✅ 完美配置
- ✅ 错误配置
- ✅ 通配符配置

#### 4. **性能基准测试**
- ✅ `BenchmarkIsModelSupported` - 模型支持检查性能
- ✅ `BenchmarkGetEffectiveModel` - 模型映射性能
- ✅ `BenchmarkReplaceModelInRequestBody` - JSON 替换性能

## 🔍 测试场景详解

### 场景 1：基础精确匹配
```go
Provider {
    SupportedModels: {"claude-sonnet-4": true},
}
请求: claude-sonnet-4 → ✅ 支持
请求: gpt-4 → ❌ 不支持
```

### 场景 2：通配符白名单
```go
Provider {
    SupportedModels: {"claude-*": true},
}
请求: claude-sonnet-4 → ✅ 支持（通配符匹配）
请求: claude-opus-4 → ✅ 支持（通配符匹配）
请求: gpt-4 → ❌ 不支持
```

### 场景 3：精确映射
```go
Provider {
    SupportedModels: {"anthropic/claude-sonnet-4": true},
    ModelMapping: {"claude-sonnet-4": "anthropic/claude-sonnet-4"},
}
请求: claude-sonnet-4
  → IsModelSupported: ✅ true
  → GetEffectiveModel: "anthropic/claude-sonnet-4"
  → 请求体: {"model": "anthropic/claude-sonnet-4", ...}
```

### 场景 4：通配符映射
```go
Provider {
    SupportedModels: {"anthropic/claude-*": true},
    ModelMapping: {"claude-*": "anthropic/claude-*"},
}
请求: claude-sonnet-4
  → IsModelSupported: ✅ true（通配符匹配）
  → GetEffectiveModel: "anthropic/claude-sonnet-4"（通配符展开）
  → 请求体: {"model": "anthropic/claude-sonnet-4", ...}
```

### 场景 5：实际降级流程
```
1. 用户请求: {"model": "claude-sonnet-4", ...}
2. Provider A (Anthropic Official):
   - IsModelSupported("claude-sonnet-4") = true
   - GetEffectiveModel("claude-sonnet-4") = "claude-sonnet-4"
   - 转发请求 → 成功 ✅

3. 如果 Provider A 失败，降级到 Provider B (OpenRouter):
   - IsModelSupported("claude-sonnet-4") = true（映射支持）
   - GetEffectiveModel("claude-sonnet-4") = "anthropic/claude-sonnet-4"
   - ReplaceModelInRequestBody → {"model": "anthropic/claude-sonnet-4", ...}
   - 转发修改后的请求 → 成功 ✅
```

## ✅ 验证清单

运行测试后，验证以下输出：

```bash
$ go test ./services/... -v

=== RUN   TestMatchWildcard
=== RUN   TestMatchWildcard/精确匹配-成功
=== RUN   TestMatchWildcard/前缀通配符-成功
=== RUN   TestMatchWildcard/中间通配符-成功
...
--- PASS: TestMatchWildcard (0.00s)

=== RUN   TestApplyWildcardMapping
...
--- PASS: TestApplyWildcardMapping (0.00s)

=== RUN   TestProvider_IsModelSupported
...
--- PASS: TestProvider_IsModelSupported (0.00s)

=== RUN   TestProvider_GetEffectiveModel
...
--- PASS: TestProvider_GetEffectiveModel (0.00s)

=== RUN   TestProvider_ValidateConfiguration
...
--- PASS: TestProvider_ValidateConfiguration (0.00s)

=== RUN   TestReplaceModelInRequestBody
...
--- PASS: TestReplaceModelInRequestBody (0.00s)

=== RUN   TestModelMappingEndToEnd
...
--- PASS: TestModelMappingEndToEnd (0.00s)

=== RUN   TestProviderConfigValidation
...
--- PASS: TestProviderConfigValidation (0.00s)

PASS
ok      codeswitch/services     0.XXXs
```

## 🐛 常见问题

### 问题 1：`go: command not found`
**解决**：确保已安装 Go 1.24+ 并配置环境变量。

### 问题 2：依赖缺失
**解决**：运行 `go mod tidy` 安装依赖。

### 问题 3：测试超时
**解决**：增加超时时间 `go test ./services/... -timeout 30s`

## 📈 性能基准

预期性能指标（参考值）：

```
BenchmarkIsModelSupported-8            10000000    100 ns/op    0 B/op    0 allocs/op
BenchmarkGetEffectiveModel-8            5000000    200 ns/op   32 B/op    1 allocs/op
BenchmarkReplaceModelInRequestBody-8     500000   3000 ns/op  512 B/op    5 allocs/op
```

**性能特点**：
- ✅ 模型支持检查：O(1) 时间复杂度（map 查找）
- ✅ 模型映射：O(1) 精确匹配 + O(n) 通配符回退（n 通常很小）
- ✅ JSON 替换：O(k) k 为 JSON 深度（通常 <10）

## 🎓 下一步

测试通过后，建议：
1. 📝 同步 skills 统一目录相关文档
2. 🧪 在 Windows 环境补充 junction/权限链路验证
3. 🚀 在实际用户目录上验证首次迁移与备份流程
