import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  openDispute,
  getDispute,
  counterPropose,
  respondToCounter,
  cancelDispute,
  respondToCancellation,
} from "../dispute-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...args: unknown[]) => mockApiClient(...args),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({ id: "d-1" })
})

describe("dispute-api", () => {
  it("openDispute POSTs to /api/v1/disputes with the payload", () => {
    const payload = {
      proposal_id: "p-1",
      reason: "non_delivery",
      description: "Nothing delivered",
      message_to_party: "Please deliver",
      requested_amount: 50000,
    }
    openDispute(payload)
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/disputes", {
      method: "POST",
      body: payload,
    })
  })

  it("openDispute supports optional attachments", () => {
    openDispute({
      proposal_id: "p-1",
      reason: "harassment",
      description: "",
      message_to_party: "stop",
      requested_amount: 10000,
      attachments: [
        { filename: "x.pdf", url: "https://x", size: 1, mime_type: "application/pdf" },
      ],
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { attachments: unknown[] } }).body
    expect(body.attachments).toHaveLength(1)
  })

  it("getDispute issues a GET on /api/v1/disputes/{id}", () => {
    getDispute("d-x")
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/disputes/d-x")
  })

  it("counterPropose POSTs the body to /counter-propose", () => {
    counterPropose("d-1", { amount_client: 100, amount_provider: 200, message: "split" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/disputes/d-1/counter-propose",
      {
        method: "POST",
        body: { amount_client: 100, amount_provider: 200, message: "split" },
      },
    )
  })

  it("respondToCounter sends accept=true", () => {
    respondToCounter("d-1", "cp-9", true)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/disputes/d-1/counter-proposals/cp-9/respond",
      { method: "POST", body: { accept: true } },
    )
  })

  it("respondToCounter sends accept=false", () => {
    respondToCounter("d-1", "cp-9", false)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/disputes/d-1/counter-proposals/cp-9/respond",
      { method: "POST", body: { accept: false } },
    )
  })

  it("cancelDispute POSTs the cancel endpoint", () => {
    cancelDispute("d-1")
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/disputes/d-1/cancel", {
      method: "POST",
    })
  })

  it("respondToCancellation POSTs the response", () => {
    respondToCancellation("d-1", true)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/disputes/d-1/cancellation/respond",
      { method: "POST", body: { accept: true } },
    )
  })

  it("respondToCancellation accepts false too", () => {
    respondToCancellation("d-1", false)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/disputes/d-1/cancellation/respond",
      { method: "POST", body: { accept: false } },
    )
  })

  it("propagates apiClient errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("nope"))
    await expect(getDispute("z")).rejects.toThrow("nope")
  })
})
