/**
 * 供应商运维驾驶舱只读 API。
 * overview 继续走旧 /dashboard；accounts/rates/providers 走新 detail 接口。
 */

import { apiClient } from '../client'

export type SupplierDashboardRange = '24h' | '7d' | '30d'
export type SupplierDashboardSeverity = 'critical' | 'high' | 'medium' | 'low'
export type SupplierDashboardRiskType =
  | 'all'
  | 'critical'
  | 'traffic'
  | 'rate_up'
  | 'not_lowest'
  | 'balance'
  | 'sync'
  | 'task'
export type SupplierDashboardRateView = 'risk' | 'changed' | 'all'
export type SupplierDashboardProviderStatus =
  | 'healthy'
  | 'warning'
  | 'high_risk'
  | 'disabled'
  | 'unknown'
export type SupplierDashboardComparisonStatus =
  | 'lowest'
  | 'tied_lowest'
  | 'not_lowest'
  | 'missing_group'
  | 'insufficient_accounts'
  | 'unknown'

export interface SupplierDashboardWarning {
  source: string
  message: string
}

export interface SupplierDashboardOverviewResponse {
  range: SupplierDashboardRange
  summary?: Record<string, unknown>
  stability?: Record<string, unknown>
  cost?: Record<string, unknown>
  issues?: Array<Record<string, unknown>>
  tasks?: Array<Record<string, unknown>>
  provider_rankings?: Array<Record<string, unknown>>
  model_rankings?: Array<Record<string, unknown>>
  trends?: Array<Record<string, unknown>>
  warnings?: SupplierDashboardWarning[]
  generated_at: string
}

export interface SupplierDashboardAccountItem {
  account_id: number
  account_name: string
  provider_slug: string
  provider_name: string
  group_key: string
  group_name: string
  severity: SupplierDashboardSeverity
  risk_types: SupplierDashboardRiskType[]
  request_count: number | null
  success_rate: number | null
  current_rate: number | null
  lowest_rate: number | null
  rate_delta_percent: number | null
  balance: number | null
  balance_currency: string | null
  estimated_days: number | null
  status: string
  reason: string
  period_cost: number | null
  estimated_extra_cost: number | null
  traffic_impact: number
  detected_at: string
  target_path: string
}

export interface SupplierDashboardRateItem {
  provider_slug: string
  provider_name: string
  group_key: string
  group_name: string
  enabled_account_count: number
  current_account_id: number
  current_account_name: string
  current_rate: number | null
  lowest_rate: number | null
  lowest_account_ids: number[]
  lowest_account_names: string[]
  rate_delta_percent: number | null
  estimated_extra_cost: number | null
  cost_currency: string | null
  comparison_status: SupplierDashboardComparisonStatus
  last_synced_at: string | null
  target_path: string
}

export interface SupplierDashboardProviderItem {
  provider_slug: string
  provider_name: string
  enabled: boolean
  status: SupplierDashboardProviderStatus
  critical_issue_count: number | null
  enabled_account_count: number
  schedulable_account_count: number
  request_count: number | null
  success_rate: number | null
  period_cost: number | null
  cost_currency: string | null
  balance: number | null
  balance_currency: string | null
  estimated_days: number | null
  rate_risk_count: number
  balance_risk: boolean
  sync_risk: boolean
  target_path: string
}

export interface SupplierDashboardAccountsResponse {
  range: SupplierDashboardRange
  items: SupplierDashboardAccountItem[]
  total: number
  page: number
  page_size: number
  warnings: SupplierDashboardWarning[]
  generated_at: string
}

export interface SupplierDashboardRatesResponse {
  range: SupplierDashboardRange
  items: SupplierDashboardRateItem[]
  total: number
  page: number
  page_size: number
  warnings: SupplierDashboardWarning[]
  generated_at: string
}

export interface SupplierDashboardProvidersResponse {
  range: SupplierDashboardRange
  items: SupplierDashboardProviderItem[]
  total: number
  page: number
  page_size: number
  warnings: SupplierDashboardWarning[]
  generated_at: string
}

export interface SupplierDashboardAccountsQuery {
  range?: SupplierDashboardRange
  risk_type?: SupplierDashboardRiskType
  provider_slug?: string
  group_key?: string
  page?: number
  page_size?: number
}

export interface SupplierDashboardRatesQuery {
  range?: SupplierDashboardRange
  view?: SupplierDashboardRateView
  comparison_status?: SupplierDashboardComparisonStatus | ''
  provider_slug?: string
  group_key?: string
  page?: number
  page_size?: number
}

export interface SupplierDashboardProvidersQuery {
  range?: SupplierDashboardRange
  status?: SupplierDashboardProviderStatus | ''
  page?: number
  page_size?: number
}

export interface SupplierDashboardTrafficPoint {
  time: string
  requests: number
  tokens: number
}

export interface SupplierDashboardTrafficAccount {
  account_id: number
  account_name: string
  provider_slug: string
  provider_name: string
  group_key: string
  group_name: string
}

export interface SupplierDashboardTrafficResponse {
  range: SupplierDashboardRange
  series: SupplierDashboardTrafficPoint[]
  accounts: SupplierDashboardTrafficAccount[]
  warnings: SupplierDashboardWarning[]
  generated_at: string
}

export interface SupplierDashboardProfitItem {
  account_id: number
  account_name: string
  provider_slug: string
  provider_name: string
  group_key: string
  group_name: string
  requests: number
  tokens: number
  actual_cost: number
  user_cost: number
  profit: number
}

export interface SupplierDashboardProfitResponse {
  items: SupplierDashboardProfitItem[]
  warnings: SupplierDashboardWarning[]
  generated_at: string
}

export interface SupplierDashboardHealthStatusCounts {
  healthy: number
  slow: number
  failed: number
  unavailable: number
  skipped: number
}

export interface SupplierDashboardHealthHour {
  time: string
  status_counts: SupplierDashboardHealthStatusCounts
  total: number
}

export interface SupplierDashboardHealthCell {
  time: string
  status: string
}

export interface SupplierDashboardHealthAccount {
  account_id: number
  account_name: string
  provider_slug: string
  provider_name: string
  group_key: string
  group_name: string
  cells: SupplierDashboardHealthCell[]
}

export interface SupplierDashboardAccountHealthResponse {
  range: SupplierDashboardRange
  accounts: SupplierDashboardHealthAccount[]
  hours: SupplierDashboardHealthHour[]
  warnings: SupplierDashboardWarning[]
  generated_at: string
}

export interface SupplierDashboardTrafficQuery {
  range?: SupplierDashboardRange
  provider_slug?: string
  group_key?: string
}

export interface SupplierDashboardProfitQuery {
  range?: SupplierDashboardRange
  provider_slug?: string
  group_key?: string
  limit?: number
}

export interface SupplierDashboardAccountHealthQuery {
  range?: SupplierDashboardRange
  provider_slug?: string
  group_key?: string
  limit?: number
  buckets?: number
}

export interface SupplierDashboardRequestOptions {
  signal?: AbortSignal
}

function compactParams<T extends Record<string, unknown>>(params: T): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') {
      continue
    }
    result[key] = value
  }
  return result
}

/** 概览继续请求旧 dashboard 接口。 */
export async function getOverview(
  range: SupplierDashboardRange = '24h',
  options?: SupplierDashboardRequestOptions
): Promise<SupplierDashboardOverviewResponse> {
  const { data } = await apiClient.get<SupplierDashboardOverviewResponse>(
    '/admin/upstream-management/dashboard',
    {
      params: { range },
      signal: options?.signal,
    }
  )
  return data
}

/** 异常账号明细。 */
export async function getAccounts(
  query: SupplierDashboardAccountsQuery = {},
  options?: SupplierDashboardRequestOptions
): Promise<SupplierDashboardAccountsResponse> {
  const { data } = await apiClient.get<SupplierDashboardAccountsResponse>(
    '/admin/upstream-management/dashboard/accounts',
    {
      params: compactParams({
        range: query.range ?? '24h',
        risk_type: query.risk_type ?? 'all',
        provider_slug: query.provider_slug,
        group_key: query.group_key,
        page: query.page ?? 1,
        page_size: query.page_size ?? 20,
      }),
      signal: options?.signal,
    }
  )
  return data
}

/** Provider + Group 倍率分析。 */
export async function getRates(
  query: SupplierDashboardRatesQuery = {},
  options?: SupplierDashboardRequestOptions
): Promise<SupplierDashboardRatesResponse> {
  const { data } = await apiClient.get<SupplierDashboardRatesResponse>(
    '/admin/upstream-management/dashboard/rates',
    {
      params: compactParams({
        range: query.range ?? '24h',
        view: query.view ?? 'risk',
        comparison_status: query.comparison_status,
        provider_slug: query.provider_slug,
        group_key: query.group_key,
        page: query.page ?? 1,
        page_size: query.page_size ?? 20,
      }),
      signal: options?.signal,
    }
  )
  return data
}

/** 供应商运行概览。 */
export async function getProviders(
  query: SupplierDashboardProvidersQuery = {},
  options?: SupplierDashboardRequestOptions
): Promise<SupplierDashboardProvidersResponse> {
  const { data } = await apiClient.get<SupplierDashboardProvidersResponse>(
    '/admin/upstream-management/dashboard/providers',
    {
      params: compactParams({
        range: query.range ?? '24h',
        status: query.status,
        page: query.page ?? 1,
        page_size: query.page_size ?? 20,
      }),
      signal: options?.signal,
    }
  )
  return data
}

/** 账号小时级请求量 / Token 流量趋势。 */
export async function getAccountTraffic(
  query: SupplierDashboardTrafficQuery = {},
  options?: SupplierDashboardRequestOptions
): Promise<SupplierDashboardTrafficResponse> {
  const { data } = await apiClient.get<SupplierDashboardTrafficResponse>(
    '/admin/upstream-management/dashboard/traffic',
    {
      params: compactParams({
        range: query.range ?? '30d',
        provider_slug: query.provider_slug,
        group_key: query.group_key,
      }),
      signal: options?.signal,
    }
  )
  return data
}

/** 账号盈利排行。 */
export async function getAccountProfitRanking(
  query: SupplierDashboardProfitQuery = {},
  options?: SupplierDashboardRequestOptions
): Promise<SupplierDashboardProfitResponse> {
  const { data } = await apiClient.get<SupplierDashboardProfitResponse>(
    '/admin/upstream-management/dashboard/profit-ranking',
    {
      params: compactParams({
        range: query.range ?? '30d',
        provider_slug: query.provider_slug,
        group_key: query.group_key,
        limit: query.limit ?? 20,
      }),
      signal: options?.signal,
    }
  )
  return data
}

/** 账号健康状态小时级时间线。 */
export async function getAccountHealthTimeline(
  query: SupplierDashboardAccountHealthQuery = {},
  options?: SupplierDashboardRequestOptions
): Promise<SupplierDashboardAccountHealthResponse> {
  const { data } = await apiClient.get<SupplierDashboardAccountHealthResponse>(
    '/admin/upstream-management/dashboard/health-timeline',
    {
      params: compactParams({
        range: query.range ?? '30d',
        provider_slug: query.provider_slug,
        group_key: query.group_key,
        limit: query.limit ?? 30,
        buckets: query.buckets ?? 72,
      }),
      signal: options?.signal,
    }
  )
  return data
}

export const supplierDashboardAPI = {
  getOverview,
  getAccounts,
  getRates,
  getProviders,
  getAccountTraffic,
  getAccountProfitRanking,
  getAccountHealthTimeline,
}

export default supplierDashboardAPI
