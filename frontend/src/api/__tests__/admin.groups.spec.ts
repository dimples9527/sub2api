import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post
  }
}))

import {
  getAccountRateGuardAudits,
  getAccountRateGuardStatus,
  getUpstreamAvailableGroups,
  getUpstreamKeySummary,
  getUpstreamRateWarnings,
  runAccountRateGuard
} from '@/api/admin/groups'

describe('admin groups api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
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

  it('fetches upstream key summary for the admin upstream groups page', async () => {
    const response = {
      groups: [
        {
          name: 'codex福利',
          normalized_name: 'codex福利',
          key_count: 2
        }
      ]
    }
    get.mockResolvedValue({ data: response })

    await expect(getUpstreamKeySummary()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/admin/upstream-management/key-summary')
  })

  it('runs account rate guard in dry-run mode', async () => {
    const response = {
      dry_run: true,
      checked_key_count: 2,
      matched_account_count: 1,
      violation_count: 1,
      unbound_count: 0,
      violations: [],
      providers: []
    }
    post.mockResolvedValue({ data: response })

    await expect(runAccountRateGuard(true)).resolves.toEqual(response)

    expect(post).toHaveBeenCalledWith('/admin/upstream-management/account-rate-guard/run', {
      dry_run: true
    })
  })

  it('fetches account rate guard status and audits', async () => {
    get.mockResolvedValueOnce({ data: { audits: [] } })
    get.mockResolvedValueOnce({ data: [{ run_id: 1 }] })

    await expect(getAccountRateGuardStatus()).resolves.toEqual({ audits: [] })
    await expect(getAccountRateGuardAudits()).resolves.toEqual([{ run_id: 1 }])

    expect(get).toHaveBeenNthCalledWith(
      1,
      '/admin/upstream-management/account-rate-guard/status'
    )
    expect(get).toHaveBeenNthCalledWith(
      2,
      '/admin/upstream-management/account-rate-guard/audits'
    )
  })
})
