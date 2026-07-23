package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestAccountRateGuardRepositoryRemovesOnlyRequestedGroupsAndKeepsConcurrentBindings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewAccountRateGuardRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT schedulable FROM accounts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE")).
		WithArgs(int64(21)).WillReturnRows(sqlmock.NewRows([]string{"schedulable"}).AddRow(true))
	mock.ExpectQuery("DELETE FROM account_groups").
		WithArgs(int64(21), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id"}).AddRow(int64(31)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT group_id FROM account_groups WHERE account_id=$1 ORDER BY group_id")).
		WithArgs(int64(21)).WillReturnRows(sqlmock.NewRows([]string{"group_id"}).AddRow(int64(32)).AddRow(int64(33)).AddRow(int64(34)))
	mock.ExpectExec("INSERT INTO scheduler_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.RemoveAccountGroupsForRateGuard(context.Background(), 21, []int64{31})

	require.NoError(t, err)
	require.Equal(t, []int64{31}, result.RemovedGroupIDs)
	require.Equal(t, []int64{32, 33, 34}, result.RemainingGroupIDs)
	require.True(t, result.SchedulableAfter)
	require.False(t, result.SchedulableChanged)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRateGuardRepositoryDisablesSchedulingAfterLastGroupRemoved(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewAccountRateGuardRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT schedulable FROM accounts").WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"schedulable"}).AddRow(true))
	mock.ExpectQuery("DELETE FROM account_groups").WithArgs(int64(21), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id"}).AddRow(int64(31)))
	mock.ExpectQuery("SELECT group_id FROM account_groups").WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id"}))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE accounts SET schedulable=FALSE, updated_at=NOW() WHERE id=$1")).
		WithArgs(int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO scheduler_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.RemoveAccountGroupsForRateGuard(context.Background(), 21, []int64{31})

	require.NoError(t, err)
	require.False(t, result.SchedulableAfter)
	require.True(t, result.SchedulableChanged)
	require.Empty(t, result.RemainingGroupIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}
