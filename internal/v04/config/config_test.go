package config

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadCanonicalDefault(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_4.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if cfg.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %q", cfg.SchemaVersion)
	}
	if len(cfg.OrganizationalScenarios) != 4 {
		t.Fatalf("scenario count = %d, want 4", len(cfg.OrganizationalScenarios))
	}
	if warnings := Warnings(cfg); len(warnings) != 0 {
		t.Fatalf("default warnings = %#v, want none", warnings)
	}
	flags := AssumptionFlags(cfg)
	if len(flags) != 3 || flags[0].Code != AssumptionSimplifiedTaxModel ||
		flags[1].Code != AssumptionHighPerformerOnly || flags[2].Code != AssumptionHighPerformerOnly {
		t.Fatalf("assumption flags = %#v", flags)
	}
}

func TestLoadRejectsUnknownFieldWithFullPath(t *testing.T) {
	data := replaceDefault(t, `"trust_index": 0.5`, `"trust_index": 0.5, "mystery": 1`)
	_, err := Load(bytes.NewReader(data))
	assertErrorContains(t, err, "organizational_scenarios[0].governance.mystery: unknown field")
}

func TestLoadRejectsRecursiveDuplicateKeyWithFullPath(t *testing.T) {
	data := replaceDefault(t, `"trust_index": 0.5`, `"trust_index": 0.5, "trust_index": 0.6`)
	_, err := Load(bytes.NewReader(data))
	assertErrorContains(t, err, "organizational_scenarios[0].governance.trust_index: duplicate field")
}

func TestLoadRejectsNaNAndInfinityAtFieldPath(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"nan", "NaN"},
		{"positive-infinity", "Infinity"},
		{"negative-infinity", "-Infinity"},
		{"overflow-to-infinity", "1e9999"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := replaceDefault(t, `"epsilon": 1e-09`, `"epsilon": `+test.value)
			_, err := Load(bytes.NewReader(data))
			assertErrorContains(t, err, "simulation.epsilon")
		})
	}
}

func TestLoadRejectsMissingNullAndWrongTypeWithPaths(t *testing.T) {
	tests := []struct {
		name        string
		old         string
		replacement string
		want        string
	}{
		{"missing", `    "epsilon": 1e-09,` + "\n", "", "simulation.epsilon: is required"},
		{"null", `"market_growth_monthly": 0.003`, `"market_growth_monthly": null`, "market.market_growth_monthly: must not be null"},
		{"wrong-type", `"months": 240`, `"months": "240"`, "simulation.months: must be a JSON integer"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := replaceDefault(t, test.old, test.replacement)
			_, err := Load(bytes.NewReader(data))
			assertErrorContains(t, err, test.want)
		})
	}
}

func TestLoadRejectsTrailingJSONValue(t *testing.T) {
	data := append(defaultData(t), []byte("\n{}")...)
	_, err := Load(bytes.NewReader(data))
	assertErrorContains(t, err, "multiple JSON values")
}

func TestValidateStrictFlagsAndSchemaVersion(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"schema", func(c *Config) { c.SchemaVersion = "0.3" }, "schema_version"},
		{"unknown-fields", func(c *Config) { c.ConfigValidation.RejectUnknownFields = false }, "reject_unknown_fields"},
		{"duplicates", func(c *Config) { c.ConfigValidation.RejectDuplicateFields = false }, "reject_duplicate_fields"},
		{"normalization", func(c *Config) { c.ConfigValidation.AllowNameNormalization = true }, "allow_name_normalization"},
		{"finite", func(c *Config) { c.ConfigValidation.RejectNaNAndInfinity = false }, "reject_nan_and_infinity"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestValidateEnums(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"mode", func(c *Config) { c.Simulation.Mode = "random" }, "simulation.mode"},
		{"headcount", func(c *Config) { c.Simulation.HeadcountMode = "people" }, "simulation.headcount_mode"},
		{"market", func(c *Config) { c.Market.MarketProcess = "normal" }, "market.market_process"},
		{"turnover", func(c *Config) { c.Workforce.TurnoverRandomMode = "poisson" }, "workforce.turnover_random_mode"},
		{"financing", func(c *Config) { c.Financing.ExternalCapitalType = "equity" }, "financing.external_capital_type"},
		{"system", func(c *Config) { c.OrganizationalScenarios[0].SystemType = "unknown" }, "organizational_scenarios[0].system_type"},
		{"distribution", func(c *Config) { c.OrganizationalScenarios[0].DistributionRule = "seniority" }, "organizational_scenarios[0].distribution_rule"},
		{"metric", func(c *Config) { c.Analysis.BreakEvenMetric = "founder_equity" }, "analysis.break_even_metric"},
		{"compatibility", func(c *Config) { c.CompatibilityV03.HeadcountPolicy = "mixed" }, "compatibility_v0_3.headcount_policy"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestValidateHorizonsAndDeterministicRuns(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"greater-than-months", func(c *Config) { c.Simulation.Months = 120 }, "horizons_months[2]"},
		{"duplicate", func(c *Config) { c.Simulation.HorizonsMonths[1] = 60 }, "duplicate horizon"},
		{"empty", func(c *Config) { c.Simulation.HorizonsMonths = nil }, "at least one horizon"},
		{"deterministic-runs", func(c *Config) { c.Simulation.Mode = ModeDeterministic }, "must be 1 in deterministic mode"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestValidateScenarioNamesAndReferences(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"name-pattern", func(c *Config) { c.OrganizationalScenarios[0].Name = "Traditional Company" }, "must match"},
		{"duplicate-name", func(c *Config) { c.OrganizationalScenarios[1].Name = c.OrganizationalScenarios[0].Name }, "duplicate scenario name"},
		{"unknown-behavior", func(c *Config) { c.OrganizationalScenarios[0].BehaviorCaseRefs[0] = "missing" }, "unknown behavior case"},
		{"duplicate-behavior", func(c *Config) {
			c.OrganizationalScenarios[0].BehaviorCaseRefs[1] = c.OrganizationalScenarios[0].BehaviorCaseRefs[0]
		}, "duplicate behavior case reference"},
		{"empty-behavior-list", func(c *Config) { c.OrganizationalScenarios[0].BehaviorCaseRefs = nil }, "at least one behavior case"},
		{"unknown-paired", func(c *Config) { c.Analysis.PairedReferenceScenarios[0] = "missing" }, "unknown scenario"},
		{"duplicate-paired", func(c *Config) { c.Analysis.PairedReferenceScenarios[1] = c.Analysis.PairedReferenceScenarios[0] }, "duplicate scenario reference"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestValidateAllocationSumAndExactPriorityCoverage(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"sum", func(c *Config) { c.OrganizationalScenarios[0].ReinvestmentRate = 0.96 }, "allocation rates sum"},
		{"invalid", func(c *Config) { c.OrganizationalScenarios[0].AllocationPriority[0] = "unknown" }, "invalid allocation id"},
		{"duplicate", func(c *Config) {
			c.OrganizationalScenarios[0].AllocationPriority[1] = c.OrganizationalScenarios[0].AllocationPriority[0]
		}, "duplicate allocation id"},
		{"missing-positive", func(c *Config) {
			c.OrganizationalScenarios[0].AllocationPriority = c.OrganizationalScenarios[0].AllocationPriority[:1]
		}, "missing positive-rate allocation"},
		{"includes-zero", func(c *Config) {
			c.OrganizationalScenarios[0].AllocationPriority = append(c.OrganizationalScenarios[0].AllocationPriority, AllocationEmployeeDistribution)
		}, "has no positive allocation rate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestValidateRampArraysAndOrderedRanges(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"ramp-length", func(c *Config) { c.Workforce.RampDurationMonths = 2 }, "length (3) must equal"},
		{"ramp-value", func(c *Config) { c.Workforce.RampProductivityMultipliers[0] = 1.1 }, "ramp_productivity_multipliers[0]"},
		{"market", func(c *Config) { c.Market.MarketFactorMin = 2 }, "market.market_factor_min"},
		{"turnover", func(c *Config) { c.Workforce.MinTurnoverRateAnnual = 0.3 }, "min_turnover_rate_annual <="},
		{"productivity", func(c *Config) { c.Workforce.MinProductivityUplift = 0.5 }, "min_productivity_uplift"},
		{"decision-quality", func(c *Config) { c.OrganizationalScenarios[0].Governance.DecisionQualityMin = 2 }, "decision_quality_min"},
		{"capability", func(c *Config) { c.OrganizationalScenarios[0].Governance.GovernanceCapabilityIndex = 0 }, "governance_capability_index"},
		{"break-even", func(c *Config) { c.Analysis.BreakEvenUpliftRange = []float64{0.2, 0.1} }, "lower bound"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestValidateSensitivityPaths(t *testing.T) {
	cfg := defaultConfig(t)
	if err := Validate(cfg); err != nil {
		t.Fatalf("default sensitivity paths: %v", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"organizational_scenarios.missing.reinvestment_rate", "unknown scenario"},
		{"behavior_cases.missing.base_productivity_uplift_direct", "unknown name"},
		{"organizational_scenarios.worker_cooperative.name", "must resolve to a numeric field"},
		{"company_economics.missing", "unknown field"},
		{"market.seasonality_multipliers.0", "cannot traverse an array by name"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			mutated := cfg.DeepCopy()
			mutated.Analysis.SensitivityParameters[0].Path = test.path
			assertErrorContains(t, Validate(mutated), test.want)
		})
	}
}

func TestValidateRejectsProgrammaticNonFiniteValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"nan", func(c *Config) { c.CompanyEconomics.StartingCash = math.NaN() }, "company_economics.starting_cash: must be finite"},
		{"infinity", func(c *Config) { c.Analysis.SensitivityParameters[0].Values[0] = math.Inf(1) }, "analysis.sensitivity_parameters[0].values[0]: must be finite"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestWarningsHaveDocumentedDeterministicTriggers(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		code   string
	}{
		{"turnover-double-count", func(c *Config) { c.Workforce.LostProductivityCostPerLeaver = 1 }, WarningTurnoverDoubleCount},
		{"canonical-label", func(c *Config) { c.OrganizationalScenarios[0].SystemType = SystemProfitSharing }, WarningScenarioLabelMismatch},
		{"coop-governance", func(c *Config) { c.OrganizationalScenarios[3].Governance.GovernanceParticipationIntensity = 0 }, WarningScenarioLabelMismatch},
		{"redemption", func(c *Config) { c.Financing.MemberCapitalRedemptionLagMonths = 11 }, WarningHighRedemptionLiquidity},
		{"external-distribution", func(c *Config) { c.OrganizationalScenarios[0].ExternalDistributionRate = 0.01 }, WarningExternalDistribution},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			warnings := Warnings(cfg)
			found := false
			for _, warning := range warnings {
				if warning.Code == test.code {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("warnings = %#v, want code %q", warnings, test.code)
			}
		})
	}
}

func TestLookupAndDeepCopyAreIndependent(t *testing.T) {
	cfg := defaultConfig(t)
	copyConfig := cfg.DeepCopy()
	scenario, ok := copyConfig.ScenarioByName(SystemWorkerCooperative)
	if !ok {
		t.Fatal("worker cooperative scenario not found")
	}
	scenario.AllocationPriority[0] = AllocationExternalDistribution
	behavior, ok := copyConfig.BehaviorByName("no_effect")
	if !ok {
		t.Fatal("no_effect behavior not found")
	}
	behavior.BaseProductivityUpliftDirect = 1
	copyConfig.SetBehavior("no_effect", behavior)
	copyConfig.Analysis.SensitivityParameters[0].Values[0] = 99
	copyConfig.Units["money"] = "changed"

	originalScenario, _ := cfg.ScenarioByName(SystemWorkerCooperative)
	if originalScenario.AllocationPriority[0] == AllocationExternalDistribution {
		t.Fatal("scenario slice aliases deep copy")
	}
	originalBehavior, _ := cfg.BehaviorByName("no_effect")
	if originalBehavior.BaseProductivityUpliftDirect != 0 {
		t.Fatal("behavior map aliases deep copy")
	}
	if cfg.Analysis.SensitivityParameters[0].Values[0] == 99 || cfg.Units["money"] == "changed" {
		t.Fatal("nested analysis/units data aliases deep copy")
	}
}

func TestValidateExecutionCrossFieldRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"positive-epsilon", func(c *Config) { c.Simulation.Epsilon = 0 }, "simulation.epsilon: must be > 0"},
		{"positive-labor-revenue", func(c *Config) { c.CompanyEconomics.BaseRevenuePerEffectiveEmployeeMonthly = 0 }, "company_economics.base_revenue_per_effective_employee_monthly: must be > 0"},
		{"deterministic-binomial", func(c *Config) {
			c.Simulation.Mode = ModeDeterministic
			c.Simulation.Runs = 1
			c.Workforce.TurnoverRandomMode = TurnoverBinomial
		}, "workforce.turnover_random_mode: must be deterministic"},
		{"deterministic-random-headcount", func(c *Config) {
			c.Simulation.Mode = ModeDeterministic
			c.Simulation.Runs = 1
			c.Simulation.HeadcountMode = HeadcountIntegerRandom
		}, "simulation.headcount_mode: must not be integer_random"},
		{"binomial-fractional", func(c *Config) {
			c.Workforce.TurnoverRandomMode = TurnoverBinomial
			c.Simulation.HeadcountMode = HeadcountFractional
		}, "simulation.headcount_mode: must be an integer mode"},
		{"integer-initial-headcount", func(c *Config) {
			c.Simulation.HeadcountMode = HeadcountIntegerExpected
			c.CompanyEconomics.InitialHeadcount = 50.5
		}, "company_economics.initial_headcount: must be an integer"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func TestValidateNoEffectIsSemanticallyNeutral(t *testing.T) {
	cfg := defaultConfig(t)
	behavior := cfg.BehaviorCases["no_effect"]
	behavior.FairnessProductivitySensitivity = 0.1
	cfg.BehaviorCases["no_effect"] = behavior
	assertErrorContains(t, Validate(cfg), "behavior_cases.no_effect.fairness_productivity_sensitivity: must be 0 for no_effect")
}

func TestValidateRejectsUnimplementedCompatibilitySwitches(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"enabled", func(c *Config) { c.CompatibilityV03.Enabled = true }, "compatibility_v0_3.enabled"},
		{"legacy-output", func(c *Config) { c.CompatibilityV03.LegacyOutputsEnabled = true }, "compatibility_v0_3.legacy_outputs_enabled"},
		{"fixed-headcount", func(c *Config) { c.CompatibilityV03.HeadcountPolicy = HeadcountPolicyFixedV03 }, "compatibility_v0_3.headcount_policy"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultConfig(t)
			test.mutate(&cfg)
			assertErrorContains(t, Validate(cfg), test.want)
		})
	}
}

func defaultConfig(t *testing.T) Config {
	t.Helper()
	cfg, err := Load(bytes.NewReader(defaultData(t)))
	if err != nil {
		t.Fatalf("load default config: %v", err)
	}
	return cfg
}

func defaultData(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(repositoryPath(t, "doc", "default_config_v0_4.json"))
	if err != nil {
		t.Fatalf("read default config: %v", err)
	}
	return data
}

func replaceDefault(t *testing.T, old, replacement string) []byte {
	t.Helper()
	data := defaultData(t)
	if !bytes.Contains(data, []byte(old)) {
		t.Fatalf("default config does not contain %q", old)
	}
	return []byte(strings.Replace(string(data), old, replacement, 1))
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want substring %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err, want)
	}
}

func repositoryPath(t *testing.T, elements ...string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	return filepath.Join(append([]string{root}, elements...)...)
}
