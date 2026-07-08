import { apiClient } from '../client'
import type { UpstreamProviderConfig } from './upstreamProviders'

export interface UpstreamGroupComparison {
  provider_slug: string
  provider_name: string
  upstream_group_name: string
  upstream_group_key: string
  upstream_rate: number
  upstream_key_count: number
  local_group_id?: number
  local_group_name?: string
  local_group_platform?: string
  local_rate?: number
  matched: boolean
  match_source?: 'manual' | 'name' | string
  match_ignored?: boolean
  needs_rate_increase: boolean
}

export interface UpstreamGroupRateFixRecord {
  group_id: number
  group_name: string
  provider_slug: string
  provider_name: string
  upstream_group_name: string
  old_rate: number
  new_rate: number
  changed_at: string
  handled?: boolean
}

export interface UpstreamGroupCompareResult {
  default_provider: UpstreamProviderConfig
  items: UpstreamGroupComparison[]
  warnings?: string[]
  records: UpstreamGroupRateFixRecord[]
}

export interface UpstreamGroupAutoRateFixConfig {
  enabled: boolean
  interval_seconds: number
  last_run_at?: string
  last_run_status?: string
  last_run_message?: string
  updated_at?: string
}

export interface UpstreamGroupLocalCreateRequest {
  upstream_group_name: string
  platform: string
  rate_multiplier: number
}

export async function getGroups(): Promise<UpstreamGroupCompareResult> {
  const { data } = await apiClient.get<UpstreamGroupCompareResult>(
    '/admin/upstream-management/groups'
  )
  return data
}

export async function applyRateFixes(): Promise<UpstreamGroupCompareResult> {
  const { data } = await apiClient.post<UpstreamGroupCompareResult>(
    '/admin/upstream-management/groups/rate-fixes'
  )
  return data
}

export async function markRateFixRecordHandled(key: string): Promise<UpstreamGroupRateFixRecord[]> {
  const { data } = await apiClient.post<UpstreamGroupRateFixRecord[]>(
    `/admin/upstream-management/groups/rate-fix-records/${encodeURIComponent(key)}/handled`
  )
  return data
}

export async function getRateFixConfig(): Promise<UpstreamGroupAutoRateFixConfig> {
  const { data } = await apiClient.get<UpstreamGroupAutoRateFixConfig>(
    '/admin/upstream-management/groups/rate-fix-config'
  )
  return data
}

export async function updateRateFixConfig(
  payload: Pick<UpstreamGroupAutoRateFixConfig, 'enabled' | 'interval_seconds'>
): Promise<UpstreamGroupAutoRateFixConfig> {
  const { data } = await apiClient.put<UpstreamGroupAutoRateFixConfig>(
    '/admin/upstream-management/groups/rate-fix-config',
    payload
  )
  return data
}

export async function saveGroupMapping(
  upstreamGroupName: string,
  localGroupId: number | null,
  ignored = false
): Promise<UpstreamGroupCompareResult> {
  const payload: {
    upstream_group_name: string
    local_group_id: number | null
    ignored?: boolean
  } = {
    upstream_group_name: upstreamGroupName,
    local_group_id: localGroupId
  }
  if (ignored) {
    payload.ignored = true
  }
  const { data } = await apiClient.put<UpstreamGroupCompareResult>(
    '/admin/upstream-management/groups/mappings',
    payload
  )
  return data
}

export async function createLocalGroupFromUpstream(
  payload: UpstreamGroupLocalCreateRequest
): Promise<UpstreamGroupCompareResult> {
  const { data } = await apiClient.post<UpstreamGroupCompareResult>(
    '/admin/upstream-management/groups/local-groups',
    payload
  )
  return data
}

export const upstreamManagementAPI = {
  getGroups,
  applyRateFixes,
  markRateFixRecordHandled,
  getRateFixConfig,
  updateRateFixConfig,
  saveGroupMapping,
  createLocalGroupFromUpstream
}

export default upstreamManagementAPI
