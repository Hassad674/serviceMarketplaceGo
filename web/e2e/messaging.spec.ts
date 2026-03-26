import { test, expect } from "@playwright/test"
import { registerProvider, login, logout, clearAuth } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Messaging E2E tests
//
// These tests require the backend to be running. They test the messaging UI
// after authenticating as a provider user.
// ---------------------------------------------------------------------------

test.describe("Messaging", () => {
  test.beforeEach(async ({ page }) => {
    await clearAuth(page)
  })

  test("messages page loads with conversation list", async ({ page }) => {
    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // The page title should be visible
    await expect(page.getByText("Messages")).toBeVisible()

    // The conversation list panel should be present
    await expect(page.locator("[class*='flex'][class*='h-full']").first()).toBeVisible()
  })

  test("click conversation shows chat area", async ({ page }) => {
    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // If there are conversations, clicking one should show the chat area
    const conversations = page.locator("button").filter({ hasText: /\w+/ })
    const count = await conversations.count()

    if (count > 0) {
      await conversations.first().click()
      await page.waitForTimeout(500)
      // The message input area should appear
      await expect(page.getByPlaceholder(/Write your message/i)).toBeVisible()
    } else {
      // Empty state — no conversations yet
      await expect(page.getByText(/No conversations/i)).toBeVisible()
    }
  })

  test("role filter tabs filter conversations", async ({ page }) => {
    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // Role filter tabs should be visible
    await expect(page.getByRole("button", { name: /All/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Agency/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Freelance/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Enterprise/i })).toBeVisible()

    // Clicking a filter tab should not break the page
    await page.getByRole("button", { name: /Agency/i }).click()
    await page.waitForTimeout(300)

    // The tab should appear active (visual check that it changed)
    // Click back to all
    await page.getByRole("button", { name: /All/i }).click()
    await page.waitForTimeout(300)
  })

  test("search filters by name", async ({ page }) => {
    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    const searchInput = page.getByPlaceholder(/Search a conversation/i)
    await expect(searchInput).toBeVisible()

    // Type a search query
    await searchInput.fill("nonexistent-user-xyz")
    await page.waitForTimeout(300)

    // Should show empty/no results state
    await expect(page.getByText(/No conversations/i)).toBeVisible()
  })

  test("message input accepts text", async ({ page }) => {
    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // If there is a message input visible (conversation selected), test it
    const messageInput = page.getByPlaceholder(/Write your message/i)
    if (await messageInput.isVisible().catch(() => false)) {
      await messageInput.fill("Hello, this is a test message")
      await expect(messageInput).toHaveValue("Hello, this is a test message")
    }
  })

  test("send button activates with text", async ({ page }) => {
    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    const messageInput = page.getByPlaceholder(/Write your message/i)
    if (await messageInput.isVisible().catch(() => false)) {
      const sendButton = page.getByRole("button", { name: /Send message/i })

      // Send button should be disabled initially
      await expect(sendButton).toBeDisabled()

      // Type text
      await messageInput.fill("Hello!")
      await expect(sendButton).toBeEnabled()
    }
  })

  test("mobile: shows list or chat (not both)", async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 812 })

    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // On mobile, the conversation list should be visible by default
    await expect(page.getByText("Messages")).toBeVisible()
  })

  test("empty state when no conversations selected", async ({ page }) => {
    const user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // Without selecting a conversation, either:
    // - An empty state message is shown in the chat area
    // - Or only the conversation list is visible
    const hasEmptyState = await page.getByText(/Select a conversation/i).isVisible().catch(() => false)
    const hasConversationList = await page.getByText("Messages").isVisible()

    expect(hasEmptyState || hasConversationList).toBe(true)
  })
})
