package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUpstreamAccountHealthGuardRecordRepositoryListRecordsLoadsLatestItemsAndHistoricalAdjustments(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	runRows := sqlmock.NewRows([]string{
		"id", "trigger", "status", "message", "started_at", "finished_at",
		"total_accounts", "checked_count", "healthy_count", "slow_count", "failed_count",
		"skipped_count", "disabled_count", "recovered_count", "unchanged_count", "skip_reasons",
	}).
		AddRow("run-new", "manual", "success", "checked", now, now.Add(time.Second), 3, 2, 1, 1, 0, 1, 0, 0, 2, []byte(`[{"reason":"missing_provider_slug","count":1}]`)).
		AddRow("run-old", "scheduled", "success", "old", now.Add(-time.Hour), now.Add(-time.Hour+time.Second), 3, 3, 3, 0, 0, 0, 0, 0, 3, []byte(`[]`))
	mock.ExpectQuery(regexp.QuoteMeta("FROM upstream_account_health_guard_runs")).
		WithArgs(50).
		WillReturnRows(runRows)

	itemRows := sqlmock.NewRows([]string{
		"account_id", "account_name", "platform", "provider_slug", "provider_name",
		"model_id", "schedulable_before", "schedulable_after", "status", "test_status",
		"latency_ms", "latency_limit_ms", "consecutive_failed", "consecutive_slow",
		"consecutive_healthy", "action", "reason", "error_message", "started_at", "finished_at",
	}).AddRow(
		1, "account", "openai", "main", "Main", "gpt-4o-mini",
		true, true, "healthy", "success", 100, 15000, 0, 0, 1, "none", "test passed", "", now, now.Add(100*time.Millisecond),
	)
	mock.ExpectQuery(regexp.QuoteMeta("FROM upstream_account_health_guard_run_items")).
		WithArgs("run-new").
		WillReturnRows(itemRows)

	adjustedRows := sqlmock.NewRows([]string{
		"run_id", "account_id", "account_name", "platform", "provider_slug", "provider_name",
		"model_id", "schedulable_before", "schedulable_after", "status", "test_status",
		"latency_ms", "latency_limit_ms", "consecutive_failed", "consecutive_slow",
		"consecutive_healthy", "action", "reason", "error_message", "started_at", "finished_at",
	}).AddRow(
		"run-old", 2, "old-account", "openai", "main", "Main", "gpt-4o-mini",
		true, false, "failed", "failed", 0, 15000, 3, 0, 0, "disabled", "failure threshold reached", "401", now.Add(-time.Hour), now.Add(-time.Hour+100*time.Millisecond),
	)
	mock.ExpectQuery(regexp.QuoteMeta("AND action IN ('disabled', 'recovered')")).
		WithArgs("run-old").
		WillReturnRows(adjustedRows)

	repo := NewUpstreamAccountHealthGuardRecordRepository(db)
	records, err := repo.ListRecords(context.Background(), 50)
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, "run-new", records[0].ID)
	require.Len(t, records[0].Summary.SkipReasons, 1)
	require.Equal(t, "missing_provider_slug", records[0].Summary.SkipReasons[0].Reason)
	require.Equal(t, 1, records[0].Summary.SkipReasons[0].Count)
	require.Len(t, records[0].Items, 1)
	require.Equal(t, int64(1), records[0].Items[0].AccountID)
	require.Equal(t, "run-old", records[1].ID)
	require.Len(t, records[1].Items, 1)
	require.Equal(t, int64(2), records[1].Items[0].AccountID)
	require.Equal(t, "disabled", records[1].Items[0].Action)
	require.NoError(t, mock.ExpectationsWereMet())
}
