import { test, expect, type BrowserContext } from "@playwright/test"
import { registerProvider, registerEnterprise } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const API_URL = "http://localhost:8083"
const INITIAL_CREDITS = 10
const BONUS_PER_MISSION = 5

// ---------------------------------------------------------------------------
// API helpers
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
  expect(
    res.ok,
    `Failed to start conversation: ${res.status}`,
  ).toBe(true)
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
      title: "E2E Bonus Credits Test Mission",
      description: "Test mission to verify bonus credits are awarded after payment",
      amount: 100000, // 1000 EUR in centimes
    }),
  })
  expect(
    res.ok,
    `Failed to create proposal: ${res.status}`,
  ).toBe(true)
  const data = await res.json()
  return data.id
}

async function acceptProposal(cookie: string, proposalId: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/proposals/${proposalId}/accept`, {
    method: "POST",
    headers: { Cookie: cookie },
  })
  expect(
    res.ok,
    `Failed to accept proposal: ${res.status}`,
  ).toBe(true)
}

async function payProposal(cookie: string, proposalId: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/proposals/${proposalId}/pay`, {
    method: "POST",
    headers: { Cookie: cookie },
  })
  expect(
    res.ok,
    `Failed to pay proposal: ${res.status}`,
  ).toBe(true)
}

async function getProposalStatus(cookie: string, proposalId: string): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/proposals/${proposalId}`, {
    headers: { Cookie: cookie },
  })
  expect(res.ok, `Failed to get proposal: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.status
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("Bonus credits after mission payment", () => {
  test.beforeEach(async ({ page, context }) => {
    await context.clearCookies()
    await page.goto("/")
  })

  test("provider receives 5 bonus credits after mission payment", async ({
    page,
    context,
  }) => {
    test.setTimeout(120_000)

    // Step 1: Register an enterprise
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)
    const enterpriseId = await getMyUserID(enterpriseCookie)

    // Step 2: Register a provider (clears enterprise session)
    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerId = await getMyUserID(providerCookie)

    // Step 3: Verify initial credits = 10
    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    // Step 4: Enterprise starts a conversation with provider
    const conversationId = await startConversation(enterpriseCookie, providerId)
    expect(conversationId).toBeTruthy()

    // Step 5: Enterprise creates a proposal
    const proposalId = await createProposal(
      enterpriseCookie,
      conversationId,
      providerId,
    )
    expect(proposalId).toBeTruthy()

    // Step 6: Provider accepts the proposal
    await acceptProposal(providerCookie, proposalId)

    // Step 7: Enterprise pays (simulation mode in dev — no Stripe needed)
    await payProposal(enterpriseCookie, proposalId)

    // Step 8: Verify proposal is now active
    const status = await getProposalStatus(enterpriseCookie, proposalId)
    expect(status).toBe("active")

    // Step 9: Verify provider credits went from 10 to 15
    const creditsAfter = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfter).toBe(INITIAL_CREDITS + BONUS_PER_MISSION)
  })

  test("credits remain unchanged when proposal already active (idempotency)", async ({
    page,
    context,
  }) => {
    test.setTimeout(120_000)

    // Step 1: Register enterprise + provider and complete full payment flow
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerId = await getMyUserID(providerCookie)

    const conversationId = await startConversation(enterpriseCookie, providerId)
    const proposalId = await createProposal(
      enterpriseCookie,
      conversationId,
      providerId,
    )
    await acceptProposal(providerCookie, proposalId)
    await payProposal(enterpriseCookie, proposalId)

    // Verify credits = 15 after first payment
    const creditsAfterPayment = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterPayment).toBe(INITIAL_CREDITS + BONUS_PER_MISSION)

    // Step 2: Attempt to confirm payment again (idempotent endpoint)
    const confirmRes = await fetch(
      `${API_URL}/api/v1/proposals/${proposalId}/confirm-payment`,
      {
        method: "POST",
        headers: { Cookie: enterpriseCookie },
      },
    )
    // Should succeed (idempotent) or return ok
    expect(confirmRes.ok).toBe(true)

    // Step 3: Credits must NOT increase further — still 15
    const creditsAfterRetry = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterRetry).toBe(INITIAL_CREDITS + BONUS_PER_MISSION)
  })

  test("multiple missions award cumulative bonus credits", async ({
    page,
    context,
  }) => {
    test.setTimeout(180_000)

    // Register enterprise and provider
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerId = await getMyUserID(providerCookie)

    // Verify initial credits
    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    // Complete 2 separate missions
    for (let i = 0; i < 2; i++) {
      const convId = await startConversation(enterpriseCookie, providerId)
      const proposalId = await createProposal(
        enterpriseCookie,
        convId,
        providerId,
      )
      await acceptProposal(providerCookie, proposalId)
      await payProposal(enterpriseCookie, proposalId)
    }

    // Credits should be 10 + 5 + 5 = 20
    const creditsAfterTwoMissions = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterTwoMissions).toBe(INITIAL_CREDITS + 2 * BONUS_PER_MISSION)
  })
})
