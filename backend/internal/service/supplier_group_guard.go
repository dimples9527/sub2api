package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	RateGuardSelectionModeAuto   = "auto"
	RateGuardSelectionModeManual = "manual"
)

var ErrSupplierRateGuardSelectionInvalid = infraerrors.BadRequest(
	"SUPPLIER_RATE_GUARD_SELECTION_INVALID",
	"supplier rate guard requires an active mapped group",
)

type SupplierGroupGuardRepository interface {
	ListMappingsByLocalGroup(ctx context.Context, localGroupIDs []int64) ([]SupplierProviderGroup, error)
	GetGroupForRateGuard(ctx context.Context, groupID int64) (SupplierProviderGroup, error)
	SelectRateGuard(ctx context.Context, groupID int64, mode string) error
	ClearRateGuard(ctx context.Context, groupID int64, mode string) error
}

type SupplierGroupGuardReconciler struct {
	repo SupplierGroupGuardRepository
}

func NewSupplierGroupGuardReconciler(repo SupplierGroupGuardRepository) *SupplierGroupGuardReconciler {
	return &SupplierGroupGuardReconciler{repo: repo}
}

func (r *SupplierGroupGuardReconciler) ReconcileLocalGroups(ctx context.Context, localGroupIDs []int64) error {
	localGroupIDs = uniquePositiveInt64s(localGroupIDs)
	if r == nil || r.repo == nil || len(localGroupIDs) == 0 {
		return nil
	}
	mappings, err := r.repo.ListMappingsByLocalGroup(ctx, localGroupIDs)
	if err != nil {
		return err
	}
	byLocalGroup := make(map[int64][]SupplierProviderGroup, len(localGroupIDs))
	for _, mapping := range mappings {
		if mapping.LocalGroupID != nil {
			byLocalGroup[*mapping.LocalGroupID] = append(byLocalGroup[*mapping.LocalGroupID], mapping)
		}
	}

	for _, localGroupID := range localGroupIDs {
		groupMappings := byLocalGroup[localGroupID]
		var selected *SupplierProviderGroup
		active := make([]SupplierProviderGroup, 0, len(groupMappings))
		for index := range groupMappings {
			mapping := &groupMappings[index]
			if mapping.RateGuardSelected {
				selected = mapping
			}
			if mapping.Active {
				active = append(active, *mapping)
			}
		}

		if selected != nil && !selected.Active {
			continue
		}
		if len(active) == 1 {
			if selected == nil || selected.ID != active[0].ID || selected.RateGuardSelectionMode != RateGuardSelectionModeAuto {
				if err := r.repo.SelectRateGuard(ctx, active[0].ID, RateGuardSelectionModeAuto); err != nil {
					return err
				}
			}
			continue
		}
		if selected != nil && selected.RateGuardSelectionMode == RateGuardSelectionModeManual {
			continue
		}
		if selected != nil && selected.RateGuardSelectionMode == RateGuardSelectionModeAuto {
			if err := r.repo.ClearRateGuard(ctx, selected.ID, RateGuardSelectionModeAuto); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *SupplierGroupGuardReconciler) SetManualGuard(ctx context.Context, groupID int64, selected bool) error {
	if r == nil || r.repo == nil {
		return ErrSupplierRateGuardSelectionInvalid
	}
	group, err := r.repo.GetGroupForRateGuard(ctx, groupID)
	if err != nil {
		return err
	}
	if !selected {
		return r.repo.ClearRateGuard(ctx, groupID, "")
	}
	if group.LocalGroupID == nil || !group.Active {
		return ErrSupplierRateGuardSelectionInvalid
	}
	mappings, err := r.repo.ListMappingsByLocalGroup(ctx, []int64{*group.LocalGroupID})
	if err != nil {
		return err
	}
	activeCount := 0
	for _, mapping := range mappings {
		if mapping.Active {
			activeCount++
		}
	}
	if activeCount <= 1 {
		return ErrSupplierRateGuardSelectionInvalid
	}
	return r.repo.SelectRateGuard(ctx, groupID, RateGuardSelectionModeManual)
}

func uniquePositiveInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
