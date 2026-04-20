//go:build unit

package service

import "testing"

func TestBuildRechargeAmountOptions(t *testing.T) {
	t.Parallel()

	cfg := &PaymentConfig{
		RechargeOptions:     []float64{2, 5, 10, 20},
		IntroRechargePay:    2,
		IntroRechargeCredit: 10,
	}

	t.Run("intro available replaces normal intro amounts", func(t *testing.T) {
		t.Parallel()

		options := buildRechargeAmountOptions(cfg, true)
		if len(options) == 0 {
			t.Fatal("expected intro options")
		}
		if options[0].PayAmount != 2 || options[0].CreditAmount != 10 {
			t.Fatalf("first option = %+v, want pay 2.00 credit 10.00", options[0])
		}
		if options[0].OriginalPayAmount == nil || *options[0].OriginalPayAmount != 10 {
			t.Fatalf("expected original pay amount 10, got %+v", options[0].OriginalPayAmount)
		}
		if len(options) != 3 || options[1].PayAmount != 5 || options[1].CreditAmount != 5 || options[2].PayAmount != 20 || options[2].CreditAmount != 20 {
			t.Fatalf("expected intro option plus normal 5/20 options, got %+v", options)
		}
	})

	t.Run("regular options are returned after intro is consumed", func(t *testing.T) {
		t.Parallel()

		options := buildRechargeAmountOptions(cfg, false)
		if len(options) != 4 {
			t.Fatalf("expected 4 regular options, got %+v", options)
		}
		if options[0].PayAmount != 2 || options[1].PayAmount != 5 || options[2].PayAmount != 10 {
			t.Fatalf("unexpected first three options: %+v", options[:3])
		}
	})

	t.Run("invalid intro config falls back to regular options", func(t *testing.T) {
		t.Parallel()

		options := buildRechargeAmountOptions(&PaymentConfig{
			RechargeOptions:     []float64{6, 30},
			IntroRechargePay:    10,
			IntroRechargeCredit: 0,
		}, true)
		if len(options) != 2 || options[0].PayAmount != 6 || options[1].PayAmount != 30 {
			t.Fatalf("unexpected options without intro: %+v", options)
		}
	})
}

func TestResolveRechargeAmounts(t *testing.T) {
	t.Parallel()

	cfg := &PaymentConfig{
		IntroRechargePay:    3,
		IntroRechargeCredit: 9,
	}

	t.Run("configured intro amount credits more balance before it is claimed", func(t *testing.T) {
		t.Parallel()

		credited, charged, applied := resolveRechargeAmounts(3, cfg, true)
		if !applied {
			t.Fatal("expected intro offer to apply")
		}
		if credited != 9 || charged != 3 {
			t.Fatalf("resolveRechargeAmounts = credited %.2f charged %.2f, want 9 and 3", credited, charged)
		}
	})

	t.Run("same amount becomes normal recharge after intro is claimed", func(t *testing.T) {
		t.Parallel()

		credited, charged, applied := resolveRechargeAmounts(3, cfg, false)
		if applied {
			t.Fatal("did not expect intro offer to apply")
		}
		if credited != 3 || charged != 3 {
			t.Fatalf("resolveRechargeAmounts = credited %.2f charged %.2f, want 3 and 3", credited, charged)
		}
	})

	t.Run("non-discount intro config does not apply promo", func(t *testing.T) {
		t.Parallel()

		credited, charged, applied := resolveRechargeAmounts(3, &PaymentConfig{IntroRechargePay: 3, IntroRechargeCredit: 3}, true)
		if applied {
			t.Fatal("did not expect intro offer to apply when credit is not greater than pay")
		}
		if credited != 3 || charged != 3 {
			t.Fatalf("resolveRechargeAmounts = credited %.2f charged %.2f, want 3 and 3", credited, charged)
		}
	})
}
