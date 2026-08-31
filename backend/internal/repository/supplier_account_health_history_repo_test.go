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

func TestSupplierAccountHealthHistoryRepositorySave(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	latency := int64(240)
	checkedAt := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO supplier_account_health_history`)).
		WithArgs(int64(21), "account-one", int64(3), "provider-a", "openai", checkedAt, checkedAt.Add(-time.Second), checkedAt, "healthy", latency, int64(500), "gpt-4o", true, true, "none", 0, 0, 1, "probe succeeded", "").
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	err = repo.Save(context.Background(), service.SupplierAccountHealthHistoryRecord{
		LocalAccountID: 21, LocalAccountName: "account-one", ProviderID: 3, ProviderName: "provider-a", Platform: "openai",
		CheckedAt: checkedAt, StartedAt: checkedAt.Add(-time.Second), FinishedAt: checkedAt, Status: "healthy",
		LatencyMs: &latency, LatencyLimitMs: 500, ModelID: "gpt-4o", SchedulableBefore: true, SchedulableAfter: true,
		Action: "none", ConsecutiveHealthy: 1, Reason: "probe succeeded",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthHistoryRepositoryListAccounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	checkedAt := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)ARRAY_AGG\(DISTINCT a\.provider_id\).*COUNT\(\*\) OVER\(\).*supplier_automation_tasks.*src\.provider_ids @>`).
		WithArgs(int64(3), "openai", "%account%", "failed", service.SupplierAutomationTaskAccountHealthGuard, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"local_account_id", "local_account_name", "provider_id", "provider_name", "platform", "schedulable",
			"status", "checked_at", "latency_ms", "latency_limit_ms", "consecutive_failures",
			"upstream_rate_multiplier", "effective_rate_multiplier",
			"total_count", "guard_enabled",
		}).AddRow(21, "account-one", 3, "provider-a", "openai", false, "failed", checkedAt, nil, 500, 2, 1.5, 3.0, 1, true))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.ListAccounts(context.Background(), service.SupplierAccountHealthAccountListParams{
		ProviderID: 3, Platform: "openai", Search: "account", HealthStatus: "failed", Page: 1, PageSize: 20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(21), result.Items[0].LocalAccountID)
	require.Nil(t, result.Items[0].LatencyMs)
	require.Equal(t, 1.5, result.Items[0].UpstreamRateMultiplier)
	require.Equal(t, 3.0, result.Items[0].EffectiveRateMultiplier)
	require.False(t, result.Items[0].Schedulable)
	require.True(t, result.Items[0].GuardEnabled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthHistoryRepositoryListAccountsAllowsAccountsWithoutHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`(?s)ARRAY_AGG\(DISTINCT a\.provider_id\).*COUNT\(\*\) OVER\(\).*supplier_automation_tasks.*WHERE TRUE`).
		WithArgs(service.SupplierAutomationTaskAccountHealthGuard, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"local_account_id", "local_account_name", "provider_id", "provider_name", "platform", "schedulable",
			"status", "checked_at", "latency_ms", "latency_limit_ms", "consecutive_failures",
			"upstream_rate_multiplier", "effective_rate_multiplier",
			"total_count", "guard_enabled",
		}).AddRow(21, "account-one", 3, "provider-a", "openai", true, nil, nil, nil, 500, 0, 2.5, 2.5, 1, false))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.ListAccounts(context.Background(), service.SupplierAccountHealthAccountListParams{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "", result.Items[0].Status)
	require.Nil(t, result.Items[0].CheckedAt)
	require.Nil(t, result.Items[0].LatencyMs)
	require.Equal(t, 2.5, result.Items[0].UpstreamRateMultiplier)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 「上游倍率」必须取自 supplier_provider_accounts，而不是本地 accounts.rate_multiplier
// （后者是账号计费倍率、默认 1.0，会让整列都显示 1x）。
func TestSupplierAccountHealthHistoryRepositoryListAccountsReadsUpstreamRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`(?s)MAX\(a\.rate_multiplier \* p\.account_rate_multiplier_scale\) AS effective_rate_multiplier.*ARRAY_AGG\(a\.rate_multiplier ORDER BY.*\)\)\[1\] AS upstream_rate_multiplier.*src\.upstream_rate_multiplier,\s+src\.effective_rate_multiplier`).
		WithArgs(service.SupplierAutomationTaskAccountHealthGuard, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"local_account_id", "local_account_name", "provider_id", "provider_name", "platform", "schedulable",
			"status", "checked_at", "latency_ms", "latency_limit_ms", "consecutive_failures",
			"upstream_rate_multiplier", "effective_rate_multiplier",
			"total_count", "guard_enabled",
		}).AddRow(21, "account-one", 3, "provider-a", "openai", true, nil, nil, nil, 500, 0, 2.0, 5.0, 1, true))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.ListAccounts(context.Background(), service.SupplierAccountHealthAccountListParams{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, 2.0, result.Items[0].UpstreamRateMultiplier)
	require.Equal(t, 5.0, result.Items[0].EffectiveRateMultiplier)
	require.NoError(t, mock.ExpectationsWereMet())
}
func TestSupplierAccountHealthHistoryRepositoryListAccountsFiltersUncheckedAccounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`(?s)latest\.status IS NULL.*ORDER BY local_account\.name ASC`).
		WithArgs(service.SupplierAutomationTaskAccountHealthGuard, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"local_account_id", "local_account_name", "provider_id", "provider_name", "platform", "schedulable",
			"status", "checked_at", "latency_ms", "latency_limit_ms", "consecutive_failures",
			"upstream_rate_multiplier", "effective_rate_multiplier",
			"total_count", "guard_enabled",
		}).AddRow(21, "account-one", 3, "provider-a", "openai", true, nil, nil, nil, 500, 0, 1, 1, 1, true))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.ListAccounts(context.Background(), service.SupplierAccountHealthAccountListParams{
		HealthStatus: service.SupplierAccountHealthStatusUnchecked, Page: 1, PageSize: 20,
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "", result.Items[0].Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthHistoryRepositoryGetSummaryIgnoresStatusFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER \(WHERE latest\.status = 'healthy'\).*COUNT\(\*\) FILTER \(WHERE latest\.status IS NULL\).*src\.provider_ids @>.*src\.platform = \$2`).
		WithArgs(int64(3), "openai").
		WillReturnRows(sqlmock.NewRows([]string{"total", "healthy", "slow", "failed", "unchecked"}).AddRow(9, 5, 2, 1, 1))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	summary, err := repo.GetSummary(context.Background(), service.SupplierAccountHealthAccountListParams{
		ProviderID: 3, Platform: "openai", HealthStatus: "failed",
	})

	require.NoError(t, err)
	require.Equal(t, service.SupplierAccountHealthSummary{Total: 9, Healthy: 5, Slow: 2, Failed: 1, Unchecked: 1}, summary)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthHistoryRepositoryGetTrendAndDeleteBefore(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	checkedAt := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	since := checkedAt.Add(-24 * time.Hour)
	until := checkedAt.Add(time.Hour)
	mock.ExpectQuery(`(?s)SELECT checked_at, status, latency_ms, latency_limit_ms, reason, action, error_message.*checked_at >= \$2.*checked_at < \$3.*checked_at ASC, id ASC`).
		WithArgs(int64(21), since, until).
		WillReturnRows(sqlmock.NewRows([]string{"checked_at", "status", "latency_ms", "latency_limit_ms", "reason", "action", "error_message"}).
			AddRow(checkedAt, "failed", nil, 500, "timeout", "disable", "timeout"))
	mock.ExpectExec(`(?s)WITH target AS.*checked_at < \$1.*LIMIT \$2.*DELETE FROM supplier_account_health_history`).
		WithArgs(checkedAt, 1000).
		WillReturnResult(sqlmock.NewResult(0, 2))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.GetTrend(context.Background(), 21, since, until)
	require.NoError(t, err)
	require.Len(t, result.Points, 1)
	require.Nil(t, result.Points[0].LatencyMs)
	deleted, err := repo.DeleteBefore(context.Background(), checkedAt, 1000)
	require.NoError(t, err)
	require.Equal(t, 2, deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthHistoryRepositoryValidateAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`(?s)SELECT EXISTS.*accounts local_account.*supplier_provider_accounts a ON a\.active = TRUE.*supplier_providers p ON p\.id = a\.provider_id AND p\.enabled = TRUE.*local_account\.deleted_at IS NULL`).
		WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	require.NoError(t, repo.ValidateAccount(context.Background(), 21))
	require.NoError(t, mock.ExpectationsWereMet())
}
func TestSupplierAccountHealthHistoryRepositoryGetTrendsReturnsAllRangePoints(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	checkedAt := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	since := checkedAt.Add(-24 * time.Hour)
	until := checkedAt.Add(time.Hour)
	mock.ExpectQuery(`(?s)SELECT DISTINCT local_account\.id.*local_account\.id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21).AddRow(37))
	mock.ExpectQuery(`(?s)SELECT local_account_id, checked_at, status, latency_ms, latency_limit_ms,.*FROM supplier_account_health_history.*local_account_id = ANY\(\$1\).*checked_at >= \$2.*checked_at < \$3.*ORDER BY local_account_id ASC, checked_at ASC, id ASC`).
		WithArgs(sqlmock.AnyArg(), since, until).
		WillReturnRows(sqlmock.NewRows([]string{"local_account_id", "checked_at", "status", "latency_ms", "latency_limit_ms", "reason", "action", "error_message"}).
			AddRow(21, checkedAt, "healthy", int64(120), 500, "ok", "none", "").
			AddRow(37, checkedAt, "failed", nil, 500, "timeout", "disable", "timeout"))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.GetTrends(context.Background(), []int64{21, 37}, since, until, 50)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Len(t, result[21].Points, 1)
	require.Equal(t, "healthy", result[21].Points[0].Status)
	require.NotNil(t, result[21].Points[0].LatencyMs)
	require.Equal(t, "ok", result[21].Points[0].Reason)
	require.Len(t, result[37].Points, 1)
	require.Nil(t, result[37].Points[0].LatencyMs)
	require.Equal(t, "timeout", result[37].Points[0].ErrorMessage)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthHistoryRepositoryGetUpstreamTrends(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	checkedAt := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	since := checkedAt.Add(-24 * time.Hour)
	until := checkedAt.Add(time.Hour)
	lastSeenAt := checkedAt.Add(-time.Minute)
	mock.ExpectQuery(`(?s)FROM supplier_provider_monitor_bindings binding.*supplier_provider_monitor_targets target.*target\.active = TRUE.*binding\.local_account_id = ANY\(\$1\) AND binding\.match_status = 'active'`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"local_account_id", "id", "provider_id", "name", "monitor_key", "monitor_name",
			"primary_model", "availability_7d", "last_seen_at",
		}).
			AddRow(21, 5, 3, "provider-a", "claude", "Claude 监控", "claude-sonnet-4", 99.5, lastSeenAt).
			AddRow(21, 6, 3, "provider-a", "codex", "Codex 监控", "gpt-5", 97.25, lastSeenAt))
	mock.ExpectQuery(`(?s)NULLIF\(COALESCE\(NULLIF\(sample\.latency_ms, 0\), NULLIF\(sample\.ping_latency_ms, 0\)\), 0\).*FROM supplier_provider_monitor_samples sample.*sample\.checked_at >= \$2.*sample\.checked_at < \$3.*ORDER BY binding\.local_account_id ASC, sample\.checked_at ASC`).
		WithArgs(sqlmock.AnyArg(), since, until).
		WillReturnRows(sqlmock.NewRows([]string{"local_account_id", "checked_at", "status", "latency_ms"}).
			AddRow(21, checkedAt.Add(-2*time.Minute), "healthy", int64(180)).
			AddRow(21, checkedAt.Add(-time.Minute), "unavailable", nil).
			AddRow(37, checkedAt, "failed", int64(900)))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.GetUpstreamTrends(context.Background(), []int64{21, 37}, since, until)

	require.NoError(t, err)
	// 账号 37 没有绑定行，它的样本不该被算进来。
	require.Len(t, result, 1)
	require.Len(t, result[21].Monitors, 2)
	require.Equal(t, int64(5), result[21].Monitors[0].TargetID)
	require.Equal(t, "Claude 监控", result[21].Monitors[0].MonitorName)
	require.Equal(t, "claude-sonnet-4", result[21].Monitors[0].PrimaryModel)
	require.Equal(t, 99.5, result[21].Monitors[0].Availability7D)
	require.NotNil(t, result[21].Monitors[0].LastSeenAt)
	require.Equal(t, lastSeenAt, *result[21].Monitors[0].LastSeenAt)
	require.Equal(t, int64(6), result[21].Monitors[1].TargetID)
	require.Len(t, result[21].Points, 2)
	require.Equal(t, "healthy", result[21].Points[0].Status)
	require.NotNil(t, result[21].Points[0].LatencyMs)
	require.Equal(t, int64(180), *result[21].Points[0].LatencyMs)
	require.Equal(t, "unavailable", result[21].Points[1].Status)
	require.Nil(t, result[21].Points[1].LatencyMs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthHistoryRepositoryGetUpstreamTrendsSkipsSamplesWithoutBindings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	since := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)FROM supplier_provider_monitor_bindings binding`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"local_account_id", "id", "provider_id", "name", "monitor_key", "monitor_name",
			"primary_model", "availability_7d", "last_seen_at",
		}))

	repo := NewSupplierAccountHealthHistoryRepository(db)
	result, err := repo.GetUpstreamTrends(context.Background(), []int64{21}, since, since.Add(time.Hour))

	require.NoError(t, err)
	require.Empty(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}
