package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	upstreamAccountHealthGuardRunTimeout    = 30 * time.Minute
	upstreamAccountHealthGuardCheckInterval = time.Second
)

type UpstreamAccountHealthGuardSchedulerService interface {
	GetConfig(ctx context.Context) (UpstreamAccountHealthGuardConfig, error)
	RunScheduled(ctx context.Context) (UpstreamAccountHealthGuardRunResponse, error)
	Run(ctx context.Context, trigger string) (UpstreamAccountHealthGuardRunResponse, error)
}

type UpstreamAccountHealthGuardPollLog struct {
	CheckedAt time.Time `json:"checked_at"`
	Trigger   string    `json:"trigger"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
}

type UpstreamAccountHealthGuardScheduler struct {
	service UpstreamAccountHealthGuardSchedulerService

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	mu        sync.Mutex
	lastRunAt time.Time
	running   bool
	pollLogs  []UpstreamAccountHealthGuardPollLog
}

func NewUpstreamAccountHealthGuardScheduler(service UpstreamAccountHealthGuardSchedulerService) *UpstreamAccountHealthGuardScheduler {
	return &UpstreamAccountHealthGuardScheduler{
		service: service,
		stopCh:  make(chan struct{}),
	}
}

func (s *UpstreamAccountHealthGuardScheduler) Start() {
	if s == nil || s.service == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(upstreamAccountHealthGuardCheckInterval)
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

func (s *UpstreamAccountHealthGuardScheduler) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UpstreamAccountHealthGuardScheduler) runDue(ctx context.Context, now time.Time) bool {
	if s == nil || s.service == nil {
		return false
	}
	config, err := s.service.GetConfig(ctx)
	if err != nil {
		s.appendPollLog(UpstreamAccountHealthGuardPollLog{CheckedAt: now.UTC(), Trigger: "scheduled", Status: "failed", Message: err.Error()})
		slog.Warn("upstream_account_health_guard.load_config_failed", "error", err)
		return false
	}
	if !config.Enabled {
		return false
	}
	interval := time.Duration(config.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Duration(DefaultUpstreamAccountHealthGuardIntervalSeconds) * time.Second
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.appendPollLog(UpstreamAccountHealthGuardPollLog{CheckedAt: now.UTC(), Trigger: "scheduled", Status: "skipped", Message: "run already in flight"})
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

	runCtx, cancel := context.WithTimeout(ctx, upstreamAccountHealthGuardRunTimeout)
	defer cancel()
	response, err := s.service.RunScheduled(runCtx)
	if err != nil {
		s.appendPollLog(UpstreamAccountHealthGuardPollLog{CheckedAt: now.UTC(), Trigger: "scheduled", Status: "failed", Message: err.Error()})
		slog.Warn("upstream_account_health_guard.run_failed", "error", err)
		return true
	}
	message := response.Record.Message
	if message == "" {
		message = "executed"
	}
	s.appendPollLog(UpstreamAccountHealthGuardPollLog{CheckedAt: now.UTC(), Trigger: "scheduled", Status: "success", Message: message})
	return true
}

func (s *UpstreamAccountHealthGuardScheduler) RunNow(ctx context.Context) (UpstreamAccountHealthGuardRunResponse, error) {
	if s == nil || s.service == nil {
		return UpstreamAccountHealthGuardRunResponse{}, infraerrors.ServiceUnavailable("UPSTREAM_ACCOUNT_HEALTH_GUARD_UNAVAILABLE", "upstream account health guard scheduler is unavailable")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		err := infraerrors.Conflict("UPSTREAM_ACCOUNT_HEALTH_GUARD_RUNNING", "upstream account health guard is already running")
		s.appendPollLog(UpstreamAccountHealthGuardPollLog{CheckedAt: now, Trigger: "manual", Status: "skipped", Message: "run already in flight"})
		return UpstreamAccountHealthGuardRunResponse{}, err
	}
	s.lastRunAt = now
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	runCtx, cancel := context.WithTimeout(ctx, upstreamAccountHealthGuardRunTimeout)
	defer cancel()
	response, err := s.service.Run(runCtx, UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		s.appendPollLog(UpstreamAccountHealthGuardPollLog{CheckedAt: now, Trigger: "manual", Status: "failed", Message: err.Error()})
		return response, err
	}
	message := response.Record.Message
	if message == "" {
		message = "executed"
	}
	s.appendPollLog(UpstreamAccountHealthGuardPollLog{CheckedAt: now, Trigger: "manual", Status: "success", Message: message})
	return response, nil
}

func (s *UpstreamAccountHealthGuardScheduler) ListPollLogs() []UpstreamAccountHealthGuardPollLog {
	if s == nil {
		return []UpstreamAccountHealthGuardPollLog{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]UpstreamAccountHealthGuardPollLog, len(s.pollLogs))
	copy(out, s.pollLogs)
	return out
}

func (s *UpstreamAccountHealthGuardScheduler) appendPollLog(log UpstreamAccountHealthGuardPollLog) {
	if s == nil {
		return
	}
	if log.CheckedAt.IsZero() {
		log.CheckedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollLogs = append([]UpstreamAccountHealthGuardPollLog{log}, s.pollLogs...)
	if len(s.pollLogs) > 10 {
		s.pollLogs = s.pollLogs[:10]
	}
}
