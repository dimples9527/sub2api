package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierAutomationRunsFinishedIndexMigration(t *testing.T) {
	content, err := FS.ReadFile("222_supplier_automation_runs_finished_index_notx.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_automation_runs_task_finished")
	require.Contains(t, sql, "ON supplier_automation_runs (task_code, finished_at DESC, id DESC)")
	require.Contains(t, sql, "WHERE finished_at IS NOT NULL")
}
