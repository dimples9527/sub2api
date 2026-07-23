package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type accountRateGuardRepository struct {
	db *sql.DB
}

func NewAccountRateGuardRepository(db *sql.DB) service.AccountRateGuardGroupRemover {
	return &accountRateGuardRepository{db: db}
}

func (r *accountRateGuardRepository) RemoveAccountGroupsForRateGuard(ctx context.Context, accountID int64, groupIDs []int64) (service.AccountRateGuardGroupRemovalResult, error) {
	result := service.AccountRateGuardGroupRemovalResult{}
	ids := normalizePositiveInt64s(groupIDs)
	if accountID <= 0 || len(ids) == 0 {
		return result, service.ErrSupplierProviderInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("开始账号倍率守护解绑事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := tx.QueryRowContext(ctx, "SELECT schedulable FROM accounts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", accountID).Scan(&result.SchedulableBefore); err != nil {
		if err == sql.ErrNoRows {
			return result, service.ErrAccountNotFound
		}
		return result, fmt.Errorf("锁定倍率守护账号失败: %w", err)
	}
	result.SchedulableAfter = result.SchedulableBefore

	rows, err := tx.QueryContext(ctx, `
DELETE FROM account_groups
WHERE account_id=$1 AND group_id = ANY($2)
RETURNING group_id`, accountID, pq.Array(ids))
	if err != nil {
		return result, fmt.Errorf("解除账号倍率守护分组失败: %w", err)
	}
	for rows.Next() {
		var groupID int64
		if err := rows.Scan(&groupID); err != nil {
			_ = rows.Close()
			return result, fmt.Errorf("读取已解绑分组失败: %w", err)
		}
		result.RemovedGroupIDs = append(result.RemovedGroupIDs, groupID)
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("遍历已解绑分组失败: %w", err)
	}
	sort.Slice(result.RemovedGroupIDs, func(i, j int) bool { return result.RemovedGroupIDs[i] < result.RemovedGroupIDs[j] })

	remainingRows, err := tx.QueryContext(ctx, "SELECT group_id FROM account_groups WHERE account_id=$1 ORDER BY group_id", accountID)
	if err != nil {
		return result, fmt.Errorf("读取账号剩余分组失败: %w", err)
	}
	for remainingRows.Next() {
		var groupID int64
		if err := remainingRows.Scan(&groupID); err != nil {
			_ = remainingRows.Close()
			return result, fmt.Errorf("扫描账号剩余分组失败: %w", err)
		}
		result.RemainingGroupIDs = append(result.RemainingGroupIDs, groupID)
	}
	if err := remainingRows.Close(); err != nil {
		return result, err
	}
	if err := remainingRows.Err(); err != nil {
		return result, fmt.Errorf("遍历账号剩余分组失败: %w", err)
	}

	if len(result.RemovedGroupIDs) > 0 && len(result.RemainingGroupIDs) == 0 && result.SchedulableBefore {
		if _, err := tx.ExecContext(ctx, "UPDATE accounts SET schedulable=FALSE, updated_at=NOW() WHERE id=$1", accountID); err != nil {
			return result, fmt.Errorf("关闭无分组账号调度失败: %w", err)
		}
		result.SchedulableAfter = false
		result.SchedulableChanged = true
	}
	if len(result.RemovedGroupIDs) > 0 {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, buildSchedulerGroupPayload(result.RemovedGroupIDs)); err != nil {
			return result, fmt.Errorf("写入账号调度刷新事件失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("提交账号倍率守护解绑事务失败: %w", err)
	}
	return result, nil
}

func normalizePositiveInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
