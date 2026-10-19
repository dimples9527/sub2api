package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 检测历史可见性配置的持久化：管理端维护「隐藏哪些分组 / 上游主机」，
// 普通用户读取历史时按它过滤。表结构见迁移 254。

// modelAuditHistoryHostOfR 从 base_url 里提取小写主机名（端口不参与匹配）。
//
// split_part 链：先取 scheme 后的第一段（无 scheme 的怪值整串当第一段），
// 再截到第一个 '/'，最后截到第一个 ':' 去掉端口。
// 口径与前端 groupBucketOf 的主机名桶、service.normalizeHistoryHost 三者保持一致：
// 管理员配置里存的小写主机名要能命中这里查出来的值。
const modelAuditHistoryHostOfR = `lower(split_part(split_part(split_part(r.base_url, '://', 2), '/', 1), ':', 1))`

// modelAuditHistoryVisibilityFilter 只保留「对普通用户可见」的记录：
//   - 分组桶：命中的分组在隐藏清单里 → 不下发；
//   - 主机桶：只有没命中分组的记录才按主机归桶（group_id IS NULL），
//     所以主机隐藏只作用于这类记录 —— 命中分组的记录仍归在它的分组下，
//     不会被主机配置误伤（同一个主机可能是平台自身地址，同时也是第三方上游地址）。
const modelAuditHistoryVisibilityFilter = `
WHERE NOT EXISTS (
    SELECT 1 FROM model_audit_history_hidden_groups hg
    WHERE hg.group_id = r.group_id
)
AND NOT (
    r.group_id IS NULL
    AND EXISTS (
        SELECT 1 FROM model_audit_history_hidden_hosts hh
        WHERE hh.host = ` + modelAuditHistoryHostOfR + `
    )
)`

// ListHiddenHistoryTargets 读取隐藏配置：分组 ID 与上游主机两张清单。
func (r *modelDegradationRepository) ListHiddenHistoryTargets(ctx context.Context) ([]int64, []string, error) {
	if r == nil || r.db == nil {
		return nil, nil, fmt.Errorf("model audit repository database is not initialized")
	}

	groupIDs := make([]int64, 0, 8)
	rows, err := r.db.QueryContext(ctx,
		`SELECT group_id FROM model_audit_history_hidden_groups ORDER BY group_id`)
	if err != nil {
		return nil, nil, fmt.Errorf("list hidden model audit history groups: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var groupID int64
		if err := rows.Scan(&groupID); err != nil {
			return nil, nil, fmt.Errorf("scan hidden model audit history group: %w", err)
		}
		groupIDs = append(groupIDs, groupID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate hidden model audit history groups: %w", err)
	}

	hosts := make([]string, 0, 8)
	hostRows, err := r.db.QueryContext(ctx,
		`SELECT host FROM model_audit_history_hidden_hosts ORDER BY host`)
	if err != nil {
		return nil, nil, fmt.Errorf("list hidden model audit history hosts: %w", err)
	}
	defer func() { _ = hostRows.Close() }()
	for hostRows.Next() {
		var host string
		if err := hostRows.Scan(&host); err != nil {
			return nil, nil, fmt.Errorf("scan hidden model audit history host: %w", err)
		}
		hosts = append(hosts, host)
	}
	if err := hostRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate hidden model audit history hosts: %w", err)
	}

	return groupIDs, hosts, nil
}

// SetHiddenHistoryGroup 追加 / 移除一个隐藏分组，幂等：
// 重复隐藏不会报错，取消未隐藏的分组也不会报错（管理端并发点击不产生副作用）。
func (r *modelDegradationRepository) SetHiddenHistoryGroup(ctx context.Context, groupID int64, hidden bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}

	if hidden {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO model_audit_history_hidden_groups (group_id) VALUES ($1)
ON CONFLICT (group_id) DO NOTHING`,
			groupID,
		); err != nil {
			return fmt.Errorf("hide model audit history group %d: %w", groupID, err)
		}
		return nil
	}
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM model_audit_history_hidden_groups WHERE group_id = $1`, groupID,
	); err != nil {
		return fmt.Errorf("unhide model audit history group %d: %w", groupID, err)
	}
	return nil
}

// SetHiddenHistoryHost 追加 / 移除一个隐藏主机，幂等。host 由 service 归一化后传入。
func (r *modelDegradationRepository) SetHiddenHistoryHost(ctx context.Context, host string, hidden bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("model audit repository database is not initialized")
	}

	if hidden {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO model_audit_history_hidden_hosts (host) VALUES ($1)
ON CONFLICT (host) DO NOTHING`,
			host,
		); err != nil {
			return fmt.Errorf("hide model audit history host %s: %w", host, err)
		}
		return nil
	}
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM model_audit_history_hidden_hosts WHERE host = $1`, host,
	); err != nil {
		return fmt.Errorf("unhide model audit history host %s: %w", host, err)
	}
	return nil
}

// ListHistoryHostStats 按主机统计没命中分组的记录数（历史墙的「上游主机」桶清单），
// 供管理端挑选要隐藏的上游。limit 由上层收敛，这里再做一次兜底。
func (r *modelDegradationRepository) ListHistoryHostStats(ctx context.Context, limit int) ([]service.ModelDegradationHistoryHostStat, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model audit repository database is not initialized")
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT `+modelAuditHistoryHostOfR+` AS host, COUNT(*) AS run_count
FROM model_audit_runs r
WHERE r.group_id IS NULL AND r.base_url <> ''
GROUP BY 1
HAVING `+modelAuditHistoryHostOfR+` <> ''
ORDER BY run_count DESC, host ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list model audit history host stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	stats := make([]service.ModelDegradationHistoryHostStat, 0, 16)
	for rows.Next() {
		var stat service.ModelDegradationHistoryHostStat
		if err := rows.Scan(&stat.Host, &stat.RunCount); err != nil {
			return nil, fmt.Errorf("scan model audit history host stat: %w", err)
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model audit history host stats: %w", err)
	}
	return stats, nil
}
