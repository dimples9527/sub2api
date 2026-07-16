import { apiClient } from '../client'

export interface SupplierProvider {
  id: number
  code: string
  name: string
  provider_type: string
  base_url: string
  login_url: string
  api_keys_url: string
  groups_url: string
  available_groups_url: string
  balance_url: string
  usage_cost_url: string
  account_name_prefix: string
  temp_disable_minutes: number
  account_rate_multiplier_scale: number
  sort_order: number
  enabled: boolean
  is_default: boolean
  email: string
  username: string
  credential_configured: boolean
  status: string
  risk_level: string
  valid_account_count: number
  schedulable_account_count: number
  request_count: number
  success_rate: number
  period_cost: number
  current_balance: number
  today_cost: number
  estimated_days?: number
  rate_risk_count: number
  sync_status: string
  sync_message: string
  last_sync_at?: string
  created_at: string
  updated_at: string
}

export interface SupplierProviderSummary {
  total_count: number
  enabled_count: number
  high_risk_count: number
  low_balance_count: number
  sync_failure_count: number
  rate_risk_count: number
}

export interface SupplierProviderListResult {
  items: SupplierProvider[]
  summary: SupplierProviderSummary
  total: number
  page: number
  page_size: number
}

export interface SupplierProviderUpsertPayload {
  code: string
  name: string
  provider_type: string
  base_url: string
  login_url?: string
  api_keys_url?: string
  groups_url?: string
  available_groups_url?: string
  balance_url?: string
  usage_cost_url?: string
  email?: string
  username?: string
  password?: string
  account_name_prefix?: string
  temp_disable_minutes?: number
  account_rate_multiplier_scale: number
  sort_order?: number
  enabled: boolean
  is_default?: boolean
}

export interface SupplierProviderListParams {
  search?: string
  enabled?: boolean
  page?: number
  page_size?: number
}

export async function list(params: SupplierProviderListParams = {}): Promise<SupplierProviderListResult> {
  const { data } = await apiClient.get<SupplierProviderListResult>(
    '/admin/supplier-management/providers',
    { params }
  )
  return data
}

export async function get(id: number): Promise<SupplierProvider> {
  const { data } = await apiClient.get<SupplierProvider>(
    `/admin/supplier-management/providers/${id}`
  )
  return data
}

export async function create(payload: SupplierProviderUpsertPayload): Promise<SupplierProvider> {
  const { data } = await apiClient.post<SupplierProvider>(
    '/admin/supplier-management/providers',
    payload
  )
  return data
}

export async function update(
  id: number,
  payload: SupplierProviderUpsertPayload
): Promise<SupplierProvider> {
  const { data } = await apiClient.put<SupplierProvider>(
    `/admin/supplier-management/providers/${id}`,
    payload
  )
  return data
}

export async function deleteProvider(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/supplier-management/providers/${id}`
  )
  return data
}

export async function setDefault(id: number): Promise<SupplierProvider> {
  const { data } = await apiClient.put<SupplierProvider>(
    `/admin/supplier-management/providers/${id}/default`
  )
  return data
}

export const supplierProvidersAPI = {
  list,
  get,
  create,
  update,
  delete: deleteProvider,
  setDefault
}

export default supplierProvidersAPI
