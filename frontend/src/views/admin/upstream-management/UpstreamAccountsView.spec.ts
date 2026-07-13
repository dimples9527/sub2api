import { flushPromises, mount } from '@vue/test-utils'
import { h, onMounted } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import upstreamAccountsSource from './UpstreamAccountsView.vue?raw'
import UpstreamAccountsView from './UpstreamAccountsView.vue'

const { upstreamAccountSyncMock, accountsMock, groupsMock, proxiesMock, appStoreMock, routeMock, routerReplaceMock } = vi.hoisted(() => ({
  upstreamAccountSyncMock: {
    getPreview: vi.fn(),
    runSync: vi.fn(),
    getRateGuardConfig: vi.fn(),
    updateRateGuardConfig: vi.fn(),
    runRateGuardNow: vi.fn(),
    getBalanceConsumption: vi.fn(),
    updateBalanceSamplerConfig: vi.fn(),
    addBalanceRecharge: vi.fn(),
    runBalanceSampleNow: vi.fn(),
    markRecordHandled: vi.fn(),
  },
  accountsMock: {
    getById: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batchTestAccounts: vi.fn(),
    getBatchTestJob: vi.fn(),
    cancelBatchTestJob: vi.fn(),
    setSchedulable: vi.fn(),
    getAvailableModels: vi.fn(),
  },
  groupsMock: {
    getAllIncludingInactive: vi.fn(),
    getAll: vi.fn(),
  },
  proxiesMock: {
    getAll: vi.fn(),
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  },
  routeMock: { query: {} as Record<string, string> },
  routerReplaceMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeMock,
  useRouter: () => ({ replace: routerReplaceMock }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: any) => {
        if (key === 'admin.upstreamAccounts.rateGuardIgnoredAccountId') return `ID ${params?.id ?? ''}`.trim()
        if (key === 'admin.upstreamAccounts.rateGuardUnknownAccount') return `Unknown account ${params?.id ?? ''}`.trim()
        if (key === 'admin.upstreamAccounts.rateGuardIgnoredSummary') return `${params?.count ?? 0} ignored`
        return key
      },
    }),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    upstreamAccountSync: upstreamAccountSyncMock,
    accounts: accountsMock,
    groups: groupsMock,
    proxies: proxiesMock,
  },
}))

vi.mock('@/api/admin/index', () => ({
  adminAPI: {
    upstreamAccountSync: upstreamAccountSyncMock,
    accounts: accountsMock,
    groups: groupsMock,
    proxies: proxiesMock,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

describe('UpstreamAccountsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    routeMock.query = {}
    window.localStorage.clear()
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
      ignored_account_ids: [],
    })
    upstreamAccountSyncMock.updateRateGuardConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
      ignored_account_ids: [],
    })
    upstreamAccountSyncMock.runSync.mockResolvedValue({
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
    upstreamAccountSyncMock.runRateGuardNow.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
      last_run_status: 'success',
    })
    upstreamAccountSyncMock.markRecordHandled.mockResolvedValue([
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
            remaining_group_ids: [],
            handled: true,
          },
        ],
      },
    ])
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
    groupsMock.getAll.mockResolvedValue([
      { id: 7, name: 'VIP', platform: 'openai', rate_multiplier: 2, status: 'active' },
      { id: 8, name: 'Trial', platform: 'openai', rate_multiplier: 0.5, status: 'active' },
    ])
    proxiesMock.getAll.mockResolvedValue([])
    accountsMock.getById.mockResolvedValue({
      id: 12,
      name: 'local-a',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      group_ids: [7],
      groups: [
        { id: 7, name: 'VIP', description: null, platform: 'openai', rate_multiplier: 2, is_exclusive: false, status: 'active', subscription_type: 'standard', daily_limit_usd: null, weekly_limit_usd: null, monthly_limit_usd: null, allow_image_generation: false, image_rate_independent: false, image_rate_multiplier: 1, image_price_1k: null, image_price_2k: null, image_price_4k: null, fallback_group_id: null, fallback_group_id_on_invalid_request: null, require_oauth_only: false, require_privacy_set: false, created_at: '2026-06-15T00:00:00Z', updated_at: '2026-06-15T00:00:00Z' },
      ],
    })
    accountsMock.update.mockResolvedValue({})
    accountsMock.delete.mockResolvedValue({})
    accountsMock.setSchedulable.mockResolvedValue({})
    accountsMock.getAvailableModels.mockResolvedValue([])
  })

  it('focuses the rate guard panel from the automation query and clears it', async () => {
    routeMock.query = { automation: 'rate-guard' }
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' }, EmptyState: true, Icon: true, Select: true, GroupSelector: true,
          UpstreamBalanceCharts: { template: '<div />' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-test="rate-guard-panel"]').classes()).toContain('is-automation-target')
    expect(routerReplaceMock).toHaveBeenCalledWith({ query: {} })
  })

  it('opens sync log entries in a dialog when legacy remaining group ids are null', async () => {
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
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.records-panel').exists()).toBe(false)
    expect(wrapper.find('.accounts-actions').text()).toContain('admin.upstreamAccounts.openSyncLogs')
    expect(wrapper.text()).toContain('admin.upstreamAccounts.openSyncLogs')
    expect(wrapper.text()).not.toContain('local-a')
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.openSyncLogs'))?.trigger('click')
    await flushPromises()

    expect(wrapper.find('.sync-logs-dialog').exists()).toBe(true)
    expect(wrapper.find('.sync-logs-dialog').text()).toContain('local-a')
    expect(wrapper.text()).toContain('-')
  })

  it('keeps sync log cards constrained for mobile dialogs', () => {
    expect(upstreamAccountsSource).toContain('.sync-logs-table-wrap {\n    display: none;')
    expect(upstreamAccountsSource).toContain('.sync-log-card-list {\n    display: grid;')
    expect(upstreamAccountsSource).toContain('.sync-log-card-head {\n    display: grid;')
    expect(upstreamAccountsSource).toContain('grid-template-columns: minmax(0, 1fr);')
    expect(upstreamAccountsSource).toContain('grid-auto-rows: max-content;')
    expect(upstreamAccountsSource).toContain('overflow: visible;')
    expect(upstreamAccountsSource).toContain('.sync-log-card .table-tag,\n  .sync-log-card .log-chip,\n  .sync-log-card .trigger-chip')
    expect(upstreamAccountsSource).toContain('min-height: 24px;')
    expect(upstreamAccountsSource).toContain('white-space: nowrap;')
    expect(upstreamAccountsSource).toContain('.sync-log-card-action {\n    width: 100%;')
    expect(upstreamAccountsSource).toContain('min-height: 34px;')
  })

  it('opens stat card detail dialogs with matching preview rows', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [{ slug: 'upstream-a', name: 'Upstream A' }],
      summary: {
        upstream_key_count: 4,
        matched_account_count: 2,
        create_count: 1,
        update_count: 2,
        skip_count: 0,
        conflict_count: 0,
        rate_violation_count: 1,
        unbound_group_count: 1,
      },
      items: [
        {
          action: 'create',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-new',
          upstream_api_key: 'sk-key-new',
          local_account_name: 'local-new',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          local_rate_multiplier: 2,
          rate_violation: false,
        },
        {
          action: 'create',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'display-only',
          local_account_name: 'local-display-only',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          local_rate_multiplier: 2,
          rate_violation: false,
        },
        {
          action: 'update',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-existing',
          local_account_name: 'local-existing',
          matched_account_id: 12,
          matched_account_name: 'local-existing',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          bound_groups: [
            { id: 7, name: 'VIP', rate_multiplier: 2, rate_violation: false },
          ],
          rate_violation: false,
        },
        {
          action: 'update',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-risk',
          local_account_name: 'local-risk',
          matched_account_id: 13,
          matched_account_name: 'local-risk',
          upstream_group_name: 'basic',
          upstream_rate_multiplier: 1,
          bound_groups: [
            { id: 8, name: 'Trial', rate_multiplier: 0.5, rate_violation: true },
          ],
          unbound_group_ids: [8],
          unbound_group_names: ['Trial'],
          rate_violation: true,
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
          DataTable: { template: '<div data-test="accounts-table" />' },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()

    await wrapper.find('[data-test="upstream-stat-card-total"]').trigger('click')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).toContain('key-new')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).toContain('display-only')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).toContain('key-existing')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).toContain('key-risk')

    await wrapper.find('.stat-details-dialog .modal-close-button').trigger('click')
    await wrapper.find('[data-test="upstream-stat-card-create"]').trigger('click')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).toContain('key-new')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).not.toContain('display-only')

    await wrapper.find('.stat-details-dialog .modal-close-button').trigger('click')
    await wrapper.find('[data-test="upstream-stat-card-update"]').trigger('click')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).toContain('key-existing')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).toContain('key-risk')
    expect(wrapper.find('[data-test="stat-details-dialog"]').text()).not.toContain('key-new')

    await wrapper.find('.stat-details-dialog .modal-close-button').trigger('click')
    await wrapper.find('[data-test="upstream-stat-card-risk"]').trigger('click')
    const riskDialog = wrapper.find('[data-test="stat-details-dialog"]').text()
    expect(riskDialog).toContain('key-risk')
    expect(riskDialog).toContain('Trial')
    expect(riskDialog).not.toContain('key-existing')
  })

  it('shows detailed sync confirmation and submits selected sync categories', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [{ slug: 'upstream-a', name: 'Upstream A' }],
      summary: {
        upstream_key_count: 3,
        matched_account_count: 2,
        create_count: 1,
        update_count: 2,
        skip_count: 0,
        conflict_count: 0,
        rate_violation_count: 1,
        unbound_group_count: 1,
      },
      items: [
        {
          action: 'create',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-new',
          upstream_api_key: 'sk-key-new',
          local_account_name: 'local-new',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          local_rate_multiplier: 2,
          rate_violation: false,
        },
        {
          action: 'create',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'display-only',
          local_account_name: 'local-display-only',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          local_rate_multiplier: 2,
          rate_violation: false,
        },
        {
          action: 'update',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-existing',
          local_account_name: 'local-existing',
          matched_account_id: 12,
          matched_account_name: 'local-existing',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          local_rate_multiplier: 2,
          rate_violation: false,
          bound_groups: [
            { id: 9, name: 'Legacy', rate_multiplier: 1, rate_violation: false },
          ],
          change_details: [
            { kind: 'credential', field: 'api_key', label: 'API key', before: 'old-key', after: 'key-existing' },
            { kind: 'credential', field: 'base_url', label: 'Base URL', before: 'https://old.example.com', after: 'https://upstream.example.com' },
            { kind: 'metadata', field: 'upstream', label: 'Upstream sync metadata' },
            { kind: 'group_bind', field: 'group_ids', label: 'Bind local group', group_ids: [7], group_names: ['VIP'] },
          ],
        },
        {
          action: 'update',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-risk',
          local_account_name: 'local-risk',
          matched_account_id: 13,
          matched_account_name: 'local-risk',
          upstream_group_name: 'basic',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          local_rate_multiplier: 2,
          rate_violation: true,
          unbound_group_ids: [8],
          unbound_group_names: ['Trial'],
          bound_groups: [
            { id: 7, name: 'VIP', rate_multiplier: 2, rate_violation: false },
            { id: 8, name: 'Trial', rate_multiplier: 0.5, rate_violation: true },
          ],
          change_details: [
            { kind: 'group_unbind', field: 'group_ids', label: 'Unbind low-rate groups', group_ids: [8], group_names: ['Trial'] },
          ],
        },
      ],
      warnings: [],
      records: [],
    })
    upstreamAccountSyncMock.runSync.mockResolvedValueOnce({
      default_provider: {},
      providers: [{ slug: 'upstream-a', name: 'Upstream A' }],
      summary: {
        upstream_key_count: 3,
        matched_account_count: 2,
        create_count: 0,
        update_count: 2,
        skip_count: 0,
        conflict_count: 0,
        rate_violation_count: 0,
        unbound_group_count: 0,
      },
      items: [
        {
          action: 'update',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-existing',
          local_account_name: 'local-existing',
          matched_account_id: 12,
          matched_account_name: 'local-existing',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: false,
          execution: {
            executed: true,
            action: 'update',
            account_id: 12,
            account_name: 'local-existing',
          },
        },
        {
          action: 'update',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-risk',
          local_account_name: 'local-risk',
          matched_account_id: 13,
          matched_account_name: 'local-risk',
          upstream_group_name: 'basic',
          upstream_rate_multiplier: 1,
          rate_violation: true,
          execution: {
            executed: true,
            action: 'update',
            account_id: 13,
            account_name: 'local-risk',
            unbound_group_ids: [8],
            unbound_group_names: ['Trial'],
          },
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
          DataTable: { template: '<div data-test="accounts-table" />' },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.syncNow'))?.trigger('click')
    await flushPromises()

    const dialog = wrapper.find('[data-test="sync-confirm-dialog"]')
    expect(dialog.exists()).toBe(true)
    expect(dialog.text()).toContain('key-new')
    expect(dialog.text()).toContain('local-new')
    expect(dialog.text()).not.toContain('display-only')
    expect(dialog.text()).not.toContain('local-display-only')
    expect(dialog.text()).toContain('key-existing')
    expect(dialog.text()).toContain('local-existing')
    expect(dialog.text()).toContain('key-risk')
    expect(dialog.text()).toContain('Trial')
    expect(dialog.text()).toContain('API key: old-key -> key-existing')
    expect(dialog.text()).toContain('Base URL: https://old.example.com -> https://upstream.example.com')
    expect(dialog.text()).toContain('Upstream sync metadata')
    expect(dialog.text()).toContain('Bind local group: VIP')
    expect(dialog.text()).toContain('VIP')
    expect(dialog.text()).toContain('Unbind low-rate groups: Trial')

    await wrapper.find('[data-test="sync-confirm-item-create-upstream-a-key-new"]').setValue(false)
    await wrapper.find('[data-test="sync-confirm-apply-rate-guard"]').setValue(false)
    await wrapper.find('[data-test="sync-confirm-submit"]').trigger('click')
    await flushPromises()

    expect(upstreamAccountSyncMock.runSync).toHaveBeenCalledWith({
      create_missing: false,
      update_existing: true,
      apply_rate_guard: false,
      selected_items: [
        {
          provider_slug: 'upstream-a',
          upstream_key_name: 'key-existing',
          create_missing: false,
          update_existing: true,
          apply_rate_guard: false,
        },
        {
          provider_slug: 'upstream-a',
          upstream_key_name: 'key-risk',
          create_missing: false,
          update_existing: true,
          apply_rate_guard: false,
        },
      ],
    })
    expect(wrapper.find('[data-test="sync-result-dialog"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="sync-result-dialog"]').text()).toContain('local-existing')
    expect(wrapper.find('[data-test="sync-result-dialog"]').text()).toContain('local-risk')
    expect(wrapper.find('[data-test="sync-result-dialog"]').text()).toContain('Trial')
  })

  it('keeps disabled provider accounts visible without rendering the schedulable toggle', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [{ slug: 'disabled-upstream', name: 'Disabled Upstream', enabled: false }],
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
          provider_slug: 'disabled-upstream',
          provider_name: 'Disabled Upstream',
          upstream_key_name: 'key-disabled',
          local_account_name: 'local-disabled',
          matched_account_id: 12,
          matched_account_name: 'local-disabled',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockResolvedValueOnce({
      id: 12,
      name: 'local-disabled',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [7],
      groups: [],
    })

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns', 'data', 'rowClass'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any, index: number) => h('div', {
                class: ['table-row', typeof props.rowClass === 'function' ? props.rowClass(row, index) : props.rowClass]
              }, [
                h('div', { class: 'source-cell-test' }, slots['cell-source']?.({ row })),
                h('div', { class: 'schedulable-cell-test' }, slots['cell-schedulable']?.({ row })),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          AccountStatusIndicator: true,
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Disabled Upstream')
    expect(wrapper.find('.table-row').classes()).toContain('provider-disabled-row')
    expect(wrapper.find('.source-cell-test .source-cell').exists()).toBe(true)
    expect(wrapper.find('.source-cell-test').text()).toContain('Disabled Upstream')
    const toggle = wrapper.find('.schedulable-cell-test .schedulable-toggle')
    expect(toggle.exists()).toBe(false)
    expect(accountsMock.setSchedulable).not.toHaveBeenCalled()
    expect(upstreamAccountsSource).toContain('.accounts-table-card :deep(.data-table-row.provider-disabled-row td > *)')
    expect(upstreamAccountsSource).toContain('filter: grayscale(1);')
    expect(upstreamAccountsSource).toContain('opacity: 0.46;')
  })

  it('filters upstream accounts by enabled provider from quick tags', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [
        { slug: 'enabled-upstream', name: 'Enabled Upstream', enabled: true },
        { slug: 'disabled-upstream', name: 'Disabled Upstream', enabled: false },
      ],
      summary: {
        upstream_key_count: 2,
        matched_account_count: 0,
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
          provider_slug: 'enabled-upstream',
          provider_name: 'Enabled Upstream',
          upstream_key_name: 'key-enabled',
          local_account_name: 'local-enabled',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
        {
          action: 'noop',
          provider_slug: 'disabled-upstream',
          provider_name: 'Disabled Upstream',
          upstream_key_name: 'key-disabled',
          local_account_name: 'local-disabled',
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
          TablePageLayout: { template: '<div><slot name=filters /><slot name=table /></div>' },
          DataTable: {
            props: ['data'],
            setup(props) {
              return () => h('div', { class: 'account-rows' }, props.data.map((row: any) => h('div', { class: 'account-row' }, row.upstream_key_name)))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          UpstreamBalanceCharts: { template: '<div data-test=balance-charts />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.findAll('.account-row').map(row => row.text())).toEqual(['key-enabled', 'key-disabled'])
    const enabledQuickTag = wrapper.findAll('.quick-tag').find(button => button.text().includes('admin.upstreamAccounts.quickFilterEnabledProvider'))
    expect(enabledQuickTag?.text()).toContain('1')

    await enabledQuickTag?.trigger('click')

    expect(wrapper.findAll('.account-row').map(row => row.text())).toEqual(['key-enabled'])
  })

  it('filters ignored rate guard accounts from quick tags and labels ignored accounts', async () => {
    upstreamAccountSyncMock.getRateGuardConfig.mockResolvedValueOnce({
      enabled: false,
      interval_seconds: 3600,
      ignored_account_ids: [12],
    })
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [{ slug: 'upstream-a', name: 'Upstream A', enabled: true }],
      summary: {
        upstream_key_count: 2,
        matched_account_count: 2,
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
          upstream_rate_multiplier: 1,
          rate_violation: false,
          rate_guard_ignored: true,
        },
        {
          action: 'noop',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-b',
          local_account_name: 'local-b',
          matched_account_id: 13,
          matched_account_name: 'local-b',
          upstream_group_name: 'trial',
          upstream_rate_multiplier: 1,
          rate_violation: false,
          rate_guard_ignored: false,
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockImplementation(async (id: number) => ({
      id,
      name: id === 12 ? 'local-a' : 'local-b',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      group_ids: [],
      groups: [],
    }))

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name=filters /><slot name=table /></div>' },
          DataTable: {
            props: ['data'],
            setup(props) {
              return () => h('div', { class: 'account-rows' }, props.data.map((row: any) => h('div', { class: 'account-row' }, row.upstream_key_name)))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          UpstreamBalanceCharts: { template: '<div data-test=balance-charts />' },
        },
      },
    })

    await flushPromises()

    const manageButton = wrapper.find('[data-test="rate-guard-ignored-manage"]')
    expect(manageButton.exists()).toBe(true)
    expect(manageButton.text()).toContain('1')
    expect(wrapper.find('[data-test="rate-guard-ignored-account-chips"]').exists()).toBe(false)

    await manageButton.trigger('click')
    await flushPromises()

    const chips = wrapper.find('[data-test="rate-guard-ignored-account-chips"]')
    expect(chips.exists()).toBe(true)
    expect(chips.text()).toContain('local-a')
    expect(chips.text()).toContain('ID 12')
    expect(wrapper.findAll('.account-row').map(row => row.text())).toEqual(['key-a', 'key-b'])

    const ignoredQuickTag = wrapper.findAll('.quick-tag').find(button => button.text().includes('admin.upstreamAccounts.quickFilterIgnoredAccounts'))
    expect(ignoredQuickTag?.text()).toContain('1')
    await ignoredQuickTag?.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('.account-row').map(row => row.text())).toEqual(['key-a'])
  })

  it('marks provider fetch fallback rows as local snapshots', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [{ slug: 'bad-upstream', name: 'Bad Upstream', enabled: true }],
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
          provider_slug: 'bad-upstream',
          provider_name: 'Bad Upstream',
          upstream_key_name: 'key-local',
          local_account_name: 'local-account',
          matched_account_id: 12,
          matched_account_name: 'local-account',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          provider_fetch_error: 'newapi login failed: Turnstile token is empty',
          rate_violation: false,
        },
      ],
      warnings: ['Bad Upstream: newapi login failed: Turnstile token is empty'],
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
              return () => h('div', props.data.map((row: any) => h('div', { class: 'source-row' }, slots['cell-source']?.({ row }))))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.source-row').text()).toContain('admin.upstreamAccounts.localSnapshotTag')
    expect(wrapper.find('.tag-local-snapshot').attributes('title')).toBe('newapi login failed: Turnstile token is empty')
    expect(wrapper.find('.source-line').classes()).toContain('source-line-amber')
    expect(wrapper.text()).toContain('Bad Upstream: newapi login failed: Turnstile token is empty')
  })

  it('uses disabled default provider state for upstream account rows', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: { slug: 'main', name: 'Default Upstream', enabled: false, is_default: true },
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
          provider_slug: 'main',
          provider_name: 'Default Upstream',
          upstream_key_name: 'default-key',
          local_account_name: 'default-key',
          matched_account_id: 12,
          matched_account_name: 'default-key',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockResolvedValueOnce({
      id: 12,
      name: 'default-key',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [7],
      groups: [],
    })

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data', 'rowClass'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any, index: number) => h('div', {
                class: ['table-row', typeof props.rowClass === 'function' ? props.rowClass(row, index) : props.rowClass]
              }, [
                h('div', { class: 'source-cell-test' }, slots['cell-source']?.({ row })),
                h('div', { class: 'schedulable-cell-test' }, slots['cell-schedulable']?.({ row })),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          AccountStatusIndicator: true,
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.table-row').classes()).toContain('provider-disabled-row')
    expect(wrapper.find('.source-cell-test').text()).toContain('Default Upstream')
    expect(wrapper.find('.schedulable-cell-test .schedulable-toggle').exists()).toBe(false)
  })

  it('does not render persisted sync records without unbind details', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
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
          created_count: 1,
          updated_count: 2,
          skipped_count: 0,
          conflict_count: 0,
          rate_violation_count: 0,
          unbound_group_count: 0,
          created_at: '2026-06-15T00:00:00Z',
          trigger_source: 'manual_sync',
          unbind_details: [],
        },
      ],
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
          GroupSelector: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).not.toContain('Upstream A')
    expect(wrapper.text()).not.toContain('admin.upstreamAccounts.syncSummaryCreated 1')
    expect(wrapper.text()).toContain('admin.upstreamAccounts.openSyncLogs')

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.openSyncLogs'))?.trigger('click')
    await flushPromises()

    expect(wrapper.find('.sync-logs-dialog').text()).toContain('admin.upstreamAccounts.noSyncLogs')
  })

  it('normalizes sync log timestamps before marking unhandled entries handled', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [],
      summary: {
        upstream_key_count: 0,
        matched_account_count: 0,
        create_count: 0,
        update_count: 0,
        skip_count: 0,
        conflict_count: 0,
        rate_violation_count: 1,
        unbound_group_count: 1,
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
          created_at: '2026-06-15T00:00:00.000Z',
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

    expect(wrapper.find('.guard-sync-log-warning').exists()).toBe(true)
    expect(wrapper.find('.guard-sync-log-warning').text()).toContain('1')

    await wrapper.find('.guard-sync-log-warning button').trigger('click')
    await flushPromises()
    expect(wrapper.find('.sync-logs-dialog').exists()).toBe(true)
    expect(wrapper.find('.sync-logs-dialog').text()).toContain('admin.upstreamAccounts.syncLogUnhandled')

    const unhandledStatus = wrapper.find('.sync-log-status-unhandled')
    expect(unhandledStatus.element.tagName).toBe('BUTTON')
    await unhandledStatus.trigger('click')
    await flushPromises()

    expect(wrapper.find('.guard-sync-log-warning').exists()).toBe(false)
    expect(wrapper.find('.sync-logs-dialog').text()).toContain('admin.upstreamAccounts.syncLogHandled')
    expect(upstreamAccountSyncMock.markRecordHandled).toHaveBeenCalledWith('2026-06-15T00:00:00Z-12-key-a-8')
  })

  it('does not render balance charts above the upstream account table', async () => {
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><div data-test="filters"><slot name="filters" /></div><div data-test="table"><slot name="table" /></div></div>' },
          DataTable: { template: '<div data-test="accounts-table" />' },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('[data-test="balance-charts"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="accounts-table"]').exists()).toBe(true)
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
          AccountTestModal: true,
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

  it('toggles rate guard ignore for a matched account', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValue({
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
        unbound_group_count: 1,
      },
      items: [
        {
          action: 'update',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-a',
          local_account_name: 'local-a',
          matched_account_id: 12,
          matched_account_name: 'local-a',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: true,
          unbound_group_ids: [8],
          bound_groups: [{ id: 8, name: 'low-rate', rate_multiplier: 0.5, rate_violation: true }],
        },
      ],
      warnings: [],
      records: [],
    })
    upstreamAccountSyncMock.updateRateGuardConfig.mockResolvedValueOnce({
      enabled: false,
      interval_seconds: 3600,
      ignored_account_ids: [12],
    })

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div><div v-for="row in data" :key="row.upstream_key_name"><slot name="cell-actions" :row="row" /></div></div>',
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          AccountTestModal: true,
        },
      },
    })

    await flushPromises()
    await wrapper.find('[data-test="rate-guard-ignore-toggle-12"]').trigger('click')
    await flushPromises()

    expect(upstreamAccountSyncMock.updateRateGuardConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: false,
        interval_seconds: 3600,
        ignored_account_ids: [12],
      })
    )
    expect(upstreamAccountSyncMock.getPreview).toHaveBeenCalledTimes(2)
    expect(appStoreMock.showSuccess).toHaveBeenCalledWith('admin.upstreamAccounts.rateGuardSaved')
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

  it('filters upstream accounts by bound group', async () => {
    groupsMock.getAllIncludingInactive.mockResolvedValueOnce([
      { id: 7, name: 'VIP', platform: 'openai', rate_multiplier: 2, status: 'active' },
      { id: 8, name: 'Trial', platform: 'openai', rate_multiplier: 0.5, status: 'active' },
    ])
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [],
      summary: {
        upstream_key_count: 2,
        matched_account_count: 2,
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
          local_group_id: 7,
          local_group_name: 'VIP',
          upstream_rate_multiplier: 2,
          rate_violation: false,
          bound_groups: [
            { id: 7, name: 'VIP', rate_multiplier: 2, rate_violation: false },
          ],
        },
        {
          action: 'noop',
          provider_slug: 'upstream-b',
          provider_name: 'Upstream B',
          upstream_key_name: 'key-b',
          local_account_name: 'local-b',
          matched_account_id: 13,
          matched_account_name: 'local-b',
          upstream_group_name: 'trial',
          local_group_id: 8,
          local_group_name: 'Trial',
          upstream_rate_multiplier: 0.5,
          rate_violation: false,
          bound_groups: [
            { id: 8, name: 'Trial', rate_multiplier: 0.5, rate_violation: false },
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
              return () => h('div', props.data.map((row: any) => h('div', { class: 'row' }, [
                h('div', { class: 'account-cell' }, slots['cell-local_account_name']?.({ row })),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            setup(props, { emit }) {
              return () => h('button', {
                class: 'select-stub',
                onClick: () => emit('update:modelValue', props.options?.[1]?.value ?? ''),
              }, props.options?.map((option: any) => option.label).join(','))
            },
          },
          GroupSelector: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('local-a')
    expect(wrapper.text()).toContain('local-b')

    const groupSelect = wrapper.findAll('.select-stub').at(3)
    expect(groupSelect).toBeTruthy()
    expect(groupSelect!.text()).toContain('VIP')
    expect(groupSelect!.text()).toContain('Trial')
    await groupSelect!.trigger('click')

    expect(wrapper.text()).toContain('local-a')
    expect(wrapper.text()).not.toContain('local-b')
  })

  it('filters upstream accounts by platform dropdown', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [{ slug: 'upstream-a', name: 'Upstream A' }],
      summary: {
        upstream_key_count: 2,
        matched_account_count: 2,
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
          upstream_key_name: 'key-openai',
          local_account_name: 'local-openai',
          matched_account_id: 12,
          matched_account_name: 'local-openai',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
        {
          action: 'noop',
          provider_slug: 'upstream-a',
          provider_name: 'Upstream A',
          upstream_key_name: 'key-claude',
          local_account_name: 'local-claude',
          matched_account_id: 13,
          matched_account_name: 'local-claude',
          upstream_group_name: 'claude',
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockImplementation(async (id: number) => ({
      id,
      name: id === 12 ? 'local-openai' : 'local-claude',
      platform: id === 12 ? 'openai' : 'anthropic',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [],
      groups: [],
    }))

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props) {
              return () => h('div', props.data.map((row: any) => h('div', { class: 'row-key' }, row.upstream_key_name)))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            setup(props, { emit }) {
              return () => h('select', {
                class: 'select-stub',
                value: props.modelValue,
                onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
              }, (props.options || []).map((option: any) => h('option', { value: option.value }, option.label)))
            },
          },
          GroupSelector: true,
          AccountStatusIndicator: true,
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(wrapper.findAll('.row-key').map(node => node.text())).toEqual(['key-openai', 'key-claude'])

    await wrapper.findAll('select.select-stub')[0].setValue('anthropic')
    await flushPromises()

    expect(wrapper.findAll('.row-key').map(node => node.text())).toEqual(['key-claude'])
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

  it('marks account table metric and state columns as sortable', async () => {
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['columns'],
            setup(props) {
              return () => h('div', { class: 'columns' }, props.columns.map((column: any) => `${column.key}:${column.sortable ? '1' : '0'}`).join(','))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.find('.columns').text()).toContain('source:1')
    expect(wrapper.find('.columns').text()).toContain('upstream_rate_multiplier:1')
    expect(wrapper.find('.columns').text()).toContain('balance:1')
    expect(wrapper.find('.columns').text()).toContain('status:1')
    expect(wrapper.find('.columns').text()).toContain('schedulable:1')
    expect(wrapper.find('.columns').text()).toContain('test_status:1')
  })

  it('passes derived sortable values to the upstream account table', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValueOnce({
      default_provider: {},
      providers: [
        { slug: 'upstream-a', name: 'Upstream A' },
        { slug: 'upstream-b', name: 'Upstream B' },
      ],
      summary: {
        upstream_key_count: 2,
        matched_account_count: 2,
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
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
        {
          action: 'noop',
          provider_slug: 'upstream-b',
          provider_name: 'Upstream B',
          upstream_key_name: 'key-b',
          local_account_name: 'local-b',
          matched_account_id: 13,
          matched_account_name: 'local-b',
          upstream_group_name: 'vip',
          upstream_rate_multiplier: 1,
          rate_violation: false,
        },
      ],
      warnings: [],
      records: [],
    })
    upstreamAccountSyncMock.getBalanceConsumption.mockResolvedValueOnce({
      config: { enabled: false, interval_seconds: 3600, provider_amount_scales: {} },
      summaries: {
        'upstream-a': {
          provider_slug: 'upstream-a',
          current_balance: 12.5,
          today_consumption: 1,
          amount_scale: 1,
          complete: true,
          anomaly: false,
          snapshot_count: 1,
        },
        'upstream-b': {
          provider_slug: 'upstream-b',
          current_balance: 5,
          today_consumption: 2,
          amount_scale: 1,
          complete: true,
          anomaly: false,
          snapshot_count: 1,
        },
      },
      rows: [],
    })
    accountsMock.getById.mockImplementation(async (id: number) => ({
      id,
      name: id === 12 ? 'local-a' : 'local-b',
      platform: 'openai',
      type: 'apikey',
      status: id === 12 ? 'active' : 'disabled',
      schedulable: id === 12,
      last_test_status: id === 12 ? 'success' : 'failed',
      group_ids: [],
      groups: [],
    }))

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props) {
              return () => h('div', { class: 'rows' }, props.data.map((row: any) => h('div', { class: 'row-sort-values' }, [
                row.upstream_key_name,
                row.source,
                row.balance,
                row.status,
                row.schedulable,
                row.test_status,
              ].join(':'))))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: true,
          UpstreamBalanceCharts: { template: '<div data-test="balance-charts" />' },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(wrapper.findAll('.row-sort-values').map(node => node.text())).toEqual([
      'key-a:Upstream A:12.5:active:1:1',
      'key-b:Upstream B:5:disabled:0:2',
    ])
  })

  it('adds status, schedulable, and last tested columns to the upstream account table', async () => {
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

    expect(wrapper.find('.columns').text()).toContain('status')
    expect(wrapper.find('.columns').text()).toContain('schedulable')
    expect(wrapper.find('.columns').text()).toContain('last_tested_at')
  })

  it('uses fixed upstream account table column classes for stable headers and wrapping groups', async () => {
    const wrapper = mount(UpstreamAccountsView, {
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
          Select: true,
        },
      },
    })

    await flushPromises()

    const classes = wrapper.find('.columns').text()
    expect(classes).toContain('source:upstream-center-column upstream-source-column')
    expect(classes).toContain('local_group_name:upstream-center-column upstream-bound-groups-column')
    expect(classes).toContain('actions:upstream-center-column upstream-actions-column')
  })

  it('lets the fixed upstream account table fill wide containers', () => {
    expect(upstreamAccountsSource).toContain('width: max(100%, 1700px);')
    expect(upstreamAccountsSource).not.toMatch(/^\s+width:\s*1700px;$/m)
  })

  it('lets the upstream account table occupy the page instead of reserving space for a sync log card', () => {
    expect(upstreamAccountsSource).not.toContain('class="records-panel"')
    expect(upstreamAccountsSource).not.toContain('max-height: 42rem;')
    expect(upstreamAccountsSource).toContain('flex: 1 1 auto;')
    expect(upstreamAccountsSource).toMatch(/^\s+height: 80vh;$/m)
    expect(upstreamAccountsSource).toContain('max-height: 80vh;')
    expect(upstreamAccountsSource).toContain('.records-table-wrap.sync-logs-table-wrap')
  })

  it('keeps the batch test result dialog scrollable on narrow screens', () => {
    expect(upstreamAccountsSource).toContain('.batch-test-result-dialog {')
    expect(upstreamAccountsSource).toContain('overflow: auto;')
    expect(upstreamAccountsSource).toContain('.batch-test-result-modal .sync-confirm-body')
    expect(upstreamAccountsSource).toContain('.batch-test-result-modal .sync-confirm-section')
    expect(upstreamAccountsSource).toContain('.batch-test-result-modal .batch-test-table-wrap')
    expect(upstreamAccountsSource).toContain('-webkit-overflow-scrolling: touch;')
    expect(upstreamAccountsSource).toContain('.sync-result-modal.batch-test-result-modal')
    expect(upstreamAccountsSource).toContain('height: calc(100dvh - 24px);')
  })

  it('opens create account modal from upstream account toolbar and refreshes after create', async () => {
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div><slot name="empty" /></div>' },
          EmptyState: true,
          Icon: true,
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

    expect(proxiesMock.getAll).toHaveBeenCalled()
    expect(groupsMock.getAll).toHaveBeenCalled()
    expect(wrapper.find('.create-account-modal').exists()).toBe(true)

    await wrapper.find('.create-account-modal').trigger('click')
    await flushPromises()

    expect(upstreamAccountSyncMock.getPreview).toHaveBeenCalledTimes(2)
  })

  it('opens create account modal from an unmatched upstream account row with upstream defaults', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValue({
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
          provider_slug: 'backup',
          provider_name: 'Backup',
          provider_base_url: 'https://backup.example.com',
          upstream_key_name: 'sk-live-001',
          upstream_api_key: 'sk-live-001',
          upstream_base_url: 'https://backup.example.com',
          local_account_name: 'backup-sk-live-001',
          upstream_group_name: 'VIP',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          rate_violation: false,
        },
      ],
      warnings: [],
      records: [],
    })
    let modalProps: any
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                h('div', { class: 'actions-slot' }, slots['cell-actions']?.({ row })),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          CreateAccountModal: {
            props: ['show', 'proxies', 'groups', 'initialValues'],
            setup(props) {
              modalProps = props
              return () => props.show ? h('div', { class: 'create-account-modal' }, 'create') : null
            },
          },
        },
      },
    })

    await flushPromises()
    await wrapper.find('[data-test="create-local-account-backup-sk-live-001"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('.create-account-modal').exists()).toBe(true)
    expect(modalProps.initialValues).toEqual({
      name: 'backup-sk-live-001',
      platform: 'openai',
      type: 'apikey',
      base_url: 'https://backup.example.com',
      api_key: 'sk-live-001',
      group_ids: [7],
    })
  })

  it('does not prefill create account api key from upstream key name', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValue({
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
          provider_slug: 'backup',
          provider_name: 'Backup',
          provider_base_url: 'https://backup.example.com',
          upstream_key_name: 'display-name-only',
          upstream_base_url: 'https://backup.example.com',
          local_account_name: 'backup-display-name-only',
          upstream_group_name: 'VIP',
          upstream_rate_multiplier: 1,
          local_group_id: 7,
          local_group_name: 'VIP',
          rate_violation: false,
        },
      ],
      warnings: [],
      records: [],
    })
    let modalProps: any
    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                h('div', { class: 'actions-slot' }, slots['cell-actions']?.({ row })),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          CreateAccountModal: {
            props: ['show', 'proxies', 'groups', 'initialValues'],
            setup(props) {
              modalProps = props
              return () => props.show ? h('div', { class: 'create-account-modal' }, 'create') : null
            },
          },
        },
      },
    })

    await flushPromises()
    await wrapper.find('[data-test="create-local-account-backup-display-name-only"]').trigger('click')
    await flushPromises()

    expect(modalProps.initialValues.api_key).toBe('')
  })

  it('colors upstream key, local account, and bound groups by matched account platform', async () => {
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
          provider_slug: 'anthropic-upstream',
          provider_name: 'Anthropic Upstream',
          upstream_key_name: 'claude-key',
          local_account_name: 'local-claude',
          matched_account_id: 12,
          matched_account_name: 'local-claude',
          upstream_group_name: 'claude',
          upstream_rate_multiplier: 1,
          rate_violation: false,
          bound_groups: [
            { id: 9, name: 'Claude', rate_multiplier: 1, rate_violation: false },
          ],
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockResolvedValueOnce({
      id: 12,
      name: 'local-claude',
      platform: 'anthropic',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [9],
      groups: [],
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
                h('div', { class: 'upstream-slot' }, slots['cell-upstream_key_name']?.({ row })),
                h('div', { class: 'local-slot' }, slots['cell-local_account_name']?.({ row })),
                h('div', { class: 'groups-slot' }, slots['cell-local_group_name']?.({ row })),
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

    expect(wrapper.find('.upstream-slot .main-text').classes()).toContain('platform-text-anthropic')
    expect(wrapper.find('.local-slot .main-text').classes()).toContain('platform-text-anthropic')
    expect(wrapper.find('.local-slot .account-id-tag').classes()).toContain('platform-tag-anthropic')
    expect(wrapper.find('.groups-slot .group-chip').classes()).toContain('platform-tag-anthropic')
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
                slots['cell-test_status']?.({ row }),
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

  it('refreshes upstream preview after saving matched account group bindings', async () => {
    const previewWithVipOnly = {
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
    }
    upstreamAccountSyncMock.getPreview
      .mockResolvedValueOnce(previewWithVipOnly)
      .mockResolvedValueOnce({
        ...previewWithVipOnly,
        items: [
          {
            ...previewWithVipOnly.items[0],
            bound_groups: [
              { id: 7, name: 'VIP', rate_multiplier: 2, rate_violation: false },
              { id: 8, name: 'Trial', rate_multiplier: 0.5, rate_violation: false },
            ],
          },
        ],
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
                h('div', { class: 'groups-slot' }, slots['cell-local_group_name']?.({ row })),
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

    expect(wrapper.find('.groups-slot').text()).toContain('VIP')
    expect(wrapper.find('.groups-slot').text()).not.toContain('Trial')

    const editButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.editBoundGroups'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const checkboxes = wrapper.find('.account-group-dialog').findAll('input[type="checkbox"]')
    await checkboxes[1].setValue(true)
    const saveButtons = wrapper.findAll('button').filter(button => button.text().includes('common.save'))
    await saveButtons[saveButtons.length - 1].trigger('click')
    await flushPromises()

    expect(accountsMock.update).toHaveBeenCalledWith(12, { group_ids: [7, 8] })
    expect(upstreamAccountSyncMock.getPreview).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.groups-slot').text()).toContain('Trial')
  })

  it('updates bound groups locally when the refreshed preview is stale after saving', async () => {
    const previewWithVipOnly = {
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
    }
    upstreamAccountSyncMock.getPreview
      .mockResolvedValueOnce(previewWithVipOnly)
      .mockResolvedValueOnce(previewWithVipOnly)

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                h('div', { class: 'groups-slot' }, slots['cell-local_group_name']?.({ row })),
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

    expect(wrapper.find('.groups-slot').text()).toContain('VIP')
    expect(wrapper.find('.groups-slot').text()).not.toContain('Trial')

    const editButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.editBoundGroups'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const checkboxes = wrapper.find('.account-group-dialog').findAll('input[type="checkbox"]')
    await checkboxes[1].setValue(true)
    const saveButtons = wrapper.findAll('button').filter(button => button.text().includes('common.save'))
    await saveButtons[saveButtons.length - 1].trigger('click')
    await flushPromises()

    expect(accountsMock.update).toHaveBeenCalledWith(12, { group_ids: [7, 8] })
    expect(upstreamAccountSyncMock.getPreview).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.groups-slot').text()).toContain('VIP')
    expect(wrapper.find('.groups-slot').text()).toContain('Trial')
  })

  it('shows anthropic groups in the edit bindings dialog for anthropic accounts', async () => {
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
          provider_slug: 'anthropic-upstream',
          provider_name: 'Anthropic Upstream',
          upstream_key_name: 'claude-key',
          local_account_name: 'local-claude',
          matched_account_id: 12,
          matched_account_name: 'local-claude',
          upstream_group_name: 'claude',
          upstream_rate_multiplier: 1,
          rate_violation: false,
          bound_groups: [
            { id: 9, name: 'Claude', rate_multiplier: 1, rate_violation: false },
          ],
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockResolvedValueOnce({
      id: 12,
      name: 'local-claude',
      platform: 'anthropic',
      type: 'apikey',
      status: 'active',
      group_ids: [9],
      groups: [
        { id: 9, name: 'Claude', description: null, platform: 'anthropic', rate_multiplier: 1, is_exclusive: false, status: 'active', subscription_type: 'standard', daily_limit_usd: null, weekly_limit_usd: null, monthly_limit_usd: null, allow_image_generation: false, image_rate_independent: false, image_rate_multiplier: 1, image_price_1k: null, image_price_2k: null, image_price_4k: null, fallback_group_id: null, fallback_group_id_on_invalid_request: null, require_oauth_only: false, require_privacy_set: false, created_at: '2026-06-15T00:00:00Z', updated_at: '2026-06-15T00:00:00Z' },
      ],
    })
    groupsMock.getAllIncludingInactive.mockResolvedValueOnce([
      { id: 7, name: 'VIP', platform: 'openai', rate_multiplier: 2, status: 'active' },
      { id: 8, name: 'Trial', platform: 'openai', rate_multiplier: 0.5, status: 'active' },
      { id: 9, name: 'Claude', platform: 'anthropic', rate_multiplier: 1, status: 'active' },
    ])

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div><div v-for="row in data" :key="row.upstream_key_name"><slot name="cell-actions" :row="row" /></div></div>',
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          GroupSelector: {
            props: ['platform', 'groups', 'modelValue'],
            setup(props) {
              return () => h('div', { class: 'group-selector', 'data-platform': props.platform }, props.groups.map((group: any) => group.name).join(','))
            },
          },
        },
      },
    })

    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.editBoundGroups'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const selector = wrapper.find('.group-selector')
    expect(selector.exists()).toBe(true)
    expect(selector.attributes('data-platform')).toBe('anthropic')
    expect(selector.text()).toContain('Claude')
    expect(selector.text()).not.toContain('VIP')
    expect(selector.text()).not.toContain('Trial')
  })

  it('opens the account test modal from the action column', async () => {
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
                slots['cell-test_status']?.({ row }),
                slots['cell-actions']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          AccountTestModal: {
            props: ['show'],
            emits: ['close'],
            template: '<div v-if="show" class="account-test-modal"><slot /></div>',
          },
        },
      },
    })

    await flushPromises()

    const testButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamAccounts.testConnection'))
    expect(testButton).toBeTruthy()
    await testButton!.trigger('click')
    await flushPromises()

    expect(accountsMock.getById).toHaveBeenCalledWith(12)
    expect(wrapper.find('.account-test-modal').exists()).toBe(true)
  })

  it('toggles schedulable state from the upstream table', async () => {
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
    accountsMock.getById.mockResolvedValueOnce({
      id: 12,
      name: 'local-a',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [7],
      groups: [],
    })
    accountsMock.setSchedulable = vi.fn().mockResolvedValue({
      id: 12,
      name: 'local-a',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: false,
      group_ids: [7],
      groups: [],
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
                slots['cell-schedulable']?.({ row }),
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

    const toggle = wrapper.find('button.schedulable-toggle')
    expect(toggle.exists()).toBe(true)
    await toggle.trigger('click')
    await flushPromises()

    expect(accountsMock.setSchedulable).toHaveBeenCalledWith(12, false)
  })

  it('opens temp unsched modal from the status indicator', async () => {
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
    accountsMock.getById.mockResolvedValueOnce({
      id: 12,
      name: 'local-a',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      temp_unschedulable_until: '2026-06-16T00:00:00Z',
      group_ids: [7],
      groups: [],
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
                slots['cell-status']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          AccountStatusIndicator: {
            emits: ['show-temp-unsched'],
            template: '<button class="account-status-indicator" @click="$emit(\'show-temp-unsched\', { id: 12, name: \'local-a\', schedulable: true, status: \'active\' })">status</button>',
          },
          TempUnschedStatusModal: {
            props: ['show'],
            emits: ['close'],
            template: '<div v-if="show" class="temp-unsched-modal"></div>',
          },
        },
      },
    })

    await flushPromises()

    await wrapper.find('.account-status-indicator').trigger('click')
    await flushPromises()

    expect(wrapper.find('.temp-unsched-modal').exists()).toBe(true)
  })

  it('shows account test status in the action column after test completion', async () => {
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
                slots['cell-test_status']?.({ row }),
                slots['cell-actions']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          AccountTestModal: {
            props: ['show'],
            emits: ['close', 'test-result'],
            setup(_, { emit }) {
              onMounted(() => {
                emit('test-result', { accountId: 12, status: 'success' })
              })
              return () => h('div', { class: 'account-test-modal' })
            },
          },
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('admin.upstreamAccounts.testStatusSuccess')
  })

  it('restores the last final account test status from the matched account response', async () => {
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
    accountsMock.getById.mockResolvedValueOnce({
      id: 12,
      name: 'local-a',
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      group_ids: [7],
      groups: [],
      last_test_status: 'failed',
      last_tested_at: '2026-06-16T00:00:00Z',
      last_test_error: 'upstream failed',
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
                slots['cell-test_status']?.({ row }),
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

    expect(wrapper.text()).toContain('admin.upstreamAccounts.testStatusFailed')
  })

  it('renders and refreshes the last account test time', async () => {
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
    accountsMock.getById
      .mockResolvedValueOnce({
        id: 12,
        name: 'local-a',
        platform: 'openai',
        type: 'apikey',
        status: 'active',
        schedulable: true,
        group_ids: [7],
        groups: [],
        last_test_status: 'failed',
        last_tested_at: '2026-06-16T00:00:00Z',
      })
      .mockResolvedValueOnce({
        id: 12,
        name: 'local-a',
        platform: 'openai',
        type: 'apikey',
        status: 'active',
        schedulable: true,
        group_ids: [7],
        groups: [],
        last_test_status: 'success',
        last_tested_at: '2026-06-17T00:00:00Z',
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
                slots['cell-last_tested_at']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          AccountTestModal: {
            emits: ['test-result'],
            setup(_, { emit }) {
              onMounted(() => {
                emit('test-result', { accountId: 12, status: 'success' })
              })
              return () => h('div')
            },
          },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(accountsMock.getById).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('06/17/2026')
  })

  it('opens edit account modal from upstream account actions and applies updates', async () => {
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
                slots['cell-last_tested_at']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          EditAccountModal: {
            props: ['show', 'account'],
            emits: ['updated', 'close'],
            setup(props, { emit }) {
              return () => props.show
                ? h('button', {
                  class: 'edit-account-modal',
                  onClick: () => emit('updated', { ...props.account, id: 12, name: 'local-a-updated', last_tested_at: '2026-06-18T00:00:00Z' }),
                }, 'edit')
                : null
            },
          },
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('common.edit'))?.trigger('click')
    await flushPromises()

    expect(proxiesMock.getAll).toHaveBeenCalled()
    expect(groupsMock.getAll).toHaveBeenCalled()
    expect(wrapper.find('.edit-account-modal').exists()).toBe(true)

    await wrapper.find('.edit-account-modal').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('06/18/2026')
  })

  it('deletes matched account from upstream account actions after confirmation', async () => {
    upstreamAccountSyncMock.getPreview
      .mockResolvedValueOnce({
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
      .mockResolvedValueOnce({
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
            upstream_key_name: 'key-a',
            local_account_name: 'local-a',
            upstream_group_name: 'vip',
            upstream_rate_multiplier: 2,
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
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                slots['cell-actions']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          Select: true,
          ConfirmDialog: {
            props: ['show'],
            emits: ['confirm', 'cancel'],
            setup(props, { emit }) {
              return () => props.show ? h('button', { class: 'confirm-delete', onClick: () => emit('confirm') }, 'confirm') : null
            },
          },
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('common.delete'))?.trigger('click')
    await flushPromises()
    await wrapper.find('.confirm-delete').trigger('click')
    await flushPromises()
    await flushPromises()

    expect(accountsMock.delete).toHaveBeenCalledWith(12)
    expect(upstreamAccountSyncMock.getPreview).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-test="create-local-account-upstream-a-key-a"]').exists()).toBe(true)
  })

  it('configures platform models before batch tests and toggles schedulable in the result dialog', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValue({
      default_provider: {},
      providers: [
        { slug: 'upstream-a', name: 'Upstream A', enabled: true },
        { slug: 'upstream-b', name: 'Upstream B', enabled: true },
        { slug: 'upstream-c', name: 'Upstream C', enabled: false },
      ],
      summary: {
        upstream_key_count: 2,
        matched_account_count: 2,
        create_count: 0,
        update_count: 0,
        skip_count: 2,
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
          upstream_rate_multiplier: 2,
          rate_violation: false,
          bound_groups: [],
        },
        {
          action: 'noop',
          provider_slug: 'upstream-b',
          provider_name: 'Upstream B',
          upstream_key_name: 'key-b',
          local_account_name: 'local-b',
          matched_account_id: 13,
          matched_account_name: 'local-b',
          upstream_rate_multiplier: 1,
          rate_violation: false,
          bound_groups: [],
        },
        {
          action: 'noop',
          provider_slug: 'upstream-c',
          provider_name: 'Upstream C',
          upstream_key_name: 'key-c',
          local_account_name: 'local-c',
          matched_account_id: 14,
          matched_account_name: 'local-c',
          upstream_rate_multiplier: 1.5,
          rate_violation: false,
          bound_groups: [],
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockImplementation(async (id: number) => ({
      id,
      name: id === 12 ? 'local-a' : id === 13 ? 'local-b' : 'local-c',
      platform: id === 13 ? 'gemini' : 'openai',
      type: 'apikey',
      status: 'active',
      schedulable: id === 12,
      last_test_status: id === 13 ? 'failed' : 'success',
      last_tested_at: '2026-06-27T10:00:00Z',
    }))
    accountsMock.setSchedulable.mockResolvedValue({
      id: 13,
      name: 'local-b',
      platform: 'gemini',
      type: 'apikey',
      status: 'active',
      schedulable: true,
    })
    accountsMock.getAvailableModels.mockImplementation(async (id: number) => {
      if (id === 12) {
        return [{ id: 'gpt-4.1-mini', display_name: 'GPT 4.1 Mini' }]
      }
      return [{ id: 'gemini-2.5-flash', display_name: 'Gemini 2.5 Flash' }]
    })
    accountsMock.batchTestAccounts.mockResolvedValue({
      job_id: 'job-1',
      status: 'completed',
      total: 3,
      completed: 3,
      success: 2,
      failed: 1,
      results: [
        {
          account_id: 12,
          account_name: 'local-a',
          platform: 'openai',
          schedulable: true,
          status: 'success',
          latency_ms: 42,
          finished_at: '2026-06-27T10:00:00Z',
        },
        {
          account_id: 13,
          account_name: 'local-b',
          platform: 'gemini',
          schedulable: false,
          status: 'timeout',
          error_message: 'account test timed out',
          latency_ms: 90000,
          finished_at: '2026-06-27T10:01:30Z',
        },
        {
          account_id: 14,
          account_name: 'local-c',
          platform: 'openai',
          schedulable: false,
          status: 'success',
          latency_ms: 84,
          finished_at: '2026-06-27T10:02:00Z',
        },
      ],
    })

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: true,
          EmptyState: true,
          Icon: true,
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue', 'change'],
            setup(props, { emit, attrs }) {
              return () => h('button', {
                ...attrs,
                class: ['framework-select-stub', attrs.class],
                type: 'button',
                'data-options': JSON.stringify(props.options || []),
                onClick: () => {
                  const firstOption = (props.options || [])[0]
                  const value = firstOption?.value ?? ''
                  emit('update:modelValue', value)
                  emit('change', value, firstOption ?? null)
                },
              }, String(props.modelValue || ''))
            },
          },
          ConfirmDialog: true,
          AccountStatusIndicator: true,
          AccountTestModal: true,
          CreateAccountModal: true,
          EditAccountModal: {
            props: ['show', 'account'],
            emits: ['close', 'updated'],
            setup(props) {
              return () => props.show
                ? h('div', { class: 'edit-account-modal' }, props.account?.name || '')
                : null
            },
          },
          TempUnschedStatusModal: true,
          UpstreamProviderTrendModal: true,
        },
      },
    })

    await flushPromises()
    await wrapper.find('[data-test="batch-test-accounts"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="batch-test-config-dialog"]').exists()).toBe(true)
    const openAISelect = wrapper.find('[data-test="batch-test-model-openai"]')
    const geminiSelect = wrapper.find('[data-test="batch-test-model-gemini"]')
    expect(openAISelect.classes()).toContain('framework-select-stub')
    expect(geminiSelect.classes()).toContain('framework-select-stub')

    await openAISelect.trigger('click')
    await geminiSelect.trigger('click')
    await wrapper.find('[data-test="batch-test-config-submit"]').trigger('click')
    await flushPromises()
    await flushPromises()

    expect(accountsMock.batchTestAccounts).toHaveBeenCalledWith({
      account_ids: [12, 13, 14],
      model_ids_by_platform: {
        openai: 'gpt-4.1-mini',
        gemini: 'gemini-2.5-flash',
      },
      concurrency: 3,
      timeout_per_account_seconds: 90,
      timeout_seconds: 600,
    })
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('local-a')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('local-b')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('local-c')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('2.00x')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('1.00x')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('admin.upstreamAccounts.batchTestSchedulableEnabled')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('admin.upstreamAccounts.batchTestSchedulableDisabled')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('account test timed out')
    expect(wrapper.find('[data-test="batch-test-filter-failed_schedulable"]').text()).toContain('0')
    expect(wrapper.find('[data-test="batch-test-filter-failed_unschedulable"]').text()).toContain('1')
    expect(wrapper.find('[data-test="batch-test-filter-success_unschedulable"]').text()).toContain('0')
    expect(wrapper.find('[data-test="batch-test-filter-success_upstream_disabled"]').text()).toContain('1')

    await wrapper.find('[data-test="batch-test-filter-success_unschedulable"]').trigger('click')
    await flushPromises()

    let dialogText = wrapper.find('[data-test="batch-test-result-dialog"]').text()
    expect(dialogText).toContain('admin.upstreamAccounts.batchTestNoFilteredResults')
    expect(dialogText).not.toContain('local-c')
    expect(dialogText).not.toContain('local-a')
    expect(dialogText).not.toContain('local-b')

    await wrapper.find('[data-test="batch-test-filter-success_upstream_disabled"]').trigger('click')
    await flushPromises()

    dialogText = wrapper.find('[data-test="batch-test-result-dialog"]').text()
    expect(dialogText).toContain('local-c')
    expect(dialogText).not.toContain('local-a')
    expect(dialogText).not.toContain('local-b')

    await wrapper.find('[data-test="batch-test-filter-failed_unschedulable"]').trigger('click')
    await flushPromises()

    dialogText = wrapper.find('[data-test="batch-test-result-dialog"]').text()
    expect(dialogText).toContain('local-b')
    expect(dialogText).not.toContain('local-a')
    expect(dialogText).not.toContain('local-c')

    await wrapper.find('[data-test="batch-test-filter-all"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="batch-test-sort-upstream_rate"]').trigger('click')
    await flushPromises()

    dialogText = wrapper.find('[data-test="batch-test-result-dialog"]').text()
    expect(dialogText.indexOf('local-b')).toBeLessThan(dialogText.indexOf('local-a'))

    await wrapper.find('[data-test="batch-test-sort-upstream_rate"]').trigger('click')
    await flushPromises()

    dialogText = wrapper.find('[data-test="batch-test-result-dialog"]').text()
    expect(dialogText.indexOf('local-a')).toBeLessThan(dialogText.indexOf('local-b'))

    await wrapper.find('[data-test="batch-test-schedulable-toggle-13"]').trigger('click')
    await flushPromises()

    expect(accountsMock.setSchedulable).toHaveBeenCalledWith(13, true)
    expect(wrapper.find('[data-test="batch-test-filter-failed_schedulable"]').text()).toContain('1')
    expect(wrapper.find('[data-test="batch-test-filter-failed_unschedulable"]').text()).toContain('0')
    expect(wrapper.find('[data-test="batch-test-result-dialog"]').text()).toContain('admin.upstreamAccounts.batchTestFailedSchedulableTag')
    expect(wrapper.find('.batch-result-card.failed-schedulable').exists()).toBe(true)
    expect(wrapper.find('.batch-test-risk-row').exists()).toBe(true)

    await wrapper.find('[data-test="batch-test-filter-failed_schedulable"]').trigger('click')
    await flushPromises()

    dialogText = wrapper.find('[data-test="batch-test-result-dialog"]').text()
    expect(dialogText).toContain('local-b')
    expect(dialogText).not.toContain('local-a')

    await wrapper.find('[data-test="batch-test-filter-all"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="batch-test-edit-account-12"]').trigger('click')
    await flushPromises()

    expect(proxiesMock.getAll).toHaveBeenCalled()
    expect(groupsMock.getAll).toHaveBeenCalled()
    expect(wrapper.find('.edit-account-modal').exists()).toBe(true)

    await wrapper.find('[data-test="batch-test-delete-account-13"]').trigger('click')
    await flushPromises()

    expect(wrapper.findComponent({ name: 'ConfirmDialog' }).exists()).toBe(true)
    expect(appStoreMock.showWarning).toHaveBeenCalled()
  })

  it('uses an API key account as the Anthropic batch-test model representative', async () => {
    upstreamAccountSyncMock.getPreview.mockResolvedValue({
      default_provider: {},
      providers: [],
      summary: {
        upstream_key_count: 2,
        matched_account_count: 2,
        create_count: 0,
        update_count: 0,
        skip_count: 2,
        conflict_count: 0,
        rate_violation_count: 0,
        unbound_group_count: 0,
      },
      items: [
        {
          action: 'noop',
          provider_slug: 'oauth-provider',
          provider_name: 'OAuth Provider',
          upstream_key_name: 'oauth-first',
          local_account_name: 'oauth-first',
          matched_account_id: 21,
          matched_account_name: 'oauth-first',
          upstream_rate_multiplier: 1,
          rate_violation: false,
          bound_groups: [],
        },
        {
          action: 'noop',
          provider_slug: 'upstream-provider',
          provider_name: 'Upstream Provider',
          upstream_key_name: 'apikey-second',
          local_account_name: 'apikey-second',
          matched_account_id: 22,
          matched_account_name: 'apikey-second',
          upstream_rate_multiplier: 1,
          rate_violation: false,
          bound_groups: [],
        },
      ],
      warnings: [],
      records: [],
    })
    accountsMock.getById.mockImplementation(async (id: number) => ({
      id,
      name: id === 21 ? 'oauth-first' : 'apikey-second',
      platform: 'anthropic',
      type: id === 21 ? 'oauth' : 'apikey',
      status: 'active',
      schedulable: true,
    }))
    accountsMock.getAvailableModels.mockResolvedValue([
      { id: 'claude-from-apikey', display_name: 'Claude From API Key' },
    ])

    const wrapper = mount(UpstreamAccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: true,
          EmptyState: true,
          Icon: true,
          Select: true,
          ConfirmDialog: true,
          AccountStatusIndicator: true,
          AccountTestModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          TempUnschedStatusModal: true,
          UpstreamProviderTrendModal: true,
        },
      },
    })

    await flushPromises()
    await wrapper.find('[data-test="batch-test-accounts"]').trigger('click')
    await flushPromises()

    expect(accountsMock.getAvailableModels).toHaveBeenCalledWith(22)
    expect(accountsMock.getAvailableModels).not.toHaveBeenCalledWith(21)
  })
})
