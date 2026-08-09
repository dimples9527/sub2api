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

  it('????????????????? ID ?????????', () => {
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
        effective_platform_name: '?? GLM',
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
      businessPlatformName: '?? GLM',
      effectivePlatform: 'glm',
      actualPlatform: 'glm',
    })
    expect(map.has(0)).toBe(false)
  })

  it('??????????????????????? API Key ???????', async () => {
    apiGetMock.mockResolvedValueOnce({
      data: [
        {
          id: 9,
          platform: 'composite',
          actual_platform: 'custom-foo',
          effective_platform: 'custom-foo',
          effective_platform_name: '??? Foo',
        },
      ],
    } as never)

    const map = await loadGroupBusinessPlatformMap()

    expect(buildGatewayUrlMock).toHaveBeenCalledWith('/api/llm-monitor/groups')
    expect(apiGetMock).toHaveBeenCalledWith('gateway:/api/llm-monitor/groups')
    expect(map.get(9)).toEqual({
      businessPlatform: 'custom-foo',
      businessPlatformName: '??? Foo',
      effectivePlatform: 'custom-foo',
      actualPlatform: 'custom-foo',
    })
  })

  it('??????????????API ?????????????', async () => {
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
