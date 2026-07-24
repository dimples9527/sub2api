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

func TestSupplierAccountRateGuardRepositoryGroupsCandidateRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewSupplierAccountRateGuardRepository(db)

	rows := sqlmock.NewRows([]string{
		"provider_account_id", "provider_id", "provider_name", "upstream_account_key", "upstream_account_name",
		"raw_rate", "rate_scale", "match_count", "local_account_id", "local_account_name", "reverse_match_count",
		"schedulable", "local_group_id", "local_group_name", "local_group_rate",
	}).
		AddRow(int64(11), int64(1), "供应商甲", "key-1", "上游账号", 1.0, 1.2, 1, int64(21), "本地账号", 1, true, int64(31), "分组一", 1.1).
		AddRow(int64(11), int64(1), "供应商甲", "key-1", "上游账号", 1.0, 1.2, 1, int64(21), "本地账号", 1, true, int64(32), "分组二", 1.3)
	mock.ExpectQuery("SELECT a.id AS provider_account_id").WithArgs(int64(1), sqlmock.AnyArg()).WillReturnRows(rows)

	items, err := repo.ListAccountRateGuardCandidates(context.Background(), 1, []string{"key-1"})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, service.SupplierAccountRateGuardMatchMatched, items[0].MatchStatus)
	require.Len(t, items[0].Groups, 2)
	require.Equal(t, int64(32), items[0].Groups[1].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountRateGuardRepositoryCreatesAndListsLogs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewSupplierAccountRateGuardRepository(db)
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	handledAt := now.Add(time.Minute)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO supplier_account_rate_guard_unbind_logs").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err = repo.CreateAccountRateGuardUnbindLogs(context.Background(), []service.SupplierAccountRateGuardUnbindLog{{
		RunID: 9, ProviderID: 1, ProviderName: "供应商甲", UpstreamAccountKey: "key-1",
		Mode: string(service.SupplierAccountRateGuardModeExecute), Result: service.SupplierAccountRateGuardLogResultFailed,
		ErrorMessage: "解绑失败", CreatedAt: now,
	}})
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_account_rate_guard_unbind_logs WHERE 1=1 AND run_id = $1 AND mode = $2 AND result = $3 AND status = $4")).
		WithArgs(int64(9), string(service.SupplierAccountRateGuardModeExecute), service.SupplierAccountRateGuardLogResultFailed, service.SupplierAccountRateGuardLogStatusHandled).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_account_rate_guard_unbind_logs WHERE 1=1 AND run_id = $1 AND mode = $2 AND result = $3 AND status = $4")).
		WithArgs(int64(9), string(service.SupplierAccountRateGuardModeExecute), service.SupplierAccountRateGuardLogResultFailed, service.SupplierAccountRateGuardLogStatusPending).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery("SELECT id, run_id, provider_id").
		WithArgs(int64(9), string(service.SupplierAccountRateGuardModeExecute), service.SupplierAccountRateGuardLogResultFailed, service.SupplierAccountRateGuardLogStatusHandled, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "run_id", "provider_id", "provider_name", "supplier_provider_account_id", "upstream_account_key", "upstream_account_name",
			"local_account_id", "local_account_name", "local_group_id", "local_group_name", "raw_upstream_rate", "rate_scale",
			"effective_upstream_rate", "local_group_rate", "mode", "result", "status", "before_bound", "after_bound",
			"before_schedulable", "after_schedulable", "error_message", "handled_at", "created_at", "platform",
		}).AddRow(int64(1), int64(9), int64(1), "供应商甲", nil, "key-1", "上游账号", nil, "", nil, "", 1.0, 1.0, 1.0, 1.2, "execute", "failed", "handled", true, true, nil, nil, "解绑失败", handledAt, now, "openai"))

	result, err := repo.ListAccountRateGuardUnbindLogs(context.Background(), service.SupplierAccountRateGuardUnbindLogListParams{
		RunID: 9, Mode: string(service.SupplierAccountRateGuardModeExecute), Result: service.SupplierAccountRateGuardLogResultFailed,
		Status: service.SupplierAccountRateGuardLogStatusHandled, Page: 1, PageSize: 20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Zero(t, result.PendingCount)
	require.Len(t, result.Items, 1)
	require.Equal(t, service.SupplierAccountRateGuardLogStatusHandled, result.Items[0].Status)
	require.Equal(t, handledAt, *result.Items[0].HandledAt)
	require.Equal(t, "解绑失败", result.Items[0].ErrorMessage)
	require.Equal(t, "openai", result.Items[0].Platform)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountRateGuardRepositoryOnlyUnboundAndMarksHandled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewSupplierAccountRateGuardRepository(db)
	now := time.Date(2026, 7, 23, 13, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_account_rate_guard_unbind_logs WHERE 1=1 AND result = $1")).
		WithArgs(service.SupplierAccountRateGuardLogResultUnbound).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_account_rate_guard_unbind_logs WHERE 1=1 AND result = $1 AND status = $2")).
		WithArgs(service.SupplierAccountRateGuardLogResultUnbound, service.SupplierAccountRateGuardLogStatusPending).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
	mock.ExpectQuery("SELECT id, run_id, provider_id").
		WithArgs(service.SupplierAccountRateGuardLogResultUnbound, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "run_id", "provider_id", "provider_name", "supplier_provider_account_id", "upstream_account_key", "upstream_account_name",
			"local_account_id", "local_account_name", "local_group_id", "local_group_name", "raw_upstream_rate", "rate_scale",
			"effective_upstream_rate", "local_group_rate", "mode", "result", "status", "before_bound", "after_bound",
			"before_schedulable", "after_schedulable", "error_message", "handled_at", "created_at", "platform",
		}).AddRow(int64(2), int64(10), int64(1), "供应商甲", nil, "key-2", "上游账号", int64(21), "本地账号", int64(31), "风险组", 1.0, 1.0, 1.0, 1.2, "execute", "unbound", "pending", true, false, true, false, "", nil, now, "openai"))

	result, err := repo.ListAccountRateGuardUnbindLogs(context.Background(), service.SupplierAccountRateGuardUnbindLogListParams{OnlyUnbound: true, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(2), result.PendingCount)

	mock.ExpectQuery("UPDATE supplier_account_rate_guard_unbind_logs").
		WithArgs(int64(2), service.SupplierAccountRateGuardLogStatusHandled).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "run_id", "provider_id", "provider_name", "supplier_provider_account_id", "upstream_account_key", "upstream_account_name",
			"local_account_id", "local_account_name", "local_group_id", "local_group_name", "raw_upstream_rate", "rate_scale",
			"effective_upstream_rate", "local_group_rate", "mode", "result", "status", "before_bound", "after_bound",
			"before_schedulable", "after_schedulable", "error_message", "handled_at", "created_at", "platform",
		}).AddRow(int64(2), int64(10), int64(1), "供应商甲", nil, "key-2", "上游账号", int64(21), "本地账号", int64(31), "风险组", 1.0, 1.0, 1.0, 1.2, "execute", "unbound", "handled", true, false, true, false, "", now, now, ""))

	item, err := repo.MarkAccountRateGuardUnbindLogHandled(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, service.SupplierAccountRateGuardLogStatusHandled, item.Status)
	require.NotNil(t, item.HandledAt)
	require.NoError(t, mock.ExpectationsWereMet())
}
