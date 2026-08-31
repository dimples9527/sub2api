<template>
  <SupplierModuleLayout>
    <header class="sp-page-head sp-health-head">
      <div>
        <div class="sp-eyebrow">供应商运营 / 账号健康</div>
        <h1>账号健康趋势</h1>
        <p class="sp-subtitle">按账号查看健康状态与响应时间变化，快速定位慢响应和连续失败。</p>
      </div>
      <div class="sp-controls">
        <span v-if="lastLoadedAt" class="sp-health-loaded"><i aria-hidden="true"></i>更新于 {{ formatDateTime(lastLoadedAt) }}</span>
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

    <section class="sp-metric-grid sp-health-metrics" aria-label="账号健康概览">
      <button
        v-for="metric in summaryMetrics"
        :key="metric.key || 'all'"
        type="button"
        class="sp-metric-card"
        :class="[`sp-${metric.tone}`, { selected: healthStatus === metric.key }]"
        :aria-pressed="healthStatus === metric.key"
        @click="selectHealthStatus(metric.key)"
      >
        <div class="sp-metric-label">{{ metric.label }}</div>
        <div class="sp-metric-value">{{ metric.value }}</div>
        <div class="sp-metric-foot">{{ metric.foot }}</div>
      </button>
    </section>

    <section class="sp-panel sp-health-account-panel">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <div>
            <span class="sp-panel-kicker">Accounts</span>
            <h2>账号列表</h2>
            <p>筛选账号后查看健康状态和响应时间趋势。</p>
          </div>
        </div>
        <div class="sp-health-head-meta">
          <span v-if="guardTaskDisabled" class="sp-health-guard-warning">
            <span class="sp-status warn">健康守护任务未启用</span>
            <router-link to="/admin/supplier-management/automations">前往开启</router-link>
          </span>
          <span class="sp-status info">{{ total }} 个账号</span>
        </div>
      </header>
      <div class="sp-health-filters" role="group" aria-label="账号健康筛选">
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
      <DataTable :columns="accountColumns" :data="accountHealthSortData" :loading="loading" row-key="local_account_id">
        <template #cell-account_sort="{ row: account }">
          <div class="sp-health-account-cell">
            <button
              class="sp-health-account-button"
              type="button"
              :class="{ active: selectedAccountId === account.local_account_id }"
              @click="openHealthDetail(account.local_account_id)"
            >
              <strong>{{ account.local_account_name || ('账号 #' + account.local_account_id) }}</strong>
              <span>ID {{ account.local_account_id }}</span>
            </button>
            <div class="sp-health-account-tags">
              <span :class="['sp-health-chip', 'sp-health-chip--provider', providerTone(account.provider_name)]">
                {{ account.provider_name || '未关联供应商' }}
              </span>
              <span :class="['sp-health-chip', 'sp-health-chip--platform', platformTone(account.platform)]">
                {{ platformDisplayLabel(account.platform) }}
              </span>
            </div>
          </div>
        </template>
        <template #cell-upstream_rate_multiplier="{ row: account }">
          <span class="sp-health-value" :class="rateMultiplierTone(account.upstream_rate_multiplier)">
            {{ formatAccountRateMultiplier(account.upstream_rate_multiplier) }}
          </span>
        </template>
        <template #cell-effective_rate_multiplier="{ row: account }">
          <span class="sp-health-value" :class="rateMultiplierTone(account.effective_rate_multiplier)">
            {{ formatAccountRateMultiplier(account.effective_rate_multiplier) }}
          </span>
        </template>
        <template #cell-status_sort="{ row: account }">
          <span class="sp-status" :class="statusTone(account.status)">{{ statusLabel(account.status) }}</span>
          <div v-if="account.consecutive_failures > 0" class="sp-health-failure-count">
            <span class="sp-status bad">连续失败 {{ account.consecutive_failures }} 次</span>
          </div>
        </template>
        <template #cell-latency_ms="{ row: account }">
          <span class="sp-health-value" :class="latencyTone(account)">
            {{ formatLatency(account.latency_ms) }}
          </span>
        </template>
        <template #cell-checked_at_sort="{ row: account }">
          <span class="sp-health-value" :class="checkedAtTone(account.checked_at)">
            {{ account.checked_at ? formatDateTime(account.checked_at) : '尚未检测' }}
          </span>
        </template>
        <template #cell-health_trend_sort="{ row: account }">
          <div class="sp-health-trend-cell" :title="accountTrendTitle(account)">
            <div v-if="visibleAccountTrend(account.local_account_id).length" class="sp-health-trend-meta">
              <span>{{ formatTrendHealthRate(account.local_account_id) }}</span>
              <em v-if="account.latency_limit_ms > 0">阈值 {{ account.latency_limit_ms }} ms</em>
              <time>{{ trendLatestTime(account.local_account_id) }}</time>
            </div>
            <div v-if="trendLoadingByAccountId[account.local_account_id]" class="sp-health-trend-bars sp-health-trend-bars--loading" aria-label="正在加载健康趋势">
              <span v-for="index in TREND_BAR_COUNT" :key="index" class="sp-health-trend-bar sp-health-trend-bar--loading" />
            </div>
            <div
              v-else-if="visibleAccountTrend(account.local_account_id).length"
              :class="['sp-health-trend-bars', { 'sp-health-trend-bars--threshold': account.latency_limit_ms > 0 }]"
              aria-label="账号健康趋势"
            >
              <span
                v-for="(point, index) in visibleAccountTrend(account.local_account_id)"
                :key="`${point.checked_at}-${index}`"
                :class="['sp-health-trend-bar', `sp-health-trend-bar--${statusTone(point.status)}`]"
                :style="{ height: latencyBarHeight(point, account) }"
                :title="accountTrendPointTitle(point)"
              />
            </div>
            <span v-else-if="!visibleUpstreamTrend(account.local_account_id).length" class="sp-health-trend-empty">暂无趋势</span>
            <div
              v-if="visibleUpstreamTrend(account.local_account_id).length"
              class="sp-health-upstream-strip"
              aria-label="上游监控状态"
            >
              <span
                v-for="(point, index) in visibleUpstreamTrend(account.local_account_id)"
                :key="`${point.checked_at}-${index}`"
                :class="['sp-health-upstream-cell', `sp-health-upstream-cell--${statusTone(point.status)}`]"
                :title="upstreamTrendPointTitle(point)"
              />
            </div>
          </div>
        </template>
        <template #cell-actions="{ row: account }">
          <button class="sp-button sp-health-detail-button" type="button" @click="openHealthDetail(account.local_account_id)">
            查看详情
          </button>
        </template>
        <template #empty>
          <div class="sp-empty-state sp-health-empty">
            <template v-if="error">
              <strong>账号健康列表加载失败</strong>
              <span>{{ error }}</span>
              <button class="sp-button" type="button" :disabled="loading" @click="refresh">重试</button>
            </template>
            <template v-else>
              <strong>暂无可展示的账号</strong>
              <span>请调整筛选条件，或先启用供应商账号健康守护任务。</span>
            </template>
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

    <BaseDialog :show="healthDetailVisible" title="账号健康详情" width="full" @close="closeHealthDetail">
      <div class="sp-health-detail-dialog supplier-management-page">
        <div v-if="selectedAccount" class="sp-health-detail-grid">
          <article class="sp-panel sp-health-summary-panel">
            <header class="sp-panel-head">
              <div class="sp-panel-title">
                <div>
                  <span class="sp-panel-kicker">Selected Account</span>
                  <h2>{{ selectedAccount.local_account_name || ('账号 #' + selectedAccount.local_account_id) }}</h2>
                  <p>{{ selectedAccount.provider_name || '—' }} · {{ platformDisplayLabel(selectedAccount.platform) }} · ID {{ selectedAccount.local_account_id }}</p>
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
                <b>{{ trendSampleCount }}</b>
              </div>
            </div>
            <div v-if="!trendLoading && !hasAnyTrendSamples" class="sp-empty-state sp-health-no-history">
              <strong>尚无健康检测记录</strong>
              <span>{{ selectedAccount.guard_enabled ? '健康守护运行后会在这里生成趋势记录。' : '健康守护任务未启用，不会产生趋势记录。' }}</span>
            </div>
            <div v-else-if="hasTrendSamples" class="sp-health-latest">
              <div><span>最近检测</span><strong>{{ latestPoint ? formatDateTime(latestPoint.checked_at) : '—' }}</strong></div>
              <div><span>失败原因</span><strong>{{ latestPoint?.reason || '—' }}</strong></div>
              <div><span>动作</span><strong>{{ latestPoint?.action || '—' }}</strong></div>
              <div class="sp-health-latest-error"><span>错误详情</span><strong>{{ latestPoint?.error_message || '—' }}</strong></div>
            </div>
            <div v-if="upstreamMonitors.length" class="sp-health-upstream">
              <div class="sp-health-upstream-head">
                <span>上游监控</span>
                <span v-if="upstreamLatestPoint" class="sp-status" :class="statusTone(upstreamLatestPoint.status)">{{ statusLabel(upstreamLatestPoint.status) }}</span>
                <em v-else>所选范围内暂无上报</em>
              </div>
              <div v-for="monitor in upstreamMonitors" :key="monitor.target_id" class="sp-health-upstream-item">
                <strong>{{ monitor.monitor_name || monitor.monitor_key }}</strong>
                <b>7 天可用率 {{ formatAvailability(monitor.availability_7d) }}</b>
                <span>{{ monitor.provider_name || '—' }} · {{ monitor.primary_model || '未标注主模型' }}</span>
                <time>最近上报 {{ formatDateTime(monitor.last_seen_at) }}</time>
              </div>
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
            <div v-else-if="hasAnyTrendSamples" class="sp-health-chart"><Line :data="statusChartData" :options="statusChartOptions" /></div>
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
            <div v-else-if="hasAnyTrendSamples" class="sp-health-chart"><Line :data="latencyChartData" :options="latencyChartOptions" /></div>
            <div v-else class="sp-health-chart-state">尚无健康检测记录</div>
          </article>

          <section v-if="detailEventPoints.length" class="sp-panel sp-health-event-panel">
            <header class="sp-panel-head">
              <div class="sp-panel-title">
                <div>
                  <span class="sp-panel-kicker">Diagnostics</span>
                  <h2>检测明细</h2>
                  <p>{{ detailEventHint }}</p>
                </div>
              </div>
              <span class="sp-status info">{{ trendSampleCount }} 次检测</span>
            </header>
            <div class="sp-health-events">
              <article
                v-for="point in detailEventPoints"
                :key="point.checked_at + '-' + point.status"
                :class="['sp-health-event', `sp-health-event--${statusTone(point.status)}`]"
              >
                <div class="sp-health-event-head">
                  <span class="sp-status" :class="statusTone(point.status)">{{ statusLabel(point.status) }}</span>
                  <time>{{ trendPointLabel(point) }}</time>
                  <strong>{{ formatLatency(point.latency_ms) }}</strong>
                </div>
                <dl>
                  <div><dt>失败原因</dt><dd>{{ point.reason || '—' }}</dd></div>
                  <div><dt>动作</dt><dd>{{ point.action || '—' }}</dd></div>
                  <div class="sp-health-event-error"><dt>错误详情</dt><dd>{{ point.error_message || '—' }}</dd></div>
                </dl>
              </article>
            </div>
          </section>
        </div>
      </div>
    </BaseDialog>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDebounceFn } from '@vueuse/core'
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
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { createKeyedRequestLoader } from '@/composables/useKeyedRequestLoader'
import {
  getSupplierAccountHealthSummary,
  getSupplierAccountHealthTrend,
  getSupplierAccountHealthTrends,
  listSupplierAccountHealthAccounts,
  type SupplierAccountHealthAccount,
  type SupplierAccountHealthPoint,
  type SupplierAccountHealthRange,
  type SupplierAccountHealthSummary,
  type SupplierAccountHealthTrend,
  type SupplierAccountHealthUpstreamMonitor,
} from '@/api/admin/supplierAccountHealth'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { customPlatformsAPI, type CustomPlatform } from '@/api/admin/customPlatforms'
import { buildPlatformOptions, loadPlatformCatalog } from '@/utils/platformOptions'
import { resolvePlatformDisplayLabel, setCustomPlatformLabels } from '@/utils/customPlatformLabels'
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
const summary = ref<SupplierAccountHealthSummary>({ total: 0, healthy: 0, slow: 0, failed: 0, unchecked: 0 })
const providers = ref<SupplierProvider[]>([])
const customPlatforms = ref<CustomPlatform[]>([])
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
const healthDetailVisible = ref(false)
const trendPoints = ref<SupplierAccountHealthPoint[]>([])
const latestTrendPoint = ref<SupplierAccountHealthPoint | null>(null)
const upstreamTrendPoints = ref<SupplierAccountHealthPoint[]>([])
const upstreamLatestPoint = ref<SupplierAccountHealthPoint | null>(null)
const upstreamMonitors = ref<SupplierAccountHealthUpstreamMonitor[]>([])
const healthTrendByAccountId = ref<Record<number, SupplierAccountHealthPoint[]>>({})
const upstreamTrendByAccountId = ref<Record<number, SupplierAccountHealthPoint[]>>({})
const detailTrendCache = ref<Record<string, SupplierAccountHealthTrend>>({})
const trendLoadingByAccountId = ref<Record<number, boolean>>({})
const lastLoadedAt = ref('')
const TREND_BAR_COUNT = 96
const TREND_THRESHOLD_RATIO = 0.6
const DETAIL_EVENT_LIMIT = 50
// 上游监控是与本地健康守护并列的第二个来源，两条曲线共用同一批时间桶，靠数据集名区分
const UPSTREAM_SERIES_LABEL = '上游监控'
const LATENCY_THRESHOLD_LABEL = '慢响应阈值'
// 倍率高于 2 视为明显偏贵；响应超过阈值 75% 视为接近超时；检测超过一天视为守护失效
const RATE_BAD_THRESHOLD = 2
const LATENCY_WARN_RATIO = 0.75
const CHECKED_FRESH_MS = 60 * 60 * 1000
const CHECKED_STALE_MS = 24 * 60 * 60 * 1000
let trendLoadSequence = 0
let accountTrendLoadSequence = 0

const loadTrendRequest = createKeyedRequestLoader(
  getSupplierAccountHealthTrend,
  (accountId, range) => `${accountId}:${range}`,
)

type HealthValueTone = 'good' | 'warn' | 'bad' | 'muted'

type AccountHealthSortAccount = SupplierAccountHealthAccount & {
  account_sort: string
  status_sort: number
  checked_at_sort: number | null
  health_trend_sort: number
}

const accountColumns: Column[] = [
  { key: 'account_sort', label: '账号 / 供应商 / 平台', sortable: true },
  { key: 'upstream_rate_multiplier', label: '上游倍率', sortable: true, class: 'min-w-[88px]' },
  { key: 'effective_rate_multiplier', label: '有效倍率', sortable: true, class: 'min-w-[88px]' },
  { key: 'status_sort', label: '当前健康状态', sortable: true },
  { key: 'latency_ms', label: '最近响应', sortable: true, class: 'min-w-[96px]' },
  { key: 'checked_at_sort', label: '检测时间', sortable: true, class: 'min-w-[132px]' },
  { key: 'health_trend_sort', label: '健康趋势', sortable: true },
  { key: 'actions', label: '操作', class: 'min-w-[96px]' },
]

const accountHealthSortData = computed<AccountHealthSortAccount[]>(() => accounts.value.map((account) => ({
  ...account,
  account_sort: `${account.local_account_name || `账号 #${account.local_account_id}`}:${account.local_account_id}`,
  status_sort: statusSortValue(account.status),
  checked_at_sort: checkedAtSortValue(account.checked_at),
  health_trend_sort: accountTrendHealthRateSortValue(account.local_account_id),
})))

const providerOptions = computed<SelectOption[]>(() => [
  { value: null, label: '全部供应商' },
  ...providers.value.filter(provider => provider.enabled).map(provider => ({ value: provider.id, label: provider.name })),
])
const platformOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部平台' },
  ...buildPlatformOptions(customPlatforms.value),
])
const healthStatusOptions: SelectOption[] = [
  { value: '', label: '全部健康状态' },
  { value: 'healthy', label: '可用' },
  { value: 'slow', label: '慢响应' },
  { value: 'failed', label: '失败' },
  { value: 'unchecked', label: '未检测' },
]
const summaryMetrics = computed(() => [
  { key: '', tone: 'blue', label: '账号总数', value: summary.value.total, foot: '当前筛选范围内的供应商账号' },
  { key: 'healthy', tone: 'green', label: '可用', value: summary.value.healthy, foot: '最近一次检测在阈值内通过' },
  { key: 'slow', tone: 'amber', label: '慢响应', value: summary.value.slow, foot: '最近一次检测超过慢响应阈值' },
  { key: 'failed', tone: 'red', label: '失败', value: summary.value.failed, foot: '最近一次检测未通过' },
  { key: 'unchecked', tone: 'muted-tone', label: '未检测', value: summary.value.unchecked, foot: '还没有健康检测记录' },
])

const selectedAccount = computed(() => accounts.value.find(account => account.local_account_id === selectedAccountId.value) || null)
const latestPoint = computed(() => latestTrendPoint.value || [...trendPoints.value].reverse().find(point => point.sample_count > 0) || null)
const trendSampleCount = computed(() => trendPoints.value.reduce((total, point) => total + point.sample_count, 0))
const hasTrendSamples = computed(() => trendSampleCount.value > 0)
const hasUpstreamTrend = computed(() => upstreamTrendPoints.value.some(point => point.sample_count > 0))
// 只有上游数据的账号也要出图，否则会卡在「尚无健康检测记录」空态
const hasAnyTrendSamples = computed(() => hasTrendSamples.value || hasUpstreamTrend.value)
const detailEventPoints = computed(() => trendPoints.value.filter(point => point.sample_count > 0).reverse().slice(0, DETAIL_EVENT_LIMIT))
const detailEventHint = computed(() => (trendPoints.value.filter(point => point.sample_count > 0).length > DETAIL_EVENT_LIMIT
  ? `按时间桶倒序展示最近 ${DETAIL_EVENT_LIMIT} 个时间桶的状态、失败原因、动作和错误详情。`
  : '查看每个有检测样本的时间桶状态、失败原因、动作和错误详情。'))
// guard_enabled 取自全局健康守护任务开关，所有账号取值一致，因此不按账号计数
const guardTaskDisabled = computed(() => accounts.value.length > 0 && !accounts.value.some(account => account.guard_enabled))

function platformDisplayLabel(platform?: string | null): string {
  return platform ? resolvePlatformDisplayLabel(platform) : '未知平台'
}

function statusLabel(status?: string | null): string {
  if (status === 'healthy') return '可用'
  if (status === 'slow') return '慢响应'
  if (status === 'failed') return '失败'
  // unavailable 只来自上游样本，是上游没给出可解析状态时的兜底值，属于「不知道」而不是「挂了」
  if (status === 'unavailable') return '上游未上报'
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
  return healthTrendByAccountId.value[accountId] || []
}

function visibleUpstreamTrend(accountId: number): SupplierAccountHealthPoint[] {
  return upstreamTrendByAccountId.value[accountId] || []
}

function formatTrendHealthRate(accountId: number): string {
  const points = visibleAccountTrend(accountId)
  const sampleCount = points.reduce((total, point) => total + point.sample_count, 0)
  if (!sampleCount) return '健康率 —'
  const healthyCount = points.reduce((total, point) => total + point.healthy_count, 0)
  return `健康率 ${((healthyCount / sampleCount) * 100).toFixed(0)}%`
}

function trendLatestTime(accountId: number): string {
  const latest = [...visibleAccountTrend(accountId)].reverse().find(point => point.sample_count > 0)
  return latest ? formatDateTime(latest.latest_checked_at || latest.checked_at) : ''
}

function accountTrendTitle(account: SupplierAccountHealthAccount): string {
  const points = visibleAccountTrend(account.local_account_id)
  if (!points.length) return '暂无健康趋势数据'
  const sampleCount = points.reduce((total, point) => total + point.sample_count, 0)
  return `${account.local_account_name || `账号 #${account.local_account_id}`}：${sampleCount || '暂无'}次检测，${points.length}个时间桶`
}

function accountTrendPointTitle(point: SupplierAccountHealthPoint): string {
  const bucketEnd = point.bucket_end_at ? ` - ${formatDateTime(point.bucket_end_at)}` : ''
  if (!point.sample_count) return `${formatDateTime(point.checked_at)}${bucketEnd} 未检测`
  const threshold = point.latency_limit_ms > 0 ? `，阈值 ${point.latency_limit_ms} ms` : ''
  return `${formatDateTime(point.checked_at)}${bucketEnd} ${statusLabel(point.status)}，${point.sample_count}次检测，健康 ${point.healthy_count} 次，响应 ${formatLatency(point.latency_ms)}${threshold}`
}

function upstreamTrendPointTitle(point: SupplierAccountHealthPoint): string {
  const bucketEnd = point.bucket_end_at ? ` - ${formatDateTime(point.bucket_end_at)}` : ''
  if (!point.sample_count) return `${formatDateTime(point.checked_at)}${bucketEnd} 上游未上报`
  return `${formatDateTime(point.checked_at)}${bucketEnd} 上游 ${statusLabel(point.status)}，${point.sample_count} 个样本，响应 ${formatLatency(point.latency_ms)}`
}

// 优先按慢响应阈值归一，让阈值线固定在同一高度、不同账号之间可以横向对比；
// 没有配置阈值时退回按该账号自身最大响应时间归一。
function latencyBarHeight(point: SupplierAccountHealthPoint, account: SupplierAccountHealthAccount): string {
  const latency = point.latency_ms
  if (latency === null || latency === undefined || latency <= 0) return '14%'
  const limit = point.latency_limit_ms > 0 ? point.latency_limit_ms : account.latency_limit_ms
  if (limit > 0) return clampBarHeight((latency / limit) * TREND_THRESHOLD_RATIO * 100)
  const latencies = (healthTrendByAccountId.value[account.local_account_id] || [])
    .map(item => item.latency_ms)
    .filter((value): value is number => typeof value === 'number' && value > 0)
  const maxLatency = Math.max(...latencies, 0)
  if (maxLatency <= 0) return '14%'
  return clampBarHeight((latency / maxLatency) * 100)
}

function clampBarHeight(value: number): string {
  return `${Math.max(16, Math.min(100, Math.round(value)))}%`
}

function formatLatency(value?: number | null): string {
  if (value === null || value === undefined) return '—'
  return value + ' ms'
}

// availability_7d 是 0-100 的百分数，与监控绑定页的展示口径一致
function formatAvailability(value?: number | null): string {
  const rate = Number(value)
  if (!Number.isFinite(rate)) return '—'
  return `${rate.toFixed(2)}%`
}

function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function healthValue(status: string): number | null {
  if (status === 'healthy') return 2
  if (status === 'slow') return 1
  if (status === 'failed') return 0
  return null
}

function chartColors() {
  const dark = document.documentElement.classList.contains('dark')
  return { text: dark ? '#e5e7eb' : '#374151', grid: dark ? '#374151' : '#e5e7eb' }
}

function trendPointColor(status: string): string {
  if (status === 'failed') return '#dc2626'
  if (status === 'slow') return '#d97706'
  if (status === 'healthy') return '#16a34a'
  return '#9ca3af'
}

function trendPointLabel(point?: SupplierAccountHealthPoint): string {
  if (!point) return '—'
  const end = point.bucket_end_at ? ` - ${formatDateTime(point.bucket_end_at)}` : ''
  return `${formatDateTime(point.checked_at)}${end}`
}

// 两条序列共用同一批时间桶，因此按数据集名而不是下标取点，阈值线插在哪个位置都不影响
function seriesPointAt(label: string | undefined, index: number): SupplierAccountHealthPoint | undefined {
  return label === UPSTREAM_SERIES_LABEL ? upstreamTrendPoints.value[index] : trendPoints.value[index]
}

const statusChartData = computed<ChartData<'line'>>(() => {
  const colors = chartColors()
  const datasets: ChartData<'line'>['datasets'] = [{
    label: '健康状态',
    data: trendPoints.value.map(point => healthValue(point.status)),
    borderColor: '#16a34a',
    backgroundColor: 'rgba(22, 163, 74, 0.12)',
    pointBackgroundColor: trendPoints.value.map(point => trendPointColor(point.status)),
    pointBorderColor: trendPoints.value.map(point => trendPointColor(point.status)),
    tension: 0.25,
    fill: true,
  }]
  if (hasUpstreamTrend.value) {
    datasets.push({
      label: UPSTREAM_SERIES_LABEL,
      data: upstreamTrendPoints.value.map(point => healthValue(point.status)),
      borderColor: '#0891b2',
      borderDash: [5, 4],
      pointBackgroundColor: '#0891b2',
      pointBorderColor: '#0891b2',
      pointRadius: 2,
      tension: 0.25,
      fill: false,
    })
  }
  return {
    labels: trendPoints.value.map(point => formatDateTime(point.checked_at)),
    datasets,
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
          label: (context) => `${context.dataset.label}：${statusLabel(seriesPointAt(context.dataset.label, context.dataIndex)?.status)}`,
          afterLabel: (context) => {
            const point = seriesPointAt(context.dataset.label, context.dataIndex)
            if (!point) return ''
            return `${trendPointLabel(point)}，${point.sample_count} 次检测，健康 ${point.healthy_count} 次，响应 ${formatLatency(point.latency_ms)}`
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

const latencyChartData = computed<ChartData<'line'>>(() => {
  const datasets: ChartData<'line'>['datasets'] = [
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
      label: LATENCY_THRESHOLD_LABEL,
      data: trendPoints.value.map(point => point.latency_limit_ms > 0 ? point.latency_limit_ms : null),
      borderColor: '#d97706',
      borderDash: [6, 4],
      pointRadius: 0,
      tension: 0,
      fill: false,
    },
  ]
  if (hasUpstreamTrend.value) {
    datasets.push({
      label: UPSTREAM_SERIES_LABEL,
      data: upstreamTrendPoints.value.map(point => point.latency_ms === null ? null : point.latency_ms ?? null),
      borderColor: '#0891b2',
      borderDash: [5, 4],
      pointBackgroundColor: '#0891b2',
      pointRadius: 2,
      tension: 0.25,
      spanGaps: false,
      fill: false,
    })
  }
  return {
    labels: trendPoints.value.map(point => formatDateTime(point.checked_at)),
    datasets,
  }
})

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
            if (context.dataset.label === LATENCY_THRESHOLD_LABEL) return `${LATENCY_THRESHOLD_LABEL}：${formatLatency(Number(context.raw))}`
            const point = seriesPointAt(context.dataset.label, context.dataIndex)
            return `${context.dataset.label}：${formatLatency(point?.latency_ms === null ? null : point?.latency_ms)}`
          },
          afterLabel: (context) => {
            if (context.dataset.label === LATENCY_THRESHOLD_LABEL) return ''
            const point = seriesPointAt(context.dataset.label, context.dataIndex)
            if (!point) return ''
            const sampleHint = `${trendPointLabel(point)}，${point.sample_count} 次检测`
            return point.status === 'failed' ? `${sampleHint}；失败原因：${point.reason || '—'}` : sampleHint
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

async function loadFilterOptions() {
  try {
    const [providerResult, platformItems] = await Promise.all([
      supplierProvidersAPI.list({ page: 1, page_size: 200 }),
      customPlatformsAPI.list(),
      loadPlatformCatalog(),
    ])
    providers.value = providerResult.items || []
    customPlatforms.value = platformItems
    setCustomPlatformLabels(platformItems)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载筛选选项失败'))
  }
}

async function loadSummary() {
  try {
    summary.value = await getSupplierAccountHealthSummary({
      provider_id: providerId.value || undefined,
      platform: platform.value || undefined,
      search: search.value.trim() || undefined,
    })
  } catch (err) {
    summary.value = { total: 0, healthy: 0, slow: 0, failed: 0, unchecked: 0 }
    appStore.showError(extractApiErrorMessage(err, '加载账号健康概览失败'))
  }
}

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
    upstreamTrendByAccountId.value = {}
    trendLoadingByAccountId.value = {}
    detailTrendCache.value = {}
    latestTrendPoint.value = null
    syncSelectedAccount()
    void loadAccountTrends(accounts.value, selectedRange.value)
    void loadTrend()
  } catch (err) {
    accounts.value = []
    total.value = 0
    error.value = extractApiErrorMessage(err, '加载账号健康列表失败')
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function loadAccountTrends(accountList: SupplierAccountHealthAccount[], range: SupplierAccountHealthRange) {
  const requestSequence = ++accountTrendLoadSequence
  const ids = accountList.map(account => account.local_account_id)
  if (!ids.length) {
    trendLoadingByAccountId.value = {}
    return
  }
  trendLoadingByAccountId.value = Object.fromEntries(ids.map(accountId => [accountId, true])) as Record<number, boolean>
  try {
    const result = await getSupplierAccountHealthTrends(ids, range)
    if (requestSequence !== accountTrendLoadSequence) return
    const trendMap = new Map((result.items || []).map(trend => [trend.account_id, trend]))
    healthTrendByAccountId.value = Object.fromEntries(
      ids.map(accountId => [accountId, trendMap.get(accountId)?.points || []]),
    ) as Record<number, SupplierAccountHealthPoint[]>
    upstreamTrendByAccountId.value = Object.fromEntries(
      ids.map(accountId => [accountId, trendMap.get(accountId)?.upstream_points || []]),
    ) as Record<number, SupplierAccountHealthPoint[]>
  } catch (err) {
    if (requestSequence !== accountTrendLoadSequence) return
    healthTrendByAccountId.value = Object.fromEntries(
      ids.map(accountId => [accountId, []]),
    ) as Record<number, SupplierAccountHealthPoint[]>
    upstreamTrendByAccountId.value = Object.fromEntries(
      ids.map(accountId => [accountId, []]),
    ) as Record<number, SupplierAccountHealthPoint[]>
    appStore.showError(extractApiErrorMessage(err, '加载账号健康趋势失败'))
  } finally {
    if (requestSequence === accountTrendLoadSequence) {
      trendLoadingByAccountId.value = Object.fromEntries(ids.map(accountId => [accountId, false])) as Record<number, boolean>
    }
  }
}

function statusSortValue(status?: string | null): number {
  if (status === 'healthy') return 0
  if (status === 'slow') return 1
  if (status === 'failed') return 2
  return 3
}

// 返回 null 让 DataTable 把「尚未检测」的账号排到末尾
function checkedAtSortValue(checkedAt?: string | null): number | null {
  if (!checkedAt) return null
  const timestamp = Date.parse(checkedAt)
  return Number.isNaN(timestamp) ? null : timestamp
}

function rateMultiplierTone(value?: number | null): HealthValueTone {
  const rate = Number(value)
  if (!Number.isFinite(rate)) return 'muted'
  if (rate > RATE_BAD_THRESHOLD) return 'bad'
  if (rate > 1) return 'warn'
  return rate < 1 ? 'good' : 'muted'
}

function latencyTone(account: SupplierAccountHealthAccount): HealthValueTone {
  if (account.latency_ms === null || account.latency_ms === undefined) return 'muted'
  if (account.latency_limit_ms <= 0) return 'muted'
  if (account.latency_ms > account.latency_limit_ms) return 'bad'
  return account.latency_ms > account.latency_limit_ms * LATENCY_WARN_RATIO ? 'warn' : 'good'
}

// 「尚未检测」和「一天没再检测」都说明守护没覆盖到这个账号，用同一个警示色
function checkedAtTone(checkedAt?: string | null): HealthValueTone {
  const timestamp = checkedAtSortValue(checkedAt)
  if (timestamp === null) return 'warn'
  const age = Date.now() - timestamp
  if (age >= CHECKED_STALE_MS) return 'warn'
  return age <= CHECKED_FRESH_MS ? 'good' : 'muted'
}

function accountTrendHealthRateSortValue(accountId: number): number {
  const points = healthTrendByAccountId.value[accountId] || []
  const sampleCount = points.reduce((total, point) => total + point.sample_count, 0)
  if (!sampleCount) return -1
  const healthyCount = points.reduce((total, point) => total + point.healthy_count, 0)
  return healthyCount / sampleCount
}

function formatAccountRateMultiplier(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(Number(value))) return '—'
  return `× ${Number(value)}`
}

function resetDetailTrend() {
  trendPoints.value = []
  latestTrendPoint.value = null
  upstreamTrendPoints.value = []
  upstreamLatestPoint.value = null
  upstreamMonitors.value = []
}

function applyDetailTrend(trend: SupplierAccountHealthTrend) {
  trendPoints.value = trend.points || []
  latestTrendPoint.value = trend.latest || null
  upstreamTrendPoints.value = trend.upstream_points || []
  upstreamLatestPoint.value = trend.upstream_latest || null
  upstreamMonitors.value = trend.upstream_monitors || []
}

async function loadTrend() {
  const requestSequence = ++trendLoadSequence
  const accountId = selectedAccountId.value
  const range = selectedRange.value
  if (!accountId) {
    resetDetailTrend()
    trendLoading.value = false
    return
  }
  const cacheKey = `${accountId}:${range}`
  if (Object.prototype.hasOwnProperty.call(detailTrendCache.value, cacheKey)) {
    applyDetailTrend(detailTrendCache.value[cacheKey])
    trendLoading.value = false
    return
  }
  trendLoading.value = true
  error.value = ''
  try {
    const result = await loadTrendRequest(accountId, range)
    if (requestSequence === trendLoadSequence) {
      detailTrendCache.value = { ...detailTrendCache.value, [cacheKey]: result }
      applyDetailTrend(result)
    }
  } catch (err) {
    if (requestSequence === trendLoadSequence) {
      resetDetailTrend()
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
    healthDetailVisible.value = true
    return
  }
  if (selectedAccountId.value && !accounts.value.some(account => account.local_account_id === selectedAccountId.value)) {
    selectedAccountId.value = null
  }
}

function openHealthDetail(accountId: number) {
  selectedAccountId.value = accountId
  healthDetailVisible.value = true
  void router.replace({ query: { ...route.query, account_id: String(accountId) } })
}

function closeHealthDetail() {
  healthDetailVisible.value = false
  const query = { ...route.query }
  delete query.account_id
  void router.replace({ query })
}

function selectRange(range: SupplierAccountHealthRange) {
  if (selectedRange.value === range) return
  selectedRange.value = range
  resetDetailTrend()
  healthTrendByAccountId.value = {}
  upstreamTrendByAccountId.value = {}
  detailTrendCache.value = {}
  void loadAccountTrends(accounts.value, range)
  void loadTrend()
}

function selectHealthStatus(status: string) {
  healthStatus.value = healthStatus.value === status ? '' : status
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  void loadAccounts()
}

async function refresh() {
  void loadSummary()
  await loadAccounts()
}

const debouncedSearchReload = useDebounceFn(() => {
  page.value = 1
  void loadSummary()
  void loadAccounts()
}, 300)

// 概览卡按状态切换筛选，其数字本身不受状态筛选影响，因此只有供应商/平台/搜索变化才重新汇总
watch([providerId, platform], () => {
  page.value = 1
  void loadSummary()
  void loadAccounts()
})
watch(healthStatus, () => {
  page.value = 1
  void loadAccounts()
})
watch(search, () => {
  void debouncedSearchReload()
})
watch(selectedAccountId, () => {
  void loadTrend()
})
watch(() => route.query.account_id, () => {
  syncSelectedAccount()
})
onMounted(() => {
  void loadFilterOptions()
  void loadSummary()
  void loadAccounts()
})
</script>

<style scoped>
.sp-health-head { align-items: center; }
.sp-health-loaded { display: inline-flex; align-items: center; gap: 0.4rem; color: var(--sp-muted); font-size: 0.75rem; font-variant-numeric: tabular-nums; }
.sp-health-loaded i { width: 0.375rem; height: 0.375rem; border-radius: 50%; background: var(--sp-green); }
/* 概览卡用 button 承载点击筛选，配色沿用同模块的 --sp-metric-accent 强调色方案 */
.sp-health-metrics .sp-metric-card { --sp-metric-accent: var(--sp-blue); width: 100%; border-color: color-mix(in srgb, var(--sp-metric-accent) 24%, var(--sp-line)); background: linear-gradient(150deg, color-mix(in srgb, var(--sp-metric-accent) 7%, transparent), transparent 56%), var(--sp-panel); font: inherit; text-align: left; }
.sp-health-metrics .sp-metric-card::before { content: ''; position: absolute; inset: 0 0 auto 0; height: 3px; background: linear-gradient(90deg, var(--sp-metric-accent), color-mix(in srgb, var(--sp-metric-accent) 30%, transparent)); }
.sp-health-metrics .sp-metric-card.sp-green { --sp-metric-accent: var(--sp-green); }
.sp-health-metrics .sp-metric-card.sp-amber { --sp-metric-accent: var(--sp-amber); }
.sp-health-metrics .sp-metric-card.sp-red { --sp-metric-accent: var(--sp-red); }
.sp-health-metrics .sp-metric-card.sp-muted-tone { --sp-metric-accent: var(--sp-muted); }
.sp-health-metrics .sp-metric-card:hover,
.sp-health-metrics .sp-metric-card.selected { border-color: color-mix(in srgb, var(--sp-metric-accent) 55%, var(--sp-line)); box-shadow: 0 0 0 1px color-mix(in srgb, var(--sp-metric-accent) 30%, transparent); }
.sp-health-metrics .sp-metric-value { color: var(--sp-metric-accent); font-variant-numeric: tabular-nums; letter-spacing: -0.02em; }
.sp-panel-kicker { color: var(--sp-muted); font-size: 0.625rem; font-weight: 800; letter-spacing: 0.12em; text-transform: uppercase; }
.sp-panel-title p { margin: 0.25rem 0 0; color: var(--sp-muted); font-size: 0.75rem; line-height: 1.5; }
.sp-health-account-panel,
.sp-health-event-panel { margin-bottom: 1rem; }
.sp-health-head-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 0.5rem; }
.sp-health-filters { display: grid; grid-template-columns: minmax(10rem, 1fr) minmax(10rem, 1fr) minmax(14rem, 2fr) minmax(10rem, 1fr); gap: 0.75rem; padding: 0.75rem 1rem; border-bottom: 1px solid var(--sp-soft); background: var(--sp-panel-2); }
.sp-health-filter-control { min-width: 0; }
.sp-health-account-button { display: grid; gap: 0.15rem; padding: 0; border: 0; background: transparent; color: var(--sp-text); text-align: left; cursor: pointer; }
.sp-health-account-button strong { font-size: 0.875rem; }
.sp-health-account-button span { color: var(--sp-muted); font-size: 0.72rem; }
.sp-health-account-button.active strong { color: var(--sp-cyan); }
.sp-health-failure-count { margin-top: 0.3rem; }
.sp-health-failure-count .sp-status { padding: 0.1rem 0.45rem; font-size: 0.68rem; }
.sp-health-value { font-variant-numeric: tabular-nums; font-weight: 600; }
.sp-health-value.good { color: var(--sp-green); }
.sp-health-value.warn { color: var(--sp-amber); }
.sp-health-value.bad { color: var(--sp-red); }
.sp-health-value.muted { color: var(--sp-muted); font-weight: 500; }
.dark .sp-health-value.good { color: color-mix(in srgb, var(--sp-green) 55%, #ffffff); }
.dark .sp-health-value.warn { color: color-mix(in srgb, var(--sp-amber) 55%, #ffffff); }
.dark .sp-health-value.bad { color: color-mix(in srgb, var(--sp-red) 55%, #ffffff); }
.sp-health-pagination { display: flex; justify-content: flex-end; padding: 0.75rem 1rem 1rem; }
.sp-health-guard-warning { display: inline-flex; align-items: center; gap: 0.4rem; font-size: 0.75rem; }
.sp-health-guard-warning a { color: var(--sp-cyan); text-decoration: underline; text-underline-offset: 0.15rem; }
.sp-health-empty,
.sp-health-no-history,
.sp-health-chart-state { display: grid; min-height: 10rem; place-items: center; align-content: center; gap: 0.4rem; padding: 2rem; text-align: center; }
.sp-health-empty span,
.sp-health-no-history span { color: var(--sp-muted); font-size: 0.8125rem; }
.sp-health-empty .sp-button { margin-top: 0.35rem; }
/* 弹窗根节点复用 supplier-management-page 提供 --sp-* 变量，并抵消其 min-height 副作用 */
.sp-health-detail-dialog { min-height: 0; }
/* BaseDialog teleport 到 body，通过 :has 匹配弹层并放大到超过 full 档位默认宽度 */
:global(.modal-content:has(.sp-health-detail-dialog)) {
  width: 100%;
  max-width: min(96rem, calc(100vw - 2rem));
}
.sp-health-detail-grid { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr); gap: 1rem; margin-bottom: 1rem; }
.sp-health-summary-panel { grid-row: span 2; }
.sp-health-kpis { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.5rem; padding: 0 1rem 1rem; }
.sp-health-kpis .sp-chart-kpi { min-height: 4.5rem; }
.sp-health-kpis .sp-chart-kpi b.good { color: var(--sp-green); }
.sp-health-kpis .sp-chart-kpi b.warn { color: var(--sp-amber); }
.sp-health-kpis .sp-chart-kpi b.bad { color: var(--sp-red); }
.sp-health-latest { display: grid; gap: 0.625rem; padding: 0 1rem 1rem; }
.sp-health-latest > div { display: grid; grid-template-columns: 5rem minmax(0, 1fr); gap: 0.75rem; padding: 0.625rem 0.75rem; border: 1px solid var(--sp-soft); border-radius: 0.5rem; background: var(--sp-panel-2); }
.sp-health-latest span { color: var(--sp-muted); font-size: 0.75rem; }
.sp-health-latest strong { overflow-wrap: anywhere; font-size: 0.8125rem; font-weight: 500; }
.sp-health-latest-error strong,
.sp-health-event-error dd { font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace; font-size: 0.75rem; line-height: 1.55; }
.sp-health-upstream { display: grid; gap: 0.5rem; padding: 0 1rem 1rem; }
.sp-health-upstream-head { display: flex; align-items: center; gap: 0.5rem; color: var(--sp-muted); font-size: 0.75rem; }
.sp-health-upstream-head em { margin-left: auto; font-style: normal; }
.sp-health-upstream-item { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 0.2rem 0.75rem; padding: 0.55rem 0.75rem; border: 1px solid var(--sp-soft); border-radius: 0.5rem; background: var(--sp-panel-2); }
.sp-health-upstream-item strong { font-size: 0.8125rem; font-weight: 600; }
.sp-health-upstream-item b { color: var(--sp-cyan); font-size: 0.8125rem; font-variant-numeric: tabular-nums; text-align: right; }
.sp-health-upstream-item span { color: var(--sp-muted); font-size: 0.72rem; }
.sp-health-upstream-item time { color: var(--sp-muted); font-size: 0.72rem; font-variant-numeric: tabular-nums; text-align: right; }
.sp-health-chart-panel { min-width: 0; }
.sp-health-chart { height: 18rem; padding: 0 1rem 1rem; }
.sp-health-chart-state { min-height: 18rem; color: var(--sp-muted); }
.sp-health-event-panel { overflow: hidden; grid-column: 1 / -1; }
.sp-health-events { display: grid; grid-template-columns: repeat(auto-fill, minmax(20rem, 1fr)); gap: 0.625rem; max-height: 24rem; padding: 0 1rem 1rem; overflow-y: auto; }
.sp-health-event { padding: 0.75rem; border: 1px solid var(--sp-soft); border-left: 3px solid var(--sp-line); border-radius: 0.5rem; background: var(--sp-panel-2); }
.sp-health-event--good { border-left-color: var(--sp-green); }
.sp-health-event--warn { border-left-color: var(--sp-amber); }
.sp-health-event--bad { border-left-color: var(--sp-red); }
.sp-health-event--info { border-left-color: var(--sp-blue); }
.sp-health-event-head { display: flex; align-items: center; flex-wrap: wrap; gap: 0.625rem; }
.sp-health-event-head time { color: var(--sp-muted); font-size: 0.75rem; }
.sp-health-event-head strong { margin-left: auto; font-size: 0.8125rem; font-variant-numeric: tabular-nums; }
.sp-health-event dl { display: grid; gap: 0.35rem; margin: 0.65rem 0 0; }
.sp-health-event dl > div { display: grid; grid-template-columns: 4.5rem minmax(0, 1fr); gap: 0.75rem; }
.sp-health-event dt { color: var(--sp-muted); font-size: 0.72rem; }
.sp-health-event dd { margin: 0; overflow-wrap: anywhere; font-size: 0.78rem; }
.sp-health-account-cell { display: grid; gap: 0.35rem; min-width: 12.5rem; }
.sp-health-account-tags { display: flex; flex-wrap: wrap; gap: 0.3rem; }
/* 芯片配色只声明色相，底色和描边由 color-mix 混入当前面板色，明暗主题共用一套规则 */
.sp-health-chip { display: inline-flex; align-items: center; min-height: 1.25rem; padding: 0.1rem 0.45rem; border: 1px solid color-mix(in srgb, var(--chip-hue) 28%, var(--sp-line)); border-radius: 999px; background: color-mix(in srgb, var(--chip-hue) 10%, var(--sp-panel)); color: var(--chip-hue); font-size: 0.68rem; font-weight: 600; line-height: 1.1; }
.dark .sp-health-chip { color: color-mix(in srgb, var(--chip-hue) 55%, #ffffff); }
.sp-health-chip--provider.provider-0 { --chip-hue: #2563eb; }
.sp-health-chip--provider.provider-1 { --chip-hue: #7c3aed; }
.sp-health-chip--provider.provider-2 { --chip-hue: #c2410c; }
.sp-health-chip--provider.provider-3 { --chip-hue: #047857; }
.sp-health-chip--provider.provider-4 { --chip-hue: #be185d; }
.sp-health-chip--platform.platform-0 { --chip-hue: #0f766e; }
.sp-health-chip--platform.platform-1 { --chip-hue: #4338ca; }
.sp-health-chip--platform.platform-2 { --chip-hue: #b45309; }
.sp-health-chip--platform.platform-3 { --chip-hue: #15803d; }
.sp-health-chip--platform.platform-4 { --chip-hue: #0369a1; }
.sp-health-trend-cell { display: grid; min-width: 20rem; gap: 0.4rem; padding: 0.45rem 0.55rem 0.35rem; border: 1px solid var(--sp-soft, #e5e7eb); border-radius: 0.5rem; background: linear-gradient(180deg, color-mix(in srgb, var(--sp-panel-2, #fff) 92%, var(--sp-cyan, #0891b2) 8%), var(--sp-panel-2, #fff)); }
.sp-health-trend-meta { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; color: var(--sp-muted); font-size: 0.7rem; line-height: 1; }
.sp-health-trend-meta span { color: var(--sp-text); font-variant-numeric: tabular-nums; }
.sp-health-trend-meta em { color: var(--sp-amber, #d97706); font-style: normal; font-variant-numeric: tabular-nums; }
.sp-health-trend-meta time { margin-left: auto; font-variant-numeric: tabular-nums; }
.sp-health-trend-bars { display: flex; align-items: end; gap: 0.08rem; height: 6.5rem; padding: 0.35rem 0; border-bottom: 1px solid var(--sp-soft, #d1d5db); background: repeating-linear-gradient(to top, transparent 0, transparent calc(33.333% - 1px), color-mix(in srgb, var(--sp-soft, #d1d5db) 72%, transparent) calc(33.333% - 1px), color-mix(in srgb, var(--sp-soft, #d1d5db) 72%, transparent) 33.333%); }
/* 阈值线画在内容区 60% 处，与 latencyBarHeight 的阈值归一比例一致，过线即慢响应 */
.sp-health-trend-bars--threshold { background: linear-gradient(to top, transparent calc(60% - 1px), color-mix(in srgb, var(--sp-amber, #d97706) 60%, transparent) calc(60% - 1px), color-mix(in srgb, var(--sp-amber, #d97706) 60%, transparent) 60%, transparent 60%); background-clip: content-box; }
.sp-health-trend-bar { display: block; flex: 1 1 0; min-width: 0.12rem; max-width: 0.36rem; min-height: 0.5rem; border-radius: 0.16rem 0.16rem 0.06rem 0.06rem; box-shadow: inset 0 1px rgb(255 255 255 / 25%); transition: height 160ms ease, opacity 160ms ease; }
.sp-health-trend-bar:hover { opacity: 0.72; }
.sp-health-trend-bar--good { background: var(--sp-green, #16a34a); }
.sp-health-trend-bar--warn { background: var(--sp-amber, #d97706); }
.sp-health-trend-bar--bad { background: var(--sp-red, #dc2626); }
.sp-health-trend-bar--info { background: #9ca3af; }
.sp-health-trend-bar--loading { height: 45%; background: var(--sp-soft, #d1d5db); animation: sp-health-trend-pulse 1.1s ease-in-out infinite alternate; }
.sp-health-trend-bar--loading:nth-child(2n) { height: 70%; animation-delay: 120ms; }
.sp-health-trend-bar--loading:nth-child(3n) { height: 30%; animation-delay: 220ms; }
.sp-health-trend-empty { color: var(--sp-muted); font-size: 0.75rem; }
/* 上游色带是参照信息，整体降透明度让它在视觉层级上低于守护柱 */
.sp-health-upstream-strip { display: flex; gap: 0.08rem; height: 0.3rem; margin-top: 0.15rem; opacity: 0.85; }
.sp-health-upstream-cell { flex: 1 1 0; min-width: 0; border-radius: 0.08rem; }
.sp-health-upstream-cell--good { background: var(--sp-green, #16a34a); }
.sp-health-upstream-cell--warn { background: var(--sp-amber, #d97706); }
.sp-health-upstream-cell--bad { background: var(--sp-red, #dc2626); }
.sp-health-upstream-cell--info { background: var(--sp-soft, #d1d5db); }
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
  .sp-health-upstream-item { grid-template-columns: 1fr; }
  .sp-health-upstream-item b,
  .sp-health-upstream-item time { text-align: left; }
  .sp-health-chart { height: 15rem; }
}
</style>
