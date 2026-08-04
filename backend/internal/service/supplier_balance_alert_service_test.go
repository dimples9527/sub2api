package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type supplierBalanceAlertRepoStub struct {
	configs   map[int64]SupplierBalanceAlertConfig
	active    map[int64]*SupplierBalanceAlertEvent
	events    []SupplierBalanceAlertEvent
	states    map[int64]supplierBalanceAlertState
	nextEvent int64
}

type supplierBalanceAlertState struct {
	now     time.Time
	balance *decimal.Decimal
	status  string
	message string
}

func supplierBalanceAlertTestSyncTime() *time.Time {
	syncedAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	return &syncedAt
}

func (r *supplierBalanceAlertRepoStub) ListConfigs(context.Context, int64) ([]SupplierBalanceAlertConfig, error) {
	items := make([]SupplierBalanceAlertConfig, 0, len(r.configs))
	for _, item := range r.configs {
		items = append(items, item)
	}
	return items, nil
}
func (r *supplierBalanceAlertRepoStub) GetConfig(_ context.Context, providerID int64) (*SupplierBalanceAlertConfig, error) {
	item, ok := r.configs[providerID]
	if !ok {
		return nil, ErrSupplierBalanceAlertConfigNotFound
	}
	return &item, nil
}
func (r *supplierBalanceAlertRepoStub) UpsertConfig(_ context.Context, providerID int64, enabled bool, threshold decimal.Decimal, cooldownSeconds int) (*SupplierBalanceAlertConfig, error) {
	item := SupplierBalanceAlertConfig{ProviderID: providerID, Enabled: enabled, Threshold: threshold, CooldownSeconds: cooldownSeconds}
	r.configs[providerID] = item
	return &item, nil
}
func (r *supplierBalanceAlertRepoStub) UpdateScanState(_ context.Context, providerID int64, now time.Time, balance *decimal.Decimal, status, message string) error {
	r.states[providerID] = supplierBalanceAlertState{now: now, balance: balance, status: status, message: message}
	return nil
}
func (r *supplierBalanceAlertRepoStub) GetActiveLowEvent(_ context.Context, providerID int64) (*SupplierBalanceAlertEvent, error) {
	item := r.active[providerID]
	if item == nil {
		return nil, nil
	}
	copy := *item
	return &copy, nil
}
func (r *supplierBalanceAlertRepoStub) CreateEvent(_ context.Context, event *SupplierBalanceAlertEvent) error {
	r.nextEvent++
	event.ID = r.nextEvent
	if event.CreatedAt.IsZero() {
		event.CreatedAt = event.ObservedAt
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = event.ObservedAt
	}
	copy := *event
	r.events = append(r.events, copy)
	if event.EventType == SupplierBalanceAlertEventLow && event.Status == SupplierBalanceAlertEventActive {
		r.active[event.ProviderID] = &copy
	}
	return nil
}
func (r *supplierBalanceAlertRepoStub) TouchActiveLowEvent(_ context.Context, eventID int64, balance decimal.Decimal, now time.Time) error {
	for providerID, event := range r.active {
		if event.ID == eventID {
			event.Balance = balance
			event.LastSeenAt = now
			r.active[providerID] = event
			return nil
		}
	}
	return ErrSupplierBalanceAlertEventNotFound
}
func (r *supplierBalanceAlertRepoStub) ResolveActiveLowEvent(_ context.Context, eventID int64, now time.Time, balance decimal.Decimal) error {
	for providerID, event := range r.active {
		if event.ID == eventID {
			event.Status = SupplierBalanceAlertEventResolved
			event.Balance = balance
			event.ResolvedAt = &now
			event.LastSeenAt = now
			delete(r.active, providerID)
			return nil
		}
	}
	return ErrSupplierBalanceAlertEventNotFound
}
func (r *supplierBalanceAlertRepoStub) ListEvents(context.Context, SupplierBalanceAlertEventListParams) (SupplierBalanceAlertEventListResult, error) {
	return SupplierBalanceAlertEventListResult{Items: r.events, Total: int64(len(r.events)), Page: 1, PageSize: 50}, nil
}

type supplierBalanceSourceStub struct {
	providers []SupplierBalanceProvider
	balances  map[int64]decimal.Decimal
	errors    map[int64]error
}

func (s *supplierBalanceSourceStub) ListEnabledProviders(context.Context) ([]SupplierBalanceProvider, error) {
	return s.providers, nil
}
func (s *supplierBalanceSourceStub) FetchBalance(_ context.Context, provider SupplierBalanceProvider) (decimal.Decimal, error) {
	if err := s.errors[provider.ID]; err != nil {
		return decimal.Zero, err
	}
	return s.balances[provider.ID], nil
}

type supplierBalanceDispatcherStub struct {
	events []SupplierBalanceAlertEvent
}

func (d *supplierBalanceDispatcherStub) Dispatch(_ context.Context, event SupplierBalanceAlertEvent) error {
	d.events = append(d.events, event)
	return nil
}

func TestSupplierBalanceAlertServiceTriggersOnlyWhenBalanceIsStrictlyBelowThreshold(t *testing.T) {
	repo := &supplierBalanceAlertRepoStub{
		configs: map[int64]SupplierBalanceAlertConfig{
			1: {ProviderID: 1, ProviderName: "供应商一", Enabled: true, Threshold: decimal.NewFromInt(10), CooldownSeconds: 3600},
			2: {ProviderID: 2, ProviderName: "供应商二", Enabled: true, Threshold: decimal.NewFromInt(10), CooldownSeconds: 3600},
			3: {ProviderID: 3, ProviderName: "供应商三", Enabled: true, Threshold: decimal.Zero, CooldownSeconds: 3600},
		},
		active: make(map[int64]*SupplierBalanceAlertEvent),
		states: make(map[int64]supplierBalanceAlertState),
	}
	source := &supplierBalanceSourceStub{
		providers: []SupplierBalanceProvider{{ID: 1, Name: "供应商一", LastSyncAt: supplierBalanceAlertTestSyncTime()}, {ID: 2, Name: "供应商二", LastSyncAt: supplierBalanceAlertTestSyncTime()}, {ID: 3, Name: "供应商三", LastSyncAt: supplierBalanceAlertTestSyncTime()}},
		balances:  map[int64]decimal.Decimal{1: decimal.NewFromInt(9), 2: decimal.NewFromInt(10), 3: decimal.NewFromInt(-1)},
		errors:    make(map[int64]error),
	}
	dispatcher := &supplierBalanceDispatcherStub{}
	service := NewSupplierBalanceAlertService(repo, source, dispatcher)

	result, err := service.RunNow(context.Background())
	if err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}
	if result.Triggered != 1 || len(dispatcher.events) != 1 {
		t.Fatalf("triggered/events = %d/%d, want 1/1", result.Triggered, len(dispatcher.events))
	}
	if dispatcher.events[0].ProviderID != 1 || dispatcher.events[0].EventType != SupplierBalanceAlertEventLow {
		t.Fatalf("dispatched event = %+v", dispatcher.events[0])
	}
	if got := repo.states[2].status; got != SupplierBalanceAlertScanStatusOK {
		t.Fatalf("equal threshold scan status = %q, want %q", got, SupplierBalanceAlertScanStatusOK)
	}
	if got := repo.states[3].status; got != SupplierBalanceAlertScanStatusSkip {
		t.Fatalf("zero threshold scan status = %q, want %q", got, SupplierBalanceAlertScanStatusSkip)
	}
}

func TestSupplierBalanceAlertServiceDoesNotDuplicateActiveEventAndCreatesRecovery(t *testing.T) {
	now := time.Now()
	active := &SupplierBalanceAlertEvent{ID: 7, ProviderID: 1, ProviderName: "供应商一", EventType: SupplierBalanceAlertEventLow, Status: SupplierBalanceAlertEventActive, Balance: decimal.NewFromInt(5), Threshold: decimal.NewFromInt(10), ObservedAt: now, LastSeenAt: now}
	repo := &supplierBalanceAlertRepoStub{
		configs: map[int64]SupplierBalanceAlertConfig{1: {ProviderID: 1, ProviderName: "供应商一", Enabled: true, Threshold: decimal.NewFromInt(10), CooldownSeconds: 60}},
		active:  map[int64]*SupplierBalanceAlertEvent{1: active},
		states:  make(map[int64]supplierBalanceAlertState),
	}
	source := &supplierBalanceSourceStub{providers: []SupplierBalanceProvider{{ID: 1, Name: "供应商一", LastSyncAt: supplierBalanceAlertTestSyncTime()}}, balances: map[int64]decimal.Decimal{1: decimal.NewFromInt(11)}, errors: map[int64]error{}}
	dispatcher := &supplierBalanceDispatcherStub{}
	service := NewSupplierBalanceAlertService(repo, source, dispatcher)

	result, err := service.RunNow(context.Background())
	if err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}
	if result.Recovered != 1 || len(repo.events) != 1 || len(dispatcher.events) != 1 {
		t.Fatalf("recovered/events/dispatches = %d/%d/%d, want 1/1/1", result.Recovered, len(repo.events), len(dispatcher.events))
	}
	if repo.events[0].EventType != SupplierBalanceAlertEventRecovered || repo.events[0].Status != SupplierBalanceAlertEventResolved {
		t.Fatalf("recovery event = %+v", repo.events[0])
	}
	if dispatcher.events[0].EventType != SupplierBalanceAlertEventRecovered {
		t.Fatalf("dispatched recovery event = %+v", dispatcher.events[0])
	}
}

func TestSupplierBalanceAlertServiceDispatchesAgainForActiveLowEvent(t *testing.T) {
	now := time.Now()
	active := &SupplierBalanceAlertEvent{
		ID:           7,
		ProviderID:   1,
		ProviderName: "供应商一",
		EventType:    SupplierBalanceAlertEventLow,
		Status:       SupplierBalanceAlertEventActive,
		Balance:      decimal.NewFromInt(5),
		Threshold:    decimal.NewFromInt(10),
		ObservedAt:   now,
		LastSeenAt:   now,
	}
	repo := &supplierBalanceAlertRepoStub{
		configs: map[int64]SupplierBalanceAlertConfig{
			1: {ProviderID: 1, ProviderName: "供应商一", Enabled: true, Threshold: decimal.NewFromInt(10), CooldownSeconds: 60},
		},
		active: map[int64]*SupplierBalanceAlertEvent{1: active},
		states: make(map[int64]supplierBalanceAlertState),
	}
	source := &supplierBalanceSourceStub{
		providers: []SupplierBalanceProvider{{ID: 1, Name: "供应商一", LastSyncAt: supplierBalanceAlertTestSyncTime()}},
		balances:  map[int64]decimal.Decimal{1: decimal.NewFromInt(5)},
		errors:    map[int64]error{},
	}
	dispatcher := &supplierBalanceDispatcherStub{}
	service := NewSupplierBalanceAlertService(repo, source, dispatcher)

	result, err := service.RunNow(context.Background())
	if err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}
	if result.Triggered != 1 || len(repo.events) != 0 || len(dispatcher.events) != 1 {
		t.Fatalf("triggered/events/dispatches = %d/%d/%d, want 1/0/1", result.Triggered, len(repo.events), len(dispatcher.events))
	}
	if event := dispatcher.events[0]; event.ID != active.ID || event.EventType != SupplierBalanceAlertEventLow || !event.Balance.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("dispatched event = %+v", event)
	}
}

func TestSupplierBalanceAlertServiceIsolatesBalanceFetchFailures(t *testing.T) {
	repo := &supplierBalanceAlertRepoStub{
		configs: map[int64]SupplierBalanceAlertConfig{
			1: {ProviderID: 1, Enabled: true, Threshold: decimal.NewFromInt(10)},
			2: {ProviderID: 2, Enabled: true, Threshold: decimal.NewFromInt(10)},
		},
		active: make(map[int64]*SupplierBalanceAlertEvent),
		states: make(map[int64]supplierBalanceAlertState),
	}
	source := &supplierBalanceSourceStub{
		providers: []SupplierBalanceProvider{{ID: 1, Name: "失败供应商", LastSyncAt: supplierBalanceAlertTestSyncTime()}, {ID: 2, Name: "正常供应商", LastSyncAt: supplierBalanceAlertTestSyncTime()}},
		balances:  map[int64]decimal.Decimal{2: decimal.NewFromInt(1)},
		errors:    map[int64]error{1: errors.New("余额接口失败")},
	}
	dispatcher := &supplierBalanceDispatcherStub{}
	service := NewSupplierBalanceAlertService(repo, source, dispatcher)

	result, err := service.RunNow(context.Background())
	if err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}
	if result.Failed != 1 || result.Triggered != 1 {
		t.Fatalf("failed/triggered = %d/%d, want 1/1", result.Failed, result.Triggered)
	}
	if repo.states[1].status != SupplierBalanceAlertScanStatusError || repo.states[2].status != SupplierBalanceAlertScanStatusOK {
		t.Fatalf("scan states = %+v", repo.states)
	}
}

func TestSupplierBalanceAlertSourceUsesLocalBalanceWithoutRemote(t *testing.T) {
	syncedAt := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	repo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID:             7,
		Name:           "本地余额供应商",
		Code:           "local-balance",
		Enabled:        true,
		CurrentBalance: 7.25,
		LastSyncAt:     &syncedAt,
	}}}
	source := NewSupplierBalanceAlertSource(repo)

	providers, err := source.ListEnabledProviders(context.Background())
	if err != nil {
		t.Fatalf("ListEnabledProviders returned error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("providers length = %d, want 1", len(providers))
	}
	if providers[0].CurrentBalance != 7.25 {
		t.Fatalf("current balance = %v, want 7.25", providers[0].CurrentBalance)
	}
	if providers[0].LastSyncAt == nil || !providers[0].LastSyncAt.Equal(syncedAt) {
		t.Fatalf("last sync at = %v, want %v", providers[0].LastSyncAt, syncedAt)
	}

	balance, err := source.FetchBalance(context.Background(), providers[0])
	if err != nil {
		t.Fatalf("FetchBalance returned error: %v", err)
	}
	if !balance.Equal(decimal.RequireFromString("7.25")) {
		t.Fatalf("balance = %s, want 7.25", balance.String())
	}
}

func TestSupplierBalanceAlertServiceSkipsProviderWithoutLocalBalance(t *testing.T) {
	repo := &supplierBalanceAlertRepoStub{
		configs: map[int64]SupplierBalanceAlertConfig{
			7: {ProviderID: 7, Enabled: true, Threshold: decimal.NewFromInt(10)},
		},
		active: make(map[int64]*SupplierBalanceAlertEvent),
		states: make(map[int64]supplierBalanceAlertState),
	}
	source := &supplierBalanceSourceStub{
		providers: []SupplierBalanceProvider{{ID: 7, Name: "未同步供应商"}},
		balances:  map[int64]decimal.Decimal{7: decimal.Zero},
		errors:    make(map[int64]error),
	}
	service := NewSupplierBalanceAlertService(repo, source, nil)

	result, err := service.RunNow(context.Background())
	if err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}
	if result.Skipped != 1 || result.Failed != 0 || result.Triggered != 0 {
		t.Fatalf("skipped/failed/triggered = %d/%d/%d, want 1/0/0", result.Skipped, result.Failed, result.Triggered)
	}
	if repo.states[7].status != SupplierBalanceAlertScanStatusSkip {
		t.Fatalf("scan state status = %q, want %q", repo.states[7].status, SupplierBalanceAlertScanStatusSkip)
	}
	if repo.states[7].message != "暂无本地余额数据" {
		t.Fatalf("scan state message = %q, want 暂无本地余额数据", repo.states[7].message)
	}
}
