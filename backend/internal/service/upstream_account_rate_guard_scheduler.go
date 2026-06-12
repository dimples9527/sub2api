package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	upstreamAccountRateGuardRunTimeout    = 2 * time.Minute
	upstreamAccountRateGuardCheckInterval = time.Second
)

type UpstreamAccountRateGuardSchedulerService interface {
	GetRateGuardConfig(ctx context.Context) (UpstreamAccountRateGuardConfig, error)
	RunScheduledRateGuard(ctx context.Context) (UpstreamAccountRateGuardConfig, error)
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
	if _, err := s.service.RunScheduledRateGuard(runCtx); err != nil {
		slog.Warn("upstream_account_rate_guard.run_failed", "error", err)
	}
	return true
}
