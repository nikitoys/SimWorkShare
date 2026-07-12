package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"simworkshare/internal/domain"
)

func TestRunWritesPairedComparison(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run(
		repositoryPath(t, "doc", "default_config_v0_3_implementation_ready.json"),
		"profit_share_equal_10",
		"no_effect",
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &root); err != nil {
		t.Fatalf("decode comparison JSON object: %v", err)
	}
	for _, key := range []string{"fixed_only", "profit_share", "summary"} {
		if _, ok := root[key]; !ok {
			t.Fatalf("comparison JSON is missing key %q", key)
		}
	}

	var result domain.ComparisonResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode comparison JSON: %v", err)
	}
	if len(result.FixedOnly.MonthlyResults) != 60 || len(result.ProfitShare.MonthlyResults) != 60 {
		t.Fatalf("comparison months = %d/%d, want 60/60",
			len(result.FixedOnly.MonthlyResults), len(result.ProfitShare.MonthlyResults))
	}
	if result.Summary.ProfitShareScenario != "profit_share_equal_10" ||
		result.ProfitShare.TerminalSummary.Compensation == nil {
		t.Fatalf("unexpected comparison summary: %+v", result.Summary)
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
