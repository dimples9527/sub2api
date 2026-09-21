package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newSupplierGroupMonitorSnapshotMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock, func() { _ = db.Close() }
}

func TestCaptureGroupMonitorSnapshotsOnlyReadsSchedulingAccount(t *testing.T) {
	db, mock, closeDB := newSupplierGroupMonitorSnapshotMock(t)
	defer closeDB()
	checkedAt := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	// SQL 里必须带 schedulable 过滤：只取当前开启调度的那条账号，而不是分组里所有账号。
	mock.ExpectQuery(`(?s)SELECT DISTINCT ON \(account_group\.group_id\).*FROM supplier_provider_monitor_samples.*COALESCE\(local_account\.schedulable, FALSE\) = TRUE.*`).
		WithArgs(service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "local_account_id", "checked_at", "status", "latency_ms"}).
			AddRow(int64(5), int64(55), checkedAt, "operational", int64(120)))

	got, err := captureGroupMonitorSnapshots(context.Background(), db, service.SupplierGroupMonitorSnapshotSourceMonitor)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(5), got[0].GroupID)
	require.Equal(t, int64(55), got[0].LocalAccountID)
	// 历史样本里可能留着归一化之前的状态值，落库前必须归一，否则会撞上快照表的 status 约束。
	require.Equal(t, service.SupplierAccountHealthGuardStatusHealthy, got[0].Status)
	require.Equal(t, 100.0, got[0].Availability)
	require.Equal(t, int64(120), got[0].LatencyMS)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCaptureGroupMonitorSnapshotsRejectsUnknownSource(t *testing.T) {
	db, _, closeDB := newSupplierGroupMonitorSnapshotMock(t)
	defer closeDB()

	_, err := captureGroupMonitorSnapshots(context.Background(), db, "unknown_source")

	require.Error(t, err)
	require.Contains(t, err.Error(), "未知的分组监控快照来源")
}

func TestUpsertGroupMonitorSnapshotsSkipsIncompleteRows(t *testing.T) {
	db, mock, closeDB := newSupplierGroupMonitorSnapshotMock(t)
	defer closeDB()
	checkedAt := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)INSERT INTO supplier_group_monitor_snapshots.*ON CONFLICT \(group_id, source, checked_at\) DO UPDATE.*`).
		WithArgs(int64(5), int64(55), service.SupplierGroupMonitorSnapshotSourceMonitor, "healthy", int64(120), 100.0, checkedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := upsertGroupMonitorSnapshots(context.Background(), db, []service.SupplierGroupMonitorSnapshot{
		{GroupID: 5, LocalAccountID: 55, Source: service.SupplierGroupMonitorSnapshotSourceMonitor, Status: service.SupplierAccountHealthGuardStatusHealthy, LatencyMS: 120, Availability: 100, CheckedAt: checkedAt},
		// 缺账号 / 缺时间的行不能写进去，否则脏行会把「最新一刻」带偏。
		{GroupID: 6, Source: service.SupplierGroupMonitorSnapshotSourceMonitor, Status: service.SupplierAccountHealthGuardStatusFailed},
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListLatestGroupMonitorSnapshotsPicksNewestPerGroup(t *testing.T) {
	db, mock, closeDB := newSupplierGroupMonitorSnapshotMock(t)
	defer closeDB()
	older := time.Date(2026, 9, 21, 11, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	// 读侧必须带 schedulable 过滤：账号被关掉调度之后，它的最后一条采样不能再当「最新一刻」。
	mock.ExpectQuery(`(?s)SELECT DISTINCT ON \(snapshot\.group_id\).*FROM supplier_group_monitor_snapshots.*COALESCE\(local_account\.schedulable, FALSE\) = TRUE.*ORDER BY snapshot\.group_id, snapshot\.checked_at DESC, snapshot\.id DESC`).
		WithArgs("{5,6}", service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "local_account_id", "source", "status", "latency_ms", "availability", "checked_at"}).
			AddRow(int64(5), int64(55), service.SupplierGroupMonitorSnapshotSourceMonitor, "healthy", int64(120), 100.0, older).
			AddRow(int64(6), int64(66), service.SupplierGroupMonitorSnapshotSourceHealthGuard, "failed", int64(0), 0.0, newer))

	got, err := listLatestGroupMonitorSnapshots(context.Background(), db, []int64{5, 6})

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, int64(5), got[0].GroupID)
	require.Equal(t, int64(6), got[1].GroupID)
	require.Equal(t, older, got[0].CheckedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}
