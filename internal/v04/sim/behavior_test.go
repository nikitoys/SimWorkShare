package sim

import (
	"math"
	"testing"

	"simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func TestCalculateBehaviorExactSections83To87(t *testing.T) {
	input := BehaviorInput{
		HeadcountBegin:                10,
		StandardHoursPerEmployeeMonth: 160,
		IncomeHistory: []domain.IncomeMonth{
			{FixedSalaryPaid: 800, CashDistributionPaid: 200, PerEmployeeIncome: 100},
			{FixedSalaryPaid: 800, CashDistributionPaid: 0, PerEmployeeIncome: 200},
		},
		MemberCapitalAccountsTotalBegin: 1000,
		ZeroDistributionStreak:          2,
		LaborMarketFactor:               1.1,
		Epsilon:                         1e-9,
		Governance: GovernanceResult{
			GovernanceHours:           80,
			DecisionQualityMultiplier: 1.1,
		},
		Scenario: config.OrganizationalScenario{
			EmployeeOwnershipFraction:      0.4,
			EmployeeCashDistributionRate:   0.1,
			MemberCapitalAllocationRate:    0.05,
			DistributionRule:               config.DistributionHybrid,
			ContributionMeasurementQuality: 0.6,
			PeerMonitoringEffectiveness:    0.25,
			TransparencyIndex:              0.5,
			Governance: config.Governance{
				GovernanceParticipationIntensity: 0.5,
			},
		},
		Case: config.BehaviorCase{
			BaseProductivityUpliftDirect:                         0.01,
			OwnershipProductivitySensitivity:                     0.02,
			ProfitDistributionProductivitySensitivity:            0.003,
			GovernanceVoiceProductivitySensitivity:               0.04,
			BaseTurnoverDeltaAnnualPP:                            0.01,
			OwnershipRetentionDeltaAnnualPPPerFullOwnership:      -0.02,
			ProfitDistributionRetentionDeltaAnnualPPPer10PP:      -0.01,
			GovernanceRetentionDeltaAnnualPPPerFullParticipation: -0.02,
			FairnessBase:                                 0.1,
			TransparencyToFairness:                       0.2,
			EqualDistributionFairnessEffect:              0.04,
			ContributionBasedDistributionFairnessEffect:  0.06,
			UnpaidGovernanceBurdenPenalty:                0.2,
			ZeroDistributionFairnessPenalty:              0.03,
			FairnessProductivitySensitivity:              0.03,
			FairnessTurnoverSensitivityAnnualPP:          0.05,
			FreeRiderBasePenalty:                         0.1,
			FreeRiderSizeExponent:                        0.5,
			FreeRiderReferenceHeadcount:                  5,
			FreeRiderMaxSizeMultiplier:                   3,
			RiskConcentrationTurnoverSensitivityAnnualPP: 0.02,
			IncomeVolatilityTurnoverSensitivityAnnualPP:  0.03,
		},
		Workforce: config.Workforce{
			BaseTurnoverRateAnnual:                       0.2,
			MinTurnoverRateAnnual:                        0.03,
			MaxTurnoverRateAnnual:                        0.6,
			MinProductivityUplift:                        -0.15,
			MaxProductivityUplift:                        0.2,
			TurnoverProductivityPenaltyPerAnnualTurnover: 0.1,
		},
		EmployeeRisk: config.EmployeeRisk{
			EmployeeExternalSavingsProxyPerEmployee: 300,
			EmploymentDependenceIndex:               0.8,
			RiskWeightVariableIncome:                0.3,
			RiskWeightMemberCapital:                 0.4,
			RiskWeightEmploymentDependence:          0.3,
		},
	}

	result, err := CalculateBehavior(input)
	if err != nil {
		t.Fatal(err)
	}

	wantRuleEffect := 0.5*0.04 + 0.5*0.06*0.6
	wantBurden := 80.0 / (10 * 160)
	wantFairness := 0.1 + 0.2*0.5 + wantRuleEffect - 0.2*wantBurden - 0.03*2
	wantSize := math.Sqrt(10.0 / 5)
	wantMonitoring := 0.25 * 0.6
	wantFreeRider := 0.1 * (0.1 + 0.05) * 0.5 * wantSize * (1 - wantMonitoring)
	wantVariableIncomeShare := 200.0 / 1800
	wantCapitalConcentration := 100.0 / (100 + 300)
	wantRisk := 0.3*wantVariableIncomeShare + 0.4*wantCapitalConcentration + 0.3*0.8
	wantVolatility := 1.0 / 3.0
	wantOwnershipSalience := 0.4 * 0.5
	wantDistributionSalience := 1.0
	wantGovernanceSalience := 0.5
	wantMotivation := 0.01 + 0.02*wantOwnershipSalience + 0.003*wantDistributionSalience + 0.04*wantGovernanceSalience
	wantFairnessTurnover := -0.05 * wantFairness
	wantRiskTurnover := 0.02 * wantRisk
	wantVolatilityTurnover := 0.03 * wantVolatility
	wantBehavioralTurnover := 0.01 +
		-0.02*wantOwnershipSalience + -0.01*wantDistributionSalience + -0.02*wantGovernanceSalience +
		wantFairnessTurnover + wantRiskTurnover + wantVolatilityTurnover
	wantAnnual := 0.2*1.1 + wantBehavioralTurnover
	wantMonthly := 1 - math.Pow(1-wantAnnual, 1.0/12)
	wantTurnoverLoss := 0.1 * math.Max(0, wantAnnual-0.2)
	wantProductivity := wantMotivation + 0.03*wantFairness + 0.04*(1.1-1) - wantFreeRider - wantTurnoverLoss

	assertClose(t, "distribution rule effect", result.DistributionRuleEffect, wantRuleEffect)
	assertClose(t, "governance burden", result.GovernanceBurdenShare, wantBurden)
	assertClose(t, "pay dispersion default", result.PayDispersionIndex, 0)
	assertClose(t, "fairness", result.FairnessIndex, wantFairness)
	assertClose(t, "free rider", result.FreeRiderPenalty, wantFreeRider)
	assertClose(t, "variable income share", result.VariableIncomeShare12M, wantVariableIncomeShare)
	assertClose(t, "capital concentration", result.CapitalConcentration, wantCapitalConcentration)
	assertClose(t, "employee risk", result.EmployeeRiskConcentration, wantRisk)
	assertClose(t, "income volatility", result.IncomeVolatilityIndex12M, wantVolatility)
	assertClose(t, "motivation raw", result.MotivationUpliftRaw, wantMotivation)
	assertClose(t, "behavioral turnover delta", result.BehavioralTurnoverDeltaAnnual, wantBehavioralTurnover)
	assertClose(t, "annual turnover", result.TurnoverRateAnnual, wantAnnual)
	assertClose(t, "monthly turnover", result.TurnoverRateMonthly, wantMonthly)
	assertClose(t, "productivity uplift", result.ProductivityUplift, wantProductivity)
	assertClose(t, "productivity multiplier", result.ProductivityMultiplier, 1+wantProductivity)
}

func TestCalculateBehaviorNoEffectNeutrality(t *testing.T) {
	input := neutralBehaviorInput()
	result, err := CalculateBehavior(input)
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "motivation raw", result.MotivationUpliftRaw, 0)
	assertClose(t, "behavioral turnover delta", result.BehavioralTurnoverDeltaAnnual, 0)
	assertClose(t, "productivity uplift", result.ProductivityUplift, 0)
	assertClose(t, "annual turnover", result.TurnoverRateAnnual, input.Workforce.BaseTurnoverRateAnnual)
}

func TestCalculateBehaviorGovernanceVoiceSensitivityAppearsTwice(t *testing.T) {
	input := neutralBehaviorInput()
	input.Scenario.Governance.GovernanceParticipationIntensity = 0.5
	input.Governance.DecisionQualityMultiplier = 1.2
	input.Case.GovernanceVoiceProductivitySensitivity = 0.1

	result, err := CalculateBehavior(input)
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "motivation governance voice", result.MotivationUpliftRaw, 0.05)
	assertClose(t, "total governance voice productivity", result.ProductivityUplift, 0.07)
}

func TestCalculateBehaviorNegativeCaseCanReduceProductivityAndRaiseTurnover(t *testing.T) {
	input := neutralBehaviorInput()
	input.Case.BaseProductivityUpliftDirect = -0.01
	input.Case.BaseTurnoverDeltaAnnualPP = 0.02
	input.Case.FairnessBase = -0.1
	input.Case.EqualDistributionFairnessEffect = -0.15
	input.Case.FairnessProductivitySensitivity = 0.03
	input.Case.FairnessTurnoverSensitivityAnnualPP = 0.04
	input.Case.FreeRiderBasePenalty = 0.03

	result, err := CalculateBehavior(input)
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "negative fairness", result.FairnessIndex, -0.25)
	assertClose(t, "negative case turnover delta", result.BehavioralTurnoverDeltaAnnual, 0.03)
	assertClose(t, "negative case turnover", result.TurnoverRateAnnual, 0.23)
	assertClose(t, "negative case productivity", result.ProductivityUplift, -0.0265)
	if result.ProductivityUplift >= 0 || result.TurnoverRateAnnual <= input.Workforce.BaseTurnoverRateAnnual {
		t.Fatal("negative behavior case did not remain negative")
	}
}

func TestIncomeVolatilityUsesPopulationCVAndLastTwelvePriorMonths(t *testing.T) {
	assertClose(t, "fewer than two months", incomeVolatilityIndex([]domain.IncomeMonth{{PerEmployeeIncome: 100}}, 1e-9), 0)
	assertClose(t, "population CV", incomeVolatilityIndex([]domain.IncomeMonth{{PerEmployeeIncome: 100}, {PerEmployeeIncome: 200}}, 1e-9), 1.0/3.0)
	assertClose(t, "CV upper clamp", incomeVolatilityIndex([]domain.IncomeMonth{{PerEmployeeIncome: 0}, {PerEmployeeIncome: 2}}, 1e-9), 1)
	assertClose(t, "nonpositive mean", incomeVolatilityIndex([]domain.IncomeMonth{{PerEmployeeIncome: -1}, {PerEmployeeIncome: 1}}, 1e-9), 0)

	history := []domain.IncomeMonth{{PerEmployeeIncome: 1e9}}
	for range 12 {
		history = append(history, domain.IncomeMonth{PerEmployeeIncome: 100})
	}
	assertClose(t, "last twelve only", incomeVolatilityIndex(lastIncomeMonths(history, 12), 1e-9), 0)
}

func neutralBehaviorInput() BehaviorInput {
	return BehaviorInput{
		HeadcountBegin:                  50,
		StandardHoursPerEmployeeMonth:   160,
		MemberCapitalAccountsTotalBegin: 10000,
		LaborMarketFactor:               1,
		Epsilon:                         1e-9,
		Governance: GovernanceResult{
			GovernanceHours:           100,
			DecisionQualityMultiplier: 1.2,
		},
		Scenario: config.OrganizationalScenario{
			EmployeeOwnershipFraction:      1,
			EmployeeCashDistributionRate:   0.15,
			MemberCapitalAllocationRate:    0.05,
			DistributionRule:               config.DistributionEqualPerCapita,
			ContributionMeasurementQuality: 0.5,
			TransparencyIndex:              0.8,
			Governance: config.Governance{
				GovernanceParticipationIntensity: 0.8,
			},
		},
		Case: config.BehaviorCase{
			FreeRiderSizeExponent:       0.5,
			FreeRiderReferenceHeadcount: 50,
			FreeRiderMaxSizeMultiplier:  3,
		},
		Workforce: config.Workforce{
			BaseTurnoverRateAnnual:                       0.2,
			MinTurnoverRateAnnual:                        0,
			MaxTurnoverRateAnnual:                        1,
			MinProductivityUplift:                        -0.15,
			MaxProductivityUplift:                        0.2,
			TurnoverProductivityPenaltyPerAnnualTurnover: 0.1,
		},
		EmployeeRisk: config.EmployeeRisk{
			EmployeeExternalSavingsProxyPerEmployee: 600000,
			EmploymentDependenceIndex:               1,
			RiskWeightVariableIncome:                0.4,
			RiskWeightMemberCapital:                 0.4,
			RiskWeightEmploymentDependence:          0.2,
		},
	}
}
