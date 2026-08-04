import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, putMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  putMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('../client', () => ({
  apiClient: {
    get: getMock,
    put: putMock,
    post: postMock,
  },
}))

import {
  listSupplierBalanceAlertConfigs,
  listSupplierBalanceAlertEvents,
  scanSupplierBalanceAlerts,
  updateSupplierBalanceAlertConfig,
} from './supplierBalanceAlert'

describe('supplierBalanceAlert API', () => {
  beforeEach(() => {
    getMock.mockReset()
    putMock.mockReset()
    postMock.mockReset()
    getMock.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20 } })
    putMock.mockResolvedValue({ data: { provider_id: 8, threshold: '12.50' } })
    postMock.mockResolvedValue({ data: { checked: 1, triggered: 0 } })
  })

  it('loads all balance alert configs without adding an empty provider filter', async () => {
    await listSupplierBalanceAlertConfigs()

    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/balance-alert/configs', {
      params: undefined,
    })
  })

  it('updates a provider threshold using a decimal string and cooldown seconds', async () => {
    await updateSupplierBalanceAlertConfig(8, {
      enabled: true,
      threshold: '12.50',
      cooldown_seconds: 3600,
    })

    expect(putMock).toHaveBeenCalledWith(
      '/admin/supplier-management/balance-alert/configs/8',
      { enabled: true, threshold: '12.50', cooldown_seconds: 3600 }
    )
  })

  it('keeps event filters and pagination in the event request', async () => {
    await listSupplierBalanceAlertEvents({
      provider_id: 8,
      event_type: 'balance_low',
      status: 'active',
      page: 2,
      page_size: 10,
    })

    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/balance-alert/events', {
      params: {
        provider_id: 8,
        event_type: 'balance_low',
        status: 'active',
        page: 2,
        page_size: 10,
      },
    })
  })

  it('starts a manual balance scan without a request body', async () => {
    await scanSupplierBalanceAlerts()

    expect(postMock).toHaveBeenCalledWith('/admin/supplier-management/balance-alert/scan')
  })
})
