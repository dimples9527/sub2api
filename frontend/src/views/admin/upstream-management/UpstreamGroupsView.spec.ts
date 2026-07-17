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
      applyRateFixes: vi.fn(),
      createLocalGroupFromUpstream: vi.fn(),
      saveGroupMapping: vi.fn(),
      markRateFixRecordHandled: vi.fn(),
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
    adminAPIMock.upstreamManagement.applyRateFixes.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [],
      warnings: [],
      records: [],
    })
    adminAPIMock.upstreamManagement.createLocalGroupFromUpstream.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [],
      warnings: [],
      records: [],
    })
    adminAPIMock.upstreamManagement.saveGroupMapping.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [],
      warnings: [],
      records: [],
    })
    adminAPIMock.upstreamManagement.markRateFixRecordHandled.mockResolvedValue([])
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

  it('reports a monitor request failure once instead of repeating it in every trend cell', async () => {
    adminAPIMock.groups.getUpstreamMonitorStatus.mockRejectedValueOnce({
      message: 'Request failed with status code 502',
      error: 'monitor upstream request failed',
    })

    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => slots['cell-monitor_trend']?.({ row })))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: {
            props: ['error'],
            template: '<div class="trend-error-prop">{{ error }}</div>',
          },
          Select: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.trend-error-prop').text()).toBe('')
    expect(appStoreMock.showError).toHaveBeenCalledWith('monitor upstream request failed')
  })

  it('lets the fixed upstream group table fill wide containers', () => {
    expect(upstreamGroupsSource).toContain('table-layout: fixed;')
    expect(upstreamGroupsSource).toContain('width: max(100%, 1560px);')
    expect(upstreamGroupsSource).not.toMatch(/^\s+width:\s*1560px;$/m)
    expect(upstreamGroupsSource).not.toContain('class="ug-records-card"')
    expect(upstreamGroupsSource).toContain('flex: 1 1 auto;')
    expect(upstreamGroupsSource).toMatch(/^\s+height: 80vh;$/m)
    expect(upstreamGroupsSource).toContain('max-height: 80vh;')
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

  it('places dropdown filters before search and filters groups by platform', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'OpenAI VIP',
          upstream_group_key: 'openai-vip',
          upstream_rate: 2.5,
          upstream_key_count: 3,
          local_group_id: 7,
          local_group_name: 'OpenAI local',
          local_group_platform: 'openai',
          local_rate: 2.5,
          matched: true,
          needs_rate_increase: false,
        },
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'Gemini VIP',
          upstream_group_key: 'gemini-vip',
          upstream_rate: 1.5,
          upstream_key_count: 2,
          local_group_id: 8,
          local_group_name: 'Gemini local',
          local_group_platform: 'gemini',
          local_rate: 1.5,
          matched: true,
          needs_rate_increase: false,
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
            setup(props) {
              return () => h('div', props.data.map((row: any) => h('div', { class: 'row-name' }, row.upstream_group_name)))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            template: '<select class="select-stub" :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
          },
        },
      },
    })

    await flushPromises()

    const filterTop = wrapper.find('.ug-filter-top')
    const selectIndex = filterTop.element.innerHTML.indexOf('select-stub')
    const groupSearchIndex = filterTop.element.innerHTML.indexOf('admin.upstreamGroups.searchPlaceholder')
    expect(selectIndex).toBeGreaterThanOrEqual(0)
    expect(groupSearchIndex).toBeGreaterThan(selectIndex)

    expect(wrapper.findAll('.row-name').map(node => node.text())).toEqual(['OpenAI VIP', 'Gemini VIP'])

    await wrapper.findAll('select.select-stub')[0].setValue('gemini')
    await flushPromises()

    expect(wrapper.findAll('.row-name').map(node => node.text())).toEqual(['Gemini VIP'])
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
    expect(wrapper.text()).toContain('admin.upstreamGroups.manageBoundAccounts')

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.manageBoundAccounts'))?.trigger('click')
    await flushPromises()

    expect(wrapper.find('.ug-bound-accounts-dialog').text()).toContain('admin.upstreamGroups.editAccount')
    expect(wrapper.find('.ug-bound-accounts-dialog').text()).toContain('admin.upstreamGroups.editAccountBinding')
  })

  it('opens a full bound account manager from hidden account count and compact action button', async () => {
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
      items: Array.from({ length: 6 }, (_, index) => ({
        id: 12 + index,
        name: `local-${String.fromCharCode(97 + index)}`,
        platform: 'openai',
        type: 'apikey',
        status: index === 5 ? 'disabled' : 'active',
        schedulable: true,
        group_ids: [7],
        groups: [],
      })),
      total: 6,
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
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                h('div', { class: 'bound-cell' }, slots['cell-bound_accounts']?.({ row })),
                h('div', { class: 'status-cell' }, slots['cell-account_status']?.({ row })),
                h('div', { class: 'action-cell' }, slots['cell-action']?.({ row })),
              ])))
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

    expect(wrapper.find('.bound-cell').text()).toContain('+2')
    expect(wrapper.find('.bound-cell').text()).not.toContain('local-f')
    expect(wrapper.findAll('.ug-account-action-row')).toHaveLength(0)
    expect(wrapper.find('.action-cell').text()).toContain('admin.upstreamGroups.manageBoundAccounts')

    const moreButton = wrapper.find('.bound-cell .ug-account-more-button')
    expect(moreButton.element.tagName).toBe('BUTTON')
    await moreButton.trigger('click')
    await flushPromises()

    expect(wrapper.find('.ug-bound-accounts-dialog').exists()).toBe(true)
    expect(wrapper.find('.ug-bound-accounts-dialog').text()).toContain('local-f')
    expect(wrapper.find('.ug-bound-accounts-dialog').text()).toContain('disabled')
    expect(wrapper.find('.ug-bound-accounts-dialog').text()).toContain('admin.upstreamGroups.editAccount')
    expect(wrapper.find('.ug-bound-accounts-dialog').text()).toContain('admin.upstreamGroups.editAccountBinding')
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

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.manageBoundAccounts'))?.trigger('click')
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

  it('unbinds a matched upstream group and stores an ignored mapping', async () => {
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
          local_group_name: 'VIP',
          local_rate: 2.5,
          matched: true,
          match_source: 'name',
          needs_rate_increase: false,
        },
      ],
      warnings: [],
      records: [],
    })
    adminAPIMock.upstreamManagement.saveGroupMapping.mockResolvedValue({
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
          match_ignored: true,
          needs_rate_increase: false,
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
                slots['cell-local_group_name']?.({ row }),
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

    const unbindButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.unbindGroup'))
    expect(unbindButton).toBeTruthy()
    await unbindButton!.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamManagement.saveGroupMapping).toHaveBeenCalledWith('VIP', null, true)
    expect(appStoreMock.showSuccess).toHaveBeenCalledWith('admin.upstreamGroups.unbindGroupSuccess')
    expect(wrapper.text()).toContain('admin.upstreamGroups.matchIgnored')
  })

  it('clears an ignored mapping to rematch an upstream group', async () => {
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
          match_ignored: true,
          needs_rate_increase: false,
        },
      ],
      warnings: [],
      records: [],
    })
    adminAPIMock.upstreamManagement.saveGroupMapping.mockResolvedValue({
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
          local_group_name: 'VIP',
          local_rate: 2.5,
          matched: true,
          match_source: 'name',
          needs_rate_increase: false,
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
                slots['cell-local_group_name']?.({ row }),
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

    expect(wrapper.text()).toContain('admin.upstreamGroups.matchIgnored')
    expect(wrapper.text()).toContain('admin.upstreamGroups.rematchGroup')
    expect(wrapper.text()).not.toContain('admin.upstreamGroups.syncLocalGroup')

    const rematchButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.rematchGroup'))
    expect(rematchButton).toBeTruthy()
    await rematchButton!.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamManagement.saveGroupMapping).toHaveBeenCalledWith('VIP', null)
    expect(appStoreMock.showSuccess).toHaveBeenCalledWith('admin.upstreamGroups.rematchGroupSuccess')
    expect(wrapper.text()).toContain('admin.upstreamGroups.nameMatched')
  })

  it('opens rate fix logs in a dialog and marks pending records handled', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [],
      warnings: [],
      records: [
        {
          group_id: 42,
          group_name: 'VIP local',
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          old_rate: 1.5,
          new_rate: 2.5,
          changed_at: '2026-06-20T00:00:00Z',
          handled: false,
        },
      ],
    })
    adminAPIMock.upstreamManagement.markRateFixRecordHandled.mockResolvedValue([
      {
        group_id: 42,
        group_name: 'VIP local',
        provider_slug: 'default-upstream',
        provider_name: 'Default upstream',
        upstream_group_name: 'VIP',
        old_rate: 1.5,
        new_rate: 2.5,
        changed_at: '2026-06-20T00:00:00Z',
        handled: true,
      },
    ])

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
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.ug-rate-fix-warning').exists()).toBe(true)
    expect(wrapper.find('.ug-rate-fix-warning').text()).toContain('1')
    expect(wrapper.text()).toContain('admin.upstreamGroups.openRateFixLogs')
    expect(wrapper.find('.ug-records-card').exists()).toBe(false)

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.openRateFixLogs'))?.trigger('click')
    await flushPromises()

    expect(wrapper.find('.ug-rate-fix-logs-dialog').exists()).toBe(true)
    expect(wrapper.find('.ug-rate-fix-logs-dialog').text()).toContain('VIP local')
    expect(wrapper.find('.ug-rate-fix-logs-dialog').text()).toContain('admin.upstreamGroups.rateFixLogUnhandled')

    await wrapper.find('.ug-rate-fix-log-status-unhandled').trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamManagement.markRateFixRecordHandled).toHaveBeenCalledWith('2026-06-20T00:00:00Z-42-default-upstream-VIP')
    expect(wrapper.find('.ug-rate-fix-warning').exists()).toBe(false)
    expect(wrapper.find('.ug-rate-fix-logs-dialog').text()).toContain('admin.upstreamGroups.rateFixLogHandled')
  })

  it('opens a rate fix preview without applying changes immediately', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          upstream_group_key: 'vip',
          upstream_rate: 2.5,
          local_group_id: 7,
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
          DataTable: { template: '<div />' },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: true,
        },
      },
    })
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.fixRates'))!.trigger('click')

    expect(wrapper.find('.ug-rate-fix-preview-dialog').exists()).toBe(true)
    expect(wrapper.find('.ug-rate-fix-preview-dialog').text()).toContain('VIP local')
    expect(adminAPIMock.upstreamManagement.applyRateFixes).not.toHaveBeenCalled()
  })

  it('refreshes risks and applies all rate fixes after confirmation', async () => {
    const riskResult = {
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream', provider_name: 'Default upstream', upstream_group_name: 'VIP', upstream_group_key: 'vip',
          upstream_rate: 2.5, local_group_id: 7, local_group_name: 'VIP local', local_rate: 1.5, matched: true, needs_rate_increase: true,
        },
      ],
      warnings: [],
      records: [],
    }
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue(riskResult)
    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }, TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' }, EmptyState: true, Icon: true, UpstreamGroupAvailabilityTrend: { template: '<div />' }, Select: true,
        },
      },
    })
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.fixRates'))!.trigger('click')
    const callsBeforeConfirm = adminAPIMock.upstreamManagement.getGroups.mock.calls.length

    await wrapper.find('.ug-rate-fix-preview-confirm').trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamManagement.getGroups.mock.calls.length).toBeGreaterThan(callsBeforeConfirm)
    expect(adminAPIMock.upstreamManagement.applyRateFixes).toHaveBeenCalledTimes(1)
    expect(wrapper.find('.ug-rate-fix-preview-dialog').exists()).toBe(false)
  })

  it('closes the rate fix preview without applying when cancelled', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [{ provider_slug: 'default-upstream', upstream_group_name: 'VIP', upstream_group_key: 'vip', upstream_rate: 2.5, local_group_id: 7, local_group_name: 'VIP local', local_rate: 1.5, matched: true, needs_rate_increase: true }],
      warnings: [], records: [],
    })
    const wrapper = mount(UpstreamGroupsView, {
      global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' }, DataTable: { template: '<div />' }, EmptyState: true, Icon: true, UpstreamGroupAvailabilityTrend: { template: '<div />' }, Select: true } },
    })
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.fixRates'))!.trigger('click')

    await wrapper.find('.ug-rate-fix-preview-cancel').trigger('click')

    expect(adminAPIMock.upstreamManagement.applyRateFixes).not.toHaveBeenCalled()
    expect(wrapper.find('.ug-rate-fix-preview-dialog').exists()).toBe(false)
  })

  it('keeps the rate fix preview open when applying fails', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [{ provider_slug: 'default-upstream', upstream_group_name: 'VIP', upstream_group_key: 'vip', upstream_rate: 2.5, local_group_id: 7, local_group_name: 'VIP local', local_rate: 1.5, matched: true, needs_rate_increase: true }],
      warnings: [], records: [],
    })
    adminAPIMock.upstreamManagement.applyRateFixes.mockRejectedValue(new Error('apply failed'))
    const wrapper = mount(UpstreamGroupsView, {
      global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' }, DataTable: { template: '<div />' }, EmptyState: true, Icon: true, UpstreamGroupAvailabilityTrend: { template: '<div />' }, Select: true } },
    })
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.fixRates'))!.trigger('click')

    await wrapper.find('.ug-rate-fix-preview-confirm').trigger('click')
    await flushPromises()

    expect(wrapper.find('.ug-rate-fix-preview-dialog').exists()).toBe(true)
    expect(appStoreMock.showError).toHaveBeenCalled()
  })
})
