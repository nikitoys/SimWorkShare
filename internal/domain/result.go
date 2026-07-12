package domain

type MonthlyResult struct {
	Scenario        string             `json:"scenario"`
	BehaviorCase    string             `json:"behavior_case"`
	EnvironmentCase string             `json:"environment_case"`
	Currency        string             `json:"currency"`
	Month           int                `json:"month"`
	Environment     EnvironmentMonth   `json:"environment"`
	Workforce       WorkforceState     `json:"workforce"`
	PnL             PnLState           `json:"pnl"`
	Cash            CashState          `json:"cash"`
	Compensation    *CompensationState `json:"compensation,omitempty"`
	Risks           RiskFlags          `json:"risks"`
}

type CompensationSummary struct {
	TotalGrossBonusAccrued          Money `json:"total_gross_bonus_accrued"`
	TotalBonusPayrollTaxAccrued     Money `json:"total_bonus_payroll_tax_accrued"`
	TotalEmployerBonusCostAccrued   Money `json:"total_employer_bonus_cost_accrued"`
	TotalGrossBonusPaid             Money `json:"total_gross_bonus_paid"`
	TotalBonusPayrollTaxPaid        Money `json:"total_bonus_payroll_tax_paid"`
	ClosingBonusPayableGross        Money `json:"closing_bonus_payable_gross"`
	ClosingBonusPayablePayrollTax   Money `json:"closing_bonus_payable_payroll_tax"`
	ClosingBonusPayable             Money `json:"closing_bonus_payable"`
	ClosingRestrictedBonusCash      Money `json:"closing_restricted_bonus_cash"`
	UnclosedPeriodProfitAccumulator Money `json:"unclosed_period_profit_accumulator"`
}

type TerminalSummary struct {
	MonthsCompleted               int                  `json:"months_completed"`
	FinalMonth                    int                  `json:"final_month"`
	ClosingCashTotal              Money                `json:"closing_cash_total"`
	ClosingUnrestrictedCash       Money                `json:"closing_unrestricted_cash"`
	MinimumUnrestrictedCash       Money                `json:"minimum_unrestricted_cash"`
	OutstandingAccountsReceivable Money                `json:"outstanding_accounts_receivable"`
	ClosingTaxPayable             Money                `json:"closing_tax_payable"`
	TotalTaxesPaidCash            Money                `json:"total_taxes_paid_cash"`
	RiskFlagsEver                 RiskFlags            `json:"risk_flags_ever"`
	Compensation                  *CompensationSummary `json:"compensation,omitempty"`
}

type ComparisonSummary struct {
	ProfitShareScenario              string `json:"profit_share_scenario"`
	ProfitShareBehaviorCase          string `json:"profit_share_behavior_case"`
	FinalClosingCashDelta            Money  `json:"final_closing_cash_delta"`
	FinalUnrestrictedCashDelta       Money  `json:"final_unrestricted_cash_delta"`
	FinalOwnerDistributableCashDelta Money  `json:"final_owner_distributable_cash_delta"`
	CumulativeAccountingProfitDelta  Money  `json:"cumulative_accounting_profit_delta"`
	TotalGrossBonusAccrued           Money  `json:"total_gross_bonus_accrued"`
	TotalEmployerBonusCostAccrued    Money  `json:"total_employer_bonus_cost_accrued"`
	TotalGrossBonusPaid              Money  `json:"total_gross_bonus_paid"`
	TotalEmployerBonusCostPaid       Money  `json:"total_employer_bonus_cost_paid"`
}

type ComparisonResult struct {
	Currency    string            `json:"currency"`
	FixedOnly   SimulationResult  `json:"fixed_only"`
	ProfitShare SimulationResult  `json:"profit_share"`
	Summary     ComparisonSummary `json:"summary"`
}

type SimulationResult struct {
	Scenario        string          `json:"scenario"`
	BehaviorCase    string          `json:"behavior_case"`
	EnvironmentCase string          `json:"environment_case"`
	Currency        string          `json:"currency"`
	MonthlyResults  []MonthlyResult `json:"monthly_results"`
	TerminalSummary TerminalSummary `json:"terminal_summary"`
}
