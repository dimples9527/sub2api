package service

import (
	"context"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// BatchBindAccountGroupsMaxAccounts 限制单次批量绑定的账号数。
//
// 每个账号都要独立走一遍「校验分组 → simple 模式可绑定性 → 混合渠道检查 → 写绑定 → 投递调度事件」，
// 一次请求塞进几百个账号会把 HTTP 请求拖成分钟级操作，也会让部分失败的提示变得难以阅读。
const BatchBindAccountGroupsMaxAccounts = 200

// BatchBindAccountGroupsInput 是「把若干账号追加绑定到若干分组」的入参。
type BatchBindAccountGroupsInput struct {
	AccountIDs []int64
	GroupIDs   []int64
	// SkipMixedChannelCheck 跳过混合渠道检查，只有调用方已确认风险时才允许置位。
	SkipMixedChannelCheck bool
}

// BatchBindAccountGroupResult 描述单个账号的处理结果。
// Status 取值：bound（已追加绑定）/ unchanged（本来就在这些分组里）/ failed。
type BatchBindAccountGroupResult struct {
	AccountID int64   `json:"account_id"`
	Status    string  `json:"status"`
	GroupIDs  []int64 `json:"group_ids"`
	Added     []int64 `json:"added"`
	Error     string  `json:"error,omitempty"`
}

// BatchBindAccountGroupsResult 是批量绑定分组的汇总结果。
type BatchBindAccountGroupsResult struct {
	Bound     int                          `json:"bound"`
	Unchanged int                          `json:"unchanged"`
	Failed    int                          `json:"failed"`
	Results   []BatchBindAccountGroupResult `json:"results"`
}

// BatchBindAccountGroups 把 groupIDs 追加绑定到 accountIDs 指向的账号上，并返回逐账号结果。
//
// 语义与 UpdateAccount / BulkUpdateAccounts 的「整体替换」刻意不同：这里保留账号原有分组，
// 只补齐缺失的那些（并集）。批量入口的用户意图是「让这批账号也加入这个分组」，
// 用替换语义会把它们从其它分组里悄悄摘掉——不报错，但调度范围已经变了。
//
// 校验与副作用全部复用 BulkUpdateAccounts：分组存在性、simple 模式可绑定性、
// 混合渠道风险，以及 BindGroups 内部投递的调度事件，避免在这里另写一份规则而慢慢跑偏。
func (s *adminServiceImpl) BatchBindAccountGroups(ctx context.Context, input *BatchBindAccountGroupsInput) (*BatchBindAccountGroupsResult, error) {
	if s == nil || s.accountRepo == nil {
		return nil, fmt.Errorf("account repository not configured")
	}
	if input == nil {
		return nil, infraerrors.BadRequest("BATCH_BIND_INVALID_INPUT", "批量绑定参数不能为空")
	}

	accountIDs := uniquePositiveInt64s(input.AccountIDs)
	groupIDs := uniquePositiveInt64s(input.GroupIDs)
	if len(accountIDs) == 0 {
		return nil, infraerrors.BadRequest("BATCH_BIND_NO_ACCOUNTS", "至少需要选择一个账号")
	}
	if len(groupIDs) == 0 {
		return nil, infraerrors.BadRequest("BATCH_BIND_NO_GROUPS", "至少需要选择一个分组")
	}
	if len(accountIDs) > BatchBindAccountGroupsMaxAccounts {
		return nil, infraerrors.BadRequest("BATCH_BIND_TOO_MANY_ACCOUNTS", fmt.Sprintf(
			"一次最多绑定 %d 个账号，当前选择了 %d 个",
			BatchBindAccountGroupsMaxAccounts, len(accountIDs),
		))
	}

	accounts, err := s.accountRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			byID[account.ID] = account
		}
	}

	result := &BatchBindAccountGroupsResult{Results: make([]BatchBindAccountGroupResult, 0, len(accountIDs))}
	for _, accountID := range accountIDs {
		account, ok := byID[accountID]
		if !ok {
			result.Failed++
			result.Results = append(result.Results, BatchBindAccountGroupResult{
				AccountID: accountID,
				Status:    "failed",
				GroupIDs:  []int64{},
				Added:     []int64{},
				Error:     "账号不存在",
			})
			continue
		}

		merged := mergeAccountGroupIDs(account.GroupIDs, groupIDs)
		added := missingAccountGroupIDs(account.GroupIDs, groupIDs)
		if len(added) == 0 {
			result.Unchanged++
			result.Results = append(result.Results, BatchBindAccountGroupResult{
				AccountID: accountID,
				Status:    "unchanged",
				GroupIDs:  merged,
				Added:     []int64{},
			})
			continue
		}

		// 逐个账号调用：分组校验与混合渠道检查都按账号平台判定，一个账号被拦下
		// 不应该连带整批失败，逐条返回原因才能让操作者知道该去掉哪一个。
		writeResult, writeErr := s.BulkUpdateAccounts(ctx, &BulkUpdateAccountsInput{
			AccountIDs:            []int64{accountID},
			GroupIDs:              &merged,
			SkipMixedChannelCheck: input.SkipMixedChannelCheck,
		})
		if writeErr != nil {
			result.Failed++
			result.Results = append(result.Results, BatchBindAccountGroupResult{
				AccountID: accountID,
				Status:    "failed",
				GroupIDs:  merged,
				Added:     added,
				Error:     writeErr.Error(),
			})
			continue
		}
		if writeResult != nil && writeResult.Failed > 0 {
			result.Failed++
			result.Results = append(result.Results, BatchBindAccountGroupResult{
				AccountID: accountID,
				Status:    "failed",
				GroupIDs:  merged,
				Added:     added,
				Error:     batchBindFailureMessage(writeResult, accountID),
			})
			continue
		}

		result.Bound++
		result.Results = append(result.Results, BatchBindAccountGroupResult{
			AccountID: accountID,
			Status:    "bound",
			GroupIDs:  merged,
			Added:     added,
		})
	}

	return result, nil
}

// batchBindFailureMessage 从批量更新结果里取出该账号的失败原因。
func batchBindFailureMessage(result *BulkUpdateAccountsResult, accountID int64) string {
	for _, entry := range result.Results {
		if entry.AccountID == accountID && entry.Error != "" {
			return entry.Error
		}
	}
	return "绑定分组失败"
}

// mergeAccountGroupIDs 返回 a 与 b 的并集：先保留 a 的顺序，再按 b 的顺序补齐缺的。
func mergeAccountGroupIDs(a, b []int64) []int64 {
	merged := uniquePositiveInt64s(a)
	seen := make(map[int64]struct{}, len(merged))
	for _, id := range merged {
		seen[id] = struct{}{}
	}
	for _, id := range uniquePositiveInt64s(b) {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}
	return merged
}

// missingAccountGroupIDs 返回 b 中不属于 a 的分组，保持 b 的顺序；为空表示无需写入。
func missingAccountGroupIDs(a, b []int64) []int64 {
	existing := make(map[int64]struct{}, len(a))
	for _, id := range a {
		existing[id] = struct{}{}
	}
	missing := make([]int64, 0, len(b))
	for _, id := range uniquePositiveInt64s(b) {
		if _, ok := existing[id]; ok {
			continue
		}
		missing = append(missing, id)
	}
	return missing
}
