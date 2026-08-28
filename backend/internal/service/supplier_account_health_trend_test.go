package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type accountHealthTrendHistoryStub struct {
	records []SupplierAccountHealthHistoryRecord
}

type accountHealthTrendRepositoryStub struct {
	accountHealthTrendHistoryStub
	getTrendCalls  int
	getTrendsCalls int
	lastBatchIDs   []int64
	lastPointLimit int
	validateErr    error
	batchTrends    map[int64]SupplierAccountHealthTrendResult
}

func (s *accountHealthTrendRepositoryStub) ValidateAccount(_ context.Context, _ int64) error {
	return s.validateErr
}

func (s *accountHealthTrendRepositoryStub) ListAccounts(_ context.Context, _ SupplierAccountHealthAccountListParams) (SupplierAccountHealthAccountListResult, error) {
	return SupplierAccountHealthAccountListResult{}, nil
}

func (s *accountHealthTrendRepositoryStub) GetTrend(_ context.Context, _ int64, _ time.Time) (SupplierAccountHealthTrendResult, error) {
	s.getTrendCalls++
	return SupplierAccountHealthTrendResult{}, nil
}

func (s *accountHealthTrendRepositoryStub) GetTrends(_ context.Context, accountIDs []int64, _ time.Time, pointLimit int) (map[int64]SupplierAccountHealthTrendResult, error) {
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
	require.Len(t, results[0].Points, 1)
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
