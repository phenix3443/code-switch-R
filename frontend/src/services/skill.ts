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

const SKILL_SERVICE_TIMEOUT_MS = 6000

const inBrowserDev = () => {
  if (typeof window === 'undefined') return false
  const protocol = window.location.protocol
  const hostname = window.location.hostname
  return (protocol === 'http:' || protocol === 'https:') && (hostname === 'localhost' || hostname === '127.0.0.1')
}

const withTimeout = async <T>(promise: Promise<T>, ms: number): Promise<T> => {
  let timer: ReturnType<typeof setTimeout> | undefined
  try {
    return await Promise.race([
      promise,
      new Promise<T>((_, reject) => {
        timer = setTimeout(() => reject(new Error(`skill service timeout after ${ms}ms`)), ms)
      })
    ])
  } finally {
    if (timer) clearTimeout(timer)
  }
}

const fallbackGroupedSkills = (): GroupedSkills => ({
  installed: [
    {
      group_key: 'local',
      group_label: 'LOCAL',
      skills: [
        {
          key: 'template-skill:local',
          name: 'template-skill',
          description: '统一管理本地 skills 的模板示例，用于验证安装、同步与详情展示。',
          directory: 'template-skill',
          readme_url: 'https://www.skills.sh/qu-skills/skills/ai-video-generation',
          installed: true,
          agents: { claude: true, codex: true },
          enabled: true,
          source_group_key: 'local',
          source_group_label: 'LOCAL'
        }
      ]
    }
  ],
  available: [
    {
      group_key: 'anthropics/skills',
      group_label: 'anthropics/skills',
      skills: [
        {
          key: 'ai-video-generation:anthropics/skills',
          name: 'ai-video-generation',
          description: 'Generate, iterate, and package AI video workflows with reusable skill prompts.',
          directory: 'ai-video-generation',
          readme_url: 'https://www.skills.sh/qu-skills/skills/ai-video-generation',
          installed: false,
          agents: { claude: false, codex: false },
          enabled: false,
          source_group_key: 'anthropics/skills',
          source_group_label: 'anthropics/skills',
          repo_owner: 'anthropics',
          repo_name: 'skills',
          repo_branch: 'main'
        }
      ]
    }
  ]
})

const fallbackLinkStatus = (): SkillLinkStatus => ({
  user_skills_exists: true,
  user_skill_count: 1,
  claude: {
    platform: 'claude',
    status: 'linked',
    target: '~/.agents/skills',
    linked_skill_count: 1
  },
  codex: {
    platform: 'codex',
    status: 'linked',
    target: '~/.agents/skills',
    linked_skill_count: 1
  }
})

const fallbackDiagnostics = (): SkillDiagnostics => ({
  user_skills_path: '~/.agents/skills',
  backup_root: '~/.code-switch/backups/skills',
  claude_link_path: '~/.claude/skills',
  codex_link_path: '~/.codex/skills',
  backups: [],
  migrations: []
})

const fallbackRepos = (): SkillRepoConfig[] => [
  { owner: 'anthropics', name: 'skills', branch: 'main', enabled: true },
  { owner: 'ComposioHQ', name: 'awesome-claude-skills', branch: 'main', enabled: true }
]

const invokeSkillCall = async <T>(method: string, fallback?: () => T, ...args: unknown[]): Promise<T> => {
  try {
    const call = Call.ByName(method, ...args) as Promise<T>
    if (inBrowserDev()) {
      return await withTimeout(call, SKILL_SERVICE_TIMEOUT_MS)
    }
    return await call
  } catch (error) {
    if (fallback && inBrowserDev()) {
      console.warn(`[skill service fallback] ${method}`, error)
      return fallback()
    }
    throw error
  }
}

export const fetchSkills = async (): Promise<SkillSummary[]> => {
  const response = await invokeSkillCall<SkillSummary[]>('codeswitch/services.SkillService.ListSkills', () => fallbackGroupedSkills().installed[0]?.skills ?? [])
  return (response as SkillSummary[]) ?? []
}

export const fetchInstalledSkills = async (): Promise<SkillSummary[]> => {
  const response = await invokeSkillCall<SkillSummary[]>(
    'codeswitch/services.SkillService.ListInstalledSkills',
    () => fallbackGroupedSkills().installed.flatMap((group) => group.skills)
  )
  return (response as SkillSummary[]) ?? []
}

export const fetchAvailableSkills = async (): Promise<SkillSummary[]> => {
  const response = await invokeSkillCall<SkillSummary[]>(
    'codeswitch/services.SkillService.ListAvailableSkills',
    () => fallbackGroupedSkills().available.flatMap((group) => group.skills)
  )
  return (response as SkillSummary[]) ?? []
}

export const fetchGroupedSkills = async (): Promise<GroupedSkills> => {
  const response = await invokeSkillCall<GroupedSkills>(
    'codeswitch/services.SkillService.ListGroupedSkills',
    fallbackGroupedSkills
  )
  return (response as GroupedSkills) ?? { installed: [], available: [] }
}

export const fetchSkillLinkStatus = async (): Promise<SkillLinkStatus> => {
  const response = await invokeSkillCall<SkillLinkStatus>(
    'codeswitch/services.SkillService.GetSkillLinkStatus',
    fallbackLinkStatus
  )
  return response as SkillLinkStatus
}

export const ensureSkillLinks = async (): Promise<void> => {
  await Call.ByName('codeswitch/services.SkillService.EnsureSkillLinks')
}

export const fetchSkillDiagnostics = async (): Promise<SkillDiagnostics> => {
  const response = await invokeSkillCall<SkillDiagnostics>(
    'codeswitch/services.SkillService.GetSkillDiagnostics',
    fallbackDiagnostics
  )
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
  const response = await invokeSkillCall<SkillRepoConfig[]>(
    'codeswitch/services.SkillService.ListRepos',
    fallbackRepos
  )
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
