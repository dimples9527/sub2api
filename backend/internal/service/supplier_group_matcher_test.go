package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type supplierGroupMatcherDataRepoFake struct {
	groups  []SupplierProviderGroup
	state   map[int64]string
	pending map[int64]bool
	bound   map[int64]int64
	ignored map[int64]bool
	acked   map[int64]string
}

func (f *supplierGroupMatcherDataRepoFake) ListGroupsForAutoMatch(_ context.Context, providerID int64) ([]SupplierProviderGroup, error) {
	groups := make([]SupplierProviderGroup, 0, len(f.groups))
	for _, group := range f.groups {
		if providerID == 0 || group.ProviderID == providerID {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func (f *supplierGroupMatcherDataRepoFake) ApplyAutoMatch(_ context.Context, groupID, localGroupID int64, _ string) (bool, error) {
	if f.bound == nil {
		f.bound = make(map[int64]int64)
	}
	f.bound[groupID] = localGroupID
	return true, nil
}

func (f *supplierGroupMatcherDataRepoFake) UpdateAutoMatchState(_ context.Context, groupID int64, status string, pending bool) error {
	if f.state == nil {
		f.state = make(map[int64]string)
	}
	if f.pending == nil {
		f.pending = make(map[int64]bool)
	}
	f.state[groupID] = status
	f.pending[groupID] = pending
	return nil
}

func (f *supplierGroupMatcherDataRepoFake) GetGroupForAutoMatch(_ context.Context, groupID int64) (SupplierProviderGroup, error) {
	for _, group := range f.groups {
		if group.ID == groupID {
			return group, nil
		}
	}
	return SupplierProviderGroup{}, ErrSupplierProviderGroupNotFound
}

func (f *supplierGroupMatcherDataRepoFake) UpdateGroupMapping(context.Context, int64, *int64) error {
	return nil
}

func (f *supplierGroupMatcherDataRepoFake) UpdateAutoMatchIgnored(_ context.Context, groupID int64, ignored bool) error {
	if f.ignored == nil {
		f.ignored = make(map[int64]bool)
	}
	f.ignored[groupID] = ignored
	for index := range f.groups {
		if f.groups[index].ID == groupID {
			f.groups[index].AutoMatchIgnored = ignored
		}
	}
	return nil
}

func (f *supplierGroupMatcherDataRepoFake) AcknowledgeNameChange(_ context.Context, groupID int64, matchedUpstreamName string) error {
	if f.acked == nil {
		f.acked = make(map[int64]string)
	}
	f.acked[groupID] = matchedUpstreamName
	return nil
}

type supplierGroupMatcherLocalRepoFake struct {
	groups []Group
}

func (f *supplierGroupMatcherLocalRepoFake) ListActive(context.Context) ([]Group, error) {
	return f.groups, nil
}

func (f *supplierGroupMatcherLocalRepoFake) GetByID(_ context.Context, id int64) (*Group, error) {
	for index := range f.groups {
		if f.groups[index].ID == id {
			group := f.groups[index]
			return &group, nil
		}
	}
	return nil, ErrGroupNotFound
}

func (f *supplierGroupMatcherLocalRepoFake) ExistsByName(_ context.Context, name string) (bool, error) {
	for _, group := range f.groups {
		if group.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (f *supplierGroupMatcherLocalRepoFake) Update(_ context.Context, updated *Group) error {
	for index := range f.groups {
		if f.groups[index].ID == updated.ID {
			f.groups[index] = *updated
			return nil
		}
	}
	return ErrGroupNotFound
}

func TestSupplierProviderGroupMatcherMatchesOnlyUniqueActiveName(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{
		{ID: 1, ProviderID: 42, Name: " VIP-Plus ", UpstreamKey: "vip-plus"},
		{ID: 2, ProviderID: 42, Name: "Standard", UpstreamKey: "standard"},
	}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{
		{ID: 7, Name: "vip plus", Status: StatusActive},
		{ID: 8, Name: "Standard", Status: StatusActive},
	}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	result, err := matcher.AutoMatch(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, 2, result.Scanned)
	require.Equal(t, 2, result.AutoMatched)
	require.Equal(t, int64(7), dataRepo.bound[1])
	require.Equal(t, int64(8), dataRepo.bound[2])
}

func TestSupplierProviderGroupMatcherLeavesAmbiguousAndIgnoredGroupsUnmatched(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{
		{ID: 1, ProviderID: 42, Name: "VIP Plus", UpstreamKey: "vip-plus"},
		{ID: 2, ProviderID: 42, Name: "VIP_Ignore", UpstreamKey: "vip-ignore", AutoMatchIgnored: true},
	}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{
		{ID: 7, Name: "vip-plus", Status: StatusActive},
		{ID: 8, Name: "VIP Plus", Status: StatusActive},
	}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	result, err := matcher.AutoMatch(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, 1, result.Ambiguous)
	require.Equal(t, 1, result.Ignored)
	require.Empty(t, dataRepo.bound)
	require.Equal(t, AutoMatchStatusAmbiguous, dataRepo.state[1])
}

func TestSupplierProviderGroupMatcherDoesNotOverwriteExistingMapping(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{
		{ID: 1, ProviderID: 42, Name: "VIP", UpstreamKey: "vip", LocalGroupID: int64PtrForMatcher(99)},
	}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{{ID: 7, Name: "VIP", Status: StatusActive}}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	result, err := matcher.AutoMatch(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, 1, result.AlreadyMapped)
	require.Empty(t, dataRepo.bound)
}

func TestSupplierProviderGroupMatcherMarksNormalizedNameChangeWithoutUnbinding(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{{
		ID: 1, ProviderID: 42, Name: "VIP New", UpstreamKey: "vip-new", LocalGroupID: int64PtrForMatcher(99),
		AutoMatchStatus: AutoMatchStatusManual, MatchedUpstreamName: "VIP Old",
	}}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{{ID: 99, Name: "VIP Old", Status: StatusActive}}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	result, err := matcher.AutoMatch(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, 1, result.AlreadyMapped)
	require.Empty(t, dataRepo.bound)
	require.Equal(t, AutoMatchStatusManual, dataRepo.state[1])
	require.True(t, dataRepo.pending[1])
}

func TestSupplierProviderGroupMatcherRefreshesEquivalentNameSnapshotWithoutWarning(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{{
		ID: 1, ProviderID: 42, Name: "VIP Plus", UpstreamKey: "vip-plus", LocalGroupID: int64PtrForMatcher(99),
		AutoMatchStatus: AutoMatchStatusManual, MatchedUpstreamName: "VIP-Plus",
	}}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{{ID: 99, Name: "VIP Plus", Status: StatusActive}}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	_, err := matcher.AutoMatch(context.Background(), 42)

	require.NoError(t, err)
	require.Equal(t, "VIP Plus", dataRepo.acked[1])
	require.False(t, dataRepo.pending[1])
}

func TestSupplierProviderGroupMatcherDisablingIgnoreImmediatelyRetriesMatch(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{{
		ID: 1, ProviderID: 42, Name: "VIP", UpstreamKey: "vip", Active: true, AutoMatchIgnored: true,
	}}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{{ID: 7, Name: "VIP", Status: StatusActive}}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	result, err := matcher.SetIgnored(context.Background(), 1, false)

	require.NoError(t, err)
	require.False(t, dataRepo.ignored[1])
	require.Equal(t, 1, result.AutoMatched)
	require.Equal(t, int64(7), dataRepo.bound[1])
}

func TestSupplierProviderGroupMatcherDoesNotMatchInactiveUpstreamWhenIgnoreRemoved(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{{
		ID: 1, ProviderID: 42, Name: "VIP", UpstreamKey: "vip", Active: false, AutoMatchIgnored: true,
	}}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{{ID: 7, Name: "VIP", Status: StatusActive}}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	result, err := matcher.SetIgnored(context.Background(), 1, false)

	require.NoError(t, err)
	require.Zero(t, result.AutoMatched)
	require.Empty(t, dataRepo.bound)
}

func TestSupplierProviderGroupMatcherResolvesNameChangeByRenamingLocalGroup(t *testing.T) {
	dataRepo := &supplierGroupMatcherDataRepoFake{groups: []SupplierProviderGroup{{
		ID: 1, ProviderID: 42, Name: "VIP New", LocalGroupID: int64PtrForMatcher(7),
		MatchedUpstreamName: "VIP Old", NameChangePending: true,
	}}}
	localRepo := &supplierGroupMatcherLocalRepoFake{groups: []Group{{ID: 7, Name: "VIP Old", Status: StatusActive}}}

	matcher := NewSupplierProviderGroupMatcher(dataRepo, localRepo)
	err := matcher.ResolveNameChange(context.Background(), 1, NameChangeActionSyncLocal)

	require.NoError(t, err)
	require.Equal(t, "VIP New", localRepo.groups[0].Name)
	require.Equal(t, "VIP New", dataRepo.acked[1])
}

func int64PtrForMatcher(value int64) *int64 {
	return &value
}

func TestNormalizeSupplierGroupMatchNameRemovesNonLettersAndDigits(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "unicode punctuation and spaces", input: " GPT-4（专线） ", want: "gpt4专线"},
		{name: "case and separators", input: " VIP _ A / Plus ", want: "vipaplus"},
		{name: "emoji and punctuation", input: "A+B😀", want: "ab"},
		{name: "empty", input: " -_ / ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeSupplierGroupMatchName(tt.input))
		})
	}
}

func TestSupplierGroupMatchKeyFallsBackToUpstreamKeyWhenNameEmpty(t *testing.T) {
	require.Equal(t, "vipplus", supplierGroupMatchKey("", " VIP-Plus "))
	require.Equal(t, "vipplus", supplierGroupMatchKey(" VIP Plus ", "ignored-key"))
	require.Empty(t, supplierGroupMatchKey(" -_ ", " / "))
}

func TestResolveSupplierGroupCandidatesRequiresUniqueLocalGroup(t *testing.T) {
	localGroups := []Group{
		{ID: 7, Name: "VIP-Plus"},
		{ID: 8, Name: "Standard"},
	}

	matched, ok, ambiguous := resolveSupplierGroupCandidates(localGroups, "vip plus")
	require.True(t, ok)
	require.False(t, ambiguous)
	require.Equal(t, int64(7), matched.ID)

	matched, ok, ambiguous = resolveSupplierGroupCandidates(
		[]Group{{ID: 7, Name: "VIP Plus"}, {ID: 8, Name: "vip_plus"}},
		"VIP-Plus",
	)
	require.False(t, ok)
	require.True(t, ambiguous)
	require.Zero(t, matched.ID)

	matched, ok, ambiguous = resolveSupplierGroupCandidates(localGroups, "missing")
	require.False(t, ok)
	require.False(t, ambiguous)
	require.Zero(t, matched.ID)
}
