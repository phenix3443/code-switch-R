#!/usr/bin/env bash

set -euo pipefail

# Reinstall public global skills via skills.sh for Codex and Claude Code.
# This script intentionally does not manage:
# - gstack-related skills
# - project-private skills such as soft-copyright-application

SKILLS_CMD=(npx -y skills)
AGENTS=(codex claude-code)

cleanup_removed_skills() {
  rm -rf "$HOME/.agents/skills/claude-sync"
  rm -f "$HOME/.claude/skills/claude-sync"
  rm -f "$HOME/.codex/skills/claude-sync"
}

install_repo_skills() {
  local repo="$1"
  shift
  "${SKILLS_CMD[@]}" add "$repo" -g -a "${AGENTS[@]}" --skill "$@" -y
}

cleanup_removed_skills

install_repo_skills obra/superpowers \
  brainstorming \
  dispatching-parallel-agents \
  executing-plans \
  finishing-a-development-branch \
  receiving-code-review \
  requesting-code-review \
  subagent-driven-development \
  systematic-debugging \
  test-driven-development \
  using-git-worktrees \
  using-superpowers \
  verification-before-completion \
  writing-plans \
  writing-skills

install_repo_skills mattpocock/skills \
  caveman \
  diagnose \
  grill-me \
  grill-with-docs \
  improve-codebase-architecture \
  prototype \
  setup-matt-pocock-skills \
  tdd \
  to-issues \
  to-prd \
  triage \
  zoom-out \
  handoff \
  write-a-skill \
  git-guardrails-claude-code \
  setup-pre-commit

install_repo_skills othmanadi/planning-with-files \
  planning-with-files \
  pi-planning-with-files

install_repo_skills anthropics/skills \
  frontend-design \
  skill-creator

install_repo_skills openai/skills \
  cli-creator

install_repo_skills kamalnrf/claude-plugins \
  skills-discovery

install_repo_skills pleaseprompto/notebooklm-skill \
  notebooklm

install_repo_skills nextlevelbuilder/ui-ux-pro-max-skill \
  ui-ux-pro-max

install_repo_skills vercel-labs/skills \
  find-skills

install_repo_skills rand/cc-polymath \
  anti-slop

echo "Reinstall complete."
