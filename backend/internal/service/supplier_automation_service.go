package service

import (
	"context"
	"encoding/json"
	"fmt"
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
	AutomationRunRetentionDays     int     `json:"automation_run_retention_days"`
	SyncRunRetentionDays           int     `json:"sync_run_retention_days"`
	MetricRetentionDays            int     `json:"metric_snapshot_retention_days"`
	DailyStatRetentionDays         int     `json:"daily_stat_retention_days"`
	InactiveAccountDays            int     `json:"inactive_account_retention_days"`
	InactiveGroupDays              int     `json:"inactive_group_retention_days"`
	RateGuardSafetyMultiplier      float64 `json:"rate_guard_safety_multiplier"`
	RateGuardMaxSnapshotAgeSeconds int     `json:"rate_guard_max_snapshot_age_seconds"`

	AccountHealthGuardMaxAccountsPerRun        int               `json:"account_health_guard_max_accounts_per_run"`
	AccountHealthGuardConcurrency              int               `json:"account_health_guard_concurrency"`
	AccountHealthGuardTimeoutPerAccountSeconds int               `json:"account_health_guard_timeout_per_account_seconds"`
	AccountHealthGuardFailureThreshold         int               `json:"account_health_guard_failure_threshold"`
	AccountHealthGuardSlowThreshold            int               `json:"account_health_guard_slow_threshold"`
	AccountHealthGuardRecoveryThreshold        int               `json:"account_health_guard_recovery_threshold"`
	AccountHealthGuardHealthyLatencyMs         int64             `json:"account_health_guard_healthy_latency_ms"`
	AccountHealthGuardIgnoredAccountIDs        []int64           `json:"account_health_guard_ignored_account_ids"`
	AccountHealthGuardAccountModels            map[int64]string  `json:"account_health_guard_account_models"`
	AccountHealthGuardPlatformModels           map[string]string `json:"account_health_guard_platform_models"`
	AccountHealthGuardPlatformLatencyMs        map[string]int64  `json:"account_health_guard_platform_latency_ms"`
	AccountHealthGuardCursorAccountID          int64             `json:"account_health_guard_cursor_account_id"`
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
	Providers          []SupplierAutomationProviderRunDetail `json:"providers,omitempty"`
	Cleanup            *SupplierAutomationCleanupRunDetail   `json:"cleanup,omitempty"`
	RateGuard          *SupplierRateGuardResult              `json:"rate_guard,omitempty"`
	AccountRateGuard   *SupplierAccountRateGuardResult       `json:"account_rate_guard,omitempty"`
	AccountHealthGuard *SupplierAccountHealthGuardResult     `json:"account_health_guard,omitempty"`
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
	AutomationRuns  int `json:"automation_runs"`
	SyncRuns        int `json:"sync_runs"`
	MetricSnapshots int `json:"metric_snapshots"`
	DailyStats      int `json:"daily_stats"`
	Accounts        int `json:"accounts"`
	Groups          int `json:"groups"`
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
	UpdateTask(ctx context.Context, task *SupplierAutomationTask) error
	CreateRun(ctx context.Context, run *SupplierAutomationRun) error
	FinishRun(ctx context.Context, run *SupplierAutomationRun) error
	ListRuns(ctx context.Context, params SupplierAutomationRunListParams) (SupplierAutomationRunListResult, error)
	RecoverRunning(ctx context.Context, message string) error
}

type SupplierAutomationLock interface {
	TryAcquireAutomationLock(ctx context.Context, taskCode, owner string, ttl time.Duration) (bool, error)
	ReleaseAutomationLock(ctx context.Context, taskCode, owner string) error
}

type SupplierAutomationSchedulerReloader interface {
	Reload(ctx context.Context) error
}

type SupplierProviderBatchSyncer interface {
	SyncAllEnabled(ctx context.Context, trigger string) (SupplierProviderBatchSyncResult, error)
}

type SupplierRateGuardRunner interface {
	Run(ctx context.Context, config SupplierRateGuardConfig, now time.Time) (SupplierRateGuardResult, error)
}

type SupplierAccountRateGuardRunner interface {
	Run(ctx context.Context, runID int64, mode SupplierAccountRateGuardMode, now time.Time) (SupplierAccountRateGuardResult, error)
}

const (
	SupplierAutomationRunModePreview = "preview"
	SupplierAutomationRunModeExecute = "execute"

	SupplierAutomationTaskSync               = "supplier_data_sync"
	SupplierAutomationTaskCleanup            = "supplier_data_cleanup"
	SupplierAutomationTaskRateGuard          = "supplier_rate_guard"
	SupplierAutomationTaskAccountRateGuard   = "supplier_account_rate_guard"
	SupplierAutomationTaskAccountHealthGuard = "supplier_account_health_guard"

	SupplierAutomationStatusRunning = "running"
	SupplierAutomationStatusSuccess = "success"
	SupplierAutomationStatusPartial = "partial"
	SupplierAutomationStatusFailed  = "failed"
)

func supplierAutomationConfigJSON(config SupplierAutomationConfig) string {
	raw, _ := json.Marshal(config)
	return string(raw)
}

type SupplierAutomationService struct {
	repo               SupplierAutomationRepository
	lock               SupplierAutomationLock
	syncer             SupplierProviderBatchSyncer
	dataRepo           SupplierProviderDataRepository
	rateGuard          SupplierRateGuardRunner
	accountRateGuard   SupplierAccountRateGuardRunner
	accountHealthGuard SupplierAccountHealthGuardRunner
	accountRateLogs    SupplierAccountRateGuardRepository
	reloader           SupplierAutomationSchedulerReloader
}

func NewSupplierAutomationService(repo SupplierAutomationRepository, lock SupplierAutomationLock, syncer SupplierProviderBatchSyncer, dataRepo SupplierProviderDataRepository) *SupplierAutomationService {
	return &SupplierAutomationService{repo: repo, lock: lock, syncer: syncer, dataRepo: dataRepo}
}

func (s *SupplierAutomationService) SetSchedulerReloader(reloader SupplierAutomationSchedulerReloader) {
	s.reloader = reloader
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
	if mode == SupplierAutomationRunModePreview && task.TaskCode != SupplierAutomationTaskAccountRateGuard {
		return SupplierAutomationRun{}, ErrSupplierProviderInvalid
	}
	owner := uuid.NewString()
	if s.lock != nil {
		acquired, err := s.lock.TryAcquireAutomationLock(ctx, task.TaskCode, owner, time.Duration(task.TimeoutSeconds+60)*time.Second)
		if err != nil {
			return SupplierAutomationRun{}, err
		}
		if !acquired {
			return SupplierAutomationRun{}, ErrSupplierProviderSyncConflict
		}
		defer func() { _ = s.lock.ReleaseAutomationLock(context.Background(), task.TaskCode, owner) }()
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
	_ = s.repo.UpdateTask(ctx, task)
	return run, execErr
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
	case SupplierAutomationTaskCleanup:
		counts, err := s.dataRepo.Cleanup(ctx, SupplierCleanupPolicy{
			AutomationRunRetentionDays: task.Config.AutomationRunRetentionDays,
			SyncRunRetentionDays:       task.Config.SyncRunRetentionDays,
			MetricRetentionDays:        task.Config.MetricRetentionDays,
			DailyStatRetentionDays:     task.Config.DailyStatRetentionDays,
			InactiveAccountDays:        task.Config.InactiveAccountDays,
			InactiveGroupDays:          task.Config.InactiveGroupDays,
		}, time.Now(), 1000)
		if err != nil {
			return err
		}
		run.ProcessedCount = counts.AutomationRuns + counts.SyncRuns + counts.MetricSnapshots + counts.DailyStats + counts.Accounts + counts.Groups
		run.ResultDetail = &SupplierAutomationRunDetail{Cleanup: &SupplierAutomationCleanupRunDetail{
			AutomationRuns:  counts.AutomationRuns,
			SyncRuns:        counts.SyncRuns,
			MetricSnapshots: counts.MetricSnapshots,
			DailyStats:      counts.DailyStats,
			Accounts:        counts.Accounts,
			Groups:          counts.Groups,
		}}
		return nil
	case SupplierAutomationTaskRateGuard:
		if s.rateGuard == nil {
			return fmt.Errorf("supplier rate guard service is required")
		}
		result, err := s.rateGuard.Run(ctx, SupplierRateGuardConfig{
			SafetyMultiplier: task.Config.RateGuardSafetyMultiplier,
			MaxSnapshotAge:   time.Duration(task.Config.RateGuardMaxSnapshotAgeSeconds) * time.Second,
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
			MaxAccountsPerRun:        task.Config.AccountHealthGuardMaxAccountsPerRun,
			Concurrency:              task.Config.AccountHealthGuardConcurrency,
			TimeoutPerAccountSeconds: task.Config.AccountHealthGuardTimeoutPerAccountSeconds,
			FailureThreshold:         task.Config.AccountHealthGuardFailureThreshold,
			SlowThreshold:            task.Config.AccountHealthGuardSlowThreshold,
			RecoveryThreshold:        task.Config.AccountHealthGuardRecoveryThreshold,
			HealthyLatencyMs:         task.Config.AccountHealthGuardHealthyLatencyMs,
			IgnoredAccountIDs:        task.Config.AccountHealthGuardIgnoredAccountIDs,
			AccountModels:            task.Config.AccountHealthGuardAccountModels,
			PlatformModels:           task.Config.AccountHealthGuardPlatformModels,
			PlatformLatencyMs:        task.Config.AccountHealthGuardPlatformLatencyMs,
			CursorAccountID:          task.Config.AccountHealthGuardCursorAccountID,
		}, time.Now())
		run.ProcessedCount = result.CheckedCount
		run.SuccessCount = result.HealthyCount + result.SlowCount
		run.FailedCount = result.FailedCount
		run.ResultDetail = &SupplierAutomationRunDetail{AccountHealthGuard: &result}
		task.Config.AccountHealthGuardCursorAccountID = result.CursorAccountID
		if err != nil {
			return err
		}
		if result.FailedCount > 0 {
			run.Status = SupplierAutomationStatusPartial
			run.Message = fmt.Sprintf("健康守护发现 %d 个失败账号", result.FailedCount)
		} else {
			run.Status = SupplierAutomationStatusSuccess
			run.Message = fmt.Sprintf("健康守护检查 %d 个账号，跳过 %d 个", result.CheckedCount, result.SkippedCount)
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
		result, err := s.accountRateGuard.Run(ctx, run.ID, guardMode, time.Now())
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
		if item.Status == SupplierSyncStatusSuccess {
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
		if task.Config.RateGuardSafetyMultiplier <= 0 || task.Config.RateGuardMaxSnapshotAgeSeconds < 60 {
			return ErrSupplierProviderInvalid
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
	_ = s.repo.RecoverRunning(context.Background(), "服务重启后恢复任务状态")
	_ = s.Reload(context.Background())
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
			return err
		}
		schedule, err := supplierAutomationCronParser.Parse(task.CronExpression)
		if err != nil {
			return err
		}
		next := schedule.Next(now)
		task.NextRunAt = &next
		if err := s.repo.UpdateTask(ctx, &task); err != nil {
			return err
		}
		if task.Enabled {
			taskCode := task.TaskCode
			if _, err := nextCron.AddFunc(task.CronExpression, func() {
				_, _ = s.service.Run(context.Background(), taskCode, SupplierSyncTriggerScheduled)
			}); err != nil {
				return err
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
