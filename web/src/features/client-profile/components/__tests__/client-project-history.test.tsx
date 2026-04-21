import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ClientProjectHistory } from "../client-project-history"
import type { ClientProjectHistoryEntry } from "../../api/client-profile-api"

// Mock next-intl navigation's Link — the feature under test uses
// `@i18n/navigation` which resolves to a Next.js-aware Link. In a
// unit test we just need the same semantics as a plain <a>.
vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    ...props
  }: {
    href: string
    children: React.ReactNode
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}))

import { vi } from "vitest"

function renderHistory(entries: ClientProjectHistoryEntry[]) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ClientProjectHistory entries={entries} />
    </NextIntlClientProvider>,
  )
}

const SAMPLE_ENTRY: ClientProjectHistoryEntry = {
  proposal_id: "prop-1",
  title: "Landing page redesign",
  amount: 500000, // 5 000,00 €
  completed_at: "2026-03-01T10:00:00Z",
  provider: {
    organization_id: "org-provider-1",
    display_name: "Alice Freelance",
    avatar_url: null,
  },
}

describe("ClientProjectHistory", () => {
  it("renders the empty state when no entries are provided", () => {
    renderHistory([])
    expect(
      screen.getByText(messages.clientProfile.projectHistoryEmpty),
    ).toBeInTheDocument()
  })

  it("renders an entry with title, amount and provider link", () => {
    renderHistory([SAMPLE_ENTRY])

    expect(screen.getByText("Landing page redesign")).toBeInTheDocument()
    expect(screen.getByText(/5\s?000/)).toBeInTheDocument()

    const providerLink = screen.getByRole("link", {
      name: /Alice Freelance/,
    })
    expect(providerLink).toHaveAttribute(
      "href",
      "/freelancers/org-provider-1",
    )
  })

  it("renders initials fallback when the provider has no avatar", () => {
    renderHistory([SAMPLE_ENTRY])
    // "A" initial is rendered inside the decorative avatar span.
    expect(screen.getByText("A")).toBeInTheDocument()
  })
})
