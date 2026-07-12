package sim

import "testing"

func TestMandatorySettlementUsesReserveThenCreditAndKeepsIdentity(t *testing.T) {
	funding := FundMandatoryGap(50, 100, 40, 0.5, 1_000, 100)
	if funding.RestrictedReserveReleased != 20 || funding.CreditDraw != 30 {
		t.Fatalf("funding = %+v, want release=20 draw=30", funding)
	}
	settlement := SettleMandatory(MandatorySettlementInput{
		Obligations: MandatoryObligations{
			Salary:       100,
			Distribution: DistributionDue{Gross: 10, PayrollTax: 2},
		},
		CashTotalBeforePayments:    102,
		RestrictedDistributionCash: 12,
		RestrictedReserveCash:      40,
		Funding:                    funding,
	})
	if settlement.Paid.Salary != 100 || settlement.Paid.Distribution.Total() != 12 || settlement.UnpaidTotal != 0 {
		t.Fatalf("settlement = %+v", settlement)
	}
	if settlement.CashTotalAfterPayments != 20 || settlement.RestrictedReserveCash != 20 || settlement.RestrictedDistributionCash != 0 {
		t.Fatalf("closing settlement = %+v", settlement)
	}
}

func TestMandatoryPriorityStopsAfterUnpaidEarlierItem(t *testing.T) {
	settlement := SettleMandatory(MandatorySettlementInput{
		Obligations: MandatoryObligations{
			Salary:       100,
			FixedCosts:   50,
			Distribution: DistributionDue{Gross: 20},
		},
		CashTotalBeforePayments:    100,
		RestrictedDistributionCash: 20,
		Funding: LiquidityFunding{
			GeneralCashAvailable: 80,
		},
	})
	if settlement.Paid.Salary != 80 || settlement.Paid.FixedCosts != 0 || settlement.Paid.Distribution.Total() != 0 {
		t.Fatalf("priority settlement = %+v", settlement)
	}
	if settlement.UnpaidTotal != 90 {
		t.Fatalf("unpaid = %g, want 90", settlement.UnpaidTotal)
	}
}

func TestMandatorySettlementCarriesNonQueuedArrearsBeforeCurrentCosts(t *testing.T) {
	settlement := SettleMandatory(MandatorySettlementInput{
		Obligations: MandatoryObligations{
			PriorArrears: 40,
			Salary:       100,
			Interest:     10,
			Principal:    20,
		},
		CashTotalBeforePayments: 70,
		Funding: LiquidityFunding{
			GeneralCashAvailable: 70,
		},
	})
	if settlement.Paid.PriorArrears != 40 || settlement.Paid.Salary != 30 {
		t.Fatalf("arrears priority settlement = %+v", settlement)
	}
	if settlement.UnpaidCarryForward != 80 {
		t.Fatalf("carry-forward unpaid = %g, want 80", settlement.UnpaidCarryForward)
	}
	if settlement.UnpaidTotal != 100 {
		t.Fatalf("total unpaid = %g, want 100", settlement.UnpaidTotal)
	}
}
