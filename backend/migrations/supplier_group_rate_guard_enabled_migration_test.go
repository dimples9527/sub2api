package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierGroupRateGuardEnabledMigrationAddsFlagAndMigratesIgnoreData(t *testing.T) {
	content, err := FS.ReadFile("226_supplier_group_rate_guard_enabled.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "rate_guard_enabled BOOLEAN NOT NULL DEFAULT TRUE")
	require.Contains(t, sqlText, "UPDATE supplier_provider_groups")
	require.Contains(t, sqlText, "SET rate_guard_enabled = FALSE")
	require.Contains(t, sqlText, "WHERE rate_guard_ignored = TRUE")
}
