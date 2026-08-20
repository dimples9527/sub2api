package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

func ProvideSupplierProviderSyncHandler(
	syncService *service.SupplierProviderSyncService,
	dataRepo service.SupplierProviderDataRepository,
	groupMatcher *service.SupplierProviderGroupMatcher,
	groupGuard *service.SupplierGroupGuardReconciler,
	customPlatformService service.CustomPlatformService,
	monitorGroupPlatformOverrideService service.MonitorGroupPlatformOverrideService,
) *admin.SupplierProviderSyncHandler {
	h := admin.NewSupplierProviderSyncHandler(syncService, dataRepo)
	h.SetGroupMatcher(groupMatcher)
	h.SetGroupGuard(groupGuard)
	h.SetCustomPlatformResolver(customPlatformService)
	h.SetGroupPlatformOverrideService(monitorGroupPlatformOverrideService)
	return h
}

func ProvideSupplierProviderRechargeHandler(
	rechargeService *service.SupplierProviderRechargeService,
) *admin.SupplierProviderRechargeHandler {
	return admin.NewSupplierProviderRechargeHandler(rechargeService)
}

var SupplierProviderWiringSet = wire.NewSet(
	ProvideSupplierProviderSyncHandler,
	ProvideSupplierProviderRechargeHandler,
	admin.NewSupplierProviderAuthHandler,
)
