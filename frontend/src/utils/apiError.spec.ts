import { describe, expect, it } from 'vitest'

import { extractApiErrorMessage } from './apiError'

describe('extractApiErrorMessage', () => {
  it('prefers the backend error over the generic Axios status message', () => {
    expect(extractApiErrorMessage({
      status: 502,
      message: 'Request failed with status code 502',
      error: 'monitor upstream request failed',
    }, 'Failed to load availability trend')).toBe('monitor upstream request failed')
  })
})
