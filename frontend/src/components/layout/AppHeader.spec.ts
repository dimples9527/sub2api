import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import AppHeader from './AppHeader.vue'
import { useAppStore, useAuthStore } from '@/stores'
import type { User } from '@/types'

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    install: vi.fn(),
    global: { t: (key: string) => key, locale: { value: 'en' }, setLocaleMessage: vi.fn() },
  }),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('vue-router', () => ({
  useRoute: () => ({ meta: {}, params: {} }),
  useRouter: () => ({ push: vi.fn() }),
}))

function makeAdmin(): User {
  return {
    id: 1, username: 'admin', email: 'admin@example.com', role: 'admin',
    balance: 0, concurrency: 0, status: 'active', allowed_groups: null,
    balance_notify_enabled: false, balance_notify_threshold: null,
    balance_notify_extra_emails: [], created_at: '2026-07-11T00:00:00Z',
    updated_at: '2026-07-11T00:00:00Z',
  }
}

describe('AppHeader admin tools', () => {
  it('always shows the model monitor entry without an API URL setting', () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const authStore = useAuthStore()
    authStore.user = makeAdmin()
    authStore.token = 'test-token'
    const appStore = useAppStore()
    appStore.cachedPublicSettings = { custom_menu_items: [] } as typeof appStore.cachedPublicSettings

    const wrapper = mount(AppHeader, {
      global: {
        plugins: [pinia],
        stubs: {
          AnnouncementBell: true, LocaleSwitcher: true, SubscriptionProgressMini: true,
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
          VersionBadge: true,
        },
      },
    })

    expect(wrapper.find('a[href="/model-monitor.html"]').exists()).toBe(true)
  })

  it('keeps version downloads beside the model monitor entry', () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const authStore = useAuthStore()
    authStore.user = makeAdmin()
    authStore.token = 'test-token'
    const appStore = useAppStore()
    appStore.siteVersion = '0.1.151'
    appStore.cachedPublicSettings = {
      llm_monitor_status_api_url: 'https://status.example.com', custom_menu_items: [],
    } as typeof appStore.cachedPublicSettings
    const wrapper = mount(AppHeader, {
      global: {
        plugins: [pinia],
        stubs: {
          AnnouncementBell: true, LocaleSwitcher: true, SubscriptionProgressMini: true,
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
          VersionBadge: { props: ['version'], template: '<div data-testid="header-version-badge">{{ version }}</div>' },
        },
      },
    })
    expect(wrapper.find('a[href="/model-monitor.html"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="header-version-badge"]').text()).toBe('0.1.151')
  })
})
