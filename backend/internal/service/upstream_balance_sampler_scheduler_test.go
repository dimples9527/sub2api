package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

type upstreamBalanceSamplerSchedulerServiceStub struct {
	config UpstreamBalanceSamplerConfig

	mu   sync.Mutex
	runs int
}

func (s *upstreamBalanceSamplerSchedulerServiceStub) GetConfig(context.Context) (UpstreamBalanceSamplerConfig, error) {
	return s.config, nil
}

func (s *upstreamBalanceSamplerSchedulerServiceStub) RunSample(context.Context) (UpstreamBalanceSamplerConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs++
	return s.config, nil
}

func (s *upstreamBalanceSamplerSchedulerServiceStub) runCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runs
}

func TestUpstreamBalanceSamplerSchedulerRunDueTakesFinalDailySampleInLastMinute(t *testing.T) {
	stub := &upstreamBalanceSamplerSchedulerServiceStub{
		config: UpstreamBalanceSamplerConfig{Enabled: true, IntervalSeconds: 3600},
	}
	scheduler := NewUpstreamBalanceSamplerScheduler(stub)
	loc := time.FixedZone("CST", 8*60*60)
	first := time.Date(2026, 6, 17, 23, 58, 30, 0, loc)

	if !scheduler.runDue(context.Background(), first) {
		t.Fatalf("first due run should execute")
	}
	if !scheduler.runDue(context.Background(), time.Date(2026, 6, 17, 23, 59, 0, 0, loc)) {
		t.Fatalf("last-minute daily sample should execute even before the interval elapses")
	}
	if scheduler.runDue(context.Background(), time.Date(2026, 6, 17, 23, 59, 30, 0, loc)) {
		t.Fatalf("last-minute daily sample should only execute once per local day")
	}
	if stub.runCount() != 2 {
		t.Fatalf("run count = %d, want 2", stub.runCount())
	}
	logs := scheduler.ListPollLogs()
	if len(logs) == 0 || logs[0].Trigger != "scheduled_day_end" {
		t.Fatalf("latest poll log trigger = %+v, want scheduled_day_end", logs)
	}
}
