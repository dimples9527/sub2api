package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type modelMonitorGroupHealthRepoStub struct {
	groups          []ModelMonitorGroupHealthGroup
	usageBuckets    []ModelMonitorGroupHealthBucket
	errorBuckets    []ModelMonitorGroupHealthBucket
	errorCategories []ModelMonitorGroupHealthErrorCount
}

func (s *modelMonitorGroupHealthRepoStub) ListGroups(context.Context, []int64, string) ([]ModelMonitorGroupHealthGroup, error) {
	return s.groups, nil
}

func (s *modelMonitorGroupHealthRepoStub) ListUsageBuckets(context.Context, time.Time, time.Time, time.Duration, []int64, string) ([]ModelMonitorGroupHealthBucket, error) {
	return s.usageBuckets, nil
}

func (s *modelMonitorGroupHealthRepoStub) ListErrorBuckets(context.Context, time.Time, time.Time, time.Duration, []int64, string) ([]ModelMonitorGroupHealthBucket, error) {
	return s.errorBuckets, nil
}

func (s *modelMonitorGroupHealthRepoStub) ListErrorCategories(context.Context, time.Time, time.Time, []int64, string) ([]ModelMonitorGroupHealthErrorCount, error) {
	return s.errorCategories, nil
}

func TestNormalizeModelMonitorGroupHealthQueryAppliesDefaults(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	query, err := NormalizeModelMonitorGroupHealthQuery(ModelMonitorGroupHealthQuery{
		Range:    " 24H ",
		GroupIDs: []int64{2, 1, 2},
		Platform: " OpenAI ",
		Now:      now,
	})
	if err != nil {
		t.Fatalf("NormalizeModelMonitorGroupHealthQuery() error = %v", err)
	}
	if query.Range != ModelMonitorGroupHealthRange24H {
		t.Fatalf("range = %q, want %q", query.Range, ModelMonitorGroupHealthRange24H)
	}
	if len(query.GroupIDs) != 2 || query.GroupIDs[0] != 2 || query.GroupIDs[1] != 1 {
		t.Fatalf("group ids = %v, want [2 1]", query.GroupIDs)
	}
	if query.Platform != "openai" {
		t.Fatalf("platform = %q, want openai", query.Platform)
	}
	if !query.Now.Equal(now) {
		t.Fatalf("now = %v, want %v", query.Now, now)
	}
}

func TestNormalizeModelMonitorGroupHealthQueryRejectsInvalidValues(t *testing.T) {
	if _, err := NormalizeModelMonitorGroupHealthQuery(ModelMonitorGroupHealthQuery{Range: "10d"}); err == nil {
		t.Fatal("expected invalid range error")
	}
	if _, err := NormalizeModelMonitorGroupHealthQuery(ModelMonitorGroupHealthQuery{GroupIDs: []int64{0}}); err == nil {
		t.Fatal("expected invalid group id error")
	}
	ids := make([]int64, ModelMonitorGroupHealthMaxGroupIDs+1)
	for index := range ids {
		ids[index] = int64(index + 1)
	}
	if _, err := NormalizeModelMonitorGroupHealthQuery(ModelMonitorGroupHealthQuery{GroupIDs: ids}); err == nil {
		t.Fatal("expected too many group ids error")
	}
}

func TestModelMonitorGroupHealthServiceCalculatesMetricsAndExcludesBusinessLimited(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	bucketStart := now.Add(-time.Hour)
	repo := &modelMonitorGroupHealthRepoStub{
		groups: []ModelMonitorGroupHealthGroup{
			{ID: 1, Name: "OpenAI 分组", Platform: "openai", EffectivePlatform: "openai"},
			{ID: 2, Name: "空分组", Platform: "openai", EffectivePlatform: "openai"},
		},
		usageBuckets: []ModelMonitorGroupHealthBucket{
			{GroupID: 1, BucketStart: bucketStart, SuccessCount: 95, AvgLatencyMS: 1200, P95LatencyMS: 1200, P95FirstTokenMS: 320},
		},
		errorBuckets: []ModelMonitorGroupHealthBucket{
			{GroupID: 1, BucketStart: bucketStart, ErrorCount: 5, BusinessLimitedCount: 3},
		},
		errorCategories: []ModelMonitorGroupHealthErrorCount{
			{GroupID: 1, Category: ModelMonitorGroupHealthErrorUpstreamRateLimit, Count: 2},
			{GroupID: 1, Category: ModelMonitorGroupHealthErrorBusinessLimited, Count: 3},
		},
	}
	service := NewModelMonitorGroupHealthService(repo)

	results, err := service.Get(context.Background(), ModelMonitorGroupHealthQuery{Range: ModelMonitorGroupHealthRange24H, GroupIDs: []int64{1, 2}, Now: now})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}

	first := results[0]
	if first.RequestCount != 100 || first.SuccessCount != 95 || first.ErrorCount != 5 {
		t.Fatalf("counts = (%d, %d, %d), want (100, 95, 5)", first.RequestCount, first.SuccessCount, first.ErrorCount)
	}
	if first.BusinessLimitedCount != 3 || first.ServiceErrorCount != 2 {
		t.Fatalf("business/service = (%d, %d), want (3, 2)", first.BusinessLimitedCount, first.ServiceErrorCount)
	}
	if first.SuccessRate != 95 || first.ErrorRate != 5 {
		t.Fatalf("rates = (%v, %v), want (95, 5)", first.SuccessRate, first.ErrorRate)
	}
	if first.ServiceSuccessRate != 97.94 {
		t.Fatalf("service success rate = %v, want 97.94", first.ServiceSuccessRate)
	}
	if first.Status != ModelMonitorGroupHealthStatusWarning {
		t.Fatalf("status = %q, want warning", first.Status)
	}
	if first.AvgLatencyMS != 1200 || first.P95LatencyMS != 1200 || first.P95FirstTokenMS != 320 {
		t.Fatalf("latencies = (%v, %v, %v), want (1200, 1200, 320)", first.AvgLatencyMS, first.P95LatencyMS, first.P95FirstTokenMS)
	}
	if len(first.Trend) != 1 || first.Trend[0].RequestCount != 100 {
		t.Fatalf("trend = %+v", first.Trend)
	}
	if len(first.TopErrors) != 2 || first.TopErrors[0].Category != ModelMonitorGroupHealthErrorBusinessLimited {
		t.Fatalf("top errors = %+v", first.TopErrors)
	}

	second := results[1]
	if second.Status != ModelMonitorGroupHealthStatusNoData || second.ServiceSuccessRate != 0 {
		t.Fatalf("empty group = %+v", second)
	}
	if second.TopErrors == nil || second.Trend == nil {
		t.Fatalf("empty group slices must be JSON-safe empty arrays: top_errors=%#v trend=%#v", second.TopErrors, second.Trend)
	}
}

func TestModelMonitorGroupHealthServiceMarksLowSample(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	repo := &modelMonitorGroupHealthRepoStub{
		groups: []ModelMonitorGroupHealthGroup{{ID: 1, Name: "低流量", Platform: "openai", EffectivePlatform: "openai"}},
		usageBuckets: []ModelMonitorGroupHealthBucket{
			{GroupID: 1, BucketStart: now.Add(-time.Hour), SuccessCount: 3},
		},
	}
	service := NewModelMonitorGroupHealthService(repo)

	results, err := service.Get(context.Background(), ModelMonitorGroupHealthQuery{GroupIDs: []int64{1}, Now: now})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(results) != 1 || results[0].Status != ModelMonitorGroupHealthStatusLowSample {
		t.Fatalf("results = %+v", results)
	}
}

func TestModelMonitorGroupHealthServiceIncludesTrendLatency(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	repo := &modelMonitorGroupHealthRepoStub{
		groups: []ModelMonitorGroupHealthGroup{{ID: 1, Name: "延迟分组", Platform: "openai", EffectivePlatform: "openai"}},
		usageBuckets: []ModelMonitorGroupHealthBucket{
			{
				GroupID:         1,
				BucketStart:     now.Add(-time.Hour),
				SuccessCount:    5,
				AvgLatencyMS:    880,
				P95LatencyMS:    1500,
				P95FirstTokenMS: 420,
			},
		},
	}
	service := NewModelMonitorGroupHealthService(repo)

	results, err := service.Get(context.Background(), ModelMonitorGroupHealthQuery{GroupIDs: []int64{1}, Now: now})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(results) != 1 || len(results[0].Trend) != 1 {
		t.Fatalf("results = %+v", results)
	}
	point := results[0].Trend[0]
	if point.AvgLatencyMS != 880 || point.P95LatencyMS != 1500 {
		t.Fatalf("trend latency = (%v, %v), want (880, 1500)", point.AvgLatencyMS, point.P95LatencyMS)
	}
}

func TestModelMonitorGroupHealthServiceAggregatesAverageAndP95LatencySeparately(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	repo := &modelMonitorGroupHealthRepoStub{
		groups: []ModelMonitorGroupHealthGroup{{ID: 1, Name: "多时间桶", Platform: "openai", EffectivePlatform: "openai"}},
		usageBuckets: []ModelMonitorGroupHealthBucket{
			{
				GroupID:      1,
				BucketStart:  now.Add(-2 * time.Hour),
				SuccessCount: 10,
				AvgLatencyMS: 100,
				P95LatencyMS: 200,
			},
			{
				GroupID:      1,
				BucketStart:  now.Add(-time.Hour),
				SuccessCount: 10,
				AvgLatencyMS: 300,
				P95LatencyMS: 600,
			},
		},
	}
	service := NewModelMonitorGroupHealthService(repo)

	results, err := service.Get(context.Background(), ModelMonitorGroupHealthQuery{GroupIDs: []int64{1}, Now: now})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count = %d, want 1", len(results))
	}
	if results[0].AvgLatencyMS != 200 || results[0].P95LatencyMS != 600 {
		t.Fatalf("latencies = (%v, %v), want (200, 600)", results[0].AvgLatencyMS, results[0].P95LatencyMS)
	}
}

func TestModelMonitorGroupHealthServiceCriticalThreshold(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	repo := &modelMonitorGroupHealthRepoStub{
		groups: []ModelMonitorGroupHealthGroup{{ID: 1, Name: "异常", Platform: "openai", EffectivePlatform: "openai"}},
		usageBuckets: []ModelMonitorGroupHealthBucket{
			{GroupID: 1, BucketStart: now.Add(-time.Hour), SuccessCount: 10},
		},
		errorBuckets: []ModelMonitorGroupHealthBucket{
			{GroupID: 1, BucketStart: now.Add(-time.Hour), ErrorCount: 5},
		},
	}
	service := NewModelMonitorGroupHealthService(repo)

	results, err := service.Get(context.Background(), ModelMonitorGroupHealthQuery{GroupIDs: []int64{1}, Now: now})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(results) != 1 || results[0].Status != ModelMonitorGroupHealthStatusCritical {
		t.Fatalf("results = %+v", results)
	}
	if results[0].ServiceSuccessRate != 66.67 {
		t.Fatalf("service success rate = %v, want 66.67", results[0].ServiceSuccessRate)
	}
}

func TestModelMonitorGroupHealthServiceReturnsRepositoryError(t *testing.T) {
	expected := errors.New("repository unavailable")
	service := NewModelMonitorGroupHealthService(&modelMonitorGroupHealthRepoErrorStub{err: expected})
	if _, err := service.Get(context.Background(), ModelMonitorGroupHealthQuery{Now: time.Now().UTC()}); !errors.Is(err, expected) {
		t.Fatalf("Get() error = %v, want %v", err, expected)
	}
}

type modelMonitorGroupHealthRepoErrorStub struct {
	err error
}

func (s *modelMonitorGroupHealthRepoErrorStub) ListGroups(context.Context, []int64, string) ([]ModelMonitorGroupHealthGroup, error) {
	return nil, s.err
}

func (s *modelMonitorGroupHealthRepoErrorStub) ListUsageBuckets(context.Context, time.Time, time.Time, time.Duration, []int64, string) ([]ModelMonitorGroupHealthBucket, error) {
	return nil, s.err
}

func (s *modelMonitorGroupHealthRepoErrorStub) ListErrorBuckets(context.Context, time.Time, time.Time, time.Duration, []int64, string) ([]ModelMonitorGroupHealthBucket, error) {
	return nil, s.err
}

func (s *modelMonitorGroupHealthRepoErrorStub) ListErrorCategories(context.Context, time.Time, time.Time, []int64, string) ([]ModelMonitorGroupHealthErrorCount, error) {
	return nil, s.err
}
