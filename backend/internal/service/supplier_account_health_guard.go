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
	AccountSchedulingChange  map[int64]bool    `json:"account_health_guard_account_scheduling_change"`
	// 账号级阈值覆盖：键为本地账号 ID，未出现的账号回落到上面的全局阈值。
	// 存独立映射而不是「账号 → 整组阈值」，是为了让同一账号只覆盖部分阈值时不会隐式冻结另外两项。
	AccountFailureThresholds  map[int64]int `json:"account_health_guard_account_failure_thresholds"`
	AccountSlowThresholds     map[int64]int `json:"account_health_guard_account_slow_thresholds"`
	AccountRecoveryThresholds map[int64]int `json:"account_health_guard_account_recovery_thresholds"`
	// 未开调度账号按平台倍率区间取检查间隔：总开关开启后，!Schedulable 的账号改由
	// PlatformMultiplierIntervals 命中的区间决定间隔，覆盖账号级 AccountIntervals；
	// 未命中任何区间则不设间隔（每轮都测）。已开调度账号不受影响，仍走 AccountIntervals。
	PlatformMultiplierIntervalsEnabled bool                                                      `json:"account_health_guard_platform_multiplier_intervals_enabled"`
	PlatformMultiplierIntervals        map[string][]SupplierAccountHealthGuardMultiplierInterval `json:"account_health_guard_platform_multiplier_intervals"`
	CursorAccountID                    int64                                                     `json:"account_health_guard_cursor_account_id"`
}

// SupplierAccountHealthGuardMultiplierInterval 描述一个「倍率区间 → 检查间隔」规则。
// 区间取 [MinMultiplier, MaxMultiplier)：下界含、上界不含；MaxMultiplier<=0 表示无上界。
type SupplierAccountHealthGuardMultiplierInterval struct {
	MinMultiplier   float64 `json:"min_multiplier"`
	MaxMultiplier   float64 `json:"max_multiplier"`
	IntervalSeconds int     `json:"interval_seconds"`
}

type SupplierAccountHealthGuardSource struct {
	ProviderID          int64  `json:"provider_id"`
	ProviderName        string `json:"provider_name"`
	ProviderAccountID   int64  `json:"supplier_provider_account_id"`
	UpstreamAccountKey  string `json:"upstream_account_key"`
	UpstreamAccountName string `json:"upstream_account_name"`
	// RateMultiplier 供应商侧账号的计费倍率（supplier_provider_accounts.rate_multiplier，由供应商数据同步任务写入）。
	// 未开调度账号按倍率区间取检查间隔、明细按倍率排序，用的都是这个供应商倍率，
	// 而不是本地账号的 accounts.rate_multiplier（后者是本地计费口径，实测常年为 1.0，落桶时会全挤进同一个区间）。
	RateMultiplier float64 `json:"rate_multiplier"`
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
	// IntervalSeconds 该账号本轮生效的检查间隔（秒）；0 表示每轮都测。
	// 用指针是为了区分「不适用」——不可用 / 未匹配 / 已停用的账号不参与本轮检查，
	// 给它们显示一个用不上的间隔会误导，那类行该字段为 nil。
	IntervalSeconds *int `json:"interval_seconds,omitempty"`
	// NextCheckAt 下次检查时间 = 上次检查时间 + 间隔。从未检查过、或每轮都测时为空。
	NextCheckAt *time.Time `json:"next_check_at,omitempty"`
	// BillingRateMultiplier 该账号的供应商侧计费倍率（取自匹配到的第一个供应商来源），明细列表按它升序排序。
	// 与 supplierAccountHealthGuardResolveInterval 用的是同一个值：倍率区间规则就是按它落桶的，
	// 两处取不同的值会出现「排序显示在同一桶、间隔却按另一桶算」的错乱。
	// 用指针是为了区分「未知」——账号已查不到、没有任何供应商来源时没有倍率可言；0 是合法值（计费为 0），
	// 靠 omitempty 省掉会把 0 变成「未知」，所以必须是 *float64。
	BillingRateMultiplier *float64  `json:"billing_rate_multiplier,omitempty"`
	StartedAt             time.Time `json:"started_at"`
	FinishedAt            time.Time `json:"finished_at"`
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
	// SnapshotErrorMessage 记录分组监控快照写库失败的痕迹：快照只影响模型监控的「最新一刻」，
	// 写失败不该让整轮守护被判失败，但也不能无声吞掉。
	SnapshotErrorMessage string `json:"snapshot_error_message,omitempty"`
}

// SupplierAccountHealthGuardUnavailableCause 白名单账号在守护候选里查不到时的**具体成因**。
//
// 为什么需要它：候选查询的 WHERE 是 a.active = TRUE AND p.enabled = TRUE，
// 于是「供应商被停用」「上游账号已下线」「本地账号改名导致匹配不上」三种情况全都塌成
// 「候选为空」一条，明细表里统一显示「账号当前不可用」—— 排查时只能去翻库。
// 这个枚举就是为了让明细表能直接说清楚是哪一种。
type SupplierAccountHealthGuardUnavailableCause string

const (
	// 成因未知（诊断查询没查到 / 没跑）。此时沿用原来的笼统文案，行为与加枚举之前一致。
	SupplierAccountHealthGuardCauseUnknown SupplierAccountHealthGuardUnavailableCause = ""
	// 本地账号不存在或已软删。
	SupplierAccountHealthGuardCauseLocalMissing SupplierAccountHealthGuardUnavailableCause = "local_missing"
	// 没有任何供应商账号的名字能对得上（本地账号改名 / 上游账号改名 / 上游账号已删除）。
	SupplierAccountHealthGuardCauseNoProviderAccount SupplierAccountHealthGuardUnavailableCause = "no_provider_account"
	// 名字能配上，但同名的本地账号不止一个 ⇒ 候选取不出唯一一个，判为冲突。
	SupplierAccountHealthGuardCauseMatchConflict SupplierAccountHealthGuardUnavailableCause = "match_conflict"
	// 名字能配上、供应商也在启用，但供应商侧账号已停用或删除。
	SupplierAccountHealthGuardCauseProviderAccountInactive SupplierAccountHealthGuardUnavailableCause = "provider_account_inactive"
	// 名字能配上，但所属供应商被停用。
	SupplierAccountHealthGuardCauseProviderDisabled SupplierAccountHealthGuardUnavailableCause = "provider_disabled"
)

type SupplierAccountHealthGuardRepository interface {
	ListAccountHealthGuardCandidates(ctx context.Context) ([]SupplierAccountHealthGuardCandidate, error)
	// ListAccountHealthGuardUnavailableReasons 批量诊断「这些本地账号为什么在候选里查不到」。
	// 只在候选为空时才需要，正常路径不会调用；返回的 map 里没有的账号按成因未知处理。
	ListAccountHealthGuardUnavailableReasons(ctx context.Context, accountIDs []int64) (map[int64]SupplierAccountHealthGuardUnavailableCause, error)
	// RecordGroupMonitorSnapshots 记录「某时刻该分组正在调度的账号」的监控结果。
	// 给本接口加方法不会改变 ProvideSupplierAccountHealthGuardService 的参数列表，Wire 生成代码无需重新生成。
	RecordGroupMonitorSnapshots(ctx context.Context, source string) error
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
	repository      SupplierAccountHealthGuardRepository
	accountStore    supplierAccountHealthGuardAccountStore
	tester          supplierAccountHealthGuardTester
	historyRecorder SupplierAccountHealthHistoryRecorder
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
	// 不可用的账号先只攒起来、不急着生成明细行：要说清「为什么不可用」得再查一次库，
	// 而绝大多数账号是可用的，逐条查太浪费 ⇒ 攒完一次性批量查。
	var unavailable []supplierAccountHealthGuardUnavailableInput
	for _, accountID := range config.AccountIDs {
		accountCandidates := candidatesByID[accountID]
		target, available := supplierAccountHealthGuardBuildTarget(accountID, accountCandidates, config)
		if !available {
			unavailable = append(unavailable, supplierAccountHealthGuardUnavailableInput{
				accountID:  accountID,
				candidates: accountCandidates,
			})
			continue
		}
		targets = append(targets, target)
	}
	if len(unavailable) > 0 {
		causes := s.supplierAccountHealthGuardUnavailableCauses(ctx, unavailable)
		for _, input := range unavailable {
			result.Items = append(result.Items, supplierAccountHealthGuardUnavailableItem(input.accountID, input.candidates, causes[input.accountID], now))
		}
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
	// 守护跑完后才记快照：本轮刚改过的 schedulable 与健康历史都已落库，
	// 此刻采到的就是「现在真正在调度的账号」，顺序反了会记到换人之前的那条。
	if err := s.repository.RecordGroupMonitorSnapshots(ctx, SupplierGroupMonitorSnapshotSourceHealthGuard); err != nil {
		result.SnapshotErrorMessage = fmt.Sprintf("记录分组监控快照失败: %v", err)
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

// supplierAccountHealthGuardUnavailableInput 攒下来的「不可用账号」及其候选，
// 供批量诊断原因后再统一生成明细行。
type supplierAccountHealthGuardUnavailableInput struct {
	accountID  int64
	candidates []SupplierAccountHealthGuardCandidate
}

// supplierAccountHealthGuardUnavailableCauses 批量查「这些账号为什么在候选里查不到」。
//
// 只查候选为空的那部分：候选非空时成因已经能从候选举里看出来（本地账号被停用），
// 再查一次是白跑。诊断失败不能拖垮整轮守护 ⇒ 返回 nil，明细行退化成原来的笼统文案。
func (s *SupplierAccountHealthGuardService) supplierAccountHealthGuardUnavailableCauses(ctx context.Context, inputs []supplierAccountHealthGuardUnavailableInput) map[int64]SupplierAccountHealthGuardUnavailableCause {
	if s == nil || s.repository == nil {
		return nil
	}
	accountIDs := make([]int64, 0, len(inputs))
	for _, input := range inputs {
		if len(input.candidates) == 0 {
			accountIDs = append(accountIDs, input.accountID)
		}
	}
	if len(accountIDs) == 0 {
		return nil
	}
	causes, err := s.repository.ListAccountHealthGuardUnavailableReasons(ctx, accountIDs)
	if err != nil {
		return nil
	}
	return causes
}

// supplierAccountHealthGuardUnavailableText 把成因翻成明细表里给人看的话。
// 未知成因沿用「账号当前不可用」，保证诊断拿不到结论时显示与改造前完全一致。
func supplierAccountHealthGuardUnavailableText(cause SupplierAccountHealthGuardUnavailableCause) string {
	switch cause {
	case SupplierAccountHealthGuardCauseLocalMissing:
		return "本地账号已不存在"
	case SupplierAccountHealthGuardCauseNoProviderAccount:
		return "没有匹配的供应商账号"
	case SupplierAccountHealthGuardCauseMatchConflict:
		return "多个本地账号同名，匹配冲突"
	case SupplierAccountHealthGuardCauseProviderAccountInactive:
		return "供应商侧账号已下线"
	case SupplierAccountHealthGuardCauseProviderDisabled:
		return "所属供应商已停用"
	default:
		return "账号当前不可用"
	}
}

func supplierAccountHealthGuardUnavailableItem(accountID int64, candidates []SupplierAccountHealthGuardCandidate, cause SupplierAccountHealthGuardUnavailableCause, now time.Time) SupplierAccountHealthGuardRunItem {
	item := SupplierAccountHealthGuardRunItem{
		LocalAccountID: accountID,
		Status:         SupplierAccountHealthGuardStatusUnavailable,
		Action:         SupplierAccountHealthGuardActionNone,
		Reason:         supplierAccountHealthGuardUnavailableText(cause),
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
	// 倍率取第一个供应商来源的 rate_multiplier：明细按它排序，与落桶用的是同一个值。
	// 没有任何候选来源（账号已彻底查不到）时留空（nil），排序时排到最后而不是当成 0。
	item.BillingRateMultiplier = supplierAccountHealthGuardSourceMultiplier(item.Sources)
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
	// 间隔、上次检查时间与倍率在两条返回路径里都要带上：明细列表靠它们解释
	// 「这个账号多久测一次」「为什么这轮被跳过」「按倍率排序时它排在哪」，提前算一次比两处各算一遍安全。
	intervalSeconds := supplierAccountHealthGuardResolveInterval(config, target)
	nextCheckAt := supplierAccountHealthGuardNextCheckAt(
		supplierAccountHealthGuardLastCheckedAt(target.account.Extra), intervalSeconds,
	)
	multiplier := supplierAccountHealthGuardSourceMultiplier(target.sources)
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
			IntervalSeconds:   &intervalSeconds,
			NextCheckAt:       nextCheckAt,

			BillingRateMultiplier: multiplier,
			StartedAt:             startedAt,
			FinishedAt:            time.Now(),
		}
	}
	item := SupplierAccountHealthGuardRunItem{
		LocalAccountID: target.account.ID, LocalAccountName: target.account.Name, Platform: target.platform,
		Sources: append([]SupplierAccountHealthGuardSource(nil), target.sources...), ModelID: target.modelID,
		SchedulableBefore: target.account.Schedulable, SchedulableAfter: target.account.Schedulable,
		Action: SupplierAccountHealthGuardActionNone, StartedAt: startedAt,
		IntervalSeconds:       &intervalSeconds,
		NextCheckAt:           nextCheckAt,
		BillingRateMultiplier: multiplier,
		ConsecutiveFailed:     supplierAccountHealthGuardExtraInt(target.account.Extra, supplierHealthGuardFailureCountExtraKey),
		ConsecutiveSlow:       supplierAccountHealthGuardExtraInt(target.account.Extra, supplierHealthGuardSlowCountExtraKey),
		ConsecutiveHealthy:    supplierAccountHealthGuardExtraInt(target.account.Extra, supplierHealthGuardHealthyCountExtraKey),
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
	if value, exists := config.AccountSchedulingChange[item.LocalAccountID]; exists && !value {
		item.SchedulableAfter = item.SchedulableBefore
		item.Action = SupplierAccountHealthGuardActionNone
		item.Reason = "已关闭修改调度"
	}
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

// supplierAccountHealthGuardResolveInterval 决定单个 target 的检查间隔（秒，<=0 表示每轮都测）。
// 未开调度账号在总开关开启时以平台倍率规则为准，覆盖账号级 AccountIntervals；其余情况沿用 AccountIntervals。
func supplierAccountHealthGuardResolveInterval(config SupplierAccountHealthGuardConfig, target supplierAccountHealthGuardTarget) int {
	if config.PlatformMultiplierIntervalsEnabled && !target.account.Schedulable {
		platform := strings.ToLower(strings.TrimSpace(target.platform))
		// 落桶用供应商来源倍率；没有任何来源时按 1.0 兜底（供应商倍率的库内默认值）。
		multiplier := 1.0
		if m := supplierAccountHealthGuardSourceMultiplier(target.sources); m != nil {
			multiplier = *m
		}
		for _, rule := range config.PlatformMultiplierIntervals[platform] {
			if multiplier < rule.MinMultiplier {
				continue
			}
			if rule.MaxMultiplier > 0 && multiplier >= rule.MaxMultiplier {
				continue
			}
			return rule.IntervalSeconds
		}
		return 0
	}
	return config.AccountIntervals[target.account.ID]
}

func supplierAccountHealthGuardFilterNotDue(targets []supplierAccountHealthGuardTarget, config SupplierAccountHealthGuardConfig, now time.Time) ([]supplierAccountHealthGuardTarget, []SupplierAccountHealthGuardRunItem) {
	out := make([]supplierAccountHealthGuardTarget, 0, len(targets))
	notDue := make([]SupplierAccountHealthGuardRunItem, 0)
	for _, target := range targets {
		interval := supplierAccountHealthGuardResolveInterval(config, target)
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
				IntervalSeconds:   &interval,
				NextCheckAt:       supplierAccountHealthGuardNextCheckAt(lastCheckedAt, interval),

				BillingRateMultiplier: supplierAccountHealthGuardSourceMultiplier(target.sources),
				StartedAt:             now,
				FinishedAt:            now,
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

// supplierAccountHealthGuardNextCheckAt 由「上次检查时间 + 间隔」推出下次检查时间。
// 每轮都测（间隔 <= 0）没有「下次」的概念，从未检查过的账号也推不出时间点，两种情况都返回 nil。
// 与 supplierAccountHealthGuardFilterNotDue 的跳过判据共用同一个算式：那两处一旦不一致，
// 就会出现「明细说还要等 3 分钟，却根本没被跳过」这种自相矛盾的结果。
func supplierAccountHealthGuardNextCheckAt(lastCheckedAt time.Time, intervalSeconds int) *time.Time {
	if intervalSeconds <= 0 || lastCheckedAt.IsZero() {
		return nil
	}
	next := lastCheckedAt.Add(time.Duration(intervalSeconds) * time.Second)
	return &next
}

// supplierAccountHealthGuardSourceMultiplier 取供应商侧账号的计费倍率，供倍率区间落桶与明细排序共用。
// 一个本地账号可能对应多个供应商来源，取第一个来源的倍率，与平台 / 模型的取值口径一致（都取首个匹配来源）。
// nil 与 0 必须分开：0 是合法的「计费为 0」，升序排序时要排在最前面；
// 没有任何来源时返回 nil，表示这个账号已经查不到、没有供应商倍率可言，排序时应排在最后。
func supplierAccountHealthGuardSourceMultiplier(sources []SupplierAccountHealthGuardSource) *float64 {
	if len(sources) == 0 {
		return nil
	}
	multiplier := sources[0].RateMultiplier
	return &multiplier
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
	config.PlatformMultiplierIntervals = normalizeSupplierAccountHealthGuardPlatformMultiplierIntervals(config.PlatformMultiplierIntervals)
	config.AccountFailureThresholds = normalizeSupplierAccountHealthGuardAccountThresholds(config.AccountFailureThresholds)
	config.AccountSlowThresholds = normalizeSupplierAccountHealthGuardAccountThresholds(config.AccountSlowThresholds)
	config.AccountRecoveryThresholds = normalizeSupplierAccountHealthGuardAccountThresholds(config.AccountRecoveryThresholds)
	if config.AccountSchedulingChange == nil {
		config.AccountSchedulingChange = map[int64]bool{}
	}
	return config
}

func normalizeSupplierAccountHealthGuardAccountThresholds(values map[int64]int) map[int64]int {
	out := make(map[int64]int)
	for accountID, threshold := range values {
		if accountID > 0 && threshold > 0 {
			out[accountID] = threshold
		}
	}
	return out
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

// normalizeSupplierAccountHealthGuardPlatformMultiplierIntervals 清洗每个平台的倍率区间规则：
// 平台 key 转小写、丢弃间隔不足或非法区间（min<0、max>0 且 min>=max）的规则，并按下界升序排。
func normalizeSupplierAccountHealthGuardPlatformMultiplierIntervals(values map[string][]SupplierAccountHealthGuardMultiplierInterval) map[string][]SupplierAccountHealthGuardMultiplierInterval {
	out := make(map[string][]SupplierAccountHealthGuardMultiplierInterval)
	for platform, rules := range values {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if platform == "" {
			continue
		}
		cleaned := make([]SupplierAccountHealthGuardMultiplierInterval, 0, len(rules))
		for _, rule := range rules {
			if rule.IntervalSeconds < MinSupplierAccountHealthGuardAccountIntervalSeconds {
				continue
			}
			if rule.MinMultiplier < 0 {
				continue
			}
			if rule.MaxMultiplier > 0 && rule.MinMultiplier >= rule.MaxMultiplier {
				continue
			}
			cleaned = append(cleaned, rule)
		}
		if len(cleaned) == 0 {
			continue
		}
		sort.Slice(cleaned, func(i, j int) bool { return cleaned[i].MinMultiplier < cleaned[j].MinMultiplier })
		out[platform] = cleaned
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
	recovery, slow, failure := supplierAccountHealthGuardThresholdsForAccount(config, item.LocalAccountID)
	switch item.Status {
	case SupplierAccountHealthGuardStatusHealthy:
		if !item.SchedulableBefore && item.ConsecutiveHealthy >= recovery {
			return true, SupplierAccountHealthGuardActionRecovered, fmt.Sprintf("连续健康 %d 次", item.ConsecutiveHealthy)
		}
	case SupplierAccountHealthGuardStatusSlow:
		if item.SchedulableBefore && item.ConsecutiveSlow >= slow {
			return false, SupplierAccountHealthGuardActionDisabled, fmt.Sprintf("连续慢响应 %d 次", item.ConsecutiveSlow)
		}
	case SupplierAccountHealthGuardStatusFailed:
		if item.SchedulableBefore && item.ConsecutiveFailed >= failure {
			return false, SupplierAccountHealthGuardActionDisabled, fmt.Sprintf("连续失败 %d 次", item.ConsecutiveFailed)
		}
	}
	return item.SchedulableBefore, SupplierAccountHealthGuardActionNone, item.Reason
}

// 账号级阈值优先，未单独配置的账号沿用全局阈值；全局阈值已被归一化为正数，因此这里的兜底是防御性的。
func supplierAccountHealthGuardThresholdsForAccount(config SupplierAccountHealthGuardConfig, accountID int64) (recovery, slow, failure int) {
	recovery = config.RecoveryThreshold
	if recovery <= 0 {
		recovery = DefaultSupplierAccountHealthGuardRecoveryThreshold
	}
	slow = config.SlowThreshold
	if slow <= 0 {
		slow = DefaultSupplierAccountHealthGuardSlowThreshold
	}
	failure = config.FailureThreshold
	if failure <= 0 {
		failure = DefaultSupplierAccountHealthGuardFailureThreshold
	}
	if threshold := config.AccountRecoveryThresholds[accountID]; threshold > 0 {
		recovery = threshold
	}
	if threshold := config.AccountSlowThresholds[accountID]; threshold > 0 {
		slow = threshold
	}
	if threshold := config.AccountFailureThresholds[accountID]; threshold > 0 {
		failure = threshold
	}
	return recovery, slow, failure
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
