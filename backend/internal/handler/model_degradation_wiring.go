package handler

import "github.com/google/wire"

// ModelDegradationWiringSet 注册降智检测用户侧 handler。
var ModelDegradationWiringSet = wire.NewSet(
	NewModelDegradationHandler,
)
