package model

import (
	"testing"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

func TestCanonicalProfitShareMonthOneAccrual(t *testing.T) {
	cfg := loadDefaultConfig(t)
	scenario := scenarioForTest(t, cfg, "profit_share_equal_10")
	env := neutralEnvironment(cfg)
	workforce := CalculateWorkforce(cfg, cfg.BehaviorCases["no_effect"], env)
	pnl := CalculateOperatingPnL(cfg, workforce, env)
	cashBeforeBonus := CalculateCash(cfg, pnl, env, firstMonthCashInput(cfg))

	pnl, cash, compensation, err := ApplyProfitShareAccrual(
		cfg,
		scenario,
		pnl,
		cashBeforeBonus,
		ProfitShareAccrualInput{
			PeriodProfitBaseAccumulator: pnl.OperatingProfitBeforeBonus,
			MonthsInBonusPeriod:         1,
			IsBonusPeriodEnd:            true,
		},
	)
	if err != nil {
		t.Fatalf("ApplyProfitShareAccrual() error = %v", err)
	}
	pnl = AccrueProfitTax(cfg, pnl)

	assertMoney(t, "gross bonus", compensation.GrossBonusPoolAccrued, 150_985.33117109482)
	assertMoney(t, "payroll tax", compensation.BonusPayrollTaxAccrued, 0)
	assertMoney(t, "employer cost", compensation.TotalBonusEmployerCostAccrued, 150_985.33117109482)
	assertMoney(t, "per employee", compensation.BonusPerEmployeeAccrued, 3_019.7066234218965)
	assertMoney(t, "profit tax", pnl.ProfitTaxAccrual, 271_773.5961079707)
	assertMoney(t, "accrual leaves total cash unchanged", cash.ClosingCashTotal, cashBeforeBonus.ClosingCashTotal)
	assertMoney(t, "restricted cash", cash.RestrictedBonusCash, 150_985.33117109482)
	assertMoney(t, "unrestricted cash", cash.ClosingUnrestrictedCash, 16_358_867.980539853)
}

func TestProfitSharePolicyLimits(t *testing.T) {
	baseCfg := loadDefaultConfig(t)
	baseCfg.Cashflow.BonusPayrollTaxRate = 0.25
	baseCfg.Cashflow.ReserveCurrentProfitTax = false
	baseCfg.Cashflow.PlannedReinvestmentRate = 0
	baseCfg.Company.RequiredCashReserveMonths = 0
	baseScenario := scenarioForTest(t, baseCfg, "profit_share_equal_10")
	share := 0.20
	hurdle := 200.0
	eligible := 20
	baseScenario.ProfitSharePercent = &share
	baseScenario.ProfitHurdleMonthly = &hurdle
	baseScenario.EligibleEmployeesCount = &eligible
	baseScenario.BonusCapTotal = nil
	baseScenario.BonusCapPerEmployee = nil

	tests := []struct {
		name           string
		cash           domain.Money
		hurdle         float64
		capTotal       *float64
		capPerEmployee *float64
		wantGross      domain.Money
	}{
		{name: "percentage and hurdle", cash: 10_000, hurdle: 200, wantGross: 160},
		{name: "hurdle blocks bonus", cash: 10_000, hurdle: 1_000, wantGross: 0},
		{name: "total cap", cash: 10_000, hurdle: 0, capTotal: floatPointer(100), wantGross: 100},
		{name: "per employee cap", cash: 10_000, hurdle: 0, capPerEmployee: floatPointer(3), wantGross: 60},
		{name: "cash affordability", cash: 125, hurdle: 0, wantGross: 100},
		{name: "zero cash base", cash: 0, hurdle: 0, wantGross: 0},
		{name: "explicit zero cap", cash: 10_000, hurdle: 0, capTotal: floatPointer(0), wantGross: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := baseScenario
			scenario.ProfitHurdleMonthly = floatPointer(tt.hurdle)
			scenario.BonusCapTotal = tt.capTotal
			scenario.BonusCapPerEmployee = tt.capPerEmployee
			pnl := domain.PnLState{
				OperatingProfitBeforeBonus: 1_000,
				ProfitAfterBonusBeforeTax:  1_000,
			}
			cash := domain.CashState{
				ClosingCashTotal:        tt.cash,
				ClosingUnrestrictedCash: tt.cash,
			}
			_, _, compensation, err := ApplyProfitShareAccrual(
				baseCfg,
				scenario,
				pnl,
				cash,
				ProfitShareAccrualInput{
					PeriodProfitBaseAccumulator: 1_000,
					MonthsInBonusPeriod:         1,
					IsBonusPeriodEnd:            true,
				},
			)
			if err != nil {
				t.Fatalf("ApplyProfitShareAccrual() error = %v", err)
			}
			assertMoney(t, "gross bonus", compensation.GrossBonusPoolAccrued, tt.wantGross)
			assertMoney(t, "payroll tax", compensation.BonusPayrollTaxAccrued, tt.wantGross*0.25)
			assertMoney(t, "employer cost", compensation.TotalBonusEmployerCostAccrued, tt.wantGross*1.25)
		})
	}
}

func TestProfitShareUsesPreBonusTaxReserveButPostBonusTaxAccrual(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Cashflow.BonusPayrollTaxRate = 0.25
	cfg.Cashflow.ProfitTaxRate = 0.20
	cfg.Cashflow.ReserveCurrentProfitTax = true
	cfg.Cashflow.PlannedReinvestmentRate = 0
	cfg.Company.RequiredCashReserveMonths = 0
	scenario := scenarioForTest(t, cfg, "profit_share_equal_10")
	share := 0.30
	hurdle := 0.0
	eligible := 20
	scenario.ProfitSharePercent = &share
	scenario.ProfitHurdleMonthly = &hurdle
	scenario.EligibleEmployeesCount = &eligible
	scenario.BonusCapTotal = nil
	scenario.BonusCapPerEmployee = nil

	pnl := domain.PnLState{
		OperatingProfitBeforeBonus: 1_000,
		ProfitAfterBonusBeforeTax:  1_000,
	}
	cash := domain.CashState{
		ClosingCashTotal:        325,
		ClosingUnrestrictedCash: 325,
	}
	pnl, cash, compensation, err := ApplyProfitShareAccrual(
		cfg,
		scenario,
		pnl,
		cash,
		ProfitShareAccrualInput{
			PeriodProfitBaseAccumulator: 1_000,
			MonthsInBonusPeriod:         1,
			IsBonusPeriodEnd:            true,
		},
	)
	if err != nil {
		t.Fatalf("ApplyProfitShareAccrual() error = %v", err)
	}
	pnl = AccrueProfitTax(cfg, pnl)

	assertMoney(t, "cash base after pre-bonus tax reserve", compensation.CashBaseForTotalEmployerBonusCost, 125)
	assertMoney(t, "cash-affordable gross bonus", compensation.GrossBonusPoolAccrued, 100)
	assertMoney(t, "bonus payroll tax", compensation.BonusPayrollTaxAccrued, 25)
	assertMoney(t, "post-bonus profit tax", pnl.ProfitTaxAccrual, 175)
	assertMoney(t, "bonus accrual does not reduce total cash", cash.ClosingCashTotal, 325)
}

func scenarioForTest(t *testing.T, cfg config.Config, name string) config.CompensationScenario {
	t.Helper()
	for _, scenario := range cfg.CompensationScenarios {
		if scenario.Name == name {
			return scenario
		}
	}
	t.Fatalf("scenario %q not found", name)
	return config.CompensationScenario{}
}

func floatPointer(value float64) *float64 {
	return &value
}
