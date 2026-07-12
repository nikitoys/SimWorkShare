package sim

import "math"

// MemberCapitalRedemptionAccrual assigns a pro-rata share of beginning member
// capital to all current-month exits. It is capped by the account balance and
// removed from member-capital liability when accrued; cash leaves only when
// the redemption queue becomes due.
func MemberCapitalRedemptionAccrual(
	memberCapitalBegin float64,
	headcountBegin float64,
	voluntaryLeavers float64,
	layoffs float64,
	redemptionFraction float64,
	epsilon float64,
) float64 {
	if memberCapitalBegin <= 0 || headcountBegin <= math.Max(0, epsilon) {
		return 0
	}
	exitShare := math.Min(1, math.Max(0, voluntaryLeavers+layoffs)/headcountBegin)
	return math.Min(memberCapitalBegin, memberCapitalBegin*exitShare*redemptionFraction)
}
