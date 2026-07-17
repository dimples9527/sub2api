import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UpstreamAutomationCenterView from './UpstreamAutomationCenterView.vue'

const { apiMock, pushMock, showErrorMock, showSuccessMock } = vi.hoisted(() => ({
  apiMock: {
    upstreamManagement: { getRateFixConfig: vi.fn(), getGroups: vi.fn() },
    upstreamAccountSync: {
      getRecords: vi.fn(),
      getRateGuardConfig: vi.fn(),
      getRateGuardPollLogs: vi.fn(),
      getBalanceSamplerConfig: vi.fn(),
      getBalanceSamplerPollLogs: vi.fn(),
      getHealthGuardConfig: vi.fn(),
      getHealthGuardPollLogs: vi.fn(),
      getHealthGuardRecords: vi.fn(),
      runRateGuardNow: vi.fn(),
      runBalanceSampleNow: vi.fn(),
      runHealthGuardNow: vi.fn(),
    },
  },
  pushMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('vue-router', () => ({ useRouter: () => ({ push: pushMock }) }))
vi.mock('@/api/admin', () => ({ adminAPI: apiMock }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: showErrorMock, showSuccess: showSuccessMock }) }))

describe('UpstreamAutomationCenterView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMock.upstreamManagement.getRateFixConfig.mockResolvedValue({ enabled: true, interval_seconds: 3600, last_run_status: 'success' })
    apiMock.upstreamManagement.getGroups.mockResolvedValue({
      items: [], warnings: [],
      records: [{ group_id: 7, group_name: 'VIP', provider_slug: 'main', provider_name: 'Main', upstream_group_name: 'VIP', old_rate: 1, new_rate: 2, changed_at: '2026-07-13T09:00:00Z' }],
    })
    apiMock.upstreamAccountSync.getRecords.mockResolvedValue([{ provider_slug: 'main', provider_name: 'Main', created_count: 1, updated_count: 2, skipped_count: 0, conflict_count: 0, rate_violation_count: 0, unbound_group_count: 0, created_at: '2026-07-13T08:00:00Z' }])
    apiMock.upstreamAccountSync.getRateGuardConfig.mockResolvedValue({ enabled: true, interval_seconds: 900, last_run_status: 'success', last_run_at: '2026-07-13T08:00:00Z' })
    apiMock.upstreamAccountSync.getRateGuardPollLogs.mockResolvedValue([{ checked_at: '2026-07-13T10:00:00Z', trigger: 'scheduled', status: 'failed', message: 'rate guard failed' }])
    apiMock.upstreamAccountSync.getBalanceSamplerConfig.mockResolvedValue({ enabled: true, interval_seconds: 1800, last_run_status: 'success', last_run_at: '2026-07-13T08:00:00Z' })
    apiMock.upstreamAccountSync.getBalanceSamplerPollLogs.mockResolvedValue([{ checked_at: '2026-07-13T11:00:00Z', trigger: 'manual', status: 'success', message: 'sampled' }])
    apiMock.upstreamAccountSync.getHealthGuardConfig.mockResolvedValue({ enabled: true, interval_seconds: 1800, last_run_status: 'success', last_run_at: '2026-07-13T08:00:00Z' })
    apiMock.upstreamAccountSync.getHealthGuardPollLogs.mockResolvedValue([])
    apiMock.upstreamAccountSync.getHealthGuardRecords.mockResolvedValue([{
      id: 'health-1', trigger: 'manual', status: 'failed', message: 'timeout', started_at: '2026-07-13T12:00:00Z', finished_at: '2026-07-13T12:01:00Z',
      summary: { total_accounts: 10, checked_count: 8, healthy_count: 5, slow_count: 1, failed_count: 2, skipped_count: 2, disabled_count: 1, recovered_count: 0, unchanged_count: 7 }, items: [],
    }])
    apiMock.upstreamAccountSync.runRateGuardNow.mockResolvedValue({ enabled: true, interval_seconds: 900, last_run_status: 'success' })
    apiMock.upstreamAccountSync.runBalanceSampleNow.mockResolvedValue({ enabled: true, interval_seconds: 1800, last_run_status: 'success' })
    apiMock.upstreamAccountSync.runHealthGuardNow.mockResolvedValue({ config: {}, record: {} })
  })

  it('renders all automation flows and keeps risky operations in their existing preview pages', async () => {
    const wrapper = mount(UpstreamAutomationCenterView, { global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: { template: '<span />' } } } })
    await flushPromises()

    expect(wrapper.findAll('[data-test^="automation-card-"]')).toHaveLength(5)
    await wrapper.get('[data-test="automation-action-account-sync"]').trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/admin/upstream-management/accounts')
    await wrapper.get('[data-test="automation-action-group-rate-fix"]').trigger('click')
    expect(pushMock).toHaveBeenCalledWith('/admin/upstream-management/groups?rateRisk=true')
  })

  it.each([
    ['account-rate-guard', 'runRateGuardNow'],
    ['balance-sampler', 'runBalanceSampleNow'],
    ['health-guard', 'runHealthGuardNow'],
  ])('runs %s directly and reloads statuses', async (taskKey, method) => {
    const wrapper = mount(UpstreamAutomationCenterView, { global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: { template: '<span />' } } } })
    await flushPromises()
    const configCallsBefore = apiMock.upstreamAccountSync.getRateGuardConfig.mock.calls.length + apiMock.upstreamAccountSync.getBalanceSamplerConfig.mock.calls.length + apiMock.upstreamAccountSync.getHealthGuardConfig.mock.calls.length

    await wrapper.get(`[data-test="automation-action-${taskKey}"]`).trigger('click')
    await flushPromises()

    expect(apiMock.upstreamAccountSync[method as keyof typeof apiMock.upstreamAccountSync]).toHaveBeenCalledTimes(1)
    const configCallsAfter = apiMock.upstreamAccountSync.getRateGuardConfig.mock.calls.length + apiMock.upstreamAccountSync.getBalanceSamplerConfig.mock.calls.length + apiMock.upstreamAccountSync.getHealthGuardConfig.mock.calls.length
    expect(configCallsAfter).toBeGreaterThan(configCallsBefore)
    expect(showSuccessMock).toHaveBeenCalled()
  })

  it('normalizes execution history from all task sources and sorts newest first', async () => {
    const wrapper = mount(UpstreamAutomationCenterView, { global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: { template: '<span />' } } } })
    await flushPromises()

    const rows = wrapper.findAll('[data-test^="history-row-"]')
    expect(rows.length).toBeGreaterThanOrEqual(5)
    expect(rows[0].text()).toContain('admin.upstreamAutomations.tasks.health-guard.name')
    expect(wrapper.text()).toContain('rate guard failed')
    expect(wrapper.text()).toContain('VIP')
  })

  it('filters execution history by task, status, and trigger', async () => {
    const wrapper = mount(UpstreamAutomationCenterView, { global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: { template: '<span />' } } } })
    await flushPromises()

    await wrapper.get('[data-test="history-task-filter"]').setValue('account-rate-guard')
    await wrapper.get('[data-test="history-status-filter"]').setValue('failed')
    await wrapper.get('[data-test="history-trigger-filter"]').setValue('scheduled')

    const rows = wrapper.findAll('[data-test^="history-row-"]')
    expect(rows).toHaveLength(1)
    expect(rows[0].text()).toContain('rate guard failed')
  })

  it('opens failed execution details and retries the matching safe task', async () => {
    const wrapper = mount(UpstreamAutomationCenterView, { global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: { template: '<span />' } } } })
    await flushPromises()

    await wrapper.get('[data-test="history-row-health-guard-health-1"]').trigger('click')
    expect(wrapper.get('[data-test="history-detail"]').text()).toContain('timeout')

    await wrapper.get('[data-test="history-retry"]').trigger('click')
    await flushPromises()
    expect(apiMock.upstreamAccountSync.runHealthGuardNow).toHaveBeenCalledTimes(1)
  })
})
