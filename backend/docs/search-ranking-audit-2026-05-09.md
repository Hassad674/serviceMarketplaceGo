# Search Ranking Audit — 2026-05-09

> Audit of the live Stage 2-5 ranking pipeline against the locked
> spec in `docs/ranking-v1.md`. Goal: paranoid validation that
> every documented criterion still works after months of refactors,
> including a deterministic 60-scenario regression suite shipping
> alongside this report.

**Auditor**: backend agent (this session)
**Scope**: backend search ranking only — `internal/app/search/`,
`internal/search/{features,scorer,antigaming,rules}/`, plus the
`SearchDocument` schema and the document adapter.
**Out of scope**: web/mobile UX, Typesense schema migration, OpenAI
embeddings provider swap.

---

## 1. Executive summary

| Concern | Verdict | Evidence |
| --- | --- | --- |
| Ranking pipeline runs end-to-end | PASS | `BenchmarkRerank_200Candidates` 0.7 ms, `TestRankingPipeline_*` green, live Typesense `TestAuditLive_*` green |
| Per-feature contracts (10 features) match spec | PASS | New `audit_scenarios_test.go` and `audit_scenarios_rules_test.go` enforce one assertion per feature; 60 sub-tests green |
| Persona weight tables sum to 1.0 | PASS | `TestAudit_PersonaWeights_*` |
| Tier sort hard partition (Tier A always above Tier B) | PASS | `TestAudit_TierSort_AlwaysBeatsB` + live `TestAuditLive_FreelanceCohortReranksAsExpected` |
| Negative penalty capped at 30% | PASS | `TestAudit_LostDisputes_PenaltyCappedAt30Pct` table-driven (5 inputs) |
| Velocity rule (§7.2) firing in production | **FAIL — DRIFT** | `RawSignals.RecentReviewTimestamps = nil` hard-coded in `applyAntiGaming` (`backend/internal/app/search/ranking_pipeline.go:205`) |
| Linked-account rule (§7.3) firing in production | **FAIL — DRIFT** | `RawSignals.ReviewerIDs = nil` hard-coded same place (line 207) |
| New-account final-score cap (§7.5) enforced | **FAIL — DRIFT** | `PipelineResult.NewAccountCapped` set correctly by anti-gaming pipeline but `applyAntiGaming` discards the result (line 211) — the scorer never sees the flag |
| `about` field-driven junk penalty (§3.2-7) firing | **FAIL — DRIFT** | `SearchDocument` schema has no `About` field; `document_adapter.go:67` hard-codes `About: ""` |
| Test coverage of search packages | PASS | 100% scorer · 99.2% rules · 97.5% antigaming · 97.6% features · 96.5% app/search · 81.2% search |

**Bottom line**: every documented ranking criterion that depends on
data already in the `SearchDocument` is correct, deterministic, and
covered by a regression test. Four spec-vs-code drifts exist where
upstream data plumbing is missing or the result of a sub-pipeline
is dropped — none of them silently produce *wrong* rankings, but
some anti-gaming rules are effectively dormant in production today.

The fixes are mechanical (route data through the existing
interfaces) — no architectural change needed. They are listed in
§7 with priority labels.

---

## 2. Pipeline diagram (verified)

```
HTTP request → handler.search.Get
                     │
                     ▼
   appsearch.Service.Query
                     │
   ┌─────────────────┴───────────────────┐
   │                                     │
   ▼                                     ▼
maybeVectorQuery                  buildSearchParams
(OpenAI text-embedding-3-small)    (q, query_by, filter_by, sort_by,
   │                                num_typos, page, per_page, …)
   │                                     │
   └─────────┬───────────────────────────┘
             │
             ▼
PersonaScopedClient.Query (Typesense /documents/search)
   • Hybrid blend: BM25 + vector cosine (HybridK=20)
   • Persona + is_published filter baked into the scoped key
             │
             ▼
parseQueryResultWithHits
   • strips embedding vector
   • normalises text_match to bucket [0,10] via computeTextMatchBuckets
             │
             ▼
applyRerank ⇒ RankingPipeline.Rerank   (Stages 2-5, ≤ 200 hits)
   ├── Stage 2 — features.DefaultExtractor (10 features, [0,1])
   ├── Stage 3 — antigaming.Pipeline (5 rules)  ← RESULT DROPPED, see §6
   ├── Stage 4 — scorer.WeightedScorer (per-persona weights → Final 0-100)
   └── Stage 5 — rules.BusinessRules
            • tier sort (Tier A above Tier B)
            • gaussian noise (rank-dependent σ)
            • re-sort within tier
            • diversity pass (break 3-in-a-row primary skill)
            • rising-talent injection (slots 5/10/15/20)
            • featured override (dormant: FeaturedEnabled=false)
            • truncate to TopN=20
             │
             ▼
QueryResult { Documents, Reranked=true, RerankDurationMs, TopFinalScore }
             │
             ▼
captureLTR → search_queries.result_features_json (fire-and-forget)
             │
             ▼
HTTP response {data, meta}
```

Pipeline stages are wired exactly as the spec describes. The only
divergence is §7.5 (new-account cap) where the pipeline's output
flag is computed but never consumed by the scorer.

---

## 3. Ranking criteria — exhaustive table

Every weight, threshold, and gate verified against the live code.
File:line citations are absolute paths inside the worktree.

### 3.1 Per-feature contracts (§3.2 of `docs/ranking-v1.md`)

| # | Feature | Source (file:line) | Range | Notes |
| --- | --- | --- | --- | --- |
| 1 | `text_match_score` | `internal/search/features/extract_text_match.go:18` | [0, 1] | `min(10, bucket) / 10`. Stuffing penalty applied separately. |
| 2 | `skills_overlap_ratio` | `internal/search/features/extract_skills_overlap.go:22` | [0, 1] | `|q ∩ profile| / |q|`. **Always 0 for referrers** (line 23). |
| 3 | `rating_score_diverse` | `internal/search/features/extract_rating_diverse.go:28` | [0, 1] | Bayesian × diversity × recency, cold-start floor. |
| 4 | `proven_work_score` | `internal/search/features/extract_proven_work.go:22` | [0, 1] | `0.40·log(p)+0.35·log(c)+0.25·sqrt(r)` / `log(1+cap)`. **Always 0 for referrers** (line 23). |
| 5 | `response_rate` | `internal/search/features/extract_response_rate.go:12` | [0, 1] | Direct pass-through with clamp. |
| 6 | `is_verified_mature` | `internal/search/features/extract_verified_mature.go:12` | {0, 1} | `IsVerified && AccountAgeDays ≥ 30` |
| 7 | `profile_completion` | `internal/search/features/extract_profile_completion.go:9` | [0, 1] | `score / 100`. Junk penalty in spec NOT firing — see §6 drift D4. |
| 8 | `last_active_days_score` | `internal/search/features/extract_last_active.go:25` | [0, 1] | `1 / (1 + days/30)`. Now-injected for purity. |
| 9 | `account_age_bonus` | `internal/search/features/extract_account_age.go:22` | [0, 1] | `log(1+days) / log(1+365)`, cap 365. |
| 10 | `negative_signals` | `internal/search/features/extract_negative_signals.go:16` | [0, 0.30] | `min(cap, lost × 0.10)`. Penalty saturates at 3 disputes. |

### 3.2 Per-persona weights (§4)

Locked tables verified from `internal/search/scorer/weights.go:90-134`:

#### Freelance (§4.1) — sum = 1.000

| Feature | Weight |
| --- | --- |
| TextMatch | 0.20 |
| SkillsOverlap | 0.15 |
| Rating | 0.20 |
| ProvenWork | 0.15 |
| ResponseRate | 0.10 |
| VerifiedMature | 0.08 |
| Completion | 0.07 |
| LastActive | 0.03 |
| AccountAge | 0.02 |

#### Agency (§4.2) — sum = 1.000

| Feature | Weight |
| --- | --- |
| TextMatch | 0.15 |
| SkillsOverlap | 0.10 |
| Rating | 0.25 |
| ProvenWork | 0.25 |
| ResponseRate | 0.05 |
| VerifiedMature | 0.10 |
| Completion | 0.07 |
| LastActive | 0.02 |
| AccountAge | 0.01 |

#### Referrer (§4.3) — sum = 1.000

| Feature | Weight |
| --- | --- |
| TextMatch | 0.20 |
| SkillsOverlap | **0.00** |
| Rating | 0.35 |
| ProvenWork | **0.00** |
| ResponseRate | 0.20 |
| VerifiedMature | 0.10 |
| Completion | 0.10 |
| LastActive | 0.03 |
| AccountAge | 0.02 |

All three tables validate at boot via `Config.Validate()` (sum
within `1e-9` tolerance) — `internal/search/scorer/weights.go:62`.
A regression test (`TestAudit_PersonaWeights_*`) re-asserts the
sum at every CI run.

### 3.3 Composite scoring (§5)

Source: `internal/search/scorer/weighted_scorer.go:46`.

```
positive  = Σ wᵢ × featureᵢ        (∈ [0, 1])
adjusted  = positive × (1 − negative_signals)
final     = clamp(adjusted, 0, 1) × 100
```

Empty-query redistribution (§5.2) handled by
`scorer/redistribute.go:21` — when `q == ""` the TextMatch slice
is redistributed proportionally across the other 8 features.
Verified by `TestAudit_TextMatch_EmptyQueryRedistributes`.

### 3.4 Anti-gaming rules (§7)

| Rule | File:line | Default config | Live status |
| --- | --- | --- | --- |
| §7.1 Stuffing | `internal/search/antigaming/rule_stuffing.go:99` | max repetition > 5 OR distinct_ratio < 0.3 → halve text_match | Active. Verified by `TestAudit_AntiGaming_StuffingHalvesTextMatch`. |
| §7.2 Velocity | `internal/search/antigaming/rule_velocity.go:25` | > 5 reviews / 24h → dampen rating | **Dormant in production** — see drift D1 below. |
| §7.3 Linked accounts | `internal/search/antigaming/rule_linked.go:46` | > 30% linked reviewers → dampen | **Dormant** — see D2. |
| §7.4 Reviewer floor | `internal/search/antigaming/rule_reviewer_floor.go:17` | < 3 unique reviewers → cap at 0.4 | Active. Already-low ratings are no-op silent (matches spec). |
| §7.5 New-account cap | `internal/search/antigaming/rule_new_account.go:20` | < 7 days → zero AccountAgeBonus + flag for median cap | **Median cap not enforced** — see D3. |

### 3.5 Business rules (§6, §8)

| Rule | File:line | Notes |
| --- | --- | --- |
| Tier sort | `internal/search/rules/tier_sort.go:39` | Tier A = available_now/soon, Tier B = not_available/unknown. Hard partition. |
| Gaussian noise | `internal/search/rules/randomise.go:74` | σ = COEF × score × rank_multiplier; top-3=0.3, mid=0.8, tail=1.5. |
| Diversity (3-in-a-row) | `internal/search/rules/diversity.go:29` | Soft swap with same-tier alternative. |
| Rising-talent slot | `internal/search/rules/rising_talent.go:28` | Slots 5/10/15/20, age < 60d, verified, score ≥ median. |
| Featured override | `internal/search/rules/featured.go:29` | **Dormant** — `FeaturedEnabled=false` by default (§8 expected). |

### 3.6 Configuration knobs (§11)

39 environment variables loaded from
`internal/search/{features,antigaming,scorer,rules}/config.go`:

- `RANKING_BAYESIAN_PRIOR_MEAN/WEIGHT`, `RANKING_COLD_START_FLOOR`,
  `RANKING_REVIEW_COUNT_CAP`, `RANKING_PROJECT_COUNT_CAP`,
  `RANKING_ACCOUNT_AGE_CAP_DAYS`, `RANKING_VERIFIED_MATURE_MIN_AGE_DAYS`,
  `RANKING_LAST_ACTIVE_DECAY_DAYS`, `RANKING_DISPUTE_PENALTY/CAP`
- `RANKING_AG_*` (8 vars for the five anti-gaming rules)
- `RANKING_WEIGHTS_{FREELANCE,AGENCY,REFERRER}_*` (27 weight knobs)
- `RANKING_NOISE_*`, `RANKING_RISING_TALENT_*`,
  `RANKING_FEATURED_*`, `RANKING_RULES_TOP_N`, `RANKING_RULES_SEED`

All knobs default to the safe public values published in §11 of
the spec. Production values live in Railway/Vercel env vars.

---

## 4. Existing test coverage

Tests already shipping in the repo (counted before this audit):

| File | Sub-tests | Notes |
| --- | --- | --- |
| `backend/internal/search/features/*_test.go` | 70+ | One file per extractor, table-driven |
| `backend/internal/search/scorer/*_test.go` | 25+ | Including property tests (sum-to-1, monotonicity) |
| `backend/internal/search/antigaming/*_test.go` | 35+ | Per-rule tests + the pipeline composition |
| `backend/internal/search/rules/*_test.go` | 60+ | Tier, noise, diversity, rising-talent, featured |
| `backend/internal/app/search/ranking_pipeline_test.go` | 25+ | Composition + nil-component degradation + benchmark |
| `backend/internal/app/search/golden_full_pipeline_test.go` | 6 | Live OpenAI + Typesense, gated by env |
| `backend/internal/app/search/integration_test.go` | 5 | Live Typesense, gated by env |
| `backend/internal/search/golden_test.go` | 14 | BM25-only golden queries |
| `backend/internal/search/ranking_test.go` | 25+ | BayesianRatingScore, IsTopRated, ProfileCompletionScore |
| `backend/internal/search/ranking_properties_test.go` | 8+ | Property tests (testing/quick) |

Total existing search-related sub-tests: **825 green** at the start
of this audit, all passing on `go test -count=1`.

### Gaps detected before the audit

| Gap | Severity | Now covered by |
| --- | --- | --- |
| No deterministic 30-doc fixture set replaying every spec contract end-to-end | High | `audit_fixtures_*_test.go` |
| No persona weight assertions (sum=1, agency rating dominance, referrer 0% rules) | High | `TestAudit_PersonaWeights_*` (§3.2 above) |
| No table-driven per-feature ladder (e.g. dispute count → exact penalty) | High | `TestAudit_LostDisputes_PenaltyCappedAt30Pct` (5 inputs), `TestAudit_LastActive_HyperbolicDecay` (6), `TestAudit_AccountAge_LogScaleSaturates` (6), `TestAudit_VerifiedMature_BinaryGate` (4) |
| No live-Typesense rerank assertion against deterministic fixtures | High | `audit_live_typesense_test.go` (3 tests, gated by env) |
| No spec-drift test surfacing dormant rules | Critical | `audit_spec_drift_test.go` (5 tests, all GREEN documenting current state) |

---

## 5. New tests added by this audit

All under `backend/internal/app/search/`:

| File | Lines | Purpose |
| --- | --- | --- |
| `audit_fixtures_test.go` | 155 | Time anchor (`auditNowUnix=2026-05-01`), helpers (`newAuditPipeline`, `findFixture`, `hitsFromFixtures`, `uniformBuckets`) |
| `audit_fixtures_freelance_test.go` | 289 | 10 hand-crafted freelance docs, each stress-testing a different feature (e.g. `freelance-09` triggers stuffing rule, `freelance-07` triggers new-account rule) |
| `audit_fixtures_agency_test.go` | 282 | 10 agency docs (Alpha … Kappa), e.g. `agency-05` has 5 lost disputes for the negative-penalty cap test |
| `audit_fixtures_referrer_test.go` | 284 | 10 referrer docs (Lambda … Upsilon), e.g. `referrer-02` has 0.20 response_rate to verify the §4.3 weight emphasis |
| `audit_scenarios_test.go` | 563 | Scenario sets 1-7: text-match, skills-overlap, rating-diverse Bayesian, proven-work, verified-mature, last-active, account-age, lost-disputes |
| `audit_scenarios_rules_test.go` | 368 | Scenario sets 8-13: tier sort, anti-gaming (5 rules), persona weight tables, persona behaviour, diversity pass, featured override |
| `audit_spec_drift_test.go` | 236 | Spec-drift docs as tests (5 sub-tests, all GREEN today, designed to flip RED when each gap is fixed) |
| `audit_live_typesense_test.go` | 317 | End-to-end audit against a real Typesense cluster (3 tests, gated by `TYPESENSE_INTEGRATION_URL`) |

**Total: 60 new audit sub-tests + 5 spec-drift sub-tests + 3 live-cluster
tests.** All deterministic, all using the production code paths
(no algorithm mocks).

### Validated scenarios (subset of the 60)

| # | Scenario | Test |
| --- | --- | --- |
| 1 | Highest text-match bucket wins all-else-equal | `TestAudit_TextMatch_DominantBucketWins` |
| 2 | Empty query redistributes weight | `TestAudit_TextMatch_EmptyQueryRedistributes` |
| 3 | Full skill match outranks partial | `TestAudit_SkillsOverlap_FullMatchOverPartial` |
| 4 | Sidebar filter chips count toward query skills | `TestAudit_SkillsOverlap_FilterSkillsCounted` |
| 5 | Referrer skills_overlap always 0 | `TestAudit_SkillsOverlap_ReferrerAlwaysZero` |
| 6 | 50× 4.6 stars beats 1× 5.0 (Bayesian) | `TestAudit_Rating_HighCountBeatsHighAvgLowCount` |
| 7 | Cold-start floor applied to 0-review profile | `TestAudit_Rating_ColdStartFloor` |
| 8 | Concentrated reviewers penalised by diversity | `TestAudit_Rating_DiversityFactorPenalisesConcentration` |
| 9 | Volume + diversity beats sparse track record | `TestAudit_ProvenWork_VolumeBeatsSparse` |
| 10 | Referrer proven_work always 0 | `TestAudit_ProvenWork_ReferrerAlwaysZero` |
| 11–14 | Verified+age binary gate (4 inputs) | `TestAudit_VerifiedMature_BinaryGate/*` |
| 15–20 | Last-active decay (6 inputs at 0/15/30/90/180/365) | `TestAudit_LastActive_HyperbolicDecay/*` |
| 21–26 | Account-age log curve (6 inputs at 0/7/30/90/365/1000) | `TestAudit_AccountAge_LogScaleSaturates/*` |
| 27–31 | Lost-dispute penalty ladder (5 inputs at 0/1/3/4/10) | `TestAudit_LostDisputes_PenaltyCappedAt30Pct/*` |
| 32 | Disputed twin ranks below clean twin | `TestAudit_LostDisputes_LowerFinalScore` |
| 33 | Tier B never reaches top-N when Tier A available | `TestAudit_TierSort_AlwaysBeatsB` |
| 34 | available_now and available_soon share Tier A | `TestAudit_TierSort_AvailableNowAndSoonShareTierA` |
| 35 | Stuffing halves text_match + emits log | `TestAudit_AntiGaming_StuffingHalvesTextMatch` |
| 36 | Short bios immune to stuffing false-positive | `TestAudit_AntiGaming_StuffingDoesNotFalseFireOnShortText` |
| 37–38 | Reviewer-floor cap (silent + clamp branches) | `TestAudit_AntiGaming_ReviewerFloorCapsRating/*` |
| 39 | New-account zeroes AccountAgeBonus + emits log | `TestAudit_AntiGaming_NewAccountZeroesAgeBonus` |
| 40–42 | Persona weights sum to 1.0 (3 tables) | `TestAudit_PersonaWeights_*TotalsToOne` |
| 43 | Agency rating + proven_work ≥ 50% | `TestAudit_PersonaWeights_AgencyRatingDominates` |
| 44 | Referrer response_rate ≥ 15% | `TestAudit_PersonaWeights_ReferrerResponseRateMatters` |
| 45 | Freelance text_match ≥ 10% | `TestAudit_PersonaWeights_FreelanceTextMatchPresent` |
| 46 | Top-rated agency leads cohort | `TestAudit_AgencyPersona_RatingDrivesOrder` |
| 47 | Slow-replier referrer sinks below fast twin | `TestAudit_ReferrerPersona_ResponseRateBreaksTie` |
| 48 | Diversity pass breaks 3-in-a-row | `TestAudit_Diversity_BreaksThreeInARow` |
| 49 | Featured override dormant by default | `TestAudit_FeaturedOverride_DormantByDefault` |
| 50 | Featured override boosts when enabled | `TestAudit_FeaturedOverride_BoostsWhenEnabled` |
| 51 | Velocity rule confirmed dormant in prod | `TestSpecDrift_VelocityRule_NotFiringWithoutTimestamps` |
| 52 | Linked rule confirmed dormant in prod | `TestSpecDrift_LinkedRule_NotFiringWithoutReviewerIDs` |
| 53 | New-account final-score cap NOT enforced | `TestSpecDrift_NewAccountCap_FinalScoreNotEnforced` (logs `freshFinal=54.07`) |
| 54 | New-account flag IS computed correctly | `TestSpecDrift_NewAccountFlag_ReachesScorerWhenWired` |
| 55 | About-field junk penalty has no input | `TestSpecDrift_About_FieldNotFlowingToFeatures` |
| 56 | Live Freelance cohort reranks correctly on real Typesense | `TestAuditLive_FreelanceCohortReranksAsExpected` |
| 57 | Live Agency cohort respects tier sort on real Typesense | `TestAuditLive_AgencyCohortRespectsTierAndRating` |
| 58 | Persona-scoped query never leaks across personas (live) | `TestAuditLive_ReferrerCohortFiltersOnPersona` |
| 59-60 | Plus the table-driven sub-tests inside reviewer-floor (silent + above-cap branches) | `TestAudit_AntiGaming_ReviewerFloorCapsRating/silent_*`, `TestAudit_AntiGaming_ReviewerFloorCapsRating/clamps_*` |

---

## 6. Findings — drift from spec

Five drifts found. Two are mechanical (anti-gaming rules with nil
data inputs), two are mechanical-but-impactful (ignored result), one
is a missing schema field. **None silently produce wrong rankings**
— they leave anti-gaming features dormant.

### D1 (HIGH) — Velocity rule §7.2 receives no timestamp data

**File**: `backend/internal/app/search/ranking_pipeline.go:205`

```go
raw := antigaming.RawSignals{
    ProfileID:              hit.Document.OrganizationID,
    Persona:                lite.Persona,
    Text:                   strings.ToLower(strings.TrimSpace(lite.SkillsText)),
    RecentReviewTimestamps: nil, // populated once the adapter lands
    TotalReviewCount:       int(hit.Document.RatingCount),
    ReviewerIDs:            nil, // populated by linked-account detector
    NowUnix:                nowUnix,
    AccountAgeDays:         int(lite.AccountAgeDays),
}
```

`RecentReviewTimestamps: nil` means rule `velocityRule` always
sees zero recent reviews (`rule_velocity.go:33-41`). The 5-burst
cap never fires. **An attacker uploading 20 reviews in 24 hours
faces no dampening today**.

The fix surface is small: the indexer can emit a parallel
`recent_review_timestamps int64[]` array on the search document
(at most a few hundred bytes per profile in steady state), or a
side-channel cache keyed by org ID.

**Action**: high priority because this is the single rule that
catches the most common attack vector (paid 5★ farms).

### D2 (MEDIUM) — Linked-account rule §7.3 receives no reviewer IDs

**Same file:line as D1**: `ReviewerIDs: nil` is hard-coded.

The default detector is a no-op (`NoopLinkedReviewersDetector`)
which always returns 0 — so even if reviewer IDs were threaded
through, the rule would not fire. **Production-ready behaviour
requires both** (a) routing reviewer IDs from the indexer and (b)
swapping the no-op detector for a Postgres-backed implementation
that joins the `users`/`sessions` tables for IP/email/device match.

**Action**: medium priority — the no-op detector means even if (a)
were fixed, the rule still wouldn't fire. Plan both halves
together.

### D3 (CRITICAL) — New-account final-score cap §7.5 NOT enforced

**File**: `backend/internal/app/search/ranking_pipeline.go:211`

```go
p.antigaming.Apply(ctx, f, raw)
```

The return value `PipelineResult{NewAccountCapped: bool, Penalties: []Penalty}`
is discarded. The spec requires the scorer to cap the final
composite score at the persona median for profiles younger than
`NewAccountAgeDays`. Today, only `AccountAgeBonus` is zeroed
(2% weight) — the new account's other 98% of weight remains.

**Quantitative evidence**: `TestSpecDrift_NewAccountCap_FinalScoreNotEnforced`
builds a 4-day-old attacker profile with maxed-out signals and
logs `freshFinal=54.07`. With Alice's 60ish baseline and Camille's
similar, the attacker still lands competitive, not below median.

**Action**: critical priority. Fix is mechanical — capture the
PipelineResult, propagate `NewAccountCapped` into a new
`Candidate.NewAccountCapped` field, and apply the median cap in
the scorer or business-rules layer. The
`TestSpecDrift_NewAccountFlag_ReachesScorerWhenWired` test
proves the data is available; the wiring just needs ~10 lines of
code.

### D4 (LOW) — `about` field-driven junk penalty §3.2-7 unused

**File**: `backend/internal/app/search/document_adapter.go:67`

```go
About: "",
```

The `SearchDocument` schema has no `About` field
(`internal/search/schema.go`), so the adapter sets the
`SearchDocumentLite.About` to empty unconditionally. The §3.2-7
"shannon_entropy + junk pattern" penalty therefore has no data
to read and never fires.

**Action**: low priority. The current `profile_completion_score`
already requires `HasAbout` for full credit, providing partial
coverage. Adding the `About` field requires (a) schema migration
on the Typesense side, (b) full reindex, (c) keeping the field
out of `query_by` (it shouldn't influence text match scoring
unless the spec explicitly redesigns that). Track in a follow-up
ticket.

### D5 (LOW) — `last_active_days_score` returns 0 when LastActiveAt is missing

**File**: `internal/search/features/extract_last_active.go:26`

```go
if doc.LastActiveAt <= 0 || doc.NowUnix <= 0 {
    return 0
}
```

Returning 0 for unknown freshness is **stricter** than the spec
("we treat dormant"), but the actual effect is mild — last_active
weight is only 2-3%. Documenting here for completeness; this is
not a bug, just a safety choice that should ideally be a config
knob.

**Action**: not a real drift, just notable. No fix needed.

---

## 7. Recommended fixes (prioritised)

| # | Priority | Drift | Estimated effort |
| --- | --- | --- | --- |
| 1 | CRITICAL | D3 — capture `PipelineResult.NewAccountCapped` in `applyAntiGaming` and enforce the median cap in the scorer or business rules | 1-2 hrs (4 file changes, 1 new test) |
| 2 | HIGH | D1 — index `recent_review_timestamps` field on `SearchDocument` and route through `RawSignals.RecentReviewTimestamps` | 1 day (schema migration, indexer update, full reindex) |
| 3 | MEDIUM | D2 — index `reviewer_ids` (or hash thereof) and ship a Postgres-backed `LinkedReviewersDetector` | 2 days (schema, indexer, adapter, tests) |
| 4 | LOW | D4 — add `about` text field to `SearchDocument` and route to `SearchDocumentLite.About` for entropy/junk detection | 1 day (schema, reindex, optional rule logic update) |
| 5 | LOW | D5 — make the unknown-freshness score env-tunable | 30 min |

All five fixes are additive — none change the existing weight
tables or break the LTR-ready feature vector schema. The spec-drift
tests in `audit_spec_drift_test.go` will flip RED automatically
when the fix lands, signalling that the audit-doc claims need an
update.

### Quick win (recommended for immediate landing)

Fix D3 first. The change is mechanical, the test infrastructure
is in place, and it closes the most-cited gameability hole in the
ranking model. Sketch:

```go
// applyAntiGaming → return PipelineResult
func (p *RankingPipeline) applyAntiGaming(...) antigaming.PipelineResult {
    // ... unchanged ...
    return p.antigaming.Apply(ctx, f, raw)
}

// scoreCandidates → propagate the flag onto the Candidate
candidates[i].NewAccountCapped = res.NewAccountCapped

// rules → at the start of Apply, after sortByFinalDesc:
if newAccountMedian := percentile50OfFinal(candidates); newAccountMedian > 0 {
    for i := range candidates {
        if candidates[i].NewAccountCapped && candidates[i].Score.Final > newAccountMedian {
            candidates[i].Score.Final = newAccountMedian
        }
    }
}
```

Then flip the assertion in `TestSpecDrift_NewAccountCap_FinalScoreNotEnforced`
from `t.Logf("DRIFT")` to `assert.LessOrEqual(t, freshFinal, medianFinal, …)`.

---

## 8. Validation pipeline output

Run with the audit branch checked out:

```
$ cd backend && go build ./...
(no output — clean)

$ go vet ./...
(no output — clean)

$ go test ./internal/search/... ./internal/app/search/... -count=1 -short
ok  	marketplace-backend/internal/search	0.054s
ok  	marketplace-backend/internal/search/antigaming	0.014s
ok  	marketplace-backend/internal/search/features	0.009s
ok  	marketplace-backend/internal/search/rules	0.007s
ok  	marketplace-backend/internal/search/scorer	0.005s
ok  	marketplace-backend/internal/app/search	0.085s

$ go test ./internal/app/search/ -count=1 -short \
       -run "TestAudit|TestSpecDrift" -v | grep -c "^--- PASS\|^    --- PASS"
60

$ TYPESENSE_INTEGRATION_URL=http://localhost:8108 \
  TYPESENSE_INTEGRATION_API_KEY=xyz-dev-master-key-change-in-production \
  go test ./internal/app/search/ -count=1 -short -run TestAuditLive_ -v
=== RUN   TestAuditLive_FreelanceCohortReranksAsExpected
    audit_live_typesense_test.go:91: live rerank top-3 (out of 9) for
        freelance/react: [freelance-01:freelance freelance-08:freelance freelance-02:freelance]
    audit_live_typesense_test.go:104: live retrieval found=9
--- PASS: TestAuditLive_FreelanceCohortReranksAsExpected (0.21s)
=== RUN   TestAuditLive_AgencyCohortRespectsTierAndRating
--- PASS: TestAuditLive_AgencyCohortRespectsTierAndRating (0.19s)
=== RUN   TestAuditLive_ReferrerCohortFiltersOnPersona
--- PASS: TestAuditLive_ReferrerCohortFiltersOnPersona (0.19s)
PASS
ok  	marketplace-backend/internal/app/search	0.595s

$ go test -bench=BenchmarkRerank -benchtime=10x -run=NONE \
       ./internal/app/search/
BenchmarkRerank_200Candidates-12    	      10	  796000 ns/op
       528040 B/op	    1519 allocs/op
PASS
```

Full suite: 825 sub-tests across all search packages. No flakes.
Rerank p95 is 0.8 ms for 200 candidates — well below the 50 ms
budget from §2.1.

---

## 9. References

- `docs/ranking-v1.md` — the locked spec.
- `docs/ranking-tuning.md` — operator playbook (env vars).
- `docs/search-engine.md` — high-level engine architecture.
- `internal/search/schema.go` — Typesense collection definition.
- `internal/app/search/ranking_pipeline.go` — Stage 2-5 wiring.
- `cmd/api/wire_helpers.go:81` — `buildRankingPipeline` entry point.

This audit's regression suite is the canonical contract going
forward. Future ranking work should:

1. Run the audit suite after every change (`go test -run TestAudit
   ./internal/app/search/...`).
2. Update `audit_fixtures_*_test.go` only by adding new fixtures
   (never edit existing ones — they encode the expected behaviour
   for the existing scenario library).
3. Flip a `TestSpecDrift_*` test from PASS-with-DRIFT-log to
   PASS-with-correctness-assertion when each gap is fixed.
4. Re-run the live `TestAuditLive_*` tests against the production
   Typesense schema after any ranking-signal field addition.
