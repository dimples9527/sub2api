import { apiClient } from '../client'

export type UpstreamProviderType = 'sub2api' | 'newapi' | string

export interface UpstreamProviderConfig {
  type: UpstreamProviderType
  slug: string
  name: string
  sort_order?: number
  enabled: boolean
  is_default?: boolean
  base_url: string
  login_url?: string
  api_keys_url: string
  groups_url?: string
  available_groups_url?: string
  balance_url?: string
  usage_cost_url?: string
  email?: string
  username?: string
  password?: string
  password_configured?: boolean
  account_name_prefix?: string
  temp_disable_minutes?: number
  account_rate_multiplier_scale: number
}

export interface UpstreamProviderKey {
  provider_slug: string
  provider_name: string
  provider_type: string
  key_name: string
  group_name: string
  rate_multiplier: number
  raw_status?: string
  raw_group_id?: string
}

export interface UpstreamProviderTestStage {
  ok: boolean
  status_code?: number
  user_id?: number
  cookie_present?: boolean
  item_count?: number
  group_count?: number
  error?: string
}

export interface UpstreamProviderTestResult {
  type: string
  slug: string
  name: string
  base_url: string
  login_url: string
  keys_url: string
  groups_url?: string
  available_groups_url?: string
  account_name_prefix: string
  login: UpstreamProviderTestStage
  keys: UpstreamProviderTestStage
  groups?: UpstreamProviderTestStage
  parsed_keys: UpstreamProviderKey[]
  warnings?: string[]
}

export interface UpstreamProviderKeysResult {
  items: UpstreamProviderKey[]
  warnings: string[]
}

export interface UpstreamProviderBalance {
  provider_slug: string
  provider_name: string
  provider_type: string
  balance: number
  today_cost?: number
}

export async function list(): Promise<UpstreamProviderConfig[]> {
  const { data } = await apiClient.get<UpstreamProviderConfig[]>(
    '/admin/upstream-management/providers'
  )
  return data
}

export async function create(payload: UpstreamProviderConfig): Promise<UpstreamProviderConfig> {
  const { data } = await apiClient.post<UpstreamProviderConfig>(
    '/admin/upstream-management/providers',
    payload
  )
  return data
}

export async function update(
  slug: string,
  payload: UpstreamProviderConfig
): Promise<UpstreamProviderConfig> {
  const { data } = await apiClient.put<UpstreamProviderConfig>(
    `/admin/upstream-management/providers/${encodeURIComponent(slug)}`,
    payload
  )
  return data
}

export async function deleteProvider(slug: string): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/upstream-management/providers/${encodeURIComponent(slug)}`
  )
  return data
}

export async function setDefault(slug: string): Promise<UpstreamProviderConfig> {
  const { data } = await apiClient.post<UpstreamProviderConfig>(
    `/admin/upstream-management/providers/${encodeURIComponent(slug)}/default`
  )
  return data
}

export async function testSaved(slug: string): Promise<UpstreamProviderTestResult> {
  const { data } = await apiClient.post<UpstreamProviderTestResult>(
    `/admin/upstream-management/providers/${encodeURIComponent(slug)}/test`
  )
  return data
}

export async function testConfig(
  payload: UpstreamProviderConfig
): Promise<UpstreamProviderTestResult> {
  const { data } = await apiClient.post<UpstreamProviderTestResult>(
    '/admin/upstream-management/providers/test',
    payload
  )
  return data
}

export async function getKeys(slug: string): Promise<UpstreamProviderKeysResult> {
  const { data } = await apiClient.get<UpstreamProviderKeysResult>(
    `/admin/upstream-management/providers/${encodeURIComponent(slug)}/keys`
  )
  return data
}

export async function getBalance(slug: string): Promise<UpstreamProviderBalance> {
  const { data } = await apiClient.get<UpstreamProviderBalance>(
    `/admin/upstream-management/providers/${encodeURIComponent(slug)}/balance`
  )
  return data
}

export const upstreamProvidersAPI = {
  list,
  create,
  update,
  delete: deleteProvider,
  setDefault,
  testSaved,
  testConfig,
  getKeys,
  getBalance
}

export default upstreamProvidersAPI
