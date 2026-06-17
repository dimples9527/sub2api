import { flushPromises, mount } from '@vue/test-utils'
import { h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UpstreamProvidersView from './UpstreamProvidersView.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const { adminAPIMock } = vi.hoisted(() => ({
  adminAPIMock: {
    upstreamProviders: {
      list: vi.fn().mockResolvedValue([]),
      getBalance: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      testConfig: vi.fn(),
    },
    upstreamAccountSync: {
      getBalanceConsumption: vi.fn(),
      updateBalanceSamplerConfig: vi.fn(),
      addBalanceRecharge: vi.fn(),
      runBalanceSampleNow: vi.fn(),
    },
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: adminAPIMock,
}))

vi.mock('@/api/admin/index', () => ({
  adminAPI: {
    upstreamProviders: adminAPIMock.upstreamProviders,
    upstreamAccountSync: adminAPIMock.upstreamAccountSync,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

describe('UpstreamProvidersView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
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

    expect(wrapper.find('.columns').text()).toBe('homepage,name,interface,prefix,rate_scale,temp_disable_minutes,balance,today_consumption,actions')
    expect(wrapper.find('.provider-name-card').exists()).toBe(true)
    expect(wrapper.find('.provider-type-tag').exists()).toBe(true)
    expect(wrapper.find('.provider-enabled-tag').exists()).toBe(true)

    const homepage = wrapper.find('a[href="https://upstream.example.com"]')
    expect(homepage.exists()).toBe(true)
    expect(homepage.attributes('target')).toBe('_blank')

    const balanceButton = wrapper
      .findAll('button')
      .find((button) => button.attributes('title') === 'admin.upstreamProviders.fetchBalance')
    expect(balanceButton).toBeTruthy()

    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledWith('sub-main')
    expect(wrapper.text()).toContain('9.5000')
    expect(wrapper.find('.today-cost-cell').text()).toContain('1.7500')
    expect(wrapper.find('.numeric-alert').exists()).toBe(true)

    await balanceButton!.trigger('click')
    await flushPromises()
    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledTimes(2)
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
                  h('div', { class: 'interface-cell' }, slots['cell-interface']?.({ row })),
                  h('div', { class: 'temp-cell' }, slots['cell-temp_disable_minutes']?.({ row })),
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

    expect(wrapper.find('.columns').text()).toBe('homepage,name,interface,prefix,rate_scale,temp_disable_minutes,balance,today_consumption,actions')
    expect(wrapper.find('.details-cell').text()).toBe('')

    const expandButton = wrapper.find('.expand-toggle')
    expect(expandButton.exists()).toBe(true)
    await expandButton.trigger('click')

    expect(wrapper.find('.details-cell').text()).toContain('https://new.example.com')
    expect(wrapper.find('.details-cell').text()).toContain('admin@example.com')
    expect(wrapper.find('.details-cell').text()).toContain('/api/token/')
    expect(wrapper.find('.details-cell').text()).toContain('/api/user/login')
    expect(wrapper.find('.details-cell').text()).toContain('/api/group/')

    expect(wrapper.find('.temp-cell').text()).toContain('15分钟')
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
    expect(wrapper.findAll('.snapshot-row')).toHaveLength(3)
    expect(wrapper.find('.balance-dialog').text()).toContain('100.0000')
    expect(wrapper.find('.balance-dialog').text()).toContain('90.0000')
    expect(wrapper.find('.balance-dialog').text()).toContain('80.0000')
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
