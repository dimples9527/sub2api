import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierRechargeHistoryDialog from './SupplierRechargeHistoryDialog.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  sync: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin/supplierProviderRecharges', () => ({
  listSupplierProviderRecharges: mocks.list,
  syncSupplierProviderRecharges: mocks.sync,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
  }),
}))

describe('SupplierRechargeHistoryDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.list.mockResolvedValue({ items: [], total: 0, total_amount: 0, page: 1, page_size: 20 })
    mocks.sync.mockResolvedValue({ status: 'success', record_count: 2 })
  })

  function mountDialog(props: Record<string, unknown> = {}) {
    return mount(SupplierRechargeHistoryDialog, {
      props: { show: true, ...props },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            template: '<section v-if="show"><h2>{{ title }}</h2><slot /></section>',
          },
          Input: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', Number($event.target.value) || null)"><option v-for="option in options" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select>',
          },
          DataTable: {
            props: ['data', 'loading'],
            template: '<div data-test="table"><div v-for="row in data" :key="row.id">{{ row.provider_name }} {{ row.amount }}</div><slot name="empty" v-if="!data.length" /></div>',
          },
          Pagination: {
            template: '<button data-test="next-page" @click="$emit(\'update:page\', 2)">next</button>',
          },
        },
      },
    })
  }

  it('loads all supplier recharge records when opened', async () => {
    const wrapper = mountDialog({ providers: [{ id: 7, name: 'Alpha' }] })
    await flushPromises()

    expect(mocks.list).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(wrapper.text()).toContain('所有供应商充值记录')
  })

  it('fixes the provider filter and loads one supplier records', async () => {
    mountDialog({ providerId: 7, providerName: 'Alpha' })
    await flushPromises()

    expect(mocks.list).toHaveBeenCalledWith({ provider_id: 7, page: 1, page_size: 20 })
  })

  it('syncs history and reloads records with a success toast', async () => {
    const wrapper = mountDialog({ providerId: 7, providerName: 'Alpha' })
    await flushPromises()
    mocks.list.mockClear()

    await wrapper.get('button.sp-button.primary').trigger('click')
    await flushPromises()

    expect(mocks.sync).toHaveBeenCalledWith(7, true)
    expect(mocks.showSuccess).toHaveBeenCalledWith('充值历史同步完成')
    expect(mocks.list).toHaveBeenCalledWith({ provider_id: 7, page: 1, page_size: 20 })
  })

  it('shows failed enabled supplier details after a partial sync', async () => {
    mocks.sync.mockResolvedValueOnce({
      status: 'success',
      success_count: 3,
      failed_count: 1,
      items: [
        { provider_id: 11, provider_name: 'TKAPI2', status: 'failed', message: '\u4E0A\u6E38\u8FDE\u63A5\u8D85\u65F6' },
      ],
    })
    const wrapper = mountDialog()
    await flushPromises()

    await wrapper.get('button.sp-button.primary').trigger('click')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('\u5145\u503C\u8BB0\u5F55\u540C\u6B65\u5B8C\u6210\uFF0C\u4F46\u6709 1 \u4E2A\u4F9B\u5E94\u5546\u5931\u8D25\uFF1ATKAPI2\uFF08\u4E0A\u6E38\u8FDE\u63A5\u8D85\u65F6\uFF09')
  })

  it('shows API errors through the dialog error state', async () => {
    mocks.list.mockRejectedValueOnce(new Error('加载失败'))
    const wrapper = mountDialog()
    await flushPromises()

    expect(wrapper.get('[data-test="supplier-recharge-error"]').text()).toContain('加载失败')
  })
})