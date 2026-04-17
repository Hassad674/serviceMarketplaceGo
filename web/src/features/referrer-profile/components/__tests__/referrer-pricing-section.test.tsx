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

describe("ReferrerPricingSection (V1 single-field form)", () => {
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

  it("opens the inline editor without the legacy type dropdown", () => {
    renderSection()
    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.pricing.editButton }),
    )
    // V1: no type radio group — the referrer persona is locked to
    // commission_pct.
    expect(
      screen.queryByRole("radiogroup", {
        name: messages.profile.pricing.typeGroupLabel,
      }),
    ).not.toBeInTheDocument()
    // Only the single "Commission (%)" input is present.
    expect(
      screen.getByLabelText(messages.profile.pricing.referrerCommissionLabel),
    ).toBeInTheDocument()
  })

  it("calls the upsert mutation with commission_pct + min==max basis points", async () => {
    upsertMutateAsync.mockResolvedValue({})
    renderSection()
    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.pricing.editButton }),
    )
    fireEvent.change(
      screen.getByLabelText(messages.profile.pricing.referrerCommissionLabel),
      { target: { value: "10" } },
    )
    fireEvent.click(screen.getByTestId("referrer-pricing-save"))
    await waitFor(() => expect(upsertMutateAsync).toHaveBeenCalledTimes(1))
    const payload = upsertMutateAsync.mock.calls[0][0]
    expect(payload.type).toBe("commission_pct")
    expect(payload.min_amount).toBe(1000)
    expect(payload.max_amount).toBe(1000)
    expect(payload.currency).toBe("pct")
  })

  it("clamps input to the [0..100] percentage range", async () => {
    upsertMutateAsync.mockResolvedValue({})
    renderSection()
    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.pricing.editButton }),
    )
    const input = screen.getByLabelText(
      messages.profile.pricing.referrerCommissionLabel,
    )
    // Above 100 must be clamped down to 100.
    fireEvent.change(input, { target: { value: "250" } })
    expect((input as HTMLInputElement).value).toBe("100")
    // Below 0 must be clamped up to 0.
    fireEvent.change(input, { target: { value: "-5" } })
    expect((input as HTMLInputElement).value).toBe("0")
  })
})
