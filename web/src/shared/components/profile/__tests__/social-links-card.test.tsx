import { describe, expect, it, vi } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import {
  SocialLinksCard,
  type SocialLinkEntry,
} from "../social-links-card"

function renderCard(props: Parameters<typeof SocialLinksCard>[0]) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SocialLinksCard {...props} />
    </NextIntlClientProvider>,
  )
}

describe("SocialLinksCard", () => {
  const sampleLinks: SocialLinkEntry[] = [
    { platform: "github", url: "https://github.com/u" },
    { platform: "linkedin", url: "https://linkedin.com/in/u" },
  ]

  it("renders read-only display of links with no editor", () => {
    renderCard({ links: sampleLinks })
    expect(screen.getByText("github")).toBeInTheDocument()
    expect(screen.getByText("linkedin")).toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: messages.profile.editSocialLinks }),
    ).toBeNull()
  })

  it("collapses to nothing when read-only and empty", () => {
    const { container } = renderCard({ links: [] })
    expect(container.firstChild).toBeNull()
  })

  it("shows the empty placeholder when editor.canEdit is true and list is empty", () => {
    renderCard({
      links: [],
      editor: {
        canEdit: true,
        onUpsert: vi.fn(),
        onDelete: vi.fn(),
      },
    })
    expect(
      screen.getByText(messages.profile.noSocialLinks),
    ).toBeInTheDocument()
  })

  it("renders skeleton when isLoading is true", () => {
    const { container } = renderCard({ links: [], isLoading: true })
    expect(container.querySelector(".animate-shimmer")).not.toBeNull()
  })

  it("opens the editor when edit button is clicked", () => {
    renderCard({
      links: sampleLinks,
      editor: {
        canEdit: true,
        onUpsert: vi.fn(),
        onDelete: vi.fn(),
      },
    })
    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.editSocialLinks }),
    )
    expect(
      screen.getByRole("textbox", { name: messages.profile.github }),
    ).toBeInTheDocument()
  })

  it("hides the edit button when editor.canEdit is false", () => {
    renderCard({
      links: sampleLinks,
      editor: {
        canEdit: false,
        onUpsert: vi.fn(),
        onDelete: vi.fn(),
      },
    })
    expect(
      screen.queryByRole("button", { name: messages.profile.editSocialLinks }),
    ).toBeNull()
  })

  it("calls onUpsert for filled inputs and onDelete for cleared inputs on save", async () => {
    const onUpsert = vi.fn().mockResolvedValue(undefined)
    const onDelete = vi.fn().mockResolvedValue(undefined)

    renderCard({
      links: sampleLinks,
      editor: { canEdit: true, onUpsert, onDelete },
    })

    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.editSocialLinks }),
    )

    const githubInput = screen.getByRole("textbox", {
      name: messages.profile.github,
    }) as HTMLInputElement
    const linkedinInput = screen.getByRole("textbox", {
      name: messages.profile.linkedin,
    }) as HTMLInputElement
    const websiteInput = screen.getByRole("textbox", {
      name: messages.profile.website,
    }) as HTMLInputElement

    // Keep github, clear linkedin (should delete), add website (should upsert).
    fireEvent.change(githubInput, {
      target: { value: "https://github.com/u" },
    })
    fireEvent.change(linkedinInput, { target: { value: "" } })
    fireEvent.change(websiteInput, {
      target: { value: "https://example.com" },
    })

    fireEvent.click(
      screen.getByRole("button", { name: messages.common.save }),
    )

    await waitFor(() => {
      expect(onUpsert).toHaveBeenCalledWith("github", "https://github.com/u")
      expect(onUpsert).toHaveBeenCalledWith("website", "https://example.com")
      expect(onDelete).toHaveBeenCalledWith("linkedin")
    })
  })
})
