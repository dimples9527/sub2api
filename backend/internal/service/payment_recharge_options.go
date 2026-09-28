package service

import "time"

type RechargeAmountOption struct {
	PayAmount    float64 `json:"pay_amount"`
	CreditAmount float64 `json:"credit_amount"`
}

func BuildRechargeAmountOptions(cfg *PaymentConfig, now time.Time) []RechargeAmountOption {
	if cfg == nil {
		return nil
	}
	amounts := normalizeRechargeOptionAmounts(cfg.RechargeOptions)
	options := make([]RechargeAmountOption, 0, len(amounts))
	for _, amount := range amounts {
		options = append(options, RechargeAmountOption{
			PayAmount:    amount,
			CreditAmount: calculateCreditedBalance(amount, EffectiveRechargeMultiplier(cfg, amount, now)),
		})
	}
	return options
}
