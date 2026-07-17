<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <MonitorFiltersBar
          v-model:search="searchQuery"
          v-model:provider="providerFilter"
          v-model:enabled="enabledFilter"
          :loading="loading"
          @reload="reload"
          @create="openCreateDialog"
          @manage-templates="showTemplateManager = true"
          @search-input="handleSearch"
        />
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="monitors"
          :loading="loading"
          row-key="id"
          :row-class="monitorRowClass"
          class="monitor-table"
        >
          <template #cell-name="{ row, value }">
            <div class="monitor-name-cell flex min-w-0 flex-wrap items-center justify-end gap-1.5 md:justify-start">
              <span class="monitor-name-text min-w-0 break-all font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <HelpTooltip v-if="row.api_key_decrypt_failed" :content="t('admin.channelMonitor.apiKeyDecryptFailed')">
                <Icon name="exclamationTriangle" size="sm" class="text-red-500" />
              </HelpTooltip>
            </div>
          </template>

          <template #cell-provider="{ row }">
            <span class="monitor-provider-badge inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium" :class="providerBadgeClass(row.provider)">
              {{ providerLabel(row.provider) }}
            </span>
          </template>

          <template #cell-primary_model="{ row }">
            <MonitorPrimaryModelCell :row="row" />
          </template>

          <template #cell-availability_7d="{ row }">
            <span class="monitor-metric-text text-sm text-gray-900 dark:text-gray-100">{{ formatAvailability(row) }}</span>
          </template>

          <template #cell-latency="{ row }">
            <span class="monitor-metric-text text-sm text-gray-900 dark:text-gray-100">{{ formatLatency(row.primary_latency_ms) }}</span>
          </template>

          <template #cell-enabled="{ row }">
            <div class="monitor-toggle-cell">
              <Toggle :modelValue="row.enabled" @update:modelValue="toggleEnabled(row)" />
            </div>
          </template>

          <template #cell-actions="{ row }">
            <MonitorActionsCell
              :row="row"
              :running="runningId === row.id"
              :duplicating="duplicatingIds.has(row.id)"
              @run="handleRunNow"
              @duplicate="handleDuplicate"
              @edit="openEditDialog"
              @delete="handleDelete"
            />
          </template>

          <template #mobile-card="{ row }">
            <article class="monitor-mobile-card">
              <div class="monitor-mobile-card-head">
                <div class="monitor-mobile-title-block">
                  <div class="monitor-mobile-title-row">
                    <strong>{{ row.name }}</strong>
                    <HelpTooltip v-if="row.api_key_decrypt_failed" :content="t('admin.channelMonitor.apiKeyDecryptFailed')">
                      <Icon name="exclamationTriangle" size="sm" class="text-red-500" />
                    </HelpTooltip>
                  </div>
                  <div class="monitor-mobile-badges">
                    <span class="monitor-provider-badge inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium" :class="providerBadgeClass(row.provider)">
                      {{ providerLabel(row.provider) }}
                    </span>
                    <span class="monitor-status-badge" :class="statusBadgeClass(row.primary_status)">
                      {{ statusLabel(row.primary_status) }}
                    </span>
                  </div>
                </div>
                <Toggle :modelValue="row.enabled" @update:modelValue="toggleEnabled(row)" />
              </div>

              <div class="monitor-mobile-model">
                <span>{{ t('admin.channelMonitor.columns.primaryModel') }}</span>
                <strong>{{ row.primary_model }}</strong>
              </div>

              <div class="monitor-mobile-metrics">
                <div class="monitor-mobile-metric">
                  <span>{{ t('admin.channelMonitor.columns.availability7d') }}</span>
                  <strong>{{ formatAvailability(row) }}</strong>
                </div>
                <div class="monitor-mobile-metric">
                  <span>{{ t('admin.channelMonitor.columns.latency') }}</span>
                  <strong>{{ formatLatencyWithUnit(row.primary_latency_ms) }}</strong>
                </div>
                <div class="monitor-mobile-metric">
                  <span>{{ t('admin.channelMonitor.form.intervalSeconds') }}</span>
                  <strong>{{ row.interval_seconds }}s</strong>
                </div>
                <div class="monitor-mobile-metric">
                  <span>{{ t('monitorCommon.extraModelsHeader') }}</span>
                  <strong>{{ row.extra_models?.length || 0 }}</strong>
                </div>
              </div>

              <div v-if="row.group_name" class="monitor-mobile-group">
                <span>{{ t('admin.channelMonitor.form.groupName') }}</span>
                <strong>{{ row.group_name }}</strong>
              </div>

              <div class="monitor-mobile-actions">
                <MonitorActionsCell
                  :row="row"
                  :running="runningId === row.id"
                  :duplicating="duplicatingIds.has(row.id)"
                  @run="handleRunNow"
                  @duplicate="handleDuplicate"
                  @edit="openEditDialog"
                  @delete="handleDelete"
                />
              </div>
            </article>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.channelMonitor.noMonitorsYet')"
              :description="t('admin.channelMonitor.createFirstMonitor')"
              :action-text="t('admin.channelMonitor.createButton')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <MonitorFormDialog
      :show="showDialog"
      :monitor="editing"
      @close="closeDialog"
      @saved="reload"
    />

    <MonitorTemplateManagerDialog
      :show="showTemplateManager"
      @close="showTemplateManager = false"
      @updated="reload"
    />

    <MonitorRunResultDialog
      :show="showRunResult"
      :results="runResults"
      @close="showRunResult = false"
    />

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="deleteConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type {
  ChannelMonitor,
  CheckResult,
  ListParams,
  Provider,
} from '@/api/admin/channelMonitor'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import MonitorFiltersBar from '@/components/admin/monitor/MonitorFiltersBar.vue'
import MonitorFormDialog from '@/components/admin/monitor/MonitorFormDialog.vue'
import MonitorTemplateManagerDialog from '@/components/admin/monitor/MonitorTemplateManagerDialog.vue'
import MonitorRunResultDialog from '@/components/admin/monitor/MonitorRunResultDialog.vue'
import MonitorPrimaryModelCell from '@/components/admin/monitor/MonitorPrimaryModelCell.vue'
import MonitorActionsCell from '@/components/admin/monitor/MonitorActionsCell.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

const { t } = useI18n()
const appStore = useAppStore()
const {
  providerLabel,
  providerBadgeClass,
  statusLabel,
  statusBadgeClass,
  formatLatency,
  formatAvailability,
} = useChannelMonitorFormat()

const monitors = ref<ChannelMonitor[]>([])
const loading = ref(false)
const runningId = ref<number | null>(null)
const searchQuery = ref('')
const providerFilter = ref<Provider | ''>('')
const enabledFilter = ref<'' | 'true' | 'false'>('')
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })

const showDialog = ref(false)
const showTemplateManager = ref(false)
const editing = ref<ChannelMonitor | null>(null)
const showDeleteDialog = ref(false)
const deleting = ref<ChannelMonitor | null>(null)
const showRunResult = ref(false)
const runResults = ref<CheckResult[]>([])
const duplicatingIds = reactive(new Set<number>())

let abortController: AbortController | null = null
let searchTimeout: ReturnType<typeof setTimeout> | null = null

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.channelMonitor.columns.name'), sortable: false, class: 'monitor-name-column' },
  { key: 'provider', label: t('admin.channelMonitor.columns.provider'), sortable: false, class: 'monitor-provider-column' },
  { key: 'primary_model', label: t('admin.channelMonitor.columns.primaryModel'), sortable: false, class: 'monitor-model-column' },
  { key: 'availability_7d', label: t('admin.channelMonitor.columns.availability7d'), sortable: false, class: 'monitor-compact-column' },
  { key: 'latency', label: t('admin.channelMonitor.columns.latency'), sortable: false, class: 'monitor-compact-column' },
  { key: 'enabled', label: t('admin.channelMonitor.columns.enabled'), sortable: false, class: 'monitor-enabled-column' },
  { key: 'actions', label: t('admin.channelMonitor.columns.actions'), sortable: false, class: 'monitor-actions-column' },
])

const deleteConfirmMessage = computed(() => {
  const name = deleting.value?.name || ''
  return t('admin.channelMonitor.deleteConfirm', { name })
})

async function reload() {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  loading.value = true
  try {
    const params: ListParams = {
      page: pagination.page,
      page_size: pagination.page_size,
    }
    if (providerFilter.value) params.provider = providerFilter.value
    if (enabledFilter.value === 'true') params.enabled = true
    if (enabledFilter.value === 'false') params.enabled = false
    if (searchQuery.value.trim()) params.search = searchQuery.value.trim()

    const res = await adminAPI.channelMonitor.list(params, { signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    monitors.value = res.items || []
    pagination.total = res.total
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.loadError')))
  } finally {
    if (abortController === ctrl) {
      loading.value = false
      abortController = null
    }
  }
}

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    reload()
  }, 300)
}

function onPageChange(page: number) {
  pagination.page = page
  reload()
}

function onPageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  reload()
}

function openCreateDialog() {
  editing.value = null
  showDialog.value = true
}

function openEditDialog(row: ChannelMonitor) {
  editing.value = row
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editing.value = null
}

async function toggleEnabled(row: ChannelMonitor) {
  const next = !row.enabled
  try {
    await adminAPI.channelMonitor.update(row.id, { enabled: next })
    row.enabled = next
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

async function handleRunNow(row: ChannelMonitor) {
  if (runningId.value != null) return
  runningId.value = row.id
  try {
    const res = await adminAPI.channelMonitor.runNow(row.id)
    runResults.value = res.results || []
    showRunResult.value = true
    appStore.showSuccess(t('admin.channelMonitor.runSuccess'))
    // Refresh row to get latest status from backend
    void reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.runFailed')))
  } finally {
    runningId.value = null
  }
}

async function handleDuplicate(row: ChannelMonitor) {
  if (row.api_key_decrypt_failed) {
    appStore.showError(t('admin.channelMonitor.duplicateKeyUnavailable'))
    return
  }
  if (duplicatingIds.has(row.id)) return

  duplicatingIds.add(row.id)
  try {
    const duplicate = await adminAPI.channelMonitor.duplicate(row.id)
    appStore.showSuccess(t('admin.channelMonitor.duplicateSuccess', { name: duplicate.name }))
    await reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.duplicateFailed')))
  } finally {
    duplicatingIds.delete(row.id)
  }
}

function handleDelete(row: ChannelMonitor) {
  deleting.value = row
  showDeleteDialog.value = true
}

function monitorRowClass(row: ChannelMonitor) {
  const classes = ['monitor-row-card']
  if (!row.enabled) classes.push('is-disabled')
  if (row.primary_status) classes.push(`status-${row.primary_status}`)
  return classes
}

function formatLatencyWithUnit(ms: number | null | undefined) {
  if (ms == null) return formatLatency(ms)
  return `${formatLatency(ms)} ms`
}

async function confirmDelete() {
  if (!deleting.value) return
  try {
    await adminAPI.channelMonitor.del(deleting.value.id)
    appStore.showSuccess(t('admin.channelMonitor.deleteSuccess'))
    showDeleteDialog.value = false
    deleting.value = null
    reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

onMounted(reload)
onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
  abortController?.abort()
})
</script>

<style scoped>
.monitor-table :deep(table) {
  min-width: 860px;
}

.monitor-table :deep(.monitor-name-column) {
  min-width: 12rem;
}

.monitor-table :deep(.monitor-model-column) {
  min-width: 14rem;
}

.monitor-table :deep(.monitor-provider-column),
.monitor-table :deep(.monitor-compact-column),
.monitor-table :deep(.monitor-enabled-column) {
  min-width: 7rem;
}

.monitor-table :deep(.monitor-actions-column) {
  min-width: 10rem;
}

@media (max-width: 767px) {
  .monitor-table :deep(.monitor-row-card) {
    position: relative;
    overflow: hidden;
    border-radius: 8px;
    padding: 0;
    box-shadow: none;
  }

  .monitor-table :deep(.monitor-row-card::before) {
    position: absolute;
    inset: 0 auto 0 0;
    width: 3px;
    background: #94a3b8;
    content: "";
  }

  .monitor-table :deep(.monitor-row-card.status-operational::before) {
    background: #059669;
  }

  .monitor-table :deep(.monitor-row-card.status-degraded::before) {
    background: #d97706;
  }

  .monitor-table :deep(.monitor-row-card.status-failed::before),
  .monitor-table :deep(.monitor-row-card.status-error::before) {
    background: #dc2626;
  }

  .monitor-table :deep(.monitor-row-card.is-disabled) {
    background: #f8fafc;
    opacity: 0.82;
  }

  .monitor-table :deep(.monitor-provider-badge) {
    border-radius: 6px;
    padding: 2px 8px;
  }

  .monitor-table :deep(.monitor-metric-text) {
    display: inline-flex;
    min-width: 58px;
    justify-content: center;
    border-radius: 6px;
    background: #f1f5f9;
    padding: 3px 8px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 12px;
    font-weight: 700;
  }

  .monitor-mobile-card {
    display: grid;
    gap: 10px;
    padding: 12px;
  }

  .monitor-mobile-card-head {
    display: flex;
    min-width: 0;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
  }

  .monitor-mobile-title-block {
    display: grid;
    min-width: 0;
    gap: 6px;
  }

  .monitor-mobile-title-row {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 6px;
  }

  .monitor-mobile-title-row strong {
    min-width: 0;
    overflow-wrap: anywhere;
    color: #0f172a;
    font-size: 14px;
    font-weight: 800;
    line-height: 1.25;
  }

  .monitor-mobile-badges {
    display: flex;
    min-width: 0;
    flex-wrap: wrap;
    gap: 6px;
  }

  .monitor-status-badge {
    display: inline-flex;
    max-width: 100%;
    align-items: center;
    border-radius: 999px;
    padding: 2px 8px;
    font-size: 11px;
    font-weight: 800;
    line-height: 18px;
    white-space: nowrap;
  }

  .monitor-mobile-model,
  .monitor-mobile-group,
  .monitor-mobile-metric {
    display: grid;
    min-width: 0;
    gap: 4px;
    border-radius: 8px;
    background: #f8fafc;
    padding: 8px;
  }

  .monitor-mobile-model span,
  .monitor-mobile-group span,
  .monitor-mobile-metric span {
    color: #64748b;
    font-size: 11px;
    font-weight: 800;
    line-height: 1.15;
  }

  .monitor-mobile-model strong,
  .monitor-mobile-group strong {
    min-width: 0;
    overflow-wrap: anywhere;
    color: #0f172a;
    font-size: 13px;
    font-weight: 800;
    line-height: 1.3;
  }

  .monitor-mobile-metrics {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .monitor-mobile-metric strong {
    color: #0f172a;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 13px;
    font-weight: 800;
    line-height: 1.2;
  }

  .monitor-mobile-actions {
    border-top: 1px solid #eef2f7;
    padding-top: 8px;
  }

  :global(.dark) .monitor-table :deep(.monitor-row-card.is-disabled) {
    background: #111827;
  }

  :global(.dark) .monitor-table :deep(.monitor-metric-text) {
    background: #1f2937;
  }

  :global(.dark) .monitor-mobile-title-row strong,
  :global(.dark) .monitor-mobile-model strong,
  :global(.dark) .monitor-mobile-group strong,
  :global(.dark) .monitor-mobile-metric strong {
    color: #e5e7eb;
  }

  :global(.dark) .monitor-mobile-model,
  :global(.dark) .monitor-mobile-group,
  :global(.dark) .monitor-mobile-metric {
    background: #111827;
  }

  :global(.dark) .monitor-mobile-actions {
    border-top-color: #1f2937;
  }

  .monitor-table :deep(.monitor-name-column),
  .monitor-table :deep(.monitor-model-column),
  .monitor-table :deep(.monitor-provider-column),
  .monitor-table :deep(.monitor-compact-column),
  .monitor-table :deep(.monitor-enabled-column),
  .monitor-table :deep(.monitor-actions-column) {
    min-width: 0;
  }
}
</style>
