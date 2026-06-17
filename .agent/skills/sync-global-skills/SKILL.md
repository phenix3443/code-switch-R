---
name: sync-global-skills
description: Synchronize the user's public global skills for Codex and Claude Code via `npx skills`. Use this whenever the user wants to reinstall, reconcile, standardize, migrate, or repair shared skills across Codex and Claude Code, especially when the goal is to replace manual copies or ad-hoc symlinks with `skills.sh`-managed global installs. Also use when the user wants to remove deprecated shared skills, preserve project-private skills as exceptions, or regenerate the public skill set from a known manifest/script.
---

# Sync Global Skills

Use this skill to keep the user's shared public skills in a consistent state across Codex and Claude Code.

This workflow is intentionally opinionated:

- Public skills are installed via `npx skills add ... -g`
- `Claude Code` gets explicit symlinks under `~/.claude/skills`
- `Codex` may use `skills` global registration instead of recreating entries under `~/.codex/skills`
- Skill groups that cannot be reproduced cleanly from `skills.sh` are treated as exceptions
- Project-private skills remain outside `npx skills` management unless the user explicitly wants to package them

## When To Apply

Apply this skill when the user asks to:

- sync Codex and Claude skills
- reinstall skills globally
- convert skills to `npx skills` management
- replace manual skill directories or symlinks
- repair drift between Codex and Claude Code skill installs
- remove deprecated shared skills

Do not use this skill for:

- editing the content of an individual skill
- project-private skill authoring, except to preserve documented exceptions

## Known Exceptions

Treat these categories as explicit exceptions unless the user says otherwise:

- Private/local skills: preserve them outside `npx skills`
- Deprecated shared skills: remove instead of reinstalling
- Unresolved skill families: leave untouched until they can be mapped cleanly to `skills.sh`

## Workflow

1. Inspect the current state.
2. Classify skills as public/global, private/local, deprecated, or unresolved.
3. Remove deprecated shared skills the user no longer wants.
4. Reinstall the public skill set via the bundled script.
5. Verify the resulting global skill list.
6. Report any agent-specific behavior differences, especially Codex vs Claude Code.

## Classification Rules

Classify each skill before reinstalling:

- `Public/global`: can be reinstalled from `skills.sh` with a known package and skill name
- `Private/local`: only exists in local folders, project directories, private repos, or other user-managed locations
- `Deprecated`: explicitly removed by the user or superseded by a newer workflow
- `Unresolved`: exists locally but cannot yet be reproduced 1:1 from `skills.sh`

Only reinstall the `public/global` bucket automatically.

## Handling Private Skills

Do not hardcode private repository names or paths into this skill.

Instead:

1. Detect private skills from local context.
2. Preserve them where the user manages them today unless the user explicitly asks to move them.
3. Exclude them from the `npx skills` reinstall script.
4. Mention them in the final report as preserved exceptions.

If the user wants a private skill to become portable, treat that as a separate packaging task.

## Handling Unresolved Skill Families

Some skill families do not map cleanly to the registry view in `skills.sh`.

Typical signs:

- the registry exposes only a root skill, but the local install contains many split skills
- a local skill family appears to come from a custom plugin or handwritten bundle
- the package exists, but its install shape does not match what is installed locally

When this happens:

1. Do not delete or overwrite that family during the public reinstall pass.
2. Mark it as `unresolved`.
3. Tell the user it could not be reproduced 1:1 from `skills.sh`.
4. Offer the user the next decision:
   - keep it as-is
   - migrate it manually
   - package it properly for future `npx skills` use

This is the correct handling for ecosystems such as `gstack` when the local split-skill layout cannot be recreated directly from the registry.

## Inspect Current State

Use these checks before making changes:

```bash
npx -y skills ls -g --json
find ~/.claude/skills -mindepth 1 -maxdepth 1 | sort
find ~/.codex/skills -mindepth 1 -maxdepth 1 | sort
```

When checking Codex results, do not assume missing entries under `~/.codex/skills` means install failure. `skills` may register Codex-compatible global skills without recreating local directory entries.

## Reinstall Public Skills

Run the bundled script:

```bash
bash scripts/reinstall-global-skills.sh
```

The script:

- reinstalls the approved public skill set with `npx skills`
- does not modify unresolved skill families
- does not modify private/local exceptions

## Verify

After reinstall, verify with:

```bash
npx -y skills ls -g --json | jq -r '.[] | .name' | sort
```

Expected public skills include:

- `anti-slop`
- `brainstorming`
- `caveman`
- `cli-creator`
- `diagnose`
- `dispatching-parallel-agents`
- `executing-plans`
- `find-skills`
- `finishing-a-development-branch`
- `frontend-design`
- `git-guardrails-claude-code`
- `grill-me`
- `grill-with-docs`
- `handoff`
- `improve-codebase-architecture`
- `notebooklm`
- `pi-planning-with-files`
- `planning-with-files`
- `prototype`
- `receiving-code-review`
- `requesting-code-review`
- `setup-matt-pocock-skills`
- `setup-pre-commit`
- `skill-creator`
- `skills-discovery`
- `subagent-driven-development`
- `systematic-debugging`
- `tdd`
- `test-driven-development`
- `to-issues`
- `to-prd`
- `triage`
- `ui-ux-pro-max`
- `using-git-worktrees`
- `using-superpowers`
- `verification-before-completion`
- `write-a-skill`
- `writing-plans`
- `writing-skills`
- `zoom-out`

Also verify exceptions remain correct:

- private/local skills were preserved
- unresolved families remain untouched
- deprecated shared skills are absent

## Reporting

When closing out:

- state that the public skill set was reinstalled via `npx skills`
- mention that `Claude Code` uses explicit symlinks while `Codex` may use global registration
- call out any preserved private skills
- call out any unresolved families, such as `gstack`-like structures that could not be reproduced 1:1 from `skills.sh`
