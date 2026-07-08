package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettingKeyUpstreamGroupRateFixRecords = "upstream_group_rate_fix_records"
	SettingKeyUpstreamGroupMappings       = "upstream_group_mappings"
	SettingKeyUpstreamGroupRateFixConfig  = "upstream_group_rate_fix_config"

	DefaultUpstreamGroupRateFixIntervalSeconds = 3600
	MinUpstreamGroupRateFixIntervalSeconds     = 1
)

type UpstreamManagementProviderSource interface {
	GetDefaultProvider(ctx context.Context) (UpstreamProviderConfig, error)
	FetchProviderKeys(ctx context.Context, slug string) ([]UpstreamProviderKey, []string, error)
}

type upstreamManagementProviderGroupSource interface {
	FetchProviderGroups(ctx context.Context, slug string) ([]UpstreamProviderGroup, []string, error)
}

type UpstreamGroupComparison struct {
	ProviderSlug       string   `json:"provider_slug"`
	ProviderName       string   `json:"provider_name"`
	UpstreamGroupName  string   `json:"upstream_group_name"`
	UpstreamGroupKey   string   `json:"upstream_group_key"`
	UpstreamRate       float64  `json:"upstream_rate"`
	UpstreamKeyCount   int      `json:"upstream_key_count"`
	LocalGroupID       *int64   `json:"local_group_id,omitempty"`
	LocalGroupName     string   `json:"local_group_name,omitempty"`
	LocalGroupPlatform string   `json:"local_group_platform,omitempty"`
	LocalRate          *float64 `json:"local_rate,omitempty"`
	Matched            bool     `json:"matched"`
	MatchSource        string   `json:"match_source,omitempty"`
	MatchIgnored       bool     `json:"match_ignored,omitempty"`
	NeedsRateIncrease  bool     `json:"needs_rate_increase"`
}

type UpstreamGroupCompareResult struct {
	DefaultProvider UpstreamProviderConfig       `json:"default_provider"`
	Items           []UpstreamGroupComparison    `json:"items"`
	Warnings        []string                     `json:"warnings,omitempty"`
	Records         []UpstreamGroupRateFixRecord `json:"records"`
}

type UpstreamGroupRateFixRecord struct {
	GroupID           int64     `json:"group_id"`
	GroupName         string    `json:"group_name"`
	ProviderSlug      string    `json:"provider_slug"`
	ProviderName      string    `json:"provider_name"`
	UpstreamGroupName string    `json:"upstream_group_name"`
	OldRate           float64   `json:"old_rate"`
	NewRate           float64   `json:"new_rate"`
	ChangedAt         time.Time `json:"changed_at"`
	Handled           bool      `json:"handled,omitempty"`
}

type UpstreamGroupAutoRateFixConfig struct {
	Enabled         bool       `json:"enabled"`
	IntervalSeconds int        `json:"interval_seconds"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastRunStatus   string     `json:"last_run_status,omitempty"`
	LastRunMessage  string     `json:"last_run_message,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type UpstreamGroupMappingInput struct {
	UpstreamGroupName string `json:"upstream_group_name"`
	LocalGroupID      *int64 `json:"local_group_id"`
	Ignored           bool   `json:"ignored,omitempty"`
}

type UpstreamGroupLocalCreateInput struct {
	UpstreamGroupName string  `json:"upstream_group_name"`
	Platform          string  `json:"platform"`
	RateMultiplier    float64 `json:"rate_multiplier"`
}

type UpstreamGroupMappingRecord struct {
	ProviderSlug      string    `json:"provider_slug"`
	UpstreamGroupName string    `json:"upstream_group_name"`
	UpstreamGroupKey  string    `json:"upstream_group_key"`
	LocalGroupID      int64     `json:"local_group_id"`
	Ignored           bool      `json:"ignored,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type UpstreamManagementService struct {
	providerSource       UpstreamManagementProviderSource
	groupRepo            GroupRepository
	settingRepo          SettingRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func NewUpstreamManagementService(
	providerSource UpstreamManagementProviderSource,
	groupRepo GroupRepository,
	settingRepo SettingRepository,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
) *UpstreamManagementService {
	return &UpstreamManagementService{
		providerSource:       providerSource,
		groupRepo:            groupRepo,
		settingRepo:          settingRepo,
		authCacheInvalidator: authCacheInvalidator,
	}
}

func (s *UpstreamManagementService) FetchDefaultModelSquare(ctx context.Context) (json.RawMessage, UpstreamProviderConfig, error) {
	source, ok := s.providerSource.(interface {
		FetchDefaultModelSquare(context.Context) (json.RawMessage, UpstreamProviderConfig, error)
	})
	if !ok {
		return nil, UpstreamProviderConfig{}, infraerrors.InternalServer("UPSTREAM_MODEL_SQUARE_UNAVAILABLE", "upstream model square service is unavailable")
	}
	payload, provider, err := source.FetchDefaultModelSquare(ctx)
	if err != nil {
		return nil, UpstreamProviderConfig{}, err
	}
	merged, err := s.mergeDefaultModelSquareGroups(ctx, payload, provider)
	if err != nil {
		return nil, UpstreamProviderConfig{}, err
	}
	return merged, provider, nil
}

func (s *UpstreamManagementService) mergeDefaultModelSquareGroups(ctx context.Context, payload json.RawMessage, provider UpstreamProviderConfig) (json.RawMessage, error) {
	if s == nil || s.groupRepo == nil || len(payload) == 0 {
		return payload, nil
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, fmt.Errorf("decode upstream model-square response: %w", err)
	}
	container := modelSquarePayloadContainer(body)
	rawGroups, ok := container["groups"].([]any)
	if !ok || len(rawGroups) == 0 {
		return payload, nil
	}

	localGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list local groups for model square: %w", err)
	}
	localByID := make(map[int64]Group, len(localGroups))
	localByName := make(map[string]Group, len(localGroups))
	for _, group := range localGroups {
		localByID[group.ID] = group
		key := normalizeUpstreamGroupMatchName(group.Name)
		if key != "" {
			if _, exists := localByName[key]; !exists {
				localByName[key] = group
			}
		}
	}

	mappings, err := s.loadGroupMappings(ctx)
	if err != nil {
		return nil, err
	}
	mappedLocalGroupIDs := make(map[string]int64, len(mappings))
	ignoredGroupKeys := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		if mapping.ProviderSlug != provider.Slug {
			continue
		}
		key := normalizeUpstreamGroupMatchName(mapping.UpstreamGroupKey)
		if key == "" {
			key = normalizeUpstreamGroupMatchName(mapping.UpstreamGroupName)
		}
		if key == "" {
			continue
		}
		if mapping.Ignored {
			ignoredGroupKeys[key] = struct{}{}
			continue
		}
		if mapping.LocalGroupID > 0 {
			mappedLocalGroupIDs[key] = mapping.LocalGroupID
		}
	}

	keptGroupIDs := make(map[string]struct{}, len(rawGroups))
	mergedGroups := make([]any, 0, len(rawGroups))
	for _, rawGroup := range rawGroups {
		groupMap, ok := rawGroup.(map[string]any)
		if !ok {
			continue
		}
		remoteID, hasID := groupMap["id"]
		if !hasID {
			continue
		}
		remoteName, _ := groupMap["name"].(string)
		key := normalizeUpstreamGroupMatchName(remoteName)
		if _, ignored := ignoredGroupKeys[key]; ignored {
			continue
		}
		localGroup, matched := localByName[key]
		if mappedID, ok := mappedLocalGroupIDs[key]; ok {
			if mappedLocal, exists := localByID[mappedID]; exists {
				localGroup = mappedLocal
				matched = true
			}
		}
		if !matched {
			continue
		}

		mergedGroup := modelSquareGroupMapFromLocal(localGroup)
		mergedGroup["id"] = remoteID
		keptGroupIDs[modelSquareIDKey(remoteID)] = struct{}{}
		mergedGroups = append(mergedGroups, mergedGroup)
	}

	container["groups"] = mergedGroups
	filterModelSquareGroupIDs(container, keptGroupIDs)
	mergedPayload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode merged upstream model-square response: %w", err)
	}
	return mergedPayload, nil
}

func modelSquarePayloadContainer(body map[string]any) map[string]any {
	if data, ok := body["data"].(map[string]any); ok {
		if _, hasGroups := data["groups"]; hasGroups {
			return data
		}
		if _, hasModels := data["models"]; hasModels {
			return data
		}
	}
	return body
}

func modelSquareGroupMapFromLocal(group Group) map[string]any {
	return map[string]any{
		"id":                     group.ID,
		"name":                   group.Name,
		"description":            group.Description,
		"platform":               group.Platform,
		"rate_multiplier":        group.RateMultiplier,
		"status":                 group.Status,
		"is_exclusive":           group.IsExclusive,
		"subscription_type":      group.SubscriptionType,
		"allow_image_generation": group.AllowImageGeneration,
		"image_rate_independent": group.ImageRateIndependent,
		"image_rate_multiplier":  group.ImageRateMultiplier,
	}
}

func filterModelSquareGroupIDs(container map[string]any, keptGroupIDs map[string]struct{}) {
	rawModels, ok := container["models"].([]any)
	if !ok {
		return
	}
	for _, rawModel := range rawModels {
		modelMap, ok := rawModel.(map[string]any)
		if !ok {
			continue
		}
		rawGroupIDs, ok := modelMap["group_ids"].([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(rawGroupIDs))
		for _, groupID := range rawGroupIDs {
			if _, ok := keptGroupIDs[modelSquareIDKey(groupID)]; ok {
				filtered = append(filtered, groupID)
			}
		}
		modelMap["group_ids"] = filtered
	}
}

func modelSquareIDKey(value any) string {
	return fmt.Sprint(value)
}

func (s *UpstreamManagementService) CompareGroups(ctx context.Context) (UpstreamGroupCompareResult, error) {
	result, err := s.compareGroups(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	records, err := s.loadRateFixRecords(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	result.Records = records
	return result, nil
}

func (s *UpstreamManagementService) GetRateFixConfig(ctx context.Context) (UpstreamGroupAutoRateFixConfig, error) {
	if s == nil || s.settingRepo == nil {
		return defaultUpstreamGroupRateFixConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamGroupRateFixConfig)
	if err != nil {
		if err == ErrSettingNotFound {
			return defaultUpstreamGroupRateFixConfig(), nil
		}
		return UpstreamGroupAutoRateFixConfig{}, fmt.Errorf("load upstream group rate fix config: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultUpstreamGroupRateFixConfig(), nil
	}
	var config UpstreamGroupAutoRateFixConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return UpstreamGroupAutoRateFixConfig{}, infraerrors.InternalServer("UPSTREAM_GROUP_RATE_FIX_CONFIG_INVALID", "upstream group rate fix config is invalid")
	}
	return normalizeUpstreamGroupRateFixConfig(config), nil
}

func (s *UpstreamManagementService) UpdateRateFixConfig(ctx context.Context, input UpstreamGroupAutoRateFixConfig) (UpstreamGroupAutoRateFixConfig, error) {
	config := normalizeUpstreamGroupRateFixConfig(input)
	if input.IntervalSeconds > 0 && input.IntervalSeconds < MinUpstreamGroupRateFixIntervalSeconds {
		return UpstreamGroupAutoRateFixConfig{}, infraerrors.BadRequest("UPSTREAM_GROUP_RATE_FIX_INTERVAL_INVALID", fmt.Sprintf("interval_seconds must be at least %d", MinUpstreamGroupRateFixIntervalSeconds))
	}
	now := time.Now().UTC()
	config.UpdatedAt = &now
	if s == nil || s.settingRepo == nil {
		return config, nil
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return UpstreamGroupAutoRateFixConfig{}, fmt.Errorf("marshal upstream group rate fix config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamGroupRateFixConfig, string(raw)); err != nil {
		return UpstreamGroupAutoRateFixConfig{}, fmt.Errorf("save upstream group rate fix config: %w", err)
	}
	return config, nil
}

func (s *UpstreamManagementService) RunScheduledRateFix(ctx context.Context) (UpstreamGroupAutoRateFixConfig, error) {
	config, err := s.GetRateFixConfig(ctx)
	if err != nil {
		return UpstreamGroupAutoRateFixConfig{}, err
	}
	now := time.Now().UTC()
	config.LastRunAt = &now
	_, runErr := s.ApplyRateFixes(ctx)
	if runErr != nil {
		config.LastRunStatus = "failed"
		config.LastRunMessage = runErr.Error()
	} else {
		config.LastRunStatus = "success"
		config.LastRunMessage = ""
	}
	config.UpdatedAt = &now
	if s != nil && s.settingRepo != nil {
		raw, marshalErr := json.Marshal(config)
		if marshalErr != nil {
			return UpstreamGroupAutoRateFixConfig{}, fmt.Errorf("marshal upstream group rate fix config: %w", marshalErr)
		}
		if saveErr := s.settingRepo.Set(ctx, SettingKeyUpstreamGroupRateFixConfig, string(raw)); saveErr != nil {
			return UpstreamGroupAutoRateFixConfig{}, fmt.Errorf("save upstream group rate fix config: %w", saveErr)
		}
	}
	return config, runErr
}

func (s *UpstreamManagementService) ApplyRateFixes(ctx context.Context) (UpstreamGroupCompareResult, error) {
	result, err := s.compareGroups(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	newRecords := make([]UpstreamGroupRateFixRecord, 0)
	now := time.Now().UTC()
	for _, item := range result.Items {
		if !item.Matched || !item.NeedsRateIncrease || item.LocalGroupID == nil || item.LocalRate == nil {
			continue
		}
		group, err := s.groupRepo.GetByID(ctx, *item.LocalGroupID)
		if err != nil {
			return UpstreamGroupCompareResult{}, fmt.Errorf("get local group for rate fix: %w", err)
		}
		if item.UpstreamRate <= group.RateMultiplier {
			continue
		}
		oldRate := group.RateMultiplier
		group.RateMultiplier = item.UpstreamRate
		if err := s.groupRepo.Update(ctx, group); err != nil {
			return UpstreamGroupCompareResult{}, fmt.Errorf("update local group rate: %w", err)
		}
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, group.ID)
		}
		newRecords = append(newRecords, UpstreamGroupRateFixRecord{
			GroupID:           group.ID,
			GroupName:         group.Name,
			ProviderSlug:      item.ProviderSlug,
			ProviderName:      item.ProviderName,
			UpstreamGroupName: item.UpstreamGroupName,
			OldRate:           oldRate,
			NewRate:           item.UpstreamRate,
			ChangedAt:         now,
		})
	}
	records, err := s.prependRateFixRecords(ctx, newRecords)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	result, err = s.compareGroups(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	result.Records = records
	return result, nil
}

func (s *UpstreamManagementService) MarkRateFixRecordHandled(ctx context.Context, key string) ([]UpstreamGroupRateFixRecord, error) {
	records, err := s.loadRateFixRecords(ctx)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_GROUP_RATE_FIX_RECORD_KEY_REQUIRED", "upstream group rate fix record key is required")
	}

	found := false
	for index := range records {
		if upstreamGroupRateFixRecordKey(records[index]) != key {
			continue
		}
		records[index].Handled = true
		found = true
		break
	}
	if !found {
		return nil, infraerrors.NotFound("UPSTREAM_GROUP_RATE_FIX_RECORD_NOT_FOUND", "upstream group rate fix record was not found")
	}
	if s == nil || s.settingRepo == nil {
		return records, nil
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream group rate fix records: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamGroupRateFixRecords, string(raw)); err != nil {
		return nil, fmt.Errorf("save upstream group rate fix records: %w", err)
	}
	return records, nil
}

func (s *UpstreamManagementService) SaveGroupMapping(ctx context.Context, input UpstreamGroupMappingInput) (UpstreamGroupCompareResult, error) {
	if s == nil || s.providerSource == nil {
		return UpstreamGroupCompareResult{}, infraerrors.InternalServer("UPSTREAM_MANAGEMENT_PROVIDER_SOURCE_UNAVAILABLE", "upstream management provider source unavailable")
	}
	if s.groupRepo == nil {
		return UpstreamGroupCompareResult{}, infraerrors.InternalServer("UPSTREAM_MANAGEMENT_GROUP_REPO_UNAVAILABLE", "upstream management group repository unavailable")
	}
	defaultProvider, err := s.providerSource.GetDefaultProvider(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	upstreamGroupName := strings.TrimSpace(input.UpstreamGroupName)
	upstreamGroupKey := normalizeUpstreamGroupMatchName(upstreamGroupName)
	if upstreamGroupKey == "" {
		return UpstreamGroupCompareResult{}, infraerrors.BadRequest("UPSTREAM_GROUP_NAME_REQUIRED", "upstream group name is required")
	}

	records, err := s.loadGroupMappings(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	records = removeUpstreamGroupMapping(records, defaultProvider.Slug, upstreamGroupKey)
	if input.LocalGroupID != nil && *input.LocalGroupID > 0 {
		group, err := s.groupRepo.GetByID(ctx, *input.LocalGroupID)
		if err != nil {
			return UpstreamGroupCompareResult{}, fmt.Errorf("get local group for upstream mapping: %w", err)
		}
		if group.Status != StatusActive {
			return UpstreamGroupCompareResult{}, infraerrors.BadRequest("UPSTREAM_GROUP_MAPPING_LOCAL_GROUP_INACTIVE", "local group must be active")
		}
		records = append(records, UpstreamGroupMappingRecord{
			ProviderSlug:      defaultProvider.Slug,
			UpstreamGroupName: upstreamGroupName,
			UpstreamGroupKey:  upstreamGroupKey,
			LocalGroupID:      group.ID,
			UpdatedAt:         time.Now().UTC(),
		})
	} else if input.Ignored {
		records = append(records, UpstreamGroupMappingRecord{
			ProviderSlug:      defaultProvider.Slug,
			UpstreamGroupName: upstreamGroupName,
			UpstreamGroupKey:  upstreamGroupKey,
			Ignored:           true,
			UpdatedAt:         time.Now().UTC(),
		})
	}
	if err := s.saveGroupMappings(ctx, records); err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	return s.CompareGroups(ctx)
}

func (s *UpstreamManagementService) CreateLocalGroupFromUpstream(ctx context.Context, input UpstreamGroupLocalCreateInput) (UpstreamGroupCompareResult, error) {
	if s == nil || s.providerSource == nil {
		return UpstreamGroupCompareResult{}, infraerrors.InternalServer("UPSTREAM_MANAGEMENT_PROVIDER_SOURCE_UNAVAILABLE", "upstream management provider source unavailable")
	}
	if s.groupRepo == nil {
		return UpstreamGroupCompareResult{}, infraerrors.InternalServer("UPSTREAM_MANAGEMENT_GROUP_REPO_UNAVAILABLE", "upstream management group repository unavailable")
	}
	defaultProvider, err := s.providerSource.GetDefaultProvider(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	upstreamGroupName := strings.TrimSpace(input.UpstreamGroupName)
	upstreamGroupKey := normalizeUpstreamGroupMatchName(upstreamGroupName)
	if upstreamGroupKey == "" {
		return UpstreamGroupCompareResult{}, infraerrors.BadRequest("UPSTREAM_GROUP_NAME_REQUIRED", "upstream group name is required")
	}
	if input.RateMultiplier <= 0 {
		return UpstreamGroupCompareResult{}, infraerrors.BadRequest("UPSTREAM_GROUP_RATE_INVALID", "upstream group rate multiplier must be greater than 0")
	}
	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	if platform == "" {
		return UpstreamGroupCompareResult{}, infraerrors.BadRequest("UPSTREAM_GROUP_PLATFORM_REQUIRED", "platform is required")
	}
	if platform != PlatformAnthropic && platform != PlatformOpenAI && platform != PlatformGemini && platform != PlatformAntigravity {
		return UpstreamGroupCompareResult{}, infraerrors.BadRequest("UPSTREAM_GROUP_PLATFORM_INVALID", "platform is invalid")
	}
	upstreamGroupName, err = s.resolveDefaultUpstreamGroupName(ctx, defaultProvider, upstreamGroupKey)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	exists, err := s.groupRepo.ExistsByName(ctx, upstreamGroupName)
	if err != nil {
		return UpstreamGroupCompareResult{}, fmt.Errorf("check local group exists: %w", err)
	}
	if exists {
		return UpstreamGroupCompareResult{}, ErrGroupExists
	}
	localGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, fmt.Errorf("list local groups before create: %w", err)
	}
	for _, localGroup := range localGroups {
		if normalizeUpstreamGroupMatchName(localGroup.Name) == upstreamGroupKey {
			return UpstreamGroupCompareResult{}, ErrGroupExists
		}
	}

	group := &Group{
		Name:                 upstreamGroupName,
		Platform:             platform,
		RateMultiplier:       input.RateMultiplier,
		Status:               StatusActive,
		SubscriptionType:     SubscriptionTypeStandard,
		ImageRateMultiplier:  1,
		AllowImageGeneration: false,
		ImageRateIndependent: false,
	}
	if err := s.groupRepo.Create(ctx, group); err != nil {
		return UpstreamGroupCompareResult{}, fmt.Errorf("create local group from upstream group: %w", err)
	}

	records, err := s.loadGroupMappings(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	records = removeUpstreamGroupMapping(records, defaultProvider.Slug, upstreamGroupKey)
	records = append(records, UpstreamGroupMappingRecord{
		ProviderSlug:      defaultProvider.Slug,
		UpstreamGroupName: upstreamGroupName,
		UpstreamGroupKey:  upstreamGroupKey,
		LocalGroupID:      group.ID,
		UpdatedAt:         time.Now().UTC(),
	})
	if err := s.saveGroupMappings(ctx, records); err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	return s.CompareGroups(ctx)
}

func defaultUpstreamGroupRateFixConfig() UpstreamGroupAutoRateFixConfig {
	return UpstreamGroupAutoRateFixConfig{
		Enabled:         false,
		IntervalSeconds: DefaultUpstreamGroupRateFixIntervalSeconds,
	}
}

func normalizeUpstreamGroupRateFixConfig(config UpstreamGroupAutoRateFixConfig) UpstreamGroupAutoRateFixConfig {
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = DefaultUpstreamGroupRateFixIntervalSeconds
	}
	return config
}

func (s *UpstreamManagementService) resolveDefaultUpstreamGroupName(ctx context.Context, defaultProvider UpstreamProviderConfig, upstreamGroupKey string) (string, error) {
	if source, ok := s.providerSource.(upstreamManagementProviderGroupSource); ok {
		groups, _, err := source.FetchProviderGroups(ctx, defaultProvider.Slug)
		if err == nil {
			for _, group := range groups {
				if group.ProviderSlug != "" && group.ProviderSlug != defaultProvider.Slug {
					continue
				}
				if normalizeUpstreamGroupMatchName(group.GroupName) != upstreamGroupKey {
					continue
				}
				name := strings.TrimSpace(group.GroupName)
				if name != "" {
					return name, nil
				}
			}
		}
	}
	keys, _, err := s.providerSource.FetchProviderKeys(ctx, defaultProvider.Slug)
	if err != nil {
		return "", err
	}
	for _, key := range keys {
		if key.ProviderSlug != "" && key.ProviderSlug != defaultProvider.Slug {
			continue
		}
		if normalizeUpstreamGroupMatchName(key.GroupName) != upstreamGroupKey {
			continue
		}
		name := strings.TrimSpace(key.GroupName)
		if name != "" {
			return name, nil
		}
	}
	return "", infraerrors.BadRequest("UPSTREAM_GROUP_NOT_FOUND", "upstream group does not exist on default provider")
}

func (s *UpstreamManagementService) compareGroups(ctx context.Context) (UpstreamGroupCompareResult, error) {
	if s == nil || s.providerSource == nil {
		return UpstreamGroupCompareResult{}, infraerrors.InternalServer("UPSTREAM_MANAGEMENT_PROVIDER_SOURCE_UNAVAILABLE", "upstream management provider source unavailable")
	}
	if s.groupRepo == nil {
		return UpstreamGroupCompareResult{}, infraerrors.InternalServer("UPSTREAM_MANAGEMENT_GROUP_REPO_UNAVAILABLE", "upstream management group repository unavailable")
	}
	defaultProvider, err := s.providerSource.GetDefaultProvider(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	keys, warnings, err := s.providerSource.FetchProviderKeys(ctx, defaultProvider.Slug)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	groups := []UpstreamProviderGroup{}
	groupsAuthoritative := false
	if source, ok := s.providerSource.(upstreamManagementProviderGroupSource); ok {
		providerGroups, groupWarnings, err := source.FetchProviderGroups(ctx, defaultProvider.Slug)
		if err == nil {
			warnings = append(warnings, groupWarnings...)
			groups = providerGroups
			groupsAuthoritative = true
		} else if !isUpstreamProviderGroupsUnsupported(err) {
			warnings = append(warnings, groupWarnings...)
			warnings = append(warnings, fmt.Sprintf("fetch upstream groups failed: %v", err))
		}
	}
	localGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, fmt.Errorf("list local groups: %w", err)
	}
	mappings, err := s.loadGroupMappings(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	if groupsAuthoritative {
		prunedMappings, changed := pruneMissingUpstreamGroupMappings(mappings, defaultProvider.Slug, groups)
		if changed {
			if err := s.saveGroupMappings(ctx, prunedMappings); err != nil {
				return UpstreamGroupCompareResult{}, err
			}
			mappings = prunedMappings
		}
	}
	items := compareUpstreamGroups(defaultProvider, keys, groups, localGroups, mappings, groupsAuthoritative)
	return UpstreamGroupCompareResult{
		DefaultProvider: redactUpstreamProvider(defaultProvider),
		Items:           items,
		Warnings:        warnings,
		Records:         []UpstreamGroupRateFixRecord{},
	}, nil
}

func isUpstreamProviderGroupsUnsupported(err error) bool {
	return infraerrors.Reason(err) == "UPSTREAM_PROVIDER_GROUPS_UNSUPPORTED"
}

func compareUpstreamGroups(provider UpstreamProviderConfig, keys []UpstreamProviderKey, groups []UpstreamProviderGroup, localGroups []Group, mappings []UpstreamGroupMappingRecord, groupsAuthoritative bool) []UpstreamGroupComparison {
	localByName := make(map[string]Group, len(localGroups))
	localByID := make(map[int64]Group, len(localGroups))
	for _, group := range localGroups {
		localByID[group.ID] = group
		normalized := normalizeUpstreamGroupMatchName(group.Name)
		if normalized == "" {
			continue
		}
		if _, exists := localByName[normalized]; !exists {
			localByName[normalized] = group
		}
	}
	mappedLocalGroupIDs := make(map[string]int64, len(mappings))
	ignoredGroupKeys := make(map[string]struct{}, len(mappings))
	for _, mapping := range mappings {
		if mapping.ProviderSlug != provider.Slug {
			continue
		}
		key := normalizeUpstreamGroupMatchName(mapping.UpstreamGroupKey)
		if key == "" {
			key = normalizeUpstreamGroupMatchName(mapping.UpstreamGroupName)
		}
		if key == "" {
			continue
		}
		if mapping.Ignored {
			ignoredGroupKeys[key] = struct{}{}
			continue
		}
		if mapping.LocalGroupID > 0 {
			mappedLocalGroupIDs[key] = mapping.LocalGroupID
		}
	}

	keyCounts := map[string]int{}
	for _, key := range keys {
		if key.ProviderSlug != "" && key.ProviderSlug != provider.Slug {
			continue
		}
		normalized := normalizeUpstreamGroupMatchName(key.GroupName)
		if normalized == "" {
			continue
		}
		keyCounts[normalized]++
	}

	type upstreamGroupAggregate struct {
		name     string
		rate     float64
		keyCount int
	}
	aggregates := map[string]upstreamGroupAggregate{}

	for _, group := range groups {
		if group.ProviderSlug != "" && group.ProviderSlug != provider.Slug {
			continue
		}
		normalized := normalizeUpstreamGroupMatchName(group.GroupName)
		if normalized == "" {
			continue
		}
		aggregate := aggregates[normalized]
		if aggregate.name == "" {
			aggregate.name = strings.TrimSpace(group.GroupName)
		}
		if group.RateMultiplier > aggregate.rate {
			aggregate.rate = group.RateMultiplier
		}
		aggregate.keyCount = keyCounts[normalized]
		aggregates[normalized] = aggregate
	}
	if len(aggregates) == 0 && !groupsAuthoritative {
		for _, key := range keys {
			if key.ProviderSlug != "" && key.ProviderSlug != provider.Slug {
				continue
			}
			normalized := normalizeUpstreamGroupMatchName(key.GroupName)
			if normalized == "" {
				continue
			}
			aggregate := aggregates[normalized]
			if aggregate.name == "" {
				aggregate.name = strings.TrimSpace(key.GroupName)
			}
			if key.RateMultiplier > aggregate.rate {
				aggregate.rate = key.RateMultiplier
			}
			aggregate.keyCount++
			aggregates[normalized] = aggregate
		}
	}

	names := make([]string, 0, len(aggregates))
	for name := range aggregates {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]UpstreamGroupComparison, 0, len(names))
	for _, normalized := range names {
		aggregate := aggregates[normalized]
		item := UpstreamGroupComparison{
			ProviderSlug:      provider.Slug,
			ProviderName:      provider.Name,
			UpstreamGroupName: aggregate.name,
			UpstreamGroupKey:  normalized,
			UpstreamRate:      aggregate.rate,
			UpstreamKeyCount:  aggregate.keyCount,
		}
		if mappedID, ok := mappedLocalGroupIDs[normalized]; ok {
			if local, exists := localByID[mappedID]; exists {
				applyUpstreamGroupLocalMatch(&item, local, "manual")
			}
		}
		if !item.Matched {
			if _, ignored := ignoredGroupKeys[normalized]; ignored {
				item.MatchIgnored = true
			}
		}
		if !item.Matched && !item.MatchIgnored {
			if local, ok := localByName[normalized]; ok {
				applyUpstreamGroupLocalMatch(&item, local, "name")
			}
		}
		out = append(out, item)
	}
	return out
}

func applyUpstreamGroupLocalMatch(item *UpstreamGroupComparison, local Group, source string) {
	localID := local.ID
	localRate := local.RateMultiplier
	item.LocalGroupID = &localID
	item.LocalGroupName = local.Name
	item.LocalGroupPlatform = local.Platform
	item.LocalRate = &localRate
	item.Matched = true
	item.MatchSource = source
	item.NeedsRateIncrease = item.UpstreamRate > local.RateMultiplier
}

func (s *UpstreamManagementService) loadGroupMappings(ctx context.Context) ([]UpstreamGroupMappingRecord, error) {
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

func (s *UpstreamManagementService) saveGroupMappings(ctx context.Context, records []UpstreamGroupMappingRecord) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	records = normalizeUpstreamGroupMappings(records)
	raw, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("marshal upstream group mappings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamGroupMappings, string(raw)); err != nil {
		return fmt.Errorf("save upstream group mappings: %w", err)
	}
	return nil
}

func normalizeUpstreamGroupMappings(records []UpstreamGroupMappingRecord) []UpstreamGroupMappingRecord {
	out := make([]UpstreamGroupMappingRecord, 0, len(records))
	seen := map[string]int{}
	for _, record := range records {
		record.ProviderSlug = strings.TrimSpace(record.ProviderSlug)
		record.UpstreamGroupName = strings.TrimSpace(record.UpstreamGroupName)
		record.UpstreamGroupKey = normalizeUpstreamGroupMatchName(record.UpstreamGroupKey)
		if record.UpstreamGroupKey == "" {
			record.UpstreamGroupKey = normalizeUpstreamGroupMatchName(record.UpstreamGroupName)
		}
		if record.LocalGroupID > 0 {
			record.Ignored = false
		} else if record.Ignored {
			record.LocalGroupID = 0
		}
		if record.ProviderSlug == "" || record.UpstreamGroupKey == "" || (record.LocalGroupID <= 0 && !record.Ignored) {
			continue
		}
		key := record.ProviderSlug + "\x00" + record.UpstreamGroupKey
		if index, exists := seen[key]; exists {
			out[index] = record
			continue
		}
		seen[key] = len(out)
		out = append(out, record)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ProviderSlug == out[j].ProviderSlug {
			return out[i].UpstreamGroupKey < out[j].UpstreamGroupKey
		}
		return out[i].ProviderSlug < out[j].ProviderSlug
	})
	return out
}

func removeUpstreamGroupMapping(records []UpstreamGroupMappingRecord, providerSlug string, upstreamGroupKey string) []UpstreamGroupMappingRecord {
	providerSlug = strings.TrimSpace(providerSlug)
	upstreamGroupKey = normalizeUpstreamGroupMatchName(upstreamGroupKey)
	out := make([]UpstreamGroupMappingRecord, 0, len(records))
	for _, record := range normalizeUpstreamGroupMappings(records) {
		if record.ProviderSlug == providerSlug && record.UpstreamGroupKey == upstreamGroupKey {
			continue
		}
		out = append(out, record)
	}
	return out
}

func pruneMissingUpstreamGroupMappings(records []UpstreamGroupMappingRecord, providerSlug string, groups []UpstreamProviderGroup) ([]UpstreamGroupMappingRecord, bool) {
	providerSlug = strings.TrimSpace(providerSlug)
	validGroupKeys := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if group.ProviderSlug != "" && group.ProviderSlug != providerSlug {
			continue
		}
		key := normalizeUpstreamGroupMatchName(group.GroupName)
		if key != "" {
			validGroupKeys[key] = struct{}{}
		}
	}

	normalized := normalizeUpstreamGroupMappings(records)
	out := make([]UpstreamGroupMappingRecord, 0, len(normalized))
	changed := false
	for _, record := range normalized {
		if record.ProviderSlug == providerSlug {
			if _, ok := validGroupKeys[record.UpstreamGroupKey]; !ok {
				changed = true
				continue
			}
		}
		out = append(out, record)
	}
	return out, changed
}

func (s *UpstreamManagementService) loadRateFixRecords(ctx context.Context) ([]UpstreamGroupRateFixRecord, error) {
	if s == nil || s.settingRepo == nil {
		return []UpstreamGroupRateFixRecord{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamGroupRateFixRecords)
	if err != nil {
		if err == ErrSettingNotFound {
			return []UpstreamGroupRateFixRecord{}, nil
		}
		return nil, fmt.Errorf("load upstream group rate fix records: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []UpstreamGroupRateFixRecord{}, nil
	}
	var records []UpstreamGroupRateFixRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		return nil, infraerrors.InternalServer("UPSTREAM_GROUP_RATE_FIX_RECORDS_INVALID", "upstream group rate fix records are invalid")
	}
	return limitUpstreamGroupRateFixRecords(records), nil
}

func (s *UpstreamManagementService) prependRateFixRecords(ctx context.Context, records []UpstreamGroupRateFixRecord) ([]UpstreamGroupRateFixRecord, error) {
	existing, err := s.loadRateFixRecords(ctx)
	if err != nil {
		return nil, err
	}
	out := append([]UpstreamGroupRateFixRecord{}, records...)
	out = append(out, existing...)
	out = limitUpstreamGroupRateFixRecords(out)
	if s == nil || s.settingRepo == nil {
		return out, nil
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream group rate fix records: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamGroupRateFixRecords, string(raw)); err != nil {
		return nil, fmt.Errorf("save upstream group rate fix records: %w", err)
	}
	return out, nil
}

func upstreamGroupRateFixRecordKey(record UpstreamGroupRateFixRecord) string {
	return fmt.Sprintf(
		"%s-%d-%s-%s",
		record.ChangedAt.Format(time.RFC3339),
		record.GroupID,
		record.ProviderSlug,
		record.UpstreamGroupName,
	)
}

func limitUpstreamGroupRateFixRecords(records []UpstreamGroupRateFixRecord) []UpstreamGroupRateFixRecord {
	if len(records) <= 100 {
		return records
	}
	out := make([]UpstreamGroupRateFixRecord, 100)
	copy(out, records[:100])
	return out
}

func normalizeUpstreamGroupMatchName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	b.Grow(len(normalized))
	for _, r := range normalized {
		switch {
		case unicode.IsSpace(r):
			continue
		case r == '_' || r == '-':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
