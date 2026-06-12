package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	upstreamAccountRateGuardRunTimeout    = 2 * time.Minute
	upstreamAccountRateGuardCheckInterval = time.Second
)

type UpstreamAccountRateGuardSchedulerService interface {
	GetRateGuardConfig(ctx context.Context) (UpstreamAccountRateGuardConfig, error)
	RunScheduledRateGuard(ctx context.Context) (UpstreamAccountRateGuardConfig, error)
	RunRateGuard(ctx context.Context, triggerSource string) (UpstreamAccountRateGuardConfig, error)
}

type UpstreamAccountRateGuardPollLog struct {
	CheckedAt time.Time `json:"checked_at"`
	Trigger   string    `json:"trigger"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
}

type UpstreamAccountRateGuardScheduler struct {
	service UpstreamAccountRateGuardSchedulerService

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	mu        sync.Mutex
	lastRunAt time.Time
	running   bool
	pollLogs  []UpstreamAccountRateGuardPollLog
}

func NewUpstreamAccountRateGuardScheduler(service UpstreamAccountRateGuardSchedulerService) *UpstreamAccountRateGuardScheduler {
	return &UpstreamAccountRateGuardScheduler{
		service: service,
		stopCh:  make(chan struct{}),
	}
}

func (s *UpstreamAccountRateGuardScheduler) Start() {
	if s == nil || s.service == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(upstreamAccountRateGuardCheckInterval)
			defer ticker.Stop()
			for {
				select {
				case now := <-ticker.C:
					s.runDue(context.Background(), now)
				case <-s.stopCh:
					return
				}
			}
		}()
	})
}

func (s *UpstreamAccountRateGuardScheduler) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UpstreamAccountRateGuardScheduler) runDue(ctx context.Context, now time.Time) bool {
	if s == nil || s.service == nil {
		return false
	}
	config, err := s.service.GetRateGuardConfig(ctx)
	if err != nil {
		s.appendPollLog(UpstreamAccountRateGuardPollLog{
			CheckedAt: now.UTC(),
			Trigger:   "scheduled",
			Status:    "failed",
			Message:   err.Error(),
		})
		slog.Warn("upstream_account_rate_guard.load_config_failed", "error", err)
		return false
	}
	if !config.Enabled {
		return false
	}
	interval := time.Duration(config.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Duration(DefaultUpstreamAccountRateGuardIntervalSeconds) * time.Second
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.appendPollLog(UpstreamAccountRateGuardPollLog{
			CheckedAt: now.UTC(),
			Trigger:   "scheduled",
			Status:    "skipped",
			Message:   "run already in flight",
		})
		return false
	}
	if !s.lastRunAt.IsZero() && now.Sub(s.lastRunAt) < interval {
		s.mu.Unlock()
		return false
	}
	s.lastRunAt = now
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	runCtx, cancel := context.WithTimeout(ctx, upstreamAccountRateGuardRunTimeout)
	defer cancel()
	if _, err := s.service.RunRateGuard(runCtx, UpstreamAccountSyncTriggerScheduledRateGuard); err != nil {
		s.appendPollLog(UpstreamAccountRateGuardPollLog{
			CheckedAt: now.UTC(),
			Trigger:   "scheduled",
			Status:    "failed",
			Message:   err.Error(),
		})
		slog.Warn("upstream_account_rate_guard.run_failed", "error", err)
		return true
	}
	s.appendPollLog(UpstreamAccountRateGuardPollLog{
		CheckedAt: now.UTC(),
		Trigger:   "scheduled",
		Status:    "success",
		Message:   "executed",
	})
	return true
}

func (s *UpstreamAccountRateGuardScheduler) RunNow(ctx context.Context) (UpstreamAccountRateGuardConfig, error) {
	if s == nil || s.service == nil {
		return UpstreamAccountRateGuardConfig{}, infraerrors.ServiceUnavailable("UPSTREAM_ACCOUNT_RATE_GUARD_UNAVAILABLE", "upstream account rate guard scheduler is unavailable")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		err := infraerrors.Conflict("UPSTREAM_ACCOUNT_RATE_GUARD_RUNNING", "upstream account rate guard is already running")
		s.appendPollLog(UpstreamAccountRateGuardPollLog{
			CheckedAt: now,
			Trigger:   "manual",
			Status:    "skipped",
			Message:   "run already in flight",
		})
		return UpstreamAccountRateGuardConfig{}, err
	}
	s.lastRunAt = now
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	runCtx, cancel := context.WithTimeout(ctx, upstreamAccountRateGuardRunTimeout)
	defer cancel()
	config, err := s.service.RunRateGuard(runCtx, UpstreamAccountSyncTriggerManualRateGuard)
	if err != nil {
		s.appendPollLog(UpstreamAccountRateGuardPollLog{
			CheckedAt: now,
			Trigger:   "manual",
			Status:    "failed",
			Message:   err.Error(),
		})
		return config, err
	}
	s.appendPollLog(UpstreamAccountRateGuardPollLog{
		CheckedAt: now,
		Trigger:   "manual",
		Status:    "success",
		Message:   "executed",
	})
	return config, nil
}

func (s *UpstreamAccountRateGuardScheduler) ListPollLogs() []UpstreamAccountRateGuardPollLog {
	if s == nil {
		return []UpstreamAccountRateGuardPollLog{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]UpstreamAccountRateGuardPollLog, len(s.pollLogs))
	copy(out, s.pollLogs)
	return out
}

func (s *UpstreamAccountRateGuardScheduler) appendPollLog(log UpstreamAccountRateGuardPollLog) {
	if s == nil {
		return
	}
	if log.CheckedAt.IsZero() {
		log.CheckedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollLogs = append([]UpstreamAccountRateGuardPollLog{log}, s.pollLogs...)
	if len(s.pollLogs) > 10 {
		s.pollLogs = s.pollLogs[:10]
	}
}
