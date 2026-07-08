package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type upstreamManagementProviderSourceStub struct {
	defaultProvider  UpstreamProviderConfig
	keys             []UpstreamProviderKey
	groups           []UpstreamProviderGroup
	modelSquare      json.RawMessage
	defaultErr       error
	keysErr          error
	groupsErr        error
	modelSquareErr   error
	fetchedSlug      string
	fetchedGroupSlug string
}

func (s *upstreamManagementProviderSourceStub) GetDefaultProvider(ctx context.Context) (UpstreamProviderConfig, error) {
	return s.defaultProvider, s.defaultErr
}

func (s *upstreamManagementProviderSourceStub) FetchProviderKeys(ctx context.Context, slug string) ([]UpstreamProviderKey, []string, error) {
	s.fetchedSlug = slug
	return s.keys, []string{"upstream warning"}, s.keysErr
}

func (s *upstreamManagementProviderSourceStub) FetchProviderGroups(ctx context.Context, slug string) ([]UpstreamProviderGroup, []string, error) {
	s.fetchedGroupSlug = slug
	if s.groupsErr != nil {
		return nil, nil, s.groupsErr
	}
	if s.groups == nil {
		return nil, nil, infraerrors.BadRequest("UPSTREAM_PROVIDER_GROUPS_UNSUPPORTED", "upstream provider groups are unsupported")
	}
	return s.groups, []string{"upstream group warning"}, s.groupsErr
}

func (s *upstreamManagementProviderSourceStub) FetchDefaultModelSquare(context.Context) (json.RawMessage, UpstreamProviderConfig, error) {
	return s.modelSquare, s.defaultProvider, s.modelSquareErr
}

type upstreamManagementGroupRepoStub struct {
	groups                 []Group
	updates                []Group
	creates                []Group
	nextID                 int64
	updateErr              error
	boundGroupsByAccountID map[int64][]*Group
}

func (s *upstreamManagementGroupRepoStub) Create(_ context.Context, group *Group) error {
	s.creates = append(s.creates, *group)
	if group.ID == 0 {
		if s.nextID == 0 {
			s.nextID = 100
		}
		group.ID = s.nextID
		s.nextID++
	}
	s.groups = append(s.groups, *group)
	return nil
}

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
	if s.updateErr != nil {
		return s.updateErr
	}
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

func (s *upstreamManagementGroupRepoStub) ListGroupsByAccountIDs(_ context.Context, accountIDs []int64) (map[int64][]*Group, error) {
	if s.boundGroupsByAccountID == nil {
		return nil, nil
	}
	out := make(map[int64][]*Group, len(accountIDs))
	for _, accountID := range accountIDs {
		groups := s.boundGroupsByAccountID[accountID]
		if len(groups) == 0 {
			continue
		}
		out[accountID] = append([]*Group(nil), groups...)
	}
	return out, nil
}

func (s *upstreamManagementGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	panic("unexpected")
}

func (s *upstreamManagementGroupRepoStub) ExistsByName(_ context.Context, name string) (bool, error) {
	for _, group := range s.groups {
		if group.Name == name {
			return true, nil
		}
	}
	return false, nil
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

func TestUpstreamManagementServiceCompareGroupsUsesAvailableProviderGroups(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
		groups: []UpstreamProviderGroup{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5, RawGroupID: "1"},
			{ProviderSlug: "default-upstream", GroupName: "No Key Group", RateMultiplier: 0.15, RawGroupID: "2"},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{ID: 1, Name: "VIP", RateMultiplier: 1.5, Status: StatusActive}}}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	result, err := svc.CompareGroups(context.Background())
	if err != nil {
		t.Fatalf("CompareGroups returned error: %v", err)
	}
	if providerSource.fetchedGroupSlug != "default-upstream" {
		t.Fatalf("fetched group slug = %q, want default-upstream", providerSource.fetchedGroupSlug)
	}
	if len(result.Items) != 2 {
		t.Fatalf("items = %+v, want upstream groups from available groups endpoint", result.Items)
	}
	byName := map[string]UpstreamGroupComparison{}
	for _, item := range result.Items {
		byName[item.UpstreamGroupName] = item
	}
	if byName["VIP"].UpstreamKeyCount != 1 {
		t.Fatalf("VIP item = %+v, want key count 1", byName["VIP"])
	}
	noKey := byName["No Key Group"]
	if noKey.UpstreamRate != 0.15 || noKey.UpstreamKeyCount != 0 || noKey.Matched {
		t.Fatalf("No Key Group item = %+v, want unmatched group with rate 0.15 and no keys", noKey)
	}
}

func TestUpstreamManagementServiceCompareGroupsDoesNotReviveDeletedProviderGroupsFromKeys(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
		groups: []UpstreamProviderGroup{},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{
		{ID: 9, Name: "Renamed Local", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive},
	}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	settingRepo.values[SettingKeyUpstreamGroupMappings] = `[{
		"provider_slug": "default-upstream",
		"upstream_group_name": "VIP",
		"upstream_group_key": "vip",
		"local_group_id": 9
	}]`
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.CompareGroups(context.Background())
	if err != nil {
		t.Fatalf("CompareGroups returned error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("items = %+v, want deleted upstream group hidden despite stale key group", result.Items)
	}
	var stored []UpstreamGroupMappingRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupMappings]), &stored); err != nil {
		t.Fatalf("stored mappings should be JSON: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("stored mappings = %+v, want stale mapping pruned", stored)
	}
}

func TestUpstreamManagementServiceFetchDefaultModelSquareUsesLocalGroupRates(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		modelSquare: json.RawMessage(`{
			"groups":[
				{"id":"remote-1","name":" VIP Group ","rate_multiplier":9.9,"description":"remote"},
				{"id":"remote-2","name":"unmatched","rate_multiplier":2.2}
			],
			"models":[{"id":"gpt-5.2","group_ids":["remote-1","remote-2"]}]
		}`),
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{
		ID:             7,
		Name:           "vip group",
		Description:    "local",
		Platform:       PlatformOpenAI,
		RateMultiplier: 0.25,
		Status:         StatusActive,
	}}}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	payload, provider, err := svc.FetchDefaultModelSquare(context.Background())
	if err != nil {
		t.Fatalf("FetchDefaultModelSquare returned error: %v", err)
	}
	if provider.Slug != "default-upstream" {
		t.Fatalf("provider slug = %q, want default-upstream", provider.Slug)
	}

	var body struct {
		Groups []map[string]any `json:"groups"`
		Models []struct {
			GroupIDs []string `json:"group_ids"`
		} `json:"models"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("payload should be JSON: %v", err)
	}
	if len(body.Groups) != 1 {
		t.Fatalf("group count = %d, want 1: %s", len(body.Groups), string(payload))
	}
	if body.Groups[0]["id"] != "remote-1" {
		t.Fatalf("group id = %v, want remote-1", body.Groups[0]["id"])
	}
	if body.Groups[0]["name"] != "vip group" {
		t.Fatalf("group name = %v, want local group name", body.Groups[0]["name"])
	}
	if body.Groups[0]["rate_multiplier"] != 0.25 {
		t.Fatalf("rate_multiplier = %v, want local 0.25", body.Groups[0]["rate_multiplier"])
	}
	if len(body.Models) != 1 || len(body.Models[0].GroupIDs) != 1 || body.Models[0].GroupIDs[0] != "remote-1" {
		t.Fatalf("model group ids not filtered to local matches: %+v", body.Models)
	}
}

func TestUpstreamManagementServiceFetchDefaultModelSquareMatchesGroupsIgnoringSeparators(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		modelSquare: json.RawMessage(`{
			"groups":[{"id":"remote-plus","name":"gpt plus","rate_multiplier":9.9}],
			"models":[{"id":"gpt-5.2","group_ids":["remote-plus"]}]
		}`),
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{
		ID:             8,
		Name:           "gptplus",
		Platform:       PlatformOpenAI,
		RateMultiplier: 0.35,
		Status:         StatusActive,
	}}}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	payload, _, err := svc.FetchDefaultModelSquare(context.Background())
	if err != nil {
		t.Fatalf("FetchDefaultModelSquare returned error: %v", err)
	}

	var body struct {
		Groups []map[string]any `json:"groups"`
		Models []struct {
			GroupIDs []string `json:"group_ids"`
		} `json:"models"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("payload should be JSON: %v", err)
	}
	if len(body.Groups) != 1 {
		t.Fatalf("group count = %d, want 1: %s", len(body.Groups), string(payload))
	}
	if body.Groups[0]["name"] != "gptplus" {
		t.Fatalf("group name = %v, want local gptplus", body.Groups[0]["name"])
	}
	if body.Groups[0]["rate_multiplier"] != 0.35 {
		t.Fatalf("rate_multiplier = %v, want local 0.35", body.Groups[0]["rate_multiplier"])
	}
	if len(body.Models) != 1 || len(body.Models[0].GroupIDs) != 1 || body.Models[0].GroupIDs[0] != "remote-plus" {
		t.Fatalf("model group ids not retained for matched local group: %+v", body.Models)
	}
}

func TestUpstreamManagementServiceFetchDefaultModelSquareSkipsIgnoredGroupMatch(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		modelSquare: json.RawMessage(`{
			"groups":[{"id":"remote-vip","name":"VIP","rate_multiplier":9.9}],
			"models":[{"id":"gpt-5.2","group_ids":["remote-vip"]}]
		}`),
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{
		ID:             8,
		Name:           "VIP",
		Platform:       PlatformOpenAI,
		RateMultiplier: 0.35,
		Status:         StatusActive,
	}}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	settingRepo.values[SettingKeyUpstreamGroupMappings] = `[{
		"provider_slug": "default-upstream",
		"upstream_group_name": "VIP",
		"upstream_group_key": "vip",
		"ignored": true
	}]`
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	payload, _, err := svc.FetchDefaultModelSquare(context.Background())
	if err != nil {
		t.Fatalf("FetchDefaultModelSquare returned error: %v", err)
	}

	var body struct {
		Groups []map[string]any `json:"groups"`
		Models []struct {
			GroupIDs []string `json:"group_ids"`
		} `json:"models"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("payload should be JSON: %v", err)
	}
	if len(body.Groups) != 0 {
		t.Fatalf("groups = %+v, want ignored name match removed", body.Groups)
	}
	if len(body.Models) != 1 || len(body.Models[0].GroupIDs) != 0 {
		t.Fatalf("model group ids = %+v, want ignored group filtered", body.Models)
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

func TestUpstreamManagementServiceCompareGroupsSkipsIgnoredNameMatch(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{ID: 7, Name: "VIP", RateMultiplier: 3, Status: StatusActive}}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	settingRepo.values[SettingKeyUpstreamGroupMappings] = `[{
		"provider_slug": "default-upstream",
		"upstream_group_name": "VIP",
		"upstream_group_key": "vip",
		"ignored": true
	}]`
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.CompareGroups(context.Background())
	if err != nil {
		t.Fatalf("CompareGroups returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.Matched || !item.MatchIgnored || item.LocalGroupID != nil {
		t.Fatalf("ignored name match item = %+v, want unmatched ignored row", item)
	}
}

func TestUpstreamManagementServiceCompareGroupsPrefersManualMapping(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{
		{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 3, Status: StatusActive},
		{ID: 9, Name: "Mapped VIP", Platform: PlatformGemini, RateMultiplier: 1, Status: StatusActive},
	}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	settingRepo.values[SettingKeyUpstreamGroupMappings] = `[{
		"provider_slug": "default-upstream",
		"upstream_group_name": "VIP",
		"upstream_group_key": "vip",
		"local_group_id": 9
	}]`
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.CompareGroups(context.Background())
	if err != nil {
		t.Fatalf("CompareGroups returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if !item.Matched || item.LocalGroupID == nil || *item.LocalGroupID != 9 {
		t.Fatalf("expected manual mapped local group 9, got %+v", item)
	}
	if item.MatchSource != "manual" {
		t.Fatalf("match source = %q, want manual", item.MatchSource)
	}
	if item.LocalGroupPlatform != PlatformGemini {
		t.Fatalf("local group platform = %q, want %q", item.LocalGroupPlatform, PlatformGemini)
	}
	if !item.NeedsRateIncrease {
		t.Fatalf("manual mapped group should need rate increase: %+v", item)
	}
}

func TestUpstreamManagementServiceSaveGroupMappingStoresMapping(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{
		{ID: 9, Name: "Mapped VIP", RateMultiplier: 1, Status: StatusActive},
	}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.SaveGroupMapping(context.Background(), UpstreamGroupMappingInput{
		UpstreamGroupName: " VIP ",
		LocalGroupID:      ptrInt64(9),
	})
	if err != nil {
		t.Fatalf("SaveGroupMapping returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].LocalGroupID == nil || *result.Items[0].LocalGroupID != 9 {
		t.Fatalf("expected mapped comparison result, got %+v", result.Items)
	}
	var stored []UpstreamGroupMappingRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupMappings]), &stored); err != nil {
		t.Fatalf("stored mappings should be JSON: %v", err)
	}
	if len(stored) != 1 || stored[0].ProviderSlug != "default-upstream" || stored[0].UpstreamGroupKey != "vip" || stored[0].LocalGroupID != 9 {
		t.Fatalf("unexpected stored mappings: %+v", stored)
	}
}

func TestUpstreamManagementServiceSaveGroupMappingClearsMapping(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{ID: 9, Name: "Mapped VIP", RateMultiplier: 1, Status: StatusActive}}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	settingRepo.values[SettingKeyUpstreamGroupMappings] = `[{
		"provider_slug": "default-upstream",
		"upstream_group_name": "VIP",
		"upstream_group_key": "vip",
		"local_group_id": 9
	}]`
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.SaveGroupMapping(context.Background(), UpstreamGroupMappingInput{
		UpstreamGroupName: "VIP",
		LocalGroupID:      nil,
	})
	if err != nil {
		t.Fatalf("SaveGroupMapping clear returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Matched {
		t.Fatalf("expected cleared mapping without name fallback, got %+v", result.Items)
	}
	var stored []UpstreamGroupMappingRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupMappings]), &stored); err != nil {
		t.Fatalf("stored mappings should be JSON: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("stored mappings = %+v, want empty", stored)
	}
}

func TestUpstreamManagementServiceSaveGroupMappingIgnoresAutomaticNameMatch(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{{ID: 9, Name: "VIP", RateMultiplier: 1, Status: StatusActive}}}
	settingRepo := newUpstreamManagementSettingRepoStub()
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.SaveGroupMapping(context.Background(), UpstreamGroupMappingInput{
		UpstreamGroupName: "VIP",
		Ignored:           true,
	})
	if err != nil {
		t.Fatalf("SaveGroupMapping ignore returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Matched || !result.Items[0].MatchIgnored {
		t.Fatalf("expected ignored automatic name match, got %+v", result.Items)
	}
	var stored []UpstreamGroupMappingRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupMappings]), &stored); err != nil {
		t.Fatalf("stored mappings should be JSON: %v", err)
	}
	if len(stored) != 1 || !stored[0].Ignored || stored[0].LocalGroupID != 0 || stored[0].UpstreamGroupKey != "vip" {
		t.Fatalf("stored mappings = %+v, want ignored vip mapping", stored)
	}
}

func TestUpstreamManagementServiceCreateLocalGroupCreatesAndMapsDefaultUpstreamGroup(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{}, nextID: 42}
	settingRepo := newUpstreamManagementSettingRepoStub()
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.CreateLocalGroupFromUpstream(context.Background(), UpstreamGroupLocalCreateInput{
		UpstreamGroupName: " VIP ",
		Platform:          PlatformAnthropic,
		RateMultiplier:    2.5,
	})
	if err != nil {
		t.Fatalf("CreateLocalGroupFromUpstream returned error: %v", err)
	}
	if len(groupRepo.creates) != 1 {
		t.Fatalf("created group count = %d, want 1", len(groupRepo.creates))
	}
	created := groupRepo.groups[len(groupRepo.groups)-1]
	if created.ID != 42 || created.Name != "VIP" || created.Platform != PlatformAnthropic || created.Status != StatusActive {
		t.Fatalf("created group = %+v, want active Anthropic VIP group", created)
	}
	if created.RateMultiplier != 2.5 || created.SubscriptionType != SubscriptionTypeStandard {
		t.Fatalf("created group pricing = %+v, want rate 2.5 standard", created)
	}
	var stored []UpstreamGroupMappingRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupMappings]), &stored); err != nil {
		t.Fatalf("stored mappings should be JSON: %v", err)
	}
	if len(stored) != 1 || stored[0].ProviderSlug != "default-upstream" || stored[0].UpstreamGroupKey != "vip" || stored[0].LocalGroupID != 42 {
		t.Fatalf("unexpected stored mappings: %+v", stored)
	}
	if len(result.Items) != 1 || !result.Items[0].Matched || result.Items[0].LocalGroupID == nil || *result.Items[0].LocalGroupID != 42 {
		t.Fatalf("expected comparison result mapped to new group, got %+v", result.Items)
	}
}

func TestUpstreamManagementServiceCreateLocalGroupUsesAvailableProviderGroups(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		groups: []UpstreamProviderGroup{
			{ProviderSlug: "default-upstream", GroupName: "No Key Group", RateMultiplier: 0.15, RawGroupID: "2"},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{}, nextID: 42}
	settingRepo := newUpstreamManagementSettingRepoStub()
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)

	result, err := svc.CreateLocalGroupFromUpstream(context.Background(), UpstreamGroupLocalCreateInput{
		UpstreamGroupName: " no-key group ",
		Platform:          PlatformOpenAI,
		RateMultiplier:    0.15,
	})
	if err != nil {
		t.Fatalf("CreateLocalGroupFromUpstream returned error: %v", err)
	}
	if len(groupRepo.creates) != 1 {
		t.Fatalf("created group count = %d, want 1", len(groupRepo.creates))
	}
	created := groupRepo.groups[len(groupRepo.groups)-1]
	if created.ID != 42 || created.Name != "No Key Group" {
		t.Fatalf("created group = %+v, want canonical upstream group name", created)
	}
	var stored []UpstreamGroupMappingRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupMappings]), &stored); err != nil {
		t.Fatalf("stored mappings should be JSON: %v", err)
	}
	if len(stored) != 1 || stored[0].UpstreamGroupKey != "nokeygroup" || stored[0].LocalGroupID != 42 {
		t.Fatalf("unexpected stored mappings: %+v", stored)
	}
	if len(result.Items) != 1 || !result.Items[0].Matched || result.Items[0].UpstreamKeyCount != 0 {
		t.Fatalf("expected no-key upstream group mapped to new local group, got %+v", result.Items)
	}
}

func TestUpstreamManagementServiceCreateLocalGroupRejectsUnknownUpstreamGroup(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{}, nextID: 42}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	_, err := svc.CreateLocalGroupFromUpstream(context.Background(), UpstreamGroupLocalCreateInput{
		UpstreamGroupName: "Missing",
		Platform:          PlatformOpenAI,
		RateMultiplier:    1,
	})
	if err == nil || !infraerrors.IsBadRequest(err) {
		t.Fatalf("CreateLocalGroupFromUpstream error = %v, want bad request", err)
	}
	if len(groupRepo.creates) != 0 {
		t.Fatalf("created group count = %d, want 0", len(groupRepo.creates))
	}
}

func TestUpstreamManagementServiceCreateLocalGroupRequiresPlatform(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{groups: []Group{}, nextID: 42}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	_, err := svc.CreateLocalGroupFromUpstream(context.Background(), UpstreamGroupLocalCreateInput{
		UpstreamGroupName: "VIP",
		RateMultiplier:    2.5,
	})
	if err == nil || !infraerrors.IsBadRequest(err) {
		t.Fatalf("CreateLocalGroupFromUpstream error = %v, want bad request", err)
	}
	if len(groupRepo.creates) != 0 {
		t.Fatalf("created group count = %d, want 0", len(groupRepo.creates))
	}
}

func TestUpstreamManagementServiceCreateLocalGroupRejectsNormalizedDuplicateLocalGroup(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{
		groups: []Group{{ID: 7, Name: " vip ", RateMultiplier: 1, Status: StatusActive}},
		nextID: 42,
	}
	svc := NewUpstreamManagementService(providerSource, groupRepo, newUpstreamManagementSettingRepoStub(), nil)

	_, err := svc.CreateLocalGroupFromUpstream(context.Background(), UpstreamGroupLocalCreateInput{
		UpstreamGroupName: "VIP",
		Platform:          PlatformOpenAI,
		RateMultiplier:    2.5,
	})
	if err == nil || !infraerrors.IsConflict(err) {
		t.Fatalf("CreateLocalGroupFromUpstream error = %v, want conflict", err)
	}
	if len(groupRepo.creates) != 0 {
		t.Fatalf("created group count = %d, want 0", len(groupRepo.creates))
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

func TestUpstreamManagementServiceMarkRateFixRecordHandledPersistsRecord(t *testing.T) {
	settingRepo := newUpstreamManagementSettingRepoStub()
	changedAt := mustParseTime(t, "2026-06-20T00:00:00Z")
	rawRecords, err := json.Marshal([]UpstreamGroupRateFixRecord{{
		GroupID:           7,
		GroupName:         "VIP",
		ProviderSlug:      "default-upstream",
		ProviderName:      "Default upstream",
		UpstreamGroupName: "VIP",
		OldRate:           1,
		NewRate:           2.5,
		ChangedAt:         changedAt,
	}})
	if err != nil {
		t.Fatalf("marshal records: %v", err)
	}
	settingRepo.values[SettingKeyUpstreamGroupRateFixRecords] = string(rawRecords)
	svc := NewUpstreamManagementService(
		&upstreamManagementProviderSourceStub{},
		&upstreamManagementGroupRepoStub{},
		settingRepo,
		&upstreamManagementAuthCacheInvalidatorStub{},
	)

	records, err := svc.MarkRateFixRecordHandled(context.Background(), "2026-06-20T00:00:00Z-7-default-upstream-VIP")
	if err != nil {
		t.Fatalf("MarkRateFixRecordHandled returned error: %v", err)
	}
	if len(records) != 1 || !records[0].Handled {
		t.Fatalf("records = %+v, want handled record", records)
	}
	var persisted []UpstreamGroupRateFixRecord
	if err := json.Unmarshal([]byte(settingRepo.values[SettingKeyUpstreamGroupRateFixRecords]), &persisted); err != nil {
		t.Fatalf("decode persisted records: %v", err)
	}
	if len(persisted) != 1 || !persisted[0].Handled {
		t.Fatalf("persisted records = %+v, want handled record", persisted)
	}
}

func TestUpstreamManagementServiceRateFixConfigDefaultsDisabledWithSecondsInterval(t *testing.T) {
	svc := NewUpstreamManagementService(
		&upstreamManagementProviderSourceStub{},
		&upstreamManagementGroupRepoStub{},
		newUpstreamManagementSettingRepoStub(),
		nil,
	)

	cfg, err := svc.GetRateFixConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRateFixConfig returned error: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("default config should be disabled: %+v", cfg)
	}
	if cfg.IntervalSeconds != DefaultUpstreamGroupRateFixIntervalSeconds {
		t.Fatalf("default interval = %d, want %d", cfg.IntervalSeconds, DefaultUpstreamGroupRateFixIntervalSeconds)
	}
}

func TestUpstreamManagementServiceUpdateRateFixConfigStoresSecondsInterval(t *testing.T) {
	settingRepo := newUpstreamManagementSettingRepoStub()
	svc := NewUpstreamManagementService(
		&upstreamManagementProviderSourceStub{},
		&upstreamManagementGroupRepoStub{},
		settingRepo,
		nil,
	)

	cfg, err := svc.UpdateRateFixConfig(context.Background(), UpstreamGroupAutoRateFixConfig{
		Enabled:         true,
		IntervalSeconds: 45,
	})
	if err != nil {
		t.Fatalf("UpdateRateFixConfig returned error: %v", err)
	}
	if !cfg.Enabled || cfg.IntervalSeconds != 45 || cfg.UpdatedAt == nil {
		t.Fatalf("stored config = %+v, want enabled interval 45 with updated_at", cfg)
	}

	loaded, err := svc.GetRateFixConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRateFixConfig returned error: %v", err)
	}
	if !loaded.Enabled || loaded.IntervalSeconds != 45 {
		t.Fatalf("loaded config = %+v, want enabled interval 45", loaded)
	}
}

func TestUpstreamManagementServiceRunScheduledRateFixStoresFailureStatus(t *testing.T) {
	providerSource := &upstreamManagementProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "default-upstream", Name: "Default upstream", IsDefault: true},
		keys: []UpstreamProviderKey{
			{ProviderSlug: "default-upstream", GroupName: "VIP", RateMultiplier: 2.5},
		},
	}
	settingRepo := newUpstreamManagementSettingRepoStub()
	groupRepo := &upstreamManagementGroupRepoStub{
		groups:    []Group{{ID: 1, Name: "VIP", RateMultiplier: 1, Status: StatusActive}},
		updateErr: errors.New("database unavailable"),
	}
	svc := NewUpstreamManagementService(providerSource, groupRepo, settingRepo, nil)
	_, err := svc.UpdateRateFixConfig(context.Background(), UpstreamGroupAutoRateFixConfig{
		Enabled:         true,
		IntervalSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("UpdateRateFixConfig returned error: %v", err)
	}

	cfg, err := svc.RunScheduledRateFix(context.Background())
	if err == nil {
		t.Fatalf("RunScheduledRateFix should return the rate-fix error")
	}
	if cfg.LastRunAt == nil || cfg.LastRunStatus != "failed" || cfg.LastRunMessage == "" {
		t.Fatalf("config after failed run = %+v, want failed status with message", cfg)
	}

	loaded, err := svc.GetRateFixConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRateFixConfig returned error: %v", err)
	}
	if loaded.LastRunAt == nil || loaded.LastRunStatus != "failed" || loaded.LastRunMessage == "" {
		t.Fatalf("persisted config = %+v, want failed status with message", loaded)
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
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
