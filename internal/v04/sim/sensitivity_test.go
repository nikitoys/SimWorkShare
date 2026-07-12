package sim

import (
	"bytes"
	"encoding/csv"
	"math"
	"reflect"
	"strings"
	"testing"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func TestRunSensitivityDeepCopiesSelectsAndPairsStably(t *testing.T) {
	cfg := deterministicConfig(t, 2)
	cfg.Analysis.BreakEvenMetric = "" // The analysis runner supplies the documented default.
	cfg.Analysis.SensitivityParameters = []v04config.SensitivityParameter{
		{
			Path:   "behavior_cases.moderate_positive.ownership_productivity_sensitivity",
			Values: []float64{0.04, 0},
		},
	}
	original := cfg.DeepCopy()
	options := RunOptions{
		ScenarioNames:     []string{v04config.SystemWorkerCooperative},
		BehaviorCaseNames: []string{"moderate_positive"},
	}

	first, err := RunSensitivity(cfg, options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunSensitivity(cfg, options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("sensitivity result changed across identical calls")
	}
	if !reflect.DeepEqual(cfg, original) {
		t.Fatal("RunSensitivity mutated the base config")
	}
	if len(first) != 4 {
		t.Fatalf("sensitivity rows = %d, want 4", len(first))
	}

	wantValues := []float64{0, 0, 0.04, 0.04}
	wantHorizons := []int{1, 2, 1, 2}
	for index, row := range first {
		if row.ParameterValue != wantValues[index] || row.HorizonMonths != wantHorizons[index] {
			t.Fatalf("row %d is not stably sorted: %+v", index, row)
		}
		if row.Scenario != v04config.SystemWorkerCooperative || row.BehaviorCase != "moderate_positive" {
			t.Fatalf("row %d leaked an internal reference case: %+v", index, row)
		}
		if row.Metric != v04config.BreakEvenSustainableDevelopment {
			t.Fatalf("row %d metric = %q", index, row.Metric)
		}
		if row.Classification == "" || row.Classification == ClassificationUnclassified {
			t.Fatalf("row %d has no paired classification: %+v", index, row)
		}
		if !domain.Finite(row.MedianValue) || !domain.Finite(row.MedianPairedDelta) {
			t.Fatalf("row %d is non-finite: %+v", index, row)
		}
	}

	// Classification must be copied from the terminal summary produced with
	// the internally included references, rather than inferred by this runner.
	expectedConfig := original.DeepCopy()
	expectedConfig.Analysis.BreakEvenMetric = v04config.BreakEvenSustainableDevelopment
	if err := v04config.SetNumericPath(&expectedConfig, first[0].ParameterPath, first[0].ParameterValue); err != nil {
		t.Fatal(err)
	}
	expected, err := Run(expectedConfig, RunOptions{
		ScenarioNames: []string{
			v04config.SystemTraditionalCompany,
			v04config.SystemProfitSharing,
			v04config.SystemWorkerCooperative,
		},
		BehaviorCaseNames: []string{"moderate_positive"},
		StoreRunSummaries: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := findTerminalScenarioSummary(
		expected.TerminalSummaries,
		v04config.SystemWorkerCooperative,
		"moderate_positive",
		first[0].HorizonMonths,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first[0].Classification != summary.Classification {
		t.Fatalf("classification = %q, terminal summary = %q", first[0].Classification, summary.Classification)
	}
}

func TestTerminalMetricValueSupportsAnalysisEnumsAndRejectsNonFinite(t *testing.T) {
	summary := domain.RunTerminalSummary{
		SustainableDevelopmentValueProxy: 11,
		CashEndUnrestricted:              22,
		ProductiveCapacityGrowthRate:     0.33,
		RevenueCAGR:                      0.44,
	}
	tests := []struct {
		metric string
		want   float64
	}{
		{"", 11},
		{v04config.BreakEvenSustainableDevelopment, 11},
		{v04config.BreakEvenCashEndUnrestricted, 22},
		{v04config.BreakEvenCapacityGrowth, 0.33},
		{v04config.BreakEvenRevenueCAGR, 0.44},
	}
	for _, test := range tests {
		got, err := terminalMetricValue(summary, test.metric)
		if err != nil {
			t.Fatalf("terminalMetricValue(%q): %v", test.metric, err)
		}
		if got != test.want {
			t.Fatalf("terminalMetricValue(%q) = %g, want %g", test.metric, got, test.want)
		}
	}
	if _, err := terminalMetricValue(summary, "unsupported"); err == nil {
		t.Fatal("unsupported analysis metric was accepted")
	}
	summary.RevenueCAGR = math.Inf(1)
	if _, err := terminalMetricValue(summary, v04config.BreakEvenRevenueCAGR); err == nil {
		t.Fatal("non-finite analysis metric was accepted")
	}
}

func TestRunSensitivityValidatesEveryMutatedCopy(t *testing.T) {
	cfg := deterministicConfig(t, 1)
	cfg.Analysis.SensitivityParameters = []v04config.SensitivityParameter{
		{
			Path:   "company_economics.required_cash_reserve_months",
			Values: []float64{25},
		},
	}
	original := cfg.DeepCopy()
	_, err := RunSensitivity(cfg, RunOptions{
		ScenarioNames:     []string{v04config.SystemWorkerCooperative},
		BehaviorCaseNames: []string{noEffectBehaviorCase},
	})
	if err == nil || !strings.Contains(err.Error(), "analysis.sensitivity_parameters[0].values[0]") ||
		!strings.Contains(err.Error(), "required_cash_reserve_months") {
		t.Fatalf("mutated validation error = %v", err)
	}
	if !reflect.DeepEqual(cfg, original) {
		t.Fatal("invalid sensitivity mutation leaked into the base config")
	}
}

func TestWriteSensitivityCSVStableAndFinite(t *testing.T) {
	rows := []domain.SensitivityResult{
		{
			ParameterPath:     "z.path",
			ParameterValue:    2,
			Scenario:          "z",
			BehaviorCase:      "case",
			HorizonMonths:     12,
			Metric:            v04config.BreakEvenCashEndUnrestricted,
			MedianValue:       10,
			MedianPairedDelta: 1,
			Classification:    ClassificationDevelopmentDominant,
		},
		{
			ParameterPath:     "a.path",
			ParameterValue:    1,
			Scenario:          "a",
			BehaviorCase:      "case",
			HorizonMonths:     6,
			Metric:            v04config.BreakEvenRevenueCAGR,
			MedianValue:       0.1,
			MedianPairedDelta: -0.2,
			Classification:    ClassificationDevelopmentTradeoff,
		},
	}
	var output bytes.Buffer
	if err := WriteSensitivityCSV(&output, rows); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(output.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(records[0], sensitivityCSVHeader) {
		t.Fatalf("header = %v", records[0])
	}
	if records[1][0] != "a.path" || records[2][0] != "z.path" {
		t.Fatalf("rows are not stable: %v", records)
	}

	rows[0].MedianValue = math.NaN()
	if err := WriteSensitivityCSV(&bytes.Buffer{}, rows); err == nil || !strings.Contains(err.Error(), "finite") {
		t.Fatalf("non-finite sensitivity CSV value error = %v", err)
	}
}
