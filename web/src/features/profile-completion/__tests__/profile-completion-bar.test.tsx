import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { NextIntlClientProvider } from "next-intl"

import messages from "@/../messages/fr.json"

vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    className,
    onClick,
    ...rest
  }: {
    href: string
    children: React.ReactNode
    className?: string
    onClick?: () => void
    [key: string]: unknown
  }) => (
    <a href={href} className={className} onClick={onClick} {...rest}>
      {children}
    </a>
  ),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: vi.fn(() => "user-1"),
}))

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: vi.fn(),
  API_BASE_URL: "http://localhost:8080",
}))

// useUser / useOrganization both read the same `/auth/me` cache. The
// mock returns whatever the test sets via `setSessionUser` so the bar
// can resolve role + org type before rendering.
let mockUser: { role: string } | undefined
let mockOrg: { type: string } | undefined
vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: mockUser }),
  useOrganization: () => ({ data: mockOrg }),
}))

import { apiClient } from "@/shared/lib/api-client"
import { ProfileCompletionBar } from "../components/profile-completion-bar"
import type { ProfileCompletionReport } from "../api/profile-completion-api"

const mockedApiClient = vi.mocked(apiClient)

function renderBar(
  report: ProfileCompletionReport,
  props: Parameters<typeof ProfileCompletionBar>[0] = {},
  session: { role?: string; orgType?: string } = {},
) {
  mockUser = session.role ? { role: session.role } : { role: "provider" }
  mockOrg = session.orgType ? { type: session.orgType } : { type: "provider_personal" }
  mockedApiClient.mockResolvedValue(report)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <QueryClientProvider client={client}>
      <NextIntlClientProvider locale="fr" messages={messages}>
        <ProfileCompletionBar {...props} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

const baseReport: ProfileCompletionReport = {
  role: "provider",
  persona: "freelance",
  percent: 50,
  total_sections: 11,
  filled_sections: 5,
  sections: [
    {
      key: "title",
      filled: true,
      label_key: "profile.completion.section.title",
      completion_path: "/dashboard/profile/edit",
    },
    {
      key: "about",
      filled: false,
      label_key: "profile.completion.section.about",
      completion_path: "/dashboard/profile/edit",
    },
  ],
}

beforeEach(() => {
  vi.clearAllMocks()
  mockUser = undefined
  mockOrg = undefined
})

describe("ProfileCompletionBar", () => {
  it("renders title with the percent and the filled/total subtitle once data loads", async () => {
    renderBar(baseReport)
    expect(await screen.findByText(/Profil rempli à 50%/)).toBeInTheDocument()
    expect(screen.getByText(/5\/11 sections complétées/)).toBeInTheDocument()
  })

  it("renders progressbar with correct aria values", async () => {
    renderBar(baseReport)
    const bar = await screen.findByRole("progressbar")
    expect(bar.getAttribute("aria-valuenow")).toBe("50")
    expect(bar.getAttribute("aria-valuemin")).toBe("0")
    expect(bar.getAttribute("aria-valuemax")).toBe("100")
  })

  it("renders zero percent state with 0/N subtitle", async () => {
    renderBar({ ...baseReport, percent: 0, filled_sections: 0 })
    const bar = await screen.findByRole("progressbar")
    expect(bar.getAttribute("aria-valuenow")).toBe("0")
    expect(screen.getByText(/0\/11 sections complétées/)).toBeInTheDocument()
  })

  it("renders the complete subtitle at 100%", async () => {
    renderBar({
      ...baseReport,
      percent: 100,
      filled_sections: baseReport.total_sections,
    })
    expect(
      await screen.findByText(/Toutes les sections sont complètes/),
    ).toBeInTheDocument()
  })

  it("hides itself at 100% when hideWhenComplete is true", async () => {
    const { container } = renderBar(
      { ...baseReport, percent: 100, filled_sections: baseReport.total_sections },
      { hideWhenComplete: true },
    )
    // Wait a tick for the query to resolve and the component to re-render.
    await new Promise((r) => setTimeout(r, 30))
    expect(container.firstChild).toBeNull()
  })

  it("renders as a Link to /profile for provider/freelance users", async () => {
    renderBar(baseReport, {}, { role: "provider", orgType: "provider_personal" })
    const link = await screen.findByRole("link", {
      name: /Profil rempli à 50%/,
    })
    expect(link.getAttribute("href")).toBe("/profile")
  })

  it("renders as a Link to /profile for agency users", async () => {
    renderBar(baseReport, {}, { role: "agency", orgType: "agency" })
    const link = await screen.findByRole("link", {
      name: /Profil rempli à 50%/,
    })
    expect(link.getAttribute("href")).toBe("/profile")
  })

  it("renders as a Link to /client-profile for enterprise users", async () => {
    renderBar(baseReport, {}, { role: "enterprise", orgType: "enterprise" })
    const link = await screen.findByRole("link", {
      name: /Profil rempli à 50%/,
    })
    expect(link.getAttribute("href")).toBe("/client-profile")
  })

  it("does NOT open a modal on click — navigation replaces the missing-list modal", async () => {
    renderBar(baseReport)
    await screen.findByRole("link", { name: /Profil rempli à 50%/ })
    // No dialog must ever exist — the bar is a plain Link now, not a
    // button with a popup.
    expect(screen.queryByRole("dialog")).toBeNull()
    expect(screen.queryByTestId("completion-section-list")).toBeNull()
  })

  it("renders the compact pill (still a Link) when sidebar variant is collapsed", async () => {
    renderBar(baseReport, { variant: "sidebar", collapsed: true })
    const pill = await screen.findByLabelText(/Profil rempli à 50%/)
    expect(pill.tagName.toLowerCase()).toBe("a")
    expect(pill.getAttribute("href")).toBe("/profile")
    expect(pill.textContent).toContain("50%")
  })
})
