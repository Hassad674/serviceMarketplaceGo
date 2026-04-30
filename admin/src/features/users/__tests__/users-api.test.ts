import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listUsers,
  getUser,
  suspendUser,
  unsuspendUser,
  banUser,
  unbanUser,
  getUserOrganization,
  forceTransferOwnership,
  forceUpdateMemberRole,
  forceRemoveMember,
  forceCancelInvitation,
} from "../api/users-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockAdminApi.mockResolvedValue({})
})

describe("admin users-api / listUsers", () => {
  it("calls /admin/users with limit=20 by default", () => {
    listUsers({ role: "", status: "", search: "", page: 0, reported: false })
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/users?limit=20")
  })

  it("appends role filter", () => {
    listUsers({ role: "agency", status: "", search: "", page: 0, reported: false })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("role=agency")
  })

  it("appends status filter", () => {
    listUsers({ role: "", status: "active", search: "", page: 0, reported: false })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("status=active")
  })

  it("appends search filter", () => {
    listUsers({ role: "", status: "", search: "joe", page: 0, reported: false })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("search=joe")
  })

  it("appends page when > 0", () => {
    listUsers({ role: "", status: "", search: "", page: 3, reported: false })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("page=3")
  })

  it("does not append page when 0", () => {
    listUsers({ role: "", status: "", search: "", page: 0, reported: false })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).not.toContain("page=")
  })

  it("appends reported=true when reported=true", () => {
    listUsers({ role: "", status: "", search: "", page: 0, reported: true })
    const call = mockAdminApi.mock.calls[0][0] as string
    expect(call).toContain("reported=true")
  })
})

describe("admin users-api / single user actions", () => {
  it("getUser GETs by id", () => {
    getUser("u-1")
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/users/u-1")
  })

  it("suspendUser POSTs the payload", () => {
    suspendUser("u-1", { reason: "spam", expires_at: "2026-12-01" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/users/u-1/suspend",
      { method: "POST", body: { reason: "spam", expires_at: "2026-12-01" } },
    )
  })

  it("unsuspendUser POSTs with empty body", () => {
    unsuspendUser("u-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/users/u-1/unsuspend",
      { method: "POST", body: {} },
    )
  })

  it("banUser POSTs the reason", () => {
    banUser("u-1", { reason: "fraud" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/users/u-1/ban",
      { method: "POST", body: { reason: "fraud" } },
    )
  })

  it("unbanUser POSTs with empty body", () => {
    unbanUser("u-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/users/u-1/unban",
      { method: "POST", body: {} },
    )
  })
})

describe("admin users-api / team management", () => {
  it("getUserOrganization GETs scoped to user", () => {
    getUserOrganization("u-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/users/u-1/organization",
    )
  })

  it("forceTransferOwnership POSTs to the org transfer endpoint", () => {
    forceTransferOwnership("org-1", { target_user_id: "u-2" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/organizations/org-1/force-transfer",
      { method: "POST", body: { target_user_id: "u-2" } },
    )
  })

  it("forceUpdateMemberRole PATCHes the member endpoint", () => {
    forceUpdateMemberRole("org-1", "u-2", { role: "admin" })
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/organizations/org-1/members/u-2",
      { method: "PATCH", body: { role: "admin" } },
    )
  })

  it("forceRemoveMember DELETEs the member", () => {
    forceRemoveMember("org-1", "u-2")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/organizations/org-1/members/u-2",
      { method: "DELETE" },
    )
  })

  it("forceCancelInvitation DELETEs the invitation", () => {
    forceCancelInvitation("org-1", "inv-1")
    expect(mockAdminApi).toHaveBeenCalledWith(
      "/api/v1/admin/organizations/org-1/invitations/inv-1",
      { method: "DELETE" },
    )
  })
})

describe("admin users-api / errors", () => {
  it("propagates errors from listUsers", async () => {
    mockAdminApi.mockRejectedValueOnce(new Error("500"))
    await expect(
      listUsers({ role: "", status: "", search: "", page: 0, reported: false }),
    ).rejects.toThrow("500")
  })
})
