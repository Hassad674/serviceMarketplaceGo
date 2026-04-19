# Ranking V1 — Tuning Guide (INTERNAL)

> **Audience**: SRE + ranking engineer. Do NOT share the exact numbers
> below outside the team — the algorithm is open-source, the weights
> are not. This document is checked into the repo only because access
> is already team-gated at the GitHub level.
>
> Last touched: Phase 6I (2026-04-19). Tracks `docs/ranking-v1.md`
> locked spec (2026-04-17).

---

## How to read this guide

Every row lists a knob, its current recommendation, and the direction
to move it when a specific symptom appears in prod. The published
defaults in `docs/ranking-v1.md` §11 are SAFE defaults — suitable for
bootstrapping. The production values are tracked in Railway / Vercel
env + in the team's 1Password vault.

**Before changing any weight in prod:**

1. Read the relevant section of `docs/ranking-v1.md` — the formula
   must be understood before retuning it.
2. Draft the change in a dated entry in the "Tuning log" table at the
   bottom of this doc. Include: date, knob, from → to value, the
   observed symptom motivating the change, and the expected effect.
3. Roll the change out behind the A/B framework (§13a.2 in
   ranking-v1.md) once the framework lands. Until then, gate large
   swings via an office-hours rollout with Datadog watch.

---

## 1. Per-persona feature weights

Six env vars per persona × 3 personas = 27 rows. All values are
unitless weights that sum to 1.0 per persona. See §4 of the V1 spec
for the rationales.

### 1.1 Freelance (`RANKING_WEIGHTS_FREELANCE_*`)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_WEIGHTS_FREELANCE_TEXT_MATCH` | 0.20 | ↑ if relevance feedback drops; ↓ if keyword stuffers are surfacing. |
| `RANKING_WEIGHTS_FREELANCE_SKILLS_OVERLAP` | 0.15 | ↑ when users filter by skill and expect strict matches; ↓ if skill-empty queries suffer. |
| `RANKING_WEIGHTS_FREELANCE_RATING` | 0.20 | ↑ when high-quality profiles sit too low; ↓ if Bayesian shrinkage overfires. |
| `RANKING_WEIGHTS_FREELANCE_PROVEN_WORK` | 0.15 | ↑ when users say "I want someone who's shipped"; ↓ when junior talent is invisible. |
| `RANKING_WEIGHTS_FREELANCE_RESPONSE_RATE` | 0.10 | ↑ when ghost-rate is high; ↓ when responsive-but-low-quality dominates. |
| `RANKING_WEIGHTS_FREELANCE_VERIFIED_MATURE` | 0.08 | ↑ after a trust incident; ↓ rarely — KYC is near-free signal. |
| `RANKING_WEIGHTS_FREELANCE_COMPLETION` | 0.07 | ↑ when half-filled profiles game ranking; ↓ rarely. |
| `RANKING_WEIGHTS_FREELANCE_LAST_ACTIVE` | 0.03 | ↑ only on complaint "profile I hit messaged last seen 2 months ago". |
| `RANKING_WEIGHTS_FREELANCE_ACCOUNT_AGE` | 0.02 | Never ↑ meaningfully — veteran bias trap. |

**Sum invariant:** the 9 weights must total 1.0 within ±0.01.
`scorer.Config.Validate()` rejects a mis-sum at boot — prod will
refuse to start. Fix the env vars, don't bypass the check.

### 1.2 Agency (`RANKING_WEIGHTS_AGENCY_*`)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_WEIGHTS_AGENCY_TEXT_MATCH` | 0.15 | ↑ if B2B queries on tech-stack terms underperform. |
| `RANKING_WEIGHTS_AGENCY_SKILLS_OVERLAP` | 0.10 | Keep low — agencies cover broad skills, strict overlap misleads. |
| `RANKING_WEIGHTS_AGENCY_RATING` | 0.25 | ↑ carefully — reviews are the primary B2B trust signal. |
| `RANKING_WEIGHTS_AGENCY_PROVEN_WORK` | 0.25 | ↑ when "agency with references" complaints spike. |
| `RANKING_WEIGHTS_AGENCY_RESPONSE_RATE` | 0.05 | ↑ slowly — agency sales cycles are structurally slower. |
| `RANKING_WEIGHTS_AGENCY_VERIFIED_MATURE` | 0.10 | ↑ after trust incidents; more weight than freelance because contract sizes are larger. |
| `RANKING_WEIGHTS_AGENCY_COMPLETION` | 0.07 | ↑ if empty-portfolio agencies rank too high. |
| `RANKING_WEIGHTS_AGENCY_LAST_ACTIVE` | 0.02 | Keep tiny. |
| `RANKING_WEIGHTS_AGENCY_ACCOUNT_AGE` | 0.01 | Keep tinier. |

### 1.3 Referrer / apporteur d'affaires (`RANKING_WEIGHTS_REFERRER_*`)

Referrers don't sell skills and don't complete projects themselves —
two weights are pinned at zero by design.

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_WEIGHTS_REFERRER_TEXT_MATCH` | 0.20 | ↑ when sector/vertical queries rank poorly. |
| `RANKING_WEIGHTS_REFERRER_SKILLS_OVERLAP` | 0.00 | Do NOT raise — referrers don't sell skills. |
| `RANKING_WEIGHTS_REFERRER_RATING` | 0.35 | Dominant signal. ↑ rarely; ↓ when the 3-review reviewer-diversity rule overfires. |
| `RANKING_WEIGHTS_REFERRER_PROVEN_WORK` | 0.00 | Do NOT raise — feature is undefined for referrers. |
| `RANKING_WEIGHTS_REFERRER_RESPONSE_RATE` | 0.20 | ↑ when ghosting complaints spike (most impactful). |
| `RANKING_WEIGHTS_REFERRER_VERIFIED_MATURE` | 0.10 | Keep — trust anchor for commission-based relationships. |
| `RANKING_WEIGHTS_REFERRER_COMPLETION` | 0.10 | ↑ when network-description quality drops. |
| `RANKING_WEIGHTS_REFERRER_LAST_ACTIVE` | 0.03 | Slightly higher than other personas — stale network = dead value. |
| `RANKING_WEIGHTS_REFERRER_ACCOUNT_AGE` | 0.02 | Keep tiny. |

---

## 2. Formula parameters

### 2.1 Bayesian rating (§3.2-3)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_BAYESIAN_PRIOR_MEAN` | 4.0 | ↑ if platform-wide avg shifts >0.3 above. Rarely touched. |
| `RANKING_BAYESIAN_PRIOR_WEIGHT` | 8 | ↑ when new profiles with single glowing review outrank veterans; ↓ if shrinkage is unfair to genuinely good newcomers. |
| `RANKING_COLD_START_FLOOR` | 0.15 | ↑ "increase cold-start floor if newcomers complain"; ↓ if newcomers gaming floor + Rising Talent combo. Keep ∈ [0.05, 0.25]. |
| `RANKING_REVIEW_COUNT_CAP` | 50 | Rarely touched. |

### 2.2 Proven work (§3.2-4)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_PROJECT_COUNT_CAP` | 100 | ↑ if top 1 % of highly-active profiles complain of saturation; rarely touched. |

### 2.3 Profile completion (§3.2-7)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_ENTROPY_MIN` | 3.5 | ↑ if junk-text profiles still rank despite the rule; ↓ only when genuine low-entropy text (terse bios in one language) is being penalised. |

### 2.4 Account age (§3.2-9)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_ACCOUNT_AGE_CAP_DAYS` | 365 | Never increase — avoids a "veterans win" singularity. |

### 2.5 Negative signals (§5.3)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_DISPUTE_PENALTY` | 0.10 | ↑ after a trust incident. Each lost dispute multiplies the score by (1 − RANKING_DISPUTE_PENALTY). |
| `RANKING_DISPUTE_PENALTY_CAP` | 0.30 | ↑ carefully — the cap prevents 5+ disputes zeroing a profile that might genuinely still be useful for non-disputed work. |

---

## 3. Randomisation (`internal/search/rules/`, §6.1)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_NOISE_COEFFICIENT` | 0.006 | ↑ to rotate mid-tier more (give impressions to positions 11-20); ↓ when top-3 jitters annoy users. |
| `RANKING_NOISE_TOP3_MULTIPLIER` | 0.3 | Keep low — top-3 should be stable. |
| `RANKING_NOISE_MID_MULTIPLIER` | 0.8 | Moderate rotation. |
| `RANKING_NOISE_TAIL_MULTIPLIER` | 1.5 | Aggressive rotation — tail candidates should get fair impression budget. |

**Rule of thumb:** a change of 0.001 in `NOISE_COEFFICIENT` shifts σ by
one point on a score of 100 at the tail. Start small.

---

## 4. Rising Talent (§6.3)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_RISING_TALENT_MAX_AGE` | 60 (days) | ↑ when "new profiles never show up" complaints spike; ↓ when gaming is detected (fresh accounts spamming verification). |
| `RANKING_RISING_TALENT_SLOT_EVERY` | 5 | ↑ (e.g. 10) to rely less on slotting; ↓ to surface more newcomers. |
| `RANKING_RISING_TALENT_DELTA` | 5.0 | Score delta below the incumbent a rising candidate can be. ↑ to surface more aggressively; ↓ when the diversity of top-20 suffers. |

---

## 5. Featured override (§8)

Dormant V1. Flip to true only after product decides on admin-promoted
slots.

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_FEATURED_ENABLED` | false | Flip to `true` to activate. Make the flip a product decision — leaves a trace in audit logs. |
| `RANKING_FEATURED_BOOST` | 0.0 | Multiplicative boost applied to Score.Final. 0.15 = +15 %. Keep ≤ 0.2 even when enabled — beyond that, featured fight the baseline too hard. |

---

## 6. Anti-gaming knobs (§7)

### 6.1 Keyword stuffing (§7.1)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_AG_MAX_TOKEN_REPETITION` | 5 | ↓ to catch stuffers more aggressively; ↑ if false positives on legitimate glossaries. |
| `RANKING_AG_MIN_DISTINCT_RATIO` | 0.3 | ↑ to demand more variety; ↓ only when legitimate single-topic profiles are being flagged. |
| `RANKING_STUFFING_PENALTY` | 0.5 | Multiplicative hit on `text_match_score`. 0.5 = cut in half. ↓ to escalate; ↑ (toward 1.0) softens. |

### 6.2 Review velocity cap (§7.2)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_AG_VELOCITY_CAP_24H` | 5 | Max reviews counted per 24 h window. ↓ to escalate; ↑ to loosen. |
| `RANKING_AG_VELOCITY_COOLDOWN_DAYS` | 14 | Cooldown after a spike. ↑ to keep the cap active longer. |

### 6.3 Linked accounts (§7.3)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_AG_LINKED_MAX_FRACTION` | 0.3 | Max share of reviews from linked accounts (IP/email-domain/device). ↓ to escalate. |

### 6.4 Unique reviewer floor (§7.4)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_AG_UNIQUE_REVIEWER_FLOOR` | 3 | Below this, cap `rating_score_diverse`. ↑ to demand more diversity. |
| `RANKING_AG_FEW_REVIEWER_CAP` | 0.4 | Hard cap when the floor kicks in. ↓ to escalate; ↑ rarely. |

### 6.5 New account score cap (§7.5)

| Env var | Default | Direction to move |
| --- | ---: | --- |
| `RANKING_AG_NEW_ACCOUNT_AGE_DAYS` | 7 | Grace period before a new account can rank freely. ↑ (e.g. 14) after a spam wave. |

---

## 7. LTR logging (`migration 113`, §9)

No knobs. Every ranked response writes its feature vector to
`search_queries.result_features_json`. Storage: ≈ 3 KB/query, ≈ 30
MB/day at 10 k queries/day. Monitor `pg_total_relation_size`
('public.search_queries') monthly; archive to cold storage after 12
months if it exceeds 15 GB.

---

## 8. Tuning playbook — common symptoms

| Symptom | First lever | Second lever |
| --- | --- | --- |
| "Newcomers never show up" | ↑ `RANKING_RISING_TALENT_MAX_AGE` | ↑ `RANKING_COLD_START_FLOOR` |
| "Top-1 is always the same, clicks boring" | ↑ `RANKING_NOISE_COEFFICIENT` | ↓ `RANKING_NOISE_TOP3_MULTIPLIER` cautiously (keep ≥ 0.2) |
| "Keyword-stuffer on page 1" | ↓ `RANKING_AG_MAX_TOKEN_REPETITION` | ↓ `RANKING_STUFFING_PENALTY` |
| "Fresh KYC'd spammer ranking high" | ↑ `RANKING_AG_NEW_ACCOUNT_AGE_DAYS` | ↑ `RANKING_AG_VELOCITY_COOLDOWN_DAYS` |
| "Agency with no portfolio outranks one with 20 projects" | ↑ `RANKING_WEIGHTS_AGENCY_PROVEN_WORK` | ↑ `RANKING_WEIGHTS_AGENCY_COMPLETION` |
| "Referrer list feels random" | ↑ `RANKING_WEIGHTS_REFERRER_RATING` | ↑ `RANKING_WEIGHTS_REFERRER_RESPONSE_RATE` |
| "Top-20 is all React devs" | ↓ `RANKING_WEIGHTS_FREELANCE_SKILLS_OVERLAP` | (diversity already breaks 3+ runs; consider a persona-broader query instead) |
| "Dispute loser still on page 1" | ↑ `RANKING_DISPUTE_PENALTY` | ↑ `RANKING_DISPUTE_PENALTY_CAP` |
| "B2B client wants exact-skills match" | ↑ `RANKING_WEIGHTS_FREELANCE_SKILLS_OVERLAP` | ↑ `RANKING_WEIGHTS_FREELANCE_TEXT_MATCH` |

---

## 9. Observability

All anti-gaming penalties emit a structured log line (`event:
ranking.penalty_applied`) per §7.6. Aggregate with a Datadog saved
query:

```
service:api event:ranking.penalty_applied | rollup count by rule, persona
```

Business-rule effects are NOT logged today (§6 is deterministic given
seed). The LTR feature-vector table is the ground truth for any
retrospective analysis — every ranked profile's features + score +
position are recoverable.

---

## 10. Tuning log

Append-only. Do not edit historical entries.

| Date | Knob | From → To | Symptom / Expected effect | Operator |
| --- | --- | --- | --- | --- |
| 2026-04-17 | — | — | Spec locked, defaults published. | Hassad |
| 2026-04-19 | — | — | Phase 6E + 6G + 6I merged. Rules layer + LTR logging live. | Claude R2-R |

*Keep this table short. Move entries older than 12 months to an
archive file if the list grows past 40 rows.*
