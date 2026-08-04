import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock, putMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
  deleteMock: vi.fn(),
}))

vi.mock('../client', () => ({
  apiClient: {
    get: getMock,
    post: postMock,
    put: putMock,
    delete: deleteMock,
  },
}))

import {
  createSupplierNotificationChannel,
  createSupplierNotificationSubscription,
  deleteSupplierNotificationChannel,
  deleteSupplierNotificationSubscription,
  getSupplierNotificationDelivery,
  listSupplierNotificationChannels,
  listSupplierNotificationDeliveryAttempts,
  listSupplierNotificationDeliveries,
  listSupplierNotificationSubscriptions,
  sendSupplierNotificationChannelTest,
  updateSupplierNotificationChannel,
  updateSupplierNotificationSubscription,
} from './supplierNotifications'

describe('supplierNotifications API', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
    putMock.mockReset()
    deleteMock.mockReset()
    getMock.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20 } })
    postMock.mockResolvedValue({ data: { id: 1 } })
    putMock.mockResolvedValue({ data: { id: 1 } })
    deleteMock.mockResolvedValue({ data: { message: 'ok' } })
  })

  it('uses the notification channel CRUD endpoints', async () => {
    const input = {
      name: '飞书告警',
      channel_type: 'feishu' as const,
      enabled: true,
      feishu: { webhook_url: 'https://example.test/hook', secret: 'new-secret' },
    }

    await createSupplierNotificationChannel(input)
    expect(postMock).toHaveBeenCalledWith('/admin/supplier-management/notification-channels', input)

    await updateSupplierNotificationChannel(3, input)
    expect(putMock).toHaveBeenCalledWith('/admin/supplier-management/notification-channels/3', input)

    await deleteSupplierNotificationChannel(3)
    expect(deleteMock).toHaveBeenCalledWith('/admin/supplier-management/notification-channels/3')
  })

  it('keeps the test-send endpoint separate from channel updates', async () => {
    await sendSupplierNotificationChannelTest(3)

    expect(postMock).toHaveBeenCalledWith('/admin/supplier-management/notification-channels/3/test')
  })

  it('supports subscription CRUD and optional channel filtering', async () => {
    await listSupplierNotificationSubscriptions(3)
    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/notification-subscriptions', {
      params: { channel_id: 3 },
    })

    const input = { channel_id: 3, provider_id: null, event_type: 'balance_low' as const, enabled: true }
    await createSupplierNotificationSubscription(input)
    expect(postMock).toHaveBeenCalledWith('/admin/supplier-management/notification-subscriptions', input)

    await updateSupplierNotificationSubscription(7, input)
    expect(putMock).toHaveBeenCalledWith('/admin/supplier-management/notification-subscriptions/7', input)

    await deleteSupplierNotificationSubscription(7)
    expect(deleteMock).toHaveBeenCalledWith('/admin/supplier-management/notification-subscriptions/7')
  })

  it('loads delivery records, detail, and attempts with pagination', async () => {
    await listSupplierNotificationDeliveries({ channel_id: 3, status: 'failed', page: 2, page_size: 10 })
    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/notification-deliveries', {
      params: { channel_id: 3, status: 'failed', page: 2, page_size: 10 },
    })

    await getSupplierNotificationDelivery(11)
    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/notification-deliveries/11')

    await listSupplierNotificationDeliveryAttempts(11)
    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/notification-deliveries/11/attempts')
  })
})
