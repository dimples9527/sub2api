package repository

import (
	"context"
	"encoding/json"
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

func TestSupplierAutomationRepositoryUpdateTaskOnlyWritesSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierAutomationRepository(db)
	task := &service.SupplierAutomationTask{
		TaskCode:       "supplier_account_rate_guard",
		Enabled:        true,
		CronExpression: "@every 300s",
		TimeoutSeconds: 600,
		Config:         service.SupplierAutomationConfig{},
		LastStatus:     "should-not-be-written",
		LastMessage:    "should-not-be-written",
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE supplier_automation_tasks
SET enabled=$2, cron_expression=$3, timeout_seconds=$4, config_json=$5, updated_at=NOW()
WHERE task_code=$1`)).
		WithArgs(task.TaskCode, task.Enabled, task.CronExpression, task.TimeoutSeconds, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateTask(context.Background(), task))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAutomationRepositoryUpdateTaskRuntimeOnlyWritesStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierAutomationRepository(db)
	now := time.Now()
	next := now.Add(time.Minute)
	task := &service.SupplierAutomationTask{
		TaskCode:       "supplier_account_rate_guard",
		Enabled:        false,
		CronExpression: "@every 20s",
		TimeoutSeconds: 1,
		LastStatus:     service.SupplierAutomationStatusSuccess,
		LastMessage:    "执行成功",
		LastRunAt:      &now,
		NextRunAt:      &next,
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE supplier_automation_tasks
SET last_status=$2, last_message=$3, last_run_at=$4, next_run_at=$5, updated_at=NOW()
WHERE task_code=$1`)).
		WithArgs(task.TaskCode, task.LastStatus, task.LastMessage, task.LastRunAt, task.NextRunAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateTaskRuntime(context.Background(), task))
	require.NoError(t, mock.ExpectationsWereMet())
}

// 下面这段夹具是从本地真实库中 supplier_account_rate_guard 任务的 config_json 原样抓下来的
// （jsonb 会重排键序），代表"改动上线前就存在的旧配置"。
// 它存在的意义：账号倍率守护的分组开关字段在老配置里**不存在**，
// 必须验证老配置读进来后不会出错、也不会意外带出非空的分组列表。
const realLegacyAccountRateGuardConfigJSON = `{
  "sync_run_retention_days": 0,
  "daily_stat_retention_days": 0,
  "automation_run_retention_days": 0,
  "inactive_group_retention_days": 0,
  "metric_snapshot_retention_days": 0,
  "inactive_account_retention_days": 0,
  "account_health_guard_account_ids": [],
  "account_health_guard_concurrency": 3,
  "account_health_guard_account_models": {},
  "account_health_guard_slow_threshold": 3,
  "rate_guard_max_snapshot_age_seconds": 0,
  "account_health_guard_platform_models": {},
  "account_health_guard_cursor_account_id": 0,
  "account_health_guard_failure_threshold": 3,
  "account_health_guard_healthy_latency_ms": 15000,
  "account_health_guard_recovery_threshold": 2,
  "account_health_guard_platform_latency_ms": {},
  "account_health_guard_max_accounts_per_run": 200,
  "account_health_guard_timeout_per_account_seconds": 90
}`

// TestSupplierAutomationRepositoryReadsLegacyConfigWithoutDisabledGroups 锁定向后兼容：
// 旧配置里没有 account_rate_guard_disabled_group_ids，读出来必须是空（即"所有分组都参与守护"）。
func TestSupplierAutomationRepositoryReadsLegacyConfigWithoutDisabledGroups(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierAutomationRepository(db)
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, task_code, name, enabled, cron_expression")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "task_code", "name", "enabled", "cron_expression", "timeout_seconds",
			"config_json", "last_status", "last_message", "last_run_at", "next_run_at",
		}).AddRow(int64(4), service.SupplierAutomationTaskAccountRateGuard, "账号倍率守护",
			true, "@every 300s", 600, realLegacyAccountRateGuardConfigJSON, "success", "ok", now, now))

	tasks, err := repo.ListTasks(context.Background())
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// 旧配置没有该字段 ⇒ 空列表 ⇒ 所有分组都参与守护（默认开启的落点）。
	require.Empty(t, tasks[0].Config.AccountRateGuardDisabledGroupIDs)
	// 顺带确认同结构体里的其他字段没被新字段挤掉（struct tag 冲突会在这里暴露）。
	require.Equal(t, 3, tasks[0].Config.AccountHealthGuardConcurrency)
	require.Equal(t, 200, tasks[0].Config.AccountHealthGuardMaxAccountsPerRun)
	require.Equal(t, int64(15000), tasks[0].Config.AccountHealthGuardHealthyLatencyMs)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestSupplierAutomationRepositoryWritesDisabledGroupIDsIntoConfigJSON 确保关闭分组能真正落库：
// config_json 是整块覆盖写，新字段必须出现在被写入的 JSON 里，否则保存后会被静默丢弃。
func TestSupplierAutomationRepositoryWritesDisabledGroupIDsIntoConfigJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierAutomationRepository(db)
	task := &service.SupplierAutomationTask{
		TaskCode:       service.SupplierAutomationTaskAccountRateGuard,
		Enabled:        true,
		CronExpression: "@every 300s",
		TimeoutSeconds: 600,
		Config: service.SupplierAutomationConfig{
			AccountRateGuardDisabledGroupIDs: []int64{81, 66},
			AccountHealthGuardConcurrency:    3,
		},
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE supplier_automation_tasks
SET enabled=$2, cron_expression=$3, timeout_seconds=$4, config_json=$5, updated_at=NOW()
WHERE task_code=$1`)).
		WithArgs(task.TaskCode, task.Enabled, task.CronExpression, task.TimeoutSeconds, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateTask(context.Background(), task))
	require.NoError(t, mock.ExpectationsWereMet())

	// 仓储实现就是 json.Marshal(task.Config)，这里直接对序列化结果断言，
	// 确认新字段确实落在了被整块覆盖写入的 JSON 里（否则保存后会被静默丢弃）。
	raw, err := json.Marshal(task.Config)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"account_rate_guard_disabled_group_ids":[81,66]`)

	// 反向：空列表必须序列化成 [] 而不是被省略，否则"全部参与"无法把已有关闭项清掉。
	task.Config.AccountRateGuardDisabledGroupIDs = []int64{}
	raw, err = json.Marshal(task.Config)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"account_rate_guard_disabled_group_ids":[]`)
}
