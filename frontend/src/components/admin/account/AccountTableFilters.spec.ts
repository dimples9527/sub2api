import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountTableFilters from './AccountTableFilters.vue'
import Select from '@/components/common/Select.vue'
import { i18n } from '@/i18n'

const { listCustomPlatforms } = vi.hoisted(() => ({
  listCustomPlatforms: vi.fn(),
}))

vi.mock('@/api/admin/customPlatforms', () => ({
  customPlatformsAPI: {
    list: listCustomPlatforms,
  },
}))

describe('AccountTableFilters', () => {
  beforeEach(() => {
    listCustomPlatforms.mockReset()
  })

  it('shows enabled custom platforms in the platform filter', async () => {
    listCustomPlatforms.mockResolvedValue([
      { id: 1, code: 'provider_x', name: 'Provider X', color: '#111111', enabled: true, sort_order: 1, created_at: '', updated_at: '' },
      { id: 2, code: 'disabled_x', name: 'Disabled', color: '#222222', enabled: false, sort_order: 2, created_at: '', updated_at: '' },
    ])

    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: { platform: '', type: '', status: '', privacy_mode: '', group: '' },
      },
      global: { plugins: [i18n] },
    })

    await flushPromises()

    expect(listCustomPlatforms).toHaveBeenCalledWith(true)
    expect(wrapper.findAllComponents(Select)[0].props('options')).toContainEqual({
      value: 'provider_x',
      label: 'Provider X',
    })
    expect(wrapper.findAllComponents(Select)[0].props('options')).not.toContainEqual({
      value: 'disabled_x',
      label: 'Disabled',
    })
  })
})
