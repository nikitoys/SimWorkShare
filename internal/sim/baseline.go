package sim

import (
	"fmt"
	"math"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
	"simworkshare/internal/model"
)

// RunDeterministicFixedOnly preserves the completed fixed_only + no_effect +
// normal_market execution profile.
func RunDeterministicFixedOnly(cfg config.Config) (domain.SimulationResult, error) {
	return runDeterministicScenarioMonths(cfg, "fixed_only", "no_effect", cfg.Simulation.Months)
}

// RunDeterministicScenario runs one supported compensation scenario against a
// deterministic normal_market path. fixed_only is intentionally limited to
// no_effect; profit_share can use any declared behavior case.
func RunDeterministicScenario(cfg config.Config, scenarioName, behaviorCase string) (domain.SimulationResult, error) {
	return runDeterministicScenarioMonths(cfg, scenarioName, behaviorCase, cfg.Simulation.Months)
}

// RunDeterministicBaseline preserves the stage-one API and exact first-month
// fixed_only MonthlyResult.
func RunDeterministicBaseline(cfg config.Config) (domain.MonthlyResult, error) {
	result, err := runDeterministicScenarioMonths(cfg, "fixed_only", "no_effect", 1)
	if err != nil {
		return domain.MonthlyResult{}, err
	}
	return result.MonthlyResults[0], nil
}

func runDeterministicScenarioMonths(
	cfg config.Config,
	scenarioName string,
	behaviorCase string,
	months int,
) (domain.SimulationResult, error) {
	if err := config.Validate(cfg); err != nil {
		return domain.SimulationResult{}, fmt.Errorf("validate deterministic config: %w", err)
	}
	if months < 1 {
		return domain.SimulationResult{}, fmt.Errorf("months must be >= 1")
	}

	scenario, err := selectedScenario(cfg, scenarioName)
	if err != nil {
		return domain.SimulationResult{}, err
	}
	behavior, ok := cfg.BehaviorCases[behaviorCase]
	if !ok {
		return domain.SimulationResult{}, fmt.Errorf("unknown behavior case %q", behaviorCase)
	}
	if scenario.Type == "fixed_only" && behaviorCase != "no_effect" {
		return domain.SimulationResult{}, fmt.Errorf("fixed_only supports only behavior case %q", "no_effect")
	}
	if scenario.Type == "fixed_raise_same_expected_cost" {
		return domain.SimulationResult{}, fmt.Errorf("scenario %q is outside the current deterministic stage", scenarioName)
	}
	if scenario.Type == "profit_share" {
		if scenario.BonusPeriod != "monthly" {
			return domain.SimulationResult{}, fmt.Errorf("scenario %q: only monthly bonus period is supported in this stage", scenarioName)
		}
		if scenario.BonusPayoutLagMonths == nil || *scenario.BonusPayoutLagMonths < 1 {
			return domain.SimulationResult{}, fmt.Errorf("scenario %q: bonus payout lag must be >= 1 in this stage", scenarioName)
		}
	}

	arQueue := newDueQueue()
	if err := arQueue.add(1, domain.Money(cfg.Company.OpeningAccountsReceivable)); err != nil {
		return domain.SimulationResult{}, fmt.Errorf("initialize AR queue: %w", err)
	}
	taxQueue := newDueQueue()
	bonuses := newBonusQueue()
	cashTotal := domain.Money(cfg.Company.StartingCash)
	restrictedBonusCash := domain.Money(0)
	monthlyResults := make([]domain.MonthlyResult, 0, months)

	var minimumUnrestrictedCash domain.Money
	var totalTaxesPaid domain.Money
	var risksEver domain.RiskFlags
	var totalGrossBonusAccrued domain.Money
	var totalBonusPayrollTaxAccrued domain.Money
	var totalGrossBonusPaid domain.Money
	var totalBonusPayrollTaxPaid domain.Money

	for month := 1; month <= months; month++ {
		env := deterministicEnvironment(cfg, month)
		workforce := model.CalculateWorkforce(cfg, behavior, env)
		pnl := model.CalculateOperatingPnL(cfg, workforce, env)

		openingAR := arQueue.outstandingBalance()
		collectedFromAR, err := arQueue.takeDue(month)
		if err != nil {
			return domain.SimulationResult{}, fmt.Errorf("month %d collect AR: %w", month, err)
		}
		openingTaxPayable := taxQueue.outstandingBalance()
		taxesPaid, err := taxQueue.takeDue(month)
		if err != nil {
			return domain.SimulationResult{}, fmt.Errorf("month %d pay tax: %w", month, err)
		}
		openingBonusPayable := bonuses.outstandingBalance()
		bonusPaid, err := bonuses.takeDue(month)
		if err != nil {
			return domain.SimulationResult{}, fmt.Errorf("month %d pay bonus: %w", month, err)
		}

		cashInput := domain.CashMonthInput{
			OpeningCashTotal:           cashTotal,
			OpeningAccountsReceivable:  openingAR,
			CashCollectedFromAR:        collectedFromAR,
			OpeningTaxPayable:          openingTaxPayable,
			TaxesPaidCash:              taxesPaid,
			OpeningRestrictedBonusCash: restrictedBonusCash,
			BonusDueGross:              bonusPaid.Gross,
			BonusDuePayrollTax:         bonusPaid.PayrollTax,
		}
		cash := model.CalculateCash(cfg, pnl, env, cashInput)
		if scenario.Type == "profit_share" {
			refreshBonusCashBalances(&cash, bonuses.outstandingBalance().total())
		}

		if err := arQueue.add(month+cfg.Cashflow.AccountsReceivableLagMonths, cash.AccountsReceivable.New); err != nil {
			return domain.SimulationResult{}, fmt.Errorf("month %d schedule AR: %w", month, err)
		}
		cash.AccountsReceivable.Closing = arQueue.outstandingBalance()

		var compensation *domain.CompensationState
		if scenario.Type == "profit_share" {
			updatedPnL, updatedCash, state, err := model.ApplyProfitShareAccrual(
				cfg,
				scenario,
				pnl,
				cash,
				model.ProfitShareAccrualInput{
					PeriodProfitBaseAccumulator:   pnl.OperatingProfitBeforeBonus,
					MonthsInBonusPeriod:           1,
					IsBonusPeriodEnd:              true,
					OpeningBonusPayableGross:      openingBonusPayable.Gross,
					OpeningBonusPayablePayrollTax: openingBonusPayable.PayrollTax,
					BonusPaidCash:                 bonusPaid.Gross,
					BonusPayrollTaxPaid:           bonusPaid.PayrollTax,
				},
			)
			if err != nil {
				return domain.SimulationResult{}, fmt.Errorf("month %d accrue bonus: %w", month, err)
			}
			pnl = updatedPnL
			cash = updatedCash

			dueMonth := month + *scenario.BonusPayoutLagMonths
			if err := bonuses.add(dueMonth, bonusDue{
				Gross:      state.GrossBonusPoolAccrued,
				PayrollTax: state.BonusPayrollTaxAccrued,
			}); err != nil {
				return domain.SimulationResult{}, fmt.Errorf("month %d schedule bonus: %w", month, err)
			}
			closingBonusPayable := bonuses.outstandingBalance()
			refreshBonusCashBalances(&cash, closingBonusPayable.total())
			state.ClosingBonusPayableGross = closingBonusPayable.Gross
			state.ClosingBonusPayablePayrollTax = closingBonusPayable.PayrollTax
			state.ClosingBonusPayable = closingBonusPayable.total()
			compensation = &state

			totalGrossBonusAccrued += state.GrossBonusPoolAccrued
			totalBonusPayrollTaxAccrued += state.BonusPayrollTaxAccrued
			totalGrossBonusPaid += state.BonusPaidCash
			totalBonusPayrollTaxPaid += state.BonusPayrollTaxPaid
		}

		pnl = model.AccrueProfitTax(cfg, pnl)
		cash = model.RecordProfitTaxPayable(cash, pnl.ProfitTaxAccrual)
		if err := taxQueue.add(month+cfg.Cashflow.ProfitTaxPaymentLagMonths, pnl.ProfitTaxAccrual); err != nil {
			return domain.SimulationResult{}, fmt.Errorf("month %d schedule tax: %w", month, err)
		}
		cash.TaxPayableClosing = taxQueue.outstandingBalance()

		risks := model.CalculateRiskFlags(cfg, cash)
		if compensation != nil {
			risks.ZeroBonusPeriod = compensation.IsBonusPeriodEnd && compensation.GrossBonusPoolAccrued == 0
			risks.BonusNotAccruedDueToCash = bonusWasCashCapped(cfg, scenario, *compensation)
		}
		monthly := domain.MonthlyResult{
			Scenario:        scenario.Name,
			BehaviorCase:    behaviorCase,
			EnvironmentCase: "normal_market",
			Currency:        cfg.Simulation.Currency,
			Month:           month,
			Environment:     env,
			Workforce:       workforce,
			PnL:             pnl,
			Cash:            cash,
			Compensation:    compensation,
			Risks:           risks,
		}
		bonusOutstanding := bonuses.outstandingBalance()
		if err := validateFiniteResult(monthly); err != nil {
			return domain.SimulationResult{}, fmt.Errorf("month %d: %w", month, err)
		}
		if err := validateMonthInvariants(cfg, scenario, monthly, arQueue.outstandingBalance(), taxQueue.outstandingBalance(), bonusOutstanding); err != nil {
			return domain.SimulationResult{}, fmt.Errorf("month %d: %w", month, err)
		}

		monthlyResults = append(monthlyResults, monthly)
		cashTotal = cash.ClosingCashTotal
		restrictedBonusCash = cash.RestrictedBonusCash
		totalTaxesPaid += cash.TaxesPaidCash
		if month == 1 || cash.ClosingUnrestrictedCash < minimumUnrestrictedCash {
			minimumUnrestrictedCash = cash.ClosingUnrestrictedCash
		}
		risksEver = mergeRiskFlags(risksEver, risks)
	}

	last := monthlyResults[len(monthlyResults)-1]
	result := domain.SimulationResult{
		Scenario:        scenario.Name,
		BehaviorCase:    behaviorCase,
		EnvironmentCase: "normal_market",
		Currency:        cfg.Simulation.Currency,
		MonthlyResults:  monthlyResults,
		TerminalSummary: domain.TerminalSummary{
			MonthsCompleted:               len(monthlyResults),
			FinalMonth:                    last.Month,
			ClosingCashTotal:              last.Cash.ClosingCashTotal,
			ClosingUnrestrictedCash:       last.Cash.ClosingUnrestrictedCash,
			MinimumUnrestrictedCash:       minimumUnrestrictedCash,
			OutstandingAccountsReceivable: arQueue.outstandingBalance(),
			ClosingTaxPayable:             taxQueue.outstandingBalance(),
			TotalTaxesPaidCash:            totalTaxesPaid,
			RiskFlagsEver:                 risksEver,
		},
	}
	if scenario.Type == "profit_share" {
		closingBonusPayable := bonuses.outstandingBalance()
		result.TerminalSummary.Compensation = &domain.CompensationSummary{
			TotalGrossBonusAccrued:        totalGrossBonusAccrued,
			TotalBonusPayrollTaxAccrued:   totalBonusPayrollTaxAccrued,
			TotalEmployerBonusCostAccrued: totalGrossBonusAccrued + totalBonusPayrollTaxAccrued,
			TotalGrossBonusPaid:           totalGrossBonusPaid,
			TotalBonusPayrollTaxPaid:      totalBonusPayrollTaxPaid,
			ClosingBonusPayableGross:      closingBonusPayable.Gross,
			ClosingBonusPayablePayrollTax: closingBonusPayable.PayrollTax,
			ClosingBonusPayable:           closingBonusPayable.total(),
			ClosingRestrictedBonusCash:    last.Cash.RestrictedBonusCash,
		}
	}
	if err := validateFiniteSummary(result.TerminalSummary); err != nil {
		return domain.SimulationResult{}, err
	}
	return result, nil
}

func refreshBonusCashBalances(cash *domain.CashState, restricted domain.Money) {
	cash.RestrictedBonusCash = restricted
	cash.ClosingUnrestrictedCash = cash.ClosingCashTotal - restricted
	cash.OwnerDistributableCash = 0
	if domain.MoneyLess(cash.RequiredCashReserve, cash.ClosingUnrestrictedCash) {
		cash.OwnerDistributableCash = cash.ClosingUnrestrictedCash - cash.RequiredCashReserve
	}
}

func selectedScenario(cfg config.Config, name string) (config.CompensationScenario, error) {
	for _, scenario := range cfg.CompensationScenarios {
		if scenario.Name == name {
			return scenario, nil
		}
	}
	return config.CompensationScenario{}, fmt.Errorf("unknown compensation scenario %q", name)
}

func bonusWasCashCapped(cfg config.Config, scenario config.CompensationScenario, state domain.CompensationState) bool {
	if !state.IsBonusPeriodEnd || scenario.ProfitSharePercent == nil {
		return false
	}
	potential := *scenario.ProfitSharePercent * float64(state.PeriodProfitBase)
	if scenario.BonusCapTotal != nil {
		potential = math.Min(potential, *scenario.BonusCapTotal)
	}
	if scenario.BonusCapPerEmployee != nil && scenario.EligibleEmployeesCount != nil {
		potential = math.Min(potential, *scenario.BonusCapPerEmployee*float64(*scenario.EligibleEmployeesCount))
	}
	maxGrossByCash := float64(state.CashBaseForTotalEmployerBonusCost) / (1 + cfg.Cashflow.BonusPayrollTaxRate)
	return domain.MoneyLess(state.GrossBonusPoolAccrued, domain.Money(potential)) && maxGrossByCash < potential
}

func deterministicEnvironment(cfg config.Config, month int) domain.EnvironmentMonth {
	exponent := float64(month - 1)
	return domain.EnvironmentMonth{
		Month:                    month,
		CumulativeMarketTrend:    math.Pow(1+cfg.Environment.MarketGrowthMonthly, exponent),
		MarketFactor:             1,
		CostInflationFactor:      math.Pow(1+cfg.Environment.CostInflationMonthly, exponent),
		LaborMarketFactor:        cfg.Environment.LaborMarketFactor,
		ShockHappened:            false,
		ShockRevenueMultiplier:   cfg.Environment.ShockRevenueMultiplier,
		ShockCost:                0,
		CollectionRateMultiplier: 1,
	}
}

func mergeRiskFlags(current, next domain.RiskFlags) domain.RiskFlags {
	return domain.RiskFlags{
		ReserveBreach:            current.ReserveBreach || next.ReserveBreach,
		LiquidityGap:             current.LiquidityGap || next.LiquidityGap,
		CashGap:                  current.CashGap || next.CashGap,
		Bankruptcy:               current.Bankruptcy || next.Bankruptcy,
		BonusNotAccruedDueToCash: current.BonusNotAccruedDueToCash || next.BonusNotAccruedDueToCash,
		ZeroBonusPeriod:          current.ZeroBonusPeriod || next.ZeroBonusPeriod,
	}
}
