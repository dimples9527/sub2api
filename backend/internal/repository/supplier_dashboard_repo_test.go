package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.SupplierDashboardDetailRepository = (*supplierDashboardRepository)(nil)

var dashboardAccountColumns = []string{
	"account_id", "account_name", "provider_slug", "provider_name", "provider_enabled", "account_enabled", "group_key", "group_name",
	"provider_risk_level", "provider_risk_updated_at", "account_status", "rate_sync_status", "balance_sync_status", "balance_synced_at", "task_status", "task_reason", "task_finished_at",
	"current_rate", "previous_rate", "snapshot_count", "rate_change_old", "rate_change_new", "rate_change_count", "rate_changed_at", "balance", "estimated_days", "success_count", "error_count", "period_cost", "last_rate_synced_at", "observed_at",
}
var dashboardRateColumns = []string{
	"account_id", "account_name", "provider_slug", "provider_name", "provider_enabled", "account_enabled", "group_key", "group_name",
	"current_rate", "previous_rate", "snapshot_count", "rate_change_old", "rate_change_new", "rate_change_count", "rate_changed_at", "success_count", "error_count", "period_cost", "last_rate_synced_at", "observed_at",
}
var dashboardProviderColumns = []string{
	"provider_slug", "provider_name", "enabled", "data_complete", "provider_risk_level", "sync_status", "group_sync_status", "balance_sync_status",
	"rate_risk_count", "enabled_account_count", "schedulable_account_count", "success_count", "error_count", "balance", "estimated_days", "period_cost", "last_synced_at",
}

func TestSupplierDashboardRepositoryScansNeutralFactsAndBatchedWindow(t *testing.T) {
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	repo, mock, seen := newDashboardRepoMock(t)
	mock.ExpectQuery("(?s)FROM supplier_provider_accounts").WithArgs(start, end, "", "").WillReturnRows(sqlmock.NewRows(dashboardAccountColumns).
		AddRow(int64(1), "a", "p", "P", true, true, "g", "G", "critical", end, "failed", "partial", "success", end, "failed", "test timeout", end, 0.0, 0.8, 3, 0.8, 0.9, 2, end, 0.0, nil, int64(0), int64(0), 0.0, end, end).
		AddRow(int64(2), "b", "p", "P", false, false, "g2", "G2", "normal", nil, "active", "success", "never", nil, nil, nil, nil, nil, nil, 1, nil, nil, 0, nil, nil, nil, nil, nil, nil, nil, end))
	accounts, err := repo.ListDashboardAccounts(context.Background(), start, end, "", "")
	if err != nil || len(accounts) != 2 {
		t.Fatalf("accounts = %+v, err=%v", accounts, err)
	}
	first := accounts[0]
	if first.ProviderRiskUpdatedAt == nil || first.BalanceSyncStatus != "success" || first.BalanceSyncedAt == nil || first.TaskStatus != "failed" || first.TaskReason != "test timeout" || first.TaskFinishedAt == nil || first.PreviousRate == nil || *first.PreviousRate != 0.8 || first.SnapshotCount != 3 || first.RateChangeOld == nil || *first.RateChangeOld != 0.8 || first.RateChangeNew == nil || *first.RateChangeNew != 0.9 || first.RateChangeCount != 2 || first.RateChangedAt == nil {
		t.Fatalf("account task/history facts: %+v", first)
	}
	if first.SuccessCount == nil || *first.SuccessCount != 0 || first.ErrorCount == nil || *first.ErrorCount != 0 || first.PeriodCost == nil || *first.PeriodCost != 0 {
		t.Fatalf("account raw traffic null/zero contract: %+v", first)
	}
	if first.Balance == nil || *first.Balance != 0 || first.EstimatedDays != nil || first.LastRateSyncedAt == nil {
		t.Fatalf("account collection facts: %+v", first)
	}
	if accounts[1].SuccessCount != nil || accounts[1].ErrorCount != nil || accounts[1].CurrentRate != nil || accounts[1].PeriodCost != nil || accounts[1].ProviderEnabled || accounts[1].AccountEnabled {
		t.Fatalf("account unmatched/disabled contract: %+v", accounts[1])
	}
	for _, forbidden := range []string{"Severity", "RiskTypes", "TrafficImpact", "LowestRate", "RateDeltaPercent", "EstimatedExtraCost", "SuccessRate", "RequestCount"} {
		if _, ok := reflect.TypeOf(service.SupplierDashboardAccountSnapshot{}).FieldByName(forbidden); ok {
			t.Fatalf("repository account snapshot still exposes derived field %s", forbidden)
		}
	}

	mock.ExpectQuery("(?s)FROM supplier_provider_accounts").WithArgs(start, end, "", "").WillReturnRows(sqlmock.NewRows(dashboardRateColumns).
		AddRow(int64(3), "r", "p", "P", true, true, "g", "G", 0.0, nil, 1, nil, nil, 0, nil, int64(0), int64(0), 0.0, end, end))
	rates, err := repo.ListDashboardRates(context.Background(), start, end, "", "")
	if err != nil || len(rates) != 1 || rates[0].CurrentRate == nil || *rates[0].CurrentRate != 0 || rates[0].SuccessCount == nil || *rates[0].SuccessCount != 0 || rates[0].ErrorCount == nil || *rates[0].ErrorCount != 0 || rates[0].PeriodCost == nil || *rates[0].PeriodCost != 0 {
		t.Fatalf("rates = %+v, err=%v", rates, err)
	}
	if _, ok := reflect.TypeOf(service.SupplierDashboardRateSnapshot{}).FieldByName("EstimatedExtraCost"); ok {
		t.Fatal("repository rate snapshot still exposes derived extra cost")
	}

	mock.ExpectQuery("(?s)FROM supplier_providers").WithArgs(start, end, "", "").WillReturnRows(sqlmock.NewRows(dashboardProviderColumns).
		AddRow("p", "P", true, false, "normal", "never", "never", "never", 0, 2, 1, nil, nil, nil, nil, nil, nil))
	providers, err := repo.ListDashboardProviders(context.Background(), start, end)
	if err != nil || len(providers) != 1 || providers[0].DataComplete || providers[0].Balance != nil || providers[0].EnabledAccountCount != 2 || providers[0].SchedulableAccountCount != 1 {
		t.Fatalf("providers = %+v, err=%v", providers, err)
	}
	for _, forbidden := range []string{"CriticalIssueCount", "BalanceRisk", "SyncRisk", "SuccessRate", "RequestCount"} {
		if _, ok := reflect.TypeOf(service.SupplierDashboardProviderSnapshot{}).FieldByName(forbidden); ok {
			t.Fatalf("repository provider snapshot still exposes derived field %s", forbidden)
		}
	}
	if len(*seen) != 3 {
		t.Fatalf("query count = %d, want 3", len(*seen))
	}
	for _, query := range *seen {
		assertDashboardSQLSafe(t, query)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSupplierDashboardRepositoryQueriesEncodeSharedOpsAndUniqueSemantics(t *testing.T) {
	for name, query := range map[string]string{"accounts": supplierDashboardAccountsQuery, "rates": supplierDashboardRatesQuery, "providers": supplierDashboardProvidersQuery} {
		t.Run(name, func(t *testing.T) {
			lower := strings.ToLower(query)
			for _, part := range []string{
				"dashboard_local_account_candidates as materialized", "dashboard_supplier_account_candidates as materialized",
				"dashboard_unique_local_account_ids as materialized", "dashboard_forward_match_counts", "dashboard_reverse_match_counts",
				"forward.forward_match_count = 1", "reverse_match.reverse_match_count = 1",
				"join dashboard_unique_local_account_ids unique_accounts", "usage_logs.created_at >= $1", "usage_logs.created_at < $2",
				"oel.created_at >= $1", "oel.created_at < $2", "coalesce(oel.status_code, 0) >= 400", "count(*) as success_count", "count(*) as error_count",
			} {
				if !strings.Contains(lower, part) {
					t.Fatalf("SQL missing shared fact %q", part)
				}
			}
			if strings.Contains(lower, "lateral") || strings.Contains(lower, "actual_cost > 0") || strings.Contains(lower, "as success_rate") {
				t.Fatalf("SQL contains forbidden derived/correlated logic: %s", query)
			}
			assertDashboardSQLSafe(t, query)
		})
	}
}

func TestSupplierDashboardRepositoryPushesProviderAndGroupFiltersWithoutHidingConflicts(t *testing.T) {
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	providerSlug := "target-provider"
	groupKey := "target-group"

	for name, query := range map[string]string{"accounts": supplierDashboardAccountsQuery, "rates": supplierDashboardRatesQuery} {
		t.Run(name+" sql", func(t *testing.T) {
			lower := strings.ToLower(query)
			for _, part := range []string{
				"dashboard_target_supplier_accounts as materialized",
				"($3 = '' or target_provider.code = $3)",
				"($4 = '' or spa.group_key = $4)",
				"dashboard_target_normalized_names as materialized",
				"join dashboard_target_normalized_names",
			} {
				if !strings.Contains(lower, part) {
					t.Fatalf("%s SQL missing safe filter structure %q", name, part)
				}
			}

			supplierCandidatesStart := strings.Index(lower, "dashboard_supplier_account_candidates as materialized")
			forwardStart := strings.Index(lower, "dashboard_forward_match_counts as")
			if supplierCandidatesStart < 0 || forwardStart <= supplierCandidatesStart {
				t.Fatalf("%s SQL missing supplier conflict candidate section", name)
			}
			supplierCandidates := lower[supplierCandidatesStart:forwardStart]
			if strings.Contains(supplierCandidates, "sp.code = $3") || strings.Contains(supplierCandidates, "spa.group_key = $4") {
				t.Fatalf("%s SQL filters reverse-conflict candidates to the target provider/group: %s", name, supplierCandidates)
			}
		})
	}

	repo, mock, _ := newDashboardRepoMock(t)
	mock.ExpectQuery("(?s)FROM supplier_provider_accounts").
		WithArgs(start, end, providerSlug, groupKey).
		WillReturnRows(sqlmock.NewRows(dashboardAccountColumns))
	if _, err := repo.ListDashboardAccounts(context.Background(), start, end, providerSlug, groupKey); err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("(?s)FROM supplier_provider_accounts").
		WithArgs(start, end, providerSlug, groupKey).
		WillReturnRows(sqlmock.NewRows(dashboardRateColumns))
	if _, err := repo.ListDashboardRates(context.Background(), start, end, providerSlug, groupKey); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSupplierDashboardRepositoryFilterTargetsPreserveInactiveAccountsAndEmptyProviders(t *testing.T) {
	query := strings.ToLower(supplierDashboardProvidersQuery)
	for _, part := range []string{
		"dashboard_target_providers as materialized",
		"from supplier_providers target_provider",
		"target_provider.deleted_at is null",
		"dashboard_target_active_supplier_accounts as materialized",
	} {
		if !strings.Contains(query, part) {
			t.Fatalf("provider SQL missing target preservation structure %q", part)
		}
	}
	targetStart := strings.Index(query, "dashboard_target_supplier_accounts as materialized")
	activeStart := strings.Index(query, "dashboard_target_active_supplier_accounts as materialized")
	if targetStart < 0 || activeStart <= targetStart {
		t.Fatal("target supplier account CTE boundaries not found")
	}
	if strings.Contains(query[targetStart:activeStart], "spa.active = true") {
		t.Fatal("target supplier accounts exclude inactive neutral rows")
	}
	forwardStart := strings.Index(query, "dashboard_forward_match_counts as")
	if forwardStart < 0 || !strings.Contains(query[forwardStart:], "from dashboard_target_active_supplier_accounts target") {
		t.Fatal("unique matching must only attribute active target accounts")
	}
}

func TestSupplierDashboardMigrationsSeparateTransactionalSchemaAndConcurrentIndexes(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "migrations", "192_supplier_dashboard_query_schema.sql")
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	schemaSQL := strings.ToLower(string(schemaContent))
	for _, part := range []string{
		"add column if not exists supplier_dashboard_normalized_effective_name text",
		"supplier_dashboard_refresh_account_normalized_name",
		"before insert or update of provider_id, name, active",
		"supplier_dashboard_refresh_provider_account_names",
		"after update of account_name_prefix, deleted_at",
		"add column if not exists risk_updated_at timestamptz",
		"supplier_dashboard_track_provider_risk_updated_at",
		"before insert or update of risk_level, rate_risk_count",
	} {
		if !strings.Contains(schemaSQL, part) {
			t.Fatalf("supplier dashboard schema migration missing %q: %s", part, schemaContent)
		}
	}
	if strings.Contains(schemaSQL, "concurrently") {
		t.Fatalf("transactional schema migration must not contain CONCURRENTLY: %s", schemaContent)
	}
	if nonTx, err := validateMigrationExecutionMode(filepath.Base(schemaPath), string(schemaContent)); err != nil || nonTx {
		t.Fatalf("schema migration execution mode nonTx=%v err=%v", nonTx, err)
	}

	indexPath := filepath.Join("..", "..", "migrations", "193_supplier_dashboard_query_indexes_notx.sql")
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	indexSQL := strings.ToLower(string(indexContent))
	for _, part := range []string{
		"create index concurrently if not exists idx_supplier_dashboard_accounts_normalized_name",
		"regexp_replace(lower(name), '[^[:alnum:]]', '', 'g')",
		"where deleted_at is null",
		"create index concurrently if not exists idx_supplier_dashboard_supplier_accounts_normalized_active",
		"(supplier_dashboard_normalized_effective_name, id)",
		"where active = true",
		"create index concurrently if not exists idx_supplier_dashboard_supplier_accounts_provider_group",
		"(provider_id, group_key, id)",
		"create index concurrently if not exists idx_supplier_dashboard_health_items_account_finished",
		"(account_id, finished_at desc, id desc)",
		"create index concurrently if not exists idx_supplier_dashboard_sync_runs_provider_scope_finished",
		"(provider_id, sync_scope, finished_at desc, id desc)",
		"where finished_at is not null",
		"create index concurrently if not exists idx_supplier_dashboard_rate_changes_mapping_time",
		"(mapping_id, changed_at desc, id desc)",
	} {
		if !strings.Contains(indexSQL, part) {
			t.Fatalf("supplier dashboard concurrent index migration missing %q: %s", part, indexContent)
		}
	}
	if nonTx, err := validateMigrationExecutionMode(filepath.Base(indexPath), string(indexContent)); err != nil || !nonTx {
		t.Fatalf("index migration execution mode nonTx=%v err=%v", nonTx, err)
	}
}

func TestSupplierDashboardRepositoryUsesPersistedSupplierNormalizationAndDedicatedRiskTime(t *testing.T) {
	for name, query := range map[string]string{
		"accounts":  supplierDashboardAccountsQuery,
		"rates":     supplierDashboardRatesQuery,
		"providers": supplierDashboardProvidersQuery,
	} {
		t.Run(name, func(t *testing.T) {
			lower := strings.ToLower(query)
			if strings.Contains(lower, "regexp_replace(lower(sp.account_name_prefix || spa.name)") {
				t.Fatalf("%s SQL still normalizes supplier accounts with regexp at query time", name)
			}
			for _, part := range []string{
				"spa.supplier_dashboard_normalized_effective_name as normalized_name",
				"target_name.normalized_name = spa.supplier_dashboard_normalized_effective_name",
			} {
				if !strings.Contains(lower, part) {
					t.Fatalf("%s SQL missing persisted supplier normalized key access %q", name, part)
				}
			}
		})
	}

	accounts := strings.ToLower(supplierDashboardAccountsQuery)
	if !strings.Contains(accounts, "runtime.risk_updated_at as provider_risk_updated_at") {
		t.Fatalf("account SQL does not use dedicated provider risk time: %s", supplierDashboardAccountsQuery)
	}
	if strings.Contains(accounts, "runtime.updated_at as provider_risk_updated_at") {
		t.Fatal("account SQL still aliases generic runtime updated_at as provider risk time")
	}
}

func TestSupplierDashboardRepositoryTaskAndCollectionQueriesUseStableEvidence(t *testing.T) {
	accounts := strings.ToLower(supplierDashboardAccountsQuery)
	for _, part := range []string{
		"dashboard_latest_task_facts as materialized", "upstream_account_health_guard_run_items", "item.finished_at >= $1", "item.finished_at < $2",
		"rate_change.old_rate as previous_rate", "coalesce(rate_change.change_count, 0) + 1 as snapshot_count", "spa.last_rate_sync_at",
		"dashboard_latest_group_runs", "sync_scope = 'groups'", "dashboard_latest_balance_runs", "sync_scope = 'balance'",
		"then case when group_run.status in ('success', 'failed') then group_run.status else 'unknown' end",
	} {
		if !strings.Contains(accounts, part) {
			t.Fatalf("account SQL missing stable evidence %q", part)
		}
	}
	if strings.Contains(accounts, "as rate_delta_percent") {
		t.Fatal("repository still derives rate delta")
	}
	providers := strings.ToLower(supplierDashboardProvidersQuery)
	for _, part := range []string{
		"enabled_account_count", "schedulable_account_count", "local_account.status = 'active'", "local_account.schedulable = true",
		"provider_ops.matched_account_count <> provider_ops.enabled_account_count", "evidence.group_finished_at", "as last_synced_at",
	} {
		if !strings.Contains(providers, part) {
			t.Fatalf("provider SQL missing neutral aggregate %q", part)
		}
	}
}

func TestSupplierDashboardRateQueriesUseCommittedGroupChangeLogs(t *testing.T) {
	for name, query := range map[string]string{"accounts": supplierDashboardAccountsQuery, "rates": supplierDashboardRatesQuery} {
		t.Run(name, func(t *testing.T) {
			lower := strings.ToLower(query)
			for _, part := range []string{
				"supplier_rate_guard_change_logs", "join supplier_provider_groups", "history.mapping_id",
				"mapping.provider_id", "history.upstream_group_key", "history.old_rate", "history.new_rate",
				"history.changed_at >= $1", "history.changed_at < $2", "dashboard_rate_changes",
				"left join dashboard_rate_changes", "rate_change.provider_id = spa.provider_id", "rate_change.group_key = spa.group_key",
			} {
				if !strings.Contains(lower, part) {
					t.Fatalf("SQL missing committed group change fact %q", part)
				}
			}
			for _, forbidden := range []string{"supplier_account_rate_guard_unbind_logs", "dashboard_rate_observations", "effective_upstream_rate"} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("SQL still uses pseudo generic rate history %q", forbidden)
				}
			}
			eventsStart := strings.Index(lower, "dashboard_rate_change_events as materialized (")
			eventsEnd := strings.Index(lower, "dashboard_rate_changes as (")
			if eventsStart < 0 || eventsEnd <= eventsStart {
				t.Fatal("rate change event CTE boundaries not found")
			}
			events := lower[eventsStart:eventsEnd]
			targetPos := strings.Index(events, "from dashboard_target_provider_groups target_group")
			mappingPos := strings.Index(events, "join supplier_provider_groups mapping")
			historyPos := strings.Index(events, "join supplier_rate_guard_change_logs history")
			if targetPos < 0 || mappingPos <= targetPos || historyPos <= mappingPos {
				t.Fatalf("rate history is not driven from target provider/group mappings: %s", events)
			}
			if !strings.Contains(events, "history.mapping_id = mapping.id") {
				t.Fatalf("rate history does not use mapping_id lookup: %s", events)
			}
		})
	}
}

func TestSupplierDashboardRateChangeEventsUseRequestedWindow(t *testing.T) {
	for name, query := range map[string]string{"accounts": supplierDashboardAccountsQuery, "rates": supplierDashboardRatesQuery} {
		t.Run(name, func(t *testing.T) {
			lower := strings.ToLower(query)
			start := strings.Index(lower, "dashboard_rate_change_events as materialized (")
			end := strings.Index(lower, "dashboard_rate_changes as (")
			if start < 0 || end <= start {
				t.Fatalf("rate change event CTE not found in SQL: %s", query)
			}
			observations := lower[start:end]
			for _, predicate := range []string{"history.changed_at >= $1", "history.changed_at < $2"} {
				if !strings.Contains(observations, predicate) {
					t.Fatalf("rate change event CTE missing window predicate %q: %s", predicate, observations)
				}
			}
		})
	}
}
func TestSupplierDashboardRepositoryPropagatesQueryRowsAndScanErrors(t *testing.T) {
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	cases := []struct {
		name, pattern string
		columns       []string
		good, bad     []driver.Value
		call          func(service.SupplierDashboardDetailRepository) error
	}{
		{"accounts", "(?s)FROM supplier_provider_accounts", dashboardAccountColumns,
			[]driver.Value{int64(1), "a", "p", "P", true, true, "g", "G", "normal", nil, "active", "success", "never", nil, nil, nil, nil, nil, nil, 1, nil, nil, 0, nil, nil, nil, nil, nil, nil, nil, end},
			[]driver.Value{"bad-id", "a", "p", "P", true, true, "g", "G", "normal", nil, "active", "success", "never", nil, nil, nil, nil, nil, nil, 1, nil, nil, 0, nil, nil, nil, nil, nil, nil, nil, end},
			func(r service.SupplierDashboardDetailRepository) error {
				_, e := r.ListDashboardAccounts(context.Background(), start, end, "", "")
				return e
			}},
		{"rates", "(?s)FROM supplier_provider_accounts", dashboardRateColumns,
			[]driver.Value{int64(1), "a", "p", "P", true, true, "g", "G", nil, nil, 1, nil, nil, 0, nil, nil, nil, nil, nil, end},
			[]driver.Value{"bad-id", "a", "p", "P", true, true, "g", "G", nil, nil, 1, nil, nil, 0, nil, nil, nil, nil, nil, end},
			func(r service.SupplierDashboardDetailRepository) error {
				_, e := r.ListDashboardRates(context.Background(), start, end, "", "")
				return e
			}},
		{"providers", "(?s)FROM supplier_providers", dashboardProviderColumns,
			[]driver.Value{"p", "P", true, false, "normal", "never", "never", "never", 0, 0, 0, int64(0), int64(0), nil, nil, 0.0, nil},
			[]driver.Value{"p", "P", "bad-bool", false, "normal", "never", "never", "never", 0, 0, 0, int64(0), int64(0), nil, nil, 0.0, nil},
			func(r service.SupplierDashboardDetailRepository) error {
				_, e := r.ListDashboardProviders(context.Background(), start, end)
				return e
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" query", func(t *testing.T) {
			repo, mock, _ := newDashboardRepoMock(t)
			want := errors.New("query failed")
			mock.ExpectQuery(tc.pattern).WithArgs(start, end, "", "").WillReturnError(want)
			if !errors.Is(tc.call(repo), want) {
				t.Fatal("query error was not propagated")
			}
		})
		t.Run(tc.name+" rows", func(t *testing.T) {
			repo, mock, _ := newDashboardRepoMock(t)
			want := errors.New("rows failed")
			mock.ExpectQuery(tc.pattern).WithArgs(start, end, "", "").WillReturnRows(sqlmock.NewRows(tc.columns).AddRow(tc.good...).RowError(0, want))
			if !errors.Is(tc.call(repo), want) {
				t.Fatal("rows error was not propagated")
			}
		})
		t.Run(tc.name+" scan", func(t *testing.T) {
			repo, mock, _ := newDashboardRepoMock(t)
			mock.ExpectQuery(tc.pattern).WithArgs(start, end, "", "").WillReturnRows(sqlmock.NewRows(tc.columns).AddRow(tc.bad...))
			if err := tc.call(repo); err == nil || !strings.Contains(err.Error(), "scan supplier dashboard") {
				t.Fatalf("scan error = %v", err)
			}
		})
	}
}

func newDashboardRepoMock(t *testing.T) (service.SupplierDashboardDetailRepository, sqlmock.Sqlmock, *[]string) {
	t.Helper()
	seen := []string{}
	matcher := sqlmock.QueryMatcherFunc(func(expected, actual string) error {
		seen = append(seen, actual)
		ok, err := regexp.MatchString(expected, actual)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("SQL %q did not match %q", actual, expected)
		}
		return nil
	})
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewSupplierDashboardRepository(db), mock, &seen
}

func assertDashboardSQLSafe(t *testing.T, query string) {
	t.Helper()
	lower := strings.ReplaceAll(strings.ToLower(query), "is_count_tokens", "")
	for _, pattern := range []string{`\bemail\b`, `\busername\b`, `password`, `\btoken\b`, `secret`, `cookie`, `credential`, `authorization`, `bearer`, `\b(?:api|access|refresh|private|public|secret)_?key\b`} {
		if regexp.MustCompile(pattern).FindStringIndex(lower) != nil {
			t.Fatalf("sensitive SQL field pattern %q: %s", pattern, query)
		}
	}
}

func TestSupplierDashboardCollectionEvidenceScansSyncRunsOnce(t *testing.T) {
	for name, query := range map[string]string{"accounts": supplierDashboardAccountsQuery, "providers": supplierDashboardProvidersQuery} {
		lower := strings.ToLower(query)
		if got := strings.Count(lower, "from supplier_provider_sync_runs run"); got != 1 {
			t.Fatalf("%s scans supplier_provider_sync_runs %d times", name, got)
		}
		for _, scope := range []string{"accounts", "groups", "balance", "all"} {
			if !strings.Contains(lower, "sync_scope = '"+scope+"'") {
				t.Fatalf("%s missing scope %s", name, scope)
			}
		}
	}
}
