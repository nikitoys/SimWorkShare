package sim

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

func TestCanonicalProfitShareMonthOneMatchesGoldenJSON(t *testing.T) {
	cfg := loadDefaultConfig(t)
	result, err := RunDeterministicScenario(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("RunDeterministicScenario() error = %v", err)
	}
	got, err := json.Marshal(result.MonthlyResults[0])
	if err != nil {
		t.Fatalf("marshal month 1: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "profit_share_month1_golden.json"))
	if err != nil {
		t.Fatalf("read profit-share golden: %v", err)
	}
	var compactWant bytes.Buffer
	if err := json.Compact(&compactWant, want); err != nil {
		t.Fatalf("compact profit-share golden: %v", err)
	}
	if !bytes.Equal(got, compactWant.Bytes()) {
		t.Fatalf("profit-share month 1 JSON changed\n--- got ---\n%s\n--- want ---\n%s", got, compactWant.Bytes())
	}
}

func TestCanonicalProfitShareMonthOneAndQueueContinuity(t *testing.T) {
	cfg := loadDefaultConfig(t)
	result, err := RunDeterministicScenario(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("RunDeterministicScenario() error = %v", err)
	}
	if len(result.MonthlyResults) != cfg.Simulation.Months {
		t.Fatalf("monthly results = %d, want %d", len(result.MonthlyResults), cfg.Simulation.Months)
	}

	fixed, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicFixedOnly() error = %v", err)
	}
	first := result.MonthlyResults[0]
	if first.Compensation == nil {
		t.Fatal("month 1 has no compensation state")
	}
	comp := first.Compensation
	assertMoneyClose(t, "month 1 operating profit", first.PnL.OperatingProfitBeforeBonus, 1_509_853.3117109481)
	assertMoneyClose(t, "month 1 gross bonus", comp.GrossBonusPoolAccrued, 150_985.33117109482)
	assertMoneyClose(t, "month 1 payroll tax", comp.BonusPayrollTaxAccrued, 0)
	assertMoneyClose(t, "month 1 profit tax", first.PnL.ProfitTaxAccrual, 271_773.5961079707)
	assertMoneyClose(t, "bonus accrual leaves total cash unchanged", first.Cash.ClosingCashTotal,
		fixed.MonthlyResults[0].Cash.ClosingCashTotal)
	assertMoneyClose(t, "month 1 restricted cash", first.Cash.RestrictedBonusCash, 150_985.33117109482)
	assertMoneyClose(t, "month 1 unrestricted cash", first.Cash.ClosingUnrestrictedCash, 16_358_867.980539853)

	for index, month := range result.MonthlyResults {
		if month.Compensation == nil {
			t.Fatalf("month %d has no compensation state", month.Month)
		}
		comp := month.Compensation
		if index == 0 {
			assertMoneyClose(t, "opening bonus payable", comp.OpeningBonusPayable, 0)
			assertMoneyClose(t, "first bonus payment", comp.BonusPaidCash, 0)
		} else {
			previous := result.MonthlyResults[index-1]
			assertMoneyClose(t, "opening bonus payable continuity", comp.OpeningBonusPayable,
				previous.Compensation.ClosingBonusPayable)
			assertMoneyClose(t, "restricted cash continuity before payment", comp.OpeningBonusPayable,
				previous.Cash.RestrictedBonusCash)
			origin := result.MonthlyResults[index-*scenarioByNameForTest(t, cfg, "profit_share_equal_10").BonusPayoutLagMonths]
			assertMoneyClose(t, "gross bonus paid on due month", comp.BonusPaidCash,
				origin.Compensation.GrossBonusPoolAccrued)
			assertMoneyClose(t, "bonus payroll tax paid on due month", comp.BonusPayrollTaxPaid,
				origin.Compensation.BonusPayrollTaxAccrued)
		}
		assertMoneyClose(t, "restricted equals bonus payable", month.Cash.RestrictedBonusCash,
			comp.ClosingBonusPayable)
	}

	terminal := result.TerminalSummary.Compensation
	if terminal == nil {
		t.Fatal("terminal compensation summary is nil")
	}
	assertMoneyClose(t, "terminal restricted/payable", terminal.ClosingRestrictedBonusCash,
		terminal.ClosingBonusPayable)
}

func TestProfitShareBonusAndTaxLedgersFiveMonths(t *testing.T) {
	cfg := simpleProfitShareConfig(t)
	result, err := RunDeterministicScenario(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("RunDeterministicScenario() error = %v", err)
	}

	wantRevenue := []domain.Money{100, 200, 400, 800, 1_600}
	wantGrossAccrued := []domain.Money{10, 20, 40, 80, 160}
	wantPayrollAccrued := []domain.Money{2.5, 5, 10, 20, 40}
	wantGrossPaid := []domain.Money{0, 0, 10, 20, 40}
	wantPayrollPaid := []domain.Money{0, 0, 2.5, 5, 10}
	wantTaxAccrued := []domain.Money{17.5, 35, 70, 140, 280}
	wantTaxPaid := []domain.Money{0, 0, 0, 17.5, 35}
	wantRestricted := []domain.Money{12.5, 37.5, 75, 150, 300}
	wantClosingCash := []domain.Money{10_100, 10_300, 10_687.5, 11_445, 12_960}

	for index, month := range result.MonthlyResults {
		comp := month.Compensation
		if comp == nil {
			t.Fatalf("month %d compensation is nil", month.Month)
		}
		assertMoneyClose(t, "revenue", month.PnL.Revenue, wantRevenue[index])
		assertMoneyClose(t, "gross accrued", comp.GrossBonusPoolAccrued, wantGrossAccrued[index])
		assertMoneyClose(t, "payroll accrued", comp.BonusPayrollTaxAccrued, wantPayrollAccrued[index])
		assertMoneyClose(t, "gross paid", comp.BonusPaidCash, wantGrossPaid[index])
		assertMoneyClose(t, "payroll paid", comp.BonusPayrollTaxPaid, wantPayrollPaid[index])
		assertMoneyClose(t, "tax accrued", month.PnL.ProfitTaxAccrual, wantTaxAccrued[index])
		assertMoneyClose(t, "tax paid", month.Cash.TaxesPaidCash, wantTaxPaid[index])
		assertMoneyClose(t, "restricted cash", month.Cash.RestrictedBonusCash, wantRestricted[index])
		assertMoneyClose(t, "closing cash", month.Cash.ClosingCashTotal, wantClosingCash[index])
	}

	terminal := result.TerminalSummary.Compensation
	if terminal == nil {
		t.Fatal("terminal compensation summary is nil")
	}
	assertMoneyClose(t, "terminal gross payable", terminal.ClosingBonusPayableGross, 240)
	assertMoneyClose(t, "terminal payroll payable", terminal.ClosingBonusPayablePayrollTax, 60)
	assertMoneyClose(t, "terminal total payable", terminal.ClosingBonusPayable, 300)
	assertMoneyClose(t, "terminal restricted", terminal.ClosingRestrictedBonusCash, 300)
	assertMoneyClose(t, "terminal tax payable", result.TerminalSummary.ClosingTaxPayable, 490)
}

func TestProfitShareZeroEqualsFixedOnlyAndFixedRegression(t *testing.T) {
	fixedCfg := loadDefaultConfig(t)
	fixedViaWrapper, err := RunDeterministicFixedOnly(fixedCfg)
	if err != nil {
		t.Fatalf("RunDeterministicFixedOnly() error = %v", err)
	}
	fixedViaGeneric, err := RunDeterministicScenario(fixedCfg, "fixed_only", "no_effect")
	if err != nil {
		t.Fatalf("RunDeterministicScenario(fixed_only) error = %v", err)
	}
	if !reflect.DeepEqual(fixedViaWrapper, fixedViaGeneric) {
		t.Fatal("generic fixed_only result differs from preserved wrapper")
	}

	zeroCfg := loadDefaultConfig(t)
	zero := 0.0
	mutateScenarioForTest(t, &zeroCfg, "profit_share_equal_10", func(s *config.CompensationScenario) {
		s.ProfitSharePercent = &zero
	})
	profitZero, err := RunDeterministicScenario(zeroCfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("zero-share run error = %v", err)
	}
	for index := range fixedViaWrapper.MonthlyResults {
		fixedMonth := fixedViaWrapper.MonthlyResults[index]
		profitMonth := profitZero.MonthlyResults[index]
		if !reflect.DeepEqual(fixedMonth.Environment, profitMonth.Environment) ||
			!reflect.DeepEqual(fixedMonth.Workforce, profitMonth.Workforce) ||
			!reflect.DeepEqual(fixedMonth.PnL, profitMonth.PnL) ||
			!reflect.DeepEqual(fixedMonth.Cash, profitMonth.Cash) {
			t.Fatalf("month %d zero-share economics differ from fixed_only", index+1)
		}
		if profitMonth.Compensation == nil || profitMonth.Compensation.ClosingBonusPayable != 0 {
			t.Fatalf("month %d zero-share created bonus state", index+1)
		}
	}
}

func TestPairedFixedVsProfitShareComparisonIsDeterministic(t *testing.T) {
	cfg := simpleProfitShareConfig(t)
	configBefore, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config before comparison: %v", err)
	}
	first, err := RunDeterministicComparison(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("RunDeterministicComparison() error = %v", err)
	}
	second, err := RunDeterministicComparison(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("second comparison error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated paired comparisons differ")
	}
	configAfter, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config after comparison: %v", err)
	}
	if string(configBefore) != string(configAfter) {
		t.Fatal("comparison runner mutated config")
	}

	wantFixedCash := []domain.Money{10_100, 10_300, 10_700, 11_480, 13_040}
	var previousCashDelta domain.Money
	for index := range first.FixedOnly.MonthlyResults {
		fixedMonth := first.FixedOnly.MonthlyResults[index]
		profitMonth := first.ProfitShare.MonthlyResults[index]
		if !reflect.DeepEqual(fixedMonth.Environment, profitMonth.Environment) ||
			!reflect.DeepEqual(fixedMonth.Workforce, profitMonth.Workforce) ||
			fixedMonth.PnL.Revenue != profitMonth.PnL.Revenue ||
			fixedMonth.PnL.TotalCostsBeforeBonus != profitMonth.PnL.TotalCostsBeforeBonus {
			t.Fatalf("month %d comparison arms do not share the same pre-bonus path", index+1)
		}
		assertMoneyClose(t, "fixed closing cash", fixedMonth.Cash.ClosingCashTotal, wantFixedCash[index])
		cashDelta := fixedMonth.Cash.ClosingCashTotal - profitMonth.Cash.ClosingCashTotal
		bonusPaid := profitMonth.Compensation.BonusPaidCash + profitMonth.Compensation.BonusPayrollTaxPaid
		expectedDelta := previousCashDelta + bonusPaid + profitMonth.Cash.TaxesPaidCash - fixedMonth.Cash.TaxesPaidCash
		assertMoneyClose(t, "paired cash bridge", cashDelta, expectedDelta)
		previousCashDelta = cashDelta
	}

	assertMoneyClose(t, "final cash delta", first.Summary.FinalClosingCashDelta, -80)
	assertMoneyClose(t, "final unrestricted cash delta", first.Summary.FinalUnrestrictedCashDelta, -380)
	assertMoneyClose(t, "final owner cash delta", first.Summary.FinalOwnerDistributableCashDelta, -380)
	assertMoneyClose(t, "cumulative accounting profit delta", first.Summary.CumulativeAccountingProfitDelta, -310)
	assertMoneyClose(t, "total gross bonus accrued", first.Summary.TotalGrossBonusAccrued, 310)
	assertMoneyClose(t, "total employer bonus accrued", first.Summary.TotalEmployerBonusCostAccrued, 387.5)
	assertMoneyClose(t, "total gross bonus paid", first.Summary.TotalGrossBonusPaid, 70)
	assertMoneyClose(t, "total employer bonus paid", first.Summary.TotalEmployerBonusCostPaid, 87.5)

	cfg.Simulation.Runs = 7
	cfg.Simulation.RandomSeed = 999
	cfg.Simulation.CommonRandomNumbers = false
	changedControls, err := RunDeterministicComparison(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("comparison with changed Monte Carlo controls: %v", err)
	}
	if !reflect.DeepEqual(first, changedControls) {
		t.Fatal("unused Monte Carlo controls changed paired deterministic output")
	}
}

func TestProfitShareAcceptsExplicitBehaviorAssumption(t *testing.T) {
	cfg := loadDefaultConfig(t)
	noEffect, err := RunDeterministicScenario(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("no-effect run error = %v", err)
	}
	moderate, err := RunDeterministicScenario(cfg, "profit_share_equal_10", "moderate_effect")
	if err != nil {
		t.Fatalf("moderate-effect run error = %v", err)
	}
	if moderate.BehaviorCase != "moderate_effect" || moderate.MonthlyResults[0].BehaviorCase != "moderate_effect" {
		t.Fatalf("moderate behavior metadata = %q/%q", moderate.BehaviorCase, moderate.MonthlyResults[0].BehaviorCase)
	}
	assertFloatClose(t, "moderate productivity uplift", moderate.MonthlyResults[0].Workforce.ProductivityUplift, 0.02)
	if moderate.MonthlyResults[0].PnL.Revenue <= noEffect.MonthlyResults[0].PnL.Revenue {
		t.Fatal("moderate behavior did not increase revenue relative to no_effect")
	}
	comparison, err := RunDeterministicComparison(cfg, "profit_share_equal_10", "moderate_effect")
	if err != nil {
		t.Fatalf("moderate comparison error = %v", err)
	}
	if comparison.FixedOnly.BehaviorCase != "no_effect" || comparison.ProfitShare.BehaviorCase != "moderate_effect" {
		t.Fatalf("comparison behaviors = %q/%q",
			comparison.FixedOnly.BehaviorCase, comparison.ProfitShare.BehaviorCase)
	}
}

func TestSelectedProfitShareScopeIsExplicit(t *testing.T) {
	cfg := loadDefaultConfig(t)
	tests := []struct {
		name     string
		scenario string
		behavior string
		want     string
	}{
		{name: "unknown scenario", scenario: "missing", behavior: "no_effect", want: "unknown compensation scenario"},
		{name: "fixed raise", scenario: "fixed_raise_same_expected_cost_for_10", behavior: "no_effect", want: "outside the current deterministic stage"},
		{name: "quarterly", scenario: "profit_share_equal_10_quarterly", behavior: "no_effect", want: "only monthly bonus period"},
		{name: "annual", scenario: "profit_share_equal_10_annual", behavior: "no_effect", want: "only monthly bonus period"},
		{name: "fixed behavior", scenario: "fixed_only", behavior: "moderate_effect", want: "fixed_only supports only behavior"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RunDeterministicScenario(cfg, tt.scenario, tt.behavior)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}

	lagZeroCfg := loadDefaultConfig(t)
	zero := 0
	mutateScenarioForTest(t, &lagZeroCfg, "profit_share_equal_10", func(s *config.CompensationScenario) {
		s.BonusPayoutLagMonths = &zero
	})
	if _, err := RunDeterministicScenario(lagZeroCfg, "profit_share_equal_10", "no_effect"); err == nil || !strings.Contains(err.Error(), "bonus payout lag must be >= 1") {
		t.Fatalf("lag-zero error = %v", err)
	}
}

func TestProfitShareCashSafetyFlags(t *testing.T) {
	cashLimited := loadDefaultConfig(t)
	cashLimited.Simulation.Months = 1
	cashLimited.Company.StartingCash = -100_000_000
	result, err := RunDeterministicScenario(cashLimited, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("cash-limited run error = %v", err)
	}
	month := result.MonthlyResults[0]
	if month.Compensation == nil || month.Compensation.GrossBonusPoolAccrued != 0 {
		t.Fatalf("cash-limited bonus = %+v", month.Compensation)
	}
	if !month.Risks.BonusNotAccruedDueToCash || !month.Risks.ZeroBonusPeriod ||
		!result.TerminalSummary.RiskFlagsEver.BonusNotAccruedDueToCash {
		t.Fatalf("cash-safety flags = monthly %+v, terminal %+v", month.Risks, result.TerminalSummary.RiskFlagsEver)
	}

	zeroCap := loadDefaultConfig(t)
	zeroCap.Simulation.Months = 1
	cap := 0.0
	mutateScenarioForTest(t, &zeroCap, "profit_share_equal_10", func(s *config.CompensationScenario) {
		s.BonusCapTotal = &cap
	})
	result, err = RunDeterministicScenario(zeroCap, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("zero-cap run error = %v", err)
	}
	month = result.MonthlyResults[0]
	if !month.Risks.ZeroBonusPeriod || month.Risks.BonusNotAccruedDueToCash {
		t.Fatalf("zero-cap flags = %+v", month.Risks)
	}
}

func TestProfitShareAffordabilityIncludesReserveReinvestmentAndOldRestrictedCash(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Simulation.Months = 2
	cfg.Company.EmployeesCount = 1
	cfg.Company.BaseSalaryPerEmployee = 100
	cfg.Company.BaseRevenuePerEmployee = 1_000
	cfg.Company.FixedCostsMonthly = 0
	cfg.Company.VariableCostRate = 0
	cfg.Company.StartingCash = 970
	cfg.Company.OpeningAccountsReceivable = 0
	cfg.Company.RequiredCashReserveMonths = 1
	cfg.Company.RevenueProductivityElasticity = 0
	cfg.Company.DemandCapMultiplier = nil
	cfg.Cashflow.RevenueCollectionRateCurrentMonth = 0
	cfg.Cashflow.AccountsReceivableLagMonths = 3
	cfg.Cashflow.ProfitTaxRate = 0.20
	cfg.Cashflow.ProfitTaxPaymentLagMonths = 3
	cfg.Cashflow.ReserveCurrentProfitTax = true
	cfg.Cashflow.BonusPayrollTaxRate = 0.25
	cfg.Cashflow.PlannedReinvestmentRate = 0.10
	cfg.Workforce.RecruitingCostPerLeaver = 0
	cfg.Workforce.OnboardingCostPerLeaver = 0
	cfg.Workforce.ManagerTimeCostPerLeaver = 0
	cfg.Workforce.LostProductivityCostPerLeaver = 0
	cfg.Environment.MarketGrowthMonthly = 0
	cfg.Environment.CostInflationMonthly = 0
	for index := range cfg.CompensationScenarios {
		if cfg.CompensationScenarios[index].Type == "profit_share" {
			eligible := 1
			cfg.CompensationScenarios[index].EligibleEmployeesCount = &eligible
		}
	}
	mutateScenarioForTest(t, &cfg, "profit_share_equal_10", func(s *config.CompensationScenario) {
		share := 0.30
		lag := 2
		s.ProfitSharePercent = &share
		s.BonusPayoutLagMonths = &lag
		s.BonusCapTotal = nil
		s.BonusCapPerEmployee = nil
	})

	result, err := RunDeterministicScenario(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("RunDeterministicScenario() error = %v", err)
	}
	first := result.MonthlyResults[0]
	second := result.MonthlyResults[1]
	assertMoneyClose(t, "month 1 cash after due", first.Cash.ClosingCashTotal, 870)
	assertMoneyClose(t, "month 1 gross bonus", first.Compensation.GrossBonusPoolAccrued, 270)
	assertMoneyClose(t, "month 1 restricted", first.Cash.RestrictedBonusCash, 337.5)
	assertMoneyClose(t, "month 2 cash after due", second.Cash.ClosingCashTotal, 770)
	assertMoneyClose(t, "month 2 old restricted opening", second.Compensation.OpeningBonusPayable, 337.5)
	assertMoneyClose(t, "month 2 required reserve", second.Cash.RequiredCashReserve, 100)
	assertMoneyClose(t, "month 2 cash base", second.Compensation.CashBaseForTotalEmployerBonusCost, 62.5)
	assertMoneyClose(t, "month 2 cash-capped gross", second.Compensation.GrossBonusPoolAccrued, 50)
	assertMoneyClose(t, "month 2 cash-capped payroll", second.Compensation.BonusPayrollTaxAccrued, 12.5)
	if !second.Risks.BonusNotAccruedDueToCash {
		t.Fatal("month 2 did not report cash-capped bonus")
	}
}

func TestProfitShareRunnerPreservesSmallFuturePayableAcrossScales(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Simulation.Months = 4
	cfg.Company.EmployeesCount = 1
	cfg.Company.BaseSalaryPerEmployee = 0
	cfg.Company.BaseRevenuePerEmployee = 1e20
	cfg.Company.FixedCostsMonthly = 0
	cfg.Company.VariableCostRate = 0
	cfg.Company.StartingCash = 1e20
	cfg.Company.OpeningAccountsReceivable = 0
	cfg.Company.RequiredCashReserveMonths = 0
	cfg.Company.RevenueProductivityElasticity = 0
	cfg.Company.DemandCapMultiplier = nil
	cfg.Cashflow.RevenueCollectionRateCurrentMonth = 1
	cfg.Cashflow.BadDebtRate = 0
	cfg.Cashflow.ProfitTaxRate = 0
	cfg.Cashflow.BonusPayrollTaxRate = 0
	cfg.Cashflow.PlannedReinvestmentRate = 0
	cfg.Workforce.RecruitingCostPerLeaver = 0
	cfg.Workforce.OnboardingCostPerLeaver = 0
	cfg.Workforce.ManagerTimeCostPerLeaver = 0
	cfg.Workforce.LostProductivityCostPerLeaver = 0
	cfg.Environment.MarketGrowthMonthly = math.Nextafter(-1, 0)
	cfg.Environment.CostInflationMonthly = 0
	for index := range cfg.CompensationScenarios {
		if cfg.CompensationScenarios[index].Type == "profit_share" {
			eligible := 1
			cfg.CompensationScenarios[index].EligibleEmployeesCount = &eligible
		}
	}
	mutateScenarioForTest(t, &cfg, "profit_share_equal_10", func(s *config.CompensationScenario) {
		share := 0.10
		lag := 2
		s.ProfitSharePercent = &share
		s.BonusPayoutLagMonths = &lag
		s.BonusCapTotal = nil
		s.BonusCapPerEmployee = nil
	})

	result, err := RunDeterministicScenario(cfg, "profit_share_equal_10", "no_effect")
	if err != nil {
		t.Fatalf("cross-scale runner error = %v", err)
	}
	month3 := result.MonthlyResults[2]
	month4 := result.MonthlyResults[3]
	if month3.Compensation.BonusPaidCash < 1e18 {
		t.Fatalf("month 3 did not pay the large entry: %.12g", month3.Compensation.BonusPaidCash)
	}
	if month3.Compensation.ClosingBonusPayable <= 0 ||
		month3.Cash.RestrictedBonusCash != month3.Compensation.ClosingBonusPayable {
		t.Fatalf("small future entry was lost after large payment: compensation=%+v cash=%+v",
			month3.Compensation, month3.Cash)
	}
	assertMoneyClose(t, "month 4 pays preserved small entry", month4.Compensation.BonusPaidCash,
		result.MonthlyResults[1].Compensation.GrossBonusPoolAccrued)
}

func simpleProfitShareConfig(t *testing.T) config.Config {
	t.Helper()
	cfg := loadDefaultConfig(t)
	cfg.Simulation.Months = 5
	cfg.Company.EmployeesCount = 1
	cfg.Company.BaseSalaryPerEmployee = 0
	cfg.Company.BaseRevenuePerEmployee = 100
	cfg.Company.FixedCostsMonthly = 0
	cfg.Company.VariableCostRate = 0
	cfg.Company.StartingCash = 10_000
	cfg.Company.OpeningAccountsReceivable = 0
	cfg.Company.RequiredCashReserveMonths = 0
	cfg.Company.RevenueProductivityElasticity = 0
	cfg.Company.DemandCapMultiplier = nil
	cfg.Cashflow.RevenueCollectionRateCurrentMonth = 1
	cfg.Cashflow.AccountsReceivableLagMonths = 1
	cfg.Cashflow.BadDebtRate = 0
	cfg.Cashflow.ProfitTaxRate = 0.20
	cfg.Cashflow.ProfitTaxPaymentLagMonths = 3
	cfg.Cashflow.ReserveCurrentProfitTax = false
	cfg.Cashflow.BonusPayrollTaxRate = 0.25
	cfg.Cashflow.DebtServiceMonthly = 0
	cfg.Cashflow.CapexMonthly = 0
	cfg.Cashflow.PlannedReinvestmentRate = 0
	cfg.Workforce.RecruitingCostPerLeaver = 0
	cfg.Workforce.OnboardingCostPerLeaver = 0
	cfg.Workforce.ManagerTimeCostPerLeaver = 0
	cfg.Workforce.LostProductivityCostPerLeaver = 0
	cfg.Environment.MarketGrowthMonthly = 1
	cfg.Environment.CostInflationMonthly = 0
	for index := range cfg.CompensationScenarios {
		if cfg.CompensationScenarios[index].Type == "profit_share" {
			eligible := 1
			cfg.CompensationScenarios[index].EligibleEmployeesCount = &eligible
		}
	}

	mutateScenarioForTest(t, &cfg, "profit_share_equal_10", func(s *config.CompensationScenario) {
		share := 0.10
		hurdle := 0.0
		eligible := 1
		lag := 2
		s.ProfitSharePercent = &share
		s.ProfitHurdleMonthly = &hurdle
		s.EligibleEmployeesCount = &eligible
		s.BonusPayoutLagMonths = &lag
		s.BonusCapTotal = nil
		s.BonusCapPerEmployee = nil
	})
	return cfg
}

func scenarioByNameForTest(t *testing.T, cfg config.Config, name string) config.CompensationScenario {
	t.Helper()
	for _, scenario := range cfg.CompensationScenarios {
		if scenario.Name == name {
			return scenario
		}
	}
	t.Fatalf("scenario %q not found", name)
	return config.CompensationScenario{}
}

func mutateScenarioForTest(t *testing.T, cfg *config.Config, name string, mutate func(*config.CompensationScenario)) {
	t.Helper()
	for index := range cfg.CompensationScenarios {
		if cfg.CompensationScenarios[index].Name == name {
			mutate(&cfg.CompensationScenarios[index])
			return
		}
	}
	t.Fatalf("scenario %q not found", name)
}
