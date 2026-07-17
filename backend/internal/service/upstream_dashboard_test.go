package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func TestSortUpstreamDashboardIssuesUsesSeverityImpactAndTime(t *testing.T) {
	base := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	issues := []UpstreamDashboardIssue{
		{ID: "low", Severity: UpstreamDashboardSeverityLow, ImpactCount: 99, DetectedAt: base},
		{ID: "high-old", Severity: UpstreamDashboardSeverityHigh, ImpactCount: 3, DetectedAt: base.Add(-time.Hour)},
		{ID: "critical", Severity: UpstreamDashboardSeverityCritical, ImpactCount: 1, DetectedAt: base.Add(-2 * time.Hour)},
		{ID: "high-new", Severity: UpstreamDashboardSeverityHigh, ImpactCount: 3, DetectedAt: base},
		{ID: "high-impact", Severity: UpstreamDashboardSeverityHigh, ImpactCount: 8, DetectedAt: base.Add(-3 * time.Hour)},
	}

	sortUpstreamDashboardIssues(issues)

	want := []string{"critical", "high-impact", "high-new", "high-old", "low"}
	for index, id := range want {
		if issues[index].ID != id {
			t.Fatalf("issues[%d].ID = %q, want %q; issues=%+v", index, issues[index].ID, id, issues)
		}
	}
}

type dashboardProviderSource struct {
	items []UpstreamProviderConfig
	err   error
}

func (s dashboardProviderSource) ListProviders(context.Context) ([]UpstreamProviderConfig, error) {
	return s.items, s.err
}

type dashboardGroupSource struct{ result UpstreamGroupCompareResult }

func (s dashboardGroupSource) CompareGroups(context.Context) (UpstreamGroupCompareResult, error) {
	return s.result, nil
}

type dashboardSyncSource struct{ result UpstreamAccountSyncResult }

func (s dashboardSyncSource) Preview(context.Context) (UpstreamAccountSyncResult, error) {
	return s.result, nil
}

func (s dashboardSyncSource) GetRateGuardConfig(context.Context) (UpstreamAccountRateGuardConfig, error) {
	return UpstreamAccountRateGuardConfig{Enabled: true, LastRunStatus: "success"}, nil
}

type dashboardBalanceSource struct {
	overview UpstreamBalanceConsumptionOverview
}

func (s dashboardBalanceSource) GetOverview(context.Context, int) (UpstreamBalanceConsumptionOverview, error) {
	return s.overview, nil
}

type dashboardHealthSource struct {
	records []UpstreamAccountHealthGuardRunRecord
}

func (s dashboardHealthSource) GetConfig(context.Context) (UpstreamAccountHealthGuardConfig, error) {
	return UpstreamAccountHealthGuardConfig{Enabled: true, LastRunStatus: "success"}, nil
}

func (s dashboardHealthSource) ListRecords(context.Context) ([]UpstreamAccountHealthGuardRunRecord, error) {
	return s.records, nil
}

type dashboardOpsSource struct{ overview *OpsDashboardOverview }

func (s dashboardOpsSource) GetDashboardOverview(context.Context, *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	return s.overview, nil
}

type dashboardUsageSource struct{ models []usagestats.ModelStat }

func (s dashboardUsageSource) GetModelStatsWithFiltersBySource(context.Context, time.Time, time.Time, usagestats.UsageLogFilters, string) ([]usagestats.ModelStat, error) {
	return s.models, nil
}

func TestUpstreamDashboardServiceGetAggregatesAvailableData(t *testing.T) {
	now := time.Date(2026, 7, 13, 8, 0, 0, 0, time.UTC)
	lastSnapshot := now.Add(-45 * time.Minute)
	service := NewUpstreamDashboardService(
		dashboardProviderSource{items: []UpstreamProviderConfig{{Slug: "main", Enabled: true}, {Slug: "backup", Enabled: false}}},
		dashboardGroupSource{result: UpstreamGroupCompareResult{Items: []UpstreamGroupComparison{{NeedsRateIncrease: true}}}},
		dashboardSyncSource{result: UpstreamAccountSyncResult{Summary: UpstreamAccountSyncSummary{MatchedAccountCount: 16, ConflictCount: 3}}},
		dashboardBalanceSource{overview: UpstreamBalanceConsumptionOverview{
			Config:    UpstreamBalanceSamplerConfig{Enabled: true, LastRunStatus: "failed", LastRunAt: dashboardTimePointer(now.Add(-time.Hour))},
			Summaries: map[string]UpstreamBalanceProviderSummary{"main": {ProviderSlug: "main", CurrentBalance: 120, TodayConsumption: 20, LastSnapshotAt: &lastSnapshot}},
		}},
		dashboardHealthSource{records: []UpstreamAccountHealthGuardRunRecord{{ID: "health", FinishedAt: now, Summary: UpstreamAccountHealthGuardRunSummary{FailedCount: 2}}}},
		dashboardOpsSource{overview: &OpsDashboardOverview{RequestCountTotal: 1000, SuccessCount: 980, SLA: 98, Duration: OpsPercentiles{P95: dashboardIntPointer(840)}}},
		dashboardUsageSource{models: []usagestats.ModelStat{{Model: "gpt-5", Requests: 700, AccountCost: 70}}},
	)
	service.now = func() time.Time { return now }

	result, err := service.Get(context.Background(), UpstreamDashboardRange24Hours)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.Summary.ProviderCount != 2 || result.Summary.DisabledProviderCount != 1 || result.Summary.RateRiskCount != 1 {
		t.Fatalf("summary = %+v", result.Summary)
	}
	if result.Stability.RequestCount != 1000 || result.Stability.SuccessRate != 98 || result.Stability.P95LatencyMs != 840 {
		t.Fatalf("stability = %+v", result.Stability)
	}
	if result.Cost.TotalBalance != 120 || result.Cost.PeriodCost != 20 || result.Cost.EstimatedDays == nil || *result.Cost.EstimatedDays != 6 {
		t.Fatalf("cost = %+v", result.Cost)
	}
	if len(result.ModelRankings) != 1 || result.ModelRankings[0].Model != "gpt-5" || len(result.Issues) < 4 {
		t.Fatalf("rankings/issues = %+v / %+v", result.ModelRankings, result.Issues)
	}
}

func TestUpstreamDashboardServiceGetKeepsPartialResults(t *testing.T) {
	service := NewUpstreamDashboardService(
		dashboardProviderSource{err: errors.New("providers unavailable")},
		dashboardGroupSource{}, dashboardSyncSource{}, dashboardBalanceSource{}, dashboardHealthSource{},
		dashboardOpsSource{overview: &OpsDashboardOverview{}}, dashboardUsageSource{},
	)
	result, err := service.Get(context.Background(), UpstreamDashboardRange7Days)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Source != "providers" {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
}

func dashboardIntPointer(value int) *int { return &value }

func dashboardTimePointer(value time.Time) *time.Time { return &value }
