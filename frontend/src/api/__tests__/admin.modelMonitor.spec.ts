import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from '../client'
import modelMonitorAPI from '../admin/modelMonitor'

vi.mock('../client', () => ({
  apiClient: {
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

const mockedClient = vi.mocked(apiClient)

describe('admin model monitor API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('获取模型监控分组平台配置', async () => {
    mockedClient.get.mockResolvedValueOnce({ data: [] })

    await modelMonitorAPI.listLLMMonitorGroupPlatformOverrides()

    expect(mockedClient.get).toHaveBeenCalledWith('/admin/model-monitor/platform-overrides')
  })

  it('设置分组实际平台', async () => {
    mockedClient.put.mockResolvedValueOnce({ data: { message: 'ok' } })

    await modelMonitorAPI.setLLMMonitorGroupPlatformOverride(12, 'anthropic')

    expect(mockedClient.put).toHaveBeenCalledWith('/admin/model-monitor/platform-overrides/12', {
      actual_platform: 'anthropic',
    })
  })

  it('清除分组实际平台', async () => {
    mockedClient.delete.mockResolvedValueOnce({ data: { message: 'ok' } })

    await modelMonitorAPI.clearLLMMonitorGroupPlatformOverride(12)

    expect(mockedClient.delete).toHaveBeenCalledWith('/admin/model-monitor/platform-overrides/12')
  })
})
