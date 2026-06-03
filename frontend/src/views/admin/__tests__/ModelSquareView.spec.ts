import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ModelSquareView from '../ModelSquareView.vue'

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
  },
}))

describe('ModelSquareView', () => {
  it('shows the lowest-rate groups first so displayed groups match price calculation', async () => {
    getMock.mockResolvedValueOnce({
      data: {
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
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()

    const cardText = wrapper.find('article').text()
    const proIndex = cardText.indexOf('codex pro')
    const fallbackIndex = cardText.indexOf('codex fallback')
    const welfareIndex = cardText.indexOf('codex welfare')

    expect(proIndex).toBeGreaterThanOrEqual(0)
    expect(fallbackIndex).toBeGreaterThan(proIndex)
    expect(welfareIndex).toBeGreaterThan(fallbackIndex)
    expect(cardText).toContain('$0.14')
  })
})
