import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { ReferrerPricingSection } from "../referrer-pricing-section"

const upsertMutateAsync = vi.fn()
const deleteMutateAsync = vi.fn()

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "user-1",
}))
vi.mock("../../hooks/use-referrer-pricing", () => ({
  useReferrerPricing: () => ({ data: null }),
}))
vi.mock("../../hooks/use-upsert-referrer-pricing", () => ({
  useUpsertReferrerPricing: () => ({
    mutateAsync: upsertMutateAsync,
    isPending: false,
  }),
}))
vi.mock("../../hooks/use-delete-referrer-pricing", () => ({
  useDeleteReferrerPricing: () => ({
    mutateAsync: deleteMutateAsync,
    isPending: false,
  }),
}))

function renderSection() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <QueryClientProvider client={client}>
        <ReferrerPricingSection />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

describe("ReferrerPricingSection", () => {
  beforeEach(() => {
    upsertMutateAsync.mockReset()
    deleteMutateAsync.mockReset()
  })

  it("renders the referral empty placeholder", () => {
    renderSection()
    expect(
      screen.getByText(messages.profile.pricing.empty),
    ).toBeInTheDocument()
  })

  it("exposes only commission pricing types in the editor", () => {
    renderSection()
    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.pricing.editButton }),
    )
    expect(
      screen.getByRole("radio", {
        name: messages.profile.pricing.typeCommissionPct,
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("radio", {
        name: messages.profile.pricing.typeCommissionFlat,
      }),
    ).toBeInTheDocument()
    // Freelance types must NOT be reachable from the referrer editor.
    expect(
      screen.queryByRole("radio", {
        name: messages.profile.pricing.typeDaily,
      }),
    ).not.toBeInTheDocument()
  })

  it("calls the upsert mutation with basis-point commission on save", async () => {
    upsertMutateAsync.mockResolvedValue({})
    renderSection()
    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.pricing.editButton }),
    )
    fireEvent.click(
      screen.getByRole("radio", {
        name: messages.profile.pricing.typeCommissionPct,
      }),
    )
    fireEvent.change(
      screen.getByLabelText(messages.profile.pricing.minAmountLabel),
      { target: { value: "5" } },
    )
    fireEvent.change(
      screen.getByLabelText(messages.profile.pricing.maxAmountLabel),
      { target: { value: "15" } },
    )
    fireEvent.click(screen.getByTestId("referrer-pricing-save"))
    await waitFor(() => expect(upsertMutateAsync).toHaveBeenCalledTimes(1))
    const payload = upsertMutateAsync.mock.calls[0][0]
    expect(payload.type).toBe("commission_pct")
    expect(payload.min_amount).toBe(500)
    expect(payload.max_amount).toBe(1500)
    expect(payload.currency).toBe("pct")
  })
})
