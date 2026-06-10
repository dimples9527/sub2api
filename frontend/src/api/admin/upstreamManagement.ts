import { apiClient } from '../client'
import type { UpstreamProviderConfig } from './upstreamProviders'

export interface UpstreamGroupComparison {
  provider_slug: string
  provider_name: string
  upstream_group_name: string
  upstream_rate: number
  upstream_key_count: number
  local_group_id?: number
  local_group_name?: string
  local_rate?: number
  matched: boolean
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
}

export interface UpstreamGroupCompareResult {
  default_provider: UpstreamProviderConfig
  items: UpstreamGroupComparison[]
  warnings?: string[]
  records: UpstreamGroupRateFixRecord[]
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

export const upstreamManagementAPI = {
  getGroups,
  applyRateFixes
}

export default upstreamManagementAPI
