package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
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
	"group_key", "group_name", "platform", "rate_multiplier", "raw_status", "active",
	"last_seen_at", "inactive_at",
	"local_account_match_status", "local_account_match_count",
	"local_account_id", "local_account_name", "local_account_priority",
	"local_account_status", "local_account_schedulable",
	"local_account_last_test_status", "local_account_last_tested_at", "local_account_last_test_error",
	"binding_groups",
	"supplier_current_balance", "supplier_today_cost",
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
       matched_account.priority AS local_account_priority,
       COALESCE(matched_account.status, '') AS local_account_status,
       matched_account.schedulable AS local_account_schedulable,
       COALESCE(matched_account.extra->>'last_test_status', '') AS local_account_last_test_status,
       COALESCE(matched_account.extra->>'last_tested_at', '') AS local_account_last_tested_at,
       COALESCE(matched_account.extra->>'last_test_error', '') AS local_account_last_test_error,
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
       COALESCE(runtime.today_cost, 0) AS supplier_today_cost
FROM supplier_provider_accounts a
JOIN supplier_providers p ON p.id = a.provider_id
LEFT JOIN supplier_provider_runtime_stats runtime ON runtime.provider_id = p.id
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS match_count,
         MIN(local_account.id) AS local_account_id
  FROM accounts local_account
  WHERE local_account.deleted_at IS NULL
    AND regexp_replace(lower(local_account.name), '[^[:alnum:]]', '', 'g')
        = regexp_replace(
            lower(p.account_name_prefix || a.name),
            '[^[:alnum:]]',
            '',
            'g'
          )
) local_match ON TRUE
LEFT JOIN accounts matched_account
  ON matched_account.id = local_match.local_account_id
 AND local_match.match_count = 1
WHERE ` + whereSQL + `
ORDER BY a.active DESC, a.last_seen_at DESC, a.id ASC LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder

	return "^" + regexp.QuoteMeta(strings.Join(strings.Fields(querySQL), " ")) + "$"
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
	mock.ExpectQuery(supplierProviderAccountListQueryContractPattern(
		"p.deleted_at IS NULL AND a.provider_id = $1 AND a.active = $2 AND (a.name ILIKE $3 OR a.upstream_account_key ILIKE $3)",
		"$4",
		"$5",
	)).
		WithArgs(int64(42), active, "%pri%", 20, 20).
		WillReturnRows(sqlmock.NewRows(supplierProviderAccountListColumns).AddRow(
			int64(7), int64(42), "Supplier A", "account-1", "Primary", "active", "group-1", "VIP", "openai", 2.5, "active", true, now, nil,
			"matched", 1, int64(101), "prefix-key-1", 80, "active", true, "success", "2026-07-16T09:30:00Z", "upstream authentication failed",
			`[{"id":202,"name":"Claude 订阅","platform":"anthropic","rate_multiplier":2,"subscription_type":"subscription"},{"id":201,"name":"OpenAI 专线","platform":"openai","rate_multiplier":1.5,"subscription_type":"standard"}]`,
			12.5, 3.25,
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
	require.Equal(t, "matched", result.Items[0].LocalAccountMatchStatus)
	require.Equal(t, 1, result.Items[0].LocalAccountMatchCount)
	require.NotNil(t, result.Items[0].LocalAccountID)
	require.Equal(t, int64(101), *result.Items[0].LocalAccountID)
	require.Equal(t, "prefix-key-1", result.Items[0].LocalAccountName)
	require.NotNil(t, result.Items[0].LocalAccountPriority)
	require.Equal(t, 80, *result.Items[0].LocalAccountPriority)
	require.Equal(t, "active", result.Items[0].LocalAccountStatus)
	require.NotNil(t, result.Items[0].LocalAccountSchedulable)
	require.True(t, *result.Items[0].LocalAccountSchedulable)
	require.Equal(t, "success", result.Items[0].LocalAccountLastTestStatus)
	require.Equal(t, "2026-07-16T09:30:00Z", result.Items[0].LocalAccountLastTestedAt)
	require.Equal(t, "upstream authentication failed", result.Items[0].LocalAccountLastTestError)
	require.Equal(t, []service.SupplierProviderAccountBindingGroup{
		{ID: 202, Name: "Claude 订阅", Platform: "anthropic", RateMultiplier: 2, SubscriptionType: "subscription"},
		{ID: 201, Name: "OpenAI 专线", Platform: "openai", RateMultiplier: 1.5, SubscriptionType: "standard"},
	}, result.Items[0].BindingGroups)
	require.Equal(t, 12.5, result.Items[0].SupplierCurrentBalance)
	require.Equal(t, 3.25, result.Items[0].SupplierTodayCost)
	require.NoError(t, mock.ExpectationsWereMet())
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
				int64(7), int64(42), "Supplier A", "missing-key", "Missing", "active", "group-1", "VIP", "openai", 2.5, "active", true, now, nil,
				"unmatched", 0, nil, "", nil, "", nil, "", "", "", `[]`, 12.5, 3.25,
			).
			AddRow(
				int64(8), int64(42), "Supplier A", "duplicate-key", "Duplicate", "active", "group-2", "Standard", "openai", 1.5, "active", true, now, nil,
				"conflict", 2, nil, "", nil, "", nil, "", "", "", `[]`, 12.5, 3.25,
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
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryListGroupsIncludesFilteredSummary(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	active := true
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`COUNT\(\*\) FILTER \(WHERE local_group_id IS NOT NULL\) AS linked_group_count`).
		WithArgs(int64(42), active, "%vip%").
		WillReturnRows(sqlmock.NewRows([]string{
			"group_count", "account_count", "linked_group_count", "unlinked_group_count", "rate_risk_count",
		}).AddRow(int64(4), int64(9), int64(3), int64(1), int64(2)))
	mock.ExpectQuery(`(?s)LEFT JOIN groups lg ON lg\.id = g\.local_group_id.*LEFT JOIN supplier_provider_groups guardian_group ON guardian_group\.id = guard_state\.rate_guard_group_id.*LEFT JOIN supplier_providers guardian_provider ON guardian_provider\.id = guardian_group\.provider_id.*ORDER BY LOWER\(lg\.name\) DESC NULLS LAST, g\.id ASC`).
		WithArgs(int64(42), active, "%vip%", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name",
			"rate_multiplier", "raw_status", "active", "local_group_id", "local_group_name",
			"local_group_platform", "local_rate_multiplier", "local_group_status", "auto_match_ignored",
			"auto_match_status", "matched_upstream_name", "name_change_pending",
			"rate_guard_selected", "rate_guard_selection_mode", "rate_guard_last_snapshot_at", "rate_guard_last_checked_at",
			"group_sync_status", "last_group_sync_at", "local_group_active_mapping_count", "local_group_rate_guard_group_id", "local_group_rate_guard_group_name", "local_group_rate_guard_provider_name", "account_count",
			"last_seen_at", "inactive_at",
		}).AddRow(
			int64(7), int64(42), "Supplier A", "group-1", "VIP", 2.5, "active", true,
			int64(12), "VIP 本地", "openai", 3.0, "active", false, "manual", "VIP", false,
			true, "manual", now.Add(-time.Minute), now, "success", now.Add(-time.Minute), 2, int64(7), "VIP Guardian", "Supplier B", 5, now, nil,
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
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(12), *result.Items[0].LocalGroupID)
	require.Equal(t, "VIP 本地", result.Items[0].LocalGroupName)
	require.Equal(t, "openai", result.Items[0].LocalGroupPlatform)
	require.Equal(t, 3.0, *result.Items[0].LocalRateMultiplier)
	require.Equal(t, "active", result.Items[0].LocalGroupStatus)
	require.False(t, result.Items[0].AutoMatchIgnored)
	require.Equal(t, service.AutoMatchStatusManual, result.Items[0].AutoMatchStatus)
	require.Equal(t, "VIP", result.Items[0].MatchedUpstreamName)
	require.True(t, result.Items[0].RateGuardSelected)
	require.Equal(t, "manual", result.Items[0].RateGuardSelectionMode)
	require.Equal(t, "success", result.Items[0].GroupSyncStatus)
	require.NotNil(t, result.Items[0].LastGroupSyncAt)
	require.Equal(t, 2, result.Items[0].LocalGroupActiveMappingCount)
	require.Equal(t, int64(7), *result.Items[0].LocalGroupRateGuardGroupID)
	require.Equal(t, "VIP Guardian", result.Items[0].LocalGroupRateGuardGroupName)
	require.Equal(t, "Supplier B", result.Items[0].LocalGroupRateGuardProviderName)
	require.NoError(t, mock.ExpectationsWereMet())
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
	require.Contains(t, where, "lg.platform = $4")
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
	require.Contains(t, where, "local_group.platform = $4")
	require.Equal(t, []any{int64(42), active, "%primary%", "openai"}, args)
}

func TestSupplierProviderAccountWhereIncludesLocalGroup(t *testing.T) {
	where, args := supplierProviderAccountWhere(service.SupplierProviderDataListParams{
		ProviderID: 42,
		GroupID:    201,
	})

	require.Contains(t, where, "a.provider_id = $1")
	require.Contains(t, where, "mapped_group.provider_id = a.provider_id")
	require.Contains(t, where, "mapped_group.upstream_group_key = a.group_key")
	require.Contains(t, where, "mapped_group.local_group_id = $2")
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

	mock.ExpectQuery(`(?s)COUNT\(\*\) AS group_count.*WHERE .*lg\.platform = \$3`).
		WithArgs(int64(42), active, "openai").
		WillReturnRows(sqlmock.NewRows([]string{
			"group_count", "account_count", "linked_group_count", "unlinked_group_count", "rate_risk_count",
		}).AddRow(int64(4), int64(9), int64(3), int64(1), int64(2)))
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM supplier_provider_groups g.*g\.auto_match_status = 'manual'.*lg\.rate_multiplier > g\.rate_multiplier \+ 0\.000000001`).
		WithArgs(int64(42), active, "openai").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)FROM supplier_provider_groups g.*g\.auto_match_status = 'manual'.*lg\.rate_multiplier > g\.rate_multiplier \+ 0\.000000001`).
		WithArgs(int64(42), active, "openai", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_name", "upstream_group_key", "name",
			"rate_multiplier", "raw_status", "active", "local_group_id", "local_group_name",
			"local_group_platform", "local_rate_multiplier", "local_group_status", "auto_match_ignored",
			"auto_match_status", "matched_upstream_name", "name_change_pending",
			"rate_guard_selected", "rate_guard_selection_mode", "rate_guard_last_snapshot_at", "rate_guard_last_checked_at",
			"group_sync_status", "last_group_sync_at", "local_group_active_mapping_count", "local_group_rate_guard_group_id", "local_group_rate_guard_group_name", "local_group_rate_guard_provider_name", "account_count",
			"last_seen_at", "inactive_at",
		}).AddRow(
			int64(7), int64(42), "Supplier A", "group-1", "VIP", 2.5, "active", true,
			int64(12), "VIP 本地", "openai", 2.6, "active", false, "manual", "VIP", false,
			false, "", nil, nil, "never", nil, 2, nil, "", "", 5, now, nil,
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

func TestSupplierProviderDataRepositoryUpdateGroupMappingSetsAndClearsLocalGroup(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	localGroupID := int64(12)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1 AND status = 'active')")).
		WithArgs(localGroupID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET local_group_id = $2, auto_match_status = CASE WHEN $2 IS NULL THEN 'unmatched' ELSE 'manual' END, auto_match_ignored = CASE WHEN $2 IS NULL THEN TRUE ELSE auto_match_ignored END, matched_upstream_name = CASE WHEN $2 IS NULL THEN NULL ELSE name END, name_change_pending = FALSE, rate_guard_selected = CASE WHEN local_group_id IS DISTINCT FROM $2 THEN FALSE ELSE rate_guard_selected END, rate_guard_selection_mode = CASE WHEN local_group_id IS DISTINCT FROM $2 THEN '' ELSE rate_guard_selection_mode END, updated_at = NOW() WHERE id = $1")).
		WithArgs(int64(7), localGroupID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateGroupMapping(context.Background(), 7, &localGroupID))

	mock.ExpectExec(regexp.QuoteMeta("UPDATE supplier_provider_groups SET local_group_id = $2, auto_match_status = CASE WHEN $2 IS NULL THEN 'unmatched' ELSE 'manual' END, auto_match_ignored = CASE WHEN $2 IS NULL THEN TRUE ELSE auto_match_ignored END, matched_upstream_name = CASE WHEN $2 IS NULL THEN NULL ELSE name END, name_change_pending = FALSE, rate_guard_selected = CASE WHEN local_group_id IS DISTINCT FROM $2 THEN FALSE ELSE rate_guard_selected END, rate_guard_selection_mode = CASE WHEN local_group_id IS DISTINCT FROM $2 THEN '' ELSE rate_guard_selection_mode END, updated_at = NOW() WHERE id = $1")).
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
			"rate_guard_selected", "rate_guard_selection_mode", "last_seen_at", "inactive_at",
		}).AddRow(
			int64(10), int64(42), "Supplier A", "vip", "VIP", 2.5, "active", true,
			int64(7), false, service.AutoMatchStatusAutoMatched, "VIP", false,
			true, service.RateGuardSelectionModeAuto, now, nil,
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

func TestSupplierProviderDataRepositoryListsSelectedRateGuardCandidates(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`WHERE g\.rate_guard_selected = TRUE`).
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
