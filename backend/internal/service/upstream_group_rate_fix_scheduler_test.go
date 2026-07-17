package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type upstreamGroupRateFixSchedulerServiceStub struct {
	config UpstreamGroupAutoRateFixConfig
	runErr error

	mu         sync.Mutex
	runs       int
	activeRuns int
	maxActive  int
	blockRun   chan struct{}
	runStarted chan struct{}
}

func (s *upstreamGroupRateFixSchedulerServiceStub) GetRateFixConfig(context.Context) (UpstreamGroupAutoRateFixConfig, error) {
	return s.config, nil
}

func (s *upstreamGroupRateFixSchedulerServiceStub) RunScheduledRateFix(context.Context) (UpstreamGroupAutoRateFixConfig, error) {
	s.mu.Lock()
	s.runs++
	s.activeRuns++
	if s.activeRuns > s.maxActive {
		s.maxActive = s.activeRuns
	}
	if s.runStarted != nil {
		select {
		case s.runStarted <- struct{}{}:
		default:
		}
	}
	s.mu.Unlock()
	if s.blockRun != nil {
		<-s.blockRun
	}
	s.mu.Lock()
	s.activeRuns--
	s.mu.Unlock()
	return s.config, s.runErr
}

func TestUpstreamGroupRateFixSchedulerRunDueHonorsEnabledAndSecondsInterval(t *testing.T) {
	stub := &upstreamGroupRateFixSchedulerServiceStub{
		config: UpstreamGroupAutoRateFixConfig{Enabled: true, IntervalSeconds: 5},
	}
	scheduler := NewUpstreamGroupRateFixScheduler(stub)
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)

	if !scheduler.runDue(context.Background(), now) {
		t.Fatalf("first due run should execute")
	}
	if scheduler.runDue(context.Background(), now.Add(4*time.Second)) {
		t.Fatalf("second run before interval should be skipped")
	}
	if !scheduler.runDue(context.Background(), now.Add(5*time.Second)) {
		t.Fatalf("run at interval boundary should execute")
	}
	if stub.runs != 2 {
		t.Fatalf("run count = %d, want 2", stub.runs)
	}
}

func TestUpstreamGroupRateFixSchedulerRunDueSkipsWhenDisabled(t *testing.T) {
	stub := &upstreamGroupRateFixSchedulerServiceStub{
		config: UpstreamGroupAutoRateFixConfig{Enabled: false, IntervalSeconds: 1},
	}
	scheduler := NewUpstreamGroupRateFixScheduler(stub)

	if scheduler.runDue(context.Background(), time.Now()) {
		t.Fatalf("disabled scheduler should skip")
	}
	if stub.runs != 0 {
		t.Fatalf("run count = %d, want 0", stub.runs)
	}
}

func TestUpstreamGroupRateFixSchedulerRunDueSkipsWhileRunInFlight(t *testing.T) {
	stub := &upstreamGroupRateFixSchedulerServiceStub{
		config:     UpstreamGroupAutoRateFixConfig{Enabled: true, IntervalSeconds: 1},
		blockRun:   make(chan struct{}),
		runStarted: make(chan struct{}, 2),
	}
	scheduler := NewUpstreamGroupRateFixScheduler(stub)
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)

	done := make(chan bool, 1)
	go func() {
		done <- scheduler.runDue(context.Background(), now)
	}()
	select {
	case <-stub.runStarted:
	case <-time.After(time.Second):
		t.Fatalf("first run did not start")
	}

	secondDone := make(chan bool, 1)
	go func() {
		secondDone <- scheduler.runDue(context.Background(), now.Add(2*time.Second))
	}()
	select {
	case ran := <-secondDone:
		if ran {
			t.Fatalf("run while previous run is in flight should be skipped")
		}
	case <-time.After(100 * time.Millisecond):
		close(stub.blockRun)
		<-done
		t.Fatalf("run while previous run is in flight should return without blocking")
	}
	close(stub.blockRun)
	if !<-done {
		t.Fatalf("first due run should execute")
	}
	if stub.maxActive != 1 {
		t.Fatalf("max active runs = %d, want 1", stub.maxActive)
	}
}

func TestUpstreamGroupRateFixSchedulerRunDueReportsRunErrorAsExecuted(t *testing.T) {
	stub := &upstreamGroupRateFixSchedulerServiceStub{
		config: UpstreamGroupAutoRateFixConfig{Enabled: true, IntervalSeconds: 1},
		runErr: errors.New("boom"),
	}
	scheduler := NewUpstreamGroupRateFixScheduler(stub)

	if !scheduler.runDue(context.Background(), time.Now()) {
		t.Fatalf("failed scheduled run should still count as an attempted run")
	}
	if stub.runs != 1 {
		t.Fatalf("run count = %d, want 1", stub.runs)
	}
}
