package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	upstreamAccountSyncPreviewRefreshInterval = 5 * time.Minute
	upstreamAccountSyncPreviewRefreshTimeout  = 2 * time.Minute
)

type UpstreamAccountSyncPreviewSchedulerService interface {
	RefreshPreviewCache(ctx context.Context) (UpstreamAccountSyncResult, error)
}

type UpstreamAccountSyncPreviewScheduler struct {
	service UpstreamAccountSyncPreviewSchedulerService

	stopCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

func NewUpstreamAccountSyncPreviewScheduler(service UpstreamAccountSyncPreviewSchedulerService) *UpstreamAccountSyncPreviewScheduler {
	return &UpstreamAccountSyncPreviewScheduler{
		service: service,
		stopCh:  make(chan struct{}),
	}
}

func (s *UpstreamAccountSyncPreviewScheduler) Start() {
	if s == nil || s.service == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.refresh(context.Background())
			ticker := time.NewTicker(upstreamAccountSyncPreviewRefreshInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					s.refresh(context.Background())
				case <-s.stopCh:
					return
				}
			}
		}()
	})
}

func (s *UpstreamAccountSyncPreviewScheduler) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UpstreamAccountSyncPreviewScheduler) refresh(ctx context.Context) bool {
	if s == nil || s.service == nil {
		return false
	}
	runCtx, cancel := context.WithTimeout(ctx, upstreamAccountSyncPreviewRefreshTimeout)
	defer cancel()
	if _, err := s.service.RefreshPreviewCache(runCtx); err != nil {
		slog.Warn("upstream_account_sync_preview.refresh_failed", "error", err)
		return false
	}
	return true
}
