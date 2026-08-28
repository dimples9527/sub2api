package admin

import "github.com/google/wire"

// SupplierCostAlertWiringSet 注册供应商成本超额预警 Handler。
var SupplierCostAlertWiringSet = wire.NewSet(NewSupplierCostAlertHandler)
