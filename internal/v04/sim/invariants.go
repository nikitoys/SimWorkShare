package sim

import (
	"fmt"
	"math"
	"reflect"

	v04config "simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func ValidateMonthInvariants(cfg v04config.Config, result domain.MonthlyResult) error {
	epsilon := cfg.Simulation.Epsilon
	if err := finiteValue(reflect.ValueOf(result), "monthly_result"); err != nil {
		return err
	}
	if !domain.AlmostEqual(
		result.HeadcountEnd,
		result.HeadcountBegin-result.VoluntaryLeavers-result.Layoffs+result.Hires,
		epsilon,
	) {
		return fmt.Errorf("headcount identity failed")
	}
	if result.HeadcountEnd < -epsilon {
		return fmt.Errorf("headcount is negative")
	}
	if !result.ActiveCompanyFlag {
		if !result.Risks.Bankruptcy {
			return fmt.Errorf("inactive month must preserve bankruptcy flag")
		}
		if !domain.AlmostEqual(result.CashTotalClose, result.CashTotalBegin, epsilon) ||
			!domain.AlmostEqual(result.ProductiveCapacityClose, result.ProductiveCapacityBegin, epsilon) ||
			!domain.AlmostEqual(result.HeadcountEnd, result.HeadcountBegin, epsilon) {
			return fmt.Errorf("inactive month changed absorbing state")
		}
		if !domain.AlmostEqual(result.RestrictedCashClose, result.RestrictedDistributionClose+result.RestrictedReserveClose, epsilon) ||
			!domain.AlmostEqual(result.UnrestrictedCashClose, result.CashTotalClose-result.RestrictedCashClose, epsilon) {
			return fmt.Errorf("inactive month cash classification identity failed")
		}
		if !domain.AlmostEqual(result.GeneralMandatoryArrearsClose, result.GeneralMandatoryArrearsBegin, epsilon) {
			return fmt.Errorf("inactive month changed general mandatory arrears")
		}
		return nil
	}
	for name, limit := range map[string]float64{
		"market demand":       result.MarketDemand,
		"labor capacity":      result.LaborRevenueCapacity,
		"productive capacity": result.ProductiveCapacityRevenueMonthly,
	} {
		if result.Revenue-limit > math.Max(epsilon, domain.DefaultAbsoluteTolerance) {
			return fmt.Errorf("revenue exceeds %s", name)
		}
	}

	expectedCash := result.CashTotalBegin + result.CashCollectedCurrent + result.CashCollectedFromAR +
		result.CreditDrawForLiquidity + result.ExternalGrowthCapitalDraw -
		result.MandatoryCashPayments - result.ReinvestmentCashPaid -
		result.ExternalDistributionPaid - result.ExternalGrowthCapitalSpent
	if !domain.AlmostEqual(result.CashTotalClose, expectedCash, epsilon) {
		return fmt.Errorf("cash identity failed: close=%g expected=%g", result.CashTotalClose, expectedCash)
	}
	if !domain.AlmostEqual(
		result.RestrictedCashClose,
		result.RestrictedDistributionClose+result.RestrictedReserveClose,
		epsilon,
	) {
		return fmt.Errorf("restricted cash identity failed")
	}
	if !domain.AlmostEqual(result.UnrestrictedCashClose, result.CashTotalClose-result.RestrictedCashClose, epsilon) {
		return fmt.Errorf("unrestricted cash identity failed")
	}
	expectedGeneralArrears := result.GeneralMandatoryArrearsBegin +
		result.GeneralMandatoryCurrentScheduled - result.GeneralMandatoryPayments
	if !domain.AlmostEqual(result.GeneralMandatoryArrearsClose, expectedGeneralArrears, epsilon) {
		return fmt.Errorf("general mandatory arrears identity failed: close=%g expected=%g", result.GeneralMandatoryArrearsClose, expectedGeneralArrears)
	}
	expectedDebt := result.DebtBalanceBegin - result.PrincipalPaid +
		result.CreditDrawForLiquidity
	if cfg.Financing.ExternalCapitalType == v04config.ExternalCapitalDebt {
		expectedDebt += result.ExternalGrowthCapitalDraw
	}
	if !domain.AlmostEqual(result.DebtBalanceClose, expectedDebt, epsilon) {
		return fmt.Errorf("debt identity failed")
	}
	expectedCapacity := result.ProductiveCapacityBegin*(1-cfg.CompanyEconomics.CapacityDepreciationRateMonthly) + result.CapacityAdditionsDue
	if !domain.AlmostEqual(result.ProductiveCapacityClose, expectedCapacity, epsilon) {
		return fmt.Errorf("capacity identity failed: close=%g expected=%g", result.ProductiveCapacityClose, expectedCapacity)
	}
	expectedMemberCapital := result.MemberCapitalBegin + result.MemberCapitalAllocation - result.MemberCapitalRedemptionAccrual
	if !domain.AlmostEqual(result.MemberCapitalClose, expectedMemberCapital, epsilon) {
		return fmt.Errorf("member capital identity failed")
	}

	allocationCash := result.ActualAllocations.EmployeeCashDistribution*(1+cfg.Financing.DistributionPayrollTaxRate) +
		result.ActualAllocations.MemberCapitalAllocation + result.ActualAllocations.Reinvestment +
		result.ActualAllocations.OrganizationalReserve + result.ActualAllocations.ExternalDistribution
	if allocationCash-result.CashSafeAllocationBudget > math.Max(epsilon, domain.DefaultAbsoluteTolerance) {
		return fmt.Errorf("cash-safe allocation identity failed")
	}
	if !domain.LedgerAlmostEqual(
		result.ClosingAccountsReceivable,
		result.OpeningAccountsReceivable,
		result.NewAccountsReceivable,
		result.CashCollectedFromAR,
		epsilon,
	) {
		return fmt.Errorf("accounts receivable queue identity failed")
	}
	if !domain.LedgerAlmostEqual(result.TaxPayableClose, result.TaxPayableBegin, result.ProfitTaxAccrual, result.TaxesPaid, epsilon) {
		return fmt.Errorf("tax payable queue identity failed")
	}
	distributionPaid := result.EmployeeCashDistributionPaid + result.EmployeeDistributionPayrollTaxPaid
	if !domain.LedgerAlmostEqual(
		result.EmployeeDistributionPayableClose,
		result.RestrictedDistributionBegin,
		result.RestrictedDistributionCashNew,
		distributionPaid,
		epsilon,
	) {
		return fmt.Errorf("employee distribution queue identity failed")
	}
	if !domain.AlmostEqual(result.RestrictedDistributionClose, result.EmployeeDistributionPayableClose, epsilon) {
		return fmt.Errorf("restricted distribution cash and payable disagree")
	}
	if !domain.LedgerAlmostEqual(
		result.RestrictedReserveClose,
		result.RestrictedReserveBegin,
		result.OrganizationalReserveAllocation,
		result.RestrictedReserveReleased,
		epsilon,
	) {
		return fmt.Errorf("organizational reserve identity failed")
	}
	if !domain.LedgerAlmostEqual(
		result.MemberCapitalRedemptionPayableClose,
		result.MemberCapitalRedemptionPayableBegin,
		result.MemberCapitalRedemptionAccrual,
		result.MemberCapitalRedemptionPaid,
		epsilon,
	) {
		return fmt.Errorf("member capital redemption queue identity failed")
	}
	if result.BehaviorCase == "no_effect" {
		if math.Abs(result.MotivationUpliftRaw) > epsilon || math.Abs(result.BehavioralTurnoverDeltaAnnual) > epsilon {
			return fmt.Errorf("no-effect behavior created an automatic motivation or retention effect")
		}
	}
	return nil
}

func finiteValue(value reflect.Value, path string) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		return finiteValue(value.Elem(), path)
	}
	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		number := value.Float()
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("%s is not finite", path)
		}
	case reflect.Struct:
		typeOf := value.Type()
		for index := 0; index < value.NumField(); index++ {
			if err := finiteValue(value.Field(index), path+"."+typeOf.Field(index).Name); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := finiteValue(value.Index(index), fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}
