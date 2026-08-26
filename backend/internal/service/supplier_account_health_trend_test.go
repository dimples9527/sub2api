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
	getTrendCalls int
	validateErr   error
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
