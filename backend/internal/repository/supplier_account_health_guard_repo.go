package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
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
