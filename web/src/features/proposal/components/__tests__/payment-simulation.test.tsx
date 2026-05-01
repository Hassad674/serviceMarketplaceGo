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

// Lucide icons render as inert spans — keeps the DOM uncluttered.
vi.mock("lucide-react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("lucide-react")>()
  return new Proxy(actual, {
    get(target, prop) {
      if (typeof prop === "string" && /^[A-Z]/.test(prop)) {
        return () => <span data-testid={`icon-${prop}`} />
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
})
