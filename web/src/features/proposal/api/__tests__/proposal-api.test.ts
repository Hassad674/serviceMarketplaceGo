import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  createProposal,
  getProposal,
  acceptProposal,
  declineProposal,
  modifyProposal,
  initiatePayment,
  confirmPayment,
  listProjects,
  getUploadURL,
  fundMilestone,
  submitMilestone,
  approveMilestone,
  rejectMilestone,
  cancelProposal,
} from "../proposal-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("proposal-api / createProposal", () => {
  it("POSTs the body to /api/v1/proposals", async () => {
    await createProposal({
      recipient_id: "u-2",
      conversation_id: "c-1",
      title: "Build site",
      description: "...",
      amount: 100000,
    })
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/proposals", {
      method: "POST",
      body: expect.objectContaining({
        recipient_id: "u-2",
        amount: 100000,
      }),
    })
  })

  it("supports milestone payment mode", async () => {
    await createProposal({
      recipient_id: "u-2",
      conversation_id: "c-1",
      title: "Build site",
      description: "...",
      amount: 100000,
      payment_mode: "milestone",
      milestones: [
        { sequence: 1, title: "M1", description: "first", amount: 50000 },
        { sequence: 2, title: "M2", description: "second", amount: 50000 },
      ],
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { milestones: unknown[] } }).body
    expect(body.milestones).toHaveLength(2)
  })
})

describe("proposal-api / accept/decline/cancel", () => {
  it("getProposal GETs by id", () => {
    getProposal("p-1")
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/proposals/p-1")
  })

  it("acceptProposal POSTs the accept endpoint", () => {
    acceptProposal("p-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/accept",
      { method: "POST" },
    )
  })

  it("declineProposal POSTs the decline endpoint", () => {
    declineProposal("p-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/decline",
      { method: "POST" },
    )
  })

  it("cancelProposal POSTs the cancel endpoint", () => {
    cancelProposal("p-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/cancel",
      { method: "POST" },
    )
  })
})

describe("proposal-api / payment", () => {
  it("initiatePayment POSTs /pay", () => {
    initiatePayment("p-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/pay",
      { method: "POST" },
    )
  })

  it("confirmPayment POSTs /confirm-payment", () => {
    confirmPayment("p-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/confirm-payment",
      { method: "POST" },
    )
  })
})

describe("proposal-api / modifyProposal", () => {
  it("POSTs /modify with the new payload", async () => {
    await modifyProposal("p-1", {
      title: "Updated",
      description: "..",
      amount: 200000,
    })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/modify",
      {
        method: "POST",
        body: expect.objectContaining({ title: "Updated" }),
      },
    )
  })
})

describe("proposal-api / listProjects", () => {
  it("calls without cursor when none provided", () => {
    listProjects()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/projects")
  })

  it("appends URL-encoded cursor when provided", () => {
    listProjects("abc/def")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/projects?cursor=abc%2Fdef",
    )
  })
})

describe("proposal-api / getUploadURL", () => {
  it("POSTs filename and content_type", () => {
    getUploadURL("doc.pdf", "application/pdf")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/upload-url",
      {
        method: "POST",
        body: { filename: "doc.pdf", content_type: "application/pdf" },
      },
    )
  })
})

describe("proposal-api / milestone endpoints", () => {
  it("fundMilestone hits the correct path", () => {
    fundMilestone("p-1", "m-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/milestones/m-1/fund",
      { method: "POST" },
    )
  })

  it("submitMilestone hits the correct path", () => {
    submitMilestone("p-1", "m-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/milestones/m-1/submit",
      { method: "POST" },
    )
  })

  it("approveMilestone hits the correct path", () => {
    approveMilestone("p-1", "m-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/milestones/m-1/approve",
      { method: "POST" },
    )
  })

  it("rejectMilestone hits the correct path", () => {
    rejectMilestone("p-1", "m-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/proposals/p-1/milestones/m-1/reject",
      { method: "POST" },
    )
  })
})

describe("proposal-api / errors", () => {
  it("propagates network errors from createProposal", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("network"))
    await expect(
      createProposal({
        recipient_id: "u",
        conversation_id: "c",
        title: "t",
        description: "d",
        amount: 1,
      }),
    ).rejects.toThrow("network")
  })
})
