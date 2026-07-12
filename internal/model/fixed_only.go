package model

import (
	"math"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

func CalculateWorkforce(cfg config.Config, behavior config.BehaviorCase, env domain.EnvironmentMonth) domain.WorkforceState {
	annualRaw := cfg.Workforce.BaseTurnoverRateAnnual*env.LaborMarketFactor + behavior.TurnoverDeltaAnnualPP
	annual := clamp(cfg.Workforce.MinTurnoverRateAnnual, cfg.Workforce.MaxTurnoverRateAnnual, annualRaw)
	monthly := 1 - math.Pow(1-annual, 1.0/12.0)
	leavers := float64(cfg.Company.EmployeesCount) * monthly

	turnoverExcess := math.Max(0, annual-cfg.Workforce.BaseTurnoverRateAnnual)
	turnoverProductivityLoss := cfg.Workforce.TurnoverProductivityPenaltyPerAnnualTurnover * turnoverExcess
	uplift := clamp(
		cfg.Workforce.MinProductivityUplift,
		cfg.Workforce.MaxProductivityUplift,
		behavior.ProductivityUpliftDirect-turnoverProductivityLoss+behavior.FairnessPenaltyToQuality,
	)

	return domain.WorkforceState{
		EmployeesCount:                cfg.Company.EmployeesCount,
		TurnoverRateAnnual:            annual,
		TurnoverRateMonthly:           monthly,
		LeaversCount:                  leavers,
		TurnoverProductivityLoss:      turnoverProductivityLoss,
		ProductivityUplift:            uplift,
		ProductivityLevel:             1 + uplift,
		HighPerformerAttritionWarning: behavior.HighPerfAttritionDeltaPP > 0 || behavior.FairnessPenaltyToQuality < 0,
	}
}

// CalculateOperatingPnL performs the accrual calculations through
// operating_profit_before_bonus. Tax accrual is a later monthly step.
func CalculateOperatingPnL(cfg config.Config, workforce domain.WorkforceState, env domain.EnvironmentMonth) domain.PnLState {
	baselineRevenue := float64(workforce.EmployeesCount) * cfg.Company.BaseRevenuePerEmployee * env.CumulativeMarketTrend
	productivityMultiplier := 1 + cfg.Company.RevenueProductivityElasticity*workforce.ProductivityUplift
	potentialRevenue := baselineRevenue * env.MarketFactor * productivityMultiplier
	if env.ShockHappened {
		potentialRevenue *= env.ShockRevenueMultiplier
	}

	revenue := math.Max(0, potentialRevenue)
	if cfg.Company.DemandCapMultiplier != nil {
		revenue = math.Min(revenue, baselineRevenue*(*cfg.Company.DemandCapMultiplier))
	}

	salaryCosts := float64(workforce.EmployeesCount) * cfg.Company.BaseSalaryPerEmployee * env.CostInflationFactor
	fixedCosts := cfg.Company.FixedCostsMonthly * env.CostInflationFactor
	variableCosts := revenue * cfg.Company.VariableCostRate
	costPerLeaver := cfg.Workforce.RecruitingCostPerLeaver +
		cfg.Workforce.OnboardingCostPerLeaver +
		cfg.Workforce.ManagerTimeCostPerLeaver +
		cfg.Workforce.LostProductivityCostPerLeaver
	turnoverCost := workforce.LeaversCount * costPerLeaver
	shockCost := float64(env.ShockCost)
	totalCosts := salaryCosts + fixedCosts + variableCosts + turnoverCost + shockCost
	operatingProfit := revenue - totalCosts

	// fixed_only has no bonus expense.
	profitAfterBonusBeforeTax := operatingProfit

	return domain.PnLState{
		Revenue:                    domain.Money(revenue),
		SalaryCosts:                domain.Money(salaryCosts),
		FixedCosts:                 domain.Money(fixedCosts),
		VariableCosts:              domain.Money(variableCosts),
		TurnoverCost:               domain.Money(turnoverCost),
		ShockCost:                  domain.Money(shockCost),
		TotalCostsBeforeBonus:      domain.Money(totalCosts),
		OperatingProfitBeforeBonus: domain.Money(operatingProfit),
		BonusExpenseAccrual:        0,
		BonusPayrollTaxAccrual:     0,
		ProfitAfterBonusBeforeTax:  domain.Money(profitAfterBonusBeforeTax),
	}
}

// AccrueProfitTax is deliberately separate from operating P&L and cash
// settlement to preserve the monthly order from specification section 8.
func AccrueProfitTax(cfg config.Config, pnl domain.PnLState) domain.PnLState {
	profitTaxAccrual := math.Max(0, float64(pnl.ProfitAfterBonusBeforeTax)) * cfg.Cashflow.ProfitTaxRate
	pnl.ProfitTaxAccrual = domain.Money(profitTaxAccrual)
	pnl.AccountingProfitAfterBonus = pnl.ProfitAfterBonusBeforeTax - pnl.ProfitTaxAccrual
	return pnl
}

// CalculateCash settles current collections and obligations that are already
// due. Opening balances and due queue amounts are supplied by the runner.
func CalculateCash(cfg config.Config, pnl domain.PnLState, env domain.EnvironmentMonth, input domain.CashMonthInput) domain.CashState {
	effectiveCollectionRate := clamp(0, 1, cfg.Cashflow.RevenueCollectionRateCurrentMonth*env.CollectionRateMultiplier)
	cashCollectedCurrent := float64(pnl.Revenue) * effectiveCollectionRate
	newAR := float64(pnl.Revenue) * (1 - effectiveCollectionRate) * (1 - cfg.Cashflow.BadDebtRate)
	collections := cashCollectedCurrent + float64(input.CashCollectedFromAR)

	directPayments := float64(pnl.SalaryCosts+pnl.FixedCosts+pnl.VariableCosts+pnl.TurnoverCost+pnl.ShockCost) +
		cfg.Cashflow.DebtServiceMonthly + cfg.Cashflow.CapexMonthly
	taxesPaid := float64(input.TaxesPaidCash)
	bonusDue := float64(input.BonusDueGross + input.BonusDuePayrollTax)
	duePayments := directPayments + taxesPaid + bonusDue
	closingCash := float64(input.OpeningCashTotal) + collections - duePayments
	restrictedBonusCash := float64(input.OpeningRestrictedBonusCash) - bonusDue
	unrestrictedCash := closingCash - restrictedBonusCash
	requiredReserve := cfg.Company.RequiredCashReserveMonths *
		(float64(pnl.SalaryCosts) + float64(pnl.FixedCosts) + cfg.Cashflow.DebtServiceMonthly)
	ownerDistributableCash := 0.0
	if domain.MoneyLess(domain.Money(requiredReserve), domain.Money(unrestrictedCash)) {
		ownerDistributableCash = unrestrictedCash - requiredReserve
	}

	cash := domain.CashState{
		OpeningCashTotal:         input.OpeningCashTotal,
		CashCollectedCurrent:     domain.Money(cashCollectedCurrent),
		CashCollectedFromAR:      input.CashCollectedFromAR,
		CashCollectedFromRevenue: domain.Money(collections),
		AccountsReceivable: domain.AccountsReceivableState{
			Opening:             input.OpeningAccountsReceivable,
			Collected:           input.CashCollectedFromAR,
			New:                 domain.Money(newAR),
			Closing:             (input.OpeningAccountsReceivable - input.CashCollectedFromAR) + domain.Money(newAR),
			CollectionLagMonths: cfg.Cashflow.AccountsReceivableLagMonths,
		},
		DueCashPayments:         domain.Money(duePayments),
		TaxesPaidCash:           input.TaxesPaidCash,
		ClosingCashTotal:        domain.Money(closingCash),
		RestrictedBonusCash:     domain.Money(restrictedBonusCash),
		ClosingUnrestrictedCash: domain.Money(unrestrictedCash),
		RequiredCashReserve:     domain.Money(requiredReserve),
		OwnerDistributableCash:  domain.Money(ownerDistributableCash),
		TaxPayableClosing:       input.OpeningTaxPayable - input.TaxesPaidCash,
	}
	return cash
}

// RecordProfitTaxPayable records the P&L accrual as a future cash liability; it
// does not change current cash.
func RecordProfitTaxPayable(cash domain.CashState, taxAccrual domain.Money) domain.CashState {
	cash.TaxPayableClosing += taxAccrual
	return cash
}

// CalculateRiskFlags closes the month after all current-stage accruals have
// been recorded.
func CalculateRiskFlags(cfg config.Config, cash domain.CashState) domain.RiskFlags {
	return domain.RiskFlags{
		ReserveBreach: domain.MoneyLess(cash.ClosingUnrestrictedCash, cash.RequiredCashReserve),
		LiquidityGap:  domain.MoneyLess(cash.ClosingUnrestrictedCash, 0),
		CashGap:       domain.MoneyLess(cash.ClosingCashTotal, 0),
		Bankruptcy:    domain.MoneyLess(cash.ClosingCashTotal, domain.Money(-cfg.Cashflow.AvailableCreditLine)),
	}
}

func clamp(minimum, maximum, value float64) float64 {
	return math.Min(maximum, math.Max(minimum, value))
}
