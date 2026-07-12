package sim

import (
	"fmt"
	"sort"
	"strings"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

var modelLimitations = []string{
	"behavioral effects are scenario assumptions, not causal estimates",
	"workforce and member capital are aggregated",
	"tax, financing and governance are simplified stylized mechanisms",
	"the model is not a full balance sheet or legal model",
	"sustainable_development_value_proxy is not founder or equity value",
	"high-performer attrition inputs are diagnostic assumptions because section 8 defines no aggregate-turnover term for them",
}

// Run executes the selected scenario matrix. Monthly slices are always kept
// only for one run/case at a time; callers opt into retaining them in the final
// result, which avoids multi-gigabyte output for publication-size runs.
func Run(cfg v04config.Config, options RunOptions) (domain.SimulationResult, error) {
	if err := v04config.Validate(cfg); err != nil {
		return domain.SimulationResult{}, fmt.Errorf("validate v0.4 config: %w", err)
	}
	cases, err := ExpandScenarioCases(cfg, options)
	if err != nil {
		return domain.SimulationResult{}, err
	}

	result := domain.SimulationResult{
		SchemaVersion: cfg.SchemaVersion,
		Mode:          cfg.Simulation.Mode,
		Currency:      cfg.Simulation.Currency,
		MarketCase:    domain.DefaultMarketCase,
		RandomSeed:    cfg.Simulation.RandomSeed,
		Runs:          cfg.Simulation.Runs,
	}
	for _, warning := range v04config.Warnings(cfg) {
		result.Warnings = append(result.Warnings, warning.Code+": "+warning.Path+": "+warning.Message)
	}
	if cfg.Reporting.PrintAssumptionFlags {
		for _, flag := range v04config.AssumptionFlags(cfg) {
			result.AssumptionFlags = append(result.AssumptionFlags, flag.Code+": "+flag.Message)
		}
	}
	if cfg.Reporting.PrintModelLimitations {
		result.ModelLimitations = append([]string(nil), modelLimitations...)
	}

	allRunSummaries := make([]domain.RunTerminalSummary, 0, cfg.Simulation.Runs*len(cases)*len(cfg.Simulation.HorizonsMonths))
	for run := 1; run <= cfg.Simulation.Runs; run++ {
		var commonPath []domain.EnvironmentMonth
		if cfg.Simulation.CommonRandomNumbers {
			commonPath, err = GenerateEnvironmentPath(cfg, run)
			if err != nil {
				return domain.SimulationResult{}, fmt.Errorf("generate environment run %d: %w", run, err)
			}
		}
		for _, scenarioCase := range cases {
			path := commonPath
			if !cfg.Simulation.CommonRandomNumbers {
				path, err = GenerateEnvironmentPath(cfg, run, scenarioCase.Scenario.Name, scenarioCase.BehaviorName)
				if err != nil {
					return domain.SimulationResult{}, fmt.Errorf("generate environment run %d case %s/%s: %w", run, scenarioCase.Scenario.Name, scenarioCase.BehaviorName, err)
				}
			}
			months, err := runScenarioCase(cfg, scenarioCase, run, path)
			if err != nil {
				return domain.SimulationResult{}, fmt.Errorf("run %d scenario %s behavior %s: %w", run, scenarioCase.Scenario.Name, scenarioCase.BehaviorName, err)
			}
			if options.StoreMonthlyResults {
				result.MonthlyResults = append(result.MonthlyResults, months...)
			}
			summaries, err := BuildRunTerminalSummaries(cfg, months)
			if err != nil {
				return domain.SimulationResult{}, fmt.Errorf("summarize run %d scenario %s behavior %s: %w", run, scenarioCase.Scenario.Name, scenarioCase.BehaviorName, err)
			}
			allRunSummaries = append(allRunSummaries, summaries...)
		}
	}
	if options.StoreRunSummaries {
		result.RunTerminalSummaries = append(result.RunTerminalSummaries, allRunSummaries...)
	}
	result.TerminalSummaries = AggregateScenarioSummaries(allRunSummaries)
	result.PairedDeltas = BuildPairedDeltas(allRunSummaries, cfg.Analysis.PairedReferenceScenarios)
	ApplyClassifications(result.TerminalSummaries, result.PairedDeltas, cfg.Analysis.ClassificationTolerance)
	if cfg.Reporting.PrintAssumptionFlags {
		applySummaryAssumptionFlags(cfg, result.TerminalSummaries)
	}
	sortSimulationResult(&result)
	return result, nil
}

func applySummaryAssumptionFlags(cfg v04config.Config, summaries []domain.ScenarioSummary) {
	flags := v04config.AssumptionFlags(cfg)
	for index := range summaries {
		behaviorPrefix := "behavior_cases." + summaries[index].BehaviorCase + "."
		for _, flag := range flags {
			if strings.HasPrefix(flag.Path, "behavior_cases.") && !strings.HasPrefix(flag.Path, behaviorPrefix) {
				continue
			}
			summaries[index].AssumptionFlags = append(
				summaries[index].AssumptionFlags,
				flag.Code+": "+flag.Message,
			)
		}
		sort.Strings(summaries[index].AssumptionFlags)
	}
}

func RunDeterministicScenario(cfg v04config.Config, scenario, behavior string) (domain.SimulationResult, error) {
	if cfg.Simulation.Mode != v04config.ModeDeterministic {
		return domain.SimulationResult{}, fmt.Errorf("simulation.mode must be deterministic")
	}
	return Run(cfg, RunOptions{
		ScenarioNames:       []string{scenario},
		BehaviorCaseNames:   []string{behavior},
		StoreMonthlyResults: true,
		StoreRunSummaries:   true,
	})
}

func runScenarioCase(
	cfg v04config.Config,
	caseConfig ScenarioCase,
	run int,
	path []domain.EnvironmentMonth,
) ([]domain.MonthlyResult, error) {
	if len(path) != cfg.Simulation.Months {
		return nil, fmt.Errorf("environment path has %d months, want %d", len(path), cfg.Simulation.Months)
	}
	queues, err := newRuntimeQueues(cfg)
	if err != nil {
		return nil, err
	}
	state := initialScenarioState(cfg)
	months := make([]domain.MonthlyResult, 0, cfg.Simulation.Months)
	for _, environment := range path {
		monthly, next, err := stepMonth(cfg, caseConfig, run, environment, state, queues)
		if err != nil {
			return nil, err
		}
		months = append(months, monthly)
		state = next
	}
	return months, nil
}

func sortSimulationResult(result *domain.SimulationResult) {
	sort.Slice(result.MonthlyResults, func(i, j int) bool {
		a, b := result.MonthlyResults[i], result.MonthlyResults[j]
		if a.Run != b.Run {
			return a.Run < b.Run
		}
		if a.Scenario != b.Scenario {
			return a.Scenario < b.Scenario
		}
		if a.BehaviorCase != b.BehaviorCase {
			return a.BehaviorCase < b.BehaviorCase
		}
		return a.Month < b.Month
	})
	sort.Slice(result.RunTerminalSummaries, func(i, j int) bool {
		a, b := result.RunTerminalSummaries[i], result.RunTerminalSummaries[j]
		if a.Run != b.Run {
			return a.Run < b.Run
		}
		if a.Scenario != b.Scenario {
			return a.Scenario < b.Scenario
		}
		if a.BehaviorCase != b.BehaviorCase {
			return a.BehaviorCase < b.BehaviorCase
		}
		return a.HorizonMonths < b.HorizonMonths
	})
}
