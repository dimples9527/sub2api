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

        <div class="ug-provider-strip">
          <div class="ug-provider-meta">
            <span class="ug-meta-label">{{ t('admin.upstreamGroups.defaultProvider') }}</span>
            <span class="ug-provider-name">{{ result?.default_provider?.name || '-' }}</span>
            <code v-if="result?.default_provider?.slug" class="ug-provider-slug">{{ result.default_provider.slug }}</code>
          </div>
          <span class="ug-provider-count">{{ result?.default_provider?.slug ? 1 : 0 }}</span>
        </div>

        <div class="ug-filter-card">
          <div class="ug-filter-top">
            <div class="ug-search">
              <Icon name="search" size="sm" class="ug-search-icon" />
              <input
                v-model.trim="searchQuery"
                type="search"
                class="ug-input ug-search-input"
                :placeholder="t('admin.upstreamGroups.searchPlaceholder')"
              />
            </div>
            <div class="ug-filter-right">
              <Select v-model="matchFilter" class="ug-filter-select" :options="matchFilterOptions" />
              <Select v-model="rateFilter" class="ug-filter-select" :options="rateFilterOptions" />
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
                @click="applyRateFixes"
              >
                <Icon name="sync" size="sm" :class="applying ? 'animate-spin' : ''" />
                <span>{{ t('admin.upstreamGroups.fixRates') }}</span>
              </button>
            </div>
          </div>
          <div class="ug-auto-row">
            <span class="ug-auto-meta">{{ t('admin.upstreamGroups.autoFixLastRun') }}: {{ autoFixLastRunText }}</span>
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
                  :error="monitorError"
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
                    <span class="ug-tag ug-tag-warning">{{ t('admin.upstreamGroups.notMatched') }}</span>
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

              <template #cell-status="{ row }">
                <span :class="['ug-status', statusClass(row)]">{{ statusLabel(row) }}</span>
              </template>

              <template #cell-action="{ row }">
                <button
                  v-if="!row.matched"
                  type="button"
                  class="ug-btn ug-btn-primary ug-btn-small ug-btn-cell"
                  :disabled="syncingGroupKey === row.upstream_group_key"
                  @click="openSyncDialog(row)"
                >
                  <Icon name="sync" size="sm" :class="syncingGroupKey === row.upstream_group_key ? 'animate-spin' : ''" />
                  <span>{{ t('admin.upstreamGroups.syncLocalGroup') }}</span>
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

          <div class="ug-records-card">
            <div class="ug-records-header">
              <div class="ug-records-title-block">
                <span class="ug-records-title">{{ t('admin.upstreamGroups.changeRecords') }}</span>
                <span class="ug-records-sub">{{ t('admin.upstreamGroups.latestRecords') }}</span>
              </div>
              <div class="ug-records-actions">
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
                <span class="ug-records-count">{{ visibleRecords.length }}</span>
              </div>
            </div>
            <div class="ug-records-table-wrapper">
              <table class="ug-records-table">
                <thead>
                  <tr>
                    <th>{{ t('admin.upstreamGroups.localGroup') }}</th>
                    <th>{{ t('admin.upstreamGroups.upstreamGroup') }}</th>
                    <th>{{ t('admin.upstreamGroups.oldRate') }}</th>
                    <th>{{ t('admin.upstreamGroups.newRate') }}</th>
                    <th class="ug-records-time-th">{{ t('admin.upstreamGroups.changedAt') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="record in visibleRecords" :key="`${record.group_id}-${record.changed_at}`">
                    <td><span class="ug-tag ug-tag-default">{{ record.group_name }}</span></td>
                    <td><span class="ug-tag ug-tag-default">{{ record.upstream_group_name }}</span></td>
                    <td><span class="ug-old-rate">{{ formatRate(record.old_rate) }}</span></td>
                    <td><span class="ug-new-rate">{{ formatRate(record.new_rate) }}</span></td>
                    <td class="ug-records-time">{{ formatDateTime(record.changed_at) }}</td>
                  </tr>
                  <tr v-if="!visibleRecords.length">
                    <td colspan="5" class="ug-records-empty">{{ t('admin.upstreamGroups.noRecords') }}</td>
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
import { formatDateTime } from '@/utils/format'
import {
  buildUpstreamMonitorTrendIndex,
  normalizeUpstreamMonitorGroupKey,
  type UpstreamMonitorTrendRow
} from '@/utils/upstreamMonitorTrend'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import UpstreamGroupAvailabilityTrend from '@/components/admin/upstream/UpstreamGroupAvailabilityTrend.vue'

const { t } = useI18n()
const appStore = useAppStore()

const RECORDS_VISIBLE_LIMIT = 10

const result = ref<UpstreamGroupCompareResult | null>(null)
const loading = ref(false)
const applying = ref(false)
const loadingRateFixConfig = ref(false)
const savingRateFixConfig = ref(false)
const syncingGroupKey = ref<string | null>(null)
const savingLocalRateGroupId = ref<number | null>(null)
const monitorTrendIndex = ref<Map<string, UpstreamMonitorTrendRow>>(new Map())
const monitorLoading = ref(false)
const monitorError = ref('')
const loadError = ref('')
const rateFixConfig = ref<UpstreamGroupAutoRateFixConfig | null>(null)
const autoFixForm = ref({
  enabled: false,
  interval_seconds: 3600,
})
const searchQuery = ref('')
const matchFilter = ref('')
const rateFilter = ref('')
const syncDialogItem = ref<UpstreamGroupComparison | null>(null)
const syncRateMultiplier = ref(1)
const syncPlatform = ref('')
const localRateDialogItem = ref<UpstreamGroupComparison | null>(null)
const localRateInput = ref(1)
const recordsSortOrder = ref<'desc' | 'asc'>('desc')
let reloadRequestId = 0

const platformOptions = computed<SelectOption[]>(() => [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
])
const syncPlatformOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamGroups.selectPlatform') },
  ...platformOptions.value,
])

const columns = computed<Column[]>(() => [
  { key: 'upstream_group_name', label: t('admin.upstreamGroups.columns.upstreamGroup'), class: 'min-w-[12rem]', sortable: true },
  { key: 'upstream_rate', label: t('admin.upstreamGroups.columns.upstreamRate'), sortable: true },
  { key: 'monitor_trend', label: t('admin.upstreamGroups.columns.monitorTrend'), class: 'min-w-[10.5rem]' },
  { key: 'local_group_name', label: t('admin.upstreamGroups.columns.matchResult'), sortable: true },
  { key: 'local_rate', label: t('admin.upstreamGroups.columns.localRate'), sortable: true },
  { key: 'rate_delta', label: t('admin.upstreamGroups.columns.rateDelta'), sortable: true },
  { key: 'status', label: t('admin.upstreamGroups.columns.status'), sortable: true },
  { key: 'action', label: t('admin.upstreamGroups.columns.action') },
])

const items = computed<UpstreamGroupComparison[]>(() => result.value?.items || [])
const warnings = computed(() => result.value?.warnings || [])
const records = computed<UpstreamGroupRateFixRecord[]>(() => result.value?.records || [])
const visibleRecords = computed<UpstreamGroupRateFixRecord[]>(() => {
  const sorted = [...records.value].sort((a, b) => {
    const aTime = recordTimestamp(a.changed_at)
    const bTime = recordTimestamp(b.changed_at)
    return recordsSortOrder.value === 'desc' ? bTime - aTime : aTime - bTime
  })
  return sorted.slice(0, RECORDS_VISIBLE_LIMIT)
})
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
const filteredItems = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return items.value
    .filter((item) => {
      if (matchFilter.value === 'matched' && !item.matched) return false
      if (matchFilter.value === 'unmatched' && item.matched) return false
      if (rateFilter.value === 'risk' && !item.needs_rate_increase) return false
      if (rateFilter.value === 'ok' && item.needs_rate_increase) return false
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
  } catch (err) {
    const message = extractApiErrorMessage(err, t('admin.upstreamGroups.loadFailed'))
    loadError.value = message
    result.value = null
    appStore.showError(message)
  } finally {
    loading.value = false
    loadingRateFixConfig.value = false
  }
}

async function loadMonitorTrend(requestId: number) {
  monitorLoading.value = true
  monitorError.value = ''
  try {
    const payload = await adminAPI.groups.getUpstreamMonitorStatus({ period: '90m', board: 'hot' })
    if (requestId !== reloadRequestId) return
    monitorTrendIndex.value = buildUpstreamMonitorTrendIndex(payload)
  } catch (err) {
    if (requestId !== reloadRequestId) return
    monitorTrendIndex.value = new Map()
    monitorError.value = extractApiErrorMessage(err, t('admin.upstreamGroups.monitorTrendLoadFailed'))
  } finally {
    if (requestId === reloadRequestId) {
      monitorLoading.value = false
    }
  }
}

async function applyRateFixes() {
  applying.value = true
  try {
    result.value = await adminAPI.upstreamManagement.applyRateFixes()
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

function normalizePositiveRate(value: number | undefined, fallback: number) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : fallback
}

function normalizePositiveInteger(value: number | undefined, fallback: number) {
  const n = Number(value)
  return Number.isInteger(n) && n > 0 ? n : fallback
}

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
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
  return `${n > 0 ? '+' : ''}${n.toFixed(2)}x`
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

function rowClass(row: UpstreamGroupComparison) {
  if (!row.matched) return 'ug-row-unmatched'
  if (row.needs_rate_increase) return 'ug-row-risk'
  return 'ug-row-ok'
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

.ug-provider-strip {
  @apply mt-3 flex items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800/40;
}

.ug-provider-meta {
  @apply flex min-w-0 flex-wrap items-center gap-2;
}

.ug-meta-label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.ug-provider-name {
  @apply truncate text-sm font-semibold text-gray-900 dark:text-white;
}

.ug-provider-slug {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300;
}

.ug-provider-count {
  @apply flex h-8 min-w-8 shrink-0 items-center justify-center rounded-md bg-gray-100 px-2 font-mono text-sm font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.ug-filter-card {
  @apply mt-3 rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800/40;
}

.ug-filter-top {
  @apply flex flex-wrap items-center gap-3;
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
}

.ug-filter-select {
  @apply w-32;
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

.ug-btn-small {
  @apply h-8 px-3 text-xs;
}

.ug-btn-cell {
  @apply whitespace-nowrap;
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
  @apply min-w-0 truncate;
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
  @apply flex h-full min-h-0 flex-col overflow-y-auto;
}

.ug-warning-banner {
  background: #FFF7E8;
  border: 1px solid #FFE4B3;
  color: #B25A00;
  @apply mb-3 rounded-lg px-4 py-2 text-sm;
}

.ug-table-card {
  @apply flex flex-none flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800/30;
  height: clamp(28rem, 54vh, 44rem);
  min-height: 28rem;
}

.ug-table-card :deep(.table-wrapper) {
  @apply min-h-0;
}

.ug-table-card :deep(.table-wrapper) {
  border-radius: 0.5rem;
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
}

.ug-group-title {
  @apply flex min-w-0 flex-wrap items-center gap-2;
}

.ug-group-name {
  @apply truncate font-semibold text-gray-900 dark:text-white;
}

.ug-group-sub {
  @apply flex flex-wrap items-center gap-1 text-xs text-gray-500 dark:text-gray-400;
}

.ug-group-sub-sep {
  @apply text-gray-300 dark:text-gray-600;
}

.ug-group-sub-code {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300;
}

.ug-match-cell {
  @apply flex flex-col gap-1.5 leading-tight;
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

.ug-records-card {
  @apply mt-4 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800/30;
}

.ug-records-header {
  @apply flex items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700;
}

.ug-records-title-block {
  @apply flex min-w-0 flex-wrap items-center gap-2;
}

.ug-records-title {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
}

.ug-records-sub {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.ug-records-actions {
  @apply flex items-center gap-2;
}

.ug-records-sort-btn {
  @apply inline-flex h-7 items-center gap-1 rounded-md border border-gray-200 bg-white px-2 text-xs font-medium text-gray-600 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-900 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300;
}

.ug-records-count {
  @apply flex h-7 min-w-7 items-center justify-center rounded-md bg-gray-100 px-2 font-mono text-xs font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.ug-records-table-wrapper {
  @apply max-h-72 overflow-auto;
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

@media (max-width: 1023px) {
  .ug-content {
    @apply h-auto overflow-visible;
  }

  .ug-table-card {
    @apply h-auto min-h-0 overflow-visible;
  }
}
</style>
