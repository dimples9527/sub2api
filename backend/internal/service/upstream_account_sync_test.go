package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type upstreamAccountSyncProviderSourceStub struct {
	defaultProvider UpstreamProviderConfig
	providers       []UpstreamProviderConfig
	storedProviders map[string]UpstreamProviderConfig
	keys            []UpstreamProviderKey
	keysBySlug      map[string][]UpstreamProviderKey
	keysErrBySlug   map[string]error
	keyFetchDelay   time.Duration
	keyFetchCount   map[string]int
	keyFetchMu      sync.Mutex
	defaultErr      error
	providersErr    error
	keysErr         error
	fetchedSlug     string
	fetchedSlugs    []string
	fetchedMu       sync.Mutex
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
	if s.keyFetchDelay > 0 {
		timer := time.NewTimer(s.keyFetchDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}
	s.keyFetchMu.Lock()
	if s.keyFetchCount == nil {
		s.keyFetchCount = map[string]int{}
	}
	s.keyFetchCount[slug]++
	s.keyFetchMu.Unlock()

	s.fetchedMu.Lock()
	s.fetchedSlug = slug
	s.fetchedSlugs = append(s.fetchedSlugs, slug)
	s.fetchedMu.Unlock()
	if s.keysErrBySlug != nil {
		if err, ok := s.keysErrBySlug[slug]; ok {
			return nil, nil, err
		}
	}
	if s.keysBySlug != nil {
		return s.keysBySlug[slug], []string{"provider warning"}, s.keysErr
	}
	return s.keys, []string{"provider warning"}, s.keysErr
}

func (s *upstreamAccountSyncProviderSourceStub) fetchCount(slug string) int {
	s.keyFetchMu.Lock()
	defer s.keyFetchMu.Unlock()
	return s.keyFetchCount[slug]
}

type upstreamAccountSyncAccountManagerStub struct {
	accounts             []Account
	createdInputs        []CreateAccountInput
	updateInputs         []upstreamAccountSyncUpdateCall
	setSchedulableInputs []upstreamAccountSyncSetSchedulableCall
}

type upstreamAccountSyncUpdateCall struct {
	id    int64
	input UpdateAccountInput
}

type upstreamAccountSyncSetSchedulableCall struct {
	id          int64
	schedulable bool
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
		Schedulable: true,
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
			groupIDs := append([]int64(nil), (*input.GroupIDs)...)
			s.accounts[i].GroupIDs = groupIDs
			allowed := make(map[int64]struct{}, len(groupIDs))
			for _, groupID := range groupIDs {
				allowed[groupID] = struct{}{}
			}
			filteredGroups := make([]*Group, 0, len(s.accounts[i].Groups))
			for _, group := range s.accounts[i].Groups {
				if group == nil {
					continue
				}
				if _, ok := allowed[group.ID]; ok {
					filteredGroups = append(filteredGroups, group)
				}
			}
			s.accounts[i].Groups = filteredGroups
		}
		return &s.accounts[i], nil
	}
	return nil, ErrAccountNotFound
}

func (s *upstreamAccountSyncAccountManagerStub) SetAccountSchedulable(ctx context.Context, id int64, schedulable bool) (*Account, error) {
	s.setSchedulableInputs = append(s.setSchedulableInputs, upstreamAccountSyncSetSchedulableCall{id: id, schedulable: schedulable})
	for i := range s.accounts {
		if s.accounts[i].ID != id {
			continue
		}
		s.accounts[i].Schedulable = schedulable
		return &s.accounts[i], nil
	}
	return nil, ErrAccountNotFound
}

type upstreamAccountSyncPreviewCacheStub struct {
	mu      sync.Mutex
	result  UpstreamAccountSyncResult
	found   bool
	gets    int
	sets    int
	deletes int
}

func (s *upstreamAccountSyncPreviewCacheStub) Get(ctx context.Context) (UpstreamAccountSyncResult, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gets++
	return s.result, s.found, nil
}

func (s *upstreamAccountSyncPreviewCacheStub) Set(ctx context.Context, result UpstreamAccountSyncResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sets++
	s.result = result
	s.found = true
	return nil
}

func (s *upstreamAccountSyncPreviewCacheStub) Delete(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletes++
	s.result = UpstreamAccountSyncResult{}
	s.found = false
	return nil
}

func (s *upstreamAccountSyncPreviewCacheStub) stats() (sets int, deletes int, found bool, result UpstreamAccountSyncResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sets, s.deletes, s.found, s.result
}

func waitForPreviewCacheSets(t *testing.T, cache *upstreamAccountSyncPreviewCacheStub, minSets int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		sets, _, _, _ := cache.stats()
		if sets >= minSets {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	sets, deletes, found, result := cache.stats()
	t.Fatalf("cache sets/deletes/found/result = %d/%d/%v/%+v, want at least %d sets", sets, deletes, found, result, minSets)
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

func TestUpstreamAccountSyncPreviewReturnsCachedSnapshotWithoutFetchingProviders(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup upstream", AccountNamePrefix: "backup-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "live", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)
	cache := &upstreamAccountSyncPreviewCacheStub{
		found: true,
		result: UpstreamAccountSyncResult{
			Summary: UpstreamAccountSyncSummary{UpstreamKeyCount: 1},
			Items: []UpstreamAccountSyncItem{{
				Action:           UpstreamAccountSyncActionNoop,
				ProviderSlug:     "backup",
				ProviderName:     "Backup upstream",
				UpstreamKeyName:  "cached",
				LocalAccountName: "backup-cached",
			}},
			Records: []UpstreamAccountSyncRecord{},
		},
	}
	svc.SetPreviewCache(cache)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].UpstreamKeyName != "cached" {
		t.Fatalf("items = %+v, want cached snapshot", result.Items)
	}
	if provider.fetchCount("backup") != 0 {
		t.Fatalf("backup fetch count = %d, want cache hit to avoid provider fetch", provider.fetchCount("backup"))
	}
	if cache.gets != 1 || cache.sets != 0 {
		t.Fatalf("cache gets/sets = %d/%d, want one get and no set", cache.gets, cache.sets)
	}
}

func TestUpstreamAccountSyncPreviewKeepsDisabledProvidersFromCachedSnapshot(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "enabled", Name: "Enabled upstream", Enabled: true},
			{Slug: "disabled", Name: "Disabled upstream", Enabled: false},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(provider, nil, nil, newUpstreamManagementSettingRepoStub())
	cache := &upstreamAccountSyncPreviewCacheStub{
		found: true,
		result: UpstreamAccountSyncResult{
			Providers: []UpstreamProviderConfig{
				{Slug: "enabled", Name: "Enabled upstream", Enabled: true},
				{Slug: "disabled", Name: "Disabled upstream", Enabled: false},
			},
			Summary: UpstreamAccountSyncSummary{UpstreamKeyCount: 2, CreateCount: 2},
			Items: []UpstreamAccountSyncItem{
				{Action: UpstreamAccountSyncActionCreate, ProviderSlug: "enabled", ProviderName: "Enabled upstream", UpstreamKeyName: "sk-enabled"},
				{Action: UpstreamAccountSyncActionCreate, ProviderSlug: "disabled", ProviderName: "Disabled upstream", UpstreamKeyName: "sk-disabled"},
			},
			Records: []UpstreamAccountSyncRecord{},
		},
	}
	svc.SetPreviewCache(cache)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Providers) != 2 {
		t.Fatalf("providers = %+v, want enabled and disabled providers", result.Providers)
	}
	if result.Providers[1].Slug != "disabled" || result.Providers[1].Enabled {
		t.Fatalf("disabled provider = %+v, want disabled provider retained with enabled=false", result.Providers[1])
	}
	if len(result.Items) != 2 || result.Items[1].ProviderSlug != "disabled" {
		t.Fatalf("items = %+v, want disabled provider item retained", result.Items)
	}
	if result.Summary.UpstreamKeyCount != 2 || result.Summary.CreateCount != 2 {
		t.Fatalf("summary = %+v, want cached summary retained for all items", result.Summary)
	}
}

func TestUpstreamAccountSyncPreviewRefreshesAndStoresSnapshotOnCacheMiss(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup upstream", AccountNamePrefix: "backup-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "live", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)
	cache := &upstreamAccountSyncPreviewCacheStub{}
	svc.SetPreviewCache(cache)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].UpstreamKeyName != "live" {
		t.Fatalf("items = %+v, want live snapshot", result.Items)
	}
	if provider.fetchCount("backup") != 1 {
		t.Fatalf("backup fetch count = %d, want cache miss to fetch provider", provider.fetchCount("backup"))
	}
	if cache.sets != 1 || !cache.found || len(cache.result.Items) != 1 || cache.result.Items[0].UpstreamKeyName != "live" {
		t.Fatalf("cache sets/found/result = %d/%v/%+v, want stored live snapshot", cache.sets, cache.found, cache.result.Items)
	}
}

func TestUpstreamAccountSyncSyncRefreshesPreviewCacheAfterRealtimeFetch(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup upstream", AccountNamePrefix: "backup-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "fresh", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)
	cache := &upstreamAccountSyncPreviewCacheStub{
		found: true,
		result: UpstreamAccountSyncResult{
			Items: []UpstreamAccountSyncItem{{ProviderSlug: "backup", UpstreamKeyName: "stale"}},
		},
	}
	svc.SetPreviewCache(cache)

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{CreateMissing: true, UpdateExisting: true})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].UpstreamKeyName != "fresh" {
		t.Fatalf("sync items = %+v, want fresh provider data", result.Items)
	}
	waitForPreviewCacheSets(t, cache, 1)
	sets, deletes, found, refreshed := cache.stats()
	if deletes != 0 || !found {
		t.Fatalf("cache sets/deletes/found = %d/%d/%v, want refreshed cache retained", sets, deletes, found)
	}
	if len(refreshed.Items) != 1 || refreshed.Items[0].UpstreamKeyName != "fresh" {
		t.Fatalf("refreshed cache items = %+v, want fresh provider data", refreshed.Items)
	}
}

func TestUpstreamAccountSyncPreviewIncludesProvidersAndManualGroupMapping(t *testing.T) {
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
	if provider.fetchCount("main") != 1 || provider.fetchCount("backup") != 1 || provider.fetchCount("disabled") != 0 {
		t.Fatalf(
			"fetch counts main/backup/disabled = %d/%d/%d, want disabled provider skipped",
			provider.fetchCount("main"),
			provider.fetchCount("backup"),
			provider.fetchCount("disabled"),
		)
	}
	if result.Summary.UpstreamKeyCount != 1 || result.Summary.CreateCount != 1 {
		t.Fatalf("summary = %+v, want one upstream key and one create", result.Summary)
	}
	if len(result.Providers) != 2 || result.Providers[0].Slug != "main" || result.Providers[1].Slug != "backup" {
		t.Fatalf("providers = %+v, want enabled providers only", result.Providers)
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
	if item.ProviderBaseURL != "https://backup.example.com" {
		t.Fatalf("provider base url = %q, want backup provider base url", item.ProviderBaseURL)
	}
	if item.LocalGroupID == nil || *item.LocalGroupID != 9 || item.LocalGroupName != "Mapped VIP" {
		t.Fatalf("local group match = id %v name %q, want 9 Mapped VIP", item.LocalGroupID, item.LocalGroupName)
	}
}

func TestUpstreamAccountSyncPreviewIncludesDefaultProviderWithoutAccountPrefix(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{
			Slug:              "main",
			Name:              "Main upstream",
			Type:              UpstreamProviderTypeSub2API,
			BaseURL:           "https://main.example.com",
			AccountNamePrefix: "main-",
			IsDefault:         true,
			Enabled:           true,
		},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", Type: UpstreamProviderTypeSub2API, BaseURL: "https://main.example.com", AccountNamePrefix: "main-", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup upstream", Type: UpstreamProviderTypeSub2API, BaseURL: "https://backup.example.com", AccountNamePrefix: "backup-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"main": {
				{ProviderSlug: "main", KeyName: "Codex Pro", GroupName: "VIP", RateMultiplier: 1},
			},
			"backup": {
				{ProviderSlug: "backup", KeyName: "Codex Pro", GroupName: "VIP", RateMultiplier: 1},
			},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		[]Account{{
			ID:          10,
			Name:        " codex pro ",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"base_url": "https://main.example.com"},
			GroupIDs:    []int64{7},
		}, {
			ID:          11,
			Name:        "backup-Codex Pro",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"base_url": "https://backup.example.com"},
			GroupIDs:    []int64{7},
		}},
		nil,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if provider.fetchCount("main") != 1 || provider.fetchCount("backup") != 1 {
		t.Fatalf("fetch counts main/backup = %d/%d, want one fetch for each provider", provider.fetchCount("main"), provider.fetchCount("backup"))
	}
	if len(result.Providers) != 2 || result.Providers[0].Slug != "main" || !result.Providers[0].IsDefault || result.Providers[1].Slug != "backup" {
		t.Fatalf("providers = %+v, want default main followed by backup", result.Providers)
	}
	byProvider := map[string]UpstreamAccountSyncItem{}
	for _, item := range result.Items {
		byProvider[item.ProviderSlug] = item
	}
	main := byProvider["main"]
	if main.LocalAccountName != "Codex Pro" {
		t.Fatalf("default local account name = %q, want upstream key without prefix", main.LocalAccountName)
	}
	if main.MatchedAccountID == nil || *main.MatchedAccountID != 10 || main.Action != UpstreamAccountSyncActionUpdate {
		t.Fatalf("default item = %+v, want matched metadata update account 10", main)
	}
	backup := byProvider["backup"]
	if backup.LocalAccountName != "backup-Codex Pro" {
		t.Fatalf("backup local account name = %q, want prefix applied", backup.LocalAccountName)
	}
	if backup.MatchedAccountID == nil || *backup.MatchedAccountID != 11 {
		t.Fatalf("backup matched account id = %+v, want 11", backup.MatchedAccountID)
	}
}

func TestUpstreamAccountSyncPreviewDefaultProviderOnlyIgnoresSpacesAndCase(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{
			Slug:              "main",
			Name:              "Main upstream",
			AccountNamePrefix: "main-",
			IsDefault:         true,
			Enabled:           true,
		},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", AccountNamePrefix: "main-", IsDefault: true, Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"main": {{ProviderSlug: "main", KeyName: "Codex-Pro", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		[]Account{{
			ID:       10,
			Name:     "codex pro",
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
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.MatchedAccountID != nil {
		t.Fatalf("matched account id = %+v, want nil because default matching should not ignore hyphen", item.MatchedAccountID)
	}
	if item.Action != UpstreamAccountSyncActionCreate {
		t.Fatalf("action = %q, want create", item.Action)
	}
}

func TestUpstreamAccountSyncPreviewKeepsAvailableProvidersWhenOneProviderKeysFail(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "bad", Name: "Bad upstream", AccountNamePrefix: "bad-", Enabled: true},
			{Slug: "good", Name: "Good upstream", AccountNamePrefix: "good-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"good": {
				{ProviderSlug: "good", KeyName: "alice", GroupName: "VIP", RateMultiplier: 1},
			},
		},
		keysErrBySlug: map[string]error{
			"bad": errors.New("newapi provider keys failed: HTTP 502"),
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if result.Summary.UpstreamKeyCount != 1 || len(result.Items) != 1 {
		t.Fatalf("summary/items = %+v/%+v, want one good provider key", result.Summary, result.Items)
	}
	hasBadWarning := false
	for _, warning := range result.Warnings {
		if warning == "Bad upstream: newapi provider keys failed: HTTP 502" {
			hasBadWarning = true
			break
		}
	}
	if !hasBadWarning {
		t.Fatalf("warnings = %+v, want failed provider warning", result.Warnings)
	}
	if provider.fetchCount("bad") != 1 || provider.fetchCount("good") != 1 {
		t.Fatalf("fetch counts bad/good = %d/%d, want one fetch for each provider", provider.fetchCount("bad"), provider.fetchCount("good"))
	}
}

func TestUpstreamAccountSyncPreviewShowsLocalSnapshotWhenProviderKeysFail(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "bad", Name: "Bad upstream", BaseURL: "https://bad.example.com", AccountNamePrefix: "bad-", Enabled: true},
			{Slug: "good", Name: "Good upstream", BaseURL: "https://good.example.com", AccountNamePrefix: "good-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"good": {{ProviderSlug: "good", KeyName: "alice", GroupName: "VIP", RateMultiplier: 1}},
		},
		keysErrBySlug: map[string]error{
			"bad": errors.New("newapi login failed: Turnstile token is empty"),
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		[]Account{{
			ID:          42,
			Name:        "bad-alice",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "sk-local", "base_url": "https://bad.example.com"},
			Extra: map[string]any{
				"upstream_provider_slug":   "bad",
				"upstream_provider_name":   "Bad upstream",
				"upstream_key_name":        "alice",
				"upstream_group_name":      "VIP",
				"upstream_rate_multiplier": 1.5,
			},
			GroupIDs: []int64{7},
			Groups:   []*Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		}},
		newUpstreamManagementSettingRepoStub(),
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if result.Summary.UpstreamKeyCount != 2 || result.Summary.MatchedAccountCount != 1 || result.Summary.CreateCount != 1 {
		t.Fatalf("summary = %+v, want one live key plus one local snapshot", result.Summary)
	}
	if len(result.Items) != 2 {
		t.Fatalf("item count = %d, want 2", len(result.Items))
	}

	var snapshot UpstreamAccountSyncItem
	for _, item := range result.Items {
		if item.ProviderSlug == "bad" {
			snapshot = item
			break
		}
	}
	if snapshot.MatchedAccountID == nil || *snapshot.MatchedAccountID != 42 {
		t.Fatalf("snapshot matched account id = %+v, want 42", snapshot.MatchedAccountID)
	}
	if snapshot.Action != UpstreamAccountSyncActionNoop {
		t.Fatalf("snapshot action = %q, want noop", snapshot.Action)
	}
	if snapshot.ProviderFetchError != "newapi login failed: Turnstile token is empty" {
		t.Fatalf("provider fetch error = %q", snapshot.ProviderFetchError)
	}
	if snapshot.UpstreamKeyName != "alice" || snapshot.UpstreamGroupName != "VIP" || snapshot.UpstreamRateMultiplier != 1.5 {
		t.Fatalf("snapshot item = %+v, want stored upstream metadata", snapshot)
	}
	if len(snapshot.BoundGroups) != 1 || snapshot.BoundGroups[0].ID != 7 {
		t.Fatalf("snapshot bound groups = %+v, want local account bound groups", snapshot.BoundGroups)
	}
}

func TestUpstreamAccountSyncPreviewFetchesProviderKeysConcurrently(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "first", Name: "First upstream", AccountNamePrefix: "first-", Enabled: true},
			{Slug: "second", Name: "Second upstream", AccountNamePrefix: "second-", Enabled: true},
			{Slug: "third", Name: "Third upstream", AccountNamePrefix: "third-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"first":  {{ProviderSlug: "first", KeyName: "alice", GroupName: "VIP", RateMultiplier: 1}},
			"second": {{ProviderSlug: "second", KeyName: "bob", GroupName: "VIP", RateMultiplier: 1}},
			"third":  {{ProviderSlug: "third", KeyName: "carol", GroupName: "VIP", RateMultiplier: 1}},
		},
		keyFetchDelay: 80 * time.Millisecond,
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)

	start := time.Now()
	result, err := svc.Preview(context.Background())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if result.Summary.UpstreamKeyCount != 3 {
		t.Fatalf("summary = %+v, want three upstream keys", result.Summary)
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("preview elapsed = %s, want concurrent provider fetches under 200ms", elapsed)
	}
}

func TestUpstreamAccountSyncPreviewHonorsProviderFetchContextDeadline(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "slow", Name: "Slow upstream", AccountNamePrefix: "slow-", Enabled: true},
		},
		keyFetchDelay: 200 * time.Millisecond,
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	result, err := svc.Preview(ctx)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("preview elapsed = %s, want provider fetch cancellation before full delay", elapsed)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("warnings = %+v, want provider timeout warning", result.Warnings)
	}
}

func TestUpstreamAccountSyncPreviewCachesProviderKeysForThirtySeconds(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup upstream", AccountNamePrefix: "backup-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)

	if _, _, err := svc.preview(context.Background(), true); err != nil {
		t.Fatalf("first Preview returned error: %v", err)
	}
	if _, _, err := svc.preview(context.Background(), true); err != nil {
		t.Fatalf("second Preview returned error: %v", err)
	}
	if count := provider.fetchCount("backup"); count != 1 {
		t.Fatalf("backup fetch count = %d, want cached second preview", count)
	}
	if count := provider.fetchCount("main"); count != 1 {
		t.Fatalf("main fetch count = %d, want cached second preview", count)
	}
}

func TestUpstreamAccountSyncSyncBypassesPreviewProviderKeysCache(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main upstream", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup upstream", AccountNamePrefix: "backup-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "old", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		newUpstreamManagementSettingRepoStub(),
	)

	if _, err := svc.Preview(context.Background()); err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	provider.keysBySlug["backup"] = []UpstreamProviderKey{{ProviderSlug: "backup", KeyName: "new", APIKey: "sk-new", GroupName: "VIP", RateMultiplier: 1}}

	if _, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{CreateMissing: true, UpdateExisting: true}); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if len(accounts.createdInputs) != 1 || accounts.createdInputs[0].Name != "backup-new" {
		t.Fatalf("created inputs = %+v, want sync to bypass cached preview keys and create backup-new", accounts.createdInputs)
	}
	if count := provider.fetchCount("backup"); count != 2 {
		t.Fatalf("backup fetch count = %d, want preview fetch plus sync fetch", count)
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

func TestUpstreamAccountSyncPreviewMatchesPrefixedKeyNameAndQueuesMissingMetadataUpdate(t *testing.T) {
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
	if item.Action != UpstreamAccountSyncActionUpdate {
		t.Fatalf("action = %q, want update", item.Action)
	}
	if result.Summary.UpdateCount != 1 {
		t.Fatalf("update count = %d, want 1", result.Summary.UpdateCount)
	}
}

func TestUpstreamAccountSyncUpdatesMetadataForMatchedNonOpenAIAccount(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", Type: UpstreamProviderTypeSub2API, IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", Type: UpstreamProviderTypeSub2API, IsDefault: true, Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"main": {{ProviderSlug: "main", KeyName: "claude福利", GroupName: "Claude VIP", RateMultiplier: 1.2}},
		},
	}
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		nil,
		[]Account{{
			ID:          10,
			Name:        "claude福利",
			Platform:    PlatformAnthropic,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"refresh_token": "refresh"},
			Extra:       map[string]any{"existing": "kept"},
			Status:      StatusActive,
			Schedulable: true,
		}},
		nil,
	)

	preview, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("preview items = %d, want 1", len(preview.Items))
	}
	item := preview.Items[0]
	if item.Action != UpstreamAccountSyncActionUpdate || item.MatchedAccountID == nil || *item.MatchedAccountID != 10 {
		t.Fatalf("preview item = %+v, want update for matched account 10", item)
	}
	if preview.Summary.UpdateCount != 1 || preview.Summary.SkipCount != 0 {
		t.Fatalf("preview summary = %+v, want one update and no skips", preview.Summary)
	}

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{UpdateExisting: true})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if result.Summary.UpdateCount != 1 {
		t.Fatalf("sync update count = %d, want 1", result.Summary.UpdateCount)
	}
	if len(accounts.updateInputs) != 1 || accounts.updateInputs[0].id != 10 {
		t.Fatalf("update inputs = %+v, want account 10", accounts.updateInputs)
	}
	input := accounts.updateInputs[0].input
	if input.Credentials != nil {
		t.Fatalf("credentials update = %+v, want nil for metadata-only update", input.Credentials)
	}
	if input.GroupIDs != nil {
		t.Fatalf("group ids update = %+v, want nil for metadata-only update", input.GroupIDs)
	}
	if input.Extra["existing"] != "kept" ||
		input.Extra["upstream_provider_slug"] != "main" ||
		input.Extra["upstream_key_name"] != "claude福利" ||
		input.Extra["upstream_group_name"] != "Claude VIP" {
		t.Fatalf("extra update = %+v, want upstream metadata merged", input.Extra)
	}
}

func TestUpstreamAccountSyncInfersMissingMetadataFromLocalAccountName(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{
			Slug:      "findcg",
			Name:      "FindCG",
			Type:      UpstreamProviderTypeSub2API,
			BaseURL:   "https://www.findcg.com",
			IsDefault: true,
			Enabled:   true,
		},
		providers: []UpstreamProviderConfig{
			{Slug: "findcg", Name: "FindCG", Type: UpstreamProviderTypeSub2API, BaseURL: "https://www.findcg.com", IsDefault: true, Enabled: true},
			{Slug: "toltol", Name: "Toltol", Type: UpstreamProviderTypeSub2API, BaseURL: "https://toltol.me", AccountNamePrefix: "toltol-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"findcg": {},
			"toltol": {},
		},
	}
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		nil,
		[]Account{{
			ID:          10,
			Name:        "toltol-kiro-pro-channel",
			Platform:    PlatformAnthropic,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "anthropic-key", "base_url": "https://toltol.me"},
			Extra:       map[string]any{"existing": "kept"},
			Status:      StatusActive,
		}, {
			ID:          11,
			Name:        "codex pro",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "openai-key", "base_url": "https://www.findcg.com/"},
			Status:      StatusActive,
		}, {
			ID:          12,
			Name:        "unknown-prefix-account",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"base_url": "https://unknown.example.com"},
			Status:      StatusActive,
		}, {
			ID:          13,
			Name:        "disabled account",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"base_url": "https://www.findcg.com"},
			Status:      StatusDisabled,
		}},
		nil,
	)

	preview, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if preview.Summary.UpdateCount != 2 || preview.Summary.MatchedAccountCount != 2 {
		t.Fatalf("preview summary = %+v, want two inferred metadata updates", preview.Summary)
	}
	byID := map[int64]UpstreamAccountSyncItem{}
	for _, item := range preview.Items {
		if item.MatchedAccountID != nil {
			byID[*item.MatchedAccountID] = item
		}
	}
	toltolItem := byID[10]
	if toltolItem.Action != UpstreamAccountSyncActionUpdate || toltolItem.ProviderSlug != "toltol" || toltolItem.UpstreamKeyName != "kiro-pro-channel" {
		t.Fatalf("toltol item = %+v, want inferred toltol metadata update", toltolItem)
	}
	defaultItem := byID[11]
	if defaultItem.Action != UpstreamAccountSyncActionUpdate || defaultItem.ProviderSlug != "findcg" || defaultItem.UpstreamKeyName != "codex pro" {
		t.Fatalf("default item = %+v, want inferred default metadata update", defaultItem)
	}
	if _, exists := byID[12]; exists {
		t.Fatalf("unknown base url account should not be inferred as default: %+v", byID[12])
	}
	if _, exists := byID[13]; exists {
		t.Fatalf("disabled account should not be inferred: %+v", byID[13])
	}

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{UpdateExisting: true})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if result.Summary.UpdateCount != 2 {
		t.Fatalf("sync update count = %d, want 2", result.Summary.UpdateCount)
	}
	if len(accounts.updateInputs) != 2 {
		t.Fatalf("update inputs = %+v, want two metadata-only updates", accounts.updateInputs)
	}
	updatesByID := map[int64]UpdateAccountInput{}
	for _, update := range accounts.updateInputs {
		updatesByID[update.id] = update.input
		if update.input.Credentials != nil {
			t.Fatalf("credentials update for account %d = %+v, want nil", update.id, update.input.Credentials)
		}
		if update.input.GroupIDs != nil {
			t.Fatalf("group update for account %d = %+v, want nil", update.id, update.input.GroupIDs)
		}
	}
	if updatesByID[10].Extra["existing"] != "kept" ||
		updatesByID[10].Extra["upstream_provider_slug"] != "toltol" ||
		updatesByID[10].Extra["upstream_key_name"] != "kiro-pro-channel" {
		t.Fatalf("toltol extra = %+v, want inferred upstream metadata", updatesByID[10].Extra)
	}
	if updatesByID[11].Extra["upstream_provider_slug"] != "findcg" ||
		updatesByID[11].Extra["upstream_key_name"] != "codex pro" {
		t.Fatalf("default extra = %+v, want inferred default metadata", updatesByID[11].Extra)
	}
}

func TestUpstreamAccountSyncInfersMetadataWithLongestPrefixFirst(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", BaseURL: "https://main.example.com", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", BaseURL: "https://main.example.com", IsDefault: true, Enabled: true},
			{Slug: "short", Name: "Short", BaseURL: "https://short.example.com", AccountNamePrefix: "up-", Enabled: true},
			{Slug: "long", Name: "Long", BaseURL: "https://long.example.com", AccountNamePrefix: "up-pro-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"main":  {},
			"short": {},
			"long":  {},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		nil,
		[]Account{{
			ID:          10,
			Name:        "up-pro-alice",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"base_url": "https://long.example.com"},
			Status:      StatusActive,
		}},
		nil,
	)

	preview, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("preview items = %+v, want one inferred item", preview.Items)
	}
	item := preview.Items[0]
	if item.ProviderSlug != "long" || item.UpstreamKeyName != "alice" {
		t.Fatalf("item = %+v, want longest prefix provider with stripped key alice", item)
	}
}

func TestUpstreamAccountSyncPreviewMatchesRenamedAccountByStoredUpstreamIdentity(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "findcg", Name: "findcg", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"findcg": {{ProviderSlug: "findcg", KeyName: "cc官转max", GroupName: "cc官转max", RateMultiplier: 1.1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "cc官转max", Platform: PlatformOpenAI, RateMultiplier: 1.6, Status: StatusActive}},
		[]Account{{
			ID:       10,
			Name:     "cc官转manx",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"upstream_provider_slug": "findcg",
				"upstream_key_name":      "cc官转max",
			},
			GroupIDs: []int64{7},
			Groups: []*Group{
				{ID: 7, Name: "cc官转max", RateMultiplier: 1.6},
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
	item := result.Items[0]
	if item.MatchedAccountID == nil || *item.MatchedAccountID != 10 {
		t.Fatalf("matched account id = %v, want renamed account 10", item.MatchedAccountID)
	}
	if item.MatchedAccountName != "cc官转manx" {
		t.Fatalf("matched account name = %q, want renamed account", item.MatchedAccountName)
	}
	if item.RateViolation || item.Action != UpstreamAccountSyncActionNoop {
		t.Fatalf("item = %+v, want no rate violation because local 1.6 is above upstream 1.1", item)
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

func TestUpstreamAccountSyncPreviewIncludesChangeDetails(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", BaseURL: "https://backup.example.com", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "alice-new", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive},
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
		},
		[]Account{{
			ID:          10,
			Name:        "up-alice-new",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "alice-old", "base_url": "https://old.example.com/"},
			GroupIDs:    []int64{8},
			Groups: []*Group{
				{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
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
	item := result.Items[0]
	if item.Action != UpstreamAccountSyncActionUpdate {
		t.Fatalf("action = %q, want update", item.Action)
	}
	if len(item.ChangeDetails) != 4 {
		t.Fatalf("change details = %+v, want base_url/metadata/bind/unbind", item.ChangeDetails)
	}
	assertChangeDetail := func(kind, field string) UpstreamAccountSyncChangeDetail {
		t.Helper()
		for _, detail := range item.ChangeDetails {
			if detail.Kind == kind && detail.Field == field {
				return detail
			}
		}
		t.Fatalf("change details = %+v, missing %s/%s", item.ChangeDetails, kind, field)
		return UpstreamAccountSyncChangeDetail{}
	}
	baseURL := assertChangeDetail("credential", "base_url")
	if baseURL.Before != "https://old.example.com" || baseURL.After != "https://backup.example.com" {
		t.Fatalf("base url detail = %+v, want normalized old/new base urls", baseURL)
	}
	metadata := assertChangeDetail("metadata", "upstream")
	if metadata.Label == "" {
		t.Fatalf("metadata detail = %+v, want label", metadata)
	}
	bind := assertChangeDetail("group_bind", "group_ids")
	if len(bind.GroupIDs) != 1 || bind.GroupIDs[0] != 7 || len(bind.GroupNames) != 1 || bind.GroupNames[0] != "VIP" {
		t.Fatalf("bind detail = %+v, want VIP group 7", bind)
	}
	unbind := assertChangeDetail("group_unbind", "group_ids")
	if len(unbind.GroupIDs) != 1 || unbind.GroupIDs[0] != 8 || len(unbind.GroupNames) != 1 || unbind.GroupNames[0] != "Trial" {
		t.Fatalf("unbind detail = %+v, want Trial group 8", unbind)
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
			Name:     "Up Stream-Alice Key",
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
	if item.MatchedAccountName != "Up Stream-Alice Key" {
		t.Fatalf("matched account name = %q, want Up Stream-Alice Key", item.MatchedAccountName)
	}
	if len(item.BoundGroups) != 1 || item.BoundGroups[0].ID != 7 || item.BoundGroups[0].Name != "VIP" {
		t.Fatalf("bound groups = %+v, want VIP from matched account", item.BoundGroups)
	}
	if item.Action != UpstreamAccountSyncActionUpdate || len(item.ChangeDetails) != 1 || item.ChangeDetails[0].Kind != "metadata" {
		t.Fatalf("action/details = %q/%+v, want metadata-only update", item.Action, item.ChangeDetails)
	}
	if result.Summary.MatchedAccountCount != 1 || result.Summary.UpdateCount != 1 || result.Summary.SkipCount != 0 {
		t.Fatalf("summary = %+v, want one matched account and one metadata update", result.Summary)
	}

	syncResult, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{UpdateExisting: true})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if syncResult.Summary.UpdateCount != 1 {
		t.Fatalf("sync update count = %d, want 1", syncResult.Summary.UpdateCount)
	}
	if len(accountManager.updateInputs) != 1 {
		t.Fatalf("update inputs = %+v, want one metadata-only update", accountManager.updateInputs)
	}
	update := accountManager.updateInputs[0]
	if update.input.Credentials != nil || update.input.GroupIDs != nil {
		t.Fatalf("update input = %+v, want extra-only metadata update", update.input)
	}
	if update.input.Extra["upstream_provider_slug"] != "backup" || update.input.Extra["upstream_key_name"] != "Alice Key" {
		t.Fatalf("update extra = %+v, want backup/Alice Key metadata", update.input.Extra)
	}
}

func TestUpstreamAccountSyncPreviewAddsSeparatorBetweenPrefixAndKeyName(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "backup", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {{ProviderSlug: "backup", KeyName: "alice", GroupName: "VIP", RateMultiplier: 1}},
		},
	}
	svc, _ := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		nil,
		nil,
	)

	result, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(result.Items))
	}
	if result.Items[0].LocalAccountName != "backup-alice" {
		t.Fatalf("local account name = %q, want backup-alice", result.Items[0].LocalAccountName)
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
	if len(result.Providers) != 2 {
		t.Fatalf("providers = %+v, want default provider and one sync provider", result.Providers)
	}
	var beikunProvider UpstreamProviderConfig
	for _, provider := range result.Providers {
		if provider.Slug == "beikun" {
			beikunProvider = provider
			break
		}
	}
	if beikunProvider.AccountNamePrefix != "北鲲-" {
		t.Fatalf("provider prefix = %q, want stored prefix", beikunProvider.AccountNamePrefix)
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
	if result.Summary.CreateCount != 0 || result.Summary.UpdateCount != 1 || result.Summary.RateViolationCount != 1 || result.Summary.UnboundGroupCount != 1 {
		t.Fatalf("summary = %+v, want update/rate guard counts and no created account without real api key", result.Summary)
	}
	if len(accounts.createdInputs) != 0 {
		t.Fatalf("created count = %d, want 0 because upstream key name is not a usable api key", len(accounts.createdInputs))
	}
	if len(accounts.updateInputs) != 1 {
		t.Fatalf("update count = %d, want 1", len(accounts.updateInputs))
	}
	update := accounts.updateInputs[0]
	if update.id != 10 {
		t.Fatalf("updated account id = %d, want 10", update.id)
	}
	if update.input.Credentials["api_key"] != "old-key" || update.input.Credentials["base_url"] != "https://backup.example.com" {
		t.Fatalf("updated credentials = %+v, want existing api_key preserved and base_url refreshed", update.input.Credentials)
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
	if len(records) != 1 || records[0].CreatedCount != 0 || records[0].UpdatedCount != 1 || records[0].UnboundGroupCount != 1 {
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

func TestUpstreamAccountSyncRunHonorsSelectedItems(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", BaseURL: "https://backup.example.com", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "new-a", APIKey: "sk-new-a", GroupName: "VIP", RateMultiplier: 1},
				{ProviderSlug: "backup", KeyName: "new-b", APIKey: "sk-new-b", GroupName: "VIP", RateMultiplier: 1},
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "VIP", RateMultiplier: 1},
			},
		},
	}
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive},
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
		},
		[]Account{{
			ID:          10,
			Name:        "up-matched",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "matched-old", "base_url": "https://old.example.com"},
			GroupIDs:    []int64{8},
			Groups: []*Group{
				{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
			},
		}},
		nil,
	)

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{
		CreateMissing:  true,
		UpdateExisting: true,
		ApplyRateGuard: true,
		SelectedItems: []UpstreamAccountSyncSelectedItem{
			{ProviderSlug: "backup", UpstreamKeyName: "new-b", CreateMissing: true},
			{ProviderSlug: "backup", UpstreamKeyName: "matched", UpdateExisting: true, ApplyRateGuard: false},
		},
	})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if result.Summary.CreateCount != 1 || result.Summary.UpdateCount != 1 || result.Summary.UnboundGroupCount != 0 {
		t.Fatalf("summary = %+v, want one selected create, one selected update, no unbind", result.Summary)
	}
	executed := map[string]UpstreamAccountSyncExecutionResult{}
	for _, item := range result.Items {
		if item.Execution.Executed {
			executed[item.UpstreamKeyName] = item.Execution
		}
	}
	if len(executed) != 2 {
		t.Fatalf("executed items = %+v, want new-b and matched", executed)
	}
	if executed["new-b"].Action != UpstreamAccountSyncActionCreate {
		t.Fatalf("new-b execution = %+v, want create", executed["new-b"])
	}
	if executed["matched"].Action != UpstreamAccountSyncActionUpdate || len(executed["matched"].UnboundGroupIDs) != 0 {
		t.Fatalf("matched execution = %+v, want update without unbound groups", executed["matched"])
	}
	if len(accounts.createdInputs) != 1 || accounts.createdInputs[0].Name != "up-new-b" {
		t.Fatalf("created inputs = %+v, want only up-new-b", accounts.createdInputs)
	}
	if len(accounts.updateInputs) != 1 || accounts.updateInputs[0].id != 10 {
		t.Fatalf("update inputs = %+v, want only matched account", accounts.updateInputs)
	}
	if accounts.updateInputs[0].input.GroupIDs == nil || len(*accounts.updateInputs[0].input.GroupIDs) != 2 {
		t.Fatalf("updated group ids = %+v, want existing low group retained and VIP added when rate guard is disabled", accounts.updateInputs[0].input.GroupIDs)
	}
}

func TestUpstreamAccountRateGuardDisablesAccountsFromDisabledProviders(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: false},
		},
	}
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		nil,
		[]Account{{
			ID:          10,
			Name:        "up-disabled",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"upstream_provider_slug": "backup",
			},
		}},
		newUpstreamManagementSettingRepoStub(),
	)

	cfg, err := svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunAt == nil || cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want successful last run", cfg)
	}
	if len(accounts.setSchedulableInputs) != 1 {
		t.Fatalf("set schedulable calls = %+v, want one call", accounts.setSchedulableInputs)
	}
	call := accounts.setSchedulableInputs[0]
	if call.id != 10 || call.schedulable {
		t.Fatalf("set schedulable call = %+v, want account 10 disabled", call)
	}
	if accounts.accounts[0].Schedulable {
		t.Fatalf("account schedulable = true, want false")
	}
}

func TestUpstreamAccountRateGuardDisablesMatchedAccountsFromDisabledProvidersWithoutExtraSlug(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: false},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "VIP", RateMultiplier: 1},
			},
		},
	}
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 1, Status: StatusActive}},
		[]Account{{
			ID:          10,
			Name:        "up-matched",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			GroupIDs:    []int64{7},
		}},
		newUpstreamManagementSettingRepoStub(),
	)

	cfg, err := svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunAt == nil || cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want successful last run", cfg)
	}
	if len(accounts.setSchedulableInputs) != 1 {
		t.Fatalf("set schedulable calls = %+v, want one call for matched account without extra slug", accounts.setSchedulableInputs)
	}
	call := accounts.setSchedulableInputs[0]
	if call.id != 10 || call.schedulable {
		t.Fatalf("set schedulable call = %+v, want account 10 disabled", call)
	}
}

func TestUpstreamAccountSyncRunDoesNotPersistRecordsWithoutUnbindDetails(t *testing.T) {
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
				{ProviderSlug: "backup", KeyName: "alice", APIKey: "sk-alice", GroupName: "VIP", RateMultiplier: 2},
			},
			"mirror": {
				{ProviderSlug: "mirror", KeyName: "bob", APIKey: "sk-bob", GroupName: "VIP", RateMultiplier: 2},
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

	records, err := svc.ListRecords(context.Background())
	if err != nil {
		t.Fatalf("ListRecords returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("records = %+v, want no persisted sync records without unbind details", records)
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
			Extra: map[string]any{
				"upstream_provider_slug": "backup",
				"upstream_key_name":      "stable",
			},
			GroupIDs: []int64{7},
			Groups:   []*Group{{ID: 7, Name: "VIP", RateMultiplier: 2}},
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

func TestUpstreamAccountRateGuardConfigNormalizesIgnoredAccountIDs(t *testing.T) {
	settings := newUpstreamManagementSettingRepoStub()
	svc, _ := newUpstreamAccountSyncServiceForTest(
		&upstreamAccountSyncProviderSourceStub{},
		nil,
		nil,
		settings,
	)

	cfg, err := svc.UpdateRateGuardConfig(context.Background(), UpstreamAccountRateGuardConfig{
		Enabled:           true,
		IntervalSeconds:   60,
		IgnoredAccountIDs: []int64{12, 0, 5, 12, -1},
	})
	if err != nil {
		t.Fatalf("UpdateRateGuardConfig returned error: %v", err)
	}
	if len(cfg.IgnoredAccountIDs) != 2 || cfg.IgnoredAccountIDs[0] != 5 || cfg.IgnoredAccountIDs[1] != 12 {
		t.Fatalf("ignored account ids = %+v, want [5 12]", cfg.IgnoredAccountIDs)
	}

	loaded, err := svc.GetRateGuardConfig(context.Background())
	if err != nil {
		t.Fatalf("GetRateGuardConfig returned error: %v", err)
	}
	if len(loaded.IgnoredAccountIDs) != 2 || loaded.IgnoredAccountIDs[0] != 5 || loaded.IgnoredAccountIDs[1] != 12 {
		t.Fatalf("loaded ignored account ids = %+v, want [5 12]", loaded.IgnoredAccountIDs)
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

func TestUpstreamAccountRateGuardIgnoresConfiguredAccounts(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", BaseURL: "https://backup.example.com", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "VIP", RateMultiplier: 2},
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
			Extra: map[string]any{
				"upstream_provider_slug": "backup",
				"upstream_key_name":      "matched",
			},
			GroupIDs: []int64{7, 8},
			Groups: []*Group{
				{ID: 7, Name: "VIP", RateMultiplier: 2},
				{ID: 8, Name: "Trial", RateMultiplier: 0.5},
			},
		}},
		settings,
	)
	if _, err := svc.UpdateRateGuardConfig(context.Background(), UpstreamAccountRateGuardConfig{
		Enabled:           true,
		IntervalSeconds:   5,
		IgnoredAccountIDs: []int64{10},
	}); err != nil {
		t.Fatalf("UpdateRateGuardConfig returned error: %v", err)
	}

	preview, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if preview.Summary.RateViolationCount != 0 || preview.Summary.UnboundGroupCount != 0 || preview.Summary.UpdateCount != 0 {
		t.Fatalf("preview summary = %+v, want ignored account excluded from rate guard work", preview.Summary)
	}
	if len(preview.Items) != 1 || !preview.Items[0].RateGuardIgnored || preview.Items[0].RateViolation {
		t.Fatalf("preview items = %+v, want ignored non-violating matched item", preview.Items)
	}

	cfg, err := svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunAt == nil || cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want successful last run", cfg)
	}
	if len(cfg.IgnoredAccountIDs) != 1 || cfg.IgnoredAccountIDs[0] != 10 {
		t.Fatalf("ignored account ids after run = %+v, want [10]", cfg.IgnoredAccountIDs)
	}
	if len(accounts.updateInputs) != 0 {
		t.Fatalf("update count = %d, want ignored account untouched", len(accounts.updateInputs))
	}
	if rawRecords := settings.values[SettingKeyUpstreamAccountSyncRecords]; rawRecords != "" {
		t.Fatalf("records = %s, want no rate guard record for ignored account", rawRecords)
	}
}

func TestUpstreamAccountSyncApplyRateGuardIgnoresConfiguredAccounts(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", BaseURL: "https://backup.example.com", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "VIP", RateMultiplier: 2},
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
			Credentials: map[string]any{"api_key": "matched", "base_url": "https://old.example.com"},
			Extra: map[string]any{
				"upstream_provider_slug": "backup",
				"upstream_key_name":      "matched",
			},
			GroupIDs: []int64{7, 8},
			Groups: []*Group{
				{ID: 7, Name: "VIP", RateMultiplier: 2},
				{ID: 8, Name: "Trial", RateMultiplier: 0.5},
			},
		}},
		settings,
	)
	if _, err := svc.UpdateRateGuardConfig(context.Background(), UpstreamAccountRateGuardConfig{
		Enabled:           true,
		IntervalSeconds:   5,
		IgnoredAccountIDs: []int64{10},
	}); err != nil {
		t.Fatalf("UpdateRateGuardConfig returned error: %v", err)
	}

	result, err := svc.Sync(context.Background(), UpstreamAccountSyncRequest{
		UpdateExisting: true,
		ApplyRateGuard: true,
	})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if len(accounts.updateInputs) != 1 {
		t.Fatalf("update count = %d, want metadata/credential update only", len(accounts.updateInputs))
	}
	update := accounts.updateInputs[0]
	if update.input.GroupIDs == nil || len(*update.input.GroupIDs) != 2 || (*update.input.GroupIDs)[0] != 7 || (*update.input.GroupIDs)[1] != 8 {
		t.Fatalf("updated group ids = %+v, want low-rate groups preserved", update.input.GroupIDs)
	}
	if result.Summary.RateViolationCount != 0 || result.Summary.UnboundGroupCount != 0 {
		t.Fatalf("summary = %+v, want no ignored rate guard unbinds", result.Summary)
	}
	if len(result.Items) != 1 || !result.Items[0].RateGuardIgnored || len(result.Items[0].Execution.UnboundGroupIDs) != 0 {
		t.Fatalf("result items = %+v, want ignored account without unbound groups", result.Items)
	}
}

func TestUpstreamAccountRateGuardUsesProviderAccountRateMultiplierScale(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{
				Slug:                       "backup",
				Name:                       "Backup",
				BaseURL:                    "https://backup.example.com",
				AccountNamePrefix:          "up-",
				Enabled:                    true,
				AccountRateMultiplierScale: 0.1,
			},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "Trial", RateMultiplier: 1},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.5, Status: StatusActive},
		},
		[]Account{{
			ID:          10,
			Name:        "up-matched",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "matched", "base_url": "https://backup.example.com"},
			Extra: map[string]any{
				"upstream_provider_slug": "backup",
				"upstream_key_name":      "matched",
			},
			GroupIDs: []int64{8},
			Groups: []*Group{
				{ID: 8, Name: "Trial", RateMultiplier: 0.5},
			},
		}},
		settings,
	)

	preview, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(preview.Items))
	}
	item := preview.Items[0]
	if item.UpstreamRateMultiplier != 0.1 {
		t.Fatalf("upstream rate multiplier = %v, want scaled 0.1", item.UpstreamRateMultiplier)
	}
	if item.RateViolation || item.Action != UpstreamAccountSyncActionNoop {
		t.Fatalf("item = %+v, want no rate violation and noop action", item)
	}

	cfg, err := svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunAt == nil || cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want successful last run", cfg)
	}
	if len(accounts.updateInputs) != 0 {
		t.Fatalf("update count = %d, want no rate guard update", len(accounts.updateInputs))
	}
}

func TestUpstreamAccountRateGuardDoesNotPersistEmptyRecords(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "VIP", RateMultiplier: 2},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	settings.values[SettingKeyUpstreamAccountSyncRecords] = `[{
		"provider_slug":"backup",
		"provider_name":"Backup",
		"updated_count":1,
		"rate_violation_count":1,
		"unbound_group_count":1,
		"created_at":"2026-06-18T00:00:00Z",
		"trigger_source":"scheduled_rate_guard",
		"unbind_details":[{
			"provider_slug":"backup",
			"provider_name":"Backup",
			"upstream_key_name":"matched",
			"matched_local_account_id":10,
			"matched_local_account_name":"up-matched",
			"upstream_group_name":"VIP",
			"upstream_rate_multiplier":2,
			"local_min_rate_multiplier":0.5,
			"unbound_group_ids":[8],
			"unbound_group_names":["Trial"],
			"remaining_group_ids":[7],
			"trigger_source":"scheduled_rate_guard"
		}]
	}]`
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 7, Name: "VIP", Platform: PlatformOpenAI, RateMultiplier: 2, Status: StatusActive},
		},
		[]Account{{
			ID:          10,
			Name:        "up-matched",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "matched"},
			GroupIDs:    []int64{7},
			Groups: []*Group{
				{ID: 7, Name: "VIP", RateMultiplier: 2},
			},
		}},
		settings,
	)

	if _, err := svc.RunScheduledRateGuard(context.Background()); err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if len(accounts.updateInputs) != 0 {
		t.Fatalf("update count = %d, want no rate guard update", len(accounts.updateInputs))
	}

	rawRecords := settings.values[SettingKeyUpstreamAccountSyncRecords]
	var records []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(rawRecords), &records); err != nil {
		t.Fatalf("decode records: %v raw=%s", err, rawRecords)
	}
	if len(records) != 1 || len(records[0].UnbindDetails) != 1 {
		t.Fatalf("records = %+v, want existing unbind record without a new empty record", records)
	}
}

func TestUpstreamAccountSyncMarkRecordHandledPersistsUnbindDetail(t *testing.T) {
	settings := newUpstreamManagementSettingRepoStub()
	settings.values[SettingKeyUpstreamAccountSyncRecords] = `[{
		"provider_slug":"backup",
		"provider_name":"Backup",
		"updated_count":1,
		"rate_violation_count":1,
		"unbound_group_count":1,
		"created_at":"2026-06-18T00:00:00Z",
		"trigger_source":"scheduled_rate_guard",
		"unbind_details":[{
			"provider_slug":"backup",
			"provider_name":"Backup",
			"upstream_key_name":"matched",
			"matched_local_account_id":10,
			"matched_local_account_name":"up-matched",
			"upstream_group_name":"VIP",
			"upstream_rate_multiplier":2,
			"local_min_rate_multiplier":0.5,
			"unbound_group_ids":[8],
			"unbound_group_names":["Trial"],
			"remaining_group_ids":[7],
			"trigger_source":"scheduled_rate_guard"
		}]
	}]`
	svc := NewUpstreamAccountSyncService(nil, nil, nil, settings)

	records, err := svc.MarkRecordHandled(context.Background(), "2026-06-18T00:00:00Z-10-matched-8")
	if err != nil {
		t.Fatalf("MarkRecordHandled returned error: %v", err)
	}
	if len(records) != 1 || len(records[0].UnbindDetails) != 1 || !records[0].UnbindDetails[0].Handled {
		t.Fatalf("records = %+v, want handled unbind detail", records)
	}

	var persisted []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(settings.values[SettingKeyUpstreamAccountSyncRecords]), &persisted); err != nil {
		t.Fatalf("decode persisted records: %v", err)
	}
	if !persisted[0].UnbindDetails[0].Handled {
		t.Fatalf("persisted records = %+v, want handled unbind detail", persisted)
	}
}

func TestUpstreamAccountSyncMarkRecordHandledAcceptsFractionalTimestampKey(t *testing.T) {
	settings := newUpstreamManagementSettingRepoStub()
	settings.values[SettingKeyUpstreamAccountSyncRecords] = `[{
		"provider_slug":"backup",
		"provider_name":"Backup",
		"updated_count":1,
		"rate_violation_count":1,
		"unbound_group_count":1,
		"created_at":"2026-06-18T00:00:00.575Z",
		"trigger_source":"scheduled_rate_guard",
		"unbind_details":[{
			"provider_slug":"backup",
			"provider_name":"Backup",
			"upstream_key_name":"matched",
			"matched_local_account_id":10,
			"matched_local_account_name":"up-matched",
			"upstream_group_name":"VIP",
			"upstream_rate_multiplier":2,
			"local_min_rate_multiplier":0.5,
			"unbound_group_ids":[8],
			"unbound_group_names":["Trial"],
			"remaining_group_ids":[7],
			"trigger_source":"scheduled_rate_guard"
		}]
	}]`
	svc := NewUpstreamAccountSyncService(nil, nil, nil, settings)

	records, err := svc.MarkRecordHandled(context.Background(), "2026-06-18T00:00:00.575Z-10-matched-8")
	if err != nil {
		t.Fatalf("MarkRecordHandled returned error: %v", err)
	}
	if len(records) != 1 || len(records[0].UnbindDetails) != 1 || !records[0].UnbindDetails[0].Handled {
		t.Fatalf("records = %+v, want handled unbind detail", records)
	}
}

func TestUpstreamAccountRateGuardUnbindsLowGroupsForUnschedulableAccounts(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "backup", Name: "Backup", BaseURL: "https://backup.example.com", AccountNamePrefix: "up-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"backup": {
				{ProviderSlug: "backup", KeyName: "matched", GroupName: "Trial", RateMultiplier: 0.13},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 8, Name: "Trial", Platform: PlatformOpenAI, RateMultiplier: 0.1, Status: StatusActive},
		},
		[]Account{{
			ID:          10,
			Name:        "up-matched",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Schedulable: false,
			Credentials: map[string]any{"api_key": "matched", "base_url": "https://backup.example.com"},
			GroupIDs:    []int64{8},
			Groups: []*Group{
				{ID: 8, Name: "Trial", RateMultiplier: 0.1},
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
	if update.input.GroupIDs == nil || len(*update.input.GroupIDs) != 0 {
		t.Fatalf("updated group ids = %+v, want empty groups", update.input.GroupIDs)
	}

	rawRecords := settings.values[SettingKeyUpstreamAccountSyncRecords]
	var records []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(rawRecords), &records); err != nil {
		t.Fatalf("decode records: %v raw=%s", err, rawRecords)
	}
	if len(records) != 1 || records[0].UnboundGroupCount != 1 || records[0].RateViolationCount != 1 {
		t.Fatalf("records = %+v, want one rate guard unbind record", records)
	}
	if len(records[0].UnbindDetails) != 1 {
		t.Fatalf("unbind details = %+v, want one detail", records[0].UnbindDetails)
	}
	detail := records[0].UnbindDetails[0]
	if detail.MatchedLocalAccountID != 10 || detail.UpstreamRateMultiplier != 0.13 {
		t.Fatalf("unbind detail = %+v, want account 10 with upstream rate 0.13", detail)
	}
	if len(detail.UnboundGroupIDs) != 1 || detail.UnboundGroupIDs[0] != 8 {
		t.Fatalf("unbound groups = %+v, want [8]", detail.UnboundGroupIDs)
	}
	if len(detail.RemainingGroupIDs) != 0 {
		t.Fatalf("remaining groups = %+v, want empty", detail.RemainingGroupIDs)
	}
}

func TestUpstreamAccountRateGuardInvalidatesPreviewCacheAfterUnbindingAllLowGroups(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "toltol", Name: "toltol", AccountNamePrefix: "toltol-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"toltol": {
				{ProviderSlug: "toltol", KeyName: "kiroPro渠道", GroupName: "kiroPro渠道", RateMultiplier: 0.35},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 25, Name: "cc混合渠道", Platform: PlatformOpenAI, RateMultiplier: 0.25, Status: StatusActive},
			{ID: 16, Name: "claude专属分组", Platform: PlatformOpenAI, RateMultiplier: 0.16, Status: StatusActive},
		},
		[]Account{{
			ID:       202,
			Name:     "toltol-kiroPro渠道",
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			GroupIDs: []int64{25, 16},
			Groups: []*Group{
				{ID: 25, Name: "cc混合渠道", RateMultiplier: 0.25},
				{ID: 16, Name: "claude专属分组", RateMultiplier: 0.16},
			},
		}},
		settings,
	)
	cache := &upstreamAccountSyncPreviewCacheStub{
		found: true,
		result: UpstreamAccountSyncResult{
			Items: []UpstreamAccountSyncItem{{
				ProviderSlug:           "toltol",
				UpstreamKeyName:        "kiroPro渠道",
				MatchedAccountID:       ptrInt64(202),
				UpstreamRateMultiplier: 0.35,
				BoundGroups: []UpstreamAccountSyncBoundGroup{
					{ID: 25, Name: "cc混合渠道", RateMultiplier: 0.25, RateViolation: true},
					{ID: 16, Name: "claude专属分组", RateMultiplier: 0.16, RateViolation: true},
				},
			}},
		},
	}
	svc.SetPreviewCache(cache)

	cfg, err := svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want success", cfg)
	}
	if len(accounts.updateInputs) != 1 {
		t.Fatalf("update count = %d, want one rate guard update", len(accounts.updateInputs))
	}
	if got := accounts.updateInputs[0].input.GroupIDs; got == nil || len(*got) != 0 {
		t.Fatalf("updated group ids = %+v, want all low-rate groups unbound", got)
	}
	waitForPreviewCacheSets(t, cache, 1)
	sets, deletes, found, _ := cache.stats()
	if deletes != 0 || !found {
		t.Fatalf("cache sets/deletes/found = %d/%d/%v, want refreshed cache retained", sets, deletes, found)
	}

	preview, err := svc.Preview(context.Background())
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("preview items = %+v, want one item", preview.Items)
	}
	if len(preview.Items[0].BoundGroups) != 0 || preview.Items[0].RateViolation {
		t.Fatalf("preview item = %+v, want no remaining low-rate bound groups after cache invalidation", preview.Items[0])
	}
}

func TestUpstreamAccountRateGuardUnbindsLowGroupsForNonOpenAIAPIKeyAccounts(t *testing.T) {
	provider := &upstreamAccountSyncProviderSourceStub{
		defaultProvider: UpstreamProviderConfig{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
		providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", IsDefault: true, Enabled: true},
			{Slug: "toltol", Name: "toltol", AccountNamePrefix: "toltol-", Enabled: true},
		},
		keysBySlug: map[string][]UpstreamProviderKey{
			"toltol": {
				{ProviderSlug: "toltol", KeyName: "kiroPro渠道", GroupName: "kiroPro渠道", RateMultiplier: 0.35},
			},
		},
	}
	settings := newUpstreamManagementSettingRepoStub()
	svc, accounts := newUpstreamAccountSyncServiceForTest(
		provider,
		[]Group{
			{ID: 25, Name: "cc混合渠道", Platform: PlatformAnthropic, RateMultiplier: 0.25, Status: StatusActive},
			{ID: 16, Name: "claude专属分组", Platform: PlatformAnthropic, RateMultiplier: 0.16, Status: StatusActive},
		},
		[]Account{{
			ID:       202,
			Name:     "toltol-kiroPro渠道",
			Platform: PlatformAnthropic,
			Type:     AccountTypeAPIKey,
			GroupIDs: []int64{25, 16},
			Groups: []*Group{
				{ID: 25, Name: "cc混合渠道", Platform: PlatformAnthropic, RateMultiplier: 0.25},
				{ID: 16, Name: "claude专属分组", Platform: PlatformAnthropic, RateMultiplier: 0.16},
			},
		}},
		settings,
	)

	cfg, err := svc.RunScheduledRateGuard(context.Background())
	if err != nil {
		t.Fatalf("RunScheduledRateGuard returned error: %v", err)
	}
	if cfg.LastRunStatus != "success" {
		t.Fatalf("run config = %+v, want success", cfg)
	}
	if len(accounts.updateInputs) != 1 {
		t.Fatalf("update count = %d, want one rate guard update", len(accounts.updateInputs))
	}
	if got := accounts.updateInputs[0].input.GroupIDs; got == nil || len(*got) != 0 {
		t.Fatalf("updated group ids = %+v, want all low-rate groups unbound", got)
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
