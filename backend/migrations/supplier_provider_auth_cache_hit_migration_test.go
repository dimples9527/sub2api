package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderAuthCacheHitMigrationBackfillsLoginSummary(t *testing.T) {
	content, err := FS.ReadFile("200_supplier_provider_auth_cache_hit.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "UPDATE supplier_provider_runtime_stats")
	require.Contains(t, sqlText, "supplier_provider_auth_events")
	require.Contains(t, sqlText, "event_type = 'cache_hit'")
	require.Contains(t, sqlText, "auth_login_count")
	require.Contains(t, sqlText, "auth_login_success_count")
	require.Contains(t, sqlText, "auth_last_login_at")
	require.Contains(t, sqlText, "auth_last_login_status")
}
