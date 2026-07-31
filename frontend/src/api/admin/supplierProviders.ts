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
  turnstile_enabled: boolean
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
  turnstile_enabled: boolean
  is_default?: boolean
}


export interface SupplierProviderCostTrendPoint {
  date: string
  upstream_cost: number
  local_cost: number
}

export interface SupplierProviderCostTrendResult {
  days: number
  start_date?: string
  end_date?: string
  provider_id?: number
  points: SupplierProviderCostTrendPoint[]
}

export interface SupplierProviderCostTrendParams {
  days?: number
  start_date?: string
  end_date?: string
  provider_id?: number
}

export interface SupplierProviderCostBackfillParams {
  start_date: string
  end_date: string
  provider_id?: number
}

export interface SupplierProviderCostBackfillItem {
  provider_id: number
  provider_name: string
  provider_type: string
  date: string
  status: string
  cost?: number
  message?: string
}

export interface SupplierProviderCostBackfillResult {
  start_date: string
  end_date: string
  provider_id?: number
  provider_count: number
  day_count: number
  success_count: number
  failed_count: number
  skipped_count: number
  items: SupplierProviderCostBackfillItem[]
  started_at: string
  finished_at?: string
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


export async function listCostTrends(
  params: number | SupplierProviderCostTrendParams = 14
): Promise<SupplierProviderCostTrendResult> {
  let query: Record<string, string | number>
  if (typeof params === 'number') {
    query = { days: params }
  } else if (params.start_date && params.end_date) {
    query = {
      start_date: params.start_date,
      end_date: params.end_date,
      ...(params.provider_id && params.provider_id > 0 ? { provider_id: params.provider_id } : {}),
    }
  } else {
    query = {
      days: params.days ?? 14,
      ...(params.provider_id && params.provider_id > 0 ? { provider_id: params.provider_id } : {}),
    }
  }
  const { data } = await apiClient.get<SupplierProviderCostTrendResult>(
    '/admin/supplier-management/providers/cost-trends',
    { params: query }
  )
  return data
}

/** ??????????????? daily_stats?????????? */
export async function backfillCostTrends(
  params: SupplierProviderCostBackfillParams
): Promise<SupplierProviderCostBackfillResult> {
  const body: Record<string, string | number> = {
    start_date: params.start_date,
    end_date: params.end_date,
  }
  if (params.provider_id && params.provider_id > 0) {
    body.provider_id = params.provider_id
  }
  const { data } = await apiClient.post<SupplierProviderCostBackfillResult>(
    '/admin/supplier-management/providers/cost-trends/backfill',
    body
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
  listCostTrends,
  backfillCostTrends,
  get,
  create,
  update,
  delete: deleteProvider,
  setDefault
}

export default supplierProvidersAPI
