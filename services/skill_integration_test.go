package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeswitch/services/testhelpers"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestSkillModuleIntegrationFlow(t *testing.T) {
	home := testhelpers.SetIsolatedHome(t)

	platformSkillDir := filepath.Join(home, ".claude", "skills", "legacy-skill")
	testhelpers.MustWriteFile(t, filepath.Join(platformSkillDir, "SKILL.md"), `---
name: Legacy Skill
description: migrated from claude
disable-model-invocation: false
---

# Legacy Skill
`)

	ss := NewSkillService()

	repoRoot := filepath.Join(home, "repo-snapshot")
	copyTestSkillFixture(t, filepath.Join("testdata", "skills", "demo-skill"), filepath.Join(repoRoot, "demo-skill"))
	ss.repoSnapshotter = func(repo skillRepoConfig) (string, string, func(), error) {
		return repoRoot, repo.Branch, func() {}, nil
	}

	repos, err := ss.AddRepo(skillRepoConfig{
		Owner:   "example",
		Name:    "skills",
		Branch:  "main",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("AddRepo() 失败: %v", err)
	}
	if len(repos) == 0 {
		t.Fatalf("期望 repo 已加入")
	}

	if err := ss.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup() 失败: %v", err)
	}

	status := ss.GetSkillLinkStatus()
	assertLinkStatus(t, status.Claude, skillLinkStatusLinked, getPlatformSkillsLinkPath(skillPlatformClaude))
	assertLinkStatus(t, status.Codex, skillLinkStatusLinked, getPlatformSkillsLinkPath(skillPlatformCodex))

	legacyPath := filepath.Join(getUserSkillsPath(), "legacy-skill", "SKILL.md")
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("期望 legacy skill 已迁移: %v", err)
	}

	groupedBefore, err := ss.ListGroupedSkills()
	if err != nil {
		t.Fatalf("ListGroupedSkills() 失败: %v", err)
	}
	if countGroupedSkills(groupedBefore.Installed) != 1 {
		t.Fatalf("期望初始 installed skills 为 1，得到 %d", countGroupedSkills(groupedBefore.Installed))
	}
	if countGroupedSkills(groupedBefore.Available) == 0 {
		t.Fatalf("期望存在可安装 skills")
	}

	if err := ss.InstallSkill("demo-skill", "example", "skills", "main", nil); err != nil {
		t.Fatalf("InstallSkill() 失败: %v", err)
	}

	groupedAfterInstall, err := ss.ListGroupedSkills()
	if err != nil {
		t.Fatalf("ListGroupedSkills() 失败: %v", err)
	}
	if countGroupedSkills(groupedAfterInstall.Installed) != 2 {
		t.Fatalf("期望安装后 installed skills 为 2，得到 %d", countGroupedSkills(groupedAfterInstall.Installed))
	}

	content, err := ss.GetSkillContent("demo-skill")
	if err != nil {
		t.Fatalf("GetSkillContent() 失败: %v", err)
	}
	if !strings.Contains(content, "Demo Skill") {
		t.Fatalf("期望读取到 demo skill 内容，得到: %s", content)
	}

	updatedContent := strings.Replace(content, "integration test demo skill", "updated by integration test", 1)
	if err := ss.SaveSkillContent("demo-skill", updatedContent); err != nil {
		t.Fatalf("SaveSkillContent() 失败: %v", err)
	}

	contentAfterSave, err := ss.GetSkillContent("demo-skill")
	if err != nil {
		t.Fatalf("GetSkillContent() 失败: %v", err)
	}
	if !strings.Contains(contentAfterSave, "updated by integration test") {
		t.Fatalf("期望内容已更新，得到: %s", contentAfterSave)
	}

	if err := ss.ToggleSkill("demo-skill", false); err != nil {
		t.Fatalf("ToggleSkill(false) 失败: %v", err)
	}

	contentAfterDisable, err := ss.GetSkillContent("demo-skill")
	if err != nil {
		t.Fatalf("GetSkillContent() 失败: %v", err)
	}
	if !strings.Contains(contentAfterDisable, "disable-model-invocation: true") {
		t.Fatalf("期望 skill 已禁用，得到: %s", contentAfterDisable)
	}

	if err := ss.UninstallSkill("demo-skill", nil); err != nil {
		t.Fatalf("UninstallSkill() 失败: %v", err)
	}

	groupedAfterUninstall, err := ss.ListGroupedSkills()
	if err != nil {
		t.Fatalf("ListGroupedSkills() 失败: %v", err)
	}
	if countGroupedSkills(groupedAfterUninstall.Installed) != 1 {
		t.Fatalf("期望卸载后 installed skills 为 1，得到 %d", countGroupedSkills(groupedAfterUninstall.Installed))
	}

	diagnostics, err := ss.GetSkillDiagnostics()
	if err != nil {
		t.Fatalf("GetSkillDiagnostics() 失败: %v", err)
	}
	if len(diagnostics.Backups) == 0 {
		t.Fatalf("期望存在迁移备份记录")
	}
	if len(diagnostics.Migrations) == 0 {
		t.Fatalf("期望存在迁移记录")
	}
}

func copyTestSkillFixture(t *testing.T, sourceDir, targetDir string) {
	t.Helper()

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("读取 fixture 目录失败 %s: %v", sourceDir, err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("创建 target 目录失败 %s: %v", targetDir, err)
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(sourceDir, entry.Name())
		targetPath := filepath.Join(targetDir, entry.Name())
		if entry.IsDir() {
			copyTestSkillFixture(t, sourcePath, targetPath)
			continue
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("读取 fixture 文件失败 %s: %v", sourcePath, err)
		}
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			t.Fatalf("写入 fixture 文件失败 %s: %v", targetPath, err)
		}
	}
}

func countGroupedSkills(groups []SkillGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Skills)
	}
	return total
}
