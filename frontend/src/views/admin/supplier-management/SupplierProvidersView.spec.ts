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
  listProviderTypes: vi.fn(),
  updateProvider: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data', 'options'],
    template: '<div class="supplier-cost-trend-chart" data-test="supplier-cost-trend-chart" />',
  },
}))

vi.mock('@/api/admin/supplierProviders', () => ({
  default: {
    list: providerViewMocks.listProviders,
    listCostTrends: providerViewMocks.listCostTrends,
    update: providerViewMocks.updateProvider,
  },
}))

vi.mock('@/api/admin/supplierProviderTypes', () => ({
  default: { list: providerViewMocks.listProviderTypes },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: providerViewMocks.showError,
    showSuccess: providerViewMocks.showSuccess,
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
        SupplierDrawer: true,
        BaseDialog: true,
        Input: true,
        Select: true,
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
    providerViewMocks.listCostTrends.mockResolvedValue({
      days: 14,
      points: [
        { date: '2026-07-16', upstream_cost: 12, local_cost: 10 },
        { date: '2026-07-17', upstream_cost: 15, local_cost: 14 },
      ],
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
