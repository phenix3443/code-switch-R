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
            <span class="row-badge agent-chip" :class="{ active: skill.agents?.claude }" :title="skill.agents?.claude ? 'Claude 已同步' : 'Claude 未同步'"></span>
            <span class="row-badge agent-chip" :class="{ active: skill.agents?.codex }" :title="skill.agents?.codex ? 'Codex 已同步' : 'Codex 未同步'"></span>
          </template>
          <span v-if="conflict" class="row-badge conflict">
            {{ t('components.skill.badges.conflict') }}
          </span>
        </div>
      </div>
      <p class="skill-row-desc">{{ compactDescription }}</p>
      <div class="skill-row-meta">
        <span class="skill-row-source">{{ sourceLabel }}</span>
        <span v-if="skill.installed && skill.enabled" class="skill-row-state">{{ t('components.skill.badges.enabled') }}</span>
        <span v-else-if="skill.installed" class="skill-row-state muted">{{ t('components.skill.badges.disabled') }}</span>
      </div>
    </button>

    <div class="skill-row-actions">
      <button
        v-if="showInstallButton"
        type="button"
        class="ghost-icon sm"
        :disabled="disabled || loading"
        :title="loading ? t('components.skill.actions.loading') : t('components.skill.actions.more')"
        @click.stop="$emit('menu', skill)"
      >
        <span v-if="loading" class="skill-action-spinner" aria-hidden="true"></span>
        <svg v-else viewBox="0 0 24 24" aria-hidden="true">
          <path d="M10.12 2.23a1 1 0 011.76 0l.38.77a1 1 0 00.76.54l.85.12a1 1 0 01.56 1.7l-.62.6a1 1 0 00-.29.89l.15.85a1 1 0 01-1.45 1.05l-.76-.4a1 1 0 00-.94 0l-.76.4A1 1 0 018.6 8.92l.15-.85a1 1 0 00-.29-.89l-.62-.6a1 1 0 01.56-1.7l.85-.12a1 1 0 00.76-.54l.11-.22z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/>
          <path d="M13.54 14.22a1 1 0 011.24-.66l.82.24a1 1 0 00.92-.18l.67-.53a1 1 0 011.6.75l.07.86a1 1 0 00.48.8l.74.45a1 1 0 01.08 1.79l-.66.56a1 1 0 00-.34.87l.1.86a1 1 0 01-1.5.98l-.75-.43a1 1 0 00-.93-.07l-.81.29a1 1 0 01-1.3-1.24l.24-.82a1 1 0 00-.18-.92l-.53-.67a1 1 0 01.75-1.6l.86-.07a1 1 0 00.8-.48l.17-.27z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/>
          <circle cx="10.95" cy="11.05" r="2.15" fill="none" stroke="currentColor" stroke-width="1.3"/>
          <circle cx="16.85" cy="17.05" r="2" fill="none" stroke="currentColor" stroke-width="1.3"/>
        </svg>
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

const compactDescription = computed(() => {
  const raw = props.skill.description?.trim()
  if (!raw) return t('components.skill.list.noDescription')
  return raw.replace(/\s+/g, ' ')
})
</script>

<style scoped>
.skill-row {
  width: 100%;
  display: grid;
  grid-template-columns: 26px minmax(0, 1fr) 24px;
  align-items: center;
  gap: 8px;
  padding: 6px 12px 6px 8px;
  border: 0;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 52%, transparent);
  background: transparent;
  color: inherit;
  text-align: left;
  transition: background 0.18s ease;
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
  background: color-mix(in srgb, var(--mac-surface) 24%, transparent);
}

.skill-row.selected {
  background: color-mix(in srgb, #ffffff 2%, transparent);
  box-shadow: inset 1px 0 0 color-mix(in srgb, var(--mac-accent) 78%, transparent);
}

.skill-row.conflict {
  box-shadow: inset 1px 0 0 color-mix(in srgb, #ef4444 72%, transparent);
}

.skill-row-icon {
  width: 26px;
  height: 26px;
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--mac-accent) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--mac-border) 42%, transparent);
  color: var(--mac-text);
  font-size: 0.76rem;
  font-weight: 600;
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
  gap: 4px;
  min-width: 0;
}

.skill-row-name {
  margin: 0;
  font-size: 0.78rem;
  font-weight: 600;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-row-badges {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.row-badge {
  padding: 0;
  border-radius: 0;
  font-size: 0.64rem;
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
  background: transparent;
}

.row-badge.agent-chip {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--mac-border) 70%, transparent);
  color: transparent;
  border: 1px solid color-mix(in srgb, var(--mac-surface) 90%, transparent);
}

.row-badge.agent-chip.active {
  background: #22c55e;
}

.skill-row-desc,
.skill-row-source,
.skill-row-state {
  margin: 1px 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-row-desc {
  font-size: 0.72rem;
  color: var(--mac-text-secondary);
  line-height: 1.25;
}

.skill-row-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.skill-row-source {
  font-size: 0.64rem;
  color: var(--mac-text-secondary);
}

.skill-row-state {
  flex-shrink: 0;
  font-size: 0.62rem;
  color: #22c55e;
}

.skill-row-state.muted {
  color: var(--mac-text-secondary);
}

.skill-row-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  padding-top: 0;
}

.ghost-icon.sm {
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 2px;
  background: transparent;
  color: var(--mac-text-secondary);
}

.ghost-icon.sm:hover {
  background: color-mix(in srgb, var(--mac-surface) 36%, transparent);
  color: var(--mac-text);
}

.ghost-icon.sm svg {
  width: 14px;
  height: 14px;
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
