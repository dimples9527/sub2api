package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderAuthRefreshEventsMigrationPreservesLoginStatistics(t *testing.T) {
	content, err := FS.ReadFile("202_supplier_provider_auth_refresh_events.sql")
	require.NoError(t, err)

	sqlText := string(content)
	for _, eventType := range []string{
		"cache_hit",
		"cache_miss",
		"login_success",
		"login_failed",
		"cache_invalidated",
		"cache_error",
		"refresh_success",
		"refresh_failed",
	} {
		require.Contains(t, sqlText, "'"+eventType+"'")
	}
	require.Contains(t, sqlText, "DROP CONSTRAINT IF EXISTS")
	require.Contains(t, sqlText, "ADD CONSTRAINT")
	require.NotContains(t, sqlText, "UPDATE supplier_provider_runtime_stats")
	require.NotContains(t, sqlText, "auth_login_count")
	require.NotContains(t, sqlText, "auth_login_success_count")
	require.NotContains(t, sqlText, "auth_login_failure_count")
}
