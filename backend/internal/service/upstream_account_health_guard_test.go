package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type upstreamAccountHealthGuardAccountStoreStub struct {
	accounts     []Account
	setCalls     []upstreamAccountHealthGuardSetCall
	extraUpdates map[int64]map[string]any
}

type upstreamAccountHealthGuardSetCall struct {
	id          int64
	schedulable bool
}

func (s *upstreamAccountHealthGuardAccountStoreStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, _, _, _, _ string, _ int64, _ string) ([]Account, *pagination.PaginationResult, error) {
	out := make([]Account, len(s.accounts))
	copy(out, s.accounts)
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (s *upstreamAccountHealthGuardAccountStoreStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if s.extraUpdates == nil {
		s.extraUpdates = map[int64]map[string]any{}
	}
	s.extraUpdates[id] = copyAnyMap(updates)
	for i := range s.accounts {
		if s.accounts[i].ID != id {
			continue
		}
		if s.accounts[i].Extra == nil {
			s.accounts[i].Extra = map[string]any{}
		}
		for key, value := range updates {
			s.accounts[i].Extra[key] = value
		}
	}
	return nil
}

func (s *upstreamAccountHealthGuardAccountStoreStub) SetSchedulable(_ context.Context, id int64, schedulable bool) error {
	s.setCalls = append(s.setCalls, upstreamAccountHealthGuardSetCall{id: id, schedulable: schedulable})
	for i := range s.accounts {
		if s.accounts[i].ID == id {
			s.accounts[i].Schedulable = schedulable
			return nil
		}
	}
	return ErrAccountNotFound
}

type upstreamAccountHealthGuardProviderStub struct {
	providers []UpstreamProviderConfig
	err       error
}

func (s *upstreamAccountHealthGuardProviderStub) ListProviders(context.Context) ([]UpstreamProviderConfig, error) {
	return s.providers, s.err
}

type upstreamAccountHealthGuardTesterStub struct {
	results map[int64]*ScheduledTestResult
	errs    map[int64]error
	calls   []upstreamAccountHealthGuardTestCall
}

type upstreamAccountHealthGuardTestCall struct {
	accountID int64
	modelID   string
}

func (s *upstreamAccountHealthGuardTesterStub) runTestBackground(_ context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	s.calls = append(s.calls, upstreamAccountHealthGuardTestCall{accountID: accountID, modelID: modelID})
	if s.errs != nil && s.errs[accountID] != nil {
		return s.results[accountID], s.errs[accountID]
	}
	return s.results[accountID], nil
}

type upstreamAccountHealthGuardRecordStoreStub struct {
	records []UpstreamAccountHealthGuardRunRecord
}

func (s *upstreamAccountHealthGuardRecordStoreStub) SaveRecord(_ context.Context, record UpstreamAccountHealthGuardRunRecord, keepLimit int) error {
	next := []UpstreamAccountHealthGuardRunRecord{record}
	for _, existing := range s.records {
		if existing.ID != record.ID {
			next = append(next, existing)
		}
	}
	if keepLimit > 0 && len(next) > keepLimit {
		next = next[:keepLimit]
	}
	s.records = next
	return nil
}

func (s *upstreamAccountHealthGuardRecordStoreStub) ListRecords(_ context.Context, limit int) ([]UpstreamAccountHealthGuardRunRecord, error) {
	out := make([]UpstreamAccountHealthGuardRunRecord, len(s.records))
	copy(out, s.records)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type upstreamAccountHealthGuardSettingRepoStub struct {
	values map[string]string
}

func (s *upstreamAccountHealthGuardSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *upstreamAccountHealthGuardSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *upstreamAccountHealthGuardSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *upstreamAccountHealthGuardSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *upstreamAccountHealthGuardSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *upstreamAccountHealthGuardSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *upstreamAccountHealthGuardSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func newUpstreamAccountHealthGuardTestService(accounts []Account, tester *upstreamAccountHealthGuardTesterStub) (*UpstreamAccountHealthGuardService, *upstreamAccountHealthGuardAccountStoreStub, *upstreamAccountHealthGuardSettingRepoStub) {
	store := &upstreamAccountHealthGuardAccountStoreStub{accounts: accounts}
	settings := &upstreamAccountHealthGuardSettingRepoStub{}
	svc := newUpstreamAccountHealthGuardServiceWithDeps(
		store,
		&upstreamAccountHealthGuardProviderStub{providers: []UpstreamProviderConfig{{Slug: "main", Name: "Main", Enabled: true}}},
		settings,
		tester,
		&upstreamAccountHealthGuardRecordStoreStub{},
	)
	svc.now = func() time.Time { return time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC) }
	return svc, store, settings
}

func upstreamAccountHealthGuardAccount(id int64, schedulable bool, extra map[string]any) Account {
	return Account{
		ID:          id,
		Name:        "account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: schedulable,
		Extra:       extra,
	}
}

func upstreamAccountHealthGuardResult(status string, latency int64) *ScheduledTestResult {
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	return &ScheduledTestResult{
		Status:     status,
		LatencyMs:  latency,
		StartedAt:  now,
		FinishedAt: now.Add(time.Duration(latency) * time.Millisecond),
	}
}

func TestUpstreamAccountHealthGuardFailureThresholdDisablesScheduling(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{1: {Status: "failed", ErrorMessage: "401"}},
		errs:    map[int64]error{1: errors.New("401")},
	}
	svc, store, _ := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{
			"upstream_provider_slug":                "main",
			upstreamHealthGuardFailureCountExtraKey: 2,
		}),
	}, tester)
	_, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{
		FailureThreshold:  3,
		SlowThreshold:     3,
		RecoveryThreshold: 2,
		Concurrency:       1,
		PlatformModels:    map[string]string{"openai": "gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	response, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if got := len(store.setCalls); got != 1 {
		t.Fatalf("set calls = %d, want 1", got)
	}
	if store.setCalls[0].schedulable {
		t.Fatalf("account should be disabled")
	}
	item := response.Record.Items[0]
	if item.Action != UpstreamAccountHealthGuardActionDisabled || item.ConsecutiveFailed != 3 {
		t.Fatalf("item = %+v, want disabled with 3 failures", item)
	}
	if tester.calls[0].modelID != "gpt-4o-mini" {
		t.Fatalf("model = %q, want platform model", tester.calls[0].modelID)
	}
}

func TestUpstreamAccountHealthGuardSlowSuccessDisablesAndDoesNotRecover(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{
			1: upstreamAccountHealthGuardResult("success", 20000),
			2: upstreamAccountHealthGuardResult("success", 20000),
		},
	}
	svc, store, _ := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{
			"upstream_provider_slug":             "main",
			upstreamHealthGuardSlowCountExtraKey: 1,
		}),
		upstreamAccountHealthGuardAccount(2, false, map[string]any{
			"upstream_provider_slug": "main",
		}),
	}, tester)
	_, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{
		HealthyLatencyMs:  10000,
		FailureThreshold:  3,
		SlowThreshold:     2,
		RecoveryThreshold: 2,
		Concurrency:       1,
	})
	if err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	response, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(store.setCalls) != 1 || store.setCalls[0].id != 1 || store.setCalls[0].schedulable {
		t.Fatalf("set calls = %+v, want only account 1 disabled", store.setCalls)
	}
	if response.Record.Summary.SlowCount != 2 || response.Record.Summary.RecoveredCount != 0 {
		t.Fatalf("summary = %+v, want slow count 2 and no recovery", response.Record.Summary)
	}
}

func TestUpstreamAccountHealthGuardHealthyThresholdRecoversScheduling(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{1: upstreamAccountHealthGuardResult("success", 200)},
	}
	svc, store, _ := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, false, map[string]any{
			"upstream_provider_slug":                "main",
			upstreamHealthGuardHealthyCountExtraKey: 1,
		}),
	}, tester)
	_, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{
		HealthyLatencyMs:  10000,
		FailureThreshold:  3,
		SlowThreshold:     3,
		RecoveryThreshold: 2,
		Concurrency:       1,
	})
	if err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	response, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(store.setCalls) != 1 || !store.setCalls[0].schedulable {
		t.Fatalf("set calls = %+v, want recovery", store.setCalls)
	}
	item := response.Record.Items[0]
	if item.Action != UpstreamAccountHealthGuardActionRecovered || item.ConsecutiveHealthy != 2 {
		t.Fatalf("item = %+v, want recovered with 2 healthy", item)
	}
}

func TestUpstreamAccountHealthGuardSkipsDisabledProviderAndAccount(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{5: upstreamAccountHealthGuardResult("success", 100)},
	}
	store := &upstreamAccountHealthGuardAccountStoreStub{accounts: []Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{"upstream_provider_slug": "disabled"}),
		{ID: 2, Name: "disabled-account", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusDisabled, Schedulable: true, Extra: map[string]any{"upstream_provider_slug": "main"}},
		upstreamAccountHealthGuardAccount(3, true, map[string]any{}),
		upstreamAccountHealthGuardAccount(4, true, map[string]any{"upstream_provider_slug": "missing"}),
		upstreamAccountHealthGuardAccount(5, true, map[string]any{"upstream_provider_slug": "main"}),
	}}
	settings := &upstreamAccountHealthGuardSettingRepoStub{}
	svc := newUpstreamAccountHealthGuardServiceWithDeps(
		store,
		&upstreamAccountHealthGuardProviderStub{providers: []UpstreamProviderConfig{
			{Slug: "main", Name: "Main", Enabled: true},
			{Slug: "disabled", Name: "Disabled", Enabled: false},
		}},
		settings,
		tester,
		&upstreamAccountHealthGuardRecordStoreStub{},
	)
	if _, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{Concurrency: 1}); err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	response, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if response.Record.Summary.CheckedCount != 1 || response.Record.Summary.SkippedCount != 4 {
		t.Fatalf("summary = %+v, want 1 checked and 4 skipped", response.Record.Summary)
	}
	if len(tester.calls) != 1 || tester.calls[0].accountID != 5 {
		t.Fatalf("calls = %+v, want only account 5 tested", tester.calls)
	}
	reasons := map[string]UpstreamAccountHealthGuardSkipReason{}
	for _, reason := range response.Record.Summary.SkipReasons {
		reasons[reason.Reason] = reason
	}
	for _, reason := range []string{
		upstreamAccountHealthGuardSkipProviderDisabled,
		upstreamAccountHealthGuardSkipAccountDisabled,
		upstreamAccountHealthGuardSkipMissingProvider,
		upstreamAccountHealthGuardSkipProviderNotFound,
	} {
		if reasons[reason].Count != 1 {
			t.Fatalf("skip reason %q = %+v, want count 1", reason, reasons[reason])
		}
		if len(reasons[reason].SampleAccounts) != 1 {
			t.Fatalf("skip reason %q samples = %+v, want one sample", reason, reasons[reason].SampleAccounts)
		}
	}
}

func TestUpstreamAccountHealthGuardIgnoresConfiguredAccounts(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{
			1: {Status: "failed", ErrorMessage: "401"},
			2: upstreamAccountHealthGuardResult("success", 100),
		},
		errs: map[int64]error{1: errors.New("401")},
	}
	svc, store, _ := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{
			"upstream_provider_slug":                "main",
			upstreamHealthGuardFailureCountExtraKey: 2,
		}),
		upstreamAccountHealthGuardAccount(2, true, map[string]any{"upstream_provider_slug": "main"}),
	}, tester)
	if _, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{
		FailureThreshold:  3,
		SlowThreshold:     3,
		RecoveryThreshold: 2,
		Concurrency:       1,
		IgnoredAccountIDs: []int64{1, 1, 0, -2},
	}); err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	response, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(store.setCalls) != 0 {
		t.Fatalf("set calls = %+v, want ignored account unchanged", store.setCalls)
	}
	if len(tester.calls) != 1 || tester.calls[0].accountID != 2 {
		t.Fatalf("calls = %+v, want only account 2 tested", tester.calls)
	}
	if response.Config.IgnoredAccountIDs == nil || len(response.Config.IgnoredAccountIDs) != 1 || response.Config.IgnoredAccountIDs[0] != 1 {
		t.Fatalf("ignored account ids = %+v, want [1]", response.Config.IgnoredAccountIDs)
	}
	if response.Record.Summary.CheckedCount != 1 || response.Record.Summary.SkippedCount != 1 {
		t.Fatalf("summary = %+v, want 1 checked and 1 skipped", response.Record.Summary)
	}
	reasons := map[string]UpstreamAccountHealthGuardSkipReason{}
	for _, reason := range response.Record.Summary.SkipReasons {
		reasons[reason.Reason] = reason
	}
	if reasons[upstreamAccountHealthGuardSkipAccountIgnored].Count != 1 {
		t.Fatalf("ignored skip reason = %+v, want count 1", reasons[upstreamAccountHealthGuardSkipAccountIgnored])
	}
}

func TestUpstreamAccountHealthGuardAccountModelOverridesPlatformModel(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{
			1: upstreamAccountHealthGuardResult("success", 100),
			2: upstreamAccountHealthGuardResult("success", 100),
		},
	}
	svc, _, _ := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{"upstream_provider_slug": "main"}),
		upstreamAccountHealthGuardAccount(2, true, map[string]any{"upstream_provider_slug": "main"}),
	}, tester)
	if _, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{
		Concurrency:    1,
		PlatformModels: map[string]string{"openai": "platform-model"},
		AccountModels:  map[int64]string{2: "account-model", 0: "invalid", 3: ""},
	}); err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	response, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if got := response.Config.AccountModels; len(got) != 1 || got[2] != "account-model" {
		t.Fatalf("account models = %+v, want only account 2 override", got)
	}
	if len(tester.calls) != 2 {
		t.Fatalf("calls = %+v, want 2 calls", tester.calls)
	}
	if tester.calls[0].accountID != 1 || tester.calls[0].modelID != "platform-model" {
		t.Fatalf("first call = %+v, want account 1 platform model", tester.calls[0])
	}
	if tester.calls[1].accountID != 2 || tester.calls[1].modelID != "account-model" {
		t.Fatalf("second call = %+v, want account 2 account model", tester.calls[1])
	}
}

func TestUpstreamAccountHealthGuardHonorsRateGuardIgnoredAccounts(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{
			1: {Status: "failed", ErrorMessage: "401"},
			2: upstreamAccountHealthGuardResult("success", 100),
		},
		errs: map[int64]error{1: errors.New("401")},
	}
	svc, store, settings := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{
			"upstream_provider_slug":                "main",
			upstreamHealthGuardFailureCountExtraKey: 2,
		}),
		upstreamAccountHealthGuardAccount(2, true, map[string]any{"upstream_provider_slug": "main"}),
	}, tester)
	settings.values = map[string]string{
		SettingKeyUpstreamAccountRateGuardConfig: `{"ignored_account_ids":[1]}`,
	}
	if _, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{
		FailureThreshold:  3,
		SlowThreshold:     3,
		RecoveryThreshold: 2,
		Concurrency:       1,
	}); err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	response, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(store.setCalls) != 0 {
		t.Fatalf("set calls = %+v, want rate-guard ignored account unchanged", store.setCalls)
	}
	if len(tester.calls) != 1 || tester.calls[0].accountID != 2 {
		t.Fatalf("calls = %+v, want only account 2 tested", tester.calls)
	}
	if response.Record.Summary.CheckedCount != 1 || response.Record.Summary.SkippedCount != 1 {
		t.Fatalf("summary = %+v, want 1 checked and 1 skipped", response.Record.Summary)
	}
}

func TestUpstreamAccountHealthGuardRotatesWhenRunLimitReached(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{
			1: upstreamAccountHealthGuardResult("success", 100),
			2: upstreamAccountHealthGuardResult("success", 100),
			3: upstreamAccountHealthGuardResult("success", 100),
		},
	}
	svc, _, _ := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{"upstream_provider_slug": "main"}),
		upstreamAccountHealthGuardAccount(2, true, map[string]any{"upstream_provider_slug": "main"}),
		upstreamAccountHealthGuardAccount(3, true, map[string]any{"upstream_provider_slug": "main"}),
	}, tester)
	if _, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{
		MaxAccountsPerRun: 2,
		Concurrency:       1,
	}); err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}

	first, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("first Run error: %v", err)
	}
	second, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual)
	if err != nil {
		t.Fatalf("second Run error: %v", err)
	}

	if got := []int64{first.Record.Items[0].AccountID, first.Record.Items[1].AccountID}; got[0] != 1 || got[1] != 2 {
		t.Fatalf("first accounts = %+v, want [1 2]", got)
	}
	if got := []int64{second.Record.Items[0].AccountID, second.Record.Items[1].AccountID}; got[0] != 1 || got[1] != 3 {
		t.Fatalf("second sorted accounts = %+v, want [1 3]", got)
	}
	if second.Config.CursorAccountID != 1 {
		t.Fatalf("cursor = %d, want 1", second.Config.CursorAccountID)
	}
}

func TestUpstreamAccountHealthGuardRecordsPersist(t *testing.T) {
	tester := &upstreamAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{1: upstreamAccountHealthGuardResult("success", 100)},
	}
	svc, _, settings := newUpstreamAccountHealthGuardTestService([]Account{
		upstreamAccountHealthGuardAccount(1, true, map[string]any{"upstream_provider_slug": "main"}),
	}, tester)
	if _, err := svc.UpdateConfig(context.Background(), UpstreamAccountHealthGuardConfig{Concurrency: 1}); err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}
	if _, err := svc.Run(context.Background(), UpstreamAccountHealthGuardTriggerManual); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	records, err := svc.ListRecords(context.Background())
	if err != nil {
		t.Fatalf("ListRecords error: %v", err)
	}
	if len(records) != 1 || records[0].Summary.CheckedCount != 1 {
		t.Fatalf("records = %+v, want one checked record", records)
	}
	if _, ok := settings.values[SettingKeyUpstreamAccountHealthGuardRecords]; ok {
		t.Fatalf("records should not be persisted in settings")
	}
}
