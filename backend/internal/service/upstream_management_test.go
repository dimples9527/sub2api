package service

import (
	"context"
	"encoding/json"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type upstreamManagementProviderSourceStub struct {
	defaultProvider UpstreamProviderConfig
	keys            []UpstreamProviderKey
	defaultErr      error
	keysErr         error
	fetchedSlug     string
}

func (s *upstreamManagementProviderSourceStub) GetDefaultProvider(ctx context.Context) (UpstreamProviderConfig, error) {
	return s.defaultProvider, s.defaultErr
}

func (s *upstreamManagementProviderSourceStub) FetchProviderKeys(ctx context.Context, slug string) ([]UpstreamProviderKey, []string, error) {
	s.fetchedSlug = slug
	return s.keys, []string{"upstream warning"}, s.keysErr
}

type upstreamManagementGroupRepoStub struct {
	groups  []Group
	updates []Group
}

func (s *upstreamManagementGroupRepoStub) Create(context.Context, *Group) error { panic("unexpected") }

func (s *upstreamManagementGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	for i := range s.groups {
		if s.groups[i].ID == id {
			group := s.groups[i]
			return &group, nil
		}
	}
	return nil, ErrGroupNotFound
}

func (s *upstreamManagementGroupRepoStub) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return s.GetByID(ctx, id)
}

func (s *upstreamManagementGroupRepoStub) Update(_ context.Context, group *Group) error {
	s.updates = append(s.updates, *group)
	for i := range s.groups {
		if s.groups[i].ID == group.ID {
			s.groups[i] = *group
		}
	}
	return nil
}

func (s *upstreamManagementGroupRepoStub) Delete(context.Context, int64) error { panic("unexpected") }

func (s *upstreamManagementGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	out := make([]Group, len(s.groups))
	copy(out, s.groups)
	return out, nil
}

func (s *upstreamManagementGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected")
}

type upstreamManagementSettingRepoStub struct {
	values map[string]string
}

func newUpstreamManagementSettingRepoStub() *upstreamManagementSettingRepoStub {
	return &upstreamManagementSettingRepoStub{values: map[string]string{}}
}

func (s *upstreamManagementSettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *upstreamManagementSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *upstreamManagementSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *upstreamManagementSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *upstreamManagementSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *upstreamManagementSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *upstreamManagementSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type upstreamManagementAuthCacheInvalidatorStub struct {
	groupIDs []int64
}

func (s *upstreamManagementAuthCacheInvalidatorStub) InvalidateAuthCacheByUserID(context.Context, int64) {
}
func (s *upstreamManagementAuthCacheInvalidatorStub) InvalidateAuthCacheByKeyID(context.Context, int64) {
}
func (s *upstreamManagementAuthCacheInvalidatorStub) InvalidateAuthCacheByKey(context.Context, string) {
}
func (s *upstreamManagementAuthCacheInvalidatorStub) InvalidateAuthCacheByGroupID(_ context.Context, groupID int64) {
	s.groupIDs = append(s.groupIDs, groupID)
}
func (s *upstreamManagementAuthCacheInvalidatorStub) InvalidateAuthCacheByAllUsers(context.Context) {}

func TestUpstreamManagementServiceCompareGroupsUsesDefaultProviderOnly(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "other", GroupName: "Ignored Source", RateMultiplier: 9},
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{ID: 1, Name: "VIP", RateMultiplier: 1.5, Status: StatusActive}}}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	result, err := svc.CompareGroups(context.Background())
	if err != nil {
		t.Fatalf("CompareGroups returned error: %v", err)
	}
	if providerSource.fetchedSlug != "default-upstream" {
		t.Fatalf("fetched slug = %q, want default-upstream", providerSource.fetchedSlug)
	}
	if result.DefaultProvider.Slug != "default-upstream" {
		t.Fatalf("default provider slug = %q", result.DefaultProvider.Slug)
	}
}

func TestUpstreamManagementServiceCompareGroupsMatchesByNormalizedName(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: " vip ", RateMultiplier: 2},
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{ID: 7, Name: "ViP", RateMultiplier: 3, Status: StatusActive}}}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	result, err := svc.CompareGroups(context.Background())
	if err != nil {
		t.Fatalf("CompareGroups returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if !item.Matched || item.LocalGroupID == nil || *item.LocalGroupID != 7 {
		t.Fatalf("expected matched local group 7, got %+v", item)
	}
	if item.UpstreamRate != 2.5 {
		t.Fatalf("upstream rate = %v, want max 2.5", item.UpstreamRate)
	}
	if item.UpstreamKeyCount != 2 {
		t.Fatalf("upstream key count = %d, want 2", item.UpstreamKeyCount)
	}
}

func TestUpstreamManagementServiceApplyRateFixesRaisesOnlyLowerLocalRates(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
			{ProviderSlug: "default-upstream", GroupName: "MAX", RateMultiplier: 2},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{
		{ID: 1, Name: "VIP", RateMultiplier: 1, Status: StatusActive},
		{ID: 2, Name: "MAX", RateMultiplier: 3, Status: StatusActive},
	}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	cache := &upstreamManagementAuthCacheInvalidatorStub{}
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, cache)

	result, err := svc.ApplyRateFixes(context.Background())
	if err != nil {
		t.Fatalf("ApplyRateFixes returned error: %v", err)
	}
	if len(groupRepo.updates) != 1 {
		t.Fatalf("update count = %d, want 1", len(groupRepo.updates))
	}
	if groupRepo.updates[0].ID != 1 || groupRepo.updates[0].RateMultiplier != 2.5 {
		t.Fatalf("unexpected update: %+v", groupRepo.updates[0])
	}
	if len(result.Records) != 1 || result.Records[0].OldRate != 1 || result.Records[0].NewRate != 2.5 {
		t.Fatalf("unexpected records: %+v", result.Records)
	}
	if len(cache.groupIDs) != 1 || cache.groupIDs[0] != 1 {
		t.Fatalf("invalidated group IDs = %v, want [1]", cache.groupIDs)
	}
	var stored []UpstreamGroupRateFixRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupRateFixRecords]), &stored); err != nil {
		t.Fatalf("stored records should be JSON: %v", err)
	}
	if len(stored) != 1 || stored[0].GroupID != 1 {
		t.Fatalf("stored records = %+v", stored)
	}
}

func TestUpstreamManagementServiceCompareGroupsRequiresDefaultProvider(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultErr: infraerrors.NotFound("UPSTREAM_PROVIDER_DEFAULT_NOT_CONFIGURED", "default upstream provider is not configured"),
	}
	svc := NewUpstreamManagementService(providerSource, &upstreamManagementGroupRepoStub{}, newUpstreamManagementSettingRepoStub(), nil)

	_, err := svc.CompareGroups(context.Background())
	if err == nil {
		t.Fatalf("CompareGroups should require default provider")
	}
}
