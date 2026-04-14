import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { PricingSection } from "../pricing-section"
import type { Pricing } from "../../api/profile-api"

let mockRows: Pricing[] = []

vi.mock("../../hooks/use-pricing", () => ({
  usePricing: () => ({ data: mockRows }),
  pricingQueryKey: () => ["user", "x", "profile", "pricing"],
}))

// Stub the heavy modal so this test stays focused on the card summary.
vi.mock("../pricing-editor-modal", () => ({
  PricingEditorModal: ({ open }: { open: boolean }) =>
    open ? <div role="dialog">modal-open</div> : null,
}))

function renderSection(
  props: Partial<Parameters<typeof PricingSection>[0]> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const defaults = {
    variant: "direct" as const,
    orgType: "provider_personal",
    referrerEnabled: false,
    readOnly: false,
  }
  return render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <PricingSection {...defaults} {...props} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockRows = []
})

describe("PricingSection", () => {
  it("renders nothing for enterprise", () => {
    const { container } = renderSection({ orgType: "enterprise" })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders nothing for referral variant when referrer_enabled is false", () => {
    const { container } = renderSection({
      variant: "referral",
      orgType: "provider_personal",
      referrerEnabled: false,
    })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the referral variant for providers with referrer_enabled", () => {
    renderSection({
      variant: "referral",
      orgType: "provider_personal",
      referrerEnabled: true,
    })
    expect(
      screen.getByText(messages.profile.pricing.referralSectionTitle),
    ).toBeInTheDocument()
  })

  it("renders nothing for referral variant when org is agency", () => {
    const { container } = renderSection({
      variant: "referral",
      orgType: "agency",
      referrerEnabled: true,
    })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the empty state when no pricing rows are persisted", () => {
    renderSection()
    expect(
      screen.getByText(messages.profile.pricing.empty),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", {
        name: new RegExp(messages.profile.pricing.editButton, "i"),
      }),
    ).toBeInTheDocument()
  })

  it("renders the direct pricing row as a chip with negotiable badge", () => {
    mockRows = [
      {
        kind: "direct",
        type: "daily",
        min_amount: 50000,
        max_amount: null,
        currency: "EUR",
        note: "",
        negotiable: true,
      },
    ]
    renderSection()
    expect(screen.getByText(/500/)).toBeInTheDocument()
    expect(
      screen.getByText(messages.profile.pricing.negotiableBadge),
    ).toBeInTheDocument()
  })

  it("opens the modal when the edit button is clicked", async () => {
    const user = userEvent.setup()
    renderSection()
    await user.click(
      screen.getByRole("button", {
        name: new RegExp(messages.profile.pricing.editButton, "i"),
      }),
    )
    expect(screen.getByRole("dialog")).toBeInTheDocument()
  })

  it("hides the edit button in readOnly mode", () => {
    mockRows = [
      {
        kind: "direct",
        type: "hourly",
        min_amount: 7500,
        max_amount: null,
        currency: "EUR",
        note: "",
        negotiable: false,
      },
    ]
    renderSection({ readOnly: true })
    expect(
      screen.queryByRole("button", {
        name: new RegExp(messages.profile.pricing.editButton, "i"),
      }),
    ).not.toBeInTheDocument()
  })

  it("only displays the row matching the variant", () => {
    mockRows = [
      {
        kind: "direct",
        type: "daily",
        min_amount: 50000,
        max_amount: null,
        currency: "EUR",
        note: "",
        negotiable: false,
      },
      {
        kind: "referral",
        type: "commission_pct",
        min_amount: 500,
        max_amount: 1500,
        currency: "pct",
        note: "",
        negotiable: false,
      },
    ]
    renderSection({ variant: "direct" })
    expect(screen.getByText(/500/)).toBeInTheDocument()
  })
})
