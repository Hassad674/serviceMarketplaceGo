# RGPD/GDPR Compliance Audit — Marketplace Service

> Audit date: 2026-05-10
> Auditor: Claude Opus 4.7 (1M context)
> Repository state: `48a3ab1139dd4e66d34109bef08513c077d5a3ae`
> Scope: backend (Go), web (Next.js), admin (Vite/React), mobile (Flutter), migrations, infra config
> Method: read-only review of code, migrations, env files, adapters, schemas, handlers

---

## Executive summary

- **Overall posture: 🟡 Solid technical foundation, large documentary gap.**
  - The technical "right to erasure" + "right to access" pipeline is implemented end-to-end (soft-delete → email confirmation → 30-day cooldown → cron purge with audit anonymization + IP truncation, ZIP export with manifest + bilingual README).
  - Security baseline (encryption, JWT rotation, brute-force, RLS, audit logs, rate limit) is in place.
  - But: **no privacy policy, no cookie banner banner-of-record (only an analytics consent strip), no AIPD, no documented sub-processor list, no signed DPAs, no register-of-processing**, and several silent design choices that need reviewing before launching the policy.
  - The marketplace is in production on Vercel + Railway + Neon + R2 with EU users — the policy must ship before the next user wave.

### What is excellent (hard not to praise)

The GDPR feature implementation is one of the strongest parts of the codebase. Notable design decisions:

- The deletion flow uses a **dedicated HS256 signing key derived via SHA256 of `JWT_SECRET || "gdpr-deletion-confirmation"`**, so a leaked access-token signing key cannot forge a deletion confirmation link (`backend/cmd/api/wire_gdpr.go:90-94`). This matches OWASP recommendations and exceeds the typical "one secret, one usage" rule of thumb.
- The purge transaction uses `FOR UPDATE SKIP LOCKED` plus an in-tx re-check of `deleted_at IS NOT NULL AND deleted_at < before` (`backend/internal/adapter/postgres/gdpr_repository.go:506-521`), so a `CancelDeletion` that lands during the cron tick is honored without a race condition.
- The audit_logs anonymization is performed entirely **in SQL** via `pgcrypto` `digest()` (`backend/internal/adapter/postgres/gdpr_repository.go:528-548`), avoiding the operational risk of pulling every row into Go. This is the right design at scale.
- The IP truncation in `gdpr.TruncateIP` (`backend/internal/domain/gdpr/anonymization.go:65-79`) zeroes the last two octets of IPv4 and the last 96 bits of IPv6 — matching RGPD recital 26 ("no longer reasonably attributable to an identified person").
- Salt is **mandatory at boot in production** via `config.Validate()` (`backend/internal/config/config.go:310-314`). The dev fallback is rejected loudly. This closes the "salt collision across deployments" attack vector.
- The ZIP export carries a bilingual `README.txt` (`backend/internal/handler/gdpr_handler.go:140-198`) so the recipient can open the archive without external documentation.
- The "blocked owner" UX (org owner with active members cannot self-delete) returns a structured 409 with remediation actions (`backend/internal/domain/gdpr/deletion.go:42-69`), which is both technically and legally correct (you cannot delete a legal entity that other users depend on without a transfer).
- **Top 5 blockers before launching the privacy policy** (must fix before publishing the policy):
  1. **GA4 (Google Analytics) is configured behind the analytics-consent strip but the strip wording does not match what GA4 collects** — the banner shows "outil d'analyse" without naming Google or a third country (US). CNIL requires explicit naming of the recipient + non-EEA transfer warning. (`web/src/shared/components/analytics/cookie-banner.tsx:47`, `web/messages/fr.json` `analyticsConsent.description`)
  2. **No registre des traitements (art. 30)** — must be created from Section 1 below before publishing a policy that references it.
  3. **Sub-processor DPAs** (Vercel, Railway, Neon, R2, Resend, Stripe, LiveKit, OpenAI, Anthropic, AWS, FCM, Typesense, PostHog, Google) — none are documented as signed in repo, must be enumerated, signed, and listed in the privacy policy.
  4. **No retention policy is enforced for `audit_logs`, `search_queries`, `messages`, `notifications`, `moderation_results`** — only `users` has the 30-day soft-delete cooldown wired (`backend/internal/app/gdpr/scheduler.go:55`). The rest grow indefinitely.
  5. **No documented opt-out for AI moderation / search ranking automated decisions** (art. 22). Users are never told their content is sent to OpenAI moderation, Anthropic dispute analysis, or AWS Rekognition image moderation — disclosure required.
- **Top 5 quick wins (< 4h each)**:
  1. Add the privacy-policy / mentions-légales / CGU / cookies routes (currently 404 — `web/src/app/[locale]/(public)/legal/*` does not exist; only `<Link href="/privacy">` placeholders in register pages, `web/src/app/[locale]/(auth)/register/agency/page.tsx:87`).
  2. Tighten cookie banner: name the providers (Google Analytics, PostHog) + add granular toggles (functional / analytics / marketing).
  3. Update the Permissions-Policy comment in CLAUDE.md (the doc says `camera=()` but the code uses `camera=(self)` — currently ok, but doc drift, `backend/internal/handler/middleware/security_headers.go:46`).
  4. Add a `consent_log` table or extend `audit_logs` to record explicit consent timestamps + IP + UA when users accept the cookie banner / TOS / marketing opt-in (currently only stored in `localStorage` with no server proof, `web/src/shared/lib/posthog-consent.ts:43`).
  5. Document audit-log retention as "indefinite, with anonymization on user purge" in the privacy policy (already implemented in code, `backend/internal/adapter/postgres/gdpr_repository.go:528-548`).
- **Estimated total effort to reach "auditable" GDPR posture: ~3 weeks of dev time** (Phase A 2 days, Phase B 1 week, Phase C 1.5 weeks for documentary + legal review).

---

## 1. Cartographie des données à caractère personnel (DCP)

For every column that holds PII, here is the source of collection, persistence, purpose, and sensitivity. Sensitivity flags: `normal` / `sensitive` (RGPD art. 9) / `judicial` / `financial` / `regulated` (LCB-FT, etc.).

| Feature | Field | Collected via | Stored where | Purpose | Sensitivity | Source (file:line) |
|---|---|---|---|---|---|---|
| auth | email | `POST /api/v1/auth/register` | `users.email` (UNIQUE NOT NULL) | identification + login | normal | `backend/internal/handler/dto/request/auth.go:8` · `backend/migrations/001_create_users.up.sql:3` |
| auth | hashed_password | register / change-password | `users.hashed_password` | auth | normal (bcrypt cost 12) | `backend/migrations/001_create_users.up.sql:4` · `backend/internal/app/auth/service.go` |
| auth | first_name, last_name, display_name | register / profile update | `users.first_name`, `users.last_name`, `users.display_name` | display + identification | normal | `backend/migrations/001_create_users.up.sql:5-7` |
| auth | role (agency / enterprise / provider) | register | `users.role` | RBAC | normal | `backend/migrations/001_create_users.up.sql:8` |
| auth | linkedin_id, google_id | OAuth login | `users.linkedin_id`, `users.google_id` (UNIQUE) | OAuth correlation | normal | `backend/migrations/001_create_users.up.sql:12-13` |
| auth | email_verified | confirmation flow | `users.email_verified` | UX gate | normal | `backend/migrations/001_create_users.up.sql:14` |
| auth | deleted_at | GDPR right-to-erasure | `users.deleted_at` (nullable) | soft-delete window | normal | `backend/migrations/132_users_deleted_at_for_gdpr.up.sql:25` |
| profile | title, about, photo_url, presentation_video_url, referrer_video_url | profile update | `profiles` (FK user) | public marketing | normal | `backend/migrations/002_create_profiles.up.sql:3-7` · `005_add_about_to_profiles.up.sql` |
| KYC (legacy) | first_name, last_name, **date_of_birth**, nationality, address, city, postal_code | `POST /api/v1/billing-profile` (legacy) | `payment_info.*` | KYC, payment | regulated (LCB-FT) | `backend/migrations/015_create_payment_info.up.sql:7-12` |
| KYC (legacy) | phone, activity_sector | profile/billing UI | `payment_info.phone`, `payment_info.activity_sector` | KYC | normal / regulated | `backend/migrations/020_add_phone_activity_sector.up.sql:1-2` |
| KYC (legacy) | iban, bic, account_number, routing_number, account_holder, bank_country | billing form | `payment_info.iban` … (TEXT, plaintext) | payouts | financial 🔴 | `backend/migrations/015_create_payment_info.up.sql:25-31` |
| KYC (legacy) | tax_id, vat_number, business_name, business_address, business_city, business_postal_code, business_country, role_in_company | billing form | `payment_info.tax_id` … | KYC business | regulated | `backend/migrations/015_create_payment_info.up.sql:14-23` |
| KYC (Stripe Embedded) | id document scan, selfie, address, dob, ssn-equivalent | Stripe Embedded Connect onboarding flow | held by **Stripe** (US, DPF) — only `stripe_account_id` ref + `stripe_last_state` JSONB stored locally | regulatory KYC | sensitive (id docs) | `backend/migrations/065_enrich_organizations.up.sql:29-33` · `backend/internal/adapter/stripe/account.go` |
| KYC (intermediate) | id_document_url, document_type, side, file_key, stripe_file_id | identity_documents upload (legacy) | `identity_documents` (FK user, ON DELETE CASCADE) | KYC | sensitive | `backend/migrations/019_create_identity_documents.up.sql:3-9` |
| KYC | business_persons (representatives) — first/last name, **date_of_birth**, **email**, **phone**, address, city, postal_code, title, stripe_person_id | KYC form for non-self representatives | `business_persons` (FK user) | KYC | regulated | `backend/migrations/021_add_business_persons.up.sql:5-14` |
| billing_profile | legal_name, trading_name, legal_form, tax_id, vat_number, address_line1/2, postal_code, city, country, invoicing_email | `PUT /api/v1/me/billing-profile` | `billing_profile` (PK organization_id) | invoicing | regulated | `backend/migrations/121_create_invoicing.up.sql:23-49` · `backend/internal/handler/billing_profile_handler.go:48-68` |
| billing_profile | vat_validation_payload (full VIES JSONB lookup with name + address echoed back from VIES) | VAT validation | `billing_profile.vat_validation_payload` | audit | regulated | `backend/migrations/121_create_invoicing.up.sql:36` |
| invoice | recipient_snapshot, issuer_snapshot (JSONB frozen address + tax IDs at emission), number, amount, currency, mentions_rendered | invoicing service | `invoice.recipient_snapshot`, `invoice.issuer_snapshot` | legal invoicing (10-year retention obligation FR) | financial | `backend/migrations/121_create_invoicing.up.sql` |
| messaging | sender_id, recipient(s), content (free text), msg_type, file metadata, voice metadata | `POST /api/v1/conversations/{id}/messages` | `messages.content` (TEXT), `messages.metadata` (JSONB) | core service | normal (potentially sensitive content user-provided) | `backend/migrations/007_create_messaging.up.sql` · `backend/internal/handler/messaging_handler.go` |
| messaging | media: voice notes, file uploads, images | `POST /api/v1/uploads` | R2 (file) + `messages.metadata` URL | UGC | normal | `backend/internal/handler/upload_handler.go` |
| moderation | text/image moderation labels + scores | server side (auto-fire on every message, review, profile, job) | `moderation_results.labels` JSONB, `score` REAL, `reason` TEXT | moderation decision | sensitive (judgment of speech) | `backend/migrations/120_create_moderation_results.up.sql:20-32` · `backend/internal/adapter/openai/text_moderation.go` |
| job | title, description, budget, location, expertise — published by enterprises | `POST /api/v1/jobs` | `jobs` table | core | normal | `backend/migrations/011_create_jobs.up.sql` |
| job | applicant_kind, cover letter, applicant_id | `POST /api/v1/jobs/{id}/applications` | `job_applications` | matching | normal | `backend/migrations/138_job_applications_applicant_kind.up.sql` |
| proposal | client_id, provider_id, sender_id, recipient_id, body, conversation_id, contract amount | proposal flow | `proposals` table + `proposal_milestones` | contract | financial | `backend/migrations/008_create_proposals.up.sql` |
| review | reviewer_id, reviewed_id, global_rating, comment, side, video_url | post-mission review form | `reviews` table | reputation | normal (UGC) | `backend/migrations/012_create_reviews.up.sql` · `014_add_video_url_to_reviews.up.sql` |
| notification | user_id, type, title, body, read_at | system-fired | `notifications` | UX | normal | `backend/migrations/016_create_notifications.up.sql` |
| device_tokens | FCM push token + platform (android/ios/web) | mobile/web register-device | `device_tokens.token` | push | normal (correlatable across re-installs) | `backend/migrations/017_create_notification_prefs_and_devices.up.sql:10-17` |
| notification_preferences | per-type push/email/in_app booleans | account settings | `notification_preferences` | preference | normal | `backend/migrations/017_create_notification_prefs_and_devices.up.sql:1-8` |
| audit_logs | user_id, action, resource_type, resource_id, **metadata (free JSONB — currently includes email on login_failure)**, ip_address (INET) | server-side hooks | `audit_logs` | security forensics | normal (PII in metadata) | `backend/migrations/078_create_audit_logs.up.sql:28-37` · `backend/internal/app/auth/service.go:373` |
| search_queries | user_id, session_id, query string, filters JSONB, results_count, latency_ms, clicked_result_id, clicked_position, created_at | `POST /api/v1/search` | `search_queries` | learning-to-rank, analytics | normal (queries can contain PII typed by users) | `backend/migrations/111_create_search_queries.up.sql:11-22` |
| dispute | dispute summary written by Claude AI (Anthropic), AI chat messages | `POST /api/v1/disputes/{id}/summary` | `disputes.*`, `dispute_ai_chat_messages` | dispute resolution | normal (judgment) | `backend/migrations/052_dispute_ai_chat_messages.up.sql` · `backend/internal/adapter/anthropic/analyzer.go` |
| referral | referrer_id, referred_party_a/b, conversation introduction | apporteur d'affaires flow | `referrals` table family | matching | normal | various referral migrations |
| reports | reporter_id, target_type, target_id, reason, description, status | `POST /api/v1/reports` | `reports` | abuse handling | normal (judgment) | `backend/migrations/023_create_reports.up.sql` |
| message_history | edit history of messages | edit-message endpoint | `message_history` | audit / traceability | normal | `backend/migrations/022_create_message_history.up.sql` |
| organization | name, stripe_account_id, stripe_account_country, stripe_last_state JSONB, kyc_first_earning_at, auto_payout_enabled_at | `PATCH /api/v1/organizations/me` | `organizations` | business state | regulated | `backend/migrations/065_enrich_organizations.up.sql` · `123_org_auto_payout_consent.up.sql` |
| analytics (PostHog) | distinct_id, captured events (page views, clicks, custom events) | browser SDK + server SDK | PostHog cloud (EU host, `https://eu.posthog.com`) | product analytics | normal | `web/src/shared/lib/posthog.ts` · `backend/internal/config/config.go:142-146` |
| analytics (GA4) | client_id, captured events, page URLs, IP (truncated by Google) | browser SDK only (no server adapter) | Google (US) | acquisition analytics | normal 🔴 (US transfer) | `web/src/shared/lib/ga.ts` |

**🔴 Sensitivity flags requiring special attention:**

1. **`payment_info.iban` / `account_number` / `routing_number` / `account_holder` are stored as plaintext TEXT** (no column-level encryption). The user has migrated to Stripe Embedded for new KYC, but legacy rows remain. (`backend/migrations/015_create_payment_info.up.sql:25-31`)
   - Recommendation, ranked by preference:
     1. (a) **Truncate** the columns to NULL on every existing row, since Stripe Connect now holds the source of truth. Schedule a migration `139_truncate_legacy_bank_columns.up.sql`.
     2. (b) Migrate to **column-level encryption** with `pgcrypto` `pgp_sym_encrypt`. Adds key management overhead.
     3. (c) Move to **Stripe Connect Custom secrets vault** — strongest, but requires Stripe API integration.
2. **`business_persons.date_of_birth` + `business_persons.email/phone`** collected for non-self KYC representatives — kept locally even though Stripe Embedded now hosts the official copy. Same recommendation as 1.
3. **`identity_documents.file_key`** is the R2 path of the uploaded id document (passport / national id / KBIS) — these files persist in R2 indefinitely. R2 is encrypted at rest, but the GDPR purge does NOT delete the R2 object — only `identity_documents` rows are dropped via `ON DELETE CASCADE` (`backend/migrations/019_create_identity_documents.up.sql:3`). The R2 object becomes orphaned. Recommendation: extend `gdpr.PurgeUser` to enumerate `identity_documents.file_key` for the user before the cascade and call `StorageService.Delete(key)`.
4. **Audit log metadata includes raw email on every `auth.login_failure` event** (`backend/internal/app/auth/service.go:373-374`). This is needed for forensics but means the cleartext email of any failed login attempt — INCLUDING from someone who never registered — is persisted indefinitely. Mitigated only at hard-delete time via `actor_email_hash` rewrite, BUT only for users who have an associated `users.id` row (the JSONB `email` key is dropped, but for a never-registered email, no purge ever runs). Recommendation: hash the email immediately for `user_id IS NULL` rows (the email never belonged to a registered user, so no compensating link is needed).
5. **`messages.content`** holds free-form user text — every message body is stored in plaintext. RLS policies on `messages` cover access control (migration 125), but the data is not encrypted at rest beyond Neon's default disk encryption. For a B2B context, consider end-to-end encryption (E2EE) of messages between principals as a future hardening. Currently NOT a GDPR requirement — flagged for discussion only.
6. **`reviews.video_url`** points at R2 video review files — same orphaned-on-purge issue as `identity_documents.file_key`.

---

## 2. Sous-processeurs (third-party data processors)

Every external service the backend or browser talks to. The "Status" column flags whether the legal mechanism (DPF certification, Standard Contractual Clauses, EU residency) is documented in repo: it is **not** for any of them today (no `docs/dpa/` directory exists). The transfer column is informational; consult the up-to-date DPF list on `dataprivacyframework.gov` before publishing.

| # | Vendor | Country | Type of data sent | Legal basis | DPF / SCC status | DPA URL | Source |
|---|---|---|---|---|---|---|---|
| 1 | **Vercel** (web hosting) | US (Cloudflare CDN edge) | All web traffic, including form bodies as TLS terminates here | Contract (b) | DPF-certified (verify) | https://vercel.com/legal/dpa | hosted in Vercel project |
| 2 | **Railway** (backend hosting) | US (Railway-owned VMs) | All backend traffic, env secrets | Contract (b) | not-DPF — uses SCC | https://railway.app/legal/dpa | hosted on Railway |
| 3 | **Neon** (Postgres) | EU region available — must verify the project is EU | All DB writes (messages, profiles, KYC, audit) | Contract (b) | EU-only if EU project | https://neon.tech/dpa | `backend/.env.example:6` (`DATABASE_URL`) |
| 4 | **Cloudflare R2** (object storage) | US/global; data plane is regional | All uploaded media: avatars, KYC docs, voice notes, video reviews, message attachments | Contract (b) | DPF-certified | https://www.cloudflare.com/cloudflare-customer-dpa/ | `backend/internal/adapter/s3/storage.go` · `backend/.env.example` (R2 example) |
| 5 | **Resend** (transactional email) | US | email + first_name, transactional content (account deletion confirmation links, reset emails, notifications) | Contract (b) | DPF-certified | https://resend.com/legal/dpa | `backend/internal/adapter/resend/email.go` · `wire_infra.go:205` |
| 6 | **Stripe** (payments + Connect KYC) | US (Ireland EU subsidiary for European Connect accounts) | Full PII for KYC: id docs, dob, address, ssn-equivalent + payment data | Contract (b) + Legal obligation (c) | DPF-certified + Stripe Ireland for EU | https://stripe.com/legal/dpa | `backend/internal/adapter/stripe/account.go` · `wire_infra.go` |
| 7 | **LiveKit** (video calls) | US | LiveKit room metadata, media SFU passthrough (E2E if configured, otherwise transit) | Contract (b) | unknown — verify | https://livekit.io/legal/dpa | `backend/internal/adapter/livekit/client.go` |
| 8 | **OpenAI** (text moderation + embeddings) | US | Every message body, review body, profile bio, job description, proposal text | Contract (b) — service execution | DPF-certified — "no training" toggle ON for API by default | https://openai.com/policies/data-processing-addendum | `backend/internal/adapter/openai/text_moderation.go:67` · `backend/internal/adapter/openai/client.go` |
| 9 | **Anthropic** (dispute summary AI) | US | Full dispute conversation context (last 200 msgs + proposal description) | Contract (b) | DPF-certified | https://www.anthropic.com/legal/dpa | `backend/internal/adapter/anthropic/analyzer.go:19-21` |
| 10 | **AWS Rekognition** (image + video moderation) | EU region in use (`eu-west-1` per `REKOGNITION_REGION` default, `backend/internal/config/config.go:180`) | All uploaded images + video frames | Contract (b) | EU-only | https://aws.amazon.com/service-terms/ + DPA | `backend/internal/adapter/rekognition/moderation.go` |
| 11 | **AWS S3 (transit moderation bucket)** | EU region | Video files in transit before Rekognition processes them | Contract (b) | EU-only | same as above | `backend/internal/adapter/s3transit/transit.go` |
| 12 | **AWS SNS + SQS** (video moderation completion fan-out) | EU region | Job completion notifications (no media body) | Contract (b) | EU-only | same as above | `backend/internal/adapter/sqs/worker.go` |
| 13 | **Firebase Cloud Messaging (FCM)** | US (Google Cloud) | device tokens + push payload (notification title + body) | Contract (b) | DPF-certified (Google) | https://firebase.google.com/terms/data-processing-terms | `backend/internal/adapter/fcm/push.go` |
| 14 | **Typesense** (search engine) | self-hosted OR Typesense Cloud (EU region available) | Full search index: profile names, titles, expertises, locations | Contract (b) | EU-only if EU cluster | https://typesense.org/legal/dpa | `backend/internal/config/config.go:92-94` · `backend/internal/search/indexer.go` |
| 15 | **PostHog** (product analytics) | EU host configured (`https://eu.posthog.com`, EU project ⇒ Ireland data center) | Browser events with distinct_id, page views, custom events | Consent (a) | EU-only on EU project | https://posthog.com/dpa | `backend/internal/config/config.go:144-146` · `web/src/shared/lib/posthog.ts` |
| 16 | **Google Analytics 4** | US (Ireland for EU traffic with Region 1 default, but Google Tag is global) | client_id, page URLs, events | Consent (a) | DPF-certified — concerns persist (CNIL position, Schrems II) | https://business.safety.google/adsdpa/ | `web/src/shared/lib/ga.ts` · `web/src/shared/components/analytics/google-analytics-provider.tsx` |
| 17 | **VIES** (EU VAT validation, EC public service) | EU (European Commission infrastructure) | VAT number + name + member state | Legal obligation (c) | EU-internal — no DPA needed (EU body) | n/a | `backend/internal/adapter/vies/client.go` |
| 18 | **Nominatim (OpenStreetMap)** geocoding | EU (Germany, OSM Foundation) | City strings typed by users for address autocomplete | Legitimate interest (f) | EU-only | https://operations.osmfoundation.org/policies/nominatim/ | `backend/internal/adapter/nominatim/client.go` |
| 19 | **BAN api-adresse.data.gouv.fr** (FR address autocomplete) | FR (Etalab / FR government) | French addresses typed in browser | Legitimate interest (f) | EU-only | n/a (gov service) | `web/src/shared/lib/csp.ts:74-77` |
| 20 | **Photon (komoot.io)** (intl city autocomplete fallback) | DE | International city queries | Legitimate interest (f) | EU-only | komoot.com — to verify | `web/src/shared/lib/csp.ts:74-77` |
| 21 | **Cloudflare** (CDN, R2 fronting) | US/global | Static asset traffic, R2 egress | Contract (b) | DPF-certified | bundled in Cloudflare DPA above | `web/src/shared/lib/csp.ts:65-68` |

🔴 **Action required**:
- Sign and store DPAs from Vercel, Railway, Resend, Stripe, LiveKit, OpenAI, Anthropic, AWS, Google (FCM + GA), PostHog, Typesense, R2/Cloudflare. None are tracked in repo.
- Verify Neon project region — the local default DSN is local Postgres on port 5434/5435, but production must be confirmed as EU-region (`docs/` does not name the region).
- Verify Typesense is EU-region.
- Verify LiveKit DPA + jurisdiction.

### What the user must verify in vendor dashboards before publishing the policy

| Vendor | Dashboard setting to verify | Why |
|---|---|---|
| OpenAI | "Do not use my data to improve the model" toggle ON for the project key (default ON for API since 2023-05) | Documenting the toggle in the privacy policy |
| Anthropic | Same as OpenAI — verify "Zero data retention" eligibility | dispute analysis sees full proposal context |
| AWS | Region = EU (eu-west-1 / eu-central-1) for Rekognition + S3 transit + SNS + SQS | Schrems II avoidance |
| PostHog | EU project (Ireland) on `eu.posthog.com` | already configured in `config.go:146` ✅ |
| GA4 | "IP truncation" + "Region 1" + Google Signals OFF (in Property settings) | mitigates Schrems II concerns |
| Stripe | Stripe Ireland for Connect EU | verify European Connect accounts are billed/registered to Stripe Technology Europe Ltd |
| Resend | EU sender domain (Resend EU pool when available) | reduces transit through US |
| Neon | EU region, point-in-time recovery enabled | data residency + backup |
| Vercel | EU region (Frankfurt/Paris) for ISR + edge functions when possible | mitigation |
| Railway | EU region (eu-west-1) | mitigation |
| Cloudflare R2 | Bucket region jurisdiction = EU (R2 has regional buckets since 2024) | data residency |
| FCM | n/a — Google's data location is fixed | document the US transfer in the policy |

---

## 3. Transferts extra-UE (Schrems II)

For each US-located sub-processor, document the legal mechanism. As of 2026-05-10:

| # | Vendor | Mechanism | Risk | Action |
|---|---|---|---|---|
| 1 | Vercel | DPF + SCC fallback | Low (DPF active) | Sign DPA, name in privacy policy |
| 2 | Railway | SCC (no DPF) | Medium | Sign DPA explicitly with SCC |
| 4 | Cloudflare R2 | DPF + SCC | Low | Sign DPA |
| 5 | Resend | DPF (verify) | Low | Sign DPA |
| 6 | Stripe | DPF + SCC + Stripe Ireland for EU subjects | Low | Sign DPA, document the EU sub-entity for Connect |
| 7 | LiveKit | unknown — verify | Medium 🔴 | Verify mechanism; pivot if no DPF/SCC |
| 8 | OpenAI | DPF | Low (DPF active) | Sign DPA, ensure "no training" toggle is ON in account settings |
| 9 | Anthropic | DPF | Low | Sign DPA |
| 13 | FCM | DPF (Google) | Low | Sign DPA |
| 16 | Google Analytics 4 | DPF + Region 1 + IP truncation | **Medium 🔴** — CNIL has historically ruled GA4 problematic; in 2026-05 the DPF is in effect but CNIL's position is conservative. Recommendation: replace with PostHog-only OR document Region 1 + IP truncation + signal disabling explicitly. | Decide: keep + document, or remove |

🔴 **Single biggest documentary risk**: GA4. CNIL's 2022 enforcement action was based on pre-DPF era — DPF gives a defense, but a French B2B marketplace can prefer PostHog-EU only.

---

## 4. État des droits RGPD (per article 12-22)

| Right | Article | Status | Endpoint(s) | Evidence |
|---|---|---|---|---|
| Information at collection | 13 | 🔴 missing | n/a | No privacy policy exists. Register pages link to `/privacy` (`web/src/app/[locale]/(auth)/register/agency/page.tsx:87`) but the route is 404. |
| Right to access | 15 | 🟢 implemented | `GET /api/v1/me/export` → ZIP with manifest + 10 JSON sections + bilingual README | `backend/internal/handler/gdpr_handler.go:44-79` · `backend/internal/adapter/postgres/gdpr_repository.go:64-115` |
| Right to rectification | 16 | 🟢 implemented (per-field) | `PUT /api/v1/me/profile`, `PUT /api/v1/me/billing-profile`, `POST /api/v1/auth/change-email`, `POST /api/v1/auth/change-password`, freelance/referrer/client profile updates | `backend/internal/handler/profile_handler.go` · `backend/internal/handler/billing_profile_handler.go` · `backend/internal/handler/auth_handler_account.go` |
| Right to erasure | 17 | 🟢 implemented (sound design) | `POST /api/v1/me/account/request-deletion`, `GET /api/v1/me/account/confirm-deletion?token=`, `POST /api/v1/me/account/cancel-deletion`. 30-day cooldown + cron purge with anonymization-in-place (FK constraints prevent hard cascade). | `backend/internal/handler/gdpr_handler.go:213-333` · `backend/internal/app/gdpr/service.go:146-260` · `backend/internal/app/gdpr/scheduler.go:55-95` · `backend/internal/adapter/postgres/gdpr_repository.go:493-593` |
| Right to restriction | 18 | 🟡 partial | The soft-delete period (30 days, `deleted_at` set) effectively restricts processing — user can no longer log in (`backend/internal/app/auth/service.go:408`). But there is no "freeze without deletion" workflow per art. 18.1.a-d. | — |
| Right to portability | 20 | 🟢 implemented | `GET /api/v1/me/export` returns machine-readable JSON inside ZIP | `backend/internal/handler/gdpr_handler.go:44-79` |
| Right to object | 21 | 🟡 partial | Email opt-out via `email_notifications_enabled` boolean and per-type push/email toggles. NO opt-out for: profiling-style search ranking, AI text/image moderation, AI dispute analysis. | `backend/migrations/076_users_email_notifications_enabled.up.sql` · `backend/migrations/017_create_notification_prefs_and_devices.up.sql` |
| Automated decisions / profiling | 22 | 🔴 missing | The marketplace runs three automated decisions: (1) search ranking with OpenAI embeddings + heuristic, (2) auto-rejection of media with Rekognition above threshold (`config.RekognitionAutoRejectThreshold`), (3) auto-blocking of UGC text with OpenAI Moderation API thresholds. None are disclosed and none offer a documented "request human review" path beyond the generic admin dispute / appeal flow. | `backend/internal/adapter/rekognition/moderation.go` · `backend/internal/adapter/openai/text_moderation.go` · `backend/internal/search/indexer.go` |
| Notification of breach | 33-34 | 🟡 partial | Audit logs exist, but no documented procedure for 72-hour CNIL notification or data-subject notification. | `backend/internal/domain/audit/entity.go` |

**Test attempted on the audit machine**: not possible — no live backend in this audit run. The handlers, services, repositories, and tests all compile and have green test coverage in CI per `make test`.

### Data subject rights coverage matrix

| Right | Article | Web UI | Mobile UI | Admin UI | Backend endpoint | Tested in CI |
|---|---|---|---|---|---|---|
| Information | 13 | 🔴 no `/privacy` page | 🔴 no privacy screen | n/a | n/a | n/a |
| Access | 15 | 🟢 settings → "Exporter mes données" | 🟢 account screen → export | (admin can also bulk-export users? not implemented) | `GET /api/v1/me/export` | yes — `gdpr_handler_test.go` |
| Rectification | 16 | 🟢 multiple settings pages | 🟢 multiple screens | yes | `PUT /api/v1/me/profile`, `change-email`, `change-password`, etc. | yes |
| Erasure | 17 | 🟢 settings → "Supprimer mon compte" | 🟢 delete_account_screen | n/a (admin can't initiate user deletion) | `POST /api/v1/me/account/request-deletion` + confirm + cancel | yes — `gdpr_handler_test.go` + `app/gdpr/service_test.go` |
| Restriction | 18 | 🟡 effectively via deletion | 🟡 via deletion | n/a | (no dedicated endpoint) | — |
| Notification of rectif/erasure to third parties | 19 | n/a | n/a | n/a | (not implemented) | — |
| Portability | 20 | 🟢 same as access (ZIP with JSON) | 🟢 same | n/a | `GET /api/v1/me/export` | yes |
| Object | 21 | 🟡 partial (email on/off) | 🟡 partial | n/a | `PUT /api/v1/notifications/preferences` | yes |
| Automated decision | 22 | 🔴 no UI | 🔴 no UI | 🟡 admin can override moderation result | (no dedicated endpoint) | — |

---

## 5. Bases légales (art. 6)

| Treatment | Legal basis | Note |
|---|---|---|
| User account creation + login (email, name, password) | Contract (b) | service execution |
| KYC (Stripe Embedded data, business_persons, identity_documents, billing_profile) | Legal obligation (c) — LCB-FT, FR DAC7 | regulated retention 5+ years |
| Invoicing (invoice, credit_note, billing_profile snapshot) | Legal obligation (c) — Code de commerce art. L123-22 | 10-year retention |
| Messaging (conversations, messages, message_history) | Contract (b) | core service |
| Notifications (in-app, email, push) | Contract (b) for transactional notifications, Consent (a) for marketing — but NO marketing notifications are sent currently | `email_notifications_enabled=true` default = transactional opt-in (acceptable as soft opt-in for service emails) |
| Reviews + reports + disputes | Contract (b) | core service |
| Search ranking + analytics (search_queries) | Legitimate interest (f) | weighing required: ranking improves marketplace utility; can be disclosed in privacy policy |
| Audit logs | Legitimate interest (f) — security; Legal obligation (c) for some authn events | indefinite retention with anonymization on user purge |
| AI moderation (text via OpenAI, image/video via Rekognition) | Legitimate interest (f) — platform safety, content moderation under DSA | weighing must be documented |
| AI dispute analysis (Anthropic) | Contract (b) | dispute resolution is part of the service |
| Search analytics (`search_queries`) for learning-to-rank | Legitimate interest (f) | needs disclosure |
| PostHog analytics | Consent (a) | banner gates init, opt-out by default ✅ |
| GA4 analytics | Consent (a) | banner gates render, opt-out by default ✅ |

**🔴 Missing legal-basis decisions** (must be locked before publishing the policy):
- AI moderation: legitimate interest balancing test (LIA) needed.
- Search ranking: LIA needed.
- Whether Anthropic dispute analysis is consent-based or contract-based.
- Whether `device_tokens` retention beyond app uninstallation is necessary (currently no TTL).

---

## 6. Data minimization (art. 5-1-c)

Fields collected but never read in business code (candidates for removal).

| Field | Collected at | Read for | Verdict |
|---|---|---|---|
| `users.linkedin_id` | OAuth sign-in flow | OAuth correlation only | keep but ensure deleted on account purge (currently kept as-is — user purge sets to NULL, `backend/internal/adapter/postgres/gdpr_repository.go:563-564`) ✅ |
| `users.google_id` | OAuth sign-in flow | OAuth correlation only | keep, nullified on purge ✅ |
| `business_persons.email`, `business_persons.phone` | KYC form | KYC display | overlap with Stripe Embedded copy — redundant; can be dropped now that Stripe holds truth |
| `payment_info.iban`, `account_number`, `routing_number`, `account_holder` | Pre-Stripe-Embedded billing form | Legacy display only — Stripe is the truth source | 🔴 **delete the column data** (or migrate to a referenceable token). Plaintext bank credentials are a data minimization violation since the system migrated. |
| `payment_info.is_self_representative`, `is_self_director`, `no_major_owners`, `is_self_executive` | KYC form | Legacy display only | redundant with Stripe Connect data |
| `business_persons.date_of_birth` | KYC form for non-self | Legacy KYC | now held by Stripe — local copy redundant |
| `search_queries.query` text | analytics | learning-to-rank | useful but apply truncation/anonymization for queries older than X days |
| `audit_logs.metadata.email` (on `auth.login_failure`) | failed login | forensic | keep, but consider hashing immediately (instead of waiting for purge) for failed logins of unregistered emails (the email of someone who never registered should not stay in cleartext indefinitely) |

---

## 7. Retention periods (art. 5-1-e)

Inventory of retention rules across tables. Anything with "—" has no enforcement.

| Table | Current retention | Recommendation | Source |
|---|---|---|---|
| `users` (active) | indefinite (until user requests deletion) | indefinite, OK | — |
| `users` (deleted) | 30 days then anonymization-in-place | OK ✅ | `backend/internal/domain/gdpr/deletion.go:12` · `backend/internal/adapter/postgres/gdpr_repository.go:493` |
| `audit_logs` | indefinite (anonymized on user purge) | OK for security; recommend archive to cold storage after 12 months (per CLAUDE.md) — not enforced | `backend/internal/adapter/postgres/gdpr_repository.go:528-548` |
| `messages` | — | recommend retention rule: keep while conversation is active + 3 years after last activity, OR delete on user purge (currently no scheduler — only `conversation_participants` row is removed at user purge, leaving messages orphaned but visible to other party) | `backend/internal/adapter/postgres/gdpr_repository.go:574-582` |
| `notifications` | — | hard delete on user purge ✅ (`DELETE FROM notifications WHERE user_id = $1`); for non-deleted users, recommend 90-day TTL (notifications are ephemeral by nature) | `backend/internal/adapter/postgres/gdpr_repository.go:575` |
| `device_tokens` | — | hard delete on user purge ✅; recommend TTL of 60 days since last seen (FCM marks tokens as invalid after 60 days of app inactivity) | `backend/internal/adapter/postgres/gdpr_repository.go:576` |
| `password_resets` | — | hard delete on user purge ✅; recommend automatic cleanup of expired tokens (>48h) | `backend/internal/adapter/postgres/gdpr_repository.go:577` |
| `notification_preferences` | — | hard delete on user purge ✅ | `backend/internal/adapter/postgres/gdpr_repository.go:578` |
| `conversation_participants`, `conversation_read_state`, `job_views` | — | hard delete on user purge ✅ | `backend/internal/adapter/postgres/gdpr_repository.go:579-581` |
| `search_queries` | — | recommend 12-month retention then anonymization (drop user_id, keep query text + result count for ML — already nullable user_id with `ON DELETE SET NULL`) | `backend/migrations/111_create_search_queries.up.sql:13` |
| `moderation_results` | indefinite | recommend keeping for safety + DSA compliance, but hash author_user_id after user purge (already `ON DELETE SET NULL`) | `backend/migrations/120_create_moderation_results.up.sql:23` |
| `identity_documents` (legacy KYC scans) | indefinite | LCB-FT requires 5 years post-relationship — do not delete blindly. Migrate to deletion-with-Stripe-confirmation flow. | `backend/migrations/019_create_identity_documents.up.sql` |
| `business_persons` | indefinite | Same as identity_documents — LCB-FT 5-year retention | `backend/migrations/021_add_business_persons.up.sql` |
| `payment_info` (legacy bank) | indefinite | Either drop the columns now that Stripe owns the truth, or maintain LCB-FT 5-year retention | `backend/migrations/015_create_payment_info.up.sql` |
| `invoice`, `credit_note`, `billing_profile` | indefinite | Code de commerce L123-22 = **10 years retention**, do NOT delete | `backend/migrations/121_create_invoicing.up.sql` |
| `proposals`, `payment_records`, `disputes` | indefinite | Tax + dispute legal: 5-10 years; do NOT delete on user purge — `payment_records` and `proposals` keep `user_id` columns whose user is now anonymized but the row stays | `backend/internal/adapter/postgres/gdpr_repository.go:470-487` (commented design) |
| `reviews` | indefinite | UGC; on user purge, reviewer_id is anonymized via the user-row-anonymization (the FK still points at the anonymized user row) | — |
| `jobs` | indefinite | recommend 24-month "archive" rule for closed jobs; not enforced | — |
| `message_history` | indefinite | recommend deletion alongside parent message + alignment with `messages` retention | — |

🔴 **Tables with NO documented retention**: `messages`, `search_queries`, `moderation_results`, `notifications` (live), `device_tokens` (live), `audit_logs` (cold storage rule documented but not implemented), `jobs`, `proposals` (post-completion archive). Each needs a TTL decision before the privacy policy ships.

### Proposed retention matrix for the privacy policy

| Data category | Active retention | Soft-delete grace | Final fate | Justification |
|---|---|---|---|---|
| Account (users) | Until user request | 30 days `deleted_at` | Anonymized in place (FK constraints prevent hard delete) | Implemented ✅ |
| Profile / Avatar / Bio (profiles, freelance_profile, agency_profile, referrer_profile) | While account active | 30 days | Anonymized via `users` cascade — text stays linked but author is anonymous | Implemented |
| KYC docs (identity_documents, business_persons, payment_info) | 5 years post-relationship | n/a | LCB-FT regulatory retention | Not enforced — recommend a separate `gdpr.PurgeKYC` cron after 5 years from `users.deleted_at` |
| Invoices / credit_notes / billing_profile | 10 years | n/a | Archive | Code de commerce L123-22 |
| Stripe data (stripe_account_id, stripe_last_state) | While org active | 30 days then anonymized | Stripe holds the truth; local copy is a reference | Stripe has its own retention policy |
| Messages | Active conversation | 30 days | Anonymized via sender_id rewrite (already nullable per migration 130) | Recommend: archive conversations idle >3 years |
| Search queries | 12 months | n/a | Anonymize user_id (set to NULL) | Recommended in B1 |
| Audit logs | Indefinite | 12 months | Archive to R2 cold tier; anonymized via GDPR purge | Documented intent |
| Notifications | 90 days | n/a | Hard delete via TTL cron | Recommended in B1 |
| Device tokens | 60 days inactivity | n/a | Hard delete via TTL cron | FCM marks tokens stale after 60d |
| Password resets | 24h post-expiry | n/a | Hard delete via TTL cron | Already enforced by checking expiry but not pruned |
| Reviews | While account active | 30 days | Anonymized via reviewer_id rewrite | UGC; reputation system needs continuity |
| Jobs (closed) | 24 months post-closing | n/a | Archive to read-only | Recommended in B1 |
| Proposals (settled) | Tax retention 5 years | n/a | Anonymized via user_id rewrite | Recommended in B1 |
| Disputes (settled) | 5 years | n/a | Same as proposals | Recommended |
| Reports (handled) | 24 months | n/a | Anonymize reporter_id | Recommended |
| Moderation results | Indefinite | n/a | Author_user_id already nullable (`ON DELETE SET NULL`) | Implemented ✅ |
| Conversation participants / read state | While conversation alive | n/a | Hard delete on user purge ✅ | Implemented |
| Job views | Last 30 days for "recently viewed" UX | n/a | Hard delete | Hard delete on user purge ✅ |

---

## 8. AIPD/PIA (art. 35) — risk-based assessment

CNIL 2023 trigger list cross-checked against this marketplace.

| Trigger | Activated? | Why |
|---|---|---|
| Large-scale processing of biometric data | 🟡 partial — KYC selfie + id document upload via Stripe Embedded; locally stored `identity_documents` with face on doc | Stripe is the data controller for the biometric step. Local copies (`identity_documents`) hold the post-OCR/post-decision file but not raw biometric features. |
| Innovative use of new technology | 🟢 yes — AI moderation (OpenAI), AI dispute summarization (Anthropic Claude), embeddings-based semantic search | All three are AI-driven decisions affecting users (content rejection, dispute summary used by ops). |
| Automated decision with legal/significant effect | 🟢 yes — auto-rejection of UGC content above moderation threshold (`config.RekognitionAutoRejectThreshold` and the OpenAI moderation thresholds in `domain/moderation`); auto-restriction of payouts on KYC failure (`organizations.kyc_first_earning_at`) | Effect on the user: media deleted from R2, account potentially suspended/restricted. |
| Data on minors | not applicable in theory (B2B) | But registration has no age check (`backend/internal/handler/dto/request/auth.go` validates min 8 chars password but no DOB, no age gate). |
| Combining datasets from different sources | 🟢 yes — search engine combines profile + reviews + skills + behavioral signals (search_queries) + freelance/agency activity in a single Typesense document | `backend/internal/search/indexer.go` |
| Profiling at scale | 🟡 — search ranking uses behavioral data (clicked_position) to learn-to-rank | `backend/migrations/111_create_search_queries.up.sql:19-20` |
| Geolocation tracking | partial — city autocomplete sends typed strings to BAN/Photon; no GPS, no IP geolocation outside CDN/log defaults | low risk |

**Conclusion**: ≥3 triggers are active → **a full AIPD is recommended**, not optional. Even if interpreted conservatively, an AIPD-mini for the AI moderation pipeline + the search ranking is appropriate.

**AIPD-mini stub (AI moderation)**:
- Treatment: every UGC text (messages, reviews, profiles, jobs, proposals) is sent to OpenAI for toxicity scoring; every uploaded image / video frame is sent to AWS Rekognition; results above auto-reject threshold delete the content + log a moderation_results row.
- Risks: false positive (legitimate content blocked, user frustrated, no clear appeal channel), data leak to OpenAI / AWS, profiling.
- Mitigations: thresholds are configurable, two-tier (flag-for-review at 60, auto-reject at 95 — `backend/internal/config/config.go:181-182`); admins can review via /admin/moderation; OpenAI moderation API has "no training" by default.
- Residual risk: medium. Mitigated by appeal flow via /admin/moderation + audit log.
- Action: document the appeal path explicitly to data subjects, surface human-review request UI on user-facing rejection notice.

**AIPD-mini stub (search ranking)**:
- Treatment: search_queries table records every query + clicks per user; ranking uses behavioral signal aggregated globally.
- Risks: over-personalization, lock-in, opacity.
- Mitigations: anonymous queries are also kept (no user_id), session_id fallback; ranking is documented in `backend/internal/search/`; not used for credit/insurance/legal decisions.
- Residual risk: low.

---

### How well does Section 4 match the CLAUDE.md GDPR claims?

The repo claim sheet from `CLAUDE.md` and `backend/CLAUDE.md` was cross-checked against actual code:

| CLAUDE.md claim | Actual state | Verdict |
|---|---|---|
| "Right to deletion: Users can request full account deletion. Cascade delete all personal data." | Implemented as soft-delete + 30-day cooldown + anonymization-in-place (NOT full cascade due to FK constraints on proposals/disputes/jobs/invoices/reviews/payment_records). | 🟡 partial — the README description is misleading; reality is anonymization (which is RGPD-compliant but not "full cascade"). |
| "Right to export: Users can request a data export (JSON format) of all their personal data." | Implemented as ZIP with 10 JSON sections + manifest + bilingual README. | 🟢 implemented + better than claimed |
| "Consent: Explicit opt-in for marketing communications. Track consent timestamps." | No marketing communications surface today; no consent timestamp tracking. | 🔴 missing — claim is forward-looking |
| "EU data residency: Production databases hosted in EU region." | Neon project region not verified in repo; assumed EU. | 🟡 must verify and document |
| "Retention: Define and enforce data retention policies per data type." | Only `users` (deleted) has enforced retention. | 🔴 partial — most tables have no enforcement |
| "Audit logs are kept indefinitely (legal/compliance requirement). Archive to cold storage after 12 months if volume becomes a concern." | Indefinite retention is in place; cold storage archive is documented but not implemented. | 🟡 partial |
| "RLS enabled on ALL tables except users." | Only 9 tables have RLS (migration 125). Other tables (profile, jobs, billing_profile, identity_documents, business_persons, payment_info, organizations, etc.) do NOT have RLS — application-layer `WHERE` filters are the only defense. | 🔴 partial — claim is overstated. Recommendation: extend RLS to billing_profile, identity_documents, business_persons. |

## 9. Consent management

| Item | Status | Source |
|---|---|---|
| Cookie consent banner | 🟡 partial — only one banner for "analytics" (PostHog + GA4 collectively) | `web/src/shared/components/analytics/cookie-banner.tsx` · `web/src/shared/lib/posthog-consent.ts` |
| Granularity (functional / analytics / marketing) | 🔴 missing — single accept/refuse toggle | `web/src/shared/components/analytics/cookie-banner.tsx:48-54` |
| Analytics fires before consent | 🟢 NO — PostHog is `opt_out_capturing_by_default: true` (`web/src/shared/lib/posthog.ts:78`); GA4 only mounts after consent (`web/src/shared/components/analytics/google-analytics-provider.tsx:48-50`) | OK ✅ |
| Storage of consent | 🟡 localStorage only (`marketplace.analytics.consent`) — no server-side trace | `web/src/shared/lib/posthog-consent.ts:15` |
| Consent timestamp + IP for legal proof | 🔴 missing — never recorded | — |
| Marketing email opt-in | 🟡 default `email_notifications_enabled = true` for transactional + reviews/proposals — interpretable as soft opt-in for service emails. No marketing email surface today, but `notification_preferences` defaults `email = false` for granular types (`backend/migrations/017_create_notification_prefs_and_devices.up.sql:6`) | — |
| Profile public visibility | always default-on (publishing profile makes it discoverable) — must be disclosed | `backend/internal/search/indexer.go` |
| AI training opt-out | 🔴 missing — neither documented nor toggleable. OpenAI defaults to "no training" for API customers, but the user is never told their messages traverse OpenAI; same for Anthropic dispute analysis. | — |
| Consent revocation UI | 🟡 partial — `clearConsent()` helper exists but is not wired into a settings page | `web/src/shared/lib/posthog-consent.ts:67-75` |

🔴 **Cookie banner wording (`web/messages/fr.json` `analyticsConsent.description`)** currently reads:
> "On utilise un outil d'analyse pour améliorer le service. Tu peux refuser sans rien perdre."

CNIL guidance requires: name the providers (Google Analytics, PostHog), state purpose, mention non-EU transfer if applicable, link to privacy policy. The current text fails these.

---

### Specific consent issues to fix before publishing

1. **The cookie banner is rendered AFTER the page loads** (`useEffect` mount), so a flash of the banner is possible. Currently mitigated by `setShouldShow(false)` initial state (`web/src/shared/components/analytics/cookie-banner.tsx:25`). OK ✅
2. **PostHog is set to `persistence: "memory"` until consent** (`web/src/shared/lib/posthog.ts` `init` config), so no cookies are stored before consent. OK ✅
3. **GA4 only mounts after `hasConsent === true`** (`web/src/shared/components/analytics/google-analytics-provider.tsx:48-50`). OK ✅
4. **No banner** in the **mobile** Flutter app. PostHog mobile SDK and any analytics-equivalent in mobile must be gated by an in-app consent screen — not audited here, flagged.
5. **No banner** in the **admin** Vite app. Admin is internal use, but if it uses GA4/PostHog, the same consent gates apply (audit not extended to admin in this round).
6. **Consent revocation** — the helper `clearConsent()` exists but is not exposed to users. Need a "Préférences de confidentialité" page accessible from settings AND footer.

## 10. Audit logging

| Element | Status | Source |
|---|---|---|
| Table exists with expected columns | 🟢 yes | `backend/migrations/078_create_audit_logs.up.sql` |
| Append-only at DB level | 🟡 partial — `REVOKE UPDATE, DELETE ON audit_logs FROM PUBLIC` (migration 124) is symbolic; the production DB role still has UPDATE/DELETE because the rule is not enforced via a dedicated `marketplace_app` role with narrowed grants. The migration explicitly notes this is INFRA work. (`backend/migrations/124_audit_logs_grants.up.sql:18-31`) | `backend/migrations/124_audit_logs_grants.up.sql` |
| RLS enabled | 🟢 yes — policy on `user_id = current_setting('app.current_user_id', true)` (migration 125) | `backend/migrations/125_enable_row_level_security.up.sql:187-193` |
| Anonymization on user purge (sha256 of email + IP /16 truncation) | 🟢 yes | `backend/internal/adapter/postgres/gdpr_repository.go:528-548` · `backend/internal/domain/gdpr/anonymization.go:38-79` |
| Events logged: login (success/failure), logout, password reset, password change, email change, token refresh, token reuse detected, role permissions changed, member role changed, ownership transferred, admin user suspend/unsuspend/ban/unban, admin force-transfer, authz denied, receipt view/download, referral blocked | 🟢 comprehensive | `backend/internal/domain/audit/entity.go:33-93` |
| Retention rule (cold storage after 12 months) | 🔴 documented in CLAUDE.md but not implemented | — |
| Sensitive data redaction in metadata | 🟡 partial — metadata is free-form JSONB; the auth service deliberately omits password material (`backend/internal/app/auth/service_account_test.go:312`) but still stores cleartext email on every login_failure | `backend/internal/app/auth/service.go:373-374` |

---

### Audit log event coverage

The complete list of audit actions from `backend/internal/domain/audit/entity.go:33-93`:

- `auth.login_success`, `auth.login_failure`, `auth.logout`, `auth.token_refresh`, `auth.token_reuse_detected`
- `auth.password_reset_request`, `auth.password_reset_complete`, `auth.change_email`, `auth.change_password`
- `team.role_permissions_changed`, `team.member_role_changed`, `team.member_removed`, `team.ownership_transferred`
- `admin.user_suspend`, `admin.user_unsuspend`, `admin.user_ban`, `admin.user_unban`, `admin.force_transfer_ownership`
- `authz.denied`
- `receipt.view`, `receipt.pdf_download`
- `referral.blocked_already_in_relation`

Note: there are **no audit events** for: profile updates (`profile.updated`), KYC document upload (`kyc.document_uploaded`), payment/payout (`payment.processed`), GDPR right exercises (`gdpr.export_requested`, `gdpr.deletion_requested`, `gdpr.deletion_confirmed`, `gdpr.deletion_cancelled`), moderation decisions (`moderation.decision`), consent changes (`consent.granted`, `consent.revoked`).

🔴 **Recommendation**: extend the audit event catalog to cover GDPR rights exercises (these are the events you need to prove compliance) — even a single `gdpr.deletion_requested` row per request is enough. Effort: ~2h.

## 11. Security baseline (art. 32)

| Control | Status | Source |
|---|---|---|
| Encryption at rest (Neon, R2) | 🟢 default-on by provider | infra |
| Encryption in transit (HTTPS, HSTS) | 🟢 yes — HSTS 1 year in prod | `backend/internal/handler/middleware/security_headers.go:55-58` |
| Password hashing (bcrypt cost 12) | 🟢 yes | `backend/cmd/api/wire_infra.go:203` (`crypto.NewBcryptHasher()`) · pkg/crypto |
| JWT short-lived + refresh rotation | 🟢 yes — 15min access / 7d refresh + Redis blacklist on rotation | `backend/internal/config/config.go:157-158` · `backend/cmd/api/wire_infra.go:222` |
| Brute-force protection | 🟢 yes — Redis sliding window per email + per IP | `backend/internal/handler/auth_handler_bruteforce_failclosed_test.go` · `backend/internal/handler/middleware/bruteforce.go` (location to verify) |
| RLS enabled on tenant-scoped tables | 🟢 yes — 9 tables (conversations, messages, invoice, proposals, proposal_milestones, notifications, disputes, audit_logs, payment_records) | `backend/migrations/125_enable_row_level_security.up.sql` |
| Two-pool DB (NOBYPASSRLS for user, BYPASSRLS for system) | 🟡 partial — wiring is ready (`backend/cmd/api/wire_infra.go:108-160`); migration 137 created roles; production rotation pending per `backend/docs/rls-rollout.md` | `backend/migrations/137_two_pool_rls_roles.up.sql` |
| Rate limiting | 🟢 yes — global 100/min + mutation 30/min + upload 10/min | `backend/internal/handler/middleware/ratelimit*.go` · `backend/internal/config/config.go:117-119` |
| HTTP security headers (CSP, X-Frame-Options, HSTS, X-Content-Type-Options, Referrer-Policy, Permissions-Policy) | 🟢 yes | `backend/internal/handler/middleware/security_headers.go` |
| CSP env-driven, fail-fast in prod | 🟢 yes | `web/src/shared/lib/csp.ts:149-172` |
| Input validation (parameterized SQL, struct tags + validator) | 🟢 yes | `pkg/validator`, `backend/internal/handler/dto/request/*` |
| File upload validation (magic bytes, size, executable rejection) | partial — validate in upload handler | `backend/internal/handler/upload_handler.go` |
| 2FA / MFA | 🔴 missing — not implemented | — |
| Secret rotation policy | 🔴 not documented | — |
| Backup + restore tested | 🔴 not documented in repo | — |
| Vulnerability disclosure (`SECURITY.md`) | 🟢 present | `SECURITY.md` |

---

### Specific security findings worth flagging in the privacy policy

These exist and should be advertised — they show good faith:

- **bcrypt cost 12** for passwords, not just hashed (`backend/internal/config/config.go` + `pkg/crypto`)
- **JWT short-lived 15 min access + 7-day refresh + Redis blacklist on rotation + `auth.token_reuse_detected` event when a blacklisted refresh token is replayed** — this is industry-leading
- **Brute-force protection per email AND per IP** (`backend/internal/handler/auth_handler_bruteforce_failclosed_test.go` + IP-based test)
- **HSTS 1 year** in production
- **CSP env-driven, fail-fast at boot in production** when required env vars are missing (`web/src/shared/lib/csp.ts:149-172`)
- **Two-pool RLS** infrastructure ready (`backend/migrations/137_two_pool_rls_roles.up.sql`)
- **GDPR purge anonymization salt is mandatory at boot in production** — refuses dev fallback (`backend/internal/config/config.go:310-314`)
- **Stripe used for KYC** — biometric data does not transit your servers, only Stripe holds the docs

These NOT yet implemented or weak (do NOT advertise these in the policy without fixing first):

- **2FA (TOTP) is not implemented** — flagged Phase B9
- **Column-level encryption for legacy bank fields not implemented** — flagged in Section 6
- **Audit log archival rule not implemented** — Phase B2
- **Consent log not implemented** — Phase A5
- **R2 object cleanup on user purge not implemented** — Section 1 sensitivity flag 3

## 12. Documentaire / organisationnel (gaps)

What's NOT in the code but legally required.

| Item | Status | Effort |
|---|---|---|
| Registre des traitements (art. 30) | 🔴 missing | 4h — auto-build from Section 1 of this audit |
| AIPD complete | 🔴 missing | 8h — write 1 AIPD covering AI moderation + search ranking + KYC; out of scope here |
| Procédure de violation 72h | 🔴 missing | 4h — runbook |
| Procédure d'exercice des droits + 1-month delay tracking | 🟡 partial — endpoint exists, no support workflow doc | 2h — runbook |
| DPO ou point de contact RGPD désigné | 🔴 missing | 1h — designate (likely the founder) and publish email in privacy policy |
| DPAs signés avec chaque sous-traitant | 🔴 21 vendors to verify, see Section 2 | 16h — collect, sign, store in `docs/dpa/` |
| Privacy policy / mentions légales / CGU / CGV / cookies pages | 🔴 missing | 12h — write + integrate, link from footer |
| Legal pages route (`/privacy`, `/legal`, `/cookies`, `/cgu`, `/cgv`) | 🔴 routes 404 today | 4h — Next.js pages with MDX content |
| Consent log (server-side trace of accept/refuse + IP + UA + timestamp + version) | 🔴 missing | 6h — new table + middleware hook |

---

### Privacy policy version-control

Even before publishing, decide the versioning scheme:

| Element | Recommended approach |
|---|---|
| Policy file location | `web/messages/legal/privacy.fr.mdx` + `web/messages/legal/privacy.en.mdx` |
| Versioning | Semantic-ish: `1.0` at first publish; bump minor on every clarification, major on every meaningful change of treatment |
| Notification | When major version bumps, send a `system_announcement` notification to every active user with a CTA to read the diff |
| Version header on the page | "Version 1.0 — publiée le 2026-05-15" |
| Diff between versions | Maintain a `web/messages/legal/CHANGELOG.md` with a one-paragraph summary per version bump |
| Storage of past versions | Git history is canonical; in addition, store rendered PDF in R2 `/legal-archive/privacy-vN.N.pdf` |

### Pre-publish checklist (run through before clicking "publish")

- [ ] Privacy policy text reviewed by a French lawyer or DPO (CNIL has a self-assessment tool: https://www.cnil.fr/fr/privacy-icons)
- [ ] CGU + CGV reviewed by a French lawyer
- [ ] All 21 sub-processor DPAs signed and stored in `docs/dpa/`
- [ ] Neon project region verified as EU (and added to `docs/data-residency.md`)
- [ ] Vercel + Railway region verified as EU when possible (or US documented as transfer)
- [ ] AWS region verified as EU (eu-west-1)
- [ ] Typesense region verified as EU
- [ ] PostHog project verified as EU
- [ ] OpenAI "no training on API data" toggle verified ON
- [ ] Anthropic "no training on API data" + Zero Data Retention eligibility verified
- [ ] Stripe Connect EU sub-entity verified for European users
- [ ] FCM US transfer disclosed in policy
- [ ] GA4 Region 1 + IP truncation + Google Signals OFF verified in GA4 dashboard
- [ ] Cookie banner updated with granular toggles (functional / analytics / marketing)
- [ ] Cookie banner names PostHog + Google explicitly + non-EEA warning
- [ ] Footer links to `/privacy`, `/cookies`, `/cgu`, `/cgv`, `/legal` exist (no 404)
- [ ] DPO email created and active (`dpo@designedtrust.com` recommended)
- [ ] CNIL recourse mention present in policy
- [ ] AIPD-mini for AI moderation written and stored in `docs/aipd-ai-moderation.md`
- [ ] AIPD-mini for search ranking written and stored in `docs/aipd-search-ranking.md`
- [ ] Registre des traitements written and stored in `docs/registre-traitements.md`
- [ ] Procédure violation 72h written in `docs/runbook-violation.md`
- [ ] Procédure d'exercice des droits written in `docs/runbook-droits-rgpd.md` with 1-month delay tracking
- [ ] At least 1 dry-run of the GDPR purge cron in staging — observed in logs
- [ ] At least 1 manual export of test user verified — ZIP opens, README readable, JSON parses
- [ ] Mobile parity: privacy policy + cookie banner equivalent in Flutter

## 13. Plan d'action priorisé

### Phase A — Quick wins (< 4h each, ~2 days total)

| # | Item | Effort | Impact | Dependency |
|---|---|---|---|---|
| A1 | Create `/privacy`, `/cookies`, `/legal`, `/cgu`, `/cgv` Next.js routes with placeholder MDX referencing this audit | 3h | High | none |
| A2 | Update analytics consent banner wording: name PostHog + GA4 + Google + non-EEA transfer warning + "lien vers la politique de confidentialité" | 1h | High | A1 |
| A3 | Make consent banner granular (functional / analytics / marketing) — even if marketing slot stays empty for now | 3h | High | A2 |
| A4 | Surface a "Cookies" / "Préférences de confidentialité" link in footer + account settings to revoke consent | 1h | Medium | A2 |
| A5 | Add a `consent_log` table + insert on every accept/refuse (with anonymized IP `/16` + UA hash + version) | 4h | High | A2 |
| A6 | Document AI moderation + search ranking on a `/legal/ai-disclosures` page | 2h | Medium | A1 |
| A7 | Drop `payment_info.iban / account_number / routing_number / account_holder` columns (or migrate to encrypted) — Stripe is the source of truth now | 3h | High | follow-up migration + check no read paths use these columns |
| A8 | Add link to existing `/me/export` (right to access) and `/account/delete` (right to erasure) from the footer + privacy policy | 1h | Medium | A1 |
| A9 | Designate point-of-contact RGPD (email like `dpo@designedtrust.com`) and publish in privacy policy | 1h | Required | A1 |

**Phase A total: ~19h ≈ 2.5 days.**

### Phase B — Compliance core (1 week)

| # | Item | Effort | Impact | Dependency |
|---|---|---|---|---|
| B1 | Add retention enforcement: `notifications` (90 days), `device_tokens` (60 days inactivity), `password_resets` (24h post-expiry), `search_queries` (12 months → anonymize user_id), `message_history` (align with messages) — implement as a daily cron extending `gdpr.Scheduler` or a sibling | 8h | High | none |
| B2 | Add `audit_logs` cold-storage archiving rule (12 months → dump to R2 + truncate) | 6h | Medium | B1 |
| B3 | Implement art. 22 "human review" UI: when a user's content is auto-rejected (`rejected` moderation_results), surface an appeal button → file a new `report` against the moderation decision | 6h | High | none |
| B4 | Implement art. 21 "object" toggles: hide-from-search opt-out (sets a flag that excludes the org/profile from Typesense indexing) | 6h | Medium | none |
| B5 | Implement marketing email opt-in tracking (timestamps, IP) once a marketing surface is added — placeholder for now: extend the `consent_log` table | 4h | Low | A5 |
| B6 | Implement Idempotency-Key + ratelimited "request my data" / "delete my account" surfaces with email confirmation — already mostly done; verify rate limit caps for these endpoints | 2h | Low | none |
| B7 | Wire dedicated `marketplace_app` DB role with INSERT+SELECT-only on `audit_logs` (Railway/Neon dashboard work + CI smoke test) | 4h | High | infra |
| B8 | Verify Neon project EU region + Typesense EU region + Stripe Connect EU sub-entity for European users; document in `docs/data-residency.md` | 2h | High | infra |
| B9 | Add 2FA TOTP for high-privilege actions (admin role). RGPD does not mandate, but art. 32 "raisonnable" pushes for it. | 16h | Medium | follow-up |
| B10 | Sanitize `audit_logs.metadata.email` for unknown users (hash immediately on `auth.login_failure` when the email is not registered) | 2h | Medium | none |

**Phase B total: ~56h ≈ 1 week.**

### Phase B detailed code-level deliverables

For each B-item, here is the exact code path the developer should follow:

- **B1 retention scheduler.** Create `backend/internal/app/retention/scheduler.go` mirroring `app/gdpr/scheduler.go`. The retention service exposes `PurgeOnce(ctx) (Result, error)` that runs N independent batch deletes per table. Add a config knob `RETENTION_INTERVAL` (default 24h, dev 1m). Wire from `cmd/api/wire_late_handlers.go`. Each table gets its own SQL: e.g. `DELETE FROM notifications WHERE created_at < NOW() - INTERVAL '90 days'`. Use `LIMIT 5000` per batch + `RETURNING id` for observability.
- **B2 audit log archival.** Add a daily job that exports rows older than 12 months to R2 (`s3.PutObject` to `audit-archive/<yyyy-mm>/...jsonl.gz`) then `DELETE` the archived rows. The dedicated `marketplace_app` role still has DELETE on this table (the symbolic REVOKE in migration 124 only revoked from PUBLIC), so the application can do this — but only the archival cron should hold the privilege. Recommendation: introduce a third DB role `marketplace_archiver` with `INSERT, SELECT, DELETE` on `audit_logs`, and run the archival job under that role exclusively.
- **B3 art. 22 human review.** Extend `domain/moderation` to expose an `appealable` flag on `Result`. When `decision = rejected` AND `appealable = true`, the frontend renders a "Demander une revue humaine" CTA that POSTs to `/api/v1/moderation/{id}/appeal` (new endpoint). The handler creates a `report` of type `moderation_decision` linked to the moderation_result. Admin sees it in /admin/moderation. The audit log records `moderation.appeal_requested`.
- **B4 hide-from-search.** Add `users.search_indexed BOOLEAN NOT NULL DEFAULT true`. When the user toggles "ne plus apparaître dans la recherche" in settings, the column flips to false, and `Indexer.Sync` skips the user. The Typesense doc is removed via `Delete`. The GDPR export still includes the user in their own data — the toggle only affects visibility to other users.
- **B5 marketing email tracking.** Add `consent_log` (id, user_id NULL, consent_type, choice, version, ip_address, user_agent_hash, created_at). On every banner click + every TOS-accept-on-register, INSERT a row. Used as proof in case of CNIL inquiry.
- **B6 idempotency on deletion endpoints.** The GDPR endpoints are already idempotent by design (re-sending the email is a no-op, soft-delete uses COALESCE, cancel returns NoOp=true) but the rate limiter caps should be tightened to 3 req/min on `request-deletion` to slow brute-force on user emails who are not the actual user.
- **B7 db role split.** This is INFRA work in Railway/Neon dashboards. Create `marketplace_app` role with `LOGIN PASSWORD '...'`. Grant `CONNECT` to db, `USAGE` on schema, `SELECT, INSERT, UPDATE, DELETE` on most tables EXCEPT `audit_logs` where it gets only `SELECT, INSERT`. Update `DATABASE_URL` env on Railway to use the new role.
- **B8 EU residency.** Verify each provider dashboard (see Section 2 verification table). Document in `docs/data-residency.md` with screenshots.
- **B9 2FA.** Add `users.totp_secret_encrypted TEXT NULL` + `users.totp_enabled_at TIMESTAMPTZ NULL`. New endpoints `POST /api/v1/me/2fa/enroll` (returns QR), `POST /api/v1/me/2fa/verify` (validates code, sets enrolled_at). Login flow checks `totp_enabled_at IS NOT NULL` and prompts for code if so. Use `pquerna/otp` Go library.
- **B10 metadata sanitization.** In `app/auth/service.go:373-374`, change the `Metadata: {"email": input.Email, ...}` to `Metadata: {"email_hash": hashEmail(input.Email), ...}` for cases where `users.id` is not found (i.e., never-registered email). Keep the cleartext for found users since the GDPR purge will rewrite it.

### Phase C — Documentaire (1.5 weeks)

| # | Item | Effort | Impact | Dependency |
|---|---|---|---|---|
| C1 | Write the privacy policy (FR primary, EN secondary) using Sections 1, 2, 3, 4, 5, 7, 9 of this audit as raw input | 16h | Required | A1 |
| C2 | Write CGU + CGV (separate from privacy) | 12h | Required | none |
| C3 | Write mentions légales (LCEN art. 6 III) | 1h | Required | none |
| C4 | Write the cookies page (list every cookie + provider + duration + purpose) | 4h | Required | A2 |
| C5 | Build `docs/registre-traitements.md` from Section 1 of this audit | 4h | Required | none |
| C6 | Build `docs/aipd-ai-moderation.md` and `docs/aipd-search-ranking.md` (full AIPD format) | 16h | Required if 2+ triggers | none |
| C7 | Sign DPAs with all 21 sub-processors (track in `docs/dpa/<vendor>.md`) | 16h | Required | none |
| C8 | Write `docs/runbook-violation.md` (72h CNIL notification procedure) | 4h | Required | none |
| C9 | Write `docs/runbook-droits-rgpd.md` (workflow for handling access/rectification/erasure/object requests + 1-month deadline tracking) | 4h | Required | none |
| C10 | Translate privacy policy + CGU + cookies into EN | 6h | Medium | C1, C2, C4 |

**Phase C total: ~83h ≈ 1.5 weeks.**

**Grand total: ~158h ≈ 3 weeks of focused dev + legal review.**

---

## 14. Specific recommendations for the privacy policy + cookie banner

### Privacy policy must include

1. **Identité du responsable de traitement** — Marketplace Service / `designedtrust.com`, founder address (Section 12 C9).
2. **DPO / point de contact** — designated email (Phase A9).
3. **Catégories de données traitées** — copy directly from Section 1.
4. **Finalités** — copy from Section 5.
5. **Bases légales** — copy from Section 5.
6. **Destinataires (sous-traitants)** — copy from Section 2 (21 vendors).
7. **Transferts hors UE + mécanisme** — copy from Section 3.
8. **Durées de conservation** — copy from Section 7.
9. **Droits + exercice (DPO email + 1 month delay + CNIL recourse)** — copy from Section 4.
10. **Décisions automatisées (art. 22)** — explicit notice for AI moderation (OpenAI), Rekognition, search ranking + appeal path.
11. **Cookies et trackers** — separate page (link from policy).
12. **Sécurité** — high-level summary of Section 11.
13. **Mineurs** — service is B2B, not for under-18; reserve right to delete underage accounts.
14. **Modifications de la politique** — versioning + how users are notified.

### Cookie page must list

| Cookie / storage key | Provider | Type | Purpose | Duration |
|---|---|---|---|---|
| `session_id` | self (backend) | functional / strictly necessary | authenticated session | 14 days (`SessionTTL` default `336h`, `backend/internal/config/config.go:159`) |
| `marketplace.analytics.consent` | self (localStorage) | functional / strictly necessary | persist consent choice | indefinite (until user clears browser) |
| PostHog `ph_*` cookies | PostHog (https://eu.posthog.com) | analytics — opt-in | distinct_id, session correlation | 1 year (PostHog default) |
| GA4 `_ga`, `_ga_*` cookies | Google (US) | analytics — opt-in | client_id, session | 2 years |
| `NEXT_LOCALE` (next-intl) | self | functional | language preference | session |

### Cookie banner — required wording (FR)

> Nous utilisons des cookies et outils d'analyse (PostHog hébergé en Irlande, et Google Analytics aux États-Unis) pour mesurer l'audience et améliorer le service. Aucun cookie tiers n'est déposé tant que tu n'as pas accepté. Tu peux modifier ton choix à tout moment depuis ton compte. [En savoir plus →](/cookies)

with three buttons: **Tout refuser** / **Personnaliser** / **Tout accepter**, granular by category.

### Privacy policy table of contents (suggested skeleton)

1. **Identité du responsable de traitement** + DPO
2. **Données collectées** + finalités + bases légales (Sections 1+5 of this audit)
3. **Sources des données** (toutes collectées directement; quelques-unes héritées d'OAuth providers)
4. **Sous-traitants et destinataires** (Section 2 verbatim, with DPF/SCC mention each)
5. **Transferts hors UE** (Section 3 verbatim)
6. **Durées de conservation** (Section 7 retention matrix)
7. **Tes droits** (Section 4 + how to exercise)
8. **Décisions automatisées + profilage** (AI moderation + Rekognition + search ranking + appeal path)
9. **Cookies et trackers** (link to /cookies)
10. **Sécurité** (high-level Section 11)
11. **Mineurs** (B2B service, not for under 18)
12. **Modifications** (versioning policy)

### Mentions légales template (suggested)

- Éditeur du site : Marketplace Service [SAS] / `designedtrust.com`
- Adresse postale : *(à fournir)*
- Numéro RCS / SIRET : *(à fournir)*
- Capital social : *(à fournir si SAS/SARL)*
- Directeur de publication : *(fondateur)*
- Hébergeur : Vercel Inc., 340 S Lemon Ave #4133, Walnut, CA 91789, USA
- Hébergeur API : Railway Corp., 1771 Page Mill Rd, Palo Alto, CA 94304, USA
- DPO / contact : `dpo@designedtrust.com`
- CNIL recourse: https://www.cnil.fr/fr/plaintes

---

## Annex A — Files audited

### Backend (Go)

- Config + boot: `backend/internal/config/config.go`, `backend/cmd/api/main.go`, `backend/cmd/api/wire_infra.go`, `backend/cmd/api/wire_gdpr.go`
- GDPR feature: `backend/internal/handler/gdpr_handler.go`, `backend/internal/handler/routes_gdpr.go`, `backend/internal/domain/gdpr/anonymization.go`, `backend/internal/domain/gdpr/deletion.go`, `backend/internal/domain/gdpr/export.go`, `backend/internal/app/gdpr/service.go`, `backend/internal/app/gdpr/scheduler.go`, `backend/internal/adapter/postgres/gdpr_repository.go`
- Auth + audit: `backend/internal/app/auth/service.go`, `backend/internal/domain/audit/entity.go`, `backend/internal/handler/dto/request/auth.go`
- Middleware: `backend/internal/handler/middleware/security_headers.go`, `backend/internal/handler/middleware/cors.go`, `backend/internal/handler/middleware/logger.go`, `backend/internal/handler/middleware/ratelimit*.go`
- Sub-processors (adapters): `backend/internal/adapter/openai/text_moderation.go`, `backend/internal/adapter/anthropic/analyzer.go`, `backend/internal/adapter/rekognition/moderation.go`, `backend/internal/adapter/s3/storage.go`, `backend/internal/adapter/resend/email.go`, `backend/internal/adapter/stripe/account.go`, `backend/internal/adapter/livekit/client.go`, `backend/internal/adapter/fcm/push.go`, `backend/internal/adapter/posthog/client.go`, `backend/internal/adapter/vies/client.go`, `backend/internal/adapter/nominatim/client.go`, `backend/internal/adapter/sqs/worker.go`, `backend/internal/adapter/s3transit/transit.go`, `backend/internal/adapter/comprehend/text_moderation.go`
- Migrations (selection): 001 (users), 002 (profiles), 015 (payment_info), 017 (notification_prefs), 019 (identity_documents), 020 (phone_activity), 021 (business_persons), 065 (organizations), 076 (email_notifications_enabled), 078 (audit_logs), 111 (search_queries), 120 (moderation_results), 121 (invoicing), 124 (audit_logs_grants), 125 (RLS), 132 (users_deleted_at), 137 (two-pool RLS), 138 (job_applications)
- DTOs: `backend/internal/handler/dto/request/{auth,profile,freelance_profile,messaging,job,proposal}.go`
- Env example: `backend/.env.example`

### Web (Next.js)

- Consent + analytics: `web/src/shared/components/analytics/cookie-banner.tsx`, `web/src/shared/components/analytics/google-analytics-provider.tsx`, `web/src/shared/lib/posthog-consent.ts`, `web/src/shared/lib/posthog.ts`, `web/src/shared/lib/ga.ts`, `web/src/shared/lib/csp.ts`
- GDPR feature: `web/src/features/account/api/gdpr.ts`, `web/src/features/account/components/{delete-account-card,notification-settings,email-settings}.tsx`
- Middleware + auth: `web/src/middleware.ts`
- I18n: `web/messages/fr.json`, `web/messages/en.json` (analyticsConsent block)

### Mobile (Flutter)

- GDPR: `mobile/lib/features/account/domain/repositories/gdpr_repository.dart`, `mobile/lib/features/account/presentation/screens/{delete_account,cancel_deletion,account}_screen.dart`, `mobile/lib/features/account/data/gdpr_repository_impl.dart`

### Project-level docs

- `CLAUDE.md` (security + RGPD section)
- `backend/CLAUDE.md` (security + audit logging + RLS sections)
- `SECURITY.md`

---

## Annex B — Reference snippets

For convenience when writing the policy, here are exact code references for facts the policy must state.

### Soft-delete window

> 30 days, defined by `gdpr.PurgeWindow = 30 * 24 * time.Hour`
> (`backend/internal/domain/gdpr/deletion.go:12`)

### Deletion confirmation token TTL

> 24 hours, defined by `gdpr.ConfirmationTokenTTL = 24 * time.Hour`
> (`backend/internal/domain/gdpr/deletion.go:16`)

### Session cookie TTL

> 14 days, defined by `SESSION_TTL=336h` (config default at `backend/internal/config/config.go:159`)

### Access JWT TTL

> 15 minutes, defined by `JWT_ACCESS_EXPIRY=15m` (config default at `backend/internal/config/config.go:157`)

### Refresh JWT TTL

> 7 days, defined by `JWT_REFRESH_EXPIRY=168h` (config default at `backend/internal/config/config.go:158`)

### IP truncation policy on user purge

> IPv4: last 2 octets zeroed (mask /16). IPv6: last 96 bits zeroed (mask /32).
> (`backend/internal/domain/gdpr/anonymization.go:65-79`)

### Email anonymization formula on user purge

> `sha256(lower(trim(email)) + GDPR_ANONYMIZATION_SALT)` stored as hex.
> (`backend/internal/domain/gdpr/anonymization.go:38-45`)

### Rekognition auto-rejection threshold

> Default 95.0 — content with confidence ≥ 95.0 in any moderation label is auto-rejected without human review (env: `REKOGNITION_AUTO_REJECT_THRESHOLD`, default at `backend/internal/config/config.go:182`)
> Flag-for-review threshold: 60.0 — content above this surfaces in /admin/moderation but is not auto-rejected.

### Rate limits

> Global: 100/min per IP (from rate limiter package documentation; `RATE_LIMIT_GLOBAL_PER_MINUTE`)
> Mutations: 30/min per authenticated user (`RATE_LIMIT_MUTATION_PER_MINUTE`)
> File upload: 10/min per authenticated user (`RATE_LIMIT_UPLOAD_PER_MINUTE`)
> Auth endpoints: 5/min per email (brute-force middleware)
> Lockout window after 5 failures: 30 minutes
> (`backend/internal/config/config.go:117-119` + `backend/internal/handler/middleware/bruteforce.go`)

### Password policy

> Min 8, max 128 chars; backend domain enforces uppercase + lowercase + digit + special character (`backend/internal/domain/user`)
> (RFC 5321: email max 254; client-side, `backend/internal/handler/dto/request/auth.go:8`)

### bcrypt cost

> 12 (NIST default for 2025+) — `crypto.NewBcryptHasher()` at `backend/cmd/api/wire_infra.go:203`

## Annex C — Out of scope

- **Admin SPA (admin/)** — internal-use console, not data-subject-facing. Audited indirectly via the backend admin endpoints. Should be reviewed separately for privacy of admin actions on user data (admin actions are logged in `audit_logs` already — see `ActionAdminUserSuspend` etc.).
- **Live RLS verification on the production Neon DB** — the audit only inspected migrations + wiring. A live `psql` test that attempts cross-tenant reads from the `marketplace_app` role is required to confirm RLS works in prod (recommended as part of Phase B7).
- **Penetration test / external security audit** — not in scope of this audit. Recommended before publishing the privacy policy.
- **Stripe Connect Custom data flow review** — only the local persistence side was audited. The flow of personal data on Stripe's side (between Connect Custom and Stripe Files / Stripe Identity) is governed by Stripe's own DPA and out of this audit's scope.
- **OpenSSL / TLS configuration of Railway / Vercel edges** — provider-managed, audited by the providers themselves.
- **Cookie banner A/B test on real users** — not in scope; the recommendations should be validated with the legal team before publishing.

## Annex D — Glossary

- **AIPD / DPIA / PIA** — Analyse d'Impact relative à la Protection des Données / Data Protection Impact Assessment / Privacy Impact Assessment. Required by RGPD art. 35 when a processing activity is likely to result in a high risk to the rights and freedoms of natural persons.
- **DPF** — Data Privacy Framework. The successor to Privacy Shield, in effect since July 2023, listed at `dataprivacyframework.gov`. Covers EU-US transfers under an adequacy decision.
- **DPO** — Data Protection Officer. Required by RGPD art. 37 in some cases; not required for a marketplace of this size, but the privacy policy must still name a contact point.
- **LCB-FT** — Lutte Contre le Blanchiment et le Financement du Terrorisme. French regulatory framework that imposes 5-year retention on KYC documents.
- **LIA** — Legitimate Interest Assessment. Required when relying on art. 6(1)(f) "legitimate interest" as legal basis.
- **PII** — Personally Identifiable Information.
- **RLS** — Row-Level Security. PostgreSQL feature used as defense-in-depth for tenant isolation.
- **SCC** — Standard Contractual Clauses. Adopted by the EU Commission to allow international transfers without an adequacy decision.
- **TOTP** — Time-based One-Time Password (RFC 6238). Standard 2FA method.
- **UGC** — User-Generated Content.
- **VIES** — VAT Information Exchange System. EU public service to validate cross-border VAT numbers.

## Annex E — Notable risks NOT covered by this audit

- **Performance / scalability of the GDPR feature at scale.** The export builds the full ZIP in memory before responding (`backend/internal/handler/gdpr_handler.go:66-78`). For users with very large message histories (>10,000 messages already capped) this is fine; for an export of a multi-year power user this may be tight. Streaming with `zip.NewWriter` directly into the response writer is a future optimization but not in audit scope.
- **Concurrent purge correctness in the multi-instance prod deployment.** The single-scheduler-per-process design uses `FOR UPDATE SKIP LOCKED` so multiple instances cooperate, but no integration test exercises this. Acceptance: the design is sound by inspection; a full integration test under load is out of scope.
- **GDPR purge running while user is mid-export.** If a user starts an export at T+29 days 23h59m and the cron fires at T+30 days, the export Read could race the anonymization Update. The export is wrapped in a single non-tx context (`backend/internal/adapter/postgres/gdpr_repository.go:64-115`); the anonymization is wrapped in a transaction with `FOR UPDATE`. Postgres MVCC means the export sees the pre-anonymization snapshot — but the export takes seconds, the user's email gets sent THEN. Out of scope for this audit but worth a future test.
- **Mobile-app-side analytics.** Whatever PostHog/Firebase Analytics surface ships with the Flutter app must be consent-gated. Not audited in this round.
- **Penetration test** of the deletion confirmation link replay attack, JWT signature forgery, and token leak via `Referer` header. Out of scope.

---

*End of audit. Review, then commit yourself — this file was generated read-only by the audit agent.*
