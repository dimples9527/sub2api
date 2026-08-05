package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderDataRepositoryDeletesGroupRecordAndClearsReferences(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*WHERE\s+g\.id\s*=\s*\$1\s+RETURNING\s+g\.provider_id,\s*g\.upstream_group_key`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"provider_id", "upstream_group_key"}).AddRow(int64(3), "174"))
	mock.ExpectExec(`(?is)UPDATE\s+supplier_provider_accounts\s+AS\s+a.*SET\s+group_key\s*=\s*'',\s*group_name\s*=\s*'',\s*updated_at\s*=\s*NOW\(\).*a\.provider_id\s*=\s*\$1.*a\.group_key\s*=\s*\$2`).
		WithArgs(int64(3), "174").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	require.NoError(t, repo.DeleteGroup(context.Background(), 7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryDeletesInactiveGroupRecordWithLocalMappingAndActiveAccount(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*WHERE\s+g\.id\s*=\s*\$1\s+RETURNING\s+g\.provider_id,\s*g\.upstream_group_key`).
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"provider_id", "upstream_group_key"}).AddRow(int64(3), "175"))
	mock.ExpectExec(`(?is)UPDATE\s+supplier_provider_accounts\s+a.*SET\s+group_key\s*=\s*'',\s*group_name\s*=\s*''.*a\.provider_id\s*=\s*\$1.*a\.group_key\s*=\s*\$2`).
		WithArgs(int64(3), "175").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.DeleteGroup(context.Background(), 8))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryDeletesActiveOrGuardedGroupRecord(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*WHERE\s+g\.id\s*=\s*\$1\s+RETURNING\s+g\.provider_id,\s*g\.upstream_group_key`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"provider_id", "upstream_group_key"}).AddRow(int64(3), "174"))
	mock.ExpectExec(`(?is)UPDATE\s+supplier_provider_accounts\s+a.*SET\s+group_key\s*=\s*'',\s*group_name\s*=\s*''.*a\.provider_id\s*=\s*\$1.*a\.group_key\s*=\s*\$2`).
		WithArgs(int64(3), "174").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.DeleteGroup(context.Background(), 7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryClearsAllAccountGroupReferencesWhenDeletingGroup(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*WHERE\s+g\.id\s*=\s*\$1\s+RETURNING\s+g\.provider_id,\s*g\.upstream_group_key`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"provider_id", "upstream_group_key"}).AddRow(int64(3), "174"))
	mock.ExpectExec(`(?is)UPDATE\s+supplier_provider_accounts\s+a.*SET\s+group_key\s*=\s*'',\s*group_name\s*=\s*''.*a\.provider_id\s*=\s*\$1.*a\.group_key\s*=\s*\$2`).
		WithArgs(int64(3), "174").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	require.NoError(t, repo.DeleteGroup(context.Background(), 7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryDeletesSupplierAccountRecord(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectExec(`(?is)DELETE\s+FROM\s+supplier_provider_accounts\s+a\s+WHERE\s+a\.id\s*=\s*\$1`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.DeleteAccount(context.Background(), 9))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryRejectsUnsafeStaleSupplierAccountDelete(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectExec(`(?is)DELETE\s+FROM\s+supplier_provider_accounts\s+a.*a\.id\s*=\s*\$1`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteAccount(context.Background(), 9)
	require.ErrorIs(t, err, service.ErrSupplierProviderAccountDeleteConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}
