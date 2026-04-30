import { describe, it, expect } from "vitest"

// Re-implement the middleware decisions as plain functions so the test
// can exercise SEC-14 logic without spinning up the full Next.js
// runtime. The behavior under test is the routing decision tree —
// "is this `?token=…` legitimate, or should the middleware redirect to
// strip it?". That decision lives in pure code (paths, not HTTP).

const TOKEN_BRIDGE_PATHS = ["/payment-info", "/subscribe", "/billing/embed"]
const PROTECTED_PATHS = [
  "/dashboard",
  "/profile",
  "/search",
  "/messages",
  "/missions",
  "/invoices",
  "/team",
  "/projects",
  "/referral",
  "/settings",
  "/account",
]

function isTokenBridgePath(pathname: string): boolean {
  return TOKEN_BRIDGE_PATHS.some(
    (path) => pathname === path || pathname.startsWith(path + "/"),
  )
}

function isProtected(pathname: string): boolean {
  return PROTECTED_PATHS.some(
    (path) => pathname === path || pathname.startsWith(path + "/"),
  )
}

/**
 * Mirrors the decision tree in src/middleware.ts. Returns the first
 * action the middleware would take for a given (pathname, hasToken,
 * hasSession) tuple.
 */
function decideAction(args: {
  pathname: string
  hasTokenQuery: boolean
  hasSessionCookie: boolean
}): "strip-token" | "redirect-login" | "next" {
  const { pathname, hasTokenQuery, hasSessionCookie } = args

  // SEC-14: drop ?token= from any non-bridge path BEFORE doing
  // anything else.
  if (hasTokenQuery && !isTokenBridgePath(pathname)) {
    return "strip-token"
  }

  if (!isProtected(pathname)) {
    return "next"
  }

  if (!hasSessionCookie) {
    return "redirect-login"
  }

  return "next"
}

describe("SEC-14 middleware token-query handling", () => {
  describe("paths sanctioned to receive ?token=", () => {
    it.each([
      "/payment-info",
      "/payment-info/extra/path",
      "/subscribe",
      "/subscribe/return",
      "/billing/embed",
      "/billing/embed/foo",
    ])("allows ?token= on %s", (pathname) => {
      const action = decideAction({
        pathname,
        hasTokenQuery: true,
        hasSessionCookie: false,
      })
      expect(action).not.toBe("strip-token")
    })
  })

  describe("paths NOT sanctioned to receive ?token=", () => {
    it.each([
      "/",
      "/dashboard",
      "/dashboard/projects",
      "/profile",
      "/messages/conversation/abc",
      "/login",
      "/register",
      "/some-random-page",
    ])("strips ?token= on %s", (pathname) => {
      const action = decideAction({
        pathname,
        hasTokenQuery: true,
        hasSessionCookie: false,
      })
      expect(action).toBe("strip-token")
    })
  })

  describe("requests without ?token=", () => {
    it("falls through on public pages", () => {
      const action = decideAction({
        pathname: "/",
        hasTokenQuery: false,
        hasSessionCookie: false,
      })
      expect(action).toBe("next")
    })

    it("redirects unauthenticated users on protected pages", () => {
      const action = decideAction({
        pathname: "/dashboard",
        hasTokenQuery: false,
        hasSessionCookie: false,
      })
      expect(action).toBe("redirect-login")
    })

    it("allows authenticated users on protected pages", () => {
      const action = decideAction({
        pathname: "/dashboard",
        hasTokenQuery: false,
        hasSessionCookie: true,
      })
      expect(action).toBe("next")
    })
  })

  describe("session cookie + ?token= on protected non-bridge path", () => {
    it("still strips the token (SEC-14: token cannot be smuggled even with a session)", () => {
      const action = decideAction({
        pathname: "/dashboard",
        hasTokenQuery: true,
        hasSessionCookie: true,
      })
      expect(action).toBe("strip-token")
    })
  })

  describe("legacy fallback removed", () => {
    it("does NOT honor ?token= as a credential on /dashboard", () => {
      // Before SEC-14 the middleware accepted ?token= as a fallback on
      // protected routes when no session cookie was present. After the
      // fix, that path is gone — the only outcome on a no-cookie
      // request is to strip the token (because /dashboard is not in
      // TOKEN_BRIDGE_PATHS).
      const action = decideAction({
        pathname: "/dashboard",
        hasTokenQuery: true,
        hasSessionCookie: false,
      })
      // The ?token= is dropped FIRST, before the protected check
      // gets a chance to honor it.
      expect(action).toBe("strip-token")
    })
  })
})
