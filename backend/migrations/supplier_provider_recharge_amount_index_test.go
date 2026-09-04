package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderRechargeAmountCoverIndexMigration(t *testing.T) {
	content, err := FS.ReadFile("243_supplier_provider_recharges_amount_cover_notx.sql")
	require.NoError(t, err)

	sqlText := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sqlText, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_provider_recharges_provider_occurred_amount")
	require.Contains(t, sqlText, "ON supplier_provider_recharges (provider_id, occurred_at DESC, id DESC) INCLUDE (amount)")
	require.Contains(t, sqlText, "DROP INDEX CONCURRENTLY IF EXISTS idx_supplier_provider_recharges_provider_occurred;")
}
