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
  account_health_guard_account_intervals: Record<string, number>
  account_health_guard_account_scheduling_change: Record<string, boolean>
  /** 账号级阈值覆盖，键为本地账号 ID；未列出的账号沿用同名的全局阈值。 */
  account_health_guard_account_failure_thresholds: Record<string, number>
  account_health_guard_account_slow_thresholds: Record<string, number>
  account_health_guard_account_recovery_thresholds: Record<string, number>
  account_health_guard_cursor_account_id: number
  /**
   * 账号倍率守护按本地分组开关：这里存"被关闭守护"的分组 ID。
   * 空列表或不传 = 所有分组都参与守护（新增分组自动参与，无需补齐配置）。
   */
  account_rate_guard_disabled_group_ids?: number[]
  /** 分组择优调度：每个分组保持开启调度的最优账号数量，默认 1（严格单活）。 */
  group_scheduling_election_top_n?: number
  /**
   * 分组择优调度按本地分组开关：这里存"不参与择优"的分组 ID。
   * 空列表或不传 = 所有分组都参与择优（新增分组自动参与，无需补齐配置）。
   */
  group_scheduling_election_disabled_group_ids?: number[]
  /**
   * 分组择优调度综合分里「连续成功次数」的权重，默认 1。
   * 两项各自归一到 [0,1] 后加权，所以与用时权重的比值就是两者的相对话语权。
   */
  group_scheduling_election_count_weight?: number
  /** 分组择优调度综合分里「测试用时」的权重，默认 0.5（影响上限为次数的一半）。 */
  group_scheduling_election_latency_weight?: number
  /**
   * 分组择优调度：测试失败的账号要连续失败多少轮才关闭调度，默认 2（一次抖动不关）。
   * 配成 1 即退回「一次失败立刻关」；分组里没有备选账号时无论阈值多少都不关。
   */
  group_scheduling_election_failure_threshold?: number
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
  supplier_monitor?: SupplierProviderMonitorSyncResult
  recharge_sync?: SupplierProviderRechargeSyncAllResult
  cleanup?: SupplierAutomationCleanupRunDetail
  group_election?: SupplierGroupSchedulingElectionResult
}

export interface SupplierGroupSchedulingElectionGroupDetail {
  group_id: number
  group_name: string
  member_count: number
  success_count: number
  failed_count: number
  untested_count: number
  winner_count: number
  winner_ids?: number[]
}

export interface SupplierGroupSchedulingElectionAccountItem {
  account_id: number
  account_name: string
  platform?: string
  test_status?: string
  healthy_count: number
  latency_ms?: number
  schedulable_before: boolean
  schedulable_after: boolean
  action: string
  reason?: string
  group_ids?: number[]
  error_message?: string
}

export interface SupplierGroupSchedulingElectionResult {
  top_n: number
  group_count: number
  account_count: number
  enabled_count: number
  disabled_count: number
  unchanged_count: number
  skipped_count: number
  failed_write_count: number
  /** 连续失败未达阈值、本轮"故意没关"的账号数。旧运行记录没有这个字段。 */
  pending_count?: number
  /** 分组内无备选账号而保留调度、待人工确认的账号数。旧运行记录没有这个字段。 */
  kept_count?: number
  groups: SupplierGroupSchedulingElectionGroupDetail[]
  items: SupplierGroupSchedulingElectionAccountItem[]
}

export interface SupplierProviderRechargeSyncResult {
  provider_id: number
  provider_name: string
  status: string
  message: string
  record_count: number
  synced_at: string
}

export interface SupplierProviderRechargeSyncAllResult {
  items: SupplierProviderRechargeSyncResult[]
  success_count: number
  failed_count: number
}

export interface SupplierProviderMonitorSyncResult {
  processed_count: number
  success_count: number
  failed_count: number
  skipped_count: number
  items: SupplierProviderMonitorSyncItem[]
}

export interface SupplierProviderMonitorSyncItem {
  provider_id: number
  provider_name: string
  provider_type: string
  local_account_id?: number
  local_account_name?: string
  local_group_ids?: number[]
  local_group_names?: string[]
  upstream_key?: string
  upstream_name: string
  monitor_provider?: string
  primary_model?: string
  status: string
  raw_status?: string
  latency_ms: number
  ping_latency_ms?: number
  availability_7d?: number
  checked_at: string
  message?: string
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

// 一条「某账号的调度开关真的被拨动」的记录。
// 与运行明细的区别：运行明细是「一次任务执行」视角（含未变更和写库失败），
// 这里只保留开关前后不一致的条目。
export interface SupplierGroupElectionChangeLog {
  run_id: number
  run_status: string
  changed_at: string
  account_id: number
  account_name: string
  platform?: string
  test_status?: string
  healthy_count: number
  latency_ms?: number
  schedulable_before: boolean
  schedulable_after: boolean
  direction: 'enabled' | 'disabled'
  action: string
  reason?: string
  error_message?: string
  group_ids?: number[]
  group_names?: string[]
}

export interface SupplierGroupElectionChangeLogListParams {
  group_id?: number
  account_id?: number
  search?: string
  direction?: 'enabled' | 'disabled'
  started_from?: string
  started_to?: string
  page?: number
  page_size?: number
}

export interface SupplierGroupElectionChangeLogListResult {
  items: SupplierGroupElectionChangeLog[]
  total: number
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

// 一键处理的返回。has_more 表示"这次打满了后端单批上限、后面大概率还有"，
// 前端据此决定要不要再发一次 —— 不能靠 handled < 期望条数去猜。
export interface SupplierAccountRateGuardUnbindLogBatchHandledResult {
  handled: number
  batch: number
  has_more: boolean
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

// 一键处理：把「当前筛选条件下的全部待处理」标记为已处理。
// 筛选走 query 而不是 body —— 与列表接口共用同一套参数名，页面上怎么筛的这里就怎么传，
// 不会出现"列表带了筛选、批量忘了带"的口径漂移。page / page_size 有意不透传（批量不分页）。
export async function markAccountRateGuardUnbindLogsHandled(
  params: SupplierAccountRateGuardUnbindLogListParams = {}
): Promise<SupplierAccountRateGuardUnbindLogBatchHandledResult> {
  const { data } = await apiClient.post<SupplierAccountRateGuardUnbindLogBatchHandledResult>(
    '/admin/supplier-management/automation/account-rate-guard-unbind-logs/handled-batch',
    null,
    { params }
  )
  return data
}

export async function listGroupElectionChangeLogs(
  params: SupplierGroupElectionChangeLogListParams = {}
): Promise<SupplierGroupElectionChangeLogListResult> {
  const { data } = await apiClient.get<SupplierGroupElectionChangeLogListResult>(
    '/admin/supplier-management/automation/group-election-change-logs',
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
  listGroupElectionChangeLogs,
  markRateGuardChangeLogHandled,
  listAccountRateGuardUnbindLogs,
  markAccountRateGuardUnbindLogHandled,
  markAccountRateGuardUnbindLogsHandled,
}

export default supplierAutomationAPI
