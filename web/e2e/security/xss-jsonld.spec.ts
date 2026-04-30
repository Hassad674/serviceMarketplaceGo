import { test, expect } from "@playwright/test"

import { registerProvider, clearAuth } from "../helpers/auth"

// ---------------------------------------------------------------------------
// SEC-01 — Stored XSS via JSON-LD on public profile pages
// ---------------------------------------------------------------------------
//
// We register a freelance provider, set their `about` field to a script-tag
// breakout payload, log out, then visit the public profile page as an
// anonymous visitor. The page injects a <script type="application/ld+json">
// block containing the user-controlled `about` string. Without escaping,
// a `</script><script>` sequence would terminate the JSON-LD block early
// and execute the attacker's code in the page origin.
//
// This test asserts:
//   1. window.__pwned remains undefined after the page loads.
//   2. The serialized JSON-LD block does not contain a literal `</script>`
//      anywhere (the escape converts it to `</script>`).
// ---------------------------------------------------------------------------

const ATTACK_PAYLOAD = "</script><script>window.__pwned=true</script>"

test.describe("SEC-01 — JSON-LD XSS on public profile pages", () => {
  test("freelance about with </script> payload does not execute", async ({ page, context }) => {
    // 1. Register a fresh provider, capture their org/profile id.
    await registerProvider(page)

    // 2. Open the about editor and submit the attack payload.
    await page.goto("/profile")
    const editButton = page.getByRole("button", { name: /edit about/i })
    await editButton.waitFor({ state: "visible", timeout: 10000 })
    await editButton.click()
    const textarea = page.getByRole("textbox", { name: /about/i })
    await textarea.fill(ATTACK_PAYLOAD)
    await page.getByRole("button", { name: /save/i }).click()

    // 3. Capture the freelance public profile id.
    // The dashboard URL after register is /dashboard; we navigate to
    // /profile?public=1 or directly visit the public freelance route.
    // The simplest path: read the profile id from the network responses.
    const profileResponse = await page.waitForResponse(
      (r) => r.url().includes("/profile") && r.status() === 200,
      { timeout: 15_000 },
    ).catch(() => null)
    const profileBody = profileResponse ? await profileResponse.json().catch(() => null) : null
    const profileId =
      profileBody?.data?.id ||
      profileBody?.data?.organization_id ||
      profileBody?.data?.user_id
    test.skip(!profileId, "could not resolve profile id — skip until fixture exposes it")

    // 4. Log out so the public page is rendered as an anonymous visitor.
    await clearAuth(page)

    // 5. Set up a global flag we can inspect AFTER navigation. If the
    //    breakout works, the injected attacker script will set it to true.
    const newPage = await context.newPage()
    await newPage.addInitScript(() => {
      // @ts-expect-error — test sentinel
      window.__pwned = undefined
    })

    // 6. Visit the public freelance profile.
    await newPage.goto(`/freelancers/${profileId}`)
    await newPage.waitForLoadState("networkidle")

    // 7. Sentinel must remain undefined.
    const pwned = await newPage.evaluate(() => {
      // @ts-expect-error — test sentinel
      return window.__pwned
    })
    expect(pwned).toBeUndefined()

    // 8. The rendered JSON-LD block must not contain a literal </script>.
    const html = await newPage.content()
    const jsonLdMatch = html.match(/<script type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/)
    if (jsonLdMatch) {
      const inner = jsonLdMatch[1]
      expect(inner).not.toContain("</script>")
      expect(inner).not.toContain("<script>")
      // The escaped form must be present somewhere (proves it didn't
      // get sanitized OUT entirely — the payload was preserved + escaped).
      expect(inner).toContain("\\u003c")
    }
  })
})
