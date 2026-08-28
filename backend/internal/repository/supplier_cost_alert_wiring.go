package repository

import "github.com/google/wire"

// SupplierCostAlertWiringSet 注册供应商成本超额预警仓储。
var SupplierCostAlertWiringSet = wire.NewSet(NewSupplierCostAlertRepository)
