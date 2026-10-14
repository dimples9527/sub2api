package handler

import "github.com/google/wire"

// ModelAuditWiringSet 注册降智检测用户侧 handler。
var ModelAuditWiringSet = wire.NewSet(
	NewModelAuditHandler,
)
