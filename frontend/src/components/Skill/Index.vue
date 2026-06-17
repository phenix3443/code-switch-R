<template>
  <div class="main-shell">
    <div class="global-actions">
      <p class="global-eyebrow">{{ t('components.skill.hero.eyebrow') }}</p>
      <button class="ghost-icon" :title="t('components.skill.actions.back')" @click="goHome">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M15 18l-6-6 6-6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
      <button class="ghost-icon" :title="t('components.skill.actions.refresh')" :disabled="refreshing" @click="refresh">
        <svg viewBox="0 0 24 24" aria-hidden="true" :class="{ spin: refreshing }">
          <path d="M20.5 8a8.5 8.5 0 10-2.38 7.41" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M20.5 4v4h-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>
      <div class="menu-anchor">
        <button class="ghost-icon" :title="t('components.skill.actions.more')" @click="toggleOverflowMenu">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M12 6a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 9a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 9a1.5 1.5 0 110-3 1.5 1.5 0 010 3z" fill="currentColor" />
          </svg>
        </button>
        <div v-if="overflowMenuOpen" class="menu-popover global-menu-popover">
          <button type="button" @click="openRepoModal">
            {{ t('components.skill.repos.open') }}
          </button>
          <button type="button" @click="handleOpenFolder">
            {{ t('components.skill.actions.openFolder') }}
          </button>
          <button type="button" @click="handleRepairLinks" :disabled="repairing">
            {{ repairing ? t('components.skill.actions.repairing') : t('components.skill.actions.repairLinks') }}
          </button>
          <button type="button" @click="openBackupModal">
            {{ t('components.skill.actions.viewBackups') }}
          </button>
          <button type="button" @click="conflictModalOpen = true" :disabled="!conflictRecords.length">
            {{ t('components.skill.actions.viewConflicts') }}
          </button>
        </div>
      </div>
    </div>

    <div class="skill-workspace">
      <div v-if="skillsError" class="skill-banner error">{{ skillsError }}</div>
      <div v-else-if="notice" class="skill-banner">{{ notice }}</div>

      <section class="skill-layout">
        <aside class="skill-sidebar">
          <header class="sidebar-header">
            <div class="search-shell">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M11 18a7 7 0 100-14 7 7 0 000 14zm8 3l-4.35-4.35" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('components.skill.search.placeholder')"
              />
              <button v-if="searchQuery" class="ghost-inline" type="button" @click="searchQuery = ''">
                {{ t('components.skill.search.clear') }}
              </button>
              <button class="ghost-inline filter" type="button" @click="cycleFilterMode">
                {{ filterLabel }}
              </button>
            </div>
          </header>

          <div class="sidebar-body">
            <div v-if="loading" class="skill-empty">{{ t('components.skill.list.loading') }}</div>
            <template v-else>
              <section class="skill-group-panel">
                <button class="group-toggle" type="button" @click="collapsed.installed = !collapsed.installed">
                  <span>{{ t('components.skill.groups.installed') }}</span>
                  <span class="group-badge">{{ filteredInstalledCount }}</span>
                </button>

                <div v-if="!collapsed.installed" class="group-body">
                  <div v-if="!filteredInstalledGroups.length" class="skill-empty compact">
                    {{ t('components.skill.list.noInstalled') }}
                  </div>

                  <section v-for="group in filteredInstalledGroups" :key="group.group_key" class="subgroup-panel">
                    <button class="subgroup-toggle" type="button" @click="toggleSubgroup(group.group_key)">
                      <div class="subgroup-title">
                        <span>{{ group.group_label }}</span>
                        <span class="group-badge">{{ group.skills.length }}</span>
                      </div>
                      <span class="subgroup-caret">{{ subgroupCollapsed[group.group_key] ? '+' : '−' }}</span>
                    </button>

                    <div v-if="!subgroupCollapsed[group.group_key]" class="subgroup-list">
                      <div v-for="skill in group.skills" :key="skillIdentity(skill)" class="skill-row-wrap">
                        <SkillCard
                          :skill="skill"
                          :selected="selectedSkillKey === skillIdentity(skill)"
                          :conflict="isConflictSkill(skill)"
                          @select="selectSkill"
                          @menu="toggleSkillMenu"
                        />
                        <div v-if="openSkillMenuKey === skillIdentity(skill)" class="row-menu">
                          <button type="button" @click="handleToggle(skill, !skill.enabled)">
                            {{ skill.enabled ? t('components.skill.actions.disable') : t('components.skill.actions.enable') }}
                          </button>
                          <button type="button" @click="openInstalledFolder(skill)">
                            {{ t('components.skill.actions.openSkillFolder') }}
                          </button>
                          <button type="button" @click="openSkillRepo(skill)" :disabled="!skill.readme_url">
                            {{ t('components.skill.actions.openRepo') }}
                          </button>
                          <button type="button" class="danger" @click="handleUninstall(skill)">
                            {{ t('components.skill.actions.uninstall') }}
                          </button>
                        </div>
                      </div>
                    </div>
                  </section>
                </div>
              </section>

              <section class="skill-group-panel">
                <button class="group-toggle" type="button" @click="collapsed.recommended = !collapsed.recommended">
                  <span>{{ t('components.skill.groups.recommended') }}</span>
                  <span class="group-badge">{{ filteredAvailableCount }}</span>
                </button>

                <div v-if="!collapsed.recommended" class="group-body">
                  <div v-if="!filteredAvailableGroups.length" class="skill-empty compact">
                    {{ t('components.skill.list.noRecommended') }}
                  </div>

                  <section v-for="group in filteredAvailableGroups" :key="group.group_key" class="subgroup-panel recommended">
                    <div class="subgroup-label">{{ group.group_label }}</div>
                    <div class="subgroup-list">
                      <SkillCard
                        v-for="skill in group.skills"
                        :key="skillIdentity(skill)"
                        :skill="skill"
                        :selected="selectedSkillKey === skillIdentity(skill)"
                        :loading="isInstallingSkill(skill)"
                        :disabled="!canInstallSkill(skill)"
                        :show-install-button="true"
                        @select="selectSkill"
                        @install="handleInstall"
                      />
                    </div>
                  </section>
                </div>
              </section>
            </template>
          </div>
        </aside>

        <section class="skill-detail">
          <div v-if="!selectedSkill" class="detail-empty">
            <h2>{{ t('components.skill.detail.emptyTitle') }}</h2>
            <p>{{ t('components.skill.detail.emptyDesc') }}</p>
          </div>

          <template v-else>
            <header class="detail-header">
              <div class="detail-header-main">
                <div class="detail-icon" :class="{ conflict: isConflictSkill(selectedSkill) }">
                  {{ detailAvatar }}
                </div>
                <div class="detail-copy">
                  <p class="detail-publisher">{{ selectedSourceLabel }}</p>
                  <h2>{{ selectedSkill.name }}</h2>
                  <p class="detail-tagline">{{ selectedSkill.description || t('components.skill.list.noDescription') }}</p>
                  <div class="detail-subline">
                    <span>{{ selectedSkill.directory }}</span>
                    <span>{{ selectedSkill.repo_branch || 'main' }}</span>
                    <span>{{ selectedSkill.installed ? (selectedSkill.enabled ? t('components.skill.badges.enabled') : t('components.skill.badges.disabled')) : t('components.skill.groups.recommended') }}</span>
                  </div>
                </div>
              </div>

              <div class="detail-actions">
                <template v-if="selectedSkill.installed">
                  <div class="agent-chips">
                    <span class="agent-chip" :class="{ active: selectedSkill.agents?.claude }">C</span>
                    <span class="agent-chip" :class="{ active: selectedSkill.agents?.codex }">X</span>
                  </div>
                  <button
                    class="btn-primary"
                    :disabled="togglingSkill === selectedSkill.directory || isConflictSkill(selectedSkill)"
                    @click="handleToggle(selectedSkill, !selectedSkill.enabled)"
                  >
                    {{ selectedSkill.enabled ? t('components.skill.actions.disableDropdown') : t('components.skill.actions.enableDropdown') }}
                  </button>
                  <div class="split-btn-group">
                    <button
                      class="btn-secondary split-main"
                      :disabled="processingSkill === uninstallProcessingKey(selectedSkill)"
                      @click="handleUninstall(selectedSkill)"
                    >
                      {{ t('components.skill.actions.uninstall') }}
                    </button>
                    <button
                      class="btn-secondary split-caret"
                      :disabled="processingSkill === uninstallProcessingKey(selectedSkill)"
                      @click.stop="uninstallDropdownOpen = !uninstallDropdownOpen"
                      aria-label="uninstall options"
                    >▾</button>
                    <div v-if="uninstallDropdownOpen" class="split-dropdown">
                      <button type="button" @click="handleUninstallAgent(selectedSkill, 'claude')">{{ t('components.skill.actions.uninstallClaude') }}</button>
                      <button type="button" @click="handleUninstallAgent(selectedSkill, 'codex')">{{ t('components.skill.actions.uninstallCodex') }}</button>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div class="split-btn-group">
                    <button
                      class="btn-primary split-main"
                      :disabled="isInstallingSkill(selectedSkill) || !canInstallSkill(selectedSkill)"
                      @click="handleInstall(selectedSkill, [])"
                    >
                      {{ isInstallingSkill(selectedSkill) ? t('components.skill.install.installing') : t('components.skill.actions.installAll') }}
                    </button>
                    <button
                      class="btn-primary split-caret"
                      :disabled="isInstallingSkill(selectedSkill) || !canInstallSkill(selectedSkill)"
                      @click.stop="installDropdownOpen = !installDropdownOpen"
                      aria-label="install options"
                    >▾</button>
                    <div v-if="installDropdownOpen" class="split-dropdown">
                      <button type="button" @click="handleInstall(selectedSkill, ['claude'])">{{ t('components.skill.actions.installClaude') }}</button>
                      <button type="button" @click="handleInstall(selectedSkill, ['codex'])">{{ t('components.skill.actions.installCodex') }}</button>
                    </div>
                  </div>
                </template>
              </div>
            </header>

            <nav class="detail-tabs">
              <button :class="{ active: activeTab === 'overview' }" @click="activeTab = 'overview'">
                {{ t('components.skill.tabs.overview') }}
              </button>
              <button :class="{ active: activeTab === 'content' }" @click="activeTab = 'content'">
                {{ t('components.skill.tabs.content') }}
              </button>
              <button :class="{ active: activeTab === 'status' }" @click="activeTab = 'status'">
                {{ t('components.skill.tabs.status') }}
              </button>
            </nav>

            <div v-if="activeTab === 'overview'" class="detail-layout-vscode">
              <section class="detail-content-pane">
                <section class="detail-block install">
                  <div class="detail-block-label">INSTALLATION</div>
                  <div class="install-command-row">
                    <code class="install-command">{{ installCommand || selectedSkill.directory }}</code>
                    <button
                      class="tool-icon compact"
                      type="button"
                      :data-tooltip="copyTooltip"
                      :disabled="!installCommand"
                      @click="copyInstallCommand"
                    >
                      <svg viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M9 9.75A2.25 2.25 0 0111.25 7.5h6A2.25 2.25 0 0119.5 9.75v6A2.25 2.25 0 0117.25 18h-6A2.25 2.25 0 019 15.75v-6z" fill="none" stroke="currentColor" stroke-width="1.5" />
                        <path d="M5.25 15V8.25A2.25 2.25 0 017.5 6h6.75" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                      </svg>
                    </button>
                  </div>
                </section>

                <section class="detail-block">
                  <div class="detail-block-label">SKILL.MD</div>
                  <pre class="detail-prose">{{ selectedOverview }}</pre>
                </section>
              </section>

              <aside class="detail-sidebar-vscode">
                <section class="side-panel">
                  <h3>Installation</h3>
                  <dl class="side-facts">
                    <div>
                      <dt>{{ t('components.skill.info.directory') }}</dt>
                      <dd>{{ selectedSkill.directory }}</dd>
                    </div>
                    <div>
                      <dt>{{ t('components.skill.info.branch') }}</dt>
                      <dd>{{ selectedSkill.repo_branch || 'main' }}</dd>
                    </div>
                    <div>
                      <dt>Status</dt>
                      <dd>{{ selectedSkill.installed ? (selectedSkill.enabled ? t('components.skill.badges.enabled') : t('components.skill.badges.disabled')) : t('components.skill.groups.recommended') }}</dd>
                    </div>
                  </dl>
                </section>

                <section class="side-panel">
                  <h3>Repository</h3>
                  <p class="side-panel-value">{{ selectedSourceLabel }}</p>
                  <p v-if="selectedSkill.readme_url" class="side-panel-subtle">{{ selectedSkill.readme_url }}</p>
                </section>

                <section class="side-panel">
                  <h3>Resources</h3>
                  <div class="side-icon-links">
                    <button class="side-icon-link" type="button" @click="selectedSkill.installed ? openSelectedFolder() : handleOpenFolder()">
                      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>
                      {{ selectedSkill.installed ? t('components.skill.actions.openSkillFolder') : t('components.skill.actions.openFolder') }}
                    </button>
                    <button class="side-icon-link" type="button" :disabled="!selectedSkill.readme_url" @click="openSkillRepo(selectedSkill)">
                      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2C6.48 2 2 6.48 2 12c0 4.42 2.87 8.17 6.84 9.5.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34-.46-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.6.07-.6 1 .07 1.53 1.03 1.53 1.03.87 1.52 2.34 1.07 2.91.83.09-.65.35-1.09.63-1.34-2.22-.25-4.55-1.11-4.55-4.92 0-1.11.38-2 1.03-2.71-.1-.25-.45-1.29.1-2.64 0 0 .84-.27 2.75 1.02.79-.22 1.65-.33 2.5-.33.85 0 1.71.11 2.5.33 1.91-1.29 2.75-1.02 2.75-1.02.55 1.35.2 2.39.1 2.64.65.71 1.03 1.6 1.03 2.71 0 3.82-2.34 4.66-4.57 4.91.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0012 2z" fill="currentColor"/></svg>
                      {{ t('components.skill.actions.openRepo') }}
                    </button>
                    <button class="side-icon-link" type="button" @click="backupModalOpen = true">
                      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5.25 8.25h13.5M8.25 12h7.5m-9 3.75h10.5A2.25 2.25 0 0019.5 13.5v-6A2.25 2.25 0 0017.25 5.25H6.75A2.25 2.25 0 004.5 7.5v6a2.25 2.25 0 002.25 2.25z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                      {{ t('components.skill.actions.viewBackups') }}
                    </button>
                  </div>
                </section>
              </aside>
            </div>

            <div v-else-if="activeTab === 'content'" class="detail-panel">
              <div v-if="selectedSkill.installed" class="editor-shell">
                <div class="editor-toolbar">
                  <span>{{ t('components.skill.tabs.contentHelp') }}</span>
                  <button class="btn-secondary" :disabled="!contentDirty || savingContent" @click="saveSelectedContent">
                    {{ savingContent ? t('common.saving') : t('common.save') }}
                  </button>
                </div>
                <textarea v-model="skillContentDraft" class="skill-editor" spellcheck="false"></textarea>
              </div>
              <div v-else class="detail-placeholder">
                {{ t('components.skill.detail.availableContentPlaceholder') }}
              </div>
            </div>

            <div v-else class="detail-panel">
              <div class="timeline-block">
                <h3>{{ t('components.skill.summary.migrations') }}</h3>
                <div v-if="relatedMigrationRecords.length" class="timeline-list">
                  <article v-for="record in relatedMigrationRecords" :key="recordKey(record)" class="timeline-item">
                    <div class="timeline-item-head">
                      <strong>{{ record.directory || record.platform || t('components.skill.summary.unknownRecord') }}</strong>
                      <span class="row-badge" :class="badgeClassForMigration(record.status)">{{ formatMigrationStatus(record.status) }}</span>
                    </div>
                    <p>{{ record.message || t('components.skill.summary.noMessage') }}</p>
                    <small>{{ formatDate(record.created_at) }}</small>
                  </article>
                </div>
                <div v-else class="detail-placeholder">
                  {{ t('components.skill.summary.noMigrations') }}
                </div>
              </div>
            </div>
          </template>
        </section>
      </section>
    </div>

    <BaseModal :open="repoModalOpen" :title="t('components.skill.repos.title')" @close="closeRepoModal">
      <div class="repo-modal-content">
        <p class="repo-modal-copy">{{ t('components.skill.repos.subtitle') }}</p>
        <form class="repo-form" @submit.prevent="submitRepo">
          <input v-model="repoForm.url" type="text" :placeholder="t('components.skill.repos.urlPlaceholder')" :disabled="repoBusy" />
          <div class="repo-form-row">
            <input v-model="repoForm.branch" type="text" :placeholder="t('components.skill.repos.branchPlaceholder')" :disabled="repoBusy" />
            <button class="btn-primary" type="submit" :disabled="repoBusy">
              {{ repoBusy ? t('components.skill.actions.loading') : t('components.skill.repos.addLabel') }}
            </button>
          </div>
        </form>
        <p v-if="repoError" class="skill-banner error compact">{{ repoError }}</p>
        <div class="repo-list">
          <p v-if="repoLoading" class="skill-empty compact">{{ t('components.skill.repos.loading') }}</p>
          <p v-else-if="!repoList.length" class="skill-empty compact">{{ t('components.skill.repos.empty') }}</p>
          <article v-for="repo in repoList" :key="repoKey(repo)" class="repo-item">
            <div>
              <strong>{{ repo.owner }}/{{ repo.name }}</strong>
              <p>{{ t('components.skill.repos.branchLabel', { branch: repo.branch }) }}</p>
            </div>
            <div class="repo-item-actions">
              <button class="btn-secondary" @click="openRepoGithub(repo)">
                {{ t('components.skill.actions.openRepo') }}
              </button>
              <button class="btn-secondary danger" :disabled="repoBusy" @click="removeRepoItem(repo)">
                {{ t('components.skill.repos.removeLabel') }}
              </button>
            </div>
          </article>
        </div>
      </div>
    </BaseModal>

    <BaseModal :open="backupModalOpen" :title="t('components.skill.backups.title')" @close="backupModalOpen = false">
      <div class="modal-list">
        <p class="repo-modal-copy">{{ t('components.skill.backups.subtitle') }}</p>
        <p v-if="!diagnostics.backups.length" class="skill-empty compact">{{ t('components.skill.backups.empty') }}</p>
        <article v-for="backup in diagnostics.backups" :key="backup.path" class="modal-list-item">
          <div>
            <strong>{{ backup.platform || 'unknown' }}</strong>
            <p>{{ backup.path }}</p>
            <small>{{ formatDate(backup.created_at) }}</small>
          </div>
          <button class="btn-secondary" @click="openBackup(backup.path || '')">
            {{ t('components.skill.backups.open') }}
          </button>
        </article>
      </div>
    </BaseModal>

    <BaseModal :open="conflictModalOpen" :title="t('components.skill.conflicts.title')" @close="conflictModalOpen = false">
      <div class="modal-list">
        <p class="repo-modal-copy">{{ t('components.skill.conflicts.subtitle') }}</p>
        <p v-if="!conflictRecords.length" class="skill-empty compact">{{ t('components.skill.conflicts.empty') }}</p>
        <article v-for="record in conflictRecords" :key="recordKey(record)" class="modal-list-item conflict">
          <div>
            <strong>{{ record.directory || t('components.skill.summary.unknownRecord') }}</strong>
            <p>{{ record.platform || 'unknown' }} · {{ formatMigrationStatus(record.status) }}</p>
            <small>{{ record.message || t('components.skill.summary.noMessage') }}</small>
          </div>
        </article>
      </div>
    </BaseModal>

    <BaseModal
      :open="Boolean(pendingUninstallSkill)"
      :title="t('components.skill.actions.uninstall')"
      variant="confirm"
      @close="closeUninstallModal"
    >
      <div class="modal-list">
        <p class="repo-modal-copy">
          {{ t('components.skill.actions.confirmUninstall', { name: pendingUninstallSkill?.name || '' }) }}
        </p>
        <div class="confirm-actions">
          <button class="btn-secondary" type="button" @click="closeUninstallModal">
            {{ t('common.cancel') }}
          </button>
          <button
            class="btn-secondary danger"
            type="button"
            :disabled="!pendingUninstallSkill || processingSkill === uninstallProcessingKey(pendingUninstallSkill)"
            @click="confirmUninstall"
          >
            {{ t('components.skill.actions.uninstall') }}
          </button>
        </div>
      </div>
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { Browser } from '@wailsio/runtime'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import BaseModal from '../common/BaseModal.vue'
import SkillCard from './SkillCard.vue'
import {
  addSkillRepo,
  ensureSkillLinks,
  fetchGroupedSkills,
  fetchSkillDiagnostics,
  fetchSkillLinkStatus,
  fetchSkillRepos,
  getSkillContent,
  installSkill,
  openInstalledSkillFolder,
  openPlatformSkillsLink,
  openSkillBackup,
  openUserSkillsFolder,
  removeSkillRepo,
  saveSkillContent,
  toggleSkill,
  uninstallSkill,
  type GroupedSkills,
  type MigrationRecord,
  type SkillDiagnostics,
  type SkillGroup,
  type SkillLinkStatus,
  type SkillRepoConfig,
  type SkillSummary
} from '../../services/skill'

type FilterMode = 'all' | 'enabled' | 'conflict'
type DetailTab = 'overview' | 'content' | 'status'

const router = useRouter()
const { t, locale } = useI18n()

const groupedSkills = ref<GroupedSkills>({ installed: [], available: [] })
const linkStatus = ref<SkillLinkStatus | null>(null)
const diagnostics = ref<SkillDiagnostics>({
  user_skills_path: '',
  backup_root: '',
  claude_link_path: '',
  codex_link_path: '',
  backups: [],
  migrations: []
})
const repoList = ref<SkillRepoConfig[]>([])
const loading = ref(false)
const repoLoading = ref(false)
const skillsError = ref('')
const repoError = ref('')
const notice = ref('')
const processingSkill = ref('')
const togglingSkill = ref('')
const repoBusy = ref(false)
const repairing = ref(false)
const activeTab = ref<DetailTab>('overview')
const selectedSkillKey = ref('')
const searchQuery = ref('')
const filterMode = ref<FilterMode>('all')
const overflowMenuOpen = ref(false)
const openSkillMenuKey = ref('')
const skillContentDraft = ref('')
const originalSkillContent = ref('')
const savingContent = ref(false)
const repoModalOpen = ref(false)
const backupModalOpen = ref(false)
const conflictModalOpen = ref(false)
const pendingUninstallSkill = ref<SkillSummary | null>(null)
const installDropdownOpen = ref(false)
const uninstallDropdownOpen = ref(false)

const collapsed = reactive({
  installed: false,
  recommended: false
})
const subgroupCollapsed = reactive<Record<string, boolean>>({})
const repoForm = reactive({ url: '', branch: 'main' })

const refreshing = computed(() => loading.value || repoLoading.value || repairing.value)
const installedGroups = computed(() => groupedSkills.value.installed ?? [])
const availableGroups = computed(() => groupedSkills.value.available ?? [])
const installedCount = computed(() => installedGroups.value.reduce((sum, group) => sum + group.skills.length, 0))
const availableCount = computed(() => availableGroups.value.reduce((sum, group) => sum + group.skills.length, 0))
const conflictDirectories = computed(() => {
  const directories = new Set<string>()
  for (const record of diagnostics.value.migrations ?? []) {
    if (record.status === 'conflict' && record.directory) {
      directories.add(record.directory.toLowerCase())
    }
  }
  return directories
})

const conflictRecords = computed(() =>
  (diagnostics.value.migrations ?? []).filter((record) => record.status === 'conflict')
)

const hasHealthyLinks = computed(() =>
  Boolean(linkStatus.value) &&
  linkStatus.value?.claude.status === 'linked' &&
  linkStatus.value?.codex.status === 'linked'
)

const allSkills = computed(() => [
  ...installedGroups.value.flatMap((group) => group.skills),
  ...availableGroups.value.flatMap((group) => group.skills)
])

const selectedSkill = computed(() => {
  return allSkills.value.find((skill) => skillIdentity(skill) === selectedSkillKey.value) ?? null
})

const detailAvatar = computed(() => {
  const source = selectedSkill.value?.name?.trim() || selectedSkill.value?.directory?.trim() || '?'
  return source.charAt(0).toUpperCase()
})

const selectedSourceLabel = computed(() => {
  if (!selectedSkill.value) return ''
  if (selectedSkill.value.repo_owner && selectedSkill.value.repo_name) {
    return `${selectedSkill.value.repo_owner}/${selectedSkill.value.repo_name}`
  }
  return selectedSkill.value.source_group_label || t('components.skill.groups.unknownSource')
})

const contentDirty = computed(() => skillContentDraft.value !== originalSkillContent.value)

const selectedOverview = computed(() => {
  if (!selectedSkill.value) return ''
  if (selectedSkill.value.installed && originalSkillContent.value) {
    return extractSkillBody(originalSkillContent.value) || selectedSkill.value.description || t('components.skill.list.noDescription')
  }
  return [
    selectedSkill.value.description || t('components.skill.list.noDescription'),
    '',
    `${t('components.skill.info.directory')}: ${selectedSkill.value.directory}`,
    `${t('components.skill.info.source')}: ${selectedSourceLabel.value}`,
    selectedSkill.value.readme_url ? `${t('components.skill.info.readme')}: ${selectedSkill.value.readme_url}` : ''
  ].filter(Boolean).join('\n')
})

const installCommand = computed(() => {
  if (!selectedSkill.value?.repo_owner || !selectedSkill.value?.repo_name) return ''
  return `npx skills add https://github.com/${selectedSkill.value.repo_owner}/${selectedSkill.value.repo_name} --skill ${selectedSkill.value.directory}`
})

const copyTooltip = computed(() =>
  notice.value === installCommand.value && installCommand.value ? 'Copied' : 'Copy install command'
)

const relatedMigrationRecords = computed(() => {
  if (!selectedSkill.value) return diagnostics.value.migrations.slice(0, 10)
  return diagnostics.value.migrations
    .filter((record) => record.directory === selectedSkill.value?.directory || !record.directory)
    .slice(0, 12)
})

const summaryHeadline = computed(() => {
  if (!linkStatus.value) return t('components.skill.summary.loading')
  if (!hasHealthyLinks.value) return t('components.skill.summary.linksNeedRepair')
  if (conflictRecords.value.length) return t('components.skill.summary.conflictDetected', { count: conflictRecords.value.length })
  return t('components.skill.summary.healthy')
})

const lastBackupLabel = computed(() => {
  const record = diagnostics.value.backups?.[0]
  return record ? formatDate(record.created_at) : t('components.skill.summary.none')
})

const lastMigrationLabel = computed(() => {
  const record = diagnostics.value.migrations?.[0]
  return record ? formatMigrationStatus(record.status) : t('components.skill.summary.none')
})

const userSyncStatus = computed(() => {
  if (!linkStatus.value) return 'neutral'
  if (!linkStatus.value.user_skills_exists) return 'neutral'
  if (conflictRecords.value.length) return 'danger'
  return 'linked'
})

const filteredInstalledGroups = computed(() => filterGroups(installedGroups.value, true))
const filteredAvailableGroups = computed(() => filterGroups(availableGroups.value, false))
const filteredInstalledCount = computed(() => filteredInstalledGroups.value.reduce((sum, group) => sum + group.skills.length, 0))
const filteredAvailableCount = computed(() => filteredAvailableGroups.value.reduce((sum, group) => sum + group.skills.length, 0))

const filterLabel = computed(() => t(`components.skill.search.filters.${filterMode.value}`))

const syncNodeClass = (status?: string) => {
  if (status === 'linked') return 'is-linked'
  if (status === 'missing' || status === 'danger') return 'is-danger'
  return 'is-neutral'
}

const syncTooltip = (label: string, path: string, status?: string) => {
  const state = status === 'linked'
    ? t('components.skill.status.linked')
    : status === 'missing' || status === 'danger'
      ? t('components.skill.status.missing')
      : t('components.skill.status.notDetected')
  return [label, state, path].filter(Boolean).join(' · ')
}

watch(selectedSkill, async (skill) => {
  activeTab.value = 'overview'
  openSkillMenuKey.value = ''
  if (!skill?.installed) {
    originalSkillContent.value = ''
    skillContentDraft.value = ''
    return
  }

  try {
    const content = await getSkillContent(skill.directory)
    originalSkillContent.value = content
    skillContentDraft.value = content
  } catch (error) {
    console.error('failed to load skill content', error)
    originalSkillContent.value = t('components.skill.actions.loadFailed')
    skillContentDraft.value = originalSkillContent.value
  }
}, { immediate: true })

const skillIdentity = (skill: SkillSummary) =>
  skill.key || `${skill.directory.toLowerCase()}:${(skill.repo_owner || 'local').toLowerCase()}:${(skill.repo_name || 'unknown').toLowerCase()}`

const installProcessingKey = (skill: SkillSummary) => `install:${skillIdentity(skill)}`
const uninstallProcessingKey = (skill: SkillSummary) => `uninstall:${skillIdentity(skill)}`

const isInstallingSkill = (skill: SkillSummary) => processingSkill.value === installProcessingKey(skill)
const canInstallSkill = (skill: SkillSummary) => Boolean(skill.repo_owner && skill.repo_name && hasHealthyLinks.value)

const filterGroups = (groups: SkillGroup[], installed: boolean) => {
  const query = searchQuery.value.trim().toLowerCase()

  return groups
    .map((group) => {
      const skills = group.skills.filter((skill) => {
        if (query) {
          const haystack = [
            skill.name,
            skill.directory,
            skill.description,
            skill.repo_owner,
            skill.repo_name,
            skill.source_group_label
          ]
            .filter(Boolean)
            .join(' ')
            .toLowerCase()
          if (!haystack.includes(query)) {
            return false
          }
        }

        if (filterMode.value === 'enabled') {
          return installed ? skill.enabled : false
        }
        if (filterMode.value === 'conflict') {
          return isConflictSkill(skill)
        }
        return true
      })

      return { ...group, skills }
    })
    .filter((group) => group.skills.length > 0)
}

const loadGroupedSkills = async () => {
  loading.value = true
  skillsError.value = ''
  try {
    groupedSkills.value = await fetchGroupedSkills()
    ensureSelection()
    ensureSubgroupState()
  } catch (error) {
    console.error('failed to load grouped skills', error)
    skillsError.value = t('components.skill.list.error')
  } finally {
    loading.value = false
    processingSkill.value = ''
  }
}

const loadStatus = async () => {
  try {
    const [status, skillDiagnostics] = await Promise.all([fetchSkillLinkStatus(), fetchSkillDiagnostics()])
    linkStatus.value = status
    diagnostics.value = skillDiagnostics
  } catch (error) {
    console.error('failed to load skill diagnostics', error)
  }
}

const loadRepos = async () => {
  repoLoading.value = true
  repoError.value = ''
  try {
    repoList.value = await fetchSkillRepos()
  } catch (error) {
    console.error('failed to load skill repos', error)
    repoError.value = t('components.skill.repos.loadError')
  } finally {
    repoLoading.value = false
  }
}

const refresh = () => {
  notice.value = ''
  void Promise.all([loadGroupedSkills(), loadStatus(), loadRepos()])
}

const ensureSelection = () => {
  const skills = allSkills.value
  if (!skills.length) {
    selectedSkillKey.value = ''
    return
  }
  if (!selectedSkillKey.value || !skills.some((skill) => skillIdentity(skill) === selectedSkillKey.value)) {
    selectedSkillKey.value = skillIdentity(skills[0])
  }
}

const ensureSubgroupState = () => {
  for (const group of installedGroups.value) {
    if (!(group.group_key in subgroupCollapsed)) {
      subgroupCollapsed[group.group_key] = false
    }
  }
}

const selectSkill = (skill: SkillSummary) => {
  selectedSkillKey.value = skillIdentity(skill)
}

const toggleSubgroup = (key: string) => {
  subgroupCollapsed[key] = !subgroupCollapsed[key]
}

const toggleOverflowMenu = () => {
  overflowMenuOpen.value = !overflowMenuOpen.value
}

const toggleSkillMenu = (skill: SkillSummary) => {
  const key = skillIdentity(skill)
  selectSkill(skill)
  openSkillMenuKey.value = openSkillMenuKey.value === key ? '' : key
}

const cycleFilterMode = () => {
  const modes: FilterMode[] = ['all', 'enabled', 'conflict']
  const nextIndex = (modes.indexOf(filterMode.value) + 1) % modes.length
  filterMode.value = modes[nextIndex]
}

const isConflictSkill = (skill: SkillSummary) => conflictDirectories.value.has(skill.directory.toLowerCase())

const handleOpenFolder = async () => {
  overflowMenuOpen.value = false
  try {
    await openUserSkillsFolder()
  } catch (error) {
    console.error('failed to open user skills folder', error)
  }
}

const openPlatformLink = async (platform: string) => {
  try {
    await openPlatformSkillsLink(platform)
  } catch (error) {
    console.error('failed to open platform skills link', error)
  }
}

const handleRepairLinks = async () => {
  overflowMenuOpen.value = false
  repairing.value = true
  notice.value = ''
  try {
    await ensureSkillLinks()
    await Promise.all([loadStatus(), loadGroupedSkills()])
    notice.value = t('components.skill.actions.repairSuccess')
  } catch (error) {
    console.error('failed to repair skill links', error)
    skillsError.value = t('components.skill.actions.repairError')
  } finally {
    repairing.value = false
  }
}

const handleInstall = async (skill: SkillSummary, agents: string[] = []) => {
  installDropdownOpen.value = false
  selectSkill(skill)
  if (!hasHealthyLinks.value) {
    skillsError.value = t('components.skill.install.linksRequired')
    activeTab.value = 'status'
    return
  }
  processingSkill.value = installProcessingKey(skill)
  try {
    await installSkill(skill.directory, skill.repo_owner ?? '', skill.repo_name ?? '', skill.repo_branch ?? '', agents)
    skillsError.value = ''
    notice.value = t('components.skill.install.success', { name: skill.name })
    await Promise.all([loadGroupedSkills(), loadStatus()])
  } catch (error) {
    console.error('failed to install skill', error)
    skillsError.value = t('components.skill.actions.installError', { name: skill.name })
  } finally {
    processingSkill.value = ''
  }
}

const handleUninstall = async (skill: SkillSummary) => {
  uninstallDropdownOpen.value = false
  openSkillMenuKey.value = ''
  selectSkill(skill)
  pendingUninstallSkill.value = skill
}

const handleUninstallAgent = async (skill: SkillSummary, agent: string) => {
  uninstallDropdownOpen.value = false
  selectSkill(skill)
  processingSkill.value = uninstallProcessingKey(skill)
  try {
    await uninstallSkill(skill.directory, [agent])
    skillsError.value = ''
    notice.value = t('components.skill.actions.uninstallSuccess', { name: skill.name })
    await Promise.all([loadGroupedSkills(), loadStatus()])
  } catch (error) {
    console.error('failed to uninstall skill for agent', error)
    skillsError.value = t('components.skill.actions.uninstallError', { name: skill.name })
  } finally {
    processingSkill.value = ''
  }
}

const closeUninstallModal = () => {
  pendingUninstallSkill.value = null
}

const confirmUninstall = async () => {
  const skill = pendingUninstallSkill.value
  if (!skill) {
    return
  }
  processingSkill.value = uninstallProcessingKey(skill)
  try {
    await uninstallSkill(skill.directory)
    skillsError.value = ''
    notice.value = t('components.skill.actions.uninstallSuccess', { name: skill.name })
    await Promise.all([loadGroupedSkills(), loadStatus()])
  } catch (error) {
    console.error('failed to uninstall skill', error)
    skillsError.value = t('components.skill.actions.uninstallError', { name: skill.name })
  } finally {
    processingSkill.value = ''
    pendingUninstallSkill.value = null
  }
}

const handleToggle = async (skill: SkillSummary, enabled: boolean) => {
  selectSkill(skill)
  openSkillMenuKey.value = ''
  togglingSkill.value = skill.directory
  try {
    await toggleSkill(skill.directory, enabled)
    notice.value = t(enabled ? 'components.skill.actions.enableSuccess' : 'components.skill.actions.disableSuccess', { name: skill.name })
    await Promise.all([loadGroupedSkills(), loadStatus()])
  } catch (error) {
    console.error('failed to toggle skill', error)
    skillsError.value = t('components.skill.actions.toggleError')
  } finally {
    togglingSkill.value = ''
  }
}

const openExternal = (target: string) => {
  if (!target) return
  Browser.OpenURL(target).catch(() => {
    console.error('failed to open external url', target)
  })
}

const openSkillRepo = (skill: SkillSummary) => {
  if (!skill.readme_url) return
  openExternal(skill.readme_url)
}

const openSelectedFolder = async () => {
  if (!selectedSkill.value?.installed) return
  await openInstalledFolder(selectedSkill.value)
}

const copyInstallCommand = async () => {
  if (!installCommand.value) return
  await navigator.clipboard.writeText(installCommand.value)
  notice.value = 'Copied install command'
}

const openInstalledFolder = async (skill: SkillSummary) => {
  try {
    await openInstalledSkillFolder(skill.directory)
  } catch (error) {
    console.error('failed to open installed skill folder', error)
  }
}

const saveSelectedContent = async () => {
  if (!selectedSkill.value?.installed || !contentDirty.value) return
  savingContent.value = true
  try {
    await saveSkillContent(selectedSkill.value.directory, skillContentDraft.value)
    originalSkillContent.value = skillContentDraft.value
    notice.value = t('components.skill.actions.saveSuccess')
    await loadGroupedSkills()
  } catch (error) {
    console.error('failed to save skill content', error)
    skillsError.value = t('components.skill.actions.saveError')
  } finally {
    savingContent.value = false
  }
}

const openBackup = async (path: string) => {
  if (!path) return
  try {
    await openSkillBackup(path)
  } catch (error) {
    console.error('failed to open backup path', error)
  }
}

const openBackupModal = () => {
  overflowMenuOpen.value = false
  backupModalOpen.value = true
}

const openRepoModal = () => {
  overflowMenuOpen.value = false
  repoModalOpen.value = true
}

const closeRepoModal = () => {
  repoModalOpen.value = false
}

const repoKey = (repo: SkillRepoConfig) => `${repo.owner}/${repo.name}`

const parseRepoInput = (value: string) => {
  let input = value.trim()
  if (!input) return null
  input = input.replace(/^https?:\/\/(www\.)?github\.com\//i, '')
  input = input.replace(/\.git$/i, '')
  const parts = input.split('/')
  if (parts.length < 2) return null
  const owner = parts[0]
  const name = parts[1]
  if (!owner || !name) return null
  return { owner, name }
}

const submitRepo = async () => {
  const parsed = parseRepoInput(repoForm.url)
  if (!parsed) {
    repoError.value = t('components.skill.repos.formError')
    return
  }

  repoBusy.value = true
  repoError.value = ''
  try {
    repoList.value = await addSkillRepo({
      owner: parsed.owner,
      name: parsed.name,
      branch: repoForm.branch || 'main',
      enabled: true
    })
    repoForm.url = ''
    repoForm.branch = 'main'
    await loadGroupedSkills()
    notice.value = t('components.skill.repos.addSuccess')
  } catch (error) {
    console.error('failed to add repo', error)
    repoError.value = t('components.skill.repos.addError')
  } finally {
    repoBusy.value = false
  }
}

const removeRepoItem = async (repo: SkillRepoConfig) => {
  repoBusy.value = true
  repoError.value = ''
  try {
    repoList.value = await removeSkillRepo(repo.owner, repo.name)
    await loadGroupedSkills()
    notice.value = t('components.skill.repos.removeSuccess')
  } catch (error) {
    console.error('failed to remove repo', error)
    repoError.value = t('components.skill.repos.removeError')
  } finally {
    repoBusy.value = false
  }
}

const openRepoGithub = (repo: SkillRepoConfig) => {
  if (!repo.owner || !repo.name) return
  openExternal(`https://github.com/${repo.owner}/${repo.name}`)
}

const goHome = () => {
  router.push('/')
}

const formatLinkStatus = (status?: string) => {
  if (!status) return t('components.skill.status.notDetected')
  return t(`components.skill.statusMap.${status}`)
}

const formatMigrationStatus = (status?: string) => {
  if (!status) return t('components.skill.summary.unknownStatus')
  const key = `components.skill.migrationStatus.${status}`
  const knownStatuses = new Set(['migrated', 'duplicate', 'conflict', 'linked', 'missing', 'repair_failed', 'unsupported'])
  return knownStatuses.has(status) ? t(key) : status
}

const statusClass = (status?: string) => {
  if (status === 'linked') return 'linked'
  if (status === 'missing') return 'missing'
  if (status === 'conflict' || status === 'repair_failed') return 'danger'
  return 'neutral'
}

const badgeClassForMigration = (status?: string) => {
  if (status === 'migrated' || status === 'duplicate') return 'enabled'
  if (status === 'conflict') return 'conflict'
  return 'enabled off'
}

const recordKey = (record: MigrationRecord) => {
  return `${record.platform || 'na'}:${record.directory || 'none'}:${record.status || 'unknown'}:${record.created_at || ''}`
}

const formatDate = (value?: string) => {
  if (!value) return t('components.skill.summary.none')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US', {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(date)
}

const extractSkillBody = (content: string) => {
  const normalized = content.replace(/\r\n/g, '\n')
  const parts = normalized.split('\n---\n')
  if (parts.length < 3) return normalized
  return parts.slice(2).join('\n---\n').trim()
}

onMounted(() => {
  void Promise.all([loadGroupedSkills(), loadStatus(), loadRepos()])
})
</script>

<style scoped>
.skill-workspace {
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--mac-text);
}

.skill-header {
  display: block;
  padding: 0 2px;
}

.micro-label {
  margin: 0 0 8px;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--mac-text-secondary);
}

.skill-header h1 {
  margin: 0;
  font-size: clamp(1.8rem, 2.4vw, 2.4rem);
  font-weight: 800;
  letter-spacing: -0.04em;
  line-height: 1.05;
}

.sync-strip,
.skill-sidebar,
.skill-detail {
  border: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 90%, transparent);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.08);
}

.sync-strip {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border-radius: 18px;
  flex-wrap: wrap;
}

.sync-node-shell {
  position: relative;
}

.sync-node {
  width: 72px;
  height: 72px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 58%, transparent);
  background: color-mix(in srgb, var(--mac-surface-strong) 72%, transparent);
  color: inherit;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.sync-node:hover {
  transform: translateY(-1px);
}

.sync-node-icon {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  font-size: 0.84rem;
  font-weight: 700;
  background: color-mix(in srgb, var(--mac-text-secondary) 12%, transparent);
}

.sync-node-label {
  font-size: 0.72rem;
  color: var(--mac-text-secondary);
}

.sync-node.is-linked {
  border-color: color-mix(in srgb, #22c55e 42%, var(--mac-border));
  background: color-mix(in srgb, #22c55e 12%, var(--mac-surface));
  color: #22c55e;
}

.sync-node.is-danger {
  border-color: color-mix(in srgb, #ef4444 42%, var(--mac-border));
  background: color-mix(in srgb, #ef4444 12%, var(--mac-surface));
  color: #ef4444;
}

.sync-node.is-neutral {
  color: var(--mac-text-secondary);
}

.sync-node[data-tooltip]::after {
  content: attr(data-tooltip);
  position: absolute;
  left: 50%;
  bottom: calc(100% + 8px);
  transform: translateX(-50%);
  padding: 6px 8px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.94);
  color: #fff;
  font-size: 11px;
  line-height: 1.25;
  white-space: nowrap;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease, transform 0.15s ease;
  z-index: 20;
}

.sync-node[data-tooltip]:hover::after,
.sync-node[data-tooltip]:focus-visible::after {
  opacity: 1;
  transform: translateX(-50%) translateY(-2px);
}

.sync-node-corner {
  position: relative;
  position: absolute;
  top: 8px;
  right: 8px;
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, currentColor 18%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 82%, transparent);
  color: var(--mac-text-secondary);
  z-index: 1;
}

.sync-node-corner svg {
  width: 12px;
  height: 12px;
}

.sync-node-corner:disabled {
  opacity: 0.55;
}

.sync-node-corner[data-tooltip]::after {
  content: attr(data-tooltip);
  position: absolute;
  left: 50%;
  bottom: calc(100% + 8px);
  transform: translateX(-50%);
  padding: 6px 8px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.94);
  color: #fff;
  font-size: 11px;
  line-height: 1.25;
  white-space: nowrap;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease, transform 0.15s ease;
  z-index: 20;
}

.sync-node-corner[data-tooltip]:hover::after,
.sync-node-corner[data-tooltip]:focus-visible::after {
  opacity: 1;
  transform: translateX(-50%) translateY(-2px);
}

.skill-banner {
  padding: 12px 14px;
  border-radius: 12px;
  background: color-mix(in srgb, #0a84ff 16%, transparent);
  color: var(--mac-text);
  border: 1px solid color-mix(in srgb, #0a84ff 28%, transparent);
}

.skill-banner.error {
  background: color-mix(in srgb, #ef4444 16%, transparent);
  border-color: color-mix(in srgb, #ef4444 28%, transparent);
}

.skill-banner.compact {
  margin-top: 12px;
}

.skill-layout {
  display: grid;
  grid-template-columns: minmax(330px, 390px) minmax(0, 1fr);
  gap: 18px;
  min-height: 740px;
}

.skill-sidebar,
.skill-detail {
  border-radius: 26px;
  overflow: hidden;
}

.sidebar-header {
  padding: 20px 20px 18px;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--mac-surface-strong) 92%, transparent), color-mix(in srgb, var(--mac-surface) 86%, transparent));
}

.sidebar-tools-row {
  display: flex;
  justify-content: flex-end;
}

.sidebar-tools {
  display: flex;
  gap: 8px;
  align-items: center;
}

.menu-anchor {
  position: relative;
}

.menu-popover,
.row-menu {
  position: absolute;
  right: 0;
  z-index: 20;
  min-width: 190px;
  padding: 6px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 96%, transparent);
  box-shadow: 0 20px 36px rgba(0, 0, 0, 0.22);
}

.menu-popover {
  top: calc(100% + 8px);
}

.menu-popover button,
.row-menu button {
  width: 100%;
  padding: 9px 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: inherit;
  text-align: left;
}

.menu-popover button:hover,
.row-menu button:hover {
  background: color-mix(in srgb, var(--mac-surface-strong) 78%, transparent);
}

.row-menu button.danger {
  color: #ef4444;
}

.search-shell {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  padding: 12px 12px;
  border-radius: 16px;
  background: color-mix(in srgb, var(--mac-surface) 78%, transparent);
  border: 1px solid color-mix(in srgb, #9ec0ff 14%, var(--mac-border));
}

.search-shell svg {
  width: 16px;
  height: 16px;
  color: var(--mac-text-secondary);
}

.search-shell input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: inherit;
}

.ghost-inline {
  border: 0;
  background: transparent;
  color: var(--mac-text-secondary);
  font-size: 0.76rem;
}

.ghost-inline.filter {
  padding: 4px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--mac-surface-strong) 78%, transparent);
}

.sidebar-body {
  padding: 16px;
  max-height: 980px;
  overflow: auto;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--mac-surface) 92%, transparent), color-mix(in srgb, var(--mac-surface-strong) 74%, transparent));
}

.skill-group-panel + .skill-group-panel {
  margin-top: 10px;
}

.group-toggle,
.subgroup-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border: 0;
  background: transparent;
  color: inherit;
}

.group-toggle {
  padding: 12px 12px;
  border-radius: 14px;
  font-size: 0.8rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.group-toggle:hover,
.subgroup-toggle:hover {
  background: color-mix(in srgb, var(--mac-surface) 82%, transparent);
}

.group-badge {
  min-width: 26px;
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--mac-surface-strong) 82%, transparent);
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  font-size: 0.72rem;
  font-weight: 700;
  text-align: center;
}

.group-body {
  margin-top: 6px;
}

.subgroup-panel {
  margin-top: 8px;
  padding: 10px;
  border-radius: 16px;
  background: linear-gradient(180deg, color-mix(in srgb, var(--mac-surface) 74%, transparent), color-mix(in srgb, var(--mac-surface-strong) 68%, transparent));
  border: 1px solid color-mix(in srgb, var(--mac-border) 52%, transparent);
}

.subgroup-panel.recommended {
  padding: 10px;
}

.subgroup-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.subgroup-caret,
.subgroup-label {
  color: var(--mac-text-secondary);
  font-size: 0.78rem;
}

.subgroup-list {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.skill-row-wrap {
  position: relative;
}

.row-menu {
  top: calc(100% + 6px);
}

.skill-empty {
  padding: 26px 20px;
  text-align: center;
  color: var(--mac-text-secondary);
}

.skill-empty.compact {
  padding: 14px 10px;
}

.detail-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 10px;
  color: var(--mac-text-secondary);
}

.detail-header {
  padding: 24px 24px 18px;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background:
    linear-gradient(180deg, rgba(12, 22, 40, 0.92), rgba(14, 22, 34, 0.92));
}

.detail-header-main {
  display: flex;
  gap: 16px;
  align-items: center;
}

.detail-icon {
  width: 72px;
  height: 72px;
  border-radius: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.7rem;
  font-weight: 800;
  background:
    radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.24), transparent 55%),
    linear-gradient(145deg, rgba(10, 132, 255, 0.26), rgba(16, 185, 129, 0.22));
}

.detail-icon.conflict {
  background:
    radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.22), transparent 55%),
    linear-gradient(145deg, rgba(239, 68, 68, 0.38), rgba(245, 158, 11, 0.24));
}

.detail-copy {
  min-width: 0;
}

.detail-publisher {
  margin: 0 0 4px;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--mac-text-secondary);
}

.detail-copy h2 {
  margin: 0;
  font-size: clamp(1.6rem, 2vw, 1.95rem);
  letter-spacing: -0.03em;
}

.detail-tagline {
  margin: 8px 0 0;
  font-size: 1rem;
  line-height: 1.5;
}

.detail-subline {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  margin-top: 10px;
  font-size: 0.82rem;
  color: var(--mac-text-secondary);
}

.detail-subline span {
  position: relative;
}

.detail-subline span:not(:first-child)::before {
  content: '';
  position: absolute;
  left: -9px;
  top: 50%;
  width: 3px;
  height: 3px;
  border-radius: 999px;
  background: var(--mac-text-secondary);
  transform: translateY(-50%);
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}

.detail-tabs {
  display: flex;
  gap: 8px;
  padding: 0 24px;
  margin: 0;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
}

.detail-tabs button {
  padding: 12px 2px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--mac-text-secondary);
}

.detail-tabs button.active {
  color: var(--mac-text);
  border-bottom: 2px solid var(--mac-accent);
}

.detail-panel,
.timeline-item {
  border-radius: 18px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 88%, transparent);
}

.detail-panel {
  margin: 20px 24px 24px;
  padding: 18px;
  min-height: 440px;
}

.detail-layout-vscode {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 28px;
  padding: 24px;
}

.detail-content-pane,
.detail-sidebar-vscode {
  min-width: 0;
}

.detail-content-pane {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.detail-sidebar-vscode {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.detail-block,
.side-panel {
  padding-top: 2px;
  border-top: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
}

.detail-block-label,
.side-panel h3 {
  margin: 0 0 14px;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--mac-text-secondary);
}

.install-command-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.detail-prose {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.65;
  color: var(--mac-text-secondary);
  font-family: inherit;
}

.side-facts {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
}

.side-facts dt {
  font-size: 0.74rem;
  color: var(--mac-text-secondary);
  margin-bottom: 4px;
}

.side-facts dd {
  margin: 0;
  word-break: break-word;
  line-height: 1.5;
}

.side-panel-value {
  margin: 0;
  line-height: 1.5;
}

.side-panel-subtle,
.fact-link {
  margin: 8px 0 0;
  line-height: 1.5;
  color: var(--mac-text-secondary);
}

.side-links {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.side-link-btn {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
}

.editor-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--mac-text-secondary);
  font-size: 0.82rem;
}

.skill-editor {
  flex: 1;
  min-height: 360px;
  resize: vertical;
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: color-mix(in srgb, var(--mac-surface-strong) 90%, transparent);
  color: var(--mac-text);
  padding: 14px;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.82rem;
  line-height: 1.5;
  outline: none;
}

.detail-placeholder {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: var(--mac-text-secondary);
}

.timeline-block h3 {
  margin: 0 0 8px;
  font-size: 0.92rem;
}

.timeline-block {
  height: 100%;
}

.timeline-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.timeline-item {
  padding: 14px;
}

.timeline-item-head {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: center;
}

.timeline-item p,
.timeline-item small {
  margin: 8px 0 0;
}

.timeline-item p {
  color: var(--mac-text-secondary);
}

.resource-toolbar,
.repo-item-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.resource-toolbar {
  margin-top: 2px;
}

.tool-icon {
  position: relative;
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  background: color-mix(in srgb, var(--mac-surface-strong) 82%, transparent);
  color: var(--mac-text);
}

.tool-icon.compact {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
}

.tool-icon svg {
  width: 18px;
  height: 18px;
}

.tool-icon:disabled {
  opacity: 0.45;
}

.tool-icon[data-tooltip] {
  position: relative;
}

.tool-icon[data-tooltip]::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  padding: 5px 8px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.94);
  color: #fff;
  font-size: 11px;
  line-height: 1.2;
  white-space: nowrap;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease, transform 0.15s ease;
  z-index: 20;
}

.tool-icon[data-tooltip]:hover::after,
.tool-icon[data-tooltip]:focus-visible::after {
  opacity: 1;
  transform: translateX(-50%) translateY(-2px);
}

.tool-status-dot {
  position: absolute;
  right: 7px;
  bottom: 7px;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #9ca3af;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--mac-surface) 88%, transparent);
}

.tool-status-dot.linked {
  background: #22c55e;
}

.tool-status-dot.missing {
  background: #f59e0b;
}

.tool-status-dot.danger {
  background: #ef4444;
}

.btn-primary,
.btn-secondary {
  border-radius: 10px;
  border: 1px solid transparent;
  padding: 10px 14px;
  font-weight: 700;
}

.btn-primary {
  background: color-mix(in srgb, var(--mac-accent) 88%, white 8%);
  color: white;
}

.btn-primary:disabled,
.btn-secondary:disabled {
  opacity: 0.55;
}

.btn-secondary {
  background: color-mix(in srgb, var(--mac-surface-strong) 88%, transparent);
  color: var(--mac-text);
  border-color: color-mix(in srgb, var(--mac-border) 68%, transparent);
}

.btn-secondary.danger {
  color: #ef4444;
}

.ghost-icon,
.ghost-icon.sm {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--mac-text-secondary);
}

.ghost-icon.sm {
  width: 30px;
  height: 30px;
}

.ghost-icon:hover {
  background: color-mix(in srgb, var(--mac-surface) 82%, transparent);
  color: var(--mac-text);
}

.ghost-icon svg {
  width: 18px;
  height: 18px;
}

.ghost-icon svg.spin {
  animation: skill-spin 0.8s linear infinite;
}

.repo-modal-content,
.modal-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.repo-modal-copy {
  margin: 0;
  color: var(--mac-text-secondary);
  line-height: 1.5;
}

.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.repo-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.repo-form input {
  width: 100%;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 90%, transparent);
  color: inherit;
  padding: 11px 12px;
  outline: none;
}

.repo-form-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
}

.repo-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.repo-item,
.modal-list-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  padding: 14px;
  border-radius: 14px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 88%, transparent);
}

.repo-item p,
.modal-list-item p,
.modal-list-item small {
  margin: 6px 0 0;
  color: var(--mac-text-secondary);
}

.modal-list-item.conflict {
  border-color: color-mix(in srgb, #ef4444 28%, var(--mac-border));
}

@keyframes skill-spin {
  to {
    transform: rotate(360deg);
  }
}

.split-btn-group {
  position: relative;
  display: inline-flex;
}

.split-btn-group .split-main {
  border-radius: 10px 0 0 10px;
  border-right: none;
}

.split-btn-group .split-caret {
  border-radius: 0 10px 10px 0;
  padding: 10px 10px;
  font-size: 0.72rem;
}

.split-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 30;
  min-width: 170px;
  padding: 6px;
  border-radius: 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 96%, transparent);
  box-shadow: 0 20px 36px rgba(0, 0, 0, 0.22);
}

.split-dropdown button {
  width: 100%;
  padding: 9px 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: inherit;
  text-align: left;
  font-size: 0.88rem;
}

.split-dropdown button:hover {
  background: color-mix(in srgb, var(--mac-surface-strong) 78%, transparent);
}

.agent-chips {
  display: flex;
  gap: 6px;
  align-items: center;
}

.agent-chip {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 58%, transparent);
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--mac-text-secondary);
  background: color-mix(in srgb, var(--mac-surface-strong) 72%, transparent);
}

.agent-chip.active {
  color: #22c55e;
  border-color: color-mix(in srgb, #22c55e 48%, transparent);
  background: color-mix(in srgb, #22c55e 12%, var(--mac-surface));
}

.side-icon-links {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.side-icon-link {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 0;
  border: 0;
  background: transparent;
  color: var(--mac-accent);
  text-align: left;
  font-size: 0.88rem;
  cursor: pointer;
}

.side-icon-link:disabled {
  color: var(--mac-text-secondary);
  cursor: default;
}

.side-icon-link:hover:not(:disabled) {
  text-decoration: underline;
}

.side-icon-link svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.global-menu-popover {
  top: calc(100% + 8px);
  right: 0;
  left: auto;
}

@media (max-width: 1180px) {
  .detail-layout-vscode {
    grid-template-columns: 1fr;
  }

  .skill-layout {
    grid-template-columns: 1fr;
  }

  .skill-sidebar {
    min-height: 420px;
  }
}

@media (max-width: 720px) {
  .skill-header,
  .detail-header-main {
    flex-direction: column;
  }

  .sync-strip {
    flex-wrap: wrap;
  }

  .detail-actions {
    width: 100%;
  }

  .btn-primary,
  .btn-secondary {
    width: 100%;
  }

  .detail-tabs,
  .detail-layout-vscode,
  .detail-panel {
    padding-left: 16px;
    padding-right: 16px;
  }
}
</style>
