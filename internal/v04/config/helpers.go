package config

import "strconv"

// ScenarioByName returns a pointer to the named scenario in c. The pointer is
// suitable for sensitivity mutations on a deep-copied configuration.
func (c *Config) ScenarioByName(name string) (*OrganizationalScenario, bool) {
	if c == nil {
		return nil, false
	}
	for index := range c.OrganizationalScenarios {
		if c.OrganizationalScenarios[index].Name == name {
			return &c.OrganizationalScenarios[index], true
		}
	}
	return nil, false
}

// BehaviorByName returns the named behavior case.
func (c Config) BehaviorByName(name string) (BehaviorCase, bool) {
	behavior, ok := c.BehaviorCases[name]
	return behavior, ok
}

// SetBehavior stores a behavior case. It is useful because Go map values are
// not addressable directly when a sensitivity changes one field.
func (c *Config) SetBehavior(name string, behavior BehaviorCase) {
	if c == nil {
		return
	}
	if c.BehaviorCases == nil {
		c.BehaviorCases = make(map[string]BehaviorCase)
	}
	c.BehaviorCases[name] = behavior
}

// DeepCopy returns an independent structural copy. In particular all maps,
// slices and nullable caps are copied, so sensitivity runs cannot mutate the
// base configuration or each other.
func (c Config) DeepCopy() Config {
	copyConfig := c
	copyConfig.Units = cloneStringMap(c.Units)
	copyConfig.Simulation.HorizonsMonths = cloneSlice(c.Simulation.HorizonsMonths)
	copyConfig.Market.SeasonalityMultipliers = cloneSlice(c.Market.SeasonalityMultipliers)
	copyConfig.Workforce.RampProductivityMultipliers = cloneSlice(c.Workforce.RampProductivityMultipliers)
	if c.BehaviorCases != nil {
		copyConfig.BehaviorCases = make(map[string]BehaviorCase, len(c.BehaviorCases))
		for name, behavior := range c.BehaviorCases {
			copyConfig.BehaviorCases[name] = behavior
		}
	}
	copyConfig.OrganizationalScenarios = cloneSlice(c.OrganizationalScenarios)
	for index := range copyConfig.OrganizationalScenarios {
		original := c.OrganizationalScenarios[index]
		copyConfig.OrganizationalScenarios[index].AllocationPriority = cloneSlice(original.AllocationPriority)
		copyConfig.OrganizationalScenarios[index].BehaviorCaseRefs = cloneSlice(original.BehaviorCaseRefs)
		if original.MaxDistributionPerEmployeePeriod != nil {
			capValue := *original.MaxDistributionPerEmployeePeriod
			copyConfig.OrganizationalScenarios[index].MaxDistributionPerEmployeePeriod = &capValue
		}
	}
	copyConfig.Analysis.PairedReferenceScenarios = cloneSlice(c.Analysis.PairedReferenceScenarios)
	copyConfig.Analysis.BreakEvenUpliftRange = cloneSlice(c.Analysis.BreakEvenUpliftRange)
	copyConfig.Analysis.SensitivityParameters = cloneSlice(c.Analysis.SensitivityParameters)
	for index := range copyConfig.Analysis.SensitivityParameters {
		copyConfig.Analysis.SensitivityParameters[index].Values = cloneSlice(c.Analysis.SensitivityParameters[index].Values)
	}
	return copyConfig
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneSlice[T any](source []T) []T {
	if source == nil {
		return nil
	}
	return append([]T(nil), source...)
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
