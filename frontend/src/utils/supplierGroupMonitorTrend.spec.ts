import { describe, expect, it } from 'vitest'
import {
  buildSupplierGroupMonitorTrendIndex,
  normalizeSupplierGroupMonitorKey,
} from './supplierGroupMonitorTrend'

describe('supplierGroupMonitorTrend', () => {
  it('将名称、分组 Key 和监控记录的 group_name 归一化为同一索引', () => {
    const index = buildSupplierGroupMonitorTrendIndex({
      data: [{
        provider: 'VIP Group',
        group_name: 'vip_group',
        layers: [{
          current_status: { status: 1, latency: 120, timestamp: 1_784_604_000 },
          timeline: [
            { status: 0, availability: 0, latency: 320, timestamp: 1_784_600_400 },
            { status: 1, availability: 100, latency: 120, timestamp: 1_784_604_000 },
          ],
        }],
      }],
    })

    const byName = index.get(normalizeSupplierGroupMonitorKey('VIP Group'))
    const byKey = index.get(normalizeSupplierGroupMonitorKey('vip-group'))
    expect(byName).toBeDefined()
    expect(byKey).toBe(byName)
    expect(byName).toMatchObject({ provider: 'VIP Group', availability: 50, latency: 120 })
    expect(byName?.trend.map(point => point.tone)).toEqual(['red', 'green'])
  })

  it('对空或无法识别的监控响应返回空索引', () => {
    expect(buildSupplierGroupMonitorTrendIndex({ message: 'empty' })).toEqual(new Map())
  })
})
