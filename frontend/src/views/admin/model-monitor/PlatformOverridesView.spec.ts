import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import PlatformOverridesView from './PlatformOverridesView.vue'

const { adminAPIMock, appStoreMock } = vi.hoisted(() => ({
  adminAPIMock: {
    modelMonitor: {
      listLLMMonitorGroupPlatformOverrides: vi.fn(),
      setLLMMonitorGroupPlatformOverride: vi.fn(),
      clearLLMMonitorGroupPlatformOverride: vi.fn(),
      setLLMMonitorGroupVisibility: vi.fn(),
    },
    customPlatforms: {
      list: vi.fn(),
    },
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: adminAPIMock,
  default: adminAPIMock,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

function createGroup(index: number) {
  return {
    id: index,
    name: `group-${String(index).padStart(3, '0')}`,
    platform: 'anthropic',
    actual_platform: '',
    effective_platform: 'anthropic',
    effective_platform_name: 'Anthropic',
    rate_multiplier: 1,
    show_in_monitor: true,
  }
}

function mountView(groupCount: number) {
  adminAPIMock.modelMonitor.listLLMMonitorGroupPlatformOverrides.mockResolvedValue(
    Array.from({ length: groupCount }, (_, i) => createGroup(i + 1))
  )
  adminAPIMock.customPlatforms.list.mockResolvedValue([])

  return mount(PlatformOverridesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>',
        },
        DataTable: {
          props: ['data'],
          template:
            '<div><div v-for="row in data" :key="row.id" class="row">{{ row.name }}</div></div>',
        },
        Pagination: {
          props: ['page', 'total', 'pageSize'],
          emits: ['update:page', 'update:pageSize'],
          template:
            '<div class="pagination" :data-total="total" :data-page="page" :data-page-size="pageSize">'
            + '<button class="next" @click="$emit(\'update:page\', page + 1)" /></div>',
        },
        SearchInput: { template: '<input />' },
        Select: { template: '<div />' },
        Toggle: { template: '<div />' },
        EmptyState: { template: '<div />' },
        BaseDialog: { template: '<div />' },
        ConfirmDialog: { template: '<div />' },
      },
    },
  })
}

describe('PlatformOverridesView 分页', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('分组数量超过每页容量时只渲染当前页并展示分页器', async () => {
    const wrapper = mountView(45)
    await flushPromises()

    expect(wrapper.findAll('.row')).toHaveLength(20)
    expect(wrapper.find('.row').text()).toBe('group-001')

    const pagination = wrapper.find('.pagination')
    expect(pagination.exists()).toBe(true)
    expect(pagination.attributes('data-total')).toBe('45')
  })

  it('切换页码渲染下一页分组', async () => {
    const wrapper = mountView(45)
    await flushPromises()

    await wrapper.find('.pagination .next').trigger('click')

    const rows = wrapper.findAll('.row')
    expect(rows).toHaveLength(20)
    expect(rows[0].text()).toBe('group-021')
  })

  it('无数据时不渲染分页器', async () => {
    const wrapper = mountView(0)
    await flushPromises()

    expect(wrapper.find('.pagination').exists()).toBe(false)
  })
})
