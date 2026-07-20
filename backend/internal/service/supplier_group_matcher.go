package service

import (
	"context"
	"strings"
	"unicode"
)

const (
	AutoMatchStatusUnmatched   = "unmatched"
	AutoMatchStatusAutoMatched = "auto_matched"
	AutoMatchStatusManual      = "manual"
	AutoMatchStatusAmbiguous   = "ambiguous"

	NameChangeActionKeepLocal = "keep_local"
	NameChangeActionSyncLocal = "sync_local_name"
)

type SupplierProviderGroupMatcherDataRepository interface {
	ListGroupsForAutoMatch(ctx context.Context, providerID int64) ([]SupplierProviderGroup, error)
	ApplyAutoMatch(ctx context.Context, groupID, localGroupID int64, matchedUpstreamName string) (bool, error)
	UpdateAutoMatchState(ctx context.Context, groupID int64, status string, nameChangePending bool) error
	GetGroupForAutoMatch(ctx context.Context, groupID int64) (SupplierProviderGroup, error)
	UpdateGroupMapping(ctx context.Context, groupID int64, localGroupID *int64) error
	UpdateAutoMatchIgnored(ctx context.Context, groupID int64, ignored bool) error
	AcknowledgeNameChange(ctx context.Context, groupID int64, matchedUpstreamName string) error
}

type SupplierProviderGroupMatcherLocalRepository interface {
	ListActive(ctx context.Context) ([]Group, error)
	GetByID(ctx context.Context, id int64) (*Group, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	Update(ctx context.Context, group *Group) error
}

type SupplierGroupAutoMatchResult struct {
	ProviderID    int64 `json:"provider_id"`
	Scanned       int   `json:"scanned"`
	AutoMatched   int   `json:"auto_matched"`
	Ambiguous     int   `json:"ambiguous"`
	Ignored       int   `json:"ignored"`
	NoCandidate   int   `json:"no_candidate"`
	AlreadyMapped int   `json:"already_mapped"`
}

type SupplierProviderGroupMatcher struct {
	dataRepo  SupplierProviderGroupMatcherDataRepository
	localRepo SupplierProviderGroupMatcherLocalRepository
	guard     *SupplierGroupGuardReconciler
}

func NewSupplierProviderGroupMatcher(dataRepo SupplierProviderGroupMatcherDataRepository, localRepo SupplierProviderGroupMatcherLocalRepository) *SupplierProviderGroupMatcher {
	return &SupplierProviderGroupMatcher{dataRepo: dataRepo, localRepo: localRepo}
}

func (m *SupplierProviderGroupMatcher) SetGuardReconciler(guard *SupplierGroupGuardReconciler) {
	if m != nil {
		m.guard = guard
	}
}

func (m *SupplierProviderGroupMatcher) AutoMatch(ctx context.Context, providerID int64) (SupplierGroupAutoMatchResult, error) {
	result := SupplierGroupAutoMatchResult{ProviderID: providerID}
	if m == nil || m.dataRepo == nil || m.localRepo == nil {
		return result, nil
	}
	upstreamGroups, err := m.dataRepo.ListGroupsForAutoMatch(ctx, providerID)
	if err != nil {
		return result, err
	}
	return m.autoMatchGroups(ctx, providerID, upstreamGroups)
}

func (m *SupplierProviderGroupMatcher) autoMatchGroups(ctx context.Context, providerID int64, upstreamGroups []SupplierProviderGroup) (SupplierGroupAutoMatchResult, error) {
	result := SupplierGroupAutoMatchResult{ProviderID: providerID}
	affectedLocalGroupIDs := make([]int64, 0, len(upstreamGroups))
	localGroups, err := m.localRepo.ListActive(ctx)
	if err != nil {
		return result, err
	}
	candidates := make(map[string][]Group)
	for _, localGroup := range localGroups {
		if localGroup.Status != "" && localGroup.Status != StatusActive {
			continue
		}
		key := normalizeSupplierGroupMatchName(localGroup.Name)
		if key != "" {
			candidates[key] = append(candidates[key], localGroup)
		}
	}

	for _, upstreamGroup := range upstreamGroups {
		result.Scanned++
		if upstreamGroup.LocalGroupID != nil {
			affectedLocalGroupIDs = append(affectedLocalGroupIDs, *upstreamGroup.LocalGroupID)
			status := upstreamGroup.AutoMatchStatus
			if status == "" {
				status = AutoMatchStatusManual
			}
			matchedName := supplierGroupMatchedName(upstreamGroup)
			pending := upstreamGroup.MatchedUpstreamName != "" &&
				normalizeSupplierGroupMatchName(upstreamGroup.MatchedUpstreamName) != supplierGroupMatchKey(upstreamGroup.Name, upstreamGroup.UpstreamKey)
			if pending != upstreamGroup.NameChangePending {
				if err := m.dataRepo.UpdateAutoMatchState(ctx, upstreamGroup.ID, status, pending); err != nil {
					return result, err
				}
			} else if !pending && upstreamGroup.MatchedUpstreamName != matchedName {
				if err := m.dataRepo.AcknowledgeNameChange(ctx, upstreamGroup.ID, matchedName); err != nil {
					return result, err
				}
			}
			result.AlreadyMapped++
			continue
		}
		if upstreamGroup.AutoMatchIgnored {
			result.Ignored++
			continue
		}
		key := supplierGroupMatchKey(upstreamGroup.Name, upstreamGroup.UpstreamKey)
		matches := candidates[key]
		switch len(matches) {
		case 1:
			updated, err := m.dataRepo.ApplyAutoMatch(ctx, upstreamGroup.ID, matches[0].ID, supplierGroupMatchedName(upstreamGroup))
			if err != nil {
				return result, err
			}
			if updated {
				result.AutoMatched++
				affectedLocalGroupIDs = append(affectedLocalGroupIDs, matches[0].ID)
			}
		case 0:
			if err := m.dataRepo.UpdateAutoMatchState(ctx, upstreamGroup.ID, AutoMatchStatusUnmatched, false); err != nil {
				return result, err
			}
			result.NoCandidate++
		default:
			if err := m.dataRepo.UpdateAutoMatchState(ctx, upstreamGroup.ID, AutoMatchStatusAmbiguous, false); err != nil {
				return result, err
			}
			result.Ambiguous++
		}
	}
	if m.guard != nil {
		if err := m.guard.ReconcileLocalGroups(ctx, affectedLocalGroupIDs); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (m *SupplierProviderGroupMatcher) UpdateMapping(ctx context.Context, groupID int64, localGroupID *int64) error {
	group, err := m.dataRepo.GetGroupForAutoMatch(ctx, groupID)
	if err != nil {
		return err
	}
	if err := m.dataRepo.UpdateGroupMapping(ctx, groupID, localGroupID); err != nil {
		return err
	}
	if m.guard == nil {
		return nil
	}
	affected := make([]int64, 0, 2)
	if group.LocalGroupID != nil {
		affected = append(affected, *group.LocalGroupID)
	}
	if localGroupID != nil {
		affected = append(affected, *localGroupID)
	}
	return m.guard.ReconcileLocalGroups(ctx, affected)
}

func (m *SupplierProviderGroupMatcher) SetIgnored(ctx context.Context, groupID int64, ignored bool) (SupplierGroupAutoMatchResult, error) {
	group, err := m.dataRepo.GetGroupForAutoMatch(ctx, groupID)
	if err != nil {
		return SupplierGroupAutoMatchResult{}, err
	}
	if err := m.dataRepo.UpdateAutoMatchIgnored(ctx, groupID, ignored); err != nil {
		return SupplierGroupAutoMatchResult{}, err
	}
	result := SupplierGroupAutoMatchResult{ProviderID: group.ProviderID}
	if ignored || group.LocalGroupID != nil || !group.Active {
		return result, nil
	}
	group.AutoMatchIgnored = false
	return m.autoMatchGroups(ctx, group.ProviderID, []SupplierProviderGroup{group})
}

func (m *SupplierProviderGroupMatcher) ResolveNameChange(ctx context.Context, groupID int64, action string) error {
	group, err := m.dataRepo.GetGroupForAutoMatch(ctx, groupID)
	if err != nil {
		return err
	}
	if group.LocalGroupID == nil {
		return ErrSupplierLocalGroupNotFound
	}
	matchedName := supplierGroupMatchedName(group)
	switch action {
	case NameChangeActionKeepLocal:
		return m.dataRepo.AcknowledgeNameChange(ctx, groupID, matchedName)
	case NameChangeActionSyncLocal:
		localGroup, err := m.localRepo.GetByID(ctx, *group.LocalGroupID)
		if err != nil {
			return err
		}
		if localGroup.Name != matchedName {
			exists, err := m.localRepo.ExistsByName(ctx, matchedName)
			if err != nil {
				return err
			}
			if exists {
				return ErrGroupExists
			}
			localGroup.Name = matchedName
			if err := m.localRepo.Update(ctx, localGroup); err != nil {
				return err
			}
		}
		return m.dataRepo.AcknowledgeNameChange(ctx, groupID, matchedName)
	default:
		return ErrSupplierProviderInvalid
	}
}

func normalizeSupplierGroupMatchName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var normalized strings.Builder
	normalized.Grow(len(name))
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(r)
		}
	}
	return normalized.String()
}

func supplierGroupMatchKey(name, upstreamKey string) string {
	if strings.TrimSpace(name) == "" {
		name = upstreamKey
	}
	return normalizeSupplierGroupMatchName(name)
}

func supplierGroupMatchedName(group SupplierProviderGroup) string {
	if name := strings.TrimSpace(group.Name); name != "" {
		return name
	}
	return strings.TrimSpace(group.UpstreamKey)
}

func resolveSupplierGroupCandidates(localGroups []Group, matchKey string) (Group, bool, bool) {
	matchKey = normalizeSupplierGroupMatchName(matchKey)
	var matched Group
	count := 0
	for _, localGroup := range localGroups {
		if normalizeSupplierGroupMatchName(localGroup.Name) != matchKey {
			continue
		}
		matched = localGroup
		count++
	}
	if count == 1 {
		return matched, true, false
	}
	return Group{}, false, count > 1
}
