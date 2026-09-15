import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierProvidersView from './SupplierProvidersView.vue'

const providerViewMocks = vi.hoisted(() => ({
  listProviders: vi.fn(),
  listCostTrends: vi.fn(),
  backfillCostTrends: vi.fn(),
  getBalanceSummary: vi.fn(),
  listProviderTypes: vi.fn(),
  updateProvider: vi.fn(),
  refreshToken: vi.fn(),
  getUpstreamSessions: vi.fn(),
  getUpstreamSessionDetails: vi.fn(),
  revokeUpstreamSessions: vi.fn(),
  getAuthStatus: vi.fn(),
  listAuthHistory: vi.fn(),
  syncProvider: vi.fn(),
  streamSupplierProviderSync: vi.fn(),
  testProviderEndpoint: vi.fn(),
  getCostDeviationSettings: vi.fn(),
  updateCostDeviationSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
}))

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data', 'options'],
    template: '<div class="supplier-cost-trend-chart" data-test="supplier-cost-trend-chart" />',
  },
  Bar: {
    name: 'Bar',
    props: ['data', 'options'],
    template: '<div class="supplier-cost-breakdown-chart" data-test="supplier-cost-breakdown-chart" />',
  },
}))

vi.mock('@/api/admin/supplierProviders', () => ({
  default: {
    list: providerViewMocks.listProviders,
    listCostTrends: providerViewMocks.listCostTrends,
    backfillCostTrends: providerViewMocks.backfillCostTrends,
    getBalanceSummary: providerViewMocks.getBalanceSummary,
    update: providerViewMocks.updateProvider,
    refreshToken: providerViewMocks.refreshToken,
    getUpstreamSessions: providerViewMocks.getUpstreamSessions,
    getUpstreamSessionDetails: providerViewMocks.getUpstreamSessionDetails,
    revokeUpstreamSessions: providerViewMocks.revokeUpstreamSessions,
    getAuthStatus: providerViewMocks.getAuthStatus,
    listAuthHistory: providerViewMocks.listAuthHistory,
    getCostDeviationSettings: providerViewMocks.getCostDeviationSettings,
    updateCostDeviationSettings: providerViewMocks.updateCostDeviationSettings,
  },
}))


vi.mock('@/api/admin/supplierProviderTypes', () => ({
  default: {
    list: providerViewMocks.listProviderTypes,
  },
}))

vi.mock('@/api/admin/supplierProviderData', () => ({
  syncProvider: providerViewMocks.syncProvider,
  streamSupplierProviderSync: providerViewMocks.streamSupplierProviderSync,
  testProviderEndpoint: providerViewMocks.testProviderEndpoint,
  default: {
    syncProvider: providerViewMocks.syncProvider,
    streamSupplierProviderSync: providerViewMocks.streamSupplierProviderSync,
    testProviderEndpoint: providerViewMocks.testProviderEndpoint,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: providerViewMocks.showError,
    showSuccess: providerViewMocks.showSuccess,
    showWarning: providerViewMocks.showWarning,
  }),
}))

const supplierProvidersSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierProvidersView.vue'),
  'utf-8'
)

function createProviderRows() {
  return [
    createProviderRow(1, 'Alpha', 30, 100, 1),
    { ...createProviderRow(2, 'Beta', 10, 20, 2), enabled: false },
    { ...createProviderRow(3, 'Gamma', 20, 50, 3), is_default: true },
  ]
}

function createProviderRow(id: number, name: string, todayCost: number, balance: number, sortOrder: number) {
  return {
    id,
    code: name.toLowerCase(),
    name,
    provider_type: 'sub2api',
    base_url: `https://${name.toLowerCase()}.example.com`,
    enabled: true,
    is_default: false,
    credential_configured: true,
    status: 'ready',
    risk_level: 'low',
    valid_account_count: id,
    schedulable_account_count: id,
    success_rate: 100 - id,
    current_balance: balance,
    today_cost: todayCost,
    rate_risk_count: 0,
    sync_status: 'success',
    sort_order: sortOrder,
    last_sync_at: `2026-07-${26 - id}T08:00:00Z`,
  }
}

async function mountSupplierProviders() {
  const wrapper = mount(SupplierProvidersView, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': {} } })],
      stubs: {
        SupplierModuleLayout: { template: '<div><slot /></div>' },
        SupplierRechargeHistoryDialog: {
          name: 'SupplierRechargeHistoryDialog',
          props: ['show', 'providerId', 'providerName', 'providers'],
          emits: ['close'],
          template: `<section v-if="show" data-test="supplier-recharge-dialog-stub">{{ providerId || 'all' }} {{ providerName }}</section>`,
        },
        SupplierDrawer: {
          props: ['show', 'title', 'eyebrow'],
          emits: ['close'],
          template: '<aside v-if="show" data-test="supplier-drawer"><button type="button" data-test="supplier-drawer-close" @click="$emit(\'close\')">关闭</button><slot /></aside>',
        },
        BaseDialog: {
          name: 'BaseDialog',
          props: ['show', 'title'],
          template: '<section v-if="show" data-test="base-dialog-stub"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>',
        },
        Input: {
          name: 'Input',
          inheritAttrs: false,
          props: ['modelValue', 'type'],
          emits: ['update:modelValue'],
          template: '<div v-bind="$attrs"><input :type="type" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></div>',
        },
        Select: {
          name: 'Select',
          inheritAttrs: false,
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<button type="button" v-bind="$attrs" @click="$emit(\'update:modelValue\', \'login_failed\')">{{ modelValue }}</button>',
        },
        DateRangePicker: {
          name: 'DateRangePicker',
          props: ['startDate', 'endDate'],
          emits: ['update:startDate', 'update:endDate', 'change'],
          template: '<button type="button" data-test="supplier-cost-date-range-trigger" @click="$emit(\'change\', { startDate: \'2026-07-01\', endDate: \'2026-07-10\', preset: null })">date-range</button>',
        },
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<button type="button" v-bind="$attrs" :data-enabled="String(modelValue)" @click="$emit(\'update:modelValue\', !modelValue)"></button>',
        },
        Icon: true,
      },
    },
  })
  await flushPromises()
  return wrapper
}

type ProvidersWrapper = Awaited<ReturnType<typeof mountSupplierProviders>>

async function openCostDialog(wrapper: ProvidersWrapper) {
  await wrapper.get('[data-test="supplier-cost-dialog-open"]').trigger('click')
  await flushPromises()
}

// mockSingleNewAPIProvider 让列表只返回一个 New API 供应商，供上游会话相关用例复用。
function mockSingleNewAPIProvider() {
  providerViewMocks.listProviders.mockResolvedValueOnce({
    items: [{ ...createProviderRow(1, 'Alpha', 30, 100, 1), provider_type: 'newapi' }],
    summary: {
      total_count: 1,
      enabled_count: 1,
      high_risk_count: 0,
      low_balance_count: 0,
      sync_failure_count: 0,
      rate_risk_count: 0,
    },
    total: 1,
    page: 1,
    page_size: 100,
  })
}

// createUpstreamSession 构造一条上游会话明细，字段值可在调用处按需覆盖。
function createUpstreamSession(sid: string, current = false) {
  return {
    sid,
    current,
    status: 'active',
    login_method: 'password',
    ip: '10.0.0.1',
    user_agent: 'curl/8',
    created_at: '2026-08-05T07:00:00Z',
    last_active_at: '2026-08-05T07:30:00Z',
    expires_at: '2026-09-05T07:00:00Z',
  }
}

async function openSessionDetailDialog(wrapper: ProvidersWrapper) {
  await wrapper.get('[data-test="supplier-upstream-session-detail-1"]').trigger('click')
  await flushPromises()
}

describe('SupplierProvidersView payload normalization', () => {
  let providerRows: ReturnType<typeof createProviderRows>

  beforeEach(() => {
    vi.clearAllMocks()
    providerRows = createProviderRows()
    providerViewMocks.listProviderTypes.mockResolvedValue([])
    providerViewMocks.updateProvider.mockResolvedValue({})
    providerViewMocks.getAuthStatus.mockResolvedValue({
      provider_id: 1,
      summary: {
        login_count: 4,
        login_success_count: 3,
        login_failure_count: 1,
        refresh_count: 4,
        refresh_success_count: 3,
        refresh_failure_count: 1,
        cache_hit_count: 7,
        cache_miss_count: 2,
        last_login_at: '2026-08-05T06:30:00Z',
        last_login_status: 'success',
        last_login_error: '',
        last_cache_hit_at: '2026-08-05T06:35:00Z',
        last_cache_error: '',
        last_token_expires_at: '2026-08-05T08:30:00Z',
        last_token_fingerprint: 'fingerprint-only',
      },
      cache: {
        status: 'cached',
        cached: true,
        token_type: 'Bearer',
        token_summary: 'abcd…wxyz',
        token_length: 64,
        token_fingerprint: 'fingerprint-only',
        token_expires_at: '2026-08-05T08:30:00Z',
        remaining_seconds: 3600,
        ttl_seconds: 3500,
        cookie_present: true,
      },
      login_lock: { held: false, status: 'available', remaining_seconds: 0 },
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.listAuthHistory.mockResolvedValue({
      items: [
        {
          id: 9,
          provider_id: 1,
          event_type: 'refresh_success',
          source: 'sync',
          status: 'success',
          started_at: '2026-08-05T06:30:00Z',
          finished_at: '2026-08-05T06:30:01Z',
          duration_ms: 1000,
          http_status: 200,
          token_fingerprint: 'fingerprint-only',
          token_length: 64,
          cookie_present: true,
          created_at: '2026-08-05T06:30:01Z',
        },
      ],
      total: 25,
      page: 1,
      page_size: 20,
    })
    providerViewMocks.listCostTrends.mockResolvedValue({
      days: 14,
      points: [
        { date: '2026-07-16', upstream_cost: 12, calculated_cost: 11, local_cost: 10, effective_cost: 10 },
        { date: '2026-07-17', upstream_cost: 15, calculated_cost: 15, local_cost: 14, effective_cost: 15 },
      ],
      breakdown: [
        { provider_id: 1, provider_name: 'Alpha', provider_type: 'sub2api', upstream_cost: 120, calculated_cost: 110, local_cost: 80, effective_cost: 80 },
        { provider_id: 2, provider_name: 'Beta', provider_type: 'sub2api', upstream_cost: 90, calculated_cost: 88, local_cost: 45, effective_cost: 90 },
      ],
    })
    providerViewMocks.backfillCostTrends.mockResolvedValue({
      start_date: '2026-07-16',
      end_date: '2026-07-17',
      provider_count: 1,
      day_count: 2,
      success_count: 2,
      failed_count: 0,
      skipped_count: 0,
      items: [],
      started_at: '2026-07-31T00:00:00Z',
    })
    providerViewMocks.getCostDeviationSettings.mockResolvedValue({ threshold: 0.5 })
    providerViewMocks.updateCostDeviationSettings.mockResolvedValue({ threshold: 0.5 })
    providerViewMocks.getBalanceSummary.mockResolvedValue({
      latest_date: '2026-08-14',
      today: { date: '2026-08-14', balance: 170, cost: 60 },
      previous: { date: '2026-08-13', balance: 140, cost: 55 },
      history: { first_date: '2026-08-01', days: 14, total_balance: 2400, total_cost: 620 },
    })
    providerViewMocks.listProviders.mockResolvedValue({
      items: providerRows,
      summary: {
        total_count: providerRows.length,
        enabled_count: providerRows.length,
        high_risk_count: 0,
        low_balance_count: 0,
        sync_failure_count: 0,
        rate_risk_count: 0,
      },
      total: providerRows.length,
      page: 1,
      page_size: 100,
    })
    providerViewMocks.streamSupplierProviderSync.mockResolvedValue(undefined)
  })

  it('loads masked token status and paged login history without triggering a new login', async () => {
    const wrapper = await mountSupplierProviders()
    const loginHistoryButton = wrapper.findAll('button').find(button => button.text() === '登录记录')
    expect(loginHistoryButton).toBeDefined()

    await loginHistoryButton!.trigger('click')
    await flushPromises()

    expect(providerViewMocks.getAuthStatus).toHaveBeenCalledWith(1)
    expect(providerViewMocks.listAuthHistory).toHaveBeenCalledWith(1, {
      page: 1,
      page_size: 20,
      event_type: '',
    })
    expect(wrapper.text()).toContain('abcd…wxyz')
    expect(wrapper.text()).toContain('刷新成功')
    expect(wrapper.text()).toContain('Token 刷新')
    expect(wrapper.text()).toContain('3 / 1')
    expect(wrapper.text()).toContain('共 4 次')
    expect(wrapper.text()).toContain('缓存命中 / 未命中')
    expect(wrapper.text()).toContain('7 / 2')
    expect(wrapper.text()).toContain('最近缓存命中')
    expect(wrapper.text()).not.toContain('access-token')

    await wrapper.get('[data-test="supplier-auth-event-filter"]').trigger('click')
    await flushPromises()
    expect(providerViewMocks.listAuthHistory).toHaveBeenLastCalledWith(1, {
      page: 1,
      page_size: 20,
      event_type: 'login_failed',
    })

    await wrapper.get('[data-test="supplier-auth-next"]').trigger('click')
    await flushPromises()
    expect(providerViewMocks.listAuthHistory).toHaveBeenLastCalledWith(1, {
      page: 2,
      page_size: 20,
      event_type: 'login_failed',
    })

    providerViewMocks.getAuthStatus.mockClear()
    providerViewMocks.listAuthHistory.mockClear()
    await wrapper.get('[data-test="supplier-auth-refresh"]').trigger('click')
    await flushPromises()
    expect(providerViewMocks.getAuthStatus).toHaveBeenCalledTimes(1)
    expect(providerViewMocks.listAuthHistory).toHaveBeenCalledTimes(1)
    expect(providerViewMocks.syncProvider).not.toHaveBeenCalled()
    expect(providerViewMocks.testProviderEndpoint).not.toHaveBeenCalled()
  })

  it('shows the manual refresh button only for NewAPI providers and refreshes the token', async () => {
    providerRows.splice(0, providerRows.length, ...[
      { ...createProviderRow(1, 'NewAPI', 30, 100, 1), provider_type: 'newapi' },
      createProviderRow(2, 'Sub2API', 10, 20, 2),
    ])
    providerViewMocks.refreshToken.mockResolvedValue({
      provider_id: 1,
      expires_at: '2026-08-08T08:30:00Z',
      message: 'Token 刷新成功',
    })

    const wrapper = await mountSupplierProviders()
    const refreshButton = wrapper.get('[data-test="supplier-provider-refresh-token-1"]')

    expect(wrapper.find('[data-test="supplier-provider-refresh-token-2"]').exists()).toBe(false)
    await refreshButton.trigger('click')
    await flushPromises()

    expect(providerViewMocks.refreshToken).toHaveBeenCalledWith(1)
    expect(providerViewMocks.showSuccess).toHaveBeenCalledWith('Token 刷新成功')
  })

  it('disables the manual refresh button while the request is running', async () => {
    providerRows.splice(0, providerRows.length, { ...createProviderRow(1, 'NewAPI', 30, 100, 1), provider_type: 'newapi' })
    let resolveRefresh!: (value: unknown) => void
    providerViewMocks.refreshToken.mockReturnValue(new Promise(resolve => {
      resolveRefresh = resolve
    }))

    const wrapper = await mountSupplierProviders()
    const refreshButton = wrapper.get('[data-test="supplier-provider-refresh-token-1"]')
    await refreshButton.trigger('click')

    expect(refreshButton.text()).toBe('刷新中')
    expect((refreshButton.element as HTMLButtonElement).disabled).toBe(true)

    resolveRefresh({ message: 'Token 刷新成功' })
  })

  it('re-authenticates Cookie-session NewAPI providers through the existing refresh endpoint', async () => {
    providerRows.splice(0, providerRows.length, {
      ...createProviderRow(1, 'MidNux', 30, 100, 1),
      provider_type: 'newapi',
      newapi_auth_mode: 'cookie_session',
    })
    let resolveRefresh!: (value: unknown) => void
    providerViewMocks.refreshToken.mockReturnValue(new Promise(resolve => {
      resolveRefresh = resolve
    }))

    const wrapper = await mountSupplierProviders()
    const refreshButton = wrapper.get('[data-test="supplier-provider-refresh-token-1"]')
    expect(refreshButton.text()).toBe('重新登录')

    await refreshButton.trigger('click')
    expect(refreshButton.text()).toBe('登录中')
    expect(providerViewMocks.refreshToken).toHaveBeenCalledWith(1)

    resolveRefresh({ message: '登录会话已更新' })
    await flushPromises()
    expect(providerViewMocks.showSuccess).toHaveBeenCalledWith('登录会话已更新')
  })

  it('declares refresh event filters and API types for login history', () => {
    expect(supplierProvidersSource).toContain("{ value: 'refresh_success', label: '刷新成功' }")
    expect(supplierProvidersSource).toContain("{ value: 'refresh_failed', label: '刷新失败' }")
    expect(supplierProvidersSource).toContain("refresh_success: '刷新成功'")
    expect(supplierProvidersSource).toContain("refresh_failed: '刷新失败'")
  })

  it('sorts provider rows when a sortable table header is clicked', async () => {
    const wrapper = await mountSupplierProviders()
    const rowIds = () => wrapper.findAll('tbody tr[data-row-id]').map(row => row.attributes('data-row-id'))

    expect(rowIds()).toEqual(['1', '2', '3'])

    const todayCostHeader = wrapper.findAll('thead th').at(5)
    await todayCostHeader.trigger('click')
    await flushPromises()
    expect(rowIds()).toEqual(['2', '3', '1'])

    await todayCostHeader.trigger('click')
    await flushPromises()
    expect(rowIds()).toEqual(['1', '3', '2'])
  })
  it('filters provider rows by the selected quick filter', async () => {
    const wrapper = await mountSupplierProviders()
    const rowIds = () => wrapper.findAll('tbody tr[data-row-id]').map(row => row.attributes('data-row-id'))

    expect(rowIds()).toEqual(['1', '2', '3'])

    await wrapper.get('[data-test="supplier-provider-filter-enabled"]').trigger('click')
    expect(rowIds()).toEqual(['1', '3'])

    await wrapper.get('[data-test="supplier-provider-filter-disabled"]').trigger('click')
    expect(rowIds()).toEqual(['2'])

    await wrapper.get('[data-test="supplier-provider-filter-default"]').trigger('click')
    expect(rowIds()).toEqual(['3'])

    await wrapper.get('[data-test="supplier-provider-filter-all"]').trigger('click')
    expect(rowIds()).toEqual(['1', '2', '3'])
  })

  it('prevents syncing a disabled provider until it is enabled again', async () => {
    const wrapper = await mountSupplierProviders()

    const syncButton = wrapper.get('[data-test="supplier-provider-sync-all-2"]')

    expect(syncButton.attributes('disabled')).toBeDefined()
    expect(syncButton.attributes('title')).toBe('供应商已停用，请先启用后再同步')
    expect(providerViewMocks.streamSupplierProviderSync).not.toHaveBeenCalled()
  })

  it('updates a provider enabled state from the table switch', async () => {
    const wrapper = await mountSupplierProviders()

    await wrapper.get('[data-test="supplier-provider-enabled-1"]').trigger('click')
    await flushPromises()

    expect(providerViewMocks.updateProvider).toHaveBeenCalledWith(1, expect.objectContaining({
      code: 'alpha',
      enabled: false,
      is_default: false,
    }))
  })

  it('restores the table switch when updating a provider enabled state fails', async () => {
    providerViewMocks.updateProvider.mockRejectedValueOnce(new Error('更新失败'))
    const wrapper = await mountSupplierProviders()
    const toggle = wrapper.get('[data-test="supplier-provider-enabled-1"]')

    await toggle.trigger('click')
    await flushPromises()

    expect(providerViewMocks.showError).toHaveBeenCalledWith('更新失败')
    expect(wrapper.get('[data-test="supplier-provider-enabled-1"]').attributes('data-enabled')).toBe('true')
  })

  it('opens the provider drawer and renders each live sync stage', async () => {
    providerViewMocks.streamSupplierProviderSync.mockImplementationOnce(async (_id, _scope, options) => {
      options.onEvent({ stage: 'prepare', message: '准备同步', time: '2026-08-05T07:00:00Z' })
      options.onEvent({ stage: 'captcha', message: 'YesCaptcha 打码成功', ok: true, time: '2026-08-05T07:00:01Z' })
      options.onEvent({ stage: 'done', message: '同步完成', ok: true, time: '2026-08-05T07:00:02Z' })
    })

    const wrapper = await mountSupplierProviders()
    const syncButton = wrapper.findAll('button').find(button => button.text() === '同步全部')
    expect(syncButton).toBeDefined()

    await syncButton!.trigger('click')
    await flushPromises()

    expect(providerViewMocks.streamSupplierProviderSync).toHaveBeenCalledWith(
      1,
      'all',
      expect.objectContaining({ onEvent: expect.any(Function) }),
    )
    const progress = wrapper.get('[data-test="supplier-sync-progress"]')
    expect(progress.text()).toContain('实时同步诊断')
    expect(progress.text()).toContain('准备同步')
    expect(progress.text()).toContain('打码')
    expect(progress.text()).toContain('YesCaptcha 打码成功')
    expect(progress.text()).toContain('已完成')
  })

  it('keeps a failed sync message after closing and reopening the provider drawer', async () => {
    providerViewMocks.streamSupplierProviderSync.mockImplementationOnce(async (_id, _scope, options) => {
      options.onEvent({ stage: 'prepare', message: '准备同步', time: '2026-08-05T07:00:00Z' })
      options.onEvent({ stage: 'error', message: '上游登录失败：打码平台超时', ok: false, time: '2026-08-05T07:00:01Z' })
    })

    const wrapper = await mountSupplierProviders()
    const syncButton = wrapper.findAll('button').find(button => button.text() === '同步全部')
    expect(syncButton).toBeDefined()

    await syncButton!.trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-test="supplier-sync-progress"]').text()).toContain('上游登录失败：打码平台超时')
    expect(wrapper.get('[data-test="supplier-sync-progress-all"]').text()).toContain('失败')

    await wrapper.get('[data-test="supplier-drawer-close"]').trigger('click')
    expect(wrapper.find('[data-test="supplier-sync-progress"]').exists()).toBe(false)

    await wrapper.get('tbody tr[data-row-id="1"]').trigger('click')
    expect(wrapper.get('[data-test="supplier-sync-progress"]').text()).toContain('上游登录失败：打码平台超时')
  })

  it('供应商名称按类型展示不同颜色', () => {
    expect(supplierProvidersSource).toContain('sp-provider-name')
    expect(supplierProvidersSource).toContain('providerNameTypeClass(provider.provider_type)')
    expect(supplierProvidersSource).toContain('providerNameTypeStyle(provider.provider_type)')
    expect(supplierProvidersSource).toContain("sub2api: 'type-sub2api'")
    expect(supplierProvidersSource).toContain("newapi: 'type-newapi'")
    expect(supplierProvidersSource).toContain("return normalized ? 'type-random' : 'type-default'")
    expect(supplierProvidersSource).toContain('function providerNameTypeStyle')
  })


  it('uses one unified provider filter card without a repeated page heading', () => {
    expect(supplierProvidersSource).not.toContain('class="sp-page-head"')
    expect(supplierProvidersSource).not.toContain('Provider Operations')
    expect(supplierProvidersSource).not.toContain('class="sp-subtitle"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-card"')
    expect(supplierProvidersSource).toContain('class="sp-filter-card-head"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-body"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-fields"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-actions"')
    expect(supplierProvidersSource).toContain('筛选供应商')
    expect(supplierProvidersSource).toContain('@media (max-width: 900px)')
    expect(supplierProvidersSource).toContain('@media (max-width: 520px)')
  })

  it('provides a direct create-provider-type action and dedicated dialog', () => {
    expect(supplierProvidersSource).toContain('@click="openCreateProviderType"')
    expect(supplierProvidersSource).toContain('新增供应商类型')
    expect(supplierProvidersSource).toContain(':show="createTypeVisible"')
    expect(supplierProvidersSource).toContain('class="sp-type-create-dialog"')
    expect(supplierProvidersSource).toContain('@submit.prevent="submitNewProviderType"')
    expect(supplierProvidersSource).toContain('const createTypeVisible = ref(false)')
    expect(supplierProvidersSource).toContain('function openCreateProviderType()')
    expect(supplierProvidersSource).toContain('function closeCreateProviderType()')
    expect(supplierProvidersSource).toContain('async function submitNewProviderType()')
  })

  it('uses structured page-level styling for all supplier dialogs', () => {
    expect(supplierProvidersSource).toContain('class="sp-provider-dialog"')
    expect(supplierProvidersSource).toContain('class="sp-dialog-summary"')
    expect(supplierProvidersSource).toContain('class="sp-type-manager-dialog"')
    expect(supplierProvidersSource).toContain('class="sp-test-dialog"')
    expect(supplierProvidersSource).toContain('class="sp-dialog-section-head"')
    expect(supplierProvidersSource).toContain('.sp-dialog-section {')
    expect(supplierProvidersSource).toContain('.sp-type-manager-dialog {')
    expect(supplierProvidersSource).toContain('.sp-test-dialog {')
    expect(supplierProvidersSource).toContain('@media (max-width: 760px)')
    expect(supplierProvidersSource).toContain(':global(.dark .modal-content:has(.sp-provider-dialog))')
  })
  it('submits Sub2API credentials as email only and clears stale username', () => {
    expect(supplierProvidersSource).toContain('const normalizedProviderType = payload.provider_type.trim()')
    expect(supplierProvidersSource).toContain("email: normalizedProviderType === 'sub2api' ? payload.email?.trim() || '' : ''")
    expect(supplierProvidersSource).toContain("username: normalizedProviderType === 'sub2api' ? '' : payload.username?.trim() || ''")
  })

  it('provides per-scope test buttons and a frontend diagnostics dialog', () => {
    expect(supplierProvidersSource).toContain('testProviderEndpoint')
    expect(supplierProvidersSource).toContain('测试 API Key')
    expect(supplierProvidersSource).toContain('测试分组')
    expect(supplierProvidersSource).toContain('测试余额')
    expect(supplierProvidersSource).toContain('测试成本')
    expect(supplierProvidersSource).toContain('接口测试结果')
    expect(supplierProvidersSource).toContain('testResultVisible')
  })

  it('uses the global app toast store for provider operation feedback', () => {
    expect(supplierProvidersSource).toContain("import { useAppStore } from '@/stores/app'")
    expect(supplierProvidersSource).toContain('const appStore = useAppStore()')
    expect(supplierProvidersSource).toContain('appStore.showError(')
    expect(supplierProvidersSource).toContain('appStore.showSuccess(')
    expect(supplierProvidersSource).not.toContain('class="sp-toast"')
  })

  it('uses existing framework components instead of native table, modal, and form controls', () => {
    expect(supplierProvidersSource).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(supplierProvidersSource).toContain("import DataTable from '@/components/common/DataTable.vue'")
    expect(supplierProvidersSource).toContain("import Input from '@/components/common/Input.vue'")
    expect(supplierProvidersSource).toContain("import Select, { type SelectOption } from '@/components/common/Select.vue'")
    expect(supplierProvidersSource).toContain("import Toggle from '@/components/common/Toggle.vue'")
    expect(supplierProvidersSource).toContain('<BaseDialog')
    expect(supplierProvidersSource).toContain('<DataTable')
    expect(supplierProvidersSource).toContain('<Input')
    expect(supplierProvidersSource).toContain('<Select')
    expect(supplierProvidersSource).toContain('<Toggle')
    expect(supplierProvidersSource).not.toContain('SupplierModal')
    expect(supplierProvidersSource).not.toContain('<table')
    expect(supplierProvidersSource).not.toContain('<select')
    expect(supplierProvidersSource).not.toContain('<input')
    expect(supplierProvidersSource).not.toContain('type="checkbox"')
  })

  it('places a homepage shortcut first and opens the supplier base URL in a new tab', () => {
    expect(supplierProvidersSource).toContain("import Icon from '@/components/icons/Icon.vue'")
    expect(supplierProvidersSource).toContain("{ key: 'homepage', label: '主页'")
    expect(supplierProvidersSource.indexOf("{ key: 'homepage'")).toBeLessThan(
      supplierProvidersSource.indexOf("{ key: 'name'")
    )
    expect(supplierProvidersSource).toContain('<template #cell-homepage="{ row: provider }">')
    expect(supplierProvidersSource).toContain(':data-test="`supplier-provider-home-${provider.id}`"')
    expect(supplierProvidersSource).toContain('@click.stop="openProviderHomepage(provider)"')
    expect(supplierProvidersSource).toContain("window.open(url, '_blank', 'noopener,noreferrer')")
  })

  it('sorts all primary data columns from their table headers', () => {
    for (const key of [
      'name',
      'status',
      'account_counts',
      'success_rate',
      'today_cost',
      'current_balance',
      'rate_risk_count',
      'credential_configured',
      'auth_summary',
      'last_sync_at',
    ]) {
      expect(supplierProvidersSource).toContain(`{ key: '${key}',`)
      expect(supplierProvidersSource).toMatch(
        new RegExp(`\\{ key: '${key}',[^}]*sortable: true`)
      )
    }
    expect(supplierProvidersSource).toContain('server-side-sort')
    expect(supplierProvidersSource).toContain('@sort="handleProviderSort"')
    expect(supplierProvidersSource).toContain("const providerSortKey = ref('')")
    expect(supplierProvidersSource).toContain("const providerSortOrder = ref<'asc' | 'desc'>('asc')")
    expect(supplierProvidersSource).toContain("case 'auth_summary':")
    expect(supplierProvidersSource).toContain('numericValue(left.auth_summary?.login_count) - numericValue(right.auth_summary?.login_count)')
    expect(supplierProvidersSource).not.toContain("const sorts = ['风险优先', '成本效率', '最近同步']")
  })

  it('renders the upstream session panel in place of the provider health panel', async () => {
    const wrapper = await mountSupplierProviders()

    const defaultEnd = new Date()
    const defaultStart = new Date()
    defaultStart.setDate(defaultEnd.getDate() - 13)
    const pad = (value: number) => String(value).padStart(2, '0')
    const formatDate = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
    expect(providerViewMocks.listCostTrends).toHaveBeenCalledWith({
      start_date: formatDate(defaultStart),
      end_date: formatDate(defaultEnd),
    })
    expect(wrapper.get('[data-test="supplier-upstream-session-panel"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-upstream-session-tone"]').text()).toContain('正常')
    expect(supplierProvidersSource).not.toContain('供应商组合健康')
    await openCostDialog(wrapper)
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('成本对比')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('上游成本')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('计算成本')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('本地成本')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('生效成本')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('生效合计')
    expect(supplierProvidersSource).toContain('loadUpstreamSessions')
    expect(supplierProvidersSource).toContain('costTrendChartData')
    expect(supplierProvidersSource).not.toContain('class="sp-stat-list"')
  })

  it('只对 New API 供应商查询上游会话，并按会话数量给出告警', async () => {
    providerViewMocks.listProviders.mockResolvedValueOnce({
      items: [
        { ...createProviderRow(1, 'Alpha', 30, 100, 1), provider_type: 'newapi' },
        { ...createProviderRow(2, 'Beta', 10, 20, 2), provider_type: 'newapi' },
      ],
      summary: {
        total_count: 2,
        enabled_count: 2,
        high_risk_count: 0,
        low_balance_count: 0,
        sync_failure_count: 0,
        rate_risk_count: 0,
      },
      total: 2,
      page: 1,
      page_size: 100,
    })
    providerViewMocks.getUpstreamSessions.mockImplementation(async (id: number) => ({
      provider_id: id,
      supported: true,
      count: id === 1 ? 4 : 1,
      message: '',
      checked_at: '2026-08-05T07:30:00Z',
    }))

    const wrapper = await mountSupplierProviders()
    await flushPromises()

    // 只查询 newapi 类型，sub2api 不在统计范围内。
    expect(providerViewMocks.getUpstreamSessions).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-test="supplier-upstream-session-tone"]').text()).toContain('需清理')
    expect(wrapper.get('[data-test="supplier-upstream-session-item-1"]').text()).toContain('4 个')
    expect(wrapper.get('[data-test="supplier-upstream-session-item-1"]').text()).toContain('存在 3 个残留会话')
    expect(wrapper.get('[data-test="supplier-upstream-session-item-2"]').text()).toContain('无残留会话')
  })

  it('清理残留会话后重新查询上游会话数量', async () => {
    providerViewMocks.listProviders.mockResolvedValueOnce({
      items: [{ ...createProviderRow(1, 'Alpha', 30, 100, 1), provider_type: 'newapi' }],
      summary: {
        total_count: 1,
        enabled_count: 1,
        high_risk_count: 0,
        low_balance_count: 0,
        sync_failure_count: 0,
        rate_risk_count: 0,
      },
      total: 1,
      page: 1,
      page_size: 100,
    })
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      count: 5,
      message: '',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.revokeUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      revoked_count: 4,
      message: '',
      revoked_at: '2026-08-05T07:30:00Z',
    })

    const wrapper = await mountSupplierProviders()
    await flushPromises()

    expect(providerViewMocks.getUpstreamSessions).toHaveBeenCalledTimes(1)
    await wrapper.get('[data-test="supplier-upstream-session-revoke-1"]').trigger('click')
    await flushPromises()

    expect(providerViewMocks.revokeUpstreamSessions).toHaveBeenCalledWith(1)
    expect(providerViewMocks.showSuccess).toHaveBeenCalledWith('已清理 4 个残留会话')
    expect(providerViewMocks.getUpstreamSessions).toHaveBeenCalledTimes(2)
  })

  it('上游不支持会话管理时降级提示而不是报错', async () => {
    providerViewMocks.listProviders.mockResolvedValueOnce({
      items: [{ ...createProviderRow(1, 'Alpha', 30, 100, 1), provider_type: 'newapi' }],
      summary: {
        total_count: 1,
        enabled_count: 1,
        high_risk_count: 0,
        low_balance_count: 0,
        sync_failure_count: 0,
        rate_risk_count: 0,
      },
      total: 1,
      page: 1,
      page_size: 100,
    })
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: false,
      count: 0,
      message: '上游未提供会话管理接口，无法统计登录会话',
      checked_at: '2026-08-05T07:30:00Z',
    })

    const wrapper = await mountSupplierProviders()
    await flushPromises()

    const item = wrapper.get('[data-test="supplier-upstream-session-item-1"]')
    expect(item.text()).toContain('不支持')
    expect(item.text()).toContain('上游未提供会话管理接口')
    // 不支持时既不提供清理入口，也不提供明细入口。
    expect(wrapper.find('[data-test="supplier-upstream-session-revoke-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="supplier-upstream-session-detail-1"]').exists()).toBe(false)
  })

  it('打开会话明细弹窗并逐条列出上游登录会话', async () => {
    mockSingleNewAPIProvider()
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      count: 3,
      message: '',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessionDetails.mockResolvedValue({
      provider_id: 1,
      supported: true,
      sessions: [
        createUpstreamSession('sess-current', true),
        createUpstreamSession('sess-residual-1'),
        createUpstreamSession('sess-residual-2'),
      ],
      checked_at: '2026-08-05T07:30:00Z',
    })

    const wrapper = await mountSupplierProviders()
    await flushPromises()

    // 未打开弹窗前不拉明细，避免页面加载时为每个供应商拉全量会话对象。
    expect(providerViewMocks.getUpstreamSessionDetails).not.toHaveBeenCalled()

    await openSessionDetailDialog(wrapper)

    // 打开弹窗属于只读路径，绝不能带上 allowLogin，否则会凭空在上游新增会话。
    expect(providerViewMocks.getUpstreamSessionDetails).toHaveBeenCalledWith(1, { allowLogin: false })
    const dialog = wrapper.get('[data-test="supplier-upstream-session-detail-dialog"]')
    expect(dialog.text()).toContain('Alpha')
    expect(dialog.text()).toContain('共 3 个会话，其中残留 2 个')

    // 当前会话与残留会话必须分别标注，便于判断哪条可以清理。
    expect(wrapper.get('[data-test="supplier-upstream-session-detail-item-sess-current"]').text()).toContain('当前会话')
    expect(wrapper.get('[data-test="supplier-upstream-session-detail-item-sess-residual-1"]').text()).toContain('残留会话')
    expect(wrapper.get('[data-test="supplier-upstream-session-detail-item-sess-residual-1"]').text()).toContain('10.0.0.1')

    expect(wrapper.get('[data-test="supplier-upstream-session-detail-revoke"]').text()).toContain('清理 2 个残留会话')
  })

  it('明细弹窗内清理残留会话后同时刷新明细与数量', async () => {
    mockSingleNewAPIProvider()
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      count: 3,
      message: '',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessionDetails.mockResolvedValue({
      provider_id: 1,
      supported: true,
      sessions: [
        createUpstreamSession('sess-current', true),
        createUpstreamSession('sess-residual-1'),
        createUpstreamSession('sess-residual-2'),
      ],
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.revokeUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      revoked_count: 2,
      message: '',
      revoked_at: '2026-08-05T07:30:00Z',
    })

    const wrapper = await mountSupplierProviders()
    await flushPromises()
    await openSessionDetailDialog(wrapper)

    const summaryCallsBefore = providerViewMocks.getUpstreamSessions.mock.calls.length

    await wrapper.get('[data-test="supplier-upstream-session-detail-revoke"]').trigger('click')
    await flushPromises()

    expect(providerViewMocks.revokeUpstreamSessions).toHaveBeenCalledWith(1)
    expect(providerViewMocks.showSuccess).toHaveBeenCalledWith('已清理 2 个残留会话')
    // 明细重新拉取一次（打开时一次 + 清理后一次）。
    expect(providerViewMocks.getUpstreamSessionDetails).toHaveBeenCalledTimes(2)
    // 面板的会话数量也要同步刷新，避免弹窗与列表数字不一致。
    expect(providerViewMocks.getUpstreamSessions.mock.calls.length).toBe(summaryCallsBefore + 1)
  })

  it('读取会话明细失败时在弹窗内提示而不是静默', async () => {
    mockSingleNewAPIProvider()
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      count: 2,
      message: '',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessionDetails.mockRejectedValue(new Error('boom'))

    const wrapper = await mountSupplierProviders()
    await flushPromises()
    await openSessionDetailDialog(wrapper)

    expect(wrapper.get('[data-test="supplier-upstream-session-detail-error"]').text()).toContain('boom')
    // 读取失败时无法判断残留数量，因此不提供清理入口。
    expect(wrapper.find('[data-test="supplier-upstream-session-detail-revoke"]').exists()).toBe(false)
  })

  it('本地无凭据时显示无法查询而不是谎报 0 条会话', async () => {
    mockSingleNewAPIProvider()
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      credential_available: false,
      count: 0,
      message: '本地未缓存该供应商的登录凭据，本次没有真正查询上游；请先在该供应商上完成一次登录或同步，再重新检查。',
      checked_at: '2026-08-05T07:30:00Z',
    })

    const wrapper = await mountSupplierProviders()
    await flushPromises()

    const item = wrapper.get('[data-test="supplier-upstream-session-item-1"]')
    // 关键回归点：0 条必须显示成「无法查询」，否则用户会以为上游很干净。
    expect(item.text()).toContain('无法查询')
    expect(item.text()).not.toContain('0 个')
    expect(item.text()).toContain('本地未缓存该供应商的登录凭据')
    // 面板整体也要提示「部分无法检查」，不能落到「正常」。
    expect(wrapper.get('[data-test="supplier-upstream-session-summary"]').text()).toContain('部分无法检查')
    // 没有凭据时不能给出清理入口（清不了），但明细入口要保留，用户才有机会登录后读取。
    expect(wrapper.find('[data-test="supplier-upstream-session-revoke-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="supplier-upstream-session-detail-1"]').exists()).toBe(true)
  })

  it('明细弹窗在无凭据时给出登录并读取入口，且只在点按后才带上 allowLogin', async () => {
    mockSingleNewAPIProvider()
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      credential_available: false,
      count: 0,
      message: '本地未缓存该供应商的登录凭据，本次没有真正查询上游；请先在该供应商上完成一次登录或同步，再重新检查。',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessionDetails.mockResolvedValueOnce({
      provider_id: 1,
      supported: true,
      credential_available: false,
      sessions: [],
      message: '本地未缓存该供应商的登录凭据，本次没有真正查询上游；请先在该供应商上完成一次登录或同步，再重新检查。',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessionDetails.mockResolvedValueOnce({
      provider_id: 1,
      supported: true,
      credential_available: true,
      sessions: [createUpstreamSession('sess-current', true), createUpstreamSession('sess-residual-1')],
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      credential_available: true,
      count: 2,
      message: '',
      checked_at: '2026-08-05T07:30:00Z',
    })

    const wrapper = await mountSupplierProviders()
    await flushPromises()
    await openSessionDetailDialog(wrapper)

    // 首次打开仍是只读，不能凭空在上游新增会话。
    expect(providerViewMocks.getUpstreamSessionDetails).toHaveBeenNthCalledWith(1, 1, { allowLogin: false })
    expect(wrapper.get('[data-test="supplier-upstream-session-detail-no-credential"]').text()).toContain('本地未缓存该供应商的登录凭据')
    // 无凭据时不应提供清理入口。
    expect(wrapper.find('[data-test="supplier-upstream-session-detail-revoke"]').exists()).toBe(false)

    await wrapper.get('[data-test="supplier-upstream-session-detail-login"]').trigger('click')
    await flushPromises()

    // 只有用户显式点按才允许登录后查询。
    expect(providerViewMocks.getUpstreamSessionDetails).toHaveBeenNthCalledWith(2, 1, { allowLogin: true })
    expect(wrapper.get('[data-test="supplier-upstream-session-detail-item-sess-residual-1"]').text()).toContain('残留会话')
    expect(wrapper.get('[data-test="supplier-upstream-session-detail-revoke"]').text()).toContain('清理 1 个残留会话')
    // 登录成功后面板的「无法查询」也要同步刷新掉。
    expect(wrapper.get('[data-test="supplier-upstream-session-item-1"]').text()).toContain('2 个')
  })

  it('无凭据且明细读取失败时不把面板刷新成正常', async () => {
    mockSingleNewAPIProvider()
    providerViewMocks.getUpstreamSessions.mockResolvedValue({
      provider_id: 1,
      supported: true,
      credential_available: false,
      count: 0,
      message: '本地未缓存该供应商的登录凭据，本次没有真正查询上游；请先在该供应商上完成一次登录或同步，再重新检查。',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessionDetails.mockResolvedValueOnce({
      provider_id: 1,
      supported: true,
      credential_available: false,
      sessions: [],
      message: '本地未缓存该供应商的登录凭据，本次没有真正查询上游；请先在该供应商上完成一次登录或同步，再重新检查。',
      checked_at: '2026-08-05T07:30:00Z',
    })
    providerViewMocks.getUpstreamSessionDetails.mockRejectedValueOnce(new Error('boom'))

    const wrapper = await mountSupplierProviders()
    await flushPromises()
    await openSessionDetailDialog(wrapper)

    await wrapper.get('[data-test="supplier-upstream-session-detail-login"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="supplier-upstream-session-detail-error"]').text()).toContain('boom')
    // 登录读取失败时面板必须保持「无法查询」，不能被刷新成正常。
    expect(wrapper.get('[data-test="supplier-upstream-session-item-1"]').text()).toContain('无法查询')
  })

  it('renders grouped upstream, calculated, local and effective cost bars for each supplier', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)

    const chart = wrapper.get('[data-test="supplier-cost-breakdown-chart"]')
    expect(chart.exists()).toBe(true)

    const bar = wrapper.findComponent({ name: 'Bar' })
    expect(bar.exists()).toBe(true)
    // 上游成本可能只有余额兜底而为 0，所以柱状图按生效成本从高到低排：Beta 生效 90 排在 Alpha 生效 80 之前。
    expect(bar.props('data')).toMatchObject({
      labels: ['Beta', 'Alpha'],
      datasets: [
        { label: '上游成本', data: [90, 120] },
        { label: '计算成本', data: [88, 110] },
        { label: '本地成本', data: [45, 80] },
        { label: '生效成本', data: [90, 80] },
      ],
    })
    expect(supplierProvidersSource).toContain('costBreakdownChartData')
    expect(supplierProvidersSource).toContain('costBreakdownChartOptions')
    expect(supplierProvidersSource).toContain('Bar')
  })

  it('opens both cost charts in a single dialog from the filter action button', async () => {
    const wrapper = await mountSupplierProviders()

    expect(wrapper.get('[data-test="supplier-cost-dialog-open"]').text()).toContain('成本分析')
    expect(wrapper.find('[data-test="supplier-cost-trend"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="supplier-cost-breakdown-panel"]').exists()).toBe(false)

    await openCostDialog(wrapper)
    expect(wrapper.get('[data-test="supplier-cost-trend"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-breakdown-panel"]').exists()).toBe(true)
  })

  it('shows today and historical balance/cost summary cards', async () => {
    const wrapper = await mountSupplierProviders()

    expect(wrapper.get('[data-test="supplier-balance-today"]').text()).toContain('今日总余额')
    expect(wrapper.get('[data-test="supplier-balance-today"]').text()).toContain('170')
    expect(wrapper.get('[data-test="supplier-cost-today"]').text()).toContain('今日总成本')
    expect(wrapper.get('[data-test="supplier-balance-previous"]').text()).toContain('历史总余额')
    expect(wrapper.get('[data-test="supplier-balance-previous"]').text()).toContain('140')
    expect(wrapper.get('[data-test="supplier-cost-history"]').text()).toContain('历史总成本')
    // 历史总成本按上一统计日对比口径展示，脚注保留累计成本。
    expect(wrapper.get('[data-test="supplier-cost-history"]').text()).toContain('55')
    expect(wrapper.get('[data-test="supplier-cost-history"]').text()).toContain('620')
    expect(supplierProvidersSource).toContain('getBalanceSummary')
    expect(supplierProvidersSource).toContain('loadBalanceSummary')
  })

  it('opens one supplier recharge history dialog from the provider row', async () => {
    const wrapper = await mountSupplierProviders()
    await wrapper.get('[data-test="supplier-provider-recharges-1"]').trigger('click')
    await flushPromises()

    const dialog = wrapper.get('[data-test="supplier-recharge-dialog-stub"]')
    expect(dialog.text()).toContain('1 Alpha')
    expect(supplierProvidersSource).toContain('openProviderRechargeHistory')
  })

  it('opens all supplier recharge history dialog from the toolbar', async () => {
    const wrapper = await mountSupplierProviders()
    await wrapper.get('.sp-filter-action-recharge').trigger('click')

    expect(wrapper.get('[data-test="supplier-recharge-dialog-stub"]').text()).toContain('all')
    expect(supplierProvidersSource).toContain('openAllRechargeHistory')
  })

  it('places the supplier cost breakdown in a full-width panel without horizontal scrolling', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)

    const breakdownPanel = wrapper.get('[data-test="supplier-cost-breakdown-panel"]')
    expect(breakdownPanel.classes()).toContain('sp-panel')
    expect(breakdownPanel.classes()).toContain('sp-cost-breakdown-panel')
    expect(wrapper.get('[data-test="supplier-upstream-session-panel"]').find('[data-test="supplier-cost-breakdown"]').exists()).toBe(false)
    expect(supplierProvidersSource).not.toContain('sp-health-breakdown-chart-scroll')
    expect(supplierProvidersSource).not.toContain('costBreakdownChartMinWidth')
    expect(supplierProvidersSource).toMatch(
      /\.sp-health-breakdown-chart\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;/,
    )
  })


  it('switches cost trend date range and provider filter', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)
    providerViewMocks.listCostTrends.mockClear()

    await wrapper.get('[data-test="supplier-cost-date-range-trigger"]').trigger('click')
    await flushPromises()
    expect(providerViewMocks.listCostTrends).toHaveBeenCalledWith({
      start_date: '2026-07-01',
      end_date: '2026-07-10',
    })

    // 通过源码契约确认时间范围与供应商筛选控件存在
    expect(supplierProvidersSource).toContain('DateRangePicker')
    expect(supplierProvidersSource).toContain('costTrendStartDate')
    expect(supplierProvidersSource).toContain('costTrendEndDate')
    expect(supplierProvidersSource).toContain('onCostTrendDateRangeChange')
    expect(supplierProvidersSource).toContain('costTrendProviderOptions')
    expect(supplierProvidersSource).toContain('onCostTrendProviderChange')
    expect(wrapper.get('[data-test="supplier-cost-controls"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-provider"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-date-range"]').exists()).toBe(true)
  })

  it('separates the cost breakdown date range from the cost trend chart', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)
    providerViewMocks.listCostTrends.mockClear()

    const breakdownDateRange = wrapper.get('[data-test="supplier-cost-breakdown-date-range"]')
    await breakdownDateRange.get('[data-test="supplier-cost-date-range-trigger"]').trigger('click')
    await flushPromises()

    // 两个图时间范围独立：改拆分图日期只刷新拆分图，只发一次请求。
    expect(providerViewMocks.listCostTrends).toHaveBeenCalledTimes(1)
    expect(providerViewMocks.listCostTrends).toHaveBeenLastCalledWith({
      start_date: '2026-07-01',
      end_date: '2026-07-10',
    })
    expect(supplierProvidersSource).toContain('costTrendStartDate')
    expect(supplierProvidersSource).toContain('costTrendEndDate')
    expect(supplierProvidersSource).toContain('costBreakdownStartDate')
    expect(supplierProvidersSource).toContain('costBreakdownEndDate')
    expect(supplierProvidersSource).toContain('onCostBreakdownDateRangeChange')
    expect(supplierProvidersSource).toContain('costBreakdownLoading')
    expect(wrapper.get('[data-test="supplier-cost-controls"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-breakdown-controls"]').exists()).toBe(true)
  })

  it('highlights cost trend points whose deviation exceeds the threshold', async () => {
    // 默认阈值 0.5：首日计算成本偏低使偏差超过 50%。
    // 本地成本贴着上游也不该消掉红点——偏差基准是计算成本，它才是顶替生效成本的那个值。
    providerViewMocks.listCostTrends.mockResolvedValue({
      days: 14,
      points: [
        { date: '2026-07-16', upstream_cost: 12, calculated_cost: 5, local_cost: 11, effective_cost: 5 },
        { date: '2026-07-17', upstream_cost: 15, calculated_cost: 14, local_cost: 4, effective_cost: 15 },
      ],
      breakdown: [],
    })
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)

    const line = wrapper.findComponent({ name: 'Line' })
    expect(line.exists()).toBe(true)
    const upstream = line.props('data').datasets[0]
    expect(upstream.pointBackgroundColor).toEqual(['#dc2626', '#3b82f6'])
    const calculated = line.props('data').datasets[1]
    expect(calculated.pointBackgroundColor).toEqual(['#dc2626', '#7c3aed'])
    const tooltipCallbacks = line.props('options').plugins.tooltip.callbacks
    expect(tooltipCallbacks.labelColor({ datasetIndex: 0 })).toEqual({
      borderColor: '#3b82f6',
      backgroundColor: '#3b82f6',
    })
    expect(tooltipCallbacks.labelColor({ datasetIndex: 1 })).toEqual({
      borderColor: '#7c3aed',
      backgroundColor: '#7c3aed',
    })
    expect(tooltipCallbacks.labelColor({ datasetIndex: 2 })).toEqual({
      borderColor: '#d97706',
      backgroundColor: '#d97706',
    })
    expect(tooltipCallbacks.labelColor({ datasetIndex: 3 })).toEqual({
      borderColor: '#059669',
      backgroundColor: '#059669',
    })
    expect(wrapper.get('[data-test="supplier-cost-deviation-summary"]').text()).toContain('偏差超阈值 1/2 天')
    expect(supplierProvidersSource).toContain('costTrendDeviation')
    expect(supplierProvidersSource).toContain('pointBackgroundColor')
  })

  it('does not flag deviation on days without a calculated cost', async () => {
    // 成本核对记录上线前的历史日期没有计算成本，缺值不能当成 100% 偏差报红。
    providerViewMocks.listCostTrends.mockResolvedValue({
      days: 14,
      points: [
        { date: '2026-07-16', upstream_cost: 12, calculated_cost: 0, local_cost: 11, effective_cost: 12 },
        { date: '2026-07-17', upstream_cost: 15, calculated_cost: 14, local_cost: 14, effective_cost: 15 },
      ],
      breakdown: [],
    })
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)

    const line = wrapper.findComponent({ name: 'Line' })
    expect(line.props('data').datasets[0].pointBackgroundColor).toEqual(['#3b82f6', '#3b82f6'])
    expect(wrapper.get('[data-test="supplier-cost-deviation-summary"]').text()).toContain('偏差均未超')
  })

  it('shows the saved threshold and only persists an edited value after clicking save', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)
    const thresholdInput = wrapper.get('[data-test="supplier-cost-deviation-threshold"] input')
    const savedThreshold = wrapper.get('[data-test="supplier-cost-deviation-current"]')
    const saveButton = wrapper.get('[data-test="supplier-cost-deviation-save"]')

    expect(savedThreshold.text()).toContain('50%')
    expect(saveButton.text()).toContain('保存')
    expect(saveButton.attributes('disabled')).toBeDefined()

    await thresholdInput.setValue('250')
    await flushPromises()

    expect(providerViewMocks.updateCostDeviationSettings).not.toHaveBeenCalled()
    expect(savedThreshold.text()).toContain('50%')
    expect((thresholdInput.element as HTMLInputElement).value).toBe('250')
    expect(saveButton.attributes('disabled')).toBeUndefined()

    await saveButton.trigger('click')
    await flushPromises()

    expect(providerViewMocks.updateCostDeviationSettings).toHaveBeenCalledWith(2.5)
    expect(savedThreshold.text()).toContain('250%')
    expect(providerViewMocks.showSuccess).toHaveBeenCalled()
  })

  it('allows a zero threshold and rejects negative or empty input', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)
    const thresholdInput = wrapper.get('[data-test="supplier-cost-deviation-threshold"] input')
    const saveButton = wrapper.get('[data-test="supplier-cost-deviation-save"]')
    providerViewMocks.updateCostDeviationSettings.mockClear()
    providerViewMocks.showError.mockClear()

    await thresholdInput.setValue('0')
    await flushPromises()
    expect(providerViewMocks.updateCostDeviationSettings).not.toHaveBeenCalled()
    await saveButton.trigger('click')
    await flushPromises()
    expect(providerViewMocks.updateCostDeviationSettings).toHaveBeenCalledWith(0)

    providerViewMocks.updateCostDeviationSettings.mockClear()
    await thresholdInput.setValue('-1')
    await flushPromises()
    expect(providerViewMocks.updateCostDeviationSettings).not.toHaveBeenCalled()
    expect(providerViewMocks.showError).toHaveBeenCalled()

    providerViewMocks.showError.mockClear()
    await thresholdInput.setValue('')
    await flushPromises()
    expect(providerViewMocks.updateCostDeviationSettings).not.toHaveBeenCalled()
    expect(providerViewMocks.showError).toHaveBeenCalled()
  })
  it('fetches cost for the selected provider on the chosen date', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)
    providerViewMocks.listCostTrends.mockClear()
    providerViewMocks.streamSupplierProviderSync.mockClear()
    providerViewMocks.showSuccess.mockClear()

    const row = wrapper.find('tbody tr[data-row-id="1"]')
    expect(row.exists()).toBe(true)
    await row.trigger('click')
    await flushPromises()

    const pad = (value: number) => String(value).padStart(2, '0')
    const today = new Date()
    const expectedDate = `${today.getFullYear()}-${pad(today.getMonth() + 1)}-${pad(today.getDate())}`

    const button = wrapper.get('[data-test="supplier-cost-single-day"]')
    expect(button.attributes('disabled')).toBeUndefined()
    providerViewMocks.streamSupplierProviderSync.mockImplementationOnce(async (_id, _scope, options) => {
      options.onEvent({ stage: 'done', message: '成本同步完成', ok: true, time: '2026-08-05T07:00:02Z' })
    })
    await button.trigger('click')
    await flushPromises()

    expect(providerViewMocks.streamSupplierProviderSync).toHaveBeenCalledWith(
      1,
      'cost',
      expect.objectContaining({ params: { date: expectedDate }, onEvent: expect.any(Function) }),
    )
    expect(providerViewMocks.showSuccess).toHaveBeenCalledWith(expect.stringContaining('已获取'))
    // 获取成功后刷新成本曲线。
    expect(providerViewMocks.listCostTrends).toHaveBeenCalled()
  })

  it('shows deviation warnings on the trend and breakdown panels', async () => {
    providerViewMocks.listCostTrends.mockResolvedValue({
      days: 14,
      points: [
        { date: '2026-07-16', upstream_cost: 12, calculated_cost: 10, local_cost: 10, effective_cost: 10, warning: '上游成本 12.00 与计算成本 10.00 偏差 17%，生效成本已取计算成本' },
        { date: '2026-07-17', upstream_cost: 15, calculated_cost: 14, local_cost: 14, effective_cost: 15 },
      ],
      breakdown: [
        { provider_id: 1, provider_name: 'Alpha', provider_type: 'sub2api', upstream_cost: 120, calculated_cost: 80, local_cost: 80, effective_cost: 80, cost_warning: '上游成本 120.00 与计算成本 80.00 偏差 33%，生效成本已取计算成本' },
      ],
    })
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)

    expect(wrapper.get('[data-test="supplier-cost-warnings"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-warning-item"]').text()).toContain('生效成本已取计算成本')
    expect(wrapper.get('[data-test="supplier-cost-breakdown-warnings"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-breakdown-warning-item"]').text()).toContain('Alpha')
  })

  it('keeps the supplier cost breakdown date control in the left heading group', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)

    const header = wrapper.get('[data-test="supplier-cost-breakdown-panel"] .sp-panel-head')
    const leftGroup = header.get('[data-test="supplier-cost-breakdown-head-left"]')

    expect(leftGroup.get('.sp-panel-title').exists()).toBe(true)
    expect(leftGroup.get('[data-test="supplier-cost-breakdown-controls"]').exists()).toBe(true)
    expect(header.get('.sp-cost-breakdown-count').exists()).toBe(true)
  })

  it('places the shared cost date range control in the cost chart heading', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)

    const trend = wrapper.get('[data-test="supplier-cost-trend"]')
    const controls = trend.get('[data-test="supplier-cost-controls"]')
    expect(controls.get('[data-test="supplier-cost-date-range"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-breakdown"]').find('[data-test="supplier-cost-controls"]').exists()).toBe(false)
  })

  it('syncs cost trend provider when a table row is selected', async () => {
    const wrapper = await mountSupplierProviders()
    providerViewMocks.listCostTrends.mockClear()

    const row = wrapper.find('tbody tr[data-row-id="1"]')
    expect(row.exists()).toBe(true)
    await row.trigger('click')
    await flushPromises()

    const defaultEnd = new Date()
    const defaultStart = new Date()
    defaultStart.setDate(defaultEnd.getDate() - 13)
    const pad = (value: number) => String(value).padStart(2, '0')
    const formatDate = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
    expect(providerViewMocks.listCostTrends).toHaveBeenCalledWith({
      start_date: formatDate(defaultStart),
      end_date: formatDate(defaultEnd),
      provider_id: 1,
    })
    expect(supplierProvidersSource).toContain('selectProviderForDetail')
  })

  it('backfills upstream costs for the selected range before reloading trends', async () => {
    const wrapper = await mountSupplierProviders()
    await openCostDialog(wrapper)
    providerViewMocks.listCostTrends.mockClear()
    providerViewMocks.backfillCostTrends.mockClear()
    providerViewMocks.showSuccess.mockClear()

    await wrapper.get('[data-test="supplier-cost-refresh"]').trigger('click')
    await flushPromises()

    const defaultEnd = new Date()
    const defaultStart = new Date()
    defaultStart.setDate(defaultEnd.getDate() - 13)
    const pad = (value: number) => String(value).padStart(2, '0')
    const formatDate = (date: Date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
    const expectedRange = {
      start_date: formatDate(defaultStart),
      end_date: formatDate(defaultEnd),
    }

    expect(providerViewMocks.backfillCostTrends).toHaveBeenCalledWith(expectedRange)
    expect(providerViewMocks.listCostTrends).toHaveBeenCalledWith(expectedRange)
    expect(providerViewMocks.showSuccess).toHaveBeenCalled()
    expect(String(providerViewMocks.showSuccess.mock.calls[0][0])).toContain('上游成本回补完成')
    expect(supplierProvidersSource).toContain('backfillCostTrends')
    expect(supplierProvidersSource).toContain('notifyCostBackfillResult')
  })

  it('uses dedicated cost and balance colors with a strict ten-yuan warning threshold', () => {
    expect(supplierProvidersSource).toContain('class="sp-provider-today-cost"')
    expect(supplierProvidersSource).toContain("'sp-provider-balance-warning'")
    expect(supplierProvidersSource).toContain("'sp-provider-balance-normal'")
    expect(supplierProvidersSource).toContain('function isBalanceWarning(provider: SupplierProvider): boolean')
    expect(supplierProvidersSource).toContain('return numericValue(provider.current_balance) < 10')
    expect(supplierProvidersSource).toContain('return currency(provider.current_balance)')
    expect(supplierProvidersSource).toContain('.sp-provider-today-cost {')
    expect(supplierProvidersSource).toContain('.sp-provider-balance-normal {')
    expect(supplierProvidersSource).toContain('.sp-provider-balance-warning {')
  })
})
