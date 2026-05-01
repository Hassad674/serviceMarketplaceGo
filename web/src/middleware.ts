import createMiddleware from "next-intl/middleware"
import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"
import { routing } from "@i18n/routing"

const LOCALES = routing.locales as readonly string[]

const intlMiddleware = createMiddleware(routing)

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

/**
 * SEC-14: paths that may legitimately receive a `?token=` query param
 * from the mobile WebView so the in-app browser can bridge a
 * Bearer-authenticated session into the web UI. Outside this list the
 * query string MUST be ignored — silently dropped from the request and
 * never used as a credential. This prevents an attacker from injecting
 * a token via a referer link and having it accepted on `/dashboard?token=…`.
 */
const TOKEN_BRIDGE_PATHS = ["/payment-info", "/subscribe", "/billing/embed"]

function isTokenBridgePath(pathname: string): boolean {
  return TOKEN_BRIDGE_PATHS.some(
    (path) => pathname === path || pathname.startsWith(path + "/"),
  )
}

/**
 * Strip the locale prefix from a pathname.
 * "/fr/dashboard" -> "/dashboard"
 * "/dashboard" -> "/dashboard" (default locale, no prefix)
 */
function stripLocalePrefix(pathname: string): string {
  for (const locale of LOCALES) {
    const prefix = `/${locale}/`
    if (pathname.startsWith(prefix)) {
      return pathname.slice(prefix.length - 1) // keep the leading "/"
    }
    if (pathname === `/${locale}`) {
      return "/"
    }
  }
  return pathname
}

/**
 * SEC-14: builds a redirect that strips the `?token=` query parameter.
 * The session is exchanged for an httpOnly cookie by the page itself
 * (server-side fetch to /api/v1/auth/web-session); the middleware's
 * job is only to keep the JWT out of subsequent request URLs.
 */
function stripTokenAndRedirect(request: NextRequest): NextResponse {
  const url = request.nextUrl.clone()
  url.searchParams.delete("token")
  return NextResponse.redirect(url)
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Skip API routes entirely
  if (pathname.startsWith("/api")) {
    return NextResponse.next()
  }

  // Run next-intl middleware first to handle locale detection and rewriting
  const response = intlMiddleware(request)

  // SEC-14: when a `?token=` arrives on a path that is NOT one of the
  // sanctioned bridge routes, drop it from the URL entirely. We do
  // this BEFORE the protected-route check so the token cannot be
  // used as a poor-man's credential anywhere on the site.
  const strippedPathname = stripLocalePrefix(pathname)
  if (request.nextUrl.searchParams.has("token") && !isTokenBridgePath(strippedPathname)) {
    return stripTokenAndRedirect(request)
  }

  // Check protected routes for auth cookie
  const isProtected = PROTECTED_PATHS.some(
    (path) => strippedPathname === path || strippedPathname.startsWith(path + "/"),
  )

  if (!isProtected) {
    // If authenticated and on landing page, redirect to dashboard
    const token = request.cookies.get("session_id")?.value
    if (token && strippedPathname === "/") {
      const dashboardUrl = new URL("/dashboard", request.url)
      return NextResponse.redirect(dashboardUrl)
    }
    return response
  }

  // SEC-14: the session_id cookie is the ONLY accepted credential on
  // protected routes. The previous fallback to `?token=` is gone —
  // it would let an attacker phish a user with a crafted URL that
  // pre-populated their JWT in the query string.
  const token = request.cookies.get("session_id")?.value
  if (!token) {
    const loginUrl = new URL("/login", request.url)
    return NextResponse.redirect(loginUrl)
  }

  return response
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|public).*)"],
}
