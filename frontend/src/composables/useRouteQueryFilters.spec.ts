import { nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { route, replaceMock } = vi.hoisted(() => ({
  route: { query: { provider: 'main', rateRisk: 'true' } as Record<string, string> },
  replaceMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ replace: replaceMock }),
}))

import { useRouteQueryFilters } from './useRouteQueryFilters'

describe('useRouteQueryFilters', () => {
  beforeEach(() => replaceMock.mockReset())

  it('hydrates filters and writes changed values back to the route', async () => {
    const provider = ref('')
    const rate = ref('')
    useRouteQueryFilters([
      { queryKey: 'provider', state: provider },
      { queryKey: 'rateRisk', state: rate, fromQuery: value => value === 'true' ? 'risk' : '', toQuery: value => value === 'risk' ? 'true' : undefined },
    ])

    expect(provider.value).toBe('main')
    expect(rate.value).toBe('risk')

    provider.value = 'backup'
    await nextTick()
    expect(replaceMock).toHaveBeenCalledWith({ query: { provider: 'backup', rateRisk: 'true' } })
  })
})
