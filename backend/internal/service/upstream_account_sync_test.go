package service

import (
	"context"
	"encoding/json"
	"testing"
)

type upstreamAccountSyncProviderSourceStub struct {
	defaultProvider UpstreamProviderConfig
	providers       []UpstreamProviderConfig
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
			{ID: 1, Name: "up-alice", Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
			{ID: 2, Name: " UP-ALICE ", Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
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
