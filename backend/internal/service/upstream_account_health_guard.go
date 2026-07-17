package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	SettingKeyUpstreamAccountHealthGuardConfig  = "upstream_account_health_guard_config"
	SettingKeyUpstreamAccountHealthGuardRecords = "upstream_account_health_guard_records"

	DefaultUpstreamAccountHealthGuardIntervalSeconds          = 3600
	MinUpstreamAccountHealthGuardIntervalSeconds              = 60
	DefaultUpstreamAccountHealthGuardMaxAccountsPerRun        = 200
	MaxUpstreamAccountHealthGuardMaxAccountsPerRun            = 1000
	DefaultUpstreamAccountHealthGuardConcurrency              = 3
	MaxUpstreamAccountHealthGuardConcurrency                  = 8
	DefaultUpstreamAccountHealthGuardTimeoutPerAccountSeconds = 90
	MaxUpstreamAccountHealthGuardTimeoutPerAccountSeconds     = 300
	DefaultUpstreamAccountHealthGuardFailureThreshold         = 3
	DefaultUpstreamAccountHealthGuardSlowThreshold            = 3
	DefaultUpstreamAccountHealthGuardRecoveryThreshold        = 2
	DefaultUpstreamAccountHealthGuardHealthyLatencyMs         = 15000
	MaxUpstreamAccountHealthGuardRecords                      = 50

	UpstreamAccountHealthGuardTriggerScheduled = "scheduled"
	UpstreamAccountHealthGuardTriggerManual    = "manual"

	UpstreamAccountHealthGuardStatusHealthy = "healthy"
	UpstreamAccountHealthGuardStatusSlow    = "slow"
	UpstreamAccountHealthGuardStatusFailed  = "failed"

	UpstreamAccountHealthGuardActionNone      = "none"
	UpstreamAccountHealthGuardActionDisabled  = "disabled"
	UpstreamAccountHealthGuardActionRecovered = "recovered"
)

const (
	upstreamHealthGuardFailureCountExtraKey     = "upstream_health_guard_failure_count"
	upstreamHealthGuardSlowCountExtraKey        = "upstream_health_guard_slow_count"
	upstreamHealthGuardHealthyCountExtraKey     = "upstream_health_guard_healthy_count"
	upstreamHealthGuardLastStatusExtraKey       = "upstream_health_guard_last_status"
	upstreamHealthGuardLastLatencyMsExtraKey    = "upstream_health_guard_last_latency_ms"
	upstreamHealthGuardLastCheckedAtExtraKey    = "upstream_health_guard_last_checked_at"
	upstreamHealthGuardLastActionExtraKey       = "upstream_health_guard_last_action"
	upstreamHealthGuardLastMessageExtraKey      = "upstream_health_guard_last_message"
	upstreamHealthGuardLastTestModelExtraKey    = "upstream_health_guard_last_test_model"
	upstreamHealthGuardLastLatencyLimitExtraKey = "upstream_health_guard_last_latency_limit_ms"
)

const (
	upstreamAccountHealthGuardSkipAccountDisabled   = "account_disabled"
	upstreamAccountHealthGuardSkipAccountIgnored    = "account_ignored"
	upstreamAccountHealthGuardSkipMissingProvider   = "missing_provider_slug"
	upstreamAccountHealthGuardSkipProviderDisabled  = "provider_disabled"
	upstreamAccountHealthGuardSkipProviderNotFound  = "provider_not_found"
	upstreamAccountHealthGuardSkipReasonSampleLimit = 5
)

type UpstreamAccountHealthGuardConfig struct {
	Enabled                  bool              `json:"enabled"`
	IntervalSeconds          int               `json:"interval_seconds"`
	MaxAccountsPerRun        int               `json:"max_accounts_per_run"`
	Concurrency              int               `json:"concurrency"`
	TimeoutPerAccountSeconds int               `json:"timeout_per_account_seconds"`
	FailureThreshold         int               `json:"failure_threshold"`
	SlowThreshold            int               `json:"slow_threshold"`
	RecoveryThreshold        int               `json:"recovery_threshold"`
	HealthyLatencyMs         int64             `json:"healthy_latency_ms"`
	IgnoredAccountIDs        []int64           `json:"ignored_account_ids,omitempty"`
	AccountModels            map[int64]string  `json:"account_models,omitempty"`
	PlatformModels           map[string]string `json:"platform_models,omitempty"`
	PlatformLatencyMs        map[string]int64  `json:"platform_latency_ms,omitempty"`
	LastRunAt                *time.Time        `json:"last_run_at,omitempty"`
	LastRunStatus            string            `json:"last_run_status,omitempty"`
	LastRunMessage           string            `json:"last_run_message,omitempty"`
	CursorAccountID          int64             `json:"cursor_account_id,omitempty"`
	UpdatedAt                *time.Time        `json:"updated_at,omitempty"`
}

type UpstreamAccountHealthGuardRunResponse struct {
	Config UpstreamAccountHealthGuardConfig    `json:"config"`
	Record UpstreamAccountHealthGuardRunRecord `json:"record"`
}

type UpstreamAccountHealthGuardRunSummary struct {
	TotalAccounts  int                                    `json:"total_accounts"`
	CheckedCount   int                                    `json:"checked_count"`
	HealthyCount   int                                    `json:"healthy_count"`
	SlowCount      int                                    `json:"slow_count"`
	FailedCount    int                                    `json:"failed_count"`
	SkippedCount   int                                    `json:"skipped_count"`
	DisabledCount  int                                    `json:"disabled_count"`
	RecoveredCount int                                    `json:"recovered_count"`
	UnchangedCount int                                    `json:"unchanged_count"`
	SkipReasons    []UpstreamAccountHealthGuardSkipReason `json:"skip_reasons,omitempty"`
}

type UpstreamAccountHealthGuardSkippedAccount struct {
	AccountID    int64  `json:"account_id"`
	AccountName  string `json:"account_name"`
	Platform     string `json:"platform"`
	ProviderSlug string `json:"provider_slug,omitempty"`
}

type UpstreamAccountHealthGuardSkipReason struct {
	Reason         string                                     `json:"reason"`
	Count          int                                        `json:"count"`
	SampleAccounts []UpstreamAccountHealthGuardSkippedAccount `json:"sample_accounts,omitempty"`
}

type UpstreamAccountHealthGuardRunRecord struct {
	ID         string                               `json:"id"`
	Trigger    string                               `json:"trigger"`
	Status     string                               `json:"status"`
	Message    string                               `json:"message,omitempty"`
	StartedAt  time.Time                            `json:"started_at"`
	FinishedAt time.Time                            `json:"finished_at"`
	Summary    UpstreamAccountHealthGuardRunSummary `json:"summary"`
	Items      []UpstreamAccountHealthGuardRunItem  `json:"items"`
}

type UpstreamAccountHealthGuardRunItem struct {
	AccountID          int64     `json:"account_id"`
	AccountName        string    `json:"account_name"`
	Platform           string    `json:"platform"`
	ProviderSlug       string    `json:"provider_slug"`
	ProviderName       string    `json:"provider_name"`
	ModelID            string    `json:"model_id,omitempty"`
	SchedulableBefore  bool      `json:"schedulable_before"`
	SchedulableAfter   bool      `json:"schedulable_after"`
	Status             string    `json:"status"`
	TestStatus         string    `json:"test_status,omitempty"`
	LatencyMs          int64     `json:"latency_ms"`
	LatencyLimitMs     int64     `json:"latency_limit_ms"`
	ConsecutiveFailed  int       `json:"consecutive_failed"`
	ConsecutiveSlow    int       `json:"consecutive_slow"`
	ConsecutiveHealthy int       `json:"consecutive_healthy"`
	Action             string    `json:"action"`
	Reason             string    `json:"reason,omitempty"`
	ErrorMessage       string    `json:"error_message,omitempty"`
	StartedAt          time.Time `json:"started_at"`
	FinishedAt         time.Time `json:"finished_at"`
}

type upstreamAccountHealthGuardProviderSource interface {
	ListProviders(ctx context.Context) ([]UpstreamProviderConfig, error)
}

type upstreamAccountHealthGuardAccountStore interface {
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, *pagination.PaginationResult, error)
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
	SetSchedulable(ctx context.Context, id int64, schedulable bool) error
}

type upstreamAccountHealthGuardTester interface {
	runTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

type UpstreamAccountHealthGuardRecordStore interface {
	SaveRecord(ctx context.Context, record UpstreamAccountHealthGuardRunRecord, keepLimit int) error
	ListRecords(ctx context.Context, limit int) ([]UpstreamAccountHealthGuardRunRecord, error)
}

type UpstreamAccountHealthGuardService struct {
	accountStore   upstreamAccountHealthGuardAccountStore
	providerSource upstreamAccountHealthGuardProviderSource
	settingRepo    SettingRepository
	tester         upstreamAccountHealthGuardTester
	recordStore    UpstreamAccountHealthGuardRecordStore
	now            func() time.Time
}

type upstreamAccountHealthGuardTarget struct {
	account  Account
	provider UpstreamProviderConfig
}

type upstreamAccountHealthGuardSkipReasonCollector struct {
	order   []string
	reasons map[string]*UpstreamAccountHealthGuardSkipReason
}

func newUpstreamAccountHealthGuardSkipReasonCollector() *upstreamAccountHealthGuardSkipReasonCollector {
	return &upstreamAccountHealthGuardSkipReasonCollector{reasons: map[string]*UpstreamAccountHealthGuardSkipReason{}}
}

func (c *upstreamAccountHealthGuardSkipReasonCollector) add(reason string, account Account, providerSlug string) {
	if c == nil || reason == "" {
		return
	}
	item, ok := c.reasons[reason]
	if !ok {
		item = &UpstreamAccountHealthGuardSkipReason{Reason: reason}
		c.reasons[reason] = item
		c.order = append(c.order, reason)
	}
	item.Count++
	if len(item.SampleAccounts) >= upstreamAccountHealthGuardSkipReasonSampleLimit {
		return
	}
	item.SampleAccounts = append(item.SampleAccounts, UpstreamAccountHealthGuardSkippedAccount{
		AccountID:    account.ID,
		AccountName:  account.Name,
		Platform:     account.Platform,
		ProviderSlug: providerSlug,
	})
}

func (c *upstreamAccountHealthGuardSkipReasonCollector) list() []UpstreamAccountHealthGuardSkipReason {
	if c == nil || len(c.order) == 0 {
		return nil
	}
	out := make([]UpstreamAccountHealthGuardSkipReason, 0, len(c.order))
	for _, reason := range c.order {
		if item := c.reasons[reason]; item != nil {
			out = append(out, *item)
		}
	}
	return out
}

func NewUpstreamAccountHealthGuardService(
	accountStore AccountRepository,
	providerSource *UpstreamProviderService,
	settingRepo SettingRepository,
	tester *AccountTestService,
	recordStore UpstreamAccountHealthGuardRecordStore,
) *UpstreamAccountHealthGuardService {
	return newUpstreamAccountHealthGuardServiceWithDeps(accountStore, providerSource, settingRepo, tester, recordStore)
}

func newUpstreamAccountHealthGuardServiceWithDeps(
	accountStore upstreamAccountHealthGuardAccountStore,
	providerSource upstreamAccountHealthGuardProviderSource,
	settingRepo SettingRepository,
	tester upstreamAccountHealthGuardTester,
	recordStore UpstreamAccountHealthGuardRecordStore,
) *UpstreamAccountHealthGuardService {
	return &UpstreamAccountHealthGuardService{
		accountStore:   accountStore,
		providerSource: providerSource,
		settingRepo:    settingRepo,
		tester:         tester,
		recordStore:    recordStore,
		now:            func() time.Time { return time.Now().UTC() },
	}
}

func (s *UpstreamAccountHealthGuardService) GetConfig(ctx context.Context) (UpstreamAccountHealthGuardConfig, error) {
	if s == nil || s.settingRepo == nil {
		return defaultUpstreamAccountHealthGuardConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamAccountHealthGuardConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaultUpstreamAccountHealthGuardConfig(), nil
		}
		return UpstreamAccountHealthGuardConfig{}, fmt.Errorf("load upstream account health guard config: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultUpstreamAccountHealthGuardConfig(), nil
	}
	var config UpstreamAccountHealthGuardConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return UpstreamAccountHealthGuardConfig{}, infraerrors.InternalServer("UPSTREAM_ACCOUNT_HEALTH_GUARD_CONFIG_INVALID", "upstream account health guard config is invalid")
	}
	return normalizeUpstreamAccountHealthGuardConfig(config), nil
}

func (s *UpstreamAccountHealthGuardService) UpdateConfig(ctx context.Context, input UpstreamAccountHealthGuardConfig) (UpstreamAccountHealthGuardConfig, error) {
	if input.IntervalSeconds > 0 && input.IntervalSeconds < MinUpstreamAccountHealthGuardIntervalSeconds {
		return UpstreamAccountHealthGuardConfig{}, infraerrors.BadRequest("UPSTREAM_ACCOUNT_HEALTH_GUARD_INTERVAL_INVALID", fmt.Sprintf("interval_seconds must be at least %d", MinUpstreamAccountHealthGuardIntervalSeconds))
	}
	current, err := s.GetConfig(ctx)
	if err != nil {
		return UpstreamAccountHealthGuardConfig{}, err
	}
	config := normalizeUpstreamAccountHealthGuardConfig(input)
	config.LastRunAt = current.LastRunAt
	config.LastRunStatus = current.LastRunStatus
	config.LastRunMessage = current.LastRunMessage
	config.CursorAccountID = current.CursorAccountID
	now := s.currentTime()
	config.UpdatedAt = &now
	return s.saveConfig(ctx, config)
}

func (s *UpstreamAccountHealthGuardService) RunScheduled(ctx context.Context) (UpstreamAccountHealthGuardRunResponse, error) {
	return s.Run(ctx, UpstreamAccountHealthGuardTriggerScheduled)
}

func (s *UpstreamAccountHealthGuardService) Run(ctx context.Context, trigger string) (UpstreamAccountHealthGuardRunResponse, error) {
	config, err := s.GetConfig(ctx)
	if err != nil {
		return UpstreamAccountHealthGuardRunResponse{}, err
	}
	startedAt := s.currentTime()
	record := UpstreamAccountHealthGuardRunRecord{
		ID:        startedAt.Format("20060102T150405.000000000Z"),
		Trigger:   normalizeUpstreamAccountHealthGuardTrigger(trigger),
		Status:    "success",
		StartedAt: startedAt,
		Items:     []UpstreamAccountHealthGuardRunItem{},
	}

	if s == nil || s.accountStore == nil || s.providerSource == nil || s.tester == nil {
		err := infraerrors.ServiceUnavailable("UPSTREAM_ACCOUNT_HEALTH_GUARD_UNAVAILABLE", "upstream account health guard dependencies are unavailable")
		return s.finishRunWithError(ctx, config, record, err)
	}

	targets, totalAccounts, skipReasons, nextCursor, err := s.listTargets(ctx, config)
	if err != nil {
		return s.finishRunWithError(ctx, config, record, err)
	}
	record.Summary.TotalAccounts = totalAccounts
	record.Summary.SkipReasons = skipReasons
	record.Summary.SkippedCount = upstreamAccountHealthGuardSkipReasonTotal(skipReasons)

	items := s.runTargets(ctx, config, targets)
	sort.Slice(items, func(i, j int) bool {
		if items[i].ProviderSlug == items[j].ProviderSlug {
			return items[i].AccountID < items[j].AccountID
		}
		return items[i].ProviderSlug < items[j].ProviderSlug
	})
	record.Items = items
	record.Summary.CheckedCount = len(items)
	for _, item := range items {
		switch item.Status {
		case UpstreamAccountHealthGuardStatusHealthy:
			record.Summary.HealthyCount++
		case UpstreamAccountHealthGuardStatusSlow:
			record.Summary.SlowCount++
		default:
			record.Summary.FailedCount++
		}
		switch item.Action {
		case UpstreamAccountHealthGuardActionDisabled:
			record.Summary.DisabledCount++
		case UpstreamAccountHealthGuardActionRecovered:
			record.Summary.RecoveredCount++
		}
	}
	record.Summary.UnchangedCount = record.Summary.CheckedCount - record.Summary.DisabledCount - record.Summary.RecoveredCount
	if record.Summary.UnchangedCount < 0 {
		record.Summary.UnchangedCount = 0
	}
	record.Message = upstreamAccountHealthGuardSummaryMessage(record.Summary)

	finishedAt := s.currentTime()
	record.FinishedAt = finishedAt
	config.CursorAccountID = nextCursor
	config.LastRunAt = &finishedAt
	config.LastRunStatus = record.Status
	config.LastRunMessage = record.Message
	saved, err := s.saveConfig(ctx, config)
	if err != nil {
		return UpstreamAccountHealthGuardRunResponse{}, err
	}
	record.Status = saved.LastRunStatus
	if err := s.saveRecord(ctx, record); err != nil {
		return UpstreamAccountHealthGuardRunResponse{}, err
	}
	return UpstreamAccountHealthGuardRunResponse{Config: saved, Record: record}, nil
}

func (s *UpstreamAccountHealthGuardService) ListRecords(ctx context.Context) ([]UpstreamAccountHealthGuardRunRecord, error) {
	if s != nil && s.recordStore != nil {
		return s.recordStore.ListRecords(ctx, MaxUpstreamAccountHealthGuardRecords)
	}
	if s == nil || s.settingRepo == nil {
		return []UpstreamAccountHealthGuardRunRecord{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamAccountHealthGuardRecords)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return []UpstreamAccountHealthGuardRunRecord{}, nil
		}
		return nil, fmt.Errorf("load upstream account health guard records: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []UpstreamAccountHealthGuardRunRecord{}, nil
	}
	var records []UpstreamAccountHealthGuardRunRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, infraerrors.InternalServer("UPSTREAM_ACCOUNT_HEALTH_GUARD_RECORDS_INVALID", "upstream account health guard records are invalid")
	}
	return limitUpstreamAccountHealthGuardRecords(records), nil
}

func (s *UpstreamAccountHealthGuardService) finishRunWithError(
	ctx context.Context,
	config UpstreamAccountHealthGuardConfig,
	record UpstreamAccountHealthGuardRunRecord,
	runErr error,
) (UpstreamAccountHealthGuardRunResponse, error) {
	finishedAt := s.currentTime()
	record.FinishedAt = finishedAt
	record.Status = "failed"
	record.Message = runErr.Error()
	config.LastRunAt = &finishedAt
	config.LastRunStatus = record.Status
	config.LastRunMessage = record.Message
	saved, saveErr := s.saveConfig(ctx, config)
	if saveErr != nil {
		return UpstreamAccountHealthGuardRunResponse{}, saveErr
	}
	_ = s.saveRecord(ctx, record)
	return UpstreamAccountHealthGuardRunResponse{Config: saved, Record: record}, runErr
}

func (s *UpstreamAccountHealthGuardService) listTargets(ctx context.Context, config UpstreamAccountHealthGuardConfig) ([]upstreamAccountHealthGuardTarget, int, []UpstreamAccountHealthGuardSkipReason, int64, error) {
	providers, err := s.providerSource.ListProviders(ctx)
	if err != nil {
		return nil, 0, nil, 0, fmt.Errorf("list upstream providers: %w", err)
	}
	allProviders := make(map[string]UpstreamProviderConfig, len(providers))
	enabledProviders := make(map[string]UpstreamProviderConfig, len(providers))
	for _, provider := range providers {
		slug := strings.TrimSpace(provider.Slug)
		if slug == "" {
			continue
		}
		allProviders[slug] = provider
		if provider.Enabled {
			enabledProviders[slug] = provider
		}
	}

	maxAccounts := config.MaxAccountsPerRun
	if maxAccounts <= 0 {
		maxAccounts = DefaultUpstreamAccountHealthGuardMaxAccountsPerRun
	}
	targets := make([]upstreamAccountHealthGuardTarget, 0, maxAccounts)
	deferredTargets := make([]upstreamAccountHealthGuardTarget, 0)
	cursor := config.CursorAccountID
	totalAccounts := 0
	skipReasons := newUpstreamAccountHealthGuardSkipReasonCollector()
	ignoredAccountIDs := s.ignoredAccountIDSet(ctx, config)
	const pageSize = 500
	for page := 1; ; page++ {
		accounts, result, err := s.accountStore.ListWithFilters(ctx, pagination.PaginationParams{
			Page:      page,
			PageSize:  pageSize,
			SortBy:    "id",
			SortOrder: "asc",
		}, "", "", "", "", 0, "")
		if err != nil {
			return nil, 0, nil, 0, fmt.Errorf("list accounts: %w", err)
		}
		if result != nil && result.Total > int64(totalAccounts) {
			totalAccounts = int(result.Total)
		} else if result == nil {
			totalAccounts += len(accounts)
		}

		for _, account := range accounts {
			providerSlug := upstreamAccountHealthGuardExtraString(account.Extra, "upstream_provider_slug")
			if upstreamAccountSyncInt64SetContains(ignoredAccountIDs, account.ID) {
				skipReasons.add(upstreamAccountHealthGuardSkipAccountIgnored, account, providerSlug)
				continue
			}
			if account.Status == StatusDisabled {
				skipReasons.add(upstreamAccountHealthGuardSkipAccountDisabled, account, providerSlug)
				continue
			}
			if providerSlug == "" {
				skipReasons.add(upstreamAccountHealthGuardSkipMissingProvider, account, providerSlug)
				continue
			}
			provider, providerEnabled := enabledProviders[providerSlug]
			if !providerEnabled {
				if _, exists := allProviders[providerSlug]; exists {
					skipReasons.add(upstreamAccountHealthGuardSkipProviderDisabled, account, providerSlug)
				} else {
					skipReasons.add(upstreamAccountHealthGuardSkipProviderNotFound, account, providerSlug)
				}
				continue
			}
			target := upstreamAccountHealthGuardTarget{account: account, provider: provider}
			if cursor > 0 && account.ID <= cursor {
				deferredTargets = append(deferredTargets, target)
				continue
			}
			if len(targets) < maxAccounts {
				targets = append(targets, target)
			}
		}

		if len(accounts) == 0 {
			break
		}
		if result != nil && int64(page*pageSize) >= result.Total {
			break
		}
	}
	if len(targets) < maxAccounts {
		for _, target := range deferredTargets {
			targets = append(targets, target)
			if len(targets) >= maxAccounts {
				break
			}
		}
	}
	nextCursor := cursor
	if len(targets) > 0 {
		nextCursor = targets[len(targets)-1].account.ID
	}
	return targets, totalAccounts, skipReasons.list(), nextCursor, nil
}

func (s *UpstreamAccountHealthGuardService) ignoredAccountIDSet(ctx context.Context, config UpstreamAccountHealthGuardConfig) map[int64]struct{} {
	out := upstreamAccountSyncInt64Set(config.IgnoredAccountIDs)
	if s == nil || s.settingRepo == nil {
		return out
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamAccountRateGuardConfig)
	if err != nil || strings.TrimSpace(raw) == "" {
		return out
	}
	var rateGuardConfig UpstreamAccountRateGuardConfig
	if err := json.Unmarshal([]byte(raw), &rateGuardConfig); err != nil {
		return out
	}
	for _, accountID := range normalizeUpstreamAccountRateGuardIgnoredAccountIDs(rateGuardConfig.IgnoredAccountIDs) {
		if out == nil {
			out = map[int64]struct{}{}
		}
		out[accountID] = struct{}{}
	}
	return out
}

func (s *UpstreamAccountHealthGuardService) runTargets(ctx context.Context, config UpstreamAccountHealthGuardConfig, targets []upstreamAccountHealthGuardTarget) []UpstreamAccountHealthGuardRunItem {
	if len(targets) == 0 {
		return []UpstreamAccountHealthGuardRunItem{}
	}
	concurrency := config.Concurrency
	if concurrency <= 0 {
		concurrency = DefaultUpstreamAccountHealthGuardConcurrency
	}
	if concurrency > len(targets) {
		concurrency = len(targets)
	}
	jobs := make(chan upstreamAccountHealthGuardTarget)
	results := make(chan UpstreamAccountHealthGuardRunItem, len(targets))
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range jobs {
				results <- s.runTarget(ctx, config, target)
			}
		}()
	}

enqueue:
	for _, target := range targets {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- target:
		}
	}
	close(jobs)
	wg.Wait()
	close(results)

	items := make([]UpstreamAccountHealthGuardRunItem, 0, len(targets))
	for item := range results {
		items = append(items, item)
	}
	return items
}

func (s *UpstreamAccountHealthGuardService) runTarget(ctx context.Context, config UpstreamAccountHealthGuardConfig, target upstreamAccountHealthGuardTarget) UpstreamAccountHealthGuardRunItem {
	account := target.account
	startedAt := s.currentTime()
	modelID := upstreamAccountHealthGuardModelForAccount(config, account.ID, account.Platform)
	latencyLimit := upstreamAccountHealthGuardLatencyLimitForPlatform(config, account.Platform)
	item := UpstreamAccountHealthGuardRunItem{
		AccountID:         account.ID,
		AccountName:       account.Name,
		Platform:          account.Platform,
		ProviderSlug:      target.provider.Slug,
		ProviderName:      target.provider.Name,
		ModelID:           modelID,
		SchedulableBefore: account.Schedulable,
		SchedulableAfter:  account.Schedulable,
		LatencyLimitMs:    latencyLimit,
		Action:            UpstreamAccountHealthGuardActionNone,
		StartedAt:         startedAt,
		FinishedAt:        startedAt,
	}

	timeout := time.Duration(config.TimeoutPerAccountSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(DefaultUpstreamAccountHealthGuardTimeoutPerAccountSeconds) * time.Second
	}
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := s.tester.runTestBackground(testCtx, account.ID, modelID)
	finishedAt := s.currentTime()
	item.FinishedAt = finishedAt
	item.LatencyMs = finishedAt.Sub(startedAt).Milliseconds()
	if result != nil {
		item.TestStatus = strings.TrimSpace(result.Status)
		item.LatencyMs = result.LatencyMs
		if !result.StartedAt.IsZero() {
			item.StartedAt = result.StartedAt.UTC()
		}
		if !result.FinishedAt.IsZero() {
			item.FinishedAt = result.FinishedAt.UTC()
		}
		if result.ErrorMessage != "" {
			item.ErrorMessage = result.ErrorMessage
		}
	}
	if item.TestStatus == "" {
		item.TestStatus = "failed"
	}
	item.Status, item.Reason = upstreamAccountHealthGuardEvaluateResult(testCtx.Err(), err, result, item.LatencyMs, latencyLimit)
	if item.ErrorMessage == "" {
		if err != nil {
			item.ErrorMessage = err.Error()
		} else if errors.Is(testCtx.Err(), context.DeadlineExceeded) {
			item.ErrorMessage = "account test timed out"
		} else if errors.Is(testCtx.Err(), context.Canceled) {
			item.ErrorMessage = "account test cancelled"
		}
	}
	if item.Status == UpstreamAccountHealthGuardStatusSlow {
		item.ErrorMessage = ""
	}

	item.ConsecutiveFailed = upstreamAccountHealthGuardExtraInt(account.Extra, upstreamHealthGuardFailureCountExtraKey)
	item.ConsecutiveSlow = upstreamAccountHealthGuardExtraInt(account.Extra, upstreamHealthGuardSlowCountExtraKey)
	item.ConsecutiveHealthy = upstreamAccountHealthGuardExtraInt(account.Extra, upstreamHealthGuardHealthyCountExtraKey)
	switch item.Status {
	case UpstreamAccountHealthGuardStatusHealthy:
		item.ConsecutiveHealthy++
		item.ConsecutiveFailed = 0
		item.ConsecutiveSlow = 0
	case UpstreamAccountHealthGuardStatusSlow:
		item.ConsecutiveSlow++
		item.ConsecutiveHealthy = 0
		item.ConsecutiveFailed = 0
	default:
		item.ConsecutiveFailed++
		item.ConsecutiveHealthy = 0
		item.ConsecutiveSlow = 0
	}

	item.SchedulableAfter, item.Action, item.Reason = upstreamAccountHealthGuardNextSchedulingState(config, item)
	if item.SchedulableAfter != item.SchedulableBefore {
		if err := s.accountStore.SetSchedulable(ctx, item.AccountID, item.SchedulableAfter); err != nil {
			item.SchedulableAfter = item.SchedulableBefore
			item.Action = UpstreamAccountHealthGuardActionNone
			item.ErrorMessage = upstreamAccountHealthGuardAppendMessage(item.ErrorMessage, fmt.Sprintf("update schedulable: %v", err))
		}
	}

	updateErr := s.accountStore.UpdateExtra(ctx, item.AccountID, map[string]any{
		upstreamHealthGuardFailureCountExtraKey:     item.ConsecutiveFailed,
		upstreamHealthGuardSlowCountExtraKey:        item.ConsecutiveSlow,
		upstreamHealthGuardHealthyCountExtraKey:     item.ConsecutiveHealthy,
		upstreamHealthGuardLastStatusExtraKey:       item.Status,
		upstreamHealthGuardLastLatencyMsExtraKey:    item.LatencyMs,
		upstreamHealthGuardLastCheckedAtExtraKey:    item.FinishedAt.UTC().Format(time.RFC3339),
		upstreamHealthGuardLastActionExtraKey:       item.Action,
		upstreamHealthGuardLastMessageExtraKey:      item.Reason,
		upstreamHealthGuardLastTestModelExtraKey:    item.ModelID,
		upstreamHealthGuardLastLatencyLimitExtraKey: item.LatencyLimitMs,
	})
	if updateErr != nil {
		item.ErrorMessage = upstreamAccountHealthGuardAppendMessage(item.ErrorMessage, fmt.Sprintf("update extra: %v", updateErr))
	}
	return item
}

func upstreamAccountHealthGuardEvaluateResult(ctxErr error, runErr error, result *ScheduledTestResult, latencyMs, latencyLimitMs int64) (string, string) {
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		return UpstreamAccountHealthGuardStatusFailed, "test timeout"
	}
	if errors.Is(ctxErr, context.Canceled) {
		return UpstreamAccountHealthGuardStatusFailed, "test cancelled"
	}
	if runErr != nil {
		return UpstreamAccountHealthGuardStatusFailed, runErr.Error()
	}
	if result == nil {
		return UpstreamAccountHealthGuardStatusFailed, "empty test result"
	}
	if strings.TrimSpace(result.Status) != "success" {
		if strings.TrimSpace(result.ErrorMessage) != "" {
			return UpstreamAccountHealthGuardStatusFailed, result.ErrorMessage
		}
		return UpstreamAccountHealthGuardStatusFailed, "test failed"
	}
	if latencyLimitMs > 0 && latencyMs > latencyLimitMs {
		return UpstreamAccountHealthGuardStatusSlow, fmt.Sprintf("latency %dms exceeds %dms", latencyMs, latencyLimitMs)
	}
	return UpstreamAccountHealthGuardStatusHealthy, "test passed"
}

func upstreamAccountHealthGuardNextSchedulingState(config UpstreamAccountHealthGuardConfig, item UpstreamAccountHealthGuardRunItem) (bool, string, string) {
	switch item.Status {
	case UpstreamAccountHealthGuardStatusHealthy:
		if !item.SchedulableBefore && item.ConsecutiveHealthy >= config.RecoveryThreshold {
			return true, UpstreamAccountHealthGuardActionRecovered, fmt.Sprintf("healthy %d times", item.ConsecutiveHealthy)
		}
	case UpstreamAccountHealthGuardStatusSlow:
		if item.SchedulableBefore && item.ConsecutiveSlow >= config.SlowThreshold {
			return false, UpstreamAccountHealthGuardActionDisabled, fmt.Sprintf("slow %d times", item.ConsecutiveSlow)
		}
	default:
		if item.SchedulableBefore && item.ConsecutiveFailed >= config.FailureThreshold {
			return false, UpstreamAccountHealthGuardActionDisabled, fmt.Sprintf("failed %d times", item.ConsecutiveFailed)
		}
	}
	return item.SchedulableBefore, UpstreamAccountHealthGuardActionNone, item.Reason
}

func (s *UpstreamAccountHealthGuardService) saveConfig(ctx context.Context, config UpstreamAccountHealthGuardConfig) (UpstreamAccountHealthGuardConfig, error) {
	config = normalizeUpstreamAccountHealthGuardConfig(config)
	if s == nil || s.settingRepo == nil {
		return config, nil
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return UpstreamAccountHealthGuardConfig{}, fmt.Errorf("marshal upstream account health guard config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamAccountHealthGuardConfig, string(raw)); err != nil {
		return UpstreamAccountHealthGuardConfig{}, fmt.Errorf("save upstream account health guard config: %w", err)
	}
	return config, nil
}

func (s *UpstreamAccountHealthGuardService) saveRecord(ctx context.Context, record UpstreamAccountHealthGuardRunRecord) error {
	if s != nil && s.recordStore != nil {
		if err := s.recordStore.SaveRecord(ctx, record, MaxUpstreamAccountHealthGuardRecords); err != nil {
			return err
		}
		if s.settingRepo != nil {
			_ = s.settingRepo.Delete(ctx, SettingKeyUpstreamAccountHealthGuardRecords)
		}
		return nil
	}
	if s == nil || s.settingRepo == nil {
		return nil
	}
	records, err := s.ListRecords(ctx)
	if err != nil {
		return err
	}
	records = append([]UpstreamAccountHealthGuardRunRecord{record}, records...)
	records = limitUpstreamAccountHealthGuardRecords(records)
	raw, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshal upstream account health guard records: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamAccountHealthGuardRecords, string(raw)); err != nil {
		return fmt.Errorf("save upstream account health guard records: %w", err)
	}
	return nil
}

func (s *UpstreamAccountHealthGuardService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func defaultUpstreamAccountHealthGuardConfig() UpstreamAccountHealthGuardConfig {
	return UpstreamAccountHealthGuardConfig{
		Enabled:                  false,
		IntervalSeconds:          DefaultUpstreamAccountHealthGuardIntervalSeconds,
		MaxAccountsPerRun:        DefaultUpstreamAccountHealthGuardMaxAccountsPerRun,
		Concurrency:              DefaultUpstreamAccountHealthGuardConcurrency,
		TimeoutPerAccountSeconds: DefaultUpstreamAccountHealthGuardTimeoutPerAccountSeconds,
		FailureThreshold:         DefaultUpstreamAccountHealthGuardFailureThreshold,
		SlowThreshold:            DefaultUpstreamAccountHealthGuardSlowThreshold,
		RecoveryThreshold:        DefaultUpstreamAccountHealthGuardRecoveryThreshold,
		HealthyLatencyMs:         DefaultUpstreamAccountHealthGuardHealthyLatencyMs,
		AccountModels:            map[int64]string{},
		PlatformModels:           map[string]string{},
		PlatformLatencyMs:        map[string]int64{},
	}
}

func normalizeUpstreamAccountHealthGuardConfig(config UpstreamAccountHealthGuardConfig) UpstreamAccountHealthGuardConfig {
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = DefaultUpstreamAccountHealthGuardIntervalSeconds
	}
	if config.MaxAccountsPerRun <= 0 {
		config.MaxAccountsPerRun = DefaultUpstreamAccountHealthGuardMaxAccountsPerRun
	}
	if config.MaxAccountsPerRun > MaxUpstreamAccountHealthGuardMaxAccountsPerRun {
		config.MaxAccountsPerRun = MaxUpstreamAccountHealthGuardMaxAccountsPerRun
	}
	if config.Concurrency <= 0 {
		config.Concurrency = DefaultUpstreamAccountHealthGuardConcurrency
	}
	if config.Concurrency > MaxUpstreamAccountHealthGuardConcurrency {
		config.Concurrency = MaxUpstreamAccountHealthGuardConcurrency
	}
	if config.TimeoutPerAccountSeconds <= 0 {
		config.TimeoutPerAccountSeconds = DefaultUpstreamAccountHealthGuardTimeoutPerAccountSeconds
	}
	if config.TimeoutPerAccountSeconds > MaxUpstreamAccountHealthGuardTimeoutPerAccountSeconds {
		config.TimeoutPerAccountSeconds = MaxUpstreamAccountHealthGuardTimeoutPerAccountSeconds
	}
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultUpstreamAccountHealthGuardFailureThreshold
	}
	if config.SlowThreshold <= 0 {
		config.SlowThreshold = DefaultUpstreamAccountHealthGuardSlowThreshold
	}
	if config.RecoveryThreshold <= 0 {
		config.RecoveryThreshold = DefaultUpstreamAccountHealthGuardRecoveryThreshold
	}
	if config.HealthyLatencyMs <= 0 {
		config.HealthyLatencyMs = DefaultUpstreamAccountHealthGuardHealthyLatencyMs
	}
	config.PlatformModels = normalizeUpstreamAccountHealthGuardPlatformModels(config.PlatformModels)
	config.PlatformLatencyMs = normalizeUpstreamAccountHealthGuardPlatformLatency(config.PlatformLatencyMs)
	config.IgnoredAccountIDs = normalizeUpstreamAccountRateGuardIgnoredAccountIDs(config.IgnoredAccountIDs)
	config.AccountModels = normalizeUpstreamAccountHealthGuardAccountModels(config.AccountModels)
	return config
}

func normalizeUpstreamAccountHealthGuardAccountModels(values map[int64]string) map[int64]string {
	out := map[int64]string{}
	for accountID, model := range values {
		model = strings.TrimSpace(model)
		if accountID <= 0 || model == "" {
			continue
		}
		out[accountID] = model
	}
	return out
}

func normalizeUpstreamAccountHealthGuardPlatformModels(values map[string]string) map[string]string {
	out := map[string]string{}
	for platform, model := range values {
		platform = strings.ToLower(strings.TrimSpace(platform))
		model = strings.TrimSpace(model)
		if platform == "" || model == "" {
			continue
		}
		out[platform] = model
	}
	return out
}

func normalizeUpstreamAccountHealthGuardPlatformLatency(values map[string]int64) map[string]int64 {
	out := map[string]int64{}
	for platform, latency := range values {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if platform == "" || latency <= 0 {
			continue
		}
		out[platform] = latency
	}
	return out
}

func upstreamAccountHealthGuardModelForAccount(config UpstreamAccountHealthGuardConfig, accountID int64, platform string) string {
	if accountID > 0 && len(config.AccountModels) > 0 {
		if model := strings.TrimSpace(config.AccountModels[accountID]); model != "" {
			return model
		}
	}
	return upstreamAccountHealthGuardModelForPlatform(config, platform)
}

func upstreamAccountHealthGuardModelForPlatform(config UpstreamAccountHealthGuardConfig, platform string) string {
	if len(config.PlatformModels) == 0 {
		return ""
	}
	return strings.TrimSpace(config.PlatformModels[strings.ToLower(strings.TrimSpace(platform))])
}

func upstreamAccountHealthGuardLatencyLimitForPlatform(config UpstreamAccountHealthGuardConfig, platform string) int64 {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if len(config.PlatformLatencyMs) > 0 {
		if latency := config.PlatformLatencyMs[platform]; latency > 0 {
			return latency
		}
	}
	return config.HealthyLatencyMs
}

func upstreamAccountHealthGuardExtraString(extra map[string]any, key string) string {
	if extra == nil {
		return ""
	}
	value, _ := extra[key].(string)
	return strings.TrimSpace(value)
}

func upstreamAccountHealthGuardExtraInt(extra map[string]any, key string) int {
	if extra == nil {
		return 0
	}
	return parseExtraInt(extra[key])
}

func normalizeUpstreamAccountHealthGuardTrigger(trigger string) string {
	switch strings.TrimSpace(trigger) {
	case UpstreamAccountHealthGuardTriggerManual:
		return UpstreamAccountHealthGuardTriggerManual
	default:
		return UpstreamAccountHealthGuardTriggerScheduled
	}
}

func upstreamAccountHealthGuardSummaryMessage(summary UpstreamAccountHealthGuardRunSummary) string {
	return fmt.Sprintf(
		"total %d accounts, checked %d, skipped %d, healthy %d, slow %d, failed %d, disabled %d, recovered %d",
		summary.TotalAccounts,
		summary.CheckedCount,
		summary.SkippedCount,
		summary.HealthyCount,
		summary.SlowCount,
		summary.FailedCount,
		summary.DisabledCount,
		summary.RecoveredCount,
	)
}

func upstreamAccountHealthGuardSkipReasonTotal(reasons []UpstreamAccountHealthGuardSkipReason) int {
	total := 0
	for _, reason := range reasons {
		total += reason.Count
	}
	return total
}

func upstreamAccountHealthGuardAppendMessage(current, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	return current + "; " + next
}

func limitUpstreamAccountHealthGuardRecords(records []UpstreamAccountHealthGuardRunRecord) []UpstreamAccountHealthGuardRunRecord {
	if len(records) <= 50 {
		return records
	}
	out := make([]UpstreamAccountHealthGuardRunRecord, 50)
	copy(out, records[:50])
	return out
}
