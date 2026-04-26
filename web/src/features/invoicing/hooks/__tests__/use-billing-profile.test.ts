import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useBillingProfile,
} from "../use-billing-profile"
import { useBillingProfileCompleteness } from "../use-billing-profile-completeness"
import type { BillingProfileSnapshot } from "../../types"

const mockFetch = vi.fn()

vi.mock("../../api/invoicing-api", () => ({
  fetchBillingProfile: () => mockFetch(),
  updateBillingProfile: vi.fn(),
  syncBillingProfileFromStripe: vi.fn(),
  validateBillingProfileVAT: vi.fn(),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

const COMPLETE_SNAPSHOT: BillingProfileSnapshot = {
  profile: {
    organization_id: "org-1",
    profile_type: "business",
    legal_name: "Acme SAS",
    trading_name: "",
    legal_form: "SAS",
    tax_id: "12345678901234",
    vat_number: "FR12345678901",
    vat_validated_at: "2026-04-01T10:00:00Z",
    address_line1: "1 rue de la Paix",
    address_line2: "",
    postal_code: "75001",
    city: "Paris",
    country: "FR",
    invoicing_email: "billing@acme.com",
    synced_from_kyc_at: "2026-03-30T10:00:00Z",
  },
  missing_fields: [],
  is_complete: true,
}

const INCOMPLETE_SNAPSHOT: BillingProfileSnapshot = {
  ...COMPLETE_SNAPSHOT,
  profile: { ...COMPLETE_SNAPSHOT.profile, legal_name: "" },
  missing_fields: [{ field: "legal_name", reason: "required" }],
  is_complete: false,
}

describe("useBillingProfile", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("fetches and returns the snapshot on mount", async () => {
    mockFetch.mockResolvedValue(COMPLETE_SNAPSHOT)
    const { result } = renderHook(() => useBillingProfile(), {
      wrapper: createWrapper(),
    })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.is_complete).toBe(true)
    expect(mockFetch).toHaveBeenCalledOnce()
  })

  it("surfaces the loading state before resolution", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}))
    const { result } = renderHook(() => useBillingProfile(), {
      wrapper: createWrapper(),
    })
    expect(result.current.isLoading).toBe(true)
    expect(result.current.data).toBeUndefined()
  })

  it("propagates errors", async () => {
    mockFetch.mockRejectedValue(new Error("boom"))
    const { result } = renderHook(() => useBillingProfile(), {
      wrapper: createWrapper(),
    })
    // The hook uses `retry: 1` so a single failure does not flip the
    // status — wait long enough for both attempts to fail before
    // asserting. `waitFor` runs the callback repeatedly and times
    // out at 4500ms (well above TanStack Query's default retry delay).
    await waitFor(() => expect(result.current.isError).toBe(true), {
      timeout: 4500,
    })
  })
})

describe("useBillingProfileCompleteness", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("reports the gate as open when the snapshot is complete", async () => {
    mockFetch.mockResolvedValue(COMPLETE_SNAPSHOT)
    const { result } = renderHook(() => useBillingProfileCompleteness(), {
      wrapper: createWrapper(),
    })
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.isComplete).toBe(true)
    expect(result.current.missingFields).toEqual([])
  })

  it("exposes the missing fields when incomplete", async () => {
    mockFetch.mockResolvedValue(INCOMPLETE_SNAPSHOT)
    const { result } = renderHook(() => useBillingProfileCompleteness(), {
      wrapper: createWrapper(),
    })
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.isComplete).toBe(false)
    expect(result.current.missingFields).toHaveLength(1)
    expect(result.current.missingFields[0].field).toBe("legal_name")
  })

  it("returns a loading state while the cache is empty", () => {
    mockFetch.mockImplementation(() => new Promise(() => {}))
    const { result } = renderHook(() => useBillingProfileCompleteness(), {
      wrapper: createWrapper(),
    })
    expect(result.current.isLoading).toBe(true)
    expect(result.current.isComplete).toBe(false)
  })
})
