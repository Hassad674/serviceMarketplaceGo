import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"

export default function PublicLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const t = useTranslations("landing")
  const tCommon = useTranslations("common")

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b border-gray-200 bg-white">
        <nav className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
          <Link
            href="/"
            className="text-lg font-bold tracking-tight text-gray-900"
          >
            Marketplace Service
          </Link>
          <div className="flex items-center gap-6">
            <Link
              href="/agencies"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {t("agenciesTitle")}
            </Link>
            <Link
              href="/freelances"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {t("freelancesTitle")}
            </Link>
            <Link
              href="/projects"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {t("browseProjects")}
            </Link>
            <Link
              href="/login"
              className="text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {tCommon("signIn")}
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800"
            >
              {tCommon("createAccount")}
            </Link>
          </div>
        </nav>
      </header>
      <main className="mx-auto max-w-7xl px-6 py-10">{children}</main>
    </div>
  )
}
