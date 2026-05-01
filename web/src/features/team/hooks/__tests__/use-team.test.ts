import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useTeamMembers,
  useTeamInvitations,
  useSendInvitation,
  useRemoveMember,
  useLeaveOrganization,
  useInitiateTransfer,
  teamMembersKey,
  teamInvitationsKey,
} from "../use-team"

// Mock every API function so tests stay self-contained.
const mockListMembers = vi.fn()
const mockListInvitations = vi.fn()
const mockSendInvitation = vi.fn()
const mockRemoveMember = vi.fn()
const mockLeaveOrganization = vi.fn()
const mockInitiateTransfer = vi.fn()

vi.mock("../../api/team-api", () => ({
  listMembers: (...args: unknown[]) => mockListMembers(...args),
  listInvitations: (...args: unknown[]) => mockListInvitations(...args),
  sendInvitation: (...args: unknown[]) => mockSendInvitation(...args),
  removeMember: (...args: unknown[]) => mockRemoveMember(...args),
  leaveOrganization: (...args: unknown[]) => mockLeaveOrganization(...args),
  initiateTransferOwnership: (...args: unknown[]) => mockInitiateTransfer(...args),
  // unused in this test file but needed so the module resolves
  cancelInvitation: vi.fn(),
  resendInvitation: vi.fn(),
  updateMember: vi.fn(),
  cancelTransferOwnership: vi.fn(),
  acceptTransferOwnership: vi.fn(),
  declineTransferOwnership: vi.fn(),
  validateInvitation: vi.fn(),
  acceptInvitation: vi.fn(),
  getRoleDefinitions: vi.fn(),
}))

function createWrapper(client?: QueryClient) {
  const queryClient =
    client ??
    new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
    })
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
}

describe("team query keys", () => {
  it("teamMembersKey is stable per orgID", () => {
    expect(teamMembersKey("org-1")).toEqual(["team", "org-1", "members"])
  })
  it("teamInvitationsKey is stable per orgID", () => {
    expect(teamInvitationsKey("org-1")).toEqual(["team", "org-1", "invitations"])
  })
})

describe("useTeamMembers", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("is disabled when orgID is undefined", async () => {
    const { result } = renderHook(() => useTeamMembers(undefined), {
      wrapper: createWrapper(),
    })
    // Not enabled → query never runs
    expect(result.current.fetchStatus).toBe("idle")
    expect(mockListMembers).not.toHaveBeenCalled()
  })

  it("fetches members once orgID is provided", async () => {
    const members = { data: [{ id: "m1", user_id: "u1", role: "owner" }] }
    mockListMembers.mockResolvedValue(members)

    const { result } = renderHook(() => useTeamMembers("org-1"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(members)
    expect(mockListMembers).toHaveBeenCalledWith("org-1")
  })
})

describe("useSendInvitation", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("invalidates the team queries on success", async () => {
    mockSendInvitation.mockResolvedValue({ id: "inv-1" })
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHook(() => useSendInvitation("org-1"), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({
        email: "x@test.com",
        first_name: "X",
        last_name: "Y",
        title: "",
        role: "member",
      })
    })

    expect(mockSendInvitation).toHaveBeenCalledWith("org-1", expect.objectContaining({
      email: "x@test.com",
    }))
    // Team scope invalidated
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["team", "org-1"] })
  })
})

describe("useRemoveMember", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("invalidates team and session on success", async () => {
    mockRemoveMember.mockResolvedValue(undefined)
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHook(() => useRemoveMember("org-1", "u-target"), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(mockRemoveMember).toHaveBeenCalledWith("org-1", "u-target")
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["team", "org-1"] })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["session"] })
  })
})

describe("useLeaveOrganization + useInitiateTransfer — smoke", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("leave calls the API with the org id", async () => {
    mockLeaveOrganization.mockResolvedValue(undefined)
    const { result } = renderHook(() => useLeaveOrganization("org-1"), {
      wrapper: createWrapper(),
    })
    await act(async () => {
      await result.current.mutateAsync()
    })
    expect(mockLeaveOrganization).toHaveBeenCalledWith("org-1")
  })

  it("initiate transfer passes the target user id", async () => {
    mockInitiateTransfer.mockResolvedValue(undefined)
    const { result } = renderHook(() => useInitiateTransfer("org-1"), {
      wrapper: createWrapper(),
    })
    await act(async () => {
      await result.current.mutateAsync({ target_user_id: "u-new-owner" })
    })
    expect(mockInitiateTransfer).toHaveBeenCalledWith("org-1", {
      target_user_id: "u-new-owner",
    })
  })
})
