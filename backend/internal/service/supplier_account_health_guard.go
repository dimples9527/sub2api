package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DefaultSupplierAccountHealthGuardMaxAccountsPerRun        = 200
	MaxSupplierAccountHealthGuardMaxAccountsPerRun            = 1000
	DefaultSupplierAccountHealthGuardConcurrency              = 3
	MaxSupplierAccountHealthGuardConcurrency                  = 32
	DefaultSupplierAccountHealthGuardTimeoutPerAccountSeconds = 30
	MinSupplierAccountHealthGuardTimeoutPerAccountSeconds     = 5
	MaxSupplierAccountHealthGuardTimeoutPerAccountSeconds     = 300
	DefaultSupplierAccountHealthGuardFailureThreshold         = 3
	DefaultSupplierAccountHealthGuardSlowThreshold            = 3
	DefaultSupplierAccountHealthGuardRecoveryThreshold        = 2
	DefaultSupplierAccountHealthGuardHealthyLatencyMs         = 15000
	MinSupplierAccountHealthGuardAccountIntervalSeconds       = 60

	SupplierAccountHealthGuardStatusHealthy     = "healthy"
	SupplierAccountHealthGuardStatusSlow        = "slow"
	SupplierAccountHealthGuardStatusFailed      = "failed"
	SupplierAccountHealthGuardStatusSkipped     = "skipped"
	SupplierAccountHealthGuardStatusUnavailable = "unavailable"

	SupplierAccountHealthGuardActionNone      = "none"
	SupplierAccountHealthGuardActionDisabled  = "disabled"
	SupplierAccountHealthGuardActionRecovered = "recovered"

	SupplierAccountHealthGuardMatchMatched   = "matched"
	SupplierAccountHealthGuardMatchConflict  = "conflict"
	SupplierAccountHealthGuardMatchUnmatched = "unmatched"
)

const (
	supplierHealthGuardFailureCountExtraKey     = "supplier_health_guard_failure_count"
	supplierHealthGuardSlowCountExtraKey        = "supplier_health_guard_slow_count"
	supplierHealthGuardHealthyCountExtraKey     = "supplier_health_guard_healthy_count"
	supplierHealthGuardLastStatusExtraKey       = "supplier_health_guard_last_status"
	supplierHealthGuardLastLatencyMsExtraKey    = "supplier_health_guard_last_latency_ms"
	supplierHealthGuardLastCheckedAtExtraKey    = "supplier_health_guard_last_checked_at"
	supplierHealthGuardLastActionExtraKey       = "supplier_health_guard_last_action"
	supplierHealthGuardLastMessageExtraKey      = "supplier_health_guard_last_message"
	supplierHealthGuardLastTestModelExtraKey    = "supplier_health_guard_last_test_model"
	supplierHealthGuardLastLatencyLimitExtraKey = "supplier_health_guard_last_latency_limit_ms"
)

const (
	supplierAccountHealthGuardSkipUnmatched           = "unmatched"
	supplierAccountHealthGuardSkipConflict            = "conflict"
	supplierAccountHealthGuardSkipLocalAccountMissing = "local_account_missing"
	supplierAccountHealthGuardSkipAccountIgnored      = "account_ignored"
	supplierAccountHealthGuardSkipAccountDisabled     = "account_disabled"
	supplierAccountHealthGuardSkipTestModelMissing    = "test_model_missing"
	supplierAccountHealthGuardSkipReasonSampleLimit   = 5
)

type SupplierAccountHealthGuardConfig struct {
	MaxAccountsPerRun        int               `json:"account_health_guard_max_accounts_per_run"`
	Concurrency              int               `json:"account_health_guard_concurrency"`
	TimeoutPerAccountSeconds int               `json:"account_health_guard_timeout_per_account_seconds"`
	FailureThreshold         int               `json:"account_health_guard_failure_threshold"`
	SlowThreshold            int               `json:"account_health_guard_slow_threshold"`
	RecoveryThreshold        int               `json:"account_health_guard_recovery_threshold"`
	HealthyLatencyMs         int64             `json:"account_health_guard_healthy_latency_ms"`
	AccountIDs               []int64           `json:"account_health_guard_account_ids"`
	AccountModels            map[int64]string  `json:"account_health_guard_account_models"`
	PlatformModels           map[string]string `json:"account_health_guard_platform_models"`
	PlatformLatencyMs        map[string]int64  `json:"account_health_guard_platform_latency_ms"`
	AccountIntervals         map[int64]int     `json:"account_health_guard_account_intervals"`
	CursorAccountID          int64             `json:"account_health_guard_cursor_account_id"`
}

type SupplierAccountHealthGuardSource struct {
	ProviderID          int64  `json:"provider_id"`
	ProviderName        string `json:"provider_name"`
	ProviderAccountID   int64  `json:"supplier_provider_account_id"`
	UpstreamAccountKey  string `json:"upstream_account_key"`
	UpstreamAccountName string `json:"upstream_account_name"`
}

type SupplierAccountHealthGuardCandidate struct {
	Source            SupplierAccountHealthGuardSource `json:"source"`
	MatchStatus       string                           `json:"match_status"`
	MatchCount        int                              `json:"match_count"`
	LocalAccountID    int64                            `json:"local_account_id"`
	PlatformOverride  string                           `json:"platform_override,omitempty"`
	EffectivePlatform string                           `json:"effective_platform"`
	LocalAccount      *Account                         `json:"-"`
}

type SupplierAccountHealthGuardSkippedAccount struct {
	LocalAccountID      int64  `json:"local_account_id,omitempty"`
	LocalAccountName    string `json:"local_account_name,omitempty"`
	ProviderAccountID   int64  `json:"supplier_provider_account_id,omitempty"`
	UpstreamAccountName string `json:"upstream_account_name,omitempty"`
}

type SupplierAccountHealthGuardSkipReason struct {
	Reason         string                                     `json:"reason"`
	Count          int                                        `json:"count"`
	SampleAccounts []SupplierAccountHealthGuardSkippedAccount `json:"sample_accounts,omitempty"`
}

type SupplierAccountHealthGuardRunItem struct {
	LocalAccountID     int64                              `json:"local_account_id"`
	LocalAccountName   string                             `json:"local_account_name"`
	Platform           string                             `json:"platform"`
	Sources            []SupplierAccountHealthGuardSource `json:"sources,omitempty"`
	MatchStatus        string                             `json:"match_status,omitempty"`
	ModelID            string                             `json:"model_id,omitempty"`
	SchedulableBefore  bool                               `json:"schedulable_before"`
	SchedulableAfter   bool                               `json:"schedulable_after"`
	Status             string                             `json:"status"`
	TestStatus         string                             `json:"test_status,omitempty"`
	LatencyMs          int64                              `json:"latency_ms"`
	LatencyLimitMs     int64                              `json:"latency_limit_ms"`
	ConsecutiveFailed  int                                `json:"consecutive_failed"`
	ConsecutiveSlow    int                                `json:"consecutive_slow"`
	ConsecutiveHealthy int                                `json:"consecutive_healthy"`
	Action             string                             `json:"action"`
	Reason             string                             `json:"reason,omitempty"`
	ErrorMessage       string                             `json:"error_message,omitempty"`
	StartedAt          time.Time                          `json:"started_at"`
	FinishedAt         time.Time                          `json:"finished_at"`
}

type SupplierAccountHealthGuardResult struct {
	TotalAccounts    int                                    `json:"total_accounts"`
	SelectedCount    int                                    `json:"selected_count"`
	CheckedCount     int                                    `json:"checked_count"`
	HealthyCount     int                                    `json:"healthy_count"`
	SlowCount        int                                    `json:"slow_count"`
	FailedCount      int                                    `json:"failed_count"`
	SkippedCount     int                                    `json:"skipped_count"`
	UnavailableCount int                                    `json:"unavailable_count"`
	PendingCount     int                                    `json:"pending_count"`
	DisabledCount    int                                    `json:"disabled_count"`
	RecoveredCount   int                                    `json:"recovered_count"`
	UnchangedCount   int                                    `json:"unchanged_count"`
	CursorAccountID  int64                                  `json:"cursor_account_id"`
	SkipReasons      []SupplierAccountHealthGuardSkipReason `json:"skip_reasons,omitempty"`
	Items            []SupplierAccountHealthGuardRunItem    `json:"items"`
}

type SupplierAccountHealthGuardRepository interface {
	ListAccountHealthGuardCandidates(ctx context.Context) ([]SupplierAccountHealthGuardCandidate, error)
}

type supplierAccountHealthGuardAccountStore interface {
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
	SetSchedulable(ctx context.Context, id int64, schedulable bool) error
}

type supplierAccountHealthGuardTester interface {
	runTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

type SupplierAccountHealthGuardRunner interface {
	Run(ctx context.Context, config SupplierAccountHealthGuardConfig, now time.Time) (SupplierAccountHealthGuardResult, error)
}

type SupplierAccountHealthGuardService struct {
	repository       SupplierAccountHealthGuardRepository
	accountStore     supplierAccountHealthGuardAccountStore
	tester           supplierAccountHealthGuardTester
	historyRecorder  SupplierAccountHealthHistoryRecorder
}

type supplierAccountHealthGuardTarget struct {
	account  Account
	sources  []SupplierAccountHealthGuardSource
	platform string
	modelID  string
}

type supplierAccountHealthGuardSkipCollector struct {
	order   []string
	reasons map[string]*SupplierAccountHealthGuardSkipReason
}

func NewSupplierAccountHealthGuardService(repository SupplierAccountHealthGuardRepository, accountStore supplierAccountHealthGuardAccountStore, tester supplierAccountHealthGuardTester) *SupplierAccountHealthGuardService {
	return &SupplierAccountHealthGuardService{repository: repository, accountStore: accountStore, tester: tester}
}

func (s *SupplierAccountHealthGuardService) SetHistoryRecorder(recorder SupplierAccountHealthHistoryRecorder) {
	if s != nil {
		s.historyRecorder = recorder
	}
}

func (s *SupplierAccountHealthGuardService) Run(ctx context.Context, config SupplierAccountHealthGuardConfig, now time.Time) (SupplierAccountHealthGuardResult, error) {
	config = normalizeSupplierAccountHealthGuardConfig(config)
	if len(config.AccountIDs) == 0 {
		return SupplierAccountHealthGuardResult{}, errors.New("请至少选择一个需要检查的账号")
	}
	if s == nil || s.repository == nil || s.accountStore == nil || s.tester == nil {
		return SupplierAccountHealthGuardResult{}, errors.New("供应商账号健康守护依赖未初始化")
	}
	candidates, err := s.repository.ListAccountHealthGuardCandidates(ctx)
	if err != nil {
		return SupplierAccountHealthGuardResult{}, err
	}

	selectedIDs := make(map[int64]struct{}, len(config.AccountIDs))
	candidatesByID := make(map[int64][]SupplierAccountHealthGuardCandidate, len(config.AccountIDs))
	for _, accountID := range config.AccountIDs {
		selectedIDs[accountID] = struct{}{}
	}
	for _, candidate := range candidates {
		if _, selected := selectedIDs[candidate.LocalAccountID]; !selected {
			continue
		}
		candidatesByID[candidate.LocalAccountID] = append(candidatesByID[candidate.LocalAccountID], candidate)
	}

	result := SupplierAccountHealthGuardResult{
		TotalAccounts: len(config.AccountIDs),
		Items:         make([]SupplierAccountHealthGuardRunItem, 0, len(config.AccountIDs)),
	}
	targets := make([]supplierAccountHealthGuardTarget, 0, len(config.AccountIDs))
	for _, accountID := range config.AccountIDs {
		accountCandidates := candidatesByID[accountID]
		target, available := supplierAccountHealthGuardBuildTarget(accountID, accountCandidates, config)
		if !available {
			result.Items = append(result.Items, supplierAccountHealthGuardUnavailableItem(accountID, accountCandidates, now))
			continue
		}
		targets = append(targets, target)
	}
	targets, notDueItems := supplierAccountHealthGuardFilterNotDue(targets, config, now)
	if len(notDueItems) > 0 {
		result.Items = append(result.Items, notDueItems...)
		result.SkipReasons = append(result.SkipReasons, supplierAccountHealthGuardNotDueSkipReasons(notDueItems)...)
	}
	missingModels := make([]string, 0)
	for _, target := range targets {
		if target.modelID == "" {
			accountName := strings.TrimSpace(target.account.Name)
			if accountName == "" {
				accountName = fmt.Sprintf("账号 #%d", target.account.ID)
			}
			missingModels = append(missingModels, accountName)
		}
	}
	if len(missingModels) > 0 {
		return SupplierAccountHealthGuardResult{}, fmt.Errorf("以下账号尚未配置测试模型：%s", strings.Join(missingModels, "、"))
	}

	selected := supplierAccountHealthGuardSelectTargets(targets, config.CursorAccountID, config.MaxAccountsPerRun)
	result.SelectedCount = len(selected)
	result.PendingCount = len(targets) - len(selected)
	if len(selected) > 0 {
		result.CursorAccountID = selected[len(selected)-1].account.ID
	} else {
		result.CursorAccountID = config.CursorAccountID
	}
	runItems := s.runTargets(ctx, config, now, selected)
	result.Items = append(result.Items, runItems...)
	cancelledSkips := make([]SupplierAccountHealthGuardRunItem, 0)
	for _, item := range runItems {
		if item.Status == SupplierAccountHealthGuardStatusSkipped && item.Reason == "任务时间不足，本次跳过" {
			cancelledSkips = append(cancelledSkips, item)
		}
	}
	if len(cancelledSkips) > 0 {
		result.SkipReasons = append(result.SkipReasons, supplierAccountHealthGuardCancelledSkipReasons(cancelledSkips)...)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		return result.Items[i].LocalAccountID < result.Items[j].LocalAccountID
	})
	for _, item := range result.Items {
		switch item.Status {
		case SupplierAccountHealthGuardStatusHealthy:
			result.CheckedCount++
			result.HealthyCount++
		case SupplierAccountHealthGuardStatusSlow:
			result.CheckedCount++
			result.SlowCount++
		case SupplierAccountHealthGuardStatusFailed:
			result.CheckedCount++
			result.FailedCount++
		case SupplierAccountHealthGuardStatusSkipped:
			result.SkippedCount++
		case SupplierAccountHealthGuardStatusUnavailable:
			result.UnavailableCount++
		}
		switch item.Action {
		case SupplierAccountHealthGuardActionDisabled:
			result.DisabledCount++
		case SupplierAccountHealthGuardActionRecovered:
			result.RecoveredCount++
		case SupplierAccountHealthGuardActionNone:
			if item.Status == SupplierAccountHealthGuardStatusHealthy || item.Status == SupplierAccountHealthGuardStatusSlow || item.Status == SupplierAccountHealthGuardStatusFailed {
				result.UnchangedCount++
			}
		}
	}
	return result, nil
}

func supplierAccountHealthGuardBuildTarget(accountID int64, candidates []SupplierAccountHealthGuardCandidate, config SupplierAccountHealthGuardConfig) (supplierAccountHealthGuardTarget, bool) {
	var target supplierAccountHealthGuardTarget
	for _, candidate := range candidates {
		if candidate.MatchStatus != SupplierAccountHealthGuardMatchMatched || candidate.LocalAccountID != accountID || candidate.LocalAccount == nil || candidate.LocalAccount.ID != accountID {
			continue
		}
		if strings.TrimSpace(candidate.LocalAccount.Status) != StatusActive {
			continue
		}
		if target.account.ID == 0 {
			target.account = *candidate.LocalAccount
			target.platform = supplierAccountHealthGuardPlatformForCandidate(candidate)
			target.modelID = supplierAccountHealthGuardModelForAccount(config, accountID, target.platform)
		}
		target.sources = append(target.sources, candidate.Source)
	}
	return target, target.account.ID > 0
}

func supplierAccountHealthGuardUnavailableItem(accountID int64, candidates []SupplierAccountHealthGuardCandidate, now time.Time) SupplierAccountHealthGuardRunItem {
	item := SupplierAccountHealthGuardRunItem{
		LocalAccountID: accountID,
		Status:         SupplierAccountHealthGuardStatusUnavailable,
		Action:         SupplierAccountHealthGuardActionNone,
		Reason:         "账号当前不可用",
		StartedAt:      now,
		FinishedAt:     now,
	}
	if len(candidates) > 0 {
		item.MatchStatus = candidates[0].MatchStatus
		item.Reason = "账号匹配已失效"
	}
	for _, candidate := range candidates {
		item.Sources = append(item.Sources, candidate.Source)
		if candidate.LocalAccount == nil {
			continue
		}
		item.LocalAccountName = candidate.LocalAccount.Name
		item.Platform = supplierAccountHealthGuardPlatformForCandidate(candidate)
		item.SchedulableBefore = candidate.LocalAccount.Schedulable
		item.SchedulableAfter = candidate.LocalAccount.Schedulable
		if strings.TrimSpace(candidate.LocalAccount.Status) != StatusActive {
			item.Reason = "账号已停用"
		}
	}
	return item
}

func (s *SupplierAccountHealthGuardService) runTargets(ctx context.Context, config SupplierAccountHealthGuardConfig, now time.Time, targets []supplierAccountHealthGuardTarget) []SupplierAccountHealthGuardRunItem {
	if len(targets) == 0 {
		return nil
	}
	workers := config.Concurrency
	if workers > len(targets) {
		workers = len(targets)
	}
	type job struct {
		index  int
		target supplierAccountHealthGuardTarget
	}
	jobs := make(chan job)
	items := make([]SupplierAccountHealthGuardRunItem, len(targets))
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for current := range jobs {
				items[current.index] = s.runTarget(ctx, config, now, current.target)
			}
		}()
	}
	for index, target := range targets {
		jobs <- job{index: index, target: target}
	}
	close(jobs)
	wait.Wait()
	return items
}

func (s *SupplierAccountHealthGuardService) runTarget(ctx context.Context, config SupplierAccountHealthGuardConfig, now time.Time, target supplierAccountHealthGuardTarget) SupplierAccountHealthGuardRunItem {
	startedAt := now
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	if ctx.Err() != nil {
		return SupplierAccountHealthGuardRunItem{
			LocalAccountID: target.account.ID, LocalAccountName: target.account.Name, Platform: target.platform,
			Sources:           append([]SupplierAccountHealthGuardSource(nil), target.sources...),
			ModelID:           target.modelID,
			SchedulableBefore: target.account.Schedulable,
			SchedulableAfter:  target.account.Schedulable,
			Status:            SupplierAccountHealthGuardStatusSkipped,
			Action:            SupplierAccountHealthGuardActionNone,
			Reason:            "任务时间不足，本次跳过",
			StartedAt:         startedAt,
			FinishedAt:        time.Now(),
		}
	}
	item := SupplierAccountHealthGuardRunItem{
		LocalAccountID: target.account.ID, LocalAccountName: target.account.Name, Platform: target.platform,
		Sources: append([]SupplierAccountHealthGuardSource(nil), target.sources...), ModelID: target.modelID,
		SchedulableBefore: target.account.Schedulable, SchedulableAfter: target.account.Schedulable,
		Action: SupplierAccountHealthGuardActionNone, StartedAt: startedAt,
		ConsecutiveFailed:  supplierAccountHealthGuardExtraInt(target.account.Extra, supplierHealthGuardFailureCountExtraKey),
		ConsecutiveSlow:    supplierAccountHealthGuardExtraInt(target.account.Extra, supplierHealthGuardSlowCountExtraKey),
		ConsecutiveHealthy: supplierAccountHealthGuardExtraInt(target.account.Extra, supplierHealthGuardHealthyCountExtraKey),
	}
	originalFailureCount := item.ConsecutiveFailed
	item.LatencyLimitMs = supplierAccountHealthGuardLatencyLimitForPlatform(config, target.platform)
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutPerAccountSeconds)*time.Second)
	result, runErr := s.tester.runTestBackground(testCtx, target.account.ID, target.modelID)
	contextErr := testCtx.Err()
	cancel()
	item.FinishedAt = time.Now()
	if result != nil {
		item.TestStatus = result.Status
		item.LatencyMs = result.LatencyMs
		if !result.StartedAt.IsZero() {
			item.StartedAt = result.StartedAt
		}
		if !result.FinishedAt.IsZero() {
			item.FinishedAt = result.FinishedAt
		}
	}
	item.Status, item.Reason = supplierAccountHealthGuardEvaluateResult(contextErr, runErr, result, item.LatencyMs, item.LatencyLimitMs)
	if item.Status == SupplierAccountHealthGuardStatusFailed && runErr != nil {
		item.ErrorMessage = runErr.Error()
	} else if item.Status == SupplierAccountHealthGuardStatusFailed && result != nil {
		item.ErrorMessage = strings.TrimSpace(result.ErrorMessage)
	}
	switch item.Status {
	case SupplierAccountHealthGuardStatusHealthy:
		item.ConsecutiveHealthy++
		item.ConsecutiveFailed = 0
		item.ConsecutiveSlow = 0
	case SupplierAccountHealthGuardStatusSlow:
		item.ConsecutiveSlow++
		item.ConsecutiveHealthy = 0
		item.ConsecutiveFailed = 0
	default:
		item.ConsecutiveFailed++
		item.ConsecutiveHealthy = 0
		item.ConsecutiveSlow = 0
	}
	item.SchedulableAfter, item.Action, item.Reason = supplierAccountHealthGuardNextSchedulingState(config, item)
	if item.SchedulableAfter != item.SchedulableBefore {
		if err := s.accountStore.SetSchedulable(ctx, item.LocalAccountID, item.SchedulableAfter); err != nil {
			item.SchedulableAfter = item.SchedulableBefore
			item.Action = SupplierAccountHealthGuardActionNone
			supplierAccountHealthGuardMarkWriteFailure(&item, originalFailureCount, "更新调度状态失败", err)
		}
	}
	if err := s.accountStore.UpdateExtra(ctx, item.LocalAccountID, map[string]any{
		supplierHealthGuardFailureCountExtraKey:     item.ConsecutiveFailed,
		supplierHealthGuardSlowCountExtraKey:        item.ConsecutiveSlow,
		supplierHealthGuardHealthyCountExtraKey:     item.ConsecutiveHealthy,
		supplierHealthGuardLastStatusExtraKey:       item.Status,
		supplierHealthGuardLastLatencyMsExtraKey:    item.LatencyMs,
		supplierHealthGuardLastCheckedAtExtraKey:    item.FinishedAt.UTC().Format(time.RFC3339),
		supplierHealthGuardLastActionExtraKey:       item.Action,
		supplierHealthGuardLastMessageExtraKey:      item.Reason,
		supplierHealthGuardLastTestModelExtraKey:    item.ModelID,
		supplierHealthGuardLastLatencyLimitExtraKey: item.LatencyLimitMs,
	}); err != nil {
		supplierAccountHealthGuardMarkWriteFailure(&item, originalFailureCount, "更新健康守护状态失败", err)
	}
	if s.historyRecorder != nil && (item.Status == SupplierAccountHealthGuardStatusHealthy || item.Status == SupplierAccountHealthGuardStatusSlow || item.Status == SupplierAccountHealthGuardStatusFailed) {
		if err := s.historyRecorder.Save(ctx, supplierAccountHealthHistoryRecordFromRunItem(item)); err != nil {
			item.ErrorMessage = supplierAccountHealthGuardAppendMessage(item.ErrorMessage, fmt.Sprintf("保存健康历史失败: %v", err))
		}
	}
	return item
}

func supplierAccountHealthHistoryRecordFromRunItem(item SupplierAccountHealthGuardRunItem) SupplierAccountHealthHistoryRecord {
	var source SupplierAccountHealthGuardSource
	if len(item.Sources) > 0 {
		source = item.Sources[0]
	}
	var latency *int64
	if item.Status != SupplierAccountHealthGuardStatusFailed {
		value := item.LatencyMs
		latency = &value
	}
	return SupplierAccountHealthHistoryRecord{
		LocalAccountID: item.LocalAccountID, LocalAccountName: item.LocalAccountName,
		ProviderID: source.ProviderID, ProviderName: source.ProviderName, Platform: item.Platform,
		CheckedAt: item.FinishedAt, StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
		Status: item.Status, LatencyMs: latency, LatencyLimitMs: item.LatencyLimitMs, ModelID: item.ModelID,
		SchedulableBefore: item.SchedulableBefore, SchedulableAfter: item.SchedulableAfter, Action: item.Action,
		ConsecutiveFailed: item.ConsecutiveFailed, ConsecutiveSlow: item.ConsecutiveSlow,
		ConsecutiveHealthy: item.ConsecutiveHealthy, Reason: item.Reason, ErrorMessage: item.ErrorMessage,
	}
}

func supplierAccountHealthGuardMarkWriteFailure(item *SupplierAccountHealthGuardRunItem, originalFailureCount int, reason string, err error) {
	item.Status = SupplierAccountHealthGuardStatusFailed
	item.ConsecutiveFailed = originalFailureCount + 1
	item.ConsecutiveSlow = 0
	item.ConsecutiveHealthy = 0
	item.Reason = reason
	item.ErrorMessage = supplierAccountHealthGuardAppendMessage(item.ErrorMessage, fmt.Sprintf("%s: %v", reason, err))
}

func supplierAccountHealthGuardSelectTargets(targets []supplierAccountHealthGuardTarget, cursor int64, limit int) []supplierAccountHealthGuardTarget {
	if len(targets) == 0 || limit <= 0 {
		return nil
	}
	if limit > len(targets) {
		limit = len(targets)
	}
	start := 0
	for index, target := range targets {
		if target.account.ID > cursor {
			start = index
			break
		}
		if index == len(targets)-1 {
			start = 0
		}
	}
	selected := make([]supplierAccountHealthGuardTarget, 0, limit)
	for offset := 0; offset < limit; offset++ {
		selected = append(selected, targets[(start+offset)%len(targets)])
	}
	return selected
}

func supplierAccountHealthGuardSkippedItem(candidate SupplierAccountHealthGuardCandidate, reason string, now time.Time) SupplierAccountHealthGuardRunItem {
	item := SupplierAccountHealthGuardRunItem{
		LocalAccountID: candidate.LocalAccountID, Sources: []SupplierAccountHealthGuardSource{candidate.Source},
		MatchStatus: candidate.MatchStatus, Status: SupplierAccountHealthGuardStatusSkipped,
		Action: SupplierAccountHealthGuardActionNone, Reason: reason, StartedAt: now, FinishedAt: now,
	}
	if candidate.LocalAccount != nil {
		item.LocalAccountName = candidate.LocalAccount.Name
		item.Platform = supplierAccountHealthGuardPlatformForCandidate(candidate)
		item.SchedulableBefore = candidate.LocalAccount.Schedulable
		item.SchedulableAfter = candidate.LocalAccount.Schedulable
	}
	return item
}

func supplierAccountHealthGuardFilterNotDue(targets []supplierAccountHealthGuardTarget, config SupplierAccountHealthGuardConfig, now time.Time) ([]supplierAccountHealthGuardTarget, []SupplierAccountHealthGuardRunItem) {
	out := make([]supplierAccountHealthGuardTarget, 0, len(targets))
	notDue := make([]SupplierAccountHealthGuardRunItem, 0)
	for _, target := range targets {
		interval := config.AccountIntervals[target.account.ID]
		if interval <= 0 {
			out = append(out, target)
			continue
		}
		lastCheckedAt := supplierAccountHealthGuardLastCheckedAt(target.account.Extra)
		if !lastCheckedAt.IsZero() && now.Sub(lastCheckedAt) < time.Duration(interval)*time.Second {
			notDue = append(notDue, SupplierAccountHealthGuardRunItem{
				LocalAccountID:    target.account.ID,
				LocalAccountName:  target.account.Name,
				Platform:          target.platform,
				Sources:           append([]SupplierAccountHealthGuardSource(nil), target.sources...),
				ModelID:           target.modelID,
				SchedulableBefore: target.account.Schedulable,
				SchedulableAfter:  target.account.Schedulable,
				Status:            SupplierAccountHealthGuardStatusSkipped,
				Action:            SupplierAccountHealthGuardActionNone,
				Reason:            fmt.Sprintf("距上次检查不足 %d 秒", interval),
				StartedAt:         now,
				FinishedAt:        now,
			})
			continue
		}
		out = append(out, target)
	}
	return out, notDue
}

func supplierAccountHealthGuardLastCheckedAt(extra map[string]any) time.Time {
	if extra == nil {
		return time.Time{}
	}
	raw, _ := extra[supplierHealthGuardLastCheckedAtExtraKey].(string)
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func supplierAccountHealthGuardNotDueSkipReasons(items []SupplierAccountHealthGuardRunItem) []SupplierAccountHealthGuardSkipReason {
	if len(items) == 0 {
		return nil
	}
	reason := SupplierAccountHealthGuardSkipReason{Reason: "未到检查间隔"}
	for _, item := range items {
		reason.Count++
		if len(reason.SampleAccounts) >= supplierAccountHealthGuardSkipReasonSampleLimit {
			continue
		}
		reason.SampleAccounts = append(reason.SampleAccounts, SupplierAccountHealthGuardSkippedAccount{
			LocalAccountID:   item.LocalAccountID,
			LocalAccountName: item.LocalAccountName,
		})
	}
	return []SupplierAccountHealthGuardSkipReason{reason}
}

func supplierAccountHealthGuardCancelledSkipReasons(items []SupplierAccountHealthGuardRunItem) []SupplierAccountHealthGuardSkipReason {
	if len(items) == 0 {
		return nil
	}
	reason := SupplierAccountHealthGuardSkipReason{Reason: "任务时间不足"}
	for _, item := range items {
		reason.Count++
		if len(reason.SampleAccounts) >= supplierAccountHealthGuardSkipReasonSampleLimit {
			continue
		}
		reason.SampleAccounts = append(reason.SampleAccounts, SupplierAccountHealthGuardSkippedAccount{
			LocalAccountID:   item.LocalAccountID,
			LocalAccountName: item.LocalAccountName,
		})
	}
	return []SupplierAccountHealthGuardSkipReason{reason}
}

func newSupplierAccountHealthGuardSkipCollector() *supplierAccountHealthGuardSkipCollector {
	return &supplierAccountHealthGuardSkipCollector{reasons: make(map[string]*SupplierAccountHealthGuardSkipReason)}
}

func (c *supplierAccountHealthGuardSkipCollector) Add(reason string, candidate SupplierAccountHealthGuardCandidate) {
	entry := c.reasons[reason]
	if entry == nil {
		entry = &SupplierAccountHealthGuardSkipReason{Reason: reason}
		c.reasons[reason] = entry
		c.order = append(c.order, reason)
	}
	entry.Count++
	if len(entry.SampleAccounts) < supplierAccountHealthGuardSkipReasonSampleLimit {
		sample := SupplierAccountHealthGuardSkippedAccount{
			LocalAccountID: candidate.LocalAccountID, ProviderAccountID: candidate.Source.ProviderAccountID,
			UpstreamAccountName: candidate.Source.UpstreamAccountName,
		}
		if candidate.LocalAccount != nil {
			sample.LocalAccountName = candidate.LocalAccount.Name
		}
		entry.SampleAccounts = append(entry.SampleAccounts, sample)
	}
}

func (c *supplierAccountHealthGuardSkipCollector) List() []SupplierAccountHealthGuardSkipReason {
	out := make([]SupplierAccountHealthGuardSkipReason, 0, len(c.order))
	for _, reason := range c.order {
		out = append(out, *c.reasons[reason])
	}
	return out
}

func normalizeSupplierAccountHealthGuardConfig(config SupplierAccountHealthGuardConfig) SupplierAccountHealthGuardConfig {
	if config.MaxAccountsPerRun <= 0 {
		config.MaxAccountsPerRun = DefaultSupplierAccountHealthGuardMaxAccountsPerRun
	}
	if config.MaxAccountsPerRun > MaxSupplierAccountHealthGuardMaxAccountsPerRun {
		config.MaxAccountsPerRun = MaxSupplierAccountHealthGuardMaxAccountsPerRun
	}
	if config.Concurrency <= 0 {
		config.Concurrency = DefaultSupplierAccountHealthGuardConcurrency
	}
	if config.Concurrency > MaxSupplierAccountHealthGuardConcurrency {
		config.Concurrency = MaxSupplierAccountHealthGuardConcurrency
	}
	if config.TimeoutPerAccountSeconds <= 0 {
		config.TimeoutPerAccountSeconds = DefaultSupplierAccountHealthGuardTimeoutPerAccountSeconds
	}
	if config.TimeoutPerAccountSeconds > MaxSupplierAccountHealthGuardTimeoutPerAccountSeconds {
		config.TimeoutPerAccountSeconds = MaxSupplierAccountHealthGuardTimeoutPerAccountSeconds
	}
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultSupplierAccountHealthGuardFailureThreshold
	}
	if config.SlowThreshold <= 0 {
		config.SlowThreshold = DefaultSupplierAccountHealthGuardSlowThreshold
	}
	if config.RecoveryThreshold <= 0 {
		config.RecoveryThreshold = DefaultSupplierAccountHealthGuardRecoveryThreshold
	}
	if config.HealthyLatencyMs <= 0 {
		config.HealthyLatencyMs = DefaultSupplierAccountHealthGuardHealthyLatencyMs
	}
	config.AccountIDs = normalizeSupplierAccountHealthGuardAccountIDs(config.AccountIDs)
	config.AccountModels = normalizeSupplierAccountHealthGuardAccountModels(config.AccountModels)
	config.PlatformModels = normalizeSupplierAccountHealthGuardPlatformModels(config.PlatformModels)
	config.PlatformLatencyMs = normalizeSupplierAccountHealthGuardPlatformLatency(config.PlatformLatencyMs)
	config.AccountIntervals = normalizeSupplierAccountHealthGuardAccountIntervals(config.AccountIntervals)
	return config
}

func normalizeSupplierAccountHealthGuardAccountIDs(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, accountID := range values {
		if accountID <= 0 {
			continue
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		out = append(out, accountID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeSupplierAccountHealthGuardAccountModels(values map[int64]string) map[int64]string {
	out := make(map[int64]string)
	for accountID, model := range values {
		model = strings.TrimSpace(model)
		if accountID > 0 && model != "" {
			out[accountID] = model
		}
	}
	return out
}

func normalizeSupplierAccountHealthGuardPlatformModels(values map[string]string) map[string]string {
	out := make(map[string]string)
	for platform, model := range values {
		platform = strings.ToLower(strings.TrimSpace(platform))
		model = strings.TrimSpace(model)
		if platform != "" && model != "" {
			out[platform] = model
		}
	}
	return out
}

func normalizeSupplierAccountHealthGuardPlatformLatency(values map[string]int64) map[string]int64 {
	out := make(map[string]int64)
	for platform, latency := range values {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if platform != "" && latency > 0 {
			out[platform] = latency
		}
	}
	return out
}

func normalizeSupplierAccountHealthGuardAccountIntervals(values map[int64]int) map[int64]int {
	out := make(map[int64]int)
	for accountID, interval := range values {
		if accountID > 0 && interval >= MinSupplierAccountHealthGuardAccountIntervalSeconds {
			out[accountID] = interval
		}
	}
	return out
}

func supplierAccountHealthGuardPlatformForCandidate(candidate SupplierAccountHealthGuardCandidate) string {
	if platform := strings.TrimSpace(candidate.EffectivePlatform); platform != "" {
		return platform
	}
	if platform := strings.TrimSpace(candidate.PlatformOverride); platform != "" {
		return platform
	}
	if candidate.LocalAccount != nil {
		return strings.TrimSpace(candidate.LocalAccount.Platform)
	}
	return ""
}

func supplierAccountHealthGuardModelForAccount(config SupplierAccountHealthGuardConfig, accountID int64, platform string) string {
	if model := strings.TrimSpace(config.AccountModels[accountID]); model != "" {
		return model
	}
	return strings.TrimSpace(config.PlatformModels[strings.ToLower(strings.TrimSpace(platform))])
}

func supplierAccountHealthGuardLatencyLimitForPlatform(config SupplierAccountHealthGuardConfig, platform string) int64 {
	if latency := config.PlatformLatencyMs[strings.ToLower(strings.TrimSpace(platform))]; latency > 0 {
		return latency
	}
	return config.HealthyLatencyMs
}

func supplierAccountHealthGuardEvaluateResult(contextErr error, runErr error, result *ScheduledTestResult, latencyMs, latencyLimitMs int64) (string, string) {
	if errors.Is(contextErr, context.DeadlineExceeded) {
		return SupplierAccountHealthGuardStatusFailed, "测试超时"
	}
	if errors.Is(contextErr, context.Canceled) {
		return SupplierAccountHealthGuardStatusFailed, "测试已取消"
	}
	if runErr != nil {
		return SupplierAccountHealthGuardStatusFailed, runErr.Error()
	}
	if result == nil {
		return SupplierAccountHealthGuardStatusFailed, "测试结果为空"
	}
	if strings.TrimSpace(result.Status) != "success" {
		if message := strings.TrimSpace(result.ErrorMessage); message != "" {
			return SupplierAccountHealthGuardStatusFailed, message
		}
		return SupplierAccountHealthGuardStatusFailed, "测试失败"
	}
	if latencyLimitMs > 0 && latencyMs > latencyLimitMs {
		return SupplierAccountHealthGuardStatusSlow, fmt.Sprintf("响应耗时 %dms 超过阈值 %dms", latencyMs, latencyLimitMs)
	}
	return SupplierAccountHealthGuardStatusHealthy, "测试通过"
}

func supplierAccountHealthGuardNextSchedulingState(config SupplierAccountHealthGuardConfig, item SupplierAccountHealthGuardRunItem) (bool, string, string) {
	switch item.Status {
	case SupplierAccountHealthGuardStatusHealthy:
		if !item.SchedulableBefore && item.ConsecutiveHealthy >= config.RecoveryThreshold {
			return true, SupplierAccountHealthGuardActionRecovered, fmt.Sprintf("连续健康 %d 次", item.ConsecutiveHealthy)
		}
	case SupplierAccountHealthGuardStatusSlow:
		if item.SchedulableBefore && item.ConsecutiveSlow >= config.SlowThreshold {
			return false, SupplierAccountHealthGuardActionDisabled, fmt.Sprintf("连续慢响应 %d 次", item.ConsecutiveSlow)
		}
	case SupplierAccountHealthGuardStatusFailed:
		if item.SchedulableBefore && item.ConsecutiveFailed >= config.FailureThreshold {
			return false, SupplierAccountHealthGuardActionDisabled, fmt.Sprintf("连续失败 %d 次", item.ConsecutiveFailed)
		}
	}
	return item.SchedulableBefore, SupplierAccountHealthGuardActionNone, item.Reason
}

func supplierAccountHealthGuardExtraInt(extra map[string]any, key string) int {
	if extra == nil {
		return 0
	}
	return parseExtraInt(extra[key])
}

func supplierAccountHealthGuardAppendMessage(current, next string) string {
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
