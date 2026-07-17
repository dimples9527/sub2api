package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

type UpstreamDashboardRange string

const (
	UpstreamDashboardRange24Hours UpstreamDashboardRange = "24h"
	UpstreamDashboardRange7Days   UpstreamDashboardRange = "7d"
)

type UpstreamDashboardSeverity string

const (
	UpstreamDashboardSeverityCritical UpstreamDashboardSeverity = "critical"
	UpstreamDashboardSeverityHigh     UpstreamDashboardSeverity = "high"
	UpstreamDashboardSeverityMedium   UpstreamDashboardSeverity = "medium"
	UpstreamDashboardSeverityLow      UpstreamDashboardSeverity = "low"
)

type UpstreamDashboardIssue struct {
	ID          string                    `json:"id"`
	Type        string                    `json:"type"`
	Source      string                    `json:"source"`
	Severity    UpstreamDashboardSeverity `json:"severity"`
	EntityKey   string                    `json:"entity_key"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	ImpactCount int                       `json:"impact_count"`
	Action      string                    `json:"action,omitempty"`
	TargetPath  string                    `json:"target_path,omitempty"`
	DetectedAt  time.Time                 `json:"detected_at"`
}

type UpstreamDashboardSummary struct {
	ProviderCount         int `json:"provider_count"`
	DisabledProviderCount int `json:"disabled_provider_count"`
	MatchedAccountCount   int `json:"matched_account_count"`
	PendingAccountCount   int `json:"pending_account_count"`
	RateRiskCount         int `json:"rate_risk_count"`
	ModelCount            int `json:"model_count"`
}

type UpstreamDashboardStability struct {
	RequestCount int64   `json:"request_count"`
	SuccessCount int64   `json:"success_count"`
	ErrorCount   int64   `json:"error_count"`
	SuccessRate  float64 `json:"success_rate"`
	ErrorRate    float64 `json:"error_rate"`
	P95LatencyMs int     `json:"p95_latency_ms"`
	HealthScore  int     `json:"health_score"`
}

type UpstreamDashboardCost struct {
	PeriodCost       float64  `json:"period_cost"`
	TotalBalance     float64  `json:"total_balance"`
	EstimatedDays    *float64 `json:"estimated_days,omitempty"`
	AnomalyProviders int      `json:"anomaly_providers"`
}

type UpstreamDashboardTask struct {
	Key            string     `json:"key"`
	Name           string     `json:"name"`
	Enabled        bool       `json:"enabled"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	LastRunStatus  string     `json:"last_run_status,omitempty"`
	LastRunMessage string     `json:"last_run_message,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	AffectedCount  int        `json:"affected_count"`
	SettingsPath   string     `json:"settings_path"`
}

type UpstreamDashboardProviderRanking struct {
	ProviderSlug string  `json:"provider_slug"`
	ProviderName string  `json:"provider_name"`
	Balance      float64 `json:"balance"`
	Cost         float64 `json:"cost"`
}

type UpstreamDashboardModelRanking struct {
	Model    string  `json:"model"`
	Requests int64   `json:"requests"`
	Cost     float64 `json:"cost"`
}

type UpstreamDashboardTrendPoint struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

type UpstreamDashboardWarning struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

type UpstreamDashboardResponse struct {
	Range            UpstreamDashboardRange             `json:"range"`
	Summary          UpstreamDashboardSummary           `json:"summary"`
	Stability        UpstreamDashboardStability         `json:"stability"`
	Cost             UpstreamDashboardCost              `json:"cost"`
	Issues           []UpstreamDashboardIssue           `json:"issues"`
	Tasks            []UpstreamDashboardTask            `json:"tasks"`
	ProviderRankings []UpstreamDashboardProviderRanking `json:"provider_rankings"`
	ModelRankings    []UpstreamDashboardModelRanking    `json:"model_rankings"`
	Trends           []UpstreamDashboardTrendPoint      `json:"trends"`
	Warnings         []UpstreamDashboardWarning         `json:"warnings,omitempty"`
	GeneratedAt      time.Time                          `json:"generated_at"`
}

type upstreamDashboardProviderSource interface {
	ListProviders(ctx context.Context) ([]UpstreamProviderConfig, error)
}

type upstreamDashboardGroupSource interface {
	CompareGroups(ctx context.Context) (UpstreamGroupCompareResult, error)
}

type upstreamDashboardSyncSource interface {
	Preview(ctx context.Context) (UpstreamAccountSyncResult, error)
	GetRateGuardConfig(ctx context.Context) (UpstreamAccountRateGuardConfig, error)
}

type upstreamDashboardBalanceSource interface {
	GetOverview(ctx context.Context, days int) (UpstreamBalanceConsumptionOverview, error)
}

type upstreamDashboardHealthSource interface {
	GetConfig(ctx context.Context) (UpstreamAccountHealthGuardConfig, error)
	ListRecords(ctx context.Context) ([]UpstreamAccountHealthGuardRunRecord, error)
}

type upstreamDashboardOpsSource interface {
	GetDashboardOverview(ctx context.Context, filter *OpsDashboardFilter) (*OpsDashboardOverview, error)
}

type upstreamDashboardUsageSource interface {
	GetModelStatsWithFiltersBySource(ctx context.Context, startTime, endTime time.Time, filters usagestats.UsageLogFilters, modelSource string) ([]usagestats.ModelStat, error)
}

type UpstreamDashboardService struct {
	providers upstreamDashboardProviderSource
	groups    upstreamDashboardGroupSource
	sync      upstreamDashboardSyncSource
	balance   upstreamDashboardBalanceSource
	health    upstreamDashboardHealthSource
	ops       upstreamDashboardOpsSource
	usage     upstreamDashboardUsageSource
	now       func() time.Time
}

func NewUpstreamDashboardService(
	providers upstreamDashboardProviderSource,
	groups upstreamDashboardGroupSource,
	syncSource upstreamDashboardSyncSource,
	balance upstreamDashboardBalanceSource,
	health upstreamDashboardHealthSource,
	ops upstreamDashboardOpsSource,
	usage upstreamDashboardUsageSource,
) *UpstreamDashboardService {
	return &UpstreamDashboardService{
		providers: providers,
		groups:    groups,
		sync:      syncSource,
		balance:   balance,
		health:    health,
		ops:       ops,
		usage:     usage,
		now:       time.Now,
	}
}

func (s *UpstreamDashboardService) Get(ctx context.Context, rangeValue UpstreamDashboardRange) (UpstreamDashboardResponse, error) {
	start, end, days, err := s.resolveRange(rangeValue)
	if err != nil {
		return UpstreamDashboardResponse{}, err
	}
	result := UpstreamDashboardResponse{
		Range:            rangeValue,
		Issues:           []UpstreamDashboardIssue{},
		Tasks:            []UpstreamDashboardTask{},
		ProviderRankings: []UpstreamDashboardProviderRanking{},
		ModelRankings:    []UpstreamDashboardModelRanking{},
		Trends:           []UpstreamDashboardTrendPoint{},
		GeneratedAt:      end,
	}

	s.collectProviders(ctx, &result)
	s.collectGroups(ctx, &result)
	s.collectSync(ctx, &result)
	s.collectBalance(ctx, days, &result)
	s.collectHealth(ctx, &result)
	s.collectOps(ctx, start, end, &result)
	s.collectUsage(ctx, start, end, &result)
	sortUpstreamDashboardIssues(result.Issues)
	sort.SliceStable(result.ProviderRankings, func(i, j int) bool { return result.ProviderRankings[i].Cost > result.ProviderRankings[j].Cost })
	sort.SliceStable(result.ModelRankings, func(i, j int) bool { return result.ModelRankings[i].Requests > result.ModelRankings[j].Requests })
	result.ProviderRankings = limitUpstreamDashboardSlice(result.ProviderRankings, 10)
	result.ModelRankings = limitUpstreamDashboardSlice(result.ModelRankings, 10)
	return result, nil
}

func (s *UpstreamDashboardService) resolveRange(value UpstreamDashboardRange) (time.Time, time.Time, int, error) {
	end := s.now().UTC()
	switch value {
	case UpstreamDashboardRange24Hours:
		return end.Add(-24 * time.Hour), end, 1, nil
	case UpstreamDashboardRange7Days:
		return end.Add(-7 * 24 * time.Hour), end, 7, nil
	default:
		return time.Time{}, time.Time{}, 0, fmt.Errorf("unsupported upstream dashboard range %q", value)
	}
}

func (s *UpstreamDashboardService) collectProviders(ctx context.Context, result *UpstreamDashboardResponse) {
	providers, err := s.providers.ListProviders(ctx)
	if err != nil {
		result.addWarning("providers", err)
		return
	}
	result.Summary.ProviderCount = len(providers)
	for _, provider := range providers {
		if provider.Enabled {
			continue
		}
		result.Summary.DisabledProviderCount++
		result.Issues = append(result.Issues, UpstreamDashboardIssue{
			ID: "provider-disabled:" + provider.Slug, Type: "provider_disabled", Source: "providers",
			Severity: UpstreamDashboardSeverityCritical, EntityKey: provider.Slug,
			Title: "上游供应商已停用", Description: provider.Name + " 当前处于停用状态",
			ImpactCount: 1, Action: "view_provider", TargetPath: "/admin/upstream-management/providers?status=disabled",
			DetectedAt: result.GeneratedAt,
		})
	}
}

func (s *UpstreamDashboardService) collectGroups(ctx context.Context, result *UpstreamDashboardResponse) {
	groups, err := s.groups.CompareGroups(ctx)
	if err != nil {
		result.addWarning("groups", err)
		return
	}
	for _, item := range groups.Items {
		if !item.NeedsRateIncrease {
			continue
		}
		result.Summary.RateRiskCount++
		result.Issues = append(result.Issues, UpstreamDashboardIssue{
			ID:   "group-rate-risk:" + item.ProviderSlug + ":" + item.UpstreamGroupKey,
			Type: "group_rate_risk", Source: "groups", Severity: UpstreamDashboardSeverityHigh,
			EntityKey: item.UpstreamGroupKey, Title: "分组倍率低于上游成本",
			Description: item.UpstreamGroupName + " 需要提高本地倍率", ImpactCount: item.UpstreamKeyCount,
			Action: "preview_rate_fix", TargetPath: "/admin/upstream-management/groups?rateRisk=true", DetectedAt: result.GeneratedAt,
		})
	}
}

func (s *UpstreamDashboardService) collectSync(ctx context.Context, result *UpstreamDashboardResponse) {
	preview, err := s.sync.Preview(ctx)
	if err != nil {
		result.addWarning("account_sync", err)
	} else {
		result.Summary.MatchedAccountCount = preview.Summary.MatchedAccountCount
		result.Summary.PendingAccountCount = preview.Summary.CreateCount + preview.Summary.UpdateCount + preview.Summary.ConflictCount
		if preview.Summary.ConflictCount > 0 {
			result.Issues = append(result.Issues, UpstreamDashboardIssue{
				ID: "account-sync-conflicts", Type: "account_sync_conflict", Source: "account_sync",
				Severity: UpstreamDashboardSeverityHigh, EntityKey: "conflicts", Title: "账号同步存在冲突",
				Description: fmt.Sprintf("%d 个上游账号无法自动匹配", preview.Summary.ConflictCount),
				ImpactCount: preview.Summary.ConflictCount, Action: "resolve_conflicts",
				TargetPath: "/admin/upstream-management/accounts?status=conflict", DetectedAt: result.GeneratedAt,
			})
		}
	}
	config, err := s.sync.GetRateGuardConfig(ctx)
	if err != nil {
		result.addWarning("account_rate_guard", err)
		return
	}
	result.Tasks = append(result.Tasks, upstreamDashboardTask(
		"account_rate_guard", "账号倍率保护", config.Enabled, config.LastRunAt, config.LastRunStatus,
		config.LastRunMessage, config.IntervalSeconds, 0, "/admin/upstream-management/accounts",
	))
}

func (s *UpstreamDashboardService) collectBalance(ctx context.Context, days int, result *UpstreamDashboardResponse) {
	overview, err := s.balance.GetOverview(ctx, days)
	if err != nil {
		result.addWarning("balance", err)
		return
	}
	for slug, summary := range overview.Summaries {
		result.Cost.TotalBalance += summary.CurrentBalance
		result.Cost.PeriodCost += summary.TodayConsumption
		name := summary.ProviderName
		if name == "" {
			name = slug
		}
		result.ProviderRankings = append(result.ProviderRankings, UpstreamDashboardProviderRanking{
			ProviderSlug: slug, ProviderName: name, Balance: summary.CurrentBalance, Cost: summary.TodayConsumption,
		})
		stale := summary.LastSnapshotAt == nil || result.GeneratedAt.Sub(*summary.LastSnapshotAt) > 30*time.Minute
		if summary.Anomaly || summary.LastSnapshotError != "" || stale {
			result.Cost.AnomalyProviders++
			description := summary.LastSnapshotError
			if description == "" && stale {
				description = "余额数据超过 30 分钟未更新"
			}
			result.Issues = append(result.Issues, UpstreamDashboardIssue{
				ID: "balance:" + slug, Type: "balance_anomaly", Source: "balance",
				Severity: UpstreamDashboardSeverityMedium, EntityKey: slug, Title: "上游余额数据异常",
				Description: description, ImpactCount: 1, Action: "view_balance",
				TargetPath: "/admin/upstream-management/providers?provider=" + slug, DetectedAt: result.GeneratedAt,
			})
		}
	}
	if result.Cost.PeriodCost > 0 && result.Cost.TotalBalance >= 0 {
		daysValue := math.Round(result.Cost.TotalBalance/result.Cost.PeriodCost*10) / 10
		result.Cost.EstimatedDays = &daysValue
	}
	for _, row := range overview.LocalDailyConsumptions {
		result.Trends = append(result.Trends, UpstreamDashboardTrendPoint{Date: row.Date, Cost: row.ActualCost})
	}
	result.Tasks = append(result.Tasks, upstreamDashboardTask(
		"balance_sampler", "余额采样", overview.Config.Enabled, overview.Config.LastRunAt,
		overview.Config.LastRunStatus, overview.Config.LastRunMessage, overview.Config.IntervalSeconds,
		result.Cost.AnomalyProviders, "/admin/upstream-management/providers",
	))
}

func (s *UpstreamDashboardService) collectHealth(ctx context.Context, result *UpstreamDashboardResponse) {
	config, err := s.health.GetConfig(ctx)
	if err != nil {
		result.addWarning("health_guard", err)
		return
	}
	records, err := s.health.ListRecords(ctx)
	if err != nil {
		result.addWarning("health_guard_records", err)
	}
	affected := 0
	if len(records) > 0 {
		record := records[0]
		affected = record.Summary.FailedCount + record.Summary.SlowCount
		if affected > 0 {
			result.Issues = append(result.Issues, UpstreamDashboardIssue{
				ID: "health:" + record.ID, Type: "health_guard_alert", Source: "health_guard",
				Severity: UpstreamDashboardSeverityHigh, EntityKey: record.ID, Title: "账号健康巡检发现异常",
				Description: fmt.Sprintf("失败 %d 个，慢响应 %d 个", record.Summary.FailedCount, record.Summary.SlowCount),
				ImpactCount: affected, Action: "view_health", TargetPath: "/admin/upstream-management/providers?status=health_error",
				DetectedAt: record.FinishedAt,
			})
		}
	}
	result.Tasks = append(result.Tasks, upstreamDashboardTask(
		"health_guard", "健康巡检", config.Enabled, config.LastRunAt, config.LastRunStatus,
		config.LastRunMessage, config.IntervalSeconds, affected, "/admin/upstream-management/providers",
	))
}

func (s *UpstreamDashboardService) collectOps(ctx context.Context, start, end time.Time, result *UpstreamDashboardResponse) {
	overview, err := s.ops.GetDashboardOverview(ctx, &OpsDashboardFilter{StartTime: start, EndTime: end})
	if err != nil {
		result.addWarning("ops", err)
		return
	}
	if overview == nil {
		return
	}
	result.Stability.RequestCount = overview.RequestCountTotal
	result.Stability.SuccessCount = overview.SuccessCount
	result.Stability.ErrorCount = overview.ErrorCountTotal
	result.Stability.SuccessRate = overview.SLA
	result.Stability.ErrorRate = overview.ErrorRate
	result.Stability.HealthScore = overview.HealthScore
	if overview.Duration.P95 != nil {
		result.Stability.P95LatencyMs = *overview.Duration.P95
	}
}

func (s *UpstreamDashboardService) collectUsage(ctx context.Context, start, end time.Time, result *UpstreamDashboardResponse) {
	stats, err := s.usage.GetModelStatsWithFiltersBySource(ctx, start, end, usagestats.UsageLogFilters{}, usagestats.ModelSourceUpstream)
	if err != nil {
		result.addWarning("model_usage", err)
		return
	}
	result.Summary.ModelCount = len(stats)
	for _, stat := range stats {
		model := strings.TrimSpace(stat.Model)
		if model == "" {
			continue
		}
		result.ModelRankings = append(result.ModelRankings, UpstreamDashboardModelRanking{
			Model: model, Requests: stat.Requests, Cost: stat.AccountCost,
		})
	}
}

func (r *UpstreamDashboardResponse) addWarning(source string, err error) {
	if err == nil {
		return
	}
	r.Warnings = append(r.Warnings, UpstreamDashboardWarning{Source: source, Message: err.Error()})
}

func upstreamDashboardTask(key, name string, enabled bool, lastRunAt *time.Time, status, message string, intervalSeconds, affected int, path string) UpstreamDashboardTask {
	task := UpstreamDashboardTask{
		Key: key, Name: name, Enabled: enabled, LastRunAt: lastRunAt, LastRunStatus: status,
		LastRunMessage: message, AffectedCount: affected, SettingsPath: path,
	}
	if enabled && lastRunAt != nil && intervalSeconds > 0 {
		next := lastRunAt.Add(time.Duration(intervalSeconds) * time.Second)
		task.NextRunAt = &next
	}
	return task
}

func limitUpstreamDashboardSlice[T any](items []T, limit int) []T {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func sortUpstreamDashboardIssues(issues []UpstreamDashboardIssue) {
	sort.SliceStable(issues, func(leftIndex, rightIndex int) bool {
		left := issues[leftIndex]
		right := issues[rightIndex]
		leftSeverity := upstreamDashboardSeverityWeight(left.Severity)
		rightSeverity := upstreamDashboardSeverityWeight(right.Severity)
		if leftSeverity != rightSeverity {
			return leftSeverity > rightSeverity
		}
		if left.ImpactCount != right.ImpactCount {
			return left.ImpactCount > right.ImpactCount
		}
		return left.DetectedAt.After(right.DetectedAt)
	})
}

func upstreamDashboardSeverityWeight(severity UpstreamDashboardSeverity) int {
	switch severity {
	case UpstreamDashboardSeverityCritical:
		return 4
	case UpstreamDashboardSeverityHigh:
		return 3
	case UpstreamDashboardSeverityMedium:
		return 2
	case UpstreamDashboardSeverityLow:
		return 1
	default:
		return 0
	}
}
