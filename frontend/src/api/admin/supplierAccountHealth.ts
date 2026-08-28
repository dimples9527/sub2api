import { apiClient } from '../client'

export type SupplierAccountHealthStatus = 'healthy' | 'slow' | 'failed'
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
  rate_multiplier: number
  guard_enabled: boolean
}

export interface SupplierAccountHealthPoint {
  checked_at: string
  status: SupplierAccountHealthStatus | string
  latency_ms?: number | null
  latency_limit_ms: number
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

export interface SupplierAccountHealthTrend {
  account_id: number
  range: SupplierAccountHealthRange | string
  points: SupplierAccountHealthPoint[]
  latest?: SupplierAccountHealthPoint
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

export const supplierAccountHealthAPI = {
  listAccounts: listSupplierAccountHealthAccounts,
  getTrend: getSupplierAccountHealthTrend,
}

export default supplierAccountHealthAPI
