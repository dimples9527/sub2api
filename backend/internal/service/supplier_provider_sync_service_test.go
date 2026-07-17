package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type supplierProviderDataRepoStub struct {
	accountsCalls int
	groupsCalls   int
	balanceCalls  int
	costCalls     int
	createdRuns   []SupplierProviderSyncRun
	finishedRuns  []SupplierProviderSyncRun
	statusUpdates []string

	accountsErr error
	groupsErr   error
	balanceErr  error
	costErr     error
}

func (r *supplierProviderDataRepoStub) ListAccounts(context.Context, SupplierProviderDataListParams) (SupplierProviderAccountListResult, error) {
	return SupplierProviderAccountListResult{}, nil
}
func (r *supplierProviderDataRepoStub) ListGroups(context.Context, SupplierProviderDataListParams) (SupplierProviderGroupListResult, error) {
	return SupplierProviderGroupListResult{}, nil
}
func (r *supplierProviderDataRepoStub) ReplaceAccounts(_ context.Context, _ int64, items []SupplierProviderRemoteAccount, _ time.Time) (SupplierSyncCounts, error) {
	r.accountsCalls++
	if r.accountsErr != nil {
		return SupplierSyncCounts{}, r.accountsErr
	}
	return SupplierSyncCounts{CheckedCount: len(items), UpdatedCount: len(items)}, nil
}
func (r *supplierProviderDataRepoStub) ReplaceGroups(_ context.Context, _ int64, items []SupplierProviderRemoteGroup, _ time.Time) (SupplierSyncCounts, error) {
	r.groupsCalls++
	if r.groupsErr != nil {
		return SupplierSyncCounts{}, r.groupsErr
	}
	return SupplierSyncCounts{CheckedCount: len(items), UpdatedCount: len(items)}, nil
}
func (r *supplierProviderDataRepoStub) UpdateBalance(context.Context, int64, float64, time.Time) error {
	r.balanceCalls++
	return r.balanceErr
}
func (r *supplierProviderDataRepoStub) UpdateCost(context.Context, int64, float64, time.Time) error {
	r.costCalls++
	return r.costErr
}
func (r *supplierProviderDataRepoStub) CreateSyncRun(_ context.Context, run *SupplierProviderSyncRun) error {
	run.ID = int64(len(r.createdRuns) + 1)
	r.createdRuns = append(r.createdRuns, *run)
	return nil
}
func (r *supplierProviderDataRepoStub) FinishSyncRun(_ context.Context, run *SupplierProviderSyncRun) error {
	r.finishedRuns = append(r.finishedRuns, *run)
	return nil
}
func (r *supplierProviderDataRepoStub) UpdateSyncStatus(_ context.Context, _ int64, status, _ string, _ time.Time) error {
	r.statusUpdates = append(r.statusUpdates, status)
	return nil
}
func (r *supplierProviderDataRepoStub) Cleanup(context.Context, SupplierCleanupPolicy, time.Time, int) (SupplierCleanupCounts, error) {
	return SupplierCleanupCounts{}, nil
}

type supplierRemoteClientStub struct {
	passwords []string

	accountsErr error
	groupsErr   error
	balanceErr  error
	costErr     error

	testCalls []string
	testErr   error
}

func (c *supplierRemoteClientStub) FetchAccounts(_ context.Context, _ *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error) {
	c.passwords = append(c.passwords, password)
	if c.accountsErr != nil {
		return nil, c.accountsErr
	}
	return []SupplierProviderRemoteAccount{{Key: "account-1", Name: "Primary", Status: "active"}}, nil
}
func (c *supplierRemoteClientStub) FetchGroups(_ context.Context, _ *SupplierProvider, password string) ([]SupplierProviderRemoteGroup, error) {
	c.passwords = append(c.passwords, password)
	if c.groupsErr != nil {
		return nil, c.groupsErr
	}
	return []SupplierProviderRemoteGroup{{Key: "group-1", Name: "VIP"}}, nil
}
func (c *supplierRemoteClientStub) FetchBalance(_ context.Context, _ *SupplierProvider, password string) (float64, error) {
	c.passwords = append(c.passwords, password)
	if c.balanceErr != nil {
		return 0, c.balanceErr
	}
	return 123.5, nil
}
func (c *supplierRemoteClientStub) FetchCost(_ context.Context, _ *SupplierProvider, password string, _ time.Time) (float64, error) {
	c.passwords = append(c.passwords, password)
	if c.costErr != nil {
		return 0, c.costErr
	}
	return 45.6, nil
}
func (c *supplierRemoteClientStub) TestEndpoint(_ context.Context, _ *SupplierProvider, password string, scope string) (SupplierProviderEndpointTestResult, error) {
	c.passwords = append(c.passwords, password)
	c.testCalls = append(c.testCalls, scope)
	if c.testErr != nil {
		return SupplierProviderEndpointTestResult{}, c.testErr
	}
	return SupplierProviderEndpointTestResult{
		Scope:           scope,
		Endpoint:        "/test/" + scope,
		HTTPStatus:      200,
		DurationMS:      12,
		ResponseSummary: `{"code":0}`,
		ParsedData:      map[string]any{"ok": true},
	}, nil
}

type supplierSyncLockStub struct {
	acquired bool
	released int
}

func (l *supplierSyncLockStub) TryAcquireSyncLock(context.Context, int64, string, time.Duration) (bool, error) {
	return l.acquired, nil
}
func (l *supplierSyncLockStub) ReleaseSyncLock(context.Context, int64, string) error {
	l.released++
	return nil
}

type supplierDecryptFailureEncryptor struct{}

func (supplierDecryptFailureEncryptor) Encrypt(value string) (string, error) { return value, nil }
func (supplierDecryptFailureEncryptor) Decrypt(string) (string, error) {
	return "", errors.New("cipher: message authentication failed")
}

func TestSupplierProviderSyncServiceSyncAccountsDecryptsPasswordAndPersists(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID:                42,
		Code:              "supplier-a",
		ProviderType:      "sub2api",
		Enabled:           true,
		PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{}
	lock := &supplierSyncLockStub{acquired: true}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, lock)

	result, err := service.SyncAccounts(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, SupplierSyncScopeAccounts, result.Scope)
	require.Equal(t, []string{"secret"}, remote.passwords)
	require.Equal(t, 1, dataRepo.accountsCalls)
	require.Len(t, dataRepo.createdRuns, 1)
	require.Len(t, dataRepo.finishedRuns, 1)
	require.Equal(t, 1, lock.released)
}

func TestSupplierProviderSyncServiceUsesStoredCredentialWhenDecryptFails(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID:                42,
		Code:              "supplier-a",
		ProviderType:      "sub2api",
		Enabled:           true,
		PasswordEncrypted: "plain-secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierDecryptFailureEncryptor{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAccounts(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []string{"plain-secret"}, remote.passwords)
	require.Equal(t, 1, dataRepo.accountsCalls)
}

func TestSupplierProviderSyncServiceRejectsUnsupportedProviderType(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, ProviderType: "custom", Enabled: true}}}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	_, err := service.SyncAccounts(context.Background(), 42, SupplierSyncTriggerManual)

	require.Error(t, err)
}

func TestSupplierProviderSyncServiceAllowsNewAPIProviderType(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID:                42,
		Code:              "supplier-newapi",
		ProviderType:      "newapi",
		Enabled:           true,
		PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAccounts(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []string{"secret"}, remote.passwords)
	require.Equal(t, 1, dataRepo.accountsCalls)
}

func TestSupplierProviderSyncServiceSyncAllReturnsPartialWhenOneStageFails(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{groupsErr: errors.New("upstream groups unavailable")}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAll(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusPartial, result.Status)
	require.Len(t, result.Stages, 4)
	require.Equal(t, 1, dataRepo.accountsCalls)
	require.Equal(t, 1, dataRepo.balanceCalls)
	require.Equal(t, 1, dataRepo.costCalls)
	require.Len(t, dataRepo.createdRuns, 1)
}

func TestSupplierProviderSyncServiceSyncAllEnabledContinuesAfterProviderFailure(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{
		{ID: 1, ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"},
		{ID: 2, ProviderType: "newapi", Enabled: true, PasswordEncrypted: "secret"},
	}}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAllEnabled(context.Background(), SupplierSyncTriggerScheduled)

	require.NoError(t, err)
	require.Equal(t, 2, result.ProcessedCount)
	require.Equal(t, 2, result.SuccessCount)
	require.Equal(t, 0, result.FailedCount)
	require.Len(t, result.Results, 2)
}

func TestSupplierProviderSyncServiceRejectsConcurrentProviderSync(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"}}}
	remote := &supplierRemoteClientStub{}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: false})

	_, err := service.SyncAccounts(context.Background(), 42, SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Empty(t, remote.passwords)
}

func TestSupplierProviderSyncServiceTestsEndpointWithoutPersisting(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.TestEndpoint(context.Background(), 42, SupplierSyncScopeBalance)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncScopeBalance, result.Scope)
	require.Equal(t, "/test/balance", result.Endpoint)
	require.Equal(t, []string{"secret"}, remote.passwords)
	require.Equal(t, []string{SupplierSyncScopeBalance}, remote.testCalls)
	require.Empty(t, dataRepo.createdRuns)
	require.Empty(t, dataRepo.finishedRuns)
	require.Zero(t, dataRepo.balanceCalls)
}

func TestSupplierProviderServiceUpdateClearsTokenWhenAuthConfigurationChanges(t *testing.T) {
	repo := &supplierProviderRepoStub{next: 1, items: []*SupplierProvider{{ID: 1, Code: "primary", ProviderType: "sub2api", BaseURL: "https://old.example.com", Email: "old@example.com", PasswordEncrypted: "encrypted:old"}}}
	cache := newSupplierSub2APIFakeTokenCache()
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	service.SetTokenCache(cache)
	params := validSupplierProviderParams()
	params.Email = "new@example.com"

	_, err := service.Update(context.Background(), 1, params)

	require.NoError(t, err)
	require.Equal(t, 1, cache.deleteCalls)
}

func TestSupplierProviderServiceUpdateKeepsTokenForSortOnlyChange(t *testing.T) {
	repo := &supplierProviderRepoStub{next: 1, items: []*SupplierProvider{{ID: 1, Code: "primary", Name: "主供应商", ProviderType: "sub2api", BaseURL: "https://supplier.example.com", Email: "", PasswordEncrypted: "encrypted:secret", AccountRateMultiplierScale: 1}}}
	cache := newSupplierSub2APIFakeTokenCache()
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	service.SetTokenCache(cache)
	params := validSupplierProviderParams()
	params.Password = ""
	params.SortOrder = 100

	_, err := service.Update(context.Background(), 1, params)

	require.NoError(t, err)
	require.Equal(t, 0, cache.deleteCalls)
}

func TestSupplierProviderServiceDeleteClearsToken(t *testing.T) {
	repo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Code: "primary", ProviderType: "sub2api"}}}
	cache := newSupplierSub2APIFakeTokenCache()
	service := NewSupplierProviderService(repo, supplierEncryptorStub{})
	service.SetTokenCache(cache)

	require.NoError(t, service.Delete(context.Background(), 1))
	require.Equal(t, 1, cache.deleteCalls)
}
