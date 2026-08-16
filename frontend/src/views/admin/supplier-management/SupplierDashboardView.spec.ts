import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  getOverviewMock,
  getAccountsMock,
  getRatesMock,
  getProvidersMock,
  getAccountTrafficMock,
  getAccountProfitRankingMock,
  getAccountHealthTimelineMock,
  pushMock,
} = vi.hoisted(() => ({
  getOverviewMock: vi.fn(),
  getAccountsMock: vi.fn(),
  getRatesMock: vi.fn(),
  getProvidersMock: vi.fn(),
  getAccountTrafficMock: vi.fn(),
  getAccountProfitRankingMock: vi.fn(),
  getAccountHealthTimelineMock: vi.fn(),
  pushMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

vi.mock('@/api/admin/supplierDashboard', () => ({
  getOverview: getOverviewMock,
  getAccounts: getAccountsMock,
  getRates: getRatesMock,
  getProviders: getProvidersMock,
  getAccountTraffic: getAccountTrafficMock,
  getAccountProfitRanking: getAccountProfitRankingMock,
  getAccountHealthTimeline: getAccountHealthTimelineMock,
  supplierDashboardAPI: {
    getOverview: getOverviewMock,
    getAccounts: getAccountsMock,
    getRates: getRatesMock,
    getProviders: getProvidersMock,
    getAccountTraffic: getAccountTrafficMock,
    getAccountProfitRanking: getAccountProfitRankingMock,
    getAccountHealthTimeline: getAccountHealthTimelineMock,
  },
}))

vi.mock('vue-chartjs', () => ({
  Line: { name: 'Line', props: ['data', 'options'], template: '<div data-test="line-chart" />' },
}))

vi.mock('chart.js', () => ({
  Chart: { register: vi.fn() },
  CategoryScale: {},
  LinearScale: {},
  PointElement: {},
  LineElement: {},
  Title: {},
  Tooltip: {},
  Legend: {},
}))

import SupplierDashboardView from './SupplierDashboardView.vue'

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

const overviewResponse = {
  range: '24h' as const,
  summary: {
    provider_count: 6,
    disabled_provider_count: 2,
    matched_account_count: 68,
    pending_account_count: 4,
    rate_risk_count: 6,
    model_count: 40,
  },
  stability: {
    request_count: 120000,
    success_count: 116640,
    error_count: 3360,
    success_rate: 97.2,
    error_rate: 2.8,
    p95_latency_ms: 820,
    health_score: 85,
  },
  cost: {
    period_cost: 3860,
    total_balance: 4820,
    estimated_days: 12.4,
    anomaly_providers: 1,
  },
  issues: [],
  tasks: [
    {
      key: 'account_sync',
      name: '上游账号同步',
      enabled: true,
      last_run_at: '2026-07-25T09:40:00Z',
      last_run_status: 'success',
      last_run_message: 'ok',
      next_run_at: '2026-07-25T09:55:00Z',
      affected_count: 6,
      settings_path: '/admin/supplier-management/automation',
    },
    {
      key: 'balance_sample',
      name: '余额采样',
      enabled: true,
      last_run_at: '2026-07-25T09:21:00Z',
      last_run_status: 'failed',
      last_run_message: '连续失败 2 次',
      next_run_at: '2026-07-25T09:51:00Z',
      affected_count: 7,
      settings_path: '/admin/supplier-management/automation?task=balance_sample',
    },
  ],
  provider_rankings: [],
  model_rankings: [],
  trends: [],
  warnings: [{ source: 'usage', message: '部分用量延迟 2 分钟' }],
  generated_at: '2026-07-25T09:42:18Z',
}

const accountsResponse = {
  range: '24h' as const,
  items: [
    {
      account_id: 12,
      account_name: 'prod-cn-02',
      provider_slug: 'volc',
      provider_name: '火山引擎',
      group_key: 'ent-a',
      group_name: '企业 A 组',
      severity: 'critical' as const,
      risk_types: ['critical', 'traffic', 'rate_up', 'not_lowest'],
      request_count: 12684,
      success_rate: 71.4,
      current_rate: 1.25,
      lowest_rate: 1.1,
      rate_delta_percent: 13.6,
      balance: 326,
      balance_currency: 'CNY',
      estimated_days: 0.8,
      status: '测试失败',
      reason: '账号测试连续失败 3 次，但近 24 小时仍承载流量。',
      period_cost: 240.5,
      estimated_extra_cost: 31.2,
      traffic_impact: 38,
      detected_at: '2026-07-25T08:00:00Z',
      target_path: '/admin/supplier-management/accounts?account_id=12',
    },
    {
      account_id: 15,
      account_name: 'gpt-primary',
      provider_slug: 'new-api',
      provider_name: 'New API',
      group_key: 'vip-3',
      group_name: 'vip-3',
      severity: 'critical' as const,
      risk_types: ['critical', 'traffic', 'balance'],
      request_count: 0,
      success_rate: 98.8,
      current_rate: 1.1,
      lowest_rate: 1.1,
      rate_delta_percent: 0,
      balance: null,
      balance_currency: null,
      estimated_days: 0.7,
      status: '余额紧急',
      reason: '余额预计不足 20 小时。',
      period_cost: null,
      estimated_extra_cost: null,
      traffic_impact: 64,
      detected_at: '2026-07-25T08:10:00Z',
      target_path: '/admin/supplier-management/accounts?account_id=15',
    },
  ],
  total: 3,
  page: 1,
  page_size: 20,
  warnings: [],
  generated_at: '2026-07-25T09:42:18Z',
}

const ratesResponse = {
  range: '24h' as const,
  items: [
    {
      provider_slug: 'openrouter',
      provider_name: 'OpenRouter',
      group_key: 'claude',
      group_name: 'Claude 专线',
      enabled_account_count: 4,
      current_account_id: 21,
      current_account_name: 'claude-route-01',
      current_rate: 1.18,
      lowest_rate: 1.05,
      lowest_account_ids: [22],
      lowest_account_names: ['route-03'],
      rate_delta_percent: 12.4,
      estimated_extra_cost: 42.6,
      cost_currency: 'CNY',
      comparison_status: 'not_lowest' as const,
      last_synced_at: '2026-07-25T09:30:00Z',
      target_path: '/admin/supplier-management/groups?provider=openrouter&group=claude',
    },
  ],
  total: 1,
  page: 1,
  page_size: 20,
  warnings: [],
  generated_at: '2026-07-25T09:42:18Z',
}

const providersResponse = {
  range: '24h' as const,
  items: [
    {
      provider_slug: 'volc',
      provider_name: '火山引擎',
      enabled: true,
      status: 'high_risk' as const,
      critical_issue_count: 2,
      enabled_account_count: 24,
      schedulable_account_count: 18,
      request_count: 50000,
      success_rate: 89.1,
      period_cost: 2408,
      cost_currency: 'CNY',
      balance: 800,
      balance_currency: 'CNY',
      estimated_days: 3.2,
      rate_risk_count: 3,
      balance_risk: true,
      sync_risk: false,
      target_path: '/admin/supplier-management/providers?slug=volc',
    },
    {
      provider_slug: 'self-hosted',
      provider_name: '自建中转',
      enabled: true,
      status: 'healthy' as const,
      critical_issue_count: 0,
      enabled_account_count: 14,
      schedulable_account_count: 13,
      request_count: 0,
      success_rate: 99.6,
      period_cost: 0,
      cost_currency: 'CNY',
      balance: null,
      balance_currency: null,
      estimated_days: null,
      rate_risk_count: 0,
      balance_risk: false,
      sync_risk: false,
      target_path: '/admin/supplier-management/providers?slug=self-hosted',
    },
  ],
  total: 2,
  page: 1,
  page_size: 20,
  warnings: [],
  generated_at: '2026-07-25T09:42:18Z',
}

const trafficResponse = {
  range: '30d' as const,
  series: [
    { time: '2026-07-24T00:00:00Z', requests: 1200, tokens: 1800000 },
    { time: '2026-07-24T01:00:00Z', requests: 2400, tokens: 3600000 },
    { time: '2026-07-24T02:00:00Z', requests: 900, tokens: 1200000 },
  ],
  accounts: [
    {
      account_id: 12,
      account_name: 'prod-cn-02',
      provider_slug: 'volc',
      provider_name: '火山引擎',
      group_key: 'ent-a',
      group_name: '企业 A 组',
    },
  ],
  warnings: [],
  generated_at: '2026-07-25T09:42:18Z',
}

const profitResponse = {
  items: [
    {
      account_id: 12,
      account_name: 'prod-cn-02',
      provider_slug: 'volc',
      provider_name: '火山引擎',
      group_key: 'ent-a',
      group_name: '企业 A 组',
      requests: 12684,
      tokens: 9000000,
      actual_cost: 240.5,
      user_cost: 310,
      profit: 69.5,
    },
    {
      account_id: 15,
      account_name: 'gpt-primary',
      provider_slug: 'new-api',
      provider_name: 'New API',
      group_key: 'vip-3',
      group_name: 'vip-3',
      requests: 5000,
      tokens: 2000000,
      actual_cost: 80,
      user_cost: 60,
      profit: -20,
    },
    {
      account_id: 16,
      account_name: 'free-tier',
      provider_slug: 'openrouter',
      provider_name: 'OpenRouter',
      group_key: 'free',
      group_name: '免费额度',
      requests: 300,
      tokens: 100000,
      actual_cost: 50,
      user_cost: 0,
      profit: 0,
    },
  ],
  warnings: [],
  generated_at: '2026-07-25T09:42:18Z',
}

const healthTimelineResponse = {
  range: '30d' as const,
  accounts: [
    {
      account_id: 12,
      account_name: 'prod-cn-02',
      provider_slug: 'volc',
      provider_name: '火山引擎',
      group_key: 'ent-a',
      group_name: '企业 A 组',
      cells: [
        { time: '2026-07-24T00:00:00Z', status: 'healthy' },
        { time: '2026-07-24T01:00:00Z', status: 'failed' },
        { time: '2026-07-24T02:00:00Z', status: 'slow' },
      ],
    },
  ],
  hours: [
    {
      time: '2026-07-24T00:00:00Z',
      status_counts: { healthy: 1, slow: 0, failed: 0, unavailable: 0, skipped: 0 },
      total: 1,
    },
    {
      time: '2026-07-24T01:00:00Z',
      status_counts: { healthy: 0, slow: 0, failed: 1, unavailable: 0, skipped: 0 },
      total: 1,
    },
    {
      time: '2026-07-24T02:00:00Z',
      status_counts: { healthy: 0, slow: 1, failed: 0, unavailable: 0, skipped: 0 },
      total: 1,
    },
  ],
  warnings: [],
  generated_at: '2026-07-25T09:42:18Z',
}

const emptyAccounts = {
  ...accountsResponse,
  items: [],
  total: 0,
}

async function mountView(): Promise<VueWrapper<any>> {
  const wrapper = mount(SupplierDashboardView, {
    global: {
      stubs: {
        SupplierModuleLayout: { template: '<div data-test="layout"><slot /></div>' },
        SupplierDrawer: {
          props: ['show', 'title', 'eyebrow'],
          emits: ['close'],
          template: '<div v-if="show" data-test="drawer"><slot /></div>',
        },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('SupplierDashboardView real data', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    pushMock.mockReset()
    getOverviewMock.mockReset()
    getAccountsMock.mockReset()
    getRatesMock.mockReset()
    getProvidersMock.mockReset()
    getAccountTrafficMock.mockReset()
    getAccountProfitRankingMock.mockReset()
    getAccountHealthTimelineMock.mockReset()
    getOverviewMock.mockResolvedValue(overviewResponse)
    getAccountsMock.mockResolvedValue(accountsResponse)
    getRatesMock.mockResolvedValue(ratesResponse)
    getProvidersMock.mockResolvedValue(providersResponse)
    getAccountTrafficMock.mockResolvedValue(trafficResponse)
    getAccountProfitRankingMock.mockResolvedValue(profitResponse)
    getAccountHealthTimelineMock.mockResolvedValue(healthTimelineResponse)
  })

  it('loads overview/accounts/rates/providers in parallel on mount with defaults', async () => {
    const wrapper = await mountView()

    expect(getOverviewMock).toHaveBeenCalledTimes(1)
    expect(getAccountsMock).toHaveBeenCalledTimes(1)
    expect(getRatesMock).toHaveBeenCalledTimes(1)
    expect(getProvidersMock).toHaveBeenCalledTimes(1)

    const overviewArgs = getOverviewMock.mock.calls[0]
    expect(overviewArgs[0]).toBe('24h')
    expect(overviewArgs[1]?.signal).toBeInstanceOf(AbortSignal)

    expect(getAccountsMock.mock.calls[0][0]).toMatchObject({
      range: '24h',
      risk_type: 'critical',
      page: 1,
      page_size: 20,
    })
    expect(getRatesMock.mock.calls[0][0]).toMatchObject({
      range: '24h',
      view: 'risk',
      page: 1,
      page_size: 20,
    })
    expect(getProvidersMock.mock.calls[0][0]).toMatchObject({
      range: '24h',
      page: 1,
      page_size: 20,
    })

    expect(wrapper.text()).toContain('prod-cn-02')
    expect(wrapper.text()).toContain('火山引擎')
    expect(wrapper.text()).toContain('OpenRouter')
    expect(wrapper.text()).toContain('上游账号同步')
    expect(wrapper.text()).not.toContain('gemini-backup')
  })

  it('reloads all sections when range switches to 7d', async () => {
    const wrapper = await mountView()
    getOverviewMock.mockClear()
    getAccountsMock.mockClear()
    getRatesMock.mockClear()
    getProvidersMock.mockClear()

    await wrapper.get('[data-test="range-7d"]').trigger('click')
    await flushPromises()

    expect(getOverviewMock).toHaveBeenCalledWith('7d', expect.objectContaining({ signal: expect.any(AbortSignal) }))
    expect(getAccountsMock.mock.calls[0][0]).toMatchObject({ range: '7d', risk_type: 'critical' })
    expect(getRatesMock.mock.calls[0][0]).toMatchObject({ range: '7d', view: 'risk' })
    expect(getProvidersMock.mock.calls[0][0]).toMatchObject({ range: '7d' })
  })

  it('reloads all sections on refresh', async () => {
    const wrapper = await mountView()
    const before = getOverviewMock.mock.calls.length

    await wrapper.get('[data-test="refresh"]').trigger('click')
    await flushPromises()

    expect(getOverviewMock.mock.calls.length).toBeGreaterThan(before)
    expect(getAccountsMock).toHaveBeenCalled()
    expect(getRatesMock).toHaveBeenCalled()
    expect(getProvidersMock).toHaveBeenCalled()
  })

  it('reloads accounts only when risk filter changes', async () => {
    const wrapper = await mountView()
    getOverviewMock.mockClear()
    getAccountsMock.mockClear()
    getRatesMock.mockClear()
    getProvidersMock.mockClear()
    getAccountsMock.mockResolvedValueOnce({ ...accountsResponse, total: 4, items: accountsResponse.items })

    await wrapper.get('[data-test="risk-balance"]').trigger('click')
    await flushPromises()

    expect(getAccountsMock).toHaveBeenCalledTimes(1)
    expect(getAccountsMock.mock.calls[0][0]).toMatchObject({ risk_type: 'balance', range: '24h' })
    expect(getOverviewMock).not.toHaveBeenCalled()
    expect(getRatesMock).not.toHaveBeenCalled()
    expect(getProvidersMock).not.toHaveBeenCalled()
  })

  it('reloads rates only when rate tab changes', async () => {
    const wrapper = await mountView()
    getOverviewMock.mockClear()
    getAccountsMock.mockClear()
    getRatesMock.mockClear()
    getProvidersMock.mockClear()

    await wrapper.get('[data-test="rate-tab-changed"]').trigger('click')
    await flushPromises()

    expect(getRatesMock).toHaveBeenCalledTimes(1)
    expect(getRatesMock.mock.calls[0][0]).toMatchObject({ view: 'changed', range: '24h' })
    expect(getAccountsMock).not.toHaveBeenCalled()
    expect(getOverviewMock).not.toHaveBeenCalled()
  })

  it('shows section empty states without falling back to demo rows', async () => {
    getAccountsMock.mockResolvedValue(emptyAccounts)
    getRatesMock.mockResolvedValue({ ...ratesResponse, items: [], total: 0 })
    getProvidersMock.mockResolvedValue({ ...providersResponse, items: [], total: 0 })
    getAccountTrafficMock.mockResolvedValue({ ...trafficResponse, series: [], accounts: [] })
    getAccountProfitRankingMock.mockResolvedValue({ ...profitResponse, items: [] })
    getAccountHealthTimelineMock.mockResolvedValue({ ...healthTimelineResponse, accounts: [], hours: [] })
    getOverviewMock.mockResolvedValue({
      ...overviewResponse,
      tasks: [],
      summary: { ...overviewResponse.summary, matched_account_count: 0 },
      stability: { ...overviewResponse.stability, health_score: 0 },
    })

    const wrapper = await mountView()

    expect(wrapper.get('[data-test="accounts-empty"]').text()).toMatch(/暂无|没有/)
    expect(wrapper.get('[data-test="rates-empty"]').text()).toMatch(/暂无|没有/)
    expect(wrapper.get('[data-test="providers-empty"]').text()).toMatch(/暂无|没有/)
    expect(wrapper.text()).not.toContain('gemini-backup')
    expect(wrapper.text()).not.toContain('prod-cn-02')
    expect(wrapper.text()).not.toContain('执行账号巡检演示已提交')
  })

  it('shows independent section errors and keeps other sections', async () => {
    getAccountsMock.mockRejectedValueOnce(new Error('账号区失败'))
    getRatesMock.mockResolvedValue(ratesResponse)
    getProvidersMock.mockResolvedValue(providersResponse)
    getOverviewMock.mockResolvedValue(overviewResponse)

    const wrapper = await mountView()

    expect(wrapper.get('[data-test="accounts-error"]').text()).toContain('账号区失败')
    expect(wrapper.text()).toContain('OpenRouter')
    expect(wrapper.text()).toContain('火山引擎')
    expect(wrapper.text()).toContain('上游账号同步')
  })

  it('drills down with router.push only and has no write-action demo button', async () => {
    const wrapper = await mountView()

    expect(wrapper.text()).not.toContain('执行账号巡检')

    await wrapper.get('[data-test="account-row-12"]').trigger('click')
    await flushPromises()
    expect(pushMock).toHaveBeenCalledWith('/admin/supplier-management/accounts?account_id=12')

    pushMock.mockClear()
    await wrapper.get('[data-test="provider-card-volc"]').trigger('click')
    await flushPromises()
    expect(pushMock).toHaveBeenCalledWith('/admin/supplier-management/providers?slug=volc')

    pushMock.mockClear()
    await wrapper.get('[data-test="rate-row-openrouter-claude"]').trigger('click')
    await flushPromises()
    expect(pushMock).toHaveBeenCalledWith('/admin/supplier-management/groups?provider=openrouter&group=claude')

    pushMock.mockClear()
    await wrapper.get('[data-test="task-action-balance_sample"]').trigger('click')
    await flushPromises()
    expect(pushMock).toHaveBeenCalledWith('/admin/supplier-management/automation?task=balance_sample')
  })

  it('ignores stale responses when a newer request sequence wins', async () => {
    const firstOverview = createDeferred<typeof overviewResponse>()
    const secondOverview = createDeferred<typeof overviewResponse>()
    getOverviewMock
      .mockImplementationOnce(() => firstOverview.promise)
      .mockImplementationOnce(() => secondOverview.promise)
    getAccountsMock.mockResolvedValue(accountsResponse)
    getRatesMock.mockResolvedValue(ratesResponse)
    getProvidersMock.mockResolvedValue(providersResponse)

    const wrapper = mount(SupplierDashboardView, {
      global: {
        stubs: {
          SupplierModuleLayout: { template: '<div data-test="layout"><slot /></div>' },
          SupplierDrawer: {
            props: ['show', 'title'],
            template: '<div v-if="show" data-test="drawer"><slot /></div>',
          },
        },
      },
    })

    await wrapper.get('[data-test="range-7d"]').trigger('click')
    secondOverview.resolve({
      ...overviewResponse,
      range: '7d',
      generated_at: '2026-07-25T10:00:00Z',
      warnings: [{ source: 'latest', message: '最新 7d 数据' }],
    })
    await flushPromises()
    firstOverview.resolve({
      ...overviewResponse,
      range: '24h',
      generated_at: '2026-07-25T09:00:00Z',
      warnings: [{ source: 'stale', message: '过期 24h 数据' }],
    })
    await flushPromises()

    expect(wrapper.text()).toContain('最新 7d 数据')
    expect(wrapper.text()).not.toContain('过期 24h 数据')
  })

  it('renders null metrics as dash and zero metrics as zero', async () => {
    const wrapper = await mountView()
    const row = wrapper.get('[data-test="account-row-15"]')
    expect(row.text()).toMatch(/0/)
    expect(row.text()).toContain('—')
  })

  it('renders traffic chart, profit ranking and health timeline sections with data', async () => {
    const wrapper = await mountView()

    // 时间流量：双指标折线图 + 汇总脚注
    expect(wrapper.get('[data-test="traffic-section"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="line-chart"]').exists()).toBe(true)
    const trafficFoot = wrapper.get('[data-test="traffic-summary"]').text()
    expect(trafficFoot).toContain('覆盖账号 1 个')
    expect(trafficFoot).toContain('4,500')
    expect(trafficFoot).toContain('660万')

    // 盈利排行：按账号渲染利润与成本
    const profitRow = wrapper.get('[data-test="profit-row-12"]')
    expect(profitRow.text()).toContain('prod-cn-02')
    expect(profitRow.text()).toContain('69.5')
    expect(wrapper.get('[data-test="profit-row-15"]').text()).toContain('gpt-primary')

    // 健康时间线：按账号状态分级的折线图
    expect(wrapper.get('[data-test="health-chart"]').exists()).toBe(true)
  })

  it('shows profit margin, refresh time, bucket label and accessible annotations', async () => {
    const wrapper = await mountView()

    // 利润率列：正负利润按 user_cost 计算百分比
    expect(wrapper.get('[data-test="profit-row-12"]').text()).toContain('22.4%')
    expect(wrapper.get('[data-test="profit-row-15"]').text()).toContain('-33.3%')
    // user_cost 为 0 时无法计算利润率，显示占位符
    expect(wrapper.get('[data-test="profit-row-16"]').text()).toContain('—')

    // 刷新时间辅助信息
    expect(wrapper.get('[data-test="refresh-meta"]').text()).toContain('最近刷新')

    // 30 天趋势默认按 6 小时分桶
    expect(wrapper.text()).toContain('每 6 小时')

    // 风险卡具备按钮语义
    expect(wrapper.get('[data-test="risk-critical"]').attributes('role')).toBe('button')
    expect(wrapper.get('[data-test="risk-critical"]').attributes('tabindex')).toBe('0')
  })

  it('renders health timeline as per-account status line chart', async () => {
    const wrapper = await mountView()
    // 流量图 + 健康时间线图共两张折线图
    expect(wrapper.findAll('[data-test="line-chart"]').length).toBe(2)
    expect(wrapper.get('[data-test="health-chart"]').exists()).toBe(true)
    // 原逐时段点阵表已移除
    expect(wrapper.find('[data-test="health-row-12"]').exists()).toBe(false)
    // 账号数量较少（<=10）时显示折线图例，不显示账号过多的提示
    expect(wrapper.find('[data-test="health-chart-note"]').exists()).toBe(false)
  })

  it('keeps trend range independent from the main range', async () => {
    const wrapper = await mountView()

    // 首次加载：主接口 24h，趋势接口 30d
    expect(getAccountTrafficMock.mock.calls[0][0]).toMatchObject({ range: '30d' })
    expect(getAccountProfitRankingMock.mock.calls[0][0]).toMatchObject({ range: '30d' })
    expect(getAccountHealthTimelineMock.mock.calls[0][0]).toMatchObject({ range: '30d', limit: 30, buckets: 120, bucket_hours: 6 })

    getOverviewMock.mockClear()
    getAccountsMock.mockClear()
    getRatesMock.mockClear()
    getProvidersMock.mockClear()
    getAccountTrafficMock.mockClear()
    getAccountProfitRankingMock.mockClear()
    getAccountHealthTimelineMock.mockClear()

    // 主范围切到 7d：旧接口用 7d，趋势接口仍保持 30d
    await wrapper.get('[data-test="range-7d"]').trigger('click')
    await flushPromises()
    expect(getOverviewMock).toHaveBeenCalledWith('7d', expect.objectContaining({ signal: expect.any(AbortSignal) }))
    expect(getAccountTrafficMock.mock.calls[0][0]).toMatchObject({ range: '30d' })
    expect(getAccountProfitRankingMock.mock.calls[0][0]).toMatchObject({ range: '30d' })
    expect(getAccountHealthTimelineMock.mock.calls[0][0]).toMatchObject({ range: '30d', limit: 30, buckets: 120, bucket_hours: 6 })

    getOverviewMock.mockClear()
    getAccountsMock.mockClear()
    getRatesMock.mockClear()
    getProvidersMock.mockClear()
    getAccountTrafficMock.mockClear()
    getAccountProfitRankingMock.mockClear()
    getAccountHealthTimelineMock.mockClear()

    // 趋势范围切到 7d：仅趋势接口重载，旧接口不重载
    await wrapper.get('[data-test="trend-range-7d"]').trigger('click')
    await flushPromises()
    expect(getAccountTrafficMock.mock.calls[0][0]).toMatchObject({ range: '7d' })
    expect(getAccountProfitRankingMock.mock.calls[0][0]).toMatchObject({ range: '7d' })
    expect(getAccountHealthTimelineMock.mock.calls[0][0]).toMatchObject({ range: '7d', limit: 30, buckets: 168, bucket_hours: 1 })
    expect(getOverviewMock).not.toHaveBeenCalled()
    expect(getAccountsMock).not.toHaveBeenCalled()
    expect(getRatesMock).not.toHaveBeenCalled()
    expect(getProvidersMock).not.toHaveBeenCalled()
  })

  it('shows empty states for traffic, profit ranking and health timeline', async () => {
    getAccountTrafficMock.mockResolvedValue({ ...trafficResponse, series: [], accounts: [] })
    getAccountProfitRankingMock.mockResolvedValue({ ...profitResponse, items: [] })
    getAccountHealthTimelineMock.mockResolvedValue({ ...healthTimelineResponse, accounts: [], hours: [] })

    const wrapper = await mountView()

    expect(wrapper.get('[data-test="traffic-empty"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="profit-empty"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="health-timeline-empty"]').exists()).toBe(true)
  })

  it('shows independent errors for traffic, profit ranking and health timeline', async () => {
    getAccountTrafficMock.mockRejectedValueOnce(new Error('流量加载失败'))
    getAccountProfitRankingMock.mockRejectedValueOnce(new Error('排行加载失败'))
    getAccountHealthTimelineMock.mockRejectedValueOnce(new Error('时间线加载失败'))

    const wrapper = await mountView()

    expect(wrapper.get('[data-test="traffic-error"]').text()).toContain('流量加载失败')
    expect(wrapper.get('[data-test="profit-error"]').text()).toContain('排行加载失败')
    expect(wrapper.get('[data-test="health-timeline-error"]').text()).toContain('时间线加载失败')
    // 其他区块保持正常渲染
    expect(wrapper.get('[data-test="accounts-section"]').exists()).toBe(true)
  })
})
