import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listMembers,
  updateMember,
  removeMember,
  leaveOrganization,
  getRoleDefinitions,
  getRolePermissionsMatrix,
  updateRolePermissions,
  listInvitations,
  sendInvitation,
  resendInvitation,
  cancelInvitation,
  initiateTransferOwnership,
  cancelTransferOwnership,
  acceptTransferOwnership,
  declineTransferOwnership,
  validateInvitation,
  acceptInvitation,
} from "../team-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("team-api / members", () => {
  it("listMembers GETs scoped to org with limit=100", () => {
    listMembers("org-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/members?limit=100",
    )
  })

  it("updateMember PATCHes the member endpoint", () => {
    updateMember("org-1", "u-1", { role: "admin" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/members/u-1",
      { method: "PATCH", body: { role: "admin" } },
    )
  })

  it("removeMember DELETEs the member endpoint", () => {
    removeMember("org-1", "u-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/members/u-1",
      { method: "DELETE" },
    )
  })

  it("leaveOrganization POSTs the leave endpoint with empty body", () => {
    leaveOrganization("org-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/leave",
      { method: "POST", body: {} },
    )
  })
})

describe("team-api / role definitions", () => {
  it("getRoleDefinitions GETs the static catalogue", () => {
    getRoleDefinitions()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/role-definitions",
    )
  })

  it("getRolePermissionsMatrix GETs the org matrix", () => {
    getRolePermissionsMatrix("org-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/role-permissions",
    )
  })

  it("updateRolePermissions PATCHes the matrix", () => {
    const payload = { role: "admin" as const, overrides: {} }
    updateRolePermissions("org-1", payload)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/role-permissions",
      { method: "PATCH", body: payload },
    )
  })
})

describe("team-api / invitations", () => {
  it("listInvitations GETs scoped to org", () => {
    listInvitations("org-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/invitations?limit=100",
    )
  })

  it("sendInvitation POSTs the payload", () => {
    sendInvitation("org-1", { email: "x@x.com", role: "member" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/invitations",
      { method: "POST", body: { email: "x@x.com", role: "member" } },
    )
  })

  it("resendInvitation POSTs /resend with empty body", () => {
    resendInvitation("org-1", "inv-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/invitations/inv-1/resend",
      { method: "POST", body: {} },
    )
  })

  it("cancelInvitation DELETEs the invitation", () => {
    cancelInvitation("org-1", "inv-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/invitations/inv-1",
      { method: "DELETE" },
    )
  })
})

describe("team-api / ownership transfer", () => {
  it("initiateTransferOwnership POSTs to /transfer", () => {
    initiateTransferOwnership("org-1", { successor_user_id: "u-2" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/transfer",
      { method: "POST", body: { successor_user_id: "u-2" } },
    )
  })

  it("cancelTransferOwnership DELETEs /transfer", () => {
    cancelTransferOwnership("org-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/transfer",
      { method: "DELETE" },
    )
  })

  it("acceptTransferOwnership POSTs /transfer/accept with empty body", () => {
    acceptTransferOwnership("org-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/transfer/accept",
      { method: "POST", body: {} },
    )
  })

  it("declineTransferOwnership POSTs /transfer/decline with empty body", () => {
    declineTransferOwnership("org-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organizations/org-1/transfer/decline",
      { method: "POST", body: {} },
    )
  })
})

describe("team-api / public invitation landing", () => {
  it("validateInvitation URL-encodes the token", () => {
    validateInvitation("a/b+c=")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/invitations/validate?token=a%2Fb%2Bc%3D",
    )
  })

  it("acceptInvitation POSTs with explicit cookie auth header", () => {
    acceptInvitation({ token: "tok", password: "pwd" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/invitations/accept",
      {
        method: "POST",
        body: { token: "tok", password: "pwd" },
        headers: { "X-Auth-Mode": "cookie" },
      },
    )
  })
})
