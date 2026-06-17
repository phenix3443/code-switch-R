package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	skillLinkStatusLinked       = "linked"
	skillLinkStatusMissing      = "missing"
	skillLinkStatusConflict     = "conflict"
	skillLinkStatusRepairFailed = "repair_failed"
	skillLinkStatusUnsupported  = "unsupported"

	skillMigrationStatusMigrated  = "migrated"
	skillMigrationStatusDuplicate = "duplicate"
	skillMigrationStatusConflict  = "conflict"
)

type SkillLinkStatus struct {
	UserSkillsExists bool           `json:"user_skills_exists"`
	UserSkillCount   int            `json:"user_skill_count"`
	Claude           SkillLinkEntry `json:"claude"`
	Codex            SkillLinkEntry `json:"codex"`
}

type SkillLinkEntry struct {
	Platform        string `json:"platform"`
	Status          string `json:"status"`
	Target          string `json:"target,omitempty"`
	Error           string `json:"error,omitempty"`
	LinkedSkillCount int   `json:"linked_skill_count,omitempty"`
}

func (ss *SkillService) GetSkillLinkStatus() SkillLinkStatus {
	storeDir := getUserSkillsPath()
	status := SkillLinkStatus{
		UserSkillsExists: dirExists(storeDir),
		UserSkillCount:   countInstalledSkillDirs(storeDir),
		Claude:           ss.getPlatformLinkEntry(skillPlatformClaude, storeDir),
		Codex:            ss.getPlatformLinkEntry(skillPlatformCodex, storeDir),
	}
	return status
}

func (ss *SkillService) EnsureSkillLinks() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	store, err := ss.loadStoreLocked()
	if err != nil {
		return err
	}

	userSkillsPath := getUserSkillsPath()
	if err := os.MkdirAll(userSkillsPath, 0o755); err != nil {
		return fmt.Errorf("创建 user skills 目录失败: %w", err)
	}

	var errs []error
	for _, platform := range []string{skillPlatformClaude, skillPlatformCodex} {
		if err := ss.recoverPlatformDirIfNeeded(platform); err != nil {
			errs = append(errs, err)
		}
		if err := ss.ensurePlatformLink(platform, userSkillsPath, &store); err != nil {
			errs = append(errs, err)
		}
	}

	for directory, enabled := range store.EnabledOverrides {
		skillMDPath := filepath.Join(userSkillsPath, directory, "SKILL.md")
		if _, err := os.Stat(skillMDPath); err != nil {
			continue
		}
		if err := ss.applyEnabledOverride(skillMDPath, enabled); err != nil {
			errs = append(errs, fmt.Errorf("reconcile enabled override for %s failed: %w", directory, err))
		}
	}

	if err := ss.saveStoreLocked(store); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// isAgentDirManaged 判断 agent skills 目录是否为 app 托管：仅含 per-skill 符号链接时视为托管
func isAgentDirManaged(agentSkillsPath string) bool {
	entries, err := os.ReadDir(agentSkillsPath)
	if err != nil {
		return true // 不可读时不误报冲突
	}
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			continue
		}
		// 非链接的实体目录且含 SKILL.md → 尚未迁移的遗留内容
		if _, statErr := os.Stat(filepath.Join(agentSkillsPath, entry.Name(), "SKILL.md")); statErr == nil {
			return false
		}
	}
	return true
}

// ensureSkillAgentLink 确保 per-skill 符号链接存在：~/<agent>/skills/<dir> → ~/.agents/skills/<dir>
func ensureSkillAgentLink(platform, dir string) error {
	storeDir := getUserSkillsPath()
	target := filepath.Join(storeDir, dir)
	linkPath := agentLinkPath(platform, dir)
	if linkPath == "" {
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return err
	}

	info, err := os.Lstat(linkPath)
	if err == nil {
		if !isSymlink(info) {
			return fmt.Errorf("%s 已存在且不是符号链接", linkPath)
		}
		existing, resolveErr := resolveLinkTarget(linkPath)
		if resolveErr == nil && samePath(existing, target) {
			return nil // 已正确
		}
		if err := os.Remove(linkPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return createPlatformLink(target, linkPath)
}

// ensureSkillAgentLinks 为多个 agent 建 per-skill 链接
func ensureSkillAgentLinks(dir string, agents []string) error {
	var errs []error
	for _, agent := range agents {
		if err := ensureSkillAgentLink(agent, dir); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", agent, err))
		}
	}
	return errors.Join(errs...)
}

// removeSkillAgentLink 删除 per-skill 符号链接（~/<agent>/skills/<dir>），非链接时跳过
func removeSkillAgentLink(platform, dir string) error {
	linkPath := agentLinkPath(platform, dir)
	if linkPath == "" {
		return nil
	}
	info, err := os.Lstat(linkPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isSymlink(info) {
		return fmt.Errorf("%s 不是符号链接，跳过删除", linkPath)
	}
	return os.Remove(linkPath)
}

// agentInstallState 通过检查磁盘上的 per-skill 符号链接判断安装状态，不依赖 CLI
func agentInstallState(dir string) SkillAgents {
	storeDir := getUserSkillsPath()
	expected := filepath.Join(storeDir, dir)
	return SkillAgents{
		Claude: isAgentSkillLinked(skillPlatformClaude, dir, expected),
		Codex:  isAgentSkillLinked(skillPlatformCodex, dir, expected),
	}
}

func isAgentSkillLinked(platform, dir, expectedTarget string) bool {
	linkPath := agentLinkPath(platform, dir)
	if linkPath == "" {
		return false
	}
	info, err := os.Lstat(linkPath)
	if err != nil || !isSymlink(info) {
		return false
	}
	target, err := resolveLinkTarget(linkPath)
	if err != nil {
		return false
	}
	return samePath(target, expectedTarget)
}

// agentLinkPath 返回 per-skill 链接路径：~/<agent>/skills/<dir>
func agentLinkPath(platform, dir string) string {
	base := getPlatformSkillsLinkPath(platform)
	if base == "" {
		return ""
	}
	return filepath.Join(base, dir)
}

// countLinkedSkillDirs 统计 agent skills 目录中指向 store 的 per-skill 符号链接数量
func countLinkedSkillDirs(agentSkillsPath, storeDir string) int {
	entries, err := os.ReadDir(agentSkillsPath)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink == 0 {
			continue
		}
		linkPath := filepath.Join(agentSkillsPath, entry.Name())
		target, err := resolveLinkTarget(linkPath)
		if err != nil {
			continue
		}
		expected := filepath.Join(storeDir, entry.Name())
		if samePath(target, expected) {
			count++
		}
	}
	return count
}

// migrateLegacyWholeDirLink 将整目录符号链接（~/<agent>/skills → store）迁移为真实目录 + per-skill 链接
func migrateLegacyWholeDirLink(platform, storeDir string) error {
	linkPath := getPlatformSkillsLinkPath(platform)
	if linkPath == "" {
		return nil
	}
	info, err := os.Lstat(linkPath)
	if err != nil {
		return nil // 不存在或无法读取，由后续步骤处理
	}
	if !isSymlink(info) {
		return nil // 已是真实目录，无需迁移
	}
	target, err := resolveLinkTarget(linkPath)
	if err != nil {
		return fmt.Errorf("迁移 %s: 读取链接目标失败: %w", platform, err)
	}
	if !samePath(target, storeDir) {
		return nil // 指向其他目标，不处理（由 ensurePlatformLink 报告冲突）
	}

	// 整目录链接 → 删除 → 创建真实目录 → 建 per-skill 链接
	if err := os.Remove(linkPath); err != nil {
		return fmt.Errorf("迁移 %s: 删除整目录链接失败: %w", platform, err)
	}
	if err := os.MkdirAll(linkPath, 0o755); err != nil {
		_ = createPlatformLink(storeDir, linkPath) // 尝试回滚
		return fmt.Errorf("迁移 %s: 创建真实目录失败: %w", platform, err)
	}

	entries, err := os.ReadDir(storeDir)
	if err != nil {
		return nil // store 不可读，迁移完成但无链接可建
	}
	var errs []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := entry.Name()
		if _, statErr := os.Stat(filepath.Join(storeDir, dir, "SKILL.md")); statErr != nil {
			continue
		}
		if err := ensureSkillAgentLink(platform, dir); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", dir, err))
		}
	}
	return errors.Join(errs...)
}

func (ss *SkillService) backupPlatformDir(platform string) (backupRecord, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	store, err := ss.loadStoreLocked()
	if err != nil {
		return backupRecord{}, err
	}

	record, err := ss.backupPlatformDirLocked(platform, &store)
	if err != nil {
		return backupRecord{}, err
	}
	if err := ss.saveStoreLocked(store); err != nil {
		return backupRecord{}, err
	}

	return record, nil
}

func (ss *SkillService) computeSkillFingerprint(dir string) (string, error) {
	hasher := sha256.New()
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(files)
	for _, rel := range files {
		if _, err := hasher.Write([]byte(rel)); err != nil {
			return "", err
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return "", err
		}

		filePath := filepath.Join(dir, filepath.FromSlash(rel))
		f, err := os.Open(filePath)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(hasher, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
		if _, err := hasher.Write([]byte{0}); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (ss *SkillService) getPlatformLinkEntry(platform string, storeDir string) SkillLinkEntry {
	entry := SkillLinkEntry{
		Platform: platform,
		Status:   skillLinkStatusMissing,
	}

	linkPath := getPlatformSkillsLinkPath(platform)
	if linkPath == "" {
		entry.Status = skillLinkStatusUnsupported
		entry.Error = "unknown platform"
		return entry
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return entry
		}
		entry.Status = skillLinkStatusRepairFailed
		entry.Error = err.Error()
		return entry
	}

	if isSymlink(info) {
		// 旧整目录链接或指向其他目标 → 需要迁移/修复
		target, err := resolveLinkTarget(linkPath)
		if err != nil {
			entry.Status = skillLinkStatusRepairFailed
			entry.Error = err.Error()
			return entry
		}
		entry.Target = target
		entry.Status = skillLinkStatusConflict
		return entry
	}

	if !info.IsDir() {
		entry.Status = skillLinkStatusConflict
		entry.Error = "platform skills path is not a directory"
		return entry
	}

	// 真实目录：若含有非链接的 skill 实体目录，说明迁移未完成（conflict）
	if !isAgentDirManaged(linkPath) {
		entry.Status = skillLinkStatusConflict
		entry.Target = linkPath
		return entry
	}

	// 健康：真实托管目录，仅含 per-skill 符号链接
	entry.Status = skillLinkStatusLinked
	entry.Target = linkPath
	entry.LinkedSkillCount = countLinkedSkillDirs(linkPath, storeDir)
	return entry
}

func (ss *SkillService) ensurePlatformLink(platform, userSkillsPath string, store *skillStore) error {
	linkPath := getPlatformSkillsLinkPath(platform)
	if linkPath == "" {
		return ss.appendPlatformError(store, platform, skillLinkStatusUnsupported, "unknown platform")
	}
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, err.Error())
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 不存在：创建真实目录
			if err := os.MkdirAll(linkPath, 0o755); err != nil {
				return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, err.Error())
			}
			return nil
		}
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, err.Error())
	}

	if isSymlink(info) {
		target, err := resolveLinkTarget(linkPath)
		if err != nil {
			return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, err.Error())
		}
		if samePath(target, userSkillsPath) {
			// 旧整目录链接 → 迁移为真实目录 + per-skill 链接
			if err := migrateLegacyWholeDirLink(platform, userSkillsPath); err != nil {
				return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, err.Error())
			}
			return nil
		}
		return ss.appendPlatformError(store, platform, skillLinkStatusConflict, fmt.Sprintf("existing link points to %s", target))
	}

	if !info.IsDir() {
		return ss.appendPlatformError(store, platform, skillLinkStatusConflict, "platform skills path is not a directory")
	}

	// 真实目录且已托管（仅含 per-skill 符号链接）：无需迁移
	if isAgentDirManaged(linkPath) {
		return nil
	}

	// 真实目录含遗留 skill 文件：备份 + 合并到 store + 重命名 + 创建新真实目录 + per-skill 链接
	record, err := ss.backupPlatformDirLocked(platform, store)
	if err != nil {
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, err.Error())
	}
	_ = record

	if err := ss.mergePlatformSkills(platform, linkPath, userSkillsPath, store); err != nil {
		return err
	}

	tempBakPath := fmt.Sprintf("%s.bak-%d", linkPath, time.Now().UnixNano())
	if err := os.Rename(linkPath, tempBakPath); err != nil {
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("rename platform dir failed: %v", err))
	}

	if err := os.MkdirAll(linkPath, 0o755); err != nil {
		_ = os.Rename(tempBakPath, linkPath)
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("create real dir failed: %v", err))
	}

	// 为迁移后已在 store 中的 skill 建 per-skill 链接
	if entries, readErr := os.ReadDir(userSkillsPath); readErr == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := entry.Name()
			if _, statErr := os.Stat(filepath.Join(userSkillsPath, dir, "SKILL.md")); statErr != nil {
				continue
			}
			_ = ensureSkillAgentLink(platform, dir) // 尽力而为，不中断迁移
		}
	}

	if err := os.RemoveAll(tempBakPath); err != nil {
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("cleanup temp bak failed: %v", err))
	}

	return nil
}

func (ss *SkillService) mergePlatformSkills(platform, platformPath, userSkillsPath string, store *skillStore) error {
	entries, err := os.ReadDir(platformPath)
	if err != nil {
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, err.Error())
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sourceDir := filepath.Join(platformPath, entry.Name())
		targetDir := filepath.Join(userSkillsPath, entry.Name())
		if _, err := os.Stat(filepath.Join(sourceDir, "SKILL.md")); err != nil {
			continue
		}

		if !dirExists(targetDir) {
			if err := copyDirectory(sourceDir, targetDir); err != nil {
				return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("copy %s failed: %v", entry.Name(), err))
			}
			store.Migrations = append(store.Migrations, migrationRecord{
				Platform:  platform,
				Directory: entry.Name(),
				Status:    skillMigrationStatusMigrated,
				CreatedAt: time.Now(),
			})
			continue
		}

		sourceFingerprint, err := ss.computeSkillFingerprint(sourceDir)
		if err != nil {
			return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("fingerprint %s failed: %v", entry.Name(), err))
		}
		targetFingerprint, err := ss.computeSkillFingerprint(targetDir)
		if err != nil {
			return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("fingerprint %s failed: %v", entry.Name(), err))
		}

		if sourceFingerprint == targetFingerprint {
			store.Migrations = append(store.Migrations, migrationRecord{
				Platform:  platform,
				Directory: entry.Name(),
				Status:    skillMigrationStatusDuplicate,
				CreatedAt: time.Now(),
			})
			continue
		}

		store.Migrations = append(store.Migrations, migrationRecord{
			Platform:  platform,
			Directory: entry.Name(),
			Status:    skillMigrationStatusConflict,
			Message:   "skill content differs across sources",
			CreatedAt: time.Now(),
		})
		return fmt.Errorf("%s skill %s 内容冲突", platform, entry.Name())
	}

	return nil
}

func (ss *SkillService) backupPlatformDirLocked(platform string, store *skillStore) (backupRecord, error) {
	platformPath := getPlatformSkillsLinkPath(platform)
	info, err := os.Lstat(platformPath)
	if err != nil {
		return backupRecord{}, err
	}
	if isSymlink(info) || !info.IsDir() {
		return backupRecord{}, fmt.Errorf("%s skills 目录不是可备份的实体目录", platform)
	}

	backupRoot := getSkillBackupRoot()
	backupPath := filepath.Join(backupRoot, fmt.Sprintf("%s-%d", platform, time.Now().UnixNano()))
	if err := copyDirectory(platformPath, backupPath); err != nil {
		return backupRecord{}, err
	}

	record := backupRecord{
		Platform:  platform,
		Path:      backupPath,
		CreatedAt: time.Now(),
	}
	store.Backups = append(store.Backups, record)
	return record, nil
}

func (ss *SkillService) appendPlatformError(store *skillStore, platform, status, message string) error {
	store.Migrations = append(store.Migrations, migrationRecord{
		Platform:  platform,
		Status:    status,
		Message:   message,
		CreatedAt: time.Now(),
	})
	return fmt.Errorf("%s: %s", platform, message)
}

func (ss *SkillService) recoverPlatformDirIfNeeded(platform string) error {
	linkPath := getPlatformSkillsLinkPath(platform)
	if linkPath == "" {
		return nil
	}

	if _, err := os.Lstat(linkPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	matches, err := filepath.Glob(linkPath + ".bak-*")
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return nil
	}

	sort.Strings(matches)
	return os.Rename(matches[len(matches)-1], linkPath)
}

func countInstalledSkillDirs(root string) int {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), "SKILL.md")); err == nil {
			count++
		}
	}
	return count
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func resolveLinkTarget(linkPath string) (string, error) {
	target, err := os.Readlink(linkPath)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(linkPath), target)
	}
	return filepath.Clean(target), nil
}

func isSymlink(info fs.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
