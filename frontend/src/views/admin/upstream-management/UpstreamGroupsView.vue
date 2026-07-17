<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="ug-stats-row">
          <div class="ug-stat-card">
            <span class="ug-stat-bar ug-stat-bar-primary"></span>
            <div class="ug-stat-content">
              <div class="ug-stat-label">{{ t('admin.upstreamGroups.upstreamGroups') }}</div>
              <div class="ug-stat-value">{{ summary.upstreamGroups }}</div>
            </div>
          </div>
          <div class="ug-stat-card">
            <span class="ug-stat-bar ug-stat-bar-success"></span>
            <div class="ug-stat-content">
              <div class="ug-stat-label">{{ t('admin.upstreamGroups.matchedGroups') }}</div>
              <div class="ug-stat-value">{{ summary.matchedGroups }}</div>
            </div>
          </div>
          <div class="ug-stat-card">
            <span class="ug-stat-bar ug-stat-bar-warning"></span>
            <div class="ug-stat-content">
              <div class="ug-stat-label">{{ t('admin.upstreamGroups.rateRisks') }}</div>
              <div class="ug-stat-value">{{ summary.rateRisks }}</div>
            </div>
          </div>
          <div class="ug-stat-card">
            <span class="ug-stat-bar ug-stat-bar-info"></span>
            <div class="ug-stat-content">
              <div class="ug-stat-label">{{ t('admin.upstreamGroups.filteredCount') }}</div>
              <div class="ug-stat-value">{{ filteredItems.length }}</div>
            </div>
          </div>
        </div>

        <div class="ug-filter-card" :class="{ 'ug-filters-expanded': showGroupAdvancedFilters }">
          <div class="ug-filter-top">
            <div class="ug-filter-left" :class="{ 'is-open': showGroupAdvancedFilters }">
              <Select v-model="platformFilter" class="ug-filter-select" :options="platformFilterOptions" />
              <Select v-model="matchFilter" class="ug-filter-select" :options="matchFilterOptions" />
              <Select v-model="rateFilter" class="ug-filter-select" :options="rateFilterOptions" />
            </div>
            <div class="ug-filter-main">
              <div class="ug-search">
                <Icon name="search" size="sm" class="ug-search-icon" />
                <input
                  v-model.trim="searchQuery"
                  type="search"
                  class="ug-input ug-search-input"
                  :placeholder="t('admin.upstreamGroups.searchPlaceholder')"
                />
              </div>
              <button
                type="button"
                class="ug-filter-toggle-button"
                :aria-expanded="showGroupAdvancedFilters"
                @click="showGroupAdvancedFilters = !showGroupAdvancedFilters"
              >
                <Icon name="filter" size="sm" />
                <span>{{ t('admin.upstreamGroups.mobileFilterToggle') }}</span>
                <strong v-if="activeGroupFilterCount">{{ activeGroupFilterCount }}</strong>
                <Icon :name="showGroupAdvancedFilters ? 'chevronUp' : 'chevronDown'" size="sm" />
              </button>
            </div>
            <div class="ug-filter-right">
              <button
                type="button"
                class="ug-btn ug-btn-primary"
                :disabled="loading || applying"
                @click="openCreateAccountDialog"
              >
                <Icon name="plus" size="sm" />
                <span>{{ t('admin.accounts.createAccount') }}</span>
              </button>
              <button
                type="button"
                class="ug-btn ug-btn-default"
                :disabled="loading || applying"
                :title="t('common.refresh')"
                @click="reload"
              >
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                <span>{{ t('common.refresh') }}</span>
              </button>
              <button
                type="button"
                class="ug-btn ug-btn-primary"
                :disabled="loading || applying || summary.rateRisks === 0"
                @click="openRateFixPreview"
              >
                <Icon name="sync" size="sm" :class="applying ? 'animate-spin' : ''" />
                <span>{{ t('admin.upstreamGroups.fixRates') }}</span>
              </button>
              <button type="button" class="ug-btn ug-btn-default" @click="openRateFixLogsDialog">
                <Icon name="document" size="sm" />
                <span>{{ t('admin.upstreamGroups.openRateFixLogs') }}</span>
              </button>
            </div>
          </div>
          <div class="ug-auto-row">
            <span class="ug-auto-meta">
              {{ t('admin.upstreamGroups.autoFixLastRun') }}: {{ autoFixLastRunText }}
              <button
                v-if="unhandledRateFixRecords.length"
                type="button"
                class="ug-rate-fix-warning"
                @click="openRateFixLogsDialog"
              >
                {{ t('admin.upstreamGroups.unhandledRateFixLogs') }} {{ unhandledRateFixRecords.length }}
              </button>
            </span>
            <div class="ug-auto-controls">
              <label class="ug-auto-toggle">
                <input
                  v-model="autoFixForm.enabled"
                  type="checkbox"
                  class="ug-checkbox"
                  :disabled="savingRateFixConfig || loadingRateFixConfig"
                />
                <span>{{ t('admin.upstreamGroups.autoFixEnabled') }}</span>
              </label>
              <label class="ug-auto-interval">
                <span>{{ t('admin.upstreamGroups.autoFixIntervalSeconds') }}</span>
                <input
                  id="auto-fix-interval-seconds"
                  v-model.number="autoFixForm.interval_seconds"
                  type="number"
                  min="1"
                  step="1"
                  class="ug-input ug-auto-input"
                  :disabled="savingRateFixConfig || loadingRateFixConfig"
                />
              </label>
              <button
                type="button"
                class="ug-btn ug-btn-default ug-btn-small"
                :disabled="savingRateFixConfig || loadingRateFixConfig"
                @click="saveRateFixConfig"
              >
                {{ savingRateFixConfig ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>
        </div>

        <nav class="ug-quick-tags" aria-label="quick filters">
          <button
            v-for="option in quickFilterOptions"
            :key="option.key"
            type="button"
            :class="['ug-quick-tag', { active: activeQuickFilter === option.key }, option.tone ? `ug-quick-tag-${option.tone}` : '']"
            @click="activeQuickFilter = option.key"
          >
            <span>{{ option.label }}</span>
            <strong>{{ option.count }}</strong>
          </button>
        </nav>
      </template>

      <template #table>
        <div class="ug-content">
          <div v-if="warnings.length" class="ug-warning-banner">
            <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
          </div>

          <div class="ug-table-card">
            <DataTable
              :columns="columns"
              :data="filteredItems"
              :loading="loading"
              :row-class="rowClass"
              :estimate-row-height="80"
              default-sort-key="status"
              default-sort-order="asc"
              sort-storage-key="upstream-groups-sort"
            >
              <template #cell-upstream_group_name="{ row }">
                <div class="ug-group-cell">
                  <div class="ug-group-title">
                    <span class="ug-group-name">{{ row.upstream_group_name }}</span>
                    <span class="ug-tag ug-tag-info">{{ row.provider_name || row.provider_slug }}</span>
                  </div>
                  <div class="ug-group-sub">
                    <span>{{ t('admin.upstreamGroups.keyCount', { count: row.upstream_key_count }) }}</span>
                    <span class="ug-group-sub-sep">·</span>
                    <code class="ug-group-sub-code">{{ row.upstream_group_key }}</code>
                  </div>
                </div>
              </template>

              <template #cell-upstream_rate="{ value }">
                <span :class="['ug-rate', rateToneClass(value)]">{{ formatRate(value) }}</span>
              </template>

              <template #cell-monitor_trend="{ row }">
                <UpstreamGroupAvailabilityTrend
                  :row="monitorTrendFor(row)"
                  :loading="monitorLoading"
                  :empty-text="t('admin.upstreamGroups.monitorTrendEmpty')"
                  :loading-text="t('admin.upstreamGroups.monitorTrendLoading')"
                  :label="t('admin.upstreamGroups.columns.monitorTrend')"
                />
              </template>

              <template #cell-local_group_name="{ row }">
                <div class="ug-match-cell">
                  <template v-if="row.matched">
                    <div class="ug-match-id">
                      <span>{{ row.local_group_name }}</span>
                      <span class="ug-match-id-num">#{{ row.local_group_id }}</span>
                    </div>
                    <div class="ug-match-desc">
                      <span :class="['ug-tag', row.match_source === 'manual' ? 'ug-tag-violet' : 'ug-tag-info']">
                        {{ matchSourceLabel(row) }}
                      </span>
                      <span v-if="row.needs_rate_increase" class="ug-match-desc-text ug-match-desc-warn">
                        {{ t('admin.upstreamGroups.localRateLow') }} · {{ t('admin.upstreamGroups.needsAdjust') }}
                      </span>
                      <span v-else class="ug-match-desc-text">{{ t('admin.upstreamGroups.inSync') }}</span>
                    </div>
                  </template>
                  <template v-else>
                    <span :class="['ug-tag', row.match_ignored ? 'ug-tag-default' : 'ug-tag-warning']">
                      {{ row.match_ignored ? t('admin.upstreamGroups.matchIgnored') : t('admin.upstreamGroups.notMatched') }}
                    </span>
                    <div class="ug-match-desc-text ug-match-desc-muted">{{ row.upstream_group_key }}</div>
                  </template>
                </div>
              </template>

              <template #cell-local_rate="{ row }">
                <span
                  v-if="row.local_rate !== undefined"
                  :class="['ug-rate', row.needs_rate_increase ? 'ug-rate-warning' : 'ug-rate-success']"
                >
                  {{ formatRate(row.local_rate) }}
                </span>
                <span v-else class="ug-rate-empty">-</span>
              </template>

              <template #cell-rate_delta="{ row }">
                <span
                  v-if="rateProfit(row) !== undefined"
                  :class="['ug-profit', profitClass(rateProfit(row))]"
                >
                  {{ formatProfit(rateProfit(row)) }}
                </span>
                <span v-else class="ug-rate-empty">-</span>
              </template>

              <template #cell-bound_accounts="{ row }">
                <span v-if="loadingGroupAccounts" class="ug-rate-empty">{{ t('common.loading') }}</span>
                <div v-else-if="boundAccountsFor(row).length" class="ug-account-list">
                  <span
                    v-for="account in visibleBoundAccounts(row)"
                    :key="account.id"
                    class="ug-account-chip"
                    :title="`#${account.id} ${account.name}`"
                  >
                    <span class="ug-account-chip-name">{{ account.name }}</span>
                    <span class="ug-account-chip-id">#{{ account.id }}</span>
                  </span>
                  <button
                    v-if="hiddenBoundAccountCount(row) > 0"
                    type="button"
                    class="ug-account-more ug-account-more-button"
                    :title="t('admin.upstreamGroups.viewAllBoundAccounts', '查看全部绑定账号')"
                    @click="openBoundAccountsDialog(row)"
                  >
                    +{{ hiddenBoundAccountCount(row) }} {{ t('admin.upstreamGroups.moreBoundAccounts', '更多') }}
                  </button>
                </div>
                <span v-else class="ug-rate-empty">-</span>
              </template>

              <template #cell-account_status="{ row }">
                <span v-if="loadingGroupAccounts" class="ug-rate-empty">{{ t('common.loading') }}</span>
                <div v-else-if="boundAccountsFor(row).length" class="ug-account-status-list">
                  <AccountStatusIndicator
                    v-for="account in visibleBoundAccounts(row)"
                    :key="account.id"
                    :account="account"
                    @show-temp-unsched="handleShowTempUnsched"
                  />
                  <button
                    v-if="hiddenBoundAccountCount(row) > 0"
                    type="button"
                    class="ug-account-more ug-account-more-button"
                    :title="t('admin.upstreamGroups.viewAllBoundAccounts', '查看全部绑定账号')"
                    @click="openBoundAccountsDialog(row)"
                  >
                    +{{ hiddenBoundAccountCount(row) }} {{ t('admin.upstreamGroups.moreBoundAccounts', '更多') }}
                  </button>
                </div>
                <span v-else class="ug-rate-empty">-</span>
              </template>

              <template #cell-status="{ row }">
                <span :class="['ug-status', statusClass(row)]">{{ statusLabel(row) }}</span>
              </template>

              <template #cell-action="{ row }">
                <div class="ug-action-stack">
                  <button
                    v-if="!row.matched && !row.match_ignored"
                    type="button"
                    class="ug-btn ug-btn-primary ug-btn-small ug-btn-cell"
                    :disabled="syncingGroupKey === row.upstream_group_key"
                    @click="openSyncDialog(row)"
                  >
                    <Icon name="sync" size="sm" :class="syncingGroupKey === row.upstream_group_key ? 'animate-spin' : ''" />
                    <span>{{ t('admin.upstreamGroups.syncLocalGroup') }}</span>
                  </button>
                  <button
                    v-else-if="row.match_ignored"
                    type="button"
                    class="ug-btn ug-btn-primary ug-btn-small ug-btn-cell"
                    :disabled="rematchingGroupKey === row.upstream_group_key"
                    @click="rematchGroupMapping(row)"
                  >
                    <Icon name="link" size="sm" :class="rematchingGroupKey === row.upstream_group_key ? 'animate-spin' : ''" />
                    <span>{{ t('admin.upstreamGroups.rematchGroup') }}</span>
                  </button>
                  <button
                    v-else
                    type="button"
                    class="ug-btn-text"
                    :disabled="savingLocalRateGroupId === row.local_group_id"
                    @click="openLocalRateDialog(row)"
                  >
                    {{ t('admin.upstreamGroups.editLocalRate') }}
                  </button>
                  <button
                    v-if="row.matched"
                    type="button"
                    class="ug-btn ug-btn-danger ug-btn-small ug-btn-cell"
                    :disabled="unbindingGroupKey === row.upstream_group_key"
                    @click="unbindGroupMapping(row)"
                  >
                    <Icon name="ban" size="sm" :class="unbindingGroupKey === row.upstream_group_key ? 'animate-spin' : ''" />
                    <span>{{ t('admin.upstreamGroups.unbindGroup') }}</span>
                  </button>
                  <button
                    v-if="boundAccountsFor(row).length"
                    type="button"
                    class="ug-btn ug-btn-default ug-btn-small ug-btn-cell"
                    @click="openBoundAccountsDialog(row)"
                  >
                    <Icon name="users" size="sm" />
                    <span>{{ t('admin.upstreamGroups.manageBoundAccounts', '账号管理') }}</span>
                    <span class="ug-action-count">{{ boundAccountTotal(row) }}</span>
                  </button>
                  <button
                    type="button"
                    class="ug-btn ug-btn-default ug-btn-small ug-btn-cell ug-mobile-detail-toggle"
                    @click="toggleMobileGroupDetails(row)"
                  >
                    <Icon :name="isMobileGroupDetailsExpanded(row) ? 'chevronUp' : 'chevronDown'" size="sm" />
                    <span>
                      {{ isMobileGroupDetailsExpanded(row)
                        ? t('admin.upstreamGroups.hideMobileDetails')
                        : t('admin.upstreamGroups.showMobileDetails') }}
                    </span>
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
          </div>
        </div>

        <div v-if="showRateFixLogsDialog" class="ug-rate-fix-logs-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" @click.self="closeRateFixLogsDialog">
          <div class="ug-rate-fix-logs-modal">
            <div class="ug-bound-accounts-header">
              <div>
                <h3>{{ t('admin.upstreamGroups.changeRecords') }}</h3>
                <p>{{ t('admin.upstreamGroups.latestRecords') }} {{ sortedRateFixRecords.length }}</p>
              </div>
              <button type="button" class="ug-dialog-close" :aria-label="t('common.close')" @click="closeRateFixLogsDialog">
                <Icon name="x" size="md" />
              </button>
            </div>
            <div class="ug-records-actions ug-rate-fix-logs-actions">
              <button type="button" class="ug-records-sort-btn" @click="toggleRecordsSort">
                <Icon
                  name="chevronDown"
                  size="sm"
                  :class="recordsSortOrder === 'asc' ? 'rotate-180' : ''"
                />
                <span>
                  {{ recordsSortOrder === 'desc'
                    ? t('admin.upstreamGroups.recordsSortNewest')
                    : t('admin.upstreamGroups.recordsSortOldest') }}
                </span>
              </button>
            </div>
            <div class="ug-records-table-wrapper ug-rate-fix-logs-table-wrapper">
              <div class="ug-rate-fix-log-cards">
                <article
                  v-for="record in sortedRateFixRecords"
                  :key="`card-${rateFixRecordKey(record)}`"
                  class="ug-rate-fix-log-card"
                >
                  <div class="ug-rate-fix-log-card-head">
                    <span v-if="record.handled" class="ug-rate-fix-log-status ug-rate-fix-log-status-handled">
                      {{ t('admin.upstreamGroups.rateFixLogHandled') }}
                    </span>
                    <button
                      v-else
                      type="button"
                      class="ug-rate-fix-log-status ug-rate-fix-log-status-unhandled"
                      :disabled="markingRateFixRecordKey === rateFixRecordKey(record)"
                      @click="markRateFixLogHandled(record)"
                    >
                      {{ t('admin.upstreamGroups.rateFixLogUnhandled') }}
                    </button>
                    <time>{{ formatDateTime(record.changed_at) }}</time>
                  </div>
                  <div class="ug-rate-fix-log-card-body">
                    <div>
                      <span>{{ t('admin.upstreamGroups.localGroup') }}</span>
                      <strong>{{ record.group_name }}</strong>
                    </div>
                    <div>
                      <span>{{ t('admin.upstreamGroups.upstreamGroup') }}</span>
                      <strong>{{ record.upstream_group_name }}</strong>
                    </div>
                    <div>
                      <span>{{ t('admin.upstreamGroups.oldRate') }}</span>
                      <strong class="ug-old-rate">{{ formatRate(record.old_rate) }}</strong>
                    </div>
                    <div>
                      <span>{{ t('admin.upstreamGroups.newRate') }}</span>
                      <strong class="ug-new-rate">{{ formatRate(record.new_rate) }}</strong>
                    </div>
                  </div>
                </article>
                <div v-if="!sortedRateFixRecords.length" class="ug-records-empty">{{ t('admin.upstreamGroups.noRecords') }}</div>
              </div>
              <table class="ug-records-table">
                <thead>
                  <tr>
                    <th>{{ t('admin.upstreamGroups.rateFixLogStatus') }}</th>
                    <th>{{ t('admin.upstreamGroups.localGroup') }}</th>
                    <th>{{ t('admin.upstreamGroups.upstreamGroup') }}</th>
                    <th>{{ t('admin.upstreamGroups.oldRate') }}</th>
                    <th>{{ t('admin.upstreamGroups.newRate') }}</th>
                    <th class="ug-records-time-th">{{ t('admin.upstreamGroups.changedAt') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="record in sortedRateFixRecords" :key="rateFixRecordKey(record)">
                    <td>
                      <span v-if="record.handled" class="ug-rate-fix-log-status ug-rate-fix-log-status-handled">
                        {{ t('admin.upstreamGroups.rateFixLogHandled') }}
                      </span>
                      <button
                        v-else
                        type="button"
                        class="ug-rate-fix-log-status ug-rate-fix-log-status-unhandled"
                        :disabled="markingRateFixRecordKey === rateFixRecordKey(record)"
                        @click="markRateFixLogHandled(record)"
                      >
                        {{ t('admin.upstreamGroups.rateFixLogUnhandled') }}
                      </button>
                    </td>
                    <td><span class="ug-tag ug-tag-default">{{ record.group_name }}</span></td>
                    <td><span class="ug-tag ug-tag-default">{{ record.upstream_group_name }}</span></td>
                    <td><span class="ug-old-rate">{{ formatRate(record.old_rate) }}</span></td>
                    <td><span class="ug-new-rate">{{ formatRate(record.new_rate) }}</span></td>
                    <td class="ug-records-time">{{ formatDateTime(record.changed_at) }}</td>
                  </tr>
                  <tr v-if="!sortedRateFixRecords.length">
                    <td colspan="6" class="ug-records-empty">{{ t('admin.upstreamGroups.noRecords') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div v-if="syncDialogItem" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" @click.self="closeSyncDialog">
          <div class="w-full max-w-lg overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h3 class="text-lg font-semibold text-gray-950 dark:text-white">{{ t('admin.upstreamGroups.syncDialogTitle') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.syncDialogDescription') }}</p>
            </div>
            <div class="space-y-4 px-5 py-4">
              <div class="grid gap-3 sm:grid-cols-2">
                <div>
                  <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.upstreamGroup') }}</div>
                  <div class="mt-1 break-words text-sm font-semibold text-gray-950 dark:text-white">{{ syncDialogItem.upstream_group_name }}</div>
                </div>
                <div>
                  <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.upstreamRate') }}</div>
                  <div class="mt-1 font-mono text-sm font-semibold text-gray-950 dark:text-white">{{ formatRate(syncDialogItem.upstream_rate) }}</div>
                </div>
                <div class="sm:col-span-2">
                  <label class="input-label" for="sync-local-platform">{{ t('admin.groups.form.platform') }}</label>
                  <Select
                    id="sync-local-platform"
                    v-model="syncPlatform"
                    class="mt-1"
                    :options="syncPlatformOptions"
                  />
                  <p class="input-hint">{{ t('admin.groups.platformHint') }}</p>
                </div>
                <div class="sm:col-span-2">
                  <label class="input-label" for="sync-rate-multiplier">{{ t('admin.upstreamGroups.localRate') }}</label>
                  <input
                    id="sync-rate-multiplier"
                    v-model.number="syncRateMultiplier"
                    type="number"
                    min="0.0001"
                    step="0.0001"
                    class="input mt-1"
                  />
                </div>
              </div>
            </div>
            <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="syncingGroupKey === syncDialogItem.upstream_group_key" @click="closeSyncDialog">
                {{ t('common.cancel') }}
              </button>
              <button type="button" class="btn btn-primary btn-sm" :disabled="syncingGroupKey === syncDialogItem.upstream_group_key" @click="syncLocalGroup">
                <Icon name="sync" size="sm" class="mr-1" :class="syncingGroupKey === syncDialogItem.upstream_group_key ? 'animate-spin' : ''" />
                {{ t('admin.upstreamGroups.confirmSync') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="localRateDialogItem" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" @click.self="closeLocalRateDialog">
          <div class="w-full max-w-lg overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h3 class="text-lg font-semibold text-gray-950 dark:text-white">{{ t('admin.upstreamGroups.editLocalRateTitle') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.editLocalRateDescription') }}</p>
            </div>
            <div class="space-y-4 px-5 py-4">
              <div class="grid gap-3 sm:grid-cols-2">
                <div>
                  <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.localGroup') }}</div>
                  <div class="mt-1 break-words text-sm font-semibold text-gray-950 dark:text-white">{{ localRateDialogItem.local_group_name }}</div>
                </div>
                <div>
                  <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.upstreamRate') }}</div>
                  <div class="mt-1 font-mono text-sm font-semibold text-gray-950 dark:text-white">{{ formatRate(localRateDialogItem.upstream_rate) }}</div>
                </div>
                <div class="sm:col-span-2">
                  <label class="input-label" for="local-rate-multiplier">{{ t('admin.upstreamGroups.localRate') }}</label>
                  <input
                    id="local-rate-multiplier"
                    v-model.number="localRateInput"
                    type="number"
                    min="0.0001"
                    step="0.0001"
                    class="input mt-1"
                  />
                </div>
              </div>
            </div>
            <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="savingLocalRateGroupId === localRateDialogItem.local_group_id" @click="closeLocalRateDialog">
                {{ t('common.cancel') }}
              </button>
              <button type="button" class="btn btn-primary btn-sm" :disabled="savingLocalRateGroupId === localRateDialogItem.local_group_id" @click="saveLocalGroupRate">
                <Icon name="cog" size="sm" class="mr-1" :class="savingLocalRateGroupId === localRateDialogItem.local_group_id ? 'animate-spin' : ''" />
                {{ t('common.save') }}
              </button>
            </div>
          </div>
        </div>

        <UpstreamRateFixPreviewDialog
          :show="showRateFixPreview"
          :items="rateFixPreviewItems"
          :loading="applying"
          @cancel="closeRateFixPreview"
          @confirm="confirmRateFixes"
        />

        <CreateAccountModal
          v-if="showCreateAccountModal"
          :show="showCreateAccountModal"
          :proxies="accountProxies"
          :groups="accountEditGroups"
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
        <div
          v-if="boundAccountsDialogRow"
          class="ug-bound-accounts-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
          @click.self="closeBoundAccountsDialog"
        >
          <div class="ug-bound-accounts-modal">
            <div class="ug-bound-accounts-header">
              <div>
                <h3>{{ t('admin.upstreamGroups.boundAccountsManagerTitle', '绑定账号管理') }}</h3>
                <p>
                  {{ boundAccountsDialogRow.local_group_name || boundAccountsDialogRow.upstream_group_name }}
                  · {{ t('admin.upstreamGroups.boundAccountsTotal', '共') }} {{ boundAccountTotal(boundAccountsDialogRow) }} {{ t('admin.upstreamGroups.boundAccountsUnit', '个账号') }}
                </p>
              </div>
              <button type="button" class="ug-dialog-close" :aria-label="t('common.close')" @click="closeBoundAccountsDialog">
                <Icon name="x" size="md" />
              </button>
            </div>
            <div class="ug-bound-accounts-list">
              <div
                v-for="account in boundAccountsFor(boundAccountsDialogRow)"
                :key="account.id"
                class="ug-bound-account-row"
              >
                <div class="ug-bound-account-main">
                  <div class="ug-bound-account-title">
                    <span>{{ account.name }}</span>
                    <code>#{{ account.id }}</code>
                  </div>
                  <div class="ug-bound-account-meta">
                    <span class="ug-tag ug-tag-default">{{ account.platform }}</span>
                    <span class="ug-tag ug-tag-info">{{ account.type || '-' }}</span>
                    <span>{{ account.status }}</span>
                  </div>
                </div>
                <div class="ug-bound-account-status">
                  <AccountStatusIndicator :account="account" @show-temp-unsched="handleShowTempUnsched" />
                </div>
                <div class="ug-bound-account-actions">
                  <button
                    type="button"
                    class="ug-btn-text"
                    :disabled="editingAccountId === account.id || savingAccountGroupId === account.id"
                    @click="openAccountEditDialog(account)"
                  >
                    {{ t('admin.upstreamGroups.editAccount', '编辑账号') }}
                  </button>
                  <button
                    type="button"
                    class="ug-btn-text"
                    :disabled="editingAccountId === account.id || savingAccountGroupId === account.id"
                    @click="openAccountGroupDialog(account)"
                  >
                    {{ t('admin.upstreamGroups.editAccountBinding', '编辑绑定') }}
                  </button>
                </div>
              </div>
              <div v-if="!boundAccountsFor(boundAccountsDialogRow).length" class="ug-bound-accounts-empty">
                {{ t('admin.upstreamGroups.noBoundAccounts', '暂无绑定账号') }}
              </div>
            </div>
          </div>
        </div>
        <div
          v-if="accountGroupDialogAccount"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
          @click.self="closeAccountGroupDialog"
        >
          <div class="w-full max-w-xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h3 class="text-lg font-semibold text-gray-950 dark:text-white">{{ t('admin.upstreamGroups.editAccountBindingTitle', '编辑账号绑定分组') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.editAccountBindingDescription', '添加或移除该账号绑定的本地分组。') }}</p>
            </div>
            <div class="space-y-4 px-5 py-4">
              <div class="grid gap-3 sm:grid-cols-2">
                <div>
                  <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.boundAccounts', '绑定账号') }}</div>
                  <div class="mt-1 text-sm font-semibold text-gray-950 dark:text-white">
                    #{{ accountGroupDialogAccount.id }} {{ accountGroupDialogAccount.name }}
                  </div>
                </div>
                <div>
                  <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.accounts.platform') }}</div>
                  <div class="mt-1 text-sm font-semibold text-gray-950 dark:text-white">{{ accountGroupDialogAccount.platform }}</div>
                </div>
              </div>
              <GroupSelector
                v-model="accountGroupIds"
                :groups="accountEditGroups"
                :platform="accountGroupPlatform"
                searchable
              />
            </div>
            <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="savingAccountGroupId === accountGroupDialogAccount.id" @click="closeAccountGroupDialog">
                {{ t('common.cancel') }}
              </button>
              <button type="button" class="btn btn-primary btn-sm" :disabled="savingAccountGroupId === accountGroupDialogAccount.id" @click="saveAccountGroups">
                <Icon name="cog" size="sm" class="mr-1" :class="savingAccountGroupId === accountGroupDialogAccount.id ? 'animate-spin' : ''" />
                {{ t('common.save') }}
              </button>
            </div>
          </div>
        </div>
        <TempUnschedStatusModal
          :show="showTempUnsched"
          :account="tempUnschedAccount"
          @close="closeTempUnschedModal"
          @reset="handleTempUnschedReset"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  UpstreamGroupAutoRateFixConfig,
  UpstreamGroupCompareResult,
  UpstreamGroupComparison,
  UpstreamGroupRateFixRecord
} from '@/api/admin/upstreamManagement'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useRouteQueryFilters } from '@/composables/useRouteQueryFilters'
import { formatDateTime } from '@/utils/format'
import {
  buildUpstreamMonitorTrendIndex,
  normalizeUpstreamMonitorGroupKey,
  type UpstreamMonitorTrendRow
} from '@/utils/upstreamMonitorTrend'
import type { Account, AdminGroup, GroupPlatform, Proxy as AccountProxy } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import Icon from '@/components/icons/Icon.vue'
import UpstreamGroupAvailabilityTrend from '@/components/admin/upstream/UpstreamGroupAvailabilityTrend.vue'
import UpstreamRateFixPreviewDialog from '@/components/admin/upstream/UpstreamRateFixPreviewDialog.vue'
import { AccountStatusIndicator, CreateAccountModal, EditAccountModal, TempUnschedStatusModal } from '@/components/account'

const { t } = useI18n()
const appStore = useAppStore()

type QuickGroupFilterKey = 'all' | 'unmatched' | 'risk' | 'matched'

const result = ref<UpstreamGroupCompareResult | null>(null)
const loading = ref(false)
const applying = ref(false)
const showRateFixPreview = ref(false)
const rateFixPreviewItems = ref<UpstreamGroupComparison[]>([])
const loadingRateFixConfig = ref(false)
const savingRateFixConfig = ref(false)
const syncingGroupKey = ref<string | null>(null)
const unbindingGroupKey = ref<string | null>(null)
const rematchingGroupKey = ref<string | null>(null)
const savingLocalRateGroupId = ref<number | null>(null)
const savingAccountGroupId = ref<number | null>(null)
const editingAccountId = ref<number | null>(null)
const showCreateAccountModal = ref(false)
const showEditAccountModal = ref(false)
const accountEditGroups = ref<AdminGroup[]>([])
const accountProxies = ref<AccountProxy[]>([])
const editingAccount = ref<Account | null>(null)
const accountGroupDialogAccount = ref<Account | null>(null)
const accountGroupIds = ref<number[]>([])
const accountGroupPlatform = ref<GroupPlatform | undefined>()
const boundAccountsDialogRow = ref<UpstreamGroupComparison | null>(null)
const groupAccountsByGroupId = ref<Record<number, Account[]>>({})
const groupAccountTotalsByGroupId = ref<Record<number, number>>({})
const loadingGroupAccounts = ref(false)
const showTempUnsched = ref(false)
const tempUnschedAccount = ref<Account | null>(null)
const monitorTrendIndex = ref<Map<string, UpstreamMonitorTrendRow>>(new Map())
const monitorLoading = ref(false)
const loadError = ref('')
const rateFixConfig = ref<UpstreamGroupAutoRateFixConfig | null>(null)
const autoFixForm = ref({
  enabled: false,
  interval_seconds: 3600,
})
const searchQuery = ref('')
const platformFilter = ref('')
const matchFilter = ref('')
const rateFilter = ref('')
const showGroupAdvancedFilters = ref(false)
const activeQuickFilter = ref<QuickGroupFilterKey>('all')
useRouteQueryFilters([
  { queryKey: 'provider', state: searchQuery },
  { queryKey: 'rateRisk', state: rateFilter, fromQuery: value => value === 'true' ? 'risk' : '', toQuery: value => value === 'risk' ? 'true' : undefined },
])
const expandedMobileGroupKeys = ref<Set<string>>(new Set())
const syncDialogItem = ref<UpstreamGroupComparison | null>(null)
const syncRateMultiplier = ref(1)
const syncPlatform = ref('')
const localRateDialogItem = ref<UpstreamGroupComparison | null>(null)
const localRateInput = ref(1)
const showRateFixLogsDialog = ref(false)
const markingRateFixRecordKey = ref<string | null>(null)
const recordsSortOrder = ref<'desc' | 'asc'>('desc')
let reloadRequestId = 0

const platformOptions = computed<SelectOption[]>(() => [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
])
const platformFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamGroups.allPlatforms', '全部平台') },
  ...platformOptions.value,
])
const syncPlatformOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamGroups.selectPlatform') },
  ...platformOptions.value,
])

const columns = computed<Column[]>(() => [
  { key: 'upstream_group_name', label: t('admin.upstreamGroups.columns.upstreamGroup'), class: 'ug-table-upstream-group-column', sortable: true },
  { key: 'upstream_rate', label: t('admin.upstreamGroups.columns.upstreamRate'), class: 'ug-table-rate-column', sortable: true },
  { key: 'monitor_trend', label: t('admin.upstreamGroups.columns.monitorTrend'), class: 'ug-table-monitor-column' },
  { key: 'local_group_name', label: t('admin.upstreamGroups.columns.matchResult'), class: 'ug-table-local-group-column', sortable: true },
  { key: 'local_rate', label: t('admin.upstreamGroups.columns.localRate'), class: 'ug-table-rate-column', sortable: true },
  { key: 'rate_delta', label: t('admin.upstreamGroups.columns.rateDelta'), class: 'ug-table-delta-column', sortable: true },
  { key: 'bound_accounts', label: t('admin.upstreamGroups.columns.boundAccounts', '绑定账号'), class: 'ug-table-bound-accounts-column' },
  { key: 'account_status', label: t('admin.upstreamGroups.columns.accountStatus', '账号状态'), class: 'ug-table-account-status-column' },
  { key: 'status', label: t('admin.upstreamGroups.columns.status'), class: 'ug-table-status-column', sortable: true },
  { key: 'action', label: t('admin.upstreamGroups.columns.action'), class: 'ug-table-action-column' },
])

const items = computed<UpstreamGroupComparison[]>(() => result.value?.items || [])
const warnings = computed(() => result.value?.warnings || [])
const records = computed<UpstreamGroupRateFixRecord[]>(() => result.value?.records || [])
const sortedRateFixRecords = computed<UpstreamGroupRateFixRecord[]>(() => {
  const sorted = [...records.value].sort((a, b) => {
    const aTime = recordTimestamp(a.changed_at)
    const bTime = recordTimestamp(b.changed_at)
    return recordsSortOrder.value === 'desc' ? bTime - aTime : aTime - bTime
  })
  return sorted
})
const unhandledRateFixRecords = computed(() => records.value.filter(record => !record.handled))
const autoFixLastRunText = computed(() => {
  if (!rateFixConfig.value?.last_run_at) return t('admin.upstreamGroups.autoFixNeverRun')
  const status = rateFixConfig.value.last_run_status === 'failed'
    ? t('admin.upstreamGroups.autoFixStatusFailed')
    : t('admin.upstreamGroups.autoFixStatusSuccess')
  const message = rateFixConfig.value.last_run_message ? ` - ${rateFixConfig.value.last_run_message}` : ''
  return `${formatDateTime(rateFixConfig.value.last_run_at)} - ${status}${message}`
})
const matchFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamGroups.allMatches') },
  { value: 'matched', label: t('admin.upstreamGroups.matchedOnly') },
  { value: 'unmatched', label: t('admin.upstreamGroups.unmatchedOnly') },
])
const rateFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamGroups.allRates') },
  { value: 'risk', label: t('admin.upstreamGroups.rateRiskOnly') },
  { value: 'ok', label: t('admin.upstreamGroups.rateOkOnly') },
])
const activeGroupFilterCount = computed(() => {
  let count = 0
  if (platformFilter.value) count += 1
  if (matchFilter.value) count += 1
  if (rateFilter.value) count += 1
  if (activeQuickFilter.value !== 'all') count += 1
  return count
})
const quickFilterOptions = computed<Array<{ key: QuickGroupFilterKey; label: string; count: number; tone?: string }>>(() => {
  const list = items.value
  return [
    { key: 'all', label: t('admin.upstreamGroups.quickFilterAll'), count: list.length },
    {
      key: 'unmatched',
      label: t('admin.upstreamGroups.quickFilterUnmatched'),
      count: list.filter(item => !item.matched).length,
      tone: 'warning',
    },
    {
      key: 'risk',
      label: t('admin.upstreamGroups.quickFilterRateRisk'),
      count: list.filter(item => item.needs_rate_increase).length,
      tone: 'danger',
    },
    {
      key: 'matched',
      label: t('admin.upstreamGroups.quickFilterMatched'),
      count: list.filter(item => item.matched).length,
      tone: 'success',
    },
  ]
})
const filteredItems = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return items.value
    .filter((item) => {
      if (activeQuickFilter.value === 'unmatched' && item.matched) return false
      if (activeQuickFilter.value === 'risk' && !item.needs_rate_increase) return false
      if (activeQuickFilter.value === 'matched' && !item.matched) return false
      if (matchFilter.value === 'matched' && !item.matched) return false
      if (matchFilter.value === 'unmatched' && item.matched) return false
      if (rateFilter.value === 'risk' && !item.needs_rate_increase) return false
      if (rateFilter.value === 'ok' && item.needs_rate_increase) return false
      if (platformFilter.value && !platformValues(item).includes(platformFilter.value)) return false
      if (!keyword) return true
      const haystack = [
        item.upstream_group_name,
        item.upstream_group_key,
        item.local_group_name,
        item.provider_name,
        item.provider_slug,
        item.match_source,
      ].filter(Boolean).join(' ').toLowerCase()
      return haystack.includes(keyword)
    })
    .map((item) => ({
      ...item,
      rate_delta: rateProfit(item),
      status: statusSortValue(item),
    }))
})

function platformValues(item: UpstreamGroupComparison) {
  return [
    item.local_group_platform,
    ...boundAccountsFor(item).map(account => account.platform),
  ]
    .filter((platform): platform is string => Boolean(platform))
    .map(platform => platform.toLowerCase())
}

const summary = computed(() => {
  const list = items.value
  return {
    upstreamGroups: list.length,
    matchedGroups: list.filter(item => item.matched).length,
    rateRisks: list.filter(item => item.needs_rate_increase).length,
  }
})

const emptyTitle = computed(() => {
  return loadError.value ? t('admin.upstreamGroups.emptyNoDefaultTitle') : t('admin.upstreamGroups.emptyTitle')
})

const emptyDescription = computed(() => {
  return loadError.value || t('admin.upstreamGroups.emptyDescription')
})

async function reload() {
  const requestId = ++reloadRequestId
  loading.value = true
  loadingRateFixConfig.value = true
  loadError.value = ''
  void loadMonitorTrend(requestId)
  try {
    const [groupsResult, config] = await Promise.all([
      adminAPI.upstreamManagement.getGroups(),
      adminAPI.upstreamManagement.getRateFixConfig()
    ])
    result.value = groupsResult
    applyRateFixConfig(config)
    await syncBoundAccounts(groupsResult.items || [], requestId)
  } catch (err) {
    const message = extractApiErrorMessage(err, t('admin.upstreamGroups.loadFailed'))
    loadError.value = message
    result.value = null
    groupAccountsByGroupId.value = {}
    groupAccountTotalsByGroupId.value = {}
    appStore.showError(message)
  } finally {
    loading.value = false
    loadingRateFixConfig.value = false
  }
}

async function syncBoundAccounts(groupItems: UpstreamGroupComparison[], requestId: number) {
  const groupIds = Array.from(
    new Set(
      groupItems
        .map((item) => Number(item.local_group_id))
        .filter((id) => Number.isFinite(id) && id > 0)
    )
  )
  if (!groupIds.length) {
    groupAccountsByGroupId.value = {}
    groupAccountTotalsByGroupId.value = {}
    return
  }

  loadingGroupAccounts.value = true
  try {
    const entries = await Promise.allSettled(
      groupIds.map(async (groupId) => {
        const response = await adminAPI.accounts.list(1, 100, {
          group: String(groupId),
          sort_by: 'id',
          sort_order: 'asc',
        })
        return [groupId, response] as const
      })
    )
    if (requestId !== reloadRequestId) return

    const nextAccounts: Record<number, Account[]> = {}
    const nextTotals: Record<number, number> = {}
    let hasFailure = false
    for (const entry of entries) {
      if (entry.status !== 'fulfilled') {
        hasFailure = true
        continue
      }
      const [groupId, response] = entry.value
      nextAccounts[groupId] = response.items || []
      nextTotals[groupId] = response.total ?? response.items?.length ?? 0
    }
    groupAccountsByGroupId.value = nextAccounts
    groupAccountTotalsByGroupId.value = nextTotals
    if (hasFailure) {
      appStore.showError(t('admin.upstreamGroups.boundAccountsLoadFailed', '加载绑定账号失败'))
    }
  } finally {
    if (requestId === reloadRequestId) {
      loadingGroupAccounts.value = false
    }
  }
}

async function loadMonitorTrend(requestId: number) {
  monitorLoading.value = true
  try {
    const payload = await adminAPI.groups.getUpstreamMonitorStatus({ period: '90m', board: 'hot' })
    if (requestId !== reloadRequestId) return
    monitorTrendIndex.value = buildUpstreamMonitorTrendIndex(payload)
  } catch (err) {
    if (requestId !== reloadRequestId) return
    monitorTrendIndex.value = new Map()
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.monitorTrendLoadFailed')))
  } finally {
    if (requestId === reloadRequestId) {
      monitorLoading.value = false
    }
  }
}

function openRateFixPreview() {
  rateFixPreviewItems.value = (result.value?.items || []).filter(item => item.needs_rate_increase)
  if (!rateFixPreviewItems.value.length) return
  showRateFixPreview.value = true
}

function closeRateFixPreview() {
  if (applying.value) return
  showRateFixPreview.value = false
}

async function confirmRateFixes() {
  applying.value = true
  try {
    const latestResult = await adminAPI.upstreamManagement.getGroups()
    result.value = latestResult
    rateFixPreviewItems.value = (latestResult.items || []).filter(item => item.needs_rate_increase)
    if (!rateFixPreviewItems.value.length) {
      showRateFixPreview.value = false
      appStore.showSuccess(t('admin.upstreamGroups.rateFixNoRisks'))
      return
    }
    result.value = await adminAPI.upstreamManagement.applyRateFixes()
    showRateFixPreview.value = false
    appStore.showSuccess(t('admin.upstreamGroups.fixSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.fixFailed')))
  } finally {
    applying.value = false
  }
}

async function saveRateFixConfig() {
  const intervalSeconds = Number(autoFixForm.value.interval_seconds)
  if (!Number.isInteger(intervalSeconds) || intervalSeconds < 1) {
    appStore.showError(t('admin.upstreamGroups.invalidAutoFixInterval'))
    return
  }
  savingRateFixConfig.value = true
  try {
    const config = await adminAPI.upstreamManagement.updateRateFixConfig({
      enabled: Boolean(autoFixForm.value.enabled),
      interval_seconds: intervalSeconds,
    })
    applyRateFixConfig(config)
    appStore.showSuccess(t('admin.upstreamGroups.autoFixSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.autoFixSaveFailed')))
  } finally {
    savingRateFixConfig.value = false
  }
}

function applyRateFixConfig(config: UpstreamGroupAutoRateFixConfig) {
  rateFixConfig.value = config
  autoFixForm.value = {
    enabled: Boolean(config.enabled),
    interval_seconds: normalizePositiveInteger(config.interval_seconds, 3600),
  }
}

function monitorTrendFor(row: UpstreamGroupComparison) {
  const keys = [
    row.upstream_group_name,
    row.upstream_group_key,
    row.local_group_name,
  ]
  for (const key of keys) {
    const trendRow = monitorTrendIndex.value.get(normalizeUpstreamMonitorGroupKey(key))
    if (trendRow) return trendRow
  }
  return undefined
}

function openSyncDialog(row: UpstreamGroupComparison) {
  syncDialogItem.value = row
  syncRateMultiplier.value = normalizePositiveRate(row.upstream_rate, 1)
  syncPlatform.value = ''
}

function closeSyncDialog() {
  if (syncingGroupKey.value) return
  syncDialogItem.value = null
  syncPlatform.value = ''
}

function openLocalRateDialog(row: UpstreamGroupComparison) {
  if (!row.local_group_id) return
  localRateDialogItem.value = row
  localRateInput.value = normalizePositiveRate(row.local_rate, 1)
}

function closeLocalRateDialog() {
  if (savingLocalRateGroupId.value) return
  localRateDialogItem.value = null
}

async function saveLocalGroupRate() {
  const row = localRateDialogItem.value
  if (!row?.local_group_id) return
  const rate = Number(localRateInput.value)
  if (!Number.isFinite(rate) || rate <= 0) {
    appStore.showError(t('admin.upstreamGroups.invalidRate'))
    return
  }
  savingLocalRateGroupId.value = row.local_group_id
  try {
    await adminAPI.groups.update(row.local_group_id, { rate_multiplier: rate })
    localRateDialogItem.value = null
    await reload()
    appStore.showSuccess(t('admin.upstreamGroups.localRateSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.localRateSaveFailed')))
  } finally {
    savingLocalRateGroupId.value = null
  }
}

async function unbindGroupMapping(row: UpstreamGroupComparison) {
  unbindingGroupKey.value = row.upstream_group_key
  try {
    const requestId = ++reloadRequestId
    const nextResult = await adminAPI.upstreamManagement.saveGroupMapping(row.upstream_group_name, null, true)
    result.value = nextResult
    await syncBoundAccounts(nextResult.items || [], requestId)
    appStore.showSuccess(t('admin.upstreamGroups.unbindGroupSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.unbindGroupFailed')))
  } finally {
    unbindingGroupKey.value = null
  }
}

async function rematchGroupMapping(row: UpstreamGroupComparison) {
  rematchingGroupKey.value = row.upstream_group_key
  try {
    const requestId = ++reloadRequestId
    const nextResult = await adminAPI.upstreamManagement.saveGroupMapping(row.upstream_group_name, null)
    result.value = nextResult
    await syncBoundAccounts(nextResult.items || [], requestId)
    appStore.showSuccess(t('admin.upstreamGroups.rematchGroupSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.rematchGroupFailed')))
  } finally {
    rematchingGroupKey.value = null
  }
}

async function loadAccountEditOptions() {
  const [proxies, groups] = await Promise.all([
    adminAPI.proxies.getAll(),
    adminAPI.groups.getAll()
  ])
  accountProxies.value = proxies
  accountEditGroups.value = groups
}

async function openCreateAccountDialog() {
  try {
    await loadAccountEditOptions()
    showCreateAccountModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
  }
}

function closeCreateAccountDialog() {
  showCreateAccountModal.value = false
}

async function handleAccountCreated() {
  showCreateAccountModal.value = false
  await reload()
}

async function openAccountEditDialog(account: Account) {
  editingAccountId.value = account.id
  try {
    await loadAccountEditOptions()
    editingAccount.value = account
    showEditAccountModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
  } finally {
    editingAccountId.value = null
  }
}

function closeAccountEditDialog() {
  showEditAccountModal.value = false
  editingAccount.value = null
}

async function handleAccountUpdated() {
  showEditAccountModal.value = false
  editingAccount.value = null
  await reload()
}

async function openAccountGroupDialog(account: Account) {
  try {
    if (!accountEditGroups.value.length) {
      await loadAccountEditOptions()
    }
    accountGroupDialogAccount.value = account
    accountGroupIds.value = [...(account.group_ids || account.groups?.map(group => group.id) || [])]
    accountGroupPlatform.value = account.platform
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
  }
}

function closeAccountGroupDialog() {
  if (savingAccountGroupId.value) return
  accountGroupDialogAccount.value = null
  accountGroupIds.value = []
  accountGroupPlatform.value = undefined
}

function openBoundAccountsDialog(row: UpstreamGroupComparison) {
  boundAccountsDialogRow.value = row
}

function closeBoundAccountsDialog() {
  boundAccountsDialogRow.value = null
}

function openRateFixLogsDialog() {
  showRateFixLogsDialog.value = true
}

function closeRateFixLogsDialog() {
  if (markingRateFixRecordKey.value) return
  showRateFixLogsDialog.value = false
}

async function saveAccountGroups() {
  const account = accountGroupDialogAccount.value
  if (!account) return
  savingAccountGroupId.value = account.id
  try {
    await adminAPI.accounts.update(account.id, { group_ids: accountGroupIds.value })
    closeAccountGroupDialog()
    appStore.showSuccess(t('admin.upstreamAccounts.boundGroupsSaved'))
    await reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.boundGroupsSaveFailed')))
  } finally {
    savingAccountGroupId.value = null
  }
}

function handleShowTempUnsched(account: Account) {
  tempUnschedAccount.value = account
  showTempUnsched.value = true
}

function handleTempUnschedReset() {
  tempUnschedAccount.value = null
  showTempUnsched.value = false
  void reload()
}

function closeTempUnschedModal() {
  tempUnschedAccount.value = null
  showTempUnsched.value = false
}

async function syncLocalGroup() {
  if (!syncDialogItem.value) return
  const rate = Number(syncRateMultiplier.value)
  if (!Number.isFinite(rate) || rate <= 0) {
    appStore.showError(t('admin.upstreamGroups.invalidRate'))
    return
  }
  if (!syncPlatform.value) {
    appStore.showError(t('admin.upstreamGroups.invalidPlatform'))
    return
  }
  const item = syncDialogItem.value
  syncingGroupKey.value = item.upstream_group_key
  try {
    result.value = await adminAPI.upstreamManagement.createLocalGroupFromUpstream({
      upstream_group_name: item.upstream_group_name,
      platform: syncPlatform.value,
      rate_multiplier: rate,
    })
    syncDialogItem.value = null
    syncPlatform.value = ''
    appStore.showSuccess(t('admin.upstreamGroups.syncSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.syncFailed')))
  } finally {
    syncingGroupKey.value = null
  }
}

function toggleRecordsSort() {
  recordsSortOrder.value = recordsSortOrder.value === 'desc' ? 'asc' : 'desc'
}

function recordTimestamp(value: string | number | undefined) {
  if (typeof value === 'number') return value
  if (typeof value === 'string') {
    const parsed = Date.parse(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return 0
}

function toRFC3339(value: string) {
  if (!value) return value
  const parsed = new Date(value)
  if (!Number.isFinite(parsed.getTime())) return value
  return parsed.toISOString().replace(/\.\d+Z$/, 'Z')
}

function rateFixRecordKey(record: UpstreamGroupRateFixRecord) {
  return `${toRFC3339(record.changed_at)}-${record.group_id}-${record.provider_slug}-${record.upstream_group_name}`
}

async function markRateFixLogHandled(record: UpstreamGroupRateFixRecord) {
  const key = rateFixRecordKey(record)
  markingRateFixRecordKey.value = key
  try {
    const nextRecords = await adminAPI.upstreamManagement.markRateFixRecordHandled(key)
    if (result.value) {
      result.value = {
        ...result.value,
        records: nextRecords,
      }
    }
    appStore.showSuccess(t('admin.upstreamGroups.rateFixLogMarkedHandled'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.rateFixLogMarkHandledFailed')))
  } finally {
    markingRateFixRecordKey.value = null
  }
}

function normalizePositiveRate(value: number | undefined, fallback: number) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : fallback
}

function normalizePositiveInteger(value: number | undefined, fallback: number) {
  const n = Number(value)
  return Number.isInteger(n) && n > 0 ? n : fallback
}

function localGroupId(row: UpstreamGroupComparison) {
  const id = Number(row.local_group_id)
  return Number.isFinite(id) && id > 0 ? id : null
}

function boundAccountsFor(row: UpstreamGroupComparison) {
  const id = localGroupId(row)
  return id ? groupAccountsByGroupId.value[id] || [] : []
}

function visibleBoundAccounts(row: UpstreamGroupComparison) {
  return boundAccountsFor(row).slice(0, 4)
}

function hiddenBoundAccountCount(row: UpstreamGroupComparison) {
  const id = localGroupId(row)
  const shown = visibleBoundAccounts(row).length
  const total = id ? groupAccountTotalsByGroupId.value[id] ?? boundAccountsFor(row).length : 0
  return Math.max(0, total - shown)
}

function boundAccountTotal(row: UpstreamGroupComparison) {
  const id = localGroupId(row)
  return id ? groupAccountTotalsByGroupId.value[id] ?? boundAccountsFor(row).length : boundAccountsFor(row).length
}

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${formatRateNumber(n)}x` : '-'
}

function rateProfit(row: UpstreamGroupComparison) {
  const upstream = Number(row.upstream_rate)
  const local = Number(row.local_rate)
  if (!Number.isFinite(upstream) || !Number.isFinite(local)) return undefined
  return local - upstream
}

function formatProfit(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  if (Math.abs(n) <= 0.0000001) return '0.00x'
  return `${n > 0 ? '+' : ''}${formatRateNumber(n)}x`
}

function formatRateNumber(value: number) {
  const normalized = Math.abs(value) <= 0.0000001 ? 0 : value
  const [integerPart, fractionPart = ''] = normalized.toFixed(6).split('.')
  const trimmedFraction = fractionPart.replace(/0+$/, '')
  return `${integerPart}.${trimmedFraction.padEnd(2, '0')}`
}

function profitClass(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return 'ug-profit-neutral'
  if (n > 0.0000001) return 'ug-profit-positive'
  if (n < -0.0000001) return 'ug-profit-negative'
  return 'ug-profit-neutral'
}

function matchSourceLabel(row: UpstreamGroupComparison) {
  if (row.match_source === 'manual') return t('admin.upstreamGroups.manualMatched')
  return t('admin.upstreamGroups.nameMatched')
}

function mobileGroupKey(row: UpstreamGroupComparison) {
  return `${row.provider_slug}:${row.upstream_group_key}:${row.local_group_id || 0}`
}

function isMobileGroupDetailsExpanded(row: UpstreamGroupComparison) {
  return expandedMobileGroupKeys.value.has(mobileGroupKey(row))
}

function toggleMobileGroupDetails(row: UpstreamGroupComparison) {
  const key = mobileGroupKey(row)
  const nextKeys = new Set(expandedMobileGroupKeys.value)
  if (nextKeys.has(key)) {
    nextKeys.delete(key)
  } else {
    nextKeys.add(key)
  }
  expandedMobileGroupKeys.value = nextKeys
}

function rowClass(row: UpstreamGroupComparison) {
  const classes = ['ug-mobile-row-card']
  if (isMobileGroupDetailsExpanded(row)) classes.push('ug-mobile-row-expanded')
  if (!row.matched) classes.push('ug-row-unmatched')
  else if (row.needs_rate_increase) classes.push('ug-row-risk')
  else classes.push('ug-row-ok')
  return classes
}

function rateToneClass(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return ''
  if (n >= 2) return 'ug-rate-purple'
  if (n > 1) return 'ug-rate-primary'
  return 'ug-rate-success'
}

function statusClass(row: UpstreamGroupComparison) {
  if (!row.matched) return 'ug-status-muted'
  if (row.needs_rate_increase) return 'ug-status-warning'
  return 'ug-status-success'
}

function statusLabel(row: UpstreamGroupComparison) {
  if (row.match_ignored) return t('admin.upstreamGroups.matchIgnored')
  if (!row.matched) return t('admin.upstreamGroups.notMatched')
  if (row.needs_rate_increase) return t('admin.upstreamGroups.rateRiskStatus')
  return t('admin.upstreamGroups.rateOkStatus')
}

function statusSortValue(row: UpstreamGroupComparison) {
  if (row.matched && row.needs_rate_increase) return 0
  if (!row.matched) return 1
  return 2
}

onMounted(reload)
</script>

<style scoped>
.ug-stats-row {
  @apply grid grid-cols-2 gap-3 sm:grid-cols-4;
}

.ug-stat-card {
  @apply flex items-center gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm transition-shadow dark:border-dark-600 dark:bg-dark-800/60;
}

.ug-stat-card:hover {
  @apply shadow-md;
}

.ug-stat-bar {
  @apply h-9 w-1 rounded-full;
}

.ug-stat-bar-primary { background-color: #00B42A; }
.ug-stat-bar-success { background-color: #00B42A; }
.ug-stat-bar-warning { background-color: #FF7D00; }
.ug-stat-bar-info { background-color: #165DFF; }

.ug-stat-content {
  @apply flex-1 min-w-0;
}

.ug-stat-label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.ug-stat-value {
  @apply mt-1 font-mono text-xl font-semibold text-gray-900 dark:text-white;
}

.ug-filter-card {
  @apply mt-3 rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800/40;
}

.ug-filter-top {
  @apply flex flex-wrap items-center gap-3;
}

.ug-filter-main {
  @apply flex min-w-0 flex-1 items-center gap-2;
  order: 2;
}

.ug-search {
  @apply relative min-w-0 flex-1;
}

.ug-search-icon {
  @apply absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500;
}

.ug-search-input {
  @apply w-full pl-9;
}

.ug-input {
  @apply h-9 rounded-md border border-gray-200 bg-white px-3 text-sm text-gray-900 outline-none transition-colors focus:border-primary-500 focus:ring-1 focus:ring-primary-500/20 dark:border-dark-600 dark:bg-dark-900 dark:text-white;
}

.ug-filter-right {
  @apply flex flex-wrap items-center gap-2;
  order: 3;
}

.ug-filter-left {
  @apply flex flex-wrap items-center gap-2;
  order: 1;
}

.ug-filter-toggle-button {
  display: none;
}

.ug-filter-select {
  @apply w-32;
}

.ug-quick-tags {
  @apply mt-3 flex flex-wrap items-center gap-2;
}

.ug-quick-tag {
  @apply inline-flex h-8 items-center gap-2 rounded-md border border-gray-200 bg-white px-3 text-xs font-semibold text-gray-600 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800/60 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300;
}

.ug-quick-tag strong {
  @apply inline-flex min-w-5 items-center justify-center rounded-full bg-gray-100 px-1.5 font-mono text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-300;
}

.ug-quick-tag.active {
  border-color: #00B42A;
  background: #E8FFEA;
  color: #008A22;
}

.ug-quick-tag.active strong {
  background: rgba(0, 180, 42, 0.14);
  color: #007A1D;
}

.ug-quick-tag-warning.active {
  border-color: #FFB46B;
  background: #FFF7E8;
  color: #B25A00;
}

.ug-quick-tag-danger.active {
  border-color: #FCA5A5;
  background: #FEF2F2;
  color: #DC2626;
}

.ug-quick-tag-success.active {
  border-color: #A7F3D0;
  background: #ECFDF5;
  color: #047857;
}

.ug-btn {
  @apply inline-flex h-9 items-center gap-1.5 rounded-md px-3 text-sm font-medium transition-all duration-150;
}

.ug-btn:hover:not(:disabled) {
  @apply -translate-y-px;
}

.ug-btn:active:not(:disabled) {
  @apply translate-y-0;
}

.ug-btn:disabled {
  @apply cursor-not-allowed opacity-60;
}

.ug-btn-primary {
  background-color: #00B42A;
  @apply text-white;
}

.ug-btn-primary:hover:not(:disabled) {
  background-color: #00A026;
}

.ug-btn-default {
  @apply border border-gray-200 bg-white text-gray-700 dark:border-dark-600 dark:bg-dark-900 dark:text-gray-200;
}

.ug-btn-default:hover:not(:disabled) {
  @apply border-primary-400 text-primary-600 dark:border-primary-500 dark:text-primary-300;
}

.ug-btn-danger {
  border: 1px solid #FCA5A5;
  background: #FEF2F2;
  color: #DC2626;
}

.ug-btn-danger:hover:not(:disabled) {
  border-color: #F87171;
  background: #FEE2E2;
  color: #B91C1C;
}

.ug-btn-small {
  @apply h-8 px-3 text-xs;
}

.ug-btn-cell {
  @apply whitespace-nowrap;
}

.ug-mobile-detail-toggle {
  display: none;
}

.ug-btn-text {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  @apply text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300;
}

.ug-btn-text:disabled {
  @apply cursor-not-allowed opacity-60;
}

.ug-auto-row {
  @apply mt-3 flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 pt-3 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400;
}

.ug-auto-meta {
  @apply flex min-w-0 flex-wrap items-center gap-2;
}

.ug-rate-fix-warning {
  @apply inline-flex items-center rounded-md border px-2 py-1 text-xs font-semibold transition-colors;
  border-color: #FFB46B;
  background: #FFF7E8;
  color: #B25A00;
}

.ug-rate-fix-warning:hover {
  border-color: #FF7D00;
  background: #FFF3E8;
  color: #873800;
}

.ug-auto-controls {
  @apply flex flex-wrap items-center gap-3;
}

.ug-auto-toggle {
  @apply inline-flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200;
}

.ug-checkbox {
  @apply h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500;
}

.ug-auto-interval {
  @apply inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300;
}

.ug-auto-input {
  @apply h-8 w-20 px-2 text-sm;
}

.ug-content {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.ug-warning-banner {
  background: #FFF7E8;
  border: 1px solid #FFE4B3;
  color: #B25A00;
  @apply mb-3 rounded-lg px-4 py-2 text-sm;
}

.ug-table-card {
  @apply flex flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800/30;
  flex: 1 1 auto;
  min-height: 0;
}

.ug-table-card :deep(.table-wrapper) {
  @apply min-h-0;
  flex: 1 1 auto;
}

.ug-table-card :deep(.table-wrapper) {
  border-radius: 0.5rem;
}

.ug-table-card :deep(table) {
  border-collapse: collapse;
  table-layout: fixed;
  width: max(100%, 1560px);
  min-width: 1560px;
}

.ug-table-card :deep(th),
.ug-table-card :deep(td) {
  vertical-align: middle;
}

.ug-table-card :deep(.ug-table-upstream-group-column) {
  width: 220px;
  white-space: normal;
}

.ug-table-card :deep(.ug-table-rate-column) {
  width: 105px;
}

.ug-table-card :deep(.ug-table-monitor-column) {
  width: 160px;
  white-space: normal;
}

.ug-table-card :deep(.ug-table-local-group-column) {
  width: 220px;
  white-space: normal;
}

.ug-table-card :deep(.ug-table-delta-column) {
  width: 105px;
}

.ug-table-card :deep(.ug-table-bound-accounts-column) {
  width: 240px;
  white-space: normal;
}

.ug-table-card :deep(.ug-table-account-status-column) {
  width: 220px;
  white-space: normal;
}

.ug-table-card :deep(.ug-table-status-column) {
  width: 105px;
}

.ug-table-card :deep(.ug-table-action-column) {
  width: 170px;
  white-space: normal;
}

.ug-table-card :deep(tr.ug-row-unmatched) > td:first-child {
  border-left: 3px solid #FF7D00;
}

.ug-table-card :deep(tr.ug-row-risk) > td:first-child {
  border-left: 3px solid #F53F3F;
}

.ug-table-card :deep(tr.ug-row-ok) > td:first-child {
  border-left: 3px solid transparent;
}

.ug-group-cell {
  @apply flex flex-col gap-1 leading-tight;
  overflow-wrap: anywhere;
}

.ug-group-title {
  @apply flex min-w-0 flex-wrap items-center gap-2;
}

.ug-group-name {
  @apply font-semibold text-gray-900 dark:text-white;
  overflow-wrap: anywhere;
}

.ug-group-sub {
  @apply flex flex-wrap items-center gap-1 text-xs text-gray-500 dark:text-gray-400;
}

.ug-group-sub-sep {
  @apply text-gray-300 dark:text-gray-600;
}

.ug-group-sub-code {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300;
  overflow-wrap: anywhere;
  white-space: normal;
}

.ug-match-cell {
  @apply flex flex-col gap-1.5 leading-tight;
  overflow-wrap: anywhere;
}

.ug-account-list,
.ug-account-status-list,
.ug-action-stack {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.ug-account-status-list {
  align-items: flex-start;
}

.ug-action-stack {
  flex-direction: column;
  align-items: flex-start;
}

.ug-account-chip {
  display: inline-flex;
  max-width: 100%;
  align-items: center;
  gap: 5px;
  border-radius: 6px;
  background: #f1f5f9;
  padding: 2px 8px;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
}

.ug-account-chip-name {
  min-width: 0;
  overflow-wrap: anywhere;
}

.ug-account-chip-id {
  flex: none;
  color: #94a3b8;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

.ug-account-more {
  display: inline-flex;
  align-items: center;
  border: 0;
  border-radius: 6px;
  background: #e5e7eb;
  padding: 2px 7px;
  color: #64748b;
  font-size: 12px;
  font-weight: 650;
  line-height: 18px;
}

.ug-account-more-button {
  cursor: pointer;
  transition: background 150ms ease, color 150ms ease;
}

.ug-account-more-button:hover {
  background: #dbeafe;
  color: #1d4ed8;
}

.ug-account-action-row {
  display: flex;
  max-width: 100%;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.ug-account-action-name {
  color: #94a3b8;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

.ug-action-count {
  display: inline-flex;
  min-width: 18px;
  justify-content: center;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.08);
  padding: 0 5px;
  font-size: 11px;
  font-weight: 700;
}

.ug-bound-accounts-modal {
  display: flex;
  width: min(980px, 100%);
  max-height: min(78vh, 760px);
  flex-direction: column;
  overflow: hidden;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.28);
}

.ug-bound-accounts-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid #e5e7eb;
  padding: 18px 20px;
}

.ug-bound-accounts-header h3 {
  margin: 0;
  color: #111827;
  font-size: 16px;
  font-weight: 750;
}

.ug-bound-accounts-header p {
  margin: 4px 0 0;
  color: #64748b;
  font-size: 12px;
}

.ug-dialog-close {
  display: inline-flex;
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  color: #64748b;
}

.ug-dialog-close:hover {
  border-color: #cbd5e1;
  background: #f8fafc;
  color: #111827;
}

.ug-bound-accounts-list {
  flex: 1 1 auto;
  overflow: auto;
  padding: 12px;
}

.ug-bound-account-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(130px, auto) auto;
  gap: 12px;
  align-items: center;
  border-bottom: 1px solid #eef2f7;
  padding: 12px 8px;
}

.ug-bound-account-row:last-child {
  border-bottom: 0;
}

.ug-bound-account-main {
  min-width: 0;
}

.ug-bound-account-title {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: #111827;
  font-size: 14px;
  font-weight: 700;
}

.ug-bound-account-title code {
  color: #94a3b8;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

.ug-bound-account-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
  color: #64748b;
  font-size: 12px;
}

.ug-bound-account-status {
  display: flex;
  justify-content: flex-start;
}

.ug-bound-account-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.ug-bound-accounts-empty {
  padding: 36px 12px;
  color: #64748b;
  text-align: center;
  font-size: 13px;
}

:global(.dark) .ug-account-chip {
  background: #334155;
  color: #e2e8f0;
}

:global(.dark) .ug-account-more {
  background: #1f2937;
  color: #cbd5e1;
}

.ug-match-id {
  @apply flex flex-wrap items-center gap-1.5 text-sm font-semibold text-gray-900 dark:text-white;
}

.ug-match-id-num {
  @apply font-mono text-xs font-normal text-gray-400 dark:text-gray-500;
}

.ug-match-desc {
  @apply flex flex-wrap items-center gap-1.5;
}

.ug-match-desc-text {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.ug-match-desc-warn {
  color: #F53F3F;
}

.ug-match-desc-muted {
  @apply text-xs text-gray-400 dark:text-gray-500;
}

.ug-tag {
  @apply inline-flex items-center rounded px-2 py-0.5 text-xs font-medium;
}

.ug-tag-info {
  background-color: #E8F3FF;
  color: #165DFF;
}

.ug-tag-violet {
  background-color: #F2EBFF;
  color: #722ED1;
}

.ug-tag-warning {
  background-color: #FFF3E8;
  color: #FF7D00;
}

.ug-tag-default {
  @apply bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

:global(.dark) .ug-tag-info {
  background-color: rgba(22, 93, 255, 0.15);
  color: #6FAAFF;
}

:global(.dark) .ug-tag-violet {
  background-color: rgba(114, 46, 209, 0.18);
  color: #B58BE6;
}

:global(.dark) .ug-tag-warning {
  background-color: rgba(255, 125, 0, 0.16);
  color: #FFB46B;
}

.ug-rate {
  @apply inline-flex font-mono text-sm font-semibold;
}

.ug-rate-text {
  @apply text-gray-900 dark:text-gray-100;
}

.ug-rate-success {
  color: #00B42A;
}

.ug-rate-warning {
  color: #FF7D00;
}

.ug-rate-primary {
  color: #165DFF;
}

.ug-rate-purple {
  color: #722ED1;
}

.ug-rate-empty {
  @apply font-mono text-sm text-gray-400 dark:text-gray-500;
}

.ug-profit {
  @apply font-mono text-sm font-semibold;
}

.ug-profit-positive {
  color: #00B42A;
}

.ug-profit-negative {
  color: #F53F3F;
}

.ug-profit-neutral {
  @apply text-gray-500 dark:text-gray-400;
}

.ug-status {
  @apply inline-flex rounded-full px-2.5 py-1 text-xs font-bold;
}

.ug-status-success {
  background-color: #E8FFEA;
  color: #00B42A;
}

.ug-status-warning {
  background-color: #FFECE8;
  color: #F53F3F;
}

.ug-status-muted {
  @apply bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300;
}

:global(.dark) .ug-status-success {
  background-color: rgba(0, 180, 42, 0.18);
  color: #6FE08A;
}

:global(.dark) .ug-status-warning {
  background-color: rgba(245, 63, 63, 0.18);
  color: #FF8C8C;
}

.ug-records-actions {
  @apply flex items-center gap-2;
}

.ug-records-sort-btn {
  @apply inline-flex h-7 items-center gap-1 rounded-md border border-gray-200 bg-white px-2 text-xs font-medium text-gray-600 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-900 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300;
}

.ug-records-table-wrapper {
  @apply max-h-72 overflow-auto;
}

.ug-rate-fix-log-cards {
  display: none;
}

.ug-rate-fix-log-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 12px;
}

.ug-rate-fix-log-card + .ug-rate-fix-log-card {
  margin-top: 8px;
}

.ug-rate-fix-log-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-bottom: 1px solid #f1f5f9;
  padding-bottom: 8px;
}

.ug-rate-fix-log-card-head time {
  color: #64748b;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.ug-rate-fix-log-card-body {
  display: grid;
  gap: 8px;
  padding-top: 10px;
}

.ug-rate-fix-log-card-body > div {
  display: grid;
  grid-template-columns: minmax(76px, auto) minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
}

.ug-rate-fix-log-card-body span {
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
}

.ug-rate-fix-log-card-body strong {
  min-width: 0;
  overflow-wrap: anywhere;
  color: #111827;
  font-size: 13px;
  text-align: right;
}

.ug-records-table {
  @apply w-full min-w-[760px] divide-y divide-gray-100 text-sm dark:divide-dark-700;
}

.ug-records-table thead {
  background-color: #FAFBFC;
  @apply text-xs font-medium text-gray-500 dark:bg-dark-800/60 dark:text-gray-400;
}

.ug-records-table th {
  @apply px-4 py-2 text-left;
}

.ug-records-table tbody tr {
  @apply transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40;
}

.ug-records-table tbody td {
  @apply px-4 py-2;
}

.ug-old-rate {
  @apply font-mono text-sm text-gray-400 line-through;
}

.ug-new-rate {
  @apply font-mono text-sm font-semibold;
  color: #00B42A;
}

.ug-records-time-th,
.ug-records-time {
  @apply text-right tabular-nums text-gray-500 dark:text-gray-400;
}

.ug-records-empty {
  @apply px-4 py-8 text-center text-sm text-gray-400 dark:text-gray-500;
}

.ug-rate-fix-logs-modal {
  @apply flex w-full overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800;
  max-width: min(980px, 100%);
  height: 80vh;
  max-height: 80vh;
  flex-direction: column;
}

.ug-rate-fix-logs-actions {
  @apply border-b border-gray-100 px-4 py-3 dark:border-dark-700;
}

.ug-rate-fix-logs-table-wrapper {
  @apply max-h-none flex-1;
  min-height: 0;
  height: 100%;
}

.ug-rate-fix-log-status {
  @apply inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold transition-colors;
}

.ug-rate-fix-log-status-unhandled {
  border-color: #FFB46B;
  background: #FFF7E8;
  color: #B25A00;
  cursor: pointer;
}

.ug-rate-fix-log-status-unhandled:hover:not(:disabled) {
  border-color: #FF7D00;
  background: #FFF3E8;
  color: #873800;
}

.ug-rate-fix-log-status-unhandled:disabled {
  cursor: wait;
  opacity: 0.65;
}

.ug-rate-fix-log-status-handled {
  border-color: #A7F3D0;
  background: #ECFDF5;
  color: #047857;
}

:global(.dark) .ug-rate-fix-warning,
:global(.dark) .ug-rate-fix-log-status-unhandled {
  border-color: rgba(255, 125, 0, 0.45);
  background: rgba(255, 125, 0, 0.16);
  color: #FFB46B;
}

:global(.dark) .ug-rate-fix-log-status-handled {
  border-color: rgba(0, 180, 42, 0.35);
  background: rgba(0, 180, 42, 0.18);
  color: #6FE08A;
}

:global(.dark) .ug-rate-fix-log-card {
  border-color: #334155;
  background: #111827;
}

:global(.dark) .ug-rate-fix-log-card-head {
  border-color: #1f2937;
}

:global(.dark) .ug-rate-fix-log-card-head time,
:global(.dark) .ug-rate-fix-log-card-body span {
  color: #94a3b8;
}

:global(.dark) .ug-rate-fix-log-card-body strong {
  color: #e5e7eb;
}

@media (max-width: 1023px) {
  .ug-stats-row {
    @apply grid-cols-2;
  }

  .ug-filter-top,
  .ug-auto-row {
    @apply items-stretch;
  }

  .ug-search,
  .ug-filter-main,
  .ug-filter-left,
  .ug-auto-meta,
  .ug-auto-controls {
    @apply w-full;
  }

  .ug-filter-right {
    @apply w-full justify-start;
  }

  .ug-filter-select {
    @apply w-full flex-1;
  }

  .ug-filter-right .ug-btn {
    flex: 0 0 auto;
    justify-content: center;
  }

  .ug-auto-controls {
    @apply justify-start;
  }

  .ug-auto-interval {
    @apply flex-wrap;
  }

  .ug-auto-input {
    @apply w-28;
  }

  .ug-content {
    @apply h-auto overflow-visible;
  }

  .ug-table-card {
    @apply h-auto min-h-0 overflow-visible;
  }

  .ug-group-name,
  .ug-tag,
  .ug-match-desc-text,
  .ug-account-chip-name {
    overflow-wrap: anywhere;
    white-space: normal;
  }

  .ug-account-list,
  .ug-account-status-list,
  .ug-match-desc,
  .ug-group-title,
  .ug-action-stack {
    justify-content: flex-end;
  }

  .ug-action-stack {
    align-items: flex-end;
  }

  .ug-bound-account-row {
    grid-template-columns: 1fr;
    align-items: flex-start;
  }

  .ug-bound-account-actions {
    justify-content: flex-start;
  }

  .ug-bound-accounts-modal,
  .ug-rate-fix-logs-modal {
    max-height: 86vh;
  }

  .ug-bound-accounts-header {
    align-items: flex-start;
    padding: 14px 16px;
  }

  .ug-records-table-wrapper {
    max-width: 100%;
    overflow: auto;
  }

  .ug-records-table {
    min-width: 760px;
  }
}

@media (max-width: 767px) {
  .ug-stats-row {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
  }

  .ug-stat-card {
    min-height: 52px;
    gap: 5px;
    padding: 8px 6px;
  }

  .ug-stat-bar {
    width: 3px;
    height: 28px;
  }

  .ug-stat-label {
    font-size: 10px;
    line-height: 1.15;
  }

  .ug-stat-value {
    margin-top: 2px;
    font-size: 16px;
    line-height: 1;
  }

  .ug-filter-main {
    order: 1;
    display: grid;
    width: 100%;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 8px;
  }

  .ug-filter-toggle-button {
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

  .ug-filter-toggle-button strong {
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

  .ug-filter-left {
    order: 2;
    display: none;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }

  .ug-filter-left.is-open {
    display: grid;
  }

  .ug-filter-right {
    order: 3;
  }

  .ug-quick-tags {
    flex-wrap: nowrap;
    overflow-x: auto;
    padding-bottom: 2px;
    scrollbar-width: none;
  }

  .ug-quick-tags::-webkit-scrollbar {
    display: none;
  }

  .ug-quick-tag {
    flex: 0 0 auto;
    height: 30px;
    padding: 0 10px;
  }

  .ug-table-card :deep(.ug-mobile-row-card) {
    position: relative;
    overflow: hidden;
    border-color: #e2e8f0;
    border-radius: 8px;
    background: #fff;
    padding: 12px;
    box-shadow: none;
  }

  .ug-table-card :deep(.ug-mobile-row-card::before) {
    position: absolute;
    inset: 0 auto 0 0;
    width: 3px;
    background: #00B42A;
    content: "";
  }

  .ug-table-card :deep(.ug-mobile-row-card.ug-row-unmatched) {
    border-color: #fed7aa;
    background: #fffaf5;
  }

  .ug-table-card :deep(.ug-mobile-row-card.ug-row-unmatched::before) {
    background: #FF7D00;
  }

  .ug-table-card :deep(.ug-mobile-row-card.ug-row-risk) {
    border-color: #fecaca;
    background: #fffafa;
  }

  .ug-table-card :deep(.ug-mobile-row-card.ug-row-risk::before) {
    background: #F53F3F;
  }

  .ug-table-card :deep(.ug-mobile-row-card > .space-y-3 > .flex) {
    gap: 10px;
  }

  .ug-table-card :deep(.ug-mobile-row-card > .space-y-3 > .flex:nth-child(-n + 3)) {
    border-bottom: 1px solid #f1f5f9;
    padding-bottom: 8px;
  }

  .ug-table-card :deep(.ug-mobile-row-card:not(.ug-mobile-row-expanded) > .space-y-3 > .flex:nth-child(3)),
  .ug-table-card :deep(.ug-mobile-row-card:not(.ug-mobile-row-expanded) > .space-y-3 > .flex:nth-child(7)),
  .ug-table-card :deep(.ug-mobile-row-card:not(.ug-mobile-row-expanded) > .space-y-3 > .flex:nth-child(8)) {
    display: none;
  }

  .ug-table-card :deep(.ug-mobile-row-card > .space-y-3 > .flex > span) {
    color: #64748b;
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0;
    text-transform: none;
  }

  .ug-table-card :deep(.ug-mobile-row-card > .space-y-3 > .flex > div) {
    color: #0f172a;
    font-size: 13px;
  }

  .ug-table-card :deep(.ug-mobile-row-card) .ug-action-stack {
    flex-direction: row;
    align-items: center;
    justify-content: flex-end;
    gap: 6px;
  }

  .ug-table-card :deep(.ug-mobile-row-card) .ug-btn-small {
    height: 30px;
    padding: 0 9px;
    font-size: 12px;
  }

  .ug-table-card :deep(.ug-mobile-row-card) .ug-btn-text {
    min-height: 30px;
    border: 1px solid #bbf7d0;
    border-radius: 7px;
    background: #ecfdf5;
    padding: 0 9px;
    color: #047857;
    font-size: 12px;
    line-height: 28px;
  }

  .ug-table-card :deep(.ug-mobile-row-card) .ug-mobile-detail-toggle {
    display: inline-flex;
  }

  .ug-records-table {
    display: none;
  }

  .ug-rate-fix-log-cards {
    display: block;
    padding: 10px;
  }
}

@media (max-width: 520px) {
  .ug-stats-row {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 5px;
  }

  .ug-stat-card {
    min-height: 48px;
    gap: 4px;
    padding: 7px 5px;
  }

  .ug-stat-bar {
    width: 3px;
    height: 24px;
  }

  .ug-stat-label {
    font-size: 10px;
    line-height: 1.1;
  }

  .ug-stat-value {
    margin-top: 2px;
    font-size: 15px;
    line-height: 1;
  }

  .ug-filter-card {
    padding: 10px;
  }

  .ug-filter-top {
    gap: 8px;
  }

  .ug-filter-left.is-open {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }

  .ug-filter-select {
    min-width: 0;
  }

  .ug-filter-right {
    gap: 8px;
  }

  .ug-filter-right .ug-btn {
    height: 32px;
    padding: 0 10px;
    font-size: 12px;
  }

  .ug-auto-controls {
    @apply items-center;
    gap: 8px;
  }

  .ug-auto-controls .ug-btn {
    width: auto;
    height: 30px;
    padding: 0 10px;
    font-size: 12px;
  }

  .ug-auto-interval {
    width: auto;
    flex: 1 1 170px;
    min-width: 0;
  }

  .ug-auto-input {
    width: 78px;
  }

  .ug-account-list,
  .ug-account-status-list,
  .ug-match-desc,
  .ug-group-title,
  .ug-action-stack {
    justify-content: flex-start;
  }

  .ug-action-stack {
    align-items: flex-start;
  }

  .ug-bound-accounts-dialog,
  .ug-rate-fix-logs-dialog {
    @apply px-2 py-4;
  }

  .ug-bound-accounts-modal,
  .ug-rate-fix-logs-modal {
    width: 100%;
    max-height: calc(100vh - 32px);
  }

  .ug-bound-accounts-list {
    padding: 8px;
  }

  .ug-bound-account-row {
    gap: 10px;
    padding: 10px 6px;
  }

  .ug-rate-fix-logs-actions {
    padding: 10px 12px;
  }
}

@media (max-width: 380px) {
  .ug-filter-left.is-open {
    grid-template-columns: 1fr;
  }

  .ug-filter-right .ug-btn {
    flex: 1 1 calc(50% - 4px);
  }

  .ug-auto-interval {
    width: 100%;
    flex-basis: 100%;
  }

  .ug-auto-input {
    flex: 1 1 auto;
  }
}
</style>
