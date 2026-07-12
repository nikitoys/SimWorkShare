package config

import "testing"

func TestSetNumericPathSupportsBehaviorScenarioAndTopLevel(t *testing.T) {
	cfg := defaultConfig(t)
	copyConfig := cfg.DeepCopy()
	for path, value := range map[string]float64{
		"behavior_cases.moderate_positive.ownership_productivity_sensitivity":                             0.123,
		"organizational_scenarios.worker_cooperative.governance.base_governance_hours_per_employee_month": 7,
		"company_economics.required_cash_reserve_months":                                                  4,
	} {
		if err := SetNumericPath(&copyConfig, path, value); err != nil {
			t.Fatalf("SetNumericPath(%q): %v", path, err)
		}
	}
	if copyConfig.BehaviorCases["moderate_positive"].OwnershipProductivitySensitivity != 0.123 {
		t.Fatal("behavior map value was not updated")
	}
	scenario, _ := copyConfig.ScenarioByName("worker_cooperative")
	if scenario.Governance.BaseGovernanceHoursPerEmployeeMonth != 7 {
		t.Fatal("named scenario value was not updated")
	}
	if cfg.BehaviorCases["moderate_positive"].OwnershipProductivitySensitivity == 0.123 {
		t.Fatal("DeepCopy mutation leaked into source config")
	}
}
