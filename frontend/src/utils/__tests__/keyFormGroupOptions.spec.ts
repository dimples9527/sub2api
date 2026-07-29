import { describe, expect, it } from 'vitest'
import type { GroupPlatform } from '@/types'
import {
  buildKeyFormPlatformOptions,
  filterAndSortKeyFormGroupOptions,
  isSelectedGroupValidForPlatformFilter,
  matchesKeyFormPlatformFilter,
  sortKeyFormGroupsByRateAsc
} from '@/utils/keyFormGroupOptions'

function opt(partial: {
  label: string
  rate: number
  platform: GroupPlatform
  value?: number
}) {
  return {
    value: partial.value ?? 1,
    label: partial.label,
    rate: partial.rate,
    platform: partial.platform
  }
}

describe('keyFormGroupOptions', () => {
  it('全部平台时匹配任意分组', () => {
    expect(matchesKeyFormPlatformFilter('openai', '')).toBe(true)
    expect(matchesKeyFormPlatformFilter('composite', null)).toBe(true)
  })

  it('选择具体平台时包含同平台和 composite', () => {
    expect(matchesKeyFormPlatformFilter('openai', 'openai')).toBe(true)
    expect(matchesKeyFormPlatformFilter('composite', 'openai')).toBe(true)
    expect(matchesKeyFormPlatformFilter('anthropic', 'openai')).toBe(false)
  })

  it('按倍率升序排序，倍率相同按名称', () => {
    const sorted = sortKeyFormGroupsByRateAsc([
      opt({ label: 'Beta', rate: 1.5, platform: 'openai' }),
      opt({ label: 'Alpha', rate: 1, platform: 'openai' }),
      opt({ label: 'Gamma', rate: 1, platform: 'openai' })
    ])

    expect(sorted.map((item) => item.label)).toEqual(['Alpha', 'Gamma', 'Beta'])
  })

  it('过滤并按倍率排序', () => {
    const result = filterAndSortKeyFormGroupOptions(
      [
        opt({ label: 'Claude', rate: 2, platform: 'anthropic', value: 1 }),
        opt({ label: 'GPT', rate: 1.2, platform: 'openai', value: 2 }),
        opt({ label: 'Mix', rate: 0.8, platform: 'composite', value: 3 }),
        opt({ label: 'GPT-Pro', rate: 0.5, platform: 'openai', value: 4 })
      ],
      'openai'
    )

    expect(result.map((item) => item.label)).toEqual(['GPT-Pro', 'Mix', 'GPT'])
  })

  it('平台选项仅包含出现过的平台，并始终有全部', () => {
    const options = buildKeyFormPlatformOptions(
      [
        { platform: 'openai' },
        { platform: 'gemini' },
        { platform: 'openai' }
      ],
      {
        all: '全部平台',
        platformLabel: (platform) => platform
      }
    )

    expect(options).toEqual([
      { value: '', label: '全部平台' },
      { value: 'openai', label: 'openai' },
      { value: 'gemini', label: 'gemini' }
    ])
  })

  it('切换平台后校验已选分组是否仍有效', () => {
    expect(isSelectedGroupValidForPlatformFilter(1, 'openai', 'openai')).toBe(true)
    expect(isSelectedGroupValidForPlatformFilter(1, 'composite', 'openai')).toBe(true)
    expect(isSelectedGroupValidForPlatformFilter(1, 'anthropic', 'openai')).toBe(false)
    expect(isSelectedGroupValidForPlatformFilter(null, null, 'openai')).toBe(true)
  })
})
