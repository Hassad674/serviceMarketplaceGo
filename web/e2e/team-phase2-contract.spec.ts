/**
 * Phase 2 backend contract E2E test — web perspective.
 *
 * Validates the team invitation flow from the Playwright test harness
 * used by the web app. Runs entirely via APIRequestContext — no
 * browser, no Next.js server — so it focuses purely on what the web
 * HTTP client will see when calling the invitation endpoints.
 *
 * Prerequisites:
 *   - The team E2E backend must be running on http://localhost:8084
 *     against the isolated marketplace_go_team DB.
 *
 * Run:
 *   cd web && PLAYWRIGHT_BASE_URL=http://localhost:8084 \
 *     TEAM_E2E_BACKEND_URL=http://localhost:8084 \
 *     npx playwright test e2e/team-phase2-contract.spec.ts \
 *     --reporter=list --project=chromium
 */

import { test, expect, APIRequestContext, request } from "@playwright/test"

const BACKEND_URL = process.env.TEAM_E2E_BACKEND_URL ?? "http://localhost:8084"
const TS = Date.now()

let api: APIRequestContext
let ownerToken: string
let orgId: string

test.describe.configure({ mode: "serial" })

test.beforeAll(async () => {
  api = await request.newContext({ baseURL: BACKEND_URL })

  const health = await api.get("/health").catch(() => null)
  if (!health || !health.ok()) {
    throw new Error(
      `Team E2E backend not reachable at ${BACKEND_URL}. Start it first.`,
    )
  }

  // Register the Agency Owner once for the whole suite.
  const reg = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: {
      email: `agency-web-p2-${TS}@phase2.test`,
      password: "TestPass1!",
      first_name: "Sarah",
      last_name: "Connor",
      display_name: "Acme Corp",
      role: "agency",
    },
  })
  expect(reg.status()).toBe(201)
  const body = await reg.json()
  ownerToken = body.access_token
  orgId = body.organization.id
})

test.afterAll(async () => {
  await api.dispose()
})

test("TEST 1 — Owner sends an invitation (201, returns pending row)", async () => {
  const inviteeEmail = `invitee-web-${TS}@phase2.test`

  const res = await api.post(`/api/v1/organizations/${orgId}/invitations`, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ownerToken}`,
    },
    data: {
      email: inviteeEmail,
      first_name: "Paul",
      last_name: "Dupont",
      title: "Office Manager",
      role: "member",
    },
  })
  expect(res.status()).toBe(201)

  const body = await res.json()
  expect(body.id).toBeTruthy()
  expect(body.email).toBe(inviteeEmail)
  expect(body.role).toBe("member")
  expect(body.status).toBe("pending")
  expect(body.organization_id).toBe(orgId)
  // Token must NEVER appear in the API response payload.
  expect(body.token).toBeUndefined()
})

test("TEST 2 — List pending invitations returns items (>= 1)", async () => {
  const res = await api.get(`/api/v1/organizations/${orgId}/invitations`, {
    headers: { Authorization: `Bearer ${ownerToken}` },
  })
  expect(res.status()).toBe(200)
  const body = await res.json()
  expect(Array.isArray(body.data)).toBeTruthy()
  expect(body.data.length).toBeGreaterThanOrEqual(1)
  expect(body.data[0].status).toBe("pending")
})

test("TEST 3 — Duplicate pending invitation for same email → 409", async () => {
  const email = `dup-web-${TS}@phase2.test`
  const first = await api.post(`/api/v1/organizations/${orgId}/invitations`, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ownerToken}`,
    },
    data: {
      email,
      first_name: "Dup",
      last_name: "One",
      role: "member",
    },
  })
  expect(first.status()).toBe(201)

  const second = await api.post(`/api/v1/organizations/${orgId}/invitations`, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ownerToken}`,
    },
    data: {
      email,
      first_name: "Dup",
      last_name: "Two",
      role: "viewer",
    },
  })
  expect(second.status()).toBe(409)
})

test("TEST 4 — Inviting with role=owner rejected (400)", async () => {
  const res = await api.post(`/api/v1/organizations/${orgId}/invitations`, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${ownerToken}`,
    },
    data: {
      email: `owner-role-web-${TS}@phase2.test`,
      first_name: "X",
      last_name: "Y",
      role: "owner",
    },
  })
  expect(res.status()).toBe(400)
})

test("TEST 5 — Provider cannot send invitations (403)", async () => {
  // Register a fresh provider
  const reg = await api.post("/api/v1/auth/register", {
    headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    data: {
      email: `provider-web-p2-${TS}@phase2.test`,
      password: "TestPass1!",
      first_name: "Marie",
      last_name: "D",
      role: "provider",
    },
  })
  const providerToken = (await reg.json()).access_token

  const res = await api.post(`/api/v1/organizations/${orgId}/invitations`, {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${providerToken}`,
    },
    data: {
      email: `blocked-${TS}@phase2.test`,
      first_name: "N",
      last_name: "A",
      role: "member",
    },
  })
  expect(res.status()).toBe(403)
})

test("TEST 6 — Cancel pending invitation → 204", async () => {
  const createRes = await api.post(
    `/api/v1/organizations/${orgId}/invitations`,
    {
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${ownerToken}`,
      },
      data: {
        email: `cancel-web-${TS}@phase2.test`,
        first_name: "Cancel",
        last_name: "Me",
        role: "viewer",
      },
    },
  )
  expect(createRes.status()).toBe(201)
  const { id } = await createRes.json()

  const cancelRes = await api.delete(
    `/api/v1/organizations/${orgId}/invitations/${id}`,
    { headers: { Authorization: `Bearer ${ownerToken}` } },
  )
  expect(cancelRes.status()).toBe(204)
})

test("TEST 7 — Missing token on /validate → 400", async () => {
  const res = await api.get("/api/v1/invitations/validate")
  expect(res.status()).toBe(400)
})

test("TEST 8 — Accept with bogus token → 404", async () => {
  const res = await api.post("/api/v1/invitations/accept", {
    headers: { "Content-Type": "application/json" },
    data: {
      token: "this_is_not_a_real_token_0000000000000000000000000000",
      password: "StrongPass1!",
    },
  })
  expect(res.status()).toBe(404)
})
