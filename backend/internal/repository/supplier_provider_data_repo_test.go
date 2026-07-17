package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newSupplierProviderDataRepoMock(t *testing.T) (*supplierProviderDataRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewSupplierProviderDataRepository(db).(*supplierProviderDataRepository), mock
}

type supplierProviderNonNilArg struct{}

func (supplierProviderNonNilArg) Match(value driver.Value) bool {
	return value != nil
}

func TestSupplierProviderDataRepositoryReplaceAccountsUpsertsAndDeactivatesMissing(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	seenAt := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_accounts")).
		WithArgs(int64(42), "account-1", "Primary", "active", "group-1", "VIP", 2.5, "active", seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_accounts")).
		WithArgs(int64(42), "account-2", "Second", "disabled", "", "", 0.0, "disabled", seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_accounts SET active = FALSE")).
		WithArgs(int64(42), sqlmock.AnyArg(), seenAt).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_runtime_stats")).
		WithArgs(int64(42), 1, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	counts, err := repo.ReplaceAccounts(context.Background(), 42, []service.SupplierProviderRemoteAccount{
		{Key: "account-1", Name: "Primary", Status: "active", GroupKey: "group-1", GroupName: "VIP", RateMultiplier: 2.5, RawStatus: "active"},
		{Key: "account-2", Name: "Second", Status: "disabled", RawStatus: "disabled"},
	}, seenAt)

	require.NoError(t, err)
	require.Equal(t, 2, counts.CheckedCount)
	require.Equal(t, 3, counts.SkippedCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryReplaceGroupsUpsertsAndDeactivatesMissing(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	seenAt := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_groups")).
		WithArgs(int64(42), "group-1", "VIP", 2.5, "active", seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET active = FALSE")).
		WithArgs(int64(42), sqlmock.AnyArg(), seenAt).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	counts, err := repo.ReplaceGroups(context.Background(), 42, []service.SupplierProviderRemoteGroup{
		{Key: "group-1", Name: "VIP", RateMultiplier: 2.5, RawStatus: "active"},
	}, seenAt)

	require.NoError(t, err)
	require.Equal(t, 1, counts.CheckedCount)
	require.Equal(t, 2, counts.SkippedCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryUpdateBalanceAndCostUpsertsDailyStats(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	seenAt := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_runtime_stats SET current_balance")).
		WithArgs(int64(42), 321.5, seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at)")).
		WithArgs(int64(42), 321.5, supplierProviderNonNilArg{}, seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_daily_stats")).
		WithArgs(int64(42), sqlmock.AnyArg(), 321.5).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.UpdateBalance(context.Background(), 42, 321.5, seenAt))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_runtime_stats SET today_cost")).
		WithArgs(int64(42), 45.625, seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at)")).
		WithArgs(int64(42), supplierProviderNonNilArg{}, 45.625, seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_daily_stats")).
		WithArgs(int64(42), sqlmock.AnyArg(), 45.625).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.UpdateCost(context.Background(), 42, 45.625, seenAt))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListAccountsPaginates(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	active := true
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_provider_accounts a")).
		WithArgs(int64(42), active, "%pri%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("FROM supplier_provider_accounts a")).
		WithArgs(int64(42), active, "%pri%", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_account_key", "name", "status",
			"group_key", "group_name", "rate_multiplier", "raw_status", "active", "last_seen_at", "inactive_at",
		}).AddRow(int64(7), int64(42), "Supplier A", "account-1", "Primary", "active", "group-1", "VIP", 2.5, "active", true, now, nil))

	result, err := repo.ListAccounts(context.Background(), service.SupplierProviderDataListParams{
		ProviderID: 42,
		Active:     &active,
		Search:     "pri",
		Page:       2,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Equal(t, 2, result.Page)
	require.Equal(t, 20, result.PageSize)
	require.Len(t, result.Items, 1)
	require.Equal(t, "Primary", result.Items[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryCleanupUsesBatchLimit(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	policy := service.SupplierCleanupPolicy{
		AutomationRunRetentionDays: 30,
		SyncRunRetentionDays:       30,
		MetricRetentionDays:        30,
		DailyStatRetentionDays:     365,
		InactiveAccountDays:        90,
		InactiveGroupDays:          90,
	}

	for _, rows := range []int64{2, 1} {
		mock.ExpectExec(regexp.QuoteMeta("WITH target AS")).
			WithArgs(sqlmock.AnyArg(), 2).
			WillReturnResult(sqlmock.NewResult(0, rows))
	}
	for range 5 {
		mock.ExpectExec(regexp.QuoteMeta("WITH target AS")).
			WithArgs(sqlmock.AnyArg(), 2).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	counts, err := repo.Cleanup(context.Background(), policy, now, 2)

	require.NoError(t, err)
	require.Equal(t, 3, counts.AutomationRuns)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryCleanupUsesCapturedAtForMetricSnapshots(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	policy := service.SupplierCleanupPolicy{
		AutomationRunRetentionDays: 30,
		SyncRunRetentionDays:       30,
		MetricRetentionDays:        30,
		DailyStatRetentionDays:     365,
		InactiveAccountDays:        90,
		InactiveGroupDays:          90,
	}

	for range 2 {
		mock.ExpectExec(regexp.QuoteMeta("WITH target AS")).
			WithArgs(sqlmock.AnyArg(), 1000).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectExec("supplier_provider_metric_snapshots WHERE captured_at <").
		WithArgs(now.AddDate(0, 0, -30), 1000).
		WillReturnResult(sqlmock.NewResult(0, 0))
	for range 3 {
		mock.ExpectExec(regexp.QuoteMeta("WITH target AS")).
			WithArgs(sqlmock.AnyArg(), 1000).
			WillReturnResult(sqlmock.NewResult(0, 0))
	}

	_, err := repo.Cleanup(context.Background(), policy, now, 1000)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
