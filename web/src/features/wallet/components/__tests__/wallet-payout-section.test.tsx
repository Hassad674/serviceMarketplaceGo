import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { ApiError } from "@/shared/lib/api-client"
import {
  WalletPayoutSection,
  extractMessage,
  extractMissingFields,
} from "../wallet-payout-section"

// next/link stub
vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...rest
  }: {
    children: React.ReactNode
    href: string
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

// next/navigation stub for KYCIncompleteModal
const mockPush = vi.fn()
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}))

// Permission hook — flip via mockReturnValue inside tests when needed
const mockHasPermission = vi.fn((_perm: string) => true)
vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: (perm: string) => mockHasPermission(perm),
}))

// Billing-profile completeness — full mock so tests stay deterministic
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
// `useBillingProfileCompleteness` moved to shared (P9). wallet-payout-section
// now imports the shared path.
vi.mock("@/shared/hooks/billing-profile/use-billing-profile-completeness", () => ({
  useBillingProfileCompleteness: () => mockCompleteness(),
}))

// Stub the BillingProfileCompletionModal to render an aria-labeled
// dialog only when open — keeps the assertions simple.
vi.mock("@/shared/components/billing-profile/billing-profile-completion-modal", () => ({
  BillingProfileCompletionModal: ({
    open,
    onClose,
    missingFields,
  }: {
    open: boolean
    onClose: () => void
    missingFields: { field: string; reason: string }[]
  }) =>
    open ? (
      <div role="dialog" aria-label="billing-modal">
        billing-incomplete missing={missingFields.length}
        <button onClick={onClose}>close-billing</button>
      </div>
    ) : null,
}))

// Wallet hooks — useRequestPayout is the unit under test for the modal
// flow. The mutation is fully driven by what we plug in here.
const mockMutate = vi.fn()
const mockRequestPayoutResult = {
  mutate: mockMutate,
  isPending: false,
  isError: false,
  error: null as Error | null,
}
vi.mock("../../hooks/use-wallet", () => ({
  useRequestPayout: () => mockRequestPayoutResult,
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockHasPermission.mockReturnValue(true)
  mockCompleteness.mockReturnValue({
    isComplete: true,
    missingFields: [],
    isLoading: false,
    isError: false,
  })
  mockRequestPayoutResult.isPending = false
  mockRequestPayoutResult.isError = false
  mockRequestPayoutResult.error = null
})

function defaultProps(overrides: Partial<Parameters<typeof WalletPayoutSection>[0]> = {}) {
  return {
    totalEarned: 1000_00,
    available: 500_00,
    stripeAccountId: "acct_x",
    payoutsEnabled: true,
    ...overrides,
  }
}

describe("WalletPayoutSection — render", () => {
  it("renders the overview card", () => {
    render(<WalletPayoutSection {...defaultProps()} />)
    expect(
      screen.getByRole("heading", { name: /Portefeuille/i }),
    ).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Retirer/i })).toBeInTheDocument()
  })

  it("does not render any modal by default", () => {
    render(<WalletPayoutSection {...defaultProps()} />)
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  })
})

describe("WalletPayoutSection — KYC pre-flight", () => {
  it("opens the KYC modal when payouts_enabled is false (skips request)", () => {
    render(<WalletPayoutSection {...defaultProps({ payoutsEnabled: false })} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    expect(mockMutate).not.toHaveBeenCalled()
    expect(
      screen.getByRole("heading", {
        name: /Termine ton onboarding Stripe/i,
      }),
    ).toBeInTheDocument()
  })

  it("opens the KYC modal on a 403 kyc_incomplete response", async () => {
    mockMutate.mockImplementation((_input, opts) => {
      const err = new ApiError(403, "kyc_incomplete", "kyc not done", {
        error: { code: "kyc_incomplete", message: "Server says: complete KYC" },
      })
      opts?.onError?.(err)
    })

    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))

    expect(mockMutate).toHaveBeenCalledOnce()
    await waitFor(() =>
      expect(
        screen.getByText(/Server says: complete KYC/),
      ).toBeInTheDocument(),
    )
  })
})

describe("WalletPayoutSection — billing-profile pre-flight", () => {
  it("opens the billing modal when completeness reports incomplete", () => {
    mockCompleteness.mockReturnValue({
      isComplete: false,
      missingFields: [{ field: "tax_id", reason: "required" }],
      isLoading: false,
      isError: false,
    })
    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    expect(mockMutate).not.toHaveBeenCalled()
    expect(
      screen.getByRole("dialog", { name: "billing-modal" }),
    ).toBeInTheDocument()
    expect(screen.getByText(/missing=1/)).toBeInTheDocument()
  })

  it("opens the billing modal on 403 billing_profile_incomplete using server-provided fields", async () => {
    mockMutate.mockImplementation((_input, opts) => {
      const err = new ApiError(
        403,
        "billing_profile_incomplete",
        "incomplete",
        {
          missing_fields: [
            { field: "siret", reason: "required" },
            { field: "address_line1", reason: "required" },
          ],
        },
      )
      opts?.onError?.(err)
    })
    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    await waitFor(() =>
      expect(
        screen.getByRole("dialog", { name: "billing-modal" }),
      ).toBeInTheDocument(),
    )
    // Server sent 2 fields → modal mock prints "missing=2"
    expect(screen.getByText(/missing=2/)).toBeInTheDocument()
  })

  it("falls back to cached missing fields when server envelope omits them", async () => {
    mockCompleteness.mockReturnValue({
      isComplete: true, // cached says ok, but server disagrees
      missingFields: [{ field: "vat_number", reason: "required" }],
      isLoading: false,
      isError: false,
    })
    mockMutate.mockImplementation((_input, opts) => {
      const err = new ApiError(
        403,
        "billing_profile_incomplete",
        "incomplete",
        // No missing_fields key in body
        { error: { code: "billing_profile_incomplete", message: "x" } },
      )
      opts?.onError?.(err)
    })
    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    await waitFor(() =>
      expect(
        screen.getByRole("dialog", { name: "billing-modal" }),
      ).toBeInTheDocument(),
    )
    expect(screen.getByText(/missing=1/)).toBeInTheDocument()
  })

  it("ignores non-403 errors silently (button stays available)", () => {
    mockMutate.mockImplementation((_input, opts) => {
      const err = new ApiError(500, "internal_error", "x")
      opts?.onError?.(err)
    })
    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  })
})

describe("WalletPayoutSection — happy path", () => {
  it("calls mutate when KYC + billing pass and surfaces the success message", async () => {
    mockMutate.mockImplementation((_input, opts) => {
      opts?.onSuccess?.({ status: "ok", message: "Virement initié" })
    })
    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    expect(mockMutate).toHaveBeenCalledOnce()
    expect(await screen.findByText(/Virement initié/)).toBeInTheDocument()
  })
})

describe("WalletPayoutSection — modal close callbacks", () => {
  it("dismisses the billing modal when the close button is clicked", () => {
    mockCompleteness.mockReturnValue({
      isComplete: false,
      missingFields: [{ field: "tax_id", reason: "required" }],
      isLoading: false,
      isError: false,
    })
    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    expect(
      screen.getByRole("dialog", { name: "billing-modal" }),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: /close-billing/i }))
    expect(
      screen.queryByRole("dialog", { name: "billing-modal" }),
    ).not.toBeInTheDocument()
  })

  it("dismisses the KYC modal when the secondary 'Plus tard' is clicked", () => {
    render(<WalletPayoutSection {...defaultProps({ payoutsEnabled: false })} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    expect(
      screen.getByRole("heading", { name: /Termine ton onboarding Stripe/i }),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: /Plus tard/i }))
    expect(
      screen.queryByRole("heading", { name: /Termine ton onboarding Stripe/i }),
    ).not.toBeInTheDocument()
  })

  it("waits before opening billing modal while completeness is still loading", () => {
    mockCompleteness.mockReturnValue({
      isComplete: false,
      missingFields: [],
      isLoading: true,
      isError: false,
    })
    mockMutate.mockImplementation(() => {
      // No-op — should not be called either, because the parent skips
      // mutate during the loading window. Actually the source calls
      // mutate when not loading and not complete; here we are loading
      // so mutate IS called (the gate only opens when loading is done).
    })
    render(<WalletPayoutSection {...defaultProps()} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    // Loading branch: gate skipped, mutate called
    expect(mockMutate).toHaveBeenCalledOnce()
  })
})

// ---------------------------------------------------------------------------
// Pure parser helpers
// ---------------------------------------------------------------------------

describe("extractMessage", () => {
  it("returns the nested error.message", () => {
    expect(extractMessage({ error: { message: "hi" } })).toBe("hi")
  })

  it("returns the top-level message when no nested error", () => {
    expect(extractMessage({ message: "top" })).toBe("top")
  })

  it("returns undefined for malformed payloads", () => {
    expect(extractMessage(null)).toBeUndefined()
    expect(extractMessage(undefined)).toBeUndefined()
    expect(extractMessage("string")).toBeUndefined()
    expect(extractMessage({ message: "" })).toBeUndefined()
    expect(extractMessage({ error: 42 })).toBeUndefined()
  })
})

describe("extractMissingFields", () => {
  it("returns parsed entries for a well-formed envelope", () => {
    expect(
      extractMissingFields({
        missing_fields: [
          { field: "tax_id", reason: "required" },
          { field: "address", reason: "required" },
        ],
      }),
    ).toEqual([
      { field: "tax_id", reason: "required" },
      { field: "address", reason: "required" },
    ])
  })

  it("returns an empty array when missing_fields is absent or wrong shape", () => {
    expect(extractMissingFields(null)).toEqual([])
    expect(extractMissingFields({ missing_fields: "nope" })).toEqual([])
    expect(extractMissingFields({})).toEqual([])
  })

  it("filters out malformed entries", () => {
    expect(
      extractMissingFields({
        missing_fields: [
          { field: "ok", reason: "required" },
          { field: 1, reason: "required" },
          { reason: "required" },
          null,
        ],
      }),
    ).toEqual([{ field: "ok", reason: "required" }])
  })
})
