package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierGroupRateGuardMigrationDefinesGuardStateAndTask(t *testing.T) {
	content, err := FS.ReadFile("184_supplier_group_rate_guard.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "rate_guard_selected BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sqlText, "rate_guard_selection_mode VARCHAR(16) NOT NULL DEFAULT ''")
	require.Contains(t, sqlText, "rate_guard_last_snapshot_at TIMESTAMPTZ NULL")
	require.Contains(t, sqlText, "rate_guard_last_checked_at TIMESTAMPTZ NULL")
	require.Contains(t, sqlText, "CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_group_rate_guard")
	require.Contains(t, sqlText, "WHERE rate_guard_selected = TRUE")
	require.Contains(t, sqlText, "group_sync_status VARCHAR(32) NOT NULL DEFAULT 'never'")
	require.Contains(t, sqlText, "group_sync_message TEXT NOT NULL DEFAULT ''")
	require.Contains(t, sqlText, "last_group_sync_at TIMESTAMPTZ NULL")
	require.Contains(t, sqlText, "'supplier_rate_guard'")
	require.Contains(t, sqlText, "FALSE")
	require.Contains(t, sqlText, "'2-59/5 * * * *'")
	require.Contains(t, sqlText, `"rate_guard_safety_multiplier": 1.1`)
	require.Contains(t, sqlText, `"rate_guard_max_snapshot_age_seconds": 1800`)
}
