import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierAccountHealthView.vue'),
  'utf8'
)

const apiSource = readFileSync(
  resolve(process.cwd(), 'src/api/admin/supplierAccountHealth.ts'),
  'utf8'
)

const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')

describe('SupplierAccountHealthView', () => {
  it('provides an independent account health trend route and API', () => {
    expect(routerSource).toContain("name: 'SupplierAccountHealth'")
    expect(routerSource).toContain("path: '/admin/supplier-management/account-health'")
    expect(apiSource).toContain("'/admin/supplier-management/account-health/accounts'")
    expect(apiSource).toContain("'/admin/supplier-management/account-health/trend'")
  })

  it('renders account filters, health status, latency charts, and range switching', () => {
    expect(source).toContain('<SupplierModuleLayout>')
    expect(source).toContain('供应商筛选')
    expect(source).toContain('当前健康状态')
    expect(source).toContain('账号名称或 ID')
    expect(source).toContain("'24h'")
    expect(source).toContain("'7d'")
    expect(source).toContain("'30d'")
    expect(source).toContain('健康状态趋势')
    expect(source).toContain('响应时间趋势')
    expect(source).toContain('<Line')
  })

  it('selects the account from account_id and avoids rendering failed latency as zero', () => {
    expect(source).toContain("route.query.account_id")
    expect(source).toContain('selectedAccountId')
    expect(source).toContain('latency_ms === null')
    expect(source).toContain('尚无健康检测记录')
  })

  it('exposes diagnostic details and uses the global error toast', () => {
    expect(source).toContain('失败原因')
    expect(source).toContain('动作')
    expect(source).toContain('错误详情')
    expect(source).toContain('useAppStore')
    expect(source).toContain('showError')
    expect(source).toContain('extractApiErrorMessage')
    expect(source).not.toContain('sp-alert')
  })

  it('\u5408\u5e76\u8d26\u53f7\u4f9b\u5e94\u5546\u5e73\u53f0\u4fe1\u606f\u5e76\u5c55\u793a\u6309\u72b6\u6001\u4e0e\u54cd\u5e94\u65f6\u95f4\u7f16\u7801\u7684\u5065\u5eb7\u8d8b\u52bf', () => {
    expect(source).toContain('\u8d26\u53f7 / \u4f9b\u5e94\u5546 / \u5e73\u53f0')
    expect(source).toContain('providerTone(account.provider_name)')
    expect(source).toContain('platformTone(account.platform)')
    expect(source).toContain('healthTrendByAccountId')
    expect(source).toContain('sp-health-trend-bar')
    expect(source).toContain('latencyBarHeight(point, account)')
    expect(source).toContain('formatTrendHealthRate(account.local_account_id)')
    expect(source).toContain('statusTone(point.status)')
  })

  it('renders an enlarged chart for each account health trend', () => {
    expect(source).toContain('const TREND_BAR_COUNT = 96')
    expect(source).toContain('.sp-health-trend-cell { display: grid; min-width: 20rem;')
    expect(source).toContain('.sp-health-trend-bars { display: flex; align-items: end; gap: 0.08rem; height: 6.5rem;')
    expect(source).not.toContain('slice(-TREND_BAR_COUNT)')
  })

  it('uses time-bucket samples for health rate and empty bucket display', () => {
    expect(apiSource).toContain('sample_count: number')
    expect(apiSource).toContain('healthy_count: number')
    expect(apiSource).toContain('latest_checked_at?: string')
    expect(source).toContain('point.sample_count')
    expect(source).toContain('trendSampleCount')
    expect(source).toContain('latest_checked_at')
  })

  it('renders the account list before the per-account trends finish loading', () => {
    const loadAccountsSource = source.match(/async function loadAccounts\(\) \{([\s\S]*?)\n\}/)?.[1] || ''

    expect(loadAccountsSource).toContain('void loadAccountTrends(accounts.value, selectedRange.value)')
    expect(loadAccountsSource).not.toContain('await loadAccountTrends(accounts.value, selectedRange.value)')
  })

  it('loads all account trends with one batch request to avoid a request burst', () => {
    expect(source).toContain('await getSupplierAccountHealthTrends(ids, range)')
    expect(source).not.toContain('TREND_LOAD_CONCURRENCY')
    expect(source).not.toContain('loadNextTrend')
  })

  it('\u62c6\u5206\u4e0a\u6e38\u500d\u7387\u4e0e\u6709\u6548\u500d\u7387\u4e24\u5217\u5e76\u652f\u6301\u6392\u5e8f', () => {
    expect(apiSource).toContain('upstream_rate_multiplier: number')
    expect(apiSource).toContain('effective_rate_multiplier: number')
    expect(source).toContain("{ key: 'upstream_rate_multiplier', label: '\u4e0a\u6e38\u500d\u7387', sortable: true")
    expect(source).toContain("{ key: 'effective_rate_multiplier', label: '\u6709\u6548\u500d\u7387', sortable: true")
    expect(source).toContain('formatAccountRateMultiplier(account.upstream_rate_multiplier)')
    expect(source).toContain('formatAccountRateMultiplier(account.effective_rate_multiplier)')
    expect(source).toContain("{ key: 'account_sort', label: '\u8d26\u53f7 / \u4f9b\u5e94\u5546 / \u5e73\u53f0', sortable: true }")
    expect(source).toContain("{ key: 'status_sort', label: '\u5f53\u524d\u5065\u5eb7\u72b6\u6001', sortable: true }")
    expect(source).toContain("{ key: 'health_trend_sort', label: '\u5065\u5eb7\u8d8b\u52bf', sortable: true }")
    expect(source).toContain('accountHealthSortData')
  })

  it('\u5728\u64cd\u4f5c\u5217\u6253\u5f00\u5f39\u7a97\u67e5\u770b\u8d26\u53f7\u5065\u5eb7\u8be6\u60c5', () => {
    expect(source).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(source).toContain("{ key: 'actions', label: '\u64cd\u4f5c'")
    expect(source).toContain('openHealthDetail(account.local_account_id)')
    expect(source).toContain('healthDetailVisible')
    expect(source).toContain('sp-health-detail-dialog')
    expect(source).toContain('Selected Account')
    expect(source).toContain('Health Trend')
    expect(source).toContain('Latency')
  })

  it('keeps diagnostics inside the detail dialog instead of auto-selecting the first account', () => {
    expect(source).toContain('sp-health-event-panel')
    expect(source).toContain('detailEventPoints')
    expect(source).not.toContain('accounts.value[0]')
    expect(source).not.toContain('function selectAccount')
  })

  it('builds provider and platform filters from independent catalogs', () => {
    expect(source).toContain('async function loadFilterOptions()')
    expect(source).toContain('supplierProvidersAPI.list(')
    expect(source).toContain('buildPlatformOptions(customPlatforms.value)')
  })

  it('shows the health guard hint from the global automation switch', () => {
    expect(source).toContain('guardTaskDisabled')
    expect(source).toContain('健康守护任务未启用')
    expect(source).toContain('/admin/supplier-management/automations')
  })

  it('normalizes trend bars by the latency threshold and draws a threshold line', () => {
    expect(source).toContain('const TREND_THRESHOLD_RATIO = 0.6')
    expect(source).toContain('sp-health-trend-bars--threshold')
    expect(source).toContain('阈值 {{ account.latency_limit_ms }} ms')
  })

  it('splits latency and checked time into two sortable columns', () => {
    expect(source).toContain("{ key: 'latency_ms', label: '最近响应', sortable: true")
    expect(source).toContain("{ key: 'checked_at_sort', label: '检测时间', sortable: true")
    expect(source).toContain('checked_at_sort: checkedAtSortValue(account.checked_at)')
    expect(source).toContain('function checkedAtSortValue(')
  })

  it('tones rate, latency and checked time cells by business thresholds', () => {
    expect(source).toContain('rateMultiplierTone(account.upstream_rate_multiplier)')
    expect(source).toContain('rateMultiplierTone(account.effective_rate_multiplier)')
    expect(source).toContain('latencyTone(account)')
    expect(source).toContain('checkedAtTone(account.checked_at)')
    expect(source).toContain('const RATE_BAD_THRESHOLD = 2')
    expect(source).toContain('const LATENCY_WARN_RATIO = 0.75')
    expect(source).toContain('const CHECKED_STALE_MS = 24 * 60 * 60 * 1000')
    expect(source).toContain('.sp-health-value.bad { color: var(--sp-red); }')
    expect(source).toContain('.dark .sp-health-value.good')
    expect(source).not.toContain('sp-health-latency')
  })

  it('explains list failures in the empty state and debounces search', () => {
    expect(source).toContain('账号健康列表加载失败')
    expect(source).toContain('重试')
    expect(source).toContain("import { useDebounceFn } from '@vueuse/core'")
    expect(source).toContain('debouncedSearchReload')
  })

  it('keeps the filter bar inside the account panel and drops the standalone filter panel', () => {
    expect(source).not.toContain('sp-health-filter-panel')
    expect(source).toContain('<div class="sp-health-filters" role="group" aria-label="账号健康筛选">')
    expect(source).toContain('sp-health-head-meta')
    expect(source).not.toContain('sp-data-note')
  })

  it('colors chips through a single theme-safe rule instead of hardcoded light backgrounds', () => {
    expect(source).toContain('--chip-hue')
    expect(source).toContain('.dark .sp-health-chip')
    expect(source).not.toContain('background: #eff6ff')
  })

  it('tones diagnostics cards by status and renders error details in monospace', () => {
    expect(source).toContain('sp-health-event--${statusTone(point.status)}')
    expect(source).toContain('.sp-health-event--bad { border-left-color: var(--sp-red); }')
    expect(source).toContain('.sp-health-latest-error strong,')
  })

  it('renders clickable health overview cards backed by a summary endpoint', () => {
    expect(apiSource).toContain("'/admin/supplier-management/account-health/summary'")
    expect(apiSource).toContain('unchecked: number')
    expect(source).toContain('<section class="sp-metric-grid sp-health-metrics" aria-label="账号健康概览">')
    expect(source).toContain('summaryMetrics')
    expect(source).toContain('selectHealthStatus(metric.key)')
    expect(source).toContain("{ value: 'unchecked', label: '未检测' }")
    expect(source).toContain('--sp-metric-accent')
  })

  it('keeps overview counts stable while switching the health status filter', () => {
    expect(source).toContain('watch([providerId, platform], () => {')
    expect(source).not.toContain('watch([providerId, platform, healthStatus]')
    const statusWatchSource = source.match(/watch\(healthStatus, \(\) => \{([\s\S]*?)\n\}\)/)?.[1] || ''
    expect(statusWatchSource).toContain('void loadAccounts()')
    expect(statusWatchSource).not.toContain('loadSummary')
  })

  it('renders the bound upstream monitor series alongside the guard series', () => {
    expect(apiSource).toContain('upstream_points?: SupplierAccountHealthPoint[]')
    expect(apiSource).toContain('upstream_latest?: SupplierAccountHealthPoint')
    expect(apiSource).toContain('upstream_monitors?: SupplierAccountHealthUpstreamMonitor[]')
    expect(apiSource).toContain("export type SupplierAccountHealthStatus = 'healthy' | 'slow' | 'failed' | 'unavailable'")
    expect(source).toContain("const UPSTREAM_SERIES_LABEL = '上游监控'")
    expect(source).toContain('label: UPSTREAM_SERIES_LABEL')
    expect(source).toContain('borderDash: [5, 4]')
    expect(source).toContain('hasAnyTrendSamples')
    expect(source).toContain('upstream_points || []')
    expect(source).toContain('trend.upstream_monitors || []')
  })

  it('indexes both trend series by dataset label instead of dataset index', () => {
    expect(source).toContain('function seriesPointAt(label: string | undefined, index: number)')
    expect(source).toContain('seriesPointAt(context.dataset.label, context.dataIndex)')
    expect(source).toContain("const LATENCY_THRESHOLD_LABEL = '慢响应阈值'")
    expect(source).toContain('context.dataset.label === LATENCY_THRESHOLD_LABEL')
    expect(source).not.toContain('context.datasetIndex')
  })

  it('paints the upstream status ribbon under the guard bars in the account list', () => {
    expect(source).toContain('upstreamTrendByAccountId')
    expect(source).toContain('function visibleUpstreamTrend(accountId: number)')
    expect(source).toContain('class="sp-health-upstream-strip"')
    expect(source).toContain('sp-health-upstream-cell--${statusTone(point.status)}')
    expect(source).toContain('upstreamTrendPointTitle(point)')
    expect(source).toContain('.sp-health-upstream-strip { display: flex; gap: 0.08rem; height: 0.3rem;')
    expect(source).toContain('.sp-health-upstream-cell { flex: 1 1 0; min-width: 0;')
    // 有色带时不能再显示「暂无趋势」，否则同一格里两种结论并存
    expect(source).toContain('v-else-if="!visibleUpstreamTrend(account.local_account_id).length"')
  })

  it('lists bound upstream monitors and separates no-binding from no-samples', () => {
    expect(source).toContain('v-if="upstreamMonitors.length"')
    expect(source).toContain('所选范围内暂无上报')
    expect(source).toContain('formatAvailability(monitor.availability_7d)')
    expect(source).toContain('monitor.monitor_name || monitor.monitor_key')
    expect(source).toContain('未标注主模型')
    expect(source).toContain("if (status === 'unavailable') return '上游未上报'")
    // 只有上游数据时守护明细整块不渲染，避免一屏「—」
    expect(source).toContain('v-else-if="hasTrendSamples" class="sp-health-latest"')
  })

  it('在监控趋势列中把健康守护图表与账号监控项数据分成两行', () => {
    expect(source).toContain('class="sp-health-trend-row sp-health-trend-row--guard"')
    expect(source).toContain('class="sp-health-trend-row sp-health-trend-row--monitor"')
    expect(source).toContain('健康守护')
    expect(source).toContain('账号监控项')
    expect(source).toContain('const upstreamMonitorsByAccountId')
    expect(source).toContain('function visibleUpstreamMonitors(accountId: number)')
    expect(source).toContain('formatAvailability(monitor.availability_7d)')
    expect(source).toContain('.sp-health-trend-row--monitor')
  })

})
