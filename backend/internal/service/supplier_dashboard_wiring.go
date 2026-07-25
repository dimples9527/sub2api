package service

import "github.com/google/wire"

func ProvideSupplierDashboardService(
	detailRepository SupplierDashboardDetailRepository,
	opsService *OpsService,
) *SupplierDashboardService {
	return NewSupplierDashboardService(detailRepository, opsService)
}

var SupplierDashboardWiringSet = wire.NewSet(
	ProvideSupplierDashboardService,
)
