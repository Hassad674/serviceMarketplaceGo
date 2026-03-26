import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ProfileAbout } from "../profile-about"

function renderProfileAbout(
  props: Partial<Parameters<typeof ProfileAbout>[0]> = {},
) {
  const defaultProps = {
    content: "",
    onSave: vi.fn().mockResolvedValue(undefined),
    saving: false,
    ...props,
  }

  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ProfileAbout {...defaultProps} />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ProfileAbout", () => {
  it("renders empty state when no content", () => {
    renderProfileAbout({ content: "" })

    expect(screen.getByText(messages.profile.clickToEdit)).toBeInTheDocument()
  })

  it("renders content when provided", () => {
    renderProfileAbout({
      content: "I am a senior developer with 10 years experience.",
    })

    expect(
      screen.getByText("I am a senior developer with 10 years experience."),
    ).toBeInTheDocument()
  })

  it("shows edit button", () => {
    renderProfileAbout({ content: "Some content" })

    const editButton = screen.getByRole("button", {
      name: new RegExp(`${messages.common.edit}`, "i"),
    })
    expect(editButton).toBeInTheDocument()
  })

  it("switches to edit mode on click", async () => {
    const user = userEvent.setup()
    renderProfileAbout({ content: "Existing content" })

    const editButton = screen.getByRole("button", {
      name: new RegExp(`${messages.common.edit}`, "i"),
    })
    await user.click(editButton)

    // Textarea should appear with the existing content
    const textarea = screen.getByRole("textbox")
    expect(textarea).toBeInTheDocument()
    expect(textarea).toHaveValue("Existing content")
  })

  it("shows character count in edit mode", async () => {
    const user = userEvent.setup()
    renderProfileAbout({ content: "Hello" })

    const editButton = screen.getByRole("button", {
      name: new RegExp(`${messages.common.edit}`, "i"),
    })
    await user.click(editButton)

    // "5 / 1000 characters"
    expect(screen.getByText(/5 \/ 1000/)).toBeInTheDocument()
    expect(
      screen.getByText(new RegExp(messages.profile.characters)),
    ).toBeInTheDocument()
  })

  it("shows save and cancel buttons in edit mode", async () => {
    const user = userEvent.setup()
    renderProfileAbout({ content: "" })

    const editButton = screen.getByRole("button", {
      name: new RegExp(`${messages.common.edit}`, "i"),
    })
    await user.click(editButton)

    expect(
      screen.getByRole("button", { name: messages.common.save }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: messages.common.cancel }),
    ).toBeInTheDocument()
  })

  it("cancels editing and restores original content", async () => {
    const user = userEvent.setup()
    renderProfileAbout({ content: "Original text" })

    // Enter edit mode
    const editButton = screen.getByRole("button", {
      name: new RegExp(`${messages.common.edit}`, "i"),
    })
    await user.click(editButton)

    // Modify text
    const textarea = screen.getByRole("textbox")
    await user.clear(textarea)
    await user.type(textarea, "Modified text")

    // Cancel
    const cancelButton = screen.getByRole("button", {
      name: messages.common.cancel,
    })
    await user.click(cancelButton)

    // Should show original content, not edit mode
    expect(screen.getByText("Original text")).toBeInTheDocument()
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument()
  })

  it("calls onSave with trimmed text when save is clicked", async () => {
    const mockSave = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderProfileAbout({ content: "", onSave: mockSave })

    // Enter edit mode
    const editButton = screen.getByRole("button", {
      name: new RegExp(`${messages.common.edit}`, "i"),
    })
    await user.click(editButton)

    // Type text
    const textarea = screen.getByRole("textbox")
    await user.type(textarea, "  New content  ")

    // Save
    const saveButton = screen.getByRole("button", {
      name: messages.common.save,
    })
    await user.click(saveButton)

    await waitFor(() => {
      expect(mockSave).toHaveBeenCalledWith("New content")
    })
  })

  it("displays section heading", () => {
    renderProfileAbout({ content: "" })

    expect(
      screen.getByRole("heading", { name: messages.profile.about }),
    ).toBeInTheDocument()
  })

  it("accepts custom label", () => {
    renderProfileAbout({ content: "", label: "Custom Section" })

    expect(
      screen.getByRole("heading", { name: "Custom Section" }),
    ).toBeInTheDocument()
  })

  it("updates character count as user types", async () => {
    const user = userEvent.setup()
    renderProfileAbout({ content: "" })

    const editButton = screen.getByRole("button", {
      name: new RegExp(`${messages.common.edit}`, "i"),
    })
    await user.click(editButton)

    const textarea = screen.getByRole("textbox")
    await user.type(textarea, "Hello")

    expect(screen.getByText(/5 \/ 1000/)).toBeInTheDocument()
  })
})
