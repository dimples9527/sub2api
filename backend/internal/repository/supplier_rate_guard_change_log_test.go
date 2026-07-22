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

func TestSupplierProviderDataRepositoryListsAndHandlesRateGuardChangeLogs(t *testing.T) {
	repo, mock := newSupplierProviderDataRepoMock(t)
	now := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_rate_guard_change_logs")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_rate_guard_change_logs WHERE status = $1")).
		WithArgs(service.SupplierRateGuardChangeLogStatusPending).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, mapping_id, local_group_id, local_group_name")).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "mapping_id", "local_group_id", "local_group_name", "upstream_group_key", "upstream_group_name",
			"old_rate", "new_rate", "status", "changed_at", "handled_at", "created_at",
		}).AddRow(int64(9), int64(10), int64(7), "本地 VIP", "vip", "上游 VIP", 2.5, 2.75, "pending", now, nil, now))

	result, err := repo.ListRateGuardChangeLogs(context.Background(), service.SupplierRateGuardChangeLogListParams{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.PendingCount)
	require.Len(t, result.Items, 1)
	require.Equal(t, "本地 VIP", result.Items[0].LocalGroupName)

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE supplier_rate_guard_change_logs SET status = $2, handled_at = COALESCE(handled_at, NOW()) WHERE id = $1 RETURNING")).
		WithArgs(int64(9), service.SupplierRateGuardChangeLogStatusHandled).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "mapping_id", "local_group_id", "local_group_name", "upstream_group_key", "upstream_group_name",
			"old_rate", "new_rate", "status", "changed_at", "handled_at", "created_at",
		}).AddRow(int64(9), int64(10), int64(7), "本地 VIP", "vip", "上游 VIP", 2.5, 2.75, "handled", now, now, now))

	item, err := repo.MarkRateGuardChangeLogHandled(context.Background(), 9)

	require.NoError(t, err)
	require.Equal(t, service.SupplierRateGuardChangeLogStatusHandled, item.Status)
	require.NotNil(t, item.HandledAt)
	require.NoError(t, mock.ExpectationsWereMet())
}
