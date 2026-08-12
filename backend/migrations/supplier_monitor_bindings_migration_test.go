package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierMonitorBindingsMigrationDefinesTargetsBindingsAndSamples(t *testing.T) {
	content, err := FS.ReadFile("221_supplier_monitor_bindings.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_provider_monitor_targets")
	require.Contains(t, sqlText, "UNIQUE(provider_id, monitor_key)")
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_provider_monitor_bindings")
	require.Contains(t, sqlText, "local_account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE")
	require.Contains(t, sqlText, "UNIQUE(provider_id, monitor_target_id)")
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_provider_monitor_samples")
	require.Contains(t, sqlText, "UNIQUE(monitor_target_id, checked_at)")
}
