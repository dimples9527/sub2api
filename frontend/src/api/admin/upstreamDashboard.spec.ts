import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))

vi.mock('../client', () => ({
  apiClient: { get: getMock },
}))

import { get } from './upstreamDashboard'

describe('upstreamDashboard API', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockResolvedValue({ data: { range: '24h', issues: [] } })
  })

  it('loads the dashboard with the selected range', async () => {
    await expect(get('7d')).resolves.toMatchObject({ range: '24h' })
    expect(getMock).toHaveBeenCalledWith('/admin/upstream-management/dashboard', {
      params: { range: '7d' },
    })
  })
})
