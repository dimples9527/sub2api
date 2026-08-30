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

func ProvideSupplierProviderRechargeRemoteClient(remote *SupplierProviderRemoteRegistry) SupplierProviderRemoteRechargeHistoryClient {
	return remote
}

func ProvideSupplierProviderRechargeService(
	providerRepo SupplierProviderRepository,
	rechargeRepo SupplierProviderRechargeRepository,
	remote SupplierProviderRemoteRechargeHistoryClient,
	encryptor SecretEncryptor,
) *SupplierProviderRechargeService {
	return NewSupplierProviderRechargeService(providerRepo, rechargeRepo, remote, encryptor)
}

func ProvideSupplierProviderSyncService(
	providerRepo SupplierProviderRepository,
	dataRepo SupplierProviderDataRepository,
	rechargeRepo SupplierProviderRechargeRepository,
	remote SupplierProviderRemoteClient,
	encryptor SecretEncryptor,
	syncLock SupplierProviderSyncLock,
	groupMatcher *SupplierProviderGroupMatcher,
	costDeviationSettings *SupplierCostDeviationSettingsService,
	costSourceService *SupplierCostSourceConfigService,
	costReviewService *SupplierProviderCostReviewService,
	costAlertHandler SupplierCostAlertHandler,
	groupChangeNotifier SupplierGroupChangeNotifier,
) *SupplierProviderSyncService {
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, encryptor, syncLock, rechargeRepo)
	svc.SetGroupMatcher(groupMatcher)
	svc.SetCostDeviationThresholdProvider(costDeviationSettings)
	svc.SetCostSourceResolver(costSourceService)
	svc.SetCostReviewService(costReviewService)
	svc.SetCostAlertHandler(costAlertHandler)
	svc.SetGroupChangeNotifier(groupChangeNotifier)
	return svc
}

func ProvideSupplierProviderCostReviewService(repo SupplierProviderCostReviewRepository) *SupplierProviderCostReviewService {
	return NewSupplierProviderCostReviewService(repo)
}

// ProvideSupplierProviderRemoteClient 构造供应商远程客户端，并注入上游 Turnstile 打码求解器。
func ProvideSupplierProviderRemoteClient(tokenCache SupplierProviderTokenCache, settingService *SettingService, authAudit *SupplierProviderAuthAuditService) *SupplierProviderRemoteRegistry {
	solver := NewSettingBackedSupplierTurnstileSolver(settingService, nil)
	registry := NewSupplierProviderRemoteRegistry(nil, tokenCache, solver)
	registry.SetAuthAuditor(authAudit)
	return registry
}

var SupplierProviderWiringSet = wire.NewSet(
	ProvideSupplierGroupGuardReconciler,
	ProvideSupplierRateGuardService,
	ProvideSupplierProviderGroupMatcher,
	ProvideSupplierProviderSyncService,
	ProvideSupplierProviderCostReviewService,
	ProvideSupplierProviderRechargeRemoteClient,
	ProvideSupplierProviderRechargeService,
	NewSupplierProviderAuthAuditService,
	ProvideSupplierAccountRateGuardRateSyncer,
	NewSupplierAccountRateGuardService,
)
