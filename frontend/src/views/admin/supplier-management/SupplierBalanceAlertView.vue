<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Supplier Operations / Balance Guard</div>
        <h1>供应商余额预警</h1>
        <p class="sp-subtitle">按供应商设置余额阈值，追踪低余额与恢复事件，并把事件交给独立通知渠道投递。</p>
      </div>
      <div class="sp-controls">
        <span v-if="lastLoadedAt" class="sp-data-note sp-balance-alert-loaded">更新于 {{ formatDateTime(lastLoadedAt) }}</span>
        <button class="sp-button" type="button" :disabled="loading || scanning" @click="loadAll">
          {{ loading ? '刷新中…' : '刷新数据' }}
        </button>
        <button class="sp-button primary" data-test="scan-balance" type="button" :disabled="scanning" @click="runScan">
          {{ scanning ? '扫描中…' : '手动扫描' }}
        </button>
      </div>
    </header>

    <div v-if="error" class="sp-alert sp-error-line" data-test="balance-alert-error">{{ error }}</div>
    <section class="sp-metric-grid sp-balance-alert-metrics" aria-label="余额预警概览">
      <article class="sp-metric-card sp-blue">
        <div class="sp-metric-label">供应商数</div>
        <div class="sp-metric-value">{{ configs.length }}</div>
        <div class="sp-metric-foot">已纳入余额扫描范围</div>
      </article>
      <article class="sp-metric-card sp-green">
        <div class="sp-metric-label">已启用预警</div>
        <div class="sp-metric-value">{{ enabledConfigCount }}</div>
        <div class="sp-metric-foot">阈值为 0 的配置不会触发</div>
      </article>
      <article class="sp-metric-card sp-amber">
        <div class="sp-metric-label">活动低余额</div>
        <div class="sp-metric-value">{{ activeLowCount }}</div>
        <div class="sp-metric-foot">根据最近一次余额结果估算</div>
      </article>
      <article class="sp-metric-card sp-red">
        <div class="sp-metric-label">最近扫描失败</div>
        <div class="sp-metric-value">{{ scanFailureCount }}</div>
        <div class="sp-metric-foot">单个供应商失败不会阻断其它扫描</div>
      </article>
    </section>

    <section class="sp-panel sp-balance-alert-panel" data-test="balance-config-section">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">01</span>
          <div>
            <h2>供应商余额配置</h2>
            <span>余额严格小于阈值时产生低余额事件</span>
          </div>
        </div>
        <span class="sp-status info">{{ configs.length }} 个供应商</span>
      </header>

      <DataTable
        :columns="configColumns"
        :data="configs"
        :loading="loading"
        row-key="provider_id"
        :virtualize-threshold="1000"
      >
        <template #cell-provider_name="{ row }">
          <div class="sp-entity">{{ row.provider_name }}</div>
          <div class="sp-sub">{{ row.provider_code }} · {{ row.provider_type }}</div>
        </template>
        <template #cell-last_balance="{ row }">
          <span :class="balanceTone(row)">{{ formatBalance(row.last_balance) }}</span>
        </template>
        <template #cell-threshold="{ row }">
          <span class="sp-num">{{ formatBalance(row.threshold) }}</span>
        </template>
        <template #cell-enabled="{ row }">
          <div class="sp-inline">
            <Toggle
              :model-value="row.enabled"
              :aria-label="`${row.provider_name}余额预警${row.enabled ? '已启用' : '已停用'}`"
              @click.stop
              @update:model-value="toggleConfig(row, $event)"
            />
            <span class="sp-status" :class="row.enabled ? 'good' : 'info'">{{ row.enabled ? '已启用' : '已停用' }}</span>
          </div>
        </template>
        <template #cell-cooldown_seconds="{ row }">
          {{ formatCooldown(row.cooldown_seconds) }}
        </template>
        <template #cell-last_scan_status="{ row }">
          <span class="sp-status" :class="scanStatusTone(row.last_scan_status)">{{ scanStatusLabel(row.last_scan_status) }}</span>
          <div v-if="row.last_scan_error" class="sp-sub sp-balance-alert-error-text">{{ row.last_scan_error }}</div>
        </template>
        <template #cell-actions="{ row }">
          <button class="sp-button small ghost" type="button" @click="openConfigDialog(row)">编辑配置</button>
        </template>
        <template #empty>
          <div class="sp-panel-body sp-empty-state">暂无已配置供应商，请先在供应商管理中创建供应商。</div>
        </template>
      </DataTable>
    </section>

    <section class="sp-panel sp-balance-alert-panel" data-test="balance-events-section">
      <header class="sp-panel-head sp-balance-alert-filter-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">02</span>
          <div>
            <h2>余额预警事件</h2>
            <span>低余额事件去重，余额恢复后自动闭环</span>
          </div>
        </div>
        <div class="sp-controls sp-balance-alert-filters">
          <Select
            v-model="eventTypeFilter"
            :options="eventTypeOptions"
            clearable
            aria-label="事件类型"
            class="sp-balance-alert-filter-control"
          />
          <Select
            v-model="eventStatusFilter"
            :options="eventStatusOptions"
            clearable
            aria-label="事件状态"
            class="sp-balance-alert-filter-control"
          />
          <button class="sp-button small ghost" type="button" @click="resetEventFilters">重置筛选</button>
        </div>
      </header>

      <DataTable
        :columns="eventColumns"
        :data="events"
        :loading="eventsLoading"
        row-key="id"
        :virtualize-threshold="1000"
      >
        <template #cell-provider_name="{ row }">
          <div class="sp-entity">{{ row.provider_name }}</div>
          <div class="sp-sub">{{ row.provider_code }}</div>
        </template>
        <template #cell-event_type="{ row }">
          <span class="sp-tag" :class="row.event_type === 'balance_recovered' ? 'good' : 'warn'">{{ eventTypeLabel(row.event_type) }}</span>
        </template>
        <template #cell-status="{ row }">
          <span class="sp-status" :class="row.status === 'active' ? 'warn' : 'good'">{{ eventStatusLabel(row.status) }}</span>
        </template>
        <template #cell-balance="{ row }">{{ formatBalance(row.balance) }}</template>
        <template #cell-threshold="{ row }">{{ formatBalance(row.threshold) }}</template>
        <template #cell-observed_at="{ row }">{{ formatDateTime(row.observed_at) }}</template>
        <template #cell-actions="{ row }">
          <button
            class="sp-button small"
            :class="row.status === 'resolved' ? 'danger' : 'ghost'"
            type="button"
            :disabled="row.status !== 'resolved' || deletingEventId !== null"
            :title="row.status === 'resolved' ? '删除余额预警事件' : '活动中的事件需等待余额恢复后才能删除'"
            @click="deleteEvent(row)"
          >
            {{ deletingEventId === row.id ? '删除中…' : row.status === 'resolved' ? '删除' : '等待恢复' }}
          </button>
        </template>
        <template #empty>
          <div class="sp-panel-body sp-empty-state">暂无余额预警事件。</div>
        </template>
      </DataTable>

      <div class="sp-pagination-row">
        <Pagination
          v-model:page="eventPage"
          v-model:page-size="eventPageSize"
          :total="eventTotal"
          :show-jump="eventTotal > 100"
          @update:page="loadEvents"
          @update:page-size="onEventPageSizeChange"
        />
      </div>
    </section>

    <BaseDialog :show="configDialogVisible" :title="configDialogTitle" width="normal" @close="closeConfigDialog">
      <div v-if="configForm" class="sp-form">
        <Input v-model="configForm.threshold" type="number" min="0" step="0.01" label="余额阈值" hint="余额严格小于该值时触发预警；填写 0 表示跳过预警。" />
        <Input
          :model-value="configForm.cooldown_seconds"
          type="number"
          min="0"
          step="60"
          label="通知冷却（秒）"
          hint="同一供应商、同一事件类型在冷却时间内不会重复投递。"
          @update:model-value="configForm.cooldown_seconds = $event"
        />
        <label class="sp-switch-field">
          <span>启用余额预警</span>
          <span class="sp-inline">
            <Toggle v-model="configForm.enabled" />
            <span class="sp-status" :class="configForm.enabled ? 'good' : 'info'">{{ configForm.enabled ? '已启用' : '已停用' }}</span>
          </span>
        </label>
        <div class="sp-form-note">供应商：{{ configForm.providerName }}。保存后下一次定时扫描会使用新的阈值和冷却时间。</div>
      </div>
      <template #footer>
        <button class="sp-button" type="button" @click="closeConfigDialog">取消</button>
        <button class="sp-button primary" type="button" :disabled="savingProviderId !== null" @click="saveConfig">
          {{ savingProviderId !== null ? '保存中…' : '保存配置' }}
        </button>
      </template>
    </BaseDialog>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { Column } from '@/components/common/types'
import {
  listSupplierBalanceAlertConfigs,
  listSupplierBalanceAlertEvents,
  deleteSupplierBalanceAlertEvent,
  scanSupplierBalanceAlerts,
  updateSupplierBalanceAlertConfig,
  type SupplierBalanceAlertConfig,
  type SupplierBalanceAlertConfigInput,
  type SupplierBalanceAlertEvent,
  type SupplierBalanceAlertEventListParams,
  type SupplierBalanceAlertScanResult,
} from '@/api/admin/supplierBalanceAlert'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const configs = ref<SupplierBalanceAlertConfig[]>([])
const events = ref<SupplierBalanceAlertEvent[]>([])
const loading = ref(false)
const eventsLoading = ref(false)
const scanning = ref(false)
const error = ref('')
const lastLoadedAt = ref('')
const scanResult = ref<SupplierBalanceAlertScanResult | null>(null)
const savingProviderId = ref<number | null>(null)
const eventPage = ref(1)
const eventPageSize = ref(10)
const eventTotal = ref(0)
const deletingEventId = ref<number | null>(null)
const eventTypeFilter = ref<string | number | boolean | null>(null)
const eventStatusFilter = ref<string | number | boolean | null>(null)
const configDialogVisible = ref(false)
const configDialogTitle = ref('编辑余额预警配置')
const configForm = ref<{
  providerId: number
  providerName: string
  enabled: boolean
  threshold: string
  cooldown_seconds: string
} | null>(null)

const configColumns: Column[] = [
  { key: 'provider_name', label: '供应商' },
  { key: 'last_balance', label: '当前余额' },
  { key: 'threshold', label: '预警阈值' },
  { key: 'enabled', label: '预警开关' },
  { key: 'cooldown_seconds', label: '冷却时间' },
  { key: 'last_scan_status', label: '最近扫描' },
  { key: 'actions', label: '操作' },
]

const eventColumns: Column[] = [
  { key: 'provider_name', label: '供应商' },
  { key: 'event_type', label: '事件类型' },
  { key: 'status', label: '状态' },
  { key: 'balance', label: '观测余额' },
  { key: 'threshold', label: '阈值' },
  { key: 'observed_at', label: '发生时间' },
  { key: 'actions', label: '操作' },
]

const eventTypeOptions: SelectOption[] = [
  { value: 'balance_low', label: '余额不足' },
  { value: 'balance_recovered', label: '余额恢复' },
]

const eventStatusOptions: SelectOption[] = [
  { value: 'active', label: '活动中' },
  { value: 'resolved', label: '已恢复' },
]

const enabledConfigCount = computed(() => configs.value.filter((item) => item.enabled).length)
const activeLowCount = computed(() => configs.value.filter((item) => isActiveLowConfig(item)).length)
const scanFailureCount = computed(() => configs.value.filter((item) => item.last_scan_status === 'error').length)

async function loadAll(): Promise<boolean> {
  loading.value = true
  error.value = ''
  try {
    const [configResult, eventsLoaded] = await Promise.all([
      listSupplierBalanceAlertConfigs(),
      loadEvents(),
    ])
    configs.value = configResult.items ?? []
    if (!eventsLoaded) return false
    lastLoadedAt.value = new Date().toISOString()
    return true
  } catch (err) {
    error.value = extractApiErrorMessage(err, '加载余额预警数据失败')
    return false
  } finally {
    loading.value = false
  }
}

async function loadEvents(): Promise<boolean> {
  eventsLoading.value = true
  try {
    const params: SupplierBalanceAlertEventListParams = {
      page: eventPage.value,
      page_size: eventPageSize.value,
      event_type: asStringFilter(eventTypeFilter.value) as SupplierBalanceAlertEventListParams['event_type'],
      status: asStringFilter(eventStatusFilter.value) as SupplierBalanceAlertEventListParams['status'],
    }
    const result = await listSupplierBalanceAlertEvents(params)
    events.value = result.items ?? []
    eventTotal.value = result.total ?? 0
    return true
  } catch (err) {
    error.value = extractApiErrorMessage(err, '加载余额预警事件失败')
    return false
  } finally {
    eventsLoading.value = false
  }
}

async function deleteEvent(event: SupplierBalanceAlertEvent): Promise<void> {
  if (deletingEventId.value !== null) return
  if (event.status !== 'resolved') {
    error.value = '活动中的余额预警事件不能删除，请等待余额恢复后再删除'
    return
  }
  if (!window.confirm(`确认删除「${event.provider_name}」的${eventTypeLabel(event.event_type)}事件？删除后无法恢复。`)) return

  deletingEventId.value = event.id
  error.value = ''
  try {
    await deleteSupplierBalanceAlertEvent(event.id)
    if (!(await loadEvents())) return
    const maxPage = Math.max(1, Math.ceil(eventTotal.value / eventPageSize.value))
    if (eventPage.value > maxPage) {
      eventPage.value = maxPage
      if (!(await loadEvents())) return
    }
    appStore.showSuccess('余额预警事件已删除')
  } catch (err) {
    error.value = extractApiErrorMessage(err, '删除余额预警事件失败')
  } finally {
    deletingEventId.value = null
  }
}

async function runScan(): Promise<void> {
  scanning.value = true
  error.value = ''
  scanResult.value = null
  try {
    scanResult.value = await scanSupplierBalanceAlerts()
    if (!(await loadAll())) return
    appStore.showSuccess(scanSummary(scanResult.value))
  } catch (err) {
    error.value = extractApiErrorMessage(err, '手动扫描余额失败')
  } finally {
    scanning.value = false
  }
}

function openConfigDialog(config: SupplierBalanceAlertConfig): void {
  configDialogTitle.value = `编辑「${config.provider_name}」余额预警`
  configForm.value = {
    providerId: config.provider_id,
    providerName: config.provider_name,
    enabled: config.enabled,
    threshold: config.threshold || '0',
    cooldown_seconds: String(config.cooldown_seconds || 0),
  }
  configDialogVisible.value = true
}

function forceCloseConfigDialog(): void {
  configDialogVisible.value = false
  configForm.value = null
}

function closeConfigDialog(): void {
  if (savingProviderId.value !== null) return
  forceCloseConfigDialog()
}

async function saveConfig(): Promise<void> {
  const form = configForm.value
  if (!form) return
  const threshold = form.threshold.trim()
  const cooldownSeconds = Number(form.cooldown_seconds)
  if (!threshold || Number.isNaN(Number(threshold)) || Number(threshold) < 0) {
    error.value = '余额阈值必须是大于或等于 0 的数字'
    return
  }
  if (!Number.isFinite(cooldownSeconds) || cooldownSeconds < 0) {
    error.value = '通知冷却时间必须是大于或等于 0 的秒数'
    return
  }

  const input: SupplierBalanceAlertConfigInput = {
    enabled: form.enabled,
    threshold,
    cooldown_seconds: Math.floor(cooldownSeconds),
  }
  savingProviderId.value = form.providerId
  error.value = ''
  try {
    const saved = await updateSupplierBalanceAlertConfig(form.providerId, input)
    const index = configs.value.findIndex((item) => item.provider_id === form.providerId)
    if (index >= 0) configs.value[index] = saved
    forceCloseConfigDialog()
    appStore.showSuccess('余额预警配置已保存')
  } catch (err) {
    error.value = extractApiErrorMessage(err, '保存余额预警配置失败')
  } finally {
    savingProviderId.value = null
  }
}

async function toggleConfig(config: SupplierBalanceAlertConfig, enabled: boolean): Promise<void> {
  if (savingProviderId.value !== null) return
  savingProviderId.value = config.provider_id
  try {
    const saved = await updateSupplierBalanceAlertConfig(config.provider_id, {
      enabled,
      threshold: config.threshold,
      cooldown_seconds: config.cooldown_seconds,
    })
    const index = configs.value.findIndex((item) => item.provider_id === config.provider_id)
    if (index >= 0) configs.value[index] = saved
    appStore.showSuccess(`${config.provider_name}余额预警已${enabled ? '启用' : '停用'}`)
  } catch (err) {
    error.value = extractApiErrorMessage(err, '更新余额预警开关失败')
  } finally {
    savingProviderId.value = null
  }
}

function resetEventFilters(): void {
  eventTypeFilter.value = null
  eventStatusFilter.value = null
  eventPage.value = 1
  void loadEvents()
}

function onEventPageSizeChange(): void {
  eventPage.value = 1
  void loadEvents()
}

function scanSummary(result: SupplierBalanceAlertScanResult): string {
  return `扫描完成：检查 ${result.checked} 个，触发 ${result.triggered} 个，恢复 ${result.recovered} 个，失败 ${result.failed} 个`
}

function formatBalance(value?: string | null): string {
  if (value === undefined || value === null || value === '') return '—'
  const number = Number(value)
  return Number.isFinite(number) ? number.toLocaleString('zh-CN', { maximumFractionDigits: 8 }) : value
}

function formatCooldown(seconds: number): string {
  if (!seconds || seconds < 0) return '不冷却'
  if (seconds % 86400 === 0) return `${seconds / 86400} 天`
  if (seconds % 3600 === 0) return `${seconds / 3600} 小时`
  if (seconds % 60 === 0) return `${seconds / 60} 分钟`
  return `${seconds} 秒`
}

function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function isActiveLowConfig(config: SupplierBalanceAlertConfig): boolean {
  if (!config.enabled || !config.last_balance || !config.threshold || Number(config.threshold) <= 0) return false
  return Number(config.last_balance) < Number(config.threshold)
}

function balanceTone(config: SupplierBalanceAlertConfig): string {
  return isActiveLowConfig(config) ? 'sp-balance-alert-low' : 'sp-balance-alert-normal'
}

function scanStatusTone(status: string): string {
  if (status === 'ok') return 'good'
  if (status === 'error') return 'bad'
  if (status === 'skipped') return 'warn'
  return 'info'
}

function scanStatusLabel(status: string): string {
  if (status === 'ok') return '正常'
  if (status === 'error') return '失败'
  if (status === 'skipped') return '已跳过'
  return '未扫描'
}

function eventTypeLabel(eventType: string): string {
  return eventType === 'balance_recovered' ? '余额恢复' : '余额不足'
}

function eventStatusLabel(status: string): string {
  return status === 'active' ? '活动中' : status === 'resolved' ? '已恢复' : status
}

function asStringFilter(value: string | number | boolean | null): string | undefined {
  return typeof value === 'string' && value ? value : undefined
}

watch([eventTypeFilter, eventStatusFilter], () => {
  eventPage.value = 1
  void loadEvents()
})

onMounted(() => {
  void loadAll()
})
</script>

<style scoped>
.sp-balance-alert-metrics { grid-template-columns: repeat(4, minmax(145px, 1fr)); }
.sp-balance-alert-panel { margin-bottom: 1rem; }
.sp-balance-alert-loaded { margin: 0; padding: 0.45rem 0.65rem; border-left: 0; background: var(--sp-panel-2); }
.sp-balance-alert-filter-head { align-items: flex-start; }
.sp-balance-alert-filters { justify-content: flex-end; }
.sp-balance-alert-filter-control { min-width: 9rem; }
.sp-balance-alert-error-text { max-width: 16rem; overflow: hidden; color: var(--sp-red); text-overflow: ellipsis; }
.sp-balance-alert-low { color: var(--sp-red); font-weight: 700; }
.sp-balance-alert-normal { color: var(--sp-text); }
.sp-empty-state { color: var(--sp-muted); text-align: center; }
.sp-pagination-row { display: flex; justify-content: flex-end; padding: 0.75rem 1rem 1rem; }

@media (max-width: 760px) {
  .sp-balance-alert-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .sp-balance-alert-filter-head { align-items: stretch; }
  .sp-balance-alert-filters { justify-content: stretch; }
  .sp-balance-alert-filter-control { min-width: 0; flex: 1 1 8rem; }
}

@media (max-width: 480px) {
  .sp-balance-alert-metrics { grid-template-columns: 1fr 1fr; gap: 0.5rem; }
  .sp-balance-alert-metrics .sp-metric-card { min-height: 92px; padding: 0.75rem; }
  .sp-balance-alert-metrics .sp-metric-value { font-size: 1.5rem; }
}
</style>
