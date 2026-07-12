package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	v04config "simworkshare/internal/v04/config"
	v04domain "simworkshare/internal/v04/domain"
	v04sim "simworkshare/internal/v04/sim"
)

type v04CLIOptions struct {
	Scenario            string
	Behavior            string
	OutputPath          string
	MonthlyCSVPath      string
	SummaryCSVPath      string
	PairedCSVPath       string
	IncludeMonthlyJSON  bool
	IncludeRunSummaries bool
	Mode                string
	Runs                int
	Seed                int64
	RunSensitivity      bool
	SensitivityCSVPath  string
	RunBreakEven        bool
	BreakEvenCSVPath    string
	Reference           string
	Horizon             int
	BreakEvenSteps      int
}

func isV04Config(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read config %q: %w", path, err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return false, fmt.Errorf("decode config %q: %w", path, err)
	}
	_, ok := root["schema_version"]
	if !ok {
		return false, nil
	}
	// Legacy v0.3 documents have no schema_version. Any document that does is
	// routed to the strict v0.4 loader so unknown versions, wrong types, and
	// duplicate schema_version keys receive v0.4 field-path errors rather than
	// being misclassified as legacy input.
	return true, nil
}

func runV04CLI(configPath string, options v04CLIOptions, stdout, stderr io.Writer) error {
	cfg, err := v04config.LoadFile(configPath)
	if err != nil {
		return err
	}
	if options.Mode != "" {
		cfg.Simulation.Mode = options.Mode
		if options.Mode == v04config.ModeDeterministic && options.Runs == 0 {
			cfg.Simulation.Runs = 1
		}
	}
	if options.Runs != 0 {
		cfg.Simulation.Runs = options.Runs
	}
	if options.Seed >= 0 {
		cfg.Simulation.RandomSeed = options.Seed
	}
	if err := v04config.Validate(cfg); err != nil {
		return fmt.Errorf("validate v0.4 CLI overrides: %w", err)
	}
	if options.MonthlyCSVPath != "" && !cfg.Reporting.WriteMonthlyCSV {
		return fmt.Errorf("-monthly-csv is disabled by reporting.write_monthly_csv")
	}
	if options.SummaryCSVPath != "" && !cfg.Reporting.WriteSummaryCSV {
		return fmt.Errorf("-summary-csv is disabled by reporting.write_summary_csv")
	}
	if options.SensitivityCSVPath != "" && !cfg.Reporting.WriteSensitivityCSV {
		return fmt.Errorf("-sensitivity-csv is disabled by reporting.write_sensitivity_csv")
	}
	if options.BreakEvenCSVPath != "" && !cfg.Reporting.WriteBreakEvenCSV {
		return fmt.Errorf("-break-even-csv is disabled by reporting.write_break_even_csv")
	}
	for _, warning := range v04config.Warnings(cfg) {
		fmt.Fprintf(stderr, "warning: %s: %s: %s\n", warning.Code, warning.Path, warning.Message)
	}
	runOptions := v04sim.RunOptions{
		StoreMonthlyResults: options.IncludeMonthlyJSON || options.MonthlyCSVPath != "" || cfg.Simulation.Mode == v04config.ModeDeterministic,
		StoreRunSummaries:   options.IncludeRunSummaries || cfg.Simulation.Mode == v04config.ModeDeterministic,
	}
	if options.Scenario != "" {
		scenarioNames := []string{options.Scenario}
		if options.PairedCSVPath != "" {
			scenarioNames = append(scenarioNames, cfg.Analysis.PairedReferenceScenarios...)
		}
		runOptions.ScenarioNames = uniqueSortedStrings(scenarioNames)
	}
	if options.Behavior != "" {
		runOptions.BehaviorCaseNames = []string{options.Behavior}
	}
	result, err := v04sim.Run(cfg, runOptions)
	if err != nil {
		return err
	}
	if options.RunSensitivity {
		result.SensitivityResults, err = v04sim.RunSensitivity(cfg, runOptions)
		if err != nil {
			return fmt.Errorf("run sensitivity: %w", err)
		}
	}
	if options.RunBreakEven {
		if options.Scenario == "" {
			return fmt.Errorf("-break-even requires -scenario")
		}
		behavior := options.Behavior
		if behavior == "" {
			behavior = "moderate_positive"
		}
		horizon := options.Horizon
		if horizon == 0 {
			horizons := append([]int(nil), cfg.Simulation.HorizonsMonths...)
			sort.Ints(horizons)
			horizon = horizons[len(horizons)-1]
		}
		breakEven, findErr := v04sim.FindBreakEven(
			cfg,
			options.Scenario,
			behavior,
			options.Reference,
			horizon,
			options.BreakEvenSteps,
		)
		if findErr != nil {
			return fmt.Errorf("find break-even: %w", findErr)
		}
		result.BreakEvenResults = []v04domain.BreakEvenResult{breakEven}
	}

	if options.MonthlyCSVPath != "" {
		if err := writeOutputFile(options.MonthlyCSVPath, func(writer io.Writer) error {
			return v04sim.WriteMonthlyCSV(writer, result.MonthlyResults)
		}); err != nil {
			return err
		}
	}
	if options.SummaryCSVPath != "" {
		if err := writeOutputFile(options.SummaryCSVPath, func(writer io.Writer) error {
			return v04sim.WriteSummaryCSV(writer, result.TerminalSummaries)
		}); err != nil {
			return err
		}
	}
	if options.PairedCSVPath != "" {
		if err := writeOutputFile(options.PairedCSVPath, func(writer io.Writer) error {
			return v04sim.WritePairedCSV(writer, result.PairedDeltas)
		}); err != nil {
			return err
		}
	}
	if options.SensitivityCSVPath != "" {
		if err := writeOutputFile(options.SensitivityCSVPath, func(writer io.Writer) error {
			return v04sim.WriteSensitivityCSV(writer, result.SensitivityResults)
		}); err != nil {
			return err
		}
	}
	if options.BreakEvenCSVPath != "" {
		if err := writeOutputFile(options.BreakEvenCSVPath, func(writer io.Writer) error {
			return v04sim.WriteBreakEvenCSV(writer, result.BreakEvenResults)
		}); err != nil {
			return err
		}
	}

	if options.OutputPath != "" {
		return writeOutputFile(options.OutputPath, func(writer io.Writer) error {
			return encodeV04JSON(writer, result)
		})
	}
	return encodeV04JSON(stdout, result)
}

func encodeV04JSON(writer io.Writer, result v04domain.SimulationResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("write v0.4 JSON result: %w", err)
	}
	return nil
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func writeOutputFile(path string, write func(io.Writer) error) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output %q: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close output %q: %w", path, closeErr)
		}
	}()
	if err := write(file); err != nil {
		return fmt.Errorf("write output %q: %w", path, err)
	}
	return nil
}
