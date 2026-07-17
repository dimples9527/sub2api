package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSupplierAutomationLockReleasesOnlyOwner(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	lock := NewSupplierAutomationLock(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	ctx := context.Background()

	acquired, err := lock.TryAcquireAutomationLock(ctx, "supplier_data_sync", "owner-a", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)

	require.NoError(t, lock.ReleaseAutomationLock(ctx, "supplier_data_sync", "owner-b"))
	acquired, err = lock.TryAcquireAutomationLock(ctx, "supplier_data_sync", "owner-b", time.Minute)
	require.NoError(t, err)
	require.False(t, acquired)

	require.NoError(t, lock.ReleaseAutomationLock(ctx, "supplier_data_sync", "owner-a"))
	acquired, err = lock.TryAcquireAutomationLock(ctx, "supplier_data_sync", "owner-b", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
}
