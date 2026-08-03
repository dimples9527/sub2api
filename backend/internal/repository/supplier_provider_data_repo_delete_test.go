package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderDataRepositoryDeletesOnlySafeInactiveGroupRecord(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*g\.active\s*=\s*FALSE.*g\.local_group_id\s+IS\s+NULL.*g\.rate_guard_selected\s*=\s*FALSE.*NOT\s+EXISTS\s*\(\s*SELECT\s+1\s+FROM\s+supplier_provider_accounts\s+a.*a\.provider_id\s*=\s*g\.provider_id.*a\.group_key\s*=\s*g\.upstream_group_key.*a\.active\s*=\s*TRUE.*RETURNING\s+g\.provider_id,\s*g\.upstream_group_key`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"provider_id", "upstream_group_key"}).AddRow(int64(3), "174"))
	mock.ExpectExec(`(?is)UPDATE\s+supplier_provider_accounts\s+AS\s+a.*SET\s+group_key\s*=\s*'',\s*group_name\s*=\s*'',\s*updated_at\s*=\s*NOW\(\).*a\.provider_id\s*=\s*\$1.*a\.group_key\s*=\s*\$2.*a\.active\s*=\s*FALSE`).
		WithArgs(int64(3), "174").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	require.NoError(t, repo.DeleteGroup(context.Background(), 7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryRejectsUnsafeInactiveGroupDelete(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*RETURNING\s+g\.provider_id,\s*g\.upstream_group_key`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"provider_id", "upstream_group_key"}))
	mock.ExpectRollback()

	err := repo.DeleteGroup(context.Background(), 7)
	require.ErrorIs(t, err, service.ErrSupplierProviderGroupDeleteConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryClearsInactiveAccountGroupReferenceWhenDeletingGroup(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*RETURNING\s+g\.provider_id,\s*g\.upstream_group_key`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"provider_id", "upstream_group_key"}).AddRow(int64(3), "174"))
	mock.ExpectExec(`(?is)UPDATE\s+supplier_provider_accounts\s+a.*SET\s+group_key\s*=\s*'',\s*group_name\s*=\s*''.*a\.provider_id\s*=\s*\$1.*a\.group_key\s*=\s*\$2.*a\.active\s*=\s*FALSE`).
		WithArgs(int64(3), "174").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	require.NoError(t, repo.DeleteGroup(context.Background(), 7))
	require.NoError(t, mock.ExpectationsWereMet())
}
