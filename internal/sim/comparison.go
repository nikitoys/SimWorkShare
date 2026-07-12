package sim

import (
	"fmt"
	"math"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

// RunDeterministicComparison runs both compensation policies against the same
// configured economics and deterministic environment path. The profit-share
// behavior case remains explicit because it is a scenario assumption, not an
// effect inferred by the model.
func RunDeterministicComparison(
	cfg config.Config,
	profitShareScenario string,
	profitShareBehaviorCase string,
) (domain.ComparisonResult, error) {
	fixed, err := RunDeterministicScenario(cfg, "fixed_only", "no_effect")
	if err != nil {
		return domain.ComparisonResult{}, fmt.Errorf("run fixed_only comparison arm: %w", err)
	}
	profitShare, err := RunDeterministicScenario(cfg, profitShareScenario, profitShareBehaviorCase)
	if err != nil {
		return domain.ComparisonResult{}, fmt.Errorf("run profit_share comparison arm: %w", err)
	}
	if profitShare.TerminalSummary.Compensation == nil {
		return domain.ComparisonResult{}, fmt.Errorf("scenario %q is not a profit_share result", profitShareScenario)
	}

	fixedLast := fixed.MonthlyResults[len(fixed.MonthlyResults)-1]
	profitLast := profitShare.MonthlyResults[len(profitShare.MonthlyResults)-1]
	var cumulativeAccountingProfitDelta domain.Money
	for index := range fixed.MonthlyResults {
		cumulativeAccountingProfitDelta += profitShare.MonthlyResults[index].PnL.AccountingProfitAfterBonus -
			fixed.MonthlyResults[index].PnL.AccountingProfitAfterBonus
	}
	compensation := profitShare.TerminalSummary.Compensation

	result := domain.ComparisonResult{
		Currency:    cfg.Simulation.Currency,
		FixedOnly:   fixed,
		ProfitShare: profitShare,
		Summary: domain.ComparisonSummary{
			ProfitShareScenario:              profitShareScenario,
			ProfitShareBehaviorCase:          profitShareBehaviorCase,
			FinalClosingCashDelta:            profitLast.Cash.ClosingCashTotal - fixedLast.Cash.ClosingCashTotal,
			FinalUnrestrictedCashDelta:       profitLast.Cash.ClosingUnrestrictedCash - fixedLast.Cash.ClosingUnrestrictedCash,
			FinalOwnerDistributableCashDelta: profitLast.Cash.OwnerDistributableCash - fixedLast.Cash.OwnerDistributableCash,
			CumulativeAccountingProfitDelta:  cumulativeAccountingProfitDelta,
			TotalGrossBonusAccrued:           compensation.TotalGrossBonusAccrued,
			TotalEmployerBonusCostAccrued:    compensation.TotalEmployerBonusCostAccrued,
			TotalGrossBonusPaid:              compensation.TotalGrossBonusPaid,
			TotalEmployerBonusCostPaid:       compensation.TotalGrossBonusPaid + compensation.TotalBonusPayrollTaxPaid,
		},
	}
	for _, item := range []struct {
		name  string
		value domain.Money
	}{
		{"final closing cash delta", result.Summary.FinalClosingCashDelta},
		{"final unrestricted cash delta", result.Summary.FinalUnrestrictedCashDelta},
		{"final owner distributable cash delta", result.Summary.FinalOwnerDistributableCashDelta},
		{"cumulative accounting profit delta", result.Summary.CumulativeAccountingProfitDelta},
	} {
		if math.IsNaN(float64(item.value)) || math.IsInf(float64(item.value), 0) {
			return domain.ComparisonResult{}, fmt.Errorf("comparison integrity: %s is not finite", item.name)
		}
	}
	return result, nil
}
