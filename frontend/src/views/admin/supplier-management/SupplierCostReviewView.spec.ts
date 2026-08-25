import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierCostReviewView from './SupplierCostReviewView.vue'

const mocks = vi.hoisted(() => ({
  listReviews: vi.fn(),
  listHistory: vi.fn(),
  approve: vi.fn(),
  bulkApprove: vi.fn(),
  listProviders: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin/supplierProviderCostReviews', () => ({
  listSupplierProviderCostReviews: mocks.listReviews,
  listSupplierProviderCostReviewHistory: mocks.listHistory,
  approveSupplierProviderCostReview: mocks.approve,
  bulkApproveSupplierProviderCostReviews: mocks.bulkApprove,
}))

vi.mock('@/api/admin/supplierProviders', () => ({
  list: mocks.listProviders,
  supplierProvidersAPI: { list: mocks.listProviders },
  default: { list: mocks.listProviders },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
  }),
}))

const review = {
  id: 7,
  provider_id: 3,
  provider_name: '示例供应商',
  stat_date: '2026-08-24',
  upstream_cost: 12.345678,
  calculated_cost: 10.123456,
  auto_adopted_cost: 10.123456,
  final_cost: null,
  effective_cost: 10.123456,
  cost_delta: 2.222222,
  effective_delta: 0,
  status: 'pending_review',
  decision_type: 'none',
  approved_by: null,
  approved_at: null,
  sync_count: 1,
  last_sync_run_id: 99,
  last_synced_at: '2026-08-24T08:00:00Z',
  version: 4,
  created_at: '2026-08-24T08:00:00Z',
  updated_at: '2026-08-24T08:00:00Z',
}

function mountView() {
  return mount(SupplierCostReviewView, {
    global: {
      stubs: {
        SupplierModuleLayout: { template: '<div><slot /></div>' },
        DateRangePicker: { template: '<div />' },
        Select: { props: ['modelValue', 'options'], template: '<div />' },
        DataTable: {
          props: ['data', 'selectable', 'selectedKeys'],
          emits: ['update:selectedKeys', 'selectionChange'],
          template:
            '<div data-test="cost-review-table"><button v-if="data.length > 1" data-test="select-both" @click="$emit(\'update:selectedKeys\', data.map(row => row.id))">选择当前页</button><div v-for="row in data" :key="row.id"><span>{{ row.provider_name }}</span><slot name="cell-actions" :row="row" /></div></div>',
        },
        Pagination: { template: '<div />' },
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
      },
    },
  })
}

describe('SupplierCostReviewView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listReviews.mockResolvedValue({ items: [review], total: 1, page: 1, page_size: 20 })
    mocks.listProviders.mockResolvedValue({ items: [{ id: 3, name: '示例供应商' }], total: 1 })
    mocks.listHistory.mockResolvedValue([
      { id: 1, event_type: 'sync', operated_at: '2026-08-24T08:00:00Z', status: 'pending_review' },
    ])
    mocks.approve.mockResolvedValue({ ...review, status: 'approved', version: 5 })
    mocks.bulkApprove.mockResolvedValue({ items: [{ ...review, status: 'approved', version: 5 }], count: 1 })
  })

  it('加载供应商和成本核对列表', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(mocks.listProviders).toHaveBeenCalled()
    expect(mocks.listReviews).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      start_date: undefined,
      end_date: undefined,
    }))
    expect(wrapper.text()).toContain('示例供应商')
    expect(wrapper.text()).toContain('待审批')
  })

  it('重置筛选后查询全部日期的成本核对记录', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('button.sp-button.ghost').trigger('click')
    await flushPromises()

    expect(mocks.listReviews).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: undefined,
      end_date: undefined,
    }))
  })

  it('校验手动成本并提交当前版本', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="approve-7"]').trigger('click')
    await wrapper.find('[data-test="decision-manual"]').trigger('click')
    await wrapper.find('[data-test="manual-cost"] input').setValue('-1')
    await wrapper.find('[data-test="submit-approval"]').trigger('click')

    expect(mocks.approve).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith('请输入非负且最多 6 位小数的金额')

    await wrapper.find('[data-test="manual-cost"] input').setValue('8.500001')
    await wrapper.find('[data-test="submit-approval"]').trigger('click')
    await flushPromises()

    expect(mocks.approve).toHaveBeenCalledWith(7, {
      decision_type: 'manual',
      manual_cost: 8.500001,
      version: 4,
    })
    expect(mocks.showSuccess).toHaveBeenCalledWith('成本审批已提交')
  })

  it('加载历史并通过全局 Toast 提示版本冲突', async () => {
    mocks.approve.mockRejectedValue({ response: { data: { message: '版本冲突，请刷新后重试' } } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="history-7"]').trigger('click')
    await flushPromises()
    expect(mocks.listHistory).toHaveBeenCalledWith(7)
    expect(wrapper.text()).toContain('同步')

    await wrapper.find('[data-test="approve-7"]').trigger('click')
    await wrapper.find('[data-test="decision-upstream"]').trigger('click')
    await wrapper.find('[data-test="submit-approval"]').trigger('click')
    await flushPromises()
    expect(mocks.showError).toHaveBeenCalledWith('版本冲突，请刷新后重试')
  })

  it('选择当前页记录并默认采用计算值批量审批', async () => {
    const secondReview = { ...review, id: 8, provider_name: '第二个供应商', version: 6 }
    mocks.listReviews.mockResolvedValue({ items: [review, secondReview], total: 2, page: 1, page_size: 20 })
    mocks.bulkApprove.mockResolvedValue({ items: [review, secondReview], count: 2 })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="select-both"]').trigger('click')
    expect(wrapper.find('[data-test="bulk-approve"]').exists()).toBe(true)
    await wrapper.find('[data-test="bulk-approve"]').trigger('click')
    expect(wrapper.text()).toContain('计算成本')

    await wrapper.find('[data-test="submit-bulk-approval"]').trigger('click')
    await flushPromises()

    expect(mocks.bulkApprove).toHaveBeenCalledWith({
      items: [{ id: 7, version: 4 }, { id: 8, version: 6 }],
      decision_type: 'calculated',
    })
    expect(mocks.showSuccess).toHaveBeenCalledWith('已批量审批 2 条成本核对记录')
  })

  it('批量审批手动成本非法时不调用接口', async () => {
    const secondReview = { ...review, id: 8, provider_name: '第二个供应商', version: 6 }
    mocks.listReviews.mockResolvedValue({ items: [review, secondReview], total: 2, page: 1, page_size: 20 })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="select-both"]').trigger('click')
    await wrapper.find('[data-test="bulk-approve"]').trigger('click')
    await wrapper.find('[data-test="bulk-decision-manual"]').trigger('click')
    await wrapper.find('[data-test="bulk-manual-cost"] input').setValue('1.1234567')
    await wrapper.find('[data-test="submit-bulk-approval"]').trigger('click')

    expect(mocks.bulkApprove).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith('请输入非负且最多 6 位小数的金额')
  })

  it('批量审批失败时保留选择并提示错误', async () => {
    const secondReview = { ...review, id: 8, provider_name: '第二个供应商', version: 6 }
    mocks.listReviews.mockResolvedValue({ items: [review, secondReview], total: 2, page: 1, page_size: 20 })
    mocks.bulkApprove.mockRejectedValue({ response: { data: { message: '版本冲突，请刷新后重试' } } })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="select-both"]').trigger('click')
    await wrapper.find('[data-test="bulk-approve"]').trigger('click')
    await wrapper.find('[data-test="submit-bulk-approval"]').trigger('click')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('版本冲突，请刷新后重试')
    expect(wrapper.find('[data-test="submit-bulk-approval"]').exists()).toBe(true)
  })
})
