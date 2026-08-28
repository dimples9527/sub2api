package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierProviderCostReviewRepository struct{ db *sql.DB }

func NewSupplierProviderCostReviewRepository(db *sql.DB) service.SupplierProviderCostReviewRepository {
	return &supplierProviderCostReviewRepository{db: db}
}

const costReviewSelect = `SELECT r.id, r.provider_id, p.name AS provider_name, r.stat_date,
       r.upstream_cost, r.calculated_cost, r.auto_adopted_cost, r.final_cost, r.effective_cost,
       r.cost_delta, r.effective_delta, r.status, r.decision_type, r.approved_by, r.approved_at,
       r.sync_count, r.last_sync_run_id, r.last_synced_at, r.version, r.created_at, r.updated_at
FROM supplier_provider_cost_reviews r
JOIN supplier_providers p ON p.id = r.provider_id`

func (r *supplierProviderCostReviewRepository) List(ctx context.Context, params service.SupplierProviderCostReviewListParams) (service.SupplierProviderCostReviewListResult, error) {
	page, pageSize := params.Page, params.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	keyword := strings.TrimSpace(params.Keyword)
	keywordPattern := ""
	if keyword != "" {
		keywordPattern = "%" + keyword + "%"
	}
	args := []any{params.ProviderID, params.StartDate, params.EndDate, params.Status, keywordPattern}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM supplier_provider_cost_reviews r
JOIN supplier_providers p ON p.id = r.provider_id
WHERE ($1 = 0 OR r.provider_id = $1)
  AND ($2 = '' OR r.stat_date >= $2::date)
  AND ($3 = '' OR r.stat_date <= $3::date)
  AND ($4 = '' OR r.status = $4)
  AND ($5 = '' OR p.name ILIKE $5)`, args...).Scan(&total); err != nil {
		return service.SupplierProviderCostReviewListResult{}, fmt.Errorf("查询成本核对记录总数失败: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, costReviewSelect+`
WHERE ($1 = 0 OR r.provider_id = $1)
  AND ($2 = '' OR r.stat_date >= $2::date)
  AND ($3 = '' OR r.stat_date <= $3::date)
  AND ($4 = '' OR r.status = $4)
  AND ($5 = '' OR p.name ILIKE $5)
ORDER BY r.stat_date DESC, r.id DESC
LIMIT $6 OFFSET $7`, params.ProviderID, params.StartDate, params.EndDate, params.Status, keywordPattern, pageSize, offset)
	if err != nil {
		return service.SupplierProviderCostReviewListResult{}, fmt.Errorf("查询成本核对记录失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierProviderCostReview, 0)
	for rows.Next() {
		item, err := scanCostReview(rows)
		if err != nil {
			return service.SupplierProviderCostReviewListResult{}, fmt.Errorf("读取成本核对记录失败: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierProviderCostReviewListResult{}, fmt.Errorf("遍历成本核对记录失败: %w", err)
	}
	return service.SupplierProviderCostReviewListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *supplierProviderCostReviewRepository) History(ctx context.Context, reviewID int64) ([]service.SupplierProviderCostReviewHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, review_id, provider_id, stat_date, event_type, sync_run_id,
       upstream_cost, calculated_cost, auto_adopted_cost, final_cost, cost_delta, effective_delta,
       status, decision_type, manual_cost, operator_id, operated_at
FROM supplier_provider_cost_review_histories
WHERE review_id = $1
ORDER BY operated_at DESC, id DESC`, reviewID)
	if err != nil {
		return nil, fmt.Errorf("查询成本核对历史失败: %w", err)
	}
	defer rows.Close()
	histories := make([]service.SupplierProviderCostReviewHistory, 0)
	for rows.Next() {
		item, err := scanCostReviewHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("读取成本核对历史失败: %w", err)
		}
		histories = append(histories, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历成本核对历史失败: %w", err)
	}
	return histories, nil
}

func (r *supplierProviderCostReviewRepository) Approve(ctx context.Context, reviewID int64, input service.SupplierProviderCostReviewApproveInput) (*service.SupplierProviderCostReview, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开始成本核对审批事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	review, err := approveCostReviewTx(ctx, tx, reviewID, input, false)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交成本核对审批事务失败: %w", err)
	}
	return &review, nil
}

func (r *supplierProviderCostReviewRepository) ApproveMany(ctx context.Context, input service.SupplierProviderCostReviewBulkApproveInput) ([]service.SupplierProviderCostReview, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开始成本核对批量审批事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	items := make([]service.SupplierProviderCostReview, 0, len(input.Items))
	for _, item := range input.Items {
		review, err := approveCostReviewTx(ctx, tx, item.ID, service.SupplierProviderCostReviewApproveInput{
			DecisionType: input.DecisionType,
			ManualCost:   input.ManualCost,
			Version:      item.Version,
			OperatorID:   input.OperatorID,
		}, true)
		if err != nil {
			return nil, err
		}
		items = append(items, review)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交成本核对批量审批事务失败: %w", err)
	}
	return items, nil
}

func approveCostReviewTx(ctx context.Context, tx *sql.Tx, reviewID int64, input service.SupplierProviderCostReviewApproveInput, requireApprovableStatus bool) (service.SupplierProviderCostReview, error) {
	review, err := queryCostReviewForUpdate(ctx, tx, reviewID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.SupplierProviderCostReview{}, service.ErrSupplierProviderCostReviewNotFound
	}
	if err != nil {
		return service.SupplierProviderCostReview{}, fmt.Errorf("锁定成本核对记录失败: %w", err)
	}
	if requireApprovableStatus && review.Status != service.CostReviewStatusPending && review.Status != service.CostReviewStatusChanged {
		return service.SupplierProviderCostReview{}, fmt.Errorf("成本核对记录当前状态不可审批")
	}
	if review.Version != input.Version {
		return service.SupplierProviderCostReview{}, service.ErrSupplierProviderCostReviewVersionConflict
	}

	finalCost, err := selectedCost(review, input)
	if err != nil {
		return service.SupplierProviderCostReview{}, err
	}
	effectiveDelta := costReviewRound6(finalCost - costReviewValueOrZero(review.CalculatedCost))
	result, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_cost_reviews
SET final_cost = $1, effective_cost = $1, effective_delta = $2,
    status = 'approved', decision_type = $3, approved_by = NULLIF($4, 0),
    approved_at = NOW(), version = version + 1, updated_at = NOW()
WHERE id = $5 AND version = $6`, finalCost, effectiveDelta, input.DecisionType, input.OperatorID, reviewID, input.Version)
	if err != nil {
		return service.SupplierProviderCostReview{}, fmt.Errorf("更新成本核对记录失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return service.SupplierProviderCostReview{}, fmt.Errorf("检查成本核对更新结果失败: %w", err)
	}
	if affected != 1 {
		return service.SupplierProviderCostReview{}, service.ErrSupplierProviderCostReviewVersionConflict
	}

	statDate := review.StatDate.Format("2006-01-02")
	if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_daily_stats (provider_id, stat_date, today_cost, updated_at)
VALUES ($1, $2::date, $3, NOW())
ON CONFLICT (provider_id, stat_date) DO UPDATE SET today_cost = EXCLUDED.today_cost, updated_at = NOW()`, review.ProviderID, statDate, finalCost); err != nil {
		return service.SupplierProviderCostReview{}, fmt.Errorf("更新每日成本失败: %w", err)
	}
	manualCost := any(nil)
	if input.DecisionType == service.CostReviewDecisionManual {
		manualCost = input.ManualCost
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_cost_review_histories (
  review_id, provider_id, stat_date, event_type, upstream_cost, calculated_cost, auto_adopted_cost,
  final_cost, cost_delta, effective_delta, status, decision_type, manual_cost, operator_id, operated_at
) VALUES ($1, $2, $3::date, 'approve', $4, $5, $6, $7, $8, $9, 'approved', $10, $11, NULLIF($12, 0), NOW())`,
		review.ID, review.ProviderID, statDate, review.UpstreamCost, review.CalculatedCost, review.AutoAdoptedCost,
		finalCost, review.CostDelta, effectiveDelta, input.DecisionType, manualCost, input.OperatorID); err != nil {
		return service.SupplierProviderCostReview{}, fmt.Errorf("写入成本核对审批历史失败: %w", err)
	}
	review.FinalCost = &finalCost
	review.EffectiveCost = finalCost
	review.EffectiveDelta = &effectiveDelta
	review.Status = service.CostReviewStatusApproved
	review.DecisionType = input.DecisionType
	review.Version++
	return review, nil
}

func (r *supplierProviderCostReviewRepository) Sync(ctx context.Context, input service.SupplierProviderCostReviewSyncInput) (*service.SupplierProviderCostReview, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开始成本核对同步事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	review, err := syncCostReviewTx(ctx, tx, input, true)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交成本核对同步事务失败: %w", err)
	}
	return &review, nil
}

func syncCostReviewTx(ctx context.Context, tx *sql.Tx, input service.SupplierProviderCostReviewSyncInput, writeDailyStats bool) (service.SupplierProviderCostReview, error) {
	syncedAt := input.SyncedAt.UTC()
	statDate := input.StatDate.UTC().Format("2006-01-02")
	review, err := queryCostReviewForUpdateByDate(ctx, tx, input.ProviderID, statDate)
	if errors.Is(err, sql.ErrNoRows) {
		effective := pendingEffectiveCost(input)
		delta := nullableDelta(input.UpstreamCost, input.CalculatedCost)
		effectiveDelta := costReviewRound6(effective - costReviewValueOrZero(input.CalculatedCost))
		var id int64
		err = tx.QueryRowContext(ctx, `
INSERT INTO supplier_provider_cost_reviews (
  provider_id, stat_date, upstream_cost, calculated_cost, auto_adopted_cost, final_cost, effective_cost,
  cost_delta, effective_delta, status, decision_type, sync_count, last_sync_run_id, last_synced_at, version
) VALUES ($1, $2::date, $3, $4, $5, NULL, $6, $7, $8, 'pending_review', 'none', 1, $9, $10, 1)
RETURNING id`, input.ProviderID, statDate, input.UpstreamCost, input.CalculatedCost, input.AutoAdoptedCost, effective, delta, effectiveDelta, input.SyncRunID, syncedAt).Scan(&id)
		if err != nil {
			return service.SupplierProviderCostReview{}, fmt.Errorf("创建成本核对记录失败: %w", err)
		}
		review = service.SupplierProviderCostReview{ID: id, ProviderID: input.ProviderID, StatDate: input.StatDate, UpstreamCost: input.UpstreamCost, CalculatedCost: input.CalculatedCost, AutoAdoptedCost: input.AutoAdoptedCost, EffectiveCost: effective, CostDelta: delta, EffectiveDelta: &effectiveDelta, Status: service.CostReviewStatusPending, DecisionType: service.CostReviewDecisionNone, SyncCount: 1, LastSyncRunID: input.SyncRunID, LastSyncedAt: &syncedAt, Version: 1}
	} else if err != nil {
		return service.SupplierProviderCostReview{}, fmt.Errorf("锁定成本核对记录失败: %w", err)
	} else {
		review = applySyncToReview(review, input)
		if _, err := tx.ExecContext(ctx, `
UPDATE supplier_provider_cost_reviews
SET upstream_cost = $1, calculated_cost = $2, auto_adopted_cost = $3, cost_delta = $4,
    effective_cost = $5, effective_delta = $6, status = $7,
    sync_count = sync_count + 1, last_sync_run_id = $8, last_synced_at = $9,
    version = version + 1, updated_at = NOW()
WHERE id = $10`, review.UpstreamCost, review.CalculatedCost, review.AutoAdoptedCost, review.CostDelta, review.EffectiveCost, review.EffectiveDelta, review.Status, input.SyncRunID, syncedAt, review.ID); err != nil {
			return service.SupplierProviderCostReview{}, fmt.Errorf("更新成本核对记录失败: %w", err)
		}
	}
	if writeDailyStats && review.Status != service.CostReviewStatusChanged {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_daily_stats (provider_id, stat_date, today_cost, updated_at)
VALUES ($1, $2::date, $3, NOW())
		ON CONFLICT (provider_id, stat_date) DO UPDATE SET today_cost = EXCLUDED.today_cost, updated_at = NOW()`, input.ProviderID, statDate, review.EffectiveCost); err != nil {
			return service.SupplierProviderCostReview{}, fmt.Errorf("写入每日成本失败: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO supplier_provider_cost_review_histories (
  review_id, provider_id, stat_date, event_type, sync_run_id, upstream_cost, calculated_cost, auto_adopted_cost,
  final_cost, cost_delta, effective_delta, status, decision_type, operated_at
) VALUES ($1, $2, $3::date, 'sync', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		review.ID, review.ProviderID, statDate, input.SyncRunID, review.UpstreamCost, review.CalculatedCost, review.AutoAdoptedCost,
		review.FinalCost, review.CostDelta, review.EffectiveDelta, review.Status, review.DecisionType, syncedAt); err != nil {
		return service.SupplierProviderCostReview{}, fmt.Errorf("写入成本核对历史失败: %w", err)
	}
	return review, nil
}

func applySyncToReview(review service.SupplierProviderCostReview, input service.SupplierProviderCostReviewSyncInput) service.SupplierProviderCostReview {
	wasApproved := review.Status == service.CostReviewStatusApproved
	latestDataUnchanged := wasApproved && sameCostReviewValue(review.UpstreamCost, input.UpstreamCost) &&
		sameCostReviewValue(review.CalculatedCost, input.CalculatedCost) &&
		sameCostReviewValue(review.AutoAdoptedCost, input.AutoAdoptedCost)
	review.UpstreamCost = input.UpstreamCost
	review.CalculatedCost = input.CalculatedCost
	review.AutoAdoptedCost = input.AutoAdoptedCost
	review.CostDelta = nullableDelta(input.UpstreamCost, input.CalculatedCost)
	if wasApproved && !latestDataUnchanged {
		review.Status = service.CostReviewStatusChanged
	} else if review.Status == service.CostReviewStatusPending {
		review.Status = service.CostReviewStatusPending
		review.EffectiveCost = pendingEffectiveCost(input)
		delta := costReviewRound6(review.EffectiveCost - costReviewValueOrZero(input.CalculatedCost))
		review.EffectiveDelta = &delta
	}
	review.SyncCount++
	review.LastSyncRunID = input.SyncRunID
	syncedAt := input.SyncedAt.UTC()
	review.LastSyncedAt = &syncedAt
	review.Version++
	return review
}

func pendingEffectiveCost(input service.SupplierProviderCostReviewSyncInput) float64 {
	// 上游成本优先模式：待审批记录默认采用上游接口成本。
	if input.CostSource == service.SupplierCostSourceUpstream && input.UpstreamCost != nil {
		return costReviewRound6(*input.UpstreamCost)
	}
	if input.CalculatedCost != nil {
		return costReviewRound6(*input.CalculatedCost)
	}
	return costReviewRound6(input.EffectiveCost)
}

func selectedCost(review service.SupplierProviderCostReview, input service.SupplierProviderCostReviewApproveInput) (float64, error) {
	switch input.DecisionType {
	case service.CostReviewDecisionUpstream:
		if review.UpstreamCost == nil {
			return 0, fmt.Errorf("missing upstream cost")
		}
		return *review.UpstreamCost, nil
	case service.CostReviewDecisionCalculated:
		if review.CalculatedCost == nil {
			return 0, fmt.Errorf("missing calculated cost")
		}
		return *review.CalculatedCost, nil
	case service.CostReviewDecisionManual:
		return *input.ManualCost, nil
	default:
		return 0, fmt.Errorf("invalid decision type")
	}
}

func queryCostReviewForUpdate(ctx context.Context, tx *sql.Tx, id int64) (service.SupplierProviderCostReview, error) {
	return scanCostReview(tx.QueryRowContext(ctx, costReviewSelect+` WHERE r.id = $1 FOR UPDATE`, id))
}
func queryCostReviewForUpdateByDate(ctx context.Context, tx *sql.Tx, providerID int64, statDate string) (service.SupplierProviderCostReview, error) {
	return scanCostReview(tx.QueryRowContext(ctx, costReviewSelect+` WHERE r.provider_id = $1 AND r.stat_date = $2::date FOR UPDATE`, providerID, statDate))
}

func scanCostReview(scanner interface{ Scan(...any) error }) (service.SupplierProviderCostReview, error) {
	var item service.SupplierProviderCostReview
	var upstream, calculated, autoAdopted, final, delta, effectiveDelta sql.NullFloat64
	var approvedBy, lastRun sql.NullInt64
	var approvedAt, lastSynced sql.NullTime
	err := scanner.Scan(&item.ID, &item.ProviderID, &item.ProviderName, &item.StatDate, &upstream, &calculated, &autoAdopted, &final, &item.EffectiveCost, &delta, &effectiveDelta, &item.Status, &item.DecisionType, &approvedBy, &approvedAt, &item.SyncCount, &lastRun, &lastSynced, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.UpstreamCost = nullFloatPtr(upstream)
	item.CalculatedCost = nullFloatPtr(calculated)
	item.AutoAdoptedCost = nullFloatPtr(autoAdopted)
	item.FinalCost = nullFloatPtr(final)
	item.CostDelta = nullFloatPtr(delta)
	item.EffectiveDelta = nullFloatPtr(effectiveDelta)
	item.ApprovedBy = nullIntPtr(approvedBy)
	item.LastSyncRunID = nullIntPtr(lastRun)
	item.ApprovedAt = nullTimePtr(approvedAt)
	item.LastSyncedAt = nullTimePtr(lastSynced)
	return item, nil
}

func scanCostReviewHistory(scanner interface{ Scan(...any) error }) (service.SupplierProviderCostReviewHistory, error) {
	var item service.SupplierProviderCostReviewHistory
	var reviewID, syncRunID, operatorID sql.NullInt64
	var upstream, calculated, autoAdopted, final, delta, effectiveDelta, manual sql.NullFloat64
	if err := scanner.Scan(&item.ID, &reviewID, &item.ProviderID, &item.StatDate, &item.EventType, &syncRunID, &upstream, &calculated, &autoAdopted, &final, &delta, &effectiveDelta, &item.Status, &item.DecisionType, &manual, &operatorID, &item.OperatedAt); err != nil {
		return item, err
	}
	item.ReviewID = nullIntPtr(reviewID)
	item.SyncRunID = nullIntPtr(syncRunID)
	item.UpstreamCost = nullFloatPtr(upstream)
	item.CalculatedCost = nullFloatPtr(calculated)
	item.AutoAdoptedCost = nullFloatPtr(autoAdopted)
	item.FinalCost = nullFloatPtr(final)
	item.CostDelta = nullFloatPtr(delta)
	item.EffectiveDelta = nullFloatPtr(effectiveDelta)
	item.ManualCost = nullFloatPtr(manual)
	item.OperatorID = nullIntPtr(operatorID)
	return item, nil
}

func nullFloatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	x := v.Float64
	return &x
}
func nullIntPtr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}
func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	x := v.Time
	return &x
}
func costReviewRound6(v float64) float64 { return math.Round(v*1e6) / 1e6 }
func costReviewValueOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
func nullableDelta(a, b *float64) *float64 {
	if a == nil || b == nil {
		return nil
	}
	v := costReviewRound6(*a - *b)
	return &v
}

func sameCostReviewValue(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return costReviewRound6(*a) == costReviewRound6(*b)
}
