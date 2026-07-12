// Package compatv03 preserves the behavior that the Go implementation of
// SimWorkShare v0.3 actually supported. It intentionally does not turn
// configuration-only v0.3 features into compatibility promises.
package compatv03

import (
	"errors"
	"fmt"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
	"simworkshare/internal/sim"
)

const (
	FixedOnlyScenario = "fixed_only"
	NoEffectBehavior  = "no_effect"
)

// ErrUnsupportedFeature identifies a v0.3 configuration feature for which the
// Go implementation never had an execution path.
var ErrUnsupportedFeature = errors.New("v0.3 compatibility: feature was not implemented")

// LoadFile uses the strict v0.3 parser and validator.
func LoadFile(path string) (config.Config, error) {
	cfg, err := config.LoadFile(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("load v0.3 compatibility config: %w", err)
	}
	return cfg, nil
}

// RunDeterministicFixedOnly runs the canonical fixed_only/no_effect profile
// through the preserved v0.3 engine.
func RunDeterministicFixedOnly(cfg config.Config) (domain.SimulationResult, error) {
	return RunDeterministicScenario(cfg, FixedOnlyScenario, NoEffectBehavior)
}

// RunDeterministicScenario runs only behavior implemented by the v0.3 Go
// engine: fixed_only/no_effect or a monthly profit_share scenario. Quarterly
// and annual profit sharing and fixed_raise_same_expected_cost were present in
// the configuration document but did not have executable implementations.
func RunDeterministicScenario(
	cfg config.Config,
	scenarioName string,
	behaviorCase string,
) (domain.SimulationResult, error) {
	if err := ensureImplemented(cfg, scenarioName, behaviorCase); err != nil {
		return domain.SimulationResult{}, err
	}
	result, err := sim.RunDeterministicScenario(cfg, scenarioName, behaviorCase)
	if err != nil {
		return domain.SimulationResult{}, fmt.Errorf("run v0.3 compatibility scenario %q: %w", scenarioName, err)
	}
	return result, nil
}

// RunDeterministicComparison preserves the implemented fixed-only versus
// monthly-profit-share comparison.
func RunDeterministicComparison(
	cfg config.Config,
	profitShareScenario string,
	behaviorCase string,
) (domain.ComparisonResult, error) {
	if err := ensureImplemented(cfg, FixedOnlyScenario, NoEffectBehavior); err != nil {
		return domain.ComparisonResult{}, err
	}
	if err := ensureImplemented(cfg, profitShareScenario, behaviorCase); err != nil {
		return domain.ComparisonResult{}, err
	}
	result, err := sim.RunDeterministicComparison(cfg, profitShareScenario, behaviorCase)
	if err != nil {
		return domain.ComparisonResult{}, fmt.Errorf(
			"run v0.3 compatibility comparison with %q: %w",
			profitShareScenario,
			err,
		)
	}
	return result, nil
}

func ensureImplemented(cfg config.Config, scenarioName, behaviorCase string) error {
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("validate v0.3 compatibility config: %w", err)
	}

	var selected *config.CompensationScenario
	for i := range cfg.CompensationScenarios {
		if cfg.CompensationScenarios[i].Name == scenarioName {
			selected = &cfg.CompensationScenarios[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("unknown v0.3 compensation scenario %q", scenarioName)
	}

	switch selected.Type {
	case "fixed_only":
		if behaviorCase != NoEffectBehavior {
			return fmt.Errorf(
				"%w: scenario %q supported only behavior case %q",
				ErrUnsupportedFeature,
				scenarioName,
				NoEffectBehavior,
			)
		}
	case "fixed_raise_same_expected_cost":
		return fmt.Errorf(
			"%w: scenario %q uses fixed_raise_same_expected_cost",
			ErrUnsupportedFeature,
			scenarioName,
		)
	case "profit_share":
		if selected.BonusPeriod != "monthly" {
			return fmt.Errorf(
				"%w: scenario %q uses %s profit sharing; v0.3 implemented only monthly profit sharing",
				ErrUnsupportedFeature,
				scenarioName,
				selected.BonusPeriod,
			)
		}
	}
	return nil
}
