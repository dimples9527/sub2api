import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import OpsSystemLogTable from '../OpsSystemLogTable.vue'

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listSystemLogs: vi.fn(async () => ({
      items: [
        {
          id: 9,
          created_at: '2026-06-09T15:00:00Z',
          level: 'error',
          component: 'handler.openai_gateway.responses',
          message: 'openai.forward_failed',
          request_id: 'req-9',
          client_request_id: 'client-9',
          user_id: 112,
          account_id: 174,
          platform: 'openai',
          model: 'gpt-5.4',
          extra: {
            error: 'upstream response failed: Upstream request failed',
            upstream_error_detail: '{"raw_body":"provider detail"}',
          },
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
    })),
    getSystemLogSinkHealth: vi.fn(async () => ({
      queue_depth: 0,
      queue_capacity: 1000,
      dropped_count: 0,
      write_failed_count: 0,
      written_count: 1,
      avg_write_delay_ms: 0,
    })),
    getRuntimeLogConfig: vi.fn(async () => ({
      level: 'info',
      enable_sampling: false,
      sampling_initial: 100,
      sampling_thereafter: 100,
      caller: true,
      stacktrace_level: 'error',
      retention_days: 30,
    })),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

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
    pageSize: { type: Number, default: 20 },
  },
  emits: ['update:page', 'update:pageSize'],
  template: '<div class="pagination-stub" />',
})

describe('OpsSystemLogTable', () => {
  it('可以打开系统日志详情并查看完整 extra JSON', async () => {
    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub,
        },
      },
    })

    await flushPromises()

    const detailButton = wrapper.findAll('button').find((button) => button.text() === '详情')
    expect(detailButton).toBeTruthy()
    await detailButton!.trigger('click')

    expect(wrapper.text()).toContain('系统日志详情')
    expect(wrapper.text()).toContain('req-9')
    expect(wrapper.text()).toContain('provider detail')
  })
})
