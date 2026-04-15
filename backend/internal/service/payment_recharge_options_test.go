//go:build unit

package service

import "testing"

func TestBuildRechargeAmountOptions(t *testing.T) {
	t.Parallel()

	t.Run("intro available replaces normal 10 button and keeps 5", func(t *testing.T) {
		t.Parallel()

		options := buildRechargeAmountOptions(true)
		if len(options) == 0 {
			t.Fatal("expected intro options")
		}
		if options[0].PayAmount != introRechargeRequestAmount || options[0].CreditAmount != introRechargeCreditAmount {
			t.Fatalf("first option = %+v, want pay %.2f credit %.2f", options[0], introRechargeRequestAmount, introRechargeCreditAmount)
		}
		if options[0].OriginalPayAmount == nil || *options[0].OriginalPayAmount != introRechargeCreditAmount {
			t.Fatalf("expected original pay amount %.2f, got %+v", introRechargeCreditAmount, options[0].OriginalPayAmount)
		}
		if options[1].PayAmount != 5 || options[1].CreditAmount != 5 {
			t.Fatalf("expected second option to be normal 5 recharge, got %+v", options[1])
		}
		for _, option := range options {
			if option.PayAmount == 10 && option.CreditAmount == 10 {
				t.Fatalf("did not expect normal 10 option when intro offer is available: %+v", option)
			}
		}
	})

	t.Run("regular options include 2 5 10 after intro is consumed", func(t *testing.T) {
		t.Parallel()

		options := buildRechargeAmountOptions(false)
		if len(options) < 3 {
			t.Fatalf("expected regular options, got %+v", options)
		}
		if options[0].PayAmount != 2 || options[1].PayAmount != 5 || options[2].PayAmount != 10 {
			t.Fatalf("unexpected first three options: %+v", options[:3])
		}
	})
}

func TestResolveRechargeAmounts(t *testing.T) {
	t.Parallel()

	t.Run("intro amount credits more balance before it is claimed", func(t *testing.T) {
		t.Parallel()

		credited, charged, applied := resolveRechargeAmounts(2, true)
		if !applied {
			t.Fatal("expected intro offer to apply")
		}
		if credited != 10 || charged != 2 {
			t.Fatalf("resolveRechargeAmounts = credited %.2f charged %.2f, want 10 and 2", credited, charged)
		}
	})

	t.Run("same amount becomes normal recharge after intro is claimed", func(t *testing.T) {
		t.Parallel()

		credited, charged, applied := resolveRechargeAmounts(2, false)
		if applied {
			t.Fatal("did not expect intro offer to apply")
		}
		if credited != 2 || charged != 2 {
			t.Fatalf("resolveRechargeAmounts = credited %.2f charged %.2f, want 2 and 2", credited, charged)
		}
	})
}
