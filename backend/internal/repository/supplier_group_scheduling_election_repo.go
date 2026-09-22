package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierGroupSchedulingElectionRepository struct {
	db *sql.DB
}

func NewSupplierGroupSchedulingElectionRepository(db *sql.DB) service.SupplierGroupSchedulingElectionRepository {
	return &supplierGroupSchedulingElectionRepository{db: db}
}

// ListGroupSchedulingElectionMembers 返回所有活跃本地分组下的活跃账号成员行。
// 择优调度只读现有测试状态与连续成功计数（不重测），因此这里连同 extra 一并取出，
// 在 Go 里解析 last_test_status / supplier_health_guard_healthy_count / last_tested_at。
func (r *supplierGroupSchedulingElectionRepository) ListGroupSchedulingElectionMembers(ctx context.Context, latencyWindowMinutes int) ([]service.SupplierGroupSchedulingElectionMember, error) {
	if latencyWindowMinutes <= 0 {
		latencyWindowMinutes = service.DefaultSupplierGroupSchedulingElectionLatencyWindowMinutes
	}
	// LATERAL 聚合最近 $2 分钟的健康历史：延迟平均只取成功样本（healthy/slow 且 latency>0，
	// 剔除失败/超时），但总样本数含 failed —— 失败要靠成功率在评分侧重新计入代价。
	// 已有 idx_supplier_account_health_history_account_checked（account+checked_at）索引，聚合很便宜。
	rows, err := r.db.QueryContext(ctx, `
SELECT g.id AS group_id,
       COALESCE(g.name, '') AS group_name,
       a.id AS account_id,
       COALESCE(a.name, '') AS account_name,
       COALESCE(a.platform, '') AS platform,
       COALESCE(a.type, '') AS account_type,
       COALESCE(a.schedulable, FALSE) AS schedulable,
       COALESCE(a.extra, '{}'::jsonb)::text AS extra,
       COALESCE(a.credentials->'model_mapping', 'null'::jsonb)::text AS model_mapping,
       COALESCE(lat.avg_latency_ms, 0) AS avg_latency_ms,
       COALESCE(lat.success_count, 0) AS latency_success_count,
       COALESCE(lat.total_count, 0) AS latency_total_count
FROM account_groups ag
JOIN groups g ON g.id = ag.group_id AND g.deleted_at IS NULL AND g.status = $1
JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL AND a.status = $1
LEFT JOIN LATERAL (
    SELECT AVG(h.latency_ms) FILTER (
             WHERE h.status IN ('healthy', 'slow') AND h.latency_ms > 0
           )::BIGINT AS avg_latency_ms,
           COUNT(*) FILTER (
             WHERE h.status IN ('healthy', 'slow') AND h.latency_ms > 0
           ) AS success_count,
           COUNT(*) AS total_count
    FROM supplier_account_health_history h
    WHERE h.local_account_id = a.id
      AND h.checked_at >= now() - make_interval(mins => $2)
) lat ON TRUE
ORDER BY g.id ASC, a.id ASC`, service.StatusActive, latencyWindowMinutes)
	if err != nil {
		return nil, fmt.Errorf("查询分组择优调度成员失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	members := make([]service.SupplierGroupSchedulingElectionMember, 0)
	for rows.Next() {
		var member service.SupplierGroupSchedulingElectionMember
		var extraRaw string
		var modelMappingRaw string
		if err := rows.Scan(
			&member.GroupID,
			&member.GroupName,
			&member.AccountID,
			&member.AccountName,
			&member.Platform,
			&member.AccountType,
			&member.Schedulable,
			&extraRaw,
			&modelMappingRaw,
			&member.AvgLatencyMs,
			&member.LatencySuccessCount,
			&member.LatencyTotalCount,
		); err != nil {
			return nil, fmt.Errorf("扫描分组择优调度成员失败: %w", err)
		}
		if extraRaw != "" && extraRaw != "null" {
			extra := map[string]any{}
			if err := json.Unmarshal([]byte(extraRaw), &extra); err != nil {
				return nil, fmt.Errorf("解析分组择优调度成员扩展字段失败: %w", err)
			}
			member.LastTestStatus = strings.TrimSpace(supplierGroupElectionExtraString(extra, "last_test_status"))
			member.HealthyCount = parseSupplierGroupElectionInt(extra["supplier_health_guard_healthy_count"])
			member.LastTestedAt = parseSupplierGroupElectionTime(supplierGroupElectionExtraString(extra, "last_tested_at"))
			// 用时只在测试成功时写入（失败不覆盖），与 last_test_status 同源，故口径一致。
			member.LastTestLatencyMs = int64(parseSupplierGroupElectionInt(extra["last_test_latency_ms"]))
			// 连续失败轮次由择优调度任务自己累计回写，是本任务「连续失败达阈值才关」的依据。
			member.FailedCount = parseSupplierGroupElectionInt(extra["supplier_group_election_failed_count"])
			// Extra 整体带给必需模型覆盖用：IsModelSupported 会读 openai_passthrough 等开关。
			member.Extra = extra
		}
		// 必需模型覆盖判定用的 model_mapping 子对象（不含 token）；空/缺失 = 空映射 = 支持所有模型。
		if modelMappingRaw != "" && modelMappingRaw != "null" {
			mapping := map[string]any{}
			if err := json.Unmarshal([]byte(modelMappingRaw), &mapping); err != nil {
				return nil, fmt.Errorf("解析分组择优调度成员模型映射失败: %w", err)
			}
			if len(mapping) > 0 {
				member.ModelMapping = mapping
			}
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分组择优调度成员失败: %w", err)
	}
	return members, nil
}

func supplierGroupElectionExtraString(extra map[string]any, key string) string {
	if extra == nil {
		return ""
	}
	if value, ok := extra[key].(string); ok {
		return value
	}
	return ""
}

func parseSupplierGroupElectionInt(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed)
		}
	}
	return 0
}

func parseSupplierGroupElectionTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
