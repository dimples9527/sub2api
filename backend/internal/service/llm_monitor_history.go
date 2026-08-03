package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// llmMonitorHistoryRetentionDays 是远程模型监控快照的默认保留天数。
	llmMonitorHistoryRetentionDays = 30
	// llmMonitorHistoryCleanupBatchSize 限制单次清理删除的行数，避免长事务。
	llmMonitorHistoryCleanupBatchSize = 1000
)

// LLMMonitorHistorySnapshot 表示一份远程模型监控响应快照。
// SourceKey 只保存配置地址的摘要，不把可能含有敏感参数的原始地址写入数据库。
type LLMMonitorHistorySnapshot struct {
	SourceKey  string
	Period     string
	Board      string
	Payload    []byte
	CapturedAt time.Time
}

// LLMMonitorHistoryStore 是模型监控路由需要的快照读写能力。
type LLMMonitorHistoryStore interface {
	SaveSnapshot(ctx context.Context, snapshot LLMMonitorHistorySnapshot) error
	LoadLatestSnapshot(ctx context.Context, sourceKey, period, board string) (*LLMMonitorHistorySnapshot, error)
}

// LLMMonitorHistoryRepository 是模型监控历史数据访问接口。
type LLMMonitorHistoryRepository interface {
	LLMMonitorHistoryStore
	DeleteBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
}

// LLMMonitorHistoryMaintainer 是每日运维清理任务需要的维护能力。
type LLMMonitorHistoryMaintainer interface {
	RunDailyMaintenance(ctx context.Context) error
}

// LLMMonitorHistoryService 提供模型监控快照存取和定期清理能力。
type LLMMonitorHistoryService struct {
	repo LLMMonitorHistoryRepository
}

// NewLLMMonitorHistoryService 创建模型监控历史服务。
func NewLLMMonitorHistoryService(repo LLMMonitorHistoryRepository) *LLMMonitorHistoryService {
	return &LLMMonitorHistoryService{repo: repo}
}

// SaveSnapshot 保存一份经过路由层脱敏并确认包含有效 timeline 的快照。
func (s *LLMMonitorHistoryService) SaveSnapshot(ctx context.Context, snapshot LLMMonitorHistorySnapshot) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if strings.TrimSpace(snapshot.SourceKey) == "" || strings.TrimSpace(snapshot.Period) == "" || strings.TrimSpace(snapshot.Board) == "" {
		return fmt.Errorf("模型监控历史快照缺少索引字段")
	}
	if !json.Valid(snapshot.Payload) {
		return fmt.Errorf("模型监控历史快照不是有效 JSON")
	}
	if snapshot.CapturedAt.IsZero() {
		snapshot.CapturedAt = time.Now().UTC()
	} else {
		snapshot.CapturedAt = snapshot.CapturedAt.UTC()
	}
	return s.repo.SaveSnapshot(ctx, snapshot)
}

// LoadLatestSnapshot 读取指定监控请求的最近一份快照。
func (s *LLMMonitorHistoryService) LoadLatestSnapshot(ctx context.Context, sourceKey, period, board string) (*LLMMonitorHistorySnapshot, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.LoadLatestSnapshot(ctx, sourceKey, period, board)
}

// RunDailyMaintenance 批量清理默认保留期之外的模型监控快照。
func (s *LLMMonitorHistoryService) RunDailyMaintenance(ctx context.Context) error {
	return s.RunDailyMaintenanceAt(ctx, time.Now().UTC())
}

// RunDailyMaintenanceAt 使用指定时间执行清理，便于测试和保持清理边界稳定。
func (s *LLMMonitorHistoryService) RunDailyMaintenanceAt(ctx context.Context, now time.Time) error {
	if s == nil || s.repo == nil {
		return nil
	}
	cutoff := now.UTC().AddDate(0, 0, -llmMonitorHistoryRetentionDays)
	for {
		deleted, err := s.repo.DeleteBefore(ctx, cutoff, llmMonitorHistoryCleanupBatchSize)
		if err != nil {
			return fmt.Errorf("清理模型监控历史失败: %w", err)
		}
		if deleted < llmMonitorHistoryCleanupBatchSize {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

// LLMMonitorHistorySourceKey 将配置中的监控地址转换为不可逆摘要。
func LLMMonitorHistorySourceKey(rawURL string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawURL)))
	return hex.EncodeToString(sum[:])
}
