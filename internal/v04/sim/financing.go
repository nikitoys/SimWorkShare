package sim

import (
	"math"

	v04config "simworkshare/internal/v04/config"
)

func EffectiveCreditLine(
	financing v04config.Financing,
	scenario v04config.OrganizationalScenario,
	creditMarketFactor float64,
) float64 {
	return math.Max(0, financing.BaseCreditLine*scenario.ExternalCapitalAccessMultiplier*creditMarketFactor)
}

type DebtService struct {
	InterestExpense float64
	PrincipalDue    float64
}

func CalculateDebtService(financing v04config.Financing, debtBalanceBegin float64) DebtService {
	debt := math.Max(0, debtBalanceBegin)
	return DebtService{
		InterestExpense: debt * financing.DebtInterestRateAnnual / 12,
		PrincipalDue:    math.Min(debt, financing.ScheduledPrincipalPaymentMonthly),
	}
}

type LiquidityFunding struct {
	RestrictedReserveReleased float64
	CreditDraw                float64
	CreditHeadroomBeforeDraw  float64
	GeneralCashAvailable      float64
}

// FundMandatoryGap releases only the configured share of the organizational
// reserve and then draws the effective credit line. Reserve release is a cash
// reclassification; credit is a cash inflow and debt increase.
func FundMandatoryGap(
	unrestrictedCashBeforePayments float64,
	generalMandatoryScheduled float64,
	restrictedReserve float64,
	reserveReleaseRate float64,
	creditLineLimit float64,
	debtBalanceBegin float64,
) LiquidityFunding {
	baseAvailable := math.Max(0, unrestrictedCashBeforePayments)
	gap := math.Max(0, generalMandatoryScheduled-baseAvailable)
	release := math.Min(math.Max(0, restrictedReserve)*reserveReleaseRate, gap)
	gap -= release
	headroom := math.Max(0, creditLineLimit-math.Max(0, debtBalanceBegin))
	draw := math.Min(gap, headroom)
	return LiquidityFunding{
		RestrictedReserveReleased: release,
		CreditDraw:                draw,
		CreditHeadroomBeforeDraw:  headroom,
		GeneralCashAvailable:      baseAvailable + release + draw,
	}
}

type GrowthCapital struct {
	Draw     float64
	DebtDraw float64
	Grant    float64
}

func CalculateExternalGrowthCapital(
	rawReinvestment float64,
	cashReinvestment float64,
	financing v04config.Financing,
	scenario v04config.OrganizationalScenario,
	creditMarketFactor float64,
	creditLineLimit float64,
	debtAfterLiquidityAndPrincipal float64,
) GrowthCapital {
	shortfall := math.Max(0, rawReinvestment-cashReinvestment)
	monthlyLimit := financing.ExternalGrowthCapitalLimitMonthly *
		scenario.ExternalCapitalAccessMultiplier * creditMarketFactor
	draw := math.Min(shortfall, math.Max(0, monthlyLimit))
	result := GrowthCapital{}
	if financing.ExternalCapitalType == v04config.ExternalCapitalDebt {
		headroom := math.Max(0, creditLineLimit-math.Max(0, debtAfterLiquidityAndPrincipal))
		draw = math.Min(draw, headroom)
		result.DebtDraw = draw
	} else {
		result.Grant = draw
	}
	result.Draw = draw
	return result
}
