package sim

import (
	"math"
	"reflect"
	"testing"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func TestBuildRunTerminalSummaryExactMetricsAndNeutralShortCAGR(t *testing.T) {
	cfg := metricTestConfig()
	months := []domain.MonthlyResult{
		{
			Run: 7, Month: 1, Scenario: "cooperative", SystemType: "worker_cooperative", BehaviorCase: "case",
			ActiveCompanyFlag: true, Revenue: 100, OperatingProfitBeforeAllocation: 20,
			NetProfitAfterTaxAndEmployeeDistribution: 10, PaidEmployees: 10, TurnoverRateAnnual: .12,
			VoluntaryLeavers: 1, Layoffs: 2, Hires: 3, HiringCost: 7,
			SalaryCost: 900, MandatoryCashPayments: 1000, EmployeeCashDistributionPaid: 100,
			MemberCapitalClose: 20, ReinvestmentCashPaid: 8, ExternalGrowthCapitalDraw: 3,
			RawAllocations: domain.AllocationAmounts{Reinvestment: 20}, ActualAllocations: domain.AllocationAmounts{Reinvestment: 10},
			CapacityAddedByInvestment: 5, ProductiveCapacityBegin: 100, ProductiveCapacityClose: 105,
			CashTotalClose: 1000, UnrestrictedCashClose: 800, HeadcountEnd: 10,
			EmployeeRiskConcentration: .2, ShockHappened: true,
			Risks: domain.RiskFlags{ReserveBreach: true, LiquidityDeficit: true},
		},
		{
			Run: 7, Month: 2, Scenario: "cooperative", SystemType: "worker_cooperative", BehaviorCase: "case",
			ActiveCompanyFlag: true, Revenue: 200, OperatingProfitBeforeAllocation: 30,
			NetProfitAfterTaxAndEmployeeDistribution: 15, PaidEmployees: 20, TurnoverRateAnnual: .24,
			VoluntaryLeavers: 2, Layoffs: 3, Hires: 4, HiringCost: 11,
			SalaryCost: 3800, MandatoryCashPayments: 4000, EmployeeCashDistributionPaid: 200,
			MemberCapitalClose: 30, ReinvestmentCashPaid: 12, ExternalGrowthCapitalDraw: 4,
			RawAllocations: domain.AllocationAmounts{Reinvestment: 20}, ActualAllocations: domain.AllocationAmounts{Reinvestment: 20},
			CapacityAddedByInvestment: 10, ProductiveCapacityBegin: 105, ProductiveCapacityClose: 120,
			CashTotalClose: 900, UnrestrictedCashClose: 700, DebtBalanceClose: 20,
			UnpaidMandatoryObligations: 1, UnpaidMandatoryObligationsToDate: 8,
			MemberCapitalRedemptionDue: 999, MemberCapitalRedemptionPayableClose: 4,
			HeadcountEnd: 18, EmployeeRiskConcentration: .4,
		},
	}

	summary, err := BuildRunTerminalSummary(cfg, []domain.MonthlyResult{months[1], months[0]}, 2)
	if err != nil {
		t.Fatalf("BuildRunTerminalSummary() error = %v", err)
	}
	want := map[string]struct{ got, want float64 }{
		"cumulative revenue":         {summary.CumulativeRevenue, 300},
		"cumulative operating":       {summary.CumulativeOperatingProfit, 50},
		"cumulative net":             {summary.CumulativeNetProfit, 25},
		"productivity":               {summary.ProductivityPerEmployee, 10},
		"turnover":                   {summary.TurnoverRateAnnualAverage, .18},
		"leavers":                    {summary.VoluntaryLeaversTotal, 3},
		"layoffs":                    {summary.LayoffsTotal, 5},
		"hires":                      {summary.HiresTotal, 7},
		"hiring costs":               {summary.HiringAndOnboardingCostsTotal, 18},
		"average income":             {summary.AverageEmployeeIncomeMonthly, 5000.0 / 30.0},
		"income population stddev":   {summary.EmployeeIncomeVolatility, 50},
		"risk adjusted income":       {summary.RiskAdjustedEmployeeIncome, 5000.0/30.0 - 25},
		"cash distributions":         {summary.EmployeeCashDistributionTotal, 300},
		"member capital":             {summary.MemberCapitalAccountsTotal, 30},
		"reinvestment cash":          {summary.ReinvestmentTotalCash, 20},
		"external growth capital":    {summary.ExternalGrowthCapitalTotal, 7},
		"actual reinvestment":        {summary.ActualReinvestmentTotal, 37},
		"underfunding":               {summary.ReinvestmentUnderfundingRate, .075},
		"capacity added":             {summary.ProductiveCapacityAddedTotal, 15},
		"capacity growth":            {summary.ProductiveCapacityGrowthRate, .2},
		"cash total":                 {summary.CashEndTotal, 900},
		"cash unrestricted":          {summary.CashEndUnrestricted, 700},
		"minimum unrestricted":       {summary.MinimumUnrestrictedCash, 700},
		"debt":                       {summary.DebtBalance, 20},
		"cumulative unpaid":          {summary.UnpaidObligations, 8},
		"redemption payable closing": {summary.MemberCapitalRedemptionDue, 4},
		"final headcount":            {summary.FinalHeadcount, 18},
		"short revenue CAGR":         {summary.RevenueCAGR, 0},
		"short capacity CAGR":        {summary.CapacityCAGR, 0},
		"risk concentration":         {summary.EmployeeRiskConcentrationIndexAverage, .3},
		"sustainable value":          {summary.SustainableDevelopmentValueProxy, 728},
	}
	for name, test := range want {
		if math.Abs(test.got-test.want) > 1e-9 {
			t.Errorf("%s = %.12g, want %.12g", name, test.got, test.want)
		}
	}
	if summary.MarketCase != domain.DefaultMarketCase {
		t.Errorf("MarketCase = %q, want %q", summary.MarketCase, domain.DefaultMarketCase)
	}
	if !summary.HadLiquidityDeficit || summary.HadBankruptcy || !summary.HadShock ||
		summary.ShockSurvivalEvaluable || summary.ShockSurvived {
		t.Errorf("unexpected run risks: %+v", summary)
	}
	if !summary.RiskFlagsEver.ReserveBreach || !summary.RiskFlagsEver.LiquidityDeficit {
		t.Errorf("RiskFlagsEver = %+v", summary.RiskFlagsEver)
	}
}

func TestBuildRunTerminalSummaryLongCAGRs(t *testing.T) {
	cfg := metricTestConfig()
	months := make([]domain.MonthlyResult, 24)
	for i := range months {
		revenue := 100.0
		if i >= 12 {
			revenue = 121
		}
		months[i] = domain.MonthlyResult{
			Run: 1, Month: i + 1, Scenario: "s", SystemType: "x", BehaviorCase: "b",
			ActiveCompanyFlag: true, Revenue: revenue, ProductiveCapacityBegin: 100,
			ProductiveCapacityClose: 100,
		}
	}
	months[len(months)-1].ProductiveCapacityClose = 121
	summary, err := BuildRunTerminalSummary(cfg, months, 24)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(summary.RevenueCAGR-.21) > 1e-12 {
		t.Errorf("RevenueCAGR = %g, want .21", summary.RevenueCAGR)
	}
	if math.Abs(summary.CapacityCAGR-.1) > 1e-12 {
		t.Errorf("CapacityCAGR = %g, want .1", summary.CapacityCAGR)
	}
}

func TestBuildRunTerminalSummaryZeroHeadcountIsFinite(t *testing.T) {
	cfg := metricTestConfig()
	month := domain.MonthlyResult{
		Run: 1, Month: 1, Scenario: "s", SystemType: "x", BehaviorCase: "b", ActiveCompanyFlag: true,
		Revenue: 100, SalaryCost: 100, MandatoryCashPayments: 100,
	}
	summary, err := BuildRunTerminalSummary(cfg, []domain.MonthlyResult{month}, 1)
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]float64{
		"productivity":  summary.ProductivityPerEmployee,
		"income":        summary.AverageEmployeeIncomeMonthly,
		"volatility":    summary.EmployeeIncomeVolatility,
		"revenue CAGR":  summary.RevenueCAGR,
		"capacity CAGR": summary.CapacityCAGR,
	} {
		if value != 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			t.Errorf("%s = %v, want finite zero", name, value)
		}
	}
}

func TestBuildRunTerminalSummariesUsesSortedConfiguredHorizons(t *testing.T) {
	cfg := metricTestConfig()
	cfg.Simulation.HorizonsMonths = []int{2, 1, 2}
	months := []domain.MonthlyResult{
		{Run: 1, Month: 1, Scenario: "s", BehaviorCase: "b", ActiveCompanyFlag: true},
		{Run: 1, Month: 2, Scenario: "s", BehaviorCase: "b", ActiveCompanyFlag: true},
	}
	summaries, err := BuildRunTerminalSummaries(cfg, months)
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{summaries[0].HorizonMonths, summaries[1].HorizonMonths}; !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("horizons = %v", got)
	}
	if _, err := BuildRunTerminalSummary(cfg, months[:1], 2); err == nil {
		t.Fatal("incomplete horizon accepted")
	}
}

func TestPercentileLinearInterpolationAndAggregationProbabilities(t *testing.T) {
	values := []float64{40, 0, 30, 10, 20}
	if got := Percentile(values, .1); got != 4 {
		t.Errorf("P10 = %g, want 4", got)
	}
	if got := Percentile(values, .5); got != 20 {
		t.Errorf("median = %g, want 20", got)
	}
	if got := Percentile(values, .9); got != 36 {
		t.Errorf("P90 = %g, want 36", got)
	}
	if !reflect.DeepEqual(values, []float64{40, 0, 30, 10, 20}) {
		t.Fatal("Percentile mutated its input")
	}

	runs := make([]domain.RunTerminalSummary, 5)
	for i, value := range []float64{40, 0, 30, 10, 20} {
		runs[i] = domain.RunTerminalSummary{
			Run: i + 1, Scenario: "s", SystemType: "x", BehaviorCase: "b", HorizonMonths: 12,
			CumulativeRevenue: value, CumulativeOperatingProfit: value, CumulativeNetProfit: value,
			SustainableDevelopmentValueProxy: value,
		}
	}
	runs[0].HadLiquidityDeficit = true
	runs[1].HadLiquidityDeficit = true
	runs[0].HadBankruptcy = true
	for i := 0; i < 3; i++ {
		runs[i].HadShock = true
		runs[i].ShockSurvivalEvaluable = true
		runs[i].ShockSurvived = i != 0
	}
	summaries := AggregateScenarioSummaries(runs)
	if len(summaries) != 1 {
		t.Fatalf("len summaries = %d", len(summaries))
	}
	summary := summaries[0]
	if summary.MedianCumulativeRevenue != 20 || summary.P10CumulativeRevenue != 4 || summary.P90CumulativeRevenue != 36 {
		t.Errorf("revenue percentiles = %g/%g/%g", summary.P10CumulativeRevenue, summary.MedianCumulativeRevenue, summary.P90CumulativeRevenue)
	}
	if summary.ShockSurvivalRate == nil {
		t.Fatal("shock survival rate is unexpectedly unavailable")
	}
	if summary.LiquidityDeficitProbability != .4 || summary.BankruptcyProbability != .2 || math.Abs(*summary.ShockSurvivalRate-2.0/3.0) > 1e-12 {
		t.Errorf("probabilities = liquidity %g bankruptcy %g shock %g", summary.LiquidityDeficitProbability, summary.BankruptcyProbability, *summary.ShockSurvivalRate)
	}
}

func TestShockSurvivalUsesCompleteTwelveMonthFollowUp(t *testing.T) {
	cfg := metricTestConfig()
	months := make([]domain.MonthlyResult, 15)
	for index := range months {
		months[index] = domain.MonthlyResult{
			Run: 1, Month: index + 1, Scenario: "s", SystemType: "x", BehaviorCase: "b",
			ActiveCompanyFlag: true,
		}
	}
	months[1].ShockHappened = true
	months[9].Risks.Bankruptcy = true

	summary, err := BuildRunTerminalSummary(cfg, months, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !summary.ShockSurvivalEvaluable || summary.ShockSurvived {
		t.Fatalf("shock follow-up = evaluable %v survived %v, want true/false",
			summary.ShockSurvivalEvaluable, summary.ShockSurvived)
	}
}

func TestShockSurvivalIsUnavailableWithoutCompleteFollowUp(t *testing.T) {
	runs := []domain.RunTerminalSummary{{
		Run: 1, Scenario: "s", SystemType: "x", BehaviorCase: "b", HorizonMonths: 12,
		HadShock: true, ShockSurvivalEvaluable: false,
	}}
	summaries := AggregateScenarioSummaries(runs)
	if len(summaries) != 1 || summaries[0].ShockSurvivalRate != nil {
		t.Fatalf("shock survival rate = %v, want unavailable", summaries[0].ShockSurvivalRate)
	}
}

func TestBuildPairedDeltasMatchesRunBehaviorAndBothReferences(t *testing.T) {
	runs := []domain.RunTerminalSummary{
		pairedRun(2, "worker_cooperative", "b", 180, 1800),
		pairedRun(1, "traditional_company", "b", 100, 1000),
		pairedRun(1, "profit_sharing", "b", 120, 1200),
		pairedRun(2, "traditional_company", "b", 200, 2000),
		pairedRun(1, "worker_cooperative", "b", 150, 1500),
		pairedRun(2, "profit_sharing", "b", 250, 2500),
		pairedRun(1, "traditional_company", "other", 999, 999),
	}
	references := []string{"traditional_company", "profit_sharing"}
	got := BuildPairedDeltas(runs, references)
	reversed := append([]domain.RunTerminalSummary(nil), runs...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	if other := BuildPairedDeltas(reversed, references); !reflect.DeepEqual(got, other) {
		t.Fatal("paired output depends on input order")
	}

	traditional := findPairedDelta(t, got, "worker_cooperative", "traditional_company", MetricSustainableDevelopmentValue)
	if traditional.Median != 15 || traditional.P10 != -13 || traditional.P90 != 43 ||
		traditional.ProbabilityPositive != .5 || traditional.ProbabilityNegative != .5 {
		t.Errorf("traditional paired delta = %+v", traditional)
	}
	profitSharing := findPairedDelta(t, got, "worker_cooperative", "profit_sharing", MetricSustainableDevelopmentValue)
	if profitSharing.Median != -20 || profitSharing.P10 != -60 || profitSharing.P90 != 20 {
		t.Errorf("profit-sharing paired delta = %+v", profitSharing)
	}
	if len(got) != len(pairedMetrics)*3 { // profit-vs-traditional plus two cooperative references
		t.Errorf("paired rows = %d, want %d", len(got), len(pairedMetrics)*3)
	}
}

func TestApplyClassificationsUsesPairedMedianAndRiskGate(t *testing.T) {
	reference := domain.ScenarioSummary{
		Scenario: "traditional_company", BehaviorCase: "b", MarketCase: domain.DefaultMarketCase, HorizonMonths: 60,
	}
	candidate := domain.ScenarioSummary{
		Scenario: "worker_cooperative", BehaviorCase: "b", MarketCase: domain.DefaultMarketCase, HorizonMonths: 60,
		LiquidityDeficitProbability: .02,
	}
	deltas := []domain.PairedDeltaSummary{{
		Scenario: candidate.Scenario, BehaviorCase: "b", ReferenceScenario: reference.Scenario,
		ReferenceBehaviorCase: "b", MarketCase: domain.DefaultMarketCase, HorizonMonths: 60,
		Metric: MetricSustainableDevelopmentValue, Median: 1,
	}}
	summaries := ApplyClassifications([]domain.ScenarioSummary{candidate, reference}, deltas, .01)
	if summaries[0].Classification != ClassificationRiskConstrained {
		t.Fatalf("classification = %q", summaries[0].Classification)
	}
	candidate.LiquidityDeficitProbability = .01
	summaries = ApplyClassifications([]domain.ScenarioSummary{candidate, reference}, deltas, .01)
	if summaries[0].Classification != ClassificationDevelopmentDominant {
		t.Fatalf("classification at tolerance boundary = %q", summaries[0].Classification)
	}
	if summaries[1].Classification != ClassificationReference {
		t.Fatalf("reference classification = %q", summaries[1].Classification)
	}
	deltas[0].Median = -1
	summaries = ApplyClassifications([]domain.ScenarioSummary{candidate, reference}, deltas, .01)
	if summaries[0].Classification != ClassificationDevelopmentTradeoff {
		t.Fatalf("negative development classification = %q", summaries[0].Classification)
	}
}

func metricTestConfig() v04config.Config {
	return v04config.Config{
		Simulation:       v04config.Simulation{Epsilon: 1e-9, HorizonsMonths: []int{2}},
		CompanyEconomics: v04config.CompanyEconomics{CapacityRevenueCreatedPerCurrencyInvested: 2},
		Analysis:         v04config.Analysis{VolatilityPenaltyLambda: .5},
	}
}

func pairedRun(run int, scenario, behavior string, sustainable, revenue float64) domain.RunTerminalSummary {
	return domain.RunTerminalSummary{
		Run: run, Scenario: scenario, BehaviorCase: behavior, MarketCase: domain.DefaultMarketCase, HorizonMonths: 60,
		SustainableDevelopmentValueProxy: sustainable,
		CumulativeRevenue:                revenue,
		CumulativeOperatingProfit:        revenue / 2,
		CumulativeNetProfit:              revenue / 3,
		CashEndTotal:                     revenue / 4,
		CashEndUnrestricted:              revenue / 5,
		FinalHeadcount:                   revenue / 100,
		ProductiveCapacityGrowthRate:     revenue / 10000,
		ProductiveCapacityAddedTotal:     revenue / 20,
		CapacityCAGR:                     revenue / 100000,
		AverageEmployeeIncomeMonthly:     revenue / 50,
		RiskAdjustedEmployeeIncome:       revenue / 60,
	}
}

func findPairedDelta(t *testing.T, values []domain.PairedDeltaSummary, scenario, reference, metric string) domain.PairedDeltaSummary {
	t.Helper()
	for _, value := range values {
		if value.Scenario == scenario && value.ReferenceScenario == reference && value.Metric == metric {
			return value
		}
	}
	t.Fatalf("paired delta not found: scenario=%s reference=%s metric=%s", scenario, reference, metric)
	return domain.PairedDeltaSummary{}
}
