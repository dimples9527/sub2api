import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get
  }
}))

import { getUpstreamAvailableGroups, getUpstreamRateWarnings } from '@/api/admin/groups'

describe('admin groups api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('fetches upstream rate warnings from the read-only endpoint', async () => {
    const response = {
      checked_count: 2,
      matched_count: 1,
      updated_count: 0,
      rate_warnings: [
        {
          group_id: 10,
          group_name: 'codex special',
          local_rate_multiplier: 0.5,
          upstream_rate_multiplier: 0.8
        }
      ]
    }
    get.mockResolvedValue({ data: response })

    await expect(getUpstreamRateWarnings()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/admin/upstream-management/rate-warnings')
  })

  it('fetches upstream available groups for the admin upstream groups page', async () => {
    const response = [
      {
        id: 2,
        name: 'codex福利',
        platform: 'openai',
        rate_multiplier: 0.15,
        status: 'active',
        local_group_id: 10,
        local_group_name: 'codex 福利',
        local_rate_multiplier: 0.2
      }
    ]
    get.mockResolvedValue({ data: response })

    await expect(getUpstreamAvailableGroups()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/admin/upstream-management/groups')
  })
})
