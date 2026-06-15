import { flushPromises, mount } from '@vue/test-utils'
import { h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UpstreamGroupsView from './UpstreamGroupsView.vue'

const { adminAPIMock, appStoreMock } = vi.hoisted(() => ({
  adminAPIMock: {
    upstreamManagement: {
      getGroups: vi.fn(),
      getRateFixConfig: vi.fn(),
      createLocalGroupFromUpstream: vi.fn(),
    },
    groups: {
      getUpstreamMonitorStatus: vi.fn(),
      update: vi.fn(),
    },
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: adminAPIMock,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

vi.mock('@/utils/upstreamMonitorTrend', () => ({
  buildUpstreamMonitorTrendIndex: () => new Map(),
  normalizeUpstreamMonitorGroupKey: (value: string) => value,
}))

describe('UpstreamGroupsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          upstream_group_key: 'vip',
          upstream_rate: 2.5,
          upstream_key_count: 3,
          matched: false,
          needs_rate_increase: false,
        },
      ],
      warnings: [],
      records: [],
    })
    adminAPIMock.upstreamManagement.getRateFixConfig.mockResolvedValue({
      enabled: false,
      interval_seconds: 3600,
    })
    adminAPIMock.upstreamManagement.createLocalGroupFromUpstream.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [],
      warnings: [],
      records: [],
    })
    adminAPIMock.groups.getUpstreamMonitorStatus.mockResolvedValue({ rows: [] })
    adminAPIMock.groups.update.mockResolvedValue({})
  })

  it('requires selecting a platform when syncing a local group from upstream', async () => {
    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                slots['cell-action']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue'],
            setup(props, { emit }) {
              return () => h('select', {
                value: props.modelValue,
                onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
              }, (props.options || []).map((option: any) => h('option', { value: option.value }, option.label)))
            },
          },
        },
      },
    })

    await flushPromises()

    const syncButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.syncLocalGroup'))
    expect(syncButton).toBeTruthy()
    await syncButton!.trigger('click')
    await flushPromises()

    const selects = wrapper.findAll('select')
    const platformSelect = selects.at(selects.length - 1)
    expect(platformSelect).toBeTruthy()
    expect((platformSelect!.element as HTMLSelectElement).value).toBe('')

    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.confirmSync'))!.trigger('click')
    expect(appStoreMock.showError).toHaveBeenCalledWith('admin.upstreamGroups.invalidPlatform')
    expect(adminAPIMock.upstreamManagement.createLocalGroupFromUpstream).not.toHaveBeenCalled()

    await platformSelect!.setValue('gemini')
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.confirmSync'))!.trigger('click')
    await flushPromises()

    expect(adminAPIMock.upstreamManagement.createLocalGroupFromUpstream).toHaveBeenCalledWith({
      upstream_group_name: 'VIP',
      platform: 'gemini',
      rate_multiplier: 2.5,
    })
  })

  it('updates matched local group rate from the action column', async () => {
    adminAPIMock.upstreamManagement.getGroups.mockResolvedValue({
      default_provider: { slug: 'default-upstream', name: 'Default upstream' },
      items: [
        {
          provider_slug: 'default-upstream',
          provider_name: 'Default upstream',
          upstream_group_name: 'VIP',
          upstream_group_key: 'vip',
          upstream_rate: 2.5,
          upstream_key_count: 3,
          local_group_id: 42,
          local_group_name: 'VIP local',
          local_rate: 1.5,
          matched: true,
          needs_rate_increase: true,
        },
      ],
      warnings: [],
      records: [],
    })

    const wrapper = mount(UpstreamGroupsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            setup(props, { slots }) {
              return () => h('div', props.data.map((row: any) => h('div', [
                slots['cell-action']?.({ row }),
              ])))
            },
          },
          EmptyState: true,
          Icon: true,
          UpstreamGroupAvailabilityTrend: { template: '<div />' },
          Select: true,
        },
      },
    })

    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text().includes('admin.upstreamGroups.editLocalRate'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()

    const input = wrapper.find('#local-rate-multiplier')
    expect((input.element as HTMLInputElement).value).toBe('1.5')
    await input.setValue('2.5')
    const saveButtons = wrapper.findAll('button').filter(button => button.text().includes('common.save'))
    await saveButtons[saveButtons.length - 1].trigger('click')
    await flushPromises()

    expect(adminAPIMock.groups.update).toHaveBeenCalledWith(42, { rate_multiplier: 2.5 })
  })
})
