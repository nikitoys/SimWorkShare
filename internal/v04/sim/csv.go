package sim

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"simworkshare/internal/v04/domain"
)

var monthlyCSVHeader = []string{
	"run", "month", "scenario", "system_type", "behavior_case", "market_case", "active_company_flag",
	"market_factor", "shock_happened", "effective_collection_rate", "headcount_begin",
	"voluntary_leavers", "layoffs", "hires", "headcount_end", "effective_employees",
	"productivity_uplift", "productivity_multiplier", "governance_hours",
	"governance_admin_equivalent_employees", "governance_cash_cost", "decision_delay_months",
	"decision_quality_multiplier", "fairness_index", "free_rider_penalty",
	"employee_risk_concentration_index", "market_demand", "labor_revenue_capacity",
	"productive_capacity_revenue_monthly", "revenue", "salary_cost", "fixed_costs",
	"variable_costs", "turnover_and_workforce_cost", "shock_cost",
	"operating_profit_before_allocation", "interest_expense",
	"profit_before_tax_before_distribution", "positive_result_base",
	"cash_collected_current", "cash_collected_from_ar", "mandatory_cash_payments",
	"cash_after_mandatory", "credit_draw_for_liquidity",
	"employee_cash_distribution_accrued", "employee_cash_distribution_paid",
	"member_capital_allocation", "member_capital_redemption_due",
	"reinvestment_cash_paid", "external_growth_capital_draw",
	"organizational_reserve_allocation", "profit_tax_accrual", "taxes_paid",
	"cash_total_close", "restricted_distribution_cash_close",
	"restricted_reserve_cash_close", "unrestricted_cash_close", "debt_balance_close",
	"required_cash_reserve", "reserve_breach_flag", "cash_gap_flag",
	"liquidity_deficit_flag", "bankruptcy_flag",
}

var summaryCSVHeader = []string{
	"scenario", "system_type", "behavior_case", "market_case", "horizon_months", "runs",
	"median_cumulative_revenue", "p10_cumulative_revenue", "p90_cumulative_revenue",
	"median_operating_profit", "median_net_profit", "median_productivity_per_employee",
	"turnover_rate_annual_average", "hiring_and_onboarding_costs_total",
	"average_employee_income_monthly", "risk_adjusted_employee_income",
	"employee_cash_distribution_total", "member_capital_accounts_total", "reinvestment_total_cash",
	"productive_capacity_growth_rate", "cash_end_total_median", "cash_end_unrestricted_p10",
	"min_unrestricted_cash_p10", "liquidity_deficit_probability", "bankruptcy_probability",
	"shock_survival_rate", "final_headcount_median", "revenue_cagr_median", "capacity_cagr_median",
	"employee_risk_concentration_index_average", "sustainable_development_value_proxy_median",
	"classification", "assumption_flags",
}

var pairedCSVHeader = []string{
	"scenario", "behavior_case", "reference_scenario", "reference_behavior_case", "market_case",
	"horizon_months", "metric", "median", "p10", "p90", "probability_positive", "probability_negative",
}

// WriteMonthlyCSV writes exactly the section 15.5 monthly columns. Rows are
// sorted independently of simulation execution order.
func WriteMonthlyCSV(writer io.Writer, results []domain.MonthlyResult) error {
	if writer == nil {
		return fmt.Errorf("monthly CSV writer is nil")
	}
	ordered := append([]domain.MonthlyResult(nil), results...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Run != ordered[j].Run {
			return ordered[i].Run < ordered[j].Run
		}
		if ordered[i].Scenario != ordered[j].Scenario {
			return ordered[i].Scenario < ordered[j].Scenario
		}
		if ordered[i].BehaviorCase != ordered[j].BehaviorCase {
			return ordered[i].BehaviorCase < ordered[j].BehaviorCase
		}
		if marketCase(ordered[i].MarketCase) != marketCase(ordered[j].MarketCase) {
			return marketCase(ordered[i].MarketCase) < marketCase(ordered[j].MarketCase)
		}
		return ordered[i].Month < ordered[j].Month
	})

	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(monthlyCSVHeader); err != nil {
		return fmt.Errorf("write monthly CSV header: %w", err)
	}
	for i, result := range ordered {
		row, err := monthlyCSVRow(result)
		if err != nil {
			return fmt.Errorf("monthly CSV row %d: %w", i, err)
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write monthly CSV row %d: %w", i, err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush monthly CSV: %w", err)
	}
	return nil
}

// WriteSummaryCSV writes scenario aggregates using stable columns and rows.
func WriteSummaryCSV(writer io.Writer, summaries []domain.ScenarioSummary) error {
	if writer == nil {
		return fmt.Errorf("summary CSV writer is nil")
	}
	ordered := append([]domain.ScenarioSummary(nil), summaries...)
	sort.Slice(ordered, func(i, j int) bool { return lessScenarioSummary(ordered[i], ordered[j]) })
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(summaryCSVHeader); err != nil {
		return fmt.Errorf("write summary CSV header: %w", err)
	}
	for i, summary := range ordered {
		row, err := summaryCSVRow(summary)
		if err != nil {
			return fmt.Errorf("summary CSV row %d: %w", i, err)
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write summary CSV row %d: %w", i, err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush summary CSV: %w", err)
	}
	return nil
}

// WritePairedCSV writes aggregated paired deltas using stable columns and rows.
func WritePairedCSV(writer io.Writer, deltas []domain.PairedDeltaSummary) error {
	if writer == nil {
		return fmt.Errorf("paired CSV writer is nil")
	}
	ordered := append([]domain.PairedDeltaSummary(nil), deltas...)
	sort.Slice(ordered, func(i, j int) bool { return lessPairedSummary(ordered[i], ordered[j]) })
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(pairedCSVHeader); err != nil {
		return fmt.Errorf("write paired CSV header: %w", err)
	}
	for i, delta := range ordered {
		row, err := pairedCSVRow(delta)
		if err != nil {
			return fmt.Errorf("paired CSV row %d: %w", i, err)
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write paired CSV row %d: %w", i, err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush paired CSV: %w", err)
	}
	return nil
}

// WritePairedDeltasCSV is a descriptive alias for WritePairedCSV.
func WritePairedDeltasCSV(writer io.Writer, deltas []domain.PairedDeltaSummary) error {
	return WritePairedCSV(writer, deltas)
}

func monthlyCSVRow(result domain.MonthlyResult) ([]string, error) {
	row := []string{
		strconv.Itoa(result.Run), strconv.Itoa(result.Month), result.Scenario, result.SystemType,
		result.BehaviorCase, marketCase(result.MarketCase), strconv.FormatBool(result.ActiveCompanyFlag),
	}
	values := []struct {
		name  string
		value float64
	}{
		{"market_factor", result.MarketFactor},
		{"effective_collection_rate", result.EffectiveCollectionRate},
		{"headcount_begin", result.HeadcountBegin},
		{"voluntary_leavers", result.VoluntaryLeavers},
		{"layoffs", result.Layoffs},
		{"hires", result.Hires},
		{"headcount_end", result.HeadcountEnd},
		{"effective_employees", result.EffectiveEmployees},
		{"productivity_uplift", result.ProductivityUplift},
		{"productivity_multiplier", result.ProductivityMultiplier},
		{"governance_hours", result.GovernanceHours},
		{"governance_admin_equivalent_employees", result.GovernanceAdminEquivalent},
		{"governance_cash_cost", result.GovernanceCashCost},
		{"decision_delay_months", result.DecisionDelayMonths},
		{"decision_quality_multiplier", result.DecisionQualityMultiplier},
		{"fairness_index", result.FairnessIndex},
		{"free_rider_penalty", result.FreeRiderPenalty},
		{"employee_risk_concentration_index", result.EmployeeRiskConcentration},
		{"market_demand", result.MarketDemand},
		{"labor_revenue_capacity", result.LaborRevenueCapacity},
		{"productive_capacity_revenue_monthly", result.ProductiveCapacityRevenueMonthly},
		{"revenue", result.Revenue},
		{"salary_cost", result.SalaryCost},
		{"fixed_costs", result.FixedCosts},
		{"variable_costs", result.VariableCosts},
		{"turnover_and_workforce_cost", result.TurnoverAndWorkforceCost},
		{"shock_cost", result.ShockCost},
		{"operating_profit_before_allocation", result.OperatingProfitBeforeAllocation},
		{"interest_expense", result.InterestExpense},
		{"profit_before_tax_before_distribution", result.ProfitBeforeTaxBeforeDistribution},
		{"positive_result_base", result.PositiveResultBase},
		{"cash_collected_current", result.CashCollectedCurrent},
		{"cash_collected_from_ar", result.CashCollectedFromAR},
		{"mandatory_cash_payments", result.MandatoryCashPayments},
		{"cash_after_mandatory", result.CashAfterMandatory},
		{"credit_draw_for_liquidity", result.CreditDrawForLiquidity},
		{"employee_cash_distribution_accrued", result.EmployeeCashDistributionAccrued},
		{"employee_cash_distribution_paid", result.EmployeeCashDistributionPaid},
		{"member_capital_allocation", result.MemberCapitalAllocation},
		{"member_capital_redemption_due", result.MemberCapitalRedemptionDue},
		{"reinvestment_cash_paid", result.ReinvestmentCashPaid},
		{"external_growth_capital_draw", result.ExternalGrowthCapitalDraw},
		{"organizational_reserve_allocation", result.OrganizationalReserveAllocation},
		{"profit_tax_accrual", result.ProfitTaxAccrual},
		{"taxes_paid", result.TaxesPaid},
		{"cash_total_close", result.CashTotalClose},
		{"restricted_distribution_cash_close", result.RestrictedDistributionClose},
		{"restricted_reserve_cash_close", result.RestrictedReserveClose},
		{"unrestricted_cash_close", result.UnrestrictedCashClose},
		{"debt_balance_close", result.DebtBalanceClose},
		{"required_cash_reserve", result.RequiredCashReserve},
	}
	// shock_happened appears after market_factor, so insert it before the
	// remaining numerical fields while retaining a single finite-value path.
	formatted := make([]string, len(values))
	for i, value := range values {
		text, err := formatCSVFloat(value.name, value.value)
		if err != nil {
			return nil, err
		}
		formatted[i] = text
	}
	row = append(row, formatted[0], strconv.FormatBool(result.ShockHappened))
	row = append(row, formatted[1:]...)
	row = append(row,
		strconv.FormatBool(result.Risks.ReserveBreach),
		strconv.FormatBool(result.Risks.CashGap),
		strconv.FormatBool(result.Risks.LiquidityDeficit),
		strconv.FormatBool(result.Risks.Bankruptcy),
	)
	if len(row) != len(monthlyCSVHeader) {
		return nil, fmt.Errorf("internal column mismatch: header=%d row=%d", len(monthlyCSVHeader), len(row))
	}
	return row, nil
}

func summaryCSVRow(summary domain.ScenarioSummary) ([]string, error) {
	row := []string{
		summary.Scenario, summary.SystemType, summary.BehaviorCase, marketCase(summary.MarketCase),
		strconv.Itoa(summary.HorizonMonths), strconv.Itoa(summary.Runs),
	}
	shockSurvivalRate := 0.0
	shockSurvivalAvailable := summary.ShockSurvivalRate != nil
	if shockSurvivalAvailable {
		shockSurvivalRate = *summary.ShockSurvivalRate
	}
	values := []struct {
		name      string
		value     float64
		available bool
	}{
		{"median_cumulative_revenue", summary.MedianCumulativeRevenue, true},
		{"p10_cumulative_revenue", summary.P10CumulativeRevenue, true},
		{"p90_cumulative_revenue", summary.P90CumulativeRevenue, true},
		{"median_operating_profit", summary.MedianOperatingProfit, true},
		{"median_net_profit", summary.MedianNetProfit, true},
		{"median_productivity_per_employee", summary.MedianProductivityPerEmployee, true},
		{"turnover_rate_annual_average", summary.TurnoverRateAnnualAverage, true},
		{"hiring_and_onboarding_costs_total", summary.HiringAndOnboardingCostsTotal, true},
		{"average_employee_income_monthly", summary.AverageEmployeeIncomeMonthly, true},
		{"risk_adjusted_employee_income", summary.RiskAdjustedEmployeeIncome, true},
		{"employee_cash_distribution_total", summary.EmployeeCashDistributionTotal, true},
		{"member_capital_accounts_total", summary.MemberCapitalAccountsTotal, true},
		{"reinvestment_total_cash", summary.ReinvestmentTotalCash, true},
		{"productive_capacity_growth_rate", summary.ProductiveCapacityGrowthRate, true},
		{"cash_end_total_median", summary.CashEndTotalMedian, true},
		{"cash_end_unrestricted_p10", summary.CashEndUnrestrictedP10, true},
		{"min_unrestricted_cash_p10", summary.MinimumUnrestrictedCashP10, true},
		{"liquidity_deficit_probability", summary.LiquidityDeficitProbability, true},
		{"bankruptcy_probability", summary.BankruptcyProbability, true},
		{"shock_survival_rate", shockSurvivalRate, shockSurvivalAvailable},
		{"final_headcount_median", summary.FinalHeadcountMedian, true},
		{"revenue_cagr_median", summary.RevenueCAGRMedian, true},
		{"capacity_cagr_median", summary.CapacityCAGRMedian, true},
		{"employee_risk_concentration_index_average", summary.EmployeeRiskConcentrationIndexAverage, true},
		{"sustainable_development_value_proxy_median", summary.SustainableDevelopmentValueMedian, true},
	}
	for _, value := range values {
		if !value.available {
			row = append(row, "")
			continue
		}
		text, err := formatCSVFloat(value.name, value.value)
		if err != nil {
			return nil, err
		}
		row = append(row, text)
	}
	flags := append([]string(nil), summary.AssumptionFlags...)
	sort.Strings(flags)
	row = append(row, summary.Classification, strings.Join(flags, ";"))
	if len(row) != len(summaryCSVHeader) {
		return nil, fmt.Errorf("internal column mismatch: header=%d row=%d", len(summaryCSVHeader), len(row))
	}
	return row, nil
}

func pairedCSVRow(delta domain.PairedDeltaSummary) ([]string, error) {
	row := []string{
		delta.Scenario, delta.BehaviorCase, delta.ReferenceScenario, delta.ReferenceBehaviorCase,
		marketCase(delta.MarketCase), strconv.Itoa(delta.HorizonMonths), delta.Metric,
	}
	values := []struct {
		name  string
		value float64
	}{
		{"median", delta.Median},
		{"p10", delta.P10},
		{"p90", delta.P90},
		{"probability_positive", delta.ProbabilityPositive},
		{"probability_negative", delta.ProbabilityNegative},
	}
	for _, value := range values {
		text, err := formatCSVFloat(value.name, value.value)
		if err != nil {
			return nil, err
		}
		row = append(row, text)
	}
	return row, nil
}

func formatCSVFloat(name string, value float64) (string, error) {
	if !domain.Finite(value) {
		return "", fmt.Errorf("%s must be finite", name)
	}
	if value == 0 {
		value = 0 // normalize negative zero
	}
	return strconv.FormatFloat(value, 'g', -1, 64), nil
}

func lessScenarioSummary(left, right domain.ScenarioSummary) bool {
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
	return left.SystemType < right.SystemType
}

func lessPairedSummary(left, right domain.PairedDeltaSummary) bool {
	if left.Scenario != right.Scenario {
		return left.Scenario < right.Scenario
	}
	if left.BehaviorCase != right.BehaviorCase {
		return left.BehaviorCase < right.BehaviorCase
	}
	if left.ReferenceScenario != right.ReferenceScenario {
		return left.ReferenceScenario < right.ReferenceScenario
	}
	if left.ReferenceBehaviorCase != right.ReferenceBehaviorCase {
		return left.ReferenceBehaviorCase < right.ReferenceBehaviorCase
	}
	if marketCase(left.MarketCase) != marketCase(right.MarketCase) {
		return marketCase(left.MarketCase) < marketCase(right.MarketCase)
	}
	if left.HorizonMonths != right.HorizonMonths {
		return left.HorizonMonths < right.HorizonMonths
	}
	return left.Metric < right.Metric
}
