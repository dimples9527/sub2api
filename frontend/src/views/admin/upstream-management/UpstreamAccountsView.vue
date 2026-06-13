<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="accounts-toolbar">
          <div class="provider-panel">
            <div class="min-w-0">
              <div class="meta-label">{{ t('admin.upstreamAccounts.syncProviders') }}</div>
              <div class="mt-1 flex min-w-0 items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span class="truncate">{{ syncProviderLabel }}</span>
                <code v-if="syncProviderCode" class="text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ syncProviderCode }}
                </code>
              </div>
            </div>
            <div class="provider-count">{{ syncProviders.length }}</div>
          </div>

          <div class="stats-strip">
            <div class="stat-tile">
              <span>{{ t('admin.upstreamAccounts.upstreamKeys') }}</span>
              <strong>{{ summary.upstream_key_count }}</strong>
            </div>
            <div class="stat-tile stat-tile-create">
              <span>{{ t('admin.upstreamAccounts.toCreate') }}</span>
              <strong>{{ summary.create_count }}</strong>
            </div>
            <div class="stat-tile stat-tile-update">
              <span>{{ t('admin.upstreamAccounts.toUpdate') }}</span>
              <strong>{{ summary.update_count }}</strong>
            </div>
            <div class="stat-tile" :class="summary.rate_violation_count > 0 ? 'stat-tile-warning' : ''">
              <span>{{ t('admin.upstreamAccounts.rateRisks') }}</span>
              <strong>{{ summary.rate_violation_count }}</strong>
            </div>
          </div>

          <div class="accounts-actions">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading || syncing"
              :title="t('common.refresh')"
              @click="reload"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              type="button"
              class="btn btn-primary"
              :disabled="loading || syncing || !canSync"
              @click="runSync"
            >
              <Icon name="sync" size="sm" class="mr-2" :class="syncing ? 'animate-spin' : ''" />
              {{ t('admin.upstreamAccounts.syncNow') }}
            </button>
          </div>
        </div>

        <div class="filter-row">
          <div class="relative min-w-0">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model.trim="searchQuery"
              type="search"
              class="input filter-search pl-10"
              :placeholder="t('admin.upstreamAccounts.searchPlaceholder')"
            />
          </div>
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
            v-model="actionFilter"
            class="filter-select"
            :options="actionOptions"
          />
          <div class="filtered-count">
            <span>{{ t('admin.upstreamAccounts.filteredCount') }}</span>
            <strong>{{ filteredItems.length }}</strong>
          </div>
        </div>

        <div class="rate-guard-panel">
          <div class="min-w-0">
            <div class="meta-label">{{ t('admin.upstreamAccounts.rateGuardTitle') }}</div>
            <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">
              {{ t('admin.upstreamAccounts.rateGuardDescription') }}
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-xs">
              <span :class="['badge', rateGuardForm.enabled ? 'badge-success' : 'badge-gray']">
                {{ rateGuardForm.enabled ? t('admin.upstreamAccounts.rateGuardEnabled') : t('admin.upstreamAccounts.rateGuardDisabled') }}
              </span>
              <span class="text-gray-500 dark:text-gray-400">
                {{ t('admin.upstreamAccounts.rateGuardLastRun') }}:
                {{ rateGuardLastRunText }}
              </span>
              <span
                v-if="rateGuardConfig?.last_run_status"
                :class="['record-status', rateGuardConfig.last_run_status === 'failed' ? 'record-status-error' : 'record-status-success']"
              >
                {{ rateGuardConfig.last_run_status === 'failed' ? t('admin.upstreamAccounts.rateGuardStatusFailed') : t('admin.upstreamAccounts.rateGuardStatusSuccess') }}
              </span>
              <span v-if="rateGuardConfig?.last_run_message" class="text-red-600 dark:text-red-300">
                {{ rateGuardConfig.last_run_message }}
              </span>
            </div>
          </div>
          <label class="guard-toggle">
            <input v-model="rateGuardForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('admin.upstreamAccounts.rateGuardAutoRun') }}</span>
          </label>
          <label class="guard-interval">
            <span>{{ t('admin.upstreamAccounts.rateGuardIntervalSeconds') }}</span>
            <input
              v-model.number="rateGuardForm.interval_seconds"
              type="number"
              min="1"
              class="input h-9 w-28"
            />
          </label>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loadingRateGuardConfig || savingRateGuardConfig"
            @click="saveRateGuardConfig"
          >
            <Icon name="cog" size="sm" class="mr-2" :class="savingRateGuardConfig ? 'animate-spin' : ''" />
            {{ t('common.save') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="loadingRateGuardConfig || savingRateGuardConfig || runningRateGuardNow"
            @click="runRateGuardNow"
          >
            <Icon name="play" size="sm" class="mr-2" :class="runningRateGuardNow ? 'animate-pulse' : ''" />
            {{ t('admin.upstreamAccounts.rateGuardRunNow') }}
          </button>
        </div>
      </template>

      <template #table>
        <div class="accounts-table-content">
          <div v-if="warnings.length" class="warning-banner">
            <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
          </div>

          <div class="accounts-table-primary">
            <DataTable :columns="columns" :data="filteredItems" :loading="loading">
              <template #cell-source="{ row }">
                <div class="table-main-cell min-w-[11rem]">
                  <span class="font-medium text-gray-900 dark:text-white">{{ row.provider_name || row.provider_slug }}</span>
                  <code class="text-xs text-gray-500 dark:text-gray-400">{{ row.provider_slug }}</code>
                </div>
              </template>

              <template #cell-upstream_key_name="{ row }">
                <div class="table-main-cell min-w-[14rem]">
                  <span class="font-medium text-gray-900 dark:text-white">{{ row.upstream_key_name }}</span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">{{ row.upstream_group_name }}</span>
                </div>
              </template>

              <template #cell-local_account_name="{ row }">
                <div class="table-main-cell min-w-[14rem]">
                  <span class="font-medium text-gray-900 dark:text-white">{{ row.local_account_name || '-' }}</span>
                  <span v-if="row.matched_account_id" class="text-xs text-gray-500 dark:text-gray-400">
                    #{{ row.matched_account_id }} {{ row.matched_account_name }}
                  </span>
                  <div v-else-if="row.conflict_accounts?.length" class="tag-list max-w-[24rem]">
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
                  <span v-else-if="row.conflict_account_ids?.length" class="text-xs text-amber-600 dark:text-amber-300">
                    {{ t('admin.upstreamAccounts.conflictIds', { ids: row.conflict_account_ids.join(', ') }) }}
                  </span>
                </div>
              </template>

              <template #cell-local_group_name="{ row }">
                <div class="table-main-cell min-w-[16rem]">
                  <div v-if="row.bound_groups?.length" class="tag-list max-w-[22rem]">
                    <span
                      v-for="group in row.bound_groups"
                      :key="`${row.provider_slug}-${row.upstream_key_name}-${group.id}`"
                      class="group-chip"
                      :class="group.rate_violation ? 'group-chip-warning' : ''"
                      :title="`${group.name} ${formatRate(group.rate_multiplier)}`"
                    >
                      {{ group.name }}
                      <span class="font-mono">{{ formatRate(group.rate_multiplier) }}</span>
                    </span>
                  </div>
                  <template v-else>
                    <span>{{ row.local_group_name || '-' }}</span>
                    <span v-if="row.local_rate_multiplier !== undefined" class="text-xs font-mono text-gray-500 dark:text-gray-400">
                      {{ formatRate(row.local_rate_multiplier) }}
                    </span>
                  </template>
                </div>
              </template>

              <template #cell-upstream_rate_multiplier="{ value }">
                <span class="rate-value">{{ formatRate(value) }}</span>
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

          <div class="records-panel">
            <div class="records-header">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreamAccounts.syncLogs') }}</h3>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.syncLogsDescription') }}</span>
              </div>
              <div class="records-total">{{ syncLogEntries.length }}</div>
            </div>
            <div class="max-h-80 overflow-auto">
              <table class="records-table min-w-[1080px]">
                <thead class="bg-gray-50 dark:bg-dark-800">
                  <tr>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logTime') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logTriggerSource') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logAccount') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logUpstream') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logRateCompare') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logUnboundGroups') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logRemainingGroups') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr v-for="entry in syncLogEntries" :key="entry.key" class="records-row">
                    <td class="px-4 py-3 text-gray-600 dark:text-gray-300">{{ formatDateTime(entry.created_at) }}</td>
                    <td class="px-4 py-3">{{ upstreamAccountSyncTriggerSourceLabel(entry.trigger_source) }}</td>
                    <td class="px-4 py-3">
                      <div class="table-main-cell min-w-[12rem]">
                        <span class="font-medium text-gray-900 dark:text-white">{{ entry.matched_local_account_name }}</span>
                        <code class="text-xs text-gray-500 dark:text-gray-400">#{{ entry.matched_local_account_id }}</code>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="table-main-cell min-w-[14rem]">
                        <span class="font-medium text-gray-900 dark:text-white">{{ entry.upstream_key_name }}</span>
                        <span class="text-xs text-gray-500 dark:text-gray-400">{{ entry.provider_name || entry.provider_slug }} / {{ entry.upstream_group_name }}</span>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="rate-compare">
                        <span>{{ formatRate(entry.upstream_rate_multiplier) }}</span>
                        <span class="text-gray-400">/</span>
                        <span>{{ formatRate(entry.local_min_rate_multiplier) }}</span>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="tag-list">
                        <span v-for="group in entry.unbound_group_names" :key="`${entry.key}-${group}`" class="log-chip log-chip-warning">{{ group }}</span>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="tag-list">
                        <span v-if="!entry.remaining_group_ids.length" class="text-xs text-gray-400">-</span>
                        <code v-for="groupID in entry.remaining_group_ids" :key="`${entry.key}-${groupID}`" class="log-chip">#{{ groupID }}</code>
                      </div>
                    </td>
                  </tr>
                  <tr v-if="!syncLogEntries.length">
                    <td colspan="7" class="px-4 py-8 text-center text-gray-400">{{ t('admin.upstreamAccounts.noSyncLogs') }}</td>
                  </tr>
                </tbody>
              </table>
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
  UpstreamAccountRateGuardConfig,
  UpstreamAccountSyncConflictAccount,
  UpstreamAccountSyncItem,
  UpstreamAccountSyncRecord,
  UpstreamAccountSyncResult,
  UpstreamAccountSyncUnbindDetail
} from '@/api/admin/upstreamAccountSync'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const result = ref<UpstreamAccountSyncResult | null>(null)
const loading = ref(false)
const syncing = ref(false)
const loadingRateGuardConfig = ref(false)
const savingRateGuardConfig = ref(false)
const runningRateGuardNow = ref(false)
const loadError = ref('')
const searchQuery = ref('')
const providerFilter = ref('')
const sourceFilter = ref('')
const actionFilter = ref('')
const rateGuardConfig = ref<UpstreamAccountRateGuardConfig | null>(null)
const rateGuardForm = ref({
  enabled: false,
  interval_seconds: 3600
})

type UpstreamAccountSyncLogEntry = UpstreamAccountSyncUnbindDetail & {
  created_at: string
  key: string
}

const columns = computed<Column[]>(() => [
  { key: 'source', label: t('admin.upstreamAccounts.columns.source') },
  { key: 'upstream_key_name', label: t('admin.upstreamAccounts.columns.upstreamKey') },
  { key: 'upstream_rate_multiplier', label: t('admin.upstreamAccounts.columns.upstreamRate') },
  { key: 'local_account_name', label: t('admin.upstreamAccounts.columns.localAccount') },
  { key: 'local_group_name', label: t('admin.upstreamAccounts.columns.boundGroups') },
])

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
const syncProviders = computed(() => result.value?.providers || [])
const items = computed<UpstreamAccountSyncItem[]>(() => result.value?.items || [])
const warnings = computed(() => result.value?.warnings || [])
const records = computed<UpstreamAccountSyncRecord[]>(() => result.value?.records || [])
const syncLogEntries = computed<UpstreamAccountSyncLogEntry[]>(() => {
  const entries: UpstreamAccountSyncLogEntry[] = []
  for (const record of records.value) {
    for (const detail of record.unbind_details || []) {
      entries.push({
        ...detail,
        created_at: record.created_at,
        key: `${record.created_at}-${detail.matched_local_account_id}-${detail.upstream_key_name}-${detail.unbound_group_ids.join('_')}`
      })
    }
  }
  return entries
})
const canSync = computed(() => summary.value.create_count > 0 || summary.value.update_count > 0 || summary.value.rate_violation_count > 0)
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
const actionOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamAccounts.allActions') },
  { value: 'create', label: t('admin.upstreamAccounts.actions.create') },
  { value: 'update', label: t('admin.upstreamAccounts.actions.update') },
  { value: 'noop', label: t('admin.upstreamAccounts.actions.noop') },
  { value: 'skip', label: t('admin.upstreamAccounts.actions.skip') },
  { value: 'conflict', label: t('admin.upstreamAccounts.actions.conflict') },
  { value: 'rate_risk', label: t('admin.upstreamAccounts.rateRisks') }
])
const filteredItems = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return items.value.filter((item) => {
    if (providerFilter.value && item.provider_slug !== providerFilter.value) return false
    if (sourceFilter.value === 'synced' && !item.matched_account_id) return false
    if (sourceFilter.value === 'unsynced' && item.matched_account_id) return false
    if (actionFilter.value === 'rate_risk') {
      if (!item.rate_violation) return false
    } else if (actionFilter.value && item.action !== actionFilter.value) {
      return false
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
    const [preview, config] = await Promise.all([
      adminAPI.upstreamAccountSync.getPreview(),
      adminAPI.upstreamAccountSync.getRateGuardConfig()
    ])
    result.value = preview
    applyRateGuardConfig(config)
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

async function runSync() {
  const confirmed = window.confirm(
    t('admin.upstreamAccounts.syncConfirm', {
      create: summary.value.create_count,
      update: summary.value.update_count,
      unbind: summary.value.unbound_group_count
    })
  )
  if (!confirmed) {
    return
  }

  syncing.value = true
  try {
    result.value = await adminAPI.upstreamAccountSync.runSync({
      create_missing: true,
      update_existing: true,
      apply_rate_guard: true
    })
    appStore.showSuccess(t('admin.upstreamAccounts.syncSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.syncFailed')))
  } finally {
    syncing.value = false
  }
}

function applyRateGuardConfig(config: UpstreamAccountRateGuardConfig) {
  rateGuardConfig.value = config
  rateGuardForm.value = {
    enabled: Boolean(config.enabled),
    interval_seconds: Number(config.interval_seconds) > 0 ? Number(config.interval_seconds) : 3600
  }
}

async function saveRateGuardConfig() {
  if (!Number.isInteger(rateGuardForm.value.interval_seconds) || rateGuardForm.value.interval_seconds <= 0) {
    appStore.showError(t('admin.upstreamAccounts.invalidRateGuardInterval'))
    return
  }
  savingRateGuardConfig.value = true
  try {
    const base = rateGuardConfig.value || {
      enabled: false,
      interval_seconds: 3600
    }
    const config = await adminAPI.upstreamAccountSync.updateRateGuardConfig({
      ...base,
      enabled: rateGuardForm.value.enabled,
      interval_seconds: rateGuardForm.value.interval_seconds
    })
    applyRateGuardConfig(config)
    appStore.showSuccess(t('admin.upstreamAccounts.rateGuardSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rateGuardSaveFailed')))
  } finally {
    savingRateGuardConfig.value = false
  }
}

async function runRateGuardNow() {
  runningRateGuardNow.value = true
  try {
    const config = await adminAPI.upstreamAccountSync.runRateGuardNow()
    applyRateGuardConfig(config)
    const preview = await adminAPI.upstreamAccountSync.getPreview()
    result.value = preview
    appStore.showSuccess(t('admin.upstreamAccounts.rateGuardRunSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rateGuardRunFailed')))
  } finally {
    runningRateGuardNow.value = false
  }
}

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
}

function conflictAccountRates(account: UpstreamAccountSyncConflictAccount) {
  return (account.bound_groups || [])
    .map(group => formatRate(group.rate_multiplier))
    .join(' / ')
}

function conflictAccountTitle(account: UpstreamAccountSyncConflictAccount) {
  const groups = (account.bound_groups || [])
    .map(group => `${group.name} ${formatRate(group.rate_multiplier)}`)
    .join(', ')
  return groups ? `${account.name}: ${groups}` : account.name
}

function upstreamAccountSyncTriggerSourceLabel(triggerSource: string | undefined) {
  if (triggerSource === 'scheduled_rate_guard') return t('admin.upstreamAccounts.triggerScheduledRateGuard')
  if (triggerSource === 'manual_rate_guard') return t('admin.upstreamAccounts.triggerManualRateGuard')
  return t('admin.upstreamAccounts.triggerManualSync')
}

onMounted(reload)
</script>

<style scoped>
.accounts-toolbar {
  @apply grid gap-3 xl:grid-cols-[minmax(14rem,18rem)_1fr_auto];
}

.provider-panel {
  @apply flex min-h-16 items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800/40;
}

.meta-label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.provider-count {
  @apply flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-gray-100 font-mono text-sm font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.stats-strip {
  @apply grid grid-cols-2 gap-2 sm:grid-cols-4;
}

.stat-tile {
  @apply flex min-h-16 flex-col justify-center rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-600 dark:border-dark-600 dark:bg-dark-800/40 dark:text-gray-300;
}

.stat-tile span {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.stat-tile strong {
  @apply mt-1 font-mono text-xl text-gray-950 dark:text-white;
}

.stat-tile-create {
  @apply border-sky-200 bg-sky-50/60 dark:border-sky-800/50 dark:bg-sky-950/20;
}

.stat-tile-update {
  @apply border-emerald-200 bg-emerald-50/60 dark:border-emerald-800/50 dark:bg-emerald-950/20;
}

.stat-tile-warning {
  @apply border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200;
}

.accounts-actions {
  @apply flex flex-wrap items-center justify-end gap-2 xl:min-h-16;
}

.filter-row {
  @apply mt-3 grid gap-3 md:grid-cols-[minmax(14rem,1fr)_12rem_11rem_11rem_auto];
}

.filter-search {
  @apply w-full;
}

.filter-select {
  @apply w-full;
}

.filtered-count {
  @apply flex h-10 items-center justify-between gap-3 rounded-lg border border-gray-200 px-3 text-sm text-gray-600 dark:border-dark-600 dark:text-gray-300;
}

.filtered-count strong {
  @apply font-mono text-base text-gray-900 dark:text-white;
}

.rate-guard-panel {
  @apply mt-3 grid items-center gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800/40 lg:grid-cols-[minmax(16rem,1fr)_auto_auto_auto_auto];
}

.guard-toggle {
  @apply inline-flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200;
}

.guard-interval {
  @apply flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300;
}

.accounts-table-content {
  @apply flex h-full min-h-0 flex-col overflow-y-auto;
}

.warning-banner {
  @apply mb-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200;
}

.accounts-table-primary {
  @apply flex flex-none flex-col overflow-hidden rounded-lg border border-gray-100 dark:border-dark-700;
  height: clamp(28rem, 52vh, 42rem);
  min-height: 28rem;
}

.accounts-table-primary :deep(.table-wrapper) {
  @apply min-h-0;
}

.accounts-table-primary :deep(tbody tr) {
  @apply transition-colors;
}

.records-panel {
  @apply mt-4 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800/30;
}

.records-header {
  @apply flex items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-600;
}

.records-total {
  @apply flex h-8 min-w-8 items-center justify-center rounded-md bg-gray-100 px-2 font-mono text-sm font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.records-row {
  @apply align-top;
}

.records-table {
  @apply w-full divide-y divide-gray-100 text-sm dark:divide-dark-700;
}

.records-table tbody {
  @apply divide-y divide-gray-100 dark:divide-dark-700;
}

.records-table tbody tr {
  @apply transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40;
}

.table-main-cell {
  @apply flex flex-col gap-1 leading-tight;
}

.rate-value {
  @apply inline-flex rounded-md bg-gray-100 px-2 py-1 font-mono text-sm font-semibold text-gray-900 dark:bg-dark-700 dark:text-white;
}

.record-status {
  @apply inline-flex rounded-md px-2 py-1 text-xs font-medium;
}

.record-status-success {
  @apply bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200;
}

.record-status-error {
  @apply bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-200;
}

.rate-compare {
  @apply inline-flex items-center gap-2 rounded-md bg-gray-100 px-2 py-1 font-mono text-sm font-semibold text-gray-900 dark:bg-dark-700 dark:text-white;
}

.tag-list {
  @apply flex max-w-[18rem] flex-wrap gap-1;
}

.group-chip {
  @apply inline-flex items-center gap-1 rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.group-chip-warning {
  @apply bg-amber-50 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-900/20 dark:text-amber-200 dark:ring-amber-700/40;
}

.log-chip {
  @apply inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.log-chip-warning {
  @apply bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-200;
}

@media (max-width: 1023px) {
  .accounts-table-content {
    @apply h-auto overflow-visible;
  }

  .accounts-table-primary {
    @apply h-auto min-h-0 overflow-visible;
  }
}
</style>
