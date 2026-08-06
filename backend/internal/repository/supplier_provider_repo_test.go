package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderRepositoryDisableAfterAuthFailureUpdatesProviderAndRuntimeStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	syncedAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	message := "供应商登录失败，已自动停用。"

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE supplier_providers
SET enabled=FALSE, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`)).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO supplier_provider_runtime_stats (
  provider_id, sync_status, sync_message, last_sync_at, updated_at
) VALUES ($1,$2,$3,$4,$4)
ON CONFLICT (provider_id) DO UPDATE SET
  sync_status=EXCLUDED.sync_status,
  sync_message=EXCLUDED.sync_message,
  last_sync_at=EXCLUDED.last_sync_at,
  updated_at=EXCLUDED.updated_at`)).
		WithArgs(int64(42), service.SupplierSyncStatusFailed, message, syncedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.DisableAfterAuthFailure(context.Background(), 42, message, syncedAt)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryDisableAfterAuthFailureReturnsNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE supplier_providers
SET enabled=FALSE, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`)).
		WithArgs(int64(404)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = repo.DisableAfterAuthFailure(context.Background(), 404, "登录失败", time.Now())

	require.ErrorIs(t, err, service.ErrSupplierProviderNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
