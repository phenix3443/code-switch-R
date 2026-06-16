<template>
  <button
    type="button"
    :class="[
      'skill-row',
      {
        selected,
        conflict,
        installed: skill.installed,
        recommended: !skill.installed
      }
    ]"
    @click="$emit('select', skill)"
  >
    <div class="skill-row-icon" :class="{ installed: skill.installed, conflict }">
      <span>{{ avatar }}</span>
    </div>

    <div class="skill-row-main">
      <div class="skill-row-head">
        <h3 class="skill-row-name">{{ skill.name }}</h3>
        <div class="skill-row-badges">
          <span v-if="skill.installed" class="row-badge enabled" :class="{ off: !skill.enabled }">
            {{ skill.enabled ? t('components.skill.badges.enabled') : t('components.skill.badges.disabled') }}
          </span>
          <span v-if="conflict" class="row-badge conflict">
            {{ t('components.skill.badges.conflict') }}
          </span>
        </div>
      </div>
      <p class="skill-row-desc">{{ skill.description || t('components.skill.list.noDescription') }}</p>
      <p class="skill-row-source">{{ sourceLabel }}</p>
    </div>

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
  </button>
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
  padding: 10px 12px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: inherit;
  text-align: left;
  transition: background 0.18s ease, border-color 0.18s ease, transform 0.18s ease;
}

.skill-row:hover {
  background: color-mix(in srgb, var(--mac-surface) 82%, transparent);
  border-color: color-mix(in srgb, var(--mac-border) 70%, transparent);
}

.skill-row.selected {
  background: color-mix(in srgb, var(--mac-accent) 14%, transparent);
  border-color: color-mix(in srgb, var(--mac-accent) 36%, var(--mac-border));
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
  background:
    radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.28), transparent 55%),
    linear-gradient(145deg, rgba(10, 132, 255, 0.28), rgba(16, 185, 129, 0.22));
  border: 1px solid color-mix(in srgb, var(--mac-border) 60%, transparent);
  color: var(--mac-text);
  font-weight: 700;
  letter-spacing: 0.04em;
}

.skill-row-icon.conflict {
  background:
    radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.18), transparent 55%),
    linear-gradient(145deg, rgba(239, 68, 68, 0.34), rgba(245, 158, 11, 0.26));
}

.skill-row-main {
  min-width: 0;
}

.skill-row-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.skill-row-name {
  margin: 0;
  font-size: 0.95rem;
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
  padding: 2px 7px;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.04em;
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
  font-size: 0.74rem;
  color: color-mix(in srgb, var(--mac-text-secondary) 82%, transparent);
}

.skill-row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.row-install-btn {
  min-width: 66px;
  height: 30px;
  padding: 0 12px;
  border: 1px solid color-mix(in srgb, var(--mac-accent) 24%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--mac-accent) 88%, white 8%);
  color: white;
  font-size: 0.78rem;
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
