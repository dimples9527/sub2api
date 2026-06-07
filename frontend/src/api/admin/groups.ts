/**
 * Admin Groups API endpoints
 * Handles API key group management for administrators
 */

import { apiClient } from '../client'
import type {
  AdminGroup,
  GroupPlatform,
  CreateGroupRequest,
  UpdateGroupRequest,
  PaginatedResponse
} from '@/types'

/**
 * List all groups with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters (platform, status, is_exclusive, search)
 * @returns Paginated list of groups
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    platform?: GroupPlatform
    status?: 'active' | 'inactive'
    is_exclusive?: boolean
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<AdminGroup>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminGroup>>('/admin/groups', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

/**
 * Get all active groups (without pagination)
 * @param platform - Optional platform filter
 * @returns List of all active groups
 */
export async function getAll(platform?: GroupPlatform): Promise<AdminGroup[]> {
  const { data } = await apiClient.get<AdminGroup[]>('/admin/groups/all', {
    params: platform ? { platform } : undefined
  })
  return data
}

/**
 * Get active groups by platform
 * @param platform - Platform to filter by
 * @returns List of groups for the specified platform
 */
export async function getByPlatform(platform: GroupPlatform): Promise<AdminGroup[]> {
  return getAll(platform)
}

/**
 * Get group by ID
 * @param id - Group ID
 * @returns Group details
 */
export async function getById(id: number): Promise<AdminGroup> {
  const { data } = await apiClient.get<AdminGroup>(`/admin/groups/${id}`)
  return data
}

/**
 * Get candidate models for custom /v1/models list.
 * id=0 returns platform default models for create flow.
 */
export async function getModelsListCandidates(
  id: number,
  platform?: GroupPlatform
): Promise<string[]> {
  const { data } = await apiClient.get<{ models: string[] }>(
    `/admin/groups/${id}/models-list-candidates`,
    {
      params: platform ? { platform } : undefined
    }
  )
  return data.models || []
}

/**
 * Create new group
 * @param groupData - Group data
 * @returns Created group
 */
export async function create(groupData: CreateGroupRequest): Promise<AdminGroup> {
  const { data } = await apiClient.post<AdminGroup>('/admin/groups', groupData)
  return data
}

/**
 * Update group
 * @param id - Group ID
 * @param updates - Fields to update
 * @returns Updated group
 */
export async function update(id: number, updates: UpdateGroupRequest): Promise<AdminGroup> {
  const { data } = await apiClient.put<AdminGroup>(`/admin/groups/${id}`, updates)
  return data
}

/**
 * Delete group
 * @param id - Group ID
 * @returns Success confirmation
 */
export async function deleteGroup(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/groups/${id}`)
  return data
}

/**
 * Toggle group status
 * @param id - Group ID
 * @param status - New status
 * @returns Updated group
 */
export async function toggleStatus(id: number, status: 'active' | 'inactive'): Promise<AdminGroup> {
  return update(id, { status })
}

/**
 * Get group statistics
 * @param id - Group ID
 * @returns Group usage statistics
 */
export async function getStats(id: number): Promise<{
  total_api_keys: number
  active_api_keys: number
  total_requests: number
  total_cost: number
}> {
  const { data } = await apiClient.get<{
    total_api_keys: number
    active_api_keys: number
    total_requests: number
    total_cost: number
  }>(`/admin/groups/${id}/stats`)
  return data
}

/**
 * Get API keys in a group
 * @param id - Group ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of API keys in the group
 */
export async function getGroupApiKeys(
  id: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<any>> {
  const { data } = await apiClient.get<PaginatedResponse<any>>(`/admin/groups/${id}/api-keys`, {
    params: { page, page_size: pageSize }
  })
  return data
}

/**
 * Rate multiplier entry for a user in a group
 */
export interface GroupRateMultiplierEntry {
  user_id: number
  user_name: string
  user_email: string
  user_notes: string
  user_status: string
  rate_multiplier?: number | null
  rpm_override?: number | null
}

export interface UpstreamRateSyncResult {
  checked_count: number
  matched_count: number
  updated_count: number
  rate_warnings?: Array<{
    group_id: number
    group_name: string
    local_rate_multiplier: number
    upstream_rate_multiplier: number
  }>
}

export interface UpstreamAvailableGroup {
  id: number | string
  name: string
  description?: string | null
  platform?: string
  rate_multiplier?: number | null
  status?: string
  subscription_type?: string
  daily_limit_usd?: number | null
  weekly_limit_usd?: number | null
  monthly_limit_usd?: number | null
  image_price_1k?: number | null
  image_price_2k?: number | null
  image_price_4k?: number | null
  claude_code_only?: boolean
  allow_messages_dispatch?: boolean
  per_request_price?: number | null
  created_at?: string
  updated_at?: string
  local_group_id?: number | null
  local_group_name?: string
  local_rate_multiplier?: number | null
}

export interface UpstreamKeySummary {
  groups: Array<{
    name: string
    normalized_name: string
    key_count: number
    keys?: Array<{
      name: string
    }>
  }>
}

export interface AccountRateGuardProviderResult {
  slug: string
  name: string
  account_name_prefix: string
  checked_key_count: number
  error?: string
}

export interface AccountRateGuardViolation {
  provider_slug: string
  provider_name: string
  upstream_key_name: string
  matched_local_account_id: number
  matched_local_account_name: string
  upstream_group_name: string
  upstream_rate_multiplier: number
  local_min_rate_multiplier: number
  unbound_group_ids?: number[]
  unbound_group_names?: string[]
}

export interface AccountRateGuardResult {
  dry_run: boolean
  checked_key_count: number
  matched_account_count: number
  violation_count: number
  unbound_count: number
  violations: AccountRateGuardViolation[]
  providers: AccountRateGuardProviderResult[]
}

export interface AccountRateGuardAuditEntry {
  run_id: number
  created_at: string
  provider_slug: string
  provider_name: string
  upstream_key_name: string
  matched_local_account_id: number
  matched_local_account_name: string
  upstream_group_name: string
  upstream_rate_multiplier: number
  local_min_rate_multiplier: number
  unbound_group_ids: number[]
  unbound_group_names: string[]
  remaining_group_ids: number[]
}

export interface AccountRateGuardRunRecord {
  run_id: number
  started_at: string
  completed_at: string
  result: AccountRateGuardResult
  error?: string
}

export interface AccountRateGuardStatus {
  last_run?: AccountRateGuardRunRecord
  audits: AccountRateGuardAuditEntry[]
}

export async function syncUpstreamRates(): Promise<UpstreamRateSyncResult> {
  const { data } = await apiClient.post<UpstreamRateSyncResult>('/admin/upstream-management/sync')
  return data
}

export async function getUpstreamRateWarnings(): Promise<UpstreamRateSyncResult> {
  const { data } = await apiClient.get<UpstreamRateSyncResult>(
    '/admin/upstream-management/rate-warnings'
  )
  return data
}

export async function getUpstreamAvailableGroups(): Promise<UpstreamAvailableGroup[]> {
  const { data } = await apiClient.get<UpstreamAvailableGroup[]>('/admin/upstream-management/groups')
  return Array.isArray(data) ? data : []
}

export async function getUpstreamKeySummary(): Promise<UpstreamKeySummary> {
  const { data } = await apiClient.get<UpstreamKeySummary>('/admin/upstream-management/key-summary')
  return data && Array.isArray(data.groups) ? data : { groups: [] }
}

export async function runAccountRateGuard(dryRun = false): Promise<AccountRateGuardResult> {
  const { data } = await apiClient.post<AccountRateGuardResult>(
    '/admin/upstream-management/account-rate-guard/run',
    { dry_run: dryRun }
  )
  return data
}

export async function getAccountRateGuardStatus(): Promise<AccountRateGuardStatus> {
  const { data } = await apiClient.get<AccountRateGuardStatus>(
    '/admin/upstream-management/account-rate-guard/status'
  )
  return data || { audits: [] }
}

export async function getAccountRateGuardAudits(): Promise<AccountRateGuardAuditEntry[]> {
  const { data } = await apiClient.get<AccountRateGuardAuditEntry[]>(
    '/admin/upstream-management/account-rate-guard/audits'
  )
  return Array.isArray(data) ? data : []
}

export async function getUpstreamMonitorStatus(params?: {
  period?: string
  board?: string
}): Promise<unknown> {
  const { data } = await apiClient.get<unknown>('/admin/upstream-management/monitor-status', {
    params: {
      period: params?.period || '90m',
      board: params?.board || 'hot'
    }
  })
  return data
}

export type ModelSquareRateSyncResult = UpstreamRateSyncResult
export type ModelSquareAvailableGroup = UpstreamAvailableGroup

export const syncModelSquareRates = syncUpstreamRates
export const getModelSquareRateWarnings = getUpstreamRateWarnings
export const getModelSquareAvailableGroups = getUpstreamAvailableGroups

/**
 * Get rate multipliers for users in a group
 * @param id - Group ID
 * @returns List of user rate multiplier entries
 */
export async function getGroupRateMultipliers(id: number): Promise<GroupRateMultiplierEntry[]> {
  const { data } = await apiClient.get<GroupRateMultiplierEntry[]>(
    `/admin/groups/${id}/rate-multipliers`
  )
  return data
}

/**
 * Update group sort orders
 * @param updates - Array of { id, sort_order } objects
 * @returns Success confirmation
 */
export async function updateSortOrder(
  updates: Array<{ id: number; sort_order: number }>
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>('/admin/groups/sort-order', {
    updates
  })
  return data
}

/**
 * Clear all rate multipliers for a group
 * @param id - Group ID
 * @returns Success confirmation
 */
export async function clearGroupRateMultipliers(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/groups/${id}/rate-multipliers`)
  return data
}

/**
 * Batch set rate multipliers for users in a group
 * Only touches rate_multiplier column; preserves rpm_override on existing rows.
 */
export async function batchSetGroupRateMultipliers(
  id: number,
  entries: Array<{ user_id: number; rate_multiplier: number }>
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(
    `/admin/groups/${id}/rate-multipliers`,
    { entries }
  )
  return data
}

/**
 * RPM override entry for a user in a group
 */
export interface GroupRPMOverrideEntry {
  user_id: number
  user_name: string
  user_email: string
  user_notes: string
  user_status: string
  rpm_override: number
}

/**
 * Get RPM overrides for users in a group (subset of rate-multipliers endpoint).
 */
export async function getGroupRPMOverrides(id: number): Promise<GroupRPMOverrideEntry[]> {
  const { data } = await apiClient.get<GroupRateMultiplierEntry[]>(
    `/admin/groups/${id}/rate-multipliers`
  )
  return data
    .filter(e => e.rpm_override != null)
    .map(e => ({
      user_id: e.user_id,
      user_name: e.user_name,
      user_email: e.user_email,
      user_notes: e.user_notes,
      user_status: e.user_status,
      rpm_override: e.rpm_override as number
    }))
}

/**
 * Batch set RPM overrides for users in a group.
 * Only touches rpm_override column; preserves rate_multiplier on existing rows.
 */
export async function batchSetGroupRPMOverrides(
  id: number,
  entries: Array<{ user_id: number; rpm_override: number }>
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(
    `/admin/groups/${id}/rpm-overrides`,
    { entries }
  )
  return data
}

/**
 * Clear all RPM overrides for a group (preserves rate_multiplier).
 */
export async function clearGroupRPMOverrides(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/groups/${id}/rpm-overrides`)
  return data
}

/**
 * Get usage summary (today + cumulative cost) for all groups
 * @param timezone - IANA timezone string (e.g. "Asia/Shanghai")
 * @returns Array of group usage summaries
 */
export async function getUsageSummary(
  timezone?: string
): Promise<{ group_id: number; today_cost: number; total_cost: number }[]> {
  const { data } = await apiClient.get<
    { group_id: number; today_cost: number; total_cost: number }[]
  >('/admin/groups/usage-summary', {
    params: timezone ? { timezone } : undefined
  })
  return data
}

/**
 * Get capacity summary (concurrency/sessions/RPM) for all active groups
 */
export async function getCapacitySummary(): Promise<
  { group_id: number; concurrency_used: number; concurrency_max: number; sessions_used: number; sessions_max: number; rpm_used: number; rpm_max: number }[]
> {
  const { data } = await apiClient.get<
    { group_id: number; concurrency_used: number; concurrency_max: number; sessions_used: number; sessions_max: number; rpm_used: number; rpm_max: number }[]
  >('/admin/groups/capacity-summary')
  return data
}

export const groupsAPI = {
  list,
  getAll,
  getByPlatform,
  getById,
  getModelsListCandidates,
  create,
  update,
  delete: deleteGroup,
  toggleStatus,
  getStats,
  getGroupApiKeys,
  getGroupRateMultipliers,
  syncUpstreamRates,
  getUpstreamRateWarnings,
  getUpstreamAvailableGroups,
  getUpstreamKeySummary,
  getUpstreamMonitorStatus,
  syncModelSquareRates,
  getModelSquareRateWarnings,
  getModelSquareAvailableGroups,
  clearGroupRateMultipliers,
  batchSetGroupRateMultipliers,
  getGroupRPMOverrides,
  clearGroupRPMOverrides,
  batchSetGroupRPMOverrides,
  updateSortOrder,
  getUsageSummary,
  getCapacitySummary
}

export default groupsAPI
