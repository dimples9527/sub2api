package service

type RechargeAmountOption struct {
	PayAmount    float64 `json:"pay_amount"`
	CreditAmount float64 `json:"credit_amount"`
}

func BuildRechargeAmountOptions(cfg *PaymentConfig) []RechargeAmountOption {
	if cfg == nil {
		return nil
	}
	amounts := normalizeRechargeOptionAmounts(cfg.RechargeOptions)
	options := make([]RechargeAmountOption, 0, len(amounts))
	for _, amount := range amounts {
		options = append(options, RechargeAmountOption{
			PayAmount:    amount,
			CreditAmount: amount,
		})
	}
	return options
}
