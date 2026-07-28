import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../client', () => ({
  apiClient: { get: getMock },
}))

import {
  getAccounts,
  getOverview,
  getProviders,
  getRates,
} from './supplierDashboard'

describe('supplierDashboard API', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockResolvedValue({ data: { range: '24h', items: [], total: 0, page: 1, page_size: 20, warnings: [], generated_at: '2026-07-25T08:00:00Z' } })
  })

  it('loads overview from the legacy dashboard endpoint with abort signal', async () => {
    const signal = new AbortController().signal
    getMock.mockResolvedValueOnce({ data: { range: '7d', generated_at: '2026-07-25T08:00:00Z' } })
    await expect(getOverview('7d', { signal })).resolves.toMatchObject({ range: '7d' })
    expect(getMock).toHaveBeenCalledWith('/admin/upstream-management/dashboard', {
      params: { range: '7d' },
      signal,
    })
  })

  it('loads accounts with defaults and nullable-friendly payload', async () => {
    const signal = new AbortController().signal
    getMock.mockResolvedValueOnce({
      data: {
        range: '24h',
        items: [{
          account_id: 1,
          account_name: 'a1',
          provider_slug: 'p',
          provider_name: 'P',
          group_key: 'g',
          group_name: 'G',
          severity: 'high',
          risk_types: ['balance'],
          request_count: null,
          success_rate: null,
          current_rate: 1.1,
          lowest_rate: null,
          rate_delta_percent: null,
          balance: null,
          balance_currency: null,
          estimated_days: 1.5,
          status: 'active',
          reason: '',
          period_cost: 0,
          estimated_extra_cost: null,
          traffic_impact: 0,
          detected_at: '2026-07-25T08:00:00Z',
          target_path: '/admin/upstream-management/accounts?account_id=1',
        }],
        total: 1,
        page: 1,
        page_size: 20,
        warnings: [],
        generated_at: '2026-07-25T08:00:00Z',
      },
    })

    const result = await getAccounts({}, { signal })
    expect(result.items[0]?.request_count).toBeNull()
    expect(result.items[0]?.period_cost).toBe(0)
    expect(getMock).toHaveBeenCalledWith('/admin/upstream-management/dashboard/accounts', {
      params: {
        range: '24h',
        risk_type: 'all',
        page: 1,
        page_size: 20,
      },
      signal,
    })
  })

  it('loads rates and providers with filters while omitting empty optional params', async () => {
    const signal = new AbortController().signal
    await getRates({ range: '7d', view: 'changed', comparison_status: 'not_lowest', provider_slug: 'p1', page: 2, page_size: 50 }, { signal })
    expect(getMock).toHaveBeenCalledWith('/admin/upstream-management/dashboard/rates', {
      params: {
        range: '7d',
        view: 'changed',
        comparison_status: 'not_lowest',
        provider_slug: 'p1',
        page: 2,
        page_size: 50,
      },
      signal,
    })

    getMock.mockClear()
    await getProviders({ range: '24h', status: 'high_risk', page: 3, page_size: 10 }, { signal })
    expect(getMock).toHaveBeenCalledWith('/admin/upstream-management/dashboard/providers', {
      params: {
        range: '24h',
        status: 'high_risk',
        page: 3,
        page_size: 10,
      },
      signal,
    })
  })
})
