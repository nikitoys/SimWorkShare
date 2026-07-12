package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	v04config "simworkshare/internal/v04/config"
)

func TestRequiredTestsManifestIsFullyRepresented(t *testing.T) {
	// Each ID is exercised either in this file or by the focused config,
	// mechanism, engine, and v0.3 compatibility tests in the same test run.
	implemented := map[string]bool{
		"baseline_static_traditional_sanity":                true,
		"headcount_identity":                                true,
		"no_effect_no_productivity_gain":                    true,
		"profit_sharing_no_ownership":                       true,
		"ownership_no_distribution_possible":                true,
		"governance_admin_cost_reduces_effective_employees": true,
		"cash_safe_distribution":                            true,
		"distribution_reserve_at_accrual":                   true,
		"distribution_payment_releases_restricted_cash":     true,
		"member_capital_redemption_queue":                   true,
		"reinvestment_capacity_lag":                         true,
		"external_capital_constraint":                       true,
		"common_random_numbers_market":                      true,
		"shock_response_quality":                            true,
		"bankruptcy_absorbing":                              true,
		"allocation_sum_validation":                         true,
		"strict_unknown_field":                              true,
		"duplicate_json_field":                              true,
		"v03_compat_fixed_only":                             true,
		"v03_compat_profit_share_10_no_effect":              true,
	}
	data, err := os.ReadFile(filepath.Join(v04RepositoryRoot(t), "doc", "required_tests_v0_4.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Tests []struct {
			ID string `json:"id"`
		} `json:"tests"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode required test manifest: %v", err)
	}
	seen := make(map[string]bool, len(manifest.Tests))
	for _, test := range manifest.Tests {
		if seen[test.ID] {
			t.Fatalf("duplicate required test ID %q", test.ID)
		}
		seen[test.ID] = true
		if !implemented[test.ID] {
			t.Errorf("required test %q is not represented", test.ID)
		}
	}
	if len(seen) != len(implemented) {
		var extras []string
		for id := range implemented {
			if !seen[id] {
				extras = append(extras, id)
			}
		}
		sort.Strings(extras)
		t.Fatalf("test mapping differs from manifest; extra IDs: %v", extras)
	}
}

func TestRequiredHeadcountIdentity(t *testing.T) {
	cfg := deterministicConfig(t, 4)
	result, err := RunDeterministicScenario(cfg, "traditional_company", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	for _, month := range result.MonthlyResults {
		want := month.HeadcountBegin - month.VoluntaryLeavers - month.Layoffs + month.Hires
		assertCloseV04(t, "headcount identity", month.HeadcountEnd, want)
	}
}

func TestRequiredProfitSharingNoOwnership(t *testing.T) {
	cfg := loadV04Config(t)
	scenario, ok := cfg.ScenarioByName("profit_sharing")
	if !ok {
		t.Fatal("profit_sharing scenario is missing")
	}
	if scenario.EmployeeOwnershipFraction != 0 || scenario.Governance.GovernanceParticipationIntensity != 0 {
		t.Fatalf("profit_sharing ownership/governance = %g/%g, want 0/0",
			scenario.EmployeeOwnershipFraction, scenario.Governance.GovernanceParticipationIntensity)
	}
}

func TestRequiredOwnershipNoDistributionPossible(t *testing.T) {
	cfg := deterministicConfig(t, 2)
	scenario, ok := cfg.ScenarioByName("employee_ownership_partial")
	if !ok {
		t.Fatal("employee_ownership_partial scenario is missing")
	}
	if scenario.EmployeeOwnershipFraction <= 0 {
		t.Fatal("fixture must have positive employee ownership")
	}
	scenario.EmployeeCashDistributionRate = 0
	scenario.AllocationPriority = removeAllocation(scenario.AllocationPriority, v04config.AllocationEmployeeDistribution)
	result, err := RunDeterministicScenario(cfg, scenario.Name, "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	for _, month := range result.MonthlyResults {
		if month.EmployeeCashDistributionAccrued != 0 || month.EmployeeCashDistributionPaid != 0 {
			t.Fatalf("month %d ownership alone created cash distribution", month.Month)
		}
	}
}

func TestRequiredGovernanceAdminCostReducesEffectiveEmployees(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	cfg.Workforce.RampDurationMonths = 0
	cfg.Workforce.RampProductivityMultipliers = nil
	cfg.Workforce.BaseTurnoverRateAnnual = 0
	cfg.Workforce.MinTurnoverRateAnnual = 0
	cfg.Workforce.MaxTurnoverRateAnnual = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	scenario, _ := cfg.ScenarioByName("traditional_company")
	input := WorkforceInput{
		Run:                                   1,
		Month:                                 1,
		HeadcountBegin:                        10,
		UnrestrictedCashBegin:                 1e9,
		MarketDemandForecast:                  0,
		ProductiveCapacityRevenueMonthlyBegin: 0,
		ProductivityMultiplier:                1,
	}
	without, err := CalculateWorkforce(cfg, *scenario, input)
	if err != nil {
		t.Fatal(err)
	}
	input.GovernanceAdminEquivalentEmployees = 2
	with, err := CalculateWorkforce(cfg, *scenario, input)
	if err != nil {
		t.Fatal(err)
	}
	assertCloseV04(t, "governance employee reduction", without.EffectiveEmployees-with.EffectiveEmployees, 2)
}

func TestRequiredCashSafeDistribution(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	cfg.CompanyEconomics.StartingCash = 0
	cfg.CompanyEconomics.OpeningAccountsReceivable = 0
	cfg.Market.RevenueCollectionRateCurrentMonth = 0
	cfg.Financing.BaseCreditLine = 100_000_000
	result, err := RunDeterministicScenario(cfg, "profit_sharing", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	month := result.MonthlyResults[0]
	assertCloseV04(t, "cash-safe budget", month.CashSafeAllocationBudget, 0)
	assertCloseV04(t, "cash distribution accrual", month.EmployeeCashDistributionAccrued, 0)
}

func TestRequiredDistributionReserveAtAccrual(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	result, err := RunDeterministicScenario(cfg, "profit_sharing", "no_effect")
	if err != nil {
		t.Fatal(err)
	}
	month := result.MonthlyResults[0]
	want := month.EmployeeCashDistributionAccrued * (1 + cfg.Financing.DistributionPayrollTaxRate)
	if want <= 0 {
		t.Fatal("fixture did not accrue a distribution")
	}
	assertCloseV04(t, "new restricted distribution cash", month.RestrictedDistributionCashNew, want)
	assertCloseV04(t, "restricted distribution close", month.RestrictedDistributionClose, want)
}

func TestRequiredExternalCapitalConstraint(t *testing.T) {
	financing := v04config.Financing{
		ExternalGrowthCapitalLimitMonthly: 1_000,
		ExternalCapitalType:               v04config.ExternalCapitalDebt,
	}
	high := CalculateExternalGrowthCapital(1_000, 0, financing, v04config.OrganizationalScenario{
		ExternalCapitalAccessMultiplier: 1,
	}, 1, 10_000, 0)
	low := CalculateExternalGrowthCapital(1_000, 0, financing, v04config.OrganizationalScenario{
		ExternalCapitalAccessMultiplier: 0.25,
	}, 1, 10_000, 0)
	if !(low.Draw < high.Draw) {
		t.Fatalf("low/high external capital draw = %g/%g, want low < high", low.Draw, high.Draw)
	}
	assertCloseV04(t, "high external capital", high.Draw, 1_000)
	assertCloseV04(t, "low external capital", low.Draw, 250)
}

func removeAllocation(priority []string, remove string) []string {
	result := make([]string, 0, len(priority))
	for _, value := range priority {
		if value != remove {
			result = append(result, value)
		}
	}
	return result
}

func v04RepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
