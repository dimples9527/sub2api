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
  provider_base_url?: string
  upstream_key_name: string
  upstream_api_key?: string
  upstream_base_url?: string
  provider_fetch_error?: string
  local_account_name: string
  matched_account_id?: number
  matched_account_name?: string
  upstream_group_name: string
  upstream_rate_multiplier: number
  local_group_id?: number
  local_group_name?: string
  local_rate_multiplier?: number
  rate_violation: boolean
  rate_guard_ignored?: boolean
  unbound_group_ids?: number[]
  unbound_group_names?: string[]
  skip_reason?: string
  conflict_account_ids?: number[]
  conflict_accounts?: UpstreamAccountSyncConflictAccount[]
  bound_groups?: UpstreamAccountSyncBoundGroup[]
  change_details?: UpstreamAccountSyncChangeDetail[]
  execution?: UpstreamAccountSyncExecutionResult
}

export interface UpstreamAccountSyncChangeDetail {
  kind: 'credential' | 'metadata' | 'group_bind' | 'group_unbind' | string
  field?: string
  label?: string
  before?: string
  after?: string
  group_ids?: number[]
  group_names?: string[]
}

export interface UpstreamAccountSyncExecutionResult {
  executed?: boolean
  action?: 'create' | 'update' | string
  account_id?: number
  account_name?: string
  unbound_group_ids?: number[]
  unbound_group_names?: string[]
}

export interface UpstreamAccountSyncConflictAccount {
  id: number
  name: string
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
  trigger_source?: string
  handled?: boolean
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
  trigger_source?: string
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
  selected_items?: UpstreamAccountSyncSelectedItem[]
}

export interface UpstreamAccountSyncSelectedItem {
  provider_slug: string
  upstream_key_name: string
  create_missing: boolean
  update_existing: boolean
  apply_rate_guard: boolean
}

export interface UpstreamAccountRateGuardConfig {
  enabled: boolean
  interval_seconds: number
  ignored_account_ids?: number[]
  last_run_at?: string
  last_run_status?: string
  last_run_message?: string
  updated_at?: string
}

export interface UpstreamAccountRateGuardPollLog {
  checked_at: string
  trigger: 'scheduled' | 'manual' | string
  status: 'success' | 'failed' | 'skipped' | string
  message?: string
}

export interface UpstreamBalanceSamplerConfig {
  enabled: boolean
  interval_seconds: number
  provider_amount_scales?: Record<string, number>
  last_run_at?: string
  last_run_status?: string
  last_run_message?: string
  updated_at?: string
}

export interface UpstreamBalanceProviderSummary {
  provider_slug: string
  provider_name?: string
  current_balance: number
  today_consumption: number
  amount_scale: number
  complete: boolean
  anomaly: boolean
  snapshot_count: number
  last_snapshot_at?: string
  last_snapshot_error?: string
}

export interface UpstreamBalanceDailyRow {
  provider_slug: string
  provider_name?: string
  date: string
  amount_scale: number
  opening_balance: number
  closing_balance: number
  current_balance: number
  recharge_amount: number
  consumption_amount: number
  snapshot_count: number
  complete: boolean
  anomaly: boolean
  first_snapshot_at?: string
  last_snapshot_at?: string
}

export interface UpstreamBalanceSnapshot {
  id: number
  provider_slug: string
  provider_name?: string
  provider_type?: string
  balance: number
  today_cost?: number
  amount_scale: number
  status: 'success' | 'failed' | string
  error?: string
  captured_at: string
  created_at: string
}

export interface UpstreamLocalDailyConsumption {
  date: string
  actual_cost: number
}

export interface UpstreamBalanceConsumptionOverview {
  config: UpstreamBalanceSamplerConfig
  summaries: Record<string, UpstreamBalanceProviderSummary>
  rows: UpstreamBalanceDailyRow[]
  snapshots: UpstreamBalanceSnapshot[]
  local_daily_consumptions?: UpstreamLocalDailyConsumption[]
}

export interface UpstreamBalanceRechargeInput {
  provider_slug: string
  amount: number
  amount_scale?: number
  note?: string
  occurred_at?: string
}

export interface UpstreamBalanceRecharge {
  id: number
  provider_slug: string
  provider_name?: string
  amount: number
  amount_scale: number
  note?: string
  occurred_at: string
  created_at: string
}

export interface UpstreamBalanceSamplerPollLog {
  checked_at: string
  trigger: 'scheduled' | 'manual' | string
  status: 'success' | 'failed' | 'skipped' | string
  message?: string
}

export interface UpstreamAccountHealthGuardConfig {
  enabled: boolean
  interval_seconds: number
  max_accounts_per_run: number
  concurrency: number
  timeout_per_account_seconds: number
  failure_threshold: number
  slow_threshold: number
  recovery_threshold: number
  healthy_latency_ms: number
  ignored_account_ids?: number[]
  account_models?: Record<string, string>
  platform_models?: Record<string, string>
  platform_latency_ms?: Record<string, number>
  last_run_at?: string
  last_run_status?: string
  last_run_message?: string
  cursor_account_id?: number
  updated_at?: string
}

export interface UpstreamAccountHealthGuardRunSummary {
  total_accounts: number
  checked_count: number
  healthy_count: number
  slow_count: number
  failed_count: number
  skipped_count: number
  disabled_count: number
  recovered_count: number
  unchanged_count: number
  skip_reasons?: UpstreamAccountHealthGuardSkipReason[]
}

export interface UpstreamAccountHealthGuardSkippedAccount {
  account_id: number
  account_name: string
  platform: string
  provider_slug?: string
}

export interface UpstreamAccountHealthGuardSkipReason {
  reason: string
  count: number
  sample_accounts?: UpstreamAccountHealthGuardSkippedAccount[]
}

export interface UpstreamAccountHealthGuardRunItem {
  account_id: number
  account_name: string
  platform: string
  provider_slug: string
  provider_name: string
  model_id?: string
  schedulable_before: boolean
  schedulable_after: boolean
  status: 'healthy' | 'slow' | 'failed' | string
  test_status?: string
  latency_ms: number
  latency_limit_ms: number
  consecutive_failed: number
  consecutive_slow: number
  consecutive_healthy: number
  action: 'none' | 'disabled' | 'recovered' | string
  reason?: string
  error_message?: string
  started_at: string
  finished_at: string
}

export interface UpstreamAccountHealthGuardRunRecord {
  id: string
  trigger: 'scheduled' | 'manual' | string
  status: 'success' | 'failed' | string
  message?: string
  started_at: string
  finished_at: string
  summary: UpstreamAccountHealthGuardRunSummary
  items: UpstreamAccountHealthGuardRunItem[]
}

export interface UpstreamAccountHealthGuardRunResponse {
  config: UpstreamAccountHealthGuardConfig
  record: UpstreamAccountHealthGuardRunRecord
}

export interface UpstreamAccountHealthGuardPollLog {
  checked_at: string
  trigger: 'scheduled' | 'manual' | string
  status: 'success' | 'failed' | 'skipped' | string
  message?: string
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

export async function markRecordHandled(key: string): Promise<UpstreamAccountSyncRecord[]> {
  const { data } = await apiClient.post<UpstreamAccountSyncRecord[]>(
    `/admin/upstream-management/accounts/sync-records/${encodeURIComponent(key)}/handled`
  )
  return data
}

export async function getRateGuardConfig(): Promise<UpstreamAccountRateGuardConfig> {
  const { data } = await apiClient.get<UpstreamAccountRateGuardConfig>(
    '/admin/upstream-management/accounts/rate-guard-config'
  )
  return data
}

export async function updateRateGuardConfig(
  payload: UpstreamAccountRateGuardConfig
): Promise<UpstreamAccountRateGuardConfig> {
  const { data } = await apiClient.put<UpstreamAccountRateGuardConfig>(
    '/admin/upstream-management/accounts/rate-guard-config',
    payload
  )
  return data
}

export async function runRateGuardNow(): Promise<UpstreamAccountRateGuardConfig> {
  const { data } = await apiClient.post<UpstreamAccountRateGuardConfig>(
    '/admin/upstream-management/accounts/rate-guard-runs'
  )
  return data
}

export async function getRateGuardPollLogs(): Promise<UpstreamAccountRateGuardPollLog[]> {
  const { data } = await apiClient.get<UpstreamAccountRateGuardPollLog[]>(
    '/admin/upstream-management/accounts/rate-guard-poll-logs'
  )
  return data
}

export async function getBalanceConsumption(
  days = 30
): Promise<UpstreamBalanceConsumptionOverview> {
  const { data } = await apiClient.get<UpstreamBalanceConsumptionOverview>(
    '/admin/upstream-management/providers/balance-consumption',
    { params: { days } }
  )
  return data
}

export async function getBalanceSamplerConfig(): Promise<UpstreamBalanceSamplerConfig> {
  const { data } = await apiClient.get<UpstreamBalanceSamplerConfig>(
    '/admin/upstream-management/providers/balance-consumption/config'
  )
  return data
}

export async function updateBalanceSamplerConfig(
  payload: UpstreamBalanceSamplerConfig
): Promise<UpstreamBalanceSamplerConfig> {
  const { data } = await apiClient.put<UpstreamBalanceSamplerConfig>(
    '/admin/upstream-management/providers/balance-consumption/config',
    payload
  )
  return data
}

export async function addBalanceRecharge(
  payload: UpstreamBalanceRechargeInput
): Promise<UpstreamBalanceRecharge> {
  const { data } = await apiClient.post<UpstreamBalanceRecharge>(
    '/admin/upstream-management/providers/balance-consumption/recharges',
    payload
  )
  return data
}

export async function runBalanceSampleNow(): Promise<UpstreamBalanceSamplerConfig> {
  const { data } = await apiClient.post<UpstreamBalanceSamplerConfig>(
    '/admin/upstream-management/providers/balance-consumption/samples'
  )
  return data
}

export async function getBalanceSamplerPollLogs(): Promise<UpstreamBalanceSamplerPollLog[]> {
  const { data } = await apiClient.get<UpstreamBalanceSamplerPollLog[]>(
    '/admin/upstream-management/providers/balance-consumption/poll-logs'
  )
  return data
}

export async function getHealthGuardConfig(): Promise<UpstreamAccountHealthGuardConfig> {
  const { data } = await apiClient.get<UpstreamAccountHealthGuardConfig>(
    '/admin/upstream-management/providers/health-guard/config'
  )
  return data
}

export async function updateHealthGuardConfig(
  payload: UpstreamAccountHealthGuardConfig
): Promise<UpstreamAccountHealthGuardConfig> {
  const { data } = await apiClient.put<UpstreamAccountHealthGuardConfig>(
    '/admin/upstream-management/providers/health-guard/config',
    payload
  )
  return data
}

export async function runHealthGuardNow(): Promise<UpstreamAccountHealthGuardRunResponse> {
  const { data } = await apiClient.post<UpstreamAccountHealthGuardRunResponse>(
    '/admin/upstream-management/providers/health-guard/runs'
  )
  return data
}

export async function getHealthGuardRecords(): Promise<UpstreamAccountHealthGuardRunRecord[]> {
  const { data } = await apiClient.get<UpstreamAccountHealthGuardRunRecord[]>(
    '/admin/upstream-management/providers/health-guard/records'
  )
  return data
}

export async function getHealthGuardPollLogs(): Promise<UpstreamAccountHealthGuardPollLog[]> {
  const { data } = await apiClient.get<UpstreamAccountHealthGuardPollLog[]>(
    '/admin/upstream-management/providers/health-guard/poll-logs'
  )
  return data
}

export const upstreamAccountSyncAPI = {
  getPreview,
  runSync,
  getRecords,
  markRecordHandled,
  getRateGuardConfig,
  updateRateGuardConfig,
  runRateGuardNow,
  getRateGuardPollLogs,
  getBalanceConsumption,
  getBalanceSamplerConfig,
  updateBalanceSamplerConfig,
  addBalanceRecharge,
  runBalanceSampleNow,
  getBalanceSamplerPollLogs,
  getHealthGuardConfig,
  updateHealthGuardConfig,
  runHealthGuardNow,
  getHealthGuardRecords,
  getHealthGuardPollLogs
}

export default upstreamAccountSyncAPI
