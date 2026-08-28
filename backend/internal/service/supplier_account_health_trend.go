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

	// 批量趋势接口的防护上限，避免一次请求拖垮数据库或返回过大的响应。
	SupplierAccountHealthBatchMaxAccounts = 100
	SupplierAccountHealthBatchPointLimit  = 50

	// SupplierAccountHealthStatusUnchecked 用于筛选尚无健康检测记录的账号。
	SupplierAccountHealthStatusUnchecked = "unchecked"
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
	GetSummary(ctx context.Context, params SupplierAccountHealthAccountListParams) (SupplierAccountHealthSummary, error)
	GetTrend(ctx context.Context, accountID int64, since time.Time) (SupplierAccountHealthTrendResult, error)
	GetTrends(ctx context.Context, accountIDs []int64, since time.Time, pointLimit int) (map[int64]SupplierAccountHealthTrendResult, error)
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
	RateMultiplier      float64    `json:"rate_multiplier"`
	GuardEnabled        bool       `json:"guard_enabled"`
}

type SupplierAccountHealthAccountListResult struct {
	Items    []SupplierAccountHealthAccount `json:"items"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"page_size"`
}

// SupplierAccountHealthSummary 按最近一次检测状态汇总账号数量，供列表页概览卡展示。
type SupplierAccountHealthSummary struct {
	Total     int64 `json:"total"`
	Healthy   int64 `json:"healthy"`
	Slow      int64 `json:"slow"`
	Failed    int64 `json:"failed"`
	Unchecked int64 `json:"unchecked"`
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

func (s *SupplierAccountHealthTrendService) GetSummary(ctx context.Context, params SupplierAccountHealthAccountListParams) (SupplierAccountHealthSummary, error) {
	if s == nil || s.repository == nil {
		return SupplierAccountHealthSummary{}, errors.New("账号健康历史服务未初始化")
	}
	return s.repository.GetSummary(ctx, params)
}

func (s *SupplierAccountHealthTrendService) GetTrend(ctx context.Context, accountID int64, rangeValue string) (SupplierAccountHealthTrendResult, error) {
	if s == nil || s.repository == nil {
		return SupplierAccountHealthTrendResult{}, errors.New("账号健康历史服务未初始化")
	}
	normalizedRange, err := normalizeSupplierAccountHealthTrendRange(rangeValue)
	if err != nil {
		return SupplierAccountHealthTrendResult{}, errors.New("健康趋势范围无效")
	}
	if err := s.repository.ValidateAccount(ctx, accountID); err != nil {
		return SupplierAccountHealthTrendResult{}, err
	}
	result, err := s.repository.GetTrend(ctx, accountID, supplierAccountHealthTrendSince(normalizedRange))
	if err != nil {
		return SupplierAccountHealthTrendResult{}, err
	}
	result.Range = normalizedRange
	if len(result.Points) > 0 {
		result.Latest = &result.Points[len(result.Points)-1]
	}
	return result, nil
}

// GetTrends 一次查询多个账号的健康趋势，供列表页批量展示迷你趋势使用。
func (s *SupplierAccountHealthTrendService) GetTrends(ctx context.Context, accountIDs []int64, rangeValue string) ([]SupplierAccountHealthTrendResult, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("账号健康历史服务未初始化")
	}
	normalizedRange, err := normalizeSupplierAccountHealthTrendRange(rangeValue)
	if err != nil {
		return nil, errors.New("健康趋势范围无效")
	}
	uniqueIDs := make([]int64, 0, len(accountIDs))
	seen := make(map[int64]struct{}, len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			continue
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		uniqueIDs = append(uniqueIDs, accountID)
	}
	if len(uniqueIDs) == 0 {
		return []SupplierAccountHealthTrendResult{}, nil
	}
	if len(uniqueIDs) > SupplierAccountHealthBatchMaxAccounts {
		return nil, errors.New("一次最多查询 100 个账号的健康趋势")
	}
	trendMap, err := s.repository.GetTrends(ctx, uniqueIDs, supplierAccountHealthTrendSince(normalizedRange), SupplierAccountHealthBatchPointLimit)
	if err != nil {
		return nil, err
	}
	results := make([]SupplierAccountHealthTrendResult, 0, len(uniqueIDs))
	for _, accountID := range uniqueIDs {
		result, exists := trendMap[accountID]
		if !exists {
			result = SupplierAccountHealthTrendResult{AccountID: accountID, Points: []SupplierAccountHealthPoint{}}
		}
		result.AccountID = accountID
		result.Range = normalizedRange
		if result.Points == nil {
			result.Points = []SupplierAccountHealthPoint{}
		}
		if len(result.Points) > 0 {
			result.Latest = &result.Points[len(result.Points)-1]
		}
		results = append(results, result)
	}
	return results, nil
}

func normalizeSupplierAccountHealthTrendRange(rangeValue string) (string, error) {
	switch rangeValue {
	case "", SupplierAccountHealthRange24h:
		return SupplierAccountHealthRange24h, nil
	case SupplierAccountHealthRange7d, SupplierAccountHealthRange30d:
		return rangeValue, nil
	default:
		return "", errors.New("健康趋势范围无效")
	}
}

func supplierAccountHealthTrendSince(rangeValue string) time.Time {
	days := map[string]int{SupplierAccountHealthRange24h: 1, SupplierAccountHealthRange7d: 7, SupplierAccountHealthRange30d: 30}[rangeValue]
	return time.Now().Add(-time.Duration(days) * 24 * time.Hour)
}
