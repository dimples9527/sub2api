import { describe, expect, it } from 'vitest'
import { getSupplierGroupRateInsight, getSupplierUpstreamRateBand } from './supplierGroupRates'

describe('supplier group rate insight', () => {
  it.each([
    [{ localGroupID: undefined, upstreamRate: 2, localRate: undefined, localStatus: '' }, 'unmatched', '未匹配', null],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 3, localStatus: 'inactive' }, 'inactive', '本地已停用', 1.5],
    [{ localGroupID: 7, upstreamRate: 0, localRate: 3, localStatus: 'active' }, 'invalid', '数据异常', null],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 1.8, localStatus: 'active' }, 'inverted', '倒挂风险', 0.9],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 2, localStatus: 'active' }, 'equal', '倍率持平', 1],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 2.1, localStatus: 'active' }, 'low', '收益偏低', 1.05],
    [{ localGroupID: 7, upstreamRate: 2, localRate: 2.2, localStatus: 'active' }, 'normal', '正常', 1.1],
  ])('classifies %o as %s', (input, code, label, ratio) => {
    expect(getSupplierGroupRateInsight(input)).toEqual({ code, label, ratio })
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
