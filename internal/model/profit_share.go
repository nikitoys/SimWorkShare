package model

import (
	"fmt"
	"math"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

type ProfitShareAccrualInput struct {
	PeriodProfitBaseAccumulator   domain.Money
	MonthsInBonusPeriod           int
	IsBonusPeriodEnd              bool
	OpeningBonusPayableGross      domain.Money
	OpeningBonusPayablePayrollTax domain.Money
	BonusPaidCash                 domain.Money
	BonusPayrollTaxPaid           domain.Money
}

// ApplyProfitShareAccrual applies the detailed affordability formula from
// specification section 9.7. It changes P&L and restricted cash, but does not
// reduce total cash; total cash changes only when a queued bonus is paid.
func ApplyProfitShareAccrual(
	cfg config.Config,
	scenario config.CompensationScenario,
	pnl domain.PnLState,
	cash domain.CashState,
	input ProfitShareAccrualInput,
) (domain.PnLState, domain.CashState, domain.CompensationState, error) {
	compensation := domain.CompensationState{
		BonusPeriod:                   scenario.BonusPeriod,
		IsBonusPeriodEnd:              input.IsBonusPeriodEnd,
		MonthsInBonusPeriod:           input.MonthsInBonusPeriod,
		PeriodProfitBaseAccumulator:   input.PeriodProfitBaseAccumulator,
		OpeningBonusPayableGross:      input.OpeningBonusPayableGross,
		OpeningBonusPayablePayrollTax: input.OpeningBonusPayablePayrollTax,
		OpeningBonusPayable:           input.OpeningBonusPayableGross + input.OpeningBonusPayablePayrollTax,
		BonusPaidCash:                 input.BonusPaidCash,
		BonusPayrollTaxPaid:           input.BonusPayrollTaxPaid,
	}

	if !input.IsBonusPeriodEnd {
		return pnl, cash, compensation, nil
	}
	if scenario.ProfitSharePercent == nil || scenario.ProfitHurdleMonthly == nil ||
		scenario.EligibleEmployeesCount == nil {
		return domain.PnLState{}, domain.CashState{}, domain.CompensationState{},
			fmt.Errorf("profit_share scenario is not normalized")
	}

	share := *scenario.ProfitSharePercent
	eligibleEmployees := *scenario.EligibleEmployeesCount
	periodHurdle := *scenario.ProfitHurdleMonthly * float64(input.MonthsInBonusPeriod)
	periodProfitBase := math.Max(0, float64(input.PeriodProfitBaseAccumulator)-periodHurdle)

	currentTaxReserve := 0.0
	if cfg.Cashflow.ReserveCurrentProfitTax {
		currentTaxReserve = math.Max(0, float64(pnl.OperatingProfitBeforeBonus)) * cfg.Cashflow.ProfitTaxRate
	}
	plannedReinvestmentReserve := cfg.Cashflow.PlannedReinvestmentRate *
		math.Max(0, float64(pnl.OperatingProfitBeforeBonus))
	unrestrictedBeforeNewBonus := float64(cash.ClosingCashTotal - cash.RestrictedBonusCash)
	cashBase := math.Max(0,
		unrestrictedBeforeNewBonus-
			float64(cash.RequiredCashReserve)-
			currentTaxReserve-
			plannedReinvestmentReserve,
	)
	maxGrossByCash := cashBase / (1 + cfg.Cashflow.BonusPayrollTaxRate)
	rawGrossBonus := share * periodProfitBase
	grossBonus := math.Min(rawGrossBonus, maxGrossByCash)
	if scenario.BonusCapTotal != nil {
		grossBonus = math.Min(grossBonus, *scenario.BonusCapTotal)
	}
	if scenario.BonusCapPerEmployee != nil {
		grossBonus = math.Min(grossBonus, *scenario.BonusCapPerEmployee*float64(eligibleEmployees))
	}
	grossBonus = math.Max(0, grossBonus)
	payrollTax := grossBonus * cfg.Cashflow.BonusPayrollTaxRate
	totalEmployerCost := grossBonus + payrollTax
	distributableBase := 0.0
	if share > 0 {
		distributableBase = grossBonus / share
	}

	pnl.BonusExpenseAccrual = domain.Money(grossBonus)
	pnl.BonusPayrollTaxAccrual = domain.Money(payrollTax)
	pnl.ProfitAfterBonusBeforeTax = pnl.OperatingProfitBeforeBonus -
		pnl.BonusExpenseAccrual - pnl.BonusPayrollTaxAccrual

	cash.RestrictedBonusCash += domain.Money(totalEmployerCost)
	cash.ClosingUnrestrictedCash = cash.ClosingCashTotal - cash.RestrictedBonusCash
	cash.OwnerDistributableCash = 0
	if domain.MoneyLess(cash.RequiredCashReserve, cash.ClosingUnrestrictedCash) {
		cash.OwnerDistributableCash = cash.ClosingUnrestrictedCash - cash.RequiredCashReserve
	}

	compensation.PeriodProfitBase = domain.Money(periodProfitBase)
	compensation.CashBaseForTotalEmployerBonusCost = domain.Money(cashBase)
	compensation.DistributableBase = domain.Money(distributableBase)
	compensation.GrossBonusPoolAccrued = domain.Money(grossBonus)
	compensation.BonusPayrollTaxAccrued = domain.Money(payrollTax)
	compensation.TotalBonusEmployerCostAccrued = domain.Money(totalEmployerCost)
	compensation.BonusPerEmployeeAccrued = domain.Money(grossBonus / float64(eligibleEmployees))
	return pnl, cash, compensation, nil
}
