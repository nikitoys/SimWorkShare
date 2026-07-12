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

func TestFindBreakEvenFindsMinimumWithoutMutatingConfig(t *testing.T) {
	cfg := breakEvenTestConfig(t, 2)
	cfg.Analysis.BreakEvenMetric = v04config.BreakEvenCashEndUnrestricted
	cfg.Analysis.BreakEvenUpliftRange = []float64{-0.05, 0.05}
	original := cfg.DeepCopy()

	result, err := FindBreakEven(
		cfg,
		v04config.SystemWorkerCooperative,
		"moderate_positive",
		v04config.SystemTraditionalCompany,
		2,
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, original) {
		t.Fatal("FindBreakEven mutated the base config")
	}
	if result.ProductivityUplift == nil {
		t.Fatalf("break-even was not found: %+v", result)
	}
	if math.Abs(*result.ProductivityUplift) > 1e-7 {
		t.Fatalf("break-even uplift = %.12g, want 0", *result.ProductivityUplift)
	}
	if result.Metric != v04config.BreakEvenCashEndUnrestricted || len(result.Flags) != 0 {
		t.Fatalf("unexpected break-even result: %+v", result)
	}
}

func TestFindBreakEvenReturnsNullAndFlagOutsideRange(t *testing.T) {
	cfg := breakEvenTestConfig(t, 1)
	cfg.Analysis.BreakEvenMetric = v04config.BreakEvenCashEndUnrestricted
	cfg.Analysis.BreakEvenUpliftRange = []float64{-0.10, -0.05}

	result, err := FindBreakEven(
		cfg,
		v04config.SystemWorkerCooperative,
		"moderate_positive",
		v04config.SystemTraditionalCompany,
		1,
		4,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProductivityUplift != nil {
		t.Fatalf("unexpected break-even uplift: %g", *result.ProductivityUplift)
	}
	if !reflect.DeepEqual(result.Flags, []string{noBreakEvenInTestedRangeFlag}) {
		t.Fatalf("flags = %v", result.Flags)
	}
}

func TestFindBreakEvenReproducibleWithSeparateCommonRandomRuns(t *testing.T) {
	cfg := breakEvenTestConfig(t, 3)
	cfg.Simulation.Mode = v04config.ModeMonteCarlo
	cfg.Simulation.Runs = 3
	cfg.Simulation.CommonRandomNumbers = false // FindBreakEven must override this on its copy.
	cfg.CompanyEconomics.InitialMarketDemandMonthly = 100_000_000
	cfg.CompanyEconomics.InitialProductiveCapacityRevenueMonthly = 100_000_000
	cfg.Market.MarketVolatilityMonthly = 0.2
	cfg.Market.ShockProbabilityMonthly = 0.5
	cfg.Analysis.BreakEvenMetric = v04config.BreakEvenCashEndUnrestricted
	cfg.Analysis.BreakEvenUpliftRange = []float64{-0.02, 0.02}

	first, err := FindBreakEven(
		cfg,
		v04config.SystemWorkerCooperative,
		"moderate_positive",
		v04config.SystemTraditionalCompany,
		3,
		4,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := FindBreakEven(
		cfg,
		v04config.SystemWorkerCooperative,
		"moderate_positive",
		v04config.SystemTraditionalCompany,
		3,
		4,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("break-even changed for the same seed: first=%+v second=%+v", first, second)
	}
	if first.ProductivityUplift == nil || math.Abs(*first.ProductivityUplift) > 1e-7 {
		t.Fatalf("common random paths did not preserve the zero threshold: %+v", first)
	}
}

func TestFindBreakEvenRejectsNoEffectCandidate(t *testing.T) {
	cfg := breakEvenTestConfig(t, 1)
	_, err := FindBreakEven(
		cfg,
		v04config.SystemWorkerCooperative,
		noEffectBehaviorCase,
		v04config.SystemTraditionalCompany,
		1,
		4,
	)
	if err == nil || !strings.Contains(err.Error(), "must not be") {
		t.Fatalf("no_effect candidate error = %v", err)
	}
}

func TestQualifiesBreakEvenUsesExactMetricAndRiskGates(t *testing.T) {
	reference := domain.ScenarioSummary{
		BankruptcyProbability:       0.20,
		LiquidityDeficitProbability: 0.30,
	}
	atBoundary := breakEvenEvaluation{
		medianPairedMetric:          0,
		bankruptcyProbability:       0.21,
		liquidityDeficitProbability: 0.31,
	}
	if !qualifiesBreakEven(atBoundary, reference, 0.01) {
		t.Fatal("exact section 17 boundary did not qualify")
	}
	negativeMetric := atBoundary
	negativeMetric.medianPairedMetric = -math.SmallestNonzeroFloat64
	if qualifiesBreakEven(negativeMetric, reference, 0.01) {
		t.Fatal("negative paired median qualified")
	}
	worseRisk := atBoundary
	worseRisk.bankruptcyProbability = 0.210001
	if qualifiesBreakEven(worseRisk, reference, 0.01) {
		t.Fatal("candidate outside bankruptcy tolerance qualified")
	}
	nonFinite := atBoundary
	nonFinite.medianPairedMetric = math.NaN()
	if qualifiesBreakEven(nonFinite, reference, 0.01) {
		t.Fatal("non-finite candidate qualified")
	}
}

func TestWriteBreakEvenCSVStableNullAndFinite(t *testing.T) {
	uplift := 0.03
	rows := []domain.BreakEvenResult{
		{
			Scenario:           "z",
			BehaviorCase:       "case",
			ReferenceScenario:  "reference",
			HorizonMonths:      12,
			Metric:             v04config.BreakEvenRevenueCAGR,
			ProductivityUplift: &uplift,
			Flags:              []string{"z", "a"},
		},
		{
			Scenario:          "a",
			BehaviorCase:      "case",
			ReferenceScenario: "reference",
			HorizonMonths:     6,
			Metric:            v04config.BreakEvenSustainableDevelopment,
			Flags:             []string{noBreakEvenInTestedRangeFlag},
		},
	}
	var output bytes.Buffer
	if err := WriteBreakEvenCSV(&output, rows); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(output.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(records[0], breakEvenCSVHeader) {
		t.Fatalf("header = %v", records[0])
	}
	if records[1][0] != "a" || records[1][5] != "" || records[2][0] != "z" || records[2][6] != "a;z" {
		t.Fatalf("unexpected stable/null CSV rows: %v", records)
	}

	nonFinite := math.Inf(1)
	rows[0].ProductivityUplift = &nonFinite
	if err := WriteBreakEvenCSV(&bytes.Buffer{}, rows); err == nil || !strings.Contains(err.Error(), "finite") {
		t.Fatalf("non-finite break-even CSV value error = %v", err)
	}
}

func breakEvenTestConfig(t *testing.T, months int) v04config.Config {
	t.Helper()
	cfg := deterministicConfig(t, months)
	cfg.Market.MarketGrowthMonthly = 0
	cfg.Market.ShockProbabilityMonthly = 0
	cfg.Workforce.BaseTurnoverRateAnnual = 0
	cfg.Workforce.MinTurnoverRateAnnual = 0
	cfg.Workforce.MaxTurnoverRateAnnual = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	cfg.CompanyEconomics.CapacityDepreciationRateMonthly = 0

	reference, ok := cfg.ScenarioByName(v04config.SystemTraditionalCompany)
	if !ok {
		t.Fatal("traditional scenario is missing")
	}
	candidate, ok := cfg.ScenarioByName(v04config.SystemWorkerCooperative)
	if !ok {
		t.Fatal("worker cooperative scenario is missing")
	}
	matchingScenario := *reference
	matchingScenario.Name = v04config.SystemWorkerCooperative
	matchingScenario.SystemType = v04config.SystemWorkerCooperative
	matchingScenario.BehaviorCaseRefs = []string{noEffectBehaviorCase, "moderate_positive"}
	*candidate = matchingScenario

	neutral := cfg.BehaviorCases[noEffectBehaviorCase]
	cfg.BehaviorCases["moderate_positive"] = neutral
	return cfg
}
