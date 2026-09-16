import { describe, expect, it, vi, beforeEach } from 'vitest'
import { apiClient, buildGatewayUrl } from '@/api/client'
import {
  buildGroupBusinessPlatformMap,
  loadGroupBusinessPlatformMap,
} from './groupBusinessPlatformData'

vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn(),
  },
  buildGatewayUrl: vi.fn((path: string) => `gateway:${path}`),
}))

const apiGetMock = vi.mocked(apiClient.get)
const buildGatewayUrlMock = vi.mocked(buildGatewayUrl)

describe('groupBusinessPlatformData', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('构建分组业务平台映射表，跳过无效 ID 并保留解析后的平台字段', () => {
    const map = buildGroupBusinessPlatformMap([
      {
        id: 1,
        platform: 'openai',
        actual_platform: 'openai',
        effective_platform: 'openai',
        effective_platform_name: 'OpenAI',
      },
      {
        id: 2,
        platform: 'composite',
        actual_platform: 'glm',
        effective_platform: 'glm',
        effective_platform_name: '智谱 GLM',
      },
      {
        id: 0,
        platform: 'gemini',
        effective_platform: 'gemini',
      },
    ])

    expect(map.get(1)).toEqual({
      businessPlatform: 'openai',
      businessPlatformName: 'OpenAI',
      effectivePlatform: 'openai',
      actualPlatform: 'openai',
    })
    expect(map.get(2)).toEqual({
      businessPlatform: 'glm',
      businessPlatformName: '智谱 GLM',
      effectivePlatform: 'glm',
      actualPlatform: 'glm',
    })
    expect(map.has(0)).toBe(false)
  })

  it('通过网关接口加载分组业务平台映射，供 API Key 表单使用', async () => {
    apiGetMock.mockResolvedValueOnce({
      data: [
        {
          id: 9,
          platform: 'composite',
          actual_platform: 'custom-foo',
          effective_platform: 'custom-foo',
          effective_platform_name: '自定义 Foo',
        },
      ],
    } as never)

    const map = await loadGroupBusinessPlatformMap()

    expect(buildGatewayUrlMock).toHaveBeenCalledWith('/api/llm-monitor/groups')
    expect(apiGetMock).toHaveBeenCalledWith('gateway:/api/llm-monitor/groups')
    expect(map.get(9)).toEqual({
      businessPlatform: 'custom-foo',
      businessPlatformName: '自定义 Foo',
      effectivePlatform: 'custom-foo',
      actualPlatform: 'custom-foo',
    })
  })

  it('网关接口请求失败时降级为空映射，并记录 API 错误', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    apiGetMock.mockRejectedValueOnce(new Error('network error'))

    await expect(loadGroupBusinessPlatformMap()).resolves.toEqual(new Map())
    expect(consoleSpy).toHaveBeenCalledWith(
      'Failed to load group business platform data:',
      expect.any(Error)
    )

    consoleSpy.mockRestore()
  })
})
