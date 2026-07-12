package runprofile

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"simworkshare/internal/config"
)

func TestComparisonProfilesShareIdenticalEconomics(t *testing.T) {
	root := repositoryRoot(t)
	fixed, fixedConfigPath, err := LoadFile(filepath.Join(root, "profiles", "fixed_only.json"))
	if err != nil {
		t.Fatalf("load fixed profile: %v", err)
	}
	profit, profitConfigPath, err := LoadFile(filepath.Join(root, "profiles", "profit_share_10.json"))
	if err != nil {
		t.Fatalf("load profit profile: %v", err)
	}
	if fixedConfigPath != profitConfigPath {
		t.Fatalf("profiles use different economics: %q vs %q", fixedConfigPath, profitConfigPath)
	}
	if fixed.Scenario != "fixed_only" || profit.Scenario != "profit_share_equal_10" {
		t.Fatalf("profile scenarios = %q/%q", fixed.Scenario, profit.Scenario)
	}
	if fixed.BehaviorCase != "no_effect" || profit.BehaviorCase != "no_effect" {
		t.Fatalf("profile behavior cases = %q/%q", fixed.BehaviorCase, profit.BehaviorCase)
	}
	if fixed.CalibrationStatus != "template_defaults_not_real_data" ||
		profit.CalibrationStatus != "template_defaults_not_real_data" {
		t.Fatalf("profile calibration statuses = %q/%q", fixed.CalibrationStatus, profit.CalibrationStatus)
	}

	fixedConfig, err := config.LoadFile(fixedConfigPath)
	if err != nil {
		t.Fatalf("load fixed base config: %v", err)
	}
	profitConfig, err := config.LoadFile(profitConfigPath)
	if err != nil {
		t.Fatalf("load profit base config: %v", err)
	}
	if !reflect.DeepEqual(fixedConfig, profitConfig) {
		t.Fatal("comparison profiles do not share identical non-selection inputs")
	}
}

func TestLoadFileRejectsDuplicateSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplicate.json")
	data := []byte(`{
  "base_config": "base.json",
  "scenario": "fixed_only",
  "scenario": "profit_share_equal_10",
  "behavior_case": "no_effect",
  "environment_case": "normal_market",
  "calibration_status": "template_defaults_not_real_data"
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write duplicate profile: %v", err)
	}
	if _, _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "scenario: duplicate field") {
		t.Fatalf("LoadFile() error = %v, want duplicate scenario error", err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
