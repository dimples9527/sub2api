package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierMonitorTargetInactiveMigrationAddsInactiveAt(t *testing.T) {
	content, err := FS.ReadFile("242_supplier_monitor_target_inactive.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "ALTER TABLE supplier_provider_monitor_targets")
	require.Contains(t, sqlText, "ADD COLUMN IF NOT EXISTS inactive_at TIMESTAMPTZ")
	require.Contains(t, sqlText, "idx_supplier_monitor_targets_inactive")
	require.Contains(t, sqlText, "WHERE active = FALSE")
}
