package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type supplierProviderDataRepoStub struct {
	accountsCalls      int
	groupsCalls        int
	balanceCalls       int
	costCalls          int
	costDays           []time.Time
	costValues         []float64
	createdRuns        []SupplierProviderSyncRun
	finishedRuns       []SupplierProviderSyncRun
	statusUpdates      []string
	groupStatusUpdates []string

	accountsErr    error
	groupsErr      error
	balanceErr     error
	costErr        error
	groupStatusErr error
	finishErr      error
	statusErr      error
}

func (r *supplierProviderDataRepoStub) ListAccounts(context.Context, SupplierProviderDataListParams) (SupplierProviderAccountListResult, error) {
	return SupplierProviderAccountListResult{}, nil
}
func (r *supplierProviderDataRepoStub) ListGroups(context.Context, SupplierProviderDataListParams) (SupplierProviderGroupListResult, error) {
	return SupplierProviderGroupListResult{}, nil
}
func (r *supplierProviderDataRepoStub) ListGroupHealthTrends(context.Context, SupplierProviderGroupHealthTrendParams) ([]SupplierProviderGroupHealthTrend, error) {
	return []SupplierProviderGroupHealthTrend{}, nil
}
func (r *supplierProviderDataRepoStub) ListLocalGroupHealthTrends(context.Context, SupplierProviderGroupHealthTrendParams) ([]SupplierProviderGroupHealthTrend, error) {
	return []SupplierProviderGroupHealthTrend{}, nil
}
func (*supplierProviderDataRepoStub) IsUniqueMatchedLocalAccount(context.Context, int64) (bool, error) {
	return false, nil
}
func (*supplierProviderDataRepoStub) GetLocalAccountEffectivePlatform(context.Context, int64) (string, error) {
	return "", nil
}
func (*supplierProviderDataRepoStub) GetLocalAccountPlatformOverride(context.Context, int64) (string, error) {
	return "", nil
}
func (*supplierProviderDataRepoStub) SetLocalAccountPlatformOverride(context.Context, int64, string) error {
	return nil
}
func (*supplierProviderDataRepoStub) ClearLocalAccountPlatformOverride(context.Context, int64) error {
	return nil
}

func (r *supplierProviderDataRepoStub) ListGroupsForAutoMatch(context.Context, int64) ([]SupplierProviderGroup, error) {
	return []SupplierProviderGroup{}, nil
}
func (r *supplierProviderDataRepoStub) GetGroupForAutoMatch(context.Context, int64) (SupplierProviderGroup, error) {
	return SupplierProviderGroup{}, nil
}
func (r *supplierProviderDataRepoStub) UpdateGroupMapping(context.Context, int64, *int64) error {
	return nil
}
func (r *supplierProviderDataRepoStub) DeleteGroup(context.Context, int64) error {
	return nil
}
func (r *supplierProviderDataRepoStub) DeleteAccount(context.Context, int64) error {
	return nil
}
func (r *supplierProviderDataRepoStub) ApplyAutoMatch(context.Context, int64, int64, string) (bool, error) {
	return false, nil
}
func (r *supplierProviderDataRepoStub) UpdateAutoMatchState(context.Context, int64, string, bool) error {
	return nil
}
func (r *supplierProviderDataRepoStub) UpdateAutoMatchIgnored(context.Context, int64, bool) error {
	return nil
}
func (r *supplierProviderDataRepoStub) AcknowledgeNameChange(context.Context, int64, string) error {
	return nil
}
func (r *supplierProviderDataRepoStub) ListMappingsByLocalGroup(context.Context, []int64) ([]SupplierProviderGroup, error) {
	return nil, nil
}
func (r *supplierProviderDataRepoStub) GetGroupForRateGuard(context.Context, int64) (SupplierProviderGroup, error) {
	return SupplierProviderGroup{}, nil
}
func (r *supplierProviderDataRepoStub) SelectRateGuard(context.Context, int64, string) error {
	return nil
}
func (r *supplierProviderDataRepoStub) ClearRateGuard(context.Context, int64, string) error {
	return nil
}
func (r *supplierProviderDataRepoStub) SetRateGuardIgnored(context.Context, int64, bool) error {
	return nil
}
func (r *supplierProviderDataRepoStub) ListRateGuardCandidates(context.Context) ([]SupplierRateGuardCandidate, error) {
	return nil, nil
}
func (r *supplierProviderDataRepoStub) ApplyRateGuard(context.Context, SupplierRateGuardApplyInput) (SupplierRateGuardApplyResult, error) {
	return SupplierRateGuardApplyResult{}, nil
}
func (r *supplierProviderDataRepoStub) MarkRateGuardChecked(context.Context, int64, time.Time) error {
	return nil
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
func (r *supplierProviderDataRepoStub) UpdateCost(_ context.Context, _ int64, cost float64, seenAt time.Time) error {
	r.costCalls++
	r.costDays = append(r.costDays, seenAt)
	r.costValues = append(r.costValues, cost)
	return r.costErr
}
func (r *supplierProviderDataRepoStub) CreateSyncRun(_ context.Context, run *SupplierProviderSyncRun) error {
	run.ID = int64(len(r.createdRuns) + 1)
	r.createdRuns = append(r.createdRuns, *run)
	return nil
}
func (r *supplierProviderDataRepoStub) FinishSyncRun(_ context.Context, run *SupplierProviderSyncRun) error {
	r.finishedRuns = append(r.finishedRuns, *run)
	return r.finishErr
}
func (r *supplierProviderDataRepoStub) UpdateSyncStatus(_ context.Context, _ int64, status, _ string, _ time.Time) error {
	r.statusUpdates = append(r.statusUpdates, status)
	return r.statusErr
}
func (r *supplierProviderDataRepoStub) UpdateGroupSyncStatus(_ context.Context, _ int64, status, _ string, _ time.Time) error {
	r.groupStatusUpdates = append(r.groupStatusUpdates, status)
	return r.groupStatusErr
}
func (r *supplierProviderDataRepoStub) Cleanup(context.Context, SupplierCleanupPolicy, time.Time, int) (SupplierCleanupCounts, error) {
	return SupplierCleanupCounts{}, nil
}

type supplierRemoteClientStub struct {
	passwords     []string
	authSources   []SupplierProviderAuthSource
	accounts      []SupplierProviderRemoteAccount
	costDays      []time.Time
	accountsCalls int
	groupsCalls   int
	balanceCalls  int
	costCalls     int

	accountsErr error
	groupsErr   error
	balanceErr  error
	costErr     error
	costFn      func(day time.Time) (float64, error)

	testCalls  []string
	testErr    error
	testResult *SupplierProviderEndpointTestResult
}

func (c *supplierRemoteClientStub) FetchAccounts(ctx context.Context, _ *SupplierProvider, password string) ([]SupplierProviderRemoteAccount, error) {
	c.accountsCalls++
	c.passwords = append(c.passwords, password)
	c.authSources = append(c.authSources, supplierProviderAuthSourceFromContext(ctx))
	if c.accountsErr != nil {
		return nil, c.accountsErr
	}
	if c.accounts != nil {
		return c.accounts, nil
	}
	return []SupplierProviderRemoteAccount{{Key: "account-1", Name: "Primary", Status: "active"}}, nil
}
func (c *supplierRemoteClientStub) FetchGroups(_ context.Context, _ *SupplierProvider, password string) ([]SupplierProviderRemoteGroup, error) {
	c.groupsCalls++
	c.passwords = append(c.passwords, password)
	if c.groupsErr != nil {
		return nil, c.groupsErr
	}
	return []SupplierProviderRemoteGroup{{Key: "group-1", Name: "VIP"}}, nil
}
func (c *supplierRemoteClientStub) FetchBalance(_ context.Context, _ *SupplierProvider, password string) (float64, error) {
	c.balanceCalls++
	c.passwords = append(c.passwords, password)
	if c.balanceErr != nil {
		return 0, c.balanceErr
	}
	return 123.5, nil
}
func (c *supplierRemoteClientStub) FetchCost(_ context.Context, _ *SupplierProvider, password string, day time.Time) (float64, error) {
	c.costCalls++
	c.passwords = append(c.passwords, password)
	c.costDays = append(c.costDays, day)
	if c.costErr != nil {
		return 0, c.costErr
	}
	if c.costFn != nil {
		return c.costFn(day)
	}
	return 45.6, nil
}
func (c *supplierRemoteClientStub) TestEndpoint(ctx context.Context, _ *SupplierProvider, password string, scope string) (SupplierProviderEndpointTestResult, error) {
	c.passwords = append(c.passwords, password)
	c.authSources = append(c.authSources, supplierProviderAuthSourceFromContext(ctx))
	c.testCalls = append(c.testCalls, scope)
	if c.testErr != nil {
		return SupplierProviderEndpointTestResult{}, c.testErr
	}
	if c.testResult != nil {
		return *c.testResult, nil
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
	acquired           bool
	acquiredByProvider map[int64]bool
	released           int
}

type supplierGroupAutoMatcherStub struct {
	calls []int64
	err   error
}

func (s *supplierGroupAutoMatcherStub) AutoMatch(_ context.Context, providerID int64) (SupplierGroupAutoMatchResult, error) {
	s.calls = append(s.calls, providerID)
	return SupplierGroupAutoMatchResult{ProviderID: providerID}, s.err
}

func (l *supplierSyncLockStub) TryAcquireSyncLock(_ context.Context, providerID int64, _ string, _ time.Duration) (bool, error) {
	if l.acquiredByProvider != nil {
		return l.acquiredByProvider[providerID], nil
	}
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

type supplierProviderRateDataRepoStub struct {
	*supplierProviderDataRepoStub
	knownKeys map[string]bool
	updates   map[string]float64
}

func (r *supplierProviderRateDataRepoStub) UpdateAccountRateSnapshot(_ context.Context, _ int64, upstreamKey string, rate float64, _ time.Time) (bool, error) {
	if !r.knownKeys[upstreamKey] {
		return false, nil
	}
	if r.updates == nil {
		r.updates = make(map[string]float64)
	}
	r.updates[upstreamKey] = rate
	return true, nil
}

func TestSupplierProviderSyncServiceSyncAccountRatesOnlyUpdatesKnownValidRates(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderRateDataRepoStub{
		supplierProviderDataRepoStub: &supplierProviderDataRepoStub{},
		knownKeys:                    map[string]bool{"known": true, "zero": true},
	}
	remote := &supplierRemoteClientStub{accounts: []SupplierProviderRemoteAccount{
		{Key: "known", Name: "已知账号", RateMultiplier: 1.25},
		{Key: "missing", Name: "未知账号", RateMultiplier: 2},
		{Key: "invalid", Name: "无效账号", RateMultiplier: -1},
		{Key: "zero", Name: "零倍率账号", RateMultiplier: 0},
	}}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAccountRates(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusPartial, result.Status)
	require.Equal(t, 4, result.CheckedCount)
	require.Equal(t, 2, result.UpdatedCount)
	require.Equal(t, 2, result.SkippedCount)
	require.ElementsMatch(t, []string{"known", "zero"}, result.UpdatedKeys)
	require.Equal(t, map[string]float64{"known": 1.25, "zero": 0}, dataRepo.updates)
	require.Zero(t, dataRepo.accountsCalls)
}

func TestSupplierProviderSyncServiceSyncAccountRatesDoesNotPersistWhenRemoteFails(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderRateDataRepoStub{
		supplierProviderDataRepoStub: &supplierProviderDataRepoStub{},
		knownKeys:                    map[string]bool{"known": true},
	}
	remote := &supplierRemoteClientStub{accountsErr: errors.New("upstream unavailable")}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAccountRates(context.Background(), 42, SupplierSyncTriggerScheduled)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Empty(t, dataRepo.updates)
	require.Zero(t, dataRepo.accountsCalls)
}

func TestSupplierProviderSyncServiceSyncAccountRatesDisablesProviderAfterAuthFailure(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderRateDataRepoStub{
		supplierProviderDataRepoStub: &supplierProviderDataRepoStub{},
		knownKeys:                    map[string]bool{"known": true},
	}
	remote := &supplierRemoteClientStub{accountsErr: &SupplierProviderAuthFailureError{Err: errors.New("login failed")}}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAccountRates(context.Background(), 42, SupplierSyncTriggerManual)

	require.Error(t, err)
	require.True(t, IsSupplierProviderAuthFailure(err))
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.False(t, providerRepo.items[0].Enabled)
	require.Equal(t, 1, providerRepo.disableAfterAuthFailureCalls)
	require.Empty(t, dataRepo.updates)
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
	require.Equal(t, []SupplierProviderAuthSource{SupplierProviderAuthSourceSync}, remote.authSources)
	require.Equal(t, 1, dataRepo.accountsCalls)
	require.Len(t, dataRepo.createdRuns, 1)
	require.Len(t, dataRepo.finishedRuns, 1)
	require.Equal(t, 1, lock.released)
}

func TestSupplierProviderSyncServiceSyncGroupsRunsAutoMatcherAfterPersisting(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	matcher := &supplierGroupAutoMatcherStub{}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	service.SetGroupMatcher(matcher)

	result, err := service.SyncGroups(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, 1, dataRepo.groupsCalls)
	require.Equal(t, []int64{42}, matcher.calls)
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

func TestSupplierProviderSyncServiceReportsDisabledProviderSeparately(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID:           42,
		Name:         "供应商甲",
		ProviderType: SupplierProviderTypeSub2API,
		Enabled:      false,
	}}}
	remote := &supplierRemoteClientStub{}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	_, err := service.SyncAll(context.Background(), 42, SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, "SUPPLIER_PROVIDER_DISABLED", infraerrors.Reason(err))
	require.Equal(t, "supplier provider is disabled", infraerrors.Message(err))
	require.Zero(t, remote.accountsCalls)
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
	require.Equal(t, []SupplierProviderAuthSource{SupplierProviderAuthSourceSync}, remote.authSources)
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
	require.Equal(t, []string{SupplierSyncStatusRunning, SupplierSyncStatusFailed}, dataRepo.groupStatusUpdates)
}

func TestSupplierProviderSyncServiceSyncAllStopsAfterAuthFailureAndDisablesProvider(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, Name: "供应商甲", ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{accountsErr: &SupplierProviderAuthFailureError{Err: errors.New("账号密码认证失败")}}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAll(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Len(t, result.Stages, 1)
	require.Equal(t, 1, remote.accountsCalls)
	require.Zero(t, remote.groupsCalls)
	require.Zero(t, remote.balanceCalls)
	require.Zero(t, remote.costCalls)
	require.False(t, providerRepo.items[0].Enabled)
	require.Equal(t, 1, providerRepo.disableAfterAuthFailureCalls)
	require.Contains(t, providerRepo.items[0].SyncMessage, "自动停用")
	require.Contains(t, providerRepo.items[0].SyncMessage, "手动重新启用")
}

func TestSupplierProviderSyncServiceSyncAllStopsAfterSessionFailureWithoutDisablingProvider(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	sessionErr := &SupplierProviderSessionFailureError{Err: errors.New("token cache get failed")}
	remote := &supplierRemoteClientStub{accountsErr: sessionErr}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAll(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Len(t, result.Stages, 1)
	require.Equal(t, 1, remote.accountsCalls)
	require.Zero(t, remote.groupsCalls)
	require.Zero(t, remote.balanceCalls)
	require.Zero(t, remote.costCalls)
	require.Zero(t, providerRepo.disableAfterAuthFailureCalls)
	require.True(t, providerRepo.items[0].Enabled)
	require.Contains(t, result.Message, "token cache get failed")
}

func TestSupplierProviderSyncServiceSyncAllEnabledSkipsProviderAfterAuthFailure(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, Name: "供应商甲", ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{accountsErr: &SupplierProviderAuthFailureError{Err: errors.New("登录失败")}}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	first, err := service.SyncAllEnabled(context.Background(), SupplierSyncTriggerScheduled)
	require.NoError(t, err)
	require.Equal(t, 1, first.ProcessedCount)
	require.Equal(t, 1, first.FailedCount)
	require.Equal(t, 1, remote.accountsCalls)

	second, err := service.SyncAllEnabled(context.Background(), SupplierSyncTriggerScheduled)
	require.NoError(t, err)
	require.Zero(t, second.ProcessedCount)
	require.Zero(t, second.FailedCount)
	require.Equal(t, 1, remote.accountsCalls)
}

func TestSupplierProviderSyncServiceSyncAccountsDisablesProviderAfterAuthFailure(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"}}}
	remote := &supplierRemoteClientStub{accountsErr: &SupplierProviderAuthFailureError{Err: errors.New("登录失败")}}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncAccounts(context.Background(), 42, SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.False(t, providerRepo.items[0].Enabled)
	require.Equal(t, 1, providerRepo.disableAfterAuthFailureCalls)
	require.Equal(t, 1, remote.accountsCalls)

	callsBeforeDisabledProviderRetry := remote.accountsCalls
	_, err = service.SyncAccounts(context.Background(), 42, SupplierSyncTriggerManual)
	require.Error(t, err)
	require.Equal(t, callsBeforeDisabledProviderRetry, remote.accountsCalls)
}

func TestSupplierProviderSyncServiceBackfillCostsStopsAfterAuthFailure(t *testing.T) {
	today := supplierCostBackfillToday()
	start := today.AddDate(0, 0, -2)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeNewAPI, Enabled: true, PasswordEncrypted: "secret"}}}
	remote := &supplierRemoteClientStub{costErr: &SupplierProviderAuthFailureError{Err: errors.New("登录失败")}}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.BackfillCosts(context.Background(), start.Format("2006-01-02"), today.Format("2006-01-02"), 42, SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, 1, remote.costCalls)
	require.False(t, providerRepo.items[0].Enabled)
	require.Equal(t, 1, providerRepo.disableAfterAuthFailureCalls)
	require.Len(t, result.Items, 3)
	require.Equal(t, SupplierSyncStatusFailed, result.Items[0].Status)
	require.Equal(t, SupplierSyncStatusSkipped, result.Items[1].Status)
	require.Equal(t, SupplierSyncStatusSkipped, result.Items[2].Status)
}

func TestSupplierProviderSyncServiceBackfillCostsStopsAfterSessionFailureWithoutDisablingProvider(t *testing.T) {
	today := supplierCostBackfillToday()
	start := today.AddDate(0, 0, -2)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 7, Name: "NewAPI-A", ProviderType: SupplierProviderTypeNewAPI, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	sessionErr := &SupplierProviderSessionFailureError{Err: errors.New("session lock unavailable")}
	remote := &supplierRemoteClientStub{costErr: sessionErr}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.BackfillCosts(context.Background(), start.Format("2006-01-02"), today.Format("2006-01-02"), 7, SupplierSyncTriggerManual)

	require.ErrorIs(t, err, sessionErr)
	require.Equal(t, 1, result.FailedCount)
	require.Equal(t, 2, result.SkippedCount)
	require.Zero(t, result.SuccessCount)
	require.Equal(t, 1, remote.costCalls)
	require.Zero(t, providerRepo.disableAfterAuthFailureCalls)
	require.True(t, providerRepo.items[0].Enabled)
	require.Len(t, result.Items, 3)
	require.Equal(t, SupplierSyncStatusFailed, result.Items[0].Status)
	require.Equal(t, SupplierSyncStatusSkipped, result.Items[1].Status)
	require.Equal(t, SupplierSyncStatusSkipped, result.Items[2].Status)
}

func TestSupplierProviderSyncServiceRecordsSuccessfulGroupStageIndependently(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 42, ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"}}}
	dataRepo := &supplierProviderDataRepoStub{}
	service := NewSupplierProviderSyncService(providerRepo, dataRepo, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.SyncGroups(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []string{SupplierSyncStatusRunning, SupplierSyncStatusSuccess}, dataRepo.groupStatusUpdates)
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

func TestSupplierProviderSyncServiceSyncAllEnabledSkipsLockedProvider(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{
		{ID: 1, Name: "正在同步的供应商", ProviderType: "sub2api", Enabled: true, PasswordEncrypted: "secret"},
		{ID: 2, Name: "待同步的供应商", ProviderType: "newapi", Enabled: true, PasswordEncrypted: "secret"},
	}}
	service := NewSupplierProviderSyncService(
		providerRepo,
		&supplierProviderDataRepoStub{},
		&supplierRemoteClientStub{},
		supplierEncryptorStub{},
		&supplierSyncLockStub{acquiredByProvider: map[int64]bool{1: false, 2: true}},
	)

	result, err := service.SyncAllEnabled(context.Background(), SupplierSyncTriggerScheduled)

	require.NoError(t, err)
	require.Equal(t, 2, result.ProcessedCount)
	require.Equal(t, 1, result.SuccessCount)
	require.Zero(t, result.FailedCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Len(t, result.Results, 2)
	require.Equal(t, SupplierSyncStatusSkipped, result.Results[0].Status)
	require.Contains(t, result.Results[0].Message, "下次")
	require.Equal(t, SupplierSyncStatusSuccess, result.Results[1].Status)
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
	require.Equal(t, []SupplierProviderAuthSource{SupplierProviderAuthSourceEndpointTest}, remote.authSources)
	require.Equal(t, []string{SupplierSyncScopeBalance}, remote.testCalls)
	require.Empty(t, dataRepo.createdRuns)
	require.Empty(t, dataRepo.finishedRuns)
	require.Zero(t, dataRepo.balanceCalls)
}

func TestSupplierProviderSyncServiceTestEndpointDisablesProviderAfterUnauthorizedResponse(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	remote := &supplierRemoteClientStub{testResult: &SupplierProviderEndpointTestResult{
		Scope:      SupplierSyncScopeBalance,
		Endpoint:   "/test/balance",
		HTTPStatus: http.StatusUnauthorized,
	}}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.TestEndpoint(context.Background(), 42, SupplierSyncScopeBalance)

	require.Error(t, err)
	require.True(t, IsSupplierProviderAuthFailure(err))
	require.Equal(t, supplierProviderAuthFailureDisableMessage, result.Error)
	require.False(t, providerRepo.items[0].Enabled)
	require.Equal(t, 1, providerRepo.disableAfterAuthFailureCalls)
}

func TestSupplierProviderSyncServiceBackfillCostsNewAPIPullsEachDay(t *testing.T) {
	today := supplierCostBackfillToday()
	start := today.AddDate(0, 0, -2)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 7, Name: "NewAPI-A", ProviderType: SupplierProviderTypeNewAPI, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{
		costFn: func(day time.Time) (float64, error) {
			return float64(day.Day()) + 0.5, nil
		},
	}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.BackfillCosts(context.Background(), start.Format("2006-01-02"), today.Format("2006-01-02"), 7, SupplierSyncTriggerManual)
	require.NoError(t, err)
	require.Equal(t, 3, result.DayCount)
	require.Equal(t, 1, result.ProviderCount)
	require.Equal(t, 3, result.SuccessCount)
	require.Zero(t, result.FailedCount)
	require.Zero(t, result.SkippedCount)
	require.Len(t, remote.costDays, 3)
	require.Len(t, dataRepo.costDays, 3)
	require.Equal(t, start.Format("2006-01-02"), dataRepo.costDays[0].Format("2006-01-02"))
	require.Equal(t, today.Format("2006-01-02"), dataRepo.costDays[2].Format("2006-01-02"))
	require.Equal(t, float64(start.Day())+0.5, dataRepo.costValues[0])
	require.Len(t, dataRepo.createdRuns, 1)
	require.Len(t, dataRepo.finishedRuns, 1)
	require.Equal(t, SupplierSyncScopeCost, dataRepo.finishedRuns[0].SyncScope)
	require.Equal(t, SupplierSyncStatusSuccess, dataRepo.finishedRuns[0].Status)
}

func TestSupplierProviderSyncServiceBackfillCostsSub2APISkipsHistoryOnlyToday(t *testing.T) {
	today := supplierCostBackfillToday()
	start := today.AddDate(0, 0, -1)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 9, Name: "Sub2API-B", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.BackfillCosts(context.Background(), start.Format("2006-01-02"), today.Format("2006-01-02"), 9, SupplierSyncTriggerManual)
	require.NoError(t, err)
	require.Equal(t, 2, result.DayCount)
	require.Equal(t, 1, result.SuccessCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Zero(t, result.FailedCount)
	require.Len(t, remote.costDays, 1)
	require.Equal(t, today.Format("2006-01-02"), remote.costDays[0].Format("2006-01-02"))
	require.Len(t, dataRepo.costDays, 1)
	require.Equal(t, today.Format("2006-01-02"), dataRepo.costDays[0].Format("2006-01-02"))

	var skipped, success int
	for _, item := range result.Items {
		switch item.Status {
		case SupplierSyncStatusSkipped:
			skipped++
			require.Contains(t, item.Message, "仅支持回补当天")
			require.Equal(t, start.Format("2006-01-02"), item.Date)
		case SupplierSyncStatusSuccess:
			success++
			require.Equal(t, today.Format("2006-01-02"), item.Date)
		}
	}
	require.Equal(t, 1, skipped)
	require.Equal(t, 1, success)
}

func TestSupplierProviderSyncServiceBackfillCostsRejectsInvalidRange(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 1, ProviderType: SupplierProviderTypeNewAPI, Enabled: true, PasswordEncrypted: "secret",
	}}}
	svc := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, &supplierRemoteClientStub{}, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	_, err := svc.BackfillCosts(context.Background(), "2026-07-20", "2026-07-10", 1, SupplierSyncTriggerManual)
	require.Error(t, err)
	require.Contains(t, err.Error(), "end_date")
}

func TestSupplierProviderSyncServiceBackfillCostsMarksLockedProviderAsSkipped(t *testing.T) {
	today := supplierCostBackfillToday()
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 11, Name: "Locked", ProviderType: SupplierProviderTypeNewAPI, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: false})

	result, err := svc.BackfillCosts(context.Background(), today.Format("2006-01-02"), today.Format("2006-01-02"), 11, SupplierSyncTriggerManual)
	require.NoError(t, err)
	require.Equal(t, 1, result.SkippedCount)
	require.Zero(t, result.SuccessCount)
	require.Empty(t, remote.costDays)
	require.Empty(t, dataRepo.costDays)
	require.Equal(t, SupplierSyncStatusSkipped, result.Items[0].Status)
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
