import { apiClient } from '../client'

export interface SupplierAutomationConfig {
  rate_guard_safety_multiplier: number
  rate_guard_max_snapshot_age_seconds: number
  automation_run_retention_days: number
  sync_run_retention_days: number
  metric_snapshot_retention_days: number
  daily_stat_retention_days: number
  inactive_account_retention_days: number
  inactive_group_retention_days: number
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
  raw_upstream_rate: number
  rate_scale: number
  effective_upstream_rate: number
  local_group_rate: number
  mode: SupplierAccountRateGuardRunMode
  result: 'planned' | 'unbound' | 'failed' | 'skipped'
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
  page?: number
  page_size?: number
}

export interface SupplierAccountRateGuardUnbindLogListResult {
  items: SupplierAccountRateGuardUnbindLog[]
  total: number
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
  const { data } = await apiClient.post<SupplierAutomationRun>(
    `/admin/supplier-management/automation/tasks/${taskCode}/run`,
    { mode }
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

export const supplierAutomationAPI = {
  listTasks,
  updateTask,
  runTask,
  listRuns,
  listRateGuardChangeLogs,
  markRateGuardChangeLogHandled,
  listAccountRateGuardUnbindLogs,
}

export default supplierAutomationAPI
