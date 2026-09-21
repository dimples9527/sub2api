package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	DefaultSupplierGroupSchedulingElectionTopN = 1
	MaxSupplierGroupSchedulingElectionTopN     = 100

	SupplierGroupSchedulingElectionTestStatusSuccess = "success"
	SupplierGroupSchedulingElectionTestStatusFailed  = "failed"

	SupplierGroupSchedulingElectionActionNone     = "none"
	SupplierGroupSchedulingElectionActionEnabled  = "enabled"
	SupplierGroupSchedulingElectionActionDisabled = "disabled"

	SupplierGroupSchedulingElectionReasonFailed      = "测试失败，关闭调度"
	SupplierGroupSchedulingElectionReasonElected     = "分组内最优，开启调度"
	SupplierGroupSchedulingElectionReasonNotElected  = "非分组最优，关闭调度"
	SupplierGroupSchedulingElectionReasonUntested    = "尚未测试，保持原状"
	SupplierGroupSchedulingElectionReasonWriteFailed = "更新调度状态失败"
)

// SupplierGroupSchedulingElectionConfig 是分组择优调度任务的配置。
type SupplierGroupSchedulingElectionConfig struct {
	// TopN 是每个分组允许保持开启调度的最优账号数量，默认 1（严格单活）。
	TopN int `json:"group_scheduling_election_top_n"`
	// DisabledGroupIDs 存"被关闭择优"的分组 ID，空列表表示全部分组都参与。
	// 存关闭项而不是开启项，是为了让"新增分组默认参与"无需数据迁移。
	DisabledGroupIDs []int64 `json:"group_scheduling_election_disabled_group_ids"`
}

// SupplierGroupSchedulingElectionMember 是仓储层返回的一条"分组×账号"成员行。
type SupplierGroupSchedulingElectionMember struct {
	GroupID        int64
	GroupName      string
	AccountID      int64
	AccountName    string
	Platform       string
	Schedulable    bool
	LastTestStatus string
	HealthyCount   int
	LastTestedAt   time.Time
}

type SupplierGroupSchedulingElectionRepository interface {
	ListGroupSchedulingElectionMembers(ctx context.Context) ([]SupplierGroupSchedulingElectionMember, error)
}

type supplierGroupSchedulingElectionAccountStore interface {
	SetSchedulable(ctx context.Context, id int64, schedulable bool) error
}

type SupplierGroupSchedulingElectionRunner interface {
	Run(ctx context.Context, config SupplierGroupSchedulingElectionConfig, now time.Time) (SupplierGroupSchedulingElectionResult, error)
}

type SupplierGroupSchedulingElectionService struct {
	repository   SupplierGroupSchedulingElectionRepository
	accountStore supplierGroupSchedulingElectionAccountStore
}

func NewSupplierGroupSchedulingElectionService(repository SupplierGroupSchedulingElectionRepository, accountStore supplierGroupSchedulingElectionAccountStore) *SupplierGroupSchedulingElectionService {
	return &SupplierGroupSchedulingElectionService{repository: repository, accountStore: accountStore}
}

// SupplierGroupSchedulingElectionGroupDetail 汇总单个分组的裁决结果。
type SupplierGroupSchedulingElectionGroupDetail struct {
	GroupID       int64   `json:"group_id"`
	GroupName     string  `json:"group_name"`
	MemberCount   int     `json:"member_count"`
	SuccessCount  int     `json:"success_count"`
	FailedCount   int     `json:"failed_count"`
	UntestedCount int     `json:"untested_count"`
	WinnerCount   int     `json:"winner_count"`
	WinnerIDs     []int64 `json:"winner_ids,omitempty"`
}

// SupplierGroupSchedulingElectionAccountItem 是发生调度变更（或写库失败）的账号明细。
type SupplierGroupSchedulingElectionAccountItem struct {
	AccountID         int64   `json:"account_id"`
	AccountName       string  `json:"account_name"`
	Platform          string  `json:"platform,omitempty"`
	TestStatus        string  `json:"test_status,omitempty"`
	HealthyCount      int     `json:"healthy_count"`
	SchedulableBefore bool    `json:"schedulable_before"`
	SchedulableAfter  bool    `json:"schedulable_after"`
	Action            string  `json:"action"`
	Reason            string  `json:"reason,omitempty"`
	GroupIDs          []int64 `json:"group_ids,omitempty"`
	ErrorMessage      string  `json:"error_message,omitempty"`
}

type SupplierGroupSchedulingElectionResult struct {
	TopN             int                                          `json:"top_n"`
	GroupCount       int                                          `json:"group_count"`
	AccountCount     int                                          `json:"account_count"`
	EnabledCount     int                                          `json:"enabled_count"`
	DisabledCount    int                                          `json:"disabled_count"`
	UnchangedCount   int                                          `json:"unchanged_count"`
	SkippedCount     int                                          `json:"skipped_count"`
	FailedWriteCount int                                          `json:"failed_write_count"`
	Groups           []SupplierGroupSchedulingElectionGroupDetail `json:"groups"`
	Items            []SupplierGroupSchedulingElectionAccountItem `json:"items"`
}

// supplierGroupElectionAccount 是把同一账号在多个分组里的成员行聚合后的视图。
type supplierGroupElectionAccount struct {
	id                int64
	name              string
	platform          string
	schedulableBefore bool
	testStatus        string
	healthyCount      int
	lastTestedAt      time.Time
	groupIDs          []int64
	// winner 表示该账号在其所属的任意分组里被选为最优（union-enable）。
	winner bool
}

func (s *SupplierGroupSchedulingElectionService) Run(ctx context.Context, config SupplierGroupSchedulingElectionConfig, now time.Time) (SupplierGroupSchedulingElectionResult, error) {
	config = normalizeSupplierGroupSchedulingElectionConfig(config)
	if s == nil || s.repository == nil || s.accountStore == nil {
		return SupplierGroupSchedulingElectionResult{}, errors.New("分组择优调度依赖未初始化")
	}
	members, err := s.repository.ListGroupSchedulingElectionMembers(ctx)
	if err != nil {
		return SupplierGroupSchedulingElectionResult{}, err
	}

	disabledGroups := make(map[int64]struct{}, len(config.DisabledGroupIDs))
	for _, groupID := range config.DisabledGroupIDs {
		disabledGroups[groupID] = struct{}{}
	}

	// 1) 按分组归拢成员，同时聚合账号级视图（同一账号可能横跨多个分组）。
	membersByGroup := make(map[int64][]SupplierGroupSchedulingElectionMember)
	groupNames := make(map[int64]string)
	accounts := make(map[int64]*supplierGroupElectionAccount)
	groupOrder := make([]int64, 0)
	for _, member := range members {
		if _, skip := disabledGroups[member.GroupID]; skip {
			continue
		}
		if _, seen := membersByGroup[member.GroupID]; !seen {
			groupOrder = append(groupOrder, member.GroupID)
		}
		membersByGroup[member.GroupID] = append(membersByGroup[member.GroupID], member)
		groupNames[member.GroupID] = member.GroupName

		account := accounts[member.AccountID]
		if account == nil {
			account = &supplierGroupElectionAccount{
				id:                member.AccountID,
				name:              member.AccountName,
				platform:          member.Platform,
				schedulableBefore: member.Schedulable,
				testStatus:        strings.TrimSpace(member.LastTestStatus),
				healthyCount:      member.HealthyCount,
				lastTestedAt:      member.LastTestedAt,
			}
			accounts[member.AccountID] = account
		}
		account.groupIDs = append(account.groupIDs, member.GroupID)
	}
	sort.Slice(groupOrder, func(i, j int) bool { return groupOrder[i] < groupOrder[j] })

	result := SupplierGroupSchedulingElectionResult{
		TopN:         config.TopN,
		GroupCount:   len(groupOrder),
		AccountCount: len(accounts),
		Groups:       make([]SupplierGroupSchedulingElectionGroupDetail, 0, len(groupOrder)),
		Items:        make([]SupplierGroupSchedulingElectionAccountItem, 0),
	}

	// 2) 逐分组选出最优（前 N），并集式标记赢家（union-enable）。
	for _, groupID := range groupOrder {
		detail := SupplierGroupSchedulingElectionGroupDetail{
			GroupID:   groupID,
			GroupName: groupNames[groupID],
		}
		successMembers := make([]SupplierGroupSchedulingElectionMember, 0)
		for _, member := range membersByGroup[groupID] {
			detail.MemberCount++
			switch strings.TrimSpace(member.LastTestStatus) {
			case SupplierGroupSchedulingElectionTestStatusSuccess:
				detail.SuccessCount++
				successMembers = append(successMembers, member)
			case SupplierGroupSchedulingElectionTestStatusFailed:
				detail.FailedCount++
			default:
				detail.UntestedCount++
			}
		}
		sort.SliceStable(successMembers, func(i, j int) bool {
			return supplierGroupSchedulingElectionMemberLess(successMembers[i], successMembers[j])
		})
		winners := config.TopN
		if winners > len(successMembers) {
			winners = len(successMembers)
		}
		for index := 0; index < winners; index++ {
			winnerID := successMembers[index].AccountID
			if account := accounts[winnerID]; account != nil {
				account.winner = true
			}
			detail.WinnerCount++
			detail.WinnerIDs = append(detail.WinnerIDs, winnerID)
		}
		result.Groups = append(result.Groups, detail)
	}

	// 3) 账号级最终裁决：失败即关；成功赢家开、成功落选关；未测过跳过。
	accountIDs := make([]int64, 0, len(accounts))
	for accountID := range accounts {
		accountIDs = append(accountIDs, accountID)
	}
	sort.Slice(accountIDs, func(i, j int) bool { return accountIDs[i] < accountIDs[j] })

	for _, accountID := range accountIDs {
		account := accounts[accountID]
		item := SupplierGroupSchedulingElectionAccountItem{
			AccountID:         account.id,
			AccountName:       account.name,
			Platform:          account.platform,
			TestStatus:        account.testStatus,
			HealthyCount:      account.healthyCount,
			SchedulableBefore: account.schedulableBefore,
			SchedulableAfter:  account.schedulableBefore,
			Action:            SupplierGroupSchedulingElectionActionNone,
			GroupIDs:          account.groupIDs,
		}

		desiredSchedulable, action, reason := supplierGroupSchedulingElectionDecide(account)
		item.SchedulableAfter = desiredSchedulable
		item.Action = action
		item.Reason = reason

		if account.testStatus != SupplierGroupSchedulingElectionTestStatusSuccess &&
			account.testStatus != SupplierGroupSchedulingElectionTestStatusFailed {
			result.SkippedCount++
			// 未测过的账号不产生调度变更，也不计入明细，避免噪声。
			continue
		}

		if item.SchedulableAfter == item.SchedulableBefore {
			result.UnchangedCount++
			continue
		}

		if err := s.accountStore.SetSchedulable(ctx, account.id, item.SchedulableAfter); err != nil {
			item.SchedulableAfter = item.SchedulableBefore
			item.Action = SupplierGroupSchedulingElectionActionNone
			item.Reason = SupplierGroupSchedulingElectionReasonWriteFailed
			item.ErrorMessage = err.Error()
			result.FailedWriteCount++
			result.Items = append(result.Items, item)
			continue
		}

		switch item.Action {
		case SupplierGroupSchedulingElectionActionEnabled:
			result.EnabledCount++
		case SupplierGroupSchedulingElectionActionDisabled:
			result.DisabledCount++
		}
		result.Items = append(result.Items, item)
	}

	return result, nil
}

// supplierGroupSchedulingElectionDecide 给出账号的目标调度状态。
// 规则（复用 last_test_status 与 healthy_count，不重测）：
//   - failed  → 关（无条件）
//   - success → 在任一所属分组是最优则开；否则关（落选）
//   - 其它（未测过） → 保持原状
func supplierGroupSchedulingElectionDecide(account *supplierGroupElectionAccount) (bool, string, string) {
	switch account.testStatus {
	case SupplierGroupSchedulingElectionTestStatusFailed:
		if account.schedulableBefore {
			return false, SupplierGroupSchedulingElectionActionDisabled, SupplierGroupSchedulingElectionReasonFailed
		}
		return false, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonFailed
	case SupplierGroupSchedulingElectionTestStatusSuccess:
		if account.winner {
			if !account.schedulableBefore {
				return true, SupplierGroupSchedulingElectionActionEnabled, SupplierGroupSchedulingElectionReasonElected
			}
			return true, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonElected
		}
		if account.schedulableBefore {
			return false, SupplierGroupSchedulingElectionActionDisabled, SupplierGroupSchedulingElectionReasonNotElected
		}
		return false, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonNotElected
	default:
		return account.schedulableBefore, SupplierGroupSchedulingElectionActionNone, SupplierGroupSchedulingElectionReasonUntested
	}
}

// supplierGroupSchedulingElectionMemberLess 定义"更优"排序：连续成功次数多者优先。
// 次数并列时优先保留「当前已开启调度」的账号（黏性），只有被严格超过才换人——
// 否则两个次数长期咬死的账号会因并发测试导致 last_tested_at 轮流领先，赢家在两者间反复横跳、
// 被开启的账号跟着抖动。次数与调度状态都相同时，再按最近测试时间、账号 ID 兜底，保证确定性。
func supplierGroupSchedulingElectionMemberLess(a, b SupplierGroupSchedulingElectionMember) bool {
	if a.HealthyCount != b.HealthyCount {
		return a.HealthyCount > b.HealthyCount
	}
	if a.Schedulable != b.Schedulable {
		return a.Schedulable
	}
	if !a.LastTestedAt.Equal(b.LastTestedAt) {
		return a.LastTestedAt.After(b.LastTestedAt)
	}
	return a.AccountID < b.AccountID
}

func normalizeSupplierGroupSchedulingElectionConfig(config SupplierGroupSchedulingElectionConfig) SupplierGroupSchedulingElectionConfig {
	if config.TopN <= 0 {
		config.TopN = DefaultSupplierGroupSchedulingElectionTopN
	}
	if config.TopN > MaxSupplierGroupSchedulingElectionTopN {
		config.TopN = MaxSupplierGroupSchedulingElectionTopN
	}
	config.DisabledGroupIDs = uniquePositiveInt64s(config.DisabledGroupIDs)
	sort.Slice(config.DisabledGroupIDs, func(i, j int) bool {
		return config.DisabledGroupIDs[i] < config.DisabledGroupIDs[j]
	})
	return config
}
