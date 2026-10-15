package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelAuditGroupMigration 钉住 252 给检测记录加的分组列。
//
// 用 ADD COLUMN IF NOT EXISTS 而不是新建表：迁移要求幂等，且 249/250/251 已经落库，
// 只能增量改。两列都必须可空或有默认值 —— 存量记录没有分组，回填不了。
func TestModelAuditGroupMigration(t *testing.T) {
	content, err := FS.ReadFile("252_model_audit_group.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ALTER TABLE model_audit_runs")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS group_id BIGINT")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS group_name TEXT NOT NULL DEFAULT ''")

	// 该列仍然不得成为存放凭据的地方（沿用 249 的安全口径）：
	// 分组是拿明文 Key 反查出来的结果，Key 本身不能落库。
	require.NotContains(t, sql, "api_key")
}
