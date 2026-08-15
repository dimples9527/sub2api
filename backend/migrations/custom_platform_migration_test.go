package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomPlatformMigrationCreatesDictionaryAndSeedsDefaultPlatforms(t *testing.T) {
	content, err := FS.ReadFile("205_custom_platforms.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS custom_platforms")
	require.Contains(t, sqlText, "idx_custom_platforms_code_active")
	require.Contains(t, sqlText, "'glm', 'GLM'")
	require.Contains(t, sqlText, "'deepseek', 'DeepSeek'")
	require.Contains(t, sqlText, "'kimi', 'Kimi'")
}

func TestCustomPlatformColorMigrationAddsColorColumnAndSeedBrandColors(t *testing.T) {
	content, err := FS.ReadFile("225_custom_platforms_color.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "ADD COLUMN IF NOT EXISTS color VARCHAR(16) NOT NULL DEFAULT '#64748b'")
	require.Contains(t, sqlText, "color = '#2563eb'")
	require.Contains(t, sqlText, "color = '#4f46e5'")
	require.Contains(t, sqlText, "color = '#db2777'")
	require.False(t, strings.HasPrefix(sqlText, "\ufeff"), "迁移 SQL 不应包含 UTF-8 BOM")
}
