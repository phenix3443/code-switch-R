package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillStoreMigratesLegacySkillsToProvenance(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	storePath := filepath.Join(home, skillStoreDir, skillStoreFile)
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		t.Fatalf("创建 store 目录失败: %v", err)
	}

	legacy := `{
  "skills": {
    "demo-skill": {
      "installed": true
    }
  },
  "repos": [
    {
      "owner": "example",
      "name": "skills",
      "branch": "main",
      "enabled": true
    }
  ]
}`
	if err := os.WriteFile(storePath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("写入旧 store 失败: %v", err)
	}

	ss := NewSkillService()
	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("加载 store 失败: %v", err)
	}

	if got := store.Provenance["demo-skill"]; got.Type != "local" {
		t.Fatalf("期望旧 skills 迁移为 local provenance，得到 %#v", got)
	}
	if _, ok := store.Skills["demo-skill"]; !ok {
		t.Fatalf("期望兼容保留旧 skills 映射")
	}
	if len(store.Repos) != 1 || store.Repos[0].Owner != "example" {
		t.Fatalf("期望保留原有 repos，得到 %#v", store.Repos)
	}
}

func TestSkillStoreSaveDoesNotPersistDeprecatedSkills(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	store := newDefaultSkillStore()
	store.Skills["legacy-only"] = skillState{Installed: true}
	store.EnabledOverrides["legacy-only"] = true
	store.Provenance["legacy-only"] = skillProvenance{
		Type:       "github",
		RepoOwner:  "owner",
		RepoName:   "repo",
		RepoBranch: "main",
	}
	store.Migrations = append(store.Migrations, migrationRecord{
		Platform:  skillPlatformClaude,
		Directory: "legacy-only",
		Status:    "migrated",
	})
	store.Backups = append(store.Backups, backupRecord{
		Platform: skillPlatformClaude,
		Path:     "/tmp/backup",
	})

	if err := ss.saveStoreLocked(store); err != nil {
		t.Fatalf("保存 store 失败: %v", err)
	}

	data, err := os.ReadFile(ss.storePath)
	if err != nil {
		t.Fatalf("读取已保存 store 失败: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "\"skills\": {") {
		t.Fatalf("不应再持久化废弃的 skills 字段: %s", content)
	}
	if !strings.Contains(content, `"enabled_overrides"`) {
		t.Fatalf("期望持久化 enabled_overrides: %s", content)
	}
	if !strings.Contains(content, `"provenance"`) {
		t.Fatalf("期望持久化 provenance: %s", content)
	}
	if !strings.Contains(content, `"migrations"`) {
		t.Fatalf("期望持久化 migrations: %s", content)
	}
	if !strings.Contains(content, `"backups"`) {
		t.Fatalf("期望持久化 backups: %s", content)
	}
}

func TestSkillPathHelpers(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	if got := getUserSkillsPath(); got != filepath.Join(home, ".agent", "skills") {
		t.Fatalf("getUserSkillsPath() = %q", got)
	}
	if got := getPlatformSkillsLinkPath(skillPlatformClaude); got != filepath.Join(home, ".claude", "skills") {
		t.Fatalf("getPlatformSkillsLinkPath(claude) = %q", got)
	}
	if got := getPlatformSkillsLinkPath(skillPlatformCodex); got != filepath.Join(home, ".codex", "skills") {
		t.Fatalf("getPlatformSkillsLinkPath(codex) = %q", got)
	}
	if got := getSkillBackupRoot(); got != filepath.Join(home, skillStoreDir, "backups", "skills") {
		t.Fatalf("getSkillBackupRoot() = %q", got)
	}
}

func TestInstallSkillUsesUnifiedUserSkillsDirectory(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	source := filepath.Join(home, "source-skill")
	createSkillFixture(t, source, "demo-skill", "Demo Skill", "demo desc", "")

	if err := ss.installFromPath("demo-skill", source); err != nil {
		t.Fatalf("installFromPath() 失败: %v", err)
	}

	assertSkillDirExists(t, filepath.Join(getUserSkillsPath(), "demo-skill"))

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	provenance := store.Provenance["demo-skill"]
	if provenance.Type != "local" {
		t.Fatalf("期望本地安装写入 local provenance，得到 %#v", provenance)
	}

	status := ss.GetSkillLinkStatus()
	assertLinkStatus(t, status.Claude, skillLinkStatusLinked, getUserSkillsPath())
	assertLinkStatus(t, status.Codex, skillLinkStatusLinked, getUserSkillsPath())
}

func TestInstallSkillBlocksConflictingDirectoryOwnership(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	source := filepath.Join(home, "source-skill")
	createSkillFixture(t, source, "shared-skill", "Shared Skill", "demo desc", "")

	store := newDefaultSkillStore()
	store.Provenance["shared-skill"] = skillProvenance{
		Type:       "github",
		RepoOwner:  "other",
		RepoName:   "repo",
		RepoBranch: "main",
	}
	if err := ss.saveStoreLocked(store); err != nil {
		t.Fatalf("预写 store 失败: %v", err)
	}

	err := ss.installFromPath("shared-skill", source)
	if err == nil {
		t.Fatalf("期望目录来源冲突时安装失败")
	}
	if !strings.Contains(err.Error(), "目录名已被其他来源占用") {
		t.Fatalf("期望返回来源冲突错误，得到: %v", err)
	}
}

func TestUninstallSkillRemovesProvenanceAndOverride(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	skillDir := filepath.Join(getUserSkillsPath(), "demo-skill")
	createSkillFixture(t, skillDir, "demo-skill", "Demo Skill", "demo desc", "")

	store := newDefaultSkillStore()
	store.Provenance["demo-skill"] = skillProvenance{
		Type:       "github",
		RepoOwner:  "owner",
		RepoName:   "repo",
		RepoBranch: "main",
	}
	store.EnabledOverrides["demo-skill"] = false
	if err := ss.saveStoreLocked(store); err != nil {
		t.Fatalf("预写 store 失败: %v", err)
	}

	if err := ss.UninstallSkill("demo-skill"); err != nil {
		t.Fatalf("UninstallSkill() 失败: %v", err)
	}

	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("期望 skill 目录被删除，得到 err=%v", err)
	}

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	if _, ok := store.Provenance["demo-skill"]; ok {
		t.Fatalf("期望卸载后清理 provenance")
	}
	if _, ok := store.EnabledOverrides["demo-skill"]; ok {
		t.Fatalf("期望卸载后清理 enabled_overrides")
	}
}

func TestToggleSkillPersistsOverrideAndRewritesSkillMD(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	skillDir := filepath.Join(getUserSkillsPath(), "demo-skill")
	createSkillFixture(t, skillDir, "demo-skill", "Demo Skill", "demo desc", "disable-model-invocation: false\n")

	if err := ss.ToggleSkill("demo-skill", false); err != nil {
		t.Fatalf("ToggleSkill() 失败: %v", err)
	}

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	override, ok := store.EnabledOverrides["demo-skill"]
	if !ok || override {
		t.Fatalf("期望持久化 enabled override=false，得到 %#v", store.EnabledOverrides)
	}

	content, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("读取 SKILL.md 失败: %v", err)
	}
	if !strings.Contains(string(content), "disable-model-invocation: true") {
		t.Fatalf("期望 SKILL.md 被改写为禁用状态，得到:\n%s", string(content))
	}
}

func TestGetAndSaveSkillContentUseUnifiedUserSkillsDirectory(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	skillDir := filepath.Join(getUserSkillsPath(), "demo-skill")
	createSkillFixture(t, skillDir, "demo-skill", "Demo Skill", "demo desc", "")

	content, err := ss.GetSkillContent("demo-skill")
	if err != nil {
		t.Fatalf("GetSkillContent() 失败: %v", err)
	}
	if !strings.Contains(content, "# Demo Skill") {
		t.Fatalf("期望读取统一目录中的 SKILL.md，得到:\n%s", content)
	}

	updated := "---\nname: Demo Skill\ndescription: changed\n---\n\n# Demo Skill\n"
	if err := ss.SaveSkillContent("demo-skill", updated); err != nil {
		t.Fatalf("SaveSkillContent() 失败: %v", err)
	}

	saved, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("读取更新后 SKILL.md 失败: %v", err)
	}
	if string(saved) != updated {
		t.Fatalf("期望 SKILL.md 写回统一目录，得到:\n%s", string(saved))
	}
}

func TestListInstalledSkillsUsesProvenanceGroups(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	createSkillFixture(t, filepath.Join(getUserSkillsPath(), "repo-skill"), "repo-skill", "Repo Skill", "repo desc", "")
	createSkillFixture(t, filepath.Join(getUserSkillsPath(), "local-skill"), "local-skill", "Local Skill", "local desc", "")

	store := newDefaultSkillStore()
	store.Provenance["repo-skill"] = skillProvenance{
		Type:       "github",
		RepoOwner:  "owner",
		RepoName:   "repo",
		RepoBranch: "main",
	}
	if err := ss.saveStoreLocked(store); err != nil {
		t.Fatalf("预写 store 失败: %v", err)
	}

	skills, err := ss.ListInstalledSkills()
	if err != nil {
		t.Fatalf("ListInstalledSkills() 失败: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("期望返回 2 个 installed skills，得到 %d", len(skills))
	}

	byDir := make(map[string]Skill)
	for _, skill := range skills {
		byDir[skill.Directory] = skill
	}

	repoSkill := byDir["repo-skill"]
	if repoSkill.SourceGroupKey != "owner/repo" || repoSkill.SourceGroupLabel != "owner/repo" {
		t.Fatalf("期望 repo skill 按 owner/repo 分组，得到 %#v", repoSkill)
	}
	if repoSkill.RepoOwner != "owner" || repoSkill.RepoName != "repo" || repoSkill.RepoBranch != "main" {
		t.Fatalf("期望 repo provenance 被稳定填充，得到 %#v", repoSkill)
	}

	localSkill := byDir["local-skill"]
	if localSkill.SourceGroupKey != "local" || localSkill.SourceGroupLabel != "Local / Unknown Source" {
		t.Fatalf("期望未知来源归入 local 分组，得到 %#v", localSkill)
	}
}

func TestListGroupedSkillsSplitsInstalledAndAvailableGroups(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	createSkillFixture(t, filepath.Join(getUserSkillsPath(), "installed-skill"), "installed-skill", "Installed Skill", "installed desc", "")

	store := newDefaultSkillStore()
	store.Provenance["installed-skill"] = skillProvenance{
		Type:       "github",
		RepoOwner:  "owner",
		RepoName:   "repo",
		RepoBranch: "main",
	}
	if err := ss.saveStoreLocked(store); err != nil {
		t.Fatalf("预写 store 失败: %v", err)
	}

	repoRoot := filepath.Join(home, "remote-snapshot")
	createSkillFixture(t, filepath.Join(repoRoot, "available-skill"), "available-skill", "Available Skill", "available desc", "")
	ss.repoSnapshotter = func(repo skillRepoConfig) (string, string, func(), error) {
		return repoRoot, "main", func() {}, nil
	}

	grouped, err := ss.ListGroupedSkills()
	if err != nil {
		t.Fatalf("ListGroupedSkills() 失败: %v", err)
	}

	if len(grouped.Installed) != 1 {
		t.Fatalf("期望 1 个 installed group，得到 %#v", grouped.Installed)
	}
	if grouped.Installed[0].GroupKey != "owner/repo" || len(grouped.Installed[0].Skills) != 1 {
		t.Fatalf("installed groups 不符合预期，得到 %#v", grouped.Installed)
	}
	if len(grouped.Available) != 1 {
		t.Fatalf("期望 1 个 available group，得到 %#v", grouped.Available)
	}
	if grouped.Available[0].GroupKey == "" || len(grouped.Available[0].Skills) != 1 {
		t.Fatalf("available groups 不符合预期，得到 %#v", grouped.Available)
	}
}

func createSkillFixture(t *testing.T, skillDir, directory, name, description, extraFrontMatter string) {
	t.Helper()

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("创建 skill fixture 目录失败: %v", err)
	}

	frontMatter := "name: " + name + "\n" + "description: " + description + "\n"
	if extraFrontMatter != "" {
		frontMatter += extraFrontMatter
	}
	content := "---\n" + frontMatter + "---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("写入 skill fixture 失败: %v", err)
	}
}

func assertSkillDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("期望 skill 目录存在 %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("期望 %s 是目录", path)
	}
}

func setTestHomeDir(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}
