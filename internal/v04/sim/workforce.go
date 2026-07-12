package sim

import (
	"fmt"
	"math"
	"sort"

	"simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

type WorkforceInput struct {
	Run                                   int
	Month                                 int
	RandomKey                             string
	HeadcountBegin                        float64
	RampCohortsBegin                      []float64
	TurnoverRateMonthly                   float64
	UnrestrictedCashBegin                 float64
	SalaryCostsReferenceMonthly           float64
	FixedCostsReferenceMonthly            float64
	ScheduledDebtServiceReferenceMonthly  float64
	MarketDemandForecast                  float64
	ProductiveCapacityRevenueMonthlyBegin float64
	ProductivityMultiplier                float64
	GovernanceAdminEquivalentEmployees    float64
}

type WorkforceResult struct {
	RequiredCashReserveBegin  float64
	BeginCashStressRatio      float64
	CashStressLayoffPressure  float64
	DistressLayoffRate        float64
	VoluntaryLeavers          float64
	DistressLayoffs           float64
	HeadcountAfterExits       float64
	TargetRevenueForStaffing  float64
	DesiredHeadcount          float64
	GrossHiringNeed           float64
	HireCapacity              float64
	HiringCashAvailable       float64
	AffordableHires           float64
	Hires                     float64
	HeadcountEnd              float64
	PaidEmployees             float64
	EffectiveRampingEmployees float64
	EffectiveNewHires         float64
	EffectiveEmployees        float64
	Ramp                      domain.RampState
}

// CalculateWorkforce implements sections 8.7 through 8.9 for leavers,
// layoffs, hiring, ramp cohorts and effective employees. MarketDemandForecast
// is supplied by the engine; the current implementation convention is current
// market demand rather than a hidden forecasting model.
func CalculateWorkforce(cfg config.Config, scenario config.OrganizationalScenario, input WorkforceInput) (WorkforceResult, error) {
	if err := validateWorkforceInput(cfg, input); err != nil {
		return WorkforceResult{}, err
	}

	randomKey := ""
	if !cfg.Simulation.CommonRandomNumbers {
		randomKey = input.RandomKey
		if randomKey == "" {
			randomKey = scenario.Name
		}
	}
	rngFor := func(mechanism string) *stableRNG {
		return newStableRNG(deriveStableSeed(cfg.Simulation.RandomSeed, input.Run, input.Month, streamLabor+"."+mechanism, randomKey))
	}

	voluntaryLeavers, err := calculateVoluntaryLeavers(
		input.HeadcountBegin,
		input.TurnoverRateMonthly,
		cfg.Workforce.TurnoverRandomMode,
		cfg.Simulation.HeadcountMode,
		rngFor("voluntary_turnover"),
	)
	if err != nil {
		return WorkforceResult{}, err
	}

	requiredReserve := cfg.CompanyEconomics.RequiredCashReserveMonths * (input.SalaryCostsReferenceMonthly + input.FixedCostsReferenceMonthly + input.ScheduledDebtServiceReferenceMonthly)
	beginCashStressRatio := input.UnrestrictedCashBegin / math.Max(cfg.Simulation.Epsilon, requiredReserve)
	cashStressPressure := math.Max(0, cfg.Workforce.LayoffTriggerCashRatio-beginCashStressRatio) /
		math.Max(cfg.Simulation.Epsilon, cfg.Workforce.LayoffTriggerCashRatio)
	distressLayoffRate := cfg.Workforce.MaxLayoffsPerMonthRate * cashStressPressure * (1 - scenario.EmploymentStabilizationPreference)
	headcountAfterVoluntary := input.HeadcountBegin - voluntaryLeavers
	rawLayoffs := math.Min(headcountAfterVoluntary, headcountAfterVoluntary*distressLayoffRate)
	layoffs := quantizeCount(rawLayoffs, cfg.Simulation.HeadcountMode, rngFor("layoffs"))
	layoffs = domain.Clamp(layoffs, 0, headcountAfterVoluntary)
	headcountAfterExits := headcountAfterVoluntary - layoffs

	targetRevenue := math.Min(
		input.MarketDemandForecast*cfg.Workforce.TargetStaffingBuffer,
		input.ProductiveCapacityRevenueMonthlyBegin,
	)
	desiredHeadcount := targetRevenue / (cfg.CompanyEconomics.BaseRevenuePerEffectiveEmployeeMonthly * math.Max(0.10, input.ProductivityMultiplier))
	grossHiringNeed := math.Max(0, desiredHeadcount-headcountAfterExits)
	hireCapacity := cfg.Workforce.MaxHiresPerMonthRate * math.Max(1, input.HeadcountBegin)
	hiringCashAvailable := math.Max(0, input.UnrestrictedCashBegin-requiredReserve) * cfg.Workforce.MaxCashShareForHiring
	hiringCostPerHire := cfg.Workforce.RecruitingCostPerHire + cfg.Workforce.OnboardingCostPerHire + cfg.Workforce.ManagerTimeCostPerHire
	affordableHires := hiringCashAvailable / math.Max(cfg.Simulation.Epsilon, hiringCostPerHire)
	rawHires := math.Min(grossHiringNeed, math.Min(hireCapacity, affordableHires))
	hires := quantizeCount(rawHires, cfg.Simulation.HeadcountMode, rngFor("hires"))
	if cfg.Simulation.HeadcountMode != config.HeadcountFractional {
		// Hiring need, monthly capacity and affordability are hard upper bounds.
		// Rounding an expected count upward must never authorize an otherwise
		// unaffordable whole hire.
		hires = math.Min(hires, math.Floor(rawHires+cfg.Simulation.Epsilon))
	}
	hires = math.Max(0, hires)
	headcountEnd := headcountAfterExits + hires

	fullBegin := input.HeadcountBegin - sum(input.RampCohortsBegin)
	buckets := make([]float64, 1+len(input.RampCohortsBegin))
	buckets[0] = fullBegin
	copy(buckets[1:], input.RampCohortsBegin)
	buckets, err = applyProRataExits(buckets, voluntaryLeavers, cfg.Simulation.HeadcountMode)
	if err != nil {
		return WorkforceResult{}, fmt.Errorf("allocate voluntary leavers: %w", err)
	}
	buckets, err = applyProRataExits(buckets, layoffs, cfg.Simulation.HeadcountMode)
	if err != nil {
		return WorkforceResult{}, fmt.Errorf("allocate layoffs: %w", err)
	}
	fullAfter := buckets[0]
	rampAfter := append([]float64(nil), buckets[1:]...)

	rampClose := make([]float64, cfg.Workforce.RampDurationMonths)
	fullClose := fullAfter
	effectiveRamping := 0.0
	effectiveNewHires := hires
	if cfg.Workforce.RampDurationMonths == 0 {
		fullClose += hires
	} else {
		for index, cohort := range rampAfter {
			effectiveRamping += cohort * cfg.Workforce.RampProductivityMultipliers[index]
		}
		effectiveNewHires = hires * cfg.Workforce.RampProductivityMultipliers[0]

		// Current hires receive multiplier[0] exactly once in this month. The
		// production cohorts are then advanced, avoiding a second multiplier[0]
		// application next month.
		productionCohorts := append([]float64(nil), rampAfter...)
		productionCohorts[0] += hires
		for index := 0; index < len(productionCohorts)-1; index++ {
			rampClose[index+1] = productionCohorts[index]
		}
		fullClose += productionCohorts[len(productionCohorts)-1]
	}
	effectiveEmployees := math.Max(0,
		fullAfter+effectiveRamping+effectiveNewHires-input.GovernanceAdminEquivalentEmployees,
	)
	paidEmployees := input.HeadcountBegin -
		cfg.Workforce.LeaverPaidFractionOfMonth*voluntaryLeavers -
		cfg.Workforce.LeaverPaidFractionOfMonth*layoffs +
		cfg.Workforce.NewHirePaidFractionOfMonth*hires

	result := WorkforceResult{
		RequiredCashReserveBegin:  requiredReserve,
		BeginCashStressRatio:      beginCashStressRatio,
		CashStressLayoffPressure:  cashStressPressure,
		DistressLayoffRate:        distressLayoffRate,
		VoluntaryLeavers:          voluntaryLeavers,
		DistressLayoffs:           layoffs,
		HeadcountAfterExits:       headcountAfterExits,
		TargetRevenueForStaffing:  targetRevenue,
		DesiredHeadcount:          desiredHeadcount,
		GrossHiringNeed:           grossHiringNeed,
		HireCapacity:              hireCapacity,
		HiringCashAvailable:       hiringCashAvailable,
		AffordableHires:           affordableHires,
		Hires:                     hires,
		HeadcountEnd:              headcountEnd,
		PaidEmployees:             paidEmployees,
		EffectiveRampingEmployees: effectiveRamping,
		EffectiveNewHires:         effectiveNewHires,
		EffectiveEmployees:        effectiveEmployees,
		Ramp: domain.RampState{
			Begin:      append([]float64(nil), input.RampCohortsBegin...),
			AfterExits: rampAfter,
			Close:      rampClose,
			FullBegin:  fullBegin,
			FullAfter:  fullAfter,
			FullClose:  fullClose,
		},
	}
	if err := validateWorkforceResult(cfg, result); err != nil {
		return WorkforceResult{}, err
	}
	return result, nil
}

func validateWorkforceInput(cfg config.Config, input WorkforceInput) error {
	if input.Run < 0 {
		return fmt.Errorf("run must be >= 0")
	}
	if input.Month < 1 {
		return fmt.Errorf("month must be >= 1")
	}
	if cfg.Simulation.Epsilon <= 0 || !domain.Finite(cfg.Simulation.Epsilon) {
		return fmt.Errorf("simulation.epsilon must be finite and > 0")
	}
	if cfg.Workforce.RampDurationMonths < 0 || len(cfg.Workforce.RampProductivityMultipliers) != cfg.Workforce.RampDurationMonths {
		return fmt.Errorf("workforce.ramp_productivity_multipliers length must equal ramp_duration_months")
	}
	if len(input.RampCohortsBegin) != cfg.Workforce.RampDurationMonths {
		return fmt.Errorf("ramp_cohorts_begin length must equal workforce.ramp_duration_months")
	}
	values := map[string]float64{
		"headcount_begin":                           input.HeadcountBegin,
		"turnover_rate_monthly":                     input.TurnoverRateMonthly,
		"unrestricted_cash_begin":                   input.UnrestrictedCashBegin,
		"salary_costs_reference_monthly":            input.SalaryCostsReferenceMonthly,
		"fixed_costs_reference_monthly":             input.FixedCostsReferenceMonthly,
		"scheduled_debt_service_reference_monthly":  input.ScheduledDebtServiceReferenceMonthly,
		"market_demand_forecast":                    input.MarketDemandForecast,
		"productive_capacity_revenue_monthly_begin": input.ProductiveCapacityRevenueMonthlyBegin,
		"productivity_multiplier":                   input.ProductivityMultiplier,
		"governance_admin_equivalent_employees":     input.GovernanceAdminEquivalentEmployees,
	}
	for name, value := range values {
		if !domain.Finite(value) {
			return fmt.Errorf("%s must be finite", name)
		}
	}
	if input.HeadcountBegin < 0 {
		return fmt.Errorf("headcount_begin must be >= 0")
	}
	if input.TurnoverRateMonthly < 0 || input.TurnoverRateMonthly > 1 {
		return fmt.Errorf("turnover_rate_monthly must be within [0,1]")
	}
	for index, cohort := range input.RampCohortsBegin {
		if !domain.Finite(cohort) || cohort < 0 {
			return fmt.Errorf("ramp_cohorts_begin[%d] must be finite and >= 0", index)
		}
	}
	if sum(input.RampCohortsBegin) > input.HeadcountBegin+cfg.Simulation.Epsilon {
		return fmt.Errorf("sum of ramp_cohorts_begin must not exceed headcount_begin")
	}
	if cfg.CompanyEconomics.BaseRevenuePerEffectiveEmployeeMonthly <= 0 {
		return fmt.Errorf("company_economics.base_revenue_per_effective_employee_monthly must be > 0")
	}
	if cfg.Simulation.HeadcountMode != config.HeadcountFractional {
		if !isIntegral(input.HeadcountBegin, cfg.Simulation.Epsilon) {
			return fmt.Errorf("headcount_begin must be integral in %s mode", cfg.Simulation.HeadcountMode)
		}
		for index, cohort := range input.RampCohortsBegin {
			if !isIntegral(cohort, cfg.Simulation.Epsilon) {
				return fmt.Errorf("ramp_cohorts_begin[%d] must be integral in %s mode", index, cfg.Simulation.HeadcountMode)
			}
		}
	}
	if cfg.Workforce.TurnoverRandomMode == config.TurnoverBinomial && cfg.Simulation.HeadcountMode == config.HeadcountFractional {
		return fmt.Errorf("binomial turnover requires an integer headcount mode")
	}
	return nil
}

func calculateVoluntaryLeavers(headcount, monthlyRate float64, turnoverMode, headcountMode string, rng *stableRNG) (float64, error) {
	switch turnoverMode {
	case config.TurnoverDeterministic:
		return domain.Clamp(quantizeCount(headcount*monthlyRate, headcountMode, rng), 0, headcount), nil
	case config.TurnoverBinomial:
		if !isIntegral(headcount, 1e-9) {
			return 0, fmt.Errorf("binomial turnover requires integral headcount")
		}
		leavers := 0
		for employee := 0; employee < int(math.Round(headcount)); employee++ {
			if rng.float64() < monthlyRate {
				leavers++
			}
		}
		return float64(leavers), nil
	default:
		return 0, fmt.Errorf("unsupported turnover_random_mode %q", turnoverMode)
	}
}

func quantizeCount(value float64, mode string, rng *stableRNG) float64 {
	value = math.Max(0, value)
	switch mode {
	case config.HeadcountIntegerExpected:
		return math.Floor(value + 0.5)
	case config.HeadcountIntegerRandom:
		floor := math.Floor(value)
		if rng.float64() < value-floor {
			return floor + 1
		}
		return floor
	default:
		return value
	}
}

func applyProRataExits(buckets []float64, exits float64, mode string) ([]float64, error) {
	result := append([]float64(nil), buckets...)
	total := sum(result)
	if exits <= 0 || total <= 0 {
		return result, nil
	}
	exits = math.Min(exits, total)
	if mode == config.HeadcountFractional {
		survival := (total - exits) / total
		for index := range result {
			result[index] *= survival
		}
		return result, nil
	}
	if !isIntegral(exits, 1e-9) {
		return nil, fmt.Errorf("exit count must be integral in %s mode", mode)
	}

	type remainder struct {
		index int
		value float64
	}
	allocated := make([]float64, len(result))
	remainders := make([]remainder, len(result))
	allocatedTotal := 0
	for index, bucket := range result {
		quota := exits * bucket / total
		allocated[index] = math.Floor(quota)
		allocatedTotal += int(allocated[index])
		remainders[index] = remainder{index: index, value: quota - allocated[index]}
	}
	sort.SliceStable(remainders, func(i, j int) bool {
		return remainders[i].value > remainders[j].value
	})
	remaining := int(math.Round(exits)) - allocatedTotal
	for _, item := range remainders {
		if remaining == 0 {
			break
		}
		if allocated[item.index] < result[item.index] {
			allocated[item.index]++
			remaining--
		}
	}
	if remaining != 0 {
		return nil, fmt.Errorf("could not apportion %g exits", exits)
	}
	for index := range result {
		result[index] -= allocated[index]
	}
	return result, nil
}

func validateWorkforceResult(cfg config.Config, result WorkforceResult) error {
	values := map[string]float64{
		"required_cash_reserve_begin": result.RequiredCashReserveBegin,
		"begin_cash_stress_ratio":     result.BeginCashStressRatio,
		"voluntary_leavers":           result.VoluntaryLeavers,
		"distress_layoffs":            result.DistressLayoffs,
		"hires":                       result.Hires,
		"headcount_end":               result.HeadcountEnd,
		"paid_employees":              result.PaidEmployees,
		"effective_employees":         result.EffectiveEmployees,
	}
	for name, value := range values {
		if !domain.Finite(value) {
			return fmt.Errorf("%s is not finite", name)
		}
	}
	if !domain.AlmostEqual(
		result.HeadcountEnd,
		result.Ramp.FullClose+sum(result.Ramp.Close),
		cfg.Simulation.Epsilon,
	) {
		return fmt.Errorf("closing ramp cohorts do not reconcile to headcount_end")
	}
	return nil
}

func sum(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}

func isIntegral(value, epsilon float64) bool {
	return math.Abs(value-math.Round(value)) <= epsilon
}
