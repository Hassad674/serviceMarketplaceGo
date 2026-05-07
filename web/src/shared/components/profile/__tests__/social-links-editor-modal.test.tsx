import { describe, expect, it, vi } from "vitest"
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { SocialLinksEditorModal } from "../social-links-editor-modal"
import type { SocialLinkEntry } from "../social-links-card"

function renderModal(
  overrides?: Partial<Parameters<typeof SocialLinksEditorModal>[0]>,
) {
  const onClose = vi.fn()
  const onUpsert = vi.fn().mockResolvedValue(undefined)
  const onDelete = vi.fn().mockResolvedValue(undefined)
  const links: SocialLinkEntry[] = overrides?.links ?? []
  const utils = render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SocialLinksEditorModal
        open
        onClose={onClose}
        links={links}
        onUpsert={onUpsert}
        onDelete={onDelete}
        {...overrides}
      />
    </NextIntlClientProvider>,
  )
  return { ...utils, onClose, onUpsert, onDelete }
}

function getInput(label: string) {
  return screen.getByLabelText(label) as HTMLInputElement
}

function changeInput(label: string, value: string) {
  fireEvent.change(getInput(label), { target: { value } })
}

describe("SocialLinksEditorModal", () => {
  it("renders the modal title and all six platform inputs", () => {
    renderModal()
    expect(
      screen.getByText(messages.profile.socialLinksModalTitle),
    ).toBeInTheDocument()
    expect(getInput(messages.profile.linkedin)).toBeInTheDocument()
    expect(getInput(messages.profile.instagram)).toBeInTheDocument()
    expect(getInput(messages.profile.youtube)).toBeInTheDocument()
    expect(getInput(messages.profile.twitter)).toBeInTheDocument()
    expect(getInput(messages.profile.github)).toBeInTheDocument()
    expect(getInput(messages.profile.website)).toBeInTheDocument()
  })

  it("has the save button disabled when nothing has changed", () => {
    renderModal()
    const save = screen.getByRole("button", {
      name: messages.common.save,
    }) as HTMLButtonElement
    expect(save.disabled).toBe(true)
  })

  it("rejects a non-linkedin URL in the linkedin field with the regex error", async () => {
    renderModal()
    changeInput(messages.profile.linkedin, "https://google.com/foo")
    await waitFor(() =>
      expect(
        screen.getByText(messages.profile.socialLinkErrorLinkedin),
      ).toBeInTheDocument(),
    )
    const save = screen.getByRole("button", {
      name: messages.common.save,
    }) as HTMLButtonElement
    expect(save.disabled).toBe(true)
  })

  it("clears the regex error when a valid linkedin URL is entered", async () => {
    renderModal()
    changeInput(messages.profile.linkedin, "https://google.com/foo")
    await waitFor(() =>
      expect(
        screen.getByText(messages.profile.socialLinkErrorLinkedin),
      ).toBeInTheDocument(),
    )

    changeInput(
      messages.profile.linkedin,
      "https://www.linkedin.com/in/jeanne-doe",
    )
    await waitFor(() =>
      expect(
        screen.queryByText(messages.profile.socialLinkErrorLinkedin),
      ).toBeNull(),
    )
  })

  it("rejects a malformed URL with the generic invalid-URL message", async () => {
    renderModal()
    changeInput(messages.profile.linkedin, "not-a-url")
    await waitFor(() =>
      expect(
        screen.getByText(messages.profile.socialLinksUrlInvalid),
      ).toBeInTheDocument(),
    )
  })

  it("accepts youtube.com AND youtu.be URLs", async () => {
    renderModal()
    changeInput(messages.profile.youtube, "https://www.youtube.com/@jdoe")
    await waitFor(() =>
      expect(
        screen.queryByText(messages.profile.socialLinkErrorYoutube),
      ).toBeNull(),
    )
    changeInput(messages.profile.youtube, "https://youtu.be/abcd1234")
    await waitFor(() =>
      expect(
        screen.queryByText(messages.profile.socialLinkErrorYoutube),
      ).toBeNull(),
    )
  })

  it("accepts twitter.com AND x.com URLs", async () => {
    renderModal()
    changeInput(messages.profile.twitter, "https://twitter.com/jdoe")
    await waitFor(() =>
      expect(
        screen.queryByText(messages.profile.socialLinkErrorTwitter),
      ).toBeNull(),
    )
    changeInput(messages.profile.twitter, "https://x.com/jdoe")
    await waitFor(() =>
      expect(
        screen.queryByText(messages.profile.socialLinkErrorTwitter),
      ).toBeNull(),
    )
  })

  it("rejects a non-instagram URL in the instagram field", async () => {
    renderModal()
    changeInput(messages.profile.instagram, "https://facebook.com/jdoe")
    await waitFor(() =>
      expect(
        screen.getByText(messages.profile.socialLinkErrorInstagram),
      ).toBeInTheDocument(),
    )
  })

  it("rejects a non-github URL in the github field", async () => {
    renderModal()
    changeInput(messages.profile.github, "https://gitlab.com/jdoe")
    await waitFor(() =>
      expect(
        screen.getByText(messages.profile.socialLinkErrorGithub),
      ).toBeInTheDocument(),
    )
  })

  it("accepts any valid URL in the website field (free-form)", async () => {
    renderModal()
    changeInput(messages.profile.website, "https://example.com/about")
    // No domain-specific error key exists for website. Confirm that
    // none of the other domain errors leak through, and that the save
    // button becomes enabled.
    await waitFor(() => {
      const save = screen.getByRole("button", {
        name: messages.common.save,
      }) as HTMLButtonElement
      expect(save.disabled).toBe(false)
    })
  })

  it("submits — calls onUpsert for filled inputs and onDelete for cleared ones", async () => {
    const links: SocialLinkEntry[] = [
      { platform: "linkedin", url: "https://www.linkedin.com/in/old" },
      { platform: "github", url: "https://github.com/old" },
    ]
    const { onUpsert, onDelete, onClose } = renderModal({ links })

    // Update linkedin -> new value
    changeInput(
      messages.profile.linkedin,
      "https://www.linkedin.com/in/new",
    )
    // Clear github -> should delete
    changeInput(messages.profile.github, "")
    // Add website -> should upsert
    changeInput(messages.profile.website, "https://example.com")

    await waitFor(() => {
      const save = screen.getByRole("button", {
        name: messages.common.save,
      }) as HTMLButtonElement
      expect(save.disabled).toBe(false)
    })

    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: messages.common.save }),
      )
    })

    await waitFor(() => {
      expect(onUpsert).toHaveBeenCalledWith(
        "linkedin",
        "https://www.linkedin.com/in/new",
      )
      expect(onUpsert).toHaveBeenCalledWith(
        "website",
        "https://example.com",
      )
      expect(onDelete).toHaveBeenCalledWith("github")
      expect(onClose).toHaveBeenCalled()
    })
    // Untouched, unchanged: no extra calls beyond the three above.
    expect(onUpsert).toHaveBeenCalledTimes(2)
    expect(onDelete).toHaveBeenCalledTimes(1)
  })

  it("does not call onUpsert when a value is unchanged from initial", async () => {
    const links: SocialLinkEntry[] = [
      { platform: "linkedin", url: "https://www.linkedin.com/in/me" },
    ]
    const { onUpsert, onDelete } = renderModal({ links })

    // Make a different field dirty so the form is dirty enough to submit.
    changeInput(messages.profile.website, "https://example.com")

    await waitFor(() => {
      const save = screen.getByRole("button", {
        name: messages.common.save,
      }) as HTMLButtonElement
      expect(save.disabled).toBe(false)
    })

    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: messages.common.save }),
      )
    })

    await waitFor(() => {
      expect(onUpsert).toHaveBeenCalledWith(
        "website",
        "https://example.com",
      )
    })
    // Linkedin was unchanged — never re-upserted.
    expect(onUpsert).not.toHaveBeenCalledWith(
      "linkedin",
      expect.any(String),
    )
    expect(onDelete).not.toHaveBeenCalled()
  })

  it("calls onClose when the cancel button is clicked", () => {
    const { onClose } = renderModal()
    fireEvent.click(
      screen.getByRole("button", { name: messages.common.cancel }),
    )
    expect(onClose).toHaveBeenCalled()
  })
})
