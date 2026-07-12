package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"simworkshare/internal/domain"
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

func repositoryPath(t *testing.T, elements ...string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(append([]string{root}, elements...)...)
}
