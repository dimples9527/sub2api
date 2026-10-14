package repository

import "github.com/google/wire"

// ModelDegradationWiringSet 注册降智检测仓储。
var ModelDegradationWiringSet = wire.NewSet(
	NewModelDegradationRepository,
)
