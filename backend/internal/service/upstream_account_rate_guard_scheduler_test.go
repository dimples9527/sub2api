package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type upstreamAccountRateGuardSchedulerServiceStub struct {
	config UpstreamAccountRateGuardConfig
	runErr error

	mu         sync.Mutex
	runs       int
	runSources []string
	activeRuns int
	maxActive  int
	blockRun   chan struct{}
	runStarted chan struct{}
}

func (s *upstreamAccountRateGuardSchedulerServiceStub) GetRateGuardConfig(context.Context) (UpstreamAccountRateGuardConfig, error) {
	return s.config, nil
}

func (s *upstreamAccountRateGuardSchedulerServiceStub) RunScheduledRateGuard(context.Context) (UpstreamAccountRateGuardConfig, error) {
	return s.RunRateGuard(context.Background(), UpstreamAccountSyncTriggerScheduledRateGuard)
}

func (s *upstreamAccountRateGuardSchedulerServiceStub) RunRateGuard(_ context.Context, triggerSource string) (UpstreamAccountRateGuardConfig, error) {
	s.mu.Lock()
	s.runs++
	s.runSources = append(s.runSources, triggerSource)
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

func TestUpstreamAccountRateGuardSchedulerRunDueHonorsEnabledAndSecondsInterval(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config: UpstreamAccountRateGuardConfig{Enabled: true, IntervalSeconds: 5},
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)
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

func TestUpstreamAccountRateGuardSchedulerRunDueDoesNotLogWaitingTicks(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config: UpstreamAccountRateGuardConfig{Enabled: true, IntervalSeconds: 60},
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)

	if !scheduler.runDue(context.Background(), now) {
		t.Fatalf("first due run should execute")
	}
	if scheduler.runDue(context.Background(), now.Add(time.Second)) {
		t.Fatalf("second run before interval should be skipped")
	}
	if scheduler.runDue(context.Background(), now.Add(2*time.Second)) {
		t.Fatalf("third run before interval should be skipped")
	}

	logs := scheduler.ListPollLogs()
	if len(logs) != 1 {
		t.Fatalf("log count = %d, want only executed run log", len(logs))
	}
	if logs[0].Status != "success" || logs[0].Message != "executed" {
		t.Fatalf("log = %+v, want scheduled execution log", logs[0])
	}
	if stub.runs != 1 {
		t.Fatalf("run count = %d, want 1", stub.runs)
	}
}

func TestUpstreamAccountRateGuardSchedulerRunDueSkipsWhenDisabled(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config: UpstreamAccountRateGuardConfig{Enabled: false, IntervalSeconds: 1},
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)

	if scheduler.runDue(context.Background(), time.Now()) {
		t.Fatalf("disabled scheduler should skip")
	}
	if stub.runs != 0 {
		t.Fatalf("run count = %d, want 0", stub.runs)
	}
	if logs := scheduler.ListPollLogs(); len(logs) != 0 {
		t.Fatalf("logs = %+v, want no disabled heartbeat logs", logs)
	}
}

func TestUpstreamAccountRateGuardSchedulerRunDueSkipsWhileRunInFlight(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config:     UpstreamAccountRateGuardConfig{Enabled: true, IntervalSeconds: 1},
		blockRun:   make(chan struct{}),
		runStarted: make(chan struct{}, 2),
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)
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

func TestUpstreamAccountRateGuardSchedulerRunDueReportsRunErrorAsExecuted(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config: UpstreamAccountRateGuardConfig{Enabled: true, IntervalSeconds: 1},
		runErr: errors.New("boom"),
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)

	if !scheduler.runDue(context.Background(), time.Now()) {
		t.Fatalf("failed scheduled run should still count as an attempted run")
	}
	if stub.runs != 1 {
		t.Fatalf("run count = %d, want 1", stub.runs)
	}
}

func TestUpstreamAccountRateGuardSchedulerPassesTriggerSource(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config: UpstreamAccountRateGuardConfig{Enabled: true, IntervalSeconds: 1},
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)

	if !scheduler.runDue(context.Background(), time.Now()) {
		t.Fatalf("scheduled run should execute")
	}
	if _, err := scheduler.RunNow(context.Background()); err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}

	if len(stub.runSources) != 2 {
		t.Fatalf("run sources = %+v, want scheduled and manual", stub.runSources)
	}
	if stub.runSources[0] != UpstreamAccountSyncTriggerScheduledRateGuard {
		t.Fatalf("scheduled source = %q, want %q", stub.runSources[0], UpstreamAccountSyncTriggerScheduledRateGuard)
	}
	if stub.runSources[1] != UpstreamAccountSyncTriggerManualRateGuard {
		t.Fatalf("manual source = %q, want %q", stub.runSources[1], UpstreamAccountSyncTriggerManualRateGuard)
	}
}

func TestUpstreamAccountRateGuardSchedulerKeepsLatestTenLogs(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config: UpstreamAccountRateGuardConfig{Enabled: true, IntervalSeconds: 1},
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)
	now := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)

	for i := 0; i < 12; i++ {
		scheduler.runDue(context.Background(), now.Add(time.Duration(i)*time.Second))
	}

	logs := scheduler.ListPollLogs()
	if len(logs) != 10 {
		t.Fatalf("log count = %d, want latest 10", len(logs))
	}
	if !logs[0].CheckedAt.Equal(now.Add(11 * time.Second)) {
		t.Fatalf("first log time = %s, want newest tick", logs[0].CheckedAt)
	}
	if !logs[9].CheckedAt.Equal(now.Add(2 * time.Second)) {
		t.Fatalf("last log time = %s, want tenth newest tick", logs[9].CheckedAt)
	}
	for _, log := range logs {
		if log.Status != "success" || log.Trigger != "scheduled" {
			t.Fatalf("log = %+v, want scheduled success logs", log)
		}
	}
}

func TestUpstreamAccountRateGuardSchedulerRunNowUsesSameInFlightLock(t *testing.T) {
	stub := &upstreamAccountRateGuardSchedulerServiceStub{
		config:     UpstreamAccountRateGuardConfig{Enabled: true, IntervalSeconds: 1},
		blockRun:   make(chan struct{}),
		runStarted: make(chan struct{}, 2),
	}
	scheduler := NewUpstreamAccountRateGuardScheduler(stub)

	done := make(chan bool, 1)
	go func() {
		done <- scheduler.runDue(context.Background(), time.Now())
	}()
	select {
	case <-stub.runStarted:
	case <-time.After(time.Second):
		t.Fatalf("scheduled run did not start")
	}

	_, err := scheduler.RunNow(context.Background())
	if err == nil {
		close(stub.blockRun)
		<-done
		t.Fatalf("RunNow should reject while scheduled run is in flight")
	}
	logs := scheduler.ListPollLogs()
	if len(logs) == 0 || logs[0].Status != "skipped" || logs[0].Trigger != "manual" {
		close(stub.blockRun)
		<-done
		t.Fatalf("latest log = %+v, want manual skipped log before scheduled run completes", logs)
	}
	close(stub.blockRun)
	<-done

	if stub.runs != 1 {
		t.Fatalf("run count = %d, want only scheduled run", stub.runs)
	}
}
