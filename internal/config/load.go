package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// LoadFile loads, normalizes, and validates a v0.3 configuration file.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := validateNoDuplicateKeys(data); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	if err := validateRequiredJSONFields(data); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	normalize(&cfg)
	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config %q: %w", path, err)
	}
	return cfg, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("multiple JSON values are not allowed")
	}
	return err
}

func normalize(cfg *Config) {
	for i := range cfg.CompensationScenarios {
		scenario := &cfg.CompensationScenarios[i]
		if scenario.Type != "profit_share" {
			continue
		}
		if scenario.ProfitHurdleMonthly == nil {
			value := 0.0
			scenario.ProfitHurdleMonthly = &value
		}
		if scenario.BonusPeriod == "" {
			scenario.BonusPeriod = "monthly"
		}
		if scenario.BonusPayoutLagMonths == nil {
			value := 1
			scenario.BonusPayoutLagMonths = &value
		}
		if scenario.EqualDistribution == nil {
			value := true
			scenario.EqualDistribution = &value
		}
		if scenario.BonusBaseType == "" {
			scenario.BonusBaseType = "distributable_base"
		}
		if scenario.EligibleEmployeesCount == nil {
			value := cfg.Company.EmployeesCount
			scenario.EligibleEmployeesCount = &value
		}
		if scenario.BonusSmoothingReserveRate == nil {
			value := 0.0
			scenario.BonusSmoothingReserveRate = &value
		}
	}
}

func validateRequiredJSONFields(data []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	if err := rejectUnknownKeys("", root,
		"simulation", "company", "cashflow", "workforce", "environment",
		"behavior_cases", "fixed_raise_budget", "compensation_scenarios",
		"environment_cases", "reporting"); err != nil {
		return err
	}
	if err := requireKeys("config", root,
		"simulation", "company", "cashflow", "workforce", "environment",
		"behavior_cases", "fixed_raise_budget", "compensation_scenarios",
		"environment_cases", "reporting"); err != nil {
		return err
	}

	requiredObjects := []struct {
		path string
		keys []string
	}{
		{"simulation", []string{"months", "runs", "random_seed", "common_random_numbers", "currency"}},
		{"company", []string{"employees_count", "base_salary_per_employee", "base_revenue_per_employee", "fixed_costs_monthly", "variable_cost_rate", "starting_cash", "opening_accounts_receivable", "required_cash_reserve_months", "revenue_productivity_elasticity", "demand_cap_multiplier"}},
		{"cashflow", []string{"revenue_collection_rate_current_month", "accounts_receivable_lag_months", "bad_debt_rate", "profit_tax_rate", "profit_tax_payment_lag_months", "reserve_current_profit_tax", "bonus_payroll_tax_rate", "debt_service_monthly", "capex_monthly", "planned_reinvestment_rate", "available_credit_line", "owner_dividend_policy", "reserve_bonus_cash_at_accrual"}},
		{"workforce", []string{"base_turnover_rate_annual", "min_turnover_rate_annual", "max_turnover_rate_annual", "turnover_random_mode", "recruiting_cost_per_leaver", "onboarding_cost_per_leaver", "manager_time_cost_per_leaver", "lost_productivity_cost_per_leaver", "turnover_productivity_penalty_per_annual_turnover", "high_performer_share", "min_productivity_uplift", "max_productivity_uplift"}},
		{"environment", []string{"market_growth_monthly", "market_volatility_monthly", "market_process", "cost_inflation_monthly", "labor_market_factor", "shock_probability_monthly", "shock_revenue_multiplier", "shock_cost_mean", "cash_collection_stress_multiplier"}},
		{"fixed_raise_budget", []string{"mode", "reference_compensation_scenario", "reference_behavior_case", "reference_environment_case", "statistic"}},
		{"reporting", []string{"volatility_penalty_lambda", "small_threshold_for_roi", "print_model_limitations", "print_assumption_flags"}},
	}
	for _, required := range requiredObjects {
		object, err := decodeObject(root[required.path], required.path)
		if err != nil {
			return err
		}
		if err := requireKeys(required.path, object, required.keys...); err != nil {
			return err
		}
		if err := rejectUnknownKeys(required.path, object, required.keys...); err != nil {
			return err
		}
		for _, key := range required.keys {
			if required.path == "company" && key == "demand_cap_multiplier" {
				continue
			}
			if isJSONNull(object[key]) {
				return fieldError(required.path+"."+key, "must not be null")
			}
		}
	}

	cashflow, err := decodeObject(root["cashflow"], "cashflow")
	if err != nil {
		return err
	}
	ownerPolicy, err := decodeObject(cashflow["owner_dividend_policy"], "cashflow.owner_dividend_policy")
	if err != nil {
		return err
	}
	if err := requireKeys("cashflow.owner_dividend_policy", ownerPolicy, "type"); err != nil {
		return err
	}
	if err := rejectUnknownKeys("cashflow.owner_dividend_policy", ownerPolicy, "type"); err != nil {
		return err
	}

	behaviors, err := decodeObject(root["behavior_cases"], "behavior_cases")
	if err != nil {
		return err
	}
	behaviorNames := make([]string, 0, len(behaviors))
	for name := range behaviors {
		behaviorNames = append(behaviorNames, name)
	}
	sort.Strings(behaviorNames)
	for _, name := range behaviorNames {
		raw := behaviors[name]
		behavior, err := decodeObject(raw, "behavior_cases."+name)
		if err != nil {
			return err
		}
		if err := requireKeys("behavior_cases."+name, behavior,
			"productivity_uplift_direct", "turnover_delta_annual_pp",
			"high_perf_attrition_delta_pp", "fairness_penalty_to_quality"); err != nil {
			return err
		}
		if err := rejectUnknownKeys("behavior_cases."+name, behavior,
			"productivity_uplift_direct", "turnover_delta_annual_pp",
			"high_perf_attrition_delta_pp", "fairness_penalty_to_quality"); err != nil {
			return err
		}
		for _, key := range []string{
			"productivity_uplift_direct", "turnover_delta_annual_pp",
			"high_perf_attrition_delta_pp", "fairness_penalty_to_quality",
		} {
			if isJSONNull(behavior[key]) {
				return fieldError("behavior_cases."+name+"."+key, "must not be null")
			}
		}
	}

	var scenarios []map[string]json.RawMessage
	if err := json.Unmarshal(root["compensation_scenarios"], &scenarios); err != nil || scenarios == nil {
		return fieldError("compensation_scenarios", "must be a JSON array")
	}
	for i, scenario := range scenarios {
		prefix := fmt.Sprintf("compensation_scenarios[%d]", i)
		if err := rejectUnknownKeys(prefix, scenario,
			"name", "type", "reference", "profit_share_percent",
			"profit_hurdle_monthly", "bonus_base_type",
			"eligible_employees_count", "bonus_cap_total",
			"bonus_cap_per_employee", "bonus_period",
			"bonus_payout_lag_months", "equal_distribution",
			"bonus_smoothing_reserve_rate"); err != nil {
			return err
		}
		if err := requireKeys(prefix, scenario, "name", "type"); err != nil {
			return err
		}
		var scenarioType string
		if err := json.Unmarshal(scenario["type"], &scenarioType); err != nil {
			return fieldError(prefix+".type", "must be a string")
		}
		switch scenarioType {
		case "fixed_only":
			if _, exists := scenario["reference"]; exists {
				return fieldError(prefix+".reference", "is not valid for fixed_only")
			}
			fallthrough
		case "fixed_raise_same_expected_cost":
			for _, key := range []string{
				"profit_share_percent", "profit_hurdle_monthly",
				"bonus_base_type", "eligible_employees_count",
				"bonus_cap_total", "bonus_cap_per_employee", "bonus_period",
				"bonus_payout_lag_months", "equal_distribution",
				"bonus_smoothing_reserve_rate",
			} {
				if _, exists := scenario[key]; exists {
					return fieldError(prefix+"."+key, "is only valid for profit_share")
				}
			}
		case "profit_share":
			if _, exists := scenario["reference"]; exists {
				return fieldError(prefix+".reference", "is not valid for profit_share")
			}
			for _, key := range []string{"bonus_base_type", "bonus_period"} {
				raw, exists := scenario[key]
				if !exists || isJSONNull(raw) {
					continue
				}
				var value string
				if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
					return fieldError(prefix+"."+key, "must not be empty; omit it to use a defined default")
				}
			}
		}
		for _, key := range []string{
			"name", "type", "reference", "profit_share_percent",
			"profit_hurdle_monthly", "bonus_base_type",
			"eligible_employees_count", "bonus_period",
			"bonus_payout_lag_months", "equal_distribution",
			"bonus_smoothing_reserve_rate",
		} {
			if raw, exists := scenario[key]; exists && isJSONNull(raw) {
				return fieldError(prefix+"."+key, "must not be null; omit it to use a defined default")
			}
		}
		// bonus_cap_total and bonus_cap_per_employee intentionally permit
		// explicit null, the documented representation of an absent cap.
	}
	return nil
}

func validateNoDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeJSONValue(decoder, ""); err != nil {
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]bool)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fieldError(path, "object key must be a string")
			}
			childPath := joinJSONPath(path, key)
			if seen[key] {
				return fieldError(childPath, "duplicate field")
			}
			seen[key] = true
			if err := consumeJSONValue(decoder, childPath); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		index := 0
		for decoder.More() {
			childPath := fmt.Sprintf("%s[%d]", path, index)
			if err := consumeJSONValue(decoder, childPath); err != nil {
				return err
			}
			index++
		}
		_, err = decoder.Token()
		return err
	default:
		return fieldError(path, "unexpected JSON delimiter %q", delimiter)
	}
}

func rejectUnknownKeys(path string, object map[string]json.RawMessage, allowedKeys ...string) error {
	allowed := make(map[string]bool, len(allowedKeys))
	for _, key := range allowedKeys {
		allowed[key] = true
	}
	var unknown []string
	for key := range object {
		if !allowed[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fieldError(joinJSONPath(path, unknown[0]), "unknown field")
}

func joinJSONPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func decodeObject(raw json.RawMessage, path string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fieldError(path, "must be a JSON object")
	}
	if object == nil {
		return nil, fieldError(path, "must be a JSON object")
	}
	return object, nil
}

func requireKeys(path string, object map[string]json.RawMessage, keys ...string) error {
	for _, key := range keys {
		if _, ok := object[key]; !ok {
			return fieldError(path+"."+key, "is required")
		}
	}
	return nil
}
