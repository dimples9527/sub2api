package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 本文件按「供应商管理模块」业务代码管理（见 AGENTS.md）：批量同步的入口在供应商账号页，
// handler / 路由 / 前端都在供应商模块内，这里的编排只是复用框架既有的单账号同步能力。
// 方法挂在 AccountTestService 上，是因为它要用该服务已注入的账号仓储与单账号同步实现；
// 本机无法离线重跑 Wire，不宜为它新增独立 service（新增 provider 需要改依赖图）。
//
// 批量同步上游模型的能力来自单账号的 SyncUpstreamModelCatalog：它按「平台 + 账号类型」
// 构造探测请求，这里只负责把它并发地跑在一批账号上，并按调用方选定的口径写回白名单。
//
// 白名单与模型映射在存储层是同一个 credentials.model_mapping（白名单 = 自映射 model→model，
// 映射 = model→别名）。**本功能只处理白名单那部分条目**：手写的别名映射与通配符规则
// 在两种模式下都原样保留，见 planUpstreamModelSync。

const (
	UpstreamModelBatchSyncModeMerge   = "merge"
	UpstreamModelBatchSyncModeReplace = "replace"

	upstreamModelBatchSyncMaxAccounts        = 200
	upstreamModelBatchSyncDefaultConcurrency = 3
	upstreamModelBatchSyncMaxConcurrency     = 8
	upstreamModelBatchSyncDefaultPerTimeout  = 30 * time.Second
	upstreamModelBatchSyncMaxPerTimeout      = 90 * time.Second
	upstreamModelBatchSyncDefaultTimeout     = 5 * time.Minute
	upstreamModelBatchSyncMaxTimeout         = 10 * time.Minute

	// job 模式（弹窗进度条）单独给整批预算：每账号除了探测上游还要写库，
	// 用户又会盯着进度条等，所以比同步接口的默认值放宽。
	upstreamModelSyncJobDefaultTimeout = 10 * time.Minute
	upstreamModelSyncJobMaxTimeout     = 30 * time.Minute
	upstreamModelSyncJobRetention      = 30 * time.Minute
	maxUpstreamModelSyncJobs           = 100
)

// 状态值与批量账号测试（BatchAccountTestItem）保持一致，前端可以按同一套语义渲染。
// running 只出现在 job 模式的中间态：该项已被 worker 领走但还没出终态。
const (
	UpstreamModelBatchSyncStatusRunning     = "running"
	UpstreamModelBatchSyncStatusSuccess     = "success"
	UpstreamModelBatchSyncStatusFailed      = "failed"
	UpstreamModelBatchSyncStatusNotFound    = "not_found"
	UpstreamModelBatchSyncStatusInterrupted = "interrupted"
)

// 阶段驱动弹窗里的逐账号进度条：单账号同步依次经过
// fetching（拉上游模型列表）→ enriching（补能力元数据）→ applying（写白名单），
// 每段开始时上报一次；done 表示该项已有终态。
const (
	UpstreamModelBatchSyncPhaseQueued    = "queued"
	UpstreamModelBatchSyncPhaseFetching  = "fetching"
	UpstreamModelBatchSyncPhaseEnriching = "enriching"
	UpstreamModelBatchSyncPhaseApplying  = "applying"
	UpstreamModelBatchSyncPhaseDone      = "done"
)

// job 级状态；cancelling 与批量测试同义 —— 已请求取消，等后台 goroutine 收尾。
const (
	UpstreamModelSyncJobStatusQueued     = "queued"
	UpstreamModelSyncJobStatusRunning    = "running"
	UpstreamModelSyncJobStatusCancelling = "cancelling"
	UpstreamModelSyncJobStatusCompleted  = "completed"
	UpstreamModelSyncJobStatusCancelled  = "cancelled"
	UpstreamModelSyncJobStatusFailed     = "failed"
)

type UpstreamModelBatchSyncInput struct {
	AccountIDs        []int64
	Mode              string
	Apply             bool
	Concurrency       int
	TimeoutPerAccount time.Duration
	Timeout           time.Duration
}

type UpstreamModelBatchSyncItem struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Status      string `json:"status"`
	// Phase / Progress 是给弹窗进度条用的中间态，取值见 UpstreamModelBatchSyncPhase*。
	// 终态项一律 Phase=done、Progress=100；同步接口不消费这两个字段，行为不受影响。
	Phase    string `json:"phase,omitempty"`
	Progress int    `json:"progress"`
	// ErrorMessage 只回传 SafeMessage：上游 401/403 等响应体不得外泄（同单账号接口）。
	ErrorMessage string `json:"error_message,omitempty"`
	// UpstreamTotal 是上游返回的模型数；Added/Removed/FinalCount 一律按「应用后」的口径算，
	// 预览（Apply=false）也是这个口径，前端拿到的两个模式可比对的数字是同一套算法。
	UpstreamTotal int `json:"upstream_total"`
	Added         int `json:"added"`
	Removed       int `json:"removed"`
	FinalCount    int `json:"final_count"`
	CurrentCount  int `json:"current_count"`
	// CustomMappingKept 是同步后原样保留的手写别名映射条数：两种模式都不动它们，
	// 报出来是为了让管理员确认「我手写的映射还在」。
	CustomMappingKept int                        `json:"custom_mapping_kept"`
	Warnings          []UpstreamModelSyncWarning `json:"warnings,omitempty"`
}

type UpstreamModelBatchSyncResult struct {
	Total   int                          `json:"total"`
	Success int                          `json:"success"`
	Failed  int                          `json:"failed"`
	Mode    string                       `json:"mode"`
	Applied bool                         `json:"applied"`
	Results []UpstreamModelBatchSyncItem `json:"results"`
}

// UpstreamModelBatchSyncJob 是 job 模式下的任务快照。
//
// 与 BatchAccountTestJob 同构（同页同 group 的两个批量入口，前端用同一套轮询写法），
// 差别只在多带 Mode/Applied，以及每项多带 Phase/Progress 供进度条渲染。
type UpstreamModelBatchSyncJob struct {
	JobID        string                       `json:"job_id"`
	Status       string                       `json:"status"`
	Total        int                          `json:"total"`
	Completed    int                          `json:"completed"`
	Success      int                          `json:"success"`
	Failed       int                          `json:"failed"`
	Mode         string                       `json:"mode"`
	Applied      bool                         `json:"applied"`
	Results      []UpstreamModelBatchSyncItem `json:"results"`
	ErrorMessage string                       `json:"error_message,omitempty"`
	CreatedAt    string                       `json:"created_at,omitempty"`
	StartedAt    string                       `json:"started_at,omitempty"`
	FinishedAt   string                       `json:"finished_at,omitempty"`

	cancel         context.CancelFunc
	resultSlots    []UpstreamModelBatchSyncItem
	createdAtTime  time.Time
	finishedAtTime time.Time
}

type normalizedUpstreamModelBatchSyncInput struct {
	AccountIDs        []int64
	Mode              string
	Apply             bool
	Concurrency       int
	TimeoutPerAccount time.Duration
	Timeout           time.Duration
}

// SyncUpstreamModelCatalogBatch 并发同步一批账号的上游模型。
//
// Apply=false 时只返回「应用后会长什么样」，不写白名单（能力快照仍会刷新，这与单账号
// 同步按钮一致）。Apply=true 时按 Mode 写回 credentials.model_mapping 的白名单条目，
// 手写映射不受影响。
func (s *AccountTestService) SyncUpstreamModelCatalogBatch(
	ctx context.Context,
	input UpstreamModelBatchSyncInput,
) (*UpstreamModelBatchSyncResult, error) {
	normalized, err := normalizeUpstreamModelBatchSyncInput(input, s)
	if err != nil {
		return nil, err
	}
	if len(normalized.AccountIDs) == 0 {
		return &UpstreamModelBatchSyncResult{
			Mode:    normalized.Mode,
			Applied: normalized.Apply,
			Results: []UpstreamModelBatchSyncItem{},
		}, nil
	}

	runCtx, cancel := context.WithTimeout(ctx, normalized.Timeout)
	defer cancel()

	return s.runUpstreamModelBatchSync(runCtx, normalized, nil)
}

// runUpstreamModelBatchSync 是同步接口与 job 模式共用的编排。
//
// onItem 不为 nil 时，每项「被 worker 领走」「进入新阶段」「出终态」各回调一次，
// 供 job 上报进度条；同步接口传 nil，拿到的结果与改造前完全一致。
// onItem 会被多个 worker 并发调用，实现必须自带同步（job 侧用的是带锁的更新函数）。
func (s *AccountTestService) runUpstreamModelBatchSync(
	ctx context.Context,
	input normalizedUpstreamModelBatchSyncInput,
	onItem func(index int, item UpstreamModelBatchSyncItem),
) (*UpstreamModelBatchSyncResult, error) {
	ids := input.AccountIDs
	if len(ids) == 0 {
		return &UpstreamModelBatchSyncResult{
			Mode:    input.Mode,
			Applied: input.Apply,
			Results: []UpstreamModelBatchSyncItem{},
		}, nil
	}

	accounts, err := s.accountRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load accounts: %w", err)
	}
	accountByID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			accountByID[account.ID] = account
		}
	}

	results := make([]UpstreamModelBatchSyncItem, len(ids))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for worker := 0; worker < input.Concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				account := accountByID[ids[index]]
				if onItem != nil {
					onItem(index, upstreamModelBatchSyncPendingItem(ids[index], account, UpstreamModelBatchSyncPhaseQueued))
				}
				item := s.runUpstreamModelBatchSyncItem(ctx, account, input, func(phase string) {
					if onItem == nil {
						return
					}
					onItem(index, upstreamModelBatchSyncPendingItem(ids[index], account, phase))
				})
				// account 为 nil 时同步流程拿不到 ID，这里补齐，
				// 否则前端没法按 account_id 把进度挂到对应那一行。
				if item.AccountID == 0 {
					item.AccountID = ids[index]
				}
				mu.Lock()
				results[index] = item
				mu.Unlock()
				if onItem != nil {
					onItem(index, item)
				}
			}
		}()
	}
enqueue:
	for index := range ids {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()

	result := &UpstreamModelBatchSyncResult{
		Total:   len(ids),
		Mode:    input.Mode,
		Applied: input.Apply,
		Results: results,
	}
	for i := range results {
		if results[i].Status == "" {
			// 整批超时/取消时没跑到的一项：不能留空状态，否则前端算不出成功数。
			results[i] = UpstreamModelBatchSyncItem{
				AccountID:    ids[i],
				Status:       UpstreamModelBatchSyncStatusInterrupted,
				Phase:        UpstreamModelBatchSyncPhaseDone,
				Progress:     upstreamModelBatchSyncPhaseProgress(UpstreamModelBatchSyncPhaseDone),
				ErrorMessage: "batch sync was interrupted",
			}
			if account := accountByID[ids[i]]; account != nil {
				results[i].AccountName = account.Name
				results[i].Platform = account.Platform
			}
			if onItem != nil {
				onItem(i, results[i])
			}
		}
		if results[i].Status == UpstreamModelBatchSyncStatusSuccess {
			result.Success++
		} else {
			result.Failed++
		}
	}
	return result, nil
}

func (s *AccountTestService) runUpstreamModelBatchSyncItem(
	ctx context.Context,
	account *Account,
	input normalizedUpstreamModelBatchSyncInput,
	onPhase func(phase string),
) UpstreamModelBatchSyncItem {
	if account == nil {
		return upstreamModelBatchSyncFinishedItem(UpstreamModelBatchSyncItem{
			Status:       UpstreamModelBatchSyncStatusNotFound,
			ErrorMessage: "account not found",
		})
	}

	item := UpstreamModelBatchSyncItem{
		AccountID:   account.ID,
		AccountName: account.Name,
		Platform:    account.Platform,
	}

	itemCtx, cancel := context.WithTimeout(ctx, input.TimeoutPerAccount)
	defer cancel()

	// SyncUpstreamModelCatalogWithProgress 会写回传入账号的能力快照缓存，多 worker 各跑各的
	// 账号，这里仍给每个账号一份副本，避免与账号列表里还在被别处读取的对象共享可变状态。
	working := accountShallowCopyForUpstreamSync(account)
	catalog, err := s.SyncUpstreamModelCatalogWithProgress(itemCtx, working, func(phase UpstreamModelSyncPhase) {
		if onPhase != nil {
			onPhase(string(phase))
		}
	})
	if err != nil {
		item.Status = UpstreamModelBatchSyncStatusFailed
		item.ErrorMessage = upstreamModelBatchSyncSafeMessage(err)
		return upstreamModelBatchSyncFinishedItem(item)
	}

	models := dedupeAndSortModelIDs(catalog.Models)
	// 预览与落库共用同一个 plan：预览展示的 Added/Removed/FinalCount 就是落库会得到的
	// 结果，避免「预览说删 3 条、实际删 5 条」。
	plan := planUpstreamModelSync(working.Credentials, models, input.Mode)
	item.UpstreamTotal = len(models)
	item.CurrentCount = plan.CurrentCount
	item.Added = plan.Added
	item.Removed = plan.Removed
	item.FinalCount = len(plan.Next)
	item.CustomMappingKept = plan.CustomKept
	item.Warnings = catalog.Warnings

	if input.Apply {
		if onPhase != nil {
			onPhase(UpstreamModelBatchSyncPhaseApplying)
		}
		if err := applyUpstreamModelSyncToAccount(ctx, s.accountRepo, working, plan); err != nil {
			item.Status = UpstreamModelBatchSyncStatusFailed
			item.ErrorMessage = "failed to save upstream models"
			return upstreamModelBatchSyncFinishedItem(item)
		}
	}
	item.Status = UpstreamModelBatchSyncStatusSuccess
	return upstreamModelBatchSyncFinishedItem(item)
}

// upstreamModelBatchSyncPendingItem 构造「尚未出终态」的进度项：Status 固定为 running，
// 具体位置由 Phase/Progress 表达。
func upstreamModelBatchSyncPendingItem(accountID int64, account *Account, phase string) UpstreamModelBatchSyncItem {
	item := UpstreamModelBatchSyncItem{
		AccountID: accountID,
		Status:    UpstreamModelBatchSyncStatusRunning,
		Phase:     phase,
		Progress:  upstreamModelBatchSyncPhaseProgress(phase),
	}
	if account != nil {
		item.AccountName = account.Name
		item.Platform = account.Platform
	}
	return item
}

// upstreamModelBatchSyncFinishedItem 给终态项补上收尾的 Phase/Progress。
// 进度条走到头必须显式落到 100，否则「已完成的账号」会停在中途的百分比上。
func upstreamModelBatchSyncFinishedItem(item UpstreamModelBatchSyncItem) UpstreamModelBatchSyncItem {
	item.Phase = UpstreamModelBatchSyncPhaseDone
	item.Progress = upstreamModelBatchSyncPhaseProgress(UpstreamModelBatchSyncPhaseDone)
	return item
}

// upstreamModelBatchSyncPhaseProgress 把阶段映射成进度条百分比。
// 数字只服务进度条，不参与任何业务判断：三段外部/写库动作各占一段，done 必为 100。
func upstreamModelBatchSyncPhaseProgress(phase string) int {
	switch phase {
	case UpstreamModelBatchSyncPhaseQueued:
		return 0
	case UpstreamModelBatchSyncPhaseFetching:
		return 25
	case UpstreamModelBatchSyncPhaseEnriching:
		return 60
	case UpstreamModelBatchSyncPhaseApplying:
		return 85
	default:
		return 100
	}
}

func normalizeUpstreamModelBatchSyncInput(
	input UpstreamModelBatchSyncInput,
	s *AccountTestService,
) (normalizedUpstreamModelBatchSyncInput, error) {
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = UpstreamModelBatchSyncModeMerge
	}
	if mode != UpstreamModelBatchSyncModeMerge && mode != UpstreamModelBatchSyncModeReplace {
		return normalizedUpstreamModelBatchSyncInput{}, infraerrors.BadRequest(
			"UPSTREAM_MODEL_BATCH_SYNC_INVALID_MODE", "mode must be merge or replace")
	}

	ids := make([]int64, 0, len(input.AccountIDs))
	seen := make(map[int64]struct{}, len(input.AccountIDs))
	for _, id := range input.AccountIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return normalizedUpstreamModelBatchSyncInput{Mode: mode, Apply: input.Apply}, nil
	}
	if len(ids) > upstreamModelBatchSyncMaxAccounts {
		return normalizedUpstreamModelBatchSyncInput{}, infraerrors.BadRequest(
			"UPSTREAM_MODEL_BATCH_SYNC_TOO_MANY_ACCOUNTS",
			fmt.Sprintf("too many accounts: max %d", upstreamModelBatchSyncMaxAccounts))
	}
	if s == nil || s.accountRepo == nil {
		return normalizedUpstreamModelBatchSyncInput{}, fmt.Errorf("account repository is not configured")
	}

	concurrency := input.Concurrency
	if concurrency <= 0 {
		concurrency = upstreamModelBatchSyncDefaultConcurrency
	}
	if concurrency > upstreamModelBatchSyncMaxConcurrency {
		concurrency = upstreamModelBatchSyncMaxConcurrency
	}
	if concurrency > len(ids) {
		concurrency = len(ids)
	}

	perTimeout := input.TimeoutPerAccount
	if perTimeout <= 0 {
		perTimeout = upstreamModelBatchSyncDefaultPerTimeout
	}
	if perTimeout > upstreamModelBatchSyncMaxPerTimeout {
		perTimeout = upstreamModelBatchSyncMaxPerTimeout
	}

	totalTimeout := input.Timeout
	if totalTimeout <= 0 {
		totalTimeout = upstreamModelBatchSyncDefaultTimeout
	}
	if totalTimeout > upstreamModelBatchSyncMaxTimeout {
		totalTimeout = upstreamModelBatchSyncMaxTimeout
	}

	return normalizedUpstreamModelBatchSyncInput{
		AccountIDs:        ids,
		Mode:              mode,
		Apply:             input.Apply,
		Concurrency:       concurrency,
		TimeoutPerAccount: perTimeout,
		Timeout:           totalTimeout,
	}, nil
}

func accountShallowCopyForUpstreamSync(account *Account) *Account {
	if account == nil {
		return nil
	}
	copied := *account
	copied.Credentials = shallowCopyMap(account.Credentials)
	copied.Extra = shallowCopyMap(account.Extra)
	// 热路径缓存是惰性计算的，副本重新走一次计算，避免与源对象共享可变 map。
	copied.modelMappingCache = nil
	copied.modelMappingCacheReady = false
	copied.headerOverrideCache = nil
	copied.headerOverrideCacheReady = false
	return &copied
}

// upstreamModelMappingEntryIsWhitelist 判定一条 model_mapping 条目是不是「白名单条目」。
// 白名单在存储层就是自映射（model→model，且不含通配符）；其余一律视为管理员手写的
// 别名映射或通配符规则 —— 那些在同步里永远不动。
func upstreamModelMappingEntryIsWhitelist(key string, value any) bool {
	target, ok := value.(string)
	if !ok {
		return false
	}
	if strings.Contains(key, "*") {
		return false
	}
	return strings.TrimSpace(target) == key
}

type upstreamModelSyncPlan struct {
	Next         map[string]any
	Added        int
	Removed      int
	CustomKept   int
	CurrentCount int
}

// planUpstreamModelSync 是预览与落库共用的唯一算法：预览拿它算「应用后长什么样」，
// 落库直接写它算出的 Next，两者不可能不一致。
//
// 口径（用户 2026-09-29 明确要求：只处理模型白名单，模型映射不动）：
//   - 手写别名映射（from≠to 或含通配符）**原样保留**，merge 与 replace 都不动；
//   - 白名单条目（自映射）：merge 保留现有并追加缺失的上游模型；replace 换成上游列表，
//     本地独有（上游已下架）的白名单条目会被移除；
//   - 上游模型与已保留条目同名时不覆盖 —— 手写映射优先，否则就等于把手写映射改掉了。
func planUpstreamModelSync(credentials map[string]any, models []string, mode string) upstreamModelSyncPlan {
	next := map[string]any{}
	whitelistKeys := map[string]struct{}{}
	plan := upstreamModelSyncPlan{Next: next}

	raw, _ := credentials["model_mapping"].(map[string]any)
	for key, value := range raw {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		plan.CurrentCount++
		if upstreamModelMappingEntryIsWhitelist(trimmed, value) {
			whitelistKeys[trimmed] = struct{}{}
			if mode == UpstreamModelBatchSyncModeMerge {
				next[trimmed] = trimmed
			}
			continue
		}
		next[trimmed] = value
		plan.CustomKept++
	}

	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, exists := next[model]; exists {
			continue
		}
		next[model] = model
		// Added 只统计「白名单里新出现的模型」：原本就在白名单里的（replace 时会被重写一遍）
		// 不算新增，否则覆盖模式下的数字会虚高。
		if _, wasWhitelist := whitelistKeys[model]; !wasWhitelist {
			plan.Added++
		}
	}

	if mode == UpstreamModelBatchSyncModeReplace {
		for key := range whitelistKeys {
			if _, kept := next[key]; !kept {
				plan.Removed++
			}
		}
	}
	return plan
}

func applyUpstreamModelSyncToAccount(
	ctx context.Context,
	repo AccountRepository,
	account *Account,
	plan upstreamModelSyncPlan,
) error {
	if account == nil || account.ID <= 0 {
		return fmt.Errorf("account is required")
	}
	updater, ok := any(repo).(accountCredentialsUpdater)
	if !ok {
		return fmt.Errorf("account repository does not support credentials update")
	}
	if len(plan.Next) == 0 {
		// 兜底：上游层对空列表已经会报错，正常走不到这里。但白名单/映射是账号的转发口径，
		// 写空等于让账号什么都不认，宁可失败也不落一个空对象。
		return fmt.Errorf("refusing to write an empty model mapping")
	}

	credentials := shallowCopyMap(account.Credentials)
	if credentials == nil {
		credentials = map[string]any{}
	}
	credentials["model_mapping"] = plan.Next
	account.Credentials = credentials
	return updater.UpdateCredentials(ctx, account.ID, credentials)
}

func upstreamModelBatchSyncSafeMessage(err error) string {
	var syncErr *UpstreamModelSyncError
	if errors.As(err, &syncErr) {
		return syncErr.SafeMessage()
	}
	return "failed to sync upstream models"
}

// StartSyncUpstreamModelCatalogBatchJob 启动一次批量同步任务并立即返回任务快照。
//
// 与 SyncUpstreamModelCatalogBatch 的区别只有一处：不阻塞等结果，改为后台跑 + 轮询。
// 弹窗要逐账号显示进度条，而进度只能在每项「被领走 / 换阶段 / 出终态」时上报，
// 同步接口一次性返回是拿不到的。
func (s *AccountTestService) StartSyncUpstreamModelCatalogBatchJob(
	ctx context.Context,
	input UpstreamModelBatchSyncInput,
) (*UpstreamModelBatchSyncJob, error) {
	normalized, err := normalizeUpstreamModelBatchSyncInput(input, s)
	if err != nil {
		return nil, err
	}

	// 整批预算按原始输入重算：normalize 已把 0 兜成同步接口的 5 分钟，
	// 而 job 模式每账号还要写库、用户又在等进度条，给更宽的默认值。
	jobTimeout := input.Timeout
	if jobTimeout <= 0 {
		jobTimeout = upstreamModelSyncJobDefaultTimeout
	}
	if jobTimeout > upstreamModelSyncJobMaxTimeout {
		jobTimeout = upstreamModelSyncJobMaxTimeout
	}
	normalized.Timeout = jobTimeout

	jobCtx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	now := time.Now().UTC()
	job := &UpstreamModelBatchSyncJob{
		JobID:         uuid.NewString(),
		Status:        UpstreamModelSyncJobStatusQueued,
		Total:         len(normalized.AccountIDs),
		Mode:          normalized.Mode,
		Applied:       normalized.Apply,
		CreatedAt:     now.Format(time.RFC3339Nano),
		cancel:        cancel,
		resultSlots:   make([]UpstreamModelBatchSyncItem, len(normalized.AccountIDs)),
		createdAtTime: now,
	}
	if err := s.storeUpstreamModelSyncJob(job); err != nil {
		cancel()
		return nil, err
	}
	go s.runUpstreamModelSyncJob(jobCtx, job.JobID, normalized)

	return s.GetSyncUpstreamModelCatalogBatchJob(ctx, job.JobID)
}

// GetSyncUpstreamModelCatalogBatchJob 返回任务快照，供弹窗轮询。
func (s *AccountTestService) GetSyncUpstreamModelCatalogBatchJob(
	_ context.Context,
	jobID string,
) (*UpstreamModelBatchSyncJob, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_MODEL_SYNC_JOB_ID_REQUIRED", "job_id is required")
	}
	s.upstreamModelSyncJobsMu.RLock()
	defer s.upstreamModelSyncJobsMu.RUnlock()
	job := s.upstreamModelSyncJobs[jobID]
	if job == nil {
		return nil, infraerrors.NotFound("UPSTREAM_MODEL_SYNC_JOB_NOT_FOUND", "upstream model sync job not found")
	}
	return snapshotUpstreamModelSyncJob(job), nil
}

// CancelSyncUpstreamModelCatalogBatchJob 请求取消任务，语义与批量测试的取消一致：
// 只把状态推成 cancelling 并触发 context 取消，收尾由后台 goroutine 完成。
func (s *AccountTestService) CancelSyncUpstreamModelCatalogBatchJob(
	ctx context.Context,
	jobID string,
) (*UpstreamModelBatchSyncJob, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_MODEL_SYNC_JOB_ID_REQUIRED", "job_id is required")
	}

	s.upstreamModelSyncJobsMu.Lock()
	job := s.upstreamModelSyncJobs[jobID]
	if job == nil {
		s.upstreamModelSyncJobsMu.Unlock()
		return nil, infraerrors.NotFound("UPSTREAM_MODEL_SYNC_JOB_NOT_FOUND", "upstream model sync job not found")
	}
	if job.Status == UpstreamModelSyncJobStatusQueued || job.Status == UpstreamModelSyncJobStatusRunning {
		job.Status = UpstreamModelSyncJobStatusCancelling
		if job.cancel != nil {
			job.cancel()
		}
	}
	s.upstreamModelSyncJobsMu.Unlock()
	return s.GetSyncUpstreamModelCatalogBatchJob(ctx, jobID)
}

func (s *AccountTestService) runUpstreamModelSyncJob(
	ctx context.Context,
	jobID string,
	input normalizedUpstreamModelBatchSyncInput,
) {
	s.markUpstreamModelSyncJobStarted(jobID)
	_, err := s.runUpstreamModelBatchSync(ctx, input, func(index int, item UpstreamModelBatchSyncItem) {
		s.updateUpstreamModelSyncJobItem(jobID, index, item)
	})
	s.finishUpstreamModelSyncJob(jobID, ctx, err)
}

func (s *AccountTestService) markUpstreamModelSyncJobStarted(jobID string) {
	s.upstreamModelSyncJobsMu.Lock()
	defer s.upstreamModelSyncJobsMu.Unlock()
	job := s.upstreamModelSyncJobs[jobID]
	if job == nil {
		return
	}
	if job.Status == UpstreamModelSyncJobStatusQueued {
		job.Status = UpstreamModelSyncJobStatusRunning
	}
	job.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
}

func (s *AccountTestService) updateUpstreamModelSyncJobItem(
	jobID string,
	index int,
	item UpstreamModelBatchSyncItem,
) {
	s.upstreamModelSyncJobsMu.Lock()
	defer s.upstreamModelSyncJobsMu.Unlock()
	job := s.upstreamModelSyncJobs[jobID]
	if job == nil || index < 0 || index >= len(job.resultSlots) {
		return
	}
	job.resultSlots[index] = item
	recountUpstreamModelSyncJobLocked(job)
}

// recountUpstreamModelSyncJobLocked 每次都按槽位重算计数，而不是增量累加。
//
// 同一项会被上报多次（领走 / 换阶段 / 出终态），增量累加很容易把成功数算重；
// 槽位数上限 200，整批重算的代价可以忽略。
func recountUpstreamModelSyncJobLocked(job *UpstreamModelBatchSyncJob) {
	completed, success, failed := 0, 0, 0
	for index := range job.resultSlots {
		status := job.resultSlots[index].Status
		if !isUpstreamModelBatchSyncTerminalStatus(status) {
			continue
		}
		completed++
		if status == UpstreamModelBatchSyncStatusSuccess {
			success++
		} else {
			failed++
		}
	}
	job.Completed, job.Success, job.Failed = completed, success, failed
}

func (s *AccountTestService) finishUpstreamModelSyncJob(jobID string, ctx context.Context, err error) {
	s.upstreamModelSyncJobsMu.Lock()
	defer s.upstreamModelSyncJobsMu.Unlock()
	job := s.upstreamModelSyncJobs[jobID]
	if job == nil {
		return
	}

	// 超时/取消时仍在「进行中」的槽位收不到终态回调，这里统一补成 interrupted：
	// 否则前端那一行的进度条会永远停在中途，也统计不出最终成功/失败数。
	for index := range job.resultSlots {
		slot := &job.resultSlots[index]
		if isUpstreamModelBatchSyncTerminalStatus(slot.Status) {
			continue
		}
		*slot = UpstreamModelBatchSyncItem{
			AccountID:    slot.AccountID,
			AccountName:  slot.AccountName,
			Platform:     slot.Platform,
			Status:       UpstreamModelBatchSyncStatusInterrupted,
			Phase:        UpstreamModelBatchSyncPhaseDone,
			Progress:     upstreamModelBatchSyncPhaseProgress(UpstreamModelBatchSyncPhaseDone),
			ErrorMessage: "batch sync was interrupted",
		}
	}
	recountUpstreamModelSyncJobLocked(job)

	if err != nil {
		job.Status = UpstreamModelSyncJobStatusFailed
		job.ErrorMessage = err.Error()
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		job.Status = UpstreamModelSyncJobStatusCompleted
		job.ErrorMessage = "batch upstream model sync timed out"
	} else if ctx.Err() != nil {
		job.Status = UpstreamModelSyncJobStatusCancelled
	} else {
		job.Status = UpstreamModelSyncJobStatusCompleted
	}
	now := time.Now().UTC()
	job.FinishedAt = now.Format(time.RFC3339Nano)
	job.finishedAtTime = now
	job.cancel = nil
}

func (s *AccountTestService) storeUpstreamModelSyncJob(job *UpstreamModelBatchSyncJob) error {
	s.upstreamModelSyncJobsMu.Lock()
	defer s.upstreamModelSyncJobsMu.Unlock()
	if s.upstreamModelSyncJobs == nil {
		s.upstreamModelSyncJobs = make(map[string]*UpstreamModelBatchSyncJob)
	}
	s.pruneUpstreamModelSyncJobsLocked(time.Now())
	if len(s.upstreamModelSyncJobs) >= maxUpstreamModelSyncJobs {
		return infraerrors.TooManyRequests(
			"UPSTREAM_MODEL_SYNC_JOBS_FULL",
			"too many upstream model sync jobs are running or retained",
		)
	}
	s.upstreamModelSyncJobs[job.JobID] = job
	return nil
}

func (s *AccountTestService) pruneUpstreamModelSyncJobsLocked(now time.Time) {
	for jobID, job := range s.upstreamModelSyncJobs {
		if !isUpstreamModelSyncJobTerminal(job.Status) || job.finishedAtTime.IsZero() {
			continue
		}
		if now.Sub(job.finishedAtTime) > upstreamModelSyncJobRetention {
			delete(s.upstreamModelSyncJobs, jobID)
		}
	}
}

func isUpstreamModelBatchSyncTerminalStatus(status string) bool {
	switch status {
	case UpstreamModelBatchSyncStatusSuccess,
		UpstreamModelBatchSyncStatusFailed,
		UpstreamModelBatchSyncStatusNotFound,
		UpstreamModelBatchSyncStatusInterrupted:
		return true
	default:
		return false
	}
}

func isUpstreamModelSyncJobTerminal(status string) bool {
	return status == UpstreamModelSyncJobStatusCompleted ||
		status == UpstreamModelSyncJobStatusCancelled ||
		status == UpstreamModelSyncJobStatusFailed
}

func snapshotUpstreamModelSyncJob(job *UpstreamModelBatchSyncJob) *UpstreamModelBatchSyncJob {
	out := &UpstreamModelBatchSyncJob{
		JobID:        job.JobID,
		Status:       job.Status,
		Total:        job.Total,
		Completed:    job.Completed,
		Success:      job.Success,
		Failed:       job.Failed,
		Mode:         job.Mode,
		Applied:      job.Applied,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		StartedAt:    job.StartedAt,
		FinishedAt:   job.FinishedAt,
	}
	out.Results = make([]UpstreamModelBatchSyncItem, 0, len(job.resultSlots))
	for _, item := range job.resultSlots {
		if item.Status != "" {
			out.Results = append(out.Results, item)
		}
	}
	return out
}
