import { describe, expect, it } from 'vitest'
import type { ApiKey } from '@/types'
import {
  buildImageGenerationPayload,
  getImageCapableOpenAIKeys,
  parseImageGenerationResponse,
} from './imageGeneration'

function makeKey(overrides: Partial<ApiKey>): ApiKey {
  return {
    id: 1,
    user_id: 1,
    key: 'sk-test',
    name: 'OpenAI Key',
    group_id: 1,
    status: 'active',
    ip_whitelist: [],
    ip_blacklist: [],
    last_used_at: null,
    quota: 0,
    quota_used: 0,
    expires_at: null,
    created_at: '2026-06-29T00:00:00Z',
    updated_at: '2026-06-29T00:00:00Z',
    rate_limit_5h: 0,
    rate_limit_1d: 0,
    rate_limit_7d: 0,
    usage_5h: 0,
    usage_1d: 0,
    usage_7d: 0,
    window_5h_start: null,
    window_1d_start: null,
    window_7d_start: null,
    reset_5h_at: null,
    reset_1d_at: null,
    reset_7d_at: null,
    group: {
      id: 1,
      name: 'OpenAI',
      description: null,
      platform: 'openai',
      rate_multiplier: 1,
      is_exclusive: false,
      status: 'active',
      subscription_type: 'standard',
      daily_limit_usd: null,
      weekly_limit_usd: null,
      monthly_limit_usd: null,
      allow_image_generation: true,
      image_rate_independent: false,
      image_rate_multiplier: 1,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      require_oauth_only: false,
      require_privacy_set: false,
      created_at: '2026-06-29T00:00:00Z',
      updated_at: '2026-06-29T00:00:00Z',
    },
    ...overrides,
  }
}

describe('imageGeneration helpers', () => {
  it('builds a compact OpenAI images payload', () => {
    expect(
      buildImageGenerationPayload({
        model: 'gpt-image-2',
        prompt: '  draw a quiet desk  ',
        size: '1920x1088',
        count: 2,
        quality: 'high',
      }),
    ).toEqual({
      model: 'gpt-image-2',
      prompt: 'draw a quiet desk',
      size: '1920x1088',
      n: 2,
      quality: 'high',
      response_format: 'b64_json',
    })
  })

  it('parses base64 and url image responses', () => {
    const parsed = parseImageGenerationResponse({
      data: [
        { b64_json: 'abc123', revised_prompt: 'revised prompt' },
        { url: 'https://example.com/image.png' },
      ],
    })

    expect(parsed).toEqual([
      {
        id: expect.any(String),
        src: 'data:image/png;base64,abc123',
        revisedPrompt: 'revised prompt',
      },
      {
        id: expect.any(String),
        src: 'https://example.com/image.png',
        revisedPrompt: '',
      },
    ])
  })

  it('keeps active OpenAI keys whose groups allow image generation', () => {
    const keys = [
      makeKey({ id: 1, name: 'usable' }),
      makeKey({ id: 2, status: 'inactive' }),
      makeKey({
        id: 3,
        group: {
          ...makeKey({}).group!,
          platform: 'anthropic',
        },
      }),
      makeKey({
        id: 4,
        group: {
          ...makeKey({}).group!,
          allow_image_generation: false,
        },
      }),
      makeKey({ id: 5, group: undefined }),
    ]

    expect(getImageCapableOpenAIKeys(keys).map((key) => key.id)).toEqual([1])
  })
})
