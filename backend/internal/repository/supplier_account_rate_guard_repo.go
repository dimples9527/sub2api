package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type supplierAccountRateGuardRepository struct {
	db *sql.DB
}

func NewSupplierAccountRateGuardRepository(db *sql.DB) service.SupplierAccountRateGuardRepository {
	return &supplierAccountRateGuardRepository{db: db}
}

func (r *supplierAccountRateGuardRepository) ListAccountRateGuardCandidates(ctx context.Context, providerID int64, upstreamKeys []string) ([]service.SupplierAccountRateGuardCandidate, error) {
	keys := make([]string, 0, len(upstreamKeys))
	seen := make(map[string]struct{}, len(upstreamKeys))
	for _, rawKey := range upstreamKeys {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if providerID <= 0 || len(keys) == 0 {
		return []service.SupplierAccountRateGuardCandidate{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT a.id AS provider_account_id, a.provider_id, p.name AS provider_name,
       a.upstream_account_key, a.name AS upstream_account_name,
       a.rate_multiplier AS raw_rate, p.account_rate_multiplier_scale AS rate_scale,
       local_match.match_count, matched_account.id AS local_account_id,
       COALESCE(matched_account.name, '') AS local_account_name,
       COALESCE(reverse_match.match_count, 0) AS reverse_match_count,
       COALESCE(matched_account.schedulable, FALSE) AS schedulable,
       local_group.id AS local_group_id,
       COALESCE(local_group.name, '') AS local_group_name,
       local_group.rate_multiplier AS local_group_rate
FROM supplier_provider_accounts a
JOIN supplier_providers p ON p.id = a.provider_id
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS match_count, MIN(local_account.id) AS local_account_id
  FROM accounts local_account
  WHERE local_account.deleted_at IS NULL
    AND regexp_replace(lower(local_account.name), '[^[:alnum:]]', '', 'g')
        = regexp_replace(lower(p.account_name_prefix || a.name), '[^[:alnum:]]', '', 'g')
) local_match ON TRUE
LEFT JOIN accounts matched_account
  ON matched_account.id = local_match.local_account_id
 AND local_match.match_count = 1
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS match_count
  FROM supplier_provider_accounts reverse_account
  JOIN supplier_providers reverse_provider ON reverse_provider.id = reverse_account.provider_id
  WHERE reverse_account.active = TRUE
    AND matched_account.id IS NOT NULL
    AND regexp_replace(lower(reverse_provider.account_name_prefix || reverse_account.name), '[^[:alnum:]]', '', 'g')
        = regexp_replace(lower(matched_account.name), '[^[:alnum:]]', '', 'g')
) reverse_match ON TRUE
LEFT JOIN account_groups account_group ON account_group.account_id = matched_account.id
LEFT JOIN groups local_group ON local_group.id = account_group.group_id AND local_group.deleted_at IS NULL
WHERE a.provider_id=$1 AND a.active=TRUE AND p.enabled=TRUE
  AND a.upstream_account_key = ANY($2)
ORDER BY a.id, local_group.id`, providerID, pq.Array(keys))
	if err != nil {
		return nil, fmt.Errorf("查询供应商账号倍率守护候选失败: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierAccountRateGuardCandidate, 0)
	indexByID := make(map[int64]int)
	for rows.Next() {
		var candidate service.SupplierAccountRateGuardCandidate
		var localAccountID sql.NullInt64
		var localGroupID sql.NullInt64
		var localGroupName string
		var localGroupRate sql.NullFloat64
		if err := rows.Scan(
			&candidate.ProviderAccountID, &candidate.ProviderID, &candidate.ProviderName,
			&candidate.UpstreamAccountKey, &candidate.UpstreamAccountName,
			&candidate.RawRate, &candidate.RateScale, &candidate.MatchCount,
			&localAccountID, &candidate.LocalAccountName, &candidate.ReverseMatchCount,
			&candidate.Schedulable, &localGroupID, &localGroupName, &localGroupRate,
		); err != nil {
			return nil, fmt.Errorf("扫描供应商账号倍率守护候选失败: %w", err)
		}
		if localAccountID.Valid {
			candidate.LocalAccountID = localAccountID.Int64
		}
		switch {
		case candidate.MatchCount == 0:
			candidate.MatchStatus = service.SupplierAccountRateGuardMatchUnmatched
		case candidate.MatchCount == 1 && candidate.ReverseMatchCount <= 1:
			candidate.MatchStatus = service.SupplierAccountRateGuardMatchMatched
		default:
			candidate.MatchStatus = service.SupplierAccountRateGuardMatchConflict
		}
		index, exists := indexByID[candidate.ProviderAccountID]
		if !exists {
			candidate.Groups = make([]service.SupplierAccountRateGuardGroup, 0)
			index = len(items)
			indexByID[candidate.ProviderAccountID] = index
			items = append(items, candidate)
		}
		if localGroupID.Valid && localGroupRate.Valid {
			items[index].Groups = append(items[index].Groups, service.SupplierAccountRateGuardGroup{
				ID: localGroupID.Int64, Name: localGroupName, RateMultiplier: localGroupRate.Float64,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商账号倍率守护候选失败: %w", err)
	}
	return items, nil
}

func (r *supplierAccountRateGuardRepository) CreateAccountRateGuardUnbindLogs(ctx context.Context, logs []service.SupplierAccountRateGuardUnbindLog) error {
	if len(logs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始保存倍率守护解绑日志失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range logs {
		createdAt := item.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO supplier_account_rate_guard_unbind_logs (
  run_id, provider_id, provider_name, supplier_provider_account_id,
  upstream_account_key, upstream_account_name, local_account_id, local_account_name,
  local_group_id, local_group_name, raw_upstream_rate, rate_scale,
  effective_upstream_rate, local_group_rate, mode, result,
  before_bound, after_bound, before_schedulable, after_schedulable,
  error_message, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
			item.RunID, item.ProviderID, item.ProviderName, nullablePositiveInt64(item.ProviderAccountID),
			item.UpstreamAccountKey, item.UpstreamAccountName, nullablePositiveInt64(item.LocalAccountID), item.LocalAccountName,
			nullablePositiveInt64(item.LocalGroupID), item.LocalGroupName, item.RawUpstreamRate, item.RateScale,
			item.EffectiveUpstreamRate, item.LocalGroupRate, item.Mode, item.Result,
			item.BeforeBound, item.AfterBound, item.BeforeSchedulable, item.AfterSchedulable,
			item.ErrorMessage, createdAt)
		if err != nil {
			return fmt.Errorf("保存倍率守护解绑日志失败: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交倍率守护解绑日志失败: %w", err)
	}
	return nil
}

func (r *supplierAccountRateGuardRepository) ListAccountRateGuardUnbindLogs(ctx context.Context, params service.SupplierAccountRateGuardUnbindLogListParams) (service.SupplierAccountRateGuardUnbindLogListResult, error) {
	params = normalizeSupplierAccountRateGuardLogParams(params)
	where, args := supplierAccountRateGuardLogWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_account_rate_guard_unbind_logs WHERE "+where, args...).Scan(&total); err != nil {
		return service.SupplierAccountRateGuardUnbindLogListResult{}, fmt.Errorf("统计倍率守护解绑日志失败: %w", err)
	}
	queryArgs := append(append([]any{}, args...), params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT id, run_id, provider_id, provider_name, supplier_provider_account_id,
       upstream_account_key, upstream_account_name, local_account_id, local_account_name,
       local_group_id, local_group_name, raw_upstream_rate, rate_scale,
       effective_upstream_rate, local_group_rate, mode, result,
       before_bound, after_bound, before_schedulable, after_schedulable,
       error_message, created_at
FROM supplier_account_rate_guard_unbind_logs
WHERE `+where+fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), queryArgs...)
	if err != nil {
		return service.SupplierAccountRateGuardUnbindLogListResult{}, fmt.Errorf("查询倍率守护解绑日志失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierAccountRateGuardUnbindLog, 0)
	for rows.Next() {
		var item service.SupplierAccountRateGuardUnbindLog
		var providerAccountID, localAccountID, localGroupID sql.NullInt64
		var beforeSchedulable, afterSchedulable sql.NullBool
		if err := rows.Scan(
			&item.ID, &item.RunID, &item.ProviderID, &item.ProviderName, &providerAccountID,
			&item.UpstreamAccountKey, &item.UpstreamAccountName, &localAccountID, &item.LocalAccountName,
			&localGroupID, &item.LocalGroupName, &item.RawUpstreamRate, &item.RateScale,
			&item.EffectiveUpstreamRate, &item.LocalGroupRate, &item.Mode, &item.Result,
			&item.BeforeBound, &item.AfterBound, &beforeSchedulable, &afterSchedulable,
			&item.ErrorMessage, &item.CreatedAt,
		); err != nil {
			return service.SupplierAccountRateGuardUnbindLogListResult{}, fmt.Errorf("扫描倍率守护解绑日志失败: %w", err)
		}
		if providerAccountID.Valid {
			item.ProviderAccountID = providerAccountID.Int64
		}
		if localAccountID.Valid {
			item.LocalAccountID = localAccountID.Int64
		}
		if localGroupID.Valid {
			item.LocalGroupID = localGroupID.Int64
		}
		if beforeSchedulable.Valid {
			value := beforeSchedulable.Bool
			item.BeforeSchedulable = &value
		}
		if afterSchedulable.Valid {
			value := afterSchedulable.Bool
			item.AfterSchedulable = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierAccountRateGuardUnbindLogListResult{}, fmt.Errorf("遍历倍率守护解绑日志失败: %w", err)
	}
	return service.SupplierAccountRateGuardUnbindLogListResult{Items: items, Total: total, Page: params.Page, PageSize: params.PageSize}, nil
}

func normalizeSupplierAccountRateGuardLogParams(params service.SupplierAccountRateGuardUnbindLogListParams) service.SupplierAccountRateGuardUnbindLogListParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 200 {
		params.PageSize = 200
	}
	params.Search = strings.TrimSpace(params.Search)
	params.Result = strings.TrimSpace(params.Result)
	return params
}

func supplierAccountRateGuardLogWhere(params service.SupplierAccountRateGuardUnbindLogListParams) (string, []any) {
	conditions := []string{"1=1"}
	args := make([]any, 0, 5)
	if params.RunID > 0 {
		args = append(args, params.RunID)
		conditions = append(conditions, fmt.Sprintf("run_id = $%d", len(args)))
	}
	if params.ProviderID > 0 {
		args = append(args, params.ProviderID)
		conditions = append(conditions, fmt.Sprintf("provider_id = $%d", len(args)))
	}
	if params.LocalAccountID > 0 {
		args = append(args, params.LocalAccountID)
		conditions = append(conditions, fmt.Sprintf("local_account_id = $%d", len(args)))
	}
	if params.Result != "" {
		args = append(args, params.Result)
		conditions = append(conditions, fmt.Sprintf("result = $%d", len(args)))
	}
	if params.Search != "" {
		args = append(args, "%"+params.Search+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, "(provider_name ILIKE "+placeholder+" OR upstream_account_key ILIKE "+placeholder+" OR upstream_account_name ILIKE "+placeholder+" OR local_account_name ILIKE "+placeholder+" OR local_group_name ILIKE "+placeholder+")")
	}
	return strings.Join(conditions, " AND "), args
}

func nullablePositiveInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}
