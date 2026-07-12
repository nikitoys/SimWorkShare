package sim

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type goldenMonthV04 struct {
	Scenario                        string  `json:"scenario"`
	BehaviorCase                    string  `json:"behavior_case"`
	Revenue                         float64 `json:"revenue"`
	OperatingProfitBeforeAllocation float64 `json:"operating_profit_before_allocation"`
	ProfitTaxAccrual                float64 `json:"profit_tax_accrual"`
	EmployeeCashDistributionAccrued float64 `json:"employee_cash_distribution_accrued"`
	MemberCapitalAllocation         float64 `json:"member_capital_allocation"`
	ReinvestmentCashPaid            float64 `json:"reinvestment_cash_paid"`
	CashTotalClose                  float64 `json:"cash_total_close"`
	RestrictedDistributionClose     float64 `json:"restricted_distribution_cash_close"`
	RestrictedReserveClose          float64 `json:"restricted_reserve_cash_close"`
	DebtBalanceClose                float64 `json:"debt_balance_close"`
	ProductiveCapacityClose         float64 `json:"productive_capacity_close"`
	HeadcountEnd                    float64 `json:"headcount_end"`
}

func TestDefaultV04MonthOneGolden(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	result, err := Run(cfg, RunOptions{
		BehaviorCaseNames:   []string{"no_effect"},
		StoreMonthlyResults: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	gotRows := make([]goldenMonthV04, 0, len(result.MonthlyResults))
	for _, month := range result.MonthlyResults {
		gotRows = append(gotRows, goldenMonthV04{
			Scenario:                        month.Scenario,
			BehaviorCase:                    month.BehaviorCase,
			Revenue:                         month.Revenue,
			OperatingProfitBeforeAllocation: month.OperatingProfitBeforeAllocation,
			ProfitTaxAccrual:                month.ProfitTaxAccrual,
			EmployeeCashDistributionAccrued: month.EmployeeCashDistributionAccrued,
			MemberCapitalAllocation:         month.MemberCapitalAllocation,
			ReinvestmentCashPaid:            month.ReinvestmentCashPaid,
			CashTotalClose:                  month.CashTotalClose,
			RestrictedDistributionClose:     month.RestrictedDistributionClose,
			RestrictedReserveClose:          month.RestrictedReserveClose,
			DebtBalanceClose:                month.DebtBalanceClose,
			ProductiveCapacityClose:         month.ProductiveCapacityClose,
			HeadcountEnd:                    month.HeadcountEnd,
		})
	}
	got, err := json.Marshal(gotRows)
	if err != nil {
		t.Fatal(err)
	}
	wantRaw, err := os.ReadFile(filepath.Join("testdata", "month1_no_effect_v04_golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := json.Compact(&want, wantRaw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatalf("v0.4 month-one golden changed\ngot:  %s\nwant: %s", got, want.Bytes())
	}
}
