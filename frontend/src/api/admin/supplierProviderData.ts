import { apiClient } from '../client'

export type SupplierSyncScope = 'accounts' | 'groups' | 'balance' | 'cost' | 'all'
export type SupplierSyncStatus = 'success' | 'partial' | 'failed'

export interface SupplierSyncCounts {
  checked_count: number
  created_count: number
  updated_count: number
  skipped_count: number
}

export interface SupplierProviderSyncStage {
  scope: SupplierSyncScope
  status: SupplierSyncStatus
  message: string
  counts: SupplierSyncCounts
}

export interface SupplierProviderSyncResult {
  provider_id: number
  scope: SupplierSyncScope
  status: SupplierSyncStatus
  message: string
  counts: SupplierSyncCounts
  stages?: SupplierProviderSyncStage[]
  started_at: string
  finished_at: string
}

export interface SupplierProviderEndpointTestAttempt {
  endpoint: string
  http_status: number
  duration_ms: number
  response_bytes: number
  response_summary: string
  parsed_data?: unknown
  parse_error?: string
  error?: string
}

export interface SupplierProviderEndpointTestResult extends SupplierProviderEndpointTestAttempt {
  provider_id: number
  scope: Exclude<SupplierSyncScope, 'all'>
  attempts: SupplierProviderEndpointTestAttempt[]
  sensitive_redacted: boolean
}

export interface SupplierProviderAccount {
  id: number
  provider_id: number
  provider_name: string
  upstream_account_key: string
  name: string
  status: string
  group_key: string
  group_name: string
  rate_multiplier: number
  raw_status: string
  active: boolean
  last_seen_at: string
  inactive_at?: string
}

export interface SupplierProviderGroup {
  id: number
  provider_id: number
  provider_name: string
  upstream_group_key: string
  name: string
  rate_multiplier: number
  raw_status: string
  active: boolean
  account_count: number
  last_seen_at: string
  inactive_at?: string
}

export interface SupplierProviderDataListParams {
  provider_id?: number
  active?: boolean
  search?: string
  page?: number
  page_size?: number
}

export interface SupplierProviderAccountListResult {
  items: SupplierProviderAccount[]
  total: number
  page: number
  page_size: number
}

export interface SupplierProviderGroupListResult {
  items: SupplierProviderGroup[]
  total: number
  page: number
  page_size: number
}

export async function syncProvider(id: number, scope: SupplierSyncScope): Promise<SupplierProviderSyncResult> {
  const { data } = await apiClient.post<SupplierProviderSyncResult>(
    `/admin/supplier-management/providers/${id}/sync/${scope}`
  )
  return data
}

export async function testProviderEndpoint(id: number, scope: Exclude<SupplierSyncScope, 'all'>): Promise<SupplierProviderEndpointTestResult> {
  const { data } = await apiClient.post<SupplierProviderEndpointTestResult>(
    `/admin/supplier-management/providers/${id}/test/${scope}`
  )
  return data
}

export async function listSupplierAccounts(params: SupplierProviderDataListParams = {}): Promise<SupplierProviderAccountListResult> {
  const { data } = await apiClient.get<SupplierProviderAccountListResult>(
    '/admin/supplier-management/accounts',
    { params }
  )
  return data
}

export async function listSupplierGroups(params: SupplierProviderDataListParams = {}): Promise<SupplierProviderGroupListResult> {
  const { data } = await apiClient.get<SupplierProviderGroupListResult>(
    '/admin/supplier-management/groups',
    { params }
  )
  return data
}

export const supplierProviderDataAPI = {
  syncProvider,
  testProviderEndpoint,
  listSupplierAccounts,
  listSupplierGroups,
}

export default supplierProviderDataAPI
