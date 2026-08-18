package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierCostDeviationSettingRepositoryGetThreshold(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierCostDeviationSettingRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT threshold FROM supplier_cost_deviation_settings WHERE id = 1`)).
		WillReturnRows(sqlmock.NewRows([]string{"threshold"}).AddRow(0.3))

	threshold, err := repo.GetThreshold(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0.3, threshold)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierCostDeviationSettingRepositoryGetThresholdFallsBackOnNoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierCostDeviationSettingRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT threshold FROM supplier_cost_deviation_settings WHERE id = 1`)).
		WillReturnRows(sqlmock.NewRows([]string{"threshold"}))

	threshold, err := repo.GetThreshold(context.Background())
	require.NoError(t, err)
	require.Equal(t, service.DefaultSupplierCostDeviationThreshold, threshold)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierCostDeviationSettingRepositoryGetThresholdError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierCostDeviationSettingRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT threshold FROM supplier_cost_deviation_settings WHERE id = 1`)).
		WillReturnError(errors.New("db down"))

	threshold, err := repo.GetThreshold(context.Background())
	require.Error(t, err)
	require.Zero(t, threshold)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierCostDeviationSettingRepositorySetThreshold(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierCostDeviationSettingRepository(db)
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO supplier_cost_deviation_settings (id, threshold)
VALUES (1, $1)
ON CONFLICT (id) DO UPDATE SET
  threshold = EXCLUDED.threshold,
  updated_at = NOW()`)).
		WithArgs(0.6).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.SetThreshold(context.Background(), 0.6))
	require.NoError(t, mock.ExpectationsWereMet())
}
