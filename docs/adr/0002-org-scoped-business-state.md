# 0002. Org-scoped business state

Date: 2026-04-30

## Status

Accepted

## Context

The marketplace serves three primary roles — Agency
(prestataire), Enterprise (client), and Provider (freelance) —
plus an admin role. An Agency typically has multiple users
(operations lead, account managers, billing manager) sharing one
billing address, one Stripe Connect account, one wallet balance,
and one Premium subscription. The marketplace contracts
(missions, proposals, invoices) are issued **between agencies**,
not between individual users.

The first migrations modeled business state per `user_id`:
subscriptions, wallet, KYC, contracts, all keyed off the
authenticated user. This works for a marketplace of solo
freelancers but fails for agencies the moment a teammate departs:

- The departed teammate's account holds the Stripe Connect link
  and the agency cannot receive payouts until support manually
  re-attaches.
- The agency's Premium subscription is tied to that user, not to
  the company — losing the user means losing the subscription.
- Invoices say "John Doe" instead of "Acme Studio Ltd", which
  causes accounting trouble at the agency's end.

We discovered this drift mid-project after the team feature
(invitations, member roles) was implemented but business tables
still keyed off `user_id`. The fix was a 6-week migration to move
ownership semantics to a new `organizations` table.

## Decision

**All business state is owned by an `organization`, not by a
`user`.** Users are *members* of an organization; the organization
is the legal/economic entity.

Concrete rules:

1. Tables holding business state use `organization_id UUID NOT
   NULL` foreign-keying `organizations(id)`. Examples: `proposals`,
   `wallet_balances`, `invoices`, `kyc_records`, `subscriptions`,
   `social_links`, `billing_profiles`.
2. `user_id` columns are reserved for **action authorship** —
   `created_by`, `updated_by`, audit log entries. Never for "who
   owns this resource".
3. Handlers resolve `organization_id` from the authenticated user's
   membership (carried in the JWT/session) via
   `middleware.GetOrganizationID(ctx)`. Queries always filter by
   `organization_id`, never by `user_id`, for any business read.
4. RBAC has two dimensions: the **primary role** (`agency`,
   `provider`, `enterprise`, `admin`) and the **org role**
   (`owner`, `admin`, `member`, `viewer`). The latter is the
   per-org permission cursor.
5. The org-resolver utilities (`internal/system/system_actor.go`)
   document the system-actor escape valve for cron jobs and
   webhooks where no end-user is in scope (see ADR 0003).
6. Solo Providers are still backed by an organization — a
   single-member auto-provisioned `solo_provider` org. The
   abstraction is uniform from the database's perspective.

## Consequences

### Positive

- A teammate can leave Acme Studio Ltd without breaking its
  subscription, wallet, or Stripe Connect link. The org owns the
  resources; the departing user simply loses membership.
- Invoices issued to "Acme Studio Ltd" reflect the legal entity
  matching the agency's tax filings.
- The team feature (invitations, role updates, transfers) operates
  on a clean object model — no fictitious "co-owners" of one
  user's records.
- A future B2B enterprise feature (multiple billing contacts,
  cost-center tags) extends naturally as additional fields on
  `organizations`.

### Negative

- All `user_id`-keyed business tables had to migrate. We did this
  in 6 sub-phases (R-series migrations) to avoid a single risky
  release. Each phase had its own validation harness.
- Backwards compatibility for legacy data: pre-org data was
  back-filled by auto-provisioning a solo-provider org per user
  and re-pointing the existing rows. The migration is documented
  in `migrations/0XX_backfill_org_scoped_business_state.up.sql`.
- `subscriptions` was discovered to still use `user_id` for
  ownership in 2026-04-22. Flagged as a known bug (see
  `MEMORY.md::project_org_based_model.md`); will migrate in a
  follow-up phase.

## Alternatives considered

- **Tenant-per-user with a virtual `team` view** — keep `user_id`
  as the shard key but materialize a "team aggregate" view for
  agency dashboards. Rejected because every business table would
  still leak the per-user shard identity into application logic;
  the join layer is brittle.
- **Multi-org per user with a default org** — a user can be a
  member of N orgs and switches "active org" via a UI selector.
  We chose to implement this on top of org-scoped state (a user's
  `org_memberships` is a list); the underlying business tables
  still own state via `organization_id`. The selector is purely
  UI/JWT.

## References

- `backend/internal/handler/middleware/auth.go` — `Auth`
  middleware injects `ContextKeyOrganizationID`.
- `backend/internal/handler/middleware/requestid.go::GetOrganizationID` /
  `MustGetOrgID` — accessor + panic-on-missing helper.
- `backend/internal/system/` — system-actor helpers for
  background jobs that legitimately have no user context.
- Migration series `R*` — moved ownership of subscriptions,
  wallet, KYC, etc. to `organization_id`.
- `CLAUDE.md` lines 122-135 — the org-scoping rule for new
  features.
