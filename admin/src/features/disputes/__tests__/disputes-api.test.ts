import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listDisputes,
  getDispute,
  resolveDispute,
  countDisputes,
  forceEscalateDispute,
  askAIDispute,
  increaseAIBudget,
} from "../api/disputes-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockAdminApi.mockResolvedValue({})
})

describe("admin disputes api", () => {
  it("listDisputes appends limit=20", () => {
    listDisputes({ status: "", cursor: "" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/disputes?limit=20",
    )
  })

  it("listDisputes appends status when present", () => {
    listDisputes({ status: "open", cursor: "" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/disputes?status=open&limit=20",
    )
  })

  it("listDisputes appends cursor when present", () => {
    listDisputes({ status: "", cursor: "tok" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/disputes?cursor=tok&limit=20",
    )
  })

  it("getDispute GETs by id", () => {
    getDispute("d-1")
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/disputes/d-1")
  })

  it("resolveDispute POSTs with the body", () => {
    resolveDispute("d-1", {
      amount_client: 5000,
      amount_provider: 5000,
      note: "split 50/50",
    })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/disputes/d-1/resolve",
      {
        method: "POST",
        body: {
          amount_client: 5000,
          amount_provider: 5000,
          note: "split 50/50",
        },
      },
    )
  })

  it("countDisputes GETs the count endpoint", () => {
    countDisputes()
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/disputes/count")
  })

  it("forceEscalateDispute POSTs the force-escalate endpoint", () => {
    forceEscalateDispute("d-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/disputes/d-1/force-escalate",
      { method: "POST" },
    )
  })

  it("askAIDispute POSTs the question to /ai-chat", () => {
    askAIDispute("d-1", "what should I do?")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/disputes/d-1/ai-chat",
      { method: "POST", body: { question: "what should I do?" } },
    )
  })

  it("increaseAIBudget POSTs to the budget endpoint", () => {
    increaseAIBudget("d-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/disputes/d-1/ai-budget",
      { method: "POST" },
    )
  })

  it("propagates errors", async () => {
    mockAdminApi.mockRejectedValueOnce(new Error("403"))
    await expect(getDispute("d-1")).rejects.toThrow("403")
  })
})
