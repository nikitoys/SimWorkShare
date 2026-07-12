package sim

import (
	"fmt"
	"sort"

	v04config "simworkshare/internal/v04/config"
)

type ScenarioCase struct {
	Scenario     v04config.OrganizationalScenario
	BehaviorName string
	Behavior     v04config.BehaviorCase
}

type RunOptions struct {
	ScenarioNames       []string
	BehaviorCaseNames   []string
	StoreMonthlyResults bool
	StoreRunSummaries   bool
}

func ExpandScenarioCases(cfg v04config.Config, options RunOptions) ([]ScenarioCase, error) {
	scenarioFilter := make(map[string]bool, len(options.ScenarioNames))
	for _, name := range options.ScenarioNames {
		scenarioFilter[name] = true
	}
	behaviorFilter := make(map[string]bool, len(options.BehaviorCaseNames))
	for _, name := range options.BehaviorCaseNames {
		behaviorFilter[name] = true
	}
	seenScenarios := make(map[string]bool)
	seenBehaviors := make(map[string]bool)
	var cases []ScenarioCase
	for _, scenario := range cfg.OrganizationalScenarios {
		if len(scenarioFilter) > 0 && !scenarioFilter[scenario.Name] {
			continue
		}
		seenScenarios[scenario.Name] = true
		for _, behaviorName := range scenario.BehaviorCaseRefs {
			if len(behaviorFilter) > 0 && !behaviorFilter[behaviorName] {
				continue
			}
			behavior, ok := cfg.BehaviorCases[behaviorName]
			if !ok {
				return nil, fmt.Errorf("scenario %q references unknown behavior case %q", scenario.Name, behaviorName)
			}
			seenBehaviors[behaviorName] = true
			cases = append(cases, ScenarioCase{Scenario: scenario, BehaviorName: behaviorName, Behavior: behavior})
		}
	}
	for name := range scenarioFilter {
		if !seenScenarios[name] {
			return nil, fmt.Errorf("unknown organizational scenario %q", name)
		}
	}
	for name := range behaviorFilter {
		if _, exists := cfg.BehaviorCases[name]; !exists {
			return nil, fmt.Errorf("unknown behavior case %q", name)
		}
		if !seenBehaviors[name] {
			return nil, fmt.Errorf("behavior case %q is not enabled for any selected scenario", name)
		}
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("scenario expansion produced no cases")
	}
	sort.Slice(cases, func(i, j int) bool {
		if cases[i].Scenario.Name != cases[j].Scenario.Name {
			return cases[i].Scenario.Name < cases[j].Scenario.Name
		}
		return cases[i].BehaviorName < cases[j].BehaviorName
	})
	return cases, nil
}
