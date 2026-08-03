package service

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/redis/go-redis/v9"
)

// ProvideOpsCleanupService 创建并启动运维清理服务。
// 模型监控历史和渠道监控共用运维清理任务的调度、主节点锁和心跳记录。
func ProvideOpsCleanupService(
	opsRepo OpsRepository,
	db *sql.DB,
	redisClient *redis.Client,
	cfg *config.Config,
	channelMonitorSvc *ChannelMonitorService,
	llmMonitorHistory *LLMMonitorHistoryService,
	settingRepo SettingRepository,
	opsService *OpsService,
) *OpsCleanupService {
	svc := NewOpsCleanupService(opsRepo, db, redisClient, cfg, channelMonitorSvc, llmMonitorHistory, settingRepo)
	svc.Start()
	if opsService != nil {
		opsService.SetCleanupReloader(svc)
	}
	return svc
}
