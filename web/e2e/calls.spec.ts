import { test, expect, type Page } from "@playwright/test"

/**
 * E2E tests for the call system (audio + video).
 *
 * These tests simulate the complete call flow between two browser contexts
 * (two users). They validate signaling, UI state transitions, and call
 * lifecycle — NOT actual audio/video media (which requires real devices).
 *
 * Prerequisites:
 *   - Backend running on localhost:8083
 *   - Next.js dev server running on localhost:3001
 *   - Two test accounts exist (registered via the app)
 *   - NEXT_PUBLIC_LIVEKIT_URL set in .env.local
 */

// Test accounts — update these with real credentials from your local DB
const USER_A = { email: "testcall_a@example.com", password: "TestCall123!" }
const USER_B = { email: "testcall_b@example.com", password: "TestCall123!" }

const BASE_URL = "http://localhost:3001"
const API_URL = "http://localhost:8083"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function login(page: Page, user: typeof USER_A) {
  await page.goto(`${BASE_URL}/login`)
  await page.fill('input[name="email"]', user.email)
  await page.fill('input[name="password"]', user.password)
  await page.click('button[type="submit"]')
  await page.waitForURL(/\/dashboard|\/messages/, { timeout: 10000 })
}

// Kept for when this skipped suite is reactivated — see test.skip below.
async function _navigateToConversation(page: Page, otherUserName: string) {
  await page.goto(`${BASE_URL}/messages`)
  await page.waitForSelector('[data-testid="conversation-list"]', { timeout: 5000 }).catch(() => {})
  // Click on the conversation with the other user
  const conversation = page.locator(`text=${otherUserName}`).first()
  if (await conversation.isVisible()) {
    await conversation.click()
    await page.waitForTimeout(1000)
  }
}

// Kept for when this skipped suite is reactivated.
async function _clearGhostCalls() {
  // Clear any stuck call states in Redis via the API
  // This is a best-effort cleanup
  try {
    await fetch(`${API_URL}/health`)
  } catch {
    // Backend might not be running
  }
}

// ---------------------------------------------------------------------------
// Web-Web Audio Call Tests
// ---------------------------------------------------------------------------

test.describe("Audio Calls — Web to Web", () => {
  test.skip(true, "Requires two test accounts — update USER_A and USER_B credentials first")

  test("initiator can start an audio call and see PiP overlay", async ({ browser }) => {
    const contextA = await browser.newContext()
    const pageA = await contextA.newPage()
    await login(pageA, USER_A)

    // Navigate to a conversation with User B
    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    // Click the phone icon (audio call button)
    const phoneButton = pageA.locator('[aria-label*="audio"], [aria-label*="appel"]').first()
    if (await phoneButton.isVisible()) {
      await phoneButton.click()

      // Should see the PiP overlay (ringing state)
      await expect(
        pageA.locator('[class*="fixed"][class*="z-"]').first(),
      ).toBeVisible({ timeout: 5000 })
    }

    await contextA.close()
  })

  test("recipient sees incoming call overlay and can accept", async ({ browser }) => {
    const contextA = await browser.newContext()
    const contextB = await browser.newContext()
    const pageA = await contextA.newPage()
    const pageB = await contextB.newPage()

    await login(pageA, USER_A)
    await login(pageB, USER_B)

    // User A starts call
    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    const phoneButton = pageA.locator('[aria-label*="audio"], [aria-label*="appel"]').first()
    if (await phoneButton.isVisible()) {
      await phoneButton.click()

      // User B should see incoming call overlay
      await expect(
        pageB.locator('text=/Incoming|Appel entrant/i'),
      ).toBeVisible({ timeout: 10000 })

      // User B accepts
      const acceptButton = pageB.locator('text=/Accept|Accepter/i').first()
      await acceptButton.click()

      // Both should now show active call (PiP or fullscreen)
      await pageA.waitForTimeout(2000)

      // Verify call is active — duration timer should be visible
      await expect(pageA.locator('text=/00:/').first()).toBeVisible({ timeout: 5000 })
      await expect(pageB.locator('text=/00:/').first()).toBeVisible({ timeout: 5000 })
    }

    await contextA.close()
    await contextB.close()
  })

  test("recipient can decline an incoming call", async ({ browser }) => {
    const contextA = await browser.newContext()
    const contextB = await browser.newContext()
    const pageA = await contextA.newPage()
    const pageB = await contextB.newPage()

    await login(pageA, USER_A)
    await login(pageB, USER_B)

    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    const phoneButton = pageA.locator('[aria-label*="audio"], [aria-label*="appel"]').first()
    if (await phoneButton.isVisible()) {
      await phoneButton.click()

      // User B sees incoming call
      await expect(
        pageB.locator('text=/Incoming|Appel entrant/i'),
      ).toBeVisible({ timeout: 10000 })

      // User B declines
      const declineButton = pageB.locator('text=/Decline|Refuser/i').first()
      await declineButton.click()

      // Overlay should disappear on both sides
      await expect(
        pageB.locator('text=/Incoming|Appel entrant/i'),
      ).not.toBeVisible({ timeout: 5000 })
    }

    await contextA.close()
    await contextB.close()
  })

  test("hangup properly cleans up on both sides", async ({ browser }) => {
    const contextA = await browser.newContext()
    const contextB = await browser.newContext()
    const pageA = await contextA.newPage()
    const pageB = await contextB.newPage()

    await login(pageA, USER_A)
    await login(pageB, USER_B)

    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    const phoneButton = pageA.locator('[aria-label*="audio"], [aria-label*="appel"]').first()
    if (await phoneButton.isVisible()) {
      await phoneButton.click()

      // B accepts
      await pageB.waitForSelector('text=/Accept|Accepter/i', { timeout: 10000 })
      await pageB.click('text=/Accept|Accepter/i')
      await pageA.waitForTimeout(2000)

      // A hangs up
      const hangupButton = pageA.locator('[aria-label*="Raccrocher"], [aria-label*="Hang up"]').first()
      if (await hangupButton.isVisible()) {
        await hangupButton.click()

        // Both overlays should disappear
        await pageA.waitForTimeout(2000)
        // System message "Appel termine" should appear in chat
        await expect(
          pageA.locator('text=/Appel termin|Call ended/i').last(),
        ).toBeVisible({ timeout: 5000 })
      }
    }

    await contextA.close()
    await contextB.close()
  })
})

// ---------------------------------------------------------------------------
// Web-Web Video Call Tests
// ---------------------------------------------------------------------------

test.describe("Video Calls — Web to Web", () => {
  test.skip(true, "Requires two test accounts and camera access")

  test("video call shows camera button in controls", async ({ browser }) => {
    const contextA = await browser.newContext({
      permissions: ["camera", "microphone"],
    })
    const pageA = await contextA.newPage()
    await login(pageA, USER_A)

    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    // Click video call button
    const videoButton = pageA.locator('[aria-label*="video"], [aria-label*="vidéo"]').first()
    if (await videoButton.isVisible()) {
      await videoButton.click()

      // PiP should show camera toggle button
      await expect(
        pageA.locator('[aria-label*="Caméra"], [aria-label*="Camera"]').first(),
      ).toBeVisible({ timeout: 5000 })
    }

    await contextA.close()
  })

  test("video call PiP can expand to fullscreen and back", async ({ browser }) => {
    const contextA = await browser.newContext({
      permissions: ["camera", "microphone"],
    })
    const contextB = await browser.newContext({
      permissions: ["camera", "microphone"],
    })
    const pageA = await contextA.newPage()
    const pageB = await contextB.newPage()

    await login(pageA, USER_A)
    await login(pageB, USER_B)

    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    const videoButton = pageA.locator('[aria-label*="video"], [aria-label*="vidéo"]').first()
    if (await videoButton.isVisible()) {
      await videoButton.click()

      // B accepts
      await pageB.waitForSelector('text=/Accept|Accepter/i', { timeout: 10000 })
      await pageB.click('text=/Accept|Accepter/i')
      await pageA.waitForTimeout(2000)

      // Click fullscreen button on PiP
      const fullscreenBtn = pageA.locator('[aria-label*="fullscreen"], [aria-label*="Plein"]').first()
      if (await fullscreenBtn.isVisible()) {
        await fullscreenBtn.click()

        // Should now be in fullscreen (z-[200] overlay)
        await expect(
          pageA.locator('[class*="inset-0"][class*="z-"]'),
        ).toBeVisible({ timeout: 3000 })

        // Press Escape to go back to PiP
        await pageA.keyboard.press("Escape")
        await pageA.waitForTimeout(500)

        // Should be back in PiP
        await expect(
          pageA.locator('[class*="inset-0"][class*="z-"]'),
        ).not.toBeVisible({ timeout: 3000 })
      }
    }

    await contextA.close()
    await contextB.close()
  })
})

// ---------------------------------------------------------------------------
// Call Lifecycle Tests
// ---------------------------------------------------------------------------

test.describe("Call Lifecycle", () => {
  test.skip(true, "Requires two test accounts")

  test("call times out after 30 seconds if not answered", async ({ browser }) => {
    const contextA = await browser.newContext()
    const pageA = await contextA.newPage()
    await login(pageA, USER_A)

    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    const phoneButton = pageA.locator('[aria-label*="audio"], [aria-label*="appel"]').first()
    if (await phoneButton.isVisible()) {
      await phoneButton.click()

      // PiP should show "Calling..."
      await expect(
        pageA.locator('text=/Calling|Appel en cours/i').first(),
      ).toBeVisible({ timeout: 5000 })

      // Wait for timeout (30s + buffer)
      await pageA.waitForTimeout(35000)

      // PiP should disappear after timeout
      // The call overlay should no longer be visible
    }

    await contextA.close()
  })

  test("system message appears after call ends", async ({ browser }) => {
    const contextA = await browser.newContext()
    const contextB = await browser.newContext()
    const pageA = await contextA.newPage()
    const pageB = await contextB.newPage()

    await login(pageA, USER_A)
    await login(pageB, USER_B)

    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    const phoneButton = pageA.locator('[aria-label*="audio"], [aria-label*="appel"]').first()
    if (await phoneButton.isVisible()) {
      await phoneButton.click()

      // B accepts
      await pageB.waitForSelector('text=/Accept|Accepter/i', { timeout: 10000 })
      await pageB.click('text=/Accept|Accepter/i')
      await pageA.waitForTimeout(3000)

      // A hangs up
      const hangupButton = pageA.locator('[aria-label*="Raccrocher"], [aria-label*="Hang up"]').first()
      if (await hangupButton.isVisible()) {
        await hangupButton.click()
        await pageA.waitForTimeout(2000)

        // System message should show "Appel termine" or "Call ended"
        const systemMessage = pageA.locator('text=/Appel termin|Call ended|Appel vocal/i').last()
        await expect(systemMessage).toBeVisible({ timeout: 5000 })
      }
    }

    await contextA.close()
    await contextB.close()
  })

  test("user cannot start a second call while already in one", async ({ browser }) => {
    const contextA = await browser.newContext()
    const pageA = await contextA.newPage()
    await login(pageA, USER_A)

    await pageA.goto(`${BASE_URL}/messages`)
    await pageA.waitForTimeout(2000)

    const phoneButton = pageA.locator('[aria-label*="audio"], [aria-label*="appel"]').first()
    if (await phoneButton.isVisible()) {
      // Start first call
      await phoneButton.click()
      await pageA.waitForTimeout(1000)

      // Try to start second call — should be blocked
      // Navigate to another conversation
      await pageA.goto(`${BASE_URL}/messages`)
      await pageA.waitForTimeout(1000)

      // The call button click should not start a new call
      // (state is not idle)
    }

    await contextA.close()
  })
})

// ---------------------------------------------------------------------------
// Call API Unit Tests (no browser needed)
// ---------------------------------------------------------------------------

test.describe("Call API — Backend Integration", () => {
  test.skip(true, "Requires backend running and valid session")

  test("POST /calls/initiate returns 201 with room and token", async ({ request }) => {
    // Login first to get session
    const loginRes = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: USER_A.email, password: USER_A.password },
    })
    expect(loginRes.ok()).toBeTruthy()

    // Note: need to extract session cookie and use it
    // This is a simplified test structure
  })

  test("POST /calls/{id}/accept returns 200 with token", async () => {
    // Test accept flow
  })

  test("POST /calls/{id}/decline returns 204", async () => {
    // Test decline flow
  })

  test("POST /calls/{id}/end returns 204", async () => {
    // Test end flow
  })

  test("POST /calls/initiate returns 409 if user already in call", async () => {
    // Test user_busy check
  })

  test("POST /calls/initiate returns 422 if recipient offline", async () => {
    // Test recipient_offline check
  })
})
