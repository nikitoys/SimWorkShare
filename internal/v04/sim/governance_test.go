package sim

import (
	"testing"

	"simworkshare/internal/v04/config"
)

func TestCalculateGovernanceExactSection82Formulas(t *testing.T) {
	parameters := config.Governance{
		GovernanceParticipationIntensity:     0.5,
		BaseGovernanceHoursPerEmployeeMonth:  4,
		FixedGovernanceHoursMonthly:          20,
		GovernanceCashCostFixedMonthly:       1000,
		GovernanceCashCostPerEmployeeMonthly: 50,
		DecisionComplexityIndex:              1.2,
		BaseDecisionDelayMonths:              0.2,
		DelayPerParticipationMonths:          0.6,
		LocalAutonomyIndex:                   0.5,
		DecentralizationSpeedGainMonths:      0.1,
		GovernanceCapabilityIndex:            0.8,
		InformationSharingQuality:            0.8,
		TrustIndex:                           0.7,
		QualityGainFromParticipation:         0.1,
		CoordinationLossFromParticipation:    0.03,
		ConflictLossSensitivity:              0.02,
		DecisionDelayQualityLoss:             0.05,
		DecisionQualityMin:                   0.9,
		DecisionQualityMax:                   1.1,
	}
	result, err := CalculateGovernance(GovernanceInput{
		HeadcountBegin:                100,
		StandardHoursPerEmployeeMonth: 160,
		Parameters:                    parameters,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertClose(t, "governance hours", result.GovernanceHours, 220)
	assertClose(t, "governance admin equivalent", result.GovernanceAdminEquivalentEmployees, 1.375)
	assertClose(t, "governance cash cost", result.GovernanceCashCost, 6000)
	assertClose(t, "decision delay", result.DecisionDelayMonths, 0.6)
	assertClose(t, "decision quality raw", result.DecisionQualityRaw, 0.986)
	assertClose(t, "decision quality", result.DecisionQualityMultiplier, 0.986)
}

func TestCalculateGovernanceClampsDelayAndQuality(t *testing.T) {
	result, err := CalculateGovernance(GovernanceInput{
		HeadcountBegin:                5,
		StandardHoursPerEmployeeMonth: 100,
		Parameters: config.Governance{
			BaseDecisionDelayMonths:         0.1,
			LocalAutonomyIndex:              1,
			DecentralizationSpeedGainMonths: 1,
			GovernanceCapabilityIndex:       1,
			TrustIndex:                      0,
			ConflictLossSensitivity:         2,
			DecisionQualityMin:              0.7,
			DecisionQualityMax:              1.2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "nonnegative delay", result.DecisionDelayMonths, 0)
	assertClose(t, "quality lower clamp", result.DecisionQualityMultiplier, 0.7)
}

func TestCalculateGovernanceRejectsZeroCapability(t *testing.T) {
	_, err := CalculateGovernance(GovernanceInput{
		HeadcountBegin:                1,
		StandardHoursPerEmployeeMonth: 160,
		Parameters: config.Governance{
			DecisionQualityMin: 0.7,
			DecisionQualityMax: 1.2,
		},
	})
	if err == nil {
		t.Fatal("expected zero governance capability to be rejected")
	}
}
