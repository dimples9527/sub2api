import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AppSidebar from './AppSidebar.vue'
import { useAppStore, useAuthStore } from '@/stores'
import type { User } from '@/types'

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    install: vi.fn(),
    global: {
      t: (key: string) => key,
      locale: { value: 'en' },
      setLocaleMessage: vi.fn(),
    },
  }),
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/dashboard' }),
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings: vi.fn().mockResolvedValue({ custom_menu_items: [] }),
    },
    payment: {
      getConfig: vi.fn().mockResolvedValue({ data: { enabled: false } }),
    },
  },
}))

function makeUser(role: User['role']): User {
  return {
    id: role === 'admin' ? 1 : 2,
    username: role,
    email: `${role}@example.com`,
    role,
    balance: 0,
    concurrency: 0,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-06-29T00:00:00Z',
    updated_at: '2026-06-29T00:00:00Z',
  }
}

function mountSidebar(role: User['role']) {
  const pinia = createPinia()
  setActivePinia(pinia)

  const authStore = useAuthStore()
  authStore.user = makeUser(role)
  authStore.token = 'test-token'

  const appStore = useAppStore()
  appStore.cachedPublicSettings = {
    backend_mode_enabled: false,
    custom_menu_items: [],
    channel_monitor_enabled: true,
    available_channels_enabled: true,
    payment_enabled: true,
    affiliate_enabled: true,
  } as typeof appStore.cachedPublicSettings

  return mount(AppSidebar, {
    global: {
      plugins: [pinia],
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

describe('AppSidebar image generation visibility', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  })

  it('shows image generation for all users', () => {
    const wrapper = mountSidebar('user')

    expect(wrapper.find('a[href="/image"]').exists()).toBe(true)
  })

  it('keeps image generation visible for admins in the personal navigation', () => {
    const wrapper = mountSidebar('admin')

    expect(wrapper.find('a[href="/image"]').exists()).toBe(true)
  })
})
