"use client"

import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { ThemeToggle } from "@/shared/components/theme-toggle"

export function PublicNavbar() {
  const t = useTranslations("landing")
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
            {t("agenciesTitle")}
          </Link>
          <Link
            href="/freelances"
            className="text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
          >
            {t("freelancesTitle")}
          </Link>
          <Link
            href="/projects"
            className="text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
          >
            {t("browseProjects")}
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
