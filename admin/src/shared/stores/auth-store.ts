// In-memory authentication store for the admin SPA.
//
// SECURITY (SEC-FINAL-07): the bearer token MUST NOT be persisted in
// localStorage / sessionStorage / IndexedDB. Any cross-site script
// injection (a transitive dependency compromise, a stale cached
// dependency, even a vulnerable browser extension) can read those
// surfaces and exfiltrate the token. The admin token grants full
// platform-wide write access, so the blast radius of a leak is
// catastrophic — the cost is a re-login on page reload, which is
// acceptable on an admin-only surface that is not used hourly.
//
// On boot, the app calls `restoreSession()` which hits `/auth/me` with
// the existing httpOnly session cookie. If the backend confirms the
// session, the user is restored; otherwise they are sent to /login.
// The Bearer token returned by /auth/login lives ONLY here, in this
// Zustand store, in JavaScript memory, with no persist middleware.
//
// Why no `persist` middleware:
//   - localStorage is XSS-readable.
//   - sessionStorage is XSS-readable AND cleared on tab close anyway.
//   - cookies cannot be set as httpOnly from JavaScript.
//   - IndexedDB has the same XSS exposure as localStorage.
// The only secure place is JS memory — which dies on reload, which is
// the intended behaviour.

import { create } from "zustand"

type AuthState = {
  token: string | null
  isHydrated: boolean
  setToken: (token: string | null) => void
  clear: () => void
  markHydrated: () => void
}

// useAuthStore is the single source of truth for the admin's bearer
// token. The `persist` middleware is intentionally absent — see the
// security note at the top of the file. Any future contributor
// tempted to "fix" the page-reload flow by adding persistence MUST
// instead extend the cookie-based session restore path in
// `restoreSession()`.
export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  isHydrated: false,
  setToken: (token) => set({ token }),
  clear: () => set({ token: null }),
  markHydrated: () => set({ isHydrated: true }),
}))

// getAuthToken is the synchronous accessor for non-React code paths
// (api-client fetch wrappers). Reading directly from the store avoids
// passing the token through every helper signature while keeping it
// out of any persistent storage. Returns null when the user is not
// signed in — callers must handle that case (typically by omitting
// the Authorization header).
export function getAuthToken(): string | null {
  return useAuthStore.getState().token
}

// clearAuthToken is the imperative drop used by the 401 interceptor
// and the logout action. Kept as a free function so non-React code
// (api-client) can call it without subscribing to the store.
export function clearAuthToken(): void {
  useAuthStore.getState().clear()
}
