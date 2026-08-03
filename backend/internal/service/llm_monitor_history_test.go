package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type llmMonitorHistoryRepositoryStub struct {
	deleteResults []int64
	cutoffs       []time.Time
	batchSizes    []int
}

func (s *llmMonitorHistoryRepositoryStub) SaveSnapshot(context.Context, LLMMonitorHistorySnapshot) error {
	return nil
}

func (s *llmMonitorHistoryRepositoryStub) LoadLatestSnapshot(context.Context, string, string, string) (*LLMMonitorHistorySnapshot, error) {
	return nil, nil
}

func (s *llmMonitorHistoryRepositoryStub) DeleteBefore(_ context.Context, cutoff time.Time, batchSize int) (int64, error) {
	s.cutoffs = append(s.cutoffs, cutoff)
	s.batchSizes = append(s.batchSizes, batchSize)
	if len(s.deleteResults) == 0 {
		return 0, nil
	}
	deleted := s.deleteResults[0]
	s.deleteResults = s.deleteResults[1:]
	return deleted, nil
}

func TestLLMMonitorHistoryServiceRunDailyMaintenanceDeletesInBatches(t *testing.T) {
	repo := &llmMonitorHistoryRepositoryStub{deleteResults: []int64{1000, 2}}
	svc := NewLLMMonitorHistoryService(repo)
	now := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)

	err := svc.RunDailyMaintenanceAt(context.Background(), now)

	require.NoError(t, err)
	require.Equal(t, []int{llmMonitorHistoryCleanupBatchSize, llmMonitorHistoryCleanupBatchSize}, repo.batchSizes)
	require.Len(t, repo.cutoffs, 2)
	require.Equal(t, now.AddDate(0, 0, -llmMonitorHistoryRetentionDays), repo.cutoffs[0])
}

func TestLLMMonitorHistorySourceKeyDoesNotExposeURL(t *testing.T) {
	key := LLMMonitorHistorySourceKey("https://user:secret@example.com/status?token=hidden")

	require.Len(t, key, 64)
	require.NotContains(t, key, "secret")
	require.NotContains(t, key, "hidden")
}
