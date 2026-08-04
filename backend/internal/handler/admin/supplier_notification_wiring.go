package admin

import "github.com/google/wire"

// SupplierNotificationWiringSet 注册供应商通知 Handler。
var SupplierNotificationWiringSet = wire.NewSet(
	NewSupplierNotificationHandler,
)
