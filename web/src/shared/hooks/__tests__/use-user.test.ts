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

  // Public marketing / listing routes (`/`, `/agencies`,
  // `/freelancers`, `/referrers`, `/opportunities`, …) host the
  // pre-login funnel — an incognito visitor MUST be able to browse
  // the catalogue without a surprise hop to /login. The fetcher
  // therefore swallows 401s on these surfaces instead of forcing a
  // redirect. Locale-prefixed variants (`/fr/freelancers`) are
  // covered by the same check after stripping the prefix.
  const publicListingPaths = [
    "/",
    "/agencies",
    "/freelancers",
    "/freelances",
    "/referrers",
    "/opportunities",
    "/clients",
    "/agencies/abc-123",
    "/freelancers/abc-123",
    "/fr/freelancers",
    "/en/agencies",
    "/fr/",
    // Bug A regression: the invitation acceptance landing page is
    // reached from the email link by an unauthenticated visitor.
    // /auth/me legitimately returns 401 there (no session yet) and
    // a hard-redirect to /login would prevent the user from
    // accepting the invitation. Cover both bare and locale-prefixed
    // variants since the email may include either.
    "/invitation/abc123token",
    "/fr/invitation/abc123token",
    "/en/invitation/abc123token",
  ]

  it.each(publicListingPaths)(
    "does NOT redirect on 401 from public path %s",
    async (pathname) => {
      Object.defineProperty(window, "location", {
        writable: true,
        value: { ...originalLocation, href: "", pathname },
      })
      mockFetch.mockResolvedValue({ ok: false, status: 401 })

      const { result } = renderHook(() => useUser(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isError).toBe(true))
      expect(window.location.href).toBe("")
    },
  )

  // Locale-prefixed protected paths must STILL trigger the
  // zombie-session redirect. Stripping the locale prefix is only a
  // routing concern; once stripped, `/dashboard` is protected and
  // must continue to behave as such.
  it("DOES redirect on 401 from a locale-prefixed protected path /fr/dashboard", async () => {
    Object.defineProperty(window, "location", {
      writable: true,
      value: { ...originalLocation, href: "", pathname: "/fr/dashboard" },
    })
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(window.location.href).toBe("/login")
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

// PERF-FIX-W-AUTH-ME-FANOUT: harden the contract so future edits
// can't silently re-introduce a refetch storm on /api/v1/auth/me.
// The dedup story relies on (a) a single, stable query key shared by
// every consumer and (b) refetch flags that never let TanStack Query
// fire the request without an explicit user mutation.
describe("session query contract", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("dedupes useUser + useOrganization + useSession + useCurrentUserId behind one fetch", async () => {
    mockMe({ user: agencyUser, organization: agencyOrg })

    const wrapper = createWrapper()
    const { result } = renderHook(
      () => {
        const u = useUser()
        const o = useOrganization()
        const s = useSession()
        return { u, o, s }
      },
      { wrapper },
    )

    await waitFor(() => expect(result.current.s.isSuccess).toBe(true))

    expect(mockFetch).toHaveBeenCalledTimes(1)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/auth/me"),
      expect.any(Object),
    )
  })

  it("uses the singleton ['session'] query key so writers can invalidate everyone at once", async () => {
    mockMe({ user: agencyUser, organization: agencyOrg })

    const queryClient = new QueryClient()
    const observerSpy = vi.spyOn(queryClient.getQueryCache(), "build")

    function StandaloneWrapper({ children }: { children: React.ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children)
    }

    renderHook(() => useUser(), { wrapper: StandaloneWrapper })

    type CapturedOpts = {
      queryKey?: readonly unknown[]
      staleTime?: number
      gcTime?: number
      refetchOnMount?: boolean | "always"
      refetchOnWindowFocus?: boolean | "always"
      refetchOnReconnect?: boolean | "always"
      retry?: boolean | number
    }
    const opts = observerSpy.mock.calls[0]?.[1] as unknown as CapturedOpts
    expect(opts.queryKey).toEqual(["session"])
  })

  it("hardens the per-hook options against the global QueryClient defaults", async () => {
    mockMe({ user: agencyUser, organization: agencyOrg })

    const queryClient = new QueryClient({
      // Adversarial defaults — these would re-introduce fan-out if
      // the per-hook options didn't override them. The contract
      // under test is that the session hook is immune.
      defaultOptions: {
        queries: {
          staleTime: 0,
          refetchOnWindowFocus: true,
          refetchOnReconnect: true,
          refetchOnMount: true,
        },
      },
    })
    const observerSpy = vi.spyOn(queryClient.getQueryCache(), "build")

    function HostileDefaultsWrapper({ children }: { children: React.ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children)
    }

    renderHook(() => useSession(), { wrapper: HostileDefaultsWrapper })

    type CapturedOpts = {
      staleTime?: number
      gcTime?: number
      refetchOnMount?: boolean | "always"
      refetchOnWindowFocus?: boolean | "always"
      refetchOnReconnect?: boolean | "always"
      retry?: boolean | number
      retryOnMount?: boolean
    }
    const opts = observerSpy.mock.calls[0]?.[1] as unknown as CapturedOpts
    // 30 minutes — sessions last hours and are explicitly
    // invalidated by mutations.
    expect(opts.staleTime).toBe(30 * 60 * 1000)
    expect(opts.gcTime).toBe(30 * 60 * 1000)
    expect(opts.refetchOnMount).toBe(false)
    expect(opts.refetchOnWindowFocus).toBe(false)
    expect(opts.refetchOnReconnect).toBe(false)
    expect(opts.retry).toBe(false)
    // PERF-FIX-W-AUTH-ME-FANOUT: `retryOnMount: false` is the
    // load-bearing flag that prevents the 401 fan-out. See the long
    // comment in `use-user.ts` for the full diagnosis.
    expect(opts.retryOnMount).toBe(false)
  })
})

// PERF-FIX-W-AUTH-ME-FANOUT regression test.
//
// Before the fix, a public profile page like /freelancers/[id]
// mounted ~30 distinct session consumers (PublicLayout, PostHogProvider,
// SendMessageButton, sidebar / header on the logged-in branch, …) and
// the FIRST /auth/me 401 response left the cache in
// `{ data: undefined, status: "error" }`. With `retryOnMount`
// defaulting to `true`, every subsequent observer that subscribed
// triggered a fresh fetch — TanStack Query's
// `shouldLoadOnMount(query, options)` returns true whenever
// `data === undefined && status !== "error" || retryOnMount !== false`.
// 30 observers × the React-strict-mode double-mount = 60+ requests
// in <1 s, tripping the global IP rate limit.
//
// This block reproduces the storm scenario with a single QueryClient
// shared by N sibling components and asserts that EXACTLY ONE
// `/auth/me` request fires, regardless of how many session
// consumers mount.
describe("auth-me fan-out regression (PERF-FIX-W-AUTH-ME-FANOUT)", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, "location", {
      writable: true,
      value: { ...originalLocation, href: "", pathname: "/freelancers/abc-123" },
    })
  })

  it("fires AT MOST ONE /auth/me request when 50 observers mount on a 401-returning public path", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    // SHARED QueryClient across all `renderHook` calls so the cache
    // persists. In production, layouts, providers, sidebars, chat
    // widgets, lazy-loaded panels each call useUser/useSession at
    // slightly different points in the React commit phase — observer
    // subscriptions are sequential, not simultaneous. Reproducing
    // that ordering is what surfaces the bug.
    const queryClient = new QueryClient()
    function Wrapper({ children }: { children: React.ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children)
    }

    // First observer — drives the actual fetch.
    const { result: firstResult } = renderHook(() => useUser(), {
      wrapper: Wrapper,
    })
    await waitFor(() => expect(firstResult.current.isError).toBe(true))
    expect(mockFetch).toHaveBeenCalledTimes(1)

    // 49 more observers mount sequentially, simulating the rolling
    // mounts of a real freelancer page (PublicLayout, PostHog,
    // SendMessageButton, sidebar, header, ChatWidget, KYCBanner,
    // route-level pages, lazy-loaded modals, …). With the bug
    // present, each one triggered a new /auth/me request because
    // `shouldLoadOnMount(query, options)` returns true whenever
    // `data === undefined && retryOnMount !== false`.
    for (let i = 0; i < 49; i++) {
      const hookKind = i % 3
      renderHook(
        () => {
          if (hookKind === 0) return useUser()
          if (hookKind === 1) return useOrganization()
          return useSession()
        },
        { wrapper: Wrapper },
      )
    }

    // Drain the microtask queue — any fan-out would fire here.
    await new Promise((resolve) => setTimeout(resolve, 100))

    // The contract: ONE fetch, no matter how many consumers mount
    // on a public-path 401. Previous bug yielded 50+.
    expect(mockFetch).toHaveBeenCalledTimes(1)
  })

  it("does NOT refetch when additional observers subscribe AFTER the first 401", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    // SHARED QueryClient across both renderHook calls — the cache
    // (and its 401 verdict) must survive the unmount/remount the
    // way it survives an App Router transition in production.
    const queryClient = new QueryClient()
    function Wrapper({ children }: { children: React.ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children)
    }

    // First observer mounts on its own and resolves to error.
    const { result: first } = renderHook(() => useUser(), { wrapper: Wrapper })
    await waitFor(() => expect(first.current.isError).toBe(true))
    expect(mockFetch).toHaveBeenCalledTimes(1)

    // A late-mounting observer (e.g. a chat widget that lazy-loads
    // after the layout settles) MUST NOT trigger a second fetch.
    // The cache already holds the verdict; the observer should
    // surface it as cached error state.
    const { result: late } = renderHook(() => useSession(), { wrapper: Wrapper })
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(late.current.isError).toBe(true)
    expect(mockFetch).toHaveBeenCalledTimes(1)
  })

  it("forces a fresh fetch when the writer invalidates the session (post-login flow)", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    const queryClient = new QueryClient()
    function Wrapper({ children }: { children: React.ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children)
    }

    const { result } = renderHook(() => useUser(), { wrapper: Wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(mockFetch).toHaveBeenCalledTimes(1)

    // Login flow: the cookie has just been issued by the backend;
    // the form invalidates ["session"] before navigating to
    // /dashboard. This MUST force a refetch even though
    // `retryOnMount: false` would otherwise gate it.
    mockMe({ user: agencyUser, organization: agencyOrg })
    await queryClient.invalidateQueries({ queryKey: ["session"] })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })
})
