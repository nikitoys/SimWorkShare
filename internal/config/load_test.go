package config

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadCanonicalDefaultConfig(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if cfg.Simulation.Currency != "RUB" {
		t.Fatalf("currency = %q, want RUB", cfg.Simulation.Currency)
	}
	if cfg.Company.EmployeesCount != 50 {
		t.Fatalf("employees_count = %d, want 50", cfg.Company.EmployeesCount)
	}
	if _, ok := cfg.BehaviorCases["no_effect"]; !ok {
		t.Fatal("no_effect behavior was not loaded")
	}
}

func TestLoadRejectsMissingCanonicalField(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	data = []byte(strings.Replace(string(data), "\n    \"employees_count\": 50,", "", 1))
	missingPath := filepath.Join(t.TempDir(), "missing.json")
	if err := os.WriteFile(missingPath, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err = LoadFile(missingPath)
	if err == nil {
		t.Fatal("LoadFile() error = nil, want missing-field error")
	}
	if !strings.Contains(err.Error(), "company.employees_count") {
		t.Fatalf("LoadFile() error = %q, want missing field path", err)
	}
}

func TestAssumptionWarningsDetectTurnoverDoubleCount(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if warnings := AssumptionWarnings(cfg); len(warnings) != 0 {
		t.Fatalf("default warnings = %v, want none", warnings)
	}

	cfg.Workforce.LostProductivityCostPerLeaver = 1
	warnings := AssumptionWarnings(cfg)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "double count") {
		t.Fatalf("warnings = %v, want turnover double-count warning", warnings)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	data = []byte(strings.Replace(string(data), "\"months\": 60,", "\"unknown\": 1, \"months\": 60,", 1))
	unknownPath := filepath.Join(t.TempDir(), "unknown.json")
	if err := os.WriteFile(unknownPath, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err = LoadFile(unknownPath)
	if err == nil {
		t.Fatal("LoadFile() error = nil, want unknown-field error")
	}
	if !strings.Contains(err.Error(), "simulation.unknown: unknown field") {
		t.Fatalf("LoadFile() error = %q, want full unknown-field path", err)
	}
}

func TestLoadRejectsDuplicateJSONKey(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	data = []byte(strings.Replace(string(data), "\"months\": 60,", "\"months\": 12, \"months\": 60,", 1))
	duplicatePath := filepath.Join(t.TempDir(), "duplicate.json")
	if err := os.WriteFile(duplicatePath, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err = LoadFile(duplicatePath)
	if err == nil || !strings.Contains(err.Error(), "simulation.months: duplicate field") {
		t.Fatalf("LoadFile() error = %v, want duplicate-key path", err)
	}
}

func TestLoadRejectsNullRequiredScalars(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	tests := []struct {
		name      string
		oldValue  string
		newValue  string
		wantField string
	}{
		{"number", "\"fixed_costs_monthly\": 2000000", "\"fixed_costs_monthly\": null", "company.fixed_costs_monthly"},
		{"boolean", "\"common_random_numbers\": true", "\"common_random_numbers\": null", "simulation.common_random_numbers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(strings.Replace(string(original), tt.oldValue, tt.newValue, 1))
			configPath := filepath.Join(t.TempDir(), tt.name+".json")
			if err := os.WriteFile(configPath, data, 0o600); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}

			_, err := LoadFile(configPath)
			if err == nil {
				t.Fatal("LoadFile() error = nil, want null-field error")
			}
			if !strings.Contains(err.Error(), tt.wantField) || !strings.Contains(err.Error(), "must not be null") {
				t.Fatalf("LoadFile() error = %q, want null error for %s", err, tt.wantField)
			}
		})
	}
}

func TestValidateRejectsFieldsFromAnotherPolicyType(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	value := 0.2
	cfg.CompensationScenarios[0].ProfitSharePercent = &value

	err = Validate(cfg)
	if err == nil {
		t.Fatal("Validate() error = nil, want policy-field error")
	}
	if !strings.Contains(err.Error(), "profit_share_percent") || !strings.Contains(err.Error(), "only valid for profit_share") {
		t.Fatalf("Validate() error = %q, want policy-field detail", err)
	}
}

func TestLoadRejectsNullProfitShareFieldOnFixedOnly(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	data := []byte(strings.Replace(string(original), "\"type\": \"fixed_only\"", "\"type\": \"fixed_only\", \"bonus_cap_total\": null", 1))
	configPath := filepath.Join(t.TempDir(), "wrong-policy-null.json")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err = LoadFile(configPath)
	if err == nil || !strings.Contains(err.Error(), "compensation_scenarios[0].bonus_cap_total") {
		t.Fatalf("LoadFile() error = %v, want policy-specific field error", err)
	}
}

func TestLoadRejectsPresentButEmptyPolicyStrings(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	tests := []struct {
		name      string
		oldValue  string
		newValue  string
		wantField string
	}{
		{"fixed-only-reference", "\"type\": \"fixed_only\"", "\"type\": \"fixed_only\", \"reference\": \"\"", "compensation_scenarios[0].reference"},
		{"profit-share-period", "\"bonus_period\": \"monthly\"", "\"bonus_period\": \"\"", "compensation_scenarios[2].bonus_period"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(strings.Replace(string(original), tt.oldValue, tt.newValue, 1))
			configPath := filepath.Join(t.TempDir(), tt.name+".json")
			if err := os.WriteFile(configPath, data, 0o600); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}
			_, err := LoadFile(configPath)
			if err == nil || !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("LoadFile() error = %v, want field %s", err, tt.wantField)
			}
		})
	}
}

func TestProfitSharePolicyDefaultsAreNormalized(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	scenario := cfg.CompensationScenarios[2]
	if scenario.Type != "profit_share" {
		t.Fatalf("scenario type = %q, want profit_share", scenario.Type)
	}
	if scenario.BonusBaseType != "distributable_base" {
		t.Fatalf("bonus_base_type = %q", scenario.BonusBaseType)
	}
	if scenario.EligibleEmployeesCount == nil || *scenario.EligibleEmployeesCount != cfg.Company.EmployeesCount {
		t.Fatalf("eligible_employees_count = %v, want %d", scenario.EligibleEmployeesCount, cfg.Company.EmployeesCount)
	}
	if scenario.BonusSmoothingReserveRate == nil || *scenario.BonusSmoothingReserveRate != 0 {
		t.Fatalf("bonus_smoothing_reserve_rate = %v, want 0", scenario.BonusSmoothingReserveRate)
	}
	if scenario.BonusCapTotal != nil || scenario.BonusCapPerEmployee != nil {
		t.Fatalf("bonus caps = (%v, %v), want (nil, nil)", scenario.BonusCapTotal, scenario.BonusCapPerEmployee)
	}
}

func TestOmittedProfitSharePolicyDefaultsAreNormalized(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(original, &document); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	scenarios := document["compensation_scenarios"].([]any)
	scenario := scenarios[2].(map[string]any)
	delete(scenario, "profit_hurdle_monthly")
	delete(scenario, "bonus_period")
	delete(scenario, "bonus_payout_lag_months")
	delete(scenario, "equal_distribution")
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "omitted-defaults.json")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	normalized := cfg.CompensationScenarios[2]
	if normalized.ProfitHurdleMonthly == nil || *normalized.ProfitHurdleMonthly != 0 {
		t.Fatalf("profit_hurdle_monthly = %v, want 0", normalized.ProfitHurdleMonthly)
	}
	if normalized.BonusPeriod != "monthly" {
		t.Fatalf("bonus_period = %q, want monthly", normalized.BonusPeriod)
	}
	if normalized.BonusPayoutLagMonths == nil || *normalized.BonusPayoutLagMonths != 1 {
		t.Fatalf("bonus_payout_lag_months = %v, want 1", normalized.BonusPayoutLagMonths)
	}
	if normalized.EqualDistribution == nil || !*normalized.EqualDistribution {
		t.Fatalf("equal_distribution = %v, want true", normalized.EqualDistribution)
	}
}

func TestLoadAcceptsExplicitSpecPolicyFields(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	insert := "\"profit_share_percent\": 0.05,\n      \"bonus_base_type\": \"distributable_base\",\n      \"eligible_employees_count\": 40,\n      \"bonus_cap_total\": null,\n      \"bonus_cap_per_employee\": 100000,\n      \"bonus_smoothing_reserve_rate\": 0,"
	data := []byte(strings.Replace(string(original), "\"profit_share_percent\": 0.05,", insert, 1))
	configPath := filepath.Join(t.TempDir(), "policy-fields.json")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	scenario := cfg.CompensationScenarios[2]
	if scenario.EligibleEmployeesCount == nil || *scenario.EligibleEmployeesCount != 40 {
		t.Fatalf("eligible_employees_count = %v, want 40", scenario.EligibleEmployeesCount)
	}
	if scenario.BonusCapTotal != nil {
		t.Fatalf("bonus_cap_total = %v, want nil", scenario.BonusCapTotal)
	}
	if scenario.BonusCapPerEmployee == nil || *scenario.BonusCapPerEmployee != 100000 {
		t.Fatalf("bonus_cap_per_employee = %v, want 100000", scenario.BonusCapPerEmployee)
	}
}

func TestLoadRejectsExplicitNullPolicyScalar(t *testing.T) {
	path := repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	data := []byte(strings.Replace(string(original), "\"bonus_period\": \"monthly\"", "\"bonus_period\": null", 1))
	configPath := filepath.Join(t.TempDir(), "null-policy.json")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err = LoadFile(configPath)
	if err == nil || !strings.Contains(err.Error(), "compensation_scenarios[2].bonus_period") {
		t.Fatalf("LoadFile() error = %v, want explicit-null policy error", err)
	}
}

func TestValidateNoEffectMustBeExactlyZero(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	noEffect := cfg.BehaviorCases["no_effect"]
	noEffect.ProductivityUpliftDirect = 1e-13
	cfg.BehaviorCases["no_effect"] = noEffect

	err = Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "behavior_cases.no_effect") {
		t.Fatalf("Validate() error = %v, want exact-zero error", err)
	}
}

func TestValidateFixedRaiseReferenceMustTargetProfitShare(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.CompensationScenarios[1].Reference = "fixed_only"

	err = Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "must reference a profit_share scenario") {
		t.Fatalf("Validate() error = %v, want reference-type error", err)
	}
}

func TestValidateBonusPayrollTaxIsShare(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.Cashflow.BonusPayrollTaxRate = 1.01

	err = Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "cashflow.bonus_payroll_tax_rate") {
		t.Fatalf("Validate() error = %v, want share-range error", err)
	}
}

func TestValidateRejectsNonFiniteProfitShareInputs(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.CompensationScenarios[2].ProfitSharePercent = floatPointer(math.NaN())
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "profit_share_percent") {
		t.Fatalf("Validate() percent error = %v", err)
	}

	cfg, err = LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.CompensationScenarios[2].ProfitHurdleMonthly = floatPointer(math.Inf(1))
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "profit_hurdle_monthly") {
		t.Fatalf("Validate() hurdle error = %v", err)
	}
}

func floatPointer(value float64) *float64 {
	return &value
}

func TestValidateRejectsTaxLagZeroForCurrentProfile(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.Cashflow.ProfitTaxPaymentLagMonths = 0

	err = Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "profit_tax_payment_lag_months") {
		t.Fatalf("Validate() error = %v, want current-profile lag error", err)
	}
}

func TestValidateRejectsLagMonthOverflow(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.Cashflow.AccountsReceivableLagMonths = int(^uint(0) >> 1)
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "accounts_receivable_lag_months") {
		t.Fatalf("Validate() error = %v, want lag-overflow error", err)
	}

	cfg, err = LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	maxInt := int(^uint(0) >> 1)
	for index := range cfg.CompensationScenarios {
		if cfg.CompensationScenarios[index].Name == "profit_share_equal_10" {
			cfg.CompensationScenarios[index].BonusPayoutLagMonths = &maxInt
			break
		}
	}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "bonus_payout_lag_months") {
		t.Fatalf("Validate() error = %v, want bonus lag-overflow error", err)
	}
}

func TestValidateAllowsZeroDemandCap(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	zero := 0.0
	cfg.Company.DemandCapMultiplier = &zero
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate() error = %v, want zero demand cap to be valid", err)
	}
}

func TestValidateFixedRaiseBudgetReferenceAndStatistic(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.FixedRaiseBudget.ReferenceCompensationScenario = "fixed_only"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "must reference a profit_share scenario") {
		t.Fatalf("Validate() reference error = %v", err)
	}

	cfg, err = LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.FixedRaiseBudget.Statistic = "median"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "fixed_raise_budget.statistic") {
		t.Fatalf("Validate() statistic error = %v", err)
	}
}

func TestValidateReportsFieldPath(t *testing.T) {
	cfg, err := LoadFile(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cfg.Company.EmployeesCount = 0

	err = Validate(cfg)
	if err == nil {
		t.Fatal("Validate() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "company.employees_count") {
		t.Fatalf("Validate() error = %q, want full field path", err)
	}
}

func repositoryPath(t *testing.T, elements ...string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(append([]string{root}, elements...)...)
}
