package service

import "github.com/google/wire"

func ProvideSupplierAccountHealthGuardAccountStore(repo AccountRepository) supplierAccountHealthGuardAccountStore {
	return repo
}

func ProvideSupplierAccountHealthGuardTester(tester *AccountTestService) supplierAccountHealthGuardTester {
	return tester
}

func ProvideSupplierAccountHealthGuardService(
	repository SupplierAccountHealthGuardRepository,
	accountStore supplierAccountHealthGuardAccountStore,
	tester supplierAccountHealthGuardTester,
	recorder SupplierAccountHealthHistoryRecorder,
) *SupplierAccountHealthGuardService {
	svc := NewSupplierAccountHealthGuardService(repository, accountStore, tester)
	svc.SetHistoryRecorder(recorder)
	return svc
}

func ProvideSupplierAccountHealthTrendService(
	repository SupplierAccountHealthHistoryRepository,
) *SupplierAccountHealthTrendService {
	return NewSupplierAccountHealthTrendService(repository, repository)
}

func ProvideSupplierAutomationService(
	repo SupplierAutomationRepository,
	lock SupplierAutomationLock,
	syncer *SupplierProviderSyncService,
	dataRepo SupplierProviderDataRepository,
	rechargeSync *SupplierProviderRechargeService,
	rateGuard *SupplierRateGuardService,
	accountRateGuard *SupplierAccountRateGuardService,
	accountHealthGuard *SupplierAccountHealthGuardService,
	accountRateGuardRepo SupplierAccountRateGuardRepository,
) *SupplierAutomationService {
	svc := NewSupplierAutomationService(repo, lock, syncer, dataRepo)
	svc.SetMonitorSyncService(syncer)
	svc.SetRechargeSyncService(rechargeSync)
	svc.SetRateGuardService(rateGuard)
	svc.SetAccountRateGuardService(accountRateGuard)
	svc.SetAccountHealthGuardService(accountHealthGuard)
	svc.SetAccountRateGuardRepository(accountRateGuardRepo)
	return svc
}

var SupplierAccountHealthGuardWiringSet = wire.NewSet(
	ProvideSupplierAccountHealthGuardAccountStore,
	ProvideSupplierAccountHealthGuardTester,
	ProvideSupplierAccountHealthGuardService,
	ProvideSupplierAccountHealthTrendService,
	ProvideSupplierAutomationService,
)
