import {
  createContext,
  useContext,
  useCallback,
  useEffect,
  useState,
  type ReactNode,
} from "react"
import { adminApi } from "@/shared/lib/api-client"
import { useAuthStore } from "@/shared/stores/auth-store"

type AuthState = {
  token: string | null
  isAuthenticated: boolean
  isHydrating: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthState | null>(null)

type LoginResponse = {
  access_token: string
  user: { id: string; email: string; is_admin?: boolean }
}

type MeResponse = {
  user?: { id: string; email: string; is_admin?: boolean }
}

// AuthProvider is a thin React wrapper around `useAuthStore`. The
// actual token storage lives in the Zustand store — this provider
// exists only to expose login/logout actions and to drive the
// boot-time cookie-session restore (see useEffect below).
//
// Page reload behaviour (SEC-FINAL-07):
//   - Bearer token is in-memory only (never in localStorage), so a
//     hard reload drops it.
//   - On boot the provider hits `/auth/me` with `credentials:include`.
//     If the user has a valid web session cookie, the backend echoes
//     the user payload and we mark them authenticated WITHOUT a
//     bearer token (cookie-only mode is enough for admin browsing).
//   - If the cookie probe returns 401, the store stays empty and the
//     <AdminLayout> redirects to /login.
//
// The boot probe is fire-and-forget — UI shows a hydrating state for
// the duration of the round-trip. We do not block forever: if /auth/me
// fails for a reason other than 401 (network blip, backend down) we
// fall through to the unauthenticated state, same as a brand-new tab.
export function AuthProvider({ children }: { children: ReactNode }) {
  const token = useAuthStore((s) => s.token)
  const setToken = useAuthStore((s) => s.setToken)
  const clear = useAuthStore((s) => s.clear)
  const [hasCookieSession, setHasCookieSession] = useState(false)
  const [isHydrating, setIsHydrating] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function restoreSession() {
      try {
        const me = await adminApi<MeResponse>("/api/v1/auth/me")
        if (!cancelled && me.user?.is_admin) {
          setHasCookieSession(true)
        }
      } catch {
        // 401 / network / 404 — fall through to logged-out state.
        // The api-client already redirects to /login on 401, so we
        // don't have to do it again here.
      } finally {
        if (!cancelled) {
          setIsHydrating(false)
        }
      }
    }

    restoreSession()
    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(
    async (email: string, password: string) => {
      const data = await adminApi<LoginResponse>("/api/v1/auth/login", {
        method: "POST",
        body: { email, password },
        headers: { "X-Auth-Mode": "token" },
      })

      if (!data.user.is_admin) {
        throw new Error("Acces reserve aux administrateurs")
      }

      setToken(data.access_token)
      setHasCookieSession(true)
    },
    [setToken],
  )

  const logout = useCallback(() => {
    clear()
    setHasCookieSession(false)
    if (typeof window !== "undefined") {
      window.location.href = "/login"
    }
  }, [clear])

  return (
    <AuthContext.Provider
      value={{
        token,
        isAuthenticated: !!token || hasCookieSession,
        isHydrating,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within AuthProvider")
  return ctx
}
