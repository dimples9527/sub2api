import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import UpstreamProvidersView from './UpstreamProvidersView.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const { adminAPIMock } = vi.hoisted(() => ({
  adminAPIMock: {
    upstreamProviders: {
      list: vi.fn().mockResolvedValue([]),
    },
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: adminAPIMock,
}))

vi.mock('@/api/admin/index', () => ({
  adminAPI: {
    upstreamProviders: adminAPIMock.upstreamProviders,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

describe('UpstreamProvidersView', () => {
  it('accepts 0.1 as an account rate conversion factor', async () => {
    const wrapper = mount(UpstreamProvidersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
          DataTable: { template: '<div />' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>',
          },
          ConfirmDialog: true,
          EmptyState: true,
          Icon: true,
        },
      },
    })

    const createButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.upstreamProviders.createProvider'))
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')

    const input = wrapper.find('input[type="number"][required]')
    expect(input.exists()).toBe(true)

    const element = input.element as HTMLInputElement
    element.value = '0.1'

    expect(element.checkValidity()).toBe(true)
  })
})
