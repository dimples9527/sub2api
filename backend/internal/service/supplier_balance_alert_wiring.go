package service

import "github.com/google/wire"

// ProvideSupplierBalanceAlertSource 将供应商同步任务写入的本地余额提供给预警模块。
func ProvideSupplierBalanceAlertSource(providerRepo SupplierProviderRepository) SupplierBalanceSource {
	return NewSupplierBalanceAlertSource(providerRepo)
}

// ProvideSupplierBalanceAlertService 创建并启动供应商余额扫描服务。
func ProvideSupplierBalanceAlertService(
	repo SupplierBalanceAlertRepository,
	source SupplierBalanceSource,
	dispatcher SupplierBalanceAlertDispatcher,
) *SupplierBalanceAlertService {
	svc := NewSupplierBalanceAlertService(repo, source, dispatcher)
	svc.Start()
	return svc
}

var SupplierBalanceAlertWiringSet = wire.NewSet(
	ProvideSupplierBalanceAlertSource,
	ProvideSupplierBalanceAlertService,
)
