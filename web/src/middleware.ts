import createMiddleware from "next-intl/middleware"
import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"
import { routing } from "@i18n/routing"

const LOCALES = routing.locales as readonly string[]
const DEFAULT_LOCALE = routing.defaultLocale as string

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
]

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

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Skip API routes entirely
  if (pathname.startsWith("/api")) {
    return NextResponse.next()
  }

  // Run next-intl middleware first to handle locale detection and rewriting
  const response = intlMiddleware(request)

  // Check protected routes for auth cookie
  const strippedPathname = stripLocalePrefix(pathname)
  const isProtected = PROTECTED_PATHS.some(
    (path) => strippedPathname === path || strippedPathname.startsWith(path + "/"),
  )

  if (!isProtected) {
    return response
  }

  // Auth check: cookie must exist
  const token = request.cookies.get("access_token")?.value
  if (!token) {
    const loginUrl = new URL("/login", request.url)
    return NextResponse.redirect(loginUrl)
  }

  return response
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|public).*)"],
}
