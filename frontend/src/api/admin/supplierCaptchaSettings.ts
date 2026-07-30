import { apiClient } from '../client'

export interface SupplierCaptchaSettings {
  provider: string
  api_key_configured: boolean
  endpoint: string
}

export interface UpdateSupplierCaptchaSettingsRequest {
  provider: string
  api_key?: string
  endpoint?: string
  clear_api_key?: boolean
}

export const supplierCaptchaSettingsAPI = {
  async get(): Promise<SupplierCaptchaSettings> {
    const { data } = await apiClient.get<SupplierCaptchaSettings>(
      '/admin/supplier-management/captcha-settings',
    )
    return data
  },

  async update(payload: UpdateSupplierCaptchaSettingsRequest): Promise<SupplierCaptchaSettings> {
    const { data } = await apiClient.put<SupplierCaptchaSettings>(
      '/admin/supplier-management/captcha-settings',
      payload,
    )
    return data
  },
}
