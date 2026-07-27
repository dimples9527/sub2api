import { describe, expect, it } from 'vitest'
import { buildSupplierGroupHealthTrendIndex } from './supplierGroupHealthTrend'

describe('supplierGroupHealthTrend', () => {
  it('按分组 ID 转换健康守护趋势，并保留无样本时间桶', () => {
    const index = buildSupplierGroupHealthTrendIndex([{
      group_id: 42,
      source: 'supplier_account_health_guard',
      availability: 50,
      latency: 180,
      time: '2026-07-27T10:25:00Z',
      trend: [
        {
          time: '2026-07-27T10:20:00Z',
          availability: 0,
          latency: 0,
          tested_account_count: 0,
          tone: 'gray',
        },
        {
          time: '2026-07-27T10:25:00Z',
          availability: 50,
          latency: 180,
          tested_account_count: 2,
          tone: 'yellow',
        },
      ],
    }])

    const trend = index.get(42)
    expect(trend).toMatchObject({
      provider: '供应商账号健康守护',
      availability: 50,
      latency: 180,
    })
    expect(trend?.trend).toEqual(expect.arrayContaining([
      expect.objectContaining({ tone: 'gray', statusText: '暂无检测', availability: 0 }),
      expect.objectContaining({ tone: 'yellow', statusText: '已测 2 个账号', availability: 50 }),
    ]))
  })
})