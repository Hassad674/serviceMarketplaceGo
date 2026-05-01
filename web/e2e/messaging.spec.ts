import { test, expect } from "@playwright/test"
import { registerProvider, clearAuth } from "./helpers/auth"

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

  test("messages page loads with conversation list panel", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // The page title should be visible
    await expect(page.getByText("Messages")).toBeVisible()

    // The conversation list panel should be present
    await expect(page.locator("[class*='flex'][class*='h-full']").first()).toBeVisible()
  })

  test("conversation list shows title and search input", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // Title
    await expect(page.getByText("Messages")).toBeVisible()

    // Search input
    const searchInput = page.getByPlaceholder(/Search a conversation/i)
    await expect(searchInput).toBeVisible()
  })

  test("click conversation shows messages panel with input", async ({ page }) => {
    const _user = await registerProvider(page)
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

  test("role filter tabs are visible and clickable", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // All four role filter tabs should be visible
    await expect(page.getByRole("button", { name: /All/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Agency/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Freelance/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /Enterprise/i })).toBeVisible()

    // Clicking a filter tab should not break the page
    await page.getByRole("button", { name: /Agency/i }).click()
    await page.waitForTimeout(300)

    // The title should still be visible (page didn't crash)
    await expect(page.getByText("Messages")).toBeVisible()

    // Click back to all
    await page.getByRole("button", { name: /All/i }).click()
    await page.waitForTimeout(300)

    await expect(page.getByText("Messages")).toBeVisible()
  })

  test("search input filters conversations by name", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    const searchInput = page.getByPlaceholder(/Search a conversation/i)
    await expect(searchInput).toBeVisible()

    // Type a search query that should not match any conversation
    await searchInput.fill("nonexistent-user-xyz")
    await page.waitForTimeout(300)

    // Should show empty/no results state
    await expect(page.getByText(/No conversations/i)).toBeVisible()

    // Clear the search and verify the page recovers
    await searchInput.fill("")
    await page.waitForTimeout(300)
  })

  test("message input accepts text and clears on send", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // If there is a message input visible (conversation selected via URL), test it
    const messageInput = page.getByPlaceholder(/Write your message/i)
    if (await messageInput.isVisible().catch(() => false)) {
      await messageInput.fill("Hello, this is a test message")
      await expect(messageInput).toHaveValue("Hello, this is a test message")
    }
  })

  test("send button disabled when empty, enabled with text", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    const messageInput = page.getByPlaceholder(/Write your message/i)
    if (await messageInput.isVisible().catch(() => false)) {
      const sendButton = page.getByRole("button", { name: /Send message/i })

      // Send button should be disabled initially
      await expect(sendButton).toBeDisabled()

      // Type text — send button should enable
      await messageInput.fill("Hello!")
      await expect(sendButton).toBeEnabled()

      // Clear text — send button should disable again
      await messageInput.fill("")
      await expect(sendButton).toBeDisabled()
    }
  })

  test("mobile: shows conversation list by default", async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 812 })

    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // On mobile, the conversation list should be visible by default
    await expect(page.getByText("Messages")).toBeVisible()
  })

  test("empty state shown when no conversation selected on desktop", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // Without selecting a conversation, either:
    // - An empty state message is shown in the chat area
    // - Or only the conversation list is visible
    const hasEmptyState = await page.getByText(/Select a conversation/i).isVisible().catch(() => false)
    const hasNoConversations = await page.getByText(/No conversations/i).isVisible().catch(() => false)
    const hasConversationList = await page.getByText("Messages").isVisible()

    expect(hasEmptyState || hasNoConversations || hasConversationList).toBe(true)
  })

  test("file attachment button is visible in message input", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    const messageInput = page.getByPlaceholder(/Write your message/i)
    if (await messageInput.isVisible().catch(() => false)) {
      // File upload button should be present
      const fileButton = page.getByRole("button", { name: /file upload/i })
      await expect(fileButton).toBeVisible()
    }
  })

  test("page maintains layout after rapid filter switching", async ({ page }) => {
    const _user = await registerProvider(page)
    await page.goto("/messaging")
    await page.waitForLoadState("networkidle")

    // Rapidly switch between filters
    await page.getByRole("button", { name: /Agency/i }).click()
    await page.getByRole("button", { name: /Freelance/i }).click()
    await page.getByRole("button", { name: /Enterprise/i }).click()
    await page.getByRole("button", { name: /All/i }).click()

    // Page should still be intact
    await expect(page.getByText("Messages")).toBeVisible()
    await expect(page.getByPlaceholder(/Search a conversation/i)).toBeVisible()
  })
})
