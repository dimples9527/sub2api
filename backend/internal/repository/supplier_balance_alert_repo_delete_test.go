package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierBalanceAlertRepositoryDeleteEventDeletesResolvedEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM supplier_balance_alert_events WHERE id = $1 FOR UPDATE`)).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("resolved"))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM supplier_balance_alert_events WHERE id = $1 AND status = 'resolved'`)).
		WithArgs(int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &supplierBalanceAlertRepository{db: db}
	require.NoError(t, repo.DeleteEvent(context.Background(), 12))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierBalanceAlertRepositoryDeleteEventRejectsActiveEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM supplier_balance_alert_events WHERE id = $1 FOR UPDATE`)).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))
	mock.ExpectRollback()

	repo := &supplierBalanceAlertRepository{db: db}
	err = repo.DeleteEvent(context.Background(), 12)
	require.ErrorIs(t, err, service.ErrSupplierBalanceAlertEventActive)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierBalanceAlertRepositoryDeleteEventReturnsNotFoundForMissingEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status FROM supplier_balance_alert_events WHERE id = $1 FOR UPDATE`)).
		WithArgs(int64(12)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	repo := &supplierBalanceAlertRepository{db: db}
	err = repo.DeleteEvent(context.Background(), 12)
	require.ErrorIs(t, err, service.ErrSupplierBalanceAlertEventNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierBalanceAlertRepositoryListConfigsExcludesDeletedProviders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`(?s)FROM supplier_providers p\s+LEFT JOIN supplier_balance_alert_configs c ON c\.provider_id = p\.id\s+WHERE p\.deleted_at IS NULL\s+ORDER BY p\.sort_order ASC, p\.id ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "provider_code", "provider_name", "provider_type", "provider_enabled",
			"enabled", "threshold", "cooldown_seconds", "last_scan_at", "last_balance", "last_scan_status",
			"last_scan_error", "created_at", "updated_at",
		}).AddRow(
			int64(0), int64(7), "active-provider", "Active Provider", "custom", true,
			false, "0", 3600, nil, nil, "never", "", now, now,
		))

	repo := &supplierBalanceAlertRepository{db: db}
	items, err := repo.ListConfigs(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(7), items[0].ProviderID)
	require.NoError(t, mock.ExpectationsWereMet())
}
