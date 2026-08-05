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
    })
    captchaSettingsMocks.update.mockResolvedValue({
      provider: '2captcha',
      api_key_configured: true,
      endpoint: 'https://api.2captcha.com',
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
})
