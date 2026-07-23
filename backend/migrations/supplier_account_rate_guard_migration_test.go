package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierAccountRateGuardMigrationDefinesRateSnapshotLogsAndTask(t *testing.T) {
	content, err := FS.ReadFile("188_supplier_account_rate_guard.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "rate_sync_status VARCHAR(32) NOT NULL DEFAULT 'never'")
	require.Contains(t, sqlText, "rate_sync_message TEXT NOT NULL DEFAULT ''")
	require.Contains(t, sqlText, "last_rate_sync_at TIMESTAMPTZ NULL")
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_account_rate_guard_unbind_logs")
	require.Contains(t, sqlText, "REFERENCES supplier_automation_runs(id) ON DELETE CASCADE")
	require.Contains(t, sqlText, "idx_supplier_account_rate_guard_logs_run")
	require.Contains(t, sqlText, "idx_supplier_account_rate_guard_logs_created")
	require.Contains(t, sqlText, "idx_supplier_account_rate_guard_logs_provider")
	require.Contains(t, sqlText, "idx_supplier_account_rate_guard_logs_local_account")
	require.Contains(t, sqlText, "idx_supplier_account_rate_guard_logs_result")
	require.Contains(t, sqlText, "'supplier_account_rate_guard'")
	require.Contains(t, sqlText, "'供应商账号倍率守护'")
	require.Contains(t, sqlText, "FALSE")
	require.Contains(t, sqlText, "'@every 300s'")
	require.Contains(t, sqlText, "600")
}
