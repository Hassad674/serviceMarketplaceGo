/**
 * Phase 1 backend contract E2E test — web perspective.
 *
 * Validates the same backend API contract as
 * backend/test/e2e/phase1_e2e.sh but from the Playwright test harness
 * used by the web app. Runs entirely via APIRequestContext — no browser
 * navigation, no Next.js server, no cookies — so it focuses purely on
 * what the web HTTP client will see.
 *
 * Prerequisites:
 *   - The team E2E backend must be running on http://localhost:8084
 *     against the isolated marketplace_go_team DB. Run the orchestrator
 *     bash script first (or start it manually — see PROGRESS.md CP1).
 *
 * Run:
 *   cd web && npx playwright test e2e/team-phase1-contract.spec.ts --reporter=list
 */

import { test, expect, APIRequestContext, request } from "@playwright/test"

const BACKEND_URL = process.env.TEAM_E2E_BACKEND_URL ?? "http://localhost:8084"
const TS = Date.now()

// Small helper so we can use a dedicated APIRequestContext that always
// targets the team E2E backend, regardless of the project's baseURL.
let api: APIRequestContext

test.beforeAll(async () => {
  api = await request.newContext({ baseURL: BACKEND_URL })

  // Skip the whole suite if the backend isn't running — makes CI fail
  // loudly instead of hanging.
  const health = await api.get("/health").catch(() => null)
  if (!health || !health.ok()) {
    throw new Error(
      `Team E2E backend not reachable at ${BACKEND_URL}. ` +
        `Start it first via backend/test/e2e/phase1_e2e.sh or manually ` +
        `(see PROGRESS.md CP1).`,
    )
  }
})

test.afterAll(async () => {
  await api.dispose()
})

// ---- shared payloads ----

function agencyPayload(ts: number) {
  return {
    email: `agency-web-${ts}@phase1.test`,
    password: "TestPass1!",
    first_name: "Sarah",
    last_name: "Connor",
    display_name: "Acme Corp",
    role: "agency",
  }
}

function enterprisePayload(ts: number) {
  return {
    email: `enterprise-web-${ts}@phase1.test`,
    password: "TestPass1!",
    first_name: "John",
    last_name: "Smith",
    display_name: "Enterprise SAS",
    role: "enterprise",
  }
}

function providerPayload(ts: number) {
  return {
    email: `provider-web-${ts}@phase1.test`,
    password: "TestPass1!",
    first_name: "Marie",
    last_name: "Durand",
    role: "provider",
  }
}

// The backend is expected to grant the Owner all 21 permissions defined
// in backend/internal/domain/organization/permissions.go.
const OWNER_PERMISSIONS_COUNT = 21

const OWNER_CRITICAL_PERMS = [
  "wallet.withdraw",
  "team.transfer_ownership",
  "org.delete",
  "billing.manage",
  "kyc.manage",
]

// ---- tests ----

test("TEST 1 — Agency registration auto-provisions organization with Owner", async () => {
  const res = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: agencyPayload(TS),
  })
  expect(res.status()).toBe(201)

  const body = await res.json()

  expect(body.user.email).toBe(agencyPayload(TS).email)
  expect(body.user.role).toBe("agency")
  expect(body.user.account_type).toBe("marketplace_owner")
  expect(body.access_token).toBeTruthy()
  expect(body.refresh_token).toBeTruthy()

  expect(body.organization).toBeTruthy()
  expect(body.organization.type).toBe("agency")
  expect(body.organization.member_role).toBe("owner")
  expect(body.organization.owner_user_id).toBe(body.user.id)
  expect(body.organization.permissions).toHaveLength(OWNER_PERMISSIONS_COUNT)

  for (const perm of OWNER_CRITICAL_PERMS) {
    expect(body.organization.permissions).toContain(perm)
  }
})

test("TEST 2 — Enterprise registration auto-provisions organization with Owner", async () => {
  const res = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: enterprisePayload(TS),
  })
  expect(res.status()).toBe(201)

  const body = await res.json()
  expect(body.user.role).toBe("enterprise")
  expect(body.organization.type).toBe("enterprise")
  expect(body.organization.member_role).toBe("owner")
  expect(body.organization.owner_user_id).toBe(body.user.id)
  expect(body.organization.permissions).toHaveLength(OWNER_PERMISSIONS_COUNT)
})

test("TEST 3 — Provider registration creates solo user (no organization)", async () => {
  const res = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: providerPayload(TS),
  })
  expect(res.status()).toBe(201)

  const body = await res.json()
  expect(body.user.role).toBe("provider")
  expect(body.user.account_type).toBe("marketplace_owner")
  expect(body.access_token).toBeTruthy()

  // CRITICAL: Provider response must NOT include an organization field.
  expect(body.organization).toBeFalsy()

  // Decode the JWT payload (middle segment, base64url) and ensure no
  // org claims leaked into a Provider token.
  const payloadB64 = body.access_token.split(".")[1]
  const normalized = payloadB64.replace(/-/g, "+").replace(/_/g, "/")
  const padded = normalized + "===".slice((normalized.length + 3) % 4)
  const payload = JSON.parse(Buffer.from(padded, "base64").toString("utf-8"))

  expect(payload.user_id).toBeTruthy()
  expect(payload.role).toBe("provider")
  expect(payload.org_id).toBeFalsy()
  expect(payload.org_role).toBeFalsy()
})

test("TEST 4 — GET /me for Agency returns user + organization", async () => {
  // Register a fresh agency to get a clean token for this test
  const reg = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: {
      ...agencyPayload(TS),
      email: `agency-web-me-${TS}@phase1.test`,
    },
  })
  const regBody = await reg.json()
  const token = regBody.access_token as string

  const me = await api.get("/api/v1/auth/me", {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(me.status()).toBe(200)
  const body = await me.json()

  expect(body.user.id).toBe(regBody.user.id)
  expect(body.user.role).toBe("agency")

  expect(body.organization).toBeTruthy()
  expect(body.organization.type).toBe("agency")
  expect(body.organization.member_role).toBe("owner")
  expect(body.organization.permissions).toHaveLength(OWNER_PERMISSIONS_COUNT)
})

test("TEST 5 — GET /me for Provider returns user only (no organization)", async () => {
  const reg = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: {
      ...providerPayload(TS),
      email: `provider-web-me-${TS}@phase1.test`,
    },
  })
  const token = (await reg.json()).access_token as string

  const me = await api.get("/api/v1/auth/me", {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(me.status()).toBe(200)
  const body = await me.json()

  expect(body.user.role).toBe("provider")
  expect(body.user.account_type).toBe("marketplace_owner")
  expect(body.organization).toBeFalsy()
})

test("TEST 6 — Duplicate email registration returns 409", async () => {
  const email = `dup-web-${TS}@phase1.test`

  const first = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: { ...agencyPayload(TS), email },
  })
  expect(first.status()).toBe(201)

  const second = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: { ...enterprisePayload(TS), email },
  })
  expect(second.status()).toBe(409)
})
