package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func newSupplierProviderDataRepoMock(t *testing.T) (*supplierProviderDataRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewSupplierProviderDataRepository(db).(*supplierProviderDataRepository), mock
}

func TestSupplierProviderDataRepositoryListGroupHealthTrendsUsesHealthGuardHistory(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT g\.id AS group_id,.*FROM supplier_automation_runs run.*g\.id = ANY\(\$4\)`).
		WithArgs(
			service.SupplierAutomationTaskAccountHealthGuard,
			now.Add(-90*time.Minute),
			now,
			"{12}",
		).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account_id", "status", "latency_ms", "finished_at", "source"}).
			AddRow(int64(12), int64(98), service.SupplierAccountHealthGuardStatusHealthy, int64(140), now.Add(-time.Minute), service.SupplierProviderGroupHealthTrendSource))

	trends, err := repo.ListGroupHealthTrends(context.Background(), service.SupplierProviderGroupHealthTrendParams{
		GroupIDs:    []int64{12},
		Period:      90 * time.Minute,
		BucketCount: 18,
		Now:         now,
	})

	require.NoError(t, err)
	require.Len(t, trends, 1)
	require.Equal(t, int64(12), trends[0].GroupID)
	require.Equal(t, service.SupplierProviderGroupHealthTrendSource, trends[0].Source)
	require.Equal(t, 100.0, trends[0].Availability)
	require.Len(t, trends[0].Trend, 18)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListLocalGroupHealthTrendsResolvesAccountToEveryBoundLocalGroup(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT account_group\.group_id AS group_id,.*NULLIF\(item->>'local_account_id', ''\)::bigint AS account_id,.*FROM supplier_automation_runs run.*JOIN supplier_provider_accounts account.*JOIN account_groups account_group.*result_detail->'supplier_monitor'->'items'.*account_group\.group_id = ANY\(\$5\)`).
		WithArgs(
			service.SupplierAutomationTaskAccountHealthGuard,
			service.SupplierAutomationTaskMonitorSync,
			now.Add(-24*time.Hour),
			now,
			"{101,202}",
		).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account_id", "status", "latency_ms", "finished_at", "source"}).
			AddRow(int64(101), int64(98), service.SupplierAccountHealthGuardStatusHealthy, int64(140), now.Add(-time.Minute), service.SupplierProviderGroupHealthTrendSource).
			AddRow(int64(202), int64(98), service.SupplierAccountHealthGuardStatusHealthy, int64(140), now.Add(-time.Minute), service.SupplierProviderGroupHealthTrendSource))

	trends, err := repo.ListLocalGroupHealthTrends(context.Background(), service.SupplierProviderGroupHealthTrendParams{
		GroupIDs:    []int64{101, 202},
		Period:      24 * time.Hour,
		BucketCount: 18,
		Now:         now,
	})

	require.NoError(t, err)
	require.Len(t, trends, 2)
	require.Equal(t, int64(101), trends[0].GroupID)
	require.Equal(t, int64(202), trends[1].GroupID)
	require.Equal(t, 100.0, trends[0].Availability)
	require.Equal(t, 100.0, trends[1].Availability)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListLocalGroupHealthTrendsAllHistoryOmitsLowerBound(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT account_group\.group_id AS group_id,.*NULLIF\(item->>'local_account_id', ''\)::bigint AS account_id,.*JOIN account_groups account_group.*WHERE run\.task_code = \$1\s+AND run\.finished_at IS NOT NULL\s+AND run\.finished_at <= \$3\s+AND account_group\.group_id IS NOT NULL\s+AND account_group\.group_id = ANY\(\$4\).*result_detail->'supplier_monitor'->'items'`).
		WithArgs(
			service.SupplierAutomationTaskAccountHealthGuard,
			service.SupplierAutomationTaskMonitorSync,
			now,
			"{101}",
		).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account_id", "status", "latency_ms", "finished_at", "source"}).
			AddRow(int64(101), int64(98), service.SupplierAccountHealthGuardStatusHealthy, int64(140), now.Add(-30*24*time.Hour), service.SupplierProviderGroupHealthTrendSource))

	trends, err := repo.ListLocalGroupHealthTrends(context.Background(), service.SupplierProviderGroupHealthTrendParams{
		GroupIDs:    []int64{101},
		Period:      30 * 24 * time.Hour,
		BucketCount: 18,
		Now:         now,
		AllHistory:  true,
	})

	require.NoError(t, err)
	require.Len(t, trends, 1)
	require.Equal(t, int64(101), trends[0].GroupID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListLocalGroupHealthTrendsIncludesSupplierMonitorHistory(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	checkedAt := now.Add(-30 * time.Second)
	mock.ExpectQuery(`(?s)result_detail->'supplier_monitor'->'items'.*COALESCE\(NULLIF\(item->>'checked_at', ''\)::timestamptz, run\.finished_at\).*account_group\.group_id = ANY\(\$5\)`).
		WithArgs(
			service.SupplierAutomationTaskAccountHealthGuard,
			service.SupplierAutomationTaskMonitorSync,
			now.Add(-10*time.Minute),
			now,
			"{101}",
		).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account_id", "status", "latency_ms", "finished_at", "source"}).
			AddRow(int64(101), int64(98), service.SupplierAccountHealthGuardStatusSlow, int64(10122), checkedAt, service.SupplierProviderGroupHealthTrendMonitorSource))

	trends, err := repo.ListLocalGroupHealthTrends(context.Background(), service.SupplierProviderGroupHealthTrendParams{
		GroupIDs:                 []int64{101},
		Period:                   10 * time.Minute,
		BucketCount:              10,
		Now:                      now,
		PreferRawMonitorTimeline: true,
	})

	require.NoError(t, err)
	require.Len(t, trends, 1)
	require.Equal(t, int64(101), trends[0].GroupID)
	require.Equal(t, int64(10122), trends[0].Latency)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListLocalGroupHealthTrendsTreatsNullTimelineItemsAsEmptyArrays(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)jsonb_array_length\(\s*CASE\s+WHEN jsonb_typeof\(run\.result_detail->'supplier_monitor'->'items'\) = 'array'.*jsonb_array_elements\(\s*CASE\s+WHEN jsonb_typeof\(run\.result_detail->'account_health_guard'->'items'\) = 'array'.*jsonb_array_elements\(\s*CASE\s+WHEN jsonb_typeof\(item->'sources'\) = 'array'`).
		WithArgs(
			service.SupplierAutomationTaskAccountHealthGuard,
			service.SupplierAutomationTaskMonitorSync,
			now.Add(-24*time.Hour),
			now,
			"{101}",
		).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account_id", "status", "latency_ms", "finished_at", "source"}))

	trends, err := repo.ListLocalGroupHealthTrends(context.Background(), service.SupplierProviderGroupHealthTrendParams{
		GroupIDs:    []int64{101},
		Period:      24 * time.Hour,
		BucketCount: 18,
		Now:         now,
	})

	require.NoError(t, err)
	require.Empty(t, trends)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListLocalGroupHealthTrendsIncludesStructuredMonitorBindings(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	checkedAt := now.Add(-time.Minute)
	mock.ExpectQuery(`(?s)supplier_provider_monitor_samples sample.*supplier_provider_monitor_bindings binding.*account_group\.group_id = ANY\(\$5\)`).
		WithArgs(
			service.SupplierAutomationTaskAccountHealthGuard,
			service.SupplierAutomationTaskMonitorSync,
			now.Add(-10*time.Minute),
			now,
			"{81}",
		).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account_id", "status", "latency_ms", "finished_at", "source"}).
			AddRow(int64(81), int64(777), service.SupplierAccountHealthGuardStatusHealthy, int64(880), checkedAt, service.SupplierProviderGroupHealthTrendMonitorSource))

	trends, err := repo.ListLocalGroupHealthTrends(context.Background(), service.SupplierProviderGroupHealthTrendParams{
		GroupIDs:                 []int64{81},
		Period:                   10 * time.Minute,
		BucketCount:              10,
		Now:                      now,
		PreferRawMonitorTimeline: true,
	})

	require.NoError(t, err)
	require.Len(t, trends, 1)
	require.Equal(t, int64(81), trends[0].GroupID)
	require.Equal(t, int64(880), trends[0].Latency)
	require.Equal(t, service.SupplierProviderGroupHealthTrendMonitorSource, trends[0].Source)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositorySaveMonitorSnapshotReturnsExplicitBindings(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	seenAt := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	checkedAt := seenAt.Add(-time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO supplier_provider_monitor_targets.*ON CONFLICT \(provider_id, monitor_key\).*RETURNING id`).
		WithArgs(int64(7), "2", "Plus-稳定", "sub2api", "gpt-5", 99.5, seenAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(31)))
	mock.ExpectExec(`(?s)INSERT INTO supplier_provider_monitor_samples.*ON CONFLICT \(monitor_target_id, checked_at\)`).
		WithArgs(int64(31), checkedAt, service.SupplierAccountHealthGuardStatusHealthy, "operational", int64(120), int64(30), 99.5).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`(?s)SELECT t\.id, t\.provider_id, t\.monitor_key, t\.monitor_name,.*account_groups account_group.*FROM supplier_provider_monitor_targets t.*supplier_provider_monitor_bindings b`).
		WithArgs(int64(7), `{"2"}`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_id", "monitor_key", "monitor_name", "local_account_id", "local_account_name", "binding_groups"}).
			AddRow(int64(31), int64(7), "2", "Plus-稳定", int64(777), "皓悦-福利-Codex高并发", []byte(`[{"id":81,"name":"AAA","platform":"openai","rate_multiplier":1,"subscription_type":"plus"}]`)))

	bindings, err := repo.SaveMonitorSnapshot(context.Background(), int64(7), []service.SupplierProviderMonitorItem{{
		Key:            "2",
		Name:           "Plus-稳定",
		Provider:       "sub2api",
		PrimaryModel:   "gpt-5",
		Availability7D: ptr(99.5),
		Timeline: []service.SupplierProviderMonitorPoint{{
			Status:        "operational",
			LatencyMS:     120,
			PingLatencyMS: 30,
			CheckedAt:     checkedAt,
		}},
	}}, seenAt)

	require.NoError(t, err)
	require.Len(t, bindings, 1)
	require.Equal(t, int64(31), bindings[0].MonitorTargetID)
	require.Equal(t, int64(777), bindings[0].LocalAccountID)
	require.Equal(t, "皓悦-福利-Codex高并发", bindings[0].LocalAccountName)
	require.Equal(t, []service.SupplierProviderAccountBindingGroup{{ID: 81, Name: "AAA", Platform: "openai", RateMultiplier: 1, SubscriptionType: "plus"}}, bindings[0].BindingGroups)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListMonitorTargetsReturnsBindingAccountAndGroups(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	active := true
	lastSeenAt := time.Date(2026, 8, 12, 11, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM supplier_provider_monitor_targets t WHERE TRUE AND t\.provider_id = \$1 AND t\.active = \$2 AND \(t\.monitor_key ILIKE \$3 OR t\.monitor_name ILIKE \$3 OR t\.primary_model ILIKE \$3\)`).
		WithArgs(int64(7), true, "%Plus%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)SELECT t\.id, t\.provider_id, p\.name, t\.monitor_key, t\.monitor_name, t\.monitor_provider,.*FROM supplier_provider_monitor_targets t.*JOIN supplier_providers p ON p\.id = t\.provider_id.*supplier_provider_monitor_bindings b.*ORDER BY p\.name ASC, CASE WHEN local_account\.id IS NULL THEN 0 ELSE 1 END ASC, t\.monitor_name ASC, t\.id ASC`).
		WithArgs(int64(7), true, "%Plus%", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_id", "provider_name", "monitor_key", "monitor_name", "monitor_provider", "primary_model", "availability_7d", "active", "last_seen_at", "local_account_id", "local_account_name", "binding_groups"}).
			AddRow(int64(31), int64(7), "皓悦", "2", "Plus-稳定", "sub2api", "gpt-5", 99.5, true, lastSeenAt, int64(777), "皓悦-福利-Codex高并发", []byte(`[{"id":81,"name":"AAA","platform":"openai","rate_multiplier":1,"subscription_type":"plus"}]`)))

	result, err := repo.ListMonitorTargets(context.Background(), service.SupplierProviderMonitorTargetListParams{
		ProviderID: 7,
		Active:     &active,
		Search:     " Plus ",
		Page:       2,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(777), result.Items[0].LocalAccountID)
	require.Equal(t, "皓悦", result.Items[0].ProviderName)
	require.Equal(t, "皓悦-福利-Codex高并发", result.Items[0].LocalAccountName)
	require.Equal(t, ptr(99.5), result.Items[0].Availability7D)
	require.Equal(t, []service.SupplierProviderAccountBindingGroup{{ID: 81, Name: "AAA", Platform: "openai", RateMultiplier: 1, SubscriptionType: "plus"}}, result.Items[0].BindingGroups)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListMonitorTargetsKeepsNullAvailabilityDistinctFromZero(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	lastSeenAt := time.Date(2026, 8, 12, 11, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM supplier_provider_monitor_targets t WHERE TRUE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
	mock.ExpectQuery(`(?s)SELECT t\.id, t\.provider_id, p\.name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_id", "provider_name", "monitor_key", "monitor_name", "monitor_provider", "primary_model", "availability_7d", "active", "last_seen_at", "local_account_id", "local_account_name", "binding_groups"}).
			AddRow(int64(31), int64(7), "皓悦", "2", "无上报", "sub2api", "gpt-5", nil, true, lastSeenAt, int64(0), "", []byte(`[]`)).
			AddRow(int64(32), int64(7), "皓悦", "3", "真实全挂", "sub2api", "gpt-5", 0.0, true, lastSeenAt, int64(0), "", []byte(`[]`)))

	result, err := repo.ListMonitorTargets(context.Background(), service.SupplierProviderMonitorTargetListParams{})

	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	require.Nil(t, result.Items[0].Availability7D)
	require.Equal(t, ptr(0.0), result.Items[1].Availability7D)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListMonitorTargetsAppliesWhitelistedSort(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM supplier_provider_monitor_targets t WHERE TRUE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`(?s)ORDER BY t\.availability_7d DESC NULLS LAST, p\.name ASC, CASE WHEN local_account\.id IS NULL THEN 0 ELSE 1 END ASC, t\.monitor_name ASC, t\.id ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_id", "provider_name", "monitor_key", "monitor_name", "monitor_provider", "primary_model", "availability_7d", "active", "last_seen_at", "local_account_id", "local_account_name", "binding_groups"}))

	_, err := repo.ListMonitorTargets(context.Background(), service.SupplierProviderMonitorTargetListParams{
		Sort:  service.SupplierProviderMonitorTargetSortAvailability,
		Order: "desc",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListMonitorTargetsRejectsUnknownSort(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM supplier_provider_monitor_targets t WHERE TRUE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`(?s)ORDER BY p\.name ASC, CASE WHEN local_account\.id IS NULL THEN 0 ELSE 1 END ASC, t\.monitor_name ASC, t\.id ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_id", "provider_name", "monitor_key", "monitor_name", "monitor_provider", "primary_model", "availability_7d", "active", "last_seen_at", "local_account_id", "local_account_name", "binding_groups"}))

	_, err := repo.ListMonitorTargets(context.Background(), service.SupplierProviderMonitorTargetListParams{
		Sort:  "t.id; DROP TABLE accounts",
		Order: "desc",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListBindableLocalAccountsFiltersByProviderAndPlatform(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)

	mock.ExpectQuery(`(?s)WITH account_sources AS .*ARRAY_AGG\(DISTINCT a\.provider_id\) AS provider_ids.*FROM accounts local_account.*JOIN supplier_provider_accounts a ON TRUE.*LEFT JOIN account_sources src ON src\.local_account_id = local_account\.id.*WHERE local_account\.deleted_at IS NULL AND local_account\.status = 'active' AND src\.provider_ids @> ARRAY\[\$1\]::BIGINT\[\] AND COALESCE\(NULLIF\(platform_override\.platform, ''\), NULLIF\(local_account\.platform, ''\), ''\) = \$2 AND \(local_account\.name ILIKE \$3 OR local_account\.id::text ILIKE \$3\).*ORDER BY LOWER\(local_account\.name\) ASC, local_account\.id ASC.*LIMIT \$4 OFFSET \$5`).
		WithArgs(int64(7), "openai", "%Codex%", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "platform", "provider_name", "groups", "total_count"}).
			AddRow(int64(777), "皓悦-福利-Codex高并发", "openai", "皓悦", []byte(`[{"id":81,"name":"AAA","platform":"openai","rate_multiplier":1,"subscription_type":"plus"}]`), int64(3)))

	result, err := repo.ListBindableLocalAccounts(context.Background(), service.SupplierBindableLocalAccountListParams{
		ProviderID: 7,
		Platform:   " OpenAI ",
		Search:     " Codex ",
		Page:       2,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(3), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(777), result.Items[0].ID)
	require.Equal(t, "皓悦", result.Items[0].ProviderName)
	require.Equal(t, []service.SupplierProviderAccountBindingGroup{{ID: 81, Name: "AAA", Platform: "openai", RateMultiplier: 1, SubscriptionType: "plus"}}, result.Items[0].Groups)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListBindableLocalAccountsKeepsUnmatchedAccounts(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)

	mock.ExpectQuery(`(?s)WITH account_sources AS .*WHERE local_account\.deleted_at IS NULL AND local_account\.status = 'active' ORDER BY`).
		WithArgs(50, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "platform", "provider_name", "groups", "total_count"}).
			AddRow(int64(9), "独立账号", "anthropic", "", []byte(`[]`), int64(1)))

	result, err := repo.ListBindableLocalAccounts(context.Background(), service.SupplierBindableLocalAccountListParams{PageSize: 500})

	require.NoError(t, err)
	require.Equal(t, 1, result.Page)
	require.Equal(t, 50, result.PageSize)
	require.Len(t, result.Items, 1)
	require.Empty(t, result.Items[0].ProviderName)
	require.Empty(t, result.Items[0].Groups)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryBindMonitorTargetUpsertsManualBinding(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectExec(`(?s)WITH valid_binding AS .*INSERT INTO supplier_provider_monitor_bindings .*ON CONFLICT \(provider_id, monitor_target_id\) DO UPDATE SET`).
		WithArgs(int64(31), int64(777)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.BindMonitorTarget(context.Background(), 31, 777)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryApplyMonitorAutoMatchWritesAutoBinding(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectExec(`(?s)WITH valid_binding AS .*INSERT INTO supplier_provider_monitor_bindings .*ON CONFLICT \(provider_id, monitor_target_id\) DO UPDATE SET`).
		WithArgs(int64(31), int64(777)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.ApplyMonitorAutoMatch(context.Background(), 31, 777)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryApplyMonitorAutoMatchRejectsInvalidIDs(t *testing.T) {
	repo, _ := newSupplierProviderDataRepoMock(t)

	err := repo.ApplyMonitorAutoMatch(context.Background(), 0, 777)
	require.ErrorIs(t, err, service.ErrSupplierProviderMonitorBindingInvalid)

	err = repo.ApplyMonitorAutoMatch(context.Background(), 31, 0)
	require.ErrorIs(t, err, service.ErrSupplierProviderMonitorBindingInvalid)
}

func TestSupplierProviderDataRepositoryUnbindMonitorTargetMarksBindingInactive(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectExec(`(?s)UPDATE supplier_provider_monitor_bindings\s+SET match_status = 'inactive', updated_at = NOW\(\)\s+WHERE monitor_target_id = \$1\s+AND match_status = 'active'`).
		WithArgs(int64(31)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UnbindMonitorTarget(context.Background(), 31)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryUpdateAccountRateSnapshotOnlyTouchesRateFields(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(`
UPDATE supplier_provider_accounts
SET rate_multiplier=$3, rate_sync_status='success', rate_sync_message='',
    last_rate_sync_at=$4, updated_at=$4
WHERE provider_id=$1 AND upstream_account_key=$2
RETURNING id`)).
		WithArgs(int64(7), "upstream-key", 1.25, now).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(99)))

	updated, err := repo.UpdateAccountRateSnapshot(context.Background(), 7, "upstream-key", 1.25, now)

	require.NoError(t, err)
	require.True(t, updated)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryUpdateAccountRateSnapshotSkipsUnknownKey(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("UPDATE supplier_provider_accounts").
		WithArgs(int64(7), "missing", 2.0, now).
		WillReturnError(sql.ErrNoRows)

	updated, err := repo.UpdateAccountRateSnapshot(context.Background(), 7, "missing", 2, now)

	require.NoError(t, err)
	require.False(t, updated)
	require.NoError(t, mock.ExpectationsWereMet())
}

type supplierProviderNonNilArg struct{}

func (supplierProviderNonNilArg) Match(value driver.Value) bool {
	return value != nil
}

var supplierProviderAccountListColumns = []string{
	"id", "provider_id", "provider_name", "upstream_account_key", "name", "status",
	"group_key", "group_name", "platform", "group_status", "rate_multiplier", "raw_status", "active",
	"last_seen_at", "inactive_at",
	"local_account_match_status", "local_account_match_count",
	"local_account_id", "local_account_name", "local_account_platform", "local_account_type", "platform_override", "effective_platform", "local_account_priority",
	"local_account_status", "local_account_schedulable",
	"local_account_last_test_status", "local_account_last_tested_at", "local_account_last_test_error", "local_account_health_guard_last_checked_at",
	"binding_groups",
	"supplier_current_balance", "supplier_today_cost", "group_record_id", "group_record_delete_eligible",
}

// supplierProviderAccountListQueryContractPattern 约束列表查询的字段顺序与匹配关联，不能替代真实 PostgreSQL 行为测试。
func supplierProviderAccountListQueryContractPattern(whereSQL, limitPlaceholder, offsetPlaceholder string) string {
	querySQL := `
SELECT a.id, a.provider_id, p.name AS provider_name, a.upstream_account_key, a.name, a.status,
       a.group_key, a.group_name,
       COALESCE((
         SELECT local_group.platform
         FROM supplier_provider_groups mapped_group
         JOIN groups local_group ON local_group.id = mapped_group.local_group_id AND local_group.deleted_at IS NULL
         WHERE mapped_group.provider_id = a.provider_id
           AND mapped_group.upstream_group_key = a.group_key
         LIMIT 1
       ), '') AS platform,
       CASE
         WHEN NULLIF(TRIM(a.group_key), '') IS NULL THEN ''
         WHEN a.active = FALSE AND LOWER(a.status) = 'deleted' THEN ''
         WHEN EXISTS (
           SELECT 1 FROM supplier_provider_groups g
           WHERE g.provider_id = a.provider_id
             AND g.upstream_group_key = a.group_key
             AND g.active = TRUE
         ) THEN 'active'
         WHEN EXISTS (
           SELECT 1 FROM supplier_provider_groups g
           WHERE g.provider_id = a.provider_id
             AND g.upstream_group_key = a.group_key
         ) THEN 'inactive'
         ELSE 'missing'
       END AS group_status,
       a.rate_multiplier, a.raw_status, a.active,
       a.last_seen_at, a.inactive_at,
       CASE
         WHEN local_match.match_count = 0 THEN 'unmatched'
         WHEN local_match.match_count = 1 THEN 'matched'
         ELSE 'conflict'
       END AS local_account_match_status,
       local_match.match_count AS local_account_match_count,
       matched_account.id AS local_account_id,
       COALESCE(matched_account.name, '') AS local_account_name,
       COALESCE(matched_account.platform, '') AS local_account_platform,
       COALESCE(matched_account.type, '') AS local_account_type,
       COALESCE(platform_override.platform, '') AS platform_override,
       COALESCE(NULLIF(platform_override.platform, ''), NULLIF(matched_account.platform, ''), COALESCE((
         SELECT local_group.platform
         FROM supplier_provider_groups mapped_group
         JOIN groups local_group ON local_group.id = mapped_group.local_group_id AND local_group.deleted_at IS NULL
         WHERE mapped_group.provider_id = a.provider_id
           AND mapped_group.upstream_group_key = a.group_key
         LIMIT 1
       ), '')) AS effective_platform,
       matched_account.priority AS local_account_priority,
       COALESCE(matched_account.status, '') AS local_account_status,
       matched_account.schedulable AS local_account_schedulable,
       COALESCE(matched_account.extra->>'last_test_status', '') AS local_account_last_test_status,
       COALESCE(matched_account.extra->>'last_tested_at', '') AS local_account_last_tested_at,
       COALESCE(matched_account.extra->>'last_test_error', '') AS local_account_last_test_error,
       COALESCE(matched_account.extra->>'supplier_health_guard_last_checked_at', '') AS local_account_health_guard_last_checked_at,
       COALESCE((
         SELECT jsonb_agg(
           jsonb_build_object(
             'id', local_group.id,
             'name', local_group.name,
             'platform', local_group.platform,
             'rate_multiplier', local_group.rate_multiplier,
             'subscription_type', local_group.subscription_type
           )
           ORDER BY LOWER(local_group.name), local_group.id
         )
         FROM account_groups account_group
         JOIN groups local_group
           ON local_group.id = account_group.group_id
          AND local_group.deleted_at IS NULL
         WHERE account_group.account_id = matched_account.id
       ), '[]'::jsonb) AS binding_groups,
       COALESCE(runtime.current_balance, 0) AS supplier_current_balance,
       COALESCE(runtime.today_cost, 0) AS supplier_today_cost,
       inactive_group_record.id AS group_record_id,
       COALESCE(
         inactive_group_record.id IS NOT NULL
         AND inactive_group_record.rate_guard_selected = FALSE,
         FALSE
       ) AS group_record_delete_eligible
FROM supplier_provider_accounts a
JOIN supplier_providers p ON p.id = a.provider_id
LEFT JOIN supplier_provider_runtime_stats runtime ON runtime.provider_id = p.id
LEFT JOIN LATERAL (
  SELECT g.id, g.provider_id, g.upstream_group_key, g.local_group_id, g.rate_guard_selected
  FROM supplier_provider_groups g
  WHERE g.provider_id = a.provider_id
    AND g.upstream_group_key = a.group_key
    AND g.active = FALSE
  ORDER BY g.id DESC
  LIMIT 1
) inactive_group_record ON TRUE
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS match_count,
         MIN(local_account.id) AS local_account_id
  FROM accounts local_account
  WHERE local_account.deleted_at IS NULL
    AND ` + supplierProviderLocalAccountMatchCondition("local_account.name", "a.name") + `) local_match ON TRUE
LEFT JOIN accounts matched_account
  ON matched_account.id = local_match.local_account_id
 AND local_match.match_count = 1
LEFT JOIN supplier_local_account_platform_overrides platform_override
  ON platform_override.local_account_id = matched_account.id
WHERE ` + whereSQL + `
ORDER BY a.active DESC, a.last_seen_at DESC, a.id ASC LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder

	normalizedQuery := strings.Join(strings.Fields(querySQL), " ")
	normalizedMatchCondition := strings.Join(strings.Fields(supplierProviderLocalAccountMatchCondition("local_account.name", "a.name")), " ")
	quotedQuery := regexp.QuoteMeta(normalizedQuery)
	quotedMatchCondition := regexp.QuoteMeta(normalizedMatchCondition)
	return "^" + strings.Replace(quotedQuery, quotedMatchCondition, `(?s:.*?)`, 1) + "$"
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
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_accounts SET active = FALSE, status = 'deleted', raw_status = 'deleted'")).
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
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT upstream_group_key, name, rate_multiplier, active
FROM supplier_provider_groups
WHERE provider_id = $1
FOR UPDATE`)).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"upstream_group_key", "name", "rate_multiplier", "active"}).
			AddRow("group-1", "VIP", 2.5, true).
			AddRow("group-removed", "Old", 1.5, true))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_groups")).
		WithArgs(int64(42), "group-1", "VIP", 2.5, "active", seenAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET active = FALSE")).
		WithArgs(int64(42), sqlmock.AnyArg(), seenAt).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	result, err := repo.ReplaceGroups(context.Background(), 42, []service.SupplierProviderRemoteGroup{
		{Key: "group-1", Name: "VIP", RateMultiplier: 2.5, RawStatus: "active"},
	}, seenAt)

	require.NoError(t, err)
	require.Equal(t, 1, result.Counts.CheckedCount)
	require.Equal(t, 2, result.Counts.SkippedCount)
	require.Len(t, result.Changes.Removed, 1)
	require.Equal(t, "group-removed", result.Changes.Removed[0].UpstreamKey)
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

	// 历史日期：只写入快照与 daily_stats，不覆盖 runtime.today_cost
	historicalDay := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at)")).
		WithArgs(int64(42), supplierProviderNonNilArg{}, 45.625, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_daily_stats")).
		WithArgs(int64(42), sqlmock.AnyArg(), 45.625, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repo.UpdateCost(context.Background(), 42, 45.625, historicalDay))

	// 今天：同步更新 runtime.today_cost
	today := time.Now()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_runtime_stats SET today_cost")).
		WithArgs(int64(42), 12.5, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at)")).
		WithArgs(int64(42), supplierProviderNonNilArg{}, 12.5, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_daily_stats")).
		WithArgs(int64(42), sqlmock.AnyArg(), 12.5, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repo.UpdateCost(context.Background(), 42, 12.5, today))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetCostFallbackBalances(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	statDay := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT s\.current_balance,.*FROM supplier_provider_runtime_stats s.*WHERE s\.provider_id = \$1`).
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"current_balance", "day_start_balance"}).AddRow(80.5, 100.0))

	bal, ok, err := repo.GetCostFallbackBalances(context.Background(), 42, statDay)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, service.SupplierProviderCostFallbackBalance{CurrentBalance: 80.5, DayStartBalance: 100.0}, bal)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetCostFallbackBalancesMissingRuntimeStats(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	statDay := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT s\.current_balance,.*FROM supplier_provider_runtime_stats s.*WHERE s\.provider_id = \$1`).
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	bal, ok, err := repo.GetCostFallbackBalances(context.Background(), 42, statDay)
	require.NoError(t, err)
	require.False(t, ok)
	require.Zero(t, bal)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetCostFallbackBalancesNegativeBalance(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	statDay := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT s\.current_balance,.*FROM supplier_provider_runtime_stats s.*WHERE s\.provider_id = \$1`).
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"current_balance", "day_start_balance"}).AddRow(-1.0, 100.0))

	bal, ok, err := repo.GetCostFallbackBalances(context.Background(), 42, statDay)
	require.NoError(t, err)
	require.False(t, ok)
	require.Zero(t, bal)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetCostFallbackBalancesQueryError(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	statDay := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT s\.current_balance,.*FROM supplier_provider_runtime_stats s.*WHERE s\.provider_id = \$1`).
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnError(errors.New("db down"))

	_, ok, err := repo.GetCostFallbackBalances(context.Background(), 42, statDay)
	require.Error(t, err)
	require.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryUpdateCostDetailedUpsertsRawAndWarning(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	historicalDay := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	raw := 45.625
	warning := "上游成本 45.63 与本地成本 5.00 偏差 89%，已按本地成本展示"

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at)")).
		WithArgs(int64(42), supplierProviderNonNilArg{}, 5.0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_daily_stats")).
		WithArgs(int64(42), sqlmock.AnyArg(), 5.0, 45.625, warning).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.UpdateCostDetailed(context.Background(), 42, 5.0, &raw, &warning, historicalDay))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryUpdateCostDetailedNilsRawAndWarning(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	historicalDay := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_metric_snapshots (provider_id, current_balance, today_cost, captured_at)")).
		WithArgs(int64(42), supplierProviderNonNilArg{}, 5.0, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_daily_stats")).
		WithArgs(int64(42), sqlmock.AnyArg(), 5.0, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.UpdateCostDetailed(context.Background(), 42, 5.0, nil, nil, historicalDay))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetLocalCostForDay(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	day := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)WITH provider_account_matches.*COUNT\(DISTINCT u\.local_account_id\).*WHERE ul\.created_at >= \$2.*ul\.created_at < \$3`).
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"local_cost", "matched_count"}).AddRow(15.25, 1))

	local, ok, err := repo.GetLocalCostForDay(context.Background(), 42, day)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 15.25, local)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetLocalCostForDayNoMatch(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	day := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)WITH provider_account_matches.*COUNT\(DISTINCT u\.local_account_id\).*WHERE ul\.created_at >= \$2.*ul\.created_at < \$3`).
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"local_cost", "matched_count"}).AddRow(0.0, 0))

	local, ok, err := repo.GetLocalCostForDay(context.Background(), 42, day)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, 0.0, local)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetBalanceDeltaForDay(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	day := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT (SELECT current_balance FROM supplier_provider_metric_snapshots")).
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"first_balance", "last_balance"}).AddRow(100.50, 85.25))

	delta, ok, err := repo.GetBalanceDeltaForDay(context.Background(), 42, day)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 15.25, delta)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetBalanceDeltaForDayNoData(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	day := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT (SELECT current_balance FROM supplier_provider_metric_snapshots")).
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"first_balance", "last_balance"}).AddRow(nil, nil))

	delta, ok, err := repo.GetBalanceDeltaForDay(context.Background(), 42, day)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, 0.0, delta)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetBalanceDeltaForDayNegativeDelta(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	day := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT (SELECT current_balance FROM supplier_provider_metric_snapshots")).
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"first_balance", "last_balance"}).AddRow(85.25, 100.50))

	delta, ok, err := repo.GetBalanceDeltaForDay(context.Background(), 42, day)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, 0.0, delta)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetBalanceDeltaForDayQueryError(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	day := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT (SELECT current_balance FROM supplier_provider_metric_snapshots")).
		WithArgs(int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("database error"))

	_, ok, err := repo.GetBalanceDeltaForDay(context.Background(), 42, day)
	require.Error(t, err)
	require.False(t, ok)
	require.Contains(t, err.Error(), "query supplier balance delta for day")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListAccountsPaginates(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	active := true
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_provider_accounts a")).
		WithArgs(int64(42), active, "%pri%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(supplierProviderAccountListQueryContractPattern(
		"p.deleted_at IS NULL AND a.provider_id = $1 AND a.active = $2 AND (a.name ILIKE $3 OR a.upstream_account_key ILIKE $3)",
		"$4",
		"$5",
	)).
		WithArgs(int64(42), active, "%pri%", 20, 20).
		WillReturnRows(sqlmock.NewRows(supplierProviderAccountListColumns).AddRow(
			int64(7), int64(42), "Supplier A", "account-1", "Primary", "active", "group-1", "VIP", "openai", "active", 2.5, "active", true, now, nil,
			"matched", 1, int64(101), "prefix-key-1", "anthropic", "apikey", "", "anthropic", 80, "active", true, "success", "2026-07-16T09:30:00Z", "upstream authentication failed", "2026-08-27T08:30:00Z",
			`[{"id":202,"name":"Claude 订阅","platform":"anthropic","rate_multiplier":2,"subscription_type":"subscription"},{"id":201,"name":"OpenAI 专线","platform":"openai","rate_multiplier":1.5,"subscription_type":"standard"}]`,
			12.5, 3.25, nil, false,
		))

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
	require.Equal(t, "openai", result.Items[0].Platform)
	require.Equal(t, "active", result.Items[0].GroupStatus)
	require.Equal(t, "matched", result.Items[0].LocalAccountMatchStatus)
	require.Equal(t, 1, result.Items[0].LocalAccountMatchCount)
	require.NotNil(t, result.Items[0].LocalAccountID)
	require.Equal(t, int64(101), *result.Items[0].LocalAccountID)
	require.Equal(t, "prefix-key-1", result.Items[0].LocalAccountName)
	require.Equal(t, "anthropic", result.Items[0].LocalAccountPlatform)
	require.NotNil(t, result.Items[0].LocalAccountPriority)
	require.Equal(t, 80, *result.Items[0].LocalAccountPriority)
	require.Equal(t, "active", result.Items[0].LocalAccountStatus)
	require.NotNil(t, result.Items[0].LocalAccountSchedulable)
	require.True(t, *result.Items[0].LocalAccountSchedulable)
	require.Equal(t, "success", result.Items[0].LocalAccountLastTestStatus)
	require.Equal(t, "2026-07-16T09:30:00Z", result.Items[0].LocalAccountLastTestedAt)
	require.Equal(t, "upstream authentication failed", result.Items[0].LocalAccountLastTestError)
	require.Equal(t, "2026-08-27T08:30:00Z", result.Items[0].LocalAccountHealthGuardLastCheckedAt)
	require.Equal(t, []service.SupplierProviderAccountBindingGroup{
		{ID: 202, Name: "Claude 订阅", Platform: "anthropic", RateMultiplier: 2, SubscriptionType: "subscription"},
		{ID: 201, Name: "OpenAI 专线", Platform: "openai", RateMultiplier: 1.5, SubscriptionType: "standard"},
	}, result.Items[0].BindingGroups)
	require.Equal(t, 12.5, result.Items[0].SupplierCurrentBalance)
	require.Equal(t, 3.25, result.Items[0].SupplierTodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListAccountsExposesInactiveGroupDeleteMetadata(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	columns := append([]string{}, supplierProviderAccountListColumns...)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_provider_accounts a")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)inactive_group_record\.id AS group_record_id\s*,\s*COALESCE\(\s*inactive_group_record\.id IS NOT NULL\s+AND inactive_group_record\.rate_guard_selected = FALSE\s*,\s*FALSE\s*\)\s+AS group_record_delete_eligible`).
		WithArgs(int64(42), 20, 0).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			int64(7), int64(42), "Supplier A", "account-1", "Primary", "active", "group-1", "VIP", "", "inactive",
			2.5, "active", true, now, nil,
			"unmatched", 0, nil, "", "", "", "", "", nil, "", nil, "", "", "", "", `[]`, 12.5, 3.25,
			int64(88), true,
		))

	result, err := repo.ListAccounts(context.Background(), service.SupplierProviderDataListParams{
		ProviderID: 42,
		Page:       1,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.NotNil(t, result.Items[0].GroupRecordID)
	require.Equal(t, int64(88), *result.Items[0].GroupRecordID)
	require.True(t, result.Items[0].GroupRecordDeleteEligible)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderLocalAccountMatchConditionSupportsReorderedChineseAndLatinAccountName(t *testing.T) {
	condition := supplierProviderLocalAccountMatchCondition("local_account.name", "a.name")

	require.Contains(t, condition, "^([^a-z0-9]+)([a-z]+)([0-9]*)$")
	require.Contains(t, condition, "^([a-z]+)([^a-z0-9]+)([0-9]*)$")
	require.Contains(t, condition, `\2\1\3`)
}
func TestSupplierProviderDataRepositoryListAccountsSQLContractMapsUnmatchedAndConflictRows(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_provider_accounts a")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
	mock.ExpectQuery(supplierProviderAccountListQueryContractPattern(
		"p.deleted_at IS NULL AND a.provider_id = $1",
		"$2",
		"$3",
	)).
		WithArgs(int64(42), 20, 0).
		WillReturnRows(sqlmock.NewRows(supplierProviderAccountListColumns).
			AddRow(
				int64(7), int64(42), "Supplier A", "missing-key", "Missing", "active", "group-1", "VIP", "openai", "active", 2.5, "active", true, now, nil,
				"unmatched", 0, nil, "", "", "", "", "", nil, "", nil, "", "", "", "", `[]`, 12.5, 3.25, nil, false,
			).
			AddRow(
				int64(8), int64(42), "Supplier A", "duplicate-key", "Duplicate", "active", "group-2", "Standard", "openai", "active", 1.5, "active", true, now, nil,
				"conflict", 2, nil, "", "", "", "", "", nil, "", nil, "", "", "", "", `[]`, 12.5, 3.25, nil, false,
			))

	result, err := repo.ListAccounts(context.Background(), service.SupplierProviderDataListParams{
		ProviderID: 42,
		Page:       1,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(2), result.Total)
	require.Len(t, result.Items, 2)

	unmatched := result.Items[0]
	require.Equal(t, "unmatched", unmatched.LocalAccountMatchStatus)
	require.Zero(t, unmatched.LocalAccountMatchCount)
	require.Nil(t, unmatched.LocalAccountID)
	require.Empty(t, unmatched.LocalAccountName)
	require.Nil(t, unmatched.LocalAccountPriority)
	require.Empty(t, unmatched.LocalAccountStatus)
	require.Nil(t, unmatched.LocalAccountSchedulable)
	require.Empty(t, unmatched.LocalAccountLastTestStatus)
	require.Empty(t, unmatched.LocalAccountLastTestedAt)
	require.Empty(t, unmatched.LocalAccountLastTestError)
	require.Empty(t, unmatched.LocalAccountHealthGuardLastCheckedAt)
	require.Equal(t, "active", unmatched.GroupStatus)

	conflict := result.Items[1]
	require.Equal(t, "conflict", conflict.LocalAccountMatchStatus)
	require.Equal(t, 2, conflict.LocalAccountMatchCount)
	require.Nil(t, conflict.LocalAccountID)
	require.Empty(t, conflict.LocalAccountName)
	require.Nil(t, conflict.LocalAccountPriority)
	require.Empty(t, conflict.LocalAccountStatus)
	require.Nil(t, conflict.LocalAccountSchedulable)
	require.Empty(t, conflict.LocalAccountLastTestStatus)
	require.Empty(t, conflict.LocalAccountLastTestedAt)
	require.Empty(t, conflict.LocalAccountLastTestError)
	require.Empty(t, conflict.LocalAccountHealthGuardLastCheckedAt)
	require.Equal(t, "active", conflict.GroupStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListAccountsExposesLocalAccountTypeForDuplication(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	columns := []string{
		"id", "provider_id", "provider_name", "upstream_account_key", "name", "status",
		"group_key", "group_name", "platform", "group_status", "rate_multiplier", "raw_status", "active",
		"last_seen_at", "inactive_at",
		"local_account_match_status", "local_account_match_count",
		"local_account_id", "local_account_name", "local_account_platform", "local_account_type", "platform_override", "effective_platform", "local_account_priority",
		"local_account_status", "local_account_schedulable",
		"local_account_last_test_status", "local_account_last_tested_at", "local_account_last_test_error", "local_account_health_guard_last_checked_at",
		"binding_groups",
		"supplier_current_balance", "supplier_today_cost", "group_record_id", "group_record_delete_eligible",
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_provider_accounts a")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)COALESCE\(matched_account\.type, ''\) AS local_account_type.*COALESCE\(p\.name, ''\) \|\| '-' \|\| a\.name`).
		WithArgs(int64(42), 20, 0).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			int64(7), int64(42), "Supplier A", "upstream-key", "Primary", "active", "group-1", "VIP", "openai", "active", 1.5, "active", true, now, nil,
			"matched", 1, int64(101), "local-account", "openai", "apikey", "", "openai", 80, "active", true, "success", "2026-07-28T09:30:00Z", "", "", `[]`, 12.5, 3.25, nil, false,
		))

	result, err := repo.ListAccounts(context.Background(), service.SupplierProviderDataListParams{
		ProviderID: 42,
		Page:       1,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	payload, err := json.Marshal(result.Items[0])
	require.NoError(t, err)
	var account map[string]any
	require.NoError(t, json.Unmarshal(payload, &account))
	require.Equal(t, "apikey", account["local_account_type"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListAccountsUsesBusinessPlatformOverride(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	columns := supplierProviderAccountListColumns

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_provider_accounts a")).
		WithArgs(int64(42), "grok").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)COALESCE\(NULLIF\(platform_override\.platform, ''\), NULLIF\(matched_account\.platform, ''\).*AS effective_platform.*LEFT JOIN supplier_local_account_platform_overrides platform_override ON platform_override\.local_account_id = matched_account\.id`).
		WithArgs(int64(42), "grok", 20, 0).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			int64(7), int64(42), "Supplier A", "upstream-key", "Primary", "active", "group-1", "VIP", "openai", "active", 1.5, "active", true, now, nil,
			"matched", 1, int64(101), "local-account", "openai", "apikey", "grok", "grok", 80, "active", true, "success", "2026-07-27T11:30:00Z", "", "", `[]`, 12.5, 3.25, nil, false,
		))

	result, err := repo.ListAccounts(context.Background(), service.SupplierProviderDataListParams{
		ProviderID: 42,
		Platform:   "grok",
		Page:       1,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "openai", result.Items[0].LocalAccountPlatform)
	require.Equal(t, "grok", result.Items[0].PlatformOverride)
	require.Equal(t, "grok", result.Items[0].EffectivePlatform)
	require.NoError(t, mock.ExpectationsWereMet())
}
func TestSupplierProviderDataRepositoryListGroupsIncludesFilteredSummary(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	active := true
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`COUNT\(\*\) FILTER \(WHERE active = TRUE AND local_group_id IS NOT NULL\) AS linked_group_count`).
		WithArgs(int64(42), "%vip%").
		WillReturnRows(sqlmock.NewRows([]string{
			"group_count", "account_count", "linked_group_count", "unlinked_group_count", "rate_risk_count",
			"active_group_count", "removed_group_count", "created_key_group_count", "attention_group_count",
		}).AddRow(int64(4), int64(9), int64(3), int64(1), int64(2), int64(4), int64(0), int64(3), int64(1)))
	mock.ExpectQuery(`(?s)LEFT JOIN groups lg ON lg\.id = g\.local_group_id.*LEFT JOIN supplier_provider_groups guardian_group ON guardian_group\.id = guard_state\.rate_guard_group_id.*LEFT JOIN supplier_providers guardian_provider ON guardian_provider\.id = guardian_group\.provider_id.*ORDER BY LOWER\(lg\.name\) DESC NULLS LAST, g\.id ASC`).
		WithArgs(int64(42), active, "%vip%", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name",
			"rate_multiplier", "raw_status", "active", "local_group_id", "local_group_name",
			"local_group_platform", "platform_override", "effective_platform", "local_rate_multiplier", "local_group_status", "auto_match_ignored",
			"auto_match_status", "matched_upstream_name", "name_change_pending",
			"rate_guard_selected", "rate_guard_enabled", "rate_guard_selection_mode", "rate_guard_last_snapshot_at", "rate_guard_last_checked_at",
			"group_sync_status", "last_group_sync_at", "local_group_active_mapping_count", "local_group_rate_guard_group_id", "local_group_rate_guard_group_name", "local_group_rate_guard_provider_name", "account_count",
			"last_seen_at", "inactive_at", "key_sync_status",
		}).AddRow(
			int64(7), int64(42), "Supplier A", "group-1", "VIP", 2.5, "active", true,
			int64(12), "VIP 本地", "openai", "grok", "grok", 3.0, "active", false, "manual", "VIP", false,
			true, false, "manual", now.Add(-time.Minute), now, "success", now.Add(-time.Minute), 2, int64(7), "VIP Guardian", "Supplier B", 5, now, nil,
			"success",
		))

	result, err := repo.ListGroups(context.Background(), service.SupplierProviderDataListParams{
		ProviderID: 42,
		Active:     &active,
		Search:     "vip",
		SortBy:     "local_group_name",
		SortOrder:  "desc",
		Page:       2,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(4), result.Total)
	require.Equal(t, int64(4), result.Summary.GroupCount)
	require.Equal(t, int64(9), result.Summary.AccountCount)
	require.Equal(t, int64(3), result.Summary.LinkedGroupCount)
	require.Equal(t, int64(1), result.Summary.UnlinkedGroupCount)
	require.Equal(t, int64(2), result.Summary.RateRiskCount)
	require.Equal(t, int64(4), result.Summary.ActiveGroupCount)
	require.Equal(t, int64(0), result.Summary.RemovedGroupCount)
	require.Equal(t, int64(3), result.Summary.CreatedKeyGroupCount)
	require.Equal(t, int64(1), result.Summary.AttentionGroupCount)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(12), *result.Items[0].LocalGroupID)
	require.Equal(t, "VIP 本地", result.Items[0].LocalGroupName)
	require.Equal(t, "openai", result.Items[0].LocalGroupPlatform)
	require.Equal(t, "grok", result.Items[0].PlatformOverride)
	require.Equal(t, "grok", result.Items[0].EffectivePlatform)
	require.Equal(t, 3.0, *result.Items[0].LocalRateMultiplier)
	require.Equal(t, "active", result.Items[0].LocalGroupStatus)
	require.False(t, result.Items[0].AutoMatchIgnored)
	require.Equal(t, service.AutoMatchStatusManual, result.Items[0].AutoMatchStatus)
	require.Equal(t, "VIP", result.Items[0].MatchedUpstreamName)
	require.True(t, result.Items[0].RateGuardSelected)
	require.Equal(t, "manual", result.Items[0].RateGuardSelectionMode)
	require.Equal(t, "success", result.Items[0].GroupSyncStatus)
	require.NotNil(t, result.Items[0].LastGroupSyncAt)
	require.Equal(t, "success", result.Items[0].KeySyncStatus)
	require.Equal(t, "created", result.Items[0].KeyStatus)
	require.Equal(t, 2, result.Items[0].LocalGroupActiveMappingCount)
	require.Equal(t, int64(7), *result.Items[0].LocalGroupRateGuardGroupID)
	require.Equal(t, "VIP Guardian", result.Items[0].LocalGroupRateGuardGroupName)
	require.Equal(t, "Supplier B", result.Items[0].LocalGroupRateGuardProviderName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListGroupsReturnsKeyStatus(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`COUNT\(\*\) FILTER \(WHERE TRUE AND local_group_id IS NOT NULL\) AS linked_group_count`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"group_count", "account_count", "linked_group_count", "unlinked_group_count", "rate_risk_count",
			"active_group_count", "removed_group_count", "created_key_group_count", "attention_group_count",
		}).AddRow(int64(3), int64(2), int64(0), int64(3), int64(0), int64(3), int64(0), int64(1), int64(0)))
	mock.ExpectQuery(`(?s)SELECT g\.id, g\.provider_id.*LEFT JOIN LATERAL.*sync_scope = 'accounts'.*ORDER BY`).
		WithArgs(int64(42), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name",
			"rate_multiplier", "raw_status", "active", "local_group_id", "local_group_name",
			"local_group_platform", "platform_override", "effective_platform", "local_rate_multiplier", "local_group_status", "auto_match_ignored",
			"auto_match_status", "matched_upstream_name", "name_change_pending",
			"rate_guard_selected", "rate_guard_enabled", "rate_guard_selection_mode", "rate_guard_last_snapshot_at", "rate_guard_last_checked_at",
			"group_sync_status", "last_group_sync_at", "local_group_active_mapping_count", "local_group_rate_guard_group_id", "local_group_rate_guard_group_name", "local_group_rate_guard_provider_name", "account_count",
			"last_seen_at", "inactive_at", "key_sync_status",
		}).AddRow(
			int64(1), int64(42), "Supplier A", "group-created", "已创建分组", 1.5, "active", true,
			nil, "", "", "", "", nil, "", false, "unmatched", "", false,
			false, false, "", nil, nil, "success", now, 0, nil, "", "", 2, now, nil, "failed",
		).AddRow(
			int64(2), int64(42), "Supplier A", "group-empty", "未创建分组", 1.5, "active", true,
			nil, "", "", "", "", nil, "", false, "unmatched", "", false,
			false, false, "", nil, nil, "success", now, 0, nil, "", "", 0, now, nil, "success",
		).AddRow(
			int64(3), int64(42), "Supplier A", "group-running", "同步中分组", 1.5, "active", true,
			nil, "", "", "", "", nil, "", false, "unmatched", "", false,
			false, false, "", nil, nil, "success", now, 0, nil, "", "", 0, now, nil, "running",
		))

	result, err := repo.ListGroups(context.Background(), service.SupplierProviderDataListParams{
		ProviderID: 42,
		Page:       1,
		PageSize:   20,
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 3)
	require.Equal(t, "failed", result.Items[0].KeySyncStatus)
	require.Equal(t, "created", result.Items[0].KeyStatus)
	require.Equal(t, "success", result.Items[1].KeySyncStatus)
	require.Equal(t, "not_created", result.Items[1].KeyStatus)
	require.Equal(t, "running", result.Items[2].KeySyncStatus)
	require.Equal(t, "unknown", result.Items[2].KeyStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderGroupKeyStatusPrefersExistingActiveKeys(t *testing.T) {
	tests := []struct {
		name           string
		keySyncStatus  string
		activeKeyCount int
		want           string
	}{
		{name: "同步失败但已有密钥", keySyncStatus: service.SupplierSyncStatusFailed, activeKeyCount: 2, want: "created"},
		{name: "尚未同步但已有密钥", keySyncStatus: "never", activeKeyCount: 1, want: "created"},
		{name: "同步成功且没有密钥", keySyncStatus: service.SupplierSyncStatusSuccess, activeKeyCount: 0, want: "not_created"},
		{name: "部分成功且没有密钥", keySyncStatus: service.SupplierSyncStatusPartial, activeKeyCount: 0, want: "not_created"},
		{name: "同步失败且没有密钥", keySyncStatus: service.SupplierSyncStatusFailed, activeKeyCount: 0, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, supplierProviderGroupKeyStatus(tt.keySyncStatus, tt.activeKeyCount))
		})
	}
}

func TestSupplierProviderDataRepositoryUpdatesGroupSyncStatus(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	syncedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_runtime_stats SET group_sync_status=$2, group_sync_message=$3, last_group_sync_at=$4, updated_at=$4 WHERE provider_id=$1")).
		WithArgs(int64(42), service.SupplierSyncStatusFailed, "upstream unavailable", syncedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateGroupSyncStatus(context.Background(), 42, service.SupplierSyncStatusFailed, "upstream unavailable", syncedAt))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderGroupBaseWhereIncludesPlatform(t *testing.T) {
	active := true
	where, args := supplierProviderGroupBaseWhere(service.SupplierProviderDataListParams{
		ProviderID: 42,
		Active:     &active,
		Search:     "vip",
		Platform:   "openai",
	})

	require.Contains(t, where, "g.provider_id = $1")
	require.Contains(t, where, "g.active = $2")
	require.Contains(t, where, "(g.name ILIKE $3 OR g.upstream_group_key ILIKE $3)")
	require.Contains(t, where, "COALESCE(NULLIF(group_platform_override.actual_platform, ''), lg.platform) = $4")
	require.Equal(t, []any{int64(42), active, "%vip%", "openai"}, args)
}

func TestSupplierProviderAccountWhereIncludesMappedGroupPlatform(t *testing.T) {
	active := true
	where, args := supplierProviderAccountWhere(service.SupplierProviderDataListParams{
		ProviderID: 42,
		Active:     &active,
		Search:     "primary",
		Platform:   "openai",
	})

	require.Contains(t, where, "a.provider_id = $1")
	require.Contains(t, where, "a.active = $2")
	require.Contains(t, where, "(a.name ILIKE $3 OR a.upstream_account_key ILIKE $3)")
	require.Contains(t, where, "mapped_group.upstream_group_key = a.group_key")
	require.Contains(t, where, "FROM accounts local_account")
	require.Contains(t, where, "COUNT(*) = 1")
	require.Contains(t, where, "LEFT JOIN supplier_local_account_platform_overrides platform_override")
	require.Contains(t, where, "COALESCE(NULLIF(platform_override.platform, ''), local_account.platform)")
	require.Contains(t, where, "COALESCE(")
	require.Contains(t, where, ") = $4")
	require.Contains(t, where, "lower(COALESCE(p.account_name_prefix, '') || a.name)")
	require.Equal(t, []any{int64(42), active, "%primary%", "openai"}, args)
}

func TestSupplierProviderAccountWhereIncludesLocalGroup(t *testing.T) {
	where, args := supplierProviderAccountWhere(service.SupplierProviderDataListParams{
		ProviderID: 42,
		GroupID:    201,
	})

	require.Contains(t, where, "a.provider_id = $1")
	require.Contains(t, where, "FROM accounts local_account")
	require.Contains(t, where, "JOIN account_groups account_group")
	require.Contains(t, where, "account_group.group_id = $2")
	require.Contains(t, where, "local_group.deleted_at IS NULL")
	require.Contains(t, where, "lower(COALESCE(p.account_name_prefix, '') || a.name)")
	require.Contains(t, where, ") = 1")
	require.NotContains(t, where, "mapped_group.local_group_id = $2")
	require.Equal(t, []any{int64(42), int64(201)}, args)
}

func TestSupplierProviderGroupListWhereAddsMatchStatusFilters(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		condition string
	}{
		{name: "linked", status: "linked", condition: "g.local_group_id IS NOT NULL"},
		{name: "unlinked", status: "unlinked", condition: "g.local_group_id IS NULL"},
		{name: "automatic", status: "auto_matched", condition: "g.local_group_id IS NOT NULL AND g.auto_match_status = 'auto_matched'"},
		{name: "manual", status: "manual", condition: "g.local_group_id IS NOT NULL AND g.auto_match_status = 'manual'"},
		{name: "ambiguous", status: "ambiguous", condition: "g.local_group_id IS NULL AND g.auto_match_status = 'ambiguous'"},
		{name: "ignored", status: "ignored", condition: "g.auto_match_ignored = TRUE"},
		{name: "name changed", status: "name_changed", condition: "g.name_change_pending = TRUE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, _ := supplierProviderGroupListWhere(service.SupplierProviderDataListParams{MatchStatus: tt.status})
			require.Contains(t, where, tt.condition)
		})
	}
}

func TestSupplierProviderGroupListWhereAddsRateStatusFilters(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		condition string
	}{
		{name: "normal", status: "normal", condition: "lg.rate_multiplier >= g.rate_multiplier * 1.1"},
		{name: "low", status: "low", condition: "lg.rate_multiplier > g.rate_multiplier + 0.000000001 AND lg.rate_multiplier < g.rate_multiplier * 1.1"},
		{name: "equal", status: "equal", condition: "ABS(lg.rate_multiplier - g.rate_multiplier) <= 0.000000001"},
		{name: "inverted", status: "inverted", condition: "lg.rate_multiplier < g.rate_multiplier - 0.000000001"},
		{name: "inactive", status: "inactive", condition: "COALESCE(lg.status, '') = 'inactive'"},
		{name: "invalid", status: "invalid", condition: "(g.rate_multiplier <= 0 OR lg.rate_multiplier IS NULL OR lg.rate_multiplier <= 0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, _ := supplierProviderGroupListWhere(service.SupplierProviderDataListParams{RateStatus: tt.status})
			require.Contains(t, where, tt.condition)
		})
	}
}

func TestSupplierProviderGroupOrderBy(t *testing.T) {
	tests := []struct {
		name     string
		sortBy   string
		order    string
		expected string
	}{
		{name: "default", expected: "g.active DESC, g.last_seen_at DESC, g.id ASC"},
		{name: "invalid field", sortBy: "updated_at", order: "asc", expected: "g.active DESC, g.last_seen_at DESC, g.id ASC"},
		{name: "provider name ascending", sortBy: "provider_name", order: "asc", expected: "LOWER(p.name) ASC, g.id ASC"},
		{name: "upstream group name descending", sortBy: "name", order: "desc", expected: "LOWER(g.name) DESC, g.id ASC"},
		{name: "upstream rate ascending", sortBy: "rate_multiplier", order: "asc", expected: "g.rate_multiplier ASC, g.id ASC"},
		{name: "local group name ascending", sortBy: "local_group_name", order: "asc", expected: "LOWER(lg.name) ASC NULLS LAST, g.id ASC"},
		{name: "trimmed local group name", sortBy: " local_group_name ", order: " asc ", expected: "LOWER(lg.name) ASC NULLS LAST, g.id ASC"},
		{name: "local rate descending", sortBy: "local_rate_multiplier", order: "desc", expected: "lg.rate_multiplier DESC NULLS LAST, g.id ASC"},
		{name: "account count ascending", sortBy: "account_count", order: "asc", expected: "COALESCE(COUNT(a.id) FILTER (WHERE a.active = TRUE), 0) ASC, g.id ASC"},
		{name: "invalid direction defaults ascending", sortBy: "name", order: "sideways", expected: "LOWER(g.name) ASC, g.id ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := supplierProviderGroupOrderBy(service.SupplierProviderDataListParams{
				SortBy:    tt.sortBy,
				SortOrder: tt.order,
			})
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestSupplierProviderAccountOrderBy(t *testing.T) {
	tests := []struct {
		name     string
		sortBy   string
		order    string
		expected string
	}{
		{name: "default", expected: "a.active DESC, a.last_seen_at DESC, a.id ASC"},
		{name: "provider name ascending", sortBy: "provider_name", order: "asc", expected: "LOWER(p.name) ASC, a.id ASC"},
		{name: "upstream account descending", sortBy: "upstream_account_key", order: "desc", expected: "LOWER(a.name) DESC, LOWER(a.upstream_account_key) DESC, a.id ASC"},
		{name: "local account name ascending", sortBy: "local_account_name", order: "asc", expected: "LOWER(matched_account.name) ASC NULLS LAST, a.id ASC"},
		{name: "local account priority descending", sortBy: "local_account_priority", order: "desc", expected: "matched_account.priority DESC NULLS LAST, a.id ASC"},
		{name: "upstream rate ascending", sortBy: "rate_multiplier", order: "asc", expected: "a.rate_multiplier ASC, a.id ASC"},
		{name: "local account status ascending", sortBy: "local_account_status", order: "asc", expected: "LOWER(matched_account.status) ASC NULLS LAST, a.id ASC"},
		{name: "schedulable descending", sortBy: "local_account_schedulable", order: "desc", expected: "matched_account.schedulable DESC NULLS LAST, a.id ASC"},
		{name: "last test status ascending", sortBy: "local_account_last_test_status", order: "asc", expected: "LOWER(NULLIF(matched_account.extra->>'last_test_status', '')) ASC NULLS LAST, a.id ASC"},
		{name: "last tested time descending", sortBy: "local_account_last_tested_at", order: "desc", expected: "NULLIF(matched_account.extra->>'last_tested_at', '') DESC NULLS LAST, a.id ASC"},
		{name: "current balance ascending", sortBy: "supplier_current_balance", order: "asc", expected: "COALESCE(runtime.current_balance, 0) ASC, a.id ASC"},
		{name: "today cost descending", sortBy: "supplier_today_cost", order: "desc", expected: "COALESCE(runtime.today_cost, 0) DESC, a.id ASC"},
		{name: "invalid field", sortBy: "binding_groups", order: "asc", expected: "a.active DESC, a.last_seen_at DESC, a.id ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := supplierProviderAccountOrderBy(service.SupplierProviderDataListParams{
				SortBy:    tt.sortBy,
				SortOrder: tt.order,
			})
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestSupplierProviderDataRepositoryListGroupsKeepsSummaryOutsideStatusFilters(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	active := true
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER \(WHERE active = TRUE\) AS group_count.*WHERE .*COALESCE\(NULLIF\(group_platform_override\.actual_platform, ''\), lg\.platform\) = \$2`).
		WithArgs(int64(42), "openai").
		WillReturnRows(sqlmock.NewRows([]string{
			"group_count", "account_count", "linked_group_count", "unlinked_group_count", "rate_risk_count",
			"active_group_count", "removed_group_count", "created_key_group_count", "attention_group_count",
		}).AddRow(int64(4), int64(9), int64(3), int64(1), int64(2), int64(4), int64(0), int64(3), int64(1)))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM supplier_provider_groups g.*g\.auto_match_status = 'manual'.*lg\.rate_multiplier > g\.rate_multiplier \+ 0\.000000001`).
		WithArgs(int64(42), active, "openai").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)FROM supplier_provider_groups g.*g\.auto_match_status = 'manual'.*lg\.rate_multiplier > g\.rate_multiplier \+ 0\.000000001`).
		WithArgs(int64(42), active, "openai", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name",
			"rate_multiplier", "raw_status", "active", "local_group_id", "local_group_name",
			"local_group_platform", "platform_override", "effective_platform", "local_rate_multiplier", "local_group_status", "auto_match_ignored",
			"auto_match_status", "matched_upstream_name", "name_change_pending",
			"rate_guard_selected", "rate_guard_enabled", "rate_guard_selection_mode", "rate_guard_last_snapshot_at", "rate_guard_last_checked_at",
			"group_sync_status", "last_group_sync_at", "local_group_active_mapping_count", "local_group_rate_guard_group_id", "local_group_rate_guard_group_name", "local_group_rate_guard_provider_name", "account_count",
			"last_seen_at", "inactive_at", "key_sync_status",
		}).AddRow(
			int64(7), int64(42), "Supplier A", "group-1", "VIP", 2.5, "active", true,
			int64(12), "VIP 本地", "openai", "", "openai", 2.6, "active", false, "manual", "VIP", false,
			false, false, "", nil, nil, "never", nil, 2, nil, "", "", 5, now, nil, "never",
		))

	result, err := repo.ListGroups(context.Background(), service.SupplierProviderDataListParams{
		ProviderID:  42,
		Active:      &active,
		Platform:    "openai",
		MatchStatus: "manual",
		RateStatus:  "low",
		Page:        1,
		PageSize:    20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(4), result.Summary.GroupCount)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderGroupListWhereFiltersKeyStatusOnServer(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		wantParts []string
	}{
		{
			name:   "created",
			status: "created",
			wantParts: []string{
				"EXISTS (",
				"supplier_provider_accounts key_account",
				"key_account.active = TRUE",
			},
		},
		{
			name:   "not created",
			status: "not_created",
			wantParts: []string{
				"NOT EXISTS (",
				"supplier_provider_accounts key_account",
				"COALESCE((",
				"sync_scope = 'accounts'",
				"IN ('success', 'partial')",
			},
		},
		{
			name:   "unknown",
			status: "unknown",
			wantParts: []string{
				"NOT EXISTS (",
				"supplier_provider_accounts key_account",
				"COALESCE((",
				"sync_scope = 'accounts'",
				"NOT IN ('success', 'partial')",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, args := supplierProviderGroupListWhere(service.SupplierProviderDataListParams{KeyStatus: tt.status})
			for _, part := range tt.wantParts {
				require.Contains(t, where, part)
			}
			require.Empty(t, args)
		})
	}
}

func TestSupplierProviderDataRepositoryUpdateGroupMappingSetsAndClearsLocalGroup(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	localGroupID := int64(12)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1 AND status = 'active')")).
		WithArgs(localGroupID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET local_group_id = $2::bigint, auto_match_status = CASE WHEN $2::bigint IS NULL THEN 'unmatched' ELSE 'manual' END, auto_match_ignored = CASE WHEN $2::bigint IS NULL THEN TRUE ELSE auto_match_ignored END, matched_upstream_name = CASE WHEN $2::bigint IS NULL THEN NULL ELSE name END, name_change_pending = FALSE, rate_guard_selected = CASE WHEN local_group_id IS DISTINCT FROM $2::bigint THEN FALSE ELSE rate_guard_selected END, rate_guard_selection_mode = CASE WHEN local_group_id IS DISTINCT FROM $2::bigint THEN '' ELSE rate_guard_selection_mode END, updated_at = NOW() WHERE id = $1")).
		WithArgs(int64(7), localGroupID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateGroupMapping(context.Background(), 7, &localGroupID))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET local_group_id = $2::bigint, auto_match_status = CASE WHEN $2::bigint IS NULL THEN 'unmatched' ELSE 'manual' END, auto_match_ignored = CASE WHEN $2::bigint IS NULL THEN TRUE ELSE auto_match_ignored END, matched_upstream_name = CASE WHEN $2::bigint IS NULL THEN NULL ELSE name END, name_change_pending = FALSE, rate_guard_selected = CASE WHEN local_group_id IS DISTINCT FROM $2::bigint THEN FALSE ELSE rate_guard_selected END, rate_guard_selection_mode = CASE WHEN local_group_id IS DISTINCT FROM $2::bigint THEN '' ELSE rate_guard_selection_mode END, updated_at = NOW() WHERE id = $1")).
		WithArgs(int64(7), nil).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateGroupMapping(context.Background(), 7, nil))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryAutoMatchStateOperations(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM supplier_provider_groups g\s+JOIN supplier_providers p ON p.id = g.provider_id\s+WHERE \(g.active = TRUE OR g.rate_guard_selected = TRUE\) AND g.provider_id = \$1`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name", "rate_multiplier", "raw_status", "active",
			"local_group_id", "auto_match_ignored", "auto_match_status", "matched_upstream_name", "name_change_pending", "last_seen_at", "inactive_at",
		}).
			AddRow(int64(7), int64(42), "Supplier A", "group-1", "VIP", 2.5, "active", true, nil, false, service.AutoMatchStatusUnmatched, "", false, now, nil).
			AddRow(int64(8), int64(42), "Supplier A", "group-old", "VIP Old", 2.5, "inactive", false, int64(12), false, service.AutoMatchStatusAutoMatched, "VIP Old", false, now.Add(-time.Hour), now))

	groups, err := repo.ListGroupsForAutoMatch(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	require.Nil(t, groups[0].LocalGroupID)
	require.False(t, groups[1].Active)
	require.Equal(t, int64(12), *groups[1].LocalGroupID)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET local_group_id = $2, auto_match_status = 'auto_matched', matched_upstream_name = $3, name_change_pending = FALSE, updated_at = NOW() WHERE id = $1 AND active = TRUE AND local_group_id IS NULL AND auto_match_ignored = FALSE")).
		WithArgs(int64(7), int64(12), "VIP").
		WillReturnResult(sqlmock.NewResult(0, 1))

	updated, err := repo.ApplyAutoMatch(context.Background(), 7, 12, "VIP")
	require.NoError(t, err)
	require.True(t, updated)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET auto_match_status = $2, name_change_pending = $3, updated_at = NOW() WHERE id = $1")).
		WithArgs(int64(7), service.AutoMatchStatusAmbiguous, true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.UpdateAutoMatchState(context.Background(), 7, service.AutoMatchStatusAmbiguous, true))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET auto_match_ignored = $2, updated_at = NOW() WHERE id = $1")).
		WithArgs(int64(7), true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.UpdateAutoMatchIgnored(context.Background(), 7, true))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListsMappingsForGuardReconciliation(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM supplier_provider_groups g.*g\.local_group_id = ANY\(\$1\)`).
		WithArgs(pq.Array([]int64{7})).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name", "rate_multiplier", "raw_status", "active",
			"local_group_id", "auto_match_ignored", "auto_match_status", "matched_upstream_name", "name_change_pending",
			"rate_guard_selected", "rate_guard_enabled", "rate_guard_selection_mode", "last_seen_at", "inactive_at",
		}).AddRow(
			int64(10), int64(42), "Supplier A", "vip", "VIP", 2.5, "active", true,
			int64(7), false, service.AutoMatchStatusAutoMatched, "VIP", false,
			true, true, service.RateGuardSelectionModeAuto, now, nil,
		))

	groups, err := repo.ListMappingsByLocalGroup(context.Background(), []int64{7})

	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.True(t, groups[0].RateGuardSelected)
	require.Equal(t, service.RateGuardSelectionModeAuto, groups[0].RateGuardSelectionMode)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositorySelectsRateGuardAtomically(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT local_group_id, active FROM supplier_provider_groups WHERE id=$1 FOR UPDATE")).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"local_group_id", "active"}).AddRow(int64(7), true))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock($1)")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_xact_lock"}).AddRow(nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET rate_guard_selected=FALSE, rate_guard_selection_mode='', updated_at=NOW() WHERE local_group_id=$1 AND rate_guard_selected=TRUE")).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET rate_guard_selected=TRUE, rate_guard_selection_mode=$2, updated_at=NOW() WHERE id=$1")).
		WithArgs(int64(10), service.RateGuardSelectionModeManual).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SelectRateGuard(context.Background(), 10, service.RateGuardSelectionModeManual))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryClearsOnlyMatchingGuardMode(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET rate_guard_selected=FALSE, rate_guard_selection_mode='', updated_at=NOW() WHERE id=$1 AND rate_guard_selection_mode=$2")).
		WithArgs(int64(10), service.RateGuardSelectionModeAuto).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.ClearRateGuard(context.Background(), 10, service.RateGuardSelectionModeAuto))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetsRateGuardSelectionWithEnabledPolicy(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`WHERE g\.id = \$1`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name", "rate_multiplier", "raw_status", "active",
			"local_group_id", "auto_match_ignored", "auto_match_status", "matched_upstream_name", "name_change_pending",
			"rate_guard_selected", "rate_guard_enabled", "rate_guard_selection_mode", "last_seen_at", "inactive_at",
		}).AddRow(
			int64(10), int64(42), "Supplier A", "vip", "VIP", 2.5, "active", true,
			int64(7), false, service.AutoMatchStatusManual, "VIP", false,
			true, true, service.RateGuardSelectionModeManual, now, nil,
		))

	group, err := repo.GetGroupForRateGuard(context.Background(), 10)

	require.NoError(t, err)
	require.True(t, group.RateGuardSelected)
	require.True(t, group.RateGuardEnabled)
	require.Equal(t, service.RateGuardSelectionModeManual, group.RateGuardSelectionMode)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryUpdatesRateGuardEnabledPolicy(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	expectedSQL := regexp.QuoteMeta("UPDATE supplier_provider_groups SET rate_guard_enabled=$2, updated_at=NOW() WHERE id=$1")

	mock.ExpectExec(expectedSQL).
		WithArgs(int64(10), true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.SetRateGuardEnabled(context.Background(), 10, true))

	mock.ExpectExec(expectedSQL).
		WithArgs(int64(10), false).
		WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.SetRateGuardEnabled(context.Background(), 10, false), service.ErrSupplierProviderGroupNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
func TestSupplierProviderDataRepositoryListsSelectedAndEnabledRateGuardCandidates(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`WHERE g\.rate_guard_selected = TRUE\s+AND g\.rate_guard_enabled = TRUE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"mapping_id", "provider_id", "provider_name", "provider_enabled",
			"upstream_group_key", "upstream_group_name", "upstream_rate_multiplier", "guardian_active",
			"local_group_id", "local_group_name", "local_group_status", "local_rate_multiplier",
			"snapshot_at", "last_snapshot_at", "group_sync_status", "last_group_sync_at",
		}).AddRow(
			int64(10), int64(42), "Supplier A", true,
			"vip", "VIP", 2.5, true,
			int64(7), "VIP Local", "active", 2.6,
			now.Add(-time.Minute), nil, "success", now.Add(-time.Minute),
		))

	candidates, err := repo.ListRateGuardCandidates(context.Background())

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, int64(10), candidates[0].MappingID)
	require.Equal(t, int64(7), candidates[0].LocalGroupID)
	require.InDelta(t, 2.6, candidates[0].LocalRateMultiplier, 0.000000001)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryRateGuardRechecksLocalRateBeforeRaise(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	snapshotAt := now.Add(-time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE OF g, lg`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"rate_guard_selected", "guardian_active", "snapshot_at", "last_snapshot_at",
			"provider_enabled", "local_group_id", "local_rate_multiplier", "local_group_status",
			"group_sync_status", "last_group_sync_at",
		}).AddRow(true, true, snapshotAt, nil, true, int64(7), 3.0, "active", "success", snapshotAt))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET rate_guard_last_snapshot_at=$2, rate_guard_last_checked_at=$3, updated_at=NOW() WHERE id=$1")).
		WithArgs(int64(10), snapshotAt, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.ApplyRateGuard(context.Background(), service.SupplierRateGuardApplyInput{
		MappingID: 10, ExpectedSnapshotAt: snapshotAt, CheckedAt: now,
		TargetRate: 2.75, MaxSnapshotAge: 30 * time.Minute,
	})

	require.NoError(t, err)
	require.Equal(t, service.SupplierRateGuardActionUnchanged, result.Action)
	require.InDelta(t, 3.0, result.OldRate, 0.000000001)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryRateGuardRaisesEnqueuesGroupChangeAndCreatesPendingLog(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	snapshotAt := now.Add(-time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE OF g, lg`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"rate_guard_selected", "guardian_active", "snapshot_at", "last_snapshot_at",
			"provider_enabled", "local_group_id", "local_rate_multiplier", "local_group_status",
			"group_sync_status", "last_group_sync_at",
		}).AddRow(true, true, snapshotAt, nil, true, int64(7), 2.6, "active", "success", snapshotAt))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE groups SET rate_multiplier=$2, updated_at=NOW() WHERE id=$1 AND rate_multiplier < $2")).
		WithArgs(int64(7), 2.75).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO scheduler_outbox")).
		WithArgs(service.SchedulerOutboxEventGroupChanged, nil, int64(7), nil, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_rate_guard_change_logs")).
		WithArgs(int64(10), 2.6, 2.75, service.SupplierRateGuardChangeLogStatusPending, now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET rate_guard_last_snapshot_at=$2, rate_guard_last_checked_at=$3, updated_at=NOW() WHERE id=$1")).
		WithArgs(int64(10), snapshotAt, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.ApplyRateGuard(context.Background(), service.SupplierRateGuardApplyInput{
		MappingID: 10, ExpectedSnapshotAt: snapshotAt, CheckedAt: now,
		TargetRate: 2.75, MaxSnapshotAge: 30 * time.Minute,
	})

	require.NoError(t, err)
	require.Equal(t, service.SupplierRateGuardActionRaised, result.Action)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryRateGuardDoesNotConsumeSnapshotWhenSyncFailsDuringLock(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	snapshotAt := now.Add(-time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(`FOR UPDATE OF g, lg`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{
			"rate_guard_selected", "guardian_active", "snapshot_at", "last_snapshot_at",
			"provider_enabled", "local_group_id", "local_rate_multiplier", "local_group_status",
			"group_sync_status", "last_group_sync_at",
		}).AddRow(true, true, snapshotAt, nil, true, int64(7), 2.6, "active", "failed", snapshotAt))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET rate_guard_last_checked_at=$2, updated_at=NOW() WHERE id=$1")).
		WithArgs(int64(10), now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.ApplyRateGuard(context.Background(), service.SupplierRateGuardApplyInput{
		MappingID: 10, ExpectedSnapshotAt: snapshotAt, CheckedAt: now,
		TargetRate: 2.75, MaxSnapshotAge: 30 * time.Minute,
	})

	require.NoError(t, err)
	require.Equal(t, service.SupplierRateGuardActionInvalid, result.Action)
	require.Equal(t, service.SupplierRateGuardReasonGroupSyncFailed, result.Reason)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryLoadsAndAcknowledgesNameChange(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`WHERE g.id = \$1`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name", "rate_multiplier", "raw_status", "active",
			"local_group_id", "auto_match_ignored", "auto_match_status", "matched_upstream_name", "name_change_pending", "last_seen_at", "inactive_at",
		}).AddRow(int64(7), int64(42), "Supplier A", "group-1", "VIP New", 2.5, "active", true, int64(12), false, service.AutoMatchStatusManual, "VIP Old", true, now, nil))

	group, err := repo.GetGroupForAutoMatch(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "VIP Old", group.MatchedUpstreamName)
	require.True(t, group.NameChangePending)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET matched_upstream_name = $2, name_change_pending = FALSE, updated_at = NOW() WHERE id = $1 AND local_group_id IS NOT NULL")).
		WithArgs(int64(7), "VIP New").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.AcknowledgeNameChange(context.Background(), 7, "VIP New"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryCleanupUsesBatchLimit(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	policy := service.SupplierCleanupPolicy{
		AutomationRunRetentionDays:        30,
		SyncRunRetentionDays:              30,
		MetricRetentionDays:               30,
		DailyStatRetentionDays:            365,
		InactiveAccountDays:               90,
		InactiveGroupDays:                 90,
		AccountHealthHistoryRetentionDays: 30,
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
	mock.ExpectExec("supplier_account_health_history WHERE checked_at <").
		WithArgs(now.AddDate(0, 0, -30), 2).
		WillReturnResult(sqlmock.NewResult(0, 0))

	counts, err := repo.Cleanup(context.Background(), policy, now, 2)

	require.NoError(t, err)
	require.Equal(t, 3, counts.AutomationRuns)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryCleanupUsesCapturedAtForMetricSnapshots(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	policy := service.SupplierCleanupPolicy{
		AutomationRunRetentionDays:        30,
		SyncRunRetentionDays:              30,
		MetricRetentionDays:               30,
		DailyStatRetentionDays:            365,
		InactiveAccountDays:               90,
		InactiveGroupDays:                 90,
		AccountHealthHistoryRetentionDays: 30,
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
	mock.ExpectExec("supplier_account_health_history WHERE checked_at <").
		WithArgs(now.AddDate(0, 0, -30), 1000).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := repo.Cleanup(context.Background(), policy, now, 1000)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryGetsEffectiveLocalAccountPlatform(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectQuery(`(?s)SELECT COALESCE\(.*supplier_local_account_platform_overrides.*local_account\.deleted_at IS NULL`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"effective_platform"}).AddRow("grok"))

	platform, err := repo.GetLocalAccountEffectivePlatform(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, service.PlatformGrok, platform)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryLocalAccountPlatformOverrideLifecycle(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)

	mock.ExpectExec(`INSERT INTO supplier_local_account_platform_overrides`).
		WithArgs(int64(42), "grok").
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.SetLocalAccountPlatformOverride(context.Background(), 42, " GROK "))

	mock.ExpectQuery(`SELECT platform FROM supplier_local_account_platform_overrides WHERE local_account_id = \$1`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"platform"}).AddRow("grok"))
	platform, err := repo.GetLocalAccountPlatformOverride(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, "grok", platform)

	mock.ExpectExec(`DELETE FROM supplier_local_account_platform_overrides WHERE local_account_id = \$1`).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.ClearLocalAccountPlatformOverride(context.Background(), 42))

	mock.ExpectQuery(`SELECT platform FROM supplier_local_account_platform_overrides WHERE local_account_id = \$1`).
		WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)
	platform, err = repo.GetLocalAccountPlatformOverride(context.Background(), 42)
	require.NoError(t, err)
	require.Empty(t, platform)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNormalizeSupplierAccountStatusFilter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"active", "active"},
		{" Disabled ", "disabled"},
		{"EXPIRED", "expired"},
		{"quota_exhausted", "quota_exhausted"},
		{"unknown", "unknown"},
		{"inactive", ""},
		{"", ""},
		{"deleted", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, normalizeSupplierAccountStatusFilter(tc.in))
		})
	}
}

func TestSupplierProviderAccountWhereFiltersNormalizedStatus(t *testing.T) {
	t.Parallel()

	where, args := supplierProviderAccountWhere(service.SupplierProviderDataListParams{
		Status: " Active ",
	})
	require.Contains(t, where, "LOWER(a.status) = $")
	require.Equal(t, []any{"active"}, args[len(args)-1:])

	where, args = supplierProviderAccountWhere(service.SupplierProviderDataListParams{
		Status: "deleted",
	})
	require.NotContains(t, where, "a.status")
	require.Empty(t, args)
}

func TestSupplierProviderAccountOrderByUpstreamStatus(t *testing.T) {
	t.Parallel()

	got := supplierProviderAccountOrderBy(service.SupplierProviderDataListParams{
		SortBy:    "upstream_account_status",
		SortOrder: "desc",
	})
	require.Equal(t, "LOWER(a.status) DESC, a.id ASC", got)
}
