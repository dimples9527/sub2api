package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

var ErrSupplierProviderSyncConflict = infraerrors.Conflict("SUPPLIER_PROVIDER_SYNC_CONFLICT", "supplier provider sync already running")
var ErrSupplierProviderGroupNotFound = infraerrors.NotFound("SUPPLIER_PROVIDER_GROUP_NOT_FOUND", "supplier provider group not found")
var ErrSupplierLocalGroupNotFound = infraerrors.NotFound("SUPPLIER_LOCAL_GROUP_NOT_FOUND", "active local group not found")
var ErrSupplierProviderMonitorTargetNotFound = infraerrors.NotFound("SUPPLIER_PROVIDER_MONITOR_TARGET_NOT_FOUND", "supplier provider monitor target not found")
var ErrSupplierProviderMonitorBindingInvalid = infraerrors.BadRequest("SUPPLIER_PROVIDER_MONITOR_BINDING_INVALID", "supplier provider monitor binding is invalid")
var ErrSupplierProviderGroupDeleteConflict = infraerrors.Conflict("SUPPLIER_PROVIDER_GROUP_DELETE_CONFLICT", "供应商上游分组记录不存在或已删除")
var ErrSupplierProviderAccountDeleteConflict = infraerrors.Conflict("SUPPLIER_PROVIDER_ACCOUNT_DELETE_CONFLICT", "供应商上游账号记录不存在或已删除")

type SupplierProviderAccountBindingGroup struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	SubscriptionType string  `json:"subscription_type"`
}

type SupplierProviderAccount struct {
	ID                          int64                                 `json:"id"`
	ProviderID                  int64                                 `json:"provider_id"`
	ProviderName                string                                `json:"provider_name"`
	UpstreamKey                 string                                `json:"upstream_account_key"`
	Name                        string                                `json:"name"`
	Status                      string                                `json:"status"`
	GroupKey                    string                                `json:"group_key"`
	GroupName                   string                                `json:"group_name"`
	Platform                    string                                `json:"platform,omitempty"`
	RateMultiplier              float64                               `json:"rate_multiplier"`
	RawStatus                   string                                `json:"raw_status"`
	Active                      bool                                  `json:"active"`
	LastSeenAt                  time.Time                             `json:"last_seen_at"`
	InactiveAt                  *time.Time                            `json:"inactive_at,omitempty"`
	LocalAccountMatchStatus     string                                `json:"local_account_match_status"`
	LocalAccountMatchCount      int                                   `json:"local_account_match_count"`
	LocalAccountID              *int64                                `json:"local_account_id,omitempty"`
	LocalAccountName            string                                `json:"local_account_name,omitempty"`
	LocalAccountPlatform        string                                `json:"local_account_platform,omitempty"`
	LocalAccountType            string                                `json:"local_account_type,omitempty"`
	PlatformOverride            string                                `json:"platform_override,omitempty"`
	EffectivePlatform           string                                `json:"effective_platform,omitempty"`
	LocalAccountPriority        *int                                  `json:"local_account_priority,omitempty"`
	LocalAccountStatus          string                                `json:"local_account_status,omitempty"`
	LocalAccountSchedulable     *bool                                 `json:"local_account_schedulable,omitempty"`
	LocalAccountLastTestStatus  string                                `json:"local_account_last_test_status,omitempty"`
	LocalAccountLastTestedAt    string                                `json:"local_account_last_tested_at,omitempty"`
	LocalAccountLastTestError   string                                `json:"local_account_last_test_error,omitempty"`
	GroupStatus                 string                                `json:"group_status,omitempty"`
	BindingGroups               []SupplierProviderAccountBindingGroup `json:"binding_groups"`
	SupplierCurrentBalance      float64                               `json:"supplier_current_balance"`
	SupplierTodayCost           float64                               `json:"supplier_today_cost"`
	GroupRecordID               *int64                                `json:"group_record_id,omitempty"`
	GroupRecordDeleteEligible   bool                                  `json:"group_record_delete_eligible"`
	AccountRecordDeleteEligible bool                                  `json:"account_record_delete_eligible"`
}

type SupplierProviderGroup struct {
	ID                              int64      `json:"id"`
	ProviderID                      int64      `json:"provider_id"`
	ProviderName                    string     `json:"provider_name"`
	UpstreamKey                     string     `json:"upstream_group_key"`
	Name                            string     `json:"name"`
	RateMultiplier                  float64    `json:"rate_multiplier"`
	RawStatus                       string     `json:"raw_status"`
	Active                          bool       `json:"active"`
	LocalGroupID                    *int64     `json:"local_group_id,omitempty"`
	LocalGroupName                  string     `json:"local_group_name,omitempty"`
	LocalGroupPlatform              string     `json:"local_group_platform,omitempty"`
	PlatformOverride                string     `json:"platform_override,omitempty"`
	EffectivePlatform               string     `json:"effective_platform,omitempty"`
	LocalRateMultiplier             *float64   `json:"local_rate_multiplier,omitempty"`
	LocalGroupStatus                string     `json:"local_group_status,omitempty"`
	AutoMatchIgnored                bool       `json:"auto_match_ignored"`
	AutoMatchStatus                 string     `json:"auto_match_status"`
	MatchedUpstreamName             string     `json:"matched_upstream_name,omitempty"`
	NameChangePending               bool       `json:"name_change_pending"`
	RateGuardSelected               bool       `json:"rate_guard_selected"`
	RateGuardEnabled                bool       `json:"rate_guard_enabled"`
	RateGuardSelectionMode          string     `json:"rate_guard_selection_mode"`
	RateGuardLastSnapshotAt         *time.Time `json:"rate_guard_last_snapshot_at,omitempty"`
	RateGuardLastCheckedAt          *time.Time `json:"rate_guard_last_checked_at,omitempty"`
	LocalGroupActiveMappingCount    int        `json:"local_group_active_mapping_count"`
	LocalGroupRateGuardGroupID      *int64     `json:"local_group_rate_guard_group_id,omitempty"`
	LocalGroupRateGuardGroupName    string     `json:"local_group_rate_guard_group_name,omitempty"`
	LocalGroupRateGuardProviderName string     `json:"local_group_rate_guard_provider_name,omitempty"`
	GroupSyncStatus                 string     `json:"group_sync_status"`
	LastGroupSyncAt                 *time.Time `json:"last_group_sync_at,omitempty"`
	AccountCount                    int        `json:"account_count"`
	KeySyncStatus                   string     `json:"key_sync_status"`
	KeyStatus                       string     `json:"key_status"`
	LastSeenAt                      time.Time  `json:"last_seen_at"`
	InactiveAt                      *time.Time `json:"inactive_at,omitempty"`
}

// SupplierProviderDataListParams 供应商上游数据列表查询参数。
// Status 为上游密钥业务状态：active/disabled/expired/quota_exhausted/unknown。
type SupplierProviderDataListParams struct {
	ProviderID  int64
	GroupID     int64
	Active      *bool
	Status      string
	KeyStatus   string
	Search      string
	Platform    string
	MatchStatus string
	RateStatus  string
	SortBy      string
	SortOrder   string
	Page        int
	PageSize    int
}

type SupplierProviderAccountListResult struct {
	Items    []SupplierProviderAccount `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}

type SupplierProviderGroupSummary struct {
	GroupCount           int64 `json:"group_count"`
	AccountCount         int64 `json:"account_count"`
	LinkedGroupCount     int64 `json:"linked_group_count"`
	UnlinkedGroupCount   int64 `json:"unlinked_group_count"`
	RateRiskCount        int64 `json:"rate_risk_count"`
	ActiveGroupCount     int64 `json:"active_group_count"`
	RemovedGroupCount    int64 `json:"removed_group_count"`
	CreatedKeyGroupCount int64 `json:"created_key_group_count"`
	AttentionGroupCount  int64 `json:"attention_group_count"`
}

type SupplierProviderGroupListResult struct {
	Items    []SupplierProviderGroup      `json:"items"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
	Summary  SupplierProviderGroupSummary `json:"summary"`
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
	AutomationRunRetentionDays        int
	SyncRunRetentionDays              int
	MetricRetentionDays               int
	DailyStatRetentionDays            int
	InactiveAccountDays               int
	InactiveGroupDays                 int
	AccountHealthHistoryRetentionDays int
}

type SupplierCleanupCounts struct {
	AutomationRuns       int
	SyncRuns             int
	MetricSnapshots      int
	DailyStats           int
	Accounts             int
	Groups               int
	AccountHealthHistory int
}

// SupplierProviderCostFallbackBalance 保存成本保底估算所需的余额基线。
// CurrentBalance 为当前余额（supplier_provider_runtime_stats.current_balance），
// DayStartBalance 为统计日 statDay 当天开始时的余额（前一统计日的最终余额）。
type SupplierProviderCostFallbackBalance struct {
	CurrentBalance  float64
	DayStartBalance float64
}

type SupplierProviderDataRepository interface {
	ListAccounts(ctx context.Context, params SupplierProviderDataListParams) (SupplierProviderAccountListResult, error)
	IsUniqueMatchedLocalAccount(ctx context.Context, localAccountID int64) (bool, error)
	GetLocalAccountEffectivePlatform(ctx context.Context, localAccountID int64) (string, error)
	GetLocalAccountPlatformOverride(ctx context.Context, localAccountID int64) (string, error)
	SetLocalAccountPlatformOverride(ctx context.Context, localAccountID int64, platform string) error
	ClearLocalAccountPlatformOverride(ctx context.Context, localAccountID int64) error
	ListGroups(ctx context.Context, params SupplierProviderDataListParams) (SupplierProviderGroupListResult, error)
	ListGroupHealthTrends(ctx context.Context, params SupplierProviderGroupHealthTrendParams) ([]SupplierProviderGroupHealthTrend, error)
	ListLocalGroupHealthTrends(ctx context.Context, params SupplierProviderGroupHealthTrendParams) ([]SupplierProviderGroupHealthTrend, error)
	ListMonitorTargets(ctx context.Context, params SupplierProviderMonitorTargetListParams) (SupplierProviderMonitorTargetListResult, error)
	BindMonitorTarget(ctx context.Context, monitorTargetID, localAccountID int64) error
	UnbindMonitorTarget(ctx context.Context, monitorTargetID int64) error
	ListGroupsForAutoMatch(ctx context.Context, providerID int64) ([]SupplierProviderGroup, error)
	GetGroupForAutoMatch(ctx context.Context, groupID int64) (SupplierProviderGroup, error)
	UpdateGroupMapping(ctx context.Context, groupID int64, localGroupID *int64) error
	DeleteGroup(ctx context.Context, groupID int64) error
	DeleteAccount(ctx context.Context, accountID int64) error
	ApplyAutoMatch(ctx context.Context, groupID, localGroupID int64, matchedUpstreamName string) (bool, error)
	UpdateAutoMatchState(ctx context.Context, groupID int64, status string, nameChangePending bool) error
	UpdateAutoMatchIgnored(ctx context.Context, groupID int64, ignored bool) error
	AcknowledgeNameChange(ctx context.Context, groupID int64, matchedUpstreamName string) error
	ListMappingsByLocalGroup(ctx context.Context, localGroupIDs []int64) ([]SupplierProviderGroup, error)
	GetGroupForRateGuard(ctx context.Context, groupID int64) (SupplierProviderGroup, error)
	SelectRateGuard(ctx context.Context, groupID int64, mode string) error
	ClearRateGuard(ctx context.Context, groupID int64, mode string) error
	SetRateGuardEnabled(ctx context.Context, groupID int64, enabled bool) error
	ListRateGuardCandidates(ctx context.Context) ([]SupplierRateGuardCandidate, error)
	ApplyRateGuard(ctx context.Context, input SupplierRateGuardApplyInput) (SupplierRateGuardApplyResult, error)
	MarkRateGuardChecked(ctx context.Context, mappingID int64, checkedAt time.Time) error
	ReplaceAccounts(ctx context.Context, providerID int64, items []SupplierProviderRemoteAccount, seenAt time.Time) (SupplierSyncCounts, error)
	ReplaceGroups(ctx context.Context, providerID int64, items []SupplierProviderRemoteGroup, seenAt time.Time) (SupplierSyncCounts, error)
	UpdateBalance(ctx context.Context, providerID int64, balance float64, seenAt time.Time) error
	UpdateCost(ctx context.Context, providerID int64, cost float64, seenAt time.Time) error
	// UpdateCostDetailed 写入成本生效值与上游原始值、偏差覆盖提示。
	UpdateCostDetailed(ctx context.Context, providerID int64, cost float64, rawUpstream *float64, warning *string, statDay time.Time) error
	// UpdateCostDetailedWithReview 在同一事务中写入每日成本和成本核对记录。
	UpdateCostDetailedWithReview(ctx context.Context, providerID int64, cost float64, rawUpstream *float64, warning *string, statDay time.Time, review SupplierProviderCostReviewSyncInput) error
	// GetLocalCostForDay 获取指定供应商指定统计日的本地成本；ok=false 表示无本地口径可校验。
	GetLocalCostForDay(ctx context.Context, providerID int64, day time.Time) (float64, bool, error)
	// GetBalanceDeltaForDay 获取指定供应商指定统计日的余额差值（当天第一条余额 - 当天最后一条余额）；
	// ok=false 表示当天没有足够的余额快照数据，无法计算差值。
	GetBalanceDeltaForDay(ctx context.Context, providerID int64, day time.Time) (float64, bool, error)
	// GetCostFallbackBalances 获取成本保底估算所需的余额基线；
	// ok=false 表示当前余额或当天起始余额缺失，无法进行保底估算。
	GetCostFallbackBalances(ctx context.Context, providerID int64, statDay time.Time) (SupplierProviderCostFallbackBalance, bool, error)
	CreateSyncRun(ctx context.Context, run *SupplierProviderSyncRun) error
	FinishSyncRun(ctx context.Context, run *SupplierProviderSyncRun) error
	UpdateSyncStatus(ctx context.Context, providerID int64, status, message string, syncedAt time.Time) error
	UpdateGroupSyncStatus(ctx context.Context, providerID int64, status, message string, syncedAt time.Time) error
	Cleanup(ctx context.Context, policy SupplierCleanupPolicy, now time.Time, batchSize int) (SupplierCleanupCounts, error)
}

// SupplierProviderRateSnapshotRepository 只负责更新上游账号倍率快照，避免守护任务覆盖完整同步字段。
type SupplierProviderRateSnapshotRepository interface {
	UpdateAccountRateSnapshot(ctx context.Context, providerID int64, upstreamKey string, rate float64, syncedAt time.Time) (bool, error)
}

type SupplierProviderRateSyncResult struct {
	ProviderID   int64     `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	CheckedCount int       `json:"checked_count"`
	UpdatedCount int       `json:"updated_count"`
	SkippedCount int       `json:"skipped_count"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	UpdatedKeys  []string  `json:"updated_keys,omitempty"`
	SyncedAt     time.Time `json:"synced_at"`
}

type SupplierProviderGroupAutoMatcher interface {
	AutoMatch(ctx context.Context, providerID int64) (SupplierGroupAutoMatchResult, error)
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
	SkippedCount   int                          `json:"skipped_count"`
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
	SupplierSyncStatusSkipped    = "skipped"

	SupplierSyncScopeAccounts = "accounts"
	SupplierSyncScopeGroups   = "groups"
	SupplierSyncScopeBalance  = "balance"
	SupplierSyncScopeCost     = "cost"
	SupplierSyncScopeMonitor  = "monitor"
	SupplierSyncScopeAll      = "all"
)

type SupplierProviderSyncService struct {
	providerRepo      SupplierProviderRepository
	dataRepo          SupplierProviderDataRepository
	rechargeRepo      SupplierProviderRechargeRepository
	remote            SupplierProviderRemoteClient
	encryptor         SecretEncryptor
	syncLock          SupplierProviderSyncLock
	groupMatcher      SupplierProviderGroupAutoMatcher
	thresholdProvider SupplierCostDeviationThresholdProvider
	costReviewService *SupplierProviderCostReviewService
}

func (s *SupplierProviderSyncService) SetGroupMatcher(matcher SupplierProviderGroupAutoMatcher) {
	if s != nil {
		s.groupMatcher = matcher
	}
}

// SetCostDeviationThresholdProvider 注入成本偏差覆盖阈值提供方，未注入时使用默认阈值。
func (s *SupplierProviderSyncService) SetCostDeviationThresholdProvider(provider SupplierCostDeviationThresholdProvider) {
	if s != nil {
		s.thresholdProvider = provider
	}
}

func (s *SupplierProviderSyncService) SetCostReviewService(service *SupplierProviderCostReviewService) {
	if s != nil {
		s.costReviewService = service
	}
}

func (s *SupplierProviderSyncService) syncCostReview(ctx context.Context, providerID int64, statDay time.Time, upstream, calculated, adopted *float64, effective float64, runID *int64, syncedAt time.Time) error {
	if s.costReviewService == nil {
		return nil
	}
	_, err := s.costReviewService.Sync(ctx, SupplierProviderCostReviewSyncInput{ProviderID: providerID, StatDate: statDay, UpstreamCost: upstream, CalculatedCost: calculated, AutoAdoptedCost: adopted, EffectiveCost: effective, SyncRunID: runID, SyncedAt: syncedAt})
	return err
}

// SetRechargeRepository 注入已落库的供应商充值记录仓储。
func (s *SupplierProviderSyncService) SetRechargeRepository(repo SupplierProviderRechargeRepository) {
	if s != nil {
		s.rechargeRepo = repo
	}
}

func (s *SupplierProviderSyncService) costDeviationThreshold(ctx context.Context) float64 {
	if s != nil && s.thresholdProvider != nil {
		return s.thresholdProvider.SupplierCostDeviationThreshold(ctx)
	}
	return DefaultSupplierCostDeviationThreshold
}

func NewSupplierProviderSyncService(providerRepo SupplierProviderRepository, dataRepo SupplierProviderDataRepository, remote SupplierProviderRemoteClient, encryptor SecretEncryptor, syncLock SupplierProviderSyncLock, rechargeRepo ...SupplierProviderRechargeRepository) *SupplierProviderSyncService {
	service := &SupplierProviderSyncService{
		providerRepo: providerRepo,
		dataRepo:     dataRepo,
		remote:       remote,
		encryptor:    encryptor,
		syncLock:     syncLock,
	}
	if len(rechargeRepo) > 0 {
		service.rechargeRepo = rechargeRepo[0]
	}
	return service
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

func (s *SupplierProviderSyncService) disableProviderAfterAuthFailure(ctx context.Context, providerID int64, syncedAt time.Time) error {
	return s.providerRepo.DisableAfterAuthFailure(ctx, providerID, supplierProviderAuthFailureDisableMessage, syncedAt)
}

func supplierProviderAuthFailureWithDisableError(authErr, disableErr error) error {
	if authErr == nil {
		return disableErr
	}
	if disableErr == nil {
		return authErr
	}
	// 以认证错误作为首个 %w，确保调用方仍可用 IsSupplierProviderAuthFailure 判断。
	return fmt.Errorf("%w；自动停用供应商失败: %v", authErr, disableErr)
}

func (s *SupplierProviderSyncService) SyncAccounts(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncStage(ctx, provider, password, SupplierSyncScopeAccounts, trigger, true)
	})
}

// SyncAccountRates 仅刷新已存在上游账号的倍率快照，不修改名称、状态、分组或 active 状态。
func (s *SupplierProviderSyncService) SyncAccountRates(ctx context.Context, providerID int64, trigger string) (SupplierProviderRateSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	_ = trigger
	provider, err := s.validSyncProvider(ctx, providerID)
	if err != nil {
		return SupplierProviderRateSyncResult{ProviderID: providerID, Status: SupplierSyncStatusFailed, Message: err.Error()}, err
	}
	rateRepo, ok := s.dataRepo.(SupplierProviderRateSnapshotRepository)
	if !ok {
		err := fmt.Errorf("supplier provider rate snapshot repository is required")
		return SupplierProviderRateSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Status: SupplierSyncStatusFailed, Message: err.Error()}, err
	}

	owner := uuid.NewString()
	if s.syncLock != nil {
		acquired, lockErr := s.syncLock.TryAcquireSyncLock(ctx, providerID, owner, 15*time.Minute)
		if lockErr != nil {
			err = fmt.Errorf("acquire supplier sync lock: %w", lockErr)
			return SupplierProviderRateSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Status: SupplierSyncStatusFailed, Message: err.Error()}, err
		}
		if !acquired {
			return SupplierProviderRateSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Status: SupplierSyncStatusFailed, Message: ErrSupplierProviderSyncConflict.Error()}, ErrSupplierProviderSyncConflict
		}
		defer func() { _ = s.syncLock.ReleaseSyncLock(context.Background(), providerID, owner) }()
	}

	syncedAt := time.Now()
	result := SupplierProviderRateSyncResult{
		ProviderID: provider.ID, ProviderName: provider.Name, Status: SupplierSyncStatusSuccess, SyncedAt: syncedAt,
		UpdatedKeys: make([]string, 0),
	}
	items, err := s.remote.FetchAccounts(ctx, provider, s.providerPassword(provider))
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
		if IsSupplierProviderAuthFailure(err) {
			result.Message = supplierProviderAuthFailureDisableMessage
			err = supplierProviderAuthFailureWithDisableError(err, s.disableProviderAfterAuthFailure(ctx, provider.ID, syncedAt))
		}
		return result, err
	}
	result.CheckedCount = len(items)
	for _, item := range items {
		key := supplierProviderRemoteAccountKey(item)
		if key == "" || math.IsNaN(item.RateMultiplier) || math.IsInf(item.RateMultiplier, 0) || item.RateMultiplier < 0 {
			result.SkippedCount++
			continue
		}
		updated, updateErr := rateRepo.UpdateAccountRateSnapshot(ctx, provider.ID, key, item.RateMultiplier, syncedAt)
		if updateErr != nil {
			result.Status = SupplierSyncStatusFailed
			result.Message = updateErr.Error()
			return result, updateErr
		}
		if !updated {
			result.SkippedCount++
			continue
		}
		result.UpdatedCount++
		result.UpdatedKeys = append(result.UpdatedKeys, key)
	}
	if result.SkippedCount > 0 {
		result.Status = SupplierSyncStatusPartial
		result.Message = fmt.Sprintf("倍率刷新完成，更新 %d 个，跳过 %d 个", result.UpdatedCount, result.SkippedCount)
	} else {
		result.Message = fmt.Sprintf("倍率刷新完成，更新 %d 个", result.UpdatedCount)
	}
	return result, nil
}

func supplierProviderRemoteAccountKey(item SupplierProviderRemoteAccount) string {
	key := strings.TrimSpace(item.Key)
	if key == "" {
		key = strings.ToLower(strings.Join(strings.Fields(item.Name), " "))
	}
	return key
}

func (s *SupplierProviderSyncService) SyncGroups(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncStage(ctx, provider, password, SupplierSyncScopeGroups, trigger, true)
	})
}

func (s *SupplierProviderSyncService) SyncBalance(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncStage(ctx, provider, password, SupplierSyncScopeBalance, trigger, true)
	})
}

// SupplierProviderCostBackfillItem 表示单日回补结果。
type SupplierProviderCostBackfillItem struct {
	ProviderID   int64   `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	ProviderType string  `json:"provider_type"`
	Date         string  `json:"date"`
	Status       string  `json:"status"`
	Cost         float64 `json:"cost,omitempty"`
	Message      string  `json:"message,omitempty"`
}

// SupplierProviderCostBackfillResult 是按时间范围回补上游成本的汇总。
type SupplierProviderCostBackfillResult struct {
	StartDate     string                             `json:"start_date"`
	EndDate       string                             `json:"end_date"`
	ProviderID    int64                              `json:"provider_id,omitempty"`
	ProviderCount int                                `json:"provider_count"`
	DayCount      int                                `json:"day_count"`
	SuccessCount  int                                `json:"success_count"`
	FailedCount   int                                `json:"failed_count"`
	SkippedCount  int                                `json:"skipped_count"`
	Items         []SupplierProviderCostBackfillItem `json:"items"`
	StartedAt     time.Time                          `json:"started_at"`
	FinishedAt    time.Time                          `json:"finished_at"`
}

// BackfillCosts 按闭区间日期从上游拉取成本并写入本地 daily_stats。
// NewAPI 支持按天查询历史成本；Sub2API 的历史日期使用余额差加当日充值回补。
func (s *SupplierProviderSyncService) BackfillCosts(ctx context.Context, startDate, endDate string, providerID int64, trigger string) (SupplierProviderCostBackfillResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	startedAt := time.Now()
	result := SupplierProviderCostBackfillResult{
		StartDate:  strings.TrimSpace(startDate),
		EndDate:    strings.TrimSpace(endDate),
		ProviderID: providerID,
		Items:      make([]SupplierProviderCostBackfillItem, 0),
		StartedAt:  startedAt,
	}

	start, end, err := parseSupplierCostBackfillRange(startDate, endDate)
	if err != nil {
		return result, err
	}
	result.StartDate = start.Format("2006-01-02")
	result.EndDate = end.Format("2006-01-02")
	result.DayCount = int(end.Sub(start).Hours()/24) + 1

	providers, err := s.resolveCostBackfillProviders(ctx, providerID)
	if err != nil {
		return result, err
	}
	result.ProviderCount = len(providers)
	trigger = normalizeSupplierSyncTrigger(trigger)

	for _, provider := range providers {
		itemResult, syncErr := s.backfillProviderCosts(ctx, provider, start, end, trigger)
		result.SuccessCount += itemResult.SuccessCount
		result.FailedCount += itemResult.FailedCount
		result.SkippedCount += itemResult.SkippedCount
		result.Items = append(result.Items, itemResult.Items...)
		if syncErr != nil && errors.Is(syncErr, ErrSupplierProviderSyncConflict) {
			// 整段被锁占用时，按天记为跳过，便于前端提示。
			for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 0, 1) {
				result.SkippedCount++
				result.Items = append(result.Items, SupplierProviderCostBackfillItem{
					ProviderID:   provider.ID,
					ProviderName: provider.Name,
					ProviderType: provider.ProviderType,
					Date:         cursor.Format("2006-01-02"),
					Status:       SupplierSyncStatusSkipped,
					Message:      syncErr.Error(),
				})
			}
			continue
		}
		if syncErr != nil {
			return result, syncErr
		}
	}

	// 成本数据已写入，清空趋势缓存，避免「重新获取」后读到旧数据。
	invalidateSupplierCostTrendCache()
	result.FinishedAt = time.Now()
	return result, nil
}

type supplierCostBackfillPartial struct {
	SuccessCount int
	FailedCount  int
	SkippedCount int
	Items        []SupplierProviderCostBackfillItem
}

func (s *SupplierProviderSyncService) backfillProviderCosts(ctx context.Context, provider *SupplierProvider, start, end time.Time, trigger string) (supplierCostBackfillPartial, error) {
	partial := supplierCostBackfillPartial{Items: make([]SupplierProviderCostBackfillItem, 0)}
	_, err := s.syncWithLock(ctx, provider.ID, func(locked *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(locked)
		runStarted := time.Now()
		run := &SupplierProviderSyncRun{
			ProviderID:    locked.ID,
			SyncScope:     SupplierSyncScopeCost,
			TriggerSource: trigger,
			Status:        SupplierSyncStatusRunning,
			StartedAt:     runStarted,
		}
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			return SupplierProviderSyncResult{}, fmt.Errorf("create supplier cost backfill run: %w", err)
		}

		supportsHistory := supplierProviderSupportsHistoricalCost(locked)
		today := supplierCostBackfillToday()
		threshold := s.costDeviationThreshold(ctx)
		var firstErr error
		authFailure := false
		sessionFailure := false
		for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 0, 1) {
			dateText := cursor.Format("2006-01-02")
			item := SupplierProviderCostBackfillItem{
				ProviderID:   locked.ID,
				ProviderName: locked.Name,
				ProviderType: locked.ProviderType,
				Date:         dateText,
			}

			historicalBalanceOnly := !supportsHistory && !sameSupplierCostStatDay(cursor, today)

			// 先计算余额差值加当日充值修正值，避免充值导致成本少算。
			balanceDelta, balanceOk, balanceErr := s.dataRepo.GetBalanceDeltaForDay(ctx, locked.ID, cursor)
			balanceCost := 0.0
			rechargeAmount := 0.0
			balanceUsable := false
			var rechargeErr error
			if balanceErr == nil && balanceOk {
				balanceCost, rechargeAmount, balanceUsable, rechargeErr = s.rechargeAdjustedBalanceEstimate(ctx, locked, password, cursor, balanceDelta)
			}
			if historicalBalanceOnly {
				if balanceUsable {
					message := fmt.Sprintf("历史日期不请求 Sub2API 当天成本接口，使用充值修正余额成本回补：余额差 %.2f + 充值 %.2f = %.2f", balanceDelta, rechargeAmount, balanceCost)
					if updateErr := s.dataRepo.UpdateCostDetailed(ctx, locked.ID, balanceCost, nil, &message, cursor); updateErr != nil {
						item.Status = SupplierSyncStatusFailed
						item.Message = updateErr.Error()
						partial.FailedCount++
						partial.Items = append(partial.Items, item)
						run.Counts.CheckedCount++
						if firstErr == nil {
							firstErr = updateErr
						}
						continue
					}
					item.Status = SupplierSyncStatusSuccess
					item.Cost = balanceCost
					item.Message = message
					partial.SuccessCount++
					partial.Items = append(partial.Items, item)
					run.Counts.CheckedCount++
					run.Counts.UpdatedCount++
					continue
				}

				if rechargeErr != nil {
					item.Status = SupplierSyncStatusFailed
					item.Message = fmt.Sprintf("当前供应商类型不支持查询历史上游成本，且充值记录查询失败，无法安全使用余额差回补：%v", rechargeErr)
					partial.FailedCount++
					partial.Items = append(partial.Items, item)
					run.Counts.CheckedCount++
					if firstErr == nil {
						firstErr = rechargeErr
					}
					continue
				}

				item.Status = SupplierSyncStatusSkipped
				item.Message = "当前供应商类型不支持查询历史上游成本，且没有可用的充值修正余额成本"
				if balanceErr != nil {
					item.Message = fmt.Sprintf("%s：%v", item.Message, balanceErr)
				}
				partial.SkippedCount++
				partial.Items = append(partial.Items, item)
				run.Counts.SkippedCount++
				continue
			}

			// 当天或支持历史成本的供应商先写入余额估算值，等待上游成本验证。
			if balanceUsable {
				warning := fmt.Sprintf("使用充值修正余额成本预填充：余额差 %.2f + 充值 %.2f = %.2f，等待上游成本验证", balanceDelta, rechargeAmount, balanceCost)
				_ = s.dataRepo.UpdateCostDetailed(ctx, locked.ID, balanceCost, nil, &warning, cursor)
				// 预填充失败不影响后续流程，忽略错误
			}

			cost, fetchErr := s.remote.FetchCost(ctx, locked, password, cursor)
			if fetchErr != nil {
				// 如果上游接口失败，但充值修正后的余额成本已预填充，则视为成功。
				if balanceUsable {
					item.Status = SupplierSyncStatusSuccess
					item.Cost = balanceCost
					item.Message = fmt.Sprintf("上游接口失败，已使用充值修正余额成本 %.2f 兜底（余额差 %.2f + 充值 %.2f）", balanceCost, balanceDelta, rechargeAmount)
					partial.SuccessCount++
					partial.Items = append(partial.Items, item)
					run.Counts.CheckedCount++
					run.Counts.UpdatedCount++
					continue
				}
				// 没有余额差值兜底，按原逻辑处理失败
				item.Status = SupplierSyncStatusFailed
				item.Message = fetchErr.Error()
				if rechargeErr != nil {
					item.Message = fmt.Sprintf("%s；充值记录查询失败，未使用余额差兜底：%v", item.Message, rechargeErr)
				}
				stopAfterFetchFailure := false
				if IsSupplierProviderAuthFailure(fetchErr) {
					authFailure = true
					stopAfterFetchFailure = true
					item.Message = supplierProviderAuthFailureDisableMessage
					fetchErr = supplierProviderAuthFailureWithDisableError(fetchErr, s.disableProviderAfterAuthFailure(ctx, locked.ID, time.Now()))
				} else if IsSupplierProviderSessionFailure(fetchErr) {
					sessionFailure = true
					stopAfterFetchFailure = true
				}
				partial.FailedCount++
				partial.Items = append(partial.Items, item)
				run.Counts.CheckedCount++
				if firstErr == nil {
					firstErr = fetchErr
				}
				if stopAfterFetchFailure {
					for next := cursor.AddDate(0, 0, 1); !next.After(end); next = next.AddDate(0, 0, 1) {
						partial.SkippedCount++
						partial.Items = append(partial.Items, SupplierProviderCostBackfillItem{
							ProviderID:   locked.ID,
							ProviderName: locked.Name,
							ProviderType: locked.ProviderType,
							Date:         next.Format("2006-01-02"),
							Status:       SupplierSyncStatusSkipped,
							Message:      item.Message,
						})
						run.Counts.SkippedCount++
					}
					break
				}
				continue
			}

			// 上游接口返回了数据，判断是否合理（偏差是否在阈值内）
			effective := cost
			rawUpstream := &cost
			var warning *string
			if balanceUsable {
				deviation := supplierCostDeviation(cost, balanceCost)
				if deviation > threshold {
					// 偏差超过阈值，保留充值修正后的余额成本，记录警告。
					effective = balanceCost
					msg := fmt.Sprintf("上游成本 %.2f 与充值修正余额成本 %.2f 偏差 %.1f%% 超过阈值，保留余额成本（余额差 %.2f + 充值 %.2f）", cost, balanceCost, deviation*100, balanceDelta, rechargeAmount)
					warning = &msg
				} else {
					// 偏差在阈值内，用上游成本覆盖
					effective = cost
					warning = nil
				}
			}
			if updateErr := s.dataRepo.UpdateCostDetailed(ctx, locked.ID, effective, rawUpstream, warning, cursor); updateErr != nil {
				item.Status = SupplierSyncStatusFailed
				item.Message = updateErr.Error()
				partial.FailedCount++
				partial.Items = append(partial.Items, item)
				run.Counts.CheckedCount++
				if firstErr == nil {
					firstErr = updateErr
				}
				continue
			}

			item.Status = SupplierSyncStatusSuccess
			item.Cost = effective
			partial.SuccessCount++
			partial.Items = append(partial.Items, item)
			run.Counts.CheckedCount++
			run.Counts.UpdatedCount++
		}

		finishedAt := time.Now()
		run.FinishedAt = &finishedAt
		switch {
		case authFailure || sessionFailure:
			run.Status = SupplierSyncStatusFailed
		case partial.FailedCount == 0 && partial.SuccessCount > 0:
			run.Status = SupplierSyncStatusSuccess
		case partial.SuccessCount > 0 && partial.FailedCount > 0:
			run.Status = SupplierSyncStatusPartial
		case partial.SuccessCount == 0 && partial.FailedCount > 0:
			run.Status = SupplierSyncStatusFailed
		default:
			run.Status = SupplierSyncStatusSuccess
		}
		if authFailure {
			run.ErrorMessage = supplierProviderAuthFailureDisableMessage
		} else if firstErr != nil {
			run.ErrorMessage = firstErr.Error()
		} else {
			run.ErrorMessage = supplierSyncMessage(run.Status)
		}
		var callbackErr error
		if finishErr := s.dataRepo.FinishSyncRun(ctx, run); finishErr != nil {
			if firstErr != nil && IsSupplierProviderAuthFailure(firstErr) {
				callbackErr = supplierProviderAuthFailureWithDisableError(firstErr, fmt.Errorf("finish supplier cost backfill run: %w", finishErr))
			} else if firstErr != nil && IsSupplierProviderSessionFailure(firstErr) {
				callbackErr = supplierProviderSessionFailureWithFinishError(firstErr, finishErr)
			} else {
				callbackErr = fmt.Errorf("finish supplier cost backfill run: %w", finishErr)
			}
		}
		_ = s.dataRepo.UpdateSyncStatus(ctx, locked.ID, run.Status, run.ErrorMessage, finishedAt)
		result := SupplierProviderSyncResult{
			ProviderID:   locked.ID,
			ProviderName: locked.Name,
			Scope:        SupplierSyncScopeCost,
			Status:       run.Status,
			Message:      run.ErrorMessage,
			Counts:       run.Counts,
			StartedAt:    runStarted,
			FinishedAt:   finishedAt,
		}
		if callbackErr != nil {
			return result, callbackErr
		}
		if authFailure || sessionFailure {
			return result, firstErr
		}
		return result, nil
	})
	if err != nil {
		return partial, err
	}
	return partial, nil
}

func (s *SupplierProviderSyncService) resolveCostBackfillProviders(ctx context.Context, providerID int64) ([]*SupplierProvider, error) {
	if providerID > 0 {
		provider, err := s.validSyncProvider(ctx, providerID)
		if err != nil {
			return nil, err
		}
		return []*SupplierProvider{provider}, nil
	}
	enabled := true
	providers, _, err := s.providerRepo.List(ctx, SupplierProviderListParams{Enabled: &enabled, Page: 1, PageSize: 1000})
	if err != nil {
		return nil, fmt.Errorf("list enabled supplier providers: %w", err)
	}
	return providers, nil
}

func parseSupplierCostBackfillRange(startDate, endDate string) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), loc)
	if err != nil {
		return time.Time{}, time.Time{}, infraerrors.BadRequest("INVALID_COST_BACKFILL_START_DATE", "start_date must be YYYY-MM-DD")
	}
	end, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), loc)
	if err != nil {
		return time.Time{}, time.Time{}, infraerrors.BadRequest("INVALID_COST_BACKFILL_END_DATE", "end_date must be YYYY-MM-DD")
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, infraerrors.BadRequest("INVALID_COST_BACKFILL_RANGE", "end_date must be on or after start_date")
	}
	today := supplierCostBackfillToday()
	if end.After(today) {
		end = today
	}
	if start.After(end) {
		start = end
	}
	if int(end.Sub(start).Hours()/24) > 89 {
		start = end.AddDate(0, 0, -89)
	}
	return start, end, nil
}

func supplierCostBackfillToday() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}

func sameSupplierCostStatDay(a, b time.Time) bool {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	aa := a.In(loc)
	bb := b.In(loc)
	return aa.Year() == bb.Year() && aa.Month() == bb.Month() && aa.Day() == bb.Day()
}

func supplierProviderSupportsHistoricalCost(provider *SupplierProvider) bool {
	if provider == nil {
		return false
	}
	// NewAPI 成本 URL 支持按 start/end timestamp 查历史；Sub2API 目前仅 today_actual_cost。
	return normalizeSupplierProviderType(provider.ProviderType) == SupplierProviderTypeNewAPI
}

func (s *SupplierProviderSyncService) SyncCost(ctx context.Context, providerID int64, day time.Time, trigger string) (SupplierProviderSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		return s.syncCostStage(ctx, provider, password, day, trigger, true)
	})
}

func (s *SupplierProviderSyncService) TestEndpoint(ctx context.Context, providerID int64, scope string) (SupplierProviderEndpointTestResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceEndpointTest)
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
	if err == nil && supplierProviderEndpointAuthFailure(result) {
		err = wrapSupplierProviderAuthFailure(fmt.Errorf("supplier provider endpoint test returned HTTP %d: %s", result.HTTPStatus, result.Error))
	}
	if err != nil {
		if IsSupplierProviderAuthFailure(err) {
			result.Error = supplierProviderAuthFailureDisableMessage
			err = supplierProviderAuthFailureWithDisableError(err, s.disableProviderAfterAuthFailure(ctx, provider.ID, time.Now()))
		}
		return result, err
	}
	result.ProviderID = provider.ID
	result.Scope = scope
	result.SensitiveRedacted = true
	return result, nil
}

func supplierProviderEndpointAuthFailure(result SupplierProviderEndpointTestResult) bool {
	text := strings.ToLower(strings.TrimSpace(result.Error + " " + result.ResponseSummary))
	if supplierSub2APIProbeBlockedText(text) {
		return false
	}
	if result.HTTPStatus == 401 {
		return true
	}
	return supplierSub2APIAuthPhrase(text) || supplierNewAPIAuthPhrase(text)
}

func (s *SupplierProviderSyncService) SyncAll(ctx context.Context, providerID int64, trigger string) (SupplierProviderSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	return s.syncWithLock(ctx, providerID, func(provider *SupplierProvider) (SupplierProviderSyncResult, error) {
		password := s.providerPassword(provider)
		startedAt := time.Now()
		run := &SupplierProviderSyncRun{ProviderID: provider.ID, SyncScope: SupplierSyncScopeAll, TriggerSource: normalizeSupplierSyncTrigger(trigger), Status: SupplierSyncStatusRunning, StartedAt: startedAt}
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			err = fmt.Errorf("create supplier sync run: %w", err)
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return SupplierProviderSyncResult{}, err
		}
		result := SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: SupplierSyncScopeAll, Status: SupplierSyncStatusSuccess, StartedAt: startedAt}
		authFailure := false
		sessionFailure := false
		for _, stageFn := range []func() (SupplierProviderSyncStage, error){
			func() (SupplierProviderSyncStage, error) {
				return s.syncStageAsSummary(ctx, provider, password, SupplierSyncScopeAccounts)
			},
			func() (SupplierProviderSyncStage, error) {
				return s.syncStageAsSummary(ctx, provider, password, SupplierSyncScopeGroups)
			},
			func() (SupplierProviderSyncStage, error) {
				return s.syncStageAsSummary(ctx, provider, password, SupplierSyncScopeBalance)
			},
			func() (SupplierProviderSyncStage, error) {
				return s.syncCostStageAsSummary(ctx, provider, password, time.Now())
			},
		} {
			stage, stageErr := stageFn()
			result.Stages = append(result.Stages, stage)
			result.Counts.CheckedCount += stage.Counts.CheckedCount
			result.Counts.CreatedCount += stage.Counts.CreatedCount
			result.Counts.UpdatedCount += stage.Counts.UpdatedCount
			result.Counts.SkippedCount += stage.Counts.SkippedCount
			if IsSupplierProviderAuthFailure(stageErr) {
				authFailure = true
				result.Status = SupplierSyncStatusFailed
				break
			}
			if IsSupplierProviderSessionFailure(stageErr) {
				sessionFailure = true
				result.Status = SupplierSyncStatusFailed
				result.Message = stage.Message
				break
			}
			if stage.Status == SupplierSyncStatusFailed {
				result.Status = SupplierSyncStatusPartial
			}
		}
		if !authFailure && !sessionFailure && allStagesFailed(result.Stages) {
			result.Status = SupplierSyncStatusFailed
		}
		result.FinishedAt = time.Now()
		if authFailure {
			result.Message = supplierProviderAuthFailureDisableMessage
		} else if !sessionFailure {
			result.Message = supplierSyncMessage(result.Status)
		}
		finishedAt := result.FinishedAt
		run.Status = result.Status
		run.Counts = result.Counts
		run.ErrorMessage = result.Message
		run.FinishedAt = &finishedAt
		if finishErr := s.dataRepo.FinishSyncRun(ctx, run); finishErr != nil {
			err := fmt.Errorf("finish supplier sync run: %w", finishErr)
			result.Status = SupplierSyncStatusFailed
			result.Message = err.Error()
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return result, err
		}
		if statusErr := s.dataRepo.UpdateSyncStatus(ctx, provider.ID, result.Status, result.Message, finishedAt); statusErr != nil {
			err := fmt.Errorf("update supplier sync status: %w", statusErr)
			result.Status = SupplierSyncStatusFailed
			result.Message = err.Error()
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return result, err
		}
		return result, nil
	})
}

func (s *SupplierProviderSyncService) SyncAllEnabled(ctx context.Context, trigger string) (SupplierProviderBatchSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	enabled := true
	providers, _, err := s.providerRepo.List(ctx, SupplierProviderListParams{Enabled: &enabled, Page: 1, PageSize: 1000})
	if err != nil {
		return SupplierProviderBatchSyncResult{}, fmt.Errorf("list enabled supplier providers: %w", err)
	}
	result := SupplierProviderBatchSyncResult{ProcessedCount: len(providers), Results: make([]SupplierProviderSyncResult, 0, len(providers))}
	for _, provider := range providers {
		item, err := s.SyncAll(ctx, provider.ID, trigger)
		if err != nil {
			now := time.Now()
			if errors.Is(err, ErrSupplierProviderSyncConflict) {
				item = SupplierProviderSyncResult{
					ProviderID:   provider.ID,
					ProviderName: provider.Name,
					Scope:        SupplierSyncScopeAll,
					Status:       SupplierSyncStatusSkipped,
					Message:      "供应商同步正在执行，已跳过，将在下次自动任务重试",
					StartedAt:    now,
					FinishedAt:   now,
				}
			} else {
				item = SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: SupplierSyncScopeAll, Status: SupplierSyncStatusFailed, Message: err.Error(), StartedAt: now, FinishedAt: now}
			}
		}
		switch item.Status {
		case SupplierSyncStatusSuccess:
			result.SuccessCount++
		case SupplierSyncStatusSkipped:
			result.SkippedCount++
		default:
			result.FailedCount++
		}
		result.Results = append(result.Results, item)
	}
	return result, nil
}

func (s *SupplierProviderSyncService) syncWithLock(ctx context.Context, providerID int64, fn func(*SupplierProvider) (SupplierProviderSyncResult, error)) (SupplierProviderSyncResult, error) {
	SupplierSyncProgress(ctx, SupplierSyncProgressStagePrepare, "正在校验供应商配置并准备同步", nil)
	provider, err := s.validSyncProvider(ctx, providerID)
	if err != nil {
		SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePrepare, err)
		return SupplierProviderSyncResult{}, err
	}
	owner := uuid.NewString()
	if s.syncLock != nil {
		acquired, err := s.syncLock.TryAcquireSyncLock(ctx, providerID, owner, 15*time.Minute)
		if err != nil {
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePrepare, err)
			return SupplierProviderSyncResult{}, fmt.Errorf("acquire supplier sync lock: %w", err)
		}
		if !acquired {
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePrepare, ErrSupplierProviderSyncConflict)
			return SupplierProviderSyncResult{}, ErrSupplierProviderSyncConflict
		}
		defer func() { _ = s.syncLock.ReleaseSyncLock(context.Background(), providerID, owner) }()
	}
	SupplierSyncProgressOK(ctx, SupplierSyncProgressStagePrepare, "供应商配置校验通过，已开始同步")
	return fn(provider)
}

func (s *SupplierProviderSyncService) validSyncProvider(ctx context.Context, providerID int64) (*SupplierProvider, error) {
	provider, err := s.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	switch normalizeSupplierProviderType(provider.ProviderType) {
	case SupplierProviderTypeSub2API, SupplierProviderTypeNewAPI:
	default:
		return nil, ErrSupplierProviderInvalid
	}
	if !provider.Enabled {
		return nil, ErrSupplierProviderDisabled
	}
	return provider, nil
}

func (s *SupplierProviderSyncService) syncStage(ctx context.Context, provider *SupplierProvider, password, scope, trigger string, createRun bool) (SupplierProviderSyncResult, error) {
	if scope == SupplierSyncScopeCost {
		return s.syncCostStage(ctx, provider, password, time.Now(), trigger, createRun)
	}
	progressStage := supplierSyncProgressStageForScope(scope)
	startedAt := time.Now()
	result := SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: scope, Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	run := &SupplierProviderSyncRun{ProviderID: provider.ID, SyncScope: scope, TriggerSource: normalizeSupplierSyncTrigger(trigger), Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	if createRun {
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			err = fmt.Errorf("create supplier sync run: %w", err)
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return result, err
		}
	}
	var counts SupplierSyncCounts
	var err error
	SupplierSyncProgress(ctx, SupplierSyncProgressStageSession, "正在获取或复用上游登录会话", nil)
	if scope == SupplierSyncScopeGroups {
		err = s.dataRepo.UpdateGroupSyncStatus(ctx, provider.ID, SupplierSyncStatusRunning, "", startedAt)
		if err != nil {
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
		}
	}
	if err == nil {
		counts, err = s.executeStage(ctx, provider, password, scope, startedAt)
	}
	result.Counts = counts
	result.FinishedAt = time.Now()
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
		if IsSupplierProviderAuthFailure(err) {
			result.Message = supplierProviderAuthFailureDisableMessage
			err = supplierProviderAuthFailureWithDisableError(err, s.disableProviderAfterAuthFailure(ctx, provider.ID, result.FinishedAt))
		}
	} else {
		result.Status = SupplierSyncStatusSuccess
		result.Message = supplierSyncMessage(result.Status)
		SupplierSyncProgressOK(ctx, progressStage, fmt.Sprintf("%s同步完成", supplierSyncProgressScopeLabel(scope)))
	}
	if scope == SupplierSyncScopeGroups {
		if statusErr := s.dataRepo.UpdateGroupSyncStatus(ctx, provider.ID, result.Status, result.Message, result.FinishedAt); statusErr != nil && err == nil {
			err = fmt.Errorf("update supplier group sync status: %w", statusErr)
			result.Status = SupplierSyncStatusFailed
			result.Message = err.Error()
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
		}
	}
	if createRun {
		finishedAt := result.FinishedAt
		run.Status = result.Status
		run.Counts = result.Counts
		run.ErrorMessage = result.Message
		run.FinishedAt = &finishedAt
		if finishErr := s.dataRepo.FinishSyncRun(ctx, run); finishErr != nil && err == nil {
			err = fmt.Errorf("finish supplier sync run: %w", finishErr)
			result.Status = SupplierSyncStatusFailed
			result.Message = err.Error()
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
		}
		if statusErr := s.dataRepo.UpdateSyncStatus(ctx, provider.ID, result.Status, result.Message, finishedAt); statusErr != nil {
			statusErr = fmt.Errorf("update supplier sync status: %w", statusErr)
			if err == nil {
				err = statusErr
				result.Status = SupplierSyncStatusFailed
				result.Message = statusErr.Error()
			}
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, statusErr)
		}
	}
	return result, err
}

func (s *SupplierProviderSyncService) syncCostStage(ctx context.Context, provider *SupplierProvider, password string, day time.Time, trigger string, createRun bool) (SupplierProviderSyncResult, error) {
	progressStage := SupplierSyncProgressStageCost
	startedAt := time.Now()
	result := SupplierProviderSyncResult{ProviderID: provider.ID, ProviderName: provider.Name, Scope: SupplierSyncScopeCost, Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	run := &SupplierProviderSyncRun{ProviderID: provider.ID, SyncScope: SupplierSyncScopeCost, TriggerSource: normalizeSupplierSyncTrigger(trigger), Status: SupplierSyncStatusRunning, StartedAt: startedAt}
	if createRun {
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			err = fmt.Errorf("create supplier sync run: %w", err)
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return result, err
		}
	}
	var syncRunID *int64
	if run.ID > 0 {
		syncRunID = &run.ID
	}
	syncAt := time.Now().UTC()
	SupplierSyncProgress(ctx, SupplierSyncProgressStageSession, "正在获取或复用上游登录会话", nil)
	SupplierSyncProgress(ctx, progressStage, "正在请求上游成本接口", nil)
	statDay := day
	if statDay.IsZero() {
		statDay = startedAt
	}
	cost, requestErr := s.remote.FetchCost(ctx, provider, password, statDay)
	var err error
	if requestErr == nil {
		SupplierSyncProgress(ctx, progressStage, "成本接口请求成功，正在写入本地数据", nil)
		SupplierSyncProgress(ctx, SupplierSyncProgressStagePersist, "正在写入成本数据", nil)
		// 按成本归属日写入 daily_stats，避免历史回补落到当天；
		// 上游成本与本地成本差距过大时直接改写为本地成本并记录提示。
		threshold := s.costDeviationThreshold(ctx)
		effective, rawUpstream, warning := s.resolveCostDeviation(ctx, provider, password, cost, statDay, threshold)
		calculated := cost
		if local, ok, localErr := s.dataRepo.GetLocalCostForDay(ctx, provider.ID, statDay); localErr == nil && ok && local >= 0 {
			calculated = local
		}
		upstream := cost
		reviewInput := SupplierProviderCostReviewSyncInput{ProviderID: provider.ID, StatDate: statDay, UpstreamCost: &upstream, CalculatedCost: &calculated, AutoAdoptedCost: &calculated, EffectiveCost: effective, SyncRunID: syncRunID, SyncedAt: syncAt}
		if s.costReviewService != nil {
			err = s.dataRepo.UpdateCostDetailedWithReview(ctx, provider.ID, effective, rawUpstream, warning, statDay, reviewInput)
		} else {
			err = s.dataRepo.UpdateCostDetailed(ctx, provider.ID, effective, rawUpstream, warning, statDay)
		}
		if err == nil {
			// 定时同步写入成功后同样失效成本趋势缓存。
			invalidateSupplierCostTrendCache()
		}
	} else {
		// 成本接口获取不到数据时，尝试用当天起始余额 - 当前余额做保底估算；
		// 认证失败、会话失败属于需要停用供应商或无法继续同步的情况，不走保底。
		if !IsSupplierProviderAuthFailure(requestErr) && !IsSupplierProviderSessionFailure(requestErr) {
			bal, ok, balErr := s.dataRepo.GetCostFallbackBalances(ctx, provider.ID, statDay)
			if balErr == nil && ok {
				rawBalanceDelta := bal.DayStartBalance - bal.CurrentBalance
				fallbackCost, rechargeAmount, usable, rechargeErr := s.rechargeAdjustedBalanceEstimate(ctx, provider, password, statDay, rawBalanceDelta)
				if rechargeErr != nil {
					err = fmt.Errorf("成本接口失败且充值记录查询失败，无法安全使用余额差兜底: %w", rechargeErr)
				} else if usable {
					SupplierSyncProgress(ctx, progressStage, "成本接口请求失败，正在用余额差额保底估算成本", nil)
					SupplierSyncProgress(ctx, SupplierSyncProgressStagePersist, "正在写入保底估算的成本数据", nil)
					cost = fallbackCost
					warning := fmt.Sprintf("上游成本接口失败，使用充值修正余额成本兜底：余额差 %.2f + 充值 %.2f = %.2f", rawBalanceDelta, rechargeAmount, fallbackCost)
					calculated := cost
					reviewInput := SupplierProviderCostReviewSyncInput{ProviderID: provider.ID, StatDate: statDay, CalculatedCost: &calculated, AutoAdoptedCost: &calculated, EffectiveCost: cost, SyncRunID: syncRunID, SyncedAt: syncAt}
					if s.costReviewService != nil {
						err = s.dataRepo.UpdateCostDetailedWithReview(ctx, provider.ID, cost, nil, &warning, statDay, reviewInput)
					} else {
						err = s.dataRepo.UpdateCostDetailed(ctx, provider.ID, cost, nil, &warning, statDay)
					}
					if err == nil {
						// 保底估算写入成功后同样失效成本趋势缓存。
						invalidateSupplierCostTrendCache()
					}
				} else {
					err = requestErr
				}
			} else {
				err = requestErr
			}
		} else {
			err = requestErr
		}
	}
	result.Counts = SupplierSyncCounts{CheckedCount: 1, UpdatedCount: boolToInt(err == nil)}
	result.FinishedAt = time.Now()
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		result.Message = err.Error()
		if IsSupplierProviderAuthFailure(err) {
			result.Message = supplierProviderAuthFailureDisableMessage
			err = supplierProviderAuthFailureWithDisableError(err, s.disableProviderAfterAuthFailure(ctx, provider.ID, result.FinishedAt))
		}
		if requestErr == nil {
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
		} else {
			SupplierSyncProgressFail(ctx, progressStage, err)
		}
	} else {
		result.Status = SupplierSyncStatusSuccess
		result.Message = supplierSyncMessage(result.Status)
		SupplierSyncProgressOK(ctx, SupplierSyncProgressStagePersist, "成本数据写入完成")
		SupplierSyncProgressOK(ctx, progressStage, "成本同步完成")
	}
	if createRun {
		finishedAt := result.FinishedAt
		run.Status = result.Status
		run.Counts = result.Counts
		run.ErrorMessage = result.Message
		run.FinishedAt = &finishedAt
		if finishErr := s.dataRepo.FinishSyncRun(ctx, run); finishErr != nil && err == nil {
			err = fmt.Errorf("finish supplier sync run: %w", finishErr)
			result.Status = SupplierSyncStatusFailed
			result.Message = err.Error()
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
		}
		if statusErr := s.dataRepo.UpdateSyncStatus(ctx, provider.ID, result.Status, result.Message, finishedAt); statusErr != nil {
			statusErr = fmt.Errorf("update supplier sync status: %w", statusErr)
			if err == nil {
				err = statusErr
				result.Status = SupplierSyncStatusFailed
				result.Message = statusErr.Error()
			}
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, statusErr)
		}
	}
	return result, err
}

func (s *SupplierProviderSyncService) executeStage(ctx context.Context, provider *SupplierProvider, password, scope string, seenAt time.Time) (SupplierSyncCounts, error) {
	progressStage := supplierSyncProgressStageForScope(scope)
	switch scope {
	case SupplierSyncScopeAccounts:
		SupplierSyncProgress(ctx, progressStage, "正在请求上游 API Key 接口", nil)
		items, err := s.remote.FetchAccounts(ctx, provider, password)
		if err != nil {
			SupplierSyncProgressFail(ctx, progressStage, err)
			return SupplierSyncCounts{}, err
		}
		SupplierSyncProgress(ctx, progressStage, fmt.Sprintf("API Key 接口请求成功，获取 %d 条记录", len(items)), nil)
		SupplierSyncProgress(ctx, SupplierSyncProgressStagePersist, "正在写入 API Key 数据", nil)
		counts, err := s.dataRepo.ReplaceAccounts(ctx, provider.ID, items, seenAt)
		if err != nil {
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return counts, err
		}
		SupplierSyncProgressOK(ctx, SupplierSyncProgressStagePersist, "API Key 数据写入完成")
		return counts, nil
	case SupplierSyncScopeGroups:
		SupplierSyncProgress(ctx, progressStage, "正在请求上游分组接口", nil)
		items, err := s.remote.FetchGroups(ctx, provider, password)
		if err != nil {
			SupplierSyncProgressFail(ctx, progressStage, err)
			return SupplierSyncCounts{}, err
		}
		SupplierSyncProgress(ctx, progressStage, fmt.Sprintf("分组接口请求成功，获取 %d 条记录", len(items)), nil)
		SupplierSyncProgress(ctx, SupplierSyncProgressStagePersist, "正在写入分组数据", nil)
		counts, err := s.dataRepo.ReplaceGroups(ctx, provider.ID, items, seenAt)
		if err != nil {
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return counts, err
		}
		if s.groupMatcher != nil {
			SupplierSyncProgress(ctx, SupplierSyncProgressStagePersist, "正在更新分组自动匹配关系", nil)
			if _, err := s.groupMatcher.AutoMatch(ctx, provider.ID); err != nil {
				SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
				return counts, fmt.Errorf("auto match supplier groups: %w", err)
			}
		}
		SupplierSyncProgressOK(ctx, SupplierSyncProgressStagePersist, "分组数据写入完成")
		return counts, nil
	case SupplierSyncScopeBalance:
		SupplierSyncProgress(ctx, progressStage, "正在请求上游余额接口", nil)
		balance, err := s.remote.FetchBalance(ctx, provider, password)
		if err != nil {
			SupplierSyncProgressFail(ctx, progressStage, err)
			return SupplierSyncCounts{}, err
		}
		SupplierSyncProgress(ctx, progressStage, "余额接口请求成功，正在写入本地数据", nil)
		SupplierSyncProgress(ctx, SupplierSyncProgressStagePersist, "正在写入余额数据", nil)
		if err := s.dataRepo.UpdateBalance(ctx, provider.ID, balance, seenAt); err != nil {
			SupplierSyncProgressFail(ctx, SupplierSyncProgressStagePersist, err)
			return SupplierSyncCounts{}, err
		}
		SupplierSyncProgressOK(ctx, SupplierSyncProgressStagePersist, "余额数据写入完成")
		return SupplierSyncCounts{CheckedCount: 1, UpdatedCount: 1}, nil
	default:
		SupplierSyncProgressFail(ctx, SupplierSyncProgressStageError, ErrSupplierProviderInvalid)
		return SupplierSyncCounts{}, ErrSupplierProviderInvalid
	}
}

func (s *SupplierProviderSyncService) syncStageAsSummary(ctx context.Context, provider *SupplierProvider, password, scope string) (SupplierProviderSyncStage, error) {
	result, err := s.syncStage(ctx, provider, password, scope, SupplierSyncTriggerScheduled, false)
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		if IsSupplierProviderAuthFailure(err) {
			result.Message = supplierProviderAuthFailureDisableMessage
		} else {
			result.Message = err.Error()
		}
	}
	return SupplierProviderSyncStage{Scope: scope, Status: result.Status, Message: result.Message, Counts: result.Counts, EndpointResult: s.lastEndpointResult(provider.ID, scope)}, err
}

func (s *SupplierProviderSyncService) syncCostStageAsSummary(ctx context.Context, provider *SupplierProvider, password string, day time.Time) (SupplierProviderSyncStage, error) {
	result, err := s.syncCostStage(ctx, provider, password, day, SupplierSyncTriggerScheduled, false)
	if err != nil {
		result.Status = SupplierSyncStatusFailed
		if IsSupplierProviderAuthFailure(err) {
			result.Message = supplierProviderAuthFailureDisableMessage
		} else {
			result.Message = err.Error()
		}
	}
	return SupplierProviderSyncStage{Scope: SupplierSyncScopeCost, Status: result.Status, Message: result.Message, Counts: result.Counts, EndpointResult: s.lastEndpointResult(provider.ID, SupplierSyncScopeCost)}, err
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

// supplierCostFallbackEstimate 根据余额基线计算成本保底估算值。
// 仅当当天起始余额大于当前余额时返回有效估算（余额未减少视为无成本消耗）。
func supplierCostFallbackEstimate(bal SupplierProviderCostFallbackBalance) (float64, bool) {
	estimated := bal.DayStartBalance - bal.CurrentBalance
	if estimated <= 0 {
		return 0, false
	}
	return estimated, true
}

// rechargeAdjustedBalanceEstimate 使用余额差值和指定统计日充值金额估算实际消耗。
func (s *SupplierProviderSyncService) rechargeAdjustedBalanceEstimate(ctx context.Context, provider *SupplierProvider, password string, statDay time.Time, balanceDelta float64) (estimated float64, rechargeAmount float64, usable bool, err error) {
	if s.rechargeRepo != nil {
		location := statDay.Location()
		if location == nil {
			location = time.UTC
		}
		start := time.Date(statDay.Year(), statDay.Month(), statDay.Day(), 0, 0, 0, 0, location)
		end := start.AddDate(0, 0, 1)
		localRecords, listErr := s.rechargeRepo.List(ctx, SupplierProviderRechargeListParams{
			ProviderID: provider.ID,
			Start:      start,
			End:        end,
			Page:       1,
			PageSize:   1,
		})
		if listErr == nil && localRecords.Total > 0 {
			rechargeAmount = localRecords.TotalAmount
			estimated = balanceDelta + rechargeAmount
			if estimated <= 0 {
				return 0, rechargeAmount, false, nil
			}
			return estimated, rechargeAmount, true, nil
		}
	}
	rechargeClient, ok := s.remote.(SupplierProviderRemoteRechargeClient)
	if !ok {
		return 0, 0, false, fmt.Errorf("supplier provider remote client does not support recharge history")
	}
	rechargeAmount, err = rechargeClient.FetchRechargeAmount(ctx, provider, password, statDay)
	if err != nil {
		return 0, 0, false, err
	}
	if rechargeAmount < 0 {
		return 0, 0, false, fmt.Errorf("supplier provider recharge amount must not be negative")
	}
	estimated = balanceDelta + rechargeAmount
	if estimated <= 0 {
		return 0, rechargeAmount, false, nil
	}
	return estimated, rechargeAmount, true, nil
}

// resolveCostDeviation 计算写入成本时的偏差覆盖结果：上游成本与本地成本差距超过阈值时，
// 优先使用充值修正余额成本（当天第一条余额快照 - 当天最后一条余额快照 + 当天充值）；
// 如果余额差值不可用，则回退到本地成本。rawUpstream 始终为接口原始上游值。
func (s *SupplierProviderSyncService) resolveCostDeviation(ctx context.Context, provider *SupplierProvider, password string, upstream float64, statDay time.Time, threshold float64) (effective float64, rawUpstream *float64, warning *string) {
	raw := upstream
	rawUpstream = &raw
	if local, ok, err := s.dataRepo.GetLocalCostForDay(ctx, provider.ID, statDay); err == nil && ok && local > 0 {
		if supplierCostDeviation(upstream, local) > threshold {
			// 偏差过大时，优先尝试用充值修正后的余额成本覆盖。
			if balanceDelta, balanceOK, balanceErr := s.dataRepo.GetBalanceDeltaForDay(ctx, provider.ID, statDay); balanceErr == nil && balanceOK {
				if balanceCost, rechargeAmount, usable, rechargeErr := s.rechargeAdjustedBalanceEstimate(ctx, provider, password, statDay, balanceDelta); rechargeErr == nil && usable {
					msg := fmt.Sprintf("上游成本 %.2f 与本地成本 %.2f 偏差过大，已用充值修正余额成本 %.2f 覆盖（余额差 %.2f + 充值 %.2f）", upstream, local, balanceCost, balanceDelta, rechargeAmount)
					return balanceCost, rawUpstream, &msg
				}
			}
			// 如果没有余额差值数据，回退到本地成本
			msg := supplierCostDeviationWarning(upstream, local)
			return local, rawUpstream, &msg
		}
	}
	return upstream, rawUpstream, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *SupplierProviderSyncService) RefreshToken(ctx context.Context, providerID int64) (SupplierProviderAuthToken, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceManual)
	provider, err := s.validSyncProvider(ctx, providerID)
	if err != nil {
		return SupplierProviderAuthToken{}, err
	}
	logger.LegacyPrintf("supplier_provider_sync_service", "manual auth action provider_id=%d provider_code=%s provider_type=%s newapi_auth_mode=%s cookie_session=%t", provider.ID, provider.Code, provider.ProviderType, provider.NewAPIAuthMode, normalizeSupplierProviderType(provider.ProviderType) == SupplierProviderTypeNewAPI && supplierNewAPIUsesCookieSession(provider))
	if normalizeSupplierProviderType(provider.ProviderType) == SupplierProviderTypeNewAPI && supplierNewAPIUsesCookieSession(provider) {
		reauthenticator, ok := s.remote.(SupplierProviderRemoteReauthenticator)
		if !ok {
			return SupplierProviderAuthToken{}, infraerrors.BadRequest("SUPPLIER_PROVIDER_REAUTH_UNSUPPORTED", "current supplier type does not support manual reauthentication")
		}
		return reauthenticator.Reauthenticate(ctx, provider, s.providerPassword(provider))
	}
	refresher, ok := s.remote.(SupplierProviderRemoteTokenRefresher)
	if !ok {
		return SupplierProviderAuthToken{}, infraerrors.BadRequest("SUPPLIER_PROVIDER_TOKEN_REFRESH_UNSUPPORTED", "current supplier type does not support manual token refresh")
	}
	return refresher.RefreshToken(ctx, provider)
}
