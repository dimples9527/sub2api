package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// supplierProviderMonitorAutoMatchDataStub is a dedicated stub for testing auto-match logic.
type supplierProviderMonitorAutoMatchDataStub struct {
	accounts      []SupplierProviderAccount
	monitorTargets []SupplierProviderMonitorTarget
	appliedBindings []appliedBinding
}

type appliedBinding struct {
	monitorTargetID int64
	localAccountID  int64
}

func (s *supplierProviderMonitorAutoMatchDataStub) ListAccounts(_ context.Context, _ SupplierProviderDataListParams) (SupplierProviderAccountListResult, error) {
	return SupplierProviderAccountListResult{Items: s.accounts, Total: int64(len(s.accounts))}, nil
}

func (s *supplierProviderMonitorAutoMatchDataStub) ListMonitorTargets(_ context.Context, _ SupplierProviderMonitorTargetListParams) (SupplierProviderMonitorTargetListResult, error) {
	return SupplierProviderMonitorTargetListResult{Items: s.monitorTargets, Total: int64(len(s.monitorTargets))}, nil
}

func (s *supplierProviderMonitorAutoMatchDataStub) ApplyMonitorAutoMatch(_ context.Context, monitorTargetID, localAccountID int64) error {
	s.appliedBindings = append(s.appliedBindings, appliedBinding{monitorTargetID: monitorTargetID, localAccountID: localAccountID})
	return nil
}

// Unused stubs: implement SupplierProviderDataRepository interface
func (s *supplierProviderMonitorAutoMatchDataStub) ListGroups(context.Context, SupplierProviderDataListParams) (SupplierProviderGroupListResult, error) {
	return SupplierProviderGroupListResult{}, nil
}
func (s *supplierProviderMonitorAutoMatchDataStub) ListBindableLocalAccounts(context.Context, SupplierBindableLocalAccountListParams) (SupplierBindableLocalAccountListResult, error) {
	return SupplierBindableLocalAccountListResult{}, nil
}
func (s *supplierProviderMonitorAutoMatchDataStub) BindMonitorTarget(context.Context, int64, int64) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UnbindMonitorTarget(context.Context, int64) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ListGroupsForAutoMatch(context.Context, int64) ([]SupplierProviderGroup, error) { return nil, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) GetGroupForAutoMatch(context.Context, int64) (SupplierProviderGroup, error) { return SupplierProviderGroup{}, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateGroupMapping(context.Context, int64, *int64) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) DeleteGroup(context.Context, int64) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) DeleteAccount(context.Context, int64) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ApplyAutoMatch(context.Context, int64, int64, string) (bool, error) { return false, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateAutoMatchState(context.Context, int64, string, bool) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateAutoMatchIgnored(context.Context, int64, bool) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) AcknowledgeNameChange(context.Context, int64, string) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ListMappingsByLocalGroup(context.Context, []int64) ([]SupplierProviderGroup, error) { return nil, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) GetGroupForRateGuard(context.Context, int64) (SupplierProviderGroup, error) { return SupplierProviderGroup{}, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) SelectRateGuard(context.Context, int64, string) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ClearRateGuard(context.Context, int64, string) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) SetRateGuardEnabled(context.Context, int64, bool) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ListRateGuardCandidates(context.Context) ([]SupplierRateGuardCandidate, error) { return nil, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ApplyRateGuard(context.Context, SupplierRateGuardApplyInput) (SupplierRateGuardApplyResult, error) { return SupplierRateGuardApplyResult{}, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) MarkRateGuardChecked(context.Context, int64, time.Time) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ReplaceAccounts(context.Context, int64, []SupplierProviderRemoteAccount, time.Time) (SupplierSyncCounts, error) { return SupplierSyncCounts{}, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ReplaceGroups(context.Context, int64, []SupplierProviderRemoteGroup, time.Time) (SupplierProviderGroupReplaceResult, error) { return SupplierProviderGroupReplaceResult{}, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ListGroupHealthTrends(context.Context, SupplierProviderGroupHealthTrendParams) ([]SupplierProviderGroupHealthTrend, error) { return nil, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ListLocalGroupHealthTrends(context.Context, SupplierProviderGroupHealthTrendParams) ([]SupplierProviderGroupHealthTrend, error) { return nil, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) IsUniqueMatchedLocalAccount(context.Context, int64) (bool, error) { return false, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) GetLocalAccountEffectivePlatform(context.Context, int64) (string, error) { return "", nil }
func (s *supplierProviderMonitorAutoMatchDataStub) GetLocalAccountPlatformOverride(context.Context, int64) (string, error) { return "", nil }
func (s *supplierProviderMonitorAutoMatchDataStub) SetLocalAccountPlatformOverride(context.Context, int64, string) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) ClearLocalAccountPlatformOverride(context.Context, int64) error { return nil }

func (s *supplierProviderMonitorAutoMatchDataStub) UpdateBalance(context.Context, int64, float64, time.Time) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateCost(context.Context, int64, float64, time.Time) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateCostDetailed(context.Context, int64, float64, *float64, *string, time.Time) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateCostDetailedWithReview(context.Context, int64, *float64, *string, time.Time, SupplierProviderCostReviewSyncInput) (float64, error) { return 0, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) GetLocalCostForDay(context.Context, int64, time.Time) (float64, bool, error) { return 0, false, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) GetBalanceDeltaForDay(context.Context, int64, time.Time) (float64, bool, error) { return 0, false, nil }
func (s *supplierProviderMonitorAutoMatchDataStub) CreateSyncRun(context.Context, *SupplierProviderSyncRun) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) FinishSyncRun(context.Context, *SupplierProviderSyncRun) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateSyncStatus(context.Context, int64, string, string, time.Time) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) UpdateGroupSyncStatus(context.Context, int64, string, string, time.Time) error { return nil }
func (s *supplierProviderMonitorAutoMatchDataStub) Cleanup(context.Context, SupplierCleanupPolicy, time.Time, int) (SupplierCleanupCounts, error) { return SupplierCleanupCounts{}, nil }

func TestAppendUniqueSupplierMonitorAccountMatch_Deduplicates(t *testing.T) {

	matches := []supplierMonitorAccountMatch{
		{localAccountID: 1, localAccountName: "a"},
		{localAccountID: 2, localAccountName: "b"},
	}
	result := appendUniqueSupplierMonitorAccountMatch(matches, supplierMonitorAccountMatch{localAccountID: 1, localAccountName: "c"})
	require.Len(t, result, 2)
	require.Equal(t, "a", result[0].localAccountName)
	require.Equal(t, "b", result[1].localAccountName)
}

func TestAppendUniqueSupplierMonitorAccountMatch_AddsNew(t *testing.T) {
	matches := []supplierMonitorAccountMatch{
		{localAccountID: 1, localAccountName: "a"},
	}
	result := appendUniqueSupplierMonitorAccountMatch(matches, supplierMonitorAccountMatch{localAccountID: 3, localAccountName: "c"})
	require.Len(t, result, 2)
	require.Equal(t, "c", result[1].localAccountName)
}

func TestMonitorAccountCandidates_BuildsIndex(t *testing.T) {
	dataStub := &supplierProviderMonitorAutoMatchDataStub{
		accounts: []SupplierProviderAccount{
			{ID: 1, Name: "p-gpt4", LocalAccountID: ptrInt64(101), LocalAccountName: "p-gpt4"},
			{ID: 2, Name: "p-gpt4", LocalAccountID: ptrInt64(102), LocalAccountName: "p-gpt4"},
		},
	}
	svc := &SupplierProviderSyncService{dataRepo: dataStub}
	provider := &SupplierProvider{ID: 100, Name: "p", AccountNamePrefix: "p-"}

	candidates, err := svc.monitorAccountCandidates(context.Background(), provider)
	require.NoError(t, err)
	// Two accounts with same normalized name "gpt4" but different IDs
	require.Contains(t, candidates, "gpt4")
	require.Len(t, candidates["gpt4"], 2)
}

func TestMonitorAccountCandidates_SkipsUnboundAccount(t *testing.T) {
	dataStub := &supplierProviderMonitorAutoMatchDataStub{
		accounts: []SupplierProviderAccount{
			{ID: 1, Name: "p-gpt4", LocalAccountID: nil},
		},
	}
	svc := &SupplierProviderSyncService{dataRepo: dataStub}
	provider := &SupplierProvider{ID: 100, Name: "p", AccountNamePrefix: "p-"}

	candidates, err := svc.monitorAccountCandidates(context.Background(), provider)
	require.NoError(t, err)
	require.Empty(t, candidates)
}

func TestAutoMatchMonitorTargetsForProvider_UniqueMatch(t *testing.T) {
	dataStub := &supplierProviderMonitorAutoMatchDataStub{
		accounts: []SupplierProviderAccount{
			{ID: 1, Name: "p-gpt4", LocalAccountID: ptrInt64(101), LocalAccountName: "p-gpt4"},
		},
		monitorTargets: []SupplierProviderMonitorTarget{
			{ID: 10, MonitorName: "gpt4", MonitorKey: "k1"},
			{ID: 11, MonitorName: "claude", MonitorKey: "k2"},
		},
	}
	svc := &SupplierProviderSyncService{dataRepo: dataStub}
	provider := &SupplierProvider{ID: 100, Name: "p", AccountNamePrefix: "p-"}

	result, err := svc.autoMatchMonitorTargetsForProvider(context.Background(), provider)
	require.NoError(t, err)
	require.Equal(t, 1, result.Matched)   // gpt4 matches
	require.Equal(t, 0, result.Ambiguous)
	require.Equal(t, 1, result.Skipped)   // claude has no match
	require.Equal(t, 2, result.Total)
	require.Len(t, dataStub.appliedBindings, 1)
	require.Equal(t, int64(10), dataStub.appliedBindings[0].monitorTargetID)
	require.Equal(t, int64(101), dataStub.appliedBindings[0].localAccountID)
}

func TestAutoMatchMonitorTargetsForProvider_AmbiguousSkip(t *testing.T) {
	dataStub := &supplierProviderMonitorAutoMatchDataStub{
		accounts: []SupplierProviderAccount{
			{ID: 1, Name: "p-gpt4", LocalAccountID: ptrInt64(101), LocalAccountName: "p-gpt4-a"},
			{ID: 2, Name: "p-gpt4", LocalAccountID: ptrInt64(102), LocalAccountName: "p-gpt4-b"},
		},
		monitorTargets: []SupplierProviderMonitorTarget{
			{ID: 10, MonitorName: "gpt4", MonitorKey: "k1"},
		},
	}
	svc := &SupplierProviderSyncService{dataRepo: dataStub}
	provider := &SupplierProvider{ID: 100, Name: "p", AccountNamePrefix: "p-"}

	result, err := svc.autoMatchMonitorTargetsForProvider(context.Background(), provider)
	require.NoError(t, err)
	require.Equal(t, 0, result.Matched)
	require.Equal(t, 1, result.Ambiguous)
	require.Equal(t, 0, result.Skipped)
	require.Equal(t, 1, result.Total)
	require.Empty(t, dataStub.appliedBindings)
}

func TestAutoMatchMonitorTargetsForProvider_SkipsExistingBinding(t *testing.T) {
	existing := int64(201)
	dataStub := &supplierProviderMonitorAutoMatchDataStub{
		accounts: []SupplierProviderAccount{
			{ID: 1, Name: "p-gpt4", LocalAccountID: ptrInt64(101), LocalAccountName: "p-gpt4"},
		},
		monitorTargets: []SupplierProviderMonitorTarget{
			{ID: 10, MonitorName: "gpt4", MonitorKey: "k1", LocalAccountID: existing},
		},
	}
	svc := &SupplierProviderSyncService{dataRepo: dataStub}
	provider := &SupplierProvider{ID: 100, Name: "p", AccountNamePrefix: "p-"}

	result, err := svc.autoMatchMonitorTargetsForProvider(context.Background(), provider)
	require.NoError(t, err)
	require.Equal(t, 0, result.Matched)
	require.Equal(t, 0, result.Ambiguous)
	require.Equal(t, 0, result.Skipped)
	require.Equal(t, 0, result.Total) // skipped because already has local account
	require.Empty(t, dataStub.appliedBindings)
}
