package service

import (
	"context"
	"fmt"
	"strings"
)

// MonitorGroupPlatformOverrideRepository 存储仅供模型监控使用的分组实际平台覆盖配置。
type MonitorGroupPlatformOverrideRepository interface {
	ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]string, error)
	Set(ctx context.Context, groupID int64, platform string) error
	Clear(ctx context.Context, groupID int64) error
}

// MonitorGroupPlatformOverrideService 管理模型监控专用的分组实际平台覆盖配置。
type MonitorGroupPlatformOverrideService interface {
	ListByGroupIDs(ctx context.Context, groupIDs []int64) (map[int64]string, error)
	Set(ctx context.Context, groupID int64, platform string) error
	Clear(ctx context.Context, groupID int64) error
}

type monitorGroupPlatformOverrideService struct {
	repo                  MonitorGroupPlatformOverrideRepository
	customPlatformService CustomPlatformService
}

// NewMonitorGroupPlatformOverrideService 创建模型监控分组实际平台覆盖服务。
func NewMonitorGroupPlatformOverrideService(repo MonitorGroupPlatformOverrideRepository) MonitorGroupPlatformOverrideService {
	return NewMonitorGroupPlatformOverrideServiceWithCustomPlatform(repo, nil)
}

// NewMonitorGroupPlatformOverrideServiceWithCustomPlatform 创建带自定义平台校验的模型监控分组实际平台覆盖服务。
func NewMonitorGroupPlatformOverrideServiceWithCustomPlatform(repo MonitorGroupPlatformOverrideRepository, customPlatformService CustomPlatformService) MonitorGroupPlatformOverrideService {
	return &monitorGroupPlatformOverrideService{repo: repo, customPlatformService: customPlatformService}
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
	if platform == "" {
		return fmt.Errorf("unsupported monitor group platform: %s", platform)
	}
	if s == nil || s.repo == nil {
		return fmt.Errorf("monitor group platform override repository is not initialized")
	}
	if !IsCorePlatform(platform) {
		if s.customPlatformService == nil {
			return fmt.Errorf("unsupported monitor group platform: %s", platform)
		}
		if _, err := s.customPlatformService.ResolveEnabled(ctx, platform); err != nil {
			return fmt.Errorf("unsupported monitor group platform: %s", platform)
		}
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
