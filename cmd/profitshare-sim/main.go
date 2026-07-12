package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"simworkshare/internal/config"
	"simworkshare/internal/runprofile"
	"simworkshare/internal/sim"
)

func main() {
	configPath := flag.String("config", "doc/default_config_v0_4.json", "path to a v0.4 or legacy v0.3 JSON configuration")
	profilePath := flag.String("profile", "", "path to a run-profile JSON (overrides config/scenario/behavior)")
	scenario := flag.String("scenario", "", "organizational scenario (v0.4) or compensation scenario (v0.3); empty runs the v0.4 matrix")
	behavior := flag.String("behavior", "", "behavior case; empty runs every enabled v0.4 case")
	outputPath := flag.String("output", "", "write JSON result to file instead of stdout")
	monthlyCSV := flag.String("monthly-csv", "", "write v0.4 monthly results CSV")
	summaryCSV := flag.String("summary-csv", "", "write v0.4 terminal summary CSV")
	pairedCSV := flag.String("paired-csv", "", "write v0.4 paired comparison CSV")
	includeMonthlyJSON := flag.Bool("include-monthly-json", false, "include v0.4 Monte Carlo monthly rows in JSON")
	includeRunSummaries := flag.Bool("include-run-summaries", false, "include per-run v0.4 terminal summaries in JSON")
	mode := flag.String("mode", "", "override v0.4 simulation mode: deterministic or monte_carlo")
	runs := flag.Int("runs", 0, "override v0.4 Monte Carlo run count; 0 keeps the configuration value")
	seed := flag.Int64("seed", -1, "override v0.4 random seed; -1 keeps the configuration value")
	sensitivity := flag.Bool("sensitivity", false, "run the configured v0.4 sensitivity grid")
	sensitivityCSV := flag.String("sensitivity-csv", "", "write v0.4 sensitivity CSV")
	breakEven := flag.Bool("break-even", false, "find the v0.4 break-even productivity uplift for the selected scenario")
	breakEvenCSV := flag.String("break-even-csv", "", "write v0.4 break-even CSV")
	reference := flag.String("reference", "traditional_company", "break-even reference scenario")
	horizon := flag.Int("horizon", 0, "break-even horizon; 0 uses the longest configured horizon")
	breakEvenSteps := flag.Int("break-even-steps", 40, "coarse break-even grid intervals before refinement")
	flag.Parse()
	var err error
	if *profilePath != "" {
		err = runProfile(*profilePath, os.Stdout, os.Stderr)
	} else {
		isV04, detectErr := isV04Config(*configPath)
		if detectErr != nil {
			err = detectErr
		} else if isV04 {
			err = runV04CLI(*configPath, v04CLIOptions{
				Scenario:            *scenario,
				Behavior:            *behavior,
				OutputPath:          *outputPath,
				MonthlyCSVPath:      *monthlyCSV,
				SummaryCSVPath:      *summaryCSV,
				PairedCSVPath:       *pairedCSV,
				IncludeMonthlyJSON:  *includeMonthlyJSON,
				IncludeRunSummaries: *includeRunSummaries,
				Mode:                *mode,
				Runs:                *runs,
				Seed:                *seed,
				RunSensitivity:      *sensitivity || *sensitivityCSV != "",
				SensitivityCSVPath:  *sensitivityCSV,
				RunBreakEven:        *breakEven || *breakEvenCSV != "",
				BreakEvenCSVPath:    *breakEvenCSV,
				Reference:           *reference,
				Horizon:             *horizon,
				BreakEvenSteps:      *breakEvenSteps,
			}, os.Stdout, os.Stderr)
		} else {
			legacyScenario := *scenario
			if legacyScenario == "" {
				legacyScenario = "fixed_only"
			}
			legacyBehavior := *behavior
			if legacyBehavior == "" {
				legacyBehavior = "no_effect"
			}
			err = runScenario(*configPath, legacyScenario, legacyBehavior, os.Stdout, os.Stderr)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runProfile(profilePath string, stdout, stderr io.Writer) error {
	profile, configPath, err := runprofile.LoadFile(profilePath)
	if err != nil {
		return err
	}
	if profile.CalibrationStatus != "calibrated" {
		fmt.Fprintln(stderr, "warning: run profile uses template defaults, not calibrated company data")
	}
	return runScenario(configPath, profile.Scenario, profile.BehaviorCase, stdout, stderr)
}

func run(configPath string, stdout, stderr io.Writer) error {
	return runScenario(configPath, "fixed_only", "no_effect", stdout, stderr)
}

func runScenario(configPath, scenario, behavior string, stdout, stderr io.Writer) error {
	cfg, err := config.LoadFile(configPath)
	if err != nil {
		return err
	}
	for _, warning := range config.AssumptionWarnings(cfg) {
		fmt.Fprintln(stderr, "warning:", warning)
	}
	result, err := sim.RunDeterministicScenario(cfg, scenario, behavior)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("write result: %w", err)
	}
	return nil
}
