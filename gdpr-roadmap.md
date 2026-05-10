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
| A.2 | Integrate Tarteaucitron CMP (replace current banner) | 4h | ⬜ | `web/src/shared/components/analytics/cookie-banner.tsx`, `web/src/app/[locale]/providers.tsx` | `web/e2e/cookies-tarteaucitron.spec.ts` (planned) | Self-hosted, no extra sub-processor. Stripe stays "strictly necessary" (RGPD art. 6-1-b); only PostHog + GA4 are gated. |
| A.3 | Add `consent_log` table + insert on accept/refuse | 3h | ⬜ | migration `139`, `backend/internal/domain/consent/`, `backend/internal/app/consent/`, `backend/internal/handler/consent_handler.go`, `web/src/shared/lib/posthog-consent.ts` | `consent_handler_test.go`, `consent/service_test.go` | Anonymized IP /16 + UA hash. Endpoint `POST /api/v1/consent/log`. |
| A.4 | Create placeholder routes `/privacy /cookies /legal /cgu /cgv /sous-processeurs` | 2h | ⬜ | 6 Next.js pages under `web/src/app/[locale]/(public)/` | `web/src/app/[locale]/(public)/__tests__/legal-pages.test.tsx` | Footer links + i18n + Soleil v2. Pages are placeholders (`noindex`) until Phase C content is written. |
| A.5 | DPO email designated + surfaced in footer + policy | 1h | ⬜ | `web/src/shared/components/layouts/footer.tsx`, `web/src/app/[locale]/(public)/privacy/page.tsx`, `messages/{fr,en}.json` | covered by A.4 footer test | Default `hassad.smara69@gmail.com` initially; configurable via `NEXT_PUBLIC_DPO_EMAIL`. |

**Phase A completion criteria:** all rows ✅ or ✳️, validation pipeline green, branch pushed, manual steps documented for the user.

---

## Phase B — Compliance core (target: 1 week, ~56h)

Source: `gdpr-audit.md` Section 13 Phase B.

| # | Item | Effort | Status | Files | Tests | Notes |
|---|---|---|---|---|---|---|
| B.1 | Retention scheduler — `notifications` (90d), `device_tokens` (60d inactivity), `password_resets` (24h post-expiry), `search_queries` (12mo → anonymize user_id), `message_history` (align w/ messages) | 8h | ⬜ | `backend/internal/app/retention/scheduler.go` (new), wire from `cmd/api/wire_late_handlers.go`, config knob `RETENTION_INTERVAL` | `app/retention/scheduler_test.go`, integration test on each table | Mirror `app/gdpr/scheduler.go`. Use `LIMIT 5000` per batch + `RETURNING id` for observability. |
| B.2 | `audit_logs` cold-storage archiving (12mo → R2 jsonl.gz then DELETE) | 6h | ⬜ | `backend/internal/app/retention/audit_archive.go`, R2 path `audit-archive/<yyyy-mm>/...jsonl.gz` | archival_test.go | Introduce dedicated `marketplace_archiver` DB role with `INSERT, SELECT, DELETE` on `audit_logs`. |
| B.3 | Art. 22 "human review" — appeal CTA on auto-rejected moderation | 6h | ⬜ | `domain/moderation` (add `Appealable` flag), new endpoint `POST /api/v1/moderation/{id}/appeal`, web "Demander une revue humaine" button | handler test + service test | Audit log records `moderation.appeal_requested`. Hooks into `/admin/moderation`. |
| B.4 | Art. 21 "object" — hide-from-search opt-out | 6h | ⬜ | migration `users.search_indexed BOOLEAN NOT NULL DEFAULT true`, `Indexer.Sync` skip + Typesense `Delete`, settings UI toggle | indexer test + handler test | Toggle persists user preference; export still includes the user in their own data. |
| B.5 | Marketing email opt-in tracking (timestamps, IP) | 4h | ⬜ | extend `consent_log` from A.3 | migration test + handler test | Placeholder until a marketing email surface is added; uses `consent_type='marketing_email'`. |
| B.6 | Idempotency-Key + ratelimit on `request-deletion` (3 req/min cap) | 2h | ⬜ | `backend/internal/handler/middleware/ratelimit.go` config | unit test on rate cap | Already idempotent by design — only the rate cap needs tightening. |
| B.7 | Wire dedicated `marketplace_app` DB role with `INSERT+SELECT-only` on `audit_logs` | 4h | ⬜ | infra (Railway/Neon dashboards) + `DATABASE_URL` env update | CI smoke test | Manual infra step; document in `MIGRATION_KYC_EMBEDDED.md`-style runbook. |
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
| 2026-05-10 | A | Roadmap created | (this commit) | `gdpr-roadmap.md` | Initial Phase A/B/C tracker built from `gdpr-audit.md`. |
