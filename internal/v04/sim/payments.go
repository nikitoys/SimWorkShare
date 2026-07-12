package sim

import "math"

type MandatoryObligations struct {
	PriorArrears      float64
	Salary            float64
	SalaryPayrollTax  float64
	FixedCosts        float64
	VariableCosts     float64
	WorkforceCosts    float64
	GovernanceCosts   float64
	ShockCosts        float64
	Taxes             float64
	Distribution      DistributionDue
	MemberRedemptions float64
	Interest          float64
	Principal         float64
}

func (o MandatoryObligations) GeneralTotal() float64 {
	return o.PriorArrears + o.Salary + o.SalaryPayrollTax + o.FixedCosts + o.VariableCosts +
		o.WorkforceCosts + o.GovernanceCosts + o.ShockCosts + o.Taxes +
		o.MemberRedemptions + o.Interest + o.Principal
}

// CarryForwardTotal contains obligations that have no dedicated payable queue.
// Unpaid principal is excluded because it remains in DebtBalance and becomes
// scheduled again by the debt-service calculation.
func (o MandatoryObligations) CarryForwardTotal() float64 {
	return o.PriorArrears + o.Salary + o.SalaryPayrollTax + o.FixedCosts +
		o.VariableCosts + o.WorkforceCosts + o.GovernanceCosts + o.ShockCosts + o.Interest
}

func (o MandatoryObligations) CurrentCarryForwardTotal() float64 {
	return o.CarryForwardTotal() - o.PriorArrears
}

func (o MandatoryObligations) Total() float64 {
	return o.GeneralTotal() + o.Distribution.Total()
}

type MandatoryPaid struct {
	PriorArrears      float64
	Salary            float64
	SalaryPayrollTax  float64
	FixedCosts        float64
	VariableCosts     float64
	WorkforceCosts    float64
	GovernanceCosts   float64
	ShockCosts        float64
	Taxes             float64
	Distribution      DistributionDue
	MemberRedemptions float64
	Interest          float64
	Principal         float64
}

func (p MandatoryPaid) Total() float64 {
	return p.PriorArrears + p.Salary + p.SalaryPayrollTax + p.FixedCosts + p.VariableCosts +
		p.WorkforceCosts + p.GovernanceCosts + p.ShockCosts + p.Taxes +
		p.Distribution.Total() + p.MemberRedemptions + p.Interest + p.Principal
}

func (p MandatoryPaid) CarryForwardTotal() float64 {
	return p.PriorArrears + p.Salary + p.SalaryPayrollTax + p.FixedCosts +
		p.VariableCosts + p.WorkforceCosts + p.GovernanceCosts + p.ShockCosts + p.Interest
}

type MandatorySettlementInput struct {
	Obligations                MandatoryObligations
	CashTotalBeforePayments    float64
	RestrictedDistributionCash float64
	RestrictedReserveCash      float64
	Funding                    LiquidityFunding
}

type MandatorySettlement struct {
	Paid                       MandatoryPaid
	UnpaidTotal                float64
	UnpaidCarryForward         float64
	CashTotalAfterPayments     float64
	RestrictedDistributionCash float64
	RestrictedReserveCash      float64
}

// SettleMandatory pays the section-13 order. If an earlier obligation cannot
// be paid in full, later obligations remain unpaid even if they have a
// dedicated restricted balance; this makes payment priority observable.
func SettleMandatory(input MandatorySettlementInput) MandatorySettlement {
	cash := input.CashTotalBeforePayments + input.Funding.CreditDraw
	restrictedDistribution := math.Max(0, input.RestrictedDistributionCash)
	restrictedReserve := math.Max(0, input.RestrictedReserveCash-input.Funding.RestrictedReserveReleased)
	generalAvailable := math.Max(0, input.Funding.GeneralCashAvailable)
	paid := MandatoryPaid{}
	blocked := false

	payGeneral := func(due float64, target *float64) {
		due = math.Max(0, due)
		if blocked {
			return
		}
		amount := math.Min(due, generalAvailable)
		*target = amount
		generalAvailable -= amount
		cash -= amount
		if amount < due {
			blocked = true
		}
	}

	payGeneral(input.Obligations.PriorArrears, &paid.PriorArrears)
	payGeneral(input.Obligations.Salary, &paid.Salary)
	payGeneral(input.Obligations.SalaryPayrollTax, &paid.SalaryPayrollTax)
	payGeneral(input.Obligations.FixedCosts, &paid.FixedCosts)
	payGeneral(input.Obligations.VariableCosts, &paid.VariableCosts)
	payGeneral(input.Obligations.WorkforceCosts, &paid.WorkforceCosts)
	payGeneral(input.Obligations.GovernanceCosts, &paid.GovernanceCosts)
	payGeneral(input.Obligations.ShockCosts, &paid.ShockCosts)
	payGeneral(input.Obligations.Taxes, &paid.Taxes)

	if !blocked {
		due := input.Obligations.Distribution.Total()
		amount := math.Min(due, restrictedDistribution)
		if due > 0 {
			paid.Distribution = input.Obligations.Distribution.Scale(amount / due)
		}
		restrictedDistribution -= amount
		cash -= amount
		if amount < due {
			blocked = true
		}
	}

	payGeneral(input.Obligations.MemberRedemptions, &paid.MemberRedemptions)
	payGeneral(input.Obligations.Interest, &paid.Interest)
	payGeneral(input.Obligations.Principal, &paid.Principal)

	unpaid := math.Max(0, input.Obligations.Total()-paid.Total())
	unpaidCarryForward := math.Max(0, input.Obligations.CarryForwardTotal()-paid.CarryForwardTotal())
	if cash < 0 && cash > -1e-9 {
		cash = 0
	}
	return MandatorySettlement{
		Paid:                       paid,
		UnpaidTotal:                unpaid,
		UnpaidCarryForward:         unpaidCarryForward,
		CashTotalAfterPayments:     cash,
		RestrictedDistributionCash: restrictedDistribution,
		RestrictedReserveCash:      restrictedReserve,
	}
}
