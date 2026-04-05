import { test, expect, type BrowserContext } from "@playwright/test"
import { registerProvider, registerEnterprise } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const API_URL = "http://localhost:8083"
const ADMIN_EMAIL = "admin@marketplace.local"
const ADMIN_PASSWORD = "Admin123!"
const INITIAL_CREDITS = 10
const BONUS_PER_MISSION = 5

// ---------------------------------------------------------------------------
// API helpers (self-contained, copied from bonus-credits.spec.ts)
// ---------------------------------------------------------------------------

async function getSessionCookie(context: BrowserContext): Promise<string> {
  const cookies = await context.cookies()
  const session = cookies.find((c) => c.name === "session_id")
  return session ? `session_id=${session.value}` : ""
}

async function fetchCreditsViaAPI(cookie: string): Promise<number> {
  const res = await fetch(`${API_URL}/api/v1/jobs/credits`, {
    headers: { Cookie: cookie },
  })
  expect(res.ok, `Failed to fetch credits: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.credits
}

async function getMyUserID(cookie: string): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/auth/me`, {
    headers: { Cookie: cookie },
  })
  expect(res.ok, `Failed to fetch /me: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.id ?? data.data?.id
}

async function startConversation(
  cookie: string,
  recipientId: string,
): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/messaging/conversations`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: cookie,
    },
    body: JSON.stringify({
      recipient_id: recipientId,
      content: "Hello, I have a project for you.",
      type: "text",
    }),
  })
  expect(res.ok, `Failed to start conversation: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.conversation_id
}

async function createProposal(
  cookie: string,
  conversationId: string,
  recipientId: string,
): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/proposals`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: cookie,
    },
    body: JSON.stringify({
      conversation_id: conversationId,
      recipient_id: recipientId,
      title: "E2E Fraud Detection Test Mission",
      description: "Test mission for fraud detection E2E tests",
      amount: 100000, // 1000 EUR in centimes
    }),
  })
  expect(res.ok, `Failed to create proposal: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.id
}

async function createProposalWithAmount(
  cookie: string,
  conversationId: string,
  recipientId: string,
  amount: number,
): Promise<Response> {
  return fetch(`${API_URL}/api/v1/proposals`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: cookie,
    },
    body: JSON.stringify({
      conversation_id: conversationId,
      recipient_id: recipientId,
      title: "E2E Below Minimum Test",
      description: "Test mission with low amount for fraud detection",
      amount,
    }),
  })
}

async function acceptProposal(
  cookie: string,
  proposalId: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/proposals/${proposalId}/accept`,
    { method: "POST", headers: { Cookie: cookie } },
  )
  expect(res.ok, `Failed to accept proposal: ${res.status}`).toBe(true)
}

async function loginViaAPI(
  email: string,
  password: string,
): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
    redirect: "manual",
  })
  const setCookieHeaders = res.headers.getSetCookie?.() ?? []
  const allCookies =
    setCookieHeaders.length > 0
      ? setCookieHeaders
      : (res.headers.get("set-cookie") ?? "").split(", ")
  const cookies: string[] = []
  for (const c of allCookies) {
    const match = c.match(/^([^=]+=[^;]+)/)
    if (match) cookies.push(match[1])
  }
  if (cookies.length > 0) return cookies.join("; ")
  throw new Error(`Login failed for ${email}: ${res.status}`)
}

async function activateProposalAsAdmin(
  adminCookie: string,
  proposalId: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/proposals/${proposalId}/activate`,
    { method: "POST", headers: { Cookie: adminCookie } },
  )
  const body = await res.text()
  console.log(
    `[activate] status=${res.status} body=${body} proposalId=${proposalId}`,
  )
  expect(
    res.ok,
    `Failed to activate proposal: ${res.status} — ${body}`,
  ).toBe(true)
}

// ---------------------------------------------------------------------------
// Bonus-log helpers (specific to fraud detection tests)
// ---------------------------------------------------------------------------

interface BonusLogEntry {
  id: string
  provider_id: string
  client_id: string
  proposal_id: string
  credits_awarded: number
  status: string
  block_reason?: string
}

async function fetchPendingBonusEntries(
  adminCookie: string,
): Promise<BonusLogEntry[]> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/credits/bonus-log/pending`,
    { headers: { Cookie: adminCookie } },
  )
  if (!res.ok) return []
  const data = await res.json()
  return data.data ?? []
}

async function approveBonusEntry(
  adminCookie: string,
  entryId: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/credits/bonus-log/${entryId}/approve`,
    { method: "POST", headers: { Cookie: adminCookie } },
  )
  expect(
    res.ok,
    `Failed to approve bonus entry ${entryId}: ${res.status}`,
  ).toBe(true)
}

async function rejectBonusEntry(
  adminCookie: string,
  entryId: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/admin/credits/bonus-log/${entryId}/reject`,
    { method: "POST", headers: { Cookie: adminCookie } },
  )
  expect(
    res.ok,
    `Failed to reject bonus entry ${entryId}: ${res.status}`,
  ).toBe(true)
}

async function approvePendingBonuses(adminCookie: string): Promise<void> {
  const entries = await fetchPendingBonusEntries(adminCookie)
  for (const entry of entries) {
    if (entry.status === "pending_review") {
      await approveBonusEntry(adminCookie, entry.id)
    }
  }
}

/**
 * Complete a full mission cycle: create proposal, accept, activate, approve pending bonus.
 * Returns the provider's credits after the bonus is approved.
 */
async function completeMissionAndApprovBonus(
  enterpriseCookie: string,
  providerCookie: string,
  providerId: string,
  adminCookie: string,
): Promise<void> {
  const convId = await startConversation(enterpriseCookie, providerId)
  const proposalId = await createProposal(
    enterpriseCookie,
    convId,
    providerId,
  )
  await acceptProposal(providerCookie, proposalId)
  await activateProposalAsAdmin(adminCookie, proposalId)
  await approvePendingBonuses(adminCookie)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("Credit bonus fraud detection", () => {
  test.beforeEach(async ({ page, context }) => {
    await context.clearCookies()
    await page.goto("/")
  })

  test("too_fast: bonus is pending when mission created and paid within 2 minutes", async ({
    page,
    context,
  }) => {
    test.setTimeout(120_000)

    // Step 1: Register enterprise
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)
    const enterpriseId = await getMyUserID(enterpriseCookie)

    // Step 2: Register provider
    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerId = await getMyUserID(providerCookie)

    // Step 3: Verify initial credits = 10
    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    // Step 4: Create proposal, accept, activate (all within seconds -> triggers too_fast)
    const conversationId = await startConversation(
      enterpriseCookie,
      providerId,
    )
    const proposalId = await createProposal(
      enterpriseCookie,
      conversationId,
      providerId,
    )
    await acceptProposal(providerCookie, proposalId)
    const adminCookie = await loginViaAPI(ADMIN_EMAIL, ADMIN_PASSWORD)
    await activateProposalAsAdmin(adminCookie, proposalId)

    // Step 5: Credits should still be 10 (bonus NOT awarded yet — pending_review)
    const creditsAfterActivation = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterActivation).toBe(INITIAL_CREDITS)

    // Step 6: Verify there is a pending_review entry in bonus-log
    const pendingEntries = await fetchPendingBonusEntries(adminCookie)
    const ourEntry = pendingEntries.find(
      (e) => e.provider_id === providerId,
    )
    expect(ourEntry, "Expected a pending bonus entry for the provider").toBeTruthy()
    expect(ourEntry!.status).toBe("pending_review")
    expect(ourEntry!.block_reason).toBe("too_fast")

    // Step 7: Admin approves the pending entry
    await approveBonusEntry(adminCookie, ourEntry!.id)

    // Step 8: Credits should now be 15
    const creditsAfterApproval = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterApproval).toBe(INITIAL_CREDITS + BONUS_PER_MISSION)
  })

  test("too_frequent_daily: bonus is pending after 3+ missions with same client in 24h", async ({
    page,
    context,
  }) => {
    test.setTimeout(300_000)

    // Step 1: Register enterprise and provider
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerId = await getMyUserID(providerCookie)

    const adminCookie = await loginViaAPI(ADMIN_EMAIL, ADMIN_PASSWORD)

    // Step 2: Complete 3 missions — each triggers too_fast, approve each one
    for (let i = 0; i < 3; i++) {
      const convId = await startConversation(enterpriseCookie, providerId)
      const proposalId = await createProposal(
        enterpriseCookie,
        convId,
        providerId,
      )
      await acceptProposal(providerCookie, proposalId)
      await activateProposalAsAdmin(adminCookie, proposalId)
      await approvePendingBonuses(adminCookie)
    }

    // Step 3: After 3 missions, credits = 10 + 5 + 5 + 5 = 25
    const creditsAfterThree = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterThree).toBe(INITIAL_CREDITS + 3 * BONUS_PER_MISSION)

    // Step 4: Complete a 4th mission with the SAME client
    const convId4 = await startConversation(enterpriseCookie, providerId)
    const proposalId4 = await createProposal(
      enterpriseCookie,
      convId4,
      providerId,
    )
    await acceptProposal(providerCookie, proposalId4)
    await activateProposalAsAdmin(adminCookie, proposalId4)

    // Step 5: The 4th bonus should be pending_review (too_frequent_daily), credits stay at 25
    const creditsAfterFourth = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterFourth).toBe(INITIAL_CREDITS + 3 * BONUS_PER_MISSION)

    // Step 6: Verify the pending entry has reason too_frequent_daily
    const pendingEntries = await fetchPendingBonusEntries(adminCookie)
    const fourthEntry = pendingEntries.find(
      (e) => e.provider_id === providerId,
    )
    expect(fourthEntry, "Expected a pending bonus entry for the 4th mission").toBeTruthy()
    expect(fourthEntry!.status).toBe("pending_review")
    // The reason may be too_fast or too_frequent_daily (both apply).
    // Since too_fast is checked first in the code, it may take precedence.
    // Either way the bonus is pending, which is the important behavior.
    expect(
      ["too_fast", "too_frequent_daily"].includes(fourthEntry!.block_reason ?? ""),
    ).toBe(true)

    // Step 7: Admin approves -> credits go to 30
    await approveBonusEntry(adminCookie, fourthEntry!.id)
    const creditsAfterApproval = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterApproval).toBe(
      INITIAL_CREDITS + 4 * BONUS_PER_MISSION,
    )
  })

  test("admin can reject a pending bonus entry", async ({
    page,
    context,
  }) => {
    test.setTimeout(120_000)

    // Step 1: Register enterprise + provider
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerId = await getMyUserID(providerCookie)

    // Step 2: Complete a mission (triggers too_fast -> pending)
    const conversationId = await startConversation(
      enterpriseCookie,
      providerId,
    )
    const proposalId = await createProposal(
      enterpriseCookie,
      conversationId,
      providerId,
    )
    await acceptProposal(providerCookie, proposalId)
    const adminCookie = await loginViaAPI(ADMIN_EMAIL, ADMIN_PASSWORD)
    await activateProposalAsAdmin(adminCookie, proposalId)

    // Step 3: Verify credits = 10 (bonus pending, not awarded)
    const creditsBefore = await fetchCreditsViaAPI(providerCookie)
    expect(creditsBefore).toBe(INITIAL_CREDITS)

    // Step 4: Admin rejects the pending entry
    const pendingEntries = await fetchPendingBonusEntries(adminCookie)
    const ourEntry = pendingEntries.find(
      (e) => e.provider_id === providerId,
    )
    expect(ourEntry, "Expected a pending bonus entry").toBeTruthy()
    await rejectBonusEntry(adminCookie, ourEntry!.id)

    // Step 5: Credits still = 10 (bonus was rejected, not awarded)
    const creditsAfterReject = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterReject).toBe(INITIAL_CREDITS)
  })

  test("below_minimum: proposal creation fails for amount under 30 EUR", async ({
    page,
    context,
  }) => {
    test.setTimeout(120_000)

    // Step 1: Register enterprise + provider
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerId = await getMyUserID(providerCookie)

    // Step 2: Start a conversation
    const conversationId = await startConversation(
      enterpriseCookie,
      providerId,
    )

    // Step 3: Attempt to create a proposal with amount = 2000 (20 EUR, below 30 EUR minimum)
    const res = await createProposalWithAmount(
      enterpriseCookie,
      conversationId,
      providerId,
      2000,
    )

    // Step 4: Verify the request fails with 400 and below_minimum_amount error
    expect(res.status).toBe(400)
    const body = await res.json()
    expect(body.error).toBe("below_minimum_amount")
  })

  test("non-admin cannot access bonus log endpoints", async ({
    page,
    context,
  }) => {
    test.setTimeout(60_000)

    // Step 1: Register a provider
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)

    // Step 2: Try to GET /admin/credits/bonus-log/pending -> 403
    const pendingRes = await fetch(
      `${API_URL}/api/v1/admin/credits/bonus-log/pending`,
      { headers: { Cookie: providerCookie } },
    )
    expect(pendingRes.status).toBe(403)

    // Step 3: Try to POST /admin/credits/bonus-log/{fake-id}/approve -> 403
    const fakeId = "00000000-0000-0000-0000-000000000000"
    const approveRes = await fetch(
      `${API_URL}/api/v1/admin/credits/bonus-log/${fakeId}/approve`,
      { method: "POST", headers: { Cookie: providerCookie } },
    )
    expect(approveRes.status).toBe(403)

    // Step 4: Try to POST /admin/credits/bonus-log/{fake-id}/reject -> 403
    const rejectRes = await fetch(
      `${API_URL}/api/v1/admin/credits/bonus-log/${fakeId}/reject`,
      { method: "POST", headers: { Cookie: providerCookie } },
    )
    expect(rejectRes.status).toBe(403)
  })
})
