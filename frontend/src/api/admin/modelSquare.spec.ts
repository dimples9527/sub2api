import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AdminGroup } from '@/types'
import type { Channel } from './channels'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../client', () => ({
  apiClient: { get: getMock }
}))

import { buildLocalModelSquareResult, getModelSquare } from './modelSquare'

const group = (id: number, name: string, rate_multiplier: number, overrides: Partial<AdminGroup> = {}): AdminGroup => ({
  id,
  name,
  platform: 'openai',
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
  ...overrides
})

const channel = (overrides: Partial<Channel> = {}): Channel => ({
  id: 1,
  name: '本地渠道',
  description: '',
  status: 'active',
  billing_model_source: 'channel_mapped',
  restrict_models: true,
  group_ids: [1],
  model_pricing: [{
    platform: 'openai',
    models: ['gpt-local'],
    billing_mode: 'token',
    input_price: 0.000001,
    output_price: 0.000002,
    cache_write_price: null,
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
  ...overrides
})

describe('admin model square API', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockImplementation((path: string) => {
      if (path === '/admin/channels') return Promise.resolve({ data: { items: [channel()], total: 1 } })
      if (path === '/admin/groups/all') return Promise.resolve({ data: [group(1, 'default', 1)] })
      if (path === '/admin/groups/1/models-list-candidates') return Promise.resolve({ data: { models: ['gpt-local'] } })
      return Promise.resolve({ data: [] })
    })
  })

  it('loads local channels and groups instead of the upstream model square endpoint', async () => {
    await expect(getModelSquare()).resolves.toMatchObject({
      provider_slug: 'local',
      payload: {
        groups: [{ id: 1, name: 'default', rate_multiplier: 1 }],
        models: [{
          id: 'gpt-local',
          provider: 'openai',
          input_price: 1,
          output_price: 2,
          cache_read_price: 0.5,
          group_ids: [1]
        }]
      }
    })
    expect(getMock).toHaveBeenCalledWith('/admin/channels', expect.objectContaining({
      params: { page: 1, page_size: 1000 }
    }))
    expect(getMock).toHaveBeenCalledWith('/admin/groups/all', {
      params: { include_inactive: true }
    })
    expect(getMock).toHaveBeenCalledWith('/admin/groups/1/models-list-candidates', {
      params: { platform: 'openai' }
    })
  })



  it('prefers configured group models and falls back to account candidates', async () => {
    getMock.mockImplementation((path: string) => {
      if (path === '/admin/channels') {
        return Promise.resolve({ data: { items: [
          channel({
            group_ids: [1],
            model_pricing: [{
              platform: 'openai',
              models: ['gpt-configured', 'gpt-hidden'],
              billing_mode: 'token',
              input_price: 0.000001,
              output_price: 0.000002,
              cache_write_price: null,
              cache_read_price: null,
              image_input_price: null,
              image_output_price: null,
              per_request_price: null,
              intervals: []
            }]
          }),
          channel({
            id: 2,
            group_ids: [2],
            model_pricing: [{
              platform: 'openai',
              models: ['gpt-fallback', 'gpt-missing'],
              billing_mode: 'token',
              input_price: 0.000003,
              output_price: 0.000004,
              cache_write_price: null,
              cache_read_price: null,
              image_input_price: null,
              image_output_price: null,
              per_request_price: null,
              intervals: []
            }]
          })
        ], total: 2 } })
      }
      if (path === '/admin/groups/all') {
        return Promise.resolve({ data: [
          group(1, 'configured', 1, { models_list_config: { enabled: true, models: ['gpt-configured'] } }),
          group(2, 'candidate', 1)
        ] })
      }
      if (path === '/admin/groups/2/models-list-candidates') {
        return Promise.resolve({ data: { models: ['gpt-fallback'] } })
      }
      return Promise.resolve({ data: [] })
    })

    const result = await getModelSquare()
    expect(result.payload.models?.map(model => model.id)).toEqual([
      'gpt-configured',
      'gpt-fallback'
    ])
    expect(getMock).not.toHaveBeenCalledWith('/admin/groups/1/models-list-candidates', expect.anything())
    expect(getMock).toHaveBeenCalledWith('/admin/groups/2/models-list-candidates', {
      params: { platform: 'openai' }
    })
  })
  it('keeps channel models when candidate lookup fails', async () => {
    getMock.mockImplementation((path: string) => {
      if (path === '/admin/channels') return Promise.resolve({ data: { items: [channel()], total: 1 } })
      if (path === '/admin/groups/all') return Promise.resolve({ data: [group(1, 'default', 1)] })
      if (path === '/admin/groups/1/models-list-candidates') return Promise.reject(new Error('unavailable'))
      return Promise.resolve({ data: [] })
    })

    const result = await getModelSquare()
    expect(result.payload.models?.map(model => model.id)).toEqual(['gpt-local'])
  })

  it('deduplicates models across channels and keeps the price from the lowest-rate active group', () => {
    const result = buildLocalModelSquareResult([
      channel({ group_ids: [10], model_pricing: [{
        platform: 'openai',
        models: ['same-model'],
        billing_mode: 'token',
        input_price: null,
        output_price: null,
        cache_write_price: null,
        cache_read_price: null,
        image_input_price: null,
        image_output_price: null,
        per_request_price: null,
        intervals: []
      }] }),
      channel({
        id: 2,
        status: 'active',
        group_ids: [11],
        model_pricing: [{
          platform: 'openai',
          models: ['same-model'],
          billing_mode: 'token',
          input_price: 0.000003,
          output_price: 0.000004,
          cache_write_price: 0.000005,
          cache_read_price: 0.000001,
          image_input_price: null,
          image_output_price: null,
          per_request_price: null,
          intervals: []
        }]
      })
    ], [group(10, '高倍率组', 1.5), group(11, '低倍率组', 0.8)])

    expect(result.payload.models).toEqual([expect.objectContaining({
      id: 'same-model',
      input_price: 3,
      output_price: 4,
      cache_create_price: 5,
      cache_read_price: 1,
      group_ids: [10, 11]
    })])
  })
})
