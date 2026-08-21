package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type supplierAutomationRepoStub struct {
	tasks        map[string]*SupplierAutomationTask
	runs         []SupplierAutomationRun
	updatedTasks []SupplierAutomationTask
}

func (r *supplierAutomationRepoStub) ListTasks(context.Context) ([]SupplierAutomationTask, error) {
	out := make([]SupplierAutomationTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		out = append(out, *task)
	}
	return out, nil
}
func (r *supplierAutomationRepoStub) GetTask(_ context.Context, code string) (*SupplierAutomationTask, error) {
	if task, ok := r.tasks[code]; ok {
		clone := *task
		return &clone, nil
	}
	return nil, ErrSupplierProviderInvalid
}
func (r *supplierAutomationRepoStub) UpdateTask(_ context.Context, task *SupplierAutomationTask) error {
	// 模拟仓储只更新配置字段，保留已有运行时状态。
	existing, ok := r.tasks[task.TaskCode]
	if !ok || existing == nil {
		clone := *task
		r.tasks[task.TaskCode] = &clone
		r.updatedTasks = append(r.updatedTasks, clone)
		return nil
	}
	existing.Enabled = task.Enabled
	existing.CronExpression = task.CronExpression
	existing.TimeoutSeconds = task.TimeoutSeconds
	existing.Config = task.Config
	if task.Name != "" {
		existing.Name = task.Name
	}
	clone := *existing
	r.updatedTasks = append(r.updatedTasks, clone)
	return nil
}

func (r *supplierAutomationRepoStub) UpdateTaskRuntime(_ context.Context, task *SupplierAutomationTask) error {
	// 模拟仓储只更新运行时状态，绝不覆盖 cron/config 等配置项。
	existing, ok := r.tasks[task.TaskCode]
	if !ok || existing == nil {
		clone := *task
		r.tasks[task.TaskCode] = &clone
		r.updatedTasks = append(r.updatedTasks, clone)
		return nil
	}
	existing.LastStatus = task.LastStatus
	existing.LastMessage = task.LastMessage
	existing.LastRunAt = task.LastRunAt
	existing.NextRunAt = task.NextRunAt
	clone := *existing
	r.updatedTasks = append(r.updatedTasks, clone)
	return nil
}
func (r *supplierAutomationRepoStub) CreateRun(_ context.Context, run *SupplierAutomationRun) error {
	run.ID = int64(len(r.runs) + 1)
	r.runs = append(r.runs, *run)
	return nil
}
func (r *supplierAutomationRepoStub) FinishRun(_ context.Context, run *SupplierAutomationRun) error {
	r.runs = append(r.runs, *run)
	return nil
}
func (r *supplierAutomationRepoStub) ListRuns(context.Context, SupplierAutomationRunListParams) (SupplierAutomationRunListResult, error) {
	return SupplierAutomationRunListResult{Items: r.runs, Total: int64(len(r.runs))}, nil
}
func (r *supplierAutomationRepoStub) RecoverRunning(context.Context, string) error { return nil }

type supplierAutomationLockStub struct {
	acquired       bool
	released       int
	acquireCalls   []string
	acquireTTLs    []time.Duration
	acquireResults map[string][]bool
}

func (l *supplierAutomationLockStub) TryAcquireAutomationLock(_ context.Context, taskCode string, _ string, ttl time.Duration) (bool, error) {
	l.acquireCalls = append(l.acquireCalls, taskCode)
	l.acquireTTLs = append(l.acquireTTLs, ttl)
	if results, ok := l.acquireResults[taskCode]; ok && len(results) > 0 {
		acquired := results[0]
		l.acquireResults[taskCode] = results[1:]
		return acquired, nil
	}
	return l.acquired, nil
}
func (l *supplierAutomationLockStub) ReleaseAutomationLock(context.Context, string, string) error {
	l.released++
	return nil
}
func (l *supplierAutomationLockStub) ForceReleaseAutomationLock(context.Context, string) error {
	l.released++
	return nil
}

type supplierAutomationSyncStub struct {
	called int
	err    error
	result SupplierProviderBatchSyncResult
}

type supplierAutomationMonitorSyncStub struct {
	called int
	err    error
	result SupplierProviderMonitorSyncResult
}

type supplierAutomationRechargeSyncStub struct {
	called   int
	fullSync []bool
	err      error
	result   SupplierProviderRechargeSyncAllResult
}

func (s *supplierAutomationRechargeSyncStub) SyncAll(_ context.Context, fullSync bool) (SupplierProviderRechargeSyncAllResult, error) {
	s.called++
	s.fullSync = append(s.fullSync, fullSync)
	return s.result, s.err
}

type supplierAutomationRateGuardStub struct {
	called int
	config SupplierRateGuardConfig
	result SupplierRateGuardResult
	err    error
}

type supplierAutomationAccountRateGuardStub struct {
	called int
	runID  int64
	mode   SupplierAccountRateGuardMode
	result SupplierAccountRateGuardResult
	err    error
}

func (s *supplierAutomationAccountRateGuardStub) Run(_ context.Context, runID int64, mode SupplierAccountRateGuardMode, _ time.Time) (SupplierAccountRateGuardResult, error) {
	s.called++
	s.runID = runID
	s.mode = mode
	return s.result, s.err
}

type supplierRateGuardChangeLogDataRepoStub struct {
	*supplierProviderDataRepoStub
	listParams SupplierRateGuardChangeLogListParams
	handledID  int64
	listResult SupplierRateGuardChangeLogListResult
	handled    SupplierRateGuardChangeLog
}

func (r *supplierRateGuardChangeLogDataRepoStub) ListRateGuardChangeLogs(_ context.Context, params SupplierRateGuardChangeLogListParams) (SupplierRateGuardChangeLogListResult, error) {
	r.listParams = params
	return r.listResult, nil
}

func (r *supplierRateGuardChangeLogDataRepoStub) MarkRateGuardChangeLogHandled(_ context.Context, id int64) (SupplierRateGuardChangeLog, error) {
	r.handledID = id
	return r.handled, nil
}

func (s *supplierAutomationRateGuardStub) Run(_ context.Context, config SupplierRateGuardConfig, _ time.Time) (SupplierRateGuardResult, error) {
	s.called++
	s.config = config
	return s.result, s.err
}

func (s *supplierAutomationSyncStub) SyncAllEnabled(context.Context, string) (SupplierProviderBatchSyncResult, error) {
	s.called++
	if s.err != nil {
		return SupplierProviderBatchSyncResult{}, s.err
	}
	if s.result.ProcessedCount > 0 || len(s.result.Results) > 0 {
		return s.result, nil
	}
	return SupplierProviderBatchSyncResult{ProcessedCount: 2, SuccessCount: 1, FailedCount: 1}, nil
}

func (s *supplierAutomationMonitorSyncStub) SyncMonitorsEnabled(context.Context, string) (SupplierProviderMonitorSyncResult, error) {
	s.called++
	return s.result, s.err
}

func TestSupplierAutomationCronParserSupportsLegacySecondsAndEvery(t *testing.T) {
	cases := []string{
		"*/5 * * * *",
		"*/5 * * * * *",
		"@every 1s",
		"@every 90s",
		"@every 300s",
	}
	for _, expression := range cases {
		_, err := supplierAutomationCronParser.Parse(expression)
		require.NoError(t, err, expression)
	}
}

func TestSupplierAutomationCronParserRejectsInvalidExpressions(t *testing.T) {
	for _, expression := range []string{"", "@every 0s", "@every -1s", "not-a-cron"} {
		_, err := supplierAutomationCronParser.Parse(expression)
		require.Error(t, err, expression)
	}
}

func TestSupplierAutomationServiceListsAndHandlesRateGuardChangeLogs(t *testing.T) {
	dataRepo := &supplierRateGuardChangeLogDataRepoStub{
		supplierProviderDataRepoStub: &supplierProviderDataRepoStub{},
		listResult: SupplierRateGuardChangeLogListResult{
			Items:        []SupplierRateGuardChangeLog{{ID: 8, Status: SupplierRateGuardChangeLogStatusPending}},
			Total:        1,
			PendingCount: 1,
			Page:         2,
			PageSize:     20,
		},
		handled: SupplierRateGuardChangeLog{ID: 8, Status: SupplierRateGuardChangeLogStatusHandled},
	}
	service := NewSupplierAutomationService(&supplierAutomationRepoStub{}, &supplierAutomationLockStub{}, &supplierAutomationSyncStub{}, dataRepo)

	result, err := service.ListRateGuardChangeLogs(context.Background(), SupplierRateGuardChangeLogListParams{Page: 2, PageSize: 20})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Equal(t, 2, dataRepo.listParams.Page)
	require.Equal(t, 20, dataRepo.listParams.PageSize)

	item, err := service.MarkRateGuardChangeLogHandled(context.Background(), 8)

	require.NoError(t, err)
	require.Equal(t, SupplierRateGuardChangeLogStatusHandled, item.Status)
	require.Equal(t, int64(8), dataRepo.handledID)
}

func TestSupplierAutomationServiceRequiresRateGuardChangeLogStore(t *testing.T) {
	service := NewSupplierAutomationService(&supplierAutomationRepoStub{}, &supplierAutomationLockStub{}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})

	_, err := service.ListRateGuardChangeLogs(context.Background(), SupplierRateGuardChangeLogListParams{})

	require.EqualError(t, err, "supplier rate guard change log store is required")
}

func TestSupplierAutomationServiceRunsRateGuardWithStructuredPartialResult(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskRateGuard: {
			TaskCode: SupplierAutomationTaskRateGuard, Name: "倍率守护", Enabled: true,
			CronExpression: "2-59/5 * * * *", TimeoutSeconds: 300,
			Config: SupplierAutomationConfig{RateGuardMaxSnapshotAgeSeconds: 1800},
		},
	}}
	rateGuard := &supplierAutomationRateGuardStub{result: SupplierRateGuardResult{
		Checked: 3, Raised: 1, Unchanged: 1, Failed: 1,
		Items: []SupplierRateGuardItemResult{{MappingID: 10, Action: SupplierRateGuardActionRaised}},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetRateGuardService(rateGuard)

	run, err := service.Run(context.Background(), SupplierAutomationTaskRateGuard, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, rateGuard.called)
	require.Equal(t, 30*time.Minute, rateGuard.config.MaxSnapshotAge)
	require.Equal(t, 3, run.ProcessedCount)
	require.Equal(t, 2, run.SuccessCount)
	require.Equal(t, 1, run.FailedCount)
	require.Equal(t, SupplierAutomationStatusPartial, run.Status)
	require.NotNil(t, run.ResultDetail)
	require.NotNil(t, run.ResultDetail.RateGuard)
	require.Equal(t, 1, run.ResultDetail.RateGuard.Raised)
}

func TestSupplierAutomationServiceRunsSupplierAccountRateGuardInPreviewMode(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountRateGuard: {
			TaskCode: SupplierAutomationTaskAccountRateGuard, Name: "供应商账号倍率守护", Enabled: true,
			CronExpression: "@every 300s", TimeoutSeconds: 600,
		},
	}}
	runner := &supplierAutomationAccountRateGuardStub{result: SupplierAccountRateGuardResult{
		CheckedAccounts: 3, RiskGroups: 2, Skipped: 1,
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountRateGuardService(runner)

	run, err := service.RunWithMode(context.Background(), SupplierAutomationTaskAccountRateGuard, SupplierSyncTriggerManual, SupplierAutomationRunModePreview)

	require.NoError(t, err)
	require.Equal(t, 1, runner.called)
	require.Equal(t, run.ID, runner.runID)
	require.Equal(t, SupplierAccountRateGuardModePreview, runner.mode)
	require.Equal(t, 3, run.ProcessedCount)
	require.Equal(t, 3, run.SuccessCount)
	require.NotNil(t, run.ResultDetail)
	require.Equal(t, 2, run.ResultDetail.AccountRateGuard.RiskGroups)
}

func TestSupplierAutomationServiceWaitsForUpstreamFetchLockBeforeRunningAccountRateGuard(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountRateGuard: {
			TaskCode: SupplierAutomationTaskAccountRateGuard, Name: "供应商账号倍率守护", Enabled: true,
			CronExpression: "@every 300s", TimeoutSeconds: 600,
		},
	}}
	lock := &supplierAutomationLockStub{
		acquired: true,
		acquireResults: map[string][]bool{
			SupplierAutomationTaskAccountRateGuard: {true},
			"supplier_upstream_fetch":              {false, true},
		},
	}
	runner := &supplierAutomationAccountRateGuardStub{}
	service := NewSupplierAutomationService(repo, lock, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountRateGuardService(runner)

	_, err := service.Run(context.Background(), SupplierAutomationTaskAccountRateGuard, SupplierSyncTriggerScheduled)

	require.NoError(t, err)
	require.Equal(t, 1, runner.called)
	require.Equal(t, []string{
		SupplierAutomationTaskAccountRateGuard,
		"supplier_upstream_fetch",
		"supplier_upstream_fetch",
	}, lock.acquireCalls)
	require.Equal(t, 21*time.Minute, lock.acquireTTLs[0])
	require.Equal(t, 11*time.Minute, lock.acquireTTLs[1])
	require.Equal(t, 2, lock.released)
}
func TestSupplierAutomationServiceUsesUpstreamFetchLockForDataSync(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {
			TaskCode: SupplierAutomationTaskSync, Name: "供应商数据同步", Enabled: true,
			CronExpression: "*/15 * * * *", TimeoutSeconds: 600,
		},
	}}
	lock := &supplierAutomationLockStub{
		acquired: true,
		acquireResults: map[string][]bool{
			SupplierAutomationTaskSync: {true},
			"supplier_upstream_fetch":  {true},
		},
	}
	syncer := &supplierAutomationSyncStub{result: SupplierProviderBatchSyncResult{ProcessedCount: 1, SuccessCount: 1}}
	service := NewSupplierAutomationService(repo, lock, syncer, &supplierProviderDataRepoStub{})

	_, err := service.Run(context.Background(), SupplierAutomationTaskSync, SupplierSyncTriggerScheduled)

	require.NoError(t, err)
	require.Equal(t, 1, syncer.called)
	require.Equal(t, []string{SupplierAutomationTaskSync, "supplier_upstream_fetch"}, lock.acquireCalls)
	require.Equal(t, 2, lock.released)
}

func TestSupplierAutomationServiceRunsSupplierMonitorSyncTask(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskMonitorSync: {
			TaskCode: SupplierAutomationTaskMonitorSync, Name: "供应商监控数据同步", Enabled: true,
			CronExpression: "@every 30s", TimeoutSeconds: 600,
		},
	}}
	lock := &supplierAutomationLockStub{
		acquired: true,
		acquireResults: map[string][]bool{
			SupplierAutomationTaskMonitorSync: {true},
			"supplier_upstream_fetch":         {true},
		},
	}
	monitorSync := &supplierAutomationMonitorSyncStub{result: SupplierProviderMonitorSyncResult{
		ProcessedCount: 2,
		SuccessCount:   2,
		Items: []SupplierProviderMonitorSyncItem{{
			ProviderID:     12,
			ProviderName:   "供应商 A",
			LocalAccountID: 98,
			UpstreamName:   "grok对话",
			Status:         SupplierAccountHealthGuardStatusSlow,
			LatencyMS:      10122,
		}},
	}}
	service := NewSupplierAutomationService(repo, lock, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetMonitorSyncService(monitorSync)

	run, err := service.Run(context.Background(), SupplierAutomationTaskMonitorSync, SupplierSyncTriggerScheduled)

	require.NoError(t, err)
	require.Equal(t, 1, monitorSync.called)
	require.Equal(t, SupplierAutomationStatusSuccess, run.Status)
	require.Equal(t, 2, run.ProcessedCount)
	require.Equal(t, 2, run.SuccessCount)
	require.NotNil(t, run.ResultDetail)
	require.NotNil(t, run.ResultDetail.SupplierMonitor)
	require.Len(t, run.ResultDetail.SupplierMonitor.Items, 1)
	require.Equal(t, int64(98), run.ResultDetail.SupplierMonitor.Items[0].LocalAccountID)
	require.Equal(t, []string{SupplierAutomationTaskMonitorSync, "supplier_upstream_fetch"}, lock.acquireCalls)
	require.Equal(t, 2, lock.released)
}

func TestSupplierAutomationServiceRunsSupplierRechargeSyncTaskIncrementally(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskRechargeSync: {
			TaskCode: SupplierAutomationTaskRechargeSync, Name: "供应商充值记录同步", Enabled: true,
			CronExpression: "@every 30m", TimeoutSeconds: 600,
		},
	}}
	rechargeSync := &supplierAutomationRechargeSyncStub{result: SupplierProviderRechargeSyncAllResult{
		Items: []SupplierProviderRechargeSyncResult{
			{ProviderID: 1, ProviderName: "供应商 A", Status: SupplierSyncStatusSuccess, RecordCount: 3},
			{ProviderID: 2, ProviderName: "供应商 B", Status: SupplierSyncStatusFailed, Message: "同步失败"},
		},
		SuccessCount: 1,
		FailedCount:  1,
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetRechargeSyncService(rechargeSync)

	run, err := service.Run(context.Background(), SupplierAutomationTaskRechargeSync, SupplierSyncTriggerScheduled)

	require.NoError(t, err)
	require.Equal(t, 1, rechargeSync.called)
	require.Equal(t, []bool{false}, rechargeSync.fullSync)
	require.Equal(t, SupplierAutomationStatusPartial, run.Status)
	require.Equal(t, 2, run.ProcessedCount)
	require.Equal(t, 1, run.SuccessCount)
	require.Equal(t, 1, run.FailedCount)
	require.NotNil(t, run.ResultDetail)
	require.NotNil(t, run.ResultDetail.RechargeSync)
	require.Equal(t, 2, len(run.ResultDetail.RechargeSync.Items))
	require.Equal(t, int64(2), run.ResultDetail.RechargeSync.Items[1].ProviderID)
}

func TestSupplierAutomationServiceRequiresRechargeSyncService(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskRechargeSync: {
			TaskCode: SupplierAutomationTaskRechargeSync, Name: "供应商充值记录同步", Enabled: true,
			CronExpression: "@every 30m", TimeoutSeconds: 600,
		},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})

	run, err := service.Run(context.Background(), SupplierAutomationTaskRechargeSync, SupplierSyncTriggerManual)

	require.EqualError(t, err, "supplier recharge sync service is required")
	require.Equal(t, SupplierAutomationStatusFailed, run.Status)
}

func TestSupplierAutomationServiceRejectsPreviewModeForOtherTasks(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {
			TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true,
			CronExpression: "*/15 * * * *", TimeoutSeconds: 600,
		},
	}}
	syncer := &supplierAutomationSyncStub{}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, syncer, &supplierProviderDataRepoStub{})

	_, err := service.RunWithMode(context.Background(), SupplierAutomationTaskSync, SupplierSyncTriggerManual, SupplierAutomationRunModePreview)

	require.Error(t, err)
	require.Zero(t, syncer.called)
}

func TestSupplierAutomationServiceListsAccountRateGuardUnbindLogs(t *testing.T) {
	logRepo := &supplierAccountRateGuardRepoStub{listResult: SupplierAccountRateGuardUnbindLogListResult{
		Items: []SupplierAccountRateGuardUnbindLog{{ID: 21, Result: SupplierAccountRateGuardLogResultFailed}},
		Total: 1, Page: 2, PageSize: 30,
	}}
	service := NewSupplierAutomationService(&supplierAutomationRepoStub{}, &supplierAutomationLockStub{}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountRateGuardRepository(logRepo)
	params := SupplierAccountRateGuardUnbindLogListParams{RunID: 8, Result: SupplierAccountRateGuardLogResultFailed, Page: 2, PageSize: 30}

	result, err := service.ListAccountRateGuardUnbindLogs(context.Background(), params)

	require.NoError(t, err)
	require.Equal(t, params, logRepo.listParams)
	require.Equal(t, int64(1), result.Total)
	require.Equal(t, int64(21), result.Items[0].ID)
}

func TestSupplierAutomationServiceMarksAccountRateGuardUnbindLogHandled(t *testing.T) {
	logRepo := &supplierAccountRateGuardRepoStub{logs: []SupplierAccountRateGuardUnbindLog{{ID: 22, Status: SupplierAccountRateGuardLogStatusPending}}}
	service := NewSupplierAutomationService(&supplierAutomationRepoStub{}, &supplierAutomationLockStub{}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountRateGuardRepository(logRepo)

	item, err := service.MarkAccountRateGuardUnbindLogHandled(context.Background(), 22)

	require.NoError(t, err)
	require.Equal(t, int64(22), item.ID)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, item.Status)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, logRepo.logs[0].Status)
}

func TestSupplierAutomationServiceValidatesRateGuardConfig(t *testing.T) {
	tests := []SupplierAutomationConfig{
		{RateGuardMaxSnapshotAgeSeconds: 59},
	}
	for _, config := range tests {
		task := SupplierAutomationTask{
			TaskCode: SupplierAutomationTaskRateGuard, CronExpression: "2-59/5 * * * *",
			TimeoutSeconds: 300, Config: config,
		}
		require.Error(t, validateSupplierAutomationTask(task))
	}
}

func TestSupplierAutomationServiceRejectsInvalidCron(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	task := *repo.tasks[SupplierAutomationTaskSync]
	task.CronExpression = "bad cron"

	err := service.UpdateTask(context.Background(), &task)

	require.Error(t, err)
}

func TestSupplierAutomationServiceRunsSyncTask(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600},
	}}
	lock := &supplierAutomationLockStub{acquired: true}
	syncSvc := &supplierAutomationSyncStub{}
	service := NewSupplierAutomationService(repo, lock, syncSvc, &supplierProviderDataRepoStub{})

	run, err := service.Run(context.Background(), SupplierAutomationTaskSync, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierAutomationStatusPartial, run.Status)
	require.Equal(t, 2, run.ProcessedCount)
	require.Equal(t, 1, syncSvc.called)
	// 同步任务会同时持有任务锁与上游拉取协调锁，结束时各释放一次。
	require.Equal(t, 2, lock.released)
}

func TestSupplierAutomationServiceIncludesFailedSupplierDetailsInPartialMessage(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600},
	}}
	syncSvc := &supplierAutomationSyncStub{result: SupplierProviderBatchSyncResult{
		ProcessedCount: 2,
		SuccessCount:   1,
		FailedCount:    1,
		Results: []SupplierProviderSyncResult{{
			ProviderID: 12,
			Scope:      SupplierSyncScopeAll,
			Status:     SupplierSyncStatusPartial,
			Message:    "部分同步失败",
			Stages: []SupplierProviderSyncStage{{
				Scope:   SupplierSyncScopeGroups,
				Status:  SupplierSyncStatusFailed,
				Message: "分组接口超时",
			}},
		}},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, syncSvc, &supplierProviderDataRepoStub{})

	run, err := service.Run(context.Background(), SupplierAutomationTaskSync, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierAutomationStatusPartial, run.Status)
	require.Contains(t, run.Message, "供应商 12")
	require.Contains(t, run.Message, "groups")
	require.Contains(t, run.Message, "分组接口超时")
	require.Equal(t, run.Message, repo.tasks[SupplierAutomationTaskSync].LastMessage)
}

func TestSupplierAutomationServiceOmitsSkippedProviderFromFailureMessage(t *testing.T) {
	result := SupplierProviderBatchSyncResult{
		ProcessedCount: 2,
		FailedCount:    1,
		SkippedCount:   1,
		Results: []SupplierProviderSyncResult{
			{
				ProviderID: 1,
				Scope:      SupplierSyncScopeAll,
				Status:     SupplierSyncStatusSkipped,
				Message:    ErrSupplierProviderSyncConflict.Error(),
			},
			{
				ProviderID: 2,
				Scope:      SupplierSyncScopeAll,
				Status:     SupplierSyncStatusFailed,
				Message:    "账号接口超时",
			},
		},
	}

	message := supplierAutomationBatchFailureMessage(result)

	require.Contains(t, message, "供应商 2")
	require.Contains(t, message, "账号接口超时")
	require.NotContains(t, message, ErrSupplierProviderSyncConflict.Error())
}
func TestSupplierAutomationServiceStoresStructuredProviderStageDetails(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600},
	}}
	syncSvc := &supplierAutomationSyncStub{result: SupplierProviderBatchSyncResult{
		ProcessedCount: 1,
		FailedCount:    1,
		Results: []SupplierProviderSyncResult{{
			ProviderID:   12,
			ProviderName: "供应商 A",
			Scope:        SupplierSyncScopeAll,
			Status:       SupplierSyncStatusPartial,
			Message:      "部分同步失败",
			Stages: []SupplierProviderSyncStage{{
				Scope:   SupplierSyncScopeAccounts,
				Status:  SupplierSyncStatusFailed,
				Message: "账号接口 404",
				Counts:  SupplierSyncCounts{CheckedCount: 1},
				EndpointResult: &SupplierProviderEndpointResult{
					Endpoint:        "/accounts",
					HTTPStatus:      404,
					DurationMS:      35,
					ResponseBytes:   18,
					ResponseSummary: "404 page not found",
					Error:           "supplier sub2api accounts failed with HTTP 404",
				},
			}},
		}},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, syncSvc, &supplierProviderDataRepoStub{})

	run, err := service.Run(context.Background(), SupplierAutomationTaskSync, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.NotNil(t, run.ResultDetail)
	require.Len(t, run.ResultDetail.Providers, 1)
	provider := run.ResultDetail.Providers[0]
	require.Equal(t, int64(12), provider.ProviderID)
	require.Equal(t, "供应商 A", provider.ProviderName)
	require.Len(t, provider.Stages, 1)
	require.Equal(t, SupplierSyncScopeAccounts, provider.Stages[0].Scope)
	require.Equal(t, 404, provider.Stages[0].HTTPStatus)
	require.Equal(t, "404 page not found", provider.Stages[0].ResponseSummary)
	require.NotNil(t, repo.runs[len(repo.runs)-1].ResultDetail)
}

func TestSupplierAutomationServiceRunsCleanupTask(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskCleanup: {
			TaskCode:       SupplierAutomationTaskCleanup,
			Name:           "清理",
			Enabled:        true,
			CronExpression: "30 3 * * *",
			TimeoutSeconds: 600,
			Config:         SupplierAutomationConfig{AutomationRunRetentionDays: 30, SyncRunRetentionDays: 30, MetricRetentionDays: 30, DailyStatRetentionDays: 365, InactiveAccountDays: 90, InactiveGroupDays: 90},
		},
	}}
	dataRepo := &supplierProviderDataRepoStub{}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, dataRepo)

	run, err := service.Run(context.Background(), SupplierAutomationTaskCleanup, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierAutomationStatusSuccess, run.Status)
}

func TestSupplierAutomationServiceReturnsConflictWhenLockBusy(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: false}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})

	_, err := service.Run(context.Background(), SupplierAutomationTaskSync, SupplierSyncTriggerManual)

	require.Error(t, err)
}

func TestSupplierAutomationSchedulerReloadsUpdatedSchedules(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	scheduler := NewSupplierAutomationScheduler(repo, service)

	require.NoError(t, scheduler.Reload(context.Background()))
	first := repo.tasks[SupplierAutomationTaskSync].NextRunAt
	require.NotNil(t, first)

	repo.tasks[SupplierAutomationTaskSync].CronExpression = "*/30 * * * *"
	require.NoError(t, scheduler.Reload(context.Background()))
	second := repo.tasks[SupplierAutomationTaskSync].NextRunAt
	require.NotNil(t, second)
}

func TestSupplierAutomationServiceFinishesFailedRun(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskSync: {TaskCode: SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{err: errors.New("sync failed")}, &supplierProviderDataRepoStub{})

	run, err := service.Run(context.Background(), SupplierAutomationTaskSync, SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierAutomationStatusFailed, run.Status)
	require.NotNil(t, run.FinishedAt)
}

type supplierAutomationAccountHealthGuardStub struct {
	called int
	config SupplierAccountHealthGuardConfig
	result SupplierAccountHealthGuardResult
	err    error
}

func (s *supplierAutomationAccountHealthGuardStub) Run(_ context.Context, config SupplierAccountHealthGuardConfig, _ time.Time) (SupplierAccountHealthGuardResult, error) {
	s.called++
	s.config = config
	return s.result, s.err
}

func TestSupplierAutomationServiceRunsAccountHealthGuardTask(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Name: "供应商账号健康守护", Enabled: true,
			CronExpression: "@every 3600s", TimeoutSeconds: 1800,
			Config: validSupplierAccountHealthGuardAutomationConfigWithIntervals(),
		},
	}}
	runner := &supplierAutomationAccountHealthGuardStub{result: SupplierAccountHealthGuardResult{
		TotalAccounts: 5, CheckedCount: 3, HealthyCount: 1, SlowCount: 1, FailedCount: 1,
		UnavailableCount: 2, DisabledCount: 1, CursorAccountID: 20,
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountHealthGuardService(runner)

	run, err := service.Run(context.Background(), SupplierAutomationTaskAccountHealthGuard, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, runner.called)
	require.Equal(t, 5, run.ProcessedCount)
	require.Equal(t, 2, run.SuccessCount)
	require.Equal(t, 3, run.FailedCount)
	require.Equal(t, SupplierAutomationStatusPartial, run.Status)
	require.Equal(t, "健康守护发现 3 个异常账号", run.Message)
	require.NotNil(t, run.ResultDetail)
	require.Equal(t, []int64{1}, runner.config.AccountIDs)
	require.Equal(t, map[int64]int{1: 600, 2: 300}, runner.config.AccountIntervals)
	require.Equal(t, int64(20), run.ResultDetail.AccountHealthGuard.CursorAccountID)
	require.Equal(t, int64(20), repo.tasks[SupplierAutomationTaskAccountHealthGuard].Config.AccountHealthGuardCursorAccountID)
}

func TestSupplierAutomationServiceBuildsAccountHealthGuardSuccessMessage(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Name: "供应商账号健康守护", Enabled: true,
			CronExpression: "@every 3600s", TimeoutSeconds: 1800,
			Config: validSupplierAccountHealthGuardAutomationConfig(),
		},
	}}
	runner := &supplierAutomationAccountHealthGuardStub{result: SupplierAccountHealthGuardResult{
		TotalAccounts: 4, CheckedCount: 3, HealthyCount: 2, SlowCount: 1, PendingCount: 1,
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountHealthGuardService(runner)

	run, err := service.Run(context.Background(), SupplierAutomationTaskAccountHealthGuard, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierAutomationStatusSuccess, run.Status)
	require.Equal(t, "健康守护检查 3 个账号，待下轮 1 个", run.Message)
}

func TestSupplierAutomationServiceRejectsSavingEmptyAccountHealthGuardWhitelist(t *testing.T) {
	config := validSupplierAccountHealthGuardAutomationConfig()
	config.AccountHealthGuardAccountIDs = nil
	task := SupplierAutomationTask{
		TaskCode: SupplierAutomationTaskAccountHealthGuard, CronExpression: "@every 3600s", TimeoutSeconds: 1800, Config: config,
	}
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})

	err := service.UpdateTask(context.Background(), &task)

	require.EqualError(t, err, "请至少选择一个需要检查的账号")
	require.Empty(t, repo.updatedTasks)
}

func TestSupplierAutomationServiceRejectsRunningEmptyAccountHealthGuardWhitelist(t *testing.T) {
	config := validSupplierAccountHealthGuardAutomationConfig()
	config.AccountHealthGuardAccountIDs = nil
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Enabled: true,
			CronExpression: "@every 3600s", TimeoutSeconds: 1800, Config: config,
		},
	}}
	runner := &supplierAutomationAccountHealthGuardStub{}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountHealthGuardService(runner)

	_, err := service.Run(context.Background(), SupplierAutomationTaskAccountHealthGuard, SupplierSyncTriggerManual)

	require.EqualError(t, err, "请至少选择一个需要检查的账号")
	require.Zero(t, runner.called)
}

func TestSupplierAutomationSchedulerAllowsDisabledEmptyAccountHealthGuardWhitelist(t *testing.T) {
	config := validSupplierAccountHealthGuardAutomationConfig()
	config.AccountHealthGuardAccountIDs = nil
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Enabled: false,
			CronExpression: "@every 3600s", TimeoutSeconds: 1800, Config: config,
		},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	scheduler := NewSupplierAutomationScheduler(repo, service)

	require.NoError(t, scheduler.Reload(context.Background()))
}

func TestSupplierAutomationSchedulerRejectsEnabledEmptyAccountHealthGuardWhitelist(t *testing.T) {
	config := validSupplierAccountHealthGuardAutomationConfig()
	config.AccountHealthGuardAccountIDs = nil
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Enabled: true,
			CronExpression: "@every 3600s", TimeoutSeconds: 1800, Config: config,
		},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	scheduler := NewSupplierAutomationScheduler(repo, service)

	// 启用但白名单为空时只跳过该任务，不再阻断整个调度器加载。
	require.NoError(t, scheduler.Reload(context.Background()))
	scheduler.Stop()
}

func TestSupplierAutomationServiceRejectsInvalidAccountHealthGuardConfig(t *testing.T) {
	valid := validSupplierAccountHealthGuardAutomationConfig()
	tests := []struct {
		name   string
		mutate func(*SupplierAutomationConfig)
	}{
		{name: "单次上限", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardMaxAccountsPerRun = 0 }},
		{name: "max accounts above limit", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardMaxAccountsPerRun = 1001 }},
		{name: "并发数", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardConcurrency = 0 }},
		{name: "concurrency above limit", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardConcurrency = 9 }},
		{name: "单账号超时", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardTimeoutPerAccountSeconds = 0 }},
		{name: "timeout below limit", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardTimeoutPerAccountSeconds = 4 }},
		{name: "timeout above limit", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardTimeoutPerAccountSeconds = 301 }},
		{name: "失败阈值", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardFailureThreshold = 0 }},
		{name: "慢响应阈值", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardSlowThreshold = 0 }},
		{name: "恢复阈值", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardRecoveryThreshold = 0 }},
		{name: "健康延迟", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardHealthyLatencyMs = 0 }},
		{name: "检查间隔低于下限", mutate: func(c *SupplierAutomationConfig) { c.AccountHealthGuardAccountIntervals = map[int64]int{1: 59} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := valid
			tt.mutate(&config)
			err := validateSupplierAutomationTask(SupplierAutomationTask{
				TaskCode: SupplierAutomationTaskAccountHealthGuard, CronExpression: "@every 3600s", TimeoutSeconds: 1800, Config: config,
			})
			require.Error(t, err)
		})
	}
}

func validSupplierAccountHealthGuardAutomationConfigWithIntervals() SupplierAutomationConfig {
	config := validSupplierAccountHealthGuardAutomationConfig()
	config.AccountHealthGuardAccountIntervals = map[int64]int{1: 600, 2: 300}
	return config
}

func validSupplierAccountHealthGuardAutomationConfig() SupplierAutomationConfig {
	return SupplierAutomationConfig{
		AccountHealthGuardMaxAccountsPerRun:        200,
		AccountHealthGuardConcurrency:              3,
		AccountHealthGuardTimeoutPerAccountSeconds: 90,
		AccountHealthGuardFailureThreshold:         3,
		AccountHealthGuardSlowThreshold:            3,
		AccountHealthGuardRecoveryThreshold:        2,
		AccountHealthGuardHealthyLatencyMs:         15000,
		AccountHealthGuardAccountIDs:               []int64{1},
		AccountHealthGuardAccountModels:            map[int64]string{},
		AccountHealthGuardPlatformModels:           map[string]string{},
		AccountHealthGuardPlatformLatencyMs:        map[string]int64{},
		AccountHealthGuardAccountIntervals:         map[int64]int{},
	}
}

func TestSupplierAutomationServiceRecordsConflictWhenLockBusy(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Enabled: true,
			CronExpression: "@every 120s", TimeoutSeconds: 1800,
			Config: validSupplierAccountHealthGuardAutomationConfig(),
		},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: false}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountHealthGuardService(&supplierAutomationAccountHealthGuardStub{})

	run, err := service.Run(context.Background(), SupplierAutomationTaskAccountHealthGuard, SupplierSyncTriggerScheduled)

	require.Error(t, err)
	require.Equal(t, ErrSupplierProviderSyncConflict, err)
	require.Len(t, repo.runs, 2)
	require.Equal(t, SupplierAutomationStatusFailed, repo.runs[1].Status)
	require.Equal(t, "任务锁占用中，跳过本次执行", repo.runs[1].Message)
	require.Equal(t, SupplierSyncTriggerScheduled, run.TriggerSource)
}

func TestSupplierAutomationRunDoesNotOverwriteCronExpression(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountRateGuard: {
			TaskCode: SupplierAutomationTaskAccountRateGuard, Name: "账号倍率守护", Enabled: true,
			CronExpression: "@every 20s", TimeoutSeconds: 600,
		},
	}}
	mutating := &supplierAutomationAccountRateGuardMutatingStub{
		repo:     repo,
		taskCode: SupplierAutomationTaskAccountRateGuard,
		newCron:  "@every 300s",
		result:   SupplierAccountRateGuardResult{CheckedAccounts: 2, UnboundGroups: 1},
	}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountRateGuardService(mutating)

	run, err := service.Run(context.Background(), SupplierAutomationTaskAccountRateGuard, SupplierSyncTriggerScheduled)
	require.NoError(t, err)
	require.Equal(t, SupplierAutomationStatusSuccess, run.Status)
	require.Equal(t, "@every 300s", repo.tasks[SupplierAutomationTaskAccountRateGuard].CronExpression)
	require.Equal(t, run.Status, repo.tasks[SupplierAutomationTaskAccountRateGuard].LastStatus)
	require.Equal(t, 1, mutating.called)
}

type supplierAutomationAccountRateGuardMutatingStub struct {
	repo     *supplierAutomationRepoStub
	taskCode string
	newCron  string
	result   SupplierAccountRateGuardResult
	err      error
	called   int
}

func (s *supplierAutomationAccountRateGuardMutatingStub) Run(_ context.Context, runID int64, mode SupplierAccountRateGuardMode, _ time.Time) (SupplierAccountRateGuardResult, error) {
	s.called++
	if task := s.repo.tasks[s.taskCode]; task != nil {
		task.CronExpression = s.newCron
	}
	return s.result, s.err
}

func TestSupplierAutomationUpdateTaskPreservesRuntimeStatus(t *testing.T) {
	now := time.Now()
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountRateGuard: {
			TaskCode: SupplierAutomationTaskAccountRateGuard, Name: "账号倍率守护", Enabled: true,
			CronExpression: "@every 20s", TimeoutSeconds: 600,
			LastStatus: SupplierAutomationStatusPartial, LastMessage: "存在告警", LastRunAt: &now,
		},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	err := service.UpdateTask(context.Background(), &SupplierAutomationTask{
		TaskCode: SupplierAutomationTaskAccountRateGuard, Name: "账号倍率守护", Enabled: true,
		CronExpression: "@every 300s", TimeoutSeconds: 600,
		LastStatus: "", LastMessage: "",
	})
	require.NoError(t, err)
	saved := repo.tasks[SupplierAutomationTaskAccountRateGuard]
	require.Equal(t, "@every 300s", saved.CronExpression)
	require.Equal(t, SupplierAutomationStatusPartial, saved.LastStatus)
	require.Equal(t, "存在告警", saved.LastMessage)
	require.NotNil(t, saved.LastRunAt)
}

func TestSupplierAutomationHealthGuardRunPreservesConcurrentConfig(t *testing.T) {
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Enabled: true,
			CronExpression: "@every 120s", TimeoutSeconds: 1800,
			Config: validSupplierAccountHealthGuardAutomationConfig(),
		},
	}}
	guard := &supplierAutomationAccountHealthGuardMutatingStub{
		repo: repo,
		result: SupplierAccountHealthGuardResult{
			CheckedCount: 1, HealthyCount: 1, CursorAccountID: 99,
		},
	}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	service.SetAccountHealthGuardService(guard)

	run, err := service.Run(context.Background(), SupplierAutomationTaskAccountHealthGuard, SupplierSyncTriggerScheduled)
	require.NoError(t, err)
	require.Equal(t, SupplierAutomationStatusSuccess, run.Status)

	saved := repo.tasks[SupplierAutomationTaskAccountHealthGuard]
	require.Equal(t, "@every 600s", saved.CronExpression)
	require.Equal(t, []int64{1, 2, 3}, saved.Config.AccountHealthGuardAccountIDs)
	require.Equal(t, int64(99), saved.Config.AccountHealthGuardCursorAccountID)
	require.Equal(t, run.Status, saved.LastStatus)
}

type supplierAutomationAccountHealthGuardMutatingStub struct {
	repo   *supplierAutomationRepoStub
	result SupplierAccountHealthGuardResult
	err    error
	called int
}

func (s *supplierAutomationAccountHealthGuardMutatingStub) Run(_ context.Context, config SupplierAccountHealthGuardConfig, _ time.Time) (SupplierAccountHealthGuardResult, error) {
	s.called++
	if task := s.repo.tasks[SupplierAutomationTaskAccountHealthGuard]; task != nil {
		task.CronExpression = "@every 600s"
		task.Config.AccountHealthGuardAccountIDs = []int64{1, 2, 3}
	}
	return s.result, s.err
}

func TestSupplierAutomationSchedulerSkipsInvalidTaskWithoutBlockingOthers(t *testing.T) {
	badConfig := validSupplierAccountHealthGuardAutomationConfig()
	badConfig.AccountHealthGuardAccountIDs = nil
	repo := &supplierAutomationRepoStub{tasks: map[string]*SupplierAutomationTask{
		SupplierAutomationTaskAccountHealthGuard: {
			TaskCode: SupplierAutomationTaskAccountHealthGuard, Enabled: true,
			CronExpression: "@every 1s", TimeoutSeconds: 1800, Config: badConfig,
		},
		SupplierAutomationTaskCleanup: {
			TaskCode: SupplierAutomationTaskCleanup, Enabled: true,
			CronExpression: "@every 1s", TimeoutSeconds: 5,
			Config: SupplierAutomationConfig{
				AutomationRunRetentionDays: 1, SyncRunRetentionDays: 1, MetricRetentionDays: 1,
				DailyStatRetentionDays: 1, InactiveAccountDays: 1, InactiveGroupDays: 1,
			},
		},
	}}
	service := NewSupplierAutomationService(repo, &supplierAutomationLockStub{acquired: true}, &supplierAutomationSyncStub{}, &supplierProviderDataRepoStub{})
	scheduler := NewSupplierAutomationScheduler(repo, service)

	require.NoError(t, scheduler.Reload(context.Background()))
	defer scheduler.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(repo.runs) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.Greater(t, len(repo.runs), 0, "valid cleanup task should still be scheduled when health guard config is invalid")
}
