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
func (r *supplierGroupSchedulingElectionRepository) ListGroupSchedulingElectionMembers(ctx context.Context) ([]service.SupplierGroupSchedulingElectionMember, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT g.id AS group_id,
       COALESCE(g.name, '') AS group_name,
       a.id AS account_id,
       COALESCE(a.name, '') AS account_name,
       COALESCE(a.platform, '') AS platform,
       COALESCE(a.schedulable, FALSE) AS schedulable,
       COALESCE(a.extra, '{}'::jsonb)::text AS extra
FROM account_groups ag
JOIN groups g ON g.id = ag.group_id AND g.deleted_at IS NULL AND g.status = $1
JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL AND a.status = $1
ORDER BY g.id ASC, a.id ASC`, service.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("查询分组择优调度成员失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	members := make([]service.SupplierGroupSchedulingElectionMember, 0)
	for rows.Next() {
		var member service.SupplierGroupSchedulingElectionMember
		var extraRaw string
		if err := rows.Scan(
			&member.GroupID,
			&member.GroupName,
			&member.AccountID,
			&member.AccountName,
			&member.Platform,
			&member.Schedulable,
			&extraRaw,
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
