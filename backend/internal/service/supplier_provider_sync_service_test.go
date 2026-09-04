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
	balanceDeltaCalls  int
	balanceDeltaValue  float64
	balanceDeltaOK     bool
	balanceDeltaErr    error
	createdRuns        []SupplierProviderSyncRun
	finishedRuns       []SupplierProviderSyncRun
	statusUpdates      []string
	groupStatusUpdates []string
	groupChanges       SupplierProviderGroupChangeSummary
	cleanupPolicy      SupplierCleanupPolicy
	cleanupCounts      SupplierCleanupCounts

	localCostCalls   int
	localCostValue   float64
	localCostOK      bool
	localCostErr     error
	detailedRawCosts []*float64
	detailedWarnings []*string
	reviewInputs     []SupplierProviderCostReviewSyncInput

	accountsErr    error
	groupsErr      error
	balanceErr     error
	costErr        error
	groupStatusErr error
	finishErr      error
	statusErr      error
}

type supplierProviderRechargeRepoStub struct {
	listResult SupplierProviderRechargeListResult
	listErr    error
	listCalls  []SupplierProviderRechargeListParams
}

func (r *supplierProviderRechargeRepoStub) Upsert(context.Context, int64, []SupplierProviderRechargeRecord) error {
	return nil
}

func (r *supplierProviderRechargeRepoStub) List(_ context.Context, params SupplierProviderRechargeListParams) (SupplierProviderRechargeListResult, error) {
	r.listCalls = append(r.listCalls, params)
	return r.listResult, r.listErr
}

func (r *supplierProviderRechargeRepoStub) Sum(context.Context, int64, time.Time, time.Time) (float64, error) {
	return 0, nil
}

func (r *supplierProviderRechargeRepoStub) HasRecords(context.Context, int64) (bool, error) {
	return false, nil
}

func (r *supplierProviderDataRepoStub) ListAccounts(context.Context, SupplierProviderDataListParams) (SupplierProviderAccountListResult, error) {
	return SupplierProviderAccountListResult{}, nil
}
func (r *supplierProviderDataRepoStub) ListGroups(context.Context, SupplierProviderDataListParams) (SupplierProviderGroupListResult, error) {
	return SupplierProviderGroupListResult{}, nil
}
func (r *supplierProviderDataRepoStub) ListMonitorTargets(context.Context, SupplierProviderMonitorTargetListParams) (SupplierProviderMonitorTargetListResult, error) {
	return SupplierProviderMonitorTargetListResult{}, nil
}
func (r *supplierProviderDataRepoStub) ListBindableLocalAccounts(context.Context, SupplierBindableLocalAccountListParams) (SupplierBindableLocalAccountListResult, error) {
	return SupplierBindableLocalAccountListResult{}, nil
}
func (r *supplierProviderDataRepoStub) BindMonitorTarget(context.Context, int64, int64) error {
	return nil
}
func (r *supplierProviderDataRepoStub) UnbindMonitorTarget(context.Context, int64) error {
	return nil
}
func (r *supplierProviderDataRepoStub) ApplyMonitorAutoMatch(_ context.Context, monitorTargetID, localAccountID int64) error {
	return nil
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
func (r *supplierProviderDataRepoStub) SetRateGuardEnabled(context.Context, int64, bool) error {
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
func (r *supplierProviderDataRepoStub) ReplaceGroups(_ context.Context, _ int64, items []SupplierProviderRemoteGroup, _ time.Time) (SupplierProviderGroupReplaceResult, error) {
	r.groupsCalls++
	if r.groupsErr != nil {
		return SupplierProviderGroupReplaceResult{}, r.groupsErr
	}
	return SupplierProviderGroupReplaceResult{
		Counts:  SupplierSyncCounts{CheckedCount: len(items), UpdatedCount: len(items)},
		Changes: r.groupChanges,
	}, nil
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
func (r *supplierProviderDataRepoStub) UpdateCostDetailed(_ context.Context, _ int64, cost float64, rawUpstream *float64, warning *string, seenAt time.Time) error {
	r.costCalls++
	r.costDays = append(r.costDays, seenAt)
	r.costValues = append(r.costValues, cost)
	r.detailedRawCosts = append(r.detailedRawCosts, rawUpstream)
	r.detailedWarnings = append(r.detailedWarnings, warning)
	return r.costErr
}
func (r *supplierProviderDataRepoStub) UpdateCostDetailedWithReview(ctx context.Context, providerID int64, rawUpstream *float64, warning *string, seenAt time.Time, review SupplierProviderCostReviewSyncInput) (float64, error) {
	r.reviewInputs = append(r.reviewInputs, review)
	// 真实实现写入的是核对记录的生效成本；这里按服务层交出的值回写，
	// 待审批记录如何取值由 repository 层测试覆盖。
	return review.EffectiveCost, r.UpdateCostDetailed(ctx, providerID, review.EffectiveCost, rawUpstream, warning, seenAt)
}
func (r *supplierProviderDataRepoStub) GetLocalCostForDay(context.Context, int64, time.Time) (float64, bool, error) {
	r.localCostCalls++
	return r.localCostValue, r.localCostOK, r.localCostErr
}
func (r *supplierProviderDataRepoStub) GetBalanceDeltaForDay(context.Context, int64, time.Time) (float64, bool, error) {
	r.balanceDeltaCalls++
	return r.balanceDeltaValue, r.balanceDeltaOK, r.balanceDeltaErr
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
func (r *supplierProviderDataRepoStub) Cleanup(_ context.Context, policy SupplierCleanupPolicy, _ time.Time, _ int) (SupplierCleanupCounts, error) {
	r.cleanupPolicy = policy
	return r.cleanupCounts, nil
}

type supplierRemoteClientStub struct {
	passwords     []string
	authSources   []SupplierProviderAuthSource
	accounts      []SupplierProviderRemoteAccount
	costDays      []time.Time
	reauthCalls   int
	reauthToken   SupplierProviderAuthToken
	accountsCalls int
	groupsCalls   int
	balanceCalls  int
	costCalls     int
	rechargeCalls int
	rechargeDays  []time.Time

	accountsErr error
	groupsErr   error
	balanceErr  error
	costErr     error
	costFn      func(day time.Time) (float64, error)
	rechargeErr error
	recharge    float64

	testCalls  []string
	testErr    error
	testResult *SupplierProviderEndpointTestResult
}

func (c *supplierRemoteClientStub) Reauthenticate(ctx context.Context, _ *SupplierProvider, password string) (SupplierProviderAuthToken, error) {
	c.reauthCalls++
	c.passwords = append(c.passwords, password)
	c.authSources = append(c.authSources, supplierProviderAuthSourceFromContext(ctx))
	return c.reauthToken, nil
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
func (c *supplierRemoteClientStub) FetchRechargeAmount(_ context.Context, _ *SupplierProvider, password string, day time.Time) (float64, error) {
	c.rechargeCalls++
	c.passwords = append(c.passwords, password)
	c.rechargeDays = append(c.rechargeDays, day)
	if c.rechargeErr != nil {
		return 0, c.rechargeErr
	}
	return c.recharge, nil
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

func TestSupplierProviderSyncServiceReauthenticatesCookieSessionNewAPI(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID:                42,
		ProviderType:      SupplierProviderTypeNewAPI,
		NewAPIAuthMode:    SupplierNewAPIAuthModeCookieSession,
		Enabled:           true,
		PasswordEncrypted: "secret",
	}}}
	remote := &supplierRemoteClientStub{reauthToken: SupplierProviderAuthToken{CookieHeader: "session=renewed"}}
	service := NewSupplierProviderSyncService(providerRepo, &supplierProviderDataRepoStub{}, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := service.RefreshToken(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, "session=renewed", result.CookieHeader)
	require.Equal(t, 1, remote.reauthCalls)
	require.Equal(t, []string{"secret"}, remote.passwords)
	require.Equal(t, []SupplierProviderAuthSource{SupplierProviderAuthSourceManual}, remote.authSources)
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

func TestSupplierProviderEndpointAuthFailureIgnoresProbeBlockedForbidden(t *testing.T) {
	result := SupplierProviderEndpointTestResult{
		HTTPStatus:      http.StatusForbidden,
		Error:           `supplier sub2api monitor failed with HTTP 403: {"error":{"message":"Probe, monitoring, and test traffic are disabled by site policy.","type":"probe_blocked"}}`,
		ResponseSummary: `{"error":{"message":"Probe, monitoring, and test traffic are disabled by site policy.","type":"probe_blocked"}}`,
	}

	require.False(t, supplierProviderEndpointAuthFailure(result))
}

func TestSupplierProviderEndpointAuthFailureKeepsInvalidTokenForbidden(t *testing.T) {
	result := SupplierProviderEndpointTestResult{
		HTTPStatus:      http.StatusForbidden,
		Error:           `supplier sub2api accounts failed with HTTP 403: {"code":403,"message":"invalid token"}`,
		ResponseSummary: `{"code":403,"message":"invalid token"}`,
	}

	require.True(t, supplierProviderEndpointAuthFailure(result))
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

func TestSupplierProviderSyncServiceBackfillCostsSub2APIUsesRechargeAdjustedBalanceForHistory(t *testing.T) {
	day := supplierCostBackfillToday().AddDate(0, 0, -1)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 9, Name: "Sub2API-B", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{
		balanceDeltaValue: -20,
		balanceDeltaOK:    true,
	}
	remote := &supplierRemoteClientStub{recharge: 50}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.BackfillCosts(context.Background(), day.Format("2006-01-02"), day.Format("2006-01-02"), 9, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Zero(t, result.SkippedCount)
	require.Zero(t, remote.costCalls)
	require.Equal(t, 1, remote.rechargeCalls)
	require.Equal(t, []float64{30}, dataRepo.costValues)
	require.Len(t, result.Items, 1)
	require.Equal(t, SupplierSyncStatusSuccess, result.Items[0].Status)
	require.Equal(t, 30.0, result.Items[0].Cost)
	require.Contains(t, result.Items[0].Message, "历史日期不请求 Sub2API 当天成本接口")
}

func TestSupplierProviderSyncServiceBackfillCostsUsesRechargeAdjustedBalanceFallback(t *testing.T) {
	day := supplierCostBackfillToday().AddDate(0, 0, -1)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 7, Name: "NewAPI-A", ProviderType: SupplierProviderTypeNewAPI, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{
		balanceDeltaValue: -20,
		balanceDeltaOK:    true,
	}
	remote := &supplierRemoteClientStub{
		costErr:  errors.New("成本接口不可用"),
		recharge: 50,
	}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.BackfillCosts(context.Background(), day.Format("2006-01-02"), day.Format("2006-01-02"), 7, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Zero(t, result.FailedCount)
	require.Equal(t, []float64{30}, dataRepo.costValues)
	require.Equal(t, 1, remote.rechargeCalls)
	require.Equal(t, day.Format("2006-01-02"), remote.rechargeDays[0].Format("2006-01-02"))
	require.Len(t, result.Items, 1)
	require.Equal(t, 30.0, result.Items[0].Cost)
	require.Contains(t, result.Items[0].Message, "余额差 -20.00 + 充值 50.00")
}

func TestSupplierProviderSyncServiceBackfillCostsWritesCostReviewForBalanceOnlyDay(t *testing.T) {
	day := supplierCostBackfillToday().AddDate(0, 0, -1)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 9, Name: "Sub2API-B", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{
		balanceDeltaValue: -20,
		balanceDeltaOK:    true,
		localCostValue:    21.68,
		localCostOK:       true,
	}
	remote := &supplierRemoteClientStub{recharge: 50}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetCostReviewService(NewSupplierProviderCostReviewService(nil))

	result, err := svc.BackfillCosts(context.Background(), day.Format("2006-01-02"), day.Format("2006-01-02"), 9, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	// 回填的历史日期同样要留下核对记录，否则核对页无从审批、成本分析拿不到计算成本。
	require.Len(t, dataRepo.reviewInputs, 1)
	review := dataRepo.reviewInputs[0]
	require.Nil(t, review.UpstreamCost)
	require.NotNil(t, review.CalculatedCost)
	require.Equal(t, 30.0, *review.CalculatedCost)
	require.NotNil(t, review.LocalCost)
	require.Equal(t, 21.68, *review.LocalCost)
	require.Equal(t, 30.0, review.EffectiveCost)
	require.Equal(t, day.Format("2006-01-02"), review.StatDate.Format("2006-01-02"))
	require.NotNil(t, review.SyncRunID)
	require.Equal(t, dataRepo.createdRuns[0].ID, *review.SyncRunID)
}

func TestSupplierProviderSyncServiceBackfillCostsWritesCostReviewWithUpstreamCost(t *testing.T) {
	day := supplierCostBackfillToday().AddDate(0, 0, -1)
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 7, Name: "NewAPI-A", ProviderType: SupplierProviderTypeNewAPI, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{
		balanceDeltaValue: 30,
		balanceDeltaOK:    true,
	}
	remote := &supplierRemoteClientStub{recharge: 10}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetCostReviewService(NewSupplierProviderCostReviewService(nil))

	result, err := svc.BackfillCosts(context.Background(), day.Format("2006-01-02"), day.Format("2006-01-02"), 7, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Len(t, dataRepo.reviewInputs, 1)
	review := dataRepo.reviewInputs[0]
	require.NotNil(t, review.UpstreamCost)
	require.Equal(t, 45.6, *review.UpstreamCost)
	require.NotNil(t, review.CalculatedCost)
	require.Equal(t, 40.0, *review.CalculatedCost)
	// 偏差 12% 未超阈值，回填生效成本仍是上游值。
	require.Equal(t, 45.6, review.EffectiveCost)
	require.Equal(t, 45.6, result.Items[0].Cost)
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
			require.Contains(t, item.Message, "没有可用的充值修正余额成本")
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

func TestSupplierCalculatedCostForDay(t *testing.T) {
	day := time.Date(2026, 8, 18, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	cases := []struct {
		name              string
		balanceDelta      float64
		balanceOK         bool
		recharge          float64
		want              *supplierCalculatedCost
		wantRechargeCalls int
	}{
		{name: "余额减少", balanceDelta: 20, balanceOK: true, want: &supplierCalculatedCost{Cost: 20, BalanceDelta: 20}, wantRechargeCalls: 1},
		{name: "余额增加但充值更多", balanceDelta: -20, balanceOK: true, recharge: 50, want: &supplierCalculatedCost{Cost: 30, BalanceDelta: -20, RechargeAmount: 50}, wantRechargeCalls: 1},
		{name: "余额未变且无充值", balanceOK: true, wantRechargeCalls: 1},
		{name: "缺少余额快照", recharge: 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &SupplierProvider{ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret"}
			dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: tc.balanceDelta, balanceDeltaOK: tc.balanceOK}
			remote := &supplierRemoteClientStub{recharge: tc.recharge}
			svc := NewSupplierProviderSyncService(&supplierProviderRepoStub{items: []*SupplierProvider{provider}}, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

			got, err := svc.calculatedCostForDay(context.Background(), provider, "secret", day)

			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.wantRechargeCalls, remote.rechargeCalls)
		})
	}
}

func TestSupplierProviderSyncServiceCostFallbackEstimatesCostWhenRequestFails(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: 20, balanceDeltaOK: true}
	remote := &supplierRemoteClientStub{costErr: errors.New("成本接口不可用")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, 1, remote.costCalls)
	require.Equal(t, 1, dataRepo.balanceDeltaCalls)
	require.Equal(t, 1, dataRepo.costCalls)
	require.Equal(t, []float64{20}, dataRepo.costValues)
	// 上游成本取不到时不能伪造原始值，只留兜底提示
	require.Len(t, dataRepo.detailedRawCosts, 1)
	require.Nil(t, dataRepo.detailedRawCosts[0])
	require.Len(t, dataRepo.detailedWarnings, 1)
	require.NotNil(t, dataRepo.detailedWarnings[0])
	require.Contains(t, *dataRepo.detailedWarnings[0], "使用计算成本兜底")
}

func TestSupplierProviderSyncServiceCostFallbackIncludesRechargeWhenBalanceIncreases(t *testing.T) {
	day := time.Date(2026, 8, 18, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: -20, balanceDeltaOK: true}
	remote := &supplierRemoteClientStub{costErr: errors.New("成本接口不可用"), recharge: 50}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.SyncCost(context.Background(), 42, day, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []float64{30}, dataRepo.costValues)
	require.Equal(t, 1, remote.rechargeCalls)
	require.Equal(t, day, remote.rechargeDays[0])
}

func TestSupplierProviderSyncServiceCostFallbackPrefersLocalRechargeRecords(t *testing.T) {
	day := time.Date(2026, 8, 18, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "supplier-a", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: -20, balanceDeltaOK: true}
	rechargeRepo := &supplierProviderRechargeRepoStub{listResult: SupplierProviderRechargeListResult{
		Total:       1,
		TotalAmount: 50,
	}}
	remote := &supplierRemoteClientStub{costErr: errors.New("cost unavailable"), rechargeErr: errors.New("recharge unavailable")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true}, rechargeRepo)

	result, err := svc.SyncCost(context.Background(), 42, day, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []float64{30}, dataRepo.costValues)
	require.Zero(t, remote.rechargeCalls)
	require.Len(t, rechargeRepo.listCalls, 1)
	require.Equal(t, int64(42), rechargeRepo.listCalls[0].ProviderID)
	require.Equal(t, day, rechargeRepo.listCalls[0].Start)
	require.Equal(t, day.AddDate(0, 0, 1), rechargeRepo.listCalls[0].End)
}

func TestSupplierProviderSyncServiceCostFallbackFailsWhenRechargeQueryFails(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: 20, balanceDeltaOK: true}
	remote := &supplierRemoteClientStub{
		costErr:     errors.New("成本接口不可用"),
		rechargeErr: errors.New("充值接口不可用"),
	}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Equal(t, 1, remote.rechargeCalls)
	require.Zero(t, dataRepo.costCalls)
}

func TestSupplierProviderSyncServiceCostFallbackKeepsFailedWhenBalanceBaselineUnavailable(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{}
	remote := &supplierRemoteClientStub{costErr: errors.New("成本接口不可用")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Equal(t, 1, dataRepo.balanceDeltaCalls)
	require.Zero(t, dataRepo.costCalls)
}

func TestSupplierProviderSyncServiceCostFallbackKeepsFailedWhenBalanceIncreasedWithoutRecharge(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: -20, balanceDeltaOK: true}
	remote := &supplierRemoteClientStub{costErr: errors.New("成本接口不可用")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Equal(t, 1, dataRepo.balanceDeltaCalls)
	require.Zero(t, dataRepo.costCalls)
}

func TestSupplierProviderSyncServiceCostFallbackSkipsAuthFailure(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: 20, balanceDeltaOK: true}
	remote := &supplierRemoteClientStub{costErr: &SupplierProviderAuthFailureError{Err: errors.New("登录失败")}}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.Error(t, err)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Zero(t, dataRepo.balanceDeltaCalls)
	require.Zero(t, dataRepo.costCalls)
	require.False(t, providerRepo.items[0].Enabled)
	require.Equal(t, 1, providerRepo.disableAfterAuthFailureCalls)
}

func TestSupplierProviderSyncServiceCostFallbackSkipsSessionFailure(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: 20, balanceDeltaOK: true}
	sessionErr := &SupplierProviderSessionFailureError{Err: errors.New("session unavailable")}
	remote := &supplierRemoteClientStub{costErr: sessionErr}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.ErrorIs(t, err, sessionErr)
	require.Equal(t, SupplierSyncStatusFailed, result.Status)
	require.Zero(t, dataRepo.balanceDeltaCalls)
	require.Zero(t, dataRepo.costCalls)
	require.True(t, providerRepo.items[0].Enabled)
	require.Zero(t, providerRepo.disableAfterAuthFailureCalls)
}

func TestSupplierProviderSyncServiceCostOverridesUpstreamWithCalculatedWhenDeviationExceedsThreshold(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	// 上游返回 45.6，计算成本 5（余额差 5 + 无充值），偏差 89% > 50%。
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: 5, balanceDeltaOK: true}
	remote := &supplierRemoteClientStub{}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	// 写入生效值改为计算成本，并记录上游原始值与覆盖提示。
	require.Equal(t, []float64{5}, dataRepo.costValues)
	require.Len(t, dataRepo.detailedRawCosts, 1)
	require.NotNil(t, dataRepo.detailedRawCosts[0])
	require.Equal(t, 45.6, *dataRepo.detailedRawCosts[0])
	require.Len(t, dataRepo.detailedWarnings, 1)
	require.NotNil(t, dataRepo.detailedWarnings[0])
	require.Contains(t, *dataRepo.detailedWarnings[0], "生效成本已取计算成本")
}

func TestSupplierProviderSyncServiceCostOverridesUpstreamWithRechargeAdjustedBalanceWhenDeviationExceedsThreshold(t *testing.T) {
	day := time.Date(2026, 8, 18, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	// 计算成本 = 余额差 10 + 充值 5 = 15，与上游 45.6 偏差 67% > 50%；
	// 本地成本 5 同时在场，用于确认覆盖值来自计算成本而不是本地成本。
	dataRepo := &supplierProviderDataRepoStub{
		localCostValue:    5,
		localCostOK:       true,
		balanceDeltaValue: 10,
		balanceDeltaOK:    true,
	}
	remote := &supplierRemoteClientStub{recharge: 5}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.SyncCost(context.Background(), 42, day, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []float64{15}, dataRepo.costValues)
	require.Equal(t, 1, remote.rechargeCalls)
	require.Equal(t, day, remote.rechargeDays[0])
	require.Len(t, dataRepo.detailedRawCosts, 1)
	require.NotNil(t, dataRepo.detailedRawCosts[0])
	require.Equal(t, 45.6, *dataRepo.detailedRawCosts[0])
	require.Len(t, dataRepo.detailedWarnings, 1)
	require.NotNil(t, dataRepo.detailedWarnings[0])
	require.Contains(t, *dataRepo.detailedWarnings[0], "余额差 10.00 + 充值 5.00")
}

func TestSupplierProviderSyncServiceCostReviewCarriesUpstreamCalculatedAndLocalCosts(t *testing.T) {
	day := time.Date(2026, 8, 18, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	// 核对页的三个数必须互相独立：上游 45.6、计算 40（余额差 30 + 充值 10）、本地 21.68。
	dataRepo := &supplierProviderDataRepoStub{
		balanceDeltaValue: 30,
		balanceDeltaOK:    true,
		localCostValue:    21.68,
		localCostOK:       true,
	}
	remote := &supplierRemoteClientStub{recharge: 10}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetCostReviewService(NewSupplierProviderCostReviewService(nil))

	result, err := svc.SyncCost(context.Background(), 42, day, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Len(t, dataRepo.reviewInputs, 1)
	review := dataRepo.reviewInputs[0]
	require.NotNil(t, review.UpstreamCost)
	require.Equal(t, 45.6, *review.UpstreamCost)
	require.NotNil(t, review.CalculatedCost)
	require.Equal(t, 40.0, *review.CalculatedCost)
	require.NotNil(t, review.LocalCost)
	require.Equal(t, 21.68, *review.LocalCost)
	// 偏差 12% 未超阈值，生效成本仍是上游值
	require.Equal(t, 45.6, review.EffectiveCost)
}

func TestSupplierProviderSyncServiceCostKeepsUpstreamWithinThreshold(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	// 上游 45.6，计算成本 30，偏差 34% <= 50%。
	dataRepo := &supplierProviderDataRepoStub{balanceDeltaValue: 30, balanceDeltaOK: true}
	remote := &supplierRemoteClientStub{}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []float64{45.6}, dataRepo.costValues)
	require.NotNil(t, dataRepo.detailedRawCosts[0])
	require.Equal(t, 45.6, *dataRepo.detailedRawCosts[0])
	require.Nil(t, dataRepo.detailedWarnings[0])
}

func TestSupplierProviderSyncServiceCostKeepsUpstreamWhenOnlyLocalCostDeviates(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	// 本地成本 5 与上游 45.6 偏差 89%，但本地成本只是参考值：
	// 没有计算成本（缺余额快照）时不得改写生效成本，否则会把用户计费口径当成供应商成本。
	dataRepo := &supplierProviderDataRepoStub{localCostValue: 5, localCostOK: true}
	remote := &supplierRemoteClientStub{}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetCostDeviationThresholdProvider(supplierCostDeviationThresholdStub{threshold: 0.5})

	result, err := svc.SyncCost(context.Background(), 42, time.Now(), SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Equal(t, []float64{45.6}, dataRepo.costValues)
	require.Equal(t, 45.6, *dataRepo.detailedRawCosts[0])
	require.Nil(t, dataRepo.detailedWarnings[0])
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
