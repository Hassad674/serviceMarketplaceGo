// Regression tests for the proposal payment page (PaymentSimulation).
//
// Bug context: when /api/v1/proposals/{id}/pay returned a 500 (e.g. the
// missing migration 131 dropped a column the SQL INSERT referenced), the
// original page caught the error and rendered "Proposal not found" with
// no diagnostic UI — making a real backend failure look like a missing
// resource. These tests pin the new behaviour:
//   - 404 on getProposal still renders proposalNotFound (legitimate).
//   - 5xx / 4xx on initiatePayment renders fetchError + a retry button,
//     never proposalNotFound. The user can recover instead of guessing.
//   - The empty-id branch still renders proposalNotFound.
//   - The success path renders the payment layout.

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, waitFor, fireEvent } from "@testing-library/react"
import type { ReactElement } from "react"
import { ApiError } from "@/shared/lib/api-client"

// ---------------------------------------------------------------------------
// Module mocks — wired before importing the component so the dynamic
// imports below see the mocked modules.
// ---------------------------------------------------------------------------

const getProposalMock = vi.fn()
const initiatePaymentMock = vi.fn()
const confirmPaymentMock = vi.fn()
vi.mock("../../api/proposal-api", () => ({
  getProposal: (...args: unknown[]) => getProposalMock(...args),
  initiatePayment: (...args: unknown[]) => initiatePaymentMock(...args),
  confirmPayment: (...args: unknown[]) => confirmPaymentMock(...args),
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

const replaceFn = vi.fn()
const pushFn = vi.fn()
const backFn = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: pushFn, back: backFn, replace: replaceFn }),
}))

let searchParamsMap = new Map<string, string>()
vi.mock("next/navigation", () => ({
  useSearchParams: () => ({
    get: (key: string) => searchParamsMap.get(key) ?? null,
  }),
}))

// Stripe is heavy and DOM-dependent — we never reach the Elements branch
// in these tests because we focus on the loading/error/success transitions
// before payment data is wired in. Mock the loaders to a no-op so the
// import graph stays cheap.
vi.mock("@stripe/stripe-js", () => ({
  loadStripe: vi.fn(() => Promise.resolve(null)),
}))

vi.mock("@stripe/react-stripe-js", () => ({
  Elements: ({ children }: { children: ReactElement }) => <div>{children}</div>,
  PaymentElement: () => <div data-testid="payment-element" />,
  useStripe: () => ({ confirmPayment: vi.fn() }),
  useElements: () => ({}),
}))

// Mock the billing-profile inline modal so the page-level tests don't
// have to stand up a QueryClientProvider just to render its form.
// The modal renders a tiny stub that exposes its open state and the
// onSaved callback so tests can verify the retry-on-save chain.
vi.mock("@/shared/components/billing-profile/billing-profile-inline-modal", () => ({
  BillingProfileInlineModal: ({
    open,
    onSaved,
    onClose,
  }: {
    open: boolean
    onSaved: () => void
    onClose: () => void
  }) =>
    open ? (
      <div data-testid="billing-modal">
        <button data-testid="billing-modal-save" onClick={onSaved}>
          save
        </button>
        <button data-testid="billing-modal-close" onClick={onClose}>
          close
        </button>
      </div>
    ) : null,
}))

// The embedded inline billing-profile card. Stubbed because its real
// implementation depends on TanStack Query + the (heavier) shared form;
// the parent's contract here is: "render embed in the requested mode,
// surface onSaved + onEdit for parent state changes".
vi.mock("@/shared/components/billing-profile/billing-profile-embed", () => ({
  BillingProfileEmbed: ({
    mode,
    onEdit,
    onSaved,
    showStripePrefill,
  }: {
    mode: "summary" | "form"
    onEdit: () => void
    onSaved: () => void
    showStripePrefill?: boolean
  }) => (
    <div
      data-testid="billing-embed"
      data-mode={mode}
      data-show-prefill={String(showStripePrefill ?? "")}
    >
      <button data-testid="billing-embed-edit" onClick={onEdit}>
        edit
      </button>
      <button data-testid="billing-embed-saved" onClick={onSaved}>
        saved
      </button>
    </div>
  ),
}))

// The TanStack Query hook used by the payment form to decide whether
// the Stripe Element should be rendered. By default the snapshot is
// COMPLETE so existing happy-path tests continue to render the layout.
// Individual tests can re-mock this with `.mockReturnValueOnce(...)`.
type BillingMockReturn = {
  data: {
    profile: {
      organization_id: string
      profile_type: "business" | "individual"
      legal_name: string
      trading_name: string
      legal_form: string
      tax_id: string
      vat_number: string
      vat_validated_at: string | null
      address_line1: string
      address_line2: string
      postal_code: string
      city: string
      country: string
      invoicing_email: string
      synced_from_kyc_at: string | null
    }
    missing_fields: { field: string; reason: string }[]
    is_complete: boolean
  } | undefined
  isLoading: boolean
}
const completeMock: BillingMockReturn = {
  data: {
    profile: {
      organization_id: "org-1",
      profile_type: "business",
      legal_name: "Acme",
      trading_name: "",
      legal_form: "",
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
    },
    missing_fields: [],
    is_complete: true,
  },
  isLoading: false,
}
const useBillingProfileMock = vi.fn<() => BillingMockReturn>(() => completeMock)
vi.mock("@/shared/hooks/billing-profile/use-billing-profile", () => ({
  useBillingProfile: () => useBillingProfileMock(),
}))

// Lucide icons render as inert spans — keeps the DOM uncluttered.
vi.mock("lucide-react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("lucide-react")>()
  return new Proxy(actual, {
    get(target, prop) {
      if (typeof prop === "string" && /^[A-Z]/.test(prop)) {
        const IconStub = () => <span data-testid={`icon-${prop}`} />
        IconStub.displayName = `IconStub(${prop})`
        return IconStub
      }
      return Reflect.get(target, prop)
    },
  })
})

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

const baseProposal = {
  id: "44ace76e-6686-4bbd-9bd6-0fd7f794851d",
  title: "fdsbgfdtrtrg",
  amount: 312300,
  status: "accepted",
  current_milestone_sequence: 1,
  milestones: [
    {
      id: "48fa0a09-18b5-45ce-8c94-c4b49620e937",
      sequence: 1,
      status: "pending_funding",
      amount: 312300,
    },
  ],
}

const stripePaymentData = {
  client_secret: "pi_test_secret",
  payment_record_id: "rec_id",
  amounts: {
    proposal_amount: 312300,
    stripe_fee: 4710,
    platform_fee: 2500,
    client_total: 317010,
    provider_payout: 309800,
  },
}

// ---------------------------------------------------------------------------
// Setup / teardown
// ---------------------------------------------------------------------------

beforeEach(() => {
  searchParamsMap = new Map([["proposal", baseProposal.id]])
  getProposalMock.mockReset()
  initiatePaymentMock.mockReset()
  confirmPaymentMock.mockReset()
  replaceFn.mockReset()
  pushFn.mockReset()
  backFn.mockReset()
  useBillingProfileMock.mockReset()
  useBillingProfileMock.mockReturnValue(completeMock)
})

afterEach(() => {
  vi.clearAllMocks()
})

// Dynamic import so each test sees the freshly-mocked dependencies.
async function renderPaymentSimulation() {
  const { PaymentSimulation } = await import("../payment-simulation")
  return render(<PaymentSimulation />)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("PaymentSimulation", () => {
  it("renders proposalNotFound when no proposal id is in the URL", async () => {
    searchParamsMap = new Map()
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    expect(screen.getByText("proposalNotFound")).toBeInTheDocument()
    expect(getProposalMock).not.toHaveBeenCalled()
  })

  it("renders proposalNotFound when getProposal returns 404", async () => {
    getProposalMock.mockRejectedValue(
      new ApiError(404, "proposal_not_found", "Proposal not found"),
    )

    await renderPaymentSimulation()

    await waitFor(() => {
      expect(screen.getByText("proposalNotFound")).toBeInTheDocument()
    })
    // The 404 path must NOT show the retry button (retrying won't help).
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument()
  })

  it("renders fetchError + retry button when initiatePayment returns 500 (regression: provider_organization_id missing column)", async () => {
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockRejectedValue(
      new ApiError(500, "internal_error", "an unexpected error occurred"),
    )

    await renderPaymentSimulation()

    // Waits for the fetchError banner — proves the page does NOT silently
    // mask a 500 as "Proposal not found" anymore.
    await waitFor(() => {
      expect(screen.getByText("paymentInitFailed")).toBeInTheDocument()
    })
    // A retry button is present so the user can recover. Without this
    // they were stuck on the misleading "not found" screen.
    const retryBtn = screen.getByRole("button", { name: /paymentRetry/i })
    expect(retryBtn).toBeInTheDocument()

    // proposalNotFound MUST NOT be on the page — that would be the
    // legacy bug behaviour.
    expect(screen.queryByText("proposalNotFound")).not.toBeInTheDocument()
  })

  it("clicking retry re-invokes initiatePayment and clears the error", async () => {
    getProposalMock.mockResolvedValue(baseProposal)
    // Toggle the implementation: every call before the user clicks
    // Retry rejects with a 500; every call after resolves with the
    // Stripe payload. Toggling on a flag (instead of using
    // mockRejectedValueOnce) survives React Strict Mode's double-
    // invoke of useEffect — a queued .once value would otherwise be
    // consumed by the strict-mode shadow render before the user ever
    // clicks the button.
    let mode: "fail" | "succeed" = "fail"
    initiatePaymentMock.mockImplementation(() =>
      mode === "fail"
        ? Promise.reject(new ApiError(500, "internal_error", "boom"))
        : Promise.resolve(stripePaymentData),
    )

    await renderPaymentSimulation()

    const retryBtn = await screen.findByRole("button", { name: /paymentRetry/i })
    mode = "succeed"
    fireEvent.click(retryBtn)

    // After retry, initiatePayment now resolves and the page
    // transitions into the payment layout — the proposal title
    // shows up. Reaching the layout proves the retry path clears
    // the error state and resumes the funding flow.
    expect(
      await screen.findByText(baseProposal.title),
    ).toBeInTheDocument()
    expect(screen.queryByText("paymentInitFailed")).not.toBeInTheDocument()
  })

  it("renders fetchError on a transient network failure (no status)", async () => {
    getProposalMock.mockRejectedValue(new TypeError("network down"))

    await renderPaymentSimulation()

    await waitFor(() => {
      expect(screen.getByText("paymentInitFailed")).toBeInTheDocument()
    })
    // The network case should also expose a retry path.
    expect(screen.getByRole("button", { name: /paymentRetry/i })).toBeInTheDocument()
  })

  it("redirects to the detail page when the proposal status is not payable", async () => {
    getProposalMock.mockResolvedValue({
      ...baseProposal,
      status: "completed",
    })
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    await waitFor(() => {
      expect(replaceFn).toHaveBeenCalledWith(`/projects/${baseProposal.id}`)
    })
    // initiatePayment must NOT be called once we redirect — we don't
    // want to lock funds on a non-payable proposal.
    expect(initiatePaymentMock).not.toHaveBeenCalled()
  })

  it("renders the Stripe payment layout on the happy path", async () => {
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    await waitFor(() => {
      // The proposal title is rendered in the layout header — proves
      // we left the loading state and entered the payment UI.
      expect(screen.getByText(baseProposal.title)).toBeInTheDocument()
    })
    expect(screen.queryByText("proposalNotFound")).not.toBeInTheDocument()
    expect(screen.queryByText("paymentInitFailed")).not.toBeInTheDocument()
  })

  it("renders the paid success state when initiatePayment returns status='paid' (simulation mode)", async () => {
    // Simulation mode has no Stripe key wired — backend returns
    // {status:"paid"} synchronously and the UI should jump straight
    // to the green checkmark + paymentSuccess copy.
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockResolvedValue({ status: "paid" })

    await renderPaymentSimulation()

    expect(await screen.findByText("paymentSuccess")).toBeInTheDocument()
  })

  it("supports a payable proposal in 'active' status with a pending_funding milestone", async () => {
    // Mid-flow milestone funding: previous milestones already
    // released, current one is awaiting funds. The page must NOT
    // redirect (it's a legal funding state) and must reach the
    // payment layout.
    getProposalMock.mockResolvedValue({
      ...baseProposal,
      status: "active",
      milestones: [
        { ...baseProposal.milestones[0], status: "released" },
        {
          id: "m2",
          sequence: 2,
          status: "pending_funding",
          amount: 100000,
        },
      ],
      current_milestone_sequence: 2,
    })
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    expect(await screen.findByText(baseProposal.title)).toBeInTheDocument()
    expect(replaceFn).not.toHaveBeenCalled()
  })

  it("renders the billing-incomplete CTA + opens the inline modal on 412 billing_profile_incomplete", async () => {
    // The backend gate (handler-level) returns a 412 when the client
    // organization has not yet filled in its billing profile. The page
    // MUST NOT show the generic "init failed" UI — instead it shows the
    // billing-incomplete copy and the inline modal carrying the reusable
    // billing-profile form.
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockRejectedValue(
      new ApiError(
        412,
        "billing_profile_incomplete",
        "Complète tes infos de facturation",
        {
          error: {
            code: "billing_profile_incomplete",
            message: "Complète tes infos de facturation",
          },
          missing_fields: [
            { field: "legal_name", reason: "required" },
            { field: "tax_id", reason: "required" },
          ],
        },
      ),
    )

    await renderPaymentSimulation()

    await waitFor(() => {
      expect(screen.getByText("billingIncompleteTitle")).toBeInTheDocument()
    })
    // Generic init-failed copy MUST be absent — the user is not lost,
    // they just need to fill a form.
    expect(screen.queryByText("paymentInitFailed")).not.toBeInTheDocument()
    expect(screen.queryByText("proposalNotFound")).not.toBeInTheDocument()

    // The inline modal opens immediately — that's what the user
    // sees first, the static page content is just the fallback when
    // they dismiss it.
    expect(screen.getByTestId("billing-modal")).toBeInTheDocument()

    // The dedicated CTA is on screen so the user can re-open the modal
    // even after dismissing it the first time.
    expect(
      screen.getByRole("button", { name: /billingIncompleteCta/i }),
    ).toBeInTheDocument()
  })

  it("retries the payment intent after the inline modal saves the profile", async () => {
    // The end-to-end chain: backend rejects with 412, the modal opens,
    // the form saves successfully, the parent retries InitiatePayment
    // and lands on the Stripe payment layout. The mocked modal exposes
    // a "save" button that invokes onSaved synchronously.
    getProposalMock.mockResolvedValue(baseProposal)

    // Toggle the implementation on a flag instead of a counter so the
    // test survives React strict-mode's effect double-invoke (a counter
    // would race with the shadow render and the wrong mode would be
    // active by the time the user clicks save). Same pattern as the
    // legacy "retry button" test above.
    let mode: "fail" | "succeed" = "fail"
    initiatePaymentMock.mockImplementation(() =>
      mode === "fail"
        ? Promise.reject(
            new ApiError(
              412,
              "billing_profile_incomplete",
              "Complète tes infos de facturation",
              {
                error: { code: "billing_profile_incomplete", message: "x" },
                missing_fields: [],
              },
            ),
          )
        : Promise.resolve(stripePaymentData),
    )

    await renderPaymentSimulation()

    const saveBtn = await screen.findByTestId("billing-modal-save")
    mode = "succeed"
    fireEvent.click(saveBtn)

    // After the modal save, the page transitions through to the
    // Stripe layout — the proposal title appears.
    expect(await screen.findByText(baseProposal.title)).toBeInTheDocument()
  })

  it("renders proposalNotFound when initiatePayment itself returns 404 (proposal vanished mid-flow)", async () => {
    // Pathological but possible: getProposal succeeds, the user
    // takes a long time to click "Pay", and meanwhile the proposal
    // was hard-deleted. initiatePayment returns 404 — we must show
    // "not found", not the recoverable error UI (retry would never
    // succeed because the resource is gone).
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockRejectedValue(
      new ApiError(404, "proposal_not_found", "Proposal not found"),
    )

    await renderPaymentSimulation()

    await waitFor(() => {
      expect(screen.getByText("proposalNotFound")).toBeInTheDocument()
    })
    expect(screen.queryByText("paymentInitFailed")).not.toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: /paymentRetry/i }),
    ).not.toBeInTheDocument()
  })

  it("renders the embedded billing-profile in summary mode when the profile is complete on first paint (BILLING-IDENTITY-CLONE)", async () => {
    // Default mock already returns a complete profile. The happy-path
    // payment flow renders the embed in read-only mode AND the
    // SimulationFallback's confirmPayment button (the test env has no
    // NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY, so the page renders the
    // simulation branch — same gating logic applies on the Stripe
    // branch in production).
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    const embed = await screen.findByTestId("billing-embed")
    expect(embed).toBeInTheDocument()
    expect(embed.getAttribute("data-mode")).toBe("summary")
    // The confirmPayment CTA is rendered (gated by isPaymentReady === true).
    expect(
      screen.getByRole("button", { name: /confirmPayment/i }),
    ).toBeInTheDocument()
  })

  it("renders the embedded billing-profile in FORM mode and HIDES the payment CTA when the profile is incomplete (BILLING-IDENTITY-CLONE gate)", async () => {
    // Mock the snapshot as incomplete — the embed should default to
    // form mode and the confirmPayment button MUST NOT render (whether
    // the underlying branch is Stripe Elements or the simulation
    // fallback). This is the gate from the brief: the client never
    // sees the card form until their receipt identity is on file.
    useBillingProfileMock.mockReturnValue({
      data: {
        profile: {
          organization_id: "org-1",
          profile_type: "business",
          legal_name: "",
          trading_name: "",
          legal_form: "",
          tax_id: "",
          vat_number: "",
          vat_validated_at: null,
          address_line1: "",
          address_line2: "",
          postal_code: "",
          city: "",
          country: "FR",
          invoicing_email: "",
          synced_from_kyc_at: null,
        },
        missing_fields: [{ field: "legal_name", reason: "required" }],
        is_complete: false,
      },
      isLoading: false,
    })

    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    const embed = await screen.findByTestId("billing-embed")
    expect(embed.getAttribute("data-mode")).toBe("form")
    expect(screen.queryByTestId("payment-element")).not.toBeInTheDocument()
    expect(
      screen.queryByRole("button", { name: /confirmPayment/i }),
    ).not.toBeInTheDocument()
  })

  it("flips the embed back to summary mode after onSaved fires, exposing the payment CTA (BILLING-IDENTITY-CLONE save→pay)", async () => {
    // Start with the default complete profile so the embed defaults to
    // summary. Clicking "edit" flips to form; clicking "saved" inside
    // the form stub flips back to summary.
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    let embed = await screen.findByTestId("billing-embed")
    expect(embed.getAttribute("data-mode")).toBe("summary")

    fireEvent.click(screen.getByTestId("billing-embed-edit"))
    // The click triggers a state change AND useRouter returns a fresh
    // object on every render → React's effect dep array sees a new
    // `router` and re-fires the data-load pipeline. waitFor re-checks
    // the embed once the pipeline has settled with the new mode.
    await waitFor(() => {
      const updated = screen.getByTestId("billing-embed")
      expect(updated.getAttribute("data-mode")).toBe("form")
    })
    // In form mode the confirmPayment CTA MUST NOT render.
    expect(
      screen.queryByRole("button", { name: /confirmPayment/i }),
    ).not.toBeInTheDocument()

    fireEvent.click(screen.getByTestId("billing-embed-saved"))
    await waitFor(() => {
      embed = screen.getByTestId("billing-embed")
      expect(embed.getAttribute("data-mode")).toBe("summary")
    })
    // Back to summary → confirmPayment CTA renders again.
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /confirmPayment/i }),
      ).toBeInTheDocument()
    })
  })

  it("passes showStripePrefill={false} to the billing embed (client context, no Connect KYC to pull from)", async () => {
    // Regression: clients (enterprise role) don't have a Stripe Connect
    // KYC record. Surfacing the "Pré-remplir depuis Stripe" button on
    // their checkout is meaningless. The payment page must explicitly
    // disable the prefill CTA via the embed prop.
    getProposalMock.mockResolvedValue(baseProposal)
    initiatePaymentMock.mockResolvedValue(stripePaymentData)

    await renderPaymentSimulation()

    const embed = await screen.findByTestId("billing-embed")
    expect(embed.getAttribute("data-show-prefill")).toBe("false")
  })
})
