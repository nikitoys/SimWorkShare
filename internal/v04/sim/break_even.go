package sim

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

const (
	noEffectBehaviorCase         = "no_effect"
	defaultBreakEvenGridSteps    = 40
	maxBreakEvenBisectionSteps   = 48
	noBreakEvenInTestedRangeFlag = "no_break_even_in_tested_range"
)

var breakEvenCSVHeader = []string{
	"scenario", "behavior_case", "reference_scenario", "horizon_months", "metric",
	"productivity_uplift", "flags",
}

type breakEvenEvaluation struct {
	medianPairedMetric          float64
	bankruptcyProbability       float64
	liquidityDeficitProbability float64
}

// FindBreakEven searches the configured uplift range for the smallest direct
// productivity uplift that satisfies the section 17 paired-metric and risk
// gates. The reference is always evaluated with no_effect. Candidate and
// reference execute as separate Runs with common random numbers forced on, so
// they receive the same stable environment path without sharing mutable state.
func FindBreakEven(
	cfg v04config.Config,
	scenario string,
	behavior string,
	reference string,
	horizon int,
	steps int,
) (domain.BreakEvenResult, error) {
	base := cfg.DeepCopy()
	metric := normalizeAnalysisMetric(base.Analysis.BreakEvenMetric)
	base.Analysis.BreakEvenMetric = metric
	if err := v04config.Validate(base); err != nil {
		return domain.BreakEvenResult{}, fmt.Errorf("validate break-even config: %w", err)
	}
	if reference == "" {
		reference = v04config.SystemTraditionalCompany
	}
	result := domain.BreakEvenResult{
		Scenario:          scenario,
		BehaviorCase:      behavior,
		ReferenceScenario: reference,
		HorizonMonths:     horizon,
		Metric:            metric,
	}

	if scenario == "" {
		return domain.BreakEvenResult{}, fmt.Errorf("break-even scenario must not be empty")
	}
	if behavior == "" {
		return domain.BreakEvenResult{}, fmt.Errorf("break-even behavior case must not be empty")
	}
	if behavior == noEffectBehaviorCase {
		return domain.BreakEvenResult{}, fmt.Errorf("break-even candidate behavior must not be %q", noEffectBehaviorCase)
	}
	if scenario == reference {
		return domain.BreakEvenResult{}, fmt.Errorf("break-even scenario %q must differ from reference scenario", scenario)
	}
	if horizon <= 0 || horizon > base.Simulation.Months {
		return domain.BreakEvenResult{}, fmt.Errorf("break-even horizon %d must be between 1 and simulation.months (%d)", horizon, base.Simulation.Months)
	}
	if steps <= 0 {
		steps = defaultBreakEvenGridSteps
	}
	if err := validateBreakEvenCases(base, scenario, behavior, reference); err != nil {
		return domain.BreakEvenResult{}, err
	}

	// Computing only the requested horizon materially reduces analysis output;
	// the underlying monthly path still covers simulation.months identically.
	base.Simulation.HorizonsMonths = []int{horizon}
	base.Simulation.CommonRandomNumbers = true
	if err := v04config.Validate(base); err != nil {
		return domain.BreakEvenResult{}, fmt.Errorf("validate prepared break-even config: %w", err)
	}

	referenceResult, err := Run(base.DeepCopy(), RunOptions{
		ScenarioNames:       []string{reference},
		BehaviorCaseNames:   []string{noEffectBehaviorCase},
		StoreMonthlyResults: false,
		StoreRunSummaries:   true,
	})
	if err != nil {
		return domain.BreakEvenResult{}, fmt.Errorf("run break-even reference %s/%s: %w", reference, noEffectBehaviorCase, err)
	}
	referenceSummary, err := findTerminalScenarioSummary(referenceResult.TerminalSummaries, reference, noEffectBehaviorCase, horizon)
	if err != nil {
		return domain.BreakEvenResult{}, fmt.Errorf("summarize break-even reference: %w", err)
	}

	tolerance := base.Analysis.ClassificationTolerance
	evaluate := func(uplift float64) (breakEvenEvaluation, error) {
		candidateConfig := base.DeepCopy()
		candidateBehavior := candidateConfig.BehaviorCases[behavior]
		candidateBehavior.BaseProductivityUpliftDirect = uplift
		candidateConfig.BehaviorCases[behavior] = candidateBehavior
		if err := v04config.Validate(candidateConfig); err != nil {
			return breakEvenEvaluation{}, fmt.Errorf("validate candidate uplift %.17g: %w", uplift, err)
		}
		candidateResult, err := Run(candidateConfig, RunOptions{
			ScenarioNames:       []string{scenario},
			BehaviorCaseNames:   []string{behavior},
			StoreMonthlyResults: false,
			StoreRunSummaries:   true,
		})
		if err != nil {
			return breakEvenEvaluation{}, fmt.Errorf("run candidate uplift %.17g: %w", uplift, err)
		}
		candidateSummary, err := findTerminalScenarioSummary(candidateResult.TerminalSummaries, scenario, behavior, horizon)
		if err != nil {
			return breakEvenEvaluation{}, fmt.Errorf("summarize candidate uplift %.17g: %w", uplift, err)
		}
		medianDelta, err := medianPairedTerminalMetric(
			candidateResult.RunTerminalSummaries,
			referenceResult.RunTerminalSummaries,
			scenario,
			behavior,
			reference,
			noEffectBehaviorCase,
			horizon,
			metric,
		)
		if err != nil {
			return breakEvenEvaluation{}, fmt.Errorf("pair candidate uplift %.17g: %w", uplift, err)
		}
		evaluation := breakEvenEvaluation{
			medianPairedMetric:          medianDelta,
			bankruptcyProbability:       candidateSummary.BankruptcyProbability,
			liquidityDeficitProbability: candidateSummary.LiquidityDeficitProbability,
		}
		if !breakEvenEvaluationFinite(evaluation) {
			return breakEvenEvaluation{}, fmt.Errorf("candidate uplift %.17g produced non-finite break-even values", uplift)
		}
		return evaluation, nil
	}
	qualifies := func(evaluation breakEvenEvaluation) bool {
		return qualifiesBreakEven(evaluation, referenceSummary, tolerance)
	}

	lower := base.Analysis.BreakEvenUpliftRange[0]
	upper := base.Analysis.BreakEvenUpliftRange[1]
	previousUplift := lower
	lowerEvaluation, err := evaluate(previousUplift)
	if err != nil {
		return domain.BreakEvenResult{}, err
	}
	if qualifies(lowerEvaluation) {
		value := normalizeZero(previousUplift)
		result.ProductivityUplift = &value
		return result, nil
	}

	gridWidth := (upper - lower) / float64(steps)
	for gridIndex := 1; gridIndex <= steps; gridIndex++ {
		currentUplift := lower + float64(gridIndex)*gridWidth
		if gridIndex == steps {
			currentUplift = upper
		}
		currentEvaluation, err := evaluate(currentUplift)
		if err != nil {
			return domain.BreakEvenResult{}, err
		}
		if qualifies(currentEvaluation) {
			value, err := bisectFirstQualifyingUplift(previousUplift, currentUplift, evaluate, qualifies)
			if err != nil {
				return domain.BreakEvenResult{}, err
			}
			value = normalizeZero(value)
			result.ProductivityUplift = &value
			return result, nil
		}
		previousUplift = currentUplift
	}

	result.Flags = []string{noBreakEvenInTestedRangeFlag}
	return result, nil
}

// WriteBreakEvenCSV writes stable break-even columns. A nil uplift is encoded
// as an empty CSV cell, matching the JSON null result semantically.
func WriteBreakEvenCSV(writer io.Writer, results []domain.BreakEvenResult) error {
	if writer == nil {
		return fmt.Errorf("break-even CSV writer is nil")
	}
	ordered := append([]domain.BreakEvenResult(nil), results...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, right := ordered[i], ordered[j]
		if left.Scenario != right.Scenario {
			return left.Scenario < right.Scenario
		}
		if left.BehaviorCase != right.BehaviorCase {
			return left.BehaviorCase < right.BehaviorCase
		}
		if left.ReferenceScenario != right.ReferenceScenario {
			return left.ReferenceScenario < right.ReferenceScenario
		}
		if left.HorizonMonths != right.HorizonMonths {
			return left.HorizonMonths < right.HorizonMonths
		}
		return left.Metric < right.Metric
	})

	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(breakEvenCSVHeader); err != nil {
		return fmt.Errorf("write break-even CSV header: %w", err)
	}
	for index, result := range ordered {
		uplift := ""
		if result.ProductivityUplift != nil {
			formatted, err := formatCSVFloat("productivity_uplift", *result.ProductivityUplift)
			if err != nil {
				return fmt.Errorf("break-even CSV row %d: %w", index, err)
			}
			uplift = formatted
		}
		flags := append([]string(nil), result.Flags...)
		sort.Strings(flags)
		row := []string{
			result.Scenario,
			result.BehaviorCase,
			result.ReferenceScenario,
			strconv.Itoa(result.HorizonMonths),
			result.Metric,
			uplift,
			strings.Join(flags, ";"),
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write break-even CSV row %d: %w", index, err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush break-even CSV: %w", err)
	}
	return nil
}

func validateBreakEvenCases(cfg v04config.Config, scenario, behavior, reference string) error {
	candidate, ok := cfg.ScenarioByName(scenario)
	if !ok {
		return fmt.Errorf("unknown break-even scenario %q", scenario)
	}
	if _, ok := cfg.BehaviorCases[behavior]; !ok {
		return fmt.Errorf("unknown break-even behavior case %q", behavior)
	}
	if !containsString(candidate.BehaviorCaseRefs, behavior) {
		return fmt.Errorf("break-even behavior case %q is not enabled for scenario %q", behavior, scenario)
	}
	referenceScenario, ok := cfg.ScenarioByName(reference)
	if !ok {
		return fmt.Errorf("unknown break-even reference scenario %q", reference)
	}
	if _, ok := cfg.BehaviorCases[noEffectBehaviorCase]; !ok {
		return fmt.Errorf("break-even reference behavior case %q is not configured", noEffectBehaviorCase)
	}
	if !containsString(referenceScenario.BehaviorCaseRefs, noEffectBehaviorCase) {
		return fmt.Errorf("break-even reference scenario %q does not enable behavior case %q", reference, noEffectBehaviorCase)
	}
	return nil
}

func findTerminalScenarioSummary(
	summaries []domain.ScenarioSummary,
	scenario string,
	behavior string,
	horizon int,
) (domain.ScenarioSummary, error) {
	var found *domain.ScenarioSummary
	for index := range summaries {
		summary := &summaries[index]
		if summary.Scenario != scenario || summary.BehaviorCase != behavior || summary.HorizonMonths != horizon {
			continue
		}
		if found != nil {
			return domain.ScenarioSummary{}, fmt.Errorf("duplicate summary for %s/%s at horizon %d", scenario, behavior, horizon)
		}
		copySummary := *summary
		found = &copySummary
	}
	if found == nil {
		return domain.ScenarioSummary{}, fmt.Errorf("missing summary for %s/%s at horizon %d", scenario, behavior, horizon)
	}
	if !domain.Finite(found.BankruptcyProbability) || !domain.Finite(found.LiquidityDeficitProbability) {
		return domain.ScenarioSummary{}, fmt.Errorf("risk probabilities for %s/%s at horizon %d must be finite", scenario, behavior, horizon)
	}
	return *found, nil
}

func qualifiesBreakEven(candidate breakEvenEvaluation, reference domain.ScenarioSummary, tolerance float64) bool {
	if !breakEvenEvaluationFinite(candidate) ||
		!domain.Finite(reference.BankruptcyProbability) ||
		!domain.Finite(reference.LiquidityDeficitProbability) ||
		!domain.Finite(tolerance) || tolerance < 0 {
		return false
	}
	return candidate.medianPairedMetric >= 0 &&
		candidate.bankruptcyProbability <= reference.BankruptcyProbability+tolerance &&
		candidate.liquidityDeficitProbability <= reference.LiquidityDeficitProbability+tolerance
}

func breakEvenEvaluationFinite(evaluation breakEvenEvaluation) bool {
	return domain.Finite(evaluation.medianPairedMetric) &&
		domain.Finite(evaluation.bankruptcyProbability) &&
		domain.Finite(evaluation.liquidityDeficitProbability)
}

func bisectFirstQualifyingUplift(
	lower float64,
	upper float64,
	evaluate func(float64) (breakEvenEvaluation, error),
	qualifies func(breakEvenEvaluation) bool,
) (float64, error) {
	resolution := math.Max(1e-9, math.Abs(upper-lower)*1e-8)
	for iteration := 0; iteration < maxBreakEvenBisectionSteps && upper-lower > resolution; iteration++ {
		midpoint := lower + (upper-lower)/2
		evaluation, err := evaluate(midpoint)
		if err != nil {
			return 0, err
		}
		if qualifies(evaluation) {
			upper = midpoint
		} else {
			lower = midpoint
		}
	}
	if !domain.Finite(upper) {
		return 0, fmt.Errorf("break-even bisection produced a non-finite uplift")
	}
	return upper, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeZero(value float64) float64 {
	if value == 0 || math.Abs(value) < 1e-15 {
		return 0
	}
	return value
}
