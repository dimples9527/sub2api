//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestMonitorGroupPlatformOverrideCascadeDelete(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()

	group, err := client.Group.Create().
		SetName(uniqueTestValue(t, "monitor-platform-cascade")).
		SetPlatform(service.PlatformAnthropic).
		SetRateMultiplier(1).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeStandard).
		Save(ctx)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
INSERT INTO monitor_group_platform_overrides (group_id, actual_platform)
VALUES ($1, $2)`, group.ID, service.PlatformOpenAI)
	require.NoError(t, err)

	_, err = client.Group.DeleteOneID(group.ID).Exec(ctx)
	require.NoError(t, err)

	var count int64
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM monitor_group_platform_overrides WHERE group_id = $1`, group.ID).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count, "override row should be removed when the group is deleted")
}
