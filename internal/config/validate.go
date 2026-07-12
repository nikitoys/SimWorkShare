package config

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Validate checks the complete supplied configuration contract. Features that
// are intentionally outside the deterministic slice may be present in the
// document, but current-stage execution settings must remain deterministic.
func Validate(cfg Config) error {
	if cfg.Simulation.Months <= 0 {
		return fieldError("simulation.months", "must be > 0")
	}
	if cfg.Simulation.Runs <= 0 {
		return fieldError("simulation.runs", "must be > 0")
	}
	if strings.TrimSpace(cfg.Simulation.Currency) == "" {
		return fieldError("simulation.currency", "must not be empty")
	}

	if cfg.Company.EmployeesCount <= 0 {
		return fieldError("company.employees_count", "must be > 0")
	}
	if err := nonNegative("company.base_salary_per_employee", cfg.Company.BaseSalaryPerEmployee); err != nil {
		return err
	}
	if err := nonNegative("company.base_revenue_per_employee", cfg.Company.BaseRevenuePerEmployee); err != nil {
		return err
	}
	if err := nonNegative("company.fixed_costs_monthly", cfg.Company.FixedCostsMonthly); err != nil {
		return err
	}
	if err := rate("company.variable_cost_rate", cfg.Company.VariableCostRate); err != nil {
		return err
	}
	if err := finite("company.starting_cash", cfg.Company.StartingCash); err != nil {
		return err
	}
	if err := nonNegative("company.opening_accounts_receivable", cfg.Company.OpeningAccountsReceivable); err != nil {
		return err
	}
	if err := nonNegative("company.required_cash_reserve_months", cfg.Company.RequiredCashReserveMonths); err != nil {
		return err
	}
	if err := nonNegative("company.revenue_productivity_elasticity", cfg.Company.RevenueProductivityElasticity); err != nil {
		return err
	}
	if cfg.Company.DemandCapMultiplier != nil {
		if err := nonNegative("company.demand_cap_multiplier", *cfg.Company.DemandCapMultiplier); err != nil {
			return err
		}
	}

	if err := rate("cashflow.revenue_collection_rate_current_month", cfg.Cashflow.RevenueCollectionRateCurrentMonth); err != nil {
		return err
	}
	if cfg.Cashflow.AccountsReceivableLagMonths < 1 {
		return fieldError("cashflow.accounts_receivable_lag_months", "must be >= 1 in v1")
	}
	maxInt := int(^uint(0) >> 1)
	if cfg.Cashflow.AccountsReceivableLagMonths > maxInt-cfg.Simulation.Months {
		return fieldError("cashflow.accounts_receivable_lag_months", "is too large for the simulation horizon")
	}
	if err := rate("cashflow.bad_debt_rate", cfg.Cashflow.BadDebtRate); err != nil {
		return err
	}
	if err := rate("cashflow.profit_tax_rate", cfg.Cashflow.ProfitTaxRate); err != nil {
		return err
	}
	if cfg.Cashflow.ProfitTaxPaymentLagMonths < 1 {
		return fieldError("cashflow.profit_tax_payment_lag_months", "must be >= 1 in the current deterministic stage")
	}
	if cfg.Cashflow.ProfitTaxPaymentLagMonths > maxInt-cfg.Simulation.Months {
		return fieldError("cashflow.profit_tax_payment_lag_months", "is too large for the simulation horizon")
	}
	if err := rate("cashflow.bonus_payroll_tax_rate", cfg.Cashflow.BonusPayrollTaxRate); err != nil {
		return err
	}
	if err := nonNegative("cashflow.debt_service_monthly", cfg.Cashflow.DebtServiceMonthly); err != nil {
		return err
	}
	if err := nonNegative("cashflow.capex_monthly", cfg.Cashflow.CapexMonthly); err != nil {
		return err
	}
	if err := rate("cashflow.planned_reinvestment_rate", cfg.Cashflow.PlannedReinvestmentRate); err != nil {
		return err
	}
	if err := nonNegative("cashflow.available_credit_line", cfg.Cashflow.AvailableCreditLine); err != nil {
		return err
	}
	if cfg.Cashflow.OwnerDividendPolicy.Type != "none" {
		return fieldError("cashflow.owner_dividend_policy.type", "only %q is supported by the deterministic slice", "none")
	}
	if !cfg.Cashflow.ReserveBonusCashAtAccrual {
		return fieldError("cashflow.reserve_bonus_cash_at_accrual", "must be true in v1")
	}

	if err := validateWorkforce(cfg.Workforce); err != nil {
		return err
	}
	if err := validateEnvironment(cfg.Environment); err != nil {
		return err
	}
	if err := validateBehaviors(cfg); err != nil {
		return err
	}
	if err := validateScenarios(cfg); err != nil {
		return err
	}
	if err := validateEnvironmentCases(cfg.EnvironmentCases); err != nil {
		return err
	}
	if err := validateFixedRaiseReference(cfg); err != nil {
		return err
	}

	if err := nonNegative("reporting.volatility_penalty_lambda", cfg.Reporting.VolatilityPenaltyLambda); err != nil {
		return err
	}
	if err := positive("reporting.small_threshold_for_roi", cfg.Reporting.SmallThresholdForROI); err != nil {
		return err
	}
	return nil
}

func validateWorkforce(cfg WorkforceConfig) error {
	for _, item := range []struct {
		name  string
		value float64
	}{
		{"workforce.base_turnover_rate_annual", cfg.BaseTurnoverRateAnnual},
		{"workforce.min_turnover_rate_annual", cfg.MinTurnoverRateAnnual},
		{"workforce.max_turnover_rate_annual", cfg.MaxTurnoverRateAnnual},
		{"workforce.high_performer_share", cfg.HighPerformerShare},
	} {
		if err := rate(item.name, item.value); err != nil {
			return err
		}
	}
	if cfg.MinTurnoverRateAnnual > cfg.BaseTurnoverRateAnnual || cfg.BaseTurnoverRateAnnual > cfg.MaxTurnoverRateAnnual {
		return fieldError("workforce.base_turnover_rate_annual", "must be within min_turnover_rate_annual and max_turnover_rate_annual")
	}
	if cfg.TurnoverRandomMode != "deterministic" {
		return fieldError("workforce.turnover_random_mode", "only %q is supported by the deterministic slice", "deterministic")
	}
	for _, item := range []struct {
		name  string
		value float64
	}{
		{"workforce.recruiting_cost_per_leaver", cfg.RecruitingCostPerLeaver},
		{"workforce.onboarding_cost_per_leaver", cfg.OnboardingCostPerLeaver},
		{"workforce.manager_time_cost_per_leaver", cfg.ManagerTimeCostPerLeaver},
		{"workforce.lost_productivity_cost_per_leaver", cfg.LostProductivityCostPerLeaver},
		{"workforce.turnover_productivity_penalty_per_annual_turnover", cfg.TurnoverProductivityPenaltyPerAnnualTurnover},
	} {
		if err := nonNegative(item.name, item.value); err != nil {
			return err
		}
	}
	if err := finite("workforce.min_productivity_uplift", cfg.MinProductivityUplift); err != nil {
		return err
	}
	if err := finite("workforce.max_productivity_uplift", cfg.MaxProductivityUplift); err != nil {
		return err
	}
	if cfg.MinProductivityUplift < -1 {
		return fieldError("workforce.min_productivity_uplift", "must be >= -1")
	}
	if cfg.MinProductivityUplift > cfg.MaxProductivityUplift {
		return fieldError("workforce.min_productivity_uplift", "must be <= max_productivity_uplift")
	}
	return nil
}

func validateEnvironment(cfg EnvironmentConfig) error {
	if err := finite("environment.market_growth_monthly", cfg.MarketGrowthMonthly); err != nil {
		return err
	}
	if cfg.MarketGrowthMonthly <= -1 {
		return fieldError("environment.market_growth_monthly", "must be > -1")
	}
	if err := nonNegative("environment.market_volatility_monthly", cfg.MarketVolatilityMonthly); err != nil {
		return err
	}
	if cfg.MarketProcess != "bounded_lognormal" {
		return fieldError("environment.market_process", "unsupported value %q", cfg.MarketProcess)
	}
	if err := finite("environment.cost_inflation_monthly", cfg.CostInflationMonthly); err != nil {
		return err
	}
	if cfg.CostInflationMonthly <= -1 {
		return fieldError("environment.cost_inflation_monthly", "must be > -1")
	}
	if err := nonNegative("environment.labor_market_factor", cfg.LaborMarketFactor); err != nil {
		return err
	}
	if err := rate("environment.shock_probability_monthly", cfg.ShockProbabilityMonthly); err != nil {
		return err
	}
	if err := nonNegative("environment.shock_revenue_multiplier", cfg.ShockRevenueMultiplier); err != nil {
		return err
	}
	if err := nonNegative("environment.shock_cost_mean", cfg.ShockCostMean); err != nil {
		return err
	}
	if err := rate("environment.cash_collection_stress_multiplier", cfg.CashCollectionStressMultiplier); err != nil {
		return err
	}
	return nil
}

func validateBehaviors(cfg Config) error {
	if len(cfg.BehaviorCases) == 0 {
		return fieldError("behavior_cases", "must not be empty")
	}
	keys := make([]string, 0, len(cfg.BehaviorCases))
	for key := range cfg.BehaviorCases {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		behavior := cfg.BehaviorCases[key]
		prefix := "behavior_cases." + key
		for _, item := range []struct {
			name  string
			value float64
		}{
			{"productivity_uplift_direct", behavior.ProductivityUpliftDirect},
			{"turnover_delta_annual_pp", behavior.TurnoverDeltaAnnualPP},
			{"high_perf_attrition_delta_pp", behavior.HighPerfAttritionDeltaPP},
			{"fairness_penalty_to_quality", behavior.FairnessPenaltyToQuality},
		} {
			if err := finite(prefix+"."+item.name, item.value); err != nil {
				return err
			}
		}
	}

	noEffect, ok := cfg.BehaviorCases["no_effect"]
	if !ok {
		return fieldError("behavior_cases.no_effect", "is required")
	}
	if !isZero(noEffect.ProductivityUpliftDirect) || !isZero(noEffect.TurnoverDeltaAnnualPP) ||
		!isZero(noEffect.HighPerfAttritionDeltaPP) || !isZero(noEffect.FairnessPenaltyToQuality) {
		return fieldError("behavior_cases.no_effect", "all effects must be zero")
	}
	return nil
}

func validateScenarios(cfg Config) error {
	if len(cfg.CompensationScenarios) == 0 {
		return fieldError("compensation_scenarios", "must not be empty")
	}
	names := make(map[string]string, len(cfg.CompensationScenarios))
	foundFixedOnly := false
	for i, scenario := range cfg.CompensationScenarios {
		prefix := fmt.Sprintf("compensation_scenarios[%d]", i)
		if strings.TrimSpace(scenario.Name) == "" {
			return fieldError(prefix+".name", "must not be empty")
		}
		if _, exists := names[scenario.Name]; exists {
			return fieldError(prefix+".name", "duplicate scenario %q", scenario.Name)
		}
		names[scenario.Name] = scenario.Type

		switch scenario.Type {
		case "fixed_only":
			if scenario.Name != "fixed_only" {
				return fieldError(prefix+".name", "fixed_only policy must use canonical name %q", "fixed_only")
			}
			if scenario.Reference != "" {
				return fieldError(prefix+".reference", "is not valid for fixed_only")
			}
			if err := rejectProfitShareFields(prefix, scenario); err != nil {
				return err
			}
			if scenario.Name == "fixed_only" {
				foundFixedOnly = true
			}
		case "fixed_raise_same_expected_cost":
			if err := rejectProfitShareFields(prefix, scenario); err != nil {
				return err
			}
			if strings.TrimSpace(scenario.Reference) == "" {
				return fieldError(prefix+".reference", "is required")
			}
		case "profit_share":
			if scenario.Reference != "" {
				return fieldError(prefix+".reference", "is not valid for profit_share")
			}
			if scenario.ProfitSharePercent == nil {
				return fieldError(prefix+".profit_share_percent", "is required")
			}
			if err := finite(prefix+".profit_share_percent", *scenario.ProfitSharePercent); err != nil {
				return err
			}
			if *scenario.ProfitSharePercent < 0 || *scenario.ProfitSharePercent > 0.30 {
				return fieldError(prefix+".profit_share_percent", "must be between 0 and 0.30")
			}
			if scenario.ProfitHurdleMonthly == nil {
				return fieldError(prefix+".profit_hurdle_monthly", "must be >= 0")
			}
			if err := nonNegative(prefix+".profit_hurdle_monthly", *scenario.ProfitHurdleMonthly); err != nil {
				return err
			}
			if *scenario.ProfitHurdleMonthly < 0 {
				return fieldError(prefix+".profit_hurdle_monthly", "must be >= 0")
			}
			if scenario.BonusBaseType != "distributable_base" {
				return fieldError(prefix+".bonus_base_type", "must be %q in v1", "distributable_base")
			}
			if scenario.EligibleEmployeesCount == nil || *scenario.EligibleEmployeesCount <= 0 || *scenario.EligibleEmployeesCount > cfg.Company.EmployeesCount {
				return fieldError(prefix+".eligible_employees_count", "must be between 1 and company.employees_count")
			}
			if scenario.BonusCapTotal != nil {
				if err := nonNegative(prefix+".bonus_cap_total", *scenario.BonusCapTotal); err != nil {
					return err
				}
			}
			if scenario.BonusCapPerEmployee != nil {
				if err := nonNegative(prefix+".bonus_cap_per_employee", *scenario.BonusCapPerEmployee); err != nil {
					return err
				}
			}
			if scenario.BonusPeriod != "monthly" && scenario.BonusPeriod != "quarterly" && scenario.BonusPeriod != "annual" {
				return fieldError(prefix+".bonus_period", "unsupported value %q", scenario.BonusPeriod)
			}
			if scenario.BonusPayoutLagMonths == nil || *scenario.BonusPayoutLagMonths < 0 {
				return fieldError(prefix+".bonus_payout_lag_months", "must be >= 0")
			}
			maxInt := int(^uint(0) >> 1)
			if *scenario.BonusPayoutLagMonths > maxInt-cfg.Simulation.Months {
				return fieldError(prefix+".bonus_payout_lag_months", "is too large for the simulation horizon")
			}
			if scenario.EqualDistribution == nil || !*scenario.EqualDistribution {
				return fieldError(prefix+".equal_distribution", "must be true in v1")
			}
			if scenario.BonusSmoothingReserveRate == nil || *scenario.BonusSmoothingReserveRate != 0 {
				return fieldError(prefix+".bonus_smoothing_reserve_rate", "must be 0 in v1")
			}
		default:
			return fieldError(prefix+".type", "unsupported value %q", scenario.Type)
		}
	}
	if !foundFixedOnly {
		return fieldError("compensation_scenarios", "must contain scenario named %q with type fixed_only", "fixed_only")
	}
	for i, scenario := range cfg.CompensationScenarios {
		if scenario.Type == "fixed_raise_same_expected_cost" {
			targetType, ok := names[scenario.Reference]
			if !ok {
				return fieldError(fmt.Sprintf("compensation_scenarios[%d].reference", i), "unknown scenario %q", scenario.Reference)
			}
			if targetType != "profit_share" {
				return fieldError(fmt.Sprintf("compensation_scenarios[%d].reference", i), "must reference a profit_share scenario")
			}
		}
	}
	return nil
}

func rejectProfitShareFields(prefix string, scenario CompensationScenario) error {
	if scenario.ProfitSharePercent != nil {
		return fieldError(prefix+".profit_share_percent", "is only valid for profit_share")
	}
	if scenario.ProfitHurdleMonthly != nil {
		return fieldError(prefix+".profit_hurdle_monthly", "is only valid for profit_share")
	}
	if scenario.BonusBaseType != "" {
		return fieldError(prefix+".bonus_base_type", "is only valid for profit_share")
	}
	if scenario.EligibleEmployeesCount != nil {
		return fieldError(prefix+".eligible_employees_count", "is only valid for profit_share")
	}
	if scenario.BonusCapTotal != nil {
		return fieldError(prefix+".bonus_cap_total", "is only valid for profit_share")
	}
	if scenario.BonusCapPerEmployee != nil {
		return fieldError(prefix+".bonus_cap_per_employee", "is only valid for profit_share")
	}
	if scenario.BonusPeriod != "" {
		return fieldError(prefix+".bonus_period", "is only valid for profit_share")
	}
	if scenario.BonusPayoutLagMonths != nil {
		return fieldError(prefix+".bonus_payout_lag_months", "is only valid for profit_share")
	}
	if scenario.EqualDistribution != nil {
		return fieldError(prefix+".equal_distribution", "is only valid for profit_share")
	}
	if scenario.BonusSmoothingReserveRate != nil {
		return fieldError(prefix+".bonus_smoothing_reserve_rate", "is only valid for profit_share")
	}
	return nil
}

func validateEnvironmentCases(cases []string) error {
	if len(cases) == 0 {
		return fieldError("environment_cases", "must not be empty")
	}
	allowed := map[string]bool{
		"normal_market": true,
		"flat_market":   true,
		"downturn":      true,
		"shock":         true,
		"labor_tight":   true,
	}
	seen := make(map[string]bool, len(cases))
	for i, name := range cases {
		if !allowed[name] {
			return fieldError(fmt.Sprintf("environment_cases[%d]", i), "unsupported value %q", name)
		}
		if seen[name] {
			return fieldError(fmt.Sprintf("environment_cases[%d]", i), "duplicate value %q", name)
		}
		seen[name] = true
	}
	if !seen["normal_market"] {
		return fieldError("environment_cases", "must contain %q", "normal_market")
	}
	return nil
}

func validateFixedRaiseReference(cfg Config) error {
	reference := cfg.FixedRaiseBudget
	if reference.Mode != "prepass_reference" {
		return fieldError("fixed_raise_budget.mode", "only %q is supported by the supplied v0.3 config", "prepass_reference")
	}
	foundScenario := false
	referenceScenarioType := ""
	for _, scenario := range cfg.CompensationScenarios {
		if scenario.Name == reference.ReferenceCompensationScenario {
			foundScenario = true
			referenceScenarioType = scenario.Type
			break
		}
	}
	if !foundScenario {
		return fieldError("fixed_raise_budget.reference_compensation_scenario", "unknown scenario %q", reference.ReferenceCompensationScenario)
	}
	if referenceScenarioType != "profit_share" {
		return fieldError("fixed_raise_budget.reference_compensation_scenario", "must reference a profit_share scenario")
	}
	if _, ok := cfg.BehaviorCases[reference.ReferenceBehaviorCase]; !ok {
		return fieldError("fixed_raise_budget.reference_behavior_case", "unknown behavior case %q", reference.ReferenceBehaviorCase)
	}
	foundEnvironment := false
	for _, name := range cfg.EnvironmentCases {
		if name == reference.ReferenceEnvironmentCase {
			foundEnvironment = true
			break
		}
	}
	if !foundEnvironment {
		return fieldError("fixed_raise_budget.reference_environment_case", "unknown environment case %q", reference.ReferenceEnvironmentCase)
	}
	if reference.Statistic != "mean_total_employer_bonus_cost_per_employee_per_month" {
		return fieldError("fixed_raise_budget.statistic", "unsupported value %q", reference.Statistic)
	}
	return nil
}

func finite(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fieldError(name, "must be finite")
	}
	return nil
}

func nonNegative(name string, value float64) error {
	if err := finite(name, value); err != nil {
		return err
	}
	if value < 0 {
		return fieldError(name, "must be >= 0")
	}
	return nil
}

func positive(name string, value float64) error {
	if err := finite(name, value); err != nil {
		return err
	}
	if value <= 0 {
		return fieldError(name, "must be > 0")
	}
	return nil
}

func rate(name string, value float64) error {
	if err := finite(name, value); err != nil {
		return err
	}
	if value < 0 || value > 1 {
		return fieldError(name, "must be between 0 and 1")
	}
	return nil
}

func isZero(value float64) bool {
	return value == 0
}

func fieldError(field, format string, args ...any) error {
	return fmt.Errorf("%s: %s", field, fmt.Sprintf(format, args...))
}
