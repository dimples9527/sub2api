import { apiClient } from '../client'

export interface SupplierProviderRecharge {
  id: number
  provider_id: number
  provider_name: string
  provider_type: string
  external_id: string
  external_code: string
  recharge_type: string
  amount: number
  status: string
  occurred_at: string
  description: string
  synced_at: string
  created_at?: string
  updated_at?: string
}

export interface SupplierProviderRechargeListParams {
  provider_id?: number
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}

export interface SupplierProviderRechargeListResult {
  items: SupplierProviderRecharge[]
  total: number
  total_amount: number
  page: number
  page_size: number
}

export interface SupplierProviderRechargeSyncResult {
  provider_id?: number
  provider_name?: string
  status: string
  message?: string
  record_count?: number
  synced_at?: string
  success_count?: number
  failed_count?: number
  items?: SupplierProviderRechargeSyncResult[]
}

function compactParams(params: SupplierProviderRechargeListParams): Record<string, unknown> | undefined {
  const entries = Object.entries(params).filter(([, value]) => value !== undefined && value !== '')
  return entries.length > 0 ? Object.fromEntries(entries) : undefined
}

export async function listSupplierProviderRecharges(
  params: SupplierProviderRechargeListParams = {}
): Promise<SupplierProviderRechargeListResult> {
  const { data } = await apiClient.get<SupplierProviderRechargeListResult>(
    '/admin/supplier-management/recharges',
    { params: compactParams(params) }
  )
  return data
}

export async function syncSupplierProviderRecharges(
  providerId?: number,
  fullSync = true
): Promise<SupplierProviderRechargeSyncResult> {
  const { data } = await apiClient.post<SupplierProviderRechargeSyncResult>(
    '/admin/supplier-management/recharges/sync',
    { provider_id: providerId || 0, full_sync: fullSync }
  )
  return data
}

export const supplierProviderRechargesAPI = {
  list: listSupplierProviderRecharges,
  sync: syncSupplierProviderRecharges,
}

export default supplierProviderRechargesAPI