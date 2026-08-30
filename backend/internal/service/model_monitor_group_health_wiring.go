package service

import "github.com/google/wire"

// ModelMonitorGroupHealthWiringSet 注册模型监控分组健康趋势服务。
var ModelMonitorGroupHealthWiringSet = wire.NewSet(
	NewModelMonitorGroupHealthService,
)
