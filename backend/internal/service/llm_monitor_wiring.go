package service

import "github.com/google/wire"

// LLMMonitorHistoryWiringSet 提供模型监控历史服务的依赖注入配置。
var LLMMonitorHistoryWiringSet = wire.NewSet(
	NewLLMMonitorHistoryService,
)
