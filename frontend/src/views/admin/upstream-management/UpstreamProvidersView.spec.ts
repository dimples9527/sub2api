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
                h('div', { class: 'balance-cell' }, slots['cell-balance_consumption']?.({ row })),
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

    expect(wrapper.find('.columns').text()).toBe('homepage,name,enabled,base_url,auth,endpoints,policy,balance_consumption,actions')
    expect(wrapper.find('.provider-name-card').exists()).toBe(true)
    expect(wrapper.find('.provider-slug-tag').exists()).toBe(true)

    const homepage = wrapper.find('a[href="https://upstream.example.com"]')
    expect(homepage.exists()).toBe(true)
    expect(homepage.attributes('target')).toBe('_blank')

    const balanceButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.balanceShort'))
    expect(balanceButton).toBeTruthy()

    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledWith('sub-main')
    expect(wrapper.text()).toContain('334.740794')
    expect(wrapper.text()).toContain('admin.upstreamProviders.balanceIncomplete')

    await balanceButton!.trigger('click')
    await flushPromises()
    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledTimes(2)
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
              return () => h('div', props.data.map((row: any) => (
                h('div', { class: 'balance-cell' }, slots['cell-balance_consumption']?.({ row }))
              )))
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

    expect(wrapper.text()).toContain('24.50')
    expect(wrapper.text()).toContain('admin.upstreamProviders.balanceComplete')

    const detailButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('common.more'))
    expect(detailButton).toBeTruthy()

    await detailButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('.balance-dialog').text()).toContain('admin.upstreamProviders.balanceDialogDescription')
    expect(wrapper.find('.balance-dialog').text()).toContain('2026-06-15')
    expect(wrapper.find('.balance-dialog').text()).toContain('24.50')
    expect(wrapper.find('.balance-dialog').text()).toContain('admin.upstreamProviders.balanceSamples')
    expect(wrapper.findAll('.snapshot-row')).toHaveLength(3)
    expect(wrapper.find('.balance-dialog').text()).toContain('100.00')
    expect(wrapper.find('.balance-dialog').text()).toContain('90.00')
    expect(wrapper.find('.balance-dialog').text()).toContain('80.00')
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
})
