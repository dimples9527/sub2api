import { describe, expect, it } from 'vitest'
import {
  buildGroupBusinessPlatformOptions,
  filterAndSortGroupsByBusinessPlatform,
  isGroupValidForBusinessPlatformFilter,
} from './groupBusinessPlatformFilter'

describe('groupBusinessPlatformFilter', () => {
  it('按分组生成平台筛选项，核心平台用标签其余用显式名称', () => {
    const groups = [
      { label: 'OpenAI 官方', rate: 1, platform: 'openai' },
      {
        label: '智谱 GLM',
        rate: 0.8,
        platform: 'composite',
        businessPlatform: 'glm',
        businessPlatformName: '智谱 GLM',
      },
    ]

    expect(
      buildGroupBusinessPlatformOptions(groups, {
        all: '全部平台',
        platformLabel: (platform) => `核心平台 ${platform}`,
      })
    ).toEqual([
      { value: '', label: '全部平台' },
      { value: 'openai', label: '核心平台 openai' },
      { value: 'glm', label: '智谱 GLM' },
    ])
  })

  it('显式业务平台分组只匹配自身，聚合分组命中核心平台', () => {
    const groups = [
      { label: 'OpenAI 官方', rate: 1, platform: 'openai' },
      { label: '聚合平台分组', rate: 2, platform: 'composite' },
      { label: '智谱 GLM', rate: 0.8, platform: 'composite', businessPlatform: 'glm' },
    ]

    expect(filterAndSortGroupsByBusinessPlatform(groups, 'glm').map((group) => group.label)).toEqual([
      '智谱 GLM',
    ])
    expect(filterAndSortGroupsByBusinessPlatform(groups, 'openai').map((group) => group.label)).toEqual([
      'OpenAI 官方',
      '聚合平台分组',
    ])
  })

  it('切换筛选后，带显式业务平台的 composite 分组只在匹配时保留', () => {
    expect(
      isGroupValidForBusinessPlatformFilter(1, 'composite', 'openai', {
        businessPlatform: 'glm',
      })
    ).toBe(false)
    expect(
      isGroupValidForBusinessPlatformFilter(1, 'composite', 'glm', {
        businessPlatform: 'glm',
      })
    ).toBe(true)
  })
})
