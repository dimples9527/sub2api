package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type supplierAccountHealthGuardRepository struct {
	db *sql.DB
}

func NewSupplierAccountHealthGuardRepository(db *sql.DB) service.SupplierAccountHealthGuardRepository {
	return &supplierAccountHealthGuardRepository{db: db}
}

func (r *supplierAccountHealthGuardRepository) ListAccountHealthGuardCandidates(ctx context.Context) ([]service.SupplierAccountHealthGuardCandidate, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT a.id AS provider_account_id,
       a.provider_id,
       p.name AS provider_name,
       a.upstream_account_key,
       a.name AS upstream_account_name,
       a.rate_multiplier AS supplier_rate_multiplier,
       local_match.match_count,
       matched_account.id AS local_account_id,
       COALESCE(matched_account.name, '') AS local_account_name,
       COALESCE(matched_account.platform, '') AS local_account_platform,
       COALESCE(platform_override.platform, '') AS platform_override,
       COALESCE(
         NULLIF(platform_override.platform, ''),
         NULLIF(matched_account.platform, ''),
         ''
       ) AS effective_platform,
       COALESCE(matched_account.status, '') AS local_account_status,
       COALESCE(matched_account.schedulable, FALSE) AS local_account_schedulable,
       COALESCE(matched_account.extra, '{}'::jsonb) AS local_account_extra
FROM supplier_provider_accounts a
JOIN supplier_providers p ON p.id = a.provider_id
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS match_count,
         MIN(local_account.id) AS local_account_id
  FROM accounts local_account
  WHERE local_account.deleted_at IS NULL
    AND regexp_replace(lower(local_account.name), '[^[:alnum:]]', '', 'g')
        = regexp_replace(lower(p.account_name_prefix || a.name), '[^[:alnum:]]', '', 'g')
) local_match ON TRUE
LEFT JOIN accounts matched_account
  ON matched_account.id = local_match.local_account_id
 AND local_match.match_count = 1
LEFT JOIN supplier_local_account_platform_overrides platform_override
  ON platform_override.local_account_id = matched_account.id
WHERE a.active = TRUE
  AND p.enabled = TRUE
ORDER BY a.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("查询供应商账号健康守护候选失败: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierAccountHealthGuardCandidate, 0)
	for rows.Next() {
		var item service.SupplierAccountHealthGuardCandidate
		var localAccountID sql.NullInt64
		var localAccountName string
		var localAccountPlatform string
		var platformOverride string
		var effectivePlatform string
		var localAccountStatus string
		var localAccountSchedulable bool
		var localAccountExtra []byte
		if err := rows.Scan(
			&item.Source.ProviderAccountID,
			&item.Source.ProviderID,
			&item.Source.ProviderName,
			&item.Source.UpstreamAccountKey,
			&item.Source.UpstreamAccountName,
			&item.Source.RateMultiplier,
			&item.MatchCount,
			&localAccountID,
			&localAccountName,
			&localAccountPlatform,
			&platformOverride,
			&effectivePlatform,
			&localAccountStatus,
			&localAccountSchedulable,
			&localAccountExtra,
		); err != nil {
			return nil, fmt.Errorf("扫描供应商账号健康守护候选失败: %w", err)
		}
		switch item.MatchCount {
		case 0:
			item.MatchStatus = service.SupplierAccountHealthGuardMatchUnmatched
		case 1:
			item.MatchStatus = service.SupplierAccountHealthGuardMatchMatched
		default:
			item.MatchStatus = service.SupplierAccountHealthGuardMatchConflict
		}
		item.PlatformOverride = platformOverride
		item.EffectivePlatform = effectivePlatform
		if localAccountID.Valid && item.MatchCount == 1 {
			extra := map[string]any{}
			if len(localAccountExtra) > 0 {
				if err := json.Unmarshal(localAccountExtra, &extra); err != nil {
					return nil, fmt.Errorf("解析供应商账号健康守护本地账号扩展字段失败: %w", err)
				}
			}
			item.LocalAccountID = localAccountID.Int64
			item.LocalAccount = &service.Account{
				ID:          localAccountID.Int64,
				Name:        localAccountName,
				Platform:    localAccountPlatform,
				Status:      localAccountStatus,
				Schedulable: localAccountSchedulable,
				Extra:       extra,
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商账号健康守护候选失败: %w", err)
	}
	return items, nil
}

func (r *supplierAccountHealthGuardRepository) ListAccountHealthGuardUnavailableReasons(ctx context.Context, accountIDs []int64) (map[int64]service.SupplierAccountHealthGuardUnavailableCause, error) {
	causes := make(map[int64]service.SupplierAccountHealthGuardUnavailableCause, len(accountIDs))
	if len(accountIDs) == 0 {
		return causes, nil
	}
	// 与候选查询相反，这里**刻意不加** a.active / p.enabled 过滤：
	// 那些被挡掉的行正是要报出来的原因本身，过滤掉就又退化成「候选为空」一句了。
	// 判定顺序按「最具体 → 最笼统」，且名字匹配条件与候选查询逐字一致
	// （p.account_name_prefix || a.name 归一化后比较），否则诊断会和实际执行结果对不上。
	rows, err := r.db.QueryContext(ctx, `
WITH local_account AS (
  SELECT a.id,
         a.deleted_at,
         regexp_replace(lower(a.name), '[^[:alnum:]]', '', 'g') AS local_key
  FROM accounts a
  WHERE a.id = ANY($1)
), provider_account AS (
  SELECT pa.active AS provider_account_active,
         sp.enabled AS provider_enabled,
         regexp_replace(lower(COALESCE(sp.account_name_prefix, '') || pa.name), '[^[:alnum:]]', '', 'g') AS guard_key,
         (SELECT COUNT(*)
            FROM accounts x
           WHERE x.deleted_at IS NULL
             AND regexp_replace(lower(x.name), '[^[:alnum:]]', '', 'g')
               = regexp_replace(lower(COALESCE(sp.account_name_prefix, '') || pa.name), '[^[:alnum:]]', '', 'g')
         ) AS match_count
  FROM supplier_provider_accounts pa
  JOIN supplier_providers sp ON sp.id = pa.provider_id
)
SELECT local_account.id AS local_account_id,
       CASE
         WHEN local_account.deleted_at IS NOT NULL THEN 'local_missing'
         WHEN NOT EXISTS (
           SELECT 1 FROM provider_account
            WHERE provider_account.guard_key = local_account.local_key
         ) THEN 'no_provider_account'
         WHEN EXISTS (
           SELECT 1 FROM provider_account
            WHERE provider_account.guard_key = local_account.local_key
              AND provider_account.provider_account_active
              AND provider_account.provider_enabled
              AND provider_account.match_count = 1
         ) THEN ''
         WHEN EXISTS (
           SELECT 1 FROM provider_account
            WHERE provider_account.guard_key = local_account.local_key
              AND provider_account.provider_account_active
              AND provider_account.provider_enabled
         ) THEN 'match_conflict'
         WHEN EXISTS (
           SELECT 1 FROM provider_account
            WHERE provider_account.guard_key = local_account.local_key
              AND provider_account.provider_enabled
         ) THEN 'provider_account_inactive'
         ELSE 'provider_disabled'
       END AS cause
FROM local_account`, pq.Array(accountIDs))
	if err != nil {
		return nil, fmt.Errorf("查询供应商账号健康守护不可用原因失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var accountID int64
		var cause string
		if err := rows.Scan(&accountID, &cause); err != nil {
			return nil, fmt.Errorf("扫描供应商账号健康守护不可用原因失败: %w", err)
		}
		causes[accountID] = service.SupplierAccountHealthGuardUnavailableCause(cause)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商账号健康守护不可用原因失败: %w", err)
	}
	return causes, nil
}
