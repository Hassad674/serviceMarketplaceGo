/**
 * proposal-actions-api.test.ts
 *
 * Unit tests for the shared accept/decline proposal API. These two
 * endpoints are exercised both from the messaging feature (in the
 * proposal-card actions) and from the proposal feature itself, so the
 * shared module is a hot rewire target for the F.3.2 sweep — pinning
 * the path + method + URL-encoding behaviour locks down the contract.
 */
import { describe, it, expect, vi, beforeEach } from "vitest"
import { acceptProposal, declineProposal } from "../proposal-actions-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue(undefined)
})

describe("acceptProposal", () => {
  it("POSTs /api/v1/proposals/:id/accept with the right id", async () => {
    await acceptProposal("p-123")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-123/accept",
      { method: "POST" },
    )
  })

  it("does NOT URL-encode the id (caller must pass a clean id)", async () => {
    // FLAGGED-FOR-FOLLOWUP: the id is interpolated raw — if a caller
    // ever passes an id containing reserved chars the URL would be
    // malformed. Lock current behaviour so the F.3.2 sweep cannot
    // silently change it.
    await acceptProposal("p/escape")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p/escape/accept",
      { method: "POST" },
    )
  })

  it("propagates apiClient errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("boom"))
    await expect(acceptProposal("p-1")).rejects.toThrow("boom")
  })
})

describe("declineProposal", () => {
  it("POSTs /api/v1/proposals/:id/decline with the right id", async () => {
    await declineProposal("p-456")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-456/decline",
      { method: "POST" },
    )
  })

  it("returns void on success", async () => {
    mockApiClient.mockResolvedValueOnce(undefined)
    const result = await declineProposal("p-1")
    expect(result).toBeUndefined()
  })

  it("propagates apiClient errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("forbidden"))
    await expect(declineProposal("p-1")).rejects.toThrow("forbidden")
  })
})
