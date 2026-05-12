import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"

import { ApiError } from "@/shared/lib/api-client"

// next-intl stub — echo the namespace.key for assertion clarity.
vi.mock("next-intl", () => ({
  useTranslations:
    (namespace?: string) =>
    (key: string, params?: Record<string, string | number>) => {
      const full = namespace ? `${namespace}.${key}` : key
      return params ? `${full}(${JSON.stringify(params)})` : full
    },
}))

// sonner toast — replace with vi.fn so we can assert calls. The
// factory must declare its own local because vi.mock hoists ABOVE
// the const declarations in the test file (vitest top-of-file hoist
// rule).
vi.mock("sonner", () => {
  const fn = vi.fn() as unknown as ((msg: string) => void) & {
    error: (msg: string) => void
  }
  ;(fn as { error: (msg: string) => void }).error = vi.fn()
  return { toast: fn }
})
import { toast as mockToast } from "sonner"

// next/link + navigation stubs (rendered modals may reach for them).
vi.mock("next/link", () => ({
  default: ({
    children,
    href,
  }: {
    children: React.ReactNode
    href: string
  }) => <a href={href}>{children}</a>,
}))
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

// Permissions + billing-profile gating mocks.
const mockHasPermission = vi.fn((_perm: string) => true)
vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: (perm: string) => mockHasPermission(perm),
}))

type CompletenessShape = {
  isComplete: boolean
  missingFields: { field: string; reason: string }[]
  isLoading: boolean
  isError: boolean
}
const mockCompleteness = vi.fn<() => CompletenessShape>(() => ({
  isComplete: true,
  missingFields: [],
  isLoading: false,
  isError: false,
}))
vi.mock(
  "@/shared/hooks/billing-profile/use-billing-profile-completeness",
  () => ({
    useBillingProfileCompleteness: () => mockCompleteness(),
  }),
)

vi.mock(
  "@/shared/components/billing-profile/billing-profile-completion-modal",
  () => ({
    BillingProfileCompletionModal: ({ open }: { open: boolean }) =>
      open ? <div role="dialog" aria-label="billing-modal" /> : null,
  }),
)

// useWalletSummary / useWalletWithdraw mocks — full control of the
// mutation lifecycle from the test.
const mockSummary = vi.fn()
const mockWithdrawMutate = vi.fn()
let mockWithdrawIsPending = false
vi.mock("../../hooks/use-wallet", () => ({
  useWalletSummary: () => mockSummary(),
  useWalletWithdraw: () => ({
    mutate: mockWithdrawMutate,
    isPending: mockWithdrawIsPending,
  }),
}))

// Lightweight stub for the history list so this test isn't entangled
// with the cursor pagination logic (that has its own dedicated test).
vi.mock("../wallet-unified-history", () => ({
  WalletUnifiedHistory: () => (
    <div data-testid="wallet-unified-history-stub" />
  ),
}))

// CommissionKYCRequiredModal stub — simple dialog flag.
vi.mock("../commission-kyc-required-modal", () => ({
  CommissionKYCRequiredModal: ({
    open,
    onboardingURL,
  }: {
    open: boolean
    onboardingURL?: string
  }) =>
    open ? (
      <div role="dialog" aria-label="kyc-modal" data-url={onboardingURL ?? ""} />
    ) : null,
}))

vi.mock("../wallet-withdraw-result-modal", () => ({
  WalletWithdrawResultModal: ({
    open,
    drainedCents,
  }: {
    open: boolean
    drainedCents: number
  }) =>
    open ? (
      <div role="dialog" aria-label="result-modal" data-drained={drainedCents} />
    ) : null,
}))

import { WalletUnifiedPage } from "../wallet-unified-page"

const emptyLeg = {
  total_cents: 0,
  available_cents: 0,
  escrowed_cents: 0,
  transmitted_cents: 0,
}

const summaryFixture = {
  currency: "EUR",
  total_cents: 1_000_00,
  available_cents: 700_00,
  escrowed_cents: 200_00,
  transmitted_cents: 100_00,
  breakdown: { missions: emptyLeg, commissions: emptyLeg },
  recent_transactions: [],
}

beforeEach(() => {
  vi.clearAllMocks()
  mockHasPermission.mockReturnValue(true)
  mockCompleteness.mockReturnValue({
    isComplete: true,
    missingFields: [],
    isLoading: false,
    isError: false,
  })
  mockWithdrawIsPending = false
  mockSummary.mockReturnValue({
    data: summaryFixture,
    isLoading: false,
    isError: false,
  })
})

describe("WalletUnifiedPage (Run C) — withdraw branches", () => {
  it("fires a success toast on a 200 OK with drained funds", async () => {
    mockWithdrawMutate.mockImplementation((_amt, opts) => {
      opts?.onSuccess?.({
        drained_cents: 700_00,
        missions_cents: 500_00,
        commissions_cents: 200_00,
        stripe_transfer_ids: ["tr_1"],
        currency: "EUR",
        errors: [],
      })
    })

    render(<WalletUnifiedPage />)
    fireEvent.click(screen.getByTestId("wallet-unified-withdraw"))
    await waitFor(() =>
      expect(mockToast).toHaveBeenCalledWith(
        "walletUnified.toast.success",
      ),
    )
    // Result modal stays closed on a clean 200.
    expect(
      screen.queryByRole("dialog", { name: /result-modal/ }),
    ).not.toBeInTheDocument()
  })

  it("opens the partial-success modal on a 207 with errors[]", async () => {
    mockWithdrawMutate.mockImplementation((_amt, opts) => {
      opts?.onSuccess?.({
        drained_cents: 300_00,
        missions_cents: 300_00,
        commissions_cents: 0,
        stripe_transfer_ids: ["tr_2"],
        currency: "EUR",
        errors: [
          {
            source: "commissions",
            code: "commission_drain_failed",
            message: "Transfer failed",
          },
        ],
      })
    })

    render(<WalletUnifiedPage />)
    fireEvent.click(screen.getByTestId("wallet-unified-withdraw"))
    await waitFor(() =>
      expect(
        screen.getByRole("dialog", { name: /result-modal/ }),
      ).toBeInTheDocument(),
    )
    expect(mockToast).toHaveBeenCalledWith("walletUnified.toast.partial")
  })

  it("opens the KYC modal on a 422 kyc_required ApiError with the onboarding URL", async () => {
    mockWithdrawMutate.mockImplementation((_amt, opts) => {
      const err = new ApiError(422, "kyc_required", "kyc required", {
        onboarding_url: "https://stripe/onboarding/abc",
      })
      opts?.onError?.(err)
    })

    render(<WalletUnifiedPage />)
    fireEvent.click(screen.getByTestId("wallet-unified-withdraw"))
    await waitFor(() =>
      expect(
        screen.getByRole("dialog", { name: /kyc-modal/ }),
      ).toBeInTheDocument(),
    )
    const dlg = screen.getByRole("dialog", { name: /kyc-modal/ })
    expect(dlg.getAttribute("data-url")).toBe(
      "https://stripe/onboarding/abc",
    )
  })

  it("opens the billing-profile modal on a 403 billing_profile_incomplete", async () => {
    mockWithdrawMutate.mockImplementation((_amt, opts) => {
      const err = new ApiError(
        403,
        "billing_profile_incomplete",
        "incomplete",
        { missing_fields: [{ field: "vat", reason: "missing" }] },
      )
      opts?.onError?.(err)
    })

    render(<WalletUnifiedPage />)
    fireEvent.click(screen.getByTestId("wallet-unified-withdraw"))
    await waitFor(() =>
      expect(
        screen.getByRole("dialog", { name: /billing-modal/ }),
      ).toBeInTheDocument(),
    )
  })

  it("pre-flights the billing-profile gate from the cached completeness", () => {
    mockCompleteness.mockReturnValue({
      isComplete: false,
      missingFields: [{ field: "vat", reason: "missing" }],
      isLoading: false,
      isError: false,
    })

    render(<WalletUnifiedPage />)
    fireEvent.click(screen.getByTestId("wallet-unified-withdraw"))
    expect(mockWithdrawMutate).not.toHaveBeenCalled()
    expect(
      screen.getByRole("dialog", { name: /billing-modal/ }),
    ).toBeInTheDocument()
  })

  it("renders the skeleton while the summary is loading", () => {
    mockSummary.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    })
    const { container } = render(<WalletUnifiedPage />)
    // Skeleton renders shimmer placeholders — no withdraw button.
    expect(
      container.querySelector('[data-testid="wallet-unified-withdraw"]'),
    ).toBeNull()
  })

  // ─── Bug 1 regression — defensive WithdrawResult consumption ────────
  // The backend `withdrawResponse.Errors` field is `omitempty`. A clean
  // 200 success returns JSON without an `errors` key, so the old
  // `result.errors.length` access threw a TypeError and surfaced the
  // Next.js error overlay on a money-moving page. Every response shape
  // below MUST render without throwing.

  it("does NOT crash and shows a success toast when the response omits `errors`", async () => {
    // Simulates the real backend: `errors` is `omitempty` and absent
    // on a clean 200 success.
    mockWithdrawMutate.mockImplementation((_amt, opts) => {
      opts?.onSuccess?.({
        drained_cents: 100_00,
        missions_cents: 100_00,
        commissions_cents: 0,
        stripe_transfer_ids: ["tr_1"],
        currency: "EUR",
        // `errors` key intentionally absent — matches the wire shape.
      })
    })

    render(<WalletUnifiedPage />)
    expect(() =>
      fireEvent.click(screen.getByTestId("wallet-unified-withdraw")),
    ).not.toThrow()
    await waitFor(() =>
      expect(mockToast).toHaveBeenCalledWith("walletUnified.toast.success"),
    )
    expect(
      screen.queryByRole("dialog", { name: /result-modal/ }),
    ).not.toBeInTheDocument()
  })

  it("does NOT crash and surfaces the defensive toast on an empty 200 body", async () => {
    // Worst case: the backend returns {} (no fields at all). The UI
    // must stay alive and surface a benign "retrait en cours" message
    // so the user knows something happened — the wallet cache
    // invalidation will refresh the truth.
    mockWithdrawMutate.mockImplementation((_amt, opts) => {
      opts?.onSuccess?.({})
    })

    render(<WalletUnifiedPage />)
    expect(() =>
      fireEvent.click(screen.getByTestId("wallet-unified-withdraw")),
    ).not.toThrow()
    await waitFor(() =>
      expect(mockToast).toHaveBeenCalledWith("walletUnified.toast.unknown"),
    )
    expect(
      screen.queryByRole("dialog", { name: /result-modal/ }),
    ).not.toBeInTheDocument()
  })

  it("does NOT crash when both `errors` and `drained_cents` are absent", async () => {
    // Defensive matrix: nothing in the body except `currency`.
    // Mirrors a degenerate backend response. UI must still surface
    // the defensive toast.
    mockWithdrawMutate.mockImplementation((_amt, opts) => {
      opts?.onSuccess?.({ currency: "EUR" })
    })

    render(<WalletUnifiedPage />)
    expect(() =>
      fireEvent.click(screen.getByTestId("wallet-unified-withdraw")),
    ).not.toThrow()
    await waitFor(() =>
      expect(mockToast).toHaveBeenCalledWith("walletUnified.toast.unknown"),
    )
  })
})
