package domain

import "testing"

func TestMoneyAlmostEqualUsesCentralTolerance(t *testing.T) {
	if !MoneyAlmostEqual(10_000_000, 10_000_000.005) {
		t.Fatal("MoneyAlmostEqual rejected a value inside relative tolerance")
	}
	if MoneyAlmostEqual(10_000_000, 10_000_000.02) {
		t.Fatal("MoneyAlmostEqual accepted a value outside relative tolerance")
	}
}

func TestMoneyLessTreatsBoundaryNoiseAsEqual(t *testing.T) {
	if MoneyLess(99.9999999, 100) {
		t.Fatal("MoneyLess treated boundary noise as a material difference")
	}
	if !MoneyLess(99.99, 100) {
		t.Fatal("MoneyLess rejected a material difference")
	}
	if !MoneyLess(1_000_000_000_000-1, 1_000_000_000_000) {
		t.Fatal("MoneyLess created a balance-scaled blind zone")
	}
}
