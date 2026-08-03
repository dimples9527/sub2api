package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderDataRepositoryDeletesOnlySafeInactiveGroupRecord(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectExec(`(?is)DELETE\s+FROM\s+supplier_provider_groups\s+g.*g\.active\s*=\s*FALSE.*g\.local_group_id\s+IS\s+NULL.*g\.rate_guard_selected\s*=\s*FALSE.*NOT\s+EXISTS\s*\(\s*SELECT\s+1\s+FROM\s+supplier_provider_accounts\s+a.*a\.provider_id\s*=\s*g\.provider_id.*a\.group_key\s*=\s*g\.upstream_group_key.*a\.active\s*=\s*TRUE`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.DeleteGroup(context.Background(), 7))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderDataRepositoryRejectsUnsafeInactiveGroupDelete(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM supplier_provider_groups")).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteGroup(context.Background(), 7)
	require.ErrorIs(t, err, service.ErrSupplierProviderGroupDeleteConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}
