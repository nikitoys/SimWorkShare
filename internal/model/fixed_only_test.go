package model

import (
	"math"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

func TestBaselineFixedOnlySanity(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Workforce.RecruitingCostPerLeaver = 0
	cfg.Workforce.OnboardingCostPerLeaver = 0
	cfg.Workforce.ManagerTimeCostPerLeaver = 0
	cfg.Workforce.LostProductivityCostPerLeaver = 0

	env := neutralEnvironment(cfg)
	workforce := CalculateWorkforce(cfg, cfg.BehaviorCases["no_effect"], env)
	pnl := CalculateOperatingPnL(cfg, workforce, env)

	assertMoney(t, "revenue", pnl.Revenue, 11_500_000)
	assertMoney(t, "salary", pnl.SalaryCosts, 5_000_000)
	assertMoney(t, "variable", pnl.VariableCosts, 2_875_000)
	assertMoney(t, "fixed", pnl.FixedCosts, 2_000_000)
	assertMoney(t, "turnover", pnl.TurnoverCost, 0)
	assertMoney(t, "operating profit", pnl.OperatingProfitBeforeBonus, 1_625_000)
}

func TestDefaultMonthIncludesDeterministicTurnover(t *testing.T) {
	cfg := loadDefaultConfig(t)
	env := neutralEnvironment(cfg)
	workforce := CalculateWorkforce(cfg, cfg.BehaviorCases["no_effect"], env)
	pnl := CalculateOperatingPnL(cfg, workforce, env)

	assertFloat(t, "annual turnover", workforce.TurnoverRateAnnual, 0.2)
	assertFloat(t, "monthly turnover", workforce.TurnoverRateMonthly, 0.018423470126248342)
	assertFloat(t, "leavers", workforce.LeaversCount, 0.9211735063124171)
	assertFloat(t, "productivity uplift", workforce.ProductivityUplift, 0)
	assertMoney(t, "turnover cost", pnl.TurnoverCost, 115_146.68828905214)
	assertMoney(t, "operating profit", pnl.OperatingProfitBeforeBonus, 1_509_853.311710948)
}

func TestMonthOneCashIdentity(t *testing.T) {
	cfg := loadDefaultConfig(t)
	env := neutralEnvironment(cfg)
	workforce := CalculateWorkforce(cfg, cfg.BehaviorCases["no_effect"], env)
	operatingPnL := CalculateOperatingPnL(cfg, workforce, env)
	cashBeforeTaxAccrual := CalculateCash(cfg, operatingPnL, env, firstMonthCashInput(cfg))
	if cashBeforeTaxAccrual.TaxPayableClosing != 0 {
		t.Fatal("cash settlement recorded current tax before the tax-accrual step")
	}
	pnl := AccrueProfitTax(cfg, operatingPnL)
	cash := RecordProfitTaxPayable(cashBeforeTaxAccrual, pnl.ProfitTaxAccrual)
	risks := CalculateRiskFlags(cfg, cash)

	assertMoney(t, "current collections", cash.CashCollectedCurrent, 9_775_000)
	assertMoney(t, "opening AR collections", cash.CashCollectedFromAR, 1_725_000)
	assertMoney(t, "new AR", cash.AccountsReceivable.New, 1_725_000)
	assertMoney(t, "closing cash", cash.ClosingCashTotal, 16_509_853.311710948)
	assertMoney(t, "required reserve", cash.RequiredCashReserve, 14_000_000)
	assertMoney(t, "owner distributable", cash.OwnerDistributableCash, 2_509_853.311710948)
	assertMoney(t, "tax payable", cash.TaxPayableClosing, pnl.ProfitTaxAccrual)
	assertMoney(t, "taxes paid", cash.TaxesPaidCash, 0)
	if risks.ReserveBreach || risks.LiquidityGap || risks.CashGap || risks.Bankruptcy {
		t.Fatalf("unexpected risk flags: %+v", risks)
	}

	expectedOperatingProfit := pnl.Revenue - pnl.TotalCostsBeforeBonus
	assertMoney(t, "P&L identity", pnl.OperatingProfitBeforeBonus, expectedOperatingProfit)
	expectedTotalCosts := pnl.SalaryCosts + pnl.FixedCosts + pnl.VariableCosts + pnl.TurnoverCost + pnl.ShockCost
	assertMoney(t, "total-cost identity", pnl.TotalCostsBeforeBonus, expectedTotalCosts)
	assertMoney(t, "fixed-only pre-tax identity", pnl.ProfitAfterBonusBeforeTax, pnl.OperatingProfitBeforeBonus)
	assertMoney(t, "after-tax identity", pnl.AccountingProfitAfterBonus, pnl.ProfitAfterBonusBeforeTax-pnl.ProfitTaxAccrual)
	assertMoney(t, "fixed-only bonus expense", pnl.BonusExpenseAccrual, 0)
	assertMoney(t, "fixed-only bonus payroll tax", pnl.BonusPayrollTaxAccrual, 0)
	expectedDuePayments := pnl.TotalCostsBeforeBonus + domain.Money(cfg.Cashflow.DebtServiceMonthly+cfg.Cashflow.CapexMonthly) + cash.TaxesPaidCash
	assertMoney(t, "due-payment identity", cash.DueCashPayments, expectedDuePayments)
	expectedClosing := cash.OpeningCashTotal + cash.CashCollectedFromRevenue - cash.DueCashPayments
	assertMoney(t, "cash identity", cash.ClosingCashTotal, expectedClosing)
	assertMoney(t, "tax accrual does not change cash", cash.ClosingCashTotal, cashBeforeTaxAccrual.ClosingCashTotal)
	assertMoney(t, "collection identity", cash.CashCollectedFromRevenue, cash.CashCollectedCurrent+cash.CashCollectedFromAR)
	assertMoney(t, "AR ledger identity", cash.AccountsReceivable.Closing, cash.AccountsReceivable.Opening+cash.AccountsReceivable.New-cash.AccountsReceivable.Collected)
	if cash.AccountsReceivable.CollectionLagMonths != cfg.Cashflow.AccountsReceivableLagMonths {
		t.Fatalf("AR lag = %d, want %d", cash.AccountsReceivable.CollectionLagMonths, cfg.Cashflow.AccountsReceivableLagMonths)
	}
}

func TestDebtAndCapexAffectCashOnceButNotPnL(t *testing.T) {
	cfg := loadDefaultConfig(t)
	env := neutralEnvironment(cfg)
	workforce := CalculateWorkforce(cfg, cfg.BehaviorCases["no_effect"], env)
	basePnL := CalculateOperatingPnL(cfg, workforce, env)
	baseCash := CalculateCash(cfg, basePnL, env, firstMonthCashInput(cfg))

	cfg.Cashflow.DebtServiceMonthly = 123
	cfg.Cashflow.CapexMonthly = 456
	updatedPnL := CalculateOperatingPnL(cfg, workforce, env)
	if !reflect.DeepEqual(basePnL, updatedPnL) {
		t.Fatalf("debt/CAPEX leaked into P&L:\nbase=%+v\nupdated=%+v", basePnL, updatedPnL)
	}
	cash := CalculateCash(cfg, updatedPnL, env, firstMonthCashInput(cfg))

	assertMoney(t, "debt+capex payment delta", cash.DueCashPayments-baseCash.DueCashPayments, 579)
	assertMoney(t, "debt+capex cash delta", baseCash.ClosingCashTotal-cash.ClosingCashTotal, 579)
	assertMoney(t, "debt reserve delta", cash.RequiredCashReserve-baseCash.RequiredCashReserve, domain.Money(cfg.Company.RequiredCashReserveMonths*123))
	assertMoney(t, "operating P&L unchanged", updatedPnL.OperatingProfitBeforeBonus, 1_509_853.311710948)
}

func TestCashRiskFlagsUseMoneyTolerance(t *testing.T) {
	cfg := loadDefaultConfig(t)
	env := neutralEnvironment(cfg)
	workforce := CalculateWorkforce(cfg, cfg.BehaviorCases["no_effect"], env)
	pnl := CalculateOperatingPnL(cfg, workforce, env)

	collections := float64(pnl.Revenue)*cfg.Cashflow.RevenueCollectionRateCurrentMonth + cfg.Company.OpeningAccountsReceivable
	due := float64(pnl.SalaryCosts+pnl.FixedCosts+pnl.VariableCosts+pnl.TurnoverCost+pnl.ShockCost) +
		cfg.Cashflow.DebtServiceMonthly + cfg.Cashflow.CapexMonthly
	requiredReserve := cfg.Company.RequiredCashReserveMonths *
		(float64(pnl.SalaryCosts) + float64(pnl.FixedCosts) + cfg.Cashflow.DebtServiceMonthly)

	cfg.Company.StartingCash = requiredReserve - collections + due - 1e-7
	nearBoundary := CalculateCash(cfg, pnl, env, firstMonthCashInput(cfg))
	nearRisks := CalculateRiskFlags(cfg, nearBoundary)
	if nearRisks.ReserveBreach {
		t.Fatal("reserve breach was set for boundary noise inside money tolerance")
	}
	assertMoney(t, "near-boundary owner cash", nearBoundary.OwnerDistributableCash, 0)

	cfg.Company.StartingCash = requiredReserve - collections + due - 0.02
	materialCash := CalculateCash(cfg, pnl, env, firstMonthCashInput(cfg))
	materialRisks := CalculateRiskFlags(cfg, materialCash)
	if !materialRisks.ReserveBreach {
		t.Fatal("reserve breach was not set for a material shortfall")
	}
}

func TestCashRiskFlagHierarchy(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Cashflow.AvailableCreditLine = 1_000
	env := neutralEnvironment(cfg)
	workforce := CalculateWorkforce(cfg, cfg.BehaviorCases["no_effect"], env)
	pnl := CalculateOperatingPnL(cfg, workforce, env)
	collections := float64(pnl.Revenue)*cfg.Cashflow.RevenueCollectionRateCurrentMonth + cfg.Company.OpeningAccountsReceivable
	due := float64(pnl.TotalCostsBeforeBonus) + cfg.Cashflow.DebtServiceMonthly + cfg.Cashflow.CapexMonthly

	cfg.Company.StartingCash = -collections + due - 500
	withinLine := CalculateRiskFlags(cfg, CalculateCash(cfg, pnl, env, firstMonthCashInput(cfg)))
	if !withinLine.ReserveBreach || !withinLine.LiquidityGap || !withinLine.CashGap || withinLine.Bankruptcy {
		t.Fatalf("risk flags within credit line = %+v", withinLine)
	}

	cfg.Company.StartingCash = -collections + due - 1_500
	beyondLine := CalculateRiskFlags(cfg, CalculateCash(cfg, pnl, env, firstMonthCashInput(cfg)))
	if !beyondLine.ReserveBreach || !beyondLine.LiquidityGap || !beyondLine.CashGap || !beyondLine.Bankruptcy {
		t.Fatalf("risk flags beyond credit line = %+v", beyondLine)
	}
}

func neutralEnvironment(cfg config.Config) domain.EnvironmentMonth {
	return domain.EnvironmentMonth{
		Month:                    1,
		CumulativeMarketTrend:    1,
		MarketFactor:             1,
		CostInflationFactor:      1,
		LaborMarketFactor:        cfg.Environment.LaborMarketFactor,
		ShockRevenueMultiplier:   cfg.Environment.ShockRevenueMultiplier,
		CollectionRateMultiplier: 1,
	}
}

func firstMonthCashInput(cfg config.Config) domain.CashMonthInput {
	return domain.CashMonthInput{
		OpeningCashTotal:          domain.Money(cfg.Company.StartingCash),
		OpeningAccountsReceivable: domain.Money(cfg.Company.OpeningAccountsReceivable),
		CashCollectedFromAR:       domain.Money(cfg.Company.OpeningAccountsReceivable),
	}
}

func loadDefaultConfig(t *testing.T) config.Config {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	cfg, err := config.LoadFile(filepath.Join(root, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	return cfg
}

func assertMoney(t *testing.T, name string, got, want domain.Money) {
	t.Helper()
	if !domain.MoneyAlmostEqual(got, want) {
		t.Fatalf("%s = %.12f, want %.12f", name, got, want)
	}
}

func assertFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	scale := math.Max(1, math.Max(math.Abs(got), math.Abs(want)))
	if math.Abs(got-want) > domain.RelativeTolerance*scale {
		t.Fatalf("%s = %.18f, want %.18f", name, got, want)
	}
}
