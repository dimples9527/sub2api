package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierRateGuardChangeLogsMigrationCreatesPendingTodoTable(t *testing.T) {
	content, err := FS.ReadFile("187_supplier_rate_guard_change_logs.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_rate_guard_change_logs")
	require.Contains(t, sqlText, "status VARCHAR(16) NOT NULL DEFAULT 'pending'")
	require.Contains(t, sqlText, "handled_at TIMESTAMPTZ NULL")
	require.Contains(t, sqlText, "changed_at TIMESTAMPTZ NOT NULL")
	require.Contains(t, sqlText, "idx_supplier_rate_guard_change_logs_status_changed_at")
}
