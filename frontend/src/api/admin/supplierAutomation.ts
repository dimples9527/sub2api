import { apiClient } from '../client'

export interface SupplierAutomationConfig {
  rate_guard_max_snapshot_age_seconds: number
  automation_run_retention_days: number
  sync_run_retention_days: number
  metric_snapshot_retention_days: number
  daily_stat_retention_days: number
  inactive_account_retention_days: number
  inactive_group_retention_days: number
  account_health_guard_max_accounts_per_run: number
  account_health_guard_concurrency: number
  account_health_guard_timeout_per_account_seconds: number
  account_health_guard_failure_threshold: number
  account_health_guard_slow_threshold: number
  account_health_guard_recovery_threshold: number
  account_health_guard_healthy_latency_ms: number
  account_health_guard_account_ids: number[]
  account_health_guard_account_models: Record<string, string>
  account_health_guard_platform_models: Record<string, string>
  account_health_guard_platform_latency_ms: Record<string, number>
  account_health_guard_cursor_account_id: number
}

export interface SupplierAutomationTask {
  id: number
  task_code: string
  name: string
  enabled: boolean
  cron_expression: string
  timeout_seconds: number
  config: SupplierAutomationConfig
  last_status: string
  last_message: string
  last_run_at?: string
  next_run_at?: string
}

export interface SupplierAutomationRun {
  id: number
  task_code: string
  trigger_source: string
  status: string
  message: string
  processed_count: number
  success_count: number
  failed_count: number
  result_detail?: SupplierAutomationRunDetail
  started_at: string
  finished_at?: string
  created_at: string
}

export interface SupplierAutomationRunDetail {
  providers?: SupplierAutomationProviderRunDetail[]
  rate_guard?: SupplierRateGuardResult
  account_rate_guard?: SupplierAccountRateGuardResult
  account_health_guard?: SupplierAccountHealthGuardResult
  cleanup?: SupplierAutomationCleanupRunDetail
}

export type SupplierAccountRateGuardRunMode = 'preview' | 'execute'

export interface SupplierAccountRateGuardResult {
  mode: SupplierAccountRateGuardRunMode
  checked_providers: number
  rate_sync_success_providers: number
  rate_sync_failed_providers: number
  checked_accounts: number
  risk_groups: number
  unbound_groups: number
  disabled_accounts: number
  skipped: number
  failed: number
}

export interface SupplierAccountHealthGuardSource {
  provider_id: number
  provider_name: string
  supplier_provider_account_id: number
  upstream_account_key: string
  upstream_account_name: string
}

export interface SupplierAccountHealthGuardSkippedAccount {
  local_account_id?: number
  local_account_name?: string
  supplier_provider_account_id?: number
  upstream_account_name?: string
}

export interface SupplierAccountHealthGuardSkipReason {
  reason: string
  count: number
  sample_accounts?: SupplierAccountHealthGuardSkippedAccount[]
}

export interface SupplierAccountHealthGuardItem {
  local_account_id: number
  local_account_name: string
  platform: string
  sources?: SupplierAccountHealthGuardSource[]
  match_status?: string
  model_id?: string
  schedulable_before: boolean
  schedulable_after: boolean
  status: 'healthy' | 'slow' | 'failed' | 'skipped' | 'unavailable' | string
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

export interface SupplierAccountHealthGuardResult {
  total_accounts: number
  selected_count: number
  checked_count: number
  healthy_count: number
  slow_count: number
  failed_count: number
  skipped_count: number
  unavailable_count: number
  pending_count: number
  disabled_count: number
  recovered_count: number
  unchanged_count: number
  cursor_account_id: number
  skip_reasons?: SupplierAccountHealthGuardSkipReason[]
  items: SupplierAccountHealthGuardItem[]
}

export interface SupplierRateGuardResult {
  checked: number
  raised: number
  unchanged: number
  duplicate: number
  stale: number
  invalid: number
  failed: number
  items: SupplierRateGuardItemResult[]
}

export interface SupplierRateGuardItemResult {
  mapping_id: number
  provider_id: number
  provider_name: string
  upstream_group_key: string
  upstream_group_name: string
  local_group_id: number
  local_group_name: string
  snapshot_at: string
  old_rate: number
  target_rate: number
  action: string
  reason?: string
}

export interface SupplierAutomationProviderRunDetail {
  provider_id: number
  provider_name: string
  scope: string
  status: string
  message: string
  counts: SupplierSyncCounts
  stages?: SupplierAutomationStageRunDetail[]
  started_at: string
  finished_at: string
}

export interface SupplierAutomationStageRunDetail {
  scope: string
  status: string
  message: string
  counts: SupplierSyncCounts
  endpoint?: string
  http_status?: number
  duration_ms?: number
  response_bytes?: number
  response_summary?: string
  parsed_summary?: string
  parse_error?: string
  error?: string
}

export interface SupplierAutomationCleanupRunDetail {
  automation_runs: number
  sync_runs: number
  metric_snapshots: number
  daily_stats: number
  accounts: number
  groups: number
}

export interface SupplierSyncCounts {
  checked_count: number
  created_count: number
  updated_count: number
  skipped_count: number
}

export interface SupplierAutomationRunListParams {
  task_code?: string
  status?: string
  page?: number
  page_size?: number
}

export interface SupplierAutomationRunListResult {
  items: SupplierAutomationRun[]
  total: number
  page: number
  page_size: number
}

export interface SupplierRateGuardChangeLog {
  id: number
  mapping_id: number
  local_group_id: number
  local_group_name: string
  upstream_group_key: string
  upstream_group_name: string
  old_rate: number
  new_rate: number
  status: 'pending' | 'handled'
  changed_at: string
  handled_at?: string
  created_at: string
}

export interface SupplierRateGuardChangeLogListParams {
  page?: number
  page_size?: number
}

export interface SupplierRateGuardChangeLogListResult {
  items: SupplierRateGuardChangeLog[]
  total: number
  pending_count: number
  page: number
  page_size: number
}

export interface SupplierAccountRateGuardUnbindLog {
  id: number
  run_id: number
  provider_id: number
  provider_name: string
  supplier_provider_account_id: number
  upstream_account_key: string
  upstream_account_name: string
  local_account_id: number
  local_account_name: string
  local_group_id: number
  local_group_name: string
  platform: string
  raw_upstream_rate: number
  rate_scale: number
  effective_upstream_rate: number
  local_group_rate: number
  mode: SupplierAccountRateGuardRunMode
  result: 'planned' | 'unbound' | 'failed' | 'skipped'
  status: 'pending' | 'handled'
  handled_at?: string
  before_bound: boolean
  after_bound: boolean
  before_schedulable?: boolean
  after_schedulable?: boolean
  error_message: string
  created_at: string
}

export interface SupplierAccountRateGuardUnbindLogListParams {
  run_id?: number
  provider_id?: number
  local_account_id?: number
  search?: string
  result?: string
  mode?: string
  status?: string
  only_unbound?: boolean
  page?: number
  page_size?: number
}

export interface SupplierAccountRateGuardUnbindLogListResult {
  items: SupplierAccountRateGuardUnbindLog[]
  total: number
  pending_count: number
  page: number
  page_size: number
}

export async function listTasks(): Promise<SupplierAutomationTask[]> {
  const { data } = await apiClient.get<SupplierAutomationTask[]>(
    '/admin/supplier-management/automation/tasks'
  )
  return data
}

export async function updateTask(taskCode: string, payload: SupplierAutomationTask): Promise<SupplierAutomationTask> {
  const { data } = await apiClient.put<SupplierAutomationTask>(
    `/admin/supplier-management/automation/tasks/${taskCode}`,
    payload
  )
  return data
}

export async function runTask(
  taskCode: string,
  mode: SupplierAccountRateGuardRunMode = 'execute'
): Promise<SupplierAutomationRun> {
  // 健康守护可能按账号串行探测，默认 30s 会把仍在执行中的请求误判为失败。
  const timeout = taskCode === 'supplier_account_health_guard'
    ? 35 * 60 * 1000
    : undefined
  const { data } = await apiClient.post<SupplierAutomationRun>(
    `/admin/supplier-management/automation/tasks/${taskCode}/run`,
    { mode },
    timeout ? { timeout } : undefined
  )
  return data
}

export async function listRuns(params: SupplierAutomationRunListParams = {}): Promise<SupplierAutomationRunListResult> {
  const { data } = await apiClient.get<SupplierAutomationRunListResult>(
    '/admin/supplier-management/automation/runs',
    { params }
  )
  return data
}

export async function listRateGuardChangeLogs(
  params: SupplierRateGuardChangeLogListParams = {}
): Promise<SupplierRateGuardChangeLogListResult> {
  const { data } = await apiClient.get<SupplierRateGuardChangeLogListResult>(
    '/admin/supplier-management/automation/rate-guard-change-logs',
    { params }
  )
  return data
}

export async function markRateGuardChangeLogHandled(id: number): Promise<SupplierRateGuardChangeLog> {
  const { data } = await apiClient.post<SupplierRateGuardChangeLog>(
    `/admin/supplier-management/automation/rate-guard-change-logs/${id}/handled`
  )
  return data
}

export async function listAccountRateGuardUnbindLogs(
  params: SupplierAccountRateGuardUnbindLogListParams = {}
): Promise<SupplierAccountRateGuardUnbindLogListResult> {
  const { data } = await apiClient.get<SupplierAccountRateGuardUnbindLogListResult>(
    '/admin/supplier-management/automation/account-rate-guard-unbind-logs',
    { params }
  )
  return data
}

export async function markAccountRateGuardUnbindLogHandled(id: number): Promise<SupplierAccountRateGuardUnbindLog> {
  const { data } = await apiClient.post<SupplierAccountRateGuardUnbindLog>(
    `/admin/supplier-management/automation/account-rate-guard-unbind-logs/${id}/handled`
  )
  return data
}

export const supplierAutomationAPI = {
  listTasks,
  updateTask,
  runTask,
  listRuns,
  listRateGuardChangeLogs,
  markRateGuardChangeLogHandled,
  listAccountRateGuardUnbindLogs,
  markAccountRateGuardUnbindLogHandled,
}

export default supplierAutomationAPI
