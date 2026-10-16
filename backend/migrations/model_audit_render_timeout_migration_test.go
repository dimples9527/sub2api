package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelAuditRenderTimeoutMigration 钉住 253 给检测记录加的生成题超时列。
//
// 用 ADD COLUMN IF NOT EXISTS + NOT NULL DEFAULT 900 而不是新建表：迁移要求幂等，
// 且 249~252 已经落库，只能增量改。默认值必须等于代码里的默认超时
// （modelDegradationRenderTaskTimeout = 15 分钟），否则历史行会被解释成别的上限。
func TestModelAuditRenderTimeoutMigration(t *testing.T) {
	content, err := FS.ReadFile("253_model_audit_render_timeout.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ALTER TABLE model_audit_runs")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS render_timeout_seconds")
	require.Contains(t, sql, "NOT NULL DEFAULT 900")

	// 该列仍然不得成为存放凭据的地方（沿用 249 的安全口径）。
	require.NotContains(t, sql, "api_key")
}
