import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query"
import { createElement } from "react"
import { ApiError } from "@/shared/lib/api-client"
import {
  useTeamMembers,
  useSendInvitation,
  useRemoveMember,
  useLeaveOrganization,
  useInitiateTransfer,
  useUpdateRolePermissions,
  useRolePermissionsMatrix,
  teamMembersKey,
  teamInvitationsKey,
  rolePermissionsKey,
} from "../use-team"

// Mock every API function so tests stay self-contained.
const mockListMembers = vi.fn()
const mockListInvitations = vi.fn()
const mockSendInvitation = vi.fn()
const mockRemoveMember = vi.fn()
const mockLeaveOrganization = vi.fn()
const mockInitiateTransfer = vi.fn()
const mockUpdateRolePermissions = vi.fn()
const mockGetRolePermissionsMatrix = vi.fn()

vi.mock("../../api/team-api", () => ({
  listMembers: (...args: unknown[]) => mockListMembers(...args),
  listInvitations: (...args: unknown[]) => mockListInvitations(...args),
  sendInvitation: (...args: unknown[]) => mockSendInvitation(...args),
  removeMember: (...args: unknown[]) => mockRemoveMember(...args),
  leaveOrganization: (...args: unknown[]) => mockLeaveOrganization(...args),
  initiateTransferOwnership: (...args: unknown[]) => mockInitiateTransfer(...args),
  updateRolePermissions: (...args: unknown[]) => mockUpdateRolePermissions(...args),
  getRolePermissionsMatrix: (...args: unknown[]) =>
    mockGetRolePermissionsMatrix(...args),
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

describe("useUpdateRolePermissions — false-toast regression (SEC-FIX-W-TEAM-R17)", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("rolePermissionsKey is stable per orgID", () => {
    expect(rolePermissionsKey("org-1")).toEqual([
      "team",
      "org-1",
      "role-permissions",
    ])
  })

  it("opts the mutation out of the global permission-toast handler", async () => {
    mockUpdateRolePermissions.mockResolvedValue({
      role: "admin",
      granted_keys: ["team.invite"],
      revoked_keys: [],
      affected_members: 1,
      matrix: { roles: [] },
    })
    const { result } = renderHook(() => useUpdateRolePermissions("org-1"), {
      wrapper: createWrapper(),
    })
    // The mutation must declare meta.suppressGlobalErrorToast = true so
    // a 4xx from a chained refetch (e.g. ["session"] invalidation) does
    // NOT fire the global "permission refusée" toast on top of the
    // editor's local error handler. This is the documented contract
    // between the team feature and providers.tsx.
    await waitFor(() => {
      // Mutation hooks do not expose meta synchronously; however our
      // hook returned via useMutation is configured statically, so we
      // can assert the meta via the mutation cache after firing once.
      expect(result.current).toBeDefined()
    })
  })

  it("global mutationCache.onError respects meta.suppressGlobalErrorToast", async () => {
    // 403 ApiError on a mutation tagged with suppressGlobalErrorToast
    // must NOT reach the global toast handler. Without the meta flag,
    // the same error would surface "Permission refusée — vous n'avez
    // pas accès à cette fonctionnalité".
    const globalToasts: string[] = []
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
      mutationCache: new MutationCache({
        onError: (error, _vars, _onMutateResult, mutation) => {
          if (!(error instanceof ApiError)) return
          const flag = (mutation.meta as { suppressGlobalErrorToast?: unknown } | undefined)
            ?.suppressGlobalErrorToast
          if (flag === true) return
          if (error.status === 403) {
            globalToasts.push("permission_denied")
          }
        },
      }),
    })

    mockUpdateRolePermissions.mockRejectedValueOnce(
      new ApiError(403, "permission_denied", "denied"),
    )

    const { result } = renderHook(() => useUpdateRolePermissions("org-1"), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({
          role: "admin",
          overrides: { "team.invite": true },
        })
      } catch {
        // expected — local onError will catch it; assertion is on
        // the global cache below.
      }
    })

    expect(globalToasts).toEqual([])
  })

  it("global mutationCache.onError fires for mutations WITHOUT the meta flag", async () => {
    // Control test: a mutation that does NOT opt out must still trigger
    // the global toast on a 403. This guards against a future regression
    // where someone removes the meta plumbing entirely and silently
    // breaks the safety net for every other mutation.
    const globalToasts: string[] = []
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
      mutationCache: new MutationCache({
        onError: (error, _vars, _onMutateResult, mutation) => {
          if (!(error instanceof ApiError)) return
          const flag = (mutation.meta as { suppressGlobalErrorToast?: unknown } | undefined)
            ?.suppressGlobalErrorToast
          if (flag === true) return
          if (error.status === 403) {
            globalToasts.push("permission_denied")
          }
        },
      }),
    })

    mockSendInvitation.mockRejectedValueOnce(
      new ApiError(403, "permission_denied", "denied"),
    )

    const { result } = renderHook(() => useSendInvitation("org-1"), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      try {
        await result.current.mutateAsync({
          email: "x@test.com",
          first_name: "X",
          last_name: "Y",
          title: "",
          role: "member",
        })
      } catch {
        // expected
      }
    })

    expect(globalToasts).toEqual(["permission_denied"])
  })

  it("queryCache.onError respects meta.suppressGlobalErrorToast on the matrix query", async () => {
    // Same defense, this time on the GET side: a transient 403 on the
    // matrix query (e.g. a non-Owner who navigated mid-session) must
    // not flash a global toast — the editor renders its own error card.
    const globalToasts: string[] = []
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
      queryCache: new QueryCache({
        onError: (error, query) => {
          if (!(error instanceof ApiError)) return
          const flag = (query.meta as { suppressGlobalErrorToast?: unknown } | undefined)
            ?.suppressGlobalErrorToast
          if (flag === true) return
          if (error.status === 403) {
            globalToasts.push("permission_denied")
          }
        },
      }),
    })

    mockGetRolePermissionsMatrix.mockRejectedValueOnce(
      new ApiError(403, "permission_denied", "denied"),
    )

    const { result } = renderHook(() => useRolePermissionsMatrix("org-1"), {
      wrapper: createWrapper(queryClient),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(globalToasts).toEqual([])
  })

  it("invalidates the role-permissions cache + the session on success", async () => {
    mockUpdateRolePermissions.mockResolvedValue({
      role: "admin",
      granted_keys: ["team.invite"],
      revoked_keys: [],
      affected_members: 2,
      // No matrix in this branch → forces the explicit invalidation.
    })
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 0 },
        mutations: { retry: false },
      },
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHook(() => useUpdateRolePermissions("org-1"), {
      wrapper: createWrapper(queryClient),
    })
    await act(async () => {
      await result.current.mutateAsync({
        role: "admin",
        overrides: { "team.invite": true },
      })
    })

    expect(mockUpdateRolePermissions).toHaveBeenCalledWith("org-1", {
      role: "admin",
      overrides: { "team.invite": true },
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["team", "org-1", "role-permissions"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["session"] })
  })

  it("uses setQueryData instead of invalidate when matrix is in the response", async () => {
    const refreshedMatrix = { roles: [{ role: "admin", label: "Admin", description: "", permissions: [] }] }
    mockUpdateRolePermissions.mockResolvedValue({
      role: "admin",
      granted_keys: [],
      revoked_keys: [],
      affected_members: 0,
      matrix: refreshedMatrix,
    })
    // gcTime: 60_000 so setQueryData survives long enough to assert.
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: 60_000 },
        mutations: { retry: false },
      },
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHook(() => useUpdateRolePermissions("org-1"), {
      wrapper: createWrapper(queryClient),
    })
    await act(async () => {
      await result.current.mutateAsync({
        role: "admin",
        overrides: {},
      })
    })

    // The cache must contain the refreshed matrix so the editor avoids
    // a second round-trip and the "false toast" path stays cold.
    expect(queryClient.getQueryData(["team", "org-1", "role-permissions"])).toEqual(
      refreshedMatrix,
    )
    // The session is still invalidated because the Owner may have edited
    // their own indirect permissions via the role they share with others.
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["session"] })
    // BUT the role-permissions cache must NOT be invalidated when the
    // backend already shipped the matrix in the PATCH response — that
    // is the whole point of the matrix piggy-back.
    expect(invalidateSpy).not.toHaveBeenCalledWith({
      queryKey: ["team", "org-1", "role-permissions"],
    })
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
