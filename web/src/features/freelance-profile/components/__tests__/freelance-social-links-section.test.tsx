import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import {
  FreelanceSocialLinksSection,
  PublicFreelanceSocialLinks,
} from "../freelance-social-links-section"

const upsertMutateAsync = vi.fn()
const deleteMutateAsync = vi.fn()
const hasPermissionMock = vi.fn().mockReturnValue(true)

vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: (...args: unknown[]) => hasPermissionMock(...args),
}))

vi.mock("../../hooks/use-freelance-social-links", () => ({
  useMyFreelanceSocialLinks: () => ({
    data: [{ platform: "github", url: "https://github.com/u" }],
    isLoading: false,
  }),
  usePublicFreelanceSocialLinks: (orgId: string) => ({
    data: orgId === "empty"
      ? []
      : [{ platform: "linkedin", url: "https://linkedin.com/in/u" }],
    isLoading: false,
  }),
  useUpsertFreelanceSocialLink: () => ({ mutateAsync: upsertMutateAsync }),
  useDeleteFreelanceSocialLink: () => ({ mutateAsync: deleteMutateAsync }),
}))

function renderWithIntl(ui: React.ReactElement) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      {ui}
    </NextIntlClientProvider>,
  )
}

describe("FreelanceSocialLinksSection", () => {
  beforeEach(() => {
    upsertMutateAsync.mockReset()
    deleteMutateAsync.mockReset()
    hasPermissionMock.mockReturnValue(true)
  })

  it("renders the owner card with the edit button when the user has permission", () => {
    renderWithIntl(<FreelanceSocialLinksSection />)
    expect(
      screen.getByRole("button", { name: messages.profile.editSocialLinks }),
    ).toBeInTheDocument()
    expect(screen.getByText("github")).toBeInTheDocument()
  })

  it("hides the edit button when the user cannot edit", () => {
    hasPermissionMock.mockReturnValue(false)
    renderWithIntl(<FreelanceSocialLinksSection />)
    expect(
      screen.queryByRole("button", { name: messages.profile.editSocialLinks }),
    ).toBeNull()
  })
})

describe("PublicFreelanceSocialLinks", () => {
  it("renders the public read-only card with links", () => {
    renderWithIntl(<PublicFreelanceSocialLinks orgId="org-1" />)
    expect(screen.getByText("linkedin")).toBeInTheDocument()
  })

  it("collapses to nothing when the set is empty", () => {
    const { container } = renderWithIntl(
      <PublicFreelanceSocialLinks orgId="empty" />,
    )
    expect(container.firstChild).toBeNull()
  })
})
