package service

import (
	"context"
	"errors"
	"time"
)

const (
	SupplierAccountHealthRange24h = "24h"
	SupplierAccountHealthRange7d  = "7d"
	SupplierAccountHealthRange30d = "30d"
)

type SupplierAccountHealthHistoryRecord struct {
	LocalAccountID     int64
	LocalAccountName   string
	ProviderID         int64
	ProviderName       string
	Platform           string
	CheckedAt          time.Time
	StartedAt          time.Time
	FinishedAt         time.Time
	Status             string
	LatencyMs          *int64
	LatencyLimitMs     int64
	ModelID            string
	SchedulableBefore  bool
	SchedulableAfter   bool
	Action             string
	ConsecutiveFailed  int
	ConsecutiveSlow    int
	ConsecutiveHealthy int
	Reason             string
	ErrorMessage       string
}

type SupplierAccountHealthHistoryRecorder interface {
	Save(ctx context.Context, record SupplierAccountHealthHistoryRecord) error
}

type SupplierAccountHealthHistoryRepository interface {
	SupplierAccountHealthHistoryRecorder
	ValidateAccount(ctx context.Context, accountID int64) error
	ListAccounts(ctx context.Context, params SupplierAccountHealthAccountListParams) (SupplierAccountHealthAccountListResult, error)
	GetTrend(ctx context.Context, accountID int64, since time.Time) (SupplierAccountHealthTrendResult, error)
	DeleteBefore(ctx context.Context, before time.Time, batchSize int) (int, error)
}

type SupplierAccountHealthAccountListParams struct {
	ProviderID   int64
	Platform     string
	Search       string
	HealthStatus string
	Page         int
	PageSize     int
}

type SupplierAccountHealthAccount struct {
	LocalAccountID      int64      `json:"local_account_id"`
	LocalAccountName    string     `json:"local_account_name"`
	ProviderID          int64      `json:"provider_id"`
	ProviderName        string     `json:"provider_name"`
	Platform            string     `json:"platform"`
	Schedulable         bool       `json:"schedulable"`
	Status              string     `json:"status,omitempty"`
	CheckedAt           *time.Time `json:"checked_at,omitempty"`
	LatencyMs           *int64     `json:"latency_ms,omitempty"`
	LatencyLimitMs      int64      `json:"latency_limit_ms"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	GuardEnabled        bool       `json:"guard_enabled"`
}

type SupplierAccountHealthAccountListResult struct {
	Items    []SupplierAccountHealthAccount `json:"items"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"page_size"`
}

type SupplierAccountHealthPoint struct {
	CheckedAt      time.Time `json:"checked_at"`
	Status         string    `json:"status"`
	LatencyMs      *int64    `json:"latency_ms,omitempty"`
	LatencyLimitMs int64     `json:"latency_limit_ms"`
	Reason         string    `json:"reason,omitempty"`
	Action         string    `json:"action,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

type SupplierAccountHealthTrendResult struct {
	AccountID int64                        `json:"account_id"`
	Range     string                       `json:"range"`
	Points    []SupplierAccountHealthPoint `json:"points"`
	Latest    *SupplierAccountHealthPoint  `json:"latest,omitempty"`
}

type SupplierAccountHealthTrendService struct {
	repository SupplierAccountHealthHistoryRepository
	recorder   SupplierAccountHealthHistoryRecorder
}

func NewSupplierAccountHealthTrendService(repository SupplierAccountHealthHistoryRepository, recorder SupplierAccountHealthHistoryRecorder) *SupplierAccountHealthTrendService {
	return &SupplierAccountHealthTrendService{repository: repository, recorder: recorder}
}

func (s *SupplierAccountHealthTrendService) RecordRunItem(ctx context.Context, item SupplierAccountHealthGuardRunItem) error {
	if s == nil || s.recorder == nil || item.LocalAccountID <= 0 {
		return errors.New("账号健康历史服务未初始化")
	}
	if item.Status != SupplierAccountHealthGuardStatusHealthy && item.Status != SupplierAccountHealthGuardStatusSlow && item.Status != SupplierAccountHealthGuardStatusFailed {
		return nil
	}
	var source SupplierAccountHealthGuardSource
	if len(item.Sources) > 0 {
		source = item.Sources[0]
	}
	var latency *int64
	if item.Status != SupplierAccountHealthGuardStatusFailed {
		value := item.LatencyMs
		latency = &value
	}
	return s.recorder.Save(ctx, SupplierAccountHealthHistoryRecord{
		LocalAccountID: item.LocalAccountID, LocalAccountName: item.LocalAccountName,
		ProviderID: source.ProviderID, ProviderName: source.ProviderName, Platform: item.Platform,
		CheckedAt: item.FinishedAt, StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
		Status: item.Status, LatencyMs: latency, LatencyLimitMs: item.LatencyLimitMs, ModelID: item.ModelID,
		SchedulableBefore: item.SchedulableBefore, SchedulableAfter: item.SchedulableAfter, Action: item.Action,
		ConsecutiveFailed: item.ConsecutiveFailed, ConsecutiveSlow: item.ConsecutiveSlow,
		ConsecutiveHealthy: item.ConsecutiveHealthy, Reason: item.Reason, ErrorMessage: item.ErrorMessage,
	})
}

func (s *SupplierAccountHealthTrendService) ListAccounts(ctx context.Context, params SupplierAccountHealthAccountListParams) (SupplierAccountHealthAccountListResult, error) {
	if s == nil || s.repository == nil {
		return SupplierAccountHealthAccountListResult{}, errors.New("账号健康历史服务未初始化")
	}
	return s.repository.ListAccounts(ctx, params)
}

func (s *SupplierAccountHealthTrendService) GetTrend(ctx context.Context, accountID int64, rangeValue string) (SupplierAccountHealthTrendResult, error) {
	if s == nil || s.repository == nil {
		return SupplierAccountHealthTrendResult{}, errors.New("账号健康历史服务未初始化")
	}
	switch rangeValue {
	case "", SupplierAccountHealthRange24h:
		rangeValue = SupplierAccountHealthRange24h
	case SupplierAccountHealthRange7d, SupplierAccountHealthRange30d:
	default:
		return SupplierAccountHealthTrendResult{}, errors.New("健康趋势范围无效")
	}
	if err := s.repository.ValidateAccount(ctx, accountID); err != nil {
		return SupplierAccountHealthTrendResult{}, err
	}
	days := map[string]int{SupplierAccountHealthRange24h: 1, SupplierAccountHealthRange7d: 7, SupplierAccountHealthRange30d: 30}[rangeValue]
	result, err := s.repository.GetTrend(ctx, accountID, time.Now().Add(-time.Duration(days)*24*time.Hour))
	if err != nil {
		return SupplierAccountHealthTrendResult{}, err
	}
	result.Range = rangeValue
	if len(result.Points) > 0 {
		result.Latest = &result.Points[len(result.Points)-1]
	}
	return result, nil
}
