package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// 模型监控分组健康趋势的错误分类。
const (
	ModelMonitorGroupHealthErrorUpstreamRateLimit = "upstream_rate_limit"
	ModelMonitorGroupHealthErrorUpstream          = "upstream_error"
	ModelMonitorGroupHealthErrorNetwork           = "network_timeout"
	ModelMonitorGroupHealthErrorAccountAuth       = "account_auth"
	ModelMonitorGroupHealthErrorRouting           = "routing"
	ModelMonitorGroupHealthErrorBusinessLimited   = "business_limited"
	ModelMonitorGroupHealthErrorClientRequest     = "client_request"
	ModelMonitorGroupHealthErrorOther             = "other"
)

// 模型监控分组健康状态。
const (
	ModelMonitorGroupHealthStatusHealthy   = "healthy"
	ModelMonitorGroupHealthStatusWarning   = "warning"
	ModelMonitorGroupHealthStatusCritical  = "critical"
	ModelMonitorGroupHealthStatusLowSample = "low_sample"
	ModelMonitorGroupHealthStatusNoData    = "no_data"
)

// 支持的查询时间范围。
const (
	ModelMonitorGroupHealthRange1H  = "1h"
	ModelMonitorGroupHealthRange24H = "24h"
	ModelMonitorGroupHealthRange7D  = "7d"
	ModelMonitorGroupHealthRange30D = "30d"
)

// 单次查询允许的最大分组数量。
const ModelMonitorGroupHealthMaxGroupIDs = 200

// 低样本阈值：小于该样本量时不直接判定健康状态。
const ModelMonitorGroupHealthMinSampleCount = 5

// 健康状态阈值，单位为百分比。
const (
	ModelMonitorGroupHealthHealthyThreshold = 98.0
	ModelMonitorGroupHealthWarningThreshold = 90.0
)

// ModelMonitorGroupHealthQuery 分组健康趋势查询参数。
type ModelMonitorGroupHealthQuery struct {
	Range    string
	GroupIDs []int64
	Platform string
	Now      time.Time
}

// ModelMonitorGroupHealthPoint 趋势采样点。
type ModelMonitorGroupHealthPoint struct {
	Time                 string  `json:"time"`
	RequestCount         int64   `json:"request_count"`
	SuccessCount         int64   `json:"success_count"`
	ErrorCount           int64   `json:"error_count"`
	ServiceErrorCount    int64   `json:"service_error_count"`
	BusinessLimitedCount int64   `json:"business_limited_count"`
	SuccessRate          float64 `json:"success_rate"`
	ServiceSuccessRate   float64 `json:"service_success_rate"`
	AvgLatencyMS         float64 `json:"avg_latency_ms"`
	P95LatencyMS         float64 `json:"p95_latency_ms"`
}

// ModelMonitorGroupHealthErrorItem 错误分类统计。
type ModelMonitorGroupHealthErrorItem struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// ModelMonitorGroupHealth 单个分组的健康指标。
type ModelMonitorGroupHealth struct {
	GroupID              int64                              `json:"group_id"`
	GroupName            string                             `json:"group_name"`
	Platform             string                             `json:"platform"`
	EffectivePlatform    string                             `json:"effective_platform"`
	RequestCount         int64                              `json:"request_count"`
	SuccessCount         int64                              `json:"success_count"`
	ErrorCount           int64                              `json:"error_count"`
	BusinessLimitedCount int64                              `json:"business_limited_count"`
	ServiceErrorCount    int64                              `json:"service_error_count"`
	SuccessRate          float64                            `json:"success_rate"`
	ServiceSuccessRate   float64                            `json:"service_success_rate"`
	ErrorRate            float64                            `json:"error_rate"`
	AvgLatencyMS         float64                            `json:"avg_latency_ms"`
	P95LatencyMS         float64                            `json:"p95_latency_ms"`
	P95FirstTokenMS      float64                            `json:"p95_first_token_ms"`
	Status               string                             `json:"status"`
	LastRequestAt        *time.Time                         `json:"last_request_at"`
	Trend                []ModelMonitorGroupHealthPoint     `json:"trend"`
	TopErrors            []ModelMonitorGroupHealthErrorItem `json:"top_errors"`
}

// ModelMonitorGroupHealthBucket 聚合查询得到的原始时间桶。
type ModelMonitorGroupHealthBucket struct {
	GroupID              int64
	BucketStart          time.Time
	SuccessCount         int64
	LatencySampleCount   int64
	AvgLatencyMS         float64
	P95LatencyMS         float64
	P95FirstTokenMS      float64
	LastRequestAt        *time.Time
	ErrorCount           int64
	BusinessLimitedCount int64
}

// ModelMonitorGroupHealthGroup 分组基础信息。
type ModelMonitorGroupHealthGroup struct {
	ID                int64
	Name              string
	Platform          string
	EffectivePlatform string
}

// ModelMonitorGroupHealthErrorCount 错误分类计数。
type ModelMonitorGroupHealthErrorCount struct {
	GroupID  int64
	Category string
	Count    int64
}

// ModelMonitorGroupHealthRepository 提供分组健康趋势所需的聚合查询。
type ModelMonitorGroupHealthRepository interface {
	ListGroups(ctx context.Context, groupIDs []int64, platform string) ([]ModelMonitorGroupHealthGroup, error)
	ListUsageBuckets(ctx context.Context, startTime, endTime time.Time, bucketInterval time.Duration, groupIDs []int64, platform string) ([]ModelMonitorGroupHealthBucket, error)
	ListErrorBuckets(ctx context.Context, startTime, endTime time.Time, bucketInterval time.Duration, groupIDs []int64, platform string) ([]ModelMonitorGroupHealthBucket, error)
	ListErrorCategories(ctx context.Context, startTime, endTime time.Time, groupIDs []int64, platform string) ([]ModelMonitorGroupHealthErrorCount, error)
}

// ModelMonitorGroupHealthService 计算模型监控分组健康趋势。
type ModelMonitorGroupHealthService struct {
	repo ModelMonitorGroupHealthRepository
}

// NewModelMonitorGroupHealthService 创建分组健康趋势服务。
func NewModelMonitorGroupHealthService(repo ModelMonitorGroupHealthRepository) *ModelMonitorGroupHealthService {
	return &ModelMonitorGroupHealthService{repo: repo}
}

// Get 返回按分组聚合的健康趋势。
func (s *ModelMonitorGroupHealthService) Get(ctx context.Context, query ModelMonitorGroupHealthQuery) ([]ModelMonitorGroupHealth, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("model monitor group health repository is not initialized")
	}
	normalized, err := NormalizeModelMonitorGroupHealthQuery(query)
	if err != nil {
		return nil, err
	}
	now := normalized.Now
	startTime, endTime, interval := modelMonitorGroupHealthWindow(now, normalized.Range)

	groups, err := s.repo.ListGroups(ctx, normalized.GroupIDs, normalized.Platform)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []ModelMonitorGroupHealth{}, nil
	}

	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}

	usageBuckets, err := s.repo.ListUsageBuckets(ctx, startTime, endTime, interval, groupIDs, normalized.Platform)
	if err != nil {
		return nil, err
	}
	errorBuckets, err := s.repo.ListErrorBuckets(ctx, startTime, endTime, interval, groupIDs, normalized.Platform)
	if err != nil {
		return nil, err
	}
	errorCategories, err := s.repo.ListErrorCategories(ctx, startTime, endTime, groupIDs, normalized.Platform)
	if err != nil {
		return nil, err
	}

	return buildModelMonitorGroupHealth(groups, groupIDs, usageBuckets, errorBuckets, errorCategories), nil
}

// NormalizeModelMonitorGroupHealthQuery 校验并归一化查询参数。
func NormalizeModelMonitorGroupHealthQuery(query ModelMonitorGroupHealthQuery) (ModelMonitorGroupHealthQuery, error) {
	query.Range = strings.ToLower(strings.TrimSpace(query.Range))
	if query.Range == "" {
		query.Range = ModelMonitorGroupHealthRange24H
	}
	switch query.Range {
	case ModelMonitorGroupHealthRange1H, ModelMonitorGroupHealthRange24H, ModelMonitorGroupHealthRange7D, ModelMonitorGroupHealthRange30D:
	default:
		return ModelMonitorGroupHealthQuery{}, fmt.Errorf("invalid health trend range: %s", query.Range)
	}

	if len(query.GroupIDs) > ModelMonitorGroupHealthMaxGroupIDs {
		return ModelMonitorGroupHealthQuery{}, fmt.Errorf("too many group ids, maximum is %d", ModelMonitorGroupHealthMaxGroupIDs)
	}
	uniqueIDs := make([]int64, 0, len(query.GroupIDs))
	seen := make(map[int64]struct{}, len(query.GroupIDs))
	for _, id := range query.GroupIDs {
		if id <= 0 {
			return ModelMonitorGroupHealthQuery{}, fmt.Errorf("group id must be positive")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	query.GroupIDs = uniqueIDs
	query.Platform = strings.ToLower(strings.TrimSpace(query.Platform))
	if query.Now.IsZero() {
		query.Now = time.Now().UTC()
	}
	return query, nil
}

func modelMonitorGroupHealthWindow(now time.Time, rangeValue string) (time.Time, time.Time, time.Duration) {
	now = now.UTC()
	switch rangeValue {
	case ModelMonitorGroupHealthRange1H:
		return now.Add(-time.Hour), now, 5 * time.Minute
	case ModelMonitorGroupHealthRange7D:
		return now.Add(-7 * 24 * time.Hour), now, 6 * time.Hour
	case ModelMonitorGroupHealthRange30D:
		return now.Add(-30 * 24 * time.Hour), now, 24 * time.Hour
	default:
		return now.Add(-24 * time.Hour), now, time.Hour
	}
}

func buildModelMonitorGroupHealth(
	groups []ModelMonitorGroupHealthGroup,
	groupIDs []int64,
	usageBuckets []ModelMonitorGroupHealthBucket,
	errorBuckets []ModelMonitorGroupHealthBucket,
	errorCategories []ModelMonitorGroupHealthErrorCount,
) []ModelMonitorGroupHealth {
	usageByGroup := make(map[int64]map[time.Time]*ModelMonitorGroupHealthPoint, len(groups))
	avgLatencySumByGroup := make(map[int64]float64, len(groups))
	avgLatencySampleCountByGroup := make(map[int64]int64, len(groups))
	p95LatenciesByGroup := make(map[int64][]float64, len(groups))
	firstTokenLatenciesByGroup := make(map[int64][]float64, len(groups))
	lastRequestByGroup := make(map[int64]*time.Time, len(groups))

	addBucket := func(groupID int64, bucket ModelMonitorGroupHealthBucket) {
		points, exists := usageByGroup[groupID]
		if !exists {
			points = make(map[time.Time]*ModelMonitorGroupHealthPoint)
			usageByGroup[groupID] = points
		}
		point, exists := points[bucket.BucketStart]
		if !exists {
			point = &ModelMonitorGroupHealthPoint{Time: bucket.BucketStart.UTC().Format(time.RFC3339)}
			points[bucket.BucketStart] = point
		}
		if bucket.LastRequestAt != nil {
			lastRequest := bucket.LastRequestAt.UTC()
			if current, exists := lastRequestByGroup[groupID]; !exists || lastRequest.After(*current) {
				lastRequestByGroup[groupID] = &lastRequest
			}
		}
		if bucket.SuccessCount > 0 {
			point.SuccessCount += bucket.SuccessCount
			point.AvgLatencyMS = modelMonitorRound2(bucket.AvgLatencyMS)
			point.P95LatencyMS = modelMonitorRound2(bucket.P95LatencyMS)
			if bucket.LatencySampleCount > 0 {
				avgLatencySumByGroup[groupID] += bucket.AvgLatencyMS * float64(bucket.LatencySampleCount)
				avgLatencySampleCountByGroup[groupID] += bucket.LatencySampleCount
			} else if bucket.AvgLatencyMS > 0 {
				// 兼容尚未提供样本数的仓储实现：按时间桶平均值计算，避免退化为 P95。
				avgLatencySumByGroup[groupID] += bucket.AvgLatencyMS
				avgLatencySampleCountByGroup[groupID]++
			}
			if bucket.P95LatencyMS > 0 {
				p95LatenciesByGroup[groupID] = append(p95LatenciesByGroup[groupID], bucket.P95LatencyMS)
			}
			if bucket.P95FirstTokenMS > 0 {
				firstTokenLatenciesByGroup[groupID] = append(firstTokenLatenciesByGroup[groupID], bucket.P95FirstTokenMS)
			}
		}
		if bucket.ErrorCount > 0 {
			point.ErrorCount += bucket.ErrorCount
			point.BusinessLimitedCount += bucket.BusinessLimitedCount
		}
	}

	for _, bucket := range usageBuckets {
		addBucket(bucket.GroupID, bucket)
	}
	for _, bucket := range errorBuckets {
		addBucket(bucket.GroupID, bucket)
	}

	errorsByGroup := make(map[int64][]ModelMonitorGroupHealthErrorItem, len(groups))
	for _, item := range errorCategories {
		if item.Count <= 0 {
			continue
		}
		errorsByGroup[item.GroupID] = append(errorsByGroup[item.GroupID], ModelMonitorGroupHealthErrorItem{
			Category: item.Category,
			Count:    item.Count,
		})
	}

	results := make([]ModelMonitorGroupHealth, 0, len(groups))
	for _, group := range groups {
		result := ModelMonitorGroupHealth{
			GroupID:           group.ID,
			GroupName:         group.Name,
			Platform:          group.Platform,
			EffectivePlatform: group.EffectivePlatform,
			LastRequestAt:     lastRequestByGroup[group.ID],
			Trend:             []ModelMonitorGroupHealthPoint{},
			TopErrors:         []ModelMonitorGroupHealthErrorItem{},
		}

		points := usageByGroup[group.ID]
		times := make([]time.Time, 0, len(points))
		for bucketTime := range points {
			times = append(times, bucketTime)
		}
		sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
		for _, bucketTime := range times {
			point := points[bucketTime]
			point.RequestCount = point.SuccessCount + point.ErrorCount
			point.ServiceErrorCount = point.ErrorCount - point.BusinessLimitedCount
			if point.RequestCount > 0 {
				point.SuccessRate = modelMonitorRound2(float64(point.SuccessCount) / float64(point.RequestCount) * 100)
			}
			serviceTotal := point.SuccessCount + point.ServiceErrorCount
			if serviceTotal > 0 {
				point.ServiceSuccessRate = modelMonitorRound2(float64(point.SuccessCount) / float64(serviceTotal) * 100)
			}
			result.Trend = append(result.Trend, *point)
			result.SuccessCount += point.SuccessCount
			result.ErrorCount += point.ErrorCount
			result.BusinessLimitedCount += point.BusinessLimitedCount
		}

		result.RequestCount = result.SuccessCount + result.ErrorCount
		result.ServiceErrorCount = result.ErrorCount - result.BusinessLimitedCount
		if result.RequestCount > 0 {
			result.SuccessRate = modelMonitorRound2(float64(result.SuccessCount) / float64(result.RequestCount) * 100)
			result.ErrorRate = modelMonitorRound2(float64(result.ErrorCount) / float64(result.RequestCount) * 100)
		}
		serviceTotal := result.SuccessCount + result.ServiceErrorCount
		if serviceTotal > 0 {
			result.ServiceSuccessRate = modelMonitorRound2(float64(result.SuccessCount) / float64(serviceTotal) * 100)
		}

		if sampleCount := avgLatencySampleCountByGroup[group.ID]; sampleCount > 0 {
			result.AvgLatencyMS = modelMonitorRound2(avgLatencySumByGroup[group.ID] / float64(sampleCount))
		}
		result.P95LatencyMS = modelMonitorPercentile95(p95LatenciesByGroup[group.ID])
		firstTokenLatencies := firstTokenLatenciesByGroup[group.ID]
		result.P95FirstTokenMS = modelMonitorPercentile95(firstTokenLatencies)

		sort.Slice(errorsByGroup[group.ID], func(i, j int) bool {
			return errorsByGroup[group.ID][i].Count > errorsByGroup[group.ID][j].Count
		})
		result.TopErrors = errorsByGroup[group.ID]
		if result.TopErrors == nil {
			result.TopErrors = []ModelMonitorGroupHealthErrorItem{}
		}

		result.Status = modelMonitorGroupHealthStatus(result.RequestCount, result.ServiceSuccessRate)
		results = append(results, result)
	}

	// 保持与传入分组顺序一致，便于前端展示。
	ordered := make([]ModelMonitorGroupHealth, 0, len(groupIDs))
	indexByID := make(map[int64]int, len(results))
	for index, item := range results {
		indexByID[item.GroupID] = index
	}
	for _, id := range groupIDs {
		if index, exists := indexByID[id]; exists {
			ordered = append(ordered, results[index])
		}
	}
	return ordered
}

func modelMonitorGroupHealthStatus(requestCount int64, serviceSuccessRate float64) string {
	if requestCount == 0 {
		return ModelMonitorGroupHealthStatusNoData
	}
	if requestCount < ModelMonitorGroupHealthMinSampleCount {
		return ModelMonitorGroupHealthStatusLowSample
	}
	switch {
	case serviceSuccessRate >= ModelMonitorGroupHealthHealthyThreshold:
		return ModelMonitorGroupHealthStatusHealthy
	case serviceSuccessRate >= ModelMonitorGroupHealthWarningThreshold:
		return ModelMonitorGroupHealthStatusWarning
	default:
		return ModelMonitorGroupHealthStatusCritical
	}
}

func modelMonitorAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return modelMonitorRound2(total / float64(len(values)))
}

func modelMonitorPercentile95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	index := int(math.Ceil(float64(len(sorted))*0.95)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return modelMonitorRound2(sorted[index])
}

func modelMonitorRound2(value float64) float64 {
	return math.Round(value*100) / 100
}
