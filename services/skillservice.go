package services

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	skillStoreDir  = ".code-switch"
	skillStoreFile = "skill.json"

	// 平台常量
	skillPlatformClaude = "claude"
	skillPlatformCodex  = "codex"
)

var (
	defaultRepoBranches = []string{"main", "master"}
	defaultSkillRepos   = []skillRepoConfig{
		{Owner: "ComposioHQ", Name: "awesome-claude-skills", Branch: "main", Enabled: true},
		{Owner: "anthropics", Name: "skills", Branch: "main", Enabled: true},
	}
)

type Skill struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Directory   string `json:"directory"`
	ReadmeURL   string `json:"readme_url"`
	Installed   bool   `json:"installed"`

	Enabled          bool   `json:"enabled"`                 // 是否启用（从 SKILL.md 读取）
	LicenseFile      string `json:"license_file,omitempty"`  // 许可证文件路径
	SourceGroupKey   string `json:"source_group_key"`        // 分组 key
	SourceGroupLabel string `json:"source_group_label"`      // 分组标签

	// 仓库字段
	RepoOwner  string `json:"repo_owner,omitempty"`
	RepoName   string `json:"repo_name,omitempty"`
	RepoBranch string `json:"repo_branch,omitempty"`
}

type SkillGroup struct {
	GroupKey   string  `json:"group_key"`
	GroupLabel string  `json:"group_label"`
	Skills     []Skill `json:"skills"`
}

type GroupedSkills struct {
	Installed []SkillGroup `json:"installed"`
	Available []SkillGroup `json:"available"`
}

type skillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// skillMetadataExtended 扩展的元数据结构（包含 enabled 状态相关字段）
type skillMetadataExtended struct {
	Name                   string `yaml:"name"`
	Description            string `yaml:"description"`
	DisableModelInvocation bool   `yaml:"disable-model-invocation"`
	UserInvocable          *bool  `yaml:"user-invocable"`
}

type skillStore struct {
	Skills           map[string]skillState      `json:"skills,omitempty"`
	EnabledOverrides map[string]bool            `json:"enabled_overrides,omitempty"`
	Provenance       map[string]skillProvenance `json:"provenance,omitempty"`
	Migrations       []migrationRecord          `json:"migrations,omitempty"`
	Backups          []backupRecord             `json:"backups,omitempty"`
	Repos            []skillRepoConfig          `json:"repos"`
}

type skillState struct {
	Installed   bool      `json:"installed"`
	InstalledAt time.Time `json:"installed_at,omitempty"`
}

type skillProvenance struct {
	Type       string `json:"type"`
	RepoOwner  string `json:"repo_owner,omitempty"`
	RepoName   string `json:"repo_name,omitempty"`
	RepoBranch string `json:"repo_branch,omitempty"`
}

type migrationRecord struct {
	Platform  string    `json:"platform,omitempty"`
	Directory string    `json:"directory,omitempty"`
	Status    string    `json:"status,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type backupRecord struct {
	Platform  string    `json:"platform,omitempty"`
	Path      string    `json:"path,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type skillRepoConfig struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Branch  string `json:"branch"`
	Enabled bool   `json:"enabled"`
}

type SkillService struct {
	httpClient     *http.Client
	storePath      string
	installDir     string
	repoSnapshotter func(skillRepoConfig) (string, string, func(), error)
	mu             sync.Mutex
}

func NewSkillService() *SkillService {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return &SkillService{
		httpClient:      &http.Client{Timeout: 60 * time.Second},
		storePath:       filepath.Join(home, skillStoreDir, skillStoreFile),
		installDir:      getUserSkillsPath(),
		repoSnapshotter: nil,
	}
}

func getUserSkillsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".agent", "skills")
}

func getPlatformSkillsLinkPath(platform string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	switch strings.ToLower(strings.TrimSpace(platform)) {
	case skillPlatformClaude:
		return filepath.Join(home, ".claude", "skills")
	case skillPlatformCodex:
		return filepath.Join(home, ".codex", "skills")
	default:
		return ""
	}
}

func getSkillBackupRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, skillStoreDir, "backups", "skills")
}

// readSkillMetadataExtended 读取技能元数据（包括 enabled 状态）
func (ss *SkillService) readSkillMetadataExtended(dir string) (skillMetadataExtended, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return skillMetadataExtended{}, false, err
	}

	meta, err := parseSkillMetadataExtended(string(data))
	if err != nil {
		return skillMetadataExtended{}, false, err
	}

	// enabled = NOT disable-model-invocation
	enabled := !meta.DisableModelInvocation

	return meta, enabled, nil
}

// parseSkillMetadataExtended 解析扩展元数据
func parseSkillMetadataExtended(content string) (skillMetadataExtended, error) {
	var meta skillMetadataExtended
	content = strings.TrimLeft(content, "\ufeff")

	// 使用 splitFrontMatter 替代 strings.SplitN，避免 YAML 值中的 --- 被误判
	_, fmLines, _, err := splitFrontMatter(content)
	if err != nil {
		return meta, errors.New("SKILL.md 缺少 front matter")
	}

	frontMatter := strings.Join(fmLines, "\n")
	if err := yaml.Unmarshal([]byte(frontMatter), &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

// ListInstalledSkills 返回统一目录中的已安装 skills。
func (ss *SkillService) ListInstalledSkills() ([]Skill, error) {
	store, err := ss.loadStore()
	if err != nil {
		return nil, err
	}

	return ss.scanInstalledSkills(store), nil
}

// ListAvailableSkills 返回远程 catalog 中可安装 skills。
func (ss *SkillService) ListAvailableSkills() ([]Skill, error) {
	store, err := ss.loadStore()
	if err != nil {
		return nil, err
	}

	skillMap := make(map[string]Skill)
	for _, repo := range store.Repos {
		if !repo.Enabled {
			continue
		}
		repoDir, branch, cleanup, err := ss.prepareRepoSnapshot(repo)
		if err != nil {
			log.Printf("skill repo fetch failed for %s/%s: %v", repo.Owner, repo.Name, err)
			continue
		}
		entries, err := os.ReadDir(repoDir)
		if err != nil {
			cleanup()
			log.Printf("skill repo read failed for %s/%s: %v", repo.Owner, repo.Name, err)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dirKey := normalizeDirectoryKey(entry.Name())
			if _, exists := skillMap[dirKey]; exists {
				continue
			}
			skillPath := filepath.Join(repoDir, entry.Name())
			meta, err := readSkillMetadata(skillPath)
			if err != nil {
				continue
			}
			name := strings.TrimSpace(meta.Name)
			if name == "" {
				name = entry.Name()
			}
			groupKey, groupLabel := buildSourceGroup("github", repo.Owner, repo.Name)
			skillMap[dirKey] = Skill{
				Key:              buildSkillKey(repo.Owner, repo.Name, entry.Name()),
				Name:             name,
				Description:      strings.TrimSpace(meta.Description),
				Directory:        entry.Name(),
				ReadmeURL:        buildRepoURL(repo, branch, entry.Name()),
				Installed:        false,
				RepoOwner:        repo.Owner,
				RepoName:         repo.Name,
				RepoBranch:       branch,
				SourceGroupKey:   groupKey,
				SourceGroupLabel: groupLabel,
			}
		}
		cleanup()
	}

	skills := make([]Skill, 0, len(skillMap))
	for _, skill := range skillMap {
		skills = append(skills, skill)
	}
	sortSkills(skills)
	return skills, nil
}

// ListGroupedSkills 返回按来源仓库分组的 installed/available 数据。
func (ss *SkillService) ListGroupedSkills() (GroupedSkills, error) {
	installed, err := ss.ListInstalledSkills()
	if err != nil {
		return GroupedSkills{}, err
	}
	available, err := ss.ListAvailableSkills()
	if err != nil {
		return GroupedSkills{}, err
	}

	installedDirs := make(map[string]struct{}, len(installed))
	for _, skill := range installed {
		installedDirs[normalizeDirectoryKey(skill.Directory)] = struct{}{}
	}

	filteredAvailable := make([]Skill, 0, len(available))
	for _, skill := range available {
		if _, exists := installedDirs[normalizeDirectoryKey(skill.Directory)]; exists {
			continue
		}
		filteredAvailable = append(filteredAvailable, skill)
	}

	return GroupedSkills{
		Installed: groupSkills(installed),
		Available: groupSkills(filteredAvailable),
	}, nil
}

// ListSkills 保持兼容，返回 installed + available 平铺结果。
func (ss *SkillService) ListSkills() ([]Skill, error) {
	installed, err := ss.ListInstalledSkills()
	if err != nil {
		return nil, err
	}
	available, err := ss.ListAvailableSkills()
	if err != nil {
		return nil, err
	}

	skillMap := make(map[string]Skill, len(installed)+len(available))
	for _, skill := range available {
		skillMap[normalizeDirectoryKey(skill.Directory)] = skill
	}
	for _, skill := range installed {
		skillMap[normalizeDirectoryKey(skill.Directory)] = skill
	}

	skills := make([]Skill, 0, len(skillMap))
	for _, skill := range skillMap {
		skills = append(skills, skill)
	}
	sortSkills(skills)
	return skills, nil
}

// InstallSkill installs a skill directory from the configured repositories.
func (ss *SkillService) InstallSkill(directory, repoOwner, repoName, repoBranch string) error {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		return errors.New("skill directory 不能为空")
	}
	if err := ss.EnsureSkillLinks(); err != nil {
		return err
	}

	store, err := ss.loadStore()
	if err != nil {
		return err
	}
	repos := ss.resolveReposForInstall(repoOwner, repoName, store.Repos)
	if len(repos) == 0 {
		return errors.New("未找到可用的技能仓库")
	}

	var lastErr error
	for _, repo := range repos {
		repoDir, _, cleanup, err := ss.prepareRepoSnapshot(repo)
		if err != nil {
			lastErr = err
			continue
		}
		skillPath := filepath.Join(repoDir, directory)
		info, err := os.Stat(skillPath)
		if err != nil || !info.IsDir() {
			cleanup()
			lastErr = fmt.Errorf("仓库 %s/%s 中未找到 %s", repo.Owner, repo.Name, directory)
			continue
		}
		provenance := skillProvenance{
			Type:       "github",
			RepoOwner:  repo.Owner,
			RepoName:   repo.Name,
			RepoBranch: repo.Branch,
		}
		if repoBranch != "" {
			provenance.RepoBranch = repoBranch
		}
		if err := ss.installFromPathWithProvenance(directory, skillPath, provenance); err != nil {
			cleanup()
			lastErr = err
			continue
		}
		cleanup()
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("skill %s 未找到", directory)
	}
	return lastErr
}

func (ss *SkillService) installFromPath(directory, source string) error {
	return ss.installFromPathWithProvenance(directory, source, skillProvenance{Type: "local"})
}

func (ss *SkillService) installFromPathWithProvenance(directory, source string, provenance skillProvenance) error {
	if err := ss.EnsureSkillLinks(); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(source, "SKILL.md")); err != nil {
		return fmt.Errorf("%s 缺少 SKILL.md", directory)
	}

	if err := os.MkdirAll(ss.installDir, 0o755); err != nil {
		return err
	}

	ss.mu.Lock()
	defer ss.mu.Unlock()

	store, err := ss.loadStoreLocked()
	if err != nil {
		return err
	}
	if isProvenanceConflict(store.Provenance[directory], provenance) {
		return fmt.Errorf("skill %s 目录名已被其他来源占用", directory)
	}

	target := filepath.Join(ss.installDir, directory)
	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := copyDirectory(source, target); err != nil {
		return err
	}

	store.Provenance[directory] = normalizeSkillProvenance(provenance)
	return ss.saveStoreLocked(store)
}

func (ss *SkillService) UninstallSkill(directory string) error {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		return errors.New("skill directory 不能为空")
	}
	target := filepath.Join(ss.installDir, directory)
	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	store, err := ss.loadStoreLocked()
	if err != nil {
		return err
	}
	if store.Skills == nil {
		store.Skills = make(map[string]skillState)
	}
	delete(store.Skills, directory)
	delete(store.Provenance, directory)
	delete(store.EnabledOverrides, directory)
	return ss.saveStoreLocked(store)
}

// ToggleSkill 切换技能的启用状态
// 通过修改 SKILL.md 的 disable-model-invocation 字段实现
func (ss *SkillService) ToggleSkill(directory string, enabled bool) error {
	if directory == "" {
		return errors.New("skill directory 不能为空")
	}
	if err := ss.EnsureSkillLinks(); err != nil {
		return err
	}

	ss.mu.Lock()
	defer ss.mu.Unlock()

	store, err := ss.loadStoreLocked()
	if err != nil {
		return err
	}
	store.EnabledOverrides[directory] = enabled

	if err := ss.applyEnabledOverride(filepath.Join(ss.installDir, directory, "SKILL.md"), enabled); err != nil {
		return err
	}
	return ss.saveStoreLocked(store)
}

func (ss *SkillService) applyEnabledOverride(skillMDPath string, enabled bool) error {
	data, err := os.ReadFile(skillMDPath)
	if err != nil {
		return fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}

	newContent, changed, err := patchSkillFrontMatterBool(string(data), "disable-model-invocation", !enabled)
	if err != nil {
		return fmt.Errorf("修改 SKILL.md 失败: %w", err)
	}
	if !changed {
		return nil
	}
	return AtomicWriteBytes(skillMDPath, []byte(newContent))
}

// splitFrontMatter 使用行首匹配 ^---\s*$ 来分割 front matter
// 返回 (prefix, frontMatterLines, body, error)
// prefix: 开始 --- 之前的行
// frontMatterLines: front matter 内容（不含边界行）
// body: 结束 --- 之后的所有内容
func splitFrontMatter(content string) (prefix string, fmLines []string, body string, err error) {
	// 统一行尾为 \n 进行处理
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	startIdx := -1
	endIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if startIdx == -1 {
				startIdx = i
			} else {
				endIdx = i
				break
			}
		}
	}

	if startIdx == -1 || endIdx == -1 {
		return "", nil, "", errors.New("无法解析 front matter：未找到有效的 --- 边界")
	}

	// prefix: lines[0:startIdx]
	if startIdx > 0 {
		prefix = strings.Join(lines[:startIdx], "\n") + "\n"
	}

	// fmLines: lines[startIdx+1:endIdx]
	fmLines = lines[startIdx+1 : endIdx]

	// body: lines[endIdx+1:]
	if endIdx+1 < len(lines) {
		body = strings.Join(lines[endIdx+1:], "\n")
	}

	return prefix, fmLines, body, nil
}

// patchSkillFrontMatterBool 最小化修改 SKILL.md 的 front matter 中的布尔字段
// 保留原有格式、注释和字段顺序
func patchSkillFrontMatterBool(markdown, key string, desired bool) (string, bool, error) {
	// 1. 保留 BOM
	hasBOM := false
	if strings.HasPrefix(markdown, "\ufeff") {
		hasBOM = true
		markdown = strings.TrimPrefix(markdown, "\ufeff")
	}

	// 2. 检测行尾风格
	hasCRLF := strings.Contains(markdown, "\r\n")

	// 3. 分割 front matter（使用行首匹配，避免内容中的 --- 被误判）
	prefix, lines, body, err := splitFrontMatter(markdown)
	if err != nil {
		return "", false, err
	}

	// 4. 按行处理 front matter
	keyFound := false
	modified := false
	desiredStr := "false"
	if desired {
		desiredStr = "true"
	}

	for i, line := range lines {
		// 移除可能的 \r
		cleanLine := strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(cleanLine)

		// 检查是否匹配目标 key
		if strings.HasPrefix(trimmed, key+":") {
			keyFound = true

			// 提取当前值
			colonIdx := strings.Index(trimmed, ":")
			valuePart := strings.TrimSpace(trimmed[colonIdx+1:])

			// 处理可能的行内注释
			comment := ""
			hashIdx := strings.Index(valuePart, "#")
			if hashIdx != -1 {
				comment = valuePart[hashIdx:]
				valuePart = strings.TrimSpace(valuePart[:hashIdx])
			}

			// 检查是否需要修改
			currentBool := strings.ToLower(valuePart) == "true"
			if currentBool == desired {
				continue // 值已经正确，无需修改
			}

			// 构建新行（保留原有缩进）
			indent := ""
			for _, ch := range cleanLine {
				if ch == ' ' || ch == '\t' {
					indent += string(ch)
				} else {
					break
				}
			}

			newLine := indent + key + ": " + desiredStr
			if comment != "" {
				newLine += " " + comment
			}

			lines[i] = newLine
			modified = true
		}
	}

	// 5. 如果 key 不存在，在 front matter 末尾插入
	if !keyFound {
		insertLine := key + ": " + desiredStr
		// 在最后一行（通常是空行）之前插入
		insertIdx := len(lines) - 1
		for insertIdx > 0 && strings.TrimSpace(lines[insertIdx]) == "" {
			insertIdx--
		}
		insertIdx++

		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, insertLine)
		newLines = append(newLines, lines[insertIdx:]...)
		lines = newLines
		modified = true
	}

	// 6. 重建文档
	newFrontMatter := strings.Join(lines, "\n")
	result := prefix + "---\n" + newFrontMatter + "\n---\n" + body

	// 7. 恢复 CRLF（如果原文使用）
	if hasCRLF {
		// 先统一为 LF，再替换为 CRLF
		result = strings.ReplaceAll(result, "\r\n", "\n")
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}

	// 8. 恢复 BOM
	if hasBOM {
		result = "\ufeff" + result
	}

	return result, modified, nil
}

// GetSkillContent 获取技能的 SKILL.md 内容
func (ss *SkillService) GetSkillContent(directory string) (string, error) {
	if directory == "" {
		return "", errors.New("skill directory 不能为空")
	}
	skillMDPath := filepath.Join(ss.installDir, directory, "SKILL.md")
	data, err := os.ReadFile(skillMDPath)
	if err != nil {
		return "", fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}

	return string(data), nil
}

// SaveSkillContent 保存技能的 SKILL.md 内容
func (ss *SkillService) SaveSkillContent(directory, content string) error {
	if directory == "" {
		return errors.New("skill directory 不能为空")
	}
	skillMDPath := filepath.Join(ss.installDir, directory, "SKILL.md")
	return AtomicWriteBytes(skillMDPath, []byte(content))
}

// OpenUserSkillsFolder 打开统一技能目录
func (ss *SkillService) OpenUserSkillsFolder() error {
	if err := os.MkdirAll(ss.installDir, 0o755); err != nil {
		return err
	}
	return OpenInExplorer(ss.installDir)
}

// Repository management ----------------------------------------------------

func (ss *SkillService) ListRepos() ([]skillRepoConfig, error) {
	store, err := ss.loadStore()
	if err != nil {
		return nil, err
	}
	return cloneRepoConfigs(store.Repos), nil
}

func (ss *SkillService) AddRepo(repo skillRepoConfig) ([]skillRepoConfig, error) {
	repo = normalizeRepoConfig(repo)
	if err := validateRepoConfig(repo); err != nil {
		return nil, err
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	store, err := ss.loadStoreLocked()
	if err != nil {
		return nil, err
	}
	replaced := false
	for i := range store.Repos {
		if equalRepo(store.Repos[i], repo) {
			store.Repos[i] = repo
			replaced = true
			break
		}
	}
	if !replaced {
		store.Repos = append(store.Repos, repo)
	}
	if err := ss.saveStoreLocked(store); err != nil {
		return nil, err
	}
	return cloneRepoConfigs(store.Repos), nil
}

func (ss *SkillService) RemoveRepo(owner, name string) ([]skillRepoConfig, error) {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" || name == "" {
		return nil, errors.New("owner/name 不能为空")
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	store, err := ss.loadStoreLocked()
	if err != nil {
		return nil, err
	}
	filtered := make([]skillRepoConfig, 0, len(store.Repos))
	for _, repo := range store.Repos {
		if strings.EqualFold(repo.Owner, owner) && strings.EqualFold(repo.Name, name) {
			continue
		}
		filtered = append(filtered, repo)
	}
	if len(filtered) == 0 {
		filtered = cloneDefaultRepos()
	}
	store.Repos = filtered
	if err := ss.saveStoreLocked(store); err != nil {
		return nil, err
	}
	return cloneRepoConfigs(store.Repos), nil
}

// Internal helpers ---------------------------------------------------------

func (ss *SkillService) loadStore() (skillStore, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.loadStoreLocked()
}

func (ss *SkillService) loadStoreLocked() (skillStore, error) {
	data, err := os.ReadFile(ss.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			store := skillStore{}
			store.ensureState()
			return store, nil
		}
		return newDefaultSkillStore(), err
	}
	store := skillStore{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &store); err != nil {
			return newDefaultSkillStore(), err
		}
	}
	store.ensureState()
	return store, nil
}

func newDefaultSkillStore() skillStore {
	store := skillStore{}
	store.ensureState()
	return store
}

func (store *skillStore) ensureState() {
	if store.Skills == nil {
		store.Skills = make(map[string]skillState)
	}
	if store.EnabledOverrides == nil {
		store.EnabledOverrides = make(map[string]bool)
	}
	if store.Provenance == nil {
		store.Provenance = make(map[string]skillProvenance)
	}
	if store.Migrations == nil {
		store.Migrations = make([]migrationRecord, 0)
	}
	if store.Backups == nil {
		store.Backups = make([]backupRecord, 0)
	}
	store.migrateLegacySkills()
	store.ensureRepos()
}

func normalizeSkillProvenance(provenance skillProvenance) skillProvenance {
	provenance.Type = strings.TrimSpace(provenance.Type)
	if provenance.Type == "" {
		provenance.Type = "local"
	}
	provenance.RepoOwner = strings.TrimSpace(provenance.RepoOwner)
	provenance.RepoName = strings.TrimSpace(provenance.RepoName)
	provenance.RepoBranch = strings.TrimSpace(provenance.RepoBranch)
	if provenance.Type == "github" && provenance.RepoBranch == "" {
		provenance.RepoBranch = "main"
	}
	return provenance
}

func isProvenanceConflict(existing, incoming skillProvenance) bool {
	existing = normalizeSkillProvenance(existing)
	incoming = normalizeSkillProvenance(incoming)
	if existing.Type == "" {
		return false
	}
	if existing.Type != incoming.Type {
		return true
	}
	if existing.Type != "github" {
		return false
	}
	return !strings.EqualFold(existing.RepoOwner, incoming.RepoOwner) ||
		!strings.EqualFold(existing.RepoName, incoming.RepoName) ||
		!strings.EqualFold(existing.RepoBranch, incoming.RepoBranch)
}

func (store *skillStore) migrateLegacySkills() {
	for directory := range store.Skills {
		directory = strings.TrimSpace(directory)
		if directory == "" {
			continue
		}
		if _, exists := store.Provenance[directory]; exists {
			continue
		}
		store.Provenance[directory] = skillProvenance{Type: "local"}
	}
}

func (store *skillStore) ensureRepos() {
	if len(store.Repos) == 0 {
		store.Repos = cloneDefaultRepos()
	}
	for i := range store.Repos {
		store.Repos[i] = normalizeRepoConfig(store.Repos[i])
		if !store.Repos[i].Enabled {
			store.Repos[i].Enabled = true
		}
	}
}

func cloneDefaultRepos() []skillRepoConfig {
	repos := make([]skillRepoConfig, len(defaultSkillRepos))
	copy(repos, defaultSkillRepos)
	return repos
}

func cloneRepoConfigs(repos []skillRepoConfig) []skillRepoConfig {
	copyRepos := make([]skillRepoConfig, len(repos))
	copy(copyRepos, repos)
	return copyRepos
}

func normalizeRepoConfig(repo skillRepoConfig) skillRepoConfig {
	repo.Owner = strings.TrimSpace(repo.Owner)
	repo.Name = strings.TrimSpace(repo.Name)
	repo.Branch = strings.TrimSpace(repo.Branch)
	if repo.Branch == "" {
		repo.Branch = "main"
	}
	if !repo.Enabled {
		repo.Enabled = true
	}
	return repo
}

func validateRepoConfig(repo skillRepoConfig) error {
	if repo.Owner == "" || repo.Name == "" {
		return errors.New("owner/name 不能为空")
	}
	return nil
}

func equalRepo(a, b skillRepoConfig) bool {
	return strings.EqualFold(a.Owner, b.Owner) && strings.EqualFold(a.Name, b.Name)
}

func (ss *SkillService) saveStoreLocked(store skillStore) error {
	if err := os.MkdirAll(filepath.Dir(ss.storePath), 0o755); err != nil {
		return err
	}
	store.ensureState()
	persisted := store
	persisted.Skills = nil
	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}
	tmp := ss.storePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ss.storePath)
}

func (ss *SkillService) prepareRepoSnapshot(repo skillRepoConfig) (string, string, func(), error) {
	if ss.repoSnapshotter != nil {
		return ss.repoSnapshotter(repo)
	}
	tmpDir, err := os.MkdirTemp("", "skill-repo-")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}
	archivePath := filepath.Join(tmpDir, "repo.zip")
	branches := buildBranchCandidates(repo.Branch)
	var lastErr error
	for _, branch := range branches {
		archiveURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", repo.Owner, repo.Name, branch)
		if err := ss.downloadFile(archiveURL, archivePath); err != nil {
			lastErr = err
			continue
		}
		rootDir, err := unzipArchive(archivePath, tmpDir)
		if err != nil {
			lastErr = err
			continue
		}
		return rootDir, branch, cleanup, nil
	}
	cleanup()
	if lastErr == nil {
		lastErr = fmt.Errorf("无法下载仓库 %s/%s", repo.Owner, repo.Name)
	}
	return "", "", nil, lastErr
}

func buildBranchCandidates(preferred string) []string {
	set := make(map[string]struct{})
	ordered := make([]string, 0, len(defaultRepoBranches)+1)
	if preferred != "" {
		set[strings.ToLower(preferred)] = struct{}{}
		ordered = append(ordered, preferred)
	}
	for _, branch := range defaultRepoBranches {
		key := strings.ToLower(branch)
		if _, ok := set[key]; ok {
			continue
		}
		set[key] = struct{}{}
		ordered = append(ordered, branch)
	}
	return ordered
}

func (ss *SkillService) downloadFile(rawURL, dest string) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ai-code-studio")
	resp, err := ss.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: %s", resp.Status)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

func unzipArchive(zipPath, dest string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	var root string
	for _, file := range reader.File {
		name := file.Name
		if name == "" {
			continue
		}
		if root == "" {
			root = strings.Split(name, "/")[0]
		}
		targetPath := filepath.Join(dest, name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return "", err
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			src.Close()
			return "", err
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			return "", err
		}
		src.Close()
		dst.Close()
	}
	if root == "" {
		return "", errors.New("压缩包内容为空")
	}
	return filepath.Join(dest, root), nil
}

func (ss *SkillService) scanInstalledSkills(store skillStore) []Skill {
	entries, err := os.ReadDir(ss.installDir)
	if err != nil {
		return []Skill{}
	}

	skills := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := entry.Name()
		skillPath := filepath.Join(ss.installDir, dir)
		meta, enabled, err := ss.readSkillMetadataExtended(skillPath)
		if err != nil {
			continue
		}

		name := strings.TrimSpace(meta.Name)
		if name == "" {
			name = dir
		}
		licenseFile := ""
		for _, lf := range []string{"LICENSE", "LICENSE.txt", "LICENSE.md"} {
			if _, err := os.Stat(filepath.Join(skillPath, lf)); err == nil {
				licenseFile = lf
				break
			}
		}

		provenance := normalizeSkillProvenance(store.Provenance[dir])
		groupKey, groupLabel := buildSourceGroup(provenance.Type, provenance.RepoOwner, provenance.RepoName)
		skills = append(skills, Skill{
			Key:              buildSkillKey(provenance.RepoOwner, provenance.RepoName, dir),
			Name:             name,
			Description:      strings.TrimSpace(meta.Description),
			Directory:        dir,
			Installed:        true,
			Enabled:          enabled,
			LicenseFile:      licenseFile,
			RepoOwner:        provenance.RepoOwner,
			RepoName:         provenance.RepoName,
			RepoBranch:       provenance.RepoBranch,
			SourceGroupKey:   groupKey,
			SourceGroupLabel: groupLabel,
		})
	}

	sortSkills(skills)
	return skills
}

func (ss *SkillService) resolveReposForInstall(owner, name string, repos []skillRepoConfig) []skillRepoConfig {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	var target []skillRepoConfig
	if owner != "" && name != "" {
		for _, repo := range repos {
			if !repo.Enabled {
				continue
			}
			if strings.EqualFold(repo.Owner, owner) && strings.EqualFold(repo.Name, name) {
				target = append(target, repo)
			}
		}
		return target
	}
	for _, repo := range repos {
		if repo.Enabled {
			target = append(target, repo)
		}
	}
	return target
}

func groupSkills(skills []Skill) []SkillGroup {
	groups := make([]SkillGroup, 0)
	indexByKey := make(map[string]int)

	for _, skill := range skills {
		groupKey := skill.SourceGroupKey
		groupLabel := skill.SourceGroupLabel
		if groupKey == "" || groupLabel == "" {
			groupKey, groupLabel = buildSourceGroup("", "", "")
			skill.SourceGroupKey = groupKey
			skill.SourceGroupLabel = groupLabel
		}

		idx, exists := indexByKey[groupKey]
		if !exists {
			indexByKey[groupKey] = len(groups)
			groups = append(groups, SkillGroup{
				GroupKey:   groupKey,
				GroupLabel: groupLabel,
				Skills:     []Skill{skill},
			})
			continue
		}
		groups[idx].Skills = append(groups[idx].Skills, skill)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].GroupLabel) < strings.ToLower(groups[j].GroupLabel)
	})
	for i := range groups {
		sortSkills(groups[i].Skills)
	}
	return groups
}

func buildRepoURL(repo skillRepoConfig, branch, directory string) string {
	dir := strings.Trim(directory, "/")
	if dir == "" {
		return fmt.Sprintf("https://github.com/%s/%s", repo.Owner, repo.Name)
	}
	return fmt.Sprintf("https://github.com/%s/%s/tree/%s/%s", repo.Owner, repo.Name, branch, dir)
}

func buildSkillKey(owner, name, directory string) string {
	owner = strings.ToLower(strings.TrimSpace(owner))
	name = strings.ToLower(strings.TrimSpace(name))
	directory = strings.ToLower(directory)
	if owner == "" && name == "" {
		return fmt.Sprintf("local:%s", directory)
	}
	return fmt.Sprintf("%s/%s:%s", owner, name, directory)
}

func normalizeDirectoryKey(directory string) string {
	return strings.ToLower(strings.TrimSpace(directory))
}

func buildSourceGroup(provenanceType, repoOwner, repoName string) (string, string) {
	if strings.EqualFold(strings.TrimSpace(provenanceType), "github") &&
		strings.TrimSpace(repoOwner) != "" &&
		strings.TrimSpace(repoName) != "" {
		group := fmt.Sprintf("%s/%s", strings.TrimSpace(repoOwner), strings.TrimSpace(repoName))
		return group, group
	}
	return "local", "Local / Unknown Source"
}

func sortSkills(skills []Skill) {
	sort.SliceStable(skills, func(i, j int) bool {
		li := strings.ToLower(skills[i].Name)
		lj := strings.ToLower(skills[j].Name)
		if li == lj {
			return strings.ToLower(skills[i].Directory) < strings.ToLower(skills[j].Directory)
		}
		return li < lj
	})
}

func (ss *SkillService) isInstalled(directory string) bool {
	info, err := os.Stat(filepath.Join(ss.installDir, directory))
	return err == nil && info.IsDir()
}

func readSkillMetadata(dir string) (skillMetadata, error) {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return skillMetadata{}, err
	}
	return parseSkillMetadata(string(data))
}

func parseSkillMetadata(content string) (skillMetadata, error) {
	var meta skillMetadata
	content = strings.TrimLeft(content, "\ufeff")

	// 使用 splitFrontMatter 替代 strings.SplitN，避免 YAML 值中的 --- 被误判
	_, fmLines, _, err := splitFrontMatter(content)
	if err != nil {
		return meta, errors.New("SKILL.md 缺少 front matter")
	}

	frontMatter := strings.Join(fmLines, "\n")
	if err := yaml.Unmarshal([]byte(frontMatter), &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func copyDirectory(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if rel == "." {
				return os.MkdirAll(dst, 0o755)
			}
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
