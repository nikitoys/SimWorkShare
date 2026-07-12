package sim

import (
	"math"
	"testing"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func TestCapacityDepreciationAndActivation(t *testing.T) {
	if got := CapacityAfterDepreciationAndDue(1_000, 0.1, 50); got != 950 {
		t.Fatalf("capacity = %g, want 950", got)
	}
}

func TestShockResponseUsesDecisionQualityFormula(t *testing.T) {
	input := ProductionInput{
		Company: v04config.CompanyEconomics{
			InitialMarketDemandMonthly:             1_000,
			BaseRevenuePerEffectiveEmployeeMonthly: 1_000,
		},
		Market: v04config.Market{ShockRevenueMultiplier: 0.8},
		Governance: v04config.Governance{
			ShockMitigationSensitivity: 0.5,
			ShockDelayAmplification:    0,
		},
		Environment: domain.EnvironmentMonth{
			MarketTrend:           1,
			MarketFactor:          1,
			SeasonalityMultiplier: 1,
			ShockHappened:         true,
		},
		EffectiveEmployees:        1,
		ProductivityMultiplier:    1,
		DecisionQualityMultiplier: 1.2,
		CapacityLimit:             2_000,
	}
	good := CalculateProduction(input)
	input.DecisionQualityMultiplier = 0.8
	poor := CalculateProduction(input)
	if math.Abs(good.EffectiveShockRevenueMultiplier-0.82) > 1e-12 || math.Abs(poor.EffectiveShockRevenueMultiplier-0.78) > 1e-12 {
		t.Fatalf("shock multipliers = %g/%g, want .82/.78", good.EffectiveShockRevenueMultiplier, poor.EffectiveShockRevenueMultiplier)
	}
	if good.Revenue <= poor.Revenue {
		t.Fatalf("good-quality revenue %g <= poor-quality %g", good.Revenue, poor.Revenue)
	}
}
