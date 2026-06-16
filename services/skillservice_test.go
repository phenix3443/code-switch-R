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

func setTestHomeDir(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}
