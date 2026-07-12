package sim

import (
	"fmt"
	"math"
	"sort"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

const (
	ClassificationUnclassified        = "unclassified"
	ClassificationReference           = "reference"
	ClassificationDevelopmentDominant = "development_dominant"
	ClassificationDevelopmentTradeoff = "development_tradeoff"
	ClassificationRiskConstrained     = "risk_constrained"
)

const (
	MetricSustainableDevelopmentValue = "sustainable_development_value_proxy"
	MetricCumulativeRevenue           = "cumulative_revenue"
	MetricCumulativeOperatingProfit   = "cumulative_operating_profit"
	MetricCumulativeNetProfit         = "cumulative_net_profit"
	MetricCashEndTotal                = "cash_end_total"
	MetricCashEndUnrestricted         = "cash_end_unrestricted"
	MetricFinalHeadcount              = "final_headcount"
	MetricProductiveCapacityGrowth    = "productive_capacity_growth_rate"
	MetricProductiveCapacityAdded     = "productive_capacity_added_total"
	MetricCapacityCAGR                = "capacity_cagr"
	MetricAverageEmployeeIncome       = "average_employee_income_monthly"
	MetricRiskAdjustedEmployeeIncome  = "risk_adjusted_employee_income"
)

// BuildRunTerminalSummaries builds one terminal record for every configured
// horizon. months must contain one run, scenario and behavior case.
func BuildRunTerminalSummaries(cfg v04config.Config, months []domain.MonthlyResult) ([]domain.RunTerminalSummary, error) {
	if len(cfg.Simulation.HorizonsMonths) == 0 {
		return nil, fmt.Errorf("simulation.horizons_months: at least one horizon is required")
	}
	horizons := append([]int(nil), cfg.Simulation.HorizonsMonths...)
	sort.Ints(horizons)
	result := make([]domain.RunTerminalSummary, 0, len(horizons))
	previous := -1
	for _, horizon := range horizons {
		if horizon == previous {
			continue
		}
		summary, err := BuildRunTerminalSummary(cfg, months, horizon)
		if err != nil {
			return nil, err
		}
		result = append(result, summary)
		previous = horizon
	}
	return result, nil
}

// BuildRunTerminalSummary calculates section 15 metrics through horizon.
// Undefined ratios use the neutral value zero; all returned numeric values are
// finite. The function rejects incomplete or mixed one-case monthly paths.
func BuildRunTerminalSummary(cfg v04config.Config, months []domain.MonthlyResult, horizon int) (domain.RunTerminalSummary, error) {
	if horizon <= 0 {
		return domain.RunTerminalSummary{}, fmt.Errorf("horizon_months: must be positive")
	}
	if len(months) == 0 {
		return domain.RunTerminalSummary{}, fmt.Errorf("monthly_results: must not be empty")
	}

	path := append([]domain.MonthlyResult(nil), months...)
	sort.Slice(path, func(i, j int) bool { return path[i].Month < path[j].Month })
	identity := path[0]
	for i, month := range path {
		if month.Run != identity.Run || month.Scenario != identity.Scenario ||
			month.SystemType != identity.SystemType || month.BehaviorCase != identity.BehaviorCase ||
			marketCase(month.MarketCase) != marketCase(identity.MarketCase) {
			return domain.RunTerminalSummary{}, fmt.Errorf("monthly_results[%d]: mixed run or scenario case", i)
		}
	}

	selected := make([]domain.MonthlyResult, 0, horizon)
	for _, month := range path {
		if month.Month <= horizon {
			selected = append(selected, month)
		}
	}
	if len(selected) != horizon {
		return domain.RunTerminalSummary{}, fmt.Errorf("monthly_results: horizon %d requires %d monthly records, got %d", horizon, horizon, len(selected))
	}
	for i, month := range selected {
		expected := i + 1
		if month.Month != expected {
			return domain.RunTerminalSummary{}, fmt.Errorf("monthly_results[%d].month: expected %d, got %d", i, expected, month.Month)
		}
	}

	epsilon := cfg.Simulation.Epsilon
	if !domain.Finite(epsilon) || epsilon <= 0 {
		epsilon = 1e-9
	}

	var cumulativeRevenue float64
	var cumulativeOperatingProfit float64
	var cumulativeNetProfit float64
	var productivitySum float64
	var turnoverAnnualSum float64
	var voluntaryLeavers float64
	var layoffs float64
	var hires float64
	var hiringCosts float64
	var employeeDistribution float64
	var reinvestmentCash float64
	var externalGrowthCapital float64
	var actualReinvestment float64
	var rawReinvestment float64
	var capacityAdded float64
	var employeeRiskSum float64
	var employeeIncomePaid float64
	var paidEmployeeMonths float64
	var riskEver domain.RiskFlags
	var incomeMonths []float64
	hadLiquidityDeficit := false
	hadBankruptcy := false
	hadShock := false
	minimumUnrestrictedCash := selected[0].UnrestrictedCashClose

	for i, month := range selected {
		values := []struct {
			name  string
			value float64
		}{
			{"revenue", month.Revenue},
			{"operating_profit_before_allocation", month.OperatingProfitBeforeAllocation},
			{"net_profit_after_tax_and_employee_distribution", month.NetProfitAfterTaxAndEmployeeDistribution},
			{"paid_employees", month.PaidEmployees},
			{"turnover_rate_annual", month.TurnoverRateAnnual},
			{"voluntary_leavers", month.VoluntaryLeavers},
			{"layoffs", month.Layoffs},
			{"hires", month.Hires},
			{"hiring_cost", month.HiringCost},
			{"salary_cost", month.SalaryCost},
			{"mandatory_cash_payments", month.MandatoryCashPayments},
			{"employee_cash_distribution_paid", month.EmployeeCashDistributionPaid},
			{"member_capital_close", month.MemberCapitalClose},
			{"reinvestment_cash_paid", month.ReinvestmentCashPaid},
			{"external_growth_capital_draw", month.ExternalGrowthCapitalDraw},
			{"raw_allocations.reinvestment", month.RawAllocations.Reinvestment},
			{"actual_allocations.reinvestment", month.ActualAllocations.Reinvestment},
			{"capacity_added_by_investment", month.CapacityAddedByInvestment},
			{"productive_capacity_begin", month.ProductiveCapacityBegin},
			{"productive_capacity_close", month.ProductiveCapacityClose},
			{"cash_total_close", month.CashTotalClose},
			{"unrestricted_cash_close", month.UnrestrictedCashClose},
			{"debt_balance_close", month.DebtBalanceClose},
			{"unpaid_mandatory_obligations_to_date", month.UnpaidMandatoryObligationsToDate},
			{"member_capital_redemption_payable_close", month.MemberCapitalRedemptionPayableClose},
			{"headcount_end", month.HeadcountEnd},
			{"employee_risk_concentration_index", month.EmployeeRiskConcentration},
		}
		for _, value := range values {
			if !domain.Finite(value.value) {
				return domain.RunTerminalSummary{}, fmt.Errorf("monthly_results[%d].%s: must be finite", i, value.name)
			}
		}

		cumulativeRevenue += month.Revenue
		cumulativeOperatingProfit += month.OperatingProfitBeforeAllocation
		cumulativeNetProfit += month.NetProfitAfterTaxAndEmployeeDistribution
		if month.PaidEmployees > epsilon {
			productivitySum += month.Revenue / month.PaidEmployees
		}
		turnoverAnnualSum += month.TurnoverRateAnnual
		voluntaryLeavers += month.VoluntaryLeavers
		layoffs += month.Layoffs
		hires += month.Hires
		hiringCosts += month.HiringCost
		employeeDistribution += month.EmployeeCashDistributionPaid
		reinvestmentCash += month.ReinvestmentCashPaid
		externalGrowthCapital += month.ExternalGrowthCapitalDraw
		rawReinvestment += month.RawAllocations.Reinvestment
		actualReinvestment += month.ActualAllocations.Reinvestment + month.ExternalGrowthCapitalDraw
		capacityAdded += month.CapacityAddedByInvestment
		employeeRiskSum += month.EmployeeRiskConcentration
		minimumUnrestrictedCash = math.Min(minimumUnrestrictedCash, month.UnrestrictedCashClose)

		// Salary is first in the mandatory-payment order. This reconstructs
		// actual fixed salary paid without treating an unpaid accrual as income.
		salaryPaid := math.Min(math.Max(0, month.SalaryCost), math.Max(0, month.MandatoryCashPayments))
		monthlyIncome := 0.0
		if month.PaidEmployees > epsilon {
			monthlyIncome = (salaryPaid + math.Max(0, month.EmployeeCashDistributionPaid)) / month.PaidEmployees
			employeeIncomePaid += salaryPaid + math.Max(0, month.EmployeeCashDistributionPaid)
			paidEmployeeMonths += month.PaidEmployees
		}
		incomeMonths = append(incomeMonths, monthlyIncome)

		riskEver = riskEver.Merge(month.Risks)
		if month.Risks.LiquidityDeficit || month.UnpaidMandatoryObligations > epsilon {
			hadLiquidityDeficit = true
		}
		if month.Risks.Bankruptcy {
			hadBankruptcy = true
		}
		if !month.ActiveCompanyFlag && !month.Risks.Bankruptcy {
			hadBankruptcy = true
		}
		if month.ShockHappened && month.ActiveCompanyFlag {
			hadShock = true
		}
	}

	last := selected[len(selected)-1]
	if !last.ActiveCompanyFlag {
		hadBankruptcy = true
	}
	if hadLiquidityDeficit {
		riskEver.LiquidityDeficit = true
	}
	if hadBankruptcy {
		riskEver.Bankruptcy = true
	}

	averageIncome := 0.0
	if paidEmployeeMonths > epsilon {
		averageIncome = employeeIncomePaid / paidEmployeeMonths
	}
	incomeVolatility := populationStdDev(incomeMonths, mean(incomeMonths))
	riskAdjustedIncome := averageIncome - cfg.Analysis.VolatilityPenaltyLambda*incomeVolatility
	if !domain.Finite(riskAdjustedIncome) {
		return domain.RunTerminalSummary{}, fmt.Errorf("risk_adjusted_employee_income: must be finite")
	}

	capacityStart := selected[0].ProductiveCapacityBegin
	capacityGrowth := neutralGrowth(last.ProductiveCapacityClose, capacityStart, epsilon)
	capacityCAGR := 0.0
	revenueCAGR := 0.0
	if horizon > 12 {
		capacityCAGR = neutralCAGR(last.ProductiveCapacityClose, capacityStart, 12/float64(horizon), epsilon)
		firstTwelveRevenue := 0.0
		lastTwelveRevenue := 0.0
		for _, month := range selected[:12] {
			firstTwelveRevenue += month.Revenue
		}
		for _, month := range selected[len(selected)-12:] {
			lastTwelveRevenue += month.Revenue
		}
		revenueCAGR = neutralCAGR(lastTwelveRevenue, firstTwelveRevenue, 12/float64(horizon-12), epsilon)
	}

	underfundingRate := 0.0
	if rawReinvestment > epsilon {
		underfundingRate = domain.Clamp((rawReinvestment-actualReinvestment)/rawReinvestment, 0, 1)
	}
	capacityConversion := math.Max(epsilon, cfg.CompanyEconomics.CapacityRevenueCreatedPerCurrencyInvested)
	capacityReplacementCost := last.ProductiveCapacityClose / capacityConversion
	sustainableValue := last.UnrestrictedCashClose + capacityReplacementCost - last.DebtBalanceClose -
		last.UnpaidMandatoryObligationsToDate - last.MemberCapitalRedemptionPayableClose
	if !domain.Finite(sustainableValue) {
		return domain.RunTerminalSummary{}, fmt.Errorf("sustainable_development_value_proxy: must be finite")
	}
	shockSurvivalEvaluable, shockSurvived := evaluateShockSurvival(path, horizon)

	return domain.RunTerminalSummary{
		Run:                                   identity.Run,
		Scenario:                              identity.Scenario,
		SystemType:                            identity.SystemType,
		BehaviorCase:                          identity.BehaviorCase,
		MarketCase:                            marketCase(identity.MarketCase),
		HorizonMonths:                         horizon,
		ActiveCompanyFlag:                     last.ActiveCompanyFlag,
		CumulativeRevenue:                     cumulativeRevenue,
		CumulativeOperatingProfit:             cumulativeOperatingProfit,
		CumulativeNetProfit:                   cumulativeNetProfit,
		ProductivityPerEmployee:               productivitySum / float64(horizon),
		TurnoverRateAnnualAverage:             turnoverAnnualSum / float64(horizon),
		VoluntaryLeaversTotal:                 voluntaryLeavers,
		LayoffsTotal:                          layoffs,
		HiresTotal:                            hires,
		HiringAndOnboardingCostsTotal:         hiringCosts,
		AverageEmployeeIncomeMonthly:          averageIncome,
		EmployeeIncomeVolatility:              incomeVolatility,
		RiskAdjustedEmployeeIncome:            riskAdjustedIncome,
		EmployeeCashDistributionTotal:         employeeDistribution,
		MemberCapitalAccountsTotal:            last.MemberCapitalClose,
		ReinvestmentTotalCash:                 reinvestmentCash,
		ExternalGrowthCapitalTotal:            externalGrowthCapital,
		ActualReinvestmentTotal:               actualReinvestment,
		ReinvestmentUnderfundingRate:          underfundingRate,
		ProductiveCapacityAddedTotal:          capacityAdded,
		ProductiveCapacityGrowthRate:          capacityGrowth,
		CashEndTotal:                          last.CashTotalClose,
		CashEndUnrestricted:                   last.UnrestrictedCashClose,
		MinimumUnrestrictedCash:               minimumUnrestrictedCash,
		DebtBalance:                           last.DebtBalanceClose,
		UnpaidObligations:                     last.UnpaidMandatoryObligationsToDate,
		MemberCapitalRedemptionDue:            last.MemberCapitalRedemptionPayableClose,
		FinalHeadcount:                        last.HeadcountEnd,
		RevenueCAGR:                           revenueCAGR,
		CapacityCAGR:                          capacityCAGR,
		EmployeeRiskConcentrationIndexAverage: employeeRiskSum / float64(horizon),
		SustainableDevelopmentValueProxy:      sustainableValue,
		HadLiquidityDeficit:                   hadLiquidityDeficit,
		HadBankruptcy:                         hadBankruptcy,
		HadShock:                              hadShock,
		ShockSurvivalEvaluable:                shockSurvivalEvaluable,
		ShockSurvived:                         shockSurvived,
		RiskFlagsEver:                         riskEver,
	}, nil
}

// AggregateScenarioSummaries aggregates run-level records by scenario case and
// horizon. Every percentile uses deterministic linear interpolation.
func AggregateScenarioSummaries(runs []domain.RunTerminalSummary) []domain.ScenarioSummary {
	type key struct {
		scenario, systemType, behavior, market string
		horizon                                int
	}
	groups := make(map[key][]domain.RunTerminalSummary)
	for _, run := range runs {
		k := key{run.Scenario, run.SystemType, run.BehaviorCase, marketCase(run.MarketCase), run.HorizonMonths}
		groups[k] = append(groups[k], run)
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].scenario != keys[j].scenario {
			return keys[i].scenario < keys[j].scenario
		}
		if keys[i].behavior != keys[j].behavior {
			return keys[i].behavior < keys[j].behavior
		}
		if keys[i].market != keys[j].market {
			return keys[i].market < keys[j].market
		}
		if keys[i].horizon != keys[j].horizon {
			return keys[i].horizon < keys[j].horizon
		}
		return keys[i].systemType < keys[j].systemType
	})

	result := make([]domain.ScenarioSummary, 0, len(keys))
	for _, k := range keys {
		group := groups[k]
		values := func(extract func(domain.RunTerminalSummary) float64) []float64 {
			out := make([]float64, len(group))
			for i, run := range group {
				out[i] = finiteOrZero(extract(run))
			}
			return out
		}
		countLiquidity := 0
		countBankruptcy := 0
		shockRuns := 0
		shockSurvivors := 0
		for _, run := range group {
			if run.HadLiquidityDeficit {
				countLiquidity++
			}
			if run.HadBankruptcy {
				countBankruptcy++
			}
			if run.ShockSurvivalEvaluable {
				shockRuns++
				if run.ShockSurvived {
					shockSurvivors++
				}
			}
		}
		cumulativeRevenue := values(func(r domain.RunTerminalSummary) float64 { return r.CumulativeRevenue })
		var shockSurvivalRate *float64
		if shockRuns > 0 {
			value := probability(shockSurvivors, shockRuns)
			shockSurvivalRate = &value
		}
		result = append(result, domain.ScenarioSummary{
			Scenario:                              k.scenario,
			SystemType:                            k.systemType,
			BehaviorCase:                          k.behavior,
			MarketCase:                            k.market,
			HorizonMonths:                         k.horizon,
			Runs:                                  len(group),
			MedianCumulativeRevenue:               Percentile(cumulativeRevenue, 0.50),
			P10CumulativeRevenue:                  Percentile(cumulativeRevenue, 0.10),
			P90CumulativeRevenue:                  Percentile(cumulativeRevenue, 0.90),
			MedianOperatingProfit:                 Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.CumulativeOperatingProfit }), 0.50),
			MedianNetProfit:                       Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.CumulativeNetProfit }), 0.50),
			MedianProductivityPerEmployee:         Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.ProductivityPerEmployee }), 0.50),
			TurnoverRateAnnualAverage:             Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.TurnoverRateAnnualAverage }), 0.50),
			HiringAndOnboardingCostsTotal:         Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.HiringAndOnboardingCostsTotal }), 0.50),
			AverageEmployeeIncomeMonthly:          Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.AverageEmployeeIncomeMonthly }), 0.50),
			RiskAdjustedEmployeeIncome:            Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.RiskAdjustedEmployeeIncome }), 0.50),
			EmployeeCashDistributionTotal:         Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.EmployeeCashDistributionTotal }), 0.50),
			MemberCapitalAccountsTotal:            Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.MemberCapitalAccountsTotal }), 0.50),
			ReinvestmentTotalCash:                 Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.ReinvestmentTotalCash }), 0.50),
			ProductiveCapacityGrowthRate:          Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.ProductiveCapacityGrowthRate }), 0.50),
			CashEndTotalMedian:                    Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.CashEndTotal }), 0.50),
			CashEndUnrestrictedP10:                Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.CashEndUnrestricted }), 0.10),
			MinimumUnrestrictedCashP10:            Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.MinimumUnrestrictedCash }), 0.10),
			LiquidityDeficitProbability:           probability(countLiquidity, len(group)),
			BankruptcyProbability:                 probability(countBankruptcy, len(group)),
			ShockSurvivalRate:                     shockSurvivalRate,
			FinalHeadcountMedian:                  Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.FinalHeadcount }), 0.50),
			RevenueCAGRMedian:                     Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.RevenueCAGR }), 0.50),
			CapacityCAGRMedian:                    Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.CapacityCAGR }), 0.50),
			EmployeeRiskConcentrationIndexAverage: Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.EmployeeRiskConcentrationIndexAverage }), 0.50),
			SustainableDevelopmentValueMedian:     Percentile(values(func(r domain.RunTerminalSummary) float64 { return r.SustainableDevelopmentValueProxy }), 0.50),
			Classification:                        ClassificationUnclassified,
		})
	}
	return result
}

type pairedRunKey struct {
	run                        int
	scenario, behavior, market string
	horizon                    int
}

type pairedGroupKey struct {
	scenario, behavior           string
	reference, referenceBehavior string
	market                       string
	horizon                      int
	metric                       string
}

type pairedMetric struct {
	name  string
	value func(domain.RunTerminalSummary) float64
}

var pairedMetrics = []pairedMetric{
	{MetricAverageEmployeeIncome, func(r domain.RunTerminalSummary) float64 { return r.AverageEmployeeIncomeMonthly }},
	{MetricCapacityCAGR, func(r domain.RunTerminalSummary) float64 { return r.CapacityCAGR }},
	{MetricCashEndTotal, func(r domain.RunTerminalSummary) float64 { return r.CashEndTotal }},
	{MetricCashEndUnrestricted, func(r domain.RunTerminalSummary) float64 { return r.CashEndUnrestricted }},
	{MetricCumulativeNetProfit, func(r domain.RunTerminalSummary) float64 { return r.CumulativeNetProfit }},
	{MetricCumulativeOperatingProfit, func(r domain.RunTerminalSummary) float64 { return r.CumulativeOperatingProfit }},
	{MetricCumulativeRevenue, func(r domain.RunTerminalSummary) float64 { return r.CumulativeRevenue }},
	{MetricFinalHeadcount, func(r domain.RunTerminalSummary) float64 { return r.FinalHeadcount }},
	{MetricProductiveCapacityAdded, func(r domain.RunTerminalSummary) float64 { return r.ProductiveCapacityAddedTotal }},
	{MetricProductiveCapacityGrowth, func(r domain.RunTerminalSummary) float64 { return r.ProductiveCapacityGrowthRate }},
	{MetricRiskAdjustedEmployeeIncome, func(r domain.RunTerminalSummary) float64 { return r.RiskAdjustedEmployeeIncome }},
	{MetricSustainableDevelopmentValue, func(r domain.RunTerminalSummary) float64 { return r.SustainableDevelopmentValueProxy }},
}

// BuildPairedDeltas pairs candidate and reference records by run, behavior,
// market and horizon before aggregating the differences. A reference scenario
// is compared only with reference scenarios that precede it in references;
// non-reference scenarios are compared with every available reference.
func BuildPairedDeltas(runs []domain.RunTerminalSummary, references []string) []domain.PairedDeltaSummary {
	if len(references) == 0 {
		references = []string{v04config.SystemTraditionalCompany, v04config.SystemProfitSharing}
	}
	refs := uniqueStrings(references)
	sort.Slice(refs, func(i, j int) bool {
		leftRank, rightRank := referenceRank(refs[i]), referenceRank(refs[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return refs[i] < refs[j]
	})
	referencePosition := make(map[string]int, len(refs))
	for i, reference := range refs {
		referencePosition[reference] = i
	}

	byKey := make(map[pairedRunKey]domain.RunTerminalSummary, len(runs))
	orderedRuns := append([]domain.RunTerminalSummary(nil), runs...)
	sort.Slice(orderedRuns, func(i, j int) bool { return lessRunSummary(orderedRuns[i], orderedRuns[j]) })
	for _, run := range orderedRuns {
		key := pairedRunKey{run.Run, run.Scenario, run.BehaviorCase, marketCase(run.MarketCase), run.HorizonMonths}
		byKey[key] = run
	}

	deltas := make(map[pairedGroupKey][]float64)
	for _, candidate := range orderedRuns {
		candidatePosition, candidateIsReference := referencePosition[candidate.Scenario]
		for referencePositionIndex, referenceName := range refs {
			if referenceName == candidate.Scenario {
				continue
			}
			if candidateIsReference && referencePositionIndex > candidatePosition {
				continue
			}
			reference, ok := byKey[pairedRunKey{
				run:      candidate.Run,
				scenario: referenceName,
				behavior: candidate.BehaviorCase,
				market:   marketCase(candidate.MarketCase),
				horizon:  candidate.HorizonMonths,
			}]
			if !ok {
				continue
			}
			for _, metric := range pairedMetrics {
				key := pairedGroupKey{
					scenario: candidate.Scenario, behavior: candidate.BehaviorCase,
					reference: referenceName, referenceBehavior: reference.BehaviorCase,
					market: marketCase(candidate.MarketCase), horizon: candidate.HorizonMonths,
					metric: metric.name,
				}
				delta := finiteOrZero(metric.value(candidate)) - finiteOrZero(metric.value(reference))
				deltas[key] = append(deltas[key], delta)
			}
		}
	}

	keys := make([]pairedGroupKey, 0, len(deltas))
	for key := range deltas {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return lessPairedGroupKey(keys[i], keys[j]) })
	result := make([]domain.PairedDeltaSummary, 0, len(keys))
	for _, key := range keys {
		values := deltas[key]
		positive := 0
		negative := 0
		for _, value := range values {
			if value > 0 {
				positive++
			} else if value < 0 {
				negative++
			}
		}
		result = append(result, domain.PairedDeltaSummary{
			Scenario:              key.scenario,
			BehaviorCase:          key.behavior,
			ReferenceScenario:     key.reference,
			ReferenceBehaviorCase: key.referenceBehavior,
			MarketCase:            key.market,
			HorizonMonths:         key.horizon,
			Metric:                key.metric,
			Median:                Percentile(values, 0.50),
			P10:                   Percentile(values, 0.10),
			P90:                   Percentile(values, 0.90),
			ProbabilityPositive:   probability(positive, len(values)),
			ProbabilityNegative:   probability(negative, len(values)),
		})
	}
	return result
}

// Classify applies the section 16 risk gate. Sustainable development at least
// as high as the reference is development-dominant only when both risk
// probabilities are within tolerance of the reference.
func Classify(candidate, reference domain.ScenarioSummary, tolerance float64) string {
	if candidate.Scenario == reference.Scenario && candidate.BehaviorCase == reference.BehaviorCase &&
		marketCase(candidate.MarketCase) == marketCase(reference.MarketCase) && candidate.HorizonMonths == reference.HorizonMonths {
		return ClassificationReference
	}
	if !domain.Finite(tolerance) || tolerance < 0 {
		tolerance = 0
	}
	if candidate.BankruptcyProbability > reference.BankruptcyProbability+tolerance ||
		candidate.LiquidityDeficitProbability > reference.LiquidityDeficitProbability+tolerance {
		return ClassificationRiskConstrained
	}
	if candidate.SustainableDevelopmentValueMedian >= reference.SustainableDevelopmentValueMedian {
		return ClassificationDevelopmentDominant
	}
	return ClassificationDevelopmentTradeoff
}

// ApplyClassifications classifies summaries in place and also returns the
// slice for convenient composition. The paired sustainable-development median
// is authoritative; traditional_company is preferred as the reference and
// profit_sharing is used when no traditional pair exists.
func ApplyClassifications(summaries []domain.ScenarioSummary, deltas []domain.PairedDeltaSummary, tolerance float64) []domain.ScenarioSummary {
	type summaryKey struct {
		scenario, behavior, market string
		horizon                    int
	}
	byKey := make(map[summaryKey]domain.ScenarioSummary, len(summaries))
	for _, summary := range summaries {
		byKey[summaryKey{summary.Scenario, summary.BehaviorCase, marketCase(summary.MarketCase), summary.HorizonMonths}] = summary
	}
	type deltaKey struct {
		scenario, behavior, market string
		horizon                    int
	}
	preferred := make(map[deltaKey]domain.PairedDeltaSummary)
	for _, delta := range deltas {
		if delta.Metric != MetricSustainableDevelopmentValue {
			continue
		}
		key := deltaKey{delta.Scenario, delta.BehaviorCase, marketCase(delta.MarketCase), delta.HorizonMonths}
		current, exists := preferred[key]
		if !exists || preferredReference(delta.ReferenceScenario, current.ReferenceScenario) {
			preferred[key] = delta
		}
	}
	for i := range summaries {
		candidate := summaries[i]
		if candidate.Scenario == v04config.SystemTraditionalCompany {
			summaries[i].Classification = ClassificationReference
			continue
		}
		delta, ok := preferred[deltaKey{candidate.Scenario, candidate.BehaviorCase, marketCase(candidate.MarketCase), candidate.HorizonMonths}]
		if !ok {
			summaries[i].Classification = ClassificationUnclassified
			continue
		}
		reference, ok := byKey[summaryKey{delta.ReferenceScenario, delta.ReferenceBehaviorCase, marketCase(delta.MarketCase), delta.HorizonMonths}]
		if !ok {
			summaries[i].Classification = ClassificationUnclassified
			continue
		}
		if !domain.Finite(tolerance) || tolerance < 0 {
			tolerance = 0
		}
		if candidate.BankruptcyProbability > reference.BankruptcyProbability+tolerance ||
			candidate.LiquidityDeficitProbability > reference.LiquidityDeficitProbability+tolerance {
			summaries[i].Classification = ClassificationRiskConstrained
		} else if delta.Median >= 0 {
			summaries[i].Classification = ClassificationDevelopmentDominant
		} else {
			summaries[i].Classification = ClassificationDevelopmentTradeoff
		}
	}
	return summaries
}

// Percentile returns a linearly interpolated percentile over a sorted copy of
// values. p is clamped to [0,1]. Empty input has the neutral value zero.
func Percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := make([]float64, len(values))
	for i, value := range values {
		ordered[i] = finiteOrZero(value)
	}
	sort.Float64s(ordered)
	p = domain.Clamp(finiteOrZero(p), 0, 1)
	position := p * float64(len(ordered)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return ordered[lower]
	}
	weight := position - float64(lower)
	return ordered[lower] + weight*(ordered[upper]-ordered[lower])
}

func marketCase(value string) string {
	if value == "" {
		return domain.DefaultMarketCase
	}
	return value
}

func neutralGrowth(close, begin, epsilon float64) float64 {
	if begin <= epsilon || close < 0 {
		return 0
	}
	return finiteOrZero(close/begin - 1)
}

func neutralCAGR(close, begin, exponent, epsilon float64) float64 {
	if begin <= epsilon || close <= epsilon || exponent <= 0 {
		return 0
	}
	return finiteOrZero(math.Pow(close/begin, exponent) - 1)
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += finiteOrZero(value)
	}
	return sum / float64(len(values))
}

func populationStdDev(values []float64, average float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSquares := 0.0
	for _, value := range values {
		delta := finiteOrZero(value) - average
		sumSquares += delta * delta
	}
	return finiteOrZero(math.Sqrt(sumSquares / float64(len(values))))
}

func evaluateShockSurvival(path []domain.MonthlyResult, horizon int) (evaluable, survived bool) {
	const followUpMonths = 12
	if len(path) == 0 {
		return false, false
	}
	lastObservedMonth := path[len(path)-1].Month
	survived = true
	for _, shock := range path {
		if shock.Month > horizon {
			break
		}
		if !shock.ShockHappened || !shock.ActiveCompanyFlag || shock.Month+followUpMonths > lastObservedMonth {
			continue
		}
		evaluable = true
		followUpEnd := shock.Month + followUpMonths
		for _, observation := range path {
			if observation.Month < shock.Month || observation.Month > followUpEnd {
				continue
			}
			if observation.Risks.Bankruptcy || !observation.ActiveCompanyFlag {
				survived = false
				break
			}
		}
	}
	if !evaluable {
		return false, false
	}
	return true, survived
}

func probability(numerator, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func finiteOrZero(value float64) float64 {
	if !domain.Finite(value) {
		return 0
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func referenceRank(reference string) int {
	switch reference {
	case v04config.SystemTraditionalCompany:
		return 0
	case v04config.SystemProfitSharing:
		return 1
	default:
		return 2
	}
}

func preferredReference(candidate, current string) bool {
	candidateRank, currentRank := referenceRank(candidate), referenceRank(current)
	if candidateRank != currentRank {
		return candidateRank < currentRank
	}
	return candidate < current
}

func lessRunSummary(left, right domain.RunTerminalSummary) bool {
	if left.Scenario != right.Scenario {
		return left.Scenario < right.Scenario
	}
	if left.BehaviorCase != right.BehaviorCase {
		return left.BehaviorCase < right.BehaviorCase
	}
	if marketCase(left.MarketCase) != marketCase(right.MarketCase) {
		return marketCase(left.MarketCase) < marketCase(right.MarketCase)
	}
	if left.HorizonMonths != right.HorizonMonths {
		return left.HorizonMonths < right.HorizonMonths
	}
	return left.Run < right.Run
}

func lessPairedGroupKey(left, right pairedGroupKey) bool {
	if left.scenario != right.scenario {
		return left.scenario < right.scenario
	}
	if left.behavior != right.behavior {
		return left.behavior < right.behavior
	}
	if left.reference != right.reference {
		return left.reference < right.reference
	}
	if left.referenceBehavior != right.referenceBehavior {
		return left.referenceBehavior < right.referenceBehavior
	}
	if left.market != right.market {
		return left.market < right.market
	}
	if left.horizon != right.horizon {
		return left.horizon < right.horizon
	}
	return left.metric < right.metric
}
