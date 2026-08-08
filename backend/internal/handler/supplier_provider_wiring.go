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
) *admin.SupplierProviderSyncHandler {
	h := admin.NewSupplierProviderSyncHandler(syncService, dataRepo)
	h.SetGroupMatcher(groupMatcher)
	h.SetGroupGuard(groupGuard)
	h.SetCustomPlatformResolver(customPlatformService)
	return h
}

var SupplierProviderWiringSet = wire.NewSet(
	ProvideSupplierProviderSyncHandler,
	admin.NewSupplierProviderAuthHandler,
)
