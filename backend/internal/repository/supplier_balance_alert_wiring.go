package repository

import "github.com/google/wire"

// SupplierBalanceAlertWiringSet 注册供应商余额预警仓储。
var SupplierBalanceAlertWiringSet = wire.NewSet(
	NewSupplierBalanceAlertRepository,
)
