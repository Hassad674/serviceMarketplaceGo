import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useUser, useOrganization, useSession, useLogout } from "../use-user"

const mockFetch = vi.fn()
vi.stubGlobal("fetch", mockFetch)

// Mock window.location. We need `pathname` explicitly because
// fetchSession inspects it to decide whether a 401 should trigger a
// hard redirect to /login (skipped on the /login and /register pages
// themselves, where an unauthenticated 401 is expected).
const originalLocation = window.location
beforeEach(() => {
  Object.defineProperty(window, "location", {
    writable: true,
    value: { ...originalLocation, href: "", pathname: "/dashboard" },
  })
})

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
}

const agencyUser = {
  id: "user-1",
  email: "test@example.com",
  first_name: "Test",
  last_name: "User",
  display_name: "Test User",
  role: "agency",
  referrer_enabled: false,
  email_verified: true,
  created_at: "2026-03-20T10:00:00Z",
}

const agencyOrg = {
  id: "org-1",
  type: "agency",
  owner_user_id: "user-1",
  member_role: "owner",
  member_title: "",
  permissions: ["jobs.create", "jobs.view", "team.invite"],
}

const providerUser = {
  ...agencyUser,
  id: "user-2",
  role: "provider",
}

function mockMe(body: unknown) {
  mockFetch.mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(body),
  })
}

describe("useUser", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("returns just the user slice from /api/v1/auth/me", async () => {
    mockMe({ user: agencyUser, organization: agencyOrg })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(agencyUser)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/auth/me"),
      expect.objectContaining({ credentials: "include" }),
    )
  })

  it("handles not authenticated error", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })

  // R16 zombie-session fix: when /auth/me returns 401 from a protected
  // page, the hook must hard-redirect to /login so the stuck "logged-in
  // but deleted" state is cleared. This is the web side of the backend
  // fix that maps ErrUserNotFound to 401 session_invalid.
  it("redirects to /login on 401 from a protected page", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(window.location.href).toBe("/login")
  })

  // Belt and braces: older backends might still return 404 for a
  // deleted-user /auth/me call. Treat 404 identically to 401 so the
  // fix is forward- and backward-compatible.
  it("redirects to /login on 404 from a protected page", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 404 })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(window.location.href).toBe("/login")
  })

  // On /login and /register, a 401 from /auth/me is the expected
  // "you're not logged in" state — triggering a redirect here would
  // produce an infinite loop or block the login form from rendering.
  it("does NOT redirect on 401 from /login", async () => {
    Object.defineProperty(window, "location", {
      writable: true,
      value: { ...originalLocation, href: "", pathname: "/login" },
    })
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(window.location.href).toBe("")
  })
})

describe("useOrganization", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("returns the organization slice for an agency user", async () => {
    mockMe({ user: agencyUser, organization: agencyOrg })

    const { result } = renderHook(() => useOrganization(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(agencyOrg)
  })

  it("returns null for a solo provider", async () => {
    mockMe({ user: providerUser, organization: null })

    const { result } = renderHook(() => useOrganization(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toBeNull()
  })
})

describe("useSession", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("returns the full { user, organization } payload", async () => {
    mockMe({ user: agencyUser, organization: agencyOrg })

    const { result } = renderHook(() => useSession(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual({
      user: agencyUser,
      organization: agencyOrg,
    })
  })

  it("issues a single fetch even when useUser + useOrganization + useSession are mounted together", async () => {
    mockMe({ user: agencyUser, organization: agencyOrg })

    const wrapper = createWrapper()
    renderHook(
      () => {
        useUser()
        useOrganization()
        useSession()
      },
      { wrapper },
    )

    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1))
  })
})

describe("useLogout", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls logout endpoint and redirects to /login", async () => {
    mockFetch.mockResolvedValue({ ok: true })

    const { result } = renderHook(() => useLogout(), {
      wrapper: createWrapper(),
    })

    await result.current()

    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/auth/logout"),
      expect.objectContaining({ method: "POST", credentials: "include" }),
    )
    expect(window.location.href).toBe("/login")
  })
})
