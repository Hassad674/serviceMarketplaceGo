import { test, expect, type BrowserContext } from "@playwright/test"
import { registerEnterprise, registerProvider } from "./helpers/auth"

// ---------------------------------------------------------------------------
// E2E: full milestone lifecycle
//
// Drives the backend directly through its public REST API the same way the
// bonus-credits suite does — registers two users via the UI to produce real
// sessions, then walks a 2-milestone proposal through its entire lifecycle:
//
//   accepted → (fund #1) → active → (submit #1) → completion_requested
//            → (approve #1) → active (current = #2)
//            → (fund #2) → active → (submit #2) → completion_requested
//            → (approve #2) → completed
//
// Assertions are made on the proposal JSON between every step so a
// regression in the milestone state machine surfaces immediately on the
// offending transition (rather than as "status is wrong at the end").
// ---------------------------------------------------------------------------

const API_URL = "http://localhost:8083"

// Small enough to stay comfortably inside the min-amount fraud threshold
// (3000 centimes = 30 EUR per mission minimum).
const MILESTONE_1_AMOUNT = 150_000 // 1500 EUR
const MILESTONE_2_AMOUNT = 100_000 // 1000 EUR

// ---------------------------------------------------------------------------
// API helpers — all use the session cookie captured from the UI flow,
// same technique as bonus-credits.spec.ts.
// ---------------------------------------------------------------------------

async function sessionCookie(context: BrowserContext): Promise<string> {
  const cookies = await context.cookies()
  const session = cookies.find((c) => c.name === "session_id")
  return session ? `session_id=${session.value}` : ""
}

async function getMyUserID(cookie: string): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/auth/me`, {
    headers: { Cookie: cookie },
  })
  expect(res.ok, `Failed to fetch /me: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.id ?? data.data?.id
}

async function startConversation(cookie: string, recipientId: string): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/messaging/conversations`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Cookie: cookie },
    body: JSON.stringify({
      recipient_id: recipientId,
      content: "Starting a milestone project",
      type: "text",
    }),
  })
  expect(res.ok, `Failed to start conversation: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.conversation_id
}

async function createMilestoneProposal(
  cookie: string,
  conversationId: string,
  recipientId: string,
): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/proposals`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Cookie: cookie },
    body: JSON.stringify({
      conversation_id: conversationId,
      recipient_id: recipientId,
      title: "E2E Milestone Test",
      description: "Two-milestone proposal for the full-lifecycle E2E test",
      amount: MILESTONE_1_AMOUNT + MILESTONE_2_AMOUNT,
      payment_mode: "milestone",
      milestones: [
        {
          sequence: 1,
          title: "Design phase",
          description: "Wireframes and visual design",
          amount: MILESTONE_1_AMOUNT,
        },
        {
          sequence: 2,
          title: "Development phase",
          description: "Implementation and QA",
          amount: MILESTONE_2_AMOUNT,
        },
      ],
    }),
  })
  expect(res.ok, `Failed to create milestone proposal: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.id ?? data.data?.id
}

async function acceptProposal(cookie: string, proposalId: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/proposals/${proposalId}/accept`, {
    method: "POST",
    headers: { Cookie: cookie },
  })
  expect(res.ok, `Failed to accept proposal: ${res.status}`).toBe(true)
}

type ProposalSnapshot = {
  status: string
  currentMilestoneSequence: number | null
  milestones: { id: string; sequence: number; status: string }[]
}

async function snapshotProposal(cookie: string, proposalId: string): Promise<ProposalSnapshot> {
  const res = await fetch(`${API_URL}/api/v1/proposals/${proposalId}`, {
    headers: { Cookie: cookie },
  })
  expect(res.ok, `Failed to fetch proposal: ${res.status}`).toBe(true)
  const raw = await res.json()
  const p = raw.data ?? raw
  return {
    status: p.status,
    currentMilestoneSequence: p.current_milestone_sequence ?? null,
    milestones: (p.milestones ?? []).map(
      (m: { id: string; sequence: number; status: string }) => ({
        id: m.id,
        sequence: m.sequence,
        status: m.status,
      }),
    ),
  }
}

async function fundMilestone(
  cookie: string,
  proposalId: string,
  milestoneId: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/proposals/${proposalId}/milestones/${milestoneId}/fund`,
    { method: "POST", headers: { Cookie: cookie } },
  )
  expect(res.ok, `Failed to fund milestone: ${res.status}`).toBe(true)
}

async function submitMilestone(
  cookie: string,
  proposalId: string,
  milestoneId: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/proposals/${proposalId}/milestones/${milestoneId}/submit`,
    { method: "POST", headers: { Cookie: cookie } },
  )
  expect(res.ok, `Failed to submit milestone: ${res.status}`).toBe(true)
}

async function approveMilestone(
  cookie: string,
  proposalId: string,
  milestoneId: string,
): Promise<void> {
  const res = await fetch(
    `${API_URL}/api/v1/proposals/${proposalId}/milestones/${milestoneId}/approve`,
    { method: "POST", headers: { Cookie: cookie } },
  )
  expect(res.ok, `Failed to approve milestone: ${res.status}`).toBe(true)
}

function milestoneBySequence(
  snap: ProposalSnapshot,
  sequence: number,
): { id: string; sequence: number; status: string } {
  const m = snap.milestones.find((x) => x.sequence === sequence)
  if (!m) throw new Error(`Milestone sequence ${sequence} not found in snapshot`)
  return m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("Milestone lifecycle", () => {
  test.beforeEach(async ({ page, context }) => {
    await context.clearCookies()
    await page.goto("/")
  })

  test("full fund → submit → approve → fund → submit → approve flow", async ({
    page,
    context,
  }) => {
    test.setTimeout(180_000)

    // --- Step 1: register both actors and capture their cookies ---
    await registerEnterprise(page)
    const enterpriseCookie = await sessionCookie(context)

    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await sessionCookie(context)
    const providerID = await getMyUserID(providerCookie)

    // --- Step 2: enterprise starts a conversation and creates the proposal ---
    const conversationID = await startConversation(enterpriseCookie, providerID)
    const proposalID = await createMilestoneProposal(
      enterpriseCookie,
      conversationID,
      providerID,
    )

    // Initial snapshot: two pending_funding milestones, proposal still pending.
    let snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("pending")
    expect(snap.milestones).toHaveLength(2)
    expect(milestoneBySequence(snap, 1).status).toBe("pending_funding")
    expect(milestoneBySequence(snap, 2).status).toBe("pending_funding")
    expect(snap.currentMilestoneSequence).toBe(1)

    // --- Step 3: provider accepts the proposal ---
    await acceptProposal(providerCookie, proposalID)
    snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("accepted")

    // --- Step 4: enterprise funds milestone 1 (simulation mode) ---
    const m1 = milestoneBySequence(snap, 1)
    await fundMilestone(enterpriseCookie, proposalID, m1.id)

    snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("active")
    expect(milestoneBySequence(snap, 1).status).toBe("funded")
    expect(milestoneBySequence(snap, 2).status).toBe("pending_funding")
    expect(snap.currentMilestoneSequence).toBe(1)

    // --- Step 5: provider submits milestone 1 for approval ---
    await submitMilestone(providerCookie, proposalID, m1.id)
    snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("completion_requested")
    expect(milestoneBySequence(snap, 1).status).toBe("submitted")

    // --- Step 6: enterprise approves milestone 1 — releases the escrow
    //             and the CURRENT milestone cursor advances to sequence 2.
    await approveMilestone(enterpriseCookie, proposalID, m1.id)
    snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("active")
    expect(milestoneBySequence(snap, 1).status).toBe("released")
    expect(milestoneBySequence(snap, 2).status).toBe("pending_funding")
    expect(snap.currentMilestoneSequence).toBe(2)

    // --- Step 7: enterprise funds milestone 2 ---
    const m2 = milestoneBySequence(snap, 2)
    await fundMilestone(enterpriseCookie, proposalID, m2.id)
    snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("active")
    expect(milestoneBySequence(snap, 2).status).toBe("funded")

    // --- Step 8: provider submits milestone 2 ---
    await submitMilestone(providerCookie, proposalID, m2.id)
    snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("completion_requested")
    expect(milestoneBySequence(snap, 2).status).toBe("submitted")

    // --- Step 9: enterprise approves milestone 2 — this is the LAST
    //             milestone, so the macro status becomes completed.
    await approveMilestone(enterpriseCookie, proposalID, m2.id)
    snap = await snapshotProposal(enterpriseCookie, proposalID)
    expect(snap.status).toBe("completed")
    expect(milestoneBySequence(snap, 1).status).toBe("released")
    expect(milestoneBySequence(snap, 2).status).toBe("released")
  })

  test("stale milestone id returns 409", async ({ page, context }) => {
    test.setTimeout(120_000)

    // Register users and create a funded milestone-1 proposal.
    await registerEnterprise(page)
    const enterpriseCookie = await sessionCookie(context)

    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await sessionCookie(context)
    const providerID = await getMyUserID(providerCookie)

    const conversationID = await startConversation(enterpriseCookie, providerID)
    const proposalID = await createMilestoneProposal(
      enterpriseCookie,
      conversationID,
      providerID,
    )
    await acceptProposal(providerCookie, proposalID)

    const snap = await snapshotProposal(enterpriseCookie, proposalID)
    const m1 = milestoneBySequence(snap, 1)
    await fundMilestone(enterpriseCookie, proposalID, m1.id)

    // Attempt to submit milestone 2 while milestone 1 is still the
    // current active one. The validateMilestoneMatchesCurrent check in
    // the handler must reject this with 409 stale_milestone.
    const m2 = milestoneBySequence(snap, 2)
    const res = await fetch(
      `${API_URL}/api/v1/proposals/${proposalID}/milestones/${m2.id}/submit`,
      { method: "POST", headers: { Cookie: providerCookie } },
    )
    expect(res.status).toBe(409)
    const body = await res.json()
    expect(body.error).toBe("stale_milestone")
  })
})
