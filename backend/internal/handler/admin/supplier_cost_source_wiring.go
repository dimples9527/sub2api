package admin

import "github.com/google/wire"

// SupplierCostSourceWiringSet 注册供应商成本来源配置 Handler。
var SupplierCostSourceWiringSet = wire.NewSet(NewSupplierCostSourceConfigHandler)
