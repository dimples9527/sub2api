package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type supplierRateGuardRepoFake struct {
	candidates []SupplierRateGuardCandidate
	applyErr   map[int64]error
	applied    []SupplierRateGuardApplyInput
	checked    []int64
}

func (f *supplierRateGuardRepoFake) ListRateGuardCandidates(context.Context) ([]SupplierRateGuardCandidate, error) {
	return f.candidates, nil
}

func (f *supplierRateGuardRepoFake) ApplyRateGuard(_ context.Context, input SupplierRateGuardApplyInput) (SupplierRateGuardApplyResult, error) {
	f.applied = append(f.applied, input)
	if err := f.applyErr[input.MappingID]; err != nil {
		return SupplierRateGuardApplyResult{}, err
	}
	for _, candidate := range f.candidates {
		if candidate.MappingID != input.MappingID {
			continue
		}
		result := SupplierRateGuardApplyResult{OldRate: candidate.LocalRateMultiplier, TargetRate: input.TargetRate}
		if candidate.LocalRateMultiplier < input.TargetRate {
			result.Action = SupplierRateGuardActionRaised
		} else {
			result.Action = SupplierRateGuardActionUnchanged
		}
		return result, nil
	}
	return SupplierRateGuardApplyResult{}, errors.New("candidate not found")
}

func (f *supplierRateGuardRepoFake) MarkRateGuardChecked(_ context.Context, mappingID int64, _ time.Time) error {
	f.checked = append(f.checked, mappingID)
	return nil
}

func TestSupplierRateGuardRaisesOnlyBelowRoundedTarget(t *testing.T) {
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	repo := &supplierRateGuardRepoFake{candidates: []SupplierRateGuardCandidate{{
		MappingID: 10, ProviderID: 42, ProviderName: "Supplier A", ProviderEnabled: true,
		UpstreamGroupKey: "vip", UpstreamGroupName: "VIP", UpstreamRateMultiplier: 2.5,
		GuardianActive: true, LocalGroupID: 7, LocalGroupName: "VIP Local", LocalGroupStatus: StatusActive,
		LocalRateMultiplier: 2.6, SnapshotAt: now.Add(-time.Minute), GroupSyncStatus: SupplierSyncStatusSuccess,
	}}}
	guard := NewSupplierRateGuardService(repo)

	result, err := guard.Run(context.Background(), SupplierRateGuardConfig{SafetyMultiplier: 1.1, MaxSnapshotAge: 30 * time.Minute}, now)

	require.NoError(t, err)
	require.Equal(t, 1, result.Checked)
	require.Equal(t, 1, result.Raised)
	require.Len(t, repo.applied, 1)
	require.InDelta(t, 2.75, repo.applied[0].TargetRate, 0.000000001)
	require.Equal(t, SupplierRateGuardActionRaised, result.Items[0].Action)
}

func TestSupplierRateGuardDoesNotLowerEqualOrHigherLocalRate(t *testing.T) {
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	repo := &supplierRateGuardRepoFake{candidates: []SupplierRateGuardCandidate{
		validSupplierRateGuardCandidate(10, 2.5, 2.75, now),
		validSupplierRateGuardCandidate(11, 2.5, 3.0, now),
	}}

	result, err := NewSupplierRateGuardService(repo).Run(context.Background(), SupplierRateGuardConfig{SafetyMultiplier: 1.1, MaxSnapshotAge: 30 * time.Minute}, now)

	require.NoError(t, err)
	require.Zero(t, result.Raised)
	require.Equal(t, 2, result.Unchanged)
}

func TestSupplierRateGuardSkipsStaleFailedInactiveAndDuplicateSnapshots(t *testing.T) {
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	stale := validSupplierRateGuardCandidate(10, 2.5, 2, now)
	stale.SnapshotAt = now.Add(-31 * time.Minute)
	failedSync := validSupplierRateGuardCandidate(11, 2.5, 2, now)
	failedSync.GroupSyncStatus = SupplierSyncStatusFailed
	inactiveGuardian := validSupplierRateGuardCandidate(12, 2.5, 2, now)
	inactiveGuardian.GuardianActive = false
	inactiveProvider := validSupplierRateGuardCandidate(13, 2.5, 2, now)
	inactiveProvider.ProviderEnabled = false
	duplicate := validSupplierRateGuardCandidate(14, 2.5, 2, now)
	duplicate.LastSnapshotAt = &duplicate.SnapshotAt
	repo := &supplierRateGuardRepoFake{candidates: []SupplierRateGuardCandidate{stale, failedSync, inactiveGuardian, inactiveProvider, duplicate}}

	result, err := NewSupplierRateGuardService(repo).Run(context.Background(), SupplierRateGuardConfig{SafetyMultiplier: 1.1, MaxSnapshotAge: 30 * time.Minute}, now)

	require.NoError(t, err)
	require.Equal(t, 5, result.Checked)
	require.Equal(t, 1, result.Stale)
	require.Equal(t, 3, result.Invalid)
	require.Equal(t, 1, result.Duplicate)
	require.Empty(t, repo.applied)
	require.Len(t, repo.checked, 5)
}

func TestSupplierRateGuardRoundsUpToHundredth(t *testing.T) {
	require.InDelta(t, 1.11, supplierRateGuardTarget(1.001, 1.1), 0.000000001)
	require.InDelta(t, 2.75, supplierRateGuardTarget(2.5, 1.1), 0.000000001)
}

func TestSupplierRateGuardContinuesAfterOneItemFails(t *testing.T) {
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	repo := &supplierRateGuardRepoFake{
		candidates: []SupplierRateGuardCandidate{
			validSupplierRateGuardCandidate(10, 2.5, 2, now),
			validSupplierRateGuardCandidate(11, 3, 2, now),
		},
		applyErr: map[int64]error{10: errors.New("write failed")},
	}

	result, err := NewSupplierRateGuardService(repo).Run(context.Background(), SupplierRateGuardConfig{SafetyMultiplier: 1.1, MaxSnapshotAge: 30 * time.Minute}, now)

	require.NoError(t, err)
	require.Equal(t, 2, result.Checked)
	require.Equal(t, 1, result.Raised)
	require.Equal(t, 1, result.Failed)
	require.Equal(t, SupplierRateGuardActionFailed, result.Items[0].Action)
	require.Equal(t, SupplierRateGuardActionRaised, result.Items[1].Action)
}

func validSupplierRateGuardCandidate(mappingID int64, upstreamRate, localRate float64, now time.Time) SupplierRateGuardCandidate {
	return SupplierRateGuardCandidate{
		MappingID: mappingID, ProviderID: 42, ProviderName: "Supplier A", ProviderEnabled: true,
		UpstreamGroupKey: "vip", UpstreamGroupName: "VIP", UpstreamRateMultiplier: upstreamRate,
		GuardianActive: true, LocalGroupID: mappingID + 100, LocalGroupName: "Local", LocalGroupStatus: StatusActive,
		LocalRateMultiplier: localRate, SnapshotAt: now.Add(-time.Minute), GroupSyncStatus: SupplierSyncStatusSuccess,
	}
}
