package service

import "github.com/google/wire"

// ModelDegradationWiringSet 注册降智检测服务。
var ModelDegradationWiringSet = wire.NewSet(
	NewModelDegradationService,
)
