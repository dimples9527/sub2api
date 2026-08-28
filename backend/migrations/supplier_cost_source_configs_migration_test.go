package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierCostSourceConfigsMigrationDefinesGlobalSourceAndOverrides(t *testing.T) {
	content, err := FS.ReadFile("237_supplier_cost_source_configs.sql")
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.NotEqual(t, []byte{0xEF, 0xBB, 0xBF}, content[:3], "迁移文件不能包含 UTF-8 BOM")

	sqlText := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sqlText, "ADD COLUMN IF NOT EXISTS cost_source VARCHAR(16) NOT NULL DEFAULT 'auto'")
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_cost_source_configs")
	require.Contains(t, sqlText, "provider_id BIGINT NOT NULL UNIQUE REFERENCES supplier_providers(id) ON DELETE CASCADE")
	require.Contains(t, sqlText, "threshold NUMERIC(20, 6) NULL")
	require.Contains(t, sqlText, "'auto', 'upstream', 'calculated'")
}
