import { test, expect } from "@playwright/test"

// ---------------------------------------------------------------------------
// D4 (GDPR Phase C) — /legal/* routes smoke + sitemap regression.
//
// Asserts that the 7 D4 legal surfaces respond 200, render the expected
// H1, and that each route is wired into sitemap.xml so search-engine
// crawlers (and Vercel ISR revalidation) can discover them.
//
// We deliberately keep the assertions visual-light: legal copy varies
// over time; the goal is to lock the route shape, not the wording.
// ---------------------------------------------------------------------------

const ROUTES: ReadonlyArray<{ path: string; expectedHeadingFragment: RegExp }> =
  [
    {
      path: "/fr/legal",
      // The legal index now serves both the mentions block AND the
      // sommaire — the H1 reuses the existing mentions title key.
      expectedHeadingFragment: /Mentions/i,
    },
    {
      path: "/fr/legal/registre",
      expectedHeadingFragment: /Registre/i,
    },
    {
      path: "/fr/legal/aipd",
      expectedHeadingFragment: /Analyse d'impact/i,
    },
    {
      path: "/fr/legal/dpa-template",
      expectedHeadingFragment: /Modèle/i,
    },
    {
      path: "/fr/legal/politique-confidentialite",
      expectedHeadingFragment: /Politique de confidentialité/i,
    },
    {
      path: "/fr/legal/cgu",
      expectedHeadingFragment: /Conditions Générales d'Utilisation/i,
    },
    {
      path: "/fr/legal/cgv",
      expectedHeadingFragment: /Conditions Générales de Vente/i,
    },
  ]

// Serial execution avoids overwhelming the dev server's per-route
// just-in-time compile when this spec is run against `next dev`.
test.describe.configure({ mode: "serial" })

test.describe("/legal/* routes (D4)", () => {
  for (const { path, expectedHeadingFragment } of ROUTES) {
    test(`${path} responds 200 and renders the H1`, async ({ page }) => {
      const response = await page.goto(path, { waitUntil: "domcontentloaded" })
      expect(response, `no response from ${path}`).not.toBeNull()
      expect(response!.status(), `status for ${path}`).toBe(200)

      // The LegalShell/LegalDocument both render a unique H1. Allow
      // a generous 15 s timeout to absorb dev-mode just-in-time compiles.
      const heading = page.getByRole("heading", { level: 1 })
      await expect(heading).toBeVisible({ timeout: 15_000 })
      await expect(heading).toHaveText(expectedHeadingFragment)
    })
  }

  test("sitemap.xml includes the 7 legal routes", async ({ request }) => {
    const response = await request.get("/sitemap.xml")
    expect(response.status()).toBe(200)
    const body = await response.text()
    // /legal index + 6 sub-pages. Match on the path so we tolerate
    // either /fr/<path> or /<path> depending on the locale config.
    const expectedFragments = [
      "/legal",
      "/legal/registre",
      "/legal/aipd",
      "/legal/dpa-template",
      "/legal/politique-confidentialite",
      "/legal/cgu",
      "/legal/cgv",
    ]
    for (const fragment of expectedFragments) {
      expect(body, `sitemap should include ${fragment}`).toContain(fragment)
    }
  })

  test("the legal footer surfaces a /legal/registre link on a public page", async ({
    page,
  }) => {
    // Pick a public route that mounts the (public) layout — that
    // layout renders the sitewide LegalFooter. Routes under (auth)
    // intentionally do NOT show the footer, so we use /fr/legal
    // itself (which is in the (public) group).
    const response = await page.goto("/fr/legal", {
      waitUntil: "domcontentloaded",
    })
    expect(response!.status()).toBe(200)
    const footerLink = page.locator(
      'footer[role="contentinfo"] a[href$="/legal/registre"]',
    )
    await expect(footerLink).toBeAttached({ timeout: 15_000 })
  })
})
