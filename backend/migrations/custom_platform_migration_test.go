package migrations

import (
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
