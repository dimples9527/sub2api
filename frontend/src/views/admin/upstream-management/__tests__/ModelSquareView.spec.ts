import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ModelSquareView from '../ModelSquareView.vue'

const { getModelSquareMock } = vi.hoisted(() => ({
  getModelSquareMock: vi.fn(),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (!params) return key
        return `${key} ${JSON.stringify(params)}`
      },
    }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    modelSquare: {
      get: getModelSquareMock,
    },
  },
}))

describe('Upstream ModelSquareView', () => {
  it('loads the default upstream model square and prices models with the lowest group rate', async () => {
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

    const wrapper = mount(ModelSquareView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          Icon: { template: '<span />' },
          EmptyState: { template: '<div />' },
        },
      },
    })

    await flushPromises()

    expect(getModelSquareMock).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('Default Sub2API')

    const cardText = wrapper.find('[data-test="model-card"]').text()
    const proIndex = cardText.indexOf('codex pro')
    const fallbackIndex = cardText.indexOf('codex fallback')
    const welfareIndex = cardText.indexOf('codex welfare')

    expect(proIndex).toBeGreaterThanOrEqual(0)
    expect(fallbackIndex).toBeGreaterThan(proIndex)
    expect(welfareIndex).toBeGreaterThan(fallbackIndex)
    expect(cardText).toContain('$0.14')
  })
})
