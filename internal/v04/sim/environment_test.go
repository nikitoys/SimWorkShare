package sim

import (
	"math"
	"reflect"
	"testing"

	"simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

func TestGenerateEnvironmentPathDeterministic(t *testing.T) {
	cfg := environmentTestConfig()
	cfg.Simulation.Mode = config.ModeDeterministic
	cfg.Simulation.Months = 3
	cfg.Market.MarketGrowthMonthly = 0.01
	cfg.Market.MarketVolatilityMonthly = 0.9
	cfg.Market.ShockProbabilityMonthly = 1
	cfg.Market.ShockCostMean = 100
	cfg.Market.SeasonalityMultipliers = []float64{1, 1.2}
	cfg.Market.LaborMarketFactor = 1.1
	cfg.Market.CreditMarketFactor = 0.8

	path, err := GenerateEnvironmentPath(cfg, 7, "ignored-scenario")
	if err != nil {
		t.Fatal(err)
	}
	if len(path) != 3 {
		t.Fatalf("len(path)=%d, want 3", len(path))
	}
	wantTrend := []float64{1.01, 1.0201, 1.030301}
	wantSeasonality := []float64{1, 1.2, 1}
	for index, month := range path {
		assertClose(t, "market trend", month.MarketTrend, wantTrend[index])
		assertClose(t, "market factor", month.MarketFactor, 1)
		assertClose(t, "seasonality", month.SeasonalityMultiplier, wantSeasonality[index])
		assertClose(t, "collection multiplier", month.CollectionRateMultiplier, 1)
		assertClose(t, "shock cost", month.ShockCost, 0)
		if month.ShockHappened {
			t.Fatalf("deterministic month %d unexpectedly has a shock", month.Month)
		}
		if month.Run != 7 || month.Month != index+1 {
			t.Fatalf("unexpected run/month: %+v", month)
		}
	}
}

func TestGenerateEnvironmentPathBoundedLognormalExactFormula(t *testing.T) {
	cfg := environmentTestConfig()
	cfg.Simulation.Months = 1
	cfg.Simulation.RandomSeed = 123456
	cfg.Market.MarketVolatilityMonthly = 0.31
	cfg.Market.MarketFactorMin = 0.9
	cfg.Market.MarketFactorMax = 1.1
	cfg.Market.ShockProbabilityMonthly = 1
	cfg.Market.ShockCostMean = 20
	cfg.Market.ShockCostStd = 7

	path, err := GenerateEnvironmentPath(cfg, 4, "scenario-a")
	if err != nil {
		t.Fatal(err)
	}
	marketRNG := newStableRNG(deriveStableSeed(cfg.Simulation.RandomSeed, 4, 1, streamMarket, ""))
	zMarket := marketRNG.normal()
	wantMarket := domain.Clamp(
		math.Exp(-0.5*cfg.Market.MarketVolatilityMonthly*cfg.Market.MarketVolatilityMonthly+cfg.Market.MarketVolatilityMonthly*zMarket),
		cfg.Market.MarketFactorMin,
		cfg.Market.MarketFactorMax,
	)
	costRNG := newStableRNG(deriveStableSeed(cfg.Simulation.RandomSeed, 4, 1, streamShockCost, ""))
	wantCost := math.Max(0, cfg.Market.ShockCostMean+cfg.Market.ShockCostStd*costRNG.normal())

	assertClose(t, "bounded lognormal market factor", path[0].MarketFactor, wantMarket)
	assertClose(t, "shock cost", path[0].ShockCost, wantCost)
	if !path[0].ShockHappened {
		t.Fatal("shock_probability_monthly=1 did not produce a shock")
	}
	assertClose(t, "stressed collection multiplier", path[0].CollectionRateMultiplier, cfg.Market.CashCollectionStressMultiplier)
}

func TestGenerateEnvironmentPathCommonRandomNumbersAndOrderIndependence(t *testing.T) {
	cfg := environmentTestConfig()
	cfg.Simulation.Months = 24
	cfg.Simulation.RandomSeed = 91

	pathA, err := GenerateEnvironmentPath(cfg, 2, "scenario-a")
	if err != nil {
		t.Fatal(err)
	}
	pathB, err := GenerateEnvironmentPath(cfg, 2, "scenario-b")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pathA, pathB) {
		t.Fatal("common_random_numbers=true produced scenario-specific paths")
	}

	cfg.Simulation.CommonRandomNumbers = false
	firstA, err := GenerateEnvironmentPath(cfg, 2, "scenario-a")
	if err != nil {
		t.Fatal(err)
	}
	firstB, err := GenerateEnvironmentPath(cfg, 2, "scenario-b")
	if err != nil {
		t.Fatal(err)
	}
	secondB, err := GenerateEnvironmentPath(cfg, 2, "scenario-b")
	if err != nil {
		t.Fatal(err)
	}
	secondA, err := GenerateEnvironmentPath(cfg, 2, "scenario-a")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstA, secondA) || !reflect.DeepEqual(firstB, secondB) {
		t.Fatal("path changed when scenario call order changed")
	}
	if reflect.DeepEqual(firstA, firstB) {
		t.Fatal("common_random_numbers=false did not use the scenario key")
	}
}

func environmentTestConfig() config.Config {
	return config.Config{
		Simulation: config.Simulation{
			Mode:                config.ModeMonteCarlo,
			Months:              12,
			RandomSeed:          42,
			CommonRandomNumbers: true,
		},
		Market: config.Market{
			MarketProcess:                  config.MarketProcessBoundedLognormal,
			MarketVolatilityMonthly:        0.08,
			MarketFactorMin:                0.5,
			MarketFactorMax:                1.8,
			SeasonalityMultipliers:         []float64{1},
			ShockProbabilityMonthly:        0.2,
			ShockRevenueMultiplier:         0.8,
			CashCollectionStressMultiplier: 0.9,
			LaborMarketFactor:              1,
			CreditMarketFactor:             1,
		},
	}
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	const tolerance = 1e-12
	scale := math.Max(1, math.Max(math.Abs(got), math.Abs(want)))
	if math.Abs(got-want) > tolerance*scale {
		t.Fatalf("%s=%0.17g, want %0.17g", name, got, want)
	}
}
