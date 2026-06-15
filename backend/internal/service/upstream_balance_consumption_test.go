package service

import (
	"testing"
	"time"
)

func TestUpstreamBalanceConsumptionDailyComplete(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
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

func TestUpstreamBalanceConsumptionDailyIncompleteWithOneSnapshot(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
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
	if row.Complete {
		t.Fatalf("row should be incomplete: %+v", row)
	}
	if row.ConsumptionAmount != 0 {
		t.Fatalf("incomplete consumption = %v, want 0", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionDailyFlagsNegativeConsumption(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 130, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
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
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
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
	if row.ConsumptionAmount != 7 {
		t.Fatalf("scaled consumption = %v, want 7", row.ConsumptionAmount)
	}
}

func TestUpstreamBalanceConsumptionDailySkipsFailedSnapshots(t *testing.T) {
	day := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rows := BuildUpstreamBalanceDailyRows(
		[]UpstreamBalanceSnapshot{
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 100, CapturedAt: day.Add(1 * time.Hour), Status: "success"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 0, CapturedAt: day.Add(12 * time.Hour), Status: "failed", Error: "upstream error"},
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
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
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, AmountScale: 0.5, CapturedAt: day.Add(23 * time.Hour), Status: "success"},
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
			{ProviderSlug: "backup", ProviderName: "Backup", Balance: 80, CapturedAt: start.Add(23 * time.Hour), Status: "success"},
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
