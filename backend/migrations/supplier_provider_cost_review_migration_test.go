package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderCostReviewMigrationDefinesCurrentAndHistoryTables(t *testing.T) {
	content, err := FS.ReadFile("234_supplier_provider_cost_reviews.sql")
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.NotEqual(t, []byte{0xEF, 0xBB, 0xBF}, content[:3], "迁移文件不能包含 UTF-8 BOM")

	sqlText := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_provider_cost_reviews")
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_provider_cost_review_histories")
	require.Contains(t, sqlText, "UNIQUE (provider_id, stat_date)")
	require.Contains(t, sqlText, "NUMERIC(20, 6)")
	require.Contains(t, sqlText, "pending_review")
	require.Contains(t, sqlText, "changed_after_approval")
	require.Contains(t, sqlText, "decision_type")
	require.Contains(t, sqlText, "event_type")
	require.Contains(t, sqlText, "ON DELETE SET NULL")
}

func TestSupplierProviderCostReviewLocalCostMigrationAddsColumns(t *testing.T) {
	content, err := FS.ReadFile("241_supplier_provider_cost_review_local_cost.sql")
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.NotEqual(t, []byte{0xEF, 0xBB, 0xBF}, content[:3], "迁移文件不能包含 UTF-8 BOM")

	sqlText := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sqlText, "ALTER TABLE supplier_provider_cost_reviews ADD COLUMN IF NOT EXISTS local_cost NUMERIC(20, 6) NULL")
	require.Contains(t, sqlText, "ALTER TABLE supplier_provider_cost_review_histories ADD COLUMN IF NOT EXISTS local_cost NUMERIC(20, 6) NULL")
}
