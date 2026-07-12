package sim

import (
	"reflect"
	"testing"

	"simworkshare/internal/v04/config"
)

func TestCalculateWorkforceExactLayoffHiringAndRampFormulas(t *testing.T) {
	cfg := workforceTestConfig()
	result, err := CalculateWorkforce(cfg, config.OrganizationalScenario{Name: "scenario", EmploymentStabilizationPreference: 0}, WorkforceInput{
		Run:                                   1,
		Month:                                 1,
		HeadcountBegin:                        10,
		RampCohortsBegin:                      []float64{3, 2, 1},
		TurnoverRateMonthly:                   0.2,
		UnrestrictedCashBegin:                 200,
		SalaryCostsReferenceMonthly:           100,
		MarketDemandForecast:                  900,
		ProductiveCapacityRevenueMonthlyBegin: 900,
		ProductivityMultiplier:                1,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertClose(t, "required reserve", result.RequiredCashReserveBegin, 100)
	assertClose(t, "cash stress ratio", result.BeginCashStressRatio, 2)
	assertClose(t, "layoff pressure", result.CashStressLayoffPressure, 0.5)
	assertClose(t, "layoff rate", result.DistressLayoffRate, 0.125)
	assertClose(t, "voluntary leavers", result.VoluntaryLeavers, 2)
	assertClose(t, "layoffs", result.DistressLayoffs, 1)
	assertClose(t, "desired headcount", result.DesiredHeadcount, 9)
	assertClose(t, "gross hiring need", result.GrossHiringNeed, 2)
	assertClose(t, "hire capacity", result.HireCapacity, 2)
	assertClose(t, "affordable hires", result.AffordableHires, 2)
	assertClose(t, "hires", result.Hires, 2)
	assertClose(t, "headcount end", result.HeadcountEnd, 9)
	assertClose(t, "paid employees", result.PaidEmployees, 9.5)

	assertSliceClose(t, "ramp after exits", result.Ramp.AfterExits, []float64{2.1, 1.4, 0.7})
	assertClose(t, "full after exits", result.Ramp.FullAfter, 2.8)
	assertClose(t, "effective ramp", result.EffectiveRampingEmployees, 2.73)
	assertClose(t, "effective current hires", result.EffectiveNewHires, 1)
	assertClose(t, "effective employees", result.EffectiveEmployees, 6.53)
	assertSliceClose(t, "ramp close", result.Ramp.Close, []float64{0, 4.1, 1.4})
	assertClose(t, "full close", result.Ramp.FullClose, 3.5)
}

func TestRampCohortContinuityUsesEachMultiplierOnce(t *testing.T) {
	cfg := workforceTestConfig()
	cfg.CompanyEconomics.RequiredCashReserveMonths = 0
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0.2
	cfg.Workforce.RecruitingCostPerHire = 1
	cfg.Workforce.OnboardingCostPerHire = 0
	cfg.Workforce.ManagerTimeCostPerHire = 0

	scenario := config.OrganizationalScenario{Name: "scenario"}
	input := WorkforceInput{
		Run:                                   1,
		HeadcountBegin:                        10,
		RampCohortsBegin:                      []float64{0, 0, 0},
		UnrestrictedCashBegin:                 1000,
		MarketDemandForecast:                  1200,
		ProductiveCapacityRevenueMonthlyBegin: 1200,
		ProductivityMultiplier:                1,
	}
	wantEffective := []float64{11, 11.5, 11.8, 12}
	wantRamps := [][]float64{{0, 2, 0}, {0, 0, 2}, {0, 0, 0}, {0, 0, 0}}
	for month := 1; month <= 4; month++ {
		input.Month = month
		result, err := CalculateWorkforce(cfg, scenario, input)
		if err != nil {
			t.Fatalf("month %d: %v", month, err)
		}
		assertClose(t, "effective employees", result.EffectiveEmployees, wantEffective[month-1])
		assertSliceClose(t, "ramp close", result.Ramp.Close, wantRamps[month-1])
		assertClose(t, "ramp/headcount continuity", result.Ramp.FullClose+sum(result.Ramp.Close), result.HeadcountEnd)
		input.HeadcountBegin = result.HeadcountEnd
		input.RampCohortsBegin = append([]float64(nil), result.Ramp.Close...)
	}
}

func TestCalculateWorkforceRampDurationZero(t *testing.T) {
	cfg := workforceTestConfig()
	cfg.Workforce.RampDurationMonths = 0
	cfg.Workforce.RampProductivityMultipliers = nil
	cfg.Workforce.MaxLayoffsPerMonthRate = 0.5
	cfg.Workforce.MaxHiresPerMonthRate = 0.4

	result, err := CalculateWorkforce(cfg, config.OrganizationalScenario{Name: "scenario"}, WorkforceInput{
		Run:                                   1,
		Month:                                 1,
		HeadcountBegin:                        5,
		RampCohortsBegin:                      nil,
		TurnoverRateMonthly:                   0.2,
		UnrestrictedCashBegin:                 200,
		SalaryCostsReferenceMonthly:           100,
		MarketDemandForecast:                  500,
		ProductiveCapacityRevenueMonthlyBegin: 500,
		ProductivityMultiplier:                1,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "duration zero voluntary", result.VoluntaryLeavers, 1)
	assertClose(t, "duration zero layoffs", result.DistressLayoffs, 1)
	assertClose(t, "duration zero hires", result.Hires, 2)
	assertClose(t, "duration zero effective hires", result.EffectiveNewHires, 2)
	assertClose(t, "duration zero effective employees", result.EffectiveEmployees, 5)
	assertClose(t, "duration zero full close", result.Ramp.FullClose, 5)
	if len(result.Ramp.Close) != 0 {
		t.Fatalf("duration zero created ramp cohorts: %v", result.Ramp.Close)
	}
}

func TestIntegerProRataExitsUseStableLargestRemainder(t *testing.T) {
	afterVoluntary, err := applyProRataExits([]float64{4, 3, 2, 1}, 2, config.HeadcountIntegerExpected)
	if err != nil {
		t.Fatal(err)
	}
	assertSliceClose(t, "after voluntary", afterVoluntary, []float64{3, 2, 2, 1})
	afterLayoffs, err := applyProRataExits(afterVoluntary, 2, config.HeadcountIntegerExpected)
	if err != nil {
		t.Fatal(err)
	}
	assertSliceClose(t, "after layoffs", afterLayoffs, []float64{2, 1, 2, 1})
}

func TestHeadcountIntegerQuantizationModes(t *testing.T) {
	assertClose(t, "integer expected down", quantizeCount(2.49, config.HeadcountIntegerExpected, newStableRNG(1)), 2)
	assertClose(t, "integer expected tie up", quantizeCount(2.5, config.HeadcountIntegerExpected, newStableRNG(1)), 3)

	seed := uint64(99)
	expectedRNG := newStableRNG(seed)
	want := 2.0
	if expectedRNG.float64() < 0.49 {
		want = 3
	}
	assertClose(t, "integer random", quantizeCount(2.49, config.HeadcountIntegerRandom, newStableRNG(seed)), want)
}

func TestBinomialTurnoverUsesStableLaborStream(t *testing.T) {
	cfg := workforceTestConfig()
	cfg.Simulation.HeadcountMode = config.HeadcountIntegerExpected
	cfg.Workforce.TurnoverRandomMode = config.TurnoverBinomial
	cfg.Workforce.RampDurationMonths = 0
	cfg.Workforce.RampProductivityMultipliers = nil
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0
	input := WorkforceInput{
		Run:                    3,
		Month:                  5,
		HeadcountBegin:         50,
		TurnoverRateMonthly:    0.2,
		ProductivityMultiplier: 1,
	}

	result, err := CalculateWorkforce(cfg, config.OrganizationalScenario{Name: "scenario"}, input)
	if err != nil {
		t.Fatal(err)
	}
	rng := newStableRNG(deriveStableSeed(cfg.Simulation.RandomSeed, input.Run, input.Month, streamLabor+".voluntary_turnover", ""))
	want := 0
	for range 50 {
		if rng.float64() < 0.2 {
			want++
		}
	}
	assertClose(t, "binomial leavers", result.VoluntaryLeavers, float64(want))
}

func TestLaborRandomnessIsReproducibleAndScenarioOrderIndependent(t *testing.T) {
	cfg := workforceTestConfig()
	cfg.Simulation.CommonRandomNumbers = false
	cfg.Simulation.HeadcountMode = config.HeadcountIntegerRandom
	cfg.Workforce.RampDurationMonths = 0
	cfg.Workforce.RampProductivityMultipliers = nil
	cfg.Workforce.MaxLayoffsPerMonthRate = 0
	cfg.Workforce.MaxHiresPerMonthRate = 0
	input := WorkforceInput{
		Run:                    9,
		Month:                  7,
		HeadcountBegin:         10,
		TurnoverRateMonthly:    0.249,
		ProductivityMultiplier: 1,
	}
	scenarioA := config.OrganizationalScenario{Name: "a"}
	scenarioB := config.OrganizationalScenario{Name: "b"}

	firstA, err := CalculateWorkforce(cfg, scenarioA, input)
	if err != nil {
		t.Fatal(err)
	}
	firstB, err := CalculateWorkforce(cfg, scenarioB, input)
	if err != nil {
		t.Fatal(err)
	}
	secondB, err := CalculateWorkforce(cfg, scenarioB, input)
	if err != nil {
		t.Fatal(err)
	}
	secondA, err := CalculateWorkforce(cfg, scenarioA, input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstA, secondA) || !reflect.DeepEqual(firstB, secondB) {
		t.Fatal("labor results depend on scenario call order")
	}
}

func TestIntegerHiringNeverExceedsAffordability(t *testing.T) {
	for _, mode := range []string{config.HeadcountIntegerExpected, config.HeadcountIntegerRandom} {
		t.Run(mode, func(t *testing.T) {
			cfg := workforceTestConfig()
			cfg.Simulation.HeadcountMode = mode
			cfg.Workforce.RampDurationMonths = 0
			cfg.Workforce.RampProductivityMultipliers = nil
			cfg.Workforce.MaxHiresPerMonthRate = 1
			cfg.Workforce.MaxLayoffsPerMonthRate = 0
			cfg.CompanyEconomics.RequiredCashReserveMonths = 0
			costPerHire := cfg.Workforce.RecruitingCostPerHire +
				cfg.Workforce.OnboardingCostPerHire +
				cfg.Workforce.ManagerTimeCostPerHire

			result, err := CalculateWorkforce(cfg, config.OrganizationalScenario{Name: "test"}, WorkforceInput{
				Run:                                   1,
				Month:                                 1,
				HeadcountBegin:                        10,
				UnrestrictedCashBegin:                 0.6 * costPerHire,
				MarketDemandForecast:                  1_000_000_000,
				ProductiveCapacityRevenueMonthlyBegin: 1_000_000_000,
				ProductivityMultiplier:                1,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.AffordableHires <= 0 || result.AffordableHires >= 1 {
				t.Fatalf("fixture affordable hires = %g, want between 0 and 1", result.AffordableHires)
			}
			if result.Hires != 0 {
				t.Fatalf("hires = %g with affordability %g, want 0", result.Hires, result.AffordableHires)
			}
		})
	}
}

func workforceTestConfig() config.Config {
	return config.Config{
		Simulation: config.Simulation{
			Mode:                config.ModeDeterministic,
			RandomSeed:          42,
			CommonRandomNumbers: true,
			Epsilon:             1e-9,
			HeadcountMode:       config.HeadcountFractional,
		},
		CompanyEconomics: config.CompanyEconomics{
			BaseRevenuePerEffectiveEmployeeMonthly: 100,
			RequiredCashReserveMonths:              1,
		},
		Workforce: config.Workforce{
			TurnoverRandomMode:          config.TurnoverDeterministic,
			RecruitingCostPerHire:       50,
			RampDurationMonths:          3,
			RampProductivityMultipliers: []float64{0.5, 0.75, 0.9},
			MaxHiresPerMonthRate:        0.2,
			MaxLayoffsPerMonthRate:      0.25,
			LayoffTriggerCashRatio:      4,
			LeaverPaidFractionOfMonth:   0.5,
			NewHirePaidFractionOfMonth:  0.5,
			TargetStaffingBuffer:        1,
			MaxCashShareForHiring:       1,
		},
	}
}

func assertSliceClose(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length=%d, want %d (%v vs %v)", name, len(got), len(want), got, want)
	}
	for index := range got {
		assertClose(t, name, got[index], want[index])
	}
}
