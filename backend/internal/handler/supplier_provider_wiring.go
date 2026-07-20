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
) *admin.SupplierProviderSyncHandler {
	h := admin.NewSupplierProviderSyncHandler(syncService, dataRepo)
	h.SetGroupMatcher(groupMatcher)
	return h
}

var SupplierProviderWiringSet = wire.NewSet(
	ProvideSupplierProviderSyncHandler,
)
