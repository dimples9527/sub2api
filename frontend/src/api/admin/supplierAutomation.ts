import { apiClient } from '../client'

/** 单条「倍率区间 → 检查间隔」规则：区间取 [min, max)，max<=0 表示无上界。 */
export interface SupplierAccountHealthGuardMultiplierInterval {
  min_multiplier: number
  max_multiplier: number
  interval_seconds: number
}

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
  /**
   * 健康守护的全局「修改调度」开关，默认 true（连续失败/慢响应达到阈值时自动暂停调度、恢复达标后自动恢复）。
   * 上面账号级映射里存在的键覆盖它：true/false 都是显式覆盖，键不存在则跟随本开关。
   */
  account_health_guard_scheduling_change_enabled: boolean
  /** 未开调度账号按平台倍率区间取检查间隔：总开关。 */
  account_health_guard_platform_multiplier_intervals_enabled: boolean
  /** 平台 → 「倍率区间 → 间隔秒」规则；仅在总开关开启时对未开调度账号生效，覆盖账号级间隔。 */
  account_health_guard_platform_multiplier_intervals?: Record<string, SupplierAccountHealthGuardMultiplierInterval[]>
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
   * 分组择优调度：每组开启账号数的分组级覆盖，分组 ID → 该组的 TopN，命中即替换全局值，
   * 未命中的分组沿用全局 group_scheduling_election_top_n。空对象或不传 = 所有分组都用全局值（默认）。
   */
  group_scheduling_election_top_n_by_group?: Record<number, number>
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
   * 分组择优调度综合分里「账号优先级」的权重，默认 0.5。
   * 优先级分在同一分组内 min-max 归一化：数值越小优先级越高 → 1.0，最大者 → 0；
   * 优先级全都相同或极差为 0 时给中性分 0.5。只有开启了「优先级计分」的分组才会计入综合分。
   */
  group_scheduling_election_priority_weight?: number
  /**
   * 分组择优调度：账号优先级是否参与择优的全局默认开关，默认 false（不参与，与升级前一致）。
   * 打开后所有分组默认计优先级；个别分组可用下面的 force-on / force-off 名单反向覆盖。
   */
  group_scheduling_election_priority_enabled_global?: boolean
  /** 分组择优调度：强制开启「优先级计分」的分组 ID 列表（force-on，空=不强制，默认）。 */
  group_scheduling_election_priority_enabled_group_ids?: number[]
  /** 分组择优调度：强制关闭「优先级计分」的分组 ID 列表（force-off，优先级最高，空=无排除，默认）。 */
  group_scheduling_election_priority_disabled_group_ids?: number[]
  /**
   * 分组择优调度：测试失败的账号要连续失败多少轮才关闭调度，默认 2（一次抖动不关）。
   * 配成 1 即退回「一次失败立刻关」；分组里没有备选账号时无论阈值多少都不关。
   */
  group_scheduling_election_failure_threshold?: number
  /**
   * 分组择优调度：换人的迟滞死区，取延迟的相对比例，默认 0.15（挑战者要快 15% 才换掉在任者）。
   * 用来压住两个正常账号因延迟抖动每轮对拍、反复开关调度的抖动；0 或缺失回落默认值。
   */
  group_scheduling_election_switch_margin?: number
  /** 分组择优调度：「最近平均延迟」的时间窗（分钟），默认 30，取健康历史窗口内成功样本求平均。 */
  group_scheduling_election_latency_window_minutes?: number
  /** 分组择优调度：信任窗口平均所需的最少成功样本数，默认 3，不足则回退到最近一次单值。 */
  group_scheduling_election_latency_min_samples?: number
  /**
   * 分组择优调度：连续成功次数对综合分的贡献上限，默认 10。
   * 次数分 = min(连续成功次数, 上限) / 上限，所以达到上限的账号得分完全相同
   * （默认值下连续成功 142 次与 190 次没有差别）。调大它会让长期稳定的账号更难被更快的账号换掉；
   * 0 或缺失回落默认值，上限 100。
   */
  group_scheduling_election_count_score_cap?: number
  /**
   * 分组择优调度：启用「在任者健康锁定」的分组 ID 列表（opt-in，空=都不锁定，默认）。
   * 列表内分组只要「当前开着调度的账号」测试都正常（且没有开着却失败的），就保留现状、
   * 跳过择优与换人；一旦有开着的账号失败，仍走正常择优交给失败闸门处理。
   */
  group_scheduling_election_keep_healthy_incumbent_group_ids?: number[]
  /**
   * 在任者健康锁定的全局默认开关（默认 false=不锁定，与升级前一致）。
   * 分组级的强制锁定/强制不锁定名单可以覆盖它：排除名单→false、锁定名单→true、都不在→跟随全局。
   */
  group_scheduling_election_keep_healthy_incumbent_global?: boolean
  /**
   * 在任者健康锁定的「强制不锁定」分组 ID 列表（force-off，空=无排除，默认）。
   * 优先级最高：在此名单内的分组无论全局开关如何都不锁定，即便同时出现在强制锁定名单里也以此为准。
   */
  group_scheduling_election_keep_healthy_incumbent_excluded_group_ids?: number[]
  /**
   * 分组必需模型：分组 ID → 该分组必须能服务的模型名列表。
   * 择优后对每个必需模型做覆盖兜底：赢家没覆盖它就补选一个健康支持者开启；
   * 支持它的账号全部测试失败则不硬留（切到能用的账号），只记一条告警待人工恢复。
   * 空 = 该分组无强制模型要求（默认）。
   */
  group_scheduling_election_required_models?: Record<number, string[]>
  /**
   * 分组择优调度演练模式总开关：打开后只算出「该开谁、该关谁」并照常写切换日志，
   * 但不修改 accounts.schedulable，日志里这些条目标记为建议。
   * 用于改权重/阈值后先空跑几轮核对，而不是直接拿线上调度做实验。
   */
  group_scheduling_election_dry_run?: boolean
  /**
   * 分组择优调度演练分组名单（opt-in，空=都不演练，默认）。
   * 与总开关是并集：总开关打开则全部分组演练；否则只有列表内的分组演练，
   * 让「只盯住一两个分组」不必让全部分组同时失去择优。
   */
  group_scheduling_election_dry_run_group_ids?: number[]
  /**
   * 分组账号异常推送总开关（默认关闭）。
   * 打开后：分组里**当前真的在任**的账号，若延迟窗口内的成功样本数不足
   * group_scheduling_election_latency_min_samples，就通过供应商通知模块推一条
   * group_account_abnormal 事件（需先在通知页订阅该事件）。只影响通知，不影响任何调度裁决。
   */
  group_scheduling_election_alert_enabled?: boolean
  /**
   * 按分组覆盖推送开关：分组 ID → 是否推送。
   * 命中即替换总开关，未命中的分组仍跟随总开关（与 top_n_by_group 的覆盖语义一致）：
   * 既能在大开关打开时静音个别长期样本不足的分组，也能在大开关关闭时单独盯住某个分组。
   * 空对象或不传 = 所有分组都用总开关（默认）。
   *
   * 2026-10-09 由「按账号」改为「按分组」：通知的事件聚合 / 订阅 / 冷却本来就是分组级，
   * 账号级覆盖在多活分组里静音不了整组 —— 同组其它样本不足的账号仍会触发推送。
   */
  group_scheduling_election_alert_group_overrides?: Record<number, boolean>
  /**
   * 分组默认账号：分组 ID → 账号 ID。
   * 该账号在本组具备参选资格（测试成功未过期，或健康守护连续成功计数 > 0）时，
   * 本组只开它一个、关闭其它成员（跳过在任者健康锁定与正常择优）；
   * 它不具备资格时回退到原有的锁定/择优流程，不会把分组关成空组。
   * 必需模型仍是硬底线：默认账号不覆盖某必需模型时会额外补选一个支持者（本组可能开 2 个）。
   * 空对象或不传 = 所有分组都不指定默认账号（默认）。
   */
  group_scheduling_election_default_account_by_group?: Record<number, number>
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
  /** 演练模式：本条只是建议，目标状态没有真的写库。旧运行记录没有这个字段。 */
  suggested?: boolean
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
  /** 配了必需模型但当前无健康账号可提供的数量（本轮已切到能用账号，仍需人工恢复）。旧运行记录没有。 */
  required_model_uncovered_count?: number
  /** 必需模型断供的明细告警。旧运行记录没有这个字段。 */
  required_model_warnings?: SupplierGroupSchedulingElectionRequiredModelWarning[]
  /** 演练模式：本轮处于演练配置下（总开关打开或配了演练分组）。旧运行记录没有这个字段。 */
  dry_run?: boolean
  /** 演练模式下「本应开启但没有真改」的账号数。旧运行记录没有这个字段。 */
  suggested_enabled_count?: number
  /** 演练模式下「本应关闭但没有真改」的账号数。旧运行记录没有这个字段。 */
  suggested_disabled_count?: number
  groups: SupplierGroupSchedulingElectionGroupDetail[]
  items: SupplierGroupSchedulingElectionAccountItem[]
}

export interface SupplierGroupSchedulingElectionRequiredModelWarning {
  group_id: number
  group_name?: string
  model: string
  failed_account_ids?: number[]
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
  /** 该账号本轮生效的检查间隔（秒）；0 表示每轮都测。不参与本轮检查的账号（不可用/未匹配/已停用）不返回。 */
  interval_seconds?: number
  /** 下次检查时间 = 上次检查时间 + 间隔；从未检查过或每轮都测时不返回。 */
  next_check_at?: string
  /**
   * 该账号的计费倍率，明细列表按它升序排序（未配置按 1.0 计，0 表示计费为 0）。
   * 不返回表示账号已查不到 —— 排序时排到最后，不能当成 0。
   */
  billing_rate_multiplier?: number
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
  account_health_history: number
  auth_events: number
  monitor_samples: number
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

// 「某账号在某分组里为什么是这个裁决」的依据，随运行明细一起落库。
// 一个账号同属多个分组时各组结论可以不同（在 A 组当选、在 B 组落选），所以按分组各存一条。
// ⚠️ 只有升级后新产生的运行才有这个字段，旧记录里没有，展示必须能降级。
export interface SupplierGroupElectionDecisionDetail {
  group_id: number
  group_name?: string
  /** false 表示该账号在该组当前不是「测试成功」，压根没进择优，下面评分字段全部无意义。 */
  scored?: boolean
  /** 是否在该组入选（择优前 N，或因覆盖必需模型被补选）。 */
  elected?: boolean
  /** 综合分 = count_weight × count_score + latency_weight × latency_score (+ priority_weight × priority_score)，各项都给出来便于复核。 */
  score?: number
  count_score?: number
  latency_score?: number
  priority_score?: number
  count_weight?: number
  latency_weight?: number
  priority_weight?: number
  /**
   * true 表示该组本轮把「账号优先级」计入了综合分（分组级开关的最终判定结果）。
   * 需要它是因为 priority_score 为 0 有两种含义：本组不计优先级，或计了但这个账号就是最低的 0 分；
   * priority_weight 两种情况都是配置值，也分不出来。
   * 旧运行记录没有这个字段（undefined），此时不做判断、按「计了」显示。
   */
  priority_enabled?: boolean
  /** 次数分的封顶值：连续成功次数超过它之后不再加分。 */
  count_score_cap?: number
  /** 真正参与评分的延迟（含成功率惩罚与在任者迟滞折算），与日志里显示的「测试用时」不是同一个数。 */
  effective_latency_ms?: number
  /** true 表示用时分取了中性值 0.5（同平台样本不足或极差为 0），不是算出来的。 */
  latency_fallback?: boolean
  /** 综合分在该组「测试成功账号」里的名次（1 起）与参评总数。 */
  rank?: number
  rank_total?: number
  /** 入选分数线（第 top_n 名的综合分），落选时用来说明「差多少」。 */
  winner_cutoff?: number
  top_n?: number
  /** true 表示该组本轮走「在任者健康锁定」，没做择优，此时名次只是参考。 */
  locked?: boolean
  /**
   * true 表示该账号在本组被「每组开启账号数」这个硬上限收敛掉了：本组当前开着的账号已经够数，
   * 本轮不再新开，超出的在任者在这里让位关闭（多组共用的账号会由那些分组保住）。
   * 与 locked 的分工：locked 说明「本组本轮没做择优」，over_capacity 说明「这个账号是这次收敛的代价」。
   */
  over_capacity?: boolean
  /**
   * true 表示这次收敛关闭还叠加了「本组必需模型已由保留的账号覆盖」：该账号不覆盖本组任何一个必需模型。
   * 它回答的是「同样超容量，为什么留下的是另一个账号」——只写 over_capacity 时，用户只能看到「超过上限」，
   * 解释不了自己手动开的那个账号为什么被换掉。旧运行记录没有这个字段，缺省按普通收敛显示。
   */
  over_capacity_required_model?: boolean
  /** 非空表示该账号是因「分组要求这些模型、而赢家里没人支持」被补选开启的。 */
  required_models?: string[]
  /** true 表示该组一个测试成功的账号都没有，失败账号因此保持原状待人工确认。 */
  no_alternative?: boolean
  /**
   * true 表示该账号在本组是被「分组保底」开启的：本组一个开启调度的账号都没有，兜底开它一个。
   * 它与 elected 同时为真（保底也是一种入选），但综合分不是前 N —— 多数情况它压根没参选（全失败/全未测），
   * 不单独标出来日志看起来像择优算错了。
   */
  keep_alive?: boolean
  /** true 表示该账号在该组当前是测试失败状态。 */
  test_failed?: boolean
  /**
   * 连续失败到第几轮（含本轮）与关闭阈值。仅 test_failed 为真时有值。
   * 失败闸门是按次数说话的：只标「测试失败」看不出这是第几次、还要几次才关。
   * ⚠️ 旧运行记录没有这两个字段，缺省时展示必须降级成不带次数的标签。
   */
  failed_count?: number
  failure_threshold?: number
  /**
   * true 表示该账号按名字匹配到的上游账号当前不可用（供应商被停用 / 上游账号下线），它因此被关闭。
   * 这类账号的 test_status / healthy_count 是上游停用后冻结的旧数据，日志上看起来一切正常 ——
   * 不单独标出来，前端只能落到「未参与择优」，运维看不出真实原因是上游掉了。
   */
  upstream_unavailable?: boolean
  /**
   * true 表示该账号在本组被「必需模型冗余收敛」关掉：本组必需模型已能由另一个账号单独完整覆盖，
   * 而它不覆盖全部必需模型 —— 再开着不产生任何覆盖增量，只是多一份成本。
   * 与 over_capacity 的分工：后者是「本组开多了」，这条是「必需模型那边一个账号就够、它的名额多余」。
   * 旧运行记录没有这个字段，缺省按普通收敛显示。
   */
  consolidated?: boolean
  /**
   * true 表示本组指定了「默认账号」、且它具备参选资格，本轮只开它一个（其余成员一律让位）。
   * 它与 elected 同时为真（默认账号也是一种入选），但入选依据是配置而不是综合分 ——
   * 不标出来，日志会把一个「指定」读成「算出来的最优」。
   */
  default_account?: boolean
  /**
   * true 表示本账号在本组被「默认账号」顶掉关闭 —— 它本身可能是健康的，只是本组指定了固定主账号。
   * 与 over_capacity / consolidated 分开：那两条是算法收敛的副作用，这一条是配置意图。
   * 旧运行记录没有这个字段，缺省按普通落选显示。
   */
  default_account_out?: boolean
}

// 一条「某账号的调度开关被拨动（或演练模式下被建议拨动）」的记录。
// 与运行明细的区别：运行明细是「一次任务执行」视角（含未变更和写库失败），
// 这里只保留开关前后不一致的条目。
// 注意 suggested：演练模式下取数条件（前后不一致）描述的是「想改成什么」而不是「改成了什么」，
// 前端必须靠它把建议与已生效区分开，否则回看日志会把没发生过的切换当成事实。
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
  /** 演练模式产生的建议切换：目标状态已算出，但没有真的写库。 */
  suggested?: boolean
  /** 按分组记录「为什么是它」：评分明细、是否因覆盖必需模型补选、是否走锁定等。旧记录无此字段。 */
  group_decisions?: SupplierGroupElectionDecisionDetail[]
}

export interface SupplierGroupElectionChangeLogListParams {
  group_id?: number
  account_id?: number
  /**
   * 锁定到若干次择优运行（任务批次），把几批放在一起对照。
   * 这里按语义收数组；「怎么序列化成 query」由 listGroupElectionChangeLogs 统一处理。
   */
  run_ids?: number[]
  /** 精确匹配平台（openai / anthropic …）。与 search 分开：search 是模糊匹配账号名或平台。 */
  platform?: string
  search?: string
  direction?: 'enabled' | 'disabled'
  started_from?: string
  started_to?: string
  /**
   * true 时把「本该动却没动」的记录也带上：在任者健康锁定、分组只剩它保留、
   * 连续失败未达阈值、写库失败，均标注原因并显示「本次跳过」。默认只看开关真被拨动的条目。
   */
  include_skipped?: boolean
  page?: number
  page_size?: number
}

/** 顶部快捷标签用的一条「最近批次」：带上变更时间，供前端显示时间流水号。 */
export interface SupplierGroupElectionRecentRun {
  run_id: number
  changed_at: string
}

export interface SupplierGroupElectionChangeLogListResult {
  items: SupplierGroupElectionChangeLog[]
  total: number
  page: number
  page_size: number
  /**
   * 最近若干次「确实拨动过开关」的批次（新到旧），给页面顶部的快捷筛选标签用。
   * 它**不受当前筛选影响**（后端只按任务类型取），所以点某个标签后其余标签不会消失。
   *
   * 带 changed_at 是为了显示时间流水号；「新到旧」按批次内的变更时间排，
   * **不是**按 run_id —— runs.id 是自增主键，多 worker 并发时会交错。
   */
  recent_runs?: SupplierGroupElectionRecentRun[]
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
  // run_ids 在入参里是数组（调用方按语义传），但 axios 默认会把数组序列化成
  // `run_ids[]=1&run_ids[]=2`，后端得用 c.QueryArray("run_ids[]") 才取得到 —— 又脆又丑。
  // 本仓既有做法是逗号分隔（同模块的 account-health/trends 就是这么传 ids 的），这里沿用。
  const { run_ids: runIDs, ...rest } = params
  const query: Record<string, unknown> = { ...rest }
  if (runIDs && runIDs.length > 0) {
    query.run_ids = runIDs.join(',')
  }
  const { data } = await apiClient.get<SupplierGroupElectionChangeLogListResult>(
    '/admin/supplier-management/automation/group-election-change-logs',
    { params: query }
  )
  return data
}

/**
 * 分组择优调度的诊断快照。
 *
 * 后端把「任务配置 + 各分组当前开启数 + 开启中的账号及其所属分组 + 最近几轮逐账号决策」
 * 拼成一段纯文本，前端只负责展示与复制、不做任何拼装 —— 这些数据的口径都在后端，
 * 前端再拼一遍必然出现两边不一致。
 *
 * 之所以需要它：这个任务不打印业务日志，而运行记录里的 items 只含「发生变化 / 被锁定保住 /
 * 被闸门拦住」的账号；「某个分组当前到底开着几个」只存在于 accounts.schedulable，
 * 以前只能手写 SQL 去查。有了它，发现分组异常时一键复制即可交给他人排查。
 */
export interface SupplierGroupElectionDiagnosticsResult {
  generated_at: string
  text: string
}

export async function getGroupElectionDiagnostics(
  params: { run_limit?: number } = {}
): Promise<SupplierGroupElectionDiagnosticsResult> {
  const { data } = await apiClient.get<SupplierGroupElectionDiagnosticsResult>(
    '/admin/supplier-management/automation/group-election-diagnostics',
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
  getGroupElectionDiagnostics,
  markRateGuardChangeLogHandled,
  listAccountRateGuardUnbindLogs,
  markAccountRateGuardUnbindLogHandled,
  markAccountRateGuardUnbindLogsHandled,
}

export default supplierAutomationAPI
