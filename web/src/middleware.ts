import createMiddleware from "next-intl/middleware"
import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"
import { routing } from "@i18n/routing"

const intlMiddleware = createMiddleware(routing)

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  // Skip API routes entirely
  if (pathname.startsWith("/api")) {
    return NextResponse.next()
  }

  // Run next-intl middleware first to handle locale detection and rewriting
  const response = intlMiddleware(request)

  // Check dashboard routes for auth token
  // With localePrefix: 'as-needed', default locale (en) paths have no prefix,
  // but non-default locale (fr) paths have /fr prefix.
  // The intl middleware rewrites the URL, so we check the original pathname.
  const isDashboard =
    pathname.startsWith("/dashboard") ||
    pathname.startsWith("/fr/dashboard") ||
    pathname.startsWith("/en/dashboard")

  if (isDashboard) {
    const token = request.cookies.get("access_token")?.value
    if (!token) {
      const loginUrl = new URL("/login", request.url)
      return NextResponse.redirect(loginUrl)
    }
  }

  return response
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|public).*)"],
}
