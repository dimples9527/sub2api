import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ModelSquareView from './ModelSquareView.vue'

// Select 组件是 Teleport 到 body 的自定义下拉框，这里用原生 select 模拟，
// 便于测试继续通过 setValue 触发 v-model 筛选
const SelectStub = defineComponent({
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: undefined,
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () => h('select', {
      ...attrs,
      class: ['input', attrs.class],
      value: props.modelValue ?? '',
      onChange: (event: Event) => {
        emit('update:modelValue', (event.target as HTMLSelectElement).value)
      },
    }, (props.options as Array<{ value: string | number | boolean | null; label: string }>)
      .map(option => h('option', { value: String(option.value) }, option.label)))
  },
})

const { getMock, showErrorMock, showSuccessMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  const labels: Record<string, string> = {
    'admin.modelSquare.inputPrice': 'Input',
    'admin.modelSquare.outputPrice': 'Output',
    'admin.modelSquare.cacheWritePrice': 'Cache write',
    'admin.modelSquare.cacheWrite1hPrice': 'Cache write 1h',
    'admin.modelSquare.cacheReadPrice': 'Cache read',
    'admin.modelSquare.priorityInputPrice': 'Priority input',
    'admin.modelSquare.priorityOutputPrice': 'Priority output',
    'admin.modelSquare.priorityCacheWritePrice': 'Priority cache write',
    'admin.modelSquare.priorityCacheReadPrice': 'Priority cache read',
    'admin.modelSquare.imageInputPrice': 'Image input',
    'admin.modelSquare.imageOutputPrice': 'Image output',
    'admin.modelSquare.perRequestPrice': 'Per request',
    'admin.modelSquare.unsetPrice': 'Not set',
    'admin.modelSquare.perMillionTokens': '$/M tokens',
    'admin.modelSquare.perRequest': '$/request',
    'admin.modelSquare.gridView': 'Grid view',
    'admin.modelSquare.listView': 'List view',
    'admin.modelSquare.available': 'Available',
    'admin.modelSquare.unavailable': 'Unavailable',
    'admin.modelSquare.copied': 'Copied',
    'admin.modelSquare.copyTitle': 'Copy model ID',
    'admin.modelSquare.unnamedModel': 'Unnamed model',
    'admin.modelSquare.allGroups': 'All groups',
    'admin.modelSquare.allProviders': 'All platforms',
    'admin.modelSquare.allModes': 'All modes',
    'admin.modelSquare.searchPlaceholder': 'Search model, platform, or mode...',
    'admin.modelSquare.emptyTitle': 'No models',
    'admin.modelSquare.emptyDescription': 'Configure model square first',
    'admin.modelSquare.providerSummary': '{count} models ? Rate {rate}',
    'admin.modelSquare.modelCount': 'Models',
    'admin.modelSquare.availableCount': 'Available count',
    'admin.modelSquare.groupCount': 'Groups count',
    'admin.modelSquare.rate': 'Rate',
    'admin.modelSquare.moreGroups': 'More',
    'admin.modelSquare.modes.image': 'Image',
    'admin.modelSquare.modes.embedding': 'Embedding',
    'admin.modelSquare.modes.responses': 'Responses',
    'admin.modelSquare.modes.chat': 'Chat',
    'admin.modelSquare.groupDialogTitle': '{id} groups',
    'admin.modelSquare.columns.status': 'Status',
    'admin.modelSquare.columns.provider': 'Platform',
    'admin.modelSquare.columns.modelId': 'Model ID',
    'admin.modelSquare.columns.input': 'Input',
    'admin.modelSquare.columns.output': 'Output',
    'admin.modelSquare.columns.cacheRead': 'Cache read',
    'admin.modelSquare.columns.cacheWrite': 'Cache write',
    'admin.modelSquare.columns.cacheWrite1h': 'Cache write 1h',
    'admin.modelSquare.columns.priorityInput': 'Priority input',
    'admin.modelSquare.columns.priorityOutput': 'Priority output',
    'admin.modelSquare.columns.priorityCacheWrite': 'Priority cache write',
    'admin.modelSquare.columns.priorityCacheRead': 'Priority cache read',
    'admin.modelSquare.columns.imageInput': 'Image input',
    'admin.modelSquare.columns.imageOutput': 'Image output',
    'admin.modelSquare.columns.perRequest': 'Per request',
    'admin.modelSquare.columns.mode': 'Mode',
    'admin.modelSquare.columns.groups': 'Groups',
    'admin.modelSquare.loading': 'Loading',
    'admin.modelSquare.sortBy': 'Sort',
    'admin.modelSquare.sortName': 'Name',
    'admin.modelSquare.sortPriceAsc': 'Price: low to high',
    'admin.modelSquare.sortPriceDesc': 'Price: high to low',
    'admin.modelSquare.filteredCount': 'Showing {filtered} / {total}',
    'admin.modelSquare.clearFilters': 'Clear filters',
    'admin.modelSquare.noMatchTitle': 'No matching models',
    'admin.modelSquare.noMatchDescription': 'No models match the current filters.',
    'admin.modelSquare.groupPricingTitle': 'Pricing by group',
    'admin.modelSquare.groupPricingEmpty': 'Not available in any group',
    'common.refresh': 'Refresh',
    'common.close': 'Close',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const label = labels[key] || key
        return label.replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? ''))
      }
    })
  }
})

vi.mock('@/api/modelSquare', () => ({
  modelSquareAPI: { get: getMock },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: showErrorMock, showSuccess: showSuccessMock }),
}))

vi.mock('@/composables/useRouteQueryFilters', () => ({
  useRouteQueryFilters: vi.fn(),
}))

const payload = {
  provider_slug: 'configured',
  provider_name: 'Model Square Config',
  provider_type: 'local',
  payload: {
    groups: [
      { id: 1, name: 'Default Group', platform: 'openai', rate_multiplier: 1 },
      { id: 2, name: 'Premium Group', platform: 'openai', rate_multiplier: 0.5 },
    ],
    models: [
      {
        id: 'gpt-5.5',
        display_name: 'GPT-5.5 Flagship',
        provider: 'OpenAI Official',
        platform: 'openai',
        available: true,
        mode: 'chat',
        input_price: 5,
        output_price: 30,
        cache_write_price: 6,
        cache_write_1h_price: 7,
        cache_read_price: 1,
        input_price_priority: 8,
        output_price_priority: 40,
        cache_write_price_priority: 9,
        cache_read_price_priority: 2,
        image_input_price: 10,
        image_output_price: 20,
        per_request_price: 0.12,
        rate_multiplier: 0.5,
        group_ids: [1],
      },
      {
        id: 'custom-model',
        display_name: 'Custom Model',
        provider: 'Custom Platform',
        platform: 'custom-platform',
        available: false,
        mode: 'image_generation',
        input_price: 0,
        rate_multiplier: 0.5,
        group_ids: [2],
      },
      {
        id: 'orphan-model',
        display_name: 'Orphan Model',
        provider: 'OpenAI Official',
        platform: 'openai',
        available: false,
        mode: 'chat',
        input_price: 1.25,
        rate_multiplier: 0.25,
        group_ids: [],
      },
    ],
  },
}

function mountView() {
  return mount(ModelSquareView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        TablePageLayout: { template: '<section><slot name="filters" /><slot name="table" /></section>' },
        EmptyState: { props: ['title', 'description'], template: '<div data-test="empty-state">{{ title }} {{ description }}<slot name="action" /></div>' },
        BaseDialog: { props: ['show', 'title'], template: '<div v-if="show" data-test="dialog"><slot /></div>' },
        Icon: { props: ['name'], template: '<span data-test="icon">{{ name }}</span>' },
        Select: SelectStub,
      },
    },
  })
}

describe('ModelSquareView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getMock.mockResolvedValue(payload)
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
  })

  it('renders configured display names, providers and prices with the fixed semantic price palette', async () => {
    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('GPT-5.5 Flagship (gpt-5.5)')
    expect(text).toContain('OpenAI Official')
    expect(text).toContain('\u8f93\u5165')
    expect(text).toContain('$5')
    expect(text).toContain('\u8f93\u51fa')
    expect(text).toContain('$30')
    expect(text).toContain('\u7f13\u5b58\u5199\u5165')
    expect(text).toContain('$6')
    expect(text).toContain('\u7f13\u5b58\u8bfb\u53d6')
    expect(text).toContain('$1')
    const openAICard = wrapper.findAll('[data-test="model-card"]')
      .find(card => card.text().includes('GPT-5.5 Flagship'))
    const cardTitle = openAICard?.attributes('title')
    expect(cardTitle).toContain('\u7f13\u5b58\u5199\u5165 1h: $7 $/\u767e\u4e07 tokens')
    expect(cardTitle).toContain('\u4f18\u5148\u7ea7\u8f93\u5165: $8 $/\u767e\u4e07 tokens')
    expect(cardTitle).toContain('\u4f18\u5148\u7ea7\u8f93\u51fa: $40 $/\u767e\u4e07 tokens')
    expect(cardTitle).toContain('\u4f18\u5148\u7ea7\u7f13\u5b58\u5199\u5165: $9 $/\u767e\u4e07 tokens')
    expect(cardTitle).toContain('\u4f18\u5148\u7ea7\u7f13\u5b58\u8bfb\u53d6: $2 $/\u767e\u4e07 tokens')
    expect(cardTitle).toContain('\u56fe\u50cf\u8f93\u5165: $10 $/\u767e\u4e07 tokens')
    expect(cardTitle).toContain('\u56fe\u50cf\u8f93\u51fa: $20 $/\u767e\u4e07 tokens')
    expect(cardTitle).toContain('\u6309\u8bf7\u6c42: $0.12 $/\u6b21')
    expect(openAICard?.find('.model-rate-chip').text()).toBe('0.5x')
    const orphanCard = wrapper.findAll('[data-test="model-card"]')
      .find(card => card.text().includes('Orphan Model'))
    expect(orphanCard?.find('.model-rate-chip').text()).toBe('0.25x')
  expect(wrapper.find('.price-box-teal').exists()).toBe(true)
  expect(wrapper.find('.price-box-orange').exists()).toBe(true)
  expect(wrapper.find('.price-box-blue').exists()).toBe(true)
  expect(wrapper.find('.price-box-violet').exists()).toBe(true)
  expect(wrapper.find('.price-box-neutral').exists()).toBe(false)
  expect(wrapper.find('.price-box-amber').exists()).toBe(false)
  expect(wrapper.find('.price-box-cyan').exists()).toBe(false)
    expect(wrapper.find('.price-box-emerald').exists()).toBe(false)
    expect(wrapper.find('.table-price-chip').exists()).toBe(false)
  })

  it('详情弹窗一次列出全部可用分组的价格，并高亮卡片倍率对应的分组', async () => {
    const wrapper = mountView()
    await flushPromises()

    const openAICard = wrapper.findAll('[data-test="model-card"]')
      .find(card => card.text().includes('GPT-5.5 Flagship'))
    await openAICard?.find('.model-detail-button').trigger('click')

    const dialog = wrapper.find('[data-test="dialog"]')
    expect(dialog.exists()).toBe(true)
    expect(dialog.text()).toContain('Pricing by group')

    // 表头保留四个语义价格位（顺序同 defaultPriceDescriptors），不因优先级/图片/按请求价格而被顶掉
    expect(dialog.findAll('thead th').map(th => th.text()))
      .toEqual(['Groups', 'Rate', '输入', '输出', '缓存读取', '缓存写入'])

    const rows = dialog.findAll('[data-test="detail-group-row"]')
    expect(rows).toHaveLength(2)

    // 分组按倍率升序：Premium(0.5) 在前、Default(1) 在后。
    // 价格 = input_price 5 ÷ 模型倍率 0.5 × 分组倍率，所以两行分别是 $5 与 $10。
    const premiumRow = rows.find(row => row.text().includes('Premium Group'))
    const defaultRow = rows.find(row => row.text().includes('Default Group'))
    expect(premiumRow?.text()).toContain('0.5x')
    expect(premiumRow?.text()).toContain('$5')
    expect(premiumRow?.text()).toContain('$30')
    expect(defaultRow?.text()).toContain('1x')
    expect(defaultRow?.text()).toContain('$10')
    expect(defaultRow?.text()).toContain('$60')

    // 卡片倍率 chip 是 0.5，只有 Premium 行应被标记为当前分组
    expect(premiumRow?.classes()).toContain('active')
    expect(defaultRow?.classes()).not.toContain('active')
  })

  it('按覆盖后的平台过滤详情弹窗中的分组', async () => {
    getMock.mockResolvedValue({
      provider_slug: 'configured',
      provider_name: 'Model Square Config',
      provider_type: 'local',
      payload: {
        groups: [
          { id: 1, name: 'OpenAI Group', platform: 'openai', rate_multiplier: 1 },
          { id: 2, name: 'GLM Group', platform: 'glm', rate_multiplier: 0.5 },
        ],
        models: [
          {
            id: 'glm-4.5',
            display_name: 'GLM-4.5',
            provider: 'GLM',
            platform: 'glm',
            available: true,
            mode: 'chat',
            input_price: 5,
            rate_multiplier: 0.5,
            group_ids: [2],
          },
        ],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    const glmCard = wrapper.findAll('[data-test="model-card"]')
      .find(card => card.text().includes('GLM-4.5'))
    await glmCard?.find('.model-detail-button').trigger('click')

    const dialogText = wrapper.find('[data-test="dialog"]').text()
    expect(dialogText).toContain('GLM Group')
    expect(dialogText).not.toContain('OpenAI Group')
  })

  it('renders unset prices as zero while preserving explicit zero prices', async () => {
    const wrapper = mountView()
    await flushPromises()

    const cards = wrapper.findAll('[data-test="model-card"]')
    const customCard = cards.find(card => card.text().includes('Custom Model'))
    expect(customCard?.text()).toContain('$0')
    expect(customCard?.text()).not.toContain('$0.000')
    expect(customCard?.findAll('.price-box-unset').length).toBeGreaterThan(0)
  })

  it('keeps search, platform filter, group filter, view switching and copying interactions', async () => {
    const wrapper = mountView()
    await flushPromises()

    const search = wrapper.find('input[type="search"]')
    await search.setValue('custom')
    expect(wrapper.text()).toContain('Custom Model (custom-model)')
    expect(wrapper.text()).not.toContain('GPT-5.5 Flagship (gpt-5.5)')

    await search.setValue('')
    const selects = wrapper.findAll('select')
    await selects[1].setValue('OpenAI Official')
    expect(wrapper.text()).toContain('GPT-5.5 Flagship (gpt-5.5)')
    expect(wrapper.text()).not.toContain('Custom Model (custom-model)')

    await selects[1].setValue('')
    await selects[0].setValue('2')
    expect(wrapper.text()).toContain('Custom Model (custom-model)')
    expect(wrapper.text()).not.toContain('GPT-5.5 Flagship (gpt-5.5)')

    await wrapper.find('button[title="List view"]').trigger('click')
    expect(wrapper.find('table').exists()).toBe(true)

    await wrapper.find('[data-test="model-row"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('custom-model')
    expect(showSuccessMock).toHaveBeenCalledWith('Copied')
  })

  it('工具条支持按名称与价格排序，网格与列表复用同一顺序', async () => {
    getMock.mockResolvedValue({
      provider_slug: 'configured',
      provider_name: 'Model Square Config',
      provider_type: 'local',
      payload: {
        groups: [{ id: 1, name: 'Default Group', platform: 'openai', rate_multiplier: 1 }],
        models: [
          { id: 'alpha', display_name: 'Alpha', provider: 'Same Platform', platform: 'openai', available: true, mode: 'chat', input_price: 30, rate_multiplier: 1, group_ids: [1] },
          { id: 'beta', display_name: 'Beta', provider: 'Same Platform', platform: 'openai', available: true, mode: 'chat', input_price: 10, rate_multiplier: 1, group_ids: [1] },
          { id: 'gamma', display_name: 'Gamma', provider: 'Same Platform', platform: 'openai', available: true, mode: 'chat', input_price: 20, rate_multiplier: 1, group_ids: [1] },
        ],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    const cardTitles = () => wrapper.findAll('[data-test="model-card"] .model-title').map(node => node.text())
    // 三个 Select 依次是分组、平台、排序
    const sortSelect = wrapper.findAll('select')[2]
    expect(sortSelect.attributes('aria-label')).toBe('Sort')

    expect(cardTitles()).toEqual(['Alpha (alpha)', 'Beta (beta)', 'Gamma (gamma)'])

    await sortSelect.setValue('price-asc')
    expect(cardTitles()).toEqual(['Beta (beta)', 'Gamma (gamma)', 'Alpha (alpha)'])

    await sortSelect.setValue('price-desc')
    expect(cardTitles()).toEqual(['Alpha (alpha)', 'Gamma (gamma)', 'Beta (beta)'])

    // 列表视图必须复用同一份排序结果，否则两种视图给出的顺序会不一致
    await wrapper.find('button[title="List view"]').trigger('click')
    const rows = wrapper.findAll('[data-test="model-row"]').map(row => row.text())
    expect(rows[0]).toContain('Alpha (alpha)')
    expect(rows[1]).toContain('Gamma (gamma)')
    expect(rows[2]).toContain('Beta (beta)')
  })

  it('筛选生效时显示计数，空结果可直接清除筛选恢复全量', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('.result-count').exists()).toBe(false)

    const search = wrapper.find('input[type="search"]')
    await search.setValue('custom')
    expect(wrapper.find('.result-count').text()).toBe('Showing 1 / 3')

    await search.setValue('no-such-model')
    const emptyState = wrapper.find('[data-test="empty-state"]')
    expect(emptyState.text()).toContain('No matching models')
    expect(emptyState.text()).toContain('No models match the current filters.')

    const clearButton = emptyState.find('button')
    expect(clearButton.text()).toContain('Clear filters')
    await clearButton.trigger('click')

    expect(wrapper.find('.result-count').exists()).toBe(false)
    expect(wrapper.find('[data-test="empty-state"]').exists()).toBe(false)
    expect(wrapper.findAll('[data-test="model-card"]')).toHaveLength(3)
  })
})
