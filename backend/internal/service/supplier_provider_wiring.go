package service

import "github.com/google/wire"

func ProvideSupplierGroupGuardReconciler(dataRepo SupplierProviderDataRepository) *SupplierGroupGuardReconciler {
	return NewSupplierGroupGuardReconciler(dataRepo)
}

func ProvideSupplierRateGuardService(dataRepo SupplierProviderDataRepository) *SupplierRateGuardService {
	return NewSupplierRateGuardService(dataRepo)
}

func ProvideSupplierAutomationService(
	repo SupplierAutomationRepository,
	lock SupplierAutomationLock,
	syncer SupplierProviderBatchSyncer,
	dataRepo SupplierProviderDataRepository,
	rateGuard *SupplierRateGuardService,
) *SupplierAutomationService {
	svc := NewSupplierAutomationService(repo, lock, syncer, dataRepo)
	svc.SetRateGuardService(rateGuard)
	return svc
}

func ProvideSupplierProviderGroupMatcher(dataRepo SupplierProviderDataRepository, groupRepo GroupRepository, guard *SupplierGroupGuardReconciler) *SupplierProviderGroupMatcher {
	matcher := NewSupplierProviderGroupMatcher(dataRepo, groupRepo)
	matcher.SetGuardReconciler(guard)
	return matcher
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
	ProvideSupplierGroupGuardReconciler,
	ProvideSupplierRateGuardService,
	ProvideSupplierProviderGroupMatcher,
	ProvideSupplierProviderSyncService,
	ProvideSupplierAutomationService,
)
