import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  usePaymentInfo,
  usePaymentInfoStatus,
  useSavePaymentInfo,
} from "../use-payment-info"

const mockGetPaymentInfo = vi.fn()
const mockSavePaymentInfo = vi.fn()
const mockGetPaymentInfoStatus = vi.fn()

vi.mock("../../api/payment-info-api", () => ({
  getPaymentInfo: (...args: unknown[]) => mockGetPaymentInfo(...args),
  savePaymentInfo: (...args: unknown[]) => mockSavePaymentInfo(...args),
  getPaymentInfoStatus: (...args: unknown[]) =>
    mockGetPaymentInfoStatus(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
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

describe("usePaymentInfo", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls getPaymentInfo API on mount", async () => {
    mockGetPaymentInfo.mockResolvedValue({
      id: "pi-1",
      user_id: "test-user-id",
      first_name: "Alice",
      last_name: "Dupont",
      iban: "FR7630006000011234567890189",
      stripe_verified: false,
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-20T10:00:00Z",
    })

    const { result } = renderHook(() => usePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetPaymentInfo).toHaveBeenCalledOnce()
  })

  it("returns payment info data from API", async () => {
    mockGetPaymentInfo.mockResolvedValue({
      id: "pi-1",
      user_id: "test-user-id",
      first_name: "Alice",
      last_name: "Dupont",
      date_of_birth: "1990-05-15",
      nationality: "FR",
      address: "12 Rue de la Paix",
      city: "Paris",
      postal_code: "75002",
      is_business: false,
      business_name: "",
      iban: "FR7630006000011234567890189",
      bic: "BNPAFRPP",
      stripe_account_id: "acct_abc",
      stripe_verified: true,
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-20T10:00:00Z",
    })

    const { result } = renderHook(() => usePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.first_name).toBe("Alice")
    expect(result.current.data?.iban).toBe("FR7630006000011234567890189")
    expect(result.current.data?.stripe_verified).toBe(true)
  })

  it("handles null response for user with no payment info", async () => {
    mockGetPaymentInfo.mockResolvedValue(null)

    const { result } = renderHook(() => usePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toBeNull()
  })
})

describe("usePaymentInfoStatus", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls getPaymentInfoStatus API on mount", async () => {
    mockGetPaymentInfoStatus.mockResolvedValue({ complete: true })

    const { result } = renderHook(() => usePaymentInfoStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetPaymentInfoStatus).toHaveBeenCalledOnce()
  })

  it("returns complete status", async () => {
    mockGetPaymentInfoStatus.mockResolvedValue({ complete: true })

    const { result } = renderHook(() => usePaymentInfoStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.complete).toBe(true)
  })

  it("returns incomplete status", async () => {
    mockGetPaymentInfoStatus.mockResolvedValue({ complete: false })

    const { result } = renderHook(() => usePaymentInfoStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.complete).toBe(false)
  })
})

describe("useSavePaymentInfo", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls savePaymentInfo API on mutate", async () => {
    mockSavePaymentInfo.mockResolvedValue({
      id: "pi-1",
      user_id: "test-user-id",
      first_name: "Alice",
      last_name: "Dupont",
      stripe_verified: false,
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-25T14:00:00Z",
    })

    const { result } = renderHook(() => useSavePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        data: {
          firstName: "Alice",
          lastName: "Dupont",
          dateOfBirth: "1990-05-15",
          nationality: "FR",
          address: "12 Rue de la Paix",
          city: "Paris",
          postalCode: "75002",
          isBusiness: false,
          businessName: "",
          businessAddress: "",
          businessCity: "",
          businessPostalCode: "",
          businessCountry: "",
          taxId: "",
          vatNumber: "",
          businessRole: "",
          phone: "+33612345678",
          activitySector: "",
          isSelfRepresentative: true,
          isSelfDirector: false,
          noMajorOwners: false,
          isSelfExecutive: false,
          businessPersons: [],
          bankMode: "iban",
          iban: "FR7630006000011234567890189",
          bic: "BNPAFRPP",
          accountNumber: "",
          routingNumber: "",
          accountHolder: "Alice Dupont",
          bankCountry: "FR",
          country: "FR",
          values: {},
          extraFields: {},
        },
        email: "alice@example.com",
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockSavePaymentInfo).toHaveBeenCalledOnce()
  })

  it("handles save failure", async () => {
    mockSavePaymentInfo.mockRejectedValue(
      new Error("Stripe verification failed"),
    )

    const { result } = renderHook(() => useSavePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        data: {
          firstName: "Alice",
          lastName: "Dupont",
          dateOfBirth: "1990-05-15",
          nationality: "FR",
          address: "12 Rue de la Paix",
          city: "Paris",
          postalCode: "75002",
          isBusiness: false,
          businessName: "",
          businessAddress: "",
          businessCity: "",
          businessPostalCode: "",
          businessCountry: "",
          taxId: "",
          vatNumber: "",
          businessRole: "",
          phone: "+33612345678",
          activitySector: "",
          isSelfRepresentative: true,
          isSelfDirector: false,
          noMajorOwners: false,
          isSelfExecutive: false,
          businessPersons: [],
          bankMode: "iban",
          iban: "FR7630006000011234567890189",
          bic: "BNPAFRPP",
          accountNumber: "",
          routingNumber: "",
          accountHolder: "Alice Dupont",
          bankCountry: "FR",
          country: "FR",
          values: {},
          extraFields: {},
        },
      })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Stripe verification failed")
  })
})
