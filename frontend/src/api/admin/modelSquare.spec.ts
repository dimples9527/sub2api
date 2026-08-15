import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AdminGroup } from '@/types'
import type { Channel } from './channels'

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

const channel = (overrides: Partial<Channel> = {}): Channel => ({
  id: 1,
  name: 'Local channel',
  description: '',
  status: 'active',
  billing_model_source: 'channel_mapped',
  restrict_models: true,
  group_ids: [1],
  model_pricing: [{
    platform: 'openai',
    models: ['gpt-5.5', 'not-configured'],
    billing_mode: 'token',
    input_price: 0.000001,
    output_price: 0.000002,
    cache_write_price: 0.000003,
    cache_read_price: 0.0000005,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: []
  }],
  model_mapping: {},
  apply_pricing_to_account_stats: false,
  account_stats_pricing_rules: [],
  created_at: '',
  updated_at: '',
  ...overrides,
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
      if (path === '/admin/channels') return Promise.resolve({ data: { items: [channel()], total: 1 } })
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
    const result = buildConfiguredModelSquareResult(config, [channel()], [group(1)])
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
      [channel({ group_ids: [1] })],
      [group(1, 'openai', 2), group(2, 'openai', 0.5), group(3, 'anthropic', 0.1)]
    )
    const model = result.payload.models?.find(item => item.id === 'gpt-5.5')

    expect(model).toEqual(expect.objectContaining({
      rate_multiplier: 0.5,
      input_price: 2.5,
      output_price: 15,
      cache_write_price: 3,
      cache_read_price: 0.5,
      per_request_price: 0.06,
    }))
  })

  it('uses official reference prices only when the configured model has no value', async () => {
    getMock.mockImplementation((path: string, options?: { params?: { model?: string } }) => {
      if (path === '/admin/upstream-management/model-square/config') {
        return Promise.resolve({ data: { platforms: [{ platform: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5.5', input_price: 0 }] }] } })
      }
      if (path === '/admin/channels') return Promise.resolve({ data: { items: [channel()], total: 1 } })
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
    }, [], [])
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

  it('exposes the platform multiplier even when the configured model has no channel group', () => {
    const result = buildConfiguredModelSquareResult(
      { platforms: [{ platform: 'openai', name: 'OpenAI', models: [{ id: 'not-bound', input_price: 0.000005 }] }] },
      [],
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

  it('matches custom platform channels and groups like built-in platforms', () => {
    const customPricing = { ...channel().model_pricing[0], platform: 'custom-platform', models: ['custom-model'] }
    const result = buildConfiguredModelSquareResult({
      platforms: [{ platform: 'custom-platform', name: 'Custom Platform', models: [{ id: 'custom-model' }] }],
    }, [channel({ group_ids: [4], model_pricing: [customPricing] })], [group(4, 'custom-platform', 1.2)])

    expect(result.payload.models).toEqual([expect.objectContaining({
      id: 'custom-model',
      provider: 'Custom Platform',
      platform: 'custom-platform',
      available: true,
      group_ids: [4],
    })])
    expect(result.payload.groups).toEqual([{ id: 4, name: 'Group 4', platform: 'custom-platform', rate_multiplier: 1.2 }])
  })

  it('exposes configured platform groups even when channel groups are missing', () => {
    const ghostPricing = { ...channel().model_pricing[0], models: ['ghost-model'] }
    const result = buildConfiguredModelSquareResult({
      platforms: [{ platform: 'openai', name: 'OpenAI', models: [{ id: 'ghost-model' }] }],
    }, [channel({ group_ids: [99], model_pricing: [ghostPricing] })], [group(1)])

    expect(result.payload.models).toEqual([expect.objectContaining({
      id: 'ghost-model',
      available: false,
      group_ids: [],
    })])
    expect(result.payload.groups).toEqual([{ id: 1, name: 'Group 1', platform: 'openai', rate_multiplier: 1 }])
  })

  it('matches configured ids against channel pricing and mappings and only keeps compatible groups', () => {
    const result = buildConfiguredModelSquareResult({
      platforms: [{ platform: 'openai', name: 'OpenAI', models: [{ id: 'mapped-model' }] }],
    }, [channel({
      status: 'disabled',
      group_ids: [1, 2],
      model_pricing: [],
      model_mapping: { openai: { 'mapped-model': 'upstream-model' } },
    }), channel({
      id: 2,
      group_ids: [3],
      model_mapping: { openai: { 'other-model': 'mapped-model' } },
    })], [group(1), group(2, 'anthropic'), group(3, 'composite', 0.8)])

    expect(result.payload.models).toEqual([expect.objectContaining({
      id: 'mapped-model',
      available: true,
      group_ids: [1, 3],
    })])
    expect(result.payload.groups).toEqual([
      { id: 1, name: 'Group 1', platform: 'openai', rate_multiplier: 1 },
      { id: 3, name: 'Group 3', platform: 'composite', rate_multiplier: 0.8 },
    ])
  })

  it('按分组平台配置的覆盖平台过滤分组归属与最低倍率', () => {
    const glmPricing = { ...channel().model_pricing[0], platform: 'glm', models: ['glm-4.5'] }
    const result = buildConfiguredModelSquareResult(
      {
        platforms: [
          { platform: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5.5' }] },
          { platform: 'glm', name: 'GLM', models: [{ id: 'glm-4.5' }] },
        ],
      },
      [channel({ group_ids: [1, 2], model_pricing: [channel().model_pricing[0], glmPricing] })],
      [group(1, 'openai', 0.3), group(2, 'openai', 0.8)],
      new Map(),
      new Map([['1', 'glm']])
    )

    const openaiModel = result.payload.models?.find(model => model.id === 'gpt-5.5')
    const glmModel = result.payload.models?.find(model => model.id === 'glm-4.5')

    expect(openaiModel?.group_ids).toEqual([2])
    expect(openaiModel?.rate_multiplier).toBe(0.8)
    expect(glmModel?.group_ids).toEqual([1])
    expect(glmModel?.rate_multiplier).toBe(0.3)
    expect(result.payload.groups).toEqual([
      { id: 1, name: 'Group 1', platform: 'glm', rate_multiplier: 0.3 },
      { id: 2, name: 'Group 2', platform: 'openai', rate_multiplier: 0.8 },
    ])
  })

  it('从分组平台配置接口加载覆盖平台并应用到模型广场', async () => {
    const glmConfig: ModelSquareConfigPayload = {
      platforms: [
        { platform: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5.5' }] },
        { platform: 'glm', name: 'GLM', models: [{ id: 'glm-4.5' }] },
      ],
    }
    const glmPricing = { ...channel().model_pricing[0], platform: 'glm', models: ['glm-4.5'] }
    getMock.mockImplementation((path: string) => {
      if (path === '/admin/upstream-management/model-square/config') return Promise.resolve({ data: glmConfig })
      if (path === '/admin/channels') return Promise.resolve({ data: { items: [channel({ group_ids: [1], model_pricing: [channel().model_pricing[0], glmPricing] })], total: 1 } })
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
    expect(openaiModel?.group_ids).toEqual([])
    expect(glmModel?.group_ids).toEqual([1])
    expect(result.payload.groups).toEqual([{ id: 1, name: 'Group 1', platform: 'glm', rate_multiplier: 0.3 }])
  })
})
