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
	configPath := flag.String("config", "doc/default_config_v0_3_implementation_ready.json", "path to v0.3 JSON configuration")
	profilePath := flag.String("profile", "", "path to a run-profile JSON (overrides config/scenario/behavior)")
	scenario := flag.String("scenario", "fixed_only", "compensation scenario name")
	behavior := flag.String("behavior", "no_effect", "behavior case name")
	flag.Parse()
	var err error
	if *profilePath != "" {
		err = runProfile(*profilePath, os.Stdout, os.Stderr)
	} else {
		err = runScenario(*configPath, *scenario, *behavior, os.Stdout, os.Stderr)
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
