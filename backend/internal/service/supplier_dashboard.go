package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"
	"time"
)

type SupplierDashboardRange string

const (
	SupplierDashboardRange1Hour   SupplierDashboardRange = "1h"
	SupplierDashboardRange6Hours  SupplierDashboardRange = "6h"
	SupplierDashboardRange24Hours SupplierDashboardRange = "24h"
	SupplierDashboardRange7Days   SupplierDashboardRange = "7d"
	SupplierDashboardRange30Days  SupplierDashboardRange = "30d"
)

type SupplierDashboardSeverity string

const (
	SupplierDashboardSeverityCritical SupplierDashboardSeverity = "critical"
	SupplierDashboardSeverityHigh     SupplierDashboardSeverity = "high"
	SupplierDashboardSeverityMedium   SupplierDashboardSeverity = "medium"
	SupplierDashboardSeverityLow      SupplierDashboardSeverity = "low"
)

type SupplierDashboardWarning struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

type SupplierDashboardService struct {
	detail     SupplierDashboardDetailRepository
	thresholds supplierDashboardMetricThresholdSource
	now        func() time.Time
}

func NewSupplierDashboardService(detail SupplierDashboardDetailRepository, thresholds supplierDashboardMetricThresholdSource) *SupplierDashboardService {
	return &SupplierDashboardService{detail: detail, thresholds: thresholds, now: time.Now}
}

func (s *SupplierDashboardService) resolveRange(value SupplierDashboardRange) (time.Time, time.Time, int, error) {
	end := s.now().UTC()
	switch value {
	case SupplierDashboardRange1Hour:
		return end.Add(-time.Hour), end, 1, nil
	case SupplierDashboardRange6Hours:
		return end.Add(-6 * time.Hour), end, 6, nil
	case SupplierDashboardRange24Hours:
		return end.Add(-24 * time.Hour), end, 1, nil
	case SupplierDashboardRange7Days:
		return end.Add(-7 * 24 * time.Hour), end, 7, nil
	case SupplierDashboardRange30Days:
		return end.Add(-30 * 24 * time.Hour), end, 30, nil
	default:
		return time.Time{}, time.Time{}, 0, fmt.Errorf("unsupported supplier dashboard range %q", value)
	}
}

type SupplierDashboardRiskType string

const (
	SupplierDashboardRiskTypeAll       SupplierDashboardRiskType = "all"
	SupplierDashboardRiskTypeCritical  SupplierDashboardRiskType = "critical"
	SupplierDashboardRiskTypeTraffic   SupplierDashboardRiskType = "traffic"
	SupplierDashboardRiskTypeRateUp    SupplierDashboardRiskType = "rate_up"
	SupplierDashboardRiskTypeNotLowest SupplierDashboardRiskType = "not_lowest"
	SupplierDashboardRiskTypeBalance   SupplierDashboardRiskType = "balance"
	SupplierDashboardRiskTypeSync      SupplierDashboardRiskType = "sync"
	SupplierDashboardRiskTypeTask      SupplierDashboardRiskType = "task"
)

type SupplierDashboardRateView string

const (
	SupplierDashboardRateViewRisk    SupplierDashboardRateView = "risk"
	SupplierDashboardRateViewChanged SupplierDashboardRateView = "changed"
	SupplierDashboardRateViewAll     SupplierDashboardRateView = "all"
)

type SupplierDashboardProviderStatus string

const (
	SupplierDashboardProviderStatusHealthy  SupplierDashboardProviderStatus = "healthy"
	SupplierDashboardProviderStatusWarning  SupplierDashboardProviderStatus = "warning"
	SupplierDashboardProviderStatusHighRisk SupplierDashboardProviderStatus = "high_risk"
	SupplierDashboardProviderStatusDisabled SupplierDashboardProviderStatus = "disabled"
	SupplierDashboardProviderStatusUnknown  SupplierDashboardProviderStatus = "unknown"
)

type SupplierDashboardComparisonStatus string

const (
	SupplierDashboardComparisonStatusLowest               SupplierDashboardComparisonStatus = "lowest"
	SupplierDashboardComparisonStatusTiedLowest           SupplierDashboardComparisonStatus = "tied_lowest"
	SupplierDashboardComparisonStatusNotLowest            SupplierDashboardComparisonStatus = "not_lowest"
	SupplierDashboardComparisonStatusMissingGroup         SupplierDashboardComparisonStatus = "missing_group"
	SupplierDashboardComparisonStatusInsufficientAccounts SupplierDashboardComparisonStatus = "insufficient_accounts"
	SupplierDashboardComparisonStatusUnknown              SupplierDashboardComparisonStatus = "unknown"
)

type SupplierDashboardAccountSnapshot struct {
	AccountID             int64      `json:"account_id"`
	AccountName           string     `json:"account_name"`
	ProviderSlug          string     `json:"provider_slug"`
	ProviderName          string     `json:"provider_name"`
	ProviderEnabled       bool       `json:"provider_enabled"`
	AccountEnabled        bool       `json:"account_enabled"`
	GroupKey              string     `json:"group_key"`
	GroupName             string     `json:"group_name"`
	ProviderRiskLevel     string     `json:"provider_risk_level"`
	ProviderRiskUpdatedAt *time.Time `json:"provider_risk_updated_at"`
	AccountStatus         string     `json:"account_status"`
	RateSyncStatus        string     `json:"rate_sync_status"`
	BalanceSyncStatus     string     `json:"balance_sync_status"`
	BalanceSyncedAt       *time.Time `json:"balance_synced_at"`
	TaskStatus            string     `json:"task_status"`
	TaskReason            string     `json:"task_reason"`
	TaskFinishedAt        *time.Time `json:"task_finished_at"`
	SuccessCount          *int64     `json:"success_count"`
	ErrorCount            *int64     `json:"error_count"`
	CurrentRate           *float64   `json:"current_rate"`
	PreviousRate          *float64   `json:"previous_rate"`
	SnapshotCount         int        `json:"snapshot_count"`
	RateChangeOld         *float64   `json:"rate_change_old"`
	RateChangeNew         *float64   `json:"rate_change_new"`
	RateChangeCount       int        `json:"rate_change_count"`
	RateChangedAt         *time.Time `json:"rate_changed_at"`
	Balance               *float64   `json:"balance"`
	EstimatedDays         *float64   `json:"estimated_days"`
	PeriodCost            *float64   `json:"period_cost"`
	LastRateSyncedAt      *time.Time `json:"last_rate_synced_at"`
	ObservedAt            time.Time  `json:"observed_at"`
}

type SupplierDashboardRateSnapshot struct {
	AccountID        int64      `json:"account_id"`
	AccountName      string     `json:"account_name"`
	ProviderSlug     string     `json:"provider_slug"`
	ProviderName     string     `json:"provider_name"`
	ProviderEnabled  bool       `json:"provider_enabled"`
	AccountEnabled   bool       `json:"account_enabled"`
	GroupKey         string     `json:"group_key"`
	GroupName        string     `json:"group_name"`
	CurrentRate      *float64   `json:"current_rate"`
	PreviousRate     *float64   `json:"previous_rate"`
	SnapshotCount    int        `json:"snapshot_count"`
	RateChangeOld    *float64   `json:"rate_change_old"`
	RateChangeNew    *float64   `json:"rate_change_new"`
	RateChangeCount  int        `json:"rate_change_count"`
	RateChangedAt    *time.Time `json:"rate_changed_at"`
	SuccessCount     *int64     `json:"success_count"`
	ErrorCount       *int64     `json:"error_count"`
	PeriodCost       *float64   `json:"period_cost"`
	LastRateSyncedAt *time.Time `json:"last_rate_synced_at"`
	ObservedAt       time.Time  `json:"observed_at"`
}

type SupplierDashboardProviderSnapshot struct {
	ProviderSlug            string     `json:"provider_slug"`
	ProviderName            string     `json:"provider_name"`
	Enabled                 bool       `json:"enabled"`
	DataComplete            bool       `json:"data_complete"`
	ProviderRiskLevel       string     `json:"provider_risk_level"`
	SyncStatus              string     `json:"sync_status"`
	GroupSyncStatus         string     `json:"group_sync_status"`
	BalanceSyncStatus       string     `json:"balance_sync_status"`
	RateRiskCount           int        `json:"rate_risk_count"`
	EnabledAccountCount     int        `json:"enabled_account_count"`
	SchedulableAccountCount int        `json:"schedulable_account_count"`
	SuccessCount            *int64     `json:"success_count"`
	ErrorCount              *int64     `json:"error_count"`
	Balance                 *float64   `json:"balance"`
	EstimatedDays           *float64   `json:"estimated_days"`
	PeriodCost              *float64   `json:"period_cost"`
	LastSyncedAt            *time.Time `json:"last_synced_at"`
}

type SupplierDashboardTrafficSnapshot struct {
	Time         string
	AccountID    int64
	AccountName  string
	ProviderSlug string
	ProviderName string
	GroupKey     string
	GroupName    string
	Requests     int64
	Tokens       int64
}

type SupplierDashboardProfitSnapshot struct {
	AccountID    int64
	AccountName  string
	ProviderSlug string
	ProviderName string
	GroupKey     string
	GroupName    string
	Requests     int64
	Tokens       int64
	UserCost     float64
	ActualCost   float64
}

type SupplierDashboardHealthSnapshot struct {
	AccountID    int64
	AccountName  string
	ProviderSlug string
	ProviderName string
	GroupKey     string
	GroupName    string
	Time         string
	Status       string
}

type SupplierDashboardDetailRepository interface {
	ListDashboardAccounts(context.Context, time.Time, time.Time, string, string) ([]SupplierDashboardAccountSnapshot, error)
	ListDashboardRates(context.Context, time.Time, time.Time, string, string) ([]SupplierDashboardRateSnapshot, error)
	ListDashboardProviders(context.Context, time.Time, time.Time) ([]SupplierDashboardProviderSnapshot, error)
	ListDashboardAccountTraffic(context.Context, time.Time, time.Time, string, string) ([]SupplierDashboardTrafficSnapshot, error)
	ListDashboardAccountProfit(context.Context, time.Time, time.Time, string, string, int) ([]SupplierDashboardProfitSnapshot, error)
	ListDashboardAccountHealth(context.Context, time.Time, time.Time, string, string, int) ([]SupplierDashboardHealthSnapshot, error)
}

type SupplierDashboardAccountsQuery struct {
	Range        SupplierDashboardRange    `json:"range"`
	RiskType     SupplierDashboardRiskType `json:"risk_type"`
	ProviderSlug string                    `json:"provider_slug"`
	GroupKey     string                    `json:"group_key"`
	Page         int                       `json:"page"`
	PageSize     int                       `json:"page_size"`
}
type SupplierDashboardRatesQuery struct {
	Range            SupplierDashboardRange            `json:"range"`
	View             SupplierDashboardRateView         `json:"view"`
	ComparisonStatus SupplierDashboardComparisonStatus `json:"comparison_status"`
	ProviderSlug     string                            `json:"provider_slug"`
	GroupKey         string                            `json:"group_key"`
	Page             int                               `json:"page"`
	PageSize         int                               `json:"page_size"`
}
type SupplierDashboardProvidersQuery struct {
	Range    SupplierDashboardRange          `json:"range"`
	Status   SupplierDashboardProviderStatus `json:"status"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"page_size"`
}

type SupplierDashboardTrafficQuery struct {
	Range        SupplierDashboardRange `json:"range"`
	ProviderSlug string                 `json:"provider_slug"`
	GroupKey     string                 `json:"group_key"`
}
type SupplierDashboardProfitQuery struct {
	Range        SupplierDashboardRange `json:"range"`
	ProviderSlug string                 `json:"provider_slug"`
	GroupKey     string                 `json:"group_key"`
	Limit        int                    `json:"limit"`
}
type SupplierDashboardAccountHealthQuery struct {
	Range         SupplierDashboardRange `json:"range"`
	ProviderSlug  string                 `json:"provider_slug"`
	GroupKey      string                 `json:"group_key"`
	Limit         int                    `json:"limit"`
	Buckets       int                    `json:"buckets"`
	BucketHours   int                    `json:"bucket_hours"`
	BucketMinutes int                    `json:"bucket_minutes"`
}

type SupplierDashboardAccountItem struct {
	AccountID          int64                       `json:"account_id"`
	AccountName        string                      `json:"account_name"`
	ProviderSlug       string                      `json:"provider_slug"`
	ProviderName       string                      `json:"provider_name"`
	GroupKey           string                      `json:"group_key"`
	GroupName          string                      `json:"group_name"`
	Severity           SupplierDashboardSeverity   `json:"severity"`
	RiskTypes          []SupplierDashboardRiskType `json:"risk_types"`
	RequestCount       *int64                      `json:"request_count"`
	SuccessRate        *float64                    `json:"success_rate"`
	CurrentRate        *float64                    `json:"current_rate"`
	LowestRate         *float64                    `json:"lowest_rate"`
	RateDeltaPercent   *float64                    `json:"rate_delta_percent"`
	Balance            *float64                    `json:"balance"`
	BalanceCurrency    *string                     `json:"balance_currency"`
	EstimatedDays      *float64                    `json:"estimated_days"`
	Status             string                      `json:"status"`
	Reason             string                      `json:"reason"`
	PeriodCost         *float64                    `json:"period_cost"`
	EstimatedExtraCost *float64                    `json:"estimated_extra_cost"`
	TrafficImpact      int64                       `json:"traffic_impact"`
	DetectedAt         time.Time                   `json:"detected_at"`
	TargetPath         string                      `json:"target_path"`
}
type SupplierDashboardRateItem struct {
	ProviderSlug        string                            `json:"provider_slug"`
	ProviderName        string                            `json:"provider_name"`
	GroupKey            string                            `json:"group_key"`
	GroupName           string                            `json:"group_name"`
	EnabledAccountCount int                               `json:"enabled_account_count"`
	CurrentAccountID    int64                             `json:"current_account_id"`
	CurrentAccountName  string                            `json:"current_account_name"`
	CurrentRate         *float64                          `json:"current_rate"`
	LowestRate          *float64                          `json:"lowest_rate"`
	LowestAccountIDs    []int64                           `json:"lowest_account_ids"`
	LowestAccountNames  []string                          `json:"lowest_account_names"`
	RateDeltaPercent    *float64                          `json:"rate_delta_percent"`
	EstimatedExtraCost  *float64                          `json:"estimated_extra_cost"`
	CostCurrency        *string                           `json:"cost_currency"`
	ComparisonStatus    SupplierDashboardComparisonStatus `json:"comparison_status"`
	LastSyncedAt        *time.Time                        `json:"last_synced_at"`
	TargetPath          string                            `json:"target_path"`
}
type SupplierDashboardProviderItem struct {
	ProviderSlug            string                          `json:"provider_slug"`
	ProviderName            string                          `json:"provider_name"`
	Enabled                 bool                            `json:"enabled"`
	Status                  SupplierDashboardProviderStatus `json:"status"`
	CriticalIssueCount      *int                            `json:"critical_issue_count"`
	EnabledAccountCount     int                             `json:"enabled_account_count"`
	SchedulableAccountCount int                             `json:"schedulable_account_count"`
	RequestCount            *int64                          `json:"request_count"`
	SuccessRate             *float64                        `json:"success_rate"`
	PeriodCost              *float64                        `json:"period_cost"`
	CostCurrency            *string                         `json:"cost_currency"`
	Balance                 *float64                        `json:"balance"`
	BalanceCurrency         *string                         `json:"balance_currency"`
	EstimatedDays           *float64                        `json:"estimated_days"`
	RateRiskCount           int                             `json:"rate_risk_count"`
	BalanceRisk             bool                            `json:"balance_risk"`
	SyncRisk                bool                            `json:"sync_risk"`
	TargetPath              string                          `json:"target_path"`
}

type SupplierDashboardAccountsResponse struct {
	Range       SupplierDashboardRange         `json:"range"`
	Items       []SupplierDashboardAccountItem `json:"items"`
	Total       int                            `json:"total"`
	Page        int                            `json:"page"`
	PageSize    int                            `json:"page_size"`
	Warnings    []SupplierDashboardWarning     `json:"warnings"`
	GeneratedAt time.Time                      `json:"generated_at"`
}
type SupplierDashboardRatesResponse struct {
	Range       SupplierDashboardRange      `json:"range"`
	Items       []SupplierDashboardRateItem `json:"items"`
	Total       int                         `json:"total"`
	Page        int                         `json:"page"`
	PageSize    int                         `json:"page_size"`
	Warnings    []SupplierDashboardWarning  `json:"warnings"`
	GeneratedAt time.Time                   `json:"generated_at"`
}
type SupplierDashboardProvidersResponse struct {
	Range       SupplierDashboardRange          `json:"range"`
	Items       []SupplierDashboardProviderItem `json:"items"`
	Total       int                             `json:"total"`
	Page        int                             `json:"page"`
	PageSize    int                             `json:"page_size"`
	Warnings    []SupplierDashboardWarning      `json:"warnings"`
	GeneratedAt time.Time                       `json:"generated_at"`
}

type SupplierDashboardTrafficPoint struct {
	Time     string `json:"time"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}
type SupplierDashboardTrafficAccount struct {
	AccountID    int64  `json:"account_id"`
	AccountName  string `json:"account_name"`
	ProviderSlug string `json:"provider_slug"`
	ProviderName string `json:"provider_name"`
	GroupKey     string `json:"group_key"`
	GroupName    string `json:"group_name"`
}
type SupplierDashboardTrafficResponse struct {
	Range       SupplierDashboardRange            `json:"range"`
	Series      []SupplierDashboardTrafficPoint   `json:"series"`
	Accounts    []SupplierDashboardTrafficAccount `json:"accounts"`
	Warnings    []SupplierDashboardWarning        `json:"warnings"`
	GeneratedAt time.Time                         `json:"generated_at"`
}
type SupplierDashboardProfitItem struct {
	AccountID    int64   `json:"account_id"`
	AccountName  string  `json:"account_name"`
	ProviderSlug string  `json:"provider_slug"`
	ProviderName string  `json:"provider_name"`
	GroupKey     string  `json:"group_key"`
	GroupName    string  `json:"group_name"`
	Requests     int64   `json:"requests"`
	Tokens       int64   `json:"tokens"`
	ActualCost   float64 `json:"actual_cost"`
	UserCost     float64 `json:"user_cost"`
	Profit       float64 `json:"profit"`
}
type SupplierDashboardProfitResponse struct {
	Items       []SupplierDashboardProfitItem `json:"items"`
	Warnings    []SupplierDashboardWarning    `json:"warnings"`
	GeneratedAt time.Time                     `json:"generated_at"`
}
type SupplierDashboardHealthStatusCounts struct {
	Healthy     int `json:"healthy"`
	Slow        int `json:"slow"`
	Failed      int `json:"failed"`
	Unavailable int `json:"unavailable"`
	Skipped     int `json:"skipped"`
}
type SupplierDashboardHealthHour struct {
	Time         string                              `json:"time"`
	StatusCounts SupplierDashboardHealthStatusCounts `json:"status_counts"`
	Total        int                                 `json:"total"`
}
type SupplierDashboardHealthCell struct {
	Time   string `json:"time"`
	Status string `json:"status"`
}
type SupplierDashboardHealthAccount struct {
	AccountID    int64                         `json:"account_id"`
	AccountName  string                        `json:"account_name"`
	ProviderSlug string                        `json:"provider_slug"`
	ProviderName string                        `json:"provider_name"`
	GroupKey     string                        `json:"group_key"`
	GroupName    string                        `json:"group_name"`
	Cells        []SupplierDashboardHealthCell `json:"cells"`
}
type SupplierDashboardAccountHealthResponse struct {
	Range       SupplierDashboardRange           `json:"range"`
	Accounts    []SupplierDashboardHealthAccount `json:"accounts"`
	Hours       []SupplierDashboardHealthHour    `json:"hours"`
	Warnings    []SupplierDashboardWarning       `json:"warnings"`
	GeneratedAt time.Time                        `json:"generated_at"`
}

func (r *SupplierDashboardAccountsResponse) addWarning(source string, err error) {
	if err != nil {
		r.Warnings = append(r.Warnings, SupplierDashboardWarning{Source: source, Message: err.Error()})
	}
}
func (r *SupplierDashboardRatesResponse) addWarning(source string, err error) {
	if err != nil {
		r.Warnings = append(r.Warnings, SupplierDashboardWarning{Source: source, Message: err.Error()})
	}
}
func (r *SupplierDashboardProvidersResponse) addWarning(source string, err error) {
	if err != nil {
		r.Warnings = append(r.Warnings, SupplierDashboardWarning{Source: source, Message: err.Error()})
	}
}
func (r *SupplierDashboardTrafficResponse) addWarning(source string, err error) {
	if err != nil {
		r.Warnings = append(r.Warnings, SupplierDashboardWarning{Source: source, Message: err.Error()})
	}
}
func (r *SupplierDashboardProfitResponse) addWarning(source string, err error) {
	if err != nil {
		r.Warnings = append(r.Warnings, SupplierDashboardWarning{Source: source, Message: err.Error()})
	}
}
func (r *SupplierDashboardAccountHealthResponse) addWarning(source string, err error) {
	if err != nil {
		r.Warnings = append(r.Warnings, SupplierDashboardWarning{Source: source, Message: err.Error()})
	}
}

func (s *SupplierDashboardService) GetAccounts(ctx context.Context, q SupplierDashboardAccountsQuery) (SupplierDashboardAccountsResponse, error) {
	q.Range = SupplierDashboardRange(strings.TrimSpace(string(q.Range)))
	q.RiskType = SupplierDashboardRiskType(strings.TrimSpace(string(q.RiskType)))
	q.ProviderSlug = strings.TrimSpace(q.ProviderSlug)
	q.GroupKey = strings.TrimSpace(q.GroupKey)
	start, end, _, err := s.resolveRange(q.Range)
	if err != nil {
		return SupplierDashboardAccountsResponse{}, err
	}
	if !validRiskFilter(q.RiskType) {
		return SupplierDashboardAccountsResponse{}, fmt.Errorf("unsupported supplier dashboard risk_type %q", q.RiskType)
	}
	if q.RiskType == "" {
		q.RiskType = SupplierDashboardRiskTypeAll
	}
	q.Page, q.PageSize = dashboardPage(q.Page, q.PageSize)
	result := SupplierDashboardAccountsResponse{Range: q.Range, Items: []SupplierDashboardAccountItem{}, Page: q.Page, PageSize: q.PageSize, Warnings: []SupplierDashboardWarning{}, GeneratedAt: end}
	snapshots, listErr := s.detail.ListDashboardAccounts(ctx, start, end, q.ProviderSlug, q.GroupKey)
	if dashboardContextErr(listErr) {
		return SupplierDashboardAccountsResponse{}, listErr
	}
	result.addWarning("dashboard_accounts", listErr)
	stats := accountGroupStats(snapshots)
	for _, snap := range snapshots {
		if !snap.ProviderEnabled || !snap.AccountEnabled || (q.ProviderSlug != "" && snap.ProviderSlug != q.ProviderSlug) || (q.GroupKey != "" && snap.GroupKey != q.GroupKey) {
			continue
		}
		item := buildAccountItem(snap, stats)
		if !accountItemMatchesRisk(item, q.RiskType) {
			continue
		}
		result.Items = append(result.Items, item)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]
		if severityWeight(a.Severity) != severityWeight(b.Severity) {
			return severityWeight(a.Severity) > severityWeight(b.Severity)
		}
		if a.TrafficImpact != b.TrafficImpact {
			return a.TrafficImpact > b.TrafficImpact
		}
		if !a.DetectedAt.Equal(b.DetectedAt) {
			return a.DetectedAt.After(b.DetectedAt)
		}
		return a.AccountID < b.AccountID
	})
	result.Total = len(result.Items)
	result.Items = pageSlice(result.Items, q.Page, q.PageSize)
	return result, nil
}

func (s *SupplierDashboardService) GetRates(ctx context.Context, q SupplierDashboardRatesQuery) (SupplierDashboardRatesResponse, error) {
	q.Range = SupplierDashboardRange(strings.TrimSpace(string(q.Range)))
	q.View = SupplierDashboardRateView(strings.TrimSpace(string(q.View)))
	q.ComparisonStatus = SupplierDashboardComparisonStatus(strings.TrimSpace(string(q.ComparisonStatus)))
	q.ProviderSlug = strings.TrimSpace(q.ProviderSlug)
	q.GroupKey = strings.TrimSpace(q.GroupKey)
	start, end, _, err := s.resolveRange(q.Range)
	if err != nil {
		return SupplierDashboardRatesResponse{}, err
	}
	if q.View == "" {
		q.View = SupplierDashboardRateViewAll
	}
	if q.View != SupplierDashboardRateViewAll && q.View != SupplierDashboardRateViewRisk && q.View != SupplierDashboardRateViewChanged {
		return SupplierDashboardRatesResponse{}, fmt.Errorf("unsupported supplier dashboard rate view %q", q.View)
	}
	if !validComparisonStatus(q.ComparisonStatus) {
		return SupplierDashboardRatesResponse{}, fmt.Errorf("unsupported supplier dashboard comparison_status %q", q.ComparisonStatus)
	}
	q.Page, q.PageSize = dashboardPage(q.Page, q.PageSize)
	result := SupplierDashboardRatesResponse{Range: q.Range, Items: []SupplierDashboardRateItem{}, Page: q.Page, PageSize: q.PageSize, Warnings: []SupplierDashboardWarning{}, GeneratedAt: end}
	snapshots, listErr := s.detail.ListDashboardRates(ctx, start, end, q.ProviderSlug, q.GroupKey)
	if dashboardContextErr(listErr) {
		return SupplierDashboardRatesResponse{}, listErr
	}
	result.addWarning("dashboard_rates", listErr)
	for _, group := range groupRateSnapshots(snapshots) {
		if q.ProviderSlug != "" && group[0].ProviderSlug != q.ProviderSlug || q.GroupKey != "" && group[0].GroupKey != q.GroupKey {
			continue
		}
		item, changed, risky := buildRateGroupItem(group)
		if q.ComparisonStatus != "" && item.ComparisonStatus != q.ComparisonStatus || q.View == SupplierDashboardRateViewChanged && !changed || q.View == SupplierDashboardRateViewRisk && !risky {
			continue
		}
		result.Items = append(result.Items, item)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]
		if a.ComparisonStatus != b.ComparisonStatus {
			return a.ComparisonStatus == SupplierDashboardComparisonStatusNotLowest
		}
		if a.ProviderSlug != b.ProviderSlug {
			return a.ProviderSlug < b.ProviderSlug
		}
		return a.GroupKey < b.GroupKey
	})
	result.Total = len(result.Items)
	result.Items = pageSlice(result.Items, q.Page, q.PageSize)
	return result, nil
}

type supplierDashboardMetricThresholdSource interface {
	GetMetricThresholds(context.Context) (*OpsMetricThresholds, error)
}

func (s *SupplierDashboardService) GetProviders(ctx context.Context, q SupplierDashboardProvidersQuery) (SupplierDashboardProvidersResponse, error) {
	q.Range = SupplierDashboardRange(strings.TrimSpace(string(q.Range)))
	q.Status = SupplierDashboardProviderStatus(strings.TrimSpace(string(q.Status)))
	start, end, _, err := s.resolveRange(q.Range)
	if err != nil {
		return SupplierDashboardProvidersResponse{}, err
	}
	if !validProviderStatus(q.Status) {
		return SupplierDashboardProvidersResponse{}, fmt.Errorf("unsupported supplier dashboard provider status %q", q.Status)
	}
	q.Page, q.PageSize = dashboardPage(q.Page, q.PageSize)
	result := SupplierDashboardProvidersResponse{Range: q.Range, Items: []SupplierDashboardProviderItem{}, Page: q.Page, PageSize: q.PageSize, Warnings: []SupplierDashboardWarning{}, GeneratedAt: end}
	slaMin, thresholdErr := s.dashboardSLAMin(ctx)
	if dashboardContextErr(thresholdErr) {
		return SupplierDashboardProvidersResponse{}, thresholdErr
	}
	result.addWarning("ops_metric_thresholds", thresholdErr)
	snapshots, listErr := s.detail.ListDashboardProviders(ctx, start, end)
	if dashboardContextErr(listErr) {
		return SupplierDashboardProvidersResponse{}, listErr
	}
	result.addWarning("dashboard_providers", listErr)
	for _, snap := range snapshots {
		item := buildProviderItem(snap, slaMin)
		if q.Status != "" && q.Status != item.Status {
			continue
		}
		result.Items = append(result.Items, item)
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		if providerStatusWeight(result.Items[i].Status) != providerStatusWeight(result.Items[j].Status) {
			return providerStatusWeight(result.Items[i].Status) > providerStatusWeight(result.Items[j].Status)
		}
		return result.Items[i].ProviderSlug < result.Items[j].ProviderSlug
	})
	result.Total = len(result.Items)
	result.Items = pageSlice(result.Items, q.Page, q.PageSize)
	return result, nil
}

func (s *SupplierDashboardService) GetAccountTraffic(ctx context.Context, q SupplierDashboardTrafficQuery) (SupplierDashboardTrafficResponse, error) {
	q.Range = SupplierDashboardRange(strings.TrimSpace(string(q.Range)))
	q.ProviderSlug = strings.TrimSpace(q.ProviderSlug)
	q.GroupKey = strings.TrimSpace(q.GroupKey)
	start, end, _, err := s.resolveRange(q.Range)
	if err != nil {
		return SupplierDashboardTrafficResponse{}, err
	}
	result := SupplierDashboardTrafficResponse{Range: q.Range, Series: []SupplierDashboardTrafficPoint{}, Accounts: []SupplierDashboardTrafficAccount{}, Warnings: []SupplierDashboardWarning{}, GeneratedAt: end}
	rows, listErr := s.detail.ListDashboardAccountTraffic(ctx, start, end, q.ProviderSlug, q.GroupKey)
	if dashboardContextErr(listErr) {
		return SupplierDashboardTrafficResponse{}, listErr
	}
	result.addWarning("dashboard_account_traffic", listErr)
	series := map[string]*SupplierDashboardTrafficPoint{}
	order := []string{}
	accountSeen := map[int64]bool{}
	for _, row := range rows {
		point, ok := series[row.Time]
		if !ok {
			point = &SupplierDashboardTrafficPoint{Time: row.Time}
			series[row.Time] = point
			order = append(order, row.Time)
		}
		point.Requests += row.Requests
		point.Tokens += row.Tokens
		if accountSeen[row.AccountID] {
			continue
		}
		accountSeen[row.AccountID] = true
		result.Accounts = append(result.Accounts, SupplierDashboardTrafficAccount{
			AccountID: row.AccountID, AccountName: row.AccountName, ProviderSlug: row.ProviderSlug, ProviderName: row.ProviderName,
			GroupKey: row.GroupKey, GroupName: row.GroupName,
		})
	}
	sort.SliceStable(result.Accounts, func(i, j int) bool {
		return result.Accounts[i].AccountID < result.Accounts[j].AccountID
	})
	for _, key := range order {
		result.Series = append(result.Series, *series[key])
	}
	return result, nil
}

func (s *SupplierDashboardService) GetAccountProfitRanking(ctx context.Context, q SupplierDashboardProfitQuery) (SupplierDashboardProfitResponse, error) {
	q.Range = SupplierDashboardRange(strings.TrimSpace(string(q.Range)))
	q.ProviderSlug = strings.TrimSpace(q.ProviderSlug)
	q.GroupKey = strings.TrimSpace(q.GroupKey)
	start, end, _, err := s.resolveRange(q.Range)
	if err != nil {
		return SupplierDashboardProfitResponse{}, err
	}
	limit := q.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	result := SupplierDashboardProfitResponse{Items: []SupplierDashboardProfitItem{}, Warnings: []SupplierDashboardWarning{}, GeneratedAt: end}
	rows, listErr := s.detail.ListDashboardAccountProfit(ctx, start, end, q.ProviderSlug, q.GroupKey, limit)
	if dashboardContextErr(listErr) {
		return SupplierDashboardProfitResponse{}, listErr
	}
	result.addWarning("dashboard_account_profit", listErr)
	for _, row := range rows {
		result.Items = append(result.Items, SupplierDashboardProfitItem{
			AccountID: row.AccountID, AccountName: row.AccountName, ProviderSlug: row.ProviderSlug, ProviderName: row.ProviderName,
			GroupKey: row.GroupKey, GroupName: row.GroupName, Requests: row.Requests, Tokens: row.Tokens,
			ActualCost: row.ActualCost, UserCost: row.UserCost, Profit: row.UserCost - row.ActualCost,
		})
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]
		if a.Profit != b.Profit {
			return a.Profit > b.Profit
		}
		return a.AccountID < b.AccountID
	})
	if len(result.Items) > limit {
		result.Items = result.Items[:limit]
	}
	return result, nil
}

func (s *SupplierDashboardService) GetAccountHealthTimeline(ctx context.Context, q SupplierDashboardAccountHealthQuery) (SupplierDashboardAccountHealthResponse, error) {
	q.Range = SupplierDashboardRange(strings.TrimSpace(string(q.Range)))
	q.ProviderSlug = strings.TrimSpace(q.ProviderSlug)
	q.GroupKey = strings.TrimSpace(q.GroupKey)
	start, end, _, err := s.resolveRange(q.Range)
	if err != nil {
		return SupplierDashboardAccountHealthResponse{}, err
	}
	limit := q.Limit
	if limit < 1 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	buckets := q.Buckets
	if buckets < 1 {
		buckets = 72
	}
	if buckets > 24*30 {
		buckets = 24 * 30
	}
	bucketSeconds := q.BucketHours * 3600
	if q.BucketMinutes > 0 {
		bucketSeconds = q.BucketMinutes * 60
	}
	if bucketSeconds <= 0 {
		bucketSeconds = 3600
	}
	if bucketSeconds < 60 {
		bucketSeconds = 60
	}
	if bucketSeconds > 24*3600 {
		bucketSeconds = 24 * 3600
	}
	result := SupplierDashboardAccountHealthResponse{Range: q.Range, Accounts: []SupplierDashboardHealthAccount{}, Hours: []SupplierDashboardHealthHour{}, Warnings: []SupplierDashboardWarning{}, GeneratedAt: end}
	rows, listErr := s.detail.ListDashboardAccountHealth(ctx, start, end, q.ProviderSlug, q.GroupKey, bucketSeconds)
	if dashboardContextErr(listErr) {
		return SupplierDashboardAccountHealthResponse{}, listErr
	}
	result.addWarning("dashboard_account_health", listErr)

	// 按账号 ID 去重收集元信息，稍后按 limit 截断。
	accountMeta := map[int64]SupplierDashboardHealthAccount{}
	accountOrder := []int64{}
	for _, row := range rows {
		if _, ok := accountMeta[row.AccountID]; ok {
			continue
		}
		accountMeta[row.AccountID] = SupplierDashboardHealthAccount{
			AccountID: row.AccountID, AccountName: row.AccountName, ProviderSlug: row.ProviderSlug, ProviderName: row.ProviderName,
			GroupKey: row.GroupKey, GroupName: row.GroupName, Cells: []SupplierDashboardHealthCell{},
		}
		accountOrder = append(accountOrder, row.AccountID)
	}
	sort.SliceStable(accountOrder, func(i, j int) bool { return accountOrder[i] < accountOrder[j] })
	if len(accountOrder) > limit {
		accountOrder = accountOrder[:limit]
	}
	selected := map[int64]bool{}
	for _, accountID := range accountOrder {
		selected[accountID] = true
		result.Accounts = append(result.Accounts, accountMeta[accountID])
	}

	// 汇总选中账号的小时桶，随后按 buckets 截取最近区间。
	hourSet := map[string]bool{}
	for _, row := range rows {
		if selected[row.AccountID] {
			hourSet[row.Time] = true
		}
	}
	hourOrder := make([]string, 0, len(hourSet))
	for key := range hourSet {
		hourOrder = append(hourOrder, key)
	}
	sort.Strings(hourOrder)
	if len(hourOrder) > buckets {
		hourOrder = hourOrder[len(hourOrder)-buckets:]
	}
	keptHours := map[string]int{}
	for i, key := range hourOrder {
		keptHours[key] = i
	}

	// 为每个账号预置小时格，再填充状态与每小时计数。
	perAccountCells := map[int64][]SupplierDashboardHealthCell{}
	for i := range result.Accounts {
		cells := make([]SupplierDashboardHealthCell, len(hourOrder))
		for j, key := range hourOrder {
			cells[j] = SupplierDashboardHealthCell{Time: key}
		}
		perAccountCells[result.Accounts[i].AccountID] = cells
	}
	perHourCounts := map[string]*SupplierDashboardHealthStatusCounts{}
	for _, row := range rows {
		if !selected[row.AccountID] {
			continue
		}
		idx, ok := keptHours[row.Time]
		if !ok {
			continue
		}
		cell := &perAccountCells[row.AccountID][idx]
		cell.Status = row.Status
		counts := perHourCounts[row.Time]
		if counts == nil {
			counts = &SupplierDashboardHealthStatusCounts{}
			perHourCounts[row.Time] = counts
		}
		switch row.Status {
		case "healthy":
			counts.Healthy++
		case "slow":
			counts.Slow++
		case "failed":
			counts.Failed++
		case "unavailable":
			counts.Unavailable++
		case "skipped":
			counts.Skipped++
		}
	}
	for i := range result.Accounts {
		result.Accounts[i].Cells = perAccountCells[result.Accounts[i].AccountID]
	}
	for _, key := range hourOrder {
		counts := perHourCounts[key]
		if counts == nil {
			counts = &SupplierDashboardHealthStatusCounts{}
		}
		total := counts.Healthy + counts.Slow + counts.Failed + counts.Unavailable + counts.Skipped
		result.Hours = append(result.Hours, SupplierDashboardHealthHour{Time: key, StatusCounts: *counts, Total: total})
	}
	return result, nil
}

func (s *SupplierDashboardService) dashboardSLAMin(ctx context.Context) (float64, error) {
	defaults := defaultOpsMetricThresholds()
	fallback := 99.5
	if defaults != nil && defaults.SLAPercentMin != nil && finite(*defaults.SLAPercentMin) {
		fallback = *defaults.SLAPercentMin
	}
	source := s.thresholds
	if source == nil {
		return fallback, nil
	}
	thresholds, err := source.GetMetricThresholds(ctx)
	if err != nil {
		return fallback, err
	}
	if thresholds == nil || thresholds.SLAPercentMin == nil || !finite(*thresholds.SLAPercentMin) || *thresholds.SLAPercentMin < 0 || *thresholds.SLAPercentMin > 100 {
		return fallback, errors.New("invalid ops SLA threshold")
	}
	return *thresholds.SLAPercentMin, nil
}

type rateStat struct {
	count  int
	lowest float64
}

func accountGroupStats(snaps []SupplierDashboardAccountSnapshot) map[string]rateStat {
	stats := map[string]rateStat{}
	for _, snap := range snaps {
		if !snap.ProviderEnabled || !snap.AccountEnabled || strings.TrimSpace(snap.GroupKey) == "" || snap.CurrentRate == nil || !finite(*snap.CurrentRate) {
			continue
		}
		key := rateGroupKey(snap.ProviderSlug, snap.GroupKey)
		stat := stats[key]
		if stat.count == 0 || *snap.CurrentRate < stat.lowest {
			stat.lowest = *snap.CurrentRate
		}
		stat.count++
		stats[key] = stat
	}
	return stats
}

func buildAccountItem(snap SupplierDashboardAccountSnapshot, stats map[string]rateStat) SupplierDashboardAccountItem {
	severity, risks := rawAccountRisks(snap.ProviderRiskLevel, snap.RateSyncStatus, snap.TaskStatus)
	requestCount, successRate := dashboardTraffic(snap.SuccessCount, snap.ErrorCount)
	rateDelta, rateIncreased, _ := dashboardRateEventChange(snap.RateChangeOld, snap.RateChangeNew, snap.RateChangeCount)
	lowestRate := (*float64)(nil)
	if snap.CurrentRate != nil && finite(*snap.CurrentRate) && strings.TrimSpace(snap.GroupKey) != "" {
		stat := stats[rateGroupKey(snap.ProviderSlug, snap.GroupKey)]
		if stat.count >= 2 {
			lowestRate = floatPtr(stat.lowest)
			if *snap.CurrentRate > stat.lowest+1e-9 {
				risks = append(risks, SupplierDashboardRiskTypeNotLowest)
				severity = promoteSeverity(severity, SupplierDashboardSeverityMedium)
			}
		}
	}
	if rateIncreased {
		risks = append(risks, SupplierDashboardRiskTypeRateUp)
		severity = promoteSeverity(severity, SupplierDashboardSeverityHigh)
	}
	if strings.EqualFold(strings.TrimSpace(snap.BalanceSyncStatus), "success") && snap.EstimatedDays != nil && finite(*snap.EstimatedDays) && *snap.EstimatedDays < 3 {
		risks = append(risks, SupplierDashboardRiskTypeBalance)
		severity = promoteSeverity(severity, SupplierDashboardSeverityMedium)
	}
	risks = normalizeRisks(risks)
	trafficImpact := int64(0)
	if len(risks) > 0 && requestCount != nil && *requestCount > 0 {
		risks = normalizeRisks(append(risks, SupplierDashboardRiskTypeTraffic))
		trafficImpact = *requestCount
		severity = promoteSeverity(severity, SupplierDashboardSeverityHigh)
	}
	if severityWeight(severity) == 0 {
		severity = SupplierDashboardSeverityLow
	}
	extraCost := (*float64)(nil)
	if lowestRate != nil && snap.CurrentRate != nil && snap.PeriodCost != nil && finite(*snap.PeriodCost) && *snap.CurrentRate > 0 {
		if *snap.CurrentRate <= *lowestRate+1e-9 {
			extraCost = floatPtr(0)
		} else {
			extraCost = finiteValue(*snap.PeriodCost * (*snap.CurrentRate - *lowestRate) / *snap.CurrentRate)
		}
	}
	status := strings.TrimSpace(snap.AccountStatus)
	if status == "" {
		status = "unknown"
	}
	detectedAt := accountRiskDetectedAt(snap, risks)
	return SupplierDashboardAccountItem{
		AccountID: snap.AccountID, AccountName: snap.AccountName, ProviderSlug: snap.ProviderSlug, ProviderName: snap.ProviderName,
		GroupKey: snap.GroupKey, GroupName: snap.GroupName, Severity: severity, RiskTypes: risks,
		RequestCount: requestCount, SuccessRate: successRate, CurrentRate: finitePtr(snap.CurrentRate), LowestRate: lowestRate,
		RateDeltaPercent: rateDelta, Balance: finitePtr(snap.Balance), BalanceCurrency: nil, EstimatedDays: finitePtr(snap.EstimatedDays),
		Status: status, Reason: snap.TaskReason, TrafficImpact: trafficImpact, PeriodCost: finitePtr(snap.PeriodCost), EstimatedExtraCost: extraCost,
		DetectedAt: detectedAt, TargetPath: fmt.Sprintf("/admin/upstream-management/accounts?account_id=%d", snap.AccountID),
	}
}

func accountRiskDetectedAt(snap SupplierDashboardAccountSnapshot, risks []SupplierDashboardRiskType) time.Time {
	detectedAt := time.Time{}
	add := func(candidate *time.Time) {
		if candidate != nil && candidate.After(detectedAt) {
			detectedAt = *candidate
		}
	}
	for _, risk := range risks {
		switch risk {
		case SupplierDashboardRiskTypeCritical:
			add(snap.ProviderRiskUpdatedAt)
		case SupplierDashboardRiskTypeSync:
			add(snap.LastRateSyncedAt)
		case SupplierDashboardRiskTypeTask:
			add(snap.TaskFinishedAt)
		case SupplierDashboardRiskTypeRateUp:
			add(snap.RateChangedAt)
		case SupplierDashboardRiskTypeBalance:
			add(snap.BalanceSyncedAt)
		case SupplierDashboardRiskTypeNotLowest:
			add(&snap.ObservedAt)
		}
	}
	if detectedAt.IsZero() {
		return snap.ObservedAt
	}
	return detectedAt
}

func groupRateSnapshots(snaps []SupplierDashboardRateSnapshot) [][]SupplierDashboardRateSnapshot {
	groups := map[string][]SupplierDashboardRateSnapshot{}
	keys := []string{}
	for _, snap := range snaps {
		if !snap.ProviderEnabled || !snap.AccountEnabled {
			continue
		}
		key := rateGroupKey(snap.ProviderSlug, snap.GroupKey)
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], snap)
	}
	sort.Strings(keys)
	result := make([][]SupplierDashboardRateSnapshot, 0, len(keys))
	for _, key := range keys {
		group := groups[key]
		sort.SliceStable(group, func(i, j int) bool { return group[i].AccountID < group[j].AccountID })
		result = append(result, group)
	}
	return result
}

func buildRateGroupItem(group []SupplierDashboardRateSnapshot) (SupplierDashboardRateItem, bool, bool) {
	current := group[0]
	currentRequests := dashboardRequestCountValue(current.SuccessCount, current.ErrorCount)
	for _, candidate := range group[1:] {
		requests := dashboardRequestCountValue(candidate.SuccessCount, candidate.ErrorCount)
		if requests > currentRequests || requests == currentRequests && candidate.AccountID < current.AccountID {
			current, currentRequests = candidate, requests
		}
	}
	item := SupplierDashboardRateItem{
		ProviderSlug: current.ProviderSlug, ProviderName: current.ProviderName, GroupKey: current.GroupKey, GroupName: current.GroupName,
		EnabledAccountCount: len(group), CurrentAccountID: current.AccountID, CurrentAccountName: current.AccountName,
		CurrentRate: finitePtr(current.CurrentRate), LowestAccountIDs: []int64{}, LowestAccountNames: []string{}, CostCurrency: nil,
		ComparisonStatus: SupplierDashboardComparisonStatusUnknown,
		TargetPath:       "/admin/upstream-management/accounts?provider=" + url.QueryEscape(current.ProviderSlug) + "&group_key=" + url.QueryEscape(current.GroupKey),
	}
	for _, snap := range group {
		if snap.LastRateSyncedAt != nil && (item.LastSyncedAt == nil || snap.LastRateSyncedAt.After(*item.LastSyncedAt)) {
			t := *snap.LastRateSyncedAt
			item.LastSyncedAt = &t
		}
	}
	if strings.TrimSpace(current.GroupKey) == "" {
		item.ComparisonStatus = SupplierDashboardComparisonStatusMissingGroup
	} else if item.CurrentRate != nil {
		for _, snap := range group {
			if snap.CurrentRate == nil || !finite(*snap.CurrentRate) {
				continue
			}
			if item.LowestRate == nil || *snap.CurrentRate < *item.LowestRate-1e-9 {
				item.LowestRate = floatPtr(*snap.CurrentRate)
				item.LowestAccountIDs = []int64{snap.AccountID}
				item.LowestAccountNames = []string{snap.AccountName}
			} else if math.Abs(*snap.CurrentRate-*item.LowestRate) <= 1e-9 {
				item.LowestAccountIDs = append(item.LowestAccountIDs, snap.AccountID)
				item.LowestAccountNames = append(item.LowestAccountNames, snap.AccountName)
			}
		}
		if len(group) < 2 {
			item.ComparisonStatus = SupplierDashboardComparisonStatusInsufficientAccounts
		} else if item.LowestRate != nil && *item.CurrentRate > *item.LowestRate+1e-9 {
			item.ComparisonStatus = SupplierDashboardComparisonStatusNotLowest
		} else if item.LowestRate != nil && len(item.LowestAccountIDs) > 1 {
			item.ComparisonStatus = SupplierDashboardComparisonStatusTiedLowest
		} else if item.LowestRate != nil {
			item.ComparisonStatus = SupplierDashboardComparisonStatusLowest
		}
	}
	rateDeltaPercent, rateIncreased, changed := dashboardRateEventChange(current.RateChangeOld, current.RateChangeNew, current.RateChangeCount)
	item.RateDeltaPercent = rateDeltaPercent
	if item.ComparisonStatus == SupplierDashboardComparisonStatusLowest || item.ComparisonStatus == SupplierDashboardComparisonStatusTiedLowest {
		item.EstimatedExtraCost = floatPtr(0)
	} else if item.ComparisonStatus == SupplierDashboardComparisonStatusNotLowest && item.CurrentRate != nil && item.LowestRate != nil && current.PeriodCost != nil && finite(*current.PeriodCost) && *item.CurrentRate > 0 {
		item.EstimatedExtraCost = finiteValue(*current.PeriodCost * (*item.CurrentRate - *item.LowestRate) / *item.CurrentRate)
	}
	return item, changed, item.ComparisonStatus == SupplierDashboardComparisonStatusNotLowest || rateIncreased
}

func buildProviderItem(snap SupplierDashboardProviderSnapshot, slaMin float64) SupplierDashboardProviderItem {
	requestCount, successRate := dashboardTraffic(snap.SuccessCount, snap.ErrorCount)
	balanceRisk := strings.EqualFold(strings.TrimSpace(snap.BalanceSyncStatus), "success") && snap.EstimatedDays != nil && finite(*snap.EstimatedDays) && *snap.EstimatedDays < 3
	syncRisk := riskySync(snap.SyncStatus) || riskySync(snap.GroupSyncStatus) || riskySync(snap.BalanceSyncStatus)
	criticalCount := (*int)(nil)
	factsComplete := snap.DataComplete && (snap.EnabledAccountCount == 0 || snap.SuccessCount != nil && snap.ErrorCount != nil)
	if factsComplete {
		count := 0
		level := strings.ToLower(strings.TrimSpace(snap.ProviderRiskLevel))
		if level == "critical" || level == "high" {
			count++
		}
		if requestCount != nil && *requestCount > 0 && successRate != nil && finite(*successRate) && *successRate < slaMin {
			count++
		}
		criticalCount = &count
	}
	status := providerStatusFromFacts(snap, requestCount, successRate, criticalCount, balanceRisk, syncRisk, slaMin)
	return SupplierDashboardProviderItem{
		ProviderSlug: snap.ProviderSlug, ProviderName: snap.ProviderName, Enabled: snap.Enabled, Status: status,
		CriticalIssueCount: criticalCount, EnabledAccountCount: snap.EnabledAccountCount, SchedulableAccountCount: snap.SchedulableAccountCount,
		RequestCount: requestCount, SuccessRate: successRate, PeriodCost: finitePtr(snap.PeriodCost), CostCurrency: nil,
		Balance: finitePtr(snap.Balance), BalanceCurrency: nil, EstimatedDays: finitePtr(snap.EstimatedDays), RateRiskCount: snap.RateRiskCount,
		BalanceRisk: balanceRisk, SyncRisk: syncRisk, TargetPath: "/admin/upstream-management/providers?provider=" + url.QueryEscape(snap.ProviderSlug),
	}
}

func providerStatusFromFacts(snap SupplierDashboardProviderSnapshot, requestCount *int64, successRate *float64, criticalCount *int, balanceRisk, syncRisk bool, slaMin float64) SupplierDashboardProviderStatus {
	if !snap.Enabled {
		return SupplierDashboardProviderStatusDisabled
	}
	level := strings.ToLower(strings.TrimSpace(snap.ProviderRiskLevel))
	lowSLA := requestCount != nil && *requestCount > 0 && successRate != nil && finite(*successRate) && *successRate < slaMin
	if level == "critical" || level == "high" || lowSLA || criticalCount != nil && *criticalCount > 0 {
		return SupplierDashboardProviderStatusHighRisk
	}
	if syncRisk || balanceRisk || snap.RateRiskCount > 0 || level == "medium" || level == "warning" {
		return SupplierDashboardProviderStatusWarning
	}
	if !snap.DataComplete || providerRawNonFinite(snap) || criticalCount == nil {
		return SupplierDashboardProviderStatusUnknown
	}
	return SupplierDashboardProviderStatusHealthy
}

func providerRawNonFinite(snap SupplierDashboardProviderSnapshot) bool {
	for _, value := range []*float64{snap.Balance, snap.EstimatedDays, snap.PeriodCost} {
		if value != nil && !finite(*value) {
			return true
		}
	}
	return false
}

func dashboardTraffic(success, failures *int64) (*int64, *float64) {
	count, ok := dashboardCheckedRequestCount(success, failures)
	if !ok {
		return nil, nil
	}
	if count == 0 {
		return &count, nil
	}
	rate := float64(*success) * 100 / float64(count)
	return &count, finiteValue(rate)
}

func dashboardRequestCountValue(success, failures *int64) int64 {
	count, ok := dashboardCheckedRequestCount(success, failures)
	if !ok {
		return -1
	}
	return count
}

func dashboardCheckedRequestCount(success, failures *int64) (int64, bool) {
	if success == nil || failures == nil || *success < 0 || *failures < 0 || *failures > math.MaxInt64-*success {
		return 0, false
	}
	return *success + *failures, true
}

func dashboardRateEventChange(oldRate, newRate *float64, changeCount int) (*float64, bool, bool) {
	if changeCount <= 0 || oldRate == nil || newRate == nil || !finite(*oldRate) || !finite(*newRate) {
		return nil, false, false
	}
	changed := math.Abs(*newRate-*oldRate) > 1e-9
	increased := *newRate > *oldRate+1e-9
	if math.Abs(*oldRate) > 1e-9 {
		return finiteValue((*newRate - *oldRate) / *oldRate * 100), increased, changed
	}
	if math.Abs(*newRate) <= 1e-9 {
		return floatPtr(0), false, false
	}
	return nil, increased, changed
}
func rawAccountRisks(provider, syncStatus, taskStatus string) (SupplierDashboardSeverity, []SupplierDashboardRiskType) {
	severity := SupplierDashboardSeverityLow
	risks := []SupplierDashboardRiskType{}
	if strings.EqualFold(strings.TrimSpace(provider), "critical") {
		severity = SupplierDashboardSeverityCritical
		risks = append(risks, SupplierDashboardRiskTypeCritical)
	}
	if riskySync(syncStatus) {
		severity = promoteSeverity(severity, SupplierDashboardSeverityHigh)
		risks = append(risks, SupplierDashboardRiskTypeSync)
	}
	switch strings.ToLower(strings.TrimSpace(taskStatus)) {
	case "failed", "error", "timeout", "timed_out":
		severity = promoteSeverity(severity, SupplierDashboardSeverityHigh)
		risks = append(risks, SupplierDashboardRiskTypeTask)
	}
	return severity, normalizeRisks(risks)
}

func promoteSeverity(current, candidate SupplierDashboardSeverity) SupplierDashboardSeverity {
	if severityWeight(candidate) > severityWeight(current) {
		return candidate
	}
	return current
}

func riskySync(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "failed")
}
func validRiskFilter(v SupplierDashboardRiskType) bool {
	return v == "" || v == SupplierDashboardRiskTypeAll || v == SupplierDashboardRiskTypeCritical || v == SupplierDashboardRiskTypeTraffic || v == SupplierDashboardRiskTypeRateUp || v == SupplierDashboardRiskTypeNotLowest || v == SupplierDashboardRiskTypeBalance || v == SupplierDashboardRiskTypeSync || v == SupplierDashboardRiskTypeTask
}
func validProviderStatus(v SupplierDashboardProviderStatus) bool {
	return v == "" || v == SupplierDashboardProviderStatusHealthy || v == SupplierDashboardProviderStatusWarning || v == SupplierDashboardProviderStatusHighRisk || v == SupplierDashboardProviderStatusDisabled || v == SupplierDashboardProviderStatusUnknown
}
func validComparisonStatus(v SupplierDashboardComparisonStatus) bool {
	return v == "" || v == SupplierDashboardComparisonStatusLowest || v == SupplierDashboardComparisonStatusTiedLowest || v == SupplierDashboardComparisonStatusNotLowest || v == SupplierDashboardComparisonStatusMissingGroup || v == SupplierDashboardComparisonStatusInsufficientAccounts || v == SupplierDashboardComparisonStatusUnknown
}
func rateGroupKey(providerSlug, groupKey string) string {
	return strings.TrimSpace(providerSlug) + "\x00" + strings.TrimSpace(groupKey)
}
func accountItemMatchesRisk(s SupplierDashboardAccountItem, r SupplierDashboardRiskType) bool {
	if r == SupplierDashboardRiskTypeAll {
		return true
	}
	if r == SupplierDashboardRiskTypeCritical && s.Severity == SupplierDashboardSeverityCritical {
		return true
	}
	for _, x := range s.RiskTypes {
		if x == r {
			return true
		}
	}
	return false
}
func normalizeRisks(in []SupplierDashboardRiskType) []SupplierDashboardRiskType {
	out := []SupplierDashboardRiskType{}
	seen := map[SupplierDashboardRiskType]bool{}
	for _, v := range in {
		if v == SupplierDashboardRiskTypeAll || v == "" || !validRiskFilter(v) || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
func dashboardPage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func pageSlice[T any](items []T, page, size int) []T {
	if len(items) == 0 || page < 1 || size < 1 || page-1 > len(items)/size {
		return []T{}
	}
	start := (page - 1) * size
	if start >= len(items) {
		return []T{}
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return append([]T{}, items[start:end]...)
}
func dashboardContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
func severityWeight(v SupplierDashboardSeverity) int {
	switch v {
	case SupplierDashboardSeverityCritical:
		return 4
	case SupplierDashboardSeverityHigh:
		return 3
	case SupplierDashboardSeverityMedium:
		return 2
	case SupplierDashboardSeverityLow:
		return 1
	}
	return 0
}
func providerStatusWeight(v SupplierDashboardProviderStatus) int {
	switch v {
	case SupplierDashboardProviderStatusHighRisk:
		return 5
	case SupplierDashboardProviderStatusWarning:
		return 4
	case SupplierDashboardProviderStatusUnknown:
		return 3
	case SupplierDashboardProviderStatusHealthy:
		return 2
	case SupplierDashboardProviderStatusDisabled:
		return 1
	}
	return 0
}
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func finitePtr(v *float64) *float64 {
	if v == nil || !finite(*v) {
		return nil
	}
	return floatPtr(*v)
}
func finiteValue(v float64) *float64 {
	if !finite(v) {
		return nil
	}
	return floatPtr(v)
}
func floatPtr(v float64) *float64 { return &v }
