package migrations

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierAccountRateGuardLogStatusMigrationAddsHandlingWorkflow(t *testing.T) {
	content, err := FS.ReadFile("189_supplier_account_rate_guard_log_status.sql")
	require.NoError(t, err)
	require.False(t, bytes.HasPrefix(content, []byte{0xEF, 0xBB, 0xBF}), "迁移 SQL 不应包含 UTF-8 BOM")

	sqlText := string(content)
	require.Contains(t, sqlText, "ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'handled'")
	require.Contains(t, sqlText, "ADD COLUMN IF NOT EXISTS handled_at TIMESTAMPTZ NULL")
	require.Contains(t, sqlText, "WHEN result = 'unbound' THEN 'pending'")
	require.Contains(t, sqlText, "WHEN result = 'unbound' THEN NULL")
	require.Contains(t, sqlText, "idx_supplier_account_rate_guard_logs_status_created")
}
