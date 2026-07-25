//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestSupplierDashboardRepositoryIntegrationAggregatesUniqueMatchesAndWindow(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	userID := dashboardInsertID(t, tx, `INSERT INTO users (email, password_hash) VALUES ($1, 'hash') RETURNING id`, "dashboard-"+suffix+"@example.com")
	apiKeyID := dashboardInsertID(t, tx, `INSERT INTO api_keys (user_id, key, name) VALUES ($1, $2, 'dashboard') RETURNING id`, userID, "sk-dashboard-"+suffix)
	groupOneID := dashboardInsertID(t, tx, `INSERT INTO groups (name) VALUES ($1) RETURNING id`, "dashboard-group-one-"+suffix)
	groupTwoID := dashboardInsertID(t, tx, `INSERT INTO groups (name) VALUES ($1) RETURNING id`, "dashboard-group-two-"+suffix)

	localUniqueID := dashboardInsertLocalAccount(t, tx, "dashunique"+suffix)
	localIdleID := dashboardInsertLocalAccount(t, tx, "dashidle"+suffix)
	localConflictID := dashboardInsertLocalAccount(t, tx, "crossconflict"+suffix)
	localSameProviderID := dashboardInsertLocalAccount(t, tx, "samesame"+suffix)
	_ = localIdleID
	_ = localConflictID
	_ = localSameProviderID
	_, err := tx.ExecContext(ctx, `INSERT INTO account_groups (account_id, group_id) VALUES ($1,$2),($1,$3)`, localUniqueID, groupOneID, groupTwoID)
	if err != nil {
		t.Fatal(err)
	}

	providerUniqueID := dashboardInsertProvider(t, tx, "dash-unique-"+suffix, "dash", true)
	providerCrossID := dashboardInsertProvider(t, tx, "dash-cross-"+suffix, "cross", true)
	providerSameID := dashboardInsertProvider(t, tx, "dash-same-"+suffix, "same", true)
	providerDefaultID := dashboardInsertProvider(t, tx, "dash-default-"+suffix, "cross", true)

	uniqueSupplierID := dashboardInsertSupplierAccount(t, tx, providerUniqueID, "unique"+suffix, "g-unique")
	idleSupplierID := dashboardInsertSupplierAccount(t, tx, providerUniqueID, "idle"+suffix, "g-idle")
	crossSupplierID := dashboardInsertSupplierAccount(t, tx, providerCrossID, "conflict"+suffix, "g-cross")
	sameFirstConflictID := dashboardInsertSupplierAccount(t, tx, providerSameID, "same"+suffix, "g-same")
	sameConflictID := dashboardInsertSupplierAccount(t, tx, providerSameID, "s-a-m-e"+suffix, "g-same")
	defaultConflictID := dashboardInsertSupplierAccount(t, tx, providerDefaultID, "conflict"+suffix, "g-default")

	_, err = tx.ExecContext(ctx, `
UPDATE supplier_provider_runtime_stats
SET current_balance=0, estimated_days=0, rate_risk_count=2,
    risk_level='warning', sync_status='failed', group_sync_status='success',
    last_sync_at=$2, last_group_sync_at=$2, updated_at=$2
WHERE provider_id=$1`, providerUniqueID, end.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	dashboardInsertSyncRun(t, tx, providerUniqueID, "accounts", "success", end.Add(-3*time.Hour))
	dashboardInsertSyncRun(t, tx, providerUniqueID, "groups", "success", end.Add(-150*time.Minute))
	dashboardInsertSyncRun(t, tx, providerUniqueID, "balance", "success", end.Add(-2*time.Hour))
	uniqueRateMappingID := dashboardInsertRateMapping(t, tx, providerUniqueID, "g-unique", suffix)
	dashboardInsertRateChange(t, tx, uniqueRateMappingID, 0.7, 0.8, end.Add(-4*time.Hour))
	dashboardInsertRateChange(t, tx, uniqueRateMappingID, 0.8, 1, end.Add(-90*time.Minute))

	dashboardInsertUsage(t, tx, userID, apiKeyID, localUniqueID, "start", 0, start)
	dashboardInsertUsage(t, tx, userID, apiKeyID, localUniqueID, "end", 99, end)
	dashboardInsertError(t, tx, localUniqueID, 500, false, start.Add(time.Hour))
	dashboardInsertError(t, tx, localUniqueID, 429, true, start.Add(2*time.Hour))

	accounts, err := repo.ListDashboardAccounts(ctx, start, end, "", "")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[int64]struct {
		success *int64
		errors  *int64
		cost    *float64
	})
	for _, item := range accounts {
		byID[item.AccountID] = struct {
			success *int64
			errors  *int64
			cost    *float64
		}{item.SuccessCount, item.ErrorCount, item.PeriodCost}
	}
	unique := byID[uniqueSupplierID]
	if unique.success == nil || *unique.success != 1 || unique.errors == nil || *unique.errors != 1 || unique.cost == nil || *unique.cost != 0 {
		t.Fatalf("unique account facts = %+v", unique)
	}
	var uniqueAccountFound bool
	for _, item := range accounts {
		if item.AccountID != uniqueSupplierID {
			continue
		}
		uniqueAccountFound = true
		if item.PreviousRate == nil || math.Abs(*item.PreviousRate-0.8) > 1e-9 || item.CurrentRate == nil || math.Abs(*item.CurrentRate-1) > 1e-9 || item.Balance == nil || *item.Balance != 0 || item.EstimatedDays == nil || *item.EstimatedDays != 0 {
			t.Fatalf("unique account production risk facts = %+v", item)
		}
	}
	if !uniqueAccountFound {
		t.Fatalf("target account %d was not returned", uniqueSupplierID)
	}
	idle := byID[idleSupplierID]
	if idle.success == nil || *idle.success != 0 || idle.errors == nil || *idle.errors != 0 || idle.cost == nil || *idle.cost != 0 {
		t.Fatalf("idle account facts = %+v", idle)
	}
	for _, id := range []int64{crossSupplierID, defaultConflictID, sameFirstConflictID, sameConflictID} {
		item := byID[id]
		if item.success != nil || item.errors != nil || item.cost != nil {
			t.Fatalf("conflicted account %d was attributed: %+v", id, item)
		}
	}
	filteredConflict, err := repo.ListDashboardAccounts(ctx, start, end, "dash-cross-"+suffix, "g-cross")
	if err != nil {
		t.Fatal(err)
	}
	if len(filteredConflict) != 1 || filteredConflict[0].AccountID != crossSupplierID || filteredConflict[0].SuccessCount != nil || filteredConflict[0].ErrorCount != nil || filteredConflict[0].PeriodCost != nil {
		t.Fatalf("filtered cross-provider conflict = %+v", filteredConflict)
	}

	rates, err := repo.ListDashboardRates(ctx, start, end, "", "")
	if err != nil {
		t.Fatal(err)
	}
	rateFound := false
	for _, item := range rates {
		if item.AccountID == uniqueSupplierID {
			rateFound = true
			if item.SnapshotCount != 3 || item.PreviousRate == nil || math.Abs(*item.PreviousRate-0.8) > 1e-9 || item.SuccessCount == nil || *item.SuccessCount != 1 || item.ErrorCount == nil || *item.ErrorCount != 1 || item.PeriodCost == nil || *item.PeriodCost != 0 {
				t.Fatalf("rate facts = %+v", item)
			}
		}
	}
	if !rateFound {
		t.Fatalf("target rate row %d was not returned", uniqueSupplierID)
	}

	providers, err := repo.ListDashboardProviders(ctx, start, end)
	if err != nil {
		t.Fatal(err)
	}
	providerBySlug := make(map[string]serviceProviderFacts)
	for _, item := range providers {
		providerBySlug[item.ProviderSlug] = serviceProviderFacts{
			complete: item.DataComplete, success: item.SuccessCount, errors: item.ErrorCount,
			balance: item.Balance, days: item.EstimatedDays, cost: item.PeriodCost, rateRisks: item.RateRiskCount,
		}
	}
	uniqueProvider := providerBySlug["dash-unique-"+suffix]
	if !uniqueProvider.complete || uniqueProvider.success == nil || *uniqueProvider.success != 1 || uniqueProvider.errors == nil || *uniqueProvider.errors != 1 || uniqueProvider.cost == nil || *uniqueProvider.cost != 0 || uniqueProvider.balance == nil || *uniqueProvider.balance != 0 || uniqueProvider.days == nil || *uniqueProvider.days != 0 || uniqueProvider.rateRisks != 2 {
		t.Fatalf("unique provider facts = %+v", uniqueProvider)
	}
	defaultProvider := providerBySlug["dash-default-"+suffix]
	if defaultProvider.complete || defaultProvider.success != nil || defaultProvider.errors != nil || defaultProvider.balance != nil || defaultProvider.days != nil || defaultProvider.cost != nil {
		t.Fatalf("default runtime row was treated as collected: %+v", defaultProvider)
	}
}

func TestSupplierDashboardRepositoryIntegrationRateChangesUseRequestedWindow(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	providerID := dashboardInsertProvider(t, tx, "dash-rate-window-"+suffix, "ratewindow", true)
	startID := dashboardInsertSupplierAccount(t, tx, providerID, "start"+suffix, "g-start")
	endID := dashboardInsertSupplierAccount(t, tx, providerID, "end"+suffix, "g-end")
	outsideID := dashboardInsertSupplierAccount(t, tx, providerID, "outside"+suffix, "g-outside")
	downID := dashboardInsertSupplierAccount(t, tx, providerID, "down"+suffix, "g-down")
	upID := dashboardInsertSupplierAccount(t, tx, providerID, "up"+suffix, "g-up")
	noneID := dashboardInsertSupplierAccount(t, tx, providerID, "none"+suffix, "g-none")

	startMapping := dashboardInsertRateMapping(t, tx, providerID, "g-start", suffix)
	endMapping := dashboardInsertRateMapping(t, tx, providerID, "g-end", suffix)
	outsideMapping := dashboardInsertRateMapping(t, tx, providerID, "g-outside", suffix)
	downMapping := dashboardInsertRateMapping(t, tx, providerID, "g-down", suffix)
	upMapping := dashboardInsertRateMapping(t, tx, providerID, "g-up", suffix)
	_ = dashboardInsertRateMapping(t, tx, providerID, "g-none", suffix)

	dashboardInsertRateChange(t, tx, startMapping, 0.9, 0.8, start)
	dashboardInsertRateChange(t, tx, endMapping, 0.8, 0.7, end)
	dashboardInsertRateChange(t, tx, outsideMapping, 0.7, 0.6, start.Add(-time.Second))
	dashboardInsertRateChange(t, tx, downMapping, 0.9, 0.8, start.Add(time.Hour))
	dashboardInsertRateChange(t, tx, upMapping, 0.8, 0.9, start.Add(2*time.Hour))
	dashboardSetSupplierAccountRate(t, tx, downID, 0.8)
	dashboardSetSupplierAccountRate(t, tx, upID, 0.9)

	rates, err := repo.ListDashboardRates(ctx, start, end, "", "")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[int64]service.SupplierDashboardRateSnapshot, len(rates))
	for _, item := range rates {
		byID[item.AccountID] = item
	}
	assertChange := func(accountID int64, oldRate, newRate *float64, count int, changedAt *time.Time) {
		t.Helper()
		item, ok := byID[accountID]
		if !ok {
			t.Fatalf("rate account %d was not returned", accountID)
		}
		if item.RateChangeCount != count || !dashboardEqualFloatPointer(item.RateChangeOld, oldRate) || !dashboardEqualFloatPointer(item.RateChangeNew, newRate) || !dashboardEqualTimePointer(item.RateChangedAt, changedAt) {
			t.Fatalf("rate account %d change=%+v, want old=%v new=%v count=%d at=%v", accountID, item, oldRate, newRate, count, changedAt)
		}
	}
	assertChange(startID, dashboardFloatPointer(0.9), dashboardFloatPointer(0.8), 1, &start)
	assertChange(endID, nil, nil, 0, nil)
	assertChange(outsideID, nil, nil, 0, nil)
	assertChange(downID, dashboardFloatPointer(0.9), dashboardFloatPointer(0.8), 1, dashboardTimePointer(start.Add(time.Hour)))
	assertChange(upID, dashboardFloatPointer(0.8), dashboardFloatPointer(0.9), 1, dashboardTimePointer(start.Add(2*time.Hour)))
	assertChange(noneID, nil, nil, 0, nil)
	if byID[upID].CurrentRate == nil || math.Abs(*byID[upID].CurrentRate-0.9) > 1e-9 {
		t.Fatalf("updated current rate not returned with change event: %+v", byID[upID])
	}
}

func TestSupplierDashboardRepositoryIntegrationGroupChangesDriveChangedAndRateUpViews(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	now := time.Now().UTC()
	suffix := fmt.Sprintf("%d", now.UnixNano())

	stableProviderID := dashboardInsertProvider(t, tx, "dash-rate-stable-"+suffix, "stable", true)
	downProviderID := dashboardInsertProvider(t, tx, "dash-rate-down-"+suffix, "down", true)
	upProviderID := dashboardInsertProvider(t, tx, "dash-rate-up-"+suffix, "up", true)
	outsideProviderID := dashboardInsertProvider(t, tx, "dash-rate-outside-"+suffix, "outside", true)
	stableID := dashboardInsertSupplierAccount(t, tx, stableProviderID, "account"+suffix, "g")
	downID := dashboardInsertSupplierAccount(t, tx, downProviderID, "account"+suffix, "g")
	upID := dashboardInsertSupplierAccount(t, tx, upProviderID, "account"+suffix, "g")
	outsideID := dashboardInsertSupplierAccount(t, tx, outsideProviderID, "account"+suffix, "g")
	_ = stableID
	_ = outsideID

	_ = dashboardInsertRateMapping(t, tx, stableProviderID, "g", suffix)
	downMapping := dashboardInsertRateMapping(t, tx, downProviderID, "g", suffix)
	upMapping := dashboardInsertRateMapping(t, tx, upProviderID, "g", suffix)
	outsideMapping := dashboardInsertRateMapping(t, tx, outsideProviderID, "g", suffix)
	dashboardInsertRateChange(t, tx, downMapping, 0.9, 0.8, now.Add(-2*time.Hour))
	dashboardInsertRateChange(t, tx, upMapping, 0.8, 0.9, now.Add(-time.Hour))
	dashboardInsertRateChange(t, tx, outsideMapping, 0.5, 0.6, now.Add(-25*time.Hour))
	dashboardSetSupplierAccountRate(t, tx, downID, 0.8)
	dashboardSetSupplierAccountRate(t, tx, upID, 0.9)

	svc := service.NewSupplierDashboardService(repo, nil)
	changed, err := svc.GetRates(ctx, service.SupplierDashboardRatesQuery{Range: service.SupplierDashboardRange24Hours, View: service.SupplierDashboardRateViewChanged})
	if err != nil {
		t.Fatal(err)
	}
	changedProviders := map[string]bool{}
	for _, item := range changed.Items {
		changedProviders[item.ProviderSlug] = true
	}
	for _, provider := range []string{"dash-rate-down-" + suffix, "dash-rate-up-" + suffix} {
		if !changedProviders[provider] {
			t.Fatalf("provider %q missing from changed view: %+v", provider, changed.Items)
		}
	}
	for _, provider := range []string{"dash-rate-stable-" + suffix, "dash-rate-outside-" + suffix} {
		if changedProviders[provider] {
			t.Fatalf("provider %q unexpectedly in changed view: %+v", provider, changed.Items)
		}
	}

	risk, err := svc.GetRates(ctx, service.SupplierDashboardRatesQuery{Range: service.SupplierDashboardRange24Hours, View: service.SupplierDashboardRateViewRisk, ProviderSlug: "dash-rate-up-" + suffix})
	if err != nil || len(risk.Items) != 1 {
		t.Fatalf("upward event risk view=%+v err=%v", risk.Items, err)
	}
	accounts, err := svc.GetAccounts(ctx, service.SupplierDashboardAccountsQuery{Range: service.SupplierDashboardRange24Hours, RiskType: service.SupplierDashboardRiskTypeRateUp, ProviderSlug: "dash-rate-up-" + suffix})
	if err != nil || len(accounts.Items) != 1 || accounts.Items[0].AccountID != upID {
		t.Fatalf("upward event account risk=%+v err=%v", accounts.Items, err)
	}
}
func TestSupplierDashboardRepositoryIntegrationUsesIndependentCollectionEvidence(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	accountsOnlyID := dashboardInsertProvider(t, tx, "dash-accounts-only-"+suffix, "accounts", true)
	balanceOnlyID := dashboardInsertProvider(t, tx, "dash-balance-only-"+suffix, "balance", true)
	accountFailedID := dashboardInsertProvider(t, tx, "dash-account-failed-"+suffix, "failed", true)
	groupFailedID := dashboardInsertProvider(t, tx, "dash-group-failed-"+suffix, "group", true)
	allPartialID := dashboardInsertProvider(t, tx, "dash-all-partial-"+suffix, "partial", true)
	defaultID := dashboardInsertProvider(t, tx, "dash-never-"+suffix, "never", true)

	dashboardUpdateRuntimeCollectionFacts(t, tx, accountsOnlyID, 11, 5, "success", end.Add(-time.Hour))
	dashboardInsertSyncRun(t, tx, accountsOnlyID, "accounts", "success", end.Add(-2*time.Hour))

	dashboardUpdateRuntimeCollectionFacts(t, tx, balanceOnlyID, 12, 6, "success", end.Add(-time.Hour))
	dashboardInsertSyncRun(t, tx, balanceOnlyID, "balance", "success", end.Add(-2*time.Hour))

	dashboardUpdateRuntimeCollectionFacts(t, tx, accountFailedID, 13, 7, "success", end.Add(-time.Hour))
	dashboardInsertSyncRun(t, tx, accountFailedID, "all", "success", end.Add(-3*time.Hour))
	dashboardInsertSyncRun(t, tx, accountFailedID, "accounts", "failed", end.Add(-time.Hour))

	dashboardUpdateRuntimeCollectionFacts(t, tx, groupFailedID, 14, 8, "failed", end.Add(-time.Hour))
	dashboardInsertSyncRun(t, tx, groupFailedID, "accounts", "success", end.Add(-3*time.Hour))
	dashboardInsertSyncRun(t, tx, groupFailedID, "balance", "success", end.Add(-2*time.Hour))
	dashboardInsertSyncRun(t, tx, groupFailedID, "groups", "failed", end.Add(-time.Hour))

	dashboardUpdateRuntimeCollectionFacts(t, tx, allPartialID, 15, 9, "success", end.Add(-time.Hour))
	dashboardInsertSyncRun(t, tx, allPartialID, "all", "partial", end.Add(-2*time.Hour))

	providers, err := repo.ListDashboardProviders(ctx, start, end)
	if err != nil {
		t.Fatal(err)
	}
	bySlug := make(map[string]serviceProviderFacts)
	for _, item := range providers {
		bySlug[item.ProviderSlug] = serviceProviderFacts{
			complete:    item.DataComplete,
			balance:     item.Balance,
			days:        item.EstimatedDays,
			sync:        item.SyncStatus,
			group:       item.GroupSyncStatus,
			balanceSync: item.BalanceSyncStatus,
		}
	}
	assertProvider := func(slug string, complete bool, syncStatus, groupStatus, balanceStatus string, balance, days *float64) {
		t.Helper()
		item, ok := bySlug[slug]
		if !ok {
			t.Fatalf("provider %q was not returned", slug)
		}
		if item.complete != complete || item.sync != syncStatus || item.group != groupStatus || item.balanceSync != balanceStatus || !dashboardEqualFloatPointer(item.balance, balance) || !dashboardEqualFloatPointer(item.days, days) {
			t.Fatalf("provider %q facts = %+v, want complete=%v sync=%q group=%q balance_sync=%q balance=%v days=%v", slug, item, complete, syncStatus, groupStatus, balanceStatus, balance, days)
		}
	}

	assertProvider("dash-accounts-only-"+suffix, false, "success", "never", "never", nil, nil)
	assertProvider("dash-balance-only-"+suffix, false, "never", "never", "success", dashboardFloatPointer(12), dashboardFloatPointer(6))
	assertProvider("dash-account-failed-"+suffix, false, "failed", "success", "success", dashboardFloatPointer(13), dashboardFloatPointer(7))
	assertProvider("dash-group-failed-"+suffix, false, "success", "failed", "success", dashboardFloatPointer(14), dashboardFloatPointer(8))
	assertProvider("dash-all-partial-"+suffix, false, "unknown", "unknown", "unknown", nil, nil)
	assertProvider("dash-never-"+suffix, false, "never", "never", "never", nil, nil)

	_ = defaultID
}

type serviceProviderFacts struct {
	complete    bool
	success     *int64
	errors      *int64
	balance     *float64
	days        *float64
	cost        *float64
	rateRisks   int
	sync        string
	group       string
	balanceSync string
}

func dashboardInsertID(t *testing.T, tx *sql.Tx, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRowContext(context.Background(), query, args...).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func dashboardInsertLocalAccount(t *testing.T, tx *sql.Tx, name string) int64 {
	t.Helper()
	return dashboardInsertID(t, tx, `INSERT INTO accounts (name, platform, type) VALUES ($1, 'anthropic', 'oauth') RETURNING id`, name)
}

func dashboardInsertProvider(t *testing.T, tx *sql.Tx, code, prefix string, enabled bool) int64 {
	t.Helper()
	id := dashboardInsertID(t, tx, `
INSERT INTO supplier_providers (code, name, provider_type, base_url, account_name_prefix, enabled)
VALUES ($1, $1, 'sub2api', 'https://example.invalid', $2, $3)
RETURNING id`, code, prefix, enabled)
	if _, err := tx.ExecContext(context.Background(), `INSERT INTO supplier_provider_runtime_stats (provider_id) VALUES ($1)`, id); err != nil {
		t.Fatal(err)
	}
	return id
}

func dashboardInsertSupplierAccount(t *testing.T, tx *sql.Tx, providerID int64, name, groupKey string) int64 {
	t.Helper()
	return dashboardInsertID(t, tx, `
INSERT INTO supplier_provider_accounts (provider_id, upstream_account_key, name, group_key, group_name, status, rate_sync_status)
VALUES ($1, $2, $2, $3, $3, 'active', 'success')
RETURNING id`, providerID, name, groupKey)
}

func dashboardInsertUsage(t *testing.T, tx *sql.Tx, userID, apiKeyID, accountID int64, requestID string, cost float64, createdAt time.Time) {
	t.Helper()
	if _, err := tx.ExecContext(context.Background(), `
INSERT INTO usage_logs (user_id, api_key_id, account_id, request_id, model, actual_cost, created_at)
VALUES ($1,$2,$3,$4,'dashboard-model',$5,$6)`, userID, apiKeyID, accountID, requestID, cost, createdAt); err != nil {
		t.Fatal(err)
	}
}

func dashboardInsertError(t *testing.T, tx *sql.Tx, accountID int64, status int, countTokens bool, createdAt time.Time) {
	t.Helper()
	if _, err := tx.ExecContext(context.Background(), `
INSERT INTO ops_error_logs (account_id, error_phase, error_type, status_code, is_count_tokens, created_at)
VALUES ($1,'upstream','dashboard',$2,$3,$4)`, accountID, status, countTokens, createdAt); err != nil {
		t.Fatal(err)
	}
}

func dashboardInsertSyncRun(t *testing.T, tx *sql.Tx, providerID int64, scope, status string, finishedAt time.Time) {
	t.Helper()
	startedAt := finishedAt.Add(-time.Minute)
	if _, err := tx.ExecContext(context.Background(), `
INSERT INTO supplier_provider_sync_runs (provider_id, sync_scope, trigger_source, status, started_at, finished_at)
VALUES ($1,$2,'manual',$3,$4,$5)`, providerID, scope, status, startedAt, finishedAt); err != nil {
		t.Fatal(err)
	}
}

func dashboardInsertRateMapping(t *testing.T, tx *sql.Tx, providerID int64, groupKey, suffix string) int64 {
	t.Helper()
	localGroupID := dashboardInsertID(t, tx, `INSERT INTO groups (name, rate_multiplier) VALUES ($1, 1) RETURNING id`, "dashboard-rate-"+fmt.Sprint(providerID)+"-"+groupKey+"-"+suffix)
	return dashboardInsertID(t, tx, `
INSERT INTO supplier_provider_groups (provider_id, upstream_group_key, name, rate_multiplier, local_group_id)
VALUES ($1,$2,$2,1,$3)
RETURNING id`, providerID, groupKey, localGroupID)
}

func dashboardInsertRateChange(t *testing.T, tx *sql.Tx, mappingID int64, oldRate, newRate float64, changedAt time.Time) {
	t.Helper()
	if _, err := tx.ExecContext(context.Background(), `
INSERT INTO supplier_rate_guard_change_logs (
  mapping_id, local_group_id, local_group_name, upstream_group_key, upstream_group_name,
  old_rate, new_rate, status, changed_at
)
SELECT mapping.id, local_group.id, local_group.name, mapping.upstream_group_key, mapping.name,
       $2, $3, 'pending', $4
FROM supplier_provider_groups mapping
JOIN groups local_group ON local_group.id = mapping.local_group_id
WHERE mapping.id = $1`, mappingID, oldRate, newRate, changedAt); err != nil {
		t.Fatal(err)
	}
}

func dashboardSetSupplierAccountRate(t *testing.T, tx *sql.Tx, accountID int64, rate float64) {
	t.Helper()
	if _, err := tx.ExecContext(context.Background(), `UPDATE supplier_provider_accounts SET rate_multiplier=$2 WHERE id=$1`, accountID, rate); err != nil {
		t.Fatal(err)
	}
}
func dashboardUpdateRuntimeCollectionFacts(t *testing.T, tx *sql.Tx, providerID int64, balance, days float64, groupStatus string, groupSyncedAt time.Time) {
	t.Helper()
	if _, err := tx.ExecContext(context.Background(), `
UPDATE supplier_provider_runtime_stats
SET current_balance=$2, estimated_days=$3, group_sync_status=$4, last_group_sync_at=$5, updated_at=$5
WHERE provider_id=$1`, providerID, balance, days, groupStatus, groupSyncedAt); err != nil {
		t.Fatal(err)
	}
}

func dashboardTimePointer(value time.Time) *time.Time {
	return &value
}

func dashboardEqualTimePointer(actual, expected *time.Time) bool {
	if actual == nil || expected == nil {
		return actual == expected
	}
	return actual.Equal(*expected)
}
func dashboardFloatPointer(value float64) *float64 {
	return &value
}

func dashboardEqualFloatPointer(actual, expected *float64) bool {
	if actual == nil || expected == nil {
		return actual == nil && expected == nil
	}
	return math.Abs(*actual-*expected) < 1e-9
}

func TestSupplierDashboardRepositoryIntegrationNormalizesCollectionStatusesConservatively(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	accountPartialID := dashboardInsertProvider(t, tx, "dash-account-partial-"+suffix, "ap", true)
	groupPartialID := dashboardInsertProvider(t, tx, "dash-group-partial-"+suffix, "gp", true)
	balancePartialID := dashboardInsertProvider(t, tx, "dash-balance-partial-"+suffix, "bp", true)
	balanceFailedID := dashboardInsertProvider(t, tx, "dash-balance-failed-"+suffix, "bf", true)
	allSuccessID := dashboardInsertProvider(t, tx, "dash-all-success-"+suffix, "as", true)

	setBalance := func(providerID int64, balance float64, days any) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, `UPDATE supplier_provider_runtime_stats SET current_balance=$2, estimated_days=$3 WHERE provider_id=$1`, providerID, balance, days); err != nil {
			t.Fatal(err)
		}
	}
	setBalance(accountPartialID, 10, 4.0)
	setBalance(groupPartialID, 20, 5.0)
	setBalance(balancePartialID, 30, 6.0)
	setBalance(balanceFailedID, 40, 7.0)
	setBalance(allSuccessID, 0, nil)

	for _, tc := range []struct {
		providerID int64
		scope      string
		status     string
		minutes    int
	}{
		{accountPartialID, "accounts", "partial", 30}, {accountPartialID, "groups", "success", 29}, {accountPartialID, "balance", "success", 28},
		{groupPartialID, "accounts", "success", 27}, {groupPartialID, "groups", "partial", 26}, {groupPartialID, "balance", "success", 25},
		{balancePartialID, "accounts", "success", 24}, {balancePartialID, "groups", "success", 23}, {balancePartialID, "balance", "partial", 22},
		{balanceFailedID, "accounts", "success", 21}, {balanceFailedID, "groups", "success", 20}, {balanceFailedID, "balance", "failed", 19},
		{allSuccessID, "all", "success", 18},
	} {
		dashboardInsertSyncRun(t, tx, tc.providerID, tc.scope, tc.status, end.Add(-time.Duration(tc.minutes)*time.Minute))
	}

	providers, err := repo.ListDashboardProviders(ctx, start, end)
	if err != nil {
		t.Fatal(err)
	}
	bySlug := map[string]serviceProviderFacts{}
	for _, item := range providers {
		bySlug[item.ProviderSlug] = serviceProviderFacts{complete: item.DataComplete, sync: item.SyncStatus, group: item.GroupSyncStatus, balanceSync: item.BalanceSyncStatus, balance: item.Balance, days: item.EstimatedDays}
	}
	assert := func(slug string, complete bool, account, group, balanceSync string, balance, days *float64) {
		t.Helper()
		got, ok := bySlug[slug]
		if !ok || got.complete != complete || got.sync != account || got.group != group || got.balanceSync != balanceSync || !dashboardEqualFloatPointer(got.balance, balance) || !dashboardEqualFloatPointer(got.days, days) {
			t.Fatalf("%s facts=%+v", slug, got)
		}
	}
	assert("dash-account-partial-"+suffix, false, "unknown", "success", "success", dashboardFloatPointer(10), dashboardFloatPointer(4))
	assert("dash-group-partial-"+suffix, false, "success", "unknown", "success", dashboardFloatPointer(20), dashboardFloatPointer(5))
	assert("dash-balance-partial-"+suffix, false, "success", "success", "unknown", nil, nil)
	assert("dash-balance-failed-"+suffix, false, "success", "success", "failed", nil, nil)
	assert("dash-all-success-"+suffix, true, "success", "success", "success", dashboardFloatPointer(0), nil)
}

func TestSupplierDashboardRepositoryIntegrationMaintainsPersistedSupplierAccountNormalization(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	firstProviderID := dashboardInsertProvider(t, tx, "dash-normalize-first-"+suffix, "Pre-", true)
	secondProviderID := dashboardInsertProvider(t, tx, "dash-normalize-second-"+suffix, "Alt_", true)
	accountID := dashboardInsertSupplierAccount(t, tx, firstProviderID, "A.1", "g-normalize")

	assertNormalized := func(want *string) {
		t.Helper()
		var got sql.NullString
		if err := tx.QueryRowContext(ctx, `
SELECT supplier_dashboard_normalized_effective_name
FROM supplier_provider_accounts
WHERE id = $1`, accountID).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if want == nil {
			if got.Valid {
				t.Fatalf("normalized effective name = %q, want NULL", got.String)
			}
			return
		}
		if !got.Valid || got.String != *want {
			t.Fatalf("normalized effective name = %+v, want %q", got, *want)
		}
	}
	stringPointer := func(value string) *string { return &value }

	assertNormalized(stringPointer("prea1"))
	if _, err := tx.ExecContext(ctx, `UPDATE supplier_provider_accounts SET name = 'B 2' WHERE id = $1`, accountID); err != nil {
		t.Fatal(err)
	}
	assertNormalized(stringPointer("preb2"))

	if _, err := tx.ExecContext(ctx, `UPDATE supplier_provider_accounts SET provider_id = $2 WHERE id = $1`, accountID, secondProviderID); err != nil {
		t.Fatal(err)
	}
	assertNormalized(stringPointer("altb2"))

	if _, err := tx.ExecContext(ctx, `UPDATE supplier_provider_accounts SET active = FALSE WHERE id = $1`, accountID); err != nil {
		t.Fatal(err)
	}
	assertNormalized(nil)
	if _, err := tx.ExecContext(ctx, `UPDATE supplier_provider_accounts SET active = TRUE WHERE id = $1`, accountID); err != nil {
		t.Fatal(err)
	}
	assertNormalized(stringPointer("altb2"))

	if _, err := tx.ExecContext(ctx, `UPDATE supplier_providers SET deleted_at = NOW() WHERE id = $1`, secondProviderID); err != nil {
		t.Fatal(err)
	}
	assertNormalized(nil)
	if _, err := tx.ExecContext(ctx, `UPDATE supplier_providers SET deleted_at = NULL WHERE id = $1`, secondProviderID); err != nil {
		t.Fatal(err)
	}
	assertNormalized(stringPointer("altb2"))
	if _, err := tx.ExecContext(ctx, `UPDATE supplier_providers SET account_name_prefix = 'New-' WHERE id = $1`, secondProviderID); err != nil {
		t.Fatal(err)
	}
	assertNormalized(stringPointer("newb2"))

	if _, err := tx.ExecContext(ctx, `UPDATE supplier_provider_accounts SET supplier_dashboard_normalized_effective_name = NULL WHERE id = $1`, accountID); err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join("..", "..", "migrations", "192_supplier_dashboard_query_schema.sql")
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, string(schemaSQL)); err != nil {
		t.Fatalf("reapply supplier dashboard schema migration: %v", err)
	}
	assertNormalized(stringPointer("newb2"))
}

func TestSupplierDashboardRepositoryIntegrationKeepsProviderRiskTimeStableAcrossUnrelatedRuntimeUpdates(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	now := time.Now().UTC().Truncate(time.Microsecond)
	start := now.Add(-24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	providerSlug := "dash-risk-time-" + suffix
	providerID := dashboardInsertProvider(t, tx, providerSlug, "dash", true)
	_ = dashboardInsertLocalAccount(t, tx, "dashrisk"+suffix)
	accountID := dashboardInsertSupplierAccount(t, tx, providerID, "risk"+suffix, "g-risk-time")

	var riskUpdatedAt time.Time
	if err := tx.QueryRowContext(ctx, `
UPDATE supplier_provider_runtime_stats
SET risk_level = 'critical'
WHERE provider_id = $1
RETURNING risk_updated_at`, providerID).Scan(&riskUpdatedAt); err != nil {
		t.Fatal(err)
	}
	riskUpdatedAt = riskUpdatedAt.UTC()
	laterRuntimeUpdate := riskUpdatedAt.Add(24 * time.Hour)
	if _, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_runtime_stats
SET current_balance = 99,
    period_cost = 123,
    sync_status = 'success',
    last_sync_at = $2,
    updated_at = $2
WHERE provider_id = $1`, providerID, laterRuntimeUpdate); err != nil {
		t.Fatal(err)
	}

	var stableRiskUpdatedAt time.Time
	if err := tx.QueryRowContext(ctx, `SELECT risk_updated_at FROM supplier_provider_runtime_stats WHERE provider_id = $1`, providerID).Scan(&stableRiskUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if !stableRiskUpdatedAt.Equal(riskUpdatedAt) {
		t.Fatalf("risk_updated_at changed after unrelated runtime update: got %s want %s", stableRiskUpdatedAt, riskUpdatedAt)
	}

	accounts, err := repo.ListDashboardAccounts(ctx, start, now.Add(time.Hour), providerSlug, "g-risk-time")
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].AccountID != accountID || accounts[0].ProviderRiskUpdatedAt == nil || !accounts[0].ProviderRiskUpdatedAt.Equal(riskUpdatedAt) {
		t.Fatalf("provider risk facts = %+v, want account %d at %s", accounts, accountID, riskUpdatedAt)
	}
}
func TestSupplierDashboardRepositoryIntegrationAccountBalanceEvidenceDrivesRiskAndDetectedAt(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	now := time.Now().UTC().Truncate(time.Microsecond)
	start := now.Add(-24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	providerSlug := "dash-account-balance-" + suffix
	groupKey := "g-balance"
	oldAccountTime := now.Add(-30 * 24 * time.Hour)
	balanceFinishedAt := now.Add(-time.Hour)

	_ = dashboardInsertLocalAccount(t, tx, "dashbalance"+suffix)
	providerID := dashboardInsertProvider(t, tx, providerSlug, "dash", true)
	supplierAccountID := dashboardInsertSupplierAccount(t, tx, providerID, "balance"+suffix, groupKey)
	if _, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_accounts
SET updated_at = $2
WHERE id = $1`, supplierAccountID, oldAccountTime); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_runtime_stats
SET current_balance = 10,
    estimated_days = 2
WHERE provider_id = $1`, providerID); err != nil {
		t.Fatal(err)
	}
	dashboardInsertSyncRun(t, tx, providerID, "accounts", "partial", now.Add(-3*time.Hour))
	dashboardInsertSyncRun(t, tx, providerID, "groups", "success", now.Add(-2*time.Hour))
	dashboardInsertSyncRun(t, tx, providerID, "balance", "success", balanceFinishedAt)

	accounts, err := repo.ListDashboardAccounts(ctx, start, now, providerSlug, groupKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].AccountID != supplierAccountID {
		t.Fatalf("filtered accounts = %+v", accounts)
	}
	snapshot := accounts[0]
	if snapshot.BalanceSyncStatus != "success" || snapshot.BalanceSyncedAt == nil || !snapshot.BalanceSyncedAt.Equal(balanceFinishedAt) || snapshot.EstimatedDays == nil || *snapshot.EstimatedDays != 2 {
		t.Fatalf("independent balance evidence = %+v", snapshot)
	}

	svc := service.NewSupplierDashboardService(repo, nil)
	result, err := svc.GetAccounts(ctx, service.SupplierDashboardAccountsQuery{
		Range:        service.SupplierDashboardRange24Hours,
		RiskType:     service.SupplierDashboardRiskTypeBalance,
		ProviderSlug: providerSlug,
		GroupKey:     groupKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].AccountID != supplierAccountID || !result.Items[0].DetectedAt.Equal(balanceFinishedAt) {
		t.Fatalf("balance risk result = %+v", result.Items)
	}
}

func TestSupplierDashboardRepositoryIntegrationDefaultPlannerUsesTargetedNormalizationAndRateHistoryIndexes(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	end := time.Now().UTC().Truncate(time.Microsecond)
	start := end.Add(-24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	providerSlug := "dash-default-plan-" + suffix
	groupKey := "g-target"
	_ = dashboardInsertLocalAccount(t, tx, "dashtarget"+suffix)
	providerID := dashboardInsertProvider(t, tx, providerSlug, "dash", true)
	accountID := dashboardInsertSupplierAccount(t, tx, providerID, "target"+suffix, groupKey)
	dashboardSetSupplierAccountRate(t, tx, accountID, 0.9)
	targetMappingID := dashboardInsertRateMapping(t, tx, providerID, groupKey, suffix)
	dashboardInsertRateChange(t, tx, targetMappingID, 0.8, 0.9, start.Add(time.Hour))

	noiseProviderID := dashboardInsertProvider(t, tx, "dash-normalized-noise-"+suffix, "noise", true)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_accounts (
  provider_id, upstream_account_key, name, group_key, group_name, status, rate_sync_status
)
SELECT $1, 'noise-key-' || series_id, 'noise-name-' || series_id, 'g-noise', 'g-noise', 'active', 'success'
FROM generate_series(1, 12000) AS series_id`, noiseProviderID); err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 4; index++ {
		decoyProviderID := dashboardInsertProvider(t, tx, fmt.Sprintf("dash-history-noise-%d-%s", index, suffix), "history", true)
		decoyMappingID := dashboardInsertRateMapping(t, tx, decoyProviderID, fmt.Sprintf("g-noise-%d", index), suffix)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_rate_guard_change_logs (
  mapping_id, local_group_id, local_group_name, upstream_group_key, upstream_group_name,
  old_rate, new_rate, status, changed_at
)
SELECT mapping.id, local_group.id, local_group.name, mapping.upstream_group_key, mapping.name,
       0.8, 0.9, 'pending', $2::timestamptz + (series_id * INTERVAL '1 millisecond')
FROM supplier_provider_groups mapping
JOIN groups local_group ON local_group.id = mapping.local_group_id
CROSS JOIN generate_series(1, 5000) AS series_id
WHERE mapping.id = $1`, decoyMappingID, start.Add(2*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}

	for _, table := range []string{"supplier_provider_accounts", "supplier_provider_groups", "supplier_rate_guard_change_logs", "accounts"} {
		if _, err := tx.ExecContext(ctx, "ANALYZE "+table); err != nil {
			t.Fatal(err)
		}
	}

	rates, err := repo.ListDashboardRates(ctx, start, end, providerSlug, groupKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 1 || rates[0].AccountID != accountID || rates[0].RateChangeCount != 1 || rates[0].RateChangeOld == nil || *rates[0].RateChangeOld != 0.8 || rates[0].RateChangeNew == nil || *rates[0].RateChangeNew != 0.9 {
		t.Fatalf("targeted rate result = %+v", rates)
	}

	rows, err := tx.QueryContext(ctx, "EXPLAIN (FORMAT TEXT) "+supplierDashboardRatesQuery, start, end, providerSlug, groupKey)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	planLines := make([]string, 0)
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		planLines = append(planLines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	plan := strings.Join(planLines, "\n")
	t.Logf("default planner supplier dashboard EXPLAIN:\n%s", plan)
	for _, indexName := range []string{
		"idx_supplier_dashboard_supplier_accounts_provider_group",
		"idx_supplier_dashboard_supplier_accounts_normalized_active",
		"idx_supplier_dashboard_rate_changes_mapping_time",
	} {
		if !strings.Contains(plan, indexName) {
			t.Fatalf("default planner does not use %s:\n%s", indexName, plan)
		}
	}
}
func TestSupplierDashboardRepositoryIntegrationFilteredQueryUsesDashboardIndexes(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	start := time.Now().UTC().Add(-24 * time.Hour)
	end := time.Now().UTC()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	providerSlug := "dash-explain-" + suffix
	groupKey := "g-explain"
	localID := dashboardInsertLocalAccount(t, tx, "dashexplain"+suffix)
	providerID := dashboardInsertProvider(t, tx, providerSlug, "dash", true)
	_ = dashboardInsertSupplierAccount(t, tx, providerID, "explain"+suffix, groupKey)
	dashboardInsertSyncRun(t, tx, providerID, "all", "success", end.Add(-time.Hour))
	runID := "dashboard-explain-health-" + suffix
	if _, err := tx.ExecContext(ctx, `INSERT INTO upstream_account_health_guard_runs (id, trigger, status, started_at, finished_at) VALUES ($1,'scheduled','success',$2,$3)`, runID, end.Add(-2*time.Hour), end.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO upstream_account_health_guard_run_items (run_id, account_id, account_name, provider_slug, provider_name, status, started_at, finished_at) VALUES ($1,$2,$3,$4,$4,'healthy',$5,$6)`, runID, localID, "dashexplain"+suffix, providerSlug, end.Add(-2*time.Hour), end.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	for _, indexName := range []string{
		"idx_supplier_dashboard_accounts_normalized_name",
		"idx_supplier_dashboard_supplier_accounts_provider_group",
		"idx_supplier_dashboard_supplier_accounts_normalized_active",
		"idx_supplier_dashboard_health_items_account_finished",
		"idx_supplier_dashboard_sync_runs_provider_scope_finished",
		"idx_supplier_dashboard_rate_changes_mapping_time",
	} {
		var indexDefinition string
		if err := tx.QueryRowContext(ctx, `SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = $1`, indexName).Scan(&indexDefinition); err != nil {
			t.Fatalf("dashboard index %s: %v", indexName, err)
		}
		if strings.TrimSpace(indexDefinition) == "" {
			t.Fatalf("dashboard index %s has empty definition", indexName)
		}
	}
	if _, err := tx.ExecContext(ctx, `SET LOCAL enable_seqscan = off`); err != nil {
		t.Fatal(err)
	}
	rows, err := tx.QueryContext(ctx, "EXPLAIN (FORMAT TEXT) "+supplierDashboardAccountsQuery, start, end, providerSlug, groupKey)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	planLines := make([]string, 0)
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		planLines = append(planLines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	plan := strings.Join(planLines, "\n")
	t.Logf("filtered supplier dashboard EXPLAIN:\n%s", plan)
	if len(planLines) == 0 {
		t.Fatal("filtered supplier dashboard EXPLAIN returned no plan")
	}
	for _, indexName := range []string{
		"idx_supplier_dashboard_accounts_normalized_name",
		"idx_supplier_dashboard_supplier_accounts_provider_group",
		"idx_supplier_dashboard_supplier_accounts_normalized_active",
		"idx_supplier_dashboard_health_items_account_finished",
		"idx_supplier_dashboard_sync_runs_provider_scope_finished",
		"idx_supplier_dashboard_rate_changes_mapping_time",
	} {
		if !strings.Contains(plan, indexName) {
			t.Fatalf("filtered plan does not use %s:\n%s", indexName, plan)
		}
	}
}

func TestSupplierDashboardRepositoryIntegrationReadsLatestHealthGuardTaskFact(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	repo := newSupplierDashboardRepository(tx)
	start := time.Date(2026, 7, 23, 8, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	localID := dashboardInsertLocalAccount(t, tx, "dashtask"+suffix)
	providerID := dashboardInsertProvider(t, tx, "dash-task-"+suffix, "dash", true)
	supplierID := dashboardInsertSupplierAccount(t, tx, providerID, "task"+suffix, "g-task")
	runID := "dashboard-health-" + suffix
	finishedAt := end.Add(-time.Hour)
	if _, err := tx.ExecContext(ctx, `INSERT INTO upstream_account_health_guard_runs (id, trigger, status, started_at, finished_at) VALUES ($1,'scheduled','success',$2,$3)`, runID, finishedAt.Add(-time.Minute), finishedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO upstream_account_health_guard_run_items (run_id, account_id, account_name, provider_slug, provider_name, status, reason, started_at, finished_at) VALUES ($1,$2,$3,$4,$4,'failed','test timeout',$5,$6)`, runID, localID, "dashtask"+suffix, "dash-task-"+suffix, finishedAt.Add(-time.Minute), finishedAt); err != nil {
		t.Fatal(err)
	}
	accounts, err := repo.ListDashboardAccounts(ctx, start, end, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range accounts {
		if item.AccountID == supplierID {
			if item.TaskStatus != "failed" || item.TaskReason != "test timeout" || item.TaskFinishedAt == nil || !item.TaskFinishedAt.Equal(finishedAt) {
				t.Fatalf("task facts=%+v", item)
			}
			return
		}
	}
	t.Fatalf("supplier account %d not found", supplierID)
}
