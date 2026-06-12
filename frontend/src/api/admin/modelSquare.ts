import { apiClient } from '../client'

export interface ModelSquareGroup {
  id: number | string
  name: string
  rate_multiplier?: number
}

export interface ModelSquareModel {
  id: string
  provider?: string
  available?: boolean
  mode?: string
  input_price?: number | string
  output_price?: number | string
  cache_read_price?: number | string
  cache_create_price?: number | string
  group_ids?: Array<number | string>
}

export interface ModelSquarePayload {
  groups?: ModelSquareGroup[]
  models?: ModelSquareModel[]
  data?: {
    groups?: ModelSquareGroup[]
    models?: ModelSquareModel[]
  }
  code?: number
  message?: string
}

export interface AdminModelSquareResult {
  provider_slug: string
  provider_name: string
  provider_type: string
  payload: ModelSquarePayload
}

export async function getModelSquare(): Promise<AdminModelSquareResult> {
  const { data } = await apiClient.get<AdminModelSquareResult>('/model-square')
  return data
}

export const modelSquareAPI = {
  get: getModelSquare
}

export default modelSquareAPI
