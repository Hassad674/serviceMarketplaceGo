# Ranking V1 — Specification

> Open-source marketplace ranking engine, built with a "hard signals first" philosophy,
> designed to be LTR-ready and gameable-by-design-not-by-accident.

**Status** : specification locked 2026-04-17. Pending implementation across 3 agent rounds.
**Scope** : ranking of search results on the three public listings (freelance / agency / referrer).
**Non-goals** : personalisation (requires user interaction data we don't have yet), ML-based ranking (deferred to V2 once click/conversion data exists).

---

## 1. Mission & principles

### 1.1 What we're building

A **5-stage server-side ranking pipeline** that takes the top-200 candidates returned by Typesense's hybrid retrieval and produces a fair, explainable, persona-aware top-20. The pipeline runs on every `/api/v1/search` request and feeds the three public listing pages (`/fr/freelancers`, `/fr/agencies`, `/fr/referrers`).

### 1.2 Design principles

| Principle | What it means concretely |
| --- | --- |
| **Hard signals first** | Features costly to fake (real paid projects, verified KYC, real conversations) get more weight than features editable from a profile form (bio text, free-form skills). |
| **Open-source-safe** | The algorithm is public; the *weights* and *anti-gaming thresholds* are private (env vars). Attackers can read the formula but not the exact numbers — mirrors Stripe's fraud-detection stance. |
| **LTR-ready** | Every ranked result is logged with its full feature vector + user-outcome events (click, message, hire). In 3-6 months of real traffic the captured data trains a LambdaMART model without re-architecture. |
| **Explainable by design** | No opaque ML in V1. The score is a weighted sum a human can audit. `Matching score: 87` can be shown to users; the internals are understood by the team. |
| **Persona-aware** | A freelance, an agency, and a business referrer don't play the same game. Weights and features differ per persona, computed from the same extractor module. |
| **Cold-start protected** | A brand-new profile with zero reviews isn't ranked into the void. Floor values + Rising Talent slot give newcomers a fair shot without rewarding them for nothing. |
| **Anti-gaming from day 1** | 5 detection rules run before scoring. Caps and penalties are applied silently; attackers get no error message that would let them probe thresholds. |

### 1.3 Comparable systems

| System | What we learn |
| --- | --- |
| **Malt** | Matching Score UI — visible to clients, never reveals internal weights. Hand-tuned V1, LTR (LambdaMART) V5. |
| **Upwork Job Success Score (JSS)** | Rolling 6-month window of feedback + completion + disputes. Strong negative signals for disputes and refunds. |
| **Contra** | Portfolio-first, lighter algorithmic ranking, Rising Talent curation. |
| **Stripe fraud / Uber ETA** | Algorithm public, parameters private. Anti-gaming in env-var thresholds. |

---

## 2. Architecture — 5-stage pipeline

```
User query (q, filters, persona)
          │
          ▼
┌───────────────────────────────────────────────────────────────┐
│ Stage 1 — CANDIDATE RETRIEVAL                                 │
│   Typesense hybrid (BM25 + vector cosine via OpenAI embedding)│
│   filter_by: persona + is_published                           │
│   sort_by: _text_match:desc (tiebreak _vector_distance)       │
│   Returns top-200 raw docs                                    │
└───────────────────────┬───────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────────────────────┐
│ Stage 2 — FEATURE EXTRACTION                                  │
│   internal/search/features/ (pure functions, no I/O)          │
│   Extract(query, doc) → Features{10 fields ∈ [0,1]}           │
│   Per-persona extractors (freelance, agency, referrer)        │
└───────────────────────┬───────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────────────────────┐
│ Stage 3 — ANTI-GAMING PENALTIES                               │
│   internal/search/antigaming/                                 │
│   5 rules: stuffing, velocity, linked, reviewer_floor, age    │
│   Applies multiplicative caps on specific features in-place   │
│   Logs each applied penalty for tuning                        │
└───────────────────────┬───────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────────────────────┐
│ Stage 4 — COMPOSITE SCORING                                   │
│   internal/search/scorer/                                     │
│   Reranker interface; WeightedScorer V1 impl                  │
│   positive = Σ wᵢ × featureᵢ (per-persona weights)            │
│   adjusted = positive × (1 − negative_signals)                │
│   score = adjusted × 100 ∈ [0, 100]                           │
└───────────────────────┬───────────────────────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────────────────────┐
│ Stage 5 — BUSINESS RULES                                      │
│   internal/search/rules/                                      │
│   1. Tier sort (available_now/soon above not_available)       │
│   2. Diversity (no 2 consecutive same primary skill)          │
│   3. Rising Talent slot (1 per 5 in top-20)                   │
│   4. Gaussian randomization (σ rank-dependent)                │
└───────────────────────┬───────────────────────────────────────┘
                        │
                        ▼
                  Top-20 rendered
```

### 2.1 Performance targets

| Stage | Target (p95) |
| --- | --- |
| Retrieval (Stage 1) | < 80 ms (Typesense owns this) |
| Re-ranking (Stages 2-5) | < 50 ms (pure Go, in-memory on 200 docs × 10 features) |
| Total search latency | < 150 ms |

Measurement : `scripts/perf/k6-search.js` + structured log duration_ms. Baseline tracked in `docs/perf/baseline.json`.

---

## 3. The 10 features

Each feature is a pure function of `(query, doc)` returning a scalar in **`[0, 1]`**. Normalisation is baked in so the weighted sum is comparable across persona and across time.

### 3.1 Summary table

| # | Feature | Range | Source | Game-resistance |
| --- | --- | :---: | --- | :---: |
| 1 | `text_match_score` | [0, 1] | Typesense BM25 bucket ÷ 10 | 🟠 soft |
| 2 | `skills_overlap_ratio` | [0, 1] | `|query_skills ∩ profile_skills| ÷ |query_skills|` | 🟠 soft |
| 3 | `rating_score_diverse` | [0, 1] | Bayesian × diversity × recency | 🟠 medium (anti-gaming) |
| 4 | `proven_work_score` | [0, 1] | projects + unique_clients + repeat_rate composite | 🟢 hard |
| 5 | `response_rate` | [0, 1] | messaging signal, directly from DB | 🟢 hard |
| 6 | `is_verified_mature` | {0, 1} | `is_verified AND account_age_days ≥ 30` | 🟢 hard |
| 7 | `profile_completion` | [0, 1] | 0-100 score ÷ 100 with anti-junk penalty | 🟠 soft (anti-junk) |
| 8 | `last_active_days_score` | [0, 1] | hyperbolic decay `1 / (1 + days/30)` | 🟠 soft (capped) |
| 9 | `account_age_bonus` | [0, 1] | log-scaled, capped at 1 year | 🟢 hard |
| 10 | `skills_overlap_ratio` + `expertise_match_binary` | domain-specific | replaces skills for referrers (0-weight freelance/agency) | 🟠 |

### 3.2 Per-feature deep dive

#### (1) `text_match_score`

Typesense returns `_text_match` as a bucketed score in `[0, 10]` (we request `buckets:10` explicitly). We normalise and apply the keyword-stuffing penalty :

```
raw = min(10, typesense_text_match_bucket) / 10
text_match_score = raw × (stuffing_detected ? STUFFING_PENALTY : 1.0)
```

Defaults : `STUFFING_PENALTY = 0.5` (env `RANKING_STUFFING_PENALTY`).
Stuffing detection lives in `internal/search/antigaming/stuffing.go` (details in §7).

**Edge case — empty query** : when the user arrives on a listing without typing anything, Typesense runs `q=*`. `text_match_score` is forced to `0`, and its 15–20% weight is redistributed proportionally across all other features for that single query. See §5.2.

#### (2) `skills_overlap_ratio`

```
query_skills  = tokenize(query_text) ∪ filter.skills          // union of typed tokens and the sidebar skill chips
profile_skills = doc.skills                                    // canonicalised, lowercased
overlap        = |query_skills ∩ profile_skills|
skills_overlap_ratio = (|query_skills| == 0) ? 0 : overlap / |query_skills|
```

For **referrers**, skills don't apply (they don't sell skills, they sell network access). Weight is 0% in the referrer table — the feature is still computed but weighted out of the composite.

#### (3) `rating_score_diverse`

The most elaborate feature. Combines three signals : Bayesian shrinkage, reviewer diversity, temporal recency.

**Step 1 — Bayesian average** (shrinks low-count averages toward the marketplace mean) :

```
rating_bayesian = (C × m + Σ ratingᵢ) / (C + n)

where n = number of reviews
      m = 4.0 (marketplace mean, env RANKING_BAYESIAN_PRIOR_MEAN)
      C = 8   (prior weight, env RANKING_BAYESIAN_PRIOR_WEIGHT)
```

Literature : Bayesian average for ratings is standard in e-commerce ranking; `C` between 5 and 15 is typical.

**Step 2 — Diversity factor** (kills "3 friends leave 10 reviews") :

```
max_reviewer_share = max(count_per_reviewer) / n
diversity_factor   = 1 - max_reviewer_share
effective_count    = unique_reviewers × diversity_factor
```

Example attack : a profile has 10 reviews, 8 from the same reviewer. `max_share = 0.8`, `diversity_factor = 0.2`, `effective_count = 2 × 0.2 = 0.4`. The rating component tanks.

**Step 3 — Recency weighting** (recent reviews count more than 2-year-old) :

```
recency_weight = Σ exp(-age_daysᵢ / 365)   // for each of the n reviews
recency_factor = recency_weight / n         // mean of the exp(-decay) terms
```

A review from today contributes 1.0; one from 365 days ago contributes 1/e ≈ 0.37; one from 730 days ≈ 0.14.

**Step 4 — Composite with cold-start floor** :

```
if n == 0:
    rating_score_diverse = COLD_START_FLOOR          // default 0.15
else:
    normalised_bayesian  = rating_bayesian / 5
    count_component      = log(1 + effective_count) / log(1 + 50)
    rating_score_diverse = normalised_bayesian × count_component × recency_factor
```

Ceiling : the count component caps at `log(51) ≈ 3.93`, so very active profiles (>50 unique reviewers) get near-full value. `COLD_START_FLOOR` is `0.15` (env `RANKING_COLD_START_FLOOR`) — a newcomer is worth slightly less than someone with one mediocre review.

#### (4) `proven_work_score`

Captures "this profile has a real track record", robust against the volume-vs-size tradeoff (a senior with 5 big missions isn't punished vs a junior with 50 small ones).

```
raw =
    0.40 × log(1 + completed_projects)                          // volume, log-scaled
  + 0.35 × log(1 + unique_clients)                              // diversity of clients
  + 0.25 × sqrt(repeat_client_rate)                             // quality: clients return

proven_work_score = min(1.0, raw / log(1 + 100))
```

`log(1 + 100) ≈ 4.62` normalises a very active pro (100 projects) to ≈ 1.0. Beyond that, diminishing returns.

Worked example :

| Profile | completed | unique_clients | repeat_rate | proven_work_score |
| --- | :---: | :---: | :---: | :---: |
| Senior · 5 big missions, 5 clients, 20% repeat | 5 | 5 | 0.20 | 0.40 |
| Junior · 50 small missions, 30 clients, 15% | 50 | 30 | 0.15 | 0.78 |
| Attack : 30 missions, 2 client (farming) | 30 | 2 | 1.00 | 0.55 |
| Senior · 10 missions, 7 clients, 40% repeat | 10 | 7 | 0.40 | 0.50 |

Applied to the referrer persona, this feature is absent (weight 0%) because referrers don't complete projects themselves. The `referral_conversion_score` that would have replaced it is **dropped from V1** (no attribution tracking infrastructure yet — see §13).

#### (5) `response_rate`

Direct `[0, 1]` signal from the messaging table: fraction of incoming messages the profile replies to within 24 hours over a rolling 90-day window. Computed at index time, not per-query.

No log transform, no amplification — the raw rate is the honest signal. A response rate of 0.80 is 80% of the theoretical max, which is exactly what we want to weigh.

Anti-gaming : if the rate dropped sharply in the last 14 days (> 30 pp drop), a cooldown penalty applies (see §7).

#### (6) `is_verified_mature`

Binary signal : `1` iff `is_verified = true AND account_age_days ≥ 30`. Combines KYC verification with account maturity to prevent fresh accounts that instantly pass KYC from gaming the "verified" boost.

Why 30 days ? Long enough to deter disposable accounts; short enough not to punish genuine newcomers.

#### (7) `profile_completion`

```
base = profile_completion_score / 100                   // from indexer, fields: photo, about, video, pricing, skills, langs, social, expertise
if shannon_entropy(profile.about) < ENTROPY_MIN || contains_junk_patterns(profile.about):
    base *= 0.3                                         // anti-junk penalty
profile_completion = base
```

`ENTROPY_MIN` default `3.5` (env `RANKING_ENTROPY_MIN`). Junk patterns include repeated lorem-ipsum, `.......`, or text that is purely emoji / punctuation.

#### (8) `last_active_days_score`

Hyperbolic decay :

```
last_active_days_score = 1 / (1 + last_active_days / 30)
```

| days since last active | score |
| :---: | :---: |
| 0 | 1.00 |
| 15 | 0.67 |
| 30 | 0.50 |
| 90 | 0.25 |
| 180 | 0.14 |
| 365 | 0.08 |

Capped at 1.0 (no benefit to activity within the same day). Encourages real engagement without letting a "log in every day" cron fully game the signal (the weight is small — 2-3% — and the cron doesn't help other features).

#### (9) `account_age_bonus`

Log-scaled maturity bonus :

```
account_age_bonus = min(1.0, log(1 + account_age_days) / log(1 + 365))
```

| account_age_days | bonus |
| :---: | :---: |
| 0 | 0 |
| 7 | 0.35 |
| 30 | 0.58 |
| 90 | 0.77 |
| 365 | 1.00 |
| > 365 | 1.00 (capped) |

Beyond 1 year the bonus saturates. Very small weight (1–2 %) — just enough to slightly favour established identities over fresh accounts, not enough to become a "veterans win" signal.

#### (10) Summary / redundancy

The 10 features cover the three key axes :

- **Relevance** (what the query asks) → features 1, 2
- **Quality & track record** (the profile's history) → features 3, 4, 5, 6
- **Maturity & freshness** (account hygiene) → features 7, 8, 9

No feature is fully redundant with another. `rating_score_diverse` and `proven_work_score` both measure "reputation" but from orthogonal angles (customer feedback vs. economic history).

---

## 4. Per-persona weight tables

### 4.1 Freelance — `RANKING_WEIGHTS_FREELANCE_*`

| Feature | Weight | Rationale |
| --- | :---: | --- |
| `text_match_score` | **20%** | A freelance hired on text-heavy queries (tech stack, role title). |
| `skills_overlap_ratio` | **15%** | Explicit skill match is a strong positive for task-fit. |
| `rating_score_diverse` | **20%** | Reviews are the primary social proof for solo operators. |
| `proven_work_score` | **15%** | Track record matters but less than for agencies. |
| `response_rate` | **10%** | A freelance who doesn't reply in 24h is effectively unhireable. |
| `is_verified_mature` | **8%** | Trust signal; combined with account age to deter drive-by accounts. |
| `profile_completion` | **7%** | Well-filled profile signals seriousness. |
| `last_active_days_score` | **3%** | Freshness nudge; small weight. |
| `account_age_bonus` | **2%** | Tiny maturity boost. |

**Total positive = 100 %**. Negative penalty up to −30 %.

### 4.2 Agency — `RANKING_WEIGHTS_AGENCY_*`

| Feature | Weight | Rationale |
| --- | :---: | --- |
| `text_match_score` | **15%** | Agencies positioned more on brand/portfolio than on keyword density. |
| `skills_overlap_ratio` | **10%** | Broader skill coverage; less dispositive. |
| `rating_score_diverse` | **25%** | Track record is paramount for B2B engagements. |
| `proven_work_score` | **25%** | Portfolio of shipped projects is the core agency signal. |
| `response_rate` | **5%** | Agencies structurally reply slower (sales cycle). |
| `is_verified_mature` | **10%** | KYC + age matters more — clients engage bigger contracts. |
| `profile_completion` | **7%** | Well-maintained profile is baseline professionalism. |
| `last_active_days_score` | **2%** | Tiny. |
| `account_age_bonus` | **1%** | Even smaller; agencies usually survive longer than freelances. |

**Total = 100 %**.

### 4.3 Referrer (apporteur d'affaires) — `RANKING_WEIGHTS_REFERRER_*`

| Feature | Weight | Rationale |
| --- | :---: | --- |
| `text_match_score` | **20%** | Query match against bio + sectors. |
| `skills_overlap_ratio` | **0%** | Referrers don't sell skills. |
| `rating_score_diverse` | **35%** | Partner feedback is THE signal : apporteurs live on their reputation with the freelances and agencies they refer. |
| `proven_work_score` | **0%** | Doesn't apply (referrers don't complete projects themselves). |
| `response_rate` | **20%** | Absolutely critical — an unresponsive referrer is useless. |
| `is_verified_mature` | **10%** | Trust is everything for a commission-based role. |
| `profile_completion` | **10%** | Sector claims + network description must be detailed. |
| `last_active_days_score` | **3%** | Slightly more than for other personas (stale referrers lose network value). |
| `account_age_bonus` | **2%** | |

**Total = 100 %**. Note the absence of `proven_work_score` and `skills_overlap_ratio` — they're replaced by higher weights on rating and responsiveness.

### 4.4 Why not a single weight table ?

A single table would either under-weight `proven_work_score` for agencies (who depend on portfolio), or break for referrers (who have no projects to prove). Per-persona is the minimum complexity that correctly captures the three distinct economic models.

---

## 5. Composite scoring formula

### 5.1 Positive composite

```
for each feature i:
    contribution = weight_persona[i] × feature[i]

positive_score = Σ contribution                         // ∈ [0, 1]
```

### 5.2 Empty-query redistribution

When `query.trim() == ""`, `text_match_score = 0` by construction, which would cost the profile its `text_match` weight (15–20 %). Instead, we redistribute :

```
if query is empty:
    missing_weight   = weight_persona[text_match_score]          // e.g. 0.20 for freelance
    scale            = 1 / (1 - missing_weight)                  // e.g. 1.25
    for each feature i != text_match_score:
        effective_weight[i] = weight_persona[i] × scale
    (text_match weight becomes 0, all others scale up to sum to 100 %)
```

This preserves the relative weight of each remaining feature without the arbitrary choice of which feature to "give" the text-match slice to.

### 5.3 Negative penalty

```
lost_disputes_count = count(disputes where outcome ∈ {refund_full, refund_partial} AND respondent = profile)
negative_signals    = min(0.30, lost_disputes_count × 0.10)

adjusted_score = positive_score × (1 − negative_signals)
```

Loss-of-dispute penalty is the **only** negative signal kept for V1. Cancellations, ghosting, and late deliveries are too noisy / ambiguous (a freelance who cancels early is arguably better than one who delivers badly). Response ghosting is already captured in the `response_rate` positive feature — no double penalty.

### 5.4 Final display score

```
base_score   = adjusted_score × 100                             // ∈ [0, 100]
noise        = gaussian(mean=0, σ=σ(base_score, rank_position)) // see §6
final_score  = clamp(base_score + noise, 0, 100)
```

The display score rendered in the UI is `round(final_score)` — an integer 0-100 users can understand.

---

## 6. Randomisation & fairness

### 6.1 Gaussian noise, rank-dependent σ

To avoid the "winner-takes-all" attractor (profile #1 always stays #1 → monopolises clicks → reinforces position 1) and to make the ranking slightly non-deterministic for gaming :

```
σ(score, rank) = NOISE_COEFFICIENT × score × rank_multiplier(rank)

where NOISE_COEFFICIENT = 0.006   (env RANKING_NOISE_COEFFICIENT)
      rank_multiplier(rank):
          if rank ≤ 3:   return 0.3             // top-3: very stable
          elif rank ≤ 10:return 0.8             // mid: moderate rotation
          else:          return 1.5             // tail: larger rotation
```

Concrete example on a score of 80 :

| rank | σ |
| :---: | :---: |
| 1 | 0.14 |
| 5 | 0.38 |
| 15 | 0.72 |

Top-3 positions swap rarely between page loads. Positions 11–20 rotate more — giving every mid-tier candidate a chance to be seen above the fold.

### 6.2 Cold-start floor

Already embedded in `rating_score_diverse` (§3.2-3). A newcomer with zero reviews receives a score of `0.15` on that component instead of `0`, preserving ~20 % of the rating dimension for new profiles rather than tanking them entirely.

### 6.3 Rising Talent slot

**1 of every 5 slots** in the top-20 is reserved for a Rising Talent candidate — a new profile with decent signals that would otherwise be crowded out by veterans.

Criteria for eligibility :
- `account_age_days < 60`
- `is_verified = true` (prevents gaming via fresh unverified accounts)
- `final_score ≥ median(final_score of all ranked candidates)`

Algorithm :
1. After sorting by `final_score`, identify the "rising" sub-list.
2. In the top-20, reserve positions 5, 10, 15, 20 for a rising candidate if one exists with a score ≥ the incumbent at that position − 5.
3. If no rising candidate fits, the slot is filled normally.

This is a business-rule-level mechanism, not a weighted feature, because it's categorical ("slot or no slot") and should remain transparent to the tuning dashboard.

### 6.4 Tier sort

Availability is not ranked, it's **tiered** :

```
Tier A:   availability_status ∈ {available_now, available_soon}
Tier B:   availability_status = not_available
```

Within each tier, the composite score (with noise) determines ordering. Tier A is rendered first, then Tier B. A tier-B profile can never appear above a tier-A profile, regardless of score.

This is cleaner than weighting availability : users who filter for "available now" get zero mismatches, and users who don't filter see a clear block separation.

### 6.5 Diversity rule

To prevent 10 consecutive React developers dominating the grid :

```
For each of the top-20 positions in order:
    if the current candidate shares a primary_skill with the previous candidate AND with the one before:
        try to swap with the next-best candidate that breaks the run
        if no alternative is available, keep the run (rare case)
```

This is a soft rule — a 3-in-a-row is broken, but a 2-in-a-row is acceptable. Primary skill is the first skill listed on the profile.

---

## 7. Anti-gaming pipeline

Five detection rules run between feature extraction and scoring. Each applies a silent cap or penalty. No user-visible error; attackers don't learn their specific threshold.

All thresholds in env vars (`RANKING_AG_*`), with safe published defaults.

### 7.1 Rule 1 — Keyword stuffing

**Attack** : stuff `skills_text` or `about` with 50 repetitions of "React React React" to inflate BM25.

**Detection** :
```
tokens = tokenize(skills_text ∪ about)
token_counts = count(tokens)
max_repetition = max(token_counts.values())
distinct_ratio = len(distinct(tokens)) / len(tokens)

if max_repetition > 5 OR distinct_ratio < 0.3:
    stuffing_detected = true
    text_match_score *= STUFFING_PENALTY   // default 0.5
```

Env : `RANKING_AG_MAX_TOKEN_REPETITION = 5`, `RANKING_AG_MIN_DISTINCT_RATIO = 0.3`, `RANKING_STUFFING_PENALTY = 0.5`.

### 7.2 Rule 2 — Review velocity cap

**Attack** : 10 fake 5★ reviews uploaded in the same afternoon to quickly boost `rating_score_diverse`.

**Detection** :
```
recent_reviews = reviews where created_at > now - 24h
if len(recent_reviews) > VELOCITY_CAP:       // default 5
    excess = len(recent_reviews) - VELOCITY_CAP
    # reduce the effective review count by the excess
    n_effective = n - excess
    rating_score_diverse is recomputed using n_effective
```

A cooldown period of 14 days keeps the cap active for affected profiles, even after the spike subsides.

Env : `RANKING_AG_VELOCITY_CAP_24H = 5`, `RANKING_AG_VELOCITY_COOLDOWN_DAYS = 14`.

### 7.3 Rule 3 — Linked-account discount

**Attack** : create 5 fake reviewer accounts all from the same IP / email domain, have them leave 5★ reviews.

**Detection** :
```
for each review:
    reviewer = lookup_user(review.reviewer_id)
    if any(other_reviewer shares IP OR email_domain OR device_id with reviewer):
        mark_as_linked(review)

linked_fraction = count(linked_reviews) / n
if linked_fraction > LINKED_MAX:            // default 0.3
    reduce rating_count effectively by linked count
```

Requires access to user metadata (IP, email, device fingerprint) — implemented via a query to the `users` + `sessions` tables. Respects GDPR (no raw IP exposure; only hashed comparison).

Env : `RANKING_AG_LINKED_MAX_FRACTION = 0.3`.

### 7.4 Rule 4 — Unique reviewer floor

**Attack** : have 1 friend leave 20 reviews at different dates to simulate a popular profile.

**Detection** :
```
if unique_reviewers < REVIEWER_FLOOR:       // default 3
    # Rating count is already dampened by diversity_factor in §3.2-3
    # This rule is an additional hard floor — if < 3 unique reviewers,
    # rating_score_diverse is capped at 0.4 regardless.
    rating_score_diverse = min(rating_score_diverse, HARD_CAP)  // default 0.4
```

Env : `RANKING_AG_UNIQUE_REVIEWER_FLOOR = 3`, `RANKING_AG_FEW_REVIEWER_CAP = 0.4`.

### 7.5 Rule 5 — New account score cap

**Attack** : create a new account, instantly fill it with fake data, spam reviews from linked accounts before any check catches up.

**Detection** :
```
if account_age_days < NEW_ACCOUNT_AGE:      // default 7
    # Cap the final composite score at the persona's median score
    # (computed on a rolling 7-day window)
    final_score = min(final_score, persona_median)
```

A profile younger than 7 days can at best rank at the median — not in the top. After 7 days, if the profile legitimately earns its position, the cap lifts.

Env : `RANKING_AG_NEW_ACCOUNT_AGE_DAYS = 7`.

### 7.6 Penalty logging

Every applied penalty emits a structured log line :

```json
{
  "event": "ranking.penalty_applied",
  "rule": "keyword_stuffing",
  "profile_id": "...",
  "persona": "freelance",
  "detection_value": 0.18,
  "threshold": 0.30,
  "penalty_factor": 0.5,
  "query_id": "..."
}
```

Logs feed an admin-only dashboard (future — not in V1 scope). For V1, logs are grep-able via `slog` JSON output.

---

## 8. Business rules layer

Applied in order on the sorted candidates :

1. **Negative cap** — each profile's `final_score` cannot exceed a hard cap based on `negative_signals` (already enforced via multiplication in §5.3).
2. **Tier sort** (§6.4) — split into Tier A / Tier B.
3. **Randomise** (§6.1) — add per-candidate gaussian noise.
4. **Re-sort** within each tier using the noise-adjusted score.
5. **Diversity pass** (§6.5) — swap adjacents if 3+ in a row share primary skill.
6. **Rising Talent injection** (§6.3) — replace specific slots with an eligible rising candidate.
7. **Featured override** — if `is_featured = true` and admin config enables it (defaults OFF in V1 — not wired).

Output : ordered list of 20 `final_score` + `rank_explanation` tuples handed back to the query service.

---

## 9. LTR-ready infrastructure

V1 is rule-based but every query logs the data needed to train an LTR model in 3-6 months.

### 9.1 Search query logging extension

Migration — extend `search_queries` (already created in phase 1, migration 111) with :

| Column | Type | Purpose |
| --- | --- | --- |
| `result_features_json` | JSONB | For each of the 20 ranked docs: `[{doc_id, rank_position, features: {...}, final_score}]` |
| `result_vector_sha` | TEXT | Hash of the result order, for deduping identical queries |

Payload size : ~20 × 12 numbers per query ≈ 3 KB per query. For 10k queries/day, ~30 MB/day ≈ 11 GB/year — negligible storage.

### 9.2 Outcome events

Already in place (phase 3) :
- `search.click` — written to `search_queries.clicked_doc_id` + `clicked_position` + `clicked_at`
- `/api/v1/search/track` endpoint wired from both web and mobile (beacon)

Additional outcome events captured downstream (wired via event bus) :
- `proposal.created` with `source_search_id` → strong positive signal
- `conversation.started` with `source_search_id` → medium positive signal
- `milestone.released` with `source_search_id` → very strong positive signal (money moved)

In 3-6 months, a training dataset of `(features, outcome)` tuples will be extracted via a one-off export tool (`cmd/search-ltr-export`).

### 9.3 V2 swap path

The `Reranker` interface (see §10.4) is explicitly designed for swap :

```go
type Reranker interface {
    Score(ctx context.Context, q Query, features Features, persona Persona) float64
}

// V1:
type WeightedScorer struct { weights PersonaWeights }
func (s *WeightedScorer) Score(...) float64 { /* linear combination */ }

// V2:
type LTRScorer struct { model *xgboost.Model }
func (s *LTRScorer) Score(...) float64 { /* model.Predict(features) */ }
```

Changing V1 → V2 is a one-line swap in `cmd/api/main.go`. Same features, same pipeline.

---

## 10. Implementation plan

9 phases organised into 3 rounds, dispatched to 5 agents.

### 10.1 Phase map

| Phase | Scope | Depends on |
| --- | --- | --- |
| **6B** | New indexed signals : 10 new `SearchDocument` fields, CTE queries, reindex | — |
| **6A** | Feature extractor module `internal/search/features/` | 6B |
| **6C** | Anti-gaming module `internal/search/antigaming/` | 6A |
| **6D** | `Reranker` interface + `WeightedScorer` impl | 6A (signatures) |
| **6E** | Business rules layer `internal/search/rules/` | 6D |
| **6F** | Wiring into query service | 6A, 6C, 6D, 6E |
| **6G** | Feature vector logging in `search_queries` | 6F |
| **6H** | Golden-test expansion (14 → 40+) + live OpenAI validation | 6F |
| **6I** | Docs : `docs/search-engine.md` ranking chapter + `docs/ranking-tuning.md` (internal) | Everything |

### 10.2 Round structure

**Round 1 — Foundation (solo agent)**
- **Agent R1-B6** : Phase 6B. Adds 10 new indexed signals + reindex. ~2 days.

**Round 2 — Parallel (3 agents)**
- **Agent R2-F** : 6A + 6C (feature extractors + anti-gaming).
- **Agent R2-S** : 6D (scorer + weights).
- **Agent R2-R** : 6E + 6G + 6I (business rules + logging + docs).

**Round 3 — Integration (solo agent)**
- **Agent R3-W** : 6F + 6H (wiring + golden validation).

### 10.3 Validation gates per round

Each agent must, before merging to main :
1. `go build ./...` clean.
2. `go vet ./...` clean.
3. `go test ./... -count=1 -race` green (≥ 90 % coverage on new packages).
4. `npx tsc --noEmit` clean (if any TS touched).
5. For R3-W only : the 14 existing golden tests + the 30 new tests all pass with live OpenAI.
6. Performance benchmark : `go test -bench=. ./internal/search/...` shows re-ranking p95 < 50 ms on 200 candidates.

### 10.4 Interfaces / contracts

**`Features`** (struct, in `internal/search/features/types.go`) :
```go
type Features struct {
    TextMatchScore        float64  // [0, 1]
    SkillsOverlapRatio    float64  // [0, 1]
    RatingScoreDiverse    float64  // [0, 1]
    ProvenWorkScore       float64  // [0, 1]
    ResponseRate          float64  // [0, 1]
    IsVerifiedMature      float64  // {0, 1}
    ProfileCompletion     float64  // [0, 1]
    LastActiveDaysScore   float64  // [0, 1]
    AccountAgeBonus       float64  // [0, 1]
    // Raw signals used by anti-gaming (not in the score directly)
    LostDisputesCount     int
    UniqueReviewers       int
    MaxReviewerShare      float64
    AccountAgeDays        int
}
```

**`Reranker`** (interface, in `internal/search/scorer/reranker.go`) :
```go
type Reranker interface {
    Score(ctx context.Context, q Query, f Features, persona Persona) RankedScore
}

type RankedScore struct {
    Base       float64    // positive composite, before negatives
    Adjusted   float64    // after negative_signals multiplication
    Final      float64    // final display score 0-100
    Breakdown  map[string]float64  // for debugging + future explainability
}
```

**`BusinessRules`** (in `internal/search/rules/`) :
```go
type BusinessRules struct{ /* env-configured knobs */ }

func (r *BusinessRules) Apply(ctx context.Context, candidates []Candidate, persona Persona) []Candidate
// Handles tier sort, randomisation, diversity, rising talent, featured.
```

---

## 11. Configuration reference

All tuneable in env vars. Safe public defaults in this table; production values set in Railway / Vercel environment.

### 11.1 Per-persona feature weights

```
RANKING_WEIGHTS_FREELANCE_TEXT_MATCH       = 0.20
RANKING_WEIGHTS_FREELANCE_SKILLS_OVERLAP   = 0.15
RANKING_WEIGHTS_FREELANCE_RATING           = 0.20
RANKING_WEIGHTS_FREELANCE_PROVEN_WORK      = 0.15
RANKING_WEIGHTS_FREELANCE_RESPONSE_RATE    = 0.10
RANKING_WEIGHTS_FREELANCE_VERIFIED_MATURE  = 0.08
RANKING_WEIGHTS_FREELANCE_COMPLETION       = 0.07
RANKING_WEIGHTS_FREELANCE_LAST_ACTIVE      = 0.03
RANKING_WEIGHTS_FREELANCE_ACCOUNT_AGE      = 0.02

RANKING_WEIGHTS_AGENCY_TEXT_MATCH          = 0.15
RANKING_WEIGHTS_AGENCY_SKILLS_OVERLAP      = 0.10
RANKING_WEIGHTS_AGENCY_RATING              = 0.25
RANKING_WEIGHTS_AGENCY_PROVEN_WORK         = 0.25
RANKING_WEIGHTS_AGENCY_RESPONSE_RATE       = 0.05
RANKING_WEIGHTS_AGENCY_VERIFIED_MATURE     = 0.10
RANKING_WEIGHTS_AGENCY_COMPLETION          = 0.07
RANKING_WEIGHTS_AGENCY_LAST_ACTIVE         = 0.02
RANKING_WEIGHTS_AGENCY_ACCOUNT_AGE         = 0.01

RANKING_WEIGHTS_REFERRER_TEXT_MATCH        = 0.20
RANKING_WEIGHTS_REFERRER_SKILLS_OVERLAP    = 0.00
RANKING_WEIGHTS_REFERRER_RATING            = 0.35
RANKING_WEIGHTS_REFERRER_PROVEN_WORK       = 0.00
RANKING_WEIGHTS_REFERRER_RESPONSE_RATE     = 0.20
RANKING_WEIGHTS_REFERRER_VERIFIED_MATURE   = 0.10
RANKING_WEIGHTS_REFERRER_COMPLETION        = 0.10
RANKING_WEIGHTS_REFERRER_LAST_ACTIVE       = 0.03
RANKING_WEIGHTS_REFERRER_ACCOUNT_AGE       = 0.02
```

### 11.2 Formula parameters

```
RANKING_BAYESIAN_PRIOR_MEAN   = 4.0
RANKING_BAYESIAN_PRIOR_WEIGHT = 8
RANKING_COLD_START_FLOOR      = 0.15
RANKING_ENTROPY_MIN           = 3.5
RANKING_REVIEW_COUNT_CAP      = 50
RANKING_PROJECT_COUNT_CAP     = 100
RANKING_ACCOUNT_AGE_CAP_DAYS  = 365

RANKING_DISPUTE_PENALTY       = 0.10
RANKING_DISPUTE_PENALTY_CAP   = 0.30
```

### 11.3 Randomisation

```
RANKING_NOISE_COEFFICIENT        = 0.006
RANKING_NOISE_TOP3_MULTIPLIER    = 0.3
RANKING_NOISE_MID_MULTIPLIER     = 0.8
RANKING_NOISE_TAIL_MULTIPLIER    = 1.5
RANKING_RISING_TALENT_MAX_AGE    = 60     # days
RANKING_RISING_TALENT_SLOT_EVERY = 5      # 1 slot per 5 positions
```

### 11.4 Anti-gaming

```
RANKING_AG_MAX_TOKEN_REPETITION   = 5
RANKING_AG_MIN_DISTINCT_RATIO     = 0.3
RANKING_STUFFING_PENALTY          = 0.5

RANKING_AG_VELOCITY_CAP_24H       = 5
RANKING_AG_VELOCITY_COOLDOWN_DAYS = 14

RANKING_AG_LINKED_MAX_FRACTION    = 0.3

RANKING_AG_UNIQUE_REVIEWER_FLOOR  = 3
RANKING_AG_FEW_REVIEWER_CAP       = 0.4

RANKING_AG_NEW_ACCOUNT_AGE_DAYS   = 7
```

---

## 12. Appendix

### 12.1 Worked ranking trace

Query : **"développeur React Paris senior"**, persona filter **freelance**.

Top-200 from Typesense : 200 candidates, ordered by BM25 + vector.

Extract features for candidate **#23** (Romain Durand, Paris, React + Flutter, 14 projets, 11 clients uniques, repeat 36 %, KYC verified, account age 420 days, 23 reviews from 18 reviewers) :

```
text_match_score       = 0.82    // strong match on "React Paris"
skills_overlap_ratio   = 0.75    // 3 of 4 query skills match
rating_score_diverse   = 0.69    // 4.7 avg × 23 reviews × 18 unique × recency 0.82
proven_work_score      = 0.72    // (0.40×log(15) + 0.35×log(12) + 0.25×sqrt(0.36)) / log(101) = ...
response_rate          = 0.91
is_verified_mature     = 1.00    // verified + 420d ≥ 30d
profile_completion     = 0.88
last_active_days_score = 0.83    // 6 days ago
account_age_bonus      = 1.00    // capped
```

Apply weights (freelance) :
```
positive_score = 0.20×0.82 + 0.15×0.75 + 0.20×0.69 + 0.15×0.72 + 0.10×0.91 + 0.08×1.00 + 0.07×0.88 + 0.03×0.83 + 0.02×1.00
               = 0.164 + 0.1125 + 0.138 + 0.108 + 0.091 + 0.08 + 0.0616 + 0.0249 + 0.02
               = 0.800
```

No disputes → `negative_signals = 0`. `adjusted = 0.800`. `base_score = 80.0`.

Noise at rank 23 : `σ = 0.006 × 80 × 1.5 = 0.72`. Draw gaussian → `noise ≈ +0.34`. `final = 80.34`.

Would rank around position 5–8 out of 20 after all candidates are ranked and the diversity / tier / rising talent rules have been applied.

### 12.2 Related work

- **Malt's Matching Score** — public-facing score, weights private; strong inspiration.
- **Upwork Job Success Score (JSS)** — 6-month rolling window of outcomes. Inspired our proven_work + rating composite.
- **Contra's Rising Talent** — inspired §6.3.
- **Bayesian average for ratings** — standard in e-commerce: Amazon, IMDb, Yelp all use variants. `C` between 5-15 is typical.
- **LambdaMART** — the industry-standard LTR algorithm (Burges et al., 2010). The feature vector we log in §9 is directly consumable by LightGBM / XGBoost for V2.
- **Wilson score confidence interval** — alternative to Bayesian average for binary (thumbs up/down). Not used here because our ratings are 1-5 ordinal, but would be appropriate if we later add a binary `would_hire_again` signal.
- **Diversity-aware ranking** — MMR (Maximal Marginal Relevance, Carbonell & Goldstein 1998) is the classical reference; §6.5 is a simplified version.

### 12.3 Glossary

- **Bayesian average** : a weighted average that blends the observed sample mean with a prior (the marketplace mean). Prevents low-count profiles from dominating.
- **LTR (Learning-to-Rank)** : machine-learning approach where a model learns the ranking function from labelled `(query, doc, relevance)` tuples.
- **BM25** : the standard TF-IDF relevance formula used by Elasticsearch, Solr, Typesense.
- **Hybrid search** : combining keyword (BM25) and semantic (vector cosine) retrieval to maximise recall.
- **Winner-takes-all** : positive feedback loop where position-1 profiles get disproportionate clicks → further reinforces their position.

---

## 13. Known deferrals (scope for V2 and beyond)

- **Referral conversion score** — requires attribution tracking infrastructure (which apporteur brought which client, did it convert). Deferred until an attribution system is designed.
- **Query intent classifier** — "senior" / "junior" / "urgent" queries should weigh differently. Requires training data.
- **Personalisation** — user-specific features (past clicks, past hires) require ≥ 1 month of data per user.
- **Review sentiment analysis** — NLP pass on review text to detect lukewarm-but-5-star reviews. Nice-to-have; not blocking.
- **`is_featured` admin override** — field exists in schema, not wired into ranking. Dormant until a product decision is made about admin-promoted slots.
- **Per-persona `m` (Bayesian mean)** — V1 uses a single `m = 4.0`. If post-launch data shows agencies and referrers have meaningfully different means, split into `RANKING_BAYESIAN_PRIOR_MEAN_{PERSONA}`.
- **Geographic proximity boost** — currently a hard filter (city / country). Could become a soft proximity signal (closer = higher weight). Deferred.

---

*Document locked 2026-04-17. Changes require explicit sign-off from the product owner. All formulas have corresponding unit tests in `internal/search/features/`, `internal/search/scorer/`, and `internal/search/antigaming/`.*
