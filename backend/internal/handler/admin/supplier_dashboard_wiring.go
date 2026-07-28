package admin

import "github.com/google/wire"

// SupplierDashboardWiringSet 注册供应商运维驾驶舱 Handler。
var SupplierDashboardWiringSet = wire.NewSet(
	NewSupplierDashboardHandler,
)
