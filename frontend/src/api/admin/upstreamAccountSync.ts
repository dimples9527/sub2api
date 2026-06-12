import { apiClient } from '../client'
import type { UpstreamProviderConfig } from './upstreamProviders'

export type UpstreamAccountSyncAction = 'create' | 'update' | 'noop' | 'skip' | 'conflict' | string

export interface UpstreamAccountSyncSummary {
  upstream_key_count: number
  matched_account_count: number
  create_count: number
  update_count: number
  skip_count: number
  conflict_count: number
  rate_violation_count: number
  unbound_group_count: number
}

export interface UpstreamAccountSyncItem {
  action: UpstreamAccountSyncAction
  provider_slug: string
  provider_name: string
  upstream_key_name: string
  local_account_name: string
  matched_account_id?: number
  matched_account_name?: string
  upstream_group_name: string
  upstream_rate_multiplier: number
  local_group_id?: number
  local_group_name?: string
  local_rate_multiplier?: number
  rate_violation: boolean
  unbound_group_ids?: number[]
  unbound_group_names?: string[]
  skip_reason?: string
  conflict_account_ids?: number[]
  bound_groups?: UpstreamAccountSyncBoundGroup[]
}

export interface UpstreamAccountSyncBoundGroup {
  id: number
  name: string
  rate_multiplier: number
  rate_violation: boolean
}

export interface UpstreamAccountSyncUnbindDetail {
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

export interface UpstreamAccountSyncRecord {
  provider_slug: string
  provider_name: string
  created_count: number
  updated_count: number
  skipped_count: number
  conflict_count: number
  rate_violation_count: number
  unbound_group_count: number
  created_at: string
  error?: string
  unbind_details?: UpstreamAccountSyncUnbindDetail[]
}

export interface UpstreamAccountSyncResult {
  default_provider: UpstreamProviderConfig
  providers: UpstreamProviderConfig[]
  summary: UpstreamAccountSyncSummary
  items: UpstreamAccountSyncItem[]
  warnings?: string[]
  records: UpstreamAccountSyncRecord[]
}

export interface UpstreamAccountSyncRequest {
  create_missing: boolean
  update_existing: boolean
  apply_rate_guard: boolean
}

export async function getPreview(): Promise<UpstreamAccountSyncResult> {
  const { data } = await apiClient.get<UpstreamAccountSyncResult>(
    '/admin/upstream-management/accounts/sync-preview'
  )
  return data
}

export async function runSync(payload: UpstreamAccountSyncRequest): Promise<UpstreamAccountSyncResult> {
  const { data } = await apiClient.post<UpstreamAccountSyncResult>(
    '/admin/upstream-management/accounts/sync',
    payload
  )
  return data
}

export async function getRecords(): Promise<UpstreamAccountSyncRecord[]> {
  const { data } = await apiClient.get<UpstreamAccountSyncRecord[]>(
    '/admin/upstream-management/accounts/sync-records'
  )
  return data
}

export const upstreamAccountSyncAPI = {
  getPreview,
  runSync,
  getRecords
}

export default upstreamAccountSyncAPI
