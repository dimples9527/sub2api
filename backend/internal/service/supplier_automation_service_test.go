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
	clone := *task
	r.tasks[task.TaskCode] = &clone
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
	acquired bool
	released int
}

func (l *supplierAutomationLockStub) TryAcquireAutomationLock(context.Context, string, string, time.Duration) (bool, error) {
	return l.acquired, nil
}
func (l *supplierAutomationLockStub) ReleaseAutomationLock(context.Context, string, string) error {
	l.released++
	return nil
}

type supplierAutomationSyncStub struct {
	called int
	err    error
	result SupplierProviderBatchSyncResult
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
	require.Equal(t, 1, lock.released)
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
