package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryGetGlobalDailyStatsAggregated(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2026, 7, 9, 16, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 10, 16, 0, 0, 0, time.UTC)

	mock.ExpectQuery("WHERE created_at >= \\$1 AND created_at < \\$2").
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"date",
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_tokens",
			"total_cost",
			"total_actual_cost",
			"avg_duration_ms",
		}).AddRow("2026-07-10", int64(2), int64(10), int64(20), int64(5), 0.3, 0.25, 123.4))

	rows, err := repo.GetGlobalDailyStatsAggregated(context.Background(), start, end)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, map[string]any{
		"date":                "2026-07-10",
		"total_requests":      int64(2),
		"total_input_tokens":  int64(10),
		"total_output_tokens": int64(20),
		"total_cache_tokens":  int64(5),
		"total_tokens":        int64(35),
		"total_cost":          0.3,
		"total_actual_cost":   0.25,
		"average_duration_ms": 123.4,
	}, rows[0])
	require.NoError(t, mock.ExpectationsWereMet())
}
