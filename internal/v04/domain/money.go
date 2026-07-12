package domain

import "math"

const (
	DefaultAbsoluteTolerance = 1e-6
	DefaultRelativeTolerance = 1e-9
)

// AlmostEqual is the centralized equality used by the v0.4 ledgers. The
// configured epsilon can make the absolute tolerance stricter or looser, but
// relative protection is retained for large nominal balances.
func AlmostEqual(a, b, epsilon float64) bool {
	abs := math.Max(DefaultAbsoluteTolerance, epsilon)
	scale := math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
	return math.Abs(a-b) <= math.Max(abs, DefaultRelativeTolerance*scale)
}

// LedgerAlmostEqual scales the relative tolerance by every bridge operand.
// This preserves small future queue entries after very large due entries are
// removed from an aggregate float64 balance.
func LedgerAlmostEqual(closing, opening, added, removed, epsilon float64) bool {
	abs := math.Max(DefaultAbsoluteTolerance, epsilon)
	expected := opening + added - removed
	scale := 1.0
	for _, value := range []float64{closing, opening, added, removed} {
		scale = math.Max(scale, math.Abs(value))
	}
	return math.Abs(closing-expected) <= math.Max(abs, DefaultRelativeTolerance*scale)
}

func Clamp(value, minimum, maximum float64) float64 {
	return math.Min(maximum, math.Max(minimum, value))
}

func Finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
