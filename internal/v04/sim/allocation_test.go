package sim

import (
	"testing"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func TestAllocationPriorityUnderCashShortage(t *testing.T) {
	raw := domain.AllocationAmounts{
		OrganizationalReserve:    40,
		Reinvestment:             40,
		EmployeeCashDistribution: 40,
	}
	got, err := AllocatePositiveResult(raw, []string{
		v04config.AllocationOrganizationalReserve,
		v04config.AllocationReinvestment,
		v04config.AllocationEmployeeDistribution,
	}, 100, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if got.Actual.OrganizationalReserve != 40 || got.Actual.Reinvestment != 40 || got.Actual.EmployeeCashDistribution != 16 {
		t.Fatalf("actual = %+v, want reserve=40 reinvest=40 distribution=16", got.Actual)
	}
	if got.CashBudgetUsed != 100 || got.RemainingBudget != 0 {
		t.Fatalf("budget = %+v", got)
	}
}

func TestRawAllocationCapAndNoEmployees(t *testing.T) {
	capPerEmployee := 10.0
	scenario := v04config.OrganizationalScenario{
		EmployeeCashDistributionRate:     0.5,
		MemberCapitalAllocationRate:      0.2,
		MaxDistributionPerEmployeePeriod: &capPerEmployee,
	}
	got := RawAllocations(scenario, 100, 100, 3)
	if got.EmployeeCashDistribution != 30 || got.MemberCapitalAllocation != 20 {
		t.Fatalf("capped raw = %+v", got)
	}
	zero := RawAllocations(scenario, 100, 100, 0)
	if zero.EmployeeCashDistribution != 0 || zero.MemberCapitalAllocation != 0 {
		t.Fatalf("zero-headcount raw = %+v", zero)
	}
}

func TestDistributionPeriodAccumulatesMonthlyPositiveResultBases(t *testing.T) {
	base, accumulator, months := AdvanceDistributionPeriod(0, 0, 0, 2)
	if base != 0 || accumulator != 0 || months != 1 {
		t.Fatalf("first period state = base %g accumulator %g months %d", base, accumulator, months)
	}
	base, accumulator, months = AdvanceDistributionPeriod(accumulator, months, 100, 2)
	if base != 100 || accumulator != 0 || months != 0 {
		t.Fatalf("closed period state = base %g accumulator %g months %d, want 100/0/0", base, accumulator, months)
	}

	scenario := v04config.OrganizationalScenario{
		EmployeeCashDistributionRate: 0.1,
		MemberCapitalAllocationRate:  0.2,
	}
	beforeDistributionDate := RawAllocations(scenario, 50, 0, 10)
	if beforeDistributionDate.EmployeeCashDistribution != 0 || beforeDistributionDate.MemberCapitalAllocation != 10 {
		t.Fatalf("period allocations = %+v, want employee=0 member=10", beforeDistributionDate)
	}
}
