package config

// Config mirrors the implementation-ready v0.3 JSON configuration. The
// deterministic runner supports fixed_only and the monthly profit_share slice,
// while still parsing and validating the complete supplied document so that
// unsupported fields are never silently ignored.
type Config struct {
	Simulation            SimulationConfig        `json:"simulation"`
	Company               CompanyConfig           `json:"company"`
	Cashflow              CashflowConfig          `json:"cashflow"`
	Workforce             WorkforceConfig         `json:"workforce"`
	Environment           EnvironmentConfig       `json:"environment"`
	BehaviorCases         map[string]BehaviorCase `json:"behavior_cases"`
	FixedRaiseBudget      FixedRaiseBudgetConfig  `json:"fixed_raise_budget"`
	CompensationScenarios []CompensationScenario  `json:"compensation_scenarios"`
	EnvironmentCases      []string                `json:"environment_cases"`
	Reporting             ReportingConfig         `json:"reporting"`
}

type SimulationConfig struct {
	Months              int    `json:"months"`
	Runs                int    `json:"runs"`
	RandomSeed          int64  `json:"random_seed"`
	CommonRandomNumbers bool   `json:"common_random_numbers"`
	Currency            string `json:"currency"`
}

type CompanyConfig struct {
	EmployeesCount                int      `json:"employees_count"`
	BaseSalaryPerEmployee         float64  `json:"base_salary_per_employee"`
	BaseRevenuePerEmployee        float64  `json:"base_revenue_per_employee"`
	FixedCostsMonthly             float64  `json:"fixed_costs_monthly"`
	VariableCostRate              float64  `json:"variable_cost_rate"`
	StartingCash                  float64  `json:"starting_cash"`
	OpeningAccountsReceivable     float64  `json:"opening_accounts_receivable"`
	RequiredCashReserveMonths     float64  `json:"required_cash_reserve_months"`
	RevenueProductivityElasticity float64  `json:"revenue_productivity_elasticity"`
	DemandCapMultiplier           *float64 `json:"demand_cap_multiplier"`
}

type CashflowConfig struct {
	RevenueCollectionRateCurrentMonth float64             `json:"revenue_collection_rate_current_month"`
	AccountsReceivableLagMonths       int                 `json:"accounts_receivable_lag_months"`
	BadDebtRate                       float64             `json:"bad_debt_rate"`
	ProfitTaxRate                     float64             `json:"profit_tax_rate"`
	ProfitTaxPaymentLagMonths         int                 `json:"profit_tax_payment_lag_months"`
	ReserveCurrentProfitTax           bool                `json:"reserve_current_profit_tax"`
	BonusPayrollTaxRate               float64             `json:"bonus_payroll_tax_rate"`
	DebtServiceMonthly                float64             `json:"debt_service_monthly"`
	CapexMonthly                      float64             `json:"capex_monthly"`
	PlannedReinvestmentRate           float64             `json:"planned_reinvestment_rate"`
	AvailableCreditLine               float64             `json:"available_credit_line"`
	OwnerDividendPolicy               OwnerDividendPolicy `json:"owner_dividend_policy"`
	ReserveBonusCashAtAccrual         bool                `json:"reserve_bonus_cash_at_accrual"`
}

type OwnerDividendPolicy struct {
	Type string `json:"type"`
}

type WorkforceConfig struct {
	BaseTurnoverRateAnnual                       float64 `json:"base_turnover_rate_annual"`
	MinTurnoverRateAnnual                        float64 `json:"min_turnover_rate_annual"`
	MaxTurnoverRateAnnual                        float64 `json:"max_turnover_rate_annual"`
	TurnoverRandomMode                           string  `json:"turnover_random_mode"`
	RecruitingCostPerLeaver                      float64 `json:"recruiting_cost_per_leaver"`
	OnboardingCostPerLeaver                      float64 `json:"onboarding_cost_per_leaver"`
	ManagerTimeCostPerLeaver                     float64 `json:"manager_time_cost_per_leaver"`
	LostProductivityCostPerLeaver                float64 `json:"lost_productivity_cost_per_leaver"`
	TurnoverProductivityPenaltyPerAnnualTurnover float64 `json:"turnover_productivity_penalty_per_annual_turnover"`
	HighPerformerShare                           float64 `json:"high_performer_share"`
	MinProductivityUplift                        float64 `json:"min_productivity_uplift"`
	MaxProductivityUplift                        float64 `json:"max_productivity_uplift"`
}

type EnvironmentConfig struct {
	MarketGrowthMonthly            float64 `json:"market_growth_monthly"`
	MarketVolatilityMonthly        float64 `json:"market_volatility_monthly"`
	MarketProcess                  string  `json:"market_process"`
	CostInflationMonthly           float64 `json:"cost_inflation_monthly"`
	LaborMarketFactor              float64 `json:"labor_market_factor"`
	ShockProbabilityMonthly        float64 `json:"shock_probability_monthly"`
	ShockRevenueMultiplier         float64 `json:"shock_revenue_multiplier"`
	ShockCostMean                  float64 `json:"shock_cost_mean"`
	CashCollectionStressMultiplier float64 `json:"cash_collection_stress_multiplier"`
}

type BehaviorCase struct {
	ProductivityUpliftDirect float64 `json:"productivity_uplift_direct"`
	TurnoverDeltaAnnualPP    float64 `json:"turnover_delta_annual_pp"`
	HighPerfAttritionDeltaPP float64 `json:"high_perf_attrition_delta_pp"`
	FairnessPenaltyToQuality float64 `json:"fairness_penalty_to_quality"`
}

type FixedRaiseBudgetConfig struct {
	Mode                          string `json:"mode"`
	ReferenceCompensationScenario string `json:"reference_compensation_scenario"`
	ReferenceBehaviorCase         string `json:"reference_behavior_case"`
	ReferenceEnvironmentCase      string `json:"reference_environment_case"`
	Statistic                     string `json:"statistic"`
}

type CompensationScenario struct {
	Name                      string   `json:"name"`
	Type                      string   `json:"type"`
	Reference                 string   `json:"reference,omitempty"`
	ProfitSharePercent        *float64 `json:"profit_share_percent,omitempty"`
	ProfitHurdleMonthly       *float64 `json:"profit_hurdle_monthly,omitempty"`
	BonusBaseType             string   `json:"bonus_base_type,omitempty"`
	EligibleEmployeesCount    *int     `json:"eligible_employees_count,omitempty"`
	BonusCapTotal             *float64 `json:"bonus_cap_total,omitempty"`
	BonusCapPerEmployee       *float64 `json:"bonus_cap_per_employee,omitempty"`
	BonusPeriod               string   `json:"bonus_period,omitempty"`
	BonusPayoutLagMonths      *int     `json:"bonus_payout_lag_months,omitempty"`
	EqualDistribution         *bool    `json:"equal_distribution,omitempty"`
	BonusSmoothingReserveRate *float64 `json:"bonus_smoothing_reserve_rate,omitempty"`
}

type ReportingConfig struct {
	VolatilityPenaltyLambda float64 `json:"volatility_penalty_lambda"`
	SmallThresholdForROI    float64 `json:"small_threshold_for_roi"`
	PrintModelLimitations   bool    `json:"print_model_limitations"`
	PrintAssumptionFlags    bool    `json:"print_assumption_flags"`
}
