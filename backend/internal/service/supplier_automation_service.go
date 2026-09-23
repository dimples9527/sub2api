package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type supplierAutomationScheduleParser struct {
	parser cron.Parser
}

func (p supplierAutomationScheduleParser) Parse(expression string) (cron.Schedule, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, fmt.Errorf("cron expression is required")
	}
	if strings.HasPrefix(expression, "@every ") {
		duration, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(expression, "@every ")))
		if err != nil || duration <= 0 {
			return nil, fmt.Errorf("invalid fixed interval: %s", expression)
		}
	}
	return p.parser.Parse(expression)
}

var supplierAutomationCronParser = supplierAutomationScheduleParser{parser: cron.NewParser(
	cron.SecondOptional |
		cron.Minute |
		cron.Hour |
		cron.Dom |
		cron.Month |
		cron.Dow |
		cron.Descriptor,
)}

type SupplierAutomationTask struct {
	ID             int64                    `json:"id"`
	TaskCode       string                   `json:"task_code"`
	Name           string                   `json:"name"`
	Enabled        bool                     `json:"enabled"`
	CronExpression string                   `json:"cron_expression"`
	TimeoutSeconds int                      `json:"timeout_seconds"`
	Config         SupplierAutomationConfig `json:"config"`
	LastStatus     string                   `json:"last_status"`
	LastMessage    string                   `json:"last_message"`
	LastRunAt      *time.Time               `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time               `json:"next_run_at,omitempty"`
}

type SupplierAutomationConfig struct {
	AutomationRunRetentionDays     int `json:"automation_run_retention_days"`
	SyncRunRetentionDays           int `json:"sync_run_retention_days"`
	MetricRetentionDays            int `json:"metric_snapshot_retention_days"`
	DailyStatRetentionDays         int `json:"daily_stat_retention_days"`
	InactiveAccountDays            int `json:"inactive_account_retention_days"`
	InactiveGroupDays              int `json:"inactive_group_retention_days"`
	RateGuardMaxSnapshotAgeSeconds int `json:"rate_guard_max_snapshot_age_seconds"`

	// 账号倍率守护按本地分组开关：这里存"被关闭守护"的分组 ID，空列表表示全部分组都参与守护。
	// 之所以存关闭项而不是开启项，是为了让"新增分组默认开启"不需要任何数据迁移或补齐动作。
	AccountRateGuardDisabledGroupIDs []int64 `json:"account_rate_guard_disabled_group_ids"`

	AccountHealthGuardMaxAccountsPerRun        int               `json:"account_health_guard_max_accounts_per_run"`
	AccountHealthGuardConcurrency              int               `json:"account_health_guard_concurrency"`
	AccountHealthGuardTimeoutPerAccountSeconds int               `json:"account_health_guard_timeout_per_account_seconds"`
	AccountHealthGuardFailureThreshold         int               `json:"account_health_guard_failure_threshold"`
	AccountHealthGuardSlowThreshold            int               `json:"account_health_guard_slow_threshold"`
	AccountHealthGuardRecoveryThreshold        int               `json:"account_health_guard_recovery_threshold"`
	AccountHealthGuardHealthyLatencyMs         int64             `json:"account_health_guard_healthy_latency_ms"`
	AccountHealthGuardAccountIDs               []int64           `json:"account_health_guard_account_ids"`
	AccountHealthGuardAccountModels            map[int64]string  `json:"account_health_guard_account_models"`
	AccountHealthGuardPlatformModels           map[string]string `json:"account_health_guard_platform_models"`
	AccountHealthGuardPlatformLatencyMs        map[string]int64  `json:"account_health_guard_platform_latency_ms"`
	AccountHealthGuardAccountIntervals         map[int64]int     `json:"account_health_guard_account_intervals"`
	AccountHealthGuardAccountSchedulingChange  map[int64]bool    `json:"account_health_guard_account_scheduling_change"`
	// 账号级阈值覆盖，未列出的账号沿用上面的全局阈值。
	AccountHealthGuardAccountFailureThresholds  map[int64]int `json:"account_health_guard_account_failure_thresholds"`
	AccountHealthGuardAccountSlowThresholds     map[int64]int `json:"account_health_guard_account_slow_thresholds"`
	AccountHealthGuardAccountRecoveryThresholds map[int64]int `json:"account_health_guard_account_recovery_thresholds"`
	// 未开调度账号按平台倍率区间取检查间隔：总开关 + 每平台的「倍率区间 → 间隔秒」规则。
	AccountHealthGuardPlatformMultiplierIntervalsEnabled bool                                                      `json:"account_health_guard_platform_multiplier_intervals_enabled"`
	AccountHealthGuardPlatformMultiplierIntervals        map[string][]SupplierAccountHealthGuardMultiplierInterval `json:"account_health_guard_platform_multiplier_intervals"`
	AccountHealthGuardCursorAccountID                    int64                                                     `json:"account_health_guard_cursor_account_id"`

	// 分组择优调度：每组保持开启的最优账号数（默认 1），以及不参与择优的分组 ID 列表（空=全部参与）。
	GroupElectionTopN             int     `json:"group_scheduling_election_top_n"`
	GroupElectionDisabledGroupIDs []int64 `json:"group_scheduling_election_disabled_group_ids"`
	// 两项各自归一到 [0,1] 后加权，比值即相对话语权；缺省由归一化回落默认值。
	GroupElectionCountWeight   float64 `json:"group_scheduling_election_count_weight"`
	GroupElectionLatencyWeight float64 `json:"group_scheduling_election_latency_weight"`
	// 连续失败多少个调度周期才真正关闭调度，默认 2（一次抖动不关）；配 1 即退回"失败即关"。
	GroupElectionFailureThreshold int `json:"group_scheduling_election_failure_threshold"`
	// 换人迟滞死区，取延迟相对比例（默认 0.15=挑战者要快 15% 才换人），压住正常账号反复对拍的抖动。
	GroupElectionSwitchMargin float64 `json:"group_scheduling_election_switch_margin"`
	// 最近平均延迟的时间窗（分钟，默认 30）：延迟改取健康历史窗口均值而非单次采样，从源头削抖。
	GroupElectionLatencyWindowMinutes int `json:"group_scheduling_election_latency_window_minutes"`
	// 信任窗口均值所需的最少成功样本数（默认 3）：不足则回退到最近单值，等于退回今天的行为。
	GroupElectionLatencyMinSamples int `json:"group_scheduling_election_latency_min_samples"`
	// 连续成功次数对综合分的贡献上限（默认 10）：次数分 = min(次数, 上限) / 上限，
	// 达到上限的账号得分完全相同。调大它会让长期稳定的账号更难被更快的账号换掉。
	GroupElectionCountScoreCap int `json:"group_scheduling_election_count_score_cap"`
	// 启用「在任者健康锁定」的分组 ID（opt-in，空=都不锁定）：列表内分组开着的账号测试都正常时保留现状、
	// 跳过择优换人，减少无谓抖动；一旦开着的账号失败仍走正常择优。
	GroupElectionKeepHealthyIncumbentGroupIDs []int64 `json:"group_scheduling_election_keep_healthy_incumbent_group_ids"`
	// 分组必需模型：group_id → 必须能服务的模型名列表。择优后对每个必需模型做覆盖兜底——赢家没覆盖它就
	// 补选一个健康支持者开启；支持它的账号全失败则不硬留、只告警待恢复。空=无强制要求。
	GroupElectionRequiredModels map[int64][]string `json:"group_scheduling_election_required_models"`
	// 演练模式：打开后只产出「建议切换」并照常写切换日志，但不拨 accounts.schedulable。
	// 分组名单是并集以外的第二档——只想盯住个别分组时用，避免全量演练让所有分组同时失去择优。
	GroupElectionDryRun          bool    `json:"group_scheduling_election_dry_run"`
	GroupElectionDryRunGroupIDs  []int64 `json:"group_scheduling_election_dry_run_group_ids"`
}

type SupplierAutomationRun struct {
	ID             int64                        `json:"id"`
	TaskCode       string                       `json:"task_code"`
	TriggerSource  string                       `json:"trigger_source"`
	Status         string                       `json:"status"`
	Message        string                       `json:"message"`
	ProcessedCount int                          `json:"processed_count"`
	SuccessCount   int                          `json:"success_count"`
	FailedCount    int                          `json:"failed_count"`
	ResultDetail   *SupplierAutomationRunDetail `json:"result_detail,omitempty"`
	StartedAt      time.Time                    `json:"started_at"`
	FinishedAt     *time.Time                   `json:"finished_at,omitempty"`
	CreatedAt      time.Time                    `json:"created_at"`
}

type SupplierAutomationRunDetail struct {
	Providers          []SupplierAutomationProviderRunDetail  `json:"providers,omitempty"`
	Cleanup            *SupplierAutomationCleanupRunDetail    `json:"cleanup,omitempty"`
	RateGuard          *SupplierRateGuardResult               `json:"rate_guard,omitempty"`
	AccountRateGuard   *SupplierAccountRateGuardResult        `json:"account_rate_guard,omitempty"`
	AccountHealthGuard *SupplierAccountHealthGuardResult      `json:"account_health_guard,omitempty"`
	SupplierMonitor    *SupplierProviderMonitorSyncResult     `json:"supplier_monitor,omitempty"`
	RechargeSync       *SupplierProviderRechargeSyncAllResult `json:"recharge_sync,omitempty"`
	GroupElection      *SupplierGroupSchedulingElectionResult `json:"group_election,omitempty"`
}

type SupplierAutomationProviderRunDetail struct {
	ProviderID   int64                              `json:"provider_id"`
	ProviderName string                             `json:"provider_name"`
	Scope        string                             `json:"scope"`
	Status       string                             `json:"status"`
	Message      string                             `json:"message"`
	Counts       SupplierSyncCounts                 `json:"counts"`
	Stages       []SupplierAutomationStageRunDetail `json:"stages,omitempty"`
	StartedAt    time.Time                          `json:"started_at"`
	FinishedAt   time.Time                          `json:"finished_at"`
}

type SupplierAutomationStageRunDetail struct {
	Scope           string             `json:"scope"`
	Status          string             `json:"status"`
	Message         string             `json:"message"`
	Counts          SupplierSyncCounts `json:"counts"`
	Endpoint        string             `json:"endpoint,omitempty"`
	HTTPStatus      int                `json:"http_status,omitempty"`
	DurationMS      int64              `json:"duration_ms,omitempty"`
	ResponseBytes   int                `json:"response_bytes,omitempty"`
	ResponseSummary string             `json:"response_summary,omitempty"`
	ParsedSummary   string             `json:"parsed_summary,omitempty"`
	ParseError      string             `json:"parse_error,omitempty"`
	Error           string             `json:"error,omitempty"`
}

type SupplierAutomationCleanupRunDetail struct {
	AutomationRuns       int `json:"automation_runs"`
	SyncRuns             int `json:"sync_runs"`
	MetricSnapshots      int `json:"metric_snapshots"`
	DailyStats           int `json:"daily_stats"`
	Accounts             int `json:"accounts"`
	Groups               int `json:"groups"`
	AccountHealthHistory int `json:"account_health_history"`
}

type SupplierAutomationRunListParams struct {
	TaskCode string
	Status   string
	Page     int
	PageSize int
}

type SupplierAutomationRunListResult struct {
	Items    []SupplierAutomationRun `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type SupplierAutomationRepository interface {
	ListTasks(ctx context.Context) ([]SupplierAutomationTask, error)
	GetTask(ctx context.Context, code string) (*SupplierAutomationTask, error)
	// UpdateTask 仅更新任务配置（开关/周期/超时/策略），不覆盖运行时状态。
	UpdateTask(ctx context.Context, task *SupplierAutomationTask) error
	// UpdateTaskRuntime 仅更新最近执行状态与调度时间，不覆盖任务配置。
	UpdateTaskRuntime(ctx context.Context, task *SupplierAutomationTask) error
	CreateRun(ctx context.Context, run *SupplierAutomationRun) error
	FinishRun(ctx context.Context, run *SupplierAutomationRun) error
	ListRuns(ctx context.Context, params SupplierAutomationRunListParams) (SupplierAutomationRunListResult, error)
	RecoverRunning(ctx context.Context, message string) error
}

type SupplierAutomationLock interface {
	TryAcquireAutomationLock(ctx context.Context, taskCode, owner string, ttl time.Duration) (bool, error)
	ReleaseAutomationLock(ctx context.Context, taskCode, owner string) error
	// ForceReleaseAutomationLock 用于服务启动恢复：清理可能因进程中断残留的任务锁。
	ForceReleaseAutomationLock(ctx context.Context, taskCode string) error
}

type SupplierAutomationSchedulerReloader interface {
	Reload(ctx context.Context) error
}

type SupplierProviderBatchSyncer interface {
	SyncAllEnabled(ctx context.Context, trigger string) (SupplierProviderBatchSyncResult, error)
}

type SupplierProviderRechargeSyncer interface {
	SyncAll(ctx context.Context, fullSync bool) (SupplierProviderRechargeSyncAllResult, error)
}

type SupplierRateGuardRunner interface {
	Run(ctx context.Context, config SupplierRateGuardConfig, now time.Time) (SupplierRateGuardResult, error)
}

type SupplierAccountRateGuardRunner interface {
	Run(ctx context.Context, runID int64, mode SupplierAccountRateGuardMode, disabledGroupIDs []int64, now time.Time) (SupplierAccountRateGuardResult, error)
}

const (
	SupplierAutomationRunModePreview = "preview"
	SupplierAutomationRunModeExecute = "execute"

	SupplierAutomationTaskSync               = "supplier_data_sync"
	SupplierAutomationTaskMonitorSync        = "supplier_monitor_sync"
	SupplierAutomationTaskCleanup            = "supplier_data_cleanup"
	SupplierAutomationTaskRateGuard          = "supplier_rate_guard"
	SupplierAutomationTaskAccountRateGuard   = "supplier_account_rate_guard"
	SupplierAutomationTaskAccountHealthGuard = "supplier_account_health_guard"
	SupplierAutomationTaskRechargeSync       = "supplier_provider_recharge_sync"
	SupplierAutomationTaskGroupElection      = "supplier_group_scheduling_election"

	SupplierAutomationStatusRunning = "running"
	SupplierAutomationStatusSuccess = "success"
	SupplierAutomationStatusPartial = "partial"
	SupplierAutomationStatusFailed  = "failed"

	supplierAutomationUpstreamFetchLock              = "supplier_upstream_fetch"
	supplierAutomationUpstreamFetchLockRetryInterval = time.Second
)

func supplierAutomationConfigJSON(config SupplierAutomationConfig) string {
	raw, _ := json.Marshal(config)
	return string(raw)
}

type SupplierAutomationService struct {
	repo               SupplierAutomationRepository
	lock               SupplierAutomationLock
	syncer             SupplierProviderBatchSyncer
	monitorSyncer      SupplierProviderMonitorSyncer
	dataRepo           SupplierProviderDataRepository
	rateGuard          SupplierRateGuardRunner
	accountRateGuard   SupplierAccountRateGuardRunner
	accountHealthGuard SupplierAccountHealthGuardRunner
	accountRateLogs    SupplierAccountRateGuardRepository
	rechargeSyncer     SupplierProviderRechargeSyncer
	groupElection      SupplierGroupSchedulingElectionRunner
	reloader           SupplierAutomationSchedulerReloader
}

func NewSupplierAutomationService(repo SupplierAutomationRepository, lock SupplierAutomationLock, syncer SupplierProviderBatchSyncer, dataRepo SupplierProviderDataRepository) *SupplierAutomationService {
	return &SupplierAutomationService{repo: repo, lock: lock, syncer: syncer, dataRepo: dataRepo}
}

func (s *SupplierAutomationService) SetSchedulerReloader(reloader SupplierAutomationSchedulerReloader) {
	s.reloader = reloader
}

func (s *SupplierAutomationService) SetMonitorSyncService(syncer SupplierProviderMonitorSyncer) {
	if s != nil {
		s.monitorSyncer = syncer
	}
}

func (s *SupplierAutomationService) SetRechargeSyncService(syncer SupplierProviderRechargeSyncer) {
	if s != nil {
		s.rechargeSyncer = syncer
	}
}

func (s *SupplierAutomationService) SetRateGuardService(rateGuard SupplierRateGuardRunner) {
	if s != nil {
		s.rateGuard = rateGuard
	}
}

func (s *SupplierAutomationService) SetAccountRateGuardService(rateGuard SupplierAccountRateGuardRunner) {
	if s != nil {
		s.accountRateGuard = rateGuard
	}
}

func (s *SupplierAutomationService) SetAccountHealthGuardService(guard SupplierAccountHealthGuardRunner) {
	if s != nil {
		s.accountHealthGuard = guard
	}
}

func (s *SupplierAutomationService) SetAccountRateGuardRepository(repository SupplierAccountRateGuardRepository) {
	if s != nil {
		s.accountRateLogs = repository
	}
}

func (s *SupplierAutomationService) SetGroupSchedulingElectionService(election SupplierGroupSchedulingElectionRunner) {
	if s != nil {
		s.groupElection = election
	}
}

func (s *SupplierAutomationService) ListTasks(ctx context.Context) ([]SupplierAutomationTask, error) {
	return s.repo.ListTasks(ctx)
}

func (s *SupplierAutomationService) UpdateTask(ctx context.Context, task *SupplierAutomationTask) error {
	if task == nil {
		return ErrSupplierProviderInvalid
	}
	if err := validateSupplierAutomationTask(*task); err != nil {
		return err
	}
	if err := validateSupplierAccountHealthGuardSelection(*task); err != nil {
		return err
	}
	if err := s.repo.UpdateTask(ctx, task); err != nil {
		return err
	}
	if s.reloader != nil {
		return s.reloader.Reload(ctx)
	}
	return nil
}

func (s *SupplierAutomationService) ListRuns(ctx context.Context, params SupplierAutomationRunListParams) (SupplierAutomationRunListResult, error) {
	return s.repo.ListRuns(ctx, params)
}

func (s *SupplierAutomationService) ListRateGuardChangeLogs(ctx context.Context, params SupplierRateGuardChangeLogListParams) (SupplierRateGuardChangeLogListResult, error) {
	store, ok := s.dataRepo.(SupplierRateGuardChangeLogStore)
	if !ok {
		return SupplierRateGuardChangeLogListResult{}, fmt.Errorf("supplier rate guard change log store is required")
	}
	return store.ListRateGuardChangeLogs(ctx, params)
}

// ListGroupSchedulingElectionChangeLogs 返回「账号调度开关被拨动」的扁平日志。
// 走 s.dataRepo 的窄接口断言而不是新增构造参数 —— 加参数会让 Wire 生成代码失效，
// 而这个仓库本来就已经注入进来了。
func (s *SupplierAutomationService) ListGroupSchedulingElectionChangeLogs(ctx context.Context, params SupplierGroupSchedulingElectionChangeLogListParams) (SupplierGroupSchedulingElectionChangeLogListResult, error) {
	store, ok := s.dataRepo.(SupplierGroupSchedulingElectionChangeLogStore)
	if !ok {
		return SupplierGroupSchedulingElectionChangeLogListResult{}, fmt.Errorf("supplier group scheduling election change log store is required")
	}
	return store.ListGroupSchedulingElectionChangeLogs(ctx, params)
}

func (s *SupplierAutomationService) ListAccountRateGuardUnbindLogs(ctx context.Context, params SupplierAccountRateGuardUnbindLogListParams) (SupplierAccountRateGuardUnbindLogListResult, error) {
	if s.accountRateLogs == nil {
		return SupplierAccountRateGuardUnbindLogListResult{}, fmt.Errorf("supplier account rate guard log repository is required")
	}
	return s.accountRateLogs.ListAccountRateGuardUnbindLogs(ctx, params)
}

func (s *SupplierAutomationService) MarkAccountRateGuardUnbindLogHandled(ctx context.Context, id int64) (SupplierAccountRateGuardUnbindLog, error) {
	if s.accountRateLogs == nil {
		return SupplierAccountRateGuardUnbindLog{}, fmt.Errorf("supplier account rate guard log repository is required")
	}
	return s.accountRateLogs.MarkAccountRateGuardUnbindLogHandled(ctx, id)
}

// MarkAccountRateGuardUnbindLogsHandled 按筛选条件批量标记已处理。
// 入参复用列表的筛选结构，列表页看到什么、这里就处理什么 —— 两者口径必须一致，
// 否则会出现"看着有 30 条待处理、一键处理后还剩几条"的困惑。
// Page/PageSize 会被仓库层忽略（批量不翻页），但仍调用规范化以保证筛选字段被 trim。
func (s *SupplierAutomationService) MarkAccountRateGuardUnbindLogsHandled(ctx context.Context, params SupplierAccountRateGuardUnbindLogListParams) (SupplierAccountRateGuardUnbindLogBatchHandledResult, error) {
	if s.accountRateLogs == nil {
		return SupplierAccountRateGuardUnbindLogBatchHandledResult{}, fmt.Errorf("supplier account rate guard log repository is required")
	}
	return s.accountRateLogs.MarkAccountRateGuardUnbindLogsHandled(ctx, params)
}

func (s *SupplierAutomationService) MarkRateGuardChangeLogHandled(ctx context.Context, id int64) (SupplierRateGuardChangeLog, error) {
	store, ok := s.dataRepo.(SupplierRateGuardChangeLogStore)
	if !ok {
		return SupplierRateGuardChangeLog{}, fmt.Errorf("supplier rate guard change log store is required")
	}
	return store.MarkRateGuardChangeLogHandled(ctx, id)
}

func (s *SupplierAutomationService) Run(ctx context.Context, taskCode, trigger string) (SupplierAutomationRun, error) {
	return s.RunWithMode(ctx, taskCode, trigger, SupplierAutomationRunModeExecute)
}

func (s *SupplierAutomationService) RunWithMode(ctx context.Context, taskCode, trigger, mode string) (SupplierAutomationRun, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = SupplierAutomationRunModeExecute
	}
	if mode != SupplierAutomationRunModePreview && mode != SupplierAutomationRunModeExecute {
		return SupplierAutomationRun{}, ErrSupplierProviderInvalid
	}
	task, err := s.repo.GetTask(ctx, strings.TrimSpace(taskCode))
	if err != nil {
		return SupplierAutomationRun{}, err
	}
	if err := validateSupplierAutomationTask(*task); err != nil {
		return SupplierAutomationRun{}, err
	}
	if err := validateSupplierAccountHealthGuardSelection(*task); err != nil {
		return SupplierAutomationRun{}, err
	}
	if mode == SupplierAutomationRunModePreview && task.TaskCode != SupplierAutomationTaskAccountRateGuard {
		return SupplierAutomationRun{}, ErrSupplierProviderInvalid
	}
	owner := uuid.NewString()
	taskLockTTL := time.Duration(task.TimeoutSeconds+60) * time.Second
	if supplierAutomationTaskRequiresUpstreamFetchLock(task.TaskCode) {
		// 同类任务锁需覆盖等待共享锁和实际执行两个阶段，避免等待期间锁过期导致同任务重叠。
		taskLockTTL += time.Duration(task.TimeoutSeconds) * time.Second
	}
	if s.lock != nil {
		acquired, err := s.lock.TryAcquireAutomationLock(ctx, task.TaskCode, owner, taskLockTTL)
		if err != nil {
			return SupplierAutomationRun{}, err
		}
		if !acquired {
			// 定时触发冲突时写一条可见记录，避免“配置了间隔却完全没有执行痕迹”。
			run := SupplierAutomationRun{
				TaskCode:      task.TaskCode,
				TriggerSource: normalizeSupplierSyncTrigger(trigger),
				Status:        SupplierAutomationStatusFailed,
				Message:       "任务锁占用中，跳过本次执行",
				StartedAt:     time.Now(),
				CreatedAt:     time.Now(),
			}
			finishedAt := time.Now()
			run.FinishedAt = &finishedAt
			if createErr := s.repo.CreateRun(ctx, &run); createErr == nil {
				_ = s.repo.FinishRun(ctx, &run)
				task.LastStatus = run.Status
				task.LastMessage = run.Message
				task.LastRunAt = &finishedAt
				_ = s.repo.UpdateTaskRuntime(ctx, task)
			}
			return run, ErrSupplierProviderSyncConflict
		}
		defer func() { _ = s.lock.ReleaseAutomationLock(context.Background(), task.TaskCode, owner) }()
	}
	if s.lock != nil && supplierAutomationTaskRequiresUpstreamFetchLock(task.TaskCode) {
		waitCtx, waitCancel := context.WithTimeout(ctx, time.Duration(task.TimeoutSeconds)*time.Second)
		releaseUpstreamFetchLock, err := s.acquireUpstreamFetchLock(waitCtx, owner, time.Duration(task.TimeoutSeconds+60)*time.Second)
		waitCancel()
		if err != nil {
			return SupplierAutomationRun{}, err
		}
		defer releaseUpstreamFetchLock()
	}

	runCtx := ctx
	cancel := func() {}
	if task.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(task.TimeoutSeconds)*time.Second)
	}
	defer cancel()

	now := time.Now()
	run := SupplierAutomationRun{
		TaskCode:      task.TaskCode,
		TriggerSource: normalizeSupplierSyncTrigger(trigger),
		Status:        SupplierAutomationStatusRunning,
		StartedAt:     now,
		CreatedAt:     now,
	}
	if err := s.repo.CreateRun(ctx, &run); err != nil {
		return run, err
	}

	execErr := s.executeTask(runCtx, task, &run, mode)
	finishedAt := time.Now()
	run.FinishedAt = &finishedAt
	if execErr != nil {
		run.Status = SupplierAutomationStatusFailed
		run.Message = execErr.Error()
	} else if run.Status == SupplierAutomationStatusRunning {
		run.Status = SupplierAutomationStatusSuccess
		run.Message = "执行成功"
	}
	if finishErr := s.repo.FinishRun(ctx, &run); finishErr != nil && execErr == nil {
		execErr = finishErr
	}
	task.LastStatus = run.Status
	task.LastMessage = run.Message
	task.LastRunAt = &finishedAt
	// 运行结束只回写状态；健康守护游标基于最新配置合并，避免覆盖用户刚改的周期。
	_ = s.persistTaskRuntimeAfterRun(ctx, task)
	return run, execErr
}

// persistTaskRuntimeAfterRun 在任务执行结束后回写运行状态；健康守护还需安全合并游标。
func (s *SupplierAutomationService) persistTaskRuntimeAfterRun(ctx context.Context, task *SupplierAutomationTask) error {
	if task == nil {
		return nil
	}
	if task.TaskCode == SupplierAutomationTaskAccountHealthGuard {
		if err := s.persistAccountHealthGuardCursor(ctx, task.TaskCode, task.Config.AccountHealthGuardCursorAccountID); err != nil {
			return err
		}
	}
	return s.repo.UpdateTaskRuntime(ctx, task)
}

// persistAccountHealthGuardCursor 基于数据库中的最新配置只更新游标，避免用执行开始时的旧配置覆盖用户改动。
func (s *SupplierAutomationService) persistAccountHealthGuardCursor(ctx context.Context, taskCode string, cursorAccountID int64) error {
	current, err := s.repo.GetTask(ctx, taskCode)
	if err != nil {
		return err
	}
	if current.Config.AccountHealthGuardCursorAccountID == cursorAccountID {
		return nil
	}
	current.Config.AccountHealthGuardCursorAccountID = cursorAccountID
	return s.repo.UpdateTask(ctx, current)
}

// supplierAutomationTaskRequiresUpstreamFetchLock 标记会读取上游供应商数据的自动化任务。
func supplierAutomationTaskRequiresUpstreamFetchLock(taskCode string) bool {
	switch taskCode {
	case SupplierAutomationTaskSync, SupplierAutomationTaskMonitorSync, SupplierAutomationTaskAccountRateGuard, SupplierAutomationTaskRechargeSync:
		return true
	default:
		return false
	}
}

// acquireUpstreamFetchLock 让供应商数据同步和账号倍率守护串行拉取上游数据。
func (s *SupplierAutomationService) acquireUpstreamFetchLock(ctx context.Context, owner string, ttl time.Duration) (func(), error) {
	for {
		acquired, err := s.lock.TryAcquireAutomationLock(ctx, supplierAutomationUpstreamFetchLock, owner, ttl)
		if err != nil {
			return nil, fmt.Errorf("获取供应商上游拉取协调锁: %w", err)
		}
		if acquired {
			return func() {
				_ = s.lock.ReleaseAutomationLock(context.Background(), supplierAutomationUpstreamFetchLock, owner)
			}, nil
		}

		timer := time.NewTimer(supplierAutomationUpstreamFetchLockRetryInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, fmt.Errorf("等待供应商上游拉取任务完成: %w", ctx.Err())
		case <-timer.C:
		}
	}
}
func (s *SupplierAutomationService) executeTask(ctx context.Context, task *SupplierAutomationTask, run *SupplierAutomationRun, mode string) error {
	switch task.TaskCode {
	case SupplierAutomationTaskSync:
		result, err := s.syncer.SyncAllEnabled(ctx, SupplierSyncTriggerScheduled)
		run.ProcessedCount = result.ProcessedCount
		run.SuccessCount = result.SuccessCount
		run.FailedCount = result.FailedCount
		run.ResultDetail = supplierAutomationRunDetailFromBatch(result)
		if err != nil {
			return err
		}
		if result.FailedCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = supplierAutomationBatchFailureMessage(result)
		}
		return nil
	case SupplierAutomationTaskMonitorSync:
		if s.monitorSyncer == nil {
			return fmt.Errorf("supplier monitor sync service is required")
		}
		result, err := s.monitorSyncer.SyncMonitorsEnabled(ctx, SupplierSyncTriggerScheduled)
		run.ProcessedCount = result.ProcessedCount
		run.SuccessCount = result.SuccessCount
		run.FailedCount = result.FailedCount
		run.ResultDetail = &SupplierAutomationRunDetail{SupplierMonitor: &result}
		if err != nil {
			return err
		}
		if result.FailedCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = fmt.Sprintf("供应商监控数据同步存在 %d 个失败供应商", result.FailedCount)
		}
		return nil
	case SupplierAutomationTaskRechargeSync:
		if s.rechargeSyncer == nil {
			return fmt.Errorf("supplier recharge sync service is required")
		}
		result, err := s.rechargeSyncer.SyncAll(ctx, false)
		run.ProcessedCount = result.SuccessCount + result.FailedCount
		run.SuccessCount = result.SuccessCount
		run.FailedCount = result.FailedCount
		run.ResultDetail = &SupplierAutomationRunDetail{RechargeSync: &result}
		if err != nil {
			return err
		}
		if result.FailedCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = fmt.Sprintf("供应商充值记录同步存在 %d 个失败供应商", result.FailedCount)
		} else {
			run.Message = fmt.Sprintf("已同步 %d 个供应商的充值记录", result.SuccessCount)
		}
		return nil
	case SupplierAutomationTaskCleanup:
		counts, err := s.dataRepo.Cleanup(ctx, SupplierCleanupPolicy{
			AutomationRunRetentionDays:        task.Config.AutomationRunRetentionDays,
			SyncRunRetentionDays:              task.Config.SyncRunRetentionDays,
			MetricRetentionDays:               task.Config.MetricRetentionDays,
			DailyStatRetentionDays:            task.Config.DailyStatRetentionDays,
			InactiveAccountDays:               task.Config.InactiveAccountDays,
			InactiveGroupDays:                 task.Config.InactiveGroupDays,
			AccountHealthHistoryRetentionDays: 30,
		}, time.Now(), 1000)
		if err != nil {
			return err
		}
		run.ProcessedCount = counts.AutomationRuns + counts.SyncRuns + counts.MetricSnapshots + counts.DailyStats + counts.Accounts + counts.Groups + counts.AccountHealthHistory
		run.ResultDetail = &SupplierAutomationRunDetail{Cleanup: &SupplierAutomationCleanupRunDetail{
			AutomationRuns:       counts.AutomationRuns,
			SyncRuns:             counts.SyncRuns,
			MetricSnapshots:      counts.MetricSnapshots,
			DailyStats:           counts.DailyStats,
			Accounts:             counts.Accounts,
			Groups:               counts.Groups,
			AccountHealthHistory: counts.AccountHealthHistory,
		}}
		return nil
	case SupplierAutomationTaskRateGuard:
		if s.rateGuard == nil {
			return fmt.Errorf("supplier rate guard service is required")
		}
		result, err := s.rateGuard.Run(ctx, SupplierRateGuardConfig{
			MaxSnapshotAge: time.Duration(task.Config.RateGuardMaxSnapshotAgeSeconds) * time.Second,
		}, time.Now())
		run.ProcessedCount = result.Checked
		run.FailedCount = result.Failed + result.Stale + result.Invalid
		run.SuccessCount = result.Checked - run.FailedCount
		run.ResultDetail = &SupplierAutomationRunDetail{RateGuard: &result}
		if err != nil {
			return err
		}
		if run.FailedCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = fmt.Sprintf("倍率守护存在 %d 项告警", run.FailedCount)
		}
		return nil
	case SupplierAutomationTaskAccountHealthGuard:
		if s.accountHealthGuard == nil {
			return fmt.Errorf("supplier account health guard service is required")
		}
		result, err := s.accountHealthGuard.Run(ctx, SupplierAccountHealthGuardConfig{
			MaxAccountsPerRun:                  task.Config.AccountHealthGuardMaxAccountsPerRun,
			Concurrency:                        task.Config.AccountHealthGuardConcurrency,
			TimeoutPerAccountSeconds:           task.Config.AccountHealthGuardTimeoutPerAccountSeconds,
			FailureThreshold:                   task.Config.AccountHealthGuardFailureThreshold,
			SlowThreshold:                      task.Config.AccountHealthGuardSlowThreshold,
			RecoveryThreshold:                  task.Config.AccountHealthGuardRecoveryThreshold,
			HealthyLatencyMs:                   task.Config.AccountHealthGuardHealthyLatencyMs,
			AccountIDs:                         task.Config.AccountHealthGuardAccountIDs,
			AccountModels:                      task.Config.AccountHealthGuardAccountModels,
			PlatformModels:                     task.Config.AccountHealthGuardPlatformModels,
			PlatformLatencyMs:                  task.Config.AccountHealthGuardPlatformLatencyMs,
			AccountIntervals:                   task.Config.AccountHealthGuardAccountIntervals,
			AccountSchedulingChange:            task.Config.AccountHealthGuardAccountSchedulingChange,
			AccountFailureThresholds:           task.Config.AccountHealthGuardAccountFailureThresholds,
			AccountSlowThresholds:              task.Config.AccountHealthGuardAccountSlowThresholds,
			AccountRecoveryThresholds:          task.Config.AccountHealthGuardAccountRecoveryThresholds,
			PlatformMultiplierIntervalsEnabled: task.Config.AccountHealthGuardPlatformMultiplierIntervalsEnabled,
			PlatformMultiplierIntervals:        task.Config.AccountHealthGuardPlatformMultiplierIntervals,
			CursorAccountID:                    task.Config.AccountHealthGuardCursorAccountID,
		}, time.Now())
		run.ProcessedCount = result.CheckedCount + result.UnavailableCount
		run.SuccessCount = result.HealthyCount + result.SlowCount
		run.FailedCount = result.FailedCount + result.UnavailableCount
		run.ResultDetail = &SupplierAutomationRunDetail{AccountHealthGuard: &result}
		task.Config.AccountHealthGuardCursorAccountID = result.CursorAccountID
		if err != nil {
			return err
		}
		if run.FailedCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = fmt.Sprintf("健康守护发现 %d 个异常账号", run.FailedCount)
		} else {
			run.Status = SupplierAutomationStatusSuccess
			run.Message = fmt.Sprintf("健康守护检查 %d 个账号，待下轮 %d 个", result.CheckedCount, result.PendingCount)
		}
		return nil
	case SupplierAutomationTaskAccountRateGuard:
		if s.accountRateGuard == nil {
			return fmt.Errorf("supplier account rate guard service is required")
		}
		guardMode := SupplierAccountRateGuardModeExecute
		if mode == SupplierAutomationRunModePreview {
			guardMode = SupplierAccountRateGuardModePreview
		}
		result, err := s.accountRateGuard.Run(ctx, run.ID, guardMode, task.Config.AccountRateGuardDisabledGroupIDs, time.Now())
		run.ProcessedCount = result.CheckedAccounts
		run.FailedCount = result.Failed + result.RateSyncFailedProviders
		run.SuccessCount = run.ProcessedCount - result.Failed
		if run.SuccessCount < 0 {
			run.SuccessCount = 0
		}
		run.ResultDetail = &SupplierAutomationRunDetail{AccountRateGuard: &result}
		if err != nil {
			return err
		}
		if run.FailedCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = fmt.Sprintf("账号倍率守护存在 %d 项失败", run.FailedCount)
		} else if guardMode == SupplierAccountRateGuardModePreview {
			run.Message = fmt.Sprintf("检测完成，发现 %d 个风险分组", result.RiskGroups)
		} else {
			run.Message = fmt.Sprintf("执行完成，解除 %d 个分组绑定", result.UnboundGroups)
		}
		return nil
	case SupplierAutomationTaskGroupElection:
		if s.groupElection == nil {
			return fmt.Errorf("supplier group scheduling election service is required")
		}
		result, err := s.groupElection.Run(ctx, SupplierGroupSchedulingElectionConfig{
			TopN:                         task.Config.GroupElectionTopN,
			DisabledGroupIDs:             task.Config.GroupElectionDisabledGroupIDs,
			CountWeight:                  task.Config.GroupElectionCountWeight,
			LatencyWeight:                task.Config.GroupElectionLatencyWeight,
			FailureThreshold:             task.Config.GroupElectionFailureThreshold,
			SwitchMargin:                 task.Config.GroupElectionSwitchMargin,
			LatencyWindowMinutes:         task.Config.GroupElectionLatencyWindowMinutes,
			LatencyMinSamples:            task.Config.GroupElectionLatencyMinSamples,
			CountScoreCap:                task.Config.GroupElectionCountScoreCap,
			KeepHealthyIncumbentGroupIDs: task.Config.GroupElectionKeepHealthyIncumbentGroupIDs,
			RequiredModelsByGroup:        task.Config.GroupElectionRequiredModels,
			DryRun:                       task.Config.GroupElectionDryRun,
			DryRunGroupIDs:               task.Config.GroupElectionDryRunGroupIDs,
		}, time.Now())
		run.ProcessedCount = result.AccountCount
		run.SuccessCount = result.EnabledCount + result.DisabledCount + result.UnchangedCount
		run.FailedCount = result.FailedWriteCount
		run.ResultDetail = &SupplierAutomationRunDetail{GroupElection: &result}
		if err != nil {
			return err
		}
		if result.FailedWriteCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = fmt.Sprintf("分组择优调度存在 %d 个账号更新失败", result.FailedWriteCount)
		} else {
			run.Message = fmt.Sprintf("择优完成，%d 个分组，开启 %d 个、关闭 %d 个账号", result.GroupCount, result.EnabledCount, result.DisabledCount)
			// 被闸门拦住的账号不算失败，但也不是"一切正常"——
			// 尤其是「分组里只剩这一个坏账号」，必须提示人工确认，否则分组会一直靠一个坏账号撑着。
			if result.PendingCount > 0 {
				run.Message += fmt.Sprintf("，%d 个账号连续失败未达阈值待观察", result.PendingCount)
			}
			if result.KeptCount > 0 {
				run.Message += fmt.Sprintf("，%d 个账号因分组无备选账号保留调度需人工确认", result.KeptCount)
			}
			// 必需模型断供：本轮已切到能用的账号，但该模型当前无健康提供者，必须提示人工恢复。
			if result.RequiredModelUncoveredCount > 0 {
				run.Message += fmt.Sprintf("，%d 个必需模型当前无健康提供者待恢复", result.RequiredModelUncoveredCount)
			}
			// 演练必须写在摘要里：运行历史默认只显示这一行，不写明的话
			//「开启 0、关闭 0」会被读成任务空转，而实际是故意不生效。
			if result.DryRun {
				run.Message += fmt.Sprintf("（演练模式：建议开启 %d 个、建议关闭 %d 个，未实际修改调度）",
					result.SuggestedEnabledCount, result.SuggestedDisabledCount)
			}
		}
		return nil
	default:
		return ErrSupplierProviderInvalid
	}
}

func supplierAutomationRunDetailFromBatch(result SupplierProviderBatchSyncResult) *SupplierAutomationRunDetail {
	detail := &SupplierAutomationRunDetail{Providers: make([]SupplierAutomationProviderRunDetail, 0, len(result.Results))}
	for _, item := range result.Results {
		provider := SupplierAutomationProviderRunDetail{
			ProviderID:   item.ProviderID,
			ProviderName: item.ProviderName,
			Scope:        item.Scope,
			Status:       item.Status,
			Message:      item.Message,
			Counts:       item.Counts,
			StartedAt:    item.StartedAt,
			FinishedAt:   item.FinishedAt,
			Stages:       make([]SupplierAutomationStageRunDetail, 0, len(item.Stages)),
		}
		for _, stage := range item.Stages {
			stageDetail := SupplierAutomationStageRunDetail{
				Scope:   stage.Scope,
				Status:  stage.Status,
				Message: stage.Message,
				Counts:  stage.Counts,
			}
			if stage.EndpointResult != nil {
				stageDetail.Endpoint = stage.EndpointResult.Endpoint
				stageDetail.HTTPStatus = stage.EndpointResult.HTTPStatus
				stageDetail.DurationMS = stage.EndpointResult.DurationMS
				stageDetail.ResponseBytes = stage.EndpointResult.ResponseBytes
				stageDetail.ResponseSummary = stage.EndpointResult.ResponseSummary
				stageDetail.ParsedSummary = stage.EndpointResult.ParsedSummary
				stageDetail.ParseError = stage.EndpointResult.ParseError
				stageDetail.Error = stage.EndpointResult.Error
			}
			provider.Stages = append(provider.Stages, stageDetail)
		}
		detail.Providers = append(detail.Providers, provider)
	}
	if len(detail.Providers) == 0 {
		return nil
	}
	return detail
}

func supplierAutomationBatchFailureMessage(result SupplierProviderBatchSyncResult) string {
	const maxDetails = 5
	details := make([]string, 0, maxDetails)
	remaining := 0
	for _, item := range result.Results {
		if item.Status == SupplierSyncStatusSuccess || item.Status == SupplierSyncStatusSkipped {
			continue
		}
		itemDetails := supplierProviderSyncFailureDetails(item)
		if len(itemDetails) == 0 {
			itemDetails = []string{fmt.Sprintf("供应商 %d %s：%s", item.ProviderID, item.Scope, strings.TrimSpace(item.Message))}
		}
		for _, detail := range itemDetails {
			if strings.TrimSpace(detail) == "" {
				continue
			}
			if len(details) < maxDetails {
				details = append(details, detail)
			} else {
				remaining++
			}
		}
	}
	if len(details) == 0 {
		return "部分供应商同步失败"
	}
	message := "部分供应商同步失败：" + strings.Join(details, "；")
	if remaining > 0 {
		message += fmt.Sprintf("；等 %d 个失败", remaining)
	}
	return message
}

func supplierProviderSyncFailureDetails(item SupplierProviderSyncResult) []string {
	details := make([]string, 0, len(item.Stages))
	for _, stage := range item.Stages {
		if stage.Status == SupplierSyncStatusSuccess {
			continue
		}
		message := strings.TrimSpace(stage.Message)
		if message == "" {
			message = supplierSyncMessage(stage.Status)
		}
		details = append(details, fmt.Sprintf("供应商 %d %s：%s", item.ProviderID, stage.Scope, message))
	}
	return details
}

func validateSupplierAutomationTask(task SupplierAutomationTask) error {
	if strings.TrimSpace(task.TaskCode) == "" || strings.TrimSpace(task.CronExpression) == "" || task.TimeoutSeconds <= 0 {
		return ErrSupplierProviderInvalid
	}
	if _, err := supplierAutomationCronParser.Parse(strings.TrimSpace(task.CronExpression)); err != nil {
		return fmt.Errorf("invalid supplier automation cron: %w", err)
	}
	if task.TaskCode == SupplierAutomationTaskRateGuard {
		if task.Config.RateGuardMaxSnapshotAgeSeconds < 60 {
			return ErrSupplierProviderInvalid
		}
	}
	if task.TaskCode == SupplierAutomationTaskGroupElection {
		// TopN 允许为 0（归一化时回落默认 1），但负数视为配置错误。
		if task.Config.GroupElectionTopN < 0 || task.Config.GroupElectionTopN > MaxSupplierGroupSchedulingElectionTopN {
			return ErrSupplierProviderInvalid
		}
		for _, groupID := range task.Config.GroupElectionDisabledGroupIDs {
			if groupID <= 0 {
				return ErrSupplierProviderInvalid
			}
		}
		// 演练分组名单同样只挡非法 ID：空列表是合法的「不演练」，由归一化去重排序。
		for _, groupID := range task.Config.GroupElectionDryRunGroupIDs {
			if groupID <= 0 {
				return ErrSupplierProviderInvalid
			}
		}
		// 权重允许缺省或 0（归一化时回落默认），负数与超上限都视为配置错误——
		// 这里拒绝而不是静默截断，免得管理员以为自己配的 200 生效了。
		if task.Config.GroupElectionCountWeight < 0 ||
			task.Config.GroupElectionCountWeight > MaxSupplierGroupSchedulingElectionWeight ||
			task.Config.GroupElectionLatencyWeight < 0 ||
			task.Config.GroupElectionLatencyWeight > MaxSupplierGroupSchedulingElectionWeight {
			return ErrSupplierProviderInvalid
		}
		// 阈值 0 表示未配置（归一化时回落默认 2），负数与超上限视为配置错误。
		if task.Config.GroupElectionFailureThreshold < 0 ||
			task.Config.GroupElectionFailureThreshold > MaxSupplierGroupSchedulingElectionFailureThreshold {
			return ErrSupplierProviderInvalid
		}
		// 次数封顶值 0 表示未配置（归一化时回落默认 10），负数与超上限视为配置错误——
		// 与权重同样选择拒绝而不是静默截断，免得管理员以为自己配的 500 生效了。
		if task.Config.GroupElectionCountScoreCap < 0 ||
			task.Config.GroupElectionCountScoreCap > MaxSupplierGroupSchedulingElectionCountScoreCap {
			return ErrSupplierProviderInvalid
		}
		// 必需模型：分组 ID 必须为正；模型名清洗由归一化处理，这里只挡明显非法的 groupID。
		for groupID := range task.Config.GroupElectionRequiredModels {
			if groupID <= 0 {
				return ErrSupplierProviderInvalid
			}
		}
	}
	if task.TaskCode == SupplierAutomationTaskAccountHealthGuard {
		config := task.Config
		if config.AccountHealthGuardMaxAccountsPerRun <= 0 ||
			config.AccountHealthGuardMaxAccountsPerRun > MaxSupplierAccountHealthGuardMaxAccountsPerRun ||
			config.AccountHealthGuardConcurrency <= 0 ||
			config.AccountHealthGuardConcurrency > MaxSupplierAccountHealthGuardConcurrency ||
			config.AccountHealthGuardTimeoutPerAccountSeconds < MinSupplierAccountHealthGuardTimeoutPerAccountSeconds ||
			config.AccountHealthGuardTimeoutPerAccountSeconds > MaxSupplierAccountHealthGuardTimeoutPerAccountSeconds ||
			config.AccountHealthGuardFailureThreshold <= 0 ||
			config.AccountHealthGuardSlowThreshold <= 0 ||
			config.AccountHealthGuardRecoveryThreshold <= 0 ||
			config.AccountHealthGuardHealthyLatencyMs <= 0 {
			return ErrSupplierProviderInvalid
		}
		for _, interval := range config.AccountHealthGuardAccountIntervals {
			if interval > 0 && interval < MinSupplierAccountHealthGuardAccountIntervalSeconds {
				return ErrSupplierProviderInvalid
			}
		}
		// 平台倍率区间规则非法时直接拒绝，避免归一化静默丢弃后管理员误以为已生效。
		for _, rules := range config.AccountHealthGuardPlatformMultiplierIntervals {
			for _, rule := range rules {
				if rule.IntervalSeconds < MinSupplierAccountHealthGuardAccountIntervalSeconds {
					return ErrSupplierProviderInvalid
				}
				if rule.MinMultiplier < 0 {
					return ErrSupplierProviderInvalid
				}
				if rule.MaxMultiplier > 0 && rule.MinMultiplier >= rule.MaxMultiplier {
					return ErrSupplierProviderInvalid
				}
			}
		}
		// 账号级阈值非正数在归一化时会被丢弃并回落全局，这里直接拒绝，避免管理员以为已生效。
		accountThresholds := []map[int64]int{
			config.AccountHealthGuardAccountFailureThresholds,
			config.AccountHealthGuardAccountSlowThresholds,
			config.AccountHealthGuardAccountRecoveryThresholds,
		}
		for _, thresholds := range accountThresholds {
			for accountID, threshold := range thresholds {
				if accountID <= 0 || threshold <= 0 {
					return ErrSupplierProviderInvalid
				}
			}
		}
	}
	return nil
}

func validateSupplierAccountHealthGuardSelection(task SupplierAutomationTask) error {
	if task.TaskCode != SupplierAutomationTaskAccountHealthGuard {
		return nil
	}
	if len(normalizeSupplierAccountHealthGuardAccountIDs(task.Config.AccountHealthGuardAccountIDs)) == 0 {
		return errors.New("请至少选择一个需要检查的账号")
	}
	return nil
}

type SupplierAutomationScheduler struct {
	repo    SupplierAutomationRepository
	service *SupplierAutomationService

	mu      sync.Mutex
	cron    *cron.Cron
	started bool
	stopped bool
}

func NewSupplierAutomationScheduler(repo SupplierAutomationRepository, service *SupplierAutomationService) *SupplierAutomationScheduler {
	return &SupplierAutomationScheduler{repo: repo, service: service}
}

func (s *SupplierAutomationScheduler) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.started || s.stopped {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()
	ctx := context.Background()
	if err := s.repo.RecoverRunning(ctx, "服务重启后恢复任务状态"); err != nil {
		slog.Error("supplier_automation.recover_running_failed", "error", err)
	}
	// 进程异常退出时 Redis 锁可能残留；启动时强制清理，避免定时触发长期静默冲突。
	if s.service != nil && s.service.lock != nil {
		for _, taskCode := range []string{
			SupplierAutomationTaskSync,
			SupplierAutomationTaskMonitorSync,
			SupplierAutomationTaskCleanup,
			SupplierAutomationTaskRateGuard,
			SupplierAutomationTaskAccountRateGuard,
			SupplierAutomationTaskAccountHealthGuard,
			SupplierAutomationTaskRechargeSync,
			SupplierAutomationTaskGroupElection,
			supplierAutomationUpstreamFetchLock,
		} {
			if err := s.service.lock.ForceReleaseAutomationLock(ctx, taskCode); err != nil {
				slog.Warn("supplier_automation.force_release_lock_failed", "task_code", taskCode, "error", err)
			}
		}
	}
	if err := s.Reload(ctx); err != nil {
		slog.Error("supplier_automation.scheduler_start_reload_failed", "error", err)
	}
}

func (s *SupplierAutomationScheduler) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped = true
	s.stopLocked()
}

func (s *SupplierAutomationScheduler) Reload(ctx context.Context) error {
	if s == nil {
		return nil
	}
	tasks, err := s.repo.ListTasks(ctx)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	nextCron := cron.New(cron.WithParser(supplierAutomationCronParser), cron.WithLocation(loc))
	now := time.Now().In(loc)
	for _, task := range tasks {
		if err := validateSupplierAutomationTask(task); err != nil {
			// 单个任务配置异常不应拖垮全部自动化调度。
			slog.Error("supplier_automation.task_invalid", "task_code", task.TaskCode, "error", err)
			continue
		}
		if task.Enabled {
			if err := validateSupplierAccountHealthGuardSelection(task); err != nil {
				slog.Error("supplier_automation.task_selection_invalid", "task_code", task.TaskCode, "error", err)
				continue
			}
		}
		schedule, err := supplierAutomationCronParser.Parse(task.CronExpression)
		if err != nil {
			slog.Error("supplier_automation.task_cron_invalid", "task_code", task.TaskCode, "error", err)
			continue
		}
		next := schedule.Next(now)
		task.NextRunAt = &next
		if err := s.repo.UpdateTaskRuntime(ctx, &task); err != nil {
			return err
		}
		if task.Enabled {
			taskCode := task.TaskCode
			if _, err := nextCron.AddFunc(task.CronExpression, func() {
				if _, err := s.service.Run(context.Background(), taskCode, SupplierSyncTriggerScheduled); err != nil {
					slog.Warn("supplier_automation.scheduled_run_failed", "task_code", taskCode, "error", err)
				}
			}); err != nil {
				slog.Error("supplier_automation.add_cron_failed", "task_code", task.TaskCode, "error", err)
				continue
			}
		}
	}
	nextCron.Start()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		nextCron.Stop()
		return nil
	}
	s.stopLocked()
	s.cron = nextCron
	return nil
}

func (s *SupplierAutomationScheduler) stopLocked() {
	if s.cron == nil {
		return
	}
	ctx := s.cron.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
	}
	s.cron = nil
}
