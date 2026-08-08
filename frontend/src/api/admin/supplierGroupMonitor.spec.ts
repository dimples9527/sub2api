import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, buildGatewayUrlMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  buildGatewayUrlMock: vi.fn((path: string) => `gateway:${path}`),
}))

vi.mock('../client', () => ({
  apiClient: { get: getMock },
  buildGatewayUrl: buildGatewayUrlMock,
}))

import { getLocalMonitorStatus } from './supplierProviderData'

describe('supplier group monitor API', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockResolvedValue({ data: { groups: [] } })
  })

  it('loads supplier group monitor data from the local aggregation endpoint', async () => {
    await expect(getLocalMonitorStatus({ period: '90m', board: 'hot' })).resolves.toEqual({ groups: [] })
    expect(buildGatewayUrlMock).toHaveBeenCalledWith('/api/llm-monitor/local-status')
    expect(getMock).toHaveBeenCalledWith('gateway:/api/llm-monitor/local-status', {
      params: { period: '90m', board: 'hot' },
    })
  })
})