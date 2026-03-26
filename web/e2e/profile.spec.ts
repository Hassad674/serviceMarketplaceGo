import { test, expect, type Page } from "@playwright/test"
import path from "path"
import {
  registerProvider,
  registerAgency,
} from "./helpers/auth"

// ---------------------------------------------------------------------------
// Profile page — view (provider)
// ---------------------------------------------------------------------------

test.describe("Provider profile view", () => {
  test("profile page shows user name", async ({ page }) => {
    const { displayName } = await registerProvider(page)

    await page.goto("/profile")
    await expect(page.locator("h1")).toContainText(displayName, { timeout: 10000 })
  })

  test("profile page shows photo placeholder when no photo uploaded", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/profile")

    // The photo area shows a Camera icon button when no photo is set
    const photoButton = page.getByRole("button", { name: /edit your photo/i })
    await expect(photoButton).toBeVisible({ timeout: 10000 })
  })

  test("profile page shows video empty state when no video uploaded", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/profile")

    // Video section shows "No presentation video" empty state
    await expect(page.getByText("No presentation video")).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Add a video to present your activity")).toBeVisible()
  })

  test("profile page shows about empty state when no about text", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/profile")

    // About section shows "Click the edit button to add a description"
    await expect(
      page.getByText("Click the edit button to add a description"),
    ).toBeVisible({ timeout: 10000 })
  })

  test("profile page shows project history section", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/profile")

    // Project History section header is visible
    await expect(page.getByText("Project History")).toBeVisible({ timeout: 10000 })
    // Empty state for new user
    await expect(page.getByText("No completed projects")).toBeVisible()
  })

  test("profile page shows no-reviews placeholder", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/profile")

    await expect(page.getByText("No reviews")).toBeVisible({ timeout: 10000 })
  })

  test("profile page shows 'Add a professional title' when no title set", async ({ page }) => {
    await registerProvider(page)

    await page.goto("/profile")

    await expect(
      page.getByRole("button", { name: /edit professional title/i }),
    ).toBeVisible({ timeout: 10000 })
    await expect(page.getByText("Add a professional title")).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Profile page — view (agency)
// ---------------------------------------------------------------------------

test.describe("Agency profile view", () => {
  test("profile page shows agency name and logo label", async ({ page }) => {
    const { displayName } = await registerAgency(page)

    await page.goto("/profile")
    await expect(page.locator("h1")).toContainText(displayName, { timeout: 10000 })
    // Agency uses "Logo" instead of "Photo"
    await expect(page.getByText("Logo")).toBeVisible()
  })

  test("agency about section uses agency-specific label", async ({ page }) => {
    await registerAgency(page)

    await page.goto("/profile")

    // Agency about section should say "About the agency"
    await expect(page.getByText("About the agency")).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Profile photo upload
// ---------------------------------------------------------------------------

test.describe("Profile photo upload", () => {
  test("click photo area opens upload modal", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    // Click the photo button to open upload modal
    const photoButton = page.getByRole("button", { name: /edit your photo/i })
    await expect(photoButton).toBeVisible({ timeout: 10000 })
    await photoButton.click()

    // Modal should appear with dialog role
    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })
    await expect(modal).toContainText("Add a photo")
  })

  test("upload modal shows drag and drop zone", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    // Open modal
    await page.getByRole("button", { name: /edit your photo/i }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })

    // Should show drag & drop instructions
    await expect(modal.getByText("Drag your file here")).toBeVisible()
    await expect(modal.getByText("or click to browse")).toBeVisible()
  })

  test("upload modal can be closed with X button", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    // Open modal
    await page.getByRole("button", { name: /edit your photo/i }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })

    // Close it
    await modal.getByRole("button", { name: /close/i }).click()
    await expect(modal).not.toBeVisible()
  })

  test("upload modal can be closed with Escape key", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: /edit your photo/i }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })

    await page.keyboard.press("Escape")
    await expect(modal).not.toBeVisible()
  })

  test("upload modal shows cancel and upload buttons", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: /edit your photo/i }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })

    // Cancel button always visible
    await expect(modal.getByRole("button", { name: /cancel/i })).toBeVisible()

    // Upload button should be disabled when no file is selected
    const uploadButton = modal.getByRole("button", { name: /upload/i })
    await expect(uploadButton).toBeVisible()
    await expect(uploadButton).toBeDisabled()
  })

  test("upload modal shows file size limit info", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: /edit your photo/i }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })

    // Should mention max file size (5 MB for photos)
    await expect(modal.getByText(/5.*MB/i)).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Profile video section
// ---------------------------------------------------------------------------

test.describe("Profile video section", () => {
  test("shows 'Add a video' button when no video", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    // The empty state shows an "Add a video" button
    await expect(
      page.getByRole("button", { name: "Add a video" }),
    ).toBeVisible({ timeout: 10000 })
  })

  test("clicking 'Add a video' opens upload modal", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: "Add a video" }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })
    await expect(modal).toContainText("Add a video")
  })

  test("video upload modal shows video-specific size limit", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: "Add a video" }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })

    // Video max size is 50 MB
    await expect(modal.getByText(/50.*MB/i)).toBeVisible()
  })

  test("video upload modal mentions accepted formats", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: "Add a video" }).click()

    const modal = page.getByRole("dialog")
    await expect(modal).toBeVisible({ timeout: 5000 })

    await expect(modal.getByText(/MP4.*WebM/i)).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Profile about section — editing
// ---------------------------------------------------------------------------

test.describe("Profile about editing", () => {
  test("click edit icon opens textarea", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    // Click the edit button for the about section
    const editButton = page.getByRole("button", { name: /edit about/i })
    await expect(editButton).toBeVisible({ timeout: 10000 })
    await editButton.click()

    // Textarea should appear
    const textarea = page.getByRole("textbox", { name: /about/i })
    await expect(textarea).toBeVisible()
  })

  test("about textarea shows character count", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: /edit about/i }).click()

    // Character counter is visible
    await expect(page.getByText(/\/ 1000 characters/i)).toBeVisible()
  })

  test("about textarea character count updates as user types", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: /edit about/i }).click()

    const textarea = page.getByRole("textbox", { name: /about/i })
    await textarea.fill("Hello world")

    // "11 / 1000 characters" should be shown
    await expect(page.getByText("11 / 1000 characters")).toBeVisible()
  })

  test("about edit shows save and cancel buttons", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: /edit about/i }).click()

    await expect(page.getByRole("button", { name: /save/i })).toBeVisible()
    await expect(page.getByRole("button", { name: /cancel/i })).toBeVisible()
  })

  test("cancel about edit reverts changes", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    await page.getByRole("button", { name: /edit about/i }).click()

    const textarea = page.getByRole("textbox", { name: /about/i })
    await textarea.fill("Some text that should be reverted")

    // Click cancel
    await page.getByRole("button", { name: /cancel/i }).click()

    // Textarea should disappear (back to view mode)
    await expect(textarea).not.toBeVisible()

    // Original empty state should be back
    await expect(
      page.getByText("Click the edit button to add a description"),
    ).toBeVisible()
  })

  test("save about text persists after page reload", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    const aboutText = `Test about text ${Date.now()}`

    // Edit about
    await page.getByRole("button", { name: /edit about/i }).click()
    const textarea = page.getByRole("textbox", { name: /about/i })
    await textarea.fill(aboutText)

    // Save
    await page.getByRole("button", { name: /save/i }).click()

    // Wait for the textarea to close (save complete)
    await expect(textarea).not.toBeVisible({ timeout: 10000 })

    // The text should be visible in view mode
    await expect(page.getByText(aboutText)).toBeVisible()

    // Reload and verify persistence
    await page.reload()
    await expect(page.getByText(aboutText)).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Profile title editing
// ---------------------------------------------------------------------------

test.describe("Profile title editing", () => {
  test("click title opens inline edit input", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    // Click the title area to enter edit mode
    const titleButton = page.getByRole("button", { name: /edit professional title/i })
    await expect(titleButton).toBeVisible({ timeout: 10000 })
    await titleButton.click()

    // An input field should appear
    const titleInput = page.getByRole("textbox", { name: /professional title/i })
    await expect(titleInput).toBeVisible()
  })

  test("pressing Enter saves the title", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    const newTitle = `Senior Developer ${Date.now()}`

    // Open title editor
    await page.getByRole("button", { name: /edit professional title/i }).click()

    const titleInput = page.getByRole("textbox", { name: /professional title/i })
    await titleInput.fill(newTitle)
    await titleInput.press("Enter")

    // Input should close, title should display
    await expect(titleInput).not.toBeVisible({ timeout: 5000 })
    await expect(page.getByText(newTitle)).toBeVisible()
  })

  test("pressing Escape cancels title edit", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    // Open title editor
    await page.getByRole("button", { name: /edit professional title/i }).click()

    const titleInput = page.getByRole("textbox", { name: /professional title/i })
    await titleInput.fill("This should not be saved")
    await titleInput.press("Escape")

    // Input should close, original empty title should be back
    await expect(titleInput).not.toBeVisible({ timeout: 5000 })
    await expect(page.getByText("Add a professional title")).toBeVisible()
  })

  test("saved title persists after page reload", async ({ page }) => {
    await registerProvider(page)
    await page.goto("/profile")

    const newTitle = `Full-Stack Engineer ${Date.now()}`

    // Open title editor, type, and save
    await page.getByRole("button", { name: /edit professional title/i }).click()
    const titleInput = page.getByRole("textbox", { name: /professional title/i })
    await titleInput.fill(newTitle)
    await titleInput.press("Enter")

    // Wait for save
    await expect(titleInput).not.toBeVisible({ timeout: 5000 })
    await expect(page.getByText(newTitle)).toBeVisible()

    // Reload
    await page.reload()
    await expect(page.getByText(newTitle)).toBeVisible({ timeout: 10000 })
  })
})

// ---------------------------------------------------------------------------
// Profile navigation
// ---------------------------------------------------------------------------

test.describe("Profile navigation", () => {
  test("sidebar 'My Profile' link navigates to /profile", async ({ page }) => {
    await registerProvider(page)

    const sidebar = page.locator("aside")
    await sidebar.getByText("My Profile").click()

    await expect(page).toHaveURL(/\/profile/, { timeout: 10000 })
  })

  test("header dropdown 'My Profile' link navigates to /profile", async ({ page }) => {
    await registerProvider(page)

    // Open user dropdown in header
    const header = page.locator("header")
    const dropdownTrigger = header.locator("button").filter({ has: page.locator(".rounded-full") })
    await dropdownTrigger.click()

    // Click "My Profile" in dropdown
    await page.getByRole("link", { name: /My Profile/i }).click()
    await expect(page).toHaveURL(/\/profile/, { timeout: 10000 })
  })
})
