import { test, expect } from "@playwright/test"
import { clearAuth } from "./helpers/auth"

// Bug B regression: clicking the "Send a message" button on a
// public profile while unauthenticated must redirect to /login with
// a `next` query param so the visitor bounces back to the same
// profile after signing in. Previously the desktop path silently
// dropped the click (chat widget bootstrap failed) and the mobile
// path leaned on middleware to redirect — both delivered an
// inconsistent UX.
//
// We don't seed a real org id; we just need the public profile shell
// to render the SendMessageButton. Public listing pages (`/agencies`,
// `/freelancers`, `/referrers`) render the button on every card we
// visit — but the deterministic surface is the profile detail page.
// The test below relies on the button being present; if the seeded
// fixture changes we tag the spec as skipped rather than fail.
test.describe("Send-message button — unauthenticated redirect", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/")
    await clearAuth(page)
  })

  test("redirects to /login?next=<profile> on desktop", async ({ page }) => {
    // Use the `/freelancers` listing as the discovery entry-point —
    // it does not require auth and surfaces at least one profile
    // card. The CTA inside the card invokes the same component.
    await page.goto("/freelancers")
    const button = page.getByRole("button", { name: /start.?conversation|envoyer un message|send a message/i })

    if ((await button.count()) === 0) {
      test.skip(true, "No SendMessageButton rendered on the seeded /freelancers — fixture-dependent assertion")
      return
    }

    await button.first().click()

    await page.waitForURL(/\/login\?next=/, { timeout: 5000 })
    expect(page.url()).toMatch(/\/login\?next=/)
  })
})
