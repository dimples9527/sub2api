import { apiClient } from '../client'

export type ModelMonitorGroupHealthRange = '1h' | '24h' | '7d' | '30d'

export type ModelMonitorGroupHealthStatus =
  | 'healthy'
  | 'warning'
  | 'critical'
  | 'low_sample'
  | 'no_data'

export interface ModelMonitorGroupHealthPoint {
  time: string
  request_count: number
  success_count: number
  error_count: number
  service_error_count: number
  business_limited_count: number
  success_rate: number
  service_success_rate: number
  avg_latency_ms: number
  p95_latency_ms: number
}

export interface ModelMonitorGroupHealthErrorItem {
  category: string
  count: number
}

export interface ModelMonitorGroupHealthItem {
  group_id: number
  group_name: string
  platform: string
  effective_platform: string
  request_count: number
  success_count: number
  error_count: number
  business_limited_count: number
  service_error_count: number
  success_rate: number
  service_success_rate: number
  error_rate: number
  avg_latency_ms: number
  p95_latency_ms: number
  p95_first_token_ms: number
  status: ModelMonitorGroupHealthStatus | string
  last_request_at: string | null
  trend: ModelMonitorGroupHealthPoint[]
  top_errors: ModelMonitorGroupHealthErrorItem[]
}

export type ModelMonitorGroupHealthResponse = ModelMonitorGroupHealthItem[]

export interface ModelMonitorGroupHealthQuery {
  range?: ModelMonitorGroupHealthRange
  groupIds?: number[]
  platform?: string
}

export async function getModelMonitorGroupHealth(
  query: ModelMonitorGroupHealthQuery = {},
): Promise<ModelMonitorGroupHealthResponse> {
  const params: Record<string, string> = {}
  if (query.range) params.range = query.range
  if (query.groupIds?.length) params.group_ids = query.groupIds.join(',')
  if (query.platform?.trim()) params.platform = query.platform.trim()

  const { data } = await apiClient.get<ModelMonitorGroupHealthResponse>('/admin/model-monitor/group-health', { params })
  return data
}

export const modelMonitorGroupHealthAPI = {
  get: getModelMonitorGroupHealth,
}

export default modelMonitorGroupHealthAPI
