package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

type dashboardDetailStub struct {
	accounts            []SupplierDashboardAccountSnapshot
	rates               []SupplierDashboardRateSnapshot
	providers           []SupplierDashboardProviderSnapshot
	start               time.Time
	end                 time.Time
	accountProviderSlug string
	accountGroupKey     string
	rateProviderSlug    string
	rateGroupKey        string
	traffic             []SupplierDashboardTrafficSnapshot
	profit              []SupplierDashboardProfitSnapshot
	health              []SupplierDashboardHealthSnapshot
	trafficProviderSlug string
	trafficGroupKey     string
	profitProviderSlug  string
	profitGroupKey      string
	profitLimit         int
	healthProviderSlug  string
	healthGroupKey      string
}

func (s *dashboardDetailStub) ListDashboardAccounts(_ context.Context, start, end time.Time, providerSlug, groupKey string) ([]SupplierDashboardAccountSnapshot, error) {
	s.start, s.end = start, end
	s.accountProviderSlug, s.accountGroupKey = providerSlug, groupKey
	return s.accounts, nil
}

func (s *dashboardDetailStub) ListDashboardRates(_ context.Context, start, end time.Time, providerSlug, groupKey string) ([]SupplierDashboardRateSnapshot, error) {
	s.start, s.end = start, end
	s.rateProviderSlug, s.rateGroupKey = providerSlug, groupKey
	return s.rates, nil
}

func (s *dashboardDetailStub) ListDashboardProviders(_ context.Context, start, end time.Time) ([]SupplierDashboardProviderSnapshot, error) {
	s.start, s.end = start, end
	return s.providers, nil
}

func (s *dashboardDetailStub) ListDashboardAccountTraffic(_ context.Context, start, end time.Time, providerSlug, groupKey string) ([]SupplierDashboardTrafficSnapshot, error) {
	s.start, s.end = start, end
	s.trafficProviderSlug, s.trafficGroupKey = providerSlug, groupKey
	return s.traffic, nil
}

func (s *dashboardDetailStub) ListDashboardAccountProfit(_ context.Context, start, end time.Time, providerSlug, groupKey string, limit int) ([]SupplierDashboardProfitSnapshot, error) {
	s.start, s.end = start, end
	s.profitProviderSlug, s.profitGroupKey = providerSlug, groupKey
	s.profitLimit = limit
	return s.profit, nil
}

func (s *dashboardDetailStub) ListDashboardAccountHealth(_ context.Context, start, end time.Time, providerSlug, groupKey string, _ int) ([]SupplierDashboardHealthSnapshot, error) {
	s.start, s.end = start, end
	s.healthProviderSlug, s.healthGroupKey = providerSlug, groupKey
	return s.health, nil
}

type dashboardThresholdStub struct {
	threshold *float64
	err       error
}

func (s *dashboardThresholdStub) GetDashboardOverview(context.Context, *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	return &OpsDashboardOverview{}, nil
}

func (s *dashboardThresholdStub) GetMetricThresholds(context.Context) (*OpsMetricThresholds, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &OpsMetricThresholds{SLAPercentMin: s.threshold}, nil
}

func dashboardDetailPointer[T any](value T) *T { return &value }

func newDashboardDetailService(now time.Time, repo *dashboardDetailStub) *SupplierDashboardService {
	svc := NewSupplierDashboardService(repo, nil)
	svc.now = func() time.Time { return now }
	return svc
}

func TestSupplierDashboardSnapshotContractContainsOnlyNeutralFacts(t *testing.T) {
	for typ, forbidden := range map[reflect.Type][]string{
		reflect.TypeOf(SupplierDashboardAccountSnapshot{}): {
			"Severity", "RiskTypes", "TrafficImpact", "RequestCount", "SuccessRate", "LowestRate", "RateDeltaPercent", "EstimatedExtraCost", "BalanceRisk", "SyncRisk", "CriticalIssueCount",
		},
		reflect.TypeOf(SupplierDashboardRateSnapshot{}): {
			"Severity", "RiskTypes", "TrafficImpact", "RequestCount", "SuccessRate", "LowestRate", "RateDeltaPercent", "EstimatedExtraCost", "BalanceRisk", "SyncRisk", "CriticalIssueCount",
		},
		reflect.TypeOf(SupplierDashboardProviderSnapshot{}): {
			"Severity", "RiskTypes", "TrafficImpact", "RequestCount", "SuccessRate", "LowestRate", "RateDeltaPercent", "EstimatedExtraCost", "BalanceRisk", "SyncRisk", "CriticalIssueCount", "Status",
		},
	} {
		for _, name := range forbidden {
			if _, ok := typ.FieldByName(name); ok {
				t.Fatalf("%s must not contain derived field %s", typ.Name(), name)
			}
		}
	}
	providerType := reflect.TypeOf(SupplierDashboardProviderSnapshot{})
	for _, required := range []string{"SyncStatus", "GroupSyncStatus", "BalanceSyncStatus", "SuccessCount", "ErrorCount", "LastSyncedAt"} {
		if _, ok := providerType.FieldByName(required); !ok {
			t.Fatalf("provider snapshot missing neutral fact %s", required)
		}
	}
	accountType := reflect.TypeOf(SupplierDashboardAccountSnapshot{})
	for _, required := range []string{"ProviderRiskUpdatedAt", "BalanceSyncStatus", "BalanceSyncedAt"} {
		if _, ok := accountType.FieldByName(required); !ok {
			t.Fatalf("account snapshot missing evidence fact %s", required)
		}
	}
}

func TestSupplierDashboardAccountRiskFiltersHaveHitAndMissPaths(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	one, two := 1.0, 2.0
	zero, oneCount := int64(0), int64(1)
	repo := &dashboardDetailStub{accounts: []SupplierDashboardAccountSnapshot{
		{AccountID: 1, AccountName: "clean", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, AccountStatus: "active", SuccessCount: &zero, ErrorCount: &zero},
		{AccountID: 2, AccountName: "critical", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, ProviderRiskLevel: "critical"},
		{AccountID: 3, AccountName: "traffic", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, TaskStatus: "failed", SuccessCount: &oneCount, ErrorCount: &oneCount},
		{AccountID: 4, AccountName: "rate-up", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, CurrentRate: &two, SnapshotCount: 1, RateChangeOld: &one, RateChangeNew: &two, RateChangeCount: 1},
		{AccountID: 5, AccountName: "lowest", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &one},
		{AccountID: 6, AccountName: "not-lowest", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &two},
		{AccountID: 7, AccountName: "balance", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, BalanceSyncStatus: "success", EstimatedDays: dashboardDetailPointer(2.0)},
		{AccountID: 8, AccountName: "sync", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, RateSyncStatus: "failed"},
		{AccountID: 9, AccountName: "task", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, TaskStatus: "failed", TaskReason: "test timeout"},
		{AccountID: 10, AccountName: "disabled", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: false, ProviderRiskLevel: "critical"},
	}}
	svc := newDashboardDetailService(now, repo)

	cases := []struct {
		risk SupplierDashboardRiskType
		hit  int64
	}{
		{SupplierDashboardRiskTypeAll, 1},
		{SupplierDashboardRiskTypeCritical, 2},
		{SupplierDashboardRiskTypeTraffic, 3},
		{SupplierDashboardRiskTypeRateUp, 4},
		{SupplierDashboardRiskTypeNotLowest, 6},
		{SupplierDashboardRiskTypeBalance, 7},
		{SupplierDashboardRiskTypeSync, 8},
		{SupplierDashboardRiskTypeTask, 9},
	}
	for _, tc := range cases {
		t.Run(string(tc.risk), func(t *testing.T) {
			result, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: SupplierDashboardRange24Hours, RiskType: tc.risk})
			if err != nil {
				t.Fatal(err)
			}
			ids := map[int64]bool{}
			for _, item := range result.Items {
				ids[item.AccountID] = true
			}
			if !ids[tc.hit] {
				t.Fatalf("risk=%q missing hit %d: %+v", tc.risk, tc.hit, result.Items)
			}
			if tc.risk != SupplierDashboardRiskTypeAll && ids[1] {
				t.Fatalf("risk=%q included clean account: %+v", tc.risk, result.Items)
			}
			if ids[10] {
				t.Fatalf("risk=%q included disabled account: %+v", tc.risk, result.Items)
			}
		})
	}
	if !repo.start.Equal(now.Add(-24*time.Hour)) || !repo.end.Equal(now) {
		t.Fatalf("window=%s..%s", repo.start, repo.end)
	}
}

func TestSupplierDashboardResolveRange30Days(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	repo := &dashboardDetailStub{}
	svc := newDashboardDetailService(now, repo)

	if _, err := svc.GetAccountTraffic(context.Background(), SupplierDashboardTrafficQuery{Range: SupplierDashboardRange30Days}); err != nil {
		t.Fatal(err)
	}
	if !repo.start.Equal(now.Add(-30*24*time.Hour)) || !repo.end.Equal(now) {
		t.Fatalf("30d window=%s..%s", repo.start, repo.end)
	}

	repo = &dashboardDetailStub{}
	svc = newDashboardDetailService(now, repo)
	if _, err := svc.GetAccountProfitRanking(context.Background(), SupplierDashboardProfitQuery{Range: SupplierDashboardRange30Days}); err != nil {
		t.Fatal(err)
	}
	if !repo.start.Equal(now.Add(-30*24*time.Hour)) || !repo.end.Equal(now) {
		t.Fatalf("30d profit window=%s..%s", repo.start, repo.end)
	}

	repo = &dashboardDetailStub{}
	svc = newDashboardDetailService(now, repo)
	if _, err := svc.GetAccountHealthTimeline(context.Background(), SupplierDashboardAccountHealthQuery{Range: SupplierDashboardRange30Days}); err != nil {
		t.Fatal(err)
	}
	if !repo.start.Equal(now.Add(-30*24*time.Hour)) || !repo.end.Equal(now) {
		t.Fatalf("30d health window=%s..%s", repo.start, repo.end)
	}
}

func TestSupplierDashboardTaskRiskRequiresFailedTaskFact(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	failedAt := now.Add(-time.Hour)
	repo := &dashboardDetailStub{accounts: []SupplierDashboardAccountSnapshot{
		{AccountID: 1, ProviderSlug: "provider", ProviderEnabled: true, AccountEnabled: true, ProviderRiskLevel: "high", AccountStatus: "failed"},
		{AccountID: 2, ProviderSlug: "provider", ProviderEnabled: true, AccountEnabled: true, TaskStatus: "healthy", TaskFinishedAt: &failedAt},
		{AccountID: 3, ProviderSlug: "provider", ProviderEnabled: true, AccountEnabled: true, TaskStatus: "failed", TaskReason: "test timeout", TaskFinishedAt: &failedAt},
	}}
	svc := newDashboardDetailService(now, repo)

	all, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: SupplierDashboardRange24Hours, RiskType: SupplierDashboardRiskTypeAll})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]SupplierDashboardAccountItem{}
	for _, item := range all.Items {
		byID[item.AccountID] = item
	}
	if dashboardHasRisk(byID[1].RiskTypes, SupplierDashboardRiskTypeTask) || dashboardHasRisk(byID[2].RiskTypes, SupplierDashboardRiskTypeTask) {
		t.Fatalf("ordinary status or healthy task created task risk: %+v", byID)
	}
	if !dashboardHasRisk(byID[3].RiskTypes, SupplierDashboardRiskTypeTask) || byID[3].Reason != "test timeout" {
		t.Fatalf("failed task fact not mapped: %+v", byID[3])
	}
}

func TestSupplierDashboardRatesAggregateByProviderAndGroup(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	one, two, three := 1.0, 2.0, 3.0
	oneRequest, tenRequests := int64(1), int64(10)
	zero := int64(0)
	lastOld, lastNew := now.Add(-2*time.Hour), now.Add(-time.Hour)
	repo := &dashboardDetailStub{rates: []SupplierDashboardRateSnapshot{
		{AccountID: 1, AccountName: "low", ProviderSlug: "p1", ProviderName: "Provider One", ProviderEnabled: true, AccountEnabled: true, GroupKey: "shared", GroupName: "Shared", CurrentRate: &one, SnapshotCount: 2, RateChangeOld: &two, RateChangeNew: &three, RateChangeCount: 1, SuccessCount: &oneRequest, ErrorCount: &zero, PeriodCost: dashboardDetailPointer(1.0), LastRateSyncedAt: &lastOld},
		{AccountID: 2, AccountName: "current", ProviderSlug: "p1", ProviderName: "Provider One", ProviderEnabled: true, AccountEnabled: true, GroupKey: "shared", GroupName: "Shared", CurrentRate: &three, SnapshotCount: 2, RateChangeOld: &two, RateChangeNew: &three, RateChangeCount: 1, SuccessCount: &tenRequests, ErrorCount: &zero, PeriodCost: dashboardDetailPointer(30.0), LastRateSyncedAt: &lastNew},
		{AccountID: 3, AccountName: "disabled-cheapest", ProviderSlug: "p1", ProviderEnabled: true, AccountEnabled: false, GroupKey: "shared", GroupName: "Shared", CurrentRate: dashboardDetailPointer(0.1)},
		{AccountID: 4, AccountName: "tie-a", ProviderSlug: "p2", ProviderName: "Provider Two", ProviderEnabled: true, AccountEnabled: true, GroupKey: "shared", GroupName: "Shared", CurrentRate: &two, SnapshotCount: 1},
		{AccountID: 5, AccountName: "tie-b", ProviderSlug: "p2", ProviderName: "Provider Two", ProviderEnabled: true, AccountEnabled: true, GroupKey: "shared", GroupName: "Shared", CurrentRate: &two, SnapshotCount: 1},
		{AccountID: 6, AccountName: "missing", ProviderSlug: "p1", ProviderName: "Provider One", ProviderEnabled: true, AccountEnabled: true, GroupKey: "", CurrentRate: &one},
		{AccountID: 7, AccountName: "solo", ProviderSlug: "p1", ProviderName: "Provider One", ProviderEnabled: true, AccountEnabled: true, GroupKey: "solo", GroupName: "Solo", CurrentRate: &one, SnapshotCount: 1},
		{AccountID: 8, AccountName: "unknown", ProviderSlug: "p3", ProviderName: "Provider Three", ProviderEnabled: true, AccountEnabled: true, GroupKey: "unknown", GroupName: "Unknown"},
	}}
	svc := newDashboardDetailService(now, repo)

	result, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, View: SupplierDashboardRateViewAll})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 5 || len(result.Items) != 5 {
		t.Fatalf("group rows=%+v", result.Items)
	}
	byKey := map[string]SupplierDashboardRateItem{}
	for _, item := range result.Items {
		byKey[item.ProviderSlug+"/"+item.GroupKey] = item
	}
	main := byKey["p1/shared"]
	if main.EnabledAccountCount != 2 || main.CurrentAccountID != 2 || main.CurrentAccountName != "current" {
		t.Fatalf("current group selection=%+v", main)
	}
	if main.CurrentRate == nil || *main.CurrentRate != 3 || main.LowestRate == nil || *main.LowestRate != 1 || main.ComparisonStatus != SupplierDashboardComparisonStatusNotLowest {
		t.Fatalf("main comparison=%+v", main)
	}
	if main.RateDeltaPercent == nil || math.Abs(*main.RateDeltaPercent-50) > 1e-9 || main.EstimatedExtraCost == nil || math.Abs(*main.EstimatedExtraCost-20) > 1e-9 {
		t.Fatalf("main rate/cost=%+v", main)
	}
	if main.CostCurrency != nil || main.LastSyncedAt == nil || !main.LastSyncedAt.Equal(lastNew) {
		t.Fatalf("main currency/sync=%+v", main)
	}
	if got, want := main.LowestAccountIDs, []int64{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lowest ids=%v want=%v", got, want)
	}

	tied := byKey["p2/shared"]
	if tied.ComparisonStatus != SupplierDashboardComparisonStatusTiedLowest || !reflect.DeepEqual(tied.LowestAccountIDs, []int64{4, 5}) || !reflect.DeepEqual(tied.LowestAccountNames, []string{"tie-a", "tie-b"}) {
		t.Fatalf("tied lowest=%+v", tied)
	}
	if tied.EstimatedExtraCost == nil || *tied.EstimatedExtraCost != 0 {
		t.Fatalf("tied extra cost must be real zero: %+v", tied)
	}
	if byKey["p1/"].ComparisonStatus != SupplierDashboardComparisonStatusMissingGroup {
		t.Fatalf("missing group=%+v", byKey["p1/"])
	}
	if byKey["p1/solo"].ComparisonStatus != SupplierDashboardComparisonStatusInsufficientAccounts || byKey["p1/solo"].RateDeltaPercent != nil {
		t.Fatalf("first sync solo=%+v", byKey["p1/solo"])
	}
	if byKey["p3/unknown"].ComparisonStatus != SupplierDashboardComparisonStatusUnknown || byKey["p3/unknown"].EstimatedExtraCost != nil {
		t.Fatalf("unknown comparison=%+v", byKey["p3/unknown"])
	}
	for status, want := range map[SupplierDashboardComparisonStatus]string{
		SupplierDashboardComparisonStatusNotLowest:            "p1/shared",
		SupplierDashboardComparisonStatusTiedLowest:           "p2/shared",
		SupplierDashboardComparisonStatusMissingGroup:         "p1/",
		SupplierDashboardComparisonStatusInsufficientAccounts: "p1/solo",
		SupplierDashboardComparisonStatusUnknown:              "p3/unknown",
	} {
		filtered, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, ComparisonStatus: status})
		if err != nil || len(filtered.Items) != 1 || filtered.Items[0].ProviderSlug+"/"+filtered.Items[0].GroupKey != want {
			t.Fatalf("status=%q items=%+v err=%v", status, filtered.Items, err)
		}
	}
}

func TestSupplierDashboardRateViewsAndComparisonFiltersHaveProductionPaths(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	one, two, three := 1.0, 2.0, 3.0
	ten, zero := int64(10), int64(0)
	repo := &dashboardDetailStub{rates: []SupplierDashboardRateSnapshot{
		{AccountID: 1, ProviderSlug: "changed", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &one, SnapshotCount: 1},
		{AccountID: 2, ProviderSlug: "changed", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &three, SnapshotCount: 2, RateChangeOld: &two, RateChangeNew: &three, RateChangeCount: 1, SuccessCount: &ten, ErrorCount: &zero},
		{AccountID: 3, ProviderSlug: "stable", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &one, SnapshotCount: 1, SuccessCount: &ten, ErrorCount: &zero},
		{AccountID: 4, ProviderSlug: "stable", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &two, SnapshotCount: 1},
		{AccountID: 5, ProviderSlug: "first", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &three, SnapshotCount: 1, SuccessCount: &ten, ErrorCount: &zero},
		{AccountID: 6, ProviderSlug: "first", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &one, SnapshotCount: 1},
	}}
	svc := newDashboardDetailService(now, repo)

	cases := []struct {
		name string
		q    SupplierDashboardRatesQuery
		want []string
	}{
		{"all", SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, View: SupplierDashboardRateViewAll}, []string{"changed", "first", "stable"}},
		{"changed", SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, View: SupplierDashboardRateViewChanged}, []string{"changed"}},
		{"risk", SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, View: SupplierDashboardRateViewRisk}, []string{"changed", "first"}},
		{"not-lowest", SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, ComparisonStatus: SupplierDashboardComparisonStatusNotLowest}, []string{"changed", "first"}},
		{"lowest", SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, ComparisonStatus: SupplierDashboardComparisonStatusLowest}, []string{"stable"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.GetRates(context.Background(), tc.q)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				got = append(got, item.ProviderSlug)
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got=%v want=%v items=%+v", got, tc.want, result.Items)
			}
		})
	}
	first := mustDashboardRateItem(t, svc, "first")
	if first.RateDeltaPercent != nil {
		t.Fatalf("first synchronization must not invent rate change: %+v", first)
	}
}

func TestSupplierDashboardUsesWindowedGroupChangeFactsForChangedAndRateUp(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	oldUp, newUp := 0.8, 0.9
	oldDown, newDown := 0.9, 0.8
	one := 1.0
	changedAt := now.Add(-time.Hour)
	repo := &dashboardDetailStub{
		accounts: []SupplierDashboardAccountSnapshot{
			{AccountID: 1, AccountName: "up", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, GroupKey: "up", CurrentRate: &newUp, SnapshotCount: 1, RateChangeOld: &oldUp, RateChangeNew: &newUp, RateChangeCount: 1, RateChangedAt: &changedAt},
			{AccountID: 2, AccountName: "down", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, GroupKey: "down", CurrentRate: &newDown, SnapshotCount: 1, RateChangeOld: &oldDown, RateChangeNew: &newDown, RateChangeCount: 1, RateChangedAt: &changedAt},
			{AccountID: 3, AccountName: "none", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true, GroupKey: "none", CurrentRate: &one, SnapshotCount: 1},
		},
		rates: []SupplierDashboardRateSnapshot{
			{AccountID: 1, AccountName: "up", ProviderSlug: "up", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &newUp, SnapshotCount: 1, RateChangeOld: &oldUp, RateChangeNew: &newUp, RateChangeCount: 1, RateChangedAt: &changedAt},
			{AccountID: 2, AccountName: "down", ProviderSlug: "down", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &newDown, SnapshotCount: 1, RateChangeOld: &oldDown, RateChangeNew: &newDown, RateChangeCount: 1, RateChangedAt: &changedAt},
			{AccountID: 3, AccountName: "none", ProviderSlug: "none", ProviderEnabled: true, AccountEnabled: true, GroupKey: "g", CurrentRate: &one, SnapshotCount: 1},
		},
	}
	svc := newDashboardDetailService(now, repo)

	accounts, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: SupplierDashboardRange24Hours, RiskType: SupplierDashboardRiskTypeRateUp})
	if err != nil || len(accounts.Items) != 1 || accounts.Items[0].AccountName != "up" {
		t.Fatalf("rate_up accounts=%+v err=%v", accounts.Items, err)
	}
	if accounts.Items[0].RateDeltaPercent == nil || math.Abs(*accounts.Items[0].RateDeltaPercent-12.5) > 1e-9 {
		t.Fatalf("rate_up delta=%+v", accounts.Items[0])
	}

	changed, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, View: SupplierDashboardRateViewChanged})
	if err != nil {
		t.Fatal(err)
	}
	gotChanged := make([]string, 0, len(changed.Items))
	for _, item := range changed.Items {
		gotChanged = append(gotChanged, item.ProviderSlug)
	}
	sort.Strings(gotChanged)
	if !reflect.DeepEqual(gotChanged, []string{"down", "up"}) {
		t.Fatalf("changed providers=%v items=%+v", gotChanged, changed.Items)
	}

	risk, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, View: SupplierDashboardRateViewRisk})
	if err != nil || len(risk.Items) != 1 || risk.Items[0].ProviderSlug != "up" {
		t.Fatalf("risk rates=%+v err=%v", risk.Items, err)
	}
}
func mustDashboardRateItem(t *testing.T, svc *SupplierDashboardService, provider string) SupplierDashboardRateItem {
	t.Helper()
	result, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, ProviderSlug: provider})
	if err != nil || len(result.Items) != 1 {
		t.Fatalf("provider=%q result=%+v err=%v", provider, result, err)
	}
	return result.Items[0]
}

func TestSupplierDashboardProviderStatusUsesConfiguredSLAToDeriveFacts(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	threshold := 99.5
	zero, one, ninetyNine, oneNinetyNine, nineNinetyNine := int64(0), int64(1), int64(99), int64(199), int64(999)
	nan := math.NaN()
	days := 2.0
	repo := &dashboardDetailStub{providers: []SupplierDashboardProviderSnapshot{
		{ProviderSlug: "zero-percent", Enabled: true, DataComplete: true, EnabledAccountCount: 1, SuccessCount: &zero, ErrorCount: &one},
		{ProviderSlug: "below", Enabled: true, DataComplete: true, EnabledAccountCount: 1, SuccessCount: &ninetyNine, ErrorCount: &one},
		{ProviderSlug: "threshold", Enabled: true, DataComplete: true, EnabledAccountCount: 1, SuccessCount: &oneNinetyNine, ErrorCount: &one},
		{ProviderSlug: "above", Enabled: true, DataComplete: true, EnabledAccountCount: 1, SuccessCount: &nineNinetyNine, ErrorCount: &one},
		{ProviderSlug: "nil-traffic", Enabled: true, DataComplete: true, EnabledAccountCount: 1},
		{ProviderSlug: "no-requests", Enabled: true, DataComplete: true, EnabledAccountCount: 1, SuccessCount: &zero, ErrorCount: &zero},
		{ProviderSlug: "non-finite", Enabled: true, DataComplete: true, SuccessCount: &zero, ErrorCount: &zero, PeriodCost: &nan},
		{ProviderSlug: "incomplete", Enabled: true, DataComplete: false, EnabledAccountCount: 1, SuccessCount: &zero, ErrorCount: &one},
		{ProviderSlug: "failed-sync", Enabled: true, DataComplete: false, SyncStatus: "failed"},
		{ProviderSlug: "partial", Enabled: true, DataComplete: false, SyncStatus: "unknown", GroupSyncStatus: "unknown", BalanceSyncStatus: "unknown"},
		{ProviderSlug: "balance", Enabled: true, DataComplete: true, BalanceSyncStatus: "success", SuccessCount: &zero, ErrorCount: &zero, EstimatedDays: &days},
		{ProviderSlug: "critical", Enabled: true, DataComplete: true, ProviderRiskLevel: "critical", SuccessCount: &zero, ErrorCount: &zero},
		{ProviderSlug: "disabled", Enabled: false, DataComplete: false},
	}}
	svc := NewSupplierDashboardService(repo, &dashboardThresholdStub{threshold: &threshold})
	svc.now = func() time.Time { return now }

	result, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: SupplierDashboardRange24Hours})
	if err != nil {
		t.Fatal(err)
	}
	items := map[string]SupplierDashboardProviderItem{}
	for _, item := range result.Items {
		items[item.ProviderSlug] = item
	}
	assertProviderStatusAndCount(t, items["zero-percent"], SupplierDashboardProviderStatusHighRisk, dashboardDetailPointer(1))
	assertProviderStatusAndCount(t, items["below"], SupplierDashboardProviderStatusHighRisk, dashboardDetailPointer(1))
	assertProviderStatusAndCount(t, items["threshold"], SupplierDashboardProviderStatusHealthy, dashboardDetailPointer(0))
	assertProviderStatusAndCount(t, items["above"], SupplierDashboardProviderStatusHealthy, dashboardDetailPointer(0))
	assertProviderStatusAndCount(t, items["nil-traffic"], SupplierDashboardProviderStatusUnknown, nil)
	assertProviderStatusAndCount(t, items["no-requests"], SupplierDashboardProviderStatusHealthy, dashboardDetailPointer(0))
	assertProviderStatusAndCount(t, items["non-finite"], SupplierDashboardProviderStatusUnknown, dashboardDetailPointer(0))
	assertProviderStatusAndCount(t, items["incomplete"], SupplierDashboardProviderStatusHighRisk, nil)
	assertProviderStatusAndCount(t, items["failed-sync"], SupplierDashboardProviderStatusWarning, nil)
	assertProviderStatusAndCount(t, items["partial"], SupplierDashboardProviderStatusUnknown, nil)
	assertProviderStatusAndCount(t, items["balance"], SupplierDashboardProviderStatusWarning, dashboardDetailPointer(0))
	assertProviderStatusAndCount(t, items["critical"], SupplierDashboardProviderStatusHighRisk, dashboardDetailPointer(1))
	assertProviderStatusAndCount(t, items["disabled"], SupplierDashboardProviderStatusDisabled, nil)

	if items["zero-percent"].SuccessRate == nil || *items["zero-percent"].SuccessRate != 0 {
		t.Fatalf("zero success rate=%+v", items["zero-percent"])
	}
	if items["threshold"].SuccessRate == nil || math.Abs(*items["threshold"].SuccessRate-threshold) > 1e-9 {
		t.Fatalf("threshold success rate=%+v", items["threshold"])
	}
	if items["no-requests"].RequestCount == nil || *items["no-requests"].RequestCount != 0 || items["no-requests"].SuccessRate != nil {
		t.Fatalf("zero requests=%+v", items["no-requests"])
	}
	if items["non-finite"].PeriodCost != nil {
		t.Fatalf("non-finite value must not reach JSON DTO: %+v", items["non-finite"])
	}
	if !items["failed-sync"].SyncRisk || items["partial"].SyncRisk || !items["balance"].BalanceRisk {
		t.Fatalf("provider risks failed=%+v partial=%+v balance=%+v", items["failed-sync"], items["partial"], items["balance"])
	}
}

func TestSupplierDashboardProviderStatusUsesHighestKnownSeverity(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	zero, one := int64(0), int64(1)
	repo := &dashboardDetailStub{providers: []SupplierDashboardProviderSnapshot{
		{ProviderSlug: "low-sla-account-failed", Enabled: true, DataComplete: false, EnabledAccountCount: 1, SyncStatus: "failed", SuccessCount: &zero, ErrorCount: &one},
		{ProviderSlug: "low-sla-balance-failed", Enabled: true, DataComplete: false, EnabledAccountCount: 1, BalanceSyncStatus: "failed", SuccessCount: &zero, ErrorCount: &one},
		{ProviderSlug: "critical-group-failed", Enabled: true, DataComplete: false, ProviderRiskLevel: "critical", GroupSyncStatus: "failed"},
		{ProviderSlug: "critical-partial", Enabled: true, DataComplete: false, ProviderRiskLevel: "critical", SyncStatus: "unknown", GroupSyncStatus: "unknown", BalanceSyncStatus: "unknown"},
		{ProviderSlug: "incomplete-only", Enabled: true, DataComplete: false, SyncStatus: "unknown", GroupSyncStatus: "unknown", BalanceSyncStatus: "unknown"},
	}}
	svc := newDashboardDetailService(now, repo)

	result, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: SupplierDashboardRange24Hours})
	if err != nil {
		t.Fatal(err)
	}
	items := make(map[string]SupplierDashboardProviderItem, len(result.Items))
	for _, item := range result.Items {
		items[item.ProviderSlug] = item
	}
	for _, slug := range []string{"low-sla-account-failed", "low-sla-balance-failed", "critical-group-failed", "critical-partial"} {
		if item := items[slug]; item.Status != SupplierDashboardProviderStatusHighRisk {
			t.Fatalf("provider %q status=%q want=%q item=%+v", slug, item.Status, SupplierDashboardProviderStatusHighRisk, item)
		}
	}
	if item := items["incomplete-only"]; item.Status != SupplierDashboardProviderStatusUnknown {
		t.Fatalf("provider incomplete-only status=%q want=%q item=%+v", item.Status, SupplierDashboardProviderStatusUnknown, item)
	}
}
func TestProviderStatusIgnoresInvalidSuccessRates(t *testing.T) {
	zero := int64(0)
	snap := SupplierDashboardProviderSnapshot{Enabled: true, DataComplete: true, SuccessCount: &zero, ErrorCount: &zero}
	count := 0
	for name, rate := range map[string]float64{"nan": math.NaN(), "positive_inf": math.Inf(1), "negative_inf": math.Inf(-1)} {
		t.Run(name, func(t *testing.T) {
			status := providerStatusFromFacts(snap, dashboardDetailPointer(int64(1)), &rate, &count, false, false, 99.5)
			if status == SupplierDashboardProviderStatusHighRisk {
				t.Fatalf("invalid success rate must not create high risk: %v", status)
			}
		})
	}
}

func TestSupplierDashboardProviderBalanceSyncFactsAreConservative(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	repo := &dashboardDetailStub{providers: []SupplierDashboardProviderSnapshot{
		{ProviderSlug: "failed", Enabled: true, DataComplete: false, SyncStatus: "success", GroupSyncStatus: "success", BalanceSyncStatus: "failed"},
		{ProviderSlug: "partial", Enabled: true, DataComplete: false, SyncStatus: "success", GroupSyncStatus: "success", BalanceSyncStatus: "unknown"},
	}}
	svc := newDashboardDetailService(now, repo)
	result, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: SupplierDashboardRange24Hours})
	if err != nil {
		t.Fatal(err)
	}
	bySlug := map[string]SupplierDashboardProviderItem{}
	for _, item := range result.Items {
		bySlug[item.ProviderSlug] = item
	}
	if !bySlug["failed"].SyncRisk || bySlug["failed"].Status != SupplierDashboardProviderStatusWarning {
		t.Fatalf("balance failed=%+v", bySlug["failed"])
	}
	if bySlug["partial"].SyncRisk || bySlug["partial"].Status != SupplierDashboardProviderStatusUnknown {
		t.Fatalf("balance partial=%+v", bySlug["partial"])
	}
}

func TestSupplierDashboardProviderBalanceRiskUsesIndependentBalanceEvidence(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	days := 2.0
	repo := &dashboardDetailStub{providers: []SupplierDashboardProviderSnapshot{
		{ProviderSlug: "account-partial", Enabled: true, DataComplete: false, SyncStatus: "unknown", GroupSyncStatus: "success", BalanceSyncStatus: "success", EstimatedDays: &days},
		{ProviderSlug: "group-partial", Enabled: true, DataComplete: false, SyncStatus: "success", GroupSyncStatus: "unknown", BalanceSyncStatus: "success", EstimatedDays: &days},
		{ProviderSlug: "balance-partial", Enabled: true, DataComplete: false, SyncStatus: "success", GroupSyncStatus: "success", BalanceSyncStatus: "unknown", EstimatedDays: &days},
		{ProviderSlug: "balance-failed", Enabled: true, DataComplete: false, SyncStatus: "success", GroupSyncStatus: "success", BalanceSyncStatus: "failed", EstimatedDays: &days},
	}}
	svc := newDashboardDetailService(now, repo)

	result, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: SupplierDashboardRange24Hours})
	if err != nil {
		t.Fatal(err)
	}
	items := make(map[string]SupplierDashboardProviderItem, len(result.Items))
	for _, item := range result.Items {
		items[item.ProviderSlug] = item
	}
	for _, slug := range []string{"account-partial", "group-partial"} {
		item := items[slug]
		if !item.BalanceRisk || item.Status != SupplierDashboardProviderStatusWarning {
			t.Fatalf("provider %q must derive warning balance risk from successful balance evidence: %+v", slug, item)
		}
	}
	if item := items["balance-partial"]; item.BalanceRisk || item.Status != SupplierDashboardProviderStatusUnknown {
		t.Fatalf("partial balance evidence must not derive balance risk: %+v", item)
	}
	if item := items["balance-failed"]; item.BalanceRisk || !item.SyncRisk || item.Status != SupplierDashboardProviderStatusWarning {
		t.Fatalf("failed balance evidence must only derive sync warning: %+v", item)
	}
}
func assertProviderStatusAndCount(t *testing.T, item SupplierDashboardProviderItem, status SupplierDashboardProviderStatus, count *int) {
	t.Helper()
	if item.Status != status || !reflect.DeepEqual(item.CriticalIssueCount, count) {
		t.Fatalf("provider=%q status/count=%+v want status=%q count=%v", item.ProviderSlug, item, status, count)
	}
}

func TestSupplierDashboardDetailJSONContract(t *testing.T) {
	zeroFloat, zeroInt, zeroInt64 := 0.0, 0, int64(0)
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	account := SupplierDashboardAccountsResponse{
		Range: SupplierDashboardRange24Hours,
		Items: []SupplierDashboardAccountItem{{
			AccountID: 1, AccountName: "account", ProviderSlug: "provider", ProviderName: "Provider", GroupKey: "group", GroupName: "Group",
			Severity: SupplierDashboardSeverityLow, RiskTypes: []SupplierDashboardRiskType{}, RequestCount: &zeroInt64, SuccessRate: nil,
			Balance: &zeroFloat, BalanceCurrency: nil, Status: "unknown", Reason: "", DetectedAt: now, TargetPath: "/accounts",
		}},
		Total: 1, Page: 1, PageSize: 20, Warnings: []SupplierDashboardWarning{}, GeneratedAt: now,
	}
	rate := SupplierDashboardRatesResponse{
		Range: SupplierDashboardRange24Hours,
		Items: []SupplierDashboardRateItem{{
			ProviderSlug: "provider", ProviderName: "Provider", GroupKey: "group", GroupName: "Group", EnabledAccountCount: 1,
			CurrentAccountID: 1, CurrentAccountName: "account", LowestAccountIDs: []int64{1}, LowestAccountNames: []string{"account"},
			EstimatedExtraCost: &zeroFloat, CostCurrency: nil, ComparisonStatus: SupplierDashboardComparisonStatusLowest, TargetPath: "/rates",
		}},
		Total: 1, Page: 1, PageSize: 20, Warnings: []SupplierDashboardWarning{}, GeneratedAt: now,
	}
	provider := SupplierDashboardProvidersResponse{
		Range: SupplierDashboardRange24Hours,
		Items: []SupplierDashboardProviderItem{{
			ProviderSlug: "provider", ProviderName: "Provider", Enabled: true, Status: SupplierDashboardProviderStatusHealthy,
			CriticalIssueCount: &zeroInt, EnabledAccountCount: 1, SchedulableAccountCount: 1, RequestCount: &zeroInt64,
			SuccessRate: nil, CostCurrency: nil, BalanceCurrency: nil, TargetPath: "/providers",
		}},
		Total: 1, Page: 1, PageSize: 20, Warnings: []SupplierDashboardWarning{}, GeneratedAt: now,
	}

	assertDashboardJSONKeys(t, account, []string{"range", "items", "total", "page", "page_size", "warnings", "generated_at"}, []string{
		"account_id", "account_name", "provider_slug", "provider_name", "group_key", "group_name", "severity", "risk_types", "request_count", "success_rate", "current_rate", "lowest_rate", "rate_delta_percent", "balance", "balance_currency", "estimated_days", "status", "reason", "period_cost", "estimated_extra_cost", "traffic_impact", "detected_at", "target_path",
	})
	assertDashboardJSONKeys(t, rate, []string{"range", "items", "total", "page", "page_size", "warnings", "generated_at"}, []string{
		"provider_slug", "provider_name", "group_key", "group_name", "enabled_account_count", "current_account_id", "current_account_name", "current_rate", "lowest_rate", "lowest_account_ids", "lowest_account_names", "rate_delta_percent", "estimated_extra_cost", "cost_currency", "comparison_status", "last_synced_at", "target_path",
	})
	assertDashboardJSONKeys(t, provider, []string{"range", "items", "total", "page", "page_size", "warnings", "generated_at"}, []string{
		"provider_slug", "provider_name", "enabled", "status", "critical_issue_count", "enabled_account_count", "schedulable_account_count", "request_count", "success_rate", "period_cost", "cost_currency", "balance", "balance_currency", "estimated_days", "rate_risk_count", "balance_risk", "sync_risk", "target_path",
	})

	payload := marshalDashboardJSONMap(t, account)
	if _, ok := payload["page_size"]; !ok {
		t.Fatal("page_size missing")
	}
	items := payload["items"].([]any)
	accountMap := items[0].(map[string]any)
	if value, ok := accountMap["success_rate"]; !ok || value != nil {
		t.Fatalf("success_rate=%v present=%v", value, ok)
	}
	if value := accountMap["balance"]; value != float64(0) {
		t.Fatalf("balance zero encoded as %v", value)
	}
	providerPayload := marshalDashboardJSONMap(t, provider)
	providerMap := providerPayload["items"].([]any)[0].(map[string]any)
	if value := providerMap["critical_issue_count"]; value != float64(0) {
		t.Fatalf("critical_issue_count zero encoded as %v", value)
	}
}

func assertDashboardJSONKeys(t *testing.T, value any, topKeys, itemKeys []string) {
	t.Helper()
	payload := marshalDashboardJSONMap(t, value)
	if got := sortedDashboardKeys(payload); !reflect.DeepEqual(got, sortedDashboardStrings(topKeys)) {
		t.Fatalf("top-level keys=%v want=%v", got, sortedDashboardStrings(topKeys))
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%T %v", payload["items"], payload["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("item=%T", items[0])
	}
	if got := sortedDashboardKeys(item); !reflect.DeepEqual(got, sortedDashboardStrings(itemKeys)) {
		t.Fatalf("item keys=%v want=%v", got, sortedDashboardStrings(itemKeys))
	}
	for key := range item {
		if key != strings.ToLower(key) || strings.Contains(key, "SuccessRate") || strings.Contains(key, "PageSize") {
			t.Fatalf("non-snake-case key %q", key)
		}
		for _, sensitive := range []string{"password", "token", "secret", "api_key", "private_key"} {
			if strings.Contains(key, sensitive) {
				t.Fatalf("sensitive key %q", key)
			}
		}
	}
}

func marshalDashboardJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func sortedDashboardKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedDashboardStrings(values []string) []string {
	result := append([]string{}, values...)
	sort.Strings(result)
	return result
}

func TestPageSliceHandlesHugePagesAndCopiesResults(t *testing.T) {
	original := []int{1, 2, 3}
	page := pageSlice(original, 1, 2)
	if !reflect.DeepEqual(page, []int{1, 2}) {
		t.Fatalf("page=%v", page)
	}
	page[0] = 99
	if original[0] != 1 {
		t.Fatalf("page aliases original: %v", original)
	}
	if got := pageSlice(original, math.MaxInt, 100); got == nil || len(got) != 0 {
		t.Fatalf("huge page=%v", got)
	}
	if got := pageSlice([]int{}, 1, 20); got == nil || len(got) != 0 {
		t.Fatalf("empty page=%v", got)
	}
	single := []int{7}
	got := pageSlice(single, 1, 20)
	if !reflect.DeepEqual(got, single) {
		t.Fatalf("single=%v", got)
	}
	got[0] = 8
	if single[0] != 7 {
		t.Fatalf("single page aliases original: %v", single)
	}
}

func TestSupplierDashboardDetailQueriesTrimStringsAndValidateComparisonStatus(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	repo := &dashboardDetailStub{}
	svc := newDashboardDetailService(now, repo)
	if _, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: " 24h ", RiskType: " all ", ProviderSlug: " p ", GroupKey: " g "}); err != nil {
		t.Fatalf("accounts trim: %v", err)
	}
	if _, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: " 24h ", View: " all ", ComparisonStatus: " lowest ", ProviderSlug: " p ", GroupKey: " g "}); err != nil {
		t.Fatalf("rates trim: %v", err)
	}
	if repo.accountProviderSlug != "p" || repo.accountGroupKey != "g" || repo.rateProviderSlug != "p" || repo.rateGroupKey != "g" {
		t.Fatalf("repository filters accounts=(%q,%q) rates=(%q,%q)", repo.accountProviderSlug, repo.accountGroupKey, repo.rateProviderSlug, repo.rateGroupKey)
	}
	if _, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: " 24h ", Status: " healthy "}); err != nil {
		t.Fatalf("providers trim: %v", err)
	}
	if _, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours, ComparisonStatus: "bogus"}); err == nil {
		t.Fatal("invalid comparison_status must fail")
	}
}

func TestSupplierDashboardConstructorsRequireDetailRepository(t *testing.T) {
	repo := &dashboardDetailStub{}
	constructed := NewSupplierDashboardService(repo, nil)
	if constructed.detail != repo {
		t.Fatal("constructor did not retain required detail repository")
	}
	provided := ProvideSupplierDashboardService(repo, nil)
	if provided.detail != repo {
		t.Fatal("wire provider did not retain required detail repository")
	}
}

func dashboardHasRisk(risks []SupplierDashboardRiskType, target SupplierDashboardRiskType) bool {
	for _, risk := range risks {
		if risk == target {
			return true
		}
	}
	return false
}

func TestSupplierDashboardProviderStatusFiltersHaveProductionPaths(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	zero, one := int64(0), int64(1)
	repo := &dashboardDetailStub{providers: []SupplierDashboardProviderSnapshot{
		{ProviderSlug: "healthy", Enabled: true, DataComplete: true, SuccessCount: &zero, ErrorCount: &zero},
		{ProviderSlug: "warning", Enabled: true, DataComplete: true, SuccessCount: &zero, ErrorCount: &zero, RateRiskCount: 1},
		{ProviderSlug: "high", Enabled: true, DataComplete: true, EnabledAccountCount: 1, SuccessCount: &zero, ErrorCount: &one},
		{ProviderSlug: "disabled", Enabled: false},
		{ProviderSlug: "unknown", Enabled: true, DataComplete: false},
	}}
	svc := newDashboardDetailService(now, repo)
	for status, want := range map[SupplierDashboardProviderStatus]string{
		SupplierDashboardProviderStatusHealthy:  "healthy",
		SupplierDashboardProviderStatusWarning:  "warning",
		SupplierDashboardProviderStatusHighRisk: "high",
		SupplierDashboardProviderStatusDisabled: "disabled",
		SupplierDashboardProviderStatusUnknown:  "unknown",
	} {
		result, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: SupplierDashboardRange24Hours, Status: status})
		if err != nil || len(result.Items) != 1 || result.Items[0].ProviderSlug != want {
			t.Fatalf("status=%q items=%+v err=%v", status, result.Items, err)
		}
	}
}

func TestSupplierDashboardServiceMapsNeutralAccountFactsAndNullZeroSemantics(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	zeroFloat := 0.0
	zero, one := int64(0), int64(1)
	repo := &dashboardDetailStub{accounts: []SupplierDashboardAccountSnapshot{
		{AccountID: 1, AccountName: "flow", ProviderSlug: "provider", ProviderName: "Provider", ProviderEnabled: true, AccountEnabled: true, GroupKey: "group", GroupName: "Group", AccountStatus: "active", SuccessCount: &one, ErrorCount: &one, CurrentRate: &zeroFloat, SnapshotCount: 1, Balance: &zeroFloat, EstimatedDays: &zeroFloat, PeriodCost: &zeroFloat, ObservedAt: now},
		{AccountID: 2, AccountName: "idle", ProviderSlug: "provider", ProviderName: "Provider", ProviderEnabled: true, AccountEnabled: true, SuccessCount: &zero, ErrorCount: &zero},
		{AccountID: 3, AccountName: "unmatched", ProviderSlug: "provider", ProviderName: "Provider", ProviderEnabled: true, AccountEnabled: true},
	}}
	svc := newDashboardDetailService(now, repo)
	result, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: SupplierDashboardRange24Hours, RiskType: SupplierDashboardRiskTypeAll})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]SupplierDashboardAccountItem{}
	for _, item := range result.Items {
		byID[item.AccountID] = item
	}
	flow := byID[1]
	if flow.AccountName != "flow" || flow.ProviderName != "Provider" || flow.GroupKey != "group" || flow.GroupName != "Group" || flow.Status != "active" {
		t.Fatalf("identity/status mapping=%+v", flow)
	}
	if flow.RequestCount == nil || *flow.RequestCount != 2 || flow.SuccessRate == nil || *flow.SuccessRate != 50 {
		t.Fatalf("traffic mapping=%+v", flow)
	}
	for name, value := range map[string]*float64{"current_rate": flow.CurrentRate, "balance": flow.Balance, "estimated_days": flow.EstimatedDays, "period_cost": flow.PeriodCost} {
		if value == nil || *value != 0 {
			t.Fatalf("%s must preserve real zero: %+v", name, flow)
		}
	}
	if flow.RateDeltaPercent != nil || flow.LowestRate != nil || flow.BalanceCurrency != nil {
		t.Fatalf("unavailable facts must stay null: %+v", flow)
	}
	idle := byID[2]
	if idle.RequestCount == nil || *idle.RequestCount != 0 || idle.SuccessRate != nil {
		t.Fatalf("matched idle account=%+v", idle)
	}
	unmatched := byID[3]
	if unmatched.RequestCount != nil || unmatched.SuccessRate != nil || unmatched.PeriodCost != nil {
		t.Fatalf("unmatched account=%+v", unmatched)
	}
}

func TestSupplierDashboardAccountDetectedAtUsesMatchedRiskEvidenceTimes(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	oldAccountTime := now.Add(-30 * 24 * time.Hour)
	olderBalanceTime := now.Add(-2 * time.Hour)
	newBalanceTime := now.Add(-time.Hour)
	providerRiskTime := now.Add(-30 * time.Minute)
	days := 2.0

	repo := &dashboardDetailStub{accounts: []SupplierDashboardAccountSnapshot{
		{
			AccountID: 1, AccountName: "new-balance", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true,
			BalanceSyncStatus: "success", BalanceSyncedAt: &newBalanceTime, EstimatedDays: &days, ObservedAt: oldAccountTime,
		},
		{
			AccountID: 2, AccountName: "old-balance", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true,
			BalanceSyncStatus: "success", BalanceSyncedAt: &olderBalanceTime, EstimatedDays: &days, ObservedAt: now,
		},
		{
			AccountID: 3, AccountName: "partial-balance", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true,
			BalanceSyncStatus: "unknown", BalanceSyncedAt: &now, EstimatedDays: &days, ObservedAt: now,
		},
		{
			AccountID: 4, AccountName: "critical", ProviderSlug: "p", ProviderEnabled: true, AccountEnabled: true,
			ProviderRiskLevel: "critical", ProviderRiskUpdatedAt: &providerRiskTime, ObservedAt: oldAccountTime,
		},
	}}
	svc := newDashboardDetailService(now, repo)

	balance, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: SupplierDashboardRange24Hours, RiskType: SupplierDashboardRiskTypeBalance})
	if err != nil {
		t.Fatal(err)
	}
	if len(balance.Items) != 2 || balance.Items[0].AccountID != 1 || !balance.Items[0].DetectedAt.Equal(newBalanceTime) || balance.Items[1].AccountID != 2 || !balance.Items[1].DetectedAt.Equal(olderBalanceTime) {
		t.Fatalf("balance evidence sorting = %+v", balance.Items)
	}

	critical, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: SupplierDashboardRange24Hours, RiskType: SupplierDashboardRiskTypeCritical})
	if err != nil {
		t.Fatal(err)
	}
	if len(critical.Items) != 1 || critical.Items[0].AccountID != 4 || !critical.Items[0].DetectedAt.Equal(providerRiskTime) {
		t.Fatalf("provider risk evidence time = %+v", critical.Items)
	}
}

type supplierDashboardErrorRepository struct{ err error }

func (r *supplierDashboardErrorRepository) ListDashboardAccounts(context.Context, time.Time, time.Time, string, string) ([]SupplierDashboardAccountSnapshot, error) {
	return nil, r.err
}
func (r *supplierDashboardErrorRepository) ListDashboardRates(context.Context, time.Time, time.Time, string, string) ([]SupplierDashboardRateSnapshot, error) {
	return nil, r.err
}
func (r *supplierDashboardErrorRepository) ListDashboardProviders(context.Context, time.Time, time.Time) ([]SupplierDashboardProviderSnapshot, error) {
	return nil, r.err
}
func (r *supplierDashboardErrorRepository) ListDashboardAccountTraffic(context.Context, time.Time, time.Time, string, string) ([]SupplierDashboardTrafficSnapshot, error) {
	return nil, r.err
}
func (r *supplierDashboardErrorRepository) ListDashboardAccountProfit(context.Context, time.Time, time.Time, string, string, int) ([]SupplierDashboardProfitSnapshot, error) {
	return nil, r.err
}
func (r *supplierDashboardErrorRepository) ListDashboardAccountHealth(context.Context, time.Time, time.Time, string, string, int) ([]SupplierDashboardHealthSnapshot, error) {
	return nil, r.err
}

func TestSupplierDashboardContextErrorsPropagate(t *testing.T) {
	now := time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC)
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		for _, wrapped := range []error{cause, fmt.Errorf("wrapped: %w", cause)} {
			repo := &supplierDashboardErrorRepository{err: wrapped}
			svc := NewSupplierDashboardService(repo, nil)
			svc.now = func() time.Time { return now }
			calls := []func() error{
				func() error {
					_, err := svc.GetAccounts(context.Background(), SupplierDashboardAccountsQuery{Range: SupplierDashboardRange24Hours})
					return err
				},
				func() error {
					_, err := svc.GetRates(context.Background(), SupplierDashboardRatesQuery{Range: SupplierDashboardRange24Hours})
					return err
				},
				func() error {
					_, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: SupplierDashboardRange24Hours})
					return err
				},
				func() error {
					_, err := svc.GetAccountTraffic(context.Background(), SupplierDashboardTrafficQuery{Range: SupplierDashboardRange24Hours})
					return err
				},
				func() error {
					_, err := svc.GetAccountProfitRanking(context.Background(), SupplierDashboardProfitQuery{Range: SupplierDashboardRange24Hours})
					return err
				},
				func() error {
					_, err := svc.GetAccountHealthTimeline(context.Background(), SupplierDashboardAccountHealthQuery{Range: SupplierDashboardRange24Hours})
					return err
				},
			}
			for _, call := range calls {
				if err := call(); !errors.Is(err, cause) {
					t.Fatalf("error %v did not propagate %v", err, cause)
				}
			}
		}
	}
}

func TestSupplierDashboardThresholdContextErrorPropagates(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		svc := NewSupplierDashboardService(&dashboardDetailStub{}, &dashboardThresholdStub{err: fmt.Errorf("threshold: %w", cause)})
		svc.now = func() time.Time { return time.Date(2026, 7, 24, 8, 0, 0, 0, time.UTC) }
		if _, err := svc.GetProviders(context.Background(), SupplierDashboardProvidersQuery{Range: SupplierDashboardRange24Hours}); !errors.Is(err, cause) {
			t.Fatalf("threshold error %v did not propagate %v", err, cause)
		}
	}
}

func TestSupplierDashboardTrafficOverflowIsUnknown(t *testing.T) {
	max, one := int64(math.MaxInt64), int64(1)
	count, rate := dashboardTraffic(&max, &one)
	if count != nil || rate != nil {
		t.Fatalf("overflow traffic = %v, %v", count, rate)
	}
	if got := dashboardRequestCountValue(&max, &one); got != -1 {
		t.Fatalf("overflow request count = %d", got)
	}
}
