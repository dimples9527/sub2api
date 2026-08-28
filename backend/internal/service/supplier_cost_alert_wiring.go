package service

import "github.com/google/wire"

// ProvideSupplierCostAlertService 创建成本超额预警服务，并绑定通知派发能力。
func ProvideSupplierCostAlertService(
	repo SupplierCostAlertRepository,
	dispatcher SupplierBalanceAlertDispatcher,
) *SupplierCostAlertService {
	return NewSupplierCostAlertService(repo, dispatcher)
}

var SupplierCostAlertWiringSet = wire.NewSet(
	ProvideSupplierCostAlertService,
	wire.Bind(new(SupplierCostAlertHandler), new(*SupplierCostAlertService)),
)
