<template>
  <div class="main-shell">
    <div class="skill-workspace">
      <div v-if="skillsError" class="skill-banner error">{{ skillsError }}</div>
      <div v-else-if="notice" class="skill-banner">{{ notice }}</div>

      <section class="skill-layout">
        <aside class="skill-sidebar">
          <header class="sidebar-header">
            <div class="sidebar-topbar">
              <span class="sidebar-title">SKILLS</span>
              <div class="sidebar-topbar-actions">
                <button class="ghost-icon sm" :title="t('components.skill.actions.refresh')" :disabled="refreshing" @click="refresh">
                  <svg viewBox="0 0 24 24" aria-hidden="true" :class="{ spin: refreshing }">
                    <path d="M20.5 8a8.5 8.5 0 10-2.38 7.41" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                    <path d="M20.5 4v4h-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
                <div class="menu-anchor">
                  <button
                    class="ghost-icon sm"
                    :class="{ active: sidebarMenuOpen }"
                    :title="t('components.skill.actions.more')"
                    @click="toggleSidebarMenu"
                  >
                    <svg viewBox="0 0 24 24" aria-hidden="true">
                      <path d="M12 6a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 9a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm0 9a1.5 1.5 0 110-3 1.5 1.5 0 010 3z" fill="currentColor" />
                    </svg>
                  </button>
                  <div v-if="sidebarMenuOpen" class="menu-popover sidebar-menu-popover">
                    <div class="menu-item-with-submenu">
                      <button type="button" class="menu-item-row" @click="toggleSidebarViews">
                        <span>Views</span>
                        <span class="submenu-caret">›</span>
                      </button>
                      <div v-if="sidebarViewsOpen" class="menu-popover submenu-popover">
                        <button type="button" @click="toggleSection('installed')">
                          <span class="menu-check">{{ visibleSections.installed ? '✓' : '' }}</span>
                          <span>Installed</span>
                        </button>
                        <button type="button" @click="toggleSection('recommended')">
                          <span class="menu-check">{{ visibleSections.recommended ? '✓' : '' }}</span>
                          <span>Recommended</span>
                        </button>
                        <button type="button" @click="toggleSection('enabled')">
                          <span class="menu-check">{{ visibleSections.enabled ? '✓' : '' }}</span>
                          <span>Enabled</span>
                        </button>
                        <button type="button" @click="toggleSection('disabled')">
                          <span class="menu-check">{{ visibleSections.disabled ? '✓' : '' }}</span>
                          <span>Disabled</span>
                        </button>
                        <button type="button" @click="setFilterMode('mcp')">
                          <span class="menu-check">{{ filterMode === 'mcp' ? '✓' : '' }}</span>
                          <span>MCP Servers</span>
                        </button>
                        <button type="button" @click="setFilterMode('plugins')">
                          <span class="menu-check">{{ filterMode === 'plugins' ? '✓' : '' }}</span>
                          <span>Agent Plugins</span>
                        </button>
                      </div>
                    </div>
                    <button type="button" @click="refresh">
                      {{ t('components.skill.menu.checkForSkillUpdates') }}
                    </button>
                    <button type="button" @click="refresh">
                      {{ t('components.skill.menu.updateAllSkills') }}
                    </button>
                    <button type="button" @click="disableAutoUpdateForAll">
                      {{ t('components.skill.menu.disableAutoUpdateForAllSkills') }}
                    </button>
                    <div class="menu-divider"></div>
                    <button type="button" @click="setAllEnabled(true)">
                      {{ t('components.skill.menu.enableAllSkills') }}
                    </button>
                    <button type="button" @click="setAllEnabled(false)">
                      {{ t('components.skill.menu.disableAllInstalledSkills') }}
                    </button>
                    <button type="button" @click="showEnabledView">
                      {{ t('components.skill.menu.showEnabledSkills') }}
                    </button>
                    <button type="button" @click="showDisabledView">
                      {{ t('components.skill.menu.showDisabledSkills') }}
                    </button>
                    <button type="button" @click="openBisect">
                      {{ t('components.skill.menu.startSkillBisect') }}
                    </button>
                    <div class="menu-divider"></div>
                    <button type="button" @click="openRepoModal">
                      {{ t('components.skill.menu.installFromRepository') }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
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
              <section v-if="visibleSections.installed" class="skill-group-panel">
                <button class="group-toggle" type="button" @click="collapsed.installed = !collapsed.installed">
                  <span class="group-toggle-main">
                    <span class="group-chevron" :class="{ collapsed: collapsed.installed }" aria-hidden="true">
                      <svg viewBox="0 0 16 16">
                        <path d="M4.5 6.5 8 10l3.5-3.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
                      </svg>
                    </span>
                    <span>Installed</span>
                  </span>
                  <span class="group-badge">{{ filteredInstalledCount }}</span>
                </button>

                <div v-if="!collapsed.installed" class="group-body">
                  <div v-if="!filteredInstalledGroups.length" class="skill-empty compact">
                    {{ t('components.skill.list.noInstalled') }}
                  </div>

                  <div class="subgroup-list">
                    <div v-for="skill in flatInstalledSkills" :key="skillIdentity(skill)" class="skill-row-wrap">
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
                </div>
              </section>

              <section v-if="visibleSections.recommended" class="skill-group-panel">
                <button class="group-toggle" type="button" @click="collapsed.recommended = !collapsed.recommended">
                  <span class="group-toggle-main">
                    <span class="group-chevron" :class="{ collapsed: collapsed.recommended }" aria-hidden="true">
                      <svg viewBox="0 0 16 16">
                        <path d="M4.5 6.5 8 10l3.5-3.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
                      </svg>
                    </span>
                    <span>Recommended</span>
                  </span>
                  <span class="group-badge">{{ filteredAvailableCount }}</span>
                </button>

                <div v-if="!collapsed.recommended" class="group-body">
                  <div v-if="!filteredAvailableGroups.length" class="skill-empty compact">
                    {{ t('components.skill.list.noRecommended') }}
                  </div>

                  <div class="subgroup-list">
                    <div v-for="skill in flatAvailableSkills" :key="skillIdentity(skill)" class="skill-row-wrap">
                      <SkillCard
                        :skill="skill"
                        :selected="selectedSkillKey === skillIdentity(skill)"
                        :loading="isInstallingSkill(skill)"
                        :disabled="!canInstallSkill(skill)"
                        :show-install-button="true"
                        @select="selectSkill"
                        @menu="toggleSkillMenu"
                      />
                      <div v-if="openSkillMenuKey === skillIdentity(skill)" class="row-menu">
                        <button type="button" @click="showRecommendedMenuNotice(skill)">
                          Add to repo skills
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              <section v-if="visibleSections.enabled" class="skill-group-panel">
                <button class="group-toggle" type="button" @click="collapsed.enabled = !collapsed.enabled">
                  <span class="group-toggle-main">
                    <span class="group-chevron" :class="{ collapsed: collapsed.enabled }" aria-hidden="true">
                      <svg viewBox="0 0 16 16">
                        <path d="M4.5 6.5 8 10l3.5-3.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
                      </svg>
                    </span>
                    <span>Enabled</span>
                  </span>
                  <span class="group-badge">{{ enabledSkillsCount }}</span>
                </button>

                <div v-if="!collapsed.enabled" class="group-body">
                  <div v-if="!enabledSkills.length" class="skill-empty compact">
                    {{ t('components.skill.list.noEnabled') }}
                  </div>

                  <div class="subgroup-list">
                    <div v-for="skill in enabledSkills" :key="`enabled:${skillIdentity(skill)}`" class="skill-row-wrap">
                      <SkillCard
                        :skill="skill"
                        :selected="selectedSkillKey === skillIdentity(skill)"
                        :conflict="isConflictSkill(skill)"
                        @select="selectSkill"
                        @menu="toggleSkillMenu"
                      />
                      <div v-if="openSkillMenuKey === skillIdentity(skill)" class="row-menu">
                        <button type="button" @click="handleToggle(skill, false)">
                          {{ t('components.skill.actions.disable') }}
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
                </div>
              </section>

              <section v-if="visibleSections.disabled" class="skill-group-panel">
                <button class="group-toggle" type="button" @click="collapsed.disabled = !collapsed.disabled">
                  <span class="group-toggle-main">
                    <span class="group-chevron" :class="{ collapsed: collapsed.disabled }" aria-hidden="true">
                      <svg viewBox="0 0 16 16">
                        <path d="M4.5 6.5 8 10l3.5-3.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" />
                      </svg>
                    </span>
                    <span>Disabled</span>
                  </span>
                  <span class="group-badge">{{ disabledSkillsCount }}</span>
                </button>

                <div v-if="!collapsed.disabled" class="group-body">
                  <div v-if="!disabledSkills.length" class="skill-empty compact">
                    {{ t('components.skill.list.noDisabled') }}
                  </div>

                  <div class="subgroup-list">
                    <div v-for="skill in disabledSkills" :key="`disabled:${skillIdentity(skill)}`" class="skill-row-wrap">
                      <SkillCard
                        :skill="skill"
                        :selected="selectedSkillKey === skillIdentity(skill)"
                        :conflict="isConflictSkill(skill)"
                        @select="selectSkill"
                        @menu="toggleSkillMenu"
                      />
                      <div v-if="openSkillMenuKey === skillIdentity(skill)" class="row-menu">
                        <button type="button" @click="handleToggle(skill, true)">
                          {{ t('components.skill.actions.enable') }}
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
                  <h2>{{ selectedSkill.name }}</h2>
                  <div class="detail-meta-row">
                    <span class="detail-publisher">{{ selectedSourceLabel }}</span>
                    <span class="detail-meta-sep">•</span>
                    <span class="detail-meta-status">{{ detailStatusLabel }}</span>
                  </div>
                  <p class="detail-tagline">{{ selectedSkill.description || t('components.skill.list.noDescription') }}</p>
                  <div class="detail-actions">
                    <button
                      class="agent-install-btn"
                      :class="{ installed: selectedSkill.agents?.claude }"
                      :disabled="isInstallingSkill(selectedSkill) || processingSkill === uninstallProcessingKey(selectedSkill)"
                      :title="selectedSkill.agents?.claude ? t('components.skill.actions.uninstallClaude') : t('components.skill.actions.installClaude')"
                      @click="selectedSkill.agents?.claude ? handleUninstallAgent(selectedSkill, 'claude') : handleInstall(selectedSkill, ['claude'])"
                    >
                      <span v-if="claudeIcon" class="agent-install-icon" v-html="claudeIcon" aria-hidden="true"></span>
                      <span class="sr-only">Claude</span>
                    </button>
                    <button
                      class="agent-install-btn"
                      :class="{ installed: selectedSkill.agents?.codex }"
                      :disabled="isInstallingSkill(selectedSkill) || processingSkill === uninstallProcessingKey(selectedSkill)"
                      :title="selectedSkill.agents?.codex ? t('components.skill.actions.uninstallCodex') : t('components.skill.actions.installCodex')"
                      @click="selectedSkill.agents?.codex ? handleUninstallAgent(selectedSkill, 'codex') : handleInstall(selectedSkill, ['codex'])"
                    >
                      <span v-if="codexIcon" class="agent-install-icon" v-html="codexIcon" aria-hidden="true"></span>
                      <span class="sr-only">Codex</span>
                    </button>
                    <button class="detail-inline-toggle" type="button" disabled :title="t('components.skill.actions.autoUpdateComingSoon')">
                      <span class="detail-inline-check" aria-hidden="true"></span>
                      <span>Auto Update</span>
                    </button>
                    <button class="detail-meta-link action" type="button" :disabled="!selectedSkill.readme_url" @click="openSkillRepo(selectedSkill)">
                      {{ t('components.skill.actions.openRepo') }}
                    </button>
                    <button
                      v-if="selectedSkill.installed"
                      class="btn-secondary"
                      :disabled="togglingSkill === selectedSkill.directory || isConflictSkill(selectedSkill)"
                      @click="handleToggle(selectedSkill, !selectedSkill.enabled)"
                    >{{ selectedSkill.enabled ? t('components.skill.actions.disable') : t('components.skill.actions.enable') }}</button>
                  </div>
                </div>
              </div>
            </header>

            <nav class="detail-tabs">
              <button :class="{ active: activeTab === 'details' }" @click="activeTab = 'details'">
                {{ t('components.skill.tabs.details') }}
              </button>
              <button :class="{ active: activeTab === 'changelog' }" @click="activeTab = 'changelog'">
                {{ t('components.skill.tabs.changelog') }}
              </button>
            </nav>

            <div v-if="activeTab === 'details'" class="detail-layout-vscode">
              <section class="detail-content-pane">
                <section class="detail-hero-block">
                  <div class="detail-block-label">OVERVIEW</div>
                  <p class="detail-hero-copy">{{ selectedSkill.description || t('components.skill.list.noDescription') }}</p>
                </section>

                <section class="detail-block install">
                  <div class="detail-block-head">
                    <div class="detail-block-label">INSTALLATION</div>
                    <span class="detail-block-kicker">{{ selectedSkill.installed ? 'Installed locally' : 'Install from repository' }}</span>
                  </div>
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
                  <div class="detail-block-head">
                    <div class="detail-block-label">README</div>
                    <span class="detail-block-kicker">{{ selectedSkill.installed ? 'Synced from SKILL.md' : 'Preview from repository metadata' }}</span>
                  </div>
                  <div class="detail-readme-shell">
                    <pre class="detail-prose">{{ selectedOverview }}</pre>
                  </div>
                </section>
              </section>

              <aside class="detail-sidebar-vscode">
                <section class="side-panel">
                  <h3>Installation</h3>
                  <dl class="side-facts">
                    <div>
                      <dt>Identifier</dt>
                      <dd>{{ detailIdentifier }}</dd>
                    </div>
                    <div>
                      <dt>Version</dt>
                      <dd>{{ detailVersion }}</dd>
                    </div>
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
                      <dd>{{ detailStatusLabel }}</dd>
                    </div>
                  </dl>
                </section>

                <section class="side-panel">
                  <h3>Marketplace</h3>
                  <dl class="side-facts">
                    <div>
                      <dt>Published</dt>
                      <dd>{{ detailPublished }}</dd>
                    </div>
                    <div>
                      <dt>Last Updated</dt>
                      <dd>{{ detailLastUpdated }}</dd>
                    </div>
                    <div>
                      <dt>Categories</dt>
                      <dd>
                        <span class="side-tag">{{ detailCategories }}</span>
                      </dd>
                    </div>
                  </dl>
                </section>

                <section class="side-panel">
                  <h3>Resources</h3>
                  <div class="side-resource-list">
                    <button class="side-resource-link" type="button" @click="selectedSkill.installed ? openSelectedFolder() : handleOpenFolder()">
                      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>
                      <span>{{ selectedSkill.installed ? 'Local folder' : 'Repo skills folder' }}</span>
                    </button>
                    <button class="side-resource-link" type="button" :disabled="!selectedSkill.readme_url" @click="openSkillRepo(selectedSkill)">
                      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M14 5h5v5M10 14L19 5M19 13v5h-5M5 19l6-6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                      <span>Marketplace page</span>
                    </button>
                    <button class="side-resource-link" type="button" @click="backupModalOpen = true">
                      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5.25 8.25h13.5M8.25 12h7.5m-9 3.75h10.5A2.25 2.25 0 0019.5 13.5v-6A2.25 2.25 0 0017.25 5.25H6.75A2.25 2.25 0 004.5 7.5v6a2.25 2.25 0 002.25 2.25z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                      <span>Backups</span>
                    </button>
                  </div>
                </section>
              </aside>
            </div>

            <div v-else class="detail-panel">
              <div v-if="selectedSkill.installed" class="editor-shell">
                <div class="editor-toolbar">
                  <span>{{ t('components.skill.tabs.contentHelp') }}</span>
                  <button class="btn-secondary" :disabled="!contentDirty || savingContent" @click="saveSelectedContent">
                    {{ savingContent ? t('common.saving') : t('common.save') }}
                  </button>
                </div>
                <textarea v-model="skillContentDraft" class="skill-editor" spellcheck="false"></textarea>
              </div>
              <div v-else class="timeline-block">
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
import lobeIcons from '../../icons/lobeIconMap'
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

type FilterMode = 'all' | 'enabled' | 'conflict' | 'disabled' | 'mcp' | 'plugins'
type DetailTab = 'details' | 'changelog'
type ViewFilterOption = 'installed' | 'recommended' | 'enabled' | 'disabled'

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
const activeTab = ref<DetailTab>('details')
const skillContentDraft = ref('')
const originalSkillContent = ref('')
const savingContent = ref(false)
const selectedSkillKey = ref('')
const searchQuery = ref('')
const filterMode = ref<FilterMode>('all')
const sidebarMenuOpen = ref(false)
const sidebarViewsOpen = ref(false)
const openSkillMenuKey = ref('')
const repoModalOpen = ref(false)
const backupModalOpen = ref(false)
const conflictModalOpen = ref(false)
const pendingUninstallSkill = ref<SkillSummary | null>(null)

const collapsed = reactive({
  installed: false,
  recommended: false,
  enabled: true,
  disabled: true
})
const visibleSections = reactive({
  installed: true,
  recommended: true,
  enabled: false,
  disabled: false
})
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

const claudeIcon = computed(() => lobeIcons['claude'] ?? lobeIcons['anthropic'] ?? '')
const codexIcon = computed(() => lobeIcons['openai'] ?? '')

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
  if (selectedSkill.value.installed) {
    const body = extractSkillBody(originalSkillContent.value).trim()
    return body || selectedSkill.value.description || t('components.skill.list.noDescription')
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

const detailIdentifier = computed(() => {
  if (!selectedSkill.value) return '-'
  const repo = selectedSkill.value.repo_name?.trim() || selectedSkill.value.repo_owner?.trim() || 'local'
  return `${repo}.${selectedSkill.value.directory}`
})

const detailVersion = computed(() => {
  if (!selectedSkill.value) return '-'
  const branch = selectedSkill.value.repo_branch?.trim() || ''
  if (/^[0-9a-f]{8,40}$/i.test(branch)) {
    return branch.slice(0, 8)
  }
  return 'unknown'
})

const detailPublished = computed(() => {
  if (!selectedSkill.value) return '-'
  return '待开发'
})

const detailLastUpdated = computed(() => {
  if (!selectedSkill.value) return '-'
  return '待配置'
})

const detailCategories = computed(() => {
  if (!selectedSkill.value) return '-'
  return '待配置'
})

const detailStatusLabel = computed(() => {
  if (!selectedSkill.value) return '-'
  if (!selectedSkill.value.installed) return 'Recommended'
  return selectedSkill.value.enabled ? t('components.skill.badges.enabled') : t('components.skill.badges.disabled')
})

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
const flatInstalledSkills = computed(() => filteredInstalledGroups.value.flatMap((group) => group.skills))
const flatAvailableSkills = computed(() => filteredAvailableGroups.value.flatMap((group) => group.skills))
const filteredInstalledCount = computed(() => flatInstalledSkills.value.length)
const filteredAvailableCount = computed(() => flatAvailableSkills.value.length)
const enabledSkills = computed(() => flatInstalledSkills.value.filter((skill) => skill.enabled))
const disabledSkills = computed(() => flatInstalledSkills.value.filter((skill) => !skill.enabled))
const enabledSkillsCount = computed(() => enabledSkills.value.length)
const disabledSkillsCount = computed(() => disabledSkills.value.length)

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
  activeTab.value = 'details'
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
        if (filterMode.value === 'disabled') {
          return installed ? !skill.enabled : false
        }
        if (filterMode.value === 'conflict') {
          return isConflictSkill(skill)
        }
        if (filterMode.value === 'mcp' || filterMode.value === 'plugins') {
          return false
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

const selectSkill = (skill: SkillSummary) => {
  selectedSkillKey.value = skillIdentity(skill)
}

const refresh = () => {
  sidebarMenuOpen.value = false
  notice.value = ''
  void Promise.all([loadGroupedSkills(), loadStatus(), loadRepos()])
}

const toggleSidebarMenu = () => {
  sidebarMenuOpen.value = !sidebarMenuOpen.value
  if (!sidebarMenuOpen.value) {
    sidebarViewsOpen.value = false
  }
}

const toggleSidebarViews = () => {
  sidebarViewsOpen.value = !sidebarViewsOpen.value
}

const toggleSection = (section: ViewFilterOption) => {
  visibleSections[section] = !visibleSections[section]
  if (visibleSections[section] && (section === 'enabled' || section === 'disabled')) {
    collapsed[section] = false
  }
  sidebarMenuOpen.value = false
  sidebarViewsOpen.value = false
}

const showEnabledView = () => {
  visibleSections.enabled = true
  collapsed.enabled = false
  sidebarMenuOpen.value = false
  sidebarViewsOpen.value = false
}

const showDisabledView = () => {
  visibleSections.disabled = true
  collapsed.disabled = false
  sidebarMenuOpen.value = false
  sidebarViewsOpen.value = false
}

const setFilterMode = (mode: FilterMode) => {
  filterMode.value = mode
  if (mode !== 'all') {
    sidebarViewsOpen.value = false
  }
}

const toggleSkillMenu = (skill: SkillSummary) => {
  const key = skillIdentity(skill)
  selectSkill(skill)
  openSkillMenuKey.value = openSkillMenuKey.value === key ? '' : key
}

const showRecommendedMenuNotice = (skill: SkillSummary) => {
  selectSkill(skill)
  notice.value = 'Add to repo skills 将在后续实现'
  openSkillMenuKey.value = ''
}

const cycleFilterMode = () => {
  const modes: FilterMode[] = ['all', 'enabled', 'disabled', 'conflict']
  const nextIndex = (modes.indexOf(filterMode.value) + 1) % modes.length
  filterMode.value = modes[nextIndex]
}

const isConflictSkill = (skill: SkillSummary) => conflictDirectories.value.has(skill.directory.toLowerCase())

const handleOpenFolder = async () => {
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
  selectSkill(skill)
  if (!hasHealthyLinks.value) {
    skillsError.value = t('components.skill.install.linksRequired')
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
  openSkillMenuKey.value = ''
  selectSkill(skill)
  pendingUninstallSkill.value = skill
}

const handleUninstallAgent = async (skill: SkillSummary, agent: string) => {
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
  backupModalOpen.value = true
}

const openRepoModal = () => {
  sidebarMenuOpen.value = false
  sidebarViewsOpen.value = false
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

const disableAutoUpdateForAll = () => {
  sidebarMenuOpen.value = false
  notice.value = t('components.skill.actions.autoUpdateComingSoon')
}

const setAllEnabled = async (enabled: boolean) => {
  sidebarMenuOpen.value = false
  sidebarViewsOpen.value = false
  const skills = installedGroups.value.flatMap((group) => group.skills).filter((skill) => skill.enabled !== enabled)
  if (!skills.length) {
    notice.value = enabled ? t('components.skill.menu.allSkillsAlreadyEnabled') : t('components.skill.menu.allInstalledSkillsAlreadyDisabled')
    return
  }
  try {
    for (const skill of skills) {
      await toggleSkill(skill.directory, enabled)
    }
    notice.value = enabled ? t('components.skill.menu.enabledAllSkills') : t('components.skill.menu.disabledAllInstalledSkills')
    await Promise.all([loadGroupedSkills(), loadStatus()])
  } catch (error) {
    console.error('failed to toggle all skills', error)
    skillsError.value = t('components.skill.actions.toggleError')
  }
}

const openBisect = () => {
  sidebarMenuOpen.value = false
  sidebarViewsOpen.value = false
  notice.value = t('components.skill.menu.skillBisectComingSoon')
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
  gap: 8px;
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
  background: color-mix(in srgb, var(--mac-surface) 92%, transparent);
  box-shadow: none;
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
  grid-template-columns: minmax(320px, 360px) minmax(0, 1fr);
  gap: 0;
  min-height: 740px;
}

.skill-sidebar,
.skill-detail {
  border-radius: 0;
  overflow: hidden;
}

.skill-detail {
  border-left: 0;
}

.sidebar-header {
  padding: 8px 16px 6px;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 94%, transparent);
}

.sidebar-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.sidebar-title {
  font-size: 0.76rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--mac-text);
}

.sidebar-topbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 96%, transparent);
  box-shadow: 0 20px 36px rgba(0, 0, 0, 0.22);
}

.menu-popover {
  top: calc(100% + 8px);
}

.sidebar-menu-popover {
  min-width: 260px;
}

.submenu-popover {
  top: 0;
  left: calc(100% + 8px);
  right: auto;
  min-width: 210px;
}

.menu-popover button,
.row-menu button {
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  text-align: left;
  font-size: 0.86rem;
}

.menu-item-with-submenu {
  position: relative;
}

.menu-item-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.submenu-caret {
  color: var(--mac-text-secondary);
}

.menu-check {
  display: inline-flex;
  width: 16px;
  justify-content: center;
  color: var(--mac-text);
}

.menu-divider {
  height: 1px;
  margin: 6px 0;
  background: color-mix(in srgb, var(--mac-border) 72%, transparent);
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
  margin-top: 4px;
  padding: 6px 0 8px;
  border-radius: 0;
  background: transparent;
  border: 0;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
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
  font-size: 0.92rem;
}

.ghost-inline {
  border: 0;
  background: transparent;
  color: var(--mac-text-secondary);
  font-size: 0.76rem;
}

.ghost-inline.filter {
  padding: 0 0 0 8px;
  border-radius: 0;
  background: transparent;
  border-left: 1px solid color-mix(in srgb, var(--mac-border) 64%, transparent);
}

.sidebar-body {
  padding: 0 0 10px;
  max-height: 980px;
  overflow: auto;
  background: transparent;
}

.skill-group-panel {
  padding: 0;
}

.skill-group-panel + .skill-group-panel {
  margin-top: 0;
  border-top: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
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
  min-height: 22px;
  height: 22px;
  padding: 0 12px 0 1px;
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--mac-text-secondary);
}

.group-toggle-main {
  display: flex;
  align-items: center;
  gap: 0;
  flex: 1;
  min-width: 0;
}

.group-chevron {
  display: inline-flex;
  width: 16px;
  justify-content: center;
  color: var(--mac-text-secondary);
  font-size: 11px;
  line-height: 1;
  margin: 0 2px;
  transition: transform 0.16s ease;
}

.group-chevron svg {
  width: 10px;
  height: 10px;
  display: block;
}

.group-chevron.collapsed {
  transform: rotate(-90deg);
}

.group-toggle:hover,
.subgroup-toggle:hover {
  background: transparent;
}

.group-badge {
  min-width: 22px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid color-mix(in srgb, var(--mac-border) 52%, transparent);
  font-size: 11px;
  font-weight: 500;
  line-height: 1;
  text-align: center;
  color: var(--mac-text-secondary);
  margin-right: 12px;
}

.group-body {
  margin-top: 0;
}

.subgroup-panel {
  margin-top: 0;
  padding: 0;
  border-radius: 0;
  background: transparent;
  border: 0;
}

.subgroup-panel.recommended {
  padding: 0;
}

.subgroup-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.subgroup-caret,
.subgroup-label {
  color: var(--mac-text-secondary);
  font-size: 0.7rem;
}

.subgroup-toggle,
.subgroup-label {
  padding: 8px 0 4px;
}

.subgroup-label {
  text-transform: none;
  letter-spacing: 0;
  font-size: 0.76rem;
}

.subgroup-list {
  margin-top: 0;
  display: flex;
  flex-direction: column;
  gap: 0;
}

.skill-row-wrap {
  position: relative;
}

.skill-row-wrap + .skill-row-wrap {
  margin-top: 0;
}

.row-menu {
  top: calc(100% + 6px);
}

.skill-empty {
  padding: 14px 12px;
  text-align: center;
  color: var(--mac-text-secondary);
  font-size: 0.78rem;
  line-height: 1.35;
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
  gap: 8px;
  color: var(--mac-text-secondary);
}

.detail-header {
  padding: 14px 20px 10px;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: transparent;
}

.detail-header-main {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.detail-icon {
  width: 56px;
  height: 56px;
  border-radius: 2px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.4rem;
  font-weight: 700;
  background: color-mix(in srgb, var(--mac-surface) 68%, rgba(10, 132, 255, 0.08));
  border: 1px solid color-mix(in srgb, var(--mac-border) 58%, transparent);
}

.detail-icon.conflict {
  background: color-mix(in srgb, var(--mac-surface) 68%, rgba(239, 68, 68, 0.12));
}

.detail-copy {
  min-width: 0;
  flex: 1;
}

.detail-copy h2 {
  margin: 0 0 4px;
  font-size: clamp(1.12rem, 1.35vw, 1.34rem);
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.detail-meta-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
  margin-bottom: 4px;
  font-size: 0.76rem;
}

.detail-publisher {
  font-weight: 700;
  color: var(--mac-accent);
}

.detail-meta-sep,
.detail-meta-status {
  color: var(--mac-text-secondary);
}

.detail-meta-link {
  border: 0;
  background: transparent;
  color: var(--mac-text-secondary);
  font-size: 0.8rem;
  padding: 0;
}

.detail-meta-link.action {
  padding: 0 4px;
  font-size: 0.78rem;
  line-height: 30px;
}

.detail-meta-link:hover:not(:disabled) {
  color: var(--mac-text);
  text-decoration: underline;
}

.detail-tagline {
  margin: 0 0 6px;
  font-size: 0.8rem;
  line-height: 1.4;
  color: var(--mac-text-secondary);
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 0;
  align-items: center;
}

.detail-actions .agent-install-btn,
.detail-actions .btn-secondary,
.detail-inline-toggle {
  height: 28px;
}

.detail-actions .agent-install-btn,
.detail-actions .btn-secondary,
.detail-inline-toggle {
  opacity: 0.82;
  font-size: 0.72rem;
}

.detail-actions .btn-secondary {
  padding: 0 6px;
  border-color: transparent;
}

.detail-actions .btn-secondary:hover:not(:disabled),
.detail-actions .agent-install-btn:hover:not(:disabled),
.detail-inline-toggle:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--mac-border) 42%, transparent);
}

.detail-tabs {
  display: flex;
  gap: 0;
  padding: 0 20px;
  margin: 0;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  min-height: 36px;
  align-items: stretch;
}

.detail-tabs button {
  padding: 0 10px;
  border-radius: 0;
  border: 1px solid transparent;
  background: transparent;
  color: var(--mac-text-secondary);
  font-size: 11px;
  font-weight: 500;
  line-height: 36px;
  text-transform: uppercase;
}

.detail-tabs button.active {
  color: var(--mac-text);
  border-bottom: 1px solid var(--mac-accent);
}

.detail-panel,
.timeline-item {
  border-radius: 0;
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  background: transparent;
}

.detail-panel {
  margin: 20px 24px 24px;
  padding: 18px 0 0;
  min-height: 440px;
  border-left: 0;
  border-right: 0;
  border-bottom: 0;
}

.detail-layout-vscode {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 18px;
  padding: 12px 20px 20px;
}

.detail-content-pane,
.detail-sidebar-vscode {
  min-width: 0;
}

.detail-content-pane {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.detail-sidebar-vscode {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-hero-block {
  padding: 2px 0 14px;
  border-bottom: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
}

.detail-hero-copy {
  margin: 8px 0 0;
  max-width: 760px;
  font-size: 0.92rem;
  line-height: 1.6;
  color: var(--mac-text);
}

.detail-block,
.side-panel {
  padding-top: 6px;
  border-top: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
}

.detail-block {
  padding-top: 14px;
}

.detail-block-label,
.side-panel h3 {
  margin: 0 0 8px;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--mac-text-secondary);
}

.detail-block-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}

.detail-block-kicker {
  font-size: 0.7rem;
  color: var(--mac-text-secondary);
  white-space: nowrap;
}

.install-command-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 2px 0 10px;
}

.install-command {
  display: block;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  background: color-mix(in srgb, var(--mac-surface) 50%, transparent);
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.8rem;
  line-height: 1.5;
  overflow: auto;
}

.detail-readme-shell {
  padding: 2px 0 6px;
}

.detail-prose {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.64;
  color: var(--mac-text);
  font-family: inherit;
  padding: 2px 0 4px;
  font-size: 0.88rem;
  max-width: 760px;
}

.side-facts {
  display: flex;
  flex-direction: column;
  gap: 0;
  margin: 0;
}

.side-facts > div {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 6px 0;
  border-top: 1px solid color-mix(in srgb, var(--mac-border) 44%, transparent);
}

.side-facts > div:first-child {
  border-top: 0;
  padding-top: 0;
}

.side-facts dt {
  font-size: 0.7rem;
  color: var(--mac-text-secondary);
  margin: 0;
  line-height: 1.45;
}

.side-facts dd {
  margin: 0;
  word-break: break-word;
  line-height: 1.42;
  text-align: left;
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
  border-radius: 0;
  border: 0;
  border-top: 1px solid color-mix(in srgb, var(--mac-border) 72%, transparent);
  background: transparent;
  color: var(--mac-text);
  padding: 14px 0;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.82rem;
  line-height: 1.5;
  outline: none;
}

.detail-empty h2 {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--mac-text);
}

.detail-empty p {
  margin: 0;
  font-size: 0.92rem;
  color: var(--mac-text-secondary);
}

.detail-inline-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 8px;
  border-radius: 2px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--mac-text-secondary);
  font-size: 0.72rem;
  font-weight: 500;
  cursor: not-allowed;
  opacity: 0.6;
}

.detail-inline-check {
  width: 11px;
  height: 11px;
  border-radius: 2px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 56%, transparent);
  display: inline-flex;
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
  padding: 14px 0;
  border-left: 0;
  border-right: 0;
  border-bottom: 0;
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
  border-radius: 4px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  background: transparent;
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
  border-radius: 2px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 68%, transparent);
  padding: 10px 14px;
  font-weight: 700;
}

.btn-primary {
  background: color-mix(in srgb, var(--mac-accent) 76%, transparent);
  color: white;
  border-color: color-mix(in srgb, var(--mac-accent) 48%, transparent);
}

.btn-primary:disabled,
.btn-secondary:disabled {
  opacity: 0.55;
}

.btn-secondary {
  background: transparent;
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
  border: 1px solid color-mix(in srgb, var(--mac-border) 34%, transparent);
  border-radius: 4px;
  background: transparent;
  color: var(--mac-text-secondary);
}

.ghost-icon.sm {
  width: 34px;
  height: 34px;
  border-radius: 8px;
}

.ghost-icon:hover {
  background: color-mix(in srgb, var(--mac-surface) 56%, transparent);
  color: var(--mac-text);
}

.ghost-icon.active {
  background: color-mix(in srgb, var(--mac-surface-strong) 84%, transparent);
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

.agent-install-btn {
  width: 34px;
  height: 28px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border-radius: 2px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--mac-text-secondary);
  font-size: 0.74rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

.agent-install-btn:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--mac-border) 42%, transparent);
  color: var(--mac-text);
}

.agent-install-btn.installed {
  background: transparent;
  border-color: transparent;
  color: var(--mac-accent);
}

.agent-install-btn.installed:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--mac-border) 42%, transparent);
  color: var(--mac-accent);
}

.agent-install-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.agent-install-icon {
  width: 14px;
  height: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.agent-install-icon :deep(svg) {
  width: 14px;
  height: 14px;
  display: block;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.auto-update-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 14px;
  border-radius: 4px;
  border: 1px solid color-mix(in srgb, var(--mac-border) 60%, transparent);
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--mac-text-secondary);
  cursor: not-allowed;
  opacity: 0.6;
  user-select: none;
}

.auto-update-check {
  accent-color: var(--mac-accent);
  cursor: not-allowed;
}

.side-tag {
  display: inline-flex;
  align-items: center;
  min-height: auto;
  padding: 0;
  border: 0;
  color: var(--mac-text-secondary);
  font-size: 0.76rem;
  line-height: 1.4;
}

.side-resource-list {
  display: flex;
  flex-direction: column;
}

.side-resource-link {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 0;
  border: 0;
  background: transparent;
  color: var(--mac-text-secondary);
  text-align: left;
  font-size: 0.78rem;
  cursor: pointer;
}

.side-resource-link:disabled {
  color: var(--mac-text-secondary);
  cursor: default;
  opacity: 0.72;
}

.side-resource-link:hover:not(:disabled) {
  color: var(--mac-accent);
  text-decoration: underline;
}

.side-resource-link svg {
  width: 14px;
  height: 14px;
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
