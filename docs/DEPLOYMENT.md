# Deployment

This is the production deployment runbook for the marketplace. It
covers every external service the platform depends on, the env vars
each one needs, and a first-deploy checklist you can run top-to-bottom.

The day-to-day operational guide (deploy order, reindexing, key
rotation, incident response) lives in [`docs/ops.md`](ops.md). Read
both before going live.

> **Placeholders only.** Every secret in this document is a placeholder
> string. Never commit real values; load them from your platform's
> secret manager (Railway env vars, Vercel project settings, GitHub
> Actions secrets, etc.).

---

## 1. Service map

The marketplace runs on managed services that are easy to swap because
the backend uses the hexagonal pattern (one adapter per provider).
The recommended production stack:

| Concern              | Provider          | Notes |
|----------------------|-------------------|-------|
| Backend host         | Railway           | Container deploy from the repo's `backend/Dockerfile`. |
| Web host             | Vercel            | Native Next.js 16 deploy from `web/`. |
| Admin host           | Vercel or Railway | Vite static SPA — any static host works. |
| Mobile distribution  | TestFlight + Play | Flutter build artefacts, signed locally. |
| PostgreSQL           | Neon              | EU region for GDPR; pooled connections. |
| Redis                | Upstash           | EU region; serverless plan suffices. |
| Object storage       | Cloudflare R2     | S3-compatible, zero egress fees. |
| Search engine        | Self-hosted Typesense on Railway | Single-node 28.x, 1GB volume to start. |
| Embeddings           | OpenAI            | `text-embedding-3-small`. |
| Transactional email  | Resend            | Domain auth required. |
| Push notifications   | Firebase Cloud Messaging | Mobile only. |
| Video calls          | LiveKit Cloud     | Token-based auth. |
| Payments + Connect   | Stripe            | Connect Custom + Embedded Components. |
| Web analytics (opt)  | (your choice)     | Plausible / PostHog / Vercel Analytics. |

You can run the entire stack on a different cloud — every adapter
satisfies a port interface. The instructions below describe the
recommended setup.

---

## 2. Backend on Railway

### Build configuration

The repo ships `backend/Dockerfile`. Railway's "Deploy from Dockerfile"
mode picks it up automatically. The image is `golang:1.25-alpine`
with a multi-stage build; the final image is roughly 30MB.

### Required environment variables

```bash
# Server
PORT=8080
ENV=production

# Database (Neon — pooled connection)
DATABASE_URL=postgres://USER:PASSWORD@HOST/DBNAME?sslmode=require

# Redis (Upstash)
REDIS_URL=redis://default:PASSWORD@HOST:6379

# JWT — MUST be at least 32 characters, fail-fast on shorter values.
JWT_SECRET=GENERATE_WITH_OPENSSL_RAND_HEX_32
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# CORS — explicit allow-list of frontends, no wildcards
ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com

# Storage — Cloudflare R2 in production
STORAGE_ENDPOINT=ACCOUNT_ID.r2.cloudflarestorage.com
STORAGE_ACCESS_KEY=R2_ACCESS_KEY_ID
STORAGE_SECRET_KEY=R2_SECRET_ACCESS_KEY
STORAGE_BUCKET=marketplace
STORAGE_USE_SSL=true
STORAGE_PUBLIC_URL=https://pub-HASH.r2.dev

# Search
TYPESENSE_HOST=https://typesense.example.com
TYPESENSE_API_KEY=GENERATE_WITH_OPENSSL_RAND_HEX_32

# Embeddings
OPENAI_API_KEY=sk-proj-PLACEHOLDER
OPENAI_EMBEDDINGS_MODEL=text-embedding-3-small

# Email (Resend)
RESEND_API_KEY=re_PLACEHOLDER
RESEND_FROM_EMAIL=hello@example.com

# Stripe
STRIPE_SECRET_KEY=sk_live_PLACEHOLDER
STRIPE_WEBHOOK_SECRET=whsec_PLACEHOLDER
STRIPE_CONNECT_CLIENT_ID=ca_PLACEHOLDER

# LiveKit
LIVEKIT_URL=wss://your-instance.livekit.cloud
LIVEKIT_API_KEY=APIxxxxxxxxxxxxxxx
LIVEKIT_API_SECRET=PLACEHOLDER

# Push notifications (Firebase service account JSON, base64-encoded)
FCM_SERVICE_ACCOUNT_BASE64=PLACEHOLDER

# Invoicing — issuer (the platform's own legal identity)
INVOICE_ISSUER_NAME="Marketplace Example SAS"
INVOICE_ISSUER_LEGAL_FORM="SAS"
INVOICE_ISSUER_ADDRESS_LINE1="1 rue de la République"
INVOICE_ISSUER_POSTAL_CODE="75001"
INVOICE_ISSUER_CITY="Paris"
INVOICE_ISSUER_COUNTRY="FR"
INVOICE_ISSUER_SIRET="00000000000000"
INVOICE_ISSUER_APE_CODE="6201Z"
INVOICE_ISSUER_EMAIL="billing@example.com"
INVOICE_ISSUER_RCS_EXEMPT="false"

# Text moderation provider — openai (default), anthropic, comprehend, noop
TEXT_MODERATION_PROVIDER=openai

# Audit log database role (optional but strongly recommended)
DATABASE_URL_AUDIT=postgres://AUDIT_USER:PASSWORD@HOST/DBNAME?sslmode=require
```

### Health checks

Configure Railway's health check to `GET /ready` (not `/health`) on
the backend service. `/ready` returns 503 if Postgres, Redis, or
Typesense is unreachable, so a misconfigured instance is rotated out
automatically.

### Run command

```bash
./api
```

The Dockerfile already builds and exposes the binary at the image
root. No `make` or `go run` in production.

### Migrations

Railway does not auto-run migrations. Run them as a one-off job before
each deploy:

```bash
railway run --service backend make migrate-up
```

For a zero-downtime release, deploy a "migrate-only" job first, then
the application service. See `docs/ops.md` §1 for the full deploy
order across backend / web / mobile.

---

## 3. Web on Vercel

### Build configuration

- **Framework preset**: Next.js
- **Root directory**: `web`
- **Build command**: `npm run build`
- **Output directory**: `.next` (auto-detected)
- **Node version**: 20

### Environment variables

```bash
# Public — safe to ship to the browser
NEXT_PUBLIC_API_URL=https://api.example.com
NEXT_PUBLIC_APP_URL=https://app.example.com
NEXT_PUBLIC_TYPESENSE_HOST=https://typesense.example.com
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_PLACEHOLDER

# Server-only — never prefixed with NEXT_PUBLIC_
INTERNAL_API_URL=http://backend.railway.internal:8080  # if Railway
SESSION_SECRET=GENERATE_WITH_OPENSSL_RAND_HEX_32
```

The web app is mostly Server Components. The internal-network
`INTERNAL_API_URL` lets server-side fetches skip the public DNS hop;
the public `NEXT_PUBLIC_API_URL` is what the browser uses.

### Custom domain

1. Add `app.example.com` as a domain in Vercel project settings.
2. Set the CNAME record at your registrar to `cname.vercel-dns.com`.
3. Wait for the SSL certificate to provision (Vercel handles this
   automatically with Let's Encrypt).

### CSP and security headers

The defaults are set via Next.js's `headers()` config in
`web/next.config.ts`. Vercel does not strip these. If you have
custom domain rewrites, double-check that the CSP `connect-src`
allow-list includes your backend, Typesense, and Stripe Elements
hosts.

---

## 4. Admin on Vercel (or any static host)

The admin app is a Vite SPA. Same flow as web with two changes:

- **Framework preset**: Other (Vite)
- **Build command**: `npm run build`
- **Output directory**: `dist`

### Environment variables

```bash
VITE_API_URL=https://api.example.com
VITE_TYPESENSE_HOST=https://typesense.example.com
```

Vite inlines `VITE_*` variables at build time, so a deploy is
required after rotating any of them.

---

## 5. PostgreSQL on Neon

Neon is the recommended managed Postgres because of its branching
feature: every PR can spin up an isolated DB copy in seconds, which
fits the project's parallel-agent workflow perfectly.

### Project setup

1. Create a project in the EU region (Frankfurt or Dublin) for GDPR.
2. Choose **PostgreSQL 16**.
3. Enable connection pooling — pick the pooled endpoint
   (`-pooler.<region>.aws.neon.tech`) for the application.

### Connection string format

```
postgres://USER:PASSWORD@ep-XXXX-pooler.eu-central-1.aws.neon.tech/DB?sslmode=require
```

Set `DATABASE_URL` to the **pooled** endpoint. The unpooled endpoint
is for migrations and admin work only.

### Pool config (in code)

The backend already configures the connection pool in `main.go`:

```go
db.SetMaxIdleConns(25)
db.SetMaxOpenConns(50)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(2 * time.Minute)
```

Neon's pooled endpoint can hold more sessions than a typical Postgres,
but **`MaxOpenConns` should never exceed your plan's session limit**.
Check the Neon dashboard before raising it.

### Two DB roles for RLS

The application user must NOT own the tables — otherwise RLS is
bypassed. Create two distinct roles in Neon:

```sql
-- One-time setup, run as the project owner
CREATE ROLE migrator WITH LOGIN PASSWORD '<migrator-password>';
CREATE ROLE app      WITH LOGIN PASSWORD '<app-password>';

GRANT CONNECT ON DATABASE marketplace TO migrator, app;
GRANT USAGE ON SCHEMA public TO migrator, app;

-- Migrations run as `migrator`. The app connects as `app`.
ALTER DEFAULT PRIVILEGES FOR ROLE migrator IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app;

-- audit_logs is INSERT/SELECT only (migration 124 enforces this)
REVOKE UPDATE, DELETE ON TABLE audit_logs FROM app;
```

Use `migrator` for `make migrate-up`. Use `app` for the running
backend. The setup is documented in detail in `backend/docs/rls.md`.

### Backups

Neon takes automatic point-in-time backups. For an extra paid layer,
enable daily logical dumps to R2 with `pg_dump`.

---

## 6. Cloudflare R2 (object storage)

R2 is S3-compatible, has zero egress fees, and is the cheapest
option for the file types the marketplace stores (avatars, portfolio
images, invoice PDFs).

### Bucket setup

1. Create a bucket named `marketplace` in your R2 account.
2. Create an API token with **Object Read & Write** scope and
   restrict it to that single bucket.
3. (Optional but recommended) Configure a custom domain (e.g.
   `cdn.example.com`) routing to the public bucket subdomain.

### CORS rules

Set the bucket CORS policy via the Cloudflare dashboard or `wrangler`:

```json
[
  {
    "AllowedOrigins": [
      "https://app.example.com",
      "https://admin.example.com"
    ],
    "AllowedMethods": ["GET", "PUT", "POST", "DELETE", "HEAD"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag", "Content-Length"],
    "MaxAgeSeconds": 600
  }
]
```

The `MaxAgeSeconds: 600` matches what we set in
`backend/internal/handler/middleware/cors.go` (reduced from 86400 in
Phase 0 — see SEC-36 in `auditsecurite.md`).

### Object naming

The backend writes objects with randomized keys (UUID + content-type
extension). Original filenames are never used in storage paths — this
prevents path traversal and metadata leaks. See
`backend/internal/adapter/s3/uploader.go` for the convention.

### Public URL

Your `STORAGE_PUBLIC_URL` should point to the bucket's public hostname
(`https://pub-HASH.r2.dev`) or your custom domain. The backend rewrites
upload responses to this URL.

---

## 7. Resend (transactional email)

Resend powers password reset, welcome, KYC reminder, and invoice
delivery emails.

### Domain authentication

1. Add your sending domain in the Resend dashboard.
2. Configure SPF, DKIM, and DMARC records at your DNS provider as
   shown by Resend.
3. Wait for verification (typically 5-30 minutes).

### Required env vars

```bash
RESEND_API_KEY=re_PLACEHOLDER
RESEND_FROM_EMAIL=hello@example.com
```

The `RESEND_FROM_EMAIL` must match an authenticated domain. The
backend will fail fast at boot if it is not set.

### Templates

Email templates live in
`backend/internal/adapter/resend/templates/` as Go `html/template`
files. They are versioned with the code, not stored in Resend.

---

## 8. LiveKit Cloud (video calls)

The LiveKit feature is **off-limits for code edits** in our
contributor flow (it works, do not touch). For deploy:

1. Create a LiveKit Cloud project in an EU region.
2. Generate API key + secret.
3. Set:

```bash
LIVEKIT_URL=wss://YOUR-INSTANCE.livekit.cloud
LIVEKIT_API_KEY=APIxxxxxxxxxxxxxxx
LIVEKIT_API_SECRET=PLACEHOLDER
```

4. The backend mints scoped JWTs per call; clients connect directly
   to LiveKit Cloud over the WSS URL above.

LiveKit's free tier covers prototype usage. Past that, switch to a
metered plan — bandwidth is the dominant cost.

---

## 9. Stripe Connect + Embedded Components

The hardest service to set up and the most important to get right.
Read this section twice before each deploy.

### Account setup

1. Create a Stripe account in the **EU region** (registered in your
   incorporation country).
2. Activate **Stripe Connect Custom**. The platform onboards
   providers via Embedded Components — they never see the Stripe
   dashboard.
3. Note the platform's `acct_xxx` ID — it will appear in API calls
   as `Stripe-Account` for some Connect endpoints.

### Webhook endpoint

Configure ONE webhook endpoint in Stripe Dashboard → Developers →
Webhooks:

```
https://api.example.com/api/v1/stripe/webhooks
```

Events to listen for:

```
account.updated
payment_intent.succeeded
payment_intent.payment_failed
charge.refunded
charge.dispute.created
transfer.created
transfer.failed
invoice.paid
invoice.payment_failed
customer.subscription.updated
customer.subscription.deleted
checkout.session.completed
```

Copy the **Signing secret** into `STRIPE_WEBHOOK_SECRET`.

### Why a single webhook URL

The backend uses the `stripe_webhook_events` table for idempotency
(UNIQUE on `event_id`). One URL, one row per event, one processing
ever. Splitting events across multiple URLs (one per type) is
tempting but breaks the idempotency guarantee.

### Embedded Components

KYC + payouts onboarding go through Embedded Components on the
frontend. The backend mints a per-user account session via
`POST /api/v1/embedded/account-sessions`. See
`backend/internal/handler/embedded_handler.go` and the playbook
`STRIPE_MANUAL_PLAYBOOK.md` at the repo root for the full flow.

### Test mode vs live mode

Always start with `sk_test_*` and `pk_test_*` keys. Run the full
proposal-to-payout flow against Stripe test cards (`4242 4242 4242
4242`). Promote to `sk_live_*` only after the smoke tests in
`./scripts/stripe-smoke-test.sh` are green.

---

## 10. First-deploy checklist

Run through this in order. Skipping a step has cost us a half-day at
least once each.

### Pre-deploy

- [ ] Repository is on a tagged commit (`git tag v0.1.0`) — no
      ephemeral SHAs in production.
- [ ] CI is green on that tag (`ci.yml`, `e2e.yml`, `security.yml`).
- [ ] `gosec` and `trivy` outputs are reviewed — any new HIGH or
      CRITICAL is acknowledged in `docs/ops.md` or fixed before deploy.
- [ ] `npm audit --audit-level=high` is clean on web and admin.
- [ ] `flutter test` is green on the mobile artefact.

### Infrastructure

- [ ] Neon project created in EU region, two roles configured
      (`migrator`, `app`), connection pooling enabled.
- [ ] Upstash Redis instance created in EU region, REDIS_URL noted.
- [ ] Cloudflare R2 bucket created, API token scoped to that bucket,
      CORS rules applied.
- [ ] Typesense self-hosted on Railway, master API key generated and
      stored in the secret manager (32-byte hex, generated with
      `openssl rand -hex 32`).
- [ ] Resend domain authenticated (SPF/DKIM/DMARC green).
- [ ] LiveKit project created, API key + secret generated.
- [ ] Stripe Connect activated, webhook endpoint registered, signing
      secret captured.

### Backend

- [ ] All env vars from §2 set in Railway.
- [ ] `make migrate-up` run with `migrator` role on the production DB.
- [ ] `make seed` run for default roles + initial admin user.
- [ ] Backend service deployed; `/ready` returns 200.
- [ ] Smoke test: `curl https://api.example.com/health` → `{"status":"ok"}`.

### Frontends

- [ ] Web `NEXT_PUBLIC_*` vars set on Vercel.
- [ ] Web first deploy successful, custom domain SSL provisioned.
- [ ] Admin `VITE_*` vars set, first deploy successful.
- [ ] Mobile build artefacts signed and uploaded to TestFlight + Play
      Console internal track.

### End-to-end smoke

- [ ] Register an Agency, an Enterprise, and a Provider through the
      web UI (or by curl against the public endpoints).
- [ ] Provider completes KYC via Embedded Components.
- [ ] Enterprise creates a job, Provider sends a proposal.
- [ ] Enterprise pays through Stripe (test mode) — verify the
      `payment_records` row reaches `status=funded`.
- [ ] Provider releases a milestone — verify a transfer reaches the
      provider's connected account.
- [ ] Verify an invoice PDF is generated and the email is delivered.
- [ ] Verify the search index has the new profiles
      (`/agencies`, `/freelancers`, `/referrers` listings populate).
- [ ] Run `./scripts/ci/security-baseline.sh --env production` —
      every check should be green.
- [ ] Run `./scripts/ci/rbac-matrix.sh --base https://api.example.com` —
      no leaks across roles.

### Day one

- [ ] Configure uptime monitoring on `/ready` (every 30s).
- [ ] Configure log alerting for `level=ERROR` aggregation.
- [ ] Configure Stripe alerting for failed transfers and disputes.
- [ ] Schedule a Postgres backup test once a quarter.

---

## 11. Rolling back

A bad deploy on Railway is recoverable in seconds: use the **rollback
to previous deploy** button. The backend's migration history makes
this safe as long as the previous deploy was running against a
compatible schema (additive migrations) — which is the case for every
migration we have shipped.

If a migration itself is the cause:

- **Forward fix.** Create a new migration
  (`<NNN>_fix_<thing>.up.sql`) that corrects the schema. Never
  `migrate-down` on a shared DB. See `CLAUDE.md` lines 575-602 for
  the full rule.

If web is broken on Vercel: roll back to the previous deploy from the
project's Deployments tab. The DNS does not change; the swap is
atomic.

---

## 12. Cost envelope (rough)

For a small production deployment with a few thousand monthly active
users:

| Item | Provider | Monthly |
|------|----------|---------|
| Backend (1 vCPU / 1GB) | Railway | $10 |
| Web | Vercel Hobby/Pro | $0-$20 |
| Admin | Vercel | $0 |
| Postgres (Pro plan) | Neon | $19 |
| Redis | Upstash free or pay-as-you-go | $0-$10 |
| R2 storage (50GB + 1M ops) | Cloudflare | ~$1 |
| Typesense (1 vCPU) | Railway | $5 |
| OpenAI embeddings | OpenAI | ~$1 per 50k profiles indexed |
| Email | Resend free tier (3k/mo) | $0 |
| LiveKit | Free tier | $0 |
| Stripe | per-transaction fee, no monthly base | — |

Total floor: about $35-$70/month. Scaling up is mostly Railway,
Neon, and OpenAI; everything else stays flat for a long time.

---

For day-two operations (deploy order, reindexes, key rotation, drift
alerts, slow-query triage) read [`docs/ops.md`](ops.md).
