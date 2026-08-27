<template>
  <SupplierModuleLayout>
    <header class="sp-page-head sp-health-head">
      <div>
        <div class="sp-eyebrow">供应商运营 / 账号健康</div>
        <h1>账号健康趋势</h1>
        <p class="sp-subtitle">按账号查看健康状态与响应时间变化，快速定位慢响应和连续失败。</p>
      </div>
      <div class="sp-controls">
        <span v-if="lastLoadedAt" class="sp-data-note sp-health-loaded">更新于 {{ formatDateTime(lastLoadedAt) }}</span>
        <div class="sp-segmented" aria-label="趋势范围">
          <button
            v-for="option in rangeOptions"
            :key="option.value"
            type="button"
            :class="{ active: selectedRange === option.value }"
            :disabled="trendLoading"
            @click="selectRange(option.value)"
          >{{ option.label }}</button>
        </div>
        <button class="sp-button" type="button" :disabled="loading || trendLoading" @click="refresh">
          {{ loading || trendLoading ? '刷新中' : '刷新' }}
        </button>
      </div>
    </header>

    <section class="sp-panel sp-health-filter-panel" aria-label="账号健康筛选">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <div>
            <span class="sp-panel-kicker">Account Health</span>
            <h2>筛选账号</h2>
            <p>选择账号后查看健康状态和响应时间趋势。</p>
          </div>
        </div>
        <span class="sp-status info">{{ total }} 个账号</span>
      </header>
      <div class="sp-health-filters">
        <div class="sp-health-filter-control" role="group" aria-labelledby="supplier-health-provider-label">
          <span id="supplier-health-provider-label" class="sr-only">供应商筛选</span>
          <Select v-model="providerId" class="w-full" :options="providerOptions" :searchable="false" />
        </div>
        <div class="sp-health-filter-control" role="group" aria-labelledby="supplier-health-platform-label">
          <span id="supplier-health-platform-label" class="sr-only">平台筛选</span>
          <Select v-model="platform" class="w-full" :options="platformOptions" :searchable="false" />
        </div>
        <div class="sp-health-filter-control sp-health-search" role="group" aria-labelledby="supplier-health-search-label">
          <span id="supplier-health-search-label" class="sr-only">账号名称或 ID</span>
          <Input v-model="search" class="w-full" placeholder="搜索账号名称或 ID" />
        </div>
        <div class="sp-health-filter-control" role="group" aria-labelledby="supplier-health-status-label">
          <span id="supplier-health-status-label" class="sr-only">当前健康状态</span>
          <Select v-model="healthStatus" class="w-full" :options="healthStatusOptions" :searchable="false" />
        </div>
      </div>
    </section>

    <section class="sp-panel sp-health-account-panel">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <div>
            <span class="sp-panel-kicker">Accounts</span>
            <h2>账号列表</h2>
          </div>
        </div>
        <span v-if="guardDisabledCount > 0" class="sp-status warn">{{ guardDisabledCount }} 个账号未启用健康守护</span>
      </header>
      <DataTable :columns="accountColumns" :data="accounts" :loading="loading" row-key="local_account_id">
        <template #cell-account="{ row: account }">
          <div class="sp-health-account-cell">
            <button
              class="sp-health-account-button"
              type="button"
              :class="{ active: selectedAccountId === account.local_account_id }"
              @click="selectAccount(account.local_account_id)"
            >
              <strong>{{ account.local_account_name || ('账号 #' + account.local_account_id) }}</strong>
              <span>ID {{ account.local_account_id }}</span>
            </button>
            <div class="sp-health-account-tags">
              <span :class="['sp-health-chip', 'sp-health-chip--provider', providerTone(account.provider_name)]">
                {{ account.provider_name || '未关联供应商' }}
              </span>
              <span :class="['sp-health-chip', 'sp-health-chip--platform', platformTone(account.platform)]">
                {{ account.platform || '未知平台' }}
              </span>
            </div>
          </div>
        </template>
        <template #cell-status="{ row: account }">
          <span class="sp-status" :class="statusTone(account.status)">{{ statusLabel(account.status) }}</span>
          <div v-if="account.consecutive_failures > 0" class="sp-sub sp-health-failure-count">连续失败 {{ account.consecutive_failures }} 次</div>
        </template>
        <template #cell-latency_ms="{ row: account }">
          <span class="sp-health-latency">{{ formatLatency(account.latency_ms) }}</span>
          <div v-if="account.latency_limit_ms > 0" class="sp-sub">阈值 {{ account.latency_limit_ms }} ms</div>
        </template>
        <template #cell-health_trend="{ row: account }">
          <div class="sp-health-trend-cell" :title="accountTrendTitle(account)">
            <div v-if="visibleAccountTrend(account.local_account_id).length" class="sp-health-trend-meta">
              <span>{{ formatTrendHealthRate(account.local_account_id) }}</span>
              <time>{{ trendLatestTime(account.local_account_id) }}</time>
            </div>
            <div v-if="trendLoadingByAccountId[account.local_account_id]" class="sp-health-trend-bars sp-health-trend-bars--loading" aria-label="正在加载健康趋势">
              <span v-for="index in TREND_BAR_COUNT" :key="index" class="sp-health-trend-bar sp-health-trend-bar--loading" />
            </div>
            <div v-else-if="visibleAccountTrend(account.local_account_id).length" class="sp-health-trend-bars" aria-label="账号健康趋势">
              <span
                v-for="(point, index) in visibleAccountTrend(account.local_account_id)"
                :key="`${point.checked_at}-${index}`"
                :class="['sp-health-trend-bar', `sp-health-trend-bar--${statusTone(point.status)}`]"
                :style="{ height: latencyBarHeight(point.latency_ms, account.local_account_id) }"
                :title="accountTrendPointTitle(point)"
              />
            </div>
            <span v-else class="sp-health-trend-empty">暂无趋势</span>
          </div>
        </template>
        <template #cell-checked_at="{ row: account }">
          <span>{{ account.checked_at ? formatDateTime(account.checked_at) : '尚未检测' }}</span>
          <div class="sp-sub">{{ account.guard_enabled ? '健康守护已启用' : '健康守护未启用' }}</div>
        </template>
        <template #empty>
          <div class="sp-empty-state sp-health-empty">
            <strong>暂无可展示的账号</strong>
            <span>请调整筛选条件，或先启用供应商账号健康守护任务。</span>
          </div>
        </template>
      </DataTable>
      <footer v-if="total > 0" class="sp-health-pagination">
        <Pagination
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="false"
          @update:page="handlePageChange"
        />
      </footer>
    </section>

    <section v-if="selectedAccount" class="sp-health-detail-grid">
      <article class="sp-panel sp-health-summary-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <div>
              <span class="sp-panel-kicker">Selected Account</span>
              <h2>{{ selectedAccount.local_account_name || ('账号 #' + selectedAccount.local_account_id) }}</h2>
              <p>{{ selectedAccount.provider_name || '—' }} · {{ selectedAccount.platform || '未知平台' }} · ID {{ selectedAccount.local_account_id }}</p>
            </div>
          </div>
          <span class="sp-status" :class="statusTone(latestPoint?.status || selectedAccount.status)">{{ statusLabel(latestPoint?.status || selectedAccount.status) }}</span>
        </header>
        <div class="sp-health-kpis">
          <div class="sp-chart-kpi">
            <span>当前状态</span>
            <b :class="statusTone(latestPoint?.status || selectedAccount.status)">{{ statusLabel(latestPoint?.status || selectedAccount.status) }}</b>
          </div>
          <div class="sp-chart-kpi">
            <span>最近响应</span>
            <b>{{ formatLatency(latestPoint?.latency_ms ?? selectedAccount.latency_ms) }}</b>
          </div>
          <div class="sp-chart-kpi">
            <span>检测阈值</span>
            <b>{{ selectedAccount.latency_limit_ms > 0 ? selectedAccount.latency_limit_ms + ' ms' : '未设置' }}</b>
          </div>
          <div class="sp-chart-kpi">
            <span>检测记录</span>
            <b>{{ trendPoints.length }}</b>
          </div>
        </div>
        <div v-if="!trendLoading && !trendPoints.length" class="sp-empty-state sp-health-no-history">
          <strong>尚无健康检测记录</strong>
          <span>{{ selectedAccount.guard_enabled ? '健康守护运行后会在这里生成趋势记录。' : '当前账号未启用健康守护，不会产生趋势记录。' }}</span>
        </div>
        <div v-else class="sp-health-latest">
          <div><span>最近检测</span><strong>{{ latestPoint ? formatDateTime(latestPoint.checked_at) : '—' }}</strong></div>
          <div><span>失败原因</span><strong>{{ latestPoint?.reason || '—' }}</strong></div>
          <div><span>动作</span><strong>{{ latestPoint?.action || '—' }}</strong></div>
          <div><span>错误详情</span><strong>{{ latestPoint?.error_message || '—' }}</strong></div>
        </div>
      </article>

      <article class="sp-panel sp-health-chart-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <div>
              <span class="sp-panel-kicker">Health Trend</span>
              <h2>健康状态趋势</h2>
              <p>可用、慢响应和失败分别映射为 2、1、0。</p>
            </div>
          </div>
          <span class="sp-status info">{{ selectedRange }}</span>
        </header>
        <div v-if="trendLoading" class="sp-health-chart-state">正在加载健康状态趋势…</div>
        <div v-else-if="trendPoints.length" class="sp-health-chart"><Line :data="statusChartData" :options="statusChartOptions" /></div>
        <div v-else class="sp-health-chart-state">尚无健康检测记录</div>
      </article>

      <article class="sp-panel sp-health-chart-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <div>
              <span class="sp-panel-kicker">Latency</span>
              <h2>响应时间趋势</h2>
              <p>失败记录保留为空值，不会误显示为 0 ms。</p>
            </div>
          </div>
          <span class="sp-status info">{{ selectedRange }}</span>
        </header>
        <div v-if="trendLoading" class="sp-health-chart-state">正在加载响应时间趋势…</div>
        <div v-else-if="trendPoints.length" class="sp-health-chart"><Line :data="latencyChartData" :options="latencyChartOptions" /></div>
        <div v-else class="sp-health-chart-state">尚无健康检测记录</div>
      </article>
    </section>

    <section v-if="selectedAccount && trendPoints.length" class="sp-panel sp-health-event-panel">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <div>
            <span class="sp-panel-kicker">Diagnostics</span>
            <h2>检测明细</h2>
            <p>查看每次检测的状态、失败原因、动作和错误详情。</p>
          </div>
        </div>
      </header>
      <div class="sp-health-events">
        <article v-for="point in reversedTrendPoints" :key="point.checked_at + '-' + point.status" class="sp-health-event">
          <div class="sp-health-event-head">
            <span class="sp-status" :class="statusTone(point.status)">{{ statusLabel(point.status) }}</span>
            <time>{{ formatDateTime(point.checked_at) }}</time>
            <strong>{{ formatLatency(point.latency_ms) }}</strong>
          </div>
          <dl>
            <div><dt>失败原因</dt><dd>{{ point.reason || '—' }}</dd></div>
            <div><dt>动作</dt><dd>{{ point.action || '—' }}</dd></div>
            <div><dt>错误详情</dt><dd>{{ point.error_message || '—' }}</dd></div>
          </dl>
        </article>
      </div>
    </section>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Title,
  Tooltip,
  type ChartData,
  type ChartOptions,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { createKeyedRequestLoader } from '@/composables/useKeyedRequestLoader'
import {
  getSupplierAccountHealthTrend,
  listSupplierAccountHealthAccounts,
  type SupplierAccountHealthAccount,
  type SupplierAccountHealthPoint,
  type SupplierAccountHealthRange,
} from '@/api/admin/supplierAccountHealth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Column } from '@/components/common/types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend)

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const rangeOptions: Array<{ value: SupplierAccountHealthRange; label: string }> = [
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' },
]
const selectedRange = ref<SupplierAccountHealthRange>('24h')
const accounts = ref<SupplierAccountHealthAccount[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 50
const loading = ref(false)
const trendLoading = ref(false)
const error = ref('')
const search = ref('')
const providerId = ref<number | null>(null)
const platform = ref('')
const healthStatus = ref('')
const selectedAccountId = ref<number | null>(null)
const trendPoints = ref<SupplierAccountHealthPoint[]>([])
const healthTrendByAccountId = ref<Record<number, SupplierAccountHealthPoint[]>>({})
const trendLoadingByAccountId = ref<Record<number, boolean>>({})
const lastLoadedAt = ref('')
const TREND_BAR_COUNT = 28
const TREND_LOAD_CONCURRENCY = 6
let trendLoadSequence = 0
let accountTrendLoadSequence = 0

const loadTrendRequest = createKeyedRequestLoader(
  getSupplierAccountHealthTrend,
  (accountId, range) => `${accountId}:${range}`,
)

const accountColumns: Column[] = [
  { key: 'account', label: '账号 / 供应商 / 平台' },
  { key: 'status', label: '当前健康状态' },
  { key: 'latency_ms', label: '最近响应' },
  { key: 'health_trend', label: '健康趋势' },
  { key: 'checked_at', label: '最近检测' },
]

const providerOptions = computed<SelectOption[]>(() => [
  { value: null, label: '全部供应商' },
  ...Array.from(new Map(accounts.value.map(account => [account.provider_id, account.provider_name || ('供应商 #' + account.provider_id)])).entries())
    .map(([value, label]) => ({ value, label })),
])
const platformOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部平台' },
  ...Array.from(new Set(accounts.value.map(account => account.platform).filter(Boolean))).map(value => ({ value, label: value })),
])
const healthStatusOptions: SelectOption[] = [
  { value: '', label: '全部健康状态' },
  { value: 'healthy', label: '可用' },
  { value: 'slow', label: '慢响应' },
  { value: 'failed', label: '失败' },
]

const selectedAccount = computed(() => accounts.value.find(account => account.local_account_id === selectedAccountId.value) || null)
const latestPoint = computed(() => trendPoints.value[trendPoints.value.length - 1] || null)
const reversedTrendPoints = computed(() => [...trendPoints.value].reverse())
const guardDisabledCount = computed(() => accounts.value.filter(account => !account.guard_enabled).length)

function statusLabel(status?: string | null): string {
  if (status === 'healthy') return '可用'
  if (status === 'slow') return '慢响应'
  if (status === 'failed') return '失败'
  return '未检测'
}

function statusTone(status?: string | null): string {
  if (status === 'healthy') return 'good'
  if (status === 'slow') return 'warn'
  if (status === 'failed') return 'bad'
  return 'info'
}

function stableTone(value: string | null | undefined, prefix: string, paletteSize: number): string {
  const text = value || ''
  let hash = 0
  for (let index = 0; index < text.length; index += 1) {
    hash = (hash * 31 + text.charCodeAt(index)) | 0
  }
  return `${prefix}-${Math.abs(hash) % paletteSize}`
}

function providerTone(value?: string | null): string {
  return stableTone(value, 'provider', 5)
}

function platformTone(value?: string | null): string {
  return stableTone(value, 'platform', 5)
}

function visibleAccountTrend(accountId: number): SupplierAccountHealthPoint[] {
  return (healthTrendByAccountId.value[accountId] || []).slice(-TREND_BAR_COUNT)
}

function formatTrendHealthRate(accountId: number): string {
  const points = visibleAccountTrend(accountId)
  if (!points.length) return ''
  const healthyCount = points.filter(point => point.status === 'healthy').length
  return `健康率 ${((healthyCount / points.length) * 100).toFixed(0)}%`
}

function trendLatestTime(accountId: number): string {
  const points = visibleAccountTrend(accountId)
  const latest = points[points.length - 1]
  return latest ? formatDateTime(latest.checked_at) : ''
}

function accountTrendTitle(account: SupplierAccountHealthAccount): string {
  const points = visibleAccountTrend(account.local_account_id)
  if (!points.length) return '暂无健康趋势数据'
  return `${account.local_account_name || `账号 #${account.local_account_id}`}：${points.length}次检测`
}

function accountTrendPointTitle(point: SupplierAccountHealthPoint): string {
  return `${formatDateTime(point.checked_at)} ${statusLabel(point.status)}，响应 ${formatLatency(point.latency_ms)}`
}

function latencyBarHeight(value: number | null | undefined, accountId: number): string {
  if (value === null || value === undefined || value <= 0) return '14%'
  const latencies = (healthTrendByAccountId.value[accountId] || [])
    .map(point => point.latency_ms)
    .filter((latency): latency is number => typeof latency === 'number' && latency > 0)
  const maxLatency = Math.max(...latencies, 0)
  if (maxLatency <= 0) return '14%'
  return `${Math.max(16, Math.min(100, Math.round((value / maxLatency) * 100)))}%`
}

function formatLatency(value?: number | null): string {
  if (value === null || value === undefined) return '—'
  return value + ' ms'
}

function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function healthValue(status: string): number {
  if (status === 'healthy') return 2
  if (status === 'slow') return 1
  return 0
}

function chartColors() {
  const dark = document.documentElement.classList.contains('dark')
  return { text: dark ? '#e5e7eb' : '#374151', grid: dark ? '#374151' : '#e5e7eb' }
}

const statusChartData = computed<ChartData<'line'>>(() => {
  const colors = chartColors()
  return {
    labels: trendPoints.value.map(point => formatDateTime(point.checked_at)),
    datasets: [{
      label: '健康状态',
      data: trendPoints.value.map(point => healthValue(point.status)),
      borderColor: '#16a34a',
      backgroundColor: 'rgba(22, 163, 74, 0.12)',
      pointBackgroundColor: trendPoints.value.map(point => point.status === 'failed' ? '#dc2626' : point.status === 'slow' ? '#d97706' : '#16a34a'),
      pointBorderColor: trendPoints.value.map(point => point.status === 'failed' ? '#dc2626' : point.status === 'slow' ? '#d97706' : '#16a34a'),
      tension: 0.25,
      fill: true,
    }],
    ...(colors && {}),
  }
})

const statusChartOptions = computed<ChartOptions<'line'>>(() => {
  const colors = chartColors()
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { intersect: false, mode: 'index' },
    plugins: {
      legend: { labels: { color: colors.text, usePointStyle: true } },
      tooltip: {
        callbacks: {
          label: (context) => '健康状态：' + statusLabel(trendPoints.value[context.dataIndex]?.status),
          afterLabel: (context) => {
            const point = trendPoints.value[context.dataIndex]
            return point ? '响应时间：' + formatLatency(point.latency_ms) : ''
          },
        },
      },
    },
    scales: {
      x: { grid: { color: colors.grid }, ticks: { color: colors.text, maxRotation: 0, autoSkip: true } },
      y: {
        min: 0,
        max: 2,
        ticks: { color: colors.text, stepSize: 1, callback: value => value === 2 ? '可用' : value === 1 ? '慢响应' : '失败' },
        grid: { color: colors.grid },
      },
    },
  }
})

const latencyChartData = computed<ChartData<'line'>>(() => ({
  labels: trendPoints.value.map(point => formatDateTime(point.checked_at)),
  datasets: [
    {
      label: '响应时间',
      data: trendPoints.value.map(point => point.latency_ms === null ? null : point.latency_ms ?? null),
      borderColor: '#2563eb',
      backgroundColor: 'rgba(37, 99, 235, 0.1)',
      pointBackgroundColor: '#2563eb',
      tension: 0.25,
      spanGaps: false,
      fill: true,
    },
    {
      label: '慢响应阈值',
      data: trendPoints.value.map(point => point.latency_limit_ms > 0 ? point.latency_limit_ms : null),
      borderColor: '#d97706',
      borderDash: [6, 4],
      pointRadius: 0,
      tension: 0,
      fill: false,
    },
  ],
}))

const latencyChartOptions = computed<ChartOptions<'line'>>(() => {
  const colors = chartColors()
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { intersect: false, mode: 'index' },
    plugins: {
      legend: { labels: { color: colors.text, usePointStyle: true } },
      tooltip: {
        callbacks: {
          label: (context) => {
            if (context.datasetIndex === 1) return '慢响应阈值：' + formatLatency(Number(context.raw))
            const point = trendPoints.value[context.dataIndex]
            return '响应时间：' + formatLatency(point?.latency_ms === null ? null : point?.latency_ms)
          },
          afterLabel: (context) => {
            const point = trendPoints.value[context.dataIndex]
            return point?.status === 'failed' ? '失败原因：' + (point.reason || '—') : ''
          },
        },
      },
    },
    scales: {
      x: { grid: { color: colors.grid }, ticks: { color: colors.text, maxRotation: 0, autoSkip: true } },
      y: { beginAtZero: true, grid: { color: colors.grid }, ticks: { color: colors.text, callback: value => value + ' ms' } },
    },
  }
})

async function loadAccounts() {
  loading.value = true
  error.value = ''
  try {
    const result = await listSupplierAccountHealthAccounts({
      provider_id: providerId.value || undefined,
      platform: platform.value || undefined,
      search: search.value.trim() || undefined,
      health_status: healthStatus.value || undefined,
      page: page.value,
      page_size: pageSize,
    })
    accounts.value = result.items || []
    total.value = Number(result.total || 0)
    lastLoadedAt.value = new Date().toISOString()
    healthTrendByAccountId.value = {}
    trendLoadingByAccountId.value = {}
    syncSelectedAccount()
    void loadAccountTrends(accounts.value, selectedRange.value)
  } catch (err) {
    error.value = extractApiErrorMessage(err, '加载账号健康列表失败')
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function loadAccountTrends(accountList: SupplierAccountHealthAccount[], range: SupplierAccountHealthRange) {
  const requestSequence = ++accountTrendLoadSequence
  const accountIds = accountList.map(account => account.local_account_id)
  const selectedId = selectedAccountId.value
  const ids = selectedId && accountIds.includes(selectedId)
    ? [selectedId, ...accountIds.filter(accountId => accountId !== selectedId)]
    : accountIds
  trendLoadingByAccountId.value = Object.fromEntries(ids.map(accountId => [accountId, true])) as Record<number, boolean>
  trendLoading.value = true
  let nextIndex = 0
  const loadNextTrend = async () => {
    while (nextIndex < ids.length) {
      if (requestSequence !== accountTrendLoadSequence) return
      const accountId = ids[nextIndex++]
      try {
        const result = await loadTrendRequest(accountId, range)
        if (requestSequence === accountTrendLoadSequence) {
          const points = result.points || []
          healthTrendByAccountId.value = { ...healthTrendByAccountId.value, [accountId]: points }
          if (selectedAccountId.value === accountId && selectedRange.value === range) trendPoints.value = points
        }
      } catch {
        if (requestSequence === accountTrendLoadSequence) {
          healthTrendByAccountId.value = { ...healthTrendByAccountId.value, [accountId]: [] }
        }
      } finally {
        if (requestSequence === accountTrendLoadSequence) {
          trendLoadingByAccountId.value = { ...trendLoadingByAccountId.value, [accountId]: false }
        }
      }
    }
  }
  const workerCount = Math.min(TREND_LOAD_CONCURRENCY, ids.length)
  await Promise.all(Array.from({ length: workerCount }, () => loadNextTrend()))
  if (requestSequence === accountTrendLoadSequence) trendLoading.value = false
}

async function loadTrend() {
  const requestSequence = ++trendLoadSequence
  const accountId = selectedAccountId.value
  const range = selectedRange.value
  if (!accountId) {
    trendPoints.value = []
    trendLoading.value = false
    return
  }
  if (Object.prototype.hasOwnProperty.call(healthTrendByAccountId.value, accountId)) {
    trendPoints.value = healthTrendByAccountId.value[accountId] || []
    trendLoading.value = false
    return
  }
  trendLoading.value = true
  error.value = ''
  try {
    const result = await loadTrendRequest(accountId, range)
    if (requestSequence === trendLoadSequence) {
      const points = result.points || []
      healthTrendByAccountId.value = { ...healthTrendByAccountId.value, [accountId]: points }
      trendPoints.value = points
    }
  } catch (err) {
    if (requestSequence === trendLoadSequence) {
      trendPoints.value = []
      error.value = extractApiErrorMessage(err, '加载账号健康趋势失败')
      appStore.showError(error.value)
    }
  } finally {
    if (requestSequence === trendLoadSequence) {
      trendLoading.value = false
    }
  }
}

function syncSelectedAccount() {
  const queryAccountId = Number(route.query.account_id)
  if (Number.isInteger(queryAccountId) && queryAccountId > 0 && accounts.value.some(account => account.local_account_id === queryAccountId)) {
    selectedAccountId.value = queryAccountId
    return
  }
  if (!selectedAccountId.value || !accounts.value.some(account => account.local_account_id === selectedAccountId.value)) {
    selectedAccountId.value = accounts.value[0]?.local_account_id || null
  }
}

function selectAccount(accountId: number) {
  selectedAccountId.value = accountId
  void router.replace({ query: { ...route.query, account_id: String(accountId) } })
}

function selectRange(range: SupplierAccountHealthRange) {
  if (selectedRange.value === range) return
  selectedRange.value = range
  trendPoints.value = []
  healthTrendByAccountId.value = {}
  void loadAccountTrends(accounts.value, range)
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  void loadAccounts()
}

async function refresh() {
  await loadAccounts()
}

watch([search, providerId, platform, healthStatus], () => {
  page.value = 1
  void loadAccounts()
})
watch(selectedAccountId, () => {
  void loadTrend()
})
watch(() => route.query.account_id, () => {
  syncSelectedAccount()
})
onMounted(() => {
  void loadAccounts()
})
</script>

<style scoped>
.sp-health-head { align-items: center; }
.sp-health-loaded { margin: 0; padding: 0.45rem 0.65rem; border-left: 0; }
.sp-health-filter-panel,
.sp-health-account-panel,
.sp-health-event-panel { margin-bottom: 1rem; }
.sp-health-filters { display: grid; grid-template-columns: minmax(10rem, 1fr) minmax(10rem, 1fr) minmax(14rem, 2fr) minmax(10rem, 1fr); gap: 0.75rem; padding: 0 1rem 1rem; }
.sp-health-filter-control { min-width: 0; }
.sp-health-account-button { display: grid; gap: 0.15rem; padding: 0; border: 0; background: transparent; color: var(--sp-text); text-align: left; cursor: pointer; }
.sp-health-account-button strong { font-size: 0.875rem; }
.sp-health-account-button span { color: var(--sp-muted); font-size: 0.72rem; }
.sp-health-account-button.active strong { color: var(--sp-cyan); }
.sp-health-failure-count { color: var(--sp-red); }
.sp-health-latency { font-variant-numeric: tabular-nums; }
.sp-health-pagination { display: flex; justify-content: flex-end; padding: 0.75rem 1rem 1rem; }
.sp-health-empty,
.sp-health-no-history,
.sp-health-chart-state { display: grid; min-height: 10rem; place-items: center; align-content: center; gap: 0.4rem; padding: 2rem; text-align: center; }
.sp-health-empty span,
.sp-health-no-history span { color: var(--sp-muted); font-size: 0.8125rem; }
.sp-health-detail-grid { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr); gap: 1rem; margin-bottom: 1rem; }
.sp-health-summary-panel { grid-row: span 2; }
.sp-health-kpis { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.5rem; padding: 0 1rem 1rem; }
.sp-health-kpis .sp-chart-kpi { min-height: 4.5rem; }
.sp-health-kpis .sp-chart-kpi b.good { color: var(--sp-green); }
.sp-health-kpis .sp-chart-kpi b.warn { color: var(--sp-amber); }
.sp-health-kpis .sp-chart-kpi b.bad { color: var(--sp-red); }
.sp-health-latest { display: grid; gap: 0.625rem; padding: 0 1rem 1rem; }
.sp-health-latest > div { display: grid; grid-template-columns: 5rem minmax(0, 1fr); gap: 0.75rem; padding: 0.625rem 0.75rem; border: 1px solid var(--sp-soft); background: var(--sp-panel-2); }
.sp-health-latest span { color: var(--sp-muted); font-size: 0.75rem; }
.sp-health-latest strong { overflow-wrap: anywhere; font-size: 0.8125rem; font-weight: 500; }
.sp-health-chart-panel { min-width: 0; }
.sp-health-chart { height: 18rem; padding: 0 1rem 1rem; }
.sp-health-chart-state { min-height: 18rem; color: var(--sp-muted); }
.sp-health-event-panel { overflow: hidden; }
.sp-health-events { display: grid; gap: 0.625rem; padding: 0 1rem 1rem; }
.sp-health-event { padding: 0.75rem; border: 1px solid var(--sp-soft); background: var(--sp-panel-2); }
.sp-health-event-head { display: flex; align-items: center; flex-wrap: wrap; gap: 0.625rem; }
.sp-health-event-head time { color: var(--sp-muted); font-size: 0.75rem; }
.sp-health-event-head strong { margin-left: auto; font-size: 0.8125rem; font-variant-numeric: tabular-nums; }
.sp-health-event dl { display: grid; gap: 0.35rem; margin: 0.65rem 0 0; }
.sp-health-event dl > div { display: grid; grid-template-columns: 4.5rem minmax(0, 1fr); gap: 0.75rem; }
.sp-health-event dt { color: var(--sp-muted); font-size: 0.72rem; }
.sp-health-event dd { margin: 0; overflow-wrap: anywhere; font-size: 0.78rem; }
.sp-health-account-cell { display: grid; gap: 0.35rem; min-width: 15rem; }
.sp-health-account-tags { display: flex; flex-wrap: wrap; gap: 0.3rem; }
.sp-health-chip { display: inline-flex; align-items: center; min-height: 1.25rem; padding: 0.1rem 0.45rem; border-radius: 999px; font-size: 0.68rem; font-weight: 600; line-height: 1.1; }
.sp-health-chip--provider.provider-0 { color: #2563eb; background: #eff6ff; }
.sp-health-chip--provider.provider-1 { color: #7c3aed; background: #f5f3ff; }
.sp-health-chip--provider.provider-2 { color: #c2410c; background: #fff7ed; }
.sp-health-chip--provider.provider-3 { color: #047857; background: #ecfdf5; }
.sp-health-chip--provider.provider-4 { color: #be185d; background: #fdf2f8; }
.sp-health-chip--platform.platform-0 { color: #0f766e; background: #f0fdfa; }
.sp-health-chip--platform.platform-1 { color: #4338ca; background: #eef2ff; }
.sp-health-chip--platform.platform-2 { color: #b45309; background: #fffbeb; }
.sp-health-chip--platform.platform-3 { color: #15803d; background: #f0fdf4; }
.sp-health-chip--platform.platform-4 { color: #0369a1; background: #f0f9ff; }
.sp-health-trend-cell { display: grid; min-width: 22rem; gap: 0.4rem; padding: 0.45rem 0.55rem 0.35rem; border: 1px solid var(--sp-soft, #e5e7eb); border-radius: 0.5rem; background: linear-gradient(180deg, color-mix(in srgb, var(--sp-panel-2, #fff) 92%, var(--sp-cyan, #0891b2) 8%), var(--sp-panel-2, #fff)); }
.sp-health-trend-meta { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; color: var(--sp-muted); font-size: 0.7rem; line-height: 1; }
.sp-health-trend-meta span { color: var(--sp-text); font-variant-numeric: tabular-nums; }
.sp-health-trend-meta time { font-variant-numeric: tabular-nums; }
.sp-health-trend-bars { display: flex; align-items: end; gap: 0.2rem; height: 6.5rem; padding: 0.35rem 0; border-bottom: 1px solid var(--sp-soft, #d1d5db); background: repeating-linear-gradient(to top, transparent 0, transparent calc(33.333% - 1px), color-mix(in srgb, var(--sp-soft, #d1d5db) 72%, transparent) calc(33.333% - 1px), color-mix(in srgb, var(--sp-soft, #d1d5db) 72%, transparent) 33.333%); }
.sp-health-trend-bar { display: block; flex: 1 1 0; min-width: 0.32rem; max-width: 0.8rem; min-height: 0.5rem; border-radius: 0.2rem 0.2rem 0.08rem 0.08rem; box-shadow: inset 0 1px rgb(255 255 255 / 25%); transition: height 160ms ease, opacity 160ms ease; }
.sp-health-trend-bar:hover { opacity: 0.72; }
.sp-health-trend-bar--good { background: var(--sp-green, #16a34a); }
.sp-health-trend-bar--warn { background: var(--sp-amber, #d97706); }
.sp-health-trend-bar--bad { background: var(--sp-red, #dc2626); }
.sp-health-trend-bar--info { background: #9ca3af; }
.sp-health-trend-bar--loading { height: 45%; background: var(--sp-soft, #d1d5db); animation: sp-health-trend-pulse 1.1s ease-in-out infinite alternate; }
.sp-health-trend-bar--loading:nth-child(2n) { height: 70%; animation-delay: 120ms; }
.sp-health-trend-bar--loading:nth-child(3n) { height: 30%; animation-delay: 220ms; }
.sp-health-trend-empty { color: var(--sp-muted); font-size: 0.75rem; }
@keyframes sp-health-trend-pulse { from { opacity: 0.45; } to { opacity: 1; } }
@media (max-width: 900px) {
  .sp-health-head { align-items: flex-start; }
  .sp-health-filters { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .sp-health-detail-grid { grid-template-columns: 1fr; }
  .sp-health-summary-panel { grid-row: auto; }
}
@media (max-width: 560px) {
  .sp-health-filters { grid-template-columns: 1fr; }
  .sp-health-kpis { grid-template-columns: 1fr 1fr; }
  .sp-health-latest > div,
  .sp-health-event dl > div { grid-template-columns: 1fr; gap: 0.2rem; }
  .sp-health-chart { height: 15rem; }
}
</style>
