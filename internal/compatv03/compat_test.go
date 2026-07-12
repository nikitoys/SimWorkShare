package compatv03

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"simworkshare/internal/config"
)

func TestImplementedMonthOneResultsMatchV03GoldenFixtures(t *testing.T) {
	cfg := loadDefaultConfig(t)
	hashFixturePath := filepath.Join(repositoryRoot(t), "internal", "compatv03", "testdata", "full_output_sha256.json")
	hashFixtureData, err := os.ReadFile(hashFixturePath)
	if err != nil {
		t.Fatalf("read full-output compatibility hashes: %v", err)
	}
	var fullOutputHashes map[string]string
	if err := json.Unmarshal(hashFixtureData, &fullOutputHashes); err != nil {
		t.Fatalf("decode full-output compatibility hashes: %v", err)
	}
	tests := []struct {
		name         string
		scenario     string
		behaviorCase string
		golden       string
	}{
		{
			name:         "fixed_only/no_effect",
			scenario:     FixedOnlyScenario,
			behaviorCase: NoEffectBehavior,
			golden:       "month1_golden.json",
		},
		{
			name:         "profit_share_equal_10/no_effect",
			scenario:     "profit_share_equal_10",
			behaviorCase: NoEffectBehavior,
			golden:       "profit_share_month1_golden.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunDeterministicScenario(cfg, tt.scenario, tt.behaviorCase)
			if err != nil {
				t.Fatalf("RunDeterministicScenario() error = %v", err)
			}
			if len(result.MonthlyResults) == 0 {
				t.Fatal("RunDeterministicScenario() returned no monthly results")
			}
			full, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(full)
			gotHash := hex.EncodeToString(sum[:])
			key := tt.scenario + "/" + tt.behaviorCase
			if gotHash != fullOutputHashes[key] {
				t.Fatalf("full v0.3 output hash for %s = %s, want %s", key, gotHash, fullOutputHashes[key])
			}

			got, err := json.Marshal(result.MonthlyResults[0])
			if err != nil {
				t.Fatalf("marshal month 1: %v", err)
			}
			goldenPath := filepath.Join(
				repositoryRoot(t),
				"internal",
				"sim",
				"testdata",
				tt.golden,
			)
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read v0.3 compatibility fixture %q: %v", goldenPath, err)
			}
			var compactWant bytes.Buffer
			if err := json.Compact(&compactWant, want); err != nil {
				t.Fatalf("compact v0.3 compatibility fixture: %v", err)
			}
			if !bytes.Equal(got, compactWant.Bytes()) {
				t.Fatalf(
					"v0.3 month-1 compatibility changed\n--- got ---\n%s\n--- want ---\n%s",
					got,
					compactWant.Bytes(),
				)
			}
		})
	}
}

func TestConfigurationOnlyV03ModesReturnUnsupportedError(t *testing.T) {
	cfg := loadDefaultConfig(t)
	tests := []struct {
		name     string
		scenario string
		wantText string
	}{
		{
			name:     "fixed raise",
			scenario: "fixed_raise_same_expected_cost_for_10",
			wantText: "uses fixed_raise_same_expected_cost",
		},
		{
			name:     "quarterly profit sharing",
			scenario: "profit_share_equal_10_quarterly",
			wantText: "uses quarterly profit sharing; v0.3 implemented only monthly profit sharing",
		},
		{
			name:     "annual profit sharing",
			scenario: "profit_share_equal_10_annual",
			wantText: "uses annual profit sharing; v0.3 implemented only monthly profit sharing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RunDeterministicScenario(cfg, tt.scenario, NoEffectBehavior)
			if err == nil {
				t.Fatal("RunDeterministicScenario() error = nil, want unsupported error")
			}
			if !errors.Is(err, ErrUnsupportedFeature) {
				t.Fatalf("error = %v, want ErrUnsupportedFeature", err)
			}
			if !strings.Contains(err.Error(), tt.scenario) || !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("error = %q, want scenario and %q", err, tt.wantText)
			}
		})
	}
}

func loadDefaultConfig(t *testing.T) config.Config {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), "doc", "default_config_v0_3_implementation_ready.json")
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile(%q) error = %v", path, err)
	}
	return cfg
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
