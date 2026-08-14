package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type monitorGroupPlatformOverrideRepository struct {
	db *sql.DB
}

// NewMonitorGroupPlatformOverrideRepository 创建模型监控专用覆盖配置仓储。
func NewMonitorGroupPlatformOverrideRepository(db *sql.DB) service.MonitorGroupPlatformOverrideRepository {
	return &monitorGroupPlatformOverrideRepository{db: db}
}

func (r *monitorGroupPlatformOverrideRepository) ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]service.MonitorGroupPlatformOverride, error) {
	result := make(map[int64]service.MonitorGroupPlatformOverride)
	if len(groupIDs) == 0 {
		return result, nil
	}
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("monitor group platform override repository is not initialized")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT group_id, COALESCE(actual_platform, ''), COALESCE(show_in_monitor, TRUE)
FROM monitor_group_platform_overrides
WHERE group_id = ANY($1)`, pq.Array(groupIDs))
	if err != nil {
		return nil, fmt.Errorf("list monitor group platform overrides: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var groupID int64
		var platform string
		var showInMonitor bool
		if err := rows.Scan(&groupID, &platform, &showInMonitor); err != nil {
			return nil, fmt.Errorf("scan monitor group platform override: %w", err)
		}
		result[groupID] = service.MonitorGroupPlatformOverride{ActualPlatform: platform, ShowInMonitor: showInMonitor}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitor group platform overrides: %w", err)
	}
	return result, nil
}

func (r *monitorGroupPlatformOverrideRepository) Set(ctx context.Context, groupID int64, platform string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("monitor group platform override repository is not initialized")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO monitor_group_platform_overrides (group_id, actual_platform)
VALUES ($1, $2)
ON CONFLICT (group_id) DO UPDATE
SET actual_platform = EXCLUDED.actual_platform, updated_at = NOW()`, groupID, platform)
	if err != nil {
		return fmt.Errorf("set monitor group platform override: %w", err)
	}
	return nil
}

func (r *monitorGroupPlatformOverrideRepository) SetShowInMonitor(ctx context.Context, groupID int64, show bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("monitor group platform override repository is not initialized")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO monitor_group_platform_overrides (group_id, actual_platform, show_in_monitor)
VALUES ($1, '', $2)
ON CONFLICT (group_id) DO UPDATE
SET show_in_monitor = EXCLUDED.show_in_monitor, updated_at = NOW()`, groupID, show)
	if err != nil {
		return fmt.Errorf("set monitor group visibility: %w", err)
	}
	return nil
}

func (r *monitorGroupPlatformOverrideRepository) Clear(ctx context.Context, groupID int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("monitor group platform override repository is not initialized")
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE monitor_group_platform_overrides
SET actual_platform = '', updated_at = NOW()
WHERE group_id = $1`, groupID)
	if err != nil {
		return fmt.Errorf("clear monitor group platform override: %w", err)
	}
	return nil
}
