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
	// 批量标记的入参要被看见：筛选口径漏传会让"一键处理"打到全表。
	batchParams SupplierAccountRateGuardUnbindLogListParams
	batchResult SupplierAccountRateGuardUnbindLogBatchHandledResult
	batchErr    error
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

func (r *supplierAccountRateGuardRepoStub) MarkAccountRateGuardUnbindLogsHandled(_ context.Context, params SupplierAccountRateGuardUnbindLogListParams) (SupplierAccountRateGuardUnbindLogBatchHandledResult, error) {
	r.batchParams = params
	return r.batchResult, r.batchErr
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
			UpstreamAccountKey: "key-1", UpstreamAccountName: "上游账号", RawRate: 0.12, RateScale: 1,
			LocalAccountID: 21, LocalAccountName: "本地账号", MatchStatus: SupplierAccountRateGuardMatchMatched,
			Schedulable: true,
			Groups: []SupplierAccountRateGuardGroup{
				{ID: 31, Name: "低倍率风险组", RateMultiplier: 0.11},
				{ID: 32, Name: "同倍率正常组", RateMultiplier: 0.12},
				{ID: 33, Name: "高倍率正常组", RateMultiplier: 0.17},
				{ID: 34, Name: "更高倍率正常组", RateMultiplier: 0.28},
			},
		}},
	}}
	remover := &accountRateGuardRemoverStub{}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, remover)

	result, err := guard.Run(context.Background(), 99, SupplierAccountRateGuardModePreview, nil, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.CheckedAccounts)
	require.Equal(t, 1, result.RiskGroups)
	require.Equal(t, 0, result.UnboundGroups)
	require.Empty(t, remover.calls)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultPlanned, repo.logs[0].Result)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, repo.logs[0].Status)
	require.Equal(t, int64(31), repo.logs[0].LocalGroupID)
	require.InDelta(t, 0.12, repo.logs[0].EffectiveUpstreamRate, 1e-9)
}

func TestSupplierAccountRateGuardUsesSameToleranceAsUpstreamAccountGuard(t *testing.T) {
	providers := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Name: "供应商甲", Enabled: true}}}
	syncer := &supplierAccountRateGuardSyncerStub{results: map[int64]SupplierProviderRateSyncResult{
		1: {ProviderID: 1, Status: SupplierSyncStatusSuccess, UpdatedKeys: []string{"key-1"}},
	}, errs: map[int64]error{}}
	repo := &supplierAccountRateGuardRepoStub{candidates: map[int64][]SupplierAccountRateGuardCandidate{
		1: {{
			ProviderID: 1, ProviderAccountID: 11, UpstreamAccountKey: "key-1", RawRate: 1, RateScale: 1,
			LocalAccountID: 21, MatchStatus: SupplierAccountRateGuardMatchMatched, Schedulable: true,
			Groups: []SupplierAccountRateGuardGroup{
				{ID: 31, Name: "容差内正常组", RateMultiplier: 1 - 0.00000005},
				{ID: 32, Name: "超过容差风险组", RateMultiplier: 1 - 0.0000002},
			},
		}},
	}}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, &accountRateGuardRemoverStub{})

	result, err := guard.Run(context.Background(), 103, SupplierAccountRateGuardModePreview, nil, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.RiskGroups)
	require.Len(t, repo.logs, 1)
	require.Equal(t, int64(32), repo.logs[0].LocalGroupID)
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
			Groups:      []SupplierAccountRateGuardGroup{{ID: 31, Name: "正常组", RateMultiplier: 1}, {ID: 32, Name: "风险组", RateMultiplier: 0.9}},
		}},
	}}
	remover := &accountRateGuardRemoverStub{results: map[int64]AccountRateGuardGroupRemovalResult{
		21: {RemovedGroupIDs: []int64{32}, RemainingGroupIDs: []int64{31}, SchedulableBefore: true, SchedulableAfter: true},
	}}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, remover)

	result, err := guard.Run(context.Background(), 100, SupplierAccountRateGuardModeExecute, nil, time.Now())

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
			Groups:      []SupplierAccountRateGuardGroup{{ID: 32, Name: "风险组", RateMultiplier: 0.9}},
		}},
	}}
	remover := &accountRateGuardRemoverStub{err: errors.New("数据库写入失败")}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, remover)

	result, err := guard.Run(context.Background(), 102, SupplierAccountRateGuardModeExecute, nil, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.Failed)
	require.Zero(t, result.DisabledAccounts)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultFailed, repo.logs[0].Result)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, repo.logs[0].Status)
	require.True(t, *repo.logs[0].BeforeSchedulable)
	require.True(t, *repo.logs[0].AfterSchedulable)
}

func TestSupplierAccountRateGuardContinuesAfterProviderFailure(t *testing.T) {
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

	result, err := guard.Run(context.Background(), 101, SupplierAccountRateGuardModeExecute, nil, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.RateSyncFailedProviders)
	require.Equal(t, 1, result.Skipped)
	require.Empty(t, remover.calls)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultSkipped, repo.logs[0].Result)
	require.Equal(t, SupplierAccountRateGuardLogStatusHandled, repo.logs[0].Status)
}

func TestSupplierAccountRateGuardDoesNotCountSyncConflictAsFailure(t *testing.T) {
	providers := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Name: "正在同步的供应商", Enabled: true}}}
	syncer := &supplierAccountRateGuardSyncerStub{
		errs: map[int64]error{1: ErrSupplierProviderSyncConflict},
	}
	guard := NewSupplierAccountRateGuardService(providers, syncer, &supplierAccountRateGuardRepoStub{}, &accountRateGuardRemoverStub{})

	result, err := guard.Run(context.Background(), 102, SupplierAccountRateGuardModeExecute, nil, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.CheckedProviders)
	require.Zero(t, result.RateSyncFailedProviders)
}

// newSupplierAccountRateGuardGroupSwitchFixture 构造一个"两个风险分组 + 一个正常分组"的候选，
// 供分组开关相关的用例共用。
func newSupplierAccountRateGuardGroupSwitchFixture() (*supplierProviderRepoStub, *supplierAccountRateGuardRepoStub, *accountRateGuardRemoverStub, *SupplierAccountRateGuardService) {
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
			Groups: []SupplierAccountRateGuardGroup{
				{ID: 31, Name: "已关闭风险组", RateMultiplier: 0.9},
				{ID: 32, Name: "未关闭风险组", RateMultiplier: 0.9},
				{ID: 33, Name: "正常组", RateMultiplier: 1},
			},
		}},
	}}
	remover := &accountRateGuardRemoverStub{results: map[int64]AccountRateGuardGroupRemovalResult{
		21: {RemovedGroupIDs: []int64{31, 32}, RemainingGroupIDs: []int64{33}, SchedulableBefore: true, SchedulableAfter: true},
	}}
	return providers, repo, remover, NewSupplierAccountRateGuardService(providers, syncer, repo, remover)
}

func TestSupplierAccountRateGuardSkipsDisabledGroupsInPreview(t *testing.T) {
	_, repo, remover, guard := newSupplierAccountRateGuardGroupSwitchFixture()

	result, err := guard.Run(context.Background(), 104, SupplierAccountRateGuardModePreview, []int64{31}, time.Now())

	require.NoError(t, err)
	// 31 被关闭后只剩 32 是风险分组，且关闭的分组不应出现在任何计划里。
	require.Equal(t, 1, result.RiskGroups)
	require.Zero(t, result.Skipped)
	require.Empty(t, remover.calls)
	require.Len(t, repo.logs, 1)
	require.Equal(t, int64(32), repo.logs[0].LocalGroupID)
	require.Equal(t, SupplierAccountRateGuardLogResultPlanned, repo.logs[0].Result)
}

func TestSupplierAccountRateGuardSkipsDisabledGroupsInExecute(t *testing.T) {
	_, repo, remover, guard := newSupplierAccountRateGuardGroupSwitchFixture()

	result, err := guard.Run(context.Background(), 105, SupplierAccountRateGuardModeExecute, []int64{31}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.RiskGroups)
	require.Equal(t, 1, result.UnboundGroups)
	// 关键不变量：解除绑定只发给未关闭的分组，关闭的分组一个都不能进 remover。
	require.Len(t, remover.calls, 1)
	require.Equal(t, []int64{32}, remover.calls[0].groupIDs)
	require.Len(t, repo.logs, 1)
	require.Equal(t, int64(32), repo.logs[0].LocalGroupID)
}

func TestSupplierAccountRateGuardLogsSkippedWhenAllRiskGroupsDisabled(t *testing.T) {
	_, repo, remover, guard := newSupplierAccountRateGuardGroupSwitchFixture()

	result, err := guard.Run(context.Background(), 106, SupplierAccountRateGuardModeExecute, []int64{31, 32}, time.Now())

	require.NoError(t, err)
	require.Zero(t, result.RiskGroups)
	require.Zero(t, result.UnboundGroups)
	require.Equal(t, 1, result.Skipped)
	// 全部分组被关闭时不能调用 remover，否则会出现"开关关掉了却仍然解绑"的事故。
	require.Empty(t, remover.calls)
	require.Len(t, repo.logs, 1)
	require.Equal(t, SupplierAccountRateGuardLogResultSkipped, repo.logs[0].Result)
	require.Contains(t, repo.logs[0].ErrorMessage, "已关闭倍率守护")
}

func TestSupplierAccountRateGuardEmptyDisabledGroupsKeepsCurrentBehavior(t *testing.T) {
	_, repo, remover, guard := newSupplierAccountRateGuardGroupSwitchFixture()

	result, err := guard.Run(context.Background(), 107, SupplierAccountRateGuardModeExecute, nil, time.Now())

	require.NoError(t, err)
	// nil 与空列表都必须等价于"全部分组开启"，这是"默认为开启"的兜底。
	require.Equal(t, 2, result.RiskGroups)
	require.Equal(t, 2, result.UnboundGroups)
	require.Len(t, remover.calls, 1)
	require.ElementsMatch(t, []int64{31, 32}, remover.calls[0].groupIDs)
	require.Len(t, repo.logs, 2)
}

func TestSupplierAccountRateGuardIgnoresNonPositiveDisabledGroupIDs(t *testing.T) {
	_, repo, remover, guard := newSupplierAccountRateGuardGroupSwitchFixture()

	result, err := guard.Run(context.Background(), 108, SupplierAccountRateGuardModeExecute, []int64{0, -1}, time.Now())

	require.NoError(t, err)
	// 非法 ID 不应误伤任何分组：0 与负数都不可能是真实分组 ID。
	require.Equal(t, 2, result.RiskGroups)
	require.Len(t, remover.calls, 1)
	require.ElementsMatch(t, []int64{31, 32}, remover.calls[0].groupIDs)
	require.Len(t, repo.logs, 2)
}

func TestSupplierAccountRateGuardNoRiskGroupsStillProducesNoLog(t *testing.T) {
	providers := &supplierProviderRepoStub{items: []*SupplierProvider{{ID: 1, Name: "供应商甲", Enabled: true}}}
	syncer := &supplierAccountRateGuardSyncerStub{results: map[int64]SupplierProviderRateSyncResult{
		1: {ProviderID: 1, Status: SupplierSyncStatusSuccess, UpdatedKeys: []string{"key-1"}},
	}, errs: map[int64]error{}}
	repo := &supplierAccountRateGuardRepoStub{candidates: map[int64][]SupplierAccountRateGuardCandidate{
		1: {{
			ProviderID: 1, ProviderAccountID: 11, UpstreamAccountKey: "key-1", RawRate: 1, RateScale: 1,
			LocalAccountID: 21, MatchStatus: SupplierAccountRateGuardMatchMatched, Schedulable: true,
			Groups: []SupplierAccountRateGuardGroup{{ID: 33, Name: "正常组", RateMultiplier: 1}},
		}},
	}}
	guard := NewSupplierAccountRateGuardService(providers, syncer, repo, &accountRateGuardRemoverStub{})

	result, err := guard.Run(context.Background(), 109, SupplierAccountRateGuardModeExecute, nil, time.Now())

	require.NoError(t, err)
	// 本来就没有风险分组时不该产生 skipped 记录，避免日志被"无风险"刷屏。
	require.Zero(t, result.Skipped)
	require.Empty(t, repo.logs)
}
