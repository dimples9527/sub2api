package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsageLogLatencyPhasesMigration(t *testing.T) {
	content, err := FS.ReadFile("239_usage_log_latency_phases.sql")
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.NotEqual(t, []byte{0xEF, 0xBB, 0xBF}, content[:3], "迁移文件不能包含 UTF-8 BOM")

	sqlText := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sqlText, "CREATE TABLE IF NOT EXISTS usage_log_latency_phases")
	require.Contains(t, sqlText, "request_id VARCHAR(64) NOT NULL")
	require.Contains(t, sqlText, "api_key_id BIGINT NOT NULL DEFAULT 0")
	for _, column := range []string{"build_ms", "slot_wait_ms", "connect_ms", "tls_ms", "first_byte_ms"} {
		require.Contains(t, sqlText, column+" INTEGER NULL CHECK ("+column+" >= 0)")
	}
	require.Contains(t, sqlText, "conn_reused BOOLEAN NULL")
	require.Contains(t, sqlText, "CREATE INDEX IF NOT EXISTS idx_usage_log_latency_phases_request ON usage_log_latency_phases (request_id, api_key_id, id DESC)")
	require.Contains(t, sqlText, "CREATE INDEX IF NOT EXISTS idx_usage_log_latency_phases_created ON usage_log_latency_phases (created_at)")

	// 侧边表刻意不加 UNIQUE：同一 request_id 可能对应多条 usage 记录。
	require.NotContains(t, sqlText, "request_id VARCHAR(64) NOT NULL UNIQUE")
	// 刻意不加外键：写入走 best-effort 旁路，不应因 usage_logs 尚未落库而失败。
	require.NotContains(t, sqlText, "REFERENCES usage_logs")
}
