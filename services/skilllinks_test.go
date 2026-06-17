package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestSkillLinksCleanState(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()

	if err := ss.EnsureSkillLinks(); err != nil {
		t.Fatalf("EnsureSkillLinks() 返回错误: %v", err)
	}

	status := ss.GetSkillLinkStatus()
	assertLinkStatus(t, status.Claude, skillLinkStatusLinked, getUserSkillsPath())
	assertLinkStatus(t, status.Codex, skillLinkStatusLinked, getUserSkillsPath())

	if !status.UserSkillsExists {
		t.Fatalf("期望 user skills 目录存在")
	}
	if status.UserSkillCount != 0 {
		t.Fatalf("期望初始 skill 数量为 0，得到 %d", status.UserSkillCount)
	}
}

func TestSkillLinksMigratePlatformDirectories(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	createTestSkill(t, getPlatformSkillsLinkPath(skillPlatformClaude), "alpha", "Alpha", "claude alpha")
	createTestSkill(t, getPlatformSkillsLinkPath(skillPlatformCodex), "beta", "Beta", "codex beta")

	ss := NewSkillService()
	if err := ss.EnsureSkillLinks(); err != nil {
		t.Fatalf("EnsureSkillLinks() 返回错误: %v", err)
	}

	assertSkillExists(t, filepath.Join(getUserSkillsPath(), "alpha"))
	assertSkillExists(t, filepath.Join(getUserSkillsPath(), "beta"))

	status := ss.GetSkillLinkStatus()
	assertLinkStatus(t, status.Claude, skillLinkStatusLinked, getUserSkillsPath())
	assertLinkStatus(t, status.Codex, skillLinkStatusLinked, getUserSkillsPath())

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	if len(store.Backups) < 2 {
		t.Fatalf("期望记录两个平台的备份，得到 %#v", store.Backups)
	}
	if len(store.Migrations) < 2 {
		t.Fatalf("期望记录迁移动作，得到 %#v", store.Migrations)
	}
}

func TestSkillLinksConflictOnDifferentContent(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	createTestSkill(t, getUserSkillsPath(), "shared", "Shared", "user version")
	createTestSkill(t, getPlatformSkillsLinkPath(skillPlatformClaude), "shared", "Shared", "claude version")

	ss := NewSkillService()
	if err := ss.EnsureSkillLinks(); err == nil {
		t.Fatalf("期望冲突时返回错误")
	}

	status := ss.GetSkillLinkStatus()
	if status.Claude.Status != skillLinkStatusConflict {
		t.Fatalf("期望 Claude 状态为 conflict，得到 %#v", status.Claude)
	}
	if _, err := os.Lstat(getPlatformSkillsLinkPath(skillPlatformClaude)); err != nil {
		t.Fatalf("期望冲突时保留原始目录: %v", err)
	}

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	if !hasMigrationStatus(store.Migrations, skillPlatformClaude, "shared", skillMigrationStatusConflict) {
		t.Fatalf("期望记录冲突迁移状态，得到 %#v", store.Migrations)
	}
}

func TestSkillLinksConflictOnWrongSymlinkTarget(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	otherTarget := filepath.Join(home, "other-skills")
	if err := os.MkdirAll(otherTarget, 0o755); err != nil {
		t.Fatalf("创建 other target 失败: %v", err)
	}
	linkPath := getPlatformSkillsLinkPath(skillPlatformClaude)
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("创建链接父目录失败: %v", err)
	}
	if err := os.Symlink(otherTarget, linkPath); err != nil {
		t.Fatalf("创建错误软链接失败: %v", err)
	}

	ss := NewSkillService()
	if err := ss.EnsureSkillLinks(); err == nil {
		t.Fatalf("期望错误软链接返回错误")
	}

	status := ss.GetSkillLinkStatus()
	if status.Claude.Status != skillLinkStatusConflict {
		t.Fatalf("期望 Claude 状态为 conflict，得到 %#v", status.Claude)
	}
	if filepath.Clean(status.Claude.Target) != filepath.Clean(otherTarget) {
		t.Fatalf("期望保留错误链接目标，得到 %q", status.Claude.Target)
	}
}

func TestSkillLinksDeduplicateAcrossPlatforms(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	createTestSkill(t, getPlatformSkillsLinkPath(skillPlatformClaude), "shared", "Shared", "same content")
	createTestSkill(t, getPlatformSkillsLinkPath(skillPlatformCodex), "shared", "Shared", "same content")

	ss := NewSkillService()
	if err := ss.EnsureSkillLinks(); err != nil {
		t.Fatalf("EnsureSkillLinks() 返回错误: %v", err)
	}

	assertSkillExists(t, filepath.Join(getUserSkillsPath(), "shared"))

	entries, err := os.ReadDir(getUserSkillsPath())
	if err != nil {
		t.Fatalf("读取 user skills 目录失败: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("期望 user skills 目录只有一个 shared，得到 %d 个条目", len(entries))
	}

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	if !hasMigrationStatus(store.Migrations, skillPlatformCodex, "shared", skillMigrationStatusDuplicate) &&
		!hasMigrationStatus(store.Migrations, skillPlatformClaude, "shared", skillMigrationStatusDuplicate) {
		t.Fatalf("期望记录重复去重状态，得到 %#v", store.Migrations)
	}
}

func TestSkillLinksAlreadyCorrectLink(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	userSkills := getUserSkillsPath()
	if err := os.MkdirAll(userSkills, 0o755); err != nil {
		t.Fatalf("创建 user skills 目录失败: %v", err)
	}
	for _, platform := range []string{skillPlatformClaude, skillPlatformCodex} {
		linkPath := getPlatformSkillsLinkPath(platform)
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			t.Fatalf("创建父目录失败: %v", err)
		}
		if err := os.Symlink(userSkills, linkPath); err != nil {
			t.Fatalf("创建正确软链接失败: %v", err)
		}
	}

	ss := NewSkillService()
	if err := ss.EnsureSkillLinks(); err != nil {
		t.Fatalf("EnsureSkillLinks() 返回错误: %v", err)
	}

	status := ss.GetSkillLinkStatus()
	assertLinkStatus(t, status.Claude, skillLinkStatusLinked, userSkills)
	assertLinkStatus(t, status.Codex, skillLinkStatusLinked, userSkills)
}

func TestEnsureSkillLinksReconcilesEnabledOverrides(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	createTestSkill(t, getUserSkillsPath(), "demo-skill", "Demo", "desc")

	ss := NewSkillService()
	store := newDefaultSkillStore()
	store.EnabledOverrides["demo-skill"] = false
	if err := ss.saveStoreLocked(store); err != nil {
		t.Fatalf("预写 store 失败: %v", err)
	}

	if err := ss.EnsureSkillLinks(); err != nil {
		t.Fatalf("EnsureSkillLinks() 返回错误: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(getUserSkillsPath(), "demo-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("读取 SKILL.md 失败: %v", err)
	}
	if !strings.Contains(string(content), "disable-model-invocation: true") {
		t.Fatalf("期望 reconcile enabled override 到 SKILL.md，得到:\n%s", string(content))
	}
}

func TestBackupPlatformDirCopiesOriginalContent(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	createTestSkill(t, getPlatformSkillsLinkPath(skillPlatformClaude), "alpha", "Alpha", "backup me")

	ss := NewSkillService()
	record, err := ss.backupPlatformDir(skillPlatformClaude)
	if err != nil {
		t.Fatalf("backupPlatformDir() 返回错误: %v", err)
	}

	assertSkillExists(t, filepath.Join(record.Path, "alpha"))

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	if len(store.Backups) != 1 {
		t.Fatalf("期望写入一个备份记录，得到 %#v", store.Backups)
	}
}

func TestSkillServiceServiceStartupEnsuresLinksOnStartup(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	ss := NewSkillService()
	if err := ss.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup() 返回错误: %v", err)
	}

	status := ss.GetSkillLinkStatus()
	assertLinkStatus(t, status.Claude, skillLinkStatusLinked, getUserSkillsPath())
	assertLinkStatus(t, status.Codex, skillLinkStatusLinked, getUserSkillsPath())
}

func TestSkillServiceServiceStartupDoesNotBlockOnMigrationConflict(t *testing.T) {
	home := t.TempDir()
	setTestHomeDir(t, home)

	createTestSkill(t, getUserSkillsPath(), "shared", "Shared", "user version")
	createTestSkill(t, getPlatformSkillsLinkPath(skillPlatformClaude), "shared", "Shared", "claude version")

	ss := NewSkillService()
	if err := ss.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("期望启动阶段冲突不阻断应用，得到错误: %v", err)
	}

	status := ss.GetSkillLinkStatus()
	if status.Claude.Status != skillLinkStatusConflict {
		t.Fatalf("期望冲突状态被保留给 UI，得到 %#v", status.Claude)
	}

	store, err := ss.loadStore()
	if err != nil {
		t.Fatalf("loadStore() 失败: %v", err)
	}
	if !hasMigrationStatus(store.Migrations, skillPlatformClaude, "shared", skillMigrationStatusConflict) {
		t.Fatalf("期望启动后仍记录 conflict migration，得到 %#v", store.Migrations)
	}
}

func createTestSkill(t *testing.T, root, directory, name, description string) {
	t.Helper()

	skillDir := filepath.Join(root, directory)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("创建测试 skill 目录失败: %v", err)
	}

	content := "---\n" +
		"name: " + name + "\n" +
		"description: " + description + "\n" +
		"---\n" +
		"\n" +
		"# " + name + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("写入测试 SKILL.md 失败: %v", err)
	}
}

func assertSkillExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("期望 skill 存在 %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("期望 %s 是目录", path)
	}
}

func assertLinkStatus(t *testing.T, got SkillLinkEntry, wantStatus, wantTarget string) {
	t.Helper()

	if got.Status != wantStatus {
		t.Fatalf("期望状态为 %s，得到 %#v", wantStatus, got)
	}
	if filepath.Clean(got.Target) != filepath.Clean(wantTarget) {
		t.Fatalf("期望目标为 %s，得到 %#v", wantTarget, got)
	}
}

func hasMigrationStatus(records []migrationRecord, platform, directory, status string) bool {
	for _, record := range records {
		if record.Platform == platform && record.Directory == directory && record.Status == status {
			return true
		}
	}
	return false
}
