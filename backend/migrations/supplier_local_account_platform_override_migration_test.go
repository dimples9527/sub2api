package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierLocalAccountPlatformOverrideMigrationDefinesDedicatedOverrideTable(t *testing.T) {
	content, err := FS.ReadFile("195_supplier_local_account_platform_override.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_local_account_platform_overrides")
	require.Contains(t, sqlText, "local_account_id BIGINT NOT NULL")
	require.Contains(t, sqlText, "platform VARCHAR(50) NOT NULL")
	require.Contains(t, sqlText, "UNIQUE (local_account_id)")
	require.Contains(t, sqlText, "REFERENCES accounts(id) ON DELETE CASCADE")
	require.Contains(t, sqlText, "idx_supplier_local_account_platform_overrides_platform")
}
