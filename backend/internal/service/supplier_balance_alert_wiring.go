package service

import "github.com/google/wire"

// ProvideSupplierBalanceAlertSource 复用供应商管理的余额客户端，但向预警模块暴露独立数据源接口。
func ProvideSupplierBalanceAlertSource(
	providerRepo SupplierProviderRepository,
	remote SupplierProviderRemoteClient,
	encryptor SecretEncryptor,
) SupplierBalanceSource {
	return NewSupplierBalanceAlertSource(providerRepo, remote, encryptor)
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
