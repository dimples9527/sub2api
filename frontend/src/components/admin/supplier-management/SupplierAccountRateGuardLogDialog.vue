<template>
  <BaseDialog :show="show" title="账号倍率守护解除绑定日志" width="full" @close="emit('close')">
    <div class="account-rate-log-dialog">
      <div class="account-rate-log-summary" aria-label="日志状态摘要">
        <span>共 {{ total }} 条</span>
        <strong :class="{ active: pendingCount > 0 }">待处理 {{ pendingCount }} 条</strong>
      </div>

      <div class="account-rate-log-filters">
        <div class="account-rate-log-filter account-rate-log-search">
          <span class="sr-only">搜索日志</span>
          <Input v-model="filters.search" class="w-full" placeholder="搜索供应商、账号、分组或 Key" @enter="applyFilters" />
        </div>
        <div class="account-rate-log-filter">
          <span class="sr-only">供应商</span>
          <Select v-model="filters.providerID" class="w-full" :options="providerOptions" search-placeholder="搜索供应商" />
        </div>
        <div class="account-rate-log-filter">
          <span class="sr-only">运行模式</span>
          <Select v-model="filters.mode" class="w-full" :options="modeOptions" :searchable="false" />
        </div>
        <div class="account-rate-log-filter">
          <span class="sr-only">执行结果</span>
          <Select v-model="filters.result" class="w-full" :options="resultOptions" :searchable="false" />
        </div>
        <div class="account-rate-log-filter">
          <span class="sr-only">处理状态</span>
          <Select v-model="filters.status" class="w-full" :options="statusOptions" :searchable="false" />
        </div>
        <div class="account-rate-log-actions">
          <button class="sp-button small ghost" type="button" :disabled="loading" @click="toggleAllRecords">
            {{ showingAllRecords ? '仅看已解绑' : '显示所有记录' }}
          </button>
          <button class="sp-button small ghost" type="button" :disabled="loading" @click="resetFilters">重置</button>
          <button class="sp-button small primary" type="button" :disabled="loading" @click="applyFilters">
            {{ loading ? '查询中' : '查询' }}
          </button>
        </div>
      </div>

      <div v-if="error" class="sp-alert sp-error-line" role="alert">{{ error }}</div>

      <div class="sp-table-region account-rate-log-table-region">
        <DataTable :columns="columns" :data="items" :loading="loading" row-key="id" :sticky-first-column="false" :sticky-actions-column="false">
          <template #cell-status="{ row: log }">
            <button
              v-if="log.status === 'pending'"
              class="sp-link-button sp-change-log-status-action account-rate-log-status-pending"
              type="button"
              :disabled="handlingID === log.id"
              @click="markHandled(log)"
            >
              {{ handlingID === log.id ? '确认中' : '待处理' }}
            </button>
            <span v-else class="sp-status good account-rate-log-status-handled">已处理</span>
          </template>

          <template #cell-provider_name="{ row: log }">
            <strong class="account-rate-log-provider">{{ log.provider_name || `供应商 ${log.provider_id}` }}</strong>
            <div class="sp-sub">#{{ log.provider_id }}</div>
          </template>

          <template #cell-account="{ row: log }">
            <div :class="['account-rate-log-account-card', platformBadgeLightClass(log.platform), platformBorderClass(log.platform)]">
              <strong class="account-rate-log-account">{{ log.upstream_account_name || log.upstream_account_key || '-' }}</strong>
              <div v-if="log.upstream_account_key" class="sp-sub">{{ log.upstream_account_key }}</div>
              <div class="account-rate-log-local">本地：{{ log.local_account_name || '-' }}</div>
            </div>
          </template>

          <template #cell-local_group_name="{ row: log }">
            <strong :class="['account-rate-log-group', platformBadgeLightClass(log.platform), platformBorderClass(log.platform)]">{{ log.local_group_name || (log.local_group_id ? `分组 ${log.local_group_id}` : '-') }}</strong>
          </template>

          <template #cell-rates="{ row: log }">
            <div class="account-rate-log-rate-compare">
              <div class="account-rate-log-rate-row account-rate-log-rate-upstream"><span>上游有效</span><strong>{{ formatRate(log.effective_upstream_rate) }}</strong></div>
              <span class="account-rate-log-rate-divider" aria-hidden="true">/</span>
              <div class="account-rate-log-rate-row account-rate-log-rate-local"><span>本地分组</span><strong>{{ formatRate(log.local_group_rate) }}</strong></div>
            </div>
          </template>

          <template #cell-result="{ row: log }">
            <div class="account-rate-log-result" :class="`account-rate-log-result-${log.result}`">
              <div class="account-rate-log-result-line">
                <span class="sp-status" :class="resultTone(log.result)">{{ resultText(log.result) }}</span>
                <span class="sp-status info">{{ modeText(log.mode) }}</span>
              </div>
              <div v-if="log.error_message" class="account-rate-log-reason" :title="log.error_message">{{ log.error_message }}</div>
            </div>
          </template>

          <template #cell-scheduling="{ row: log }">
            <div :class="['account-rate-log-scheduling', platformBadgeLightClass(log.platform), platformBorderClass(log.platform)]">
              <span>{{ boundText(log.before_bound) }} → {{ boundText(log.after_bound) }}</span>
              <div v-if="log.before_schedulable !== undefined || log.after_schedulable !== undefined" class="sp-sub">
                调度：{{ schedulableText(log.before_schedulable) }} → {{ schedulableText(log.after_schedulable) }}
              </div>
            </div>
          </template>

          <template #cell-created_at="{ row: log }">
            <span>{{ formatTime(log.created_at) }}</span>
            <div v-if="log.handled_at" class="sp-sub">处理于 {{ formatTime(log.handled_at) }}</div>
          </template>

          <template #empty>暂无符合条件的账号倍率守护日志。</template>
        </DataTable>

        <Pagination
          v-if="total > 0"
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="false"
          @update:page="changePage"
        />
      </div>
    </div>

    <template #footer>
      <button class="sp-button primary" type="button" @click="emit('close')">关闭</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { supplierProvidersAPI } from '@/api/admin/supplierProviders'
import {
  listAccountRateGuardUnbindLogs,
  markAccountRateGuardUnbindLogHandled,
  type SupplierAccountRateGuardUnbindLog,
} from '@/api/admin/supplierAutomation'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import { platformBadgeLightClass, platformBorderClass } from '@/utils/platformColors'

interface Props {
  show: boolean
  initialLocalAccountId?: number
}

interface Emits {
  (event: 'close'): void
  (event: 'pending-count-change', count: number): void
}

const props = withDefaults(defineProps<Props>(), {
  initialLocalAccountId: 0,
})
const emit = defineEmits<Emits>()

const items = ref<SupplierAccountRateGuardUnbindLog[]>([])
const providers = ref<Array<{ id: number; name: string }>>([])
const loading = ref(false)
const handlingID = ref(0)
const error = ref('')
const total = ref(0)
const pendingCount = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filters = reactive({
  search: '',
  providerID: 0,
  mode: '',
  result: 'unbound',
  status: '',
})
const showingAllRecords = computed(() => filters.result === '')

const columns: Column[] = [
  { key: 'status', label: '处理状态', class: 'account-rate-log-col-status' },
  { key: 'provider_name', label: '供应商', class: 'account-rate-log-col-provider' },
  { key: 'account', label: '账号信息', class: 'account-rate-log-col-account' },
  { key: 'local_group_name', label: '解绑分组', class: 'account-rate-log-col-group' },
  { key: 'rates', label: '倍率对比', class: 'account-rate-log-col-rates' },
  { key: 'result', label: '执行结果', class: 'account-rate-log-col-result' },
  { key: 'scheduling', label: '绑定 / 调度变化', class: 'account-rate-log-col-scheduling' },
  { key: 'created_at', label: '时间', class: 'account-rate-log-col-time' },
]

const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name })),
])
const modeOptions: SelectOption[] = [
  { value: '', label: '全部模式' },
  { value: 'preview', label: '预览' },
  { value: 'execute', label: '执行' },
]
const resultOptions: SelectOption[] = [
  { value: '', label: '全部结果' },
  { value: 'planned', label: '计划解绑' },
  { value: 'unbound', label: '已解绑' },
  { value: 'failed', label: '失败' },
  { value: 'skipped', label: '已跳过' },
]
const statusOptions: SelectOption[] = [
  { value: '', label: '全部处理状态' },
  { value: 'pending', label: '待处理' },
  { value: 'handled', label: '已处理' },
]

watch(() => props.show, async (show) => {
  if (!show) return
  filters.result = 'unbound'
  page.value = 1
  await Promise.all([loadProviders(), loadLogs()])
})

async function loadProviders() {
  if (providers.value.length > 0) return
  try {
    const result = await supplierProvidersAPI.list({ page: 1, page_size: 200 })
    providers.value = result.items.map(provider => ({ id: provider.id, name: provider.name }))
  } catch {
    // 供应商筛选加载失败不阻断日志查询。
  }
}

async function loadLogs() {
  loading.value = true
  error.value = ''
  try {
    const result = await listAccountRateGuardUnbindLogs({
      provider_id: filters.providerID || undefined,
      local_account_id: props.initialLocalAccountId || undefined,
      search: filters.search.trim() || undefined,
      mode: filters.mode || undefined,
      result: filters.result || undefined,
      status: filters.status || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items
    total.value = result.total
    pendingCount.value = result.pending_count
    page.value = result.page
    pageSize.value = result.page_size
    emit('pending-count-change', result.pending_count)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载账号倍率守护日志失败'
  } finally {
    loading.value = false
  }
}

async function applyFilters() {
  page.value = 1
  await loadLogs()
}

async function toggleAllRecords() {
  filters.result = showingAllRecords.value ? 'unbound' : ''
  page.value = 1
  await loadLogs()
}

async function resetFilters() {
  Object.assign(filters, { search: '', providerID: 0, mode: '', result: 'unbound', status: '' })
  page.value = 1
  await loadLogs()
}

async function changePage(nextPage: number) {
  page.value = nextPage
  await loadLogs()
}

async function markHandled(log: SupplierAccountRateGuardUnbindLog) {
  if (log.status !== 'pending' || handlingID.value) return
  handlingID.value = log.id
  error.value = ''
  try {
    await markAccountRateGuardUnbindLogHandled(log.id)
    await loadLogs()
    if (items.value.length === 0 && page.value > 1) {
      page.value -= 1
      await loadLogs()
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '标记日志已处理失败'
  } finally {
    handlingID.value = 0
  }
}

function formatRate(value: number): string {
  return Number.isFinite(value) ? Number(value).toFixed(4).replace(/0+$/, '').replace(/\.$/, '') : '-'
}

function formatTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function modeText(mode: string): string {
  return mode === 'preview' ? '预览' : '执行'
}

function resultText(result: string): string {
  return ({ planned: '计划解绑', unbound: '已解绑', failed: '失败', skipped: '已跳过' } as Record<string, string>)[result] || result
}

function resultTone(result: string): string {
  if (result === 'unbound') return 'good'
  if (result === 'planned') return 'info'
  if (result === 'failed') return 'bad'
  return 'warn'
}

function boundText(value: boolean): string {
  return value ? '已绑定' : '未绑定'
}

function schedulableText(value?: boolean): string {
  if (value === undefined) return '-'
  return value ? '开启' : '关闭'
}
</script>

<style scoped>
.account-rate-log-dialog {
  --sp-panel: #ffffff;
  --sp-panel-2: #f9fafb;
  --sp-panel-3: #f3f4f6;
  --sp-line: #e5e7eb;
  --sp-soft: #f1f5f9;
  --sp-text: #111827;
  --sp-muted: #64748b;
  --sp-cyan: #3b82f6;
  --sp-green: #16a34a;
  --sp-amber: #d97706;
  --sp-red: #dc2626;
  --sp-blue: #2563eb;
  --sp-violet: #7c3aed;
  display: grid;
  gap: 14px;
  color: var(--sp-text);
}

:global(.dark) .account-rate-log-dialog {
  --sp-panel: #1f2937;
  --sp-panel-2: #111827;
  --sp-panel-3: #374151;
  --sp-line: #374151;
  --sp-soft: #374151;
  --sp-text: #f9fafb;
  --sp-muted: #9ca3af;
}

.account-rate-log-summary {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  color: var(--sp-muted);
  font-size: 13px;
}

.account-rate-log-summary strong {
  border: 1px solid var(--sp-line);
  border-radius: 999px;
  padding: 5px 10px;
  background: var(--sp-panel-2);
  color: inherit;
}

.account-rate-log-summary strong.active {
  border-color: color-mix(in srgb, var(--sp-amber) 34%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 9%, var(--sp-panel));
  color: var(--sp-amber);
}

.account-rate-log-filters {
  display: grid;
  grid-template-columns: minmax(240px, 1.5fr) repeat(4, minmax(135px, 0.75fr));
  gap: 10px;
  align-items: center;
}

.account-rate-log-filter,
.account-rate-log-search {
  min-width: 0;
}


.account-rate-log-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.account-rate-log-table-region {
  min-width: 0;
  overflow: hidden;
}

.account-rate-log-table-region :deep(.table-wrapper) {
  overflow-x: hidden;
}

.account-rate-log-table-region :deep(table) {
  width: 100%;
  min-width: 0 !important;
  table-layout: fixed;
}

.account-rate-log-table-region :deep(th),
.account-rate-log-table-region :deep(td) {
  min-width: 0 !important;
  white-space: normal;
  overflow-wrap: anywhere;
}

.account-rate-log-table-region :deep(tbody tr) {
  transition: background-color 0.15s ease;
}

.account-rate-log-table-region :deep(tbody tr:nth-child(even)),
.account-rate-log-table-region :deep(tbody tr:nth-child(even) td) {
  background: color-mix(in srgb, var(--sp-blue) 2.5%, var(--sp-panel));
}

.account-rate-log-table-region :deep(tbody tr:hover),
.account-rate-log-table-region :deep(tbody tr:hover td) {
  background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel));
}

.account-rate-log-table-region :deep(.account-rate-log-col-status) {
  width: 8%;
}

.account-rate-log-table-region :deep(.account-rate-log-col-provider) {
  width: 11%;
}

.account-rate-log-table-region :deep(.account-rate-log-col-account) {
  width: 18%;
}

.account-rate-log-table-region :deep(.account-rate-log-col-group) {
  width: 11%;
}

.account-rate-log-table-region :deep(.account-rate-log-col-rates) {
  width: 11%;
}

.account-rate-log-table-region :deep(.account-rate-log-col-result) {
  width: 12%;
}

.account-rate-log-table-region :deep(.account-rate-log-col-scheduling) {
  width: 15%;
}

.account-rate-log-table-region :deep(.account-rate-log-col-time) {
  width: 14%;
}

.account-rate-log-local,
.account-rate-log-reason {
  margin-top: 4px;
  color: var(--sp-muted);
  font-size: 12px;
}

.account-rate-log-reason {
  padding: 5px 7px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
  color: var(--sp-red);
  overflow-wrap: anywhere;
}

.account-rate-log-status-pending {
  gap: 5px;
  padding: 4px 8px;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 34%, var(--sp-line));
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-amber) 9%, var(--sp-panel));
  color: var(--sp-amber);
  font-size: 12px;
  font-weight: 700;
  text-decoration: none;
}

.account-rate-log-status-pending::before,
.account-rate-log-status-handled::before {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  content: '';
}

.account-rate-log-status-pending:hover {
  border-color: var(--sp-amber);
  background: color-mix(in srgb, var(--sp-amber) 14%, var(--sp-panel));
  color: var(--sp-amber);
  text-decoration: none;
}

.account-rate-log-provider,
.account-rate-log-group {
  display: inline-block;
  max-width: 100%;
  padding: 2px 6px;
  border-radius: 6px;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.account-rate-log-provider {
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel));
  color: var(--sp-blue);
}

.account-rate-log-account-card,
.account-rate-log-scheduling {
  max-width: 100%;
  padding: 5px 7px;
  border-left-width: 2px;
  border-left-style: solid;
  border-radius: 6px;
  overflow-wrap: anywhere;
}

.account-rate-log-account {
  display: block;
  color: inherit;
  line-height: 1.45;
}

.account-rate-log-account-card .sp-sub,
.account-rate-log-account-card .account-rate-log-local,
.account-rate-log-scheduling .sp-sub {
  color: inherit;
  opacity: 0.76;
}

.account-rate-log-group {
  border-width: 1px;
  border-style: solid;
}

.account-rate-log-scheduling {
  padding: 5px 7px;
  border-left: 2px solid color-mix(in srgb, var(--sp-cyan) 58%, var(--sp-line));
  border-radius: 6px;
  background: color-mix(in srgb, var(--sp-cyan) 5%, var(--sp-panel));
}

.account-rate-log-rate-compare {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: stretch;
  gap: 4px;
}

.account-rate-log-rate-row {
  display: grid;
  min-width: 0;
  place-items: center;
  gap: 2px;
  padding: 5px 4px;
  border-radius: 6px;
  text-align: center;
}

.account-rate-log-rate-row span {
  max-width: 100%;
  overflow: hidden;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-rate-log-rate-row strong {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.account-rate-log-rate-divider {
  align-self: center;
  color: var(--sp-muted);
  font-size: 11px;
  opacity: 0.62;
}

.account-rate-log-result-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}

.account-rate-log-rate-upstream {
  background: color-mix(in srgb, var(--sp-blue) 7%, var(--sp-panel));
  color: var(--sp-blue);
}

.account-rate-log-rate-local {
  background: color-mix(in srgb, var(--sp-violet) 7%, var(--sp-panel));
  color: var(--sp-violet);
}

.account-rate-log-rate-upstream span,
.account-rate-log-rate-local span {
  color: inherit;
  opacity: 0.78;
}

.account-rate-log-result {
  padding: 6px;
  border-left: 3px solid var(--sp-line);
  border-radius: 6px;
  background: var(--sp-panel-2);
}

.account-rate-log-result-unbound {
  border-left-color: var(--sp-green);
  background: color-mix(in srgb, var(--sp-green) 6%, var(--sp-panel));
}

.account-rate-log-result-planned {
  border-left-color: var(--sp-blue);
  background: color-mix(in srgb, var(--sp-blue) 6%, var(--sp-panel));
}

.account-rate-log-result-failed {
  border-left-color: var(--sp-red);
  background: color-mix(in srgb, var(--sp-red) 6%, var(--sp-panel));
}

.account-rate-log-result-skipped {
  border-left-color: var(--sp-amber);
  background: color-mix(in srgb, var(--sp-amber) 6%, var(--sp-panel));
}

@media (max-width: 1100px) {
  .account-rate-log-filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .account-rate-log-filters {
    grid-template-columns: 1fr;
  }

  .account-rate-log-actions {
    justify-content: stretch;
  }

  .account-rate-log-actions button {
    flex: 1;
  }
}
</style>
