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

type supplierProviderRechargeRepository struct {
	db *sql.DB
}

func NewSupplierProviderRechargeRepository(db *sql.DB) service.SupplierProviderRechargeRepository {
	return &supplierProviderRechargeRepository{db: db}
}

func (r *supplierProviderRechargeRepository) Upsert(ctx context.Context, providerID int64, records []service.SupplierProviderRechargeRecord) error {
	if providerID <= 0 || len(records) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin supplier provider recharge upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	// 充值记录一旦落库几乎不再变化，而同步每 30 分钟就会重放整段历史。DO UPDATE 的
	// WHERE 只比对业务字段（不含 synced_at/updated_at，否则每次都不相等），让未变化的行
	// 不产生死元组——堆表膨胀会拖慢列表页那条全表 COUNT/SUM。
	const query = `
INSERT INTO supplier_provider_recharges (
  provider_id, external_id, external_code, recharge_type, amount, status,
  occurred_at, description, source_payload, synced_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)
ON CONFLICT (provider_id, external_id) DO UPDATE SET
  external_code = EXCLUDED.external_code,
  recharge_type = EXCLUDED.recharge_type,
  amount = EXCLUDED.amount,
  status = EXCLUDED.status,
  occurred_at = EXCLUDED.occurred_at,
  description = EXCLUDED.description,
  source_payload = EXCLUDED.source_payload,
  synced_at = EXCLUDED.synced_at,
  updated_at = EXCLUDED.updated_at
WHERE supplier_provider_recharges.external_code IS DISTINCT FROM EXCLUDED.external_code
   OR supplier_provider_recharges.recharge_type IS DISTINCT FROM EXCLUDED.recharge_type
   OR supplier_provider_recharges.amount IS DISTINCT FROM EXCLUDED.amount
   OR supplier_provider_recharges.status IS DISTINCT FROM EXCLUDED.status
   OR supplier_provider_recharges.occurred_at IS DISTINCT FROM EXCLUDED.occurred_at
   OR supplier_provider_recharges.description IS DISTINCT FROM EXCLUDED.description
   OR supplier_provider_recharges.source_payload IS DISTINCT FROM EXCLUDED.source_payload`
	now := time.Now()
	for _, record := range records {
		externalID := strings.TrimSpace(record.ExternalID)
		if externalID == "" {
			externalID = strings.TrimSpace(record.ExternalCode)
		}
		if externalID == "" || record.Amount < 0 || record.OccurredAt.IsZero() {
			continue
		}
		payload := record.RawPayload
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		if _, err := tx.ExecContext(ctx, query, providerID, externalID,
			strings.TrimSpace(record.ExternalCode), strings.TrimSpace(record.RechargeType),
			record.Amount, strings.TrimSpace(record.Status), record.OccurredAt,
			strings.TrimSpace(record.Description), payload, now); err != nil {
			return fmt.Errorf("upsert supplier provider recharge %q: %w", externalID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit supplier provider recharge upsert: %w", err)
	}
	return nil
}

func (r *supplierProviderRechargeRepository) List(ctx context.Context, params service.SupplierProviderRechargeListParams) (service.SupplierProviderRechargeListResult, error) {
	page, pageSize := params.Page, params.PageSize
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 200 { pageSize = 100 }
	where, args := supplierProviderRechargeWhere(params)
	var result service.SupplierProviderRechargeListResult
	result.Page, result.PageSize = page, pageSize
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(r.amount), 0) FROM supplier_provider_recharges r JOIN supplier_providers p ON p.id = r.provider_id WHERE `+where, args...).Scan(&result.Total, &result.TotalAmount); err != nil {
		return result, fmt.Errorf("count supplier provider recharges: %w", err)
	}
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := `
SELECT r.id, r.provider_id, p.name, p.provider_type, r.external_id, r.external_code,
       r.recharge_type, r.amount, r.status, r.occurred_at, r.description,
       r.synced_at, r.created_at, r.updated_at
FROM supplier_provider_recharges r
JOIN supplier_providers p ON p.id = r.provider_id
WHERE ` + where + fmt.Sprintf(` ORDER BY r.occurred_at DESC, r.id DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil { return result, fmt.Errorf("list supplier provider recharges: %w", err) }
	defer rows.Close()
	result.Items = make([]service.SupplierProviderRecharge, 0)
	for rows.Next() {
		var item service.SupplierProviderRecharge
		if err := rows.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.ProviderType,
			&item.ExternalID, &item.ExternalCode, &item.RechargeType, &item.Amount, &item.Status,
			&item.OccurredAt, &item.Description, &item.SyncedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return result, fmt.Errorf("scan supplier provider recharge: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil { return result, fmt.Errorf("iterate supplier provider recharges: %w", err) }
	return result, nil
}

func supplierProviderRechargeWhere(params service.SupplierProviderRechargeListParams) (string, []any) {
	conditions := []string{"p.deleted_at IS NULL"}
	args := make([]any, 0, 3)
	if params.ProviderID > 0 { args = append(args, params.ProviderID); conditions = append(conditions, fmt.Sprintf("r.provider_id = $%d", len(args))) }
	if !params.Start.IsZero() { args = append(args, params.Start); conditions = append(conditions, fmt.Sprintf("r.occurred_at >= $%d", len(args))) }
	if !params.End.IsZero() { args = append(args, params.End); conditions = append(conditions, fmt.Sprintf("r.occurred_at < $%d", len(args))) }
	return strings.Join(conditions, " AND "), args
}

func (r *supplierProviderRechargeRepository) Sum(ctx context.Context, providerID int64, start, end time.Time) (float64, error) {
	where := []string{"provider_id = $1"}
	args := []any{providerID}
	if !start.IsZero() { args = append(args, start); where = append(where, fmt.Sprintf("occurred_at >= $%d", len(args))) }
	if !end.IsZero() { args = append(args, end); where = append(where, fmt.Sprintf("occurred_at < $%d", len(args))) }
	var total float64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount), 0) FROM supplier_provider_recharges WHERE `+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum supplier provider recharges: %w", err)
	}
	return total, nil
}

func (r *supplierProviderRechargeRepository) HasRecords(ctx context.Context, providerID int64) (bool, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM supplier_provider_recharges WHERE provider_id = $1)`, providerID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check supplier provider recharge records: %w", err)
	}
	return exists, nil
}
