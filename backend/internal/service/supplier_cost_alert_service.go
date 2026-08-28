package service

import (
	"context"
	"errors"
	"fmt"

	"time"

	"github.com/shopspring/decimal"
)

// SupplierCostAlertService 负责成本超额预警的阈值选择、事件生命周期与通知派发。
type SupplierCostAlertService struct {
	repo       SupplierCostAlertRepository
	dispatcher SupplierBalanceAlertDispatcher
}

func NewSupplierCostAlertService(repo SupplierCostAlertRepository, dispatcher SupplierBalanceAlertDispatcher) *SupplierCostAlertService {
	return &SupplierCostAlertService{repo: repo, dispatcher: dispatcher}
}

// DefaultSupplierCostAlertSettings 返回未配置时的全局默认配置。
func DefaultSupplierCostAlertSettings() *SupplierCostAlertSettings {
	return &SupplierCostAlertSettings{Amount: decimal.Zero}
}

// EffectiveSupplierCostAlertAmount 根据全局配置和供应商覆盖配置选择实际生效阈值。
// 覆盖配置被禁用或阈值为 0 时回退到全局阈值。
func EffectiveSupplierCostAlertAmount(settings *SupplierCostAlertSettings, override *SupplierCostAlertOverride) decimal.Decimal {
	if override != nil && override.Enabled && override.Amount.IsPositive() {
		return override.Amount
	}
	if settings == nil || !settings.Amount.IsPositive() {
		return decimal.Zero
	}
	return settings.Amount
}

func (s *SupplierCostAlertService) GetSettings(ctx context.Context) (*SupplierCostAlertSettings, error) {
	if s == nil || s.repo == nil {
		return nil, ErrSupplierCostAlertInvalid
	}
	settings, err := s.repo.GetSettings(ctx)
	if err != nil && !errors.Is(err, ErrSupplierCostAlertConfigNotFound) {
		return nil, err
	}
	if settings == nil {
		return DefaultSupplierCostAlertSettings(), nil
	}
	return settings, nil
}

func (s *SupplierCostAlertService) UpdateSettings(ctx context.Context, amount decimal.Decimal) (*SupplierCostAlertSettings, error) {
	if s == nil || s.repo == nil || amount.IsNegative() {
		return nil, ErrSupplierCostAlertInvalid
	}
	if err := validateSupplierCostAlertAmount(amount); err != nil {
		return nil, ErrSupplierCostAlertInvalid
	}
	settings, err := s.repo.UpdateSettings(ctx, amount)
	if err != nil {
		return nil, fmt.Errorf("更新供应商成本超额预警全局配置失败: %w", err)
	}
	return settings, nil
}

func (s *SupplierCostAlertService) ListOverrides(ctx context.Context) ([]SupplierCostAlertOverride, error) {
	if s == nil || s.repo == nil {
		return nil, ErrSupplierCostAlertInvalid
	}
	return s.repo.ListOverrides(ctx)
}

func (s *SupplierCostAlertService) UpsertOverride(ctx context.Context, input SupplierCostAlertOverrideInput) (*SupplierCostAlertOverride, error) {
	if s == nil || s.repo == nil || input.ProviderID <= 0 {
		return nil, ErrSupplierCostAlertInvalid
	}
	amount, err := decimal.NewFromString(input.Amount)
	if err != nil || amount.IsNegative() {
		return nil, ErrSupplierCostAlertInvalid
	}
	if err := validateSupplierCostAlertAmount(amount); err != nil {
		return nil, ErrSupplierCostAlertInvalid
	}
	override := &SupplierCostAlertOverride{
		ProviderID: input.ProviderID,
		Enabled:    input.Enabled,
		Amount:     amount,
	}
	created, err := s.repo.UpsertOverride(ctx, override)
	if err != nil {
		return nil, fmt.Errorf("保存供应商成本超额预警覆盖配置失败: %w", err)
	}
	return created, nil
}

func (s *SupplierCostAlertService) DeleteOverride(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil || id <= 0 {
		return ErrSupplierCostAlertInvalid
	}
	return s.repo.DeleteOverride(ctx, id)
}

func (s *SupplierCostAlertService) ListEvents(ctx context.Context, params SupplierCostAlertEventListParams) (SupplierCostAlertEventListResult, error) {
	if s == nil || s.repo == nil {
		return SupplierCostAlertEventListResult{}, ErrSupplierCostAlertInvalid
	}
	return s.repo.ListEvents(ctx, params)
}

// Evaluate 在成本同步成功落库后执行一次预警检查；返回的错误不应导致同步主流程失败，
// 因为预警属于旁路能力。调用方按自身需要记录错误即可。
func (s *SupplierCostAlertService) Evaluate(ctx context.Context, evaluation SupplierCostAlertEvaluation) error {
	if s == nil || s.repo == nil {
		return ErrSupplierCostAlertInvalid
	}
	if evaluation.ProviderID <= 0 || evaluation.StatDate.IsZero() || evaluation.LocalCost < 0 {
		return ErrSupplierCostAlertInvalid
	}
	if evaluation.ObservedAt.IsZero() {
		evaluation.ObservedAt = time.Now()
	}
	upstream := decimal.NewFromFloat(evaluation.UpstreamCost)
	local := decimal.NewFromFloat(evaluation.LocalCost)
	overrun := upstream.Sub(local)
	if overrun.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	threshold, err := s.effectiveThreshold(ctx, evaluation.ProviderID)
	if err != nil {
		return err
	}
	if !threshold.IsPositive() {
		return nil
	}
	if overrun.GreaterThan(threshold) {
		return s.recordOverrun(ctx, evaluation, overrun, threshold)
	}
	return s.recordRecovered(ctx, evaluation, overrun, threshold)
}

func (s *SupplierCostAlertService) effectiveThreshold(ctx context.Context, providerID int64) (decimal.Decimal, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	override, err := s.repo.GetOverrideByProvider(ctx, providerID)
	if err != nil && !errors.Is(err, ErrSupplierCostAlertConfigNotFound) {
		return decimal.Zero, err
	}
	return EffectiveSupplierCostAlertAmount(settings, override), nil
}

func (s *SupplierCostAlertService) recordOverrun(ctx context.Context, evaluation SupplierCostAlertEvaluation, overrun, threshold decimal.Decimal) error {
	active, err := s.repo.GetActiveOverrunEvent(ctx, evaluation.ProviderID)
	if err != nil && !errors.Is(err, ErrSupplierCostAlertEventNotFound) {
		return err
	}
	now := evaluation.ObservedAt
	event := SupplierCostAlertEvent{
		ProviderID:    evaluation.ProviderID,
		ProviderCode:  evaluation.ProviderCode,
		ProviderName:  evaluation.ProviderName,
		EventType:     SupplierCostAlertEventOverrun,
		Status:        SupplierCostAlertEventActive,
		StatDate:      evaluation.StatDate,
		UpstreamCost:  decimal.NewFromFloat(evaluation.UpstreamCost),
		LocalCost:     decimal.NewFromFloat(evaluation.LocalCost),
		OverrunAmount: overrun,
		Threshold:     threshold,
		ObservedAt:    now,
		LastSeenAt:    now,
	}
	if active != nil {
		return s.repo.TouchActiveOverrunEvent(ctx, active.ID, event)
	}
	if err := s.repo.CreateEvent(ctx, &event); err != nil {
		return fmt.Errorf("创建供应商成本超额预警事件失败: %w", err)
	}
	s.dispatch(ctx, event)
	return nil
}

func (s *SupplierCostAlertService) recordRecovered(ctx context.Context, evaluation SupplierCostAlertEvaluation, overrun, threshold decimal.Decimal) error {
	active, err := s.repo.GetActiveOverrunEvent(ctx, evaluation.ProviderID)
	if err != nil {
		if errors.Is(err, ErrSupplierCostAlertEventNotFound) {
			return nil
		}
		return err
	}
	if active == nil || active.ID <= 0 {
		return nil
	}
	now := evaluation.ObservedAt
	if err := s.repo.ResolveActiveOverrunEvent(ctx, active.ID, now); err != nil {
		return fmt.Errorf("恢复供应商成本超额预警事件失败: %w", err)
	}
	recovered := SupplierCostAlertEvent{
		ProviderID:    evaluation.ProviderID,
		ProviderCode:  evaluation.ProviderCode,
		ProviderName:  evaluation.ProviderName,
		EventType:     SupplierCostAlertEventRecovered,
		Status:        SupplierCostAlertEventResolved,
		StatDate:      evaluation.StatDate,
		UpstreamCost:  decimal.NewFromFloat(evaluation.UpstreamCost),
		LocalCost:     decimal.NewFromFloat(evaluation.LocalCost),
		OverrunAmount: overrun,
		Threshold:     threshold,
		ObservedAt:    now,
		ResolvedAt:    &now,
		LastSeenAt:    now,
	}
	if err := s.repo.CreateEvent(ctx, &recovered); err != nil {
		return fmt.Errorf("创建供应商成本恢复事件失败: %w", err)
	}
	s.dispatch(ctx, recovered)
	return nil
}

func (s *SupplierCostAlertService) dispatch(ctx context.Context, event SupplierCostAlertEvent) {
	if s.dispatcher == nil {
		return
	}
	_ = s.dispatcher.Dispatch(ctx, SupplierBalanceAlertEvent{
		ProviderID:   event.ProviderID,
		ProviderCode: event.ProviderCode,
		ProviderName: event.ProviderName,
		EventType:    event.EventType,
		Status:       event.Status,
		Balance:      event.OverrunAmount,
		Threshold:    event.Threshold,
		ObservedAt:   event.ObservedAt,
		ResolvedAt:   event.ResolvedAt,
		LastSeenAt:   event.LastSeenAt,
	})
}

func validateSupplierCostAlertAmount(value decimal.Decimal) error {
	if value.IsNegative() {
		return ErrSupplierCostAlertInvalid
	}
	return nil
}
