package service

import (
	"context"
	"fmt"
	"math"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	CostReviewStatusPending      = "pending_review"
	CostReviewStatusApproved     = "approved"
	CostReviewStatusChanged      = "changed_after_approval"
	CostReviewDecisionNone       = "none"
	CostReviewDecisionUpstream   = "upstream"
	CostReviewDecisionCalculated = "calculated"
	CostReviewDecisionManual     = "manual"
)

var (
	ErrSupplierProviderCostReviewNotFound        = infraerrors.NotFound("SUPPLIER_PROVIDER_COST_REVIEW_NOT_FOUND", "供应商成本核对记录不存在")
	ErrSupplierProviderCostReviewVersionConflict = infraerrors.Conflict("SUPPLIER_PROVIDER_COST_REVIEW_VERSION_CONFLICT", "供应商成本核对记录已发生变化，请刷新后重试")
)

type SupplierProviderCostReview struct {
	ID              int64      `json:"id"`
	ProviderID      int64      `json:"provider_id"`
	ProviderName    string     `json:"provider_name"`
	StatDate        time.Time  `json:"stat_date"`
	UpstreamCost    *float64   `json:"upstream_cost"`
	CalculatedCost  *float64   `json:"calculated_cost"`
	LocalCost       *float64   `json:"local_cost"`
	AutoAdoptedCost *float64   `json:"auto_adopted_cost"`
	FinalCost       *float64   `json:"final_cost"`
	EffectiveCost   float64    `json:"effective_cost"`
	CostDelta       *float64   `json:"cost_delta"`
	EffectiveDelta  *float64   `json:"effective_delta"`
	Status          string     `json:"status"`
	DecisionType    string     `json:"decision_type"`
	ApprovedBy      *int64     `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	SyncCount       int        `json:"sync_count"`
	LastSyncRunID   *int64     `json:"last_sync_run_id,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type SupplierProviderCostReviewHistory struct {
	ID              int64     `json:"id"`
	ReviewID        *int64    `json:"review_id,omitempty"`
	ProviderID      int64     `json:"provider_id"`
	StatDate        time.Time `json:"stat_date"`
	EventType       string    `json:"event_type"`
	SyncRunID       *int64    `json:"sync_run_id,omitempty"`
	UpstreamCost    *float64  `json:"upstream_cost"`
	CalculatedCost  *float64  `json:"calculated_cost"`
	LocalCost       *float64  `json:"local_cost"`
	AutoAdoptedCost *float64  `json:"auto_adopted_cost"`
	FinalCost       *float64  `json:"final_cost"`
	CostDelta       *float64  `json:"cost_delta"`
	EffectiveDelta  *float64  `json:"effective_delta"`
	Status          string    `json:"status"`
	DecisionType    string    `json:"decision_type"`
	ManualCost      *float64  `json:"manual_cost,omitempty"`
	OperatorID      *int64    `json:"operator_id,omitempty"`
	OperatedAt      time.Time `json:"operated_at"`
}

type SupplierProviderCostReviewListParams struct {
	ProviderID                 int64
	Keyword                    string
	StartDate, EndDate, Status string
	Page, PageSize             int
}
type SupplierProviderCostReviewListResult struct {
	Items    []SupplierProviderCostReview `json:"items"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}
type SupplierProviderCostReviewApproveInput struct {
	DecisionType string   `json:"decision_type"`
	ManualCost   *float64 `json:"manual_cost"`
	Version      int64    `json:"version"`
	OperatorID   int64    `json:"-"`
}
type SupplierProviderCostReviewApproveItem struct {
	ID      int64 `json:"id"`
	Version int64 `json:"version"`
}
type SupplierProviderCostReviewBulkApproveInput struct {
	Items        []SupplierProviderCostReviewApproveItem `json:"items"`
	DecisionType string                                  `json:"decision_type"`
	ManualCost   *float64                                `json:"manual_cost"`
	OperatorID   int64                                   `json:"-"`
}
type SupplierProviderCostReviewSyncInput struct {
	ProviderID int64
	// CostSource 本次同步采用的成本来源模式，决定待审批记录的默认生效成本。
	CostSource      string
	StatDate        time.Time
	UpstreamCost    *float64
	CalculatedCost  *float64
	// LocalCost 是本地用量统计口径的成本，仅作参考展示，不参与生效成本决策。
	LocalCost       *float64
	AutoAdoptedCost *float64
	EffectiveCost   float64
	SyncRunID       *int64
	SyncedAt        time.Time
}

type SupplierProviderCostReviewRepository interface {
	List(context.Context, SupplierProviderCostReviewListParams) (SupplierProviderCostReviewListResult, error)
	History(context.Context, int64) ([]SupplierProviderCostReviewHistory, error)
	Approve(context.Context, int64, SupplierProviderCostReviewApproveInput) (*SupplierProviderCostReview, error)
	ApproveMany(context.Context, SupplierProviderCostReviewBulkApproveInput) ([]SupplierProviderCostReview, error)
	Sync(context.Context, SupplierProviderCostReviewSyncInput) (*SupplierProviderCostReview, error)
}

func NewSupplierProviderCostReviewService(repo SupplierProviderCostReviewRepository) *SupplierProviderCostReviewService {
	return &SupplierProviderCostReviewService{repo: repo}
}

type SupplierProviderCostReviewService struct {
	repo SupplierProviderCostReviewRepository
}

func (s *SupplierProviderCostReviewService) List(ctx context.Context, p SupplierProviderCostReviewListParams) (SupplierProviderCostReviewListResult, error) {
	if s == nil || s.repo == nil {
		return SupplierProviderCostReviewListResult{}, fmt.Errorf("成本核对服务未配置")
	}
	return s.repo.List(ctx, p)
}
func (s *SupplierProviderCostReviewService) History(ctx context.Context, id int64) ([]SupplierProviderCostReviewHistory, error) {
	if id <= 0 {
		return nil, fmt.Errorf("核对记录 ID 无效")
	}
	return s.repo.History(ctx, id)
}
func (s *SupplierProviderCostReviewService) Sync(ctx context.Context, input SupplierProviderCostReviewSyncInput) (*SupplierProviderCostReview, error) {
	if input.ProviderID <= 0 {
		return nil, fmt.Errorf("供应商 ID 无效")
	}
	if input.StatDate.IsZero() {
		return nil, fmt.Errorf("统计日期无效")
	}
	if input.SyncedAt.IsZero() {
		input.SyncedAt = time.Now().UTC()
	}
	return s.repo.Sync(ctx, input)
}
func (s *SupplierProviderCostReviewService) Approve(ctx context.Context, id int64, input SupplierProviderCostReviewApproveInput) (*SupplierProviderCostReview, error) {
	if id <= 0 {
		return nil, fmt.Errorf("核对记录 ID 无效")
	}
	if err := validateCostReviewApproveInput(input.Version, input.DecisionType, input.ManualCost); err != nil {
		return nil, err
	}
	review, err := s.repo.Approve(ctx, id, input)
	if err != nil {
		return nil, err
	}
	// 审批改写了 daily_stats 的生效成本与提示，趋势缓存必须失效，否则成本分析最多 30 秒还是旧值。
	invalidateSupplierCostTrendCache()
	return review, nil
}

func (s *SupplierProviderCostReviewService) ApproveMany(ctx context.Context, input SupplierProviderCostReviewBulkApproveInput) ([]SupplierProviderCostReview, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("成本核对服务未配置")
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("至少选择一条成本核对记录")
	}
	seen := make(map[int64]struct{}, len(input.Items))
	for _, item := range input.Items {
		if item.ID <= 0 {
			return nil, fmt.Errorf("核对记录 ID 无效")
		}
		if item.Version <= 0 {
			return nil, fmt.Errorf("核对记录版本无效")
		}
		if _, exists := seen[item.ID]; exists {
			return nil, fmt.Errorf("批量审批不允许重复核对记录")
		}
		seen[item.ID] = struct{}{}
	}
	if err := validateCostReviewApproveInput(1, input.DecisionType, input.ManualCost); err != nil {
		return nil, err
	}
	reviews, err := s.repo.ApproveMany(ctx, input)
	if err != nil {
		return nil, err
	}
	invalidateSupplierCostTrendCache()
	return reviews, nil
}

func validateCostReviewApproveInput(version int64, decisionType string, manualCost *float64) error {
	if version <= 0 {
		return fmt.Errorf("核对记录版本无效")
	}
	switch decisionType {
	case CostReviewDecisionUpstream, CostReviewDecisionCalculated:
		if manualCost != nil {
			return fmt.Errorf("非手动审批不应填写手动成本")
		}
	case CostReviewDecisionManual:
		if manualCost == nil || math.IsNaN(*manualCost) || math.IsInf(*manualCost, 0) || *manualCost < 0 || math.Round(*manualCost*1e6) != *manualCost*1e6 {
			return fmt.Errorf("手动成本必须是非负且不超过 6 位小数的金额")
		}
	default:
		return fmt.Errorf("审批方式无效")
	}
	return nil
}
