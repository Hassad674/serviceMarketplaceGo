import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { FreelancePricingSection } from "../freelance-pricing-section"

const upsertMutateAsync = vi.fn()
const deleteMutateAsync = vi.fn()

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "user-1",
}))
vi.mock("../../hooks/use-freelance-pricing", () => ({
  useFreelancePricing: () => ({ data: null }),
}))
vi.mock("../../hooks/use-upsert-freelance-pricing", () => ({
  useUpsertFreelancePricing: () => ({
    mutateAsync: upsertMutateAsync,
    isPending: false,
  }),
}))
vi.mock("../../hooks/use-delete-freelance-pricing", () => ({
  useDeleteFreelancePricing: () => ({
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
        <FreelancePricingSection />
      </QueryClientProvider>
    </NextIntlClientProvider>,
  )
}

describe("FreelancePricingSection (V1 single-field form)", () => {
  beforeEach(() => {
    upsertMutateAsync.mockReset()
    deleteMutateAsync.mockReset()
  })

  it("renders the empty placeholder when no pricing is set", () => {
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
    // V1: the editor MUST NOT surface the pricing-type radio group —
    // the freelance persona is locked to `daily` (TJM).
    expect(
      screen.queryByRole("radiogroup", {
        name: messages.profile.pricing.typeGroupLabel,
      }),
    ).not.toBeInTheDocument()
    // The single amount input is labelled with the V1 TJM label.
    expect(
      screen.getByLabelText(messages.profile.pricing.freelanceDailyLabel),
    ).toBeInTheDocument()
  })

  it("calls the upsert mutation with daily + EUR + cents on save", async () => {
    upsertMutateAsync.mockResolvedValue({})
    renderSection()
    fireEvent.click(
      screen.getByRole("button", { name: messages.profile.pricing.editButton }),
    )
    fireEvent.change(
      screen.getByLabelText(messages.profile.pricing.freelanceDailyLabel),
      { target: { value: "500" } },
    )
    fireEvent.click(screen.getByTestId("freelance-pricing-save"))
    await waitFor(() => expect(upsertMutateAsync).toHaveBeenCalledTimes(1))
    const payload = upsertMutateAsync.mock.calls[0][0]
    expect(payload.type).toBe("daily")
    expect(payload.min_amount).toBe(50000)
    expect(payload.max_amount).toBeNull()
    expect(payload.currency).toBe("EUR")
  })
})
