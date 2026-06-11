<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
            <div class="rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-600">
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.defaultProvider') }}</div>
              <div class="mt-0.5 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span>{{ result?.default_provider?.name || '-' }}</span>
                <code v-if="result?.default_provider?.slug" class="text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ result.default_provider.slug }}
                </code>
              </div>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.upstreamGroups.upstreamGroups') }}</span>
              <strong>{{ summary.upstreamGroups }}</strong>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.upstreamGroups.matchedGroups') }}</span>
              <strong>{{ summary.matchedGroups }}</strong>
            </div>
            <div class="summary-pill" :class="summary.rateRisks > 0 ? 'summary-pill-warning' : ''">
              <span>{{ t('admin.upstreamGroups.rateRisks') }}</span>
              <strong>{{ summary.rateRisks }}</strong>
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading || applying"
              :title="t('common.refresh')"
              @click="reload"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              type="button"
              class="btn btn-primary"
              :disabled="loading || applying || summary.rateRisks === 0"
              @click="applyRateFixes"
            >
              <Icon name="sync" size="sm" class="mr-2" :class="applying ? 'animate-spin' : ''" />
              {{ t('admin.upstreamGroups.fixRates') }}
            </button>
          </div>
        </div>
        <div class="mt-3 flex flex-wrap items-center gap-3">
          <input
            v-model.trim="searchQuery"
            type="search"
            class="input w-full sm:w-72"
            :placeholder="t('admin.upstreamGroups.searchPlaceholder')"
          />
          <Select v-model="matchFilter" class="w-44" :options="matchFilterOptions" />
          <Select v-model="rateFilter" class="w-44" :options="rateFilterOptions" />
          <div class="summary-pill h-10">
            <span>{{ t('admin.upstreamGroups.filteredCount') }}</span>
            <strong>{{ filteredItems.length }}</strong>
          </div>
        </div>
        <div class="mt-3 flex flex-wrap items-end gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <label class="flex h-10 items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200">
            <input
              v-model="autoFixForm.enabled"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :disabled="savingRateFixConfig || loadingRateFixConfig"
            />
            {{ t('admin.upstreamGroups.autoFixEnabled') }}
          </label>
          <div>
            <label class="input-label" for="auto-fix-interval-seconds">{{ t('admin.upstreamGroups.autoFixIntervalSeconds') }}</label>
            <input
              id="auto-fix-interval-seconds"
              v-model.number="autoFixForm.interval_seconds"
              type="number"
              min="1"
              step="1"
              class="input mt-1 w-36"
              :disabled="savingRateFixConfig || loadingRateFixConfig"
            />
          </div>
          <div class="min-w-[13rem] text-xs text-gray-500 dark:text-gray-400">
            <div class="font-medium text-gray-600 dark:text-gray-300">{{ t('admin.upstreamGroups.autoFixLastRun') }}</div>
            <div class="mt-1">{{ autoFixLastRunText }}</div>
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-sm h-10"
            :disabled="savingRateFixConfig || loadingRateFixConfig"
            @click="saveRateFixConfig"
          >
            {{ savingRateFixConfig ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>

      <template #table>
        <div v-if="warnings.length" class="mb-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200">
          <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
        </div>

        <DataTable :columns="columns" :data="filteredItems" :loading="loading">
          <template #cell-upstream_group_name="{ row }">
            <div class="flex min-w-[12rem] flex-col gap-1">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.upstream_group_name }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.upstreamGroups.keyCount', { count: row.upstream_key_count }) }}
              </span>
            </div>
          </template>

          <template #cell-upstream_rate="{ value }">
            <span class="font-mono text-sm text-gray-900 dark:text-white">{{ formatRate(value) }}</span>
          </template>

          <template #cell-local_group_name="{ row }">
            <div class="flex min-w-[16rem] flex-col gap-2">
              <Select
                :model-value="row.local_group_id ?? null"
                :options="localGroupOptions"
                :placeholder="t('admin.upstreamGroups.selectLocalGroup')"
                searchable
                clearable
                :disabled="savingMappingKey === row.upstream_group_key"
                @change="value => saveGroupMapping(row, value)"
              />
              <div v-if="row.matched" class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <code>#{{ row.local_group_id }}</code>
                <span :class="['badge', row.match_source === 'manual' ? 'badge-primary' : 'badge-gray']">
                  {{ matchSourceLabel(row) }}
                </span>
              </div>
              <div v-else class="text-xs text-gray-400">
                {{ t('admin.upstreamGroups.notMatched') }}
              </div>
            </div>
          </template>

          <template #cell-local_rate="{ row }">
            <span v-if="row.local_rate !== undefined" class="font-mono text-sm text-gray-900 dark:text-white">
              {{ formatRate(row.local_rate) }}
            </span>
            <span v-else class="text-sm text-gray-400">-</span>
          </template>

          <template #cell-status="{ row }">
            <span :class="['badge', statusClass(row)]">{{ statusLabel(row) }}</span>
          </template>

          <template #cell-action="{ row }">
            <button
              v-if="!row.matched"
              type="button"
              class="btn btn-primary btn-sm whitespace-nowrap"
              :disabled="syncingGroupKey === row.upstream_group_key"
              @click="openSyncDialog(row)"
            >
              <Icon name="sync" size="sm" class="mr-1" :class="syncingGroupKey === row.upstream_group_key ? 'animate-spin' : ''" />
              {{ t('admin.upstreamGroups.syncLocalGroup') }}
            </button>
            <span v-else class="text-xs font-medium text-gray-400 dark:text-gray-500">
              {{ t('admin.upstreamGroups.alreadyMatched') }}
            </span>
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

        <div class="mt-6 rounded-lg border border-gray-200 dark:border-dark-600">
          <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-dark-600">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreamGroups.changeRecords') }}</h3>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreamGroups.latestRecords') }}</span>
          </div>
          <div class="max-h-72 overflow-auto">
            <table class="w-full min-w-[760px] divide-y divide-gray-100 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamGroups.localGroup') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamGroups.upstreamGroup') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamGroups.oldRate') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamGroups.newRate') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamGroups.changedAt') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="record in records" :key="`${record.group_id}-${record.changed_at}`">
                  <td class="px-4 py-2">{{ record.group_name }}</td>
                  <td class="px-4 py-2">{{ record.upstream_group_name }}</td>
                  <td class="px-4 py-2 font-mono">{{ formatRate(record.old_rate) }}</td>
                  <td class="px-4 py-2 font-mono">{{ formatRate(record.new_rate) }}</td>
                  <td class="px-4 py-2">{{ formatDateTime(record.changed_at) }}</td>
                </tr>
                <tr v-if="!records.length">
                  <td colspan="5" class="px-4 py-8 text-center text-gray-400">{{ t('admin.upstreamGroups.noRecords') }}</td>
                </tr>
              </tbody>
            </table>
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
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import type {
  UpstreamGroupAutoRateFixConfig,
  UpstreamGroupCompareResult,
  UpstreamGroupComparison,
  UpstreamGroupRateFixRecord
} from '@/api/admin/upstreamManagement'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const result = ref<UpstreamGroupCompareResult | null>(null)
const loading = ref(false)
const applying = ref(false)
const loadingRateFixConfig = ref(false)
const savingRateFixConfig = ref(false)
const savingMappingKey = ref<string | null>(null)
const syncingGroupKey = ref<string | null>(null)
const loadError = ref('')
const localGroups = ref<AdminGroup[]>([])
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

const columns = computed<Column[]>(() => [
  { key: 'upstream_group_name', label: t('admin.upstreamGroups.columns.upstreamGroup') },
  { key: 'upstream_rate', label: t('admin.upstreamGroups.columns.upstreamRate') },
  { key: 'local_group_name', label: t('admin.upstreamGroups.columns.localGroup') },
  { key: 'local_rate', label: t('admin.upstreamGroups.columns.localRate') },
  { key: 'status', label: t('admin.upstreamGroups.columns.status') },
  { key: 'action', label: t('admin.upstreamGroups.columns.action') },
])

const items = computed<UpstreamGroupComparison[]>(() => result.value?.items || [])
const warnings = computed(() => result.value?.warnings || [])
const records = computed<UpstreamGroupRateFixRecord[]>(() => result.value?.records || [])
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
  return items.value.filter((item) => {
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
})

const localGroupOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('admin.upstreamGroups.clearMapping') },
  ...localGroups.value.map(group => ({
    value: group.id,
    label: `${group.name} (${formatRate(group.rate_multiplier)})`,
    description: `#${group.id}`
  }))
])

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
  loading.value = true
  loadingRateFixConfig.value = true
  loadError.value = ''
  try {
    const [groupsResult, groups, config] = await Promise.all([
      adminAPI.upstreamManagement.getGroups(),
      adminAPI.groups.getAll(),
      adminAPI.upstreamManagement.getRateFixConfig()
    ])
    result.value = groupsResult
    localGroups.value = groups
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

async function saveGroupMapping(row: UpstreamGroupComparison, value: string | number | boolean | null) {
  const localGroupId = typeof value === 'number' ? value : null
  savingMappingKey.value = row.upstream_group_key
  try {
    result.value = await adminAPI.upstreamManagement.saveGroupMapping(row.upstream_group_name, localGroupId)
    appStore.showSuccess(t('admin.upstreamGroups.mappingSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.mappingSaveFailed')))
  } finally {
    savingMappingKey.value = null
  }
}

function applyRateFixConfig(config: UpstreamGroupAutoRateFixConfig) {
  rateFixConfig.value = config
  autoFixForm.value = {
    enabled: Boolean(config.enabled),
    interval_seconds: normalizePositiveInteger(config.interval_seconds, 3600),
  }
}

function openSyncDialog(row: UpstreamGroupComparison) {
  syncDialogItem.value = row
  syncRateMultiplier.value = normalizePositiveRate(row.upstream_rate, 1)
}

function closeSyncDialog() {
  if (syncingGroupKey.value) return
  syncDialogItem.value = null
}

async function syncLocalGroup() {
  if (!syncDialogItem.value) return
  const rate = Number(syncRateMultiplier.value)
  if (!Number.isFinite(rate) || rate <= 0) {
    appStore.showError(t('admin.upstreamGroups.invalidRate'))
    return
  }
  const item = syncDialogItem.value
  syncingGroupKey.value = item.upstream_group_key
  try {
    result.value = await adminAPI.upstreamManagement.createLocalGroupFromUpstream({
      upstream_group_name: item.upstream_group_name,
      rate_multiplier: rate,
    })
    localGroups.value = await adminAPI.groups.getAll()
    syncDialogItem.value = null
    appStore.showSuccess(t('admin.upstreamGroups.syncSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamGroups.syncFailed')))
  } finally {
    syncingGroupKey.value = null
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

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
}

function matchSourceLabel(row: UpstreamGroupComparison) {
  if (row.match_source === 'manual') return t('admin.upstreamGroups.manualMatched')
  return t('admin.upstreamGroups.nameMatched')
}

function statusClass(row: UpstreamGroupComparison) {
  if (!row.matched) return 'badge-gray'
  if (row.needs_rate_increase) return 'badge-warning'
  return 'badge-success'
}

function statusLabel(row: UpstreamGroupComparison) {
  if (!row.matched) return t('admin.upstreamGroups.notMatched')
  if (row.needs_rate_increase) return t('admin.upstreamGroups.needsIncrease')
  return t('admin.upstreamGroups.inSync')
}

onMounted(reload)
</script>

<style scoped>
.summary-pill {
  @apply flex h-11 items-center gap-3 rounded-lg border border-gray-200 px-3 text-sm text-gray-600 dark:border-dark-600 dark:text-gray-300;
}

.summary-pill strong {
  @apply font-mono text-base text-gray-900 dark:text-white;
}

.summary-pill-warning {
  @apply border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200;
}
</style>
