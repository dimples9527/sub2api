package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettingKeyUpstreamGroupRateFixRecords = "upstream_group_rate_fix_records"
	SettingKeyUpstreamGroupMappings       = "upstream_group_mappings"
)

type UpstreamManagementProviderSource interface {
	GetDefaultProvider(ctx context.Context) (UpstreamProviderConfig, error)
	FetchProviderKeys(ctx context.Context, slug string) ([]UpstreamProviderKey, []string, error)
}

type UpstreamGroupComparison struct {
	ProviderSlug      string   `json:"provider_slug"`
	ProviderName      string   `json:"provider_name"`
	UpstreamGroupName string   `json:"upstream_group_name"`
	UpstreamGroupKey  string   `json:"upstream_group_key"`
	UpstreamRate      float64  `json:"upstream_rate"`
	UpstreamKeyCount  int      `json:"upstream_key_count"`
	LocalGroupID      *int64   `json:"local_group_id,omitempty"`
	LocalGroupName    string   `json:"local_group_name,omitempty"`
	LocalRate         *float64 `json:"local_rate,omitempty"`
	Matched           bool     `json:"matched"`
	MatchSource       string   `json:"match_source,omitempty"`
	NeedsRateIncrease bool     `json:"needs_rate_increase"`
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
}

type UpstreamGroupMappingInput struct {
	UpstreamGroupName string `json:"upstream_group_name"`
	LocalGroupID      *int64 `json:"local_group_id"`
}

type UpstreamGroupMappingRecord struct {
	ProviderSlug      string    `json:"provider_slug"`
	UpstreamGroupName string    `json:"upstream_group_name"`
	UpstreamGroupKey  string    `json:"upstream_group_key"`
	LocalGroupID      int64     `json:"local_group_id"`
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
	return source.FetchDefaultModelSquare(ctx)
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
	}
	if err := s.saveGroupMappings(ctx, records); err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	return s.CompareGroups(ctx)
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
	localGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, fmt.Errorf("list local groups: %w", err)
	}
	mappings, err := s.loadGroupMappings(ctx)
	if err != nil {
		return UpstreamGroupCompareResult{}, err
	}
	items := compareUpstreamGroups(defaultProvider, keys, localGroups, mappings)
	return UpstreamGroupCompareResult{
		DefaultProvider: redactUpstreamProvider(defaultProvider),
		Items:           items,
		Warnings:        warnings,
		Records:         []UpstreamGroupRateFixRecord{},
	}, nil
}

func compareUpstreamGroups(provider UpstreamProviderConfig, keys []UpstreamProviderKey, localGroups []Group, mappings []UpstreamGroupMappingRecord) []UpstreamGroupComparison {
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
	for _, mapping := range mappings {
		if mapping.ProviderSlug != provider.Slug {
			continue
		}
		key := normalizeUpstreamGroupMatchName(mapping.UpstreamGroupKey)
		if key == "" {
			key = normalizeUpstreamGroupMatchName(mapping.UpstreamGroupName)
		}
		if key == "" || mapping.LocalGroupID <= 0 {
			continue
		}
		mappedLocalGroupIDs[key] = mapping.LocalGroupID
	}

	type upstreamGroupAggregate struct {
		name     string
		rate     float64
		keyCount int
	}
	aggregates := map[string]upstreamGroupAggregate{}
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
		if record.ProviderSlug == "" || record.UpstreamGroupKey == "" || record.LocalGroupID <= 0 {
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

func limitUpstreamGroupRateFixRecords(records []UpstreamGroupRateFixRecord) []UpstreamGroupRateFixRecord {
	if len(records) <= 100 {
		return records
	}
	out := make([]UpstreamGroupRateFixRecord, 100)
	copy(out, records[:100])
	return out
}

func normalizeUpstreamGroupMatchName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}
