import { apiClient } from '../client'

export type ModelSquareConfigModelSource = 'manual' | 'sync'

export interface ModelSquarePlatformModelConfig {
  id: string
  display_name?: string
  source?: ModelSquareConfigModelSource
  input_price?: number | null
  output_price?: number | null
  cache_write_price?: number | null
  cache_write_1h_price?: number | null
  cache_read_price?: number | null
  input_price_priority?: number | null
  output_price_priority?: number | null
  cache_write_price_priority?: number | null
  cache_read_price_priority?: number | null
  image_input_price?: number | null
  image_output_price?: number | null
  per_request_price?: number | null
}

export interface ModelSquarePlatformConfig {
  platform: string
  name?: string
  synced_from_account_id?: number | null
  synced_from_account_name?: string
  synced_at?: string | null
  models: ModelSquarePlatformModelConfig[]
}

export interface ModelSquareConfigPayload {
  platforms: ModelSquarePlatformConfig[]
  updated_at?: string | null
}

export interface ModelSquareOfficialPricing {
  found: boolean
  input_price?: number
  output_price?: number
  cache_write_price?: number
  cache_write_1h_price?: number
  cache_read_price?: number
  input_price_priority?: number
  output_price_priority?: number
  cache_write_price_priority?: number
  cache_read_price_priority?: number
  image_input_price?: number
  image_output_price?: number
}

export async function get(): Promise<ModelSquareConfigPayload> {
  const { data } = await apiClient.get<ModelSquareConfigPayload>('/admin/upstream-management/model-square/config')
  return data
}

export async function update(payload: ModelSquareConfigPayload): Promise<ModelSquareConfigPayload> {
  const { data } = await apiClient.put<ModelSquareConfigPayload>('/admin/upstream-management/model-square/config', payload)
  return data
}

export async function getModelPricing(model: string): Promise<ModelSquareOfficialPricing> {
  const { data } = await apiClient.get<ModelSquareOfficialPricing>('/admin/upstream-management/model-square/model-pricing', {
    params: { model },
  })
  return data
}

export const modelSquareConfigAPI = {
  get,
  update,
  getModelPricing,
}

export default modelSquareConfigAPI
