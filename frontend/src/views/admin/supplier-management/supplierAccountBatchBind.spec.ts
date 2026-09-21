import { describe, expect, it } from 'vitest'
import {
  batchBindResultSummary,
  bindableAccountGroupsLabel,
  commonBatchBindPlatform,
  uniqueBatchBindAccountIDs,
} from './supplierAccountBatchBind'

describe('supplier account batch bind helpers', () => {
  it.each([
    [[1, 2, 3], [1, 2, 3]],
    [[1, null, 2, undefined], [1, 2]],
    [[0, -3, 1.5, Number.NaN], []],
    [[7, 7, 7], [7]],
    [[], []],
  ])('normalizes local account ids %o', (input, expected) => {
    expect(uniqueBatchBindAccountIDs(input as Array<number | null | undefined>)).toEqual(expected)
  })

  it.each([
    [['openai'], 'openai'],
    [['openai', 'openai'], 'openai'],
    [['OpenAI', ' openai '], 'openai'],
    [['anthropic', 'unknown', ''], 'anthropic'],
    [['openai', 'anthropic'], undefined],
    [['unknown'], undefined],
    [[], undefined],
  ])('resolves common platform for %o', (platforms, expected) => {
    expect(commonBatchBindPlatform(platforms as Array<string | undefined>)).toBe(expected)
  })

  it('reports bound, unchanged and failed counts in the toast text', () => {
    expect(
      batchBindResultSummary({ bound: 3, unchanged: 0, failed: 0, results: [] })
    ).toEqual({ text: '已为 3 个账号追加绑定分组', failed: false })
  })

  it('keeps the unchanged count so "已为 0 个账号" never shows up alone', () => {
    // 全部账号本来就在所选分组里时，只报「已为 0 个」会让用户以为操作失败。
    expect(
      batchBindResultSummary({ bound: 0, unchanged: 4, failed: 0, results: [] })
    ).toEqual({ text: '4 个账号本就在所选分组内', failed: false })
  })

  it('marks the summary as failed when any account failed', () => {
    const summary = batchBindResultSummary({ bound: 2, unchanged: 1, failed: 1, results: [] })
    expect(summary.failed).toBe(true)
    expect(summary.text).toBe('已为 2 个账号追加绑定分组，1 个账号本就在所选分组内，1 个账号失败')
  })

  it('treats a missing result as a failure instead of silently succeeding', () => {
    expect(batchBindResultSummary(null)).toEqual({
      text: '批量绑定分组没有返回结果，请刷新后重试',
      failed: true,
    })
  })

  it('renders the account group label', () => {
    expect(bindableAccountGroupsLabel({ groups: [] })).toBe('未加入分组')
    expect(
      bindableAccountGroupsLabel({
        groups: [
          { id: 1, name: '分组 A', platform: 'openai', rate_multiplier: 1, subscription_type: 'standard' },
          { id: 2, name: '分组 B', platform: 'openai', rate_multiplier: 1, subscription_type: 'standard' },
        ],
      })
    ).toBe('分组 A、分组 B')
  })
})
