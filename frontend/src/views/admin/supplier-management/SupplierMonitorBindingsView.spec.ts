import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierMonitorBindingsView from './SupplierMonitorBindingsView.vue'
import type { SupplierProviderMonitorTarget } from '@/api/admin/supplierProviderData'

const monitorViewMocks = vi.hoisted(() => ({
  listMonitorTargets: vi.fn(),
  listBindableLocalAccounts: vi.fn(),
  bindMonitorTarget: vi.fn(),
  unbindMonitorTarget: vi.fn(),
  autoMatchMonitorTargets: vi.fn(),
  listProviders: vi.fn(),
  listCustomPlatforms: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/supplierProviderData', () => ({
  listSupplierMonitorTargets: monitorViewMocks.listMonitorTargets,
  listSupplierBindableLocalAccounts: monitorViewMocks.listBindableLocalAccounts,
  bindSupplierMonitorTarget: monitorViewMocks.bindMonitorTarget,
  unbindSupplierMonitorTarget: monitorViewMocks.unbindMonitorTarget,
  autoMatchSupplierMonitorTargets: monitorViewMocks.autoMatchMonitorTargets,
  default: {
    listSupplierMonitorTargets: monitorViewMocks.listMonitorTargets,
    listSupplierBindableLocalAccounts: monitorViewMocks.listBindableLocalAccounts,
    bindSupplierMonitorTarget: monitorViewMocks.bindMonitorTarget,
    unbindSupplierMonitorTarget: monitorViewMocks.unbindMonitorTarget,
    autoMatchSupplierMonitorTargets: monitorViewMocks.autoMatchMonitorTargets,
  },
}))

vi.mock('@/api/admin/supplierProviders', () => ({
  default: { list: monitorViewMocks.listProviders },
}))

vi.mock('@/api/admin/customPlatforms', () => ({
  customPlatformsAPI: { list: monitorViewMocks.listCustomPlatforms },
  default: { list: monitorViewMocks.listCustomPlatforms },
}))

vi.mock('@/utils/platformOptions', () => ({
  loadPlatformCatalog: vi.fn().mockResolvedValue(undefined),
  buildPlatformOptions: () => [],
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: monitorViewMocks.showError,
    showSuccess: monitorViewMocks.showSuccess,
  }),
}))

function createTarget(overrides: Partial<SupplierProviderMonitorTarget> = {}): SupplierProviderMonitorTarget {
  return {
    id: 31,
    provider_id: 7,
    provider_name: '皓悦',
    provider_enabled: true,
    monitor_key: '2',
    monitor_name: 'Plus-稳定',
    monitor_provider: 'sub2api',
    primary_model: 'gpt-5',
    availability_7d: 99.5,
    active: true,
    last_seen_at: new Date(Date.now() - 60_000).toISOString(),
    inactive_at: null,
    binding_groups: [],
    ...overrides,
  }
}

async function mountView(items: SupplierProviderMonitorTarget[]) {
  monitorViewMocks.listMonitorTargets.mockResolvedValue({ items, total: items.length })
  const wrapper = mount(SupplierMonitorBindingsView, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': {} } })],
      stubs: {
        SupplierModuleLayout: { template: '<div><slot /></div>' },
        BaseDialog: {
          props: ['show'],
          template: '<section v-if="show"><slot /><slot name="footer" /></section>',
        },
        Input: {
          inheritAttrs: false,
          props: ['modelValue'],
          template: '<input :value="modelValue" v-bind="$attrs" />',
        },
        Select: {
          inheritAttrs: false,
          props: ['modelValue', 'options'],
          template: '<button type="button" v-bind="$attrs">{{ modelValue }}</button>',
        },
        Pagination: true,
        ModelIcon: true,
        Icon: true,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('SupplierMonitorBindingsView 监控项失效原因', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    monitorViewMocks.listProviders.mockResolvedValue({ items: [], total: 0 })
    monitorViewMocks.listCustomPlatforms.mockResolvedValue([])
    monitorViewMocks.listBindableLocalAccounts.mockResolvedValue({ items: [], total: 0 })
  })

  it('上游移除的监控项与整体停用的供应商给出不同提示', async () => {
    const wrapper = await mountView([
      createTarget({ id: 31, active: false, inactive_at: new Date().toISOString() }),
      createTarget({ id: 32, provider_id: 8, provider_name: '停用的供应商', provider_enabled: false }),
    ])

    const chips = wrapper.findAll('.sp-monitor-inactive-chip')
    expect(chips).toHaveLength(2)
    expect(chips[0].text()).toBe('上游已移除')
    expect(chips[0].classes()).not.toContain('sp-monitor-inactive-chip-provider')
    expect(chips[1].text()).toBe('供应商已停用')
    expect(chips[1].classes()).toContain('sp-monitor-inactive-chip-provider')
  })

  it('供应商停用优先于上游移除，避免把整批冻结说成上游删了', async () => {
    const wrapper = await mountView([
      createTarget({ provider_enabled: false, active: false, inactive_at: new Date().toISOString() }),
    ])

    expect(wrapper.get('.sp-monitor-inactive-chip').text()).toBe('供应商已停用')
  })

  it('正常监控项不带失效标记', async () => {
    const wrapper = await mountView([createTarget()])

    expect(wrapper.find('.sp-monitor-inactive-chip').exists()).toBe(false)
    expect(wrapper.find('.sp-monitor-target-inactive').exists()).toBe(false)
  })
})

describe('SupplierMonitorBindingsView 最近同步陈旧阈值', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    monitorViewMocks.listProviders.mockResolvedValue({ items: [], total: 0 })
    monitorViewMocks.listCustomPlatforms.mockResolvedValue([])
    monitorViewMocks.listBindableLocalAccounts.mockResolvedValue({ items: [], total: 0 })
  })

  it('监控同步是 30s 一轮，所以分钟级停滞就要标记出来', async () => {
    const wrapper = await mountView([
      createTarget({ id: 31, last_seen_at: new Date(Date.now() - 60_000).toISOString() }),
      createTarget({ id: 32, last_seen_at: new Date(Date.now() - 20 * 60_000).toISOString() }),
      createTarget({ id: 33, last_seen_at: new Date(Date.now() - 20 * 24 * 3600_000).toISOString() }),
      createTarget({ id: 34, last_seen_at: '' }),
    ])

    const cells = wrapper.findAll('.sp-monitor-time')
    expect(cells).toHaveLength(4)
    expect(cells[0].classes()).not.toContain('warning')
    expect(cells[0].classes()).not.toContain('stale')
    expect(cells[1].classes()).toContain('warning')
    expect(cells[2].classes()).toContain('stale')
    expect(cells[3].classes()).toContain('never')
  })
})
