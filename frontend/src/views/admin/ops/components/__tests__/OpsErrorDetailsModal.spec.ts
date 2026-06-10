import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import OpsErrorDetailsModal from '../OpsErrorDetailsModal.vue'

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listUpstreamErrors: vi.fn(async () => ({
      items: [
        {
          id: 77,
          created_at: '2026-06-09T14:00:00Z',
          phase: 'upstream',
          type: 'upstream_error',
          error_owner: 'provider',
          error_source: 'upstream_http',
          severity: 'P1',
          status_code: 502,
          platform: 'openai',
          model: 'gpt-5.5',
          resolved: false,
          client_request_id: 'client-1',
          request_id: 'req-1',
          message: 'Upstream request failed',
          user_email: '',
          account_name: 'acc',
          group_name: 'group',
        },
      ],
      total: 1,
      page: 1,
      page_size: 10,
    })),
    listRequestErrors: vi.fn(async () => ({
      items: [],
      total: 0,
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

vi.mock('../../utils/opsFormatters', () => ({
  getSeverityClass: () => '',
  formatDateTime: (value: string) => `date ${value}`,
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

const SelectStub = defineComponent({
  name: 'Select',
  props: {
    modelValue: { default: null },
    options: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  template: '<select />',
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

describe('OpsErrorDetailsModal', () => {
  it('从上游错误列表打开详情时携带 upstream 类型', async () => {
    const wrapper = mount(OpsErrorDetailsModal, {
      props: {
        show: false,
        timeRange: '1h',
        errorType: 'upstream',
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          Pagination: PaginationStub,
          'el-tooltip': defineComponent({ template: '<span><slot /></span>' }),
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    const detailButton = wrapper.findAll('button').find((button) => button.text() === 'admin.ops.errorLog.details')
    expect(detailButton).toBeTruthy()
    await detailButton!.trigger('click')

    expect(wrapper.emitted('openErrorDetail')).toEqual([[77, 'upstream']])
  })
})
