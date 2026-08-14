package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderNewAPIAuthModeMigration(t *testing.T) {
	content, err := FS.ReadFile("223_supplier_provider_newapi_auth_mode.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS newapi_auth_mode VARCHAR(32) NOT NULL DEFAULT 'auto'")
	require.Contains(t, sql, "CHECK (newapi_auth_mode IN ('auto', 'cookie_session', 'access_token_refresh'))")
}
