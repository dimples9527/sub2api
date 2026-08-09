package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierMonitorSyncMigrationDefinesMonitorURLAndTask(t *testing.T) {
	content, err := FS.ReadFile("206_supplier_monitor_sync.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "ADD COLUMN IF NOT EXISTS monitor_url TEXT NOT NULL DEFAULT ''")
	require.Contains(t, sqlText, "UPDATE supplier_provider_types")
	require.Contains(t, sqlText, "WHERE code = 'sub2api'")
	require.Contains(t, sqlText, "UPDATE supplier_providers")
	require.Contains(t, sqlText, "WHERE provider_type = 'sub2api'")
	require.Contains(t, sqlText, "'/api/v1/channel-monitors?timezone=Asia%2FShanghai'")
	require.Contains(t, sqlText, "'supplier_monitor_sync'")
	require.Contains(t, sqlText, "\u4f9b\u5e94\u5546\u76d1\u63a7\u6570\u636e\u540c\u6b65")
	require.Contains(t, sqlText, "'@every 30s'")
}
