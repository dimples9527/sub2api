package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SupplierProviderMonitorSyncResult struct {
	ProcessedCount int                               `json:"processed_count"`
	SuccessCount   int                               `json:"success_count"`
	FailedCount    int                               `json:"failed_count"`
	SkippedCount   int                               `json:"skipped_count"`
	Items          []SupplierProviderMonitorSyncItem `json:"items"`
}

type SupplierProviderMonitorSyncItem struct {
	ProviderID       int64     `json:"provider_id"`
	ProviderName     string    `json:"provider_name"`
	ProviderType     string    `json:"provider_type"`
	LocalAccountID   int64     `json:"local_account_id,omitempty"`
	LocalAccountName string    `json:"local_account_name,omitempty"`
	LocalGroupIDs    []int64   `json:"local_group_ids,omitempty"`
	LocalGroupNames  []string  `json:"local_group_names,omitempty"`
	UpstreamKey      string    `json:"upstream_key,omitempty"`
	UpstreamName     string    `json:"upstream_name"`
	MonitorProvider  string    `json:"monitor_provider,omitempty"`
	PrimaryModel     string    `json:"primary_model,omitempty"`
	Status           string    `json:"status"`
	RawStatus        string    `json:"raw_status,omitempty"`
	LatencyMS        int64     `json:"latency_ms"`
	PingLatencyMS    int64     `json:"ping_latency_ms,omitempty"`
	Availability7D   float64   `json:"availability_7d,omitempty"`
	CheckedAt        time.Time `json:"checked_at"`
	Message          string    `json:"message,omitempty"`
}

type SupplierProviderMonitorSyncer interface {
	SyncMonitorsEnabled(ctx context.Context, trigger string) (SupplierProviderMonitorSyncResult, error)
}

func (s *SupplierProviderSyncService) SyncMonitorsEnabled(ctx context.Context, trigger string) (SupplierProviderMonitorSyncResult, error) {
	ctx = WithSupplierProviderAuthSource(ctx, SupplierProviderAuthSourceSync)
	enabled := true
	providers, _, err := s.providerRepo.List(ctx, SupplierProviderListParams{Enabled: &enabled, Page: 1, PageSize: 1000})
	if err != nil {
		return SupplierProviderMonitorSyncResult{}, fmt.Errorf("list enabled supplier providers: %w", err)
	}
	result := SupplierProviderMonitorSyncResult{ProcessedCount: len(providers), Items: make([]SupplierProviderMonitorSyncItem, 0)}
	for _, provider := range providers {
		items, syncErr := s.syncProviderMonitors(ctx, provider, trigger)
		result.Items = append(result.Items, items...)
		if syncErr != nil {
			if errorsIsSyncConflict(syncErr) {
				result.SkippedCount++
				continue
			}
			result.FailedCount++
			continue
		}
		if len(items) == 0 {
			result.SkippedCount++
			continue
		}
		result.SuccessCount++
	}
	return result, nil
}

func errorsIsSyncConflict(err error) bool {
	return errors.Is(err, ErrSupplierProviderSyncConflict)
}

func (s *SupplierProviderSyncService) syncProviderMonitors(ctx context.Context, provider *SupplierProvider, trigger string) ([]SupplierProviderMonitorSyncItem, error) {
	_ = trigger
	if provider == nil {
		return nil, ErrSupplierProviderInvalid
	}
	if normalizeSupplierProviderType(provider.ProviderType) != SupplierProviderTypeSub2API {
		return nil, nil
	}
	monitorClient, ok := s.remote.(SupplierProviderRemoteMonitorClient)
	if !ok {
		return []SupplierProviderMonitorSyncItem{{ProviderID: provider.ID, ProviderName: provider.Name, ProviderType: provider.ProviderType, Status: SupplierSyncStatusFailed, Message: "supplier monitor client is required"}}, fmt.Errorf("supplier monitor client is required")
	}
	var out []SupplierProviderMonitorSyncItem
	_, err := s.syncWithLock(ctx, provider.ID, func(locked *SupplierProvider) (SupplierProviderSyncResult, error) {
		startedAt := time.Now()
		run := &SupplierProviderSyncRun{ProviderID: locked.ID, SyncScope: SupplierSyncScopeMonitor, TriggerSource: normalizeSupplierSyncTrigger(trigger), Status: SupplierSyncStatusRunning, StartedAt: startedAt}
		if err := s.dataRepo.CreateSyncRun(ctx, run); err != nil {
			return SupplierProviderSyncResult{ProviderID: locked.ID, ProviderName: locked.Name, Scope: SupplierSyncScopeMonitor, Status: SupplierSyncStatusFailed, StartedAt: startedAt, FinishedAt: time.Now()}, err
		}
		finish := func(status string, counts SupplierSyncCounts, message string) {
			finishedAt := time.Now()
			run.Status = status
			run.Counts = counts
			run.ErrorMessage = message
			run.FinishedAt = &finishedAt
			_ = s.dataRepo.FinishSyncRun(ctx, run)
		}
		monitors, fetchErr := monitorClient.FetchMonitorItems(ctx, locked, s.providerPassword(locked))
		if fetchErr != nil {
			message := fetchErr.Error()
			if IsSupplierProviderAuthFailure(fetchErr) {
				message = supplierProviderAuthFailureDisableMessage
				fetchErr = supplierProviderAuthFailureWithDisableError(fetchErr, s.disableProviderAfterAuthFailure(ctx, locked.ID, time.Now()))
			}
			out = append(out, SupplierProviderMonitorSyncItem{ProviderID: locked.ID, ProviderName: locked.Name, ProviderType: locked.ProviderType, Status: SupplierSyncStatusFailed, Message: message})
			finish(SupplierSyncStatusFailed, SupplierSyncCounts{}, message)
			return SupplierProviderSyncResult{ProviderID: locked.ID, ProviderName: locked.Name, Scope: SupplierSyncScopeMonitor, Status: SupplierSyncStatusFailed, Message: message, StartedAt: startedAt, FinishedAt: time.Now()}, fetchErr
		}
		accountMatches, matchErr := s.monitorAccountMatches(ctx, locked)
		if matchErr != nil {
			finish(SupplierSyncStatusFailed, SupplierSyncCounts{CheckedCount: len(monitors)}, matchErr.Error())
			return SupplierProviderSyncResult{ProviderID: locked.ID, ProviderName: locked.Name, Scope: SupplierSyncScopeMonitor, Status: SupplierSyncStatusFailed, Message: matchErr.Error(), StartedAt: startedAt, FinishedAt: time.Now()}, matchErr
		}
		counts := SupplierSyncCounts{CheckedCount: len(monitors)}
		updatedCount := 0
		for _, monitor := range monitors {
			matched := accountMatches[normalizeSupplierMonitorMatchKey(monitor.Name)]
			if matched.localAccountID == 0 {
				counts.SkippedCount++
			}
			points := monitor.Timeline
			if len(points) == 0 {
				points = []SupplierProviderMonitorPoint{{Status: monitor.PrimaryStatus, LatencyMS: monitor.PrimaryLatencyMS, PingLatencyMS: monitor.PrimaryPingLatencyMS, CheckedAt: time.Now().UTC()}}
			}
			for _, point := range points {
				status := normalizeSupplierMonitorStatus(point.Status)
				if point.CheckedAt.IsZero() {
					point.CheckedAt = time.Now().UTC()
				}
				out = append(out, SupplierProviderMonitorSyncItem{ProviderID: locked.ID, ProviderName: locked.Name, ProviderType: locked.ProviderType, LocalAccountID: matched.localAccountID, LocalAccountName: matched.localAccountName, LocalGroupIDs: matched.localGroupIDs, LocalGroupNames: matched.localGroupNames, UpstreamKey: monitor.Key, UpstreamName: monitor.Name, MonitorProvider: monitor.Provider, PrimaryModel: monitor.PrimaryModel, Status: status, RawStatus: point.Status, LatencyMS: point.LatencyMS, PingLatencyMS: point.PingLatencyMS, Availability7D: monitor.Availability7D, CheckedAt: point.CheckedAt})
				updatedCount++
			}
		}
		counts.UpdatedCount = updatedCount
		finish(SupplierSyncStatusSuccess, counts, "")
		return SupplierProviderSyncResult{ProviderID: locked.ID, ProviderName: locked.Name, Scope: SupplierSyncScopeMonitor, Status: SupplierSyncStatusSuccess, Counts: counts, StartedAt: startedAt, FinishedAt: time.Now()}, nil
	})
	return out, err
}

type supplierMonitorAccountMatch struct {
	localAccountID   int64
	localAccountName string
	localGroupIDs    []int64
	localGroupNames  []string
}

func supplierMonitorAccountMatchFromAccount(account SupplierProviderAccount) supplierMonitorAccountMatch {
	match := supplierMonitorAccountMatch{
		localAccountName: account.LocalAccountName,
		localGroupIDs:    make([]int64, 0, len(account.BindingGroups)),
		localGroupNames:  make([]string, 0, len(account.BindingGroups)),
	}
	if account.LocalAccountID != nil {
		match.localAccountID = *account.LocalAccountID
	}
	for _, group := range account.BindingGroups {
		if group.ID <= 0 {
			continue
		}
		match.localGroupIDs = append(match.localGroupIDs, group.ID)
		match.localGroupNames = append(match.localGroupNames, group.Name)
	}
	return match
}

func (s *SupplierProviderSyncService) monitorAccountMatches(ctx context.Context, provider *SupplierProvider) (map[string]supplierMonitorAccountMatch, error) {
	matches := make(map[string]supplierMonitorAccountMatch)
	if provider == nil {
		return matches, nil
	}
	accounts, err := s.dataRepo.ListAccounts(ctx, SupplierProviderDataListParams{ProviderID: provider.ID, Page: 1, PageSize: 10000})
	if err != nil {
		return nil, fmt.Errorf("list supplier provider account matches: %w", err)
	}
	for _, account := range accounts.Items {
		if account.LocalAccountID == nil || *account.LocalAccountID <= 0 {
			continue
		}
		match := supplierMonitorAccountMatchFromAccount(account)
		for _, key := range supplierMonitorAccountMatchKeys(provider, account) {
			normalized := normalizeSupplierMonitorMatchKey(key)
			if normalized == "" {
				continue
			}
			if _, exists := matches[normalized]; !exists {
				matches[normalized] = match
			}
		}
	}
	return matches, nil
}

func supplierMonitorAccountMatchKeys(provider *SupplierProvider, account SupplierProviderAccount) []string {
	keys := []string{account.Name, provider.AccountNamePrefix + account.Name, account.UpstreamKey, account.LocalAccountName}
	for _, value := range []string{account.Name, account.LocalAccountName} {
		for _, prefix := range []string{provider.AccountNamePrefix, provider.Name} {
			if suffix := supplierMonitorNameWithoutPrefix(value, prefix); suffix != "" {
				keys = append(keys, suffix)
			}
		}
	}
	for _, key := range append([]string(nil), keys...) {
		if alias := supplierMonitorReorderedNameAlias(key); alias != "" {
			keys = append(keys, alias)
		}
	}
	return keys
}

// supplierMonitorReorderedNameAlias 生成中英文片段重排别名，例如破甲gpt1可匹配gpt破甲1。
func supplierMonitorReorderedNameAlias(value string) string {
	value = normalizeSupplierMonitorMatchKey(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	index := 0
	take := func(predicate func(rune) bool) string {
		start := index
		for index < len(runes) && predicate(runes[index]) {
			index++
		}
		return string(runes[start:index])
	}
	isHan := func(r rune) bool { return r >= '\u4e00' && r <= '\u9fff' }
	isLatin := func(r rune) bool { return r >= 'a' && r <= 'z' }
	isDigit := func(r rune) bool { return r >= '0' && r <= '9' }

	if isHan(runes[0]) {
		han := take(isHan)
		latin := take(isLatin)
		digits := take(isDigit)
		if index == len(runes) && han != "" && latin != "" {
			return latin + han + digits
		}
		return ""
	}

	if isLatin(runes[0]) {
		latin := take(isLatin)
		han := take(isHan)
		digits := take(isDigit)
		if index == len(runes) && han != "" && latin != "" {
			return han + latin + digits
		}
	}
	return ""
}

func supplierMonitorNameWithoutPrefix(value, prefix string) string {
	value = strings.TrimSpace(value)
	prefix = strings.TrimSpace(prefix)
	if value == "" || prefix == "" || !strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
		return ""
	}
	return strings.TrimLeft(strings.TrimSpace(value[len(prefix):]), "-_/:. ")
}
func normalizeSupplierMonitorMatchKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= '\u4e00' && r <= '\u9fff') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeSupplierMonitorStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "operational", SupplierAccountHealthGuardStatusHealthy:
		return SupplierAccountHealthGuardStatusHealthy
	case "degraded", SupplierAccountHealthGuardStatusSlow:
		return SupplierAccountHealthGuardStatusSlow
	case "error", SupplierAccountHealthGuardStatusFailed:
		return SupplierAccountHealthGuardStatusFailed
	default:
		return SupplierAccountHealthGuardStatusUnavailable
	}
}
