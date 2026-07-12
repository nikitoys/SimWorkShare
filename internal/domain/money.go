package domain

import "math"

// Money is stored in nominal units of simulation.currency. The model does not
// round intermediate values; rounding is a presentation concern.
type Money float64

const (
	// RelativeTolerance is used for deterministic numeric comparisons.
	RelativeTolerance = 1e-9
	// MoneyAbsoluteTolerance prevents near-zero floating-point noise from
	// becoming a false accounting-integrity failure.
	MoneyAbsoluteTolerance Money = 1e-6
)

// MoneyAlmostEqual compares monetary values using the model-wide tolerances.
func MoneyAlmostEqual(a, b Money) bool {
	diff := math.Abs(float64(a - b))
	scale := math.Max(1, math.Max(math.Abs(float64(a)), math.Abs(float64(b))))
	return diff <= math.Max(float64(MoneyAbsoluteTolerance), RelativeTolerance*scale)
}

// MoneyLess reports a smaller monetary value at a policy/risk boundary. Unlike
// accounting equality, a boundary comparison uses only the absolute tolerance
// so that its blind zone does not grow with the size of the balance.
func MoneyLess(a, b Money) bool {
	return a < b && float64(b-a) > float64(MoneyAbsoluteTolerance)
}
