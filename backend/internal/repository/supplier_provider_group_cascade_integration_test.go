//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupRepository_DeleteCascade_ClearsSupplierGuardMapping(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	entClient := tx.Client()

	targetGroup, err := entClient.Group.Create().
		SetName(uniqueTestValue(t, "delete-supplier-guard-target")).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	var providerID int64
	rows, err := tx.QueryContext(ctx, `
INSERT INTO supplier_providers (code, name, provider_type, base_url)
VALUES ($1, $2, 'sub2api', 'https://example.invalid')
RETURNING id`, fmt.Sprintf("guard-%d", targetGroup.ID), uniqueTestValue(t, "supplier-name"))
	require.NoError(t, err)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&providerID))
	require.NoError(t, rows.Close())

	var supplierGroupID int64
	rows, err = tx.QueryContext(ctx, `
INSERT INTO supplier_provider_groups (
  provider_id, upstream_group_key, name, local_group_id,
  rate_guard_selected, rate_guard_selection_mode
) VALUES ($1, 'default', 'Default', $2, TRUE, 'auto')
RETURNING id`, providerID, targetGroup.ID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&supplierGroupID))
	require.NoError(t, rows.Close())

	groupRepo := newGroupRepositoryWithSQL(entClient, tx)
	_, err = groupRepo.DeleteCascade(ctx, targetGroup.ID)
	require.NoError(t, err)

	var localGroupID *int64
	var guardSelected bool
	var guardMode string
	rows, err = tx.QueryContext(ctx, `
SELECT local_group_id, rate_guard_selected, rate_guard_selection_mode
FROM supplier_provider_groups
WHERE id = $1`, supplierGroupID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&localGroupID, &guardSelected, &guardMode))
	require.NoError(t, rows.Close())
	require.Nil(t, localGroupID)
	require.False(t, guardSelected)
	require.Empty(t, guardMode)
}
