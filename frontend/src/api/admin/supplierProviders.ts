import { apiClient } from '../client'

export type SupplierNewAPIAuthMode =
  | 'auto'
  | 'cookie_session'
  | 'access_token_refresh'

export interface SupplierProvider {
  id: number
  code: string
  name: string
  provider_type: string
  newapi_auth_mode: SupplierNewAPIAuthMode
  base_url: string
  login_url: string
  api_keys_url: string
  groups_url: string
  available_groups_url: string
  balance_url: string
  usage_cost_url: string
  monitor_url: string
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
  auth_summary: SupplierProviderAuthSummary
  created_at: string
  updated_at: string
}

export interface SupplierProviderAuthSummary {
  login_count: number
  login_success_count: number
  login_failure_count: number
  refresh_count: number
  refresh_success_count: number
  refresh_failure_count: number
  cache_hit_count: number
  cache_miss_count: number
  last_login_at?: string
  last_login_status: string
  last_login_error: string
  last_cache_hit_at?: string
  last_cache_error: string
  last_token_expires_at?: string
  last_token_fingerprint: string
}

export interface SupplierProviderAuthTokenSnapshot {
  status: 'cached' | 'missing' | 'expired' | 'error'
  cached: boolean
  token_type?: string
  token_summary?: string
  token_length?: number
  token_fingerprint?: string
  token_expires_at?: string
  remaining_seconds: number
  ttl_seconds: number
  cookie_present: boolean
  error?: string
}

export interface SupplierProviderAuthLockSnapshot {
  held: boolean
  status: string
  remaining_seconds: number
  error?: string
}

export interface SupplierProviderTokenRefreshResult {
  provider_id: number
  expires_at?: string
  message: string
}
export interface SupplierProviderAuthStatusResult {
  provider_id: number
  summary: SupplierProviderAuthSummary
  cache: SupplierProviderAuthTokenSnapshot
  login_lock: SupplierProviderAuthLockSnapshot
  checked_at: string
}

export type SupplierProviderAuthEventType = 'cache_hit' | 'cache_miss' | 'login_success' | 'login_failed' | 'refresh_success' | 'refresh_failed' | 'cache_invalidated' | 'cache_error'

export interface SupplierProviderAuthHistoryItem {
  id: number
  provider_id: number
  event_type: SupplierProviderAuthEventType
  source: 'sync' | 'endpoint_test' | 'manual' | 'unknown'
  status: string
  started_at: string
  finished_at: string
  duration_ms: number
  http_status?: number
  error_message?: string
  token_fingerprint?: string
  token_expires_at?: string
  token_length?: number
  cookie_present: boolean
  created_at: string
}

export interface SupplierProviderAuthHistoryResult {
  items: SupplierProviderAuthHistoryItem[]
  total: number
  page: number
  page_size: number
}

export interface SupplierProviderAuthHistoryParams {
  page?: number
  page_size?: number
  event_type?: SupplierProviderAuthEventType | ''
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
  newapi_auth_mode: SupplierNewAPIAuthMode
  base_url: string
  login_url?: string
  api_keys_url?: string
  groups_url?: string
  available_groups_url?: string
  balance_url?: string
  usage_cost_url?: string
  monitor_url?: string
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
  deviation?: number   // upstream - local
  deviationPercent?: number
}

export interface SupplierProviderCostBreakdown {
  provider_id: number
  provider_name: string
  provider_type: string
  upstream_cost: number
  local_cost: number
}

export interface SupplierProviderCostTrendResult {
  days: number
  start_date?: string
  end_date?: string
  provider_id?: number
  points: SupplierProviderCostTrendPoint[]
  breakdown: SupplierProviderCostBreakdown[]
}

export interface SupplierProviderBalanceSummaryDay {
  date: string
  balance: number
  cost: number
}

export interface SupplierProviderBalanceHistory {
  first_date: string
  days: number
  total_balance: number
  total_cost: number
}

export interface SupplierProviderBalanceSummary {
  latest_date: string
  today: SupplierProviderBalanceSummaryDay
  previous: SupplierProviderBalanceSummaryDay
  history: SupplierProviderBalanceHistory
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

/** 回填供应商消耗趋势 daily_stats 数据。 */
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

export async function getBalanceSummary(): Promise<SupplierProviderBalanceSummary> {
  const { data } = await apiClient.get<SupplierProviderBalanceSummary>(
    '/admin/supplier-management/providers/balance-summary'
  )
  return data
}

export async function setDefault(id: number): Promise<SupplierProvider> {
  const { data } = await apiClient.put<SupplierProvider>(
    `/admin/supplier-management/providers/${id}/default`
  )
  return data
}

export async function refreshToken(id: number): Promise<SupplierProviderTokenRefreshResult> {
  const { data } = await apiClient.post<SupplierProviderTokenRefreshResult>(
    `/admin/supplier-management/providers/${id}/refresh-token`
  )
  return data
}

export async function getAuthStatus(id: number): Promise<SupplierProviderAuthStatusResult> {
  const { data } = await apiClient.get<SupplierProviderAuthStatusResult>(
    `/admin/supplier-management/providers/${id}/auth-status`
  )
  return data
}

export async function listAuthHistory(id: number, params: SupplierProviderAuthHistoryParams = {}): Promise<SupplierProviderAuthHistoryResult> {
  const { data } = await apiClient.get<SupplierProviderAuthHistoryResult>(
    `/admin/supplier-management/providers/${id}/auth-history`,
    { params }
  )
  return data
}

export const supplierProvidersAPI = {
  list,
  listCostTrends,
  backfillCostTrends,
  getBalanceSummary,
  get,
  create,
  update,
  delete: deleteProvider,
  setDefault,
  refreshToken,
  getAuthStatus,
  listAuthHistory
}

export default supplierProvidersAPI
