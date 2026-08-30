import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../client', () => ({
  apiClient: { get: getMock },
}))

import {
  getModelMonitorGroupHealth,
  type ModelMonitorGroupHealthResponse,
} from './modelMonitorGroupHealth'

describe('modelMonitorGroupHealth API', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockResolvedValue({ data: [] satisfies ModelMonitorGroupHealthResponse })
  })

  it('按时间范围请求分组健康趋势并省略空筛选条件', async () => {
    const response = [{
      group_id: 7,
      group_name: '主力分组',
      platform: 'openai',
      effective_platform: 'openai',
      request_count: 12,
      success_count: 11,
      error_count: 1,
      business_limited_count: 0,
      service_error_count: 1,
      success_rate: 91.67,
      service_success_rate: 91.67,
      error_rate: 8.33,
      avg_latency_ms: 840,
      p95_latency_ms: 1320,
      p95_first_token_ms: 510,
      status: 'warning',
      last_request_at: '2026-08-28T10:00:00Z',
      trend: [],
      top_errors: [],
    }]
    getMock.mockResolvedValueOnce({ data: response })

    await expect(getModelMonitorGroupHealth({ range: '24h' })).resolves.toEqual(response)
    expect(getMock).toHaveBeenCalledWith('/admin/model-monitor/group-health', {
      params: { range: '24h' },
    })
  })

  it('传递分组和平台筛选条件', async () => {
    await getModelMonitorGroupHealth({ range: '7d', groupIds: [3, 8], platform: 'gemini' })

    expect(getMock).toHaveBeenCalledWith('/admin/model-monitor/group-health', {
      params: { range: '7d', group_ids: '3,8', platform: 'gemini' },
    })
  })
})
