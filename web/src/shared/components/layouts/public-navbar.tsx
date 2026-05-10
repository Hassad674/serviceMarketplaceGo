"use client"

import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { ThemeToggle } from "@/shared/components/theme-toggle"

// PublicNavbar is the slim header rendered by the public layout
// (`(public)/layout.tsx`) for visitors who are NOT signed in. It
// links to the public listing routes — `/agencies`, `/freelancers`,
// `/opportunities` — and exposes the auth CTAs.
//
// All labels are read from the `landing.nav.*` namespace so the keys
// stay aligned with the landing-header anchored nav. The previous
// implementation read invented top-level keys (`landing.agenciesTitle`,
// `landing.freelancesTitle`, `landing.browseProjects`) which never
// existed in fr.json / en.json — visitors saw raw key names instead
// of translated copy. The trip-wire test in
// `__tests__/public-navbar.test.tsx` blocks that regression.
export function PublicNavbar() {
  const tNav = useTranslations("landing.nav")
  const tCommon = useTranslations("common")

  return (
    <header className="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
      <nav className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
        <Link
          href="/"
          className="text-lg font-bold tracking-tight text-gray-900 dark:text-white"
        >
          Marketplace Service
        </Link>
        <div className="flex items-center gap-6">
          <Link
            href="/agencies"
            className="text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
          >
            {tNav("agencies")}
          </Link>
          <Link
            href="/freelancers"
            className="text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
          >
            {tNav("freelancers")}
          </Link>
          <Link
            href="/opportunities"
            className="text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
          >
            {tNav("opportunities")}
          </Link>
          <ThemeToggle />
          <Link
            href="/login"
            className="text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
          >
            {tCommon("signIn")}
          </Link>
          <Link
            href="/register"
            className="rounded-lg bg-gray-900 dark:bg-white px-4 py-2 text-sm font-medium text-white dark:text-gray-900 hover:bg-gray-800 dark:hover:bg-gray-100"
          >
            {tCommon("createAccount")}
          </Link>
        </div>
      </nav>
    </header>
  )
}
