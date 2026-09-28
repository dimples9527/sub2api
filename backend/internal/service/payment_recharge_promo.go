package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// maxRechargePromoBonusRate caps a single tier's bonus rate (5.0 = +500%).
const maxRechargePromoBonusRate = 5.0

// RechargePromoTier is one ladder step: recharge amount >= Threshold grants BonusRate extra credit.
type RechargePromoTier struct {
	Threshold float64 `json:"threshold"`
	BonusRate float64 `json:"bonus_rate"`
}

// RechargePromo is a time-boxed, tiered "recharge more get more" promotion that stacks
// additively on top of the base BalanceRechargeMultiplier.
type RechargePromo struct {
	Enabled bool                `json:"enabled"`
	StartAt *int64              `json:"start_at,omitempty"`
	EndAt   *int64              `json:"end_at,omitempty"`
	Tiers   []RechargePromoTier `json:"tiers,omitempty"`
}

func parseRechargePromo(raw string) *RechargePromo {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var promo RechargePromo
	if err := json.Unmarshal([]byte(raw), &promo); err != nil {
		return nil
	}
	promo.Tiers = normalizeRechargePromoTiers(promo.Tiers)
	return &promo
}

func formatRechargePromo(promo *RechargePromo) string {
	if promo == nil {
		return ""
	}
	out := *promo
	out.Tiers = normalizeRechargePromoTiers(out.Tiers)
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}

// normalizeRechargePromoTiers drops invalid tiers, dedupes thresholds, and sorts ascending.
func normalizeRechargePromoTiers(tiers []RechargePromoTier) []RechargePromoTier {
	if len(tiers) == 0 {
		return nil
	}
	normalized := make([]RechargePromoTier, 0, len(tiers))
	seen := make(map[string]struct{}, len(tiers))
	for _, tier := range tiers {
		if math.IsNaN(tier.Threshold) || math.IsInf(tier.Threshold, 0) || tier.Threshold < 0 {
			continue
		}
		if math.IsNaN(tier.BonusRate) || math.IsInf(tier.BonusRate, 0) || tier.BonusRate < 0 {
			continue
		}
		threshold := math.Round(tier.Threshold*100) / 100
		key := strconv.FormatFloat(threshold, 'f', 2, 64)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, RechargePromoTier{Threshold: threshold, BonusRate: tier.BonusRate})
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].Threshold < normalized[j].Threshold })
	return normalized
}

func validateRechargePromo(promo *RechargePromo) error {
	if promo == nil {
		return nil
	}
	if promo.StartAt != nil && promo.EndAt != nil && *promo.EndAt < *promo.StartAt {
		return infraerrors.BadRequest("INVALID_RECHARGE_PROMO", "promo end time must not be earlier than start time")
	}
	seen := make(map[string]struct{}, len(promo.Tiers))
	for _, tier := range promo.Tiers {
		if math.IsNaN(tier.Threshold) || math.IsInf(tier.Threshold, 0) || tier.Threshold < 0 {
			return infraerrors.BadRequest("INVALID_RECHARGE_PROMO", "promo tier threshold must be a non-negative number")
		}
		if math.IsNaN(tier.BonusRate) || math.IsInf(tier.BonusRate, 0) || tier.BonusRate < 0 || tier.BonusRate > maxRechargePromoBonusRate {
			return infraerrors.BadRequest("INVALID_RECHARGE_PROMO", "promo tier bonus rate out of range")
		}
		key := strconv.FormatFloat(math.Round(tier.Threshold*100)/100, 'f', 2, 64)
		if _, ok := seen[key]; ok {
			return infraerrors.BadRequest("INVALID_RECHARGE_PROMO", "promo tiers must have distinct thresholds")
		}
		seen[key] = struct{}{}
	}
	return nil
}

// IsActive reports whether the promo is enabled, has tiers, and now falls within its optional window.
func (p *RechargePromo) IsActive(now time.Time) bool {
	if p == nil || !p.Enabled || len(p.Tiers) == 0 {
		return false
	}
	ts := now.Unix()
	if p.StartAt != nil && ts < *p.StartAt {
		return false
	}
	if p.EndAt != nil && ts > *p.EndAt {
		return false
	}
	return true
}

// BonusRateFor returns the bonus rate of the highest tier whose threshold <= amount, or 0.
// Tiers are kept sorted ascending by normalizeRechargePromoTiers.
func (p *RechargePromo) BonusRateFor(amount float64) float64 {
	if p == nil {
		return 0
	}
	rate := 0.0
	for _, tier := range p.Tiers {
		if amount+1e-9 >= tier.Threshold {
			rate = tier.BonusRate
			continue
		}
		break
	}
	return rate
}

// EffectiveRechargeMultiplier stacks the active promo bonus additively on the base multiplier.
func EffectiveRechargeMultiplier(cfg *PaymentConfig, amount float64, now time.Time) float64 {
	base := defaultBalanceRechargeMultiplier
	if cfg != nil {
		base = normalizeBalanceRechargeMultiplier(cfg.BalanceRechargeMultiplier)
	}
	if cfg == nil || !cfg.HolidayPromo.IsActive(now) {
		return base
	}
	return base + cfg.HolidayPromo.BonusRateFor(amount)
}

// GetRechargeHolidayPromo returns the standalone recharge holiday promo config,
// or nil when it has never been configured / is empty.
func (s *PaymentConfigService) GetRechargeHolidayPromo(ctx context.Context) (*RechargePromo, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingRechargeHolidayPromo)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get recharge holiday promo: %w", err)
	}
	return parseRechargePromo(value), nil
}

// UpdateRechargeHolidayPromo validates and persists the recharge holiday promo config
// under its single setting key (nil clears it).
func (s *PaymentConfigService) UpdateRechargeHolidayPromo(ctx context.Context, promo *RechargePromo) error {
	if err := validateRechargePromo(promo); err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, SettingRechargeHolidayPromo, formatRechargePromo(promo))
}
