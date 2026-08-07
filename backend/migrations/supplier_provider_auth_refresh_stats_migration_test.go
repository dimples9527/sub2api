package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderAuthRefreshStatsMigrationAddsAndBackfillsIndependentCounters(t *testing.T) {
	content, err := FS.ReadFile("203_supplier_provider_auth_refresh_stats.sql")
	require.NoError(t, err)

	sqlText := string(content)
	for _, column := range []string{
		"auth_refresh_count BIGINT NOT NULL DEFAULT 0",
		"auth_refresh_success_count BIGINT NOT NULL DEFAULT 0",
		"auth_refresh_failure_count BIGINT NOT NULL DEFAULT 0",
	} {
		require.Contains(t, sqlText, column)
	}
	require.Contains(t, sqlText, "event_type IN ('refresh_success', 'refresh_failed')")
	require.Contains(t, sqlText, "event_type = 'refresh_success'")
	require.Contains(t, sqlText, "event_type = 'refresh_failed'")
	require.Contains(t, sqlText, "auth_refresh_count = event_summary.refresh_count")
	require.Contains(t, sqlText, "auth_refresh_success_count = event_summary.refresh_success_count")
	require.Contains(t, sqlText, "auth_refresh_failure_count = event_summary.refresh_failure_count")
	require.NotContains(t, sqlText, "auth_login_count = event_summary.refresh_count")
	require.NotContains(t, sqlText, "auth_login_success_count = event_summary.refresh_success_count")
	require.NotContains(t, sqlText, "auth_login_failure_count = event_summary.refresh_failure_count")
}
