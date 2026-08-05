package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderAuthAuditMigrationDefinesSummaryAndHistory(t *testing.T) {
	content, err := FS.ReadFile("199_supplier_provider_auth_audit.sql")
	require.NoError(t, err)

	sqlText := string(content)
	for _, column := range []string{
		"auth_login_count BIGINT NOT NULL DEFAULT 0",
		"auth_login_success_count BIGINT NOT NULL DEFAULT 0",
		"auth_login_failure_count BIGINT NOT NULL DEFAULT 0",
		"auth_cache_hit_count BIGINT NOT NULL DEFAULT 0",
		"auth_cache_miss_count BIGINT NOT NULL DEFAULT 0",
		"auth_last_login_at TIMESTAMPTZ NULL",
		"auth_last_login_status VARCHAR(32) NOT NULL DEFAULT ''",
		"auth_last_login_error TEXT NOT NULL DEFAULT ''",
		"auth_last_cache_hit_at TIMESTAMPTZ NULL",
		"auth_last_cache_error TEXT NOT NULL DEFAULT ''",
		"auth_last_token_expires_at TIMESTAMPTZ NULL",
		"auth_last_token_fingerprint VARCHAR(64) NOT NULL DEFAULT ''",
	} {
		require.Contains(t, sqlText, column)
	}

	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS supplier_provider_auth_events")
	require.Contains(t, sqlText, "provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE")
	for _, eventType := range []string{"cache_hit", "cache_miss", "login_success", "login_failed", "cache_invalidated", "cache_error"} {
		require.Contains(t, sqlText, "'"+eventType+"'")
	}
	require.Contains(t, sqlText, "idx_supplier_provider_auth_events_provider_time")
	require.Contains(t, sqlText, "idx_supplier_provider_auth_events_provider_type_time")
	require.Contains(t, sqlText, "idx_supplier_provider_auth_events_created_at")
}