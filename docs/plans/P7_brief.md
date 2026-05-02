# P7 — ISP consumer migration (50+ call sites → segregated interfaces)

**Phase:** F.2 HIGH #2
**Source audit:** QUAL-FINAL (`auditqualite.md`) — Phase 3 J flag : segregated interfaces created but consumers still use wide repos
**Effort:** 2j est. mécanique
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p7-isp-consumer-migration`

## Problem

Phase 3 J (PR #44) created segregated child interfaces for 6 god repos:

| Wide repo (god) | Methods | Segregated children |
|---|---|---|
| `ReferralRepository` | 24 | `Reader` (10) + `Writer` (3) + `AttributionStore` (5) + `CommissionStore` (6) |
| `MessageRepository` | 21 | `Reader` (12) + `Writer` (7) + `BroadcasterStore` (2) |
| `OrganizationRepository` | 20 | `Reader` (7) + `Writer` (5) + `StripeStore` (8) |
| `DisputeRepository` | 18 | `Reader` (10) + `Writer` (5) + `EvidenceStore` (3) |
| `ProposalRepository` | 16 | `Reader` (11) + `Writer` (4) + `MilestoneStore` (1) |
| `UserRepository` | 15 | `Reader` (8) + `Writer` (3) + `AuthStore` (3) + `KYCStore` (1) |

**However**, the audit flags : ~50+ consumer call sites (services, handlers) still depend on the wide god interfaces. Examples :
- `wallet_handler.go` only needs `OrganizationStripeStore` (8 methods) but takes `OrganizationRepository` (20)
- Many `app/*/service.go` ServiceDeps fields take wide interface when they need 1-2 methods

The benefit of ISP isn't realized: mocks remain 24-method monsters, fake-construction time stays high, contract drift hurts evolution.

## Goal

For every consumer of a god interface, narrow the typed dependency to the smallest segregated child that satisfies its actual usage. Mocks shrink. Tests get faster. Future refactors become safer.

## Discovery

```bash
cd /tmp/mp-p7-isp/backend
# Inventory current consumers of each god interface
for iface in ReferralRepository MessageRepository OrganizationRepository DisputeRepository ProposalRepository UserRepository; do
  echo "=== $iface ==="
  grep -rn "$iface" internal/ --include="*.go" | grep -v "_test.go\|mocks_test.go\|port/repository/" | wc -l
done

# For each consumer file, list which methods of the wide interface it actually calls
# (heuristic — the agent will read each consumer's dependency type field + usage)
```

Expected: 50-100 typed-field sites across `internal/app/*/service*.go`, `internal/handler/*_handler.go`, `cmd/api/wire_*.go`.

## Migration pattern

For each consumer site:

```go
// BEFORE — wide dep
type Service struct {
    org repository.OrganizationRepository  // 20 methods
}

func (s *Service) DoX(ctx, orgID) {
    s.org.GetStripeAccount(ctx, orgID)  // only 1 method ever called
}

// AFTER — narrowed dep via segregated child
type Service struct {
    org repository.OrganizationStripeStore  // 8 methods
}

// Same call site, narrower contract.
```

If a consumer uses methods spanning multiple children (e.g. Reader + StripeStore), declare a custom interface in the consumer's package (Go interface composition):

```go
// app/wallet/service.go
type orgRepo interface {
    repository.OrganizationStripeStore
    GetByID(ctx context.Context, id uuid.UUID) (*organization.Organization, error)
}
```

Or accept the wider interface and document why — only if the methods truly span 3+ children.

## Plan (per-domain, ~6 commits)

### Commit 1 — `OrganizationRepository` consumers
Most-used. Sites likely: `wallet_handler.go`, `payment/service.go`, `embedded_handler.go`, `referral/wiring_adapters.go`, `organization/service.go`, etc. Narrow to the right child(ren) per call.

### Commit 2 — `ProposalRepository` consumers
Sites: `app/dispute/`, `app/review/`, `app/payment/`, `app/referral/wiring_adapters.go`, etc.

### Commit 3 — `DisputeRepository` consumers
Sites: `app/dispute/service*.go`, `handler/dispute_handler.go`, scheduler.

### Commit 4 — `MessageRepository` consumers
Sites: `app/messaging/`, `app/notification/`, `handler/messaging_handler.go`.

### Commit 5 — `ReferralRepository` consumers
Sites: `app/referral/*.go` (the bulk), `wire_referral.go`.

### Commit 6 — `UserRepository` consumers
Sites: `app/auth/`, `app/proposal/`, `app/job/`, `handler/admin_handler.go`. Note: `users` is referenced widely; expect this to be the largest commit.

## Hard constraints

- **Validation pipeline before EVERY commit**:
  ```bash
  cd /tmp/mp-p7-isp/backend
  go build ./...
  go vet ./...
  go test ./... -count=1 -short -race
  ```
  All green.

- **Zero behaviour change.** Only the typed dependency NARROWS. The function bodies don't change. The mocks in `_test.go` files are updated to satisfy the new narrower interface (often: drop unused methods).

- **Mock shrinkage**: every test file that mocked a god interface should now mock the narrower child. Drop the unused method stubs from the test mocks (`mocks_test.go`).

- **Don't introduce new interfaces** unless absolutely necessary (when a consumer truly straddles 2 children). Prefer using the existing segregated children verbatim.

- **One commit per domain** (6 commits expected per brief above).

## Tests required

No new test files needed for P7 (it's a typing refactor, not a feature). The existing test suite passing post-migration is the regression net. If a test breaks because the new narrower interface lacks a method that the test was calling on the mock — that's a sign the consumer actually needs the wider interface. Re-evaluate per case.

Bonus tests (acceptable, not required):
- 1-2 smoke tests asserting a service's `Deps` struct field type matches the segregated child it should (compile-time check via `var _ <SegregatedChild> = …`).

## OFF-LIMITS

- LiveKit / call code — never touch
- `.github/workflows/*` — token can't push
- Other plans (P6, P8-P11) — never touch
- Web / admin / mobile — out of scope
- Refactoring repository implementations themselves — only changes are the **consumer typed fields** + the **mocks_test.go** files

## Branch ownership

Agent creates `fix/p7-isp-consumer-migration` from clean `main` via `git worktree add`. Never touches another branch.

## Final report (under 800 words)

Lead with PR URL.

1. Per-domain count : sites migrated (e.g., OrganizationRepository: 12 sites narrowed → 8× StripeStore + 3× Reader + 1× full)
2. Total mocks shrunk (count + median method-drop per mock)
3. Validation pipeline output (per commit's `go test` summary)
4. Out-of-scope items flagged (consumers that truly need the wide interface — list with rationale)
5. "Branch ownership confirmed: only worked on `fix/p7-isp-consumer-migration`"

GO. Take your time per call site — narrowing wrongly causes compile errors and triggers re-work.
