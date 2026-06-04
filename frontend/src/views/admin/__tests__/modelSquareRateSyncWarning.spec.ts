import { describe, expect, it } from 'vitest'
import {
  formatModelSquareRateSyncWarning,
  remainingModelSquareRateWarningCount,
  visibleModelSquareRateWarnings,
  type ModelSquareRateWarning
} from '../modelSquareRateSyncWarning'

describe('modelSquareRateSyncWarning', () => {
  it('formats a concise warning when upstream group rates are not lower than local rates', () => {
    const warnings: ModelSquareRateWarning[] = [
      {
        group_id: 10,
        group_name: 'codex special',
        local_rate_multiplier: 0.5,
        upstream_rate_multiplier: 0.8
      },
      {
        group_id: 20,
        group_name: 'Stable Group',
        local_rate_multiplier: 0.4,
        upstream_rate_multiplier: 0.4
      }
    ]

    expect(formatModelSquareRateSyncWarning(warnings)).toBe(
      'Upstream group rates are greater than or equal to local rates: codex special (upstream 0.8x, local 0.5x), Stable Group (upstream 0.4x, local 0.4x)'
    )
  })

  it('limits listed groups and reports the remaining count', () => {
    const warnings: ModelSquareRateWarning[] = Array.from({ length: 4 }, (_, index) => ({
      group_id: index + 1,
      group_name: `Group ${index + 1}`,
      local_rate_multiplier: 0.1,
      upstream_rate_multiplier: 0.2
    }))

    expect(formatModelSquareRateSyncWarning(warnings, 2)).toBe(
      'Upstream group rates are greater than or equal to local rates: Group 1 (upstream 0.2x, local 0.1x), Group 2 (upstream 0.2x, local 0.1x) and 4 groups total'
    )
  })

  it('returns an empty string when there is nothing to warn about', () => {
    expect(formatModelSquareRateSyncWarning([])).toBe('')
  })

  it('returns visible warning items and remaining count for the page banner', () => {
    const warnings: ModelSquareRateWarning[] = Array.from({ length: 5 }, (_, index) => ({
      group_id: index + 1,
      group_name: `Group ${index + 1}`,
      local_rate_multiplier: 0.1,
      upstream_rate_multiplier: 0.2
    }))

    expect(visibleModelSquareRateWarnings(warnings, 3).map((warning) => warning.group_name)).toEqual([
      'Group 1',
      'Group 2',
      'Group 3'
    ])
    expect(remainingModelSquareRateWarningCount(warnings, 3)).toBe(2)
  })
})
