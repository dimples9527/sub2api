import { describe, expect, it } from 'vitest'
import {
  formatSupplierGroupRateDelta,
  getSupplierGroupRateInsight,
  getSupplierUpstreamRateBand,
} from './supplierGroupRates'

describe('supplier group rate insight', () => {
  it.each([
    [{ localGroupID: undefined, upstreamRate: 2, localRate: undefined, localStatus: '' }, 'unmatched', '未匹配', null],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 3, localStatus: 'inactive' }, 'inactive', '本地已停用', 1],
    [{ localGroupID: 7, upstreamRate: 0, localRate: 3, localStatus: 'active' }, 'invalid', '数据异常', null],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 1.8, localStatus: 'active' }, 'inverted', '倒挂风险', -0.2],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 2, localStatus: 'active' }, 'equal', '倍率持平', 0],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 2.1, localStatus: 'active' }, 'low', '收益偏低', 0.1],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 2.2, localStatus: 'active' }, 'normal', '正常', 0.2],
  ])('classifies %o as %s', (input, code, label, delta) => {
    const result = getSupplierGroupRateInsight(input)
    expect(result).toMatchObject({ code, label })
    if (delta === null) {
      expect(result.delta).toBeNull()
    } else {
      expect(result.delta).toBeCloseTo(delta, 12)
    }
  })

  it.each([
    [1.2, 1, '+0.2'],
    [1.23456789, 1.2, '+0.03456789'],
    [0.1, 0.3, '-0.2'],
    [2, 2, '0'],
    ['1.0000001', '1', '+0.0000001'],
    [Number.NaN, 1, '-'],
  ])('formats exact rate delta for %s - %s', (localRate, upstreamRate, expected) => {
    expect(formatSupplierGroupRateDelta(localRate, upstreamRate)).toBe(expected)
  })

  it.each([
    [Number.NaN, 'invalid'],
    [0, 'invalid'],
    [0.15, 'low'],
    [1, 'low'],
    [1.01, 'standard'],
    [2, 'standard'],
    [2.01, 'elevated'],
    [5, 'elevated'],
    [5.01, 'high'],
  ])('classifies upstream rate %s as %s', (rate, band) => {
    expect(getSupplierUpstreamRateBand(rate)).toBe(band)
  })
})
