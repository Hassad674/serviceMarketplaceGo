# `internal/search/scorer`

Stage 4 of the Ranking V1 pipeline — composite scoring. Turns the
9-feature vector produced by `internal/search/features` into a
`RankedScore{Base, Adjusted, Final}` triple plus a per-feature
breakdown for explainability.

Specification : `docs/ranking-v1.md`, sections

- **§4** — per-persona weight tables (freelance, agency, referrer)
- **§5** — composite formula (positive → adjusted → final), empty-query redistribution (§5.2), negative penalty (§5.3)
- **§10.4** — `Reranker` / `RankedScore` interface contract
- **§11.1** — env-var configuration reference

## Public surface

```go
type Reranker interface {
    Score(ctx context.Context, q Query, f Features, persona Persona) RankedScore
}

type RankedScore struct {
    Base      float64            // [0, 1]   positive composite before negatives
    Adjusted  float64            // [0, 1]   after × (1 − NegativeSignals)
    Final     float64            // [0, 100] display score
    Breakdown map[string]float64 // per-feature contributions
}

type Config struct{ Freelance, Agency, Referrer PersonaWeights }
func LoadConfigFromEnv() (Config, error)
func NewWeightedScorer(cfg Config) *WeightedScorer
```

## Formula (V1)

```
positive = Σ weight_persona[i] × feature[i]
adjusted = positive × (1 − NegativeSignals)     // NegativeSignals ∈ [0, 0.30]
final    = clamp(adjusted, 0, 1) × 100
```

All values are clamped defensively — a misbehaving feature extractor
cannot push the final score outside `[0, 100]`.

## Empty-query redistribution (§5.2)

When `query.Text` is blank, `TextMatchScore` is forced to 0 and its
weight (15–20 %) is proportionally redistributed across the remaining
eight features. The redistributed weights still sum to 1.0 within
`floatTolerance` — asserted by `TestRedistribute_PreservesSum`.

## Configuration

Every weight is loaded from `RANKING_WEIGHTS_<PERSONA>_<FEATURE>` env
vars. Defaults are hardcoded and verified (via `TestDefaultWeights_SumToOne`)
to sum to 1.0 per persona. See `docs/ranking-v1.md §11.1` for the
complete reference.

Invalid configuration (sum drift, non-parseable float) surfaces as
`ErrWeightsSum` from `LoadConfigFromEnv`, so the backend fails fast
at startup rather than silently producing biased scores.

## Performance

Benchmark on Intel i5-1334U :

```
BenchmarkScore-12              388 ns/op    504 B/op    4 allocs/op
BenchmarkScoreEmptyQuery-12    394 ns/op    504 B/op    4 allocs/op
```

The four allocations come from constructing the `Breakdown` map (9
entries + map header). At 388 ns × 200 candidates per search ≈ 78 µs,
well within the Stage 2-5 budget of 50 ms p95.

## Testing

Coverage : 100 % of statements.

Categories :

- **Table-driven** — every persona × empty/non-empty query × min/mid/max features × with/without penalty.
- **Property** — randomised 1000-iteration bounds + monotonicity + negative-signal monotonicity.
- **Contract** — compile-time + runtime assertion that `*WeightedScorer` implements `Reranker`.
- **Defensive** — NaN inputs collapse to zero; over-unity feature values clamp to 1.

Run :

```
go test ./internal/search/scorer/... -count=1 -race -cover
go test -bench=. ./internal/search/scorer/... -benchmem
```

## Cross-agent contract

This package currently mirrors `internal/search/features.Features`
locally because R2-F was still landing the `features` package at the
time of this commit. Once `features` is merged, the local `Features`
and `Persona` types in `types.go` are replaced by re-exports from
`internal/search/features` in a single follow-up commit — the wire
shape is byte-identical so no callers change.

## Stage 5 wiring

The `Reranker` consumed by the query service (Stage 5 — business
rules) expects exactly this interface. The V2 swap path is documented
in `docs/ranking-v1.md §9.3` : replace `WeightedScorer` with
`LTRScorer`; no signature change.
