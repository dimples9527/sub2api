import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import OpsRequestDetailsModal from '../OpsRequestDetailsModal.vue'

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listRequestDetails: vi.fn(async () => ({
      items: [
        {
          kind: 'error',
          created_at: '2026-06-09T14:00:00Z',
          request_id: 'req-1',
          platform: 'openai',
          model: 'gpt-5.5',
          duration_ms: 123,
          status_code: 502,
          error_id: 42,
        },
      ],
      total: 1,
      page: 1,
      page_size: 10,
    })),
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(async () => true),
  }),
}))

vi.mock('../../utils/opsFormatters', () => ({
  parseTimeRangeMinutes: () => 60,
  formatDateTime: (value: string) => value,
}))

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
  },
  emits: ['close'],
  template: '<div v-if="show"><slot /></div>',
})

const PaginationStub = defineComponent({
  name: 'Pagination',
  props: {
    total: { type: Number, default: 0 },
    page: { type: Number, default: 1 },
    pageSize: { type: Number, default: 10 },
  },
  emits: ['update:page', 'update:pageSize'],
  template: '<div class="pagination-stub" />',
})

describe('OpsRequestDetailsModal', () => {
  it('从请求明细打开错误详情时固定携带 request 类型', async () => {
    const wrapper = mount(OpsRequestDetailsModal, {
      props: {
        modelValue: false,
        timeRange: '1h',
        preset: {
          title: '请求明细',
          kind: 'all',
          sort: 'created_at_desc',
        },
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Pagination: PaginationStub,
        },
      },
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const viewButton = wrapper.findAll('button').find((button) => button.text() === 'admin.ops.requestDetails.viewError')
    expect(viewButton).toBeTruthy()
    await viewButton!.trigger('click')

    expect(wrapper.emitted('openErrorDetail')).toEqual([[42, 'request']])
  })
})
