package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelAuditReasoningEffortMigration 钉住 250 给检测记录加的推理程度列。
//
// 用 ADD COLUMN IF NOT EXISTS + NOT NULL DEFAULT '' 而不是新建表：
// 迁移要求幂等，且 249 已经落库，只能增量改。
func TestModelAuditReasoningEffortMigration(t *testing.T) {
	content, err := FS.ReadFile("250_model_audit_reasoning_effort.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ALTER TABLE model_audit_runs")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS reasoning_effort")
	require.Contains(t, sql, "NOT NULL DEFAULT ''")

	// 该列仍然不得成为存放凭据的地方（沿用 249 的安全口径）。
	require.NotContains(t, sql, "api_key")
}
