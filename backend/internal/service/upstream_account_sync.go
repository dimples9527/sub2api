package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

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

	DefaultUpstreamAccountRateGuardIntervalSeconds = 3600
	MinUpstreamAccountRateGuardIntervalSeconds     = 1
)

type UpstreamAccountSyncAccountManager interface {
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]Account, int64, error)
	CreateAccount(ctx context.Context, input *CreateAccountInput) (*Account, error)
	UpdateAccount(ctx context.Context, id int64, input *UpdateAccountInput) (*Account, error)
}

type UpstreamAccountSyncRequest struct {
	CreateMissing  bool `json:"create_missing"`
	UpdateExisting bool `json:"update_existing"`
	ApplyRateGuard bool `json:"apply_rate_guard"`
}

type UpstreamAccountRateGuardConfig struct {
	Enabled         bool       `json:"enabled"`
	IntervalSeconds int        `json:"interval_seconds"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastRunStatus   string     `json:"last_run_status,omitempty"`
	LastRunMessage  string     `json:"last_run_message,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
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
	Action                 string                          `json:"action"`
	ProviderSlug           string                          `json:"provider_slug"`
	ProviderName           string                          `json:"provider_name"`
	UpstreamKeyName        string                          `json:"upstream_key_name"`
	LocalAccountName       string                          `json:"local_account_name"`
	MatchedAccountID       *int64                          `json:"matched_account_id,omitempty"`
	MatchedAccountName     string                          `json:"matched_account_name,omitempty"`
	UpstreamGroupName      string                          `json:"upstream_group_name"`
	UpstreamRateMultiplier float64                         `json:"upstream_rate_multiplier"`
	LocalGroupID           *int64                          `json:"local_group_id,omitempty"`
	LocalGroupName         string                          `json:"local_group_name,omitempty"`
	LocalRateMultiplier    *float64                        `json:"local_rate_multiplier,omitempty"`
	RateViolation          bool                            `json:"rate_violation"`
	UnboundGroupIDs        []int64                         `json:"unbound_group_ids,omitempty"`
	UnboundGroupNames      []string                        `json:"unbound_group_names,omitempty"`
	SkipReason             string                          `json:"skip_reason,omitempty"`
	ConflictAccountIDs     []int64                         `json:"conflict_account_ids,omitempty"`
	BoundGroups            []UpstreamAccountSyncBoundGroup `json:"bound_groups,omitempty"`
}

type UpstreamAccountSyncBoundGroup struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	RateViolation  bool    `json:"rate_violation"`
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
}

type UpstreamAccountSyncService struct {
	providerSource UpstreamManagementProviderSource
	groupRepo      GroupRepository
	accountManager UpstreamAccountSyncAccountManager
	settingRepo    SettingRepository
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

func (s *UpstreamAccountSyncService) Preview(ctx context.Context) (UpstreamAccountSyncResult, error) {
	result, _, err := s.preview(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	records, err := s.ListRecords(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	return result, nil
}

func (s *UpstreamAccountSyncService) Sync(ctx context.Context, req UpstreamAccountSyncRequest) (UpstreamAccountSyncResult, error) {
	result, previewState, err := s.preview(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Summary.CreateCount = 0
	result.Summary.UpdateCount = 0
	result.Summary.RateViolationCount = 0
	result.Summary.UnboundGroupCount = 0

	now := time.Now().UTC()
	unbindDetails := make([]UpstreamAccountSyncUnbindDetail, 0)
	for index := range result.Items {
		item := &result.Items[index]
		provider := previewState.providerBySlug[item.ProviderSlug]
		if provider.Slug == "" {
			provider = result.DefaultProvider
		}
		switch item.Action {
		case UpstreamAccountSyncActionCreate:
			if !req.CreateMissing {
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
			_, err := s.accountManager.CreateAccount(ctx, &CreateAccountInput{
				Name:                  item.LocalAccountName,
				Platform:              PlatformOpenAI,
				Type:                  AccountTypeAPIKey,
				Credentials:           upstreamAccountSyncCredentials(nil, item.UpstreamKeyName, provider.BaseURL),
				Extra:                 extra,
				GroupIDs:              groupIDs,
				SkipDefaultGroupBind:  true,
				SkipMixedChannelCheck: true,
			})
			if err != nil {
				return s.finishSyncWithError(ctx, result, unbindDetails, err)
			}
			result.Summary.CreateCount++
		case UpstreamAccountSyncActionUpdate:
			if !req.UpdateExisting || item.MatchedAccountID == nil {
				continue
			}
			account := previewState.accountByID[*item.MatchedAccountID]
			if !upstreamAccountSyncAccountCompatible(account) {
				continue
			}
			nextGroupIDs := upstreamAccountSyncExistingGroupIDs(account)
			if item.LocalGroupID != nil {
				nextGroupIDs = appendUniqueInt64(nextGroupIDs, *item.LocalGroupID)
			}
			lowGroupIDs, lowGroupNames, remainingGroupIDs := upstreamAccountSyncLowRateGroups(account, item.UpstreamRateMultiplier)
			if req.ApplyRateGuard && len(lowGroupIDs) > 0 {
				nextGroupIDs = remainingGroupIDs
				item.RateViolation = true
				item.UnboundGroupIDs = lowGroupIDs
				item.UnboundGroupNames = lowGroupNames
			}
			extra := upstreamAccountSyncExtra(provider, *item, now, account.Extra)
			_, err := s.accountManager.UpdateAccount(ctx, account.ID, &UpdateAccountInput{
				Credentials:           upstreamAccountSyncCredentials(account.Credentials, item.UpstreamKeyName, provider.BaseURL),
				Extra:                 extra,
				GroupIDs:              &nextGroupIDs,
				SkipMixedChannelCheck: true,
			})
			if err != nil {
				return s.finishSyncWithError(ctx, result, unbindDetails, err)
			}
			if req.ApplyRateGuard && len(lowGroupIDs) > 0 {
				detail := buildUpstreamAccountSyncUnbindDetail(provider, *item, account, lowGroupIDs, lowGroupNames, remainingGroupIDs)
				unbindDetails = append(unbindDetails, detail)
				logUpstreamAccountSyncUnbindAudit(detail)
				result.Summary.RateViolationCount++
				result.Summary.UnboundGroupCount += len(lowGroupIDs)
			}
			item.Action = UpstreamAccountSyncActionUpdate
			result.Summary.UpdateCount++
		}
	}

	recordProviderSlug, recordProviderName := upstreamAccountSyncRecordProvider(result)
	records, err := s.prependRecord(ctx, UpstreamAccountSyncRecord{
		ProviderSlug:       recordProviderSlug,
		ProviderName:       recordProviderName,
		CreatedCount:       result.Summary.CreateCount,
		UpdatedCount:       result.Summary.UpdateCount,
		SkippedCount:       result.Summary.SkipCount,
		ConflictCount:      result.Summary.ConflictCount,
		RateViolationCount: result.Summary.RateViolationCount,
		UnboundGroupCount:  result.Summary.UnboundGroupCount,
		CreatedAt:          now,
		UnbindDetails:      unbindDetails,
	})
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	return result, nil
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
	return s.saveRateGuardConfig(ctx, config)
}

func (s *UpstreamAccountSyncService) RunScheduledRateGuard(ctx context.Context) (UpstreamAccountRateGuardConfig, error) {
	config, err := s.GetRateGuardConfig(ctx)
	if err != nil {
		return UpstreamAccountRateGuardConfig{}, err
	}
	now := time.Now().UTC()
	config.LastRunAt = &now
	_, runErr := s.Sync(ctx, UpstreamAccountSyncRequest{
		CreateMissing:  false,
		UpdateExisting: true,
		ApplyRateGuard: true,
	})
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

type upstreamAccountSyncPreviewState struct {
	accountByID    map[int64]Account
	providerBySlug map[string]UpstreamProviderConfig
}

func (s *UpstreamAccountSyncService) preview(ctx context.Context) (UpstreamAccountSyncResult, upstreamAccountSyncPreviewState, error) {
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
	accounts, err := s.loadAccounts(ctx)
	if err != nil {
		return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
	}

	accountsByName := upstreamAccountSyncAccountsByName(accounts)
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
	for _, provider := range syncProviders {
		groupResolver := newUpstreamAccountSyncGroupResolver(provider, localGroups, mappings)
		keys, warnings, err := s.providerSource.FetchProviderKeys(ctx, provider.Slug)
		if err != nil {
			return UpstreamAccountSyncResult{}, upstreamAccountSyncPreviewState{}, err
		}
		for _, warning := range warnings {
			if strings.TrimSpace(warning) == "" {
				continue
			}
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", provider.Name, warning))
		}
		for _, key := range keys {
			if key.ProviderSlug != "" && key.ProviderSlug != provider.Slug {
				continue
			}
			result.Summary.UpstreamKeyCount++
			item := UpstreamAccountSyncItem{
				ProviderSlug:           provider.Slug,
				ProviderName:           provider.Name,
				UpstreamKeyName:        strings.TrimSpace(key.KeyName),
				LocalAccountName:       upstreamProviderKeyName(provider, key.KeyName),
				UpstreamGroupName:      strings.TrimSpace(key.GroupName),
				UpstreamRateMultiplier: key.RateMultiplier,
			}
			if item.LocalAccountName == "" {
				item.Action = UpstreamAccountSyncActionSkip
				item.SkipReason = "upstream key name is empty"
				result.Summary.SkipCount++
				result.Items = append(result.Items, item)
				continue
			}
			if group, ok := groupResolver.resolve(key.GroupName); ok {
				groupID := group.ID
				rate := group.RateMultiplier
				item.LocalGroupID = &groupID
				item.LocalGroupName = group.Name
				item.LocalRateMultiplier = &rate
			} else {
				item.Action = UpstreamAccountSyncActionSkip
				item.SkipReason = "upstream group is not matched"
				result.Summary.SkipCount++
				result.Items = append(result.Items, item)
				continue
			}

			matches := accountsByName[normalizeUpstreamGroupMatchName(item.LocalAccountName)]
			if len(matches) > 1 {
				item.Action = UpstreamAccountSyncActionConflict
				item.ConflictAccountIDs = accountIDs(matches)
				result.Summary.ConflictCount++
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
			accountID := account.ID
			item.MatchedAccountID = &accountID
			item.MatchedAccountName = account.Name
			item.BoundGroups = upstreamAccountSyncBoundGroups(account, item.UpstreamRateMultiplier)
			result.Summary.MatchedAccountCount++
			if !upstreamAccountSyncAccountCompatible(account) {
				item.Action = UpstreamAccountSyncActionSkip
				item.SkipReason = "matched account is not an OpenAI API key account"
				result.Summary.SkipCount++
				result.Items = append(result.Items, item)
				continue
			}
			lowGroupIDs, lowGroupNames, _ := upstreamAccountSyncLowRateGroups(account, item.UpstreamRateMultiplier)
			if len(lowGroupIDs) > 0 {
				item.RateViolation = true
				item.UnboundGroupIDs = lowGroupIDs
				item.UnboundGroupNames = lowGroupNames
				result.Summary.RateViolationCount++
				result.Summary.UnboundGroupCount += len(lowGroupIDs)
			}
			if upstreamAccountSyncNeedsUpdate(account, item, provider) || item.RateViolation {
				item.Action = UpstreamAccountSyncActionUpdate
				result.Summary.UpdateCount++
			} else {
				item.Action = UpstreamAccountSyncActionNoop
			}
			result.Items = append(result.Items, item)
		}
	}
	return result, upstreamAccountSyncPreviewState{accountByID: accountByID, providerBySlug: providerBySlug}, nil
}

func (s *UpstreamAccountSyncService) listAccountSyncProviders(ctx context.Context, defaultProvider UpstreamProviderConfig) ([]UpstreamProviderConfig, error) {
	source, ok := s.providerSource.(interface {
		ListProviders(context.Context) ([]UpstreamProviderConfig, error)
	})
	if !ok {
		return []UpstreamProviderConfig{}, nil
	}
	providers, err := source.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list upstream account sync providers: %w", err)
	}
	out := make([]UpstreamProviderConfig, 0, len(providers))
	for _, provider := range providers {
		if provider.Slug == "" || provider.Slug == defaultProvider.Slug || provider.IsDefault || !provider.Enabled {
			continue
		}
		out = append(out, provider)
	}
	return out, nil
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

func (s *UpstreamAccountSyncService) prependRecord(ctx context.Context, record UpstreamAccountSyncRecord) ([]UpstreamAccountSyncRecord, error) {
	existing, err := s.ListRecords(ctx)
	if err != nil {
		return nil, err
	}
	records := append([]UpstreamAccountSyncRecord{record}, existing...)
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

func (s *UpstreamAccountSyncService) finishSyncWithError(ctx context.Context, result UpstreamAccountSyncResult, unbindDetails []UpstreamAccountSyncUnbindDetail, runErr error) (UpstreamAccountSyncResult, error) {
	providerSlug, providerName := upstreamAccountSyncRecordProvider(result)
	record := UpstreamAccountSyncRecord{
		ProviderSlug:       providerSlug,
		ProviderName:       providerName,
		CreatedCount:       result.Summary.CreateCount,
		UpdatedCount:       result.Summary.UpdateCount,
		SkippedCount:       result.Summary.SkipCount,
		ConflictCount:      result.Summary.ConflictCount,
		RateViolationCount: result.Summary.RateViolationCount,
		UnboundGroupCount:  result.Summary.UnboundGroupCount,
		CreatedAt:          time.Now().UTC(),
		Error:              runErr.Error(),
		UnbindDetails:      unbindDetails,
	}
	records, err := s.prependRecord(ctx, record)
	if err != nil {
		return UpstreamAccountSyncResult{}, err
	}
	result.Records = records
	return result, runErr
}

func upstreamAccountSyncRecordProvider(result UpstreamAccountSyncResult) (string, string) {
	if len(result.Providers) == 1 {
		return result.Providers[0].Slug, result.Providers[0].Name
	}
	if len(result.Providers) > 1 {
		return "multiple", "Multiple upstream providers"
	}
	return result.DefaultProvider.Slug, result.DefaultProvider.Name
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

func upstreamAccountSyncAccountCompatible(account Account) bool {
	return account.Platform == PlatformOpenAI && account.Type == AccountTypeAPIKey
}

func upstreamAccountSyncNeedsUpdate(account Account, item UpstreamAccountSyncItem, provider UpstreamProviderConfig) bool {
	if strings.TrimSpace(account.GetCredential("api_key")) != strings.TrimSpace(item.UpstreamKeyName) {
		return true
	}
	if strings.TrimRight(strings.TrimSpace(account.GetCredential("base_url")), "/") != strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/") {
		return true
	}
	if item.LocalGroupID != nil && !containsInt64(upstreamAccountSyncExistingGroupIDs(account), *item.LocalGroupID) {
		return true
	}
	return false
}

func upstreamAccountSyncCredentials(existing map[string]any, apiKey, baseURL string) map[string]any {
	out := copyAnyMap(existing)
	if out == nil {
		out = map[string]any{}
	}
	out["api_key"] = strings.TrimSpace(apiKey)
	out["base_url"] = strings.TrimRight(strings.TrimSpace(baseURL), "/")
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

func buildUpstreamAccountSyncUnbindDetail(provider UpstreamProviderConfig, item UpstreamAccountSyncItem, account Account, unboundGroupIDs []int64, unboundGroupNames []string, remainingGroupIDs []int64) UpstreamAccountSyncUnbindDetail {
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
	})
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
	return config
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
