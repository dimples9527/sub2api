import { flushPromises, mount } from '@vue/test-utils'
import { h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import upstreamGroupsSource from './UpstreamGroupsView.vue?raw'
import UpstreamGroupsView from './UpstreamGroupsView.vue'

const { adminAPIMock, appStoreMock } = vi.hoisted(() => ({
  adminAPIMock: {
    upstreamManagement: {
      getGroups: vi.fn(),
      getRateFixConfig: vi.fn(),
      createLocalGroupFromUpstream: vi.fn(),
    },
    groups: {
      getUpstreamMonitorStatus: vi.fn(),
      update: vi.fn(),
      getAll: vi.fn(),
    },
    proxies: {
      getAll: vi.fn(),
    },
    accounts: {
      list: vi.fn(),
      update: vi.fn(),
      getById: vi.fn(),
    },
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: adminAPIMock,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

vi.mock('@/utils/upstreamMonitorTrend', () => ({
  buildUpstreamMonitorTrendIndex: () => new Map(),
  normalizeUpstreamMonitorGroupKey: (value: string) => value,
}))

describe('UpstreamGroupsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          upstream_group_key: 'vip',
          upstream_rate: 2.5,
          upstream_key_count: 3,
          matched: false,
          needs_rate_increase: false,
        },
      ],
      warnings: [],
      records: [],
    })
    adminAPIMock.upstreamManagement.getRateFixConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
    })
    adminAPIMock.upstreamManagement.createLocalGroupFromUpstream.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [],
      warnings: [],
      records: [],
    })
    adminAPIMock.groups.getUpstreamMonitorStatus.mockResolvedValue({ rows: [] })
    adminAPIMock.groups.update.mockResolvedValue({})
    adminAPIMock.groups.getAll.mockResolvedValue([
      { id: 7, name: 'VIP', platform: 'openai', rate_multiplier: 2.5, status: 'active' },
    ])
    adminAPIMock.proxies.getAll.mockResolvedValue([])
    adminAPIMock.accounts.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 100, pages: 0 })
    adminAPIMock.accounts.update.mockResolvedValue({})
    adminAPIMock.accounts.getById.mockResolvedValue({
      id: 12,
      name: 'local-a',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [7],
      groups: [],
    })
  })

  it('uses fixed upstream group table column classes for stable headers and wrapping content', async () => {
    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns'],
            setup(props) {
              return () => h('div', { class: 'columns' }, props.columns.map((column: any) => `${column.key}:${column.class || ''}`).join('|'))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: true,
        },
      },
    })

    await flushPromises()

    const classes = wrapper.find('.columns').text()
    expect(classes).toContain('upstream_group_name:ug-table-upstream-group-column')
    expect(classes).toContain('local_group_name:ug-table-local-group-column')
    expect(classes).toContain('monitor_trend:ug-table-monitor-column')
    expect(classes).toContain('action:ug-table-action-column')
  })

  it('lets the fixed upstream group table fill wide containers', () => {
    expect(upstreamGroupsSource).toContain('table-layout: fixed;')
    expect(upstreamGroupsSource).toContain('width: max(100%, 1560px);')
    expect(upstreamGroupsSource).not.toMatch(/^\s+width:\s*1560px;$/m)
  })

  it('opens create account modal from upstream group toolbar and refreshes after create', async () => {
    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: true,
          CreateAccountModal: {
            props: ['show', 'proxies', 'groups'],
            emits: ['created', 'close'],
            setup(props, { emit }) {
              return () => props.show
                ? h('button', { class: 'create-account-modal', onClick: () => emit('created') }, 'create')
                : null
            },
          },
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.accounts.createAccount'))?.trigger('click')
    await flushPromises()

    expect(adminAPIMock.proxies.getAll).toHaveBeenCalled()
    expect(adminAPIMock.groups.getAll).toHaveBeenCalled()
    expect(wrapper.find('.create-account-modal').exists()).toBe(true)

    await wrapper.find('.create-account-modal').trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamManagement.getGroups).toHaveBeenCalledTimes(2)
  })

  it('loads and renders bound accounts, account status, and account actions for matched groups', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          upstream_group_key: 'vip',
          upstream_rate: 2.5,
          upstream_key_count: 3,
          local_group_id: 7,
          local_group_name: 'VIP local',
          local_group_platform: 'openai',
          local_rate: 2.5,
          matched: true,
          needs_rate_increase: false,
        },
      ],
      warnings: [],
      records: [],
    })
    adminAPIMock.accounts.list.mockResolvedValueOnce({
      items: [
        {
          id: 12,
          name: 'local-a',
          platform: 'openai',
          type: 'apikey',
          status: 'active',
          schedulable: true,
          group_ids: [7],
          groups: [],
        },
      ],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1,
    })

    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data', 'columns'],
            setup(props, { slots }) {
              return () => h('div', [
                h('div', { class: 'columns' }, props.columns.map((column: any) => column.key).join(',')),
                ...props.data.map((row: any) => h('div', [
                  slots['cell-bound_accounts']?.({ row }),
                  slots['cell-account_status']?.({ row }),
                  slots['cell-action']?.({ row }),
                ])),
              ])
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: true,
          AccountStatusIndicator: {
            props: ['account'],
            template: '<span class="account-status-indicator">{{ account.status }}</span>',
          },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(adminAPIMock.accounts.list).toHaveBeenCalledWith(1, 100, { group: '7', sort_by: 'id', sort_order: 'asc' })
    expect(wrapper.find('.columns').text()).toContain('bound_accounts')
    expect(wrapper.find('.columns').text()).toContain('account_status')
    expect(wrapper.text()).toContain('local-a')
    expect(wrapper.text()).toContain('active')
    expect(wrapper.text()).toContain('admin.upstreamGroups.editAccount')
    expect(wrapper.text()).toContain('admin.upstreamGroups.editAccountBinding')
  })

  it('edits bound account groups from the upstream group action column', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          upstream_group_key: 'vip',
          upstream_rate: 2.5,
          upstream_key_count: 3,
          local_group_id: 7,
          local_group_name: 'VIP local',
          local_group_platform: 'openai',
          local_rate: 2.5,
          matched: true,
          needs_rate_increase: false,
        },
      ],
      warnings: [],
      records: [],
    })
    adminAPIMock.accounts.list
      .mockResolvedValueOnce({
        items: [
          {
            id: 12,
            name: 'local-a',
            platform: 'openai',
            type: 'apikey',
            status: 'active',
            schedulable: true,
            group_ids: [7],
            groups: [],
          },
        ],
        total: 1,
        page: 1,
        page_size: 100,
        pages: 1,
      })
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        page_size: 100,
        pages: 0,
      })
    adminAPIMock.accounts.update.mockResolvedValue({
      id: 12,
      name: 'local-a',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [7],
      groups: [],
    })

    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                slots['cell-action']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: true,
          GroupSelector: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            setup(props, { emit }) {
              return () => h('button', {
                class: 'group-selector',
                onClick: () => emit('update:modelValue', [...props.modelValue, 8]),
              }, `groups:${props.modelValue.join(',')}`)
            },
          },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.editAccountBinding'))?.trigger('click')
    await flushPromises()

    expect(wrapper.find('.group-selector').exists()).toBe(true)
    await wrapper.find('.group-selector').trigger('click')
    const saveButtons = wrapper.findAll('button').filter(button => button.text().includes('common.save'))
    await saveButtons[saveButtons.length - 1].trigger('click')
    await flushPromises()

    expect(adminAPIMock.accounts.update).toHaveBeenCalledWith(12, { group_ids: [7, 8] })
    expect(adminAPIMock.upstreamManagement.getGroups).toHaveBeenCalledTimes(2)
  })

  it('requires selecting a platform when syncing a local group from upstream', async () => {
    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                slots['cell-action']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            setup(props, { emit }) {
              return () => h('select', {
                value: props.modelValue,
                onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
              }, (props.options || []).map((option: any) => h('option', { value: option.value }, option.label)))
            },
          },
        },
      },
    })

    await flushPromises()

    const syncButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.syncLocalGroup'))
    expect(syncButton).toBeTruthy()
    await syncButton!.trigger('click')
    await flushPromises()

    const selects = wrapper.findAll('select')
    const platformSelect = selects.at(selects.length - 1)
    expect(platformSelect).toBeTruthy()
    expect((platformSelect!.element as HTMLSelectElement).value).toBe('')

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.confirmSync'))!.trigger('click')
    expect(appStoreMock.showError).toHaveBeenCalledWith('admin.upstreamGroups.invalidPlatform')
    expect(adminAPIMock.upstreamManagement.createLocalGroupFromUpstream).not.toHaveBeenCalled()

    await platformSelect!.setValue('gemini')
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.confirmSync'))!.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamManagement.createLocalGroupFromUpstream).toHaveBeenCalledWith({
      upstream_group_name: 'VIP',
      platform: 'gemini',
      rate_multiplier: 2.5,
    })
  })

  it('updates matched local group rate from the action column', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          upstream_group_key: 'vip',
          upstream_rate: 2.5,
          upstream_key_count: 3,
          local_group_id: 42,
          local_group_name: 'VIP local',
          local_rate: 1.5,
          matched: true,
          needs_rate_increase: true,
        },
      ],
      warnings: [],
      records: [],
    })

    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                slots['cell-action']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: true,
        },
      },
    })

    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.editLocalRate'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const input = wrapper.find('#local-rate-multiplier')
    expect((input.element as HTMLInputElement).value).toBe('1.5')
    await input.setValue('2.5')
    const saveButtons = wrapper.findAll('button').filter(button => button.text().includes('common.save'))
    await saveButtons[saveButtons.length - 1].trigger('click')
    await flushPromises()

    expect(adminAPIMock.groups.update).toHaveBeenCalledWith(42, { rate_multiplier: 2.5 })
  })
})
