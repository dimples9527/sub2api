import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, putMock, postMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  putMock: vi.fn(),
  postMock: vi.fn(),
  deleteMock: vi.fn(),
}))

vi.mock('../client', () => ({
  apiClient: {
    get: getMock,
    put: putMock,
    post: postMock,
    delete: deleteMock,
  },
}))

import {
  createSupplierCostAlertOverride,
  deleteSupplierCostAlertOverride,
  getSupplierCostAlertSettings,
  listSupplierCostAlertEvents,
  listSupplierCostAlertOverrides,
  updateSupplierCostAlertOverride,
  updateSupplierCostAlertSettings,
} from './supplierCostAlert'

describe('supplierCostAlert API', () => {
  beforeEach(() => {
    getMock.mockReset()
    putMock.mockReset()
    postMock.mockReset()
    deleteMock.mockReset()
    getMock.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20 } })
    putMock.mockResolvedValue({ data: { amount: '12.500000' } })
    postMock.mockResolvedValue({ data: { id: 1, provider_id: 8, enabled: true, amount: '3.500000' } })
    deleteMock.mockResolvedValue({ data: { message: 'ok' } })
  })

  it('读取全局成本预警阈值', async () => {
    await getSupplierCostAlertSettings()

    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/cost-alert/settings')
  })

  it('使用金额字符串保存全局成本预警阈值', async () => {
    await updateSupplierCostAlertSettings({ amount: '12.500000' })

    expect(putMock).toHaveBeenCalledWith('/admin/supplier-management/cost-alert/settings', {
      amount: '12.500000',
    })
  })

  it('创建供应商成本预警覆盖配置', async () => {
    await createSupplierCostAlertOverride({ provider_id: 8, enabled: true, amount: '3.500000' })

    expect(postMock).toHaveBeenCalledWith('/admin/supplier-management/cost-alert/overrides', {
      provider_id: 8,
      enabled: true,
      amount: '3.500000',
    })
  })

  it('更新供应商成本预警覆盖配置', async () => {
    await updateSupplierCostAlertOverride(5, { enabled: false, amount: '0.000000' })

    expect(putMock).toHaveBeenCalledWith('/admin/supplier-management/cost-alert/overrides/5', {
      enabled: false,
      amount: '0.000000',
    })
  })

  it('保留事件筛选和分页参数', async () => {
    await listSupplierCostAlertEvents({
      provider_id: 8,
      event_type: 'cost_overrun',
      status: 'active',
      page: 2,
      page_size: 10,
    })

    expect(getMock).toHaveBeenCalledWith('/admin/supplier-management/cost-alert/events', {
      params: {
        provider_id: 8,
        event_type: 'cost_overrun',
        status: 'active',
        page: 2,
        page_size: 10,
      },
    })
  })

  it('删除供应商成本预警覆盖配置', async () => {
    await deleteSupplierCostAlertOverride(5)

    expect(deleteMock).toHaveBeenCalledWith('/admin/supplier-management/cost-alert/overrides/5')
  })
})
