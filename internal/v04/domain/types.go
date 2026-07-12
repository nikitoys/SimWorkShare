package domain

const DefaultMarketCase = "default_market"

type EnvironmentMonth struct {
	Run                      int     `json:"run"`
	Month                    int     `json:"month"`
	MarketTrend              float64 `json:"market_trend"`
	MarketFactor             float64 `json:"market_factor"`
	SeasonalityMultiplier    float64 `json:"seasonality_multiplier"`
	ShockHappened            bool    `json:"shock_happened"`
	ShockCost                float64 `json:"shock_cost"`
	CollectionRateMultiplier float64 `json:"collection_rate_multiplier"`
	LaborMarketFactor        float64 `json:"labor_market_factor"`
	CreditMarketFactor       float64 `json:"credit_market_factor"`
}

type IncomeMonth struct {
	FixedSalaryPaid      float64
	CashDistributionPaid float64
	PerEmployeeIncome    float64
}

// ScenarioState is the complete opening state carried between months. Queue
// entries are maintained by the simulator; their aggregate balances are
// mirrored here for reporting and invariant checks.
type ScenarioState struct {
	Month                         int
	Active                        bool
	BankruptcyAbsorbed            bool
	Headcount                     float64
	RampCohorts                   []float64
	CashTotal                     float64
	RestrictedDistributionCash    float64
	RestrictedReserveCash         float64
	DebtBalance                   float64
	ProductiveCapacityRevenue     float64
	MemberCapitalAccountsTotal    float64
	DistributionPeriodAccumulator float64
	DistributionPeriodMonths      int
	IncomeHistory                 []IncomeMonth
	ZeroDistributionStreak        int
	MinimumUnrestrictedCash       float64
	UnpaidMandatoryToDate         float64
	FirstShockMonth               int
}

func (s ScenarioState) RestrictedCashTotal() float64 {
	return s.RestrictedDistributionCash + s.RestrictedReserveCash
}

func (s ScenarioState) UnrestrictedCash() float64 {
	return s.CashTotal - s.RestrictedCashTotal()
}

type RampState struct {
	Begin      []float64 `json:"begin,omitempty"`
	AfterExits []float64 `json:"after_exits,omitempty"`
	Close      []float64 `json:"close,omitempty"`
	FullBegin  float64   `json:"full_productivity_begin"`
	FullAfter  float64   `json:"full_productivity_after_exits"`
	FullClose  float64   `json:"full_productivity_close"`
}

type AllocationAmounts struct {
	EmployeeCashDistribution float64 `json:"employee_cash_distribution"`
	MemberCapitalAllocation  float64 `json:"member_capital_allocation"`
	Reinvestment             float64 `json:"reinvestment"`
	OrganizationalReserve    float64 `json:"organizational_reserve"`
	ExternalDistribution     float64 `json:"external_distribution"`
}

func (a AllocationAmounts) Sum() float64 {
	return a.EmployeeCashDistribution + a.MemberCapitalAllocation + a.Reinvestment + a.OrganizationalReserve + a.ExternalDistribution
}

type RiskFlags struct {
	ReserveBreach           bool `json:"reserve_breach_flag"`
	CashGap                 bool `json:"cash_gap_flag"`
	LiquidityDeficit        bool `json:"liquidity_deficit_flag"`
	CreditLimitBreach       bool `json:"credit_limit_breach_flag"`
	Bankruptcy              bool `json:"bankruptcy_flag"`
	EmployeeDistributionCut bool `json:"employee_distribution_cut_flag"`
	ReinvestmentUnderfunded bool `json:"reinvestment_underfunded_flag"`
}

func (r RiskFlags) Merge(other RiskFlags) RiskFlags {
	return RiskFlags{
		ReserveBreach:           r.ReserveBreach || other.ReserveBreach,
		CashGap:                 r.CashGap || other.CashGap,
		LiquidityDeficit:        r.LiquidityDeficit || other.LiquidityDeficit,
		CreditLimitBreach:       r.CreditLimitBreach || other.CreditLimitBreach,
		Bankruptcy:              r.Bankruptcy || other.Bankruptcy,
		EmployeeDistributionCut: r.EmployeeDistributionCut || other.EmployeeDistributionCut,
		ReinvestmentUnderfunded: r.ReinvestmentUnderfunded || other.ReinvestmentUnderfunded,
	}
}

// MonthlyResult deliberately keeps the accounting and cash bridges explicit.
// Its JSON names are also the canonical monthly CSV column names where the
// specification lists a field.
type MonthlyResult struct {
	Run               int    `json:"run"`
	Month             int    `json:"month"`
	Scenario          string `json:"scenario"`
	SystemType        string `json:"system_type"`
	BehaviorCase      string `json:"behavior_case"`
	MarketCase        string `json:"market_case"`
	ActiveCompanyFlag bool   `json:"active_company_flag"`

	MarketTrend             float64 `json:"market_trend"`
	MarketFactor            float64 `json:"market_factor"`
	SeasonalityMultiplier   float64 `json:"seasonality_multiplier"`
	ShockHappened           bool    `json:"shock_happened"`
	ShockCost               float64 `json:"shock_cost"`
	EffectiveCollectionRate float64 `json:"effective_collection_rate"`
	LaborMarketFactor       float64 `json:"labor_market_factor"`
	CreditMarketFactor      float64 `json:"credit_market_factor"`

	HeadcountBegin      float64   `json:"headcount_begin"`
	VoluntaryLeavers    float64   `json:"voluntary_leavers"`
	Layoffs             float64   `json:"layoffs"`
	Hires               float64   `json:"hires"`
	HeadcountEnd        float64   `json:"headcount_end"`
	Ramp                RampState `json:"ramp"`
	PaidEmployees       float64   `json:"paid_employees"`
	EffectiveEmployees  float64   `json:"effective_employees"`
	DesiredHeadcount    float64   `json:"desired_headcount"`
	TurnoverRateAnnual  float64   `json:"turnover_rate_annual"`
	TurnoverRateMonthly float64   `json:"turnover_rate_monthly"`

	ProductivityUplift            float64 `json:"productivity_uplift"`
	ProductivityMultiplier        float64 `json:"productivity_multiplier"`
	MotivationUpliftRaw           float64 `json:"motivation_uplift_raw"`
	BehavioralTurnoverDeltaAnnual float64 `json:"behavioral_turnover_delta_annual"`
	GovernanceHours               float64 `json:"governance_hours"`
	GovernanceAdminEquivalent     float64 `json:"governance_admin_equivalent_employees"`
	GovernanceCashCost            float64 `json:"governance_cash_cost"`
	DecisionDelayMonths           float64 `json:"decision_delay_months"`
	DecisionQualityMultiplier     float64 `json:"decision_quality_multiplier"`
	FairnessIndex                 float64 `json:"fairness_index"`
	FreeRiderPenalty              float64 `json:"free_rider_penalty"`
	EmployeeRiskConcentration     float64 `json:"employee_risk_concentration_index"`
	IncomeVolatilityIndex12M      float64 `json:"income_volatility_index_12m"`

	MarketDemand                     float64 `json:"market_demand"`
	MarketDemandForecast             float64 `json:"market_demand_forecast"`
	LaborRevenueCapacity             float64 `json:"labor_revenue_capacity"`
	ProductiveCapacityBegin          float64 `json:"productive_capacity_begin"`
	CapacityDepreciation             float64 `json:"capacity_depreciation"`
	CapacityAdditionsDue             float64 `json:"capacity_additions_due"`
	ProductiveCapacityRevenueMonthly float64 `json:"productive_capacity_revenue_monthly"`
	Revenue                          float64 `json:"revenue"`

	SalaryCost                               float64 `json:"salary_cost"`
	SalaryPayrollTax                         float64 `json:"salary_payroll_tax"`
	FixedCosts                               float64 `json:"fixed_costs"`
	VariableCosts                            float64 `json:"variable_costs"`
	HiringCost                               float64 `json:"hiring_cost"`
	ExitCost                                 float64 `json:"exit_cost"`
	LayoffCost                               float64 `json:"layoff_cost"`
	TurnoverAndWorkforceCost                 float64 `json:"turnover_and_workforce_cost"`
	OperatingCostsBeforeAllocation           float64 `json:"operating_costs_before_allocation"`
	OperatingProfitBeforeAllocation          float64 `json:"operating_profit_before_allocation"`
	InterestExpense                          float64 `json:"interest_expense"`
	ProfitBeforeTaxBeforeDistribution        float64 `json:"profit_before_tax_before_distribution"`
	PositiveResultBase                       float64 `json:"positive_result_base"`
	TaxableProfit                            float64 `json:"taxable_profit"`
	ProfitTaxAccrual                         float64 `json:"profit_tax_accrual"`
	NetProfitAfterTaxAndEmployeeDistribution float64 `json:"net_profit_after_tax_and_employee_distribution"`

	OpeningAccountsReceivable float64 `json:"opening_accounts_receivable"`
	CashCollectedCurrent      float64 `json:"cash_collected_current"`
	CashCollectedFromAR       float64 `json:"cash_collected_from_ar"`
	NewAccountsReceivable     float64 `json:"new_accounts_receivable"`
	ClosingAccountsReceivable float64 `json:"closing_accounts_receivable"`

	TaxPayableBegin                     float64 `json:"tax_payable_begin"`
	TaxesDue                            float64 `json:"taxes_due"`
	TaxesPaid                           float64 `json:"taxes_paid"`
	TaxPayableClose                     float64 `json:"tax_payable_close"`
	EmployeeDistributionDueGross        float64 `json:"employee_distribution_due_gross"`
	EmployeeDistributionDuePayrollTax   float64 `json:"employee_distribution_due_payroll_tax"`
	EmployeeCashDistributionPaid        float64 `json:"employee_cash_distribution_paid"`
	EmployeeDistributionPayrollTaxPaid  float64 `json:"employee_distribution_payroll_tax_paid"`
	EmployeeDistributionPayableClose    float64 `json:"employee_distribution_payable_close"`
	MemberCapitalRedemptionPayableBegin float64 `json:"member_capital_redemption_payable_begin"`
	MemberCapitalRedemptionDue          float64 `json:"member_capital_redemption_due"`
	MemberCapitalRedemptionPaid         float64 `json:"member_capital_redemption_paid"`
	MemberCapitalRedemptionPayableClose float64 `json:"member_capital_redemption_payable_close"`

	RequiredCashReserve               float64 `json:"required_cash_reserve"`
	CashTotalBegin                    float64 `json:"cash_total_begin"`
	RestrictedDistributionBegin       float64 `json:"restricted_distribution_cash_begin"`
	RestrictedReserveBegin            float64 `json:"restricted_reserve_cash_begin"`
	UnrestrictedCashBegin             float64 `json:"unrestricted_cash_begin"`
	MandatoryCashScheduled            float64 `json:"mandatory_cash_payments_scheduled"`
	MandatoryCashPayments             float64 `json:"mandatory_cash_payments"`
	GeneralMandatoryArrearsBegin      float64 `json:"general_mandatory_arrears_begin"`
	GeneralMandatoryCurrentScheduled  float64 `json:"general_mandatory_current_scheduled"`
	GeneralMandatoryPayments          float64 `json:"general_mandatory_payments"`
	GeneralMandatoryArrearsClose      float64 `json:"general_mandatory_arrears_close"`
	UnpaidMandatoryObligations        float64 `json:"unpaid_mandatory_obligations"`
	UnpaidMandatoryObligationsToDate  float64 `json:"unpaid_mandatory_obligations_to_date"`
	RestrictedReserveReleased         float64 `json:"restricted_reserve_released"`
	CreditLineLimit                   float64 `json:"credit_line_limit"`
	CreditDrawForLiquidity            float64 `json:"credit_draw_for_liquidity"`
	CashAfterMandatory                float64 `json:"cash_after_mandatory"`
	UnrestrictedCashBeforeAllocations float64 `json:"unrestricted_cash_before_allocations"`
	TaxReserveEstimate                float64 `json:"tax_reserve_estimate"`
	CashSafeAllocationBudget          float64 `json:"cash_safe_allocation_budget"`

	RawAllocations                  AllocationAmounts `json:"raw_allocations"`
	ActualAllocations               AllocationAmounts `json:"actual_allocations"`
	EmployeeCashDistributionAccrued float64           `json:"employee_cash_distribution_accrued"`
	DistributionPayrollTaxAccrued   float64           `json:"distribution_payroll_tax_accrued"`
	RestrictedDistributionCashNew   float64           `json:"restricted_distribution_cash_new"`
	MemberCapitalBegin              float64           `json:"member_capital_begin"`
	MemberCapitalAllocation         float64           `json:"member_capital_allocation"`
	MemberCapitalRedemptionAccrual  float64           `json:"member_capital_redemption_accrual"`
	MemberCapitalClose              float64           `json:"member_capital_close"`
	ReinvestmentCashPaid            float64           `json:"reinvestment_cash_paid"`
	ExternalGrowthCapitalDraw       float64           `json:"external_growth_capital_draw"`
	ExternalGrowthCapitalSpent      float64           `json:"external_growth_capital_spent"`
	ExternalDistributionPaid        float64           `json:"external_distribution_paid"`
	OrganizationalReserveAllocation float64           `json:"organizational_reserve_allocation"`
	CapacityAddedByInvestment       float64           `json:"capacity_added_by_investment"`
	EffectiveInvestmentLag          int               `json:"effective_investment_lag_months"`

	PrincipalDue                float64   `json:"principal_due"`
	PrincipalPaid               float64   `json:"principal_paid"`
	DebtBalanceBegin            float64   `json:"debt_balance_begin"`
	DebtBalanceClose            float64   `json:"debt_balance_close"`
	CashTotalClose              float64   `json:"cash_total_close"`
	RestrictedDistributionClose float64   `json:"restricted_distribution_cash_close"`
	RestrictedReserveClose      float64   `json:"restricted_reserve_cash_close"`
	RestrictedCashClose         float64   `json:"restricted_cash_close"`
	UnrestrictedCashClose       float64   `json:"unrestricted_cash_close"`
	ProductiveCapacityClose     float64   `json:"productive_capacity_close"`
	Risks                       RiskFlags `json:"risks"`
}

type RunTerminalSummary struct {
	Run                                   int       `json:"run"`
	Scenario                              string    `json:"scenario"`
	SystemType                            string    `json:"system_type"`
	BehaviorCase                          string    `json:"behavior_case"`
	MarketCase                            string    `json:"market_case"`
	HorizonMonths                         int       `json:"horizon_months"`
	ActiveCompanyFlag                     bool      `json:"active_company_flag"`
	CumulativeRevenue                     float64   `json:"cumulative_revenue"`
	CumulativeOperatingProfit             float64   `json:"cumulative_operating_profit"`
	CumulativeNetProfit                   float64   `json:"cumulative_net_profit"`
	ProductivityPerEmployee               float64   `json:"productivity_per_employee"`
	TurnoverRateAnnualAverage             float64   `json:"turnover_rate_annual_average"`
	VoluntaryLeaversTotal                 float64   `json:"voluntary_leavers_total"`
	LayoffsTotal                          float64   `json:"layoffs_total"`
	HiresTotal                            float64   `json:"hires_total"`
	HiringAndOnboardingCostsTotal         float64   `json:"hiring_and_onboarding_costs_total"`
	AverageEmployeeIncomeMonthly          float64   `json:"average_employee_income_monthly"`
	EmployeeIncomeVolatility              float64   `json:"employee_income_volatility"`
	RiskAdjustedEmployeeIncome            float64   `json:"risk_adjusted_employee_income"`
	EmployeeCashDistributionTotal         float64   `json:"employee_cash_distribution_total"`
	MemberCapitalAccountsTotal            float64   `json:"member_capital_accounts_total"`
	ReinvestmentTotalCash                 float64   `json:"reinvestment_total_cash"`
	ExternalGrowthCapitalTotal            float64   `json:"external_growth_capital_total"`
	ActualReinvestmentTotal               float64   `json:"actual_reinvestment_total"`
	ReinvestmentUnderfundingRate          float64   `json:"reinvestment_underfunding_rate"`
	ProductiveCapacityAddedTotal          float64   `json:"productive_capacity_added_total"`
	ProductiveCapacityGrowthRate          float64   `json:"productive_capacity_growth_rate"`
	CashEndTotal                          float64   `json:"cash_end_total"`
	CashEndUnrestricted                   float64   `json:"cash_end_unrestricted"`
	MinimumUnrestrictedCash               float64   `json:"minimum_unrestricted_cash"`
	DebtBalance                           float64   `json:"debt_balance"`
	UnpaidObligations                     float64   `json:"unpaid_obligations"`
	MemberCapitalRedemptionDue            float64   `json:"member_capital_redemption_due"`
	FinalHeadcount                        float64   `json:"final_headcount"`
	RevenueCAGR                           float64   `json:"revenue_cagr"`
	CapacityCAGR                          float64   `json:"capacity_cagr"`
	EmployeeRiskConcentrationIndexAverage float64   `json:"employee_risk_concentration_index_average"`
	SustainableDevelopmentValueProxy      float64   `json:"sustainable_development_value_proxy"`
	HadLiquidityDeficit                   bool      `json:"had_liquidity_deficit"`
	HadBankruptcy                         bool      `json:"had_bankruptcy"`
	HadShock                              bool      `json:"had_shock"`
	ShockSurvivalEvaluable                bool      `json:"shock_survival_evaluable"`
	ShockSurvived                         bool      `json:"shock_survived"`
	RiskFlagsEver                         RiskFlags `json:"risk_flags_ever"`
}

type ScenarioSummary struct {
	Scenario                              string   `json:"scenario"`
	SystemType                            string   `json:"system_type"`
	BehaviorCase                          string   `json:"behavior_case"`
	MarketCase                            string   `json:"market_case"`
	HorizonMonths                         int      `json:"horizon_months"`
	Runs                                  int      `json:"runs"`
	MedianCumulativeRevenue               float64  `json:"median_cumulative_revenue"`
	P10CumulativeRevenue                  float64  `json:"p10_cumulative_revenue"`
	P90CumulativeRevenue                  float64  `json:"p90_cumulative_revenue"`
	MedianOperatingProfit                 float64  `json:"median_operating_profit"`
	MedianNetProfit                       float64  `json:"median_net_profit"`
	MedianProductivityPerEmployee         float64  `json:"median_productivity_per_employee"`
	TurnoverRateAnnualAverage             float64  `json:"turnover_rate_annual_average"`
	HiringAndOnboardingCostsTotal         float64  `json:"hiring_and_onboarding_costs_total"`
	AverageEmployeeIncomeMonthly          float64  `json:"average_employee_income_monthly"`
	RiskAdjustedEmployeeIncome            float64  `json:"risk_adjusted_employee_income"`
	EmployeeCashDistributionTotal         float64  `json:"employee_cash_distribution_total"`
	MemberCapitalAccountsTotal            float64  `json:"member_capital_accounts_total"`
	ReinvestmentTotalCash                 float64  `json:"reinvestment_total_cash"`
	ProductiveCapacityGrowthRate          float64  `json:"productive_capacity_growth_rate"`
	CashEndTotalMedian                    float64  `json:"cash_end_total_median"`
	CashEndUnrestrictedP10                float64  `json:"cash_end_unrestricted_p10"`
	MinimumUnrestrictedCashP10            float64  `json:"min_unrestricted_cash_p10"`
	LiquidityDeficitProbability           float64  `json:"liquidity_deficit_probability"`
	BankruptcyProbability                 float64  `json:"bankruptcy_probability"`
	ShockSurvivalRate                     *float64 `json:"shock_survival_rate"`
	FinalHeadcountMedian                  float64  `json:"final_headcount_median"`
	RevenueCAGRMedian                     float64  `json:"revenue_cagr_median"`
	CapacityCAGRMedian                    float64  `json:"capacity_cagr_median"`
	EmployeeRiskConcentrationIndexAverage float64  `json:"employee_risk_concentration_index_average"`
	SustainableDevelopmentValueMedian     float64  `json:"sustainable_development_value_proxy_median"`
	Classification                        string   `json:"classification"`
	AssumptionFlags                       []string `json:"assumption_flags,omitempty"`
}

type PairedDeltaSummary struct {
	Scenario              string  `json:"scenario"`
	BehaviorCase          string  `json:"behavior_case"`
	ReferenceScenario     string  `json:"reference_scenario"`
	ReferenceBehaviorCase string  `json:"reference_behavior_case"`
	MarketCase            string  `json:"market_case"`
	HorizonMonths         int     `json:"horizon_months"`
	Metric                string  `json:"metric"`
	Median                float64 `json:"median"`
	P10                   float64 `json:"p10"`
	P90                   float64 `json:"p90"`
	ProbabilityPositive   float64 `json:"probability_positive"`
	ProbabilityNegative   float64 `json:"probability_negative"`
}

type SimulationResult struct {
	SchemaVersion        string               `json:"schema_version"`
	Mode                 string               `json:"mode"`
	Currency             string               `json:"currency"`
	MarketCase           string               `json:"market_case"`
	RandomSeed           int64                `json:"random_seed"`
	Runs                 int                  `json:"runs"`
	MonthlyResults       []MonthlyResult      `json:"monthly_results,omitempty"`
	RunTerminalSummaries []RunTerminalSummary `json:"run_terminal_summaries,omitempty"`
	TerminalSummaries    []ScenarioSummary    `json:"terminal_summaries"`
	PairedDeltas         []PairedDeltaSummary `json:"paired_deltas,omitempty"`
	SensitivityResults   []SensitivityResult  `json:"sensitivity_results,omitempty"`
	BreakEvenResults     []BreakEvenResult    `json:"break_even_results,omitempty"`
	Warnings             []string             `json:"warnings,omitempty"`
	AssumptionFlags      []string             `json:"assumption_flags,omitempty"`
	ModelLimitations     []string             `json:"model_limitations,omitempty"`
}

type SensitivityResult struct {
	ParameterPath     string  `json:"parameter_path"`
	ParameterValue    float64 `json:"parameter_value"`
	Scenario          string  `json:"scenario"`
	BehaviorCase      string  `json:"behavior_case"`
	HorizonMonths     int     `json:"horizon_months"`
	Metric            string  `json:"metric"`
	MedianValue       float64 `json:"median_value"`
	MedianPairedDelta float64 `json:"median_paired_delta"`
	Classification    string  `json:"classification"`
}

type BreakEvenResult struct {
	Scenario           string   `json:"scenario"`
	BehaviorCase       string   `json:"behavior_case"`
	ReferenceScenario  string   `json:"reference_scenario"`
	HorizonMonths      int      `json:"horizon_months"`
	Metric             string   `json:"metric"`
	ProductivityUplift *float64 `json:"productivity_uplift"`
	Flags              []string `json:"flags,omitempty"`
}
