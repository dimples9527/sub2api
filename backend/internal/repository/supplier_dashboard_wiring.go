package repository

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

var SupplierDashboardWiringSet = wire.NewSet(
	NewSupplierDashboardRepository,
	wire.Bind(new(service.SupplierDashboardDetailRepository), new(*supplierDashboardRepository)),
)
