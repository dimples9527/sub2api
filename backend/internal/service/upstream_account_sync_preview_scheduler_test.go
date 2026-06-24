package service

import (
	"context"
	"testing"
)

type upstreamAccountSyncPreviewSchedulerServiceStub struct {
	refreshCount int
}

func (s *upstreamAccountSyncPreviewSchedulerServiceStub) RefreshPreviewCache(ctx context.Context) (UpstreamAccountSyncResult, error) {
	s.refreshCount++
	return UpstreamAccountSyncResult{Summary: UpstreamAccountSyncSummary{UpstreamKeyCount: 1}}, nil
}

func TestUpstreamAccountSyncPreviewSchedulerRefreshInvokesService(t *testing.T) {
	stub := &upstreamAccountSyncPreviewSchedulerServiceStub{}
	scheduler := NewUpstreamAccountSyncPreviewScheduler(stub)

	if ok := scheduler.refresh(context.Background()); !ok {
		t.Fatal("refresh returned false, want true")
	}
	if stub.refreshCount != 1 {
		t.Fatalf("refresh count = %d, want 1", stub.refreshCount)
	}
}
