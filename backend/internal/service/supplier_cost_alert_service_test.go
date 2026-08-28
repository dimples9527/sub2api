package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type supplierCostAlertRepoStub struct {
	settings       *SupplierCostAlertSettings
	override       *SupplierCostAlertOverride
	active         *SupplierCostAlertEvent
	events         []SupplierCostAlertEvent
	nextID         int64
	createCalls    int
	touchCalls     int
	resolveCalls   int
	getActiveCalls int
	createErr      error
}

func newSupplierCostAlertRepoStub() *supplierCostAlertRepoStub {
	return &supplierCostAlertRepoStub{nextID: 101}
}

func (r *supplierCostAlertRepoStub) GetSettings(context.Context) (*SupplierCostAlertSettings, error) {
	return r.settings, nil
}
func (r *supplierCostAlertRepoStub) UpdateSettings(_ context.Context, amount decimal.Decimal) (*SupplierCostAlertSettings, error) {
	r.settings = &SupplierCostAlertSettings{Amount: amount}
	return r.settings, nil
}
func (r *supplierCostAlertRepoStub) GetOverrideByProvider(context.Context, int64) (*SupplierCostAlertOverride, error) {
	return r.override, nil
}
func (r *supplierCostAlertRepoStub) ListOverrides(context.Context) ([]SupplierCostAlertOverride, error) {
	if r.override == nil {
		return nil, nil
	}
	return []SupplierCostAlertOverride{*r.override}, nil
}
func (r *supplierCostAlertRepoStub) UpsertOverride(_ context.Context, override *SupplierCostAlertOverride) (*SupplierCostAlertOverride, error) {
	if override.ID == 0 {
		r.nextID++
		override.ID = r.nextID
		now := time.Now()
		override.CreatedAt = now
	}
	override.UpdatedAt = time.Now()
	r.override = override
	return override, nil
}
func (*supplierCostAlertRepoStub) DeleteOverride(context.Context, int64) error { return nil }
func (r *supplierCostAlertRepoStub) GetActiveOverrunEvent(context.Context, int64) (*SupplierCostAlertEvent, error) {
	r.getActiveCalls++
	return r.active, nil
}
func (r *supplierCostAlertRepoStub) CreateEvent(_ context.Context, event *SupplierCostAlertEvent) error {
	r.createCalls++
	r.nextID++
	event.ID = r.nextID
	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now
	r.events = append(r.events, *event)
	return r.createErr
}
func (r *supplierCostAlertRepoStub) TouchActiveOverrunEvent(context.Context, int64, SupplierCostAlertEvent) error {
	r.touchCalls++
	r.active = nil
	return nil
}
func (r *supplierCostAlertRepoStub) ResolveActiveOverrunEvent(context.Context, int64, time.Time) error {
	r.resolveCalls++
	r.active = nil
	return nil
}
func (r *supplierCostAlertRepoStub) ListEvents(context.Context, SupplierCostAlertEventListParams) (SupplierCostAlertEventListResult, error) {
	return SupplierCostAlertEventListResult{Items: r.events}, nil
}

type supplierCostAlertDispatcherStub struct {
	events []SupplierBalanceAlertEvent
}

func (s *supplierCostAlertDispatcherStub) Dispatch(_ context.Context, event SupplierBalanceAlertEvent) error {
	s.events = append(s.events, event)
	return nil
}

func TestEffectiveSupplierCostAlertAmount(t *testing.T) {
	require.False(t, EffectiveSupplierCostAlertAmount(nil, nil).IsPositive())
	global := &SupplierCostAlertSettings{Amount: decimal.NewFromInt(10)}
	require.Equal(t, "10", EffectiveSupplierCostAlertAmount(global, nil).String())
	disabled := decimal.NewFromInt(0)
	overrideZero := &SupplierCostAlertOverride{Enabled: true, Amount: disabled}
	require.Equal(t, "10", EffectiveSupplierCostAlertAmount(global, overrideZero).String())
	override := &SupplierCostAlertOverride{Enabled: true, Amount: decimal.NewFromInt(3)}
	require.Equal(t, "3", EffectiveSupplierCostAlertAmount(global, override).String())
	off := &SupplierCostAlertOverride{Enabled: false, Amount: decimal.NewFromInt(3)}
	require.Equal(t, "10", EffectiveSupplierCostAlertAmount(global, off).String())
}

func TestSupplierCostAlertServiceEvaluateCreatesAndDispatchesOverrun(t *testing.T) {
	repo := newSupplierCostAlertRepoStub()
	repo.settings = &SupplierCostAlertSettings{Amount: decimal.NewFromInt(10)}
	dispatcher := &supplierCostAlertDispatcherStub{}
	service := NewSupplierCostAlertService(repo, dispatcher)
	statDay := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	err := service.Evaluate(context.Background(), SupplierCostAlertEvaluation{
		ProviderID:   8,
		ProviderCode: "alpha",
		ProviderName: "供应商甲",
		StatDate:     statDay,
		UpstreamCost: 120,
		LocalCost:    100,
	})
	require.NoError(t, err)
	require.Equal(t, 1, repo.createCalls)
	require.Len(t, dispatcher.events, 1)
	require.Equal(t, SupplierCostAlertEventOverrun, dispatcher.events[0].EventType)
	require.Equal(t, SupplierCostAlertEventActive, dispatcher.events[0].Status)
	require.True(t, dispatcher.events[0].Balance.Equal(decimal.NewFromInt(20)))
	require.Equal(t, decimal.NewFromInt(10), dispatcher.events[0].Threshold)
	require.True(t, dispatcher.events[0].Balance.Equal(decimal.NewFromInt(20)))
	require.True(t, repo.events[0].OverrunAmount.Equal(decimal.NewFromInt(20)))
}

func TestSupplierCostAlertServiceEvaluateTouchesSameActiveOverrun(t *testing.T) {
	repo := newSupplierCostAlertRepoStub()
	repo.settings = &SupplierCostAlertSettings{Amount: decimal.NewFromInt(10)}
	repo.active = &SupplierCostAlertEvent{ID: 7, ProviderID: 8, EventType: SupplierCostAlertEventOverrun, Status: SupplierCostAlertEventActive}
	service := NewSupplierCostAlertService(repo, &supplierCostAlertDispatcherStub{})
	err := service.Evaluate(context.Background(), SupplierCostAlertEvaluation{
		ProviderID: 8, StatDate: time.Now().UTC(), UpstreamCost: 130, LocalCost: 100,
	})
	require.NoError(t, err)
	require.Equal(t, 1, repo.touchCalls)
	require.Zero(t, repo.createCalls)
	require.Nil(t, repo.active)
}

func TestSupplierCostAlertServiceEvaluateResolvesRecoveredOverrun(t *testing.T) {
	repo := newSupplierCostAlertRepoStub()
	repo.settings = &SupplierCostAlertSettings{Amount: decimal.NewFromInt(10)}
	repo.active = &SupplierCostAlertEvent{ID: 7, ProviderID: 8, EventType: SupplierCostAlertEventOverrun, Status: SupplierCostAlertEventActive}
	dispatcher := &supplierCostAlertDispatcherStub{}
	service := NewSupplierCostAlertService(repo, dispatcher)
	err := service.Evaluate(context.Background(), SupplierCostAlertEvaluation{
		ProviderID: 8, StatDate: time.Now().UTC(), UpstreamCost: 105, LocalCost: 100,
	})
	require.NoError(t, err)
	require.Equal(t, 1, repo.resolveCalls)
	require.Equal(t, 1, repo.createCalls)
	require.Equal(t, SupplierCostAlertEventRecovered, dispatcher.events[0].EventType)
	require.Equal(t, decimal.NewFromInt(5), dispatcher.events[0].Balance)
	require.Nil(t, repo.active)
}

func TestSupplierCostAlertServiceEvaluateSkipsDisabledOrReverseGap(t *testing.T) {
	repo := newSupplierCostAlertRepoStub()
	repo.settings = &SupplierCostAlertSettings{Amount: decimal.NewFromInt(10)}
	service := NewSupplierCostAlertService(repo, &supplierCostAlertDispatcherStub{})
	err := service.Evaluate(context.Background(), SupplierCostAlertEvaluation{
		ProviderID: 8, StatDate: time.Now().UTC(), UpstreamCost: 105, LocalCost: 110,
	})
	require.NoError(t, err)
	require.Zero(t, repo.createCalls)
	repo.settings = nil
	err = service.Evaluate(context.Background(), SupplierCostAlertEvaluation{
		ProviderID: 8, StatDate: time.Now().UTC(), UpstreamCost: 130, LocalCost: 100,
	})
	require.NoError(t, err)
	require.Zero(t, repo.createCalls)
}

func TestSupplierCostAlertServiceEvaluateReturnsInvalidInput(t *testing.T) {
	repo := newSupplierCostAlertRepoStub()
	service := NewSupplierCostAlertService(repo, &supplierCostAlertDispatcherStub{})
	err := service.Evaluate(context.Background(), SupplierCostAlertEvaluation{
		ProviderID: 0, StatDate: time.Now().UTC(), UpstreamCost: 130, LocalCost: 100,
	})
	require.ErrorIs(t, err, ErrSupplierCostAlertInvalid)
	require.Zero(t, repo.getActiveCalls)
}

func TestSupplierCostAlertServiceConfigValidation(t *testing.T) {
	repo := newSupplierCostAlertRepoStub()
	repo.settings = &SupplierCostAlertSettings{Amount: decimal.NewFromInt(15)}
	service := NewSupplierCostAlertService(repo, &supplierCostAlertDispatcherStub{})
	updated, err := service.UpdateSettings(context.Background(), decimal.NewFromInt(20))
	require.NoError(t, err)
	require.Equal(t, decimal.NewFromInt(20), updated.Amount)
	_, err = service.UpdateSettings(context.Background(), decimal.NewFromInt(-1))
	require.ErrorIs(t, err, ErrSupplierCostAlertInvalid)
	override, err := service.UpsertOverride(context.Background(), SupplierCostAlertOverrideInput{
		ProviderID: 8, Enabled: true, Amount: "2",
	})
	require.NoError(t, err)
	require.Equal(t, int64(8), override.ProviderID)
	_, err = service.UpsertOverride(context.Background(), SupplierCostAlertOverrideInput{
		ProviderID: 0, Enabled: true, Amount: "2",
	})
	require.ErrorIs(t, err, ErrSupplierCostAlertInvalid)
}

func TestSupplierCostAlertServiceEvaluateKeepsSyncFailureIndependentThroughError(t *testing.T) {
	repo := newSupplierCostAlertRepoStub()
	repo.settings = &SupplierCostAlertSettings{Amount: decimal.NewFromInt(10)}
	repo.createErr = errors.New("database unavailable")
	dispatcher := &supplierCostAlertDispatcherStub{}
	service := NewSupplierCostAlertService(repo, dispatcher)
	err := service.Evaluate(context.Background(), SupplierCostAlertEvaluation{
		ProviderID: 8, StatDate: time.Now().UTC(), UpstreamCost: 120, LocalCost: 100,
	})
	require.ErrorContains(t, err, "创建供应商成本超额预警事件失败")
}
