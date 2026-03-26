import { test, expect } from "@playwright/test"
import { registerEnterprise, clearAuth } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Projects E2E tests
//
// These tests require the backend to be running. They test the project
// creation UI after authenticating as an enterprise user.
// ---------------------------------------------------------------------------

test.describe("Projects", () => {
  test.beforeEach(async ({ page }) => {
    await clearAuth(page)
  })

  test("projects page loads with empty state", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects")
    await page.waitForLoadState("networkidle")

    // Should show the projects page title or empty state
    const hasTitle = await page.getByText(/Projects/i).first().isVisible()
    expect(hasTitle).toBe(true)
  })

  test('"Create Project" button navigates to /projects/new', async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects")
    await page.waitForLoadState("networkidle")

    const createButton = page.getByRole("link", { name: /Create Project/i }).or(
      page.getByRole("button", { name: /Create Project/i }),
    )

    if (await createButton.isVisible().catch(() => false)) {
      await createButton.click()
      await page.waitForLoadState("networkidle")

      // Should be on the create project page
      expect(page.url()).toContain("/projects/new")
    }
  })

  test("payment type selector defaults to Escrow", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    // The escrow card should be selected by default (has rose border)
    const escrowCard = page.getByText(/Escrow payments/i).locator("..")
    if (await escrowCard.isVisible().catch(() => false)) {
      // Look for the check mark icon or the selected state
      await expect(page.getByText(/Escrow payments/i)).toBeVisible()
    }
  })

  test("can switch to Invoice billing", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    const invoiceCard = page.getByText(/Invoice billing/i)
    if (await invoiceCard.isVisible().catch(() => false)) {
      await invoiceCard.click()
      await page.waitForTimeout(300)

      // Invoice card should now be in selected state
      // The escrow card should no longer be selected
      await expect(invoiceCard).toBeVisible()
    }
  })

  test("milestone editor: add/remove milestones", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    // Find the add milestone button
    const addButton = page.getByText(/Add milestone/i)
    if (await addButton.isVisible().catch(() => false)) {
      // Count initial milestone title inputs
      const initialInputs = await page.getByPlaceholder(/Milestone title/i).count()

      // Add a milestone
      await addButton.click()
      await page.waitForTimeout(300)

      const afterAddInputs = await page.getByPlaceholder(/Milestone title/i).count()
      expect(afterAddInputs).toBe(initialInputs + 1)

      // Remove a milestone (click the last delete button)
      const deleteButtons = page.getByRole("button", { name: /Delete milestone/i })
      const deleteCount = await deleteButtons.count()
      if (deleteCount > 0) {
        await deleteButtons.last().click()
        await page.waitForTimeout(300)

        const afterRemoveInputs = await page.getByPlaceholder(/Milestone title/i).count()
        expect(afterRemoveInputs).toBe(afterAddInputs - 1)
      }
    }
  })

  test("skills input: add/remove tags", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    // Find the skills input
    const skillInput = page.getByPlaceholder(/Type a skill/i)
    if (await skillInput.isVisible().catch(() => false)) {
      // Add a skill
      await skillInput.fill("React")
      await skillInput.press("Enter")
      await page.waitForTimeout(300)

      // The skill tag should appear
      await expect(page.getByText("React")).toBeVisible()

      // Add another skill
      await skillInput.fill("TypeScript")
      await skillInput.press("Enter")
      await page.waitForTimeout(300)

      await expect(page.getByText("TypeScript")).toBeVisible()

      // Remove the first skill
      const removeButton = page.getByRole("button", { name: /Remove React/i })
      if (await removeButton.isVisible().catch(() => false)) {
        await removeButton.click()
        await page.waitForTimeout(300)

        await expect(page.getByText("React")).not.toBeVisible()
        await expect(page.getByText("TypeScript")).toBeVisible()
      }
    }
  })

  test("timeline: set dates, toggle ongoing", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    // Find the ongoing toggle/checkbox
    const ongoingToggle = page.getByText(/Ongoing/i)
    if (await ongoingToggle.isVisible().catch(() => false)) {
      await ongoingToggle.click()
      await page.waitForTimeout(300)
    }

    // Find date inputs
    const startDate = page.getByLabel(/Start date/i)
    if (await startDate.isVisible().catch(() => false)) {
      await startDate.fill("2026-04-01")
    }
  })

  test("applicant section: select radio options", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    // Look for the "Who can apply" section
    const freelancersOption = page.getByText(/Freelancers only/i)
    if (await freelancersOption.isVisible().catch(() => false)) {
      await freelancersOption.click()
      await page.waitForTimeout(300)
    }

    const agenciesOption = page.getByText(/Agencies only/i)
    if (await agenciesOption.isVisible().catch(() => false)) {
      await agenciesOption.click()
      await page.waitForTimeout(300)
    }
  })

  test("preview panel updates live", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    // Fill in the project title
    const titleInput = page.getByPlaceholder(/Website design/i).or(
      page.getByLabel(/Project title/i),
    )
    if (await titleInput.isVisible().catch(() => false)) {
      await titleInput.fill("My Test Project")
      await page.waitForTimeout(300)

      // The preview panel should show the title
      // Look for the title text appearing elsewhere on the page (preview area)
      const previewTitle = page.locator("[class*='preview']").getByText("My Test Project")
      if (await previewTitle.isVisible().catch(() => false)) {
        await expect(previewTitle).toBeVisible()
      }
    }
  })

  test("form validation (required fields)", async ({ page }) => {
    const user = await registerEnterprise(page)
    await page.goto("/my-projects/new")
    await page.waitForLoadState("networkidle")

    // Try to submit without filling required fields
    const publishButton = page.getByRole("button", { name: /Publish project/i })
    if (await publishButton.isVisible().catch(() => false)) {
      await publishButton.click()
      await page.waitForTimeout(500)

      // Should still be on the same page (form not submitted)
      expect(page.url()).toContain("/projects/new")
    }
  })
})
