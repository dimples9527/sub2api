package service

import "github.com/google/wire"

func ProvideSupplierGroupGuardReconciler(dataRepo SupplierProviderDataRepository) *SupplierGroupGuardReconciler {
	return NewSupplierGroupGuardReconciler(dataRepo)
}

func ProvideSupplierRateGuardService(dataRepo SupplierProviderDataRepository) *SupplierRateGuardService {
	return NewSupplierRateGuardService(dataRepo)
}

func ProvideSupplierAccountRateGuardRateSyncer(syncer *SupplierProviderSyncService) SupplierAccountRateGuardRateSyncer {
	return syncer
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

// ProvideSupplierProviderRemoteClient 构造供应商远程客户端，并注入上游 Turnstile 打码求解器。
func ProvideSupplierProviderRemoteClient(tokenCache SupplierProviderTokenCache, settingService *SettingService) *SupplierProviderRemoteRegistry {
	solver := NewSettingBackedSupplierTurnstileSolver(settingService, nil)
	return NewSupplierProviderRemoteRegistry(nil, tokenCache, solver)
}

var SupplierProviderWiringSet = wire.NewSet(
	ProvideSupplierGroupGuardReconciler,
	ProvideSupplierRateGuardService,
	ProvideSupplierProviderGroupMatcher,
	ProvideSupplierProviderSyncService,
	ProvideSupplierAccountRateGuardRateSyncer,
	NewSupplierAccountRateGuardService,
)
