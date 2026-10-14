package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelAuditProtocolMigration 钉住 251 给检测记录加的协议列。
//
// 用 ADD COLUMN IF NOT EXISTS + NOT NULL DEFAULT 'chat_completions' 而不是新建表：
// 迁移要求幂等，且 249/250 已经落库，只能增量改。
// 默认值必须落在既有协议上，否则历史记录会被解释成另一种协议。
func TestModelAuditProtocolMigration(t *testing.T) {
	content, err := FS.ReadFile("251_model_audit_protocol.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ALTER TABLE model_audit_runs")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS protocol")
	require.Contains(t, sql, "NOT NULL DEFAULT 'chat_completions'")

	// 该列仍然不得成为存放凭据的地方（沿用 249 的安全口径）。
	require.NotContains(t, sql, "api_key")
}
