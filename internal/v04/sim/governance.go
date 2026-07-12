package sim

import (
	"fmt"
	"math"

	"simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

type GovernanceInput struct {
	HeadcountBegin                float64
	StandardHoursPerEmployeeMonth float64
	Parameters                    config.Governance
}

type GovernanceResult struct {
	GovernanceHours                    float64
	GovernanceAdminEquivalentEmployees float64
	GovernanceCashCost                 float64
	DecisionDelayMonths                float64
	DecisionQualityRaw                 float64
	DecisionQualityMultiplier          float64
}

// CalculateGovernance implements specification section 8.2 without adding a
// scenario-specific benefit or penalty beyond the stated formulas.
func CalculateGovernance(input GovernanceInput) (GovernanceResult, error) {
	if !domain.Finite(input.HeadcountBegin) || input.HeadcountBegin < 0 {
		return GovernanceResult{}, fmt.Errorf("headcount_begin must be finite and >= 0")
	}
	if !domain.Finite(input.StandardHoursPerEmployeeMonth) || input.StandardHoursPerEmployeeMonth <= 0 {
		return GovernanceResult{}, fmt.Errorf("standard_hours_per_employee_month must be finite and > 0")
	}
	g := input.Parameters
	if !domain.Finite(g.GovernanceCapabilityIndex) || g.GovernanceCapabilityIndex <= 0 {
		return GovernanceResult{}, fmt.Errorf("governance_capability_index must be finite and > 0")
	}

	hours := input.HeadcountBegin*g.GovernanceParticipationIntensity*g.BaseGovernanceHoursPerEmployeeMonth + g.FixedGovernanceHoursMonthly
	adminEquivalent := hours / input.StandardHoursPerEmployeeMonth
	cashCost := g.GovernanceCashCostFixedMonthly + input.HeadcountBegin*g.GovernanceCashCostPerEmployeeMonthly
	delay := math.Max(0,
		g.BaseDecisionDelayMonths+
			g.DecisionComplexityIndex*g.GovernanceParticipationIntensity*g.DelayPerParticipationMonths/g.GovernanceCapabilityIndex-
			g.LocalAutonomyIndex*g.DecentralizationSpeedGainMonths,
	)
	qualityRaw := 1 +
		g.QualityGainFromParticipation*g.GovernanceParticipationIntensity*g.InformationSharingQuality -
		g.CoordinationLossFromParticipation*g.GovernanceParticipationIntensity*g.DecisionComplexityIndex -
		g.ConflictLossSensitivity*(1-g.TrustIndex) -
		g.DecisionDelayQualityLoss*delay
	quality := domain.Clamp(qualityRaw, g.DecisionQualityMin, g.DecisionQualityMax)

	result := GovernanceResult{
		GovernanceHours:                    hours,
		GovernanceAdminEquivalentEmployees: adminEquivalent,
		GovernanceCashCost:                 cashCost,
		DecisionDelayMonths:                delay,
		DecisionQualityRaw:                 qualityRaw,
		DecisionQualityMultiplier:          quality,
	}
	for name, value := range map[string]float64{
		"governance_hours":                      result.GovernanceHours,
		"governance_admin_equivalent_employees": result.GovernanceAdminEquivalentEmployees,
		"governance_cash_cost":                  result.GovernanceCashCost,
		"decision_delay_months":                 result.DecisionDelayMonths,
		"decision_quality_raw":                  result.DecisionQualityRaw,
		"decision_quality_multiplier":           result.DecisionQualityMultiplier,
	} {
		if !domain.Finite(value) {
			return GovernanceResult{}, fmt.Errorf("%s is not finite", name)
		}
	}
	return result, nil
}
