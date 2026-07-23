package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var (
	ErrSupplierAccountRateInvalid = errors.New("供应商上游账号倍率无效")
	ErrSupplierRateScaleInvalid   = errors.New("供应商倍率缩放无效")
)

type SupplierAccountRateGuardMode string

const (
	SupplierAccountRateGuardModePreview SupplierAccountRateGuardMode = "preview"
	SupplierAccountRateGuardModeExecute SupplierAccountRateGuardMode = "execute"

	SupplierAccountRateGuardMatchMatched   = "matched"
	SupplierAccountRateGuardMatchConflict  = "conflict"
	SupplierAccountRateGuardMatchUnmatched = "unmatched"

	SupplierAccountRateGuardLogResultPlanned = "planned"
	SupplierAccountRateGuardLogResultUnbound = "unbound"
	SupplierAccountRateGuardLogResultFailed  = "failed"
	SupplierAccountRateGuardLogResultSkipped = "skipped"
)

type SupplierAccountRateGuardGroup struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type SupplierAccountRateGuardCandidate struct {
	ProviderID          int64                           `json:"provider_id"`
	ProviderName        string                          `json:"provider_name"`
	ProviderAccountID   int64                           `json:"supplier_provider_account_id"`
	UpstreamAccountKey  string                          `json:"upstream_account_key"`
	UpstreamAccountName string                          `json:"upstream_account_name"`
	RawRate             float64                         `json:"raw_rate"`
	RateScale           float64                         `json:"rate_scale"`
	LocalAccountID      int64                           `json:"local_account_id"`
	LocalAccountName    string                          `json:"local_account_name"`
	MatchStatus         string                          `json:"match_status"`
	MatchCount          int                             `json:"match_count"`
	ReverseMatchCount   int                             `json:"reverse_match_count"`
	Schedulable         bool                            `json:"schedulable"`
	Groups              []SupplierAccountRateGuardGroup `json:"groups"`
}

type SupplierAccountRateGuardUnbindLog struct {
	ID                    int64     `json:"id"`
	RunID                 int64     `json:"run_id"`
	ProviderID            int64     `json:"provider_id"`
	ProviderName          string    `json:"provider_name"`
	ProviderAccountID     int64     `json:"supplier_provider_account_id"`
	UpstreamAccountKey    string    `json:"upstream_account_key"`
	UpstreamAccountName   string    `json:"upstream_account_name"`
	LocalAccountID        int64     `json:"local_account_id"`
	LocalAccountName      string    `json:"local_account_name"`
	LocalGroupID          int64     `json:"local_group_id"`
	LocalGroupName        string    `json:"local_group_name"`
	RawUpstreamRate       float64   `json:"raw_upstream_rate"`
	RateScale             float64   `json:"rate_scale"`
	EffectiveUpstreamRate float64   `json:"effective_upstream_rate"`
	LocalGroupRate        float64   `json:"local_group_rate"`
	Mode                  string    `json:"mode"`
	Result                string    `json:"result"`
	BeforeBound           bool      `json:"before_bound"`
	AfterBound            bool      `json:"after_bound"`
	BeforeSchedulable     *bool     `json:"before_schedulable,omitempty"`
	AfterSchedulable      *bool     `json:"after_schedulable,omitempty"`
	ErrorMessage          string    `json:"error_message"`
	CreatedAt             time.Time `json:"created_at"`
}

type SupplierAccountRateGuardUnbindLogListParams struct {
	RunID          int64
	ProviderID     int64
	LocalAccountID int64
	Search         string
	Result         string
	Page           int
	PageSize       int
}

type SupplierAccountRateGuardUnbindLogListResult struct {
	Items    []SupplierAccountRateGuardUnbindLog `json:"items"`
	Total    int64                               `json:"total"`
	Page     int                                 `json:"page"`
	PageSize int                                 `json:"page_size"`
}

type SupplierAccountRateGuardResult struct {
	Mode                     string `json:"mode"`
	CheckedProviders         int    `json:"checked_providers"`
	RateSyncSuccessProviders int    `json:"rate_sync_success_providers"`
	RateSyncFailedProviders  int    `json:"rate_sync_failed_providers"`
	CheckedAccounts          int    `json:"checked_accounts"`
	RiskGroups               int    `json:"risk_groups"`
	UnboundGroups            int    `json:"unbound_groups"`
	DisabledAccounts         int    `json:"disabled_accounts"`
	Skipped                  int    `json:"skipped"`
	Failed                   int    `json:"failed"`
}

type SupplierAccountRateGuardRateSyncer interface {
	SyncAccountRates(ctx context.Context, providerID int64, trigger string) (SupplierProviderRateSyncResult, error)
}

type SupplierAccountRateGuardRepository interface {
	ListAccountRateGuardCandidates(ctx context.Context, providerID int64, upstreamKeys []string) ([]SupplierAccountRateGuardCandidate, error)
	CreateAccountRateGuardUnbindLogs(ctx context.Context, logs []SupplierAccountRateGuardUnbindLog) error
	ListAccountRateGuardUnbindLogs(ctx context.Context, params SupplierAccountRateGuardUnbindLogListParams) (SupplierAccountRateGuardUnbindLogListResult, error)
}

type AccountRateGuardGroupRemovalResult struct {
	RemovedGroupIDs    []int64 `json:"removed_group_ids"`
	RemainingGroupIDs  []int64 `json:"remaining_group_ids"`
	SchedulableBefore  bool    `json:"schedulable_before"`
	SchedulableAfter   bool    `json:"schedulable_after"`
	SchedulableChanged bool    `json:"schedulable_changed"`
}

type AccountRateGuardGroupRemover interface {
	RemoveAccountGroupsForRateGuard(ctx context.Context, accountID int64, groupIDs []int64) (AccountRateGuardGroupRemovalResult, error)
}

type SupplierAccountRateGuardService struct {
	providerRepo SupplierProviderRepository
	rateSyncer   SupplierAccountRateGuardRateSyncer
	repo         SupplierAccountRateGuardRepository
	remover      AccountRateGuardGroupRemover
}

func NewSupplierAccountRateGuardService(providerRepo SupplierProviderRepository, rateSyncer SupplierAccountRateGuardRateSyncer, repo SupplierAccountRateGuardRepository, remover AccountRateGuardGroupRemover) *SupplierAccountRateGuardService {
	return &SupplierAccountRateGuardService{providerRepo: providerRepo, rateSyncer: rateSyncer, repo: repo, remover: remover}
}

func effectiveSupplierAccountRate(rawRate, scale float64) (float64, error) {
	if math.IsNaN(rawRate) || math.IsInf(rawRate, 0) || rawRate < 0 {
		return 0, ErrSupplierAccountRateInvalid
	}
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 {
		return 0, ErrSupplierRateScaleInvalid
	}
	return rawRate * scale, nil
}

func (s *SupplierAccountRateGuardService) Run(ctx context.Context, runID int64, mode SupplierAccountRateGuardMode, now time.Time) (SupplierAccountRateGuardResult, error) {
	result := SupplierAccountRateGuardResult{Mode: string(mode)}
	if mode != SupplierAccountRateGuardModePreview && mode != SupplierAccountRateGuardModeExecute {
		return result, ErrSupplierProviderInvalid
	}
	enabled := true
	providers, _, err := s.providerRepo.List(ctx, SupplierProviderListParams{Enabled: &enabled, Page: 1, PageSize: 1000})
	if err != nil {
		return result, fmt.Errorf("查询启用供应商失败: %w", err)
	}
	result.CheckedProviders = len(providers)
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		syncResult, syncErr := s.rateSyncer.SyncAccountRates(ctx, provider.ID, SupplierSyncTriggerScheduled)
		if syncErr != nil {
			result.RateSyncFailedProviders++
			continue
		}
		result.RateSyncSuccessProviders++
		if len(syncResult.UpdatedKeys) == 0 {
			continue
		}
		candidates, listErr := s.repo.ListAccountRateGuardCandidates(ctx, provider.ID, syncResult.UpdatedKeys)
		if listErr != nil {
			result.Failed++
			continue
		}
		for _, candidate := range candidates {
			result.CheckedAccounts++
			logs, processErr := s.processCandidate(ctx, runID, mode, now, candidate, &result)
			if len(logs) > 0 {
				if logErr := s.repo.CreateAccountRateGuardUnbindLogs(ctx, logs); logErr != nil {
					return result, fmt.Errorf("保存账号倍率守护解绑日志失败: %w", logErr)
				}
			}
			if processErr != nil {
				result.Failed++
			}
		}
	}
	if result.CheckedProviders > 0 && result.RateSyncFailedProviders == result.CheckedProviders {
		return result, errors.New("所有供应商账号倍率刷新均失败")
	}
	return result, nil
}

func (s *SupplierAccountRateGuardService) processCandidate(ctx context.Context, runID int64, mode SupplierAccountRateGuardMode, now time.Time, candidate SupplierAccountRateGuardCandidate, result *SupplierAccountRateGuardResult) ([]SupplierAccountRateGuardUnbindLog, error) {
	base := supplierAccountRateGuardLogBase(runID, mode, now, candidate)
	if candidate.MatchStatus != SupplierAccountRateGuardMatchMatched || candidate.LocalAccountID <= 0 || candidate.ReverseMatchCount > 1 {
		base.Result = SupplierAccountRateGuardLogResultSkipped
		base.ErrorMessage = supplierAccountRateGuardMatchMessage(candidate)
		result.Skipped++
		return []SupplierAccountRateGuardUnbindLog{base}, nil
	}
	effectiveRate, err := effectiveSupplierAccountRate(candidate.RawRate, candidate.RateScale)
	if err != nil {
		base.Result = SupplierAccountRateGuardLogResultSkipped
		base.ErrorMessage = err.Error()
		result.Skipped++
		return []SupplierAccountRateGuardUnbindLog{base}, nil
	}
	base.EffectiveUpstreamRate = effectiveRate
	riskGroups := make([]SupplierAccountRateGuardGroup, 0)
	for _, group := range candidate.Groups {
		if group.RateMultiplier > effectiveRate+1e-9 {
			riskGroups = append(riskGroups, group)
		}
	}
	if len(riskGroups) == 0 {
		return nil, nil
	}
	result.RiskGroups += len(riskGroups)
	logs := make([]SupplierAccountRateGuardUnbindLog, 0, len(riskGroups))
	if mode == SupplierAccountRateGuardModePreview {
		for _, group := range riskGroups {
			logItem := base
			logItem.LocalGroupID = group.ID
			logItem.LocalGroupName = group.Name
			logItem.LocalGroupRate = group.RateMultiplier
			logItem.Result = SupplierAccountRateGuardLogResultPlanned
			logs = append(logs, logItem)
		}
		return logs, nil
	}

	groupIDs := make([]int64, 0, len(riskGroups))
	for _, group := range riskGroups {
		groupIDs = append(groupIDs, group.ID)
	}
	removal, removeErr := s.remover.RemoveAccountGroupsForRateGuard(ctx, candidate.LocalAccountID, groupIDs)
	removed := make(map[int64]struct{}, len(removal.RemovedGroupIDs))
	for _, groupID := range removal.RemovedGroupIDs {
		removed[groupID] = struct{}{}
	}
	if removeErr == nil && removal.SchedulableChanged && !removal.SchedulableAfter {
		result.DisabledAccounts++
	}
	for _, group := range riskGroups {
		logItem := base
		logItem.LocalGroupID = group.ID
		logItem.LocalGroupName = group.Name
		logItem.LocalGroupRate = group.RateMultiplier
		if removeErr != nil {
			logItem.Result = SupplierAccountRateGuardLogResultFailed
			logItem.ErrorMessage = removeErr.Error()
			logs = append(logs, logItem)
			continue
		}
		before := removal.SchedulableBefore
		after := removal.SchedulableAfter
		logItem.BeforeSchedulable = &before
		logItem.AfterSchedulable = &after
		if _, ok := removed[group.ID]; ok {
			logItem.Result = SupplierAccountRateGuardLogResultUnbound
			logItem.AfterBound = false
			result.UnboundGroups++
		} else {
			logItem.Result = SupplierAccountRateGuardLogResultSkipped
			logItem.ErrorMessage = "执行时绑定关系已不存在"
			result.Skipped++
		}
		logs = append(logs, logItem)
	}
	return logs, removeErr
}

func supplierAccountRateGuardLogBase(runID int64, mode SupplierAccountRateGuardMode, now time.Time, candidate SupplierAccountRateGuardCandidate) SupplierAccountRateGuardUnbindLog {
	beforeSchedulable := candidate.Schedulable
	return SupplierAccountRateGuardUnbindLog{
		RunID: runID, ProviderID: candidate.ProviderID, ProviderName: candidate.ProviderName,
		ProviderAccountID: candidate.ProviderAccountID, UpstreamAccountKey: candidate.UpstreamAccountKey,
		UpstreamAccountName: candidate.UpstreamAccountName, LocalAccountID: candidate.LocalAccountID,
		LocalAccountName: candidate.LocalAccountName, RawUpstreamRate: candidate.RawRate,
		RateScale: candidate.RateScale, Mode: string(mode), BeforeBound: true, AfterBound: true,
		BeforeSchedulable: &beforeSchedulable, AfterSchedulable: &beforeSchedulable, CreatedAt: now,
	}
}

func supplierAccountRateGuardMatchMessage(candidate SupplierAccountRateGuardCandidate) string {
	if candidate.ReverseMatchCount > 1 {
		return "多个上游账号匹配同一本地账号"
	}
	switch strings.TrimSpace(candidate.MatchStatus) {
	case SupplierAccountRateGuardMatchConflict:
		return "本地账号匹配冲突"
	case SupplierAccountRateGuardMatchUnmatched:
		return "未匹配到本地账号"
	default:
		return "本地账号匹配状态无效"
	}
}
