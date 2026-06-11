<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
            <div class="rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-600">
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.syncProviders') }}</div>
              <div class="mt-0.5 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span>{{ syncProviderLabel }}</span>
                <code v-if="syncProviderCode" class="text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ syncProviderCode }}
                </code>
              </div>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.upstreamAccounts.upstreamKeys') }}</span>
              <strong>{{ summary.upstream_key_count }}</strong>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.upstreamAccounts.toCreate') }}</span>
              <strong>{{ summary.create_count }}</strong>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.upstreamAccounts.toUpdate') }}</span>
              <strong>{{ summary.update_count }}</strong>
            </div>
            <div class="summary-pill" :class="summary.rate_violation_count > 0 ? 'summary-pill-warning' : ''">
              <span>{{ t('admin.upstreamAccounts.rateRisks') }}</span>
              <strong>{{ summary.rate_violation_count }}</strong>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.upstreamAccounts.filteredCount') }}</span>
              <strong>{{ filteredItems.length }}</strong>
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2">
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
        <div class="mt-3 flex flex-wrap items-center gap-3">
          <input
            v-model.trim="searchQuery"
            type="search"
            class="input w-full sm:w-64"
            :placeholder="t('admin.upstreamAccounts.searchPlaceholder')"
          />
          <Select
            v-model="providerFilter"
            class="w-48"
            :options="providerOptions"
          />
          <Select
            v-model="sourceFilter"
            class="w-44"
            :options="sourceOptions"
          />
          <Select
            v-model="actionFilter"
            class="w-44"
            :options="actionOptions"
          />
        </div>
      </template>

      <template #table>
        <div v-if="warnings.length" class="mb-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200">
          <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
        </div>

        <DataTable :columns="columns" :data="filteredItems" :loading="loading">
          <template #cell-source="{ row }">
            <div class="flex min-w-[11rem] flex-col gap-1">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.provider_name || row.provider_slug }}</span>
              <code class="text-xs text-gray-500 dark:text-gray-400">{{ row.provider_slug }}</code>
            </div>
          </template>

          <template #cell-upstream_key_name="{ row }">
            <div class="flex min-w-[14rem] flex-col gap-1">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.upstream_key_name }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ row.upstream_group_name }}</span>
            </div>
          </template>

          <template #cell-local_account_name="{ row }">
            <div class="flex min-w-[14rem] flex-col gap-1">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.local_account_name || '-' }}</span>
              <span v-if="row.matched_account_id" class="text-xs text-gray-500 dark:text-gray-400">
                #{{ row.matched_account_id }} {{ row.matched_account_name }}
              </span>
              <span v-else-if="row.conflict_account_ids?.length" class="text-xs text-amber-600 dark:text-amber-300">
                {{ t('admin.upstreamAccounts.conflictIds', { ids: row.conflict_account_ids.join(', ') }) }}
              </span>
            </div>
          </template>

          <template #cell-local_group_name="{ row }">
            <div class="flex min-w-[12rem] flex-col gap-1">
              <span>{{ row.local_group_name || '-' }}</span>
              <span v-if="row.local_rate_multiplier !== undefined" class="text-xs font-mono text-gray-500 dark:text-gray-400">
                {{ formatRate(row.local_rate_multiplier) }}
              </span>
            </div>
          </template>

          <template #cell-upstream_rate_multiplier="{ value }">
            <span class="font-mono text-sm text-gray-900 dark:text-white">{{ formatRate(value) }}</span>
          </template>

          <template #cell-action="{ row }">
            <div class="flex min-w-[12rem] flex-col gap-1">
              <span :class="['badge', actionClass(row)]">{{ actionLabel(row) }}</span>
              <span v-if="row.skip_reason" class="text-xs text-gray-500 dark:text-gray-400">{{ row.skip_reason }}</span>
              <span v-if="row.unbound_group_names?.length" class="text-xs text-amber-600 dark:text-amber-300">
                {{ t('admin.upstreamAccounts.unbindGroups', { groups: row.unbound_group_names.join(', ') }) }}
              </span>
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

        <div class="mt-6 rounded-lg border border-gray-200 dark:border-dark-600">
          <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-dark-600">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreamAccounts.syncRecords') }}</h3>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.latestRecords') }}</span>
          </div>
          <div class="max-h-72 overflow-auto">
            <table class="w-full min-w-[860px] divide-y divide-gray-100 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.recordTime') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.provider') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.created') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.updated') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.unbound') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.status') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="record in records" :key="`${record.provider_slug}-${record.created_at}`">
                  <td class="px-4 py-2">{{ formatDateTime(record.created_at) }}</td>
                  <td class="px-4 py-2">{{ record.provider_name || record.provider_slug }}</td>
                  <td class="px-4 py-2 font-mono">{{ record.created_count }}</td>
                  <td class="px-4 py-2 font-mono">{{ record.updated_count }}</td>
                  <td class="px-4 py-2 font-mono">{{ record.unbound_group_count }}</td>
                  <td class="px-4 py-2">
                    <span v-if="record.error" class="text-red-600 dark:text-red-300">{{ record.error }}</span>
                    <span v-else class="text-emerald-600 dark:text-emerald-300">{{ t('common.success') }}</span>
                  </td>
                </tr>
                <tr v-if="!records.length">
                  <td colspan="6" class="px-4 py-8 text-center text-gray-400">{{ t('admin.upstreamAccounts.noRecords') }}</td>
                </tr>
              </tbody>
            </table>
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
  UpstreamAccountSyncItem,
  UpstreamAccountSyncRecord,
  UpstreamAccountSyncResult
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
const loadError = ref('')
const searchQuery = ref('')
const providerFilter = ref('')
const sourceFilter = ref('')
const actionFilter = ref('')

const columns = computed<Column[]>(() => [
  { key: 'source', label: t('admin.upstreamAccounts.columns.source') },
  { key: 'upstream_key_name', label: t('admin.upstreamAccounts.columns.upstreamKey') },
  { key: 'upstream_rate_multiplier', label: t('admin.upstreamAccounts.columns.upstreamRate') },
  { key: 'local_account_name', label: t('admin.upstreamAccounts.columns.localAccount') },
  { key: 'local_group_name', label: t('admin.upstreamAccounts.columns.localGroup') },
  { key: 'action', label: t('admin.upstreamAccounts.columns.action') },
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
  loadError.value = ''
  try {
    result.value = await adminAPI.upstreamAccountSync.getPreview()
  } catch (err) {
    const message = extractApiErrorMessage(err, t('admin.upstreamAccounts.loadFailed'))
    loadError.value = message
    result.value = null
    appStore.showError(message)
  } finally {
    loading.value = false
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

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
}

function actionLabel(row: UpstreamAccountSyncItem) {
  if (row.action === 'create') return t('admin.upstreamAccounts.actions.create')
  if (row.action === 'update') return t('admin.upstreamAccounts.actions.update')
  if (row.action === 'noop') return t('admin.upstreamAccounts.actions.noop')
  if (row.action === 'skip') return t('admin.upstreamAccounts.actions.skip')
  if (row.action === 'conflict') return t('admin.upstreamAccounts.actions.conflict')
  return row.action
}

function actionClass(row: UpstreamAccountSyncItem) {
  if (row.action === 'create') return 'badge-primary'
  if (row.action === 'update') return row.rate_violation ? 'badge-warning' : 'badge-success'
  if (row.action === 'noop') return 'badge-gray'
  if (row.action === 'conflict') return 'badge-warning'
  return 'badge-gray'
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
