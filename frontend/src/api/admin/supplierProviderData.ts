import { apiClient } from '../client'
import type { BatchAccountTestJob, GroupPlatform, SubscriptionType } from '@/types'

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

export interface SupplierProviderAccountBindingGroup {
  id: number
  name: string
  platform: GroupPlatform
  rate_multiplier: number
  subscription_type: SubscriptionType
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
  platform?: string
  rate_multiplier: number
  binding_groups: SupplierProviderAccountBindingGroup[]
  raw_status: string
  active: boolean
  last_seen_at: string
  inactive_at?: string
  local_account_match_status: 'unmatched' | 'matched' | 'conflict'
  local_account_match_count: number
  local_account_id?: number
  local_account_name?: string
  local_account_platform?: string
  local_account_priority?: number
  local_account_status?: string
  local_account_schedulable?: boolean
  local_account_last_test_status?: string
  local_account_last_tested_at?: string
  local_account_last_test_error?: string
  supplier_current_balance: number
  supplier_today_cost: number
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
  local_group_id?: number
  local_group_name?: string
  local_group_platform?: string
  local_rate_multiplier?: number
  local_group_status?: string
  auto_match_ignored: boolean
  auto_match_status: 'unmatched' | 'auto_matched' | 'manual' | 'ambiguous'
  matched_upstream_name?: string
  name_change_pending: boolean
	 rate_guard_selected: boolean
	 rate_guard_ignored: boolean
	 rate_guard_selection_mode: '' | 'auto' | 'manual'
	 rate_guard_last_snapshot_at?: string
	 rate_guard_last_checked_at?: string
	 local_group_active_mapping_count: number
	 local_group_rate_guard_group_id?: number
	 local_group_rate_guard_group_name?: string
	 local_group_rate_guard_provider_name?: string
	 group_sync_status: 'never' | 'running' | 'success' | 'failed'
	 last_group_sync_at?: string
  account_count: number
  last_seen_at: string
  inactive_at?: string
}

export interface SupplierGroupAutoMatchResult {
  provider_id: number
  scanned: number
  auto_matched: number
  ambiguous: number
  ignored: number
  no_candidate: number
  already_mapped: number
}

export interface SupplierProviderDataListParams {
  provider_id?: number
  group_id?: number
  active?: boolean
  search?: string
  platform?: string
  match_status?: string
  rate_status?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  page?: number
  page_size?: number
}

export interface SupplierProviderAccountListResult {
  items: SupplierProviderAccount[]
  total: number
  page: number
  page_size: number
}

export interface SupplierAccountBatchTestRequest {
  account_ids: number[]
  model_ids_by_platform?: Record<string, string>
  concurrency?: number
  timeout_per_account_seconds?: number
  timeout_seconds?: number
}

export interface SupplierProviderGroupSummary {
  group_count: number
  account_count: number
  linked_group_count: number
  unlinked_group_count: number
  rate_risk_count: number
}

export interface SupplierProviderGroupListResult {
  items: SupplierProviderGroup[]
  total: number
  page: number
  page_size: number
  summary: SupplierProviderGroupSummary
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

export async function startSupplierAccountBatchTest(
  payload: SupplierAccountBatchTestRequest
): Promise<BatchAccountTestJob> {
  const { data } = await apiClient.post<BatchAccountTestJob>(
    '/admin/supplier-management/accounts/batch-test',
    payload,
    { timeout: 30 * 1000 }
  )
  return data
}

export async function getSupplierAccountBatchTestJob(jobID: string): Promise<BatchAccountTestJob> {
  const { data } = await apiClient.get<BatchAccountTestJob>(
    `/admin/supplier-management/accounts/batch-test/${jobID}`
  )
  return data
}

export async function cancelSupplierAccountBatchTestJob(jobID: string): Promise<BatchAccountTestJob> {
  const { data } = await apiClient.post<BatchAccountTestJob>(
    `/admin/supplier-management/accounts/batch-test/${jobID}/cancel`
  )
  return data
}

export interface SupplierProviderGroupHealthTrendPoint {
  time: string
  availability: number
  latency: number
  tested_account_count: number
  tone: 'green' | 'yellow' | 'red' | 'gray'
}

export interface SupplierProviderGroupHealthTrend {
  group_id: number
  source: 'supplier_account_health_guard' | string
  availability: number
  latency: number
  time: string
  trend: SupplierProviderGroupHealthTrendPoint[]
}

export async function listSupplierGroupHealthTrends(
  groupIds: number[],
  period = '90m'
): Promise<SupplierProviderGroupHealthTrend[]> {
  const { data } = await apiClient.get<SupplierProviderGroupHealthTrend[]>(
    '/admin/supplier-management/groups/health-trends',
    { params: { group_ids: groupIds.join(','), period } }
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

export async function updateSupplierGroupMapping(
  id: number,
  localGroupId: number | null
): Promise<{ group_id: number; local_group_id: number | null }> {
  const { data } = await apiClient.put<{ group_id: number; local_group_id: number | null }>(
    `/admin/supplier-management/groups/${id}/mapping`,
    { local_group_id: localGroupId }
  )
  return data
}

export async function autoMatchSupplierGroups(providerId?: number): Promise<SupplierGroupAutoMatchResult> {
  const { data } = await apiClient.post<SupplierGroupAutoMatchResult>(
    '/admin/supplier-management/groups/auto-match',
    undefined,
    { params: providerId ? { provider_id: providerId } : undefined }
  )
  return data
}

export async function updateSupplierGroupAutoMatchPolicy(
  id: number,
  ignored: boolean
): Promise<SupplierGroupAutoMatchResult> {
  const { data } = await apiClient.put<SupplierGroupAutoMatchResult>(
    `/admin/supplier-management/groups/${id}/auto-match-policy`,
    { ignored }
  )
  return data
}

export async function updateSupplierGroupRateGuard(
	id: number,
	selected: boolean
): Promise<{ group_id: number; selected: boolean }> {
	const { data } = await apiClient.put<{ group_id: number; selected: boolean }>(
		`/admin/supplier-management/groups/${id}/rate-guard`,
		{ selected }
	)
	return data
}

export async function updateSupplierGroupRateGuardIgnore(
	id: number,
	ignored: boolean
): Promise<{ group_id: number; ignored: boolean }> {
	const { data } = await apiClient.put<{ group_id: number; ignored: boolean }>(
		`/admin/supplier-management/groups/${id}/rate-guard-ignore`,
		{ ignored }
	)
	return data
}

export async function resolveSupplierGroupNameChange(
  id: number,
  action: 'keep_local' | 'sync_local_name'
): Promise<{ group_id: number; action: string }> {
  const { data } = await apiClient.post<{ group_id: number; action: string }>(
    `/admin/supplier-management/groups/${id}/name-change/resolve`,
    { action }
  )
  return data
}

export const supplierProviderDataAPI = {
  syncProvider,
  testProviderEndpoint,
  listSupplierAccounts,
  startSupplierAccountBatchTest,
  getSupplierAccountBatchTestJob,
  cancelSupplierAccountBatchTestJob,
  listSupplierGroups,
  updateSupplierGroupMapping,
  autoMatchSupplierGroups,
  updateSupplierGroupAutoMatchPolicy,
	updateSupplierGroupRateGuard,
	updateSupplierGroupRateGuardIgnore,
  resolveSupplierGroupNameChange,
}

export default supplierProviderDataAPI
