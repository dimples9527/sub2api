package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltinPlatformMigrationReconcilesLegacyCustomPlatforms(t *testing.T) {
	content, err := FS.ReadFile("229_migrate_builtin_platforms.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "SET actual_platform = 'zhipu'")
	require.Contains(t, sql, "SET platform = 'zhipu'")
	require.Contains(t, sql, "LOWER(TRIM(actual_platform)) = 'glm'")
	require.Contains(t, sql, "LOWER(TRIM(platform)) = 'glm'")
	require.Contains(t, sql, "LOWER(TRIM(code)) IN ('glm', 'kimi', 'deepseek')")
	require.Contains(t, sql, "deleted_at = COALESCE(deleted_at, NOW())")
}
