package service

import (
	"context"
	"fmt"
	"strings"
)

// MonitorGroupPlatformOverrideRepository stores the optional platform used only by model monitoring.
type MonitorGroupPlatformOverrideRepository interface {
	ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]string, error)
	Set(ctx context.Context, groupID int64, platform string) error
	Clear(ctx context.Context, groupID int64) error
}

// MonitorGroupPlatformOverrideService manages the model-monitor-only group platform override.
type MonitorGroupPlatformOverrideService interface {
	ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]string, error)
	Set(ctx context.Context, groupID int64, platform string) error
	Clear(ctx context.Context, groupID int64) error
}

type monitorGroupPlatformOverrideService struct {
	repo MonitorGroupPlatformOverrideRepository
}

// NewMonitorGroupPlatformOverrideService creates the model monitor platform override service.
func NewMonitorGroupPlatformOverrideService(repo MonitorGroupPlatformOverrideRepository) MonitorGroupPlatformOverrideService {
	return &monitorGroupPlatformOverrideService{repo: repo}
}

func (s *monitorGroupPlatformOverrideService) ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]string, error) {
	if s == nil || s.repo == nil {
		return map[int64]string{}, nil
	}
	return s.repo.ListByGroupIDs(ctx, groupIDs)
}

func (s *monitorGroupPlatformOverrideService) Set(ctx context.Context, groupID int64, platform string) error {
	if groupID <= 0 {
		return fmt.Errorf("group id must be positive")
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if !isMonitorGroupPlatform(platform) {
		return fmt.Errorf("unsupported monitor group platform: %s", platform)
	}
	if s == nil || s.repo == nil {
		return fmt.Errorf("monitor group platform override repository is not initialized")
	}
	return s.repo.Set(ctx, groupID, platform)
}

func (s *monitorGroupPlatformOverrideService) Clear(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return fmt.Errorf("group id must be positive")
	}
	if s == nil || s.repo == nil {
		return fmt.Errorf("monitor group platform override repository is not initialized")
	}
	return s.repo.Clear(ctx, groupID)
}

func isMonitorGroupPlatform(platform string) bool {
	switch platform {
	case PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok, PlatformComposite:
		return true
	default:
		return false
	}
}
