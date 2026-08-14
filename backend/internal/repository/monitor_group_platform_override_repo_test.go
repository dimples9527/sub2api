package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestMonitorGroupPlatformOverrideRepositorySetListAndClear(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewMonitorGroupPlatformOverrideRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO monitor_group_platform_overrides (group_id, actual_platform)
VALUES ($1, $2)
ON CONFLICT (group_id) DO UPDATE
SET actual_platform = EXCLUDED.actual_platform, updated_at = NOW()`)).
		WithArgs(int64(7), service.PlatformOpenAI).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Set(ctx, 7, service.PlatformOpenAI))

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT group_id, COALESCE(actual_platform, ''), COALESCE(show_in_monitor, TRUE)
FROM monitor_group_platform_overrides
WHERE group_id = ANY($1)`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "actual_platform", "show_in_monitor"}).
			AddRow(int64(7), service.PlatformOpenAI, true).
			AddRow(int64(9), service.PlatformGemini, false))

	loaded, err := repo.ListByGroupIDs(ctx, []int64{7, 9})
	require.NoError(t, err)
	require.Equal(t, map[int64]service.MonitorGroupPlatformOverride{
		7: {ActualPlatform: service.PlatformOpenAI, ShowInMonitor: true},
		9: {ActualPlatform: service.PlatformGemini, ShowInMonitor: false},
	}, loaded)

	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO monitor_group_platform_overrides (group_id, actual_platform, show_in_monitor)
VALUES ($1, '', $2)
ON CONFLICT (group_id) DO UPDATE
SET show_in_monitor = EXCLUDED.show_in_monitor, updated_at = NOW()`)).
		WithArgs(int64(9), false).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.SetShowInMonitor(ctx, 9, false))

	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE monitor_group_platform_overrides
SET actual_platform = '', updated_at = NOW()
WHERE group_id = $1`)).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Clear(ctx, 7))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMonitorGroupPlatformOverrideRepositoryListByGroupIDsEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewMonitorGroupPlatformOverrideRepository(db)
	loaded, err := repo.ListByGroupIDs(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, loaded)
	require.NoError(t, mock.ExpectationsWereMet())
}
