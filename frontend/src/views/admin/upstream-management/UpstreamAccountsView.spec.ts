import { flushPromises, mount } from '@vue/test-utils'
import { h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UpstreamAccountsView from './UpstreamAccountsView.vue'

const { upstreamAccountSyncMock, accountsMock, groupsMock, appStoreMock } = vi.hoisted(() => ({
  upstreamAccountSyncMock: {
    getPreview: vi.fn(),
    getRateGuardConfig: vi.fn(),
    runRateGuardNow: vi.fn(),
    getBalanceConsumption: vi.fn(),
    updateBalanceSamplerConfig: vi.fn(),
    addBalanceRecharge: vi.fn(),
    runBalanceSampleNow: vi.fn(),
  },
  accountsMock: {
    update: vi.fn(),
  },
  groupsMock: {
    getAllIncludingInactive: vi.fn(),
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
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
  adminAPI: {
    upstreamAccountSync: upstreamAccountSyncMock,
    accounts: accountsMock,
    groups: groupsMock,
  },
}))

vi.mock('@/api/admin/index', () => ({
  adminAPI: {
    upstreamAccountSync: upstreamAccountSyncMock,
    accounts: accountsMock,
    groups: groupsMock,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

describe('UpstreamAccountsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    upstreamAccountSyncMock.getPreview.mockResolvedValue({
      default_provider: {},
      providers: [],
      summary: {
        upstream_key_count: 0,
        matched_account_count: 0,
        create_count: 0,
        update_count: 0,
        skip_count: 0,
        conflict_count: 0,
        rate_violation_count: 0,
        unbound_group_count: 0,
      },
      items: [],
      warnings: [],
      records: [
        {
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          created_count: 0,
          updated_count: 1,
          skipped_count: 0,
          conflict_count: 0,
          rate_violation_count: 1,
          unbound_group_count: 1,
          created_at: '2026-06-15T00:00:00Z',
          trigger_source: 'manual_sync',
          unbind_details: [
            {
              provider_slug: 'upstream-a',
              provider_name: 'Upstream A',
              upstream_key_name: 'key-a',
              matched_local_account_id: 12,
              matched_local_account_name: 'local-a',
              upstream_group_name: 'upstream-group',
              upstream_rate_multiplier: 1,
              local_min_rate_multiplier: 0.5,
              unbound_group_ids: [8],
              unbound_group_names: ['low-rate'],
              remaining_group_ids: null,
            },
          ],
        },
      ],
    })
    upstreamAccountSyncMock.getRateGuardConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
    })
    upstreamAccountSyncMock.runRateGuardNow.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
      last_run_status: 'success',
    })
    upstreamAccountSyncMock.getBalanceConsumption.mockResolvedValue({
      config: {
        enabled: false,
        interval_seconds: 3600,
        provider_amount_scales: {},
      },
      summaries: {},
      rows: [],
    })
    groupsMock.getAllIncludingInactive.mockResolvedValue([
      { id: 7, name: 'VIP', platform: 'openai', rate_multiplier: 2, status: 'active' },
      { id: 8, name: 'Trial', platform: 'openai', rate_multiplier: 0.5, status: 'active' },
      { id: 9, name: 'Claude', platform: 'anthropic', rate_multiplier: 1, status: 'active' },
    ])
    accountsMock.update.mockResolvedValue({})
  })

  it('renders sync log entries when legacy remaining group ids are null', async () => {
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div><slot name="empty" /></div>' },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('local-a')
    expect(wrapper.text()).toContain('-')
  })

  it('warns when manual rate guard leaves rate risks after refresh', async () => {
    upstreamAccountSyncMock.getPreview
      .mockResolvedValueOnce({
        default_provider: {},
        providers: [],
        summary: {
          upstream_key_count: 0,
          matched_account_count: 0,
          create_count: 0,
          update_count: 0,
          skip_count: 0,
          conflict_count: 0,
          rate_violation_count: 0,
          unbound_group_count: 0,
        },
        items: [],
        warnings: [],
        records: [],
      })
      .mockResolvedValueOnce({
        default_provider: {},
        providers: [],
        summary: {
          upstream_key_count: 1,
          matched_account_count: 1,
          create_count: 0,
          update_count: 1,
          skip_count: 0,
          conflict_count: 0,
          rate_violation_count: 1,
          unbound_group_count: 0,
        },
        items: [],
        warnings: [],
        records: [],
      })

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div><slot name="empty" /></div>' },
          EmptyState: true,
          Icon: true,
          Select: true,
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.rateGuardRunNow'))?.trigger('click')
    await flushPromises()

    expect(appStoreMock.showWarning).toHaveBeenCalledWith(
      'admin.upstreamAccounts.rateGuardRunCompletedWithRisks'
    )
    expect(appStoreMock.showSuccess).not.toHaveBeenCalledWith(
      'admin.upstreamAccounts.rateGuardRunSuccess'
    )
  })

  it('renders provider homepage link in source column', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [],
      summary: {
        upstream_key_count: 1,
        matched_account_count: 0,
        create_count: 1,
        update_count: 0,
        skip_count: 0,
        conflict_count: 0,
        rate_violation_count: 0,
        unbound_group_count: 0,
      },
      items: [
        {
          action: 'create',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          provider_base_url: 'https://upstream.example.com',
          upstream_key_name: 'key-a',
          local_account_name: 'local-a',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
      ],
      warnings: [],
      records: [],
    })

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div><div v-for="row in data" :key="row.upstream_key_name"><slot name="cell-source" :row="row" /></div></div>',
          },
          EmptyState: true,
          Icon: true,
          Select: true,
        },
      },
    })

    await flushPromises()

    const homepage = wrapper.find('a[href="https://upstream.example.com"]')
    expect(homepage.exists()).toBe(true)
    expect(homepage.attributes('target')).toBe('_blank')
  })

  it('does not expose provider balance consumption as an account table column', async () => {
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns'],
            setup(props) {
              return () => h('div', { class: 'columns' }, props.columns.map((column: any) => column.key).join(','))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.columns').text()).not.toContain('balance_consumption')
  })

  it('edits matched account group bindings from the action column', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValue({
      default_provider: {},
      providers: [],
      summary: {
        upstream_key_count: 1,
        matched_account_count: 1,
        create_count: 0,
        update_count: 0,
        skip_count: 0,
        conflict_count: 0,
        rate_violation_count: 0,
        unbound_group_count: 0,
      },
      items: [
        {
          action: 'noop',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-a',
          local_account_name: 'local-a',
          matched_account_id: 12,
          matched_account_name: 'local-a',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 2,
          rate_violation: false,
          bound_groups: [
            { id: 7, name: 'VIP', rate_multiplier: 2, rate_violation: false },
          ],
        },
      ],
      warnings: [],
      records: [],
    })

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                slots['cell-actions']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
        },
      },
    })

    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.editBoundGroups'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const dialog = wrapper.find('.account-group-dialog')
    expect(dialog.exists()).toBe(true)
    const checkboxes = dialog.findAll('input[type="checkbox"]')
    expect(checkboxes).toHaveLength(2)
    expect(wrapper.text()).toContain('VIP')
    expect(wrapper.text()).toContain('Trial')
    expect(wrapper.text()).not.toContain('Claude')
    expect((checkboxes[0].element as HTMLInputElement).checked).toBe(true)

    await checkboxes[1].setValue(true)
    const saveButtons = wrapper.findAll('button').filter(button => button.text().includes('common.save'))
    await saveButtons[saveButtons.length - 1].trigger('click')
    await flushPromises()

    expect(accountsMock.update).toHaveBeenCalledWith(12, { group_ids: [7, 8] })
  })
})
