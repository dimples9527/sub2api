package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	upstreamBalanceSamplerRunTimeout    = 2 * time.Minute
	upstreamBalanceSamplerCheckInterval = time.Second
)

type UpstreamBalanceSamplerSchedulerService interface {
	GetConfig(ctx context.Context) (UpstreamBalanceSamplerConfig, error)
	RunSample(ctx context.Context) (UpstreamBalanceSamplerConfig, error)
}

type UpstreamBalanceSamplerPollLog struct {
	CheckedAt time.Time `json:"checked_at"`
	Trigger   string    `json:"trigger"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
}

type UpstreamBalanceSamplerScheduler struct {
	service UpstreamBalanceSamplerSchedulerService

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup

	mu                sync.Mutex
	lastRunAt         time.Time
	lastDayEndRunDate string
	running           bool
	pollLogs          []UpstreamBalanceSamplerPollLog
}

func NewUpstreamBalanceSamplerScheduler(service UpstreamBalanceSamplerSchedulerService) *UpstreamBalanceSamplerScheduler {
	return &UpstreamBalanceSamplerScheduler{
		service: service,
		stopCh:  make(chan struct{}),
	}
}

func (s *UpstreamBalanceSamplerScheduler) Start() {
	if s == nil || s.service == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(upstreamBalanceSamplerCheckInterval)
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

func (s *UpstreamBalanceSamplerScheduler) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UpstreamBalanceSamplerScheduler) runDue(ctx context.Context, now time.Time) bool {
	if s == nil || s.service == nil {
		return false
	}
	finalSampleDate, finalSampleDue := upstreamBalanceSamplerFinalDailySampleDate(now)
	trigger := "scheduled"
	config, err := s.service.GetConfig(ctx)
	if err != nil {
		s.appendPollLog(UpstreamBalanceSamplerPollLog{CheckedAt: now.UTC(), Trigger: "scheduled", Status: "failed", Message: err.Error()})
		slog.Warn("upstream_balance_sampler.load_config_failed", "error", err)
		return false
	}
	if !config.Enabled {
		return false
	}
	interval := time.Duration(config.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Duration(DefaultUpstreamBalanceSamplerIntervalSeconds) * time.Second
	}
	s.mu.Lock()
	forceFinalSample := finalSampleDue && s.lastDayEndRunDate != finalSampleDate
	if s.running {
		s.mu.Unlock()
		if forceFinalSample {
			trigger = "scheduled_day_end"
		}
		s.appendPollLog(UpstreamBalanceSamplerPollLog{CheckedAt: now.UTC(), Trigger: trigger, Status: "skipped", Message: "run already in flight"})
		return false
	}
	if !forceFinalSample && !s.lastRunAt.IsZero() && now.Sub(s.lastRunAt) < interval {
		s.mu.Unlock()
		return false
	}
	s.lastRunAt = now
	if forceFinalSample {
		s.lastDayEndRunDate = finalSampleDate
		trigger = "scheduled_day_end"
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	runCtx, cancel := context.WithTimeout(ctx, upstreamBalanceSamplerRunTimeout)
	defer cancel()
	if _, err := s.service.RunSample(runCtx); err != nil {
		s.appendPollLog(UpstreamBalanceSamplerPollLog{CheckedAt: now.UTC(), Trigger: trigger, Status: "failed", Message: err.Error()})
		slog.Warn("upstream_balance_sampler.run_failed", "error", err)
		return true
	}
	s.appendPollLog(UpstreamBalanceSamplerPollLog{CheckedAt: now.UTC(), Trigger: trigger, Status: "success", Message: "executed"})
	return true
}

func upstreamBalanceSamplerFinalDailySampleDate(now time.Time) (string, bool) {
	loc := upstreamBalanceStatsLocation()
	localNow := now.In(loc)
	if localNow.Hour() != 23 || localNow.Minute() != 59 {
		return "", false
	}
	return localNow.Format("2006-01-02"), true
}

func (s *UpstreamBalanceSamplerScheduler) RunNow(ctx context.Context) (UpstreamBalanceSamplerConfig, error) {
	if s == nil || s.service == nil {
		return UpstreamBalanceSamplerConfig{}, infraerrors.ServiceUnavailable("UPSTREAM_BALANCE_SAMPLER_UNAVAILABLE", "upstream balance sampler scheduler is unavailable")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		err := infraerrors.Conflict("UPSTREAM_BALANCE_SAMPLER_RUNNING", "upstream balance sampler is already running")
		s.appendPollLog(UpstreamBalanceSamplerPollLog{CheckedAt: now, Trigger: "manual", Status: "skipped", Message: "run already in flight"})
		return UpstreamBalanceSamplerConfig{}, err
	}
	s.lastRunAt = now
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	runCtx, cancel := context.WithTimeout(ctx, upstreamBalanceSamplerRunTimeout)
	defer cancel()
	config, err := s.service.RunSample(runCtx)
	if err != nil {
		s.appendPollLog(UpstreamBalanceSamplerPollLog{CheckedAt: now, Trigger: "manual", Status: "failed", Message: err.Error()})
		return config, err
	}
	s.appendPollLog(UpstreamBalanceSamplerPollLog{CheckedAt: now, Trigger: "manual", Status: "success", Message: "executed"})
	return config, nil
}

func (s *UpstreamBalanceSamplerScheduler) ListPollLogs() []UpstreamBalanceSamplerPollLog {
	if s == nil {
		return []UpstreamBalanceSamplerPollLog{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]UpstreamBalanceSamplerPollLog, len(s.pollLogs))
	copy(out, s.pollLogs)
	return out
}

func (s *UpstreamBalanceSamplerScheduler) appendPollLog(log UpstreamBalanceSamplerPollLog) {
	if s == nil {
		return
	}
	if log.CheckedAt.IsZero() {
		log.CheckedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollLogs = append([]UpstreamBalanceSamplerPollLog{log}, s.pollLogs...)
	if len(s.pollLogs) > 10 {
		s.pollLogs = s.pollLogs[:10]
	}
}
