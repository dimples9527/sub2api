package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type supplierGroupGuardRepoFake struct {
	mappings []SupplierProviderGroup
}

func (f *supplierGroupGuardRepoFake) ListMappingsByLocalGroup(_ context.Context, localGroupIDs []int64) ([]SupplierProviderGroup, error) {
	wanted := make(map[int64]struct{}, len(localGroupIDs))
	for _, id := range localGroupIDs {
		wanted[id] = struct{}{}
	}
	result := make([]SupplierProviderGroup, 0)
	for _, mapping := range f.mappings {
		if mapping.LocalGroupID == nil {
			continue
		}
		if _, ok := wanted[*mapping.LocalGroupID]; ok {
			result = append(result, mapping)
		}
	}
	return result, nil
}

func (f *supplierGroupGuardRepoFake) GetGroupForRateGuard(_ context.Context, groupID int64) (SupplierProviderGroup, error) {
	for _, mapping := range f.mappings {
		if mapping.ID == groupID {
			return mapping, nil
		}
	}
	return SupplierProviderGroup{}, ErrSupplierProviderGroupNotFound
}

func (f *supplierGroupGuardRepoFake) SelectRateGuard(_ context.Context, groupID int64, mode string) error {
	for index := range f.mappings {
		if f.mappings[index].ID != groupID {
			continue
		}
		localGroupID := f.mappings[index].LocalGroupID
		if localGroupID == nil || !f.mappings[index].Active {
			return ErrSupplierRateGuardSelectionInvalid
		}
		for other := range f.mappings {
			if f.mappings[other].LocalGroupID != nil && *f.mappings[other].LocalGroupID == *localGroupID {
				f.mappings[other].RateGuardSelected = false
				f.mappings[other].RateGuardSelectionMode = ""
			}
		}
		f.mappings[index].RateGuardSelected = true
		f.mappings[index].RateGuardSelectionMode = mode
		return nil
	}
	return ErrSupplierProviderGroupNotFound
}

func (f *supplierGroupGuardRepoFake) ClearRateGuard(_ context.Context, groupID int64, mode string) error {
	for index := range f.mappings {
		if f.mappings[index].ID == groupID && (mode == "" || f.mappings[index].RateGuardSelectionMode == mode) {
			f.mappings[index].RateGuardSelected = false
			f.mappings[index].RateGuardSelectionMode = ""
		}
	}
	return nil
}

func TestSupplierGroupGuardReconcilerSelectsUniqueAutomaticMapping(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{{
		ID: 10, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true,
	}}}

	err := NewSupplierGroupGuardReconciler(repo).ReconcileLocalGroups(context.Background(), []int64{7})

	require.NoError(t, err)
	require.True(t, repo.mappings[0].RateGuardSelected)
	require.Equal(t, RateGuardSelectionModeAuto, repo.mappings[0].RateGuardSelectionMode)
}

func TestSupplierGroupGuardReconcilerConvertsSoleManualMappingToAutomaticGuard(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{{
		ID: 10, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusManual, Active: true,
		RateGuardSelected: true, RateGuardSelectionMode: RateGuardSelectionModeManual,
	}}}

	err := NewSupplierGroupGuardReconciler(repo).ReconcileLocalGroups(context.Background(), []int64{7})

	require.NoError(t, err)
	require.True(t, repo.mappings[0].RateGuardSelected)
	require.Equal(t, RateGuardSelectionModeAuto, repo.mappings[0].RateGuardSelectionMode)
}

func TestSupplierGroupGuardReconcilerClearsAutomaticGuardOnMultipleActiveMappings(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{
		{ID: 10, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true, RateGuardSelected: true, RateGuardSelectionMode: RateGuardSelectionModeAuto},
		{ID: 11, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true},
	}}

	err := NewSupplierGroupGuardReconciler(repo).ReconcileLocalGroups(context.Background(), []int64{7})

	require.NoError(t, err)
	require.False(t, repo.mappings[0].RateGuardSelected)
}

func TestSupplierGroupGuardReconcilerPreservesManualGuardOnMultipleMappings(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{
		{ID: 10, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true, RateGuardSelected: true, RateGuardSelectionMode: RateGuardSelectionModeManual},
		{ID: 11, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true},
	}}

	err := NewSupplierGroupGuardReconciler(repo).ReconcileLocalGroups(context.Background(), []int64{7})

	require.NoError(t, err)
	require.True(t, repo.mappings[0].RateGuardSelected)
	require.Equal(t, RateGuardSelectionModeManual, repo.mappings[0].RateGuardSelectionMode)
}

func TestSupplierGroupGuardReconcilerSelectsSoleAutomaticMappingAfterConflictRemoved(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{
		{ID: 10, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: false},
		{ID: 11, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true},
	}}

	err := NewSupplierGroupGuardReconciler(repo).ReconcileLocalGroups(context.Background(), []int64{7})

	require.NoError(t, err)
	require.True(t, repo.mappings[1].RateGuardSelected)
	require.Equal(t, RateGuardSelectionModeAuto, repo.mappings[1].RateGuardSelectionMode)
}

func TestSupplierGroupGuardReconcilerKeepsInactiveSelectedGuardian(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{
		{ID: 10, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: false, RateGuardSelected: true, RateGuardSelectionMode: RateGuardSelectionModeAuto},
		{ID: 11, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true},
	}}

	err := NewSupplierGroupGuardReconciler(repo).ReconcileLocalGroups(context.Background(), []int64{7})

	require.NoError(t, err)
	require.True(t, repo.mappings[0].RateGuardSelected)
	require.False(t, repo.mappings[1].RateGuardSelected)
}

func TestSupplierGroupGuardReconcilerRejectsInvalidManualSelection(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{{ID: 10, Active: true}}}
	reconciler := NewSupplierGroupGuardReconciler(repo)

	err := reconciler.SetManualGuard(context.Background(), 10, true)

	require.ErrorIs(t, err, ErrSupplierRateGuardSelectionInvalid)
}

func TestSupplierGroupGuardReconcilerRejectsManualSelectionForSoleActiveMapping(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{{
		ID: 10, LocalGroupID: int64PtrForMatcher(7), Active: true,
	}}}
	reconciler := NewSupplierGroupGuardReconciler(repo)

	err := reconciler.SetManualGuard(context.Background(), 10, true)

	require.ErrorIs(t, err, ErrSupplierRateGuardSelectionInvalid)
}

func TestSupplierGroupGuardReconcilerSelectsAndClearsManualGuard(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{
		{ID: 10, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true},
		{ID: 11, LocalGroupID: int64PtrForMatcher(7), AutoMatchStatus: AutoMatchStatusAutoMatched, Active: true},
	}}
	reconciler := NewSupplierGroupGuardReconciler(repo)

	require.NoError(t, reconciler.SetManualGuard(context.Background(), 10, true))
	require.True(t, repo.mappings[0].RateGuardSelected)
	require.Equal(t, RateGuardSelectionModeManual, repo.mappings[0].RateGuardSelectionMode)

	require.NoError(t, reconciler.SetManualGuard(context.Background(), 10, false))
	require.False(t, repo.mappings[0].RateGuardSelected)
}

func TestSupplierGroupGuardReconcilerClearsInactiveManualGuard(t *testing.T) {
	repo := &supplierGroupGuardRepoFake{mappings: []SupplierProviderGroup{{
		ID: 10, LocalGroupID: int64PtrForMatcher(7), Active: false,
		RateGuardSelected: true, RateGuardSelectionMode: RateGuardSelectionModeManual,
	}}}

	err := NewSupplierGroupGuardReconciler(repo).SetManualGuard(context.Background(), 10, false)

	require.NoError(t, err)
	require.False(t, repo.mappings[0].RateGuardSelected)
}
