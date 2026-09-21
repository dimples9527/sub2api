package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// 分组监控快照的采集 SQL：每个分组只取「当前开启调度的账号」最近一条采样。
// 只认 schedulable 是为了让「最新一刻」反映的是真正在接客的那条账号；
// 分组与账号都要求 active 且未删除，口径与分组择优调度读取成员时保持一致。
const supplierGroupMonitorSnapshotMonitorSQL = `
SELECT DISTINCT ON (account_group.group_id)
       account_group.group_id,
       binding.local_account_id,
       sample.checked_at,
       sample.status,
       sample.latency_ms
FROM supplier_provider_monitor_samples sample
JOIN supplier_provider_monitor_bindings binding
  ON binding.monitor_target_id = sample.monitor_target_id
 AND binding.match_status = 'active'
JOIN account_groups account_group
  ON account_group.account_id = binding.local_account_id
JOIN groups local_group
  ON local_group.id = account_group.group_id
 AND local_group.deleted_at IS NULL
 AND local_group.status = $1
JOIN accounts local_account
  ON local_account.id = binding.local_account_id
 AND local_account.deleted_at IS NULL
 AND local_account.status = $1
 AND COALESCE(local_account.schedulable, FALSE) = TRUE
ORDER BY account_group.group_id, sample.checked_at DESC, sample.id DESC`

const supplierGroupMonitorSnapshotHealthGuardSQL = `
SELECT DISTINCT ON (account_group.group_id)
       account_group.group_id,
       history.local_account_id,
       history.checked_at,
       history.status,
       COALESCE(history.latency_ms, 0) AS latency_ms
FROM supplier_account_health_history history
JOIN account_groups account_group
  ON account_group.account_id = history.local_account_id
JOIN groups local_group
  ON local_group.id = account_group.group_id
 AND local_group.deleted_at IS NULL
 AND local_group.status = $1
JOIN accounts local_account
  ON local_account.id = history.local_account_id
 AND local_account.deleted_at IS NULL
 AND local_account.status = $1
 AND COALESCE(local_account.schedulable, FALSE) = TRUE
ORDER BY account_group.group_id, history.checked_at DESC, history.id DESC`

// recordGroupMonitorSnapshots 采集指定来源的分组快照并落库。
// 采集与写入放在一起，是为了让调用方（任务出口）只关心"记一次"，
// 不必自己拼来源 SQL，也避免两条链路写出不同口径的行。
func recordGroupMonitorSnapshots(ctx context.Context, db *sql.DB, source string) error {
	if db == nil {
		return fmt.Errorf("分组监控快照仓储未初始化")
	}
	snapshots, err := captureGroupMonitorSnapshots(ctx, db, source)
	if err != nil {
		return err
	}
	return upsertGroupMonitorSnapshots(ctx, db, snapshots)
}

func captureGroupMonitorSnapshots(ctx context.Context, db *sql.DB, source string) ([]service.SupplierGroupMonitorSnapshot, error) {
	if db == nil {
		return nil, fmt.Errorf("分组监控快照仓储未初始化")
	}
	var query string
	switch source {
	case service.SupplierGroupMonitorSnapshotSourceMonitor:
		query = supplierGroupMonitorSnapshotMonitorSQL
	case service.SupplierGroupMonitorSnapshotSourceHealthGuard:
		query = supplierGroupMonitorSnapshotHealthGuardSQL
	default:
		return nil, fmt.Errorf("未知的分组监控快照来源: %s", source)
	}
	rows, err := db.QueryContext(ctx, query, service.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("采集分组监控快照失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	snapshots := make([]service.SupplierGroupMonitorSnapshot, 0)
	for rows.Next() {
		var snapshot service.SupplierGroupMonitorSnapshot
		if err := rows.Scan(
			&snapshot.GroupID,
			&snapshot.LocalAccountID,
			&snapshot.CheckedAt,
			&snapshot.Status,
			&snapshot.LatencyMS,
		); err != nil {
			return nil, fmt.Errorf("读取分组监控快照采集结果失败: %w", err)
		}
		// 归一化必须在落库前做：samples 表里可能还留着归一化之前写入的旧状态值（如 operational），
		// 直接写会撞上快照表的 status 约束，导致整批快照写不进去。
		snapshot.Status = normalizeSupplierProviderMonitorStatus(snapshot.Status)
		snapshot.Source = source
		snapshot.Availability = service.SupplierGroupMonitorSnapshotAvailability(snapshot.Status)
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分组监控快照采集结果失败: %w", err)
	}
	return snapshots, nil
}

func upsertGroupMonitorSnapshots(ctx context.Context, db *sql.DB, snapshots []service.SupplierGroupMonitorSnapshot) error {
	if db == nil {
		return fmt.Errorf("分组监控快照仓储未初始化")
	}
	if len(snapshots) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启分组监控快照写入事务失败: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	for _, snapshot := range snapshots {
		if snapshot.GroupID <= 0 || snapshot.LocalAccountID <= 0 || snapshot.CheckedAt.IsZero() {
			continue
		}
		availability := snapshot.Availability
		if availability <= 0 {
			availability = service.SupplierGroupMonitorSnapshotAvailability(snapshot.Status)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_group_monitor_snapshots (
	group_id, local_account_id, source, status, latency_ms, availability, checked_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (group_id, source, checked_at) DO UPDATE SET
	local_account_id = EXCLUDED.local_account_id,
	status = EXCLUDED.status,
	latency_ms = EXCLUDED.latency_ms,
	availability = EXCLUDED.availability`,
			snapshot.GroupID,
			snapshot.LocalAccountID,
			snapshot.Source,
			snapshot.Status,
			snapshot.LatencyMS,
			availability,
			snapshot.CheckedAt.UTC(),
		); err != nil {
			return fmt.Errorf("写入分组监控快照失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交分组监控快照失败: %w", err)
	}
	committed = true
	return nil
}

func listLatestGroupMonitorSnapshots(ctx context.Context, db *sql.DB, groupIDs []int64) ([]service.SupplierGroupMonitorSnapshot, error) {
	if db == nil {
		return nil, fmt.Errorf("分组监控快照仓储未初始化")
	}
	if len(groupIDs) == 0 {
		return nil, nil
	}
	// 只认「快照那条账号现在仍然开启调度」的快照：账号被守护或择优调度关掉之后，
	// 它最后一条采样就成了历史，再拿它当最新一刻会让页面停在旧绿灯上。
	// 这时该分组查不到快照，读侧回退到全组聚合，由聚合把红/黄如实显示出来。
	// 过滤在 DISTINCT ON 之前生效，所以被关掉的是最新一条时，会退回到更早但仍然有效的那条。
	rows, err := db.QueryContext(ctx, `
SELECT DISTINCT ON (snapshot.group_id)
       snapshot.group_id, snapshot.local_account_id, snapshot.source, snapshot.status,
       snapshot.latency_ms, snapshot.availability, snapshot.checked_at
FROM supplier_group_monitor_snapshots snapshot
JOIN accounts local_account
  ON local_account.id = snapshot.local_account_id
 AND local_account.deleted_at IS NULL
 AND local_account.status = $2
 AND COALESCE(local_account.schedulable, FALSE) = TRUE
WHERE snapshot.group_id = ANY($1)
ORDER BY snapshot.group_id, snapshot.checked_at DESC, snapshot.id DESC`, pq.Array(groupIDs), service.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("读取分组监控快照失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	snapshots := make([]service.SupplierGroupMonitorSnapshot, 0)
	for rows.Next() {
		var snapshot service.SupplierGroupMonitorSnapshot
		if err := rows.Scan(
			&snapshot.GroupID,
			&snapshot.LocalAccountID,
			&snapshot.Source,
			&snapshot.Status,
			&snapshot.LatencyMS,
			&snapshot.Availability,
			&snapshot.CheckedAt,
		); err != nil {
			return nil, fmt.Errorf("读取分组监控快照行失败: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分组监控快照失败: %w", err)
	}
	return snapshots, nil
}

func (r *supplierProviderDataRepository) RecordGroupMonitorSnapshots(ctx context.Context, source string) error {
	if r == nil {
		return fmt.Errorf("分组监控快照仓储未初始化")
	}
	return recordGroupMonitorSnapshots(ctx, r.db, source)
}

func (r *supplierAccountHealthGuardRepository) RecordGroupMonitorSnapshots(ctx context.Context, source string) error {
	if r == nil {
		return fmt.Errorf("分组监控快照仓储未初始化")
	}
	return recordGroupMonitorSnapshots(ctx, r.db, source)
}
