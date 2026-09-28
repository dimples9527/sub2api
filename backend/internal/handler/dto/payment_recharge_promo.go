package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

// PaymentHolidayPromoDTO is the admin-facing shape of the tiered recharge promotion.
type PaymentHolidayPromoDTO struct {
	Enabled bool                         `json:"enabled"`
	StartAt *int64                       `json:"start_at,omitempty"`
	EndAt   *int64                       `json:"end_at,omitempty"`
	Tiers   []PaymentHolidayPromoTierDTO `json:"tiers"`
}

type PaymentHolidayPromoTierDTO struct {
	Threshold float64 `json:"threshold"`
	BonusRate float64 `json:"bonus_rate"`
}

// PaymentHolidayPromoFromService maps the service promo config into the DTO (nil-safe).
func PaymentHolidayPromoFromService(promo *service.RechargePromo) *PaymentHolidayPromoDTO {
	if promo == nil {
		return nil
	}
	tiers := make([]PaymentHolidayPromoTierDTO, 0, len(promo.Tiers))
	for _, tier := range promo.Tiers {
		tiers = append(tiers, PaymentHolidayPromoTierDTO{Threshold: tier.Threshold, BonusRate: tier.BonusRate})
	}
	return &PaymentHolidayPromoDTO{
		Enabled: promo.Enabled,
		StartAt: promo.StartAt,
		EndAt:   promo.EndAt,
		Tiers:   tiers,
	}
}

// ToService maps the DTO into the service promo config (nil-safe).
func (d *PaymentHolidayPromoDTO) ToService() *service.RechargePromo {
	if d == nil {
		return nil
	}
	tiers := make([]service.RechargePromoTier, 0, len(d.Tiers))
	for _, tier := range d.Tiers {
		tiers = append(tiers, service.RechargePromoTier{Threshold: tier.Threshold, BonusRate: tier.BonusRate})
	}
	return &service.RechargePromo{
		Enabled: d.Enabled,
		StartAt: d.StartAt,
		EndAt:   d.EndAt,
		Tiers:   tiers,
	}
}
