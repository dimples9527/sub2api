import { flushPromises, mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ModelSquareConfigView from './ModelSquareConfigView.vue'

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, resolve, reject }
}

function mountPriceSummaryView() {
  return mount(ModelSquareConfigView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="table" /></div>' },
        DataTable: {
          props: ['data'],
          template: '<div><slot v-if="data[0]" name="cell-price_summary" :row="data[0]" /></div>',
        },
        EmptyState: { template: '<div />' },
        BaseDialog: { template: '<div />' },
        ConfirmDialog: { template: '<div />' },
        Input: { template: '<input />' },
        SearchInput: { template: '<input />' },
        Select: { template: '<div />' },
        TextArea: { template: '<textarea />' },
        PlatformIcon: { template: '<span />' },
        Icon: { template: '<span />' },
      },
    },
  })
}

const { adminApiMock, appStoreMock } = vi.hoisted(() => ({
  adminApiMock: {
    modelSquareConfig: {
      get: vi.fn(),
      update: vi.fn(),
      getModelPricing: vi.fn(),
      listSyncAccounts: vi.fn(),
    },
    customPlatforms: {
      list: vi.fn(),
    },
    accounts: {
      list: vi.fn(),
    },
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: adminApiMock,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

describe('model square config wiring', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminApiMock.modelSquareConfig.get.mockResolvedValue({ platforms: [], updated_at: null })
    adminApiMock.modelSquareConfig.update.mockResolvedValue({ platforms: [], updated_at: null })
    adminApiMock.customPlatforms.list.mockResolvedValue([
      {
        id: 1,
        code: 'custom-alpha',
        name: '自定义平台 Alpha',
        enabled: true,
        sort_order: 0,
        created_at: '2026-08-11T00:00:00Z',
        updated_at: '2026-08-11T00:00:00Z',
      },
      {
        id: 2,
        code: 'custom-sigma',
        name: '自定义平台 Sigma',
        enabled: true,
        sort_order: 10,
        created_at: '2026-08-11T00:00:00Z',
        updated_at: '2026-08-11T00:00:00Z',
      },
    ])
    adminApiMock.accounts.list.mockResolvedValue({ items: [] })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({ found: false })
    adminApiMock.modelSquareConfig.listSyncAccounts.mockResolvedValue([])
  })

  it('registers the admin route between model monitor and announcements', () => {
    const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    const modelMonitorIndex = routerSource.indexOf("path: '/admin/model-monitor/custom-platforms'")
    const configIndex = routerSource.indexOf("path: '/admin/model-square/config'")
    const announcementsIndex = routerSource.indexOf("path: '/admin/announcements'")

    expect(configIndex).toBeGreaterThan(modelMonitorIndex)
    expect(configIndex).toBeLessThan(announcementsIndex)
    expect(routerSource).toContain("import('@/views/admin/model-square/ModelSquareConfigView.vue')")
  })

  it('registers the sidebar entry as an admin-only item between model monitor and announcements', () => {
    const sidebarSource = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')
    const modelMonitorIndex = sidebarSource.indexOf("path: '/admin/model-monitor'")
    const configIndex = sidebarSource.indexOf("path: '/admin/model-square/config'")
    const announcementsIndex = sidebarSource.indexOf("path: '/admin/announcements'")

    expect(configIndex).toBeGreaterThan(modelMonitorIndex)
    expect(configIndex).toBeLessThan(announcementsIndex)
    expect(sidebarSource).toContain("label: '模型广场配置'")
    expect(sidebarSource).toContain('hideInSimpleMode: true')
  })

  it('wires the model square config API to the new backend endpoints', () => {
    const apiSource = readFileSync(resolve(process.cwd(), 'src/api/admin/modelSquareConfig.ts'), 'utf8')
    expect(apiSource).toContain("/admin/upstream-management/model-square/config")
    expect(apiSource).toContain('/admin/upstream-management/model-square/sync-accounts')
    expect(apiSource).toContain('export async function get()')
    expect(apiSource).toContain('export async function update(payload: ModelSquareConfigPayload)')
    expect(apiSource).toContain('export async function listSyncAccounts(platform: string)')
  })

  it('includes custom platforms in the model square platform lists', () => {
    const viewSource = readFileSync(resolve(process.cwd(), 'src/views/admin/model-square/ModelSquareConfigView.vue'), 'utf8')

    expect(viewSource).toContain('const customPlatforms = ref<CustomPlatform[]>([])')
    expect(viewSource).toContain('for (const item of customPlatforms.value)')
    expect(viewSource).toContain('rank: 1')
    expect(viewSource).toContain('setCustomPlatformLabels(customPlatformList)')
  })

  it('keeps only input, output and cache prices and uses the model square pricing source', () => {
    const viewSource = readFileSync(resolve(process.cwd(), 'src/views/admin/model-square/ModelSquareConfigView.vue'), 'utf8')
    const apiSource = readFileSync(resolve(process.cwd(), 'src/api/admin/modelSquareConfig.ts'), 'utf8')

    expect(viewSource).toContain("{ key: 'price_summary', label: '价格（每 1M Tokens）'")
    expect(viewSource).toContain("{ key: 'input_price', label: '输入价格' }")
    expect(viewSource).toContain("{ key: 'output_price', label: '输出价格' }")
    expect(viewSource).toContain("{ key: 'cache_write_price', label: '缓存写入价格' }")
    expect(viewSource).toContain("{ key: 'cache_read_price', label: '缓存读取价格' }")
    expect(viewSource).not.toContain('input_price_priority')
    expect(viewSource).not.toContain('output_price_priority')
    expect(viewSource).not.toContain('cache_write_price_priority')
    expect(viewSource).not.toContain('cache_read_price_priority')
    expect(viewSource).not.toContain('cache_write_1h_price')
    expect(viewSource).not.toContain('image_input_price')
    expect(viewSource).not.toContain('image_output_price')
    expect(viewSource).not.toContain('per_request_price')
    expect(viewSource).toContain('模型配置中心')
    expect(viewSource).not.toContain('Model Square Config')
    expect(viewSource).toContain('官方参考价格来自项目动态价格目录')
    expect(viewSource).toContain('PRICE_PER_MILLION_TOKENS')
    expect(viewSource).toContain('displayPriceToStoredPrice')
    expect(viewSource).toContain('modelPriceGroups(row)')
    expect(viewSource).toContain('price-pill')
    expect(viewSource).toContain('adminAPI.modelSquareConfig.getModelPricing')
    expect(viewSource).not.toContain('adminAPI.channels.getModelDefaultPricing')
    expect(viewSource).toContain('isOfficialReferencePriceValue')
    expect(viewSource).toContain('只会回填当前未填写的价格字段')
    expect(apiSource).toContain('/admin/upstream-management/model-square/model-pricing')
    expect(apiSource).toContain('export async function getModelPricing')
  })

  it('renders custom platforms in the platform selector strip', async () => {
    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="actions" /><slot name="filters" /><slot /></div>' },
          DataTable: { template: '<div />' },
          EmptyState: { template: '<div />' },
          BaseDialog: { template: '<div><slot /></div>' },
          ConfirmDialog: { template: '<div />' },
          Input: { template: '<input />' },
          SearchInput: { template: '<input />' },
          Select: { template: '<div />' },
          TextArea: { template: '<textarea />' },
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()

    const text = wrapper.text()
    expect(text.indexOf('自定义平台 Alpha')).toBeGreaterThanOrEqual(0)
    expect(text.indexOf('自定义平台 Sigma')).toBeGreaterThanOrEqual(0)
    expect(text.indexOf('自定义平台 Alpha')).toBeLessThan(text.indexOf('自定义平台 Sigma'))
    expect(wrapper.text()).toContain('自定义平台 Sigma')
  })

  it('loads group-effective-platform accounts when opening the sync dialog', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'glm',
        name: 'GLM',
        models: [],
      }],
    })
    adminApiMock.modelSquareConfig.listSyncAccounts.mockResolvedValue([{
      id: 11,
      name: 'glm-group-account',
      platform: 'openai',
      type: 'api_key',
      status: 'active',
      group_ids: [101],
      group_names: ['glm-group'],
      effective_platform: 'glm',
    }])

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          EmptyState: { template: '<div />' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: { template: '<div />' },
          Input: { template: '<input />' },
          SearchInput: { template: '<input />' },
          Select: {
            props: ['options'],
            template: '<div><span v-for="option in options" :key="option.value">{{ option.label }}</span></div>',
          },
          TextArea: { template: '<textarea />' },
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()
    await wrapper.findAll('.platform-chip').find(button => button.text().includes('GLM'))!.trigger('click')
    await wrapper.findAll('button.btn-secondary')[2].trigger('click')
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.listSyncAccounts).toHaveBeenCalledWith('glm')
    expect(wrapper.text()).toContain('glm-group-account')
  })

  it('renders configured prices directly in the model list', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{
          id: 'gpt-5.5',
          display_name: 'GPT-5.5',
          source: 'manual',
          input_price: 0.000005,
          output_price: 0.00003,
          cache_write_price: 0.0000075,
          cache_read_price: 0.0000005,
        }],
      }],
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div><slot v-if="data[0]" name="cell-price_summary" :row="data[0]" /></div>',
          },
          EmptyState: { template: '<div />' },
          BaseDialog: { template: '<div><slot /></div>' },
          ConfirmDialog: { template: '<div />' },
          Input: { template: '<input />' },
          SearchInput: { template: '<input />' },
          Select: { template: '<div />' },
          TextArea: { template: '<textarea />' },
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('基础')
    expect(wrapper.text()).toContain('输入')
    expect(wrapper.text()).toContain('$5')
    expect(wrapper.text()).toContain('缓存')
    expect(wrapper.text()).toContain('写入')
    expect(wrapper.text()).toContain('$7.50')
    expect(wrapper.text()).toContain('读取')
    expect(wrapper.text()).toContain('$0.50')
    expect(wrapper.text()).not.toContain('优先级')
    expect(wrapper.text()).not.toContain('图像')
  })

  it('stores manually entered token prices as per-token values after showing per-million-token inputs', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({ updated_at: null, platforms: [] })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          EmptyState: { template: '<div />' },
          BaseDialog: {
            props: ['show', 'title'],
            template: '<section v-if="show"><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></section>',
          },
          ConfirmDialog: { template: '<div />' },
          Input: {
            props: ['modelValue', 'label', 'placeholder', 'type', 'required'],
            emits: ['update:modelValue'],
            template: '<label><span>{{ label }}</span><input :aria-label="label" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></label>',
          },
          SearchInput: { template: '<input />' },
          Select: { template: '<div />' },
          TextArea: { template: '<textarea />' },
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()

    const addButton = wrapper.findAll('button').find(button => button.text().includes('添加模型'))
    expect(addButton).toBeTruthy()
    await addButton!.trigger('click')

    await wrapper.find('input[aria-label="模型 ID"]').setValue('gpt-5.5')
    await wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').setValue('5')
    await wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').setValue('30')
    await wrapper.find('input[aria-label="缓存写入价格（USD / 1M Tokens）"]').setValue('6.25')
    await wrapper.find('input[aria-label="缓存读取价格（USD / 1M Tokens）"]').setValue('0.5')

    const submitButton = wrapper.findAll('button').find(button => button.text() === '保存')
    expect(submitButton).toBeTruthy()
    await submitButton!.trigger('click')

    const saveButton = wrapper.findAll('button').find(button => button.text().includes('保存配置'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    const savedPayload = adminApiMock.modelSquareConfig.update.mock.calls[0][0]
    expect(savedPayload.platforms[0].models[0]).toEqual({
      id: 'gpt-5.5',
      display_name: 'gpt-5.5',
      source: 'manual',
      input_price: 0.000005,
      output_price: 0.00003,
      cache_write_price: 0.00000625,
      cache_read_price: 0.0000005,
    })
  })

  it('shows official reference prices for existing models without saving them as configured prices', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'sync' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({
      found: true,
      input_price: 0.000005,
      output_price: 0.00003,
      cache_read_price: 0.0000005,
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div><slot v-if="data[0]" name="cell-price_summary" :row="data[0]" /></div>',
          },
          EmptyState: { template: '<div />' },
          BaseDialog: { template: '<div><slot /></div>' },
          ConfirmDialog: { template: '<div />' },
          Input: { template: '<input />' },
          SearchInput: { template: '<input />' },
          Select: { template: '<div />' },
          TextArea: { template: '<textarea />' },
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    expect(wrapper.text()).toContain('基础')
    expect(wrapper.text()).toContain('$5')
    expect(wrapper.text()).toContain('$30')
    expect(wrapper.text()).toContain('缓存')
    expect(wrapper.text()).toContain('$0.50')
    expect(wrapper.find('.price-reference-badge').text()).toBe('官方参考')

    const saveButton = wrapper.findAll('button').find(button => button.text().includes('保存配置'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    const savedPayload = adminApiMock.modelSquareConfig.update.mock.calls[0][0]
    expect(savedPayload.platforms[0].models[0]).toEqual({
      id: 'gpt-5.5',
      display_name: 'GPT-5.5',
      source: 'sync',
      input_price: null,
      output_price: null,
      cache_write_price: null,
      cache_read_price: null,
    })
  })

  it('prefills edit dialog prices from configured values first and official references for empty fields', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{
          id: 'gpt-5.5',
          display_name: 'GPT-5.5',
          source: 'manual',
          input_price: 0.000007,
        }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({
      found: true,
      input_price: 0.000005,
      output_price: 0.00003,
      cache_read_price: 0.0000005,
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div v-if="data[0]"><slot name="cell-price_summary" :row="data[0]" /><slot name="cell-actions" :row="data[0]" /></div>',
          },
          EmptyState: { template: '<div />' },
          BaseDialog: {
            props: ['show', 'title'],
            template: '<section v-if="show"><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></section>',
          },
          ConfirmDialog: { template: '<div />' },
          Input: {
            props: ['modelValue', 'label', 'placeholder', 'type', 'required'],
            emits: ['update:modelValue'],
            template: '<label><span>{{ label }}</span><input :aria-label="label" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></label>',
          },
          SearchInput: { template: '<input />' },
          Select: { template: '<div />' },
          TextArea: { template: '<textarea />' },
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()
    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text() === '编辑')
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')

    expect((wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('7')
    expect((wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('30')
    expect((wrapper.find('input[aria-label="缓存读取价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('0.50')
  })

  it('fills still-empty edit dialog prices after an in-flight official lookup completes', async () => {
    const pricingDeferred = createDeferred<{
      found: boolean
      input_price?: number
      output_price?: number
    }>()
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockReturnValue(pricingDeferred.promise)

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="actions" /><slot name="filters" /><slot name="table" /></div>' },
          DataTable: {
            props: ['data'],
            template: '<div v-if="data[0]"><slot name="cell-price_summary" :row="data[0]" /><slot name="cell-actions" :row="data[0]" /></div>',
          },
          EmptyState: { template: '<div />' },
          BaseDialog: {
            props: ['show', 'title'],
            template: '<section v-if="show"><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></section>',
          },
          ConfirmDialog: { template: '<div />' },
          Input: {
            props: ['modelValue', 'label', 'placeholder', 'type', 'required'],
            emits: ['update:modelValue'],
            template: '<label><span>{{ label }}</span><input :aria-label="label" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></label>',
          },
          SearchInput: { template: '<input />' },
          Select: { template: '<div />' },
          TextArea: { template: '<textarea />' },
          PlatformIcon: { template: '<span />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text() === '编辑')
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    expect((wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('')

    pricingDeferred.resolve({ found: true, input_price: 0.000005, output_price: 0.00003 })
    await flushPromises()
    await flushPromises()

    expect((wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('5')
    expect((wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('30')
  })

  it('falls back to the catalog model name when a synced model has a provider prefix', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'openai/gpt-5.5', display_name: 'GPT-5.5', source: 'sync' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockImplementation(async (model: string) => {
      if (model === 'gpt-5.5') {
        return {
          found: true,
          input_price: 0.000005,
          output_price: 0.00003,
        }
      }
      return { found: false }
    })

    const wrapper = mountPriceSummaryView()

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('openai/gpt-5.5')
    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    expect(wrapper.text()).toContain('$5')
    expect(wrapper.text()).toContain('$30')
    expect(wrapper.find('.price-reference-badge').text()).toBe('官方参考')
  })

  it('shows lookup status in the price column while official reference prices are loading', async () => {
    const pricingDeferred = createDeferred<{ found: boolean }>()
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'sync' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockReturnValue(pricingDeferred.promise)

    const wrapper = mountPriceSummaryView()

    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    expect(wrapper.text()).toContain('正在查询官方参考价')
    expect(wrapper.text()).not.toContain('未设置')

    pricingDeferred.resolve({ found: false })
    await flushPromises()
  })

  it('shows not-found status in the price column when the official catalog has no price', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-unknown', display_name: 'gpt-unknown', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({ found: false })

    const wrapper = mountPriceSummaryView()

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-unknown')
    expect(wrapper.text()).toContain('官方目录无价格')
    expect(wrapper.text()).not.toContain('未设置')
  })

  it('shows failure status in the price column when official price lookup fails', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockRejectedValue(new Error('network failed'))

    const wrapper = mountPriceSummaryView()

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    expect(wrapper.text()).toContain('官方价格查询失败')
    expect(wrapper.text()).not.toContain('未设置')
  })
})
