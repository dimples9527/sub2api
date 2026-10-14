package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModelAuditMigration 钉住降智检测的两张表结构。
//
// 其中 NotContains("api_key") 是一条安全守卫：用户提交的上游凭据只在当次
// 调用中使用，任何一版迁移都不得把它写进表结构。
func TestModelAuditMigration(t *testing.T) {
	content, err := FS.ReadFile("249_model_audit.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS model_audit_runs")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS model_audit_items")
	require.Contains(t, sql, "REFERENCES model_audit_runs (id) ON DELETE CASCADE")
	require.Contains(t, sql, "idx_model_audit_runs_user_created")
	require.Contains(t, sql, "idx_model_audit_items_run")
	require.Contains(t, sql, "idx_model_audit_items_source_hash")

	require.NotContains(t, sql, "api_key")
}
