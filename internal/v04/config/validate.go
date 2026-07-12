package config

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var scenarioNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Validate applies the JSON Schema and parameter-catalog ranges plus the
// cross-field rules in section 19 of the v0.4 specification.
func Validate(cfg Config) error {
	if err := validateFiniteNumbers(reflect.ValueOf(cfg), reflect.TypeOf(cfg), ""); err != nil {
		return err
	}
	if cfg.SchemaVersion != SchemaVersion {
		return fieldError("schema_version", "must be %q", SchemaVersion)
	}
	if !cfg.ConfigValidation.RejectUnknownFields {
		return fieldError("config_validation.reject_unknown_fields", "must be true for strict v0.4 validation")
	}
	if !cfg.ConfigValidation.RejectDuplicateFields {
		return fieldError("config_validation.reject_duplicate_fields", "must be true for strict v0.4 validation")
	}
	if cfg.ConfigValidation.AllowNameNormalization {
		return fieldError("config_validation.allow_name_normalization", "must be false for strict v0.4 validation")
	}
	if !cfg.ConfigValidation.RejectNaNAndInfinity {
		return fieldError("config_validation.reject_nan_and_infinity", "must be true for strict v0.4 validation")
	}

	if err := validateSimulation(cfg.Simulation); err != nil {
		return err
	}
	if err := validateCompanyEconomics(cfg.CompanyEconomics); err != nil {
		return err
	}
	if err := validateMarket(cfg.Market); err != nil {
		return err
	}
	if err := validateWorkforce(cfg.Workforce); err != nil {
		return err
	}
	if err := validateSimulationWorkforceCompatibility(cfg); err != nil {
		return err
	}
	if err := validateEmployeeRisk(cfg.EmployeeRisk); err != nil {
		return err
	}
	if err := validateFinancing(cfg.Financing); err != nil {
		return err
	}
	if err := validateBehaviorCases(cfg.BehaviorCases); err != nil {
		return err
	}
	scenarioNames, err := validateScenarios(cfg.OrganizationalScenarios, cfg.BehaviorCases)
	if err != nil {
		return err
	}
	if err := validateAnalysis(cfg, scenarioNames); err != nil {
		return err
	}
	if err := validateCompatibilityV03(cfg.CompatibilityV03); err != nil {
		return err
	}
	return nil
}

func validateSimulation(sim Simulation) error {
	if err := oneOf("simulation.mode", sim.Mode, ModeDeterministic, ModeMonteCarlo); err != nil {
		return err
	}
	if err := intBetween("simulation.months", sim.Months, 1, 240); err != nil {
		return err
	}
	if len(sim.HorizonsMonths) == 0 {
		return fieldError("simulation.horizons_months", "must contain at least one horizon")
	}
	seen := make(map[int]struct{}, len(sim.HorizonsMonths))
	for index, horizon := range sim.HorizonsMonths {
		path := fmt.Sprintf("simulation.horizons_months[%d]", index)
		if err := intBetween(path, horizon, 1, 240); err != nil {
			return err
		}
		if horizon > sim.Months {
			return fieldError(path, "must be <= simulation.months (%d)", sim.Months)
		}
		if _, exists := seen[horizon]; exists {
			return fieldError(path, "duplicate horizon %d", horizon)
		}
		seen[horizon] = struct{}{}
	}
	if err := intBetween("simulation.runs", sim.Runs, 1, 1_000_000); err != nil {
		return err
	}
	if sim.Mode == ModeDeterministic && sim.Runs != 1 {
		return fieldError("simulation.runs", "must be 1 in deterministic mode")
	}
	if sim.RandomSeed < 0 {
		return fieldError("simulation.random_seed", "must be >= 0")
	}
	if strings.TrimSpace(sim.Currency) == "" {
		return fieldError("simulation.currency", "must not be empty")
	}
	if err := positive("simulation.epsilon", sim.Epsilon); err != nil {
		return err
	}
	if err := between("simulation.epsilon", sim.Epsilon, 0, 1e-3); err != nil {
		return err
	}
	return oneOf("simulation.headcount_mode", sim.HeadcountMode,
		HeadcountFractional, HeadcountIntegerExpected, HeadcountIntegerRandom)
}

func validateCompanyEconomics(company CompanyEconomics) error {
	if err := positive("company_economics.initial_headcount", company.InitialHeadcount); err != nil {
		return err
	}
	if err := positive("company_economics.base_revenue_per_effective_employee_monthly", company.BaseRevenuePerEffectiveEmployeeMonthly); err != nil {
		return err
	}
	for _, value := range []namedNumber{
		{"company_economics.base_salary_per_employee_monthly", company.BaseSalaryPerEmployeeMonthly},
		{"company_economics.initial_market_demand_monthly", company.InitialMarketDemandMonthly},
		{"company_economics.initial_productive_capacity_revenue_monthly", company.InitialProductiveCapacityRevenueMonthly},
		{"company_economics.fixed_costs_monthly", company.FixedCostsMonthly},
		{"company_economics.capacity_revenue_created_per_currency_invested", company.CapacityRevenueCreatedPerCurrencyInvested},
		{"company_economics.starting_cash", company.StartingCash},
		{"company_economics.opening_accounts_receivable", company.OpeningAccountsReceivable},
	} {
		if err := nonNegative(value.path, value.value); err != nil {
			return err
		}
	}
	for _, value := range []namedNumber{
		{"company_economics.salary_payroll_tax_rate", company.SalaryPayrollTaxRate},
		{"company_economics.variable_cost_rate", company.VariableCostRate},
		{"company_economics.profit_tax_rate", company.ProfitTaxRate},
	} {
		if err := rate(value.path, value.value); err != nil {
			return err
		}
	}
	if err := between("company_economics.standard_hours_per_employee_month", company.StandardHoursPerEmployeeMonth, 1, 300); err != nil {
		return err
	}
	if err := intBetween("company_economics.profit_tax_payment_lag_months", company.ProfitTaxPaymentLagMonths, 0, 24); err != nil {
		return err
	}
	if err := between("company_economics.cost_inflation_monthly", company.CostInflationMonthly, -0.05, 0.10); err != nil {
		return err
	}
	if err := between("company_economics.capacity_depreciation_rate_monthly", company.CapacityDepreciationRateMonthly, 0, 0.10); err != nil {
		return err
	}
	if err := intBetween("company_economics.investment_activation_lag_months", company.InvestmentActivationLagMonths, 0, 60); err != nil {
		return err
	}
	return between("company_economics.required_cash_reserve_months", company.RequiredCashReserveMonths, 0, 24)
}

func validateMarket(market Market) error {
	if err := oneOf("market.market_process", market.MarketProcess,
		MarketProcessDeterministic, MarketProcessBoundedLognormal); err != nil {
		return err
	}
	for _, value := range []struct {
		path    string
		value   float64
		minimum float64
		maximum float64
	}{
		{"market.market_growth_monthly", market.MarketGrowthMonthly, -0.10, 0.10},
		{"market.market_volatility_monthly", market.MarketVolatilityMonthly, 0, 1},
		{"market.market_factor_min", market.MarketFactorMin, 0, 10},
		{"market.market_factor_max", market.MarketFactorMax, 0, 10},
		{"market.revenue_collection_rate_current_month", market.RevenueCollectionRateCurrentMonth, 0, 1},
		{"market.bad_debt_rate", market.BadDebtRate, 0, 1},
		{"market.shock_probability_monthly", market.ShockProbabilityMonthly, 0, 1},
		{"market.shock_revenue_multiplier", market.ShockRevenueMultiplier, 0, 1},
		{"market.cash_collection_stress_multiplier", market.CashCollectionStressMultiplier, 0, 1},
		{"market.labor_market_factor", market.LaborMarketFactor, 0, 5},
		{"market.credit_market_factor", market.CreditMarketFactor, 0, 5},
	} {
		if err := between(value.path, value.value, value.minimum, value.maximum); err != nil {
			return err
		}
	}
	if market.MarketFactorMin > market.MarketFactorMax {
		return fieldError("market.market_factor_min", "must be <= market.market_factor_max")
	}
	if len(market.SeasonalityMultipliers) != 12 {
		return fieldError("market.seasonality_multipliers", "must contain exactly 12 values")
	}
	for index, multiplier := range market.SeasonalityMultipliers {
		if err := between(fmt.Sprintf("market.seasonality_multipliers[%d]", index), multiplier, 0, 10); err != nil {
			return err
		}
	}
	if err := intBetween("market.accounts_receivable_lag_months", market.AccountsReceivableLagMonths, 0, 24); err != nil {
		return err
	}
	if err := nonNegative("market.shock_cost_mean", market.ShockCostMean); err != nil {
		return err
	}
	return nonNegative("market.shock_cost_std", market.ShockCostStd)
}

func validateWorkforce(workforce Workforce) error {
	for _, value := range []namedNumber{
		{"workforce.base_turnover_rate_annual", workforce.BaseTurnoverRateAnnual},
		{"workforce.min_turnover_rate_annual", workforce.MinTurnoverRateAnnual},
		{"workforce.max_turnover_rate_annual", workforce.MaxTurnoverRateAnnual},
		{"workforce.high_performer_share", workforce.HighPerformerShare},
		{"workforce.max_hires_per_month_rate", workforce.MaxHiresPerMonthRate},
		{"workforce.max_layoffs_per_month_rate", workforce.MaxLayoffsPerMonthRate},
		{"workforce.leaver_paid_fraction_of_month", workforce.LeaverPaidFractionOfMonth},
		{"workforce.new_hire_paid_fraction_of_month", workforce.NewHirePaidFractionOfMonth},
		{"workforce.max_cash_share_for_hiring", workforce.MaxCashShareForHiring},
	} {
		if err := rate(value.path, value.value); err != nil {
			return err
		}
	}
	if !(workforce.MinTurnoverRateAnnual <= workforce.BaseTurnoverRateAnnual &&
		workforce.BaseTurnoverRateAnnual <= workforce.MaxTurnoverRateAnnual) {
		return fieldError("workforce.base_turnover_rate_annual",
			"must satisfy min_turnover_rate_annual <= base_turnover_rate_annual <= max_turnover_rate_annual")
	}
	if err := oneOf("workforce.turnover_random_mode", workforce.TurnoverRandomMode,
		TurnoverDeterministic, TurnoverBinomial); err != nil {
		return err
	}
	for _, value := range []namedNumber{
		{"workforce.recruiting_cost_per_hire", workforce.RecruitingCostPerHire},
		{"workforce.onboarding_cost_per_hire", workforce.OnboardingCostPerHire},
		{"workforce.manager_time_cost_per_hire", workforce.ManagerTimeCostPerHire},
		{"workforce.exit_admin_cost_per_leaver", workforce.ExitAdminCostPerLeaver},
		{"workforce.lost_productivity_cost_per_leaver", workforce.LostProductivityCostPerLeaver},
		{"workforce.severance_cost_per_layoff", workforce.SeveranceCostPerLayoff},
	} {
		if err := nonNegative(value.path, value.value); err != nil {
			return err
		}
	}
	if err := intBetween("workforce.ramp_duration_months", workforce.RampDurationMonths, 0, 24); err != nil {
		return err
	}
	if workforce.RampDurationMonths > 0 && len(workforce.RampProductivityMultipliers) != workforce.RampDurationMonths {
		return fieldError("workforce.ramp_productivity_multipliers",
			"length (%d) must equal ramp_duration_months (%d)",
			len(workforce.RampProductivityMultipliers), workforce.RampDurationMonths)
	}
	for index, multiplier := range workforce.RampProductivityMultipliers {
		if err := rate(fmt.Sprintf("workforce.ramp_productivity_multipliers[%d]", index), multiplier); err != nil {
			return err
		}
	}
	if err := between("workforce.layoff_trigger_cash_ratio", workforce.LayoffTriggerCashRatio, 0, 10); err != nil {
		return err
	}
	if err := between("workforce.turnover_productivity_penalty_per_annual_turnover", workforce.TurnoverProductivityPenaltyPerAnnualTurnover, 0, 10); err != nil {
		return err
	}
	if err := between("workforce.min_productivity_uplift", workforce.MinProductivityUplift, -1, 1); err != nil {
		return err
	}
	if err := between("workforce.max_productivity_uplift", workforce.MaxProductivityUplift, -1, 1); err != nil {
		return err
	}
	if workforce.MinProductivityUplift > workforce.MaxProductivityUplift {
		return fieldError("workforce.min_productivity_uplift", "must be <= workforce.max_productivity_uplift")
	}
	if err := between("workforce.target_staffing_buffer", workforce.TargetStaffingBuffer, 0, 3); err != nil {
		return err
	}
	return nil
}

func validateEmployeeRisk(riskConfig EmployeeRisk) error {
	if err := nonNegative("employee_risk.employee_external_savings_proxy_per_employee", riskConfig.EmployeeExternalSavingsProxyPerEmployee); err != nil {
		return err
	}
	for _, value := range []namedNumber{
		{"employee_risk.employment_dependence_index", riskConfig.EmploymentDependenceIndex},
		{"employee_risk.risk_weight_variable_income", riskConfig.RiskWeightVariableIncome},
		{"employee_risk.risk_weight_member_capital", riskConfig.RiskWeightMemberCapital},
		{"employee_risk.risk_weight_employment_dependence", riskConfig.RiskWeightEmploymentDependence},
	} {
		if err := rate(value.path, value.value); err != nil {
			return err
		}
	}
	return nil
}

func validateFinancing(financing Financing) error {
	for _, value := range []namedNumber{
		{"financing.base_credit_line", financing.BaseCreditLine},
		{"financing.scheduled_principal_payment_monthly", financing.ScheduledPrincipalPaymentMonthly},
		{"financing.external_growth_capital_limit_monthly", financing.ExternalGrowthCapitalLimitMonthly},
	} {
		if err := nonNegative(value.path, value.value); err != nil {
			return err
		}
	}
	for _, value := range []namedNumber{
		{"financing.debt_interest_rate_annual", financing.DebtInterestRateAnnual},
		{"financing.distribution_payroll_tax_rate", financing.DistributionPayrollTaxRate},
		{"financing.distribution_tax_deductible_share", financing.DistributionTaxDeductibleShare},
		{"financing.member_capital_redemption_fraction_on_exit", financing.MemberCapitalRedemptionFractionOnExit},
		{"financing.reserve_release_rate_on_stress", financing.ReserveReleaseRateOnStress},
	} {
		if err := rate(value.path, value.value); err != nil {
			return err
		}
	}
	if err := oneOf("financing.external_capital_type", financing.ExternalCapitalType,
		ExternalCapitalDebt, ExternalCapitalNonDilutiveGrant); err != nil {
		return err
	}
	if err := intBetween("financing.employee_distribution_payout_lag_months", financing.EmployeeDistributionPayoutLagMonths, 0, 24); err != nil {
		return err
	}
	if err := intBetween("financing.member_capital_redemption_lag_months", financing.MemberCapitalRedemptionLagMonths, 0, 120); err != nil {
		return err
	}
	return nil
}

func validateBehaviorCases(cases map[string]BehaviorCase) error {
	if len(cases) == 0 {
		return fieldError("behavior_cases", "must contain at least one behavior case")
	}
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			return fieldError("behavior_cases", "behavior case names must not be empty")
		}
		behavior := cases[name]
		prefix := joinJSONPath("behavior_cases", name)
		if err := nonNegative(prefix+".free_rider_size_exponent", behavior.FreeRiderSizeExponent); err != nil {
			return err
		}
		if err := positive(prefix+".free_rider_reference_headcount", behavior.FreeRiderReferenceHeadcount); err != nil {
			return err
		}
		if err := nonNegative(prefix+".free_rider_max_size_multiplier", behavior.FreeRiderMaxSizeMultiplier); err != nil {
			return err
		}
		if err := nonNegative(prefix+".free_rider_base_penalty", behavior.FreeRiderBasePenalty); err != nil {
			return err
		}
		if name == "no_effect" {
			if err := validateNoEffectBehavior(prefix, behavior); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateNoEffectBehavior(prefix string, behavior BehaviorCase) error {
	values := []namedNumber{
		{prefix + ".base_productivity_uplift_direct", behavior.BaseProductivityUpliftDirect},
		{prefix + ".ownership_productivity_sensitivity", behavior.OwnershipProductivitySensitivity},
		{prefix + ".profit_distribution_productivity_sensitivity", behavior.ProfitDistributionProductivitySensitivity},
		{prefix + ".governance_voice_productivity_sensitivity", behavior.GovernanceVoiceProductivitySensitivity},
		{prefix + ".base_turnover_delta_annual_pp", behavior.BaseTurnoverDeltaAnnualPP},
		{prefix + ".ownership_retention_delta_annual_pp_per_full_ownership", behavior.OwnershipRetentionDeltaAnnualPPPerFullOwnership},
		{prefix + ".profit_distribution_retention_delta_annual_pp_per_10pp", behavior.ProfitDistributionRetentionDeltaAnnualPPPer10PP},
		{prefix + ".governance_retention_delta_annual_pp_per_full_participation", behavior.GovernanceRetentionDeltaAnnualPPPerFullParticipation},
		{prefix + ".fairness_productivity_sensitivity", behavior.FairnessProductivitySensitivity},
		{prefix + ".fairness_turnover_sensitivity_annual_pp", behavior.FairnessTurnoverSensitivityAnnualPP},
		{prefix + ".free_rider_base_penalty", behavior.FreeRiderBasePenalty},
		{prefix + ".risk_concentration_turnover_sensitivity_annual_pp", behavior.RiskConcentrationTurnoverSensitivityAnnualPP},
		{prefix + ".income_volatility_turnover_sensitivity_annual_pp", behavior.IncomeVolatilityTurnoverSensitivityAnnualPP},
		{prefix + ".high_performer_attrition_delta_pp", behavior.HighPerformerAttritionDeltaPP},
	}
	for _, value := range values {
		if value.value != 0 {
			return fieldError(value.path, "must be 0 for no_effect")
		}
	}
	return nil
}

func validateSimulationWorkforceCompatibility(cfg Config) error {
	sim := cfg.Simulation
	workforce := cfg.Workforce
	if sim.Mode == ModeDeterministic && workforce.TurnoverRandomMode != TurnoverDeterministic {
		return fieldError("workforce.turnover_random_mode", "must be deterministic when simulation.mode is deterministic")
	}
	if sim.Mode == ModeDeterministic && sim.HeadcountMode == HeadcountIntegerRandom {
		return fieldError("simulation.headcount_mode", "must not be integer_random when simulation.mode is deterministic")
	}
	if workforce.TurnoverRandomMode == TurnoverBinomial && sim.HeadcountMode == HeadcountFractional {
		return fieldError("simulation.headcount_mode", "must be an integer mode when workforce.turnover_random_mode is binomial")
	}
	if sim.HeadcountMode != HeadcountFractional && math.Trunc(cfg.CompanyEconomics.InitialHeadcount) != cfg.CompanyEconomics.InitialHeadcount {
		return fieldError("company_economics.initial_headcount", "must be an integer when simulation.headcount_mode is %s", sim.HeadcountMode)
	}
	return nil
}

func validateCompatibilityV03(compat CompatibilityV03) error {
	if err := oneOf("compatibility_v0_3.headcount_policy", compat.HeadcountPolicy,
		HeadcountPolicyDynamicV04, HeadcountPolicyFixedV03); err != nil {
		return err
	}
	if compat.Enabled {
		return fieldError("compatibility_v0_3.enabled", "is not executed by the v0.4 engine; run a legacy v0.3 configuration through the compatibility CLI")
	}
	if compat.LegacyOutputsEnabled {
		return fieldError("compatibility_v0_3.legacy_outputs_enabled", "requires the legacy v0.3 CLI output path")
	}
	if compat.HeadcountPolicy == HeadcountPolicyFixedV03 {
		return fieldError("compatibility_v0_3.headcount_policy", "fixed_v0_3 is available only through the legacy v0.3 engine")
	}
	return nil
}

func validateScenarios(scenarios []OrganizationalScenario, cases map[string]BehaviorCase) (map[string]struct{}, error) {
	if len(scenarios) < 3 {
		return nil, fieldError("organizational_scenarios", "must contain at least 3 scenarios")
	}
	names := make(map[string]struct{}, len(scenarios))
	for index, scenario := range scenarios {
		prefix := fmt.Sprintf("organizational_scenarios[%d]", index)
		if !scenarioNamePattern.MatchString(scenario.Name) {
			return nil, fieldError(prefix+".name", "must match ^[a-z][a-z0-9_]*$")
		}
		if _, exists := names[scenario.Name]; exists {
			return nil, fieldError(prefix+".name", "duplicate scenario name %q", scenario.Name)
		}
		names[scenario.Name] = struct{}{}
		if err := oneOf(prefix+".system_type", scenario.SystemType,
			SystemTraditionalCompany, SystemProfitSharing,
			SystemEmployeeOwnershipPartial, SystemWorkerCooperative); err != nil {
			return nil, err
		}
		for _, value := range []namedNumber{
			{prefix + ".employee_ownership_fraction", scenario.EmployeeOwnershipFraction},
			{prefix + ".employee_cash_distribution_rate", scenario.EmployeeCashDistributionRate},
			{prefix + ".member_capital_allocation_rate", scenario.MemberCapitalAllocationRate},
			{prefix + ".reinvestment_rate", scenario.ReinvestmentRate},
			{prefix + ".organizational_reserve_rate", scenario.OrganizationalReserveRate},
			{prefix + ".external_distribution_rate", scenario.ExternalDistributionRate},
			{prefix + ".contribution_measurement_quality", scenario.ContributionMeasurementQuality},
			{prefix + ".peer_monitoring_effectiveness", scenario.PeerMonitoringEffectiveness},
			{prefix + ".transparency_index", scenario.TransparencyIndex},
			{prefix + ".employment_stabilization_preference", scenario.EmploymentStabilizationPreference},
		} {
			if err := rate(value.path, value.value); err != nil {
				return nil, err
			}
		}
		if err := nonNegative(prefix+".result_hurdle_monthly", scenario.ResultHurdleMonthly); err != nil {
			return nil, err
		}
		if err := between(prefix+".external_capital_access_multiplier", scenario.ExternalCapitalAccessMultiplier, 0, 5); err != nil {
			return nil, err
		}
		if err := intBetween(prefix+".profit_distribution_period_months", scenario.ProfitDistributionPeriodMonths, 1, 12); err != nil {
			return nil, err
		}
		if scenario.MaxDistributionPerEmployeePeriod != nil {
			if err := nonNegative(prefix+".max_distribution_per_employee_period", *scenario.MaxDistributionPerEmployeePeriod); err != nil {
				return nil, err
			}
		}
		if err := oneOf(prefix+".distribution_rule", scenario.DistributionRule,
			DistributionNone, DistributionEqualPerCapita,
			DistributionContributionWeighted, DistributionHybrid); err != nil {
			return nil, err
		}
		if err := validateAllocationPriority(prefix, scenario); err != nil {
			return nil, err
		}
		if err := validateGovernance(prefix+".governance", scenario.Governance); err != nil {
			return nil, err
		}
		if len(scenario.BehaviorCaseRefs) == 0 {
			return nil, fieldError(prefix+".behavior_case_refs", "must contain at least one behavior case reference")
		}
		seenRefs := make(map[string]struct{}, len(scenario.BehaviorCaseRefs))
		for refIndex, ref := range scenario.BehaviorCaseRefs {
			path := fmt.Sprintf("%s.behavior_case_refs[%d]", prefix, refIndex)
			if _, exists := cases[ref]; !exists {
				return nil, fieldError(path, "unknown behavior case %q", ref)
			}
			if _, exists := seenRefs[ref]; exists {
				return nil, fieldError(path, "duplicate behavior case reference %q", ref)
			}
			seenRefs[ref] = struct{}{}
		}
	}
	return names, nil
}

func validateAllocationPriority(prefix string, scenario OrganizationalScenario) error {
	rates := map[string]float64{
		AllocationOrganizationalReserve: scenario.OrganizationalReserveRate,
		AllocationReinvestment:          scenario.ReinvestmentRate,
		AllocationEmployeeDistribution:  scenario.EmployeeCashDistributionRate,
		AllocationMemberCapital:         scenario.MemberCapitalAllocationRate,
		AllocationExternalDistribution:  scenario.ExternalDistributionRate,
	}
	sum := 0.0
	for _, value := range rates {
		sum += value
	}
	if sum > 1 {
		return fieldError(prefix, "allocation rates sum to %.17g; must be <= 1", sum)
	}
	seen := make(map[string]struct{}, len(scenario.AllocationPriority))
	for index, allocation := range scenario.AllocationPriority {
		path := fmt.Sprintf("%s.allocation_priority[%d]", prefix, index)
		rateValue, valid := rates[allocation]
		if !valid {
			return fieldError(path, "invalid allocation id %q", allocation)
		}
		if _, exists := seen[allocation]; exists {
			return fieldError(path, "duplicate allocation id %q", allocation)
		}
		if rateValue <= 0 {
			return fieldError(path, "allocation %q has no positive allocation rate", allocation)
		}
		seen[allocation] = struct{}{}
	}
	allocationIDs := make([]string, 0, len(rates))
	for allocation := range rates {
		allocationIDs = append(allocationIDs, allocation)
	}
	sort.Strings(allocationIDs)
	for _, allocation := range allocationIDs {
		if rates[allocation] <= 0 {
			continue
		}
		if _, exists := seen[allocation]; !exists {
			return fieldError(prefix+".allocation_priority", "missing positive-rate allocation %q", allocation)
		}
	}
	return nil
}

func validateGovernance(prefix string, governance Governance) error {
	if err := rate(prefix+".governance_participation_intensity", governance.GovernanceParticipationIntensity); err != nil {
		return err
	}
	for _, value := range []namedNumber{
		{prefix + ".base_governance_hours_per_employee_month", governance.BaseGovernanceHoursPerEmployeeMonth},
		{prefix + ".fixed_governance_hours_monthly", governance.FixedGovernanceHoursMonthly},
		{prefix + ".governance_cash_cost_fixed_monthly", governance.GovernanceCashCostFixedMonthly},
		{prefix + ".governance_cash_cost_per_employee_monthly", governance.GovernanceCashCostPerEmployeeMonthly},
		{prefix + ".decision_complexity_index", governance.DecisionComplexityIndex},
		{prefix + ".base_decision_delay_months", governance.BaseDecisionDelayMonths},
		{prefix + ".delay_per_participation_months", governance.DelayPerParticipationMonths},
		{prefix + ".decentralization_speed_gain_months", governance.DecentralizationSpeedGainMonths},
	} {
		if err := nonNegative(value.path, value.value); err != nil {
			return err
		}
	}
	for _, value := range []namedNumber{
		{prefix + ".local_autonomy_index", governance.LocalAutonomyIndex},
		{prefix + ".information_sharing_quality", governance.InformationSharingQuality},
		{prefix + ".trust_index", governance.TrustIndex},
	} {
		if err := rate(value.path, value.value); err != nil {
			return err
		}
	}
	if err := positive(prefix+".governance_capability_index", governance.GovernanceCapabilityIndex); err != nil {
		return err
	}
	if err := nonNegative(prefix+".decision_quality_min", governance.DecisionQualityMin); err != nil {
		return err
	}
	if err := nonNegative(prefix+".decision_quality_max", governance.DecisionQualityMax); err != nil {
		return err
	}
	if governance.DecisionQualityMin > governance.DecisionQualityMax {
		return fieldError(prefix+".decision_quality_min", "must be <= decision_quality_max")
	}
	return nil
}

func validateAnalysis(cfg Config, scenarioNames map[string]struct{}) error {
	analysis := cfg.Analysis
	if len(analysis.PairedReferenceScenarios) == 0 {
		return fieldError("analysis.paired_reference_scenarios", "must contain at least one scenario reference")
	}
	seen := make(map[string]struct{}, len(analysis.PairedReferenceScenarios))
	for index, name := range analysis.PairedReferenceScenarios {
		path := fmt.Sprintf("analysis.paired_reference_scenarios[%d]", index)
		if _, exists := scenarioNames[name]; !exists {
			return fieldError(path, "unknown scenario %q", name)
		}
		if _, exists := seen[name]; exists {
			return fieldError(path, "duplicate scenario reference %q", name)
		}
		seen[name] = struct{}{}
	}
	if err := nonNegative("analysis.volatility_penalty_lambda", analysis.VolatilityPenaltyLambda); err != nil {
		return err
	}
	if err := rate("analysis.classification_tolerance", analysis.ClassificationTolerance); err != nil {
		return err
	}
	if err := oneOf("analysis.break_even_metric", analysis.BreakEvenMetric,
		BreakEvenSustainableDevelopment, BreakEvenCashEndUnrestricted,
		BreakEvenCapacityGrowth, BreakEvenRevenueCAGR); err != nil {
		return err
	}
	if len(analysis.BreakEvenUpliftRange) != 2 {
		return fieldError("analysis.break_even_uplift_range", "must contain exactly two values")
	}
	if analysis.BreakEvenUpliftRange[0] >= analysis.BreakEvenUpliftRange[1] {
		return fieldError("analysis.break_even_uplift_range", "lower bound must be less than upper bound")
	}
	for index, parameter := range analysis.SensitivityParameters {
		prefix := fmt.Sprintf("analysis.sensitivity_parameters[%d]", index)
		if strings.TrimSpace(parameter.Path) == "" {
			return fieldError(prefix+".path", "must not be empty")
		}
		if err := resolveNumericPath(cfg, parameter.Path); err != nil {
			return fieldError(prefix+".path", "%v", err)
		}
		if len(parameter.Values) == 0 {
			return fieldError(prefix+".values", "must contain at least one value")
		}
	}
	return nil
}

func resolveNumericPath(cfg Config, path string) error {
	segments := strings.Split(path, ".")
	for _, segment := range segments {
		if segment == "" {
			return fmt.Errorf("invalid empty path segment in %q", path)
		}
	}
	value := reflect.ValueOf(cfg)
	typ := reflect.TypeOf(cfg)
	for _, segment := range segments {
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
			if value.IsValid() && value.Kind() == reflect.Pointer {
				if value.IsNil() {
					value = reflect.Zero(typ)
				} else {
					value = value.Elem()
				}
			}
		}
		switch typ.Kind() {
		case reflect.Struct:
			fieldIndex := -1
			for index := 0; index < typ.NumField(); index++ {
				field := typ.Field(index)
				if strings.Split(field.Tag.Get("json"), ",")[0] == segment {
					fieldIndex = index
					break
				}
			}
			if fieldIndex < 0 {
				return fmt.Errorf("path %q does not resolve: unknown field %q", path, segment)
			}
			typ = typ.Field(fieldIndex).Type
			if value.IsValid() && value.Kind() == reflect.Struct {
				value = value.Field(fieldIndex)
			}
		case reflect.Map:
			if typ.Key().Kind() != reflect.String {
				return fmt.Errorf("path %q traverses an unsupported map", path)
			}
			if !value.IsValid() || value.Kind() != reflect.Map {
				return fmt.Errorf("path %q does not resolve at %q", path, segment)
			}
			entry := value.MapIndex(reflect.ValueOf(segment))
			if !entry.IsValid() {
				return fmt.Errorf("path %q does not resolve: unknown name %q", path, segment)
			}
			value = entry
			typ = typ.Elem()
		case reflect.Slice:
			if typ.Elem() != reflect.TypeOf(OrganizationalScenario{}) {
				return fmt.Errorf("path %q cannot traverse an array by name", path)
			}
			found := false
			if value.IsValid() && value.Kind() == reflect.Slice {
				for index := 0; index < value.Len(); index++ {
					scenario := value.Index(index).Interface().(OrganizationalScenario)
					if scenario.Name == segment {
						value = value.Index(index)
						found = true
						break
					}
				}
			}
			if !found {
				return fmt.Errorf("path %q does not resolve: unknown scenario %q", path, segment)
			}
			typ = typ.Elem()
		default:
			return fmt.Errorf("path %q continues beyond scalar field at %q", path, segment)
		}
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Float32, reflect.Float64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return nil
	default:
		return fmt.Errorf("path %q must resolve to a numeric field, got %s", path, typ.Kind())
	}
}

type namedNumber struct {
	path  string
	value float64
}

func validateFiniteNumbers(value reflect.Value, typ reflect.Type, path string) error {
	for typ.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		typ = typ.Elem()
		value = value.Elem()
	}
	switch typ.Kind() {
	case reflect.Float32, reflect.Float64:
		number := value.Float()
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return fieldError(path, "must be finite")
		}
	case reflect.Struct:
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			if err := validateFiniteNumbers(value.Field(index), field.Type, joinJSONPath(path, name)); err != nil {
				return err
			}
		}
	case reflect.Map:
		keys := value.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
		for _, key := range keys {
			if err := validateFiniteNumbers(value.MapIndex(key), typ.Elem(), joinJSONPath(path, key.String())); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			if err := validateFiniteNumbers(value.Index(index), typ.Elem(), fmt.Sprintf("%s[%d]", displayPath(path), index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func oneOf(path, value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fieldError(path, "invalid value %q; must be one of %s", value, strings.Join(allowed, ", "))
}

func intBetween(path string, value, minimum, maximum int) error {
	if value < minimum || value > maximum {
		return fieldError(path, "must be between %d and %d", minimum, maximum)
	}
	return nil
}

func positive(path string, value float64) error {
	if value <= 0 {
		return fieldError(path, "must be > 0")
	}
	return nil
}

func nonNegative(path string, value float64) error {
	if value < 0 {
		return fieldError(path, "must be >= 0")
	}
	return nil
}

func rate(path string, value float64) error {
	return between(path, value, 0, 1)
}

func between(path string, value, minimum, maximum float64) error {
	if value < minimum || value > maximum {
		return fieldError(path, "must be between %g and %g", minimum, maximum)
	}
	return nil
}
