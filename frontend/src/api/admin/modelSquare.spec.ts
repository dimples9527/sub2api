import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AdminGroup } from '@/types'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../client', () => ({
  apiClient: { get: getMock }
}))

import {
  buildConfiguredModelSquareResult,
  getModelSquare,
  type ModelSquareConfigPayload,
} from './modelSquare'

const group = (id: number, platform = 'openai', rate_multiplier = 1): AdminGroup => ({
  id,
  name: `Group ${id}`,
  platform: platform as AdminGroup['platform'],
  rate_multiplier,
  status: 'active',
  description: null,
  is_exclusive: false,
  subscription_type: 'standard',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  allow_batch_image_generation: false,
  image_rate_independent: false,
  batch_image_discount_multiplier: 1,
  batch_image_hold_multiplier: 1,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  video_rate_independent: false,
  video_rate_multiplier: 1,
  video_price_480p: null,
  video_price_720p: null,
  video_price_1080p: null,
  web_search_price_per_call: null,
  peak_rate_enabled: false,
  peak_start: '',
  peak_end: '',
  peak_rate_multiplier: 1,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '',
  updated_at: '',
  profit_control_enabled: false,
  profit_min_margin: 0,
  profit_safety_buffer: 0,
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: false,
  sort_order: id,
})

const config: ModelSquareConfigPayload = {
  platforms: [
    {
      platform: 'openai',
      name: 'OpenAI Official',
      models: [{
        id: 'gpt-5.5',
        display_name: 'GPT-5.5 Flagship',
        input_price: 0.000005,
        output_price: 0.00003,
        cache_write_price: 0.000006,
        cache_write_1h_price: 0.000007,
        cache_read_price: 0.000001,
        input_price_priority: 0.000008,
        output_price_priority: 0.00004,
        cache_write_price_priority: 0.000009,
        cache_read_price_priority: 0.000002,
        image_input_price: 0.00001,
        image_output_price: 0.00002,
        per_request_price: 0.12,
        group_ids: [1],
      }],
    },
    {
      platform: 'custom-platform',
      name: 'Custom Platform',
      models: [{ id: 'custom-model', display_name: 'Custom Model' }],
    },
  ],
}

describe('admin model square API', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockImplementation((path: string) => {
      if (path === '/admin/upstream-management/model-square/config') return Promise.resolve({ data: config })
      if (path === '/admin/groups/all') return Promise.resolve({ data: [group(1)] })
      if (path === '/admin/model-monitor/platform-overrides') return Promise.resolve({ data: [] })
      return Promise.resolve({ data: [] })
    })
  })

  it('uses configured platforms and models as the only model directory', async () => {
    const result = await getModelSquare()

    expect(result.payload.models).toEqual(expect.arrayContaining([
      expect.objectContaining({
        id: 'gpt-5.5',
        display_name: 'GPT-5.5 Flagship',
        provider: 'OpenAI Official',
        platform: 'openai',
        available: true,
      }),
      expect.objectContaining({
        id: 'custom-model',
        display_name: 'Custom Model',
        provider: 'Custom Platform',
        platform: 'custom-platform',
        available: false,
      }),
    ]))
    expect(result.payload.models?.map(model => model.id)).not.toContain('not-configured')
    expect(result.payload.groups).toEqual([{ id: 1, name: 'Group 1', platform: 'openai', rate_multiplier: 1 }])
    expect(getMock).not.toHaveBeenCalledWith('/admin/groups/1/models-list-candidates', expect.anything())
  })

  it('uses configured prices, converts token prices once, and leaves request prices per call', () => {
    const result = buildConfiguredModelSquareResult(config, [group(1)])
    const model = result.payload.models?.find(item => item.id === 'gpt-5.5')

    expect(model).toEqual(expect.objectContaining({
      input_price: 5,
      output_price: 30,
      cache_write_price: 6,
      cache_write_1h_price: 7,
      cache_read_price: 1,
      input_price_priority: 8,
      output_price_priority: 40,
      cache_write_price_priority: 9,
      cache_read_price_priority: 2,
      image_input_price: 10,
      image_output_price: 20,
      per_request_price: 0.12,
    }))
  })

  it('applies the lowest multiplier among the platform groups to displayed prices', () => {
    const result = buildConfiguredModelSquareResult(
      { platforms: [config.platforms[0]] },
      [group(1, 'openai', 2), group(2, 'openai', 0.5), group(3, 'anthropic', 0.1)]
    )
    const model = result.payload.models?.find(item => item.id === 'gpt-5.5')

    expect(model).toEqual(expect.objectContaining({
      rate_multiplier: 0.5,
      input_price: 2.5,
      output_price: 15,
      cache_write_price: 3,
      cache_read_price: 0.5,
      // 按张价不随倍率缩放，原样保留（token 价才乘 0.5）
      per_request_price: 0.12,
    }))
  })

  it('uses official reference prices only when the configured model has no value', async () => {
    getMock.mockImplementation((path: string, options?: { params?: { model?: string } }) => {
      if (path === '/admin/upstream-management/model-square/config') {
        return Promise.resolve({ data: { platforms: [{ platform: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5.5', input_price: 0 }] }] } })
      }
      if (path === '/admin/groups/all') return Promise.resolve({ data: [group(1)] })
      if (path === '/admin/upstream-management/model-square/model-pricing' && options?.params?.model === 'gpt-5.5') {
        return Promise.resolve({ data: { found: true, input_price: 0.000005, output_price: 0.00003, cache_read_price: 0.000001 } })
      }
      throw new Error(`unexpected request: ${path}`)
    })

    const result = await getModelSquare()
    const model = result.payload.models?.[0]

    expect(model).toMatchObject({ id: 'gpt-5.5', input_price: 0, output_price: 30, cache_read_price: 1 })
    expect(getMock).toHaveBeenCalledWith('/admin/upstream-management/model-square/model-pricing', {
      params: { model: 'gpt-5.5' },
    })
  })

  it('keeps zero prices and omits missing prices instead of producing zero values', () => {
    const result = buildConfiguredModelSquareResult({
      platforms: [{ platform: 'openai', name: 'OpenAI', models: [{ id: 'free-model', input_price: 0 }] }],
    }, [])
    const model = result.payload.models?.[0]

    expect(model).toMatchObject({ id: 'free-model', input_price: 0, available: false })
    expect(model).not.toHaveProperty('output_price')
    expect(model).not.toHaveProperty('per_request_price')
  })

  it('returns an empty payload for an empty configuration without falling back to channels', async () => {
    getMock.mockImplementation((path: string) => {
      if (path === '/admin/upstream-management/model-square/config') return Promise.resolve({ data: { platforms: [] } })
      throw new Error(`unexpected request: ${path}`)
    })

    const result = await getModelSquare()

    expect(result.payload).toEqual({ groups: [], models: [] })
    expect(getMock).toHaveBeenCalledTimes(1)
  })

  it('exposes the platform multiplier even when the model is not bound to any group', () => {
    const result = buildConfiguredModelSquareResult(
      { platforms: [{ platform: 'openai', name: 'OpenAI', models: [{ id: 'not-bound', input_price: 0.000005 }] }] },
      [group(1, 'openai', 0.25)]
    )

    expect(result.payload.models?.[0]).toEqual(expect.objectContaining({
      id: 'not-bound',
      available: false,
      rate_multiplier: 0.25,
      input_price: 1.25,
      group_ids: [],
    }))
  })

  it('matches custom platform groups like built-in platforms', () => {
    const result = buildConfiguredModelSquareResult({
      platforms: [{ platform: 'custom-platform', name: 'Custom Platform', models: [{ id: 'custom-model', group_ids: [4] }] }],
    }, [group(4, 'custom-platform', 1.2)])

    expect(result.payload.models).toEqual([expect.objectContaining({
      id: 'custom-model',
      provider: 'Custom Platform',
      platform: 'custom-platform',
      available: true,
      group_ids: [4],
    })])
    expect(result.payload.groups).toEqual([{ id: 4, name: 'Group 4', platform: 'custom-platform', rate_multiplier: 1.2 }])
  })

  it('derives availability from the configured group binding alone', () => {
    const result = buildConfiguredModelSquareResult({
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [
          { id: 'bound-model', group_ids: [1, 3] },
          { id: 'unbound-model' },
        ],
      }],
    }, [group(1), group(3, 'composite', 0.8)])

    /*
      可用性只由「有没有绑定分组」决定，与渠道、账号状态完全无关：绑了就是可用，没绑就不可用。
      分组归属照抄配置，不做任何反推。

      这里曾经断言「渠道提供定价/映射才算可用」—— 那套判据与真实调度路径不符
      （请求路由走分组→账号，渠道只管定价、映射与模型限制），会把有账号支撑的分组误判成不可用。
    */
    expect(result.payload.models).toEqual([
      expect.objectContaining({ id: 'bound-model', available: true, group_ids: [1, 3] }),
      expect.objectContaining({ id: 'unbound-model', available: false, group_ids: [] }),
    ])
    expect(result.payload.groups).toEqual([
      { id: 1, name: 'Group 1', platform: 'openai', rate_multiplier: 1 },
      { id: 3, name: 'Group 3', platform: 'composite', rate_multiplier: 0.8 },
    ])
  })

  it('按分组平台配置的覆盖平台过滤分组列表与最低倍率', () => {
    const result = buildConfiguredModelSquareResult(
      {
        platforms: [
          { platform: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5.5', group_ids: [2] }] },
          { platform: 'glm', name: 'GLM', models: [{ id: 'glm-4.5', group_ids: [1] }] },
        ],
      },
      [group(1, 'openai', 0.3), group(2, 'openai', 0.8)],
      new Map(),
      new Map([['1', 'glm']])
    )

    const openaiModel = result.payload.models?.find(model => model.id === 'gpt-5.5')
    const glmModel = result.payload.models?.find(model => model.id === 'glm-4.5')

    // 分组绑定直接取配置：平台覆盖不再改写它，只影响下面的分组列表与倍率口径。
    expect(openaiModel?.group_ids).toEqual([2])
    expect(glmModel?.group_ids).toEqual([1])
    // 最低倍率仍按「覆盖后的有效平台」算：分组 1 被覆盖成 glm，归 GLM 平台。
    expect(openaiModel?.rate_multiplier).toBe(0.8)
    expect(glmModel?.rate_multiplier).toBe(0.3)
    expect(result.payload.groups).toEqual([
      { id: 1, name: 'Group 1', platform: 'glm', rate_multiplier: 0.3 },
      { id: 2, name: 'Group 2', platform: 'openai', rate_multiplier: 0.8 },
    ])
  })

  it('从分组平台配置接口加载覆盖平台并应用到模型广场', async () => {
    const glmConfig: ModelSquareConfigPayload = {
      platforms: [
        { platform: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5.5', group_ids: [1] }] },
        { platform: 'glm', name: 'GLM', models: [{ id: 'glm-4.5' }] },
      ],
    }
    getMock.mockImplementation((path: string) => {
      if (path === '/admin/upstream-management/model-square/config') return Promise.resolve({ data: glmConfig })
      if (path === '/admin/groups/all') return Promise.resolve({ data: [group(1, 'openai', 0.3)] })
      if (path === '/admin/model-monitor/platform-overrides') {
        return Promise.resolve({ data: [{ id: 1, name: 'Group 1', platform: 'openai', actual_platform: 'glm', effective_platform: 'glm', effective_platform_name: 'GLM', rate_multiplier: 0.3, show_in_monitor: true }] })
      }
      throw new Error(`unexpected request: ${path}`)
    })

    const result = await getModelSquare()

    expect(getMock).toHaveBeenCalledWith('/admin/model-monitor/platform-overrides')
    const openaiModel = result.payload.models?.find(model => model.id === 'gpt-5.5')
    const glmModel = result.payload.models?.find(model => model.id === 'glm-4.5')

    // 绑定照抄配置，覆盖平台不改写它。
    expect(openaiModel?.group_ids).toEqual([1])
    expect(glmModel?.group_ids).toEqual([])
    /*
      可用性只看绑定：gpt-5.5 绑了分组所以可用，glm-4.5 没绑所以不可用。
      覆盖平台只影响分组列表与倍率，不再参与可用性判断。
    */
    expect(openaiModel?.available).toBe(true)
    expect(glmModel?.available).toBe(false)
    expect(result.payload.groups).toEqual([{ id: 1, name: 'Group 1', platform: 'glm', rate_multiplier: 0.3 }])
  })
})
