package repository

import "github.com/google/wire"

// SupplierCostSourceWiringSet 注册供应商成本来源配置仓储。
var SupplierCostSourceWiringSet = wire.NewSet(NewSupplierCostSourceConfigRepository)
