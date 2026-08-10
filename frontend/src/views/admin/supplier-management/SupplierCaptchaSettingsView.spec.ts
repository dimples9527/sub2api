import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SupplierCaptchaSettingsView from './SupplierCaptchaSettingsView.vue'

const captchaSettingsMocks = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin/supplierCaptchaSettings', () => ({
  supplierCaptchaSettingsAPI: {
    get: captchaSettingsMocks.get,
    update: captchaSettingsMocks.update,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: captchaSettingsMocks.showSuccess,
    showError: captchaSettingsMocks.showError,
  }),
}))

async function mountCaptchaSettingsView() {
  const wrapper = mount(SupplierCaptchaSettingsView, {
    global: {
      stubs: {
        SupplierModuleLayout: { template: '<div><slot /></div>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('SupplierCaptchaSettingsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    captchaSettingsMocks.get.mockResolvedValue({
      provider: '2captcha',
      api_key_configured: false,
      endpoint: '',
      call_total: 0,
      call_success: 0,
      call_failed: 0,
      last_called_at: '',
    })
    captchaSettingsMocks.update.mockResolvedValue({
      provider: '2captcha',
      api_key_configured: true,
      endpoint: 'https://api.2captcha.com',
      call_total: 3,
      call_success: 2,
      call_failed: 1,
      last_called_at: '2026-08-10T08:00:00Z',
    })
  })

  it('uses the global success toast after saving captcha settings', async () => {
    const wrapper = await mountCaptchaSettingsView()

    await wrapper.find('button.sp-button.primary').trigger('click')
    await flushPromises()

    expect(captchaSettingsMocks.update).toHaveBeenCalledWith({
      provider: '2captcha',
      api_key: undefined,
      endpoint: '',
    })
    expect(captchaSettingsMocks.showSuccess).toHaveBeenCalledWith('\u6253\u7801\u914d\u7f6e\u5df2\u4fdd\u5b58')
    expect(wrapper.find('.sp-success-line').exists()).toBe(false)
  })


  it('renders captcha call stats from settings response', async () => {
    captchaSettingsMocks.get.mockResolvedValue({
      provider: '2captcha',
      api_key_configured: true,
      endpoint: '',
      call_total: 12,
      call_success: 10,
      call_failed: 2,
      last_called_at: '2026-08-10T08:00:00Z',
    })

    const wrapper = await mountCaptchaSettingsView()

    expect(wrapper.text()).toContain('累计调用')
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('10')
    expect(wrapper.text()).toContain('2')
  })
})
