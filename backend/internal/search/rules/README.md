# internal/search/rules

Business rules layer of the V1 search ranking pipeline.

**Spec**: [`docs/ranking-v1.md`](../../../../docs/ranking-v1.md) §6 + §8.
**Round**: Phase 6E.

## What it does

Given a list of scored candidates (output of the scorer package), produces
the final top-20 rendered to the user. The pipeline is:

```
scorer output
      │
      ▼
┌──────────────────────────────┐
│ 1. Tier sort (§6.4)          │  available_now / available_soon above not_available
│ 2. Randomise (§6.1)          │  gaussian noise, rank-dependent σ
│ 3. Re-sort per tier          │  by noise-adjusted Final
│ 4. Diversity (§6.5)          │  break 3+ runs of same primary_skill
│ 5. Rising Talent (§6.3)      │  slots 5/10/15/20 for new+verified
│ 6. Featured (§8, dormant V1) │  admin-controlled boost
│ 7. Truncate to TopN          │  default 20
└──────────────────────────────┘
      │
      ▼
  top-20 rendered
```

## Public surface

```go
type Candidate struct {
    DocumentID         string
    OrganizationID     string
    Persona            Persona
    Feat               Features
    Score              Score
    AvailabilityStatus string
    PrimarySkill       string
    AccountAgeDays     int
    IsFeatured         bool
    IsVerified         bool
}

type Config struct { /* 12 tuneable knobs — see types.go */ }

func DefaultConfig() Config
func LoadConfigFromEnv() (Config, error)
func NewBusinessRules(cfg Config) *BusinessRules
func (r *BusinessRules) Apply(ctx context.Context, candidates []Candidate, persona Persona) []Candidate
```

## Determinism

Every rule takes a caller-injected RNG when randomness is involved.
Setting `Config.RandSeed` to a non-zero int64 locks the sequence —
tests rely on this. In production the seed defaults to `time.Now().UnixNano()`.

Concurrent calls to `Apply` are safe: each `Apply` builds its own local
`*rand.Rand`, so two goroutines never share the stateful RNG.

## Testing

```bash
cd backend
go test ./internal/search/rules/... -count=1 -race -cover
```

Coverage target: ≥ 95 %. Property tests assert:

- `Apply` never invents a candidate.
- `Apply` never duplicates a candidate.
- Top-1 is always Tier A whenever any Tier A candidate exists.
- Rising Talent eligibility is never violated — no ineligible candidate
  lands in a rising slot.

## Environment knobs

All knobs are configurable via `RANKING_*` env vars. See
[`docs/ranking-tuning.md`](../../../../docs/ranking-tuning.md) for the
operator guide (internal).

## Non-goals

- No I/O (no database, no HTTP, no cache).
- No persona-specific logic in V1 (the `persona` argument is reserved
  for future per-persona rules — §13a.1).
- No user-visible randomness beyond the documented noise envelope.
