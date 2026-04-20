package service

import (
	"context"
	"math"

	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const (
	introRechargeRequestAmount = 2.0
	introRechargeCreditAmount  = 10.0
	introRechargeCodePrefix    = "PROMO10-"
)

type RechargeAmountOption struct {
	PayAmount         float64  `json:"pay_amount"`
	CreditAmount      float64  `json:"credit_amount"`
	OriginalPayAmount *float64 `json:"original_pay_amount,omitempty"`
	OneTime           bool     `json:"one_time,omitempty"`
}

func buildRechargeAmountOptions(cfg *PaymentConfig, introAvailable bool) []RechargeAmountOption {
	amounts := defaultRechargeOptionAmounts
	if cfg != nil && len(cfg.RechargeOptions) > 0 {
		amounts = cfg.RechargeOptions
	}

	options := make([]RechargeAmountOption, 0, len(amounts)+1)
	introEnabled := isIntroRechargeEnabled(cfg)
	introPay, introCredit := introRechargeValues(cfg)
	if introAvailable && introEnabled {
		originalPay := introCredit
		options = append(options, RechargeAmountOption{
			PayAmount:         introPay,
			CreditAmount:      introCredit,
			OriginalPayAmount: &originalPay,
			OneTime:           true,
		})
	}

	for _, amount := range amounts {
		if amount <= 0 {
			continue
		}
		if introAvailable && introEnabled && (sameRechargeAmount(amount, introPay) || sameRechargeAmount(amount, introCredit)) {
			continue
		}
		options = append(options, RechargeAmountOption{PayAmount: amount, CreditAmount: amount})
	}
	return options
}

func resolveRechargeAmounts(requestedAmount float64, cfg *PaymentConfig, introAvailable bool) (creditedAmount, chargedAmount float64, appliedIntro bool) {
	if introAvailable && isIntroRechargeEnabled(cfg) {
		introPay, introCredit := introRechargeValues(cfg)
		if math.Abs(requestedAmount-introPay) <= amountToleranceCNY {
			return introCredit, introPay, true
		}
	}
	return requestedAmount, requestedAmount, false
}

func isIntroRechargeEnabled(cfg *PaymentConfig) bool {
	introPay, introCredit := introRechargeValues(cfg)
	return introPay > 0 && introCredit > introPay
}

func introRechargeValues(cfg *PaymentConfig) (payAmount, creditAmount float64) {
	if cfg == nil {
		return introRechargeRequestAmount, introRechargeCreditAmount
	}
	payAmount = cfg.IntroRechargePay
	creditAmount = cfg.IntroRechargeCredit
	if payAmount <= 0 {
		payAmount = introRechargeRequestAmount
	}
	if creditAmount <= 0 {
		creditAmount = introRechargeCreditAmount
	}
	return payAmount, creditAmount
}

func sameRechargeAmount(a, b float64) bool {
	return math.Abs(a-b) <= amountToleranceCNY
}

func (s *PaymentService) IntroRechargeAvailable(ctx context.Context, userID int64) (bool, error) {
	claimed, err := s.hasClaimedIntroRecharge(ctx, userID)
	if err != nil {
		return false, err
	}
	return !claimed, nil
}

func (s *PaymentService) GetRechargeAmountOptions(ctx context.Context, userID int64) ([]RechargeAmountOption, error) {
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return nil, err
	}
	introAvailable, err := s.IntroRechargeAvailable(ctx, userID)
	if err != nil {
		return nil, err
	}
	return buildRechargeAmountOptions(cfg, introAvailable), nil
}

func (s *PaymentService) hasClaimedIntroRecharge(ctx context.Context, userID int64) (bool, error) {
	count, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.OrderTypeEQ(payment.OrderTypeBalance),
			paymentorder.RechargeCodeHasPrefix(introRechargeCodePrefix),
			paymentorder.Not(paymentorder.StatusIn(OrderStatusCancelled, OrderStatusExpired, OrderStatusFailed)),
		).
		Limit(1).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
