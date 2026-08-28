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
    expect(source).toContain('latencyBarHeight(point.latency_ms')
    expect(source).toContain('formatTrendHealthRate(account.local_account_id)')
    expect(source).toContain('statusTone(point.status)')
  })

  it('renders an enlarged chart for each account health trend', () => {
    expect(source).toContain('const TREND_BAR_COUNT = 28')
    expect(source).toContain('.sp-health-trend-cell { display: grid; min-width: 22rem;')
    expect(source).toContain('.sp-health-trend-bars { display: flex; align-items: end; gap: 0.2rem; height: 6.5rem;')
  })

  it('renders the account list before the per-account trends finish loading', () => {
    const loadAccountsSource = source.match(/async function loadAccounts\(\) \{([\s\S]*?)\n\}/)?.[1] || ''

    expect(loadAccountsSource).toContain('void loadAccountTrends(accounts.value, selectedRange.value)')
    expect(loadAccountsSource).not.toContain('await loadAccountTrends(accounts.value, selectedRange.value)')
  })

  it('limits concurrent trend requests to avoid a request burst on first load', () => {
    expect(source).toContain('const TREND_LOAD_CONCURRENCY = 6')
    expect(source).toContain('const workerCount = Math.min(TREND_LOAD_CONCURRENCY, ids.length)')
    expect(source).toContain('while (nextIndex < ids.length)')
    expect(source).not.toContain('Promise.all(ids.map(async accountId =>')
  })

  it('\u65b0\u589e\u4e0a\u6e38\u500d\u7387\u5217\u5e76\u652f\u6301\u8d26\u53f7\u500d\u7387\u5065\u5eb7\u72b6\u6001\u548c\u5065\u5eb7\u8d8b\u52bf\u6392\u5e8f', () => {
    expect(apiSource).toContain('rate_multiplier: number')
    expect(source).toContain("{ key: 'rate_multiplier', label: '\u4e0a\u6e38\u500d\u7387', sortable: true")
    expect(source).toContain('formatAccountRateMultiplier(account.rate_multiplier)')
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

})
