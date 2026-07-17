package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	SettingKeyUpstreamAccountSyncRecords     = "upstream_account_sync_records"
	SettingKeyUpstreamAccountRateGuardConfig = "upstream_account_rate_guard_config"

	UpstreamAccountSyncActionCreate   = "create"
	UpstreamAccountSyncActionUpdate   = "update"
	UpstreamAccountSyncActionNoop     = "noop"
	UpstreamAccountSyncActionSkip     = "skip"
	UpstreamAccountSyncActionConflict = "conflict"

	UpstreamAccountSyncTriggerManualSync         = "manual_sync"
	UpstreamAccountSyncTriggerScheduledRateGuard = "scheduled_rate_guard"
	UpstreamAccountSyncTriggerManualRateGuard    = "manual_rate_guard"

	DefaultUpstreamAccountRateGuardIntervalSeconds = 3600
	MinUpstreamAccountRateGuardIntervalSeconds     = 1
	upstreamAccountSyncProviderKeysCacheTTL        = 30 * time.Second
	upstreamAccountSyncProviderKeysFetchTimeout    = 8 * time.Second
	upstreamAccountSyncProviderKeysSlowLogDuration = 1 * time.Second
)

type UpstreamAccountSyncAccountManager interface {
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]Account, int64, error)
	CreateAccount(ctx context.Context, input *CreateAccountInput) (*Account, error)
	UpdateAccount(ctx context.Context, id int64, input *UpdateAccountInput) (*Account, error)
	SetAccountSchedulable(ctx context.Context, id int64, schedulable bool) (*Account, error)
}

type upstreamAccountSyncStoredProviderSource interface {
	getStoredProvider(ctx context.Context, slug string) (UpstreamProviderConfig, error)
}

type upstreamAccountSyncBoundGroupLoader interface {
	ListGroupsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64][]*Group, error)
}

type UpstreamAccountSyncRequest struct {
	CreateMissing  bool                              `json:"create_missing"`
	UpdateExisting bool                              `json:"update_existing"`
	ApplyRateGuard bool                              `json:"apply_rate_guard"`
	TriggerSource  string                            `json:"trigger_source,omitempty"`
	SelectedItems  []UpstreamAccountSyncSelectedItem `json:"selected_items,omitempty"`
}

type UpstreamAccountSyncSelectedItem struct {
	ProviderSlug    string `json:"provider_slug"`
	UpstreamKeyName string `json:"upstream_key_name"`
	CreateMissing   bool   `json:"create_missing"`
	UpdateExisting  bool   `json:"update_existing"`
	ApplyRateGuard  bool   `json:"apply_rate_guard"`
}

type UpstreamAccountRateGuardConfig struct {
	Enabled           bool       `json:"enabled"`
	IntervalSeconds   int        `json:"interval_seconds"`
	IgnoredAccountIDs []int64    `json:"ignored_account_ids,omitempty"`
	LastRunAt         *time.Time `json:"last_run_at,omitempty"`
	LastRunStatus     string     `json:"last_run_status,omitempty"`
	LastRunMessage    string     `json:"last_run_message,omitempty"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type UpstreamAccountSyncSummary struct {
	UpstreamKeyCount    int `json:"upstream_key_count"`
	MatchedAccountCount int `json:"matched_account_count"`
	CreateCount         int `json:"create_count"`
	UpdateCount         int `json:"update_count"`
	SkipCount           int `json:"skip_count"`
	ConflictCount       int `json:"conflict_count"`
	RateViolationCount  int `json:"rate_violation_count"`
	UnboundGroupCount   int `json:"unbound_group_count"`
}

type UpstreamAccountSyncItem struct {
	Action                 string                               `json:"action"`
	ProviderSlug           string                               `json:"provider_slug"`
	ProviderName           string                               `json:"provider_name"`
	ProviderBaseURL        string                               `json:"provider_base_url,omitempty"`
	UpstreamKeyName        string                               `json:"upstream_key_name"`
	UpstreamAPIKey         string                               `json:"upstream_api_key,omitempty"`
	UpstreamBaseURL        string                               `json:"upstream_base_url,omitempty"`
	ProviderFetchError     string                               `json:"provider_fetch_error,omitempty"`
	LocalAccountName       string                               `json:"local_account_name"`
	MatchedAccountID       *int64                               `json:"matched_account_id,omitempty"`
	MatchedAccountName     string                               `json:"matched_account_name,omitempty"`
	UpstreamGroupName      string                               `json:"upstream_group_name"`
	UpstreamRateMultiplier float64                              `json:"upstream_rate_multiplier"`
	LocalGroupID           *int64                               `json:"local_group_id,omitempty"`
	LocalGroupName         string                               `json:"local_group_name,omitempty"`
	LocalRateMultiplier    *float64                             `json:"local_rate_multiplier,omitempty"`
	RateViolation          bool                                 `json:"rate_violation"`
	RateGuardIgnored       bool                                 `json:"rate_guard_ignored,omitempty"`
	UnboundGroupIDs        []int64                              `json:"unbound_group_ids,omitempty"`
	UnboundGroupNames      []string                             `json:"unbound_group_names,omitempty"`
	SkipReason             string                               `json:"skip_reason,omitempty"`
	ConflictAccountIDs     []int64                              `json:"conflict_account_ids,omitempty"`
	ConflictAccounts       []UpstreamAccountSyncConflictAccount `json:"conflict_accounts,omitempty"`
	BoundGroups            []UpstreamAccountSyncBoundGroup      `json:"bound_groups,omitempty"`
	ChangeDetails          []UpstreamAccountSyncChangeDetail    `json:"change_details,omitempty"`
	Execution              UpstreamAccountSyncExecutionResult   `json:"execution,omitempty"`
}

type UpstreamAccountSyncConflictAccount struct {
	ID          int64                           `json:"id"`
	Name        string                          `json:"name"`
	BoundGroups []UpstreamAccountSyncBoundGroup `json:"bound_groups,omitempty"`
}

type UpstreamAccountSyncBoundGroup struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	RateViolation  bool    `json:"rate_violation"`
}

type UpstreamAccountSyncChangeDetail struct {
	Kind       string   `json:"kind"`
	Field      string   `json:"field,omitempty"`
	Label      string   `json:"label,omitempty"`
	Before     string   `json:"before,omitempty"`
	After      string   `json:"after,omitempty"`
	GroupIDs   []int64  `json:"group_ids,omitempty"`
	GroupNames []string `json:"group_names,omitempty"`
}

type UpstreamAccountSyncExecutionResult struct {
	Executed          bool     `json:"executed,omitempty"`
	Action            string   `json:"action,omitempty"`
	AccountID         *int64   `json:"account_id,omitempty"`
	AccountName       string   `json:"account_name,omitempty"`
	UnboundGroupIDs   []int64  `json:"unbound_group_ids,omitempty"`
	UnboundGroupNames []string `json:"unbound_group_names,omitempty"`
}

type UpstreamAccountSyncResult struct {
	DefaultProvider UpstreamProviderConfig      `json:"default_provider"`
	Providers       []UpstreamProviderConfig    `json:"providers"`
	Summary         UpstreamAccountSyncSummary  `json:"summary"`
	Items           []UpstreamAccountSyncItem   `json:"items"`
	Warnings        []string                    `json:"warnings,omitempty"`
	Records         []UpstreamAccountSyncRecord `json:"records"`
}

type UpstreamAccountSyncRecord struct {
	ProviderSlug       string                            `json:"provider_slug"`
	ProviderName       string                            `json:"provider_name"`
	CreatedCount       int                               `json:"created_count"`
	UpdatedCount       int                               `json:"updated_count"`
	SkippedCount       int                               `json:"skipped_count"`
	ConflictCount      int                               `json:"conflict_count"`
	RateViolationCount int                               `json:"rate_violation_count"`
	UnboundGroupCount  int                               `json:"unbound_group_count"`
	CreatedAt          time.Time                         `json:"created_at"`
	TriggerSource      string                            `json:"trigger_source,omitempty"`
	Error              string                            `json:"error,omitempty"`
	UnbindDetails      []UpstreamAccountSyncUnbindDetail `json:"unbind_details,omitempty"`
}

type UpstreamAccountSyncUnbindDetail struct {
	ProviderSlug            string   `json:"provider_slug"`
	ProviderName            string   `json:"provider_name"`
	UpstreamKeyName         string   `json:"upstream_key_name"`
	MatchedLocalAccountID   int64    `json:"matched_local_account_id"`
	MatchedLocalAccountName string   `json:"matched_local_account_name"`
	UpstreamGroupName       string   `json:"upstream_group_name"`
	UpstreamRateMultiplier  float64  `json:"upstream_rate_multiplier"`
	LocalMinRateMultiplier  float64  `json:"local_min_rate_multiplier"`
	UnboundGroupIDs         []int64  `json:"unbound_group_ids"`
	UnboundGroupNames       []string `json:"unbound_group_names"`
	RemainingGroupIDs       []int64  `json:"remaining_group_ids"`
	TriggerSource           string   `json:"trigger_source,omitempty"`
	Handled                 bool     `json:"handled,omitempty"`
}

type UpstreamAccountSyncService struct {
	providerSource UpstreamManagementProviderSource
	groupRepo      GroupRepository
	accountManager UpstreamAccountSyncAccountManager
	settingRepo    SettingRepository
	previewCache   UpstreamAccountSyncPreviewCache
	keysCacheMu    sync.Mutex
	keysCache      map[string]upstreamAccountSyncProviderKeysCacheEntry

	previewRefreshMu       sync.Mutex
	previewRefreshInFlight bool
}

type UpstreamAccountSyncPreviewCache interface {
	Get(ctx context.Context) (UpstreamAccountSyncResult, bool, error)
	Set(ctx context.Context, result UpstreamAccountSyncResult) error
	Delete(ctx context.Context) error
}

type upstreamAccountSyncProviderKeysCacheEntry struct {
	keys      []UpstreamProviderKey
	warnings  []string
	expiresAt time.Time
}

type upstreamAccountSyncProviderKeysResult struct {
	keys     []UpstreamProviderKey
	warnings []string
	err      error
}

type upstreamAccountSyncRecordStats struct {
	providerSlug       string
	providerName       string
	createdCount       int
	updatedCount       int
	skippedCount       int
	conflictCount      int
	rateViolationCount int
	unboundGroupCount  int
	error              string
	unbindDetails      []UpstreamAccountSyncUnbindDetail
}

func NewUpstreamAccountSyncService(
	providerSource UpstreamManagementProviderSource,
	groupRepo GroupRepository,
	accountManager UpstreamAccountSyncAccountManager,
	settingRepo SettingRepository,
) *UpstreamAccountSyncService {
	return &UpstreamAccountSyncService{
		providerSource: providerSource,
		groupRepo:      groupRepo,
		accountManager: accountManager,
		settingRepo:    settingRepo,
	}
}

func (s *UpstreamAccountSyncService) SetPreviewCache(cache UpstreamAccountSyncPreviewCache) {
	if s == nil {
		return
	}
	s.previewCache = cache
}

func (s *UpstreamAccountSyncService) Preview(ctx context.Context) (UpstreamAccountSyncResult, error) {
	if s != nil && s.previewCache != nil {
		if result, ok, err := s.previewCache.Get(ctx); err == nil && ok {
			result = s.applyCachedProviderState(ctx, result)
			records, err := s.ListRecords(ctx)
			if err != nil {
				return UpstreamAccountSyncResult{}, err
			}
			result.Records = records
			return result, nil
		} else if err != nil {
			logger.LegacyPrintf("service.upstream_account_sync", "Warning: load preview cache failed: %v", err)
		}
	}
	result, _, err := s.preview(ctx, false)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	s.storePreviewCache(ctx, result)
	records, err := s.ListRecords(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	return result, nil
}

func (s *UpstreamAccountSyncService) applyCachedProviderState(ctx context.Context, result UpstreamAccountSyncResult) UpstreamAccountSyncResult {
	source, ok := s.providerSource.(interface {
		ListProviders(context.Context) ([]UpstreamProviderConfig, error)
	})
	if !ok {
		return result
	}
	providers, err := source.ListProviders(ctx)
	if err != nil {
		logger.LegacyPrintf("service.upstream_account_sync", "Warning: refresh cached preview providers failed: %v", err)
		return result
	}
	current := make(map[string]UpstreamProviderConfig, len(providers))
	for _, provider := range providers {
		if provider.Slug == "" {
			continue
		}
		current[provider.Slug] = provider
	}
	if defaultProvider, err := s.providerSource.GetDefaultProvider(ctx); err == nil && defaultProvider.Slug != "" {
		current[defaultProvider.Slug] = defaultProvider
		result.DefaultProvider.Enabled = defaultProvider.Enabled
	} else if err != nil {
		logger.LegacyPrintf("service.upstream_account_sync", "Warning: refresh cached preview default provider failed: %v", err)
	}
	for index := range result.Providers {
		if provider, ok := current[result.Providers[index].Slug]; ok {
			result.Providers[index].Enabled = provider.Enabled
		}
	}
	return result
}

func (s *UpstreamAccountSyncService) Sync(ctx context.Context, req UpstreamAccountSyncRequest) (UpstreamAccountSyncResult, error) {
	triggerSource := normalizeUpstreamAccountSyncTriggerSource(req.TriggerSource)
	result, previewState, err := s.preview(ctx, false)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Summary.CreateCount = 0
	result.Summary.UpdateCount = 0
	result.Summary.RateViolationCount = 0
	result.Summary.UnboundGroupCount = 0

	now := time.Now().UTC()
	recordStats, recordOrder := newUpstreamAccountSyncRecordStats(result)
	ignoredRateGuardAccountIDs := previewState.rateGuardIgnoredAccountIDs
	selection, hasSelection := upstreamAccountSyncSelectedItems(req.SelectedItems)
	for index := range result.Items {
		item := &result.Items[index]
		selected, selectedOK := selection[upstreamAccountSyncSelectionKey(item.ProviderSlug, item.UpstreamKeyName)]
		if hasSelection && !selectedOK {
			continue
		}
		createMissing := req.CreateMissing
		updateExisting := req.UpdateExisting
		applyRateGuard := req.ApplyRateGuard
		if hasSelection {
			createMissing = selected.CreateMissing
			updateExisting = selected.UpdateExisting
			applyRateGuard = selected.ApplyRateGuard
		}
		provider := previewState.providerBySlug[item.ProviderSlug]
		if provider.Slug == "" {
			provider = result.DefaultProvider
		}
		switch item.Action {
		case UpstreamAccountSyncActionCreate:
			if !createMissing {
				continue
			}
			if strings.TrimSpace(item.UpstreamAPIKey) == "" {
				item.Action = UpstreamAccountSyncActionSkip
				item.SkipReason = "upstream provider did not return a usable api key"
				result.Summary.SkipCount++
				continue
			}
			groupIDs := []int64{}
			if item.LocalGroupID != nil {
				groupIDs = append(groupIDs, *item.LocalGroupID)
			}
			if len(groupIDs) == 0 {
				item.Action = UpstreamAccountSyncActionSkip
				item.SkipReason = "upstream group is not matched"
				result.Summary.SkipCount++
				continue
			}
			extra := upstreamAccountSyncExtra(provider, *item, now, nil)
			created, err := s.accountManager.CreateAccount(ctx, &CreateAccountInput{
				Name:                  item.LocalAccountName,
				Platform:              PlatformOpenAI,
				Type:                  AccountTypeAPIKey,
				Credentials:           upstreamAccountSyncCredentials(nil, item.UpstreamAPIKey, provider.BaseURL),
				Extra:                 extra,
				GroupIDs:              groupIDs,
				SkipDefaultGroupBind:  true,
				SkipMixedChannelCheck: true,
			})
			if err != nil {
				return s.finishSyncWithError(ctx, result, triggerSource, recordStats, recordOrder, item.ProviderSlug, err)
			}
			if created != nil {
				accountID := created.ID
				item.Execution = UpstreamAccountSyncExecutionResult{
					Executed:    true,
					Action:      UpstreamAccountSyncActionCreate,
					AccountID:   &accountID,
					AccountName: created.Name,
				}
			}
			result.Summary.CreateCount++
			recordStats[item.ProviderSlug].createdCount++
		case UpstreamAccountSyncActionUpdate:
			if !updateExisting || item.MatchedAccountID == nil {
				continue
			}
			account := previewState.accountByID[*item.MatchedAccountID]
			if upstreamAccountSyncItemMetadataOnly(*item) || !upstreamAccountSyncAccountCompatible(account) {
				if !upstreamAccountSyncMetadataNeedsUpdate(account, *item, provider) {
					continue
				}
				extra := upstreamAccountSyncExtra(provider, *item, now, account.Extra)
				updated, err := s.accountManager.UpdateAccount(ctx, account.ID, &UpdateAccountInput{
					Extra:                 extra,
					SkipMixedChannelCheck: true,
				})
				if err != nil {
					return s.finishSyncWithError(ctx, result, triggerSource, recordStats, recordOrder, item.ProviderSlug, err)
				}
				accountName := account.Name
				if updated != nil && strings.TrimSpace(updated.Name) != "" {
					accountName = updated.Name
				}
				accountID := account.ID
				item.Execution = UpstreamAccountSyncExecutionResult{
					Executed:    true,
					Action:      UpstreamAccountSyncActionUpdate,
					AccountID:   &accountID,
					AccountName: accountName,
				}
				item.Action = UpstreamAccountSyncActionUpdate
				result.Summary.UpdateCount++
				recordStats[item.ProviderSlug].updatedCount++
				continue
			}
			nextGroupIDs := upstreamAccountSyncExistingGroupIDs(account)
			if item.LocalGroupID != nil {
				nextGroupIDs = appendUniqueInt64(nextGroupIDs, *item.LocalGroupID)
			}
			rateGuardIgnored := upstreamAccountSyncInt64SetContains(ignoredRateGuardAccountIDs, account.ID)
			lowGroupIDs, lowGroupNames, remainingGroupIDs := upstreamAccountSyncLowRateGroups(account, item.UpstreamRateMultiplier)
			if applyRateGuard && !rateGuardIgnored && len(lowGroupIDs) > 0 {
				nextGroupIDs = remainingGroupIDs
				item.RateViolation = true
				item.UnboundGroupIDs = lowGroupIDs
				item.UnboundGroupNames = lowGroupNames
			} else if rateGuardIgnored {
				item.RateGuardIgnored = true
				item.RateViolation = false
				item.UnboundGroupIDs = nil
				item.UnboundGroupNames = nil
			}
			extra := upstreamAccountSyncExtra(provider, *item, now, account.Extra)
			updated, err := s.accountManager.UpdateAccount(ctx, account.ID, &UpdateAccountInput{
				Credentials:           upstreamAccountSyncCredentials(account.Credentials, "", provider.BaseURL),
				Extra:                 extra,
				GroupIDs:              &nextGroupIDs,
				SkipMixedChannelCheck: true,
			})
			if err != nil {
				return s.finishSyncWithError(ctx, result, triggerSource, recordStats, recordOrder, item.ProviderSlug, err)
			}
			accountName := account.Name
			if updated != nil && strings.TrimSpace(updated.Name) != "" {
				accountName = updated.Name
			}
			accountID := account.ID
			executedUnboundGroupIDs := []int64{}
			executedUnboundGroupNames := []string{}
			if applyRateGuard && !rateGuardIgnored && len(lowGroupIDs) > 0 {
				executedUnboundGroupIDs = append([]int64(nil), lowGroupIDs...)
				executedUnboundGroupNames = append([]string(nil), lowGroupNames...)
			}
			item.Execution = UpstreamAccountSyncExecutionResult{
				Executed:          true,
				Action:            UpstreamAccountSyncActionUpdate,
				AccountID:         &accountID,
				AccountName:       accountName,
				UnboundGroupIDs:   executedUnboundGroupIDs,
				UnboundGroupNames: executedUnboundGroupNames,
			}
			if applyRateGuard && !rateGuardIgnored && len(lowGroupIDs) > 0 {
				detail := buildUpstreamAccountSyncUnbindDetail(provider, *item, account, lowGroupIDs, lowGroupNames, remainingGroupIDs, triggerSource)
				recordStats[item.ProviderSlug].unbindDetails = append(recordStats[item.ProviderSlug].unbindDetails, detail)
				logUpstreamAccountSyncUnbindAudit(detail)
				result.Summary.RateViolationCount++
				result.Summary.UnboundGroupCount += len(lowGroupIDs)
				recordStats[item.ProviderSlug].rateViolationCount++
				recordStats[item.ProviderSlug].unboundGroupCount += len(lowGroupIDs)
			}
			item.Action = UpstreamAccountSyncActionUpdate
			result.Summary.UpdateCount++
			recordStats[item.ProviderSlug].updatedCount++
		}
	}

	recordsToPrepend := upstreamAccountSyncRecordsFromStats(result, recordStats, recordOrder, now, triggerSource)
	records, err := s.prependRecords(ctx, recordsToPrepend)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	s.refreshPreviewCacheAsync()
	return result, nil
}

func (s *UpstreamAccountSyncService) RefreshPreviewCache(ctx context.Context) (UpstreamAccountSyncResult, error) {
	result, _, err := s.preview(ctx, false)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	records, err := s.ListRecords(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	s.storePreviewCache(ctx, result)
	return result, nil
}

func (s *UpstreamAccountSyncService) storePreviewCache(ctx context.Context, result UpstreamAccountSyncResult) {
	if s == nil || s.previewCache == nil {
		return
	}
	if err := s.previewCache.Set(ctx, result); err != nil {
		logger.LegacyPrintf("service.upstream_account_sync", "Warning: store preview cache failed: %v", err)
	}
}

func (s *UpstreamAccountSyncService) refreshPreviewCacheAsync() {
	if s == nil || s.previewCache == nil {
		return
	}
	s.previewRefreshMu.Lock()
	if s.previewRefreshInFlight {
		s.previewRefreshMu.Unlock()
		return
	}
	s.previewRefreshInFlight = true
	s.previewRefreshMu.Unlock()

	go func() {
		defer func() {
			s.previewRefreshMu.Lock()
			s.previewRefreshInFlight = false
			s.previewRefreshMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), upstreamAccountSyncPreviewRefreshTimeout)
		defer cancel()
		if _, err := s.RefreshPreviewCache(ctx); err != nil {
			logger.LegacyPrintf("service.upstream_account_sync", "Warning: refresh preview cache failed: %v", err)
		}
	}()
}

func (s *UpstreamAccountSyncService) GetRateGuardConfig(ctx context.Context) (UpstreamAccountRateGuardConfig, error) {
	if s == nil || s.settingRepo == nil {
		return defaultUpstreamAccountRateGuardConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamAccountRateGuardConfig)
	if err != nil {
		if err == ErrSettingNotFound {
			return defaultUpstreamAccountRateGuardConfig(), nil
		}
		return UpstreamAccountRateGuardConfig{}, fmt.Errorf("load upstream account rate guard config: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultUpstreamAccountRateGuardConfig(), nil
	}
	var config UpstreamAccountRateGuardConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return UpstreamAccountRateGuardConfig{}, infraerrors.InternalServer("UPSTREAM_ACCOUNT_RATE_GUARD_CONFIG_INVALID", "upstream account rate guard config is invalid")
	}
	return normalizeUpstreamAccountRateGuardConfig(config), nil
}

func (s *UpstreamAccountSyncService) UpdateRateGuardConfig(ctx context.Context, input UpstreamAccountRateGuardConfig) (UpstreamAccountRateGuardConfig, error) {
	config := normalizeUpstreamAccountRateGuardConfig(input)
	if input.IntervalSeconds > 0 && input.IntervalSeconds < MinUpstreamAccountRateGuardIntervalSeconds {
		return UpstreamAccountRateGuardConfig{}, infraerrors.BadRequest("UPSTREAM_ACCOUNT_RATE_GUARD_INTERVAL_INVALID", fmt.Sprintf("interval_seconds must be at least %d", MinUpstreamAccountRateGuardIntervalSeconds))
	}
	now := time.Now().UTC()
	config.UpdatedAt = &now
	saved, err := s.saveRateGuardConfig(ctx, config)
	if err != nil {
		return UpstreamAccountRateGuardConfig{}, err
	}
	s.refreshPreviewCacheAsync()
	return saved, nil
}

func (s *UpstreamAccountSyncService) RunScheduledRateGuard(ctx context.Context) (UpstreamAccountRateGuardConfig, error) {
	return s.RunRateGuard(ctx, UpstreamAccountSyncTriggerScheduledRateGuard)
}

func (s *UpstreamAccountSyncService) RunRateGuard(ctx context.Context, triggerSource string) (UpstreamAccountRateGuardConfig, error) {
	config, err := s.GetRateGuardConfig(ctx)
	if err != nil {
		return UpstreamAccountRateGuardConfig{}, err
	}
	now := time.Now().UTC()
	config.LastRunAt = &now
	_, runErr := s.runMatchedAccountRateGuard(ctx, normalizeUpstreamAccountRateGuardTriggerSource(triggerSource), config.IgnoredAccountIDs)
	if runErr != nil {
		config.LastRunStatus = "failed"
		config.LastRunMessage = runErr.Error()
	} else {
		config.LastRunStatus = "success"
		config.LastRunMessage = ""
	}
	config.UpdatedAt = &now
	saved, saveErr := s.saveRateGuardConfig(ctx, config)
	if saveErr != nil {
		return UpstreamAccountRateGuardConfig{}, saveErr
	}
	return saved, runErr
}

func (s *UpstreamAccountSyncService) runMatchedAccountRateGuard(ctx context.Context, triggerSource string, ignoredAccountIDs []int64) (UpstreamAccountSyncResult, error) {
	result, previewState, err := s.preview(ctx, false)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Summary.CreateCount = 0
	result.Summary.UpdateCount = 0
	result.Summary.RateViolationCount = 0
	result.Summary.UnboundGroupCount = 0

	now := time.Now().UTC()
	recordStats, recordOrder := newUpstreamAccountSyncRecordStats(result)
	ignoredRateGuardAccountIDs := upstreamAccountSyncInt64Set(ignoredAccountIDs)
	if err := s.disableAccountsForDisabledProviders(ctx, previewState.accountByID, result.Items, ignoredRateGuardAccountIDs); err != nil {
		return s.finishSyncWithError(ctx, result, triggerSource, recordStats, recordOrder, "", err)
	}
	for index := range result.Items {
		item := &result.Items[index]
		if item.MatchedAccountID == nil {
			continue
		}
		if upstreamAccountSyncInt64SetContains(ignoredRateGuardAccountIDs, *item.MatchedAccountID) {
			item.RateGuardIgnored = true
			item.RateViolation = false
			item.UnboundGroupIDs = nil
			item.UnboundGroupNames = nil
			continue
		}
		account := previewState.accountByID[*item.MatchedAccountID]
		if !upstreamAccountSyncRateGuardCompatible(account) {
			continue
		}
		lowGroupIDs, lowGroupNames, remainingGroupIDs := upstreamAccountSyncLowRateGroups(account, item.UpstreamRateMultiplier)
		if len(lowGroupIDs) == 0 {
			continue
		}

		_, err := s.accountManager.UpdateAccount(ctx, account.ID, &UpdateAccountInput{
			GroupIDs:              &remainingGroupIDs,
			SkipMixedChannelCheck: true,
		})
		if err != nil {
			return s.finishSyncWithError(ctx, result, triggerSource, recordStats, recordOrder, item.ProviderSlug, err)
		}

		provider := previewState.providerBySlug[item.ProviderSlug]
		if provider.Slug == "" {
			provider = result.DefaultProvider
		}
		item.RateViolation = true
		item.UnboundGroupIDs = lowGroupIDs
		item.UnboundGroupNames = lowGroupNames
		detail := buildUpstreamAccountSyncUnbindDetail(provider, *item, account, lowGroupIDs, lowGroupNames, remainingGroupIDs, triggerSource)
		recordStats[item.ProviderSlug].updatedCount++
		recordStats[item.ProviderSlug].rateViolationCount++
		recordStats[item.ProviderSlug].unboundGroupCount += len(lowGroupIDs)
		recordStats[item.ProviderSlug].unbindDetails = append(recordStats[item.ProviderSlug].unbindDetails, detail)
		logUpstreamAccountSyncUnbindAudit(detail)
		result.Summary.UpdateCount++
		result.Summary.RateViolationCount++
		result.Summary.UnboundGroupCount += len(lowGroupIDs)
	}

	recordsToPrepend := upstreamAccountSyncRecordsFromStats(result, recordStats, recordOrder, now, triggerSource)
	records, err := s.prependRecords(ctx, recordsToPrepend)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	s.refreshPreviewCacheAsync()
	return result, nil
}

func (s *UpstreamAccountSyncService) disableAccountsForDisabledProviders(ctx context.Context, accounts map[int64]Account, items []UpstreamAccountSyncItem, ignoredAccountIDs map[int64]struct{}) error {
	if len(accounts) == 0 || s == nil || s.accountManager == nil {
		return nil
	}
	disabledProviders := s.disabledAccountSyncProviders(ctx)
	if len(disabledProviders) == 0 {
		return nil
	}
	disabledProviderList := make([]UpstreamProviderConfig, 0, len(disabledProviders))
	for _, provider := range disabledProviders {
		disabledProviderList = append(disabledProviderList, provider)
	}
	accountIDsToDisable := map[int64]struct{}{}
	for _, account := range accounts {
		if upstreamAccountSyncInt64SetContains(ignoredAccountIDs, account.ID) {
			continue
		}
		if !account.Schedulable {
			continue
		}
		providerSlug := strings.TrimSpace(account.GetExtraString("upstream_provider_slug"))
		if providerSlug == "" {
			if provider, _, ok := upstreamAccountSyncInferProviderKeyFromAccountName(account, disabledProviderList); ok && provider.Slug != "" {
				accountIDsToDisable[account.ID] = struct{}{}
			}
			continue
		}
		if _, disabled := disabledProviders[providerSlug]; !disabled {
			continue
		}
		accountIDsToDisable[account.ID] = struct{}{}
	}
	for _, item := range items {
		if item.MatchedAccountID == nil {
			continue
		}
		if _, disabled := disabledProviders[item.ProviderSlug]; !disabled {
			continue
		}
		accountID := *item.MatchedAccountID
		if upstreamAccountSyncInt64SetContains(ignoredAccountIDs, accountID) {
			continue
		}
		account, ok := accounts[accountID]
		if !ok || !account.Schedulable {
			continue
		}
		accountIDsToDisable[accountID] = struct{}{}
	}
	for accountID := range accountIDsToDisable {
		if _, err := s.accountManager.SetAccountSchedulable(ctx, accountID, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *UpstreamAccountSyncService) disabledAccountSyncProviders(ctx context.Context) map[string]UpstreamProviderConfig {
	if s == nil {
		return nil
	}
	source, ok := s.providerSource.(interface {
		ListProviders(context.Context) ([]UpstreamProviderConfig, error)
	})
	if !ok {
		return nil
	}
	providers, err := source.ListProviders(ctx)
	if err != nil {
		logger.LegacyPrintf("service.upstream_account_sync", "Warning: list disabled upstream providers failed: %v", err)
		return nil
	}
	disabled := map[string]UpstreamProviderConfig{}
	for _, provider := range providers {
		if provider.Slug == "" || provider.Enabled {
			continue
		}
		disabled[provider.Slug] = provider
	}
	if len(disabled) == 0 {
		return nil
	}
	return disabled
}

func (s *UpstreamAccountSyncService) ListRecords(ctx context.Context) ([]UpstreamAccountSyncRecord, error) {
	if s == nil || s.settingRepo == nil {
		return []UpstreamAccountSyncRecord{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamAccountSyncRecords)
	if err != nil {
		if err == ErrSettingNotFound {
			return []UpstreamAccountSyncRecord{}, nil
		}
		return nil, fmt.Errorf("load upstream account sync records: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []UpstreamAccountSyncRecord{}, nil
	}
	var records []UpstreamAccountSyncRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, infraerrors.InternalServer("UPSTREAM_ACCOUNT_SYNC_RECORDS_INVALID", "upstream account sync records are invalid")
	}
	return limitUpstreamAccountSyncRecords(records), nil
}

func (s *UpstreamAccountSyncService) MarkRecordHandled(ctx context.Context, key string) ([]UpstreamAccountSyncRecord, error) {
	records, err := s.ListRecords(ctx)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_ACCOUNT_SYNC_RECORD_KEY_REQUIRED", "upstream account sync record key is required")
	}
	key = normalizeUpstreamAccountSyncRecordDetailKey(key)

	found := false
	for recordIndex := range records {
		for detailIndex := range records[recordIndex].UnbindDetails {
			if upstreamAccountSyncRecordDetailKey(records[recordIndex], records[recordIndex].UnbindDetails[detailIndex]) != key {
				continue
			}
			records[recordIndex].UnbindDetails[detailIndex].Handled = true
			found = true
			break
		}
		if found {
			break
		}
	}
	if !found {
		return nil, infraerrors.NotFound("UPSTREAM_ACCOUNT_SYNC_RECORD_NOT_FOUND", "upstream account sync record was not found")
	}
	if s == nil || s.settingRepo == nil {
		return records, nil
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream account sync records: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamAccountSyncRecords, string(raw)); err != nil {
		return nil, fmt.Errorf("save upstream account sync records: %w", err)
	}
	return records, nil
}

type upstreamAccountSyncPreviewState struct {
	accountByID                map[int64]Account
	providerBySlug             map[string]UpstreamProviderConfig
	rateGuardIgnoredAccountIDs map[int64]struct{}
}

func (s *UpstreamAccountSyncService) preview(ctx context.Context, useProviderKeysCache bool) (UpstreamAccountSyncResult, upstreamAccountSyncPreviewState, error) {
	if s == nil || s.providerSource == nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, infraerrors.InternalServer("UPSTREAM_ACCOUNT_SYNC_PROVIDER_SOURCE_UNAVAILABLE", "upstream account sync provider source unavailable")
	}
	if s.groupRepo == nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, infraerrors.InternalServer("UPSTREAM_ACCOUNT_SYNC_GROUP_REPO_UNAVAILABLE", "upstream account sync group repository unavailable")
	}
	if s.accountManager == nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, infraerrors.InternalServer("UPSTREAM_ACCOUNT_SYNC_ACCOUNT_MANAGER_UNAVAILABLE", "upstream account sync account manager unavailable")
	}
	defaultProvider, err := s.providerSource.GetDefaultProvider(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
	}
	syncProviders, err := s.listAccountSyncProviders(ctx, defaultProvider)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
	}
	localGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, fmt.Errorf("list local groups: %w", err)
	}
	mappings, err := s.loadGroupMappings(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
	}
	rateGuardConfig, err := s.GetRateGuardConfig(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
	}
	rateGuardIgnoredAccountIDs := upstreamAccountSyncInt64Set(rateGuardConfig.IgnoredAccountIDs)
	accounts, err := s.loadAccounts(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
	}
	accounts = hydrateUpstreamAccountSyncAccountGroups(accounts, localGroups)
	accounts, err = refreshUpstreamAccountSyncAccountGroups(ctx, s.groupRepo, accounts)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
	}

	accountsByName := upstreamAccountSyncAccountsByName(accounts)
	defaultAccountsByName := upstreamAccountSyncAccountsByDefaultName(accounts)
	accountsByProviderKey := upstreamAccountSyncAccountsByProviderKey(accounts)
	accountByID := make(map[int64]Account, len(accounts))
	for _, account := range accounts {
		accountByID[account.ID] = account
	}
	providerBySlug := make(map[string]UpstreamProviderConfig, len(syncProviders))
	redactedProviders := make([]UpstreamProviderConfig, 0, len(syncProviders))
	for _, provider := range syncProviders {
		providerBySlug[provider.Slug] = provider
		redactedProviders = append(redactedProviders, redactUpstreamProvider(provider))
	}

	result := UpstreamAccountSyncResult{
		DefaultProvider: redactUpstreamProvider(defaultProvider),
		Providers:       redactedProviders,
		Items:           []UpstreamAccountSyncItem{},
		Warnings:        []string{},
		Records:         []UpstreamAccountSyncRecord{},
	}
	accountItemIDs := map[int64]struct{}{}
	providerKeysBySlug := s.fetchProviderKeysForAccountSync(ctx, syncProviders, useProviderKeysCache)
	for _, provider := range syncProviders {
		groupResolver := newUpstreamAccountSyncGroupResolver(provider, localGroups, mappings)
		keyResult := providerKeysBySlug[provider.Slug]
		if keyResult.err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", provider.Name, keyResult.err.Error()))
			fallbackItems := upstreamAccountSyncLocalSnapshotItemsForProvider(provider, accounts, keyResult.err, rateGuardIgnoredAccountIDs)
			result.Summary.UpstreamKeyCount += len(fallbackItems)
			result.Summary.MatchedAccountCount += len(fallbackItems)
			upstreamAccountSyncMarkItemAccountIDs(accountItemIDs, fallbackItems)
			result.Items = append(result.Items, fallbackItems...)
			continue
		}
		for _, warning := range keyResult.warnings {
			if strings.TrimSpace(warning) == "" {
				continue
			}
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", provider.Name, warning))
		}
		for _, key := range keyResult.keys {
			if key.ProviderSlug != "" && key.ProviderSlug != provider.Slug {
				continue
			}
			upstreamRateMultiplier := effectiveUpstreamAccountRateMultiplier(key.RateMultiplier, provider.AccountRateMultiplierScale)
			result.Summary.UpstreamKeyCount++
			item := UpstreamAccountSyncItem{
				ProviderSlug:           provider.Slug,
				ProviderName:           provider.Name,
				ProviderBaseURL:        provider.BaseURL,
				UpstreamKeyName:        strings.TrimSpace(key.KeyName),
				UpstreamAPIKey:         strings.TrimSpace(key.APIKey),
				UpstreamBaseURL:        provider.BaseURL,
				LocalAccountName:       upstreamAccountSyncLocalAccountName(provider, key.KeyName),
				UpstreamGroupName:      strings.TrimSpace(key.GroupName),
				UpstreamRateMultiplier: upstreamRateMultiplier,
			}
			if item.LocalAccountName == "" {
				item.Action = UpstreamAccountSyncActionSkip
				item.SkipReason = "upstream key name is empty"
				result.Summary.SkipCount++
				result.Items = append(result.Items, item)
				continue
			}

			matches := accountsByProviderKey[upstreamAccountSyncProviderKeyMatchKey(provider.Slug, key.KeyName)]
			if len(matches) == 0 {
				matches = accountsByName[normalizeUpstreamGroupMatchName(item.LocalAccountName)]
				if provider.IsDefault {
					matches = defaultAccountsByName[normalizeDefaultUpstreamAccountMatchName(item.LocalAccountName)]
				}
			}
			if len(matches) > 1 {
				upstreamAccountSyncMarkAccounts(accountItemIDs, matches)
				item.Action = UpstreamAccountSyncActionConflict
				item.ConflictAccountIDs = accountIDs(matches)
				item.ConflictAccounts = upstreamAccountSyncConflictAccounts(matches, item.UpstreamRateMultiplier)
				result.Summary.ConflictCount++
				result.Items = append(result.Items, item)
				continue
			}

			if len(matches) == 1 {
				account := matches[0]
				upstreamAccountSyncMarkAccounts(accountItemIDs, matches)
				accountID := account.ID
				item.MatchedAccountID = &accountID
				item.MatchedAccountName = account.Name
				item.RateGuardIgnored = upstreamAccountSyncInt64SetContains(rateGuardIgnoredAccountIDs, account.ID)
				item.BoundGroups = upstreamAccountSyncBoundGroups(account, item.UpstreamRateMultiplier)
				result.Summary.MatchedAccountCount++
				if !upstreamAccountSyncAccountCompatible(account) {
					if upstreamAccountSyncMetadataNeedsUpdate(account, item, provider) {
						item.ChangeDetails = upstreamAccountSyncMetadataChangeDetails()
						item.Action = UpstreamAccountSyncActionUpdate
						result.Summary.UpdateCount++
					} else {
						item.Action = UpstreamAccountSyncActionNoop
					}
					result.Items = append(result.Items, item)
					continue
				}
			}

			if group, ok := groupResolver.resolve(key.GroupName); ok {
				groupID := group.ID
				rate := group.RateMultiplier
				item.LocalGroupID = &groupID
				item.LocalGroupName = group.Name
				item.LocalRateMultiplier = &rate
			} else {
				if len(matches) == 1 && upstreamAccountSyncMetadataNeedsUpdate(matches[0], item, provider) {
					item.ChangeDetails = upstreamAccountSyncMetadataChangeDetails()
					item.Action = UpstreamAccountSyncActionUpdate
					result.Summary.UpdateCount++
					result.Items = append(result.Items, item)
					continue
				}
				item.Action = UpstreamAccountSyncActionSkip
				item.SkipReason = "upstream group is not matched"
				result.Summary.SkipCount++
				result.Items = append(result.Items, item)
				continue
			}

			if len(matches) == 0 {
				item.Action = UpstreamAccountSyncActionCreate
				result.Summary.CreateCount++
				result.Items = append(result.Items, item)
				continue
			}

			account := matches[0]
			lowGroupIDs, lowGroupNames, _ := upstreamAccountSyncLowRateGroups(account, item.UpstreamRateMultiplier)
			if len(lowGroupIDs) > 0 && !item.RateGuardIgnored {
				item.RateViolation = true
				item.UnboundGroupIDs = lowGroupIDs
				item.UnboundGroupNames = lowGroupNames
				result.Summary.RateViolationCount++
				result.Summary.UnboundGroupCount += len(lowGroupIDs)
			}
			item.ChangeDetails = upstreamAccountSyncChangeDetails(account, item, provider)
			if upstreamAccountSyncNeedsUpdate(account, item, provider) || item.RateViolation {
				item.Action = UpstreamAccountSyncActionUpdate
				result.Summary.UpdateCount++
			} else {
				item.Action = UpstreamAccountSyncActionNoop
			}
			result.Items = append(result.Items, item)
		}
	}
	upstreamAccountSyncAppendInferredMetadataItems(&result, accounts, syncProviders, accountItemIDs, rateGuardIgnoredAccountIDs)
	return result, upstreamAccountSyncPreviewState{accountByID: accountByID, providerBySlug: providerBySlug, rateGuardIgnoredAccountIDs: rateGuardIgnoredAccountIDs}, nil
}

func upstreamAccountSyncLocalSnapshotItemsForProvider(provider UpstreamProviderConfig, accounts []Account, fetchErr error, ignoredRateGuardAccountIDs map[int64]struct{}) []UpstreamAccountSyncItem {
	if provider.Slug == "" || len(accounts) == 0 {
		return nil
	}
	fetchError := ""
	if fetchErr != nil {
		fetchError = fetchErr.Error()
	}
	providerName := strings.TrimSpace(provider.Name)
	if providerName == "" {
		providerName = provider.Slug
	}
	providerBaseURL := strings.TrimSpace(provider.BaseURL)

	items := make([]UpstreamAccountSyncItem, 0)
	for _, account := range accounts {
		if strings.TrimSpace(account.GetExtraString("upstream_provider_slug")) != provider.Slug {
			continue
		}
		keyName := strings.TrimSpace(account.GetExtraString("upstream_key_name"))
		if keyName == "" {
			continue
		}
		accountID := account.ID
		upstreamRate := 0.0
		if account.Extra != nil {
			upstreamRate = parseExtraFloat64(account.Extra["upstream_rate_multiplier"])
		}
		baseURL := providerBaseURL
		if baseURL == "" {
			baseURL = strings.TrimSpace(account.GetCredential("base_url"))
		}

		items = append(items, UpstreamAccountSyncItem{
			Action:                 UpstreamAccountSyncActionNoop,
			ProviderSlug:           provider.Slug,
			ProviderName:           providerName,
			ProviderBaseURL:        baseURL,
			UpstreamKeyName:        keyName,
			UpstreamBaseURL:        baseURL,
			ProviderFetchError:     fetchError,
			LocalAccountName:       account.Name,
			MatchedAccountID:       &accountID,
			MatchedAccountName:     account.Name,
			RateGuardIgnored:       upstreamAccountSyncInt64SetContains(ignoredRateGuardAccountIDs, account.ID),
			UpstreamGroupName:      strings.TrimSpace(account.GetExtraString("upstream_group_name")),
			UpstreamRateMultiplier: upstreamRate,
			BoundGroups:            upstreamAccountSyncBoundGroups(account, upstreamRate),
			SkipReason:             "upstream provider keys unavailable; showing local account snapshot",
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpstreamKeyName == items[j].UpstreamKeyName {
			return items[i].MatchedAccountName < items[j].MatchedAccountName
		}
		return items[i].UpstreamKeyName < items[j].UpstreamKeyName
	})
	return items
}

func upstreamAccountSyncAppendInferredMetadataItems(result *UpstreamAccountSyncResult, accounts []Account, providers []UpstreamProviderConfig, accountItemIDs map[int64]struct{}, ignoredRateGuardAccountIDs map[int64]struct{}) {
	if result == nil || len(accounts) == 0 || len(providers) == 0 {
		return
	}
	if accountItemIDs == nil {
		accountItemIDs = map[int64]struct{}{}
	}
	for _, account := range accounts {
		if _, exists := accountItemIDs[account.ID]; exists {
			continue
		}
		if !upstreamAccountSyncMetadataInferenceEligible(account) {
			continue
		}
		provider, keyName, ok := upstreamAccountSyncInferProviderKeyFromAccountName(account, providers)
		if !ok {
			continue
		}
		item := upstreamAccountSyncInferredMetadataItem(provider, account, keyName)
		item.RateGuardIgnored = upstreamAccountSyncInt64SetContains(ignoredRateGuardAccountIDs, account.ID)
		if !upstreamAccountSyncMetadataNeedsUpdate(account, item, provider) {
			continue
		}
		result.Items = append(result.Items, item)
		result.Summary.MatchedAccountCount++
		result.Summary.UpdateCount++
		accountItemIDs[account.ID] = struct{}{}
	}
}

func upstreamAccountSyncMetadataInferenceEligible(account Account) bool {
	if account.ID <= 0 || strings.TrimSpace(account.Name) == "" {
		return false
	}
	if strings.TrimSpace(account.Status) == StatusDisabled {
		return false
	}
	return account.Type == AccountTypeAPIKey
}

func upstreamAccountSyncInferProviderKeyFromAccountName(account Account, providers []UpstreamProviderConfig) (UpstreamProviderConfig, string, bool) {
	accountName := strings.TrimSpace(account.Name)
	if accountName == "" {
		return UpstreamProviderConfig{}, "", false
	}

	candidates := upstreamAccountSyncPrefixCandidates(providers)
	for _, candidate := range candidates {
		if keyName, ok := upstreamAccountSyncTrimLocalAccountPrefix(accountName, candidate.prefix); ok {
			keyName = strings.TrimSpace(keyName)
			if keyName == "" {
				continue
			}
			return candidate.provider, keyName, true
		}
	}

	defaultProvider := upstreamAccountSyncDefaultProvider(providers)
	if defaultProvider.Slug == "" {
		return UpstreamProviderConfig{}, "", false
	}
	if !upstreamAccountSyncDefaultInferenceBaseURLMatches(account, defaultProvider, providers) {
		return UpstreamProviderConfig{}, "", false
	}
	return defaultProvider, accountName, true
}

type upstreamAccountSyncPrefixCandidate struct {
	provider UpstreamProviderConfig
	prefix   string
}

func upstreamAccountSyncPrefixCandidates(providers []UpstreamProviderConfig) []upstreamAccountSyncPrefixCandidate {
	candidates := make([]upstreamAccountSyncPrefixCandidate, 0, len(providers))
	seen := map[string]struct{}{}
	for _, provider := range providers {
		if provider.Slug == "" || provider.IsDefault {
			continue
		}
		prefix := upstreamAccountSyncEffectiveAccountNamePrefix(provider)
		if prefix == "" {
			continue
		}
		key := strings.ToLower(prefix)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		candidates = append(candidates, upstreamAccountSyncPrefixCandidate{
			provider: provider,
			prefix:   prefix,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return len([]rune(candidates[i].prefix)) > len([]rune(candidates[j].prefix))
	})
	return candidates
}

func upstreamAccountSyncEffectiveAccountNamePrefix(provider UpstreamProviderConfig) string {
	prefix := strings.TrimSpace(provider.AccountNamePrefix)
	if prefix == "" {
		return ""
	}
	if strings.HasSuffix(prefix, "-") {
		return prefix
	}
	return prefix + "-"
}

func upstreamAccountSyncTrimLocalAccountPrefix(accountName, prefix string) (string, bool) {
	accountName = strings.TrimSpace(accountName)
	prefix = strings.TrimSpace(prefix)
	if accountName == "" || prefix == "" {
		return "", false
	}
	if strings.HasPrefix(accountName, prefix) {
		return accountName[len(prefix):], true
	}
	accountRunes := []rune(accountName)
	prefixRunes := []rune(prefix)
	if len(accountRunes) < len(prefixRunes) {
		return "", false
	}
	head := string(accountRunes[:len(prefixRunes)])
	if strings.EqualFold(head, prefix) {
		return string(accountRunes[len(prefixRunes):]), true
	}
	return "", false
}

func upstreamAccountSyncDefaultProvider(providers []UpstreamProviderConfig) UpstreamProviderConfig {
	for _, provider := range providers {
		if provider.Slug != "" && provider.IsDefault {
			return provider
		}
	}
	if len(providers) > 0 && providers[0].Slug != "" {
		return providers[0]
	}
	return UpstreamProviderConfig{}
}

func upstreamAccountSyncDefaultInferenceBaseURLMatches(account Account, defaultProvider UpstreamProviderConfig, providers []UpstreamProviderConfig) bool {
	defaultBaseURL := normalizeUpstreamAccountSyncBaseURL(defaultProvider.BaseURL)
	accountBaseURL := normalizeUpstreamAccountSyncBaseURL(account.GetCredential("base_url"))
	if defaultBaseURL == "" {
		return false
	}
	if accountBaseURL == "" {
		return false
	}
	if accountBaseURL == defaultBaseURL {
		return true
	}
	for _, provider := range providers {
		if provider.Slug == "" || provider.IsDefault {
			continue
		}
		if accountBaseURL != "" && accountBaseURL == normalizeUpstreamAccountSyncBaseURL(provider.BaseURL) {
			return false
		}
	}
	return false
}

func upstreamAccountSyncInferredMetadataItem(provider UpstreamProviderConfig, account Account, keyName string) UpstreamAccountSyncItem {
	providerName := strings.TrimSpace(provider.Name)
	if providerName == "" {
		providerName = provider.Slug
	}
	baseURL := strings.TrimSpace(provider.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(account.GetCredential("base_url"))
	}
	accountID := account.ID
	upstreamRate := 0.0
	if account.Extra != nil {
		upstreamRate = parseExtraFloat64(account.Extra["upstream_rate_multiplier"])
	}
	return UpstreamAccountSyncItem{
		Action:                 UpstreamAccountSyncActionUpdate,
		ProviderSlug:           provider.Slug,
		ProviderName:           providerName,
		ProviderBaseURL:        baseURL,
		UpstreamKeyName:        strings.TrimSpace(keyName),
		UpstreamBaseURL:        baseURL,
		LocalAccountName:       account.Name,
		MatchedAccountID:       &accountID,
		MatchedAccountName:     account.Name,
		UpstreamGroupName:      strings.TrimSpace(account.GetExtraString("upstream_group_name")),
		UpstreamRateMultiplier: upstreamRate,
		BoundGroups:            upstreamAccountSyncBoundGroups(account, upstreamRate),
		ChangeDetails:          upstreamAccountSyncMetadataChangeDetails(),
	}
}

func effectiveUpstreamAccountRateMultiplier(rawRate, scale float64) float64 {
	if scale == 0 {
		scale = 1
	}
	return rawRate * scale
}

func (s *UpstreamAccountSyncService) listAccountSyncProviders(ctx context.Context, defaultProvider UpstreamProviderConfig) ([]UpstreamProviderConfig, error) {
	out := make([]UpstreamProviderConfig, 0, 1)
	seen := map[string]struct{}{}
	if defaultProvider.Slug != "" {
		defaultProvider = s.hydrateStoredProvider(ctx, defaultProvider)
		defaultProvider.IsDefault = true
		out = append(out, defaultProvider)
		seen[defaultProvider.Slug] = struct{}{}
	}
	source, ok := s.providerSource.(interface {
		ListProviders(context.Context) ([]UpstreamProviderConfig, error)
	})
	if !ok {
		return out, nil
	}
	providers, err := source.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list upstream account sync providers: %w", err)
	}
	for _, provider := range providers {
		if provider.Slug == "" || provider.IsDefault {
			continue
		}
		provider = s.hydrateStoredProvider(ctx, provider)
		if provider.Slug == "" || provider.IsDefault {
			continue
		}
		if !provider.Enabled {
			continue
		}
		if _, exists := seen[provider.Slug]; exists {
			continue
		}
		out = append(out, provider)
		seen[provider.Slug] = struct{}{}
	}
	return out, nil
}

func (s *UpstreamAccountSyncService) hydrateStoredProvider(ctx context.Context, provider UpstreamProviderConfig) UpstreamProviderConfig {
	source, ok := s.providerSource.(upstreamAccountSyncStoredProviderSource)
	if !ok {
		return provider
	}
	stored, err := source.getStoredProvider(ctx, provider.Slug)
	if err != nil {
		return provider
	}
	return stored
}

func (s *UpstreamAccountSyncService) loadAccounts(ctx context.Context) ([]Account, error) {
	const pageSize = 1000
	all := []Account{}
	for page := 1; ; page++ {
		accounts, total, err := s.accountManager.ListAccounts(ctx, page, pageSize, "", "", "", "", 0, "", "name", "asc")
		if err != nil {
			return nil, fmt.Errorf("load local accounts for upstream account sync: %w", err)
		}
		all = append(all, accounts...)
		if len(all) >= int(total) || len(accounts) == 0 {
			return all, nil
		}
	}
}

func (s *UpstreamAccountSyncService) fetchProviderKeysForAccountSync(ctx context.Context, providers []UpstreamProviderConfig, useCache bool) map[string]upstreamAccountSyncProviderKeysResult {
	results := make(map[string]upstreamAccountSyncProviderKeysResult, len(providers))
	now := time.Now()
	missingProviders := make([]UpstreamProviderConfig, 0, len(providers))

	if useCache {
		s.keysCacheMu.Lock()
		for _, provider := range providers {
			if s.keysCache != nil {
				if entry, ok := s.keysCache[provider.Slug]; ok && now.Before(entry.expiresAt) {
					results[provider.Slug] = upstreamAccountSyncProviderKeysResult{
						keys:     append([]UpstreamProviderKey(nil), entry.keys...),
						warnings: append([]string(nil), entry.warnings...),
					}
					continue
				}
			}
			missingProviders = append(missingProviders, provider)
		}
		s.keysCacheMu.Unlock()
	} else {
		missingProviders = append(missingProviders, providers...)
	}

	type providerKeysFetchResult struct {
		slug     string
		keys     []UpstreamProviderKey
		warnings []string
		err      error
	}
	resultCh := make(chan providerKeysFetchResult, len(missingProviders))
	var wg sync.WaitGroup
	for _, provider := range missingProviders {
		provider := provider
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchCtx, cancel := context.WithTimeout(ctx, upstreamAccountSyncProviderKeysFetchTimeout)
			defer cancel()
			start := time.Now()
			keys, warnings, err := s.providerSource.FetchProviderKeys(fetchCtx, provider.Slug)
			elapsed := time.Since(start)
			if err != nil {
				logger.LegacyPrintf("service.upstream_account_sync", "Provider keys fetch failed: slug=%s duration=%s error=%v", provider.Slug, elapsed, err)
			} else if elapsed >= upstreamAccountSyncProviderKeysSlowLogDuration {
				logger.LegacyPrintf("service.upstream_account_sync", "Warning: provider keys fetch slow: slug=%s duration=%s key_count=%d", provider.Slug, elapsed, len(keys))
			}
			resultCh <- providerKeysFetchResult{
				slug:     provider.Slug,
				keys:     keys,
				warnings: warnings,
				err:      err,
			}
		}()
	}
	wg.Wait()
	close(resultCh)

	cacheEntries := map[string]upstreamAccountSyncProviderKeysCacheEntry{}
	for result := range resultCh {
		results[result.slug] = upstreamAccountSyncProviderKeysResult{
			keys:     result.keys,
			warnings: result.warnings,
			err:      result.err,
		}
		if useCache && result.err == nil {
			cacheEntries[result.slug] = upstreamAccountSyncProviderKeysCacheEntry{
				keys:      append([]UpstreamProviderKey(nil), result.keys...),
				warnings:  append([]string(nil), result.warnings...),
				expiresAt: now.Add(upstreamAccountSyncProviderKeysCacheTTL),
			}
		}
	}

	if len(cacheEntries) > 0 {
		s.keysCacheMu.Lock()
		if s.keysCache == nil {
			s.keysCache = map[string]upstreamAccountSyncProviderKeysCacheEntry{}
		}
		for slug, entry := range cacheEntries {
			s.keysCache[slug] = entry
		}
		s.keysCacheMu.Unlock()
	}

	return results
}

func (s *UpstreamAccountSyncService) loadGroupMappings(ctx context.Context) ([]UpstreamGroupMappingRecord, error) {
	if s == nil || s.settingRepo == nil {
		return []UpstreamGroupMappingRecord{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamGroupMappings)
	if err != nil {
		if err == ErrSettingNotFound {
			return []UpstreamGroupMappingRecord{}, nil
		}
		return nil, fmt.Errorf("load upstream group mappings: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []UpstreamGroupMappingRecord{}, nil
	}
	var records []UpstreamGroupMappingRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, infraerrors.InternalServer("UPSTREAM_GROUP_MAPPINGS_INVALID", "upstream group mappings are invalid")
	}
	return normalizeUpstreamGroupMappings(records), nil
}

func (s *UpstreamAccountSyncService) prependRecords(ctx context.Context, newRecords []UpstreamAccountSyncRecord) ([]UpstreamAccountSyncRecord, error) {
	existing, err := s.ListRecords(ctx)
	if err != nil {
		return nil, err
	}
	newRecords = meaningfulUpstreamAccountSyncRecords(newRecords)
	if len(newRecords) == 0 {
		return existing, nil
	}
	records := append([]UpstreamAccountSyncRecord{}, newRecords...)
	records = append(records, existing...)
	records = limitUpstreamAccountSyncRecords(records)
	if s == nil || s.settingRepo == nil {
		return records, nil
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream account sync records: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamAccountSyncRecords, string(raw)); err != nil {
		return nil, fmt.Errorf("save upstream account sync records: %w", err)
	}
	return records, nil
}

func meaningfulUpstreamAccountSyncRecords(records []UpstreamAccountSyncRecord) []UpstreamAccountSyncRecord {
	out := make([]UpstreamAccountSyncRecord, 0, len(records))
	for _, record := range records {
		if !upstreamAccountSyncRecordHasActivity(record) {
			continue
		}
		out = append(out, record)
	}
	return out
}

func upstreamAccountSyncRecordHasActivity(record UpstreamAccountSyncRecord) bool {
	return len(record.UnbindDetails) > 0
}

func (s *UpstreamAccountSyncService) finishSyncWithError(ctx context.Context, result UpstreamAccountSyncResult, triggerSource string, recordStats map[string]*upstreamAccountSyncRecordStats, recordOrder []string, failedProviderSlug string, runErr error) (UpstreamAccountSyncResult, error) {
	if stats := recordStats[failedProviderSlug]; stats != nil {
		stats.error = runErr.Error()
	}
	recordsToPrepend := upstreamAccountSyncRecordsFromStats(result, recordStats, recordOrder, time.Now().UTC(), triggerSource)
	records, err := s.prependRecords(ctx, recordsToPrepend)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	return result, runErr
}

func newUpstreamAccountSyncRecordStats(result UpstreamAccountSyncResult) (map[string]*upstreamAccountSyncRecordStats, []string) {
	statsByProvider := map[string]*upstreamAccountSyncRecordStats{}
	order := []string{}
	ensure := func(slug, name string) {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			slug = result.DefaultProvider.Slug
			name = result.DefaultProvider.Name
		}
		if _, exists := statsByProvider[slug]; exists {
			return
		}
		statsByProvider[slug] = &upstreamAccountSyncRecordStats{
			providerSlug: slug,
			providerName: strings.TrimSpace(name),
		}
		order = append(order, slug)
	}
	for _, item := range result.Items {
		ensure(item.ProviderSlug, item.ProviderName)
	}
	if len(order) == 0 {
		for _, provider := range result.Providers {
			ensure(provider.Slug, provider.Name)
		}
	}
	if len(order) == 0 {
		ensure(result.DefaultProvider.Slug, result.DefaultProvider.Name)
	}
	return statsByProvider, order
}

func upstreamAccountSyncRecordsFromStats(result UpstreamAccountSyncResult, statsByProvider map[string]*upstreamAccountSyncRecordStats, order []string, createdAt time.Time, triggerSource string) []UpstreamAccountSyncRecord {
	for _, item := range result.Items {
		stats := statsByProvider[item.ProviderSlug]
		if stats == nil {
			continue
		}
		switch item.Action {
		case UpstreamAccountSyncActionSkip:
			stats.skippedCount++
		case UpstreamAccountSyncActionConflict:
			stats.conflictCount++
		}
	}

	records := make([]UpstreamAccountSyncRecord, 0, len(order))
	for _, slug := range order {
		stats := statsByProvider[slug]
		if stats == nil {
			continue
		}
		records = append(records, UpstreamAccountSyncRecord{
			ProviderSlug:       stats.providerSlug,
			ProviderName:       stats.providerName,
			CreatedCount:       stats.createdCount,
			UpdatedCount:       stats.updatedCount,
			SkippedCount:       stats.skippedCount,
			ConflictCount:      stats.conflictCount,
			RateViolationCount: stats.rateViolationCount,
			UnboundGroupCount:  stats.unboundGroupCount,
			CreatedAt:          createdAt,
			TriggerSource:      normalizeUpstreamAccountSyncTriggerSource(triggerSource),
			Error:              stats.error,
			UnbindDetails:      stats.unbindDetails,
		})
	}
	return records
}

type upstreamAccountSyncGroupResolver struct {
	localByName         map[string]Group
	localByID           map[int64]Group
	mappedLocalGroupIDs map[string]int64
}

func newUpstreamAccountSyncGroupResolver(provider UpstreamProviderConfig, groups []Group, mappings []UpstreamGroupMappingRecord) upstreamAccountSyncGroupResolver {
	resolver := upstreamAccountSyncGroupResolver{
		localByName:         make(map[string]Group, len(groups)),
		localByID:           make(map[int64]Group, len(groups)),
		mappedLocalGroupIDs: make(map[string]int64, len(mappings)),
	}
	for _, group := range groups {
		resolver.localByID[group.ID] = group
		key := normalizeUpstreamGroupMatchName(group.Name)
		if key == "" {
			continue
		}
		if _, exists := resolver.localByName[key]; !exists {
			resolver.localByName[key] = group
		}
	}
	for _, mapping := range mappings {
		if mapping.ProviderSlug != provider.Slug || mapping.LocalGroupID <= 0 {
			continue
		}
		key := normalizeUpstreamGroupMatchName(mapping.UpstreamGroupKey)
		if key == "" {
			key = normalizeUpstreamGroupMatchName(mapping.UpstreamGroupName)
		}
		if key != "" {
			resolver.mappedLocalGroupIDs[key] = mapping.LocalGroupID
		}
	}
	return resolver
}

func (r upstreamAccountSyncGroupResolver) resolve(upstreamGroupName string) (Group, bool) {
	key := normalizeUpstreamGroupMatchName(upstreamGroupName)
	if key == "" {
		return Group{}, false
	}
	if mappedID, ok := r.mappedLocalGroupIDs[key]; ok {
		group, exists := r.localByID[mappedID]
		return group, exists
	}
	group, ok := r.localByName[key]
	return group, ok
}

func upstreamAccountSyncAccountsByName(accounts []Account) map[string][]Account {
	out := make(map[string][]Account, len(accounts))
	for _, account := range accounts {
		key := normalizeUpstreamGroupMatchName(account.Name)
		if key == "" {
			continue
		}
		out[key] = append(out[key], account)
	}
	return out
}

func upstreamAccountSyncAccountsByDefaultName(accounts []Account) map[string][]Account {
	out := make(map[string][]Account, len(accounts))
	for _, account := range accounts {
		key := normalizeDefaultUpstreamAccountMatchName(account.Name)
		if key == "" {
			continue
		}
		out[key] = append(out[key], account)
	}
	return out
}

func upstreamAccountSyncAccountsByProviderKey(accounts []Account) map[string][]Account {
	out := make(map[string][]Account, len(accounts))
	for _, account := range accounts {
		key := upstreamAccountSyncProviderKeyMatchKey(
			account.GetExtraString("upstream_provider_slug"),
			account.GetExtraString("upstream_key_name"),
		)
		if key == "" {
			continue
		}
		out[key] = append(out[key], account)
	}
	return out
}

func upstreamAccountSyncMarkItemAccountIDs(seen map[int64]struct{}, items []UpstreamAccountSyncItem) {
	if seen == nil {
		return
	}
	for _, item := range items {
		if item.MatchedAccountID == nil || *item.MatchedAccountID <= 0 {
			continue
		}
		seen[*item.MatchedAccountID] = struct{}{}
	}
}

func upstreamAccountSyncMarkAccounts(seen map[int64]struct{}, accounts []Account) {
	if seen == nil {
		return
	}
	for _, account := range accounts {
		if account.ID <= 0 {
			continue
		}
		seen[account.ID] = struct{}{}
	}
}

func upstreamAccountSyncProviderKeyMatchKey(providerSlug, keyName string) string {
	providerSlug = strings.TrimSpace(providerSlug)
	keyName = normalizeUpstreamGroupMatchName(keyName)
	if providerSlug == "" || keyName == "" {
		return ""
	}
	return providerSlug + "\x00" + keyName
}

func upstreamAccountSyncLocalAccountName(provider UpstreamProviderConfig, keyName string) string {
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		return ""
	}
	if provider.IsDefault {
		return keyName
	}
	return upstreamProviderKeyName(provider, keyName)
}

func normalizeDefaultUpstreamAccountMatchName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	b.Grow(len(normalized))
	for _, r := range normalized {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func hydrateUpstreamAccountSyncAccountGroups(accounts []Account, groups []Group) []Account {
	if len(accounts) == 0 || len(groups) == 0 {
		return accounts
	}
	groupByID := make(map[int64]*Group, len(groups))
	for i := range groups {
		if groups[i].ID <= 0 {
			continue
		}
		group := groups[i]
		groupByID[group.ID] = &group
	}
	if len(groupByID) == 0 {
		return accounts
	}

	out := make([]Account, len(accounts))
	copy(out, accounts)
	for i := range out {
		if len(out[i].GroupIDs) == 0 {
			continue
		}
		seen := make(map[int64]struct{}, len(out[i].Groups)+len(out[i].GroupIDs))
		merged := make([]*Group, 0, len(out[i].Groups)+len(out[i].GroupIDs))
		for _, group := range out[i].Groups {
			if group == nil || group.ID <= 0 {
				continue
			}
			if _, exists := seen[group.ID]; exists {
				continue
			}
			seen[group.ID] = struct{}{}
			merged = append(merged, group)
		}
		for _, groupID := range out[i].GroupIDs {
			if groupID <= 0 {
				continue
			}
			if _, exists := seen[groupID]; exists {
				continue
			}
			group, exists := groupByID[groupID]
			if !exists || group == nil {
				continue
			}
			seen[groupID] = struct{}{}
			merged = append(merged, group)
		}
		if len(merged) > 0 {
			sort.Slice(merged, func(a, b int) bool { return merged[a].ID < merged[b].ID })
			out[i].Groups = merged
		}
	}
	return out
}

func refreshUpstreamAccountSyncAccountGroups(ctx context.Context, groupRepo GroupRepository, accounts []Account) ([]Account, error) {
	if len(accounts) == 0 {
		return accounts, nil
	}
	loader, ok := groupRepo.(upstreamAccountSyncBoundGroupLoader)
	if !ok {
		return accounts, nil
	}

	accountIDs := make([]int64, 0, len(accounts))
	seen := make(map[int64]struct{}, len(accounts))
	for _, account := range accounts {
		if account.ID <= 0 {
			continue
		}
		if _, exists := seen[account.ID]; exists {
			continue
		}
		seen[account.ID] = struct{}{}
		accountIDs = append(accountIDs, account.ID)
	}
	if len(accountIDs) == 0 {
		return accounts, nil
	}

	groupsByAccount, err := loader.ListGroupsByAccountIDs(ctx, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("load account bound groups for upstream account sync: %w", err)
	}
	if groupsByAccount == nil {
		return accounts, nil
	}

	out := make([]Account, len(accounts))
	copy(out, accounts)
	for i := range out {
		groups := groupsByAccount[out[i].ID]
		out[i].Groups = groups
		out[i].GroupIDs = upstreamAccountSyncGroupIDs(groups)
	}
	return out, nil
}

func upstreamAccountSyncGroupIDs(groups []*Group) []int64 {
	out := make([]int64, 0, len(groups))
	for _, group := range groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		out = appendUniqueInt64(out, group.ID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func upstreamAccountSyncAccountCompatible(account Account) bool {
	return account.Platform == PlatformOpenAI && account.Type == AccountTypeAPIKey
}

func upstreamAccountSyncRateGuardCompatible(account Account) bool {
	return account.Type == AccountTypeAPIKey
}

func upstreamAccountSyncNeedsUpdate(account Account, item UpstreamAccountSyncItem, provider UpstreamProviderConfig) bool {
	nextAPIKey := strings.TrimSpace(item.UpstreamAPIKey)
	if nextAPIKey != "" && strings.TrimSpace(account.GetCredential("api_key")) != nextAPIKey {
		return true
	}
	if strings.TrimRight(strings.TrimSpace(account.GetCredential("base_url")), "/") != strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/") {
		return true
	}
	if upstreamAccountSyncMetadataNeedsUpdate(account, item, provider) {
		return true
	}
	if item.LocalGroupID != nil && !containsInt64(upstreamAccountSyncExistingGroupIDs(account), *item.LocalGroupID) {
		return true
	}
	return false
}

func upstreamAccountSyncMetadataNeedsUpdate(account Account, item UpstreamAccountSyncItem, provider UpstreamProviderConfig) bool {
	if strings.TrimSpace(account.GetExtraString("upstream_provider_slug")) != strings.TrimSpace(provider.Slug) {
		return true
	}
	if strings.TrimSpace(account.GetExtraString("upstream_key_name")) != strings.TrimSpace(item.UpstreamKeyName) {
		return true
	}
	return false
}

func upstreamAccountSyncMetadataChangeDetails() []UpstreamAccountSyncChangeDetail {
	return []UpstreamAccountSyncChangeDetail{{
		Kind:  "metadata",
		Field: "upstream",
		Label: "Upstream sync metadata",
	}}
}

func upstreamAccountSyncItemMetadataOnly(item UpstreamAccountSyncItem) bool {
	if item.LocalGroupID != nil || item.RateViolation || len(item.UnboundGroupIDs) > 0 {
		return false
	}
	if len(item.ChangeDetails) == 0 {
		return false
	}
	for _, detail := range item.ChangeDetails {
		if detail.Kind != "metadata" {
			return false
		}
	}
	return true
}

func upstreamAccountSyncSelectedItems(items []UpstreamAccountSyncSelectedItem) (map[string]UpstreamAccountSyncSelectedItem, bool) {
	if len(items) == 0 {
		return nil, false
	}
	out := make(map[string]UpstreamAccountSyncSelectedItem, len(items))
	for _, item := range items {
		key := upstreamAccountSyncSelectionKey(item.ProviderSlug, item.UpstreamKeyName)
		if key == "" {
			continue
		}
		out[key] = item
	}
	return out, len(out) > 0
}

func upstreamAccountSyncSelectionKey(providerSlug, upstreamKeyName string) string {
	providerSlug = strings.TrimSpace(providerSlug)
	upstreamKeyName = strings.TrimSpace(upstreamKeyName)
	if providerSlug == "" || upstreamKeyName == "" {
		return ""
	}
	return providerSlug + "\x00" + upstreamKeyName
}

func upstreamAccountSyncChangeDetails(account Account, item UpstreamAccountSyncItem, provider UpstreamProviderConfig) []UpstreamAccountSyncChangeDetail {
	details := []UpstreamAccountSyncChangeDetail{}
	currentAPIKey := strings.TrimSpace(account.GetCredential("api_key"))
	nextAPIKey := strings.TrimSpace(item.UpstreamAPIKey)
	if nextAPIKey != "" && currentAPIKey != nextAPIKey {
		details = append(details, UpstreamAccountSyncChangeDetail{
			Kind:   "credential",
			Field:  "api_key",
			Label:  "API key",
			Before: currentAPIKey,
			After:  nextAPIKey,
		})
	}

	currentBaseURL := normalizeUpstreamAccountSyncBaseURL(account.GetCredential("base_url"))
	nextBaseURL := normalizeUpstreamAccountSyncBaseURL(provider.BaseURL)
	if currentBaseURL != nextBaseURL {
		details = append(details, UpstreamAccountSyncChangeDetail{
			Kind:   "credential",
			Field:  "base_url",
			Label:  "Base URL",
			Before: currentBaseURL,
			After:  nextBaseURL,
		})
	}

	details = append(details, upstreamAccountSyncMetadataChangeDetails()...)

	existingGroupIDs := upstreamAccountSyncExistingGroupIDs(account)
	if item.LocalGroupID != nil && !containsInt64(existingGroupIDs, *item.LocalGroupID) {
		details = append(details, UpstreamAccountSyncChangeDetail{
			Kind:       "group_bind",
			Field:      "group_ids",
			Label:      "Bind local group",
			GroupIDs:   []int64{*item.LocalGroupID},
			GroupNames: []string{strings.TrimSpace(item.LocalGroupName)},
		})
	}

	lowGroupIDs, lowGroupNames, _ := upstreamAccountSyncLowRateGroups(account, item.UpstreamRateMultiplier)
	if len(lowGroupIDs) > 0 && !item.RateGuardIgnored {
		details = append(details, UpstreamAccountSyncChangeDetail{
			Kind:       "group_unbind",
			Field:      "group_ids",
			Label:      "Unbind low-rate groups",
			GroupIDs:   lowGroupIDs,
			GroupNames: lowGroupNames,
		})
	}
	return details
}

func normalizeUpstreamAccountSyncBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func upstreamAccountSyncCredentials(existing map[string]any, apiKey, baseURL string) map[string]any {
	out := copyAnyMap(existing)
	if out == nil {
		out = map[string]any{}
	}
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		out["api_key"] = apiKey
	}
	out["base_url"] = normalizeUpstreamAccountSyncBaseURL(baseURL)
	return out
}

func upstreamAccountSyncExtra(provider UpstreamProviderConfig, item UpstreamAccountSyncItem, syncedAt time.Time, existing map[string]any) map[string]any {
	out := copyAnyMap(existing)
	if out == nil {
		out = map[string]any{}
	}
	out["upstream_provider_slug"] = provider.Slug
	out["upstream_provider_name"] = provider.Name
	out["upstream_provider_type"] = provider.Type
	out["upstream_provider_enabled"] = provider.Enabled
	out["upstream_key_name"] = item.UpstreamKeyName
	out["upstream_group_name"] = item.UpstreamGroupName
	out["upstream_rate_multiplier"] = item.UpstreamRateMultiplier
	out["upstream_synced_at"] = syncedAt.Format(time.RFC3339)
	return out
}

func upstreamAccountSyncExistingGroupIDs(account Account) []int64 {
	out := []int64{}
	for _, id := range account.GroupIDs {
		out = appendUniqueInt64(out, id)
	}
	for _, group := range account.Groups {
		if group == nil {
			continue
		}
		out = appendUniqueInt64(out, group.ID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func upstreamAccountSyncLowRateGroups(account Account, upstreamRate float64) ([]int64, []string, []int64) {
	lowIDs := []int64{}
	lowNames := []string{}
	lowSet := map[int64]struct{}{}
	for _, group := range account.Groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		if upstreamRate-group.RateMultiplier <= 0.0000001 {
			continue
		}
		if _, exists := lowSet[group.ID]; exists {
			continue
		}
		lowSet[group.ID] = struct{}{}
		lowIDs = append(lowIDs, group.ID)
		lowNames = append(lowNames, group.Name)
	}

	remaining := []int64{}
	for _, id := range upstreamAccountSyncExistingGroupIDs(account) {
		if _, low := lowSet[id]; low {
			continue
		}
		remaining = appendUniqueInt64(remaining, id)
	}
	return lowIDs, lowNames, remaining
}

func upstreamAccountSyncBoundGroups(account Account, upstreamRate float64) []UpstreamAccountSyncBoundGroup {
	out := make([]UpstreamAccountSyncBoundGroup, 0, len(account.Groups))
	seen := map[int64]struct{}{}
	for _, group := range account.Groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		if _, exists := seen[group.ID]; exists {
			continue
		}
		seen[group.ID] = struct{}{}
		out = append(out, UpstreamAccountSyncBoundGroup{
			ID:             group.ID,
			Name:           group.Name,
			RateMultiplier: group.RateMultiplier,
			RateViolation:  upstreamRate-group.RateMultiplier > 0.0000001,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func upstreamAccountSyncConflictAccounts(accounts []Account, upstreamRate float64) []UpstreamAccountSyncConflictAccount {
	out := make([]UpstreamAccountSyncConflictAccount, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, UpstreamAccountSyncConflictAccount{
			ID:          account.ID,
			Name:        account.Name,
			BoundGroups: upstreamAccountSyncBoundGroups(account, upstreamRate),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func buildUpstreamAccountSyncUnbindDetail(provider UpstreamProviderConfig, item UpstreamAccountSyncItem, account Account, unboundGroupIDs []int64, unboundGroupNames []string, remainingGroupIDs []int64, triggerSource string) UpstreamAccountSyncUnbindDetail {
	localMinRate, _ := upstreamAccountSyncMinGroupRate(account)
	return UpstreamAccountSyncUnbindDetail{
		ProviderSlug:            provider.Slug,
		ProviderName:            provider.Name,
		UpstreamKeyName:         item.UpstreamKeyName,
		MatchedLocalAccountID:   account.ID,
		MatchedLocalAccountName: account.Name,
		UpstreamGroupName:       item.UpstreamGroupName,
		UpstreamRateMultiplier:  item.UpstreamRateMultiplier,
		LocalMinRateMultiplier:  localMinRate,
		UnboundGroupIDs:         append([]int64(nil), unboundGroupIDs...),
		UnboundGroupNames:       append([]string(nil), unboundGroupNames...),
		RemainingGroupIDs:       append([]int64(nil), remainingGroupIDs...),
		TriggerSource:           normalizeUpstreamAccountSyncTriggerSource(triggerSource),
	}
}

func upstreamAccountSyncMinGroupRate(account Account) (float64, bool) {
	minRate := 0.0
	found := false
	for _, group := range account.Groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		if !found || group.RateMultiplier < minRate {
			minRate = group.RateMultiplier
			found = true
		}
	}
	return minRate, found
}

func logUpstreamAccountSyncUnbindAudit(detail UpstreamAccountSyncUnbindDetail) {
	if len(detail.UnboundGroupIDs) == 0 {
		return
	}
	logger.WriteSinkEvent("info", "audit.upstream_account_sync", "upstream_account_sync_unbound_low_rate_groups", map[string]any{
		"provider_slug":              detail.ProviderSlug,
		"provider_name":              detail.ProviderName,
		"upstream_key_name":          detail.UpstreamKeyName,
		"matched_local_account_id":   detail.MatchedLocalAccountID,
		"matched_local_account_name": detail.MatchedLocalAccountName,
		"upstream_group_name":        detail.UpstreamGroupName,
		"upstream_rate_multiplier":   detail.UpstreamRateMultiplier,
		"local_min_rate_multiplier":  detail.LocalMinRateMultiplier,
		"unbound_group_ids":          detail.UnboundGroupIDs,
		"unbound_group_names":        detail.UnboundGroupNames,
		"remaining_group_ids":        detail.RemainingGroupIDs,
		"trigger_source":             normalizeUpstreamAccountSyncTriggerSource(detail.TriggerSource),
	})
}

func normalizeUpstreamAccountSyncTriggerSource(triggerSource string) string {
	switch strings.TrimSpace(triggerSource) {
	case UpstreamAccountSyncTriggerScheduledRateGuard:
		return UpstreamAccountSyncTriggerScheduledRateGuard
	case UpstreamAccountSyncTriggerManualRateGuard:
		return UpstreamAccountSyncTriggerManualRateGuard
	default:
		return UpstreamAccountSyncTriggerManualSync
	}
}

func normalizeUpstreamAccountRateGuardTriggerSource(triggerSource string) string {
	switch strings.TrimSpace(triggerSource) {
	case UpstreamAccountSyncTriggerManualRateGuard:
		return UpstreamAccountSyncTriggerManualRateGuard
	default:
		return UpstreamAccountSyncTriggerScheduledRateGuard
	}
}

func upstreamAccountSyncRecordDetailKey(record UpstreamAccountSyncRecord, detail UpstreamAccountSyncUnbindDetail) string {
	parts := make([]string, 0, len(detail.UnboundGroupIDs))
	for _, id := range detail.UnboundGroupIDs {
		parts = append(parts, fmt.Sprint(id))
	}
	return fmt.Sprintf("%s-%d-%s-%s", record.CreatedAt.Format(time.RFC3339), detail.MatchedLocalAccountID, detail.UpstreamKeyName, strings.Join(parts, "_"))
}

func normalizeUpstreamAccountSyncRecordDetailKey(key string) string {
	key = strings.TrimSpace(key)
	for index := len(key) - 1; index >= 0; index-- {
		if key[index] != '-' {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, key[:index])
		if err != nil {
			continue
		}
		return parsed.Format(time.RFC3339) + key[index:]
	}
	return key
}

func accountIDs(accounts []Account) []int64 {
	out := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, account.ID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func appendUniqueInt64(values []int64, value int64) []int64 {
	if value <= 0 || containsInt64(values, value) {
		return values
	}
	return append(values, value)
}

func copyAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func limitUpstreamAccountSyncRecords(records []UpstreamAccountSyncRecord) []UpstreamAccountSyncRecord {
	if len(records) <= 100 {
		return records
	}
	out := make([]UpstreamAccountSyncRecord, 100)
	copy(out, records[:100])
	return out
}

func defaultUpstreamAccountRateGuardConfig() UpstreamAccountRateGuardConfig {
	return UpstreamAccountRateGuardConfig{
		Enabled:         false,
		IntervalSeconds: DefaultUpstreamAccountRateGuardIntervalSeconds,
	}
}

func normalizeUpstreamAccountRateGuardConfig(config UpstreamAccountRateGuardConfig) UpstreamAccountRateGuardConfig {
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = DefaultUpstreamAccountRateGuardIntervalSeconds
	}
	config.IgnoredAccountIDs = normalizeUpstreamAccountRateGuardIgnoredAccountIDs(config.IgnoredAccountIDs)
	return config
}

func normalizeUpstreamAccountRateGuardIgnoredAccountIDs(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		out = appendUniqueInt64(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func upstreamAccountSyncInt64Set(values []int64) map[int64]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		out[value] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func upstreamAccountSyncInt64SetContains(set map[int64]struct{}, value int64) bool {
	if len(set) == 0 || value <= 0 {
		return false
	}
	_, ok := set[value]
	return ok
}

func (s *UpstreamAccountSyncService) saveRateGuardConfig(ctx context.Context, config UpstreamAccountRateGuardConfig) (UpstreamAccountRateGuardConfig, error) {
	config = normalizeUpstreamAccountRateGuardConfig(config)
	if s == nil || s.settingRepo == nil {
		return config, nil
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return UpstreamAccountRateGuardConfig{}, fmt.Errorf("marshal upstream account rate guard config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamAccountRateGuardConfig, string(raw)); err != nil {
		return UpstreamAccountRateGuardConfig{}, fmt.Errorf("save upstream account rate guard config: %w", err)
	}
	return config, nil
}
