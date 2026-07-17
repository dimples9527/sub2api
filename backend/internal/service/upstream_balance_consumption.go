package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettingKeyUpstreamBalanceSamplerConfig = "upstream_balance_sampler_config"

	DefaultUpstreamBalanceSamplerIntervalSeconds = 3600
	MinUpstreamBalanceSamplerIntervalSeconds     = 60
	upstreamBalanceStatsTimezone                 = "Asia/Shanghai"
)

type UpstreamBalanceSnapshot struct {
	ID           int64     `json:"id"`
	ProviderSlug string    `json:"provider_slug"`
	ProviderName string    `json:"provider_name"`
	ProviderType string    `json:"provider_type"`
	Balance      float64   `json:"balance"`
	TodayCost    float64   `json:"today_cost"`
	AmountScale  float64   `json:"amount_scale"`
	Status       string    `json:"status"`
	Error        string    `json:"error,omitempty"`
	CapturedAt   time.Time `json:"captured_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type UpstreamBalanceRecharge struct {
	ID           int64     `json:"id"`
	ProviderSlug string    `json:"provider_slug"`
	ProviderName string    `json:"provider_name,omitempty"`
	Amount       float64   `json:"amount"`
	AmountScale  float64   `json:"amount_scale"`
	Note         string    `json:"note,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type UpstreamBalanceRechargeInput struct {
	ProviderSlug string    `json:"provider_slug"`
	Amount       float64   `json:"amount"`
	AmountScale  float64   `json:"amount_scale"`
	Note         string    `json:"note,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type UpstreamBalanceDailyRow struct {
	ProviderSlug      string     `json:"provider_slug"`
	ProviderName      string     `json:"provider_name,omitempty"`
	Date              string     `json:"date"`
	AmountScale       float64    `json:"amount_scale"`
	OpeningBalance    float64    `json:"opening_balance"`
	ClosingBalance    float64    `json:"closing_balance"`
	CurrentBalance    float64    `json:"current_balance"`
	RechargeAmount    float64    `json:"recharge_amount"`
	ConsumptionAmount float64    `json:"consumption_amount"`
	SnapshotCount     int        `json:"snapshot_count"`
	Complete          bool       `json:"complete"`
	Anomaly           bool       `json:"anomaly"`
	FirstSnapshotAt   *time.Time `json:"first_snapshot_at,omitempty"`
	LastSnapshotAt    *time.Time `json:"last_snapshot_at,omitempty"`
}

type UpstreamLocalDailyConsumption struct {
	Date       string  `json:"date"`
	ActualCost float64 `json:"actual_cost"`
}

type UpstreamBalanceProviderSummary struct {
	ProviderSlug      string     `json:"provider_slug"`
	ProviderName      string     `json:"provider_name,omitempty"`
	CurrentBalance    float64    `json:"current_balance"`
	TodayConsumption  float64    `json:"today_consumption"`
	AmountScale       float64    `json:"amount_scale"`
	Complete          bool       `json:"complete"`
	Anomaly           bool       `json:"anomaly"`
	SnapshotCount     int        `json:"snapshot_count"`
	LastSnapshotAt    *time.Time `json:"last_snapshot_at,omitempty"`
	LastSnapshotError string     `json:"last_snapshot_error,omitempty"`
}

type UpstreamBalanceSamplerConfig struct {
	Enabled              bool               `json:"enabled"`
	IntervalSeconds      int                `json:"interval_seconds"`
	ProviderAmountScales map[string]float64 `json:"provider_amount_scales,omitempty"`
	LastRunAt            *time.Time         `json:"last_run_at,omitempty"`
	LastRunStatus        string             `json:"last_run_status,omitempty"`
	LastRunMessage       string             `json:"last_run_message,omitempty"`
	UpdatedAt            *time.Time         `json:"updated_at,omitempty"`
}

type UpstreamBalanceConsumptionOverview struct {
	Config                 UpstreamBalanceSamplerConfig              `json:"config"`
	Summaries              map[string]UpstreamBalanceProviderSummary `json:"summaries"`
	Rows                   []UpstreamBalanceDailyRow                 `json:"rows"`
	Snapshots              []UpstreamBalanceSnapshot                 `json:"snapshots"`
	LocalDailyConsumptions []UpstreamLocalDailyConsumption           `json:"local_daily_consumptions"`
}

type UpstreamBalanceStore interface {
	AddSnapshot(ctx context.Context, snapshot UpstreamBalanceSnapshot) (UpstreamBalanceSnapshot, error)
	ListSnapshots(ctx context.Context, startTime, endTime time.Time) ([]UpstreamBalanceSnapshot, error)
	ListSnapshotsBefore(ctx context.Context, before time.Time) ([]UpstreamBalanceSnapshot, error)
	ListLatestSnapshots(ctx context.Context) ([]UpstreamBalanceSnapshot, error)
	AddRecharge(ctx context.Context, recharge UpstreamBalanceRechargeInput) (UpstreamBalanceRecharge, error)
	ListRecharges(ctx context.Context, startTime, endTime time.Time) ([]UpstreamBalanceRecharge, error)
}

type upstreamBalanceProviderSource interface {
	ListProviders(ctx context.Context) ([]UpstreamProviderConfig, error)
	FetchProviderBalance(ctx context.Context, slug string) (UpstreamProviderBalance, error)
	FetchProviderTodayCost(ctx context.Context, slug string, day time.Time) (UpstreamProviderCost, error)
}

type upstreamLocalDailyUsageSource interface {
	GetGlobalDailyStatsAggregated(ctx context.Context, startTime, endTime time.Time) ([]map[string]any, error)
}

type UpstreamBalanceConsumptionService struct {
	store       UpstreamBalanceStore
	provider    upstreamBalanceProviderSource
	settingRepo SettingRepository
	usageSource upstreamLocalDailyUsageSource
	now         func() time.Time
}

func NewUpstreamBalanceConsumptionService(store UpstreamBalanceStore, provider upstreamBalanceProviderSource, settingRepo SettingRepository) *UpstreamBalanceConsumptionService {
	return &UpstreamBalanceConsumptionService{
		store:       store,
		provider:    provider,
		settingRepo: settingRepo,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

func (s *UpstreamBalanceConsumptionService) SetLocalDailyUsageSource(source upstreamLocalDailyUsageSource) {
	if s == nil {
		return
	}
	s.usageSource = source
}

func BuildUpstreamBalanceDailyRows(snapshots []UpstreamBalanceSnapshot, recharges []UpstreamBalanceRecharge, startTime, endTime time.Time, defaultScale float64) []UpstreamBalanceDailyRow {
	return buildUpstreamBalanceDailyRowsInLocation(snapshots, recharges, startTime, endTime, defaultScale, time.UTC)
}

func buildUpstreamBalanceDailyRowsInLocation(snapshots []UpstreamBalanceSnapshot, recharges []UpstreamBalanceRecharge, startTime, endTime time.Time, defaultScale float64, loc *time.Location) []UpstreamBalanceDailyRow {
	if endTime.Before(startTime) {
		return []UpstreamBalanceDailyRow{}
	}
	if defaultScale <= 0 {
		defaultScale = 1
	}
	if loc == nil {
		loc = time.UTC
	}
	type dayBucket struct {
		dateStart time.Time
		snapshots []UpstreamBalanceSnapshot
		recharges []UpstreamBalanceRecharge
	}
	buckets := map[string]*dayBucket{}
	ensure := func(providerSlug, date string, dateStart time.Time) *dayBucket {
		key := providerSlug + "|" + date
		if buckets[key] == nil {
			buckets[key] = &dayBucket{dateStart: dateStart}
		}
		return buckets[key]
	}
	lastBeforeDay := map[string]UpstreamBalanceSnapshot{}
	rememberOpeningCandidate := func(snap UpstreamBalanceSnapshot, dayStart time.Time) {
		key := snap.ProviderSlug + "|" + dayStart.Format("2006-01-02")
		if existing, ok := lastBeforeDay[key]; !ok || snap.CapturedAt.After(existing.CapturedAt) {
			lastBeforeDay[key] = snap
		}
	}
	for _, snap := range snapshots {
		if snap.ProviderSlug == "" || !snap.CapturedAt.Before(endTime) {
			continue
		}
		if snap.Status != "" && snap.Status != "success" {
			continue
		}
		localCapturedAt := snap.CapturedAt.In(loc)
		dayStart := time.Date(localCapturedAt.Year(), localCapturedAt.Month(), localCapturedAt.Day(), 0, 0, 0, 0, loc)
		rememberOpeningCandidate(snap, dayStart.AddDate(0, 0, 1))
		if snap.CapturedAt.Before(startTime) {
			continue
		}
		date := dayStart.Format("2006-01-02")
		b := ensure(snap.ProviderSlug, date, dayStart)
		b.snapshots = append(b.snapshots, snap)
	}
	for _, recharge := range recharges {
		if recharge.ProviderSlug == "" || recharge.OccurredAt.Before(startTime) || !recharge.OccurredAt.Before(endTime) {
			continue
		}
		localOccurredAt := recharge.OccurredAt.In(loc)
		dayStart := time.Date(localOccurredAt.Year(), localOccurredAt.Month(), localOccurredAt.Day(), 0, 0, 0, 0, loc)
		date := dayStart.Format("2006-01-02")
		b := ensure(recharge.ProviderSlug, date, dayStart)
		b.recharges = append(b.recharges, recharge)
	}

	rows := make([]UpstreamBalanceDailyRow, 0, len(buckets))
	for key, bucket := range buckets {
		parts := strings.SplitN(key, "|", 2)
		row := UpstreamBalanceDailyRow{ProviderSlug: parts[0], Date: parts[1]}
		sort.Slice(bucket.snapshots, func(i, j int) bool {
			return bucket.snapshots[i].CapturedAt.Before(bucket.snapshots[j].CapturedAt)
		})
		for _, recharge := range bucket.recharges {
			scale := recharge.AmountScale
			if scale <= 0 {
				scale = defaultScale
			}
			row.RechargeAmount += recharge.Amount * scale
			if row.ProviderName == "" {
				row.ProviderName = recharge.ProviderName
			}
		}
		row.SnapshotCount = len(bucket.snapshots)
		if row.SnapshotCount > 0 {
			first := bucket.snapshots[0]
			last := bucket.snapshots[row.SnapshotCount-1]
			row.ProviderName = first.ProviderName
			opening := first
			if prior, ok := lastBeforeDay[row.ProviderSlug+"|"+row.Date]; ok && prior.CapturedAt.Before(bucket.dateStart) {
				opening = prior
				if row.ProviderName == "" {
					row.ProviderName = prior.ProviderName
				}
			}
			firstScale := first.AmountScale
			if firstScale <= 0 {
				firstScale = defaultScale
			}
			openingScale := opening.AmountScale
			if openingScale <= 0 {
				openingScale = firstScale
			}
			lastScale := last.AmountScale
			if lastScale <= 0 {
				lastScale = defaultScale
			}
			row.AmountScale = lastScale
			row.OpeningBalance = opening.Balance * openingScale
			row.ClosingBalance = last.Balance * lastScale
			row.CurrentBalance = last.Balance * lastScale
			row.FirstSnapshotAt = &first.CapturedAt
			row.LastSnapshotAt = &last.CapturedAt
			row.Complete = true
			row.ConsumptionAmount = last.TodayCost
			row.Anomaly = row.ConsumptionAmount < 0
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Date == rows[j].Date {
			return rows[i].ProviderSlug < rows[j].ProviderSlug
		}
		return rows[i].Date > rows[j].Date
	})
	return rows
}

func (s *UpstreamBalanceConsumptionService) GetConfig(ctx context.Context) (UpstreamBalanceSamplerConfig, error) {
	if s == nil || s.settingRepo == nil {
		return defaultUpstreamBalanceSamplerConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamBalanceSamplerConfig)
	if err != nil {
		if err == ErrSettingNotFound {
			return defaultUpstreamBalanceSamplerConfig(), nil
		}
		return UpstreamBalanceSamplerConfig{}, fmt.Errorf("load upstream balance sampler config: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultUpstreamBalanceSamplerConfig(), nil
	}
	var config UpstreamBalanceSamplerConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return UpstreamBalanceSamplerConfig{}, infraerrors.InternalServer("UPSTREAM_BALANCE_SAMPLER_CONFIG_INVALID", "upstream balance sampler config is invalid")
	}
	return normalizeUpstreamBalanceSamplerConfig(config), nil
}

func (s *UpstreamBalanceConsumptionService) UpdateConfig(ctx context.Context, input UpstreamBalanceSamplerConfig) (UpstreamBalanceSamplerConfig, error) {
	if input.IntervalSeconds > 0 && input.IntervalSeconds < MinUpstreamBalanceSamplerIntervalSeconds {
		return UpstreamBalanceSamplerConfig{}, infraerrors.BadRequest("UPSTREAM_BALANCE_SAMPLER_INTERVAL_INVALID", fmt.Sprintf("interval_seconds must be at least %d", MinUpstreamBalanceSamplerIntervalSeconds))
	}
	config := normalizeUpstreamBalanceSamplerConfig(input)
	now := s.currentTime()
	config.UpdatedAt = &now
	return s.saveConfig(ctx, config)
}

func (s *UpstreamBalanceConsumptionService) AddRecharge(ctx context.Context, input UpstreamBalanceRechargeInput) (UpstreamBalanceRecharge, error) {
	if s == nil || s.store == nil {
		return UpstreamBalanceRecharge{}, infraerrors.ServiceUnavailable("UPSTREAM_BALANCE_STORE_UNAVAILABLE", "upstream balance store is unavailable")
	}
	input.ProviderSlug = strings.TrimSpace(input.ProviderSlug)
	input.Note = strings.TrimSpace(input.Note)
	if input.ProviderSlug == "" {
		return UpstreamBalanceRecharge{}, infraerrors.BadRequest("UPSTREAM_BALANCE_PROVIDER_REQUIRED", "provider_slug is required")
	}
	if input.Amount <= 0 {
		return UpstreamBalanceRecharge{}, infraerrors.BadRequest("UPSTREAM_BALANCE_RECHARGE_AMOUNT_INVALID", "amount must be greater than 0")
	}
	if input.AmountScale <= 0 {
		config, _ := s.GetConfig(ctx)
		input.AmountScale = upstreamBalanceAmountScale(config, input.ProviderSlug)
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = s.currentTime()
	}
	return s.store.AddRecharge(ctx, input)
}

func (s *UpstreamBalanceConsumptionService) GetOverview(ctx context.Context, days int) (UpstreamBalanceConsumptionOverview, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	config, err := s.GetConfig(ctx)
	if err != nil {
		return UpstreamBalanceConsumptionOverview{}, err
	}
	if s == nil || s.store == nil {
		return UpstreamBalanceConsumptionOverview{
			Config:                 config,
			Summaries:              map[string]UpstreamBalanceProviderSummary{},
			Rows:                   []UpstreamBalanceDailyRow{},
			Snapshots:              []UpstreamBalanceSnapshot{},
			LocalDailyConsumptions: []UpstreamLocalDailyConsumption{},
		}, nil
	}
	now := s.currentTime()
	loc := upstreamBalanceStatsLocation()
	localNow := now.In(loc)
	localEnd := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, 0, 0, 0, 0, loc)
	end := localEnd.UTC()
	start := localEnd.AddDate(0, 0, -days).UTC()
	snapshots, err := s.store.ListSnapshots(ctx, start, end)
	if err != nil {
		return UpstreamBalanceConsumptionOverview{}, fmt.Errorf("list upstream balance snapshots: %w", err)
	}
	periodSnapshots := append([]UpstreamBalanceSnapshot{}, snapshots...)
	openingSnapshots, err := s.store.ListSnapshotsBefore(ctx, start)
	if err != nil {
		return UpstreamBalanceConsumptionOverview{}, fmt.Errorf("list upstream balance opening snapshots: %w", err)
	}
	snapshots = append(openingSnapshots, snapshots...)
	recharges, err := s.store.ListRecharges(ctx, start, end)
	if err != nil {
		return UpstreamBalanceConsumptionOverview{}, fmt.Errorf("list upstream balance recharges: %w", err)
	}
	rows := buildUpstreamBalanceDailyRowsInLocation(snapshots, recharges, start, end, 1, loc)
	localDailyConsumptions, err := s.listLocalDailyConsumptions(ctx, start, end)
	if err != nil {
		return UpstreamBalanceConsumptionOverview{}, err
	}
	latest, _ := s.store.ListLatestSnapshots(ctx)
	summaries := buildUpstreamBalanceSummaries(rows, latest, now, loc)
	return UpstreamBalanceConsumptionOverview{
		Config:                 config,
		Summaries:              summaries,
		Rows:                   rows,
		Snapshots:              periodSnapshots,
		LocalDailyConsumptions: localDailyConsumptions,
	}, nil
}

func (s *UpstreamBalanceConsumptionService) RunSample(ctx context.Context) (UpstreamBalanceSamplerConfig, error) {
	config, err := s.GetConfig(ctx)
	if err != nil {
		return UpstreamBalanceSamplerConfig{}, err
	}
	now := s.currentTime()
	config.LastRunAt = &now
	if s == nil || s.store == nil || s.provider == nil {
		config.LastRunStatus = "failed"
		config.LastRunMessage = "upstream balance sampler dependencies are unavailable"
		saved, saveErr := s.saveConfig(ctx, config)
		if saveErr != nil {
			return UpstreamBalanceSamplerConfig{}, saveErr
		}
		return saved, infraerrors.ServiceUnavailable("UPSTREAM_BALANCE_SAMPLER_UNAVAILABLE", config.LastRunMessage)
	}
	providers, err := s.provider.ListProviders(ctx)
	if err != nil {
		config.LastRunStatus = "failed"
		config.LastRunMessage = err.Error()
		saved, saveErr := s.saveConfig(ctx, config)
		if saveErr != nil {
			return UpstreamBalanceSamplerConfig{}, saveErr
		}
		return saved, err
	}
	failures := []string{}
	for _, provider := range providers {
		if !provider.Enabled || provider.Slug == "" {
			continue
		}
		balance, err := s.provider.FetchProviderBalance(ctx, provider.Slug)
		cost, costErr := s.provider.FetchProviderTodayCost(ctx, provider.Slug, now)
		amountScale := upstreamBalanceAmountScale(config, provider.Slug)
		snapshot := UpstreamBalanceSnapshot{
			ProviderSlug: provider.Slug,
			ProviderName: provider.Name,
			ProviderType: provider.Type,
			AmountScale:  amountScale,
			Status:       "success",
			CapturedAt:   now,
		}
		if err != nil {
			snapshot.Status = "failed"
			snapshot.Error = err.Error()
			failures = append(failures, provider.Slug+": "+err.Error())
		} else {
			snapshot.ProviderName = balance.ProviderName
			snapshot.ProviderType = balance.ProviderType
			snapshot.Balance = balance.Balance
		}
		if costErr != nil {
			snapshot.Status = "failed"
			if snapshot.Error != "" {
				snapshot.Error += "; "
			}
			snapshot.Error += costErr.Error()
			failures = append(failures, provider.Slug+": "+costErr.Error())
		} else {
			if snapshot.ProviderName == "" {
				snapshot.ProviderName = cost.ProviderName
			}
			if snapshot.ProviderType == "" {
				snapshot.ProviderType = cost.ProviderType
			}
			snapshot.TodayCost = cost.TodayCost
		}
		if _, addErr := s.store.AddSnapshot(ctx, snapshot); addErr != nil {
			failures = append(failures, provider.Slug+": "+addErr.Error())
		}
	}
	if len(failures) > 0 {
		config.LastRunStatus = "failed"
		config.LastRunMessage = strings.Join(failures, "; ")
	} else {
		config.LastRunStatus = "success"
		config.LastRunMessage = ""
	}
	return s.saveConfig(ctx, config)
}

func buildUpstreamBalanceSummaries(rows []UpstreamBalanceDailyRow, latest []UpstreamBalanceSnapshot, now time.Time, loc *time.Location) map[string]UpstreamBalanceProviderSummary {
	if loc == nil {
		loc = time.UTC
	}
	today := now.In(loc).Format("2006-01-02")
	out := map[string]UpstreamBalanceProviderSummary{}
	for _, row := range rows {
		if row.Date != today {
			continue
		}
		out[row.ProviderSlug] = UpstreamBalanceProviderSummary{
			ProviderSlug:     row.ProviderSlug,
			ProviderName:     row.ProviderName,
			CurrentBalance:   row.CurrentBalance,
			TodayConsumption: row.ConsumptionAmount,
			AmountScale:      row.AmountScale,
			Complete:         row.Complete,
			Anomaly:          row.Anomaly,
			SnapshotCount:    row.SnapshotCount,
			LastSnapshotAt:   row.LastSnapshotAt,
		}
	}
	for _, snap := range latest {
		if snap.ProviderSlug == "" {
			continue
		}
		summary := out[snap.ProviderSlug]
		summary.ProviderSlug = snap.ProviderSlug
		if summary.ProviderName == "" {
			summary.ProviderName = snap.ProviderName
		}
		scale := snap.AmountScale
		if scale <= 0 {
			scale = 1
		}
		if summary.AmountScale == 0 {
			summary.AmountScale = scale
		}
		summary.LastSnapshotAt = &snap.CapturedAt
		if snap.Status == "failed" {
			summary.LastSnapshotError = snap.Error
			out[snap.ProviderSlug] = summary
			continue
		}
		summary.CurrentBalance = snap.Balance * scale
		out[snap.ProviderSlug] = summary
	}
	return out
}

func (s *UpstreamBalanceConsumptionService) listLocalDailyConsumptions(ctx context.Context, start, end time.Time) ([]UpstreamLocalDailyConsumption, error) {
	if s == nil || s.usageSource == nil {
		return []UpstreamLocalDailyConsumption{}, nil
	}
	rawRows, err := s.usageSource.GetGlobalDailyStatsAggregated(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("list local daily consumption: %w", err)
	}
	rows := make([]UpstreamLocalDailyConsumption, 0, len(rawRows))
	for _, raw := range rawRows {
		date, _ := raw["date"].(string)
		date = strings.TrimSpace(date)
		if date == "" {
			continue
		}
		rows = append(rows, UpstreamLocalDailyConsumption{
			Date:       date,
			ActualCost: float64FromAny(raw["total_actual_cost"]),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Date < rows[j].Date
	})
	return rows, nil
}

func float64FromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case json.Number:
		n, _ := v.Float64()
		return n
	case string:
		n, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n
	default:
		return 0
	}
}

func defaultUpstreamBalanceSamplerConfig() UpstreamBalanceSamplerConfig {
	return UpstreamBalanceSamplerConfig{
		Enabled:         false,
		IntervalSeconds: DefaultUpstreamBalanceSamplerIntervalSeconds,
	}
}

func normalizeUpstreamBalanceSamplerConfig(config UpstreamBalanceSamplerConfig) UpstreamBalanceSamplerConfig {
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = DefaultUpstreamBalanceSamplerIntervalSeconds
	}
	if len(config.ProviderAmountScales) > 0 {
		normalized := make(map[string]float64, len(config.ProviderAmountScales))
		for slug, scale := range config.ProviderAmountScales {
			slug = strings.TrimSpace(slug)
			if slug == "" || scale <= 0 {
				continue
			}
			normalized[slug] = scale
		}
		config.ProviderAmountScales = normalized
	}
	return config
}

func upstreamBalanceAmountScale(config UpstreamBalanceSamplerConfig, providerSlug string) float64 {
	if config.ProviderAmountScales != nil {
		if scale := config.ProviderAmountScales[strings.TrimSpace(providerSlug)]; scale > 0 {
			return scale
		}
	}
	return 1
}

func (s *UpstreamBalanceConsumptionService) saveConfig(ctx context.Context, config UpstreamBalanceSamplerConfig) (UpstreamBalanceSamplerConfig, error) {
	config = normalizeUpstreamBalanceSamplerConfig(config)
	if s == nil || s.settingRepo == nil {
		return config, nil
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return UpstreamBalanceSamplerConfig{}, fmt.Errorf("marshal upstream balance sampler config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamBalanceSamplerConfig, string(raw)); err != nil {
		return UpstreamBalanceSamplerConfig{}, fmt.Errorf("save upstream balance sampler config: %w", err)
	}
	return config, nil
}

func (s *UpstreamBalanceConsumptionService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func upstreamBalanceStatsLocation() *time.Location {
	loc, err := time.LoadLocation(upstreamBalanceStatsTimezone)
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return loc
}
