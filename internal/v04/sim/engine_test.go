package sim

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func TestBaselineStaticTraditionalSanity(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	cfg.Market.MarketGrowthMonthly = 0
	cfg.Market.ShockProbabilityMonthly = 0
	cfg.Workforce.BaseTurnoverRateAnnual = 0
	cfg.Workforce.MinTurnoverRateAnnual = 0
	cfg.Workforce.MaxTurnoverRateAnnual = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	cfg.CompanyEconomics.CapacityDepreciationRateMonthly = 0

	result, err := RunDeterministicScenario(cfg, "traditional_company", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	month := result.MonthlyResults[0]
	assertCloseV04(t, "revenue", month.Revenue, 11_500_000)
	assertCloseV04(t, "salary", month.SalaryCost, 5_000_000)
	assertCloseV04(t, "variable", month.VariableCosts, 2_875_000)
	assertCloseV04(t, "fixed", month.FixedCosts, 2_000_000)
	assertCloseV04(t, "operating profit", month.OperatingProfitBeforeAllocation, 1_625_000)
}

func TestDefaultDeterministicRunsFourScenariosAndNoEffectIsNeutral(t *testing.T) {
	cfg := deterministicConfig(t, 6)
	result, err := Run(cfg, RunOptions{
		BehaviorCaseNames:   []string{"no_effect"},
		StoreMonthlyResults: true,
		StoreRunSummaries:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MonthlyResults) != 4*cfg.Simulation.Months {
		t.Fatalf("monthly results = %d, want %d", len(result.MonthlyResults), 4*cfg.Simulation.Months)
	}
	scenarios := make(map[string]bool)
	for _, month := range result.MonthlyResults {
		scenarios[month.Scenario] = true
		if month.MarketCase != domain.DefaultMarketCase {
			t.Fatalf("market case = %q", month.MarketCase)
		}
		if math.Abs(month.MotivationUpliftRaw) > cfg.Simulation.Epsilon || math.Abs(month.BehavioralTurnoverDeltaAnnual) > cfg.Simulation.Epsilon {
			t.Fatalf("no_effect created behavior effect in %s month %d", month.Scenario, month.Month)
		}
	}
	for _, name := range []string{"traditional_company", "profit_sharing", "employee_ownership_partial", "worker_cooperative"} {
		if !scenarios[name] {
			t.Fatalf("scenario %q did not run", name)
		}
	}
}

func TestCommonRandomNumbersAndScenarioOrderIndependence(t *testing.T) {
	cfg := loadV04Config(t)
	cfg.Simulation.Mode = v04config.ModeMonteCarlo
	cfg.Simulation.Months = 4
	cfg.Simulation.HorizonsMonths = []int{4}
	cfg.Simulation.Runs = 3
	cfg.Market.ShockProbabilityMonthly = 0.5
	options := RunOptions{BehaviorCaseNames: []string{"no_effect"}, StoreMonthlyResults: true, StoreRunSummaries: true}
	first, err := Run(cfg, options)
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run <= cfg.Simulation.Runs; run++ {
		for month := 1; month <= cfg.Simulation.Months; month++ {
			var reference *domain.MonthlyResult
			for index := range first.MonthlyResults {
				item := &first.MonthlyResults[index]
				if item.Run != run || item.Month != month {
					continue
				}
				if reference == nil {
					reference = item
					continue
				}
				if item.MarketFactor != reference.MarketFactor || item.ShockHappened != reference.ShockHappened ||
					item.MarketTrend != reference.MarketTrend || item.CreditMarketFactor != reference.CreditMarketFactor {
					t.Fatalf("environment differs within run %d month %d", run, month)
				}
			}
		}
	}

	reordered := cfg.DeepCopy()
	for left, right := 0, len(reordered.OrganizationalScenarios)-1; left < right; left, right = left+1, right-1 {
		reordered.OrganizationalScenarios[left], reordered.OrganizationalScenarios[right] = reordered.OrganizationalScenarios[right], reordered.OrganizationalScenarios[left]
	}
	second, err := Run(reordered, options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		firstJSON, _ := json.Marshal(first)
		secondJSON, _ := json.Marshal(second)
		t.Fatalf("scenario order changed result\nfirst=%s\nsecond=%s", firstJSON, secondJSON)
	}
	repeated, err := Run(cfg, options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, repeated) {
		t.Fatal("same seed was not reproducible")
	}
}

func TestBankruptcyIsAbsorbing(t *testing.T) {
	cfg := deterministicConfig(t, 3)
	cfg.CompanyEconomics.StartingCash = 0
	cfg.CompanyEconomics.OpeningAccountsReceivable = 0
	cfg.Market.RevenueCollectionRateCurrentMonth = 0
	cfg.Financing.BaseCreditLine = 0
	cfg.Financing.ReserveReleaseRateOnStress = 0
	result, err := RunDeterministicScenario(cfg, "traditional_company", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	if !result.MonthlyResults[0].Risks.Bankruptcy || !result.MonthlyResults[0].ActiveCompanyFlag {
		t.Fatalf("month 1 did not become bankrupt correctly: %+v", result.MonthlyResults[0].Risks)
	}
	for _, month := range result.MonthlyResults[1:] {
		if month.ActiveCompanyFlag || !month.Risks.Bankruptcy || month.Revenue != 0 || month.Hires != 0 {
			t.Fatalf("month %d is not absorbing: %+v", month.Month, month)
		}
	}
}

func TestGeneralMandatoryArrearsPersistWhenBankruptcyDoesNotStopRun(t *testing.T) {
	cfg := deterministicConfig(t, 2)
	cfg.Simulation.StopAfterBankruptcy = false
	cfg.CompanyEconomics.StartingCash = 0
	cfg.CompanyEconomics.OpeningAccountsReceivable = 0
	cfg.Market.RevenueCollectionRateCurrentMonth = 0
	cfg.Financing.BaseCreditLine = 0
	cfg.Financing.ReserveReleaseRateOnStress = 0

	result, err := RunDeterministicScenario(cfg, "traditional_company", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	first, second := result.MonthlyResults[0], result.MonthlyResults[1]
	if first.GeneralMandatoryArrearsClose <= 0 {
		t.Fatal("month 1 did not retain unpaid general obligations")
	}
	assertCloseV04(t, "month 2 opening arrears", second.GeneralMandatoryArrearsBegin, first.GeneralMandatoryArrearsClose)
	wantClose := second.GeneralMandatoryArrearsBegin + second.GeneralMandatoryCurrentScheduled - second.GeneralMandatoryPayments
	assertCloseV04(t, "month 2 closing arrears ledger", second.GeneralMandatoryArrearsClose, wantClose)
	if second.GeneralMandatoryArrearsClose <= 0 {
		t.Fatal("month 2 cleared every arrear despite continuing cash stress")
	}
	if !second.ActiveCompanyFlag {
		t.Fatal("stop_after_bankruptcy=false unexpectedly stopped the scenario")
	}
}

func TestQueuesCapacityMemberCapitalAndTaxReserve(t *testing.T) {
	cfg := deterministicConfig(t, 5)
	cfg.Financing.MemberCapitalRedemptionLagMonths = 1
	result, err := RunDeterministicScenario(cfg, "employee_ownership_partial", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	months := result.MonthlyResults
	if months[0].MemberCapitalAllocation <= 0 {
		t.Fatal("month 1 did not allocate member capital")
	}
	if months[1].MemberCapitalRedemptionAccrual <= 0 || months[1].MemberCapitalRedemptionDue != 0 || months[2].MemberCapitalRedemptionDue <= 0 {
		t.Fatalf("member redemption accrual/due = %g/%g", months[1].MemberCapitalRedemptionAccrual, months[2].MemberCapitalRedemptionDue)
	}
	if months[3].CapacityAdditionsDue <= 0 {
		t.Fatalf("month 4 capacity addition due = %g, want >0", months[3].CapacityAdditionsDue)
	}
	for _, month := range months {
		wantReserve := math.Max(0, month.ProfitBeforeTaxBeforeDistribution) * cfg.CompanyEconomics.ProfitTaxRate
		assertCloseV04(t, "tax reserve estimate", month.TaxReserveEstimate, wantReserve)
	}
}

func TestProfitDistributionQueueReleasesRestrictedCash(t *testing.T) {
	cfg := deterministicConfig(t, 2)
	result, err := RunDeterministicScenario(cfg, "profit_sharing", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	first, second := result.MonthlyResults[0], result.MonthlyResults[1]
	if first.EmployeeCashDistributionAccrued <= 0 {
		t.Fatal("month 1 distribution was not accrued")
	}
	assertCloseV04(t, "month 2 paid gross", second.EmployeeCashDistributionPaid, first.EmployeeCashDistributionAccrued)
	paidEmployerCost := second.EmployeeCashDistributionPaid + second.EmployeeDistributionPayrollTaxPaid
	assertCloseV04(t, "restricted release", second.RestrictedDistributionBegin-second.RestrictedDistributionClose+second.RestrictedDistributionCashNew, paidEmployerCost)
}

func TestAllDefaultBehaviorCasesProduceFiniteResults(t *testing.T) {
	cfg := deterministicConfig(t, 3)
	result, err := Run(cfg, RunOptions{StoreMonthlyResults: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.MonthlyResults) == 0 {
		t.Fatal("no monthly results")
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("result contains NaN/Inf: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty JSON")
	}
	negativeSeen := false
	for _, month := range result.MonthlyResults {
		if month.BehaviorCase == "negative_fairness_free_rider" && month.ProductivityUplift < 0 {
			negativeSeen = true
		}
	}
	if !negativeSeen {
		t.Fatal("negative behavior case did not produce a negative productivity control")
	}
}

func TestMonthExecutionOrderAndAllDueQueues(t *testing.T) {
	cfg := deterministicConfig(t, 3)
	cfg.Workforce.BaseTurnoverRateAnnual = 0
	cfg.Workforce.MinTurnoverRateAnnual = 0
	cfg.Workforce.MaxTurnoverRateAnnual = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	cfg.Market.AccountsReceivableLagMonths = 1
	cfg.CompanyEconomics.ProfitTaxPaymentLagMonths = 1
	cfg.Financing.EmployeeDistributionPayoutLagMonths = 1
	cfg.CompanyEconomics.InvestmentActivationLagMonths = 2
	cfg.CompanyEconomics.CapacityDepreciationRateMonthly = 0

	result, err := RunDeterministicScenario(cfg, "profit_sharing", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	first, second, third := result.MonthlyResults[0], result.MonthlyResults[1], result.MonthlyResults[2]
	if first.NewAccountsReceivable <= 0 || first.ProfitTaxAccrual <= 0 || first.EmployeeCashDistributionAccrued <= 0 || first.CapacityAddedByInvestment <= 0 {
		t.Fatalf("month-one fixture did not populate every queue: %+v", first)
	}
	assertCloseV04(t, "cash state continuity", second.CashTotalBegin, first.CashTotalClose)
	assertCloseV04(t, "AR state continuity", second.OpeningAccountsReceivable, first.ClosingAccountsReceivable)
	assertCloseV04(t, "AR due after one month", second.CashCollectedFromAR, first.NewAccountsReceivable)
	assertCloseV04(t, "tax payable state continuity", second.TaxPayableBegin, first.TaxPayableClose)
	assertCloseV04(t, "tax due after one month", second.TaxesDue, first.ProfitTaxAccrual)
	assertCloseV04(t, "distribution due gross", second.EmployeeDistributionDueGross, first.EmployeeCashDistributionAccrued)
	assertCloseV04(t, "distribution due payroll tax", second.EmployeeDistributionDuePayrollTax, first.DistributionPayrollTaxAccrued)
	assertCloseV04(t, "no early capacity activation month 2", second.CapacityAdditionsDue, 0)
	assertCloseV04(t, "capacity activates at exact due month", third.CapacityAdditionsDue, first.CapacityAddedByInvestment)
}

func TestCashAndDebtIdentityWithLiquidityAndGrowthCredit(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	cfg.CompanyEconomics.StartingCash = 0
	cfg.CompanyEconomics.OpeningAccountsReceivable = 0
	cfg.Market.RevenueCollectionRateCurrentMonth = 0
	cfg.Financing.BaseCreditLine = 100_000_000
	result, err := RunDeterministicScenario(cfg, "traditional_company", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	month := result.MonthlyResults[0]
	if month.CreditDrawForLiquidity <= 0 {
		t.Fatal("fixture did not draw liquidity credit")
	}
	wantCash := month.CashTotalBegin + month.CashCollectedCurrent + month.CashCollectedFromAR +
		month.CreditDrawForLiquidity + month.ExternalGrowthCapitalDraw - month.MandatoryCashPayments -
		month.ReinvestmentCashPaid - month.ExternalDistributionPaid - month.ExternalGrowthCapitalSpent
	assertCloseV04(t, "cash identity with credit", month.CashTotalClose, wantCash)
	wantDebt := month.DebtBalanceBegin - month.PrincipalPaid + month.CreditDrawForLiquidity + month.ExternalGrowthCapitalDraw
	assertCloseV04(t, "debt identity with credit", month.DebtBalanceClose, wantDebt)
}

func TestEngineRemainsFiniteWhenDynamicHeadcountReachesZero(t *testing.T) {
	cfg := deterministicConfig(t, 3)
	cfg.Workforce.BaseTurnoverRateAnnual = 1
	cfg.Workforce.MinTurnoverRateAnnual = 1
	cfg.Workforce.MaxTurnoverRateAnnual = 1
	cfg.Workforce.MaxHiresPerMonthRate = 0
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	cfg.CompanyEconomics.StartingCash = 100_000_000
	result, err := RunDeterministicScenario(cfg, "traditional_company", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	if result.MonthlyResults[0].HeadcountEnd != 0 || result.MonthlyResults[1].HeadcountBegin != 0 {
		t.Fatalf("headcount did not reach and retain zero: month1=%g month2=%g",
			result.MonthlyResults[0].HeadcountEnd, result.MonthlyResults[1].HeadcountBegin)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("zero-headcount result contains NaN/Inf: %v", err)
	}
}

func TestV04PeriodicDistributionAccruesAtPeriodClose(t *testing.T) {
	cfg := deterministicConfig(t, 2)
	cfg.CompanyEconomics.StartingCash = 100_000_000
	cfg.Market.MarketGrowthMonthly = 0
	cfg.Market.ShockProbabilityMonthly = 0
	cfg.Workforce.BaseTurnoverRateAnnual = 0
	cfg.Workforce.MinTurnoverRateAnnual = 0
	cfg.Workforce.MaxTurnoverRateAnnual = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	scenario, _ := cfg.ScenarioByName("profit_sharing")
	scenario.ProfitDistributionPeriodMonths = 2
	result, err := RunDeterministicScenario(cfg, scenario.Name, "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	first, second := result.MonthlyResults[0], result.MonthlyResults[1]
	assertCloseV04(t, "no distribution before period close", first.EmployeeCashDistributionAccrued, 0)
	want := scenario.EmployeeCashDistributionRate * (first.PositiveResultBase + second.PositiveResultBase)
	assertCloseV04(t, "period-close distribution", second.EmployeeCashDistributionAccrued, want)
}

func TestSummaryAssumptionFlagsArePopulatedAndBehaviorSpecific(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	result, err := Run(cfg, RunOptions{
		ScenarioNames: []string{"worker_cooperative"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range result.TerminalSummaries {
		joined := strings.Join(summary.AssumptionFlags, "\n")
		if !strings.Contains(joined, v04config.AssumptionSimplifiedTaxModel) {
			t.Fatalf("summary %s has no simplified-tax assumption flag: %v", summary.BehaviorCase, summary.AssumptionFlags)
		}
		hasHighPerformer := strings.Contains(joined, v04config.AssumptionHighPerformerOnly)
		wantHighPerformer := cfg.BehaviorCases[summary.BehaviorCase].HighPerformerAttritionDeltaPP != 0
		if hasHighPerformer != wantHighPerformer {
			t.Fatalf("summary %s high-performer flag = %v, want %v: %v",
				summary.BehaviorCase, hasHighPerformer, wantHighPerformer, summary.AssumptionFlags)
		}
	}
}

func deterministicConfig(t *testing.T, months int) v04config.Config {
	t.Helper()
	cfg := loadV04Config(t)
	cfg.Simulation.Mode = v04config.ModeDeterministic
	cfg.Simulation.Runs = 1
	cfg.Simulation.Months = months
	cfg.Simulation.HorizonsMonths = uniqueHorizons(months)
	return cfg
}

func uniqueHorizons(months int) []int {
	values := []int{1, months}
	sort.Ints(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func loadV04Config(t *testing.T) v04config.Config {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	cfg, err := v04config.LoadFile(filepath.Join(root, "doc", "default_config_v0_4.json"))
	if err != nil {
		t.Fatalf("load v0.4 config: %v", err)
	}
	return cfg
}

func assertCloseV04(t *testing.T, name string, got, want float64) {
	t.Helper()
	if !domain.AlmostEqual(got, want, 1e-9) {
		t.Fatalf("%s = %.12f, want %.12f", name, got, want)
	}
}
