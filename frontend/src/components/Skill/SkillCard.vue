<template>
  <div
    :class="[
      'skill-row',
      {
        selected,
        conflict,
        installed: skill.installed,
        recommended: !skill.installed
      }
    ]"
  >
    <div class="skill-row-icon" :class="{ installed: skill.installed, conflict }">
      <span>{{ avatar }}</span>
    </div>

    <button type="button" class="skill-row-main hitbox-reset" @click="$emit('select', skill)">
      <div class="skill-row-head">
        <h3 class="skill-row-name">{{ skill.name }}</h3>
        <div class="skill-row-badges">
          <template v-if="skill.installed">
            <span class="row-badge agent-chip" :class="{ active: skill.agents?.claude }">C</span>
            <span class="row-badge agent-chip" :class="{ active: skill.agents?.codex }">X</span>
          </template>
          <span v-if="conflict" class="row-badge conflict">
            {{ t('components.skill.badges.conflict') }}
          </span>
        </div>
      </div>
      <p class="skill-row-desc">{{ skill.description || t('components.skill.list.noDescription') }}</p>
      <p class="skill-row-source">{{ sourceLabel }}</p>
    </button>

    <div class="skill-row-actions">
      <button
        v-if="showInstallButton"
        type="button"
        class="row-install-btn"
        :disabled="disabled || loading"
        @click.stop="$emit('install', skill)"
      >
        <span v-if="!loading">{{ t('components.skill.actions.install') }}</span>
        <span v-else class="skill-action-spinner" aria-hidden="true"></span>
      </button>

      <button
        v-else
        type="button"
        class="ghost-icon sm"
        :title="t('components.skill.actions.more')"
        @click.stop="$emit('menu', skill)"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 6a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 9a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 9a1.5 1.5 0 110-3 1.5 1.5 0 010 3z" fill="currentColor" />
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SkillSummary } from '../../services/skill'

const props = withDefaults(defineProps<{
  skill: SkillSummary
  selected?: boolean
  conflict?: boolean
  loading?: boolean
  disabled?: boolean
  showInstallButton?: boolean
}>(), {
  selected: false,
  conflict: false,
  loading: false,
  disabled: false,
  showInstallButton: false
})

defineEmits<{
  select: [skill: SkillSummary]
  install: [skill: SkillSummary]
  menu: [skill: SkillSummary]
}>()

const { t } = useI18n()

const avatar = computed(() => {
  const source = props.skill.name?.trim() || props.skill.directory?.trim() || '?'
  return source.charAt(0).toUpperCase()
})

const sourceLabel = computed(() => {
  if (props.skill.repo_owner && props.skill.repo_name) {
    return `${props.skill.repo_owner}/${props.skill.repo_name}`
  }
  return props.skill.source_group_label || t('components.skill.groups.unknownSource')
})
</script>

<style scoped>
.skill-row {
  width: 100%;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 12px 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 34%, transparent);
  border-radius: 12px;
  background: color-mix(in srgb, var(--mac-surface) 82%, transparent);
  color: inherit;
  text-align: left;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.hitbox-reset {
  border: 0;
  background: transparent;
  color: inherit;
  padding: 0;
  margin: 0;
  font: inherit;
  text-align: left;
}

.skill-row:hover {
  background: color-mix(in srgb, var(--mac-surface) 92%, transparent);
  border-color: color-mix(in srgb, var(--mac-border) 82%, transparent);
  box-shadow: 0 8px 18px rgba(0, 0, 0, 0.08);
}

.skill-row.selected {
  background: color-mix(in srgb, var(--mac-accent) 10%, var(--mac-surface));
  border-color: color-mix(in srgb, var(--mac-accent) 46%, var(--mac-border));
  box-shadow: 0 10px 22px rgba(10, 132, 255, 0.08);
}

.skill-row.conflict {
  border-color: color-mix(in srgb, #ef4444 30%, var(--mac-border));
}

.skill-row-icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--mac-accent) 12%, var(--mac-surface-strong));
  border: 1px solid color-mix(in srgb, var(--mac-border) 56%, transparent);
  color: var(--mac-text);
  font-weight: 700;
}

.skill-row-icon.conflict {
  background: color-mix(in srgb, #ef4444 14%, var(--mac-surface-strong));
}

.skill-row-main {
  min-width: 0;
  width: 100%;
}

.skill-row-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.skill-row-name {
  margin: 0;
  font-size: 0.94rem;
  font-weight: 700;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-row-badges {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.row-badge {
  padding: 3px 7px;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 600;
}

.row-badge.enabled {
  color: #22c55e;
  background: color-mix(in srgb, #22c55e 18%, transparent);
}

.row-badge.enabled.off {
  color: #f59e0b;
  background: color-mix(in srgb, #f59e0b 18%, transparent);
}

.row-badge.conflict {
  color: #ef4444;
  background: color-mix(in srgb, #ef4444 18%, transparent);
}

.row-badge.agent-chip {
  color: var(--mac-text-secondary);
  background: color-mix(in srgb, var(--mac-border) 30%, transparent);
  font-size: 0.64rem;
  padding: 2px 5px;
}

.row-badge.agent-chip.active {
  color: var(--mac-accent);
  background: color-mix(in srgb, var(--mac-accent) 16%, transparent);
}

.skill-row-desc,
.skill-row-source {
  margin: 2px 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-row-desc {
  font-size: 0.82rem;
  color: var(--mac-text-secondary);
}

.skill-row-source {
  font-size: 0.72rem;
  color: var(--mac-text-secondary);
}

.skill-row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.row-install-btn {
  min-width: 72px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid color-mix(in srgb, var(--mac-accent) 20%, transparent);
  border-radius: 10px;
  background: color-mix(in srgb, var(--mac-accent) 84%, white 10%);
  color: white;
  font-size: 0.76rem;
  font-weight: 700;
}

.row-install-btn:disabled {
  opacity: 0.55;
}

.ghost-icon.sm {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--mac-text-secondary);
}

.ghost-icon.sm:hover {
  background: color-mix(in srgb, var(--mac-surface) 82%, transparent);
  color: var(--mac-text);
}

.ghost-icon.sm svg {
  width: 16px;
  height: 16px;
}

.skill-action-spinner {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid currentColor;
  border-top-color: transparent;
  animation: skill-spin 0.8s linear infinite;
  display: inline-block;
}

@keyframes skill-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
