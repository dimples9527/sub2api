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

func buildRechargeAmountOptions(introAvailable bool) []RechargeAmountOption {
	if introAvailable {
		originalPay := introRechargeCreditAmount
		return []RechargeAmountOption{
			{PayAmount: introRechargeRequestAmount, CreditAmount: introRechargeCreditAmount, OriginalPayAmount: &originalPay, OneTime: true},
			{PayAmount: 5, CreditAmount: 5},
			{PayAmount: 20, CreditAmount: 20},
			{PayAmount: 50, CreditAmount: 50},
			{PayAmount: 100, CreditAmount: 100},
			{PayAmount: 200, CreditAmount: 200},
			{PayAmount: 500, CreditAmount: 500},
			{PayAmount: 1000, CreditAmount: 1000},
			{PayAmount: 2000, CreditAmount: 2000},
			{PayAmount: 5000, CreditAmount: 5000},
		}
	}

	return []RechargeAmountOption{
		{PayAmount: 2, CreditAmount: 2},
		{PayAmount: 5, CreditAmount: 5},
		{PayAmount: 10, CreditAmount: 10},
		{PayAmount: 20, CreditAmount: 20},
		{PayAmount: 50, CreditAmount: 50},
		{PayAmount: 100, CreditAmount: 100},
		{PayAmount: 200, CreditAmount: 200},
		{PayAmount: 500, CreditAmount: 500},
		{PayAmount: 1000, CreditAmount: 1000},
		{PayAmount: 2000, CreditAmount: 2000},
		{PayAmount: 5000, CreditAmount: 5000},
	}
}

func resolveRechargeAmounts(requestedAmount float64, introAvailable bool) (creditedAmount, chargedAmount float64, appliedIntro bool) {
	if introAvailable && math.Abs(requestedAmount-introRechargeRequestAmount) <= amountToleranceCNY {
		return introRechargeCreditAmount, introRechargeRequestAmount, true
	}
	return requestedAmount, requestedAmount, false
}

func (s *PaymentService) IntroRechargeAvailable(ctx context.Context, userID int64) (bool, error) {
	claimed, err := s.hasClaimedIntroRecharge(ctx, userID)
	if err != nil {
		return false, err
	}
	return !claimed, nil
}

func (s *PaymentService) GetRechargeAmountOptions(ctx context.Context, userID int64) ([]RechargeAmountOption, error) {
	introAvailable, err := s.IntroRechargeAvailable(ctx, userID)
	if err != nil {
		return nil, err
	}
	return buildRechargeAmountOptions(introAvailable), nil
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
