package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type accountHealthTrendHistoryStub struct {
	records []SupplierAccountHealthHistoryRecord
}

type accountHealthTrendRepositoryStub struct {
	accountHealthTrendHistoryStub
	getTrendCalls     int
	getTrendsCalls    int
	lastBatchIDs      []int64
	lastPointLimit    int
	lastSummaryParams SupplierAccountHealthAccountListParams
	validateErr       error
	batchTrends       map[int64]SupplierAccountHealthTrendResult
}

func (s *accountHealthTrendRepositoryStub) ValidateAccount(_ context.Context, _ int64) error {
	return s.validateErr
}

func (s *accountHealthTrendRepositoryStub) ListAccounts(_ context.Context, _ SupplierAccountHealthAccountListParams) (SupplierAccountHealthAccountListResult, error) {
	return SupplierAccountHealthAccountListResult{}, nil
}

func (s *accountHealthTrendRepositoryStub) GetSummary(_ context.Context, params SupplierAccountHealthAccountListParams) (SupplierAccountHealthSummary, error) {
	s.lastSummaryParams = params
	return SupplierAccountHealthSummary{Total: 4, Healthy: 2, Slow: 1, Failed: 1}, nil
}

func (s *accountHealthTrendRepositoryStub) GetTrend(_ context.Context, _ int64, _, _ time.Time) (SupplierAccountHealthTrendResult, error) {
	s.getTrendCalls++
	return SupplierAccountHealthTrendResult{}, nil
}

func (s *accountHealthTrendRepositoryStub) GetTrends(_ context.Context, accountIDs []int64, _, _ time.Time, pointLimit int) (map[int64]SupplierAccountHealthTrendResult, error) {
	s.getTrendsCalls++
	s.lastBatchIDs = append([]int64(nil), accountIDs...)
	s.lastPointLimit = pointLimit
	return s.batchTrends, nil
}

func (s *accountHealthTrendRepositoryStub) DeleteBefore(_ context.Context, _ time.Time, _ int) (int, error) {
	return 0, nil
}

func (s *accountHealthTrendHistoryStub) Save(_ context.Context, record SupplierAccountHealthHistoryRecord) error {
	s.records = append(s.records, record)
	return nil
}

func TestSupplierAccountHealthTrendServiceRecordsFailedLatencyAsNil(t *testing.T) {
	stub := &accountHealthTrendHistoryStub{}
	svc := NewSupplierAccountHealthTrendService(nil, stub)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	item := SupplierAccountHealthGuardRunItem{
		LocalAccountID: 12, LocalAccountName: "账号 A", Platform: "openai",
		Sources: []SupplierAccountHealthGuardSource{{ProviderID: 7, ProviderName: "供应商 A", ProviderAccountID: 9}},
		ModelID: "gpt-4o", Status: SupplierAccountHealthGuardStatusFailed, LatencyMs: 0,
		LatencyLimitMs: 15000, SchedulableBefore: true, SchedulableAfter: false,
		Action: SupplierAccountHealthGuardActionDisabled, ConsecutiveFailed: 3,
		StartedAt: now.Add(-time.Second), FinishedAt: now, Reason: "探测失败", ErrorMessage: "timeout",
	}

	require.NoError(t, svc.RecordRunItem(context.Background(), item))
	require.Len(t, stub.records, 1)
	require.Nil(t, stub.records[0].LatencyMs)
	require.Equal(t, int64(7), stub.records[0].ProviderID)
}

func TestSupplierAccountHealthTrendServiceRejectsUnsupportedRange(t *testing.T) {
	svc := NewSupplierAccountHealthTrendService(nil, nil)
	_, err := svc.GetTrend(context.Background(), 12, "2h")
	require.Error(t, err)
}

func TestSupplierAccountHealthTrendServiceRejectsUnknownAccount(t *testing.T) {
	repo := &accountHealthTrendRepositoryStub{validateErr: ErrAccountNotFound}
	svc := NewSupplierAccountHealthTrendService(repo, repo)

	_, err := svc.GetTrend(context.Background(), 12, SupplierAccountHealthRange24h)

	require.ErrorIs(t, err, ErrAccountNotFound)
	require.Zero(t, repo.getTrendCalls)
}
func TestSupplierAccountHealthTrendServiceGetTrendsDeduplicatesAndFillsMissing(t *testing.T) {
	repo := &accountHealthTrendRepositoryStub{
		batchTrends: map[int64]SupplierAccountHealthTrendResult{
			12: {AccountID: 12, Points: []SupplierAccountHealthPoint{{CheckedAt: time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC), Status: "healthy"}}},
		},
	}
	svc := NewSupplierAccountHealthTrendService(repo, repo)

	results, err := svc.GetTrends(context.Background(), []int64{12, 12, 37}, SupplierAccountHealthRange7d)

	require.NoError(t, err)
	require.Equal(t, []int64{12, 37}, repo.lastBatchIDs)
	require.Equal(t, SupplierAccountHealthBatchPointLimit, repo.lastPointLimit)
	require.Len(t, results, 2)
	require.Equal(t, int64(12), results[0].AccountID)
	require.Equal(t, SupplierAccountHealthRange7d, results[0].Range)
	require.Len(t, results[0].Points, SupplierAccountHealthTrendBucketCount)
	sampleIndex := -1
	for index, point := range results[0].Points {
		if point.SampleCount > 0 {
			sampleIndex = index
			break
		}
	}
	require.NotEqual(t, -1, sampleIndex)
	require.Equal(t, 1, results[0].Points[sampleIndex].SampleCount)
	require.Equal(t, 1, results[0].Points[sampleIndex].HealthyCount)
	require.NotNil(t, results[0].Latest)
	require.Equal(t, int64(37), results[1].AccountID)
	require.Empty(t, results[1].Points)
	require.Nil(t, results[1].Latest)
}

func TestSupplierAccountHealthTrendServiceRejectsOversizedTrendBatch(t *testing.T) {
	repo := &accountHealthTrendRepositoryStub{}
	svc := NewSupplierAccountHealthTrendService(repo, repo)

	ids := make([]int64, 0, SupplierAccountHealthBatchMaxAccounts+1)
	for id := 1; id <= SupplierAccountHealthBatchMaxAccounts+1; id++ {
		ids = append(ids, int64(id))
	}

	_, err := svc.GetTrends(context.Background(), ids, SupplierAccountHealthRange24h)

	require.Error(t, err)
	require.Zero(t, repo.getTrendsCalls)
}

func TestSupplierAccountHealthTrendServiceAggregatesTrendIntoFixedBuckets(t *testing.T) {
	since := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	points := []SupplierAccountHealthPoint{
		{CheckedAt: since.Add(1 * time.Minute), Status: "healthy", LatencyMs: trendInt64Ptr(120)},
		{CheckedAt: since.Add(5 * time.Minute), Status: "failed"},
		{CheckedAt: since.Add(16 * time.Minute), Status: "slow", LatencyMs: trendInt64Ptr(900)},
	}

	aggregated := aggregateSupplierAccountHealthTrendPoints(points, since, 24*time.Hour)

	require.Len(t, aggregated, SupplierAccountHealthTrendBucketCount)
	require.NotNil(t, aggregated[0].BucketEndAt)
	require.Equal(t, since.Add(15*time.Minute), *aggregated[0].BucketEndAt)
	require.Equal(t, "failed", aggregated[0].Status)
	require.Equal(t, 2, aggregated[0].SampleCount)
	require.Equal(t, 1, aggregated[0].HealthyCount)
	require.Equal(t, 1, aggregated[0].FailedCount)
	require.Nil(t, aggregated[0].LatencyMs)
	require.Equal(t, "slow", aggregated[1].Status)
	require.Equal(t, 1, aggregated[1].SampleCount)
	require.Equal(t, 1, aggregated[1].SlowCount)
	require.Equal(t, int64(900), *aggregated[1].LatencyMs)
	require.Equal(t, 0, aggregated[2].SampleCount)
	require.Equal(t, "unchecked", aggregated[2].Status)
}

func TestSupplierAccountHealthTrendServiceUsesSampleCountForHealthRate(t *testing.T) {
	since := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	aggregated := aggregateSupplierAccountHealthTrendPoints([]SupplierAccountHealthPoint{
		{CheckedAt: since.Add(1 * time.Minute), Status: "healthy"},
		{CheckedAt: since.Add(2 * time.Minute), Status: "failed"},
	}, since, 24*time.Hour)

	require.Equal(t, 1, aggregated[0].HealthyCount)
	require.Equal(t, 2, aggregated[0].SampleCount)
	require.Equal(t, 1, aggregated[0].FailedCount)
}

func TestSupplierAccountHealthTrendUsesNinetySixBucketsForEveryRange(t *testing.T) {
	for _, testCase := range []struct {
		rangeValue     string
		bucketDuration time.Duration
	}{
		{rangeValue: SupplierAccountHealthRange24h, bucketDuration: 15 * time.Minute},
		{rangeValue: SupplierAccountHealthRange7d, bucketDuration: 105 * time.Minute},
		{rangeValue: SupplierAccountHealthRange30d, bucketDuration: 450 * time.Minute},
	} {
		require.Equal(t, testCase.bucketDuration,
			supplierAccountHealthTrendDuration(testCase.rangeValue)/SupplierAccountHealthTrendBucketCount)
	}
}

func TestLatestSupplierAccountHealthPointIgnoresEmptyAggregateTail(t *testing.T) {
	checkedAt := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	aggregated := aggregateSupplierAccountHealthTrendPoints([]SupplierAccountHealthPoint{
		{CheckedAt: checkedAt, Status: "healthy"},
	}, checkedAt.Add(-15*time.Minute), 24*time.Hour)

	latest := latestSupplierAccountHealthPoint([]SupplierAccountHealthPoint{
		{CheckedAt: checkedAt, Status: "healthy"},
	})

	require.Len(t, aggregated, SupplierAccountHealthTrendBucketCount)
	require.NotNil(t, aggregated[1].LatestCheckedAt)
	require.Equal(t, checkedAt, *aggregated[1].LatestCheckedAt)
	require.Equal(t, checkedAt, latest.CheckedAt)
	require.Equal(t, "healthy", latest.Status)
	require.Equal(t, 1, latest.SampleCount)
	require.Equal(t, 1, latest.HealthyCount)
	require.Zero(t, latest.SlowCount)
	require.Zero(t, latest.FailedCount)
}

func TestAggregateSupplierAccountHealthTrendPointsIgnoresPointsOutsideRange(t *testing.T) {
	since := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	end := since.Add(24 * time.Hour)
	aggregated := aggregateSupplierAccountHealthTrendPoints([]SupplierAccountHealthPoint{
		{CheckedAt: since.Add(-time.Nanosecond), Status: "failed"},
		{CheckedAt: since, Status: "healthy"},
		{CheckedAt: end.Add(-time.Nanosecond), Status: "slow"},
		{CheckedAt: end, Status: "failed"},
		{CheckedAt: end.Add(time.Nanosecond), Status: "failed"},
	}, since, 24*time.Hour)

	require.Len(t, aggregated, SupplierAccountHealthTrendBucketCount)
	require.Equal(t, 1, aggregated[0].SampleCount)
	require.Equal(t, 1, aggregated[0].HealthyCount)
	require.Equal(t, 1, aggregated[SupplierAccountHealthTrendBucketCount-1].SampleCount)
	require.Equal(t, 1, aggregated[SupplierAccountHealthTrendBucketCount-1].SlowCount)
	var sampleCount int
	for _, point := range aggregated {
		sampleCount += point.SampleCount
	}
	require.Equal(t, 2, sampleCount)
}

func TestSupplierAccountHealthTrendServiceReturnsEmptyAccountNinetySixUncheckedBuckets(t *testing.T) {
	repo := &accountHealthTrendRepositoryStub{
		batchTrends: map[int64]SupplierAccountHealthTrendResult{
			12: {AccountID: 12, Points: []SupplierAccountHealthPoint{}},
		},
	}
	svc := NewSupplierAccountHealthTrendService(repo, repo)

	results, err := svc.GetTrends(context.Background(), []int64{12}, SupplierAccountHealthRange24h)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Points, SupplierAccountHealthTrendBucketCount)
	require.Nil(t, results[0].Latest)
	for _, point := range results[0].Points {
		require.Equal(t, SupplierAccountHealthStatusUnchecked, point.Status)
		require.Zero(t, point.SampleCount)
	}
}

func TestSupplierAccountHealthTrendPointKeepsBucketCountsInJSON(t *testing.T) {
	payload, err := json.Marshal(SupplierAccountHealthPoint{
		CheckedAt: time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC),
		Status:    SupplierAccountHealthStatusUnchecked,
	})
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal(payload, &fields))
	require.NotContains(t, fields, "bucket_end_at")
	for _, field := range []string{"sample_count", "healthy_count", "slow_count", "failed_count"} {
		require.Contains(t, fields, field)
		require.Equal(t, float64(0), fields[field])
	}
}

func trendInt64Ptr(value int64) *int64 {
	return &value
}
