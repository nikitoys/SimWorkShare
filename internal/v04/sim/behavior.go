package sim

import (
	"fmt"
	"math"

	"simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

const DefaultPayDispersionIndex = 0.0

type BehaviorInput struct {
	HeadcountBegin                  float64
	StandardHoursPerEmployeeMonth   float64
	IncomeHistory                   []domain.IncomeMonth
	MemberCapitalAccountsTotalBegin float64
	ZeroDistributionStreak          int
	LaborMarketFactor               float64
	Epsilon                         float64
	Governance                      GovernanceResult
	Scenario                        config.OrganizationalScenario
	Case                            config.BehaviorCase
	Workforce                       config.Workforce
	EmployeeRisk                    config.EmployeeRisk
}

type BehaviorResult struct {
	DistributionRuleEffect         float64
	GovernanceBurdenShare          float64
	PayDispersionIndex             float64
	FairnessIndex                  float64
	CollectiveDistributionExposure float64
	EqualDistributionFactor        float64
	SizeFactor                     float64
	MonitoringReduction            float64
	FreeRiderPenalty               float64
	VariableIncomeShare12M         float64
	MemberCapitalPerEmployee       float64
	CapitalConcentration           float64
	EmployeeRiskConcentration      float64
	IncomeVolatilityIndex12M       float64
	OwnershipSalience              float64
	ProfitDistributionSalience     float64
	GovernanceSalience             float64
	MotivationUpliftRaw            float64
	FairnessProductivityEffect     float64
	TurnoverExcess                 float64
	TurnoverProductivityLoss       float64
	ProductivityUplift             float64
	ProductivityMultiplier         float64
	FairnessTurnoverDelta          float64
	RiskTurnoverDelta              float64
	IncomeVolatilityTurnoverDelta  float64
	OwnershipTurnoverDelta         float64
	DistributionTurnoverDelta      float64
	GovernanceTurnoverDelta        float64
	BehavioralTurnoverDeltaAnnual  float64
	TurnoverRateAnnual             float64
	TurnoverRateMonthly            float64
}

// CalculateBehavior implements specification sections 8.3 through 8.7. The
// governance-voice sensitivity intentionally appears both in motivation and
// in the positive decision-quality term, exactly as the specification states.
func CalculateBehavior(input BehaviorInput) (BehaviorResult, error) {
	if !domain.Finite(input.HeadcountBegin) || input.HeadcountBegin < 0 {
		return BehaviorResult{}, fmt.Errorf("headcount_begin must be finite and >= 0")
	}
	if !domain.Finite(input.MemberCapitalAccountsTotalBegin) || input.MemberCapitalAccountsTotalBegin < 0 {
		return BehaviorResult{}, fmt.Errorf("member_capital_accounts_total_begin must be finite and >= 0")
	}
	if input.ZeroDistributionStreak < 0 {
		return BehaviorResult{}, fmt.Errorf("zero_distribution_streak must be >= 0")
	}
	if !domain.Finite(input.LaborMarketFactor) || input.LaborMarketFactor < 0 {
		return BehaviorResult{}, fmt.Errorf("labor_market_factor must be finite and >= 0")
	}
	if !domain.Finite(input.Epsilon) || input.Epsilon <= 0 {
		return BehaviorResult{}, fmt.Errorf("epsilon must be finite and > 0")
	}
	if !domain.Finite(input.StandardHoursPerEmployeeMonth) || input.StandardHoursPerEmployeeMonth <= 0 {
		return BehaviorResult{}, fmt.Errorf("standard_hours_per_employee_month must be finite and > 0")
	}
	if input.Case.FreeRiderReferenceHeadcount <= 0 || !domain.Finite(input.Case.FreeRiderReferenceHeadcount) {
		return BehaviorResult{}, fmt.Errorf("free_rider_reference_headcount must be finite and > 0")
	}

	history := lastIncomeMonths(input.IncomeHistory, 12)
	for index, month := range history {
		if !domain.Finite(month.FixedSalaryPaid) || !domain.Finite(month.CashDistributionPaid) || !domain.Finite(month.PerEmployeeIncome) {
			return BehaviorResult{}, fmt.Errorf("income_history[%d] contains a non-finite value", len(input.IncomeHistory)-len(history)+index)
		}
	}

	distributionRuleEffect, equalDistributionFactor, err := distributionEffects(input.Scenario.DistributionRule, input.Scenario.ContributionMeasurementQuality, input.Case)
	if err != nil {
		return BehaviorResult{}, err
	}

	governanceBurdenShare := input.Governance.GovernanceHours / math.Max(input.Epsilon, input.HeadcountBegin*input.StandardHoursPerEmployeeMonth)
	fairness := domain.Clamp(
		input.Case.FairnessBase+
			input.Case.TransparencyToFairness*input.Scenario.TransparencyIndex+
			distributionRuleEffect-
			input.Case.PayDispersionFairnessPenalty*DefaultPayDispersionIndex-
			input.Case.UnpaidGovernanceBurdenPenalty*governanceBurdenShare-
			input.Case.ZeroDistributionFairnessPenalty*float64(input.ZeroDistributionStreak),
		-1, 1,
	)

	collectiveExposure := input.Scenario.EmployeeCashDistributionRate + input.Scenario.MemberCapitalAllocationRate
	sizeFactor := math.Min(
		input.Case.FreeRiderMaxSizeMultiplier,
		math.Pow(input.HeadcountBegin/input.Case.FreeRiderReferenceHeadcount, input.Case.FreeRiderSizeExponent),
	)
	monitoringReduction := input.Scenario.PeerMonitoringEffectiveness * input.Scenario.ContributionMeasurementQuality
	freeRiderPenalty := input.Case.FreeRiderBasePenalty * collectiveExposure * equalDistributionFactor * sizeFactor * (1 - monitoringReduction)

	fixedSalaryPaid12M := 0.0
	distributionPaid12M := 0.0
	for _, month := range history {
		fixedSalaryPaid12M += month.FixedSalaryPaid
		distributionPaid12M += month.CashDistributionPaid
	}
	variableIncomeShare := distributionPaid12M / math.Max(input.Epsilon, fixedSalaryPaid12M+distributionPaid12M)
	memberCapitalPerEmployee := input.MemberCapitalAccountsTotalBegin / math.Max(input.Epsilon, input.HeadcountBegin)
	capitalConcentration := memberCapitalPerEmployee / math.Max(input.Epsilon, memberCapitalPerEmployee+input.EmployeeRisk.EmployeeExternalSavingsProxyPerEmployee)
	employeeRiskConcentration := domain.Clamp(
		input.EmployeeRisk.RiskWeightVariableIncome*variableIncomeShare+
			input.EmployeeRisk.RiskWeightMemberCapital*capitalConcentration+
			input.EmployeeRisk.RiskWeightEmploymentDependence*input.EmployeeRisk.EmploymentDependenceIndex,
		0, 1,
	)
	incomeVolatility := incomeVolatilityIndex(history, input.Epsilon)

	ownershipSalience := input.Scenario.EmployeeOwnershipFraction * input.Scenario.TransparencyIndex
	profitDistributionSalience := input.Scenario.EmployeeCashDistributionRate / 0.10
	governanceSalience := input.Scenario.Governance.GovernanceParticipationIntensity
	motivationRaw := input.Case.BaseProductivityUpliftDirect +
		input.Case.OwnershipProductivitySensitivity*ownershipSalience +
		input.Case.ProfitDistributionProductivitySensitivity*profitDistributionSalience +
		input.Case.GovernanceVoiceProductivitySensitivity*governanceSalience

	fairnessTurnoverDelta := -input.Case.FairnessTurnoverSensitivityAnnualPP * fairness
	riskTurnoverDelta := input.Case.RiskConcentrationTurnoverSensitivityAnnualPP * employeeRiskConcentration
	incomeVolatilityTurnoverDelta := input.Case.IncomeVolatilityTurnoverSensitivityAnnualPP * incomeVolatility
	ownershipTurnoverDelta := input.Case.OwnershipRetentionDeltaAnnualPPPerFullOwnership * ownershipSalience
	distributionTurnoverDelta := input.Case.ProfitDistributionRetentionDeltaAnnualPPPer10PP * profitDistributionSalience
	governanceTurnoverDelta := input.Case.GovernanceRetentionDeltaAnnualPPPerFullParticipation * governanceSalience
	behavioralTurnoverDelta := input.Case.BaseTurnoverDeltaAnnualPP +
		ownershipTurnoverDelta + distributionTurnoverDelta + governanceTurnoverDelta +
		fairnessTurnoverDelta + riskTurnoverDelta + incomeVolatilityTurnoverDelta
	turnoverAnnual := domain.Clamp(
		input.Workforce.BaseTurnoverRateAnnual*input.LaborMarketFactor+behavioralTurnoverDelta,
		input.Workforce.MinTurnoverRateAnnual,
		input.Workforce.MaxTurnoverRateAnnual,
	)
	if turnoverAnnual < 0 || turnoverAnnual > 1 {
		return BehaviorResult{}, fmt.Errorf("turnover_rate_annual must be within [0,1], got %g", turnoverAnnual)
	}
	turnoverMonthly := 1 - math.Pow(1-turnoverAnnual, 1.0/12.0)

	fairnessProductivityEffect := input.Case.FairnessProductivitySensitivity * fairness
	turnoverExcess := math.Max(0, turnoverAnnual-input.Workforce.BaseTurnoverRateAnnual)
	turnoverProductivityLoss := input.Workforce.TurnoverProductivityPenaltyPerAnnualTurnover * turnoverExcess
	productivityUplift := domain.Clamp(
		motivationRaw+fairnessProductivityEffect+
			input.Case.GovernanceVoiceProductivitySensitivity*math.Max(0, input.Governance.DecisionQualityMultiplier-1)-
			freeRiderPenalty-turnoverProductivityLoss,
		input.Workforce.MinProductivityUplift,
		input.Workforce.MaxProductivityUplift,
	)

	result := BehaviorResult{
		DistributionRuleEffect:         distributionRuleEffect,
		GovernanceBurdenShare:          governanceBurdenShare,
		PayDispersionIndex:             DefaultPayDispersionIndex,
		FairnessIndex:                  fairness,
		CollectiveDistributionExposure: collectiveExposure,
		EqualDistributionFactor:        equalDistributionFactor,
		SizeFactor:                     sizeFactor,
		MonitoringReduction:            monitoringReduction,
		FreeRiderPenalty:               freeRiderPenalty,
		VariableIncomeShare12M:         variableIncomeShare,
		MemberCapitalPerEmployee:       memberCapitalPerEmployee,
		CapitalConcentration:           capitalConcentration,
		EmployeeRiskConcentration:      employeeRiskConcentration,
		IncomeVolatilityIndex12M:       incomeVolatility,
		OwnershipSalience:              ownershipSalience,
		ProfitDistributionSalience:     profitDistributionSalience,
		GovernanceSalience:             governanceSalience,
		MotivationUpliftRaw:            motivationRaw,
		FairnessProductivityEffect:     fairnessProductivityEffect,
		TurnoverExcess:                 turnoverExcess,
		TurnoverProductivityLoss:       turnoverProductivityLoss,
		ProductivityUplift:             productivityUplift,
		ProductivityMultiplier:         1 + productivityUplift,
		FairnessTurnoverDelta:          fairnessTurnoverDelta,
		RiskTurnoverDelta:              riskTurnoverDelta,
		IncomeVolatilityTurnoverDelta:  incomeVolatilityTurnoverDelta,
		OwnershipTurnoverDelta:         ownershipTurnoverDelta,
		DistributionTurnoverDelta:      distributionTurnoverDelta,
		GovernanceTurnoverDelta:        governanceTurnoverDelta,
		BehavioralTurnoverDeltaAnnual:  behavioralTurnoverDelta,
		TurnoverRateAnnual:             turnoverAnnual,
		TurnoverRateMonthly:            turnoverMonthly,
	}
	if err := validateBehaviorResultFinite(result); err != nil {
		return BehaviorResult{}, err
	}
	return result, nil
}

func lastIncomeMonths(history []domain.IncomeMonth, maximum int) []domain.IncomeMonth {
	if len(history) <= maximum {
		return history
	}
	return history[len(history)-maximum:]
}

func distributionEffects(rule string, measurementQuality float64, behavior config.BehaviorCase) (ruleEffect, equalFactor float64, err error) {
	switch rule {
	case config.DistributionNone:
		return 0, 0, nil
	case config.DistributionEqualPerCapita:
		return behavior.EqualDistributionFairnessEffect, 1, nil
	case config.DistributionContributionWeighted:
		return behavior.ContributionBasedDistributionFairnessEffect * measurementQuality, 0, nil
	case config.DistributionHybrid:
		return 0.5*behavior.EqualDistributionFairnessEffect + 0.5*behavior.ContributionBasedDistributionFairnessEffect*measurementQuality, 0.5, nil
	default:
		return 0, 0, fmt.Errorf("unsupported distribution_rule %q", rule)
	}
}

// incomeVolatilityIndex is the population coefficient of variation over the
// available (at most twelve) prior per-employee income months.
func incomeVolatilityIndex(history []domain.IncomeMonth, epsilon float64) float64 {
	if len(history) < 2 {
		return 0
	}
	mean := 0.0
	for _, month := range history {
		mean += month.PerEmployeeIncome
	}
	mean /= float64(len(history))
	if mean <= epsilon {
		return 0
	}
	variance := 0.0
	for _, month := range history {
		delta := month.PerEmployeeIncome - mean
		variance += delta * delta
	}
	variance /= float64(len(history))
	return domain.Clamp(math.Sqrt(variance)/mean, 0, 1)
}

func validateBehaviorResultFinite(result BehaviorResult) error {
	values := map[string]float64{
		"distribution_rule_effect":          result.DistributionRuleEffect,
		"governance_burden_share":           result.GovernanceBurdenShare,
		"fairness_index":                    result.FairnessIndex,
		"free_rider_penalty":                result.FreeRiderPenalty,
		"employee_risk_concentration_index": result.EmployeeRiskConcentration,
		"income_volatility_index_12m":       result.IncomeVolatilityIndex12M,
		"motivation_uplift_raw":             result.MotivationUpliftRaw,
		"behavioral_turnover_delta_annual":  result.BehavioralTurnoverDeltaAnnual,
		"turnover_rate_annual":              result.TurnoverRateAnnual,
		"turnover_rate_monthly":             result.TurnoverRateMonthly,
		"productivity_uplift":               result.ProductivityUplift,
		"productivity_multiplier":           result.ProductivityMultiplier,
	}
	for name, value := range values {
		if !domain.Finite(value) {
			return fmt.Errorf("%s is not finite", name)
		}
	}
	return nil
}
