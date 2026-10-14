package repository

import "github.com/google/wire"

// ModelAuditWiringSet 注册降智检测仓储。
var ModelAuditWiringSet = wire.NewSet(
	NewModelAuditRepository,
)
