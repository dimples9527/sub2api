import { apiClient } from '../client'

export type SupplierAccountHealthStatus = 'healthy' | 'slow' | 'failed' | 'unavailable'
export type SupplierAccountHealthRange = '24h' | '7d' | '30d'

export interface SupplierAccountHealthAccount {
  local_account_id: number
  local_account_name: string
  provider_id: number
  provider_name: string
  platform: string
  schedulable: boolean
  status?: SupplierAccountHealthStatus | string
  checked_at?: string
  latency_ms?: number | null
  latency_limit_ms: number
  consecutive_failures: number
  upstream_rate_multiplier: number
  effective_rate_multiplier: number
  guard_enabled: boolean
}

export interface SupplierAccountHealthPoint {
  checked_at: string
  bucket_end_at?: string
  latest_checked_at?: string
  status: SupplierAccountHealthStatus | string
  latency_ms?: number | null
  latency_limit_ms: number
  sample_count: number
  healthy_count: number
  slow_count: number
  failed_count: number
  reason?: string
  action?: string
  error_message?: string
}

export interface SupplierAccountHealthAccountListResult {
  items: SupplierAccountHealthAccount[]
  total: number
  page: number
  page_size: number
}

export interface SupplierAccountHealthSummary {
  total: number
  healthy: number
  slow: number
  failed: number
  unchecked: number
}

export interface SupplierAccountHealthUpstreamMonitor {
  target_id: number
  provider_id: number
  provider_name: string
  monitor_key: string
  monitor_name: string
  primary_model: string
  availability_7d: number
  last_seen_at?: string
}

export interface SupplierAccountHealthTrend {
  account_id: number
  range: SupplierAccountHealthRange | string
  points: SupplierAccountHealthPoint[]
  latest?: SupplierAccountHealthPoint
  upstream_points?: SupplierAccountHealthPoint[]
  upstream_latest?: SupplierAccountHealthPoint
  upstream_monitors?: SupplierAccountHealthUpstreamMonitor[]
}

export interface SupplierAccountHealthAccountListParams {
  provider_id?: number
  platform?: string
  search?: string
  health_status?: SupplierAccountHealthStatus | string
  page?: number
  page_size?: number
}

export async function listSupplierAccountHealthAccounts(
  params: SupplierAccountHealthAccountListParams = {}
): Promise<SupplierAccountHealthAccountListResult> {
  const { data } = await apiClient.get<SupplierAccountHealthAccountListResult>(
    '/admin/supplier-management/account-health/accounts',
    { params }
  )
  return data
}

export async function getSupplierAccountHealthSummary(
  params: Pick<SupplierAccountHealthAccountListParams, 'provider_id' | 'platform' | 'search'> = {}
): Promise<SupplierAccountHealthSummary> {
  const { data } = await apiClient.get<SupplierAccountHealthSummary>(
    '/admin/supplier-management/account-health/summary',
    { params }
  )
  return data
}

export async function getSupplierAccountHealthTrend(
  accountId: number,
  range: SupplierAccountHealthRange = '24h'
): Promise<SupplierAccountHealthTrend> {
  const { data } = await apiClient.get<SupplierAccountHealthTrend>(
    '/admin/supplier-management/account-health/trend',
    { params: { account_id: accountId, range } }
  )
  return data
}

export async function getSupplierAccountHealthTrends(
  accountIds: number[],
  range: SupplierAccountHealthRange = '24h'
): Promise<{ items: SupplierAccountHealthTrend[] }> {
  const { data } = await apiClient.get<{ items: SupplierAccountHealthTrend[] }>(
    '/admin/supplier-management/account-health/trends',
    { params: { ids: accountIds.join(','), range } }
  )
  return data
}

export const supplierAccountHealthAPI = {
  listAccounts: listSupplierAccountHealthAccounts,
  getSummary: getSupplierAccountHealthSummary,
  getTrend: getSupplierAccountHealthTrend,
  getTrends: getSupplierAccountHealthTrends,
}

export default supplierAccountHealthAPI
