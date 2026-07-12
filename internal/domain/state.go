package domain

// EnvironmentMonth is a fully materialized environment input. The current
// runner constructs it deterministically; a stochastic generator is
// intentionally not part of this implementation.
type EnvironmentMonth struct {
	Month                    int     `json:"month"`
	CumulativeMarketTrend    float64 `json:"cumulative_market_trend"`
	MarketFactor             float64 `json:"market_factor"`
	CostInflationFactor      float64 `json:"cost_inflation_factor"`
	LaborMarketFactor        float64 `json:"labor_market_factor"`
	ShockHappened            bool    `json:"shock_happened"`
	ShockRevenueMultiplier   float64 `json:"shock_revenue_multiplier"`
	ShockCost                Money   `json:"shock_cost"`
	CollectionRateMultiplier float64 `json:"collection_rate_multiplier"`
}

type WorkforceState struct {
	EmployeesCount                int     `json:"employees_count"`
	TurnoverRateAnnual            float64 `json:"turnover_rate_annual"`
	TurnoverRateMonthly           float64 `json:"turnover_rate_monthly"`
	LeaversCount                  float64 `json:"leavers_count"`
	TurnoverProductivityLoss      float64 `json:"turnover_productivity_loss"`
	ProductivityUplift            float64 `json:"productivity_uplift"`
	ProductivityLevel             float64 `json:"productivity_level"`
	HighPerformerAttritionWarning bool    `json:"high_performer_attrition_warning"`
}

type PnLState struct {
	Revenue                    Money `json:"revenue"`
	SalaryCosts                Money `json:"salary_costs"`
	FixedCosts                 Money `json:"fixed_costs"`
	VariableCosts              Money `json:"variable_costs"`
	TurnoverCost               Money `json:"turnover_cost"`
	ShockCost                  Money `json:"shock_cost"`
	TotalCostsBeforeBonus      Money `json:"total_costs_before_bonus"`
	OperatingProfitBeforeBonus Money `json:"operating_profit_before_bonus"`
	BonusExpenseAccrual        Money `json:"bonus_expense_accrual"`
	BonusPayrollTaxAccrual     Money `json:"bonus_payroll_tax_accrual"`
	ProfitAfterBonusBeforeTax  Money `json:"profit_after_bonus_before_tax"`
	ProfitTaxAccrual           Money `json:"profit_tax_accrual"`
	AccountingProfitAfterBonus Money `json:"accounting_profit_after_bonus"`
}

type AccountsReceivableState struct {
	Opening             Money `json:"opening"`
	Collected           Money `json:"collected"`
	New                 Money `json:"new"`
	Closing             Money `json:"closing"`
	CollectionLagMonths int   `json:"collection_lag_months"`
}

// CashMonthInput contains opening balances and obligations already due in the
// current month. New AR, bonus, and tax accruals are scheduled after settlement.
type CashMonthInput struct {
	OpeningCashTotal           Money
	OpeningAccountsReceivable  Money
	CashCollectedFromAR        Money
	OpeningTaxPayable          Money
	TaxesPaidCash              Money
	OpeningRestrictedBonusCash Money
	BonusDueGross              Money
	BonusDuePayrollTax         Money
}

type CashState struct {
	OpeningCashTotal         Money                   `json:"opening_cash_total"`
	CashCollectedCurrent     Money                   `json:"cash_collected_current"`
	CashCollectedFromAR      Money                   `json:"cash_collected_from_ar"`
	CashCollectedFromRevenue Money                   `json:"cash_collected_from_revenue"`
	AccountsReceivable       AccountsReceivableState `json:"accounts_receivable"`
	DueCashPayments          Money                   `json:"due_cash_payments"`
	TaxesPaidCash            Money                   `json:"taxes_paid_cash"`
	ClosingCashTotal         Money                   `json:"closing_cash_total"`
	RestrictedBonusCash      Money                   `json:"restricted_bonus_cash"`
	ClosingUnrestrictedCash  Money                   `json:"closing_unrestricted_cash"`
	RequiredCashReserve      Money                   `json:"required_cash_reserve"`
	OwnerDistributableCash   Money                   `json:"owner_distributable_cash"`
	TaxPayableClosing        Money                   `json:"tax_payable_closing"`
}

type RiskFlags struct {
	ReserveBreach            bool `json:"reserve_breach"`
	LiquidityGap             bool `json:"liquidity_gap"`
	CashGap                  bool `json:"cash_gap"`
	Bankruptcy               bool `json:"bankruptcy"`
	BonusNotAccruedDueToCash bool `json:"bonus_not_accrued_due_to_cash,omitempty"`
	ZeroBonusPeriod          bool `json:"zero_bonus_period,omitempty"`
}

// CompensationState is emitted for profit_share runs only. Monetary bonus
// values distinguish employee gross amounts from the employer payroll-tax
// component so accruals and later cash payments cannot be double counted.
type CompensationState struct {
	BonusPeriod                       string `json:"bonus_period"`
	IsBonusPeriodEnd                  bool   `json:"is_bonus_period_end"`
	MonthsInBonusPeriod               int    `json:"months_in_bonus_period"`
	PeriodProfitBaseAccumulator       Money  `json:"period_profit_base_accumulator"`
	PeriodProfitBase                  Money  `json:"period_profit_base"`
	CashBaseForTotalEmployerBonusCost Money  `json:"cash_base_for_total_employer_bonus_cost"`
	DistributableBase                 Money  `json:"distributable_base"`
	GrossBonusPoolAccrued             Money  `json:"gross_bonus_pool_accrued"`
	BonusPayrollTaxAccrued            Money  `json:"bonus_payroll_tax_accrued"`
	TotalBonusEmployerCostAccrued     Money  `json:"total_bonus_employer_cost_accrued"`
	BonusPerEmployeeAccrued           Money  `json:"bonus_per_employee_accrued"`
	OpeningBonusPayableGross          Money  `json:"opening_bonus_payable_gross"`
	OpeningBonusPayablePayrollTax     Money  `json:"opening_bonus_payable_payroll_tax"`
	OpeningBonusPayable               Money  `json:"opening_bonus_payable"`
	BonusPaidCash                     Money  `json:"bonus_paid_cash"`
	BonusPayrollTaxPaid               Money  `json:"bonus_payroll_tax_paid"`
	ClosingBonusPayableGross          Money  `json:"closing_bonus_payable_gross"`
	ClosingBonusPayablePayrollTax     Money  `json:"closing_bonus_payable_payroll_tax"`
	ClosingBonusPayable               Money  `json:"closing_bonus_payable"`
}
