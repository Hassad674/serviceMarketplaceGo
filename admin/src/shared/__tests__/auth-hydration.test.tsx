import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { render, screen, act } from "@testing-library/react"
import { MemoryRouter, Routes, Route, Outlet } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import { AuthProvider, useAuth } from "@/shared/hooks/use-auth"
import { useAuthStore } from "@/shared/stores/auth-store"

// Regression contract for the recurring "admin logout on hard refresh" bug.
//
// Background. The admin SPA stores its bearer token in memory only
// (SEC-FINAL-07). On a hard refresh the bearer is intentionally
// dropped — recovery relies on the httpOnly session_id cookie that
// the backend sets at login. AuthProvider probes /auth/me on mount;
// if the cookie is valid, the user stays authenticated.
//
// The bug had two failure modes:
//   1. Backend: token-mode login returned the bearer but skipped the
//      Set-Cookie, so reload had no cookie to use. Fixed in
//      auth_handler_more.go (see backend test
//      TestAuthHandler_Login_TokenMode_SetsSessionCookie).
//   2. Frontend: AuthProvider could let AdminLayout redirect to
//      /login before the boot probe completed, even though the
//      cookie session was valid. The contract below pins the gate.
//
// Together these two contracts make the admin SPA reload-recovery
// path bullet-proof. If either drifts, this suite fails loud.

// Minimal layout fixture mirroring AdminLayout's auth gate.
function GatedLayout() {
  const { isAuthenticated, isHydrating } = useAuth()
  if (isHydrating) return <div data-testid="hydrating">hydrating</div>
  if (!isAuthenticated) return <div data-testid="redirected">redirect-to-login</div>
  return <Outlet />
}

function ProtectedPage() {
  return <div data-testid="protected">protected</div>
}

function renderWithRouter(): { unmount: () => void; queryClient: QueryClient } {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <MemoryRouter initialEntries={["/"]}>
            <Routes>
              <Route element={children}>
                <Route path="/" element={<ProtectedPage />} />
              </Route>
            </Routes>
          </MemoryRouter>
        </AuthProvider>
      </QueryClientProvider>
    )
  }
  const view = render(<GatedLayout />, { wrapper: Wrapper })
  return { unmount: view.unmount, queryClient }
}

describe("AuthProvider boot probe (hard reload recovery)", () => {
  let originalFetch: typeof fetch | undefined
  beforeEach(() => {
    originalFetch = globalThis.fetch
    useAuthStore.getState().clear()
  })
  afterEach(() => {
    if (originalFetch) globalThis.fetch = originalFetch
    useAuthStore.getState().clear()
    vi.restoreAllMocks()
  })

  it("blocks the redirect to /login while the boot probe is in flight", async () => {
    // Hold the /auth/me probe forever — we must observe the hydrating
    // state, which is the contract that prevents the false redirect.
    let resolveProbe!: () => void
    const pending = new Promise<void>((resolve) => {
      resolveProbe = resolve
    })
    globalThis.fetch = vi.fn(async () => {
      await pending
      return new Response(JSON.stringify({ user: null }), { status: 401 })
    }) as unknown as typeof fetch

    renderWithRouter()

    // First paint MUST be the hydrating state, NOT the redirect — the
    // cookie probe has not had a chance to resolve yet.
    expect(screen.queryByTestId("hydrating")).toBeInTheDocument()
    expect(screen.queryByTestId("redirected")).not.toBeInTheDocument()
    expect(screen.queryByTestId("protected")).not.toBeInTheDocument()

    // Resolve the probe with 401 → user really is logged out → NOW we
    // can redirect.
    await act(async () => {
      resolveProbe()
      await pending.catch(() => {})
    })

    expect(screen.queryByTestId("hydrating")).not.toBeInTheDocument()
    expect(screen.queryByTestId("redirected")).toBeInTheDocument()
  })

  it("authenticates the user via cookie session when /auth/me succeeds with is_admin", async () => {
    // Simulate a valid httpOnly cookie session: bearer is null
    // (memory-only store empty after reload), but /auth/me echoes the
    // user with is_admin: true. The provider must mark the user
    // authenticated WITHOUT a bearer.
    globalThis.fetch = vi.fn(
      async () =>
        new Response(
          JSON.stringify({ user: { id: "u1", email: "admin@x", is_admin: true } }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
    ) as unknown as typeof fetch

    await act(async () => {
      renderWithRouter()
    })

    // After the probe resolves, the protected route renders — the
    // SPA stays logged in across the hard reload boundary even
    // though the bearer is null.
    expect(screen.queryByTestId("protected")).toBeInTheDocument()
    expect(screen.queryByTestId("hydrating")).not.toBeInTheDocument()
    expect(screen.queryByTestId("redirected")).not.toBeInTheDocument()
    expect(useAuthStore.getState().token).toBeNull()
  })

  it("redirects to /login when /auth/me probe returns 401 and store is empty", async () => {
    globalThis.fetch = vi.fn(
      async () =>
        new Response(JSON.stringify({ error: { code: "unauthorized" } }), {
          status: 401,
          headers: { "Content-Type": "application/json" },
        }),
    ) as unknown as typeof fetch

    await act(async () => {
      renderWithRouter()
    })

    expect(screen.queryByTestId("redirected")).toBeInTheDocument()
    expect(screen.queryByTestId("protected")).not.toBeInTheDocument()
  })

  it("treats a /auth/me success without is_admin as not authenticated", async () => {
    // A demoted user (is_admin: false) must be sent to /login —
    // the cookie session is technically valid but the SPA refuses
    // anyone who is not an admin.
    globalThis.fetch = vi.fn(
      async () =>
        new Response(
          JSON.stringify({ user: { id: "u1", email: "norm@x", is_admin: false } }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
    ) as unknown as typeof fetch

    await act(async () => {
      renderWithRouter()
    })

    expect(screen.queryByTestId("redirected")).toBeInTheDocument()
    expect(screen.queryByTestId("protected")).not.toBeInTheDocument()
  })

  it("calls /auth/me with credentials: 'include' so the httpOnly cookie is sent", async () => {
    const fetchSpy = vi.fn(
      async () =>
        new Response(JSON.stringify({ user: null }), {
          status: 401,
          headers: { "Content-Type": "application/json" },
        }),
    )
    globalThis.fetch = fetchSpy as unknown as typeof fetch

    await act(async () => {
      renderWithRouter()
    })

    expect(fetchSpy).toHaveBeenCalled()
    const init = fetchSpy.mock.calls[0]?.[1] as RequestInit | undefined
    expect(init?.credentials).toBe("include")
  })
})
