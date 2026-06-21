package service

import (
	"context"
	"testing"
	"time"
)

type upstreamBalanceConsumptionMemoryStore struct {
	snapshots []UpstreamBalanceSnapshot
}

func (s *upstreamBalanceConsumptionMemoryStore) AddSnapshot(_ context.Context, snapshot UpstreamBalanceSnapshot) (UpstreamBalanceSnapshot, error) {
	snapshot.ID = int64(len(s.snapshots) + 1)
	s.snapshots = append(s.snapshots, snapshot)
	return snapshot, nil
}

func (s *upstreamBalanceConsumptionMemoryStore) ListSnapshots(_ context.Context, startTime, endTime time.Time) ([]UpstreamBalanceSnapshot, error) {
	out := []UpstreamBalanceSnapshot{}
	for _, snapshot := range s.snapshots {
		if !snapshot.CapturedAt.Before(startTime) && snapshot.CapturedAt.Before(endTime) {
			out = append(out, snapshot)
		}
	}
	return out, nil
}

func (s *upstreamBalanceConsumptionMemoryStore) ListSnapshotsBefore(_ context.Context, before time.Time) ([]UpstreamBalanceSnapshot, error) {
	latestByProvider := map[string]UpstreamBalanceSnapshot{}
	for _, snapshot := range s.snapshots {
		if snapshot.CapturedAt.Before(before) && snapshot.Status == "success" {
			existing, ok := latestByProvider[snapshot.ProviderSlug]
			if !ok || snapshot.CapturedAt.After(existing.CapturedAt) {
				latestByProvider[snapshot.ProviderSlug] = snapshot
			}
		}
	}
	out := make([]UpstreamBalanceSnapshot, 0, len(latestByProvider))
	for _, snapshot := range latestByProvider {
		out = append(out, snapshot)
	}
	return out, nil
}

func (s *upstreamBalanceConsumptionMemoryStore) ListLatestSnapshots(_ context.Context) ([]UpstreamBalanceSnapshot, error) {
	return s.ListSnapshotsBefore(context.Background(), time.Now().Add(24*time.Hour))
}

func (s *upstreamBalanceConsumptionMemoryStore) AddRecharge(_ context.Context, input UpstreamBalanceRechargeInput) (UpstreamBalanceRecharge, error) {
	return UpstreamBalanceRecharge{ProviderSlug: input.ProviderSlug, Amount: input.Amount, AmountScale: input.AmountScale, OccurredAt: input.OccurredAt}, nil
}

func (s *upstreamBalanceConsumptionMemoryStore) ListRecharges(_ context.Context, _, _ time.Time) ([]UpstreamBalanceRecharge, error) {
	return []UpstreamBalanceRecharge{}, nil
}

type upstreamBalanceConsumptionProviderStub struct {
	providers []UpstreamProviderConfig
	balances  map[string]UpstreamProviderBalance
	costs     map[string]UpstreamProviderCost
}

func (s *upstreamBalanceConsumptionProviderStub) ListProviders(context.Context) ([]UpstreamProviderConfig, error) {
	return s.providers, nil
}

func (s *upstreamBalanceConsumptionProviderStub) FetchProviderBalance(_ context.Context, slug string) (UpstreamProviderBalance, error) {
	return s.balances[slug], nil
}

func (s *upstreamBalanceConsumptionProviderStub) FetchProviderTodayCost(_ context.Context, slug string, _ time.Time) (UpstreamProviderCost, error) {
	return s.costs[slug], nil
}

type upstreamBalanceConsumptionUsageStub struct {
	rows      []map[string]any
	userID    int64
	startTime time.Time
	endTime   time.Time
}

func (s *upstreamBalanceConsumptionUsageStub) GetGlobalDailyStatsAggregated(_ context.Context, startTime, endTime time.Time) ([]map[string]any, error) {
	s.userID = 0
	s.startTime = startTime
	s.endTime = endTime
	return s.rows, nil
}

func TestUpstreamBalanceConsumptionDailyComplete(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, TodayCost: 70, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
		},
		[]UpstreamBalanceRecharge{
			{ProviderSlug: "backup", Amount: 50, OccurredAt: day.Add(12 * time.Hour)},
		},
		day,
		day.Add(24*time.Hour),
		1,
	)

	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if !row.Complete {
		t.Fatalf("row should be complete: %+v", row)
	}
	if row.OpeningBalance != 100 || row.ClosingBalance != 80 || row.RechargeAmount != 50 {
		t.Fatalf("balances = %+v, want opening 100 closing 80 recharge 50", row)
	}
	if row.ConsumptionAmount != 70 {
		t.Fatalf("consumption = %v, want 70", row.ConsumptionAmount)
	}
	if row.Anomaly {
		t.Fatalf("row should not be anomalous: %+v", row)
	}
}

func TestUpstreamBalanceConsumptionDailyUsesLatestDirectTodayCost(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, TodayCost: 12.5, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, TodayCost: 33.75, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
		},
		nil,
		day,
		day.Add(24*time.Hour),
		1,
	)

	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if !row.Complete {
		t.Fatalf("row should be complete with direct today cost: %+v", row)
	}
	if row.ConsumptionAmount != 33.75 {
		t.Fatalf("consumption = %v, want latest direct today cost", row.ConsumptionAmount)
	}
	if row.RechargeAmount != 0 {
		t.Fatalf("recharge should not affect direct cost, got %+v", row)
	}
}

func TestUpstreamBalanceConsumptionDailyDirectTodayCostWorksWithOneSnapshot(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, TodayCost: 9.25, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
		},
		nil,
		day,
		day.Add(24*time.Hour),
		1,
	)

	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if !row.Complete {
		t.Fatalf("row should be complete with one direct cost snapshot: %+v", row)
	}
	if row.ConsumptionAmount != 9.25 {
		t.Fatalf("consumption = %v, want direct today cost", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionDailySingleSnapshotUsesDirectCost(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, TodayCost: 8.5, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
		},
		nil,
		day,
		day.Add(24*time.Hour),
		1,
	)

	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if !row.Complete {
		t.Fatalf("row should be complete with direct cost: %+v", row)
	}
	if row.ConsumptionAmount != 8.5 {
		t.Fatalf("consumption = %v, want direct cost", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionDailyFlagsNegativeConsumption(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 130, TodayCost: -30, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
		},
		nil,
		day,
		day.Add(24*time.Hour),
		1,
	)

	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ConsumptionAmount != -30 {
		t.Fatalf("consumption = %v, want -30", row.ConsumptionAmount)
	}
	if !row.Anomaly {
		t.Fatalf("negative consumption should be anomalous: %+v", row)
	}
}

func TestUpstreamBalanceConsumptionDailyAppliesAmountScale(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, TodayCost: 70, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
		},
		[]UpstreamBalanceRecharge{
			{ProviderSlug: "backup", Amount: 50, OccurredAt: day.Add(12 * time.Hour)},
		},
		day,
		day.Add(24*time.Hour),
		0.1,
	)

	row := rows[0]
	if row.OpeningBalance != 10 || row.ClosingBalance != 8 || row.RechargeAmount != 5 {
		t.Fatalf("scaled balances = %+v, want 10/8/5", row)
	}
	if row.ConsumptionAmount != 70 {
		t.Fatalf("consumption = %v, want direct cost unaffected by amount scale", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionDailySkipsFailedSnapshots(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 0, CapturedAt: day.Add(12 * time.Hour), Status: "failed", Error: "upstream error"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, TodayCost: 20, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
		},
		nil,
		day,
		day.Add(24*time.Hour),
		1,
	)

	row := rows[0]
	if row.SnapshotCount != 2 {
		t.Fatalf("snapshot count = %d, want 2", row.SnapshotCount)
	}
	if row.ConsumptionAmount != 20 {
		t.Fatalf("consumption = %v, want 20", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionDailyUsesSnapshotScaleAtCaptureTime(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, AmountScale: 1, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, TodayCost: 60, AmountScale: 0.5, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
		},
		nil,
		day,
		day.Add(24*time.Hour),
		1,
	)

	row := rows[0]
	if row.OpeningBalance != 100 || row.ClosingBalance != 40 {
		t.Fatalf("scaled balances = %+v, want opening 100 closing 40", row)
	}
	if row.ConsumptionAmount != 60 {
		t.Fatalf("consumption = %v, want 60", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionDailyUsesPreviousDayOpeningSnapshot(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	start := time.Date(2026, 6, 15, 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	rows := buildUpstreamBalanceDailyRowsInLocation(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 90, CapturedAt: start.Add(-2 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: start.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, TodayCost: 10, CapturedAt: start.Add(23 * time.Hour), Status: "success"},
		},
		nil,
		start.UTC(),
		end.UTC(),
		1,
		loc,
	)

	row := rows[0]
	if row.Date != "2026-06-15" {
		t.Fatalf("date = %s, want 2026-06-15", row.Date)
	}
	if row.OpeningBalance != 90 {
		t.Fatalf("opening balance = %v, want 90", row.OpeningBalance)
	}
	if row.ClosingBalance != 80 {
		t.Fatalf("closing balance = %v, want 80", row.ClosingBalance)
	}
	if row.ConsumptionAmount != 10 {
		t.Fatalf("consumption = %v, want 10", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionRunSampleStoresTodayCost(t *testing.T) {
	store := &upstreamBalanceConsumptionMemoryStore{}
	provider := &upstreamBalanceConsumptionProviderStub{
		providers: []UpstreamProviderConfig{
			{Type: UpstreamProviderTypeSub2API, Slug: "sub-main", Name: "Sub Main", Enabled: true},
		},
		balances: map[string]UpstreamProviderBalance{
			"sub-main": {ProviderSlug: "sub-main", ProviderName: "Sub Main", ProviderType: UpstreamProviderTypeSub2API, Balance: 80},
		},
		costs: map[string]UpstreamProviderCost{
			"sub-main": {ProviderSlug: "sub-main", ProviderName: "Sub Main", ProviderType: UpstreamProviderTypeSub2API, TodayCost: 12.5},
		},
	}
	svc := NewUpstreamBalanceConsumptionService(store, provider, nil)
	now := time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	if _, err := svc.RunSample(context.Background()); err != nil {
		t.Fatalf("RunSample returned error: %v", err)
	}
	if len(store.snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(store.snapshots))
	}
	snapshot := store.snapshots[0]
	if snapshot.Balance != 80 || snapshot.TodayCost != 12.5 {
		t.Fatalf("snapshot = %+v, want balance and today cost stored", snapshot)
	}
}

func TestUpstreamBalanceConsumptionOverviewIncludesLocalDailyConsumption(t *testing.T) {
	store := &upstreamBalanceConsumptionMemoryStore{}
	usage := &upstreamBalanceConsumptionUsageStub{rows: []map[string]any{
		{"date": "2026-06-16", "total_actual_cost": 12.34},
		{"date": "2026-06-17", "total_actual_cost": int64(8)},
	}}
	svc := NewUpstreamBalanceConsumptionService(store, &upstreamBalanceConsumptionProviderStub{}, nil)
	svc.SetLocalDailyUsageSource(usage)
	svc.now = func() time.Time { return time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC) }

	overview, err := svc.GetOverview(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetOverview returned error: %v", err)
	}

	if usage.userID != 0 {
		t.Fatalf("usage userID = %d, want 0 for global daily usage", usage.userID)
	}
	if len(overview.LocalDailyConsumptions) != 2 {
		t.Fatalf("local daily consumption count = %d, want 2", len(overview.LocalDailyConsumptions))
	}
	if overview.LocalDailyConsumptions[0].Date != "2026-06-16" || overview.LocalDailyConsumptions[0].ActualCost != 12.34 {
		t.Fatalf("first local consumption = %+v, want 2026-06-16/12.34", overview.LocalDailyConsumptions[0])
	}
	if overview.LocalDailyConsumptions[1].Date != "2026-06-17" || overview.LocalDailyConsumptions[1].ActualCost != 8 {
		t.Fatalf("second local consumption = %+v, want 2026-06-17/8", overview.LocalDailyConsumptions[1])
	}
}
