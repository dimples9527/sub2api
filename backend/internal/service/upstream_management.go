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

const SettingKeyUpstreamGroupRateFixRecords = "upstream_group_rate_fix_records"

type UpstreamManagementProviderSource interface {
	GetDefaultProvider(ctx context.Context) (UpstreamProviderConfig, error)
	FetchProviderKeys(ctx context.Context, slug string) ([]UpstreamProviderKey, []string, error)
}

type UpstreamGroupComparison struct {
	ProviderSlug      string   `json:"provider_slug"`
	ProviderName      string   `json:"provider_name"`
	UpstreamGroupName string   `json:"upstream_group_name"`
	UpstreamRate      float64  `json:"upstream_rate"`
	UpstreamKeyCount  int      `json:"upstream_key_count"`
	LocalGroupID      *int64   `json:"local_group_id,omitempty"`
	LocalGroupName    string   `json:"local_group_name,omitempty"`
	LocalRate         *float64 `json:"local_rate,omitempty"`
	Matched           bool     `json:"matched"`
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
	items := compareUpstreamGroups(defaultProvider, keys, localGroups)
	return UpstreamGroupCompareResult{
		DefaultProvider: redactUpstreamProvider(defaultProvider),
		Items:           items,
		Warnings:        warnings,
		Records:         []UpstreamGroupRateFixRecord{},
	}, nil
}

func compareUpstreamGroups(provider UpstreamProviderConfig, keys []UpstreamProviderKey, localGroups []Group) []UpstreamGroupComparison {
	localByName := make(map[string]Group, len(localGroups))
	for _, group := range localGroups {
		normalized := normalizeUpstreamGroupMatchName(group.Name)
		if normalized == "" {
			continue
		}
		if _, exists := localByName[normalized]; !exists {
			localByName[normalized] = group
		}
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
			UpstreamRate:      aggregate.rate,
			UpstreamKeyCount:  aggregate.keyCount,
		}
		if local, ok := localByName[normalized]; ok {
			localID := local.ID
			localRate := local.RateMultiplier
			item.LocalGroupID = &localID
			item.LocalGroupName = local.Name
			item.LocalRate = &localRate
			item.Matched = true
			item.NeedsRateIncrease = aggregate.rate > local.RateMultiplier
		}
		out = append(out, item)
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
