package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"simworkshare/internal/config"
	"simworkshare/internal/sim"
)

func main() {
	configPath := flag.String("config", "doc/default_config_v0_3_implementation_ready.json", "path to v0.3 JSON configuration")
	profitScenario := flag.String("profit-scenario", "profit_share_equal_10", "profit_share scenario name")
	profitBehavior := flag.String("profit-behavior", "no_effect", "behavior assumption for profit_share")
	flag.Parse()
	if err := run(*configPath, *profitScenario, *profitBehavior, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(configPath, profitScenario, profitBehavior string, stdout, stderr io.Writer) error {
	cfg, err := config.LoadFile(configPath)
	if err != nil {
		return err
	}
	for _, warning := range config.AssumptionWarnings(cfg) {
		fmt.Fprintln(stderr, "warning:", warning)
	}
	result, err := sim.RunDeterministicComparison(cfg, profitScenario, profitBehavior)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("write comparison result: %w", err)
	}
	return nil
}
