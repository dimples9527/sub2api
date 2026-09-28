package service

import (
	"math"
	"testing"
	"time"
)

func promoApproxEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestParseRechargePromo(t *testing.T) {
	t.Parallel()

	if got := parseRechargePromo(""); got != nil {
		t.Fatalf("empty string: got %+v, want nil", got)
	}
	if got := parseRechargePromo("   "); got != nil {
		t.Fatalf("blank string: got %+v, want nil", got)
	}
	if got := parseRechargePromo("{not json"); got != nil {
		t.Fatalf("invalid json: got %+v, want nil", got)
	}

	raw := `{"enabled":true,"start_at":100,"end_at":200,"tiers":[{"threshold":500,"bonus_rate":0.12},{"threshold":100,"bonus_rate":0.05},{"threshold":100,"bonus_rate":0.99}]}`
	got := parseRechargePromo(raw)
	if got == nil {
		t.Fatal("valid json: got nil")
	}
	if !got.Enabled || got.StartAt == nil || *got.StartAt != 100 || got.EndAt == nil || *got.EndAt != 200 {
		t.Fatalf("fields not parsed: %+v", got)
	}
	// tiers deduped by threshold (first wins) and sorted ascending
	if len(got.Tiers) != 2 {
		t.Fatalf("tiers len = %d, want 2 (%+v)", len(got.Tiers), got.Tiers)
	}
	if got.Tiers[0].Threshold != 100 || !promoApproxEqual(got.Tiers[0].BonusRate, 0.05) {
		t.Fatalf("tier[0] = %+v, want {100, 0.05}", got.Tiers[0])
	}
	if got.Tiers[1].Threshold != 500 || !promoApproxEqual(got.Tiers[1].BonusRate, 0.12) {
		t.Fatalf("tier[1] = %+v, want {500, 0.12}", got.Tiers[1])
	}
}

func TestFormatRechargePromo(t *testing.T) {
	t.Parallel()

	if got := formatRechargePromo(nil); got != "" {
		t.Fatalf("nil: got %q, want empty", got)
	}

	// round-trip: format then parse yields equivalent normalized promo
	promo := &RechargePromo{
		Enabled: true,
		StartAt: int64Ptr(1730390400),
		EndAt:   int64Ptr(1730995140),
		Tiers: []RechargePromoTier{
			{Threshold: 1000, BonusRate: 0.20},
			{Threshold: 100, BonusRate: 0.05},
			{Threshold: 500, BonusRate: 0.12},
		},
	}
	raw := formatRechargePromo(promo)
	if raw == "" {
		t.Fatal("format returned empty for valid promo")
	}
	back := parseRechargePromo(raw)
	if back == nil {
		t.Fatal("re-parse returned nil")
	}
	if !back.Enabled || *back.StartAt != 1730390400 || *back.EndAt != 1730995140 {
		t.Fatalf("round-trip fields mismatch: %+v", back)
	}
	if len(back.Tiers) != 3 || back.Tiers[0].Threshold != 100 || back.Tiers[1].Threshold != 500 || back.Tiers[2].Threshold != 1000 {
		t.Fatalf("round-trip tiers not sorted: %+v", back.Tiers)
	}
}

func TestNormalizeRechargePromoTiers(t *testing.T) {
	t.Parallel()

	if got := normalizeRechargePromoTiers(nil); got != nil {
		t.Fatalf("nil tiers: got %+v, want nil", got)
	}

	tiers := []RechargePromoTier{
		{Threshold: math.NaN(), BonusRate: 0.1},
		{Threshold: math.Inf(1), BonusRate: 0.1},
		{Threshold: -5, BonusRate: 0.1},
		{Threshold: 100, BonusRate: math.NaN()},
		{Threshold: 100, BonusRate: -0.1},
		{Threshold: 100.129, BonusRate: 0.05},
		{Threshold: 100.13, BonusRate: 0.99},
		{Threshold: 20, BonusRate: 0.02},
	}
	got := normalizeRechargePromoTiers(tiers)
	if len(got) != 2 {
		t.Fatalf("normalized len = %d, want 2 (%+v)", len(got), got)
	}
	// 100.129 rounds to 100.13; the later 100.13 is a dup and dropped; sorted ascending
	if got[0].Threshold != 20 || !promoApproxEqual(got[0].BonusRate, 0.02) {
		t.Fatalf("tier[0] = %+v, want {20, 0.02}", got[0])
	}
	if got[1].Threshold != 100.13 || !promoApproxEqual(got[1].BonusRate, 0.05) {
		t.Fatalf("tier[1] = %+v, want {100.13, 0.05}", got[1])
	}
}

func TestValidateRechargePromo(t *testing.T) {
	t.Parallel()

	if err := validateRechargePromo(nil); err != nil {
		t.Fatalf("nil promo: unexpected error %v", err)
	}

	valid := &RechargePromo{
		Enabled: true,
		StartAt: int64Ptr(100),
		EndAt:   int64Ptr(200),
		Tiers:   []RechargePromoTier{{Threshold: 100, BonusRate: 0.05}, {Threshold: 500, BonusRate: 0.12}},
	}
	if err := validateRechargePromo(valid); err != nil {
		t.Fatalf("valid promo: unexpected error %v", err)
	}

	// end before start
	if err := validateRechargePromo(&RechargePromo{StartAt: int64Ptr(200), EndAt: int64Ptr(100)}); err == nil {
		t.Fatal("end < start: expected error")
	}
	// equal start/end is allowed
	if err := validateRechargePromo(&RechargePromo{StartAt: int64Ptr(100), EndAt: int64Ptr(100)}); err != nil {
		t.Fatalf("start == end: unexpected error %v", err)
	}
	// negative threshold
	if err := validateRechargePromo(&RechargePromo{Tiers: []RechargePromoTier{{Threshold: -1, BonusRate: 0.1}}}); err == nil {
		t.Fatal("negative threshold: expected error")
	}
	// bonus rate out of range
	if err := validateRechargePromo(&RechargePromo{Tiers: []RechargePromoTier{{Threshold: 100, BonusRate: maxRechargePromoBonusRate + 0.1}}}); err == nil {
		t.Fatal("bonus rate too high: expected error")
	}
	if err := validateRechargePromo(&RechargePromo{Tiers: []RechargePromoTier{{Threshold: 100, BonusRate: -0.01}}}); err == nil {
		t.Fatal("negative bonus rate: expected error")
	}
	// duplicate thresholds (after rounding to 2dp)
	if err := validateRechargePromo(&RechargePromo{Tiers: []RechargePromoTier{{Threshold: 100.001, BonusRate: 0.05}, {Threshold: 100.004, BonusRate: 0.06}}}); err == nil {
		t.Fatal("duplicate thresholds: expected error")
	}
}

func TestRechargePromoIsActive(t *testing.T) {
	t.Parallel()

	now := time.Unix(150, 0)
	tiers := []RechargePromoTier{{Threshold: 100, BonusRate: 0.05}}

	var nilPromo *RechargePromo
	if nilPromo.IsActive(now) {
		t.Fatal("nil promo: want inactive")
	}
	if (&RechargePromo{Enabled: false, Tiers: tiers}).IsActive(now) {
		t.Fatal("disabled: want inactive")
	}
	if (&RechargePromo{Enabled: true}).IsActive(now) {
		t.Fatal("no tiers: want inactive")
	}
	// open-ended window
	if !(&RechargePromo{Enabled: true, Tiers: tiers}).IsActive(now) {
		t.Fatal("no window: want active")
	}
	// before start
	if (&RechargePromo{Enabled: true, StartAt: int64Ptr(151), Tiers: tiers}).IsActive(now) {
		t.Fatal("before start: want inactive")
	}
	// after end
	if (&RechargePromo{Enabled: true, EndAt: int64Ptr(149), Tiers: tiers}).IsActive(now) {
		t.Fatal("after end: want inactive")
	}
	// inclusive boundaries
	if !(&RechargePromo{Enabled: true, StartAt: int64Ptr(150), EndAt: int64Ptr(150), Tiers: tiers}).IsActive(now) {
		t.Fatal("on boundary: want active")
	}
}

func TestRechargePromoBonusRateFor(t *testing.T) {
	t.Parallel()

	var nilPromo *RechargePromo
	if nilPromo.BonusRateFor(100) != 0 {
		t.Fatal("nil promo: want 0")
	}

	promo := &RechargePromo{Tiers: normalizeRechargePromoTiers([]RechargePromoTier{
		{Threshold: 100, BonusRate: 0.05},
		{Threshold: 500, BonusRate: 0.12},
		{Threshold: 1000, BonusRate: 0.20},
	})}

	cases := []struct {
		amount float64
		want   float64
	}{
		{50, 0},     // below all tiers
		{100, 0.05}, // exactly lowest tier
		{499.99, 0.05},
		{500, 0.12}, // exactly middle tier
		{999, 0.12},
		{1000, 0.20}, // exactly highest tier
		{5000, 0.20}, // above all tiers -> highest
	}
	for _, tc := range cases {
		if got := promo.BonusRateFor(tc.amount); !promoApproxEqual(got, tc.want) {
			t.Fatalf("BonusRateFor(%v) = %v, want %v", tc.amount, got, tc.want)
		}
	}
}

func TestEffectiveRechargeMultiplier(t *testing.T) {
	t.Parallel()

	now := time.Unix(150, 0)
	tiers := []RechargePromoTier{{Threshold: 100, BonusRate: 0.05}, {Threshold: 500, BonusRate: 0.12}}

	// nil cfg -> default base multiplier
	if got := EffectiveRechargeMultiplier(nil, 100, now); !promoApproxEqual(got, defaultBalanceRechargeMultiplier) {
		t.Fatalf("nil cfg = %v, want %v", got, defaultBalanceRechargeMultiplier)
	}

	// no promo -> base only
	baseCfg := &PaymentConfig{BalanceRechargeMultiplier: 1.2}
	if got := EffectiveRechargeMultiplier(baseCfg, 100, now); !promoApproxEqual(got, 1.2) {
		t.Fatalf("no promo = %v, want 1.2", got)
	}

	// active promo stacks additively on base
	activeCfg := &PaymentConfig{
		BalanceRechargeMultiplier: 1.0,
		HolidayPromo:              &RechargePromo{Enabled: true, Tiers: tiers},
	}
	if got := EffectiveRechargeMultiplier(activeCfg, 100, now); !promoApproxEqual(got, 1.05) {
		t.Fatalf("active promo @100 = %v, want 1.05", got)
	}
	if got := EffectiveRechargeMultiplier(activeCfg, 500, now); !promoApproxEqual(got, 1.12) {
		t.Fatalf("active promo @500 = %v, want 1.12", got)
	}
	if got := EffectiveRechargeMultiplier(activeCfg, 50, now); !promoApproxEqual(got, 1.0) {
		t.Fatalf("active promo below tiers @50 = %v, want 1.0", got)
	}

	// additive on non-default base
	stackedCfg := &PaymentConfig{
		BalanceRechargeMultiplier: 1.2,
		HolidayPromo:              &RechargePromo{Enabled: true, Tiers: tiers},
	}
	if got := EffectiveRechargeMultiplier(stackedCfg, 500, now); !promoApproxEqual(got, 1.32) {
		t.Fatalf("stacked base+bonus @500 = %v, want 1.32", got)
	}

	// inactive window -> base only
	inactiveCfg := &PaymentConfig{
		BalanceRechargeMultiplier: 1.0,
		HolidayPromo:              &RechargePromo{Enabled: true, EndAt: int64Ptr(149), Tiers: tiers},
	}
	if got := EffectiveRechargeMultiplier(inactiveCfg, 500, now); !promoApproxEqual(got, 1.0) {
		t.Fatalf("inactive promo = %v, want 1.0", got)
	}
}
