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
      </template>

      <template #table>
        <div v-if="warnings.length" class="mb-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200">
          <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
        </div>

        <DataTable :columns="columns" :data="items" :loading="loading">
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
            <div v-if="row.matched" class="flex min-w-[12rem] flex-col gap-1">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.local_group_name }}</span>
              <code class="text-xs text-gray-500 dark:text-gray-400">#{{ row.local_group_id }}</code>
            </div>
            <span v-else class="text-sm text-gray-400">{{ t('admin.upstreamGroups.notMatched') }}</span>
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
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
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
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const result = ref<UpstreamGroupCompareResult | null>(null)
const loading = ref(false)
const applying = ref(false)
const loadError = ref('')

const columns = computed<Column[]>(() => [
  { key: 'upstream_group_name', label: t('admin.upstreamGroups.columns.upstreamGroup') },
  { key: 'upstream_rate', label: t('admin.upstreamGroups.columns.upstreamRate') },
  { key: 'local_group_name', label: t('admin.upstreamGroups.columns.localGroup') },
  { key: 'local_rate', label: t('admin.upstreamGroups.columns.localRate') },
  { key: 'status', label: t('admin.upstreamGroups.columns.status') },
])

const items = computed<UpstreamGroupComparison[]>(() => result.value?.items || [])
const warnings = computed(() => result.value?.warnings || [])
const records = computed<UpstreamGroupRateFixRecord[]>(() => result.value?.records || [])

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
  loadError.value = ''
  try {
    result.value = await adminAPI.upstreamManagement.getGroups()
  } catch (err) {
    const message = extractApiErrorMessage(err, t('admin.upstreamGroups.loadFailed'))
    loadError.value = message
    result.value = null
    appStore.showError(message)
  } finally {
    loading.value = false
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

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
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
