package service

import "github.com/google/wire"

func ProvideSupplierProviderGroupMatcher(dataRepo SupplierProviderDataRepository, groupRepo GroupRepository) *SupplierProviderGroupMatcher {
	return NewSupplierProviderGroupMatcher(dataRepo, groupRepo)
}

func ProvideSupplierProviderSyncService(
	providerRepo SupplierProviderRepository,
	dataRepo SupplierProviderDataRepository,
	remote SupplierProviderRemoteClient,
	encryptor SecretEncryptor,
	syncLock SupplierProviderSyncLock,
	groupMatcher *SupplierProviderGroupMatcher,
) *SupplierProviderSyncService {
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, encryptor, syncLock)
	svc.SetGroupMatcher(groupMatcher)
	return svc
}

var SupplierProviderWiringSet = wire.NewSet(
	ProvideSupplierProviderGroupMatcher,
	ProvideSupplierProviderSyncService,
)
