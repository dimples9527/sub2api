package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type supplierProviderRepository struct {
	db *sql.DB
}

type supplierProviderTypeRepository struct {
	db *sql.DB
}

func NewSupplierProviderRepository(db *sql.DB) service.SupplierProviderRepository {
	return &supplierProviderRepository{db: db}
}

func NewSupplierProviderTypeRepository(db *sql.DB) service.SupplierProviderTypeRepository {
	return &supplierProviderTypeRepository{db: db}
}

const supplierProviderSelect = `
SELECT p.id, p.code, p.name, p.provider_type, p.base_url, p.login_url,
       p.api_keys_url, p.groups_url, p.available_groups_url, p.balance_url,
       p.usage_cost_url, p.account_name_prefix, p.temp_disable_minutes,
       p.account_rate_multiplier_scale, p.sort_order, p.enabled, p.is_default,
       p.created_at, p.updated_at,
       COALESCE(c.email, ''), COALESCE(c.username, ''), COALESCE(c.password_encrypted, ''),
       COALESCE(s.status, 'unknown'), COALESCE(s.risk_level, 'normal'),
       COALESCE(s.valid_account_count, 0), COALESCE(s.schedulable_account_count, 0),
       COALESCE(s.request_count, 0), COALESCE(s.success_rate, 0),
       COALESCE(s.period_cost, 0), COALESCE(s.current_balance, 0),
       COALESCE(s.today_cost, 0), s.estimated_days, COALESCE(s.rate_risk_count, 0),
       COALESCE(s.sync_status, 'never'), COALESCE(s.sync_message, ''), s.last_sync_at
FROM supplier_providers p
LEFT JOIN supplier_provider_credentials c ON c.provider_id = p.id
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = p.id`

func (r *supplierProviderRepository) List(ctx context.Context, params service.SupplierProviderListParams) ([]*service.SupplierProvider, int64, error) {
	where, args := supplierProviderWhere(params)
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_providers p WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count supplier providers: %w", err)
	}
	page, pageSize := params.Page, params.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := supplierProviderSelect + " WHERE " + where + fmt.Sprintf(" ORDER BY p.sort_order ASC, p.id ASC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query supplier providers: %w", err)
	}
	defer rows.Close()
	items := make([]*service.SupplierProvider, 0)
	for rows.Next() {
		provider, scanErr := scanSupplierProvider(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate supplier providers: %w", err)
	}
	return items, total, nil
}

func (r *supplierProviderRepository) Summary(ctx context.Context, params service.SupplierProviderListParams) (service.SupplierProviderSummary, error) {
	where, args := supplierProviderWhere(params)
	var summary service.SupplierProviderSummary
	err := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(*) AS total_count,
  COUNT(*) FILTER (WHERE p.enabled = TRUE) AS enabled_count,
  COUNT(*) FILTER (WHERE COALESCE(s.risk_level, 'normal') IN ('high', 'critical')) AS high_risk_count,
  COUNT(*) FILTER (WHERE s.estimated_days IS NOT NULL AND s.estimated_days < 3) AS low_balance_count,
  COUNT(*) FILTER (WHERE COALESCE(s.sync_status, 'never') = 'failed') AS sync_failure_count,
  COALESCE(SUM(COALESCE(s.rate_risk_count, 0)), 0) AS rate_risk_count
FROM supplier_providers p
LEFT JOIN supplier_provider_runtime_stats s ON s.provider_id = p.id
WHERE `+where, args...).Scan(
		&summary.TotalCount,
		&summary.EnabledCount,
		&summary.HighRiskCount,
		&summary.LowBalanceCount,
		&summary.SyncFailureCount,
		&summary.RateRiskCount,
	)
	if err != nil {
		return service.SupplierProviderSummary{}, fmt.Errorf("summarize supplier providers: %w", err)
	}
	return summary, nil
}

func supplierProviderWhere(params service.SupplierProviderListParams) (string, []any) {
	conditions := []string{"p.deleted_at IS NULL"}
	args := make([]any, 0, 2)
	if search := strings.TrimSpace(params.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("(p.name ILIKE $%d OR p.code ILIKE $%d OR p.base_url ILIKE $%d)", len(args), len(args), len(args)))
	}
	if params.Enabled != nil {
		args = append(args, *params.Enabled)
		conditions = append(conditions, fmt.Sprintf("p.enabled = $%d", len(args)))
	}
	return strings.Join(conditions, " AND "), args
}

func (r *supplierProviderRepository) GetByID(ctx context.Context, id int64) (*service.SupplierProvider, error) {
	provider, err := scanSupplierProvider(r.db.QueryRowContext(ctx, supplierProviderSelect+" WHERE p.id = $1 AND p.deleted_at IS NULL", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierProviderNotFound
	}
	return provider, err
}

func (r *supplierProviderRepository) Create(ctx context.Context, provider *service.SupplierProvider) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var hasActive bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM supplier_providers WHERE deleted_at IS NULL)").Scan(&hasActive); err != nil {
		return err
	}
	if provider.IsDefault || !hasActive {
		if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET is_default = FALSE, updated_at = NOW() WHERE deleted_at IS NULL AND is_default = TRUE"); err != nil {
			return err
		}
		provider.IsDefault = true
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO supplier_providers (
  code, name, provider_type, base_url, login_url, api_keys_url, groups_url,
  available_groups_url, balance_url, usage_cost_url, account_name_prefix,
  temp_disable_minutes, account_rate_multiplier_scale, sort_order, enabled, is_default
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
RETURNING id, created_at, updated_at`, provider.Code, provider.Name, provider.ProviderType,
		provider.BaseURL, provider.LoginURL, provider.APIKeysURL, provider.GroupsURL,
		provider.AvailableGroupsURL, provider.BalanceURL, provider.UsageCostURL,
		provider.AccountNamePrefix, provider.TempDisableMinutes, provider.AccountRateMultiplierScale,
		provider.SortOrder, provider.Enabled, provider.IsDefault).Scan(&provider.ID, &provider.CreatedAt, &provider.UpdatedAt)
	if err != nil {
		return mapSupplierProviderError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO supplier_provider_credentials (provider_id, email, username, password_encrypted) VALUES ($1,$2,$3,$4)`, provider.ID, provider.Email, provider.Username, provider.PasswordEncrypted); err != nil {
		return fmt.Errorf("insert supplier credentials: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO supplier_provider_runtime_stats (provider_id) VALUES ($1)`, provider.ID); err != nil {
		return fmt.Errorf("insert supplier runtime stats: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider create: %w", err)
	}
	return nil
}

func (r *supplierProviderRepository) Update(ctx context.Context, provider *service.SupplierProvider) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if provider.IsDefault {
		if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET is_default = FALSE, updated_at = NOW() WHERE id <> $1 AND deleted_at IS NULL AND is_default = TRUE", provider.ID); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `
UPDATE supplier_providers SET
  code=$2, name=$3, provider_type=$4, base_url=$5, login_url=$6,
  api_keys_url=$7, groups_url=$8, available_groups_url=$9, balance_url=$10,
  usage_cost_url=$11, account_name_prefix=$12, temp_disable_minutes=$13,
  account_rate_multiplier_scale=$14, sort_order=$15, enabled=$16,
  is_default=$17, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, provider.ID, provider.Code, provider.Name,
		provider.ProviderType, provider.BaseURL, provider.LoginURL, provider.APIKeysURL,
		provider.GroupsURL, provider.AvailableGroupsURL, provider.BalanceURL,
		provider.UsageCostURL, provider.AccountNamePrefix, provider.TempDisableMinutes,
		provider.AccountRateMultiplierScale, provider.SortOrder, provider.Enabled, provider.IsDefault)
	if err != nil {
		return mapSupplierProviderError(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierProviderNotFound
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO supplier_provider_credentials (provider_id, email, username, password_encrypted, updated_at)
VALUES ($1,$2,$3,$4,NOW())
ON CONFLICT (provider_id) DO UPDATE SET email=EXCLUDED.email, username=EXCLUDED.username,
password_encrypted=EXCLUDED.password_encrypted, updated_at=NOW()`, provider.ID, provider.Email, provider.Username, provider.PasswordEncrypted)
	if err != nil {
		return fmt.Errorf("update supplier credentials: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider update: %w", err)
	}
	return nil
}

func (r *supplierProviderRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var wasDefault bool
	if err := tx.QueryRowContext(ctx, "SELECT is_default FROM supplier_providers WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&wasDefault); errors.Is(err, sql.ErrNoRows) {
		return service.ErrSupplierProviderNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM supplier_provider_credentials WHERE provider_id=$1", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET enabled=FALSE, is_default=FALSE, deleted_at=NOW(), updated_at=NOW() WHERE id=$1", id); err != nil {
		return err
	}
	if wasDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE supplier_providers SET is_default=TRUE, updated_at=NOW() WHERE id=(SELECT id FROM supplier_providers WHERE deleted_at IS NULL ORDER BY enabled DESC, sort_order ASC, id ASC LIMIT 1)`); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *supplierProviderRepository) SetDefault(ctx context.Context, id int64) (*service.SupplierProvider, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var exists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM supplier_providers WHERE id=$1 AND deleted_at IS NULL)", id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, service.ErrSupplierProviderNotFound
	}
	if _, err := tx.ExecContext(ctx, "UPDATE supplier_providers SET is_default=(id=$1), updated_at=NOW() WHERE deleted_at IS NULL", id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

type supplierProviderScanner interface{ Scan(dest ...any) error }

func scanSupplierProvider(scanner supplierProviderScanner) (*service.SupplierProvider, error) {
	provider := &service.SupplierProvider{}
	var estimatedDays sql.NullFloat64
	var lastSyncAt sql.NullTime
	err := scanner.Scan(&provider.ID, &provider.Code, &provider.Name, &provider.ProviderType,
		&provider.BaseURL, &provider.LoginURL, &provider.APIKeysURL, &provider.GroupsURL,
		&provider.AvailableGroupsURL, &provider.BalanceURL, &provider.UsageCostURL,
		&provider.AccountNamePrefix, &provider.TempDisableMinutes,
		&provider.AccountRateMultiplierScale, &provider.SortOrder, &provider.Enabled,
		&provider.IsDefault, &provider.CreatedAt, &provider.UpdatedAt, &provider.Email,
		&provider.Username, &provider.PasswordEncrypted, &provider.Status, &provider.RiskLevel,
		&provider.ValidAccountCount, &provider.SchedulableAccountCount, &provider.RequestCount,
		&provider.SuccessRate, &provider.PeriodCost, &provider.CurrentBalance, &provider.TodayCost,
		&estimatedDays, &provider.RateRiskCount, &provider.SyncStatus, &provider.SyncMessage, &lastSyncAt)
	if err != nil {
		return nil, err
	}
	if estimatedDays.Valid {
		provider.EstimatedDays = &estimatedDays.Float64
	}
	if lastSyncAt.Valid {
		provider.LastSyncAt = &lastSyncAt.Time
	}
	provider.CredentialConfigured = provider.PasswordEncrypted != ""
	return provider, nil
}

func mapSupplierProviderError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrSupplierProviderExists
	}
	return err
}

const supplierProviderTypeSelect = `
SELECT id, code, name, login_url, api_keys_url, groups_url, available_groups_url,
       balance_url, usage_cost_url, enabled, sort_order, created_at, updated_at
FROM supplier_provider_types`

func (r *supplierProviderTypeRepository) List(ctx context.Context, enabledOnly bool) ([]*service.SupplierProviderType, error) {
	where := "deleted_at IS NULL"
	if enabledOnly {
		where += " AND enabled = TRUE"
	}
	rows, err := r.db.QueryContext(ctx, supplierProviderTypeSelect+" WHERE "+where+" ORDER BY sort_order ASC, id ASC")
	if err != nil {
		return nil, fmt.Errorf("query supplier provider types: %w", err)
	}
	defer rows.Close()
	items := make([]*service.SupplierProviderType, 0)
	for rows.Next() {
		item, scanErr := scanSupplierProviderType(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplier provider types: %w", err)
	}
	return items, nil
}

func (r *supplierProviderTypeRepository) GetByID(ctx context.Context, id int64) (*service.SupplierProviderType, error) {
	item, err := scanSupplierProviderType(r.db.QueryRowContext(ctx, supplierProviderTypeSelect+" WHERE id=$1 AND deleted_at IS NULL", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierProviderTypeNotFound
	}
	return item, err
}

func (r *supplierProviderTypeRepository) GetByCode(ctx context.Context, code string) (*service.SupplierProviderType, error) {
	item, err := scanSupplierProviderType(r.db.QueryRowContext(ctx, supplierProviderTypeSelect+" WHERE code=$1 AND deleted_at IS NULL", code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierProviderTypeNotFound
	}
	return item, err
}

func (r *supplierProviderTypeRepository) Create(ctx context.Context, item *service.SupplierProviderType) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_provider_types (
  code, name, login_url, api_keys_url, groups_url, available_groups_url,
  balance_url, usage_cost_url, enabled, sort_order
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id, created_at, updated_at`, item.Code, item.Name, item.LoginURL, item.APIKeysURL,
		item.GroupsURL, item.AvailableGroupsURL, item.BalanceURL, item.UsageCostURL,
		item.Enabled, item.SortOrder).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return mapSupplierProviderTypeError(err)
	}
	return nil
}

func (r *supplierProviderTypeRepository) Update(ctx context.Context, item *service.SupplierProviderType) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_provider_types SET
  code=$2, name=$3, login_url=$4, api_keys_url=$5, groups_url=$6,
  available_groups_url=$7, balance_url=$8, usage_cost_url=$9,
  enabled=$10, sort_order=$11, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, item.ID, item.Code, item.Name, item.LoginURL,
		item.APIKeysURL, item.GroupsURL, item.AvailableGroupsURL, item.BalanceURL,
		item.UsageCostURL, item.Enabled, item.SortOrder)
	if err != nil {
		return mapSupplierProviderTypeError(err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierProviderTypeNotFound
	}
	return nil
}

func (r *supplierProviderTypeRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "UPDATE supplier_provider_types SET enabled=FALSE, deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierProviderTypeNotFound
	}
	return nil
}

type supplierProviderTypeScanner interface{ Scan(dest ...any) error }

func scanSupplierProviderType(scanner supplierProviderTypeScanner) (*service.SupplierProviderType, error) {
	item := &service.SupplierProviderType{}
	err := scanner.Scan(&item.ID, &item.Code, &item.Name, &item.LoginURL,
		&item.APIKeysURL, &item.GroupsURL, &item.AvailableGroupsURL,
		&item.BalanceURL, &item.UsageCostURL, &item.Enabled, &item.SortOrder,
		&item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func mapSupplierProviderTypeError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrSupplierProviderTypeExists
	}
	return err
}
