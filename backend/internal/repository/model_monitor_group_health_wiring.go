package repository

import "github.com/google/wire"

// ModelMonitorGroupHealthWiringSet 注册模型监控分组健康趋势仓储。
var ModelMonitorGroupHealthWiringSet = wire.NewSet(
	NewModelMonitorGroupHealthRepository,
)
