import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import {
  ReferrerSocialLinksSection,
  PublicReferrerSocialLinks,
} from "../referrer-social-links-section"

const upsertMutateAsync = vi.fn()
const deleteMutateAsync = vi.fn()
const hasPermissionMock = vi.fn().mockReturnValue(true)

vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: (...args: unknown[]) => hasPermissionMock(...args),
}))

vi.mock("../../hooks/use-referrer-social-links", () => ({
  useMyReferrerSocialLinks: () => ({
    data: [{ platform: "website", url: "https://blog.example.com" }],
    isLoading: false,
  }),
  usePublicReferrerSocialLinks: (orgId: string) => ({
    data: orgId === "empty"
      ? []
      : [{ platform: "twitter", url: "https://twitter.com/u" }],
    isLoading: false,
  }),
  useUpsertReferrerSocialLink: () => ({ mutateAsync: upsertMutateAsync }),
  useDeleteReferrerSocialLink: () => ({ mutateAsync: deleteMutateAsync }),
}))

function renderWithIntl(ui: React.ReactElement) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      {ui}
    </NextIntlClientProvider>,
  )
}

describe("ReferrerSocialLinksSection", () => {
  beforeEach(() => {
    upsertMutateAsync.mockReset()
    deleteMutateAsync.mockReset()
    hasPermissionMock.mockReturnValue(true)
  })

  it("renders the owner card with the edit button", () => {
    renderWithIntl(<ReferrerSocialLinksSection />)
    expect(
      screen.getByRole("button", { name: messages.profile.editSocialLinks }),
    ).toBeInTheDocument()
    expect(screen.getByText("website")).toBeInTheDocument()
  })

  it("hides the edit button when the user cannot edit", () => {
    hasPermissionMock.mockReturnValue(false)
    renderWithIntl(<ReferrerSocialLinksSection />)
    expect(
      screen.queryByRole("button", { name: messages.profile.editSocialLinks }),
    ).toBeNull()
  })
})

describe("PublicReferrerSocialLinks", () => {
  it("renders the public read-only card with links", () => {
    renderWithIntl(<PublicReferrerSocialLinks orgId="org-1" />)
    expect(screen.getByText("twitter")).toBeInTheDocument()
  })

  it("collapses to nothing when the set is empty", () => {
    const { container } = renderWithIntl(
      <PublicReferrerSocialLinks orgId="empty" />,
    )
    expect(container.firstChild).toBeNull()
  })
})
