import { describe, it, expect, vi, beforeEach } from "vitest"
import { createReport } from "../reporting-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...args: unknown[]) => mockApiClient(...args),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({ id: "r-1" })
})

describe("createReport", () => {
  it("POSTs to /api/v1/reports with the body", async () => {
    await createReport({
      target_type: "user",
      target_id: "u-1",
      conversation_id: "c-1",
      reason: "harassment",
      description: "very mean",
    })
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/reports", {
      method: "POST",
      body: {
        target_type: "user",
        target_id: "u-1",
        conversation_id: "c-1",
        reason: "harassment",
        description: "very mean",
      },
    })
  })

  it("supports an empty description", async () => {
    await createReport({
      target_type: "message",
      target_id: "m-1",
      conversation_id: "c-1",
      reason: "spam",
      description: "",
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { description: string } }).body
    expect(body.description).toBe("")
  })

  it("supports an empty conversation_id (global reports)", async () => {
    await createReport({
      target_type: "job",
      target_id: "j-1",
      conversation_id: "",
      reason: "fraud_or_scam",
      description: "scam",
    })
    expect(mockApiClient).toHaveBeenCalledOnce()
  })

  it("propagates api errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("boom"))
    await expect(
      createReport({
        target_type: "user",
        target_id: "u-1",
        conversation_id: "c-1",
        reason: "spam",
        description: "x",
      }),
    ).rejects.toThrow("boom")
  })

  it("works for application target type", async () => {
    await createReport({
      target_type: "application",
      target_id: "a-1",
      conversation_id: "c-1",
      reason: "fraud_or_scam",
      description: "fake CV",
    })
    expect(mockApiClient).toHaveBeenCalledOnce()
  })
})
