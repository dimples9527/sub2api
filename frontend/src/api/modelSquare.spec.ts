import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('./client', () => ({
  apiClient: { get: getMock }
}))

import { getModelSquare } from './modelSquare'

// 与后端 GET /api/v1/model-square 返回的聚合数据结构保持一致。
const userPayload = {
  config: {
    platforms: [
      {
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', input_price: 0.000005 }],
      },
      {
        platform: 'glm',
        name: 'GLM',
        models: [{ id: 'glm-4.5', display_name: 'GLM-4.5' }],
      },
    ],
  },
  channels: [
    {
      id: 1,
      status: 'active',
      group_ids: [1, 2],
      model_pricing: [
        { platform: 'openai', models: ['gpt-5.5'] },
        { platform: 'glm', models: ['glm-4.5'] },
      ],
      model_mapping: {},
    },
  ],
  groups: [
    { id: 1, name: 'GLM Group', platform: 'openai', rate_multiplier: 0.3 },
    { id: 2, name: 'OpenAI Group', platform: 'openai', rate_multiplier: 0.8 },
  ],
  platform_overrides: [{ id: 1, effective_platform: 'glm' }],
  reference_prices: {
    'glm-4.5': { found: true, input_price: 0.000002, output_price: 0.000008 },
  },
}

describe('user model square API', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockResolvedValue({ data: userPayload })
  })

  it('调用用户端 /model-square 接口并复用聚合函数生成展示目录', async () => {
    const result = await getModelSquare()

    expect(getMock).toHaveBeenCalledWith('/model-square')

    expect(result.payload.groups).toEqual([
      { id: 1, name: 'GLM Group', platform: 'glm', rate_multiplier: 0.3 },
      { id: 2, name: 'OpenAI Group', platform: 'openai', rate_multiplier: 0.8 },
    ])

    const openaiModel = result.payload.models?.find(model => model.id === 'gpt-5.5')
    const glmModel = result.payload.models?.find(model => model.id === 'glm-4.5')

    // 分组平台覆盖后，GLM 分组不再挂在 openai 目录下，GLM 模型归属 GLM 分组。
    expect(openaiModel?.group_ids).toEqual([2])
    expect(glmModel?.group_ids).toEqual([1])
    expect(openaiModel?.available).toBe(true)
    expect(glmModel?.available).toBe(true)
    expect(openaiModel?.provider).toBe('OpenAI')
    expect(glmModel?.provider).toBe('GLM')

    // 配置缺 token 价格的 glm-4.5 用参考价补齐展示价格。
    expect(glmModel?.input_price).toBeGreaterThan(0)
  })

  it('无分组平台覆盖时沿用分组原始平台', async () => {
    getMock.mockResolvedValue({
      data: {
        ...userPayload,
        platform_overrides: [],
      },
    })

    const result = await getModelSquare()

    // 无覆盖时 GLM Group 原始平台为 openai，两个模型都会归属 openai 分组。
    expect(result.payload.groups).toEqual([
      { id: 1, name: 'GLM Group', platform: 'openai', rate_multiplier: 0.3 },
      { id: 2, name: 'OpenAI Group', platform: 'openai', rate_multiplier: 0.8 },
    ])
    const openaiModel = result.payload.models?.find(model => model.id === 'gpt-5.5')
    const glmModel = result.payload.models?.find(model => model.id === 'glm-4.5')
    expect(openaiModel?.group_ids).toEqual([1, 2])
    // 无覆盖时 GLM 分组保持原始平台 openai，glm-4.5（平台 glm）没有可归属的 glm 分组
    expect(glmModel?.group_ids).toEqual([])
  })

  it('空数据也能安全生成空目录', async () => {
    getMock.mockResolvedValue({
      data: {
        config: { platforms: [] },
        channels: [],
        groups: [],
        platform_overrides: [],
        reference_prices: {},
      },
    })

    const result = await getModelSquare()

    expect(result.payload.models).toEqual([])
    expect(result.payload.groups).toEqual([])
  })
})