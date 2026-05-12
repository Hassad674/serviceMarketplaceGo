// Tests for BillingProfileEmbed — the inline embed that wraps the
// shared `BillingProfileForm` in two modes (summary or form). The
// parent owns the mode toggle: this test suite focuses on the embed's
// rendering contract (loading state, summary vs form, completePrompt
// banner, onSaved/onEdit propagation).

import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import type {
  BillingProfile,
  BillingProfileSnapshot,
} from "@/shared/types/billing-profile"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => `i18n:${key}`,
}))

// Stub the canonical form so we don't need a QueryClientProvider just
// to exercise the embed's branches. Exposes a save button so we can
// fire onSaved synchronously from the test. The form stub also
// records the `showStripePrefill` flag as a data-attribute so the
// embed's prop threading can be asserted from the test.
vi.mock("@/features/invoicing/components/billing-profile-form", () => ({
  BillingProfileForm: ({
    variant,
    onSaved,
    showStripePrefill,
  }: {
    variant?: string
    onSaved?: () => void
    showStripePrefill?: boolean
  }) => (
    <div
      data-testid="form-stub"
      data-variant={variant ?? ""}
      data-show-prefill={String(showStripePrefill ?? "")}
    >
      <button data-testid="form-save" onClick={onSaved}>
        save
      </button>
    </div>
  ),
}))

// Replace the TanStack Query hook with a per-test variable so each
// case fixes the data + isLoading return values explicitly.
let mockReturn: {
  data: BillingProfileSnapshot | undefined
  isLoading: boolean
} = { data: undefined, isLoading: false }
vi.mock("@/shared/hooks/billing-profile/use-billing-profile", () => ({
  useBillingProfile: () => mockReturn,
}))

const fullProfile: BillingProfile = {
  organization_id: "org-1",
  profile_type: "business",
  legal_name: "Acme Studio SARL",
  trading_name: "",
  legal_form: "SARL",
  tax_id: "12345678901234",
  vat_number: "",
  vat_validated_at: null,
  address_line1: "12 rue de la Paix",
  address_line2: "",
  postal_code: "75001",
  city: "Paris",
  country: "FR",
  invoicing_email: "",
  synced_from_kyc_at: null,
}

const completeSnapshot: BillingProfileSnapshot = {
  profile: fullProfile,
  missing_fields: [],
  is_complete: true,
}

const incompleteSnapshot: BillingProfileSnapshot = {
  profile: { ...fullProfile, legal_name: "" },
  missing_fields: [{ field: "legal_name", reason: "required" }],
  is_complete: false,
}

beforeEach(() => {
  mockReturn = { data: undefined, isLoading: false }
})

async function renderEmbed(props: {
  mode: "summary" | "form"
  onEdit?: () => void
  onSaved?: () => void
  showStripePrefill?: boolean
}) {
  const { BillingProfileEmbed } = await import("../billing-profile-embed")
  return render(
    <BillingProfileEmbed
      mode={props.mode}
      onEdit={props.onEdit ?? (() => {})}
      onSaved={props.onSaved ?? (() => {})}
      {...(props.showStripePrefill !== undefined
        ? { showStripePrefill: props.showStripePrefill }
        : {})}
    />,
  )
}

describe("BillingProfileEmbed", () => {
  it("renders a loading indicator while the snapshot is fetching", async () => {
    mockReturn = { data: undefined, isLoading: true }
    await renderEmbed({ mode: "summary" })
    expect(screen.getByRole("status")).toBeInTheDocument()
  })

  it("renders the read-only summary when mode='summary' and profile is complete", async () => {
    mockReturn = { data: completeSnapshot, isLoading: false }
    await renderEmbed({ mode: "summary" })
    expect(screen.getByText("Acme Studio SARL")).toBeInTheDocument()
    expect(screen.queryByTestId("form-stub")).not.toBeInTheDocument()
  })

  it("renders the form (compact variant) when mode='form'", async () => {
    mockReturn = { data: completeSnapshot, isLoading: false }
    await renderEmbed({ mode: "form" })
    const stub = screen.getByTestId("form-stub")
    expect(stub).toBeInTheDocument()
    expect(stub.getAttribute("data-variant")).toBe("compact")
  })

  it("renders the completePrompt banner above the form when profile is incomplete", async () => {
    mockReturn = { data: incompleteSnapshot, isLoading: false }
    await renderEmbed({ mode: "form" })
    expect(screen.getByText("i18n:completePromptTitle")).toBeInTheDocument()
    expect(screen.getByText("i18n:completePromptBody")).toBeInTheDocument()
    // The form is also rendered so the user can fix the gaps.
    expect(screen.getByTestId("form-stub")).toBeInTheDocument()
  })

  it("does NOT render the completePrompt banner when profile is already complete and mode='form' (user opted to edit)", async () => {
    mockReturn = { data: completeSnapshot, isLoading: false }
    await renderEmbed({ mode: "form" })
    expect(screen.queryByText("i18n:completePromptTitle")).not.toBeInTheDocument()
    expect(screen.queryByText("i18n:completePromptBody")).not.toBeInTheDocument()
  })

  it("fires onEdit when the summary's Modifier button is clicked", async () => {
    mockReturn = { data: completeSnapshot, isLoading: false }
    const onEdit = vi.fn()
    await renderEmbed({ mode: "summary", onEdit })
    fireEvent.click(screen.getByRole("button", { name: /editCta/i }))
    expect(onEdit).toHaveBeenCalledTimes(1)
  })

  it("fires onSaved when the form fires its save callback", async () => {
    mockReturn = { data: incompleteSnapshot, isLoading: false }
    const onSaved = vi.fn()
    await renderEmbed({ mode: "form", onSaved })
    fireEvent.click(screen.getByTestId("form-save"))
    expect(onSaved).toHaveBeenCalledTimes(1)
  })

  it("falls back to the form when mode='summary' but no profile data is yet available (defensive)", async () => {
    // Edge case: snapshot loaded but returned undefined data (e.g. an
    // ApiError swallowed in TanStack Query's retry). The embed should
    // NOT crash — it falls back to the form so the user has SOMETHING
    // to do.
    mockReturn = { data: undefined, isLoading: false }
    await renderEmbed({ mode: "summary" })
    expect(screen.getByTestId("form-stub")).toBeInTheDocument()
  })

  // showStripePrefill threading — added with the client-payment-ux fix.
  // The embed must surface the prop on the underlying form so the
  // client checkout (payment-simulation.tsx) can hide the prestataire-
  // only Stripe prefill CTA.
  describe("showStripePrefill threading", () => {
    it("passes showStripePrefill={false} through to BillingProfileForm", async () => {
      mockReturn = { data: incompleteSnapshot, isLoading: false }
      await renderEmbed({ mode: "form", showStripePrefill: false })
      const stub = screen.getByTestId("form-stub")
      expect(stub.getAttribute("data-show-prefill")).toBe("false")
    })

    it("defaults to showStripePrefill=true when the prop is omitted (prestataire context preserved)", async () => {
      mockReturn = { data: incompleteSnapshot, isLoading: false }
      await renderEmbed({ mode: "form" })
      const stub = screen.getByTestId("form-stub")
      // The default surfaces as `true` on the form stub.
      expect(stub.getAttribute("data-show-prefill")).toBe("true")
    })

    it("explicit showStripePrefill={true} threads through unchanged", async () => {
      mockReturn = { data: incompleteSnapshot, isLoading: false }
      await renderEmbed({ mode: "form", showStripePrefill: true })
      const stub = screen.getByTestId("form-stub")
      expect(stub.getAttribute("data-show-prefill")).toBe("true")
    })
  })
})
