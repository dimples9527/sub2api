import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UpstreamDashboardView from './UpstreamDashboardView.vue'

const { getMock, runHealthGuardNowMock, pushMock, showErrorMock, showSuccessMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  runHealthGuardNowMock: vi.fn(),
  pushMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    upstreamDashboard: { get: getMock },
    upstreamAccountSync: { runHealthGuardNow: runHealthGuardNowMock },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: showErrorMock, showSuccess: showSuccessMock }),
}))

const response = {
  range: '24h',
  summary: {
    provider_count: 6,
    disabled_provider_count: 1,
    matched_account_count: 128,
    pending_account_count: 7,
    rate_risk_count: 3,
    model_count: 86,
  },
  stability: {
    request_count: 1200000,
    success_count: 1180000,
    error_count: 20000,
    success_rate: 98.7,
    error_rate: 1.3,
    p95_latency_ms: 842,
    health_score: 92,
  },
  cost: { period_cost: 386, total_balance: 4820, estimated_days: 12.4, anomaly_providers: 1 },
  issues: [{
    id: 'rate-risk', type: 'group_rate_risk', source: 'groups', severity: 'high', entity_key: 'premium',
    title: '倍率风险', description: '3 个分组低于上游成本', impact_count: 26,
    action: 'preview_rate_fix', target_path: '/admin/upstream-management/groups?rateRisk=true', detected_at: '2026-07-13T08:00:00Z',
  }],
  tasks: [{ key: 'health_guard', name: '健康巡检', enabled: true, last_run_status: 'success', affected_count: 2, settings_path: '/admin/upstream-management/providers' }],
  provider_rankings: [{ provider_slug: 'main', provider_name: 'Main', balance: 4000, cost: 300 }],
  model_rankings: [{ model: 'gpt-5', requests: 700000, cost: 220 }],
  trends: [{ date: '2026-07-13', cost: 386 }],
  warnings: [],
  generated_at: '2026-07-13T08:00:00Z',
}

describe('UpstreamDashboardView', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    getMock.mockReset()
    getMock.mockResolvedValue(response)
    runHealthGuardNowMock.mockReset()
    runHealthGuardNowMock.mockResolvedValue({ config: {}, record: {} })
    pushMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
  })

  it('renders operational metrics and opens an issue drawer', async () => {
    const wrapper = mount(UpstreamDashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: { template: '<span />' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-test="metric-resources"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="metric-stability"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="metric-cost"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('倍率风险')
    expect(wrapper.text()).toContain('gpt-5')

    await wrapper.get('[data-test="issue-rate-risk"]').trigger('click')
    await flushPromises()
    expect(document.body.querySelector('[data-test="issue-drawer"]')).not.toBeNull()

    const primaryAction = document.body.querySelector('[data-test="issue-primary-action"]') as HTMLButtonElement
    primaryAction.click()
    await flushPromises()
    expect(pushMock).toHaveBeenCalledWith('/admin/upstream-management/groups?rateRisk=true')
  })

  it('reloads when the range changes', async () => {
    const wrapper = mount(UpstreamDashboardView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: { template: '<span />' } } },
    })
    await flushPromises()
    await wrapper.get('[data-test="range-7d"]').trigger('click')
    await flushPromises()

    expect(getMock).toHaveBeenLastCalledWith('7d')
  })

  it('runs the health guard from the dashboard and refreshes the data', async () => {
    const wrapper = mount(UpstreamDashboardView, {
      global: { stubs: { AppLayout: { template: '<main><slot /></main>' }, Icon: { template: '<span />' } } },
    })
    await flushPromises()
    const callsBeforeRun = getMock.mock.calls.length

    await wrapper.get('[data-test="run-health-guard"]').trigger('click')
    await flushPromises()

    expect(runHealthGuardNowMock).toHaveBeenCalledTimes(1)
    expect(getMock.mock.calls.length).toBeGreaterThan(callsBeforeRun)
    expect(showSuccessMock).toHaveBeenCalledWith('admin.upstreamDashboard.inspectSuccess')
  })
})
