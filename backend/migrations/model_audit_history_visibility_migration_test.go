package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelAuditHistoryVisibilityMigration 钉住 254 给检测历史可见性配置建的两张表。
//
// 两张表刻意不建外键：分组会被改名或删除，历史记录（连同隐藏配置）要保留当时的语义。
// 主机表带 host <> '' 约束 —— 空主机名没有可匹配的记录，落库即配置失效，不如直接拦下。
func TestModelAuditHistoryVisibilityMigration(t *testing.T) {
	content, err := FS.ReadFile("254_model_audit_history_visibility.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS model_audit_history_hidden_groups")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS model_audit_history_hidden_hosts")
	require.Contains(t, sql, "group_id BIGINT PRIMARY KEY")
	require.Contains(t, sql, "host TEXT PRIMARY KEY CHECK (host <> '')")

	// 不建外键：删除 / 改名分组后，隐藏配置仍要生效（原因见迁移注释）。
	require.NotContains(t, sql, "FOREIGN KEY")
	require.NotContains(t, sql, "REFERENCES")
	// 该表仍然不得成为存放凭据的地方（沿用 249 的安全口径）。
	require.NotContains(t, sql, "api_key")
}
