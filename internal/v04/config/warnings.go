package config

import (
	"fmt"
	"sort"
	"strings"
)

const (
	WarningTurnoverDoubleCount     = "turnover_double_count"
	WarningScenarioLabelMismatch   = "scenario_label_or_governance_mismatch"
	WarningHighRedemptionLiquidity = "high_member_redemption_liquidity_risk"
	WarningExternalDistribution    = "external_distribution_not_target_metric"
	AssumptionSimplifiedTaxModel   = "simplified_tax_model"
	AssumptionHighPerformerOnly    = "high_performer_attrition_indicator_only"
)

// Warning is a non-fatal configuration diagnostic. Code is stable for
// machine-readable output; Message is intended for people.
type Warning struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// AssumptionFlag records a deliberate model simplification rather than a
// problem with a particular configuration value.
type AssumptionFlag struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Warnings implements the five section-19 warning topics with deterministic
// triggers. The simplified-tax topic is deliberately returned by
// AssumptionFlags instead, so the canonical default agrees with its validation
// report and has no warnings.
func Warnings(cfg Config) []Warning {
	var warnings []Warning
	if cfg.Workforce.LostProductivityCostPerLeaver > 0 &&
		cfg.Workforce.TurnoverProductivityPenaltyPerAnnualTurnover > 0 {
		warnings = append(warnings, Warning{
			Code:    WarningTurnoverDoubleCount,
			Path:    "workforce.lost_productivity_cost_per_leaver",
			Message: "lost-productivity exit cost and the turnover productivity penalty are both positive and may double count turnover impact",
		})
	}

	canonical := map[string]string{
		SystemTraditionalCompany:       SystemTraditionalCompany,
		SystemProfitSharing:            SystemProfitSharing,
		SystemEmployeeOwnershipPartial: SystemEmployeeOwnershipPartial,
		SystemWorkerCooperative:        SystemWorkerCooperative,
	}
	for index, scenario := range cfg.OrganizationalScenarios {
		mismatch := false
		if expected, isCanonicalLabel := canonical[scenario.Name]; isCanonicalLabel && scenario.SystemType != expected {
			mismatch = true
		}
		lowerName := strings.ToLower(scenario.Name)
		mentionsCooperative := strings.Contains(lowerName, "cooperative") || strings.Contains(lowerName, "coop")
		if mentionsCooperative && scenario.SystemType != SystemWorkerCooperative {
			mismatch = true
		}
		if scenario.SystemType == SystemWorkerCooperative && scenario.Governance.GovernanceParticipationIntensity == 0 {
			mismatch = true
		}
		if mismatch {
			warnings = append(warnings, Warning{
				Code:    WarningScenarioLabelMismatch,
				Path:    scenarioPath(index, "system_type"),
				Message: "scenario label/system type or cooperative governance intensity is inconsistent",
			})
		}
	}

	// A redemption is flagged as high-liquidity-risk when at least half of an
	// exiting member's balance is redeemable in less than twelve months and the
	// scenario actually allocates member capital. The default uses a 24-month
	// lag and therefore does not trigger this warning.
	if cfg.Financing.MemberCapitalRedemptionFractionOnExit >= 0.5 &&
		cfg.Financing.MemberCapitalRedemptionLagMonths < 12 {
		for index, scenario := range cfg.OrganizationalScenarios {
			if scenario.MemberCapitalAllocationRate > 0 {
				warnings = append(warnings, Warning{
					Code:    WarningHighRedemptionLiquidity,
					Path:    scenarioPath(index, "member_capital_allocation_rate"),
					Message: "member capital is substantially redeemable in under twelve months and may create liquidity pressure",
				})
			}
		}
	}

	for index, scenario := range cfg.OrganizationalScenarios {
		if scenario.ExternalDistributionRate > 0 {
			warnings = append(warnings, Warning{
				Code:    WarningExternalDistribution,
				Path:    scenarioPath(index, "external_distribution_rate"),
				Message: "external distribution is a cash outflow and is not included as a success metric",
			})
		}
	}
	return warnings
}

// AssumptionFlags returns model-wide simplifications that should accompany a
// report even when the configuration is otherwise canonical.
func AssumptionFlags(cfg Config) []AssumptionFlag {
	flags := []AssumptionFlag{{
		Code:    AssumptionSimplifiedTaxModel,
		Path:    "company_economics.profit_tax_rate",
		Message: "profit tax is modeled as a simplified rate-and-lag calculation, not as legal or tax advice",
	}}
	names := make([]string, 0, len(cfg.BehaviorCases))
	for name := range cfg.BehaviorCases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		behavior := cfg.BehaviorCases[name]
		if behavior.HighPerformerAttritionDeltaPP == 0 {
			continue
		}
		weightedIndicator := cfg.Workforce.HighPerformerShare * behavior.HighPerformerAttritionDeltaPP
		flags = append(flags, AssumptionFlag{
			Code: AssumptionHighPerformerOnly,
			Path: joinJSONPath("behavior_cases", name) + ".high_performer_attrition_delta_pp",
			Message: fmt.Sprintf(
				"weighted high-performer attrition indicator is %.6g; it is reported as an assumption only because section 8 defines no term for aggregate turnover",
				weightedIndicator,
			),
		})
	}
	return flags
}

func scenarioPath(index int, field string) string {
	return "organizational_scenarios[" + itoa(index) + "]." + field
}
