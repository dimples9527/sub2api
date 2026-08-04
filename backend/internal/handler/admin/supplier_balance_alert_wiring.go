package admin

import "github.com/google/wire"

// SupplierBalanceAlertWiringSet 注册供应商余额预警 Handler。
var SupplierBalanceAlertWiringSet = wire.NewSet(
	NewSupplierBalanceAlertHandler,
)
