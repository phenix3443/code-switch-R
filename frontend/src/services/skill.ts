import { Call } from '@wailsio/runtime'

export type SkillAgents = {
  claude: boolean
  codex: boolean
}

export type SkillSummary = {
  key: string
  name: string
  description: string
  directory: string
  readme_url: string
  installed: boolean
  agents: SkillAgents
  enabled: boolean
  license_file?: string
  source_group_key: string
  source_group_label: string
  repo_owner?: string
  repo_name?: string
  repo_branch?: string
}

export type SkillGroup = {
  group_key: string
  group_label: string
  skills: SkillSummary[]
}

export type GroupedSkills = {
  installed: SkillGroup[]
  available: SkillGroup[]
}

export type SkillRepoConfig = {
  owner: string
  name: string
  branch: string
  enabled: boolean
}

export type SkillLinkEntry = {
  platform: string
  status: string
  target?: string
  error?: string
  linked_skill_count?: number
}

export type SkillLinkStatus = {
  user_skills_exists: boolean
  user_skill_count: number
  claude: SkillLinkEntry
  codex: SkillLinkEntry
}

export type BackupRecord = {
  platform?: string
  path?: string
  created_at?: string
}

export type MigrationRecord = {
  platform?: string
  directory?: string
  status?: string
  message?: string
  created_at?: string
}

export type SkillDiagnostics = {
  user_skills_path: string
  backup_root: string
  claude_link_path: string
  codex_link_path: string
  backups: BackupRecord[]
  migrations: MigrationRecord[]
}

export const fetchSkills = async (): Promise<SkillSummary[]> => {
  const response = await Call.ByName('codeswitch/services.SkillService.ListSkills')
  return (response as SkillSummary[]) ?? []
}

export const fetchInstalledSkills = async (): Promise<SkillSummary[]> => {
  const response = await Call.ByName('codeswitch/services.SkillService.ListInstalledSkills')
  return (response as SkillSummary[]) ?? []
}

export const fetchAvailableSkills = async (): Promise<SkillSummary[]> => {
  const response = await Call.ByName('codeswitch/services.SkillService.ListAvailableSkills')
  return (response as SkillSummary[]) ?? []
}

export const fetchGroupedSkills = async (): Promise<GroupedSkills> => {
  const response = await Call.ByName('codeswitch/services.SkillService.ListGroupedSkills')
  return (response as GroupedSkills) ?? { installed: [], available: [] }
}

export const fetchSkillLinkStatus = async (): Promise<SkillLinkStatus> => {
  const response = await Call.ByName('codeswitch/services.SkillService.GetSkillLinkStatus')
  return response as SkillLinkStatus
}

export const ensureSkillLinks = async (): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.EnsureSkillLinks')
}

export const fetchSkillDiagnostics = async (): Promise<SkillDiagnostics> => {
  const response = await Call.ByName('codeswitch/services.SkillService.GetSkillDiagnostics')
  return response as SkillDiagnostics
}

export const installSkill = async (
  directory: string,
  repoOwner = '',
  repoName = '',
  repoBranch = '',
  agents: string[] = []
): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.InstallSkill', directory, repoOwner, repoName, repoBranch, agents)
}

export const uninstallSkill = async (directory: string, agents: string[] = []): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.UninstallSkill', directory, agents)
}

export const toggleSkill = async (directory: string, enabled: boolean): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.ToggleSkill', directory, enabled)
}

export const getSkillContent = async (directory: string): Promise<string> => {
  const response = await Call.ByName('codeswitch/services.SkillService.GetSkillContent', directory)
  return response as string
}

export const saveSkillContent = async (directory: string, content: string): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.SaveSkillContent', directory, content)
}

export const openUserSkillsFolder = async (): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.OpenUserSkillsFolder')
}

export const openInstalledSkillFolder = async (directory: string): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.OpenInstalledSkillFolder', directory)
}

export const openPlatformSkillsLink = async (platform: string): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.OpenPlatformSkillsLink', platform)
}

export const openSkillBackup = async (path: string): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.OpenSkillBackup', path)
}

export const fetchSkillRepos = async (): Promise<SkillRepoConfig[]> => {
  const response = await Call.ByName('codeswitch/services.SkillService.ListRepos')
  return (response as SkillRepoConfig[]) ?? []
}

export const addSkillRepo = async (repo: Partial<SkillRepoConfig>): Promise<SkillRepoConfig[]> => {
  const payload = {
    owner: repo.owner ?? '',
    name: repo.name ?? '',
    branch: repo.branch ?? 'main',
    enabled: repo.enabled ?? true
  }
  const response = await Call.ByName('codeswitch/services.SkillService.AddRepo', payload)
  return (response as SkillRepoConfig[]) ?? []
}

export const removeSkillRepo = async (owner: string, name: string): Promise<SkillRepoConfig[]> => {
  const response = await Call.ByName('codeswitch/services.SkillService.RemoveRepo', owner, name)
  return (response as SkillRepoConfig[]) ?? []
}
