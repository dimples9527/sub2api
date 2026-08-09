import { describe, expect, it } from 'vitest'
import {
  buildGroupBusinessPlatformOptions,
  filterAndSortGroupsByBusinessPlatform,
  isGroupValidForBusinessPlatformFilter,
} from './groupBusinessPlatformFilter'

describe('groupBusinessPlatformFilter', () => {
  it('??????????????????????????', () => {
    const groups = [
      { label: 'OpenAI ??', rate: 1, platform: 'openai' },
      {
        label: '????',
        rate: 0.8,
        platform: 'composite',
        businessPlatform: 'glm',
        businessPlatformName: '?? GLM',
      },
    ]

    expect(
      buildGroupBusinessPlatformOptions(groups, {
        all: '????',
        platformLabel: (platform) => `???${platform}`,
      })
    ).toEqual([
      { value: '', label: '????' },
      { value: 'openai', label: '???openai' },
      { value: 'glm', label: '?? GLM' },
    ])
  })

  it('??????????????????', () => {
    const groups = [
      { label: 'OpenAI ??', rate: 1, platform: 'openai' },
      { label: '??????', rate: 2, platform: 'composite' },
      { label: '????', rate: 0.8, platform: 'composite', businessPlatform: 'glm' },
    ]

    expect(filterAndSortGroupsByBusinessPlatform(groups, 'glm').map((group) => group.label)).toEqual([
      '????',
    ])
    expect(filterAndSortGroupsByBusinessPlatform(groups, 'openai').map((group) => group.label)).toEqual([
      'OpenAI ??',
      '??????',
    ])
  })

  it('????????????????? composite ???????????', () => {
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
