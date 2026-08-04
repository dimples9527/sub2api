package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	ErrSupplierBalanceAlertConfigNotFound = infraerrors.NotFound("SUPPLIER_BALANCE_ALERT_CONFIG_NOT_FOUND", "供应商余额预警配置不存在")
	ErrSupplierBalanceAlertEventNotFound  = infraerrors.NotFound("SUPPLIER_BALANCE_ALERT_EVENT_NOT_FOUND", "供应商余额预警事件不存在")
	ErrSupplierBalanceAlertInvalid        = infraerrors.BadRequest("SUPPLIER_BALANCE_ALERT_INVALID", "供应商余额预警参数无效")
	ErrSupplierBalanceAlertScanBusy       = infraerrors.Conflict("SUPPLIER_BALANCE_ALERT_SCAN_BUSY", "供应商余额预警扫描正在执行")
)

const (
	SupplierBalanceAlertEventLow        = "balance_low"
	SupplierBalanceAlertEventRecovered  = "balance_recovered"
	SupplierBalanceAlertEventActive     = "active"
	SupplierBalanceAlertEventResolved   = "resolved"
	SupplierBalanceAlertScanStatusNever = "never"
	SupplierBalanceAlertScanStatusOK    = "ok"
	SupplierBalanceAlertScanStatusSkip  = "skipped"
	SupplierBalanceAlertScanStatusError = "error"
)

const (
	SupplierBalanceAlertDefaultInterval = 15 * time.Minute
	SupplierBalanceAlertDefaultCooldown = time.Hour
)

type SupplierBalanceProvider struct {
	ID                 int64
	Code               string
	Name               string
	ProviderType       string
	BaseURL            string
	LoginURL           string
	APIKeysURL         string
	GroupsURL          string
	AvailableGroupsURL string
	BalanceURL         string
	UsageCostURL       string
	Username           string
	Email              string
	Enabled            bool
	TurnstileEnabled   bool
	CurrentBalance     float64
	LastSyncAt         *time.Time
}

type SupplierBalanceSource interface {
	ListEnabledProviders(ctx context.Context) ([]SupplierBalanceProvider, error)
	FetchBalance(ctx context.Context, provider SupplierBalanceProvider) (decimal.Decimal, error)
}

type SupplierBalanceAlertConfig struct {
	ID              int64            `json:"id"`
	ProviderID      int64            `json:"provider_id"`
	ProviderCode    string           `json:"provider_code"`
	ProviderName    string           `json:"provider_name"`
	ProviderType    string           `json:"provider_type"`
	ProviderEnabled bool             `json:"provider_enabled"`
	Enabled         bool             `json:"enabled"`
	Threshold       decimal.Decimal  `json:"threshold"`
	Cooldown        time.Duration    `json:"-"`
	CooldownSeconds int              `json:"cooldown_seconds"`
	LastScanAt      *time.Time       `json:"last_scan_at,omitempty"`
	LastBalance     *decimal.Decimal `json:"last_balance,omitempty"`
	LastScanStatus  string           `json:"last_scan_status"`
	LastScanError   string           `json:"last_scan_error,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type SupplierBalanceAlertConfigInput struct {
	Enabled         bool   `json:"enabled"`
	Threshold       string `json:"threshold"`
	CooldownSeconds int    `json:"cooldown_seconds"`
}

type SupplierBalanceAlertEvent struct {
	ID           int64           `json:"id"`
	ProviderID   int64           `json:"provider_id"`
	ProviderCode string          `json:"provider_code"`
	ProviderName string          `json:"provider_name"`
	EventType    string          `json:"event_type"`
	Status       string          `json:"status"`
	Balance      decimal.Decimal `json:"balance"`
	Threshold    decimal.Decimal `json:"threshold"`
	Cooldown     time.Duration   `json:"-"`
	ObservedAt   time.Time       `json:"observed_at"`
	ResolvedAt   *time.Time      `json:"resolved_at,omitempty"`
	LastSeenAt   time.Time       `json:"last_seen_at"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SupplierBalanceAlertEventListParams struct {
	ProviderID int64
	EventType  string
	Status     string
	Page       int
	PageSize   int
}

type SupplierBalanceAlertEventListResult struct {
	Items    []SupplierBalanceAlertEvent `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

type SupplierBalanceAlertScanProviderResult struct {
	ProviderID   int64            `json:"provider_id"`
	ProviderName string           `json:"provider_name"`
	Status       string           `json:"status"`
	Balance      *decimal.Decimal `json:"balance,omitempty"`
	EventType    string           `json:"event_type,omitempty"`
	Message      string           `json:"message,omitempty"`
}

type SupplierBalanceAlertScanResult struct {
	StartedAt  time.Time                                `json:"started_at"`
	FinishedAt time.Time                                `json:"finished_at"`
	Checked    int                                      `json:"checked"`
	Skipped    int                                      `json:"skipped"`
	Triggered  int                                      `json:"triggered"`
	Recovered  int                                      `json:"recovered"`
	Failed     int                                      `json:"failed"`
	Providers  []SupplierBalanceAlertScanProviderResult `json:"providers"`
}

type SupplierBalanceAlertRepository interface {
	ListConfigs(ctx context.Context, providerID int64) ([]SupplierBalanceAlertConfig, error)
	GetConfig(ctx context.Context, providerID int64) (*SupplierBalanceAlertConfig, error)
	UpsertConfig(ctx context.Context, providerID int64, enabled bool, threshold decimal.Decimal, cooldownSeconds int) (*SupplierBalanceAlertConfig, error)
	UpdateScanState(ctx context.Context, providerID int64, now time.Time, balance *decimal.Decimal, status, message string) error
	GetActiveLowEvent(ctx context.Context, providerID int64) (*SupplierBalanceAlertEvent, error)
	CreateEvent(ctx context.Context, event *SupplierBalanceAlertEvent) error
	TouchActiveLowEvent(ctx context.Context, eventID int64, balance decimal.Decimal, now time.Time) error
	ResolveActiveLowEvent(ctx context.Context, eventID int64, now time.Time, balance decimal.Decimal) error
	ListEvents(ctx context.Context, params SupplierBalanceAlertEventListParams) (SupplierBalanceAlertEventListResult, error)
}

type SupplierBalanceAlertDispatcher interface {
	Dispatch(ctx context.Context, event SupplierBalanceAlertEvent) error
}
