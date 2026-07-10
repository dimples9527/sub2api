import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ModelSquareView from '../ModelSquareView.vue'

const { getModelSquareMock, showErrorMock, showSuccessMock } = vi.hoisted(() => ({
  getModelSquareMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const labels: Record<string, string> = {
          'admin.modelSquare.defaultProvider': 'Default upstream',
          'admin.modelSquare.modelCount': 'Models',
          'admin.modelSquare.availableCount': 'Available',
          'admin.modelSquare.groupCount': 'Groups',
          'admin.modelSquare.providerSummary': `${params?.count ?? 0} model(s) · Rate ${params?.rate ?? ''}`,
          'admin.modelSquare.unknownProvider': 'Unknown provider',
          'admin.modelSquare.moreGroups': 'More',
          'admin.modelSquare.inputPrice': 'Input',
          'admin.modelSquare.outputPrice': 'Output',
          'admin.modelSquare.cacheReadPrice': 'Cache read',
          'admin.modelSquare.cacheWritePrice': 'Cache write',
          'admin.modelSquare.perMillionTokens': '$/M tokens',
          'admin.modelSquare.available': 'Available',
          'admin.modelSquare.unavailable': 'Unavailable',
          'admin.modelSquare.copied': 'Copied',
          'admin.modelSquare.unnamedModel': 'Unnamed model',
          'admin.modelSquare.groupDialogTitle': `${params?.id ?? ''} groups`,
          'admin.modelSquare.modes.chat': 'Chat',
          'admin.modelSquare.modes.image': 'Image',
          'admin.modelSquare.modes.embedding': 'Embedding',
          'admin.modelSquare.modes.responses': 'Responses',
          'common.refresh': 'Refresh',
          'common.close': 'Close',
        }
        return labels[key] ?? key
      },
    }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock,
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    modelSquare: {
      get: getModelSquareMock,
    },
  },
}))

function mountView() {
  return mount(ModelSquareView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
        BaseDialog: { template: '<div v-if="show"><slot /><slot name="footer" /></div>', props: ['show'] },
        EmptyState: { template: '<div />' },
        Icon: { template: '<span />' },
      },
    },
  })
}

describe('Upstream ModelSquareView', () => {
  beforeEach(() => {
    getModelSquareMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
  })

  it('loads the model square and prices models with the lowest group rate', async () => {
    getModelSquareMock.mockResolvedValueOnce({
      provider_slug: 'default-sub2api',
      provider_name: 'Default Sub2API',
      provider_type: 'sub2api',
      payload: {
        groups: [
          { id: 2, name: 'codex welfare', rate_multiplier: 0.18 },
          { id: 8, name: 'codex stable', rate_multiplier: 0.4 },
          { id: 71, name: 'codex pro', rate_multiplier: 0.08 },
          { id: 86, name: 'codex fallback', rate_multiplier: 0.15 },
        ],
        models: [
          {
            id: 'gpt-5.2',
            provider: 'openai',
            input_price: 1.75,
            output_price: 14,
            cache_read_price: 0.175,
            cache_create_price: 0,
            mode: 'chat',
            available: true,
            group_ids: [2, 8, 71, 86],
          },
        ],
      },
    })

    const wrapper = mountView()

    await flushPromises()

    expect(getModelSquareMock).toHaveBeenCalledOnce()

    const cardText = wrapper.find('[data-test="model-card"]').text()
    expect(cardText).toContain('codex pro')
    expect(cardText).toContain('0.08x')
    expect(cardText).toContain('+3')
    expect(cardText).toContain('$0.14')
  })

  it('hides upstream identity text and exposes an overflow count for extra groups', async () => {
    getModelSquareMock.mockResolvedValueOnce({
      provider_slug: 'findcg',
      provider_name: 'FindCG',
      provider_type: 'sub2api',
      payload: {
        groups: [
          { id: 1, name: 'vip', rate_multiplier: 0.4 },
          { id: 2, name: 'pro', rate_multiplier: 0.2 },
          { id: 3, name: 'basic', rate_multiplier: 0.6 },
          { id: 4, name: 'trial', rate_multiplier: 0.8 },
        ],
        models: [{
          id: 'gpt-5.2',
          provider: 'openai',
          available: true,
          mode: 'chat',
          input_price: 1,
          output_price: 2,
          cache_read_price: 0.1,
          cache_create_price: 0,
          group_ids: [1, 2, 3, 4],
        }],
      },
    })

    const wrapper = mountView()

    await flushPromises()

    const text = wrapper.text()
    expect(text).not.toContain('Default upstream')
    expect(text.toLowerCase()).not.toContain('findcg')
    expect(wrapper.find('[data-test="model-card"]').text()).toContain('+3')
  })
})
