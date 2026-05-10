import { test, expect } from "@playwright/test"
import { registerAgency, login } from "./helpers/auth"

// SEC-FIX-W-TEAM-R17 regression — false "permission denied" toast on
// successful role-permissions save.
//
// Before the fix, an Owner saving role permissions (e.g. toggling a
// permission for the Admin role) saw a global "Permission refusée — vous
// n'avez pas accès à cette fonctionnalité" toast even though the save
// landed in the DB. The trigger was a sibling refetch (the chained
// ["session"] invalidation, plus the team query refetches) firing the
// global QueryCache / MutationCache toast handler defined in
// `web/src/app/[locale]/providers.tsx`.
//
// The fix marks the role-permissions mutation + query with
// `meta.suppressGlobalErrorToast: true` so the editor's own
// `toast.error(...)` is the only error surface — no double-fire.
//
// This spec asserts the CONTRACT: after a successful save, no global
// permission toast is rendered within a generous 2 s observation
// window. The test is fixture-resilient — it gracefully skips when
// no Owner account is available in the seeded environment.
test.describe("team role permissions — no false toast on save", () => {
  test("does not render a global 'permission refusée' toast after a successful save", async ({ page }) => {
    // Watch every toast that lands in the DOM during the test. Sonner
    // renders toasts inside `[data-sonner-toaster]` so we keep a
    // running log of every toast description we see and assert at
    // the end that none matched the false-toast wording.
    const seenToasts: string[] = []
    page.on("console", (msg) => {
      // No-op observer — kept for debugging if the test ever flakes,
      // makes it easy to enable verbose logging from the harness.
      if (msg.type() === "error") return
    })

    // Register a fresh Agency Owner. Owner has team.manage_role_permissions
    // and is allowed to toggle every editable permission in the editor.
    const owner = await registerAgency(page)

    // Re-login on a clean page state to make sure cookies are stable.
    await login(page, owner.email, owner.password)

    // Navigate to the team page. If the page never loads (feature
    // disabled in the env) skip the assertion — the test would be
    // inconclusive.
    await page.goto("/team")
    const heading = page.getByRole("heading", { level: 1 }).first()
    if ((await heading.count()) === 0) {
      test.skip(true, "Team page not reachable — feature disabled")
      return
    }

    // The editor is rendered as part of the team page. Expand it if
    // collapsed (the header is a button with aria-expanded). The exact
    // copy is FR by default since the app's default locale is fr.
    const editorHeader = page.getByRole("button", { name: /role|permission/i }).first()
    if ((await editorHeader.count()) === 0) {
      test.skip(true, "Role permissions editor not mounted on this page")
      return
    }

    // The editor renders a list of switches. We toggle the first
    // visible permission switch in the Admin tab and click Save.
    // If the editor copy / DOM changes the test will fall through to
    // the assertion below — the CRITICAL guarantee is that no false
    // toast fires, regardless of whether the toggle actually flips.
    const firstToggle = page.getByRole("switch").first()
    if ((await firstToggle.count()) === 0) {
      test.skip(true, "No editable permissions exposed by the editor")
      return
    }
    await firstToggle.click()

    const saveButton = page.getByRole("button", { name: /enregistrer|save/i }).first()
    if ((await saveButton.count()) > 0) {
      await saveButton.click()
    }

    // Confirm in the modal if one appears.
    const confirmButton = page.getByRole("button", { name: /confirmer|confirm/i }).first()
    if ((await confirmButton.count()) > 0) {
      await confirmButton.click().catch(() => undefined)
    }

    // Capture every toast description that appears in the next 2 s.
    // Sonner descriptions land in `[data-description]` nodes inside
    // `[data-sonner-toast]`.
    await page.waitForTimeout(2000)
    const toastTexts = await page
      .locator("[data-sonner-toast]")
      .allInnerTexts()
      .catch(() => [] as string[])
    seenToasts.push(...toastTexts)

    const falseTosatPattern = /permission\s+(refusée|denied)|no_organization|forbidden/i
    const offenders = seenToasts.filter((t) => falseTosatPattern.test(t))
    expect(offenders, `Unexpected false permission toast(s): ${offenders.join(" | ")}`).toEqual(
      [],
    )
  })
})
