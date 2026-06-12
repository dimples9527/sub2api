import { describe, expect, it } from 'vitest'

import {
  buildUpstreamMonitorTrendIndex,
  normalizeUpstreamMonitorGroupKey
} from './upstreamMonitorTrend'

describe('upstreamMonitorTrend', () => {
  it('indexes upstream monitor rows by normalized group name and keeps recent timeline points', () => {
    const payload = {
      data: {
        groups: [
          {
            provider: 'No Key Group',
            layers: [
              {
                timeline: Array.from({ length: 20 }, (_, index) => ({
                  status: index % 3 === 0 ? 1 : index % 3 === 1 ? 2 : 0,
                  availability: index % 3 === 0 ? 100 : index % 3 === 1 ? 70 : 0,
                  latency: 100 + index,
                  timestamp: 1_710_000_000 + index * 60
                })),
                current_status: {
                  status: 1,
                  latency: 88,
                  timestamp: 1_710_001_260
                }
              }
            ]
          }
        ]
      }
    }

    const index = buildUpstreamMonitorTrendIndex(payload)
    const row = index.get(normalizeUpstreamMonitorGroupKey('no-key_group'))

    expect(row?.provider).toBe('No Key Group')
    expect(row?.latency).toBe(88)
    expect(row?.trend).toHaveLength(18)
    expect(row?.trend.map(point => point.tone)).toEqual([
      'red',
      'green',
      'yellow',
      'red',
      'green',
      'yellow',
      'red',
      'green',
      'yellow',
      'red',
      'green',
      'yellow',
      'red',
      'green',
      'yellow',
      'red',
      'green',
      'yellow'
    ])
  })
})
