package service

import (
	"math"
	"sort"
	"strings"
)

const (
	SupplierProviderGroupChangeTypeAdded       = "added"
	SupplierProviderGroupChangeTypeRemoved     = "removed"
	SupplierProviderGroupChangeTypeRateChanged = "rate_changed"
	SupplierProviderGroupChangeTypeNameChanged = "name_changed"

	SupplierGroupChangeEventType = "group_changed"

	supplierProviderGroupRateCompareEpsilon = 1e-9
)

// SupplierProviderGroupSnapshot 是供应商分组同步前后的可比较快照。
type SupplierProviderGroupSnapshot struct {
	UpstreamKey    string
	Name           string
	RateMultiplier float64
	Active         bool
}

// SupplierProviderGroupChange 表示一个供应商上游分组的单项变化。
type SupplierProviderGroupChange struct {
	ChangeType        string  `json:"change_type"`
	UpstreamKey       string  `json:"upstream_key"`
	OldName           string  `json:"old_name,omitempty"`
	NewName           string  `json:"new_name,omitempty"`
	OldRateMultiplier float64 `json:"old_rate_multiplier,omitempty"`
	NewRateMultiplier float64 `json:"new_rate_multiplier,omitempty"`
}

// SupplierProviderGroupChangeSummary 汇总一次分组同步中的所有变化。
type SupplierProviderGroupChangeSummary struct {
	Added       []SupplierProviderGroupChange `json:"added,omitempty"`
	Removed     []SupplierProviderGroupChange `json:"removed,omitempty"`
	RateChanged []SupplierProviderGroupChange `json:"rate_changed,omitempty"`
	NameChanged []SupplierProviderGroupChange `json:"name_changed,omitempty"`
}

// Empty 判断本次同步是否没有需要通知的分组变化。
func (s SupplierProviderGroupChangeSummary) Empty() bool {
	return len(s.Added) == 0 && len(s.Removed) == 0 && len(s.RateChanged) == 0 && len(s.NameChanged) == 0
}

// Count 返回本次同步的变化项数量。一个分组同时发生名称和倍率变化时计为两项。
func (s SupplierProviderGroupChangeSummary) Count() int {
	return len(s.Added) + len(s.Removed) + len(s.RateChanged) + len(s.NameChanged)
}

// BuildSupplierProviderGroupChangeSummary 对比同步前后的活动分组快照。
// 非活动旧分组重新出现时按新增处理，活动旧分组在完整返回中缺失时按删除处理。
func BuildSupplierProviderGroupChangeSummary(previous, current []SupplierProviderGroupSnapshot) SupplierProviderGroupChangeSummary {
	previousByKey := activeSupplierProviderGroupSnapshots(previous)
	currentByKey := activeSupplierProviderGroupSnapshots(current)

	changes := SupplierProviderGroupChangeSummary{}
	for key, currentGroup := range currentByKey {
		previousGroup, exists := previousByKey[key]
		if !exists {
			changes.Added = append(changes.Added, SupplierProviderGroupChange{
				ChangeType:        SupplierProviderGroupChangeTypeAdded,
				UpstreamKey:       key,
				NewName:           currentGroup.Name,
				NewRateMultiplier: currentGroup.RateMultiplier,
			})
			continue
		}

		oldName := strings.TrimSpace(previousGroup.Name)
		newName := strings.TrimSpace(currentGroup.Name)
		if !floatNearlyEqual(previousGroup.RateMultiplier, currentGroup.RateMultiplier) {
			changes.RateChanged = append(changes.RateChanged, SupplierProviderGroupChange{
				ChangeType:        SupplierProviderGroupChangeTypeRateChanged,
				UpstreamKey:       key,
				OldName:           oldName,
				NewName:           newName,
				OldRateMultiplier: previousGroup.RateMultiplier,
				NewRateMultiplier: currentGroup.RateMultiplier,
			})
		}
		if oldName != newName {
			changes.NameChanged = append(changes.NameChanged, SupplierProviderGroupChange{
				ChangeType:        SupplierProviderGroupChangeTypeNameChanged,
				UpstreamKey:       key,
				OldName:           oldName,
				NewName:           newName,
				OldRateMultiplier: previousGroup.RateMultiplier,
				NewRateMultiplier: currentGroup.RateMultiplier,
			})
		}
	}
	for key, previousGroup := range previousByKey {
		if _, exists := currentByKey[key]; exists {
			continue
		}
		changes.Removed = append(changes.Removed, SupplierProviderGroupChange{
			ChangeType:        SupplierProviderGroupChangeTypeRemoved,
			UpstreamKey:       key,
			OldName:           strings.TrimSpace(previousGroup.Name),
			OldRateMultiplier: previousGroup.RateMultiplier,
		})
	}

	sortSupplierProviderGroupChanges(changes.Added)
	sortSupplierProviderGroupChanges(changes.Removed)
	sortSupplierProviderGroupChanges(changes.RateChanged)
	sortSupplierProviderGroupChanges(changes.NameChanged)
	return changes
}

func activeSupplierProviderGroupSnapshots(items []SupplierProviderGroupSnapshot) map[string]SupplierProviderGroupSnapshot {
	result := make(map[string]SupplierProviderGroupSnapshot, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.UpstreamKey)
		if key == "" || !item.Active {
			continue
		}
		item.UpstreamKey = key
		item.Name = strings.TrimSpace(item.Name)
		result[key] = item
	}
	return result
}

func sortSupplierProviderGroupChanges(changes []SupplierProviderGroupChange) {
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].UpstreamKey < changes[j].UpstreamKey
	})
}

func floatNearlyEqual(left, right float64) bool {
	return math.Abs(left-right) <= supplierProviderGroupRateCompareEpsilon
}

// SupplierProviderGroupReplaceResult 是分组同步落库后的计数和变化结果。
type SupplierProviderGroupReplaceResult struct {
	Counts  SupplierSyncCounts
	Changes SupplierProviderGroupChangeSummary
}
