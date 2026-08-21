package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderRechargeSyncMigrationDefinesDefaultTaskWithoutOverwritingSchedule(t *testing.T) {
	content, err := FS.ReadFile("233_supplier_provider_recharge_sync.sql")
	require.NoError(t, err)

	sqlText := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sqlText, "'supplier_provider_recharge_sync'")
	require.Contains(t, sqlText, "'供应商充值记录同步'")
	require.Contains(t, sqlText, "'@every 30m'")
	require.Contains(t, sqlText, "enabled = supplier_automation_tasks.enabled")
	require.Contains(t, sqlText, "ELSE supplier_automation_tasks.cron_expression")
	require.Contains(t, sqlText, "WHEN supplier_automation_tasks.timeout_seconds <= 0")
}
