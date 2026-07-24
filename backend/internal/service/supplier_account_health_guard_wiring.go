package service

import "github.com/google/wire"

func ProvideSupplierAccountHealthGuardAccountStore(repo AccountRepository) supplierAccountHealthGuardAccountStore {
	return repo
}

func ProvideSupplierAccountHealthGuardTester(tester *AccountTestService) supplierAccountHealthGuardTester {
	return tester
}

func ProvideSupplierAutomationService(
	repo SupplierAutomationRepository,
	lock SupplierAutomationLock,
	syncer SupplierProviderBatchSyncer,
	dataRepo SupplierProviderDataRepository,
	rateGuard *SupplierRateGuardService,
	accountRateGuard *SupplierAccountRateGuardService,
	accountHealthGuard *SupplierAccountHealthGuardService,
	accountRateGuardRepo SupplierAccountRateGuardRepository,
) *SupplierAutomationService {
	svc := NewSupplierAutomationService(repo, lock, syncer, dataRepo)
	svc.SetRateGuardService(rateGuard)
	svc.SetAccountRateGuardService(accountRateGuard)
	svc.SetAccountHealthGuardService(accountHealthGuard)
	svc.SetAccountRateGuardRepository(accountRateGuardRepo)
	return svc
}

var SupplierAccountHealthGuardWiringSet = wire.NewSet(
	ProvideSupplierAccountHealthGuardAccountStore,
	ProvideSupplierAccountHealthGuardTester,
	NewSupplierAccountHealthGuardService,
	ProvideSupplierAutomationService,
)
