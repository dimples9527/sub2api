package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderAuthStatsRebuildMigrationSeparatesLoginAndCache(t *testing.T) {
	content, err := FS.ReadFile("201_supplier_provider_auth_stats_rebuild.sql")
	require.NoError(t, err)
	sqlText := string(content)
	require.Contains(t, sqlText, "event_type IN ('login_success', 'login_failed')")
	require.Contains(t, sqlText, "event_type = 'cache_hit'")
	require.Contains(t, sqlText, "event_type = 'cache_miss'")
	require.Contains(t, sqlText, "auth_login_count = event_summary.login_count")
	require.Contains(t, sqlText, "auth_cache_hit_count = event_summary.cache_hit_count")
	require.NotContains(t, sqlText, "auth_login_count = stats.auth_login_count + cache_hit_summary.cache_hit_count")
}
