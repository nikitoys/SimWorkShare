// Package config defines and strictly validates the SimWorkShare v0.4
// configuration format. It is intentionally independent from internal/config,
// which remains the v0.3 compatibility configuration.
package config

const (
	SchemaVersion = "0.4"

	ModeDeterministic = "deterministic"
	ModeMonteCarlo    = "monte_carlo"

	HeadcountFractional      = "fractional"
	HeadcountIntegerExpected = "integer_expected"
	HeadcountIntegerRandom   = "integer_random"

	MarketProcessDeterministic    = "deterministic"
	MarketProcessBoundedLognormal = "bounded_lognormal"

	TurnoverDeterministic = "deterministic"
	TurnoverBinomial      = "binomial"

	ExternalCapitalDebt             = "debt"
	ExternalCapitalNonDilutiveGrant = "non_dilutive_grant"

	SystemTraditionalCompany       = "traditional_company"
	SystemProfitSharing            = "profit_sharing"
	SystemEmployeeOwnershipPartial = "employee_ownership_partial"
	SystemWorkerCooperative        = "worker_cooperative"

	AllocationOrganizationalReserve = "organizational_reserve"
	AllocationReinvestment          = "reinvestment"
	AllocationEmployeeDistribution  = "employee_cash_distribution"
	AllocationMemberCapital         = "member_capital_allocation"
	AllocationExternalDistribution  = "external_distribution"

	DistributionNone                 = "none"
	DistributionEqualPerCapita       = "equal_per_capita"
	DistributionContributionWeighted = "contribution_weighted"
	DistributionHybrid               = "hybrid"

	BreakEvenSustainableDevelopment = "sustainable_development_value_proxy"
	BreakEvenCashEndUnrestricted    = "cash_end_unrestricted"
	BreakEvenCapacityGrowth         = "productive_capacity_growth_rate"
	BreakEvenRevenueCAGR            = "revenue_cagr"

	HeadcountPolicyDynamicV04 = "dynamic_v0_4"
	HeadcountPolicyFixedV03   = "fixed_v0_3"
)

// Config contains every field in the canonical v0.4 configuration document.
// All fields are required in JSON. MaxDistributionPerEmployeePeriod is the one
// required field that may explicitly be null, meaning that no cap is applied.
type Config struct {
	SchemaVersion           string                   `json:"schema_version"`
	ConfigValidation        ConfigValidation         `json:"config_validation"`
	Simulation              Simulation               `json:"simulation"`
	Units                   map[string]string        `json:"units"`
	CompanyEconomics        CompanyEconomics         `json:"company_economics"`
	Market                  Market                   `json:"market"`
	Workforce               Workforce                `json:"workforce"`
	EmployeeRisk            EmployeeRisk             `json:"employee_risk"`
	Financing               Financing                `json:"financing"`
	BehaviorCases           map[string]BehaviorCase  `json:"behavior_cases"`
	OrganizationalScenarios []OrganizationalScenario `json:"organizational_scenarios"`
	Analysis                Analysis                 `json:"analysis"`
	Reporting               Reporting                `json:"reporting"`
	CompatibilityV03        CompatibilityV03         `json:"compatibility_v0_3"`
}

type ConfigValidation struct {
	RejectUnknownFields    bool `json:"reject_unknown_fields"`
	RejectDuplicateFields  bool `json:"reject_duplicate_fields"`
	AllowNameNormalization bool `json:"allow_name_normalization"`
	RejectNaNAndInfinity   bool `json:"reject_nan_and_infinity"`
}

type Simulation struct {
	Mode                string  `json:"mode"`
	Months              int     `json:"months"`
	HorizonsMonths      []int   `json:"horizons_months"`
	Runs                int     `json:"runs"`
	RandomSeed          int64   `json:"random_seed"`
	CommonRandomNumbers bool    `json:"common_random_numbers"`
	Currency            string  `json:"currency"`
	Epsilon             float64 `json:"epsilon"`
	HeadcountMode       string  `json:"headcount_mode"`
	StopAfterBankruptcy bool    `json:"stop_after_bankruptcy"`
}

type CompanyEconomics struct {
	InitialHeadcount                          float64 `json:"initial_headcount"`
	BaseSalaryPerEmployeeMonthly              float64 `json:"base_salary_per_employee_monthly"`
	SalaryPayrollTaxRate                      float64 `json:"salary_payroll_tax_rate"`
	StandardHoursPerEmployeeMonth             float64 `json:"standard_hours_per_employee_month"`
	BaseRevenuePerEffectiveEmployeeMonthly    float64 `json:"base_revenue_per_effective_employee_monthly"`
	InitialMarketDemandMonthly                float64 `json:"initial_market_demand_monthly"`
	InitialProductiveCapacityRevenueMonthly   float64 `json:"initial_productive_capacity_revenue_monthly"`
	FixedCostsMonthly                         float64 `json:"fixed_costs_monthly"`
	VariableCostRate                          float64 `json:"variable_cost_rate"`
	ProfitTaxRate                             float64 `json:"profit_tax_rate"`
	ProfitTaxPaymentLagMonths                 int     `json:"profit_tax_payment_lag_months"`
	CostInflationMonthly                      float64 `json:"cost_inflation_monthly"`
	CapacityDepreciationRateMonthly           float64 `json:"capacity_depreciation_rate_monthly"`
	CapacityRevenueCreatedPerCurrencyInvested float64 `json:"capacity_revenue_created_per_currency_invested"`
	InvestmentActivationLagMonths             int     `json:"investment_activation_lag_months"`
	RequiredCashReserveMonths                 float64 `json:"required_cash_reserve_months"`
	StartingCash                              float64 `json:"starting_cash"`
	OpeningAccountsReceivable                 float64 `json:"opening_accounts_receivable"`
}

type Market struct {
	MarketProcess                     string    `json:"market_process"`
	MarketGrowthMonthly               float64   `json:"market_growth_monthly"`
	MarketVolatilityMonthly           float64   `json:"market_volatility_monthly"`
	MarketFactorMin                   float64   `json:"market_factor_min"`
	MarketFactorMax                   float64   `json:"market_factor_max"`
	SeasonalityMultipliers            []float64 `json:"seasonality_multipliers"`
	RevenueCollectionRateCurrentMonth float64   `json:"revenue_collection_rate_current_month"`
	AccountsReceivableLagMonths       int       `json:"accounts_receivable_lag_months"`
	BadDebtRate                       float64   `json:"bad_debt_rate"`
	ShockProbabilityMonthly           float64   `json:"shock_probability_monthly"`
	ShockRevenueMultiplier            float64   `json:"shock_revenue_multiplier"`
	ShockCostMean                     float64   `json:"shock_cost_mean"`
	ShockCostStd                      float64   `json:"shock_cost_std"`
	CashCollectionStressMultiplier    float64   `json:"cash_collection_stress_multiplier"`
	LaborMarketFactor                 float64   `json:"labor_market_factor"`
	CreditMarketFactor                float64   `json:"credit_market_factor"`
}

type Workforce struct {
	BaseTurnoverRateAnnual                       float64   `json:"base_turnover_rate_annual"`
	MinTurnoverRateAnnual                        float64   `json:"min_turnover_rate_annual"`
	MaxTurnoverRateAnnual                        float64   `json:"max_turnover_rate_annual"`
	TurnoverRandomMode                           string    `json:"turnover_random_mode"`
	HighPerformerShare                           float64   `json:"high_performer_share"`
	RecruitingCostPerHire                        float64   `json:"recruiting_cost_per_hire"`
	OnboardingCostPerHire                        float64   `json:"onboarding_cost_per_hire"`
	ManagerTimeCostPerHire                       float64   `json:"manager_time_cost_per_hire"`
	ExitAdminCostPerLeaver                       float64   `json:"exit_admin_cost_per_leaver"`
	LostProductivityCostPerLeaver                float64   `json:"lost_productivity_cost_per_leaver"`
	SeveranceCostPerLayoff                       float64   `json:"severance_cost_per_layoff"`
	RampDurationMonths                           int       `json:"ramp_duration_months"`
	RampProductivityMultipliers                  []float64 `json:"ramp_productivity_multipliers"`
	MaxHiresPerMonthRate                         float64   `json:"max_hires_per_month_rate"`
	MaxLayoffsPerMonthRate                       float64   `json:"max_layoffs_per_month_rate"`
	LayoffTriggerCashRatio                       float64   `json:"layoff_trigger_cash_ratio"`
	LeaverPaidFractionOfMonth                    float64   `json:"leaver_paid_fraction_of_month"`
	NewHirePaidFractionOfMonth                   float64   `json:"new_hire_paid_fraction_of_month"`
	MinProductivityUplift                        float64   `json:"min_productivity_uplift"`
	MaxProductivityUplift                        float64   `json:"max_productivity_uplift"`
	TurnoverProductivityPenaltyPerAnnualTurnover float64   `json:"turnover_productivity_penalty_per_annual_turnover"`
	TargetStaffingBuffer                         float64   `json:"target_staffing_buffer"`
	MaxCashShareForHiring                        float64   `json:"max_cash_share_for_hiring"`
}

type EmployeeRisk struct {
	EmployeeExternalSavingsProxyPerEmployee float64 `json:"employee_external_savings_proxy_per_employee"`
	EmploymentDependenceIndex               float64 `json:"employment_dependence_index"`
	RiskWeightVariableIncome                float64 `json:"risk_weight_variable_income"`
	RiskWeightMemberCapital                 float64 `json:"risk_weight_member_capital"`
	RiskWeightEmploymentDependence          float64 `json:"risk_weight_employment_dependence"`
}

type Financing struct {
	BaseCreditLine                        float64 `json:"base_credit_line"`
	DebtInterestRateAnnual                float64 `json:"debt_interest_rate_annual"`
	ScheduledPrincipalPaymentMonthly      float64 `json:"scheduled_principal_payment_monthly"`
	ExternalGrowthCapitalLimitMonthly     float64 `json:"external_growth_capital_limit_monthly"`
	ExternalCapitalType                   string  `json:"external_capital_type"`
	DistributionPayrollTaxRate            float64 `json:"distribution_payroll_tax_rate"`
	DistributionTaxDeductibleShare        float64 `json:"distribution_tax_deductible_share"`
	EmployeeDistributionPayoutLagMonths   int     `json:"employee_distribution_payout_lag_months"`
	MemberCapitalRedemptionLagMonths      int     `json:"member_capital_redemption_lag_months"`
	MemberCapitalRedemptionFractionOnExit float64 `json:"member_capital_redemption_fraction_on_exit"`
	ReserveReleaseRateOnStress            float64 `json:"reserve_release_rate_on_stress"`
}

type BehaviorCase struct {
	BaseProductivityUpliftDirect                         float64 `json:"base_productivity_uplift_direct"`
	OwnershipProductivitySensitivity                     float64 `json:"ownership_productivity_sensitivity"`
	ProfitDistributionProductivitySensitivity            float64 `json:"profit_distribution_productivity_sensitivity"`
	GovernanceVoiceProductivitySensitivity               float64 `json:"governance_voice_productivity_sensitivity"`
	BaseTurnoverDeltaAnnualPP                            float64 `json:"base_turnover_delta_annual_pp"`
	OwnershipRetentionDeltaAnnualPPPerFullOwnership      float64 `json:"ownership_retention_delta_annual_pp_per_full_ownership"`
	ProfitDistributionRetentionDeltaAnnualPPPer10PP      float64 `json:"profit_distribution_retention_delta_annual_pp_per_10pp"`
	GovernanceRetentionDeltaAnnualPPPerFullParticipation float64 `json:"governance_retention_delta_annual_pp_per_full_participation"`
	FairnessBase                                         float64 `json:"fairness_base"`
	TransparencyToFairness                               float64 `json:"transparency_to_fairness"`
	EqualDistributionFairnessEffect                      float64 `json:"equal_distribution_fairness_effect"`
	ContributionBasedDistributionFairnessEffect          float64 `json:"contribution_based_distribution_fairness_effect"`
	PayDispersionFairnessPenalty                         float64 `json:"pay_dispersion_fairness_penalty"`
	UnpaidGovernanceBurdenPenalty                        float64 `json:"unpaid_governance_burden_penalty"`
	ZeroDistributionFairnessPenalty                      float64 `json:"zero_distribution_fairness_penalty"`
	FairnessProductivitySensitivity                      float64 `json:"fairness_productivity_sensitivity"`
	FairnessTurnoverSensitivityAnnualPP                  float64 `json:"fairness_turnover_sensitivity_annual_pp"`
	FreeRiderBasePenalty                                 float64 `json:"free_rider_base_penalty"`
	FreeRiderSizeExponent                                float64 `json:"free_rider_size_exponent"`
	FreeRiderReferenceHeadcount                          float64 `json:"free_rider_reference_headcount"`
	FreeRiderMaxSizeMultiplier                           float64 `json:"free_rider_max_size_multiplier"`
	RiskConcentrationTurnoverSensitivityAnnualPP         float64 `json:"risk_concentration_turnover_sensitivity_annual_pp"`
	IncomeVolatilityTurnoverSensitivityAnnualPP          float64 `json:"income_volatility_turnover_sensitivity_annual_pp"`
	HighPerformerAttritionDeltaPP                        float64 `json:"high_performer_attrition_delta_pp"`
}

type OrganizationalScenario struct {
	Name                              string     `json:"name"`
	SystemType                        string     `json:"system_type"`
	EmployeeOwnershipFraction         float64    `json:"employee_ownership_fraction"`
	EmployeeCashDistributionRate      float64    `json:"employee_cash_distribution_rate"`
	MemberCapitalAllocationRate       float64    `json:"member_capital_allocation_rate"`
	ReinvestmentRate                  float64    `json:"reinvestment_rate"`
	OrganizationalReserveRate         float64    `json:"organizational_reserve_rate"`
	ExternalDistributionRate          float64    `json:"external_distribution_rate"`
	ResultHurdleMonthly               float64    `json:"result_hurdle_monthly"`
	AllocationPriority                []string   `json:"allocation_priority"`
	DistributionRule                  string     `json:"distribution_rule"`
	ContributionMeasurementQuality    float64    `json:"contribution_measurement_quality"`
	PeerMonitoringEffectiveness       float64    `json:"peer_monitoring_effectiveness"`
	TransparencyIndex                 float64    `json:"transparency_index"`
	EmploymentStabilizationPreference float64    `json:"employment_stabilization_preference"`
	ExternalCapitalAccessMultiplier   float64    `json:"external_capital_access_multiplier"`
	ProfitDistributionPeriodMonths    int        `json:"profit_distribution_period_months"`
	MaxDistributionPerEmployeePeriod  *float64   `json:"max_distribution_per_employee_period" nullable:"true"`
	Governance                        Governance `json:"governance"`
	BehaviorCaseRefs                  []string   `json:"behavior_case_refs"`
}

type Governance struct {
	GovernanceParticipationIntensity     float64 `json:"governance_participation_intensity"`
	BaseGovernanceHoursPerEmployeeMonth  float64 `json:"base_governance_hours_per_employee_month"`
	FixedGovernanceHoursMonthly          float64 `json:"fixed_governance_hours_monthly"`
	GovernanceCashCostFixedMonthly       float64 `json:"governance_cash_cost_fixed_monthly"`
	GovernanceCashCostPerEmployeeMonthly float64 `json:"governance_cash_cost_per_employee_monthly"`
	DecisionComplexityIndex              float64 `json:"decision_complexity_index"`
	BaseDecisionDelayMonths              float64 `json:"base_decision_delay_months"`
	DelayPerParticipationMonths          float64 `json:"delay_per_participation_months"`
	LocalAutonomyIndex                   float64 `json:"local_autonomy_index"`
	DecentralizationSpeedGainMonths      float64 `json:"decentralization_speed_gain_months"`
	GovernanceCapabilityIndex            float64 `json:"governance_capability_index"`
	InformationSharingQuality            float64 `json:"information_sharing_quality"`
	TrustIndex                           float64 `json:"trust_index"`
	QualityGainFromParticipation         float64 `json:"quality_gain_from_participation"`
	CoordinationLossFromParticipation    float64 `json:"coordination_loss_from_participation"`
	ConflictLossSensitivity              float64 `json:"conflict_loss_sensitivity"`
	DecisionDelayQualityLoss             float64 `json:"decision_delay_quality_loss"`
	DecisionQualityMin                   float64 `json:"decision_quality_min"`
	DecisionQualityMax                   float64 `json:"decision_quality_max"`
	ShockMitigationSensitivity           float64 `json:"shock_mitigation_sensitivity"`
	ShockDelayAmplification              float64 `json:"shock_delay_amplification"`
	InvestmentEfficiencySensitivity      float64 `json:"investment_efficiency_sensitivity"`
}

type Analysis struct {
	PairedReferenceScenarios []string               `json:"paired_reference_scenarios"`
	VolatilityPenaltyLambda  float64                `json:"volatility_penalty_lambda"`
	ClassificationTolerance  float64                `json:"classification_tolerance"`
	BreakEvenMetric          string                 `json:"break_even_metric"`
	BreakEvenUpliftRange     []float64              `json:"break_even_uplift_range"`
	SensitivityParameters    []SensitivityParameter `json:"sensitivity_parameters"`
}

type SensitivityParameter struct {
	Path   string    `json:"path"`
	Values []float64 `json:"values"`
}

type Reporting struct {
	PrintModelLimitations bool `json:"print_model_limitations"`
	PrintAssumptionFlags  bool `json:"print_assumption_flags"`
	WriteMonthlyCSV       bool `json:"write_monthly_csv"`
	WriteSummaryCSV       bool `json:"write_summary_csv"`
	WriteSensitivityCSV   bool `json:"write_sensitivity_csv"`
	WriteBreakEvenCSV     bool `json:"write_break_even_csv"`
}

type CompatibilityV03 struct {
	Enabled                           bool   `json:"enabled"`
	PreserveV03ProfitSharingScenarios bool   `json:"preserve_v0_3_profit_sharing_scenarios"`
	LegacyOutputsEnabled              bool   `json:"legacy_outputs_enabled"`
	HeadcountPolicy                   string `json:"headcount_policy"`
}
