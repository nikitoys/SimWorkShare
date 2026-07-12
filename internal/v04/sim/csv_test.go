package sim

import (
	"bytes"
	"encoding/csv"
	"math"
	"reflect"
	"strings"
	"testing"

	"simworkshare/internal/v04/domain"
)

func TestWriteMonthlyCSVExactSection155HeaderValuesAndSorting(t *testing.T) {
	expectedHeader := strings.Split("run,month,scenario,system_type,behavior_case,market_case,active_company_flag,market_factor,shock_happened,effective_collection_rate,headcount_begin,voluntary_leavers,layoffs,hires,headcount_end,effective_employees,productivity_uplift,productivity_multiplier,governance_hours,governance_admin_equivalent_employees,governance_cash_cost,decision_delay_months,decision_quality_multiplier,fairness_index,free_rider_penalty,employee_risk_concentration_index,market_demand,labor_revenue_capacity,productive_capacity_revenue_monthly,revenue,salary_cost,fixed_costs,variable_costs,turnover_and_workforce_cost,shock_cost,operating_profit_before_allocation,interest_expense,profit_before_tax_before_distribution,positive_result_base,cash_collected_current,cash_collected_from_ar,mandatory_cash_payments,cash_after_mandatory,credit_draw_for_liquidity,employee_cash_distribution_accrued,employee_cash_distribution_paid,member_capital_allocation,member_capital_redemption_due,reinvestment_cash_paid,external_growth_capital_draw,organizational_reserve_allocation,profit_tax_accrual,taxes_paid,cash_total_close,restricted_distribution_cash_close,restricted_reserve_cash_close,unrestricted_cash_close,debt_balance_close,required_cash_reserve,reserve_breach_flag,cash_gap_flag,liquidity_deficit_flag,bankruptcy_flag", ",")
	first := domain.MonthlyResult{
		Run: 1, Month: 2, Scenario: "a,b", SystemType: "x", BehaviorCase: "case", ActiveCompanyFlag: true,
		MarketFactor: 1.1, ShockHappened: true, EffectiveCollectionRate: .9,
		HeadcountBegin: 10, VoluntaryLeavers: 1, Layoffs: 2, Hires: 3, HeadcountEnd: 10,
		EffectiveEmployees: 9, ProductivityUplift: .1, ProductivityMultiplier: 1.1,
		GovernanceHours: 4, GovernanceAdminEquivalent: .5, GovernanceCashCost: 6,
		DecisionDelayMonths: .2, DecisionQualityMultiplier: 1.03, FairnessIndex: .7,
		FreeRiderPenalty: .02, EmployeeRiskConcentration: .3, MarketDemand: 100,
		LaborRevenueCapacity: 90, ProductiveCapacityRevenueMonthly: 80, Revenue: 70,
		SalaryCost: 60, FixedCosts: 50, VariableCosts: 40, TurnoverAndWorkforceCost: 30,
		ShockCost: 20, OperatingProfitBeforeAllocation: 10, InterestExpense: 9,
		ProfitBeforeTaxBeforeDistribution: 8, PositiveResultBase: 7, CashCollectedCurrent: 6,
		CashCollectedFromAR: 5, MandatoryCashPayments: 4, CashAfterMandatory: 3,
		CreditDrawForLiquidity: 2, EmployeeCashDistributionAccrued: 1.9,
		EmployeeCashDistributionPaid: 1.8, MemberCapitalAllocation: 1.7,
		MemberCapitalRedemptionDue: 1.6, ReinvestmentCashPaid: 1.5,
		ExternalGrowthCapitalDraw: 1.4, OrganizationalReserveAllocation: 1.3,
		ProfitTaxAccrual: 1.2, TaxesPaid: 1.1, CashTotalClose: 1000,
		RestrictedDistributionClose: 100, RestrictedReserveClose: 90,
		UnrestrictedCashClose: 810, DebtBalanceClose: 80, RequiredCashReserve: 70,
		Risks: domain.RiskFlags{ReserveBreach: true, CashGap: false, LiquidityDeficit: true, Bankruptcy: false},
	}
	second := domain.MonthlyResult{Run: 2, Month: 1, Scenario: "z", BehaviorCase: "case"}

	var output bytes.Buffer
	if err := WriteMonthlyCSV(&output, []domain.MonthlyResult{second, first}); err != nil {
		t.Fatalf("WriteMonthlyCSV() error = %v", err)
	}
	records := readCSV(t, output.String())
	if len(records) != 3 {
		t.Fatalf("CSV records = %d", len(records))
	}
	if !reflect.DeepEqual(records[0], expectedHeader) {
		t.Fatalf("header mismatch\ngot  %v\nwant %v", records[0], expectedHeader)
	}
	if len(records[1]) != len(expectedHeader) {
		t.Fatalf("row columns = %d, header columns = %d", len(records[1]), len(expectedHeader))
	}
	if records[1][csvColumn(t, expectedHeader, "run")] != "1" || records[1][csvColumn(t, expectedHeader, "scenario")] != "a,b" {
		t.Errorf("rows not sorted or quoted correctly: %v", records[1][:7])
	}
	checks := map[string]string{
		"market_case":                           domain.DefaultMarketCase,
		"market_factor":                         "1.1",
		"shock_happened":                        "true",
		"effective_collection_rate":             "0.9",
		"governance_admin_equivalent_employees": "0.5",
		"member_capital_redemption_due":         "1.6",
		"restricted_reserve_cash_close":         "90",
		"reserve_breach_flag":                   "true",
		"cash_gap_flag":                         "false",
		"liquidity_deficit_flag":                "true",
		"bankruptcy_flag":                       "false",
	}
	for name, want := range checks {
		if got := records[1][csvColumn(t, expectedHeader, name)]; got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestWriteMonthlyCSVRejectsNonFiniteValue(t *testing.T) {
	var output bytes.Buffer
	err := WriteMonthlyCSV(&output, []domain.MonthlyResult{{MarketFactor: math.NaN()}})
	if err == nil || !strings.Contains(err.Error(), "market_factor must be finite") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(output.String(), "NaN") {
		t.Fatalf("non-finite value leaked into CSV: %q", output.String())
	}
}

func TestWriteSummaryCSVStableRowsFlagsAndEncoding(t *testing.T) {
	shockRate := 0.75
	summaries := []domain.ScenarioSummary{
		{Scenario: "z", BehaviorCase: "b", HorizonMonths: 60},
		{
			Scenario: "a,quoted", SystemType: "x", BehaviorCase: "b", HorizonMonths: 12, Runs: 2,
			MedianCumulativeRevenue: 100, P10CumulativeRevenue: 90, P90CumulativeRevenue: 110,
			SustainableDevelopmentValueMedian: 77, Classification: ClassificationDevelopmentDominant,
			ShockSurvivalRate: &shockRate,
			AssumptionFlags:   []string{"z_flag", "a_flag"},
		},
	}
	var output bytes.Buffer
	if err := WriteSummaryCSV(&output, summaries); err != nil {
		t.Fatal(err)
	}
	records := readCSV(t, output.String())
	if len(records) != 3 || len(records[0]) != len(summaryCSVHeader) {
		t.Fatalf("summary CSV dimensions = rows %d columns %d", len(records), len(records[0]))
	}
	if records[1][csvColumn(t, records[0], "scenario")] != "a,quoted" {
		t.Fatalf("summary rows not sorted/encoded: %v", records[1])
	}
	if got := records[1][csvColumn(t, records[0], "assumption_flags")]; got != "a_flag;z_flag" {
		t.Errorf("flags = %q", got)
	}
	if got := records[1][csvColumn(t, records[0], "sustainable_development_value_proxy_median")]; got != "77" {
		t.Errorf("sustainable metric = %q", got)
	}
	shockColumn := csvColumn(t, records[0], "shock_survival_rate")
	if got := records[1][shockColumn]; got != "0.75" {
		t.Errorf("available shock survival rate = %q", got)
	}
	if got := records[2][shockColumn]; got != "" {
		t.Errorf("unavailable shock survival rate = %q, want empty", got)
	}
}

func TestWritePairedCSVStableHeaderRows(t *testing.T) {
	deltas := []domain.PairedDeltaSummary{
		{Scenario: "z", BehaviorCase: "b", ReferenceScenario: "traditional_company", HorizonMonths: 60, Metric: "z"},
		{
			Scenario: "a", BehaviorCase: "b", ReferenceScenario: "traditional_company",
			ReferenceBehaviorCase: "b", HorizonMonths: 12, Metric: "cash", Median: 1,
			P10: -2, P90: 3, ProbabilityPositive: .6, ProbabilityNegative: .4,
		},
	}
	var output bytes.Buffer
	if err := WritePairedDeltasCSV(&output, deltas); err != nil {
		t.Fatal(err)
	}
	records := readCSV(t, output.String())
	if !reflect.DeepEqual(records[0], pairedCSVHeader) {
		t.Fatalf("paired header = %v", records[0])
	}
	if got := records[1][csvColumn(t, records[0], "scenario")]; got != "a" {
		t.Errorf("first scenario = %q", got)
	}
	if got := records[1][csvColumn(t, records[0], "market_case")]; got != domain.DefaultMarketCase {
		t.Errorf("market case = %q", got)
	}
	if got := records[1][csvColumn(t, records[0], "probability_positive")]; got != "0.6" {
		t.Errorf("probability positive = %q", got)
	}
}

func readCSV(t *testing.T, value string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(value)).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v\n%s", err, value)
	}
	return records
}

func csvColumn(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, value := range header {
		if value == name {
			return i
		}
	}
	t.Fatalf("column %q not found in %v", name, header)
	return -1
}
