<template>
  <AppLayout>
    <TablePageLayout class="upstream-accounts-page">
      <template #filters>
        <div class="accounts-shell">
          <section class="accounts-topbar">
            <div class="stats-strip">
              <button
                v-for="card in statCards"
                :key="card.key"
                type="button"
                :class="['stat-card', `stat-card-${card.tone}`]"
                :aria-label="t('admin.upstreamAccounts.statDetailsAria', { label: card.label, count: card.value })"
                :data-test="`upstream-stat-card-${card.key}`"
                @click="openStatDetailsDialog(card.key)"
              >
                <span v-if="card.key === 'update' && summary.update_count > 0" class="stat-alert-dot"></span>
                <span class="stat-icon">
                  <Icon :name="card.icon" size="md" :stroke-width="2" />
                </span>
                <span class="stat-copy">
                  <strong>{{ card.value }}</strong>
                  <span>{{ card.label }}</span>
                </span>
              </button>
            </div>
            <div class="accounts-actions">
              <div class="provider-summary">
                <span>{{ t('admin.upstreamAccounts.syncProviders') }}</span>
                <strong>{{ syncProviderLabel }}</strong>
                <code v-if="syncProviderCode">{{ syncProviderCode }}</code>
              </div>
              <div class="accounts-button-group">
                <button
                  type="button"
                  class="ui-button ui-button-icon accounts-action-secondary"
                  :disabled="loading || syncing"
                  :title="t('common.refresh')"
                  @click="reload"
                >
                  <Icon name="refresh" size="md" :stroke-width="2" :class="loading ? 'animate-spin' : ''" />
                </button>
                <button
                  type="button"
                  class="ui-button ui-button-primary accounts-action-primary"
                  :disabled="loading || syncing"
                  @click="() => openCreateAccountDialog()"
                >
                  <Icon name="plus" size="sm" :stroke-width="2" />
                  {{ t('admin.accounts.createAccount') }}
                </button>
                <button
                  type="button"
                  class="ui-button ui-button-primary accounts-action-primary"
                  :disabled="loading || syncing || !canSync"
                  @click="openSyncConfirmDialog"
                >
                  <Icon name="sync" size="sm" :stroke-width="2" :class="syncing ? 'animate-spin' : ''" />
                  {{ t('admin.upstreamAccounts.syncNow') }}
                </button>
                <button
                  type="button"
                  class="ui-button accounts-action-test"
                  :disabled="loading || syncing || batchTesting || batchTestAccountIds.length === 0"
                  data-test="batch-test-accounts"
                  @click="openBatchTestConfigDialog"
                >
                  <Icon name="play" size="sm" :stroke-width="2" :class="batchTesting ? 'animate-pulse' : ''" />
                  {{ t('admin.upstreamAccounts.testAllConnections') }}
                </button>
                <button type="button" class="ui-button accounts-action-secondary" @click="openSyncLogsDialog">
                  <Icon name="document" size="sm" :stroke-width="2" />
                  {{ t('admin.upstreamAccounts.openSyncLogs') }}
                </button>
              </div>
            </div>
          </section>

          <UpstreamAccountRateGuardPanel
            :enabled="rateGuardForm.enabled"
            :interval-seconds="rateGuardForm.interval_seconds"
            :config="rateGuardConfig"
            :last-run-text="rateGuardLastRunText"
            :daily-runs-text="rateGuardDailyRunsText"
            :ignored-count="rateGuardIgnoredCount"
            :ignored-summary-text="rateGuardIgnoredSummaryText"
            :ignored-input-invalid="rateGuardIgnoredInputInvalid"
            :loading="loadingRateGuardConfig"
            :saving="savingRateGuardConfig"
            :running="runningRateGuardNow"
            :unhandled-sync-log-count="unhandledSyncLogEntries.length"
            :automation-target="rateGuardAutomationTarget"
            @update:enabled="rateGuardForm.enabled = $event"
            @update:interval-seconds="rateGuardForm.interval_seconds = $event"
            @manage-ignored="openRateGuardIgnoredDialog"
            @open-logs="openSyncLogsDialog"
            @save="saveRateGuardConfig"
            @run="runRateGuardNow"
          />

          <section class="filter-row" :class="{ 'filters-expanded': showAdvancedFilters }">
            <div class="filter-sticky-row">
              <div class="search-wrap">
                <Icon name="search" size="sm" :stroke-width="2" />
                <input
                  v-model.trim="searchQuery"
                  type="search"
                  class="ui-input filter-search"
                  :placeholder="t('admin.upstreamAccounts.searchPlaceholder')"
                />
              </div>
              <button
                type="button"
                class="filter-toggle-button"
                :aria-expanded="showAdvancedFilters"
                @click="showAdvancedFilters = !showAdvancedFilters"
              >
                <Icon name="filter" size="sm" :stroke-width="2" />
                {{ t('admin.upstreamAccounts.mobileFilterToggle') }}
                <strong v-if="activeFilterCount">{{ activeFilterCount }}</strong>
                <Icon :name="showAdvancedFilters ? 'chevronUp' : 'chevronDown'" size="sm" :stroke-width="2" />
              </button>
              <div class="filtered-count">
                <span>{{ t('admin.upstreamAccounts.filteredCount') }}</span>
                <strong>{{ filteredItems.length }}</strong>
              </div>
            </div>
            <div class="filter-controls" :class="{ 'is-open': showAdvancedFilters }">
              <Select
                v-model="platformFilter"
                class="filter-select"
                :options="platformFilterOptions"
              />
              <Select
                v-model="providerFilter"
                class="filter-select"
                :options="providerOptions"
              />
              <Select
                v-model="sourceFilter"
                class="filter-select"
                :options="sourceOptions"
              />
              <Select
                v-model="groupFilter"
                class="filter-select"
                :options="groupOptions"
              />
            </div>
          </section>

          <nav class="quick-tags" aria-label="quick filters">
            <button
              v-for="option in quickFilterOptions"
              :key="option.key"
              type="button"
              :class="['quick-tag', { active: activeQuickFilter === option.key }, option.tone ? `quick-tag-${option.tone}` : '']"
              @click="activeQuickFilter = option.key"
            >
              <span>{{ option.label }}</span>
              <strong>{{ option.count }}</strong>
            </button>
          </nav>
        </div>
      </template>

      <template #table>
        <div class="accounts-table-content">
          <div v-if="warnings.length" class="warning-banner">
            <Icon name="exclamationTriangle" size="sm" :stroke-width="2" />
            <div>
              <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
            </div>
          </div>

          <section class="accounts-table-card">
            <DataTable
              :columns="columns"
              :data="tableItems"
              :loading="loading"
              :row-class="accountRowClass"
              :estimate-row-height="78"
            >
              <template #cell-source="{ row }">
                <div class="source-cell">
                  <span :class="['source-line', sourceToneClass(row)]"></span>
                  <div class="source-main">
                    <div class="source-title">
                      <Icon v-if="row.rate_violation" name="exclamationTriangle" size="sm" :stroke-width="2" class="source-warning-icon" />
                      <span :class="['table-tag', providerToneClass(row.provider_slug, 'tag')]">
                        {{ row.provider_name || row.provider_slug }}
                      </span>
                      <span
                        v-if="row.provider_fetch_error"
                        class="table-tag tag-local-snapshot"
                        :title="row.provider_fetch_error"
                      >
                        {{ t('admin.upstreamAccounts.localSnapshotTag') }}
                      </span>
                    </div>
                    <code class="source-id">{{ row.provider_slug || '-' }}</code>
                  </div>
                  <a
                    v-if="row.provider_base_url"
                    :href="row.provider_base_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="table-tag home-tag"
                    :title="t('admin.upstreamProviders.openHomepage')"
                  >
                    <Icon name="home" size="xs" :stroke-width="2" />
                    {{ t('admin.upstreamProviders.homepageShort') }}
                  </a>
                </div>
              </template>

              <template #cell-upstream_key_name="{ row }">
                <div class="two-line-cell">
                  <span :class="['main-text', matchedAccountPlatformTextClass(row)]">{{ row.upstream_key_name }}</span>
                  <span class="sub-text">{{ row.upstream_group_name || '-' }}</span>
                </div>
              </template>

              <template #cell-local_account_name="{ row }">
                <div class="two-line-cell">
                  <span :class="['main-text', matchedAccountPlatformTextClass(row)]">{{ row.local_account_name || row.matched_account_name || '-' }}</span>
                  <span v-if="row.matched_account_id" :class="['table-tag', 'tag-account', 'account-id-tag', matchedAccountPlatformTagClass(row)]">
                    #{{ row.matched_account_id }} {{ row.matched_account_name || row.local_account_name }}
                  </span>
                  <div v-else-if="row.conflict_accounts?.length" class="tag-list">
                    <span
                      v-for="account in row.conflict_accounts"
                      :key="`${row.provider_slug}-${row.upstream_key_name}-conflict-${account.id}`"
                      class="group-chip group-chip-warning"
                      :title="conflictAccountTitle(account)"
                    >
                      #{{ account.id }} {{ account.name }}
                      <span v-if="account.bound_groups?.length" class="font-mono">
                        {{ conflictAccountRates(account) }}
                      </span>
                    </span>
                  </div>
                  <span v-else-if="row.conflict_account_ids?.length" class="sub-text sub-text-warning">
                    {{ t('admin.upstreamAccounts.conflictIds', { ids: row.conflict_account_ids.join(', ') }) }}
                  </span>
                  <span v-else class="table-tag tag-account account-id-tag">-</span>
                  <span v-if="isRateGuardIgnored(row)" class="table-tag tag-rate-guard-ignore">
                    <Icon name="shield" size="xs" :stroke-width="2" />
                    {{ t('admin.upstreamAccounts.rateGuardIgnoredAccountTag') }}
                  </span>
                </div>
              </template>

              <template #cell-priority="{ row, value }">
                <form v-if="isEditingPriority(row)" class="priority-edit-form" @submit.prevent="savePriority(row)">
                  <input
                    ref="priorityInputRef"
                    v-model.number="priorityDraft"
                    type="number"
                    min="0"
                    step="1"
                    class="priority-input"
                    :aria-label="t('admin.upstreamAccounts.editPriority')"
                    :disabled="savingPriorityAccountId === row.matched_account_id"
                    @blur="savePriority(row)"
                    @keydown.esc.prevent="cancelPriorityEdit"
                  />
                </form>
                <button
                  v-else-if="isPriorityEditable(row)"
                  type="button"
                  class="priority-pill priority-pill-button"
                  :disabled="savingPriorityAccountId === row.matched_account_id"
                  :title="t('admin.upstreamAccounts.editPriority')"
                  @click="startPriorityEdit(row)"
                >
                  <Icon v-if="savingPriorityAccountId === row.matched_account_id" name="cog" size="xs" class="animate-spin" />
                  <span>{{ Number.isFinite(Number(value)) ? value : '-' }}</span>
                </button>
                <span v-else class="dash">-</span>
              </template>

              <template #cell-upstream_rate_multiplier="{ value }">
                <div class="rate-cell">
                  <span :class="['rate-value', rateToneClass(value)]">{{ formatRate(value) }}</span>
                  <span :class="['rate-bar', rateToneClass(value)]">
                    <span :style="{ width: rateProgressWidth(value) }"></span>
                  </span>
                </div>
              </template>

              <template #cell-local_group_name="{ row }">
                <div v-if="row.bound_groups?.length" class="tag-list group-list">
                  <span
                    v-for="group in row.bound_groups"
                    :key="`${row.provider_slug}-${row.upstream_key_name}-${group.id}`"
                    :class="['group-chip', matchedAccountPlatformTagClass(row)]"
                    :title="`${group.name} ${formatRate(group.rate_multiplier)}`"
                  >
                    {{ group.name }}
                    <span class="font-mono">{{ formatRate(group.rate_multiplier) }}</span>
                  </span>
                </div>
                <div v-else class="two-line-cell">
                  <span class="dash">{{ row.local_group_name || '-' }}</span>
                  <span v-if="row.local_rate_multiplier !== undefined" class="sub-text">
                    {{ formatRate(row.local_rate_multiplier) }}
                  </span>
                </div>
              </template>

              <template #cell-balance="{ row }">
                <div class="balance-cell">
                  <span v-if="getProviderBalance(row.provider_slug) !== null" class="balance-value">
                    ${{ formatMoney(getProviderBalance(row.provider_slug) || 0) }}
                  </span>
                  <span v-else class="dash">-</span>
                  <button
                    v-if="getProviderBalance(row.provider_slug) !== null"
                    type="button"
                    class="trend-btn"
                    title="查看余额趋势"
                    @click="openTrendModal(row.provider_slug, row.provider_name)"
                  >
                    <Icon name="chart" size="xs" :stroke-width="2" />
                  </button>
                </div>
              </template>

              <template #cell-today_consumption="{ row }">
                <div class="balance-cell">
                  <span v-if="getProviderConsumption(row.provider_slug) !== null" class="consumption-value">
                    ${{ formatMoney(getProviderConsumption(row.provider_slug) || 0) }}
                  </span>
                  <span v-else class="dash">-</span>
                  <button
                    v-if="getProviderConsumption(row.provider_slug) !== null"
                    type="button"
                    class="trend-btn"
                    title="查看消费趋势"
                    @click="openTrendModal(row.provider_slug, row.provider_name)"
                  >
                    <Icon name="trendingUp" size="xs" :stroke-width="2" />
                  </button>
                </div>
              </template>

              <template #cell-status="{ row }">
                <div v-if="getMatchedAccount(row)" class="status-cell">
                  <AccountStatusIndicator
                    :account="getMatchedAccount(row)!"
                    @show-temp-unsched="handleShowTempUnsched"
                  />
                </div>
                <span v-else class="dash">-</span>
              </template>

              <template #cell-schedulable="{ row }">
                <div v-if="getMatchedAccount(row) && !isProviderDisabled(row)" class="status-cell">
                  <button
                    type="button"
                    class="schedulable-toggle"
                    :disabled="isSchedulableToggleDisabled(row)"
                    :class="[getMatchedAccount(row)!.schedulable ? 'schedulable-on' : 'schedulable-off']"
                    :title="schedulableToggleTitle(row)"
                    @click="handleToggleSchedulable(getMatchedAccount(row)!, row)"
                  >
                    <span class="schedulable-track">
                      <span
                        class="schedulable-thumb"
                        :class="[getMatchedAccount(row)!.schedulable ? 'schedulable-thumb-on' : 'schedulable-thumb-off']"
                      />
                    </span>
                  </button>
                </div>
                <span v-else class="dash">-</span>
              </template>

              <template #cell-test_status="{ row }">
                <div class="test-status-cell">
                  <span
                    v-if="accountTestStatusById[row.matched_account_id || 0]"
                    :class="[
                      'test-status-pill',
                      `test-status-${accountTestStatusById[row.matched_account_id || 0]}`
                    ]"
                  >
                    <Icon
                      :name="accountTestStatusById[row.matched_account_id || 0] === 'success'
                        ? 'checkCircle'
                        : accountTestStatusById[row.matched_account_id || 0] === 'failed'
                          ? 'xCircle'
                          : 'clock'"
                      size="sm"
                      :stroke-width="2"
                    />
                    {{ accountTestStatusLabel(accountTestStatusById[row.matched_account_id || 0]) }}
                  </span>
                  <span v-else class="dash">-</span>
                </div>
              </template>

              <template #cell-last_tested_at="{ row }">
                <span v-if="getMatchedAccount(row)?.last_tested_at" class="test-time">
                  {{ formatDateTime(getMatchedAccount(row)!.last_tested_at!) }}
                </span>
                <span v-else class="dash">-</span>
              </template>

              <template #cell-actions="{ row }">
                <div v-if="row.matched_account_id" class="action-cell">
                  <button
                    type="button"
                    class="text-action text-action-muted"
                    :disabled="savingAccountGroupId === row.matched_account_id || testingAccountId === row.matched_account_id"
                    @click="openAccountEditDialog(row)"
                  >
                    <Icon name="edit" size="sm" :stroke-width="2" />
                    {{ t('common.edit') }}
                  </button>
                  <button
                    type="button"
                    :class="['text-action', row.rate_violation ? 'text-action-danger' : '']"
                    :disabled="savingAccountGroupId === row.matched_account_id || testingAccountId === row.matched_account_id"
                    @click="openAccountGroupDialog(row)"
                  >
                    <Icon :name="row.rate_violation ? 'exclamationTriangle' : 'edit'" size="sm" :stroke-width="2" :class="savingAccountGroupId === row.matched_account_id ? 'animate-spin' : ''" />
                    {{ row.rate_violation ? '\u5904\u7406\u98ce\u9669' : t('admin.upstreamAccounts.editBoundGroups') }}
                  </button>
                  <button
                    type="button"
                    :class="['text-action', isRateGuardIgnored(row) ? 'text-action-primary' : 'text-action-muted']"
                    :disabled="savingAccountGroupId === row.matched_account_id || testingAccountId === row.matched_account_id || togglingRateGuardIgnoreId === row.matched_account_id"
                    :data-test="`rate-guard-ignore-toggle-${row.matched_account_id}`"
                    @click="toggleRateGuardIgnored(row)"
                  >
                    <Icon name="shield" size="sm" :stroke-width="2" :class="togglingRateGuardIgnoreId === row.matched_account_id ? 'animate-spin' : ''" />
                    {{ isRateGuardIgnored(row) ? t('admin.upstreamAccounts.rateGuardUnignoreAccount') : t('admin.upstreamAccounts.rateGuardIgnoreAccount') }}
                  </button>
                  <button
                    type="button"
                    class="text-action text-action-muted"
                    :disabled="testingAccountId === row.matched_account_id || savingAccountGroupId === row.matched_account_id"
                    @click="openAccountTestDialog(row)"
                  >
                    <Icon name="play" size="sm" :stroke-width="2" :class="testingAccountId === row.matched_account_id ? 'animate-spin' : ''" />
                    {{ t('admin.upstreamAccounts.testConnection') }}
                  </button>
                  <button
                    type="button"
                    class="text-action text-action-danger"
                    :disabled="savingAccountGroupId === row.matched_account_id || testingAccountId === row.matched_account_id"
                    @click="openAccountDeleteDialog(row)"
                  >
                    <Icon name="trash" size="sm" :stroke-width="2" />
                    {{ t('common.delete') }}
                  </button>
                </div>
                <div v-else class="action-cell">
                  <button
                    type="button"
                    class="text-action text-action-primary"
                    :data-test="`create-local-account-${slugForDataTest(row.provider_slug)}-${slugForDataTest(row.upstream_key_name)}`"
                    @click="openCreateAccountDialog(row)"
                  >
                    <Icon name="plus" size="sm" :stroke-width="2" />
                    {{ t('admin.upstreamAccounts.createLocalAccount') }}
                  </button>
                </div>
              </template>

              <template #empty>
                <EmptyState
                  :title="emptyTitle"
                  :description="emptyDescription"
                  :action-text="t('common.refresh')"
                  @action="reload"
                />
              </template>
            </DataTable>
          </section>

        </div>

        <div
          v-if="activeStatDetailsKey"
          class="stat-details-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
          data-test="stat-details-dialog"
          @click.self="closeStatDetailsDialog"
        >
          <div class="stat-details-modal">
            <div class="stat-details-header">
              <div>
                <h3>{{ activeStatDetailsTitle }}</h3>
                <p>{{ activeStatDetailsDescription }}</p>
              </div>
              <button type="button" class="modal-close-button" :aria-label="t('common.close')" @click="closeStatDetailsDialog">
                <Icon name="x" size="md" :stroke-width="2" />
              </button>
            </div>
            <div class="stat-details-summary">
              <span class="stat-details-count">{{ t('admin.upstreamAccounts.statDetailsCount', { count: activeStatDetailsItems.length }) }}</span>
            </div>
            <div class="stat-details-body">
              <div v-if="activeStatDetailsItems.length" class="stat-details-table-wrap">
                <table class="stat-details-table">
                  <thead>
                    <tr>
                      <th>{{ t('admin.upstreamAccounts.columns.source') }}</th>
                      <th>{{ t('admin.upstreamAccounts.columns.upstreamKey') }}</th>
                      <th>{{ t('admin.upstreamAccounts.columns.localAccount') }}</th>
                      <th>{{ t('admin.upstreamAccounts.columns.action') }}</th>
                      <th>{{ t('admin.upstreamAccounts.columns.upstreamRate') }}</th>
                      <th>{{ t('admin.upstreamAccounts.columns.boundGroups') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="row in activeStatDetailsItems" :key="statDetailsRowKey(row)">
                      <td>
                        <div class="stat-details-source-cell">
                          <span :class="['table-tag', providerToneClass(row.provider_slug, 'tag')]">
                            {{ row.provider_name || row.provider_slug || '-' }}
                          </span>
                          <code>{{ row.provider_slug || '-' }}</code>
                          <span
                            v-if="row.provider_fetch_error"
                            class="table-tag tag-local-snapshot"
                            :title="row.provider_fetch_error"
                          >
                            {{ t('admin.upstreamAccounts.localSnapshotTag') }}
                          </span>
                        </div>
                      </td>
                      <td>
                        <div class="two-line-cell">
                          <span :class="['main-text', matchedAccountPlatformTextClass(row)]">{{ row.upstream_key_name || '-' }}</span>
                          <span class="sub-text">{{ row.upstream_group_name || '-' }}</span>
                        </div>
                      </td>
                      <td>
                        <div class="two-line-cell">
                          <span :class="['main-text', matchedAccountPlatformTextClass(row)]">{{ row.local_account_name || row.matched_account_name || '-' }}</span>
                          <span v-if="row.matched_account_id" :class="['table-tag', 'tag-account', 'account-id-tag', matchedAccountPlatformTagClass(row)]">
                            #{{ row.matched_account_id }} {{ row.matched_account_name || row.local_account_name }}
                          </span>
                          <div v-else-if="row.conflict_accounts?.length" class="tag-list">
                            <span
                              v-for="account in row.conflict_accounts"
                              :key="`${statDetailsRowKey(row)}-conflict-${account.id}`"
                              class="group-chip group-chip-warning"
                              :title="conflictAccountTitle(account)"
                            >
                              #{{ account.id }} {{ account.name }}
                            </span>
                          </div>
                          <span v-else-if="row.conflict_account_ids?.length" class="sub-text sub-text-warning">
                            {{ t('admin.upstreamAccounts.conflictIds', { ids: row.conflict_account_ids.join(', ') }) }}
                          </span>
                          <span v-else class="table-tag tag-account account-id-tag">-</span>
                        </div>
                      </td>
                      <td>
                        <span :class="['stat-details-action', statDetailsActionClass(row)]">
                          {{ statDetailsActionLabel(row) }}
                        </span>
                      </td>
                      <td>
                        <div class="stat-details-rate-cell">
                          <span :class="['rate-value', rateToneClass(row.upstream_rate_multiplier)]">{{ formatRate(row.upstream_rate_multiplier) }}</span>
                          <span v-if="row.rate_violation" class="group-chip group-chip-warning">{{ t('admin.upstreamAccounts.rateRisks') }}</span>
                          <span v-if="row.rate_violation && syncConfirmUnboundGroups(row) !== '-'" class="sub-text sub-text-warning">
                            {{ t('admin.upstreamAccounts.unbindGroups', { groups: syncConfirmUnboundGroups(row) }) }}
                          </span>
                        </div>
                      </td>
                      <td>
                        <div v-if="statDetailsGroupTags(row).length" class="tag-list group-list">
                          <span
                            v-for="group in statDetailsGroupTags(row)"
                            :key="group.key"
                            :class="['group-chip', matchedAccountPlatformTagClass(row), { 'group-chip-warning': group.rateViolation }]"
                          >
                            {{ group.label }}
                          </span>
                        </div>
                        <span v-else class="dash">-</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div v-else class="stat-details-empty">
                <Icon name="database" size="md" :stroke-width="2" />
                <span>{{ t('admin.upstreamAccounts.statDetailsEmpty') }}</span>
              </div>
            </div>
            <div class="stat-details-footer">
              <button type="button" class="ui-button ui-button-primary" @click="closeStatDetailsDialog">
                {{ t('common.close') }}
              </button>
            </div>
          </div>
        </div>

        <div
          v-if="showRateGuardIgnoredDialog"
          class="rate-guard-ignored-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
          data-test="rate-guard-ignored-dialog"
          @click.self="closeRateGuardIgnoredDialog"
        >
          <div class="sync-result-modal rate-guard-ignored-modal">
            <div class="sync-confirm-header">
              <div>
                <h3>{{ t('admin.upstreamAccounts.rateGuardIgnoredManageTitle') }}</h3>
                <p>{{ t('admin.upstreamAccounts.rateGuardIgnoredManageDescription') }}</p>
              </div>
              <button type="button" class="modal-close-button" :aria-label="t('common.close')" @click="closeRateGuardIgnoredDialog">
                <Icon name="x" size="md" :stroke-width="2" />
              </button>
            </div>

            <div class="rate-guard-ignored-body">
              <label class="rate-guard-ignored-input-row" for="rate-guard-ignored-accounts-input">
                <span class="control-label">{{ t('admin.upstreamAccounts.rateGuardIgnoredAccounts') }}</span>
                <input
                  id="rate-guard-ignored-accounts-input"
                  v-model.trim="rateGuardIgnoredInput"
                  type="text"
                  class="ui-input ignored-accounts-input"
                  :placeholder="t('admin.upstreamAccounts.rateGuardIgnoredAccountsPlaceholder')"
                  data-test="rate-guard-ignored-input"
                />
              </label>
              <p v-if="rateGuardIgnoredInputInvalid" class="rate-guard-ignored-error">
                {{ t('admin.upstreamAccounts.invalidRateGuardIgnoredAccounts') }}
              </p>
              <div class="rate-guard-ignored-summary-row">
                <span class="guard-ignore-summary" :class="{ 'is-empty': !rateGuardIgnoredDialogCount, 'is-invalid': rateGuardIgnoredInputInvalid }">
                  <Icon name="shield" size="xs" :stroke-width="2" />
                  <span>{{ rateGuardIgnoredDialogSummaryText }}</span>
                </span>
              </div>
              <div
                v-if="rateGuardIgnoredAccountDetails.length"
                class="ignored-account-chips ignored-account-dialog-chips"
                data-test="rate-guard-ignored-account-chips"
              >
                <span
                  v-for="account in rateGuardIgnoredAccountDetails"
                  :key="account.id"
                  class="ignored-account-chip"
                  :title="account.title"
                >
                  <span class="ignored-account-chip-name">{{ account.name }}</span>
                  <small>{{ account.meta }}</small>
                  <button
                    type="button"
                    class="ignored-account-remove"
                    :aria-label="t('admin.upstreamAccounts.rateGuardIgnoredRemoveAccount', { id: account.id })"
                    :data-test="`rate-guard-ignored-remove-${account.id}`"
                    @click="removeRateGuardIgnoredAccount(account.id)"
                  >
                    <Icon name="x" size="xs" :stroke-width="2" />
                  </button>
                </span>
              </div>
              <div v-else class="rate-guard-ignored-empty">
                {{ t('admin.upstreamAccounts.rateGuardIgnoredNone') }}
              </div>
            </div>

            <div class="sync-confirm-footer">
              <button type="button" class="ui-button" :disabled="savingRateGuardConfig" @click="closeRateGuardIgnoredDialog">
                {{ t('common.cancel') }}
              </button>
              <button
                type="button"
                class="ui-button ui-button-primary"
                :disabled="loadingRateGuardConfig || savingRateGuardConfig || rateGuardIgnoredInputInvalid"
                data-test="rate-guard-ignored-save"
                @click="saveRateGuardConfigAndCloseIgnoredDialog"
              >
                {{ t('common.save') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="showSyncLogsDialog" class="sync-logs-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" @click.self="closeSyncLogsDialog">
          <div class="sync-logs-modal">
            <div class="sync-logs-modal-header">
              <div>
                <h3>{{ t('admin.upstreamAccounts.syncLogs') }}</h3>
                <p>{{ t('admin.upstreamAccounts.latestRecords', { count: syncLogEntries.length }) }}</p>
              </div>
              <button type="button" class="modal-close-button" :aria-label="t('common.close')" @click="closeSyncLogsDialog">
                <Icon name="x" size="md" :stroke-width="2" />
              </button>
            </div>
            <div class="records-info sync-logs-modal-info">{{ t('admin.upstreamAccounts.syncLogsDescription') }}</div>
            <div v-if="syncLogEntries.length" class="records-table-wrap sync-logs-table-wrap">
              <table class="records-table">
                <thead>
                  <tr>
                    <th>{{ t('admin.upstreamAccounts.logStatus', '处理状态') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logTime') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logTriggerSource') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logAccount') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logUpstream') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logRateCompare') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logUnboundGroups') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logRemainingGroups') }}</th>
                    <th>{{ t('common.actions') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="entry in syncLogEntries" :key="entry.key" :class="['records-row', { 'records-row-handled': isSyncLogHandled(entry) }]">
                    <td>
                      <span v-if="isSyncLogHandled(entry)" class="sync-log-status sync-log-status-handled">
                        {{ t('admin.upstreamAccounts.syncLogHandled', '已处理') }}
                      </span>
                      <button
                        v-else
                        type="button"
                        class="sync-log-status sync-log-status-unhandled"
                        @click="markSyncLogHandled(entry)"
                      >
                        {{ t('admin.upstreamAccounts.syncLogUnhandled', '待处理') }}
                      </button>
                    </td>
                    <td>{{ formatDateTime(entry.created_at) }}</td>
                    <td>
                      <span :class="['trigger-chip', triggerClass(entry.trigger_source)]">
                        {{ upstreamAccountSyncTriggerSourceLabel(entry.trigger_source) }}
                      </span>
                    </td>
                    <td>
                      <div class="two-line-cell">
                        <span class="main-text">{{ entry.matched_local_account_name }}</span>
                        <code class="table-tag tag-account">#{{ entry.matched_local_account_id }}</code>
                      </div>
                    </td>
                    <td>
                      <div class="two-line-cell">
                        <span class="main-text">{{ entry.upstream_key_name }}</span>
                        <div class="tag-list">
                          <span :class="['table-tag', providerToneClass(entry.provider_slug, 'tag')]">{{ entry.provider_name || entry.provider_slug }}</span>
                          <span class="table-tag tag-gray">{{ entry.upstream_group_name }}</span>
                        </div>
                      </div>
                    </td>
                    <td>
                      <div class="rate-compare">
                        <span class="rate-compare-upstream">{{ formatRate(entry.upstream_rate_multiplier) }}</span>
                        <span>/</span>
                        <span class="rate-compare-local">{{ formatRate(entry.local_min_rate_multiplier) }}</span>
                      </div>
                    </td>
                    <td>
                      <div class="tag-list">
                        <span v-for="group in entry.unbound_group_names" :key="`${entry.key}-${group}`" class="log-chip log-chip-warning">{{ group }}</span>
                      </div>
                    </td>
                    <td>
                      <div class="tag-list">
                        <span v-if="!entry.remaining_group_ids.length" class="dash">-</span>
                        <code v-for="groupID in entry.remaining_group_ids" :key="`${entry.key}-${groupID}`" class="log-chip">#{{ groupID }}</code>
                      </div>
                    </td>
                    <td>
                      <button
                        v-if="!isSyncLogHandled(entry)"
                        type="button"
                        class="text-action sync-log-handle-button"
                        @click="markSyncLogHandled(entry)"
                      >
                        {{ t('admin.upstreamAccounts.markSyncLogHandled', '标记已处理') }}
                      </button>
                      <span v-else class="dash">-</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-if="syncLogEntries.length" class="sync-log-card-list">
              <article
                v-for="entry in syncLogEntries"
                :key="`mobile-${entry.key}`"
                :class="['sync-log-card', { 'is-handled': isSyncLogHandled(entry) }]"
              >
                <div class="sync-log-card-head">
                  <div class="sync-log-card-status">
                    <span v-if="isSyncLogHandled(entry)" class="sync-log-status sync-log-status-handled">
                      {{ t('admin.upstreamAccounts.syncLogHandled') }}
                    </span>
                    <button
                      v-else
                      type="button"
                      class="sync-log-status sync-log-status-unhandled"
                      @click="markSyncLogHandled(entry)"
                    >
                      {{ t('admin.upstreamAccounts.syncLogUnhandled') }}
                    </button>
                    <span :class="['trigger-chip', triggerClass(entry.trigger_source)]">
                      {{ upstreamAccountSyncTriggerSourceLabel(entry.trigger_source) }}
                    </span>
                  </div>
                  <time>{{ formatDateTime(entry.created_at) }}</time>
                </div>

                <div class="sync-log-card-main">
                  <div class="sync-log-card-field sync-log-card-field-wide">
                    <span>{{ t('admin.upstreamAccounts.logAccount') }}</span>
                    <strong>{{ entry.matched_local_account_name }}</strong>
                    <code class="table-tag tag-account">#{{ entry.matched_local_account_id }}</code>
                  </div>
                  <div class="sync-log-card-field sync-log-card-field-wide">
                    <span>{{ t('admin.upstreamAccounts.logUpstream') }}</span>
                    <strong>{{ entry.upstream_key_name }}</strong>
                    <div class="tag-list">
                      <span :class="['table-tag', providerToneClass(entry.provider_slug, 'tag')]">{{ entry.provider_name || entry.provider_slug }}</span>
                      <span class="table-tag tag-gray">{{ entry.upstream_group_name }}</span>
                    </div>
                  </div>
                  <div class="sync-log-card-field">
                    <span>{{ t('admin.upstreamAccounts.logRateCompare') }}</span>
                    <div class="rate-compare">
                      <span class="rate-compare-upstream">{{ formatRate(entry.upstream_rate_multiplier) }}</span>
                      <span>/</span>
                      <span class="rate-compare-local">{{ formatRate(entry.local_min_rate_multiplier) }}</span>
                    </div>
                  </div>
                  <div class="sync-log-card-field sync-log-card-field-wide">
                    <span>{{ t('admin.upstreamAccounts.logUnboundGroups') }}</span>
                    <div class="tag-list">
                      <span v-for="group in entry.unbound_group_names" :key="`mobile-${entry.key}-${group}`" class="log-chip log-chip-warning">{{ group }}</span>
                    </div>
                  </div>
                  <div class="sync-log-card-field sync-log-card-field-wide">
                    <span>{{ t('admin.upstreamAccounts.logRemainingGroups') }}</span>
                    <div class="tag-list">
                      <span v-if="!entry.remaining_group_ids.length" class="dash">-</span>
                      <code v-for="groupID in entry.remaining_group_ids" :key="`mobile-${entry.key}-${groupID}`" class="log-chip">#{{ groupID }}</code>
                    </div>
                  </div>
                </div>

                <button
                  v-if="!isSyncLogHandled(entry)"
                  type="button"
                  class="text-action sync-log-card-action"
                  @click="markSyncLogHandled(entry)"
                >
                  {{ t('admin.upstreamAccounts.markSyncLogHandled') }}
                </button>
              </article>
            </div>
            <div v-else class="records-empty">
              <Icon name="document" size="xl" :stroke-width="2" />
              <span>{{ t('admin.upstreamAccounts.noSyncLogs') }}</span>
              <button type="button" class="ui-button" :disabled="loading || syncing || !canSync" @click="openSyncConfirmDialog">
                {{ t('admin.upstreamAccounts.syncNow') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="showSyncConfirmDialog" class="sync-confirm-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" data-test="sync-confirm-dialog" @click.self="closeSyncConfirmDialog">
          <div class="sync-confirm-modal">
            <div class="sync-confirm-header">
              <div>
                <h3>{{ t('admin.upstreamAccounts.syncConfirmTitle') }}</h3>
                <p>{{ t('admin.upstreamAccounts.syncConfirmDescription') }}</p>
              </div>
              <button type="button" class="modal-close-button" :aria-label="t('common.close')" @click="closeSyncConfirmDialog">
                <Icon name="x" size="md" :stroke-width="2" />
              </button>
            </div>

            <div class="sync-confirm-summary">
              <label class="sync-confirm-option">
                <input
                  v-model="syncConfirmOptions.create_missing"
                  type="checkbox"
                  :disabled="syncCreateItems.length === 0"
                  data-test="sync-confirm-create-missing"
                />
                <span>
                  <strong>{{ t('admin.upstreamAccounts.syncConfirmCreateMissing') }}</strong>
                  <small>{{ syncCreateItems.length }}</small>
                </span>
              </label>
              <label class="sync-confirm-option">
                <input
                  v-model="syncConfirmOptions.update_existing"
                  type="checkbox"
                  :disabled="syncUpdateItems.length === 0"
                  data-test="sync-confirm-update-existing"
                />
                <span>
                  <strong>{{ t('admin.upstreamAccounts.syncConfirmUpdateExisting') }}</strong>
                  <small>{{ syncUpdateItems.length }}</small>
                </span>
              </label>
              <label class="sync-confirm-option">
                <input
                  v-model="syncConfirmOptions.apply_rate_guard"
                  type="checkbox"
                  :disabled="syncRateGuardItems.length === 0 || !syncConfirmOptions.update_existing"
                  data-test="sync-confirm-apply-rate-guard"
                />
                <span>
                  <strong>{{ t('admin.upstreamAccounts.syncConfirmApplyRateGuard') }}</strong>
                  <small>{{ syncRateGuardItems.length }}</small>
                </span>
              </label>
            </div>

            <div class="sync-confirm-body">
              <section class="sync-confirm-section">
                <div class="sync-confirm-section-title">
                  <span>{{ t('admin.upstreamAccounts.syncConfirmCreateSection') }}</span>
                  <strong>{{ syncCreateItems.length }}</strong>
                </div>
                <div v-if="syncCreateItems.length" class="sync-confirm-list">
                  <article v-for="item in syncCreateItems" :key="syncConfirmItemKey(item, 'create')" class="sync-confirm-item">
                    <div class="sync-confirm-item-main">
                      <input
                        v-model="syncConfirmSelectedItems[syncConfirmSelectionKey(item)]"
                        type="checkbox"
                        class="sync-confirm-item-checkbox"
                        :data-test="syncConfirmItemDataTest(item, 'create')"
                      />
                      <span :class="['table-tag', providerToneClass(item.provider_slug, 'tag')]">{{ item.provider_name || item.provider_slug }}</span>
                      <strong>{{ item.local_account_name }}</strong>
                      <code>{{ item.upstream_key_name }}</code>
                    </div>
                    <div class="sync-confirm-item-meta">
                      <span>{{ t('admin.upstreamAccounts.syncConfirmUpstreamGroup') }}: {{ item.upstream_group_name || '-' }}</span>
                      <span>{{ t('admin.upstreamAccounts.syncConfirmLocalGroup') }}: {{ item.local_group_name || '-' }}</span>
                      <span>{{ t('admin.upstreamAccounts.syncConfirmRate') }}: {{ formatRate(item.upstream_rate_multiplier) }}</span>
                    </div>
                  </article>
                </div>
                <div v-else class="sync-confirm-empty">{{ t('admin.upstreamAccounts.syncConfirmNoCreate') }}</div>
              </section>

              <section class="sync-confirm-section">
                <div class="sync-confirm-section-title">
                  <span>{{ t('admin.upstreamAccounts.syncConfirmUpdateSection') }}</span>
                  <strong>{{ syncUpdateItems.length }}</strong>
                </div>
                <div v-if="syncUpdateItems.length" class="sync-confirm-list">
                  <article v-for="item in syncUpdateItems" :key="syncConfirmItemKey(item, 'update')" class="sync-confirm-item">
                    <div class="sync-confirm-item-main">
                      <input
                        v-model="syncConfirmSelectedItems[syncConfirmSelectionKey(item)]"
                        type="checkbox"
                        class="sync-confirm-item-checkbox"
                        :data-test="syncConfirmItemDataTest(item, 'update')"
                      />
                      <span :class="['table-tag', providerToneClass(item.provider_slug, 'tag')]">{{ item.provider_name || item.provider_slug }}</span>
                      <strong>{{ item.matched_account_name || item.local_account_name }}</strong>
                      <code v-if="item.matched_account_id">#{{ item.matched_account_id }}</code>
                      <code>{{ item.upstream_key_name }}</code>
                    </div>
                    <div class="sync-confirm-item-meta">
                      <span>{{ t('admin.upstreamAccounts.syncConfirmUpstreamGroup') }}: {{ item.upstream_group_name || '-' }}</span>
                      <span>{{ t('admin.upstreamAccounts.syncConfirmLocalGroup') }}: {{ item.local_group_name || '-' }}</span>
                      <span v-if="item.rate_violation" class="sync-confirm-danger">{{ t('admin.upstreamAccounts.syncConfirmHasRateRisk') }}</span>
                    </div>
                    <div class="sync-confirm-detail-list">
                      <span v-for="detail in syncConfirmUpdateDetails(item)" :key="`${syncConfirmItemKey(item, 'detail')}-${detail}`" class="sync-confirm-detail-chip">
                        {{ detail }}
                      </span>
                    </div>
                  </article>
                </div>
                <div v-else class="sync-confirm-empty">{{ t('admin.upstreamAccounts.syncConfirmNoUpdate') }}</div>
              </section>

              <section class="sync-confirm-section">
                <div class="sync-confirm-section-title">
                  <span>{{ t('admin.upstreamAccounts.syncConfirmRateGuardSection') }}</span>
                  <strong>{{ syncRateGuardItems.length }}</strong>
                </div>
                <div v-if="syncRateGuardItems.length" class="sync-confirm-list">
                  <article v-for="item in syncRateGuardItems" :key="syncConfirmItemKey(item, 'rate')" class="sync-confirm-item sync-confirm-item-risk">
                    <div class="sync-confirm-item-main">
                      <span :class="['table-tag', providerToneClass(item.provider_slug, 'tag')]">{{ item.provider_name || item.provider_slug }}</span>
                      <strong>{{ item.matched_account_name || item.local_account_name }}</strong>
                      <code v-if="item.matched_account_id">#{{ item.matched_account_id }}</code>
                      <code>{{ item.upstream_key_name }}</code>
                    </div>
                    <div class="sync-confirm-item-meta">
                      <span>{{ t('admin.upstreamAccounts.syncConfirmUnbindGroups') }}: {{ syncConfirmUnboundGroups(item) }}</span>
                      <span>{{ t('admin.upstreamAccounts.syncConfirmRate') }}: {{ formatRate(item.upstream_rate_multiplier) }}</span>
                    </div>
                  </article>
                </div>
                <div v-else class="sync-confirm-empty">{{ t('admin.upstreamAccounts.syncConfirmNoRateGuard') }}</div>
              </section>
            </div>

            <div class="sync-confirm-footer">
              <button type="button" class="ui-button" :disabled="syncing" @click="closeSyncConfirmDialog">
                {{ t('common.cancel') }}
              </button>
              <button
                type="button"
                class="ui-button ui-button-primary"
                :disabled="syncing || !syncConfirmCanSubmit"
                data-test="sync-confirm-submit"
                @click="submitSyncConfirm"
              >
                <Icon name="sync" size="sm" :stroke-width="2" :class="syncing ? 'animate-spin' : ''" />
                {{ t('admin.upstreamAccounts.syncConfirmSubmit') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="showSyncResultDialog" class="sync-result-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" data-test="sync-result-dialog" @click.self="closeSyncResultDialog">
          <div class="sync-result-modal">
            <div class="sync-confirm-header">
              <div>
                <h3>{{ t('admin.upstreamAccounts.syncResultTitle') }}</h3>
                <p>{{ t('admin.upstreamAccounts.syncResultDescription') }}</p>
              </div>
              <button type="button" class="modal-close-button" :aria-label="t('common.close')" @click="closeSyncResultDialog">
                <Icon name="x" size="md" :stroke-width="2" />
              </button>
            </div>
            <div class="sync-confirm-summary">
              <div class="sync-result-stat">
                <span>{{ t('admin.upstreamAccounts.created') }}</span>
                <strong>{{ syncResultCreatedItems.length }}</strong>
              </div>
              <div class="sync-result-stat">
                <span>{{ t('admin.upstreamAccounts.updated') }}</span>
                <strong>{{ syncResultUpdatedItems.length }}</strong>
              </div>
              <div class="sync-result-stat">
                <span>{{ t('admin.upstreamAccounts.unbound') }}</span>
                <strong>{{ syncResultUnboundCount }}</strong>
              </div>
            </div>
            <div class="sync-confirm-body">
              <section class="sync-confirm-section">
                <div class="sync-confirm-section-title">
                  <span>{{ t('admin.upstreamAccounts.syncResultCreatedSection') }}</span>
                  <strong>{{ syncResultCreatedItems.length }}</strong>
                </div>
                <div v-if="syncResultCreatedItems.length" class="sync-confirm-list">
                  <article v-for="item in syncResultCreatedItems" :key="syncConfirmItemKey(item, 'result-create')" class="sync-confirm-item">
                    <div class="sync-confirm-item-main">
                      <span :class="['table-tag', providerToneClass(item.provider_slug, 'tag')]">{{ item.provider_name || item.provider_slug }}</span>
                      <strong>{{ item.execution?.account_name || item.local_account_name }}</strong>
                      <code v-if="item.execution?.account_id">#{{ item.execution.account_id }}</code>
                      <code>{{ item.upstream_key_name }}</code>
                    </div>
                  </article>
                </div>
                <div v-else class="sync-confirm-empty">{{ t('admin.upstreamAccounts.syncResultNoCreated') }}</div>
              </section>

              <section class="sync-confirm-section">
                <div class="sync-confirm-section-title">
                  <span>{{ t('admin.upstreamAccounts.syncResultUpdatedSection') }}</span>
                  <strong>{{ syncResultUpdatedItems.length }}</strong>
                </div>
                <div v-if="syncResultUpdatedItems.length" class="sync-confirm-list">
                  <article v-for="item in syncResultUpdatedItems" :key="syncConfirmItemKey(item, 'result-update')" class="sync-confirm-item">
                    <div class="sync-confirm-item-main">
                      <span :class="['table-tag', providerToneClass(item.provider_slug, 'tag')]">{{ item.provider_name || item.provider_slug }}</span>
                      <strong>{{ item.execution?.account_name || item.matched_account_name || item.local_account_name }}</strong>
                      <code v-if="item.execution?.account_id">#{{ item.execution.account_id }}</code>
                      <code>{{ item.upstream_key_name }}</code>
                    </div>
                    <div v-if="syncResultUnboundGroups(item) !== '-'" class="sync-confirm-detail-list">
                      <span class="sync-confirm-detail-chip sync-confirm-detail-warning">
                        {{ t('admin.upstreamAccounts.syncResultUnboundGroups', { groups: syncResultUnboundGroups(item) }) }}
                        <strong>{{ syncResultUnboundGroups(item) }}</strong>
                      </span>
                    </div>
                  </article>
                </div>
                <div v-else class="sync-confirm-empty">{{ t('admin.upstreamAccounts.syncResultNoUpdated') }}</div>
              </section>
            </div>
            <div class="sync-confirm-footer">
              <button type="button" class="ui-button ui-button-primary" @click="closeSyncResultDialog">
                {{ t('common.close') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="showBatchTestConfigDialog" class="batch-test-config-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" data-test="batch-test-config-dialog" @click.self="closeBatchTestConfigDialog">
          <div class="sync-result-modal batch-test-config-modal">
            <div class="sync-confirm-header">
              <div>
                <h3>{{ t('admin.upstreamAccounts.batchTestConfigTitle') }}</h3>
                <p>{{ t('admin.upstreamAccounts.batchTestConfigDescription') }}</p>
              </div>
              <button type="button" class="modal-close-button" :aria-label="t('common.close')" @click="closeBatchTestConfigDialog">
                <Icon name="x" size="md" :stroke-width="2" />
              </button>
            </div>
            <div class="sync-confirm-body">
              <section class="sync-confirm-section">
                <div class="sync-confirm-section-title">
                  <span>{{ t('admin.upstreamAccounts.batchTestPlatformModels') }}</span>
                  <strong>{{ batchTestPlatformOptions.length }}</strong>
                </div>
                <div class="batch-test-config-list">
                  <label v-for="option in batchTestPlatformOptions" :key="option.platform" class="batch-test-config-row">
                    <div class="batch-test-config-platform">
                      <span :class="['table-tag', platformTagClass(option.platform)]">{{ option.platform }}</span>
                      <span>{{ t('admin.upstreamAccounts.batchTestPlatformAccountCount', { count: option.accountCount }) }}</span>
                    </div>
                    <div class="batch-test-model-control">
                      <Select
                        v-model="batchTestModelByPlatform[option.platform]"
                        class="batch-test-model-select"
                        :options="batchTestModelSelectOptions(option.platform)"
                        :placeholder="t('admin.upstreamAccounts.batchTestModelPlaceholder')"
                        :searchable="true"
                        clearable
                        :data-test="`batch-test-model-${option.platform}`"
                      />
                      <span v-if="batchTestModelLoadingByPlatform[option.platform]" class="batch-test-model-hint">
                        {{ t('admin.upstreamAccounts.batchTestModelLoading') }}
                      </span>
                    </div>
                  </label>
                </div>
              </section>
            </div>
            <div class="sync-confirm-footer">
              <button type="button" class="ui-button" :disabled="batchTesting" @click="closeBatchTestConfigDialog">
                {{ t('common.cancel') }}
              </button>
              <button type="button" class="ui-button ui-button-primary" :disabled="batchTesting" data-test="batch-test-config-submit" @click="confirmBatchAccountTest">
                {{ t('admin.upstreamAccounts.batchTestStart') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="showBatchTestResultDialog" class="batch-test-result-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" data-test="batch-test-result-dialog" @click.self="closeBatchTestResultDialog">
          <div class="sync-result-modal batch-test-result-modal">
            <div class="sync-confirm-header">
              <div>
                <h3>{{ t('admin.upstreamAccounts.batchTestResultTitle') }}</h3>
                <p>{{ batchTestProgressDescription }}</p>
              </div>
              <button type="button" class="modal-close-button" :aria-label="t('common.close')" @click="closeBatchTestResultDialog">
                <Icon name="x" size="md" :stroke-width="2" />
              </button>
            </div>
            <div class="sync-confirm-summary">
              <div class="sync-result-stat">
                <span>{{ t('admin.upstreamAccounts.batchTestTotal') }}</span>
                <strong>{{ batchTestResult?.total || 0 }}</strong>
              </div>
              <div class="sync-result-stat">
                <span>{{ t('admin.upstreamAccounts.batchTestCompleted') }}</span>
                <strong>{{ batchTestResult?.completed || 0 }}</strong>
              </div>
              <div class="sync-result-stat">
                <span>{{ t('admin.upstreamAccounts.batchTestSuccess') }}</span>
                <strong>{{ batchTestResult?.success || 0 }}</strong>
              </div>
              <div class="sync-result-stat">
                <span>{{ t('admin.upstreamAccounts.batchTestFailed') }}</span>
                <strong>{{ batchTestResult?.failed || 0 }}</strong>
              </div>
            </div>
            <div class="sync-confirm-body">
              <section class="sync-confirm-section">
                <div class="sync-confirm-section-title">
                  <span>{{ t('admin.upstreamAccounts.batchTestResultSection') }}</span>
                  <strong>{{ batchTestResultItems.length }}</strong>
                </div>
                <div v-if="batchTestResultItems.length" class="batch-result-scroll">
                  <div class="batch-result-toolbar">
                    <div class="batch-result-tabs" :aria-label="t('admin.upstreamAccounts.batchTestResultSection')">
                      <button
                        v-for="option in batchTestFilterOptions"
                        :key="option.value"
                        type="button"
                        :class="['batch-result-tab', { active: batchTestResultFilter === option.value }]"
                        :data-test="`batch-test-filter-${option.value}`"
                        @click="batchTestResultFilter = option.value"
                      >
                        <span>{{ option.label }}</span>
                        <strong>{{ option.count }}</strong>
                      </button>
                    </div>
                    <div class="batch-result-hint">
                      <span>{{ t('admin.upstreamAccounts.batchTestFailureFirstHint') }}</span>
                      <span class="table-tag tag-warning">{{ t('admin.upstreamAccounts.batchTestFailureFirstTag') }}</span>
                    </div>
                  </div>

                  <div v-if="batchTestFilteredItems.length" class="batch-result-list">
                    <article
                      v-for="item in batchTestFilteredItems"
                      :key="`batch-test-card-${item.account_id}`"
                      :class="['batch-result-card', batchTestResultCardTone(item), { 'failed-schedulable': batchTestIsFailedSchedulable(item) }]"
                    >
                      <div class="batch-result-card-head">
                        <div class="batch-result-account">
                          <strong>{{ item.account_name || `#${item.account_id}` }}</strong>
                          <span>#{{ item.account_id }}</span>
                        </div>
                        <div class="batch-result-card-status">
                          <span :class="['test-status-pill', batchTestStatusClass(item.status)]">
                            {{ batchTestStatusLabel(item.status) }}
                          </span>
                          <span v-if="batchTestIsFailedSchedulable(item)" class="table-tag batch-risk-tag">
                            {{ t('admin.upstreamAccounts.batchTestFailedSchedulableTag') }}
                          </span>
                        </div>
                      </div>
                      <div class="batch-result-grid">
                        <div class="batch-result-metric">
                          <span>{{ t('admin.upstreamAccounts.batchTestPlatform') }}</span>
                          <strong>{{ item.platform || '-' }}</strong>
                        </div>
                        <div class="batch-result-metric">
                          <span>{{ t('admin.upstreamAccounts.batchTestLatency') }}</span>
                          <strong>{{ formatLatency(item.latency_ms) }}</strong>
                        </div>
                        <div class="batch-result-metric">
                          <span>{{ t('admin.upstreamAccounts.batchTestUpstreamRate') }}</span>
                          <strong :class="['rate-value', rateToneClass(batchTestUpstreamRate(item))]">
                            {{ formatRate(batchTestUpstreamRate(item)) }}
                          </strong>
                        </div>
                        <div class="batch-result-metric">
                          <span>{{ t('admin.upstreamAccounts.batchTestSchedulable') }}</span>
                          <strong>
                            {{ batchTestItemSchedulable(item) ? t('admin.upstreamAccounts.batchTestSchedulableEnabled') : t('admin.upstreamAccounts.batchTestSchedulableDisabled') }}
                          </strong>
                        </div>
                      </div>
                      <div v-if="item.error_message" class="batch-result-error">{{ item.error_message }}</div>
                      <div class="batch-result-card-actions">
                        <button
                          type="button"
                          class="ui-button ui-button-xs"
                          :disabled="togglingSchedulableId === item.account_id"
                          :data-test="`batch-test-schedulable-toggle-${item.account_id}`"
                          @click="toggleBatchTestItemSchedulable(item)"
                        >
                          {{ batchTestItemSchedulable(item) ? t('admin.upstreamAccounts.batchTestSchedulableDisable') : t('admin.upstreamAccounts.batchTestSchedulableEnable') }}
                        </button>
                        <button
                          type="button"
                          class="ui-button ui-button-xs"
                          :data-test="`batch-test-edit-account-${item.account_id}`"
                          @click="openBatchTestAccountEditDialog(item)"
                        >
                          {{ t('common.edit') }}
                        </button>
                        <button
                          type="button"
                          class="ui-button ui-button-xs ui-button-danger"
                          :data-test="`batch-test-delete-account-${item.account_id}`"
                          @click="openBatchTestAccountDeleteDialog(item)"
                        >
                          {{ t('common.delete') }}
                        </button>
                      </div>
                    </article>
                  </div>
                  <div v-else class="batch-result-empty">{{ t('admin.upstreamAccounts.batchTestNoFilteredResults') }}</div>

                  <details class="batch-table-details">
                    <summary>
                      {{ t('admin.upstreamAccounts.batchTestTableDetails') }}
                      <span>{{ t('admin.upstreamAccounts.batchTestTableDetailsHint') }}</span>
                    </summary>
                    <div class="records-table-wrap batch-test-table-wrap">
                      <table class="records-table batch-test-table">
                        <thead>
                          <tr>
                            <th>
                              <button type="button" class="batch-test-sort-button" data-test="batch-test-sort-account" @click="toggleBatchTestSort('account')">
                                {{ t('admin.upstreamAccounts.batchTestAccount') }}
                                <span>{{ batchTestSortIndicator('account') }}</span>
                              </button>
                            </th>
                            <th>
                              <button type="button" class="batch-test-sort-button" data-test="batch-test-sort-platform" @click="toggleBatchTestSort('platform')">
                                {{ t('admin.upstreamAccounts.batchTestPlatform') }}
                                <span>{{ batchTestSortIndicator('platform') }}</span>
                              </button>
                            </th>
                            <th>
                              <button type="button" class="batch-test-sort-button" data-test="batch-test-sort-upstream_rate" @click="toggleBatchTestSort('upstream_rate')">
                                {{ t('admin.upstreamAccounts.batchTestUpstreamRate') }}
                                <span>{{ batchTestSortIndicator('upstream_rate') }}</span>
                              </button>
                            </th>
                            <th>
                              <button type="button" class="batch-test-sort-button" data-test="batch-test-sort-schedulable" @click="toggleBatchTestSort('schedulable')">
                                {{ t('admin.upstreamAccounts.batchTestSchedulable') }}
                                <span>{{ batchTestSortIndicator('schedulable') }}</span>
                              </button>
                            </th>
                            <th>
                              <button type="button" class="batch-test-sort-button" data-test="batch-test-sort-status" @click="toggleBatchTestSort('status')">
                                {{ t('admin.upstreamAccounts.batchTestStatus') }}
                                <span>{{ batchTestSortIndicator('status') }}</span>
                              </button>
                            </th>
                            <th>
                              <button type="button" class="batch-test-sort-button" data-test="batch-test-sort-latency" @click="toggleBatchTestSort('latency')">
                                {{ t('admin.upstreamAccounts.batchTestLatency') }}
                                <span>{{ batchTestSortIndicator('latency') }}</span>
                              </button>
                            </th>
                            <th>
                              <button type="button" class="batch-test-sort-button" data-test="batch-test-sort-finished_at" @click="toggleBatchTestSort('finished_at')">
                                {{ t('admin.upstreamAccounts.batchTestFinishedAt') }}
                                <span>{{ batchTestSortIndicator('finished_at') }}</span>
                              </button>
                            </th>
                            <th>{{ t('admin.upstreamAccounts.batchTestError') }}</th>
                            <th>{{ t('common.actions') }}</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr
                            v-for="item in batchTestFilteredItems"
                            :key="`batch-test-${item.account_id}`"
                            :class="{ 'batch-test-risk-row': batchTestIsFailedSchedulable(item) }"
                          >
                            <td>
                              <div class="two-line-cell">
                                <span class="main-text">{{ item.account_name || `#${item.account_id}` }}</span>
                                <code class="source-id">#{{ item.account_id }}</code>
                              </div>
                            </td>
                            <td>
                              <span :class="['table-tag', platformTagClass(item.platform)]">{{ item.platform || '-' }}</span>
                            </td>
                            <td>
                              <span :class="['rate-value', rateToneClass(batchTestUpstreamRate(item))]">
                                {{ formatRate(batchTestUpstreamRate(item)) }}
                              </span>
                            </td>
                            <td>
                              <div class="batch-test-schedulable-cell">
                                <span :class="['test-status-pill', batchTestItemSchedulable(item) ? 'test-status-success' : 'test-status-failed']">
                                  {{ batchTestItemSchedulable(item) ? t('admin.upstreamAccounts.batchTestSchedulableEnabled') : t('admin.upstreamAccounts.batchTestSchedulableDisabled') }}
                                </span>
                                <button
                                  type="button"
                                  class="ui-button ui-button-xs"
                                  :disabled="togglingSchedulableId === item.account_id"
                                  :data-test="`batch-test-table-schedulable-toggle-${item.account_id}`"
                                  @click="toggleBatchTestItemSchedulable(item)"
                                >
                                  {{ batchTestItemSchedulable(item) ? t('admin.upstreamAccounts.batchTestSchedulableDisable') : t('admin.upstreamAccounts.batchTestSchedulableEnable') }}
                                </button>
                              </div>
                            </td>
                            <td>
                              <div class="batch-test-status-cell">
                                <span :class="['test-status-pill', batchTestStatusClass(item.status)]">
                                  {{ batchTestStatusLabel(item.status) }}
                                </span>
                                <span v-if="batchTestIsFailedSchedulable(item)" class="table-tag batch-risk-tag">
                                  {{ t('admin.upstreamAccounts.batchTestFailedSchedulableTag') }}
                                </span>
                              </div>
                            </td>
                            <td>{{ formatLatency(item.latency_ms) }}</td>
                            <td>{{ item.finished_at ? formatDateTime(item.finished_at) : '-' }}</td>
                            <td class="batch-test-error">{{ item.error_message || '-' }}</td>
                            <td>
                              <div class="batch-test-actions-cell">
                                <button
                                  type="button"
                                  class="ui-button ui-button-xs"
                                  :data-test="`batch-test-table-edit-account-${item.account_id}`"
                                  @click="openBatchTestAccountEditDialog(item)"
                                >
                                  {{ t('common.edit') }}
                                </button>
                                <button
                                  type="button"
                                  class="ui-button ui-button-xs ui-button-danger"
                                  :data-test="`batch-test-table-delete-account-${item.account_id}`"
                                  @click="openBatchTestAccountDeleteDialog(item)"
                                >
                                  {{ t('common.delete') }}
                                </button>
                              </div>
                            </td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                  </details>
                </div>
                <div v-else class="sync-confirm-empty">{{ t('admin.upstreamAccounts.batchTestNoResults') }}</div>
              </section>
            </div>
            <div class="sync-confirm-footer">
              <button v-if="batchTestCanCancel" type="button" class="ui-button" @click="cancelBatchAccountTest">
                {{ t('admin.upstreamAccounts.batchTestCancel') }}
              </button>
              <button type="button" class="ui-button ui-button-primary" @click="closeBatchTestResultDialog">
                {{ t('common.close') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="accountGroupDialogItem" class="account-group-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" @click.self="closeAccountGroupDialog">
          <div class="w-full max-w-lg overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h3 class="text-lg font-semibold text-gray-950 dark:text-white">{{ t('admin.upstreamAccounts.editBoundGroupsTitle') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.editBoundGroupsDescription') }}</p>
            </div>
            <div class="space-y-4 px-5 py-4">
              <div>
                <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.columns.localAccount') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-950 dark:text-white">{{ accountGroupDialogItem.matched_account_name || accountGroupDialogItem.local_account_name }}</div>
              </div>
              <GroupSelector
                v-model="accountGroupIds"
                :groups="accountGroupOptions"
                :platform="accountGroupPlatform"
                searchable
              />
              <div v-if="accountGroupIds.length" class="tag-list">
                <span v-for="groupID in accountGroupIds" :key="groupID" :class="['group-chip', accountGroupPlatformTagClass]">
                  {{ groupNameById(groupID) }}
                </span>
              </div>
            </div>
            <div class="flex justify-between gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
              <button type="button" class="btn btn-danger btn-sm" :disabled="savingAccountGroupId === accountGroupDialogItem.matched_account_id" @click="clearAccountGroups">
                {{ t('admin.upstreamAccounts.clearBoundGroups') }}
              </button>
              <div class="flex gap-2">
                <button type="button" class="btn btn-secondary btn-sm" :disabled="savingAccountGroupId === accountGroupDialogItem.matched_account_id" @click="closeAccountGroupDialog">
                  {{ t('common.cancel') }}
                </button>
                <button type="button" class="btn btn-primary btn-sm" :disabled="savingAccountGroupId === accountGroupDialogItem.matched_account_id" @click="saveAccountGroups">
                  <Icon name="cog" size="sm" class="mr-1" :class="savingAccountGroupId === accountGroupDialogItem.matched_account_id ? 'animate-spin' : ''" />
                  {{ t('common.save') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <AccountTestModal
          :show="showTestModal"
          :account="testingAccount"
          @close="closeTestModal"
          @test-result="handleAccountTestResult"
        />
        <CreateAccountModal
          v-if="showCreateAccountModal"
          :show="showCreateAccountModal"
          :proxies="accountProxies"
          :groups="accountEditGroups"
          :initial-values="createAccountInitialValues"
          @close="closeCreateAccountDialog"
          @created="handleAccountCreated"
        />
        <EditAccountModal
          v-if="showEditAccountModal"
          :show="showEditAccountModal"
          :account="editingAccount"
          :proxies="accountProxies"
          :groups="accountEditGroups"
          @close="closeAccountEditDialog"
          @updated="handleAccountUpdated"
        />
        <ConfirmDialog
          :show="showDeleteAccountDialog"
          :title="t('admin.accounts.deleteAccount')"
          :message="t('admin.accounts.deleteConfirm', { name: deletingAccount?.name })"
          :confirm-text="t('common.delete')"
          :cancel-text="t('common.cancel')"
          danger
          @confirm="confirmDeleteAccount"
          @cancel="closeAccountDeleteDialog"
        />
        <TempUnschedStatusModal
          :show="showTempUnsched"
          :account="tempUnschedAccount"
          @close="closeTempUnschedModal"
          @reset="handleTempUnschedReset"
        />
        <UpstreamProviderTrendModal
          :show="showTrendModal"
          :provider-slug="trendProviderSlug"
          :provider-name="trendProviderName"
          :rows="balanceOverview?.rows || []"
          @close="closeTrendModal"
        />
      </template>
    </TablePageLayout>
    <button
      v-if="showMobileBackToFilters"
      type="button"
      class="mobile-back-to-filters"
      :aria-label="t('admin.upstreamAccounts.mobileBackToFilters')"
      :title="t('admin.upstreamAccounts.mobileBackToFilters')"
      @click="scrollToUpstreamFilters"
    >
      <Icon name="arrowUp" size="sm" :stroke-width="2.4" />
    </button>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import type {
  UpstreamAccountRateGuardConfig,
  UpstreamAccountSyncChangeDetail,
  UpstreamAccountSyncConflictAccount,
  UpstreamAccountSyncBoundGroup,
  UpstreamAccountSyncItem,
  UpstreamAccountSyncRecord,
  UpstreamAccountSyncResult,
  UpstreamAccountSyncUnbindDetail,
  UpstreamBalanceConsumptionOverview,
} from '@/api/admin/upstreamAccountSync'
import type { UpstreamProviderConfig } from '@/api/admin/upstreamProviders'
import type {
  Account,
  AdminGroup,
  BatchAccountTestItem,
  BatchAccountTestJob,
  BatchAccountTestJobStatus,
  BatchAccountTestStatus,
  ClaudeModel,
  GroupPlatform,
  Proxy as AccountProxy
} from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { useRouteQueryFilters } from '@/composables/useRouteQueryFilters'
import type { Column } from '@/components/common/types'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import { CreateAccountModal, EditAccountModal } from '@/components/account'
import type { CreateAccountInitialValues } from '@/components/account/CreateAccountModal.vue'
import AccountTestModal from '@/components/admin/account/AccountTestModal.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import TempUnschedStatusModal from '@/components/account/TempUnschedStatusModal.vue'
import UpstreamProviderTrendModal from '@/components/admin/upstream/UpstreamProviderTrendModal.vue'
import UpstreamAccountRateGuardPanel from '@/components/admin/upstream/UpstreamAccountRateGuardPanel.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

type AccountTestStatus = 'testing' | 'success' | 'failed'
type BatchTestPlatformOption = {
  platform: string
  accountId: number
  accountCount: number
}
type BatchTestSortKey = 'account' | 'platform' | 'upstream_rate' | 'schedulable' | 'status' | 'latency' | 'finished_at'
type BatchTestResultFilter = 'all' | 'failed' | 'failed_schedulable' | 'failed_unschedulable' | 'success' | 'success_unschedulable' | 'success_upstream_disabled' | 'skipped'
type QuickFilterKey = 'all' | 'update' | 'conflict' | 'risk' | 'ignored' | 'unbound' | 'failed' | 'enabled' | 'disabled'
type RateGuardIgnoredAccountDetail = {
  id: number
  name: string
  meta: string
  title: string
}
type StatCardKey = 'total' | 'create' | 'update' | 'risk'
type SortOrder = 'asc' | 'desc'

const MAX_BATCH_TEST_ACCOUNTS = 200
const BATCH_TEST_REFRESH_CONCURRENCY = 5
const BATCH_TEST_POLL_INTERVAL_MS = 1500
const BATCH_TEST_TOTAL_TIMEOUT_SECONDS = 10 * 60

const result = ref<UpstreamAccountSyncResult | null>(null)
const loading = ref(false)
const syncing = ref(false)
const loadingRateGuardConfig = ref(false)
const savingRateGuardConfig = ref(false)
const runningRateGuardNow = ref(false)
const togglingRateGuardIgnoreId = ref<number | null>(null)
const savingAccountGroupId = ref<number | null>(null)
const editingPriorityAccountId = ref<number | null>(null)
const savingPriorityAccountId = ref<number | null>(null)
const priorityDraft = ref<number | string | null>(null)
const priorityInputRef = ref<HTMLInputElement | null>(null)
const testingAccountId = ref<number | null>(null)
const togglingSchedulableId = ref<number | null>(null)
const showTestModal = ref(false)
const showTempUnsched = ref(false)
const showCreateAccountModal = ref(false)
const showEditAccountModal = ref(false)
const showDeleteAccountDialog = ref(false)
const testingAccount = ref<Account | null>(null)
const tempUnschedAccount = ref<Account | null>(null)
const editingAccount = ref<Account | null>(null)
const deletingAccount = ref<Account | null>(null)
const createAccountInitialValues = ref<CreateAccountInitialValues | null>(null)
const accountTestStatusById = ref<Record<number, AccountTestStatus>>({})
const matchedAccountsById = ref<Record<number, Account>>({})
const localGroups = ref<AdminGroup[]>([])
const accountEditGroups = ref<AdminGroup[]>([])
const accountProxies = ref<AccountProxy[]>([])
const loadError = ref('')
const searchQuery = ref('')
const providerFilter = ref('')
const platformFilter = ref('')
const sourceFilter = ref('')
const groupFilter = ref('')
const activeQuickFilter = ref<QuickFilterKey>('all')
useRouteQueryFilters([
  { queryKey: 'provider', state: providerFilter },
  { queryKey: 'status', state: activeQuickFilter, fromQuery: value => value === 'conflict' ? 'conflict' : 'all', toQuery: value => value === 'conflict' ? 'conflict' : undefined },
])
const showAdvancedFilters = ref(false)
const showMobileBackToFilters = ref(false)
const activeStatDetailsKey = ref<StatCardKey | null>(null)
const showRateGuardIgnoredDialog = ref(false)
const rateGuardAutomationTarget = ref(false)
const rateGuardConfig = ref<UpstreamAccountRateGuardConfig | null>(null)
const rateGuardForm = ref({
  enabled: false,
  interval_seconds: 3600,
  ignored_account_ids: [] as number[]
})
const rateGuardIgnoredInput = ref('')
const accountGroupDialogItem = ref<UpstreamAccountSyncItem | null>(null)
const accountGroupIds = ref<number[]>([])
const accountGroupPlatform = ref<GroupPlatform | undefined>()
const balanceOverview = ref<UpstreamBalanceConsumptionOverview | null>(null)
const showTrendModal = ref(false)
const trendProviderSlug = ref('')
const trendProviderName = ref('')
const showSyncLogsDialog = ref(false)
const showSyncConfirmDialog = ref(false)
const showSyncResultDialog = ref(false)
const showBatchTestConfigDialog = ref(false)
const showBatchTestResultDialog = ref(false)
const batchTesting = ref(false)
const batchTestResult = ref<BatchAccountTestJob | null>(null)
const batchTestModelByPlatform = ref<Record<string, string>>({})
const batchTestModelOptionsByPlatform = ref<Record<string, ClaudeModel[]>>({})
const batchTestModelSourceAccountByPlatform = ref<Record<string, number>>({})
const batchTestModelLoadingByPlatform = ref<Record<string, boolean>>({})
const batchTestSortKey = ref<BatchTestSortKey | null>(null)
const batchTestSortOrder = ref<SortOrder>('asc')
const batchTestResultFilter = ref<BatchTestResultFilter>('all')
const batchTestPollTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const batchTestPollToken = ref(0)
const lastSyncResult = ref<UpstreamAccountSyncResult | null>(null)
const syncConfirmOptions = ref({
  create_missing: true,
  update_existing: true,
  apply_rate_guard: true
})
const syncConfirmSelectedItems = ref<Record<string, boolean>>({})

type UpstreamAccountSyncLogEntry = UpstreamAccountSyncUnbindDetail & {
  created_at: string
  key: string
}

const upstreamAccountSortableColumnKeys = new Set(['source', 'priority', 'balance', 'status', 'schedulable', 'test_status'])

const columns = computed<Column[]>(() => [
  { key: 'source', label: t('admin.upstreamAccounts.columns.source'), class: 'upstream-center-column upstream-source-column' },
  { key: 'upstream_key_name', label: t('admin.upstreamAccounts.columns.upstreamKey'), class: 'upstream-center-column upstream-key-column' },
  { key: 'local_account_name', label: t('admin.upstreamAccounts.columns.localAccount'), class: 'upstream-center-column upstream-local-account-column' },
  { key: 'priority', label: t('admin.upstreamAccounts.columns.priority'), class: 'upstream-center-column upstream-priority-column' },
  { key: 'upstream_rate_multiplier', label: t('admin.upstreamAccounts.columns.upstreamRate'), sortable: true, class: 'upstream-center-column upstream-rate-column' },
  { key: 'local_group_name', label: t('admin.upstreamAccounts.columns.boundGroups'), class: 'upstream-center-column upstream-bound-groups-column' },
  { key: 'balance', label: '余额', class: 'upstream-center-column upstream-money-column' },
  { key: 'today_consumption', label: '今日消费', class: 'upstream-center-column upstream-money-column' },
  { key: 'status', label: t('admin.accounts.columns.status'), class: 'upstream-center-column upstream-status-column' },
  { key: 'schedulable', label: t('admin.accounts.columns.schedulable'), class: 'upstream-center-column upstream-schedulable-column' },
  { key: 'test_status', label: t('admin.upstreamAccounts.columns.testStatus'), class: 'upstream-center-column upstream-test-status-column' },
  { key: 'last_tested_at', label: t('admin.upstreamAccounts.columns.lastTestedAt'), class: 'upstream-center-column upstream-test-time-column' },
  { key: 'actions', label: t('common.actions'), class: 'upstream-center-column upstream-actions-column' }
].map(column => upstreamAccountSortableColumnKeys.has(column.key) ? { ...column, sortable: true } : column))

const emptySummary = {
  upstream_key_count: 0,
  matched_account_count: 0,
  create_count: 0,
  update_count: 0,
  skip_count: 0,
  conflict_count: 0,
  rate_violation_count: 0,
  unbound_group_count: 0
}

const summary = computed(() => result.value?.summary || emptySummary)
const syncProviders = computed(() => {
  const providers = result.value?.providers || []
  const defaultProvider = result.value?.default_provider
  if (!defaultProvider?.slug || providers.some(provider => provider.slug === defaultProvider.slug)) {
    return providers
  }
  return [defaultProvider, ...providers]
})
const syncProviderBySlug = computed(() => {
  const bySlug = new Map<string, UpstreamProviderConfig>()
  for (const provider of syncProviders.value) {
    if (provider.slug) {
      bySlug.set(provider.slug, provider)
    }
  }
  return bySlug
})
const items = computed<UpstreamAccountSyncItem[]>(() => result.value?.items || [])
const warnings = computed(() => result.value?.warnings || [])
const records = computed<UpstreamAccountSyncRecord[]>(() => result.value?.records || [])
const statCards = computed<Array<{
  key: StatCardKey
  label: string
  value: number
  icon: 'database' | 'plus' | 'refresh' | 'exclamationTriangle'
  tone: 'emerald' | 'gray' | 'orange' | 'red'
}>>(() => [
  {
    key: 'total',
    label: t('admin.upstreamAccounts.upstreamKeys'),
    value: summary.value.upstream_key_count,
    icon: 'database' as const,
    tone: 'emerald'
  },
  {
    key: 'create',
    label: t('admin.upstreamAccounts.toCreate'),
    value: summary.value.create_count,
    icon: 'plus' as const,
    tone: 'gray'
  },
  {
    key: 'update',
    label: t('admin.upstreamAccounts.toUpdate'),
    value: summary.value.update_count,
    icon: 'refresh' as const,
    tone: 'orange'
  },
  {
    key: 'risk',
    label: t('admin.upstreamAccounts.rateRisks'),
    value: summary.value.rate_violation_count,
    icon: 'exclamationTriangle' as const,
    tone: 'red'
  }
])
const syncLogEntries = computed<UpstreamAccountSyncLogEntry[]>(() => {
  const entries: UpstreamAccountSyncLogEntry[] = []
  for (const record of records.value) {
    for (const detail of record.unbind_details || []) {
      const unboundGroupIDs = numberArray(detail.unbound_group_ids)
      entries.push({
        ...detail,
        unbound_group_ids: unboundGroupIDs,
        unbound_group_names: stringArray(detail.unbound_group_names),
        remaining_group_ids: numberArray(detail.remaining_group_ids),
        created_at: record.created_at,
        key: `${toRFC3339(record.created_at)}-${detail.matched_local_account_id}-${detail.upstream_key_name}-${unboundGroupIDs.join('_')}`
      })
    }
  }
  return entries
})
const unhandledSyncLogEntries = computed(() => syncLogEntries.value.filter(entry => !isSyncLogHandled(entry)))
const canSync = computed(() => summary.value.create_count > 0 || summary.value.update_count > 0 || summary.value.rate_violation_count > 0)
const syncCreateItems = computed(() => items.value.filter(item => item.action === 'create' && item.upstream_api_key))
const syncUpdateItems = computed(() => items.value.filter(item => item.action === 'update'))
const syncRateGuardItems = computed(() => items.value.filter(item => item.rate_violation && numberArray(item.unbound_group_ids).length > 0))
const statDetailsItemsByKey = computed<Record<StatCardKey, UpstreamAccountSyncItem[]>>(() => ({
  total: items.value,
  create: syncCreateItems.value,
  update: syncUpdateItems.value,
  risk: items.value.filter(item => item.rate_violation)
}))
const activeStatDetailsItems = computed(() => {
  const key = activeStatDetailsKey.value
  return key ? statDetailsItemsByKey.value[key] : []
})
const activeStatDetailsCard = computed(() => {
  const key = activeStatDetailsKey.value
  return key ? statCards.value.find(card => card.key === key) : null
})
const activeStatDetailsTitle = computed(() => {
  const label = activeStatDetailsCard.value?.label || ''
  return label ? t('admin.upstreamAccounts.statDetailsTitle', { label }) : ''
})
const activeStatDetailsDescription = computed(() => {
  if (activeStatDetailsKey.value === 'total') return t('admin.upstreamAccounts.statDetailsTotalDescription')
  if (activeStatDetailsKey.value === 'create') return t('admin.upstreamAccounts.statDetailsCreateDescription')
  if (activeStatDetailsKey.value === 'update') return t('admin.upstreamAccounts.statDetailsUpdateDescription')
  if (activeStatDetailsKey.value === 'risk') return t('admin.upstreamAccounts.statDetailsRiskDescription')
  return t('admin.upstreamAccounts.statDetailsDescription')
})
const syncConfirmCanSubmit = computed(() => (
  (syncConfirmOptions.value.create_missing && syncCreateItems.value.some(syncConfirmItemSelected)) ||
  (syncConfirmOptions.value.update_existing && syncUpdateItems.value.some(syncConfirmItemSelected))
))
const syncResultItems = computed(() => (lastSyncResult.value?.items || []).filter(item => item.execution?.executed))
const syncResultCreatedItems = computed(() => syncResultItems.value.filter(item => item.execution?.action === 'create'))
const syncResultUpdatedItems = computed(() => syncResultItems.value.filter(item => item.execution?.action === 'update'))
const syncResultUnboundCount = computed(() => syncResultUpdatedItems.value.reduce((total, item) => total + numberArray(item.execution?.unbound_group_ids).length, 0))
const syncProviderLabel = computed(() => {
  if (syncProviders.value.length === 1) return syncProviders.value[0].name || syncProviders.value[0].slug
  if (syncProviders.value.length > 1) return t('admin.upstreamAccounts.multipleProviders', { count: syncProviders.value.length })
  return '-'
})
const syncProviderCode = computed(() => {
  if (syncProviders.value.length === 1) return syncProviders.value[0].slug
  return ''
})
const rateGuardLastRunText = computed(() => {
  if (!rateGuardConfig.value?.last_run_at) {
    return t('admin.upstreamAccounts.rateGuardNeverRun')
  }
  return formatDateTime(rateGuardConfig.value.last_run_at)
})
const rateGuardDailyRunsText = computed(() => {
  const seconds = Number(rateGuardForm.value.interval_seconds)
  if (!Number.isFinite(seconds) || seconds <= 0) return '\u7ea6\u6bcf\u65e5\u6267\u884c - \u6b21'
  return `\u7ea6\u6bcf\u65e5\u6267\u884c ${Math.floor(86400 / seconds)} \u6b21`
})
const rateGuardIgnoredAccountIDs = computed(() => normalizeRateGuardIgnoredAccountIDs(rateGuardForm.value.ignored_account_ids))
const rateGuardIgnoredCount = computed(() => rateGuardIgnoredAccountIDs.value.length)
const rateGuardIgnoredInputIDs = computed(() => parseRateGuardIgnoredInput(rateGuardIgnoredInput.value))
const rateGuardIgnoredInputInvalid = computed(() => Boolean(rateGuardIgnoredInput.value.trim()) && rateGuardIgnoredInputIDs.value === null)
const rateGuardIgnoredDialogIDs = computed(() => rateGuardIgnoredInputIDs.value ?? rateGuardIgnoredAccountIDs.value)
const rateGuardIgnoredDialogCount = computed(() => rateGuardIgnoredDialogIDs.value.length)
const rateGuardIgnoredSummaryText = computed(() => {
  const count = rateGuardIgnoredCount.value
  return count > 0
    ? t('admin.upstreamAccounts.rateGuardIgnoredSummary', { count })
    : t('admin.upstreamAccounts.rateGuardIgnoredNone')
})
const rateGuardIgnoredDialogSummaryText = computed(() => {
  const count = rateGuardIgnoredDialogCount.value
  return count > 0
    ? t('admin.upstreamAccounts.rateGuardIgnoredSummary', { count })
    : t('admin.upstreamAccounts.rateGuardIgnoredNone')
})
const rateGuardIgnoredAccountDetails = computed<RateGuardIgnoredAccountDetail[]>(() => {
  const itemByAccountID = new Map<number, UpstreamAccountSyncItem>()
  for (const item of items.value) {
    const accountID = Number(item.matched_account_id)
    if (Number.isSafeInteger(accountID) && accountID > 0 && !itemByAccountID.has(accountID)) {
      itemByAccountID.set(accountID, item)
    }
  }
  return rateGuardIgnoredDialogIDs.value.map((id) => {
    const account = matchedAccountsById.value[id]
    const item = itemByAccountID.get(id)
    const name = account?.name || item?.matched_account_name || item?.local_account_name || t('admin.upstreamAccounts.rateGuardUnknownAccount', { id })
    const platform = account?.platform || (item ? matchedAccountPlatform(item) : '')
    const metaParts = [
      platform,
      t('admin.upstreamAccounts.rateGuardIgnoredAccountId', { id })
    ].filter(Boolean)
    return {
      id,
      name,
      meta: metaParts.join(' · '),
      title: `${name} · ${metaParts.join(' · ')}`
    }
  })
})
const providerOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamAccounts.allProviders') },
  ...Array.from(new Map(items.value.map(item => [
    item.provider_slug,
    {
      value: item.provider_slug,
      label: item.provider_name ? `${item.provider_name} (${item.provider_slug})` : item.provider_slug
    }
  ])).values())
])
const sourceOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamAccounts.allSources') },
  { value: 'synced', label: t('admin.upstreamAccounts.sourceSynced') },
  { value: 'unsynced', label: t('admin.upstreamAccounts.sourceUnsynced') }
])
const platformFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamAccounts.allPlatforms', '全部平台') },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
])
const groupOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamAccounts.allGroups') },
  ...localGroups.value.map(group => ({
    value: String(group.id),
    label: group.name
  }))
])
const baseFilteredItems = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  const selectedGroupID = Number(groupFilter.value)
  return items.value.filter((item) => {
    if (providerFilter.value && item.provider_slug !== providerFilter.value) return false
    if (platformFilter.value && matchedAccountPlatform(item) !== platformFilter.value) return false
    if (sourceFilter.value === 'synced' && !item.matched_account_id) return false
    if (sourceFilter.value === 'unsynced' && item.matched_account_id) return false
    if (groupFilter.value) {
      const boundGroupIDs = [
        item.local_group_id,
        ...(item.bound_groups || []).map(group => group.id)
      ]
        .map(id => Number(id))
        .filter(id => Number.isFinite(id))
      if (!boundGroupIDs.includes(selectedGroupID)) return false
    }
    if (!keyword) return true
    const haystack = [
      item.provider_name,
      item.provider_slug,
      item.upstream_key_name,
      item.upstream_group_name,
      item.local_account_name,
      item.matched_account_name,
      item.local_group_name
    ].filter(Boolean).join(' ').toLowerCase()
    return haystack.includes(keyword)
  })
})
const quickFilterOptions = computed(() => {
  const source = baseFilteredItems.value
  return [
    {
      key: 'all' as const,
      label: t('common.all'),
      count: source.length
    },
    {
      key: 'update' as const,
      label: t('admin.upstreamAccounts.toUpdate'),
      count: source.filter(item => item.action === 'update').length
    },
    {
      key: 'conflict' as const,
      label: t('admin.upstreamAccounts.quickFilterConflict'),
      count: source.filter(item => item.action === 'conflict').length,
      tone: 'danger' as const
    },
    {
      key: 'risk' as const,
      label: t('admin.upstreamAccounts.quickFilterRateRisk'),
      count: source.filter(item => item.rate_violation).length,
      tone: 'danger' as const
    },
    {
      key: 'ignored' as const,
      label: t('admin.upstreamAccounts.quickFilterIgnoredAccounts'),
      count: source.filter(isRateGuardIgnored).length,
      tone: 'muted' as const
    },
    {
      key: 'unbound' as const,
      label: t('admin.upstreamAccounts.quickFilterNoGroups'),
      count: source.filter(upstreamAccountHasNoGroups).length
    },
    {
      key: 'failed' as const,
      label: t('admin.upstreamAccounts.quickFilterTestFailed'),
      count: source.filter(upstreamAccountTestFailed).length,
      tone: 'danger' as const
    },
    {
      key: 'enabled' as const,
      label: t('admin.upstreamAccounts.quickFilterEnabledProvider'),
      count: source.filter(isProviderEnabled).length
    },
    {
      key: 'disabled' as const,
      label: t('admin.upstreamAccounts.quickFilterDisabledProvider'),
      count: source.filter(isProviderDisabled).length,
      tone: 'muted' as const
    }
  ]
})
const filteredItems = computed(() => baseFilteredItems.value.filter(item => upstreamAccountMatchesQuickFilter(item, activeQuickFilter.value)))
const activeFilterCount = computed(() => [
  searchQuery.value.trim(),
  providerFilter.value,
  platformFilter.value,
  sourceFilter.value,
  groupFilter.value,
  activeQuickFilter.value !== 'all' ? activeQuickFilter.value : ''
].filter(Boolean).length)
const tableItems = computed(() => filteredItems.value.map(item => ({
  ...item,
  source: upstreamAccountSourceSortValue(item),
  priority: upstreamAccountPrioritySortValue(item),
  balance: upstreamAccountBalanceSortValue(item),
  status: upstreamAccountStatusSortValue(item),
  schedulable: upstreamAccountSchedulableSortValue(item),
  test_status: upstreamAccountTestStatusSortValue(item)
})))
const batchTestAccountIds = computed(() => Array.from(
  new Set(
    filteredItems.value
      .map(item => Number(item.matched_account_id))
      .filter(id => Number.isFinite(id) && id > 0)
  )
))
const batchTestPlatformOptions = computed<BatchTestPlatformOption[]>(() => {
  const byPlatform = new Map<string, { platform: string; accountIds: number[] }>()
  for (const accountId of batchTestAccountIds.value) {
    const account = matchedAccountsById.value[accountId]
    const platform = account?.platform?.trim()
    if (!platform) continue
    const existing = byPlatform.get(platform)
    if (existing) {
      existing.accountIds.push(accountId)
    } else {
      byPlatform.set(platform, {
        platform,
        accountIds: [accountId]
      })
    }
  }
  return Array.from(byPlatform.values())
    .map(group => ({
      platform: group.platform,
      accountId: selectBatchTestModelRepresentativeAccountId(group.accountIds),
      accountCount: group.accountIds.length
    }))
    .sort((a, b) => a.platform.localeCompare(b.platform))
})

function selectBatchTestModelRepresentativeAccountId(accountIds: number[]) {
  return accountIds
    .map((accountId, index) => ({
      accountId,
      index,
      account: matchedAccountsById.value[accountId]
    }))
    .sort((a, b) => {
      const scoreDelta = batchTestModelRepresentativeScore(a.account) - batchTestModelRepresentativeScore(b.account)
      if (scoreDelta !== 0) return scoreDelta
      return a.index - b.index
    })[0]?.accountId || accountIds[0] || 0
}

function batchTestModelRepresentativeScore(account: Account | undefined) {
  if (!account) return 100
  if (account.type === 'upstream') return 0
  if (account.type === 'apikey') return 1
  if (account.type === 'bedrock') return 2
  if (account.type === 'service_account') return 3
  if (account.type === 'oauth' || account.type === 'setup-token') return 20
  return 10
}

function batchTestModelSelectOptions(platform: string): SelectOption[] {
  const models = batchTestModelOptionsByPlatform.value[platform] || []
  const options = models.map(model => ({
    value: model.id,
    label: model.display_name && model.display_name !== model.id
      ? `${model.display_name} (${model.id})`
      : model.id,
  }))
  const selectedModel = batchTestModelByPlatform.value[platform]?.trim()
  if (selectedModel && !options.some(option => option.value === selectedModel)) {
    options.unshift({ value: selectedModel, label: selectedModel })
  }
  return options
}
const batchTestRateByAccountId = computed(() => {
  const byAccountId = new Map<number, number>()
  for (const item of filteredItems.value) {
    const accountId = Number(item.matched_account_id)
    const rate = Number(item.upstream_rate_multiplier)
    if (Number.isFinite(accountId) && Number.isFinite(rate)) {
      byAccountId.set(accountId, rate)
    }
  }
  return byAccountId
})
const batchTestProviderDisabledByAccountId = computed(() => {
  const byAccountId = new Map<number, boolean>()
  for (const item of filteredItems.value) {
    const accountId = Number(item.matched_account_id)
    if (!Number.isFinite(accountId) || accountId <= 0) continue
    const disabled = isProviderDisabled(item)
    if (disabled || !byAccountId.has(accountId)) {
      byAccountId.set(accountId, disabled)
    }
  }
  return byAccountId
})
const batchTestFailureFirstItems = computed<BatchAccountTestItem[]>(() => {
  const items = batchTestResult.value?.results || []
  return orderBatchTestItemsFailureFirst(items)
})
const batchTestResultItems = computed<BatchAccountTestItem[]>(() => {
  const items = batchTestFailureFirstItems.value
  if (!batchTestSortKey.value) return items
  const key = batchTestSortKey.value
  const order = batchTestSortOrder.value
  return items
    .map((item, index) => ({ item, index }))
    .sort((a, b) => {
      const compared = compareBatchTestItems(a.item, b.item, key)
      if (compared !== 0) return order === 'asc' ? compared : -compared
      return a.index - b.index
    })
    .map(entry => entry.item)
})
const batchTestResultCounts = computed(() => {
  const items = batchTestResultItems.value
  const success = items.filter(item => item.status === 'success').length
  const skipped = items.filter(batchTestIsSkipped).length
  const failed = items.filter(batchTestIsFailed).length
  const failedSchedulable = items.filter(batchTestIsFailedSchedulable).length
  const failedUnschedulable = items.filter(batchTestIsFailedUnschedulable).length
  const successUnschedulable = items.filter(batchTestIsSuccessUnschedulable).length
  const successUpstreamDisabled = items.filter(batchTestIsSuccessUpstreamDisabled).length
  return {
    all: items.length,
    failed,
    failedSchedulable,
    failedUnschedulable,
    successUnschedulable,
    successUpstreamDisabled,
    success,
    skipped
  }
})
const batchTestFilterOptions = computed(() => {
  const counts = batchTestResultCounts.value
  return [
    { value: 'all' as const, label: t('common.all'), count: counts.all },
    { value: 'failed' as const, label: t('admin.upstreamAccounts.batchTestFailed'), count: counts.failed },
    { value: 'failed_schedulable' as const, label: t('admin.upstreamAccounts.batchTestFailedSchedulable'), count: counts.failedSchedulable },
    { value: 'failed_unschedulable' as const, label: t('admin.upstreamAccounts.batchTestFailedUnschedulable'), count: counts.failedUnschedulable },
    { value: 'success' as const, label: t('admin.upstreamAccounts.batchTestSuccess'), count: counts.success },
    { value: 'success_unschedulable' as const, label: t('admin.upstreamAccounts.batchTestSuccessUnschedulable'), count: counts.successUnschedulable },
    { value: 'success_upstream_disabled' as const, label: t('admin.upstreamAccounts.batchTestSuccessUpstreamDisabled'), count: counts.successUpstreamDisabled },
    { value: 'skipped' as const, label: t('admin.upstreamAccounts.batchTestSkipped'), count: counts.skipped }
  ]
})
const batchTestFilteredItems = computed<BatchAccountTestItem[]>(() => {
  const filter = batchTestResultFilter.value
  if (filter === 'success') return batchTestResultItems.value.filter(item => item.status === 'success')
  if (filter === 'success_unschedulable') return batchTestResultItems.value.filter(batchTestIsSuccessUnschedulable)
  if (filter === 'success_upstream_disabled') return batchTestResultItems.value.filter(batchTestIsSuccessUpstreamDisabled)
  if (filter === 'failed') return batchTestResultItems.value.filter(batchTestIsFailed)
  if (filter === 'failed_schedulable') return batchTestResultItems.value.filter(batchTestIsFailedSchedulable)
  if (filter === 'failed_unschedulable') return batchTestResultItems.value.filter(batchTestIsFailedUnschedulable)
  if (filter === 'skipped') return batchTestResultItems.value.filter(batchTestIsSkipped)
  return batchTestResultItems.value
})
const batchTestCanCancel = computed(() => {
  const status = batchTestResult.value?.status
  return status === 'queued' || status === 'running'
})
const batchTestProgressDescription = computed(() => {
  const job = batchTestResult.value
  if (!job) return t('admin.upstreamAccounts.batchTestResultDescription')
  if (job.status === 'queued') return t('admin.upstreamAccounts.batchTestStatusQueued')
  if (job.status === 'running') {
    return t('admin.upstreamAccounts.batchTestProgressDescription', { completed: job.completed, total: job.total })
  }
  if (job.status === 'cancelling') return t('admin.upstreamAccounts.batchTestStatusCancelling')
  if (job.status === 'cancelled') return t('admin.upstreamAccounts.batchTestStatusCancelled')
  if (job.status === 'failed') return job.error_message || t('admin.upstreamAccounts.batchTestFailedMessage')
  return t('admin.upstreamAccounts.batchTestResultDescription')
})
const accountGroupOptions = computed(() => {
  if (!accountGroupDialogItem.value) return []
  const platform = accountGroupPlatform.value
  return localGroups.value.filter(group => (!platform || group.platform === platform) && group.status === 'active')
})
const accountGroupPlatformTagClass = computed(() => platformTagClass(accountGroupPlatform.value))

const emptyTitle = computed(() => {
  return loadError.value ? t('admin.upstreamAccounts.emptyNoDefaultTitle') : t('admin.upstreamAccounts.emptyTitle')
})

const emptyDescription = computed(() => {
  return loadError.value || t('admin.upstreamAccounts.emptyDescription')
})

async function reload() {
  loading.value = true
  loadingRateGuardConfig.value = true
  loadError.value = ''
  try {
    const [preview, config, balance] = await Promise.all([
      adminAPI.upstreamAccountSync.getPreview(),
      adminAPI.upstreamAccountSync.getRateGuardConfig(),
      adminAPI.upstreamAccountSync.getBalanceConsumption(30)
    ])
    result.value = preview
    await syncMatchedAccounts(preview.items || [])
    applyRateGuardConfig(config)
    await syncRateGuardIgnoredAccounts(config.ignored_account_ids || [])
    balanceOverview.value = balance
    void loadLocalGroups()
  } catch (err) {
    const message = extractApiErrorMessage(err, t('admin.upstreamAccounts.loadFailed'))
    loadError.value = message
    result.value = null
    appStore.showError(message)
  } finally {
    loading.value = false
    loadingRateGuardConfig.value = false
  }
}

function openSyncConfirmDialog() {
  syncConfirmOptions.value = {
    create_missing: syncCreateItems.value.length > 0,
    update_existing: syncUpdateItems.value.length > 0,
    apply_rate_guard: syncRateGuardItems.value.length > 0
  }
  const selected: Record<string, boolean> = {}
  for (const item of [...syncCreateItems.value, ...syncUpdateItems.value]) {
    selected[syncConfirmSelectionKey(item)] = true
  }
  syncConfirmSelectedItems.value = selected
  showSyncConfirmDialog.value = true
}

function closeSyncConfirmDialog() {
  if (syncing.value) return
  showSyncConfirmDialog.value = false
}

async function submitSyncConfirm() {
  const payload = {
    create_missing: syncConfirmOptions.value.create_missing && syncCreateItems.value.some(syncConfirmItemSelected),
    update_existing: syncConfirmOptions.value.update_existing && syncUpdateItems.value.some(syncConfirmItemSelected),
    apply_rate_guard: syncConfirmOptions.value.apply_rate_guard && syncConfirmOptions.value.update_existing && syncRateGuardItems.value.some(syncConfirmItemSelected),
    selected_items: syncConfirmSelectedPayload()
  }
  if (!payload.create_missing && !payload.update_existing) {
    return
  }
  syncing.value = true
  try {
    const syncResult = await adminAPI.upstreamAccountSync.runSync(payload)
    result.value = syncResult
    lastSyncResult.value = syncResult
    await syncMatchedAccounts(syncResult.items || [])
    showSyncConfirmDialog.value = false
    showSyncResultDialog.value = syncResultItems.value.length > 0
    appStore.showSuccess(t('admin.upstreamAccounts.syncSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.syncFailed')))
  } finally {
    syncing.value = false
  }
}

function applyRateGuardConfig(config: UpstreamAccountRateGuardConfig) {
  const ignoredAccountIDs = normalizeRateGuardIgnoredAccountIDs(config.ignored_account_ids)
  rateGuardConfig.value = config
  rateGuardForm.value = {
    enabled: Boolean(config.enabled),
    interval_seconds: Number(config.interval_seconds) > 0 ? Number(config.interval_seconds) : 3600,
    ignored_account_ids: ignoredAccountIDs
  }
  rateGuardIgnoredInput.value = ignoredAccountIDs.join(', ')
}

function openRateGuardIgnoredDialog() {
  showRateGuardIgnoredDialog.value = true
  const ids = rateGuardIgnoredInputIDs.value ?? rateGuardIgnoredAccountIDs.value
  void syncRateGuardIgnoredAccounts(ids)
}

function closeRateGuardIgnoredDialog() {
  if (savingRateGuardConfig.value) return
  showRateGuardIgnoredDialog.value = false
  rateGuardIgnoredInput.value = rateGuardIgnoredAccountIDs.value.join(', ')
}

function removeRateGuardIgnoredAccount(id: number) {
  const ids = rateGuardIgnoredInputIDs.value ?? rateGuardIgnoredAccountIDs.value
  rateGuardIgnoredInput.value = ids.filter(accountID => accountID !== id).join(', ')
}

async function saveRateGuardConfigAndCloseIgnoredDialog() {
  const saved = await saveRateGuardConfig()
  if (saved) {
    showRateGuardIgnoredDialog.value = false
  }
}

async function saveRateGuardConfig() {
  if (!Number.isInteger(rateGuardForm.value.interval_seconds) || rateGuardForm.value.interval_seconds <= 0) {
    appStore.showError(t('admin.upstreamAccounts.invalidRateGuardInterval'))
    return false
  }
  const ignoredAccountIDs = parseRateGuardIgnoredInput(rateGuardIgnoredInput.value)
  if (!ignoredAccountIDs) {
    appStore.showError(t('admin.upstreamAccounts.invalidRateGuardIgnoredAccounts'))
    return false
  }
  savingRateGuardConfig.value = true
  try {
    const base = rateGuardConfig.value || {
      enabled: false,
      interval_seconds: 3600,
      ignored_account_ids: []
    }
    const config = await adminAPI.upstreamAccountSync.updateRateGuardConfig({
      ...base,
      enabled: rateGuardForm.value.enabled,
      interval_seconds: rateGuardForm.value.interval_seconds,
      ignored_account_ids: ignoredAccountIDs
    })
    applyRateGuardConfig(config)
    await refreshPreview()
    await syncRateGuardIgnoredAccounts(ignoredAccountIDs)
    appStore.showSuccess(t('admin.upstreamAccounts.rateGuardSaved'))
    return true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rateGuardSaveFailed')))
    return false
  } finally {
    savingRateGuardConfig.value = false
  }
}

async function toggleRateGuardIgnored(row: UpstreamAccountSyncItem) {
  const accountID = Number(row.matched_account_id)
  if (!Number.isSafeInteger(accountID) || accountID <= 0) return
  const parsedIDs = parseRateGuardIgnoredInput(rateGuardIgnoredInput.value)
  if (!parsedIDs) {
    appStore.showError(t('admin.upstreamAccounts.invalidRateGuardIgnoredAccounts'))
    return
  }
  const nextSet = new Set(parsedIDs)
  if (nextSet.has(accountID)) {
    nextSet.delete(accountID)
  } else {
    nextSet.add(accountID)
  }
  const nextIgnoredAccountIDs = normalizeRateGuardIgnoredAccountIDs(Array.from(nextSet))
  togglingRateGuardIgnoreId.value = accountID
  savingRateGuardConfig.value = true
  try {
    const base = rateGuardConfig.value || {
      enabled: false,
      interval_seconds: 3600,
      ignored_account_ids: []
    }
    const config = await adminAPI.upstreamAccountSync.updateRateGuardConfig({
      ...base,
      enabled: rateGuardForm.value.enabled,
      interval_seconds: rateGuardForm.value.interval_seconds,
      ignored_account_ids: nextIgnoredAccountIDs
    })
    applyRateGuardConfig(config)
    await refreshPreview()
    await syncRateGuardIgnoredAccounts(nextIgnoredAccountIDs)
    appStore.showSuccess(t('admin.upstreamAccounts.rateGuardSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rateGuardSaveFailed')))
  } finally {
    savingRateGuardConfig.value = false
    togglingRateGuardIgnoreId.value = null
  }
}

async function runRateGuardNow() {
  runningRateGuardNow.value = true
  try {
    const config = await adminAPI.upstreamAccountSync.runRateGuardNow()
    applyRateGuardConfig(config)
    const preview = await refreshPreview()
    await syncRateGuardIgnoredAccounts(config.ignored_account_ids || [])
    const remainingRisks = preview.summary?.rate_violation_count || 0
    if (remainingRisks > 0) {
      appStore.showWarning(t('admin.upstreamAccounts.rateGuardRunCompletedWithRisks', { count: remainingRisks }))
    } else {
      appStore.showSuccess(t('admin.upstreamAccounts.rateGuardRunSuccess'))
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rateGuardRunFailed')))
  } finally {
    runningRateGuardNow.value = false
  }
}

async function refreshPreview() {
  const preview = await adminAPI.upstreamAccountSync.getPreview()
  result.value = preview
  await syncMatchedAccounts(preview.items || [])
  return preview
}

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${formatRateNumber(n)}x` : '-'
}

function openStatDetailsDialog(key: StatCardKey) {
  activeStatDetailsKey.value = key
}

function closeStatDetailsDialog() {
  activeStatDetailsKey.value = null
}

function statDetailsRowKey(item: UpstreamAccountSyncItem) {
  return [
    activeStatDetailsKey.value || 'stat',
    item.provider_slug,
    item.upstream_key_name,
    item.matched_account_id || item.local_account_name || ''
  ].join('-')
}

function statDetailsActionLabel(item: UpstreamAccountSyncItem) {
  if (item.action === 'create') return t('admin.upstreamAccounts.actions.create')
  if (item.action === 'update') return t('admin.upstreamAccounts.actions.update')
  if (item.action === 'noop') return t('admin.upstreamAccounts.actions.noop')
  if (item.action === 'skip') return t('admin.upstreamAccounts.actions.skip')
  if (item.action === 'conflict') return t('admin.upstreamAccounts.actions.conflict')
  return item.action || '-'
}

function statDetailsActionClass(item: UpstreamAccountSyncItem) {
  if (item.action === 'create') return 'stat-details-action-create'
  if (item.action === 'update') return 'stat-details-action-update'
  if (item.action === 'conflict') return 'stat-details-action-conflict'
  if (item.action === 'skip') return 'stat-details-action-skip'
  return 'stat-details-action-muted'
}

function statDetailsGroupTags(item: UpstreamAccountSyncItem) {
  if (item.bound_groups?.length) {
    return item.bound_groups.map(group => ({
      key: `bound-${group.id}`,
      label: `${group.name} ${formatRate(group.rate_multiplier)}`,
      rateViolation: Boolean(group.rate_violation)
    }))
  }
  if (item.local_group_name || item.local_group_id) {
    const groupName = item.local_group_name || `#${item.local_group_id}`
    const hasRate = Number.isFinite(Number(item.local_rate_multiplier))
    return [{
      key: `local-${item.local_group_id || groupName}`,
      label: hasRate ? `${groupName} ${formatRate(item.local_rate_multiplier)}` : groupName,
      rateViolation: Boolean(item.rate_violation)
    }]
  }
  return []
}

function formatRateNumber(value: number) {
  const normalized = Math.abs(value) <= 0.0000001 ? 0 : value
  const [integerPart, fractionPart = ''] = normalized.toFixed(6).split('.')
  const trimmedFraction = fractionPart.replace(/0+$/, '')
  return `${integerPart}.${trimmedFraction.padEnd(2, '0')}`
}

function formatLatency(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n) || n < 0) return '-'
  if (n >= 1000) return `${(n / 1000).toFixed(1)}s`
  return `${Math.round(n)}ms`
}

function batchTestUpstreamRate(item: BatchAccountTestItem) {
  return batchTestRateByAccountId.value.get(item.account_id)
}

function orderBatchTestItemsFailureFirst(items: BatchAccountTestItem[]) {
  return items
    .map((item, index) => ({ item, index }))
    .sort((a, b) => {
      const priorityDelta = batchTestResultPriority(a.item) - batchTestResultPriority(b.item)
      if (priorityDelta !== 0) return priorityDelta
      const statusDelta = batchTestStatusSortValue(a.item.status) - batchTestStatusSortValue(b.item.status)
      if (statusDelta !== 0) return statusDelta
      return a.index - b.index
    })
    .map(entry => entry.item)
}

function batchTestResultPriority(item: BatchAccountTestItem) {
  if (batchTestIsFailedSchedulable(item)) return 0
  if (batchTestIsFailed(item)) return 1
  if (batchTestIsSkipped(item)) return 2
  return 3
}

function batchTestIsSkipped(item: BatchAccountTestItem) {
  return item.status === 'cancelled'
}

function batchTestIsFailed(item: BatchAccountTestItem) {
  return item.status !== 'success' && !batchTestIsSkipped(item)
}

function batchTestIsFailedSchedulable(item: BatchAccountTestItem) {
  return batchTestIsFailed(item) && batchTestItemSchedulable(item)
}

function batchTestIsFailedUnschedulable(item: BatchAccountTestItem) {
  return batchTestIsFailed(item) && !batchTestItemSchedulable(item)
}

function batchTestIsSuccessUnschedulable(item: BatchAccountTestItem) {
  return item.status === 'success' && !batchTestItemSchedulable(item) && !batchTestIsProviderDisabled(item)
}

function batchTestIsSuccessUpstreamDisabled(item: BatchAccountTestItem) {
  return item.status === 'success' && batchTestIsProviderDisabled(item)
}

function batchTestIsProviderDisabled(item: BatchAccountTestItem) {
  return batchTestProviderDisabledByAccountId.value.get(item.account_id) === true
}

function toggleBatchTestSort(key: BatchTestSortKey) {
  if (batchTestSortKey.value === key) {
    batchTestSortOrder.value = batchTestSortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    batchTestSortKey.value = key
    batchTestSortOrder.value = 'asc'
  }
}

function batchTestSortIndicator(key: BatchTestSortKey) {
  if (batchTestSortKey.value !== key) return ''
  return batchTestSortOrder.value === 'asc' ? '↑' : '↓'
}

function compareBatchTestItems(a: BatchAccountTestItem, b: BatchAccountTestItem, key: BatchTestSortKey) {
  const av = batchTestSortValue(a, key)
  const bv = batchTestSortValue(b, key)
  if (typeof av === 'number' && typeof bv === 'number') {
    return av - bv
  }
  return String(av).localeCompare(String(bv), undefined, { numeric: true, sensitivity: 'base' })
}

function batchTestSortValue(item: BatchAccountTestItem, key: BatchTestSortKey): string | number {
  if (key === 'account') return (item.account_name || `#${item.account_id}`).toLowerCase()
  if (key === 'platform') return (item.platform || '').toLowerCase()
  if (key === 'upstream_rate') return batchTestUpstreamRate(item) ?? Number.POSITIVE_INFINITY
  if (key === 'schedulable') return batchTestItemSchedulable(item) ? 1 : 0
  if (key === 'status') return batchTestStatusSortValue(item.status)
  if (key === 'latency') return Number.isFinite(Number(item.latency_ms)) ? Number(item.latency_ms) : Number.POSITIVE_INFINITY
  if (key === 'finished_at') {
    const time = Date.parse(item.finished_at || '')
    return Number.isFinite(time) ? time : Number.POSITIVE_INFINITY
  }
  return ''
}

function batchTestStatusSortValue(status: BatchAccountTestStatus | string) {
  const order: Record<string, number> = {
    success: 0,
    failed: 1,
    timeout: 2,
    not_found: 3,
    cancelled: 4
  }
  return order[status] ?? 99
}

async function loadLocalGroups() {
  try {
    localGroups.value = await adminAPI.groups.getAllIncludingInactive()
  } catch {
    localGroups.value = []
  }
}

async function syncMatchedAccounts(syncItems: UpstreamAccountSyncItem[]) {
  const matchedIDs = Array.from(
    new Set(
      syncItems
        .map(item => Number(item.matched_account_id))
        .filter(id => Number.isFinite(id) && id > 0)
    )
  )
  if (!matchedIDs.length) {
    matchedAccountsById.value = {}
    return
  }

  const entries = await Promise.allSettled(
    matchedIDs.map(async (accountId) => {
      const account = await adminAPI.accounts.getById(accountId)
      return [accountId, account] as const
    })
  )

  const nextMap: Record<number, Account> = {}
  const nextTestStatusMap: Record<number, AccountTestStatus> = {}
  for (const entry of entries) {
    if (entry.status !== 'fulfilled') continue
    const [accountId, account] = entry.value
    nextMap[accountId] = account
    if (account.last_test_status === 'success' || account.last_test_status === 'failed') {
      nextTestStatusMap[accountId] = account.last_test_status
    }
  }
  matchedAccountsById.value = nextMap
  const currentTestStatuses = accountTestStatusById.value
  for (const [accountId, status] of Object.entries(currentTestStatuses)) {
    const numericId = Number(accountId)
    if (!Number.isFinite(numericId) || numericId <= 0) continue
    if (!nextTestStatusMap[numericId]) {
      nextTestStatusMap[numericId] = status
    }
  }
  accountTestStatusById.value = nextTestStatusMap
}

async function syncRateGuardIgnoredAccounts(accountIDs: unknown) {
  const ignoredIDs = normalizeRateGuardIgnoredAccountIDs(accountIDs)
  const missingIDs = ignoredIDs.filter(accountID => !matchedAccountsById.value[accountID])
  if (!missingIDs.length) return

  const entries = await Promise.allSettled(
    missingIDs.map(async (accountID) => {
      const account = await adminAPI.accounts.getById(accountID)
      return [accountID, account] as const
    })
  )

  const nextMap = { ...matchedAccountsById.value }
  for (const entry of entries) {
    if (entry.status !== 'fulfilled') continue
    const [accountID, account] = entry.value
    nextMap[accountID] = account
  }
  matchedAccountsById.value = nextMap
}

async function ensureMatchedAccount(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isFinite(accountId) || accountId <= 0) return null
  const cached = matchedAccountsById.value[accountId]
  if (cached) return cached
  try {
    const account = await adminAPI.accounts.getById(accountId)
    matchedAccountsById.value = {
      ...matchedAccountsById.value,
      [accountId]: account
    }
    return account
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
    return null
  }
}

function getMatchedAccount(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isFinite(accountId) || accountId <= 0) return null
  return matchedAccountsById.value[accountId] || null
}

function isRateGuardIgnored(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isSafeInteger(accountId) || accountId <= 0) {
    return Boolean(row.rate_guard_ignored)
  }
  return Boolean(row.rate_guard_ignored) || rateGuardIgnoredAccountIDs.value.includes(accountId)
}

function isProviderDisabled(row: UpstreamAccountSyncItem) {
  const provider = syncProviderBySlug.value.get(row.provider_slug)
  return provider?.enabled === false
}

function isProviderEnabled(row: UpstreamAccountSyncItem) {
  return !isProviderDisabled(row)
}

function upstreamAccountHasNoGroups(row: UpstreamAccountSyncItem) {
  const boundGroups = row.bound_groups || []
  return !row.local_group_id && boundGroups.length === 0
}

function upstreamAccountTestFailed(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isFinite(accountId) || accountId <= 0) return false
  const status = accountTestStatusById.value[accountId] || matchedAccountsById.value[accountId]?.last_test_status
  return status === 'failed'
}

function upstreamAccountMatchesQuickFilter(row: UpstreamAccountSyncItem, filter: QuickFilterKey) {
  if (filter === 'update') return row.action === 'update'
  if (filter === 'conflict') return row.action === 'conflict'
  if (filter === 'risk') return row.rate_violation
  if (filter === 'ignored') return isRateGuardIgnored(row)
  if (filter === 'unbound') return upstreamAccountHasNoGroups(row)
  if (filter === 'failed') return upstreamAccountTestFailed(row)
  if (filter === 'enabled') return isProviderEnabled(row)
  if (filter === 'disabled') return isProviderDisabled(row)
  return true
}

function isSchedulableToggleDisabled(row: UpstreamAccountSyncItem) {
  const account = getMatchedAccount(row)
  if (!account) return true
  return togglingSchedulableId.value === account.id || isProviderDisabled(row)
}

function schedulableToggleTitle(row: UpstreamAccountSyncItem) {
  if (isProviderDisabled(row)) {
    return `${row.provider_name || row.provider_slug} ${t('common.disabled')}`
  }
  const account = getMatchedAccount(row)
  return account?.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')
}

function accountRowClass(row: UpstreamAccountSyncItem) {
  const classes = ['mobile-row-card']
  if (isProviderDisabled(row)) classes.push('provider-disabled-row')
  if (row.rate_violation) classes.push('risk-row')
  if (upstreamAccountTestFailed(row)) classes.push('test-failed-row')
  if (upstreamAccountHasNoGroups(row)) classes.push('unbound-row')
  return classes
}

function accountTestStatusLabel(status: AccountTestStatus | undefined) {
  if (status === 'testing') return t('admin.upstreamAccounts.testStatusTesting')
  if (status === 'failed') return t('admin.upstreamAccounts.testStatusFailed')
  if (status === 'success') return t('admin.upstreamAccounts.testStatusSuccess')
  return '-'
}

function upstreamAccountSourceSortValue(row: UpstreamAccountSyncItem) {
  return row.provider_name || row.provider_slug || ''
}

function upstreamAccountBalanceSortValue(row: UpstreamAccountSyncItem) {
  return getProviderBalance(row.provider_slug)
}

function upstreamAccountPrioritySortValue(row: UpstreamAccountSyncItem) {
  const priority = getMatchedAccount(row)?.priority
  return Number.isFinite(Number(priority)) ? Number(priority) : null
}

function isEditingPriority(row: UpstreamAccountSyncItem) {
  return editingPriorityAccountId.value === Number(row.matched_account_id)
}

function isPriorityEditable(row: UpstreamAccountSyncItem) {
  return getMatchedAccount(row) !== null
}

function startPriorityEdit(row: UpstreamAccountSyncItem) {
  const account = getMatchedAccount(row)
  if (!account || savingPriorityAccountId.value === account.id) return
  editingPriorityAccountId.value = account.id
  priorityDraft.value = account.priority
  void nextTick(() => {
    priorityInputRef.value?.focus()
    priorityInputRef.value?.select()
  })
}

function cancelPriorityEdit() {
  editingPriorityAccountId.value = null
  priorityDraft.value = null
}

async function savePriority(row: UpstreamAccountSyncItem) {
  if (!isEditingPriority(row)) return
  const account = getMatchedAccount(row)
  if (!account) {
    cancelPriorityEdit()
    return
  }
  if (savingPriorityAccountId.value === account.id) return
  if (priorityDraft.value === null || priorityDraft.value === '') {
    appStore.showError(t('admin.upstreamAccounts.priorityInvalid'))
    return
  }
  const nextPriority = Number(priorityDraft.value)
  if (!Number.isInteger(nextPriority) || nextPriority < 0) {
    appStore.showError(t('admin.upstreamAccounts.priorityInvalid'))
    return
  }
  const normalizedPriority = nextPriority
  if (normalizedPriority === account.priority) {
    cancelPriorityEdit()
    return
  }

  savingPriorityAccountId.value = account.id
  try {
    const updated = await adminAPI.accounts.update(account.id, { priority: normalizedPriority })
    updateMatchedAccount(updated)
    cancelPriorityEdit()
    appStore.showSuccess(t('admin.upstreamAccounts.prioritySaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.prioritySaveFailed')))
  } finally {
    savingPriorityAccountId.value = null
  }
}

function upstreamAccountStatusSortValue(row: UpstreamAccountSyncItem) {
  return getMatchedAccount(row)?.status || ''
}

function upstreamAccountSchedulableSortValue(row: UpstreamAccountSyncItem) {
  const account = getMatchedAccount(row)
  if (!account || isProviderDisabled(row)) return null
  return account.schedulable ? 1 : 0
}

function upstreamAccountTestStatusSortValue(row: UpstreamAccountSyncItem) {
  const status = accountTestStatusById.value[row.matched_account_id || 0]
  if (status === 'testing') return 0
  if (status === 'success') return 1
  if (status === 'failed') return 2
  return null
}

function batchTestStatusLabel(status: BatchAccountTestStatus | string) {
  if (status === 'success') return t('admin.upstreamAccounts.testStatusSuccess')
  if (status === 'timeout') return t('admin.upstreamAccounts.batchTestStatusTimeout')
  if (status === 'not_found') return t('admin.upstreamAccounts.batchTestStatusNotFound')
  if (status === 'cancelled') return t('admin.upstreamAccounts.batchTestStatusCancelled')
  return t('admin.upstreamAccounts.testStatusFailed')
}

function batchTestStatusClass(status: BatchAccountTestStatus | string) {
  if (status === 'success') return 'test-status-success'
  if (status === 'cancelled') return 'test-status-skipped'
  if (status === 'timeout') return 'test-status-testing'
  return 'test-status-failed'
}

function batchTestResultCardTone(item: BatchAccountTestItem) {
  if (batchTestIsFailed(item)) return 'failed'
  if (batchTestIsSkipped(item)) return 'skipped'
  return ''
}

function batchTestItemSchedulable(item: BatchAccountTestItem) {
  if (typeof item.schedulable === 'boolean') return item.schedulable
  const account = matchedAccountsById.value[item.account_id]
  return account?.schedulable === true
}

function isBatchTestJobTerminal(status: BatchAccountTestJobStatus | undefined) {
  return status === 'completed' || status === 'cancelled' || status === 'failed'
}

function openBatchTestConfigDialog() {
  const accountIds = batchTestAccountIds.value
  if (!accountIds.length || batchTesting.value) return
  if (accountIds.length > MAX_BATCH_TEST_ACCOUNTS) {
    appStore.showWarning(t('admin.upstreamAccounts.batchTestTooManyAccounts', { max: MAX_BATCH_TEST_ACCOUNTS }))
    return
  }

  const nextModels: Record<string, string> = {}
  for (const option of batchTestPlatformOptions.value) {
    nextModels[option.platform] = batchTestModelByPlatform.value[option.platform] || ''
  }
  batchTestModelByPlatform.value = nextModels
  showBatchTestConfigDialog.value = true
  void loadBatchTestPlatformModels()
}

function closeBatchTestConfigDialog() {
  if (batchTesting.value) return
  showBatchTestConfigDialog.value = false
}

async function loadBatchTestPlatformModels() {
  const options = batchTestPlatformOptions.value
  await Promise.allSettled(options.map(async option => {
    if (
      batchTestModelSourceAccountByPlatform.value[option.platform] === option.accountId &&
      batchTestModelOptionsByPlatform.value[option.platform]?.length
    ) {
      return
    }
    batchTestModelLoadingByPlatform.value = {
      ...batchTestModelLoadingByPlatform.value,
      [option.platform]: true
    }
    try {
      const models = await adminAPI.accounts.getAvailableModels(option.accountId)
      batchTestModelOptionsByPlatform.value = {
        ...batchTestModelOptionsByPlatform.value,
        [option.platform]: models || []
      }
      batchTestModelSourceAccountByPlatform.value = {
        ...batchTestModelSourceAccountByPlatform.value,
        [option.platform]: option.accountId
      }
    } catch {
      batchTestModelOptionsByPlatform.value = {
        ...batchTestModelOptionsByPlatform.value,
        [option.platform]: []
      }
    } finally {
      batchTestModelLoadingByPlatform.value = {
        ...batchTestModelLoadingByPlatform.value,
        [option.platform]: false
      }
    }
  }))
}

async function confirmBatchAccountTest() {
  showBatchTestConfigDialog.value = false
  await runBatchAccountTest()
}

function selectedBatchTestModelsByPlatform() {
  const models: Record<string, string> = {}
  for (const option of batchTestPlatformOptions.value) {
    const modelID = batchTestModelByPlatform.value[option.platform]?.trim()
    if (modelID) {
      models[option.platform] = modelID
    }
  }
  return models
}

async function runBatchAccountTest() {
  const accountIds = batchTestAccountIds.value
  if (!accountIds.length || batchTesting.value) return
  if (accountIds.length > MAX_BATCH_TEST_ACCOUNTS) {
    appStore.showWarning(t('admin.upstreamAccounts.batchTestTooManyAccounts', { max: MAX_BATCH_TEST_ACCOUNTS }))
    return
  }

  batchTesting.value = true
  batchTestResult.value = null
  batchTestResultFilter.value = 'all'
  showBatchTestResultDialog.value = false
  clearBatchTestPollTimer()
  const pollToken = batchTestPollToken.value + 1
  batchTestPollToken.value = pollToken
  accountTestStatusById.value = {
    ...accountTestStatusById.value,
    ...Object.fromEntries(accountIds.map(id => [id, 'testing' as AccountTestStatus]))
  }
  try {
    const modelIDsByPlatform = selectedBatchTestModelsByPlatform()
    const job = await adminAPI.accounts.batchTestAccounts({
      account_ids: accountIds,
      ...(Object.keys(modelIDsByPlatform).length > 0 ? { model_ids_by_platform: modelIDsByPlatform } : {}),
      concurrency: 3,
      timeout_per_account_seconds: 90,
      timeout_seconds: BATCH_TEST_TOTAL_TIMEOUT_SECONDS
    })
    batchTestResult.value = job
    showBatchTestResultDialog.value = true
    if (isBatchTestJobTerminal(job.status)) {
      await finishBatchAccountTestJob(job)
    } else {
      scheduleBatchTestPoll(job.job_id, pollToken)
    }
  } catch (err) {
    batchTesting.value = false
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.batchTestFailedMessage')))
  }
}

function scheduleBatchTestPoll(jobId: string, pollToken: number) {
  clearBatchTestPollTimer()
  batchTestPollTimer.value = setTimeout(async () => {
    if (pollToken !== batchTestPollToken.value) return
    try {
      const job = await adminAPI.accounts.getBatchTestJob(jobId)
      if (pollToken !== batchTestPollToken.value) return
      batchTestResult.value = job
      if (isBatchTestJobTerminal(job.status)) {
        await finishBatchAccountTestJob(job)
      } else {
        scheduleBatchTestPoll(jobId, pollToken)
      }
    } catch (err) {
      batchTesting.value = false
      appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.batchTestFailedMessage')))
    }
  }, BATCH_TEST_POLL_INTERVAL_MS)
}

async function cancelBatchAccountTest() {
  const jobId = batchTestResult.value?.job_id
  if (!jobId || !batchTestCanCancel.value) return
  try {
    const job = await adminAPI.accounts.cancelBatchTestJob(jobId)
    batchTestResult.value = job
    if (isBatchTestJobTerminal(job.status)) {
      await finishBatchAccountTestJob(job)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.batchTestFailedMessage')))
  }
}

async function finishBatchAccountTestJob(job: BatchAccountTestJob) {
  clearBatchTestPollTimer()
  batchTesting.value = false
  batchTestResult.value = job
  const nextStatuses = { ...accountTestStatusById.value }
  for (const item of job.results || []) {
    nextStatuses[item.account_id] = item.status === 'success' ? 'success' : 'failed'
  }
  accountTestStatusById.value = nextStatuses
  await refreshBatchTestAccountsSilently((job.results || []).map(item => item.account_id))
  if (job.status === 'cancelled') {
    appStore.showWarning(t('admin.upstreamAccounts.batchTestCancelledMessage'))
  } else if (job.status === 'failed') {
    appStore.showError(job.error_message || t('admin.upstreamAccounts.batchTestFailedMessage'))
  } else if (job.failed > 0) {
    appStore.showWarning(t('admin.upstreamAccounts.batchTestCompletedWithFailures', { failed: job.failed, total: job.total }))
  } else {
    appStore.showSuccess(t('admin.upstreamAccounts.batchTestSuccessMessage', { total: job.total }))
  }
}

async function toggleBatchTestItemSchedulable(item: BatchAccountTestItem) {
  const accountId = item.account_id
  if (!Number.isFinite(accountId) || accountId <= 0) return
  const nextSchedulable = !batchTestItemSchedulable(item)
  togglingSchedulableId.value = accountId
  try {
    const updated = await adminAPI.accounts.setSchedulable(accountId, nextSchedulable)
    if (updated) {
      updateMatchedAccount(updated)
    }
    if (batchTestResult.value) {
      batchTestResult.value = {
        ...batchTestResult.value,
        results: (batchTestResult.value.results || []).map(resultItem => resultItem.account_id === accountId
          ? { ...resultItem, schedulable: updated?.schedulable ?? nextSchedulable }
          : resultItem)
      }
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.accounts.failedToToggleSchedulable')))
  } finally {
    togglingSchedulableId.value = null
  }
}

async function accountFromBatchTestItem(item: BatchAccountTestItem): Promise<Account | null> {
  const accountId = item.account_id
  if (!Number.isFinite(accountId) || accountId <= 0) return null
  const cached = matchedAccountsById.value[accountId]
  if (cached) return cached
  try {
    const account = await adminAPI.accounts.getById(accountId)
    updateMatchedAccount(account)
    return account
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
    return null
  }
}

async function openBatchTestAccountEditDialog(item: BatchAccountTestItem) {
  const account = await accountFromBatchTestItem(item)
  if (!account) return
  try {
    await loadAccountEditOptions()
    editingAccount.value = account
    showEditAccountModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
  }
}

async function openBatchTestAccountDeleteDialog(item: BatchAccountTestItem) {
  const account = await accountFromBatchTestItem(item)
  if (!account) return
  deletingAccount.value = account
  showDeleteAccountDialog.value = true
}

function clearBatchTestPollTimer() {
  if (batchTestPollTimer.value) {
    clearTimeout(batchTestPollTimer.value)
    batchTestPollTimer.value = null
  }
}

async function refreshBatchTestAccountsSilently(accountIds: number[]) {
  const uniqueIds = Array.from(new Set(accountIds.filter(id => Number.isFinite(id) && id > 0)))
  let cursor = 0
  const workers = Array.from({ length: Math.min(BATCH_TEST_REFRESH_CONCURRENCY, uniqueIds.length) }, async () => {
    while (cursor < uniqueIds.length) {
      const id = uniqueIds[cursor]
      cursor += 1
      await refreshMatchedAccountSilently(id)
    }
  })
  await Promise.all(workers)
}

function sourceToneClass(row: UpstreamAccountSyncItem) {
  if (row.rate_violation) return 'source-line-red'
  if (row.provider_fetch_error) return 'source-line-amber'
  const slug = (row.provider_slug || row.provider_name || '').toLowerCase()
  if (slug.includes('niko')) return 'source-line-blue'
  return 'source-line-emerald'
}

function providerToneClass(providerSlug: string | undefined, target: 'card' | 'tag') {
  const tones = [
    'sky',
    'emerald',
    'violet',
    'cyan',
    'rose',
    'amber',
    'indigo',
    'teal'
  ]
  const slug = providerSlug?.trim() || 'default'
  let hash = 0
  for (let i = 0; i < slug.length; i++) {
    hash = (hash * 31 + slug.charCodeAt(i)) >>> 0
  }
  const tone = tones[hash % tones.length]
  return `${target === 'card' ? 'source-card' : 'tag-provider'}-${tone}`
}

function matchedAccountPlatform(row: UpstreamAccountSyncItem) {
  return getMatchedAccount(row)?.platform || inferAccountGroupPlatform(row) || ''
}

function matchedAccountPlatformTagClass(row: UpstreamAccountSyncItem) {
  return platformTagClass(matchedAccountPlatform(row))
}

function matchedAccountPlatformTextClass(row: UpstreamAccountSyncItem) {
  return platformTextToneClass(matchedAccountPlatform(row))
}

function platformTagClass(platform: string | undefined) {
  if (platform === 'anthropic') return 'platform-tag-anthropic'
  if (platform === 'openai') return 'platform-tag-openai'
  if (platform === 'antigravity') return 'platform-tag-antigravity'
  if (platform === 'gemini') return 'platform-tag-gemini'
  return 'platform-tag-default'
}

function platformTextToneClass(platform: string | undefined) {
  if (platform === 'anthropic') return 'platform-text-anthropic'
  if (platform === 'openai') return 'platform-text-openai'
  if (platform === 'antigravity') return 'platform-text-antigravity'
  if (platform === 'gemini') return 'platform-text-gemini'
  return 'platform-text-default'
}

async function openAccountGroupDialog(row: UpstreamAccountSyncItem) {
  accountGroupDialogItem.value = row
  accountGroupIds.value = (row.bound_groups || [])
    .map((group: UpstreamAccountSyncBoundGroup) => Number(group.id))
    .filter((id) => Number.isFinite(id))
  accountGroupPlatform.value = inferAccountGroupPlatform(row)

  const account = await ensureMatchedAccount(row)
  if (account) {
    accountGroupPlatform.value = account.platform
  }
}

function closeAccountGroupDialog() {
  if (savingAccountGroupId.value) return
  accountGroupDialogItem.value = null
  accountGroupIds.value = []
  accountGroupPlatform.value = undefined
}

async function saveAccountGroups() {
  const row = accountGroupDialogItem.value
  if (!row?.matched_account_id) return
  savingAccountGroupId.value = row.matched_account_id
  try {
    const updated = await adminAPI.accounts.update(row.matched_account_id, { group_ids: accountGroupIds.value })
    if (updated) updateMatchedAccount(updated)
    const savedGroupIDs = [...accountGroupIds.value]
    applyAccountGroupUpdateToPreview(row.matched_account_id, savedGroupIDs, updated || null)
    accountGroupDialogItem.value = null
    accountGroupIds.value = []
    accountGroupPlatform.value = undefined
    appStore.showSuccess(t('admin.upstreamAccounts.boundGroupsSaved'))
    try {
      await refreshPreview()
      applyAccountGroupUpdateToPreview(row.matched_account_id, savedGroupIDs, updated || null)
    } catch (err) {
      appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadFailed')))
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.boundGroupsSaveFailed')))
  } finally {
    savingAccountGroupId.value = null
  }
}

async function clearAccountGroups() {
  accountGroupIds.value = []
  await saveAccountGroups()
}

async function openAccountTestDialog(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isFinite(accountId) || accountId <= 0) return
  testingAccountId.value = accountId
  try {
    testingAccount.value = await ensureMatchedAccount(row)
    if (!testingAccount.value) return
    showTestModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.testConnectionFailed')))
  } finally {
    testingAccountId.value = null
  }
}

function closeTestModal() {
  showTestModal.value = false
  testingAccount.value = null
}

async function loadAccountEditOptions() {
  const [proxies, groups] = await Promise.all([
    adminAPI.proxies.getAll(),
    adminAPI.groups.getAll()
  ])
  accountProxies.value = proxies
  accountEditGroups.value = groups
}

async function openCreateAccountDialog(row?: UpstreamAccountSyncItem) {
  try {
    createAccountInitialValues.value = row ? createAccountInitialValuesFromSyncItem(row) : null
    await loadAccountEditOptions()
    showCreateAccountModal.value = true
  } catch (err) {
    createAccountInitialValues.value = null
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
  }
}

function closeCreateAccountDialog() {
  showCreateAccountModal.value = false
  createAccountInitialValues.value = null
}

async function handleAccountCreated() {
  showCreateAccountModal.value = false
  createAccountInitialValues.value = null
  await reload()
}

function createAccountInitialValuesFromSyncItem(row: UpstreamAccountSyncItem): CreateAccountInitialValues {
  const groupIDs = numberArray([row.local_group_id])
  return {
    name: row.local_account_name || upstreamLocalAccountName(row),
    platform: 'openai',
    type: 'apikey',
    base_url: row.upstream_base_url || row.provider_base_url || undefined,
    api_key: row.upstream_api_key || '',
    group_ids: groupIDs
  }
}

function upstreamLocalAccountName(row: UpstreamAccountSyncItem) {
  const providerPrefix = row.provider_slug || row.provider_name || ''
  const keyName = row.upstream_key_name || ''
  if (!providerPrefix) return keyName
  if (!keyName) return providerPrefix
  return `${providerPrefix.replace(/-+$/g, '')}-${keyName.replace(/^-+/g, '')}`
}

async function openAccountEditDialog(row: UpstreamAccountSyncItem) {
  const account = await ensureMatchedAccount(row)
  if (!account) return
  try {
    await loadAccountEditOptions()
    editingAccount.value = account
    showEditAccountModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
  }
}

function closeAccountEditDialog() {
  showEditAccountModal.value = false
  editingAccount.value = null
}

function handleAccountUpdated(account: Account) {
  updateMatchedAccount(account)
  if (editingAccount.value?.id === account.id) {
    editingAccount.value = account
  }
  showEditAccountModal.value = false
}

async function openAccountDeleteDialog(row: UpstreamAccountSyncItem) {
  const account = await ensureMatchedAccount(row)
  if (!account) return
  deletingAccount.value = account
  showDeleteAccountDialog.value = true
}

function closeAccountDeleteDialog() {
  showDeleteAccountDialog.value = false
  deletingAccount.value = null
}

async function confirmDeleteAccount() {
  if (!deletingAccount.value) return
  const deletedAccountId = deletingAccount.value.id
  try {
    await adminAPI.accounts.delete(deletedAccountId)
    closeAccountDeleteDialog()
    removeBatchTestResultItem(deletedAccountId)
    await reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.accounts.deleteFailed')))
  }
}

function removeBatchTestResultItem(accountId: number) {
  if (!batchTestResult.value) return
  const currentResults = batchTestResult.value.results || []
  const removedItem = currentResults.find(item => item.account_id === accountId)
  if (!removedItem) return
  const results = currentResults.filter(item => item.account_id !== accountId)
  batchTestResult.value = {
    ...batchTestResult.value,
    results,
    total: Math.max(0, batchTestResult.value.total - 1),
    completed: Math.max(0, batchTestResult.value.completed - 1),
    success: Math.max(0, batchTestResult.value.success - (removedItem.status === 'success' ? 1 : 0)),
    failed: Math.max(0, batchTestResult.value.failed - (removedItem.status === 'success' ? 0 : 1))
  }
}

async function handleAccountTestResult(payload: { accountId: number; status: AccountTestStatus }) {
  accountTestStatusById.value = {
    ...accountTestStatusById.value,
    [payload.accountId]: payload.status
  }
  if (payload.status === 'success' || payload.status === 'failed') {
    await refreshMatchedAccount(payload.accountId)
  }
}

function updateMatchedAccount(account: Account) {
  matchedAccountsById.value = {
    ...matchedAccountsById.value,
    [account.id]: account
  }
}

function applyAccountGroupUpdateToPreview(accountId: number, groupIDs: number[], account: Account | null = null) {
  const current = result.value
  if (!current) return
  const numericAccountId = Number(accountId)
  if (!Number.isFinite(numericAccountId) || numericAccountId <= 0) return

  const items = (current.items || []).map(item => {
    if (Number(item.matched_account_id) !== numericAccountId) return item
    const boundGroups = buildPreviewBoundGroups(groupIDs, account, item.upstream_rate_multiplier)
    const rateViolationGroups = boundGroups.filter(group => group.rate_violation)
    return {
      ...item,
      bound_groups: boundGroups,
      local_group_id: boundGroups[0]?.id,
      local_group_name: boundGroups[0]?.name || '',
      local_rate_multiplier: boundGroups[0]?.rate_multiplier,
      rate_violation: rateViolationGroups.length > 0,
      unbound_group_ids: rateViolationGroups.map(group => group.id),
      unbound_group_names: rateViolationGroups.map(group => group.name),
    }
  })
  result.value = {
    ...current,
    items
  }
}

function buildPreviewBoundGroups(groupIDs: number[], account: Account | null, upstreamRateMultiplier: number | undefined): UpstreamAccountSyncBoundGroup[] {
  const accountGroups = account?.groups || []
  const upstreamRate = Number(upstreamRateMultiplier)
  return groupIDs
    .map(groupID => {
      const numericGroupID = Number(groupID)
      if (!Number.isFinite(numericGroupID) || numericGroupID <= 0) return null
      const group = accountGroups.find(item => Number(item.id) === numericGroupID)
        || localGroups.value.find(item => Number(item.id) === numericGroupID)
      const rateMultiplier = Number(group?.rate_multiplier) || 0
      return {
        id: numericGroupID,
        name: group?.name || `#${numericGroupID}`,
        rate_multiplier: rateMultiplier,
        rate_violation: Number.isFinite(upstreamRate) && rateMultiplier < upstreamRate
      }
    })
    .filter((group): group is UpstreamAccountSyncBoundGroup => Boolean(group))
}

async function handleToggleSchedulable(account: Account, row?: UpstreamAccountSyncItem) {
  if (row && isProviderDisabled(row)) return
  togglingSchedulableId.value = account.id
  try {
    const updated = await adminAPI.accounts.setSchedulable(account.id, !account.schedulable)
    updateMatchedAccount(updated || { ...account, schedulable: !account.schedulable })
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.accounts.failedToToggleSchedulable')))
  } finally {
    togglingSchedulableId.value = null
  }
}

async function refreshMatchedAccount(accountId: number) {
  if (!Number.isFinite(accountId) || accountId <= 0) return
  try {
    const account = await adminAPI.accounts.getById(accountId)
    updateMatchedAccount(account)
    if (account.last_test_status === 'success' || account.last_test_status === 'failed') {
      accountTestStatusById.value = {
        ...accountTestStatusById.value,
        [accountId]: account.last_test_status
      }
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
  }
}

async function refreshMatchedAccountSilently(accountId: number) {
  if (!Number.isFinite(accountId) || accountId <= 0) return
  try {
    const account = await adminAPI.accounts.getById(accountId)
    updateMatchedAccount(account)
    if (account.last_test_status === 'success' || account.last_test_status === 'failed') {
      accountTestStatusById.value = {
        ...accountTestStatusById.value,
        [accountId]: account.last_test_status
      }
    }
  } catch {
    // The batch result dialog still shows the authoritative test result.
  }
}

function handleShowTempUnsched(account: Account) {
  tempUnschedAccount.value = account
  showTempUnsched.value = true
}

function handleTempUnschedReset(updated: Account) {
  tempUnschedAccount.value = null
  showTempUnsched.value = false
  updateMatchedAccount(updated)
}

function closeTempUnschedModal() {
  tempUnschedAccount.value = null
  showTempUnsched.value = false
}

function groupNameById(groupID: number) {
  const group = localGroups.value.find(item => item.id === groupID)
  return group ? `${group.name} ${formatRate(group.rate_multiplier)}` : `#${groupID}`
}

function inferAccountGroupPlatform(row: UpstreamAccountSyncItem): GroupPlatform | undefined {
  const groupIDs = [
    row.local_group_id,
    ...(row.bound_groups || []).map((group) => group.id)
  ]
    .map((id) => Number(id))
    .filter((id) => Number.isFinite(id))
  for (const groupID of groupIDs) {
    const group = localGroups.value.find(item => item.id === groupID)
    if (group?.platform) return group.platform
  }
  return undefined
}

function rateToneClass(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return 'rate-muted'
  if (n >= 0.4) return 'rate-red'
  if (n >= 0.3) return 'rate-deep-orange'
  if (n >= 0.2) return 'rate-orange'
  if (n > 0.1) return 'rate-green'
  return 'rate-deep-green'
}

function rateProgressWidth(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n) || n <= 0) return '0%'
  return `${Math.min(100, Math.max(8, (n / 0.5) * 100))}%`
}

function triggerClass(triggerSource: string | undefined) {
  if (triggerSource === 'scheduled_rate_guard') return 'trigger-scheduled'
  if (triggerSource === 'manual_rate_guard') return 'trigger-guard'
  return 'trigger-sync'
}

function conflictAccountRates(account: UpstreamAccountSyncConflictAccount) {
  return (account.bound_groups || [])
    .map(group => formatRate(group.rate_multiplier))
    .join(' / ')
}

function toRFC3339(value: string) {
  if (!value) return value
  const parsed = new Date(value)
  if (!Number.isFinite(parsed.getTime())) return value
  return parsed.toISOString().replace(/\.\d+Z$/, 'Z')
}

function numberArray(value: unknown): number[] {
  return Array.isArray(value) ? value.filter((item): item is number => Number.isFinite(Number(item))).map(Number) : []
}

function normalizeRateGuardIgnoredAccountIDs(value: unknown): number[] {
  const raw = Array.isArray(value) ? value : []
  const seen = new Set<number>()
  for (const item of raw) {
    const id = Number(item)
    if (!Number.isSafeInteger(id) || id <= 0) continue
    seen.add(id)
  }
  return Array.from(seen).sort((a, b) => a - b)
}

function parseRateGuardIgnoredInput(value: string): number[] | null {
  const text = String(value || '').trim()
  if (!text) return []
  const tokens = text.split(/[\s,，、]+/).map(token => token.trim()).filter(Boolean)
  const ids: number[] = []
  for (const token of tokens) {
    if (!/^\d+$/.test(token)) return null
    const id = Number(token)
    if (!Number.isSafeInteger(id) || id <= 0) return null
    ids.push(id)
  }
  return normalizeRateGuardIgnoredAccountIDs(ids)
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.map(String).filter(Boolean) : []
}

function conflictAccountTitle(account: UpstreamAccountSyncConflictAccount) {
  const groups = (account.bound_groups || [])
    .map(group => `${group.name} ${formatRate(group.rate_multiplier)}`)
    .join(', ')
  return groups ? `${account.name}: ${groups}` : account.name
}

function syncConfirmItemKey(item: UpstreamAccountSyncItem, scope: string) {
  return `${scope}-${item.provider_slug}-${item.upstream_key_name}-${item.matched_account_id || item.local_account_name}`
}

function syncConfirmSelectionKey(item: UpstreamAccountSyncItem) {
  return `${item.provider_slug}\u0000${item.upstream_key_name}`
}

function syncConfirmItemSelected(item: UpstreamAccountSyncItem) {
  return Boolean(syncConfirmSelectedItems.value[syncConfirmSelectionKey(item)])
}

function syncConfirmItemDataTest(item: UpstreamAccountSyncItem, scope: string) {
  return `sync-confirm-item-${scope}-${slugForDataTest(item.provider_slug)}-${slugForDataTest(item.upstream_key_name)}`
}

function syncConfirmSelectedPayload() {
  const selected = [...syncCreateItems.value, ...syncUpdateItems.value]
    .filter(syncConfirmItemSelected)
    .map(item => ({
      provider_slug: item.provider_slug,
      upstream_key_name: item.upstream_key_name,
      create_missing: item.action === 'create' && syncConfirmOptions.value.create_missing,
      update_existing: item.action === 'update' && syncConfirmOptions.value.update_existing,
      apply_rate_guard: item.action === 'update' && syncConfirmOptions.value.update_existing && syncConfirmOptions.value.apply_rate_guard && item.rate_violation
    }))
  return selected
}

function slugForDataTest(value: string) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function syncConfirmUnboundGroups(item: UpstreamAccountSyncItem) {
  const names = stringArray(item.unbound_group_names)
  if (names.length > 0) return names.join(', ')
  const ids = numberArray(item.unbound_group_ids)
  return ids.length > 0 ? ids.map(id => `#${id}`).join(', ') : '-'
}

function closeSyncResultDialog() {
  showSyncResultDialog.value = false
}

function closeBatchTestResultDialog() {
  showBatchTestResultDialog.value = false
}

function syncResultUnboundGroups(item: UpstreamAccountSyncItem) {
  const names = stringArray(item.execution?.unbound_group_names)
  if (names.length > 0) return names.join(', ')
  const ids = numberArray(item.execution?.unbound_group_ids)
  return ids.length > 0 ? ids.map(id => `#${id}`).join(', ') : '-'
}

function syncConfirmUpdateDetails(item: UpstreamAccountSyncItem) {
  if (item.change_details?.length) {
    return item.change_details.map(syncConfirmChangeDetailLabel).filter(Boolean)
  }
  const details = [t('admin.upstreamAccounts.syncConfirmRefreshCredentials')]
  if (item.local_group_id && !(item.bound_groups || []).some(group => Number(group.id) === Number(item.local_group_id))) {
    details.push(t('admin.upstreamAccounts.syncConfirmBindGroup', { group: item.local_group_name || `#${item.local_group_id}` }))
  }
  const unboundGroups = syncConfirmUnboundGroups(item)
  if (item.rate_violation && unboundGroups !== '-') {
    details.push(t('admin.upstreamAccounts.syncConfirmUnbindGroupsDetail', { groups: unboundGroups }))
  }
  return details
}

function syncConfirmChangeDetailLabel(detail: UpstreamAccountSyncChangeDetail) {
  const label = detail.label || detail.field || detail.kind
  if (detail.kind === 'credential') {
    return `${label}: ${detail.before || '-'} -> ${detail.after || '-'}`
  }
  const groupNames = stringArray(detail.group_names)
  if (detail.kind === 'group_bind') {
    return groupNames.length > 0
      ? `${label}: ${groupNames.join(', ')}`
      : `${label}: ${numberArray(detail.group_ids).map(id => `#${id}`).join(', ') || '-'}`
  }
  if (detail.kind === 'group_unbind') {
    return groupNames.length > 0
      ? `${label}: ${groupNames.join(', ')}`
      : `${label}: ${numberArray(detail.group_ids).map(id => `#${id}`).join(', ') || '-'}`
  }
  return label
}

function upstreamAccountSyncTriggerSourceLabel(triggerSource: string | undefined) {
  if (triggerSource === 'scheduled_rate_guard') return t('admin.upstreamAccounts.triggerScheduledRateGuard')
  if (triggerSource === 'manual_rate_guard') return t('admin.upstreamAccounts.triggerManualRateGuard')
  return t('admin.upstreamAccounts.triggerManualSync')
}

function isSyncLogHandled(entry: UpstreamAccountSyncLogEntry) {
  return Boolean(entry.handled)
}

async function markSyncLogHandled(entry: UpstreamAccountSyncLogEntry) {
  try {
    const records = await adminAPI.upstreamAccountSync.markRecordHandled(entry.key)
    if (result.value) {
      result.value = {
        ...result.value,
        records
      }
    }
    appStore.showSuccess(t('admin.upstreamAccounts.syncLogMarkedHandled', '同步日志已标记为已处理'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.syncLogMarkHandledFailed', '标记同步日志失败')))
  }
}

function openSyncLogsDialog() {
  showSyncLogsDialog.value = true
}

function closeSyncLogsDialog() {
  showSyncLogsDialog.value = false
}

function getProviderBalance(providerSlug: string): number | null {
  if (!balanceOverview.value?.summaries) return null
  const summary = balanceOverview.value.summaries[providerSlug]
  return summary ? summary.current_balance : null
}

function getProviderConsumption(providerSlug: string): number | null {
  if (!balanceOverview.value?.summaries) return null
  const summary = balanceOverview.value.summaries[providerSlug]
  return summary ? summary.today_consumption : null
}

function formatMoney(value: number): string {
  if (value >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  } else if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}

function openTrendModal(providerSlug: string, providerName: string) {
  trendProviderSlug.value = providerSlug
  trendProviderName.value = providerName
  showTrendModal.value = true
}

function closeTrendModal() {
  showTrendModal.value = false
}

function updateMobileBackToFiltersVisibility() {
  showMobileBackToFilters.value = window.innerWidth <= 768 && window.scrollY > 560
}

function scrollToUpstreamFilters() {
  const tableWrapper = document.querySelector<HTMLElement>('.upstream-accounts-page .table-wrapper')
  if (tableWrapper) {
    tableWrapper.scrollTop = 0
  }
  const target = document.querySelector<HTMLElement>('.upstream-accounts-page .filter-row')
  const top = target ? target.getBoundingClientRect().top + window.scrollY - 76 : 0
  window.scrollTo({
    top: Math.max(0, top),
    behavior: 'smooth'
  })
}

async function focusAutomationSettingsFromQuery() {
  if (route.query.automation !== 'rate-guard') return
  rateGuardAutomationTarget.value = true
  await nextTick()
  document.querySelector<HTMLElement>('[data-test="rate-guard-panel"]')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  const query = { ...route.query }
  delete query.automation
  await router.replace({ query })
}

onMounted(async () => {
  await reload()
  await focusAutomationSettingsFromQuery()
  updateMobileBackToFiltersVisibility()
  window.addEventListener('scroll', updateMobileBackToFiltersVisibility, { passive: true })
  window.addEventListener('resize', updateMobileBackToFiltersVisibility)
})
onBeforeUnmount(() => {
  batchTestPollToken.value += 1
  clearBatchTestPollTimer()
  window.removeEventListener('scroll', updateMobileBackToFiltersVisibility)
  window.removeEventListener('resize', updateMobileBackToFiltersVisibility)
})
</script>

<style scoped>
.upstream-accounts-page {
  width: 100%;
  max-width: none;
  margin: 0;
}

.upstream-accounts-page :deep(.table-page-layout) {
  gap: 16px;
  width: 100%;
  max-width: none;
  height: calc(100vh - 64px - 4rem);
  min-height: calc(100vh - 64px - 4rem);
}

.upstream-accounts-page :deep(.layout-section-scrollable) {
  overflow: hidden;
}

.upstream-accounts-page :deep(.table-scroll-container) {
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  overflow: auto;
}

.accounts-shell {
  display: grid;
  gap: 16px;
}

.accounts-topbar {
  display: grid;
  grid-template-columns: minmax(460px, 1fr) minmax(360px, 0.8fr);
  gap: 16px;
  align-items: stretch;
}

.stats-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.stat-card,
.rate-guard-panel,
.accounts-table-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}

.stat-card {
  position: relative;
  display: flex;
  width: 100%;
  min-height: 82px;
  align-items: center;
  gap: 12px;
  border: 1px solid #e5e7eb;
  appearance: none;
  cursor: pointer;
  font: inherit;
  padding: 16px;
  text-align: left;
  transition: border-color 150ms ease, box-shadow 150ms ease, transform 150ms ease;
}

.stat-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.stat-card:focus-visible {
  outline: 2px solid #10b981;
  outline-offset: 2px;
}

.stat-alert-dot {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #dc2626;
}

.stat-icon {
  display: grid;
  width: 40px;
  height: 40px;
  flex: none;
  place-items: center;
  border-radius: 999px;
}

.stat-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.stat-copy strong {
  font-size: 24px;
  font-weight: 750;
  line-height: 1.1;
  color: #111827;
}

.stat-copy span {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
  font-weight: 500;
}

.stat-card-emerald .stat-icon {
  background: #ecfdf5;
  color: #059669;
}

.stat-card-gray .stat-icon {
  background: #f1f5f9;
  color: #64748b;
}

.stat-card-orange .stat-icon {
  background: #fff7ed;
  color: #d97706;
}

.stat-card-red .stat-icon {
  background: #fef2f2;
  color: #dc2626;
}

.accounts-actions {
  display: grid;
  grid-template-columns: minmax(150px, 0.8fr) minmax(0, 1.2fr);
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.accounts-button-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  min-width: 0;
}

.provider-summary {
  display: grid;
  min-width: 150px;
  gap: 2px;
  color: #64748b;
  font-size: 12px;
}

.provider-summary strong {
  overflow: hidden;
  color: #111827;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-summary code {
  color: #64748b;
  font-size: 11px;
}

.ui-button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 0 14px;
  color: #374151;
  font-weight: 600;
  transition: border-color 150ms ease, background 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.ui-button:hover:not(:disabled) {
  border-color: #a7f3d0;
  color: #059669;
  box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.08);
}

.ui-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.ui-button-primary {
  border-color: #059669;
  background: #059669;
  color: #fff;
}

.ui-button-primary:hover:not(:disabled) {
  border-color: #047857;
  background: #047857;
  color: #fff;
}

.ui-button-warning {
  border-color: #f59e0b;
  background: #fff7ed;
  color: #b45309;
}

.ui-button-warning:hover:not(:disabled) {
  border-color: #d97706;
  background: #fffbeb;
  color: #92400e;
  box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.12);
}

.ui-button-icon {
  width: 38px;
  padding: 0;
}

.accounts-action-test {
  border-color: #bfdbfe;
  background: #eff6ff;
  color: #1d4ed8;
}

.accounts-action-test:hover:not(:disabled) {
  border-color: #93c5fd;
  background: #dbeafe;
  color: #1d4ed8;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.08);
}

.accounts-action-secondary {
  color: #64748b;
}

.mobile-back-to-filters {
  display: none;
}

.rate-guard-panel {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(260px, auto) auto;
  gap: 20px;
  align-items: center;
  padding: 18px;
}

.rate-guard-panel.is-automation-target {
  @apply ring-2 ring-violet-500 ring-offset-2 dark:ring-offset-gray-950;
}

.guard-left {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 14px;
}

.guard-switch {
  position: relative;
  width: 42px;
  height: 24px;
  flex: none;
  margin-top: 2px;
  cursor: pointer;
}

.guard-switch input {
  position: absolute;
  opacity: 0;
}

.guard-switch span {
  display: block;
  width: 42px;
  height: 24px;
  border-radius: 999px;
  background: #cbd5e1;
  transition: background 150ms ease;
}

.guard-switch span::after {
  content: "";
  position: absolute;
  top: 3px;
  left: 3px;
  width: 18px;
  height: 18px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.2);
  transition: transform 150ms ease;
}

.guard-switch.is-on span {
  background: #059669;
}

.guard-switch.is-on span::after {
  transform: translateX(18px);
}

.guard-title {
  color: #111827;
  font-size: 15px;
  font-weight: 700;
}

.guard-description {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
}

.guard-status-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  color: #64748b;
  font-size: 12px;
}

.guard-ignore-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border-radius: 999px;
  background: #eef2ff;
  color: #4338ca;
  padding: 2px 8px;
  font-weight: 650;
}

.guard-sync-log-warning {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  background: #fffbeb;
  padding: 10px 12px;
}

.guard-warning-icon {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: 999px;
  background: #fef3c7;
  color: #d97706;
}

.guard-warning-copy {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.guard-warning-copy strong {
  color: #92400e;
  font-size: 13px;
  font-weight: 750;
}

.guard-warning-copy span {
  color: #b45309;
  font-size: 12px;
  line-height: 1.4;
}

.guard-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.guard-ignore-control {
  display: inline-flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.guard-ignore-summary {
  display: inline-flex;
  min-height: 28px;
  max-width: 260px;
  align-items: center;
  gap: 5px;
  border: 1px solid #c7d2fe;
  border-radius: 999px;
  background: #eef2ff;
  color: #4338ca;
  padding: 3px 10px;
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  text-align: left;
}

button.guard-ignore-summary {
  cursor: pointer;
  transition: border-color 150ms ease, background 150ms ease, box-shadow 150ms ease;
}

button.guard-ignore-summary:hover {
  border-color: #a5b4fc;
  background: #e0e7ff;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.08);
}

button.guard-ignore-summary:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.guard-ignore-summary span {
  overflow: hidden;
  min-width: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.guard-ignore-summary.is-empty {
  border-color: #e5e7eb;
  background: #f8fafc;
  color: #64748b;
}

.guard-ignore-summary.is-invalid {
  border-color: #fecaca;
  background: #fef2f2;
  color: #b91c1c;
}

.control-label {
  color: #475569;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.guard-hint {
  color: #64748b;
  font-size: 12px;
}

.ui-input {
  height: 38px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  outline: none;
  padding: 0 12px;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.ui-input:focus {
  border-color: #059669;
  box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.12);
}

.interval-input {
  width: 92px;
}

.ignored-accounts-input {
  width: min(320px, 100%);
}

.ignored-account-chips {
  display: flex;
  max-width: 460px;
  flex-wrap: wrap;
  gap: 6px;
}

.ignored-account-chip {
  display: inline-flex;
  min-width: 0;
  max-width: 220px;
  align-items: center;
  gap: 6px;
  border: 1px solid #dbeafe;
  border-radius: 8px;
  background: #eff6ff;
  padding: 5px 8px;
  color: #1e40af;
  font-size: 12px;
  font-weight: 650;
}

.ignored-account-chip-name {
  overflow: hidden;
  min-width: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ignored-account-chip small {
  flex: 0 0 auto;
  color: #64748b;
  font-size: 11px;
  font-weight: 600;
}

.ignored-account-remove {
  display: inline-flex;
  width: 18px;
  height: 18px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: rgba(30, 64, 175, 0.08);
  color: #1e40af;
  transition: background 150ms ease, color 150ms ease;
}

.ignored-account-remove:hover {
  background: rgba(185, 28, 28, 0.12);
  color: #b91c1c;
}

.rate-guard-ignored-dialog {
  overflow-y: auto;
}

.rate-guard-ignored-modal {
  width: min(680px, 100%);
}

.rate-guard-ignored-body {
  display: grid;
  gap: 12px;
  padding: 16px;
}

.rate-guard-ignored-input-row {
  display: grid;
  gap: 7px;
}

.rate-guard-ignored-body .ignored-accounts-input {
  width: 100%;
}

.rate-guard-ignored-error {
  margin: 0;
  border-radius: 6px;
  background: #fef2f2;
  padding: 8px 10px;
  color: #b91c1c;
  font-size: 12px;
  font-weight: 650;
}

.rate-guard-ignored-summary-row {
  display: flex;
}

.ignored-account-dialog-chips {
  max-width: none;
  gap: 8px;
}

.rate-guard-ignored-empty {
  display: grid;
  min-height: 92px;
  place-items: center;
  border: 1px dashed #cbd5e1;
  border-radius: 8px;
  background: #f8fafc;
  color: #64748b;
  font-size: 13px;
  font-weight: 650;
}

.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.filter-sticky-row {
  display: grid;
  grid-template-columns: minmax(300px, 1fr) auto;
  flex: 1 1 360px;
  gap: 12px;
  align-items: center;
  min-width: 360px;
}

.filter-controls {
  display: grid;
  grid-template-columns: repeat(4, minmax(142px, 1fr));
  flex: 2 1 600px;
  gap: 12px;
  min-width: 0;
}

.filter-select {
  width: 100%;
}

.filter-select :deep(select),
.filter-select :deep(button) {
  min-height: 38px;
  border-radius: 8px;
  border-color: #d1d5db;
  background: #fff;
}

.search-wrap {
  position: relative;
  min-width: 0;
}

.search-wrap > svg {
  position: absolute;
  top: 50%;
  left: 12px;
  color: #94a3b8;
  transform: translateY(-50%);
}

.filter-search {
  width: 100%;
  padding-left: 38px;
}

.filter-toggle-button {
  display: none;
}

.filtered-count {
  display: inline-flex;
  height: 34px;
  align-items: center;
  gap: 6px;
  border-radius: 8px;
  background: #f1f5f9;
  padding: 0 12px;
  color: #64748b;
  white-space: nowrap;
}

.filtered-count strong {
  color: #111827;
  font-weight: 750;
}

.quick-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  border-top: 1px solid #eef2f7;
  padding-top: 4px;
}

.quick-tag {
  display: inline-flex;
  min-height: 30px;
  align-items: center;
  gap: 6px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 0 12px;
  color: #334155;
  font-size: 12px;
  font-weight: 650;
  transition: border-color 150ms ease, background 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.quick-tag strong {
  color: #0f172a;
  font-size: 12px;
  font-weight: 800;
}

.quick-tag:hover {
  border-color: #059669;
  color: #059669;
  box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.06);
}

.quick-tag.active {
  border-color: #059669;
  background: #059669;
  color: #fff;
}

.quick-tag.active strong {
  color: inherit;
}

.quick-tag-danger.active {
  border-color: #dc2626;
  background: #dc2626;
  color: #fff;
}

.quick-tag-muted.active {
  border-color: #64748b;
  background: #64748b;
  color: #fff;
}

.accounts-table-content {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.warning-banner {
  display: flex;
  gap: 8px;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  background: #fff7ed;
  padding: 12px;
  color: #c2410c;
  font-size: 13px;
}

.accounts-table-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  height: auto;
  max-height: none;
}

.accounts-table-card :deep(.table-wrapper) {
  display: flex;
  flex: 1;
  min-height: 0;
}

.accounts-table-card :deep(table) {
  border-collapse: collapse;
  table-layout: fixed;
  width: max(100%, 1700px);
  min-width: 1700px;
}

.accounts-table-card :deep(.upstream-source-column) {
  width: 170px;
}

.accounts-table-card :deep(.upstream-key-column) {
  width: 150px;
}

.accounts-table-card :deep(.upstream-local-account-column) {
  width: 190px;
}

.accounts-table-card :deep(.upstream-priority-column) {
  width: 88px;
}

.accounts-table-card :deep(.upstream-rate-column) {
  width: 115px;
}

.accounts-table-card :deep(.upstream-bound-groups-column) {
  width: 260px;
  white-space: normal;
}

.accounts-table-card :deep(.upstream-money-column) {
  width: 120px;
}

.accounts-table-card :deep(.upstream-status-column) {
  width: 120px;
}

.accounts-table-card :deep(.upstream-schedulable-column) {
  width: 105px;
}

.accounts-table-card :deep(.upstream-test-status-column) {
  width: 130px;
}

.accounts-table-card :deep(.upstream-test-time-column) {
  width: 155px;
}

.accounts-table-card :deep(.upstream-actions-column) {
  width: 185px;
  white-space: normal;
}

.accounts-table-card :deep(thead),
.accounts-table-card :deep(.table-header),
.accounts-table-card :deep(.sticky-header-cell) {
  background: #f8fafc;
}

.accounts-table-card :deep(th) {
  border-bottom: 1px solid #e5e7eb;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
}

.accounts-table-card :deep(th.upstream-center-column),
.accounts-table-card :deep(td.upstream-center-column) {
  text-align: center;
}

.accounts-table-card :deep(th.upstream-center-column > div) {
  justify-content: center;
}

.accounts-table-card :deep(td) {
  border-bottom: 1px solid #eef2f7;
  color: #334155;
  padding-top: 11px;
  padding-bottom: 11px;
}

.accounts-table-card :deep(.data-table-row) {
  transition: background 150ms ease;
}

.accounts-table-card :deep(.data-table-row:hover) {
  background: #f8fafc;
}

.accounts-table-card :deep(.data-table-row.risk-row),
.accounts-table-card :deep(.data-table-row.risk-row .sticky-col) {
  background: #fffafa;
}

.accounts-table-card :deep(.data-table-row.risk-row:hover),
.accounts-table-card :deep(.data-table-row.risk-row:hover .sticky-col) {
  background: #fff4f4;
}

.accounts-table-card :deep(.data-table-row.test-failed-row),
.accounts-table-card :deep(.data-table-row.test-failed-row .sticky-col) {
  background: #fffaf4;
}

.accounts-table-card :deep(.data-table-row.test-failed-row:hover),
.accounts-table-card :deep(.data-table-row.test-failed-row:hover .sticky-col) {
  background: #fff3e4;
}

.accounts-table-card :deep(.data-table-row.risk-row.test-failed-row),
.accounts-table-card :deep(.data-table-row.risk-row.test-failed-row .sticky-col) {
  background: #fff8f4;
}

.accounts-table-card :deep(.data-table-row.provider-disabled-row),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .sticky-col) {
  background: #fbfcfe;
}

.accounts-table-card :deep(.data-table-row.provider-disabled-row:hover),
.accounts-table-card :deep(.data-table-row.provider-disabled-row:hover .sticky-col) {
  background: #f8fafc;
}

.accounts-table-card :deep(.data-table-row.provider-disabled-row td) {
  color: #cbd5e1;
}

.accounts-table-card :deep(.data-table-row.provider-disabled-row td > *),
.accounts-table-card :deep(.provider-disabled-row > .space-y-3) {
  filter: grayscale(1);
  opacity: 0.46;
}

.accounts-table-card :deep(.data-table-row.provider-disabled-row td),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .main-text),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .sub-text),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .source-id),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .table-tag),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .dash),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .consumption-value),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .test-status-pill),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .test-time),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .action-cell),
.accounts-table-card :deep(.data-table-row.provider-disabled-row .source-warning-icon) {
  color: #94a3b8;
}

.accounts-table-card :deep(.data-table-row.provider-disabled-row .source-line) {
  background: #cbd5e1;
}

.accounts-table-card :deep(.data-table-row.provider-disabled-row .home-tag) {
  border-color: #d1d5db;
}

.dark .accounts-table-card :deep(.data-table-row.provider-disabled-row),
.dark .accounts-table-card :deep(.data-table-row.provider-disabled-row .sticky-col) {
  background: rgb(17 24 39 / 0.72);
}

.dark .accounts-table-card :deep(.data-table-row.provider-disabled-row:hover),
.dark .accounts-table-card :deep(.data-table-row.provider-disabled-row:hover .sticky-col) {
  background: rgb(31 41 55 / 0.72);
}

.source-cell {
  display: grid;
  min-width: 13rem;
  grid-template-columns: 2px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.source-line {
  width: 2px;
  height: 42px;
  border-radius: 999px;
}

.source-line-emerald {
  background: #059669;
}

.source-line-blue {
  background: #2563eb;
}

.source-line-red {
  background: #dc2626;
}

.source-line-amber {
  background: #d97706;
}

.source-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.source-warning-icon {
  color: #dc2626;
}

.source-id,
.sub-text {
  display: block;
  overflow: hidden;
  margin-top: 5px;
  color: #64748b;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-main,
.two-line-cell {
  min-width: 0;
}

.main-text {
  display: block;
  overflow: hidden;
  color: #1f2937;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-tag,
.group-chip,
.log-chip,
.trigger-chip,
.status-pill {
  display: inline-flex;
  max-width: 100%;
  align-items: center;
  gap: 6px;
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  white-space: nowrap;
}

.table-tag {
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-pill::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
}

.status-pill-on {
  background: #ecfdf5;
  color: #059669;
}

.status-pill-muted,
.tag-gray,
.home-tag,
.tag-account {
  background: #f1f5f9;
  color: #64748b;
}

.tag-rate-guard-ignore {
  background: #eef2ff;
  color: #4338ca;
}

.record-status {
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
}

.record-status-success {
  background: #ecfdf5;
  color: #047857;
}

.record-status-error,
.status-error-message,
.sub-text-warning {
  color: #dc2626;
}

.tag-provider-sky,
.tag-provider-cyan,
.tag-provider-indigo {
  background: #eff6ff;
  color: #1d4ed8;
}

.tag-provider-emerald,
.tag-provider-teal {
  background: #ecfdf5;
  color: #047857;
}

.tag-provider-violet {
  background: #f5f3ff;
  color: #6d28d9;
}

.tag-provider-rose {
  background: #fef2f2;
  color: #b91c1c;
}

.tag-provider-amber,
.tag-warning,
.tag-local-snapshot {
  background: #fff7ed;
  color: #c2410c;
}

.account-id-tag {
  margin-top: 6px;
}

.priority-pill {
  display: inline-flex;
  min-width: 34px;
  justify-content: center;
  align-items: center;
  gap: 4px;
  border-radius: 6px;
  border: 0;
  background: #f1f5f9;
  padding: 2px 8px;
  color: #334155;
  font-family: inherit;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  line-height: 18px;
}

.priority-pill-button {
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.priority-pill-button:hover:not(:disabled) {
  background: #e2e8f0;
  color: #0f172a;
}

.priority-pill-button:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.dark .priority-pill {
  background: rgb(51 65 85 / 0.72);
  color: #e2e8f0;
}

.dark .priority-pill-button:hover:not(:disabled) {
  background: rgb(71 85 105 / 0.86);
  color: #f8fafc;
}

.priority-edit-form {
  display: inline-flex;
}

.priority-input {
  width: 64px;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  background: #fff;
  padding: 3px 6px;
  color: #0f172a;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  line-height: 18px;
  outline: none;
}

.priority-input:focus {
  border-color: #2563eb;
  box-shadow: 0 0 0 2px rgb(37 99 235 / 0.14);
}

.dark .priority-input {
  border-color: #475569;
  background: #0f172a;
  color: #e2e8f0;
}

.balance-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.balance-value {
  color: #059669;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
}

.consumption-value {
  color: #ea580c;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
}

.trend-btn {
  display: grid;
  width: 24px;
  height: 24px;
  place-items: center;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  background: #fff;
  color: #64748b;
  cursor: pointer;
  transition: all 150ms ease;
}

.trend-btn:hover {
  border-color: #2563eb;
  background: #eff6ff;
  color: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.2);
}

.trend-btn:active {
  transform: translateY(0);
}

.platform-text-openai {
  color: #047857;
}

.platform-text-anthropic {
  color: #c2410c;
}

.platform-text-gemini {
  color: #1d4ed8;
}

.platform-text-antigravity {
  color: #6d28d9;
}

.platform-text-default {
  color: #1f2937;
}

.platform-tag-openai {
  background: #ecfdf5;
  color: #047857;
}

.platform-tag-anthropic {
  background: #fff7ed;
  color: #c2410c;
}

.platform-tag-gemini {
  background: #eff6ff;
  color: #1d4ed8;
}

.platform-tag-antigravity {
  background: #f5f3ff;
  color: #6d28d9;
}

.platform-tag-default {
  background: #f1f5f9;
  color: #64748b;
}

.rate-cell {
  display: grid;
  justify-items: end;
  gap: 7px;
}

.rate-value {
  min-width: 62px;
  justify-content: center;
  font-variant-numeric: tabular-nums;
}

.rate-bar {
  display: block;
  width: 76px;
  height: 4px;
  overflow: hidden;
  border-radius: 999px;
  background: #e5e7eb;
}

.rate-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: currentColor;
}

.rate-deep-green {
  background: #d1fae5;
  color: #065f46;
}

.rate-green {
  background: #ecfdf5;
  color: #047857;
}

.rate-orange {
  background: #ffedd5;
  color: #c2410c;
}

.rate-deep-orange {
  background: #fed7aa;
  color: #9a3412;
}

.rate-red {
  background: #fef2f2;
  color: #b91c1c;
}

.rate-muted {
  background: #f1f5f9;
  color: #64748b;
}

.tag-list {
  display: flex;
  max-width: none;
  flex-wrap: wrap;
  gap: 6px;
}

.group-list {
  width: 100%;
  min-width: 0;
  white-space: normal;
}

.group-chip-blue {
  background: #eff6ff;
  color: #1d4ed8;
}

.group-chip-emerald {
  background: #ecfdf5;
  color: #047857;
}

.group-chip-violet {
  background: #f5f3ff;
  color: #6d28d9;
}

.group-chip-warning,
.log-chip-warning {
  background: #fef2f2;
  color: #b91c1c;
}

.dash {
  color: #94a3b8;
}

.action-cell {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.test-status-cell {
  display: flex;
  align-items: center;
  min-height: 32px;
}

.test-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
}

.test-status-testing {
  background: #eff6ff;
  color: #2563eb;
}

.test-status-success {
  background: #ecfdf5;
  color: #059669;
}

.test-status-failed {
  background: #fef2f2;
  color: #dc2626;
}

.test-status-skipped {
  background: #f5f3ff;
  color: #7c3aed;
}

.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 32px;
}

.schedulable-toggle {
  display: inline-flex;
  align-items: center;
  border: 0;
  background: transparent;
  padding: 0;
}

.schedulable-toggle:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.schedulable-track {
  display: inline-flex;
  width: 38px;
  height: 22px;
  align-items: center;
  border-radius: 999px;
  padding: 2px;
  transition: background 150ms ease;
}

.schedulable-on .schedulable-track {
  background: #059669;
}

.schedulable-off .schedulable-track {
  background: #cbd5e1;
}

.schedulable-thumb {
  display: block;
  width: 16px;
  height: 16px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.24);
  transition: transform 150ms ease;
}

.schedulable-thumb-on {
  transform: translateX(16px);
}

.schedulable-thumb-off {
  transform: translateX(0);
}

.accounts-table-card :deep(.upstream-rate-column .rate-cell) {
  justify-items: center;
}

.accounts-table-card :deep(.upstream-rate-column .rate-value) {
  min-width: 72px;
}

.accounts-table-card :deep(.upstream-rate-column .rate-bar) {
  margin-inline: auto;
}

.action-dash {
  display: flex;
  justify-content: flex-end;
}

.text-action {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 0;
  background: transparent;
  padding: 4px 0;
  color: #059669;
  font-weight: 650;
  transition: color 150ms ease;
}

.text-action:hover:not(:disabled) {
  color: #047857;
}

.text-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.text-action-danger {
  color: #dc2626;
}

.text-action-muted {
  color: #64748b;
}

.text-action-primary {
  color: #047857;
}

.text-action-primary:hover {
  color: #065f46;
}

.stat-details-dialog,
.sync-logs-dialog {
  overflow-y: auto;
}

.sync-confirm-dialog {
  overflow-y: auto;
}

.sync-result-dialog {
  overflow-y: auto;
}

.batch-test-result-dialog {
  overflow: auto;
}

.stat-details-modal {
  display: flex;
  width: min(1040px, 100%);
  max-height: 86vh;
  flex-direction: column;
  overflow: hidden;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.28);
}

.sync-confirm-modal {
  display: flex;
  width: min(980px, 100%);
  max-height: 86vh;
  flex-direction: column;
  overflow: hidden;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.28);
}

.sync-result-modal {
  display: flex;
  width: min(920px, 100%);
  max-height: 86vh;
  flex-direction: column;
  overflow: hidden;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.28);
}

.batch-test-config-modal {
  width: min(720px, 100%);
}

.batch-test-result-modal {
  width: min(1280px, calc(100vw - 32px));
  max-height: calc(100vh - 48px);
  max-height: calc(100dvh - 48px);
}

.batch-test-result-modal .sync-confirm-body {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.batch-test-result-modal .sync-confirm-section {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
}

.stat-details-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid #e5e7eb;
  padding: 18px 20px;
}

.stat-details-header h3 {
  margin: 0;
  color: #111827;
  font-size: 16px;
  font-weight: 750;
}

.stat-details-header p {
  margin: 4px 0 0;
  color: #64748b;
  font-size: 12px;
}

.stat-details-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #eef2f7;
  padding: 12px 18px;
  background: #f8fafc;
}

.stat-details-count {
  display: inline-flex;
  border-radius: 999px;
  background: #ecfdf5;
  padding: 3px 10px;
  color: #047857;
  font-size: 12px;
  font-weight: 800;
}

.stat-details-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 16px;
}

.stat-details-table-wrap {
  min-width: 0;
  overflow: auto;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.stat-details-table {
  width: 100%;
  min-width: 900px;
  border-collapse: collapse;
  background: #fff;
}

.stat-details-table th,
.stat-details-table td {
  border-bottom: 1px solid #eef2f7;
  padding: 11px 12px;
  text-align: left;
  vertical-align: top;
}

.stat-details-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: #f8fafc;
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
}

.stat-details-table tbody tr:last-child td {
  border-bottom: 0;
}

.stat-details-source-cell,
.stat-details-rate-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}

.stat-details-source-cell code {
  max-width: 180px;
  overflow: hidden;
  color: #64748b;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stat-details-action {
  display: inline-flex;
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
  white-space: nowrap;
}

.stat-details-action-create {
  background: #ecfdf5;
  color: #047857;
}

.stat-details-action-update {
  background: #fff7ed;
  color: #c2410c;
}

.stat-details-action-conflict,
.stat-details-action-skip {
  background: #fef2f2;
  color: #b91c1c;
}

.stat-details-action-muted {
  background: #f1f5f9;
  color: #64748b;
}

.stat-details-empty {
  display: grid;
  min-height: 180px;
  place-items: center;
  gap: 10px;
  color: #94a3b8;
  font-size: 13px;
  font-weight: 600;
}

.stat-details-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  border-top: 1px solid #e5e7eb;
  padding: 14px 18px;
}

.sync-confirm-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid #e5e7eb;
  padding: 18px 20px;
}

.sync-confirm-header h3 {
  margin: 0;
  color: #111827;
  font-size: 16px;
  font-weight: 750;
}

.sync-confirm-header p {
  margin: 4px 0 0;
  color: #64748b;
  font-size: 12px;
}

.sync-confirm-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  border-bottom: 1px solid #eef2f7;
  padding: 14px 18px;
  background: #f8fafc;
}

.sync-confirm-option {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 10px 12px;
}

.sync-confirm-option input {
  width: 16px;
  height: 16px;
  flex: none;
  accent-color: #059669;
}

.sync-confirm-option span {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.sync-confirm-option strong {
  min-width: 0;
  overflow: hidden;
  color: #334155;
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sync-confirm-option small {
  display: inline-flex;
  min-width: 28px;
  justify-content: center;
  border-radius: 999px;
  background: #ecfdf5;
  padding: 2px 8px;
  color: #047857;
  font-size: 12px;
  font-weight: 800;
}

.sync-confirm-option:has(input:disabled) {
  opacity: 0.58;
}

.sync-result-stat {
  display: grid;
  gap: 3px;
  min-width: 0;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 10px 12px;
}

.sync-result-stat span {
  color: #64748b;
  font-size: 12px;
}

.sync-result-stat strong {
  color: #111827;
  font-size: 20px;
  font-weight: 800;
  line-height: 1.1;
}

.sync-confirm-body {
  display: grid;
  flex: 1 1 auto;
  gap: 14px;
  min-height: 0;
  overflow: auto;
  padding: 16px 18px;
}

.sync-confirm-section {
  overflow: hidden;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.sync-confirm-section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid #eef2f7;
  background: #f8fafc;
  padding: 10px 12px;
  color: #334155;
  font-size: 13px;
  font-weight: 750;
}

.sync-confirm-section-title strong {
  border-radius: 999px;
  background: #e2e8f0;
  padding: 2px 8px;
  color: #475569;
  font-size: 12px;
}

.sync-confirm-list {
  display: grid;
  gap: 8px;
  padding: 10px;
}

.sync-confirm-item {
  display: grid;
  gap: 8px;
  border: 1px solid #eef2f7;
  border-radius: 8px;
  padding: 10px;
}

.sync-confirm-item-risk {
  border-color: #fed7aa;
  background: #fff7ed;
}

.sync-confirm-item-main,
.sync-confirm-item-meta {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.sync-confirm-item-main strong {
  min-width: 0;
  max-width: 320px;
  overflow: hidden;
  color: #111827;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sync-confirm-item-main code {
  border-radius: 6px;
  background: #f1f5f9;
  padding: 2px 6px;
  color: #475569;
  font-size: 12px;
}

.sync-confirm-item-checkbox {
  width: 16px;
  height: 16px;
  flex: none;
  accent-color: #059669;
}

.sync-confirm-item-meta {
  color: #64748b;
  font-size: 12px;
}

.sync-confirm-detail-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.sync-confirm-detail-chip {
  border-radius: 999px;
  background: #eef2ff;
  padding: 3px 8px;
  color: #3730a3;
  font-size: 12px;
  font-weight: 650;
}

.sync-confirm-detail-warning {
  background: #fff7ed;
  color: #c2410c;
}

.sync-confirm-danger {
  color: #dc2626;
  font-weight: 750;
}

.sync-confirm-empty {
  padding: 14px 12px;
  color: #94a3b8;
  font-size: 13px;
}

.sync-confirm-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  border-top: 1px solid #e5e7eb;
  padding: 14px 18px;
}

.sync-logs-modal {
  display: flex;
  width: min(1180px, 100%);
  height: 80vh;
  max-height: 80vh;
  flex-direction: column;
  overflow: hidden;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.28);
}

.sync-logs-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid #e5e7eb;
  padding: 18px 20px;
}

.sync-logs-modal-header h3 {
  margin: 0;
  color: #111827;
  font-size: 16px;
  font-weight: 750;
}

.sync-logs-modal-header p {
  margin: 4px 0 0;
  color: #64748b;
  font-size: 12px;
}

.modal-close-button {
  display: inline-flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  color: #64748b;
  transition: border-color 150ms ease, color 150ms ease, background 150ms ease;
}

.modal-close-button:hover {
  border-color: #cbd5e1;
  background: #f8fafc;
  color: #111827;
}

.sync-logs-modal-info {
  flex: none;
}

.sync-logs-table-wrap {
  flex: 1 1 auto;
  min-height: 0;
  max-height: none;
}

.sync-log-card-list {
  display: none;
}

.sync-log-card {
  position: relative;
  display: grid;
  gap: 10px;
  overflow: visible;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  padding: 12px;
}

.sync-log-card::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: #ea580c;
  content: "";
}

.sync-log-card.is-handled {
  background: #f8fafc;
  opacity: 0.78;
}

.sync-log-card.is-handled::before {
  background: #059669;
}

.sync-log-card-head {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.sync-log-card-status {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  gap: 6px;
}

.sync-log-card-head time {
  flex: 0 0 auto;
  color: #64748b;
  font-size: 11px;
  line-height: 1.8;
  white-space: nowrap;
}

.sync-log-card-main {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.sync-log-card-field {
  display: grid;
  min-width: 0;
  align-content: flex-start;
  gap: 5px;
  border-radius: 8px;
  background: #f8fafc;
  padding: 8px;
}

.sync-log-card-field-wide {
  grid-column: 1 / -1;
}

.sync-log-card-field > span:first-child {
  color: #64748b;
  font-size: 11px;
  font-weight: 800;
  line-height: 1.1;
}

.sync-log-card-field strong {
  min-width: 0;
  overflow-wrap: anywhere;
  color: #0f172a;
  font-size: 13px;
  font-weight: 800;
  line-height: 1.25;
}

.sync-log-card-field .tag-list {
  min-width: 0;
}

.sync-log-card-field .table-tag,
.sync-log-card-field .log-chip,
.sync-log-card-field .trigger-chip {
  min-height: 24px;
  min-width: 0;
  line-height: 18px;
}

.sync-log-card-action {
  justify-self: flex-end;
  min-height: 32px;
  border: 1px solid #fed7aa;
  border-radius: 7px;
  background: #fff7ed;
  padding: 0 10px;
  color: #c2410c;
  font-size: 12px;
  line-height: 1.2;
}

.records-info {
  margin: 14px 18px 0;
  border-radius: 8px;
  background: #f8fafc;
  padding: 10px 12px;
  color: #64748b;
  font-size: 12px;
}

.records-table-wrap {
  max-height: 20rem;
  overflow: auto;
}

.batch-test-table-wrap {
  max-height: 32rem;
}

.batch-test-result-modal .batch-test-table-wrap {
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  max-height: none;
  overflow: auto;
  -webkit-overflow-scrolling: touch;
}

.batch-test-table {
  min-width: 1280px;
}

.batch-test-error {
  max-width: 320px;
  overflow-wrap: anywhere;
  color: #64748b;
}

.batch-result-scroll {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: auto;
  -webkit-overflow-scrolling: touch;
}

.batch-result-toolbar {
  position: sticky;
  top: 0;
  z-index: 3;
  display: grid;
  gap: 8px;
  border-bottom: 1px solid #eef2f7;
  background: #fff;
  padding: 10px;
}

.batch-result-tabs {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  padding-bottom: 1px;
}

.batch-result-tab {
  display: inline-flex;
  min-height: 30px;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  border: 1px solid #dbe3ee;
  border-radius: 999px;
  background: #fff;
  padding: 0 10px;
  color: #64748b;
  font-size: 12px;
  font-weight: 800;
  white-space: nowrap;
}

.batch-result-tab strong {
  color: #0f172a;
  font-size: 12px;
}

.batch-result-tab.active {
  border-color: #bae6fd;
  background: #eff6ff;
  color: #1d4ed8;
}

.batch-result-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: #64748b;
  font-size: 12px;
}

.batch-result-hint .table-tag {
  flex: none;
}

.batch-result-list {
  display: grid;
  gap: 10px;
  padding: 10px;
}

.batch-result-card {
  position: relative;
  display: grid;
  gap: 10px;
  overflow: hidden;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  padding: 10px;
}

.batch-result-card::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: #059669;
  content: "";
}

.batch-result-card.failed::before {
  background: #dc2626;
}

.batch-result-card.skipped::before {
  background: #7c3aed;
}

.batch-result-card.failed-schedulable {
  border-color: #fdba74;
  background: #fff7ed;
}

.batch-result-card.failed-schedulable::before {
  background: #ea580c;
}

.batch-result-card-head {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.batch-result-card-status,
.batch-test-status-cell {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.batch-result-card-status {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.batch-risk-tag {
  border: 1px solid #fed7aa;
  background: #fff7ed;
  color: #c2410c;
}

.batch-result-account {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.batch-result-account strong {
  overflow-wrap: anywhere;
  color: #0f172a;
  font-size: 14px;
  font-weight: 800;
  line-height: 1.2;
}

.batch-result-account span {
  overflow-wrap: anywhere;
  color: #64748b;
  font-size: 12px;
}

.batch-result-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.batch-result-metric {
  display: grid;
  gap: 3px;
  border-radius: 8px;
  background: #f8fafc;
  padding: 8px;
}

.batch-result-metric span {
  color: #64748b;
  font-size: 11px;
  font-weight: 800;
}

.batch-result-metric strong {
  color: #0f172a;
  font-size: 13px;
  font-weight: 800;
}

.batch-result-metric .rate-value {
  display: inline-flex;
  width: fit-content;
  min-width: 58px;
  border-radius: 6px;
  padding: 2px 8px;
  line-height: 18px;
}

.batch-result-error {
  overflow-wrap: anywhere;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  background: #fff7ed;
  padding: 8px;
  color: #9a3412;
  font-size: 12px;
  line-height: 1.45;
}

.batch-result-card-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
  border-top: 1px solid #eef2f7;
  padding-top: 8px;
}

.batch-result-empty {
  padding: 18px 12px;
  color: #94a3b8;
  font-size: 13px;
  text-align: center;
}

.batch-table-details {
  border-top: 1px solid #eef2f7;
  background: #fff;
}

.batch-table-details summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  color: #334155;
  cursor: pointer;
  font-size: 13px;
  font-weight: 800;
  list-style: none;
}

.batch-table-details summary::-webkit-details-marker {
  display: none;
}

.batch-table-details summary span {
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
}

.batch-table-details .records-table-wrap {
  max-height: 320px;
  border-top: 1px solid #eef2f7;
}

.batch-test-config-list {
  display: grid;
  gap: 12px;
}

.batch-test-config-row {
  display: grid;
  grid-template-columns: minmax(180px, 0.45fr) minmax(260px, 1fr);
  gap: 12px;
  align-items: center;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 12px;
}

.batch-test-config-platform,
.batch-test-model-control,
.batch-test-schedulable-cell,
.batch-test-actions-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.batch-test-config-platform {
  flex-wrap: wrap;
  color: #64748b;
  font-size: 12px;
}

.batch-test-model-control {
  flex-wrap: wrap;
}

.batch-test-model-select {
  min-width: 220px;
  flex: 1 1 220px;
}

@media (max-width: 640px) {
  .batch-test-config-row {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .batch-test-model-control {
    width: 100%;
  }

  .batch-test-model-select {
    min-width: 0;
    width: 100%;
    flex-basis: 100%;
  }
}

.batch-test-model-hint {
  color: #64748b;
  font-size: 12px;
}

.batch-test-schedulable-cell {
  flex-wrap: wrap;
}

.batch-test-actions-cell {
  flex-wrap: nowrap;
}

.batch-test-sort-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  background: transparent;
  padding: 0;
  color: inherit;
  font: inherit;
  font-weight: 650;
  white-space: nowrap;
}

.batch-test-sort-button span {
  display: inline-block;
  min-width: 10px;
  color: #0f766e;
}

.ui-button-xs {
  min-height: 28px;
  padding: 4px 8px;
  font-size: 12px;
}

.ui-button-danger {
  border-color: #fecaca;
  color: #b91c1c;
}

.ui-button-danger:hover:not(:disabled) {
  border-color: #fca5a5;
  background: #fef2f2;
}

.records-table-wrap.sync-logs-table-wrap {
  height: 100%;
  max-height: none;
}

.records-table {
  min-width: 1240px;
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.records-table th {
  border-bottom: 1px solid #e5e7eb;
  background: #f8fafc;
  padding: 10px 16px;
  color: #64748b;
  font-weight: 600;
  text-align: left;
}

.records-table td {
  border-bottom: 1px solid #eef2f7;
  padding: 12px 16px;
  color: #334155;
  line-height: 1.45;
  vertical-align: top;
}

.records-table tbody tr {
  transition: background 150ms ease;
}

.records-table tbody tr:hover {
  background: #f8fafc;
}

.batch-test-table tr.batch-test-risk-row td {
  background: #fff7ed;
}

.batch-test-table tr.batch-test-risk-row:hover td {
  background: #ffedd5;
}

.records-row-handled {
  opacity: 0.72;
}

.sync-log-status {
  display: inline-flex;
  min-height: 24px;
  align-items: center;
  border: 0;
  border-radius: 999px;
  padding: 3px 8px;
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
  white-space: nowrap;
}

button.sync-log-status {
  cursor: pointer;
}

.sync-log-status-unhandled {
  background: #fff7ed;
  color: #c2410c;
}

button.sync-log-status-unhandled:hover {
  background: #ffedd5;
}

.sync-log-status-handled {
  background: #ecfdf5;
  color: #047857;
}

.trigger-sync {
  background: #ecfdf5;
  color: #047857;
}

.trigger-scheduled {
  background: #f5f3ff;
  color: #6d28d9;
}

.trigger-guard {
  background: #fff7ed;
  color: #c2410c;
}

.log-chip {
  background: #f1f5f9;
  color: #475569;
}

.rate-compare {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border-radius: 6px;
  background: #f1f5f9;
  padding: 4px 8px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  font-weight: 650;
}

.rate-compare-upstream {
  color: #c2410c;
}

.rate-compare-local {
  color: #047857;
}

.records-empty {
  display: grid;
  place-items: center;
  gap: 10px;
  padding: 42px 16px 46px;
  color: #64748b;
  text-align: center;
}

.records-empty svg {
  color: #cbd5e1;
}

@media (max-width: 1280px) {
  .accounts-topbar {
    grid-template-columns: 1fr;
  }

  .accounts-actions {
    grid-template-columns: minmax(180px, auto) minmax(0, 1fr);
  }
}

@media (max-width: 1023px) {
  .upstream-accounts-page :deep(.table-page-layout.mobile-mode) {
    height: auto;
    min-height: calc(100vh - 64px - 2rem);
  }

  .upstream-accounts-page :deep(.table-page-layout.mobile-mode .layout-section-scrollable) {
    overflow: visible;
  }

  .accounts-topbar,
  .rate-guard-panel,
  .filter-row {
    grid-template-columns: 1fr;
  }

  .accounts-shell {
    gap: 12px;
    padding: 12px;
  }

  .accounts-actions,
  .guard-controls {
    justify-content: flex-start;
  }

  .accounts-actions {
    grid-template-columns: 1fr;
    justify-items: flex-start;
  }

  .accounts-button-group {
    justify-content: flex-start;
    width: 100%;
  }

  .provider-summary {
    width: 100%;
  }

  .provider-summary strong,
  .provider-summary code,
  .main-text,
  .source-id,
  .sub-text,
  .table-tag,
  .group-chip,
  .log-chip,
  .trigger-chip {
    overflow: visible;
    text-overflow: clip;
    white-space: normal;
    overflow-wrap: anywhere;
  }

  .source-cell {
    min-width: 0;
    justify-items: end;
    text-align: right;
  }

  .source-title,
  .tag-list,
  .action-cell {
    justify-content: flex-end;
  }

  .rate-cell {
    justify-items: end;
  }

  .guard-sync-log-warning {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .guard-sync-log-warning .ui-button {
    grid-column: 1 / -1;
    justify-self: flex-start;
  }

  .accounts-table-content {
    height: auto;
    overflow: visible;
  }

  .accounts-table-card {
    height: auto;
    min-height: 0;
    overflow: visible;
  }

  .sync-logs-modal {
    width: 100%;
    height: min(86vh, 760px);
    max-height: 86vh;
  }

  .sync-confirm-modal {
    width: 100%;
    max-height: 88vh;
  }

  .sync-result-modal {
    width: 100%;
    max-height: 88vh;
  }

  .stat-details-modal {
    width: 100%;
    max-height: 88vh;
  }

  .stat-details-table {
    min-width: 820px;
  }

  .batch-test-result-dialog {
    align-items: stretch;
    padding: 12px;
  }

  .sync-result-modal.batch-test-result-modal {
    height: calc(100vh - 24px);
    height: calc(100dvh - 24px);
    max-height: calc(100vh - 24px);
    max-height: calc(100dvh - 24px);
  }

  .batch-test-result-modal .sync-confirm-header {
    align-items: flex-start;
    padding: 14px 16px;
  }

  .sync-confirm-summary {
    grid-template-columns: 1fr;
  }

  .batch-test-result-modal .sync-confirm-summary {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
    padding: 10px 12px;
  }

  .batch-test-result-modal .sync-result-stat {
    min-height: 54px;
    gap: 2px;
    padding: 8px;
  }

  .batch-test-result-modal .sync-result-stat span {
    overflow: hidden;
    font-size: 11px;
    line-height: 1.15;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .batch-test-result-modal .sync-result-stat strong {
    font-size: 18px;
  }

  .batch-test-result-modal .sync-confirm-body {
    padding: 12px;
  }

  .sync-confirm-item-main strong {
    max-width: 100%;
    overflow: visible;
    text-overflow: clip;
    white-space: normal;
    overflow-wrap: anywhere;
  }

  .sync-confirm-footer {
    flex-direction: column-reverse;
  }

  .sync-confirm-footer .ui-button {
    width: 100%;
    justify-content: center;
  }

  .batch-test-result-modal .sync-confirm-footer {
    flex-direction: row;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
    padding: 12px;
  }

  .batch-test-result-modal .sync-confirm-footer .ui-button {
    width: auto;
    min-height: 32px;
    padding: 0 10px;
    font-size: 12px;
  }

  .batch-table-details .records-table-wrap {
    max-height: 260px;
  }

  .sync-logs-modal-header {
    align-items: flex-start;
    padding: 14px 16px;
  }

  .records-table-wrap {
    max-width: 100%;
    overflow: auto;
  }

}

@media (max-width: 768px) {
  .stats-strip {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
  }

  .stat-card {
    min-height: 52px;
    gap: 5px;
    padding: 8px 6px;
  }

  .stat-icon {
    width: 24px;
    height: 24px;
    border-radius: 7px;
  }

  .stat-copy strong {
    font-size: 16px;
    font-weight: 800;
    line-height: 1;
  }

  .stat-copy span {
    margin-top: 2px;
    font-size: 10px;
    line-height: 1.15;
  }

  .stat-alert-dot {
    top: 7px;
    right: 7px;
    width: 6px;
    height: 6px;
  }

  .accounts-actions {
    width: 100%;
    align-items: flex-start;
    gap: 8px;
  }

  .accounts-button-group {
    gap: 8px;
  }

  .accounts-actions .ui-button {
    flex: 0 0 auto;
    width: auto;
    min-height: 32px;
    padding: 0 10px;
    font-size: 12px;
    line-height: 1;
    white-space: nowrap;
  }

  .accounts-actions .ui-button-icon {
    flex: 0 0 32px;
    width: 32px;
    min-height: 32px;
  }

  .stat-details-dialog,
  .sync-logs-dialog {
    align-items: stretch;
    padding: 10px;
  }

  .stat-details-modal {
    height: calc(100vh - 20px);
    height: calc(100dvh - 20px);
    max-height: calc(100vh - 20px);
    max-height: calc(100dvh - 20px);
  }

  .stat-details-header {
    align-items: flex-start;
    padding: 12px 14px;
  }

  .stat-details-header h3 {
    font-size: 15px;
  }

  .stat-details-summary {
    padding: 10px 12px;
  }

  .stat-details-body {
    padding: 10px;
  }

  .stat-details-footer {
    padding: 10px;
  }

  .stat-details-footer .ui-button {
    width: 100%;
    justify-content: center;
  }

  .sync-logs-modal {
    height: calc(100vh - 20px);
    height: calc(100dvh - 20px);
    max-height: calc(100vh - 20px);
    max-height: calc(100dvh - 20px);
  }

  .sync-logs-modal-header {
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
  }

  .sync-logs-modal-header h3 {
    font-size: 15px;
  }

  .sync-logs-modal-header p {
    margin-top: 2px;
    font-size: 11px;
  }

  .sync-logs-modal-info {
    margin: 10px 10px 0;
    padding: 8px 10px;
    font-size: 11px;
    line-height: 1.4;
  }

  .sync-logs-table-wrap {
    display: none;
  }

  .sync-log-card-list {
    display: grid;
    flex: 1 1 auto;
    min-height: 0;
    align-content: flex-start;
    grid-auto-rows: max-content;
    gap: 8px;
    overflow: auto;
    padding: 10px;
    -webkit-overflow-scrolling: touch;
  }

  .sync-log-card {
    gap: 8px;
    padding: 10px;
  }

  .sync-log-card-head {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 4px;
  }

  .sync-log-card-status {
    gap: 5px;
  }

  .sync-log-card-head time {
    justify-self: flex-start;
    line-height: 1.25;
    white-space: normal;
  }

  .sync-log-card-main {
    grid-template-columns: minmax(0, 1fr);
    gap: 7px;
  }

  .sync-log-card-field {
    gap: 4px;
    padding: 7px;
  }

  .sync-log-card-field strong {
    line-height: 1.2;
  }

  .sync-log-card .tag-list {
    justify-content: flex-start;
    gap: 5px;
  }

  .sync-log-card .table-tag,
  .sync-log-card .log-chip,
  .sync-log-card .trigger-chip {
    min-height: 24px;
    min-width: 0;
    max-width: 100%;
    overflow: hidden;
    padding: 3px 8px;
    line-height: 18px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sync-log-card .tag-list {
    min-width: 0;
  }

  .sync-log-card .rate-compare {
    width: fit-content;
    min-height: 26px;
    max-width: 100%;
    gap: 6px;
    overflow: hidden;
    padding: 4px 8px;
    line-height: 18px;
  }

  .sync-log-card-action {
    width: 100%;
    min-height: 34px;
    justify-content: center;
  }

  .guard-left,
  .guard-controls,
  .guard-status-line {
    align-items: flex-start;
  }

  .rate-guard-panel {
    gap: 8px;
    padding: 10px;
  }

  .guard-left {
    align-items: center;
    gap: 10px;
  }

  .guard-switch {
    width: 36px;
    height: 20px;
    margin-top: 0;
  }

  .guard-switch span {
    width: 36px;
    height: 20px;
  }

  .guard-switch span::after {
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
  }

  .guard-switch.is-on span::after {
    transform: translateX(16px);
  }

  .guard-title {
    font-size: 13px;
    line-height: 1.2;
  }

  .guard-description,
  .guard-hint {
    display: none;
  }

  .guard-status-line {
    gap: 6px;
    margin-top: 5px;
    font-size: 11px;
  }

  .guard-controls {
    display: grid;
    grid-template-columns: auto minmax(72px, 92px) minmax(0, 1fr);
    align-items: center;
    gap: 8px;
  }

  .guard-ignore-control {
    grid-column: 1 / -1;
    width: 100%;
  }

  .guard-controls .ui-input {
    width: 92px;
  }

  .guard-controls .ignored-accounts-input {
    width: 100%;
  }

  .guard-controls .ui-button,
  .guard-sync-log-warning .ui-button {
    width: auto;
    min-height: 32px;
    padding: 0 10px;
    font-size: 12px;
    white-space: nowrap;
  }

  .guard-controls .ui-button {
    justify-self: flex-start;
  }

  .guard-controls .ui-button.ui-button-primary {
    grid-column: auto;
  }

  .control-label {
    white-space: normal;
  }

  .filter-row {
    display: grid;
    position: sticky;
    top: 64px;
    z-index: 20;
    grid-template-columns: 1fr;
    gap: 8px;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: #fff;
    padding: 10px;
    box-shadow: 0 8px 20px rgba(15, 23, 42, 0.06);
  }

  .filter-sticky-row {
    grid-template-columns: minmax(0, 1fr) auto auto;
    gap: 8px;
    min-width: 0;
  }

  .filter-toggle-button {
    display: inline-flex;
    min-height: 34px;
    align-items: center;
    justify-content: center;
    gap: 5px;
    border: 1px solid #dbe3ee;
    border-radius: 8px;
    background: #f8fafc;
    padding: 0 9px;
    color: #334155;
    font-size: 12px;
    font-weight: 800;
    white-space: nowrap;
  }

  .filter-toggle-button strong {
    display: inline-flex;
    min-width: 18px;
    height: 18px;
    align-items: center;
    justify-content: center;
    border-radius: 999px;
    background: #dc2626;
    color: #fff;
    font-size: 11px;
    line-height: 1;
  }

  .filter-controls {
    display: none;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    border-top: 1px solid #eef2f7;
    padding-top: 8px;
  }

  .filter-controls.is-open {
    display: grid;
  }

  .filtered-count {
    width: auto;
    justify-content: center;
    padding: 0 10px;
  }

  .records-table {
    min-width: 900px;
  }

  .accounts-table-card :deep(.mobile-row-card) {
    position: relative;
    overflow: hidden;
    border-color: #e2e8f0;
    border-radius: 8px;
    background: #fff;
    padding: 12px;
    box-shadow: none;
  }

  .accounts-table-card :deep(.mobile-row-card::before) {
    position: absolute;
    inset: 0 auto 0 0;
    width: 3px;
    background: #059669;
    content: "";
  }

  .accounts-table-card :deep(.mobile-row-card.risk-row) {
    border-color: #fed7aa;
    background: #fffaf5;
  }

  .accounts-table-card :deep(.mobile-row-card.risk-row::before) {
    background: #ea580c;
  }

  .accounts-table-card :deep(.mobile-row-card.test-failed-row) {
    border-color: #fed7aa;
    background: #fffaf5;
  }

  .accounts-table-card :deep(.mobile-row-card.test-failed-row::before) {
    background: #dc2626;
  }

  .accounts-table-card :deep(.mobile-row-card.unbound-row:not(.risk-row):not(.test-failed-row)::before) {
    background: #2563eb;
  }

  .accounts-table-card :deep(.mobile-row-card.provider-disabled-row) {
    background: #f8fafc;
  }

  .accounts-table-card :deep(.mobile-row-card.provider-disabled-row::before) {
    background: #7c3aed;
  }

  .accounts-table-card :deep(.mobile-row-card > .space-y-3 > .flex:nth-child(1)),
  .accounts-table-card :deep(.mobile-row-card > .space-y-3 > .flex:nth-child(2)),
  .accounts-table-card :deep(.mobile-row-card > .space-y-3 > .flex:nth-child(3)) {
    border-bottom: 1px solid #f1f5f9;
    padding-bottom: 8px;
  }

  .accounts-table-card :deep(.mobile-row-card > .space-y-3 > .flex > span) {
    color: #64748b;
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0;
    text-transform: none;
  }

  .accounts-table-card :deep(.mobile-row-card > .space-y-3 > .flex > div) {
    color: #0f172a;
    font-size: 13px;
  }

  .accounts-table-card :deep(.mobile-row-card .action-cell) {
    justify-content: flex-end;
    gap: 6px;
  }

  .accounts-table-card :deep(.mobile-row-card .text-action) {
    min-height: 28px;
    border: 1px solid #dbe3ee;
    border-radius: 7px;
    background: #fff;
    padding: 0 8px;
    color: #475569;
    font-size: 12px;
    line-height: 1;
  }

  .accounts-table-card :deep(.mobile-row-card .text-action-primary) {
    border-color: #bbf7d0;
    background: #ecfdf5;
    color: #047857;
  }

  .accounts-table-card :deep(.mobile-row-card .text-action-danger) {
    border-color: #fecaca;
    background: #fef2f2;
    color: #dc2626;
  }

  .accounts-action-primary {
    font-weight: 800;
  }

  .accounts-action-secondary {
    border-color: #dbe3ee;
    background: #f8fafc;
    color: #64748b;
  }

  .mobile-back-to-filters {
    position: fixed;
    right: max(16px, env(safe-area-inset-right));
    bottom: calc(18px + env(safe-area-inset-bottom));
    z-index: 35;
    display: inline-flex;
    width: 44px;
    height: 44px;
    align-items: center;
    justify-content: center;
    border: 1px solid #bbf7d0;
    border-radius: 999px;
    background: #059669;
    color: #fff;
    box-shadow: 0 14px 34px rgba(15, 23, 42, 0.18);
  }

  .mobile-back-to-filters:active {
    transform: translateY(1px);
  }
}

@media (max-width: 520px) {
  .accounts-actions .ui-button {
    flex-basis: auto;
  }

  .accounts-actions .ui-button-icon {
    flex-basis: 32px;
  }

  .guard-sync-log-warning {
    grid-template-columns: 1fr;
  }

  .guard-warning-icon {
    display: none;
  }
}

@media (max-width: 380px) {
  .guard-controls {
    grid-template-columns: 1fr;
  }

  .filter-sticky-row {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .filter-sticky-row .filtered-count {
    grid-column: 1 / -1;
    justify-content: space-between;
    width: 100%;
  }

  .filter-controls {
    grid-template-columns: 1fr;
  }

  .guard-controls .ui-input {
    width: 100%;
  }
}
</style>
