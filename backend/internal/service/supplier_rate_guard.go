package service

import (
	"context"
	"fmt"
	"math"
	"time"
)

const (
	SupplierRateGuardActionRaised    = "raised"
	SupplierRateGuardActionUnchanged = "unchanged"
	SupplierRateGuardActionDuplicate = "duplicate"
	SupplierRateGuardActionStale     = "stale"
	SupplierRateGuardActionInvalid   = "invalid"
	SupplierRateGuardActionFailed    = "failed"
)

const (
	SupplierRateGuardReasonProviderInactive   = "provider_inactive"
	SupplierRateGuardReasonGuardianInactive   = "guardian_inactive"
	SupplierRateGuardReasonLocalGroupInactive = "local_group_inactive"
	SupplierRateGuardReasonGroupSyncFailed    = "group_sync_not_success"
	SupplierRateGuardReasonSnapshotStale      = "snapshot_stale"
	SupplierRateGuardReasonSnapshotDuplicate  = "snapshot_duplicate"
	SupplierRateGuardReasonRateInvalid        = "rate_invalid"
	SupplierRateGuardReasonSelectionChanged   = "selection_changed"
	SupplierRateGuardReasonSnapshotChanged    = "snapshot_changed"
)

type SupplierRateGuardConfig struct {
	SafetyMultiplier float64
	MaxSnapshotAge   time.Duration
}

type SupplierRateGuardCandidate struct {
	MappingID              int64
	ProviderID             int64
	ProviderName           string
	ProviderEnabled        bool
	UpstreamGroupKey       string
	UpstreamGroupName      string
	UpstreamRateMultiplier float64
	GuardianActive         bool
	LocalGroupID           int64
	LocalGroupName         string
	LocalGroupStatus       string
	LocalRateMultiplier    float64
	SnapshotAt             time.Time
	LastSnapshotAt         *time.Time
	GroupSyncStatus        string
	LastGroupSyncAt        *time.Time
}

type SupplierRateGuardApplyInput struct {
	MappingID          int64
	ExpectedSnapshotAt time.Time
	CheckedAt          time.Time
	TargetRate         float64
	MaxSnapshotAge     time.Duration
}

type SupplierRateGuardApplyResult struct {
	OldRate    float64
	TargetRate float64
	Action     string
	Reason     string
}

type SupplierRateGuardItemResult struct {
	MappingID         int64     `json:"mapping_id"`
	ProviderID        int64     `json:"provider_id"`
	ProviderName      string    `json:"provider_name"`
	UpstreamGroupKey  string    `json:"upstream_group_key"`
	UpstreamGroupName string    `json:"upstream_group_name"`
	LocalGroupID      int64     `json:"local_group_id"`
	LocalGroupName    string    `json:"local_group_name"`
	SnapshotAt        time.Time `json:"snapshot_at"`
	OldRate           float64   `json:"old_rate"`
	TargetRate        float64   `json:"target_rate"`
	Action            string    `json:"action"`
	Reason            string    `json:"reason,omitempty"`
}

type SupplierRateGuardResult struct {
	Checked   int                           `json:"checked"`
	Raised    int                           `json:"raised"`
	Unchanged int                           `json:"unchanged"`
	Duplicate int                           `json:"duplicate"`
	Stale     int                           `json:"stale"`
	Invalid   int                           `json:"invalid"`
	Failed    int                           `json:"failed"`
	Items     []SupplierRateGuardItemResult `json:"items"`
}

type SupplierRateGuardRepository interface {
	ListRateGuardCandidates(ctx context.Context) ([]SupplierRateGuardCandidate, error)
	ApplyRateGuard(ctx context.Context, input SupplierRateGuardApplyInput) (SupplierRateGuardApplyResult, error)
	MarkRateGuardChecked(ctx context.Context, mappingID int64, checkedAt time.Time) error
}

type SupplierRateGuardService struct {
	repo SupplierRateGuardRepository
}

func NewSupplierRateGuardService(repo SupplierRateGuardRepository) *SupplierRateGuardService {
	return &SupplierRateGuardService{repo: repo}
}

func (s *SupplierRateGuardService) Run(ctx context.Context, config SupplierRateGuardConfig, now time.Time) (SupplierRateGuardResult, error) {
	result := SupplierRateGuardResult{Items: make([]SupplierRateGuardItemResult, 0)}
	if s == nil || s.repo == nil {
		return result, fmt.Errorf("supplier rate guard repository is required")
	}
	if config.SafetyMultiplier <= 0 {
		return result, fmt.Errorf("supplier rate guard safety multiplier must be greater than zero")
	}
	if config.MaxSnapshotAge <= 0 {
		return result, fmt.Errorf("supplier rate guard max snapshot age must be greater than zero")
	}
	candidates, err := s.repo.ListRateGuardCandidates(ctx)
	if err != nil {
		return result, fmt.Errorf("list supplier rate guard candidates: %w", err)
	}
	result.Items = make([]SupplierRateGuardItemResult, 0, len(candidates))
	for _, candidate := range candidates {
		result.Checked++
		item := SupplierRateGuardItemResult{
			MappingID: candidate.MappingID, ProviderID: candidate.ProviderID, ProviderName: candidate.ProviderName,
			UpstreamGroupKey: candidate.UpstreamGroupKey, UpstreamGroupName: candidate.UpstreamGroupName,
			LocalGroupID: candidate.LocalGroupID, LocalGroupName: candidate.LocalGroupName,
			SnapshotAt: candidate.SnapshotAt, OldRate: candidate.LocalRateMultiplier,
		}
		if action, reason := supplierRateGuardSkip(candidate, config, now); action != "" {
			item.Action, item.Reason = action, reason
			if err := s.repo.MarkRateGuardChecked(ctx, candidate.MappingID, now); err != nil {
				item.Action, item.Reason = SupplierRateGuardActionFailed, err.Error()
			}
			result.addItem(item)
			continue
		}
		item.TargetRate = supplierRateGuardTarget(candidate.UpstreamRateMultiplier, config.SafetyMultiplier)
		applied, err := s.repo.ApplyRateGuard(ctx, SupplierRateGuardApplyInput{
			MappingID: candidate.MappingID, ExpectedSnapshotAt: candidate.SnapshotAt,
			CheckedAt: now, TargetRate: item.TargetRate, MaxSnapshotAge: config.MaxSnapshotAge,
		})
		if err != nil {
			item.Action, item.Reason = SupplierRateGuardActionFailed, err.Error()
		} else {
			item.OldRate = applied.OldRate
			item.TargetRate = applied.TargetRate
			item.Action = applied.Action
			item.Reason = applied.Reason
		}
		result.addItem(item)
	}
	return result, nil
}

func supplierRateGuardSkip(candidate SupplierRateGuardCandidate, config SupplierRateGuardConfig, now time.Time) (string, string) {
	switch {
	case !candidate.ProviderEnabled:
		return SupplierRateGuardActionInvalid, SupplierRateGuardReasonProviderInactive
	case !candidate.GuardianActive:
		return SupplierRateGuardActionInvalid, SupplierRateGuardReasonGuardianInactive
	case candidate.LocalGroupID <= 0 || candidate.LocalGroupStatus != StatusActive:
		return SupplierRateGuardActionInvalid, SupplierRateGuardReasonLocalGroupInactive
	case candidate.GroupSyncStatus != SupplierSyncStatusSuccess:
		return SupplierRateGuardActionInvalid, SupplierRateGuardReasonGroupSyncFailed
	case candidate.SnapshotAt.IsZero() || now.Sub(candidate.SnapshotAt) > config.MaxSnapshotAge:
		return SupplierRateGuardActionStale, SupplierRateGuardReasonSnapshotStale
	case candidate.LastSnapshotAt != nil && !candidate.SnapshotAt.After(*candidate.LastSnapshotAt):
		return SupplierRateGuardActionDuplicate, SupplierRateGuardReasonSnapshotDuplicate
	case candidate.UpstreamRateMultiplier <= 0 || candidate.LocalRateMultiplier <= 0:
		return SupplierRateGuardActionInvalid, SupplierRateGuardReasonRateInvalid
	default:
		return "", ""
	}
}

func supplierRateGuardTarget(upstream, safety float64) float64 {
	return math.Ceil(upstream*safety*100-1e-9) / 100
}

func (r *SupplierRateGuardResult) addItem(item SupplierRateGuardItemResult) {
	switch item.Action {
	case SupplierRateGuardActionRaised:
		r.Raised++
	case SupplierRateGuardActionUnchanged:
		r.Unchanged++
	case SupplierRateGuardActionDuplicate:
		r.Duplicate++
	case SupplierRateGuardActionStale:
		r.Stale++
	case SupplierRateGuardActionInvalid:
		r.Invalid++
	default:
		r.Failed++
	}
	r.Items = append(r.Items, item)
}
