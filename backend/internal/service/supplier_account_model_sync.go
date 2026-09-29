package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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
)

// 状态值与批量账号测试（BatchAccountTestItem）保持一致，前端可以按同一套语义渲染。
const (
	UpstreamModelBatchSyncStatusSuccess     = "success"
	UpstreamModelBatchSyncStatusFailed      = "failed"
	UpstreamModelBatchSyncStatusNotFound    = "not_found"
	UpstreamModelBatchSyncStatusInterrupted = "interrupted"
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

	accounts, err := s.accountRepo.GetByIDs(runCtx, normalized.AccountIDs)
	if err != nil {
		return nil, fmt.Errorf("load accounts: %w", err)
	}
	accountByID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			accountByID[account.ID] = account
		}
	}

	ids := normalized.AccountIDs
	results := make([]UpstreamModelBatchSyncItem, len(ids))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for worker := 0; worker < normalized.Concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				item := s.runUpstreamModelBatchSyncItem(runCtx, accountByID[ids[index]], normalized)
				mu.Lock()
				results[index] = item
				mu.Unlock()
			}
		}()
	}
enqueue:
	for index := range ids {
		select {
		case <-runCtx.Done():
			break enqueue
		case jobs <- index:
		}
	}
	close(jobs)
	wg.Wait()

	result := &UpstreamModelBatchSyncResult{
		Total:   len(ids),
		Mode:    normalized.Mode,
		Applied: normalized.Apply,
		Results: results,
	}
	for i := range results {
		if results[i].Status == "" {
			// 整批超时/取消时没跑到的一项：不能留空状态，否则前端算不出成功数。
			results[i] = UpstreamModelBatchSyncItem{
				AccountID:    ids[i],
				Status:       UpstreamModelBatchSyncStatusInterrupted,
				ErrorMessage: "batch sync was interrupted",
			}
			if account := accountByID[ids[i]]; account != nil {
				results[i].AccountName = account.Name
				results[i].Platform = account.Platform
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
) UpstreamModelBatchSyncItem {
	if account == nil {
		return UpstreamModelBatchSyncItem{Status: UpstreamModelBatchSyncStatusNotFound, ErrorMessage: "account not found"}
	}

	item := UpstreamModelBatchSyncItem{
		AccountID:   account.ID,
		AccountName: account.Name,
		Platform:    account.Platform,
	}

	itemCtx, cancel := context.WithTimeout(ctx, input.TimeoutPerAccount)
	defer cancel()

	// SyncUpstreamModelCatalog 会写回传入账号的能力快照缓存，多 worker 各跑各的账号，
	// 这里仍给每个账号一份副本，避免与账号列表里还在被别处读取的对象共享可变状态。
	working := accountShallowCopyForUpstreamSync(account)
	catalog, err := s.SyncUpstreamModelCatalog(itemCtx, working)
	if err != nil {
		item.Status = UpstreamModelBatchSyncStatusFailed
		item.ErrorMessage = upstreamModelBatchSyncSafeMessage(err)
		return item
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
		if err := applyUpstreamModelSyncToAccount(ctx, s.accountRepo, working, plan); err != nil {
			item.Status = UpstreamModelBatchSyncStatusFailed
			item.ErrorMessage = "failed to save upstream models"
			return item
		}
	}
	item.Status = UpstreamModelBatchSyncStatusSuccess
	return item
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
