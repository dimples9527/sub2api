import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('./client', () => ({
  apiClient: { get: getMock }
}))

import { getModelSquare } from './modelSquare'

// 与后端 GET /api/v1/model-square 返回的聚合数据结构保持一致。
// 模型上的 group_ids 是配置里手动绑定的分组，不再由渠道反推；
// 可用性也只看这份绑定，所以聚合数据里不再需要 channels。
const userPayload = {
  config: {
    platforms: [
      {
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', input_price: 0.000005, group_ids: [2] }],
      },
      {
        platform: 'glm',
        name: 'GLM',
        models: [{ id: 'glm-4.5', display_name: 'GLM-4.5', group_ids: [1] }],
      },
    ],
  },
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

    // 分组归属照抄配置里的手动绑定；分组列表仍按覆盖后的有效平台呈现。
    expect(openaiModel?.group_ids).toEqual([2])
    expect(glmModel?.group_ids).toEqual([1])
    // 可用性只看有没有绑定分组：两个模型都绑了，与渠道、账号状态无关。
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

    // 无覆盖时 GLM Group 保持原始平台 openai。
    expect(result.payload.groups).toEqual([
      { id: 1, name: 'GLM Group', platform: 'openai', rate_multiplier: 0.3 },
      { id: 2, name: 'OpenAI Group', platform: 'openai', rate_multiplier: 0.8 },
    ])
    const openaiModel = result.payload.models?.find(model => model.id === 'gpt-5.5')
    const glmModel = result.payload.models?.find(model => model.id === 'glm-4.5')
    // 绑定不受覆盖影响：两个模型各自保留配置里写好的分组。
    expect(openaiModel?.group_ids).toEqual([2])
    expect(glmModel?.group_ids).toEqual([1])
    /*
      覆盖消失后不再有 glm 平台的分组参与 glm-4.5 的最低倍率计算，倍率回落到默认 1 ——
      这才是「覆盖平台确实在起作用」的证据。

      可用性不随覆盖变化：它只看配置里手动绑的分组，与分组平台无关。
      这条曾经用 available 的 true/false 对照来证明覆盖生效，判据改掉后改看倍率。
    */
    expect(glmModel?.available).toBe(true)
    expect(glmModel?.rate_multiplier).toBe(1)
  })

  it('空数据也能安全生成空目录', async () => {
    getMock.mockResolvedValue({
      data: {
        config: { platforms: [] },
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
