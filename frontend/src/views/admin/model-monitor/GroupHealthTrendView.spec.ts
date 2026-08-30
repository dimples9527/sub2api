import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getHealthMock, getAllGroupsMock, getPlatformOverridesMock, showErrorMock, showSuccessMock, extractErrorMock } = vi.hoisted(() => ({
  getHealthMock: vi.fn(),
  getAllGroupsMock: vi.fn(),
  getPlatformOverridesMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
  extractErrorMock: vi.fn((error: unknown, fallback: string) => error instanceof Error ? error.message : fallback),
}))

vi.mock('@/api/admin/modelMonitorGroupHealth', () => ({
  getModelMonitorGroupHealth: getHealthMock,
}))

vi.mock('@/api/admin/groups', () => ({
  getAll: getAllGroupsMock,
  default: { getAll: getAllGroupsMock },
}))

vi.mock('@/api/admin/modelMonitor', () => ({
  listLLMMonitorGroupPlatformOverrides: getPlatformOverridesMock,
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot /></div>' } }))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: showErrorMock, showSuccess: showSuccessMock }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: extractErrorMock,
}))

import GroupHealthTrendView from './GroupHealthTrendView.vue'

const healthItem = {
  group_id: 7,
  group_name: '主力分组',
  platform: 'openai',
  effective_platform: 'openai',
  request_count: 120,
  success_count: 116,
  error_count: 4,
  business_limited_count: 2,
  service_error_count: 2,
  success_rate: 96.67,
  service_success_rate: 98.31,
  error_rate: 3.33,
  avg_latency_ms: 840,
  p95_latency_ms: 1320,
  p95_first_token_ms: 510,
  status: 'healthy',
  last_request_at: '2026-08-28T10:00:00Z',
  trend: [{
    time: '2026-08-28T09:00:00Z',
    request_count: 60,
    success_count: 58,
    error_count: 2,
    service_error_count: 1,
    business_limited_count: 1,
    success_rate: 96.67,
    service_success_rate: 98.31,
    avg_latency_ms: 800,
    p95_latency_ms: 1200,
  }],
  top_errors: [{ category: 'network_timeout', count: 2 }],
}

function mountView() {
  return mount(GroupHealthTrendView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Select: {
          props: ['modelValue', 'options'],
          template: '<select><option v-for="option in options" :key="String(option.value)">{{ option.label }}</option></select>',
        },
        GroupHealthSuccessTrendChart: { template: '<div data-test="success-chart" />' },
        GroupHealthLatencyTrendChart: { template: '<div data-test="latency-chart" />' },
        GroupHealthErrorDistributionChart: { template: '<div data-test="error-chart" />' },
      },
    },
  })
}

describe('GroupHealthTrendView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getAllGroupsMock.mockResolvedValue([
      { id: 7, name: '主力分组', platform: 'openai' },
      { id: 8, name: '备用分组', platform: 'gemini' },
    ])
    getPlatformOverridesMock.mockResolvedValue([])
    getHealthMock.mockResolvedValue([healthItem])
  })

  it('加载并展示汇总指标、分组状态和趋势图', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(getHealthMock).toHaveBeenCalledWith({ range: '24h', groupIds: [], platform: undefined })
    expect(wrapper.find('.gh-hero').exists()).toBe(false)
    expect(wrapper.text()).toContain('服务健康成功率')
    expect(wrapper.text()).toContain('主力分组')
    expect(wrapper.text()).toContain('98.31%')
    expect(wrapper.find('[data-test="success-chart"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="latency-chart"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="error-chart"]').exists()).toBe(true)
  })

  it('切换时间范围后重新加载数据', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="range-7d"]').trigger('click')
    await flushPromises()

    expect(getHealthMock).toHaveBeenLastCalledWith({ range: '7d', groupIds: [], platform: undefined })
  })

  it('加载失败时通过统一 Toast 提示错误', async () => {
    const failure = new Error('接口不可用')
    getHealthMock.mockRejectedValueOnce(failure)
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('加载分组健康趋势失败')
    expect(showErrorMock).toHaveBeenCalledWith('接口不可用')
  })

  it('健康接口失败但分组接口成功时仍展示分组状态', async () => {
    getHealthMock.mockRejectedValueOnce(new Error('健康接口暂不可用'))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('主力分组')
    expect(wrapper.text()).toContain('无数据')
    expect(wrapper.text()).toContain('加载分组健康趋势失败')
  })

  it('按模型监控实际平台构建平台筛选项', async () => {
    getPlatformOverridesMock.mockResolvedValueOnce([
      {
        id: 7,
        name: '主力分组',
        platform: 'openai',
        actual_platform: 'glm',
        effective_platform: 'glm',
        effective_platform_name: '智谱 GLM',
        rate_multiplier: 1,
        show_in_monitor: true,
      },
    ])

    const wrapper = mountView()
    await flushPromises()

    const platformSelect = wrapper.findAll('select')[0]
    expect(platformSelect.text()).toContain('智谱 GLM')
    expect(platformSelect.text()).not.toContain('OpenAI')
  })

  it('没有趋势记录时仍展示活跃分组并标记为无数据', async () => {
    getHealthMock.mockResolvedValueOnce([])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('主力分组')
    expect(wrapper.text()).toContain('无数据')
    expect(wrapper.text()).toContain('当前范围暂无请求记录')
  })

  it('辅助信息接口失败时不影响健康数据展示', async () => {
    getPlatformOverridesMock.mockRejectedValueOnce(new Error('平台配置暂不可用'))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('主力分组')
    expect(wrapper.text()).not.toContain('加载分组健康趋势失败')
    expect(showErrorMock).not.toHaveBeenCalled()
  })
})
