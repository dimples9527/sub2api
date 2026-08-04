package repository

import "github.com/google/wire"

// SupplierNotificationWiringSet 注册供应商通知仓储。
var SupplierNotificationWiringSet = wire.NewSet(
	NewSupplierNotificationRepository,
)
