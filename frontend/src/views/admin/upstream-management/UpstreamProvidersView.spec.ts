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
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: adminAPIMock,
}))

vi.mock('@/api/admin/index', () => ({
  adminAPI: {
    upstreamProviders: adminAPIMock.upstreamProviders,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
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

  it('renders a homepage link and automatically fetches provider balance', async () => {
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
                h('div', { class: 'base-url-cell' }, slots['cell-base_url']?.({ row, value: row.base_url })),
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

    const homepage = wrapper.find('a[href="https://upstream.example.com"]')
    expect(homepage.exists()).toBe(true)
    expect(homepage.attributes('target')).toBe('_blank')

    const balanceButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.balanceShort'))
    expect(balanceButton).toBeTruthy()

    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledWith('sub-main')
    expect(wrapper.text()).toContain('334.740794')

    await balanceButton!.trigger('click')
    await flushPromises()
    expect(adminAPIMock.upstreamProviders.getBalance).toHaveBeenCalledTimes(2)
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
