import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, mount } from '@vue/test-utils'
import { h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UpstreamProvidersView from './UpstreamProvidersView.vue'

const upstreamProvidersSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'UpstreamProvidersView.vue'),
  'utf-8'
)

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const { adminAPIMock, routeMock, routerReplaceMock } = vi.hoisted(() => ({
  adminAPIMock: {
    upstreamProviders: {
      list: vi.fn().mockResolvedValue([]),
      getBalance: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      testConfig: vi.fn(),
      setDefault: vi.fn(),
      testSaved: vi.fn(),
      getKeys: vi.fn(),
    },
    upstreamAccountSync: {
      getBalanceConsumption: vi.fn(),
      updateBalanceSamplerConfig: vi.fn(),
      addBalanceRecharge: vi.fn(),
      runBalanceSampleNow: vi.fn(),
      getHealthGuardConfig: vi.fn(),
      updateHealthGuardConfig: vi.fn(),
      runHealthGuardNow: vi.fn(),
      getHealthGuardRecords: vi.fn(),
      getHealthGuardPollLogs: vi.fn(),
    },
    accounts: {
      list: vi.fn(),
      getById: vi.fn(),
    },
  },
  routeMock: { query: {} as Record<string, string> },
  routerReplaceMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeMock,
  useRouter: () => ({ replace: routerReplaceMock }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: adminAPIMock,
}))

vi.mock('@/api/admin/index', () => ({
  adminAPI: {
    upstreamProviders: adminAPIMock.upstreamProviders,
    upstreamAccountSync: adminAPIMock.upstreamAccountSync,
    accounts: adminAPIMock.accounts,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/components/admin/upstream/UpstreamBalanceCharts.vue', () => ({
  default: {
    props: ['overview', 'loading', 'days'],
    template: '<div data-test="balance-charts">{{ days }}-{{ Boolean(overview) }}-{{ loading }}</div>',
  },
}))

const SelectStub = {
  props: ['modelValue', 'options', 'placeholder'],
  emits: ['update:modelValue', 'change'],
  template: `
    <select
      data-test="select-stub"
      :value="modelValue ?? ''"
      @change="$emit('update:modelValue', $event.target.value ? Number($event.target.value) : null)"
    >
      <option value="">{{ placeholder }}</option>
      <option v-for="option in options" :key="option.value" :value="option.value" :disabled="option.disabled">
        {{ option.label }} {{ option.meta }}
      </option>
    </select>
  `,
}

describe('UpstreamProvidersView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    routeMock.query = {}
    adminAPIMock.upstreamProviders.list.mockResolvedValue([])
    adminAPIMock.upstreamProviders.getBalance.mockResolvedValue({
      provider_slug: 'sub-main',
      provider_name: 'Sub Main',
      provider_type: 'sub2api',
      balance: 334.74079414,
    })
    adminAPIMock.upstreamProviders.create.mockResolvedValue({})
    adminAPIMock.upstreamProviders.update.mockResolvedValue({})
    adminAPIMock.upstreamProviders.testConfig.mockResolvedValue({})
    adminAPIMock.upstreamProviders.setDefault.mockResolvedValue({})
    adminAPIMock.upstreamProviders.testSaved.mockResolvedValue({})
    adminAPIMock.upstreamProviders.getKeys.mockResolvedValue({ items: [], warnings: [] })
    adminAPIMock.upstreamAccountSync.getBalanceConsumption.mockResolvedValue({
      config: {
        enabled: false,
        interval_seconds: 3600,
        provider_amount_scales: {},
      },
      summaries: {},
      rows: [],
    })
    adminAPIMock.upstreamAccountSync.updateBalanceSamplerConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
      provider_amount_scales: {},
    })
    adminAPIMock.upstreamAccountSync.addBalanceRecharge.mockResolvedValue({})
    adminAPIMock.upstreamAccountSync.runBalanceSampleNow.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
      provider_amount_scales: {},
    })
    adminAPIMock.upstreamAccountSync.getHealthGuardConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
      max_accounts_per_run: 200,
      concurrency: 3,
      timeout_per_account_seconds: 90,
      failure_threshold: 3,
      slow_threshold: 3,
      recovery_threshold: 2,
      healthy_latency_ms: 15000,
      platform_models: {},
      platform_latency_ms: {},
    })
    adminAPIMock.upstreamAccountSync.updateHealthGuardConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
      max_accounts_per_run: 200,
      concurrency: 3,
      timeout_per_account_seconds: 90,
      failure_threshold: 3,
      slow_threshold: 3,
      recovery_threshold: 2,
      healthy_latency_ms: 15000,
      platform_models: {},
      platform_latency_ms: {},
    })
    adminAPIMock.upstreamAccountSync.runHealthGuardNow.mockResolvedValue({
      config: {
        enabled: false,
        interval_seconds: 3600,
        max_accounts_per_run: 200,
        concurrency: 3,
        timeout_per_account_seconds: 90,
        failure_threshold: 3,
        slow_threshold: 3,
        recovery_threshold: 2,
        healthy_latency_ms: 15000,
        platform_models: {},
        platform_latency_ms: {},
      },
      record: {
        id: 'run-1',
        trigger: 'manual',
        status: 'success',
        started_at: '2026-07-05T00:00:00Z',
        finished_at: '2026-07-05T00:00:01Z',
        summary: {
          total_accounts: 1,
          checked_count: 1,
          healthy_count: 1,
          slow_count: 0,
          failed_count: 0,
          skipped_count: 0,
          disabled_count: 0,
          recovered_count: 0,
          unchanged_count: 1,
        },
        items: [],
      },
    })
    adminAPIMock.upstreamAccountSync.getHealthGuardRecords.mockResolvedValue([])
    adminAPIMock.upstreamAccountSync.getHealthGuardPollLogs.mockResolvedValue([])
    adminAPIMock.accounts.list.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    adminAPIMock.accounts.getById.mockImplementation(async (id: number) => ({
      id,
      name: `account-${id}`,
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
    }))
  })

  it.each([
    ['balance-sampler', '.balance-sampler-dialog'],
    ['health-guard', '.health-guard-dialog'],
  ])('opens the %s settings dialog from the automation query', async (automation, selector) => {
    routeMock.query = { automation }
    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="after-table" /></div>' },
          DataTable: { template: '<div />' },
          BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true, EmptyState: true, Icon: true, Select: true, Toggle: true,
          UpstreamBalanceCharts: { template: '<div />' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find(selector).exists()).toBe(true)
    expect(routerReplaceMock).toHaveBeenCalledWith({ query: {} })
  })

  it('accepts 0.1 as an account rate conversion factor', async () => {
    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.createProvider'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')

    const input = wrapper.find('input[type="number"][required]')
    expect(input.exists()).toBe(true)

    const element = input.element as HTMLInputElement
    element.value = '0.1'

    expect(element.checkValidity()).toBe(true)
  })

  it('puts homepage first, balance before actions, and automatically fetches provider balance', async () => {
    adminAPIMock.upstreamProviders.getBalance.mockResolvedValue({
      provider_slug: 'sub-main',
      provider_name: 'Sub Main',
      provider_type: 'sub2api',
      balance: 9.5,
      today_cost: 1.75,
    })
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'sub-main',
        name: 'Sub Main',
        enabled: true,
        is_default: true,
        base_url: 'https://upstream.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        account_rate_multiplier_scale: 1,
      },
    ])

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns', 'data'],
            setup(props, { slots }) {
              return () => h('div', props.data.flatMap((row: any) => [
                h('div', { class: 'columns' }, props.columns.map((column: any) => column.key).join(',')),
                h('div', { class: 'homepage-cell' }, slots['cell-homepage']?.({ row })),
                h('div', { class: 'name-cell' }, slots['cell-name']?.({ row })),
                h('div', { class: 'enabled-cell' }, slots['cell-enabled']?.({ row })),
                h('div', { class: 'balance-cell' }, slots['cell-balance']?.({ row })),
                h('div', { class: 'today-cost-cell' }, slots['cell-today_consumption']?.({ row })),
                h('div', { class: 'actions-cell' }, slots['cell-actions']?.({ row })),
              ]))
            },
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.columns').text()).toBe('homepage,name,enabled,sort_order,prefix,rate_scale,balance,today_consumption,actions')
    expect(wrapper.find('.provider-name-card').exists()).toBe(true)
    expect(wrapper.find('.provider-type-tag').exists()).toBe(true)
    expect(wrapper.find('.provider-name-card [role="switch"]').exists()).toBe(false)
    expect(wrapper.find('.enabled-cell [role="switch"]').exists()).toBe(true)

    const homepage = wrapper.find('a[href="https://upstream.example.com"]')
    expect(homepage.exists()).toBe(true)
    expect(homepage.attributes('target')).toBe('_blank')

    const balanceButton = wrapper
      .findAll('button')
      .find((button) => button.attributes('title') === 'admin.upstreamProviders.fetchBalance')
    expect(balanceButton).toBeTruthy()

    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledWith('sub-main')
    expect(wrapper.text()).toContain('9.5000')
    expect(wrapper.find('.numeric-balance').exists()).toBe(true)
    expect(wrapper.find('.numeric-alert').exists()).toBe(true)
    expect(wrapper.find('.today-cost-cell').text()).toContain('1.7500')

    await balanceButton!.trigger('click')
    await flushPromises()
    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledTimes(2)
  })

  it('enables money sorting for provider balance and today cost headers', () => {
    expect(upstreamProvidersSource).toContain("{ key: 'balance', label: t('admin.upstreamProviders.columns.balance'), sortable: true")
    expect(upstreamProvidersSource).toContain("{ key: 'today_consumption', label: t('admin.upstreamProviders.columns.todayCost'), sortable: true")
    expect(upstreamProvidersSource).toContain('balance: providerBalanceForSort(provider.slug),')
    expect(upstreamProvidersSource).toContain('today_consumption: todayConsumptionForProvider(provider.slug),')
  })

  it('toggles provider enabled state from the status column with Toggle', async () => {
    const provider = {
      type: 'sub2api',
      slug: 'sub-main',
      name: 'Sub Main',
      enabled: true,
      is_default: false,
      base_url: 'https://upstream.example.com',
      login_url: '/api/v1/auth/login',
      api_keys_url: '/api/admin/keys',
      account_rate_multiplier_scale: 1,
    }
    adminAPIMock.upstreamProviders.list.mockResolvedValue([provider])
    adminAPIMock.upstreamProviders.update.mockResolvedValue({ ...provider, enabled: false })

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns', 'data'],
            setup(props, { slots }) {
              return () => h('div', props.data.flatMap((row: any) => [
                h('div', { class: 'columns' }, props.columns.map((column: any) => column.key).join(',')),
                h('div', { class: 'enabled-cell' }, slots['cell-enabled']?.({ row })),
              ]))
            },
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.columns').text()).toContain('enabled')
    const toggle = wrapper.find('.enabled-cell [role="switch"]')
    expect(toggle.exists()).toBe(true)
    expect(toggle.attributes('aria-checked')).toBe('true')

    await toggle.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamProviders.update).toHaveBeenCalledWith(
      'sub-main',
      expect.objectContaining({
        enabled: false,
        slug: 'sub-main',
      })
    )
  })

  it('marks disabled provider rows for muted table styling', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'disabled-main',
        name: 'Disabled Main',
        enabled: false,
        is_default: false,
        base_url: 'https://disabled.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        account_rate_multiplier_scale: 1,
      },
    ])

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns', 'data', 'rowClass'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any, index: number) => h('div', {
                class: ['provider-row', typeof props.rowClass === 'function' ? props.rowClass(row, index) : props.rowClass],
              }, [
                h('div', { class: 'name-cell' }, slots['cell-name']?.({ row })),
                h('div', { class: 'enabled-cell' }, slots['cell-enabled']?.({ row })),
              ])))
            },
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.provider-row').classes()).toContain('provider-disabled-row')
    expect(wrapper.find('.name-cell').text()).toContain('Disabled Main')
    expect(wrapper.find('.enabled-cell').text()).toContain('common.disabled')
    expect(upstreamProvidersSource).toContain(':deep(.provider-disabled-row td > *)')
    expect(upstreamProvidersSource).toContain('filter: grayscale(1);')
    expect(upstreamProvidersSource).toContain('opacity: 0.46;')
  })

  it('uses distinct color treatments for balance, today cost, and low balance warnings', () => {
    expect(upstreamProvidersSource).toContain('@apply text-lg font-bold text-teal-600 dark:text-teal-300;')
    expect(upstreamProvidersSource).toContain('@apply text-lg font-bold text-emerald-600 dark:text-emerald-300;')
    expect(upstreamProvidersSource).toContain('@apply rounded-md bg-red-50 font-bold text-red-700 ring-1 ring-red-100')
  })

  it('renders balance charts below the table layout instead of inside the table scroll container', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'sub-main',
        name: 'Sub Main',
        enabled: true,
        is_default: true,
        base_url: 'https://upstream.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        account_rate_multiplier_scale: 1,
      },
    ])

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div data-test="page"><slot /></div>' },
          TablePageLayout: { template: '<div data-test="table-layout"><div data-test="filters"><slot name="filters" /></div><div data-test="table"><slot name="table" /></div></div>' },
          DataTable: { template: '<div data-test="providers-table" />' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
          UpstreamBalanceCharts: {
            props: ['overview', 'loading', 'days'],
            template: '<div data-test="balance-charts">{{ days }}-{{ Boolean(overview) }}-{{ loading }}</div>',
          },
        },
      },
    })

    await flushPromises()

    const tableArea = wrapper.find('[data-test="table"]')
    expect(tableArea.find('[data-test="providers-table"]').exists()).toBe(true)
    expect(tableArea.find('[data-test="balance-charts"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="table-layout"] [data-test="balance-charts"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="provider-balance-charts-section"] [data-test="balance-charts"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="provider-balance-charts-section"]').text()).toContain('30-true-false')
    expect(wrapper.find('[data-test="filters"] [data-test="balance-charts"]').exists()).toBe(false)

    expect(upstreamProvidersSource).toContain('.upstream-providers-page :deep(.table-page-layout)')
    expect(upstreamProvidersSource).toContain('height: auto;')
    expect(upstreamProvidersSource).toContain('.upstream-providers-page :deep(.layout-section-scrollable)')
    expect(upstreamProvidersSource).toContain('overflow: visible;')
  })

  it('supports table controls, row expansion, and secondary column visibility', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'newapi',
        slug: 'new-main',
        name: 'New Main',
        enabled: true,
        is_default: false,
        base_url: 'https://new.example.com',
        login_url: '/api/user/login',
        api_keys_url: '/api/token/',
        groups_url: '/api/group/',
        username: 'admin@example.com',
        password_configured: true,
        account_name_prefix: 'np-',
        temp_disable_minutes: 15,
        account_rate_multiplier_scale: 1.25,
      },
    ])

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns', 'data'],
            setup(props, { slots }) {
              return () => h('div', [
                h('div', { class: 'columns' }, props.columns.map((column: any) => column.key).join(',')),
                ...props.data.flatMap((row: any) => [
                  h('div', { class: 'homepage-cell' }, slots['cell-homepage']?.({ row })),
                  h('div', { class: 'details-cell' }, slots['row-detail']?.({ row, colspan: props.columns.length })),
                ]),
              ])
            },
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.columns').text()).toBe('homepage,name,enabled,sort_order,prefix,rate_scale,balance,today_consumption,actions')
    expect(wrapper.find('.details-cell').text()).toBe('')

    const expandButton = wrapper.find('.expand-toggle')
    expect(expandButton.exists()).toBe(true)
    await expandButton.trigger('click')

    expect(wrapper.find('.details-cell').text()).toContain('https://new.example.com')
    expect(wrapper.find('.details-cell').text()).toContain('admin@example.com')
    expect(wrapper.find('.details-cell').text()).toContain('/api/token/')
    expect(wrapper.find('.details-cell').text()).toContain('/api/user/login')
    expect(wrapper.find('.details-cell').text()).toContain('/api/group/')
  })

  it('opens balance maintenance from provider balance column', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'sub-main',
        name: 'Sub Main',
        enabled: true,
        is_default: true,
        base_url: 'https://upstream.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        account_rate_multiplier_scale: 1,
      },
    ])
    adminAPIMock.upstreamAccountSync.getBalanceConsumption.mockResolvedValueOnce({
      config: {
        enabled: true,
        interval_seconds: 600,
        provider_amount_scales: { 'sub-main': 1.2 },
      },
      summaries: {
        'sub-main': {
          provider_slug: 'sub-main',
          provider_name: 'Sub Main',
          current_balance: 80,
          today_consumption: 24.5,
          amount_scale: 1.2,
          complete: true,
          anomaly: false,
          snapshot_count: 2,
          last_snapshot_at: '2026-06-15T12:00:00Z',
        },
      },
      rows: [
        {
          provider_slug: 'sub-main',
          provider_name: 'Sub Main',
          date: '2026-06-15',
          amount_scale: 1.2,
          opening_balance: 100,
          closing_balance: 80,
          current_balance: 80,
          recharge_amount: 4.5,
          consumption_amount: 24.5,
          snapshot_count: 2,
          complete: true,
          anomaly: false,
        },
      ],
      snapshots: [
        {
          id: 1,
          provider_slug: 'sub-main',
          provider_name: 'Sub Main',
          provider_type: 'sub2api',
          balance: 100,
          amount_scale: 1.2,
          status: 'success',
          captured_at: '2026-06-15T01:00:00Z',
          created_at: '2026-06-15T01:00:00Z',
        },
        {
          id: 2,
          provider_slug: 'sub-main',
          provider_name: 'Sub Main',
          provider_type: 'sub2api',
          balance: 90,
          amount_scale: 1.2,
          status: 'success',
          captured_at: '2026-06-15T08:00:00Z',
          created_at: '2026-06-15T08:00:00Z',
        },
        {
          id: 3,
          provider_slug: 'sub-main',
          provider_name: 'Sub Main',
          provider_type: 'sub2api',
          balance: 80,
          amount_scale: 1.2,
          status: 'success',
          captured_at: '2026-06-15T12:00:00Z',
          created_at: '2026-06-15T12:00:00Z',
        },
      ],
    })

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.flatMap((row: any) => [
                h('div', { class: 'balance-cell' }, slots['cell-balance']?.({ row })),
                h('div', { class: 'today-cost-cell' }, slots['cell-today_consumption']?.({ row })),
              ]))
            },
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show" class="dialog"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.today-cost-cell').text()).toContain('24.5000')
    expect(wrapper.find('.numeric-cost').exists()).toBe(true)

    const detailButton = wrapper
      .findAll('button')
      .find((button) => button.attributes('title') === 'common.more')
    expect(detailButton).toBeTruthy()

    await detailButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('.balance-dialog').text()).toContain('admin.upstreamProviders.balanceDialogDescription')
    expect(wrapper.find('.balance-dialog').text()).toContain('2026-06-15')
    expect(wrapper.find('.balance-dialog').text()).toContain('24.5000')
    expect(wrapper.find('.balance-dialog').text()).toContain('admin.upstreamProviders.balanceSamples')
    expect(wrapper.find('.balance-dialog').text()).not.toContain('admin.upstreamProviders.balanceSamplerAutoRun')
    expect(wrapper.find('.balance-dialog').text()).not.toContain('admin.upstreamProviders.balanceSamplerIntervalSeconds')
    expect(wrapper.find('.balance-dialog').text()).not.toContain('admin.upstreamProviders.balanceSampleNow')
    expect(wrapper.findAll('.snapshot-row')).toHaveLength(3)
    expect(wrapper.find('.balance-dialog').text()).toContain('100.0000')
    expect(wrapper.find('.balance-dialog').text()).toContain('90.0000')
    expect(wrapper.find('.balance-dialog').text()).toContain('80.0000')
  })

  it('runs balance sampling from the main toolbar and renders refined filter selects', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'sub-main',
        name: 'Sub Main',
        enabled: true,
        is_default: true,
        base_url: 'https://upstream.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        account_rate_multiplier_scale: 1,
      },
    ])
    adminAPIMock.upstreamAccountSync.getBalanceConsumption
      .mockResolvedValueOnce({
        config: {
          enabled: true,
          interval_seconds: 3600,
          provider_amount_scales: {},
        },
        summaries: {},
        rows: [],
        snapshots: [],
      })
      .mockResolvedValueOnce({
        config: {
          enabled: true,
          interval_seconds: 3600,
          provider_amount_scales: {},
        },
        summaries: {
          'sub-main': {
            provider_slug: 'sub-main',
            provider_name: 'Sub Main',
            current_balance: 80,
            today_consumption: 12.25,
            amount_scale: 1,
            complete: true,
            anomaly: false,
            snapshot_count: 1,
          },
        },
        rows: [],
        snapshots: [],
      })

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.findAll('.upstream-filter-select')).toHaveLength(2)
    expect(wrapper.findAllComponents({ name: 'Select' })).toHaveLength(2)
    const sampleButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.balanceSampleNow'))
    expect(sampleButton).toBeTruthy()

    await sampleButton!.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamAccountSync.runBalanceSampleNow).toHaveBeenCalledTimes(1)
    expect(adminAPIMock.upstreamAccountSync.getBalanceConsumption).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('12.25')
  })

  it('opens global balance sampler settings from the main toolbar and saves provider scales', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'sub-main',
        name: 'Sub Main',
        enabled: true,
        is_default: true,
        base_url: 'https://upstream.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        account_rate_multiplier_scale: 1,
      },
      {
        type: 'newapi',
        slug: 'new-main',
        name: 'New Main',
        enabled: true,
        is_default: false,
        base_url: 'https://new.example.com',
        login_url: '/api/user/login',
        api_keys_url: '/api/token/',
        account_rate_multiplier_scale: 1,
      },
    ])
    adminAPIMock.upstreamAccountSync.getBalanceConsumption.mockResolvedValue({
      config: {
        enabled: true,
        interval_seconds: 900,
        provider_amount_scales: { 'sub-main': 1.2 },
      },
      summaries: {},
      rows: [],
      snapshots: [],
    })
    adminAPIMock.upstreamAccountSync.updateBalanceSamplerConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 1200,
      provider_amount_scales: {
        'sub-main': 1.5,
        'new-main': 1,
      },
    })

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show" class="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    const settingsButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.balanceSamplerSettings'))
    expect(settingsButton).toBeTruthy()

    await settingsButton!.trigger('click')
    await flushPromises()

    const dialog = wrapper.find('.balance-sampler-dialog')
    expect(dialog.exists()).toBe(true)
    expect(dialog.text()).toContain('admin.upstreamProviders.balanceSamplerAutoRun')
    expect(dialog.text()).toContain('admin.upstreamProviders.balanceSamplerIntervalSeconds')
    expect(dialog.text()).toContain('Sub Main')
    expect(dialog.text()).toContain('New Main')

    const autoRun = dialog.find('input[type="checkbox"]')
    expect((autoRun.element as HTMLInputElement).checked).toBe(true)
    await autoRun.setValue(false)

    const interval = dialog.find('input[data-test="balance-sampler-interval"]')
    await interval.setValue('1200')

    const subMainScale = dialog.find('input[data-test="balance-sampler-scale-sub-main"]')
    await subMainScale.setValue('1.5')

    const saveButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.save'))
    expect(saveButton).toBeTruthy()

    await saveButton!.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamAccountSync.updateBalanceSamplerConfig).toHaveBeenCalledWith({
      enabled: false,
      interval_seconds: 1200,
      provider_amount_scales: {
        'sub-main': 1.5,
        'new-main': 1,
      },
    })
  })

  it('opens health guard settings from the main toolbar and saves platform rules', async () => {
    adminAPIMock.upstreamAccountSync.getHealthGuardConfig.mockResolvedValueOnce({
      enabled: true,
      interval_seconds: 1800,
      max_accounts_per_run: 120,
      concurrency: 2,
      timeout_per_account_seconds: 60,
      failure_threshold: 3,
      slow_threshold: 2,
      recovery_threshold: 2,
      healthy_latency_ms: 12000,
      ignored_account_ids: [7, 9],
      account_models: {},
      platform_models: { anthropic: 'claude-3-5-haiku-latest' },
      platform_latency_ms: { anthropic: 16000 },
    })
    adminAPIMock.accounts.list.mockResolvedValueOnce({
      items: [
        {
          id: 7,
          name: 'unstable-account',
          platform: 'openai',
          type: 'apikey',
          status: 'error',
          schedulable: false,
        },
        {
          id: 9,
          name: 'orphan-account',
          platform: 'openai',
          type: 'apikey',
          status: 'inactive',
          schedulable: false,
        },
        {
          id: 11,
          name: 'manual-ignore',
          platform: 'anthropic',
          type: 'oauth',
          status: 'active',
          schedulable: true,
        },
      ],
      total: 3,
      page: 1,
      page_size: 200,
      pages: 1,
    })
    adminAPIMock.accounts.getById.mockImplementation(async (id: number) => {
      if (id === 7) {
        return {
          id,
          name: 'unstable-account',
          platform: 'openai',
          type: 'apikey',
          status: 'error',
          schedulable: false,
        }
      }
      return {
        id,
        name: 'orphan-account',
        platform: 'openai',
        type: 'apikey',
        status: 'inactive',
        schedulable: false,
      }
    })
    adminAPIMock.upstreamAccountSync.getHealthGuardRecords.mockResolvedValueOnce([
      {
        id: 'run-1',
        trigger: 'manual',
        status: 'success',
        started_at: '2026-07-05T00:00:00Z',
        finished_at: '2026-07-05T00:00:01Z',
        summary: {
          total_accounts: 5,
          checked_count: 1,
          healthy_count: 0,
          slow_count: 0,
          failed_count: 1,
          skipped_count: 4,
          disabled_count: 1,
          recovered_count: 0,
          unchanged_count: 0,
          skip_reasons: [
            {
              reason: 'missing_provider_slug',
              count: 4,
              sample_accounts: [
                { account_id: 9, account_name: 'orphan-account', platform: 'openai' },
              ],
            },
          ],
        },
        items: [
          {
            account_id: 7,
            account_name: 'unstable-account',
            platform: 'openai',
            provider_slug: 'sub-main',
            provider_name: 'Sub Main',
            model_id: 'gpt-4o-mini',
            schedulable_before: true,
            schedulable_after: false,
            status: 'failed',
            test_status: 'failed',
            latency_ms: 0,
            latency_limit_ms: 12000,
            consecutive_failed: 3,
            consecutive_slow: 0,
            consecutive_healthy: 0,
            action: 'disabled',
            reason: 'failure threshold reached',
            error_message: '401',
            started_at: '2026-07-05T00:00:00Z',
            finished_at: '2026-07-05T00:00:01Z',
          },
        ],
      },
    ])
    adminAPIMock.upstreamAccountSync.updateHealthGuardConfig.mockResolvedValueOnce({
      enabled: false,
      interval_seconds: 2400,
      max_accounts_per_run: 120,
      concurrency: 2,
      timeout_per_account_seconds: 60,
      failure_threshold: 3,
      slow_threshold: 2,
      recovery_threshold: 2,
      healthy_latency_ms: 12000,
      ignored_account_ids: [9, 11],
      account_models: { '11': 'claude-opus-test' },
      platform_models: { anthropic: 'claude-3-5-haiku-latest' },
      platform_latency_ms: { anthropic: 16000 },
    })

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show" class="dialog"><h2>{{ title }}</h2><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
          Select: SelectStub,
        },
      },
    })

    await flushPromises()

    const settingsButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.healthGuardSettings'))
    expect(settingsButton).toBeTruthy()

    await settingsButton!.trigger('click')
    await flushPromises()

    const dialog = wrapper.find('.health-guard-dialog')
    expect(dialog.exists()).toBe(true)
    expect(dialog.text()).toContain('admin.upstreamProviders.healthGuardAutoRun')
    expect(dialog.text()).toContain('Anthropic')
    expect(dialog.text()).toContain('admin.upstreamProviders.healthGuardIgnoredSummary')
    expect(dialog.text()).toContain('admin.upstreamProviders.healthGuardAdjustmentLogs')
    expect(dialog.text()).toContain('admin.upstreamProviders.healthGuardSkipReasons')
    expect(dialog.text()).toContain('admin.upstreamProviders.healthGuardResultList')

    const configToggle = wrapper.find('[data-test="health-guard-config-toggle"]')
    expect(configToggle.exists()).toBe(true)
    await configToggle.trigger('click')
    await flushPromises()

    const adjustmentButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.healthGuardAdjustmentLogs'))
    expect(adjustmentButton).toBeTruthy()
    await adjustmentButton!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('unstable-account')
    expect(wrapper.text()).toContain('admin.upstreamProviders.healthGuardActionDisabled')

    const skipReasonButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.healthGuardSkipReasons'))
    expect(skipReasonButton).toBeTruthy()
    await skipReasonButton!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('admin.upstreamProviders.healthGuardSkipReasonMissingProviderSlug')
    expect(wrapper.text()).toContain('admin.upstreamProviders.healthGuardSkipSampleAccounts')

    const autoRun = dialog.find('input[type="checkbox"]')
    await autoRun.setValue(false)
    const interval = dialog.findAll('input[type="number"]')[0]
    await interval.setValue('2400')
    await wrapper.find('.health-guard-account-model-select').setValue('11')
    await wrapper.find('.health-guard-account-model-input').setValue('claude-opus-test')
    await wrapper.find('[data-test="health-guard-account-model-add"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="health-guard-ignored-manage"]').trigger('click')
    await flushPromises()
    await flushPromises()
    expect(adminAPIMock.accounts.list).toHaveBeenCalledWith(1, 200, {
      lite: 'true',
      sort_by: 'name',
      sort_order: 'asc',
    })
    expect(wrapper.text()).toContain('unstable-account')
    expect(wrapper.text()).toContain('orphan-account')
    expect(wrapper.text()).toContain('manual-ignore')
    await wrapper.find('.health-guard-ignored-select').setValue('11')
    await flushPromises()
    await wrapper.find('[data-test="health-guard-ignored-add"]').trigger('click')
    await flushPromises()
    await wrapper.find('[data-test="health-guard-ignored-remove-7"]').trigger('click')
    await flushPromises()

    const saveButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.save'))
    expect(saveButton).toBeTruthy()

    await saveButton!.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamAccountSync.updateHealthGuardConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: false,
        interval_seconds: 2400,
        ignored_account_ids: [9, 11],
        account_models: { '11': 'claude-opus-test' },
        platform_models: { anthropic: 'claude-3-5-haiku-latest' },
        platform_latency_ms: { anthropic: 16000 },
      })
    )
  })

  it('offers existing URLs as datalist choices in provider form', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'sub-main',
        name: 'Sub Main',
        enabled: true,
        base_url: 'https://sub.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        available_groups_url: '/api/v1/groups/available?timezone=Asia%2FShanghai',
        account_rate_multiplier_scale: 1,
      },
      {
        type: 'newapi',
        slug: 'new-main',
        name: 'New Main',
        enabled: true,
        base_url: 'https://new.example.com',
        login_url: '/api/user/login',
        api_keys_url: '/api/token/',
        groups_url: '/api/group/',
        account_rate_multiplier_scale: 1,
      },
    ])

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()
    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.createProvider'))
    await createButton!.trigger('click')

    expect(wrapper.find('option[value="https://sub.example.com"]').exists()).toBe(true)
    expect(wrapper.find('option[value="/api/admin/keys"]').exists()).toBe(true)
    expect(wrapper.find('option[value="/api/v1/auth/login"]').exists()).toBe(true)
  })

  it('edits provider balance URL like other upstream endpoints', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'sub2api',
        slug: 'sub-main',
        name: 'Sub Main',
        enabled: true,
        base_url: 'https://sub.example.com',
        login_url: '/api/v1/auth/login',
        api_keys_url: '/api/admin/keys',
        available_groups_url: '/api/v1/groups/available?timezone=Asia%2FShanghai',
        balance_url: '/api/v1/auth/me?timezone=Asia%2FShanghai',
        account_rate_multiplier_scale: 1,
      },
    ])

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', { class: 'actions-cell' }, slots['cell-actions']?.({ row }))))
            },
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    const editButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')

    const balanceInput = wrapper.find('input[list="upstream-provider-balance-url-options"]')
    expect(balanceInput.exists()).toBe(true)
    expect((balanceInput.element as HTMLInputElement).value).toBe('/api/v1/auth/me?timezone=Asia%2FShanghai')

    await balanceInput.setValue('/api/custom/balance')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(adminAPIMock.upstreamProviders.update).toHaveBeenCalledWith(
      'sub-main',
      expect.objectContaining({
        balance_url: '/api/custom/balance',
      })
    )
  })

  it('edits provider usage cost URL like other upstream endpoints', async () => {
    adminAPIMock.upstreamProviders.list.mockResolvedValue([
      {
        type: 'newapi',
        slug: 'new-main',
        name: 'New Main',
        enabled: true,
        base_url: 'https://new.example.com',
        login_url: '/api/user/login',
        api_keys_url: '/api/token/',
        groups_url: '/api/group/',
        balance_url: '/api/user/self',
        usage_cost_url: '/api/log/self/stat?type=0&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}',
        username: 'root',
        password_configured: true,
        account_rate_multiplier_scale: 1,
      },
    ])

    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', { class: 'actions-cell' }, slots['cell-actions']?.({ row }))))
            },
          },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    const editButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')

    const costInput = wrapper.find('input[list="upstream-provider-usage-cost-url-options"]')
    expect(costInput.exists()).toBe(true)
    expect((costInput.element as HTMLInputElement).value).toContain('/api/log/self/stat')

    await costInput.setValue('/api/custom/cost?start_timestamp={start_timestamp}&end_timestamp={end_timestamp}')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(adminAPIMock.upstreamProviders.update).toHaveBeenCalledWith(
      'new-main',
      expect.objectContaining({
        usage_cost_url: '/api/custom/cost?start_timestamp={start_timestamp}&end_timestamp={end_timestamp}',
      })
    )
  })
})
