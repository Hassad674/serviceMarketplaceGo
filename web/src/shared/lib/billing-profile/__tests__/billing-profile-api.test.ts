/**
 * billing-profile-api.test.ts
 *
 * Unit tests for the shared billing-profile API client. Verifies each
 * function hits the documented path with the right method and body.
 */
import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  fetchBillingProfile,
  updateBillingProfile,
  syncBillingProfileFromStripe,
  validateBillingProfileVAT,
  fetchCurrentMonthAggregate,
} from "../billing-profile-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("billing-profile-api", () => {
  it("fetchBillingProfile GETs /api/v1/me/billing-profile", () => {
    fetchBillingProfile()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/me/billing-profile")
  })

  it("updateBillingProfile PUTs the input as the body", async () => {
    const input = {
      profile_type: "business" as const,
      legal_name: "Acme",
      trading_name: "",
      legal_form: "SAS",
      tax_id: "",
      vat_number: "FR12345678901",
      address_line1: "1 rue de la Paix",
      address_line2: "",
      postal_code: "75002",
      city: "Paris",
      country: "FR",
      invoicing_email: "billing@acme.fr",
    }
    await updateBillingProfile(input)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/me/billing-profile",
      { method: "PUT", body: input },
    )
  })

  it("syncBillingProfileFromStripe POSTs the sync endpoint", () => {
    syncBillingProfileFromStripe()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/me/billing-profile/sync-from-stripe",
      { method: "POST" },
    )
  })

  it("validateBillingProfileVAT POSTs the validate-vat endpoint", () => {
    validateBillingProfileVAT()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/me/billing-profile/validate-vat",
      { method: "POST" },
    )
  })

  it("fetchCurrentMonthAggregate GETs the running-month aggregate", () => {
    fetchCurrentMonthAggregate()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/me/invoicing/current-month",
    )
  })

  it("propagates apiClient errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("net"))
    await expect(fetchBillingProfile()).rejects.toThrow("net")
  })

  it("returns the parsed BillingProfileSnapshot body verbatim", async () => {
    const snapshot = { profile: {}, missing_fields: [], is_complete: true }
    mockApiClient.mockResolvedValueOnce(snapshot)
    const res = await fetchBillingProfile()
    expect(res).toEqual(snapshot)
  })
})
