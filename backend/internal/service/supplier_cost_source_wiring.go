package service

import "github.com/google/wire"

// ProvideSupplierCostSourceConfigService 创建成本来源配置服务。
func ProvideSupplierCostSourceConfigService(repo SupplierCostSourceRepository) *SupplierCostSourceConfigService {
	return NewSupplierCostSourceConfigService(repo)
}

// SupplierCostSourceWiringSet 注册供应商成本来源配置服务。
var SupplierCostSourceWiringSet = wire.NewSet(ProvideSupplierCostSourceConfigService)
