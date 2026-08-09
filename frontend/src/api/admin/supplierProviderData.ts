import { apiClient, buildGatewayUrl } from '../client'
import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'
import { buildApiUrl } from '../url'
import type { BatchAccountTestJob, GroupPlatform, SubscriptionType } from '@/types'

export type SupplierSyncScope = 'accounts' | 'groups' | 'balance' | 'cost' | 'monitor' | 'all'
export type SupplierSyncStatus = 'success' | 'partial' | 'failed'

export type SupplierSyncProgressStage =
  | 'prepare'
  | 'captcha'
  | 'session'
  | 'login'
  | 'accounts'
  | 'groups'
  | 'balance'
  | 'cost'
  | 'persist'
  | 'done'
  | 'error'

export interface SupplierSyncProgressEvent {
  stage: SupplierSyncProgressStage
  message: string
  ok?: boolean
  time: string
}

export interface SupplierSyncProgressStreamOptions {
  onEvent: (event: SupplierSyncProgressEvent) => void
  signal?: AbortSignal
}

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
  group_status?: 'active' | 'inactive' | 'missing' | ''
  group_record_id?: number
  group_record_delete_eligible: boolean
  account_record_delete_eligible: boolean
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
  local_account_type?: string
  platform_override?: string
  effective_platform?: string
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
  platform_override?: string
  effective_platform?: string
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
  key_sync_status: 'never' | 'running' | 'success' | 'partial' | 'failed' | 'skipped'
  key_status: 'created' | 'not_created' | 'unknown'
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
  /** 上游密钥业务状态：active/disabled/expired/quota_exhausted/unknown */
  status?: string
  /** 分组密钥状态：created/not_created/unknown */
  key_status?: 'created' | 'not_created' | 'unknown' | string
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
  model_ids_by_account?: Record<number, string>
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
  active_group_count: number
  removed_group_count: number
  created_key_group_count: number
  attention_group_count: number
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

/** 使用 SSE 读取供应商同步过程，便于在页面展示打码、登录和数据写入等阶段。 */
export async function streamSupplierProviderSync(
  id: number,
  scope: SupplierSyncScope,
  options: SupplierSyncProgressStreamOptions,
): Promise<void> {
  const path = `/admin/supplier-management/providers/${id}/sync/${scope}/stream`
  const token = localStorage.getItem('auth_token')
  const headers: HeadersInit = {
    Accept: 'text/event-stream',
    [ADMIN_UI_REQUEST_HEADER]: '1',
  }
  if (token) headers.Authorization = `Bearer ${token}`

  const requestInit: RequestInit = {
    method: 'POST',
    headers,
  }
  if (options.signal) requestInit.signal = options.signal

  const response = await fetch(buildApiUrl(path), requestInit)
  if (!response.ok) {
    const body = await response.text().catch(() => '')
    const detail = readableSupplierSyncError(body) || response.statusText || '未知错误'
    throw new Error(`同步进度请求失败（${response.status}）：${detail}`)
  }
  if (!response.body) {
    throw new Error('浏览器不支持读取同步进度流')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''

  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) {
        buffer += decoder.decode()
        if (buffer.trim()) emitSupplierSyncProgressBlock(buffer, options.onEvent)
        return
      }

      buffer += decoder.decode(value, { stream: true })
      buffer = buffer.replace(/\r\n/g, '\n')
      let separatorIndex = buffer.indexOf('\n\n')
      while (separatorIndex >= 0) {
        const block = buffer.slice(0, separatorIndex)
        buffer = buffer.slice(separatorIndex + 2)
        emitSupplierSyncProgressBlock(block, options.onEvent)
        separatorIndex = buffer.indexOf('\n\n')
      }
    }
  } finally {
    reader.releaseLock()
  }
}

function emitSupplierSyncProgressBlock(
  block: string,
  onEvent: (event: SupplierSyncProgressEvent) => void,
): void {
  const dataLines = block
    .split('\n')
    .map(line => line.endsWith('\r') ? line.slice(0, -1) : line)
    .filter(line => line.startsWith('data:'))
    .map(line => line.slice(5).trimStart())
  if (dataLines.length === 0) return

  let parsed: unknown
  try {
    parsed = JSON.parse(dataLines.join('\n'))
  } catch {
    throw new Error('同步进度事件格式无效')
  }
  if (!isSupplierSyncProgressEvent(parsed)) {
    throw new Error('同步进度事件内容无效')
  }
  onEvent(parsed)
}

function isSupplierSyncProgressEvent(value: unknown): value is SupplierSyncProgressEvent {
  if (!value || typeof value !== 'object') return false
  const event = value as Record<string, unknown>
  const stages: SupplierSyncProgressStage[] = [
    'prepare', 'captcha', 'session', 'login', 'accounts', 'groups',
    'balance', 'cost', 'persist', 'done', 'error',
  ]
  if (!stages.includes(event.stage as SupplierSyncProgressStage)) return false
  if (typeof event.message !== 'string' || typeof event.time !== 'string') return false
  return event.ok === undefined || typeof event.ok === 'boolean'
}

function readableSupplierSyncError(body: string): string {
  const raw = body.trim()
  if (!raw) return ''
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const data = parsed.data && typeof parsed.data === 'object'
      ? parsed.data as Record<string, unknown>
      : undefined
    const message = parsed.message || parsed.error || data?.message
    if (typeof message === 'string' && message.trim()) return message.trim()
  } catch {
    // 非 JSON 响应直接使用截断后的文本，避免把整段代理页面塞进提示。
  }
  return raw.slice(0, 500)
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

export async function deleteSupplierAccount(id: number): Promise<{ account_id: number }> {
  const { data } = await apiClient.delete<{ account_id: number }>(
    `/admin/supplier-management/accounts/${id}`
  )
  return data
}

export interface SupplierHealthGuardModel {
  id: string
  display_name: string
}

export async function setSupplierLocalAccountPlatformOverride(
  localAccountID: number,
  platform: string
): Promise<{ local_account_id: number; platform_override: string }> {
  const { data } = await apiClient.put<{ local_account_id: number; platform_override: string }>(
    `/admin/supplier-management/accounts/${localAccountID}/platform-override`,
    { platform }
  )
  return data
}

export async function clearSupplierLocalAccountPlatformOverride(
  localAccountID: number
): Promise<{ local_account_id: number; platform_override: string }> {
  const { data } = await apiClient.delete<{ local_account_id: number; platform_override: string }>(
    `/admin/supplier-management/accounts/${localAccountID}/platform-override`
  )
  return data
}

export async function getSupplierHealthGuardModels(
  localAccountID: number
): Promise<SupplierHealthGuardModel[]> {
  const { data } = await apiClient.get<SupplierHealthGuardModel[]>(
    `/admin/supplier-management/accounts/${localAccountID}/health-guard-models`
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
export async function getLocalMonitorStatus(params?: {
  period?: string
  board?: string
}): Promise<unknown> {
  const { data } = await apiClient.get<unknown>(buildGatewayUrl('/api/llm-monitor/local-status'), {
    params: {
      period: params?.period || '90m',
      board: params?.board || 'hot'
    }
  })
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

export async function setSupplierLocalGroupPlatformOverride(
  localGroupID: number,
  platform: string
): Promise<{ local_group_id: number; platform_override: string }> {
  const { data } = await apiClient.put<{ local_group_id: number; platform_override: string }>(
    `/admin/supplier-management/local-groups/${localGroupID}/platform-override`,
    { platform }
  )
  return data
}

export async function clearSupplierLocalGroupPlatformOverride(
  localGroupID: number
): Promise<{ local_group_id: number; platform_override: string }> {
  const { data } = await apiClient.delete<{ local_group_id: number; platform_override: string }>(
    `/admin/supplier-management/local-groups/${localGroupID}/platform-override`
  )
  return data
}

export async function deleteSupplierGroup(id: number): Promise<{ group_id: number }> {
  const { data } = await apiClient.delete<{ group_id: number }>(
    `/admin/supplier-management/groups/${id}`
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
  deleteSupplierAccount,
  setSupplierLocalAccountPlatformOverride,
  clearSupplierLocalAccountPlatformOverride,
  getSupplierHealthGuardModels,
  startSupplierAccountBatchTest,
  getSupplierAccountBatchTestJob,
  cancelSupplierAccountBatchTestJob,
  listSupplierGroups,
  updateSupplierGroupMapping,
  setSupplierLocalGroupPlatformOverride,
  clearSupplierLocalGroupPlatformOverride,
  deleteSupplierGroup,
  autoMatchSupplierGroups,
  updateSupplierGroupAutoMatchPolicy,
	updateSupplierGroupRateGuard,
	updateSupplierGroupRateGuardIgnore,
  resolveSupplierGroupNameChange,
}

export default supplierProviderDataAPI
