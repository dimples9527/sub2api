package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	upstreamGroupRateFixRunTimeout    = 2 * time.Minute
	upstreamGroupRateFixCheckInterval = time.Second
)

type UpstreamGroupRateFixSchedulerService interface {
	GetRateFixConfig(ctx context.Context) (UpstreamGroupAutoRateFixConfig, error)
	RunScheduledRateFix(ctx context.Context) (UpstreamGroupAutoRateFixConfig, error)
}

type UpstreamGroupRateFixScheduler struct {
	service UpstreamGroupRateFixSchedulerService

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	mu        sync.Mutex
	lastRunAt time.Time
	running   bool
}

func NewUpstreamGroupRateFixScheduler(service UpstreamGroupRateFixSchedulerService) *UpstreamGroupRateFixScheduler {
	return &UpstreamGroupRateFixScheduler{
		service: service,
		stopCh:  make(chan struct{}),
	}
}

func (s *UpstreamGroupRateFixScheduler) Start() {
	if s == nil || s.service == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(upstreamGroupRateFixCheckInterval)
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

func (s *UpstreamGroupRateFixScheduler) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UpstreamGroupRateFixScheduler) runDue(ctx context.Context, now time.Time) bool {
	if s == nil || s.service == nil {
		return false
	}
	config, err := s.service.GetRateFixConfig(ctx)
	if err != nil {
		slog.Warn("upstream_group_rate_fix.load_config_failed", "error", err)
		return false
	}
	if !config.Enabled {
		return false
	}
	interval := time.Duration(config.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Duration(DefaultUpstreamGroupRateFixIntervalSeconds) * time.Second
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

	runCtx, cancel := context.WithTimeout(ctx, upstreamGroupRateFixRunTimeout)
	defer cancel()
	if _, err := s.service.RunScheduledRateFix(runCtx); err != nil {
		slog.Warn("upstream_group_rate_fix.run_failed", "error", err)
	}
	return true
}
