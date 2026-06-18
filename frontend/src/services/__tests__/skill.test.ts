import { beforeEach, describe, expect, it, vi } from 'vitest'

const byName = vi.fn()

vi.mock('@wailsio/runtime', () => ({
  Call: {
    ByName: (...args: unknown[]) => byName(...args)
  }
}))

describe('skill service api wrappers', async () => {
  const skillService = await import('../skill')

  beforeEach(() => {
    byName.mockReset()
  })

  it('fetchGroupedSkills falls back to empty groups', async () => {
    byName.mockResolvedValueOnce(undefined)

    await expect(skillService.fetchGroupedSkills()).resolves.toEqual({
      installed: [],
      available: []
    })
    expect(byName).toHaveBeenCalledWith('codeswitch/services.SkillService.ListGroupedSkills')
  })

  it('fetchGroupedSkills returns response payload when service succeeds', async () => {
    const grouped = {
      installed: [
        {
          group_key: 'local',
          group_label: 'LOCAL',
          skills: []
        }
      ],
      available: []
    }
    byName.mockResolvedValueOnce(grouped)

    await expect(skillService.fetchGroupedSkills()).resolves.toEqual(grouped)
  })

  it('installSkill forwards repo metadata', async () => {
    byName.mockResolvedValueOnce(undefined)

    await skillService.installSkill('demo-skill', 'example', 'skills', 'main')

    expect(byName).toHaveBeenCalledWith(
      'codeswitch/services.SkillService.InstallSkill',
      'demo-skill',
      'example',
      'skills',
      'main',
      []
    )
  })

  it('fetchSkillDiagnostics returns raw diagnostics payload', async () => {
    const diagnostics = {
      user_skills_path: '/tmp/home/.agents/skills',
      backup_root: '/tmp/home/.code-switch/backups/skills',
      claude_link_path: '/tmp/home/.claude/skills',
      codex_link_path: '/tmp/home/.codex/skills',
      backups: [{ platform: 'claude', path: '/tmp/backup' }],
      migrations: [{ platform: 'claude', directory: 'demo-skill', status: 'migrated' }]
    }
    byName.mockResolvedValueOnce(diagnostics)

    await expect(skillService.fetchSkillDiagnostics()).resolves.toEqual(diagnostics)
    expect(byName).toHaveBeenCalledWith('codeswitch/services.SkillService.GetSkillDiagnostics')
  })

  it('toggleSkill forwards directory and enabled state', async () => {
    byName.mockResolvedValueOnce(undefined)

    await skillService.toggleSkill('demo-skill', false)

    expect(byName).toHaveBeenCalledWith('codeswitch/services.SkillService.ToggleSkill', 'demo-skill', false)
  })
})
