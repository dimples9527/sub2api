<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Upstream Operations / Live</div>
        <h1>上游运维驾驶舱</h1>
        <p class="sp-subtitle">先处理影响真实流量的问题，再判断账号池健康与成本变化。</p>
      </div>
      <div class="sp-controls">
        <div class="sp-segmented" aria-label="统计范围">
          <button data-test="range-24h" type="button" :class="{ active: range === '24h' }" @click="setRange('24h')">24 小时</button>
          <button data-test="range-7d" type="button" :class="{ active: range === '7d' }" @click="setRange('7d')">7 天</button>
        </div>
        <button class="sp-button" data-test="refresh" type="button" :disabled="refreshing" @click="refreshAll">
          {{ refreshing ? '刷新中…' : '刷新数据' }}
        </button>
        <button class="sp-button primary" type="button" @click="openAutomation">查看自动任务</button>
      </div>
    </header>

    <div class="sp-data-note" data-test="data-note" :class="{ 'sp-error-line': Boolean(overview.error) }">
      <b>{{ dataNoteTitle }}</b>
      <span>{{ dataNoteText }}</span>
    </div>

    <section class="sp-risk-grid" data-test="risk-grid">
      <article
        v-for="risk in riskCards"
        :key="risk.key"
        class="sp-risk-card"
        :class="[`sp-${risk.tone}`, { selected: selectedRisk === risk.key }]"
        :data-test="`risk-${risk.key}`"
        @click="selectRisk(risk.key)"
      >
        <div class="sp-risk-label">{{ risk.label }}</div>
        <div class="sp-risk-value">{{ risk.value }}</div>
        <div class="sp-risk-foot">{{ risk.foot }}</div>
      </article>
    </section>
    <section class="sp-grid-main">
      <div class="sp-panel" data-test="accounts-section">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">01</span>
            <div>
              <h2>高影响异常账号</h2>
              <span>已按“{{ activeRiskLabel }}”筛选 · {{ accountsMeta }}</span>
            </div>
          </div>
          <div class="sp-tools">
            <span class="sp-filter-pill">影响优先</span>
            <span class="sp-filter-pill">有效账号池</span>
            <button class="sp-button small ghost" type="button" @click="selectRisk('all')">查看全部 {{ accountsTotalLabel }}</button>
          </div>
        </header>

        <div v-if="accounts.loading && !accounts.data" class="sp-panel-body" data-test="accounts-loading">账号风险加载中…</div>
        <div v-else-if="accounts.error" class="sp-panel-body sp-error-line" data-test="accounts-error">{{ accounts.error }}</div>
        <div v-else class="sp-table-wrap">
          <table class="sp-table">
            <thead>
              <tr>
                <th>风险</th>
                <th>供应商 / 账号</th>
                <th>业务影响</th>
                <th>倍率</th>
                <th>余额</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in accountRows"
                :key="item.account_id"
                class="clickable"
                :data-test="`account-row-${item.account_id}`"
                @click="openAccount(item)"
              >
                <td><span class="sp-severity" :class="severityTone(item.severity)">{{ severityLabel(item.severity) }}</span></td>
                <td>
                  <div class="sp-account-name">{{ item.account_name }}</div>
                  <div class="sp-subline">{{ item.provider_name }} · {{ item.group_name || item.group_key || '未分组' }}</div>
                </td>
                <td>
                  <span class="sp-metric-strong">{{ formatCount(item.request_count) }}</span>
                  <div class="sp-subline">请求 · 成功率 {{ formatPercent(item.success_rate) }}</div>
                </td>
                <td>
                  <span class="sp-metric-strong">{{ formatRate(item.current_rate) }}</span>
                  <div class="sp-subline" :class="rateTone(item)">{{ rateHint(item) }}</div>
                </td>
                <td>
                  <span class="sp-metric-strong">{{ formatMoney(item.balance, item.balance_currency) }}</span>
                  <div class="sp-subline" :class="{ 'sp-up': isBalanceRisk(item) }">{{ balanceHint(item) }}</div>
                </td>
                <td><span class="sp-tag" :class="statusTone(item)">{{ item.status || '—' }}</span></td>
              </tr>
              <tr v-if="!accounts.loading && accountRows.length === 0">
                <td colspan="6" data-test="accounts-empty">该筛选下暂无异常账号</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <aside class="sp-panel" data-test="health-section">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">02</span>
            <div>
              <h2>有效账号池健康</h2>
              <span>{{ healthSubtitle }}</span>
            </div>
          </div>
          <span class="sp-tag" :class="healthTagTone">{{ healthTagLabel }}</span>
        </header>
        <div v-if="overview.loading && !overview.data" class="sp-panel-body" data-test="health-loading">健康概览加载中…</div>
        <div v-else-if="overview.error && !overview.data" class="sp-panel-body sp-error-line" data-test="health-error">{{ overview.error }}</div>
        <div v-else class="sp-health-body">
          <div class="sp-health-score">
            <div class="sp-ring">
              <svg viewBox="0 0 120 120" aria-hidden="true">
                <circle cx="60" cy="60" r="50" fill="none" stroke="var(--sp-soft)" stroke-width="8" />
                <circle
                  cx="60"
                  cy="60"
                  r="50"
                  fill="none"
                  :stroke="healthRingColor"
                  stroke-width="8"
                  stroke-linecap="square"
                  stroke-dasharray="314"
                  :stroke-dashoffset="healthRingOffset"
                />
              </svg>
              <div>
                <strong>{{ healthScoreLabel }}</strong>
                <small>综合健康度</small>
              </div>
            </div>
            <div class="sp-health-copy">
              <h3>{{ healthTitle }}</h3>
              <p>{{ healthCopy }}</p>
            </div>
          </div>
          <div class="sp-health-list">
            <div v-for="item in healthItems" :key="item.label" class="sp-health-item">
              <b :class="item.tone">{{ item.value }}</b>
              <span>{{ item.label }}</span>
            </div>
          </div>
        </div>
      </aside>
    </section>
    <section class="sp-wide-grid">
      <div class="sp-panel" data-test="rates-section">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">03</span>
            <div>
              <h2>倍率与最低倍率分析</h2>
              <span>同供应商、同上游分组比较 · {{ ratesMeta }}</span>
            </div>
          </div>
          <div class="sp-tools">
            <button
              v-for="tab in rateTabs"
              :key="tab.view"
              class="sp-pill"
              :class="{ active: rateView === tab.view }"
              type="button"
              :data-test="`rate-tab-${tab.view}`"
              @click="setRateView(tab.view)"
            >
              {{ tab.label }}
            </button>
          </div>
        </header>
        <div v-if="rates.loading && !rates.data" class="sp-panel-body" data-test="rates-loading">倍率分析加载中…</div>
        <div v-else-if="rates.error" class="sp-panel-body sp-error-line" data-test="rates-error">{{ rates.error }}</div>
        <div v-else class="sp-table-wrap">
          <table class="sp-table">
            <thead>
              <tr>
                <th>供应商 / 分组</th>
                <th>承载账号</th>
                <th>当前倍率</th>
                <th>同组最低</th>
                <th>倍率差</th>
                <th>预计额外成本</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in rateRows"
                :key="`${item.provider_slug}-${item.group_key}`"
                class="clickable"
                :data-test="`rate-row-${item.provider_slug}-${item.group_key}`"
                @click="openPath(item.target_path)"
              >
                <td>
                  <div class="sp-account-name">{{ item.provider_name }}</div>
                  <div class="sp-subline">{{ item.group_name || item.group_key || '分组键缺失' }} · {{ item.enabled_account_count }} 个有效账号</div>
                </td>
                <td>{{ item.current_account_name || '—' }}</td>
                <td>
                  <span class="sp-metric-strong">{{ formatRate(item.current_rate) }}</span>
                  <div v-if="item.rate_delta_percent != null && item.rate_delta_percent > 0" class="sp-subline sp-up">↑</div>
                </td>
                <td>{{ lowestRateLabel(item) }}</td>
                <td>
                  <div class="sp-rate-cell">
                    <span>{{ formatDelta(item.rate_delta_percent) }}</span>
                    <div class="sp-rate-line"><i :style="{ width: deltaWidth(item.rate_delta_percent) }" /></div>
                  </div>
                </td>
                <td :class="{ 'sp-up': item.estimated_extra_cost != null && item.estimated_extra_cost > 0 }">
                  {{ extraCostLabel(item) }}
                </td>
              </tr>
              <tr v-if="!rates.loading && rateRows.length === 0">
                <td colspan="6" data-test="rates-empty">当前视图暂无倍率分组</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="sp-panel" data-test="providers-section">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">04</span>
            <div>
              <h2>供应商运行概览</h2>
              <span>{{ providersMeta }}</span>
            </div>
          </div>
        </header>
        <div v-if="providers.loading && !providers.data" class="sp-panel-body" data-test="providers-loading">供应商概览加载中…</div>
        <div v-else-if="providers.error" class="sp-panel-body sp-error-line" data-test="providers-error">{{ providers.error }}</div>
        <div v-else-if="providerCards.length === 0" class="sp-panel-body" data-test="providers-empty">暂无供应商运行数据</div>
        <div v-else class="sp-provider-grid">
          <article
            v-for="item in providerCards"
            :key="item.provider_slug"
            class="sp-provider-card"
            :class="providerCardTone(item.status)"
            :data-test="`provider-card-${item.provider_slug}`"
            @click="openPath(item.target_path)"
          >
            <div class="sp-provider-head">
              <div>
                <div class="sp-account-name">{{ item.provider_name }}</div>
                <div class="sp-subline" :class="providerStatusTone(item.status)">{{ providerStatusLabel(item) }}</div>
              </div>
              <span class="sp-tag" :class="providerTagTone(item.status)">{{ providerIssueLabel(item) }}</span>
            </div>
            <div class="sp-provider-stats">
              <div class="sp-provider-stat"><span>可调度 / 有效</span><b>{{ item.schedulable_account_count }} / {{ item.enabled_account_count }}</b></div>
              <div class="sp-provider-stat"><span>成功率</span><b>{{ formatPercent(item.success_rate) }}</b></div>
              <div class="sp-provider-stat"><span>周期成本</span><b>{{ formatMoney(item.period_cost, item.cost_currency) }}</b></div>
              <div class="sp-provider-stat"><span>预计余额</span><b :class="{ 'sp-up': item.balance_risk || (item.estimated_days != null && item.estimated_days < 3) }">{{ daysLabel(item.estimated_days) }}</b></div>
            </div>
          </article>
        </div>
      </div>
    </section>

    <section class="sp-panel" style="margin-bottom: 14px" data-test="tasks-section">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">05</span>
          <div>
            <h2>自动任务运行状态</h2>
            <span>{{ tasksMeta }}</span>
          </div>
        </div>
      </header>
      <div v-if="overview.loading && !overview.data" class="sp-panel-body">任务状态加载中…</div>
      <div v-else-if="overview.error && !overview.data" class="sp-panel-body sp-error-line">{{ overview.error }}</div>
      <div v-else-if="taskRows.length === 0" class="sp-panel-body" data-test="tasks-empty">暂无自动任务数据</div>
      <div v-else class="sp-task-list">
        <div v-for="task in taskRows" :key="task.key" class="sp-task-row">
          <div>
            <div class="sp-account-name">{{ task.name }}</div>
            <div class="sp-subline">{{ taskMeta(task) }}</div>
          </div>
          <span class="sp-tag" :class="taskTone(task)">{{ taskStatusLabel(task) }}</span>
          <div><b>{{ task.affected_count }}</b><div class="sp-subline">影响数</div></div>
          <div><b>{{ formatDateTime(task.last_run_at) }}</b><div class="sp-subline">上次运行</div></div>
          <div><b>{{ formatDateTime(task.next_run_at) }}</b><div class="sp-subline">下次运行</div></div>
          <button
            class="sp-button small"
            type="button"
            :data-test="`task-action-${task.key}`"
            @click="openPath(task.settings_path)"
          >
            查看
          </button>
        </div>
      </div>
    </section>
  </SupplierModuleLayout>
</template>
<script setup lang="ts">
/**
 * 供应商运维驾驶舱：保持工业风 sp-* 视觉，仅接入真实只读数据与分区微状态。
 */
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import {
  getAccounts,
  getOverview,
  getProviders,
  getRates,
  type SupplierDashboardAccountItem,
  type SupplierDashboardAccountsResponse,
  type SupplierDashboardOverviewResponse,
  type SupplierDashboardProviderItem,
  type SupplierDashboardProvidersResponse,
  type SupplierDashboardRange,
  type SupplierDashboardRateItem,
  type SupplierDashboardRatesResponse,
  type SupplierDashboardRateView,
  type SupplierDashboardRiskType,
  type SupplierDashboardSeverity,
} from '@/api/admin/supplierDashboard'

interface SectionState<T> {
  loading: boolean
  error: string
  data: T | null
  lastUpdated: string | null
}

interface RiskCard {
  key: SupplierDashboardRiskType | 'disabled'
  tone: string
  label: string
  value: string
  foot: string
}

interface OverviewTask {
  key: string
  name: string
  enabled?: boolean
  last_run_at?: string
  last_run_status?: string
  last_run_message?: string
  next_run_at?: string
  affected_count?: number
  settings_path?: string
}

const UNKNOWN = '—'
const router = useRouter()

const range = ref<SupplierDashboardRange>('24h')
const selectedRisk = ref<SupplierDashboardRiskType | 'disabled'>('critical')
const rateView = ref<SupplierDashboardRateView>('risk')
const refreshing = ref(false)

const overview = reactive<SectionState<SupplierDashboardOverviewResponse>>({
  loading: false,
  error: '',
  data: null,
  lastUpdated: null,
})
const accounts = reactive<SectionState<SupplierDashboardAccountsResponse>>({
  loading: false,
  error: '',
  data: null,
  lastUpdated: null,
})
const rates = reactive<SectionState<SupplierDashboardRatesResponse>>({
  loading: false,
  error: '',
  data: null,
  lastUpdated: null,
})
const providers = reactive<SectionState<SupplierDashboardProvidersResponse>>({
  loading: false,
  error: '',
  data: null,
  lastUpdated: null,
})

/** 各风险筛选 total 缓存，随对应请求成功更新。 */
const riskTotals = reactive<Partial<Record<SupplierDashboardRiskType, number>>>({})

let abortController: AbortController | null = null
let requestSeq = 0
const accountsSeq = ref(0)
const ratesSeq = ref(0)

const rateTabs: Array<{ view: SupplierDashboardRateView; label: string }> = [
  { view: 'risk', label: '风险分组' },
  { view: 'changed', label: '最近变化' },
  { view: 'all', label: '全部分组' },
]

const riskDefs: Array<{ key: SupplierDashboardRiskType | 'disabled'; tone: string; label: string; foot: string }> = [
  { key: 'critical', tone: 'red', label: '紧急异常账号', foot: '优先处理严重风险' },
  { key: 'traffic', tone: 'red', label: '承载流量的异常账号', foot: '仍在承载真实请求' },
  { key: 'rate_up', tone: 'orange', label: '本次同步倍率上涨', foot: '关注成本抬升' },
  { key: 'not_lowest', tone: 'amber', label: '非最低倍率承载账号', foot: '同组存在更低倍率' },
  { key: 'balance', tone: 'orange', label: '预计余额不足 3 天', foot: '避免中断承载' },
  { key: 'sync', tone: 'blue', label: '账号同步冲突', foot: '等待确认匹配关系' },
  { key: 'task', tone: 'violet', label: '自动任务异常', foot: '巡检/采样失败' },
  { key: 'disabled', tone: 'muted-tone', label: '已禁用资源', foot: '供应商维度禁用统计' },
]

const accountRows = computed(() => accounts.data?.items ?? [])
const rateRows = computed(() => rates.data?.items ?? [])
const providerCards = computed(() => providers.data?.items ?? [])

const overviewSummary = computed(() => asRecord(overview.data?.summary))
const overviewStability = computed(() => asRecord(overview.data?.stability))
const overviewCost = computed(() => asRecord(overview.data?.cost))
const taskRows = computed<OverviewTask[]>(() => {
  const raw = overview.data?.tasks
  return Array.isArray(raw) ? (raw as unknown as OverviewTask[]) : []
})

const activeRiskLabel = computed(() => {
  if (selectedRisk.value === 'all') return '全部风险'
  return riskDefs.find((item) => item.key === selectedRisk.value)?.label || '全部风险'
})

const accountsTotalLabel = computed(() => {
  if (typeof riskTotals.all === 'number') return String(riskTotals.all)
  if (accounts.data) return String(accounts.data.total)
  return UNKNOWN
})

const accountsMeta = computed(() => {
  if (accounts.loading) return '加载中'
  if (accounts.error) return '加载失败'
  if (accounts.lastUpdated) return `更新于 ${formatClock(accounts.lastUpdated)}`
  return '等待数据'
})

const ratesMeta = computed(() => {
  if (rates.loading) return '加载中'
  if (rates.error) return '加载失败'
  if (rates.lastUpdated) return `更新于 ${formatClock(rates.lastUpdated)}`
  return '等待数据'
})

const providersMeta = computed(() => {
  if (providers.loading) return '加载中'
  if (providers.error) return '加载失败'
  if (providers.data) return `共 ${providers.data.total} 个供应商`
  return '等待数据'
})

const tasksMeta = computed(() => {
  if (overview.loading) return '加载中'
  if (overview.error) return '加载失败'
  return taskRows.value.length ? `共 ${taskRows.value.length} 个任务` : '暂无任务'
})

const dataNoteTitle = computed(() => {
  if (overview.error && !overview.data) return '数据异常'
  if (overview.loading && !overview.data) return '数据加载中'
  if ((overview.data?.warnings?.length || 0) > 0) return '数据提示'
  return '数据完整'
})

const dataNoteText = computed(() => {
  if (overview.error && !overview.data) return overview.error
  if (overview.loading && !overview.data) return '正在并行拉取概览、异常账号、倍率与供应商数据。'
  const generated = overview.data?.generated_at || accounts.data?.generated_at || rates.data?.generated_at
  const warning = overview.data?.warnings?.[0]?.message
  const base = generated
    ? `统计生成于 ${formatClock(generated)}；账号实时状态为当前值，流量与成本按所选时间范围统计。`
    : '尚未获取到生成时间。'
  return warning ? `${base} ${warning}` : base
})

const riskCards = computed<RiskCard[]>(() =>
  riskDefs.map((def) => ({
    ...def,
    value: riskValue(def.key),
    foot: riskFoot(def.key, def.foot),
  })),
)

const healthScore = computed(() => {
  const raw = overviewStability.value.health_score
  return typeof raw === 'number' && Number.isFinite(raw) ? raw : null
})
const healthScoreLabel = computed(() => (healthScore.value == null ? UNKNOWN : `${Math.round(healthScore.value)}%`))
const healthRingOffset = computed(() => {
  if (healthScore.value == null) return 314
  const clamped = Math.max(0, Math.min(100, healthScore.value))
  return 314 - (314 * clamped) / 100
})
const healthRingColor = computed(() => {
  if (healthScore.value == null) return 'var(--sp-muted)'
  if (healthScore.value >= 90) return 'var(--sp-green)'
  if (healthScore.value >= 75) return 'var(--sp-amber)'
  return 'var(--sp-red)'
})
const healthTagTone = computed(() => {
  if (healthScore.value == null) return 'info'
  if (healthScore.value >= 90) return 'good'
  if (healthScore.value >= 75) return 'warn'
  return 'bad'
})
const healthTagLabel = computed(() => {
  if (healthScore.value == null) return '未知'
  if (healthScore.value >= 90) return '稳定'
  if (healthScore.value >= 75) return '需关注'
  return '高风险'
})
const healthTitle = computed(() => {
  if (healthScore.value == null) return '健康度待评估'
  if (healthScore.value >= 90) return '账号池运行平稳'
  if (healthScore.value >= 75) return '账号池可继续承载流量'
  return '账号池存在明显风险'
})
const healthCopy = computed(() => {
  const matched = num(overviewSummary.value.matched_account_count)
  const rateRisk = num(overviewSummary.value.rate_risk_count)
  const pending = num(overviewSummary.value.pending_account_count)
  const parts: string[] = []
  if (matched != null) parts.push(`有效匹配账号 ${formatCount(matched)}`)
  if (rateRisk != null) parts.push(`倍率风险 ${formatCount(rateRisk)}`)
  if (pending != null) parts.push(`待处理同步 ${formatCount(pending)}`)
  if (!parts.length) return '概览指标暂不可用，请稍后刷新。'
  return `${parts.join('，')}。建议优先处理高影响异常账号。`
})
const healthSubtitle = computed(() => {
  const disabledProviders = num(overviewSummary.value.disabled_provider_count)
  if (disabledProviders == null) return '基于当前有效供应商与账号池'
  return `已排除 ${formatCount(disabledProviders)} 个禁用供应商`
})
const healthItems = computed(() => {
  const matched = num(overviewSummary.value.matched_account_count)
  const successRate = num(overviewStability.value.success_rate)
  const requestCount = num(overviewStability.value.request_count)
  const rateRisk = num(overviewSummary.value.rate_risk_count)
  const pending = num(overviewSummary.value.pending_account_count)
  const anomaly = num(overviewCost.value.anomaly_providers)
  const schedulable = providerCards.value.reduce((sum, item) => sum + (item.schedulable_account_count || 0), 0)
  const enabled = providerCards.value.reduce((sum, item) => sum + (item.enabled_account_count || 0), 0)
  const schedulableLabel = enabled > 0 ? `${formatCount(schedulable)} / ${formatCount(enabled)}` : formatCount(schedulable || null)
  return [
    { value: formatCount(matched), label: '有效账号' },
    { value: schedulableLabel, label: '当前可调度 / 有效' },
    { value: formatCount(requestCount), label: range.value === '7d' ? '近 7 日请求' : '近 24h 请求' },
    { value: formatPercent(successRate), label: '近期成功率' },
    { value: formatCount(rateRisk), label: '倍率风险', tone: rateRisk && rateRisk > 0 ? 'sp-up' : '' },
    { value: formatCount(pending ?? anomaly), label: '同步/异常待处理', tone: pending || anomaly ? 'sp-amber' : '' },
  ]
})

onMounted(() => {
  void loadAll()
})
onBeforeUnmount(() => {
  abortController?.abort()
  abortController = null
})
function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' ? (value as Record<string, unknown>) : {}
}

function num(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function isAbortError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false
  const name = (error as { name?: string }).name
  const code = (error as { code?: string }).code
  return name === 'CanceledError' || name === 'AbortError' || code === 'ERR_CANCELED'
}

function errorMessage(error: unknown, fallback: string): string {
  if (isAbortError(error)) return ''
  if (error instanceof Error && error.message) return error.message
  return fallback
}

function beginGlobalRequest(): { seq: number; signal: AbortSignal } {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  requestSeq += 1
  return { seq: requestSeq, signal: controller.signal }
}

function isCurrent(seq: number): boolean {
  return seq === requestSeq
}

async function loadAll(): Promise<void> {
  const { seq, signal } = beginGlobalRequest()
  overview.loading = true
  accounts.loading = true
  rates.loading = true
  providers.loading = true
  overview.error = ''
  accounts.error = ''
  rates.error = ''
  providers.error = ''

  const riskType: SupplierDashboardRiskType = selectedRisk.value === 'disabled' ? 'all' : selectedRisk.value
  await Promise.all([
    settleOverview(seq, signal),
    settleAccounts(seq, signal, riskType),
    settleRates(seq, signal, rateView.value),
    settleProviders(seq, signal),
  ])
}

async function refreshAll(): Promise<void> {
  refreshing.value = true
  try {
    await loadAll()
  } finally {
    refreshing.value = false
  }
}

async function setRange(next: SupplierDashboardRange): Promise<void> {
  if (range.value === next) return
  range.value = next
  await loadAll()
}

async function selectRisk(next: SupplierDashboardRiskType | 'disabled'): Promise<void> {
  if (selectedRisk.value === next) return
  selectedRisk.value = next
  if (next === 'disabled') return
  await reloadAccounts()
}

async function setRateView(next: SupplierDashboardRateView): Promise<void> {
  if (rateView.value === next) return
  rateView.value = next
  await reloadRates()
}

async function reloadAccounts(): Promise<void> {
  // 局部刷新：不中断其他分区，使用独立 accountsSeq + 当前 abort signal（若无则新建）。
  if (!abortController) beginGlobalRequest()
  const signal = abortController!.signal
  const seq = ++accountsSeq.value
  accounts.loading = true
  accounts.error = ''
  try {
    const riskType: SupplierDashboardRiskType = selectedRisk.value === 'disabled' ? 'all' : selectedRisk.value
    const data = await getAccounts(
      { range: range.value, risk_type: riskType, page: 1, page_size: 20 },
      { signal },
    )
    if (signal.aborted || seq !== accountsSeq.value) return
    accounts.data = data
    accounts.lastUpdated = data.generated_at
    riskTotals[riskType] = data.total
    if (riskType === 'all') riskTotals.all = data.total
  } catch (error) {
    if (signal.aborted || seq !== accountsSeq.value) return
    const message = errorMessage(error, '异常账号加载失败')
    if (!message) return
    accounts.error = message
    accounts.data = null
  } finally {
    if (seq === accountsSeq.value) accounts.loading = false
  }
}

async function reloadRates(): Promise<void> {
  if (!abortController) beginGlobalRequest()
  const signal = abortController!.signal
  const seq = ++ratesSeq.value
  rates.loading = true
  rates.error = ''
  try {
    const data = await getRates(
      { range: range.value, view: rateView.value, page: 1, page_size: 20 },
      { signal },
    )
    if (signal.aborted || seq !== ratesSeq.value) return
    rates.data = data
    rates.lastUpdated = data.generated_at
  } catch (error) {
    if (signal.aborted || seq !== ratesSeq.value) return
    const message = errorMessage(error, '倍率分析加载失败')
    if (!message) return
    rates.error = message
    rates.data = null
  } finally {
    if (seq === ratesSeq.value) rates.loading = false
  }
}

async function settleOverview(seq: number, signal: AbortSignal): Promise<void> {
  try {
    const data = await getOverview(range.value, { signal })
    if (!isCurrent(seq) || signal.aborted) return
    overview.data = data
    overview.lastUpdated = data.generated_at
    overview.error = ''
  } catch (error) {
    if (!isCurrent(seq) || signal.aborted) return
    const message = errorMessage(error, '概览加载失败')
    if (!message) return
    overview.error = message
  } finally {
    if (isCurrent(seq)) overview.loading = false
  }
}

async function settleAccounts(seq: number, signal: AbortSignal, riskType: SupplierDashboardRiskType): Promise<void> {
  const local = ++accountsSeq.value
  try {
    const data = await getAccounts(
      { range: range.value, risk_type: riskType, page: 1, page_size: 20 },
      { signal },
    )
    if (!isCurrent(seq) || signal.aborted || local !== accountsSeq.value) return
    accounts.data = data
    accounts.lastUpdated = data.generated_at
    accounts.error = ''
    riskTotals[riskType] = data.total
    if (riskType === 'all') riskTotals.all = data.total
  } catch (error) {
    if (!isCurrent(seq) || signal.aborted || local !== accountsSeq.value) return
    const message = errorMessage(error, '异常账号加载失败')
    if (!message) return
    accounts.error = message
    accounts.data = null
  } finally {
    if (isCurrent(seq) && local === accountsSeq.value) accounts.loading = false
  }
}

async function settleRates(seq: number, signal: AbortSignal, view: SupplierDashboardRateView): Promise<void> {
  const local = ++ratesSeq.value
  try {
    const data = await getRates(
      { range: range.value, view, page: 1, page_size: 20 },
      { signal },
    )
    if (!isCurrent(seq) || signal.aborted || local !== ratesSeq.value) return
    rates.data = data
    rates.lastUpdated = data.generated_at
    rates.error = ''
  } catch (error) {
    if (!isCurrent(seq) || signal.aborted || local !== ratesSeq.value) return
    const message = errorMessage(error, '倍率分析加载失败')
    if (!message) return
    rates.error = message
    rates.data = null
  } finally {
    if (isCurrent(seq) && local === ratesSeq.value) rates.loading = false
  }
}

async function settleProviders(seq: number, signal: AbortSignal): Promise<void> {
  try {
    const data = await getProviders(
      { range: range.value, page: 1, page_size: 20 },
      { signal },
    )
    if (!isCurrent(seq) || signal.aborted) return
    providers.data = data
    providers.lastUpdated = data.generated_at
    providers.error = ''
  } catch (error) {
    if (!isCurrent(seq) || signal.aborted) return
    const message = errorMessage(error, '供应商概览加载失败')
    if (!message) return
    providers.error = message
    providers.data = null
  } finally {
    if (isCurrent(seq)) providers.loading = false
  }
}

function riskValue(key: SupplierDashboardRiskType | 'disabled'): string {
  if (key === 'disabled') {
    const disabledProviders = num(overviewSummary.value.disabled_provider_count)
    return disabledProviders == null ? UNKNOWN : String(disabledProviders)
  }
  if (typeof riskTotals[key] === 'number') return String(riskTotals[key])
  if (selectedRisk.value === key && accounts.data) return String(accounts.data.total)
  if (key === 'not_lowest') {
    const rateRisk = num(overviewSummary.value.rate_risk_count)
    return rateRisk == null ? UNKNOWN : String(rateRisk)
  }
  if (key === 'sync') {
    const pending = num(overviewSummary.value.pending_account_count)
    return pending == null ? UNKNOWN : String(pending)
  }
  if (key === 'task') {
    const failed = taskRows.value.filter((task) => String(task.last_run_status || '').toLowerCase().includes('fail')).length
    return overview.data ? String(failed) : UNKNOWN
  }
  return UNKNOWN
}

function riskFoot(key: SupplierDashboardRiskType | 'disabled', fallback: string): string {
  if (key === 'disabled') {
    const disabledProviders = num(overviewSummary.value.disabled_provider_count)
    return disabledProviders == null ? fallback : `禁用供应商 ${disabledProviders} 个 · 点击查看`
  }
  if (key === selectedRisk.value && accounts.data) return `当前筛选 ${accounts.data.total} 条`
  return fallback
}

function openPath(path?: string | null): void {
  if (!path) return
  void router.push(path)
}

function openAccount(item: SupplierDashboardAccountItem): void {
  openPath(item.target_path)
}

function openAutomation(): void {
  openPath('/admin/supplier-management/automation')
}

function severityLabel(severity: SupplierDashboardSeverity): string {
  switch (severity) {
    case 'critical':
      return '紧急'
    case 'high':
      return '高'
    case 'medium':
      return '中'
    case 'low':
      return '低'
    default:
      return severity
  }
}

function severityTone(severity: SupplierDashboardSeverity): string {
  switch (severity) {
    case 'critical':
      return 'critical'
    case 'high':
      return 'high'
    case 'medium':
      return 'medium'
    default:
      return 'low'
  }
}

function formatCount(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return UNKNOWN
  return new Intl.NumberFormat('zh-CN').format(value)
}

function formatPercent(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return UNKNOWN
  return `${Number(value.toFixed(1))}%`
}

function formatRate(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return UNKNOWN
  return Number(value.toFixed(4)).toString()
}

function formatDelta(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return UNKNOWN
  const sign = value > 0 ? '+' : ''
  return `${sign}${Number(value.toFixed(1))}%`
}

function formatMoney(value: number | null | undefined, currency?: string | null): string {
  if (value === null || value === undefined || Number.isNaN(value)) return UNKNOWN
  const code = (currency || '').toUpperCase()
  const prefix = code === 'CNY' || code === 'RMB' ? '¥ ' : code === 'USD' ? '$ ' : code ? `${code} ` : ''
  return `${prefix}${new Intl.NumberFormat('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: 2 }).format(value)}`
}

function daysLabel(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) return UNKNOWN
  return `${Number(value.toFixed(1))} 天`
}

function formatClock(value?: string | null): string {
  if (!value) return UNKNOWN
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString('zh-CN', { hour12: false })
}

function formatDateTime(value?: string | null): string {
  if (!value) return UNKNOWN
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

function rateHint(item: SupplierDashboardAccountItem): string {
  const delta = item.rate_delta_percent
  const lowest = item.lowest_rate
  if (delta == null && lowest == null) return '无法比较'
  const parts: string[] = []
  if (delta != null) parts.push(`${delta > 0 ? '↑ ' : ''}${formatDelta(delta)}`)
  if (lowest != null) parts.push(`最低 ${formatRate(lowest)}`)
  if (item.risk_types.includes('not_lowest')) parts.push('非最低')
  return parts.join(' · ') || '—'
}

function rateTone(item: SupplierDashboardAccountItem): string {
  if (item.rate_delta_percent != null && item.rate_delta_percent > 0) return 'sp-up'
  if (item.risk_types.includes('not_lowest')) return 'sp-up'
  if (item.lowest_rate != null && item.current_rate != null && item.current_rate <= item.lowest_rate) return 'sp-down'
  return ''
}

function isBalanceRisk(item: SupplierDashboardAccountItem): boolean {
  return item.risk_types.includes('balance') || (item.estimated_days != null && item.estimated_days < 3)
}

function balanceHint(item: SupplierDashboardAccountItem): string {
  if (item.estimated_days == null) return '预计可用天数未知'
  return `预计 ${Number(item.estimated_days.toFixed(1))} 天`
}

function statusTone(item: SupplierDashboardAccountItem): string {
  if (item.severity === 'critical') return 'bad'
  if (item.severity === 'high') return 'warn'
  if (item.risk_types.includes('balance') || item.risk_types.includes('sync')) return 'warn'
  return 'info'
}

function lowestRateLabel(item: SupplierDashboardRateItem): string {
  if (item.lowest_rate == null) {
    if (item.comparison_status === 'missing_group') return '分组键缺失'
    if (item.comparison_status === 'insufficient_accounts') return '账号不足'
    return '无法比较'
  }
  const names = item.lowest_account_names?.filter(Boolean) ?? []
  if (item.comparison_status === 'tied_lowest') return `${formatRate(item.lowest_rate)} · 并列最低`
  if (names.length) return `${formatRate(item.lowest_rate)} · ${names[0]}`
  return formatRate(item.lowest_rate)
}

function deltaWidth(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value) || value <= 0) return '0%'
  return `${Math.max(6, Math.min(100, Math.abs(value) * 4))}%`
}

function extraCostLabel(item: SupplierDashboardRateItem): string {
  if (item.estimated_extra_cost == null) return UNKNOWN
  if (item.estimated_extra_cost === 0) return '无额外成本'
  return `${formatMoney(item.estimated_extra_cost, item.cost_currency)} / 日`
}

function providerCardTone(status: SupplierDashboardProviderItem['status']): string {
  if (status === 'high_risk') return 'bad'
  if (status === 'warning') return 'warn'
  return ''
}

function providerStatusTone(status: SupplierDashboardProviderItem['status']): string {
  if (status === 'high_risk') return 'sp-up'
  if (status === 'warning') return 'sp-amber'
  if (status === 'healthy') return 'sp-down'
  return ''
}

function providerStatusLabel(item: SupplierDashboardProviderItem): string {
  switch (item.status) {
    case 'high_risk':
      return '高风险'
    case 'warning':
      return item.balance_risk ? '余额风险' : '需关注'
    case 'healthy':
      return '稳定'
    case 'disabled':
      return '已禁用'
    default:
      return '未知'
  }
}

function providerTagTone(status: SupplierDashboardProviderItem['status']): string {
  if (status === 'high_risk') return 'bad'
  if (status === 'warning') return 'warn'
  if (status === 'healthy') return 'good'
  return 'info'
}

function providerIssueLabel(item: SupplierDashboardProviderItem): string {
  if (item.critical_issue_count == null) return '严重异常未知'
  if (item.critical_issue_count === 0) return '无严重异常'
  return `${item.critical_issue_count} 个严重异常`
}

function taskTone(task: OverviewTask): string {
  const status = String(task.last_run_status || '').toLowerCase()
  if (status.includes('fail') || status.includes('error')) return 'bad'
  if (status.includes('run') || status.includes('ing')) return 'warn'
  if (status.includes('success') || status === 'ok') return 'good'
  return 'info'
}

function taskStatusLabel(task: OverviewTask): string {
  const status = String(task.last_run_status || '').toLowerCase()
  if (!status) return task.enabled === false ? '已停用' : '未知'
  if (status.includes('fail') || status.includes('error')) return '失败'
  if (status.includes('run')) return '运行中'
  if (status.includes('success') || status === 'ok') return '正常'
  return task.last_run_status || '未知'
}

function taskMeta(task: OverviewTask): string {
  return [task.enabled === false ? '已停用' : '已启用', task.last_run_message || ''].filter(Boolean).join(' · ')
}
</script>