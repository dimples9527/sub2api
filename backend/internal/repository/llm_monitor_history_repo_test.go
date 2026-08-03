package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestLLMMonitorHistoryRepositorySaveSnapshotUpsertsLatestSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO llm_monitor_histories")+
		".*"+
		regexp.QuoteMeta("ON CONFLICT (source_key, period, board) DO UPDATE SET"+
			" payload = EXCLUDED.payload,"+
			" payload_hash = EXCLUDED.payload_hash,"+
			" captured_at = EXCLUDED.captured_at"+
			" WHERE llm_monitor_histories.payload_hash IS DISTINCT FROM EXCLUDED.payload_hash"+
			" OR llm_monitor_histories.captured_at < EXCLUDED.captured_at - INTERVAL '1 day'")).
		WithArgs("source-key", "24h", "hot", []byte(`{"groups":[]}`), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewLLMMonitorHistoryRepository(db)
	err = repo.SaveSnapshot(context.Background(), service.LLMMonitorHistorySnapshot{
		SourceKey:  "source-key",
		Period:     "24h",
		Board:      "hot",
		Payload:    []byte(`{"groups":[]}`),
		CapturedAt: time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC),
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLLMMonitorHistoryRepositoryLoadLatestSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	capturedAt := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT source_key, period, board, payload, captured_at")).
		WithArgs("source-key", "24h", "hot").
		WillReturnRows(sqlmock.NewRows([]string{"source_key", "period", "board", "payload", "captured_at"}).
			AddRow("source-key", "24h", "hot", []byte(`{"groups":[]}`), capturedAt))

	repo := NewLLMMonitorHistoryRepository(db)
	snapshot, err := repo.LoadLatestSnapshot(context.Background(), "source-key", "24h", "hot")

	require.NoError(t, err)
	require.Equal(t, &service.LLMMonitorHistorySnapshot{
		SourceKey:  "source-key",
		Period:     "24h",
		Board:      "hot",
		Payload:    []byte(`{"groups":[]}`),
		CapturedAt: capturedAt,
	}, snapshot)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLLMMonitorHistoryRepositoryDeleteBeforeUsesBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	cutoff := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM llm_monitor_histories")).
		WithArgs(cutoff, 500).
		WillReturnResult(sqlmock.NewResult(0, 3))

	repo := NewLLMMonitorHistoryRepository(db)
	deleted, err := repo.DeleteBefore(context.Background(), cutoff, 500)

	require.NoError(t, err)
	require.Equal(t, int64(3), deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}
