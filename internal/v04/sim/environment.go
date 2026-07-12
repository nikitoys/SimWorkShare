package sim

import (
	"fmt"
	"math"
	"strings"

	"simworkshare/internal/v04/config"
	"simworkshare/internal/v04/domain"
)

const (
	streamMarket    = "market_stream"
	streamShock     = "shock_stream"
	streamShockCost = "shock_cost_stream"
	streamLabor     = "labor_stream"
)

// GenerateEnvironmentPath creates the external path for one run. When common
// random numbers are enabled, scenarioKey is deliberately ignored. Otherwise
// callers may pass a stable scenario key to obtain a separate, order-independent
// path. Months in the returned slice are numbered from 1.
func GenerateEnvironmentPath(cfg config.Config, run int, scenarioKey ...string) ([]domain.EnvironmentMonth, error) {
	if run < 0 {
		return nil, fmt.Errorf("run must be >= 0")
	}
	if cfg.Simulation.Months < 0 {
		return nil, fmt.Errorf("simulation.months must be >= 0")
	}
	if len(cfg.Market.SeasonalityMultipliers) == 0 {
		return nil, fmt.Errorf("market.seasonality_multipliers must not be empty")
	}

	key := ""
	if !cfg.Simulation.CommonRandomNumbers {
		key = strings.Join(scenarioKey, "\x00")
	}

	path := make([]domain.EnvironmentMonth, cfg.Simulation.Months)
	trend := 1.0
	deterministicRun := cfg.Simulation.Mode == config.ModeDeterministic
	for i := range path {
		month := i + 1
		trend *= 1 + cfg.Market.MarketGrowthMonthly
		if !domain.Finite(trend) {
			return nil, fmt.Errorf("market trend is not finite at month %d", month)
		}

		marketFactor := 1.0
		if !deterministicRun && cfg.Market.MarketProcess == config.MarketProcessBoundedLognormal {
			rng := newStableRNG(deriveStableSeed(cfg.Simulation.RandomSeed, run, month, streamMarket, key))
			z := rng.normal()
			sigma := cfg.Market.MarketVolatilityMonthly
			raw := math.Exp(-0.5*sigma*sigma + sigma*z)
			marketFactor = domain.Clamp(raw, cfg.Market.MarketFactorMin, cfg.Market.MarketFactorMax)
		}

		shockHappened := false
		shockCost := 0.0
		collectionMultiplier := 1.0
		if !deterministicRun {
			shockRNG := newStableRNG(deriveStableSeed(cfg.Simulation.RandomSeed, run, month, streamShock, key))
			shockHappened = shockRNG.float64() < cfg.Market.ShockProbabilityMonthly
			if shockHappened {
				collectionMultiplier = cfg.Market.CashCollectionStressMultiplier
				costRNG := newStableRNG(deriveStableSeed(cfg.Simulation.RandomSeed, run, month, streamShockCost, key))
				shockCost = math.Max(0, cfg.Market.ShockCostMean+cfg.Market.ShockCostStd*costRNG.normal())
			}
		}

		seasonality := cfg.Market.SeasonalityMultipliers[i%len(cfg.Market.SeasonalityMultipliers)]
		values := []float64{marketFactor, shockCost, collectionMultiplier, seasonality, cfg.Market.LaborMarketFactor, cfg.Market.CreditMarketFactor}
		for _, value := range values {
			if !domain.Finite(value) {
				return nil, fmt.Errorf("environment value is not finite at month %d", month)
			}
		}

		path[i] = domain.EnvironmentMonth{
			Run:                      run,
			Month:                    month,
			MarketTrend:              trend,
			MarketFactor:             marketFactor,
			SeasonalityMultiplier:    seasonality,
			ShockHappened:            shockHappened,
			ShockCost:                shockCost,
			CollectionRateMultiplier: collectionMultiplier,
			LaborMarketFactor:        cfg.Market.LaborMarketFactor,
			CreditMarketFactor:       cfg.Market.CreditMarketFactor,
		}
	}
	return path, nil
}

// stableRNG is a small SplitMix64-based generator. Its algorithm is local to
// the model, so seeded results do not depend on changes to math/rand.
type stableRNG struct {
	state uint64
}

func newStableRNG(seed uint64) *stableRNG {
	return &stableRNG{state: seed}
}

func (r *stableRNG) uint64() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (r *stableRNG) float64() float64 {
	// The half-unit offset keeps Box-Muller away from both 0 and 1.
	return (float64(r.uint64()>>11) + 0.5) / (1 << 53)
}

func (r *stableRNG) normal() float64 {
	u1 := r.float64()
	u2 := r.float64()
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// deriveStableSeed addresses each random draw family independently by base
// seed, run, month, named stream and optional stable key.
func deriveStableSeed(base int64, run, month int, stream, key string) uint64 {
	h := uint64(14695981039346656037)
	mixByte := func(b byte) {
		h ^= uint64(b)
		h *= 1099511628211
	}
	mixUint64 := func(v uint64) {
		for i := 0; i < 8; i++ {
			mixByte(byte(v))
			v >>= 8
		}
	}
	mixString := func(value string) {
		mixUint64(uint64(len(value)))
		for i := 0; i < len(value); i++ {
			mixByte(value[i])
		}
	}

	mixUint64(uint64(base))
	mixUint64(uint64(run))
	mixUint64(uint64(month))
	mixString(stream)
	mixString(key)

	// One avalanche step prevents closely related FNV states from becoming
	// closely related SplitMix starting states.
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	h *= 0x94d049bb133111eb
	return h ^ (h >> 31)
}
