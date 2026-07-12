package sim

import (
	"fmt"
	"math"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

type AllocationOutput struct {
	Actual          domain.AllocationAmounts
	CashBudgetUsed  float64
	RemainingBudget float64
}

func RawAllocations(
	scenario v04config.OrganizationalScenario,
	monthlyPositiveBase float64,
	distributionPeriodBase float64,
	eligibleHeadcount float64,
) domain.AllocationAmounts {
	monthlyPositiveBase = math.Max(0, monthlyPositiveBase)
	distributionPeriodBase = math.Max(0, distributionPeriodBase)
	employeeDistribution := scenario.EmployeeCashDistributionRate * distributionPeriodBase
	if scenario.MaxDistributionPerEmployeePeriod != nil {
		employeeDistribution = math.Min(employeeDistribution, *scenario.MaxDistributionPerEmployeePeriod*math.Max(0, eligibleHeadcount))
	}
	memberAllocation := scenario.MemberCapitalAllocationRate * monthlyPositiveBase
	if eligibleHeadcount <= 0 {
		employeeDistribution = 0
		memberAllocation = 0
	}
	return domain.AllocationAmounts{
		EmployeeCashDistribution: employeeDistribution,
		MemberCapitalAllocation:  memberAllocation,
		Reinvestment:             scenario.ReinvestmentRate * monthlyPositiveBase,
		OrganizationalReserve:    scenario.OrganizationalReserveRate * monthlyPositiveBase,
		ExternalDistribution:     scenario.ExternalDistributionRate * monthlyPositiveBase,
	}
}

// AdvanceDistributionPeriod accumulates the already hurdled monthly positive
// result bases and releases them only when the configured distribution period
// closes. Other allocations remain monthly; this accumulator is used only for
// the employee cash-distribution base.
func AdvanceDistributionPeriod(
	accumulator float64,
	monthsInPeriod int,
	positiveResultBase float64,
	periodMonths int,
) (distributionBase float64, accumulatorClose float64, monthsClose int) {
	accumulatorClose = math.Max(0, accumulator) + math.Max(0, positiveResultBase)
	monthsClose = monthsInPeriod + 1
	if monthsClose >= periodMonths {
		return accumulatorClose, 0, 0
	}
	return 0, accumulatorClose, monthsClose
}

// AllocatePositiveResult applies the exact cash-safe priority algorithm from
// section 8.11. Member capital and reserve allocations are not immediate cash
// outflows, but the specification assigns them cash multiplier 1, so they
// consume allocation budget here.
func AllocatePositiveResult(
	raw domain.AllocationAmounts,
	priority []string,
	budget float64,
	distributionPayrollTaxRate float64,
) (AllocationOutput, error) {
	remaining := math.Max(0, budget)
	actual := domain.AllocationAmounts{}
	for _, item := range priority {
		var requested float64
		multiplier := 1.0
		switch item {
		case v04config.AllocationEmployeeDistribution:
			requested = raw.EmployeeCashDistribution
			multiplier += distributionPayrollTaxRate
		case v04config.AllocationMemberCapital:
			requested = raw.MemberCapitalAllocation
		case v04config.AllocationReinvestment:
			requested = raw.Reinvestment
		case v04config.AllocationOrganizationalReserve:
			requested = raw.OrganizationalReserve
		case v04config.AllocationExternalDistribution:
			requested = raw.ExternalDistribution
		default:
			return AllocationOutput{}, fmt.Errorf("unknown allocation priority item %q", item)
		}
		value := math.Min(math.Max(0, requested), remaining/multiplier)
		remaining -= value * multiplier
		if remaining < 0 && remaining > -1e-9 {
			remaining = 0
		}
		switch item {
		case v04config.AllocationEmployeeDistribution:
			actual.EmployeeCashDistribution = value
		case v04config.AllocationMemberCapital:
			actual.MemberCapitalAllocation = value
		case v04config.AllocationReinvestment:
			actual.Reinvestment = value
		case v04config.AllocationOrganizationalReserve:
			actual.OrganizationalReserve = value
		case v04config.AllocationExternalDistribution:
			actual.ExternalDistribution = value
		}
	}
	return AllocationOutput{
		Actual:          actual,
		CashBudgetUsed:  math.Max(0, budget) - remaining,
		RemainingBudget: remaining,
	}, nil
}

type TaxOutput struct {
	DeductibleEmployeeDistribution   float64
	TaxableProfit                    float64
	ProfitTaxAccrual                 float64
	NetProfitAfterTaxAndDistribution float64
}

func CalculateTaxAndNetProfit(
	profitBeforeTaxBeforeDistribution float64,
	employeeDistributionGross float64,
	distributionPayrollTax float64,
	company v04config.CompanyEconomics,
	financing v04config.Financing,
) TaxOutput {
	deductible := math.Max(0, employeeDistributionGross) * financing.DistributionTaxDeductibleShare
	taxable := math.Max(0, profitBeforeTaxBeforeDistribution-deductible)
	tax := taxable * company.ProfitTaxRate
	// v0.4 decision: this accounting metric subtracts the employee gross
	// distribution, its employer payroll tax and profit tax. Reinvestment,
	// reserves and member-capital allocations are uses of result, not expenses.
	net := profitBeforeTaxBeforeDistribution - tax - math.Max(0, employeeDistributionGross) - math.Max(0, distributionPayrollTax)
	return TaxOutput{
		DeductibleEmployeeDistribution:   deductible,
		TaxableProfit:                    taxable,
		ProfitTaxAccrual:                 tax,
		NetProfitAfterTaxAndDistribution: net,
	}
}
