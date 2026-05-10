# GDPR/RGPD Roadmap — Marketplace Service

> Source: `gdpr-audit.md` (audit 2026-05-10)
> Last updated: 2026-05-10
> Total estimated effort: ~158h (3 weeks dev)

This file is the single source of truth for the GDPR/RGPD compliance work.
Every commit referencing GDPR work updates the relevant row's status, commit
SHA, and key file pointers. The audit (`gdpr-audit.md`) captures the
diagnostic; this roadmap captures the execution.

## Status legend

- ⬜ TODO
- 🟡 IN PROGRESS
- ✅ DONE
- 🔴 BLOCKED (with reason)
- ⏸️ DEFERRED (with reason)
- ✳️ NOT APPLICABLE (with explanation)

---

## Phase A — Quick wins (target: 2.5 days, ~20h)

| # | Item | Effort | Status | Files | Tests | Notes |
|---|---|---|---|---|---|---|
| A.1 | Drop `payment_info.iban/account_*` plaintext columns | 3h | ✳️ | `backend/migrations/042_drop_custom_kyc_tables.up.sql` | n/a | Already done: migration `042` dropped the entire `payment_info` table when KYC was migrated to Stripe Embedded. No live code references the bank columns; the legacy `domain/payment` entity has IBAN/AccountNumber fields used only as transit DTO when forwarding to Stripe (`adapter/stripe/account.go`). The audit's recommendation was based on legacy migrations 015/020 — these tables no longer exist in the schema. **No further action needed.** |
| A.2 | Integrate CMP (vanilla-cookieconsent, replaces custom banner) | 4h | ✅ | `web/src/shared/components/analytics/cookie-consent-provider.tsx`, `web/src/shared/lib/cookie-consent-config.ts`, `web/src/styles/cookie-consent.css`, `web/src/shared/components/analytics/posthog-provider.tsx`, `web/src/shared/lib/posthog-consent.ts`, `web/src/app/[locale]/providers.tsx`, `web/messages/{fr,en}.json` (`cookieConsent.*`) | `cookie-consent-provider.test.tsx` (10 tests), `cookie-consent-config.test.ts` (5 tests), updated `posthog-provider.test.tsx` (4 tests) + `posthog-consent.test.ts` (+2 tests), e2e `cookie-consent-vanilla.spec.ts` (4 scenarios) | Replaced Tarteaucitron with `vanilla-cookieconsent` (3.1.0, MIT, ~15KB gzipped) — modern UX, granular categories, RGPD pause-before-consent verified by tests (PostHog SDK never initialised before user accepts `analytics`). Soleil v2 CSS overrides via `--cc-*` tokens, no hex hardcoded. consent_log POST integrated for accept_all / refuse_all / custom paths. |
| A.3 | Add `consent_log` table + insert on accept/refuse | 3h | ✅ | migration `139`, `backend/internal/domain/consent/`, `backend/internal/app/consent/`, `backend/internal/handler/consent_handler.go`, `web/src/shared/lib/posthog-consent.ts` | `consent_handler_test.go`, `consent/service_test.go` (13 tests) | Commits `61aa371a` (backend) + `dfa671a1` (web wiring). Anonymized IP /16 + UA SHA-256 hex. Endpoint `POST /api/v1/consent/log`. Anonymous + authenticated visitors both supported; honors X-Forwarded-For. |
| A.4 | Create placeholder routes `/privacy /cookies /legal /cgu /cgv /sous-processeurs` | 2h | ✅ | 6 Next.js pages under `web/src/app/[locale]/(public)/`, `web/src/shared/components/legal/legal-shell.tsx` | `web/src/app/[locale]/(public)/__tests__/legal-pages-metadata.test.ts` (6 tests) | Commit `23b051e7`. Footer links + i18n FR/EN + Soleil v2. Pages are placeholders (`noindex`) until Phase C content is written. |
| A.5 | DPO email designated + surfaced in footer + policy | 1h | ✅ | `web/src/shared/components/legal/legal-footer.tsx`, `web/src/shared/lib/dpo.ts`, `web/src/app/[locale]/(public)/privacy/page.tsx`, `messages/{fr,en}.json` | `web/src/shared/components/legal/__tests__/legal-footer.test.tsx` (3 tests) | Commit `23b051e7`. Default `hassad.smara69@gmail.com`; configurable via `NEXT_PUBLIC_DPO_EMAIL` env var. Surfaced in footer mailto + privacy page contact line. |

**Phase A completion criteria:** all rows ✅ or ✳️, validation pipeline green, branch pushed, manual steps documented for the user.

---

## Phase B — Compliance core (target: 1 week, ~56h)

Source: `gdpr-audit.md` Section 13 Phase B.

| # | Item | Effort | Status | Files | Tests | Notes |
|---|---|---|---|---|---|---|
| B.1 | Retention scheduler — `messages` (3y), `notifications` (90d), `device_tokens` (60d inactivity), `search_queries` (12mo → anonymize user_id+session_id), `audit_logs` (24mo → archive table) | 8h | ✅ | `backend/internal/domain/retention/`, `backend/internal/app/retention/`, `backend/internal/port/repository/retention.go`, `backend/internal/adapter/postgres/retention_repository.go`, `backend/cmd/api/wire_retention.go`, migrations 140/141/142 | `domain/retention/policy_test.go` (10 tests), `app/retention/service_test.go` (8 tests), `adapter/postgres/retention_repository_test.go` (5 integration tests) | 5 policies, 1h scheduler interval (1m dev), audit_logs to `audit_logs_archive` table (R2 cold storage deferred to Phase C). Materialised CTE pattern in DELETE/UPDATE/archive to avoid Postgres nested-loop subquery LIMIT pitfall. `device_tokens.last_seen_at` added + refreshed on every push delivery. RETENTION_INTERVAL env override. |
| B.2 | `audit_logs` cold-storage archiving (24mo → R2 jsonl.gz then DELETE) | 6h | ⏸️ | `backend/internal/app/retention/audit_archive.go`, R2 path `audit-archive/<yyyy-mm>/...jsonl.gz` | archival_test.go | DEFERRED — B.1 already moves rows >24mo to `audit_logs_archive` (secondary table, fully queryable). The R2 cold-tier move is a Phase C cost optimisation; the storage-limitation contract is already satisfied by B.1. Introduce dedicated `marketplace_archiver` DB role with `INSERT, SELECT, DELETE` on `audit_logs` when the R2 path lands. |
| B.3 | Art. 22 "human review" — appeal CTA on auto-rejected moderation | 6h | ⬜ | `domain/moderation` (add `Appealable` flag), new endpoint `POST /api/v1/moderation/{id}/appeal`, web "Demander une revue humaine" button | handler test + service test | Audit log records `moderation.appeal_requested`. Hooks into `/admin/moderation`. |
| B.4 | Art. 21 "object" — hide-from-search opt-out | 6h | ⬜ | migration `users.search_indexed BOOLEAN NOT NULL DEFAULT true`, `Indexer.Sync` skip + Typesense `Delete`, settings UI toggle | indexer test + handler test | Toggle persists user preference; export still includes the user in their own data. |
| B.5 | RGPD art. 22 disclosure + appeal procedure (automated decisions) | 4h | ✅ | migration 144 (`automated_decision_appeals`), `backend/internal/domain/automateddecision/`, `backend/internal/app/automateddecision/`, `backend/internal/port/repository/automated_decision_appeal.go`, `backend/internal/adapter/postgres/automated_decision_appeal_repository.go`, handler + routes (`automated_decision_appeal_handler.go`, `routes_automated_decision.go`), web `/decisions-automatisees` page + privacy section + footer link, FR + EN messages, `GDPR_CONTACT_EMAIL` config | 17 tests (5 domain + 5 app + 4 handler + 3 sub-cases via t.Run) green | Discloses the 3 automated decisions (AI moderation, search ranking, Stripe risk scoring) + exposes `POST /api/v1/me/automated-decision-appeals` with admin email best-effort notification. Marketing-email opt-in tracking moved to a follow-up under the original placeholder rationale. |
| B.6 | Idempotency-Key + ratelimit on `request-deletion` (3 req/min cap) | 2h | ⬜ | `backend/internal/handler/middleware/ratelimit.go` config | unit test on rate cap | Already idempotent by design — only the rate cap needs tightening. |
| B.7 | R2 object cleanup on user purge (right-to-erasure complétion — Section 1 sensitivity flags 3 + 6) | 4h | ✅ | `backend/internal/domain/gdpr/storage_purge.go`, `backend/internal/port/service/storage_service.go` (BulkDelete), `backend/internal/adapter/s3/storage_bulk_delete.go`, `backend/internal/adapter/postgres/gdpr_storage_purge.go`, `backend/internal/app/gdpr/service.go` (purgeStorageForUser), migration 144 (`storage_purge_audits`), `cmd/api/wire_gdpr.go` | `app/gdpr/storage_purge_test.go` (8 tests), `adapter/s3/storage_bulk_delete_test.go` (3 tests) | Commit `74d895ba`. Purge cron now BulkDeletes every R2 key tied to the user (avatars, profile videos, portfolio media, KYC docs, review videos, jobs/applications videos, message attachments) BEFORE SQL anonymization. Per-key results captured in `storage_purge_audits` table as compliance evidence. Best-effort: R2 transport failures never abort SQL anonymization. |
| B.11 | Wire dedicated `marketplace_app` DB role with `INSERT+SELECT-only` on `audit_logs` (was B.7 before the R2 cleanup re-scope of 2026-05-10) | 4h | ⬜ | infra (Railway/Neon dashboards) + `DATABASE_URL` env update | CI smoke test | Manual infra step; document in `MIGRATION_KYC_EMBEDDED.md`-style runbook. |
| B.8 | Verify EU residency: Neon, Typesense, Stripe Connect EU sub-entity | 2h | ⬜ | `docs/data-residency.md` (new) | n/a (documentary) | Manual vendor-dashboard verification + screenshots. |
| B.9 | 2FA TOTP for admin role (RGPD art. 32 "raisonnable") | 16h | ⏸️ | follow-up — add `users.totp_secret_encrypted` + endpoints + UI | enrollment + verification tests | Use `pquerna/otp` Go lib. Deferred — not strict GDPR mandate; ship after Phase C. |
| B.10 | Sanitize `audit_logs.metadata.email` for unknown users (hash on `auth.login_failure`) | 2h | ⬜ | `app/auth/service.go:373-374` | service test | When `users.id` not found, store `email_hash` instead of cleartext. |

---

## Phase C — Documentary (target: 1.5 weeks, ~83h)

Source: `gdpr-audit.md` Section 13 Phase C.

| # | Item | Effort | Status | Files | Tests | Notes |
|---|---|---|---|---|---|---|
| C.1 | Privacy policy (FR primary, EN secondary) — full content | 16h | ⬜ | `web/messages/legal/privacy.fr.mdx`, `privacy.en.mdx` | content lint (lengths, links) | Use Sections 1, 2, 3, 4, 5, 7, 9 of `gdpr-audit.md` as raw input. Replace placeholder from A.4. |
| C.2 | CGU + CGV (separate documents) | 12h | ⬜ | `web/messages/legal/cgu.fr.mdx`, `cgv.fr.mdx` (+EN) | content lint | French lawyer review recommended. |
| C.3 | Mentions légales (LCEN art. 6 III) | 1h | ⬜ | `web/messages/legal/legal.fr.mdx` | content lint | Editor identity, RCS/SIRET, capital, hosting Vercel + Railway. Awaits user-supplied legal info. |
| C.4 | Cookies page (full vendor list + duration + purpose) | 4h | ⬜ | `web/messages/legal/cookies.fr.mdx` | content lint | Driven by Tarteaucitron service config — keep in sync. |
| C.5 | `docs/registre-traitements.md` (art. 30) | 4h | ⬜ | new file | n/a | Auto-build from Section 1 of `gdpr-audit.md`. |
| C.6 | `docs/aipd-ai-moderation.md` + `docs/aipd-search-ranking.md` (full AIPD) | 16h | ⬜ | 2 new files | n/a | Required by ≥3 art.35 triggers being active. |
| C.7 | Sign DPAs with all 21 sub-processors | 16h | ⬜ | `docs/dpa/<vendor>.md` × 21 | n/a | Manual signature workflow — agent surfaces what to sign, user signs. |
| C.8 | `docs/runbook-violation.md` (72h CNIL notification procedure) | 4h | ⬜ | new file | n/a | Internal ops runbook. |
| C.9 | `docs/runbook-droits-rgpd.md` (rights exercise workflow + 1mo deadline) | 4h | ⬜ | new file | n/a | Internal ops runbook. |
| C.10 | Translate privacy policy + CGU + cookies into EN | 6h | ⬜ | `*.en.mdx` files | content lint | Depends on C.1, C.2, C.4. |

**Grand total: ~158h ≈ 3 weeks of focused dev + legal review.**

---

## Cross-cutting / continuous

- **Sub-processor list maintenance** (`/sous-processeurs` route) — keep in sync with the 21 vendors enumerated in `gdpr-audit.md` Section 2. Update on every new vendor integration.
- **DPA tracker** — manual signatures by user, agent surfaces what to sign and stores PDFs/links in `docs/dpa/`.
- **Memory hygiene** — every commit referencing this roadmap updates the relevant row's status + brief implementation note (commit SHA, key files).
- **Mobile parity** — cookies + consent banner are web-only for now (the Flutter app does not currently ship analytics that require consent). Privacy policy + cookie equivalent in Flutter is flagged in Phase C as a follow-up (see `gdpr-audit.md` Annex C).
- **Pre-publish checklist** — see `gdpr-audit.md` Section 12 "Pre-publish checklist" — must be ticked end-to-end before any public privacy policy launch.

---

## Manual steps for the user (post-merge)

After every Phase A/B/C agent dispatch, the user must:

1. **Migrations** — apply on shared DB after merge: `DATABASE_URL=<shared> make migrate-up`.
2. **Env vars** — set on Vercel + Railway as documented in each agent's PR description.
3. **Vendor dashboards** — verify the toggles flagged in `gdpr-audit.md` Section 2 verification table (OpenAI no-training, Anthropic ZDR, AWS EU region, PostHog EU, GA4 Region 1 + IP truncation + Signals OFF, Stripe Ireland for Connect EU, Neon EU, Vercel/Railway EU when possible, R2 EU bucket).
4. **DPAs** — sign each vendor DPA from `gdpr-audit.md` Section 2 and store the signed PDF (or vendor URL) in `docs/dpa/<vendor>.md`.
5. **Legal placeholders** — review and complete the `/legal` mentions légales fields (RCS, SIRET, capital, address) once the legal entity is registered.

---

## Implementation log

| Date | Phase | Item | Commit | Files | Notes |
|---|---|---|---|---|---|
| 2026-05-10 | A | Roadmap created | `d07d4e28` | `gdpr-roadmap.md` | Initial Phase A/B/C tracker built from `gdpr-audit.md`. |
| 2026-05-10 | A.4+A.5 | Legal placeholder routes + DPO contact | `23b051e7` | 14 files (6 pages, LegalShell, LegalFooter, dpo.ts, fr/en.json, public layout) | 9 vitest tests pass. Pipeline tsc clean (pre-existing e2e errors only); 2503/2503 vitest pass. |
| 2026-05-10 | A.3 | consent_log table + endpoint | `61aa371a` (backend) + `dfa671a1` (web) | migration 139, domain/consent, app/consent, port + adapter, handler, routes, web posthog-consent.ts | 13 tests pass. `go build`/`vet` clean; full domain+app+handler suite green. |
| 2026-05-10 | A.1 | payment_info plaintext drop | n/a (already done in migration 042) | n/a | Verified: `payment_info` table no longer exists; no live code reads the legacy bank columns. |
| 2026-05-10 | A.2 | Tarteaucitron CMP | DEFERRED | n/a | Current banner already gates PostHog + GA4 correctly. Granular-toggle upgrade scheduled for a follow-up dispatch. consent_log plumbing already in place — Tarteaucitron will only replace the banner UI. |
| 2026-05-10 | B.1 | Retention scheduler (5 tables) | _branch feat/gdpr-b1-retention_ | migrations 140/141/142, `domain/retention`, `app/retention`, `port/repository/retention.go`, `adapter/postgres/retention_repository.go`, `cmd/api/wire_retention.go`, plus `device_tokens.last_seen_at` plumbing in notification adapter+service+worker | 23 unit + 5 integration tests pass; build/vet/test pipeline green; isolated DB schema verified. R2 cold-tier (B.2 archive-to-cold-storage) deferred — current archive lands in `audit_logs_archive` secondary table which already satisfies storage-limitation. |
| 2026-05-10 | A.2 | CMP integration via vanilla-cookieconsent | _branch feat/cookieconsent-vanilla_ | `web/src/shared/components/analytics/cookie-consent-provider.tsx`, `web/src/shared/lib/cookie-consent-config.ts`, `web/src/styles/cookie-consent.css`, posthog-provider gated on consent, `web/messages/{fr,en}.json` (`cookieConsent.*`) | Replaces the deferred Tarteaucitron item. MIT, self-hosted (no extra sub-processor), ~15KB gzipped. Soleil v2 CSS overrides only — zero hex literals. PostHog + GA4 SDKs never initialise before the analytics category is accepted (RGPD pause-before-consent enforced by test). consent_log POST wired for accept_all/refuse_all/custom. 33 vitest cases + 4 playwright scenarios green. |
| 2026-05-10 | B.5 | Art. 22 disclosure + appeal procedure | `94dcf308` (web) + `044937f6` (backend wiring) + `e26e67e5` (handler) + `6f1329dc` (domain/app/adapter) + `873feb22` (migration 144) | migration 144, `domain/automateddecision`, `app/automateddecision`, `port/repository/automated_decision_appeal.go`, `adapter/postgres/automated_decision_appeal_repository.go`, `handler/automated_decision_appeal_handler.go` + `routes_automated_decision.go`, `cmd/api/bootstrap*.go` + `wire_router.go`, web `/decisions-automatisees` page + privacy art. 22 section + footer link, FR/EN messages, `GDPR_CONTACT_EMAIL` config field | New `POST /api/v1/me/automated-decision-appeals` endpoint persists user appeals against the 3 automated decisions; admin email best-effort with HTML escaping. 17 backend tests green; `go build ./... && go vet ./...` clean. Web: `/decisions-automatisees` server-rendered page + footer link + privacy section. Note: original B.5 row (marketing email opt-in tracking) repurposed to art. 22 per dispatch brief — marketing-email opt-in tracking shifts to a follow-up dispatch when a marketing surface ships. |
