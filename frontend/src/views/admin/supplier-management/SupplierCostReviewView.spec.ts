import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierCostReviewView from './SupplierCostReviewView.vue'

const mocks = vi.hoisted(() => ({
  listReviews: vi.fn(),
  listHistory: vi.fn(),
  approve: vi.fn(),
  bulkApprove: vi.fn(),
  listProviders: vi.fn(),
  getCostAlertSettings: vi.fn(),
  updateCostAlertSettings: vi.fn(),
  listCostAlertOverrides: vi.fn(),
  createCostAlertOverride: vi.fn(),
  updateCostAlertOverride: vi.fn(),
  deleteCostAlertOverride: vi.fn(),
  listCostAlertEvents: vi.fn(),
  getCostSourceSettings: vi.fn(),
  updateCostSourceSettings: vi.fn(),
  listCostSourceOverrides: vi.fn(),
  createCostSourceOverride: vi.fn(),
  updateCostSourceOverride: vi.fn(),
  deleteCostSourceOverride: vi.fn(),
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

vi.mock('@/api/admin/supplierCostAlert', () => ({
  getSupplierCostAlertSettings: mocks.getCostAlertSettings,
  updateSupplierCostAlertSettings: mocks.updateCostAlertSettings,
  listSupplierCostAlertOverrides: mocks.listCostAlertOverrides,
  createSupplierCostAlertOverride: mocks.createCostAlertOverride,
  updateSupplierCostAlertOverride: mocks.updateCostAlertOverride,
  deleteSupplierCostAlertOverride: mocks.deleteCostAlertOverride,
  listSupplierCostAlertEvents: mocks.listCostAlertEvents,
}))

vi.mock('@/api/admin/supplierCostSource', () => ({
  getSupplierCostSourceSettings: mocks.getCostSourceSettings,
  updateSupplierCostSourceSettings: mocks.updateCostSourceSettings,
  listSupplierCostSourceOverrides: mocks.listCostSourceOverrides,
  createSupplierCostSourceOverride: mocks.createCostSourceOverride,
  updateSupplierCostSourceOverride: mocks.updateCostSourceOverride,
  deleteSupplierCostSourceOverride: mocks.deleteCostSourceOverride,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
  }),
}))

function toDateInputValue(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const today = toDateInputValue(new Date())
const yesterday = toDateInputValue(new Date(Date.now() - 24 * 60 * 60 * 1000))

const review = {
  id: 7,
  provider_id: 3,
  provider_name: '示例供应商',
  stat_date: '2026-08-24',
  upstream_cost: 12.345678,
  calculated_cost: 10.123456,
  local_cost: 21.681234,
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
        Select: { props: ['modelValue', 'options'], template: '<div />' },
        DataTable: {
          props: ['data', 'selectable', 'selectedKeys'],
          emits: ['update:selectedKeys', 'selectionChange'],
          template:
            '<div data-test="cost-review-table"><button v-if="data.length > 1" data-test="select-both" @click="$emit(\'update:selectedKeys\', data.map(row => row.id))">选择当前页</button><div v-for="row in data" :key="row.id" :data-test="`row-${row.id}`"><slot name="cell-provider_name" :row="row" /><span data-test="cell-upstream"><slot name="cell-upstream_cost" :row="row" /></span><span data-test="cell-calculated"><slot name="cell-calculated_cost" :row="row" /></span><span data-test="cell-local"><slot name="cell-local_cost" :row="row" /></span><slot name="cell-cost_delta" :row="row" /><slot name="cell-decision_type" :row="row" /><slot name="cell-approved_at" :row="row" /><slot name="cell-sync_count" :row="row" /><slot name="cell-last_synced_at" :row="row" /><slot name="cell-actions" :row="row" /></div></div>',
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
    mocks.getCostAlertSettings.mockResolvedValue({ amount: '2.000000' })
    mocks.listCostAlertOverrides.mockResolvedValue({ items: [] })
    mocks.listCostAlertEvents.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 10 })
    mocks.getCostSourceSettings.mockResolvedValue({ cost_source: 'auto' })
    mocks.listCostSourceOverrides.mockResolvedValue({ items: [] })
    mocks.listHistory.mockResolvedValue([
      { id: 1, event_type: 'sync', operated_at: '2026-08-24T08:00:00Z', status: 'pending_review' },
    ])
    mocks.approve.mockResolvedValue({ ...review, status: 'approved', version: 5 })
    mocks.bulkApprove.mockResolvedValue({ items: [{ ...review, status: 'approved', version: 5 }], count: 1 })
  })

  it('默认按今天加载供应商和成本核对列表', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(mocks.listProviders).toHaveBeenCalled()
    expect(mocks.listReviews).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      start_date: today,
      end_date: today,
    }))
    expect(wrapper.text()).toContain('示例供应商')
    expect(wrapper.text()).toContain('待审批')
    // 接口成本、计算成本（余额差+充值）、本地成本是三个互不相同的口径，串列会让核对失去意义
    expect(wrapper.find('[data-test="cell-upstream"]').text()).toBe('12.345678')
    expect(wrapper.find('[data-test="cell-calculated"]').text()).toBe('10.123456')
    expect(wrapper.find('[data-test="cell-local"]').text()).toBe('21.681234')
    expect(wrapper.find('[data-test="day-chip"]').text()).toBe(today)
    expect(wrapper.find('[data-test="provider-identity-3"]').classes()).toContain('provider-color-3')
  })

  it('通过按钮打开成本预警配置和事件弹窗', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="cost-alert-config-button"]').classes()).toContain('cost-alert-config-button')
    expect(wrapper.find('[data-test="cost-alert-events-button"]').classes()).toContain('cost-alert-events-button')
    expect(wrapper.find('[data-test="cost-alert-settings-section"]').exists()).toBe(false)
    await wrapper.find('[data-test="cost-alert-config-button"]').trigger('click')
    expect(wrapper.find('[data-test="cost-alert-settings-section"]').exists()).toBe(true)

    expect(wrapper.find('[data-test="cost-alert-events-section"]').exists()).toBe(false)
    await wrapper.find('[data-test="cost-alert-events-button"]').trigger('click')
    expect(wrapper.find('[data-test="cost-alert-events-section"]').exists()).toBe(true)
  })

  it('打开成本来源弹窗并保存全局默认来源', async () => {
    mocks.updateCostSourceSettings.mockResolvedValue({ cost_source: 'upstream' })
    const wrapper = mountView()
    await flushPromises()

    const costSourceButton = wrapper.find('[data-test="cost-source-config-button"]')
    expect(costSourceButton.classes()).toContain('cost-source-button')
    expect(wrapper.find('[data-test="cost-source-settings-section"]').exists()).toBe(false)

    await costSourceButton.trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="cost-source-settings-section"]').exists()).toBe(true)

    await wrapper.find('[data-test="save-cost-source-settings"]').trigger('click')
    await flushPromises()

    expect(mocks.updateCostSourceSettings).toHaveBeenCalledWith({ cost_source: 'auto' })
    expect(mocks.showSuccess).toHaveBeenCalledWith('全局成本来源已保存')
  })

  it('切换前一天并用今天按钮回到当天', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="next-day"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-test="today"]').attributes('disabled')).toBeDefined()

    await wrapper.find('[data-test="prev-day"]').trigger('click')
    await flushPromises()
    expect(mocks.listReviews).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: yesterday,
      end_date: yesterday,
    }))
    expect(wrapper.find('[data-test="day-chip"]').text()).toBe(yesterday)

    await wrapper.find('[data-test="today"]').trigger('click')
    await flushPromises()
    expect(mocks.listReviews).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: today,
      end_date: today,
    }))
  })

  it('通过日期面板选昨天后重新加载当天记录', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="stat-date"]').trigger('click')
    await flushPromises()
    document.querySelector<HTMLButtonElement>('[data-test="panel-yesterday"]')?.click()
    await flushPromises()

    expect(mocks.listReviews).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: yesterday,
      end_date: yesterday,
    }))
    expect(wrapper.find('[data-test="day-chip"]').text()).toBe(yesterday)
    document.body.innerHTML = ''
  })

  it('重置筛选后回到今天的成本核对记录', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-test="prev-day"]').trigger('click')
    await flushPromises()

    const resetButton = wrapper.findAll('button.sp-button.ghost').find(button => button.text() === '重置筛选')
    await resetButton?.trigger('click')
    await flushPromises()

    expect(mocks.listReviews).toHaveBeenLastCalledWith(expect.objectContaining({
      start_date: today,
      end_date: today,
    }))
  })

  it('打开当天同步记录弹窗并只保留同步事件', async () => {
    mocks.listHistory.mockResolvedValue([
      { id: 3, event_type: 'sync', operated_at: '2026-08-24T10:00:00Z', status: 'pending_review', sync_run_id: 101 },
      { id: 2, event_type: 'approve', operated_at: '2026-08-24T09:00:00Z', status: 'approved' },
      { id: 1, event_type: 'sync', operated_at: '2026-08-24T08:00:00Z', status: 'pending_review', sync_run_id: 99 },
    ])
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="sync-records-section"]').exists()).toBe(false)
    await wrapper.find('[data-test="sync-records-7"]').trigger('click')
    await flushPromises()

    expect(mocks.listHistory).toHaveBeenCalledWith(7)
    expect(wrapper.find('[data-test="sync-records-section"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="sync-records-count"]').text()).toBe('2 次')
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
    expect(wrapper.find('.history-item').text()).toContain('同步')

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

  it('加载成本预警配置并保存全局阈值', async () => {
    mocks.updateCostAlertSettings.mockResolvedValue({ amount: '5.000000' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="cost-alert-config-button"]').trigger('click')
    await wrapper.find('[data-test="cost-alert-global-amount"] input').setValue('5')
    await wrapper.find('[data-test="save-cost-alert-settings"]').trigger('click')
    await flushPromises()

    expect(mocks.updateCostAlertSettings).toHaveBeenCalledWith({ amount: '5' })
    expect(mocks.showSuccess).toHaveBeenCalledWith('全局成本预警阈值已保存')
  })

  it('新增覆盖配置未选择供应商时不调用接口', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="cost-alert-config-button"]').trigger('click')
    await wrapper.find('[data-test="add-cost-alert-override"]').trigger('click')

    expect(mocks.createCostAlertOverride).not.toHaveBeenCalled()
    expect(mocks.showError).toHaveBeenCalledWith('请选择需要覆盖预警阈值的供应商')
  })

  it('差额超出预警阈值时进入报警态并附上占计算成本的占比', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(mocks.listCostAlertOverrides).toHaveBeenCalled()
    const row = wrapper.find('[data-test="row-7"]')
    expect(row.find('.cost-alert').exists()).toBe(true)
    expect(row.find('.cost-alert').text()).toContain('+2.222222')
    expect(row.find('.delta-percent').text()).toBe('+22.0%')
  })

  it('差额未超阈值只作普通正向差额，缺失时退化为纯文本', async () => {
    mocks.getCostAlertSettings.mockResolvedValue({ amount: '10.000000' })
    mocks.listReviews.mockResolvedValue({
      items: [review, { ...review, id: 8, cost_delta: null }],
      total: 2,
      page: 1,
      page_size: 20,
    })
    const wrapper = mountView()
    await flushPromises()

    const positiveRow = wrapper.find('[data-test="row-7"]')
    expect(positiveRow.find('.cost-alert').exists()).toBe(false)
    expect(positiveRow.find('.cost-positive').exists()).toBe(true)

    const missingRow = wrapper.find('[data-test="row-8"]')
    expect(missingRow.find('.cost-neutral').exists()).toBe(false)
    expect(missingRow.find('.delta-percent').exists()).toBe(false)
  })

  it('供应商覆盖阈值优先于全局阈值判定差额报警', async () => {
    mocks.getCostAlertSettings.mockResolvedValue({ amount: '10.000000' })
    mocks.listCostAlertOverrides.mockResolvedValue({
      items: [{ id: 1, provider_id: 3, enabled: true, amount: '1.000000' }],
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="row-7"] .cost-alert').exists()).toBe(true)
  })

  it('停用的覆盖阈值回落到全局阈值', async () => {
    mocks.getCostAlertSettings.mockResolvedValue({ amount: '10.000000' })
    mocks.listCostAlertOverrides.mockResolvedValue({
      items: [{ id: 1, provider_id: 3, enabled: false, amount: '1.000000' }],
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="row-7"] .cost-alert').exists()).toBe(false)
    expect(wrapper.find('[data-test="row-7"] .cost-positive').exists()).toBe(true)
  })

  it('审批方式按采用来源渲染色点', async () => {
    mocks.listReviews.mockResolvedValue({
      items: [
        review,
        { ...review, id: 8, decision_type: 'manual' },
        { ...review, id: 9, decision_type: 'upstream' },
      ],
      total: 3,
      page: 1,
      page_size: 20,
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="row-7"] .decision-tag').classes()).toContain('decision-none')
    expect(wrapper.find('[data-test="row-8"] .decision-tag').classes()).toContain('decision-manual')
    expect(wrapper.find('[data-test="row-9"] .decision-tag').classes()).toContain('decision-upstream')
  })

  it('同步次数按梯度渲染', async () => {
    mocks.listReviews.mockResolvedValue({
      items: [review, { ...review, id: 8, sync_count: 3 }, { ...review, id: 9, sync_count: 7 }],
      total: 3,
      page: 1,
      page_size: 20,
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="row-7"] .sync-count-chip').classes()).toContain('sync-count-low')
    expect(wrapper.find('[data-test="row-8"] .sync-count-chip').classes()).toContain('sync-count-mid')
    expect(wrapper.find('[data-test="row-9"] .sync-count-chip').classes()).toContain('sync-count-high')
  })

  it('时间列同日只显示时分秒，跨日补上月日', async () => {
    mocks.listReviews.mockResolvedValue({
      items: [{ ...review, last_synced_at: new Date().toISOString(), approved_at: '2026-08-24T12:00:00Z' }],
      total: 1,
      page: 1,
      page_size: 20,
    })
    const wrapper = mountView()
    await flushPromises()

    const cells = wrapper.findAll('[data-test="row-7"] .sp-sub').map(cell => cell.text())
    expect(cells).toContainEqual(expect.stringMatching(/^\d{1,2}:\d{2}:\d{2}$/))
    expect(cells).toContainEqual(expect.stringMatching(/^08-24 \d{1,2}:\d{2}:\d{2}$/))
  })
})
