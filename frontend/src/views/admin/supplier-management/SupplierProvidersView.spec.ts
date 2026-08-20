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
        Input: true,
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
        { date: '2026-07-16', upstream_cost: 12, local_cost: 10 },
        { date: '2026-07-17', upstream_cost: 15, local_cost: 14 },
      ],
      breakdown: [
        { provider_id: 1, provider_name: 'Alpha', provider_type: 'sub2api', upstream_cost: 120, local_cost: 80 },
        { provider_id: 2, provider_name: 'Beta', provider_type: 'sub2api', upstream_cost: 90, local_cost: 45 },
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

  it('renders redesigned supplier health panel with cost trend chart', async () => {
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
    expect(wrapper.get('[data-test="supplier-health-panel"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-health-tone"]').text()).toContain('稳定')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('成本对比')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('上游成本')
    expect(wrapper.get('[data-test="supplier-cost-trend"]').text()).toContain('本地成本')
    expect(supplierProvidersSource).toContain('priorityTodos')
    expect(supplierProvidersSource).toContain('costTrendChartData')
    expect(supplierProvidersSource).not.toContain('class="sp-stat-list"')
  })

  it('lays out the health detail list below priority todos as a compact full-width grid', () => {
    const listStyle = supplierProvidersSource.match(
      /\.sp-health-completeness-list\s*\{[\s\S]*?\n\}/,
    )?.[0] ?? ''

    expect(listStyle).toContain('display: grid')
    expect(listStyle).toContain('grid-template-columns: repeat(2, minmax(0, 1fr))')
    expect(supplierProvidersSource).toContain('.sp-health-completeness-list .sp-list-item')
  })

  it('renders grouped upstream and local cost bars for each supplier', async () => {
    const wrapper = await mountSupplierProviders()

    const chart = wrapper.get('[data-test="supplier-cost-breakdown-chart"]')
    expect(chart.exists()).toBe(true)

    const bar = wrapper.findComponent({ name: 'Bar' })
    expect(bar.exists()).toBe(true)
    expect(bar.props('data')).toMatchObject({
      labels: ['Alpha', 'Beta'],
      datasets: [
        { label: '上游成本', data: [120, 90] },
        { label: '本地成本', data: [80, 45] },
      ],
    })
    expect(supplierProvidersSource).toContain('costBreakdownChartData')
    expect(supplierProvidersSource).toContain('costBreakdownChartOptions')
    expect(supplierProvidersSource).toContain('Bar')
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

    const breakdownPanel = wrapper.get('[data-test="supplier-cost-breakdown-panel"]')
    expect(breakdownPanel.classes()).toContain('sp-panel')
    expect(breakdownPanel.classes()).toContain('sp-cost-breakdown-panel')
    expect(wrapper.get('[data-test="supplier-health-panel"]').find('[data-test="supplier-cost-breakdown"]').exists()).toBe(false)
    expect(supplierProvidersSource).not.toContain('sp-health-breakdown-chart-scroll')
    expect(supplierProvidersSource).not.toContain('costBreakdownChartMinWidth')
    expect(supplierProvidersSource).toMatch(
      /\.sp-health-breakdown-chart\s*\{[\s\S]*?width:\s*100%;[\s\S]*?min-width:\s*0;/,
    )
  })

  it('shows priority todos and filters high-risk providers when a health todo is clicked', async () => {
    providerViewMocks.listProviders.mockResolvedValueOnce({
      items: [
        {
          ...createProviderRow(1, 'Alpha', 30, 100, 1),
          risk_level: 'high',
          sync_status: 'failed',
        },
        createProviderRow(2, 'Beta', 10, 20, 2),
        { ...createProviderRow(3, 'Gamma', 20, 50, 3), is_default: true },
      ],
      summary: {
        total_count: 3,
        enabled_count: 3,
        high_risk_count: 1,
        low_balance_count: 0,
        sync_failure_count: 1,
        rate_risk_count: 0,
      },
      total: 3,
      page: 1,
      page_size: 100,
    })

    const wrapper = await mountSupplierProviders()
    expect(wrapper.get('[data-test="supplier-health-tone"]').text()).toContain('告警')
    expect(wrapper.get('[data-test="supplier-health-todo-high-risk"]').exists()).toBe(true)

    await wrapper.get('[data-test="supplier-health-todo-high-risk"]').trigger('click')
    await flushPromises()

    const rowIds = wrapper.findAll('tbody tr[data-row-id]').map(row => row.attributes('data-row-id'))
    expect(rowIds).toEqual(['1'])
  })


  it('switches cost trend date range and provider filter', async () => {
    const wrapper = await mountSupplierProviders()
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
    // 默认阈值 0.5：首日本地成本偏低使偏差超过 50%。
    providerViewMocks.listCostTrends.mockResolvedValue({
      days: 14,
      points: [
        { date: '2026-07-16', upstream_cost: 12, local_cost: 5 },
        { date: '2026-07-17', upstream_cost: 15, local_cost: 14 },
      ],
      breakdown: [],
    })
    const wrapper = await mountSupplierProviders()

    const line = wrapper.findComponent({ name: 'Line' })
    expect(line.exists()).toBe(true)
    const upstream = line.props('data').datasets[0]
    expect(upstream.pointBackgroundColor).toEqual(['#dc2626', '#3b82f6'])
    expect(wrapper.get('[data-test="supplier-cost-deviation-summary"]').text()).toContain('偏差超阈值 1/2 天')
    expect(supplierProvidersSource).toContain('costTrendDeviation')
    expect(supplierProvidersSource).toContain('pointBackgroundColor')
  })

  it('persists cost deviation threshold changes to the backend settings', async () => {
    const wrapper = await mountSupplierProviders()
    expect(providerViewMocks.getCostDeviationSettings).toHaveBeenCalled()

    // 首次加载完成前不持久化。
    expect(providerViewMocks.updateCostDeviationSettings).not.toHaveBeenCalled()
    expect(supplierProvidersSource).toContain('deviationThresholdReady')
    expect(supplierProvidersSource).toContain('persistCostDeviationThreshold')
    expect(supplierProvidersSource).toContain('updateCostDeviationSettings')
  })

  it('fetches cost for the selected provider on the chosen date', async () => {
    const wrapper = await mountSupplierProviders()
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

  it('shows deviation override warnings on the trend and breakdown panels', async () => {
    providerViewMocks.listCostTrends.mockResolvedValue({
      days: 14,
      points: [
        { date: '2026-07-16', upstream_cost: 12, local_cost: 10, warning: '上游成本 12.00 与本地成本 10.00 偏差 17%，已按本地成本展示' },
        { date: '2026-07-17', upstream_cost: 15, local_cost: 14 },
      ],
      breakdown: [
        { provider_id: 1, provider_name: 'Alpha', provider_type: 'sub2api', upstream_cost: 120, local_cost: 80, cost_warning: '上游成本 120.00 与本地成本 80.00 偏差 33%，已按本地成本展示' },
      ],
    })
    const wrapper = await mountSupplierProviders()

    expect(wrapper.get('[data-test="supplier-cost-warnings"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-warning-item"]').text()).toContain('已按本地成本展示')
    expect(wrapper.get('[data-test="supplier-cost-breakdown-warnings"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="supplier-cost-breakdown-warning-item"]').text()).toContain('Alpha')
  })

  it('keeps the supplier cost breakdown date control in the left heading group', async () => {
    const wrapper = await mountSupplierProviders()

    const header = wrapper.get('[data-test="supplier-cost-breakdown-panel"] .sp-panel-head')
    const leftGroup = header.get('[data-test="supplier-cost-breakdown-head-left"]')

    expect(leftGroup.get('.sp-panel-title').exists()).toBe(true)
    expect(leftGroup.get('[data-test="supplier-cost-breakdown-controls"]').exists()).toBe(true)
    expect(header.get('.sp-cost-breakdown-count').exists()).toBe(true)
  })

  it('places the shared cost date range control in the cost chart heading', async () => {
    const wrapper = await mountSupplierProviders()

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
