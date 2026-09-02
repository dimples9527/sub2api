import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierAccountsView from './SupplierAccountsView.vue'

const supplierAccountMocks = vi.hoisted(() => ({
  listProviders: vi.fn(),
  listGroups: vi.fn(),
  listAccounts: vi.fn(),
  listCustomPlatforms: vi.fn(),
  listAccountRateGuardUnbindLogs: vi.fn(),
  listAutomationTasks: vi.fn(),
  getAccountById: vi.fn(),
  getAvailableModels: vi.fn(),
  duplicate: vi.fn(),
  setSchedulable: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getById: supplierAccountMocks.getAccountById,
      getAvailableModels: supplierAccountMocks.getAvailableModels,
      duplicate: supplierAccountMocks.duplicate,
      setSchedulable: supplierAccountMocks.setSchedulable,
    },
    groups: {
      getAll: supplierAccountMocks.listGroups,
    },
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: supplierAccountMocks.showError,
    showSuccess: supplierAccountMocks.showSuccess,
  }),
}))

vi.mock('@/api/admin/supplierProviders', () => ({
  default: {
    list: supplierAccountMocks.listProviders,
  },
}))

vi.mock('@/api/admin/customPlatforms', () => ({
  customPlatformsAPI: {
    list: supplierAccountMocks.listCustomPlatforms,
  },
  default: {
    list: supplierAccountMocks.listCustomPlatforms,
  },
}))

vi.mock('@/api/admin/supplierAutomation', () => ({
  listAccountRateGuardUnbindLogs: supplierAccountMocks.listAccountRateGuardUnbindLogs,
  listTasks: supplierAccountMocks.listAutomationTasks,
}))

vi.mock('@/api/admin/supplierProviderData', async importOriginal => {
  const actual = await importOriginal<typeof import('@/api/admin/supplierProviderData')>()
  return {
    ...actual,
    listSupplierAccounts: supplierAccountMocks.listAccounts,
    default: {
      ...actual.default,
      listSupplierAccounts: supplierAccountMocks.listAccounts,
    },
  }
})

const currentDir = dirname(fileURLToPath(import.meta.url))

const groupsSource = readFileSync(resolve(currentDir, 'SupplierGroupsView.vue'), 'utf-8')
const accountsSource = readFileSync(resolve(currentDir, 'SupplierAccountsView.vue'), 'utf-8')
const supplierProviderDataSource = readFileSync(
  resolve(currentDir, '../../../api/admin/supplierProviderData.ts'),
  'utf-8'
)

const runtimeCellKeys = [
  'provider_name',
  'upstream_account_key',
  'upstream_account_status',
  'local_account_name',
  'local_account_priority',
  'rate_multiplier',
  'group_name',
  'local_account_status',
  'local_account_schedulable',
  'local_account_last_test_status',
  'local_account_last_tested_at',
  'supplier_current_balance',
  'supplier_today_cost',
  'actions',
]

const DataTableStub = defineComponent({
  props: {
    data: {
      type: Array,
      default: () => [],
    },
    columns: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['row-click', 'sort'],
  setup(props, { emit, slots }) {
    return () => h('div', { class: 'data-table-stub' }, props.data.flatMap((row, index) => [
      h('button', {
        class: 'account-row-trigger',
        type: 'button',
        onClick: () => emit('row-click', row),
      }, `账号行 ${index + 1}`),
      ...runtimeCellKeys.map(key => h(
        'div',
        { class: `runtime-cell runtime-cell-${key}`, 'data-row-index': index },
        slots[`cell-${key}`]?.({ row })
      )),
    ]))
  },
})

const SupplierDrawerStub = defineComponent({
  props: {
    show: Boolean,
    title: String,
  },
  setup(props, { slots }) {
    return () => props.show
      ? h('aside', { class: 'supplier-drawer-stub' }, [
          h('h2', props.title),
          slots.default?.(),
        ])
      : null
  },
})

const BaseDialogStub = defineComponent({
  props: {
    show: Boolean,
    title: String,
  },
  emits: ['close'],
  setup(props, { slots }) {
    return () => props.show
      ? h('section', { class: 'base-dialog-stub' }, [
          h('h2', props.title),
          slots.default?.(),
          slots.footer?.(),
        ])
      : null
  },
})

const ConfirmDialogStub = defineComponent({
  props: {
    show: Boolean,
    title: String,
    message: String,
    confirmText: String,
    cancelText: String,
  },
  emits: ['confirm', 'cancel'],
  setup(props, { emit }) {
    return () => props.show
      ? h('section', { class: 'confirm-dialog-stub' }, [
          h('h2', props.title),
          h('p', props.message),
          h('button', {
            type: 'button',
            onClick: () => emit('cancel'),
          }, props.cancelText || '取消'),
          h('button', {
            type: 'button',
            onClick: () => emit('confirm'),
          }, props.confirmText || '确认'),
        ])
      : null
  },
})

const AccountTestModalStub = defineComponent({
  props: {
    show: Boolean,
    account: {
      type: Object,
      default: null,
    },
  },
  emits: ['close', 'test-result'],
  setup(props) {
    return () => props.show
      ? h('section', { class: 'account-test-modal-stub' }, String(props.account?.name || ''))
      : null
  },
})

const SupplierModuleLayoutStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('main', slots.default?.())
  },
})

const InputStub = defineComponent({
  inheritAttrs: false,
  setup(_props, { attrs }) {
    return () => h('input', { ...attrs, type: 'text' })
  },
})

const SelectStub = defineComponent({
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: undefined,
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () => h('button', {
      ...attrs,
      class: ['select-trigger', attrs.class],
      type: 'button',
      'aria-label': 'Select option',
      'data-option-labels': props.options.map(option => option.label).join('|'),
      onClick: () => {
        const nextOption = props.options.find(option => (
          option.value !== props.modelValue && option.value !== '' && option.value !== 0
        ))
        if (nextOption) emit('update:modelValue', nextOption.value)
      },
    }, '筛选控件')
  },
})

const PaginationStub = defineComponent({
  setup() {
    return () => h('div')
  },
})

const GroupBadgeStub = defineComponent({
  props: {
    name: { type: String, required: true },
    platform: { type: String, default: '' },
    rateMultiplier: { type: Number, default: undefined },
  },
  setup(props) {
    return () => h('span', {
      class: [
        'group-badge-stub',
        props.platform === 'openai' ? 'text-green-700' : '',
        props.platform === 'anthropic' ? 'text-orange-700' : '',
      ],
    }, `${props.name} ${props.rateMultiplier}x`)
  },
})

const guardFreshCheckedAt = new Date(Date.now() - 2 * 60 * 1000).toISOString()
const guardStaleCheckedAt = new Date(Date.now() - 45 * 60 * 1000).toISOString()

const testAccounts = [
  {
    id: 1,
    provider_id: 11,
    provider_name: '供应商 A',
    upstream_account_key: 'key-a',
    name: '上游账号 A',
    status: 'active',
    group_key: 'group-a',
    group_name: '分组 A',
    platform: 'openai',
    local_account_platform: 'anthropic',
    local_account_type: 'apikey',
    rate_multiplier: 1.23456,
    binding_groups: [
      {
        id: 201,
        name: 'OpenAI 专线',
        platform: 'openai',
        rate_multiplier: 1.5,
        subscription_type: 'standard',
      },
      {
        id: 202,
        name: 'Claude 订阅',
        platform: 'anthropic',
        rate_multiplier: 2,
        subscription_type: 'subscription',
      },
    ],
    raw_status: 'active',
    active: true,
    last_seen_at: '2026-07-22T01:00:00Z',
    local_account_match_status: 'matched',
    local_account_match_count: 1,
    local_account_id: 101,
    local_account_name: '本地账号 A',
    local_account_priority: 0,
    local_account_status: 'active',
    local_account_schedulable: false,
    local_account_last_test_status: 'success',
    local_account_last_tested_at: '2026-07-22T02:00:00Z',
    local_account_last_test_error: '',
    local_account_health_guard_last_checked_at: guardFreshCheckedAt,
    local_account_health_guard_failure_count: 0,
    supplier_current_balance: 0,
    supplier_today_cost: 0,
  },
  {
    id: 2,
    provider_id: 11,
    provider_name: '供应商 A',
    upstream_account_key: 'key-b',
    name: '上游账号 B',
    status: 'active',
    group_key: 'group-a',
    group_name: '分组 A',
    rate_multiplier: 1,
    binding_groups: [],
    raw_status: 'active',
    active: true,
    last_seen_at: '2026-07-22T01:00:00Z',
    local_account_match_status: 'unmatched',
    local_account_match_count: 0,
    local_account_name: '不应展示的本地账号',
    local_account_priority: 99,
    local_account_schedulable: true,
    supplier_current_balance: 12.34,
    supplier_today_cost: 1.23,
  },
  {
    id: 3,
    provider_id: 11,
    provider_name: '供应商 A',
    upstream_account_key: 'key-c',
    name: '上游账号 C',
    status: 'active',
    group_key: 'group-a',
    group_name: '分组 A',
    rate_multiplier: 1,
    binding_groups: [],
    raw_status: 'active',
    active: true,
    last_seen_at: '2026-07-22T01:00:00Z',
    local_account_match_status: 'conflict',
    local_account_match_count: 2,
    supplier_current_balance: 12.34,
    supplier_today_cost: 1.23,
  },
  {
    id: 4,
    provider_id: 11,
    provider_name: '供应商 A',
    upstream_account_key: 'key-d',
    name: '上游账号 D',
    status: 'active',
    group_key: 'group-a',
    group_name: '分组 A',
    rate_multiplier: 1,
    binding_groups: [],
    raw_status: 'active',
    active: true,
    last_seen_at: '2026-07-22T01:00:00Z',
    local_account_match_status: 'unexpected',
    local_account_match_count: 0,
    local_account_name: '异常状态不应展示',
    local_account_priority: 88,
    supplier_current_balance: 12.34,
    supplier_today_cost: 1.23,
  },
  {
    id: 5,
    provider_id: 12,
    provider_name: '供应商 B',
    upstream_account_key: 'key-e',
    name: '上游账号 E',
    status: 'active',
    group_key: 'group-b',
    group_name: '分组 B',
    local_account_platform: 'anthropic',
    rate_multiplier: 1.5,
    binding_groups: [],
    raw_status: 'active',
    active: true,
    last_seen_at: '2026-07-22T01:00:00Z',
    local_account_match_status: 'matched',
    local_account_match_count: 1,
    local_account_id: 105,
    local_account_name: '本地账号 E',
    local_account_priority: 10,
    local_account_status: 'error',
    local_account_schedulable: true,
    local_account_last_test_status: 'failed',
    local_account_last_tested_at: '2026-07-22T03:00:00Z',
    local_account_last_test_error: '上游鉴权失败：invalid key',
    local_account_health_guard_last_checked_at: guardStaleCheckedAt,
    local_account_health_guard_failure_count: 4,
    supplier_current_balance: 20,
    supplier_today_cost: 2,
  },
]

async function mountSupplierAccounts() {
  const wrapper = mount(SupplierAccountsView, {
    global: {
      stubs: {
        SupplierModuleLayout: SupplierModuleLayoutStub,
        SupplierDrawer: SupplierDrawerStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        AccountTestModal: AccountTestModalStub,
        DataTable: DataTableStub,
        Input: InputStub,
        Select: SelectStub,
        Pagination: PaginationStub,
        GroupBadge: GroupBadgeStub,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('supplier local data views component usage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    supplierAccountMocks.listProviders.mockResolvedValue({ items: [] })
    supplierAccountMocks.listCustomPlatforms.mockResolvedValue([])
    supplierAccountMocks.listAccountRateGuardUnbindLogs.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 1,
      pending_count: 0,
    })
    supplierAccountMocks.listAutomationTasks.mockResolvedValue([
      { task_code: 'supplier_data_sync', cron_expression: '@every 900s' },
      { task_code: 'supplier_account_health_guard', cron_expression: '@every 600s' },
    ])
    supplierAccountMocks.listGroups.mockResolvedValue([
      { id: 201, name: '本地分组 A', platform: 'openai' },
      { id: 202, name: '本地分组 B', platform: 'anthropic' },
    ])
    supplierAccountMocks.listAccounts.mockResolvedValue({
      items: testAccounts,
      total: testAccounts.length,
      page: 1,
      page_size: 20,
    })
    supplierAccountMocks.getAccountById.mockResolvedValue({
      id: 101,
      name: '本地账号 A',
      platform: 'openai',
      status: 'active',
    })
    supplierAccountMocks.getAvailableModels.mockResolvedValue([])
    supplierAccountMocks.duplicate.mockResolvedValue({ id: 1001, name: '本地账号 A (Copy)' })
    supplierAccountMocks.setSchedulable.mockImplementation(async (_id, schedulable) => ({
      schedulable,
    }))
  })

  it('renders matching states, false, and zero values from API data', async () => {
    const wrapper = await mountSupplierAccounts()

    expect(wrapper.get('.runtime-cell-local_account_name[data-row-index="0"]').text())
      .toContain('本地账号 A')
    expect(wrapper.get('.runtime-cell-local_account_priority[data-row-index="0"]').text()).toBe('0')
    expect(wrapper.get('.runtime-cell-local_account_schedulable[data-row-index="0"] button').attributes('title'))
      .toBe('当前不参与调度，点击启用')
    expect(wrapper.get('.runtime-cell-supplier_current_balance[data-row-index="0"]').text())
      .toContain('0.00')
    expect(wrapper.get('.runtime-cell-supplier_today_cost[data-row-index="0"]').text())
      .toContain('0.00')
    expect(wrapper.get('.runtime-cell-rate_multiplier[data-row-index="0"]').text())
      .toBe('× 1.23456')

    expect(wrapper.get('.runtime-cell-local_account_name[data-row-index="1"]').text()).toBe('未匹配')
    expect(wrapper.get('.runtime-cell-local_account_priority[data-row-index="1"]').text()).toBe('—')
    expect(wrapper.get('.runtime-cell-local_account_schedulable[data-row-index="1"]').text()).toBe('—')

    expect(wrapper.get('.runtime-cell-local_account_name[data-row-index="2"]').text())
      .toBe('匹配冲突（2）')
    expect(wrapper.get('.runtime-cell-local_account_name[data-row-index="3"]').text()).toBe('—')
    expect(wrapper.get('.runtime-cell-local_account_priority[data-row-index="3"]').text()).toBe('—')

    wrapper.unmount()
  })

  it('uses stable supplier colors, hides the supplier platform subtitle, and colors rates by platform', async () => {
    const wrapper = await mountSupplierAccounts()
    const providerCells = wrapper.findAll('.sp-provider-cell')

    expect(providerCells[0].classes()).not.toEqual(providerCells[4].classes())
    expect(providerCells[0].text()).not.toContain('OpenAI')
    expect(providerCells[4].text()).not.toContain('Anthropic')
    expect(wrapper.get('.runtime-cell-rate_multiplier[data-row-index="0"] .sp-account-rate').classes())
      .toContain('text-orange-600')
    expect(wrapper.get('.runtime-cell-rate_multiplier[data-row-index="4"] .sp-account-rate').classes())
      .toContain('text-orange-600')

    wrapper.unmount()
  })

  it('shows every bound group with its rate and platform color without collapsing', async () => {
    const wrapper = await mountSupplierAccounts()
    const groupCell = wrapper.get('.runtime-cell-group_name[data-row-index="0"]')
    const badges = groupCell.findAll('.sp-account-groups > span')

    expect(groupCell.text()).toContain('OpenAI 专线')
    expect(groupCell.text()).toContain('1.5x')
    expect(groupCell.text()).toContain('Claude 订阅')
    expect(groupCell.text()).toContain('2x')
    expect(badges).toHaveLength(2)
    expect(badges[0].classes()).toContain('text-green-700')
    expect(badges[1].classes()).toContain('text-orange-700')
    expect(accountsSource).toContain('class="sp-account-groups"')
    expect(accountsSource).not.toContain('max-h-')
    expect(accountsSource).not.toContain('overflow-hidden')

    wrapper.unmount()
  })

  it('adds counted quick filters and filters accounts by their real binding relationship', async () => {
    const wrapper = await mountSupplierAccounts()
    const quickFilters = wrapper.findAll('[data-test^="supplier-account-quick-filter-"]')

    expect(quickFilters.map(button => button.text())).toEqual([
      '全部5',
      '已绑定分组1',
      '未绑定分组4',
      '可参与调度1',
      '暂停调度1',
      '测试失败1',
      '上游分组已删除0',
      '上游密钥已删除0',
    ])

    await wrapper.get('[data-test="supplier-account-quick-filter-bound"]').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.runtime-cell-group_name')).toHaveLength(1)
    expect(wrapper.get('.runtime-cell-group_name').text()).toContain('OpenAI 专线')

    await wrapper.get('[data-test="supplier-account-quick-filter-unbound"]').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.runtime-cell-group_name')).toHaveLength(4)
    expect(wrapper.findAll('.runtime-cell-group_name').every(cell => cell.text() === '—')).toBe(true)
    expect(accountsSource).toContain('function accountMatchesQuickFilter(')
    expect(accountsSource).toContain('page.value = 1')

    wrapper.unmount()
  })

  it('uses server-side sorting for unambiguous account columns', async () => {
    const wrapper = await mountSupplierAccounts()

    expect(accountsSource).toContain('server-side-sort')
    expect(accountsSource).toContain('@sort="handleAccountSort"')
    for (const key of [
      'provider_name',
      'upstream_account_key',
      'local_account_name',
      'local_account_priority',
      'rate_multiplier',
      'local_account_status',
      'local_account_schedulable',
      'local_account_last_test_status',
      'local_account_last_tested_at',
      'supplier_current_balance',
      'supplier_today_cost',
    ]) {
      expect(accountsSource).toMatch(
        new RegExp(`\\{ key: '${key}', label: '[^']+', sortable: true`)
      )
    }
    expect(accountsSource).toContain("{ key: 'provider_name', label: '供应商', sortable: true")
    expect(accountsSource).toContain("{ key: 'supplier_today_cost', label: '今日消费', sortable: true")
    expect(accountsSource).not.toContain("{ key: 'group_name', label: '账号绑定的分组', sortable: true")
    expect(accountsSource).not.toContain("{ key: 'actions', label: '操作', sortable: true")

    const table = wrapper.findComponent(DataTableStub)
    table.vm.$emit('sort', 'supplier_today_cost', 'desc')
    await flushPromises()

    expect(supplierAccountMocks.listAccounts).toHaveBeenLastCalledWith(expect.objectContaining({
      sort_by: 'supplier_today_cost',
      sort_order: 'desc',
      page: 1,
    }))

    wrapper.unmount()
  })

  it('uses platform-aware colors and selection summary in the binding dialog', () => {
    expect(accountsSource).toContain('platformButtonClass')
    expect(accountsSource).toContain('platformAccentBarClass')
    expect(accountsSource).toContain("['sp-account-binding-primary', platformButtonClass(bindingPlatform || '')]")
    expect(accountsSource).toContain('sp-account-binding-primary')
    expect(accountsSource).toContain('const selectedBindingGroups = computed')
    expect(accountsSource).toContain('v-for="group in selectedBindingGroups"')
    expect(accountsSource).toContain(':rate-multiplier="group.rate_multiplier"')
    expect(accountsSource).toContain('sp-account-binding-selected-groups')
    expect(accountsSource).toContain('selectedBindingGroupIDs.length')
    expect(accountsSource).toContain('sp-account-binding-empty')
    expect(accountsSource).not.toContain('选择该账号允许参与调度的分组')
  })

  it('shows local account status and test results in Chinese with distinct states', async () => {
    const wrapper = await mountSupplierAccounts()

    expect(wrapper.get('.runtime-cell-local_account_status[data-row-index="0"]').text()).toBe('正常')
    expect(wrapper.get('.runtime-cell-local_account_status[data-row-index="4"]').text()).toBe('异常')
    expect(wrapper.get('.runtime-cell-local_account_last_test_status[data-row-index="0"]').text())
      .toBe('成功')
    expect(wrapper.get('.runtime-cell-local_account_last_test_status[data-row-index="0"] .sp-test-status').classes())
      .toContain('success')
    expect(wrapper.get('.runtime-cell-local_account_last_test_status[data-row-index="4"]').text())
      .toContain('失败')
    expect(wrapper.get('.runtime-cell-local_account_last_test_status[data-row-index="4"] .sp-test-status').classes())
      .toContain('failed')

    wrapper.unmount()
  })

  it('emphasizes the guard check time and flags lagging checks and repeated guard failures', async () => {
    const wrapper = await mountSupplierAccounts()
    const freshGuard = wrapper.get('.runtime-cell-local_account_last_tested_at[data-row-index="0"] .sp-guard-test-time')
    const staleGuard = wrapper.get('.runtime-cell-local_account_last_tested_at[data-row-index="4"] .sp-guard-test-time')

    expect(freshGuard.find('strong').exists()).toBe(true)
    expect(freshGuard.classes()).not.toContain('is-stale')
    expect(freshGuard.find('strong').attributes('title')).toBeUndefined()

    expect(staleGuard.classes()).toContain('is-stale')
    expect(staleGuard.get('strong').attributes('title')).toContain('守护检测已滞后 45 分钟，超过 20 分钟阈值')
    expect(staleGuard.text()).toContain('守护检测已滞后')

    expect(wrapper.get('.runtime-cell-local_account_last_test_status[data-row-index="4"] .sp-guard-failure-hint').text())
      .toBe('守护连续失败 4 次')
    expect(wrapper.find('.runtime-cell-local_account_last_test_status[data-row-index="0"] .sp-guard-failure-hint').exists())
      .toBe(false)

    wrapper.unmount()
  })

  it('derives the guard lag threshold from twice the guard cron interval', async () => {
    supplierAccountMocks.listAutomationTasks.mockResolvedValue([
      { task_code: 'supplier_account_health_guard', cron_expression: '@every 3600s' },
    ])
    const hourlyWrapper = await mountSupplierAccounts()
    expect(hourlyWrapper.get('.runtime-cell-local_account_last_tested_at[data-row-index="4"] .sp-guard-test-time').classes())
      .not.toContain('is-stale')
    hourlyWrapper.unmount()

    supplierAccountMocks.listAutomationTasks.mockResolvedValue([])
    const fallbackWrapper = await mountSupplierAccounts()
    expect(fallbackWrapper.get('.runtime-cell-local_account_last_tested_at[data-row-index="4"] .sp-guard-test-time').classes())
      .toContain('is-stale')
    fallbackWrapper.unmount()
  })

  it('toggles the matched local account with the existing account API without opening the drawer', async () => {
    const wrapper = await mountSupplierAccounts()
    const toggle = wrapper.get('.runtime-cell-local_account_schedulable[data-row-index="0"] button')

    await toggle.trigger('click')
    await flushPromises()

    expect(supplierAccountMocks.setSchedulable).toHaveBeenCalledWith(101, true)
    expect(wrapper.find('.supplier-drawer-stub').exists()).toBe(false)
    expect(toggle.attributes('title')).toBe('当前参与调度，点击停用')
    expect(wrapper.find('.runtime-cell-local_account_schedulable[data-row-index="1"] button').exists())
      .toBe(false)
    expect(wrapper.find('.runtime-cell-local_account_schedulable[data-row-index="2"] button').exists())
      .toBe(false)

    wrapper.unmount()
  })

  it('keeps the original schedulable state and reports the API error when toggling fails', async () => {
    supplierAccountMocks.setSchedulable.mockRejectedValueOnce(new Error('切换失败'))
    const wrapper = await mountSupplierAccounts()
    const toggle = wrapper.get('.runtime-cell-local_account_schedulable[data-row-index="0"] button')

    await toggle.trigger('click')
    await flushPromises()

    expect(toggle.attributes('title')).toBe('当前不参与调度，点击启用')
    expect(supplierAccountMocks.showError).toHaveBeenCalledWith('切换失败')

    wrapper.unmount()
  })

  it('opens an existing dialog with the failed test error details', async () => {
    const wrapper = await mountSupplierAccounts()

    expect(wrapper.find('.base-dialog-stub').exists()).toBe(false)
    await wrapper.get('.runtime-cell-local_account_last_test_status[data-row-index="4"] button').trigger('click')

    expect(wrapper.get('.base-dialog-stub').text()).toContain('测试失败详情')
    expect(wrapper.get('.base-dialog-stub').text()).toContain('本地账号 E')
    expect(wrapper.get('.base-dialog-stub').text()).toContain('上游鉴权失败：invalid key')

    wrapper.unmount()
  })

  it('opens the account drawer from row clicks and the view action', async () => {
    const rowClickWrapper = await mountSupplierAccounts()
    expect(rowClickWrapper.find('.supplier-drawer-stub').exists()).toBe(false)
    await rowClickWrapper.get('.account-row-trigger').trigger('click')
    expect(rowClickWrapper.get('.supplier-drawer-stub').text()).toContain('上游账号 A')
    expect(rowClickWrapper.get('.supplier-drawer-stub').text()).toContain('本地账号 A')
    rowClickWrapper.unmount()

    const actionWrapper = await mountSupplierAccounts()
    expect(actionWrapper.find('.supplier-drawer-stub').exists()).toBe(false)
    await actionWrapper.get('.sp-account-view-button').trigger('click')
    expect(actionWrapper.get('.supplier-drawer-stub').text()).toContain('上游账号 A')
    actionWrapper.unmount()
  })

  it('gives every account filter a screen-reader-only accessible name', async () => {
    const wrapper = await mountSupplierAccounts()
    const filterGroups = wrapper.findAll('.sp-account-filter-control[role="group"]')

    expect(filterGroups).toHaveLength(6)
    expect(filterGroups.map(group => group.attributes('aria-labelledby'))).toEqual([
      'supplier-account-search-label',
      'supplier-account-provider-label',
      'supplier-account-platform-label',
      'supplier-account-group-label',
      'supplier-account-active-label',
      'supplier-account-upstream-status-label',
    ])
    expect(filterGroups.map(group => group.get('.sr-only').text())).toEqual([
      '账号搜索',
      '供应商',
      '平台',
      '本地分组',
      '同步有效性',
      '上游状态',
    ])
    expect(filterGroups.map(group => {
      const control = group.find('input, button')
      return control.attributes('aria-labelledby')
    })).toEqual([
      'supplier-account-search-label',
      'supplier-account-provider-label',
      'supplier-account-platform-label',
      'supplier-account-group-label',
      'supplier-account-active-label',
      'supplier-account-upstream-status-label',
    ])

    wrapper.unmount()
  })

  it('loads local groups and sends the selected local group to the paged account query', async () => {
    const wrapper = await mountSupplierAccounts()

    expect(supplierAccountMocks.listGroups).toHaveBeenCalledTimes(1)
    expect(supplierAccountMocks.listAccounts).toHaveBeenLastCalledWith(expect.objectContaining({
      group_id: undefined,
    }))

    await wrapper.findAll('.select-trigger')[2].trigger('click')
    await flushPromises()

    expect(supplierAccountMocks.listAccounts).toHaveBeenLastCalledWith(expect.objectContaining({
      group_id: 201,
      page: 1,
    }))

    wrapper.unmount()
  })

  it('includes the local group in the batch-test query contract', () => {
    expect(accountsSource).toContain('group_id: snapshot.groupID || undefined')
  })

  it('filters local-group options by platform and clears an incompatible selection', async () => {
    const wrapper = await mountSupplierAccounts()
    const selectTriggers = wrapper.findAll('.select-trigger')

    expect(selectTriggers[2].attributes('data-option-labels')).toContain('本地分组 A #201')
    expect(selectTriggers[2].attributes('data-option-labels')).toContain('本地分组 B #202')

    await selectTriggers[2].trigger('click')
    await flushPromises()
    expect(supplierAccountMocks.listAccounts).toHaveBeenLastCalledWith(expect.objectContaining({
      group_id: 201,
    }))

    await selectTriggers[1].trigger('click')
    await flushPromises()

    expect(wrapper.findAll('.select-trigger')[2].attributes('data-option-labels'))
      .not.toContain('本地分组 A #201')
    expect(wrapper.findAll('.select-trigger')[2].attributes('data-option-labels'))
      .toContain('本地分组 B #202')
    expect(supplierAccountMocks.listAccounts).toHaveBeenLastCalledWith(expect.objectContaining({
      platform: 'anthropic',
      group_id: undefined,
      page: 1,
    }))

    wrapper.unmount()
  })

  it('defines the local-account matching and supplier-summary account API contract', () => {
    const accountInterface = supplierProviderDataSource.match(
      /export interface SupplierProviderAccount \{([\s\S]*?)\n\}/
    )?.[1]

    expect(accountInterface).toBeDefined()
    ;[
      "local_account_match_status: 'unmatched' | 'matched' | 'conflict'",
      'local_account_match_count: number',
      'local_account_id?: number',
      'local_account_name?: string',
      'local_account_priority?: number',
      'local_account_status?: string',
      'local_account_schedulable?: boolean',
      'local_account_last_test_status?: string',
      'local_account_last_tested_at?: string',
      'local_account_last_test_error?: string',
      'local_account_health_guard_last_checked_at?: string',
      'local_account_health_guard_failure_count?: number',
      'supplier_current_balance: number',
      'supplier_today_cost: number',
    ].forEach(field => expect(accountInterface).toContain(field))
  })
  it.each([
    ['SupplierGroupsView', groupsSource],
    ['SupplierAccountsView', accountsSource],
  ])('%s uses existing framework controls for filters, tables, and pagination', (_name, source) => {
    expect(source).toContain("import DataTable from '@/components/common/DataTable.vue'")
    expect(source).toContain("import Input from '@/components/common/Input.vue'")
    expect(source).toContain("import Pagination from '@/components/common/Pagination.vue'")
    expect(source).toContain("import Select, { type SelectOption } from '@/components/common/Select.vue'")
    expect(source).toContain('<DataTable')
    expect(source).toContain('<Input')
    expect(source).toContain('<Pagination')
    expect(source).toContain('<Select')
    expect(source).not.toContain('<table')
    expect(source).not.toContain('<select')
    expect(source).not.toContain('<input')
  })

  it('adds enabled custom platforms to the group platform filter and editor dropdown', () => {
    expect(groupsSource).toContain("import { customPlatformsAPI, type CustomPlatform } from '@/api/admin/customPlatforms'")
    expect(groupsSource).toContain("import { buildPlatformOptions, CORE_PLATFORM_OPTIONS } from '@/utils/platformOptions'")
    expect(groupsSource).toContain('const customPlatforms = ref<CustomPlatform[]>([])')
    expect(groupsSource).toContain('const platformOptions = computed<SelectOption[]>(() => buildPlatformOptions(customPlatforms.value))')
    expect(groupsSource).toContain('const platformFilterOptions = computed<SelectOption[]>(() => [')
    expect(groupsSource).toContain("await Promise.all([ensureCustomPlatformLabels(), loadCustomPlatforms(), loadProviders(), loadLocalGroups()])")
    expect(groupsSource).toContain('customPlatforms.value = await customPlatformsAPI.list(true)')
  })

  it('supports supplier local group business platform overrides without using core group management', () => {
    expect(supplierProviderDataSource).toContain('platform_override?: string')
    expect(supplierProviderDataSource).toContain('effective_platform?: string')
    expect(supplierProviderDataSource).toContain('setSupplierLocalGroupPlatformOverride')
    expect(supplierProviderDataSource).toContain('/admin/supplier-management/local-groups/${localGroupID}/platform-override')
    expect(groupsSource).toContain('groupPlatformTarget = ref<SupplierProviderGroup | null>(null)')
    expect(groupsSource).toContain('title="配置业务平台"')
    expect(groupsSource).toContain('groupEffectivePlatform(group)')
    expect(groupsSource).toContain('openGroupPlatformDialog(group)')
    expect(groupsSource).toContain('await setSupplierLocalGroupPlatformOverride(localGroupID, platform)')
    expect(groupsSource).toContain('await clearSupplierLocalGroupPlatformOverride(localGroupID)')
    expect(groupsSource).not.toContain('adminAPI.groups.update(localGroupID, { platform')
  })



  it('uses a compact account workbench with the required 14-column order', () => {
    const accountColumnsSource = accountsSource.match(
      /const accountColumns: Column\[\] = \[([\s\S]*?)\n\]/
    )?.[1]
    const accountColumns = [
      ...accountColumnsSource!.matchAll(/\{ key: '([^']+)', label: '([^']+)'/g),
    ].map(([, key, label]) => ({ key, label }))

    expect(accountColumns).toEqual([
      { key: 'provider_name', label: '供应商' },
      { key: 'upstream_account_key', label: '上游账号' },
      { key: 'upstream_account_status', label: '上游状态' },
      { key: 'local_account_name', label: '本地账号' },
      { key: 'local_account_priority', label: '优先级' },
      { key: 'rate_multiplier', label: '上游倍率' },
      { key: 'group_name', label: '账号绑定的分组' },
      { key: 'local_account_status', label: '本地账号状态' },
      { key: 'local_account_schedulable', label: '是否调度' },
      { key: 'local_account_last_test_status', label: '测试结果' },
      { key: 'local_account_last_tested_at', label: '上次测试时间' },
      { key: 'supplier_current_balance', label: '余额' },
      { key: 'supplier_today_cost', label: '今日消费' },
      { key: 'actions', label: '操作' },
    ])
    expect(accountColumnsSource).not.toContain("{ key: 'platform'")
    expect(accountsSource).not.toContain('Local Supplier Accounts')
    expect(accountsSource).not.toContain('<h1>上游账号</h1>')
    expect(accountsSource).not.toContain('只展示已同步到本地数据库的供应商账号')
    expect(accountsSource).toContain('sp-account-toolbar')
    expect(accountsSource).toContain('class="sp-filter-card-head"')
    expect(accountsSource).toContain('class="sp-account-filter-body"')
    expect(accountsSource).toContain('class="sp-account-filter-actions"')
    expect(accountsSource).toContain('筛选账号')
    expect(accountsSource).toContain('sp-account-table-shell')
    expect(accountsSource).toContain('sp-account-pagination')
    expect(accountsSource).toContain('pageSize = ref(20)')
    expect(accountsSource).toContain(':options="pageSizeOptions"')
    expect(accountsSource).toContain('v-model="platformFilter"')
    expect(accountsSource).toContain(':options="platformFilterOptions"')
    expect(accountsSource).toContain('platform: platformFilter.value || undefined')
    expect(accountsSource).toContain('platformBadgeClass')
    expect(accountsSource).toContain('platformLabel')
    expect(accountsSource).toContain('platformTextClass')
    expect(accountsSource).toContain('handlePageSizeChange')
    expect(accountsSource).toContain(':show-page-size-selector="false"')
    expect(accountsSource).toContain('@update:page="handlePageChange"')
    expect(accountsSource).toContain('.sp-account-table-shell :deep(.table-wrapper)')
    expect(accountsSource).toContain('overflow-x: auto')
    expect(accountsSource).toContain('@media (max-width: 760px)')
    expect(accountsSource).not.toContain('查询说明')
    expect(accountsSource).not.toContain('sp-grid-2')
  })

  it('keeps platform labels only in upstream-account secondary information', () => {
    const providerCellSource = accountsSource.match(
      /<template #cell-provider_name[\s\S]*?<\/template>/
    )?.[0]
    const upstreamAccountCellSource = accountsSource.match(
      /<template #cell-upstream_account_key[\s\S]*?<\/template>/
    )?.[0]

    expect(providerCellSource).not.toContain('platformBadgeClass(account.platform)')
    expect(providerCellSource).not.toContain('platformLabel(account.platform)')
    expect(providerCellSource).toContain('supplierTone(account.provider_id).chip')
    expect(upstreamAccountCellSource).toContain('platformBadgeClass(effectivePlatform(account))')
    expect(upstreamAccountCellSource).toContain('platformLabel(effectivePlatform(account))')
    expect(upstreamAccountCellSource).toContain("account.name || '—'")
    expect(upstreamAccountCellSource).toContain("account.upstream_account_key || '—'")
    expect(accountsSource).not.toContain('<template #cell-platform')
  })

  it('uses the supplier business-platform override before the matched local account platform', async () => {
    const wrapper = await mountSupplierAccounts()

    expect(wrapper.get('.runtime-cell-upstream_account_key[data-row-index="0"]').text()).toContain('Anthropic')
    expect(wrapper.get('.runtime-cell-upstream_account_key[data-row-index="0"]').text()).not.toContain('OpenAI')
    expect(wrapper.get('.runtime-cell-upstream_account_key[data-row-index="4"]').text()).toContain('Anthropic')
    expect(accountsSource).toContain('function effectivePlatform(account: SupplierProviderAccount): string')
    expect(accountsSource).toContain('normalizePlatform(account.effective_platform)')
    expect(accountsSource).toContain('normalizePlatform(account.local_account_platform)')
    expect(supplierProviderDataSource).toContain('platform_override?: string')
    expect(supplierProviderDataSource).toContain('effective_platform?: string')
    expect(supplierProviderDataSource).toContain('local_account_platform?: string')

    const view = wrapper.vm as unknown as {
      openSupplierBatchTestDialog: (snapshot: Record<string, unknown>) => Promise<void>
      batchTestTargets: Array<{ accountID: number; platform: string }>
    }
    await view.openSupplierBatchTestDialog({
      providerID: 0,
      groupID: 0,
      platform: '',
      quickFilter: 'all',
      summary: '全部账号',
    })

    expect(view.batchTestTargets).toEqual(expect.arrayContaining([
      expect.objectContaining({ accountID: 101, platform: 'anthropic' }),
      expect.objectContaining({ accountID: 105, platform: 'anthropic' }),
    ]))
    expect(view.batchTestTargets.some(target => target.platform === 'unknown')).toBe(false)

    wrapper.unmount()
  })

  it('provides a separate supplier-only business platform configuration action', () => {
    expect(accountsSource).toContain('配置业务平台')
    expect(accountsSource).toContain('@click.stop="openBusinessPlatformDialog(account)"')
    expect(accountsSource).toContain('title="配置业务平台"')
    expect(accountsSource).toContain('sp-business-platform-save')
    expect(accountsSource).toContain('sp-business-platform-cancel')
    expect(accountsSource).toContain(':global(.modal-content:has(.sp-business-platform-dialog))')
    expect(accountsSource).toContain('业务平台跟随接入平台')
    expect(accountsSource).toContain('setSupplierLocalAccountPlatformOverride')
    expect(accountsSource).toContain('clearSupplierLocalAccountPlatformOverride')
    expect(accountsSource).toContain('openLocalAccountEditor(account)')
    expect(supplierProviderDataSource).toContain('export async function setSupplierLocalAccountPlatformOverride')
    expect(supplierProviderDataSource).toContain('export async function clearSupplierLocalAccountPlatformOverride')
    expect(supplierProviderDataSource).toContain('`/admin/supplier-management/accounts/${localAccountID}/platform-override`')
  })

  it('reuses account duplication and assigns semantic colors to supplier account actions', () => {
    expect(supplierProviderDataSource).toContain('local_account_type?: string')
    expect(accountsSource).toContain('function canDuplicateLocalAccount(account: SupplierProviderAccount): boolean')
    expect(accountsSource).toContain('@click.stop="requestDuplicateLocalAccount(account)"')
    expect(accountsSource).toContain('function confirmDuplicateLocalAccount()')
    expect(accountsSource).toContain('ConfirmDialog')
    expect(accountsSource).toContain("'复制账号'")
    expect(accountsSource).toContain('adminAPI.accounts.duplicate(localAccountID)')
    expect(accountsSource).toContain('sp-account-action-view')
    expect(accountsSource).toContain('sp-account-action-test')
    expect(accountsSource).toContain('sp-account-action-edit')
    expect(accountsSource).toContain('sp-account-action-copy')
    expect(accountsSource).toContain('sp-account-action-platform')
    expect(accountsSource).toContain('sp-account-action-binding')
    expect(accountsSource).toContain('sp-account-action-delete')
  })

  it('duplicates a uniquely matched supported local account through the shared account API', async () => {
    const wrapper = await mountSupplierAccounts()
    const copyButton = wrapper
      .findAll('.runtime-cell-actions[data-row-index="0"] button')
      .find(button => button.text().includes('\u590d\u5236\u8d26\u53f7'))

    expect(copyButton).toBeDefined()
    await copyButton!.trigger('click')
    await flushPromises()

    // 点击后先弹出确认，不立即调用复制接口
    expect(supplierAccountMocks.duplicate).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('\u786e\u8ba4\u590d\u5236\u672c\u5730\u8d26\u53f7')

    const confirmButton = wrapper
      .findAll('button')
      .find(button => button.text().includes('\u786e\u8ba4\u590d\u5236') && !button.text().includes('\u786e\u8ba4\u590d\u5236\u672c\u5730\u8d26\u53f7'))
    expect(confirmButton).toBeDefined()
    await confirmButton!.trigger('click')
    await flushPromises()

    expect(supplierAccountMocks.duplicate).toHaveBeenCalledWith(101)
    expect(supplierAccountMocks.showSuccess).toHaveBeenCalledWith(
      '账号已复制为「本地账号 A (Copy)」，已暂停调度，请确认凭据后再启用'
    )
    expect(supplierAccountMocks.listAccounts).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('\u8d26\u53f7\u5df2\u590d\u5236')
    expect(wrapper.text()).toContain('(Copy)')

    wrapper.unmount()
  })

  it('shows local-account matching states, supplier summaries, and explicit missing values', () => {
    expect(accountsSource).toContain("account.local_account_match_status === 'unmatched'")
    expect(accountsSource).toContain("account.local_account_match_status === 'conflict'")
    expect(accountsSource).toContain("v-else-if=\"account.local_account_match_status === 'matched'\"")
    expect(accountsSource).toContain('未匹配')
    expect(accountsSource).toContain('匹配冲突（{{ account.local_account_match_count }}）')
    expect(accountsSource).toContain('isMatchedLocalAccount(account)')
    expect(accountsSource).toContain('handleToggleSchedulable(account)')
    expect(accountsSource).toContain('adminAPI.accounts.setSchedulable')
    expect(accountsSource).toContain('formatCNY(account.supplier_current_balance)')
    expect(accountsSource).toContain('formatCNY(account.supplier_today_cost)')
    expect(accountsSource).toContain("currency: 'CNY'")
    expect(accountsSource.match(/供应商汇总/g)?.length).toBeGreaterThanOrEqual(2)
    expect(accountsSource).toContain("if (value === null || value === undefined || value === '') return '—'")
    expect(accountsSource).toContain("if (account.local_account_match_status === 'matched') return '已匹配'")
    expect(accountsSource).toContain("return '—'")
  })

  it('edits a matched local-account priority inline through the account API', () => {
    expect(accountsSource).toContain('@click.stop="startPriorityEdit(account)"')
    expect(accountsSource).toContain('v-model="priorityDraft"')
    expect(accountsSource).toContain('@enter="savePriority(account)"')
    expect(accountsSource).toContain('@keydown.esc="cancelPriorityEdit"')
    expect(accountsSource).toContain('@blur="savePriority(account)"')
    expect(accountsSource).toContain('adminAPI.accounts.update(localAccountID, { priority: nextPriority })')
    expect(accountsSource).toContain('local_account_priority: priority')
    expect(accountsSource).toContain('请输入有效的整数优先级')
    expect(accountsSource).toContain('修改账号优先级失败')
  })

  it('opens the existing drawer from both row clicks and the small view button', () => {
    expect(accountsSource).toContain('@row-click="openDrawer"')
    expect(accountsSource).toContain('<template #cell-actions="{ row: account }">')
    expect(accountsSource).toContain('class="sp-button small')
    expect(accountsSource).toContain('@click.stop="openDrawer(account)"')
    expect(accountsSource).toContain('>查看</button>')
    expect(accountsSource).toContain('function openDrawer(account: SupplierProviderAccount)')
    expect(accountsSource).toContain(':show="Boolean(selected)"')
    expect(accountsSource).toContain('<SupplierDrawer')
    expect(accountsSource).toContain('本地账号状态')
    expect(accountsSource).toContain('是否调度')
    expect(accountsSource).toContain('测试结果')
    expect(accountsSource).toContain('上次测试时间')
    expect(accountsSource).toContain('余额（供应商汇总）')
    expect(accountsSource).toContain('今日消费（供应商汇总）')
  })

  it('provides edit, binding, and delete actions for matched local accounts', () => {
    expect(accountsSource).toContain("import { CreateAccountModal, EditAccountModal } from '@/components/account'")
    expect(accountsSource).toContain("import GroupSelector from '@/components/common/GroupSelector.vue'")
    expect(accountsSource).toContain('@click.stop="openLocalAccountEditor(account)"')
    expect(accountsSource).toContain('@click.stop="openAccountBindingEditor(account)"')
    expect(accountsSource).toContain('@click.stop="deleteLocalAccount(account)"')
    expect(accountsSource).toContain('<EditAccountModal')
    expect(accountsSource).toContain('<GroupSelector')
    expect(accountsSource).toContain('adminAPI.accounts.getById(localAccountID)')
    expect(accountsSource).toContain('adminAPI.accounts.update(account.id, { group_ids: bindingGroupIDs })')
    expect(accountsSource).toContain('adminAPI.accounts.delete(localAccountID)')
    expect(accountsSource).toContain('window.confirm')
    expect(accountsSource).toContain('账号已删除')
  })

  it('tests a uniquely matched local account through the shared account test modal', async () => {
    const wrapper = await mountSupplierAccounts()

    expect(wrapper.find('[data-test="supplier-account-test-101"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="supplier-account-test-105"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="supplier-account-test-2"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="supplier-account-test-3"]').exists()).toBe(false)

    await wrapper.get('[data-test="supplier-account-test-101"]').trigger('click')
    await flushPromises()

    expect(supplierAccountMocks.getAccountById).toHaveBeenCalledWith(101)
    expect(wrapper.get('.account-test-modal-stub').text()).toBe('本地账号 A')

    const testModal = wrapper.findComponent(AccountTestModalStub)
    testModal.vm.$emit('test-result', { accountId: 101, status: 'success' })
    await flushPromises()

    expect(supplierAccountMocks.listAccounts).toHaveBeenCalledTimes(2)
    expect(accountsSource).toContain("import AccountTestModal from '@/components/admin/account/AccountTestModal.vue'")
    expect(accountsSource).toContain('@test-result="handleAccountTestResult"')

    wrapper.unmount()
  })

  it('opens the existing create-account modal from the supplier account toolbar', () => {
    expect(accountsSource).toContain('@click="openCreateAccountDialog"')
    expect(accountsSource).toContain('<CreateAccountModal')
    expect(accountsSource).toContain('@created="handleAccountCreated"')
    expect(accountsSource).toContain('await loadAccountEditorOptions()')
    expect(accountsSource).toContain('添加账号')
  })

  it('opens the shared account rate guard log dialog from the account toolbar', () => {
    expect(accountsSource).toContain('倍率守护日志')
    expect(accountsSource).toContain('<SupplierAccountRateGuardLogDialog')
    expect(accountsSource).toContain('@click="openAccountRateGuardLogs"')
    expect(accountsSource).not.toContain('@click.stop="openAccountRateGuardLogs(account)"')
  })

  it('shows pending account rate guard log count on the toolbar button', () => {
    expect(accountsSource).toContain("import { listAccountRateGuardUnbindLogs, listTasks as listAutomationTasks } from '@/api/admin/supplierAutomation'")
    expect(accountsSource).toContain('const accountRateGuardPendingCount = ref(0)')
    expect(accountsSource).toContain('await loadAccountRateGuardPendingCount()')
    expect(accountsSource).toContain("result: 'unbound'")
    expect(accountsSource).toContain("status: 'pending'")
    expect(accountsSource).toContain('accountRateGuardPendingCount.value = Number(result.pending_count) || 0')
    expect(accountsSource).toContain('v-if="accountRateGuardPendingCount > 0"')
    expect(accountsSource).toContain('class="sp-account-rate-guard-pending-count"')
    expect(accountsSource).toContain('@pending-count-change="updateAccountRateGuardPendingCount"')
    expect(accountsSource).toContain('data-test="supplier-account-rate-guard-pending-count"')
    expect(accountsSource).toContain('@click="refreshAccountsWorkbench"')
    expect(accountsSource).toContain('async function refreshAccountsWorkbench()')
  })

  it('runs an independently implemented batch test for all filtered matched accounts', () => {
    expect(accountsSource).toContain('测试当前筛选')
    expect(accountsSource).toContain('type SupplierAccountFilterSnapshot = {')
    expect(accountsSource).toContain('quickFilter: AccountQuickFilterKey')
    expect(accountsSource).toContain('quickFilter: accountQuickFilter.value')
    expect(accountsSource).toContain('loadFilteredTestAccounts(snapshot: SupplierAccountFilterSnapshot)')
    expect(accountsSource).toContain('accountMatchesQuickFilter(account, snapshot.quickFilter)')
    expect(accountsSource).toContain('loadFilteredTestAccounts(snapshot)')
    expect(accountsSource).toContain('batchTestFilterSummary.value = snapshot.summary')
    expect(accountsSource).toContain('{{ batchTestFilterSummary }}')
    expect(accountsSource).toContain('page_size: SUPPLIER_BATCH_TEST_PAGE_SIZE')
    expect(accountsSource).toContain("account.local_account_match_status === 'matched'")
    expect(accountsSource).toContain('const uniqueTargets = new Map<number, SupplierBatchTestTarget>()')
    expect(accountsSource).toContain('uniqueTargets.has(localAccountID)')
    expect(accountsSource).toContain('startSupplierAccountBatchTest')
    expect(accountsSource).toContain('getSupplierAccountBatchTestJob')
    expect(accountsSource).toContain('cancelSupplierAccountBatchTestJob')
    expect(accountsSource).toContain('title="供应商账号批量测试"')
    expect(accountsSource).toContain('title="批量测试结果"')
    expect(accountsSource).toContain(':disabled="!batchTesting && (loading || batchTestPreparing || total === 0)"')
    expect(accountsSource).toContain("return status === 'queued' || status === 'running'")
  })

  it('uses a wide filtered batch-result view with failure-first ordering and platform colors', () => {
    expect(accountsSource).toContain("| 'failed_schedulable'")
    expect(accountsSource).toContain("| 'failed_unschedulable'")
    expect(accountsSource).toContain("| 'success_unschedulable'")
    expect(accountsSource).toContain("| 'success_upstream_disabled'")
    expect(accountsSource).toContain("| 'skipped'")
    expect(accountsSource).toContain("const batchTestResultFilter = ref<SupplierBatchResultFilter>('all')")
    expect(accountsSource).toContain('const batchTestResultFilterOptions = computed')
    expect(accountsSource).toContain('const filteredBatchTestResultItems = computed')
    expect(accountsSource).toContain('function batchTestIsSkipped(item: BatchAccountTestItem)')
    expect(accountsSource).toContain('function batchTestIsFailedSchedulable(item: BatchAccountTestItem)')
    expect(accountsSource).toContain('function batchTestIsFailedUnschedulable(item: BatchAccountTestItem)')
    expect(accountsSource).toContain('function batchTestIsSuccessUnschedulable(item: BatchAccountTestItem)')
    expect(accountsSource).toContain('function batchTestIsSuccessUpstreamDisabled(item: BatchAccountTestItem)')
    expect(accountsSource).toContain(':data-test="`supplier-batch-result-filter-${option.key}`"')
    expect(accountsSource).toContain('`batch-result-tab-${option.key}`')
    expect(accountsSource).toContain('v-for="item in filteredBatchTestResultItems"')
    expect(accountsSource).toContain("['sp-platform-badge', platformBadgeClass(")
    expect(accountsSource).toContain('width="full"')
    expect(accountsSource).toContain(':global(.modal-content:has(.sp-batch-result-dialog))')
    expect(accountsSource).toContain('width: min(1600px, calc(100vw - 32px))')
  })

  it('shows and switches schedulable state inside supplier batch-test results', () => {
    expect(accountsSource).toContain('schedulable: boolean')
    expect(accountsSource).toContain('providerEnabled: boolean')
    expect(accountsSource).toContain('batchTestItemSchedulable(item)')
    expect(accountsSource).toContain('失败但仍在调度')
    expect(accountsSource).toContain(':data-test="`supplier-batch-result-schedulable-toggle-${item.account_id}`"')
    expect(accountsSource).toContain('@click="toggleBatchTestItemSchedulable(item)"')
    expect(accountsSource).toContain('async function toggleBatchTestItemSchedulable(item: BatchAccountTestItem)')
    expect(accountsSource).toContain('adminAPI.accounts.setSchedulable(accountID, nextSchedulable)')
    expect(accountsSource).toContain('local_account_schedulable: schedulable')
    expect(accountsSource).toContain('.batch-result-schedule-status')
    expect(accountsSource).toContain('.batch-result-card-actions')
    expect(accountsSource).toContain('.batch-result-schedule-button')
    expect(accountsSource).toContain('.batch-result-risk-tag')
  })

  it('filters supplier batch results and updates schedulable state after switching', async () => {
    const wrapper = await mountSupplierAccounts()
    const view = wrapper.vm as unknown as {
      showBatchTestResultDialog: boolean
      batchTestTargets: Array<{
        accountID: number
        accountName: string
        platform: string
        schedulable: boolean
        providerEnabled: boolean
      }>
      batchTestResult: {
        job_id: string
        status: string
        total: number
        completed: number
        success: number
        failed: number
        results: Array<Record<string, unknown>>
      }
    }

    view.batchTestTargets = [
      {
        accountID: 101,
        accountName: '失败账号',
        platform: 'openai',
        schedulable: false,
        providerEnabled: true,
      },
      {
        accountID: 105,
        accountName: '禁用供应商成功账号',
        platform: 'anthropic',
        schedulable: true,
        providerEnabled: false,
      },
    ]
    view.batchTestResult = {
      job_id: 'supplier-batch-test',
      status: 'completed',
      total: 2,
      completed: 2,
      success: 1,
      failed: 1,
      results: [
        { account_id: 101, status: 'failed', latency_ms: 120, schedulable: false },
        { account_id: 105, status: 'success', latency_ms: 80, schedulable: true },
      ],
    }
    view.showBatchTestResultDialog = true
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-test="supplier-batch-result-filter-failed_unschedulable"]').text())
      .toContain('失败且调度关闭1')
    expect(wrapper.get('[data-test="supplier-batch-result-filter-success_upstream_disabled"]').text())
      .toContain('成功且上游禁用1')

    await wrapper.get('[data-test="supplier-batch-result-filter-failed_unschedulable"]').trigger('click')
    expect(wrapper.findAll('.batch-result-card')).toHaveLength(1)
    expect(wrapper.get('.batch-result-schedule-status').text()).toBe('调度关闭')

    await wrapper.get('[data-test="supplier-batch-result-schedulable-toggle-101"]').trigger('click')
    await flushPromises()

    expect(supplierAccountMocks.setSchedulable).toHaveBeenCalledWith(101, true)
    expect(supplierAccountMocks.showSuccess).toHaveBeenCalledWith('账号调度已打开')
    expect(wrapper.get('[data-test="supplier-batch-result-filter-failed_unschedulable"]').text())
      .toContain('失败且调度关闭0')
    expect(wrapper.find('.batch-result-card').exists()).toBe(false)
  })
  it('matches the upstream batch-test result structure and button hierarchy', () => {
    expect(accountsSource).toContain('class="sp-batch-summary-metric accounts"')
    expect(accountsSource).toContain('class="sp-batch-summary-metric platforms"')
    expect(accountsSource).toContain("['sp-batch-test-platform-accent', platformAccentBarClass(summary.platform)]")
    expect(accountsSource).toContain('class="sp-button sp-batch-start-button"')
    expect(accountsSource).toContain('class="sync-confirm-summary"')
    expect(accountsSource).toContain('class="sync-confirm-body"')
    expect(accountsSource).toContain('class="sync-confirm-section"')
    expect(accountsSource).toContain('class="batch-result-toolbar"')
    expect(accountsSource).toContain('class="batch-result-list"')
    expect(accountsSource).toContain('batchTestItemTone(item.status),')
    expect(accountsSource).toContain("{ 'failed-schedulable': batchTestIsFailedSchedulable(item) }")
    expect(accountsSource).toContain('class="batch-result-button"')
    expect(accountsSource).toContain('class="batch-result-button batch-result-button-primary"')
    expect(accountsSource).toContain('.batch-result-button-primary')
    expect(accountsSource).toContain('background: #059669')
    expect(accountsSource).toContain('color: #fff')
  })

  it('uses semantic color layers for both supplier batch-test dialogs', () => {
    expect(accountsSource).toContain('class="sync-result-stat total"')
    expect(accountsSource).toContain('class="sync-result-stat completed"')
    expect(accountsSource).toContain('`batch-result-tab-${option.key}`')
    expect(accountsSource).toContain(':global(.modal-content:has(.sp-batch-test-dialog)),')
    expect(accountsSource).toContain(':global(.modal-content:has(.sp-batch-result-dialog)) {')
    expect(accountsSource).toContain('--sp-panel: #ffffff;')
    expect(accountsSource).toContain(':global(.dark .modal-content:has(.sp-batch-test-dialog)),')
    expect(accountsSource).toContain('--sp-panel: #172033;')
    expect(accountsSource).toContain('.sp-batch-test-dialog {')
    expect(accountsSource).toContain('.sp-batch-result-dialog {')
    expect(accountsSource).toContain('.sync-result-stat.total')
    expect(accountsSource).toContain('.sync-result-stat.completed')
    expect(accountsSource).toContain('.batch-result-card.success')
    expect(accountsSource).toContain('.batch-result-card.failed')
    expect(accountsSource).toContain('.batch-result-tab-failed.active')
    expect(accountsSource).toContain('color-mix(in srgb, var(--sp-green)')
    expect(accountsSource).toContain('color-mix(in srgb, var(--sp-red)')
  })

  it('compresses supplier batch-test result summary on mobile', () => {
    expect(accountsSource).toContain('@media (max-width: 640px)')
    expect(accountsSource).toContain(':global(.modal-overlay:has(.sp-batch-result-dialog))')
    expect(accountsSource).toContain(':global(.modal-content:has(.sp-batch-result-dialog))')
    expect(accountsSource).toContain('height: 100%')
    expect(accountsSource).toContain('.sp-batch-result-dialog .batch-result-scroll')
    expect(accountsSource).toContain('overflow: auto')
    expect(accountsSource).toContain('scroll-padding-bottom: 2rem')
    expect(accountsSource).toContain('.sp-batch-result-dialog .batch-result-list')
    expect(accountsSource).toContain('padding: 0.5rem 0.5rem 2rem')
    expect(accountsSource).toContain('.sp-batch-result-dialog .batch-result-hint')
    expect(accountsSource).toContain('display: none')
    expect(accountsSource).toContain('grid-template-columns: repeat(2, minmax(0, 1fr))')
    expect(accountsSource).toContain('env(safe-area-inset-bottom, 0px)')
  })

  it('declares supplier-account-specific batch-test API functions', () => {
    expect(supplierProviderDataSource).toContain('export async function startSupplierAccountBatchTest')
    expect(supplierProviderDataSource).toContain('export async function getSupplierAccountBatchTestJob')
    expect(supplierProviderDataSource).toContain('export async function cancelSupplierAccountBatchTestJob')
    expect(supplierProviderDataSource).toContain("'/admin/supplier-management/accounts/batch-test'")
    expect(supplierProviderDataSource).toContain('`/admin/supplier-management/accounts/batch-test/${jobID}`')
    expect(supplierProviderDataSource).toContain('`/admin/supplier-management/accounts/batch-test/${jobID}/cancel`')
    expect(supplierProviderDataSource).toContain('model_ids_by_account?: Record<number, string>')
    expect(accountsSource).toContain('selectedBatchTestModelsByAccount')
    expect(accountsSource).toContain('model_ids_by_account: modelIDsByAccount')
  })

  it('uses upstream-account semantics for the empty state', () => {
    expect(accountsSource).toContain('暂无上游账号数据')
    expect(accountsSource).toContain('请先同步供应商上游账号，或调整当前筛选条件。')
    expect(accountsSource).not.toContain('暂无本地账号数据')
  })
  it('uses full-filter summary cards and keeps group controls easy to scan', () => {
		expect(groupsSource).not.toContain('Supplier Group Matching')
		expect(groupsSource).not.toContain('<h1>分组管理</h1>')
		expect(groupsSource).not.toContain('对照最近一次采集到的上游分组与本地分组')
		expect(groupsSource).toContain('sp-filter-toolbar')
    expect(groupsSource).toContain('class="sp-filter-card-head"')
    expect(groupsSource).toContain('class="sp-filter-card-body"')
    expect(groupsSource).toContain('class="sp-filter-control')
    expect(groupsSource).toContain('筛选分组')
		expect(groupsSource).toContain('sp-filter-fields')
		expect(groupsSource).toContain('sp-filter-actions')
		expect(groupsSource).toContain('v-model="platformFilter"')
		expect(groupsSource).toContain('v-model="matchStatusFilter"')
		expect(groupsSource).toContain('v-model="rateStatusFilter"')
		expect(groupsSource).toContain(':searchable="true"')
    expect(groupsSource).toContain('resetGroupFilters')
    expect(groupsSource).toContain('canResetFilters')
    expect(groupsSource).toContain('handleGroupPageSizeChange')
    expect(groupsSource).toContain('pageSize = ref(20)')
    expect(groupsSource).toContain('@update:pageSize="handleGroupPageSizeChange"')
    expect(groupsSource).toContain(':show-page-size-selector="true"')
    expect(groupsSource).toContain("import StatCard from '@/components/common/StatCard.vue'")
    expect(groupsSource).toContain('<StatCard')
    expect(groupsSource).toContain('上游分组')
    expect(groupsSource).toContain('已匹配')
    expect(groupsSource).toContain('待匹配')
    expect(groupsSource).toContain('倒挂风险')
    expect(groupsSource).toContain('groupSummary')
    expect(groupsSource).toContain('result.summary')
    expect(groupsSource).toContain('matchedGroupRate')
    expect(groupsSource).toContain('unmatchedGroupRate')
    expect(groupsSource).not.toContain('var(--sp-bg)')
    expect(groupsSource).not.toContain('<article')
    expect(groupsSource).not.toContain('signalCards')
    const columns = [
      "{ key: 'provider_name', label: '供应商'",
      "{ key: 'name', label: '上游分组'",
      "{ key: 'rate_multiplier', label: '上游倍率'",
      "{ key: 'raw_status', label: '上游状态'",
      "{ key: 'local_group_name', label: '匹配本地分组'",
      "{ key: 'local_rate_multiplier', label: '本地分组倍率'",
      "{ key: 'rate_delta', label: '价差'",
      "{ key: 'account_count', label: '绑定账号'",
      "{ key: 'rate_status', label: '倍率状态'",
      "{ key: 'actions', label: '操作'",
    ]
    columns.forEach(column => expect(groupsSource).toContain(column))
    expect(groupsSource).toContain('修改后价差')
    expect(groupsSource).not.toContain('收益倍率')
    expect(groupsSource).not.toContain('formatProfitRate')
    columns.slice(1).forEach((column, index) => {
      expect(groupsSource.indexOf(columns[index])).toBeLessThan(groupsSource.indexOf(column))
    })
    expect(groupsSource).toContain('sp-console-shell')
    expect(groupsSource).toContain('sp-summary-grid')
		expect(groupsSource).toContain('sp-summary-filter')
		expect(groupsSource).toContain('applySummaryFilter')
		expect(groupsSource).toContain(':aria-pressed="isSummaryFilterActive')
    expect(groupsSource).toContain('sp-console-panel')
    expect(groupsSource).toContain('sp-table-shell')
    expect(groupsSource).toContain('height: min(64vh, 680px)')
    const mobileMediaIndex = groupsSource.indexOf('@media (max-width: 760px)')
    expect(groupsSource.indexOf('.sp-table-shell { height: auto;', mobileMediaIndex)).toBeGreaterThan(mobileMediaIndex)
		const reducedMotionIndex = groupsSource.indexOf('@media (prefers-reduced-motion: reduce)')
		const mobileStyles = groupsSource.slice(mobileMediaIndex, reducedMotionIndex)
		expect(mobileStyles).toContain('.sp-summary-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }')
		expect(mobileStyles).not.toContain('.sp-summary-grid { grid-template-columns: 1fr; }')
  })

  it('显示上游分组记录及其同步状态', () => {
    expect(groupsSource).not.toContain('activeFilter')
    expect(groupsSource).not.toContain("{ key: 'active'")
    expect(groupsSource).not.toContain('selected.active')
    expect(groupsSource).not.toContain('inactiveGroupCount')
    expect(groupsSource).not.toContain('失效记录')
    expect(groupsSource).toContain('group.active')
    expect(groupsSource).toContain('group.inactive_at')
  })

  it('uses existing dialogs and APIs for local-group operations', () => {
    expect(groupsSource).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(groupsSource).toContain("import ConfirmDialog from '@/components/common/ConfirmDialog.vue'")
    expect(groupsSource).toContain('updateSupplierGroupMapping')
		expect(groupsSource).toContain('updateSupplierGroupRateGuard')
    expect(groupsSource).toContain('adminAPI.groups.create')
    expect(groupsSource).toContain('adminAPI.groups.update')
    expect(groupsSource).toContain('匹配分组')
    expect(groupsSource).toContain('新建分组')
    expect(groupsSource).toContain('调倍率')
    expect(groupsSource).toContain('更换本地分组')
    expect(groupsSource).toContain('取消关联')
		expect(groupsSource).toContain("{ key: 'rate_guard_status', label: '倍率守护'")
		expect(groupsSource).toContain('toggleRateGuard')
		expect(groupsSource).toContain('设为守护')
  })

  it('uses distinct supplier tones and the existing group platform palette', () => {
    expect(groupsSource).toContain("import GroupBadge from '@/components/common/GroupBadge.vue'")
    expect(groupsSource).toContain('<GroupBadge')
    expect(groupsSource).toContain('SUPPLIER_TONES')
    expect(groupsSource).toContain('supplierTone(group.provider_id)')
    expect(groupsSource).toContain('sp-supplier-chip')
    expect(groupsSource).toContain('groupPlatform(groupEffectivePlatform(group))')
    expect(groupsSource).toContain('UPSTREAM_GROUP_TONES')
    expect(groupsSource).toContain('upstreamGroupTone(group.upstream_group_key)')
    expect(groupsSource).toContain('sp-upstream-group-chip')
    expect(groupsSource).toContain(
      `<span :class="['sp-rate-value', platformTextClass(group.local_group_platform || '')]">`
    )
    expect(groupsSource).toContain(
      `class="sp-rate-value" :class="platformTextClass(groupEffectivePlatform(group))"`
    )
    expect(groupsSource).not.toContain('upstreamRateTone')
    expect(groupsSource).not.toContain('UPSTREAM_RATE_TONES')
    expect(groupsSource).not.toContain('getSupplierUpstreamRateBand')
    expect(groupsSource).not.toContain('.sp-rate-value.upstream')
    expect(groupsSource).toContain('{{ supplierTypeLabel(group.provider_id) }}</span>')
    expect(groupsSource).toContain('<span>#{{ group.provider_id }}</span>')
    expect(groupsSource).not.toContain('【{{ supplierTypeLabel(group.provider_id) }}】')
    expect(groupsSource).toContain('SUPPLIER_TYPE_TONES')
    expect(groupsSource).toContain('supplierTypeTone(group.provider_id)')
    expect(groupsSource).toContain('sp-provider-type')
    expect(groupsSource).toContain('【{{ upstreamPlatformLabel(group) }}】')
  })

  it('uses semantic themes for supplier group dialogs', () => {
    expect(groupsSource).toContain('sp-dialog-context match')
    expect(groupsSource).toContain('sp-dialog-context create')
    expect(groupsSource).toContain('sp-dialog-context rate')
    expect(groupsSource).toContain('sp-dialog-primary match')
    expect(groupsSource).toContain('sp-dialog-primary create')
    expect(groupsSource).toContain('sp-dialog-primary rate')
    expect(groupsSource).toContain('sp-dialog-primary log')
    expect(groupsSource).toContain('localRateDeltaTone')
    expect(groupsSource).toContain('.sp-match-preview-card.platform')
    expect(groupsSource).toContain('.sp-rate-recommendation.create')
    expect(groupsSource).toContain('.sp-rate-recommendation.danger')
    expect(groupsSource).toContain(':global(.dark) .sp-dialog-context')
  })

  it('keeps supplier group dialog theme variables after teleport', () => {
    expect(groupsSource).toContain(':global(.modal-content:has(.sp-dialog-context))')
    expect(groupsSource).toContain('--sp-panel: #ffffff')
    expect(groupsSource).toContain('--sp-cyan: #3b82f6')
    expect(groupsSource).toContain('--sp-amber: #d97706')
    expect(groupsSource).toContain(':global(.modal-content:has(.sp-dialog-context.match))')
    expect(groupsSource).toContain(':global(.modal-content:has(.sp-dialog-context.rate))')
    expect(groupsSource).toContain('--sp-dialog-shell-accent')
  })

  it('adds supplier quick filters to the group table header', () => {
    expect(groupsSource).toContain('sp-provider-shortcuts')
    expect(groupsSource).toContain('quickProviderOptions')
    expect(groupsSource).toContain('selectProviderShortcut')
    expect(groupsSource).toContain(':aria-pressed="providerID === option.value"')
    expect(groupsSource).toContain('@click="selectProviderShortcut(option.value)"')
  })

  it('widens desktop supplier quick filters in the group table header', () => {
    expect(groupsSource).toContain('max-width: min(58vw, 52rem);')
  })

  it('supports toggling participation in automatic rate guard for a supplier group', () => {
    expect(groupsSource).toContain('updateSupplierGroupRateGuardEnabled')
    expect(groupsSource).toContain('toggleRateGuardEnabled')
    expect(groupsSource).toContain('参与守护')
    expect(groupsSource).toContain('已暂停参与')
    expect(groupsSource).toContain('保留守护来源，已暂停自动倍率调整')
    expect(supplierProviderDataSource).toContain('rate_guard_enabled')
    expect(supplierProviderDataSource).toContain('updateSupplierGroupRateGuardEnabled')
  })

})
