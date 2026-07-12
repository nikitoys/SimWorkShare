package sim

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

var sensitivityCSVHeader = []string{
	"parameter_path", "parameter_value", "scenario", "behavior_case", "horizon_months",
	"metric", "median_value", "median_paired_delta", "classification",
}

type scenarioBehaviorKey struct {
	scenario string
	behavior string
}

// RunSensitivity evaluates every configured sensitivity path/value pair on an
// independent deep copy of cfg. Reference scenarios are included in each Run
// even when the caller filters them out, while returned rows contain only the
// scenario/behavior cases selected by options.
func RunSensitivity(cfg v04config.Config, options RunOptions) ([]domain.SensitivityResult, error) {
	base := cfg.DeepCopy()
	metric := normalizeAnalysisMetric(base.Analysis.BreakEvenMetric)
	base.Analysis.BreakEvenMetric = metric
	if err := v04config.Validate(base); err != nil {
		return nil, fmt.Errorf("validate sensitivity config: %w", err)
	}

	selectedCases, err := ExpandScenarioCases(base, options)
	if err != nil {
		return nil, fmt.Errorf("select sensitivity cases: %w", err)
	}
	selected := make(map[scenarioBehaviorKey]struct{}, len(selectedCases))
	runScenarios := make(map[string]struct{}, len(selectedCases)+len(base.Analysis.PairedReferenceScenarios))
	for _, scenarioCase := range selectedCases {
		selected[scenarioBehaviorKey{scenarioCase.Scenario.Name, scenarioCase.BehaviorName}] = struct{}{}
		runScenarios[scenarioCase.Scenario.Name] = struct{}{}
	}
	for _, reference := range base.Analysis.PairedReferenceScenarios {
		runScenarios[reference] = struct{}{}
	}

	runOptions := RunOptions{
		ScenarioNames:       sortedStringSet(runScenarios),
		BehaviorCaseNames:   append([]string(nil), options.BehaviorCaseNames...),
		StoreMonthlyResults: false,
		StoreRunSummaries:   true,
	}

	var results []domain.SensitivityResult
	for parameterIndex, parameter := range base.Analysis.SensitivityParameters {
		for valueIndex, value := range parameter.Values {
			mutated := base.DeepCopy()
			if err := v04config.SetNumericPath(&mutated, parameter.Path, value); err != nil {
				return nil, fmt.Errorf("analysis.sensitivity_parameters[%d].values[%d] (%s=%g): %w",
					parameterIndex, valueIndex, parameter.Path, value, err)
			}
			if err := v04config.Validate(mutated); err != nil {
				return nil, fmt.Errorf("validate analysis.sensitivity_parameters[%d].values[%d] (%s=%g): %w",
					parameterIndex, valueIndex, parameter.Path, value, err)
			}

			runResult, err := Run(mutated, runOptions)
			if err != nil {
				return nil, fmt.Errorf("run analysis.sensitivity_parameters[%d].values[%d] (%s=%g): %w",
					parameterIndex, valueIndex, parameter.Path, value, err)
			}
			for _, summary := range runResult.TerminalSummaries {
				key := scenarioBehaviorKey{summary.Scenario, summary.BehaviorCase}
				if _, ok := selected[key]; !ok {
					continue
				}
				medianValue, err := medianTerminalMetric(
					runResult.RunTerminalSummaries,
					summary.Scenario,
					summary.BehaviorCase,
					summary.HorizonMonths,
					metric,
				)
				if err != nil {
					return nil, fmt.Errorf("sensitivity metric for %s/%s at horizon %d: %w",
						summary.Scenario, summary.BehaviorCase, summary.HorizonMonths, err)
				}
				medianDelta, err := preferredSensitivityDelta(
					runResult.RunTerminalSummaries,
					base.Analysis.PairedReferenceScenarios,
					summary.Scenario,
					summary.BehaviorCase,
					summary.HorizonMonths,
					metric,
				)
				if err != nil {
					return nil, fmt.Errorf("sensitivity paired delta for %s/%s at horizon %d: %w",
						summary.Scenario, summary.BehaviorCase, summary.HorizonMonths, err)
				}
				if !domain.Finite(medianValue) || !domain.Finite(medianDelta) {
					return nil, fmt.Errorf("sensitivity result for %s/%s at horizon %d is not finite",
						summary.Scenario, summary.BehaviorCase, summary.HorizonMonths)
				}
				results = append(results, domain.SensitivityResult{
					ParameterPath:     parameter.Path,
					ParameterValue:    value,
					Scenario:          summary.Scenario,
					BehaviorCase:      summary.BehaviorCase,
					HorizonMonths:     summary.HorizonMonths,
					Metric:            metric,
					MedianValue:       medianValue,
					MedianPairedDelta: medianDelta,
					Classification:    summary.Classification,
				})
			}
		}
	}
	sortSensitivityResults(results)
	return results, nil
}

// WriteSensitivityCSV writes stable sensitivity columns and row ordering.
func WriteSensitivityCSV(writer io.Writer, results []domain.SensitivityResult) error {
	if writer == nil {
		return fmt.Errorf("sensitivity CSV writer is nil")
	}
	ordered := append([]domain.SensitivityResult(nil), results...)
	sortSensitivityResults(ordered)

	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(sensitivityCSVHeader); err != nil {
		return fmt.Errorf("write sensitivity CSV header: %w", err)
	}
	for index, result := range ordered {
		parameterValue, err := formatCSVFloat("parameter_value", result.ParameterValue)
		if err != nil {
			return fmt.Errorf("sensitivity CSV row %d: %w", index, err)
		}
		medianValue, err := formatCSVFloat("median_value", result.MedianValue)
		if err != nil {
			return fmt.Errorf("sensitivity CSV row %d: %w", index, err)
		}
		medianDelta, err := formatCSVFloat("median_paired_delta", result.MedianPairedDelta)
		if err != nil {
			return fmt.Errorf("sensitivity CSV row %d: %w", index, err)
		}
		row := []string{
			result.ParameterPath,
			parameterValue,
			result.Scenario,
			result.BehaviorCase,
			strconv.Itoa(result.HorizonMonths),
			result.Metric,
			medianValue,
			medianDelta,
			result.Classification,
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("write sensitivity CSV row %d: %w", index, err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush sensitivity CSV: %w", err)
	}
	return nil
}

func normalizeAnalysisMetric(metric string) string {
	if metric == "" {
		return v04config.BreakEvenSustainableDevelopment
	}
	return metric
}

func terminalMetricValue(summary domain.RunTerminalSummary, metric string) (float64, error) {
	var value float64
	switch normalizeAnalysisMetric(metric) {
	case v04config.BreakEvenSustainableDevelopment:
		value = summary.SustainableDevelopmentValueProxy
	case v04config.BreakEvenCashEndUnrestricted:
		value = summary.CashEndUnrestricted
	case v04config.BreakEvenCapacityGrowth:
		value = summary.ProductiveCapacityGrowthRate
	case v04config.BreakEvenRevenueCAGR:
		value = summary.RevenueCAGR
	default:
		return 0, fmt.Errorf("unsupported analysis metric %q", metric)
	}
	if !domain.Finite(value) {
		return 0, fmt.Errorf("analysis metric %q must be finite", metric)
	}
	return value, nil
}

func medianTerminalMetric(
	runs []domain.RunTerminalSummary,
	scenario string,
	behavior string,
	horizon int,
	metric string,
) (float64, error) {
	values := make([]float64, 0)
	for _, summary := range runs {
		if summary.Scenario != scenario || summary.BehaviorCase != behavior || summary.HorizonMonths != horizon {
			continue
		}
		value, err := terminalMetricValue(summary, metric)
		if err != nil {
			return 0, err
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return 0, fmt.Errorf("no terminal runs for %s/%s at horizon %d", scenario, behavior, horizon)
	}
	median := Percentile(values, 0.5)
	if !domain.Finite(median) {
		return 0, fmt.Errorf("median analysis metric %q must be finite", metric)
	}
	return median, nil
}

func preferredSensitivityDelta(
	runs []domain.RunTerminalSummary,
	references []string,
	scenario string,
	behavior string,
	horizon int,
	metric string,
) (float64, error) {
	orderedReferences, isFirstReference := eligibleReferences(references, scenario)
	if isFirstReference {
		return 0, nil
	}
	var errorsSeen []string
	for _, reference := range orderedReferences {
		median, err := medianPairedTerminalMetric(
			runs, runs,
			scenario, behavior,
			reference, behavior,
			horizon, metric,
		)
		if err == nil {
			return median, nil
		}
		errorsSeen = append(errorsSeen, fmt.Sprintf("%s: %v", reference, err))
	}
	if len(errorsSeen) == 0 {
		return 0, fmt.Errorf("no eligible paired reference scenario")
	}
	return 0, fmt.Errorf("no complete paired reference: %v", errorsSeen)
}

func eligibleReferences(references []string, scenario string) ([]string, bool) {
	references = uniqueStrings(references)
	position := -1
	for index, reference := range references {
		if reference == scenario {
			position = index
			break
		}
	}
	if position == 0 {
		return nil, true
	}
	eligible := make([]string, 0, len(references))
	for index, reference := range references {
		if reference == scenario {
			continue
		}
		if position >= 0 && index > position {
			continue
		}
		eligible = append(eligible, reference)
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		return referenceRank(eligible[i]) < referenceRank(eligible[j])
	})
	return eligible, false
}

type terminalPairKey struct {
	run    int
	market string
}

func medianPairedTerminalMetric(
	candidateRuns []domain.RunTerminalSummary,
	referenceRuns []domain.RunTerminalSummary,
	candidateScenario string,
	candidateBehavior string,
	referenceScenario string,
	referenceBehavior string,
	horizon int,
	metric string,
) (float64, error) {
	references := make(map[terminalPairKey]float64)
	for _, summary := range referenceRuns {
		if summary.Scenario != referenceScenario || summary.BehaviorCase != referenceBehavior || summary.HorizonMonths != horizon {
			continue
		}
		key := terminalPairKey{summary.Run, marketCase(summary.MarketCase)}
		if _, exists := references[key]; exists {
			return 0, fmt.Errorf("duplicate reference terminal run %d market %s", key.run, key.market)
		}
		value, err := terminalMetricValue(summary, metric)
		if err != nil {
			return 0, err
		}
		references[key] = value
	}
	if len(references) == 0 {
		return 0, fmt.Errorf("no reference terminal runs for %s/%s at horizon %d", referenceScenario, referenceBehavior, horizon)
	}

	deltas := make([]float64, 0, len(references))
	candidateCount := 0
	for _, summary := range candidateRuns {
		if summary.Scenario != candidateScenario || summary.BehaviorCase != candidateBehavior || summary.HorizonMonths != horizon {
			continue
		}
		candidateCount++
		key := terminalPairKey{summary.Run, marketCase(summary.MarketCase)}
		referenceValue, ok := references[key]
		if !ok {
			return 0, fmt.Errorf("missing reference pair for run %d market %s", key.run, key.market)
		}
		candidateValue, err := terminalMetricValue(summary, metric)
		if err != nil {
			return 0, err
		}
		delta := candidateValue - referenceValue
		if !domain.Finite(delta) {
			return 0, fmt.Errorf("paired delta for run %d must be finite", summary.Run)
		}
		deltas = append(deltas, delta)
	}
	if candidateCount == 0 {
		return 0, fmt.Errorf("no candidate terminal runs for %s/%s at horizon %d", candidateScenario, candidateBehavior, horizon)
	}
	if len(deltas) != len(references) {
		return 0, fmt.Errorf("incomplete pairing: candidate runs=%d reference runs=%d", len(deltas), len(references))
	}
	median := Percentile(deltas, 0.5)
	if !domain.Finite(median) {
		return 0, fmt.Errorf("median paired metric %q must be finite", metric)
	}
	return median, nil
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortSensitivityResults(results []domain.SensitivityResult) {
	sort.SliceStable(results, func(i, j int) bool {
		left, right := results[i], results[j]
		if left.ParameterPath != right.ParameterPath {
			return left.ParameterPath < right.ParameterPath
		}
		if left.ParameterValue != right.ParameterValue {
			return left.ParameterValue < right.ParameterValue
		}
		if left.Scenario != right.Scenario {
			return left.Scenario < right.Scenario
		}
		if left.BehaviorCase != right.BehaviorCase {
			return left.BehaviorCase < right.BehaviorCase
		}
		if left.HorizonMonths != right.HorizonMonths {
			return left.HorizonMonths < right.HorizonMonths
		}
		return left.Metric < right.Metric
	})
}
