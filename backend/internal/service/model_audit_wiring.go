package service

import "github.com/google/wire"

// ModelAuditWiringSet 注册降智检测服务。
var ModelAuditWiringSet = wire.NewSet(
	NewModelAuditService,
)
