package sim

import (
	"math"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

type ProductionInput struct {
	Company                   v04config.CompanyEconomics
	Market                    v04config.Market
	Governance                v04config.Governance
	Environment               domain.EnvironmentMonth
	EffectiveEmployees        float64
	ProductivityMultiplier    float64
	DecisionQualityMultiplier float64
	DecisionDelayMonths       float64
	CapacityLimit             float64
}

type ProductionOutput struct {
	EffectiveShockRevenueMultiplier float64
	MarketDemand                    float64
	LaborRevenueCapacity            float64
	ProductiveCapacityLimit         float64
	Revenue                         float64
}

// CapacityAfterDepreciationAndDue is both the current production limit and the
// closing baseline before investments made in the current month. This follows
// the section-18 identity and makes additions due at the start of a month
// available to production in that month.
func CapacityAfterDepreciationAndDue(begin, depreciationRate, additionsDue float64) float64 {
	return math.Max(0, begin*(1-depreciationRate)+additionsDue)
}

func CalculateProduction(input ProductionInput) ProductionOutput {
	shockMultiplier := 1.0
	if input.Environment.ShockHappened {
		baseLoss := 1 - input.Market.ShockRevenueMultiplier
		lossMultiplier := domain.Clamp(
			1-input.Governance.ShockMitigationSensitivity*(input.DecisionQualityMultiplier-1)+
				input.Governance.ShockDelayAmplification*input.DecisionDelayMonths,
			0,
			3,
		)
		shockMultiplier = 1 - domain.Clamp(baseLoss*lossMultiplier, 0, 1)
	}
	marketDemand := input.Company.InitialMarketDemandMonthly *
		input.Environment.MarketTrend *
		input.Environment.SeasonalityMultiplier *
		input.Environment.MarketFactor *
		shockMultiplier
	laborCapacity := math.Max(0, input.EffectiveEmployees) *
		input.Company.BaseRevenuePerEffectiveEmployeeMonthly *
		math.Max(0, input.ProductivityMultiplier)
	capacity := math.Max(0, input.CapacityLimit)
	revenue := math.Max(0, math.Min(marketDemand, math.Min(laborCapacity, capacity)))
	return ProductionOutput{
		EffectiveShockRevenueMultiplier: shockMultiplier,
		MarketDemand:                    math.Max(0, marketDemand),
		LaborRevenueCapacity:            laborCapacity,
		ProductiveCapacityLimit:         capacity,
		Revenue:                         revenue,
	}
}

func CapacityCreated(
	reinvestmentTotal float64,
	company v04config.CompanyEconomics,
	governance v04config.Governance,
	decisionQuality float64,
) (added float64, efficiencyMultiplier float64) {
	efficiencyMultiplier = domain.Clamp(
		1+governance.InvestmentEfficiencySensitivity*(decisionQuality-1),
		0,
		3,
	)
	added = math.Max(0, reinvestmentTotal) *
		company.CapacityRevenueCreatedPerCurrencyInvested *
		efficiencyMultiplier
	return added, efficiencyMultiplier
}
