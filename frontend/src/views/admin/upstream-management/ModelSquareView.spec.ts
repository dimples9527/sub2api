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

vi.mock('@/api/admin', () => ({
  adminAPI: { modelSquare: { get: getMock } },
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
        EmptyState: { props: ['title', 'description'], template: '<div data-test="empty-state">{{ title }} {{ description }}</div>' },
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

  it('renders configured display names, providers and configured prices without new price styles', async () => {
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
    expect(wrapper.find('.price-box-neutral').exists()).toBe(true)
    expect(wrapper.find('.price-box-blue').exists()).toBe(true)
    expect(wrapper.find('.price-box-violet').exists()).toBe(true)
    expect(wrapper.find('.price-box-amber').exists()).toBe(false)
    expect(wrapper.find('.price-box-cyan').exists()).toBe(false)
    expect(wrapper.find('.price-box-emerald').exists()).toBe(false)
    expect(wrapper.find('.table-price-chip').exists()).toBe(false)
  })

  it('opens model details and recalculates prices when switching groups', async () => {
    const wrapper = mountView()
    await flushPromises()

    const openAICard = wrapper.findAll('[data-test="model-card"]')
      .find(card => card.text().includes('GPT-5.5 Flagship'))
    await openAICard?.find('.model-detail-button').trigger('click')

    expect(wrapper.find('[data-test="dialog"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="dialog"]').text()).toContain('Premium Group')
    expect(wrapper.find('[data-test="dialog"]').text()).toContain('$5')

    const groupOptions = wrapper.findAll('[data-test="detail-group-option"]')
    const defaultGroup = groupOptions.find(option => option.text().includes('Default Group'))
    await defaultGroup?.trigger('click')

    const dialogText = wrapper.find('[data-test="dialog"]').text()
    expect(dialogText).toContain('Default Group')
    expect(dialogText).toContain('$10')
    expect(dialogText).toContain('1x')
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

  it('does not render missing prices as zero while preserving explicit zero prices', async () => {
    const wrapper = mountView()
    await flushPromises()

    const cards = wrapper.findAll('[data-test="model-card"]')
    const customCard = cards.find(card => card.text().includes('Custom Model'))
    expect(customCard?.text()).toContain('$0')
    expect(customCard?.text()).toContain('\u672a\u8bbe\u7f6e')
    expect(customCard?.text()).not.toContain('$0.000')
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
})
