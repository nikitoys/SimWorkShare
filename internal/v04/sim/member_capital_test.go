package sim

import "testing"

func TestMemberCapitalRedemptionAccrualIsProRataAndCapped(t *testing.T) {
	got := MemberCapitalRedemptionAccrual(1_000, 10, 2, 1, 0.5, 1e-9)
	if got != 150 {
		t.Fatalf("redemption = %g, want 150", got)
	}
	if got := MemberCapitalRedemptionAccrual(1_000, 0, 1, 0, 1, 1e-9); got != 0 {
		t.Fatalf("zero-headcount redemption = %g, want 0", got)
	}
}
