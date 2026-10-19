import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import HistoryVisibilityView from './HistoryVisibilityView.vue'

const { adminAPIMock, appStoreMock } = vi.hoisted(() => ({
  adminAPIMock: {
    groups: {
      getAllIncludingInactive: vi.fn(),
    },
    modelDegradation: {
      getModelDegradationHistoryVisibility: vi.fn(),
      setModelDegradationHistoryGroupHidden: vi.fn(),
      setModelDegradationHistoryHostHidden: vi.fn(),
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
    status: 'active',
  }
}

function createVisibility(overrides: Partial<{
  hidden_group_ids: number[]
  hidden_hosts: string[]
  hosts: Array<{ host: string; run_count: number }>
}> = {}) {
  return {
    hidden_group_ids: [],
    hidden_hosts: [],
    hosts: [],
    ...overrides,
  }
}

function mountView(groups: ReturnType<typeof createGroup>[], visibility: ReturnType<typeof createVisibility>) {
  adminAPIMock.groups.getAllIncludingInactive.mockResolvedValue(groups)
  adminAPIMock.modelDegradation.getModelDegradationHistoryVisibility.mockResolvedValue(visibility)
  adminAPIMock.modelDegradation.setModelDegradationHistoryGroupHidden.mockResolvedValue({ hidden: true })
  adminAPIMock.modelDegradation.setModelDegradationHistoryHostHidden.mockResolvedValue({ hidden: true })

  return mount(HistoryVisibilityView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template:
            '<div><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
        },
        DataTable: {
          props: ['data'],
          template:
            '<div>'
            + '<div v-for="row in data" :key="row.key" class="row">'
            + '<span class="row-title">{{ row.title }}</span>'
            + '<slot name="cell-hidden" :row="row" :value="row.hidden" />'
            + '</div>'
            + '</div>',
        },
        Pagination: {
          props: ['page', 'total', 'pageSize'],
          emits: ['update:page', 'update:pageSize'],
          template:
            '<div class="pagination" :data-total="total" :data-page="page" :data-page-size="pageSize">'
            + '<button class="next" @click="$emit(\'update:page\', page + 1)" /></div>',
        },
        SearchInput: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template:
            '<input class="search" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
        },
        Select: { template: '<div />' },
        Toggle: {
          props: ['modelValue', 'disabled'],
          emits: ['update:modelValue'],
          template:
            '<button class="toggle" :data-checked="String(modelValue)" @click="$emit(\'update:modelValue\', !modelValue)" />',
        },
        EmptyState: { template: '<div />' },
      },
    },
  })
}

describe('HistoryVisibilityView 列表与开关', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('合并渲染分组与上游主机行并展示 KPI 统计', async () => {
    const wrapper = mountView(
      [createGroup(1), createGroup(2)],
      createVisibility({
        hidden_group_ids: [2],
        hidden_hosts: ['api.bar.com'],
        hosts: [
          { host: 'api.foo.com', run_count: 5 },
          { host: 'api.bar.com', run_count: 2 },
        ],
      })
    )
    await flushPromises()

    const rows = wrapper.findAll('.row')
    expect(rows.map((row) => row.find('.row-title').text())).toEqual([
      'group-001',
      'group-002',
      'api.foo.com',
      'api.bar.com',
    ])

    const kpis = wrapper.findAll('.hv-kpi-value').map((item) => item.text())
    expect(kpis).toEqual(['2', '1', '1'])

    const toggles = wrapper.findAll('.row .toggle')
    expect(toggles[0].attributes('data-checked')).toBe('true')
    expect(toggles[1].attributes('data-checked')).toBe('false')
    expect(toggles[3].attributes('data-checked')).toBe('false')
  })

  it('数据超过每页容量时只渲染当前页并支持翻页', async () => {
    const wrapper = mountView(
      Array.from({ length: 25 }, (_, index) => createGroup(index + 1)),
      createVisibility()
    )
    await flushPromises()

    expect(wrapper.findAll('.row')).toHaveLength(20)
    expect(wrapper.find('.row').text()).toContain('group-001')

    const pagination = wrapper.find('.pagination')
    expect(pagination.attributes('data-total')).toBe('25')

    await wrapper.find('.pagination .next').trigger('click')

    const rows = wrapper.findAll('.row')
    expect(rows).toHaveLength(5)
    expect(rows[0].text()).toContain('group-021')
  })

  it('切换分组开关时调用分组隐藏接口并提示成功', async () => {
    const wrapper = mountView([createGroup(11)], createVisibility())
    await flushPromises()

    await wrapper.find('.row .toggle').trigger('click')
    await flushPromises()

    expect(adminAPIMock.modelDegradation.setModelDegradationHistoryGroupHidden).toHaveBeenCalledWith(11, true)
    expect(appStoreMock.showSuccess).toHaveBeenCalled()
    expect(wrapper.find('.row .toggle').attributes('data-checked')).toBe('false')
  })

  it('切换上游主机开关时调用主机隐藏接口', async () => {
    const wrapper = mountView(
      [],
      createVisibility({ hosts: [{ host: 'api.foo.com', run_count: 3 }] })
    )
    await flushPromises()

    await wrapper.find('.row .toggle').trigger('click')
    await flushPromises()

    expect(adminAPIMock.modelDegradation.setModelDegradationHistoryHostHidden).toHaveBeenCalledWith('api.foo.com', true)
    expect(adminAPIMock.modelDegradation.setModelDegradationHistoryGroupHidden).not.toHaveBeenCalled()
  })

  it('保存失败时回滚开关状态并提示错误', async () => {
    adminAPIMock.modelDegradation.setModelDegradationHistoryGroupHidden.mockRejectedValueOnce(new Error('boom'))

    const wrapper = mountView([createGroup(7)], createVisibility())
    await flushPromises()

    await wrapper.find('.row .toggle').trigger('click')
    await flushPromises()

    expect(appStoreMock.showError).toHaveBeenCalledWith('boom')
    expect(wrapper.find('.row .toggle').attributes('data-checked')).toBe('true')
  })

  it('关键词搜索只保留匹配的行', async () => {
    const wrapper = mountView(
      [createGroup(1), createGroup(2)],
      createVisibility({ hosts: [{ host: 'api.bar.com', run_count: 2 }] })
    )
    await flushPromises()

    await wrapper.find('.search').setValue('bar')
    await flushPromises()

    const rows = wrapper.findAll('.row')
    expect(rows).toHaveLength(1)
    expect(rows[0].text()).toContain('api.bar.com')
  })
})
