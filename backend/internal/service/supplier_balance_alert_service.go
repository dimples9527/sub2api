package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// SupplierBalanceAlertService 负责供应商余额阈值配置、扫描和事件状态变更。
// 它只依赖供应商管理暴露的余额数据源，不依赖 Ops 告警模块。
type SupplierBalanceAlertService struct {
	repo       SupplierBalanceAlertRepository
	source     SupplierBalanceSource
	dispatcher SupplierBalanceAlertDispatcher
	interval   time.Duration

	scanGuard chan struct{}
	lifeMu    sync.Mutex
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func NewSupplierBalanceAlertService(repo SupplierBalanceAlertRepository, source SupplierBalanceSource, dispatcher SupplierBalanceAlertDispatcher) *SupplierBalanceAlertService {
	return &SupplierBalanceAlertService{
		repo:       repo,
		source:     source,
		dispatcher: dispatcher,
		interval:   SupplierBalanceAlertDefaultInterval,
		scanGuard:  make(chan struct{}, 1),
	}
}

func (s *SupplierBalanceAlertService) SetInterval(interval time.Duration) {
	if s == nil || interval <= 0 {
		return
	}
	s.interval = interval
}

func (s *SupplierBalanceAlertService) ListConfigs(ctx context.Context, providerID int64) ([]SupplierBalanceAlertConfig, error) {
	if s == nil || s.repo == nil {
		return nil, ErrSupplierBalanceAlertInvalid
	}
	return s.repo.ListConfigs(ctx, providerID)
}

func (s *SupplierBalanceAlertService) GetConfig(ctx context.Context, providerID int64) (*SupplierBalanceAlertConfig, error) {
	if s == nil || s.repo == nil || providerID <= 0 {
		return nil, ErrSupplierBalanceAlertInvalid
	}
	return s.repo.GetConfig(ctx, providerID)
}

func (s *SupplierBalanceAlertService) UpdateConfig(ctx context.Context, providerID int64, input SupplierBalanceAlertConfigInput) (*SupplierBalanceAlertConfig, error) {
	if s == nil || s.repo == nil || providerID <= 0 {
		return nil, ErrSupplierBalanceAlertInvalid
	}
	threshold, err := decimal.NewFromString(input.Threshold)
	if err != nil || threshold.IsNegative() {
		return nil, ErrSupplierBalanceAlertInvalid
	}
	cooldown := input.CooldownSeconds
	if cooldown <= 0 {
		cooldown = int(SupplierBalanceAlertDefaultCooldown / time.Second)
	}
	if cooldown > 7*24*60*60 {
		return nil, ErrSupplierBalanceAlertInvalid
	}
	return s.repo.UpsertConfig(ctx, providerID, input.Enabled, threshold, cooldown)
}

func (s *SupplierBalanceAlertService) ListEvents(ctx context.Context, params SupplierBalanceAlertEventListParams) (SupplierBalanceAlertEventListResult, error) {
	if s == nil || s.repo == nil {
		return SupplierBalanceAlertEventListResult{}, ErrSupplierBalanceAlertInvalid
	}
	return s.repo.ListEvents(ctx, params)
}

// RunNow 执行一轮完整扫描。同一实例同时只允许一个扫描任务运行。
func (s *SupplierBalanceAlertService) RunNow(ctx context.Context) (SupplierBalanceAlertScanResult, error) {
	result := SupplierBalanceAlertScanResult{StartedAt: time.Now(), Providers: make([]SupplierBalanceAlertScanProviderResult, 0)}
	if s == nil || s.repo == nil || s.source == nil {
		return result, ErrSupplierBalanceAlertInvalid
	}
	select {
	case s.scanGuard <- struct{}{}:
		defer func() { <-s.scanGuard }()
	default:
		return result, ErrSupplierBalanceAlertScanBusy
	}

	providers, err := s.source.ListEnabledProviders(ctx)
	if err != nil {
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("查询启用供应商失败: %w", err)
	}
	for _, provider := range providers {
		item, _ := s.scanProvider(ctx, provider)
		result.Providers = append(result.Providers, item)
		result.Checked++
		switch item.Status {
		case SupplierBalanceAlertScanStatusSkip:
			result.Skipped++
		case SupplierBalanceAlertScanStatusError:
			result.Failed++
		}
		switch item.EventType {
		case SupplierBalanceAlertEventLow:
			result.Triggered++
		case SupplierBalanceAlertEventRecovered:
			result.Recovered++
		}
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (s *SupplierBalanceAlertService) scanProvider(ctx context.Context, provider SupplierBalanceProvider) (SupplierBalanceAlertScanProviderResult, error) {
	result := SupplierBalanceAlertScanProviderResult{ProviderID: provider.ID, ProviderName: provider.Name}
	config, err := s.repo.GetConfig(ctx, provider.ID)
	if err != nil {
		if errors.Is(err, ErrSupplierBalanceAlertConfigNotFound) {
			result.Status = SupplierBalanceAlertScanStatusSkip
			result.Message = "未配置余额预警"
			return result, nil
		}
		result.Status = SupplierBalanceAlertScanStatusError
		result.Message = err.Error()
		return result, err
	}
	if config == nil || !config.Enabled || config.Threshold.LessThanOrEqual(decimal.Zero) {
		result.Status = SupplierBalanceAlertScanStatusSkip
		result.Message = "余额预警未启用或阈值为 0"
		if config != nil {
			if stateErr := s.repo.UpdateScanState(ctx, provider.ID, time.Now(), nil, SupplierBalanceAlertScanStatusSkip, result.Message); stateErr != nil {
				result.Message = stateErr.Error()
			}
		}
		return result, nil
	}
	if provider.LastSyncAt == nil {
		result.Status = SupplierBalanceAlertScanStatusSkip
		result.Message = "暂无本地余额数据"
		if stateErr := s.repo.UpdateScanState(ctx, provider.ID, time.Now(), nil, SupplierBalanceAlertScanStatusSkip, result.Message); stateErr != nil {
			result.Message = stateErr.Error()
		}
		return result, nil
	}

	balance, err := s.source.FetchBalance(ctx, provider)
	if err != nil {
		result.Status = SupplierBalanceAlertScanStatusError
		result.Message = err.Error()
		_ = s.repo.UpdateScanState(ctx, provider.ID, time.Now(), nil, SupplierBalanceAlertScanStatusError, result.Message)
		return result, err
	}
	result.Balance = &balance
	now := time.Now()
	if err := s.repo.UpdateScanState(ctx, provider.ID, now, &balance, SupplierBalanceAlertScanStatusOK, ""); err != nil {
		result.Status = SupplierBalanceAlertScanStatusError
		result.Message = err.Error()
		return result, err
	}

	active, err := s.repo.GetActiveLowEvent(ctx, provider.ID)
	if err != nil {
		result.Status = SupplierBalanceAlertScanStatusError
		result.Message = err.Error()
		return result, err
	}
	isLow := balance.LessThan(config.Threshold)
	if isLow {
		result.Status = SupplierBalanceAlertScanStatusOK
		result.EventType = SupplierBalanceAlertEventLow
		event := SupplierBalanceAlertEvent{
			ProviderID:   provider.ID,
			ProviderCode: firstNonEmptySupplierText(provider.Code, config.ProviderCode),
			ProviderName: firstNonEmptySupplierText(provider.Name, config.ProviderName),
			EventType:    SupplierBalanceAlertEventLow,
			Status:       SupplierBalanceAlertEventActive,
			Balance:      balance,
			Threshold:    config.Threshold,
			Cooldown:     supplierBalanceAlertCooldown(config),
			ObservedAt:   now,
			LastSeenAt:   now,
		}
		if active != nil {
			if err := s.repo.TouchActiveLowEvent(ctx, active.ID, balance, now); err != nil {
				result.Status = SupplierBalanceAlertScanStatusError
				result.Message = err.Error()
				return result, err
			}
			event.ID = active.ID
		} else {
			if err := s.repo.CreateEvent(ctx, &event); err != nil {
				result.Status = SupplierBalanceAlertScanStatusError
				result.Message = err.Error()
				return result, err
			}
		}
		if s.dispatcher != nil {
			if dispatchErr := s.dispatcher.Dispatch(ctx, event); dispatchErr != nil {
				result.Message = dispatchErr.Error()
			}
		}
		return result, nil
	}

	if active == nil {
		result.Status = SupplierBalanceAlertScanStatusOK
		return result, nil
	}
	if err := s.repo.ResolveActiveLowEvent(ctx, active.ID, now, balance); err != nil {
		result.Status = SupplierBalanceAlertScanStatusError
		result.Message = err.Error()
		return result, err
	}
	recovered := SupplierBalanceAlertEvent{
		ProviderID:   provider.ID,
		ProviderCode: firstNonEmptySupplierText(provider.Code, config.ProviderCode),
		ProviderName: firstNonEmptySupplierText(provider.Name, config.ProviderName),
		EventType:    SupplierBalanceAlertEventRecovered,
		Status:       SupplierBalanceAlertEventResolved,
		Balance:      balance,
		Threshold:    config.Threshold,
		Cooldown:     supplierBalanceAlertCooldown(config),
		ObservedAt:   now,
		ResolvedAt:   &now,
		LastSeenAt:   now,
	}
	if err := s.repo.CreateEvent(ctx, &recovered); err != nil {
		result.Status = SupplierBalanceAlertScanStatusError
		result.Message = err.Error()
		return result, err
	}
	result.Status = SupplierBalanceAlertScanStatusOK
	result.EventType = SupplierBalanceAlertEventRecovered
	if s.dispatcher != nil {
		if dispatchErr := s.dispatcher.Dispatch(ctx, recovered); dispatchErr != nil {
			result.Message = dispatchErr.Error()
		}
	}
	return result, nil
}

func supplierBalanceAlertCooldown(config *SupplierBalanceAlertConfig) time.Duration {
	if config == nil || config.CooldownSeconds <= 0 {
		return SupplierBalanceAlertDefaultCooldown
	}
	return time.Duration(config.CooldownSeconds) * time.Second
}

func firstNonEmptySupplierText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "供应商"
}

func (s *SupplierBalanceAlertService) Start() {
	if s == nil || s.interval <= 0 {
		return
	}
	s.lifeMu.Lock()
	if s.stopCh != nil {
		s.lifeMu.Unlock()
		return
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	s.stopCh = stopCh
	s.doneCh = doneCh
	interval := s.interval
	s.lifeMu.Unlock()

	go func() {
		defer close(doneCh)
		_, _ = s.RunNow(context.Background())
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, _ = s.RunNow(context.Background())
			case <-stopCh:
				return
			}
		}
	}()
}

func (s *SupplierBalanceAlertService) Stop() {
	if s == nil {
		return
	}
	s.lifeMu.Lock()
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.stopCh = nil
	s.doneCh = nil
	s.lifeMu.Unlock()
	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh
	}
}

// NewSupplierBalanceAlertSource 将供应商管理仓储中的本地余额数据适配为预警模块自己的数据源。
func NewSupplierBalanceAlertSource(providerRepo SupplierProviderRepository) SupplierBalanceSource {
	return &supplierBalanceAlertSource{providerRepo: providerRepo}
}

type supplierBalanceAlertSource struct {
	providerRepo SupplierProviderRepository
}

func (s *supplierBalanceAlertSource) ListEnabledProviders(ctx context.Context) ([]SupplierBalanceProvider, error) {
	if s == nil || s.providerRepo == nil {
		return nil, ErrSupplierBalanceAlertInvalid
	}
	enabled := true
	params := SupplierProviderListParams{Enabled: &enabled, Page: 1, PageSize: 200}
	result := make([]SupplierBalanceProvider, 0)
	for {
		providers, total, err := s.providerRepo.List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("查询启用供应商失败: %w", err)
		}
		for _, provider := range providers {
			if provider == nil {
				continue
			}
			result = append(result, supplierBalanceProviderFromModel(provider))
		}
		if len(result) >= int(total) || len(providers) == 0 {
			break
		}
		params.Page++
	}
	return result, nil
}

func (s *supplierBalanceAlertSource) FetchBalance(_ context.Context, provider SupplierBalanceProvider) (decimal.Decimal, error) {
	if s == nil {
		return decimal.Zero, ErrSupplierBalanceAlertInvalid
	}
	if provider.LastSyncAt == nil {
		return decimal.Zero, fmt.Errorf("暂无本地余额数据")
	}
	value := provider.CurrentBalance
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return decimal.Zero, fmt.Errorf("供应商余额不是有效数字")
	}
	return decimal.NewFromFloat(value), nil
}

func supplierBalanceProviderFromModel(provider *SupplierProvider) SupplierBalanceProvider {
	return SupplierBalanceProvider{
		ID: provider.ID, Code: provider.Code, Name: provider.Name, ProviderType: provider.ProviderType,
		BaseURL: provider.BaseURL, LoginURL: provider.LoginURL, APIKeysURL: provider.APIKeysURL,
		GroupsURL: provider.GroupsURL, AvailableGroupsURL: provider.AvailableGroupsURL,
		BalanceURL: provider.BalanceURL, UsageCostURL: provider.UsageCostURL,
		Username: provider.Username, Email: provider.Email, Enabled: provider.Enabled,
		TurnstileEnabled: provider.TurnstileEnabled, CurrentBalance: provider.CurrentBalance,
		LastSyncAt: provider.LastSyncAt,
	}
}
