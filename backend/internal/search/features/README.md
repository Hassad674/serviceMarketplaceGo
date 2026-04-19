# `internal/search/features`

Stage 2 of the ranking V1 pipeline : feature extraction.

All formulas are specified in `docs/ranking-v1.md` §3. This package is the
Go implementation + its test suite.

## Public surface

- `Features` — the frozen 10-component + raw-signals contract consumed by
  the scorer (R2-S) and the anti-gaming pipeline (§7). See `types.go`.
- `Query` — request-scoped input shared by every extractor.
- `SearchDocumentLite` — read-only copy of the subset of
  `search.SearchDocument` fields the extractors depend on. Keeps the
  package dependency-free from the wider `search` module.
- `Config` — formula parameters loaded from `RANKING_*` env vars at
  startup. `DefaultConfig()` returns the safe public defaults ;
  `LoadConfigFromEnv()` overlays env var overrides.
- `DefaultExtractor` — production extractor. `NewDefaultExtractor(cfg)`
  constructs one ; `Extract(query, doc) Features` runs all ten extractors.
- `ExtractorFunc` — function-to-interface adapter for scorer stubs.

Per-feature exported helpers (one per `extract_*.go` file) are stable + can
be called independently if a caller only needs a single feature. They all
have the signature `(Query, SearchDocumentLite, Config) -> float64` (or
tuple) and live next to their matching `_test.go` file.

## Spec cross-reference

| Extractor | Spec |
|---|---|
| `ExtractTextMatch` | §3.2-1 |
| `ExtractSkillsOverlap` | §3.2-2 |
| `ExtractRatingDiverse` | §3.2-3 |
| `ExtractProvenWork` | §3.2-4 |
| `ExtractResponseRate` | §3.2-5 |
| `ExtractVerifiedMature` | §3.2-6 |
| `ExtractProfileCompletion` | §3.2-7 |
| `ExtractLastActiveDays` | §3.2-8 |
| `ExtractAccountAgeBonus` | §3.2-9 |
| `ExtractNegativeSignals` | §5.3 |

## Invariants (enforced by tests)

1. Every positive component of `Features` stays in `[0, 1]`.
2. `Features.NegativeSignals` stays in `[0, Config.DisputePenaltyCap]`.
3. `Extract` is a pure function : same `(query, doc)` → same `Features`.
4. Referrer persona : `SkillsOverlapRatio = 0` and `ProvenWorkScore = 0`
   regardless of doc content.
5. Cold-start profile : `RatingScoreDiverse = Config.ColdStartFloor`.

## Performance

Benchmarked at ≈ 120 ns/op and 0 allocations per `Extract` on commodity
hardware (Intel i5-1334U). A 200-candidate re-rank therefore takes ~24 µs
of pure extract work — trivial against the 50 ms p95 budget.
