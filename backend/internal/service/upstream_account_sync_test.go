package service

import (
	"context"
	"encoding/json"
	"testing"
)

type upstreamAccountSyncProviderSourceStub struct {
	defaultProvider UpstreamProviderConfig
	providers       []UpstreamProviderConfig
	storedProviders map[string]UpstreamProviderConfig
	keys            []UpstreamProviderKey
	keysBySlug      map[string][]UpstreamProviderKey
	defaultErr      error
	providersErr    error
	keysErr         error
	fetchedSlug     string
	fetchedSlugs    []string
}

func (s *upstreamAccountSyncProviderSourceStub) GetDefaultProvider(ctx context.Context) (UpstreamProviderConfig, error) {
	return s.defaultProvider, s.defaultErr
}

func (s *upstreamAccountSyncProviderSourceStub) ListProviders(ctx context.Context) ([]UpstreamProviderConfig, error) {
	if s.providersErr != nil {
		return nil, s.providersErr
	}
	if s.providers != nil {
		return s.providers, nil
	}
	return []UpstreamProviderConfig{s.defaultProvider}, nil
}

func (s *upstreamAccountSyncProviderSourceStub) getStoredProvider(_ context.Context, slug string) (UpstreamProviderConfig, error) {
	if s.storedProviders != nil {
		if provider, ok := s.storedProviders[slug]; ok {
			return provider, nil
		}
	}
	for _, provider := range s.providers {
		if provider.Slug == slug {
			return provider, nil
		}
	}
	if s.defaultProvider.Slug == slug {
		return s.defaultProvider, nil
	}
	return UpstreamProviderConfig{}, ErrSettingNotFound
}

func (s *upstreamAccountSyncProviderSourceStub) FetchProviderKeys(ctx context.Context, slug string) ([]UpstreamProviderKey, []string, error) {
	s.fetchedSlug = slug
	s.fetchedSlugs = append(s.fetchedSlugs, slug)
	if s.keysBySlug != nil {
		return s.keysBySlug[slug], []string{"provider warning"}, s.keysErr
	}
	return s.keys, []string{"provider warning"}, s.keysErr
}

type upstreamAccountSyncAccountManagerStub struct {
	accounts      []Account
	createdInputs []CreateAccountInput
	updateInputs  []upstreamAccountSyncUpdateCall
}

type upstreamAccountSyncUpdateCall struct {
	id    int64
	input UpdateAccountInput
}

func (s *upstreamAccountSyncAccountManagerStub) ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]Account, int64, error) {
	out := make([]Account, len(s.accounts))
	copy(out, s.accounts)
	return out, int64(len(out)), nil
}

func (s *upstreamAccountSyncAccountManagerStub) CreateAccount(ctx context.Context, input *CreateAccountInput) (*Account, error) {
	s.createdInputs = append(s.createdInputs, *input)
	account := &Account{
		ID:          int64(1000 + len(s.createdInputs)),
		Name:        input.Name,
		Platform:    input.Platform,
		Type:        input.Type,
		Credentials: input.Credentials,
		Extra:       input.Extra,
		GroupIDs:    append([]int64(nil), input.GroupIDs...),
	}
	s.accounts = append(s.accounts, *account)
	return account, nil
}

func (s *upstreamAccountSyncAccountManagerStub) UpdateAccount(ctx context.Context, id int64, input *UpdateAccountInput) (*Account, error) {
	s.updateInputs = append(s.updateInputs, upstreamAccountSyncUpdateCall{id: id, input: *input})
	for i := range s.accounts {
		if s.accounts[i].ID != id {
			continue
		}
		if len(input.Credentials) > 0 {
			s.accounts[i].Credentials = input.Credentials
		}
		if input.Extra != nil {
			s.accounts[i].Extra = input.Extra
		}
		if input.GroupIDs != nil {
			s.accounts[i].GroupIDs = append([]int64(nil), (*input.GroupIDs)...)
		}
		return &s.accounts[i], nil
	}
	return nil, ErrAccountNotFound
}

func newUpstreamAccountSyncServiceForTest(
	provider *upstreamAccountSyncProviderSourceStub,
	groups []Group,
	accounts []Account,
	settings *upstreamManagementSettingRepoStub,
) (*UpstreamAccountSyncService, *upstreamAccountSyncAccountManagerStub) {
	if settings == nil {
		settings = newUpstreamManagementSettingRepoStub()
	}
	accountManager := &upstreamAccountSyncAccountManagerStub{accounts: accounts}
	svc := NewUpstreamAccountSyncService(
		provider,
		&upstreamManagementGroupRepoStub{groups: groups},
		accountManager,
		settings,
	)
	return svc, accountManager
}

func TestUpstreamAccountSyncPreviewUsesNonDefaultProvidersAndManualGroupMapping(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{
			Slug:              "main",
			Name:              "Main upstream",
			Type:              UpstreamProviderTypeSub2API,
			BaseURL:           "https://upstream.example.com",
			AccountNamePrefix: "up-",
			IsDefault:         true,
		},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", Type: UpstreamProviderTypeSub2API, BaseURL: "https://main.example.com", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup upstream", Type: UpstreamProviderTypeSub2API, BaseURL: "https://backup.example.com", AccountNamePrefix: "backup-", Enabled: true},
			{Slug: "disabled", Name: "Disabled upstream", Type: UpstreamProviderTypeSub2API, BaseURL: "https://disabled.example.com", Enabled: false},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 2.5},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	settings.values[SettingKeyUpstreamGroupMappings] = `[{
		"provider_slug":"backup",
		"upstream_group_name":"VIP",
		"upstream_group_key":"vip",
		"local_group_id":9
	}]`
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 3, Status: StatusActive},
			{ID: 9, Name: "Mapped VIP", Platform: PlatformOpenAI, RateMultiplier: 3, Status: StatusActive},
		},
		nil,
		settings,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(provider.fetchedSlugs) != 1 || provider.fetchedSlugs[0] != "backup" {
		t.Fatalf("fetched slugs = %+v, want [backup]", provider.fetchedSlugs)
	}
	if result.Summary.UpstreamKeyCount != 1 || result.Summary.CreateCount != 1 {
		t.Fatalf("summary = %+v, want one upstream key and one create", result.Summary)
	}
	if len(result.Providers) != 1 || result.Providers[0].Slug != "backup" {
		t.Fatalf("providers = %+v, want only backup", result.Providers)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.Action != UpstreamAccountSyncActionCreate {
		t.Fatalf("action = %q, want create", item.Action)
	}
	if item.UpstreamKeyName != "alice" {
		t.Fatalf("upstream key name = %q, want alice", item.UpstreamKeyName)
	}
	if item.LocalAccountName != "backup-alice" {
		t.Fatalf("local account name = %q, want backup-alice", item.LocalAccountName)
	}
	if item.LocalGroupID == nil || *item.LocalGroupID != 9 || item.LocalGroupName != "Mapped VIP" {
		t.Fatalf("local group match = id %v name %q, want 9 Mapped VIP", item.LocalGroupID, item.LocalGroupName)
	}
}

func TestUpstreamAccountSyncPreviewDetectsDuplicateLocalAccountNames(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", AccountNamePrefix: "main-", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		[]Account{
			{ID: 1, Name: "up-alice", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Groups: []*Group{{ID: 7, Name: "VIP", RateMultiplier: 1}}},
			{ID: 2, Name: " UP-ALICE ", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Groups: []*Group{{ID: 8, Name: "Trial", RateMultiplier: 0.5}}},
		},
		nil,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if result.Summary.ConflictCount != 1 || result.Items[0].Action != UpstreamAccountSyncActionConflict {
		t.Fatalf("result = %+v, want one conflict", result)
	}
	if len(result.Items[0].ConflictAccountIDs) != 2 {
		t.Fatalf("conflict ids = %+v, want 2 ids", result.Items[0].ConflictAccountIDs)
	}
	conflicts := result.Items[0].ConflictAccounts
	if len(conflicts) != 2 {
		t.Fatalf("conflict accounts = %+v, want 2 accounts", conflicts)
	}
	if conflicts[0].ID != 1 || conflicts[0].Name != "up-alice" || len(conflicts[0].BoundGroups) != 1 || conflicts[0].BoundGroups[0].Name != "VIP" {
		t.Fatalf("first conflict account = %+v, want up-alice with VIP", conflicts[0])
	}
	if conflicts[1].ID != 2 || conflicts[1].Name != " UP-ALICE " || len(conflicts[1].BoundGroups) != 1 || conflicts[1].BoundGroups[0].RateMultiplier != 0.5 {
		t.Fatalf("second conflict account = %+v, want duplicate account with 0.5x group", conflicts[1])
	}
}

func TestUpstreamAccountSyncPreviewMatchesPrefixedKeyNameIgnoringUnicodeSpacesAndCase(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", BaseURL: "https://backup.example.com", AccountNamePrefix: "FindCG ", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "Alice Key", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		[]Account{{
			ID:          10,
			Name:        " findcg\u3000ALICE key ",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "Alice Key", "base_url": "https://backup.example.com"},
			GroupIDs:    []int64{7},
		}},
		nil,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.MatchedAccountID == nil || *item.MatchedAccountID != 10 {
		t.Fatalf("matched account id = %v, want 10", item.MatchedAccountID)
	}
	if item.Action != UpstreamAccountSyncActionNoop {
		t.Fatalf("action = %q, want noop", item.Action)
	}
}

func TestUpstreamAccountSyncPreviewIncludesMatchedAccountBoundGroups(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 2}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive}},
		[]Account{{
			ID:       10,
			Name:     "up-alice",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			GroupIDs: []int64{7, 8},
			Groups: []*Group{
				{ID: 7, Name: "VIP", RateMultiplier: 2},
				{ID: 8, Name: "Trial", RateMultiplier: 0.5},
			},
		}},
		nil,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	groups := result.Items[0].BoundGroups
	if len(groups) != 2 {
		t.Fatalf("bound groups = %+v, want VIP and Trial", groups)
	}
	if groups[0].ID != 7 || groups[0].Name != "VIP" || groups[0].RateMultiplier != 2 || groups[0].RateViolation {
		t.Fatalf("first bound group = %+v, want non-risk VIP", groups[0])
	}
	if groups[1].ID != 8 || groups[1].Name != "Trial" || groups[1].RateMultiplier != 0.5 || !groups[1].RateViolation {
		t.Fatalf("second bound group = %+v, want low-rate Trial", groups[1])
	}
}

func TestUpstreamAccountSyncPreviewHydratesBoundGroupsFromGroupIDs(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 2}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
		},
		[]Account{{
			ID:       10,
			Name:     "up-alice",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			GroupIDs: []int64{7, 8},
		}},
		nil,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if !item.RateViolation || item.Action != UpstreamAccountSyncActionUpdate {
		t.Fatalf("item = %+v, want update with rate violation", item)
	}
	if len(item.BoundGroups) != 2 {
		t.Fatalf("bound groups = %+v, want hydrated VIP and Trial", item.BoundGroups)
	}
	if len(item.UnboundGroupIDs) != 1 || item.UnboundGroupIDs[0] != 8 {
		t.Fatalf("unbound group ids = %+v, want [8]", item.UnboundGroupIDs)
	}
}

func TestUpstreamAccountSyncPreviewLoadsBoundGroupsByMatchedAccountID(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 2}},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{
		groups: []Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
		},
		boundGroupsByAccountID: map[int64][]*Group{
			10: {
				{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
				{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
			},
		},
	}
	accountManager := &upstreamAccountSyncAccountManagerStub{
		accounts: []Account{{
			ID:       10,
			Name:     "up-alice",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
		}},
	}
	svc := NewUpstreamAccountSyncService(provider, groupRepo, accountManager, newUpstreamManagementSettingRepoStub())

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.MatchedAccountID == nil || *item.MatchedAccountID != 10 {
		t.Fatalf("matched account id = %+v, want 10", item.MatchedAccountID)
	}
	if len(item.BoundGroups) != 2 {
		t.Fatalf("bound groups = %+v, want groups loaded by matched account id", item.BoundGroups)
	}
	if item.BoundGroups[0].ID != 7 || item.BoundGroups[1].ID != 8 {
		t.Fatalf("bound groups = %+v, want [7, 8]", item.BoundGroups)
	}
	if !item.RateViolation || len(item.UnboundGroupIDs) != 1 || item.UnboundGroupIDs[0] != 8 {
		t.Fatalf("rate guard fields = violation:%v unbound:%+v, want low-rate group 8", item.RateViolation, item.UnboundGroupIDs)
	}
}

func TestUpstreamAccountSyncPreviewShowsMatchedAccountGroupsWhenUpstreamGroupUnmatched(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: " Up Stream - ", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: " Alice Key ", GroupName: "Remote Only", RateMultiplier: 2}},
		},
	}
	groupRepo := &upstreamManagementGroupRepoStub{
		groups: []Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
		},
		boundGroupsByAccountID: map[int64][]*Group{
			10: {
				{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
			},
		},
	}
	accountManager := &upstreamAccountSyncAccountManagerStub{
		accounts: []Account{{
			ID:       10,
			Name:     "upstreamalicekey",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
		}},
	}
	svc := NewUpstreamAccountSyncService(provider, groupRepo, accountManager, newUpstreamManagementSettingRepoStub())

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.MatchedAccountID == nil || *item.MatchedAccountID != 10 {
		t.Fatalf("matched account id = %+v, want 10", item.MatchedAccountID)
	}
	if item.MatchedAccountName != "upstreamalicekey" {
		t.Fatalf("matched account name = %q, want upstreamalicekey", item.MatchedAccountName)
	}
	if len(item.BoundGroups) != 1 || item.BoundGroups[0].ID != 7 || item.BoundGroups[0].Name != "VIP" {
		t.Fatalf("bound groups = %+v, want VIP from matched account", item.BoundGroups)
	}
	if item.Action != UpstreamAccountSyncActionSkip || item.SkipReason != "upstream group is not matched" {
		t.Fatalf("action/skip = %q/%q, want skip because upstream group is not matched", item.Action, item.SkipReason)
	}
	if result.Summary.MatchedAccountCount != 1 || result.Summary.SkipCount != 1 {
		t.Fatalf("summary = %+v, want one matched account and one skip", result.Summary)
	}
}

func TestUpstreamAccountSyncPreviewUsesStoredProviderAccountNamePrefix(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "beikun", Name: "北鲲", Enabled: true},
		},
		storedProviders: map[string]UpstreamProviderConfig{
			"beikun": {Slug: "beikun", Name: "北鲲", AccountNamePrefix: "北鲲-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"beikun": {{ProviderSlug: "beikun", KeyName: "codex_pro2", GroupName: "codex_pro2", RateMultiplier: 0.25}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "codex_pro2", Platform: PlatformOpenAI, RateMultiplier: 0.25, Status: StatusActive}},
		[]Account{{
			ID:       10,
			Name:     "北鲲-Codex_pro2",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			GroupIDs: []int64{7},
		}},
		nil,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("providers = %+v, want one sync provider", result.Providers)
	}
	if result.Providers[0].AccountNamePrefix != "北鲲-" {
		t.Fatalf("provider prefix = %q, want stored prefix", result.Providers[0].AccountNamePrefix)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.LocalAccountName != "北鲲-codex_pro2" {
		t.Fatalf("local account name = %q, want prefix applied", item.LocalAccountName)
	}
	if item.MatchedAccountID == nil || *item.MatchedAccountID != 10 {
		t.Fatalf("matched account id = %+v, want 10", item.MatchedAccountID)
	}
}

func TestUpstreamAccountSyncRunCreatesUpdatesAndAppliesRateGuard(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{
			Slug:              "main",
			Name:              "Main",
			Type:              UpstreamProviderTypeSub2API,
			BaseURL:           "https://upstream.example.com",
			AccountNamePrefix: "up-",
		},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", Type: UpstreamProviderTypeSub2API, BaseURL: "https://main.example.com", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", Type: UpstreamProviderTypeSub2API, BaseURL: "https://backup.example.com", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "new", GroupName: "VIP", RateMultiplier: 2},
				{ProviderSlug: "backup", KeyName: "old", GroupName: "VIP", RateMultiplier: 2},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
		},
		[]Account{{
			ID:          10,
			Name:        "up-old",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "old-key", "model_mapping": map[string]any{"gpt": "gpt"}},
			Extra:       map[string]any{"quota_used": 12.0},
			GroupIDs:    []int64{7, 8},
			Groups: []*Group{
				{ID: 7, Name: "VIP", RateMultiplier: 2},
				{ID: 8, Name: "Trial", RateMultiplier: 0.5},
			},
		}},
		settings,
	)

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{
		CreateMissing:  true,
		UpdateExisting: true,
		ApplyRateGuard: true,
	})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if result.Summary.CreateCount != 1 || result.Summary.UpdateCount != 1 || result.Summary.RateViolationCount != 1 || result.Summary.UnboundGroupCount != 1 {
		t.Fatalf("summary = %+v, want create/update/rate guard counts", result.Summary)
	}
	if len(accounts.createdInputs) != 1 {
		t.Fatalf("created count = %d, want 1", len(accounts.createdInputs))
	}
	created := accounts.createdInputs[0]
	if created.Name != "up-new" || created.Platform != PlatformOpenAI || created.Type != AccountTypeAPIKey {
		t.Fatalf("created input = %+v, want OpenAI API key named up-new", created)
	}
	if created.Credentials["api_key"] != "new" || created.Credentials["base_url"] != "https://backup.example.com" {
		t.Fatalf("created credentials = %+v, want upstream key and base_url", created.Credentials)
	}
	if len(created.GroupIDs) != 1 || created.GroupIDs[0] != 7 {
		t.Fatalf("created group ids = %+v, want [7]", created.GroupIDs)
	}
	if len(accounts.updateInputs) != 1 {
		t.Fatalf("update count = %d, want 1", len(accounts.updateInputs))
	}
	update := accounts.updateInputs[0]
	if update.id != 10 {
		t.Fatalf("updated account id = %d, want 10", update.id)
	}
	if update.input.Credentials["api_key"] != "old" || update.input.Credentials["base_url"] != "https://backup.example.com" {
		t.Fatalf("updated credentials = %+v, want refreshed upstream key and base_url", update.input.Credentials)
	}
	if _, ok := update.input.Credentials["model_mapping"]; !ok {
		t.Fatalf("updated credentials lost model_mapping: %+v", update.input.Credentials)
	}
	if update.input.GroupIDs == nil || len(*update.input.GroupIDs) != 1 || (*update.input.GroupIDs)[0] != 7 {
		t.Fatalf("updated group ids = %+v, want [7]", update.input.GroupIDs)
	}

	rawRecords := settings.values[SettingKeyUpstreamAccountSyncRecords]
	var records []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(rawRecords), &records); err != nil {
		t.Fatalf("decode records: %v raw=%s", err, rawRecords)
	}
	if len(records) != 1 || records[0].CreatedCount != 1 || records[0].UpdatedCount != 1 || records[0].UnboundGroupCount != 1 {
		t.Fatalf("records = %+v, want one sync record with counts", records)
	}
	if records[0].TriggerSource != UpstreamAccountSyncTriggerManualSync {
		t.Fatalf("record trigger source = %q, want %q", records[0].TriggerSource, UpstreamAccountSyncTriggerManualSync)
	}
	if len(records[0].UnbindDetails) != 1 {
		t.Fatalf("unbind details = %+v, want one entry", records[0].UnbindDetails)
	}
	detail := records[0].UnbindDetails[0]
	if detail.TriggerSource != UpstreamAccountSyncTriggerManualSync {
		t.Fatalf("unbind detail trigger source = %q, want %q", detail.TriggerSource, UpstreamAccountSyncTriggerManualSync)
	}
	if detail.MatchedLocalAccountID != 10 || detail.MatchedLocalAccountName != "up-old" {
		t.Fatalf("unbind detail account = %+v, want account 10 up-old", detail)
	}
	if detail.UpstreamKeyName != "old" || detail.UpstreamGroupName != "VIP" || detail.UpstreamRateMultiplier != 2 {
		t.Fatalf("unbind detail upstream = %+v, want old/VIP/2", detail)
	}
	if len(detail.UnboundGroupIDs) != 1 || detail.UnboundGroupIDs[0] != 8 || len(detail.UnboundGroupNames) != 1 || detail.UnboundGroupNames[0] != "Trial" {
		t.Fatalf("unbind detail groups = %+v/%+v, want Trial group 8", detail.UnboundGroupIDs, detail.UnboundGroupNames)
	}
	if len(detail.RemainingGroupIDs) != 1 || detail.RemainingGroupIDs[0] != 7 {
		t.Fatalf("unbind detail remaining groups = %+v, want [7]", detail.RemainingGroupIDs)
	}
}

func TestUpstreamAccountSyncRunRecordsEachProviderSeparately(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{
			Slug:    "main",
			Name:    "Main",
			Type:    UpstreamProviderTypeSub2API,
			BaseURL: "https://main.example.com",
		},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", Type: UpstreamProviderTypeSub2API, BaseURL: "https://main.example.com", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", Type: UpstreamProviderTypeSub2API, BaseURL: "https://backup.example.com", AccountNamePrefix: "backup-", Enabled: true},
			{Slug: "mirror", Name: "Mirror", Type: UpstreamProviderTypeSub2API, BaseURL: "https://mirror.example.com", AccountNamePrefix: "mirror-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 2},
			},
			"mirror": {
				{ProviderSlug: "mirror", KeyName: "bob", GroupName: "VIP", RateMultiplier: 2},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive}},
		nil,
		settings,
	)

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{
		CreateMissing: true,
		TriggerSource: UpstreamAccountSyncTriggerManualSync,
	})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if result.Summary.CreateCount != 2 {
		t.Fatalf("create count = %d, want 2", result.Summary.CreateCount)
	}

	rawRecords := settings.values[SettingKeyUpstreamAccountSyncRecords]
	var records []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(rawRecords), &records); err != nil {
		t.Fatalf("decode records: %v raw=%s", err, rawRecords)
	}
	if len(records) != 2 {
		t.Fatalf("records = %+v, want one record per provider", records)
	}
	recordsByProvider := map[string]UpstreamAccountSyncRecord{}
	for _, record := range records {
		recordsByProvider[record.ProviderSlug] = record
	}
	if _, exists := recordsByProvider["multiple"]; exists {
		t.Fatalf("records = %+v, should not use synthetic multiple provider", records)
	}
	for slug, name := range map[string]string{"backup": "Backup", "mirror": "Mirror"} {
		record, exists := recordsByProvider[slug]
		if !exists {
			t.Fatalf("records = %+v, missing provider %s", records, slug)
		}
		if record.ProviderName != name || record.CreatedCount != 1 || record.TriggerSource != UpstreamAccountSyncTriggerManualSync {
			t.Fatalf("record[%s] = %+v, want provider name %s and one create", slug, record, name)
		}
	}
}

func TestUpstreamAccountSyncRunDoesNotUpdateNoopAccount(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{
			Slug:    "main",
			Name:    "Main",
			Type:    UpstreamProviderTypeSub2API,
			BaseURL: "https://upstream.example.com",
		},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", Type: UpstreamProviderTypeSub2API, BaseURL: "https://main.example.com", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", Type: UpstreamProviderTypeSub2API, BaseURL: "https://upstream.example.com", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "stable", GroupName: "VIP", RateMultiplier: 2},
			},
		},
	}
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive}},
		[]Account{{
			ID:          10,
			Name:        "up-stable",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "stable", "base_url": "https://upstream.example.com"},
			GroupIDs:    []int64{7},
			Groups:      []*Group{{ID: 7, Name: "VIP", RateMultiplier: 2}},
		}},
		nil,
	)

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{
		CreateMissing:  true,
		UpdateExisting: true,
		ApplyRateGuard: true,
	})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if result.Summary.UpdateCount != 0 {
		t.Fatalf("update count = %d, want 0", result.Summary.UpdateCount)
	}
	if len(accounts.updateInputs) != 0 {
		t.Fatalf("update calls = %d, want 0", len(accounts.updateInputs))
	}
	if len(result.Items) != 1 || result.Items[0].Action != UpstreamAccountSyncActionNoop {
		t.Fatalf("items = %+v, want noop item", result.Items)
	}
}

func TestUpstreamAccountRateGuardConfigDefaultsDisabled(t *testing.T) {
	settings := newUpstreamManagementSettingRepoStub()
	svc, _ := newUpstreamAccountSyncServiceForTest(
		&upstreamAccountSyncProviderSourceStub{},
		nil,
		nil,
		settings,
	)

	cfg, err := svc.GetRateGuardConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRateGuardConfig returned error: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("default config should be disabled")
	}
	if cfg.IntervalSeconds != DefaultUpstreamAccountRateGuardIntervalSeconds {
		t.Fatalf("default interval = %d, want %d", cfg.IntervalSeconds, DefaultUpstreamAccountRateGuardIntervalSeconds)
	}
}

func TestUpstreamAccountRateGuardRunOnlyAppliesRateGuard(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", BaseURL: "https://backup.example.com", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "VIP", RateMultiplier: 2},
				{ProviderSlug: "backup", KeyName: "missing", GroupName: "VIP", RateMultiplier: 2},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
		},
		[]Account{{
			ID:          10,
			Name:        "up-matched",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "matched", "base_url": "https://backup.example.com"},
			GroupIDs:    []int64{7, 8},
			Groups: []*Group{
				{ID: 7, Name: "VIP", RateMultiplier: 2},
				{ID: 8, Name: "Trial", RateMultiplier: 0.5},
			},
		}},
		settings,
	)

	cfg, err := svc.UpdateRateGuardConfig(context.Background(), UpstreamAccountRateGuardConfig{
		Enabled:         true,
		IntervalSeconds: 5,
	})
	if err != nil {
		t.Fatalf("UpdateRateGuardConfig returned error: %v", err)
	}
	if !cfg.Enabled || cfg.IntervalSeconds != 5 {
		t.Fatalf("saved config = %+v, want enabled interval 5", cfg)
	}

	cfg, err = svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunAt == nil || cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want successful last run", cfg)
	}
	if len(accounts.createdInputs) != 0 {
		t.Fatalf("scheduled guard should not create missing accounts, created %d", len(accounts.createdInputs))
	}
	if len(accounts.updateInputs) != 1 {
		t.Fatalf("update count = %d, want one rate guard update", len(accounts.updateInputs))
	}
	update := accounts.updateInputs[0]
	if update.id != 10 {
		t.Fatalf("updated account id = %d, want 10", update.id)
	}
	if update.input.GroupIDs == nil || len(*update.input.GroupIDs) != 1 || (*update.input.GroupIDs)[0] != 7 {
		t.Fatalf("updated group ids = %+v, want [7]", update.input.GroupIDs)
	}

	rawRecords := settings.values[SettingKeyUpstreamAccountSyncRecords]
	var records []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(rawRecords), &records); err != nil {
		t.Fatalf("decode records: %v raw=%s", err, rawRecords)
	}
	if len(records) != 1 || records[0].CreatedCount != 0 || records[0].UpdatedCount != 1 || records[0].UnboundGroupCount != 1 {
		t.Fatalf("records = %+v, want one rate guard record without creates", records)
	}
	if records[0].TriggerSource != UpstreamAccountSyncTriggerScheduledRateGuard {
		t.Fatalf("record trigger source = %q, want %q", records[0].TriggerSource, UpstreamAccountSyncTriggerScheduledRateGuard)
	}
	if len(records[0].UnbindDetails) != 1 || records[0].UnbindDetails[0].TriggerSource != UpstreamAccountSyncTriggerScheduledRateGuard {
		t.Fatalf("unbind detail trigger source = %+v, want scheduled rate guard", records[0].UnbindDetails)
	}
}

func TestUpstreamAccountRateGuardUnbindsLowGroupsWithoutActionUpdate(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "Unmapped upstream group", RateMultiplier: 2},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
			{ID: 9, Name: "Premium", Platform: PlatformOpenAI, RateMultiplier: 3, Status: StatusActive},
		},
		[]Account{{
			ID:       10,
			Name:     "up-matched",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			GroupIDs: []int64{7, 8, 9},
			Groups: []*Group{
				{ID: 7, Name: "VIP", RateMultiplier: 2},
				{ID: 8, Name: "Trial", RateMultiplier: 0.5},
				{ID: 9, Name: "Premium", RateMultiplier: 3},
			},
		}},
		settings,
	)

	cfg, err := svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunAt == nil || cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want successful last run", cfg)
	}
	if len(accounts.updateInputs) != 1 {
		t.Fatalf("update count = %d, want one rate guard update", len(accounts.updateInputs))
	}
	update := accounts.updateInputs[0]
	if update.id != 10 {
		t.Fatalf("updated account id = %d, want 10", update.id)
	}
	if update.input.GroupIDs == nil {
		t.Fatalf("updated group ids = nil, want [7 9]")
	}
	got := *update.input.GroupIDs
	if len(got) != 2 || got[0] != 7 || got[1] != 9 {
		t.Fatalf("updated group ids = %+v, want [7 9]", got)
	}

	rawRecords := settings.values[SettingKeyUpstreamAccountSyncRecords]
	var records []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(rawRecords), &records); err != nil {
		t.Fatalf("decode records: %v raw=%s", err, rawRecords)
	}
	if len(records) != 1 || records[0].UnboundGroupCount != 1 || records[0].UpdatedCount != 1 {
		t.Fatalf("records = %+v, want one update and one unbound group", records)
	}
	if len(records[0].UnbindDetails) != 1 || records[0].UnbindDetails[0].UnboundGroupIDs[0] != 8 {
		t.Fatalf("unbind details = %+v, want group 8", records[0].UnbindDetails)
	}
}
