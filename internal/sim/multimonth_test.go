package sim

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"simworkshare/internal/domain"
)

func TestDeterministicFixedOnlySixtyMonthContract(t *testing.T) {
	cfg := loadDefaultConfig(t)
	baseline, err := RunDeterministicBaseline(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicBaseline() error = %v", err)
	}
	result, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicFixedOnly() error = %v", err)
	}
	if len(result.MonthlyResults) != cfg.Simulation.Months {
		t.Fatalf("monthly result count = %d, want %d", len(result.MonthlyResults), cfg.Simulation.Months)
	}
	if !reflect.DeepEqual(result.MonthlyResults[0], baseline) {
		t.Fatal("month 1 differs from the preserved deterministic baseline")
	}

	var previousCash domain.Money
	var previousAR domain.Money
	var previousTaxPayable domain.Money
	var totalTaxesPaid domain.Money
	var minimumUnrestricted domain.Money
	var risksEver domain.RiskFlags
	for index, month := range result.MonthlyResults {
		monthNumber := index + 1
		if month.Month != monthNumber || month.Environment.Month != monthNumber {
			t.Fatalf("month index = %d/%d, want %d", month.Month, month.Environment.Month, monthNumber)
		}
		assertFloatClose(t, "market trend", month.Environment.CumulativeMarketTrend,
			math.Pow(1+cfg.Environment.MarketGrowthMonthly, float64(index)))
		assertFloatClose(t, "cost inflation", month.Environment.CostInflationFactor,
			math.Pow(1+cfg.Environment.CostInflationMonthly, float64(index)))

		if index > 0 {
			if month.Cash.OpeningCashTotal != previousCash {
				t.Fatalf("month %d opening cash = %.12f, previous close %.12f", monthNumber, month.Cash.OpeningCashTotal, previousCash)
			}
			if month.Cash.AccountsReceivable.Opening != previousAR {
				t.Fatalf("month %d opening AR = %.12f, previous close %.12f", monthNumber, month.Cash.AccountsReceivable.Opening, previousAR)
			}
		}

		expectedTaxClosing := previousTaxPayable - month.Cash.TaxesPaidCash + month.PnL.ProfitTaxAccrual
		assertMoneyClose(t, "tax payable ledger", month.Cash.TaxPayableClosing, expectedTaxClosing)
		if monthNumber == 1 {
			assertMoneyClose(t, "opening AR collection", month.Cash.CashCollectedFromAR, domain.Money(cfg.Company.OpeningAccountsReceivable))
			assertMoneyClose(t, "opening tax payment", month.Cash.TaxesPaidCash, 0)
		} else {
			origin := result.MonthlyResults[monthNumber-cfg.Cashflow.AccountsReceivableLagMonths-1]
			assertMoneyClose(t, "AR paid on due month", month.Cash.CashCollectedFromAR, origin.Cash.AccountsReceivable.New)
			taxOrigin := result.MonthlyResults[monthNumber-cfg.Cashflow.ProfitTaxPaymentLagMonths-1]
			assertMoneyClose(t, "tax paid on due month", month.Cash.TaxesPaidCash, taxOrigin.PnL.ProfitTaxAccrual)
		}

		if month.PnL.BonusExpenseAccrual != 0 || month.PnL.BonusPayrollTaxAccrual != 0 || month.Cash.RestrictedBonusCash != 0 {
			t.Fatalf("month %d created bonus state", monthNumber)
		}
		previousCash = month.Cash.ClosingCashTotal
		previousAR = month.Cash.AccountsReceivable.Closing
		previousTaxPayable = month.Cash.TaxPayableClosing
		totalTaxesPaid += month.Cash.TaxesPaidCash
		if index == 0 || month.Cash.ClosingUnrestrictedCash < minimumUnrestricted {
			minimumUnrestricted = month.Cash.ClosingUnrestrictedCash
		}
		risksEver = mergeRiskFlags(risksEver, month.Risks)
	}

	last := result.MonthlyResults[len(result.MonthlyResults)-1]
	summary := result.TerminalSummary
	if summary.MonthsCompleted != cfg.Simulation.Months || summary.FinalMonth != cfg.Simulation.Months {
		t.Fatalf("summary months = %d/%d", summary.MonthsCompleted, summary.FinalMonth)
	}
	assertMoneyClose(t, "summary closing cash", summary.ClosingCashTotal, last.Cash.ClosingCashTotal)
	assertMoneyClose(t, "summary unrestricted cash", summary.ClosingUnrestrictedCash, last.Cash.ClosingUnrestrictedCash)
	assertMoneyClose(t, "summary minimum unrestricted", summary.MinimumUnrestrictedCash, minimumUnrestricted)
	assertMoneyClose(t, "summary AR", summary.OutstandingAccountsReceivable, last.Cash.AccountsReceivable.Closing)
	assertMoneyClose(t, "summary tax", summary.ClosingTaxPayable, last.Cash.TaxPayableClosing)
	assertMoneyClose(t, "summary taxes paid", summary.TotalTaxesPaidCash, totalTaxesPaid)
	if !reflect.DeepEqual(summary.RiskFlagsEver, risksEver) {
		t.Fatalf("summary risk flags = %+v, want %+v", summary.RiskFlagsEver, risksEver)
	}
}

func TestQueuesHonorIndependentLagsAndPayOnce(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Simulation.Months = 6
	cfg.Cashflow.AccountsReceivableLagMonths = 2
	cfg.Cashflow.ProfitTaxPaymentLagMonths = 3

	result, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicFixedOnly() error = %v", err)
	}
	var totalARCreated domain.Money = domain.Money(cfg.Company.OpeningAccountsReceivable)
	var totalARCollected domain.Money
	var totalTaxAccrued domain.Money
	var totalTaxPaid domain.Money
	for index, month := range result.MonthlyResults {
		monthNumber := index + 1
		var expectedAR domain.Money
		switch {
		case monthNumber == 1:
			expectedAR = domain.Money(cfg.Company.OpeningAccountsReceivable)
		case monthNumber > cfg.Cashflow.AccountsReceivableLagMonths:
			expectedAR = result.MonthlyResults[monthNumber-cfg.Cashflow.AccountsReceivableLagMonths-1].Cash.AccountsReceivable.New
		}
		assertMoneyClose(t, "lagged AR collection", month.Cash.CashCollectedFromAR, expectedAR)

		var expectedTax domain.Money
		if monthNumber > cfg.Cashflow.ProfitTaxPaymentLagMonths {
			expectedTax = result.MonthlyResults[monthNumber-cfg.Cashflow.ProfitTaxPaymentLagMonths-1].PnL.ProfitTaxAccrual
		}
		assertMoneyClose(t, "lagged tax payment", month.Cash.TaxesPaidCash, expectedTax)

		totalARCreated += month.Cash.AccountsReceivable.New
		totalARCollected += month.Cash.CashCollectedFromAR
		totalTaxAccrued += month.PnL.ProfitTaxAccrual
		totalTaxPaid += month.Cash.TaxesPaidCash
	}
	assertMoneyClose(t, "AR horizon balance", result.TerminalSummary.OutstandingAccountsReceivable, totalARCreated-totalARCollected)
	assertMoneyClose(t, "tax horizon balance", result.TerminalSummary.ClosingTaxPayable, totalTaxAccrued-totalTaxPaid)
	if result.TerminalSummary.OutstandingAccountsReceivable <= 0 || result.TerminalSummary.ClosingTaxPayable <= 0 {
		t.Fatal("queue tails were force-drained at the horizon")
	}
}

func TestAccountsReceivableQueueUsesNetOfBadDebt(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Simulation.Months = 3
	cfg.Cashflow.AccountsReceivableLagMonths = 1
	cfg.Cashflow.BadDebtRate = 0.25

	result, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicFixedOnly() error = %v", err)
	}
	for index, month := range result.MonthlyResults {
		grossDeferred := month.PnL.Revenue * domain.Money(1-cfg.Cashflow.RevenueCollectionRateCurrentMonth)
		expectedNetAR := grossDeferred * domain.Money(1-cfg.Cashflow.BadDebtRate)
		assertMoneyClose(t, "new AR net of bad debt", month.Cash.AccountsReceivable.New, expectedNetAR)
		if month.Cash.AccountsReceivable.New >= grossDeferred {
			t.Fatalf("month %d bad-debt haircut was not applied", month.Month)
		}
		if index > 0 {
			assertMoneyClose(t, "due AR remains net of bad debt", month.Cash.CashCollectedFromAR,
				result.MonthlyResults[index-1].Cash.AccountsReceivable.New)
		}
	}
	last := result.MonthlyResults[len(result.MonthlyResults)-1]
	assertMoneyClose(t, "terminal net AR", result.TerminalSummary.OutstandingAccountsReceivable,
		last.Cash.AccountsReceivable.New)
}

func TestTerminalRisksPersistAndBankruptcyDoesNotStopHorizon(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Simulation.Months = 3
	cfg.Company.StartingCash = -2_000_000

	result, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicFixedOnly() error = %v", err)
	}
	if len(result.MonthlyResults) != cfg.Simulation.Months || result.TerminalSummary.MonthsCompleted != cfg.Simulation.Months {
		t.Fatalf("bankruptcy stopped horizon: results/summary = %d/%d, want %d",
			len(result.MonthlyResults), result.TerminalSummary.MonthsCompleted, cfg.Simulation.Months)
	}
	if !result.MonthlyResults[0].Risks.Bankruptcy {
		t.Fatal("fixture did not create bankruptcy in month 1")
	}
	last := result.MonthlyResults[len(result.MonthlyResults)-1]
	if last.Risks.Bankruptcy {
		t.Fatal("fixture did not recover from bankruptcy before the terminal month")
	}
	if !result.TerminalSummary.RiskFlagsEver.Bankruptcy || !result.TerminalSummary.RiskFlagsEver.CashGap {
		t.Fatalf("terminal risk history lost an earlier breach: %+v", result.TerminalSummary.RiskFlagsEver)
	}
	assertMoneyClose(t, "minimum unrestricted cash after recovery",
		result.TerminalSummary.MinimumUnrestrictedCash, result.MonthlyResults[0].Cash.ClosingUnrestrictedCash)
}

func TestDeterministicFixedOnlyIsRepeatableAndIgnoresMonteCarloControls(t *testing.T) {
	cfg := loadDefaultConfig(t)
	first, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("first run error = %v", err)
	}
	second, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("second run error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated full results differ")
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first result: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second result: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("repeated full JSON differs")
	}

	cfg.Simulation.Runs = 17
	cfg.Simulation.RandomSeed = 999
	cfg.Simulation.CommonRandomNumbers = false
	changedControls, err := RunDeterministicFixedOnly(cfg)
	if err != nil {
		t.Fatalf("run with changed Monte Carlo controls: %v", err)
	}
	if !reflect.DeepEqual(first, changedControls) {
		t.Fatal("unused Monte Carlo controls changed deterministic output")
	}
}

func TestDeterministicFixedOnlyRejectsNonFiniteLaterMonth(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Simulation.Months = 2
	cfg.Environment.CostInflationMonthly = math.MaxFloat64
	if _, err := RunDeterministicBaseline(cfg); err != nil {
		t.Fatalf("month 1 should remain finite: %v", err)
	}
	_, err := RunDeterministicFixedOnly(cfg)
	if err == nil || !strings.Contains(err.Error(), "month 2: integrity:") || !strings.Contains(err.Error(), "is not finite") {
		t.Fatalf("RunDeterministicFixedOnly() error = %v, want month 2 non-finite integrity error", err)
	}
}

func assertMoneyClose(t *testing.T, name string, got, want domain.Money) {
	t.Helper()
	if !domain.MoneyAlmostEqual(got, want) {
		t.Fatalf("%s = %.12f, want %.12f", name, got, want)
	}
}

func assertFloatClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	scale := math.Max(1, math.Max(math.Abs(got), math.Abs(want)))
	if math.Abs(got-want) > domain.RelativeTolerance*scale {
		t.Fatalf("%s = %.18f, want %.18f", name, got, want)
	}
}
