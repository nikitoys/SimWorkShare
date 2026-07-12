package sim

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"simworkshare/internal/config"
	"simworkshare/internal/domain"
)

func TestDeterministicBaselineIsRepeatable(t *testing.T) {
	cfg := loadDefaultConfig(t)
	first, err := RunDeterministicBaseline(cfg)
	if err != nil {
		t.Fatalf("first run error = %v", err)
	}
	second, err := RunDeterministicBaseline(cfg)
	if err != nil {
		t.Fatalf("second run error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated results differ:\nfirst=%+v\nsecond=%+v", first, second)
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("json.Marshal(first) error = %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("json.Marshal(second) error = %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("canonical JSON differs between repeated runs")
	}
}

func TestDeterministicBaselineRejectsNonFiniteDerivedValues(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Company.BaseSalaryPerEmployee = math.MaxFloat64

	_, err := RunDeterministicBaseline(cfg)
	if err == nil {
		t.Fatal("RunDeterministicBaseline() error = nil, want integrity error")
	}
}

func TestDeterministicBaselineRejectsUnsupportedTaxLag(t *testing.T) {
	cfg := loadDefaultConfig(t)
	cfg.Cashflow.ProfitTaxPaymentLagMonths = -1

	_, err := RunDeterministicBaseline(cfg)
	if err == nil {
		t.Fatal("RunDeterministicBaseline() error = nil, want tax-lag error")
	}
}

func TestDeterministicBaselineContract(t *testing.T) {
	cfg := loadDefaultConfig(t)
	result, err := RunDeterministicBaseline(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicBaseline() error = %v", err)
	}
	if result.Scenario != "fixed_only" || result.BehaviorCase != "no_effect" || result.EnvironmentCase != "normal_market" {
		t.Fatalf("unexpected scenario key: %s/%s/%s", result.Scenario, result.BehaviorCase, result.EnvironmentCase)
	}
	if result.Month != 1 || result.Environment.MarketFactor != 1 || result.Environment.ShockHappened {
		t.Fatalf("unexpected deterministic environment: %+v", result.Environment)
	}
	if result.PnL.BonusExpenseAccrual != 0 || result.Cash.RestrictedBonusCash != 0 {
		t.Fatal("fixed_only produced bonus state")
	}
	if !domain.MoneyAlmostEqual(result.PnL.OperatingProfitBeforeBonus, 1_509_853.311710948) {
		t.Fatalf("operating profit = %.12f", result.PnL.OperatingProfitBeforeBonus)
	}
}

func TestDeterministicBaselineMatchesGoldenJSON(t *testing.T) {
	cfg := loadDefaultConfig(t)
	result, err := RunDeterministicBaseline(cfg)
	if err != nil {
		t.Fatalf("RunDeterministicBaseline() error = %v", err)
	}
	got, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "month1_golden.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(golden) error = %v", err)
	}
	if string(got)+"\n" != string(want) {
		t.Fatalf("month 1 JSON changed\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func loadDefaultConfig(t *testing.T) config.Config {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	cfg, err := config.LoadFile(filepath.Join(root, "doc", "default_config_v0_3_implementation_ready.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	return cfg
}
