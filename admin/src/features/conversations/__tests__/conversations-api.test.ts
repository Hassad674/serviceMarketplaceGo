import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listConversations,
  getConversation,
  getConversationMessages,
} from "../api/conversations-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockAdminApi.mockResolvedValue({})
})

describe("admin conversations-api", () => {
  it("listConversations calls /admin/conversations with limit=20", () => {
    listConversations({})
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/conversations?limit=20",
    )
  })

  it("listConversations appends page/sort/filter", () => {
    listConversations({ page: 3, sort: "newest", filter: "flagged" })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("page=3")
    expect(call).toContain("sort=newest")
    expect(call).toContain("filter=flagged")
  })

  it("listConversations does not send page=0", () => {
    listConversations({ page: 0 })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).not.toContain("page=")
  })

  it("getConversation GETs by id", () => {
    getConversation("c-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/conversations/c-1",
    )
  })

  it("getConversationMessages GETs without cursor", () => {
    getConversationMessages("c-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/conversations/c-1/messages?limit=50",
    )
  })

  it("getConversationMessages appends cursor", () => {
    getConversationMessages("c-1", "tok")
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("cursor=tok")
    expect(call).toContain("limit=50")
  })

  it("propagates errors", async () => {
    mockAdminApi.mockRejectedValueOnce(new Error("404"))
    await expect(getConversation("missing")).rejects.toThrow("404")
  })
})
