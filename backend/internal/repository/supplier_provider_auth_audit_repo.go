package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierProviderAuthAuditRepository struct {
	db *sql.DB
}

func NewSupplierProviderAuthAuditRepository(db *sql.DB) service.SupplierProviderAuthAuditRepository {
	return &supplierProviderAuthAuditRepository{db: db}
}

func (r *supplierProviderAuthAuditRepository) Record(ctx context.Context, event service.SupplierProviderAuthEventRecord) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider auth audit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
INSERT INTO supplier_provider_auth_events (
  provider_id, event_type, source, status, started_at, finished_at, duration_ms,
  http_status, error_message, token_fingerprint, token_length, token_expires_at,
  cookie_present, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		event.ProviderID, event.EventType, event.Source, event.Status,
		event.StartedAt, event.FinishedAt, event.DurationMS, event.HTTPStatus,
		event.ErrorMessage, event.TokenFingerprint, event.TokenLength,
		event.TokenExpiresAt, event.CookiePresent, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert supplier provider auth event: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO supplier_provider_runtime_stats (
  provider_id,
  auth_login_count, auth_login_success_count, auth_login_failure_count,
  auth_cache_hit_count, auth_cache_miss_count,
  auth_last_login_at, auth_last_login_status, auth_last_login_error,
  auth_last_cache_hit_at, auth_last_cache_error,
  auth_last_token_expires_at, auth_last_token_fingerprint, updated_at
) VALUES (
  $1,
  CASE WHEN $2 IN ('login_success', 'login_failed') THEN 1 ELSE 0 END,
  CASE WHEN $2 = 'login_success' THEN 1 ELSE 0 END,
  CASE WHEN $2 = 'login_failed' THEN 1 ELSE 0 END,
  CASE WHEN $2 = 'cache_hit' THEN 1 ELSE 0 END,
  CASE WHEN $2 = 'cache_miss' THEN 1 ELSE 0 END,
  CASE WHEN $2 IN ('login_success', 'login_failed') THEN $4 ELSE NULL END,
  CASE WHEN $2 IN ('login_success', 'login_failed') THEN $3 ELSE '' END,
  CASE WHEN $2 = 'login_failed' THEN $5 ELSE '' END,
  CASE WHEN $2 = 'cache_hit' THEN $4 ELSE NULL END,
  CASE WHEN $2 = 'cache_error' THEN $5 ELSE '' END,
  $6,
  $7,
  $4
)
ON CONFLICT (provider_id) DO UPDATE SET
  auth_login_count = supplier_provider_runtime_stats.auth_login_count + EXCLUDED.auth_login_count,
  auth_login_success_count = supplier_provider_runtime_stats.auth_login_success_count + EXCLUDED.auth_login_success_count,
  auth_login_failure_count = supplier_provider_runtime_stats.auth_login_failure_count + EXCLUDED.auth_login_failure_count,
  auth_cache_hit_count = supplier_provider_runtime_stats.auth_cache_hit_count + EXCLUDED.auth_cache_hit_count,
  auth_cache_miss_count = supplier_provider_runtime_stats.auth_cache_miss_count + EXCLUDED.auth_cache_miss_count,
  auth_last_login_at = CASE
    WHEN $2 IN ('login_success', 'login_failed') THEN EXCLUDED.auth_last_login_at
    ELSE supplier_provider_runtime_stats.auth_last_login_at
  END,
  auth_last_login_status = CASE
    WHEN $2 IN ('login_success', 'login_failed') THEN EXCLUDED.auth_last_login_status
    ELSE supplier_provider_runtime_stats.auth_last_login_status
  END,
  auth_last_login_error = CASE
    WHEN $2 = 'login_success' THEN ''
    WHEN $2 = 'login_failed' THEN EXCLUDED.auth_last_login_error
    ELSE supplier_provider_runtime_stats.auth_last_login_error
  END,
  auth_last_cache_hit_at = CASE
    WHEN $2 = 'cache_hit' THEN EXCLUDED.auth_last_cache_hit_at
    ELSE supplier_provider_runtime_stats.auth_last_cache_hit_at
  END,
  auth_last_cache_error = CASE
    WHEN $2 = 'cache_error' THEN EXCLUDED.auth_last_cache_error
    WHEN $2 = 'cache_hit' THEN ''
    ELSE supplier_provider_runtime_stats.auth_last_cache_error
  END,
  auth_last_token_expires_at = COALESCE(EXCLUDED.auth_last_token_expires_at, supplier_provider_runtime_stats.auth_last_token_expires_at),
  auth_last_token_fingerprint = CASE
    WHEN EXCLUDED.auth_last_token_fingerprint <> '' THEN EXCLUDED.auth_last_token_fingerprint
    ELSE supplier_provider_runtime_stats.auth_last_token_fingerprint
  END,
  updated_at = EXCLUDED.updated_at`,
		event.ProviderID, event.EventType, event.Status, event.FinishedAt,
		event.ErrorMessage, event.TokenExpiresAt, event.TokenFingerprint,
	)
	if err != nil {
		return fmt.Errorf("update supplier provider auth summary: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider auth audit transaction: %w", err)
	}
	return nil
}

func (r *supplierProviderAuthAuditRepository) GetSummary(ctx context.Context, providerID int64) (service.SupplierProviderAuthSummary, error) {
	var summary service.SupplierProviderAuthSummary
	var lastLoginAt sql.NullTime
	var lastCacheHitAt sql.NullTime
	var lastTokenExpiresAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(s.auth_login_count, 0),
       COALESCE(s.auth_login_success_count, 0),
       COALESCE(s.auth_login_failure_count, 0),
       COALESCE(s.auth_cache_hit_count, 0),
       COALESCE(s.auth_cache_miss_count, 0),
       s.auth_last_login_at,
       COALESCE(s.auth_last_login_status, ''),
       COALESCE(s.auth_last_login_error, ''),
       s.auth_last_cache_hit_at,
       COALESCE(s.auth_last_cache_error, ''),
       s.auth_last_token_expires_at,
       COALESCE(s.auth_last_token_fingerprint, '')
FROM supplier_providers p
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = p.id
WHERE p.id = $1 AND p.deleted_at IS NULL`, providerID).Scan(
		&summary.LoginCount,
		&summary.LoginSuccessCount,
		&summary.LoginFailureCount,
		&summary.CacheHitCount,
		&summary.CacheMissCount,
		&lastLoginAt,
		&summary.LastLoginStatus,
		&summary.LastLoginError,
		&lastCacheHitAt,
		&summary.LastCacheError,
		&lastTokenExpiresAt,
		&summary.LastTokenFingerprint,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return service.SupplierProviderAuthSummary{}, service.ErrSupplierProviderNotFound
	}
	if err != nil {
		return service.SupplierProviderAuthSummary{}, fmt.Errorf("get supplier provider auth summary: %w", err)
	}
	if lastLoginAt.Valid {
		summary.LastLoginAt = &lastLoginAt.Time
	}
	if lastCacheHitAt.Valid {
		summary.LastCacheHitAt = &lastCacheHitAt.Time
	}
	if lastTokenExpiresAt.Valid {
		summary.LastTokenExpiresAt = &lastTokenExpiresAt.Time
	}
	return summary, nil
}

func (r *supplierProviderAuthAuditRepository) ListHistory(ctx context.Context, providerID int64, params service.SupplierProviderAuthHistoryParams) (service.SupplierProviderAuthHistoryResult, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if params.EventType != "" && !isSupplierProviderAuthEventType(params.EventType) {
		return service.SupplierProviderAuthHistoryResult{}, fmt.Errorf("invalid supplier provider auth event type: %s", params.EventType)
	}

	where := "provider_id = $1"
	args := []any{providerID}
	if params.EventType != "" {
		where += " AND event_type = $2"
		args = append(args, params.EventType)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_provider_auth_events WHERE "+where, args...).Scan(&total); err != nil {
		return service.SupplierProviderAuthHistoryResult{}, fmt.Errorf("count supplier provider auth history: %w", err)
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	query := fmt.Sprintf(`
SELECT id, provider_id, event_type, source, status, started_at, finished_at,
       duration_ms, http_status, error_message, token_fingerprint, token_length,
       token_expires_at, cookie_present, created_at
FROM supplier_provider_auth_events
WHERE %s
ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d`, where, limitArg, offsetArg)
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return service.SupplierProviderAuthHistoryResult{}, fmt.Errorf("query supplier provider auth history: %w", err)
	}
	defer rows.Close()

	items := make([]service.SupplierProviderAuthHistoryItem, 0)
	for rows.Next() {
		var item service.SupplierProviderAuthHistoryItem
		var httpStatus sql.NullInt64
		var tokenExpiresAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.ProviderID,
			&item.EventType,
			&item.Source,
			&item.Status,
			&item.StartedAt,
			&item.FinishedAt,
			&item.DurationMS,
			&httpStatus,
			&item.ErrorMessage,
			&item.TokenFingerprint,
			&item.TokenLength,
			&tokenExpiresAt,
			&item.CookiePresent,
			&item.CreatedAt,
		); err != nil {
			return service.SupplierProviderAuthHistoryResult{}, fmt.Errorf("scan supplier provider auth history: %w", err)
		}
		if httpStatus.Valid {
			value := int(httpStatus.Int64)
			item.HTTPStatus = &value
		}
		if tokenExpiresAt.Valid {
			item.TokenExpiresAt = &tokenExpiresAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierProviderAuthHistoryResult{}, fmt.Errorf("iterate supplier provider auth history: %w", err)
	}
	return service.SupplierProviderAuthHistoryResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func isSupplierProviderAuthEventType(eventType service.SupplierProviderAuthEventType) bool {
	switch eventType {
	case service.SupplierProviderAuthEventCacheHit,
		service.SupplierProviderAuthEventCacheMiss,
		service.SupplierProviderAuthEventLoginSuccess,
		service.SupplierProviderAuthEventLoginFailed,
		service.SupplierProviderAuthEventCacheInvalidated,
		service.SupplierProviderAuthEventCacheError:
		return true
	default:
		return false
	}
}
