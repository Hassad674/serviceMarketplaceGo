import createMiddleware from "next-intl/middleware"
import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"
import { routing } from "@i18n/routing"

const LOCALES = routing.locales as readonly string[]
const DEFAULT_LOCALE = routing.defaultLocale as string

const intlMiddleware = createMiddleware(routing)

/**
 * Decode the JWT payload without signature verification.
 * The backend already verified the token — we only need the role claim
 * for client-side route protection.
 */
function getRoleFromToken(token: string): string | null {
  try {
    const payload = token.split(".")[1]
    if (!payload) return null
    const decoded = JSON.parse(atob(payload))
    return decoded.role || null
  } catch {
    return null
  }
}

/**
 * Strip the locale prefix from a pathname.
 * "/fr/dashboard/agency" → "/dashboard/agency"
 * "/dashboard/agency" → "/dashboard/agency" (default locale, no prefix)
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
 * Extract the locale from a pathname, falling back to the default locale.
 * "/fr/dashboard/agency" → "fr"
 * "/dashboard/agency" → "en" (default)
 */
function getLocaleFromPathname(pathname: string): string {
  for (const locale of LOCALES) {
    if (
      pathname.startsWith(`/${locale}/`) ||
      pathname === `/${locale}`
    ) {
      return locale
    }
  }
  return DEFAULT_LOCALE
}

/** Maps dashboard path prefixes to the roles allowed to access them. */
const ROUTE_ROLE_MAP: Record<string, string[]> = {
  "/dashboard/agency": ["agency"],
  "/dashboard/provider": ["provider"],
  "/dashboard/referrer": ["provider"], // referrer IS a provider with referrer_enabled
  "/dashboard/enterprise": ["enterprise"],
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Skip API routes entirely
  if (pathname.startsWith("/api")) {
    return NextResponse.next()
  }

  // Run next-intl middleware first to handle locale detection and rewriting
  const response = intlMiddleware(request)

  // Check dashboard routes for auth token and role-based access
  // With localePrefix: 'as-needed', default locale (en) paths have no prefix,
  // but non-default locale (fr) paths have /fr prefix.
  const strippedPathname = stripLocalePrefix(pathname)
  const isDashboard = strippedPathname.startsWith("/dashboard")

  if (!isDashboard) {
    return response
  }

  // --- Auth check: cookie must exist ---
  const token = request.cookies.get("access_token")?.value
  if (!token) {
    const loginUrl = new URL("/login", request.url)
    return NextResponse.redirect(loginUrl)
  }

  // --- Role check: decode JWT and verify access ---
  const role = getRoleFromToken(token)
  if (!role) {
    // Token is malformed or missing role claim — send to login
    const loginUrl = new URL("/login", request.url)
    return NextResponse.redirect(loginUrl)
  }

  for (const [prefix, allowedRoles] of Object.entries(ROUTE_ROLE_MAP)) {
    if (
      strippedPathname.startsWith(prefix) &&
      !allowedRoles.includes(role)
    ) {
      // User is accessing a dashboard section they are not allowed to see.
      // Redirect to the access-denied page with context about the mismatch.
      const attemptedRole =
        Object.entries(ROUTE_ROLE_MAP)
          .find(([routePrefix]) =>
            strippedPathname.startsWith(routePrefix),
          )?.[0]
          ?.split("/")
          .pop() || "unknown"
      const locale = getLocaleFromPathname(pathname)
      const localizedPath =
        locale === DEFAULT_LOCALE
          ? `/access-denied?role=${role}&attempted=${attemptedRole}`
          : `/${locale}/access-denied?role=${role}&attempted=${attemptedRole}`

      return NextResponse.redirect(new URL(localizedPath, request.url))
    }
  }

  return response
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|public).*)"],
}
