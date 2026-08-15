import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import CustomPlatformsView from './CustomPlatformsView.vue'

const { customPlatformsAPIMock, appStoreMock } = vi.hoisted(() => ({
  customPlatformsAPIMock: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
}))

vi.mock('@/api/admin/customPlatforms', () => ({
  customPlatformsAPI: customPlatformsAPIMock,
  default: customPlatformsAPIMock,
}))

vi.mock('@/utils/customPlatformLabels', () => ({
  setCustomPlatformLabels: vi.fn(),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

function createPlatform(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    code: 'glm',
    name: 'GLM',
    color: '#2563eb',
    enabled: true,
    sort_order: 0,
    created_at: '2026-08-11T00:00:00Z',
    updated_at: '2026-08-11T00:00:00Z',
    ...overrides,
  }
}

function mountView(platforms: unknown[]) {
  customPlatformsAPIMock.list.mockResolvedValue(platforms)
  return mount(CustomPlatformsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /></div>',
        },
        DataTable: {
          props: ['data'],
          template:
            '<div><template v-for="row in data" :key="row.id">' +
            '<slot name="cell-actions" :row="row" />' +
            '<slot name="cell-code" :row="row" :value="row.code" />' +
            '<slot name="cell-name" :row="row" :value="row.name" />' +
            '</template></div>',
        },
        SearchInput: { template: '<input />' },
        Select: { template: '<div />' },
        Input: {
          props: ['modelValue', 'placeholder', 'type', 'disabled'],
          emits: ['update:modelValue'],
          template:
            '<input :value="modelValue" :placeholder="placeholder" :type="type" :disabled="disabled" @input="$emit(\'update:modelValue\', $event.target.value)" />',
        },
        Toggle: { template: '<div />' },
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        ConfirmDialog: { template: '<div />' },
      },
    },
  })
}

describe('CustomPlatformsView 颜色搭配', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    customPlatformsAPIMock.get.mockResolvedValue(createPlatform())
    customPlatformsAPIMock.create.mockResolvedValue(createPlatform({ id: 99, code: 'new', name: 'New' }))
    customPlatformsAPIMock.update.mockResolvedValue(createPlatform())
    customPlatformsAPIMock.delete.mockResolvedValue({ message: 'ok' })
  })

  it('渲染颜色搭配横条并展示各平台色块', async () => {
    const wrapper = mountView([
      createPlatform(),
      createPlatform({ id: 2, code: 'deepseek', name: 'DeepSeek', color: '#4f46e5' }),
    ])
    await flushPromises()

    expect(wrapper.text()).toContain('颜色搭配')
    expect(wrapper.findAll('.cp-color-chip')).toHaveLength(2)
    expect(wrapper.text()).toContain('GLM')
    expect(wrapper.text()).toContain('DeepSeek')
  })

  it('无平台时颜色横条显示空态提示', async () => {
    const wrapper = mountView([])
    await flushPromises()

    expect(wrapper.find('.cp-color-strip-empty').exists()).toBe(true)
    expect(wrapper.text()).toContain('暂无平台')
  })

  it('新增平台默认选中未占用的预设色并随保存提交', async () => {
    // 现有平台占用第一个预设色 #3b82f6，新增时应自动选中下一个未使用色 #06b6d4
    const wrapper = mountView([createPlatform({ id: 1, code: 'a', name: 'A', color: '#3b82f6' })])
    await flushPromises()

    await wrapper.find('.cp-btn-primary').trigger('click')
    expect(wrapper.find('.cp-dialog-field-color').exists()).toBe(true)

    await wrapper.find('input[placeholder="例如 glm、deepseek、kimi"]').setValue('new')
    await wrapper.find('input[placeholder="例如 GLM、DeepSeek、Kimi"]').setValue('New')

    await wrapper.find('.cp-dialog-footer .cp-btn-primary').trigger('click')
    await flushPromises()

    expect(customPlatformsAPIMock.create).toHaveBeenCalledTimes(1)
    const payload = customPlatformsAPIMock.create.mock.calls[0][0]
    expect(payload.code).toBe('new')
    expect(payload.color).toBe('#06b6d4')
  })

  it('编辑平台回填颜色并按小写提交', async () => {
    const wrapper = mountView([createPlatform({ id: 7, code: 'kimi', name: 'Kimi', color: '#DB2777' })])
    await flushPromises()

    await wrapper.findAll('.cp-btn-edit')[0].trigger('click')

    const hexInput = wrapper.find('input[placeholder="#3b82f6"]')
    expect((hexInput.element as HTMLInputElement).value).toBe('#DB2777')

    await wrapper.find('.cp-dialog-footer .cp-btn-primary').trigger('click')
    await flushPromises()

    expect(customPlatformsAPIMock.update).toHaveBeenCalledTimes(1)
    expect(customPlatformsAPIMock.update).toHaveBeenCalledWith(7, expect.objectContaining({ color: '#db2777' }))
  })

  it('非法色值阻止保存并提示错误', async () => {
    const wrapper = mountView([createPlatform({ color: '#3b82f6' })])
    await flushPromises()

    await wrapper.find('.cp-btn-primary').trigger('click')
    await wrapper.find('input[placeholder="例如 glm、deepseek、kimi"]').setValue('new')
    await wrapper.find('input[placeholder="例如 GLM、DeepSeek、Kimi"]').setValue('New')
    await wrapper.find('input[placeholder="#3b82f6"]').setValue('#12345')

    expect(wrapper.find('.cp-color-error').exists()).toBe(true)

    await wrapper.find('.cp-dialog-footer .cp-btn-primary').trigger('click')
    await flushPromises()

    expect(customPlatformsAPIMock.create).not.toHaveBeenCalled()
    expect(appStoreMock.showError).toHaveBeenCalledWith(expect.stringContaining('颜色格式不正确'))
  })
})