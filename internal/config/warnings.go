package config

// AssumptionWarnings returns non-fatal, deterministic validation warnings.
func AssumptionWarnings(cfg Config) []string {
	var warnings []string
	if cfg.Workforce.LostProductivityCostPerLeaver > 0 &&
		cfg.Workforce.TurnoverProductivityPenaltyPerAnnualTurnover > 0 {
		warnings = append(warnings,
			"workforce: lost_productivity_cost_per_leaver and turnover_productivity_penalty_per_annual_turnover may double count turnover impact",
		)
	}
	return warnings
}
