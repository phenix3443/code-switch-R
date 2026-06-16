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
	Platform string `json:"platform"`
	Status   string `json:"status"`
	Target   string `json:"target,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (ss *SkillService) GetSkillLinkStatus() SkillLinkStatus {
	userSkillsPath := getUserSkillsPath()
	status := SkillLinkStatus{
		UserSkillsExists: dirExists(userSkillsPath),
		UserSkillCount:   countInstalledSkillDirs(userSkillsPath),
		Claude:           ss.getPlatformLinkEntry(skillPlatformClaude, userSkillsPath),
		Codex:            ss.getPlatformLinkEntry(skillPlatformCodex, userSkillsPath),
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

func (ss *SkillService) getPlatformLinkEntry(platform string, userSkillsPath string) SkillLinkEntry {
	entry := SkillLinkEntry{
		Platform: platform,
		Status:   skillLinkStatusMissing,
		Target:   userSkillsPath,
	}

	linkPath := getPlatformSkillsLinkPath(platform)
	if linkPath == "" {
		entry.Status = skillLinkStatusUnsupported
		entry.Target = ""
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
		target, err := resolveLinkTarget(linkPath)
		if err != nil {
			entry.Status = skillLinkStatusRepairFailed
			entry.Error = err.Error()
			entry.Target = ""
			return entry
		}
		entry.Target = target
		if samePath(target, userSkillsPath) {
			entry.Status = skillLinkStatusLinked
			return entry
		}
		entry.Status = skillLinkStatusConflict
		return entry
	}

	if info.IsDir() {
		entry.Status = skillLinkStatusConflict
		entry.Target = linkPath
		return entry
	}

	entry.Status = skillLinkStatusConflict
	entry.Target = linkPath
	entry.Error = "platform skills path is not a directory"
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
			if err := createPlatformLink(userSkillsPath, linkPath); err != nil {
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
			return nil
		}
		return ss.appendPlatformError(store, platform, skillLinkStatusConflict, fmt.Sprintf("existing link points to %s", target))
	}

	if !info.IsDir() {
		return ss.appendPlatformError(store, platform, skillLinkStatusConflict, "platform skills path is not a directory")
	}

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

	if err := createPlatformLink(userSkillsPath, linkPath); err != nil {
		_ = os.Rename(tempBakPath, linkPath)
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("create link failed: %v", err))
	}

	if target, err := resolveLinkTarget(linkPath); err != nil || !samePath(target, userSkillsPath) {
		_ = os.Remove(linkPath)
		_ = os.Rename(tempBakPath, linkPath)
		if err != nil {
			return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("verify link failed: %v", err))
		}
		return ss.appendPlatformError(store, platform, skillLinkStatusRepairFailed, fmt.Sprintf("verify link target mismatch: %s", target))
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
