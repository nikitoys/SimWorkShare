package sim

import (
	"math"

	v04config "simworkshare/internal/v04/config"
)

type CostInput struct {
	Company             v04config.CompanyEconomics
	Workforce           v04config.Workforce
	PaidEmployees       float64
	Revenue             float64
	Hires               float64
	VoluntaryLeavers    float64
	Layoffs             float64
	GovernanceCashCost  float64
	ShockCost           float64
	CostInflationFactor float64
}

type CostOutput struct {
	SalaryCost                     float64
	SalaryPayrollTax               float64
	FixedCosts                     float64
	VariableCosts                  float64
	HiringCost                     float64
	ExitCost                       float64
	LayoffCost                     float64
	TurnoverAndWorkforceCost       float64
	OperatingCostsBeforeAllocation float64
}

func CalculateCosts(input CostInput) CostOutput {
	salary := math.Max(0, input.PaidEmployees) * input.Company.BaseSalaryPerEmployeeMonthly * input.CostInflationFactor
	salaryTax := salary * input.Company.SalaryPayrollTaxRate
	fixed := input.Company.FixedCostsMonthly * input.CostInflationFactor
	variable := math.Max(0, input.Revenue) * input.Company.VariableCostRate
	hiring := math.Max(0, input.Hires) * (input.Workforce.RecruitingCostPerHire + input.Workforce.OnboardingCostPerHire + input.Workforce.ManagerTimeCostPerHire)
	exit := math.Max(0, input.VoluntaryLeavers) * (input.Workforce.ExitAdminCostPerLeaver + input.Workforce.LostProductivityCostPerLeaver)
	layoff := math.Max(0, input.Layoffs) * input.Workforce.SeveranceCostPerLayoff
	workforce := hiring + exit + layoff
	total := salary + salaryTax + fixed + variable + workforce + input.GovernanceCashCost + input.ShockCost
	return CostOutput{
		SalaryCost:                     salary,
		SalaryPayrollTax:               salaryTax,
		FixedCosts:                     fixed,
		VariableCosts:                  variable,
		HiringCost:                     hiring,
		ExitCost:                       exit,
		LayoffCost:                     layoff,
		TurnoverAndWorkforceCost:       workforce,
		OperatingCostsBeforeAllocation: total,
	}
}

// RequiredCashReserve uses current nominal payroll/fixed-cost references and
// scheduled debt service. It is computed from beginning headcount so layoffs
// cannot manufacture allocation cash in the same month.
func RequiredCashReserve(
	company v04config.CompanyEconomics,
	financing v04config.Financing,
	headcountBegin, debtBalanceBegin, costInflationFactor float64,
) float64 {
	salaryReference := math.Max(0, headcountBegin) * company.BaseSalaryPerEmployeeMonthly * costInflationFactor
	salaryReference *= 1 + company.SalaryPayrollTaxRate
	fixedReference := company.FixedCostsMonthly * costInflationFactor
	interestReference := math.Max(0, debtBalanceBegin) * financing.DebtInterestRateAnnual / 12
	debtReference := math.Min(math.Max(0, debtBalanceBegin), financing.ScheduledPrincipalPaymentMonthly) + interestReference
	return company.RequiredCashReserveMonths * (salaryReference + fixedReference + debtReference)
}
