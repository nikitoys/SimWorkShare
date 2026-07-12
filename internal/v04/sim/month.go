package sim

import (
	"fmt"
	"math"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

type runtimeQueues struct {
	AR                 *AmountQueue
	Tax                *AmountQueue
	Distribution       *DistributionQueue
	MemberRedemption   *AmountQueue
	CapacityActivation *AmountQueue
	GeneralArrears     *AmountQueue
}

func newRuntimeQueues(cfg v04config.Config) (*runtimeQueues, error) {
	queues := &runtimeQueues{
		AR:                 NewAmountQueue(),
		Tax:                NewAmountQueue(),
		Distribution:       NewDistributionQueue(),
		MemberRedemption:   NewAmountQueue(),
		CapacityActivation: NewAmountQueue(),
		GeneralArrears:     NewAmountQueue(),
	}
	if err := queues.AR.Add(1, cfg.CompanyEconomics.OpeningAccountsReceivable); err != nil {
		return nil, fmt.Errorf("initialize accounts receivable: %w", err)
	}
	return queues, nil
}

func initialScenarioState(cfg v04config.Config) domain.ScenarioState {
	state := domain.ScenarioState{
		Active:                    true,
		Headcount:                 cfg.CompanyEconomics.InitialHeadcount,
		RampCohorts:               make([]float64, cfg.Workforce.RampDurationMonths),
		CashTotal:                 cfg.CompanyEconomics.StartingCash,
		ProductiveCapacityRevenue: cfg.CompanyEconomics.InitialProductiveCapacityRevenueMonthly,
	}
	state.MinimumUnrestrictedCash = state.UnrestrictedCash()
	return state
}

func stepMonth(
	cfg v04config.Config,
	caseConfig ScenarioCase,
	run int,
	environment domain.EnvironmentMonth,
	state domain.ScenarioState,
	queues *runtimeQueues,
) (domain.MonthlyResult, domain.ScenarioState, error) {
	month := environment.Month
	if !state.Active && state.BankruptcyAbsorbed && cfg.Simulation.StopAfterBankruptcy {
		result := inactiveMonth(cfg, caseConfig, run, environment, state, queues)
		if err := ValidateMonthInvariants(cfg, result); err != nil {
			return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("inactive month %d: %w", month, err)
		}
		state.Month = month
		return result, state, nil
	}

	scenario := caseConfig.Scenario
	company := cfg.CompanyEconomics
	financing := cfg.Financing
	epsilon := cfg.Simulation.Epsilon
	costInflation := math.Pow(1+company.CostInflationMonthly, float64(month-1))

	capacityBegin := state.ProductiveCapacityRevenue
	capacityDueAtOpen := queues.CapacityActivation.TakeDue(month)
	capacityForProduction := CapacityAfterDepreciationAndDue(
		capacityBegin,
		company.CapacityDepreciationRateMonthly,
		capacityDueAtOpen,
	)

	governance, err := CalculateGovernance(GovernanceInput{
		HeadcountBegin:                state.Headcount,
		StandardHoursPerEmployeeMonth: company.StandardHoursPerEmployeeMonth,
		Parameters:                    scenario.Governance,
	})
	if err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d governance: %w", month, err)
	}
	behavior, err := CalculateBehavior(BehaviorInput{
		HeadcountBegin:                  state.Headcount,
		StandardHoursPerEmployeeMonth:   company.StandardHoursPerEmployeeMonth,
		IncomeHistory:                   state.IncomeHistory,
		MemberCapitalAccountsTotalBegin: state.MemberCapitalAccountsTotal,
		ZeroDistributionStreak:          state.ZeroDistributionStreak,
		LaborMarketFactor:               environment.LaborMarketFactor,
		Epsilon:                         epsilon,
		Governance:                      governance,
		Scenario:                        scenario,
		Case:                            caseConfig.Behavior,
		Workforce:                       cfg.Workforce,
		EmployeeRisk:                    cfg.EmployeeRisk,
	})
	if err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d behavior: %w", month, err)
	}

	forecast := CalculateProduction(ProductionInput{
		Company:                   company,
		Market:                    cfg.Market,
		Governance:                scenario.Governance,
		Environment:               environment,
		EffectiveEmployees:        0,
		ProductivityMultiplier:    behavior.ProductivityMultiplier,
		DecisionQualityMultiplier: governance.DecisionQualityMultiplier,
		DecisionDelayMonths:       governance.DecisionDelayMonths,
		CapacityLimit:             capacityForProduction,
	})
	debtService := CalculateDebtService(financing, state.DebtBalance)
	salaryReference := state.Headcount * company.BaseSalaryPerEmployeeMonthly * costInflation * (1 + company.SalaryPayrollTaxRate)
	fixedReference := company.FixedCostsMonthly * costInflation
	workforce, err := CalculateWorkforce(cfg, scenario, WorkforceInput{
		Run:                                   run,
		Month:                                 month,
		RandomKey:                             scenario.Name + "\x00" + caseConfig.BehaviorName,
		HeadcountBegin:                        state.Headcount,
		RampCohortsBegin:                      state.RampCohorts,
		TurnoverRateMonthly:                   behavior.TurnoverRateMonthly,
		UnrestrictedCashBegin:                 state.UnrestrictedCash(),
		SalaryCostsReferenceMonthly:           salaryReference,
		FixedCostsReferenceMonthly:            fixedReference,
		ScheduledDebtServiceReferenceMonthly:  debtService.InterestExpense + debtService.PrincipalDue,
		MarketDemandForecast:                  forecast.MarketDemand,
		ProductiveCapacityRevenueMonthlyBegin: capacityForProduction,
		ProductivityMultiplier:                behavior.ProductivityMultiplier,
		GovernanceAdminEquivalentEmployees:    governance.GovernanceAdminEquivalentEmployees,
	})
	if err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d workforce: %w", month, err)
	}

	production := CalculateProduction(ProductionInput{
		Company:                   company,
		Market:                    cfg.Market,
		Governance:                scenario.Governance,
		Environment:               environment,
		EffectiveEmployees:        workforce.EffectiveEmployees,
		ProductivityMultiplier:    behavior.ProductivityMultiplier,
		DecisionQualityMultiplier: governance.DecisionQualityMultiplier,
		DecisionDelayMonths:       governance.DecisionDelayMonths,
		CapacityLimit:             capacityForProduction,
	})
	costs := CalculateCosts(CostInput{
		Company:             company,
		Workforce:           cfg.Workforce,
		PaidEmployees:       workforce.PaidEmployees,
		Revenue:             production.Revenue,
		Hires:               workforce.Hires,
		VoluntaryLeavers:    workforce.VoluntaryLeavers,
		Layoffs:             workforce.DistressLayoffs,
		GovernanceCashCost:  governance.GovernanceCashCost,
		ShockCost:           environment.ShockCost,
		CostInflationFactor: costInflation,
	})
	operatingProfit := production.Revenue - costs.OperatingCostsBeforeAllocation
	profitBeforeDistribution := operatingProfit - debtService.InterestExpense
	positiveResultBase := math.Max(0, profitBeforeDistribution-scenario.ResultHurdleMonthly)

	openingAR := queues.AR.Balance()
	collectedOldAR := queues.AR.TakeDue(month)
	effectiveCollectionRate := domain.Clamp(
		cfg.Market.RevenueCollectionRateCurrentMonth*environment.CollectionRateMultiplier,
		0,
		1,
	)
	cashCollectedCurrent := production.Revenue * effectiveCollectionRate
	newAR := production.Revenue * (1 - effectiveCollectionRate) * (1 - cfg.Market.BadDebtRate)
	cashCollectedFromAR := collectedOldAR
	if cfg.Market.AccountsReceivableLagMonths == 0 {
		cashCollectedFromAR += newAR
	} else if err := queues.AR.Add(month+cfg.Market.AccountsReceivableLagMonths, newAR); err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d schedule AR: %w", month, err)
	}

	taxPayableBegin := queues.Tax.Balance()
	taxesDue := queues.Tax.TakeDue(month)
	distributionPayableBegin := queues.Distribution.Balance()
	if !domain.AlmostEqual(distributionPayableBegin.Total(), state.RestrictedDistributionCash, epsilon) {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d distribution payable and restricted cash disagree", month)
	}
	distributionDue := queues.Distribution.TakeDue(month)
	redemptionPayableBegin := queues.MemberRedemption.Balance()
	redemptionDue := queues.MemberRedemption.TakeDue(month)
	generalArrearsBegin := queues.GeneralArrears.Balance()
	generalArrearsDue := queues.GeneralArrears.TakeDue(month)
	redemptionAccrual := MemberCapitalRedemptionAccrual(
		state.MemberCapitalAccountsTotal,
		state.Headcount,
		workforce.VoluntaryLeavers,
		workforce.DistressLayoffs,
		financing.MemberCapitalRedemptionFractionOnExit,
		epsilon,
	)
	if financing.MemberCapitalRedemptionLagMonths == 0 {
		redemptionDue += redemptionAccrual
	} else if err := queues.MemberRedemption.Add(month+financing.MemberCapitalRedemptionLagMonths, redemptionAccrual); err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d schedule member redemption: %w", month, err)
	}

	creditLimit := EffectiveCreditLine(financing, scenario, environment.CreditMarketFactor)
	obligations := MandatoryObligations{
		PriorArrears:      generalArrearsDue,
		Salary:            costs.SalaryCost,
		SalaryPayrollTax:  costs.SalaryPayrollTax,
		FixedCosts:        costs.FixedCosts,
		VariableCosts:     costs.VariableCosts,
		WorkforceCosts:    costs.TurnoverAndWorkforceCost,
		GovernanceCosts:   governance.GovernanceCashCost,
		ShockCosts:        environment.ShockCost,
		Taxes:             taxesDue,
		Distribution:      distributionDue,
		MemberRedemptions: redemptionDue,
		Interest:          debtService.InterestExpense,
		Principal:         debtService.PrincipalDue,
	}
	cashBeforePayments := state.CashTotal + cashCollectedCurrent + cashCollectedFromAR
	unrestrictedBeforePayments := cashBeforePayments - state.RestrictedDistributionCash - state.RestrictedReserveCash
	funding := FundMandatoryGap(
		unrestrictedBeforePayments,
		obligations.GeneralTotal(),
		state.RestrictedReserveCash,
		financing.ReserveReleaseRateOnStress,
		creditLimit,
		state.DebtBalance,
	)
	settlement := SettleMandatory(MandatorySettlementInput{
		Obligations:                obligations,
		CashTotalBeforePayments:    cashBeforePayments,
		RestrictedDistributionCash: state.RestrictedDistributionCash,
		RestrictedReserveCash:      state.RestrictedReserveCash,
		Funding:                    funding,
	})
	if settlement.UnpaidCarryForward > epsilon {
		if err := queues.GeneralArrears.Add(month+1, settlement.UnpaidCarryForward); err != nil {
			return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d retain unpaid general obligations: %w", month, err)
		}
	}
	if unpaidTax := taxesDue - settlement.Paid.Taxes; unpaidTax > epsilon {
		if err := queues.Tax.Add(month+1, unpaidTax); err != nil {
			return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d retain unpaid tax: %w", month, err)
		}
	}
	if unpaid := subtractDistribution(distributionDue, settlement.Paid.Distribution); unpaid.Total() > epsilon {
		if err := queues.Distribution.Add(month+1, unpaid); err != nil {
			return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d retain unpaid distribution: %w", month, err)
		}
	}
	if unpaidRedemption := redemptionDue - settlement.Paid.MemberRedemptions; unpaidRedemption > epsilon {
		if err := queues.MemberRedemption.Add(month+1, unpaidRedemption); err != nil {
			return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d retain unpaid redemption: %w", month, err)
		}
	}

	unrestrictedBeforeAllocations := settlement.CashTotalAfterPayments -
		settlement.RestrictedDistributionCash - settlement.RestrictedReserveCash
	taxReserveEstimate := math.Max(0, profitBeforeDistribution) * company.ProfitTaxRate
	cashSafeBudget := math.Max(0, unrestrictedBeforeAllocations-workforce.RequiredCashReserveBegin-taxReserveEstimate)

	distributionPeriodBase, periodAccumulator, periodMonths := AdvanceDistributionPeriod(
		state.DistributionPeriodAccumulator,
		state.DistributionPeriodMonths,
		positiveResultBase,
		scenario.ProfitDistributionPeriodMonths,
	)
	rawAllocations := RawAllocations(scenario, positiveResultBase, distributionPeriodBase, workforce.HeadcountEnd)
	allocation, err := AllocatePositiveResult(
		rawAllocations,
		scenario.AllocationPriority,
		cashSafeBudget,
		financing.DistributionPayrollTaxRate,
	)
	if err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d allocate result: %w", month, err)
	}
	distributionPayrollTax := allocation.Actual.EmployeeCashDistribution * financing.DistributionPayrollTaxRate
	tax := CalculateTaxAndNetProfit(
		profitBeforeDistribution,
		allocation.Actual.EmployeeCashDistribution,
		distributionPayrollTax,
		company,
		financing,
	)

	cashAfterAccrualSettlements := settlement.CashTotalAfterPayments
	restrictedDistribution := settlement.RestrictedDistributionCash
	taxesPaid := settlement.Paid.Taxes
	distributionPaid := settlement.Paid.Distribution
	mandatoryScheduled := obligations.Total()
	mandatoryPaid := settlement.Paid.Total()
	unpaidMandatory := settlement.UnpaidTotal
	if company.ProfitTaxPaymentLagMonths == 0 {
		mandatoryScheduled += tax.ProfitTaxAccrual
		available := math.Max(0, cashAfterAccrualSettlements-restrictedDistribution-settlement.RestrictedReserveCash)
		paid := math.Min(tax.ProfitTaxAccrual, available)
		cashAfterAccrualSettlements -= paid
		taxesPaid += paid
		mandatoryPaid += paid
		unpaid := tax.ProfitTaxAccrual - paid
		unpaidMandatory += unpaid
		taxesDue += tax.ProfitTaxAccrual
		if unpaid > epsilon {
			if err := queues.Tax.Add(month+1, unpaid); err != nil {
				return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d retain same-month tax: %w", month, err)
			}
		}
	} else if err := queues.Tax.Add(month+company.ProfitTaxPaymentLagMonths, tax.ProfitTaxAccrual); err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d schedule tax: %w", month, err)
	}

	newDistribution := DistributionDue{
		Gross:      allocation.Actual.EmployeeCashDistribution,
		PayrollTax: distributionPayrollTax,
	}
	restrictedDistribution += newDistribution.Total()
	newDistributionPaid := DistributionDue{}
	if financing.EmployeeDistributionPayoutLagMonths == 0 {
		mandatoryScheduled += newDistribution.Total()
		paidTotal := math.Min(newDistribution.Total(), math.Max(0, cashAfterAccrualSettlements))
		if newDistribution.Total() > 0 {
			newDistributionPaid = newDistribution.Scale(paidTotal / newDistribution.Total())
		}
		cashAfterAccrualSettlements -= paidTotal
		restrictedDistribution -= paidTotal
		distributionPaid.Gross += newDistributionPaid.Gross
		distributionPaid.PayrollTax += newDistributionPaid.PayrollTax
		mandatoryPaid += paidTotal
		unpaid := subtractDistribution(newDistribution, newDistributionPaid)
		unpaidMandatory += unpaid.Total()
		distributionDue.Gross += newDistribution.Gross
		distributionDue.PayrollTax += newDistribution.PayrollTax
		if unpaid.Total() > epsilon {
			if err := queues.Distribution.Add(month+1, unpaid); err != nil {
				return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d retain same-month distribution: %w", month, err)
			}
		}
	} else if err := queues.Distribution.Add(month+financing.EmployeeDistributionPayoutLagMonths, newDistribution); err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d schedule distribution: %w", month, err)
	}

	debtAfterLiquidityAndPrincipal := state.DebtBalance + funding.CreditDraw - settlement.Paid.Principal
	growthCapital := CalculateExternalGrowthCapital(
		rawAllocations.Reinvestment,
		allocation.Actual.Reinvestment,
		financing,
		scenario,
		environment.CreditMarketFactor,
		creditLimit,
		debtAfterLiquidityAndPrincipal,
	)
	actualReinvestmentTotal := allocation.Actual.Reinvestment + growthCapital.Draw
	capacityAdded, _ := CapacityCreated(
		actualReinvestmentTotal,
		company,
		scenario.Governance,
		governance.DecisionQualityMultiplier,
	)
	effectiveInvestmentLag := company.InvestmentActivationLagMonths + int(math.Round(governance.DecisionDelayMonths))
	capacityAdditionsDue := capacityDueAtOpen
	capacityClose := capacityForProduction
	if effectiveInvestmentLag <= 0 {
		capacityClose += capacityAdded
		capacityAdditionsDue += capacityAdded
	} else if err := queues.CapacityActivation.Add(month+effectiveInvestmentLag, capacityAdded); err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d schedule capacity: %w", month, err)
	}

	restrictedReserveClose := settlement.RestrictedReserveCash + allocation.Actual.OrganizationalReserve
	cashClose := cashAfterAccrualSettlements - allocation.Actual.Reinvestment - allocation.Actual.ExternalDistribution +
		growthCapital.Draw - growthCapital.Draw
	debtClose := debtAfterLiquidityAndPrincipal + growthCapital.DebtDraw
	memberCapitalClose := math.Max(0, state.MemberCapitalAccountsTotal+allocation.Actual.MemberCapitalAllocation-redemptionAccrual)
	restrictedCashClose := restrictedDistribution + restrictedReserveClose
	unrestrictedClose := cashClose - restrictedCashClose
	unpaidToDate := state.UnpaidMandatoryToDate + unpaidMandatory

	risks := domain.RiskFlags{
		ReserveBreach:           unrestrictedClose < workforce.RequiredCashReserveBegin-epsilon,
		CashGap:                 cashClose < -epsilon,
		LiquidityDeficit:        unpaidMandatory > epsilon,
		CreditLimitBreach:       debtClose > creditLimit+epsilon,
		EmployeeDistributionCut: rawAllocations.EmployeeCashDistribution > allocation.Actual.EmployeeCashDistribution+epsilon,
		ReinvestmentUnderfunded: rawAllocations.Reinvestment > actualReinvestmentTotal+epsilon,
	}
	risks.Bankruptcy = risks.LiquidityDeficit || risks.CreditLimitBreach

	zeroDistributionStreak := 0
	if scenario.EmployeeCashDistributionRate > 0 {
		if allocation.Actual.EmployeeCashDistribution <= epsilon {
			zeroDistributionStreak = state.ZeroDistributionStreak + 1
		}
	}
	perEmployeeIncome := 0.0
	if workforce.PaidEmployees > epsilon {
		perEmployeeIncome = (settlement.Paid.Salary + distributionPaid.Gross) / workforce.PaidEmployees
	}
	incomeHistory := append(append([]domain.IncomeMonth(nil), state.IncomeHistory...), domain.IncomeMonth{
		FixedSalaryPaid:      settlement.Paid.Salary,
		CashDistributionPaid: distributionPaid.Gross,
		PerEmployeeIncome:    perEmployeeIncome,
	})
	if len(incomeHistory) > 12 {
		incomeHistory = append([]domain.IncomeMonth(nil), incomeHistory[len(incomeHistory)-12:]...)
	}
	minimumUnrestricted := state.MinimumUnrestrictedCash
	if month == 1 || unrestrictedClose < minimumUnrestricted {
		minimumUnrestricted = unrestrictedClose
	}
	firstShockMonth := state.FirstShockMonth
	if firstShockMonth == 0 && environment.ShockHappened {
		firstShockMonth = month
	}

	result := domain.MonthlyResult{
		Run:                                      run,
		Month:                                    month,
		Scenario:                                 scenario.Name,
		SystemType:                               scenario.SystemType,
		BehaviorCase:                             caseConfig.BehaviorName,
		MarketCase:                               domain.DefaultMarketCase,
		ActiveCompanyFlag:                        true,
		MarketTrend:                              environment.MarketTrend,
		MarketFactor:                             environment.MarketFactor,
		SeasonalityMultiplier:                    environment.SeasonalityMultiplier,
		ShockHappened:                            environment.ShockHappened,
		ShockCost:                                environment.ShockCost,
		EffectiveCollectionRate:                  effectiveCollectionRate,
		LaborMarketFactor:                        environment.LaborMarketFactor,
		CreditMarketFactor:                       environment.CreditMarketFactor,
		HeadcountBegin:                           state.Headcount,
		VoluntaryLeavers:                         workforce.VoluntaryLeavers,
		Layoffs:                                  workforce.DistressLayoffs,
		Hires:                                    workforce.Hires,
		HeadcountEnd:                             workforce.HeadcountEnd,
		Ramp:                                     workforce.Ramp,
		PaidEmployees:                            workforce.PaidEmployees,
		EffectiveEmployees:                       workforce.EffectiveEmployees,
		DesiredHeadcount:                         workforce.DesiredHeadcount,
		TurnoverRateAnnual:                       behavior.TurnoverRateAnnual,
		TurnoverRateMonthly:                      behavior.TurnoverRateMonthly,
		ProductivityUplift:                       behavior.ProductivityUplift,
		ProductivityMultiplier:                   behavior.ProductivityMultiplier,
		MotivationUpliftRaw:                      behavior.MotivationUpliftRaw,
		BehavioralTurnoverDeltaAnnual:            behavior.BehavioralTurnoverDeltaAnnual,
		GovernanceHours:                          governance.GovernanceHours,
		GovernanceAdminEquivalent:                governance.GovernanceAdminEquivalentEmployees,
		GovernanceCashCost:                       governance.GovernanceCashCost,
		DecisionDelayMonths:                      governance.DecisionDelayMonths,
		DecisionQualityMultiplier:                governance.DecisionQualityMultiplier,
		FairnessIndex:                            behavior.FairnessIndex,
		FreeRiderPenalty:                         behavior.FreeRiderPenalty,
		EmployeeRiskConcentration:                behavior.EmployeeRiskConcentration,
		IncomeVolatilityIndex12M:                 behavior.IncomeVolatilityIndex12M,
		MarketDemand:                             production.MarketDemand,
		MarketDemandForecast:                     forecast.MarketDemand,
		LaborRevenueCapacity:                     production.LaborRevenueCapacity,
		ProductiveCapacityBegin:                  capacityBegin,
		CapacityDepreciation:                     capacityBegin * company.CapacityDepreciationRateMonthly,
		CapacityAdditionsDue:                     capacityAdditionsDue,
		ProductiveCapacityRevenueMonthly:         capacityForProduction,
		Revenue:                                  production.Revenue,
		SalaryCost:                               costs.SalaryCost,
		SalaryPayrollTax:                         costs.SalaryPayrollTax,
		FixedCosts:                               costs.FixedCosts,
		VariableCosts:                            costs.VariableCosts,
		HiringCost:                               costs.HiringCost,
		ExitCost:                                 costs.ExitCost,
		LayoffCost:                               costs.LayoffCost,
		TurnoverAndWorkforceCost:                 costs.TurnoverAndWorkforceCost,
		OperatingCostsBeforeAllocation:           costs.OperatingCostsBeforeAllocation,
		OperatingProfitBeforeAllocation:          operatingProfit,
		InterestExpense:                          debtService.InterestExpense,
		ProfitBeforeTaxBeforeDistribution:        profitBeforeDistribution,
		PositiveResultBase:                       positiveResultBase,
		TaxableProfit:                            tax.TaxableProfit,
		ProfitTaxAccrual:                         tax.ProfitTaxAccrual,
		NetProfitAfterTaxAndEmployeeDistribution: tax.NetProfitAfterTaxAndDistribution,
		OpeningAccountsReceivable:                openingAR,
		CashCollectedCurrent:                     cashCollectedCurrent,
		CashCollectedFromAR:                      cashCollectedFromAR,
		NewAccountsReceivable:                    newAR,
		ClosingAccountsReceivable:                queues.AR.Balance(),
		TaxPayableBegin:                          taxPayableBegin,
		TaxesDue:                                 taxesDue,
		TaxesPaid:                                taxesPaid,
		TaxPayableClose:                          queues.Tax.Balance(),
		EmployeeDistributionDueGross:             distributionDue.Gross,
		EmployeeDistributionDuePayrollTax:        distributionDue.PayrollTax,
		EmployeeCashDistributionPaid:             distributionPaid.Gross,
		EmployeeDistributionPayrollTaxPaid:       distributionPaid.PayrollTax,
		EmployeeDistributionPayableClose:         queues.Distribution.Balance().Total(),
		MemberCapitalRedemptionPayableBegin:      redemptionPayableBegin,
		MemberCapitalRedemptionDue:               redemptionDue,
		MemberCapitalRedemptionPaid:              settlement.Paid.MemberRedemptions,
		MemberCapitalRedemptionPayableClose:      queues.MemberRedemption.Balance(),
		RequiredCashReserve:                      workforce.RequiredCashReserveBegin,
		CashTotalBegin:                           state.CashTotal,
		RestrictedDistributionBegin:              state.RestrictedDistributionCash,
		RestrictedReserveBegin:                   state.RestrictedReserveCash,
		UnrestrictedCashBegin:                    state.UnrestrictedCash(),
		MandatoryCashScheduled:                   mandatoryScheduled,
		MandatoryCashPayments:                    mandatoryPaid,
		GeneralMandatoryArrearsBegin:             generalArrearsBegin,
		GeneralMandatoryCurrentScheduled:         obligations.CurrentCarryForwardTotal(),
		GeneralMandatoryPayments:                 settlement.Paid.CarryForwardTotal(),
		GeneralMandatoryArrearsClose:             queues.GeneralArrears.Balance(),
		UnpaidMandatoryObligations:               unpaidMandatory,
		UnpaidMandatoryObligationsToDate:         unpaidToDate,
		RestrictedReserveReleased:                funding.RestrictedReserveReleased,
		CreditLineLimit:                          creditLimit,
		CreditDrawForLiquidity:                   funding.CreditDraw,
		CashAfterMandatory:                       settlement.CashTotalAfterPayments,
		UnrestrictedCashBeforeAllocations:        unrestrictedBeforeAllocations,
		TaxReserveEstimate:                       taxReserveEstimate,
		CashSafeAllocationBudget:                 cashSafeBudget,
		RawAllocations:                           rawAllocations,
		ActualAllocations:                        allocation.Actual,
		EmployeeCashDistributionAccrued:          allocation.Actual.EmployeeCashDistribution,
		DistributionPayrollTaxAccrued:            distributionPayrollTax,
		RestrictedDistributionCashNew:            newDistribution.Total(),
		MemberCapitalBegin:                       state.MemberCapitalAccountsTotal,
		MemberCapitalAllocation:                  allocation.Actual.MemberCapitalAllocation,
		MemberCapitalRedemptionAccrual:           redemptionAccrual,
		MemberCapitalClose:                       memberCapitalClose,
		ReinvestmentCashPaid:                     allocation.Actual.Reinvestment,
		ExternalGrowthCapitalDraw:                growthCapital.Draw,
		ExternalGrowthCapitalSpent:               growthCapital.Draw,
		ExternalDistributionPaid:                 allocation.Actual.ExternalDistribution,
		OrganizationalReserveAllocation:          allocation.Actual.OrganizationalReserve,
		CapacityAddedByInvestment:                capacityAdded,
		EffectiveInvestmentLag:                   effectiveInvestmentLag,
		PrincipalDue:                             debtService.PrincipalDue,
		PrincipalPaid:                            settlement.Paid.Principal,
		DebtBalanceBegin:                         state.DebtBalance,
		DebtBalanceClose:                         debtClose,
		CashTotalClose:                           cashClose,
		RestrictedDistributionClose:              restrictedDistribution,
		RestrictedReserveClose:                   restrictedReserveClose,
		RestrictedCashClose:                      restrictedCashClose,
		UnrestrictedCashClose:                    unrestrictedClose,
		ProductiveCapacityClose:                  capacityClose,
		Risks:                                    risks,
	}

	next := domain.ScenarioState{
		Month:                         month,
		Active:                        !(risks.Bankruptcy && cfg.Simulation.StopAfterBankruptcy),
		BankruptcyAbsorbed:            state.BankruptcyAbsorbed || risks.Bankruptcy,
		Headcount:                     workforce.HeadcountEnd,
		RampCohorts:                   append([]float64(nil), workforce.Ramp.Close...),
		CashTotal:                     cashClose,
		RestrictedDistributionCash:    restrictedDistribution,
		RestrictedReserveCash:         restrictedReserveClose,
		DebtBalance:                   debtClose,
		ProductiveCapacityRevenue:     capacityClose,
		MemberCapitalAccountsTotal:    memberCapitalClose,
		DistributionPeriodAccumulator: periodAccumulator,
		DistributionPeriodMonths:      periodMonths,
		IncomeHistory:                 incomeHistory,
		ZeroDistributionStreak:        zeroDistributionStreak,
		MinimumUnrestrictedCash:       minimumUnrestricted,
		UnpaidMandatoryToDate:         unpaidToDate,
		FirstShockMonth:               firstShockMonth,
	}
	if err := ValidateMonthInvariants(cfg, result); err != nil {
		return domain.MonthlyResult{}, domain.ScenarioState{}, fmt.Errorf("month %d invariants: %w", month, err)
	}
	return result, next, nil
}

func subtractDistribution(total, paid DistributionDue) DistributionDue {
	return DistributionDue{
		Gross:      math.Max(0, total.Gross-paid.Gross),
		PayrollTax: math.Max(0, total.PayrollTax-paid.PayrollTax),
	}
}

func inactiveMonth(
	cfg v04config.Config,
	caseConfig ScenarioCase,
	run int,
	environment domain.EnvironmentMonth,
	state domain.ScenarioState,
	queues *runtimeQueues,
) domain.MonthlyResult {
	restricted := state.RestrictedCashTotal()
	generalArrears := queues.GeneralArrears.Balance()
	return domain.MonthlyResult{
		Run:                                 run,
		Month:                               environment.Month,
		Scenario:                            caseConfig.Scenario.Name,
		SystemType:                          caseConfig.Scenario.SystemType,
		BehaviorCase:                        caseConfig.BehaviorName,
		MarketCase:                          domain.DefaultMarketCase,
		ActiveCompanyFlag:                   false,
		MarketTrend:                         environment.MarketTrend,
		MarketFactor:                        environment.MarketFactor,
		SeasonalityMultiplier:               environment.SeasonalityMultiplier,
		ShockHappened:                       environment.ShockHappened,
		ShockCost:                           environment.ShockCost,
		LaborMarketFactor:                   environment.LaborMarketFactor,
		CreditMarketFactor:                  environment.CreditMarketFactor,
		HeadcountBegin:                      state.Headcount,
		HeadcountEnd:                        state.Headcount,
		Ramp:                                domain.RampState{Begin: append([]float64(nil), state.RampCohorts...), Close: append([]float64(nil), state.RampCohorts...)},
		ProductivityMultiplier:              1,
		ProductiveCapacityBegin:             state.ProductiveCapacityRevenue,
		ProductiveCapacityRevenueMonthly:    state.ProductiveCapacityRevenue,
		ProductiveCapacityClose:             state.ProductiveCapacityRevenue,
		OpeningAccountsReceivable:           queues.AR.Balance(),
		ClosingAccountsReceivable:           queues.AR.Balance(),
		TaxPayableBegin:                     queues.Tax.Balance(),
		TaxPayableClose:                     queues.Tax.Balance(),
		EmployeeDistributionPayableClose:    queues.Distribution.Balance().Total(),
		MemberCapitalRedemptionPayableBegin: queues.MemberRedemption.Balance(),
		MemberCapitalRedemptionPayableClose: queues.MemberRedemption.Balance(),
		CashTotalBegin:                      state.CashTotal,
		RestrictedDistributionBegin:         state.RestrictedDistributionCash,
		RestrictedReserveBegin:              state.RestrictedReserveCash,
		UnrestrictedCashBegin:               state.UnrestrictedCash(),
		GeneralMandatoryArrearsBegin:        generalArrears,
		GeneralMandatoryArrearsClose:        generalArrears,
		UnpaidMandatoryObligationsToDate:    state.UnpaidMandatoryToDate,
		MemberCapitalBegin:                  state.MemberCapitalAccountsTotal,
		MemberCapitalClose:                  state.MemberCapitalAccountsTotal,
		DebtBalanceBegin:                    state.DebtBalance,
		DebtBalanceClose:                    state.DebtBalance,
		CashTotalClose:                      state.CashTotal,
		RestrictedDistributionClose:         state.RestrictedDistributionCash,
		RestrictedReserveClose:              state.RestrictedReserveCash,
		RestrictedCashClose:                 restricted,
		UnrestrictedCashClose:               state.UnrestrictedCash(),
		Risks:                               domain.RiskFlags{Bankruptcy: true},
	}
}
