<template>
  <AppLayout>
    <div class="dashboard-page">
      <header class="dashboard-hero">
        <div>
          <span class="eyebrow">{{ t('admin.upstreamDashboard.eyebrow') }}</span>
          <h1>{{ t('admin.upstreamDashboard.title') }}</h1>
          <p>{{ healthSummary }}</p>
        </div>
        <div class="hero-actions">
          <div class="range-switch" aria-label="Dashboard range">
            <button data-test="range-24h" type="button" :class="{ active: range === '24h' }" @click="setRange('24h')">24H</button>
            <button data-test="range-7d" type="button" :class="{ active: range === '7d' }" @click="setRange('7d')">7D</button>
          </div>
          <button type="button" class="action-button" :disabled="loading" @click="loadDashboard">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
          <button data-test="run-health-guard" type="button" class="action-button action-primary" :disabled="runningHealthGuard" @click="runHealthGuard">
            <Icon name="shield" size="sm" :class="runningHealthGuard ? 'animate-pulse' : ''" />
            {{ runningHealthGuard ? t('admin.upstreamDashboard.inspecting') : t('admin.upstreamDashboard.inspect') }}
          </button>
        </div>
      </header>

      <div v-if="loading && !dashboard" class="loading-state">
        <span class="loading-orbit"></span>
        <strong>{{ t('common.loading') }}</strong>
      </div>

      <div v-else-if="loadError && !dashboard" class="error-state">
        <Icon name="exclamationTriangle" size="lg" />
        <h2>{{ t('admin.upstreamDashboard.loadFailed') }}</h2>
        <p>{{ loadError }}</p>
        <button type="button" class="action-button action-primary" @click="loadDashboard">{{ t('common.retry') }}</button>
      </div>

      <template v-else-if="dashboard">
        <div v-if="dashboard.warnings?.length" class="warning-strip">
          <Icon name="exclamationTriangle" size="sm" />
          <span>{{ t('admin.upstreamDashboard.partialData', { count: dashboard.warnings.length }) }}</span>
        </div>

        <section class="metric-layout">
          <UpstreamDashboardMetricGroup data-test="metric-resources" tone="blue" :title="t('admin.upstreamDashboard.resources')" :items="resourceMetrics" />
          <UpstreamDashboardMetricGroup data-test="metric-stability" tone="green" :title="t('admin.upstreamDashboard.stability')" :items="stabilityMetrics" />
          <UpstreamDashboardMetricGroup data-test="metric-cost" tone="amber" :title="t('admin.upstreamDashboard.costAndBalance')" :items="costMetrics" />
        </section>

        <section class="primary-layout">
          <article class="issues-panel">
            <div class="section-header">
              <div><span class="section-kicker">{{ t('admin.upstreamDashboard.priorityQueue') }}</span><h2>{{ t('admin.upstreamDashboard.needsAttention') }}</h2></div>
              <span class="issue-count">{{ dashboard.issues.length }}</span>
            </div>
            <div v-if="dashboard.issues.length" class="issue-list">
              <button
                v-for="issue in dashboard.issues"
                :key="issue.id"
                :data-test="`issue-${issue.id}`"
                type="button"
                :class="['issue-row', `issue-${issue.severity}`]"
                @click="selectedIssue = issue"
              >
                <span class="issue-severity">{{ severityLabel(issue.severity) }}</span>
                <span class="issue-copy"><strong>{{ issue.title }}</strong><small>{{ issue.description }}</small></span>
                <span class="issue-impact">{{ t('admin.upstreamDashboard.affected', { count: issue.impact_count }) }}</span>
                <Icon name="chevronRight" size="sm" />
              </button>
            </div>
            <div v-else class="empty-panel"><Icon name="checkCircle" size="lg" /><strong>{{ t('admin.upstreamDashboard.noIssues') }}</strong></div>
          </article>

          <UpstreamTaskStatusList
            :title="t('admin.upstreamDashboard.automations')"
            :empty-text="t('admin.upstreamDashboard.noTasks')"
            :tasks="dashboard.tasks"
            @open="openTask"
          />
        </section>

        <section class="analysis-layout">
          <article class="trend-panel">
            <div class="section-header">
              <div><span class="section-kicker">{{ t('admin.upstreamDashboard.costTrend') }}</span><h2>{{ t('admin.upstreamDashboard.recentConsumption') }}</h2></div>
              <strong>{{ formatCurrency(dashboard.cost.period_cost) }}</strong>
            </div>
            <div v-if="dashboard.trends.length" class="trend-bars">
              <div v-for="point in dashboard.trends" :key="point.date" class="trend-column">
                <div class="trend-track"><span :style="{ height: trendHeight(point.cost) }"></span></div>
                <small>{{ shortDate(point.date) }}</small>
              </div>
            </div>
            <div v-else class="empty-panel compact">{{ t('admin.upstreamDashboard.noTrend') }}</div>
          </article>

          <UpstreamRankingPanel
            :title="t('admin.upstreamDashboard.providerRanking')"
            :subtitle="t('admin.upstreamDashboard.byCost')"
            :empty-text="t('admin.upstreamDashboard.noRanking')"
            :items="providerRankingItems"
            @open="openRanking"
          />
          <UpstreamRankingPanel
            :title="t('admin.upstreamDashboard.modelRanking')"
            :subtitle="t('admin.upstreamDashboard.byRequests')"
            :empty-text="t('admin.upstreamDashboard.noRanking')"
            :items="modelRankingItems"
            @open="openRanking"
          />
        </section>

        <footer class="data-freshness">
          {{ t('admin.upstreamDashboard.generatedAt') }} {{ formatDateTime(dashboard.generated_at) }}
        </footer>
      </template>
    </div>

    <UpstreamIssueDrawer :issue="selectedIssue" @close="selectedIssue = null" @primary="handleIssuePrimary" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import type { UpstreamDashboardIssue, UpstreamDashboardRange, UpstreamDashboardResponse, UpstreamDashboardSeverity, UpstreamDashboardTask } from '@/api/admin/upstreamDashboard'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UpstreamDashboardMetricGroup, { type DashboardMetricItem } from '@/components/admin/upstream/UpstreamDashboardMetricGroup.vue'
import UpstreamIssueDrawer from '@/components/admin/upstream/UpstreamIssueDrawer.vue'
import UpstreamTaskStatusList from '@/components/admin/upstream/UpstreamTaskStatusList.vue'
import UpstreamRankingPanel, { type DashboardRankingItem } from '@/components/admin/upstream/UpstreamRankingPanel.vue'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const dashboard = ref<UpstreamDashboardResponse | null>(null)
const selectedIssue = ref<UpstreamDashboardIssue | null>(null)
const range = ref<UpstreamDashboardRange>('24h')
const loading = ref(false)
const runningHealthGuard = ref(false)
const loadError = ref('')

const healthSummary = computed(() => {
  if (!dashboard.value) return t('admin.upstreamDashboard.description')
  const count = dashboard.value.issues.filter(issue => issue.severity === 'critical' || issue.severity === 'high').length
  return count ? t('admin.upstreamDashboard.healthWarning', { count }) : t('admin.upstreamDashboard.healthGood')
})

const resourceMetrics = computed<DashboardMetricItem[]>(() => dashboard.value ? [
  { label: t('admin.upstreamDashboard.providers'), value: formatNumber(dashboard.value.summary.provider_count), hint: t('admin.upstreamDashboard.disabledCount', { count: dashboard.value.summary.disabled_provider_count }), hintTone: dashboard.value.summary.disabled_provider_count ? 'danger' : 'good' },
  { label: t('admin.upstreamDashboard.accounts'), value: formatNumber(dashboard.value.summary.matched_account_count), hint: t('admin.upstreamDashboard.pendingCount', { count: dashboard.value.summary.pending_account_count }), hintTone: dashboard.value.summary.pending_account_count ? 'warning' : 'good' },
  { label: t('admin.upstreamDashboard.models'), value: formatNumber(dashboard.value.summary.model_count), hint: t('admin.upstreamDashboard.rateRiskCount', { count: dashboard.value.summary.rate_risk_count }), hintTone: dashboard.value.summary.rate_risk_count ? 'warning' : 'good' },
] : [])

const stabilityMetrics = computed<DashboardMetricItem[]>(() => dashboard.value ? [
  { label: t('admin.upstreamDashboard.successRate'), value: formatPercent(dashboard.value.stability.success_rate), hint: t('admin.upstreamDashboard.healthScore', { value: dashboard.value.stability.health_score }), hintTone: dashboard.value.stability.success_rate >= 98 ? 'good' : 'warning' },
  { label: t('admin.upstreamDashboard.requests'), value: compactNumber(dashboard.value.stability.request_count), hint: range.value.toUpperCase() },
  { label: t('admin.upstreamDashboard.p95Latency'), value: `${formatNumber(dashboard.value.stability.p95_latency_ms)} ms`, hint: t('admin.upstreamDashboard.errors', { count: compactNumber(dashboard.value.stability.error_count) }), hintTone: dashboard.value.stability.error_count ? 'warning' : 'good' },
] : [])

const costMetrics = computed<DashboardMetricItem[]>(() => dashboard.value ? [
  { label: t('admin.upstreamDashboard.periodCost'), value: formatCurrency(dashboard.value.cost.period_cost), hint: range.value.toUpperCase() },
  { label: t('admin.upstreamDashboard.totalBalance'), value: formatCurrency(dashboard.value.cost.total_balance), hint: t('admin.upstreamDashboard.balanceAnomalies', { count: dashboard.value.cost.anomaly_providers }), hintTone: dashboard.value.cost.anomaly_providers ? 'warning' : 'good' },
  { label: t('admin.upstreamDashboard.estimatedDays'), value: dashboard.value.cost.estimated_days == null ? '-' : `${dashboard.value.cost.estimated_days.toFixed(1)} d`, hint: t('admin.upstreamDashboard.estimateHint') },
] : [])

const providerRankingItems = computed<DashboardRankingItem[]>(() => (dashboard.value?.provider_rankings || []).map(item => ({
  key: item.provider_slug, name: item.provider_name || item.provider_slug,
  meta: `${t('admin.upstreamDashboard.balance')} ${formatCurrency(item.balance)}`,
  value: formatCurrency(item.cost), target: `/admin/upstream-management/providers?provider=${encodeURIComponent(item.provider_slug)}`,
})))

const modelRankingItems = computed<DashboardRankingItem[]>(() => (dashboard.value?.model_rankings || []).map(item => ({
  key: item.model, name: item.model, meta: `${compactNumber(item.requests)} ${t('admin.upstreamDashboard.requestsUnit')}`,
  value: formatCurrency(item.cost), target: `/admin/upstream-management/model-square?model=${encodeURIComponent(item.model)}`,
})))

onMounted(loadDashboard)

async function loadDashboard() {
  loading.value = true
  loadError.value = ''
  try {
    dashboard.value = await adminAPI.upstreamDashboard.get(range.value)
  } catch (error) {
    loadError.value = error instanceof Error ? error.message : t('admin.upstreamDashboard.loadFailed')
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

async function setRange(value: UpstreamDashboardRange) {
  if (range.value === value) return
  range.value = value
  await loadDashboard()
}

async function runHealthGuard() {
  runningHealthGuard.value = true
  try {
    await adminAPI.upstreamAccountSync.runHealthGuardNow()
    appStore.showSuccess(t('admin.upstreamDashboard.inspectSuccess'))
    await loadDashboard()
  } catch (error) {
    appStore.showError(error instanceof Error ? error.message : t('admin.upstreamDashboard.inspectFailed'))
  } finally {
    runningHealthGuard.value = false
  }
}

function handleIssuePrimary(issue: UpstreamDashboardIssue) {
  selectedIssue.value = null
  if (issue.target_path) router.push(issue.target_path)
}

function openTask(task: UpstreamDashboardTask) { if (task.settings_path) router.push(task.settings_path) }
function openRanking(item: DashboardRankingItem) { if (item.target) router.push(item.target) }
function severityLabel(value: UpstreamDashboardSeverity) { return { critical: t('admin.upstreamDashboard.severityCritical'), high: t('admin.upstreamDashboard.severityHigh'), medium: t('admin.upstreamDashboard.severityMedium'), low: t('admin.upstreamDashboard.severityLow') }[value] }
function formatNumber(value: number) { return new Intl.NumberFormat().format(value || 0) }
function compactNumber(value: number) { return new Intl.NumberFormat(undefined, { notation: 'compact', maximumFractionDigits: 1 }).format(value || 0) }
function formatCurrency(value: number) { return new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 2 }).format(value || 0) }
function formatPercent(value: number) { return `${Number(value || 0).toFixed(1)}%` }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString() }
function shortDate(value: string) { const parts = value.split('-'); return parts.length >= 3 ? `${parts[1]}/${parts[2]}` : value }
function trendHeight(value: number) { const max = Math.max(...(dashboard.value?.trends || []).map(item => item.cost), 1); return `${Math.max(8, (value / max) * 100)}%` }
</script>

<style scoped>
.dashboard-page { @apply min-h-full bg-gray-50 px-4 py-6 dark:bg-gray-950 sm:px-6 lg:px-8; }
.dashboard-hero { @apply mx-auto mb-6 flex max-w-[1680px] flex-col gap-5 lg:flex-row lg:items-end lg:justify-between; }
.eyebrow, .section-kicker { @apply text-xs font-bold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400; }
.dashboard-hero h1 { @apply mt-2 text-3xl font-black tracking-tight text-gray-950 dark:text-white; }
.dashboard-hero p { @apply mt-2 text-sm text-gray-500 dark:text-gray-400; }
.hero-actions { @apply flex flex-wrap items-center gap-2; }
.range-switch { @apply flex min-h-11 rounded-xl border border-gray-200 bg-white p-1 shadow-sm dark:border-gray-700 dark:bg-gray-800; }
.range-switch button { @apply min-w-12 rounded-lg px-3 text-xs font-bold text-gray-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500; }
.range-switch button.active { @apply bg-gray-950 text-white dark:bg-white dark:text-gray-950; }
.action-button { @apply inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-gray-200 bg-white px-4 text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700; }
.action-primary { @apply border-transparent bg-primary-600 text-white hover:bg-primary-700 dark:bg-primary-600 dark:text-white; }
.metric-layout { @apply mx-auto grid max-w-[1680px] gap-4 xl:grid-cols-3; }
.primary-layout { @apply mx-auto mt-4 grid max-w-[1680px] gap-4 xl:grid-cols-[1.65fr_1fr]; }
.analysis-layout { @apply mx-auto mt-4 grid max-w-[1680px] gap-4 xl:grid-cols-[1.35fr_1fr_1fr]; }
.issues-panel, .trend-panel { @apply rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800; }
.section-header { @apply mb-4 flex items-center justify-between gap-4; }
.section-header h2 { @apply mt-1 text-lg font-bold text-gray-950 dark:text-white; }
.issue-count { @apply flex h-9 min-w-9 items-center justify-center rounded-xl bg-red-50 px-2 text-sm font-bold text-red-600 dark:bg-red-900/30 dark:text-red-300; }
.issue-list { @apply space-y-2; }
.issue-row { @apply grid min-h-[72px] w-full grid-cols-[84px_1fr_auto_18px] items-center gap-3 rounded-xl border border-gray-200 px-4 text-left transition-all hover:-translate-y-px hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-gray-700; }
.issue-critical { @apply border-l-4 border-l-red-500 bg-red-50/50 dark:bg-red-950/20; }
.issue-high { @apply border-l-4 border-l-orange-500 bg-orange-50/50 dark:bg-orange-950/20; }
.issue-medium { @apply border-l-4 border-l-amber-400; }
.issue-low { @apply border-l-4 border-l-gray-300; }
.issue-severity { @apply text-xs font-bold text-gray-600 dark:text-gray-300; }
.issue-copy { @apply min-w-0; }
.issue-copy strong { @apply block truncate text-sm text-gray-950 dark:text-white; }
.issue-copy small { @apply mt-1 block truncate text-xs text-gray-500; }
.issue-impact { @apply rounded-full bg-gray-100 px-2.5 py-1 text-xs font-semibold text-gray-600 dark:bg-gray-700 dark:text-gray-300; }
.trend-bars { @apply flex h-48 items-end gap-2 border-b border-gray-200 pt-5 dark:border-gray-700; }
.trend-column { @apply flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-2; }
.trend-track { @apply flex h-full w-full max-w-8 items-end overflow-hidden rounded-t-md bg-gray-100 dark:bg-gray-700; }
.trend-track span { @apply block w-full rounded-t-md bg-gradient-to-t from-primary-700 to-cyan-400; }
.trend-column small { @apply text-[10px] text-gray-400; }
.warning-strip { @apply mx-auto mb-4 flex max-w-[1680px] items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200; }
.empty-panel { @apply flex min-h-48 flex-col items-center justify-center gap-2 text-sm text-gray-400; }
.empty-panel.compact { @apply min-h-40; }
.loading-state, .error-state { @apply mx-auto flex min-h-[480px] max-w-[1680px] flex-col items-center justify-center gap-3 rounded-2xl border border-gray-200 bg-white text-gray-500 dark:border-gray-700 dark:bg-gray-800; }
.error-state h2 { @apply text-lg font-bold text-gray-900 dark:text-white; }
.error-state p { @apply max-w-lg text-center text-sm; }
.loading-orbit { @apply h-10 w-10 rounded-full border-4 border-gray-200 border-t-primary-600; animation: spin 800ms linear infinite; }
.data-freshness { @apply mx-auto max-w-[1680px] py-5 text-right text-xs text-gray-400; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 768px) { .issue-row { @apply grid-cols-[70px_1fr_18px]; } .issue-impact { @apply hidden; } .analysis-layout { @apply grid-cols-1; } }
@media (prefers-reduced-motion: reduce) { .loading-orbit { animation: none; } .issue-row { @apply hover:translate-y-0; } }
</style>
