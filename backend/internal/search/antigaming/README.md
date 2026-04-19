# `internal/search/antigaming`

Stage 3 of the ranking V1 pipeline : silent caps + penalties applied to the
in-flight `*features.Features` value before the scorer runs.

Specification : `docs/ranking-v1.md` §7. Thresholds : `docs/ranking-v1.md`
§11.4, all env-tunable under the `RANKING_AG_*` namespace.

## Public surface

- `Config` + `DefaultConfig()` + `LoadConfigFromEnv()` — thresholds.
- `RawSignals` — per-candidate input : text content, recent review
  timestamps, reviewer IDs, account age. Populated once by the caller.
- `LinkedReviewersDetector` — the hook for rule 3. V1 ships
  `NoopLinkedReviewersDetector` that always returns 0 ; production wires
  in a real detector backed by `users` + `sessions`.
- `Logger` + `SlogLogger` + `NoopLogger` + `RecordingLogger` — every
  penalty fires through a `Logger`. `SlogLogger` writes the JSON line
  documented in §7.6.
- `Rule` + `Penalty` — identifying the fired rule + its context for
  downstream dashboards.
- `Pipeline.NewPipeline(cfg, detector, logger)` → `*Pipeline` — the
  orchestrator. `Apply(ctx, *Features, RawSignals) PipelineResult` runs
  the five rules in the order below + mutates the features in place.

## The five rules

| # | Rule | Spec | Mutation on firing |
|---|---|---|---|
| 1 | `RuleKeywordStuffing` | §7.1 | `TextMatchScore *= StuffingPenalty` (0.5) |
| 2 | `RuleReviewVelocity`  | §7.2 | `RatingScoreDiverse *= (n - excess) / n` |
| 3 | `RuleLinkedAccounts`  | §7.3 | `RatingScoreDiverse *= (1 - linked_fraction)` |
| 4 | `RuleReviewerFloor`   | §7.4 | `RatingScoreDiverse = min(rating, FewReviewerCap)` |
| 5 | `RuleNewAccount`      | §7.5 | `AccountAgeBonus = 0` + sets `PipelineResult.NewAccountCapped = true` |

## Invariants (enforced by tests)

1. `Pipeline.Apply` is safe on a `nil` features pointer (no-op).
2. `Pipeline.Apply` reaches a fixed point : applying twice on the same
   `(Features, RawSignals)` yields identical final features.
3. Detector errors never poison the scorer — swallowed silently.
4. Penalties are only emitted when the rule actually mutates a feature
   (keeps the log signal clean).
5. After `Apply`, the positive-contribution fields of `Features` stay in
   `[0, 1]`.

## Logging contract

Every fired rule emits one `slog.Info("ranking.penalty_applied", ...)`
line with eight pinned fields (see `logger.go`). Shape is regression-
locked in `TestSlogLogger_EmitsStructuredLine` — a field rename fails
the build.
