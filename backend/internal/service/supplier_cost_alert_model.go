package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	ErrSupplierCostAlertConfigNotFound = infraerrors.NotFound("SUPPLIER_COST_ALERT_CONFIG_NOT_FOUND", "供应商成本超额预警配置不存在")
	ErrSupplierCostAlertEventNotFound  = infraerrors.NotFound("SUPPLIER_COST_ALERT_EVENT_NOT_FOUND", "供应商成本超额预警事件不存在")
	ErrSupplierCostAlertInvalid        = infraerrors.BadRequest("SUPPLIER_COST_ALERT_INVALID", "供应商成本超额预警参数无效")
)

const (
	SupplierCostAlertEventOverrun   = "cost_overrun"
	SupplierCostAlertEventRecovered = "cost_recovered"
	SupplierCostAlertEventActive    = "active"
	SupplierCostAlertEventResolved  = "resolved"
)

// SupplierCostAlertSettings 是成本超额预警的全局默认配置。
type SupplierCostAlertSettings struct {
	Amount decimal.Decimal `json:"amount"`
}

// SupplierCostAlertOverride 是单个供应商的成本超额预警覆盖配置。
type SupplierCostAlertOverride struct {
	ID         int64           `json:"id"`
	ProviderID int64           `json:"provider_id"`
	Enabled    bool            `json:"enabled"`
	Amount     decimal.Decimal `json:"amount"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type SupplierCostAlertOverrideInput struct {
	ProviderID int64  `json:"provider_id"`
	Enabled    bool   `json:"enabled"`
	Amount     string `json:"amount"`
}

// SupplierCostAlertEvent 记录一次上游成本高于本地成本的预警状态。
type SupplierCostAlertEvent struct {
	ID            int64           `json:"id"`
	ProviderID    int64           `json:"provider_id"`
	ProviderCode  string          `json:"provider_code"`
	ProviderName  string          `json:"provider_name"`
	EventType     string          `json:"event_type"`
	Status        string          `json:"status"`
	StatDate      time.Time       `json:"stat_date"`
	UpstreamCost  decimal.Decimal `json:"upstream_cost"`
	LocalCost     decimal.Decimal `json:"local_cost"`
	OverrunAmount decimal.Decimal `json:"overrun_amount"`
	Threshold     decimal.Decimal `json:"threshold"`
	ObservedAt    time.Time       `json:"observed_at"`
	ResolvedAt    *time.Time      `json:"resolved_at,omitempty"`
	LastSeenAt    time.Time       `json:"last_seen_at"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type SupplierCostAlertEventListParams struct {
	ProviderID int64
	EventType  string
	Status     string
	Page       int
	PageSize   int
}

type SupplierCostAlertEventListResult struct {
	Items    []SupplierCostAlertEvent `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

// SupplierCostAlertEvaluation 是同步服务传给预警处理器的一次有效观测。
type SupplierCostAlertEvaluation struct {
	ProviderID   int64
	ProviderCode string
	ProviderName string
	StatDate     time.Time
	UpstreamCost float64
	LocalCost    float64
	ObservedAt   time.Time
}

// SupplierCostAlertHandler 由供应商成本预警模块实现，供成本同步成功落库后调用。
type SupplierCostAlertHandler interface {
	Evaluate(ctx context.Context, evaluation SupplierCostAlertEvaluation) error
}

type SupplierCostAlertRepository interface {
	GetSettings(ctx context.Context) (*SupplierCostAlertSettings, error)
	UpdateSettings(ctx context.Context, amount decimal.Decimal) (*SupplierCostAlertSettings, error)
	GetOverrideByProvider(ctx context.Context, providerID int64) (*SupplierCostAlertOverride, error)
	ListOverrides(ctx context.Context) ([]SupplierCostAlertOverride, error)
	UpsertOverride(ctx context.Context, override *SupplierCostAlertOverride) (*SupplierCostAlertOverride, error)
	DeleteOverride(ctx context.Context, id int64) error
	GetActiveOverrunEvent(ctx context.Context, providerID int64) (*SupplierCostAlertEvent, error)
	CreateEvent(ctx context.Context, event *SupplierCostAlertEvent) error
	TouchActiveOverrunEvent(ctx context.Context, eventID int64, event SupplierCostAlertEvent) error
	ResolveActiveOverrunEvent(ctx context.Context, eventID int64, now time.Time) error
	ListEvents(ctx context.Context, params SupplierCostAlertEventListParams) (SupplierCostAlertEventListResult, error)
}
