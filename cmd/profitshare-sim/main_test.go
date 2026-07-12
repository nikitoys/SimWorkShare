package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"simworkshare/internal/domain"
	v04config "simworkshare/internal/v04/config"
	v04domain "simworkshare/internal/v04/domain"
)

func TestRunWritesMultiMonthResult(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"), &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &root); err != nil {
		t.Fatalf("decode CLI JSON object: %v", err)
	}
	for _, key := range []string{"monthly_results", "terminal_summary"} {
		if _, ok := root[key]; !ok {
			t.Fatalf("CLI JSON is missing top-level key %q", key)
		}
	}

	var result domain.SimulationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode CLI JSON: %v", err)
	}
	if len(result.MonthlyResults) != 60 {
		t.Fatalf("monthly result count = %d, want 60", len(result.MonthlyResults))
	}
	last := result.MonthlyResults[len(result.MonthlyResults)-1]
	if result.TerminalSummary.FinalMonth != 60 || result.TerminalSummary.ClosingCashTotal != last.Cash.ClosingCashTotal {
		t.Fatalf("terminal summary does not match final month: %+v", result.TerminalSummary)
	}
}

func TestRunScenarioWritesProfitShareResult(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runScenario(
		repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"),
		"profit_share_equal_10",
		"no_effect",
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("runScenario() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var result domain.SimulationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode profit_share CLI JSON: %v", err)
	}
	if result.Scenario != "profit_share_equal_10" || len(result.MonthlyResults) != 60 {
		t.Fatalf("scenario/months = %q/%d", result.Scenario, len(result.MonthlyResults))
	}
	if result.MonthlyResults[0].Compensation == nil || result.TerminalSummary.Compensation == nil {
		t.Fatal("profit_share CLI output has no compensation state")
	}
}

func TestRunProfileSelectsScenarioAndWarnsAboutTemplateData(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runProfile(repositoryPath(t, "profiles", "profit_share_10.json"), &stdout, &stderr); err != nil {
		t.Fatalf("runProfile() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "template defaults") {
		t.Fatalf("stderr = %q, want calibration warning", stderr.String())
	}
	var result domain.SimulationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode profile result: %v", err)
	}
	if result.Scenario != "profit_share_equal_10" {
		t.Fatalf("profile scenario = %q", result.Scenario)
	}
}

func TestRunV04CLIProducesCleanJSONAndCSV(t *testing.T) {
	configPath := writeSmallV04Config(t)
	monthlyPath := filepath.Join(t.TempDir(), "monthly.csv")
	summaryPath := filepath.Join(t.TempDir(), "summary.csv")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runV04CLI(configPath, v04CLIOptions{
		Scenario:       "traditional_company",
		Behavior:       "no_effect",
		MonthlyCSVPath: monthlyPath,
		SummaryCSVPath: summaryPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runV04CLI() error = %v", err)
	}
	if strings.Contains(stdout.String(), "warning:") {
		t.Fatalf("JSON stdout contains a warning: %q", stdout.String())
	}

	var result v04domain.SimulationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode v0.4 CLI JSON: %v", err)
	}
	if result.SchemaVersion != "0.4" || result.MarketCase != v04domain.DefaultMarketCase {
		t.Fatalf("result identity = version %q market %q", result.SchemaVersion, result.MarketCase)
	}
	if len(result.MonthlyResults) != 2 || len(result.RunTerminalSummaries) != 2 || len(result.TerminalSummaries) != 2 {
		t.Fatalf("monthly/run-summary/summary rows = %d/%d/%d, want 2/2/2",
			len(result.MonthlyResults), len(result.RunTerminalSummaries), len(result.TerminalSummaries))
	}
	for _, path := range []string{monthlyPath, summaryPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read CSV %q: %v", path, err)
		}
		if len(data) == 0 {
			t.Fatalf("CSV %q is empty", path)
		}
	}
}

func TestRunV04CLIOverridesDefaultMonteCarloWithDeterministicMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runV04CLI(repositoryPath(t, "doc", "default_config_v0_4.json"), v04CLIOptions{
		Scenario: "traditional_company",
		Behavior: "no_effect",
		Mode:     v04config.ModeDeterministic,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runV04CLI() error = %v", err)
	}
	var result v04domain.SimulationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Mode != v04config.ModeDeterministic || result.Runs != 1 {
		t.Fatalf("mode/runs = %q/%d, want deterministic/1", result.Mode, result.Runs)
	}
}

func TestRunV04CLIPairedCSVIncludesConfiguredReferences(t *testing.T) {
	pairedPath := filepath.Join(t.TempDir(), "paired.csv")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runV04CLI(writeSmallV04Config(t), v04CLIOptions{
		Scenario:      "worker_cooperative",
		Behavior:      "no_effect",
		PairedCSVPath: pairedPath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runV04CLI() error = %v", err)
	}
	var result v04domain.SimulationResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode paired result: %v", err)
	}
	if len(result.MonthlyResults) != 6 {
		t.Fatalf("monthly rows = %d, want candidate plus two references over two months", len(result.MonthlyResults))
	}
	references := map[string]bool{}
	for _, delta := range result.PairedDeltas {
		if delta.Scenario == "worker_cooperative" {
			references[delta.ReferenceScenario] = true
		}
	}
	for _, reference := range []string{"traditional_company", "profit_sharing"} {
		if !references[reference] {
			t.Fatalf("paired result is missing reference %q; got %v", reference, references)
		}
	}
	data, err := os.ReadFile(pairedPath)
	if err != nil || !strings.Contains(string(data), "worker_cooperative") {
		t.Fatalf("paired CSV is missing candidate rows: err=%v data=%q", err, data)
	}
}

func TestDetectsV04AndLegacyConfigurations(t *testing.T) {
	v04, err := isV04Config(repositoryPath(t, "doc", "default_config_v0_4.json"))
	if err != nil || !v04 {
		t.Fatalf("v0.4 detection = %v, %v", v04, err)
	}
	v04, err = isV04Config(repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil || v04 {
		t.Fatalf("legacy detection = %v, %v", v04, err)
	}
	unknownVersion := filepath.Join(t.TempDir(), "unknown-version.json")
	if err := os.WriteFile(unknownVersion, []byte(`{"schema_version":"0.5"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	v04, err = isV04Config(unknownVersion)
	if err != nil || !v04 {
		t.Fatalf("unknown schema version must route to strict v0.4 validation: %v, %v", v04, err)
	}
}

func TestRunV04CLIHonorsReportingFormatDisable(t *testing.T) {
	path := writeSmallV04ConfigWith(t, func(cfg *v04config.Config) {
		cfg.Reporting.WriteSummaryCSV = false
	})
	err := runV04CLI(path, v04CLIOptions{
		Scenario:       "traditional_company",
		Behavior:       "no_effect",
		SummaryCSVPath: filepath.Join(t.TempDir(), "summary.csv"),
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "reporting.write_summary_csv") {
		t.Fatalf("disabled summary CSV error = %v", err)
	}
}

func writeSmallV04Config(t *testing.T) string {
	return writeSmallV04ConfigWith(t, nil)
}

func writeSmallV04ConfigWith(t *testing.T, mutate func(*v04config.Config)) string {
	t.Helper()
	cfg, err := v04config.LoadFile(repositoryPath(t, "doc", "default_config_v0_4.json"))
	if err != nil {
		t.Fatalf("load v0.4 default: %v", err)
	}
	cfg.Simulation.Mode = v04config.ModeDeterministic
	cfg.Simulation.Runs = 1
	cfg.Simulation.Months = 2
	cfg.Simulation.HorizonsMonths = []int{1, 2}
	if mutate != nil {
		mutate(&cfg)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal small v0.4 config: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write small v0.4 config: %v", err)
	}
	return path
}

func repositoryPath(t *testing.T, elements ...string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(append([]string{root}, elements...)...)
}
