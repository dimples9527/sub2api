import { apiClient } from '../client'

export interface SupplierAutomationConfig {
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
  cleanup?: SupplierAutomationCleanupRunDetail
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

export async function runTask(taskCode: string): Promise<SupplierAutomationRun> {
  const { data } = await apiClient.post<SupplierAutomationRun>(
    `/admin/supplier-management/automation/tasks/${taskCode}/run`
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

export const supplierAutomationAPI = {
  listTasks,
  updateTask,
  runTask,
  listRuns,
}

export default supplierAutomationAPI
