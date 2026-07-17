package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierAutomationRepositoryListsTasksAndRuns(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierAutomationRepository(db)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	config := `{"sync_run_retention_days":30}`

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, task_code, name, enabled, cron_expression")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "task_code", "name", "enabled", "cron_expression", "timeout_seconds",
			"config_json", "last_status", "last_message", "last_run_at", "next_run_at",
		}).AddRow(int64(1), service.SupplierAutomationTaskSync, "同步", true, "*/15 * * * *", 600, config, "success", "ok", now, now))

	tasks, err := repo.ListTasks(context.Background())
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, 30, tasks[0].Config.SyncRunRetentionDays)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_automation_runs")).
		WithArgs(service.SupplierAutomationTaskSync, "success").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, task_code, trigger_source, status")).
		WithArgs(service.SupplierAutomationTaskSync, "success", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "task_code", "trigger_source", "status", "message", "processed_count",
			"success_count", "failed_count", "result_detail", "started_at", "finished_at", "created_at",
		}).AddRow(int64(9), service.SupplierAutomationTaskSync, "manual", "success", "ok", 2, 2, 0, `{"providers":[{"provider_id":12,"provider_name":"供应商 A","status":"success","stages":[]}]}`, now, now, now))

	runs, err := repo.ListRuns(context.Background(), service.SupplierAutomationRunListParams{TaskCode: service.SupplierAutomationTaskSync, Status: "success", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), runs.Total)
	require.Len(t, runs.Items, 1)
	require.NotNil(t, runs.Items[0].ResultDetail)
	require.Equal(t, int64(12), runs.Items[0].ResultDetail.Providers[0].ProviderID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAutomationRepositoryPersistsRunResultDetail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierAutomationRepository(db)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	run := &service.SupplierAutomationRun{
		ID:             9,
		TaskCode:       service.SupplierAutomationTaskSync,
		TriggerSource:  "manual",
		Status:         service.SupplierAutomationStatusPartial,
		Message:        "部分供应商同步失败",
		ProcessedCount: 1,
		FailedCount:    1,
		StartedAt:      now,
		FinishedAt:     &now,
		ResultDetail: &service.SupplierAutomationRunDetail{Providers: []service.SupplierAutomationProviderRunDetail{{
			ProviderID:   12,
			ProviderName: "供应商 A",
			Status:       service.SupplierSyncStatusPartial,
		}}},
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_automation_runs")).
		WithArgs(run.ID, run.Status, run.Message, run.ProcessedCount, run.SuccessCount, run.FailedCount, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.FinishRun(context.Background(), run))
	require.NoError(t, mock.ExpectationsWereMet())
}
