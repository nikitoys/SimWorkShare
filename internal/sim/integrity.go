package sim

import (
	"fmt"
	"math"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

func validateFiniteResult(result domain.MonthlyResult) error {
	values := []struct {
		path  string
		value float64
	}{
		{"environment.cumulative_market_trend", result.Environment.CumulativeMarketTrend},
		{"environment.market_factor", result.Environment.MarketFactor},
		{"environment.cost_inflation_factor", result.Environment.CostInflationFactor},
		{"environment.labor_market_factor", result.Environment.LaborMarketFactor},
		{"environment.shock_revenue_multiplier", result.Environment.ShockRevenueMultiplier},
		{"environment.shock_cost", float64(result.Environment.ShockCost)},
		{"environment.collection_rate_multiplier", result.Environment.CollectionRateMultiplier},
		{"workforce.turnover_rate_annual", result.Workforce.TurnoverRateAnnual},
		{"workforce.turnover_rate_monthly", result.Workforce.TurnoverRateMonthly},
		{"workforce.leavers_count", result.Workforce.LeaversCount},
		{"workforce.turnover_productivity_loss", result.Workforce.TurnoverProductivityLoss},
		{"workforce.productivity_uplift", result.Workforce.ProductivityUplift},
		{"workforce.productivity_level", result.Workforce.ProductivityLevel},
		{"pnl.revenue", float64(result.PnL.Revenue)},
		{"pnl.salary_costs", float64(result.PnL.SalaryCosts)},
		{"pnl.fixed_costs", float64(result.PnL.FixedCosts)},
		{"pnl.variable_costs", float64(result.PnL.VariableCosts)},
		{"pnl.turnover_cost", float64(result.PnL.TurnoverCost)},
		{"pnl.shock_cost", float64(result.PnL.ShockCost)},
		{"pnl.total_costs_before_bonus", float64(result.PnL.TotalCostsBeforeBonus)},
		{"pnl.operating_profit_before_bonus", float64(result.PnL.OperatingProfitBeforeBonus)},
		{"pnl.bonus_expense_accrual", float64(result.PnL.BonusExpenseAccrual)},
		{"pnl.bonus_payroll_tax_accrual", float64(result.PnL.BonusPayrollTaxAccrual)},
		{"pnl.profit_after_bonus_before_tax", float64(result.PnL.ProfitAfterBonusBeforeTax)},
		{"pnl.profit_tax_accrual", float64(result.PnL.ProfitTaxAccrual)},
		{"pnl.accounting_profit_after_bonus", float64(result.PnL.AccountingProfitAfterBonus)},
		{"cash.opening_cash_total", float64(result.Cash.OpeningCashTotal)},
		{"cash.cash_collected_current", float64(result.Cash.CashCollectedCurrent)},
		{"cash.cash_collected_from_ar", float64(result.Cash.CashCollectedFromAR)},
		{"cash.cash_collected_from_revenue", float64(result.Cash.CashCollectedFromRevenue)},
		{"cash.accounts_receivable.opening", float64(result.Cash.AccountsReceivable.Opening)},
		{"cash.accounts_receivable.collected", float64(result.Cash.AccountsReceivable.Collected)},
		{"cash.accounts_receivable.new", float64(result.Cash.AccountsReceivable.New)},
		{"cash.accounts_receivable.closing", float64(result.Cash.AccountsReceivable.Closing)},
		{"cash.due_cash_payments", float64(result.Cash.DueCashPayments)},
		{"cash.taxes_paid_cash", float64(result.Cash.TaxesPaidCash)},
		{"cash.closing_cash_total", float64(result.Cash.ClosingCashTotal)},
		{"cash.restricted_bonus_cash", float64(result.Cash.RestrictedBonusCash)},
		{"cash.closing_unrestricted_cash", float64(result.Cash.ClosingUnrestrictedCash)},
		{"cash.required_cash_reserve", float64(result.Cash.RequiredCashReserve)},
		{"cash.owner_distributable_cash", float64(result.Cash.OwnerDistributableCash)},
		{"cash.tax_payable_closing", float64(result.Cash.TaxPayableClosing)},
	}
	for _, item := range values {
		if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
			return fmt.Errorf("integrity: %s is not finite", item.path)
		}
	}
	if result.Compensation != nil {
		comp := result.Compensation
		compensationValues := []struct {
			path  string
			value float64
		}{
			{"compensation.period_profit_base_accumulator", float64(comp.PeriodProfitBaseAccumulator)},
			{"compensation.period_profit_base", float64(comp.PeriodProfitBase)},
			{"compensation.cash_base_for_total_employer_bonus_cost", float64(comp.CashBaseForTotalEmployerBonusCost)},
			{"compensation.distributable_base", float64(comp.DistributableBase)},
			{"compensation.gross_bonus_pool_accrued", float64(comp.GrossBonusPoolAccrued)},
			{"compensation.bonus_payroll_tax_accrued", float64(comp.BonusPayrollTaxAccrued)},
			{"compensation.total_bonus_employer_cost_accrued", float64(comp.TotalBonusEmployerCostAccrued)},
			{"compensation.bonus_per_employee_accrued", float64(comp.BonusPerEmployeeAccrued)},
			{"compensation.opening_bonus_payable_gross", float64(comp.OpeningBonusPayableGross)},
			{"compensation.opening_bonus_payable_payroll_tax", float64(comp.OpeningBonusPayablePayrollTax)},
			{"compensation.opening_bonus_payable", float64(comp.OpeningBonusPayable)},
			{"compensation.bonus_paid_cash", float64(comp.BonusPaidCash)},
			{"compensation.bonus_payroll_tax_paid", float64(comp.BonusPayrollTaxPaid)},
			{"compensation.closing_bonus_payable_gross", float64(comp.ClosingBonusPayableGross)},
			{"compensation.closing_bonus_payable_payroll_tax", float64(comp.ClosingBonusPayablePayrollTax)},
			{"compensation.closing_bonus_payable", float64(comp.ClosingBonusPayable)},
		}
		for _, item := range compensationValues {
			if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
				return fmt.Errorf("integrity: %s is not finite", item.path)
			}
		}
	}
	return nil
}

func validateMonthInvariants(
	cfg config.Config,
	scenario config.CompensationScenario,
	result domain.MonthlyResult,
	arOutstanding domain.Money,
	taxOutstanding domain.Money,
	bonusOutstanding bonusDue,
) error {
	pnl := result.PnL
	cash := result.Cash

	expectedTotalCosts := pnl.SalaryCosts + pnl.FixedCosts + pnl.VariableCosts + pnl.TurnoverCost + pnl.ShockCost
	if !domain.MoneyAlmostEqual(pnl.TotalCostsBeforeBonus, expectedTotalCosts) {
		return fmt.Errorf("integrity: total pre-bonus costs do not balance")
	}
	if !domain.MoneyAlmostEqual(pnl.OperatingProfitBeforeBonus, pnl.Revenue-pnl.TotalCostsBeforeBonus) {
		return fmt.Errorf("integrity: operating P&L does not balance")
	}
	if scenario.Type == "fixed_only" {
		if result.Compensation != nil || pnl.BonusExpenseAccrual != 0 || pnl.BonusPayrollTaxAccrual != 0 ||
			cash.RestrictedBonusCash != 0 || bonusOutstanding.total() != 0 {
			return fmt.Errorf("integrity: fixed_only created bonus state")
		}
	} else if result.Compensation == nil {
		return fmt.Errorf("integrity: profit_share result has no compensation state")
	}
	if !domain.MoneyAlmostEqual(pnl.ProfitAfterBonusBeforeTax,
		pnl.OperatingProfitBeforeBonus-pnl.BonusExpenseAccrual-pnl.BonusPayrollTaxAccrual) {
		return fmt.Errorf("integrity: pre-tax profit after bonus does not balance")
	}
	expectedTaxAccrual := domain.Money(math.Max(0, float64(pnl.ProfitAfterBonusBeforeTax)) * cfg.Cashflow.ProfitTaxRate)
	if !domain.MoneyAlmostEqual(pnl.ProfitTaxAccrual, expectedTaxAccrual) {
		return fmt.Errorf("integrity: profit tax accrual does not balance")
	}
	if !domain.MoneyAlmostEqual(pnl.AccountingProfitAfterBonus, pnl.ProfitAfterBonusBeforeTax-pnl.ProfitTaxAccrual) {
		return fmt.Errorf("integrity: accounting profit does not balance")
	}
	if !domain.MoneyAlmostEqual(cash.CashCollectedFromRevenue, cash.CashCollectedCurrent+cash.CashCollectedFromAR) {
		return fmt.Errorf("integrity: revenue cash collections do not balance")
	}
	if domain.MoneyLess(cash.AccountsReceivable.Opening, cash.AccountsReceivable.Collected) {
		return fmt.Errorf("integrity: AR collection exceeds opening balance")
	}
	if !moneyLedgerAlmostEqual(cash.AccountsReceivable.Closing, cash.AccountsReceivable.Opening,
		cash.AccountsReceivable.New, cash.AccountsReceivable.Collected) {
		return fmt.Errorf("integrity: AR ledger does not balance")
	}
	if domain.MoneyLess(cash.AccountsReceivable.Closing, 0) {
		return fmt.Errorf("integrity: closing AR is negative")
	}
	if !domain.MoneyAlmostEqual(cash.AccountsReceivable.Closing, arOutstanding) {
		return fmt.Errorf("integrity: AR ledger and queue disagree")
	}
	var bonusPaidTotal domain.Money
	if result.Compensation != nil {
		bonusPaidTotal = result.Compensation.BonusPaidCash + result.Compensation.BonusPayrollTaxPaid
	}
	expectedDuePayments := pnl.TotalCostsBeforeBonus +
		domain.Money(cfg.Cashflow.DebtServiceMonthly+cfg.Cashflow.CapexMonthly) +
		cash.TaxesPaidCash + bonusPaidTotal
	if !domain.MoneyAlmostEqual(cash.DueCashPayments, expectedDuePayments) {
		return fmt.Errorf("integrity: due cash payments do not balance")
	}
	if !domain.MoneyAlmostEqual(cash.ClosingCashTotal, cash.OpeningCashTotal+cash.CashCollectedFromRevenue-cash.DueCashPayments) {
		return fmt.Errorf("integrity: cash ledger does not balance")
	}
	if !domain.MoneyAlmostEqual(cash.ClosingUnrestrictedCash, cash.ClosingCashTotal-cash.RestrictedBonusCash) {
		return fmt.Errorf("integrity: unrestricted cash does not balance")
	}
	if domain.MoneyLess(cash.RestrictedBonusCash, 0) {
		return fmt.Errorf("integrity: restricted bonus cash is negative")
	}
	if !domain.MoneyAlmostEqual(cash.RestrictedBonusCash, bonusOutstanding.total()) {
		return fmt.Errorf("integrity: restricted bonus cash and bonus queue disagree")
	}
	expectedOwnerCash := domain.Money(0)
	if domain.MoneyLess(cash.RequiredCashReserve, cash.ClosingUnrestrictedCash) {
		expectedOwnerCash = cash.ClosingUnrestrictedCash - cash.RequiredCashReserve
	}
	if !domain.MoneyAlmostEqual(cash.OwnerDistributableCash, expectedOwnerCash) {
		return fmt.Errorf("integrity: owner distributable cash does not balance")
	}
	if !domain.MoneyAlmostEqual(cash.TaxPayableClosing, taxOutstanding) {
		return fmt.Errorf("integrity: tax payable ledger and queue disagree")
	}
	if domain.MoneyLess(cash.TaxPayableClosing, 0) {
		return fmt.Errorf("integrity: closing tax payable is negative")
	}
	if result.Compensation != nil {
		comp := result.Compensation
		if !domain.MoneyAlmostEqual(pnl.BonusExpenseAccrual, comp.GrossBonusPoolAccrued) ||
			!domain.MoneyAlmostEqual(pnl.BonusPayrollTaxAccrual, comp.BonusPayrollTaxAccrued) {
			return fmt.Errorf("integrity: P&L and compensation bonus accruals disagree")
		}
		if !domain.MoneyAlmostEqual(comp.OpeningBonusPayable,
			comp.OpeningBonusPayableGross+comp.OpeningBonusPayablePayrollTax) {
			return fmt.Errorf("integrity: opening bonus payable components do not balance")
		}
		if domain.MoneyLess(comp.OpeningBonusPayableGross, comp.BonusPaidCash) ||
			domain.MoneyLess(comp.OpeningBonusPayablePayrollTax, comp.BonusPayrollTaxPaid) {
			return fmt.Errorf("integrity: bonus payment exceeds opening payable")
		}
		if !domain.MoneyAlmostEqual(comp.TotalBonusEmployerCostAccrued,
			comp.GrossBonusPoolAccrued+comp.BonusPayrollTaxAccrued) {
			return fmt.Errorf("integrity: employer bonus accrual does not balance")
		}
		if !domain.MoneyAlmostEqual(comp.BonusPayrollTaxAccrued,
			comp.GrossBonusPoolAccrued*domain.Money(cfg.Cashflow.BonusPayrollTaxRate)) {
			return fmt.Errorf("integrity: bonus payroll tax does not balance")
		}
		if !moneyLedgerAlmostEqual(comp.ClosingBonusPayableGross, comp.OpeningBonusPayableGross,
			comp.GrossBonusPoolAccrued, comp.BonusPaidCash) {
			return fmt.Errorf("integrity: gross bonus payable does not balance")
		}
		if !moneyLedgerAlmostEqual(comp.ClosingBonusPayablePayrollTax, comp.OpeningBonusPayablePayrollTax,
			comp.BonusPayrollTaxAccrued, comp.BonusPayrollTaxPaid) {
			return fmt.Errorf("integrity: bonus payroll-tax payable does not balance")
		}
		if !domain.MoneyAlmostEqual(comp.ClosingBonusPayable,
			comp.ClosingBonusPayableGross+comp.ClosingBonusPayablePayrollTax) {
			return fmt.Errorf("integrity: closing bonus payable components do not balance")
		}
		if !domain.MoneyAlmostEqual(comp.ClosingBonusPayable, bonusOutstanding.total()) ||
			!domain.MoneyAlmostEqual(comp.ClosingBonusPayableGross, bonusOutstanding.Gross) ||
			!domain.MoneyAlmostEqual(comp.ClosingBonusPayablePayrollTax, bonusOutstanding.PayrollTax) {
			return fmt.Errorf("integrity: bonus payable ledger and queue disagree")
		}
		if scenario.EligibleEmployeesCount == nil || !domain.MoneyAlmostEqual(
			comp.BonusPerEmployeeAccrued*domain.Money(*scenario.EligibleEmployeesCount),
			comp.GrossBonusPoolAccrued,
		) {
			return fmt.Errorf("integrity: per-employee bonus does not balance")
		}
		if !moneyLedgerAlmostEqual(cash.RestrictedBonusCash, comp.OpeningBonusPayable,
			comp.TotalBonusEmployerCostAccrued, comp.BonusPaidCash+comp.BonusPayrollTaxPaid) {
			return fmt.Errorf("integrity: restricted bonus cash ledger does not balance")
		}
		if comp.IsBonusPeriodEnd {
			if scenario.ProfitSharePercent == nil || scenario.ProfitHurdleMonthly == nil ||
				scenario.EligibleEmployeesCount == nil {
				return fmt.Errorf("integrity: profit_share scenario is not normalized")
			}
			expectedProfitBase := domain.Money(math.Max(0,
				float64(comp.PeriodProfitBaseAccumulator)-
					*scenario.ProfitHurdleMonthly*float64(comp.MonthsInBonusPeriod)))
			if !domain.MoneyAlmostEqual(comp.PeriodProfitBase, expectedProfitBase) {
				return fmt.Errorf("integrity: period profit base does not balance")
			}
			currentTaxReserve := 0.0
			if cfg.Cashflow.ReserveCurrentProfitTax {
				currentTaxReserve = math.Max(0, float64(pnl.OperatingProfitBeforeBonus)) * cfg.Cashflow.ProfitTaxRate
			}
			plannedReinvestmentReserve := cfg.Cashflow.PlannedReinvestmentRate *
				math.Max(0, float64(pnl.OperatingProfitBeforeBonus))
			restrictedBeforeNewBonus := cash.RestrictedBonusCash - comp.TotalBonusEmployerCostAccrued
			unrestrictedBeforeNewBonus := cash.ClosingCashTotal - restrictedBeforeNewBonus
			expectedCashBase := domain.Money(math.Max(0,
				float64(unrestrictedBeforeNewBonus-cash.RequiredCashReserve)-
					currentTaxReserve-plannedReinvestmentReserve))
			if !domain.MoneyAlmostEqual(comp.CashBaseForTotalEmployerBonusCost, expectedCashBase) {
				return fmt.Errorf("integrity: bonus cash base does not balance")
			}
			expectedGross := *scenario.ProfitSharePercent * float64(expectedProfitBase)
			expectedGross = math.Min(expectedGross,
				float64(expectedCashBase)/(1+cfg.Cashflow.BonusPayrollTaxRate))
			if scenario.BonusCapTotal != nil {
				expectedGross = math.Min(expectedGross, *scenario.BonusCapTotal)
			}
			if scenario.BonusCapPerEmployee != nil {
				expectedGross = math.Min(expectedGross,
					*scenario.BonusCapPerEmployee*float64(*scenario.EligibleEmployeesCount))
			}
			expectedGross = math.Max(0, expectedGross)
			if !domain.MoneyAlmostEqual(comp.GrossBonusPoolAccrued, domain.Money(expectedGross)) {
				return fmt.Errorf("integrity: gross bonus policy limits do not balance")
			}
			expectedDistributableBase := domain.Money(0)
			if *scenario.ProfitSharePercent > 0 {
				expectedDistributableBase = comp.GrossBonusPoolAccrued / domain.Money(*scenario.ProfitSharePercent)
			}
			if !domain.MoneyAlmostEqual(comp.DistributableBase, expectedDistributableBase) {
				return fmt.Errorf("integrity: distributable base does not balance")
			}
		}
	}
	return nil
}

func validateFiniteSummary(summary domain.TerminalSummary) error {
	values := []struct {
		path  string
		value float64
	}{
		{"terminal_summary.closing_cash_total", float64(summary.ClosingCashTotal)},
		{"terminal_summary.closing_unrestricted_cash", float64(summary.ClosingUnrestrictedCash)},
		{"terminal_summary.minimum_unrestricted_cash", float64(summary.MinimumUnrestrictedCash)},
		{"terminal_summary.outstanding_accounts_receivable", float64(summary.OutstandingAccountsReceivable)},
		{"terminal_summary.closing_tax_payable", float64(summary.ClosingTaxPayable)},
		{"terminal_summary.total_taxes_paid_cash", float64(summary.TotalTaxesPaidCash)},
	}
	for _, item := range values {
		if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
			return fmt.Errorf("integrity: %s is not finite", item.path)
		}
	}
	if summary.Compensation != nil {
		comp := summary.Compensation
		compensationValues := []struct {
			path  string
			value float64
		}{
			{"terminal_summary.compensation.total_gross_bonus_accrued", float64(comp.TotalGrossBonusAccrued)},
			{"terminal_summary.compensation.total_bonus_payroll_tax_accrued", float64(comp.TotalBonusPayrollTaxAccrued)},
			{"terminal_summary.compensation.total_employer_bonus_cost_accrued", float64(comp.TotalEmployerBonusCostAccrued)},
			{"terminal_summary.compensation.total_gross_bonus_paid", float64(comp.TotalGrossBonusPaid)},
			{"terminal_summary.compensation.total_bonus_payroll_tax_paid", float64(comp.TotalBonusPayrollTaxPaid)},
			{"terminal_summary.compensation.closing_bonus_payable_gross", float64(comp.ClosingBonusPayableGross)},
			{"terminal_summary.compensation.closing_bonus_payable_payroll_tax", float64(comp.ClosingBonusPayablePayrollTax)},
			{"terminal_summary.compensation.closing_bonus_payable", float64(comp.ClosingBonusPayable)},
			{"terminal_summary.compensation.closing_restricted_bonus_cash", float64(comp.ClosingRestrictedBonusCash)},
			{"terminal_summary.compensation.unclosed_period_profit_accumulator", float64(comp.UnclosedPeriodProfitAccumulator)},
		}
		for _, item := range compensationValues {
			if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
				return fmt.Errorf("integrity: %s is not finite", item.path)
			}
		}
		if !domain.MoneyAlmostEqual(comp.TotalEmployerBonusCostAccrued,
			comp.TotalGrossBonusAccrued+comp.TotalBonusPayrollTaxAccrued) {
			return fmt.Errorf("integrity: terminal employer bonus accrual does not balance")
		}
		if !moneyLedgerAlmostEqual(comp.ClosingBonusPayableGross, comp.TotalGrossBonusAccrued,
			0, comp.TotalGrossBonusPaid) {
			return fmt.Errorf("integrity: terminal gross bonus payable does not balance")
		}
		if !moneyLedgerAlmostEqual(comp.ClosingBonusPayablePayrollTax, comp.TotalBonusPayrollTaxAccrued,
			0, comp.TotalBonusPayrollTaxPaid) {
			return fmt.Errorf("integrity: terminal bonus payroll-tax payable does not balance")
		}
		if !domain.MoneyAlmostEqual(comp.ClosingBonusPayable,
			comp.ClosingBonusPayableGross+comp.ClosingBonusPayablePayrollTax) {
			return fmt.Errorf("integrity: terminal bonus payable components do not balance")
		}
		if !domain.MoneyAlmostEqual(comp.ClosingRestrictedBonusCash, comp.ClosingBonusPayable) {
			return fmt.Errorf("integrity: terminal restricted bonus cash and payable disagree")
		}
	}
	return nil
}

// moneyLedgerAlmostEqual uses the same absolute/relative tolerances as Money,
// but scales relative error by every ledger operand. This is necessary when a
// large due amount is removed and a much smaller future queue entry remains:
// the aggregate opening float cannot represent both magnitudes at once.
func moneyLedgerAlmostEqual(closing, opening, added, removed domain.Money) bool {
	expected := (opening - removed) + added
	diff := math.Abs(float64(closing - expected))
	scale := 1.0
	for _, value := range []domain.Money{closing, opening, added, removed} {
		scale = math.Max(scale, math.Abs(float64(value)))
	}
	return diff <= math.Max(float64(domain.MoneyAbsoluteTolerance), domain.RelativeTolerance*scale)
}
