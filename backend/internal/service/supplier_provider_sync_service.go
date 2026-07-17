package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

var ErrSupplierProviderSyncConflict = infraerrors.Conflict("SUPPLIER_PROVIDER_SYNC_CONFLICT", "supplier provider sync already running")

type SupplierProviderAccount struct {
	ID             int64      `json:"id"`
	ProviderID     int64      `json:"provider_id"`
	ProviderName   string     `json:"provider_name"`
	UpstreamKey    string     `json:"upstream_account_key"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	GroupKey       string     `json:"group_key"`
	GroupName      string     `json:"group_name"`
	RateMultiplier float64    `json:"rate_multiplier"`
	RawStatus      string     `json:"raw_status"`
	Active         bool       `json:"active"`
	LastSeenAt     time.Time  `json:"last_seen_at"`
	InactiveAt     *time.Time `json:"inactive_at,omitempty"`
}

type SupplierProviderGroup struct {
	ID             int64      `json:"id"`
	ProviderID     int64      `json:"provider_id"`
	ProviderName   string     `json:"provider_name"`
	UpstreamKey    string     `json:"upstream_group_key"`
	Name           string     `json:"name"`
	RateMultiplier float64    `json:"rate_multiplier"`
	RawStatus      string     `json:"raw_status"`
	Active         bool       `json:"active"`
	AccountCount   int        `json:"account_count"`
	LastSeenAt     time.Time  `json:"last_seen_at"`
	InactiveAt     *time.Time `json:"inactive_at,omitempty"`
}

type SupplierProviderDataListParams struct {
	ProviderID int64
	Active     *bool
	Search     string
	Page       int
	PageSize   int
}

type SupplierProviderAccountListResult struct {
	Items    []SupplierProviderAccount `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

type SupplierProviderGroupListResult struct {
	Items    []SupplierProviderGroup `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type SupplierSyncCounts struct {
	CheckedCount int `json:"checked_count"`
	CreatedCount int `json:"created_count"`
	UpdatedCount int `json:"updated_count"`
	SkippedCount int `json:"skipped_count"`
}

type SupplierProviderSyncRun struct {
	ID            int64
	ProviderID    int64
	SyncScope     string
	TriggerSource string
	Status        string
	Counts        SupplierSyncCounts
	ErrorMessage  string
	StartedAt     time.Time
	FinishedAt    *time.Time
}

type SupplierCleanupPolicy struct {
	AutomationRunRetentionDays int
	SyncRunRetentionDays       int
	MetricRetentionDays        int
	DailyStatRetentionDays     int
	InactiveAccountDays        int
	InactiveGroupDays          int
}

type SupplierCleanupCounts struct {
	AutomationRuns  int
	SyncRuns        int
	MetricSnapshots int
	DailyStats      int
	Accounts        int
	Groups          int
}

type SupplierProviderDataRepository interface {
	ListAccounts(ctx context.Context, params SupplierProviderDataListParams) (SupplierProviderAccountListResult, error)
	ListGroups(ctx context.Context, params SupplierProviderDataListParams) (SupplierProviderGroupListResult, error)
	ReplaceAccounts(ctx context.Context, providerID int64, items []SupplierProviderRemoteAccount, seenAt time.Time) (SupplierSyncCounts, error)
	ReplaceGroups(ctx context.Context, providerID int64, items []SupplierProviderRemoteGroup, seenAt time.Time) (SupplierSyncCounts, error)
	UpdateBalance(ctx context.Context, providerID int64, balance float64, seenAt time.Time) error
	UpdateCost(ctx context.Context, providerID int64, cost float64, seenAt time.Time) error
	CreateSyncRun(ctx context.Context, run *SupplierProviderSyncRun) error
	FinishSyncRun(ctx context.Context, run *SupplierProviderSyncRun) error
	UpdateSyncStatus(ctx context.Context, providerID int64, status, message string, syncedAt time.Time) error
	Cleanup(ctx context.Context, policy SupplierCleanupPolicy, now time.Time, batchSize int) (SupplierCleanupCounts, error)
}

type SupplierProviderSyncResult struct {
	ProviderID   int64                       `json:"provider_id"`
	ProviderName string                      `json:"provider_name"`
	Scope        string                      `json:"scope"`
	Status       string                      `json:"status"`
	Message      string                      `json:"message"`
	Counts       SupplierSyncCounts          `json:"counts"`
	Stages       []SupplierProviderSyncStage `json:"stages,omitempty"`
	StartedAt    time.Time                   `json:"started_at"`
	FinishedAt   time.Time                   `json:"finished_at"`
}

type SupplierProviderSyncStage struct {
	Scope          string                          `json:"scope"`
	Status         string                          `json:"status"`
	Message        string                          `json:"message"`
	Counts         SupplierSyncCounts              `json:"counts"`
	EndpointResult *SupplierProviderEndpointResult `json:"endpoint_result,omitempty"`
}

type SupplierProviderEndpointResult struct {
	Endpoint        string `json:"endpoint"`
	HTTPStatus      int    `json:"http_status"`
	DurationMS      int64  `json:"duration_ms"`
	ResponseBytes   int    `json:"response_bytes"`
	ResponseSummary string `json:"response_summary"`
	ParsedSummary   string `json:"parsed_summary,omitempty"`
	ParseError      string `json:"parse_error,omitempty"`
	Error           string `json:"error,omitempty"`
}

type SupplierProviderBatchSyncResult struct {
	ProcessedCount int                          `json:"processed_count"`
	SuccessCount   int                          `json:"success_count"`
	FailedCount    int                          `json:"failed_count"`
	Results        []SupplierProviderSyncResult `json:"results"`
}

type SupplierProviderEndpointTestAttempt struct {
	Endpoint        string `json:"endpoint"`
	HTTPStatus      int    `json:"http_status"`
	DurationMS      int64  `json:"duration_ms"`
	ResponseBytes   int    `json:"response_bytes"`
	ResponseSummary string `json:"response_summary"`
	ParsedData      any    `json:"parsed_data,omitempty"`
	ParseError      string `json:"parse_error,omitempty"`
	Error           string `json:"error,omitempty"`
}

type SupplierProviderEndpointTestResult struct {
	ProviderID        int64                                 `json:"provider_id"`
	Scope             string                                `json:"scope"`
	Endpoint          string                                `json:"endpoint"`
	HTTPStatus        int                                   `json:"http_status"`
	DurationMS        int64                                 `json:"duration_ms"`
	ResponseBytes     int                                   `json:"response_bytes"`
	ResponseSummary   string                                `json:"response_summary"`
	ParsedData        any                                   `json:"parsed_data,omitempty"`
	ParseError        string                                `json:"parse_error,omitempty"`
	Error             string                                `json:"error,omitempty"`
	Attempts          []SupplierProviderEndpointTestAttempt `json:"attempts"`
	SensitiveRedacted bool                                  `json:"sensitive_redacted"`
}

const (
	SupplierSyncTriggerManual    = "manual"
	SupplierSyncTriggerScheduled = "scheduled"
	SupplierSyncStatusRunning    = "running"
	SupplierSyncStatusSuccess    = "success"
	SupplierSyncStatusPartial    = "partial"
	SupplierSyncStatusFailed     = "failed"

	SupplierSyncScopeAccounts = "accounts"
	SupplierSyncScopeGroups   = "groups"
	SupplierSyncScopeBalance  = "balance"
	SupplierSyncScopeCost     = "cost"
	SupplierSyncScopeAll      = "all"
)

type SupplierProviderSyncService struct {
	providerRepo SupplierProviderRepository
	dataRepo     SupplierProviderDataRepository
	remote       SupplierProviderRemoteClient
	encryptor    SecretEncryptor
	syncLock     SupplierProviderSyncLock
}

func NewSupplierProviderSyncService(providerRepo SupplierProviderRepository, dataRepo SupplierProviderDataRepository, remote SupplierProviderRemoteClient, encryptor SecretEncryptor, syncLock SupplierProviderSyncLock) *SupplierProviderSyncService {
	return &SupplierProviderSyncService{
		providerRepo: providerRepo,
		dataRepo:     dataRepo,
		remote:       remote,
		encryptor:    encryptor,
		syncLock:     syncLock,
	}
}

func (s *SupplierProviderSyncService) providerPassword(provider *SupplierProvider) string {
	stored := strings.TrimSpace(provider.PasswordEncrypted)
	if stored == "" || s.encryptor == nil {
		return stored
	}
	password, err := s.encryptor.Decrypt(stored)
	if err != nil {
		return stored
	}
	return password
}

func (s *SupplierProviderSyncService) SyncAccounts(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncStage(ctx, provider, password, SupplierSyncScopeAccounts, trigger, true)
	})
}

func (s *SupplierProviderSyncService) SyncGroups(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncStage(ctx, provider, password, SupplierSyncScopeGroups, trigger, true)
	})
}

func (s *SupplierProviderSyncService) SyncBalance(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncStage(ctx, provider, password, SupplierSyncScopeBalance, trigger, true)
	})
}

func (s *SupplierProviderSyncService) SyncCost(ctx context.Context, providerID int64, day time.Time, trigger string) (SupplierProviderSyncResult, error) {
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncCostStage(ctx, provider, password, day, trigger, true)
	})
}

func (s *SupplierProviderSyncService) TestEndpoint(ctx context.Context, providerID int64, scope string) (SupplierProviderEndpointTestResult, error) {
	provider, err := s.validSyncProvider(ctx, providerID)
	if err != nil {
		return SupplierProviderEndpointTestResult{}, err
	}
	password := s.providerPassword(provider)
	tester, ok := s.remote.(SupplierProviderRemoteTester)
	if !ok {
		return SupplierProviderEndpointTestResult{}, fmt.Errorf("supplier provider remote client does not support endpoint test")
	}
	result, err := tester.TestEndpoint(ctx, provider, password, scope)
	if err != nil {
		return SupplierProviderEndpointTestResult{}, err
	}
	result.ProviderID = provider.ID
	result.Scope = scope
	result.SensitiveRedacted = true
	return result, nil
}

func (s *SupplierProviderSyncService) SyncAll(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		startedAt := time.Now()
		run := &SupplierProviderSyncRun{ProviderID: provider.ID, SyncScope: SupplierSyncScopeAll, TriggerSource: normalizeSupplierSyncTrigger(trigger), Status: SupplierSyncStatusRunning, StartedAt: startedAt}
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			return SupplierProviderSyncResult{}, fmt.Errorf("create supplier sync run: %w", err)
		}
		result := SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: SupplierSyncScopeAll, Status: SupplierSyncStatusSuccess, StartedAt: startedAt}
		for _, stageFn := range []func() SupplierProviderSyncStage{
			func() SupplierProviderSyncStage {
				return s.syncStageAsSummary(ctx, provider, password, SupplierSyncScopeAccounts)
			},
			func() SupplierProviderSyncStage {
				return s.syncStageAsSummary(ctx, provider, password, SupplierSyncScopeGroups)
			},
			func() SupplierProviderSyncStage {
				return s.syncStageAsSummary(ctx, provider, password, SupplierSyncScopeBalance)
			},
			func() SupplierProviderSyncStage { return s.syncCostStageAsSummary(ctx, provider, password, time.Now()) },
		} {
			stage := stageFn()
			result.Stages = append(result.Stages, stage)
			result.Counts.CheckedCount += stage.Counts.CheckedCount
			result.Counts.CreatedCount += stage.Counts.CreatedCount
			result.Counts.UpdatedCount += stage.Counts.UpdatedCount
			result.Counts.SkippedCount += stage.Counts.SkippedCount
			if stage.Status == SupplierSyncStatusFailed {
				result.Status = SupplierSyncStatusPartial
			}
		}
		if allStagesFailed(result.Stages) {
			result.Status = SupplierSyncStatusFailed
		}
		result.FinishedAt = time.Now()
		result.Message = supplierSyncMessage(result.Status)
		finishedAt := result.FinishedAt
		run.Status = result.Status
		run.Counts = result.Counts
		run.ErrorMessage = result.Message
		run.FinishedAt = &finishedAt
		if err := s.dataRepo.FinishSyncRun(ctx, run); err != nil {
			return result, fmt.Errorf("finish supplier sync run: %w", err)
		}
		_ = s.dataRepo.UpdateSyncStatus(ctx, provider.ID, result.Status, result.Message, finishedAt)
		return result, nil
	})
}

func (s *SupplierProviderSyncService) SyncAllEnabled(ctx context.Context, trigger string) (SupplierProviderBatchSyncResult, error) {
	enabled := true
	providers, _, err := s.providerRepo.List(ctx, SupplierProviderListParams{Enabled: &enabled, Page: 1, PageSize: 1000})
	if err != nil {
		return SupplierProviderBatchSyncResult{}, fmt.Errorf("list enabled supplier providers: %w", err)
	}
	result := SupplierProviderBatchSyncResult{ProcessedCount: len(providers), Results: make([]SupplierProviderSyncResult, 0, len(providers))}
	for _, provider := range providers {
		item, err := s.SyncAll(ctx, provider.ID, trigger)
		if err != nil {
			item = SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: SupplierSyncScopeAll, Status: SupplierSyncStatusFailed, Message: err.Error(), StartedAt: time.Now(), FinishedAt: time.Now()}
		}
		if item.Status == SupplierSyncStatusSuccess {
			result.SuccessCount++
		} else {
			result.FailedCount++
		}
		result.Results = append(result.Results, item)
	}
	return result, nil
}

func (s *SupplierProviderSyncService) syncWithLock(ctx context.Context, providerID int64, fn func(*SupplierProvider) (SupplierProviderSyncResult, error)) (SupplierProviderSyncResult, error) {
	provider, err := s.validSyncProvider(ctx, providerID)
	if err != nil {
		return SupplierProviderSyncResult{}, err
	}
	owner := uuid.NewString()
	if s.syncLock != nil {
		acquired, err := s.syncLock.TryAcquireSyncLock(ctx, providerID, owner, 15*time.Minute)
		if err != nil {
			return SupplierProviderSyncResult{}, fmt.Errorf("acquire supplier sync lock: %w", err)
		}
		if !acquired {
			return SupplierProviderSyncResult{}, ErrSupplierProviderSyncConflict
		}
		defer func() { _ = s.syncLock.ReleaseSyncLock(context.Background(), providerID, owner) }()
	}
	return fn(provider)
}

func (s *SupplierProviderSyncService) validSyncProvider(ctx context.Context, providerID int64) (*SupplierProvider, error) {
	provider, err := s.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled || !strings.EqualFold(provider.ProviderType, "sub2api") {
		return nil, ErrSupplierProviderInvalid
	}
	return provider, nil
}

func (s *SupplierProviderSyncService) syncStage(ctx context.Context, provider *SupplierProvider, password, scope, trigger string, createRun bool) (SupplierProviderSyncResult, error) {
	if scope == SupplierSyncScopeCost {
		return s.syncCostStage(ctx, provider, password, time.Now(), trigger, createRun)
	}
	startedAt := time.Now()
	result := SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: scope, Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	run := &SupplierProviderSyncRun{ProviderID: provider.ID, SyncScope: scope, TriggerSource: normalizeSupplierSyncTrigger(trigger), Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	if createRun {
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			return result, fmt.Errorf("create supplier sync run: %w", err)
		}
	}
	counts, err := s.executeStage(ctx, provider, password, scope, startedAt)
	result.Counts = counts
	result.FinishedAt = time.Now()
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
	} else {
		result.Status = SupplierSyncStatusSuccess
		result.Message = supplierSyncMessage(result.Status)
	}
	if createRun {
		finishedAt := result.FinishedAt
		run.Status = result.Status
		run.Counts = result.Counts
		run.ErrorMessage = result.Message
		run.FinishedAt = &finishedAt
		if finishErr := s.dataRepo.FinishSyncRun(ctx, run); finishErr != nil && err == nil {
			err = fmt.Errorf("finish supplier sync run: %w", finishErr)
		}
		_ = s.dataRepo.UpdateSyncStatus(ctx, provider.ID, result.Status, result.Message, finishedAt)
	}
	return result, err
}

func (s *SupplierProviderSyncService) syncCostStage(ctx context.Context, provider *SupplierProvider, password string, day time.Time, trigger string, createRun bool) (SupplierProviderSyncResult, error) {
	startedAt := time.Now()
	result := SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: SupplierSyncScopeCost, Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	run := &SupplierProviderSyncRun{ProviderID: provider.ID, SyncScope: SupplierSyncScopeCost, TriggerSource: normalizeSupplierSyncTrigger(trigger), Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	if createRun {
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			return result, fmt.Errorf("create supplier sync run: %w", err)
		}
	}
	cost, err := s.remote.FetchCost(ctx, provider, password, day)
	if err == nil {
		err = s.dataRepo.UpdateCost(ctx, provider.ID, cost, startedAt)
	}
	result.Counts = SupplierSyncCounts{CheckedCount: 1, UpdatedCount: boolToInt(err == nil)}
	result.FinishedAt = time.Now()
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
	} else {
		result.Status = SupplierSyncStatusSuccess
		result.Message = supplierSyncMessage(result.Status)
	}
	if createRun {
		finishedAt := result.FinishedAt
		run.Status = result.Status
		run.Counts = result.Counts
		run.ErrorMessage = result.Message
		run.FinishedAt = &finishedAt
		if finishErr := s.dataRepo.FinishSyncRun(ctx, run); finishErr != nil && err == nil {
			err = fmt.Errorf("finish supplier sync run: %w", finishErr)
		}
		_ = s.dataRepo.UpdateSyncStatus(ctx, provider.ID, result.Status, result.Message, finishedAt)
	}
	return result, err
}

func (s *SupplierProviderSyncService) executeStage(ctx context.Context, provider *SupplierProvider, password, scope string, seenAt time.Time) (SupplierSyncCounts, error) {
	switch scope {
	case SupplierSyncScopeAccounts:
		items, err := s.remote.FetchAccounts(ctx, provider, password)
		if err != nil {
			return SupplierSyncCounts{}, err
		}
		return s.dataRepo.ReplaceAccounts(ctx, provider.ID, items, seenAt)
	case SupplierSyncScopeGroups:
		items, err := s.remote.FetchGroups(ctx, provider, password)
		if err != nil {
			return SupplierSyncCounts{}, err
		}
		return s.dataRepo.ReplaceGroups(ctx, provider.ID, items, seenAt)
	case SupplierSyncScopeBalance:
		balance, err := s.remote.FetchBalance(ctx, provider, password)
		if err != nil {
			return SupplierSyncCounts{}, err
		}
		if err := s.dataRepo.UpdateBalance(ctx, provider.ID, balance, seenAt); err != nil {
			return SupplierSyncCounts{}, err
		}
		return SupplierSyncCounts{CheckedCount: 1, UpdatedCount: 1}, nil
	default:
		return SupplierSyncCounts{}, ErrSupplierProviderInvalid
	}
}

func (s *SupplierProviderSyncService) syncStageAsSummary(ctx context.Context, provider *SupplierProvider, password, scope string) SupplierProviderSyncStage {
	result, err := s.syncStage(ctx, provider, password, scope, SupplierSyncTriggerScheduled, false)
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
	}
	return SupplierProviderSyncStage{Scope: scope, Status: result.Status, Message: result.Message, Counts: result.Counts, EndpointResult: s.lastEndpointResult(provider.ID, scope)}
}

func (s *SupplierProviderSyncService) syncCostStageAsSummary(ctx context.Context, provider *SupplierProvider, password string, day time.Time) SupplierProviderSyncStage {
	result, err := s.syncCostStage(ctx, provider, password, day, SupplierSyncTriggerScheduled, false)
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
	}
	return SupplierProviderSyncStage{Scope: SupplierSyncScopeCost, Status: result.Status, Message: result.Message, Counts: result.Counts, EndpointResult: s.lastEndpointResult(provider.ID, SupplierSyncScopeCost)}
}

func (s *SupplierProviderSyncService) lastEndpointResult(providerID int64, scope string) *SupplierProviderEndpointResult {
	diagnostics, ok := s.remote.(SupplierProviderRemoteDiagnostics)
	if !ok {
		return nil
	}
	return diagnostics.LastEndpointResult(providerID, scope)
}

func allStagesFailed(stages []SupplierProviderSyncStage) bool {
	if len(stages) == 0 {
		return false
	}
	for _, stage := range stages {
		if stage.Status != SupplierSyncStatusFailed {
			return false
		}
	}
	return true
}

func normalizeSupplierSyncTrigger(trigger string) string {
	trigger = strings.TrimSpace(trigger)
	if trigger == SupplierSyncTriggerScheduled {
		return trigger
	}
	return SupplierSyncTriggerManual
}

func supplierSyncMessage(status string) string {
	switch status {
	case SupplierSyncStatusSuccess:
		return "同步成功"
	case SupplierSyncStatusPartial:
		return "部分同步失败"
	case SupplierSyncStatusFailed:
		return "同步失败"
	default:
		return ""
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
