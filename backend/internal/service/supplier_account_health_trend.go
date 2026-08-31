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
	// 每个趋势固定按 96 个时间桶返回，保证不同时间范围下的图表密度一致。
	SupplierAccountHealthTrendBucketCount = 96
	// 保留旧常量名，避免模块内已有调用方因趋势点上限改为时间桶而失效。
	SupplierAccountHealthBatchPointLimit = SupplierAccountHealthTrendBucketCount

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
	GetTrend(ctx context.Context, accountID int64, since, until time.Time) (SupplierAccountHealthTrendResult, error)
	GetTrends(ctx context.Context, accountIDs []int64, since, until time.Time, pointLimit int) (map[int64]SupplierAccountHealthTrendResult, error)
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
	// UpstreamRateMultiplier 是上游账号自身的倍率，EffectiveRateMultiplier 额外乘上
	// 供应商的倍率缩放，与倍率守护、供应商总览的口径一致。
	UpstreamRateMultiplier  float64 `json:"upstream_rate_multiplier"`
	EffectiveRateMultiplier float64 `json:"effective_rate_multiplier"`
	GuardEnabled            bool    `json:"guard_enabled"`
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
	CheckedAt       time.Time  `json:"checked_at"`
	BucketEndAt     *time.Time `json:"bucket_end_at,omitempty"`
	LatestCheckedAt *time.Time `json:"latest_checked_at,omitempty"`
	Status          string     `json:"status"`
	LatencyMs       *int64     `json:"latency_ms,omitempty"`
	LatencyLimitMs  int64      `json:"latency_limit_ms"`
	SampleCount     int        `json:"sample_count"`
	HealthyCount    int        `json:"healthy_count"`
	SlowCount       int        `json:"slow_count"`
	FailedCount     int        `json:"failed_count"`
	Reason          string     `json:"reason,omitempty"`
	Action          string     `json:"action,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
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
	since, until := supplierAccountHealthTrendWindow(normalizedRange)
	result, err := s.repository.GetTrend(ctx, accountID, since, until)
	if err != nil {
		return SupplierAccountHealthTrendResult{}, err
	}
	result.Range = normalizedRange
	rawPoints := result.Points
	result.Points = aggregateSupplierAccountHealthTrendPoints(rawPoints, since, supplierAccountHealthTrendDuration(normalizedRange))
	result.Latest = latestSupplierAccountHealthPoint(rawPoints)
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
	since, until := supplierAccountHealthTrendWindow(normalizedRange)
	trendMap, err := s.repository.GetTrends(ctx, uniqueIDs, since, until, SupplierAccountHealthBatchPointLimit)
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
		if _, exists := trendMap[accountID]; exists {
			rawPoints := result.Points
			result.Points = aggregateSupplierAccountHealthTrendPoints(rawPoints, since, supplierAccountHealthTrendDuration(normalizedRange))
			result.Latest = latestSupplierAccountHealthPoint(rawPoints)
		} else if result.Points == nil {
			result.Points = []SupplierAccountHealthPoint{}
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

func supplierAccountHealthTrendWindow(rangeValue string) (time.Time, time.Time) {
	until := time.Now()
	return until.Add(-supplierAccountHealthTrendDuration(rangeValue)), until
}

func supplierAccountHealthTrendDuration(rangeValue string) time.Duration {
	switch rangeValue {
	case SupplierAccountHealthRange7d:
		return 7 * 24 * time.Hour
	case SupplierAccountHealthRange30d:
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// aggregateSupplierAccountHealthTrendPoints 将范围内的原始检测记录压缩为固定数量的时间桶。
// 空桶保留为 unchecked，健康率只使用有样本的桶内检测记录计算。
func aggregateSupplierAccountHealthTrendPoints(points []SupplierAccountHealthPoint, since time.Time, duration time.Duration) []SupplierAccountHealthPoint {
	bucketCount := SupplierAccountHealthTrendBucketCount
	bucketDuration := duration / time.Duration(bucketCount)
	if bucketDuration <= 0 {
		return []SupplierAccountHealthPoint{}
	}

	aggregated := make([]SupplierAccountHealthPoint, bucketCount)
	representativeAt := make([]time.Time, bucketCount)
	latestSampleAt := make([]time.Time, bucketCount)
	for index := range aggregated {
		bucketStart := since.Add(time.Duration(index) * bucketDuration)
		bucketEnd := bucketStart.Add(bucketDuration)
		aggregated[index] = SupplierAccountHealthPoint{
			CheckedAt:   bucketStart,
			BucketEndAt: &bucketEnd,
			Status:      SupplierAccountHealthStatusUnchecked,
		}
	}

	end := since.Add(duration)
	for _, point := range points {
		if point.CheckedAt.Before(since) || !point.CheckedAt.Before(end) {
			continue
		}
		index := int(point.CheckedAt.Sub(since) / bucketDuration)
		if index < 0 || index >= bucketCount {
			continue
		}

		bucket := &aggregated[index]
		bucket.SampleCount++
		switch point.Status {
		case "healthy":
			bucket.HealthyCount++
		case "slow":
			bucket.SlowCount++
		case "failed":
			bucket.FailedCount++
		}

		if latestSampleAt[index].IsZero() || point.CheckedAt.After(latestSampleAt[index]) {
			latestSampleAt[index] = point.CheckedAt
			latestCheckedAt := point.CheckedAt
			bucket.LatestCheckedAt = &latestCheckedAt
			bucket.LatencyLimitMs = point.LatencyLimitMs
		}

		if supplierAccountHealthStatusRank(point.Status) > supplierAccountHealthStatusRank(bucket.Status) ||
			(supplierAccountHealthStatusRank(point.Status) == supplierAccountHealthStatusRank(bucket.Status) &&
				(representativeAt[index].IsZero() || point.CheckedAt.After(representativeAt[index]))) {
			representativeAt[index] = point.CheckedAt
			bucket.Status = point.Status
			bucket.Reason = point.Reason
			bucket.Action = point.Action
			bucket.ErrorMessage = point.ErrorMessage
		}

		if point.Status != "failed" && point.LatencyMs != nil && *point.LatencyMs > 0 &&
			(bucket.LatencyMs == nil || *point.LatencyMs > *bucket.LatencyMs) {
			latency := *point.LatencyMs
			bucket.LatencyMs = &latency
		}
	}

	for index := range aggregated {
		if aggregated[index].FailedCount > 0 {
			aggregated[index].LatencyMs = nil
		}
	}
	return aggregated
}

func supplierAccountHealthStatusRank(status string) int {
	switch status {
	case "failed":
		return 3
	case "slow":
		return 2
	case "healthy":
		return 1
	default:
		return 0
	}
}

func latestSupplierAccountHealthPoint(points []SupplierAccountHealthPoint) *SupplierAccountHealthPoint {
	if len(points) == 0 {
		return nil
	}
	latestIndex := 0
	for index := 1; index < len(points); index++ {
		if !points[index].CheckedAt.Before(points[latestIndex].CheckedAt) {
			latestIndex = index
		}
	}
	latest := points[latestIndex]
	latest.SampleCount = 1
	switch latest.Status {
	case SupplierAccountHealthGuardStatusHealthy:
		latest.HealthyCount = 1
	case SupplierAccountHealthGuardStatusSlow:
		latest.SlowCount = 1
	case SupplierAccountHealthGuardStatusFailed:
		latest.FailedCount = 1
	}
	return &latest
}
