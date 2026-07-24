package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type supplierAccountRateGuardSyncerStub struct {
	results map[int64]SupplierProviderRateSyncResult
	errs    map[int64]error
	calls   []int64
}

func (s *supplierAccountRateGuardSyncerStub) SyncAccountRates(_ context.Context, providerID int64, _ string) (SupplierProviderRateSyncResult, error) {
	s.calls = append(s.calls, providerID)
	return s.results[providerID], s.errs[providerID]
}

type supplierAccountRateGuardRepoStub struct {
	candidates map[int64][]SupplierAccountRateGuardCandidate
	logs       []SupplierAccountRateGuardUnbindLog
	listParams SupplierAccountRateGuardUnbindLogListParams
	listResult SupplierAccountRateGuardUnbindLogListResult
	listErr    error
	createErr  error
}

func (r *supplierAccountRateGuardRepoStub) ListAccountRateGuardCandidates(_ context.Context, providerID int64, _ []string) ([]SupplierAccountRateGuardCandidate, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.candidates[providerID], nil
}

func (r *supplierAccountRateGuardRepoStub) CreateAccountRateGuardUnbindLogs(_ context.Context, logs []SupplierAccountRateGuardUnbindLog) error {
	r.logs = append(r.logs, logs...)
	return r.createErr
}

func (r *supplierAccountRateGuardRepoStub) ListAccountRateGuardUnbindLogs(_ context.Context, params SupplierAccountRateGuardUnbindLogListParams) (SupplierAccountRateGuardUnbindLogListResult, error) {
	r.listParams = params
	return r.listResult, r.listErr
}
func (r *supplierAccountRateGuardRepoStub) MarkAccountRateGuardUnbindLogHandled(_ context.Context, id int64) (SupplierAccountRateGuardUnbindLog, error) {
	for index, logItem := range r.logs {
		if logItem.ID == id {
			r.logs[index].Status = SupplierAccountRateGuardLogStatusHandled
			return r.logs[index], nil
		}
	}
	return SupplierAccountRateGuardUnbindLog{}, errors.New("未找到解绑日志")
}

type accountRateGuardRemoverStub struct {
	calls   []accountRateGuardRemovalCall
	results map[int64]AccountRateGuardGroupRemovalResult
	err     error
}

type accountRateGuardRemovalCall struct {
	accountID int64
	groupIDs  []int64
}

func (r *accountRateGuardRemoverStub) RemoveAccountGroupsForRateGuard(_ context.Context, accountID int64, groupIDs []int64) (AccountRateGuardGroupRemovalResult, error) {
	r.calls = append(r.calls, accountRateGuardRemovalCall{accountID: accountID, groupIDs: append([]int64(nil), groupIDs...)})
	if r.err != nil {
		return AccountRateGuardGroupRemovalResult{}, r.err
	}
	return r.results[accountID], nil
}

func TestEffectiveSupplierAccountRateValidatesAndScales(t *testing.T) {
	rate, err := effectiveSupplierAccountRate(1.25, 0.8)
	require.NoError(t, err)
	require.InDelta(t, 1.0, rate, 1e-9)

	for _, input := range [][2]float64{{-1, 1}, {1, 0}} {
		_, err := effectiveSupplierAccountRate(input[0], input[1])
		require.Error(t, err)
	}
}

func TestSupplierAccountRateGuardPreviewOnlyPlansRiskGroups(t *testing.T) {
	providers := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Name: "供应商甲", Enabled: true}}}
	syncer := &supplierAccountRateGuardSyncerStub{results: map[int64]SupplierProviderRateSyncResult{
		1: {ProviderID: 1, Status: SupplierSyncStatusSuccess, UpdatedKeys: []string{"key-1"}},
	}, errs: map[int64]error{}}
	repo := &supplierAccountRateGuardRepoStub{candidates: map[int64][]SupplierAccountRateGuardCandidate{
		1: {{
			ProviderID: 1, ProviderName: "供应商甲", ProviderAccountID: 11,
			UpstreamAccountKey: "key-1", UpstreamAccountName: "上游账号", RawRate: 1, RateScale: 1.2,
			LocalAccountID: 21, LocalAccountName: "本地账号", MatchStatus: SupplierAccountRateGuardMatchMatched,
			Schedulable: true,
			Groups:      []SupplierAccountRateGuardGroup{{ID: 31, Name: "正常组", RateMultiplier: 1.2}, {ID: 32, Name: "风险组", RateMultiplier: 1.5}},
		}},
	}}
	remover := &accountRateGuardRemoverStub{}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, remover)

	result, err := guard.Run(context.Background(), 99, SupplierAccountRateGuardModePreview, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.CheckedAccounts)
	require.Equal(t, 1, result.RiskGroups)
	require.Equal(t, 0, result.UnboundGroups)
	require.Empty(t, remover.calls)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultPlanned, repo.logs[0].Result)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, repo.logs[0].Status)
	require.Equal(t, int64(32), repo.logs[0].LocalGroupID)
	require.InDelta(t, 1.2, repo.logs[0].EffectiveUpstreamRate, 1e-9)
}

func TestSupplierAccountRateGuardExecuteRemovesOnlyRiskGroupsAndKeepsScheduling(t *testing.T) {
	providers := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Name: "供应商甲", Enabled: true}}}
	syncer := &supplierAccountRateGuardSyncerStub{results: map[int64]SupplierProviderRateSyncResult{
		1: {ProviderID: 1, Status: SupplierSyncStatusSuccess, UpdatedKeys: []string{"key-1"}},
	}, errs: map[int64]error{}}
	repo := &supplierAccountRateGuardRepoStub{candidates: map[int64][]SupplierAccountRateGuardCandidate{
		1: {{
			ProviderID: 1, ProviderName: "供应商甲", ProviderAccountID: 11,
			UpstreamAccountKey: "key-1", UpstreamAccountName: "上游账号", RawRate: 1, RateScale: 1,
			LocalAccountID: 21, LocalAccountName: "本地账号", MatchStatus: SupplierAccountRateGuardMatchMatched,
			Schedulable: true,
			Groups:      []SupplierAccountRateGuardGroup{{ID: 31, Name: "正常组", RateMultiplier: 1}, {ID: 32, Name: "风险组", RateMultiplier: 1.1}},
		}},
	}}
	remover := &accountRateGuardRemoverStub{results: map[int64]AccountRateGuardGroupRemovalResult{
		21: {RemovedGroupIDs: []int64{32}, RemainingGroupIDs: []int64{31}, SchedulableBefore: true, SchedulableAfter: true},
	}}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, remover)

	result, err := guard.Run(context.Background(), 100, SupplierAccountRateGuardModeExecute, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.UnboundGroups)
	require.Equal(t, 0, result.DisabledAccounts)
	require.Equal(t, []accountRateGuardRemovalCall{{accountID: 21, groupIDs: []int64{32}}}, remover.calls)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultUnbound, repo.logs[0].Result)
	require.Equal(t, SupplierAccountRateGuardLogStatusPending, repo.logs[0].Status)
	require.False(t, repo.logs[0].AfterBound)
	require.True(t, *repo.logs[0].AfterSchedulable)
}

func TestSupplierAccountRateGuardRemovalFailureKeepsObservedSchedulingState(t *testing.T) {
	providers := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Name: "供应商甲", Enabled: true}}}
	syncer := &supplierAccountRateGuardSyncerStub{results: map[int64]SupplierProviderRateSyncResult{
		1: {ProviderID: 1, Status: SupplierSyncStatusSuccess, UpdatedKeys: []string{"key-1"}},
	}, errs: map[int64]error{}}
	repo := &supplierAccountRateGuardRepoStub{candidates: map[int64][]SupplierAccountRateGuardCandidate{
		1: {{
			ProviderID: 1, ProviderName: "供应商甲", ProviderAccountID: 11,
			UpstreamAccountKey: "key-1", UpstreamAccountName: "上游账号", RawRate: 1, RateScale: 1,
			LocalAccountID: 21, LocalAccountName: "本地账号", MatchStatus: SupplierAccountRateGuardMatchMatched,
			Schedulable: true,
			Groups:      []SupplierAccountRateGuardGroup{{ID: 32, Name: "风险组", RateMultiplier: 1.1}},
		}},
	}}
	remover := &accountRateGuardRemoverStub{err: errors.New("数据库写入失败")}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, remover)

	result, err := guard.Run(context.Background(), 102, SupplierAccountRateGuardModeExecute, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.Failed)
	require.Zero(t, result.DisabledAccounts)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultFailed, repo.logs[0].Result)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, repo.logs[0].Status)
	require.True(t, *repo.logs[0].BeforeSchedulable)
	require.True(t, *repo.logs[0].AfterSchedulable)
}

func TestSupplierAccountRateGuardSkipsConflictsAndContinuesAfterProviderFailure(t *testing.T) {
	providers := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Name: "失败供应商", Enabled: true}, {ID: 2, Name: "正常供应商", Enabled: true}}}
	syncer := &supplierAccountRateGuardSyncerStub{
		results: map[int64]SupplierProviderRateSyncResult{2: {ProviderID: 2, Status: SupplierSyncStatusSuccess, UpdatedKeys: []string{"key-2"}}},
		errs:    map[int64]error{1: errors.New("接口超时")},
	}
	repo := &supplierAccountRateGuardRepoStub{candidates: map[int64][]SupplierAccountRateGuardCandidate{
		2: {{ProviderID: 2, ProviderAccountID: 12, UpstreamAccountKey: "key-2", RawRate: 1, RateScale: 1, LocalAccountID: 22, MatchStatus: SupplierAccountRateGuardMatchConflict, Groups: []SupplierAccountRateGuardGroup{{ID: 33, RateMultiplier: 2}}}},
	}}
	remover := &accountRateGuardRemoverStub{}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, remover)

	result, err := guard.Run(context.Background(), 101, SupplierAccountRateGuardModeExecute, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.RateSyncFailedProviders)
	require.Equal(t, 1, result.Skipped)
	require.Empty(t, remover.calls)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultSkipped, repo.logs[0].Result)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, repo.logs[0].Status)
}
